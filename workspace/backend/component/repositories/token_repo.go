package repositories

import (
	"context"
	"database/sql"
	"main/component/models"
	"main/utils"
	"time"

	"github.com/jmoiron/sqlx"
)

type TokenRepo struct {
	db *sqlx.DB
}

func NewTokenRepo(db *sqlx.DB) *TokenRepo {
	return &TokenRepo{
		db: db,
	}
}

func (r *TokenRepo) UpdateAccessToken(ctx context.Context, marketPlace utils.Plaform, accessToken, refreshToken, shopId string, timeExpire time.Time) (sql.Result, error) {
	strSql := `
		UPDATE public.market_token
		SET access_token = $1, refresh_token = $2, update_at = NOW(),expire_time = $3
		WHERE market_place = $4 AND shop_id = $5
	`

	result, err := r.db.ExecContext(ctx, strSql, accessToken, refreshToken, timeExpire, marketPlace, shopId)
	return result, err
}

func (r *TokenRepo) GetRefreshToken(ctx context.Context, marketPlace utils.Plaform, shop string) string {
	strSql := `SELECT refresh_token FROM public.market_token WHERE market_place = $1 AND shop_id = $2 LIMIT 1`
	var refreshToken string
	err := r.db.QueryRowContext(ctx, strSql, marketPlace, shop).Scan(&refreshToken)
	if err != nil {
		return ""
	}
	return refreshToken
}
func (r *TokenRepo) GetAccessToken(ctx context.Context, shop string) string {
	strSql := `SELECT access_token FROM public.market_token WHERE market_place = $1 LIMIT 1`
	var accessToken string
	err := r.db.QueryRowContext(ctx, strSql, shop).Scan(&accessToken)
	if err != nil {
		return ""
	}
	return accessToken
}
func (r *TokenRepo) GetExpireTime(ctx context.Context, marketPlace utils.Plaform, shopId string) time.Time {
	strSql := `SELECT expire_time FROM public.market_token WHERE market_place = $1 AND shop_id = $2 LIMIT 1`
	var expireTime time.Time
	err := r.db.QueryRowContext(ctx, strSql, marketPlace, shopId).Scan(&expireTime)
	if err != nil {
		return time.Time{}
	}
	return expireTime
}

func (r *TokenRepo) GetShopID(ctx context.Context, marketPlace utils.Plaform) ([]string, error) {
	strSql := `SELECT shop_id FROM public.market_token WHERE market_place = $1`
	var shopIDs []string
	err := r.db.SelectContext(ctx, &shopIDs, strSql, marketPlace)
	if err != nil {
		return nil, err
	}
	return shopIDs, nil
}

func (r *TokenRepo) GetAccessTokenByShopId(ctx context.Context, marketPlace string, shopID string) string {
	strSql := `SELECT * FROM public.market_token WHERE market_place = $1 AND shop_id = $2`
	var tokens string
	err := r.db.SelectContext(ctx, &tokens, strSql, marketPlace, shopID)
	if err != nil {
		return ""
	}
	return tokens
}

func (r *TokenRepo) GetMarketToken(ctx context.Context, marketPlace string) ([]*models.MarketToken, error) {
	strSql := `SELECT * FROM public.market_token 
			 WHERE market_place = $1 `
	var marketToken []*models.MarketToken
	err := r.db.SelectContext(ctx, &marketToken, strSql, marketPlace)
	if err != nil {
		return nil, err
	}
	return marketToken, nil
}

func (r *TokenRepo) GetAll(ctx context.Context) ([]models.MarketToken, error) {
	strSql := `SELECT * FROM public.market_token `
	var marketToken []models.MarketToken
	err := r.db.SelectContext(ctx, &marketToken, strSql)
	if err != nil {
		return nil, err
	}
	return marketToken, nil
}
