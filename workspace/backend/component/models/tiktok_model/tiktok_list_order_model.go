package models

type TikTokOrderResponse struct {
	Code      int             `json:"code"`
	Data      TikTokOrderData `json:"data"`
	Message   string          `json:"message"`
	RequestID string          `json:"request_id"`
}

type TikTokOrderData struct {
	NextPageToken string        `json:"next_page_token"`
	TotalCount    int           `json:"total_count"`
	Orders        []TikTokOrder `json:"orders"`
}

type TikTokOrder struct {
	ID                         string           `json:"id"`
	BuyerMessage               string           `json:"buyer_message"`
	CancellationInitiator      string           `json:"cancellation_initiator"`
	ShippingProviderID         string           `json:"shipping_provider_id"`
	CreateTime                 int64            `json:"create_time"`
	ShippingProvider           string           `json:"shipping_provider"`
	Packages                   []TikTokPackage  `json:"packages"`
	Payment                    TikTokPayment    `json:"payment"`
	RecipientAddress           TikTokAddress    `json:"recipient_address"`
	Status                     string           `json:"status"`
	FulfillmentType            string           `json:"fulfillment_type"`
	DeliveryType               string           `json:"delivery_type"`
	PaidTime                   int64            `json:"paid_time"`
	RtsSlaTime                 int64            `json:"rts_sla_time"`
	TtsSlaTime                 int64            `json:"tts_sla_time"`
	CancelReason               string           `json:"cancel_reason"`
	UpdateTime                 int64            `json:"update_time"`
	PaymentMethodName          string           `json:"payment_method_name"`
	RtsTime                    int64            `json:"rts_time"`
	TrackingNumber             string           `json:"tracking_number"`
	SplitOrCombineTag          string           `json:"split_or_combine_tag"`
	HasUpdatedRecipientAddress bool             `json:"has_updated_recipient_address"`
	CancelOrderSlaTime         int64            `json:"cancel_order_sla_time"`
	WarehouseID                string           `json:"warehouse_id"`
	RequestCancelTime          int64            `json:"request_cancel_time"`
	ShippingType               string           `json:"shipping_type"`
	UserID                     string           `json:"user_id"`
	SellerNote                 string           `json:"seller_note"`
	DeliverySlaTime            int64            `json:"delivery_sla_time"`
	IsCod                      bool             `json:"is_cod"`
	DeliveryOptionID           string           `json:"delivery_option_id"`
	CancelTime                 int64            `json:"cancel_time"`
	NeedUploadInvoice          string           `json:"need_upload_invoice"`
	DeliveryOptionName         string           `json:"delivery_option_name"`
	Cpf                        string           `json:"cpf"`
	LineItems                  []TikTokLineItem `json:"line_items"`
	BuyerEmail                 string           `json:"buyer_email"`
	DeliveryDueTime            int64            `json:"delivery_due_time"`
	IsSampleOrder              bool             `json:"is_sample_order"`
	ShippingDueTime            int64            `json:"shipping_due_time"`
	CollectionDueTime          int64            `json:"collection_due_time"`
	DeliveryOptionRequiredTime int64            `json:"delivery_option_required_delivery_time"`
	IsOnHoldOrder              bool             `json:"is_on_hold_order"`
	DeliveryTime               int64            `json:"delivery_time"`
	IsReplacementOrder         bool             `json:"is_replacement_order"`
	CollectionTime             int64            `json:"collection_time"`
	ReplacedOrderID            string           `json:"replaced_order_id"`
	IsBuyerRequestCancel       bool             `json:"is_buyer_request_cancel"`
	PickUpCutOffTime           int64            `json:"pick_up_cut_off_time"`
	FastDispatchSlaTime        int64            `json:"fast_dispatch_sla_time"`
	CommercePlatform           string           `json:"commerce_platform"`
	OrderType                  string           `json:"order_type"`
	ReleaseDate                int64            `json:"release_date"`
	HandlingDuration           TikTokHandling   `json:"handling_duration"`
	AutoCombineGroupID         string           `json:"auto_combine_group_id"`
	CpfName                    string           `json:"cpf_name"`
	IsExchangeOrder            bool             `json:"is_exchange_order"`
	ExchangeSourceOrderID      string           `json:"exchange_source_order_id"`
	ConsultationID             string           `json:"consultation_id"`
	FastDeliveryProgram        string           `json:"fast_delivery_program"`
	FulfillmentPriorityLevel   int              `json:"fulfillment_priority_level"`
	RecommendedShippingTime    int64            `json:"recommended_shipping_time"`
}

type TikTokPackage struct {
	ID string `json:"id"`
}

type TikTokPayment struct {
	Currency                    string `json:"currency"`
	SubTotal                    string `json:"sub_total"`
	ShippingFee                 string `json:"shipping_fee"`
	SellerDiscount              string `json:"seller_discount"`
	PlatformDiscount            string `json:"platform_discount"`
	TotalAmount                 string `json:"total_amount"`
	OriginalTotalProductPrice   string `json:"original_total_product_price"`
	OriginalShippingFee         string `json:"original_shipping_fee"`
	ShippingFeeSellerDiscount   string `json:"shipping_fee_seller_discount"`
	ShippingFeePlatformDiscount string `json:"shipping_fee_platform_discount"`
	ShippingFeeCofundedDiscount string `json:"shipping_fee_cofunded_discount"`
	Tax                         string `json:"tax"`
	SmallOrderFee               string `json:"small_order_fee"`
	ShippingFeeTax              string `json:"shipping_fee_tax"`
	ProductTax                  string `json:"product_tax"`
	RetailDeliveryFee           string `json:"retail_delivery_fee"`
	BuyerServiceFee             string `json:"buyer_service_fee"`
	HandlingFee                 string `json:"handling_fee"`
	ShippingInsuranceFee        string `json:"shipping_insurance_fee"`
	ItemInsuranceFee            string `json:"item_insurance_fee"`
}

