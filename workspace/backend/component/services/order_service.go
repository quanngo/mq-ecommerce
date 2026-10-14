package services

import (
	"context"
	"fmt"
	"main/component/models"
	"main/utils"
	"strconv"
	"strings"
	"time"
)

// OrderService backs the admin app's Orders menu (EC-4): a filtered, paginated
// list and one order's detail. Read-only. Routes check orders.read /
// orders.detail before calling it.
type OrderService interface {
	ListOrders(ctx context.Context, q models.OrderListQuery) (*models.OrderListResponse, error)
	GetOrder(ctx context.Context, id int64) (*models.OrderDetail, error)
}

type orderQueryStore interface {
	List(ctx context.Context, f models.OrderFilter) ([]models.OrderSummary, int, error)
	Get(ctx context.Context, id int64) (*models.OrderDetail, error)
}

const (
	defaultOrderPageSize = 20
	maxOrderPageSize     = 100
	maxOrderSearchLen    = 100
)

// Vietnam has no DST, so a fixed zone avoids depending on tzdata (missing on
// some Windows/Alpine hosts).
var vietnamTime = time.FixedZone(utils.TimezoneVietnam, 7*60*60)

type OrderServiceImpl struct {
	orders orderQueryStore
}

func NewOrderService(orders orderQueryStore) OrderService {
	return &OrderServiceImpl{orders: orders}
}

func (s *OrderServiceImpl) ListOrders(ctx context.Context, q models.OrderListQuery) (*models.OrderListResponse, error) {
	f, page, err := parseOrderQuery(q)
	if err != nil {
		return nil, err
	}
	orders, total, err := s.orders.List(ctx, f)
	if err != nil {
		return nil, wrapAuth(ErrInternal, fmt.Errorf("[OrderService] [ListOrders] %w", err))
	}
	return &models.OrderListResponse{Orders: orders, Total: total, Page: page, PageSize: f.Limit}, nil
}

func (s *OrderServiceImpl) GetOrder(ctx context.Context, id int64) (*models.OrderDetail, error) {
	order, err := s.orders.Get(ctx, id)
	if err != nil {
		return nil, wrapAuth(ErrInternal, fmt.Errorf("[OrderService] [GetOrder] id=%d: %w", id, err))
	}
	if order == nil {
		return nil, ErrNotFound
	}
	return order, nil
}

// parseOrderQuery validates the query string. Any bad value is INVALID_REQUEST;
// the cause names the parameter for the server log only.
func parseOrderQuery(q models.OrderListQuery) (models.OrderFilter, int, error) {
	invalid := func(format string, a ...interface{}) (models.OrderFilter, int, error) {
		return models.OrderFilter{}, 0, wrapAuth(ErrInvalidRequest, fmt.Errorf(format, a...))
	}
	f := models.OrderFilter{Q: strings.TrimSpace(q.Q)}
	if len([]rune(f.Q)) > maxOrderSearchLen {
		return invalid("q longer than %d characters", maxOrderSearchLen)
	}

	switch mp := utils.Plaform(strings.TrimSpace(q.MarketPlace)); mp {
	case "":
	case utils.TikTok, utils.Lazada, utils.Shopee:
		f.MarketPlace = string(mp)
	default:
		return invalid("unknown market_place %q", q.MarketPlace)
	}

	if s := strings.TrimSpace(q.LeadStatus); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n < int(utils.LeadPending) || n > int(utils.LeadCancel) {
			return invalid("lead_status %q", q.LeadStatus)
		}
		f.LeadStatus = n
	}

	from, err := parseOrderDate(q.From)
	if err != nil {
		return invalid("from: %v", err)
	}
	to, err := parseOrderDate(q.To)
	if err != nil {
		return invalid("to: %v", err)
	}
	if from != nil && to != nil && from.After(*to) {
		return invalid("from %s after to %s", q.From, q.To)
	}
	f.CreatedFrom = from
	if to != nil {
		next := to.AddDate(0, 0, 1) // `to` is inclusive
		f.CreatedTo = &next
	}

	page, err := parseBoundedInt(q.Page, 1, 1, 1<<30)
	if err != nil {
		return invalid("page: %v", err)
	}
	size, err := parseBoundedInt(q.PageSize, defaultOrderPageSize, 1, maxOrderPageSize)
	if err != nil {
		return invalid("page_size: %v", err)
	}
	f.Limit, f.Offset = size, (page-1)*size
	return f, page, nil
}

// parseOrderDate reads YYYY-MM-DD as midnight Vietnam time; "" is no date.
func parseOrderDate(s string) (*time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	t, err := time.ParseInLocation(utils.DATE_TIME_FORMAT_YYYY_MM_DD, s, vietnamTime)
	if err != nil {
		return nil, fmt.Errorf("%q is not YYYY-MM-DD", s)
	}
	return &t, nil
}

func parseBoundedInt(s string, def, min, max int) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return def, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < min || n > max {
		return 0, fmt.Errorf("%q not in %d..%d", s, min, max)
	}
	return n, nil
}
