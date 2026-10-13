package services

import (
	"context"
	"errors"
	"fmt"
	"main/component/models"
	"main/component/repositories"
	"main/utils"
	"strings"
	"testing"
	"time"
)

// Store failures must surface as INTERNAL_ERROR (never a misleading auth
// code) and must never leave a session behind.

var errDB = errors.New("db down")

type failingUsers struct {
	*fakeUsers
	getByEmail, getBySub, create, link, recordFailure, markSuccess error
}

func (f *failingUsers) GetByEmail(ctx context.Context, e string) (*models.AppUser, error) {
	if f.getByEmail != nil {
		return nil, f.getByEmail
	}
	return f.fakeUsers.GetByEmail(ctx, e)
}
func (f *failingUsers) GetByGoogleSub(ctx context.Context, s string) (*models.AppUser, error) {
	if f.getBySub != nil {
		return nil, f.getBySub
	}
	return f.fakeUsers.GetByGoogleSub(ctx, s)
}
func (f *failingUsers) Create(ctx context.Context, u models.AppUser) (int64, error) {
	if f.create != nil {
		return 0, f.create
	}
	return f.fakeUsers.Create(ctx, u)
}
func (f *failingUsers) LinkGoogle(ctx context.Context, id int64, sub, name string, a *string) (bool, error) {
	if f.link != nil {
		return false, f.link
	}
	return f.fakeUsers.LinkGoogle(ctx, id, sub, name, a)
}
func (f *failingUsers) RecordLoginFailure(ctx context.Context, id int64, max int, d time.Duration) (int, *time.Time, error) {
	if f.recordFailure != nil {
		return 0, nil, f.recordFailure
	}
	return f.fakeUsers.RecordLoginFailure(ctx, id, max, d)
}
func (f *failingUsers) MarkLoginSuccess(ctx context.Context, id int64) (time.Time, error) {
	if f.markSuccess != nil {
		return time.Time{}, f.markSuccess
	}
	return f.fakeUsers.MarkLoginSuccess(ctx, id)
}

type failingSessions struct {
	*fakeSessions
	create, getValid, revoke, deleteStale error
	panicOnDelete                         bool
}

func (f *failingSessions) Create(ctx context.Context, h string, id int64, m models.AuthMethod, e time.Time, ua, ip string) error {
	if f.create != nil {
		return f.create
	}
	return f.fakeSessions.Create(ctx, h, id, m, e, ua, ip)
}
func (f *failingSessions) GetValid(ctx context.Context, h string) (*models.SessionUser, error) {
	if f.getValid != nil {
		return nil, f.getValid
	}
	return f.fakeSessions.GetValid(ctx, h)
}
func (f *failingSessions) Revoke(ctx context.Context, h string) error {
	if f.revoke != nil {
		return f.revoke
	}
	return f.fakeSessions.Revoke(ctx, h)
}
func (f *failingSessions) DeleteStale(ctx context.Context, d time.Duration) (int64, error) {
	if f.panicOnDelete {
		panic("boom")
	}
	if f.deleteStale != nil {
		return 0, f.deleteStale
	}
	return 3, nil
}

type failingAttempts struct {
	*fakeAttempts
	create, consume, deleteExpired error
}

func (f *failingAttempts) Create(ctx context.Context, a models.LoginAttempt) error {
	if f.create != nil {
		return f.create
	}
	return f.fakeAttempts.Create(ctx, a)
}
func (f *failingAttempts) Consume(ctx context.Context, s string) (*models.LoginAttempt, error) {
	if f.consume != nil {
		return nil, f.consume
	}
	return f.fakeAttempts.Consume(ctx, s)
}
func (f *failingAttempts) DeleteExpired(ctx context.Context) (int64, error) {
	if f.deleteExpired != nil {
		return 0, f.deleteExpired
	}
	return 1, nil
}

type failingHarness struct {
	*harness
	users    *failingUsers
	sessions *failingSessions
	attempts *failingAttempts
}

func newFailingHarness(t *testing.T, cfg utils.AuthConfig, users ...models.AppUser) *failingHarness {
	t.Helper()
	h := newHarness(t, cfg, users...)
	f := &failingHarness{
		harness:  h,
		users:    &failingUsers{fakeUsers: h.users},
		sessions: &failingSessions{fakeSessions: h.sessions},
		attempts: &failingAttempts{fakeAttempts: h.attempts},
	}
	svc := NewAuthService(cfg, f.users, f.sessions, f.attempts, h.google).(*AuthServiceImpl)
	svc.bcryptCost = h.svc.bcryptCost
	f.svc = svc
	return f
}

