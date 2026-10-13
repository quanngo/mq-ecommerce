package services

import (
	"context"
	"encoding/json"
	"fmt"
	"main/component/models"
	lazada_model "main/component/models/lazada_model"
	tiktok_model "main/component/models/tiktok_model"
	"main/component/repositories"
	"main/middleware"
	"main/utils"
	"time"
)

const (
	LAZADA                         = "Lazada"
	LazadaStatusUnpaid             = "unpaid"
	LazadaStatusPending            = "pending"
	LazadaStatusPacked             = "packed"
	LazadaStatusReadyToShip        = "ready_to_ship"
	LazadaStatusReadyToShipPending = "ready_to_ship_pending"
	LazadaStatusShipped            = "shipped"
	LazadaStatusDelivered          = "delivered"
	LazadaStatusCanceled           = "canceled"
	LazadaStatusConfirmed          = "confirmed"
	LazadaStatusReturned           = "returned"
)

const (
	LAZADA_COD = "COD"
)

type LazadaService interface {
	CheckExpiredTokenLazada()
	ScanOrder(
		ctx context.Context,
		timeCreated, timeUpdated time.Time,
		shop models.MarketToken,
		warehouse *models.MarketPlaceMapping,
		mapProducts map[string]models.MarketPlaceMapping) ([]models.UpdateDatabaseOrder,
		[]models.CreateEcommerceRequest,
		[]models.UpdateDOStatusRequest, error)
	ScanOrderPending(
		ctx context.Context,
		lstIDs []string,
		shop models.MarketToken,
		warehouse *models.MarketPlaceMapping,
		mapProducts map[string]models.MarketPlaceMapping) ([]models.CreateEcommerceRequest, []models.RequestUpdateLogOrderStatus, error)
	ScanOrderToShip(
		ctx context.Context,
		shop models.MarketToken,
		orders []models.OdEcomOrder) ([]models.UpdateDOStatusRequest, error)
	ScanOrderPayment(
		ctx context.Context,
		shop models.MarketToken,
		id int) ([]tiktok_model.Payment, int, error)
	ScanTransactionStatement(ctx context.Context, shop models.MarketToken, orders []models.OdEcomOrderID) ([]models.EcomStatementTransaction, error)
}

type LazadaServiceImpl struct {
	cfg                     *utils.Config
	mw                      *middleware.ApiMiddleware
	connectorRepo           *repositories.ConnectorRepo
	tokenRepo               *repositories.TokenRepo
	marketPlaceMappingRepo  *repositories.MarketPlaceMappingRepo
	logMarketPlaceOrderRepo *repositories.LogMarketPlaceOrderRepo
	connectorService        ConnectorService
	logisticService         LogisticService
}

func NewLazadaService(cfg *utils.Config,
	mw *middleware.ApiMiddleware,
	connectorRepo *repositories.ConnectorRepo,
	tokenRepo *repositories.TokenRepo,
	marketPlaceMappingRepo *repositories.MarketPlaceMappingRepo,
	logMarketPlaceOrderRepo *repositories.LogMarketPlaceOrderRepo,
	connectorService ConnectorService,
	logisticService LogisticService) LazadaService {
	return &LazadaServiceImpl{
		cfg:                     cfg,
		mw:                      mw,
		connectorRepo:           connectorRepo,
		tokenRepo:               tokenRepo,
		marketPlaceMappingRepo:  marketPlaceMappingRepo,
		logMarketPlaceOrderRepo: logMarketPlaceOrderRepo,
		connectorService:        connectorService,
		logisticService:         logisticService,
	}
}

func (s *LazadaServiceImpl) CheckExpiredTokenLazada() {
	ctx := context.Background()
	lazada, err := s.tokenRepo.GetShopID(ctx, utils.Lazada)
	if err != nil {
		return
	}
	for _, shopID := range lazada {
		timeExpired := s.tokenRepo.GetExpireTime(ctx, utils.Lazada, shopID)
		if timeExpired.After(time.Now().Add(-1 * time.Hour)) {

			s.mw.RefreshLazadaAccessToken(shopID)
		}
	}
}

func getLazadaPaymentMethod(paymentMethod string) int {
	switch paymentMethod {
	// case LAZADA_COD:
	// 	return utils.PAYMENT_METHOD_COD
	default:
		return utils.PAYMENT_METHOD_ECOMMERCE
	}
}

