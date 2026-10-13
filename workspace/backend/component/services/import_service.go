package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"main/component/models"
	"main/utils"
	"os"
	"strconv"
	"strings"

	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

const MaxProductsPerOrder = 5

type ImportService interface {
	ImportExcel()
}

type ImportServiceImpl struct {
	cfg              *utils.Config
	connectorService ConnectorService
}

func NewImportService(
	cfg *utils.Config,
	connectorService ConnectorService) ImportService {

	return &ImportServiceImpl{
		cfg:              cfg,
		connectorService: connectorService,
	}
}

func (s *ImportServiceImpl) ImportExcel() {
	ctx := context.Background()

	const (
		idColName     = "Tracking Code"
		statusColName = "Status"
	)

	sheetsService, err := newSheetsService(ctx)
	if err != nil {
		log.Printf("[ImportExcel] newSheetsService error: %v", err)
		return
	}

	headerIdx, headerLen, err := findHeaderIndexes(ctx, sheetsService, s.cfg.GoogleSheetAPI.SpreadsheetID, s.cfg.GoogleSheetAPI.SheetName)
	if err != nil || headerLen == 0 {
		log.Printf("[ImportExcel] Invalid header: %v", err)
		return
	}

	requiredCols := []string{idColName, statusColName, "Delivery Status"}
	for _, col := range requiredCols {
		if _, ok := headerIdx[col]; !ok {
			log.Printf("[ImportExcel] Required column '%s' not found", col)
			return
		}
	}

	readRange := fmt.Sprintf("%s!A2:%s", s.cfg.GoogleSheetAPI.SheetName, colIndexToName(headerLen-1))
	resp, err := sheetsService.Spreadsheets.Values.Get(s.cfg.GoogleSheetAPI.SpreadsheetID, readRange).Context(ctx).Do()
	if err != nil || len(resp.Values) == 0 {
		log.Printf("[ImportExcel] Failed to read data: %v", err)
		return
	}

	updates := make(map[string]string)
	for _, row := range resp.Values {
		if updateStatus, err := s.processRow(ctx, headerIdx, row); err == nil && updateStatus != "" {
			trackingCode := cell(row, headerIdx[idColName])
			updates[trackingCode] = updateStatus
		}
	}

	if len(updates) == 0 {
		return
	}

	if _, err := updateStatusByIDBatch(ctx, sheetsService, s.cfg.GoogleSheetAPI.SpreadsheetID, s.cfg.GoogleSheetAPI.SheetName, idColName, statusColName, updates); err != nil {
		log.Printf("[ImportExcel] updateStatusByIDBatch error: %v", err)
	}
}

func (s *ImportServiceImpl) processRow(ctx context.Context, headerIdx map[string]int, row []interface{}) (string, error) {
	if isRowEmpty(row) {
		return "", nil
	}

	get := func(col string) string {
		if idx, ok := headerIdx[col]; ok {
			return cell(row, idx)
		}
		return ""
	}

	trackingCode := get("Tracking Code")
	if trackingCode == "" {
		return "", nil
	}

	status := strings.ToLower(get("Status"))
	deliveryStatus := strings.ToLower(get("Delivery Status"))
	utils.ShowInfoLogs("[ImportExcel] processRow: %s, status: %s, deliveryStatus: %s", trackingCode, status, deliveryStatus)

	if status == "" && deliveryStatus == "new" {
		products, totalAmount := ParseProductsFromRow(headerIdx, row, MaxProductsPerOrder)

		paymentInt := 0
		paymentMethod := strings.ToLower(get("Payment Method"))
		switch paymentMethod {
		case "cod":
			paymentInt = utils.PAYMENT_METHOD_COD
		case "ecommerce":
			paymentInt = utils.PAYMENT_METHOD_ECOMMERCE
		}

		disCount, err := parseFloatSafe(get("Discount"))
		if err != nil {
			disCount = 0
		}

		shippingFee, err := parseFloatSafe(get("Shipping Fee"))
		if err != nil {
			shippingFee = 0
		}

		lead := models.CreateEcommerceRequest{
			TrackingCode:  trackingCode,
			MarketPlace:   utils.Plaform(get("Market Place")),
			OrderID:       get("Order ID"),
			ShopID:        get("Shop ID"),
			Name:          get("Name"),
			Phone:         get("Phone"),
			Address:       get("Address"),
			Province:      get("Province"),
			District:      get("District"),
			Subdistrict:   get("SubDistrict"),
			Neighborhood:  get("Neighborhood"),
			ShippingLabel: get("Shipping Label"),
			Warehouse:     get("Warehouse"),
			ShippingFee:   shippingFee,
			DisCount:      disCount,
			PaymentMethod: paymentInt,
			Products:      products,
			TotalAmount:   totalAmount,
		}

		if !isLeadValid(lead) {
			log.Printf("[ImportExcel] Invalid lead data for (%s)", trackingCode)
			return "", nil
		}

		if err := s.connectorService.CreateLead(ctx, lead); err != nil {
			log.Printf("[ImportExcel] CreateLead failed (%s): %v", trackingCode, err)
			return "", err
		}

		return "Delivering", nil
	} else if status == "delivering" && deliveryStatus != "new" {
		err := s.connectorService.UpdateDOStatus(models.UpdateDOStatusRequest{
			TrackingCode: trackingCode,
			Status:       deliveryStatus,
		})

		return "Done", err
	}

	return "", nil
}

