package services

import (
	"context"
	"errors"
	"main/component/models"
	"main/utils"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"golang.org/x/crypto/bcrypt"
)

// ---------- fakes ----------

type fakeUsers struct {
	byID    map[int64]*models.AppUser
	nextID  int64
	linkErr error
}

func newFakeUsers(users ...models.AppUser) *fakeUsers {
	f := &fakeUsers{byID: map[int64]*models.AppUser{}, nextID: 100}
	for i := range users {
		u := users[i]
		f.byID[u.ID] = &u
	}
	return f
}

func (f *fakeUsers) GetByEmail(_ context.Context, email string) (*models.AppUser, error) {
	for _, u := range f.byID {
		if u.Email == email {
			c := *u
			return &c, nil
		}
	}
	return nil, nil
}

func (f *fakeUsers) GetByGoogleSub(_ context.Context, sub string) (*models.AppUser, error) {
	for _, u := range f.byID {
		if u.GoogleSub != nil && *u.GoogleSub == sub {
			c := *u
			return &c, nil
		}
	}
	return nil, nil
}

func (f *fakeUsers) Create(_ context.Context, u models.AppUser) (int64, error) {
	f.nextID++
	u.ID = f.nextID
	f.byID[u.ID] = &u
	return u.ID, nil
}

func (f *fakeUsers) LinkGoogle(_ context.Context, id int64, sub, name string, avatar *string) (bool, error) {
	if f.linkErr != nil {
		return false, f.linkErr
	}
	u := f.byID[id]
	if u.GoogleSub != nil && *u.GoogleSub != sub {
		return false, nil
	}
	u.GoogleSub = &sub
	u.AvatarURL = avatar
	if u.DisplayName == "" {
		u.DisplayName = name
	}
	return true, nil
}

func (f *fakeUsers) RecordLoginFailure(_ context.Context, id int64, max int, lockFor time.Duration) (int, *time.Time, error) {
	u := f.byID[id]
	u.FailedLoginCount++
	if u.FailedLoginCount >= max {
		t := time.Now().Add(lockFor)
		u.LockedUntil = &t
	}
	return u.FailedLoginCount, u.LockedUntil, nil
}

func (f *fakeUsers) MarkLoginSuccess(_ context.Context, id int64) (time.Time, error) {
	u := f.byID[id]
	now := time.Now()
	u.FailedLoginCount, u.LockedUntil, u.LastLoginAt = 0, nil, &now
	return now, nil
}

type fakeSession struct {
	userID  int64
	method  models.AuthMethod
	expires time.Time
	revoked bool
}

type fakeSessions struct {
	users *fakeUsers
	rows  map[string]*fakeSession
}

func (f *fakeSessions) Create(_ context.Context, hash string, userID int64, m models.AuthMethod, exp time.Time, _, _ string) error {
	f.rows[hash] = &fakeSession{userID: userID, method: m, expires: exp}
	return nil
}

func (f *fakeSessions) GetValid(_ context.Context, hash string) (*models.SessionUser, error) {
	s, ok := f.rows[hash]
	if !ok || s.revoked || time.Now().After(s.expires) {
		return nil, nil
	}
	u := f.users.byID[s.userID]
	if u.Status != models.UserStatusActive {
		return nil, nil
	}
	return &models.SessionUser{AppUser: *u, AuthMethod: s.method, TokenHash: hash}, nil
}

func (f *fakeSessions) Revoke(_ context.Context, hash string) error {
	if s, ok := f.rows[hash]; ok {
		s.revoked = true
	}
	return nil
}

func (f *fakeSessions) RevokeAllForUser(_ context.Context, userID int64) (int64, error) {
	var n int64
	for _, s := range f.rows {
		if s.userID == userID && !s.revoked {
			s.revoked = true
			n++
		}
	}
	return n, nil
}

func (f *fakeSessions) DeleteStale(context.Context, time.Duration) (int64, error) { return 0, nil }

type fakeAttempts struct {
	rows map[string]models.LoginAttempt
}

func (f *fakeAttempts) Create(_ context.Context, a models.LoginAttempt) error {
	f.rows[a.State] = a
	return nil
}

