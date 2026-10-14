package services

import (
	"context"
	"database/sql"
	"errors"
	"main/component/models"
	"main/component/repositories"
	"main/utils"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"golang.org/x/crypto/bcrypt"
)

// ---------- fake user-admin store on top of fakeUsers ----------

type fakeAdminStore struct {
	*fakeUsers
	listErr error
}

func (f *fakeAdminStore) managed(u *models.AppUser) models.ManagedUser {
	role := f.roles[u.ID]
	return models.ManagedUser{
		ID: u.ID, Email: u.Email, DisplayName: u.DisplayName, Status: u.Status,
		Role: models.RoleRef{Code: role, Name: role}, HasPassword: u.PasswordHash != nil, GoogleLinked: u.GoogleSub != nil,
	}
}

func (f *fakeAdminStore) List(context.Context) ([]models.ManagedUser, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	out := []models.ManagedUser{}
	for _, u := range f.byID {
		out = append(out, f.managed(u))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (f *fakeAdminStore) GetManaged(_ context.Context, id int64) (*models.ManagedUser, error) {
	u, ok := f.byID[id]
	if !ok {
		return nil, nil
	}
	m := f.managed(u)
	return &m, nil
}

func (f *fakeAdminStore) Update(_ context.Context, id int64, upd repositories.UserUpdate) error {
	u, ok := f.byID[id]
	if !ok {
		return sql.ErrNoRows
	}
	if upd.RoleCode != nil {
		if _, ok := fakeRolePermissions[*upd.RoleCode]; !ok {
			return repositories.ErrUnknownRole
		}
		f.roles[id] = *upd.RoleCode
	}
	if upd.DisplayName != nil {
		u.DisplayName = *upd.DisplayName
	}
	if upd.Status != nil {
		u.Status = *upd.Status
	}
	return nil
}

func (f *fakeAdminStore) CountActiveWithPermission(_ context.Context, perm string, exclude int64) (int, error) {
	n := 0
	for id, u := range f.byID {
		if id == exclude || u.Status != models.UserStatusActive {
			continue
		}
		for _, p := range fakeRolePermissions[f.roles[id]] {
			if p == perm {
				n++
			}
		}
	}
	return n, nil
}

type fakeRoles struct{}

func (fakeRoles) List(context.Context) ([]models.Role, error) {
	var out []models.Role
	for _, code := range []string{"admin", "operator", "viewer"} {
		out = append(out, models.Role{Code: code, Name: code, IsSystem: true, Permissions: fakeRolePermissions[code]})
	}
	return out, nil
}

type adminHarness struct {
	*harness
	store *fakeAdminStore
	admin UserAdminService
}

func newAdminHarness(t *testing.T) *adminHarness {
	t.Helper()
	h := newHarness(t, utils.AuthConfig{})
	store := &fakeAdminStore{fakeUsers: h.users}
	return &adminHarness{harness: h, store: store, admin: NewUserAdminService(store, fakeRoles{}, h.sessions, h.svc)}
}

// addUser creates a user with a role and returns it as a session user.
func (a *adminHarness) addUser(t *testing.T, email, role string) *models.SessionUser {
	t.Helper()
	pw := hashPw(t, "correct-horse-1")
	id, err := a.users.Create(context.Background(), models.AppUser{Email: email, PasswordHash: pw, Status: models.UserStatusActive}, role)
	if err != nil {
		t.Fatal(err)
	}
	return &models.SessionUser{AppUser: *a.users.byID[id], RoleCode: role, Permissions: fakeRolePermissions[role]}
}

func strPtr(s string) *string                          { return &s }
func statusPtr(s models.UserStatus) *models.UserStatus { return &s }

// ---------- tests ----------

func TestUserAdmin_CreateUser(t *testing.T) {
	a := newAdminHarness(t)
	admin := a.addUser(t, "admin@mq.com", "admin")
	ctx := context.Background()

	u, err := a.admin.CreateUser(ctx, admin, models.CreateUserRequest{Email: " New.User@MQ.com ", DisplayName: " New ", Role: "operator", Password: strPtr("long-enough-pw")})
	if err != nil {
		t.Fatal(err)
	}
	if u.Email != "new.user@mq.com" || u.DisplayName != "New" || u.Role.Code != "operator" || !u.HasPassword {
		t.Errorf("created = %+v", u)
	}
	stored := a.users.byID[u.ID]
	if bcrypt.CompareHashAndPassword([]byte(*stored.PasswordHash), []byte("long-enough-pw")) != nil {
		t.Error("password not stored as a bcrypt hash of the input")
	}

	g, err := a.admin.CreateUser(ctx, admin, models.CreateUserRequest{Email: "g@mq.com", Role: "viewer"})
	if err != nil || g.HasPassword {
		t.Errorf("google-only user: %+v %v", g, err)
	}

	for name, tc := range map[string]struct {
		req  models.CreateUserRequest
		want *AuthError
	}{
		"duplicate":     {models.CreateUserRequest{Email: "new.user@mq.com", Role: "viewer"}, ErrEmailTaken},
		"unknown role":  {models.CreateUserRequest{Email: "x@mq.com", Role: "superuser"}, ErrInvalidRole},
		"weak password": {models.CreateUserRequest{Email: "y@mq.com", Role: "viewer", Password: strPtr("123456")}, ErrWeakPassword},
		"bad email":     {models.CreateUserRequest{Email: "Name <z@mq.com>", Role: "viewer"}, ErrInvalidRequest},
	} {
		if _, err := a.admin.CreateUser(ctx, admin, tc.req); !errors.Is(err, tc.want) {
			t.Errorf("%s: err = %v, want %s", name, err, tc.want.Code)
		}
	}
}

func TestUserAdmin_UpdateUser(t *testing.T) {
	a := newAdminHarness(t)
	admin := a.addUser(t, "admin@mq.com", "admin")
	op := a.addUser(t, "op@mq.com", "operator")
	ctx := context.Background()

	u, err := a.admin.UpdateUser(ctx, admin, op.ID, models.UpdateUserRequest{Role: strPtr("viewer"), DisplayName: strPtr(" Ops ")})
	if err != nil || u.Role.Code != "viewer" || u.DisplayName != "Ops" {
		t.Fatalf("update: %+v %v", u, err)
	}

	// Disabling signs the user out everywhere.
	sess, _ := a.svc.LoginWithPassword(ctx, "op@mq.com", "correct-horse-1", ClientMeta{})
	if _, err := a.admin.UpdateUser(ctx, admin, op.ID, models.UpdateUserRequest{Status: statusPtr(models.UserStatusDisabled)}); err != nil {
		t.Fatal(err)
	}
	if !a.sessions.rows[hashToken(sess.Token)].revoked {
		t.Error("disabled user's session not revoked")
	}

	if _, err := a.admin.UpdateUser(ctx, admin, op.ID, models.UpdateUserRequest{Role: strPtr("superuser")}); !errors.Is(err, ErrInvalidRole) {
		t.Errorf("unknown role: %v", err)
	}
	if _, err := a.admin.UpdateUser(ctx, admin, 9999, models.UpdateUserRequest{DisplayName: strPtr("x")}); !errors.Is(err, ErrNotFound) {
		t.Errorf("missing user: %v", err)
	}
}

func TestUserAdmin_CannotModifySelf(t *testing.T) {
	a := newAdminHarness(t)
	admin := a.addUser(t, "admin@mq.com", "admin")
	a.addUser(t, "admin2@mq.com", "admin") // another admin exists, so this is about "self", not "last"
	ctx := context.Background()

	if _, err := a.admin.UpdateUser(ctx, admin, admin.ID, models.UpdateUserRequest{Role: strPtr("viewer")}); !errors.Is(err, ErrSelfModify) {
		t.Errorf("self demote: %v", err)
	}
	if _, err := a.admin.UpdateUser(ctx, admin, admin.ID, models.UpdateUserRequest{Status: statusPtr(models.UserStatusDisabled)}); !errors.Is(err, ErrSelfModify) {
		t.Errorf("self disable: %v", err)
	}
	if err := a.admin.ResetPassword(ctx, admin, admin.ID, "long-enough-pw"); !errors.Is(err, ErrSelfModify) {
		t.Errorf("self reset: %v", err)
	}
	// Renaming yourself, or "changing" to the same role, is fine.
	if _, err := a.admin.UpdateUser(ctx, admin, admin.ID, models.UpdateUserRequest{DisplayName: strPtr("Me"), Role: strPtr("admin")}); err != nil {
		t.Errorf("self rename: %v", err)
	}
}

func TestUserAdmin_LastAdminIsProtected(t *testing.T) {
	a := newAdminHarness(t)
	a1 := a.addUser(t, "a1@mq.com", "admin")
	a2 := a.addUser(t, "a2@mq.com", "admin")
	ctx := context.Background()

	// a1 demotes a2: fine, a1 still manages users.
	if _, err := a.admin.UpdateUser(ctx, a1, a2.ID, models.UpdateUserRequest{Role: strPtr("viewer")}); err != nil {
		t.Fatalf("demote second admin: %v", err)
	}
	// Re-promote a2, then have a2 try to disable a1 after a1 lost... simulate: only a1 remains as manager.
	a.users.roles[a2.ID] = "operator"
	a2.Permissions = fakeRolePermissions["admin"] // a2's stale session still says admin
	if _, err := a.admin.UpdateUser(ctx, a2, a1.ID, models.UpdateUserRequest{Status: statusPtr(models.UserStatusDisabled)}); !errors.Is(err, ErrLastAdmin) {
		t.Errorf("disable last admin: %v", err)
	}
	if _, err := a.admin.UpdateUser(ctx, a2, a1.ID, models.UpdateUserRequest{Role: strPtr("viewer")}); !errors.Is(err, ErrLastAdmin) {
		t.Errorf("demote last admin: %v", err)
	}
	if a.users.roles[a1.ID] != "admin" || a.users.byID[a1.ID].Status != models.UserStatusActive {
		t.Error("last admin was changed despite the error")
	}
}

func TestUserAdmin_ResetPassword(t *testing.T) {
	a := newAdminHarness(t)
	admin := a.addUser(t, "admin@mq.com", "admin")
	op := a.addUser(t, "op@mq.com", "operator")
	ctx := context.Background()

	sess, _ := a.svc.LoginWithPassword(ctx, "op@mq.com", "correct-horse-1", ClientMeta{})
	a.users.byID[op.ID].FailedLoginCount = 3

	if err := a.admin.ResetPassword(ctx, admin, op.ID, "123456"); !errors.Is(err, ErrWeakPassword) {
		t.Errorf("weak password: %v", err)
	}
	if err := a.admin.ResetPassword(ctx, admin, op.ID, "brand-new-password"); err != nil {
		t.Fatal(err)
	}
	if !a.sessions.rows[hashToken(sess.Token)].revoked {
		t.Error("old sessions survive a password reset")
	}
	if _, err := a.svc.LoginWithPassword(ctx, "op@mq.com", "brand-new-password", ClientMeta{}); err != nil {
		t.Errorf("login with the new password: %v", err)
	}
	if err := a.admin.ResetPassword(ctx, admin, 9999, "brand-new-password"); !errors.Is(err, ErrNotFound) {
		t.Errorf("missing user: %v", err)
	}
}

func TestUserAdmin_ListErrors(t *testing.T) {
	a := newAdminHarness(t)
	a.store.listErr = errDB
	_, err := a.admin.ListUsers(context.Background())
	wantInternal(t, "list", err)
}

func TestChangePassword(t *testing.T) {
	h := newHarness(t, utils.AuthConfig{}, pwUser(t))
	ctx := context.Background()
	here, _ := h.svc.LoginWithPassword(ctx, "an.nguyen@mq.com", "correct-horse-1", ClientMeta{})
	other, _ := h.svc.LoginWithPassword(ctx, "an.nguyen@mq.com", "correct-horse-1", ClientMeta{})
	actor := &here.User

	if err := h.svc.ChangePassword(ctx, actor, "wrong-password", "brand-new-password"); !errors.Is(err, ErrWrongPassword) {
		t.Errorf("wrong current: %v", err)
	}
	if err := h.svc.ChangePassword(ctx, actor, "correct-horse-1", "short"); !errors.Is(err, ErrWeakPassword) {
		t.Errorf("weak new: %v", err)
	}
	if err := h.svc.ChangePassword(ctx, actor, "correct-horse-1", "correct-horse-1"); !errors.Is(err, ErrWeakPassword) {
		t.Errorf("same as current: %v", err)
	}
	if err := h.svc.ChangePassword(ctx, actor, "correct-horse-1", "brand-new-password"); err != nil {
		t.Fatal(err)
	}
	if h.sessions.rows[hashToken(here.Token)].revoked {
		t.Error("the session that changed the password was signed out")
	}
	if !h.sessions.rows[hashToken(other.Token)].revoked {
		t.Error("other sessions survive a password change")
	}
	if _, err := h.svc.LoginWithPassword(ctx, "an.nguyen@mq.com", "brand-new-password", ClientMeta{}); err != nil {
		t.Errorf("login with new password: %v", err)
	}
	if _, err := h.svc.LoginWithPassword(ctx, "an.nguyen@mq.com", "correct-horse-1", ClientMeta{}); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("old password still works: %v", err)
	}
}

func TestChangePassword_GoogleOnlyCanSetFirstPassword(t *testing.T) {
	h := newHarness(t, mqConfig(), activeUser(1, "an.nguyen@mq.com"))
	sess, _, err := h.googleLogin(t)
	if err != nil {
		t.Fatal(err)
	}
	if err := h.svc.ChangePassword(context.Background(), &sess.User, "", "first-password-1"); err != nil {
		t.Fatalf("set first password: %v", err)
	}
	if _, err := h.svc.LoginWithPassword(context.Background(), "an.nguyen@mq.com", "first-password-1", ClientMeta{}); err != nil {
		t.Errorf("password login after setting it: %v", err)
	}
}

func TestMinPasswordLengthIsConfigurable(t *testing.T) {
	h := newHarness(t, utils.AuthConfig{MinPasswordLength: 6})
	if _, err := h.svc.HashNewPassword("123456"); err != nil {
		t.Errorf("6 chars with min 6: %v", err)
	}
	if _, err := h.svc.HashNewPassword("12345"); !errors.Is(err, ErrWeakPassword) {
		t.Errorf("5 chars with min 6: %v", err)
	}
	if _, err := h.svc.HashNewPassword(strings.Repeat("x", 73)); !errors.Is(err, ErrWeakPassword) {
		t.Error("73 bytes accepted")
	}
}

func TestLoginReturnsRoleAndPermissions(t *testing.T) {
	h := newHarness(t, utils.AuthConfig{}, pwUser(t))
	h.users.roles[1] = "admin"
	sess, err := h.svc.LoginWithPassword(context.Background(), "an.nguyen@mq.com", "correct-horse-1", ClientMeta{})
	if err != nil {
		t.Fatal(err)
	}
	if sess.User.RoleCode != "admin" || !sess.User.Can(models.PermUsersManage) {
		t.Errorf("session user = role %q perms %v", sess.User.RoleCode, sess.User.Permissions)
	}
	if resp := models.NewUserResponse(sess.User); resp.Role.Code != "admin" || len(resp.Permissions) != 4 {
		t.Errorf("response = %+v", resp)
	}
}

func TestAutoProvisionUsesDefaultRole(t *testing.T) {
	cfg := mqConfig()
	cfg.Google.AutoProvision = true
	cfg.DefaultRole = "operator"
	h := newHarness(t, cfg)
	sess, _, err := h.googleLogin(t)
	if err != nil {
		t.Fatal(err)
	}
	if sess.User.RoleCode != "operator" {
		t.Errorf("auto-provisioned role = %q", sess.User.RoleCode)
	}
}

func TestGoogleLoginRefreshesStoredAvatar(t *testing.T) {
	h := newHarness(t, mqConfig(), activeUser(1, "an.nguyen@mq.com"))
	if _, _, err := h.googleLogin(t); err != nil {
		t.Fatal(err)
	}
	h.google.claims.Picture = "https://img/new.png"
	sess, _, err := h.googleLogin(t) // found by sub this time
	if err != nil {
		t.Fatal(err)
	}
	if got := h.users.byID[1].AvatarURL; got == nil || *got != "https://img/new.png" {
		t.Errorf("stored avatar = %v", got)
	}
	if sess.User.AvatarURL == nil || *sess.User.AvatarURL != "https://img/new.png" {
		t.Error("login response has a stale avatar")
	}
}

func TestLockoutRestartsCountAfterExpiry(t *testing.T) {
	u := pwUser(t)
	h := newHarness(t, utils.AuthConfig{}, u)
	clock := time.Now()
	h.svc.now = func() time.Time { return clock }
	h.users.now = func() time.Time { return clock }
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		h.svc.LoginWithPassword(ctx, u.Email, "wrong-wrong-1", ClientMeta{})
	}
	if h.users.byID[1].LockedUntil == nil {
		t.Fatal("not locked after 5 failures")
	}
	clock = clock.Add(16 * time.Minute)
	if _, err := h.svc.LoginWithPassword(ctx, u.Email, "wrong-wrong-1", ClientMeta{}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("first failure after the lock expired: %v, want INVALID_CREDENTIALS (not an immediate re-lock)", err)
	}
	if h.users.byID[1].FailedLoginCount != 1 {
		t.Errorf("count after expiry = %d, want 1", h.users.byID[1].FailedLoginCount)
	}
}

