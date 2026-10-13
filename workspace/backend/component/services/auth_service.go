package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"main/component/models"
	"main/component/repositories"
	"main/utils"
	"net/mail"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// AuthService signs users in to the admin app with Google OIDC or an internal
// email/password, and manages the server-side sessions behind the cookie.
// Nothing returned to a caller ever contains a password, Google token or the
// session token's hash; the raw session token only leaves via NewSession.Token.
type AuthService interface {
	Providers() models.ProvidersResponse

	BeginGoogleLogin(ctx context.Context, redirectPath string, rememberMe bool) (string, error)
	CompleteGoogleLogin(ctx context.Context, in GoogleCallbackInput) (*models.NewSession, string, error)

	LoginWithPassword(ctx context.Context, email, password string, meta ClientMeta) (*models.NewSession, error)

	Authenticate(ctx context.Context, rawToken string) (*models.SessionUser, error)
	Logout(ctx context.Context, rawToken string) error
	LogoutAll(ctx context.Context, userID int64) (int64, error)

	CreateUser(ctx context.Context, in CreateUserInput) (int64, error)
	CleanupAuthSessions()
}

type GoogleCallbackInput struct {
	Code          string
	State         string
	ProviderError string
	Meta          ClientMeta
}

// ClientMeta is what the request tells us about the client and the login it asked for.
type ClientMeta struct {
	UserAgent  string
	IP         string
	RememberMe bool // long-lived session (auth.remember-me-days) and a persistent cookie
}

type CreateUserInput struct {
	Email       string
	DisplayName string
	Password    *string // nil = Google-only account
}

type userStore interface {
	GetByEmail(ctx context.Context, email string) (*models.AppUser, error)
	GetByGoogleSub(ctx context.Context, sub string) (*models.AppUser, error)
	Create(ctx context.Context, u models.AppUser) (int64, error)
	LinkGoogle(ctx context.Context, userID int64, sub, displayName string, avatarURL *string) (bool, error)
	RecordLoginFailure(ctx context.Context, userID int64, maxFailed int, lockFor time.Duration) (int, *time.Time, error)
	MarkLoginSuccess(ctx context.Context, userID int64) (time.Time, error)
}

type sessionStore interface {
	Create(ctx context.Context, tokenHash string, userID int64, method models.AuthMethod, expiresAt time.Time, userAgent, ip string) error
	GetValid(ctx context.Context, tokenHash string) (*models.SessionUser, error)
	Revoke(ctx context.Context, tokenHash string) error
	RevokeAllForUser(ctx context.Context, userID int64) (int64, error)
	DeleteStale(ctx context.Context, grace time.Duration) (int64, error)
}

type loginAttemptStore interface {
	Create(ctx context.Context, a models.LoginAttempt) error
	Consume(ctx context.Context, state string) (*models.LoginAttempt, error)
	DeleteExpired(ctx context.Context) (int64, error)
}

const (
	defaultRedirectPath = "/profile"
	minPasswordLen      = 10
	maxPasswordLen      = 72 // bcrypt ignores anything past 72 bytes
	sessionCleanupGrace = 24 * time.Hour
)

type AuthServiceImpl struct {
	cfg        utils.AuthConfig
	users      userStore
	sessions   sessionStore
	attempts   loginAttemptStore
	google     OIDCClient // nil when Google login is disabled or discovery failed
	now        func() time.Time
	rand       io.Reader
	bcryptCost int

	dummyOnce sync.Once
	dummyHash []byte
}

// NewAuthService wires the service. Pass a nil google (untyped) to disable
// Google login; password login keeps working.
func NewAuthService(cfg utils.AuthConfig, users userStore, sessions sessionStore, attempts loginAttemptStore, google OIDCClient) AuthService {
	return &AuthServiceImpl{
		cfg:        cfg.WithDefaults(),
		users:      users,
		sessions:   sessions,
		attempts:   attempts,
		google:     google,
		now:        time.Now,
		rand:       rand.Reader,
		bcryptCost: 12,
	}
}

func (s *AuthServiceImpl) Providers() models.ProvidersResponse {
	return models.ProvidersResponse{Google: s.google != nil, Password: true}
}

// ---------- Google ----------