func (f *fakeAttempts) Consume(_ context.Context, state string) (*models.LoginAttempt, error) {
	a, ok := f.rows[state]
	if !ok || time.Now().After(a.ExpiresAt) {
		return nil, nil
	}
	delete(f.rows, state)
	return &a, nil
}

func (f *fakeAttempts) DeleteExpired(context.Context) (int64, error) { return 0, nil }

type fakeOIDC struct {
	claims       GoogleClaims // Nonce is filled from the last AuthCodeURL unless overrideNonce
	overrideNone bool
	err          error
	lastNonce    string
	lastVerifier string
	gotVerifier  string
	exchanges    int
}

func (f *fakeOIDC) AuthCodeURL(state, nonce, verifier string) string {
	f.lastNonce, f.lastVerifier = nonce, verifier
	return "https://accounts.example/auth?state=" + state
}

func (f *fakeOIDC) Exchange(_ context.Context, _ string, verifier string) (*GoogleClaims, error) {
	f.exchanges++
	f.gotVerifier = verifier
	if f.err != nil {
		return nil, f.err
	}
	c := f.claims
	if !f.overrideNone {
		c.Nonce = f.lastNonce
	}
	return &c, nil
}

// ---------- harness ----------

type harness struct {
	svc      *AuthServiceImpl
	users    *fakeUsers
	sessions *fakeSessions
	attempts *fakeAttempts
	google   *fakeOIDC
}

func strp(s string) *string { return &s }

func hashPw(t *testing.T, pw string) *string {
	t.Helper()
	h, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	return strp(string(h))
}

func newHarness(t *testing.T, cfg utils.AuthConfig, users ...models.AppUser) *harness {
	t.Helper()
	h := &harness{users: newFakeUsers(users...), attempts: &fakeAttempts{rows: map[string]models.LoginAttempt{}}}
	h.sessions = &fakeSessions{users: h.users, rows: map[string]*fakeSession{}}
	h.google = &fakeOIDC{claims: GoogleClaims{Sub: "g-1", Email: "An.Nguyen@MQ.com", EmailVerified: true, HostedDomain: "mq.com", Name: "An Nguyen", Picture: "https://img/a.png"}}
	h.svc = NewAuthService(cfg, h.users, h.sessions, h.attempts, h.google).(*AuthServiceImpl)
	h.svc.bcryptCost = bcrypt.MinCost
	return h
}

func mqConfig() utils.AuthConfig {
	return utils.AuthConfig{Google: utils.GoogleAuthConfig{Enable: true, AllowedDomains: []string{"mq.com"}}}
}

func activeUser(id int64, email string) models.AppUser {
	return models.AppUser{ID: id, Email: email, Status: models.UserStatusActive}
}

// googleLogin runs Begin + Complete and returns the Complete result.
func (h *harness) googleLogin(t *testing.T) (*models.NewSession, string, error) {
	t.Helper()
	if _, err := h.svc.BeginGoogleLogin(context.Background(), "/profile", false); err != nil {
		t.Fatalf("BeginGoogleLogin: %v", err)
	}
	var state string
	for s := range h.attempts.rows {
		state = s
	}
	return h.svc.CompleteGoogleLogin(context.Background(), GoogleCallbackInput{Code: "auth-code", State: state})
}

// ---------- Google ----------

func TestBeginGoogleLogin_StoresUniqueAttempt(t *testing.T) {
	h := newHarness(t, mqConfig())
	u1, err := h.svc.BeginGoogleLogin(context.Background(), "//evil.example", false)
	if err != nil {
		t.Fatal(err)
	}
	u2, _ := h.svc.BeginGoogleLogin(context.Background(), "/orders?x=1", false)
	if u1 == u2 || len(h.attempts.rows) != 2 {
		t.Fatalf("want 2 distinct attempts, got %d (%q, %q)", len(h.attempts.rows), u1, u2)
	}
	paths := map[string]bool{}
	for _, a := range h.attempts.rows {
		if len(a.State) < 43 || len(a.Nonce) < 43 || len(a.CodeVerifier) != 43 {
			t.Errorf("weak randomness: state=%d nonce=%d verifier=%d chars", len(a.State), len(a.Nonce), len(a.CodeVerifier))
		}
		paths[a.RedirectPath] = true
	}
	if !paths["/profile"] || !paths["/orders?x=1"] {
		t.Errorf("redirect paths = %v, want open redirect replaced by /profile", paths)
	}
}