func TestWarnIfDefaultAdminPassword(t *testing.T) {
	hook := logtest.NewGlobal()
	defer hook.Reset()
	logrus.SetLevel(logrus.DebugLevel)

	h := newHarness(t, utils.AuthConfig{})
	hash, _ := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.MinCost)
	hs := string(hash)
	h.users.Create(context.Background(), models.AppUser{Email: DefaultAdminEmail, PasswordHash: &hs, Status: models.UserStatusActive}, "admin")

	h.svc.WarnIfDefaultAdminPassword(context.Background())
	warned := false
	for _, e := range hook.AllEntries() {
		if strings.Contains(e.Message, "default password") {
			warned = true
		}
		if strings.Contains(e.Message, "123456") {
			t.Error("the warning prints the password")
		}
	}
	if !warned {
		t.Error("no warning while the default admin password is in use")
	}

	hook.Reset()
	for _, u := range h.users.byID {
		other, _ := bcrypt.GenerateFromPassword([]byte("a-changed-password"), bcrypt.MinCost)
		o := string(other)
		u.PasswordHash = &o
	}
	h.svc.WarnIfDefaultAdminPassword(context.Background())
	for _, e := range hook.AllEntries() {
		if strings.Contains(e.Message, "default password") {
			t.Error("warned although the password was changed")
		}
	}
}
