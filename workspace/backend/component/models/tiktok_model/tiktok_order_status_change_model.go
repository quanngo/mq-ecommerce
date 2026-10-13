package models

type TikTokOrderStatusChangeModel struct {
	Type           int64  `json:"type"`
	NotificationID string `json:"tts_notification_id"`
	ShopID         string `json:"shop_id"`
	Timestamp      int64  `json:"timestamp"`
	Data           struct {
		OrderID       string `json:"order_id"`
		OrderStatus   string `json:"order_status"`
		IsOnHoldOrder bool   `json:"is_on_hold_order"`
		UpdateTime    int64  `json:"update_time"`
	} `json:"data"`
}