func (s *AuthServiceImpl) BeginGoogleLogin(ctx context.Context, redirectPath string, rememberMe bool) (string, error) {
	if s.google == nil {
		return "", ErrGoogleDisabled
	}
	state, err := s.randomToken()
	if err != nil {
		return "", wrapAuth(ErrInternal, err)
	}
	nonce, err := s.randomToken()
	if err != nil {
		return "", wrapAuth(ErrInternal, err)
	}
	verifier, err := s.randomToken() // 32 bytes -> 43 base64url chars, as RFC 7636 requires
	if err != nil {
		return "", wrapAuth(ErrInternal, err)
	}
	attempt := models.LoginAttempt{
		State:        state,
		Nonce:        nonce,
		CodeVerifier: verifier,
		RedirectPath: SanitizeRedirectPath(redirectPath),
		RememberMe:   rememberMe,
		ExpiresAt:    s.now().Add(time.Duration(s.cfg.LoginAttemptTTLSeconds) * time.Second),
	}
	if err := s.attempts.Create(ctx, attempt); err != nil {
		return "", wrapAuth(ErrInternal, fmt.Errorf("store login attempt: %w", err))
	}
	return s.google.AuthCodeURL(state, nonce, verifier), nil
}

func (s *AuthServiceImpl) CompleteGoogleLogin(ctx context.Context, in GoogleCallbackInput) (*models.NewSession, string, error) {
	if s.google == nil {
		return nil, "", ErrGoogleDisabled
	}
	if in.State == "" {
		return nil, "", ErrInvalidState
	}
	// Consume before looking at the provider error, so a state can never be replayed.
	attempt, err := s.attempts.Consume(ctx, in.State)
	if err != nil {
		return nil, "", wrapAuth(ErrInternal, fmt.Errorf("consume login attempt: %w", err))
	}
	if attempt == nil {
		return nil, "", ErrInvalidState
	}
	if in.ProviderError != "" {
		if in.ProviderError == "access_denied" {
			return nil, "", ErrConsentDenied
		}
		return nil, "", wrapAuth(ErrExchangeFailed, fmt.Errorf("provider error %q", in.ProviderError))
	}
	if in.Code == "" {
		return nil, "", ErrInvalidRequest
	}

	claims, err := s.google.Exchange(ctx, in.Code, attempt.CodeVerifier)
	if err != nil {
		return nil, "", AsAuthError(err)
	}
	if subtle.ConstantTimeCompare([]byte(claims.Nonce), []byte(attempt.Nonce)) != 1 {
		return nil, "", wrapAuth(ErrInvalidIDToken, errors.New("nonce mismatch"))
	}
	if !claims.EmailVerified {
		return nil, "", ErrEmailNotVerified
	}
	email := normalizeEmail(claims.Email)
	if !s.domainAllowed(email, claims.HostedDomain) {
		return nil, "", wrapAuth(ErrDomainNotAllowed, fmt.Errorf("email %s hd %q", email, claims.HostedDomain))
	}

	user, err := s.resolveGoogleUser(ctx, email, claims)
	if err != nil {
		return nil, "", err
	}
	meta := in.Meta
	meta.RememberMe = attempt.RememberMe // chosen on the login page, carried through Google in the attempt
	session, err := s.startSession(ctx, *user, models.AuthMethodGoogle, meta)
	if err != nil {
		return nil, "", err
	}
	utils.ShowInfoLogs("[AuthServiceImpl] [CompleteGoogleLogin] success user_id=%d email=%s", user.ID, user.Email)
	return session, attempt.RedirectPath, nil
}

// resolveGoogleUser finds the user by the pinned Google sub first, then by
// verified email (linking the sub), then auto-provisions if configured.
func (s *AuthServiceImpl) resolveGoogleUser(ctx context.Context, email string, c *GoogleClaims) (*models.AppUser, error) {
	var avatar *string
	if c.Picture != "" {
		avatar = &c.Picture
	}

	user, err := s.users.GetByGoogleSub(ctx, c.Sub)
	if err != nil {
		return nil, wrapAuth(ErrInternal, err)
	}
	if user == nil {
		user, err = s.users.GetByEmail(ctx, email)
		if err != nil {
			return nil, wrapAuth(ErrInternal, err)
		}
		switch {
		case user != nil:
			linked, err := s.users.LinkGoogle(ctx, user.ID, c.Sub, c.Name, avatar)
			if err != nil {
				return nil, wrapAuth(ErrInternal, err)
			}
			if !linked {
				return nil, wrapAuth(ErrUserNotAllowed, fmt.Errorf("user_id=%d already linked to another Google account", user.ID))
			}
		case s.cfg.Google.AutoProvision:
			sub := c.Sub
			id, err := s.users.Create(ctx, models.AppUser{
				Email: email, DisplayName: c.Name, AvatarURL: avatar, GoogleSub: &sub, Status: models.UserStatusActive,
			})
			if err != nil {
				return nil, wrapAuth(ErrInternal, fmt.Errorf("auto-provision %s: %w", email, err))
			}
			utils.ShowInfoLogs("[AuthServiceImpl] [CompleteGoogleLogin] auto-provisioned user_id=%d email=%s", id, email)
			user = &models.AppUser{ID: id, Email: email, DisplayName: c.Name, AvatarURL: avatar, GoogleSub: &sub, Status: models.UserStatusActive}
		default:
			return nil, wrapAuth(ErrUserNotAllowed, fmt.Errorf("no app_user for %s", email))
		}
	}
	if user.Status != models.UserStatusActive {
		return nil, wrapAuth(ErrUserNotAllowed, fmt.Errorf("user_id=%d is %s", user.ID, user.Status))
	}
	if c.Name != "" && user.DisplayName == "" {
		user.DisplayName = c.Name
	}
	user.AvatarURL = avatar
	return user, nil
}