func newSheetsService(ctx context.Context) (*sheets.Service, error) {
	credPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if credPath == "" {
		credPath = "./config/service-account.json"
	}
	credJSON, err := os.ReadFile(credPath)
	if err != nil {
		return nil, fmt.Errorf("[ConnectorService] [newSheetsService] error: %w", err)
	}

	return sheets.NewService(
		ctx,
		option.WithCredentialsJSON(credJSON),
		option.WithScopes(sheets.SpreadsheetsScope),
	)
}

func findHeaderIndexes(ctx context.Context, srv *sheets.Service, spreadsheetID, sheetName string) (map[string]int, int, error) {
	resp, err := srv.Spreadsheets.Values.Get(spreadsheetID, sheetName+"!1:1").Context(ctx).Do()
	if err != nil {
		return nil, 0, err
	}
	if len(resp.Values) == 0 {
		return nil, 0, errors.New("[ConnectorService] [findHeaderIndexes] headers not found")
	}
	header := resp.Values[0]
	idx := make(map[string]int, len(header))
	for i, v := range header {
		name := strings.TrimSpace(fmt.Sprint(v))
		if name == "" {
			continue
		}
		idx[name] = i
	}
	return idx, len(header), nil
}

func buildIDRowMap(ctx context.Context, srv *sheets.Service, spreadsheetID, sheetName, idColName string, headerIdx map[string]int) (map[string]int, error) {
	idIdx, ok := headerIdx[idColName]
	if !ok {
		return nil, fmt.Errorf("[ConnectorService] [buildIDRowMap] not found column: %s", idColName)
	}
	idColA1 := colIndexToName(idIdx)
	rangeA1 := fmt.Sprintf("%s!%s2:%s", sheetName, idColA1, idColA1)

	resp, err := srv.Spreadsheets.Values.Get(spreadsheetID, rangeA1).Context(ctx).Do()
	if err != nil {
		return nil, err
	}

	rowMap := make(map[string]int, len(resp.Values))
	for i, row := range resp.Values {
		if len(row) == 0 {
			continue
		}
		key := strings.TrimSpace(fmt.Sprint(row[0]))
		if key == "" {
			continue
		}
		rowNumber := i + 2
		rowMap[key] = rowNumber
	}
	return rowMap, nil
}

