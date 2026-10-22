package services

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"main/component/models"
	"strings"
	"testing"
)

func pngBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func jpegBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, image.NewRGBA(image.Rect(0, 0, w, h)), nil); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

type fakeUploader struct {
	calls   []string // "<platform> <token> <use case> <file>"
	err     error
	tiktok  models.UploadedImage
	lazada  models.UploadedImage
	gotFile models.ImageFile
}

func (f *fakeUploader) TiktokUploadImage(_ context.Context, token string, img models.ImageFile, useCase string) (*models.UploadedImage, error) {
	f.calls = append(f.calls, "Tiktok "+token+" "+useCase+" "+img.Name)
	f.gotFile = img
	if f.err != nil {
		return nil, f.err
	}
	up := f.tiktok
	return &up, nil
}

func (f *fakeUploader) LazadaUploadImage(_ context.Context, token string, img models.ImageFile) (*models.UploadedImage, error) {
	f.calls = append(f.calls, "Lazada "+token+" "+img.Name)
	f.gotFile = img
	if f.err != nil {
		return nil, f.err
	}
	up := f.lazada
	return &up, nil
}

type fakeTokens struct {
	byMarket map[string][]*models.MarketToken
	err      error
}

func (f *fakeTokens) GetMarketToken(_ context.Context, m string) ([]*models.MarketToken, error) {
	return f.byMarket[m], f.err
}

type fakeImageStore struct {
	saved     []models.ProductImage
	insertErr error
	gotFilter models.ProductImageFilter
}

func (f *fakeImageStore) Insert(_ context.Context, img models.ProductImage) (*models.ProductImage, error) {
	if f.insertErr != nil {
		return nil, f.insertErr
	}
	img.ID = int64(len(f.saved) + 1)
	f.saved = append(f.saved, img)
	return &img, nil
}

func (f *fakeImageStore) List(_ context.Context, filter models.ProductImageFilter) ([]models.ProductImage, int, error) {
	f.gotFilter = filter
	return f.saved, len(f.saved), nil
}

func imageHarness() (*ProductImageServiceImpl, *fakeUploader, *fakeImageStore) {
	up := &fakeUploader{
		tiktok: models.UploadedImage{URI: "tos/abc", URL: "https://tiktok.cdn/abc.jpg", Width: 640, Height: 480},
		lazada: models.UploadedImage{URL: "https://lazada.cdn/abc.png"},
	}
	tokens := &fakeTokens{byMarket: map[string][]*models.MarketToken{
		"Tiktok": {
			{ShopID: "tt-1", AccessToken: "tt-token", Enable: true},
			{ShopID: "tt-off", AccessToken: "x", Enable: false},
		},
		"Lazada": {
			{ShopID: "lz-1", AccessToken: "lz-token", Enable: true},
			{ShopID: "lz-empty", AccessToken: "", Enable: true},
		},
	}}
	store := &fakeImageStore{}
	return NewProductImageService(up, tokens, store).(*ProductImageServiceImpl), up, store
}

var imageActor = &models.SessionUser{AppUser: models.AppUser{ID: 7}}