func pwUser(t *testing.T) models.AppUser {
	u := activeUser(1, "an.nguyen@mq.com")
	u.PasswordHash = hashPw(t, "correct-horse-1")
	return u
}

func wantInternal(t *testing.T, what string, err error) {
	t.Helper()
	if !errors.Is(err, ErrInternal) {
		t.Errorf("%s: err = %v, want INTERNAL_ERROR", what, err)
	}
	if err != nil && !errors.Is(err, errDB) {
		t.Errorf("%s: cause not kept for the logs: %v", what, err)
	}
}

func TestGoogleLogin_StoreFailures(t *testing.T) {
	cases := map[string]func(f *failingHarness){
		"get by sub":   func(f *failingHarness) { f.users.getBySub = errDB },
		"get by email": func(f *failingHarness) { f.users.getByEmail = errDB },
		"link google":  func(f *failingHarness) { f.users.link = errDB },
		"mark success": func(f *failingHarness) { f.users.markSuccess = errDB },
		"create sess":  func(f *failingHarness) { f.sessions.create = errDB },
	}
	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			f := newFailingHarness(t, mqConfig(), activeUser(1, "an.nguyen@mq.com"))
			setup(f)
			sess, _, err := f.googleLogin(t)
			wantInternal(t, name, err)
			if sess != nil || len(f.harness.sessions.rows) != 0 {
				t.Error("session created despite a store failure")
			}
		})
	}
}

func TestGoogleLogin_AutoProvisionCreateFails(t *testing.T) {
	cfg := mqConfig()
	cfg.Google.AutoProvision = true
	f := newFailingHarness(t, cfg)
	f.users.create = errDB
	_, _, err := f.googleLogin(t)
	wantInternal(t, "auto-provision", err)
}

func TestBeginGoogleLogin_Failures(t *testing.T) {
	f := newFailingHarness(t, mqConfig())
	f.attempts.create = errDB
	url, err := f.svc.BeginGoogleLogin(context.Background(), "/profile", false)
	wantInternal(t, "store attempt", err)
	if url != "" {
		t.Error("returned an authorize URL although the attempt was not stored")
	}

	f = newFailingHarness(t, mqConfig())
	f.svc.rand = strings.NewReader("too short") // crypto/rand failure
	if _, err := f.svc.BeginGoogleLogin(context.Background(), "", false); !errors.Is(err, ErrInternal) {
		t.Errorf("rand failure: %v", err)
	}
	if len(f.harness.attempts.rows) != 0 {
		t.Error("attempt stored with weak randomness")
	}
}

func TestCompleteGoogleLogin_InputErrors(t *testing.T) {
	ctx := context.Background()

	f := newFailingHarness(t, mqConfig())
	f.attempts.consume = errDB
	_, _, err := f.svc.CompleteGoogleLogin(ctx, GoogleCallbackInput{Code: "c", State: "s"})
	wantInternal(t, "consume", err)

	h := newHarness(t, mqConfig(), activeUser(1, "an.nguyen@mq.com"))
	start := func() string {
		h.svc.BeginGoogleLogin(ctx, "", false)
		for s := range h.attempts.rows {
			return s
		}
		return ""
	}
	if _, _, err := h.svc.CompleteGoogleLogin(ctx, GoogleCallbackInput{State: start(), ProviderError: "server_error"}); !errors.Is(err, ErrExchangeFailed) {
		t.Errorf("provider server_error: %v, want EXCHANGE_FAILED", err)
	}
	if _, _, err := h.svc.CompleteGoogleLogin(ctx, GoogleCallbackInput{State: start()}); !errors.Is(err, ErrInvalidRequest) {
		t.Errorf("missing code: %v, want INVALID_REQUEST", err)
	}
	if h.google.exchanges != 0 {
		t.Error("token endpoint called for an invalid callback")
	}
}