func updateStatusByIDBatch(
	ctx context.Context,
	srv *sheets.Service,
	spreadsheetID, sheetName, idColName, statusColName string,
	updates map[string]string,
) ([]string, error) {
	headerIdx, headerLen, err := findHeaderIndexes(ctx, srv, spreadsheetID, sheetName)
	if err != nil {
		return nil, err
	}
	if headerLen == 0 {
		return nil, errors.New("[ConnectorService] [updateStatusByIDBatch] header is empty")
	}

	statusIdx, ok := headerIdx[statusColName]
	if !ok {
		return nil, fmt.Errorf("Status: %s", statusColName)
	}
	statusColA1 := colIndexToName(statusIdx)

	rowByID, err := buildIDRowMap(ctx, srv, spreadsheetID, sheetName, idColName, headerIdx)
	if err != nil {
		return nil, err
	}

	var data []*sheets.ValueRange
	var missing []string

	for id, status := range updates {
		rowNum, ok := rowByID[id]
		if !ok {
			missing = append(missing, id)
			continue
		}
		cellA1 := fmt.Sprintf("%s!%s%d", sheetName, statusColA1, rowNum)
		data = append(data, &sheets.ValueRange{
			Range:          cellA1,
			MajorDimension: "ROWS",
			Values:         [][]interface{}{{status}},
		})
	}

	if len(data) == 0 {
		return missing, nil
	}

	req := &sheets.BatchUpdateValuesRequest{
		ValueInputOption: "USER_ENTERED",
		Data:             data,
	}
	_, err = srv.Spreadsheets.Values.BatchUpdate(spreadsheetID, req).Context(ctx).Do()
	return missing, err
}

func ParseProductsFromRow(headerIdx map[string]int, row []interface{}, maxN int) ([]models.ConnectorProduct, float64) {
	var products []models.ConnectorProduct
	var total float64

	for i := 1; i <= maxN; i++ {
		keyName := fmt.Sprintf("Product_%d", i)
		keyQty := fmt.Sprintf("Quantity_%d", i)
		keyPrice := fmt.Sprintf("Price_%d", i)

		nameIdx, okName := headerIdx[keyName]
		qtyIdx, okQty := headerIdx[keyQty]
		priceIdx, okPrice := headerIdx[keyPrice]

		if !okName || !okQty || !okPrice {
			continue
		}

		name := cell(row, nameIdx)
		qtyStr := cell(row, qtyIdx)
		priceStr := cell(row, priceIdx)

		if name == "" && qtyStr == "" && priceStr == "" {
			continue
		}

		if name == "" {
			continue
		}

		qty, err := parseIntSafe(qtyStr)
		if err != nil {
			qty = 0
		}
		if qty <= 0 {
			continue
		}

		price, err := parseFloatSafe(priceStr)
		if err != nil {
			price = 0
		}

		p := models.ConnectorProduct{
			Name:     name,
			Quantity: qty,
			Price:    price,
		}
		products = append(products, p)
		total += float64(qty) * price
	}

	return products, total
}

func isRowEmpty(row []interface{}) bool {
	for _, v := range row {
		if strings.TrimSpace(fmt.Sprint(v)) != "" {
			return false
		}
	}
	return true
}

func cell(row []interface{}, idx int) string {
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(row[idx]))
}

func colIndexToName(n int) string {
	n++
	name := ""
	for n > 0 {
		n--
		name = string(rune('A'+(n%26))) + name
		n /= 26
	}
	return name
}

func parseIntSafe(s string) (int, error) {
	if s == "" {
		return 0, nil
	}

	s = strings.ReplaceAll(s, ",", "")

	if i, err := strconv.Atoi(s); err == nil {
		return i, nil
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return int(f), nil
	}
	return 0, fmt.Errorf("parseIntSafe: %q", s)
}

func parseFloatSafe(s string) (float64, error) {
	if s == "" {
		return 0, nil
	}
	s = strings.ReplaceAll(s, ",", "")
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("parseFloatSafe: %q", s)
	}
	return f, nil
}

func isLeadValid(lead models.CreateEcommerceRequest) bool {

	utils.ShowInfoLogs("[ImportServiceImpl] [isLeadValid] lead: %s", utils.MarshalToJson(lead))

	return len(lead.Products) > 0 &&
		lead.MarketPlace != "" &&
		lead.ShopID != "" &&
		lead.OrderID != "" &&
		lead.TrackingCode != "" &&
		lead.Name != "" &&
		lead.Phone != "" &&
		// lead.Address != "" &&
		// lead.Province != "" &&
		// lead.District != "" &&
		// lead.Subdistrict != "" &&
		lead.ShippingLabel != "" &&
		lead.Warehouse != "" &&
		lead.PaymentMethod > 0
}
