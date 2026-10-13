package services

import (
	"context"
	"fmt"
	"main/component/models"
	"main/middleware"
	"main/utils"
)

// ConnectorService persists orders and DO status updates to public.ecom_order.
// When connector.enable is true it also forwards them to Connector as before.
type ConnectorService interface {
	CreateLead(ctx context.Context, req models.CreateEcommerceRequest) error
	UpdateDOStatus(req models.UpdateDOStatusRequest) error
}

type ecomOrderStore interface {
	CreateOrder(ctx context.Context, orgId int, order models.CreateEcommerceRequest) (bool, error)
	UpdateDOStatus(ctx context.Context, req models.UpdateDOStatusRequest) (int64, error)
}

type ConnectorServiceImpl struct {
	cfg           *utils.Config
	mw            *middleware.ApiMiddleware
	ecomOrderRepo ecomOrderStore
}

func NewConnectorService(
	cfg *utils.Config,
	mw *middleware.ApiMiddleware,
	ecomOrderRepo ecomOrderStore,
) ConnectorService {
	return &ConnectorServiceImpl{
		cfg:           cfg,
		mw:            mw,
		ecomOrderRepo: ecomOrderRepo,
	}
}

func (s *ConnectorServiceImpl) CreateLead(ctx context.Context, req models.CreateEcommerceRequest) error {
	created, err := s.ecomOrderRepo.CreateOrder(ctx, s.cfg.Server.OrgId, req)
	if err != nil {
		return fmt.Errorf("[ConnectorServiceImpl] [CreateLead] [CreateOrder] %s - %s: %w", req.MarketPlace, req.OrderID, err)
	}
	if !created {
		// Already stored (and already forwarded to Connector if enabled at the time).
		utils.ShowInfoLogs("[ConnectorServiceImpl] [CreateLead] order already exists: %s - %s", req.MarketPlace, req.OrderID)
		return nil
	}
	utils.ShowInfoLogs("[ConnectorServiceImpl] [CreateLead] order saved: %s - %s, tracking: %s", req.MarketPlace, req.OrderID, req.TrackingCode)

	if !s.cfg.Connector.Enable {
		return nil
	}
	return s.createLeadConnector(ctx, req)
}

func (s *ConnectorServiceImpl) UpdateDOStatus(req models.UpdateDOStatusRequest) error {
	affected, err := s.ecomOrderRepo.UpdateDOStatus(context.Background(), req)
	if err != nil {
		return fmt.Errorf("[ConnectorServiceImpl] [UpdateDOStatus] order: %s, tracking: %s: %w", req.OrderID, req.TrackingCode, err)
	}
	if affected == 0 {
		utils.ShowInfoLogs("[ConnectorServiceImpl] [UpdateDOStatus] no order matched, order: %s, tracking: %s", req.OrderID, req.TrackingCode)
	}

	if !s.cfg.Connector.Enable {
		return nil
	}
	return s.updateDOStatusConnector(req)
}

func (s *ConnectorServiceImpl) createLeadConnector(ctx context.Context, req models.CreateEcommerceRequest) error {
	fullUrl := fmt.Sprintf("%sapi/v1/ecommerce", s.cfg.Connector.Host)
	utils.ShowInfoLogs("[ConnectorServiceImpl] [CreateLead] fullUrl: %s, req: %s", fullUrl, utils.MarshalToJson(req))

	response, err := s.mw.DoConnectorPOST(ctx, fullUrl, req)
	if err != nil {
		return err
	}
	resp := &models.ConnectorResponse{}
	if err := utils.UnmarshalFromString(string(response), resp); err != nil {
		return err
	}
	if resp.Code != 200 {
		return fmt.Errorf("%s", resp.Message)
	}

	return nil
}

func (s *ConnectorServiceImpl) updateDOStatusConnector(req models.UpdateDOStatusRequest) error {
	headers := make(map[string]string)
	bodyRequest := make(map[string]interface{})

	headers["Content-Type"] = "application/json"
	bodyRequest["tracking_code"] = req.TrackingCode
	bodyRequest["status"] = req.Status
	bodyRequest["so_status"] = req.SoStatus
	bodyRequest["order_id"] = req.OrderID

	fullUrl := fmt.Sprintf("%sapi/v1/callback/partner/ECOMMERCE", s.cfg.Connector.Callback)
	status, _ := utils.PostAPI(fullUrl, headers, bodyRequest)
	utils.ShowInfoLogs("[ConnectorServiceImpl] [UpdateDOStatus] fullUrl: %s, body: %s, status: %d", fullUrl, utils.MarshalToJson(bodyRequest), status)
	if status != 200 {
		return fmt.Errorf("Update DO status failed with status code: %d", status)
	}

	return nil
}
