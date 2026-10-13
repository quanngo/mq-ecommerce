package rest

import (
	"context"
	"encoding/json"
	"errors"
	"main/component/models"
	"main/component/services"
	"main/utils"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type fakeAuthSvc struct {
	google       bool
	beginURL     string
	beginErr     error
	completeErr  error
	loginErr     error
	gotNext      string
	gotCallback  services.GoogleCallbackInput
	validToken   string
	loggedOut    string
	logoutErr    error
	gotRemember  bool
	gotMeta      services.ClientMeta
	logoutAllFor int64
	logoutAllErr error
	redirectPath string
}

var testUser = models.SessionUser{
	AppUser:    models.AppUser{ID: 7, Email: "an.nguyen@mq.com", DisplayName: "An Nguyen", Status: models.UserStatusActive},
	AuthMethod: models.AuthMethodPassword,
}

func (f *fakeAuthSvc) Providers() models.ProvidersResponse {
	return models.ProvidersResponse{Google: f.google, Password: true}
}
func (f *fakeAuthSvc) BeginGoogleLogin(_ context.Context, next string, remember bool) (string, error) {
	f.gotNext = next
	f.gotRemember = remember
	return f.beginURL, f.beginErr
}
func (f *fakeAuthSvc) CompleteGoogleLogin(_ context.Context, in services.GoogleCallbackInput) (*models.NewSession, string, error) {
	f.gotCallback = in
	if f.completeErr != nil {
		return nil, "", f.completeErr
	}
	return &models.NewSession{Token: "tok-google", ExpiresAt: time.Now().Add(time.Hour), User: testUser}, f.redirectPath, nil
}
func (f *fakeAuthSvc) LoginWithPassword(_ context.Context, email, password string, meta services.ClientMeta) (*models.NewSession, error) {
	f.gotMeta = meta
	if f.loginErr != nil {
		return nil, f.loginErr
	}
	return &models.NewSession{Token: "tok-pw", ExpiresAt: time.Now().Add(12 * time.Hour), Persistent: meta.RememberMe, User: testUser}, nil
}
func (f *fakeAuthSvc) LogoutAll(_ context.Context, userID int64) (int64, error) {
	f.logoutAllFor = userID
	return 3, f.logoutAllErr
}
func (f *fakeAuthSvc) Authenticate(_ context.Context, token string) (*models.SessionUser, error) {
	if token != "" && token == f.validToken {
		u := testUser
		return &u, nil
	}
	return nil, services.ErrUnauthenticated
}
func (f *fakeAuthSvc) Logout(_ context.Context, token string) error {
	f.loggedOut = token
	return f.logoutErr
}
func (f *fakeAuthSvc) CreateUser(context.Context, services.CreateUserInput) (int64, error) {
	return 0, nil
}
func (f *fakeAuthSvc) CleanupAuthSessions() {}

func newTestRouter(svc services.AuthService, secure bool) *gin.Engine {
	gin.SetMode(gin.TestMode)
	app := gin.New()
	APIRoutes(app, &ApiServer{
		Cfg:     &utils.Config{Auth: utils.AuthConfig{FrontendURL: "http://localhost:5173/", CookieSecure: secure}},
		AuthSvc: svc,
	})
	return app
}

type apiBody struct {
	Code    int               `json:"code"`
	Message string            `json:"message"`
	Data    []json.RawMessage `json:"data"`
}

func do(app *gin.Engine, method, path, body string, headers map[string]string) (*httptest.ResponseRecorder, apiBody) {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	var b apiBody
	json.Unmarshal(w.Body.Bytes(), &b)
	return w, b
}

func sessionCookie(w *httptest.ResponseRecorder) *http.Cookie {
	for _, c := range w.Result().Cookies() {
		if c.Name == "ecom_session" {
			return c
		}
	}
	return nil
}

func TestPasswordLogin_SetsHardenedCookie(t *testing.T) {
	app := newTestRouter(&fakeAuthSvc{}, true)
	w, body := do(app, http.MethodPost, "/api/v1/auth/login", `{"email":"an.nguyen@mq.com","password":"correct-horse-1","remember_me":true}`,
		map[string]string{"Origin": "http://localhost:5173"})

	if w.Code != http.StatusOK || body.Code != http.StatusOK || len(body.Data) != 1 {
		t.Fatalf("status %d body %+v", w.Code, body)
	}
	var user models.UserResponse
	json.Unmarshal(body.Data[0], &user)
	if user.ID != 7 || user.AuthMethod != models.AuthMethodPassword {
		t.Errorf("user = %+v", user)
	}
	c := sessionCookie(w)
	if c == nil || c.Value != "tok-pw" || !c.HttpOnly || !c.Secure || c.SameSite != http.SameSiteLaxMode || c.Path != "/" || c.MaxAge <= 0 {
		t.Fatalf("cookie = %+v", c)
	}
	if strings.Contains(w.Body.String(), "tok-pw") || strings.Contains(w.Body.String(), "correct-horse-1") {
		t.Error("token or password field in the response body")
	}
}

func TestPasswordLogin_Errors(t *testing.T) {
	cases := []struct {
		name, body, origin string
		svcErr             error
		wantCode           int
		wantMsg            string
	}{
		{"bad json", `{"email":`, "", nil, 400, "INVALID_REQUEST"},
		{"not an email", `{"email":"x","password":"secret-pw-123"}`, "", nil, 400, "INVALID_REQUEST"},
		{"missing password", `{"email":"a@mq.com"}`, "", nil, 400, "INVALID_REQUEST"},
		{"wrong credentials", `{"email":"a@mq.com","password":"secret-pw-123"}`, "", services.ErrInvalidCredentials, 401, "INVALID_CREDENTIALS"},
		{"locked", `{"email":"a@mq.com","password":"secret-pw-123"}`, "", services.ErrAccountLocked, 423, "ACCOUNT_LOCKED"},
		{"cross origin", `{"email":"a@mq.com","password":"secret-pw-123"}`, "https://evil.example", nil, 403, "FORBIDDEN_ORIGIN"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := newTestRouter(&fakeAuthSvc{loginErr: tc.svcErr}, false)
			h := map[string]string{}
			if tc.origin != "" {
				h["Origin"] = tc.origin
			}
			w, body := do(app, http.MethodPost, "/api/v1/auth/login", tc.body, h)
			if w.Code != http.StatusOK || body.Code != tc.wantCode || body.Message != tc.wantMsg {
				t.Fatalf("got http %d body %+v, want code %d %s", w.Code, body, tc.wantCode, tc.wantMsg)
			}
			if sessionCookie(w) != nil {
				t.Error("cookie set on failure")
			}
			if strings.Contains(w.Body.String(), "secret-pw-123") {
				t.Error("password echoed in the response")
			}
		})
	}
}

