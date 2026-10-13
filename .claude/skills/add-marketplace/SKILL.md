---
name: add-marketplace
description: Add a new marketplace (e.g. finish Shopee, or add Shopify) to the Go backend, or a new capability to an existing one, following the existing TikTok/Lazada shape — signed API client in middleware/, per-platform models, a service implementing ScanOrder and friends, dispatch cases in platform_service.go, wiring in server/base.go, shop rows in public.market_token and product/warehouse rows in public.market_place_mapping. Use when asked to integrate/support a new sales channel or e-commerce platform, or to add order scanning/refunds/payments for one.
---

Match the shape of the TikTok and Lazada integrations exactly, so that
`platform_service.go` can dispatch to the new platform like the others. Read
the platform's official API docs for signing, token refresh and paging.
**Never guess request signatures or field names.** If the developer can't
provide docs or sandbox credentials, stop and ask.

## 1. Clarify first

Ask the developer about anything not given:
- Which flows are in scope: new orders (required), pending/out-of-stock
  retry, refunds, to-ship/AWB, payments, statement transactions. Each one is
  a separate `case` in `platform_service.go`.
- Order status mapping: which platform status creates the order (TikTok uses
  `AWAITING_COLLECTION`), and which statuses map to DO updates, cancel or
  return.
- Auth model (OAuth refresh token, static partner key) and how a shop is
  identified (`shop_id`, plus anything extra like TikTok's `shop_cipher`,
  which is stored in `market_token.shop_reference`).

## 2. Implement, layer by layer

1. **Platform constant**: add `MyShop Plaform = "MyShop"` in
   `utils/constants.go` (the type name really is spelled `Plaform`). The
   string value is what goes in the `market_place` column of every table.
2. **Config**: add a `MyShopConfig` struct under `Ecommerce` in
   `utils/config.go`. Put the keys **empty** in
   `config/application.yml.example` and the real values only in the
   gitignored `config/application.yml`.
3. **Models**: create `component/models/myshop_model/` with the API
   request/response structs. Map responses into the shared types
   `models.CreateEcommerceRequest`, `models.UpdateDatabaseOrder` and
   `models.UpdateDOStatusRequest`. Don't pass platform types upward.
4. **API client**: create `middleware/myshop_auth.go` with methods on
   `*ApiMiddleware` (`DoMyShopGet/Post`, request signing, `RefreshMyShopAccessToken`,
   typed calls like `MyShopGetOrderList`). **Don't copy these existing bugs**:
   - The 401 / `IllegalAccessToken` retry in `DoTiktokPost/GET`,
     `DoLazadaPost/Get` and `DoConnectorPOST` recurses with the *old* token
     and no retry limit (`RefreshLazadaAccessToken` is a no-op). Retry **once**,
     with the token re-read from `TokenRepo` after the refresh.
   - Use an `http.Client` with a `Timeout`, not `http.DefaultClient`.
   - Return errors. Never `log.Fatal` (it kills the server), never `panic`,
     never use an unchecked type assertion on a response map.
   - Read the response body once. `DoLazadaPost` decodes and then
     `io.ReadAll`s the same body, so it always returns empty bytes.
   - Never log access tokens, refresh tokens or secrets.
5. **Service**: create `component/services/myshop_service.go` with a
   `MyShopService` interface and a `MyShopServiceImpl`. Constructor arguments
   follow `NewTikTokService`: `cfg, mw, connectorRepo, tokenRepo,
   marketPlaceMappingRepo, logMarketPlaceOrderRepo, connectorService,
   logisticSvc`. Implement at least `ScanOrder` with the **same signature**
   as `TikTokService.ScanOrder`, plus whichever other flows are in scope,
   with the same signatures as their TikTok/Lazada counterparts.
   - Before creating an order, check
     `logMarketPlaceOrderRepo.CheckOrderExists(utils.MyShop, orderID)`.
     `platform_service` writes the log row and then calls
     `connectorService.CreateLead`, which writes `public.ecom_order`.
   - Product mapping comes in as `mapProducts`, keyed by the platform's
     product/SKU id (`market_place_mapping.ecommerce_id`).
   - Don't copy `CheckExpiredTokenTiktok`. Its condition is true for almost
     every non-expired token, and it passes an empty refresh token. Refresh
     only when `expire_time` is close to now, using the stored refresh token.
6. **Dispatch**: in `component/services/platform_service.go`, add a
   `case utils.MyShop:` to every `switch shop.MarketPlace` for the flows in
   scope (`platform`, `platformPending`, `platformRefund`, `platformToShip`,
   `platformPayment`, `platformStatementTransaction`). For flows out of scope,
   return `nil` values explicitly like Lazada's refund case does, so the
   `default` branch doesn't log "unsupported market place" on every run.
   Add the service to `PlatformServiceImpl` and `NewPlatformService`.
7. **Wiring**: construct the service in **every** init function in
   `server/base.go` that needs it (`Initialize`, `InitializeWorkers`,
   `InitializeJob`). If it has a token-refresh job, register it as described
   in the `add-worker` skill.
8. **Webhooks (optional)**: if the platform pushes events, add a handler in
   `component/handlers/rest/` and mount it in `rest.APIRoutes` under
   `/api/v1/myshop`. `ShopeeHandler` exists but is **not mounted**, and no
   `ShopeeService` is constructed. Verify the webhook signature before trusting
   the body. Responses go through `api.Ok` / `api.ErrorWithMessage`, which
   always return HTTP 200 with the status in the body.

## 3. Data the developer must add (it can't come from code)

- **`public.market_token`**: one row per shop (`market_place`, `shop_id`,
  `access_token`, `refresh_token`, `shop_reference`, `expire_time`,
  `enable`). `TokenRepo.GetAll` does `SELECT *` into `models.MarketToken`, so
  **adding a column to this table without adding the field breaks every
  cron job**.
- **`public.market_place_mapping`**: product rows (`type = 'product'`,
  `name` = shop id, `ecommerce_id` = platform product id, `tms_id` =
  `"<name>|<sku>"`), plus one `type = 'warehouse'` row per shop
  (`GetWarehouseByShopID`). `scanShopOrder` skips a
  shop that has no products or no warehouse, and only logs it.

Use the `db-migrate` skill for any schema change. Write row inserts as a
migration only if the developer wants them versioned. Credentials never go
in a migration.

## 4. Verify

- Unit-test the signing function against an example from the platform docs,
  and test the status mapping with a fake store, the same way
  `component/services/connector_service_test.go` does.
- Run the `backend-check` skill.
- Run against a sandbox shop with only the new platform's worker enabled,
  and confirm rows appear in `public.log_market_place_order` and
  `public.ecom_order`.
