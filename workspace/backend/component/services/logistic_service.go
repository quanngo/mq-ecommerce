package services

import (
	"context"
	"encoding/json"
	"fmt"
	"main/component/models"
	"main/middleware"
	"main/utils"
)

type LogisticService interface {
	CheckProductQuantity(ctx context.Context, req models.LogisticInventoryRequest) bool
	GetProductQuantity(ctx context.Context, req models.LogisticInventoryRequest) (*models.LogisticInventoryResponse, error)
}

type LogisticServiceImpl struct {
	cfg              *utils.Config
	mw               *middleware.ApiMiddleware
	connectorService ConnectorService
}

func NewLogisticService(cfg *utils.Config, mw *middleware.ApiMiddleware, connectorService ConnectorService) LogisticService {
	return &LogisticServiceImpl{
		cfg:              cfg,
		mw:               mw,
		connectorService: connectorService,
	}
}

func (s *LogisticServiceImpl) CheckProductQuantity(ctx context.Context, req models.LogisticInventoryRequest) bool {
	if !s.cfg.Logistic.Enable {
		return true
	}

	inventory, err := s.GetProductQuantity(ctx, req)
	if err != nil {
		utils.ShowErrorLogs(fmt.Errorf("[LogisticServiceImpl] [CheckProductQuantity] [GetProductQuantity] error: %s , shopID: %s , marketPlace: %s , product: %s", err.Error(), req.ShopID, req.MarketPlace, req.PartnerCode))
		return false
	}

	if inventory == nil || len(inventory.Data) == 0 || len(inventory.Data[0]) == 0 {
		utils.ShowErrorLogs(fmt.Errorf("[LogisticServiceImpl] [CheckProductQuantity] inventory data is empty: shopID: %s , marketPlace: %s", req.ShopID, req.MarketPlace))
		return false
	}

	if inventory.Data[0][0].AvailableQuantity < req.Quantity {
		utils.ShowErrorLogs(fmt.Errorf("[LogisticServiceImpl] [CheckProductQuantity] [GetProductQuantity] inventory not enough: %s , shopID: %s , marketPlace: %s", req.PartnerCode, req.ShopID, req.MarketPlace))
		return false
	}

	return true
}

func (s *LogisticServiceImpl) GetProductQuantity(ctx context.Context, req models.LogisticInventoryRequest) (*models.LogisticInventoryResponse, error) {
	headers := make(map[string]string)
	bodyRequest := make(map[string]interface{})

	headers["Content-Type"] = "application/json"
	headers["Client-Id"] = s.cfg.Logistic.ClientId
	headers["Client-Secret"] = s.cfg.Logistic.ClientSecret

	bodyRequest["partnerCode"] = req.PartnerCode
	bodyRequest["partnerProductCode"] = req.PartnerProductCode

	fullUrl := fmt.Sprintf("%sapi/v1/logistic/inventory", s.cfg.Logistic.Host)
	status, resp := utils.PostAPI(fullUrl, headers, bodyRequest)
	utils.ShowInfoLogs("[ConnectorServiceImpl] [GetProductQuantity] fullUrl: %s, body: %s, status: %d", fullUrl, utils.MarshalToJson(bodyRequest), status)
	if status != 200 {
		return nil, fmt.Errorf("Update DO status failed with status code: %d", status)
	}

	result := &models.LogisticInventoryResponse{}
	err := json.Unmarshal([]byte(resp), &result)
	if err != nil {
		utils.ShowInfoLogs("[ConnectorServiceImpl] [GetProductQuantity] fullUrl: %s, body: %s, error: %v", fullUrl, utils.MarshalToJson(bodyRequest), err)
		return nil, err
	}

	return result, nil
}