func TestGoogleLogin_Redirects(t *testing.T) {
	svc := &fakeAuthSvc{beginURL: "https://accounts.google.com/o/oauth2/v2/auth?state=s"}
	app := newTestRouter(svc, false)
	w, _ := do(app, http.MethodGet, "/api/v1/auth/google/login?next=/orders", "", nil)
	if w.Code != http.StatusFound || w.Header().Get("Location") != svc.beginURL || svc.gotNext != "/orders" {
		t.Fatalf("status %d location %q next %q", w.Code, w.Header().Get("Location"), svc.gotNext)
	}

	app = newTestRouter(&fakeAuthSvc{beginErr: services.ErrGoogleDisabled}, false)
	w, _ = do(app, http.MethodGet, "/api/v1/auth/google/login", "", nil)
	if w.Header().Get("Location") != "http://localhost:5173/login?error=GOOGLE_DISABLED" {
		t.Errorf("location = %q", w.Header().Get("Location"))
	}
}

func TestGoogleCallback(t *testing.T) {
	svc := &fakeAuthSvc{redirectPath: "/profile"}
	app := newTestRouter(svc, false)
	w, _ := do(app, http.MethodGet, "/api/v1/auth/google/callback?code=c1&state=s1", "", nil)
	if w.Code != http.StatusFound || w.Header().Get("Location") != "http://localhost:5173/profile" {
		t.Fatalf("status %d location %q", w.Code, w.Header().Get("Location"))
	}
	if c := sessionCookie(w); c == nil || c.Value != "tok-google" || !c.HttpOnly {
		t.Errorf("cookie = %+v", c)
	}
	if svc.gotCallback.Code != "c1" || svc.gotCallback.State != "s1" {
		t.Errorf("callback input = %+v", svc.gotCallback)
	}
	if w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("Referrer-Policy") != "no-referrer" {
		t.Error("missing no-store / no-referrer headers")
	}

	for _, e := range []*services.AuthError{services.ErrInvalidState, services.ErrConsentDenied, services.ErrUserNotAllowed, services.ErrDomainNotAllowed, services.ErrExchangeFailed} {
		app := newTestRouter(&fakeAuthSvc{completeErr: e}, false)
		w, _ := do(app, http.MethodGet, "/api/v1/auth/google/callback?error=access_denied&state=s1", "", nil)
		if want := "http://localhost:5173/login?error=" + e.Code; w.Code != http.StatusFound || w.Header().Get("Location") != want {
			t.Errorf("%s: status %d location %q", e.Code, w.Code, w.Header().Get("Location"))
		}
		if sessionCookie(w) != nil {
			t.Errorf("%s: cookie set on failure", e.Code)
		}
	}
}

