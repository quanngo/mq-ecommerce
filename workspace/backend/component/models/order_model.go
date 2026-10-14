package models

import "time"

// Orders menu (EC-4): read-only views of public.ecom_order / ecom_order_item.
// raw_order and org_id are never returned.

// OrderListQuery is GET /api/v1/orders' query string, as sent. The service
// validates it into an OrderFilter.
type OrderListQuery struct {
	Q           string `form:"q"`
	MarketPlace string `form:"market_place"`
	LeadStatus  string `form:"lead_status"`
	From        string `form:"from"` // YYYY-MM-DD, Vietnam time, inclusive
	To          string `form:"to"`   // YYYY-MM-DD, Vietnam time, inclusive
	Page        string `form:"page"`
	PageSize    string `form:"page_size"`
}

// OrderFilter is a validated OrderListQuery. Zero values mean "no filter".
type OrderFilter struct {
	Q           string
	MarketPlace string
	LeadStatus  int
	CreatedFrom *time.Time // created_at >= CreatedFrom
	CreatedTo   *time.Time // created_at <  CreatedTo (the day after `to`)
	Limit       int
	Offset      int
}

// OrderSummary is a row of the order list.
type OrderSummary struct {
	ID            int64     `db:"id" json:"id"`
	MarketPlace   string    `db:"market_place" json:"market_place"`
	ShopID        string    `db:"shop_id" json:"shop_id"`
	OrderID       string    `db:"order_id" json:"order_id"`
	TrackingCode  string    `db:"tracking_code" json:"tracking_code"`
	LeadStatus    int       `db:"lead_status" json:"lead_status"`
	DoStatus      string    `db:"do_status" json:"do_status"`
	SoStatus      *int      `db:"so_status" json:"so_status"`
	CustomerName  string    `db:"customer_name" json:"customer_name"`
	Phone         string    `db:"phone" json:"phone"`
	TotalAmount   float64   `db:"total_amount" json:"total_amount"`
	PaymentMethod int       `db:"payment_method" json:"payment_method"`
	ItemCount     int       `db:"item_count" json:"item_count"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time `db:"updated_at" json:"updated_at"`
}

type OrderListResponse struct {
	Orders   []OrderSummary `json:"orders"`
	Total    int            `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"page_size"`
}

// OrderItem maps public.ecom_order_item.
type OrderItem struct {
	ID       int64   `db:"id" json:"id"`
	Name     string  `db:"name" json:"name"`
	Sku      string  `db:"sku" json:"sku"`
	Quantity int     `db:"quantity" json:"quantity"`
	Price    float64 `db:"price" json:"price"`
}

// OrderDetail is one order with its address, amounts and items.
type OrderDetail struct {
	OrderSummary
	Address       string      `db:"address" json:"address"`
	Province      string      `db:"province" json:"province"`
	District      string      `db:"district" json:"district"`
	SubDistrict   string      `db:"sub_district" json:"sub_district"`
	Neighborhood  string      `db:"neighborhood" json:"neighborhood"`
	Warehouse     string      `db:"warehouse" json:"warehouse"`
	ShippingLabel string      `db:"shipping_label" json:"shipping_label"`
	Comment       string      `db:"comment" json:"comment"`
	Discount      float64     `db:"discount" json:"discount"`
	ShippingFee   float64     `db:"shipping_fee" json:"shipping_fee"`
	Items         []OrderItem `db:"-" json:"items"`
}

type OrderDetailResponse struct {
	Order OrderDetail `json:"order"`
}
