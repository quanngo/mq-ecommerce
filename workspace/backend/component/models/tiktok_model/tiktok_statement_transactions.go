package models

// Root response
type OrderSettlementResponse struct {
	Code      int64                `json:"code"`
	Data      *OrderSettlementData `json:"data"`
	Message   string               `json:"message"`
	RequestID string               `json:"request_id"`
}

// Data node
type OrderSettlementData struct {
	OrderID            string           `json:"order_id"`
	OrderCreateTime    int64            `json:"order_create_time"`
	Currency           string           `json:"currency"`
	RevenueAmount      string           `json:"revenue_amount"`
	FeeAndTaxAmount    string           `json:"fee_and_tax_amount"`
	ShippingCostAmount string           `json:"shipping_cost_amount"`
	SettlementAmount   string           `json:"settlement_amount"`
	SkuTransactions    []SkuTransaction `json:"sku_transactions"`
	TotalCount         int              `json:"total_count"`
}

// One SKU transaction line
type SkuTransaction struct {
	SKUId                 string                 `json:"sku_id"`
	SKUName               string                 `json:"sku_name"`
	StatementID           string                 `json:"statement_id"`
	ProductName           string                 `json:"product_name"`
	Quantity              string                 `json:"quantity"`
	SettlementAmount      string                 `json:"settlement_amount"`
	RevenueAmount         string                 `json:"revenue_amount"`
	RevenueBreakdown      *RevenueBreakdown      `json:"revenue_breakdown"`
	ShippingCostAmount    string                 `json:"shipping_cost_amount"`
	ShippingCostBreakdown *ShippingCostBreakdown `json:"shipping_cost_breakdown"`
	FeeTaxAmount          string                 `json:"fee_tax_amount"`
	FeeTaxBreakdown       *FeeTaxBreakdown       `json:"fee_tax_breakdown"`
}

// Revenue breakdown
type RevenueBreakdown struct {
	SubtotalBeforeDiscountAmount       string `json:"subtotal_before_discount_amount"`
	SellerDiscountAmount               string `json:"seller_discount_amount"`
	RefundSubtotalBeforeDiscountAmount string `json:"refund_subtotal_before_discount_amount"`
	SellerDiscountRefundAmount         string `json:"seller_discount_refund_amount"`
	CodServiceFeeAmount                string `json:"cod_service_fee_amount"`
	RefundCodServiceFeeAmount          string `json:"refund_cod_service_fee_amount"`
}

// Shipping cost breakdown
type ShippingCostBreakdown struct {
	ActualShippingFeeAmount              string                  `json:"actual_shipping_fee_amount"`
	ShippingFeeDiscountAmount            string                  `json:"shipping_fee_discount_amount"`
	CustomerPaidShippingFeeAmount        string                  `json:"customer_paid_shipping_fee_amount"`
	ReturnShippingFeeAmount              string                  `json:"return_shipping_fee_amount"`
	ReplacementShippingFeeAmount         string                  `json:"replacement_shipping_fee_amount"`
	ExchangeShippingFeeAmount            string                  `json:"exchange_shipping_fee_amount"`
	SignatureConfirmationFeeAmount       string                  `json:"signature_confirmation_fee_amount"`
	ShippingInsuranceFeeAmount           string                  `json:"shipping_insurance_fee_amount"`
	FbtFulfillmentFeeReimbursementAmount string                  `json:"fbt_fulfillment_fee_reimbursement_amount"`
	ReturnShippingLabelFeeAmount         string                  `json:"return_shipping_label_fee_amount"`
	SellerSelfShippingServiceFeeAmount   string                  `json:"seller_self_shipping_service_fee_amount"`
	ReturnShippingFeePaidBuyerAmount     string                  `json:"return_shipping_fee_paid_buyer_amount"`
	FailedDeliverySubsidyAmount          string                  `json:"failed_delivery_subsidy_amount"`
	FbtFreeShippingFeeAmount             string                  `json:"fbt_free_shipping_fee_amount"`
	FreeReturnSubsidyAmount              string                  `json:"free_return_subsidy_amount"`
	SupplementaryComponent               *SupplementaryComponent `json:"supplementary_component"`
}

