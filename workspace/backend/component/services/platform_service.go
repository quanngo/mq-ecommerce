package services

import (
	"context"
	"fmt"
	"main/component/models"
	tiktok_model "main/component/models/tiktok_model"
	"main/component/repositories"
	"main/utils"
	"strconv"
	"strings"
	"sync"
	"time"
)

type PlatformService interface {
	CronjobScanOrder()
	CronjobScanOrderPending()
	CronjobScanOrderRefund()
	CronjobScanOrderToShip()
	CronjobScanOrderPayment()
	CronjobScanStatementTransaction()
}

type PlatformServiceImpl struct {
	cfg                     *utils.Config
	connectorRepo           *repositories.ConnectorRepo
	tokenRepo               *repositories.TokenRepo
	marketPlaceMappingRepo  *repositories.MarketPlaceMappingRepo
	logOrderStatusRepo      *repositories.LogOrderStatusRepo
	logMarketPlaceOrderRepo *repositories.LogMarketPlaceOrderRepo
	connectorService        ConnectorService
	tiktokService           TikTokService
	lazadaService           LazadaService
}

func NewPlatformService(cfg *utils.Config,
	connectorRepo *repositories.ConnectorRepo,
	tokenRepo *repositories.TokenRepo,
	marketPlaceMappingRepo *repositories.MarketPlaceMappingRepo,
	logOrderStatusRepo *repositories.LogOrderStatusRepo,
	logMarketPlaceOrderRepo *repositories.LogMarketPlaceOrderRepo,
	connectorService ConnectorService,
	tiktokService TikTokService,
	lazadaService LazadaService,
) PlatformService {
	return &PlatformServiceImpl{
		cfg:                     cfg,
		connectorRepo:           connectorRepo,
		tokenRepo:               tokenRepo,
		marketPlaceMappingRepo:  marketPlaceMappingRepo,
		logOrderStatusRepo:      logOrderStatusRepo,
		logMarketPlaceOrderRepo: logMarketPlaceOrderRepo,
		connectorService:        connectorService,
		tiktokService:           tiktokService,
		lazadaService:           lazadaService,
	}
}

func (s *PlatformServiceImpl) platform(ctx context.Context,
	timeCreated time.Time, timeUpdated time.Time,
	shop models.MarketToken,
	warehouse *models.MarketPlaceMapping,
	mapProducts map[string]models.MarketPlaceMapping) ([]models.UpdateDatabaseOrder,
	[]models.CreateEcommerceRequest,
	[]models.UpdateDOStatusRequest,
	error) {

	switch shop.MarketPlace {
	case utils.TikTok:
		newLeadStatues, newLeads, updateDOs, err := s.tiktokService.ScanOrder(ctx, timeCreated.UTC(), timeUpdated.UTC(), shop, warehouse, mapProducts)
		if err != nil {
			utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [platform] [ScanOrder] err: %s", err.Error()))
			return nil, nil, nil, err
		}

		return newLeadStatues, newLeads, updateDOs, nil

	case utils.Lazada:
		newLeadStatues, newLeads, updateDOs, err := s.lazadaService.ScanOrder(ctx, timeCreated.UTC(), timeUpdated.UTC(), shop, warehouse, mapProducts)
		if err != nil {
			utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [platform] [ScanOrder] err: %s", err.Error()))
			return nil, nil, nil, err
		}

		return newLeadStatues, newLeads, updateDOs, nil
	default:
		return nil, nil, nil, fmt.Errorf("[PlatformServiceImpl] [platform] unsupported market place: %s", shop.MarketPlace)
	}
}

func (s *PlatformServiceImpl) platformPending(ctx context.Context,
	lstID []string,
	shop models.MarketToken,
	warehouse *models.MarketPlaceMapping,
	mapProducts map[string]models.MarketPlaceMapping) ([]models.CreateEcommerceRequest, []models.RequestUpdateLogOrderStatus, error) {

	switch shop.MarketPlace {
	case utils.TikTok:
		newLeads, updateOrderLogs, err := s.tiktokService.ScanOrderPending(ctx, lstID, shop, warehouse, mapProducts)
		if err != nil {
			utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [platformPending] [ScanOrder] err: %s", err.Error()))
			return nil, nil, err
		}

		return newLeads, updateOrderLogs, nil

	case utils.Lazada:
		newLeads, updateOrderLogs, err := s.lazadaService.ScanOrderPending(ctx, lstID, shop, warehouse, mapProducts)
		if err != nil {
			utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [platformPending] [ScanOrder] err: %s", err.Error()))
			return nil, nil, err
		}

		return newLeads, updateOrderLogs, nil

	default:
		return nil, nil, fmt.Errorf("[PlatformServiceImpl] [platformPending] unsupported market place: %s", shop.MarketPlace)
	}
}

