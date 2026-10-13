package repositories

import (
	"context"
	"encoding/json"
	"main/component/models"
	"main/utils"
	"strings"

	"github.com/jmoiron/sqlx"
)

type LogMarketPlaceOrderRepo struct {
	db *sqlx.DB
}

func NewLogMarketPlaceOrderRepo(db *sqlx.DB) *LogMarketPlaceOrderRepo {
	return &LogMarketPlaceOrderRepo{
		db: db,
	}
}

func (r *LogMarketPlaceOrderRepo) CreateLogOrder(ctx context.Context, orgId int, order models.CreateEcommerceRequest) error {
	raw, _ := json.Marshal(order.RawOrderByte)
	strSql := `
		INSERT INTO public.log_market_place_order (market_place, shop_id, status, order_id, payload, created_date, org_id)
		VALUES ($1, $2, $3, $4, $5, NOW(), $6)
	`
	_, err := r.db.ExecContext(ctx, strSql, order.MarketPlace, order.ShopID, order.Status, order.OrderID, raw, orgId)
	return err
}

func (r *LogMarketPlaceOrderRepo) GetLogOrder(ctx context.Context, req models.GetLogMarketPlaceOrder) ([]models.LogMarketPlaceOrder, error) {

	conditions := make([]string, 0)
	params := make([]interface{}, 0)
	var whereParams string
	index := 1

	index = utils.AppendStringCondition("market_place", string(req.MarketPlace), index, &conditions, &params)
	index = utils.AppendStringCondition("shop_id", req.ShopID, index, &conditions, &params)
	index = utils.AppendStringCondition("order_id", req.OrderID, index, &conditions, &params)
	index = utils.AppendIntCondition("status", int(req.Status), index, &conditions, &params)

	if len(conditions) > 0 {
		whereParams = " WHERE " + (strings.Join(conditions, " AND "))
	}

	query := `
		SELECT market_place, shop_id, order_id, payload, created_date, org_id, status, updated_date 
		FROM public.log_market_place_order 
    ` + whereParams

	var orders []models.LogMarketPlaceOrder
	if err := r.db.SelectContext(ctx, &orders, query, params...); err != nil {
		return nil, err
	}
	return orders, nil
}

func (r *LogMarketPlaceOrderRepo) UpdateOrderStatus(ctx context.Context, order models.RequestUpdateLogOrderStatus) error {
	strSql := `
		UPDATE public.log_market_place_order
		SET status = $1, updated_date = $2
		WHERE order_id = $3 AND market_place = $4 AND shop_id = $5
	`
	_, err := r.db.ExecContext(ctx, strSql, order.Status, order.UpdatedAt.UTC(), order.ID, order.MarketPlace, order.ShopID)
	return err
}

func (r *LogMarketPlaceOrderRepo) GetLatestOrderRefundTime(ctx context.Context, status utils.LeadStatus, marketPlace utils.Plaform, shopID string) (*models.GetLastestOrderRefundTimeGroupMarketResponse, error) {
	query := `
        SELECT 	      
        	market_place,
	        shop_id,
	        status,
            MAX(created_date) AS latest_created_date,
            MAX(updated_date) AS latest_updated_at
        FROM public.log_market_place_order
		WHERE status = $1 AND market_place = $2 AND shop_id = $3
        GROUP BY market_place, shop_id, status
    `

	var shop []*models.GetLastestOrderRefundTimeGroupMarketResponse
	if err := r.db.SelectContext(ctx, &shop, query, status, marketPlace, shopID); err != nil {
		return nil, err
	}
	if len(shop) == 0 {
		return nil, nil
	}

	return shop[0], nil
}

func (r *LogMarketPlaceOrderRepo) CheckOrderExists(platform utils.Plaform, orderID string) (bool, error) {
	var exists bool
	query := `
        SELECT EXISTS(
            SELECT 1
            FROM public.log_market_place_order
            WHERE market_place = $1 AND order_id = $2 
			LIMIT 1
        )
    `
	err := r.db.Get(&exists, query, platform, orderID)
	return exists, err
}
