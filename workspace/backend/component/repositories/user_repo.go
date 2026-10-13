package repositories

import (
	"context"
	"database/sql"
	"errors"
	"main/component/models"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

var ErrDuplicateEmail = errors.New("app_user email already exists")

// UserRepo reads and writes public.app_user. Callers pass emails already
// lower-cased (ck_app_user_email_lc rejects anything else).
type UserRepo struct {
	db *sqlx.DB
}

func NewUserRepo(db *sqlx.DB) *UserRepo {
	return &UserRepo{
		db: db,
	}
}

// GetByEmail returns (nil, nil) when no user has this email.
func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*models.AppUser, error) {
	return r.getOne(ctx, `SELECT * FROM public.app_user WHERE email = $1`, email)
}

// GetByGoogleSub returns (nil, nil) when no user is linked to this Google account.
func (r *UserRepo) GetByGoogleSub(ctx context.Context, sub string) (*models.AppUser, error) {
	return r.getOne(ctx, `SELECT * FROM public.app_user WHERE google_sub = $1`, sub)
}

func (r *UserRepo) getOne(ctx context.Context, query string, arg interface{}) (*models.AppUser, error) {
	var u models.AppUser
	err := r.db.GetContext(ctx, &u, query, arg)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// Create inserts a user and returns its id. A taken email returns ErrDuplicateEmail.
func (r *UserRepo) Create(ctx context.Context, u models.AppUser) (int64, error) {
	strSql := `
		INSERT INTO public.app_user (email, display_name, avatar_url, password_hash, google_sub, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`
	if u.Status == "" {
		u.Status = models.UserStatusActive
	}
	var id int64
	err := r.db.QueryRowContext(ctx, strSql, u.Email, u.DisplayName, u.AvatarURL, u.PasswordHash, u.GoogleSub, u.Status).Scan(&id)
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "23505" && pqErr.Constraint == "ux_app_user_email" {
		return 0, ErrDuplicateEmail
	}
	return id, err
}

// LinkGoogle pins google_sub (only while it is still NULL or already equal)
// and refreshes the avatar; display_name is only filled when empty.
// Returns false when the user is linked to a different Google account.
func (r *UserRepo) LinkGoogle(ctx context.Context, userID int64, sub, displayName string, avatarURL *string) (bool, error) {
	strSql := `
		UPDATE public.app_user
		SET google_sub   = $2,
		    display_name = CASE WHEN display_name = '' THEN $3 ELSE display_name END,
		    avatar_url   = $4,
		    updated_at   = NOW()
		WHERE id = $1 AND (google_sub IS NULL OR google_sub = $2)
	`
	result, err := r.db.ExecContext(ctx, strSql, userID, sub, displayName, avatarURL)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n == 1, err
}

// RecordLoginFailure bumps the consecutive-failure counter and, once it
// reaches maxFailed, locks password login for lockFor. Atomic, so concurrent
// guesses cannot skip the lock.
func (r *UserRepo) RecordLoginFailure(ctx context.Context, userID int64, maxFailed int, lockFor time.Duration) (int, *time.Time, error) {
	strSql := `
		UPDATE public.app_user
		SET failed_login_count = failed_login_count + 1,
		    locked_until = CASE WHEN failed_login_count + 1 >= $2
		                        THEN NOW() + make_interval(secs => $3) ELSE locked_until END,
		    updated_at = NOW()
		WHERE id = $1
		RETURNING failed_login_count, locked_until
	`
	var count int
	var lockedUntil *time.Time
	err := r.db.QueryRowContext(ctx, strSql, userID, maxFailed, lockFor.Seconds()).Scan(&count, &lockedUntil)
	return count, lockedUntil, err
}

// MarkLoginSuccess clears the failure counter and lock and stamps last_login_at.
func (r *UserRepo) MarkLoginSuccess(ctx context.Context, userID int64) (time.Time, error) {
	strSql := `
		UPDATE public.app_user
		SET failed_login_count = 0, locked_until = NULL, last_login_at = NOW(), updated_at = NOW()
		WHERE id = $1
		RETURNING last_login_at
	`
	var at time.Time
	err := r.db.QueryRowContext(ctx, strSql, userID).Scan(&at)
	return at, err
}
