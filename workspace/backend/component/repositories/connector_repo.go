package repositories

import (
	"context"
	"encoding/json"
	"main/component/models"
	tiktok_model "main/component/models/tiktok_model"
	"time"

	"main/utils"

	"github.com/jmoiron/sqlx"
)

type ConnectorRepo struct {
	db *sqlx.DB
}

func NewConnectorRepo(db *sqlx.DB) *ConnectorRepo {
	return &ConnectorRepo{
		db: db,
	}
}

func (r *ConnectorRepo) GetOdEcomOrderDObyStatus(ctx context.Context, marketPlace utils.Plaform, status utils.DeliveryOrderStatus) ([]models.OdEcomOrder, error) {
	query := `
		SELECT
			eo.id,
			eo.market_place,
			eo.order_id,
			eo.lead_id,
			eo.tracking_code,
			eo.customer_name,
			eo.phone,
			eo.address,
			eo.province,
			eo.district,
			eo.sub_district,
			eo.payment_method,
			eo.discount,
			-- eo.product,
			eo.created_date,
			eo.shipping_label,
			eo.warehouse,
			eo.shop_id,
			eo.org_id
		FROM od_do_new odn
		JOIN od_ecom_order eo
			ON eo.tracking_code = odn.tracking_code
		WHERE eo.market_place = $1
		AND odn.status = $2;
    `

	var orders []models.OdEcomOrder
	if err := r.db.SelectContext(ctx, &orders, query, marketPlace, status); err != nil {
		return nil, err
	}

	return orders, nil
}

func (r *ConnectorRepo) GetShopOrderID(ctx context.Context, platform utils.Plaform, shopId string, id int) ([]models.OdEcomOrderID, error) {
	strSql := `SELECT id, order_id, market_place, tracking_code FROM od_ecom_order WHERE market_place = $1 AND shop_id = $2 AND id > $3 LIMIT 100`

	var orders []models.OdEcomOrderID
	err := r.db.SelectContext(ctx, &orders, strSql, platform, shopId, id)
	if err != nil {
		return nil, err
	}
	if len(orders) == 0 {
		return nil, nil
	}

	return orders, nil
}

func (r *ConnectorRepo) UpdateOrderPayment(ctx context.Context, payment tiktok_model.Payment) error {
	strSql := ` UPDATE od_ecom_order 
		SET currency = $2,
			original_shipping_fee = $3,
			original_total_product_price = $4,
			platform_discount = $5,
			seller_discount = $6,
			shipping_fee = $7,
			shipping_fee_cofunded_discount = $8,
			shipping_fee_seller_discount = $9,
			sub_total = $10,
			tax = $11,
			total_amount = $12
		WHERE id = $1
		`

	_, err := r.db.ExecContext(ctx, strSql, payment.Id, payment.Currency,
		payment.OriginalShippingFee,
		payment.OriginalTotalProductPrice,
		payment.PlatformDiscount,
		payment.SellerDiscount,
		payment.ShippingFee,
		payment.ShippingFeeCofundedDiscount,
		payment.ShippingFeeSellerDiscount,
		payment.SubTotal,
		payment.Tax,
		payment.TotalAmount)
	return err
}

func (r *ConnectorRepo) GetOrderIdForStatementTransaction(ctx context.Context, platform utils.Plaform, shopId string) ([]models.OdEcomOrderID, error) {
	const strSql = `
        SELECT oeo.id, oeo.order_id, oeo.market_place, oeo.tracking_code
        FROM public.od_ecom_order AS oeo
        JOIN public.od_sale_order AS oso
        	ON oso.lead_id = oeo.lead_id
        WHERE 
		  	oeo.market_place = $1
		  	AND oeo.shop_id = $2
			AND oso.status = $3
          	AND NOT EXISTS (
            	SELECT 1
            	FROM public.ecom_statement_transaction AS t
            	WHERE t.order_id = oeo.order_id
        )
        ORDER BY oeo.order_id ASC;
    `
	orders := []models.OdEcomOrderID{}
	if err := r.db.SelectContext(ctx, &orders, strSql, platform, shopId, utils.SaleOrderCompleted); err != nil {
		return nil, err
	}
	return orders, nil
}

func (r *ConnectorRepo) InsertEcomStatementTransaction(records []models.EcomStatementTransaction) error {

	if len(records) == 0 {
		return nil
	}

	for i := range records {
		if len(records[i].RawData) == 0 {
			records[i].RawData = json.RawMessage([]byte(`{}`))
		}
		if records[i].CreatedAt.IsZero() {
			records[i].CreatedAt = time.Now()
		}
	}

	strSql := `
        INSERT INTO public.ecom_statement_transaction
        (
            market_place,
            order_id,
            raw_data,
            created_at
        )
        VALUES
        (
            :market_place,
            :order_id,
            :raw_data,
            :created_at
        )
        ON CONFLICT (market_place, order_id) DO UPDATE SET
            raw_data       = EXCLUDED.raw_data,
            created_at         = EXCLUDED.created_at
    `

	if _, err := r.db.NamedExec(strSql, records); err != nil {
		return err
	}
	return nil
}
