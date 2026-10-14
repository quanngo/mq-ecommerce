package models

import (
	"time"

	"github.com/lib/pq"
)

type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusDisabled UserStatus = "disabled"
)

type AuthMethod string

const (
	AuthMethodGoogle   AuthMethod = "google"
	AuthMethodPassword AuthMethod = "password"
)

// AppUser maps public.app_user. PasswordHash and GoogleSub are never serialised.
type AppUser struct {
	ID               int64      `db:"id" json:"id"`
	Email            string     `db:"email" json:"email"`
	DisplayName      string     `db:"display_name" json:"display_name"`
	AvatarURL        *string    `db:"avatar_url" json:"avatar_url"`
	PasswordHash     *string    `db:"password_hash" json:"-"`
	GoogleSub        *string    `db:"google_sub" json:"-"`
	Status           UserStatus `db:"status" json:"status"`
	FailedLoginCount int        `db:"failed_login_count" json:"-"`
	LockedUntil      *time.Time `db:"locked_until" json:"-"`
	LastLoginAt      *time.Time `db:"last_login_at" json:"last_login_at"`
	CreatedAt        time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time  `db:"updated_at" json:"updated_at"`
	RoleID           int64      `db:"role_id" json:"-"`
}

// SessionUser is the signed-in user of a request, as put in the gin context by
// middleware.RequireAuth: the user, their role and the role's permissions.
type SessionUser struct {
	AppUser
	AuthMethod  AuthMethod     `db:"auth_method"`
	TokenHash   string         `db:"token_hash"`
	RoleCode    string         `db:"role_code"`
	RoleName    string         `db:"role_name"`
	Permissions pq.StringArray `db:"permissions" swaggertype:"array,string"`
}

// Can reports whether the user's role grants the permission.
func (u *SessionUser) Can(permission string) bool {
	if u == nil {
		return false
	}
	for _, p := range u.Permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// LoginAttempt maps public.oauth_login_attempt.
type LoginAttempt struct {
	State        string    `db:"state"`
	Nonce        string    `db:"nonce"`
	CodeVerifier string    `db:"code_verifier"`
	RedirectPath string    `db:"redirect_path"`
	RememberMe   bool      `db:"remember_me"`
	ExpiresAt    time.Time `db:"expires_at"`
}

// NewSession is a freshly created session. Token is the raw cookie value; it
// only lives in memory until it is written to Set-Cookie.
type NewSession struct {
	Token     string
	ExpiresAt time.Time
	// Persistent = "remember me": the cookie gets a Max-Age; otherwise it is a
	// browser-session cookie that disappears when the browser closes.
	Persistent bool
	User       SessionUser
}

func NewUserResponse(u SessionUser) UserResponse {
	return UserResponse{
		ID:          u.ID,
		Email:       u.Email,
		DisplayName: u.DisplayName,
		AvatarURL:   u.AvatarURL,
		AuthMethod:  u.AuthMethod,
		LastLoginAt: u.LastLoginAt,
		Role:        RoleRef{Code: u.RoleCode, Name: u.RoleName},
		Permissions: nonNil(u.Permissions),
	}
}

type PasswordLoginRequest struct {
	Email      string `json:"email" binding:"required,email,max=320"`
	Password   string `json:"password" binding:"required,max=256"`
	RememberMe bool   `json:"remember_me"`
}

type UserResponse struct {
	ID          int64      `json:"id"`
	Email       string     `json:"email"`
	DisplayName string     `json:"display_name"`
	AvatarURL   *string    `json:"avatar_url"`
	AuthMethod  AuthMethod `json:"auth_method"`
	LastLoginAt *time.Time `json:"last_login_at"`
	Role        RoleRef    `json:"role"`
	Permissions []string   `json:"permissions"`
}

type ProvidersResponse struct {
	Google   bool `json:"google"`
	Password bool `json:"password"`
}

type LogoutAllResponse struct {
	RevokedSessions int64 `json:"revoked_sessions"`
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// ---------- roles & permissions ----------

// Permission codes checked by the code. Keep in sync with migrations/004 and 005.
const (
	PermUsersRead    = "users.read"
	PermUsersManage  = "users.manage"
	PermOrdersRead   = "orders.read"
	PermOrdersWrite  = "orders.write"
	PermOrdersDetail = "orders.detail"
)

type RoleRef struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// Role maps public.role plus its permission codes.
type Role struct {
	ID          int64          `db:"id" json:"-"`
	Code        string         `db:"code" json:"code"`
	Name        string         `db:"name" json:"name"`
	Description string         `db:"description" json:"description"`
	IsSystem    bool           `db:"is_system" json:"is_system"`
	Permissions pq.StringArray `db:"permissions" json:"permissions" swaggertype:"array,string"`
}

// ---------- user management ----------

// ManagedUser is a row of the user-management list.
type ManagedUser struct {
	ID           int64      `db:"id" json:"id"`
	Email        string     `db:"email" json:"email"`
	DisplayName  string     `db:"display_name" json:"display_name"`
	AvatarURL    *string    `db:"avatar_url" json:"avatar_url"`
	Status       UserStatus `db:"status" json:"status"`
	RoleCode     string     `db:"role_code" json:"-"`
	RoleName     string     `db:"role_name" json:"-"`
	Role         RoleRef    `db:"-" json:"role"`
	HasPassword  bool       `db:"has_password" json:"has_password"`
	GoogleLinked bool       `db:"google_linked" json:"google_linked"`
	LockedUntil  *time.Time `db:"locked_until" json:"locked_until"`
	LastLoginAt  *time.Time `db:"last_login_at" json:"last_login_at"`
	CreatedAt    time.Time  `db:"created_at" json:"created_at"`
}

type UsersListResponse struct {
	Users []ManagedUser `json:"users"`
}

type RolesListResponse struct {
	Roles []Role `json:"roles"`
}

type CreateUserRequest struct {
	Email       string  `json:"email" binding:"required,email,max=320"`
	DisplayName string  `json:"display_name" binding:"max=255"`
	Role        string  `json:"role" binding:"required,max=64"`
	Password    *string `json:"password" binding:"omitempty,max=256"` // omitted = Google-only account
}

// UpdateUserRequest: only the fields that are present are changed.
type UpdateUserRequest struct {
	DisplayName *string     `json:"display_name" binding:"omitempty,max=255"`
	Role        *string     `json:"role" binding:"omitempty,max=64"`
	Status      *UserStatus `json:"status" binding:"omitempty,oneof=active disabled"`
}

type ResetPasswordRequest struct {
	Password string `json:"password" binding:"required,max=256"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"max=256"`
	NewPassword     string `json:"new_password" binding:"required,max=256"`
}
