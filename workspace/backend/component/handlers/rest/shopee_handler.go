package rest

import (
	"main/component/api"
	models "main/component/models/shopee_model"
	"main/component/services"
	"main/middleware"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ShopeeHandler struct {
	ApiMiddleware *middleware.ApiMiddleware
	ShopeeService services.ShopeeService
}

func (h *ShopeeHandler) Router(r *gin.RouterGroup) *gin.RouterGroup {
	r.POST("webhook", h.ShopeeWebhook)
	return r
}

func (h *ShopeeHandler) ShopeeWebhook(ctx *gin.Context) {
	data := models.OrderWebhook{}
	err := ctx.ShouldBindJSON(&data)
	if err == nil {
		respData := h.ShopeeService.OrderStatusPush(ctx, data)
		api.Ok(ctx, respData)
	} else {
		api.ErrorWithMessage(ctx, http.StatusBadRequest, err.Error())
	}
}
