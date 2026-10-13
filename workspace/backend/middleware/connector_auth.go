package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"main/utils"
	"mime/multipart"
	"net/http"
)

var ConnectorAccessToken string

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
	ExpiredAt    string `json:"expiredAt"`
}

func (mid *ApiMiddleware) GetConnectorAccessToken() error {
	url := mid.cfg.Connector.Authenticator + "oauth/token"
	utils.ShowInfoLogs("[ApiMiddleware] [GetConnectorAccessToken] url: %s", url)

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	_ = writer.WriteField("grant_type", "password")
	_ = writer.WriteField("username", mid.cfg.Connector.User)
	_ = writer.WriteField("password", mid.cfg.Connector.Password)
	_ = writer.WriteField("client_id", "logistic_client")
	_ = writer.WriteField("client_secret", "secret")

	writer.Close()

	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		utils.ShowErrorLogs(fmt.Errorf("[ApiMiddleware] [GetConnectorAccessToken] error: %v", err))
		return err
	}
	defer resp.Body.Close()

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		utils.ShowErrorLogs(fmt.Errorf("[ApiMiddleware] [GetConnectorAccessToken] error: %v", err))
		return err
	}

	ConnectorAccessToken = tokenResp.AccessToken
	utils.ShowInfoLogs("[ApiMiddleware] [GetConnectorAccessToken] success: %s", ConnectorAccessToken)

	return nil
}

func (mid *ApiMiddleware) DoConnectorPOST(ctx context.Context, url string, payload any) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ConnectorAccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		err = mid.GetConnectorAccessToken()
		if err != nil {
			log.Println("[ApiMiddleware] [GetConnectorAccessToken] error: ", err.Error())
			return nil, err
		}
		return mid.DoConnectorPOST(ctx, url, payload)
	}

	respBody, respError := io.ReadAll(resp.Body)

	utils.ShowInfoLogs("[ApiMiddleware] [DoConnectorPOST] Response: %d, %s ", resp.StatusCode, string(respBody))

	return respBody, respError
}
