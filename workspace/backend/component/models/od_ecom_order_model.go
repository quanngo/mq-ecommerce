package models

import (
	"time"
)

type OdEcomOrder struct {
	ID            int     `db:"id"`
	MarketPlace   string  `db:"market_place"`
	OrderID       string  `db:"order_id"`
	LeadID        int     `db:"lead_id"`
	TrackingCode  string  `db:"tracking_code"`
	CustomerName  string  `db:"customer_name"`
	Phone         string  `db:"phone"`
	Address       string  `db:"address"`
	Province      string  `db:"province"`
	District      string  `db:"district"`
	SubDistrict   string  `db:"sub_district"`
	PaymentMethod string  `db:"payment_method"`
	Discount      float64 `db:"discount"`
	// Product       db.RawMessage `db:"product"`
	CreatedDate   time.Time `db:"created_date"`
	ShippingLabel string    `db:"shipping_label"`
	Warehouse     string    `db:"warehouse"`
	ShopID        string    `db:"shop_id"`
	OrgID         int       `db:"org_id"`
}

type OdEcomOrderID struct {
	ID           int    `db:"id"`
	MarketPlace  string `db:"market_place"`
	OrderID      string `db:"order_id"`
	TrackingCode string `db:"tracking_code"`
}
