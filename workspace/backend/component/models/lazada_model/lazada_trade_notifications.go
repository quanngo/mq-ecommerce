package models

type LazadaTradeNotifications struct {
	SellerID    string `json:"seller_id"`
	MessageType int64  `json:"message_type"`
	Data        struct {
		OrderStatus      string `json:"order_status"`
		TradeOrderID     string `json:"trade_order_id"`
		TradeOrderLineID string `json:"trade_order_line_id"`
		StatusUpdateTime int64  `json:"status_update_time"`
	} `json:"data"`
	Timestamp int64  `json:"timestamp"`
	Site      string `json:"site"`
}
