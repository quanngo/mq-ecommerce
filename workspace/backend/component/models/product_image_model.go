package models

import "time"

// ---------- product images (EC-10) ----------

// TikTok image use cases accepted by /product/202309/images/upload.
var TiktokImageUseCases = []string{"MAIN_IMAGE", "ATTRIBUTE_IMAGE", "DESCRIPTION_IMAGE", "CERTIFICATION_IMAGE", "SIZE_CHART_IMAGE"}

// ImageFile is an image to send to a marketplace. It is only held in memory:
// the file itself is never stored by this service.
type ImageFile struct {
	Name        string
	ContentType string // image/jpeg | image/png, sniffed from the bytes
	Content     []byte
}

// UploadedImage is what a marketplace returns for an uploaded image.
type UploadedImage struct {
	URI    string // TikTok only
	URL    string
	Width  int
	Height int
}

// ProductImage maps public.ecom_product_image (+ the uploader's email).
type ProductImage struct {
	ID              int64     `db:"id" json:"id"`
	MarketPlace     string    `db:"market_place" json:"market_place"`
	ShopID          string    `db:"shop_id" json:"shop_id"`
	UseCase         string    `db:"use_case" json:"use_case"`
	URI             string    `db:"uri" json:"uri"`
	URL             string    `db:"url" json:"url"`
	Width           int       `db:"width" json:"width"`
	Height          int       `db:"height" json:"height"`
	FileName        string    `db:"file_name" json:"file_name"`
	SizeBytes       int       `db:"size_bytes" json:"size_bytes"`
	UploadedBy      *int64    `db:"uploaded_by" json:"-"`
	UploadedByEmail *string   `db:"uploaded_by_email" json:"uploaded_by"`
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
}

// ProductImageListQuery is GET /product-images; strings, validated by the service.
type ProductImageListQuery struct {
	MarketPlace string `form:"market_place"`
	ShopID      string `form:"shop_id"`
	Page        string `form:"page"`
	PageSize    string `form:"page_size"`
}

type ProductImageFilter struct {
	MarketPlace string
	ShopID      string
	Limit       int
	Offset      int
}

type ProductImageListResponse struct {
	Images   []ProductImage `json:"images"`
	Total    int            `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"page_size"`
}

// ProductImageUpload is POST /product-images after the handler read the form.
type ProductImageUpload struct {
	MarketPlace string
	ShopID      string
	UseCase     string // TikTok only; "" = MAIN_IMAGE
	File        ImageFile
}

// ImageShop is a shop images can be uploaded to (an enabled market_token row).
type ImageShop struct {
	MarketPlace string `json:"market_place"`
	ShopID      string `json:"shop_id"`
}

type ImageShopsResponse struct {
	Shops    []ImageShop `json:"shops"`
	UseCases []string    `json:"tiktok_use_cases"`
}