func TestCompleteGoogleLogin_LinksExistingUser(t *testing.T) {
	h := newHarness(t, mqConfig(), activeUser(1, "an.nguyen@mq.com"))
	sess, path, err := h.googleLogin(t)
	if err != nil {
		t.Fatalf("CompleteGoogleLogin: %v", err)
	}
	if path != "/profile" || sess.Token == "" || sess.User.ID != 1 || sess.User.AuthMethod != models.AuthMethodGoogle {
		t.Fatalf("unexpected session %+v path %q", sess.User, path)
	}
	if h.google.gotVerifier != h.google.lastVerifier {
		t.Error("PKCE verifier from the attempt was not passed to Exchange")
	}
	if u := h.users.byID[1]; u.GoogleSub == nil || *u.GoogleSub != "g-1" || u.DisplayName != "An Nguyen" {
		t.Errorf("user not linked: %+v", u)
	}
	if _, ok := h.sessions.rows[hashToken(sess.Token)]; !ok {
		t.Error("session stored under something other than sha256(token)")
	}
	if _, ok := h.sessions.rows[sess.Token]; ok {
		t.Error("raw token stored in session table")
	}
}

func TestCompleteGoogleLogin_Failures(t *testing.T) {
	cases := []struct {
		name  string
		cfg   func(*utils.AuthConfig)
		setup func(*harness)
		want  *AuthError
	}{
		{"not pre-created", nil, nil, ErrUserNotAllowed},
		{"email not verified", nil, func(h *harness) { h.google.claims.EmailVerified = false }, ErrEmailNotVerified},
		{"personal gmail", nil, func(h *harness) {
			h.google.claims.Email, h.google.claims.HostedDomain = "an@gmail.com", ""
		}, ErrDomainNotAllowed},
		{"hd mismatch", nil, func(h *harness) { h.google.claims.HostedDomain = "other.com" }, ErrDomainNotAllowed},
		{"nonce mismatch", nil, func(h *harness) { h.google.overrideNone, h.google.claims.Nonce = true, "forged" }, ErrInvalidIDToken},
		{"exchange failed", nil, func(h *harness) { h.google.err = wrapAuth(ErrExchangeFailed, errors.New("boom")) }, ErrExchangeFailed},
		{"bad id token", nil, func(h *harness) { h.google.err = wrapAuth(ErrInvalidIDToken, errors.New("bad sig")) }, ErrInvalidIDToken},
		{"disabled user", nil, func(h *harness) {
			u := activeUser(1, "an.nguyen@mq.com")
			u.Status = models.UserStatusDisabled
			h.users.byID[1] = &u
		}, ErrUserNotAllowed},
		{"linked to another google account", nil, func(h *harness) {
			u := activeUser(1, "an.nguyen@mq.com")
			u.GoogleSub = strp("g-other")
			h.users.byID[1] = &u
		}, ErrUserNotAllowed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := mqConfig()
			if tc.cfg != nil {
				tc.cfg(&cfg)
			}
			h := newHarness(t, cfg)
			if tc.setup != nil {
				tc.setup(h)
			}
			sess, _, err := h.googleLogin(t)
			if !errors.Is(err, tc.want) || sess != nil {
				t.Fatalf("err = %v, want %s", err, tc.want.Code)
			}
			if len(h.sessions.rows) != 0 {
				t.Error("session created on failure")
			}
		})
	}
}

func TestCompleteGoogleLogin_AutoProvision(t *testing.T) {
	cfg := mqConfig()
	cfg.Google.AutoProvision = true
	h := newHarness(t, cfg)
	sess, _, err := h.googleLogin(t)
	if err != nil {
		t.Fatal(err)
	}
	if sess.User.Email != "an.nguyen@mq.com" || sess.User.GoogleSub == nil {
		t.Fatalf("auto-provisioned user = %+v", sess.User.AppUser)
	}
	// A second login finds the same user by sub, not a new one.
	if _, _, err := h.googleLogin(t); err != nil || len(h.users.byID) != 1 {
		t.Fatalf("second login: err=%v users=%d", err, len(h.users.byID))
	}
}