func (s *PlatformServiceImpl) platformRefund(ctx context.Context,
	timeUpdated time.Time,
	shop models.MarketToken) ([]models.RequestUpdateLogOrderStatus, []models.UpdateDOStatusRequest, error) {

	switch shop.MarketPlace {
	case utils.TikTok:
		updateOrderLogs, updateDOStatusRequest, err := s.tiktokService.ScanOrderRefund(ctx, timeUpdated.UTC(), shop)
		if err != nil {
			utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [platformRefund] [ScanOrder] err: %s", err.Error()))
			return nil, nil, err
		}

		return updateOrderLogs, updateDOStatusRequest, nil

	case utils.Lazada:
		return nil, nil, nil

	default:
		return nil, nil, fmt.Errorf("[PlatformServiceImpl] [platformRefund] unsupported market place: %s", shop.MarketPlace)
	}
}

func (s *PlatformServiceImpl) scanShopOrder(ctx context.Context,
	shop models.MarketToken,
	timeCreated, timeUpdated time.Time) ([]models.UpdateDatabaseOrder,
	[]models.CreateEcommerceRequest,
	[]models.UpdateDOStatusRequest, error) {

	// Get products by shop ID
	products, err := s.marketPlaceMappingRepo.GetProducts(ctx, shop.MarketPlace, shop.ShopID, nil)
	if err != nil {
		utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [scanShopOrder] [GetProducts] err: %s", err.Error()))
		return nil, nil, nil, err
	}
	if len(products) == 0 {
		utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [scanShopOrder] [GetProducts] product not found in shop: %s", shop.ShopID))
		return nil, nil, nil, nil
	}

	// Mapping products by EcommerceID
	mapProducts := make(map[string]models.MarketPlaceMapping)
	for _, product := range products {
		prod := strings.Split(product.ConnectorID, "|")
		product.ProductName = prod[0]
		if len(prod) >= 2 {
			product.ProductSku = prod[1]
		}
		mapProducts[product.EcommerceID] = product
	}

	// Get warehouse by shop ID
	warehouse, err := s.marketPlaceMappingRepo.GetWarehouseByShopID(ctx, shop.MarketPlace, shop.ShopID)
	if err != nil {
		utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [scanShopOrder] [GetWarehouseByShopID] err: %s", err.Error()))
		return nil, nil, nil, err
	}
	if warehouse == nil {
		utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [scanShopOrder] [GetWarehouseByShopID] warehouse not found in shop: %s", shop.ShopID))
		return nil, nil, nil, err
	}

	// Scan market place
	newLeadStatues, newLeads, updateDOs, err := s.platform(ctx, timeCreated, timeUpdated, shop, warehouse, mapProducts)
	if err != nil {
		utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [scanShopOrder] [platform] err: %s", err.Error()))
		return nil, nil, nil, err
	}

	return newLeadStatues, newLeads, updateDOs, nil
}

func (s *PlatformServiceImpl) CronjobScanOrder() {

	defer func() {
		if r := recover(); r != nil {
			utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [CronjobScanOrder] panic: %v", r))
		}
	}()

	utils.ShowInfoLogs("[PlatformServiceImpl] [CronjobScanOrder] start cronjob scan platform order")

	// Get market place token
	marketPlace, err := s.tokenRepo.GetAll(context.Background())
	if err != nil {
		utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [CronjobScan] [GetMarketToken] err: %s", err.Error()))
		return
	}
	if marketPlace == nil {
		utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [CronjobScan] [GetMarketToken] market place is null"))
		return
	}

	var wg sync.WaitGroup
	const maxConcurrency = 10
	sem := make(chan struct{}, maxConcurrency)

	// Scan each shop from all market places
	for _, shop := range marketPlace {
		utils.ShowInfoLogs("[PlatformServiceImpl] [CronjobScanOrder] scan shop: %s - %s", shop.MarketPlace, shop.ShopID)

		if !shop.Enable {
			continue
		}

		ctx := context.Background()

		// Get lastest order time to scan
		shopTime, err := s.logOrderStatusRepo.GetLastestOrderTime(ctx, shop.MarketPlace, shop.ShopID)
		if err != nil {
			utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [CronjobScanOrder] [GetLastestOrderTime] err: %s", err.Error()))
			continue
		}

		var createdAt, updatedAt time.Time
		if shopTime == nil || shopTime.CreatedAt.IsZero() || shopTime.UpdatedAt.IsZero() { // If shopTime is null, set timeCreated and timeUpdated to now - 2 hours
			now := time.Now().UTC().Add(-2 * time.Hour)
			createdAt = now
			updatedAt = now
		} else {
			createdAt = *shopTime.CreatedAt
			updatedAt = *shopTime.UpdatedAt
		}

		// Scan shop order on a separate goroutine for not blocking other
		wg.Add(1)
		sem <- struct{}{} // acquire
		go func(shop models.MarketToken) {
			defer wg.Done()
			defer func() { <-sem }() // release

			defer func() {
				if r := recover(); r != nil {
					utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [CronjobScanOrder] [scanShop] panic shop %s - %s: %v", shop.MarketPlace, shop.ShopID, r))
				}
			}()

			if err := s.scanShop(ctx, shop, createdAt, updatedAt); err != nil {
				utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [CronjobScanOrder] [scanShop] shop %s - %s error: %v", shop.MarketPlace, shop.ShopID, err))
				return
			}
		}(shop)

	}

	wg.Wait()

	utils.ShowInfoLogs("[PlatformServiceImpl] [CronjobScanOrder] finish cronjob scan platform order")
}