func (s *LazadaServiceImpl) ScanOrder(
	ctx context.Context,
	timeCreated, timeUpdated time.Time,
	shop models.MarketToken,
	warehouse *models.MarketPlaceMapping,
	mapProducts map[string]models.MarketPlaceMapping) ([]models.UpdateDatabaseOrder,
	[]models.CreateEcommerceRequest,
	[]models.UpdateDOStatusRequest, error) {

	newLeadStatues := []models.UpdateDatabaseOrder{}
	newLeads := []models.CreateEcommerceRequest{}
	updateDOs := []models.UpdateDOStatusRequest{}

	utils.ShowInfoLogs("[LazadaServiceImpl] [ScanOrder] timeCreated: %v , timeUpdated: %v , shop: %s",
		timeCreated.UTC(), timeUpdated.UTC(), utils.MarshalToJson(shop))

	orders, err := s.fetchAllOrders(ctx, shop, timeUpdated)
	if err != nil {
		utils.ShowErrorLogs(fmt.Errorf("[LazadaServiceImpl] [ScanOrder] [GetOrders] order not found in shop: %s", err))
		return nil, nil, nil, err
	}

	for _, order := range orders {
		utils.ShowInfoLogs("[LazadaServiceImpl] [ScanOrder] [Order] Order: %s", utils.MarshalToJson(order))

		timeCreatedOrder, _ := time.Parse(utils.DATE_TIME_FORMAT_YYYY_MM_DDT_HH_MM_SS_GMT, order.CreatedAt)
		timeUpdatedOrder, _ := time.Parse(utils.DATE_TIME_FORMAT_YYYY_MM_DDT_HH_MM_SS_GMT, order.UpdatedAt)
		timeCreatedOrder = timeCreatedOrder.UTC()
		timeUpdatedOrder = timeUpdatedOrder.UTC()

		orderDetail, err := s.mw.LazadaGetOrderItems(ctx, order.OrderID, shop.AccessToken, shop.RefreshToken)
		if err != nil {
			utils.ShowInfoLogs("[LazadaServiceImpl] [ScanOrder] [LazadaGetOrderItems] error: %s , Order: %s", err.Error(), utils.MarshalToJson(order))
			continue
		}
		if orderDetail == nil {
			utils.ShowInfoLogs("[LazadaServiceImpl] [ScanOrder] [LazadaGetOrderItems] order is empty, Order: %s", utils.MarshalToJson(order))
			continue
		}

		// Skip Status
		if s.isSkipStatus(order) {
			continue
		}

		if order.Statuses[0] == LazadaStatusCanceled {
			utils.ShowInfoLogs("[LazadaServiceImpl] [ScanOrder] [LazadaGetOrderItems] order is cancelled, Order: %s", utils.MarshalToJson(orderDetail))
		}

		// Check order exists
		checkOrderExists, err := s.logMarketPlaceOrderRepo.CheckOrderExists(utils.Lazada, fmt.Sprint(order.OrderID))
		if err != nil {
			utils.ShowErrorLogs(fmt.Errorf("[LazadaServiceImpl] [ScanOrder] [CheckOrderExists] error %s , Order: %s", err.Error(), utils.MarshalToJson(order)))
			continue
		}

		// Create new order in Connector
		// Lazada only create new order when status is Packed
		if !checkOrderExists && order.Statuses[0] == LazadaStatusPacked {
			newLeadStatues, newLeads = s.newOrder(ctx, shop, order, timeCreatedOrder, orderDetail, mapProducts, warehouse, newLeads, newLeadStatues)
			continue
		}

		if timeUpdatedOrder.After(timeUpdated) {

			updateDOStatusRequest := models.UpdateDOStatusRequest{UpdatedAt: timeUpdatedOrder}

			if order.Statuses[0] == LazadaStatusConfirmed {
				updateDOStatusRequest.OrderID = fmt.Sprint(order.OrderID)
				updateDOStatusRequest.SoStatus = utils.SaleOrderCompleted
			} else {
				updateDOStatusRequest.OrderID = fmt.Sprint(order.OrderID)
				updateDOStatusRequest.TrackingCode = orderDetail.Data[0].TrackingCode
				updateDOStatusRequest.Status = order.Statuses[0]
			}

			updateDOs = append(updateDOs, updateDOStatusRequest)
		}
	}

	utils.ShowInfoLogs("[LazadaServiceImpl] [ScanOrder] total newLeadStatues: %d , total newLeads: %d , total updateDOs: %d", len(newLeadStatues), len(newLeads), len(updateDOs))

	return newLeadStatues, newLeads, updateDOs, nil
}

