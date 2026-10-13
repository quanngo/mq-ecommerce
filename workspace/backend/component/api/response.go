package api

import (
	"fmt"
	"main/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

type apiResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
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
		Code:    code,
		Message: getMessage(code),
	}
	utils.ShowErrorLogs(fmt.Errorf("%s", getMessage(code)))

	ctx.JSON(http.StatusOK, apiResp)
}

func ErrorWithMessage(ctx *gin.Context, code int, message string) {
	apiResp := apiResponse{
		Code:    code,
		Message: message,
	}
	utils.ShowErrorLogs(fmt.Errorf("Error code: %d, meg: %s", code, message))

	ctx.JSON(http.StatusOK, apiResp)
}