type TikTokAddress struct {
	FullAddress          string               `json:"full_address"`
	PhoneNumber          string               `json:"phone_number"`
	Name                 string               `json:"name"`
	FirstName            string               `json:"first_name"`
	LastName             string               `json:"last_name"`
	FirstNameLocalScript string               `json:"first_name_local_script"`
	LastNameLocalScript  string               `json:"last_name_local_script"`
	AddressDetail        string               `json:"address_detail"`
	AddressLine1         string               `json:"address_line1"`
	AddressLine2         string               `json:"address_line2"`
	AddressLine3         string               `json:"address_line3"`
	AddressLine4         string               `json:"address_line4"`
	DistrictInfo         []TikTokDistrictInfo `json:"district_info"`
	DeliveryPreferences  TikTokDeliveryPref   `json:"delivery_preferences"`
	PostalCode           string               `json:"postal_code"`
	RegionCode           string               `json:"region_code"`
	PostTown             string               `json:"post_town"`
}

type TikTokDistrictInfo struct {
	AddressLevelName string `json:"address_level_name"`
	AddressName      string `json:"address_name"`
	AddressLevel     string `json:"address_level"`
}

type TikTokDeliveryPref struct {
	DropOffLocation string `json:"drop_off_location"`
}

type TikTokLineItem struct {
	ID                   string                     `json:"id"`
	SkuID                string                     `json:"sku_id"`
	CombinedListingSkus  []TikTokCombinedListingSku `json:"combined_listing_skus"`
	DisplayStatus        string                     `json:"display_status"`
	ProductName          string                     `json:"product_name"`
	SellerSku            string                     `json:"seller_sku"`
	SkuImage             string                     `json:"sku_image"`
	SkuName              string                     `json:"sku_name"`
	ProductID            string                     `json:"product_id"`
	SalePrice            string                     `json:"sale_price"`
	PfandFee             string                     `json:"pfand_fee"`
	PlatformDiscount     string                     `json:"platform_discount"`
	SellerDiscount       string                     `json:"seller_discount"`
	SkuType              string                     `json:"sku_type"`
	CancelReason         string                     `json:"cancel_reason"`
	OriginalPrice        string                     `json:"original_price"`
	RtsTime              int64                      `json:"rts_time"`
	PackageStatus        string                     `json:"package_status"`
	Currency             string                     `json:"currency"`
	ShippingProviderName string                     `json:"shipping_provider_name"`
	CancelUser           string                     `json:"cancel_user"`
	ShippingProviderID   string                     `json:"shipping_provider_id"`
	IsGift               bool                       `json:"is_gift"`
	ItemTax              []TikTokItemTax            `json:"item_tax"`
	TrackingNumber       string                     `json:"tracking_number"`
	PackageID            string                     `json:"package_id"`
	RetailDeliveryFee    string                     `json:"retail_delivery_fee"`
	BuyerServiceFee      string                     `json:"buyer_service_fee"`
	SmallOrderFee        string                     `json:"small_order_fee"`
	HandlingDurationDays string                     `json:"handling_duration_days"`
	IsDangerousGood      bool                       `json:"is_dangerous_good"`
	NeedsPrescription    bool                       `json:"needs_prescription"`
	GiftRetailPrice      string                     `json:"gift_retail_price"`
}

type TikTokCombinedListingSku struct {
	SkuID     string `json:"sku_id"`
	SkuCount  int    `json:"sku_count"`
	ProductID string `json:"product_id"`
	SellerSku string `json:"seller_sku"`
}

type TikTokItemTax struct {
	TaxType   string `json:"tax_type"`
	TaxAmount string `json:"tax_amount"`
	TaxRate   string `json:"tax_rate"`
}

type TikTokHandling struct {
	Days string `json:"days"`
	Type string `json:"type"`
}

type TikTokOrderRequest struct {
	Code      int             `json:"code"`
	Data      TikTokOrderData `json:"data"`
	Message   string          `json:"message"`
	RequestID string          `json:"request_id"`
}

type TikTokOrderSearchFilter struct {
	CreateTimeGe int64 `json:"create_time_ge"`
	UpdateTimeGe int64 `json:"update_time_ge"`
}

type TikTokRefundResponse struct {
	Code      int              `json:"code"`
	Message   string           `json:"message"`
	RequestID string           `json:"request_id"`
	Data      TikTokRefundData `json:"data"`
}

type TikTokRefundData struct {
	ReturnOrders  []TikTokRefundOrder `json:"return_orders"`
	TotalCount    int                 `json:"total_count"`
	NextPageToken string              `json:"next_page_token"`
}

type TikTokRefundOrder struct {
	OrderID      string `json:"order_id"`
	ReturnID     string `json:"return_id"`
	ReturnStatus string `json:"return_status"`
	UpdateTime   int64  `json:"update_time"`
}
