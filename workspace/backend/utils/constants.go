package utils

const (
	DATE_TIME_FORMAT_YYYY_MM_DDT_HH_MM_SS     string = "2006-01-02T15:04:05"
	DATE_TIME_FORMAT_YYYY_MM_DD               string = "2006-01-02"
	DATE_TIME_FORMAT_YYYY_MM_DDT_HH_MM_SS_GMT string = "2006-01-02 15:04:05 -0700"
)
const (
	TimezoneVietnam = "Asia/Ho_Chi_Minh"
)

type Plaform string

const (
	Lazada Plaform = "Lazada"
	TikTok Plaform = "Tiktok"
	Shopee Plaform = "Shopee"
)
const (
	NTFS string = "NTFS"
	TFS  string = "TFS"
)
const Dropship string = "dropship"
const IllegalAccessToken string = "IllegalAccessToken"

const (
	PAYMENT_METHOD_COD       = 1
	PAYMENT_METHOD_ECOMMERCE = 8
)

type LeadStatus int

const (
	LeadPending LeadStatus = 1 // (Out of stock)
	LeadCreated LeadStatus = 2
	LeadRefund  LeadStatus = 3
	LeadCancel  LeadStatus = 4
)

type SaleOrderStatus int

const (
	SaleOrderNew                     SaleOrderStatus = 41
	SaleOrderPending                 SaleOrderStatus = 42
	SaleOrderValidated               SaleOrderStatus = 43
	SaleOrderCancel                  SaleOrderStatus = 44
	SaleOrderSuccess                 SaleOrderStatus = 45
	SaleOrderUnassigned              SaleOrderStatus = 46
	SaleOrderDelayValidate           SaleOrderStatus = 357
	SaleOrderDeliveryOrderCreateFail SaleOrderStatus = 48
	SaleOrderCompleted               SaleOrderStatus = 49
	SaleOrderRefund                  SaleOrderStatus = 50
)

type DeliveryOrderStatus int

const (
	DeliveryOrderStatusNew                   DeliveryOrderStatus = 51
	DeliveryOrderStatusReadyToPick           DeliveryOrderStatus = 52 // packed
	DeliveryOrderStatusCancel                DeliveryOrderStatus = 53
	DeliveryOrderStatusPicking               DeliveryOrderStatus = 54
	DeliveryOrderStatusInTransit             DeliveryOrderStatus = 55
	DeliveryOrderStatusReturn                DeliveryOrderStatus = 56 // returning
	DeliveryOrderStatusReject                DeliveryOrderStatus = 57
	DeliveryOrderStatusInPreparation         DeliveryOrderStatus = 58
	DeliveryOrderStatusDelivered             DeliveryOrderStatus = 59
	DeliveryOrderStatusPaid                  DeliveryOrderStatus = 60
	DeliveryOrderStatusReturned              DeliveryOrderStatus = 61
	DeliveryOrderStatusFailed                DeliveryOrderStatus = 62 // delivering
	DeliveryOrderStatusPickedUp              DeliveryOrderStatus = 63
	DeliveryOrderStatusPending               DeliveryOrderStatus = 65
	DeliveryOrderStatusWaitCreatingSuccess   DeliveryOrderStatus = 72
	DeliveryOrderStatusWaitingToRetry        DeliveryOrderStatus = 73
	DeliveryOrderStatusWaitingToFinishCreate DeliveryOrderStatus = 74
)
