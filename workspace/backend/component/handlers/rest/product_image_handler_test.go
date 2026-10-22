package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"main/component/models"
	"main/component/services"
	"main/utils"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type fakeImagesSvc struct {
	got     *models.ProductImageUpload
	actorID int64
	query   models.ProductImageListQuery
	err     error
}

func (f *fakeImagesSvc) Shops(context.Context) (*models.ImageShopsResponse, error) {
	return &models.ImageShopsResponse{Shops: []models.ImageShop{{MarketPlace: "Tiktok", ShopID: "tt-1"}}, UseCases: models.TiktokImageUseCases}, f.err
}
func (f *fakeImagesSvc) ListImages(_ context.Context, q models.ProductImageListQuery) (*models.ProductImageListResponse, error) {
	f.query = q
	if f.err != nil {
		return nil, f.err
	}
	return &models.ProductImageListResponse{Images: []models.ProductImage{}, Page: 1, PageSize: 24}, nil
}
func (f *fakeImagesSvc) Upload(_ context.Context, actor *models.SessionUser, in models.ProductImageUpload) (*models.ProductImage, error) {
	f.got, f.actorID = &in, actor.ID
	if f.err != nil {
		return nil, f.err
	}
	return &models.ProductImage{ID: 1, MarketPlace: in.MarketPlace, ShopID: in.ShopID, URL: "https://cdn/x.png"}, nil
}

func newImagesRouter(user *models.SessionUser, svc *fakeImagesSvc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	app := gin.New()
	APIRoutes(app, &ApiServer{
		Cfg:       &utils.Config{Auth: utils.AuthConfig{FrontendURL: "http://localhost:5173"}},
		AuthSvc:   &fakeAuthSvc{validToken: "tok", sessionUser: user},
		ImagesSvc: svc,
	})
	return app
}

var asWriter = sessionWith("operator", models.PermOrdersRead, models.PermOrdersWrite)

// uploadRequest builds a multipart POST like the admin app sends.
func uploadRequest(t *testing.T, fields map[string]string, fileName string, content []byte, headers map[string]string) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for k, v := range fields {
		w.WriteField(k, v)
	}
	if fileName != "" {
		part, err := w.CreateFormFile("file", fileName)
		if err != nil {
			t.Fatal(err)
		}
		part.Write(content)
	}
	w.Close()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/product-images", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return req
}

func serve(app *gin.Engine, req *http.Request) (*httptest.ResponseRecorder, apiBody) {
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	var b apiBody
	json.Unmarshal(w.Body.Bytes(), &b)
	return w, b
}

var uploadHeaders = map[string]string{"Cookie": "ecom_session=tok", "Origin": "http://localhost:5173"}

func TestProductImageUploadRoute(t *testing.T) {
	svc := &fakeImagesSvc{}
	app := newImagesRouter(asWriter, svc)
	fields := map[string]string{"market_place": "Tiktok", "shop_id": "tt-1", "use_case": "MAIN_IMAGE"}
	w, body := serve(app, uploadRequest(t, fields, "a.png", []byte("png-bytes"), uploadHeaders))
	if body.Code != http.StatusOK || svc.got == nil {
		t.Fatalf("upload: %+v", body)
	}
	if svc.got.MarketPlace != "Tiktok" || svc.got.ShopID != "tt-1" || svc.got.UseCase != "MAIN_IMAGE" ||
		svc.got.File.Name != "a.png" || string(svc.got.File.Content) != "png-bytes" || svc.actorID != asWriter.ID {
		t.Errorf("service got %+v (actor %d)", svc.got, svc.actorID)
	}
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Error("missing no-store")
	}
}

