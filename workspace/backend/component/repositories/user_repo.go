package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"main/component/models"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

var (
	ErrDuplicateEmail = errors.New("app_user email already exists")
	ErrUnknownRole    = errors.New("unknown role")
)

// appUserColumns lists app_user's columns explicitly (instead of SELECT *),
// so adding a column to the table never breaks scanning into models.AppUser,
// plus the user's settings from public.user_config (1-1; a user without a row
// reads as the defaults, see migrations/006).
func appUserColumns(alias string) string {
	cols := []string{"id", "email", "display_name", "avatar_url", "password_hash", "google_sub", "status",
		"failed_login_count", "locked_until", "last_login_at", "created_at", "updated_at", "role_id"}
	for i, c := range cols {
		cols[i] = alias + "." + c
	}
	cols = append(cols, `COALESCE((SELECT uc.locale FROM public.user_config uc WHERE uc.user_id = `+alias+`.id), 'vi') AS locale`)
	return strings.Join(cols, ", ")
}

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
	return r.getOne(ctx, `SELECT `+appUserColumns("u")+` FROM public.app_user u WHERE u.email = $1`, email)
}

// GetByGoogleSub returns (nil, nil) when no user is linked to this Google account.
func (r *UserRepo) GetByGoogleSub(ctx context.Context, sub string) (*models.AppUser, error) {
	return r.getOne(ctx, `SELECT `+appUserColumns("u")+` FROM public.app_user u WHERE u.google_sub = $1`, sub)
}

