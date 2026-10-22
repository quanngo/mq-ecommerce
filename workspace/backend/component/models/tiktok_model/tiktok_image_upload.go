package models

// UploadImageResponse is the answer of POST /product/202309/images/upload.
type UploadImageResponse struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
	Data      struct {
		URI     string `json:"uri"`
		URL     string `json:"url"`
		Width   int    `json:"width"`
		Height  int    `json:"height"`
		UseCase string `json:"use_case"`
	} `json:"data"`
}