func (s *PlatformServiceImpl) scanShop(ctx context.Context, shop models.MarketToken, createdAt, updatedAt time.Time) error {
	utils.ShowInfoLogs("[PlatformServiceImpl] [scanShop] shop: %s - %s", shop.MarketPlace, shop.ShopID)

	// Scan shop order from createdAt to updatedAt
	newLeadStatuses, newLeads, updateDOs, err := s.scanShopOrder(ctx, shop, createdAt, updatedAt)
	if err != nil {
		utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [scanShopOrder] err: %s", err.Error()))
		return err
	}

	// Update lead statuses
	s.handleLeadStatuses(ctx, shop, newLeadStatuses)

	// Create new leads to Connector
	s.handleNewLeads(ctx, shop, newLeads)

	// Update Connector DO statuses
	s.handleUpdateDOs(shop, updateDOs)

	utils.ShowInfoLogs("[PlatformServiceImpl] [scanShop] finish shop: %s - %s", shop.MarketPlace, shop.ShopID)

	return nil
}

func (s *PlatformServiceImpl) handleLeadStatuses(ctx context.Context, shop models.MarketToken, statuses []models.UpdateDatabaseOrder) {
	for _, status := range statuses {
		utils.ShowInfoLogs("[PlatformServiceImpl] [NewLeadStatus] shop: %s , status: %s",
			shop.ShopID, utils.MarshalNoEscape(status))

		if err := s.logOrderStatusRepo.InsertOrderStatus(ctx, status); err != nil {
			utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [InsertOrderStatus] shop: %s , status: %s , error: %s",
				shop.ShopID, utils.MarshalNoEscape(status), err.Error()))
		}
	}
}

func (s *PlatformServiceImpl) handleNewLeads(ctx context.Context, shop models.MarketToken, orders []models.CreateEcommerceRequest) {
	for _, order := range orders {
		utils.ShowInfoLogs("[PlatformServiceImpl] [NewLead] shop: %s , order: %s",
			shop.ShopID, utils.MarshalNoEscape(order))

		if err := s.logMarketPlaceOrderRepo.CreateLogOrder(ctx, s.cfg.Server.OrgId, order); err != nil {
			utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [handleNewLeads] [CreateLogOrder] shop: %s , order: %s , error: %s",
				shop.ShopID, utils.MarshalNoEscape(order), err.Error()))
			continue
		}

		if order.Status == utils.LeadCreated {
			order.RawOrder = utils.MarshalToJson(order.RawOrderByte)

			if err := s.connectorService.CreateLead(ctx, order); err != nil {
				utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [handleNewLeads] [CreateLead] err: %s", err.Error()))
			}
		}
	}
}

func (s *PlatformServiceImpl) handleNewLeadPendings(ctx context.Context, shop models.MarketToken, orders []models.CreateEcommerceRequest) {
	for _, order := range orders {
		utils.ShowInfoLogs("[PlatformServiceImpl] [handleNewLeadPendings] shop: %s , order: %s",
			shop.ShopID, utils.MarshalNoEscape(order))

		if order.Status == utils.LeadCreated {
			order.RawOrder = utils.MarshalToJson(order.RawOrderByte)

			if err := s.connectorService.CreateLead(ctx, order); err != nil {
				utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [handleNewLeadPendings] [CreateLead] err: %s", err.Error()))
			}
		}
	}
}

func (s *PlatformServiceImpl) handleUpdateDOs(shop models.MarketToken, updateDOs []models.UpdateDOStatusRequest) {
	for _, updateDO := range updateDOs {
		utils.ShowInfoLogs("[PlatformServiceImpl] [UpdateDO] shop: %s , updateDO: %s",
			shop.ShopID, utils.MarshalNoEscape(updateDO))

		updateDO.MarketPlace = shop.MarketPlace

		err := s.logOrderStatusRepo.UpdateOrderStatus(context.Background(), shop.MarketPlace, shop.ShopID, updateDO.OrderID, updateDO.Status, updateDO.UpdatedAt)
		if err != nil {
			utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [UpdateOrderStatus] err: %s order: %s", err.Error(), utils.MarshalToJson(updateDO)))
		}

		if err := s.connectorService.UpdateDOStatus(updateDO); err != nil {
			utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [UpdateDOStatus] err: %s order: %s", err.Error(), utils.MarshalToJson(updateDO)))
		}
	}
}

