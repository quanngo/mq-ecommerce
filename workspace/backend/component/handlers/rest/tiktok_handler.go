package rest

import (
	"main/component/services"
	"main/middleware"

	"github.com/gin-gonic/gin"
)

type TikTokHandler struct {
	ApiMiddleware *middleware.ApiMiddleware
	TikTokService services.TikTokService
}

func (h *TikTokHandler) Router(r *gin.RouterGroup) *gin.RouterGroup {
	return r
}
