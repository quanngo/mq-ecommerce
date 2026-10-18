package rest

import (
	"encoding/json"
	"main/component/models"
	"main/component/services"
	"main/utils"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
)

type localizedBody struct {
	Code      int               `json:"code"`
	Message   string            `json:"message"`
	ErrorText string            `json:"error_text"`
	Data      []json.RawMessage `json:"data"`
}

func doLocalized(app *gin.Engine, method, path, body string, headers map[string]string) (http.Header, localizedBody) {
	w, _ := do(app, method, path, body, headers)
	var b localizedBody
	json.Unmarshal(w.Body.Bytes(), &b)
	return w.Header(), b
}

func with(headers ...string) map[string]string {
	m := map[string]string{}
	for i := 0; i+1 < len(headers); i += 2 {
		m[headers[i]] = headers[i+1]
	}
	return m
}

func TestErrorTextFollowsAcceptLanguage(t *testing.T) {
	app := newTestRouter(&fakeAuthSvc{}, false)

	h, body := doLocalized(app, http.MethodGet, "/api/v1/auth/me", "", with("Accept-Language", "en-US,en;q=0.9"))
	if body.Message != "UNAUTHENTICATED" || body.ErrorText != "You are not signed in, or your session has expired." {
		t.Errorf("en: %+v", body)
	}
	if h.Get("Content-Language") != "en" {
		t.Errorf("Content-Language = %q", h.Get("Content-Language"))
	}

	_, body = doLocalized(app, http.MethodGet, "/api/v1/auth/me", "", nil)
	if body.Message != "UNAUTHENTICATED" || body.ErrorText != "Bạn chưa đăng nhập hoặc phiên đăng nhập đã hết hạn." {
		t.Errorf("default (vi): %+v", body)
	}
}

func TestSavedLocaleBeatsAcceptLanguage(t *testing.T) {
	user := sessionWith("admin", models.PermUsersRead, models.PermOrdersRead)
	user.RoleName = "Quản trị viên"
	user.Locale = "en"
	app := newUsersRouter(user, &fakeUsersSvc{})

	_, body := doLocalized(app, http.MethodGet, "/api/v1/auth/me", "", with("Cookie", "ecom_session=tok", "Accept-Language", "vi"))
	var me models.UserResponse
	json.Unmarshal(body.Data[0], &me)
	if me.Locale != "en" || me.Role.Name != "Administrator" {
		t.Errorf("me = %+v", me)
	}
	want := map[string]string{models.PermUsersRead: "View users", models.PermOrdersRead: "View orders"}
	for k, v := range want {
		if me.PermissionNames[k] != v {
			t.Errorf("permission_names = %v", me.PermissionNames)
		}
	}

	// A forbidden call is explained in the account's language too.
	_, body = doLocalized(app, http.MethodPost, "/api/v1/users", `{"email":"x@mq.com","role":"viewer"}`,
		with("Cookie", "ecom_session=tok", "Accept-Language", "vi", "Origin", "http://localhost:5173"))
	if body.Message != "FORBIDDEN" || body.ErrorText != "You don't have permission to do this." {
		t.Errorf("forbidden: %+v", body)
	}
}

func TestRolesAndUsersAreLocalized(t *testing.T) {
	app := newUsersRouter(asAdmin, &fakeUsersSvc{})
	hdr := with("Cookie", "ecom_session=tok")

	_, body := doLocalized(app, http.MethodGet, "/api/v1/roles", "", hdr)
	var roles models.RolesListResponse
	json.Unmarshal(body.Data[0], &roles)
	if r := roles.Roles[0]; r.Name != "Quản trị viên" || r.PermissionNames[models.PermUsersManage] != "Quản lý người dùng" {
		t.Errorf("vi roles = %+v", roles)
	}

	asAdmin.Locale = "en"
	defer func() { asAdmin.Locale = "" }()
	_, body = doLocalized(app, http.MethodGet, "/api/v1/roles", "", hdr)
	json.Unmarshal(body.Data[0], &roles)
	if r := roles.Roles[0]; r.Name != "Administrator" || r.Description != "Full access, including user management" ||
		r.PermissionNames[models.PermUsersManage] != "Manage users" {
		t.Errorf("en roles = %+v", roles)
	}

	_, body = doLocalized(app, http.MethodGet, "/api/v1/users", "", hdr)
	var users models.UsersListResponse
	json.Unmarshal(body.Data[0], &users)
	if users.Users[0].Role.Name != "Administrator" {
		t.Errorf("en users = %+v", users)
	}
}

