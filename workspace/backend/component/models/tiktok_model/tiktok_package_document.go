package models

type TrackingResponse struct {
	Code      int          `json:"code"`
	Data      TrackingData `json:"data"`
	Message   string       `json:"message"`
	RequestId string       `json:"request_id"`
}

type TrackingData struct {
	DocURL         string `json:"doc_url"`
	TrackingNumber string `json:"tracking_number"`
}
