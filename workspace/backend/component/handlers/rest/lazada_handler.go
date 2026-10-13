package rest

import (
	"main/component/services"
	"main/middleware"

	"github.com/gin-gonic/gin"
)

type LazadaHandler struct {
	ApiMiddleware *middleware.ApiMiddleware
	LazadaService services.LazadaService
}

func (h *LazadaHandler) Router(r *gin.RouterGroup) *gin.RouterGroup {
	return r
}
