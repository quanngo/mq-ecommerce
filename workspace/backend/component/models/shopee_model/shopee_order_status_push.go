package models

type OrderData struct {
	Items             []interface{} `json:"items"`
	OrderSN           string        `json:"ordersn"`
	Status            string        `json:"status"`
	CompletedScenario string        `json:"completed_scenario"`
	UpdateTime        int64         `json:"update_time"`
}

type OrderWebhook struct {
	Data      OrderData `json:"data"`
	ShopID    int64     `json:"shop_id"`
	Code      int       `json:"code"`
	Timestamp int64     `json:"timestamp"`
}
