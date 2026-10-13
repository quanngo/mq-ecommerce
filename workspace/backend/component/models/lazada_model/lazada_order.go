package models

import "time"

type OrderItemResponse struct {
	Data      []OrderItem `json:"data"`
	Code      string      `json:"code"`
	RequestID string      `json:"request_id"`
}

type OrderItem struct {
	PickUpStoreInfo             map[string]interface{} `json:"pick_up_store_info"` // object
	TaxAmount                   float64                `json:"tax_amount"`
	Reason                      string                 `json:"reason"`
	SlaTimeStamp                string                 `json:"sla_time_stamp"`
	VoucherSeller               float64                `json:"voucher_seller"`
	PurchaseOrderID             string                 `json:"purchase_order_id"`
	VoucherCodeSeller           string                 `json:"voucher_code_seller"`
	VoucherCode                 string                 `json:"voucher_code"`
	PackageID                   string                 `json:"package_id"`
	BuyerID                     int64                  `json:"buyer_id"`
	Variation                   string                 `json:"variation"`
	BizGroup                    int64                  `json:"biz_group"`
	ProductID                   string                 `json:"product_id"`
	VoucherCodePlatform         string                 `json:"voucher_code_platform"`
	PurchaseOrderNumber         string                 `json:"purchase_order_number"`
	SKU                         string                 `json:"sku"`
	GiftWrapping                string                 `json:"gift_wrapping"`
	OrderType                   string                 `json:"order_type"`
	InvoiceNumber               string                 `json:"invoice_number"`
	CanEscalatePickup           bool                   `json:"can_escalate_pickup"`
	CancelReturnInitiator       string                 `json:"cancel_return_initiator"`
	ShopSKU                     string                 `json:"shop_sku"`
	IsReroute                   int                    `json:"is_reroute"`
	StagePayStatus              string                 `json:"stage_pay_status"`
	SKUID                       string                 `json:"sku_id"`
	TrackingCodePre             string                 `json:"tracking_code_pre"`
	OrderItemID                 int64                  `json:"order_item_id"`
	ShopID                      string                 `json:"shop_id"`
	OrderFlag                   string                 `json:"order_flag"`
	IsFBL                       int                    `json:"is_fbl"`
	Name                        string                 `json:"name"`
	DeliveryOptionSof           int                    `json:"delivery_option_sof"`
	OrderID                     int64                  `json:"order_id"`
	FulfillmentSLA              string                 `json:"fulfillment_sla"`
	Status                      string                 `json:"status"`
	ProductMainImage            string                 `json:"product_main_image"`
	VoucherPlatform             float64                `json:"voucher_platform"`
	PaidPrice                   float64                `json:"paid_price"`
	ProductDetailURL            string                 `json:"product_detail_url"`
	WarehouseCode               string                 `json:"warehouse_code"`
	PromisedShippingTime        string                 `json:"promised_shipping_time"`
	ShippingType                string                 `json:"shipping_type"`
	CreatedAt                   string                 `json:"created_at"`
	SupplyPrice                 float64                `json:"supply_price"`
	Mp3Order                    bool                   `json:"mp3_order"`
	VoucherSellerLpi            float64                `json:"voucher_seller_lpi"`
	ShippingFeeDiscountPlatform float64                `json:"shipping_fee_discount_platform"`
	Personalization             string                 `json:"personalization"`
	WalletCredits               float64                `json:"wallet_credits"`
	UpdatedAt                   string                 `json:"updated_at"`
	Currency                    string                 `json:"currency"`
	ShippingProviderType        string                 `json:"shipping_provider_type"`
	VoucherPlatformLpi          float64                `json:"voucher_platform_lpi"`
	ShippingFeeOriginal         float64                `json:"shipping_fee_original"`
	ItemPrice                   float64                `json:"item_price"`
	IsDigital                   int                    `json:"is_digital"`
	ShippingServiceCost         float64                `json:"shipping_service_cost"`
	TrackingCode                string                 `json:"tracking_code"`
	ShippingFeeDiscountSeller   float64                `json:"shipping_fee_discount_seller"`
	ShippingAmount              float64                `json:"shipping_amount"`
	ReasonDetail                string                 `json:"reason_detail"`
	ReturnStatus                string                 `json:"return_status"`
	SemiManaged                 bool                   `json:"semi_managed"`
	ShipmentProvider            string                 `json:"shipment_provider"`
	PriorityFulfillmentTag      string                 `json:"priority_fulfillment_tag"`
	VoucherAmount               float64                `json:"voucher_amount"`
	SupplyPriceCurrency         string                 `json:"supply_price_currency"`
	DigitalDeliveryInfo         string                 `json:"digital_delivery_info"`
	ExtraAttributes             string                 `json:"extra_attributes"`
}

type OrdersResponse struct {
	Code      string `json:"code"`
	Data      Datas  `json:"data"`
	RequestID string `json:"request_id"`
}

type Datas struct {
	Count      int64   `json:"count"`
	CountTotal int64   `json:"countTotal"`
	Orders     []Order `json:"orders"`
}

