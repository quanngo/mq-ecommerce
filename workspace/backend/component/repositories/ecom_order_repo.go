package repositories

import (
	"context"
	"encoding/json"
	"main/component/models"

	"github.com/jmoiron/sqlx"
)

type EcomOrderRepo struct {
	db *sqlx.DB
}

func NewEcomOrderRepo(db *sqlx.DB) *EcomOrderRepo {
	return &EcomOrderRepo{
		db: db,
	}
}

// CreateOrder inserts the order and its items in one transaction.
// Returns created=false when (market_place, order_id) already exists.
func (r *EcomOrderRepo) CreateOrder(ctx context.Context, orgId int, order models.CreateEcommerceRequest) (bool, error) {
	var raw interface{}
	if order.RawOrderByte != nil {
		b, err := json.Marshal(order.RawOrderByte)
		if err != nil {
			return false, err
		}
		raw = b
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	strSql := `
		INSERT INTO public.ecom_order (
			org_id, market_place, shop_id, order_id, tracking_code, lead_status,
			customer_name, phone, address, province, district, sub_district, neighborhood,
			shipping_label, warehouse, comment,
			total_amount, discount, shipping_fee, payment_method, enable_address, raw_order
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22)
		ON CONFLICT (market_place, order_id) WHERE order_id <> '' DO NOTHING
		RETURNING id
	`
	var orderPk int64
	rows, err := tx.QueryContext(ctx, strSql,
		orgId, order.MarketPlace, order.ShopID, order.OrderID, order.TrackingCode, order.Status,
		order.Name, order.Phone, order.Address, order.Province, order.District, order.Subdistrict, order.Neighborhood,
		order.ShippingLabel, order.Warehouse, order.Comment,
		order.TotalAmount, order.DisCount, order.ShippingFee, order.PaymentMethod, order.EnableAddress, raw,
	)
	if err != nil {
		return false, err
	}
	created := rows.Next()
	if created {
		if err := rows.Scan(&orderPk); err != nil {
			rows.Close()
			return false, err
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return false, err
	}
	if !created {
		return false, nil
	}

	itemSql := `
		INSERT INTO public.ecom_order_item (ecom_order_id, name, sku, quantity, price)
		VALUES ($1, $2, $3, $4, $5)
	`
	for _, p := range order.Products {
		if _, err := tx.ExecContext(ctx, itemSql, orderPk, p.Name, p.Sku, p.Quantity, p.Price); err != nil {
			return false, err
		}
	}

	if err := tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

// UpdateDOStatus matches by order_id (scoped to market_place when given), or by
// tracking_code when there is no order_id. Empty status / zero so_status keep
// the stored value. Returns the number of rows updated.
func (r *EcomOrderRepo) UpdateDOStatus(ctx context.Context, req models.UpdateDOStatusRequest) (int64, error) {
	strSql := `
		UPDATE public.ecom_order
		SET do_status     = COALESCE(NULLIF($1::text, ''), do_status),
		    so_status     = COALESCE(NULLIF($2::int, 0), so_status),
		    tracking_code = COALESCE(NULLIF($3::text, ''), tracking_code),
		    updated_at    = NOW()
		WHERE ($4::text <> '' AND order_id = $4::text AND ($5::text = '' OR market_place = $5::text))
		   OR ($4::text = '' AND $3::text <> '' AND tracking_code = $3::text)
	`
	result, err := r.db.ExecContext(ctx, strSql,
		req.Status, int(req.SoStatus), req.TrackingCode, req.OrderID, string(req.MarketPlace))
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
