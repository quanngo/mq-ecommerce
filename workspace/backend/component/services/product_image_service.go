package services

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/jpeg" // register decoders for image.DecodeConfig
	_ "image/png"
	"main/component/models"
	"main/utils"
	"net/http"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// MaxProductImageBytes is the largest image accepted. TikTok allows 5 MB; we
// apply the same limit to Lazada so one rule covers both.
const MaxProductImageBytes = 5 << 20

// ProductImageService uploads product images straight to TikTok Shop or
// Lazada for one shop and keeps only what the marketplace returns (EC-10).
// Callers must already have checked the permission (orders.write).
type ProductImageService interface {
	Shops(ctx context.Context) (*models.ImageShopsResponse, error)
	ListImages(ctx context.Context, q models.ProductImageListQuery) (*models.ProductImageListResponse, error)
	Upload(ctx context.Context, actor *models.SessionUser, in models.ProductImageUpload) (*models.ProductImage, error)
}

// imageUploader is the slice of middleware.ApiMiddleware this service needs.
type imageUploader interface {
	TiktokUploadImage(ctx context.Context, accessToken string, img models.ImageFile, useCase string) (*models.UploadedImage, error)
	LazadaUploadImage(ctx context.Context, accessToken string, img models.ImageFile) (*models.UploadedImage, error)
}

type marketTokenStore interface {
	GetMarketToken(ctx context.Context, marketPlace string) ([]*models.MarketToken, error)
}

type productImageStore interface {
	Insert(ctx context.Context, img models.ProductImage) (*models.ProductImage, error)
	List(ctx context.Context, f models.ProductImageFilter) ([]models.ProductImage, int, error)
}

type ProductImageServiceImpl struct {
	uploader imageUploader
	tokens   marketTokenStore
	images   productImageStore
}

func NewProductImageService(uploader imageUploader, tokens marketTokenStore, images productImageStore) ProductImageService {
	return &ProductImageServiceImpl{uploader: uploader, tokens: tokens, images: images}
}

// imageMarketPlaces are the marketplaces images can be uploaded to (Shopee later).
var imageMarketPlaces = []utils.Plaform{utils.TikTok, utils.Lazada}

func imageMarketPlace(s string) (utils.Plaform, bool) {
	for _, m := range imageMarketPlaces {
		if string(m) == s {
			return m, true
		}
	}
	return "", false
}

func (s *ProductImageServiceImpl) Shops(ctx context.Context) (*models.ImageShopsResponse, error) {
	shops := []models.ImageShop{}
	for _, m := range imageMarketPlaces {
		tokens, err := s.tokens.GetMarketToken(ctx, string(m))
		if err != nil {
			return nil, wrapAuth(ErrInternal, err)
		}
		for _, t := range tokens {
			if t.Enable && t.AccessToken != "" {
				shops = append(shops, models.ImageShop{MarketPlace: string(m), ShopID: t.ShopID})
			}
		}
	}
	return &models.ImageShopsResponse{Shops: shops, UseCases: models.TiktokImageUseCases}, nil
}

func (s *ProductImageServiceImpl) ListImages(ctx context.Context, q models.ProductImageListQuery) (*models.ProductImageListResponse, error) {
	f := models.ProductImageFilter{MarketPlace: strings.TrimSpace(q.MarketPlace), ShopID: strings.TrimSpace(q.ShopID)}
	if f.MarketPlace != "" {
		if _, ok := imageMarketPlace(f.MarketPlace); !ok {
			return nil, ErrInvalidRequest
		}
	}
	if len(f.ShopID) > 128 {
		return nil, ErrInvalidRequest
	}
	page, err := parseBoundedInt(q.Page, 1, 1, 1_000_000)
	if err != nil {
		return nil, ErrInvalidRequest
	}
	size, err := parseBoundedInt(q.PageSize, 24, 1, 100)
	if err != nil {
		return nil, ErrInvalidRequest
	}
	f.Limit, f.Offset = size, (page-1)*size
	images, total, err := s.images.List(ctx, f)
	if err != nil {
		return nil, wrapAuth(ErrInternal, err)
	}
	return &models.ProductImageListResponse{Images: images, Total: total, Page: page, PageSize: size}, nil
}