func TestUploadToTiktok(t *testing.T) {
	svc, up, store := imageHarness()
	content := pngBytes(t, 32, 16)
	img, err := svc.Upload(context.Background(), imageActor, models.ProductImageUpload{
		MarketPlace: "Tiktok", ShopID: " tt-1 ", UseCase: "DESCRIPTION_IMAGE",
		File: models.ImageFile{Name: `C:\fakepath\Serum 01.png`, Content: content},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(up.calls) != 1 || up.calls[0] != "Tiktok tt-token DESCRIPTION_IMAGE Serum 01.png" || up.gotFile.ContentType != "image/png" {
		t.Errorf("calls = %v, file = %+v", up.calls, up.gotFile.ContentType)
	}
	want := models.ProductImage{ID: 1, MarketPlace: "Tiktok", ShopID: "tt-1", UseCase: "DESCRIPTION_IMAGE", URI: "tos/abc",
		URL: "https://tiktok.cdn/abc.jpg", Width: 640, Height: 480, FileName: "Serum 01.png", SizeBytes: len(content)}
	got := *img
	if got.UploadedBy == nil || *got.UploadedBy != 7 {
		t.Errorf("uploaded_by = %v", got.UploadedBy)
	}
	got.UploadedBy = nil
	if got != want {
		t.Errorf("saved = %+v\nwant    %+v", got, want)
	}
	if len(store.saved) != 1 {
		t.Errorf("stored %d rows", len(store.saved))
	}
}

func TestUploadToLazadaUsesOwnDimensionsAndNoUseCase(t *testing.T) {
	svc, up, _ := imageHarness()
	img, err := svc.Upload(context.Background(), imageActor, models.ProductImageUpload{
		MarketPlace: "Lazada", ShopID: "lz-1", UseCase: "MAIN_IMAGE",
		File: models.ImageFile{Name: "a.jpg", Content: jpegBytes(t, 20, 10)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if up.calls[0] != "Lazada lz-token a.jpg" {
		t.Errorf("calls = %v", up.calls)
	}
	if img.UseCase != "" || img.URI != "" || img.URL != "https://lazada.cdn/abc.png" || img.Width != 20 || img.Height != 10 {
		t.Errorf("saved = %+v", img)
	}
}

func TestUploadTiktokDefaultUseCase(t *testing.T) {
	svc, up, _ := imageHarness()
	if _, err := svc.Upload(context.Background(), imageActor, models.ProductImageUpload{
		MarketPlace: "Tiktok", ShopID: "tt-1", File: models.ImageFile{Name: "x.png", Content: pngBytes(t, 4, 4)},
	}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(up.calls[0], " MAIN_IMAGE ") {
		t.Errorf("calls = %v", up.calls)
	}
}

func TestUploadRejectsBeforeCallingTheMarketplace(t *testing.T) {
	big := append(pngBytes(t, 4, 4), make([]byte, MaxProductImageBytes)...)
	gif := []byte("GIF89a\x01\x00\x01\x00\x80\x00\x00\xff\xff\xff\x00\x00\x00!\xf9\x04\x01\x00\x00\x00\x00,\x00\x00\x00\x00\x01\x00\x01\x00\x00\x02\x02D\x01\x00;")
	brokenPNG := append([]byte("\x89PNG\r\n\x1a\n"), []byte("not really a png")...)
	ok := func() models.ImageFile { return models.ImageFile{Name: "x.png", Content: pngBytes(t, 4, 4)} }

	cases := []struct {
		name string
		in   models.ProductImageUpload
		want error
	}{
		{"shopee not supported", models.ProductImageUpload{MarketPlace: "Shopee", ShopID: "s", File: ok()}, ErrInvalidRequest},
		{"unknown marketplace", models.ProductImageUpload{MarketPlace: "tiktok", ShopID: "tt-1", File: ok()}, ErrInvalidRequest},
		{"no shop", models.ProductImageUpload{MarketPlace: "Tiktok", ShopID: " ", File: ok()}, ErrInvalidRequest},
		{"bad use case", models.ProductImageUpload{MarketPlace: "Tiktok", ShopID: "tt-1", UseCase: "BANNER", File: ok()}, ErrInvalidRequest},
		{"empty file", models.ProductImageUpload{MarketPlace: "Tiktok", ShopID: "tt-1", File: models.ImageFile{Name: "x.png"}}, ErrInvalidImage},
		{"too large", models.ProductImageUpload{MarketPlace: "Tiktok", ShopID: "tt-1", File: models.ImageFile{Name: "x.png", Content: big}}, ErrInvalidImage},
		{"gif", models.ProductImageUpload{MarketPlace: "Lazada", ShopID: "lz-1", File: models.ImageFile{Name: "x.gif", Content: gif}}, ErrInvalidImage},
		{"text named .jpg", models.ProductImageUpload{MarketPlace: "Lazada", ShopID: "lz-1", File: models.ImageFile{Name: "x.jpg", Content: []byte("hello")}}, ErrInvalidImage},
		{"corrupt png", models.ProductImageUpload{MarketPlace: "Lazada", ShopID: "lz-1", File: models.ImageFile{Name: "x.png", Content: brokenPNG}}, ErrInvalidImage},
		{"unknown shop", models.ProductImageUpload{MarketPlace: "Tiktok", ShopID: "nope", File: ok()}, ErrNotFound},
		{"disabled shop", models.ProductImageUpload{MarketPlace: "Tiktok", ShopID: "tt-off", File: ok()}, ErrNotFound},
		{"shop without token", models.ProductImageUpload{MarketPlace: "Lazada", ShopID: "lz-empty", File: ok()}, ErrNotFound},
		{"other marketplace's shop", models.ProductImageUpload{MarketPlace: "Lazada", ShopID: "tt-1", File: ok()}, ErrNotFound},
	}
	for _, tc := range cases {
		svc, up, store := imageHarness()
		_, err := svc.Upload(context.Background(), imageActor, tc.in)
		if !errors.Is(err, tc.want) {
			t.Errorf("%s: err = %v, want %v", tc.name, err, tc.want)
		}
		if len(up.calls) != 0 || len(store.saved) != 0 {
			t.Errorf("%s: called %v / stored %d", tc.name, up.calls, len(store.saved))
		}
	}

	svc, _, _ := imageHarness()
	if _, err := svc.Upload(context.Background(), nil, models.ProductImageUpload{MarketPlace: "Tiktok", ShopID: "tt-1", File: ok()}); !errors.Is(err, ErrUnauthenticated) {
		t.Errorf("nil actor: %v", err)
	}
}

func TestUploadMarketplaceAndStoreFailures(t *testing.T) {
	svc, up, store := imageHarness()
	up.err = errors.New("code=12052700 message=too small")
	_, err := svc.Upload(context.Background(), imageActor, models.ProductImageUpload{MarketPlace: "Tiktok", ShopID: "tt-1", File: models.ImageFile{Name: "x.png", Content: pngBytes(t, 4, 4)}})
	if !errors.Is(err, ErrMarketplace) || len(store.saved) != 0 {
		t.Errorf("marketplace failure: %v, stored %d", err, len(store.saved))
	}
	if !strings.Contains(err.Error(), "12052700") {
		t.Errorf("the cause should reach the logs: %v", err)
	}

	svc, _, store = imageHarness()
	store.insertErr = errors.New("db down")
	_, err = svc.Upload(context.Background(), imageActor, models.ProductImageUpload{MarketPlace: "Lazada", ShopID: "lz-1", File: models.ImageFile{Name: "x.png", Content: pngBytes(t, 4, 4)}})
	if !errors.Is(err, ErrInternal) || !strings.Contains(err.Error(), "https://lazada.cdn/abc.png") {
		t.Errorf("store failure should log the marketplace URL: %v", err)
	}
}

func TestImageShops(t *testing.T) {
	svc, _, _ := imageHarness()
	res, err := svc.Shops(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := []models.ImageShop{{MarketPlace: "Tiktok", ShopID: "tt-1"}, {MarketPlace: "Lazada", ShopID: "lz-1"}}
	if len(res.Shops) != len(want) || res.Shops[0] != want[0] || res.Shops[1] != want[1] {
		t.Errorf("shops = %+v", res.Shops)
	}
	if len(res.UseCases) != 5 || res.UseCases[0] != "MAIN_IMAGE" {
		t.Errorf("use cases = %v", res.UseCases)
	}

	svc.tokens = &fakeTokens{err: errors.New("db down")}
	if _, err := svc.Shops(context.Background()); !errors.Is(err, ErrInternal) {
		t.Errorf("err = %v", err)
	}
}

func TestListImagesValidation(t *testing.T) {
	svc, _, store := imageHarness()
	res, err := svc.ListImages(context.Background(), models.ProductImageListQuery{MarketPlace: "Lazada", ShopID: "lz-1", Page: "3", PageSize: "10"})
	if err != nil {
		t.Fatal(err)
	}
	if store.gotFilter != (models.ProductImageFilter{MarketPlace: "Lazada", ShopID: "lz-1", Limit: 10, Offset: 20}) || res.Page != 3 || res.PageSize != 10 {
		t.Errorf("filter = %+v, res = %+v", store.gotFilter, res)
	}
	if _, err := svc.ListImages(context.Background(), models.ProductImageListQuery{}); err != nil || store.gotFilter.Limit != 24 || store.gotFilter.Offset != 0 {
		t.Errorf("defaults: %v %+v", err, store.gotFilter)
	}
	for _, q := range []models.ProductImageListQuery{
		{MarketPlace: "Shopee"}, {Page: "0"}, {Page: "x"}, {PageSize: "101"}, {ShopID: strings.Repeat("s", 129)},
	} {
		if _, err := svc.ListImages(context.Background(), q); !errors.Is(err, ErrInvalidRequest) {
			t.Errorf("%+v: err = %v", q, err)
		}
	}
}

func TestCleanFileName(t *testing.T) {
	cases := map[string]string{
		"serum.png":                "serum.png",
		`C:\Users\a\Desktop\b.jpg`: "b.jpg",
		"../../etc/passwd.png":     "passwd.png",
		"":                         "image.png",
		"\x00\x07bad.png":          "bad.png",
		"\xff\xfe.png":             "image.png",
		strings.Repeat("ả", 200):   strings.Repeat("ả", 85),
	}
	for in, want := range cases {
		if got := cleanFileName(in, "image/png"); got != want {
			t.Errorf("cleanFileName(%q) = %q, want %q", in, got, want)
		}
	}
	if got := cleanFileName("", "image/jpeg"); got != "image.jpg" {
		t.Errorf("jpeg fallback = %q", got)
	}
}
