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
	"log"
	models "main/component/models/lazada_model"
	"main/utils"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

func (mw *ApiMiddleware) RefreshLazadaAccessToken(shopId string) error {
	// baseUrl := "/auth/token/refresh"
	// headers := map[string]string{
	// 	"Content-Type": "application/json;charset=utf-8",
	// }
	// params := map[string]string{
	// 	"app_key":     mw.cfg.Ecommerce.LazadaConfig.AppKey,
	// 	"timestamp":   fmt.Sprintf("%d000", time.Now().Unix()),
	// 	"sign_method": "sha256",
	// }
	// body := map[string]interface{}{
	// 	"refresh_token": mw.tokenRepo.GetRefreshToken(context.Background(), utils.Lazada, shopId),
	// }
	// paramCalSign := params
	// for k, v := range body {
	// 	paramCalSign[k] = fmt.Sprintf("%v", v)
	// }
	// sign := GenerateSign(baseUrl, paramCalSign, mw.cfg.Ecommerce.LazadaConfig.AppSecret)
	// params["sign"] = sign
	// values := url.Values{}
	// for k, v := range params {
	// 	values.Set(k, v)
	// }
	// fullURL := fmt.Sprintf("%s%s?%s", mw.cfg.Ecommerce.LazadaConfig.Url, baseUrl, values.Encode())
	// _, result := utils.PostAPI(fullURL, headers, body)
	// var respMap map[string]interface{}
	// b, _ := json.Marshal(result) // convert result về []byte
	// json.Unmarshal(b, &respMap)
	// data := respMap["data"].(map[string]interface{})
	// var LazadaAccessToken = data["access_token"].(string)
	// var LazadaRefreshToken = data["refresh_token"].(string)
	// mw.tokenRepo.UpdateAccessToken(context.Background(), utils.Lazada, LazadaAccessToken, LazadaRefreshToken, shopId, time.Now().AddDate(1, 0, 0))
	return nil
}

func (mw *ApiMiddleware) DoLazadaPost(ctx context.Context, baseURL, accessToken, rfToken string, params map[string]string, payloadKey string, payload any) ([]byte, error) {
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}
	jsonStr := string(jsonBytes)
	params[payloadKey] = jsonStr
	params["access_token"] = accessToken
	paramCalSign := params
	for k, v := range params {
		paramCalSign[k] = v
	}
	sign := LazadaGenerateSign(baseURL, paramCalSign, mw.cfg.Ecommerce.LazadaConfig.AppSecret)
	params["sign"] = sign
	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}
	fullURL := fmt.Sprintf("%s%s?%s", mw.cfg.Ecommerce.LazadaConfig.Url, baseURL, values.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, nil)
	req.Header.Set("Content-Type", "application/json;charset=utf-8")
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var m map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		log.Fatal(err)
	}
	if !mw.CheckResponseStatus(rfToken, m) {
		return mw.DoLazadaPost(ctx, baseURL, accessToken, rfToken, params, payloadKey, payload)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return respBody, nil
}

func (mw *ApiMiddleware) DoLazadaGet(ctx context.Context, baseURL, accessToken, rfToken string, params map[string]string) ([]byte, error) {
	params["access_token"] = accessToken
	sign := LazadaGenerateSign(baseURL, params, mw.cfg.Ecommerce.LazadaConfig.AppSecret)
	params["sign"] = sign

	// Build query string
	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}
	fullURL := fmt.Sprintf("%s%s?%s", mw.cfg.Ecommerce.LazadaConfig.Url, baseURL, values.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	req.Header.Set("Content-Type", "application/json;charset=utf-8")
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	respBody, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return nil, err
	}

	var m map[string]interface{}
	if err := json.Unmarshal(respBody, &m); err != nil {
		return nil, err
	}

	if !mw.CheckResponseStatus(rfToken, m) {
		return mw.DoLazadaGet(ctx, baseURL, accessToken, rfToken, params)
	}

	return respBody, nil
}