func (s *LazadaServiceImpl) newOrder(ctx context.Context,
	shop models.MarketToken,
	order lazada_model.Order,
	timeCreatedOrder time.Time,
	orderDetail *lazada_model.OrderItemResponse,
	mapProducts map[string]models.MarketPlaceMapping,
	warehouse *models.MarketPlaceMapping,
	newLeads []models.CreateEcommerceRequest,
	newLeadStatues []models.UpdateDatabaseOrder) ([]models.UpdateDatabaseOrder, []models.CreateEcommerceRequest) {

	newLeadStatues = append(newLeadStatues, models.UpdateDatabaseOrder{
		MarketPlace: utils.Lazada,
		ShopID:      shop.ShopID,
		OrderID:     fmt.Sprint(order.OrderID),
		Status:      order.Statuses[0],
		CreatedDate: timeCreatedOrder,
	})

	var (
		products    []models.ConnectorProduct
		totalAmount float64 = 0
		packageIDs  []string
		itemIDs     []int64
	)

	for _, order := range orderDetail.Data {
		utils.ShowInfoLogs("[LazadaServiceImpl] [CronjobUpdateConnector] [OrderDetail] OrderDetail: %s", utils.MarshalNoEscape(order))

		prod, exist := mapProducts[order.ProductID]
		if !exist {
			utils.ShowErrorLogs(fmt.Errorf("[LazadaServiceImpl] [CronjobUpdateConnector] [GetProducts] product not found: %s", shop.ShopID))
			continue
		}

		itemIDs = append(itemIDs, order.OrderItemID)

		packageIDs = append(packageIDs, order.PackageID)

		price := order.ItemPrice

		// Check and update quantity if product already in list
		found := false
		for i := range products {
			if products[i].Name == prod.ProductName {
				products[i].Quantity += prod.Quantity
				found = true
				break
			}
		}

		// Insert new product
		if !found {
			products = append(products, models.ConnectorProduct{
				Name:     prod.ProductName,
				Sku:      prod.ProductSku,
				Quantity: prod.Quantity,
				Price:    price / float64(prod.Quantity),
			})
		}

		totalAmount += price
	}

	newLead := models.CreateEcommerceRequest{
		Status:        utils.LeadPending,
		ShippingLabel: "",

		MarketPlace:   utils.Lazada,
		OrderID:       fmt.Sprint(order.OrderID),
		TrackingCode:  orderDetail.Data[0].TrackingCode,
		Name:          order.AddressShipping.FirstName,
		Phone:         order.AddressShipping.Phone,
		Address:       order.AddressShipping.Address1,
		Comment:       order.BuyerNote,
		Province:      order.AddressShipping.City,
		District:      order.AddressShipping.AddressDistrict,
		Subdistrict:   order.AddressShipping.Address2,
		Neighborhood:  order.AddressShipping.Address3,
		Products:      products,
		TotalAmount:   totalAmount,
		ShopID:        shop.ShopID,
		Warehouse:     warehouse.ConnectorID,
		PaymentMethod: getLazadaPaymentMethod(order.PaymentMethod),
		EnableAddress: false,

		// RawOrder:      utils.MarshalToJson(orderDetail[0]),
		RawOrderByte: orderDetail,
	}

	// Check inventory
	if !s.isInventoryAvailable(ctx, products, shop) {
		utils.ShowErrorLogs(fmt.Errorf("[LazadaServiceImpl] [ScanOrder] [CheckInventory] inventory not enough shop: %s , order: %s",
			shop.ShopID, utils.MarshalNoEscape(order)))
		newLeads = append(newLeads, newLead)
		return newLeadStatues, newLeads
	}

	// (Old version) Progress: Pack -> Print AWB -> Ready to Ship
	// (New version) Progress: Pack -> Print AWB
	// 1.Pack
	PackReq := lazada_model.PackRequest{
		PackOrderList: []lazada_model.PackOrder{
			{
				OrderItemList: itemIDs,
				OrderID:       order.OrderID,
			},
		},
		DeliveryType:     utils.Dropship,
		ShippingAllocate: utils.TFS,
	}
	_, err := s.mw.LazadaPack(ctx, PackReq, shop.AccessToken, shop.RefreshToken)
	if err != nil {
		utils.ShowInfoLogs("[LazadaServiceImpl] [ScanOrder] [newOrder] [Pack] pack fail, Detail Order %s , packageID: %s", utils.MarshalToJson(order), orderDetail.Data[0].PackageID)
		return appendLead(newLeads, newLead, newLeadStatues)
	}
	// 2.Print AWB
	awbData, err := s.mw.LazadaPrintAWB(ctx, orderDetail.Data[0].PackageID, shop.AccessToken, shop.RefreshToken)
	if err != nil {
		utils.ShowInfoLogs("[LazadaServiceImpl] [ScanOrder] [newOrder] [PrintAWB] error: %s, Detail Order %s , packageID: %s", err.Error(), utils.MarshalToJson(order), orderDetail.Data[0].PackageID)
		return appendLead(newLeads, newLead, newLeadStatues)
	}
	if awbData == nil {
		utils.ShowInfoLogs("[LazadaServiceImpl] [ScanOrder] [newOrder] [PrintAWB] print fail, Detail Order %s , packageID: %s", utils.MarshalToJson(order), orderDetail.Data[0].PackageID)
		return appendLead(newLeads, newLead, newLeadStatues)
	}
	if awbData.Result.ErrorMsg != "" {
		utils.ShowInfoLogs("[LazadaServiceImpl] [ScanOrder] [newOrder] [PrintAWB] print fail, Detail Order %s , packageID: %s , error: %s",
			utils.MarshalNoEscape(order), orderDetail.Data[0].PackageID, awbData.Result.ErrorMsg)
		return appendLead(newLeads, newLead, newLeadStatues)
	}
	// 3.Ready to Ship
	// packageIDs = utils.CleanDuplicated(packageIDs)
	// for _, packageID := range packageIDs {
	// 	rts, err := s.mw.LazadaReadyToShip(ctx, shop.AccessToken, shop.RefreshToken, packageID)
	// 	if err != nil {
	// 		utils.ShowInfoLogs("[LazadaServiceImpl] [ScanOrder] [newOrder] [ReadyToShip] error: %s, Detail Order %s , packageID: %s", err.Error(), utils.MarshalToJson(order), orderDetail.Data[0].PackageID)
	// 		return appendLead(newLeads, newLead, newLeadStatues)
	// 	}
	// 	if rts == nil {
	// 		utils.ShowInfoLogs("[LazadaServiceImpl] [ScanOrder] [newOrder] [ReadyToShip] rts is null, Detail Order %s , packageID: %s", utils.MarshalToJson(order), orderDetail.Data[0].PackageID)
	// 		return appendLead(newLeads, newLead, newLeadStatues)
	// 	}
	// 	if rts.Result.ErrorMsg != "" {
	// 		utils.ShowInfoLogs("[LazadaServiceImpl] [ScanOrder] [newOrder] [PrintAWB] print fail, Detail Order %s , packageID: %s , error: %s",
	// 			utils.MarshalNoEscape(order), orderDetail.Data[0].PackageID, awbData.Result.ErrorMsg)
	// 		return appendLead(newLeads, newLead, newLeadStatues)
	// 	}
	// }
	//

	newLead.Status = utils.LeadCreated
	newLead.ShippingLabel = awbData.Result.Data.PDFUrl

	utils.ShowInfoLogs("[LazadaServiceImpl] [ScanOrder] [CreateLead] newLead: %s", utils.MarshalNoEscape(newLead))
	return appendLead(newLeads, newLead, newLeadStatues)
}

