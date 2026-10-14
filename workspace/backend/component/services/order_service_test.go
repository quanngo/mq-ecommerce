package services

import (
	"context"
	"errors"
	"main/component/models"
	"testing"
	"time"
)

type fakeOrderStore struct {
	gotFilter *models.OrderFilter
	orders    []models.OrderSummary
	total     int
	detail    *models.OrderDetail
	err       error
}

func (f *fakeOrderStore) List(_ context.Context, filter models.OrderFilter) ([]models.OrderSummary, int, error) {
	f.gotFilter = &filter
	return f.orders, f.total, f.err
}

func (f *fakeOrderStore) Get(_ context.Context, id int64) (*models.OrderDetail, error) {
	return f.detail, f.err
}

func TestListOrders_Defaults(t *testing.T) {
	store := &fakeOrderStore{orders: []models.OrderSummary{{ID: 1}}, total: 41}
	res, err := NewOrderService(store).ListOrders(context.Background(), models.OrderListQuery{})
	if err != nil {
		t.Fatal(err)
	}
	f := store.gotFilter
	if f.Limit != 20 || f.Offset != 0 || f.Q != "" || f.MarketPlace != "" || f.LeadStatus != 0 || f.CreatedFrom != nil || f.CreatedTo != nil {
		t.Errorf("filter = %+v", f)
	}
	if res.Total != 41 || res.Page != 1 || res.PageSize != 20 || len(res.Orders) != 1 {
		t.Errorf("response = %+v", res)
	}
}

func TestListOrders_Filters(t *testing.T) {
	store := &fakeOrderStore{}
	_, err := NewOrderService(store).ListOrders(context.Background(), models.OrderListQuery{
		Q: "  0901 ", MarketPlace: "Lazada", LeadStatus: "4",
		From: "2026-10-01", To: "2026-10-14", Page: "3", PageSize: "50",
	})
	if err != nil {
		t.Fatal(err)
	}
	f := store.gotFilter
	if f.Q != "0901" || f.MarketPlace != "Lazada" || f.LeadStatus != 4 || f.Limit != 50 || f.Offset != 100 {
		t.Errorf("filter = %+v", f)
	}
	// Dates are whole days in Vietnam time; `to` is inclusive.
	wantFrom := time.Date(2026, 9, 30, 17, 0, 0, 0, time.UTC)
	wantTo := time.Date(2026, 10, 14, 17, 0, 0, 0, time.UTC)
	if f.CreatedFrom == nil || !f.CreatedFrom.Equal(wantFrom) {
		t.Errorf("CreatedFrom = %v, want %v", f.CreatedFrom, wantFrom)
	}
	if f.CreatedTo == nil || !f.CreatedTo.Equal(wantTo) {
		t.Errorf("CreatedTo = %v, want %v", f.CreatedTo, wantTo)
	}
}

func TestListOrders_SameDayRange(t *testing.T) {
	store := &fakeOrderStore{}
	if _, err := NewOrderService(store).ListOrders(context.Background(), models.OrderListQuery{From: "2026-10-14", To: "2026-10-14"}); err != nil {
		t.Fatal(err)
	}
	if d := store.gotFilter.CreatedTo.Sub(*store.gotFilter.CreatedFrom); d != 24*time.Hour {
		t.Errorf("one-day range spans %v", d)
	}
}

func TestListOrders_InvalidQuery(t *testing.T) {
	long := make([]byte, 101)
	for i := range long {
		long[i] = 'a'
	}
	cases := map[string]models.OrderListQuery{
		"market place":      {MarketPlace: "Amazon"},
		"lead status text":  {LeadStatus: "x"},
		"lead status range": {LeadStatus: "5"},
		"lead status zero":  {LeadStatus: "0"},
		"from format":       {From: "14/10/2026"},
		"to format":         {To: "2026-13-01"},
		"from after to":     {From: "2026-10-15", To: "2026-10-14"},
		"page zero":         {Page: "0"},
		"page text":         {Page: "two"},
		"page size zero":    {PageSize: "0"},
		"page size max":     {PageSize: "101"},
		"search too long":   {Q: string(long)},
	}
	for name, q := range cases {
		store := &fakeOrderStore{}
		_, err := NewOrderService(store).ListOrders(context.Background(), q)
		if !errors.Is(err, ErrInvalidRequest) {
			t.Errorf("%s: err = %v, want INVALID_REQUEST", name, err)
		}
		if store.gotFilter != nil {
			t.Errorf("%s: store called", name)
		}
	}
}

func TestListOrders_StoreError(t *testing.T) {
	_, err := NewOrderService(&fakeOrderStore{err: errors.New("db down")}).ListOrders(context.Background(), models.OrderListQuery{})
	if !errors.Is(err, ErrInternal) {
		t.Errorf("err = %v, want INTERNAL_ERROR", err)
	}
}

func TestGetOrder(t *testing.T) {
	ctx := context.Background()
	want := &models.OrderDetail{OrderSummary: models.OrderSummary{ID: 5}, Items: []models.OrderItem{{ID: 1}}}
	got, err := NewOrderService(&fakeOrderStore{detail: want}).GetOrder(ctx, 5)
	if err != nil || got != want {
		t.Errorf("GetOrder = %+v, %v", got, err)
	}
	if _, err := NewOrderService(&fakeOrderStore{}).GetOrder(ctx, 6); !errors.Is(err, ErrNotFound) {
		t.Errorf("missing order: %v, want NOT_FOUND", err)
	}
	if _, err := NewOrderService(&fakeOrderStore{err: errors.New("db down")}).GetOrder(ctx, 7); !errors.Is(err, ErrInternal) {
		t.Errorf("store error: %v, want INTERNAL_ERROR", err)
	}
}
