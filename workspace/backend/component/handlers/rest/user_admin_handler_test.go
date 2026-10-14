package rest

import (
	"context"
	"encoding/json"
	"main/component/models"
	"main/component/services"
	"main/utils"
	"net/http"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type fakeUsersSvc struct {
	created   *models.CreateUserRequest
	updatedID int64
	updated   *models.UpdateUserRequest
	resetID   int64
	resetPw   string
	actorID   int64
	err       error
}

func (f *fakeUsersSvc) ListUsers(context.Context) ([]models.ManagedUser, error) {
	return []models.ManagedUser{{ID: 1, Email: "admin@mq.com", Role: models.RoleRef{Code: "admin", Name: "Quản trị viên"}, HasPassword: true}}, f.err
}
func (f *fakeUsersSvc) ListRoles(context.Context) ([]models.Role, error) {
	return []models.Role{{Code: "admin", Permissions: []string{models.PermUsersManage}}}, f.err
}
func (f *fakeUsersSvc) CreateUser(_ context.Context, actor *models.SessionUser, req models.CreateUserRequest) (*models.ManagedUser, error) {
	f.actorID, f.created = actor.ID, &req
	if f.err != nil {
		return nil, f.err
	}
	return &models.ManagedUser{ID: 10, Email: req.Email, Role: models.RoleRef{Code: req.Role}}, nil
}
func (f *fakeUsersSvc) UpdateUser(_ context.Context, actor *models.SessionUser, id int64, req models.UpdateUserRequest) (*models.ManagedUser, error) {
	f.actorID, f.updatedID, f.updated = actor.ID, id, &req
	if f.err != nil {
		return nil, f.err
	}
	return &models.ManagedUser{ID: id}, nil
}
func (f *fakeUsersSvc) ResetPassword(_ context.Context, actor *models.SessionUser, id int64, pw string) error {
	f.actorID, f.resetID, f.resetPw = actor.ID, id, pw
	return f.err
}

func sessionWith(role string, perms ...string) *models.SessionUser {
	u := testUser
	u.RoleCode, u.RoleName, u.Permissions = role, role, perms
	return &u
}

var (
	asAdmin  = sessionWith("admin", models.PermUsersRead, models.PermUsersManage, models.PermOrdersRead, models.PermOrdersWrite)
	asReader = sessionWith("auditor", models.PermUsersRead)
	asViewer = sessionWith("viewer", models.PermOrdersRead)
)

func newUsersRouter(user *models.SessionUser, users *fakeUsersSvc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	app := gin.New()
	APIRoutes(app, &ApiServer{
		Cfg:      &utils.Config{Auth: utils.AuthConfig{FrontendURL: "http://localhost:5173"}},
		AuthSvc:  &fakeAuthSvc{validToken: "tok", sessionUser: user},
		UsersSvc: users,
	})
	return app
}

var withSession = map[string]string{"Cookie": "ecom_session=tok"}

func TestUsersRoutes_Permissions(t *testing.T) {
	cases := []struct {
		method, path, body string
		user               *models.SessionUser
		want               int
	}{
		{http.MethodGet, "/api/v1/users", "", asAdmin, 200},
		{http.MethodGet, "/api/v1/users", "", asReader, 200},
		{http.MethodGet, "/api/v1/users", "", asViewer, 403},
		{http.MethodGet, "/api/v1/roles", "", asReader, 200},
		{http.MethodGet, "/api/v1/roles", "", asViewer, 403},
		{http.MethodPost, "/api/v1/users", `{"email":"a@mq.com","role":"viewer"}`, asAdmin, 200},
		{http.MethodPost, "/api/v1/users", `{"email":"a@mq.com","role":"viewer"}`, asReader, 403},
		{http.MethodPatch, "/api/v1/users/5", `{"role":"viewer"}`, asReader, 403},
		{http.MethodPost, "/api/v1/users/5/reset-password", `{"password":"long-enough-pw"}`, asViewer, 403},
	}
	for _, tc := range cases {
		users := &fakeUsersSvc{}
		_, body := do(newUsersRouter(tc.user, users), tc.method, tc.path, tc.body, withSession)
		if body.Code != tc.want {
			t.Errorf("%s %s as %s: code %d (%s), want %d", tc.method, tc.path, tc.user.RoleCode, body.Code, body.Message, tc.want)
		}
		if tc.want == 403 && body.Message != "FORBIDDEN" {
			t.Errorf("%s %s: message %q", tc.method, tc.path, body.Message)
		}
		if tc.want == 403 && (users.created != nil || users.updated != nil || users.resetID != 0) {
			t.Errorf("%s %s: service called despite 403", tc.method, tc.path)
		}
	}
}

func TestUsersRoutes_RequireSession(t *testing.T) {
	app := newUsersRouter(asAdmin, &fakeUsersSvc{})
	for _, path := range []string{"/api/v1/users", "/api/v1/roles"} {
		if _, body := do(app, http.MethodGet, path, "", nil); body.Message != "UNAUTHENTICATED" {
			t.Errorf("%s without session: %+v", path, body)
		}
	}
}

func TestUsersRoutes_CrossOriginChangesRejected(t *testing.T) {
	users := &fakeUsersSvc{}
	app := newUsersRouter(asAdmin, users)
	h := map[string]string{"Cookie": "ecom_session=tok", "Origin": "https://evil.example"}
	for _, r := range [][3]string{
		{http.MethodPost, "/api/v1/users", `{"email":"a@mq.com","role":"admin"}`},
		{http.MethodPatch, "/api/v1/users/5", `{"role":"admin"}`},
		{http.MethodPost, "/api/v1/users/5/reset-password", `{"password":"long-enough-pw"}`},
	} {
		if _, body := do(app, r[0], r[1], r[2], h); body.Message != "FORBIDDEN_ORIGIN" {
			t.Errorf("%s %s: %+v", r[0], r[1], body)
		}
	}
	if users.created != nil || users.updated != nil || users.resetID != 0 {
		t.Error("service called for a cross-origin request")
	}
}

func TestUsersRoutes_PassActorAndBody(t *testing.T) {
	users := &fakeUsersSvc{}
	app := newUsersRouter(asAdmin, users)

	_, body := do(app, http.MethodPost, "/api/v1/users", `{"email":"new@mq.com","display_name":"New","role":"operator","password":"long-enough-pw"}`, withSession)
	if body.Code != 200 || users.actorID != asAdmin.ID || users.created.Role != "operator" || users.created.Password == nil {
		t.Fatalf("create: %+v %+v", body, users.created)
	}
	var created models.ManagedUser
	json.Unmarshal(body.Data[0], &created)
	if created.Email != "new@mq.com" {
		t.Errorf("created = %+v", created)
	}

	_, body = do(app, http.MethodPatch, "/api/v1/users/5", `{"status":"disabled"}`, withSession)
	if body.Code != 200 || users.updatedID != 5 || users.updated.Status == nil || *users.updated.Status != models.UserStatusDisabled || users.updated.Role != nil {
		t.Errorf("update: %+v %+v", body, users.updated)
	}

	_, body = do(app, http.MethodPost, "/api/v1/users/5/reset-password", `{"password":"long-enough-pw"}`, withSession)
	if body.Code != 200 || users.resetID != 5 || users.resetPw != "long-enough-pw" {
		t.Errorf("reset: %+v", body)
	}

	_, body = do(app, http.MethodGet, "/api/v1/users", "", withSession)
	var list models.UsersListResponse
	json.Unmarshal(body.Data[0], &list)
	if len(list.Users) != 1 || list.Users[0].Role.Code != "admin" {
		t.Errorf("list = %+v", list)
	}
	if strings.Contains(string(body.Data[0]), "password_hash") || strings.Contains(string(body.Data[0]), "google_sub") {
		t.Error("list leaks credential fields")
	}
}

func TestUsersRoutes_BadInput(t *testing.T) {
	users := &fakeUsersSvc{}
	app := newUsersRouter(asAdmin, users)
	cases := [][3]string{
		{http.MethodPost, "/api/v1/users", `{"email":"not-an-email","role":"viewer","password":"secret-pw-123"}`},
		{http.MethodPost, "/api/v1/users", `{"email":"a@mq.com"}`}, // role missing
		{http.MethodPatch, "/api/v1/users/5", `{"status":"deleted"}`},
		{http.MethodPatch, "/api/v1/users/abc", `{"role":"viewer"}`},
		{http.MethodPost, "/api/v1/users/0/reset-password", `{"password":"secret-pw-123"}`},
		{http.MethodPost, "/api/v1/users/5/reset-password", `{}`},
	}
	for _, c := range cases {
		w, body := do(app, c[0], c[1], c[2], withSession)
		if body.Code != 400 && body.Code != 404 {
			t.Errorf("%s %s %s: %+v", c[0], c[1], c[2], body)
		}
		if strings.Contains(w.Body.String(), "secret-pw-123") {
			t.Errorf("%s %s echoes the password", c[0], c[1])
		}
	}
}

func TestUsersRoutes_ServiceErrorsKeepTheirCode(t *testing.T) {
	for _, e := range []*services.AuthError{services.ErrLastAdmin, services.ErrSelfModify, services.ErrEmailTaken, services.ErrInvalidRole, services.ErrWeakPassword, services.ErrNotFound} {
		app := newUsersRouter(asAdmin, &fakeUsersSvc{err: e})
		_, body := do(app, http.MethodPatch, "/api/v1/users/5", `{"role":"viewer"}`, withSession)
		if body.Code != e.Status || body.Message != e.Code {
			t.Errorf("%s: %+v", e.Code, body)
		}
	}
}

func TestMe_IncludesRoleAndPermissions(t *testing.T) {
	app := newUsersRouter(asAdmin, &fakeUsersSvc{})
	_, body := do(app, http.MethodGet, "/api/v1/auth/me", "", withSession)
	var me models.UserResponse
	json.Unmarshal(body.Data[0], &me)
	if me.Role.Code != "admin" || len(me.Permissions) != 4 {
		t.Errorf("me = %+v", me)
	}

	app = newUsersRouter(sessionWith("viewer"), &fakeUsersSvc{})
	_, body = do(app, http.MethodGet, "/api/v1/auth/me", "", withSession)
	if !strings.Contains(string(body.Data[0]), `"permissions":[]`) {
		t.Errorf("no permissions must serialise as [], got %s", body.Data[0])
	}
}

func TestChangePasswordRoute(t *testing.T) {
	svc := &fakeAuthSvc{validToken: "tok"}
	app := newTestRouter(svc, false)

	_, body := do(app, http.MethodPost, "/api/v1/auth/change-password", `{"current_password":"old-password-1","new_password":"new-password-1"}`,
		map[string]string{"Cookie": "ecom_session=tok", "Origin": "http://localhost:5173"})
	if body.Code != 200 || svc.gotChange != [2]string{"old-password-1", "new-password-1"} {
		t.Fatalf("change: %+v %v", body, svc.gotChange)
	}

	if _, body = do(app, http.MethodPost, "/api/v1/auth/change-password", `{"new_password":"x"}`, nil); body.Message != "UNAUTHENTICATED" {
		t.Errorf("anonymous: %+v", body)
	}
	if _, body = do(app, http.MethodPost, "/api/v1/auth/change-password", `{"new_password":"x"}`,
		map[string]string{"Cookie": "ecom_session=tok", "Origin": "https://evil.example"}); body.Message != "FORBIDDEN_ORIGIN" {
		t.Errorf("cross-origin: %+v", body)
	}

	svc.changeErr = services.ErrWrongPassword
	w, body := do(app, http.MethodPost, "/api/v1/auth/change-password", `{"current_password":"secret-pw-123","new_password":"new-password-1"}`, map[string]string{"Cookie": "ecom_session=tok"})
	if body.Message != "WRONG_PASSWORD" || strings.Contains(w.Body.String(), "secret-pw-123") {
		t.Errorf("wrong password: %s", w.Body.String())
	}
}
