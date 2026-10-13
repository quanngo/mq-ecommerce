package models

type TikTokOrderDetailModel struct {
	ID                                 string           `json:"id"`
	CancellationInitiator              string           `json:"cancellation_initiator"`
	ShippingProvider                   string           `json:"shipping_provider"`
	ShippingProviderID                 string           `json:"shipping_provider_id"`
	UserID                             string           `json:"user_id"`
	Status                             string           `json:"status"`
	RtsTime                            int64            `json:"rts_time"`
	Payment                            Payment          `json:"payment"`
	RecipientAddress                   RecipientAddress `json:"recipient_address"`
	BuyerMessage                       string           `json:"buyer_message"`
	CreateTime                         int64            `json:"create_time"`
	TrackingNumber                     string           `json:"tracking_number"`
	CancelReason                       string           `json:"cancel_reason"`
	RtsSlaTime                         int64            `json:"rts_sla_time"`
	PaidTime                           int64            `json:"paid_time"`
	FulfillmentType                    string           `json:"fulfillment_type"`
	SellerNote                         string           `json:"seller_note"`
	TtsSlaTime                         int64            `json:"tts_sla_time"`
	CancelOrderSlaTime                 int64            `json:"cancel_order_sla_time"`
	UpdateTime                         int64            `json:"update_time"`
	Packages                           []Package        `json:"packages"`
	DeliveryType                       string           `json:"delivery_type"`
	IsSampleOrder                      bool             `json:"is_sample_order"`
	WarehouseID                        string           `json:"warehouse_id"`
	SplitOrCombineTag                  string           `json:"split_or_combine_tag"`
	HasUpdatedRecipientAddress         bool             `json:"has_updated_recipient_address"`
	Cpf                                string           `json:"cpf"`
	DeliveryOptionID                   string           `json:"delivery_option_id"`
	DeliverySlaTime                    int64            `json:"delivery_sla_time"`
	PaymentMethodName                  string           `json:"payment_method_name"`
	PaymentCardType                    string           `json:"payment_card_type"`
	ChannelEntityNationalRegistryID    string           `json:"channel_entity_national_registry_id"`
	PaymentMethodCode                  string           `json:"payment_method_code"`
	PaymentAuthCode                    string           `json:"payment_auth_code"`
	ShippingDueTime                    int64            `json:"shipping_due_time"`
	LineItems                          []LineItem       `json:"line_items"`
	ShippingType                       string           `json:"shipping_type"`
	BuyerEmail                         string           `json:"buyer_email"`
	DeliveryTime                       int64            `json:"delivery_time"`
	NeedUploadInvoice                  string           `json:"need_upload_invoice"`
	IsCOD                              bool             `json:"is_cod"`
	RequestCancelTime                  int64            `json:"request_cancel_time"`
	DeliveryOptionRequiredDeliveryTime int64            `json:"delivery_option_required_delivery_time"`
	DeliveryOptionName                 string           `json:"delivery_option_name"`
	IsBuyerRequestCancel               bool             `json:"is_buyer_request_cancel"`
	DeliveryDueTime                    int64            `json:"delivery_due_time"`
	CollectionTime                     int64            `json:"collection_time"`
	IsOnHoldOrder                      bool             `json:"is_on_hold_order"`
	CancelTime                         int64            `json:"cancel_time"`
	IsReplacementOrder                 bool             `json:"is_replacement_order"`
	ReplacedOrderID                    string           `json:"replaced_order_id"`
	CollectionDueTime                  int64            `json:"collection_due_time"`
	PickUpCutOffTime                   int64            `json:"pick_up_cut_off_time"`
	FastDispatchSlaTime                int64            `json:"fast_dispatch_sla_time"`
	CommercePlatform                   string           `json:"commerce_platform"`
	OrderType                          string           `json:"order_type"`
	ReleaseDate                        int64            `json:"release_date"`
	HandlingDuration                   HandlingDuration `json:"handling_duration"`
	AutoCombineGroupID                 string           `json:"auto_combine_group_id"`
	CpfName                            string           `json:"cpf_name"`
	IsExchangeOrder                    bool             `json:"is_exchange_order"`
	ExchangeSourceOrderID              string           `json:"exchange_source_order_id"`
	ConsultationID                     string           `json:"consultation_id"`
	FastDeliveryProgram                string           `json:"fast_delivery_program"`
}