func (s *PlatformServiceImpl) handleUpdateOrderLogs(ctx context.Context, orders []models.RequestUpdateLogOrderStatus) {
	for _, order := range orders {
		utils.ShowInfoLogs("[PlatformServiceImpl] [handleUpdateOrderLogs] updateDO: %s", utils.MarshalNoEscape(order))

		if err := s.logMarketPlaceOrderRepo.UpdateOrderStatus(ctx, order); err != nil {
			utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [UpdateDOStatus] err: %s", err.Error()))
		}
	}
}

func (s *PlatformServiceImpl) CronjobScanOrderPending() {

	defer func() {
		if r := recover(); r != nil {
			utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [CronjobScanOrderPending] panic: %v", r))
		}
	}()

	utils.ShowInfoLogs("[PlatformServiceImpl] [CronjobScanOrderPending] start cronjob scan platform order")

	// Get market place token
	marketPlace, err := s.tokenRepo.GetAll(context.Background())
	if err != nil {
		utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [CronjobScanOrderPending] [GetMarketToken] err: %s", err.Error()))
		return
	}
	if marketPlace == nil {
		utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [CronjobScanOrderPending] [GetMarketToken] market place is null"))
		return
	}

	var wg sync.WaitGroup
	const maxConcurrency = 10
	sem := make(chan struct{}, maxConcurrency)

	// Scan each shop from all market places
	for _, shop := range marketPlace {
		utils.ShowInfoLogs("[PlatformServiceImpl] [CronjobScanOrderPending] scan shop: %s - %s", shop.MarketPlace, shop.ShopID)

		if !shop.Enable {
			continue
		}

		ctx := context.Background()

		// Get Order Pending
		orders, err := s.logMarketPlaceOrderRepo.GetLogOrder(ctx, models.GetLogMarketPlaceOrder{
			MarketPlace: shop.MarketPlace,
			ShopID:      shop.ShopID,
			Status:      utils.LeadPending,
		})
		if err != nil {
			utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [CronjobScanOrderPending] [GetLogOrder] err: %s", err.Error()))
			continue
		}
		if len(orders) == 0 {
			utils.ShowInfoLogs("[PlatformServiceImpl] [CronjobScanOrderPending] [GetLogOrder] no pending orders for shop: %s - %s", shop.MarketPlace, shop.ShopID)
			continue
		}

		var lstID []string
		for _, order := range orders {
			lstID = append(lstID, order.OrderID)
		}

		// Scan shop order on a separate goroutine for not blocking other
		wg.Add(1)
		sem <- struct{}{} // acquire
		go func(shop models.MarketToken) {
			defer wg.Done()
			defer func() { <-sem }() // release

			defer func() {
				if r := recover(); r != nil {
					utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [CronjobScanOrderPending] [scanShopPending] panic shop %s - %s: %v", shop.MarketPlace, shop.ShopID, r))
				}
			}()

			if err := s.scanShopPending(ctx, shop, lstID); err != nil {
				utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [CronjobScanOrderPending] [scanShopPending] shop %s - %s error: %v", shop.MarketPlace, shop.ShopID, err))
				return
			}
		}(shop)

	}

	wg.Wait()

	utils.ShowInfoLogs("[PlatformServiceImpl] [CronjobScanOrderPending] finish cronjob scan platform order")
}

func (s *PlatformServiceImpl) scanShopPending(ctx context.Context, shop models.MarketToken, lstID []string) error {
	utils.ShowInfoLogs("[PlatformServiceImpl] [scanShop] shop: %s - %s", shop.MarketPlace, shop.ShopID)

	// Scan shop order from createdAt to updatedAt
	newLeads, updateOrderLogs, err := s.scanShopOrderPending(ctx, shop, lstID)
	if err != nil {
		utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [scanShopOrderPending] err: %s", err.Error()))
		return err
	}

	// Create new leads to Connector
	s.handleNewLeadPendings(ctx, shop, newLeads)

	// Update log order statuses
	s.handleUpdateOrderLogs(ctx, updateOrderLogs)

	utils.ShowInfoLogs("[PlatformServiceImpl] [scanShopOrderPending] finish shop: %s - %s", shop.MarketPlace, shop.ShopID)

	return nil
}