func TestCompleteGoogleLogin_AutoProvisionStillChecksDomain(t *testing.T) {
	cfg := mqConfig()
	cfg.Google.AutoProvision = true
	h := newHarness(t, cfg)
	h.google.claims.Email, h.google.claims.HostedDomain = "x@gmail.com", ""
	if _, _, err := h.googleLogin(t); !errors.Is(err, ErrDomainNotAllowed) || len(h.users.byID) != 0 {
		t.Fatalf("err=%v users=%d", err, len(h.users.byID))
	}
}

func TestCompleteGoogleLogin_State(t *testing.T) {
	h := newHarness(t, mqConfig(), activeUser(1, "an.nguyen@mq.com"))
	ctx := context.Background()

	if _, _, err := h.svc.CompleteGoogleLogin(ctx, GoogleCallbackInput{Code: "c"}); !errors.Is(err, ErrInvalidState) {
		t.Errorf("missing state: %v", err)
	}
	if _, _, err := h.svc.CompleteGoogleLogin(ctx, GoogleCallbackInput{Code: "c", State: "unknown"}); !errors.Is(err, ErrInvalidState) {
		t.Errorf("unknown state: %v", err)
	}

	h.attempts.rows["old"] = models.LoginAttempt{State: "old", ExpiresAt: time.Now().Add(-time.Second)}
	if _, _, err := h.svc.CompleteGoogleLogin(ctx, GoogleCallbackInput{Code: "c", State: "old"}); !errors.Is(err, ErrInvalidState) {
		t.Errorf("expired state: %v", err)
	}

	h.svc.BeginGoogleLogin(ctx, "", false)
	var state string
	for s := range h.attempts.rows {
		if s != "old" {
			state = s
		}
	}
	if _, _, err := h.svc.CompleteGoogleLogin(ctx, GoogleCallbackInput{Code: "c", State: state}); err != nil {
		t.Fatalf("first use: %v", err)
	}
	if _, _, err := h.svc.CompleteGoogleLogin(ctx, GoogleCallbackInput{Code: "c", State: state}); !errors.Is(err, ErrInvalidState) {
		t.Errorf("replayed state: %v", err)
	}
	if h.google.exchanges != 1 {
		t.Errorf("token endpoint called %d times, want 1 (only for the valid state)", h.google.exchanges)
	}
}

func TestCompleteGoogleLogin_ConsentDenied(t *testing.T) {
	h := newHarness(t, mqConfig(), activeUser(1, "an.nguyen@mq.com"))
	h.svc.BeginGoogleLogin(context.Background(), "", false)
	var state string
	for s := range h.attempts.rows {
		state = s
	}
	_, _, err := h.svc.CompleteGoogleLogin(context.Background(), GoogleCallbackInput{State: state, ProviderError: "access_denied"})
	if !errors.Is(err, ErrConsentDenied) {
		t.Fatalf("err = %v", err)
	}
	if len(h.attempts.rows) != 0 {
		t.Error("denied attempt was not consumed")
	}
	if h.google.exchanges != 0 {
		t.Error("exchanged a code after consent was denied")
	}
}

func TestGoogleDisabled(t *testing.T) {
	h := newHarness(t, utils.AuthConfig{})
	svc := NewAuthService(utils.AuthConfig{}, h.users, h.sessions, h.attempts, nil)
	if svc.Providers().Google {
		t.Error("Providers().Google = true with nil client")
	}
	if _, err := svc.BeginGoogleLogin(context.Background(), "", false); !errors.Is(err, ErrGoogleDisabled) {
		t.Errorf("err = %v", err)
	}
}

// ---------- Password ----------

