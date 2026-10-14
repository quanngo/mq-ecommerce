package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"main/component/models"
	"strings"

	"github.com/jmoiron/sqlx"
)

// OrderQueryRepo is the read side of public.ecom_order / ecom_order_item for
// the admin app's Orders menu. Writes stay in EcomOrderRepo.
type OrderQueryRepo struct {
	db *sqlx.DB
}

func NewOrderQueryRepo(db *sqlx.DB) *OrderQueryRepo {
	return &OrderQueryRepo{
		db: db,
	}
}

const orderSummaryColumns = `
	o.id, o.market_place, o.shop_id, o.order_id, o.tracking_code, o.lead_status,
	o.do_status, o.so_status, o.customer_name, o.phone, o.total_amount, o.payment_method,
	(SELECT COUNT(*) FROM public.ecom_order_item i WHERE i.ecom_order_id = o.id) AS item_count,
	o.created_at, o.updated_at`

// List returns one page of orders matching f, newest first, and the total
// number of matching orders.
func (r *OrderQueryRepo) List(ctx context.Context, f models.OrderFilter) ([]models.OrderSummary, int, error) {
	where, args := orderWhere(f)

	var total int
	if err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM public.ecom_order o`+where, args...); err != nil {
		return nil, 0, err
	}

	strSql := fmt.Sprintf(`SELECT %s FROM public.ecom_order o%s
		ORDER BY o.created_at DESC, o.id DESC
		LIMIT $%d OFFSET $%d`, orderSummaryColumns, where, len(args)+1, len(args)+2)
	orders := []models.OrderSummary{}
	if err := r.db.SelectContext(ctx, &orders, strSql, append(args, f.Limit, f.Offset)...); err != nil {
		return nil, 0, err
	}
	return orders, total, nil
}

func orderWhere(f models.OrderFilter) (string, []interface{}) {
	var conds []string
	var args []interface{}
	arg := func(v interface{}) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}
	if f.Q != "" {
		p := arg("%" + escapeLike(f.Q) + "%")
		conds = append(conds, fmt.Sprintf("(o.order_id ILIKE %[1]s OR o.tracking_code ILIKE %[1]s OR o.customer_name ILIKE %[1]s OR o.phone ILIKE %[1]s)", p))
	}
	if f.MarketPlace != "" {
		conds = append(conds, "o.market_place = "+arg(f.MarketPlace))
	}
	if f.LeadStatus != 0 {
		conds = append(conds, "o.lead_status = "+arg(f.LeadStatus))
	}
	if f.CreatedFrom != nil {
		conds = append(conds, "o.created_at >= "+arg(*f.CreatedFrom))
	}
	if f.CreatedTo != nil {
		conds = append(conds, "o.created_at < "+arg(*f.CreatedTo))
	}
	if len(conds) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(conds, " AND "), args
}

// escapeLike makes user input match literally inside ILIKE (default escape '\').
func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

// Get returns the order with its items, or (nil, nil) when there is none.
func (r *OrderQueryRepo) Get(ctx context.Context, id int64) (*models.OrderDetail, error) {
	strSql := `SELECT ` + orderSummaryColumns + `,
		o.address, o.province, o.district, o.sub_district, o.neighborhood,
		o.warehouse, o.shipping_label, o.comment, o.discount, o.shipping_fee
		FROM public.ecom_order o
		WHERE o.id = $1`
	var order models.OrderDetail
	if err := r.db.GetContext(ctx, &order, strSql, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	order.Items = []models.OrderItem{}
	itemSql := `
		SELECT id, name, sku, quantity, price
		FROM public.ecom_order_item
		WHERE ecom_order_id = $1
		ORDER BY id
	`
	if err := r.db.SelectContext(ctx, &order.Items, itemSql, id); err != nil {
		return nil, err
	}
	return &order, nil
}
