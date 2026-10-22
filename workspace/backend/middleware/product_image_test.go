package middleware

import (
	"bytes"
	"context"
	"errors"
	"io"
	"main/component/models"
	"main/utils"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

var testImage = models.ImageFile{Name: "serum.png", ContentType: "image/png", Content: []byte("\x89PNG fake image bytes")}

func newImageMiddleware(tiktokURL, lazadaURL string) *ApiMiddleware {
	cfg := &utils.Config{}
	cfg.Ecommerce.TiktokConfig = utils.TiktokConfig{AppKey: "tt-key", AppSecret: "tt-secret", Url: tiktokURL}
	cfg.Ecommerce.LazadaConfig = utils.LazadaConfig{AppKey: "lz-key", AppSecret: "lz-secret", Url: lazadaURL}
	return NewMiddleware(cfg, nil)
}

// readUpload parses the multipart request the way a marketplace would.
func readUpload(t *testing.T, r *http.Request, fileField string) (map[string]string, []byte, string) {
	t.Helper()
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		t.Fatalf("not multipart: %v", err)
	}
	fields := map[string]string{}
	for k, v := range r.MultipartForm.Value {
		fields[k] = v[0]
	}
	f, h, err := r.FormFile(fileField)
	if err != nil {
		t.Fatalf("no %q file: %v", fileField, err)
	}
	defer f.Close()
	content, _ := io.ReadAll(f)
	return fields, content, h.Filename
}

func TestTiktokUploadImage(t *testing.T) {
	var gotSignOK bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/product/202309/images/upload" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("x-tts-access-token") != "shop-token" {
			t.Errorf("access token header = %q", r.Header.Get("x-tts-access-token"))
		}
		q := r.URL.Query()
		if q.Get("app_key") != "tt-key" || q.Get("timestamp") == "" || q.Get("access_token") != "" {
			t.Errorf("query = %v", q)
		}
		// Re-sign without "sign": the multipart body is not part of TikTok's signature.
		unsigned := r.Clone(context.Background())
		u := *r.URL
		uq := u.Query()
		uq.Del("sign")
		u.RawQuery = uq.Encode()
		unsigned.URL = &u
		gotSignOK = TiktokGenerateSignSign(unsigned, "tt-secret") == q.Get("sign")

		fields, content, name := readUpload(t, r, "data")
		if fields["use_case"] != "DESCRIPTION_IMAGE" || !bytes.Equal(content, testImage.Content) || name != "serum.png" {
			t.Errorf("form = %v, %q, %q", fields, content, name)
		}
		w.Write([]byte(`{"code":0,"message":"Success","request_id":"r1","data":{"uri":"tos-maliva-i-o3syd03w52-us/abc","url":"https://p16-oec-va.ibyteimg.com/abc~tplv.jpg","width":800,"height":600,"use_case":"DESCRIPTION_IMAGE"}}`))
	}))
	defer srv.Close()

	up, err := newImageMiddleware(srv.URL, "").TiktokUploadImage(context.Background(), "shop-token", testImage, "DESCRIPTION_IMAGE")
	if err != nil {
		t.Fatal(err)
	}
	if !gotSignOK {
		t.Error("sign does not match TikTok's algorithm")
	}
	want := models.UploadedImage{URI: "tos-maliva-i-o3syd03w52-us/abc", URL: "https://p16-oec-va.ibyteimg.com/abc~tplv.jpg", Width: 800, Height: 600}
	if *up != want {
		t.Errorf("uploaded = %+v", *up)
	}
}

func TestLazadaUploadImage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/rest/image/upload" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		q := r.URL.Query()
		params := map[string]string{}
		for k := range q {
			if k != "sign" {
				params[k] = q.Get(k)
			}
		}
		if params["app_key"] != "lz-key" || params["access_token"] != "shop-token" || params["sign_method"] != "sha256" || len(params["timestamp"]) != 13 {
			t.Errorf("params = %v", params)
		}
		if LazadaGenerateSign("/image/upload", params, "lz-secret") != q.Get("sign") {
			t.Error("sign does not match Lazada's algorithm")
		}
		_, content, name := readUpload(t, r, "image")
		if !bytes.Equal(content, testImage.Content) || name != "serum.png" {
			t.Errorf("file = %q %q", content, name)
		}
		w.Write([]byte(`{"data":{"image":{"hash_code":"h1","url":"https://vn-live.slatic.net/p/abc.png"}},"code":"0","request_id":"r2"}`))
	}))
	defer srv.Close()

	up, err := newImageMiddleware("", srv.URL+"/rest").LazadaUploadImage(context.Background(), "shop-token", testImage)
	if err != nil {
		t.Fatal(err)
	}
	if up.URL != "https://vn-live.slatic.net/p/abc.png" || up.URI != "" {
		t.Errorf("uploaded = %+v", *up)
	}
}

func TestUploadImageFailures(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
		lazada bool
		code   string
	}{
		{"tiktok refuses", 200, `{"code":12052700,"message":"image size exceeds the limit","data":{}}`, false, "12052700"},
		{"tiktok no url", 200, `{"code":0,"message":"Success","data":{}}`, false, "0"},
		{"tiktok http 500", 500, `oops`, false, ""},
		{"tiktok not json", 200, `<html>`, false, ""},
		{"lazada refuses", 200, `{"type":"ISV","code":"IllegalAccessToken","message":"The specified access token is invalid","request_id":"r"}`, true, "IllegalAccessToken"},
		{"lazada http 502", 502, ``, true, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				w.Write([]byte(tc.body))
			}))
			defer srv.Close()
			mw := newImageMiddleware(srv.URL, srv.URL)
			var err error
			if tc.lazada {
				_, err = mw.LazadaUploadImage(context.Background(), "secret-token-123", testImage)
			} else {
				_, err = mw.TiktokUploadImage(context.Background(), "secret-token-123", testImage, "MAIN_IMAGE")
			}
			var me *MarketplaceError
			if !errors.As(err, &me) {
				t.Fatalf("err = %v, want *MarketplaceError", err)
			}
			if me.Code != tc.code {
				t.Errorf("code = %q, want %q", me.Code, tc.code)
			}
			if strings.Contains(err.Error(), "secret-token-123") {
				t.Errorf("error leaks the token: %v", err)
			}
		})
	}
}

func TestUploadImageUnreachable(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	addr := srv.URL
	srv.Close() // nothing listens any more
	_, err := newImageMiddleware(addr, "").TiktokUploadImage(context.Background(), "tok", testImage, "MAIN_IMAGE")
	var me *MarketplaceError
	if !errors.As(err, &me) || me.Platform != "Tiktok" {
		t.Errorf("err = %v", err)
	}
}
