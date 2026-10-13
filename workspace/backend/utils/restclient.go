package utils

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

// Post request type
const (
	REQUEST_BODY_JSON  string = "JSON"
	REQUEST_BODY_XFORM string = "X_WWW_FORM"
)

type Result struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Data    []byte `json:"data"`
}

func GetAPI(url string, headers map[string]string) (int, map[string]interface{}) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Errorf("[EcommerceService] [GetAPI] error: %s", err.Error())
		return http.StatusInternalServerError, nil
	}
	// Set headers
	for k, v := range headers {
		req.Header.Add(k, v)
	}

	// Call API
	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("[EcommerceService] [GetAPI] error: %s", err.Error())
		return http.StatusInternalServerError, nil
	}

	if resp.StatusCode != http.StatusOK {
		log.Errorf("[EcommerceService] [GetAPI] API response %d", resp.StatusCode)
		scanner := bufio.NewScanner(resp.Body)
		scanner.Split(bufio.ScanBytes)
		for scanner.Scan() {
			fmt.Print(scanner.Text())
		}
		return resp.StatusCode, nil
	}
	defer resp.Body.Close()

	// Parse response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("[EcommerceService] error: %s", err.Error())
		return http.StatusInternalServerError, nil
	}

	var jResult map[string]interface{}
	json.Unmarshal(body, &jResult)
	return http.StatusOK, jResult
}

func PostAPI(url string, headers map[string]string, bodyRequest map[string]interface{}) (int, string) {
	client := &http.Client{}

	newBodyRequest, _ := json.Marshal(bodyRequest)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(newBodyRequest))
	if err != nil {
		log.Errorf("[EcommerceService] [PostAPI] error: %s", err.Error())
		return http.StatusInternalServerError, ""
	}
	// Set headers
	for k, v := range headers {
		req.Header.Add(k, v)
	}

	// Call API
	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("[EcommerceService] [PostAPI] error: %s", err.Error())
		return http.StatusInternalServerError, ""
	}
	if resp.StatusCode != http.StatusOK {
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		log.Errorf("[EcommerceService] [PostAPI] Req %s, Resp %s", string(newBodyRequest), buf.String())
		return resp.StatusCode, ""
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("[EcommerceService] [PostAPI] error: %s", err.Error())
		return http.StatusInternalServerError, ""
	}

	return http.StatusOK, string(body)
}

// DoRequest Create http request
func DoRequest(method string, url string, headers map[string]string, body map[string]interface{}, typeBody string) (int, []byte) {
	var err error
	var reqBody io.Reader = nil

	if headers == nil {
		headers = make(map[string]string)
	}

	if body != nil {
		if typeBody == REQUEST_BODY_JSON {
			headers["Content-Type"] = "application/json"
			jBody, err := json.Marshal(body)
			if err != nil {
				log.Errorf("[EcommerceService] [DoRequest] error: %s", err.Error())
				panic(err)
			}
			reqBody = bytes.NewBuffer(jBody)
		} else if typeBody == REQUEST_BODY_XFORM {
			var payload string
			for key, element := range body {
				payload = payload + fmt.Sprintf("%s=%s&", key, element)
			}
			payload = strings.TrimRight(payload, "&")
			reqBody = strings.NewReader(payload)
		}
	}

	timeout := time.Duration(5 * time.Second)
	client := &http.Client{
		Timeout: timeout,
	}

	req, err := http.NewRequest(method, url, reqBody)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	if err != nil {
		panic(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		logrus.Error(err)
	}

	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	return resp.StatusCode, respBody
}
