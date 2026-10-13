package models

type LogisticInventoryRequest struct {
	PartnerCode        string `json:"partnerCode"`
	PartnerProductCode string `json:"partnerProductCode"`
	Quantity           int    `json:"quantity"`

	ShopID      string
	MarketPlace string
}

type LogisticInventoryResponse struct {
	Code    int                      `json:"code"`
	Message string                   `json:"message"`
	Data    [][]LogisticProductStock `json:"data"`
}

type LogisticProductStock struct {
	WarehouseCode      string `json:"WarehouseCode"`
	PartnerProductCode string `json:"PartnerProductCode"`
	HasStock           bool   `json:"HasStock"`
	TotalQuantity      int    `json:"TotalQuantity"`
	AvailableQuantity  int    `json:"AvailableQuantity"`
	PendingQuantity    int    `json:"PendingQuantity"`
	ReservedQuantity   int    `json:"ReservedQuantity"`
	HoldQuantity       int    `json:"HoldQuantity"`
}
