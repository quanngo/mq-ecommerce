package models

type Package struct {
	PackageId string `json:"package_id"`
}

type PackageReq struct {
	DocType  string    `json:"doc_type"`
	Packages []Package `json:"packages"`
}

// Toàn bộ request param khi gọi API /order/package/rts
type PrintAWBPayload struct {
	GetDocumentReq PackageReq `json:"readyToShipReq"`
}

type ApiPrintAWBResponse struct {
	Code   string `json:"code"`
	Result Result `json:"result"`
}

type Result struct {
	ErrorMsg  string `json:"error_msg"`
	Data      Data   `json:"data"`
	Success   bool   `json:"success"`
	ErrorCode string `json:"error_code"`
}

type Data struct {
	File    string `json:"file"`
	PDFUrl  string `json:"pdf_url"`
	DocType string `json:"doc_type"`
}

type PackRequest struct {
	PackOrderList    []PackOrder `json:"pack_order_list"`
	DeliveryType     string      `json:"delivery_type"`
	ShippingAllocate string      `json:"shipping_allocate_type"`
}

// Danh sách order cần pack
type PackOrder struct {
	OrderItemList []int64 `json:"order_item_list"`
	OrderID       int64   `json:"order_id"`
}
