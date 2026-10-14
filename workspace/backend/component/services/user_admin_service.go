package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"main/component/models"
	"main/component/repositories"
	"main/utils"
	"net/mail"
	"strings"
)

// UserAdminService is the user-management page: list users and roles, create
// accounts, change role/status, reset passwords. Callers must already have
// checked the permission (users.read / users.manage) in the route; the
// service enforces the rules that protect the system itself:
//   - an admin can't change their own role/status or reset their own
//     password here (use "Đổi mật khẩu" instead), so nobody locks themselves out;
//   - the last active user who can manage users can't be demoted or disabled.
type UserAdminService interface {
	ListUsers(ctx context.Context) ([]models.ManagedUser, error)
	ListRoles(ctx context.Context) ([]models.Role, error)
	CreateUser(ctx context.Context, actor *models.SessionUser, req models.CreateUserRequest) (*models.ManagedUser, error)
	UpdateUser(ctx context.Context, actor *models.SessionUser, id int64, req models.UpdateUserRequest) (*models.ManagedUser, error)
	ResetPassword(ctx context.Context, actor *models.SessionUser, id int64, password string) error
}

type userAdminStore interface {
	List(ctx context.Context) ([]models.ManagedUser, error)
	GetManaged(ctx context.Context, id int64) (*models.ManagedUser, error)
	Create(ctx context.Context, u models.AppUser, roleCode string) (int64, error)
	Update(ctx context.Context, id int64, upd repositories.UserUpdate) error
	SetPassword(ctx context.Context, userID int64, hash string) error
	CountActiveWithPermission(ctx context.Context, permission string, excludeUserID int64) (int, error)
}

type roleStore interface {
	List(ctx context.Context) ([]models.Role, error)
}

type sessionRevoker interface {
	RevokeAllForUser(ctx context.Context, userID int64) (int64, error)
}

type passwordHasher interface {
	HashNewPassword(pw string) (string, error)
}

type UserAdminServiceImpl struct {
	users    userAdminStore
	roles    roleStore
	sessions sessionRevoker
	hasher   passwordHasher
}

// NewUserAdminService takes the AuthService as hasher, so both share one password policy.
func NewUserAdminService(users userAdminStore, roles roleStore, sessions sessionRevoker, hasher passwordHasher) UserAdminService {
	return &UserAdminServiceImpl{users: users, roles: roles, sessions: sessions, hasher: hasher}
}

func (s *UserAdminServiceImpl) ListUsers(ctx context.Context) ([]models.ManagedUser, error) {
	users, err := s.users.List(ctx)
	if err != nil {
		return nil, wrapAuth(ErrInternal, err)
	}
	return users, nil
}

func (s *UserAdminServiceImpl) ListRoles(ctx context.Context) ([]models.Role, error) {
	roles, err := s.roles.List(ctx)
	if err != nil {
		return nil, wrapAuth(ErrInternal, err)
	}
	return roles, nil
}

func (s *UserAdminServiceImpl) CreateUser(ctx context.Context, actor *models.SessionUser, req models.CreateUserRequest) (*models.ManagedUser, error) {
	email := normalizeEmail(req.Email)
	if addr, err := mail.ParseAddress(email); err != nil || addr.Address != email {
		return nil, ErrInvalidRequest
	}
	u := models.AppUser{Email: email, DisplayName: strings.TrimSpace(req.DisplayName), Status: models.UserStatusActive}
	if req.Password != nil {
		hash, err := s.hasher.HashNewPassword(*req.Password)
		if err != nil {
			return nil, err
		}
		u.PasswordHash = &hash
	}
	id, err := s.users.Create(ctx, u, strings.TrimSpace(req.Role))
	switch {
	case errors.Is(err, repositories.ErrDuplicateEmail):
		return nil, ErrEmailTaken
	case errors.Is(err, repositories.ErrUnknownRole):
		return nil, ErrInvalidRole
	case err != nil:
		return nil, wrapAuth(ErrInternal, err)
	}
	utils.ShowInfoLogs("[UserAdminService] [CreateUser] by user_id=%d: user_id=%d email=%s role=%s password=%v",
		actor.ID, id, email, req.Role, req.Password != nil)
	return s.get(ctx, id)
}