func (s *ProductImageServiceImpl) Upload(ctx context.Context, actor *models.SessionUser, in models.ProductImageUpload) (*models.ProductImage, error) {
	if actor == nil {
		return nil, ErrUnauthenticated
	}
	market, ok := imageMarketPlace(strings.TrimSpace(in.MarketPlace))
	if !ok || strings.TrimSpace(in.ShopID) == "" {
		return nil, ErrInvalidRequest
	}
	useCase, err := imageUseCase(market, in.UseCase)
	if err != nil {
		return nil, err
	}
	img, width, height, err := checkImage(in.File)
	if err != nil {
		return nil, err
	}
	token, err := s.shopToken(ctx, market, strings.TrimSpace(in.ShopID))
	if err != nil {
		return nil, err
	}

	var up *models.UploadedImage
	switch market {
	case utils.TikTok:
		up, err = s.uploader.TiktokUploadImage(ctx, token.AccessToken, img, useCase)
	case utils.Lazada:
		up, err = s.uploader.LazadaUploadImage(ctx, token.AccessToken, img)
	}
	if err != nil {
		return nil, wrapAuth(ErrMarketplace, fmt.Errorf("shop_id=%s file=%q: %w", token.ShopID, img.Name, err))
	}
	if up.Width > 0 && up.Height > 0 {
		width, height = up.Width, up.Height
	}

	actorID := actor.ID
	saved, err := s.images.Insert(ctx, models.ProductImage{
		MarketPlace: string(market), ShopID: token.ShopID, UseCase: useCase, URI: up.URI, URL: up.URL,
		Width: width, Height: height, FileName: img.Name, SizeBytes: len(img.Content), UploadedBy: &actorID,
	})
	if err != nil {
		// Already on the marketplace: log the URL so it isn't lost.
		return nil, wrapAuth(ErrInternal, fmt.Errorf("image uploaded to %s shop_id=%s (url=%s) but not recorded: %w", market, token.ShopID, up.URL, err))
	}
	utils.ShowInfoLogs("[ProductImageService] [Upload] by user_id=%d: %s shop_id=%s image_id=%d bytes=%d",
		actor.ID, market, token.ShopID, saved.ID, saved.SizeBytes)
	return saved, nil
}

// shopToken returns the shop's enabled token; NOT_FOUND for an unknown or disabled shop.
func (s *ProductImageServiceImpl) shopToken(ctx context.Context, market utils.Plaform, shopID string) (*models.MarketToken, error) {
	tokens, err := s.tokens.GetMarketToken(ctx, string(market))
	if err != nil {
		return nil, wrapAuth(ErrInternal, err)
	}
	for _, t := range tokens {
		if t.ShopID == shopID && t.Enable && t.AccessToken != "" {
			return t, nil
		}
	}
	return nil, ErrNotFound
}

func imageUseCase(market utils.Plaform, useCase string) (string, error) {
	useCase = strings.TrimSpace(useCase)
	if market != utils.TikTok {
		return "", nil // Lazada has no use case
	}
	if useCase == "" {
		return "MAIN_IMAGE", nil
	}
	for _, u := range models.TiktokImageUseCases {
		if u == useCase {
			return useCase, nil
		}
	}
	return "", ErrInvalidRequest
}

// checkImage accepts a non-empty JPG or PNG of at most MaxProductImageBytes
// whose header decodes, judged by its bytes rather than its name or the
// browser's Content-Type. It returns the cleaned file and its dimensions.
func checkImage(f models.ImageFile) (models.ImageFile, int, int, error) {
	if len(f.Content) == 0 || len(f.Content) > MaxProductImageBytes {
		return f, 0, 0, ErrInvalidImage
	}
	ct := http.DetectContentType(f.Content)
	if ct != "image/jpeg" && ct != "image/png" {
		return f, 0, 0, ErrInvalidImage
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(f.Content))
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 {
		return f, 0, 0, ErrInvalidImage
	}
	f.ContentType = ct
	f.Name = cleanFileName(f.Name, ct)
	return f, cfg.Width, cfg.Height, nil
}

// cleanFileName keeps the base name (no path), valid UTF-8, at most 255
// bytes, and falls back to "image.jpg"/"image.png".
func cleanFileName(name, contentType string) string {
	name = strings.TrimSpace(filepath.Base(strings.ReplaceAll(name, "\\", "/")))
	if !utf8.ValidString(name) || name == "." || name == "/" {
		name = ""
	}
	name = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, name)
	for len(name) > 255 {
		_, size := utf8.DecodeLastRuneInString(name)
		name = name[:len(name)-size]
	}
	if name == "" {
		if contentType == "image/png" {
			return "image.png"
		}
		return "image.jpg"
	}
	return name
}