func (s *LazadaServiceImpl) fetchAllOrders(ctx context.Context, shop models.MarketToken, updatedAt time.Time) ([]lazada_model.Order, error) {
	var (
		orders []lazada_model.Order
		offset = 0
		limit  = 100
	)

	for {
		result, err := s.mw.LazadaGetOrderList(ctx, updatedAt.Format(utils.DATE_TIME_FORMAT_YYYY_MM_DDT_HH_MM_SS), shop.AccessToken, shop.RefreshToken, offset, limit)
		if err != nil {
			utils.ShowErrorLogs(fmt.Errorf("[LazadaServiceImpl] [ScanOrder] [GetOrders] order not found in shop: %s , err: %w", shop.ShopID, err))
			return nil, err
		}
		if result == nil {
			utils.ShowErrorLogs(fmt.Errorf("[LazadaServiceImpl] [ScanOrder] [GetOrders] result is null in shop: %s", shop.ShopID))
			return nil, err
		}

		orders = append(orders, result.Data.Orders...)

		offset += limit

		if result.Data.CountTotal == 0 {
			break
		}
	}

	return orders, nil
}

func (s *LazadaServiceImpl) ScanOrderPending(
	ctx context.Context,
	lstIDs []string,
	shop models.MarketToken,
	warehouse *models.MarketPlaceMapping,
	mapProducts map[string]models.MarketPlaceMapping) ([]models.CreateEcommerceRequest, []models.RequestUpdateLogOrderStatus, error) {

	newLeads := []models.CreateEcommerceRequest{}
	updateOrderLogs := []models.RequestUpdateLogOrderStatus{}

	utils.ShowInfoLogs("[LazadaServiceImpl] [ScanOrderPending] lstID: %s , shop: %s",
		lstIDs, utils.MarshalToJson(shop))

	for _, id := range lstIDs {
		newLeads, updateOrderLogs = s.newOrderAfterPending(ctx, id, shop, mapProducts, warehouse, newLeads, updateOrderLogs)
	}
	return newLeads, updateOrderLogs, nil
}

