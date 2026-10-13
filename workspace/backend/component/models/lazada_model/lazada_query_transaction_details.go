package models

// Root response
type SettlementResponse struct {
	Code      string           `json:"code"`
	Data      []SettlementItem `json:"data"`
	RequestID string           `json:"request_id"`
}

// One transaction line
type SettlementItem struct {
	OrderNo             string `json:"order_no"`
	TransactionDate     string `json:"transaction_date"`
	Amount              string `json:"amount"`
	PaidStatus          string `json:"paid_status"`
	ShippingProvider    string `json:"shipping_provider"`
	WHTIncludedInAmount string `json:"WHT_included_in_amount"`
	PaymentRefID        string `json:"payment_ref_id"`
	LazadaSKU           string `json:"lazada_sku"`
	FeeType             string `json:"fee_type"`
	TransactionType     string `json:"transaction_type"`
	OrderItemNo         string `json:"orderItem_no"`
	OrderItemStatus     string `json:"orderItem_status"`
	Reference           string `json:"reference"`
	FeeName             string `json:"fee_name"`
	ShippingSpeed       string `json:"shipping_speed"`
	WHTAmount           string `json:"WHT_amount"`
	TransactionNumber   string `json:"transaction_number"`
	SellerSKU           string `json:"seller_sku"`
	Statement           string `json:"statement"`
	Details             string `json:"details"`
	Comment             string `json:"comment"`
	VATInAmount         string `json:"VAT_in_amount"`
	ShipmentType        string `json:"shipment_type"`
}
