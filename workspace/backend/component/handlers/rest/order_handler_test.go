package rest

import (
	"context"
	"encoding/json"
	"main/component/models"
	"main/component/services"
	"main/utils"
	"net/http"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type fakeOrdersSvc struct {
	gotQuery *models.OrderListQuery
	gotID    int64
	err      error
}

func (f *fakeOrdersSvc) ListOrders(_ context.Context, q models.OrderListQuery) (*models.OrderListResponse, error) {
	f.gotQuery = &q
	if f.err != nil {
		return nil, f.err
	}
	return &models.OrderListResponse{Orders: []models.OrderSummary{{ID: 1, OrderID: "A1", Phone: "0901234567"}}, Total: 1, Page: 1, PageSize: 20}, nil
}

func (f *fakeOrdersSvc) GetOrder(_ context.Context, id int64) (*models.OrderDetail, error) {
	f.gotID = id
	if f.err != nil {
		return nil, f.err
	}
	return &models.OrderDetail{OrderSummary: models.OrderSummary{ID: id, OrderID: "A1"}, Items: []models.OrderItem{{ID: 3, Sku: "SR-01"}}}, nil
}

func newOrdersRouter(user *models.SessionUser, orders *fakeOrdersSvc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	app := gin.New()
	APIRoutes(app, &ApiServer{
		Cfg:       &utils.Config{Auth: utils.AuthConfig{FrontendURL: "http://localhost:5173"}},
		AuthSvc:   &fakeAuthSvc{validToken: "tok", sessionUser: user},
		OrdersSvc: orders,
	})
	return app
}

var (
	asOrderReader = sessionWith("lister", models.PermOrdersRead)
	asOrderViewer = sessionWith("viewer", models.PermOrdersRead, models.PermOrdersDetail)
	asNoOrders    = sessionWith("auditor", models.PermUsersRead)
)

func TestOrdersRoutes_Permissions(t *testing.T) {
	cases := []struct {
		path string
		user *models.SessionUser
		want int
	}{
		{"/api/v1/orders", asOrderReader, 200},
		{"/api/v1/orders", asOrderViewer, 200},
		{"/api/v1/orders", asNoOrders, 403},
		{"/api/v1/orders/5", asOrderViewer, 200},
		{"/api/v1/orders/5", asOrderReader, 403},
		{"/api/v1/orders/5", asNoOrders, 403},
	}
	for _, tc := range cases {
		orders := &fakeOrdersSvc{}
		_, body := do(newOrdersRouter(tc.user, orders), http.MethodGet, tc.path, "", withSession)
		if body.Code != tc.want {
			t.Errorf("GET %s as %s: code %d (%s), want %d", tc.path, tc.user.RoleCode, body.Code, body.Message, tc.want)
		}
		if tc.want == 403 && (body.Message != "FORBIDDEN" || orders.gotQuery != nil || orders.gotID != 0) {
			t.Errorf("GET %s as %s: message %q, service called=%v", tc.path, tc.user.RoleCode, body.Message, orders.gotQuery != nil || orders.gotID != 0)
		}
	}
}

func TestOrdersRoutes_RequireSession(t *testing.T) {
	app := newOrdersRouter(asOrderViewer, &fakeOrdersSvc{})
	for _, path := range []string{"/api/v1/orders", "/api/v1/orders/5"} {
		if _, body := do(app, http.MethodGet, path, "", nil); body.Message != "UNAUTHENTICATED" {
			t.Errorf("%s without session: %+v", path, body)
		}
	}
}

func TestOrdersList_PassesQueryAndEnvelope(t *testing.T) {
	orders := &fakeOrdersSvc{}
	w, body := do(newOrdersRouter(asOrderReader, orders), http.MethodGet,
		"/api/v1/orders?q=0901&market_place=Tiktok&lead_status=2&from=2026-10-01&to=2026-10-14&page=2&page_size=50", "", withSession)
	want := models.OrderListQuery{Q: "0901", MarketPlace: "Tiktok", LeadStatus: "2", From: "2026-10-01", To: "2026-10-14", Page: "2", PageSize: "50"}
	if orders.gotQuery == nil || *orders.gotQuery != want {
		t.Fatalf("query = %+v, want %+v", orders.gotQuery, want)
	}
	if body.Code != 200 || len(body.Data) != 1 {
		t.Fatalf("body = %+v", body)
	}
	var res models.OrderListResponse
	if err := json.Unmarshal(body.Data[0], &res); err != nil || res.Total != 1 || res.Orders[0].Phone != "0901234567" {
		t.Errorf("data = %s (%v)", body.Data[0], err)
	}
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("Cache-Control = %q", w.Header().Get("Cache-Control"))
	}
}

func TestOrdersGet_DetailHasItemsButNoRawOrder(t *testing.T) {
	orders := &fakeOrdersSvc{}
	_, body := do(newOrdersRouter(asOrderViewer, orders), http.MethodGet, "/api/v1/orders/42", "", withSession)
	if orders.gotID != 42 || body.Code != 200 || len(body.Data) != 1 {
		t.Fatalf("id=%d body=%+v", orders.gotID, body)
	}
	raw := string(body.Data[0])
	if !strings.Contains(raw, `"items":[{"id":3`) || strings.Contains(raw, "raw_order") || strings.Contains(raw, "org_id") {
		t.Errorf("data = %s", raw)
	}
}

func TestOrdersRoutes_Errors(t *testing.T) {
	cases := []struct {
		path    string
		svcErr  error
		code    int
		message string
	}{
		{"/api/v1/orders?lead_status=9", services.ErrInvalidRequest, 400, "INVALID_REQUEST"},
		{"/api/v1/orders/999", services.ErrNotFound, 404, "NOT_FOUND"},
		{"/api/v1/orders/abc", nil, 404, "NOT_FOUND"},
		{"/api/v1/orders/0", nil, 404, "NOT_FOUND"},
		{"/api/v1/orders", services.ErrInternal, 500, "INTERNAL_ERROR"},
	}
	for _, tc := range cases {
		orders := &fakeOrdersSvc{err: tc.svcErr}
		_, body := do(newOrdersRouter(asOrderViewer, orders), http.MethodGet, tc.path, "", withSession)
		if body.Code != tc.code || body.Message != tc.message {
			t.Errorf("GET %s: %d %s, want %d %s", tc.path, body.Code, body.Message, tc.code, tc.message)
		}
	}
}
