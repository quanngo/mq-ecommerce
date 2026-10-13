package services

import (
	"context"
	"encoding/json"
	"fmt"
	"main/component/models"
	tiktok_model "main/component/models/tiktok_model"
	"main/component/repositories"
	"main/middleware"
	"main/utils"
	"strconv"
	"time"
)

const (
	TIKTOK                         = "Tiktok"
	TiktokStatusUnpaid             = "UNPAID"
	TiktokStatusOnHold             = "ON_HOLD"
	TiktokStatusAwaitingShipment   = "AWAITING_SHIPMENT"
	TiktokStatusAwaitingCollection = "AWAITING_COLLECTION"
	TiktokStatusCancel             = "CANCEL"
	TiktokStatusInTransit          = "IN_TRANSIT"
	TiktokStatusDelivered          = "DELIVERED"
	TiktokStatusCompleted          = "COMPLETED"
	TiktokStatusCancelled          = "CANCELLED"
	TikTokStatusReturn             = "RETURN"

	TitkokRefundComplete = "RETURN_OR_REFUND_REQUEST_COMPLETE"
	TitkokRefundSuccess  = "RETURN_OR_REFUND_REQUEST_SUCCESS"
	TitkokRefundReject   = "REFUND_OR_RETURN_REQUEST_REJECT"

	TitkokSystem = "SYSTEM"
	TitkokBuyer  = "BUYER"
)

const (
	TikTok_COD    = "COD"
	TikTok_Derbit = "Credit/debit card"
)

type TikTokService interface {
	CheckExpiredTokenTiktok()
	ScanOrder(
		ctx context.Context,
		timeCreated, timeUpdated time.Time,
		shop models.MarketToken,
		warehouse *models.MarketPlaceMapping,
		mapProducts map[string]models.MarketPlaceMapping) (
		[]models.UpdateDatabaseOrder,
		[]models.CreateEcommerceRequest,
		[]models.UpdateDOStatusRequest, error)
	ScanOrderPending(
		ctx context.Context,
		lstID []string,
		shop models.MarketToken,
		warehouse *models.MarketPlaceMapping,
		mapProducts map[string]models.MarketPlaceMapping) (
		[]models.CreateEcommerceRequest,
		[]models.RequestUpdateLogOrderStatus,
		error)
	ScanOrderRefund(
		ctx context.Context,
		timeUpdated time.Time,
		shop models.MarketToken) (
		[]models.RequestUpdateLogOrderStatus,
		[]models.UpdateDOStatusRequest, error)
	ScanOrderPayment(
		ctx context.Context,
		shop models.MarketToken,
		id int) ([]tiktok_model.Payment, int, error)
	ScanOrderStatementTransaction(
		ctx context.Context,
		shop models.MarketToken,
		orders []models.OdEcomOrderID) ([]models.EcomStatementTransaction, error)
}

type TikTokServiceImpl struct {
	cfg                     *utils.Config
	mw                      *middleware.ApiMiddleware
	connectorRepo           *repositories.ConnectorRepo
	tokenRepo               *repositories.TokenRepo
	marketPlaceMappingRepo  *repositories.MarketPlaceMappingRepo
	logMarketPlaceOrderRepo *repositories.LogMarketPlaceOrderRepo
	connectorService        ConnectorService
	logisticService         LogisticService
}