// Supplementary component inside shipping breakdown
type SupplementaryComponent struct {
	FbtFulfillmentFeeReimbursementAmount string `json:"fbt_fulfillment_fee_reimbursement_amount"`
	PlatformShippingFeeDiscountAmount    string `json:"platform_shipping_fee_discount_amount"`
	PromoShippingIncentiveAmount         string `json:"promo_shipping_incentive_amount"`
	ShippingFeeSubsidyAmount             string `json:"shipping_fee_subsidy_amount"`
	SellerShippingFeeDiscountAmount      string `json:"seller_shipping_fee_discount_amount"`
	FbtShippingCostAmount                string `json:"fbt_shipping_cost_amount"`
	CustomerShippingFeeOffsetAmount      string `json:"customer_shipping_fee_offset_amount"`
	FbmShippingCostAmount                string `json:"fbm_shipping_cost_amount"`
	FbtFulfillmentFeeAmount              string `json:"fbt_fulfillment_fee_amount"`
	ReturnRefundSubsidyAmount            string `json:"return_refund_subsidy_amount"`
	RefundedCustomerShippingFeeAmount    string `json:"refunded_customer_shipping_fee_amount"`
	ShippingFeeGuaranteeReimbursement    string `json:"shipping_fee_guarantee_reimbursement"`
}

// Fee & tax breakdown
type FeeTaxBreakdown struct {
	Fee *FeeBreakdown `json:"fee"`
	Tax *TaxBreakdown `json:"tax"`
}

// Fee section
type FeeBreakdown struct {
	PlatformCommissionAmount            string `json:"platform_commission_amount"`
	ReferralFeeAmount                   string `json:"referral_fee_amount"`
	RefundAdministrationFeeAmount       string `json:"refund_administration_fee_amount"`
	TransactionFeeAmount                string `json:"transaction_fee_amount"`
	CreditCardHandlingFeeAmount         string `json:"credit_card_handling_fee_amount"`
	AffiliateCommissionAmount           string `json:"affiliate_commission_amount"`
	AffiliateCommissionAmountBeforePit  string `json:"affiliate_commission_amount_before_pit"`
	AffiliatePartnerCommissionAmount    string `json:"affiliate_partner_commission_amount"`
	AffiliateAdsCommissionAmount        string `json:"affiliate_ads_commission_amount"`
	SfpServiceFeeAmount                 string `json:"sfp_service_fee_amount"`
	LiveSpecialsFeeAmount               string `json:"live_specials_fee_amount"`
	BonusCashbackServiceFeeAmount       string `json:"bonus_cashback_service_fee_amount"`
	MallServiceFeeAmount                string `json:"mall_service_fee_amount"`
	VoucherXtraServiceFeeAmount         string `json:"voucher_xtra_service_fee_amount"`
	FlashSalesServiceFeeAmount          string `json:"flash_sales_service_fee_amount"`
	CofundedPromotionServiceFeeAmount   string `json:"cofunded_promotion_service_fee_amount"`
	PreOrderServiceFeeAmount            string `json:"pre_order_service_fee_amount"`
	TspCommissionAmount                 string `json:"tsp_commission_amount"`
	DtHandlingFeeAmount                 string `json:"dt_handling_fee_amount"`
	EprPobServiceFeeAmount              string `json:"epr_pob_service_fee_amount"`
	FeePerItemSoldAmount                string `json:"fee_per_item_sold_amount"`
	SellerPaylaterHandlingFeeAmount     string `json:"seller_paylater_handling_fee_amount"`
	CofundedCreatorBonusAmount          string `json:"cofunded_creator_bonus_amount"`
	DynamicCommissionAmount             string `json:"dynamic_commission_amount"`
	ExternalAffiliateMarketingFeeAmount string `json:"external_affiliate_marketing_fee_amount"`
	VnFixInfrastructureFee              string `json:"vn_fix_infrastructure_fee"`
	AffiliateCommissionDeposit          string `json:"affiliate_commission_deposit"`
	AffiliateCommissionRelease          string `json:"affiliate_commission_release"`
	TapShopAdsCommission                string `json:"tap_shop_ads_commission"`
	ShippingFeeGuaranteeServiceFee      string `json:"shipping_fee_guarantee_service_fee"`
	InstallationServiceFee              string `json:"installation_service_fee"`
	CampaignResourceFee                 string `json:"campaign_resource_fee"`
}

// Tax section
type TaxBreakdown struct {
	VatAmount              string `json:"vat_amount"`
	ImportVatAmount        string `json:"import_vat_amount"`
	CustomsDutyAmount      string `json:"customs_duty_amount"`
	CustomsClearanceAmount string `json:"customs_clearance_amount"`
	SstAmount              string `json:"sst_amount"`
	GstAmount              string `json:"gst_amount"`
	IvaAmount              string `json:"iva_amount"`
	IsrAmount              string `json:"isr_amount"`
	AntiDumpingDutyAmount  string `json:"anti_dumping_duty_amount"`
	LocalVatAmount         string `json:"local_vat_amount"`
	PitAmount              string `json:"pit_amount"`
}