func (s *LazadaServiceImpl) newOrderAfterPending(ctx context.Context,
	id string,
	shop models.MarketToken,
	mapProducts map[string]models.MarketPlaceMapping,
	warehouse *models.MarketPlaceMapping,
	newLeads []models.CreateEcommerceRequest,
	updateOrderLogs []models.RequestUpdateLogOrderStatus) ([]models.CreateEcommerceRequest, []models.RequestUpdateLogOrderStatus) {
	utils.ShowInfoLogs("[LazadaServiceImpl] [ScanOrderPending] [Order] Order: %s", id)

	order, err := s.mw.LazadaGetOrder(ctx, id, shop.AccessToken, shop.RefreshToken)
	if err != nil {
		utils.ShowErrorLogs(fmt.Errorf("[LazadaServiceImpl] [ScanOrderPending] [LazadaGetOrder] order not found in shop: %s", err))
		return newLeads, updateOrderLogs
	}
	if order == nil {
		utils.ShowErrorLogs(fmt.Errorf("[LazadaServiceImpl] [ScanOrderPending] [LazadaGetOrder] order detail is empty shop: %s , orderID: %s", shop.ShopID, id))
		return newLeads, updateOrderLogs
	}
	if order.Data.Statuses[0] == LazadaStatusCanceled {
		utils.ShowInfoLogs("[LazadaServiceImpl] [ScanOrderPending] [LazadaGetOrder] order is cancelled, Order: %s", utils.MarshalToJson(order))
		return newLeads, updateOrderLogs
	}

	orderDetail, err := s.mw.LazadaGetOrderItems(ctx, order.Data.OrderID, shop.AccessToken, shop.RefreshToken)
	if err != nil {
		utils.ShowInfoLogs("[LazadaServiceImpl] [ScanOrderPending] [LazadaGetOrderItems] error: %s , Order: %s", err.Error(), utils.MarshalToJson(order))
		return newLeads, updateOrderLogs
	}
	if orderDetail == nil {
		utils.ShowInfoLogs("[LazadaServiceImpl] [ScanOrderPending] [LazadaGetOrderItems] order is empty, Order: %s", utils.MarshalToJson(order))
		return newLeads, updateOrderLogs
	}

	var (
		products    []models.ConnectorProduct
		totalAmount float64 = 0
		packageIDs  []string
		itemIDs     []int64
	)

	utils.ShowInfoLogs("[LazadaServiceImpl] [ScanOrderPending] [OrderDetail] OrderDetail: %s", utils.MarshalNoEscape(orderDetail))

	for _, orderItem := range orderDetail.Data {
		utils.ShowInfoLogs("[LazadaServiceImpl] [ScanOrderPending] [OrderDetail] OrderItem: %s", utils.MarshalNoEscape(orderItem))

		prod, exist := mapProducts[orderItem.ProductID]
		if !exist {
			utils.ShowErrorLogs(fmt.Errorf("[LazadaServiceImpl] [ScanOrderPending] [GetProducts] product not found: %s", shop.ShopID))
			continue
		}

		packageIDs = append(packageIDs, orderItem.PackageID)
		itemIDs = append(itemIDs, orderItem.OrderItemID)

		price := orderItem.ItemPrice

		// Check and update quantity if product already in list
		found := false
		for i := range products {
			if products[i].Name == prod.ProductName {
				products[i].Quantity += prod.Quantity
				found = true
				break
			}
		}

		// Insert new product
		if !found {
			products = append(products, models.ConnectorProduct{
				Name:     prod.ProductName,
				Sku:      prod.ProductSku,
				Quantity: prod.Quantity,
				Price:    price / float64(prod.Quantity),
			})
		}

		totalAmount += price
	}

	// Check inventory
	if !s.isInventoryAvailable(ctx, products, shop) {
		utils.ShowErrorLogs(fmt.Errorf("[LazadaServiceImpl] [ScanOrderPending] [CheckInventory] inventory not enough shop: %s , order: %s",
			shop.ShopID, utils.MarshalNoEscape(order)))
		return newLeads, updateOrderLogs
	}

	// (Old version) Progress: Pack -> Print AWB -> Ready to Ship
	// (New version) Progress: Pack -> Print AWB
	// 1.Pack
	PackReq := lazada_model.PackRequest{
		PackOrderList: []lazada_model.PackOrder{
			{
				OrderItemList: itemIDs,
				OrderID:       order.Data.OrderID,
			},
		},
		DeliveryType:     utils.Dropship,
		ShippingAllocate: utils.TFS,
	}
	_, err = s.mw.LazadaPack(ctx, PackReq, shop.AccessToken, shop.RefreshToken)
	if err != nil {
		utils.ShowInfoLogs("[LazadaServiceImpl] [ScanOrderPending] [newOrderAfterPending] [Pack] pack fail, Detail Order %s , packageID: %s", utils.MarshalToJson(order), orderDetail.Data[0].PackageID)
		return newLeads, updateOrderLogs
	}
	// 2.Print AWB
	awbData, err := s.mw.LazadaPrintAWB(ctx, orderDetail.Data[0].PackageID, shop.AccessToken, shop.RefreshToken)
	if err != nil {
		utils.ShowInfoLogs("[LazadaServiceImpl] [ScanOrderPending] [newOrderAfterPending] [PrintAWB] error: %s, Detail Order %s , packageID: %s", err.Error(), utils.MarshalToJson(order), orderDetail.Data[0].PackageID)
		return newLeads, updateOrderLogs
	}
	if awbData == nil {
		utils.ShowInfoLogs("[LazadaServiceImpl] [ScanOrder] [newOrder] [newOrderAfterPending] print fail, Detail Order %s , packageID: %s", utils.MarshalToJson(order), orderDetail.Data[0].PackageID)
		return newLeads, updateOrderLogs
	}
	if awbData.Result.ErrorMsg != "" {
		utils.ShowInfoLogs("[LazadaServiceImpl] [ScanOrderPending] [newOrderAfterPending] [PrintAWB] print fail, Detail Order %s , packageID: %s , error: %s",
			utils.MarshalNoEscape(order), orderDetail.Data[0].PackageID, awbData.Result.ErrorMsg)
		return newLeads, updateOrderLogs
	}
	// 3.Ready to Ship
	// packageIDs = utils.CleanDuplicated(packageIDs)
	// for _, packageID := range packageIDs {
	// 	_, err := s.mw.LazadaReadyToShip(ctx, shop.AccessToken, shop.RefreshToken, packageID)
	// 	if err != nil {
	// 		utils.ShowInfoLogs("[LazadaServiceImpl] [ScanOrderPending] [newOrderAfterPending] [ReadyToShip] error: %s, Detail Order %s , packageID: %s", err.Error(), utils.MarshalToJson(order), orderDetail.Data[0].PackageID)
	// 		return newLeads, updateOrderLogs
	// 	}
	// 	// if rts == nil {
	// 	// 	utils.ShowInfoLogs("[LazadaServiceImpl] [ScanOrderPending] [newOrderAfterPending] [ReadyToShip] print fail, Detail Order %s , packageID: %s", utils.MarshalToJson(order), orderDetail.Data[0].PackageID)
	// 	// 	return newLeads, updateOrderLogs
	// 	// }
	// 	// if rts.Result.ErrorMsg != "" {
	// 	// 	utils.ShowInfoLogs("[LazadaServiceImpl] [ScanOrderPending] [newOrderAfterPending] [ReadyToShip] print fail, Detail Order %s , packageID: %s , error: %s",
	// 	// 		utils.MarshalNoEscape(order), orderDetail.Data[0].PackageID, awbData.Result.ErrorMsg)
	// 	// 	return newLeads, updateOrderLogs
	// 	// }
	// }
	//

	lead := models.CreateEcommerceRequest{
		Status:        utils.LeadCreated,
		ShippingLabel: awbData.Result.Data.PDFUrl,

		MarketPlace:   utils.Lazada,
		OrderID:       fmt.Sprint(order.Data.OrderID),
		TrackingCode:  orderDetail.Data[0].TrackingCode,
		Name:          order.Data.AddressShipping.FirstName,
		Phone:         order.Data.AddressShipping.Phone,
		Address:       order.Data.AddressShipping.Address1,
		Comment:       order.Data.BuyerNote,
		Province:      order.Data.AddressShipping.City,
		District:      order.Data.AddressShipping.AddressDistrict,
		Subdistrict:   order.Data.AddressShipping.Address2,
		Neighborhood:  order.Data.AddressShipping.Address3,
		Products:      products,
		TotalAmount:   totalAmount,
		ShopID:        shop.ShopID,
		Warehouse:     warehouse.ConnectorID,
		PaymentMethod: getLazadaPaymentMethod(order.Data.PaymentMethod),
		EnableAddress: false,

		// RawOrder:      utils.MarshalToJson(orderDetail[0]),
		RawOrderByte: orderDetail,
	}
	utils.ShowInfoLogs("[LazadaServiceImpl] [ScanOrderPending] [CreateLead] lead: %s", utils.MarshalNoEscape(lead))
	newLeads = append(newLeads, lead)
	updatedAt, _ := time.Parse(utils.DATE_TIME_FORMAT_YYYY_MM_DDT_HH_MM_SS_GMT, order.Data.UpdatedAt)

	log := models.RequestUpdateLogOrderStatus{
		ID:          id,
		Status:      utils.LeadCreated,
		MarketPlace: utils.Lazada,
		ShopID:      shop.ShopID,
		UpdatedAt:   updatedAt,
	}
	utils.ShowInfoLogs("[LazadaServiceImpl] [ScanOrderPending] [CreateLog] log: %s", utils.MarshalNoEscape(log))
	updateOrderLogs = append(updateOrderLogs, log)

	return newLeads, updateOrderLogs
}

