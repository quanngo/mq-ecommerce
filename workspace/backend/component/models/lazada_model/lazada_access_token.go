package models

import "time"

type CountryUserInfo struct {
	Country   string `json:"country"`
	UserID    string `json:"user_id"`
	SellerID  string `json:"seller_id"`
	ShortCode string `json:"short_code"`
}

type AuthResponse struct {
	AccessToken      string            `json:"access_token"`
	Country          string            `json:"country"`
	RefreshToken     string            `json:"refresh_token"`
	AccountID        string            `json:"account_id"`
	Code             string            `json:"code"`
	AccountPlatform  string            `json:"account_platform"`
	RefreshExpiresIn string            `json:"refresh_expires_in"`
	CountryUserInfo  []CountryUserInfo `json:"country_user_info"`
	ExpiresIn        string            `json:"expires_in"`
	RequestID        string            `json:"request_id"`
	Account          string            `json:"account"`
}

type GenerateAccessTokenRequest struct {
	AppKey     string `json:"app_key"`
	Timestamp  string `json:"timestamp"`
	SignMethod string `json:"sign_method"`
	Sign       string `json:"sign"`
	Code       string `json:"code"`
}

type LazadaTokenTable struct {
	MarketPlace  string    `db:"market_place" json:"market_place"`
	ShopID       string    `db:"shop_id" json:"shop_id"`
	AccessToken  string    `db:"access_token" json:"access_token"`
	RefreshToken string    `db:"refresh_token" json:"refresh_token"`
	ExpireTime   time.Time `db:"expire_time" json:"expire_time"`
	UpdateAt     time.Time `db:"update_at" json:"update_at"`
}