func (s *PlatformServiceImpl) scanShopOrderPending(ctx context.Context,
	shop models.MarketToken,
	lstID []string) ([]models.CreateEcommerceRequest, []models.RequestUpdateLogOrderStatus, error) {

	// Get products by shop ID
	products, err := s.marketPlaceMappingRepo.GetProducts(ctx, shop.MarketPlace, shop.ShopID, nil)
	if err != nil {
		utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [scanShopOrderPending] [GetProducts] err: %s", err.Error()))
		return nil, nil, err
	}
	if len(products) == 0 {
		utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [scanShopOrderPending] [GetProducts] product not found in shop: %s", shop.ShopID))
		return nil, nil, nil
	}

	// Mapping products by EcommerceID
	mapProducts := make(map[string]models.MarketPlaceMapping)
	for _, product := range products {
		prod := strings.Split(product.ConnectorID, "|")
		product.ProductName = prod[0]
		if len(prod) >= 2 {
			product.ProductSku = prod[1]
		}
		mapProducts[product.EcommerceID] = product
	}

	// Get warehouse by shop ID
	warehouse, err := s.marketPlaceMappingRepo.GetWarehouseByShopID(ctx, shop.MarketPlace, shop.ShopID)
	if err != nil {
		utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [scanShopOrderPending] [GetWarehouseByShopID] err: %s", err.Error()))
		return nil, nil, err
	}
	if warehouse == nil {
		utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [scanShopOrderPending] [GetWarehouseByShopID] warehouse not found in shop: %s", shop.ShopID))
		return nil, nil, err
	}

	// Scan market place
	newLeads, updateOrderLogs, err := s.platformPending(ctx, lstID, shop, warehouse, mapProducts)
	if err != nil {
		utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [scanShopOrderPending] [platform] err: %s", err.Error()))
		return nil, nil, err
	}

	return newLeads, updateOrderLogs, nil
}

func (s *PlatformServiceImpl) CronjobScanOrderRefund() {

	defer func() {
		if r := recover(); r != nil {
			utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [CronjobScanOrderRefund] panic: %v", r))
		}
	}()

	utils.ShowInfoLogs("[PlatformServiceImpl] [CronjobScanOrderRefund] start cronjob scan platform order")

	// Get market place token
	marketPlace, err := s.tokenRepo.GetAll(context.Background())
	if err != nil {
		utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [CronjobScanOrderRefund] [GetMarketToken] err: %s", err.Error()))
		return
	}
	if marketPlace == nil {
		utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [CronjobScanOrderRefund] [GetMarketToken] market place is null"))
		return
	}

	var wg sync.WaitGroup
	const maxConcurrency = 10
	sem := make(chan struct{}, maxConcurrency)

	// Scan each shop from all market places
	for _, shop := range marketPlace {
		utils.ShowInfoLogs("[PlatformServiceImpl] [CronjobScanOrderRefund] scan shop: %s - %s", shop.MarketPlace, shop.ShopID)

		if !shop.Enable {
			continue
		}

		ctx := context.Background()

		// Get latest order time to scan
		shopTime, err := s.logMarketPlaceOrderRepo.GetLatestOrderRefundTime(ctx, utils.LeadRefund, shop.MarketPlace, shop.ShopID)
		if err != nil {
			utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [CronjobScanOrderRefund] [GetLastestOrderTime] err: %s", err.Error()))
			continue
		}

		var updatedAt time.Time
		if shopTime == nil || shopTime.CreatedAt.IsZero() || shopTime.UpdatedAt.IsZero() { // If shopTime is null, set timeCreated and timeUpdated to now - 2 hours
			now := time.Now().UTC().Add(-2 * time.Hour)
			updatedAt = now
		} else {
			updatedAt = shopTime.UpdatedAt.UTC()
		}

		// Scan shop order on a separate goroutine for not blocking other
		wg.Add(1)
		sem <- struct{}{} // acquire
		go func(shop models.MarketToken) {
			defer wg.Done()
			defer func() { <-sem }() // release

			defer func() {
				if r := recover(); r != nil {
					utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [CronjobScanOrderRefund] [scanShop] panic shop %s - %s: %v", shop.MarketPlace, shop.ShopID, r))
				}
			}()

			if err := s.scanShopRefund(ctx, shop, updatedAt); err != nil {
				utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [CronjobScanOrderRefund] [scanShop] shop %s - %s error: %v", shop.MarketPlace, shop.ShopID, err))
				return
			}
		}(shop)

	}

	wg.Wait()

	utils.ShowInfoLogs("[PlatformServiceImpl] [CronjobScanOrderRefund] finish cronjob scan platform order")
}

func (s *PlatformServiceImpl) scanShopRefund(ctx context.Context, shop models.MarketToken, updatedAt time.Time) error {
	utils.ShowInfoLogs("[PlatformServiceImpl] [scanShopRefund] shop: %s - %s", shop.MarketPlace, shop.ShopID)

	// Scan shop order from createdAt to updatedAt
	updateOrderLogs, updateDOStatusRequest, err := s.platformRefund(ctx, updatedAt.UTC(), shop)
	if err != nil {
		utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [scanShopRefund] [platformRefund] err: %s", err.Error()))
		return err
	}

	// Update log order statuses
	s.handleUpdateOrderLogs(ctx, updateOrderLogs)

	// Update Connector DO statuses
	s.handleUpdateDOs(shop, updateDOStatusRequest)

	utils.ShowInfoLogs("[PlatformServiceImpl] [scanShopRefund] finish shop: %s - %s", shop.MarketPlace, shop.ShopID)

	return nil
}

func appendLead(newLeads []models.CreateEcommerceRequest, newLead models.CreateEcommerceRequest, newLeadStatues []models.UpdateDatabaseOrder) ([]models.UpdateDatabaseOrder, []models.CreateEcommerceRequest) {
	newLeads = append(newLeads, newLead)
	return newLeadStatues, newLeads
}