func TestLoginWithPassword(t *testing.T) {
	pwUser := activeUser(1, "an.nguyen@mq.com")
	pwUser.PasswordHash = hashPw(t, "correct-horse-1")
	googleOnly := activeUser(2, "google.only@mq.com")
	disabled := activeUser(3, "gone@mq.com")
	disabled.PasswordHash = hashPw(t, "correct-horse-1")
	disabled.Status = models.UserStatusDisabled

	h := newHarness(t, utils.AuthConfig{}, pwUser, googleOnly, disabled)
	ctx := context.Background()

	sess, err := h.svc.LoginWithPassword(ctx, "  AN.Nguyen@mq.com ", "correct-horse-1", ClientMeta{})
	if err != nil || sess.User.ID != 1 || sess.User.AuthMethod != models.AuthMethodPassword {
		t.Fatalf("valid login: %v %+v", err, sess)
	}

	for name, tc := range map[string][2]string{
		"wrong password": {"an.nguyen@mq.com", "nope-nope-nope"},
		"unknown email":  {"nobody@mq.com", "correct-horse-1"},
		"google only":    {"google.only@mq.com", "correct-horse-1"},
		"disabled":       {"gone@mq.com", "correct-horse-1"},
	} {
		if _, err := h.svc.LoginWithPassword(ctx, tc[0], tc[1], ClientMeta{}); !errors.Is(err, ErrInvalidCredentials) {
			t.Errorf("%s: err = %v, want INVALID_CREDENTIALS", name, err)
		}
	}
}

func TestLoginWithPassword_Lockout(t *testing.T) {
	u := activeUser(1, "an.nguyen@mq.com")
	u.PasswordHash = hashPw(t, "correct-horse-1")
	h := newHarness(t, utils.AuthConfig{}, u) // defaults: 5 failures, 15 minutes
	ctx := context.Background()

	for i := 1; i <= 4; i++ {
		if _, err := h.svc.LoginWithPassword(ctx, u.Email, "wrong-wrong-1", ClientMeta{}); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("failure %d: %v", i, err)
		}
	}
	if _, err := h.svc.LoginWithPassword(ctx, u.Email, "wrong-wrong-1", ClientMeta{}); !errors.Is(err, ErrAccountLocked) {
		t.Fatalf("5th failure: %v, want ACCOUNT_LOCKED", err)
	}
	if _, err := h.svc.LoginWithPassword(ctx, u.Email, "correct-horse-1", ClientMeta{}); !errors.Is(err, ErrAccountLocked) {
		t.Fatalf("correct password while locked: %v, want ACCOUNT_LOCKED", err)
	}

	h.svc.now = func() time.Time { return time.Now().Add(16 * time.Minute) }
	if _, err := h.svc.LoginWithPassword(ctx, u.Email, "correct-horse-1", ClientMeta{}); err != nil {
		t.Fatalf("after lockout: %v", err)
	}
	if h.users.byID[1].FailedLoginCount != 0 {
		t.Error("failure counter not reset on success")
	}
}

// ---------- Sessions ----------

func TestAuthenticateAndLogout(t *testing.T) {
	u := activeUser(1, "an.nguyen@mq.com")
	u.PasswordHash = hashPw(t, "correct-horse-1")
	h := newHarness(t, utils.AuthConfig{}, u)
	ctx := context.Background()

	if _, err := h.svc.Authenticate(ctx, ""); !errors.Is(err, ErrUnauthenticated) {
		t.Errorf("empty token: %v", err)
	}
	sess, _ := h.svc.LoginWithPassword(ctx, u.Email, "correct-horse-1", ClientMeta{})
	if got, err := h.svc.Authenticate(ctx, sess.Token); err != nil || got.ID != 1 {
		t.Fatalf("valid session: %v", err)
	}

	h.users.byID[1].Status = models.UserStatusDisabled
	if _, err := h.svc.Authenticate(ctx, sess.Token); !errors.Is(err, ErrUnauthenticated) {
		t.Errorf("disabled user: %v", err)
	}
	h.users.byID[1].Status = models.UserStatusActive

	if err := h.svc.Logout(ctx, sess.Token); err != nil {
		t.Fatal(err)
	}
	if _, err := h.svc.Authenticate(ctx, sess.Token); !errors.Is(err, ErrUnauthenticated) {
		t.Errorf("after logout: %v", err)
	}

	h.sessions.rows["expired"] = &fakeSession{userID: 1, expires: time.Now().Add(-time.Minute)}
	if got, _ := h.sessions.GetValid(ctx, "expired"); got != nil {
		t.Error("fake returned an expired session")
	}
}

