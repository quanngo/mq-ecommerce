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
	"main/utils"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

var ShopeeAccessToken string

func (mid *ApiMiddleware) RefreshShopeeAccessToken() error {
	baseUrl := "https://partner.shopeemobile.com"
	path := "/api/v2/auth/access_token/get"
	params := url.Values{}
	params.Add("sign", GenerateSignShopee(path))
	params.Add("partner_id", strconv.Itoa(utils.AppConfig.Ecommerce.ShopeeConfig.PartnerId))
	params.Add("timestamp", strconv.FormatInt(time.Now().Unix(), 10))
	params.Add("refresh_token", utils.AppConfig.Ecommerce.ShopeeConfig.RefreshToken)
	headers := map[string]string{
		"Content-Type": "application/json",
	}
	m := map[string]interface{}{
		"refresh_token": utils.AppConfig.Ecommerce.ShopeeConfig.RefreshToken,
		"partner_id":    strconv.Itoa(utils.AppConfig.Ecommerce.ShopeeConfig.PartnerId),
		"shop_id":       strconv.Itoa(utils.AppConfig.Ecommerce.ShopeeConfig.ShopId),
	}
	_, result := utils.PostAPI(baseUrl+path+"?"+params.Encode(), headers, m)
	var respMap map[string]interface{}
	b, _ := json.Marshal(result)
	json.Unmarshal(b, &respMap)
	ShopeeAccessToken = respMap["access_token"].(string)
	return nil
}

func (mid *ApiMiddleware) DoShopeeGet(ctx context.Context, params string, path string) ([]byte, error) {
	baseUrl := utils.AppConfig.Ecommerce.ShopeeConfig.Url + path + params + "&sign=" + GenerateSignShopee(path)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseUrl, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		mid.RefreshShopeeAccessToken()
		return mid.DoShopeeGet(ctx, params, path)
	}
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return respBody, nil
}
func (mid *ApiMiddleware) DoShopeePost(ctx context.Context, params string, path string, payload any) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	baseUrl := utils.AppConfig.Ecommerce.ShopeeConfig.Url + path + params + "&sign=" + GenerateSignShopee(path)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseUrl, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		mid.RefreshShopeeAccessToken()
		return mid.DoShopeePost(ctx, params, path, payload)
	}
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return respBody, nil
}
func GenerateSignShopee(path string) string {
	timest := strconv.FormatInt(time.Now().Unix(), 10)
	baseString := fmt.Sprintf("%s%s%s", strconv.Itoa(utils.AppConfig.Ecommerce.ShopeeConfig.PartnerId), path, timest)
	h := hmac.New(sha256.New, []byte(utils.AppConfig.Ecommerce.ShopeeConfig.PartnerKey))
	h.Write([]byte(baseString))
	sign := hex.EncodeToString(h.Sum(nil))
	return sign
}