func (s *PlatformServiceImpl) CronjobScanOrderToShip() {

	defer func() {
		if r := recover(); r != nil {
			utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [CronjobScanOrderToShip] panic: %v", r))
		}
	}()

	utils.ShowInfoLogs("[PlatformServiceImpl] [CronjobScanOrderToShip] start cronjob scan platform order")

	// Get market place token
	marketPlace, err := s.tokenRepo.GetAll(context.Background())
	if err != nil {
		utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [CronjobScanOrderToShip] [GetMarketToken] err: %s", err.Error()))
		return
	}
	if marketPlace == nil {
		utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [CronjobScanOrderToShip] [GetMarketToken] market place is null"))
		return
	}

	var wg sync.WaitGroup
	const maxConcurrency = 10
	sem := make(chan struct{}, maxConcurrency)

	// Scan each shop from all market places
	for _, shop := range marketPlace {
		utils.ShowInfoLogs("[PlatformServiceImpl] [CronjobScanOrderToShip] scan shop: %s - %s", shop.MarketPlace, shop.ShopID)

		if !shop.Enable {
			continue
		}

		ctx := context.Background()

		// Get lastest order time to scan
		orders, err := s.connectorRepo.GetOdEcomOrderDObyStatus(ctx, shop.MarketPlace, utils.DeliveryOrderStatusReadyToPick)
		if err != nil {
			utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [CronjobScanOrderToShip] [GetOdEcomOrderDObyStatus] err: %s", err.Error()))
			continue
		}

		// Scan shop order on a separate goroutine for not blocking other
		wg.Add(1)
		sem <- struct{}{} // acquire
		go func(shop models.MarketToken) {
			defer wg.Done()
			defer func() { <-sem }() // release

			defer func() {
				if r := recover(); r != nil {
					utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [CronjobScanOrderToShip] [scanShop] panic shop %s - %s: %v", shop.MarketPlace, shop.ShopID, r))
				}
			}()

			if err := s.scanShopToShip(ctx, shop, orders); err != nil {
				utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [CronjobScanOrderToShip] [scanShop] shop %s - %s error: %v", shop.MarketPlace, shop.ShopID, err))
				return
			}
		}(shop)

	}

	wg.Wait()

	utils.ShowInfoLogs("[PlatformServiceImpl] [CronjobScanOrderRefund] finish cronjob scan platform order")
}

func (s *PlatformServiceImpl) scanShopToShip(ctx context.Context, shop models.MarketToken, orders []models.OdEcomOrder) error {
	utils.ShowInfoLogs("[PlatformServiceImpl] [scanShopToShip] shop: %s - %s", shop.MarketPlace, shop.ShopID)

	// Scan shop order
	updateDOStatusRequest, err := s.platformToShip(ctx, shop, orders)
	if err != nil {
		utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [scanShopToShip] [platformToShip] err: %s", err.Error()))
		return err
	}

	// Update Connector DO statuses
	s.handleUpdateDOs(shop, updateDOStatusRequest)

	utils.ShowInfoLogs("[PlatformServiceImpl] [scanShopToShip] finish shop: %s - %s", shop.MarketPlace, shop.ShopID)

	return nil
}

func (s *PlatformServiceImpl) platformToShip(ctx context.Context,
	shop models.MarketToken,
	orders []models.OdEcomOrder) ([]models.UpdateDOStatusRequest, error) {

	switch shop.MarketPlace {
	case utils.TikTok:
		return nil, nil

	case utils.Lazada:
		updateDOStatusRequest, err := s.lazadaService.ScanOrderToShip(ctx, shop, orders)
		if err != nil {
			utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [platformToShip] [ScanOrder] err: %s", err.Error()))
			return nil, err
		}

		return updateDOStatusRequest, nil
	default:
		return nil, fmt.Errorf("[PlatformServiceImpl] [platformToShip] unsupported market place: %s", shop.MarketPlace)
	}
}

