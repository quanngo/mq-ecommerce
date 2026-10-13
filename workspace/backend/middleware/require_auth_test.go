package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"main/component/models"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type fakeAuthenticator struct {
	valid    string
	err      error
	gotToken string
}

func (f *fakeAuthenticator) Authenticate(_ context.Context, token string) (*models.SessionUser, error) {
	f.gotToken = token
	if f.err != nil {
		return nil, f.err
	}
	if token != "" && token == f.valid {
		return &models.SessionUser{AppUser: models.AppUser{ID: 7, Email: "an.nguyen@mq.com"}, AuthMethod: models.AuthMethodGoogle}, nil
	}
	return nil, nil
}

type body struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func serve(t *testing.T, mw gin.HandlerFunc, method string, headers map[string]string) (reached bool, user *models.SessionUser, b body) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	app := gin.New()
	app.Handle(method, "/x", mw, func(c *gin.Context) {
		reached = true
		user = CurrentUser(c)
		c.String(http.StatusOK, "handler")
	})
	req := httptest.NewRequest(method, "/x", nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("HTTP status %d, want 200 (the api envelope always uses 200)", w.Code)
	}
	if !reached {
		if err := json.Unmarshal(w.Body.Bytes(), &b); err != nil {
			t.Fatalf("rejection body is not the api envelope: %q", w.Body.String())
		}
	}
	return reached, user, b
}

func TestRequireAuth(t *testing.T) {
	cases := []struct {
		name    string
		auth    *fakeAuthenticator
		cookie  string
		reached bool
	}{
		{"valid session", &fakeAuthenticator{valid: "tok"}, "ecom_session=tok", true},
		{"no cookie", &fakeAuthenticator{valid: "tok"}, "", false},
		{"unknown token", &fakeAuthenticator{valid: "tok"}, "ecom_session=forged", false},
		{"other cookie name", &fakeAuthenticator{valid: "tok"}, "session=tok", false},
		{"authenticator error", &fakeAuthenticator{err: errors.New("db down")}, "ecom_session=tok", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := map[string]string{}
			if tc.cookie != "" {
				h["Cookie"] = tc.cookie
			}
			reached, user, b := serve(t, RequireAuth(tc.auth, "ecom_session"), http.MethodGet, h)
			if reached != tc.reached {
				t.Fatalf("handler reached = %v, want %v", reached, tc.reached)
			}
			if tc.reached {
				if user == nil || user.ID != 7 {
					t.Errorf("CurrentUser = %+v", user)
				}
				return
			}
			if b.Code != http.StatusUnauthorized || b.Message != "UNAUTHENTICATED" {
				t.Errorf("body = %+v, want {401 UNAUTHENTICATED}", b)
			}
		})
	}
}

func TestRequireAuth_PassesRawCookieValue(t *testing.T) {
	auth := &fakeAuthenticator{valid: "abc"}
	serve(t, RequireAuth(auth, "ecom_session"), http.MethodGet, map[string]string{"Cookie": "other=1; ecom_session=abc"})
	if auth.gotToken != "abc" {
		t.Errorf("Authenticate got %q, want the ecom_session value", auth.gotToken)
	}
}

func TestCurrentUser_WithoutRequireAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	if CurrentUser(c) != nil {
		t.Error("CurrentUser should be nil when RequireAuth did not run")
	}
	c.Set(ContextUserKey, "not a user")
	if CurrentUser(c) != nil {
		t.Error("CurrentUser should be nil for a value of the wrong type")
	}
}

func TestRequireSameOrigin(t *testing.T) {
	mw := RequireSameOrigin("http://localhost:5173/")
	cases := []struct {
		origin  string
		reached bool
	}{
		{"", true}, // same-origin GET/navigation without Origin, curl, server-to-server
		{"http://localhost:5173", true},
		{"http://localhost:5173/", true},
		{"HTTP://LOCALHOST:5173", true},
		{"https://localhost:5173", false}, // scheme differs
		{"http://localhost:5174", false},  // port differs
		{"http://localhost:5173.evil.example", false},
		{"https://evil.example", false},
		{"null", false},
	}
	for _, tc := range cases {
		h := map[string]string{}
		if tc.origin != "" {
			h["Origin"] = tc.origin
		}
		reached, _, b := serve(t, mw, http.MethodPost, h)
		if reached != tc.reached {
			t.Errorf("Origin %q: reached = %v, want %v", tc.origin, reached, tc.reached)
		}
		if !tc.reached && (b.Code != http.StatusForbidden || b.Message != "FORBIDDEN_ORIGIN") {
			t.Errorf("Origin %q: body = %+v", tc.origin, b)
		}
	}
}

func TestRequireSameOrigin_MisconfiguredFrontendURLRejectsCrossOrigin(t *testing.T) {
	// An unparsable frontend-url must fail closed, not open.
	reached, _, _ := serve(t, RequireSameOrigin("not a url"), http.MethodPost, map[string]string{"Origin": "https://evil.example"})
	if reached {
		t.Error("cross-origin request allowed with a broken frontend-url")
	}
}

func TestOriginOf(t *testing.T) {
	for in, want := range map[string]string{
		"http://localhost:5173":             "http://localhost:5173",
		"https://admin.mq.com/app/?x=1#top": "https://admin.mq.com",
		"admin.mq.com":                      "",
		"":                                  "",
		"://broken":                         "",
	} {
		if got := originOf(in); got != want {
			t.Errorf("originOf(%q) = %q, want %q", in, got, want)
		}
	}
}