func (s *LazadaServiceImpl) isInventoryAvailable(ctx context.Context, products []models.ConnectorProduct, shop models.MarketToken) bool {
	if len(products) == 0 {
		utils.ShowErrorLogs(fmt.Errorf("[LazadaServiceImpl] [isInventoryAvailable] [CheckProductQuantity] product is emtpy , shop: %s", shop.ShopID))
		return false
	}

	for _, prod := range products {
		enough := s.logisticService.CheckProductQuantity(ctx, models.LogisticInventoryRequest{
			PartnerCode:        s.cfg.Logistic.FfmCode,
			PartnerProductCode: prod.Sku,
			Quantity:           prod.Quantity,
		})
		if !enough {
			utils.ShowErrorLogs(fmt.Errorf("[LazadaServiceImpl] [isInventoryAvailable] [CheckProductQuantity] not enought product: %s , shop: %s", utils.MarshalNoEscape(prod), shop.ShopID))
			return false
		}
	}

	return true
}

func (s *LazadaServiceImpl) isSkipStatus(orderDetail lazada_model.Order) bool {
	if orderDetail.Statuses[0] == LazadaStatusPending {
		utils.ShowInfoLogs("[LazadaServiceImpl] [ScanOrder] [LazadaGetOrderDetail] orderDetail skipped is %s Order: %s", orderDetail.Statuses[0], utils.MarshalToJson(orderDetail))
		return true
	}
	return false
}

func (s *LazadaServiceImpl) ScanOrderToShip(
	ctx context.Context,
	shop models.MarketToken,
	orders []models.OdEcomOrder) ([]models.UpdateDOStatusRequest, error) {

	updateDOstatuses := []models.UpdateDOStatusRequest{}

	utils.ShowInfoLogs("[LazadaServiceImpl] [ScanOrderToShip] shop: %s , orders: %s ", utils.MarshalToJson(shop), utils.MarshalToJson(orders))

	for _, order := range orders {
		updateDOstatuses = s.orderToShip(ctx, shop, order, updateDOstatuses)
	}
	return updateDOstatuses, nil
}

