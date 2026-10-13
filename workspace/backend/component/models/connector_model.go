package models

import (
	"encoding/json"
	"main/utils"
	"time"
)

type ConnectorResponse struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	Total     int    `json:"total"`
	TotalSelf int    `json:"totalSelf"`
}

type ConnectorProduct struct {
	Name     string  `json:"name"`
	Sku      string  `json:"sku"`
	Quantity int     `json:"quantity"`
	Price    float64 `json:"price"`
}

type UpdateDOStatusRequest struct {
	TrackingCode string `json:"tracking_code"`
	Status       string `json:"status"`

	OrderID     string                `json:"order_id"`
	SoStatus    utils.SaleOrderStatus `json:"so_status"`
	MarketPlace utils.Plaform         `json:"market_place,omitempty"` // scopes the DB update; not sent to Connector

	UpdatedAt time.Time
}

type MarketPlaceMapping struct {
	ID          string `db:"id" json:"id"`
	ConnectorID string `db:"tms_id" json:"tms_id"`
	EcommerceID string `db:"ecommerce_id" json:"ecommerce_id"`
	Quantity    int    `db:"quantity" json:"quantity"`
	Name        string `db:"name" json:"name"`
	Type        string `db:"type" json:"Type"`
	MarketPlace string `db:"market_place" json:"market_place"`
	OrgID       int    `db:"org_id" json:"org_id"`

	ProductName string `json:"-"`
	ProductSku  string `json:"-"`
}
type WarehouseConfig struct {
	Warehouse string `db:"warehouse" json:"warehouse"`
	ShopID    string `db:"shop_id" json:"shop_id"`
}

type GetLastestOrderTimeGroupMarketResponse struct {
	MarketPlace *string    `db:"market_place" json:"market_place"`
	ShopID      *string    `db:"shop_id" json:"shop_id"`
	CreatedAt   *time.Time `db:"latest_created_date" json:"latest_created_date"`
	UpdatedAt   *time.Time `db:"latest_updated_at" json:"latest_updated_at"`
}

type OrderEcomStatementTransaction struct {
	ID               int64           `json:"id" db:"id"`
	MarketPlace      utils.Plaform   `json:"market_place" db:"market_place"`
	OrderID          string          `json:"order_id" db:"order_id"`
	TrackingCode     string          `json:"tracking_code" db:"tracking_code"`
	Currency         string          `json:"currency" db:"currency"`
	FeeAndTaxAmount  string          `json:"fee_and_tax_amount" db:"fee_and_tax_amount"`
	RevenueAmount    string          `json:"revenue_amount" db:"revenue_amount"`
	SettlementAmount string          `json:"settlement_amount" db:"settlement_amount"`
	Transactions     json.RawMessage `json:"transactions" db:"transactions"` // jsonb
	CreatedAt        time.Time       `json:"created_at" db:"created_at"`
}
