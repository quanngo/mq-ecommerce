package rest

import (
	"errors"
	"io"
	"main/component/api"
	"main/component/models"
	"main/component/services"
	"main/middleware"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ProductImageHandler serves /api/v1/product-images (EC-10): upload a product
// image straight to TikTok Shop or Lazada and list what was uploaded. Every
// route needs a session and orders.write; the upload also needs the frontend's
// origin.
type ProductImageHandler struct {
	Svc services.ProductImageService
}

// maxUploadBody leaves room for the multipart headers and text fields.
const maxUploadBody = services.MaxProductImageBytes + 1<<20

func (h *ProductImageHandler) Router(r *gin.RouterGroup, sameOrigin gin.HandlerFunc) {
	r.Use(middleware.RequirePermission(models.PermOrdersWrite))
	r.GET("/shops", h.Shops)
	r.GET("", h.List)
	r.POST("", sameOrigin, h.Upload)
}

// Shops godoc
// @Summary  TikTok and Lazada shops images can be uploaded to, and TikTok's image use cases
// @Tags     product-images
// @Produce  json
// @Success  200 {object} models.ImageShopsResponse
// @Router   /v1/product-images/shops [get]
func (h *ProductImageHandler) Shops(c *gin.Context) {
	res, err := h.Svc.Shops(c.Request.Context())
	if err != nil {
		respondAuthError(c, err)
		return
	}
	noStore(c)
	api.Ok(c, res)
}

// List godoc
// @Summary  Uploaded product images, newest first
// @Tags     product-images
// @Produce  json
// @Param    market_place query string false "Tiktok | Lazada"
// @Param    shop_id      query string false "shop id"
// @Param    page         query int    false "page, default 1"
// @Param    page_size    query int    false "1-100, default 24"
// @Success  200 {object} models.ProductImageListResponse
// @Router   /v1/product-images [get]
func (h *ProductImageHandler) List(c *gin.Context) {
	var q models.ProductImageListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		respondAuthError(c, services.ErrInvalidRequest)
		return
	}
	res, err := h.Svc.ListImages(c.Request.Context(), q)
	if err != nil {
		respondAuthError(c, err)
		return
	}
	noStore(c)
	api.Ok(c, res)
}

// Upload godoc
// @Summary  Upload one JPG/PNG (≤ 5 MB) to a TikTok or Lazada shop and keep the URL it returns
// @Tags     product-images
// @Accept   mpfd
// @Produce  json
// @Param    file         formData file   true  "JPG or PNG, at most 5 MB"
// @Param    market_place formData string true  "Tiktok | Lazada"
// @Param    shop_id      formData string true  "shop id (from /product-images/shops)"
// @Param    use_case     formData string false "TikTok only: MAIN_IMAGE (default), ATTRIBUTE_IMAGE, DESCRIPTION_IMAGE, CERTIFICATION_IMAGE, SIZE_CHART_IMAGE"
// @Success  200 {object} models.ProductImage
// @Router   /v1/product-images [post]
func (h *ProductImageHandler) Upload(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadBody)
	header, err := c.FormFile("file")
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			respondAuthError(c, services.ErrInvalidImage)
			return
		}
		respondAuthError(c, services.ErrInvalidRequest)
		return
	}
	if header.Size > services.MaxProductImageBytes {
		respondAuthError(c, services.ErrInvalidImage)
		return
	}
	f, err := header.Open()
	if err != nil {
		respondAuthError(c, services.ErrInvalidRequest)
		return
	}
	defer f.Close()
	content, err := io.ReadAll(io.LimitReader(f, services.MaxProductImageBytes+1))
	if err != nil {
		respondAuthError(c, services.ErrInvalidRequest)
		return
	}

	img, err := h.Svc.Upload(c.Request.Context(), middleware.CurrentUser(c), models.ProductImageUpload{
		MarketPlace: c.PostForm("market_place"),
		ShopID:      c.PostForm("shop_id"),
		UseCase:     c.PostForm("use_case"),
		File:        models.ImageFile{Name: header.Filename, Content: content},
	})
	if err != nil {
		respondAuthError(c, err)
		return
	}
	noStore(c)
	api.Ok(c, img)
}