func TestProductImageUploadRejections(t *testing.T) {
	fields := map[string]string{"market_place": "Lazada", "shop_id": "lz-1"}
	tooBig := make([]byte, services.MaxProductImageBytes+1)

	cases := []struct {
		name    string
		user    *models.SessionUser
		req     func(t *testing.T) *http.Request
		code    int
		message string
	}{
		{"no session", asWriter, func(t *testing.T) *http.Request {
			return uploadRequest(t, fields, "a.png", []byte("x"), map[string]string{"Origin": "http://localhost:5173"})
		}, 401, "UNAUTHENTICATED"},
		{"without orders.write", asViewer, func(t *testing.T) *http.Request {
			return uploadRequest(t, fields, "a.png", []byte("x"), uploadHeaders)
		}, 403, "FORBIDDEN"},
		{"cross origin", asWriter, func(t *testing.T) *http.Request {
			return uploadRequest(t, fields, "a.png", []byte("x"), map[string]string{"Cookie": "ecom_session=tok", "Origin": "https://evil.example"})
		}, 403, "FORBIDDEN_ORIGIN"},
		{"no file", asWriter, func(t *testing.T) *http.Request {
			return uploadRequest(t, fields, "", nil, uploadHeaders)
		}, 400, "INVALID_REQUEST"},
		{"file over 5 MB", asWriter, func(t *testing.T) *http.Request {
			return uploadRequest(t, fields, "big.png", tooBig, uploadHeaders)
		}, 400, "INVALID_IMAGE"},
		{"body over the limit", asWriter, func(t *testing.T) *http.Request {
			return uploadRequest(t, fields, "huge.png", make([]byte, services.MaxProductImageBytes+2<<20), uploadHeaders)
		}, 400, "INVALID_IMAGE"},
		{"not multipart", asWriter, func(t *testing.T) *http.Request {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/product-images", bytes.NewBufferString(`{"url":"x"}`))
			req.Header.Set("Content-Type", "application/json")
			for k, v := range uploadHeaders {
				req.Header.Set(k, v)
			}
			return req
		}, 400, "INVALID_REQUEST"},
	}
	for _, tc := range cases {
		svc := &fakeImagesSvc{}
		_, body := serve(newImagesRouter(tc.user, svc), tc.req(t))
		if body.Code != tc.code || body.Message != tc.message {
			t.Errorf("%s: %+v", tc.name, body)
		}
		if svc.got != nil {
			t.Errorf("%s: service was called", tc.name)
		}
	}
}

func TestProductImageServiceErrorsKeepTheirCode(t *testing.T) {
	for _, e := range []error{services.ErrInvalidImage, services.ErrNotFound, services.ErrMarketplace} {
		app := newImagesRouter(asWriter, &fakeImagesSvc{err: e})
		_, body := serve(app, uploadRequest(t, map[string]string{"market_place": "Tiktok", "shop_id": "x"}, "a.png", []byte("x"), uploadHeaders))
		ae := services.AsAuthError(e)
		if body.Code != ae.Status || body.Message != ae.Code {
			t.Errorf("%s: %+v", ae.Code, body)
		}
	}
}

func TestProductImageReadRoutes(t *testing.T) {
	svc := &fakeImagesSvc{}
	app := newImagesRouter(asWriter, svc)
	hdr := map[string]string{"Cookie": "ecom_session=tok"}

	_, body := do(app, http.MethodGet, "/api/v1/product-images/shops", "", hdr)
	var shops models.ImageShopsResponse
	if body.Code != 200 || json.Unmarshal(body.Data[0], &shops) != nil || len(shops.Shops) != 1 || len(shops.UseCases) != 5 {
		t.Errorf("shops: %+v", body)
	}

	_, body = do(app, http.MethodGet, "/api/v1/product-images?market_place=Lazada&shop_id=lz-1&page=2&page_size=10", "", hdr)
	if body.Code != 200 || svc.query != (models.ProductImageListQuery{MarketPlace: "Lazada", ShopID: "lz-1", Page: "2", PageSize: "10"}) {
		t.Errorf("list: %+v, query %+v", body, svc.query)
	}

	for _, path := range []string{"/api/v1/product-images", "/api/v1/product-images/shops"} {
		_, body := do(newImagesRouter(asViewer, &fakeImagesSvc{}), http.MethodGet, path, "", hdr)
		if body.Code != http.StatusForbidden {
			t.Errorf("%s without orders.write: %+v", path, body)
		}
	}
}
