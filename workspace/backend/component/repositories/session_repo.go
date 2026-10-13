package repositories

import (
	"context"
	"database/sql"
	"errors"
	"main/component/models"
	"time"

	"github.com/jmoiron/sqlx"
)

// SessionRepo reads and writes public.user_session. Only the sha256 of the
// cookie token is ever passed in.
type SessionRepo struct {
	db *sqlx.DB
}

func NewSessionRepo(db *sqlx.DB) *SessionRepo {
	return &SessionRepo{
		db: db,
	}
}

func (r *SessionRepo) Create(ctx context.Context, tokenHash string, userID int64, method models.AuthMethod, expiresAt time.Time, userAgent, ip string) error {
	strSql := `
		INSERT INTO public.user_session (token_hash, user_id, auth_method, expires_at, user_agent, ip)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.ExecContext(ctx, strSql, tokenHash, userID, method, expiresAt, userAgent, ip)
	return err
}

// GetValid returns (nil, nil) when the session is missing, expired, revoked,
// or its user is disabled, so disabling a user logs them out on the next request.
func (r *SessionRepo) GetValid(ctx context.Context, tokenHash string) (*models.SessionUser, error) {
	strSql := `
		SELECT u.*, s.auth_method, s.token_hash
		FROM public.user_session s
		JOIN public.app_user u ON u.id = s.user_id
		WHERE s.token_hash = $1
		  AND s.revoked_at IS NULL
		  AND s.expires_at > NOW()
		  AND u.status = 'active'
	`
	var su models.SessionUser
	err := r.db.GetContext(ctx, &su, strSql, tokenHash)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &su, nil
}

func (r *SessionRepo) Revoke(ctx context.Context, tokenHash string) error {
	strSql := `UPDATE public.user_session SET revoked_at = NOW() WHERE token_hash = $1 AND revoked_at IS NULL`
	_, err := r.db.ExecContext(ctx, strSql, tokenHash)
	return err
}

// RevokeAllForUser revokes every open session of a user ("sign out everywhere").
func (r *SessionRepo) RevokeAllForUser(ctx context.Context, userID int64) (int64, error) {
	strSql := `UPDATE public.user_session SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`
	result, err := r.db.ExecContext(ctx, strSql, userID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// DeleteStale removes sessions that expired or were revoked more than grace ago.
func (r *SessionRepo) DeleteStale(ctx context.Context, grace time.Duration) (int64, error) {
	strSql := `
		DELETE FROM public.user_session
		WHERE expires_at < NOW() - make_interval(secs => $1)
		   OR revoked_at < NOW() - make_interval(secs => $1)
	`
	result, err := r.db.ExecContext(ctx, strSql, grace.Seconds())
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