func TestMeAndProtectedGroups(t *testing.T) {
	svc := &fakeAuthSvc{validToken: "tok-ok"}
	app := newTestRouter(svc, false)

	// The tiktok/lazada groups also use requireAuth, but register no routes yet, so there is nothing to call.
	for _, path := range []string{"/api/v1/auth/me"} {
		_, body := do(app, http.MethodGet, path, "", nil)
		if body.Code != http.StatusUnauthorized || body.Message != "UNAUTHENTICATED" {
			t.Errorf("%s without cookie: %+v", path, body)
		}
		_, body = do(app, http.MethodGet, path, "", map[string]string{"Cookie": "ecom_session=forged"})
		if body.Code != http.StatusUnauthorized {
			t.Errorf("%s with forged cookie: %+v", path, body)
		}
	}

	_, body := do(app, http.MethodGet, "/api/v1/auth/me", "", map[string]string{"Cookie": "ecom_session=tok-ok"})
	if body.Code != http.StatusOK || len(body.Data) != 1 {
		t.Fatalf("me: %+v", body)
	}
	var user models.UserResponse
	json.Unmarshal(body.Data[0], &user)
	if user.Email != "an.nguyen@mq.com" {
		t.Errorf("me user = %+v", user)
	}

	_, body = do(app, http.MethodGet, "/api/v1/auth/providers", "", nil)
	if body.Code != http.StatusOK {
		t.Errorf("providers must be public: %+v", body)
	}
}

func TestLogout(t *testing.T) {
	svc := &fakeAuthSvc{validToken: "tok-ok"}
	app := newTestRouter(svc, false)
	w, body := do(app, http.MethodPost, "/api/v1/auth/logout", "", map[string]string{"Cookie": "ecom_session=tok-ok"})
	if body.Code != http.StatusOK || svc.loggedOut != "tok-ok" {
		t.Fatalf("logout: %+v revoked=%q", body, svc.loggedOut)
	}
	if c := sessionCookie(w); c == nil || c.MaxAge >= 0 || c.Value != "" {
		t.Errorf("cookie not cleared: %+v", c)
	}

	_, body = do(app, http.MethodPost, "/api/v1/auth/logout", "", nil)
	if body.Code != http.StatusUnauthorized {
		t.Errorf("logout without cookie: %+v", body)
	}
	_, body = do(app, http.MethodPost, "/api/v1/auth/logout", "", map[string]string{"Cookie": "ecom_session=tok-ok", "Origin": "https://evil.example"})
	if body.Message != "FORBIDDEN_ORIGIN" {
		t.Errorf("cross-origin logout: %+v", body)
	}
}

func TestInternalErrorsDoNotLeakCauses(t *testing.T) {
	cause := errors.New("pq: password authentication failed for user secret_db_user")
	app := newTestRouter(&fakeAuthSvc{loginErr: cause}, false)
	w, body := do(app, http.MethodPost, "/api/v1/auth/login", `{"email":"a@mq.com","password":"secret-pw-123"}`, nil)
	if body.Code != http.StatusInternalServerError || body.Message != "INTERNAL_ERROR" {
		t.Fatalf("body = %+v, want {500 INTERNAL_ERROR}", body)
	}
	if strings.Contains(w.Body.String(), "secret_db_user") {
		t.Error("internal error cause sent to the client")
	}

	app = newTestRouter(&fakeAuthSvc{completeErr: cause}, false)
	w, _ = do(app, http.MethodGet, "/api/v1/auth/google/callback?code=c&state=s", "", nil)
	if loc := w.Header().Get("Location"); loc != "http://localhost:5173/login?error=INTERNAL_ERROR" {
		t.Errorf("callback with a plain error: location %q", loc)
	}
}

func TestLogout_ServiceErrorStillClearsCookie(t *testing.T) {
	app := newTestRouter(&fakeAuthSvc{logoutErr: services.ErrInternal}, false)
	w, body := do(app, http.MethodPost, "/api/v1/auth/logout", "", map[string]string{"Cookie": "ecom_session=tok"})
	if body.Code != http.StatusInternalServerError {
		t.Errorf("body = %+v", body)
	}
	if c := sessionCookie(w); c == nil || c.MaxAge >= 0 {
		t.Error("cookie must be cleared even when revoking fails")
	}
}

func TestMe_WithoutMiddlewareUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	(&AuthHandler{AuthSvc: &fakeAuthSvc{}, Cfg: utils.AuthConfig{}.WithDefaults()}).Me(c)
	var body apiBody
	json.Unmarshal(w.Body.Bytes(), &body)
	if body.Code != http.StatusUnauthorized || body.Message != "UNAUTHENTICATED" {
		t.Errorf("body = %+v", body)
	}
}

