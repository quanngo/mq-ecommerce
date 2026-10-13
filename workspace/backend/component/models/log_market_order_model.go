package models

import (
	"encoding/json"
	"main/utils"
	"time"
)

type CreateEcommerceRequest struct {
	MarketPlace   utils.Plaform      `json:"marketPlace"`
	OrderID       string             `json:"orderId"`
	Status        utils.LeadStatus   `json:"status"`
	Products      []ConnectorProduct `json:"products"`
	TrackingCode  string             `json:"trackingCode"`
	ShopID        string             `json:"shopId"`
	Name          string             `json:"name"`
	Phone         string             `json:"phone"`
	Address       string             `json:"address"`
	Province      string             `json:"province"`
	District      string             `json:"district"`
	Subdistrict   string             `json:"subdistrict"`
	Neighborhood  string             `json:"neighborhood"`
	ShippingLabel string             `json:"shippingLabel"`
	Warehouse     string             `json:"warehouse"`
	Comment       string             `json:"comment"`
	TotalAmount   float64            `json:"totalAmount"`
	DisCount      float64            `json:"disCount"`
	PaymentMethod int                `json:"paymentMethod"`
	ShippingFee   float64            `json:"shippingFee"`
	EnableAddress bool               `json:"enableAddress"`
	RawOrder      string             `json:"rawOrder"`
	RawOrderByte  interface{}
}

type LogMarketPlaceOrder struct {
	MarketPlace string          `json:"market_place" db:"market_place"`
	ShopID      string          `json:"shop_id" db:"shop_id"`
	OrderID     string          `json:"order_id" db:"order_id"`
	Payload     json.RawMessage `json:"payload" db:"payload"`
	CreatedDate time.Time       `json:"created_date" db:"created_date"`
	OrgID       int             `json:"org_id" db:"org_id"`
	Status      int             `json:"status" db:"status"`
	UpdatedDate *time.Time      `json:"updated_date" db:"updated_date"`
}

type GetLogMarketPlaceOrder struct {
	MarketPlace utils.Plaform    `json:"market_place"`
	ShopID      string           `json:"shop_id"`
	OrderID     string           `json:"order_id"`
	Status      utils.LeadStatus `json:"status"`
}

type RequestUpdateLogOrderStatus struct {
	ID          string           `json:"id"`
	Status      utils.LeadStatus `json:"status"`
	UpdatedAt   time.Time        `json:"updatedAt"`
	MarketPlace utils.Plaform    `json:"marketPlace"`
	ShopID      string           `json:"shopID"`
}

type GetLastestOrderRefundTimeGroupMarketResponse struct {
	MarketPlace *string    `db:"market_place" json:"market_place"`
	ShopID      *string    `db:"shop_id" json:"shop_id"`
	Status      *string    `db:"status" json:"status"`
	CreatedAt   *time.Time `db:"latest_created_date" json:"latest_created_date"`
	UpdatedAt   *time.Time `db:"latest_updated_at" json:"latest_updated_at"`
}
