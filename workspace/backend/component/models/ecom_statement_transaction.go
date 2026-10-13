package models

import (
	"encoding/json"
	"main/utils"
	"time"
)

type EcomStatementTransaction struct {
	ID          int64           `json:"id" db:"id"`
	MarketPlace utils.Plaform   `json:"market_place" db:"market_place"`
	OrderID     string          `json:"order_id" db:"order_id"`
	RawData     json.RawMessage `json:"raw_data" db:"raw_data"` // jsonb
	CreatedAt   time.Time       `json:"created_at" db:"created_at"`
}
