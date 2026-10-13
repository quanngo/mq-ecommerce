package repositories

import (
	"context"
	"main/component/models"
	"main/utils"
	"time"

	"github.com/jmoiron/sqlx"
)

type LogOrderStatusRepo struct {
	db *sqlx.DB
}

func NewLogOrderStatusRepo(db *sqlx.DB) *LogOrderStatusRepo {
	return &LogOrderStatusRepo{
		db: db,
	}
}

func (r *LogOrderStatusRepo) InsertOrderStatus(ctx context.Context, model models.UpdateDatabaseOrder) error {
	strSql := `
		INSERT INTO public.log_order_status (market_place, shop_id, order_id, status, created_date, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.ExecContext(ctx, strSql, model.MarketPlace, model.ShopID, model.OrderID, model.Status, model.CreatedDate.UTC(), model.UpdatedAt.UTC())
	return err
}

func (r *LogOrderStatusRepo) UpdateOrderStatus(ctx context.Context, marketPlace utils.Plaform, shopID, orderID, status string, updatedAt time.Time) error {
	strSql := `
		UPDATE public.log_order_status
		SET status = $1, updated_at = $2
		WHERE order_id = $3 AND market_place = $4 AND shop_id = $5
	`
	_, err := r.db.ExecContext(ctx, strSql, status, updatedAt.UTC(), orderID, marketPlace, shopID)
	return err
}

func (r *LogOrderStatusRepo) GetLastestOrderCreatedUpdate(ctx context.Context, marketPlace string) (time.Time, time.Time, error) {
	var createdAt time.Time
	var updatedAt time.Time

	strSql := `
        SELECT 
            MAX(created_date) AS latest_created_date,
            MAX(updated_at) AS latest_updated_at
        FROM public.log_order_status
        WHERE market_place = $1
    `

	err := r.db.QueryRowContext(ctx, strSql, marketPlace).Scan(&createdAt, &updatedAt)
	return createdAt, updatedAt, err
}

func (r *LogOrderStatusRepo) GetLastestOrderTime(ctx context.Context, marketPlace utils.Plaform, shopID string) (*models.GetLastestOrderTimeGroupMarketResponse, error) {
	query := `
        SELECT 
        	market_place,
            MAX(created_date) AS latest_created_date,
            MAX(updated_at) AS latest_updated_at
        FROM public.log_order_status
		WHERE market_place = $1 AND shop_id = $2
        GROUP BY market_place, shop_id
    `

	var shop []*models.GetLastestOrderTimeGroupMarketResponse
	if err := r.db.SelectContext(ctx, &shop, query, marketPlace, shopID); err != nil {
		return nil, err
	}
	if len(shop) == 0 {
		return nil, nil
	}

	return shop[0], nil
}