func TestSetLocaleRoute(t *testing.T) {
	newApp := func(svc *fakeAuthSvc) *gin.Engine {
		gin.SetMode(gin.TestMode)
		app := gin.New()
		APIRoutes(app, &ApiServer{Cfg: &utils.Config{Auth: utils.AuthConfig{FrontendURL: "http://localhost:5173"}}, AuthSvc: svc})
		return app
	}
	origin := "http://localhost:5173"

	svc := &fakeAuthSvc{validToken: "tok"}
	h, body := doLocalized(newApp(svc), http.MethodPut, "/api/v1/auth/me/locale", `{"locale":"en"}`,
		with("Cookie", "ecom_session=tok", "Origin", origin))
	if body.Code != http.StatusOK || svc.gotLocale != "en" {
		t.Fatalf("set en: %+v (got %q)", body, svc.gotLocale)
	}
	var me models.UserResponse
	json.Unmarshal(body.Data[0], &me)
	if me.Locale != "en" || h.Get("Content-Language") != "en" || h.Get("Cache-Control") != "no-store" {
		t.Errorf("me = %+v, headers = %v", me, h)
	}

	cases := []struct {
		name, body string
		headers    map[string]string
		code       int
		message    string
	}{
		{"unsupported", `{"locale":"fr"}`, with("Cookie", "ecom_session=tok", "Origin", origin), http.StatusBadRequest, "INVALID_REQUEST"},
		{"missing", `{}`, with("Cookie", "ecom_session=tok", "Origin", origin), http.StatusBadRequest, "INVALID_REQUEST"},
		{"not json", `en`, with("Cookie", "ecom_session=tok", "Origin", origin), http.StatusBadRequest, "INVALID_REQUEST"},
		{"no session", `{"locale":"en"}`, with("Origin", origin), http.StatusUnauthorized, "UNAUTHENTICATED"},
		{"cross origin", `{"locale":"en"}`, with("Cookie", "ecom_session=tok", "Origin", "https://evil.example"), http.StatusForbidden, "FORBIDDEN_ORIGIN"},
	}
	for _, tc := range cases {
		svc := &fakeAuthSvc{validToken: "tok"}
		_, body := doLocalized(newApp(svc), http.MethodPut, "/api/v1/auth/me/locale", tc.body, tc.headers)
		if body.Code != tc.code || body.Message != tc.message {
			t.Errorf("%s: %+v", tc.name, body)
		}
		if tc.code != http.StatusBadRequest && svc.gotLocale != "" {
			t.Errorf("%s: service was called with %q", tc.name, svc.gotLocale)
		}
	}

	svc = &fakeAuthSvc{validToken: "tok", setLocaleErr: services.ErrInternal}
	_, body = doLocalized(newApp(svc), http.MethodPut, "/api/v1/auth/me/locale", `{"locale":"vi"}`,
		with("Cookie", "ecom_session=tok", "Origin", origin))
	if body.Message != "INTERNAL_ERROR" {
		t.Errorf("store failure: %+v", body)
	}
}

func TestPasswordLoginRespondsInAccountLanguage(t *testing.T) {
	svc := &fakeAuthSvc{}
	app := newTestRouter(svc, false)
	prev := testUser
	testUser.Locale, testUser.RoleCode, testUser.Permissions = "en", "viewer", []string{models.PermOrdersRead}
	defer func() { testUser = prev }()

	h, body := doLocalized(app, http.MethodPost, "/api/v1/auth/login", `{"email":"an.nguyen@mq.com","password":"correct-horse-1"}`,
		with("Origin", "http://localhost:5173", "Accept-Language", "vi"))
	var me models.UserResponse
	json.Unmarshal(body.Data[0], &me)
	if me.Locale != "en" || me.Role.Name != "Viewer" || me.PermissionNames[models.PermOrdersRead] != "View orders" || h.Get("Content-Language") != "en" {
		t.Errorf("login response = %+v", me)
	}
}
