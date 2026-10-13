package models

import "time"

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
}

// SessionUser is the signed-in user of a request, as put in the gin context by
// middleware.RequireAuth.
type SessionUser struct {
	AppUser
	AuthMethod AuthMethod `db:"auth_method"`
	TokenHash  string     `db:"token_hash"`
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
}

type ProvidersResponse struct {
	Google   bool `json:"google"`
	Password bool `json:"password"`
}

type LogoutAllResponse struct {
	RevokedSessions int64 `json:"revoked_sessions"`
}