func TestPasswordLogin_StoreFailures(t *testing.T) {
	ctx := context.Background()

	f := newFailingHarness(t, utils.AuthConfig{}, pwUser(t))
	f.users.getByEmail = errDB
	_, err := f.svc.LoginWithPassword(ctx, "an.nguyen@mq.com", "correct-horse-1", ClientMeta{})
	wantInternal(t, "lookup", err)

	f = newFailingHarness(t, utils.AuthConfig{}, pwUser(t))
	f.users.recordFailure = errDB
	_, err = f.svc.LoginWithPassword(ctx, "an.nguyen@mq.com", "wrong-password", ClientMeta{})
	wantInternal(t, "record failure", err)

	f = newFailingHarness(t, utils.AuthConfig{}, pwUser(t))
	f.sessions.create = errDB
	sess, err := f.svc.LoginWithPassword(ctx, "an.nguyen@mq.com", "correct-horse-1", ClientMeta{})
	wantInternal(t, "create session", err)
	if sess != nil {
		t.Error("returned a session that was never stored")
	}
}

func TestPasswordLogin_StoresTruncatedClientMeta(t *testing.T) {
	var gotUA, gotIP string
	h := newHarness(t, utils.AuthConfig{}, pwUser(t))
	rec := &recordingSessions{fakeSessions: h.sessions, ua: &gotUA, ip: &gotIP}
	svc := NewAuthService(utils.AuthConfig{}, h.users, rec, h.attempts, nil).(*AuthServiceImpl)

	_, err := svc.LoginWithPassword(context.Background(), "an.nguyen@mq.com", "correct-horse-1",
		ClientMeta{UserAgent: strings.Repeat("u", 2000), IP: strings.Repeat("9", 100)})
	if err != nil {
		t.Fatal(err)
	}
	if len(gotUA) != 512 || len(gotIP) != 64 {
		t.Errorf("stored ua=%d ip=%d chars, want 512/64", len(gotUA), len(gotIP))
	}
}

type recordingSessions struct {
	*fakeSessions
	ua, ip *string
}

func (r *recordingSessions) Create(ctx context.Context, h string, id int64, m models.AuthMethod, e time.Time, ua, ip string) error {
	*r.ua, *r.ip = ua, ip
	return r.fakeSessions.Create(ctx, h, id, m, e, ua, ip)
}

func TestSessionStoreFailures(t *testing.T) {
	ctx := context.Background()
	f := newFailingHarness(t, utils.AuthConfig{})
	f.sessions.getValid = errDB
	_, err := f.svc.Authenticate(ctx, "some-token")
	wantInternal(t, "authenticate", err)

	f.sessions.revoke = errDB
	wantInternal(t, "logout", f.svc.Logout(ctx, "some-token"))

	if err := f.svc.Logout(ctx, ""); err != nil {
		t.Errorf("logout without a token should be a no-op, got %v", err)
	}
}

func TestSessionTTLFromConfig(t *testing.T) {
	h := newHarness(t, utils.AuthConfig{SessionTTLHours: 2}, pwUser(t))
	fixed := time.Date(2026, 10, 14, 9, 0, 0, 0, time.UTC)
	h.svc.now = func() time.Time { return fixed }
	sess, err := h.svc.LoginWithPassword(context.Background(), "an.nguyen@mq.com", "correct-horse-1", ClientMeta{})
	if err != nil {
		t.Fatal(err)
	}
	if want := fixed.Add(2 * time.Hour); !sess.ExpiresAt.Equal(want) {
		t.Errorf("ExpiresAt = %v, want %v", sess.ExpiresAt, want)
	}
	if got := h.sessions.rows[hashToken(sess.Token)].expires; !got.Equal(sess.ExpiresAt) {
		t.Errorf("stored expiry %v != returned %v", got, sess.ExpiresAt)
	}
}

func TestSessionTokensAreUnique(t *testing.T) {
	h := newHarness(t, utils.AuthConfig{}, pwUser(t))
	seen := map[string]bool{}
	for i := 0; i < 20; i++ {
		s, err := h.svc.LoginWithPassword(context.Background(), "an.nguyen@mq.com", "correct-horse-1", ClientMeta{})
		if err != nil {
			t.Fatal(err)
		}
		if seen[s.Token] || len(s.Token) != 43 {
			t.Fatalf("token %d reused or wrong length (%d)", i, len(s.Token))
		}
		seen[s.Token] = true
	}
}