func (mw *ApiMiddleware) LazadaGetOrderList(ctx context.Context, updatedAt, accessToken, rfToken string, offset, limit int) (*models.OrdersResponse, error) {
	url := "/orders/get"
	params := map[string]string{}
	params["app_key"] = mw.cfg.Ecommerce.LazadaConfig.AppKey
	params["timestamp"] = fmt.Sprintf("%d000", time.Now().Unix())
	params["sign_method"] = "sha256"
	params["update_after"] = updatedAt
	params["offset"] = fmt.Sprint(offset)
	params["limit"] = fmt.Sprint(limit)
	params["sort_direction"] = "ASC"

	respBody, err := mw.DoLazadaGet(ctx, url, accessToken, rfToken, params)
	if err != nil {
		return nil, err
	}
	resp := &models.OrdersResponse{}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (mw *ApiMiddleware) LazadaGetOrder(ctx context.Context, orderID, accessToken, rfToken string) (*models.OrderDetailResponse, error) {
	url := "/order/get"
	params := map[string]string{}
	params["app_key"] = mw.cfg.Ecommerce.LazadaConfig.AppKey
	params["timestamp"] = fmt.Sprintf("%d000", time.Now().Unix())
	params["sign_method"] = "sha256"
	params["order_id"] = orderID

	respBody, err := mw.DoLazadaGet(ctx, url, accessToken, rfToken, params)
	if err != nil {
		return nil, err
	}
	resp := &models.OrderDetailResponse{}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (mw *ApiMiddleware) LazadaGetOrderItems(ctx context.Context, orderID int64, accessToken, rfToken string) (*models.OrderItemResponse, error) {
	url := "/order/items/get"
	params := map[string]string{}
	params["app_key"] = mw.cfg.Ecommerce.LazadaConfig.AppKey
	params["timestamp"] = fmt.Sprintf("%d000", time.Now().Unix())
	params["sign_method"] = "sha256"
	params["order_id"] = fmt.Sprintf("%d", orderID)

	respBody, err := mw.DoLazadaGet(ctx, url, accessToken, rfToken, params)
	if err != nil {
		return nil, err
	}

	resp := &models.OrderItemResponse{}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (mw *ApiMiddleware) LazadaPack(ctx context.Context, packRequest models.PackRequest, accessToken, rfToken string) ([]byte, error) {
	url := "/order/fulfill/pack"
	params := map[string]string{}
	params["app_key"] = mw.cfg.Ecommerce.LazadaConfig.AppKey
	params["timestamp"] = fmt.Sprintf("%d000", time.Now().Unix())
	params["sign_method"] = "sha256"
	var payloadKey = "packReq"

	resp, err := mw.DoLazadaPost(ctx, url, accessToken, rfToken, params, payloadKey, packRequest)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (mw *ApiMiddleware) LazadaPrintAWB(ctx context.Context, packageid, accessToken, rfToken string) (*models.ApiPrintAWBResponse, error) {
	baseUrl := "/order/package/document/get"
	params := map[string]string{}
	params["app_key"] = mw.cfg.Ecommerce.LazadaConfig.AppKey
	params["timestamp"] = fmt.Sprintf("%d000", time.Now().Unix())
	params["sign_method"] = "sha256"
	var body = models.PrintAWBPayload{
		GetDocumentReq: models.PackageReq{
			DocType: "PDF",
			Packages: []models.Package{
				{PackageId: packageid},
			},
		},
	}
	bodyBytes, err := json.Marshal(body.GetDocumentReq)
	if err != nil {
		log.Fatal(err)
	}
	params["getDocumentReq"] = string(bodyBytes)
	respBody, err := mw.DoLazadaGet(ctx, baseUrl, accessToken, rfToken, params)
	if respBody == nil && err != nil {
		return nil, err
	}

	var resp *models.ApiPrintAWBResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, err
	}

	return resp, nil
}

func (mw *ApiMiddleware) QueryTransactionDetails(ctx context.Context, accessToken, rfToken, orderId string) (*models.SettlementResponse, error) {
	baseUrl := "/finance/transaction/details/get"
	params := map[string]string{}
	params["app_key"] = mw.cfg.Ecommerce.LazadaConfig.AppKey
	params["timestamp"] = fmt.Sprintf("%d000", time.Now().Unix())
	params["sign_method"] = "sha256"
	params["start_time"] = time.Now().AddDate(0, -2, 0).Format("2006-01-02")
	params["end_time"] = time.Now().Format("2006-01-02")
	params["trade_order_id"] = orderId
	respBody, err := mw.DoLazadaGet(ctx, baseUrl, accessToken, rfToken, params)
	if err != nil {
		return nil, err
	}
	resp := &models.SettlementResponse{}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (mw *ApiMiddleware) LazadaReadyToShip(ctx context.Context, accessToken, rfToken, packageId string) (*models.ReadyToShipResponse, error) {
	baseUrl := "/order/package/rts"
	params := map[string]string{}
	params["app_key"] = mw.cfg.Ecommerce.LazadaConfig.AppKey
	params["timestamp"] = fmt.Sprintf("%d000", time.Now().Unix())
	params["sign_method"] = "sha256"
	ReadyToShip := models.ReadyToShipReq{
		Packages: []models.ReadyToShipPackage{
			{PackageID: packageId},
		},
	}
	respBody, err := mw.DoLazadaPost(ctx, baseUrl, accessToken, rfToken, params, "readyToShipReq", ReadyToShip)
	if err != nil {
		return nil, err
	}
	if len(respBody) == 0 {
		return nil, nil
	}

	utils.ShowInfoLogs("Lazada ReadyToShip PackageId: %s Response: %s", packageId, string(respBody))

	var resp *models.ReadyToShipResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, err
	}

	return resp, nil
}

func (mw *ApiMiddleware) GenerateLazadaAccessToken(ctx context.Context, accessToken, rfToken, codeOAuth string) models.AuthResponse {
	baseUrl := "https://auth.lazada.com/rest/auth/token/create"
	params := map[string]string{
		"app_key":     mw.cfg.Ecommerce.LazadaConfig.AppKey,
		"timestamp":   strconv.FormatUint(uint64(time.Now().Unix()), 10),
		"sign_method": "sha256",
		"code":        codeOAuth,
	}
	resBody, err := mw.DoLazadaGet(ctx, baseUrl, accessToken, rfToken, params)
	if resBody == nil && err != nil {
		return models.AuthResponse{}
	}
	var resp models.AuthResponse
	if err := json.Unmarshal(resBody, &resp); err != nil {
		return models.AuthResponse{}
	}
	return resp
}

func LazadaGenerateSign(apiPath string, params map[string]string, appSecret string) string {
	keys := []string{}
	union := map[string]string{}
	for key, val := range params {
		union[key] = val
		keys = append(keys, key)
	}

	// sort sys params and api params by key
	sort.Strings(keys)

	var message bytes.Buffer
	message.WriteString(apiPath)
	for _, key := range keys {
		message.WriteString(fmt.Sprintf("%s%s", key, union[key]))
	}

	hash := hmac.New(sha256.New, []byte(appSecret))
	hash.Write(message.Bytes())
	return strings.ToUpper(hex.EncodeToString(hash.Sum(nil)))
}

func (mw *ApiMiddleware) CheckResponseStatus(rfToken string, responceBody map[string]interface{}) bool {
	if code, ok := responceBody["code"].(string); ok && code == utils.IllegalAccessToken {
		mw.RefreshLazadaAccessToken(rfToken)
		return false
	}
	return true
}
