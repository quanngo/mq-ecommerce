package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"main/i18n"

	"github.com/gin-gonic/gin"
)

func errorBody(t *testing.T, locale string, send func(*gin.Context)) map[string]interface{} {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	if locale != "" {
		i18n.Set(c, locale)
	}
	send(c)
	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	return body
}

func TestErrorWithMessageTranslatesErrorText(t *testing.T) {
	en := errorBody(t, i18n.EN, func(c *gin.Context) { ErrorWithMessage(c, http.StatusForbidden, "FORBIDDEN") })
	if en["message"] != "FORBIDDEN" || en["error_text"] != "You don't have permission to do this." {
		t.Errorf("en body = %v", en)
	}
	vi := errorBody(t, "", func(c *gin.Context) { ErrorWithMessage(c, http.StatusForbidden, "FORBIDDEN") })
	if vi["message"] != "FORBIDDEN" || vi["error_text"] != "Bạn không có quyền thực hiện thao tác này." {
		t.Errorf("default body = %v", vi)
	}
}

func TestErrorTextFallsBackToStatus(t *testing.T) {
	body := errorBody(t, i18n.EN, func(c *gin.Context) { ErrorWithMessage(c, http.StatusBadRequest, "some free text") })
	if body["error_text"] != "Bad request" {
		t.Errorf("body = %v", body)
	}
	body = errorBody(t, i18n.VI, func(c *gin.Context) { Error(c, http.StatusInternalServerError) })
	if body["message"] != "Internal server error" || body["error_text"] != "Lỗi máy chủ" {
		t.Errorf("body = %v", body)
	}
	body = errorBody(t, i18n.EN, func(c *gin.Context) { ErrorWithMessage(c, 418, "TEAPOT") })
	if _, ok := body["error_text"]; ok {
		t.Errorf("unknown code and status should omit error_text: %v", body)
	}
}

func TestOkHasNoErrorText(t *testing.T) {
	body := errorBody(t, i18n.EN, func(c *gin.Context) { Ok(c, "x") })
	if _, ok := body["error_text"]; ok || body["code"].(float64) != 200 {
		t.Errorf("body = %v", body)
	}
}