func (s *LazadaServiceImpl) orderToShip(ctx context.Context,
	shop models.MarketToken,
	ecomOrder models.OdEcomOrder,
	updateDOstatuses []models.UpdateDOStatusRequest) []models.UpdateDOStatusRequest {
	utils.ShowInfoLogs("[LazadaServiceImpl] [orderToShip] Order: %s", ecomOrder.OrderID)

	order, err := s.mw.LazadaGetOrder(ctx, ecomOrder.OrderID, shop.AccessToken, shop.RefreshToken)
	if err != nil {
		utils.ShowErrorLogs(fmt.Errorf("[LazadaServiceImpl] [orderToShip] [LazadaGetOrder] order not found in shop: %s", err))
		return updateDOstatuses
	}
	if order == nil {
		utils.ShowErrorLogs(fmt.Errorf("[LazadaServiceImpl] [orderToShip] [LazadaGetOrder] order detail is empty shop: %s , orderID: %s", shop.ShopID, ecomOrder.OrderID))
		return updateDOstatuses
	}
	if order.Data == nil {
		utils.ShowErrorLogs(fmt.Errorf("[LazadaServiceImpl] [orderToShip] [LazadaGetOrder] order data is nil shop: %s , orderID: %s", shop.ShopID, ecomOrder.OrderID))
		return updateDOstatuses
	}
	if len(order.Data.Statuses) > 0 && order.Data.Statuses[0] == LazadaStatusCanceled {
		utils.ShowInfoLogs("[LazadaServiceImpl] [orderToShip] [LazadaGetOrder] order is cancelled, Order: %s", utils.MarshalToJson(order))
		return updateDOstatuses
	}

	utils.ShowInfoLogs("[LazadaServiceImpl] [orderToShip] [LazadaGetOrder] order: %s", utils.MarshalNoEscape(order))

	orderDetail, err := s.mw.LazadaGetOrderItems(ctx, order.Data.OrderID, shop.AccessToken, shop.RefreshToken)
	if err != nil {
		utils.ShowInfoLogs("[LazadaServiceImpl] [orderToShip] [LazadaGetOrderItems] error: %s , Order: %s", err.Error(), utils.MarshalToJson(order))
		return updateDOstatuses
	}
	if orderDetail == nil {
		utils.ShowInfoLogs("[LazadaServiceImpl] [orderToShip] [LazadaGetOrderItems] order is empty, Order: %s", utils.MarshalToJson(order))
		return updateDOstatuses
	}

	var (
		packageIDs []string
	)

	utils.ShowInfoLogs("[LazadaServiceImpl] [orderToShip] [OrderDetail] OrderDetail: %s", utils.MarshalNoEscape(orderDetail))

	for _, orderItem := range orderDetail.Data {
		utils.ShowInfoLogs("[LazadaServiceImpl] [orderToShip] [OrderDetail] OrderItem: %s", utils.MarshalNoEscape(orderItem))
		packageIDs = append(packageIDs, orderItem.PackageID)
	}

	packageIDs = utils.CleanDuplicated(packageIDs)
	for _, packageID := range packageIDs {
		_, err := s.mw.LazadaReadyToShip(ctx, shop.AccessToken, shop.RefreshToken, packageID)
		if err != nil {
			utils.ShowInfoLogs("[LazadaServiceImpl] [orderToShip] [ReadyToShip] error: %s, Detail Order %s , packageID: %s", err.Error(), utils.MarshalToJson(order), orderDetail.Data[0].PackageID)
			return updateDOstatuses
		}
		// if rts == nil {
		// 	utils.ShowInfoLogs("[LazadaServiceImpl] [orderToShip] [ReadyToShip] print fail, Detail Order %s , packageID: %s", utils.MarshalToJson(order), orderDetail.Data[0].PackageID)
		// 	// return updateDOstatuses
		// }
		// if rts.Result.ErrorMsg != "" {
		// 	utils.ShowInfoLogs("[LazadaServiceImpl] [orderToShip] [ReadyToShip] print fail, Detail Order %s , packageID: %s , error: %s",
		// 		utils.MarshalNoEscape(order), orderDetail.Data[0].PackageID, awbData.Result.ErrorMsg)
		// 	return updateDOstatuses
		// }
	}

	updateDOStatusRequest := models.UpdateDOStatusRequest{
		OrderID:      ecomOrder.OrderID,
		UpdatedAt:    time.Now().UTC(),
		TrackingCode: orderDetail.Data[0].TrackingCode,
		Status:       LazadaStatusReadyToShip,
	}

	utils.ShowInfoLogs("[LazadaServiceImpl] [orderToShip] updateDOStatusRequest: %s", utils.MarshalNoEscape(updateDOStatusRequest))

	updateDOstatuses = append(updateDOstatuses, updateDOStatusRequest)
	return updateDOstatuses
}