func TestCookieNotSecureWhenConfiguredOff(t *testing.T) {
	app := newTestRouter(&fakeAuthSvc{}, false)
	w, _ := do(app, http.MethodPost, "/api/v1/auth/login", `{"email":"a@mq.com","password":"secret-pw-123"}`, nil)
	if c := sessionCookie(w); c == nil || c.Secure || !c.HttpOnly {
		t.Errorf("cookie = %+v, want HttpOnly and not Secure for local http", c)
	}
}

func TestPasswordLogin_RememberMeControlsCookieLifetime(t *testing.T) {
	svc := &fakeAuthSvc{}
	app := newTestRouter(svc, false)

	w, _ := do(app, http.MethodPost, "/api/v1/auth/login", `{"email":"a@mq.com","password":"secret-pw-123"}`, nil)
	if svc.gotMeta.RememberMe {
		t.Error("remember_me defaulted to true")
	}
	if c := sessionCookie(w); c == nil || c.MaxAge != 0 || strings.Contains(w.Header().Get("Set-Cookie"), "Max-Age") {
		t.Errorf("without remember me the cookie must be a browser-session cookie: %q", w.Header().Get("Set-Cookie"))
	}

	w, _ = do(app, http.MethodPost, "/api/v1/auth/login", `{"email":"a@mq.com","password":"secret-pw-123","remember_me":true}`, nil)
	if !svc.gotMeta.RememberMe {
		t.Error("remember_me not passed to the service")
	}
	if c := sessionCookie(w); c == nil || c.MaxAge <= 0 {
		t.Errorf("remember me must set a Max-Age: %q", w.Header().Get("Set-Cookie"))
	}
}

func TestGoogleLogin_RememberParam(t *testing.T) {
	for query, want := range map[string]bool{"": false, "?remember=1": true, "?remember=true": false, "?remember=0": false} {
		svc := &fakeAuthSvc{beginURL: "https://accounts.google.com/x"}
		app := newTestRouter(svc, false)
		do(app, http.MethodGet, "/api/v1/auth/google/login"+query, "", nil)
		if svc.gotRemember != want {
			t.Errorf("%q: remember = %v, want %v", query, svc.gotRemember, want)
		}
	}
}

func TestLogoutAll(t *testing.T) {
	svc := &fakeAuthSvc{validToken: "tok-ok"}
	app := newTestRouter(svc, false)

	w, body := do(app, http.MethodPost, "/api/v1/auth/logout-all", "", map[string]string{"Cookie": "ecom_session=tok-ok", "Origin": "http://localhost:5173"})
	if body.Code != http.StatusOK || len(body.Data) != 1 || svc.logoutAllFor != 7 {
		t.Fatalf("body %+v, revoked for user %d", body, svc.logoutAllFor)
	}
	var resp models.LogoutAllResponse
	json.Unmarshal(body.Data[0], &resp)
	if resp.RevokedSessions != 3 {
		t.Errorf("revoked_sessions = %d", resp.RevokedSessions)
	}
	if c := sessionCookie(w); c == nil || c.MaxAge >= 0 {
		t.Errorf("cookie not cleared: %+v", c)
	}
}

func TestLogoutAll_Guards(t *testing.T) {
	svc := &fakeAuthSvc{validToken: "tok-ok"}
	app := newTestRouter(svc, false)

	_, body := do(app, http.MethodPost, "/api/v1/auth/logout-all", "", nil)
	if body.Message != "UNAUTHENTICATED" || svc.logoutAllFor != 0 {
		t.Errorf("anonymous: %+v", body)
	}
	_, body = do(app, http.MethodPost, "/api/v1/auth/logout-all", "", map[string]string{"Cookie": "ecom_session=tok-ok", "Origin": "https://evil.example"})
	if body.Message != "FORBIDDEN_ORIGIN" || svc.logoutAllFor != 0 {
		t.Errorf("cross-origin: %+v", body)
	}
	if w, _ := do(app, http.MethodGet, "/api/v1/auth/logout-all", "", map[string]string{"Cookie": "ecom_session=tok-ok"}); w.Code != http.StatusNotFound {
		t.Errorf("GET logout-all must not exist (a link or <img> could trigger it), got %d", w.Code)
	}

	svc = &fakeAuthSvc{validToken: "tok-ok", logoutAllErr: services.ErrInternal}
	app = newTestRouter(svc, false)
	w, body := do(app, http.MethodPost, "/api/v1/auth/logout-all", "", map[string]string{"Cookie": "ecom_session=tok-ok"})
	if body.Code != http.StatusInternalServerError {
		t.Errorf("service error: %+v", body)
	}
	if sessionCookie(w) != nil {
		t.Error("cookie cleared although sessions were not revoked")
	}
}