type Order struct {
	VoucherPlatform             float64        `json:"voucher_platform"`
	Voucher                     float64        `json:"voucher"`
	WarehouseCode               string         `json:"warehouse_code"`
	OrderNumber                 float64        `json:"order_number"`
	VoucherSeller               float64        `json:"voucher_seller"`
	CreatedAt                   string         `json:"created_at"`
	VoucherCode                 string         `json:"voucher_code"`
	GiftOption                  bool           `json:"gift_option"`
	IsCancelPending             bool           `json:"is_cancel_pending"`
	ShippingFeeDiscountPlatform float64        `json:"shipping_fee_discount_platform"`
	CustomerLastName            string         `json:"customer_last_name"`
	PromisedShippingTimes       string         `json:"promised_shipping_times"`
	UpdatedAt                   string         `json:"updated_at"`
	Price                       string         `json:"price"`
	NationalRegistrationNumber  string         `json:"national_registration_number"`
	ShippingFeeOriginal         float64        `json:"shipping_fee_original"`
	PaymentMethod               string         `json:"payment_method"`
	AddressUpdatedAt            string         `json:"address_updated_at"`
	RecipientInfo               RecipientInfos `json:"recipient_info"`
	BuyerNote                   string         `json:"buyer_note"`
	CustomerFirstName           string         `json:"customer_first_name"`
	ShippingFeeDiscountSeller   float64        `json:"shipping_fee_discount_seller"`
	ShippingFee                 float64        `json:"shipping_fee"`
	BranchNumber                string         `json:"branch_number"`
	TaxCode                     string         `json:"tax_code"`
	ItemsCount                  int64          `json:"items_count"`
	DeliveryInfo                string         `json:"delivery_info"`
	Statuses                    []string       `json:"statuses"`
	AddressBilling              Addresss       `json:"address_billing"`
	ExtraAttributes             string         `json:"extra_attributes"`
	OrderID                     int64          `json:"order_id"`
	NeedCancelConfirm           bool           `json:"need_cancel_confirm"`
	Remarks                     string         `json:"remarks"`
	GiftMessage                 string         `json:"gift_message"`
	AddressShipping             Addresss       `json:"address_shipping"`
}

type RecipientInfos struct {
	IdentifyNo    string `json:"identify_no"`
	DetailAddress string `json:"detail_address"`
	PassportNo    string `json:"passport_no"`
}

type Addresss struct {
	Country         string `json:"country"`
	Address3        string `json:"address3"`
	Address2        string `json:"address2"`
	City            string `json:"city"`
	Address1        string `json:"address1"`
	Phone2          string `json:"phone2"`
	LastName        string `json:"last_name"`
	AddressDistrict string `json:"addressDsitrict"`
	Phone           string `json:"phone"`
	PostCode        string `json:"post_code"`
	Address5        string `json:"address5"`
	Address4        string `json:"address4"`
	FirstName       string `json:"first_name"`
}

type UpdateDatabaseOrder struct {
	MarketPlace string
	OrderID     string
	Status      string
	CreatedDate time.Time
	UpdatedAt   time.Time
}

type Address struct {
	Country         string `json:"country"`
	Address1        string `json:"address1"`
	Address2        string `json:"address2"`
	Address3        string `json:"address3"`
	Address4        string `json:"address4"`
	Address5        string `json:"address5"`
	City            string `json:"city"`
	AddressDistrict string `json:"addressDistrict"`
	FirstName       string `json:"first_name"`
	LastName        string `json:"last_name"`
	Phone           string `json:"phone"`
	Phone2          string `json:"phone2"`
	PostCode        string `json:"post_code"`
}

type OrderData struct {
	Voucher                     float64  `json:"voucher"`
	WarehouseCode               string   `json:"warehouse_code"`
	OrderNumber                 int64    `json:"order_number"`
	CreatedAt                   string   `json:"created_at"`
	VoucherCode                 string   `json:"voucher_code"`
	GiftOption                  bool     `json:"gift_option"`
	IsCancelPending             bool     `json:"is_cancel_pending"`
	ShippingFeeDiscountPlatform float64  `json:"shipping_fee_discount_platform"`
	CustomerLastName            string   `json:"customer_last_name"`
	UpdatedAt                   string   `json:"updated_at"`
	PromisedShippingTimes       string   `json:"promised_shipping_times"`
	Price                       string   `json:"price"`
	NationalRegistrationNumber  string   `json:"national_registration_number"`
	ShippingFeeOriginal         float64  `json:"shipping_fee_original"`
	PaymentMethod               string   `json:"payment_method"`
	RecipientInfo               any      `json:"recipient_info"` // Nếu có cấu trúc cụ thể, bạn có thể thay `any` bằng struct
	BuyerNote                   string   `json:"buyer_note"`
	CustomerFirstName           string   `json:"customer_first_name"`
	ShippingFeeDiscountSeller   float64  `json:"shipping_fee_discount_seller"`
	ShippingFee                 float64  `json:"shipping_fee"`
	BranchNumber                string   `json:"branch_number"`
	TaxCode                     string   `json:"tax_code"`
	ItemsCount                  int      `json:"items_count"`
	DeliveryInfo                string   `json:"delivery_info"`
	Statuses                    []string `json:"statuses"`
	AddressBilling              Address  `json:"address_billing"`
	ExtraAttributes             string   `json:"extra_attributes"`
	OrderID                     int64    `json:"order_id"`
	NeedCancelConfirm           bool     `json:"need_cancel_confirm"`
	GiftMessage                 string   `json:"gift_message"`
	Remarks                     string   `json:"remarks"`
	AddressShipping             Address  `json:"address_shipping"`
}

type OrderDetailResponse struct {
	Data      *OrderData `json:"data"`
	Code      string     `json:"code"`
	RequestID string     `json:"request_id"`
	TraceID   string     `json:"_trace_id_"`
}