func (s *LazadaServiceImpl) ScanOrderPayment(
	ctx context.Context,
	shop models.MarketToken,
	id int) ([]tiktok_model.Payment, int, error) {

	orderPayments := []tiktok_model.Payment{}

	utils.ShowInfoLogs("[LazadaServiceImpl] [ScanOrderPayment] shop: %s , id: %d", utils.MarshalToJson(shop), id)

	orders, err := s.connectorRepo.GetShopOrderID(ctx, shop.MarketPlace, shop.ShopID, id)
	if err != nil {
		utils.ShowErrorLogs(fmt.Errorf("[LazadaServiceImpl] [ScanOrderPayment] [GetShopOrderID] error: %s , shop: %s", err.Error(), utils.MarshalToJson(orders)))
		return nil, 0, err
	}
	if len(orders) == 0 {
		utils.ShowErrorLogs(fmt.Errorf("[LazadaServiceImpl] [ScanOrderPayment] [GetShopOrderID] orderIds is empty , shop: %s , id: %d", shop.ShopID, id))
		return nil, 0, nil
	}

	for _, ordr := range orders {
		order, err := s.mw.LazadaGetOrder(ctx, ordr.OrderID, shop.AccessToken, shop.RefreshToken)
		if err != nil {
			utils.ShowErrorLogs(fmt.Errorf("[LazadaServiceImpl] [orderToShip] [LazadaGetOrder] order not found in shop: %s", err))
			continue
		}
		if order == nil {
			utils.ShowErrorLogs(fmt.Errorf("[LazadaServiceImpl] [orderToShip] [LazadaGetOrder] order detail is empty shop: %s , orderID: %s", shop.ShopID, ordr.OrderID))
			continue
		}
		if order.Data == nil {
			utils.ShowErrorLogs(fmt.Errorf("[LazadaServiceImpl] [orderToShip] [LazadaGetOrder] order data is nil shop: %s , orderID: %s", shop.ShopID, ordr.OrderID))
			continue
		}

		orderDetail, err := s.mw.LazadaGetOrderItems(ctx, order.Data.OrderID, shop.AccessToken, shop.RefreshToken)
		if err != nil {
			utils.ShowInfoLogs("[LazadaServiceImpl] [ScanOrderPayment] [LazadaGetOrderItems] error: %s , Order: %s", err.Error(), utils.MarshalToJson(order))
			continue
		}
		if orderDetail == nil {
			utils.ShowInfoLogs("[LazadaServiceImpl] [ScanOrderPayment] [LazadaGetOrderItems] order is empty, Order: %s", utils.MarshalToJson(order))
			continue
		}

		utils.ShowInfoLogs("[LazadaServiceImpl] [ScanOrderPayment] [Order] Order: %s", utils.MarshalToJson(order))

		payment := lazada_model.OrderItem{}

		for _, item := range orderDetail.Data {
			utils.ShowInfoLogs("[LazadaServiceImpl] [ScanOrderPayment] [OrderDetail] OrderDetail: %s", utils.MarshalNoEscape(item))

			payment.Currency = item.Currency
			payment.ShippingFeeOriginal += item.ShippingFeeOriginal
			payment.VoucherPlatform += item.VoucherPlatform
			payment.VoucherSeller += item.VoucherSeller
			payment.ShippingAmount += item.ShippingAmount
			payment.ShippingFeeDiscountPlatform += item.ShippingFeeDiscountPlatform
			payment.ShippingFeeDiscountSeller += item.ShippingFeeDiscountSeller
			payment.TaxAmount += item.TaxAmount
			payment.PaidPrice += item.PaidPrice
			payment.ItemPrice += item.ItemPrice
		}
		total := payment.ItemPrice + payment.ShippingAmount - payment.VoucherSeller - payment.ShippingFeeDiscountSeller

		orderPayments = append(orderPayments, tiktok_model.Payment{
			Id: ordr.ID,

			Currency:                    payment.Currency,
			OriginalShippingFee:         fmt.Sprintf("%v", payment.ShippingFeeOriginal),
			OriginalTotalProductPrice:   fmt.Sprintf("%v", payment.ItemPrice),
			PlatformDiscount:            fmt.Sprintf("%v", payment.VoucherPlatform),
			SellerDiscount:              fmt.Sprintf("%v", payment.VoucherSeller),
			ShippingFee:                 fmt.Sprintf("%v", payment.ShippingAmount),
			ShippingFeeCofundedDiscount: "",
			ShippingFeePlatformDiscount: fmt.Sprintf("%v", payment.ShippingFeeDiscountPlatform),
			ShippingFeeSellerDiscount:   fmt.Sprintf("%v", payment.ShippingFeeDiscountSeller),
			SubTotal:                    "",
			Tax:                         fmt.Sprintf("%v", payment.TaxAmount),
			TotalAmount:                 fmt.Sprintf("%v", total),
		})

		id = ordr.ID
	}

	utils.ShowInfoLogs("[LazadaServiceImpl] [ScanOrderPayment] total orderPayments: %d , last ID: %d", len(orderPayments), id)

	return orderPayments, id, nil
}

func (s *LazadaServiceImpl) ScanTransactionStatement(ctx context.Context, shop models.MarketToken, orders []models.OdEcomOrderID) ([]models.EcomStatementTransaction, error) {
	utils.ShowInfoLogs("[LazadaServiceImpl] [ScanOrderStatementTransaction] shop: %s , orderIds: %s", shop.ShopID, utils.MarshalToJson(orders))

	statementTrans := []models.EcomStatementTransaction{}
	for _, order := range orders {
		statements, err := s.mw.QueryTransactionDetails(ctx, shop.AccessToken, shop.RefreshToken, order.OrderID)
		if err != nil {
			utils.ShowErrorLogs(fmt.Errorf("[LazadaServiceImpl] [ScanOrderStatementTransaction] [QueryTransactionDetails] error: %s , orderID: %s , shop: %s", err.Error(), order.OrderID, shop.ShopID))
			continue
		}
		if statements == nil || len(statements.Data) == 0 {
			utils.ShowErrorLogs(fmt.Errorf("[LazadaServiceImpl] [ScanOrderStatementTransaction] [QueryTransactionDetails] statements is empty , orderID: %s , shop: %s", order.OrderID, shop.ShopID))
			continue
		}

		transactions, err := json.Marshal(statements.Data)
		if err != nil {
			utils.ShowErrorLogs(fmt.Errorf("[LazadaServiceImpl] [ScanOrderStatementTransaction] [Marshal] error: %s , orderID: %s", err.Error(), order.OrderID))
			continue
		}

		statementTrans = append(statementTrans, models.EcomStatementTransaction{
			OrderID:     order.OrderID,
			MarketPlace: utils.Lazada,
			RawData:     transactions,
			CreatedAt:   time.Now().UTC(),
		})
	}

	utils.ShowInfoLogs("[LazadaServiceImpl] [ScanOrderStatementTransaction] total statementTrans: %d , shop: %s", len(statementTrans), shop.ShopID)

	return statementTrans, nil
}
