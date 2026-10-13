-- Orders created by this service (replaces POST {connector.host}api/v1/ecommerce)
-- and their delivery-status updates (replaces the Connector ECOMMERCE callback).
-- Run once against the main database:
--   psql "$DATABASE_URL" -f migrations/001_create_ecom_order.sql

CREATE TABLE IF NOT EXISTS public.ecom_order (
    id             BIGSERIAL PRIMARY KEY,
    org_id         INT          NOT NULL,
    market_place   VARCHAR(32)  NOT NULL,
    shop_id        VARCHAR(128) NOT NULL DEFAULT '',
    order_id       VARCHAR(128) NOT NULL DEFAULT '',
    tracking_code  VARCHAR(128) NOT NULL DEFAULT '',
    lead_status    INT          NOT NULL,          -- utils.LeadStatus
    do_status      VARCHAR(64)  NOT NULL DEFAULT '', -- latest DO status from the marketplace
    so_status      INT,                             -- utils.SaleOrderStatus
    customer_name  VARCHAR(255) NOT NULL DEFAULT '',
    phone          VARCHAR(64)  NOT NULL DEFAULT '',
    address        TEXT         NOT NULL DEFAULT '',
    province       VARCHAR(255) NOT NULL DEFAULT '',
    district       VARCHAR(255) NOT NULL DEFAULT '',
    sub_district   VARCHAR(255) NOT NULL DEFAULT '',
    neighborhood   VARCHAR(255) NOT NULL DEFAULT '',
    shipping_label TEXT         NOT NULL DEFAULT '',
    warehouse      VARCHAR(255) NOT NULL DEFAULT '',
    comment        TEXT         NOT NULL DEFAULT '',
    total_amount   NUMERIC(18,2) NOT NULL DEFAULT 0,
    discount       NUMERIC(18,2) NOT NULL DEFAULT 0,
    shipping_fee   NUMERIC(18,2) NOT NULL DEFAULT 0,
    payment_method INT          NOT NULL DEFAULT 0,  -- utils.PAYMENT_METHOD_*
    enable_address BOOLEAN      NOT NULL DEFAULT FALSE,
    raw_order      JSONB,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Sheet-imported orders may have no order_id, so uniqueness only applies when it is set.
CREATE UNIQUE INDEX IF NOT EXISTS ux_ecom_order_market_order
    ON public.ecom_order (market_place, order_id) WHERE order_id <> '';
CREATE INDEX IF NOT EXISTS ix_ecom_order_tracking_code
    ON public.ecom_order (tracking_code) WHERE tracking_code <> '';

CREATE TABLE IF NOT EXISTS public.ecom_order_item (
    id            BIGSERIAL PRIMARY KEY,
    ecom_order_id BIGINT       NOT NULL REFERENCES public.ecom_order (id) ON DELETE CASCADE,
    name          VARCHAR(255) NOT NULL DEFAULT '',
    sku           VARCHAR(128) NOT NULL DEFAULT '',
    quantity      INT          NOT NULL DEFAULT 0,
    price         NUMERIC(18,2) NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS ix_ecom_order_item_order
    ON public.ecom_order_item (ecom_order_id);
