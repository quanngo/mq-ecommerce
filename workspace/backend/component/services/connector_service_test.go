package services

import (
	"context"
	"errors"
	"main/component/models"
	"main/middleware"
	"main/utils"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

type fakeEcomOrderStore struct {
	created     bool
	createErr   error
	affected    int64
	updateErr   error
	gotOrgId    int
	gotOrder    *models.CreateEcommerceRequest
	gotDOUpdate *models.UpdateDOStatusRequest
}

func (f *fakeEcomOrderStore) CreateOrder(_ context.Context, orgId int, order models.CreateEcommerceRequest) (bool, error) {
	f.gotOrgId = orgId
	f.gotOrder = &order
	return f.created, f.createErr
}

func (f *fakeEcomOrderStore) UpdateDOStatus(_ context.Context, req models.UpdateDOStatusRequest) (int64, error) {
	f.gotDOUpdate = &req
	return f.affected, f.updateErr
}

// newConnectorServer fakes both Connector endpoints and counts the calls it receives.
func newConnectorServer(t *testing.T) (*httptest.Server, *int32) {
	t.Helper()
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"code":200,"message":"Success"}`))
	}))
	t.Cleanup(srv.Close)
	return srv, &calls
}

func newTestConnectorService(connectorEnable bool, connectorURL string, store ecomOrderStore) ConnectorService {
	cfg := &utils.Config{}
	cfg.Server.OrgId = 9
	cfg.Connector.Enable = connectorEnable
	cfg.Connector.Host = connectorURL + "/"
	cfg.Connector.Callback = connectorURL + "/"
	return NewConnectorService(cfg, middleware.NewMiddleware(cfg, nil), store)
}

func sampleLead() models.CreateEcommerceRequest {
	return models.CreateEcommerceRequest{
		MarketPlace:  utils.TikTok,
		OrderID:      "576000000000000001",
		TrackingCode: "TH0001",
		Status:       utils.LeadCreated,
		Products:     []models.ConnectorProduct{{Name: "Serum", Sku: "SR-01", Quantity: 2, Price: 199}},
	}
}

func TestCreateLead_WritesDBAndSkipsConnectorWhenDisabled(t *testing.T) {
	srv, calls := newConnectorServer(t)
	store := &fakeEcomOrderStore{created: true}
	svc := newTestConnectorService(false, srv.URL, store)

	if err := svc.CreateLead(context.Background(), sampleLead()); err != nil {
		t.Fatalf("CreateLead() error = %v", err)
	}
	if store.gotOrder == nil || store.gotOrder.OrderID != "576000000000000001" {
		t.Fatalf("order not written to DB: %+v", store.gotOrder)
	}
	if store.gotOrgId != 9 {
		t.Errorf("orgId = %d, want 9", store.gotOrgId)
	}
	if *calls != 0 {
		t.Errorf("Connector called %d times, want 0", *calls)
	}
}

func TestCreateLead_ForwardsToConnectorWhenEnabled(t *testing.T) {
	srv, calls := newConnectorServer(t)
	svc := newTestConnectorService(true, srv.URL, &fakeEcomOrderStore{created: true})

	if err := svc.CreateLead(context.Background(), sampleLead()); err != nil {
		t.Fatalf("CreateLead() error = %v", err)
	}
	if *calls != 1 {
		t.Errorf("Connector called %d times, want 1", *calls)
	}
}

func TestCreateLead_DuplicateIsNotForwarded(t *testing.T) {
	srv, calls := newConnectorServer(t)
	svc := newTestConnectorService(true, srv.URL, &fakeEcomOrderStore{created: false})

	if err := svc.CreateLead(context.Background(), sampleLead()); err != nil {
		t.Fatalf("CreateLead() error = %v", err)
	}
	if *calls != 0 {
		t.Errorf("Connector called %d times for duplicate order, want 0", *calls)
	}
}

func TestCreateLead_DBErrorIsReturnedAndConnectorSkipped(t *testing.T) {
	srv, calls := newConnectorServer(t)
	dbErr := errors.New("connection refused")
	svc := newTestConnectorService(true, srv.URL, &fakeEcomOrderStore{createErr: dbErr})

	err := svc.CreateLead(context.Background(), sampleLead())
	if !errors.Is(err, dbErr) {
		t.Fatalf("CreateLead() error = %v, want wrapping %v", err, dbErr)
	}
	if *calls != 0 {
		t.Errorf("Connector called %d times after DB error, want 0", *calls)
	}
}

func TestUpdateDOStatus_WritesDB(t *testing.T) {
	srv, calls := newConnectorServer(t)
	store := &fakeEcomOrderStore{affected: 0}
	svc := newTestConnectorService(false, srv.URL, store)

	req := models.UpdateDOStatusRequest{OrderID: "1", TrackingCode: "TH0001", Status: "IN_TRANSIT", MarketPlace: utils.Lazada}
	if err := svc.UpdateDOStatus(req); err != nil {
		t.Fatalf("UpdateDOStatus() error = %v (no matching row must not be an error)", err)
	}
	if store.gotDOUpdate == nil || *store.gotDOUpdate != req {
		t.Errorf("DB update = %+v, want %+v", store.gotDOUpdate, req)
	}
	if *calls != 0 {
		t.Errorf("Connector called %d times, want 0", *calls)
	}
}

func TestUpdateDOStatus_ForwardsToConnectorWhenEnabled(t *testing.T) {
	srv, calls := newConnectorServer(t)
	svc := newTestConnectorService(true, srv.URL, &fakeEcomOrderStore{affected: 1})

	if err := svc.UpdateDOStatus(models.UpdateDOStatusRequest{TrackingCode: "TH0001", Status: "delivered"}); err != nil {
		t.Fatalf("UpdateDOStatus() error = %v", err)
	}
	if *calls != 1 {
		t.Errorf("Connector called %d times, want 1", *calls)
	}
}

func TestUpdateDOStatus_DBErrorIsReturned(t *testing.T) {
	srv, _ := newConnectorServer(t)
	dbErr := errors.New("deadlock")
	svc := newTestConnectorService(false, srv.URL, &fakeEcomOrderStore{updateErr: dbErr})

	if err := svc.UpdateDOStatus(models.UpdateDOStatusRequest{OrderID: "1"}); !errors.Is(err, dbErr) {
		t.Fatalf("UpdateDOStatus() error = %v, want wrapping %v", err, dbErr)
	}
}