// GetByID returns (nil, nil) when the user does not exist.
func (r *UserRepo) GetByID(ctx context.Context, id int64) (*models.AppUser, error) {
	return r.getOne(ctx, `SELECT `+appUserColumns("u")+` FROM public.app_user u WHERE u.id = $1`, id)
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

// Create inserts a user and returns its id. roleCode "" means the least
// privileged role (viewer). A taken email returns ErrDuplicateEmail, an
// unknown role ErrUnknownRole.
func (r *UserRepo) Create(ctx context.Context, u models.AppUser, roleCode string) (int64, error) {
	if roleCode == "" {
		roleCode = "viewer"
	}
	if u.Status == "" {
		u.Status = models.UserStatusActive
	}
	strSql := `
		WITH new_user AS (
		    INSERT INTO public.app_user (email, display_name, avatar_url, password_hash, google_sub, status, role_id)
		    SELECT $1, $2, $3, $4, $5, $6, r.id FROM public.role r WHERE r.code = $7
		    RETURNING id
		), new_config AS (
		    INSERT INTO public.user_config (user_id, locale)
		    SELECT id, COALESCE(NULLIF($8, ''), 'vi') FROM new_user
		)
		SELECT id FROM new_user
	`
	var id int64
	err := r.db.QueryRowContext(ctx, strSql, u.Email, u.DisplayName, u.AvatarURL, u.PasswordHash, u.GoogleSub, u.Status, roleCode, u.Locale).Scan(&id)
	var pqErr *pq.Error
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return 0, ErrUnknownRole
	case errors.As(err, &pqErr) && pqErr.Code == "23505" && pqErr.Constraint == "ux_app_user_email":
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
// guesses cannot skip the lock. A lock that has already expired starts a new
// count at 1, so one wrong password after a lockout does not re-lock at once.
func (r *UserRepo) RecordLoginFailure(ctx context.Context, userID int64, maxFailed int, lockFor time.Duration) (int, *time.Time, error) {
	strSql := `
		WITH next AS (
		    SELECT id,
		           CASE WHEN locked_until IS NOT NULL AND locked_until <= NOW() THEN 1
		                ELSE failed_login_count + 1 END AS cnt
		    FROM public.app_user WHERE id = $1
		    FOR UPDATE
		)
		UPDATE public.app_user u
		SET failed_login_count = next.cnt,
		    locked_until = CASE WHEN next.cnt >= $2 THEN NOW() + make_interval(secs => $3)
		                        WHEN u.locked_until <= NOW() THEN NULL
		                        ELSE u.locked_until END,
		    updated_at = NOW()
		FROM next
		WHERE u.id = next.id
		RETURNING u.failed_login_count, u.locked_until
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

// SetLocale stores the user's preferred UI language in public.user_config,
// creating the row if the user has none yet. The value is already validated
// against i18n.Supported; ck_user_config_locale is the DB-side guard.
func (r *UserRepo) SetLocale(ctx context.Context, userID int64, locale string) error {
	strSql := `
		INSERT INTO public.user_config (user_id, locale) VALUES ($1, $2)
		ON CONFLICT (user_id) DO UPDATE SET locale = EXCLUDED.locale, updated_at = NOW()
	`
	_, err := r.db.ExecContext(ctx, strSql, userID, locale)
	return err
}

// SetPassword stores a new bcrypt hash and clears any lockout.
func (r *UserRepo) SetPassword(ctx context.Context, userID int64, hash string) error {
	strSql := `
		UPDATE public.app_user
		SET password_hash = $2, failed_login_count = 0, locked_until = NULL, updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, strSql, userID, hash)
	return err
}

const managedUserSelect = `
	SELECT u.id, u.email, u.display_name, u.avatar_url, u.status,
	       r.code AS role_code, r.name AS role_name,
	       (u.password_hash IS NOT NULL) AS has_password,
	       (u.google_sub IS NOT NULL) AS google_linked,
	       CASE WHEN u.locked_until > NOW() THEN u.locked_until END AS locked_until,
	       u.last_login_at, u.created_at
	FROM public.app_user u
	JOIN public.role r ON r.id = u.role_id
`

// List returns every user for the user-management page, newest first.
func (r *UserRepo) List(ctx context.Context) ([]models.ManagedUser, error) {
	users := []models.ManagedUser{}
	if err := r.db.SelectContext(ctx, &users, managedUserSelect+` ORDER BY u.created_at DESC, u.id DESC`); err != nil {
		return nil, err
	}
	for i := range users {
		users[i].Role = models.RoleRef{Code: users[i].RoleCode, Name: users[i].RoleName}
	}
	return users, nil
}

// GetManaged returns one user-management row, or (nil, nil).
func (r *UserRepo) GetManaged(ctx context.Context, id int64) (*models.ManagedUser, error) {
	var u models.ManagedUser
	err := r.db.GetContext(ctx, &u, managedUserSelect+` WHERE u.id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	u.Role = models.RoleRef{Code: u.RoleCode, Name: u.RoleName}
	return &u, nil
}

// UserUpdate holds the fields to change; nil means "leave as is".
type UserUpdate struct {
	DisplayName *string
	RoleCode    *string
	Status      *models.UserStatus
}

// Update changes the given fields. Returns ErrUnknownRole for a bad role and
// sql.ErrNoRows when the user does not exist.
func (r *UserRepo) Update(ctx context.Context, id int64, upd UserUpdate) error {
	sets := []string{"updated_at = NOW()"}
	args := []interface{}{id}
	if upd.DisplayName != nil {
		args = append(args, *upd.DisplayName)
		sets = append(sets, fmt.Sprintf("display_name = $%d", len(args)))
	}
	if upd.Status != nil {
		args = append(args, *upd.Status)
		sets = append(sets, fmt.Sprintf("status = $%d", len(args)))
	}
	if upd.RoleCode != nil {
		args = append(args, *upd.RoleCode)
		sets = append(sets, fmt.Sprintf("role_id = (SELECT id FROM public.role WHERE code = $%d)", len(args)))
	}
	result, err := r.db.ExecContext(ctx, `UPDATE public.app_user SET `+strings.Join(sets, ", ")+` WHERE id = $1`, args...)
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "23502" { // role subquery returned NULL
		return ErrUnknownRole
	}
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// CountActiveWithPermission counts active users whose role grants the
// permission, optionally ignoring one user. Used to never remove the last
// account that can manage users.
func (r *UserRepo) CountActiveWithPermission(ctx context.Context, permission string, excludeUserID int64) (int, error) {
	strSql := `
		SELECT COUNT(*)
		FROM public.app_user u
		JOIN public.role_permission rp ON rp.role_id = u.role_id AND rp.permission_code = $1
		WHERE u.status = 'active' AND u.id <> $2
	`
	var n int
	err := r.db.GetContext(ctx, &n, strSql, permission, excludeUserID)
	return n, err
}
