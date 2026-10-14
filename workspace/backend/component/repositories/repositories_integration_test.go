package repositories

// Integration tests: run every repository query against a real Postgres that
// has migrations/*.sql and migrations/dev/connector_tables.sql applied.
//
//   ECOM_TEST_DATABASE_URL=postgres://postgres@127.0.0.1:55432/mq_test?sslmode=disable go test ./component/repositories/
//
// Skipped when the variable is unset. Each test TRUNCATEs every table it
// uses, so the database name must contain "test".

import (
	"context"
	"database/sql"
	"encoding/json"
	"main/component/models"
	tiktok_model "main/component/models/tiktok_model"
	"main/utils"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

var allTables = []string{
	"public.market_token", "public.market_place_mapping", "public.log_market_place_order",
	"public.log_order_status", "public.ecom_statement_transaction",
	"public.ecom_order_item", "public.ecom_order",
	"public.od_ecom_order", "public.od_do_new", "public.od_sale_order",
	"public.user_session", "public.oauth_login_attempt", "public.app_user",
}

func testDB(t *testing.T) *sqlx.DB {
	t.Helper()
	dsn := os.Getenv("ECOM_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("ECOM_TEST_DATABASE_URL not set")
	}
	u, err := url.Parse(dsn)
	if err != nil || !strings.Contains(strings.TrimPrefix(u.Path, "/"), "test") {
		t.Fatalf("refusing to TRUNCATE: database name in ECOM_TEST_DATABASE_URL must contain \"test\"")
	}
	db, err := sqlx.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec("TRUNCATE " + strings.Join(allTables, ", ") + " RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("truncate (are migrations applied?): %v", err)
	}
	return db
}

func mustExec(t *testing.T, db *sqlx.DB, query string, args ...interface{}) {
	t.Helper()
	if _, err := db.Exec(query, args...); err != nil {
		t.Fatalf("%s: %v", query, err)
	}
}

func TestEcomOrderRepo(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	repo := NewEcomOrderRepo(db)

	order := models.CreateEcommerceRequest{
		MarketPlace: utils.TikTok, OrderID: "A1", TrackingCode: "TRK1", ShopID: "shop1",
		Status: utils.LeadCreated, Name: "Somchai", TotalAmount: 398, PaymentMethod: utils.PAYMENT_METHOD_COD,
		Products:     []models.ConnectorProduct{{Name: "Serum", Sku: "SR-01", Quantity: 2, Price: 199}, {Name: "Toner", Sku: "TN-01", Quantity: 1, Price: 0}},
		RawOrderByte: map[string]string{"id": "A1"},
	}
	created, err := repo.CreateOrder(ctx, 9, order)
	if err != nil || !created {
		t.Fatalf("CreateOrder() = %v, %v; want true, nil", created, err)
	}
	var items int
	if err := db.Get(&items, "SELECT COUNT(*) FROM public.ecom_order_item"); err != nil || items != 2 {
		t.Fatalf("items = %d (%v), want 2", items, err)
	}

	created, err = repo.CreateOrder(ctx, 9, order)
	if err != nil || created {
		t.Fatalf("duplicate CreateOrder() = %v, %v; want false, nil", created, err)
	}
	if err := db.Get(&items, "SELECT COUNT(*) FROM public.ecom_order_item"); err != nil || items != 2 {
		t.Fatalf("items after duplicate = %d (%v), want 2", items, err)
	}

	// Sheet imports have no order_id: two such rows must not conflict.
	for _, trk := range []string{"SHEET1", "SHEET2"} {
		if created, err := repo.CreateOrder(ctx, 9, models.CreateEcommerceRequest{MarketPlace: "Sheet", TrackingCode: trk, Status: utils.LeadCreated}); err != nil || !created {
			t.Fatalf("CreateOrder(no order_id, %s) = %v, %v", trk, created, err)
		}
	}

	// By order_id, scoped to market_place.
	n, err := repo.UpdateDOStatus(ctx, models.UpdateDOStatusRequest{OrderID: "A1", MarketPlace: utils.TikTok, Status: "IN_TRANSIT"})
	if err != nil || n != 1 {
		t.Fatalf("UpdateDOStatus(order) = %d, %v; want 1", n, err)
	}
	if n, _ := repo.UpdateDOStatus(ctx, models.UpdateDOStatusRequest{OrderID: "A1", MarketPlace: utils.Lazada, Status: "X"}); n != 0 {
		t.Fatalf("UpdateDOStatus(wrong market_place) updated %d rows, want 0", n)
	}
	// so_status only: the DO status must be kept.
	if n, err := repo.UpdateDOStatus(ctx, models.UpdateDOStatusRequest{OrderID: "A1", SoStatus: utils.SaleOrderCompleted}); err != nil || n != 1 {
		t.Fatalf("UpdateDOStatus(so_status) = %d, %v", n, err)
	}
	var got struct {
		DoStatus string `db:"do_status"`
		SoStatus *int   `db:"so_status"`
	}
	if err := db.Get(&got, "SELECT do_status, so_status FROM public.ecom_order WHERE order_id = 'A1'"); err != nil {
		t.Fatal(err)
	}
	if got.DoStatus != "IN_TRANSIT" || got.SoStatus == nil || *got.SoStatus != int(utils.SaleOrderCompleted) {
		t.Fatalf("row = %+v, want do_status IN_TRANSIT, so_status %d", got, utils.SaleOrderCompleted)
	}
	// By tracking_code when there is no order_id.
	if n, err := repo.UpdateDOStatus(ctx, models.UpdateDOStatusRequest{TrackingCode: "SHEET2", Status: "delivered"}); err != nil || n != 1 {
		t.Fatalf("UpdateDOStatus(tracking) = %d, %v; want 1", n, err)
	}
}

func TestOrderQueryRepo(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	writer, repo := NewEcomOrderRepo(db), NewOrderQueryRepo(db)

	seed := []models.CreateEcommerceRequest{
		{MarketPlace: utils.TikTok, OrderID: "TT-1", TrackingCode: "TRK-A", Name: "Nguyễn Văn An", Phone: "0901111111", Status: utils.LeadCreated, TotalAmount: 398,
			Address: "1 Lê Lợi", Province: "HCM", DisCount: 10, ShippingFee: 15,
			Products: []models.ConnectorProduct{{Name: "Serum", Sku: "SR-01", Quantity: 2, Price: 199}, {Name: "Toner", Sku: "TN-01", Quantity: 1, Price: 0}}},
		{MarketPlace: utils.Lazada, OrderID: "LZ-2", Name: "Trần Thị Bình", Phone: "0902222222", Status: utils.LeadCancel},
		{MarketPlace: utils.Lazada, OrderID: "LZ_100%", Name: "Lê C", Phone: "0903333333", Status: utils.LeadCreated},
	}
	for _, o := range seed {
		if created, err := writer.CreateOrder(ctx, 1, o); err != nil || !created {
			t.Fatalf("seed %s: %v, %v", o.OrderID, created, err)
		}
	}
	// Spread creation dates: TT-1 on Oct 1, LZ-2 on Oct 10, LZ_100% on Oct 14 (Vietnam time).
	mustExec(t, db, `UPDATE public.ecom_order SET created_at = '2026-10-01 09:00+07' WHERE order_id = 'TT-1'`)
	mustExec(t, db, `UPDATE public.ecom_order SET created_at = '2026-10-10 23:30+07' WHERE order_id = 'LZ-2'`)
	mustExec(t, db, `UPDATE public.ecom_order SET created_at = '2026-10-14 08:00+07' WHERE order_id = 'LZ_100%'`)

	vn := time.FixedZone("ICT", 7*3600)
	day := func(d int) *time.Time { t := time.Date(2026, 10, d, 0, 0, 0, 0, vn); return &t }
	ids := func(list []models.OrderSummary) string {
		var s []string
		for _, o := range list {
			s = append(s, o.OrderID)
		}
		return strings.Join(s, ",")
	}

	cases := []struct {
		name  string
		f     models.OrderFilter
		want  string
		total int
	}{
		{"all, newest first", models.OrderFilter{Limit: 20}, "LZ_100%,LZ-2,TT-1", 3},
		{"page 2 of size 2", models.OrderFilter{Limit: 2, Offset: 2}, "TT-1", 3},
		{"market place", models.OrderFilter{MarketPlace: "Lazada", Limit: 20}, "LZ_100%,LZ-2", 2},
		{"lead status", models.OrderFilter{LeadStatus: int(utils.LeadCancel), Limit: 20}, "LZ-2", 1},
		{"search phone", models.OrderFilter{Q: "0902", Limit: 20}, "LZ-2", 1},
		{"search name, case-insensitive", models.OrderFilter{Q: "nguyễn văn", Limit: 20}, "TT-1", 1},
		{"search tracking code", models.OrderFilter{Q: "trk-a", Limit: 20}, "TT-1", 1},
		{"like wildcards are literal", models.OrderFilter{Q: "_100%", Limit: 20}, "LZ_100%", 1},
		{"underscore is not a wildcard", models.OrderFilter{Q: "LZ_2", Limit: 20}, "", 0},
		{"date range", models.OrderFilter{CreatedFrom: day(10), CreatedTo: day(11), Limit: 20}, "LZ-2", 1},
		{"combined", models.OrderFilter{MarketPlace: "Lazada", CreatedFrom: day(11), Q: "0903", Limit: 20}, "LZ_100%", 1},
	}
	for _, tc := range cases {
		list, total, err := repo.List(ctx, tc.f)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if got := ids(list); got != tc.want || total != tc.total {
			t.Errorf("%s: got %q (total %d), want %q (total %d)", tc.name, got, total, tc.want, tc.total)
		}
	}

	list, _, _ := repo.List(ctx, models.OrderFilter{Q: "TT-1", Limit: 20})
	if len(list) != 1 || list[0].ItemCount != 2 || list[0].TotalAmount != 398 || list[0].SoStatus != nil {
		t.Fatalf("summary = %+v", list)
	}

	detail, err := repo.Get(ctx, list[0].ID)
	if err != nil || detail == nil {
		t.Fatalf("Get = %+v, %v", detail, err)
	}
	if detail.OrderID != "TT-1" || detail.Address != "1 Lê Lợi" || detail.Discount != 10 || detail.ShippingFee != 15 ||
		len(detail.Items) != 2 || detail.Items[0].Sku != "SR-01" || detail.Items[0].Price != 199 {
		t.Errorf("detail = %+v", detail)
	}
	if missing, err := repo.Get(ctx, 999999); err != nil || missing != nil {
		t.Errorf("Get(missing) = %+v, %v; want nil, nil", missing, err)
	}
	other, _, _ := repo.List(ctx, models.OrderFilter{Q: "LZ-2", Limit: 20})
	if d, _ := repo.Get(ctx, other[0].ID); d == nil || d.Items == nil || len(d.Items) != 0 {
		t.Errorf("order without items: %+v (items must be [] not nil)", d)
	}
}

func TestTokenRepo(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	repo := NewTokenRepo(db)
	mustExec(t, db, `INSERT INTO public.market_token (market_place, shop_id, access_token, refresh_token, shop_reference, expire_time, enable)
		VALUES ('Tiktok', 'shop1', 'at', 'rt', 'cipher', NOW(), true), ('Lazada', 'shop2', 'at2', 'rt2', NULL, NOW(), false)`)

	all, err := repo.GetAll(ctx) // SELECT * must match models.MarketToken exactly
	if err != nil || len(all) != 2 {
		t.Fatalf("GetAll() = %d rows, %v", len(all), err)
	}
	if byPlace, err := repo.GetMarketToken(ctx, "Tiktok"); err != nil || len(byPlace) != 1 || byPlace[0].ShopReference == nil {
		t.Fatalf("GetMarketToken() = %+v, %v", byPlace, err)
	}
	if ids, err := repo.GetShopID(ctx, utils.TikTok); err != nil || len(ids) != 1 || ids[0] != "shop1" {
		t.Fatalf("GetShopID() = %v, %v", ids, err)
	}
	exp := time.Now().Add(time.Hour).UTC().Truncate(time.Second)
	if res, err := repo.UpdateAccessToken(ctx, utils.TikTok, "at-new", "rt-new", "shop1", exp); err != nil {
		t.Fatal(err)
	} else if n, _ := res.RowsAffected(); n != 1 {
		t.Fatalf("UpdateAccessToken() updated %d rows", n)
	}
	if rt := repo.GetRefreshToken(ctx, utils.TikTok, "shop1"); rt != "rt-new" {
		t.Fatalf("GetRefreshToken() = %q", rt)
	}
	if got := repo.GetExpireTime(ctx, utils.TikTok, "shop1"); !got.Equal(exp) {
		t.Fatalf("GetExpireTime() = %v, want %v", got, exp)
	}
}

func TestMarketPlaceMappingRepo(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	repo := NewMarketPlaceMappingRepo(db)
	mustExec(t, db, `INSERT INTO public.market_place_mapping (tms_id, ecommerce_id, quantity, name, type, market_place, org_id) VALUES
		('Serum|SR-01', 'P1', 10, 'shop1', 'product', 'Tiktok', 9),
		('Toner|TN-01', 'P2', 0, 'shop1', 'product', 'Tiktok', 9),
		('WH-BKK', 'shop1', 0, '', 'warehouse', 'Tiktok', 9),
		('AWAITING_COLLECTION', '52', 0, '', 'status', 'Tiktok', 9),
		('0', 'shop1', 0, '', 'payment configuration', 'Tiktok', 9)`)

	if p, err := repo.GetProducts(ctx, utils.TikTok, "shop1", nil); err != nil || len(p) != 2 {
		t.Fatalf("GetProducts() = %d, %v", len(p), err)
	}
	if p, err := repo.GetProducts(ctx, utils.TikTok, "shop1", []string{"P2", "P2"}); err != nil || len(p) != 1 {
		t.Fatalf("GetProducts(ids) = %d, %v", len(p), err)
	}
	if wh, err := repo.GetWarehouseByShopID(ctx, utils.TikTok, "shop1"); err != nil || wh.ConnectorID != "WH-BKK" {
		t.Fatalf("GetWarehouseByShopID() = %+v, %v", wh, err)
	}
	if s, err := repo.GetStatus(ctx, "Tiktok"); err != nil || len(s) != 1 {
		t.Fatalf("GetStatus() = %d, %v", len(s), err)
	}
	if err := repo.UpdateShopPaymentConfigValue(ctx, 42, utils.TikTok, "shop1"); err != nil {
		t.Fatal(err)
	}
	if v, err := repo.GetShopPaymentConfig(ctx, utils.TikTok, "shop1"); err != nil || v != "42" {
		t.Fatalf("GetShopPaymentConfig() = %q, %v", v, err)
	}
}

func TestLogRepos(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	orders := NewLogMarketPlaceOrderRepo(db)
	statuses := NewLogOrderStatusRepo(db)

	lead := models.CreateEcommerceRequest{MarketPlace: utils.TikTok, ShopID: "shop1", OrderID: "A1", Status: utils.LeadRefund, RawOrderByte: map[string]int{"x": 1}}
	if err := orders.CreateLogOrder(ctx, 9, lead); err != nil {
		t.Fatal(err)
	}
	if ok, err := orders.CheckOrderExists(utils.TikTok, "A1"); err != nil || !ok {
		t.Fatalf("CheckOrderExists() = %v, %v", ok, err)
	}
	if got, err := orders.GetLogOrder(ctx, models.GetLogMarketPlaceOrder{MarketPlace: utils.TikTok, OrderID: "A1"}); err != nil || len(got) != 1 {
		t.Fatalf("GetLogOrder() = %d, %v", len(got), err)
	}
	upd := models.RequestUpdateLogOrderStatus{ID: "A1", Status: utils.LeadRefund, UpdatedAt: time.Now(), MarketPlace: utils.TikTok, ShopID: "shop1"}
	if err := orders.UpdateOrderStatus(ctx, upd); err != nil {
		t.Fatal(err)
	}
	if r, err := orders.GetLatestOrderRefundTime(ctx, utils.LeadRefund, utils.TikTok, "shop1"); err != nil || r == nil || r.UpdatedAt == nil {
		t.Fatalf("GetLatestOrderRefundTime() = %+v, %v", r, err)
	}

	now := time.Now()
	if err := statuses.InsertOrderStatus(ctx, models.UpdateDatabaseOrder{MarketPlace: utils.TikTok, ShopID: "shop1", OrderID: "A1", Status: "UNPAID", CreatedDate: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := statuses.UpdateOrderStatus(ctx, utils.TikTok, "shop1", "A1", "AWAITING_SHIPMENT", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if r, err := statuses.GetLastestOrderTime(ctx, utils.TikTok, "shop1"); err != nil || r == nil || r.UpdatedAt == nil {
		t.Fatalf("GetLastestOrderTime() = %+v, %v", r, err)
	}
	if _, _, err := statuses.GetLastestOrderCreatedUpdate(ctx, "Tiktok"); err != nil {
		t.Fatalf("GetLastestOrderCreatedUpdate() error = %v", err)
	}
}

func TestConnectorRepo(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	repo := NewConnectorRepo(db)
	mustExec(t, db, `INSERT INTO public.od_ecom_order (market_place, order_id, lead_id, tracking_code, shop_id) VALUES
		('Tiktok', 'A1', 100, 'TRK1', 'shop1'), ('Tiktok', 'A2', 101, 'TRK2', 'shop1')`)
	mustExec(t, db, `INSERT INTO public.od_do_new (tracking_code, status) VALUES ('TRK1', $1)`, utils.DeliveryOrderStatusReadyToPick)
	mustExec(t, db, `INSERT INTO public.od_sale_order (lead_id, status) VALUES (100, $1), (101, $1)`, utils.SaleOrderCompleted)

	if got, err := repo.GetOdEcomOrderDObyStatus(ctx, utils.TikTok, utils.DeliveryOrderStatusReadyToPick); err != nil || len(got) != 1 || got[0].OrderID != "A1" {
		t.Fatalf("GetOdEcomOrderDObyStatus() = %+v, %v", got, err)
	}
	if got, err := repo.GetShopOrderID(ctx, utils.TikTok, "shop1", 0); err != nil || len(got) != 2 {
		t.Fatalf("GetShopOrderID() = %d, %v", len(got), err)
	}
	pay := tiktok_model.Payment{Id: 1, Currency: "THB", SubTotal: "398.00", ShippingFee: "40", SellerDiscount: "0", PlatformDiscount: "10",
		TotalAmount: "428", OriginalTotalProductPrice: "398", OriginalShippingFee: "40", ShippingFeeSellerDiscount: "0",
		ShippingFeeCofundedDiscount: "0", Tax: "0"}
	if err := repo.UpdateOrderPayment(ctx, pay); err != nil {
		t.Fatalf("UpdateOrderPayment() error = %v", err)
	}

	pending, err := repo.GetOrderIdForStatementTransaction(ctx, utils.TikTok, "shop1")
	if err != nil || len(pending) != 2 {
		t.Fatalf("GetOrderIdForStatementTransaction() = %d, %v; want 2", len(pending), err)
	}
	rec := []models.EcomStatementTransaction{{MarketPlace: utils.TikTok, OrderID: "A1", RawData: json.RawMessage(`{"v":1}`)}}
	if err := repo.InsertEcomStatementTransaction(rec); err != nil {
		t.Fatalf("InsertEcomStatementTransaction() error = %v", err)
	}
	rec[0].RawData = json.RawMessage(`{"v":2}`) // upsert via UNIQUE (market_place, order_id)
	if err := repo.InsertEcomStatementTransaction(rec); err != nil {
		t.Fatalf("InsertEcomStatementTransaction(upsert) error = %v", err)
	}
	var raw string
	if err := db.Get(&raw, "SELECT raw_data::text FROM public.ecom_statement_transaction WHERE order_id = 'A1'"); err != nil || raw != `{"v": 2}` {
		t.Fatalf("raw_data = %q, %v; want upserted value", raw, err)
	}
	if pending, _ := repo.GetOrderIdForStatementTransaction(ctx, utils.TikTok, "shop1"); len(pending) != 1 {
		t.Fatalf("pending after insert = %d, want 1", len(pending))
	}
}

func TestAuthRepos(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	users, sessions, attempts := NewUserRepo(db), NewSessionRepo(db), NewLoginAttemptRepo(db)

	hash := "$2a$04$abcdefghijklmnopqrstuuQ6v4bCw2b0mM9w1V9yQkq4m7vN3o1aS"
	id, err := users.Create(ctx, models.AppUser{Email: "an.nguyen@mq.com", DisplayName: "", PasswordHash: &hash}, "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := users.Create(ctx, models.AppUser{Email: "an.nguyen@mq.com"}, ""); err != ErrDuplicateEmail {
		t.Errorf("duplicate email: %v, want ErrDuplicateEmail", err)
	}
	if _, err := users.Create(ctx, models.AppUser{Email: "Upper@mq.com"}, ""); err == nil {
		t.Error("mixed-case email accepted; ck_app_user_email_lc missing?")
	}
	u, err := users.GetByEmail(ctx, "an.nguyen@mq.com")
	if err != nil || u == nil || u.ID != id || u.Status != models.UserStatusActive || u.PasswordHash == nil {
		t.Fatalf("GetByEmail = %+v, %v", u, err)
	}
	if u, err := users.GetByEmail(ctx, "nobody@mq.com"); u != nil || err != nil {
		t.Errorf("GetByEmail(missing) = %+v, %v", u, err)
	}

	avatar := "https://img/a.png"
	if ok, err := users.LinkGoogle(ctx, id, "g-1", "An Nguyen", &avatar); !ok || err != nil {
		t.Fatalf("LinkGoogle: %v %v", ok, err)
	}
	if ok, _ := users.LinkGoogle(ctx, id, "g-2", "x", nil); ok {
		t.Error("LinkGoogle overwrote an existing google_sub")
	}
	if u, _ := users.GetByGoogleSub(ctx, "g-1"); u == nil || u.DisplayName != "An Nguyen" {
		t.Errorf("GetByGoogleSub = %+v", u)
	}

	for i := 1; i <= 2; i++ {
		count, locked, err := users.RecordLoginFailure(ctx, id, 2, 15*time.Minute)
		if err != nil || count != i || (i == 2) != (locked != nil) {
			t.Fatalf("failure %d: count=%d locked=%v err=%v", i, count, locked, err)
		}
	}
	if _, err := users.MarkLoginSuccess(ctx, id); err != nil {
		t.Fatal(err)
	}
	if u, _ := users.GetByEmail(ctx, "an.nguyen@mq.com"); u.FailedLoginCount != 0 || u.LockedUntil != nil || u.LastLoginAt == nil {
		t.Errorf("after success: %+v", u)
	}

	// Sessions
	valid := strings.Repeat("a", 64)
	if err := sessions.Create(ctx, valid, id, models.AuthMethodGoogle, time.Now().Add(time.Hour), "ua", "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	mustExec(t, db, `INSERT INTO public.user_session (token_hash, user_id, auth_method, expires_at) VALUES ($1, $2, 'password', NOW() - interval '1 minute')`, strings.Repeat("b", 64), id)
	if su, err := sessions.GetValid(ctx, valid); err != nil || su == nil || su.ID != id || su.AuthMethod != models.AuthMethodGoogle {
		t.Fatalf("GetValid = %+v, %v", su, err)
	}
	if su, _ := sessions.GetValid(ctx, strings.Repeat("b", 64)); su != nil {
		t.Error("expired session returned")
	}
	mustExec(t, db, `UPDATE public.app_user SET status = 'disabled' WHERE id = $1`, id)
	if su, _ := sessions.GetValid(ctx, valid); su != nil {
		t.Error("session of a disabled user returned")
	}
	mustExec(t, db, `UPDATE public.app_user SET status = 'active' WHERE id = $1`, id)
	if err := sessions.Revoke(ctx, valid); err != nil {
		t.Fatal(err)
	}
	if su, _ := sessions.GetValid(ctx, valid); su != nil {
		t.Error("revoked session returned")
	}
	// Only the revoked session: the expired "b" one is less than a day past expiry and must stay.
	mustExec(t, db, `UPDATE public.user_session SET revoked_at = NOW() - interval '2 days' WHERE token_hash = $1`, valid)
	if n, err := sessions.DeleteStale(ctx, 24*time.Hour); err != nil || n != 1 {
		t.Errorf("DeleteStale = %d, %v; want 1 (the long-revoked one)", n, err)
	}

	// Login attempts
	if err := attempts.Create(ctx, models.LoginAttempt{State: "s1", Nonce: "n1", CodeVerifier: "v1", RedirectPath: "/profile", ExpiresAt: time.Now().Add(time.Minute)}); err != nil {
		t.Fatal(err)
	}
	attempts.Create(ctx, models.LoginAttempt{State: "old", Nonce: "n", CodeVerifier: "v", RedirectPath: "/profile", ExpiresAt: time.Now().Add(-time.Minute)})
	if a, err := attempts.Consume(ctx, "s1"); err != nil || a == nil || a.Nonce != "n1" || a.CodeVerifier != "v1" {
		t.Fatalf("Consume = %+v, %v", a, err)
	}
	if a, _ := attempts.Consume(ctx, "s1"); a != nil {
		t.Error("state consumed twice")
	}
	if a, _ := attempts.Consume(ctx, "old"); a != nil {
		t.Error("expired state consumed")
	}
	if n, _ := attempts.DeleteExpired(ctx); n != 1 {
		t.Errorf("DeleteExpired = %d, want 1", n)
	}
}

func TestAuthRepos_RememberMeAndRevokeAll(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	users, sessions, attempts := NewUserRepo(db), NewSessionRepo(db), NewLoginAttemptRepo(db)

	if err := attempts.Create(ctx, models.LoginAttempt{State: "r1", Nonce: "n", CodeVerifier: "v", RedirectPath: "/profile", RememberMe: true, ExpiresAt: time.Now().Add(time.Minute)}); err != nil {
		t.Fatal(err)
	}
	if a, err := attempts.Consume(ctx, "r1"); err != nil || a == nil || !a.RememberMe {
		t.Fatalf("remember_me not round-tripped: %+v, %v", a, err)
	}

	id1, _ := users.Create(ctx, models.AppUser{Email: "a@mq.com"}, "")
	id2, _ := users.Create(ctx, models.AppUser{Email: "b@mq.com"}, "")
	exp := time.Now().Add(time.Hour)
	hashes := []string{strings.Repeat("c", 64), strings.Repeat("d", 64), strings.Repeat("e", 64), strings.Repeat("f", 64)}
	for i, uid := range []int64{id1, id1, id1, id2} {
		if err := sessions.Create(ctx, hashes[i], uid, models.AuthMethodPassword, exp, "", ""); err != nil {
			t.Fatal(err)
		}
	}
	sessions.Revoke(ctx, hashes[0])

	if n, err := sessions.RevokeAllForUser(ctx, id1); err != nil || n != 2 {
		t.Fatalf("RevokeAllForUser = %d, %v; want 2", n, err)
	}
	if su, _ := sessions.GetValid(ctx, hashes[3]); su == nil || su.ID != id2 {
		t.Error("another user's session was revoked")
	}
}

func TestRBACRepos(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	users, sessions, roles := NewUserRepo(db), NewSessionRepo(db), NewRoleRepo(db)

	list, err := roles.List(ctx)
	if err != nil || len(list) < 3 {
		t.Fatalf("roles = %+v, %v (is migration 004 applied?)", list, err)
	}
	perms := map[string][]string{}
	for _, r := range list {
		perms[r.Code] = r.Permissions
	}
	// 004 seeds four permissions; 005 adds orders.detail to every seeded role.
	if len(perms["admin"]) != 5 || len(perms["operator"]) != 3 || len(perms["viewer"]) != 2 {
		t.Errorf("seeded permissions = %v (are migrations 004 and 005 applied?)", perms)
	}

	adminID, err := users.Create(ctx, models.AppUser{Email: "boss@mq.com"}, "admin")
	if err != nil {
		t.Fatal(err)
	}
	viewerID, _ := users.Create(ctx, models.AppUser{Email: "v@mq.com"}, "")
	if _, err := users.Create(ctx, models.AppUser{Email: "x@mq.com"}, "superuser"); err != ErrUnknownRole {
		t.Errorf("unknown role on create: %v", err)
	}

	// Session carries role + permissions.
	hash := strings.Repeat("a", 64)
	sessions.Create(ctx, hash, adminID, models.AuthMethodPassword, time.Now().Add(time.Hour), "", "")
	su, err := sessions.GetValid(ctx, hash)
	if err != nil || su == nil || su.RoleCode != "admin" || !su.Can(models.PermUsersManage) {
		t.Fatalf("GetValid = %+v, %v", su, err)
	}

	// Managed list / update.
	all, err := users.List(ctx)
	if err != nil || len(all) != 2 || all[0].Role.Code == "" {
		t.Fatalf("List = %+v, %v", all, err)
	}
	role := "operator"
	if err := users.Update(ctx, viewerID, UserUpdate{RoleCode: &role}); err != nil {
		t.Fatal(err)
	}
	bad := "superuser"
	if err := users.Update(ctx, viewerID, UserUpdate{RoleCode: &bad}); err != ErrUnknownRole {
		t.Errorf("unknown role on update: %v", err)
	}
	if err := users.Update(ctx, 999999, UserUpdate{RoleCode: &role}); err != sql.ErrNoRows {
		t.Errorf("missing user: %v", err)
	}
	if m, _ := users.GetManaged(ctx, viewerID); m == nil || m.Role.Code != "operator" {
		t.Errorf("after update: %+v", m)
	}

	if n, err := users.CountActiveWithPermission(ctx, models.PermUsersManage, 0); err != nil || n != 1 {
		t.Errorf("managers = %d, %v", n, err)
	}
	if n, _ := users.CountActiveWithPermission(ctx, models.PermUsersManage, adminID); n != 0 {
		t.Errorf("managers excluding the admin = %d", n)
	}

	// Lockout restarts after expiry.
	mustExec(t, db, `UPDATE public.app_user SET failed_login_count = 5, locked_until = NOW() - interval '1 minute' WHERE id = $1`, viewerID)
	if count, locked, err := users.RecordLoginFailure(ctx, viewerID, 5, 15*time.Minute); err != nil || count != 1 || locked != nil {
		t.Errorf("after expired lock: count=%d locked=%v err=%v; want 1, nil", count, locked, err)
	}

	// Password change keeps one session.
	keep, drop := strings.Repeat("k", 64), strings.Repeat("d", 64)
	sessions.Create(ctx, keep, viewerID, models.AuthMethodPassword, time.Now().Add(time.Hour), "", "")
	sessions.Create(ctx, drop, viewerID, models.AuthMethodPassword, time.Now().Add(time.Hour), "", "")
	if n, err := sessions.RevokeAllForUserExcept(ctx, viewerID, keep); err != nil || n != 1 {
		t.Errorf("RevokeAllForUserExcept = %d, %v", n, err)
	}
	if err := users.SetPassword(ctx, viewerID, "$2a$04$x"); err != nil {
		t.Fatal(err)
	}
}
