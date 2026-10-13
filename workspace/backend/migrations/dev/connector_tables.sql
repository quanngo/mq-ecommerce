-- LOCAL DEVELOPMENT ONLY. Never run this against a shared database.
--
-- Minimal stand-ins for the Connector (formerly TMS) system's public.od_*
-- tables, which this service reads and partly updates (ConnectorRepo) but
-- does not own. The real tables have more columns and are managed by the
-- Connector system. Only the columns this code touches are defined here,
-- and their types are inferred from component/repositories/connector_repo.go
-- and component/models/od_ecom_order_model.go.
--
-- migrate.sh only applies migrations/*.sql, so this subdirectory is never
-- picked up. Load it by hand into a local database:
--   psql -h 127.0.0.1 -U postgres -d <local_db> -f migrations/dev/connector_tables.sql

-- Orders as the Connector stores them. ConnectorRepo reads the identity and
-- address columns, and UpdateOrderPayment writes the payment columns
-- (TikTok sends those amounts as decimal strings).
CREATE TABLE IF NOT EXISTS public.od_ecom_order (
    id                             SERIAL       PRIMARY KEY,
    market_place                   VARCHAR(32)  NOT NULL,
    order_id                       VARCHAR(128) NOT NULL,
    lead_id                        INT          NOT NULL DEFAULT 0,
    tracking_code                  VARCHAR(128) NOT NULL DEFAULT '',
    customer_name                  VARCHAR(255) NOT NULL DEFAULT '',
    phone                          VARCHAR(64)  NOT NULL DEFAULT '',
    address                        TEXT         NOT NULL DEFAULT '',
    province                       VARCHAR(255) NOT NULL DEFAULT '',
    district                       VARCHAR(255) NOT NULL DEFAULT '',
    sub_district                   VARCHAR(255) NOT NULL DEFAULT '',
    payment_method                 VARCHAR(32)  NOT NULL DEFAULT '',
    discount                       NUMERIC(18,2) NOT NULL DEFAULT 0,
    created_date                   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    shipping_label                 TEXT         NOT NULL DEFAULT '',
    warehouse                      VARCHAR(255) NOT NULL DEFAULT '',
    shop_id                        VARCHAR(128) NOT NULL DEFAULT '',
    org_id                         INT          NOT NULL DEFAULT 0,
    currency                       VARCHAR(8),
    original_shipping_fee          NUMERIC(18,2),
    original_total_product_price   NUMERIC(18,2),
    platform_discount              NUMERIC(18,2),
    seller_discount                NUMERIC(18,2),
    shipping_fee                   NUMERIC(18,2),
    shipping_fee_cofunded_discount NUMERIC(18,2),
    shipping_fee_seller_discount   NUMERIC(18,2),
    sub_total                      NUMERIC(18,2),
    tax                            NUMERIC(18,2),
    total_amount                   NUMERIC(18,2)
);

-- Delivery orders. Joined on tracking_code by GetOdEcomOrderDObyStatus;
-- status is utils.DeliveryOrderStatus (51 new, 52 ready to pick, ...).
CREATE TABLE IF NOT EXISTS public.od_do_new (
    id            SERIAL       PRIMARY KEY,
    tracking_code VARCHAR(128) NOT NULL,
    status        INT          NOT NULL
);

-- Sale orders. Joined on lead_id by GetOrderIdForStatementTransaction;
-- status is utils.SaleOrderStatus (49 = completed).
CREATE TABLE IF NOT EXISTS public.od_sale_order (
    id      SERIAL PRIMARY KEY,
    lead_id INT    NOT NULL,
    status  INT    NOT NULL
);