func TestCreateUser_Errors(t *testing.T) {
	ctx := context.Background()
	f := newFailingHarness(t, utils.AuthConfig{})
	f.users.create = repositories.ErrDuplicateEmail
	_, err := f.svc.CreateUser(ctx, CreateUserInput{Email: "an@mq.com"})
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Errorf("duplicate email: %v", err)
	}

	h := newHarness(t, utils.AuthConfig{})
	long := strings.Repeat("x", 73)
	if _, err := h.svc.CreateUser(ctx, CreateUserInput{Email: "a@mq.com", Password: &long}); err == nil {
		t.Error("accepted a 73-byte password (bcrypt would silently truncate it)")
	}
	exact := strings.Repeat("x", 72)
	if _, err := h.svc.CreateUser(ctx, CreateUserInput{Email: "b@mq.com", Password: &exact}); err != nil {
		t.Errorf("72-byte password rejected: %v", err)
	}
	if _, err := h.svc.CreateUser(ctx, CreateUserInput{Email: "An <an@mq.com>"}); err == nil {
		t.Error("accepted a display-name address form as email")
	}
}

func TestCleanupAuthSessions(t *testing.T) {
	f := newFailingHarness(t, utils.AuthConfig{})
	f.svc.CleanupAuthSessions() // happy path

	f.sessions.deleteStale = errDB
	f.attempts.deleteExpired = errDB
	f.svc.CleanupAuthSessions() // errors are logged, not returned or panicked

	f.sessions.panicOnDelete = true
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("panic escaped the worker (would kill the whole process): %v", r)
			}
		}()
		f.svc.CleanupAuthSessions()
	}()
}

func TestDomainAllowed(t *testing.T) {
	svc := &AuthServiceImpl{cfg: utils.AuthConfig{Google: utils.GoogleAuthConfig{AllowedDomains: []string{" @MQ.com ", ""}}}}
	cases := []struct {
		email, hd string
		want      bool
	}{
		{"an@mq.com", "mq.com", true},
		{"an@mq.com", "MQ.COM", true},
		{"an@mq.com", "", false},               // personal-looking token without hd
		{"an@gmail.com", "mq.com", false},      // hd and email disagree
		{"an@sub.mq.com", "sub.mq.com", false}, // subdomains are not the same domain
		{"an@evilmq.com", "evilmq.com", false},
		{"no-at-sign", "mq.com", false},
	}
	for _, tc := range cases {
		if got := svc.domainAllowed(tc.email, tc.hd); got != tc.want {
			t.Errorf("domainAllowed(%q, hd=%q) = %v, want %v", tc.email, tc.hd, got, tc.want)
		}
	}
	open := &AuthServiceImpl{}
	if !open.domainAllowed("x@gmail.com", "") {
		t.Error("empty allowed-domains must allow any domain")
	}
}

func TestAuthErrorHelpers(t *testing.T) {
	cause := fmt.Errorf("wrapped: %w", errDB)
	err := wrapAuth(ErrExchangeFailed, cause)

	if err.Error() != "EXCHANGE_FAILED: wrapped: db down" {
		t.Errorf("Error() = %q", err.Error())
	}
	if !errors.Is(err, errDB) || errors.Unwrap(err) != cause {
		t.Error("cause not reachable via Unwrap")
	}
	if !errors.Is(err, ErrExchangeFailed) || errors.Is(err, ErrInvalidState) {
		t.Error("Is must match on Code only")
	}
	if ErrExchangeFailed.Error() != "EXCHANGE_FAILED" {
		t.Errorf("sentinel Error() = %q", ErrExchangeFailed.Error())
	}

	if ae := AsAuthError(fmt.Errorf("outer: %w", err)); ae.Code != "EXCHANGE_FAILED" || ae.Status != 502 {
		t.Errorf("AsAuthError(wrapped) = %+v", ae)
	}
	if ae := AsAuthError(errDB); ae.Code != "INTERNAL_ERROR" || ae.Status != 500 || !errors.Is(ae, errDB) {
		t.Errorf("AsAuthError(plain) = %+v", ae)
	}
	if wrapAuth(ErrInternal, nil).Error() != "INTERNAL_ERROR" {
		t.Error("nil cause should print the bare code")
	}
}

func TestSanitizeOAuthErr(t *testing.T) {
	err := sanitizeOAuthErr(errors.New(`Post "https://oauth2.googleapis.com/token": code=SECRETCODE`))
	if strings.Contains(err.Error(), "SECRETCODE") {
		t.Errorf("transport error leaked details: %v", err)
	}
}