func (s *UserAdminServiceImpl) UpdateUser(ctx context.Context, actor *models.SessionUser, id int64, req models.UpdateUserRequest) (*models.ManagedUser, error) {
	target, err := s.get(ctx, id)
	if err != nil {
		return nil, err
	}
	upd := repositories.UserUpdate{}
	if req.DisplayName != nil {
		name := strings.TrimSpace(*req.DisplayName)
		upd.DisplayName = &name
	}
	roleChanges := req.Role != nil && *req.Role != target.Role.Code
	statusChanges := req.Status != nil && *req.Status != target.Status
	if (roleChanges || statusChanges) && id == actor.ID {
		return nil, ErrSelfModify
	}
	if roleChanges {
		role := strings.TrimSpace(*req.Role)
		upd.RoleCode = &role
	}
	if statusChanges {
		upd.Status = req.Status
	}
	if roleChanges || (statusChanges && *req.Status != models.UserStatusActive) {
		if err := s.ensureNotLastManager(ctx, target); err != nil {
			return nil, err
		}
	}

	switch err := s.users.Update(ctx, id, upd); {
	case errors.Is(err, repositories.ErrUnknownRole):
		return nil, ErrInvalidRole
	case errors.Is(err, sql.ErrNoRows):
		return nil, ErrNotFound
	case err != nil:
		return nil, wrapAuth(ErrInternal, err)
	}
	if statusChanges && *req.Status == models.UserStatusDisabled {
		// Sessions already stop working for disabled users (GetValid checks
		// status); revoking them as well keeps the session list honest.
		if _, err := s.sessions.RevokeAllForUser(ctx, id); err != nil {
			utils.ShowErrorLogs(fmt.Errorf("[UserAdminService] [UpdateUser] revoke sessions of user_id=%d: %w", id, err))
		}
	}
	utils.ShowInfoLogs("[UserAdminService] [UpdateUser] by user_id=%d: user_id=%d role_changed=%v status_changed=%v",
		actor.ID, id, roleChanges, statusChanges)
	return s.get(ctx, id)
}

// ensureNotLastManager refuses a change that would leave no active user able
// to manage users. The new role's permissions aren't known here, so any role
// change of a manager counts as losing the permission unless another active
// manager exists.
func (s *UserAdminServiceImpl) ensureNotLastManager(ctx context.Context, target *models.ManagedUser) error {
	targetIsManager, err := s.isManager(ctx, target)
	if err != nil || !targetIsManager {
		return err
	}
	others, err := s.users.CountActiveWithPermission(ctx, models.PermUsersManage, target.ID)
	if err != nil {
		return wrapAuth(ErrInternal, err)
	}
	if others == 0 {
		return ErrLastAdmin
	}
	return nil
}

func (s *UserAdminServiceImpl) isManager(ctx context.Context, u *models.ManagedUser) (bool, error) {
	if u.Status != models.UserStatusActive {
		return false, nil
	}
	roles, err := s.roles.List(ctx)
	if err != nil {
		return false, wrapAuth(ErrInternal, err)
	}
	for _, r := range roles {
		if r.Code != u.Role.Code {
			continue
		}
		for _, p := range r.Permissions {
			if p == models.PermUsersManage {
				return true, nil
			}
		}
	}
	return false, nil
}

func (s *UserAdminServiceImpl) ResetPassword(ctx context.Context, actor *models.SessionUser, id int64, password string) error {
	if id == actor.ID {
		return ErrSelfModify
	}
	if _, err := s.get(ctx, id); err != nil {
		return err
	}
	hash, err := s.hasher.HashNewPassword(password)
	if err != nil {
		return err
	}
	if err := s.users.SetPassword(ctx, id, hash); err != nil {
		return wrapAuth(ErrInternal, err)
	}
	// Whoever knew the old password (or holds an old session) is signed out.
	if _, err := s.sessions.RevokeAllForUser(ctx, id); err != nil {
		return wrapAuth(ErrInternal, err)
	}
	utils.ShowInfoLogs("[UserAdminService] [ResetPassword] by user_id=%d: user_id=%d", actor.ID, id)
	return nil
}

func (s *UserAdminServiceImpl) get(ctx context.Context, id int64) (*models.ManagedUser, error) {
	u, err := s.users.GetManaged(ctx, id)
	if err != nil {
		return nil, wrapAuth(ErrInternal, err)
	}
	if u == nil {
		return nil, ErrNotFound
	}
	return u, nil
}