func (s *PlatformServiceImpl) CronjobScanOrderPayment() {

	defer func() {
		if r := recover(); r != nil {
			utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [CronjobScanOrderPayment] panic: %v", r))
		}
	}()

	utils.ShowInfoLogs("[PlatformServiceImpl] [CronjobScanOrderPayment] start cronjob scan platform order")

	// Get market place token
	marketPlace, err := s.tokenRepo.GetAll(context.Background())
	if err != nil {
		utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [CronjobScanOrderPayment] [GetMarketToken] err: %s", err.Error()))
		return
	}
	if marketPlace == nil {
		utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [CronjobScanOrderPayment] [GetMarketToken] market place is null"))
		return
	}

	var wg sync.WaitGroup
	const maxConcurrency = 10
	sem := make(chan struct{}, maxConcurrency)

	// Scan each shop from all market places
	for _, shop := range marketPlace {
		utils.ShowInfoLogs("[PlatformServiceImpl] [CronjobScanOrderPayment] scan shop: %s - %s", shop.MarketPlace, shop.ShopID)

		if !shop.Enable {
			continue
		}

		ctx := context.Background()

		// Get orders to scan
		orderId, err := s.marketPlaceMappingRepo.GetShopPaymentConfig(ctx, shop.MarketPlace, shop.ShopID)
		if err != nil {
			utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [CronjobScanOrderPayment] [GetOdEcomOrderDObyStatus] err: %s", err.Error()))
			continue
		}
		if len(orderId) == 0 {
			utils.ShowInfoLogs("[PlatformServiceImpl] [CronjobScanOrderPayment] [GetShopPaymentConfig] no orders to scan for shop: %s - %s", shop.MarketPlace, shop.ShopID)
			continue
		}
		id, err := strconv.Atoi(orderId)

		// Scan shop order on a separate goroutine for not blocking other
		wg.Add(1)
		sem <- struct{}{} // acquire
		go func(shop models.MarketToken) {
			defer wg.Done()
			defer func() { <-sem }() // release

			defer func() {
				if r := recover(); r != nil {
					utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [CronjobScanOrderPayment] [scanShop] panic shop %s - %s: %v", shop.MarketPlace, shop.ShopID, r))
				}
			}()

			if err := s.scanShopPayment(ctx, shop, id); err != nil {
				utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [CronjobScanOrderPayment] [scanShop] shop %s - %s error: %v", shop.MarketPlace, shop.ShopID, err))
				return
			}
		}(shop)

	}

	wg.Wait()

	utils.ShowInfoLogs("[PlatformServiceImpl] [CronjobScanOrderPayment] finish cronjob scan platform order")
}

func (s *PlatformServiceImpl) scanShopPayment(ctx context.Context, shop models.MarketToken, id int) error {
	utils.ShowInfoLogs("[PlatformServiceImpl] [scanShopPayment] shop: %s - %s", shop.MarketPlace, shop.ShopID)

	// Scan shop order
	orderPayments, lastID, err := s.platformPayment(ctx, shop, id)
	if err != nil {
		utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [scanShopPayment] [platformToShip] err: %s", err.Error()))
		return err
	}

	// Update Connector DO Payments
	s.handleUpdateDOPaymnents(ctx, shop, orderPayments, lastID)

	utils.ShowInfoLogs("[PlatformServiceImpl] [scanShopPayment] finish shop: %s - %s", shop.MarketPlace, shop.ShopID)

	return nil
}

func (s *PlatformServiceImpl) platformPayment(ctx context.Context,
	shop models.MarketToken,
	id int) ([]tiktok_model.Payment, int, error) {

	switch shop.MarketPlace {
	case utils.TikTok:
		orderPayments, id, err := s.tiktokService.ScanOrderPayment(ctx, shop, id)
		if err != nil {
			utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [platformPayment] [tiktokService] [ScanOrderPayment] err: %s", err.Error()))
			return nil, 0, err
		}

		return orderPayments, id, nil

	case utils.Lazada:
		orderPayments, id, err := s.lazadaService.ScanOrderPayment(ctx, shop, id)
		if err != nil {
			utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [platformPayment] [lazadaService] [ScanOrderPayment] err: %s", err.Error()))
			return nil, 0, err
		}

		return orderPayments, id, nil

	default:
		return nil, 0, fmt.Errorf("[PlatformServiceImpl] [platformPayment] unsupported market place: %s", shop.MarketPlace)
	}
}

func (s *PlatformServiceImpl) handleUpdateDOPaymnents(ctx context.Context, shop models.MarketToken, orderPayments []tiktok_model.Payment, id int) {
	if len(orderPayments) == 0 {
		utils.ShowInfoLogs("[PlatformServiceImpl] [handleUpdateDOPaymnents] no payments to update for shop: %s", shop.ShopID)
		return
	}

	for _, payment := range orderPayments {
		utils.ShowInfoLogs("[PlatformServiceImpl] [handleUpdateDOPaymnents] shop: %s , payment: %s",
			shop.ShopID, utils.MarshalNoEscape(payment))

		err := s.connectorRepo.UpdateOrderPayment(ctx, payment)
		if err != nil {
			utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [handleUpdateDOPaymnents] [UpdateOrderPayment] shop: %s , payment: %s , error: %s",
				shop.ShopID, utils.MarshalNoEscape(payment), err.Error()))
		}
	}

	err := s.marketPlaceMappingRepo.UpdateShopPaymentConfigValue(ctx, id, shop.MarketPlace, shop.ShopID)
	if err != nil {
		utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [handleUpdateDOPaymnents] [UpdateShopPaymentConfigValue] shop: %s , error: %s",
			shop.ShopID, err.Error()))
	}
}