// domainAllowed requires both the hd claim and the email's domain to be in
// AllowedDomains, so a personal Gmail (no hd) never passes a domain gate.
func (s *AuthServiceImpl) domainAllowed(email, hostedDomain string) bool {
	if len(s.cfg.Google.AllowedDomains) == 0 {
		return true
	}
	at := strings.LastIndex(email, "@")
	if at < 0 {
		return false
	}
	emailDomain := email[at+1:]
	hd := strings.ToLower(hostedDomain)
	for _, d := range s.cfg.Google.AllowedDomains {
		d = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(d), "@"))
		if d != "" && d == emailDomain && d == hd {
			return true
		}
	}
	return false
}

// ---------- Password ----------

func (s *AuthServiceImpl) LoginWithPassword(ctx context.Context, email, password string, meta ClientMeta) (*models.NewSession, error) {
	email = normalizeEmail(email)
	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		return nil, wrapAuth(ErrInternal, err)
	}
	if user == nil || user.PasswordHash == nil || user.Status != models.UserStatusActive {
		// Same work and same answer as a wrong password, so emails can't be enumerated.
		_ = bcrypt.CompareHashAndPassword(s.getDummyHash(), []byte(password))
		return nil, ErrInvalidCredentials
	}
	if user.LockedUntil != nil && s.now().Before(*user.LockedUntil) {
		return nil, ErrAccountLocked
	}
	if bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(password)) != nil {
		count, lockedUntil, err := s.users.RecordLoginFailure(ctx, user.ID, s.cfg.MaxFailedLogins, time.Duration(s.cfg.LockoutMinutes)*time.Minute)
		if err != nil {
			return nil, wrapAuth(ErrInternal, err)
		}
		utils.ShowInfoLogs("[AuthServiceImpl] [LoginWithPassword] failed user_id=%d count=%d", user.ID, count)
		if lockedUntil != nil && s.now().Before(*lockedUntil) {
			return nil, ErrAccountLocked
		}
		return nil, ErrInvalidCredentials
	}
	session, err := s.startSession(ctx, *user, models.AuthMethodPassword, meta)
	if err != nil {
		return nil, err
	}
	utils.ShowInfoLogs("[AuthServiceImpl] [LoginWithPassword] success user_id=%d", user.ID)
	return session, nil
}

func (s *AuthServiceImpl) getDummyHash() []byte {
	s.dummyOnce.Do(func() {
		s.dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy-password-for-timing"), s.bcryptCost)
	})
	return s.dummyHash
}

// ---------- Sessions ----------

func (s *AuthServiceImpl) startSession(ctx context.Context, user models.AppUser, method models.AuthMethod, meta ClientMeta) (*models.NewSession, error) {
	loginAt, err := s.users.MarkLoginSuccess(ctx, user.ID)
	if err != nil {
		return nil, wrapAuth(ErrInternal, err)
	}
	user.LastLoginAt = &loginAt
	user.FailedLoginCount = 0
	user.LockedUntil = nil

	token, err := s.randomToken()
	if err != nil {
		return nil, wrapAuth(ErrInternal, err)
	}
	hash := hashToken(token)
	ttl := time.Duration(s.cfg.SessionTTLHours) * time.Hour
	if meta.RememberMe {
		ttl = time.Duration(s.cfg.RememberMeDays) * 24 * time.Hour
	}
	expiresAt := s.now().Add(ttl)
	if err := s.sessions.Create(ctx, hash, user.ID, method, expiresAt, truncate(meta.UserAgent, 512), truncate(meta.IP, 64)); err != nil {
		return nil, wrapAuth(ErrInternal, fmt.Errorf("create session: %w", err))
	}
	return &models.NewSession{
		Token:      token,
		ExpiresAt:  expiresAt,
		Persistent: meta.RememberMe,
		User:       models.SessionUser{AppUser: user, AuthMethod: method, TokenHash: hash},
	}, nil
}

