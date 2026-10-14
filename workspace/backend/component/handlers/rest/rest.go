package rest

import (
	"main/component/services"
	"main/middleware"
	"main/utils"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type ApiServer struct {
	ApiMiddleware *middleware.ApiMiddleware
	Cfg           *utils.Config

	TiktokSvc services.TikTokService
	LazadaSvc services.LazadaService
	AuthSvc   services.AuthService
	UsersSvc  services.UserAdminService // nil = user-management routes not mounted
	OrdersSvc services.OrderService     // nil = /orders routes not mounted
}

func APIRoutes(app *gin.Engine, apiServer *ApiServer) {
	authCfg := apiServer.Cfg.Auth.WithDefaults()
	authCfg.FrontendURL = trimTrailingSlash(authCfg.FrontendURL)
	requireAuth := middleware.RequireAuth(apiServer.AuthSvc, authCfg.CookieName)
	sameOrigin := middleware.RequireSameOrigin(authCfg.FrontendURL)

	apiGroup := app.Group("/api/v1")

	authHandler := &AuthHandler{AuthSvc: apiServer.AuthSvc, Cfg: authCfg}
	authHandler.Router(apiGroup.Group("/auth"), requireAuth, sameOrigin)

	if apiServer.UsersSvc != nil {
		usersHandler := &UserAdminHandler{Svc: apiServer.UsersSvc}
		usersHandler.Router(apiGroup, requireAuth, sameOrigin)
	}

	if apiServer.OrdersSvc != nil {
		ordersHandler := &OrderHandler{Svc: apiServer.OrdersSvc}
		ordersHandler.Router(apiGroup.Group("/orders", requireAuth))
	}

	tiktokHandler := &TikTokHandler{TikTokService: apiServer.TiktokSvc}
	tiktokHandler.Router(apiGroup.Group("/tiktok", requireAuth))

	//internalGroup := apiGroup.Group("/internal")
	lazadaHandler := &LazadaHandler{LazadaService: apiServer.LazadaSvc}
	lazadaHandler.Router(apiGroup.Group("/lazada", requireAuth))
	app.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
