package models

type ReadyToShipPackageReq struct {
	PackageID string `json:"package_id"`
}

type ReadyToShipReq struct {
	Packages []ReadyToShipPackage `json:"packages"`
	ShopID   string               `json:"shop_id"`
}

type ReadyToShipResponse struct {
	Code   string `json:"code"`
	Result Result `json:"result"`
}

type ReadyToShipResult struct {
	ErrorMsg  string          `json:"error_msg"`
	Data      ReadyToShipData `json:"data"`
	Success   bool            `json:"success"`
	ErrorCode string          `json:"error_code"`
}

type ReadyToShipData struct {
	Packages []ReadyToShipPackage `json:"packages"`
}

type ReadyToShipPackage struct {
	Msg         string `json:"msg"`
	ItemErrCode string `json:"item_err_code"`
	PackageID   string `json:"package_id"`
	Retry       string `json:"retry"`
}
