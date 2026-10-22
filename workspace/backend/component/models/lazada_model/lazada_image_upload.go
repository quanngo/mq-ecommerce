package models

// UploadImageResponse is the answer of POST /image/upload. On failure Lazada
// sends a non-"0" code with a message and no data.
type UploadImageResponse struct {
	Code      string `json:"code"`
	Type      string `json:"type"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
	Data      struct {
		Image struct {
			URL      string `json:"url"`
			HashCode string `json:"hash_code"`
		} `json:"image"`
	} `json:"data"`
}
