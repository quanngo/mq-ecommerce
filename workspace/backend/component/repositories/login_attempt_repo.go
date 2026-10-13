package repositories

import (
	"context"
	"database/sql"
	"errors"
	"main/component/models"

	"github.com/jmoiron/sqlx"
)

// LoginAttemptRepo stores the state/nonce/PKCE verifier of a started Google
// login in public.oauth_login_attempt.
type LoginAttemptRepo struct {
	db *sqlx.DB
}

func NewLoginAttemptRepo(db *sqlx.DB) *LoginAttemptRepo {
	return &LoginAttemptRepo{
		db: db,
	}
}

func (r *LoginAttemptRepo) Create(ctx context.Context, a models.LoginAttempt) error {
	strSql := `
		INSERT INTO public.oauth_login_attempt (state, nonce, code_verifier, redirect_path, remember_me, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.ExecContext(ctx, strSql, a.State, a.Nonce, a.CodeVerifier, a.RedirectPath, a.RememberMe, a.ExpiresAt)
	return err
}

// Consume deletes and returns the attempt in one statement, so a state can be
// used once even under concurrent callbacks. (nil, nil) = unknown, expired or
// already used.
func (r *LoginAttemptRepo) Consume(ctx context.Context, state string) (*models.LoginAttempt, error) {
	strSql := `
		DELETE FROM public.oauth_login_attempt
		WHERE state = $1 AND expires_at > NOW()
		RETURNING state, nonce, code_verifier, redirect_path, remember_me, expires_at
	`
	var a models.LoginAttempt
	err := r.db.GetContext(ctx, &a, strSql, state)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *LoginAttemptRepo) DeleteExpired(ctx context.Context) (int64, error) {
	result, err := r.db.ExecContext(ctx, `DELETE FROM public.oauth_login_attempt WHERE expires_at < NOW()`)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