func (s *PlatformServiceImpl) CronjobScanStatementTransaction() {
	defer func() {
		if r := recover(); r != nil {
			utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [CronjobScanStatementTransaction] panic: %v", r))
		}
	}()

	utils.ShowInfoLogs("[PlatformServiceImpl] [CronjobScanStatementTransaction] start scan transaction statement")

	// Get market place token
	marketPlace, err := s.tokenRepo.GetAll(context.Background())
	if err != nil {
		utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [CronjobScanStatementTransaction] [GetMarketToken] err: %s", err.Error()))
		return
	}
	if marketPlace == nil {
		utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [CronjobScanStatementTransaction] [GetMarketToken] market place is null"))
		return
	}

	var wg sync.WaitGroup
	const maxConcurrency = 10
	sem := make(chan struct{}, maxConcurrency)

	// Scan each shop from all market places
	for _, shop := range marketPlace {
		utils.ShowInfoLogs("[PlatformServiceImpl] [CronjobScanStatementTransaction] scan shop: %s - %s", shop.MarketPlace, shop.ShopID)

		if !shop.Enable {
			continue
		}

		ctx := context.Background()

		// Get orders to scan
		orders, err := s.connectorRepo.GetOrderIdForStatementTransaction(ctx, shop.MarketPlace, shop.ShopID)
		if err != nil {
			utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [CronjobScanStatementTransaction] [GetOrderIdForStatementTransaction] err: %s , shop: %s", err.Error(), shop.ShopID))
			continue
		}
		if len(orders) == 0 {
			utils.ShowInfoLogs("[PlatformServiceImpl] [CronjobScanStatementTransaction] [GetOrderIdForStatementTransaction] no orders to scan for shop: %s - %s", shop.MarketPlace, shop.ShopID)
			continue
		}

		// Scan shop order on a separate goroutine for not blocking other
		wg.Add(1)
		sem <- struct{}{} // acquire
		go func(shop models.MarketToken) {
			defer wg.Done()
			defer func() { <-sem }() // release

			defer func() {
				if r := recover(); r != nil {
					utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [CronjobScanStatementTransaction] [scanShopStatementTransaction] panic shop %s - %s: %v", shop.MarketPlace, shop.ShopID, r))
				}
			}()

			if err := s.scanShopStatementTransaction(ctx, shop, orders); err != nil {
				utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [CronjobScanStatementTransaction] [scanShopStatementTransaction] shop %s - %s error: %v", shop.MarketPlace, shop.ShopID, err))
				return
			}
		}(shop)

	}

	wg.Wait()

	utils.ShowInfoLogs("[PlatformServiceImpl] [ScanTransactionStatement] finish scan transaction statement")
}

func (s *PlatformServiceImpl) scanShopStatementTransaction(ctx context.Context, shop models.MarketToken, orders []models.OdEcomOrderID) error {
	utils.ShowInfoLogs("[PlatformServiceImpl] [scanShopStatementTransaction] shop: %s - %s", shop.MarketPlace, shop.ShopID)

	// Scan shop order
	statementTrans, err := s.platformStatementTransaction(ctx, shop, orders)
	if err != nil {
		utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [scanShopStatementTransaction] [platformStatementTransaction] err: %s", err.Error()))
		return err
	}

	s.handleInsertStatementTransactions(shop, statementTrans)

	utils.ShowInfoLogs("[PlatformServiceImpl] [scanShopStatementTransaction] finish shop: %s - %s", shop.MarketPlace, shop.ShopID)

	return nil
}

func (s *PlatformServiceImpl) platformStatementTransaction(ctx context.Context,
	shop models.MarketToken,
	orders []models.OdEcomOrderID) ([]models.EcomStatementTransaction, error) {

	switch shop.MarketPlace {
	case utils.TikTok:
		statementTrans, err := s.tiktokService.ScanOrderStatementTransaction(ctx, shop, orders)
		if err != nil {
			utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [platformPayment] [tiktokService] [ScanOrderPayment] err: %s", err.Error()))
			return nil, err
		}

		return statementTrans, nil

	case utils.Lazada:
		statementTrans, err := s.lazadaService.ScanTransactionStatement(ctx, shop, orders)
		if err != nil {
			utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [platformPayment] [lazadaService] [ScanOrderPayment] err: %s", err.Error()))
			return nil, err
		}
		return statementTrans, nil

	default:
		return nil, fmt.Errorf("[PlatformServiceImpl] [platformPayment] unsupported market place: %s", shop.MarketPlace)
	}
}

func (s *PlatformServiceImpl) handleInsertStatementTransactions(shop models.MarketToken, statementTrans []models.EcomStatementTransaction) {
	if len(statementTrans) == 0 {
		return
	}

	err := s.connectorRepo.InsertEcomStatementTransaction(statementTrans)
	if err != nil {
		utils.ShowErrorLogs(fmt.Errorf("[PlatformServiceImpl] [handleInsertStatementTransactions] [InsertEcomStatementTransaction] shop: %s , error: %s",
			shop.ShopID, err.Error()))
	}
}