// ---------- CreateUser ----------

func TestCreateUser(t *testing.T) {
	h := newHarness(t, utils.AuthConfig{})
	ctx := context.Background()

	id, err := h.svc.CreateUser(ctx, CreateUserInput{Email: " An.Nguyen@MQ.com ", DisplayName: "An", Password: strp("long-enough-pw")})
	if err != nil {
		t.Fatal(err)
	}
	u := h.users.byID[id]
	if u.Email != "an.nguyen@mq.com" || u.PasswordHash == nil || *u.PasswordHash == "long-enough-pw" {
		t.Fatalf("stored user = %+v", u)
	}
	if bcrypt.CompareHashAndPassword([]byte(*u.PasswordHash), []byte("long-enough-pw")) != nil {
		t.Error("stored hash does not match the password")
	}
	if _, err := h.svc.CreateUser(ctx, CreateUserInput{Email: "b@mq.com", Password: strp("short")}); err == nil {
		t.Error("accepted a 5-char password")
	}
	if _, err := h.svc.CreateUser(ctx, CreateUserInput{Email: "not-an-email"}); err == nil {
		t.Error("accepted an invalid email")
	}
	if id, err := h.svc.CreateUser(ctx, CreateUserInput{Email: "g@mq.com"}); err != nil || h.users.byID[id].PasswordHash != nil {
		t.Errorf("google-only user: %v", err)
	}
}

// ---------- helpers ----------

func TestSanitizeRedirectPath(t *testing.T) {
	for in, want := range map[string]string{
		"":                       "/profile",
		"/orders":                "/orders",
		"/orders?page=2":         "/orders?page=2",
		"//evil.example":         "/profile",
		"/\\evil.example":        "/profile",
		"https://evil.example/x": "/profile",
		"/x/https://evil":        "/profile",
		"evil":                   "/profile",
		"/login?next=/profile":   "/profile",
		"/a\r\nSet-Cookie: x":    "/profile",
	} {
		if got := SanitizeRedirectPath(in); got != want {
			t.Errorf("SanitizeRedirectPath(%q) = %q, want %q", in, got, want)
		}
	}
}

// No log line from any flow may contain a secret.
func TestAuthFlows_DoNotLogSecrets(t *testing.T) {
	hook := logtest.NewGlobal()
	defer hook.Reset()
	logrus.SetLevel(logrus.DebugLevel)

	u := activeUser(1, "an.nguyen@mq.com")
	u.PasswordHash = hashPw(t, "correct-horse-1")
	h := newHarness(t, mqConfig(), u)
	ctx := context.Background()

	var secrets []string
	sess, _, err := h.googleLogin(t)
	if err != nil {
		t.Fatal(err)
	}
	secrets = append(secrets, sess.Token, hashToken(sess.Token), "auth-code", h.google.lastNonce, h.google.lastVerifier)
	for s := range h.attempts.rows {
		secrets = append(secrets, s)
	}

	sess2, _ := h.svc.LoginWithPassword(ctx, u.Email, "correct-horse-1", ClientMeta{})
	secrets = append(secrets, sess2.Token, "correct-horse-1", *u.PasswordHash)
	h.svc.LoginWithPassword(ctx, u.Email, "wrong-but-secret-pw", ClientMeta{})
	secrets = append(secrets, "wrong-but-secret-pw")

	h.google.err = wrapAuth(ErrExchangeFailed, errors.New("token endpoint returned 400 invalid_grant"))
	h.googleLogin(t)
	h.svc.Logout(ctx, sess.Token)
	h.svc.CleanupAuthSessions()

	if len(hook.AllEntries()) == 0 {
		t.Fatal("no log entries captured; the hook is not wired")
	}
	for _, e := range hook.AllEntries() {
		line, _ := e.String()
		for _, s := range secrets {
			if s != "" && strings.Contains(line, s) {
				t.Errorf("log line leaks a secret (%.8s…): %s", s, line)
			}
		}
	}
}