type Payment struct {
	Id int `json:"id"`

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

type RecipientAddress struct {
	FullAddress    string        `json:"full_address"`
	PhoneNumber    string        `json:"phone_number"`
	Name           string        `json:"name"`
	FirstName      string        `json:"first_name"`
	LastName       string        `json:"last_name"`
	FirstNameLocal string        `json:"first_name_local_script"`
	LastNameLocal  string        `json:"last_name_local_script"`
	RegionCode     string        `json:"region_code"`
	PostalCode     string        `json:"postal_code"`
	PostTown       string        `json:"post_town"`
	AddressLine1   string        `json:"address_line1"`
	AddressLine2   string        `json:"address_line2"`
	AddressLine3   string        `json:"address_line3"`
	AddressLine4   string        `json:"address_line4"`
	DistrictInfo   []District    `json:"district_info"`
	DeliveryPrefs  DeliveryPrefs `json:"delivery_preferences"`
	AddressDetail  string        `json:"address_detail"`
}

type District struct {
	AddressLevelName string `json:"address_level_name"`
	AddressName      string `json:"address_name"`
	AddressLevel     string `json:"address_level"`
}

type DeliveryPrefs struct {
	DropOffLocation string `json:"drop_off_location"`
}

type Package struct {
	ID string `json:"id"`
}

type LineItem struct {
	ID                  string        `json:"id"`
	SkuID               string        `json:"sku_id"`
	CombinedListingSkus []CombinedSku `json:"combined_listing_skus"`
	ProductID           string        `json:"product_id"`
	ProductName         string        `json:"product_name"`
	SkuName             string        `json:"sku_name"`
	SkuImage            string        `json:"sku_image"`
	OriginalPrice       string        `json:"original_price"`
	SalePrice           string        `json:"sale_price"`
	PlatformDiscount    string        `json:"platform_discount"`
	DisplayStatus       string        `json:"display_status"`
	CancelUser          string        `json:"cancel_user"`
	SkuType             string        `json:"sku_type"`
	SellerSku           string        `json:"seller_sku"`
	ShippingProviderID  string        `json:"shipping_provider_id"`
	SellerDiscount      string        `json:"seller_discount"`
	Currency            string        `json:"currency"`
	PackageID           string        `json:"package_id"`
	RtsTime             int64         `json:"rts_time"`
	ItemTax             []ItemTax     `json:"item_tax"`
	PackageStatus       string        `json:"package_status"`
	ShippingProvider    string        `json:"shipping_provider_name"`
	IsGift              bool          `json:"is_gift"`
	CancelReason        string        `json:"cancel_reason"`
	SmallOrderFee       string        `json:"small_order_fee"`
	RetailDeliveryFee   string        `json:"retail_delivery_fee"`
	TrackingNumber      string        `json:"tracking_number"`
	BuyerServiceFee     string        `json:"buyer_service_fee"`
	HandlingDuration    string        `json:"handling_duration_days"`
	IsDangerousGood     bool          `json:"is_dangerous_good"`
	NeedsPrescription   bool          `json:"needs_prescription"`
}

type CombinedSku struct {
	SkuID     string `json:"sku_id"`
	SkuCount  int    `json:"sku_count"`
	ProductID string `json:"product_id"`
	SellerSku string `json:"seller_sku"`
}

type ItemTax struct {
	TaxType   string `json:"tax_type"`
	TaxAmount string `json:"tax_amount"`
	TaxRate   string `json:"tax_rate"`
}

type HandlingDuration struct {
	Days string `json:"days"`
	Type string `json:"type"`
}
