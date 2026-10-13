package middleware

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	models "main/component/models/tiktok_model"
	"main/utils"
	"mime"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"time"
)

func (mw *ApiMiddleware) TiktokGetOrderList(ctx context.Context, updatedAfter int64, accessToken, rfToken, shopId, shopCipher, pageToken string) (models.TikTokOrderResponse, error) {
	url := "/order/202309/orders/search"
	var requestBody = models.TikTokOrderSearchFilter{
		UpdateTimeGe: updatedAfter,
	}

	params := map[string]string{}
	params["app_key"] = mw.cfg.Ecommerce.TiktokConfig.AppKey
	params["timestamp"] = fmt.Sprintf("%d", time.Now().Unix())
	params["page_size"] = strconv.Itoa(100)
	params["shop_cipher"] = shopCipher
	params["sort_order"] = "ASC"

	if pageToken != "" {
		params["page_token"] = pageToken
	}

	respBody, err := mw.DoTiktokPost(ctx, url, accessToken, rfToken, shopId, shopCipher, params, requestBody)
	if err != nil {
		return models.TikTokOrderResponse{}, err
	}
	var resp models.TikTokOrderResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return models.TikTokOrderResponse{}, err
	}
	return resp, nil
}

func (mw *ApiMiddleware) TiktokGetRefundList(ctx context.Context, updatedAfter int64, accessToken, rfToken, shopId, shopCipher, pageToken string) (*models.TikTokRefundResponse, error) {
	url := "/return_refund/202309/returns/search"
	var requestBody = models.TikTokOrderSearchFilter{
		UpdateTimeGe: updatedAfter,
	}
	params := map[string]string{}
	params["app_key"] = mw.cfg.Ecommerce.TiktokConfig.AppKey
	params["timestamp"] = fmt.Sprintf("%d", time.Now().Unix())
	params["page_size"] = strconv.Itoa(50)
	params["shop_cipher"] = shopCipher
	params["sort_order"] = "ASC"

	if pageToken != "" {
		params["page_token"] = pageToken
	}

	respBody, err := mw.DoTiktokPost(ctx, url, accessToken, rfToken, shopId, shopCipher, params, requestBody)
	if err != nil {
		return nil, err
	}
	resp := &models.TikTokRefundResponse{}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (mw *ApiMiddleware) TiktokGetOrderDetail(ctx context.Context, ids string, accessToken, rfToken, shopCipher string) ([]models.TikTokOrderDetailModel, error) {
	baseUrl := "/order/202507/orders"
	params := url.Values{}
	params.Add("ids", ids)
	respBody, err := mw.DoTiktokGET(ctx, baseUrl, accessToken, rfToken, shopCipher, params)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Data struct {
			Orders []models.TikTokOrderDetailModel `json:"orders"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, err
	}
	return resp.Data.Orders, nil
}

func (mw *ApiMiddleware) TiktokShipPackage(ctx context.Context, packageId, accessToken, rfToken, shopId, shopCipher string) ([]byte, error) {
	baseUrl := "/fulfillment/202309/packages/" + packageId + "/ship"
	body := map[string]interface{}{
		"handover_method": "PICKUP",
	}
	params := map[string]string{}
	params["app_key"] = mw.cfg.Ecommerce.TiktokConfig.AppKey
	params["timestamp"] = fmt.Sprintf("%d", time.Now().Unix())
	params["page_size"] = strconv.Itoa(100)
	params["shop_cipher"] = shopCipher

	respBody, err := mw.DoTiktokPost(ctx, baseUrl, accessToken, rfToken, shopId, shopCipher, params, body)
	if err != nil {
		return nil, err
	}

	return respBody, nil
}

func (mw *ApiMiddleware) TiktokGetPackageShippingDocument(ctx context.Context, packageId, accessToken, rfToken, shopCipher string) *models.TrackingResponse {
	baseUrl := "/fulfillment/202309/packages/" + packageId + "/shipping_documents"
	params := url.Values{}
	params.Add("document_type", "SHIPPING_LABEL")
	respBody, err := mw.DoTiktokGET(ctx, baseUrl, accessToken, rfToken, shopCipher, params)
	if err != nil {
		return nil
	}
	var resp *models.TrackingResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil
	}
	if resp.Code != 0 {
		return nil
	}
	return resp
}

func (mw *ApiMiddleware) TiktokGetStatementTransactions(ctx context.Context, accessToken, rfToken, shopId, shopCipher, orderID string) (*models.OrderSettlementResponse, error) {
	baseUrl := "/finance/202501/orders/+" + orderID + "/statement_transactions"
	params := url.Values{}
	respBody, err := mw.DoTiktokGET(ctx, baseUrl, accessToken, rfToken, shopCipher, params)
	if err != nil {
		return nil, err
	}
	resp := &models.OrderSettlementResponse{}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (mw *ApiMiddleware) TiktokRefreshAccessToken(shopId, rfToken string) error {
	baseUrl := "https://auth.tiktok-shops.com/api/v2/token/refresh"
	params := url.Values{}
	params.Add("app_key", mw.cfg.Ecommerce.TiktokConfig.AppKey)
	params.Add("app_secret", mw.cfg.Ecommerce.TiktokConfig.AppSecret)
	params.Add("refresh_token", rfToken)
	params.Add("grant_type", "refresh_token")
	apiUrl := fmt.Sprintf("%s?%s", baseUrl, params.Encode())

	utils.ShowInfoLogs("[TiktokRefreshAccessToken] shopId: %s, rfToken: %s", shopId, rfToken)
	err, result := utils.GetAPI(apiUrl, nil)
	utils.ShowInfoLogs("[TiktokRefreshAccessToken] error: %d", err)
	utils.ShowInfoLogs("[TiktokRefreshAccessToken] result: %v", result)

	data := result["data"].(map[string]interface{})

	var TiktokAccessToken = data["access_token"].(string)
	var TiktokRefreshToken = data["refresh_token"].(string)
	var expireIn = int64(data["access_token_expire_in"].(float64))
	var expireAt = time.Now().Add(time.Duration(expireIn) * time.Second)

	mw.tokenRepo.UpdateAccessToken(context.Background(), utils.TikTok, TiktokAccessToken, TiktokRefreshToken, shopId, expireAt)
	return nil
}

func (mw *ApiMiddleware) DoTiktokPost(ctx context.Context, baseURL, access_token, rfToken, shopId, shopCipher string, params map[string]string, payload any) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}
	fullURL := fmt.Sprintf("%s%s?%s", mw.cfg.Ecommerce.TiktokConfig.Url, baseURL, values.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-tts-access-token", access_token)
	req.Header.Set("Content-Type", "application/json")
	query := req.URL.Query()
	sign := TiktokGenerateSignSign(req, mw.cfg.Ecommerce.TiktokConfig.AppSecret)
	query.Add("sign", sign)
	req.URL.RawQuery = query.Encode()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		mw.TiktokRefreshAccessToken(shopId, rfToken)
		return mw.DoTiktokPost(ctx, baseURL, access_token, rfToken, shopId, shopCipher, params, payload)
	}
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return respBody, nil
}

func (mw *ApiMiddleware) DoTiktokGET(ctx context.Context, baseURL, access_token, rfToken, shopCipher string, params url.Values) ([]byte, error) {
	params.Add("app_key", mw.cfg.Ecommerce.TiktokConfig.AppKey)
	params.Add("timestamp", fmt.Sprintf("%d", time.Now().Unix()))
	params.Add("page_size", strconv.Itoa(100))
	params.Add("shop_cipher", shopCipher)
	fullURL := fmt.Sprintf("%s%s?%s", mw.cfg.Ecommerce.TiktokConfig.Url, baseURL, params.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("x-tts-access-token", access_token)
	req.Header.Set("Content-Type", "multipart/form-data")
	query := req.URL.Query()
	sign := TiktokGenerateSignSign(req, mw.cfg.Ecommerce.TiktokConfig.AppSecret)
	query.Add("sign", sign)
	req.URL.RawQuery = query.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		mw.TiktokRefreshAccessToken(shopCipher, rfToken)
		return mw.DoTiktokGET(ctx, baseURL, access_token, rfToken, shopCipher, params)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return respBody, nil
}

func TiktokGenerateSignSign(req *http.Request, secret string) string {
	queries := req.URL.Query()

	// extract all query parameters excluding sign and access_token
	keys := make([]string, len(queries))
	idx := 0
	for k := range queries {
		// params except 'sign' & 'access_token'
		if k != "sign" && k != "access_token" {
			keys[idx] = k
			idx++
		}
	}

	// reorder the parameters' key in alphabetical order
	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})

	// Concatenate all the parameters in the format of {key}{value}
	input := ""
	for _, key := range keys {
		input = input + key + queries.Get(key)
	}

	// append the request path
	input = req.URL.Path + input

	// if the request header Content-type is not multipart/form-data, append body to the end
	mediaType, _, _ := mime.ParseMediaType(req.Header.Get("Content-type"))
	if mediaType != "multipart/form-data" {
		body, _ := io.ReadAll(req.Body)
		input = input + string(body)

		req.Body.Close()
		// reset body after reading from the original
		req.Body = io.NopCloser(bytes.NewReader(body))
	}

	// wrap the string generated in step 5 with the App secret
	input = secret + input + secret

	return generateSHA256(input, secret)
}

func generateSHA256(input, secret string) string {
	// encode the digest byte stream in hexadecimal and use sha256 to generate sign with salt(secret)
	h := hmac.New(sha256.New, []byte(secret))

	if _, err := h.Write([]byte(input)); err != nil {
		// TODO error log
		return ""
	}

	return hex.EncodeToString(h.Sum(nil))
}
