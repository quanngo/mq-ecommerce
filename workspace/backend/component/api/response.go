package api

import (
	"fmt"
	"main/i18n"
	"main/utils"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// apiResponse is the envelope of every API response. On errors `message`
// carries the stable, language-neutral error code that clients switch on, and
// `error_text` the same error as text in the request's language (EC-7, see
// package i18n), for clients that have no wording of their own for it.
type apiResponse struct {
	Code      int         `json:"code"`
	Message   string      `json:"message"`
	ErrorText string      `json:"error_text,omitempty"`
	Data      interface{} `json:"data,omitempty"`
}

func getMessage(code int) string {
	strMessage := ""
	switch code {
	case http.StatusOK:
		strMessage = "Success"
	case http.StatusBadRequest:
		strMessage = "Bad request"
	case http.StatusUnauthorized:
		strMessage = "Unauthorized"
	case http.StatusForbidden:
		strMessage = "You don't have permission to access"
	case http.StatusInternalServerError:
		strMessage = "Internal server error"
	case http.StatusBadGateway:
		strMessage = "Bad gateway"
	}
	return strMessage
}

// errorText translates an error code ("error.<CODE>"), falling back to the
// generic text for the status ("status.<code>"), or "" when neither exists.
func errorText(ctx *gin.Context, code int, errorCode string) string {
	locale := i18n.Locale(ctx)
	if errorCode != "" {
		if s, ok := i18n.Lookup(locale, "error."+errorCode); ok {
			return s
		}
	}
	s, _ := i18n.Lookup(locale, "status."+strconv.Itoa(code))
	return s
}

func Ok(ctx *gin.Context, data ...interface{}) {
	apiResp := apiResponse{
		Code:    http.StatusOK,
		Message: getMessage(http.StatusOK),
		Data:    data,
	}
	ctx.JSON(http.StatusOK, apiResp)
}

func Error(ctx *gin.Context, code int) {
	apiResp := apiResponse{
		Code:      code,
		Message:   getMessage(code),
		ErrorText: errorText(ctx, code, ""),
	}
	utils.ShowErrorLogs(fmt.Errorf("%s", getMessage(code)))

	ctx.JSON(http.StatusOK, apiResp)
}

func ErrorWithMessage(ctx *gin.Context, code int, message string) {
	apiResp := apiResponse{
		Code:      code,
		Message:   message,
		ErrorText: errorText(ctx, code, message),
	}
	utils.ShowErrorLogs(fmt.Errorf("Error code: %d, meg: %s", code, message))

	ctx.JSON(http.StatusOK, apiResp)
}
