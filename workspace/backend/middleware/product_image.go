package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"main/component/models"
	lazadaModel "main/component/models/lazada_model"
	tiktokModel "main/component/models/tiktok_model"
	"mime/multipart"
	"net/http"
	"net/url"
	"time"
)

// Product image uploads (EC-10). Both marketplaces only accept product images
// that were uploaded to them first; they answer with the URL (and, on TikTok,
// a uri) that product listings reference. Unlike the order clients, these
// don't retry on an expired token: the token cron refreshes tokens, and a
// failed upload is reported to the user, who can simply try again.

const (
	tiktokImageUploadPath = "/product/202309/images/upload"
	lazadaImageUploadPath = "/image/upload"
)

// uploadHTTPClient bounds an upload; the default client has no timeout.
var uploadHTTPClient = &http.Client{Timeout: 60 * time.Second}

// MarketplaceError is a refusal or an unreadable answer from a marketplace.
// Code/Message come from the marketplace and are for the logs.
type MarketplaceError struct {
	Platform string
	Status   int
	Code     string
	Message  string
}

func (e *MarketplaceError) Error() string {
	return fmt.Sprintf("%s image upload failed: http=%d code=%s message=%s", e.Platform, e.Status, e.Code, e.Message)
}

// multipartBody writes the file under fileField plus any extra text fields.
func multipartBody(fileField, fileName string, content []byte, fields map[string]string) (*bytes.Buffer, string, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for k, v := range fields {
		if err := w.WriteField(k, v); err != nil {
			return nil, "", err
		}
	}
	part, err := w.CreateFormFile(fileField, fileName)
	if err != nil {
		return nil, "", err
	}
	if _, err := part.Write(content); err != nil {
		return nil, "", err
	}
	if err := w.Close(); err != nil {
		return nil, "", err
	}
	return &buf, w.FormDataContentType(), nil
}

func readUploadResponse(platform string, resp *http.Response) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, &MarketplaceError{Platform: platform, Status: resp.StatusCode, Message: err.Error()}
	}
	if resp.StatusCode != http.StatusOK {
		return nil, &MarketplaceError{Platform: platform, Status: resp.StatusCode, Message: string(body)}
	}
	return body, nil
}

// TiktokUploadImage sends one image to TikTok Shop (multipart field "data",
// plus "use_case"). Signed like every TikTok call, except that a multipart
// body is not part of the signature (TiktokGenerateSignSign handles that).
func (mw *ApiMiddleware) TiktokUploadImage(ctx context.Context, accessToken string, img models.ImageFile, useCase string) (*models.UploadedImage, error) {
	body, contentType, err := multipartBody("data", img.Name, img.Content, map[string]string{"use_case": useCase})
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf("app_key=%s&timestamp=%d", mw.cfg.Ecommerce.TiktokConfig.AppKey, time.Now().Unix())
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, mw.cfg.Ecommerce.TiktokConfig.Url+tiktokImageUploadPath+"?"+query, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("x-tts-access-token", accessToken)
	q := req.URL.Query()
	q.Add("sign", TiktokGenerateSignSign(req, mw.cfg.Ecommerce.TiktokConfig.AppSecret))
	req.URL.RawQuery = q.Encode()

	resp, err := uploadHTTPClient.Do(req)
	if err != nil {
		return nil, &MarketplaceError{Platform: "Tiktok", Message: err.Error()}
	}
	defer resp.Body.Close()
	raw, err := readUploadResponse("Tiktok", resp)
	if err != nil {
		return nil, err
	}
	var out tiktokModel.UploadImageResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, &MarketplaceError{Platform: "Tiktok", Status: resp.StatusCode, Message: "unreadable response"}
	}
	if out.Code != 0 || out.Data.URL == "" {
		return nil, &MarketplaceError{Platform: "Tiktok", Status: resp.StatusCode, Code: fmt.Sprint(out.Code), Message: out.Message}
	}
	return &models.UploadedImage{URI: out.Data.URI, URL: out.Data.URL, Width: out.Data.Width, Height: out.Data.Height}, nil
}

// LazadaUploadImage sends one image to Lazada (/image/upload, multipart field
// "image"). System parameters and the signature go in the query string, as in
// Lazada's SDKs; the file is not part of the signature.
func (mw *ApiMiddleware) LazadaUploadImage(ctx context.Context, accessToken string, img models.ImageFile) (*models.UploadedImage, error) {
	body, contentType, err := multipartBody("image", img.Name, img.Content, nil)
	if err != nil {
		return nil, err
	}
	params := map[string]string{
		"app_key":      mw.cfg.Ecommerce.LazadaConfig.AppKey,
		"timestamp":    fmt.Sprintf("%d", time.Now().UnixMilli()),
		"sign_method":  "sha256",
		"access_token": accessToken,
	}
	params["sign"] = LazadaGenerateSign(lazadaImageUploadPath, params, mw.cfg.Ecommerce.LazadaConfig.AppSecret)
	query := url.Values{}
	for k, v := range params {
		query.Set(k, v)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, mw.cfg.Ecommerce.LazadaConfig.Url+lazadaImageUploadPath+"?"+query.Encode(), body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)

	resp, err := uploadHTTPClient.Do(req)
	if err != nil {
		return nil, &MarketplaceError{Platform: "Lazada", Message: err.Error()}
	}
	defer resp.Body.Close()
	raw, err := readUploadResponse("Lazada", resp)
	if err != nil {
		return nil, err
	}
	var out lazadaModel.UploadImageResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, &MarketplaceError{Platform: "Lazada", Status: resp.StatusCode, Message: "unreadable response"}
	}
	if out.Code != "0" || out.Data.Image.URL == "" {
		return nil, &MarketplaceError{Platform: "Lazada", Status: resp.StatusCode, Code: out.Code, Message: out.Message}
	}
	return &models.UploadedImage{URL: out.Data.Image.URL}, nil
}