func NewTikTokService(
	cfg *utils.Config,
	mw *middleware.ApiMiddleware,
	connectorRepo *repositories.ConnectorRepo,
	tokenRepo *repositories.TokenRepo,
	marketPlaceMappingRepo *repositories.MarketPlaceMappingRepo,
	logMarketPlaceOrderRepo *repositories.LogMarketPlaceOrderRepo,
	connectorService ConnectorService,
	logisticService LogisticService) TikTokService {
	return &TikTokServiceImpl{
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

func (s *TikTokServiceImpl) CheckExpiredTokenTiktok() {
	ctx := context.Background()
	tiktok, err := s.tokenRepo.GetShopID(ctx, utils.TikTok)
	if err != nil {
		return
	}
	for _, shopID := range tiktok {
		timeExpired := s.tokenRepo.GetExpireTime(ctx, utils.TikTok, shopID)
		if timeExpired.After(time.Now().Add(-1 * time.Hour)) {
			s.mw.TiktokRefreshAccessToken(shopID, "")
		}
	}
}

func (s *TikTokServiceImpl) ScanOrder(
	ctx context.Context,
	timeCreated, timeUpdated time.Time,
	shop models.MarketToken,
	warehouse *models.MarketPlaceMapping,
	mapProducts map[string]models.MarketPlaceMapping) ([]models.UpdateDatabaseOrder, []models.CreateEcommerceRequest, []models.UpdateDOStatusRequest, error) {

	newLeadStatues := []models.UpdateDatabaseOrder{}
	newLeads := []models.CreateEcommerceRequest{}
	updateDOs := []models.UpdateDOStatusRequest{}

	utils.ShowInfoLogs("[TiktokServiceImpl] [ScanOrder] timeCreated: %v , timeUpdated: %v , shop: %s",
		timeCreated.UTC(), timeUpdated.UTC(), utils.MarshalToJson(shop))

	orders, err := s.fetchAllOrders(ctx, shop, timeUpdated)
	if err != nil {
		utils.ShowErrorLogs(fmt.Errorf("[TiktokServiceImpl] [ScanOrder] [TiktokGetOrderDetail] order not found in shop: %s", err))
		return nil, nil, nil, err
	}

	for _, order := range orders {
		utils.ShowInfoLogs("[TiktokServiceImpl] [ScanOrder] [Order] Order: %s", utils.MarshalToJson(order))

		timeCreatedOrder := time.Unix(order.CreateTime, 0)
		timeUpdatedOrder := time.Unix(order.UpdateTime, 0)
		timeCreatedOrder = timeCreatedOrder.UTC()
		timeUpdatedOrder = timeUpdatedOrder.UTC()

		orderDetail, err := s.mw.TiktokGetOrderDetail(ctx, order.ID, shop.AccessToken, shop.RefreshToken, *shop.ShopReference)
		if err != nil {
			utils.ShowErrorLogs(fmt.Errorf("[TiktokServiceImpl] [ScanOrder] [TiktokGetOrderDetail] error Order: %s", utils.MarshalToJson(order)))
			continue
		}
		if len(orderDetail) == 0 || orderDetail == nil {
			utils.ShowErrorLogs(fmt.Errorf("[TiktokServiceImpl] [ScanOrder] [TiktokGetOrderDetail] orderDetail is empty Order: %s", utils.MarshalToJson(order)))
			continue
		}
		if order.Status == TiktokStatusCancelled {
			utils.ShowInfoLogs("[TiktokServiceImpl] [ScanOrder] [TiktokGetOrderDetail] order is cancelled, Order: %s", utils.MarshalToJson(orderDetail))
		}

		// Skip Status
		if s.isSkipStatus(orderDetail[0]) {
			continue
		}

		// Check order exists
		checkOrderExists, err := s.logMarketPlaceOrderRepo.CheckOrderExists(utils.TikTok, order.ID)
		if err != nil {
			utils.ShowErrorLogs(fmt.Errorf("[TiktokServiceImpl] [ScanOrder] [CheckOrderExists] error %s ,  Order: %s", err.Error(), utils.MarshalToJson(order)))
			continue
		}

		// Create new order in Connector
		// Tiktok only create new order when status is AWAITING_COLLECTION
		if !checkOrderExists && order.Status == TiktokStatusAwaitingCollection {
			newLeadStatues, newLeads = s.newOrder(ctx, shop, order, timeCreatedOrder, orderDetail, mapProducts, warehouse, newLeads, newLeadStatues)
			continue
		}

		if timeUpdatedOrder.After(timeUpdated) {
			updateDOStatusRequest := updateDOstatus(orderDetail, shop, timeUpdatedOrder, order)
			updateDOs = append(updateDOs, updateDOStatusRequest)
		}
	}

	utils.ShowInfoLogs("[TiktokServiceImpl] [ScanOrder] total newLeadStatues: %d , total newLeads: %d , total updateDOs: %d", len(newLeadStatues), len(newLeads), len(updateDOs))

	return newLeadStatues, newLeads, updateDOs, nil
}

func updateDOstatus(orderDetail []tiktok_model.TikTokOrderDetailModel, shop models.MarketToken, timeUpdatedOrder time.Time, order tiktok_model.TikTokOrder) models.UpdateDOStatusRequest {
	doStatus := getDoStatus(orderDetail, shop, order)

	updateDOStatusRequest := models.UpdateDOStatusRequest{
		UpdatedAt: timeUpdatedOrder,
		OrderID:   order.ID,
	}

	switch {
	case len(doStatus) > 0:
		updateDOStatusRequest.TrackingCode = order.TrackingNumber
		updateDOStatusRequest.Status = doStatus

	case order.Status == TiktokStatusCompleted:
		updateDOStatusRequest.SoStatus = utils.SaleOrderCompleted

	default:
		updateDOStatusRequest.TrackingCode = order.TrackingNumber
		updateDOStatusRequest.Status = order.Status
	}
	return updateDOStatusRequest
}

func getDoStatus(orderDetail []tiktok_model.TikTokOrderDetailModel, shop models.MarketToken, order tiktok_model.TikTokOrder) string {
	doStatus := ""

	if orderDetail[0].Status == TiktokStatusCancelled {
		switch orderDetail[0].CancellationInitiator {
		case TitkokSystem:
			doStatus = TikTokStatusReturn
		case TitkokBuyer:
			doStatus = TiktokStatusCancelled
		}

		utils.ShowInfoLogs("[TiktokServiceImpl] [ScanOrder] [TiktokGetOrderDetail] order status is CANCELLED , DO status is %s shop: %s , orderID: %s , status: %s", doStatus, shop.ShopID, order.ID, orderDetail[0].Status)
	}

	return ""
}

func (s *TikTokServiceImpl) newOrder(ctx context.Context,
	shop models.MarketToken,
	order tiktok_model.TikTokOrder, timeCreatedOrder time.Time,
	orderDetail []tiktok_model.TikTokOrderDetailModel,
	mapProducts map[string]models.MarketPlaceMapping,
	warehouse *models.MarketPlaceMapping,
	newLeads []models.CreateEcommerceRequest,
	newLeadStatues []models.UpdateDatabaseOrder) ([]models.UpdateDatabaseOrder, []models.CreateEcommerceRequest) {

	newLeadStatues = append(newLeadStatues, models.UpdateDatabaseOrder{
		MarketPlace: utils.TikTok,
		ShopID:      shop.ShopID,
		OrderID:     order.ID,
		Status:      order.Status,
		CreatedDate: timeCreatedOrder,
	})

	var products []models.ConnectorProduct
	var totalAmount float64 = 0

	if orderDetail[0].LineItems == nil {
		utils.ShowErrorLogs(fmt.Errorf("[TiktokServiceImpl] [ScanOrder] [OrderDetail] LineItems is nil shop: %s , order: %s", shop.ShopID, utils.MarshalNoEscape(order)))
		return nil, nil
	}

	for _, item := range orderDetail[0].LineItems {
		utils.ShowInfoLogs("[TiktokServiceImpl] [ScanOrder] [OrderDetail] OrderDetail: %s", utils.MarshalNoEscape(order))

		prod, exist := mapProducts[item.ProductName]
		if !exist {
			utils.ShowErrorLogs(fmt.Errorf("[TiktokServiceImpl] [ScanOrder] [GetProducts] product not found: %s", shop.ShopID))
			continue
		}

		price, err := strconv.ParseFloat(item.OriginalPrice, 64)
		if err != nil {
			utils.ShowErrorLogs(fmt.Errorf("[TiktokServiceImpl] [ScanOrder] [ParseFloat] error: %s , OrderDetail: %s", err.Error(), utils.MarshalNoEscape(order)))
			continue
		}

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

		MarketPlace:   utils.TikTok,
		OrderID:       orderDetail[0].ID,
		Name:          orderDetail[0].RecipientAddress.Name,
		Phone:         orderDetail[0].RecipientAddress.PhoneNumber,
		Address:       orderDetail[0].RecipientAddress.FullAddress,
		Comment:       orderDetail[0].BuyerMessage,
		Province:      orderDetail[0].RecipientAddress.AddressLine1,
		District:      orderDetail[0].RecipientAddress.AddressLine2,
		Subdistrict:   orderDetail[0].RecipientAddress.AddressLine3,
		Neighborhood:  orderDetail[0].RecipientAddress.AddressLine4,
		Products:      products,
		TotalAmount:   totalAmount,
		ShopID:        shop.ShopID,
		Warehouse:     warehouse.ConnectorID,
		PaymentMethod: getTikTokPaymentMethod(order.PaymentMethodName),
		EnableAddress: false,

		// RawOrder:      utils.MarshalToJson(orderDetail[0]),
		RawOrderByte: orderDetail[0],
	}

	// Check inventory
	if !s.isInventoryAvailable(ctx, products, shop) {
		utils.ShowErrorLogs(fmt.Errorf("[TiktokServiceImpl] [ScanOrder] [CheckInventory] inventory not enough shop: %s , order: %s",
			shop.ShopID, utils.MarshalNoEscape(order)))
		return appendLead(newLeads, newLead, newLeadStatues)
	}

	// (Old version) Progress: Pack & Shipping -> Get Label
	// (New version) Progress: Get Label
	// 1.Pack & Shipping
	// if len(orderDetail[0].Packages) == 0 {
	// 	utils.ShowInfoLogs("[TikTokServiceImpl] [ScanOrder] [ShipPackage] Packages is empty, Detail Order %s", utils.MarshalToJson(order))
	// 	return appendLead(newLeads, newLead, newLeadStatues)
	// }
	// _, err := s.mw.TiktokShipPackage(ctx, orderDetail[0].Packages[0].ID, shop.AccessToken, shop.RefreshToken, shop.ShopID, *shop.ShopReference)
	// if err != nil {
	// 	utils.ShowInfoLogs("[TikTokServiceImpl] [ScanOrder] [ShipPackage] pack fail, Detail Order %s , packageID: %s", utils.MarshalToJson(order), orderDetail[0].Packages[0].ID)
	// 	return appendLead(newLeads, newLead, newLeadStatues)
	// }
	// 2.Get Label
	packageDocument := s.mw.TiktokGetPackageShippingDocument(ctx, orderDetail[0].Packages[0].ID, shop.AccessToken, shop.RefreshToken, *shop.ShopReference)
	if packageDocument == nil {
		utils.ShowInfoLogs("[TikTokServiceImpl] [ScanOrder] [PrintAWB] packageDocument is null Detail Order %s , packageID: %s", utils.MarshalToJson(order), orderDetail[0].Packages[0].ID)
		return appendLead(newLeads, newLead, newLeadStatues)
	} else if packageDocument.Message != "Success" {
		utils.ShowInfoLogs("[TikTokServiceImpl] [ScanOrder] [PrintAWB] print fail, Detail Order %s , packageID: %s , error: %s",
			utils.MarshalNoEscape(order), orderDetail[0].Packages[0].ID, packageDocument.Message)
		return appendLead(newLeads, newLead, newLeadStatues)
	}
	//

	newLead.Status = utils.LeadCreated
	newLead.TrackingCode = packageDocument.Data.TrackingNumber
	newLead.ShippingLabel = packageDocument.Data.DocURL

	utils.ShowInfoLogs("[TiktokServiceImpl] [ScanOrder] [CreateLead] newLead: %s", utils.MarshalNoEscape(newLead))
	return appendLead(newLeads, newLead, newLeadStatues)
}

func (s *TikTokServiceImpl) isInventoryAvailable(ctx context.Context, products []models.ConnectorProduct, shop models.MarketToken) bool {
	if len(products) == 0 {
		utils.ShowErrorLogs(fmt.Errorf("[TiktokServiceImpl] [isInventoryAvailable] [CheckProductQuantity] product is emtpy , shop: %s", shop.ShopID))
		return false
	}

	for _, prod := range products {
		enough := s.logisticService.CheckProductQuantity(ctx, models.LogisticInventoryRequest{
			PartnerCode:        s.cfg.Logistic.FfmCode,
			PartnerProductCode: prod.Sku,
		})
		if !enough {
			utils.ShowErrorLogs(fmt.Errorf("[TiktokServiceImpl] [isInventoryAvailable] [CheckProductQuantity] not enought product: %s , shop: %s", utils.MarshalNoEscape(prod), shop.ShopID))
			return false
		}
	}

	return true
}

func (s *TikTokServiceImpl) fetchAllOrders(ctx context.Context, shop models.MarketToken, timeUpdated time.Time) ([]tiktok_model.TikTokOrder, error) {
	var orders []tiktok_model.TikTokOrder
	var nextPageToken string

	for {
		result, err := s.mw.TiktokGetOrderList(ctx, timeUpdated.Unix(), shop.AccessToken, shop.RefreshToken, shop.ShopID, *shop.ShopReference, nextPageToken)
		if err != nil {
			utils.ShowErrorLogs(fmt.Errorf("[TiktokServiceImpl] [fetchAllOrders] [TiktokGetOrderList] order not found in shop: %w", err))
			return nil, err
		}

		orders = append(orders, result.Data.Orders...)

		if result.Data.NextPageToken == "" {
			break
		}
		nextPageToken = result.Data.NextPageToken
	}

	return orders, nil
}

func getTikTokPaymentMethod(paymentMethod string) int {
	switch paymentMethod {
	// case TikTok_COD:
	// 	return utils.PAYMENT_METHOD_COD
	default:
		return utils.PAYMENT_METHOD_ECOMMERCE
	}
}

func (s *TikTokServiceImpl) ScanOrderPending(
	ctx context.Context,
	lstIDs []string,
	shop models.MarketToken,
	warehouse *models.MarketPlaceMapping,
	mapProducts map[string]models.MarketPlaceMapping) ([]models.CreateEcommerceRequest, []models.RequestUpdateLogOrderStatus, error) {

	newLeads := []models.CreateEcommerceRequest{}
	updateOrderLogs := []models.RequestUpdateLogOrderStatus{}

	utils.ShowInfoLogs("[TiktokServiceImpl] [ScanOrderPending] lstID: %s , shop: %s",
		lstIDs, utils.MarshalToJson(shop))

	for _, id := range lstIDs {
		newLeads, updateOrderLogs = s.newOrderAfterPending(ctx, id, shop, mapProducts, warehouse, newLeads, updateOrderLogs)
	}

	utils.ShowInfoLogs("[TiktokServiceImpl] [ScanOrderPending] total newLeads: %d , total updateOrderLogs: %d", len(newLeads), len(updateOrderLogs))

	return newLeads, updateOrderLogs, nil
}

func (s *TikTokServiceImpl) newOrderAfterPending(ctx context.Context,
	id string,
	shop models.MarketToken,
	mapProducts map[string]models.MarketPlaceMapping,
	warehouse *models.MarketPlaceMapping,
	newLeads []models.CreateEcommerceRequest,
	updateOrderLogs []models.RequestUpdateLogOrderStatus) ([]models.CreateEcommerceRequest, []models.RequestUpdateLogOrderStatus) {

	utils.ShowInfoLogs("[TiktokServiceImpl] [ScanOrderPending] [Order] Order: %s", id)

	order, err := s.mw.TiktokGetOrderDetail(ctx, id, shop.AccessToken, shop.RefreshToken, *shop.ShopReference)
	if err != nil {
		utils.ShowErrorLogs(fmt.Errorf("[TiktokServiceImpl] [ScanOrderPending] [TiktokGetOrderDetail] order not found in shop: %s", err))
		return newLeads, updateOrderLogs
	}
	if len(order) == 0 {
		utils.ShowErrorLogs(fmt.Errorf("[TiktokServiceImpl] [ScanOrderPending] [TiktokGetOrderDetail] order detail is empty shop: %s , orderID: %s", shop.ShopID, id))
		return newLeads, updateOrderLogs
	}
	if order[0].Status == TiktokStatusCancelled {
		utils.ShowErrorLogs(fmt.Errorf("[TiktokServiceImpl] [ScanOrderPending] [TiktokGetOrderDetail] order status is CANCELLED shop: %s , orderID: %s , status: %s", shop.ShopID, id, order[0].Status))
		return newLeads, updateOrderLogs
	}
	if order[0].LineItems == nil {
		utils.ShowErrorLogs(fmt.Errorf("[TiktokServiceImpl] [ScanOrderPending] [TiktokGetOrderDetail] LineItems is nil shop: %s , order: %s", shop.ShopID, utils.MarshalNoEscape(order)))
		return newLeads, updateOrderLogs
	}

	var products []models.ConnectorProduct
	var totalAmount float64 = 0

	for _, item := range order[0].LineItems {

		utils.ShowInfoLogs("[TiktokServiceImpl] [ScanOrderPending] [OrderDetail] OrderDetail: %s", utils.MarshalNoEscape(order))

		prod, exist := mapProducts[item.ProductName]
		if !exist {
			utils.ShowErrorLogs(fmt.Errorf("[TiktokServiceImpl] [ScanOrderPending] [GetProducts] product not found: %s , shop: %s", item.ProductName, shop.ShopID))
			continue
		}

		price, err := strconv.ParseFloat(item.OriginalPrice, 64)
		if err != nil {
			utils.ShowErrorLogs(fmt.Errorf("[TiktokServiceImpl] [ScanOrderPending] [ParseFloat] error: %s , OrderDetail: %s", err.Error(), utils.MarshalNoEscape(order)))
			continue
		}

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
		utils.ShowErrorLogs(fmt.Errorf("[TiktokServiceImpl] [ScanOrderPending] [CheckInventory] inventory not enough shop: %s , order: %s",
			shop.ShopID, utils.MarshalNoEscape(order)))
		return newLeads, updateOrderLogs
	}

	// (Old version) Progress: Pack & Shipping -> Get Label
	// (New version) Progress: Get Label
	// 1.Pack & Shipping
	// if len(order[0].Packages) == 0 {
	// 	utils.ShowInfoLogs("[TikTokServiceImpl] [ScanOrder] [ShipPackage] Packages is empty, Detail Order %s", utils.MarshalToJson(order))
	// 	return newLeads, updateOrderLogs
	// }
	// _, err = s.mw.TiktokShipPackage(ctx, order[0].Packages[0].ID, shop.AccessToken, shop.RefreshToken, shop.ShopID, *shop.ShopReference)
	// if err != nil {
	// 	utils.ShowInfoLogs("[TikTokServiceImpl] [ScanOrderPending] [ShipPackage] pack fail, Detail Order %s , packageID: %s", utils.MarshalToJson(order), order[0].Packages[0].ID)
	// 	return newLeads, updateOrderLogs
	// }
	// 2.Get Label
	packageDocument := s.mw.TiktokGetPackageShippingDocument(ctx, order[0].Packages[0].ID, shop.AccessToken, shop.RefreshToken, *shop.ShopReference)
	if packageDocument == nil {
		utils.ShowInfoLogs("[TikTokServiceImpl] [ScanOrderPending] packageDocument is null, Detail Order %s , packageID: %s", utils.MarshalToJson(order), order[0].Packages[0].ID)
		return newLeads, updateOrderLogs

	} else if packageDocument.Message != "Success" {
		utils.ShowInfoLogs("[TikTokServiceImpl] [ScanOrderPending] [PrintAWB] print fail, Detail Order %s , packageID: %s , error: %s",
			utils.MarshalNoEscape(order), order[0].Packages[0].ID, packageDocument.Message)
		return newLeads, updateOrderLogs
	}
	//

	lead := models.CreateEcommerceRequest{
		MarketPlace:   utils.TikTok,
		Status:        utils.LeadCreated,
		OrderID:       order[0].ID,
		TrackingCode:  order[0].TrackingNumber,
		Name:          order[0].RecipientAddress.Name,
		Phone:         order[0].RecipientAddress.PhoneNumber,
		Address:       order[0].RecipientAddress.FullAddress,
		Comment:       order[0].BuyerMessage,
		Province:      order[0].RecipientAddress.AddressLine1,
		District:      order[0].RecipientAddress.AddressLine2,
		Subdistrict:   order[0].RecipientAddress.AddressLine3,
		Neighborhood:  order[0].RecipientAddress.AddressLine4,
		Products:      products,
		TotalAmount:   totalAmount,
		ShopID:        shop.ShopID,
		Warehouse:     warehouse.ConnectorID,
		ShippingLabel: packageDocument.Data.DocURL,
		PaymentMethod: getTikTokPaymentMethod(order[0].PaymentMethodName),
		EnableAddress: false,

		// RawOrder:      utils.MarshalToJson(orderDetail[0]),
		RawOrderByte: order[0],
	}
	utils.ShowInfoLogs("[TiktokServiceImpl] [ScanOrderPending] [CreateLead] lead: %s", utils.MarshalNoEscape(lead))
	newLeads = append(newLeads, lead)

	log := models.RequestUpdateLogOrderStatus{
		ID:          id,
		Status:      utils.LeadCreated,
		MarketPlace: utils.TikTok,
		ShopID:      shop.ShopID,
		UpdatedAt:   time.Unix(order[0].UpdateTime, 0),
	}
	utils.ShowInfoLogs("[TiktokServiceImpl] [ScanOrderPending] [CreateLog] log: %s", utils.MarshalNoEscape(log))
	updateOrderLogs = append(updateOrderLogs, log)

	return newLeads, updateOrderLogs
}

func (s *TikTokServiceImpl) ScanOrderRefund(
	ctx context.Context,
	timeUpdated time.Time,
	shop models.MarketToken) ([]models.RequestUpdateLogOrderStatus, []models.UpdateDOStatusRequest, error) {

	updateOrderLogs := []models.RequestUpdateLogOrderStatus{}
	updateDOStatusRequest := []models.UpdateDOStatusRequest{}

	utils.ShowInfoLogs("[TiktokServiceImpl] [ScanOrderRefund] timeUpdated: %v , shop: %s", timeUpdated.UTC(), utils.MarshalToJson(shop))

	orders, err := s.fetchAllRefundOrders(ctx, shop, timeUpdated.UTC())
	if err != nil {
		utils.ShowErrorLogs(fmt.Errorf("[TiktokServiceImpl] [ScanOrderRefund] [fetchAllRefundOrders] order not found in shop: %s", err))
		return nil, nil, err
	}

	for _, order := range orders {
		utils.ShowInfoLogs("[TiktokServiceImpl] [ScanOrderRefund] [Order] Order: %s", utils.MarshalToJson(order))

		logOrder, err := s.logMarketPlaceOrderRepo.GetLogOrder(ctx, models.GetLogMarketPlaceOrder{
			OrderID:     order.OrderID,
			MarketPlace: shop.MarketPlace,
			ShopID:      shop.ShopID,
		})
		if err != nil {
			utils.ShowErrorLogs(fmt.Errorf("[TiktokServiceImpl] [ScanOrderRefund] [GetLogOrder] Order: %s shop: %s", utils.MarshalToJson(order), err))
			continue
		}
		if len(logOrder) == 0 {
			utils.ShowErrorLogs(fmt.Errorf("[TiktokServiceImpl] [ScanOrderRefund] [GetLogOrder] log order not found OrderID: %s shop: %s", order.OrderID, shop.ShopID))
			continue
		}

		if order.ReturnStatus != TitkokRefundComplete &&
			order.ReturnStatus != TitkokRefundSuccess {
			utils.ShowInfoLogs("[TiktokServiceImpl] [ReturnStatus] [Order] not complete Order: %s", utils.MarshalToJson(order))
			continue
		}

		log := models.RequestUpdateLogOrderStatus{
			ID:          order.OrderID,
			Status:      utils.LeadRefund,
			MarketPlace: utils.TikTok,
			ShopID:      shop.ShopID,
			UpdatedAt:   time.Unix(order.UpdateTime, 0).UTC(),
		}
		utils.ShowInfoLogs("[TiktokServiceImpl] [ScanOrderRefund] [CreateLog] log: %s", utils.MarshalNoEscape(log))
		updateOrderLogs = append(updateOrderLogs, log)

		doStatus := models.UpdateDOStatusRequest{
			OrderID:  order.OrderID,
			SoStatus: utils.SaleOrderRefund,
		}
		utils.ShowInfoLogs("[TiktokServiceImpl] [ScanOrderRefund] [CreateDOStatus] doStatus: %s", utils.MarshalNoEscape(doStatus))
		updateDOStatusRequest = append(updateDOStatusRequest, doStatus)

	}

	utils.ShowInfoLogs("[TiktokServiceImpl] [ScanOrderRefund] total updateOrderLogs: %d , total updateDOStatusRequest: %d", len(updateOrderLogs), len(updateDOStatusRequest))

	return updateOrderLogs, updateDOStatusRequest, nil
}

func (s *TikTokServiceImpl) fetchAllRefundOrders(ctx context.Context, shop models.MarketToken, timeUpdated time.Time) ([]tiktok_model.TikTokRefundOrder, error) {
	var orders []tiktok_model.TikTokRefundOrder
	var nextPageToken string

	for {
		result, err := s.mw.TiktokGetRefundList(ctx, timeUpdated.UTC().Unix(), shop.AccessToken, shop.RefreshToken, shop.ShopID, *shop.ShopReference, nextPageToken)
		if err != nil {
			utils.ShowErrorLogs(fmt.Errorf("[TiktokServiceImpl] [fetchAllRefundOrders] [TiktokGetRefundList] order not found in shop: %w", err))
			return nil, err
		}

		orders = append(orders, result.Data.ReturnOrders...)

		if result.Data.NextPageToken == "" {
			break
		}
		nextPageToken = result.Data.NextPageToken
	}

	return orders, nil
}

func (s *TikTokServiceImpl) isSkipStatus(orderDetail tiktok_model.TikTokOrderDetailModel) bool {
	if orderDetail.Status == TiktokStatusAwaitingShipment ||
		orderDetail.Status == TiktokStatusOnHold {
		utils.ShowInfoLogs("[TiktokServiceImpl] [ScanOrder] [TiktokGetOrderDetail] orderDetail skipped is %s awaiting shipment Order: %s", orderDetail.Status, utils.MarshalToJson(orderDetail))
		return true
	}
	return false
}

func (s *TikTokServiceImpl) ScanOrderPayment(
	ctx context.Context,
	shop models.MarketToken,
	id int) ([]tiktok_model.Payment, int, error) {

	orderPayments := []tiktok_model.Payment{}

	utils.ShowInfoLogs("[TiktokServiceImpl] [ScanOrderPayment] shop: %s , id: %d", utils.MarshalToJson(shop), id)

	orders, err := s.connectorRepo.GetShopOrderID(ctx, shop.MarketPlace, shop.ShopID, id)
	if err != nil {
		utils.ShowErrorLogs(fmt.Errorf("[TiktokServiceImpl] [ScanOrderPayment] [GetShopOrderID] error: %s , shop: %s", err.Error(), utils.MarshalToJson(orders)))
		return nil, 0, err
	}
	if len(orders) == 0 {
		utils.ShowErrorLogs(fmt.Errorf("[TiktokServiceImpl] [ScanOrderPayment] [GetShopOrderID] orderIds is empty , shop: %s , id: %d", shop.ShopID, id))
		return nil, 0, nil
	}

	for _, order := range orders {
		orderDetail, err := s.mw.TiktokGetOrderDetail(ctx, order.OrderID, shop.AccessToken, shop.RefreshToken, *shop.ShopReference)
		if err != nil {
			utils.ShowErrorLogs(fmt.Errorf("[TiktokServiceImpl] [ScanOrderPayment] [TiktokGetOrderDetail] order not found in shop: %s", err))
			continue
		}
		if len(orderDetail) == 0 {
			utils.ShowErrorLogs(fmt.Errorf("[TiktokServiceImpl] [ScanOrderPayment] [TiktokGetOrderDetail] order detail is empty shop: %s , orderID: %s", shop.ShopID, order.OrderID))
			continue
		}
		utils.ShowInfoLogs("[TiktokServiceImpl] [ScanOrderPayment] [Order] Order: %s", utils.MarshalToJson(order))

		totalPrice, _ := strconv.ParseFloat(orderDetail[0].Payment.OriginalTotalProductPrice, 64)
		totalShippingFee, _ := strconv.ParseFloat(orderDetail[0].Payment.ShippingFee, 64)
		totalDiscount, _ := strconv.ParseFloat(orderDetail[0].Payment.SellerDiscount, 64)
		totalDiscountShippingFee, _ := strconv.ParseFloat(orderDetail[0].Payment.ShippingFeeSellerDiscount, 64)
		totalPaid := totalPrice + totalShippingFee - totalDiscount - totalDiscountShippingFee

		payment := orderDetail[0].Payment
		payment.TotalAmount = fmt.Sprintf("%.2f", totalPaid)
		payment.Id = order.ID
		orderPayments = append(orderPayments, payment)

		id = order.ID
	}

	utils.ShowInfoLogs("[TiktokServiceImpl] [ScanOrderPayment] total orderPayments: %d , last ID: %d", len(orderPayments), id)

	return orderPayments, id, nil
}

func (s *TikTokServiceImpl) ScanOrderStatementTransaction(ctx context.Context,
	shop models.MarketToken,
	orders []models.OdEcomOrderID) ([]models.EcomStatementTransaction, error) {
	utils.ShowInfoLogs("[TiktokServiceImpl] [ScanOrderStatementTransaction] shop: %s , orderIds: %s", shop.ShopID, utils.MarshalToJson(orders))

	statementTrans := []models.EcomStatementTransaction{}

	for _, order := range orders {
		statementTransactions, err := s.mw.TiktokGetStatementTransactions(ctx, shop.AccessToken, shop.RefreshToken, shop.ShopID, *shop.ShopReference, order.OrderID)
		if err != nil {
			utils.ShowErrorLogs(fmt.Errorf("[TiktokServiceImpl] [ScanOrderStatementTransaction] [GetStatementTransaction] order not found in shop: %s", err))
			continue
		}
		if statementTransactions == nil {
			utils.ShowErrorLogs(fmt.Errorf("[TiktokServiceImpl] [ScanOrderStatementTransaction] [GetStatementTransaction] statementTransactions is nil shop: %s , orderID: %s", shop.ShopID, order.OrderID))
			continue
		}
		if statementTransactions.Data == nil {
			utils.ShowErrorLogs(fmt.Errorf("[TiktokServiceImpl] [ScanOrderStatementTransaction] [GetStatementTransaction] statementTransactions.Data is nil shop: %s , orderID: %s", shop.ShopID, order.OrderID))
			continue
		}
		if len(statementTransactions.Data.SkuTransactions) == 0 {
			utils.ShowErrorLogs(fmt.Errorf("[TiktokServiceImpl] [ScanOrderStatementTransaction] [GetStatementTransaction] SkuTransactions is empty shop: %s , orderID: %s", shop.ShopID, order.OrderID))
			continue
		}

		utils.ShowInfoLogs("[TiktokServiceImpl] [ScanOrderStatementTransaction] [GetStatementTransaction] order: %s , statementTransactions: %s", utils.MarshalToJson(order), utils.MarshalToJson(statementTransactions))

		transactions, err := json.Marshal(statementTransactions.Data)
		if err != nil {
			utils.ShowErrorLogs(fmt.Errorf("[TiktokServiceImpl] [ScanOrderStatementTransaction] [GetStatementTransaction] Marshal Data err: %s", err))
		}

		statementTrans = append(statementTrans, models.EcomStatementTransaction{
			MarketPlace: utils.TikTok,
			OrderID:     order.OrderID,
			RawData:     transactions,
			CreatedAt:   time.Unix(int64(statementTransactions.Data.OrderCreateTime), 0),
		})
	}

	utils.ShowInfoLogs("[TiktokServiceImpl] [ScanOrderStatementTransaction] total statementTrans: %d , shop: %s", len(statementTrans), shop.ShopID)

	return statementTrans, nil
}