func (s *AuthServiceImpl) Authenticate(ctx context.Context, rawToken string) (*models.SessionUser, error) {
	if rawToken == "" {
		return nil, ErrUnauthenticated
	}
	user, err := s.sessions.GetValid(ctx, hashToken(rawToken))
	if err != nil {
		return nil, wrapAuth(ErrInternal, err)
	}
	if user == nil {
		return nil, ErrUnauthenticated
	}
	return user, nil
}

func (s *AuthServiceImpl) Logout(ctx context.Context, rawToken string) error {
	if rawToken == "" {
		return nil
	}
	if err := s.sessions.Revoke(ctx, hashToken(rawToken)); err != nil {
		return wrapAuth(ErrInternal, err)
	}
	return nil
}

// LogoutAll revokes every open session of the user ("sign out everywhere"),
// including the one making the request. Returns how many were revoked.
func (s *AuthServiceImpl) LogoutAll(ctx context.Context, userID int64) (int64, error) {
	n, err := s.sessions.RevokeAllForUser(ctx, userID)
	if err != nil {
		return 0, wrapAuth(ErrInternal, err)
	}
	utils.ShowInfoLogs("[AuthServiceImpl] [LogoutAll] user_id=%d revoked=%d", userID, n)
	return n, nil
}

// ---------- Ops ----------

func (s *AuthServiceImpl) CreateUser(ctx context.Context, in CreateUserInput) (int64, error) {
	email := normalizeEmail(in.Email)
	if addr, err := mail.ParseAddress(email); err != nil || addr.Address != email {
		return 0, fmt.Errorf("invalid email %q", in.Email)
	}
	u := models.AppUser{Email: email, DisplayName: strings.TrimSpace(in.DisplayName), Status: models.UserStatusActive}
	if in.Password != nil {
		pw := *in.Password
		if len(pw) < minPasswordLen || len(pw) > maxPasswordLen {
			return 0, fmt.Errorf("password must be %d-%d bytes", minPasswordLen, maxPasswordLen)
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(pw), s.bcryptCost)
		if err != nil {
			return 0, err
		}
		h := string(hash)
		u.PasswordHash = &h
	}
	id, err := s.users.Create(ctx, u)
	if errors.Is(err, repositories.ErrDuplicateEmail) {
		return 0, fmt.Errorf("a user with email %s already exists", email)
	}
	return id, err
}

// CleanupAuthSessions is the cleanupAuthSessions worker.
func (s *AuthServiceImpl) CleanupAuthSessions() {
	defer func() {
		if r := recover(); r != nil {
			utils.ShowErrorLogs(fmt.Errorf("[AuthServiceImpl] [CleanupAuthSessions] panic: %v", r))
		}
	}()
	ctx := context.Background()
	sessions, err := s.sessions.DeleteStale(ctx, sessionCleanupGrace)
	if err != nil {
		utils.ShowErrorLogs(fmt.Errorf("[AuthServiceImpl] [CleanupAuthSessions] [DeleteStale] %w", err))
	}
	attempts, err := s.attempts.DeleteExpired(ctx)
	if err != nil {
		utils.ShowErrorLogs(fmt.Errorf("[AuthServiceImpl] [CleanupAuthSessions] [DeleteExpired] %w", err))
	}
	utils.ShowInfoLogs("[AuthServiceImpl] [CleanupAuthSessions] removed sessions=%d login_attempts=%d", sessions, attempts)
}

// ---------- helpers ----------

func (s *AuthServiceImpl) randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := io.ReadFull(s.rand, b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

// SanitizeRedirectPath only lets through same-site absolute paths, so the
// post-login redirect cannot be turned into an open redirect.
func SanitizeRedirectPath(p string) string {
	if p == "" || len(p) > 512 || p[0] != '/' {
		return defaultRedirectPath
	}
	if strings.HasPrefix(p, "//") || strings.HasPrefix(p, "/\\") || strings.ContainsAny(p, "\\\r\n\t") || strings.Contains(p, "://") {
		return defaultRedirectPath
	}
	if strings.HasPrefix(p, "/login") {
		return defaultRedirectPath
	}
	return p
}
