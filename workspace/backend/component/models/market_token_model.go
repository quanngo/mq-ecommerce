package models

import (
	"main/utils"
	"time"
)

type MarketToken struct {
	ID            string        `db:"id"`
	MarketPlace   utils.Plaform `db:"market_place"`
	ShopID        string        `db:"shop_id"`
	AccessToken   string        `db:"access_token"`
	RefreshToken  string        `db:"refresh_token"`
	ShopReference *string       `db:"shop_reference"`
	ExpireTime    time.Time     `db:"expire_time"`
	UpdateAt      time.Time     `db:"update_at"`
	Enable        bool          `db:"enable"`
}
type UpdateDatabaseOrder struct {
	MarketPlace utils.Plaform
	ShopID      string
	OrderID     string
	Status      string
	CreatedDate time.Time
	UpdatedAt   time.Time
}
