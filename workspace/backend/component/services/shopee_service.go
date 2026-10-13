package services

import (
	"context"
	shopee_model "main/component/models/shopee_model"
	"main/middleware"
	"main/utils"
)

type ShopeeService interface {
	OrderStatusPush(ctx context.Context, orderStatus shopee_model.OrderWebhook) bool
}

type ShopeeServiceImpl struct {
	connectorService ConnectorService
	cfg              *utils.Config
	mw               *middleware.ApiMiddleware
}

func NewShopeeService(cfg *utils.Config, mw *middleware.ApiMiddleware, connectorService ConnectorService) ShopeeService {
	return &ShopeeServiceImpl{
		connectorService: connectorService,
		cfg:              cfg,
		mw:               mw,
	}
}

func (s *ShopeeServiceImpl) OrderStatusPush(ctx context.Context, orderStatus shopee_model.OrderWebhook) bool {
	switch orderStatus.Data.Status {
	case "UNPAID":
	case "READY_TO_SHIP":
	case "PROCESSED":
	case "SHIPPED":
	case "TO_CONFIRM_RECEIVE":
	case "COMPLETED":
	case "END":
	}
	return true
}

// func (s *ShopeeServiceImpl) GetOrderDetail(ctx context.Context,)
