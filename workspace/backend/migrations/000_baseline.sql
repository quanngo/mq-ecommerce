-- Baseline for the tables this service owns, all in the public schema.
--
-- Step 1 moves the tables that existing databases still have in the old
-- `ecommerce` schema into `public`. ALTER TABLE ... SET SCHEMA keeps the
-- data, indexes, constraints, grants and owned sequences, and it only takes
-- a brief lock. Code from before this change queries ecommerce.* and stops
-- working once this has run, so deploy the new code at the same time and
-- check that no other system (reports, Connector jobs) reads ecommerce.*.
-- If a table exists in both schemas, the migration aborts and nothing is
-- moved. Merge the two copies by hand first.
-- The empty `ecommerce` schema is left in place on purpose.
--
-- Step 2 creates the tables on a fresh database. It was reconstructed from
-- the queries in component/repositories/ and the structs in
-- component/models/, not dumped from a live database, so column types and
-- sizes are inferred. Compare with `pg_dump --schema-only` of the real
-- database before relying on it. Every statement is CREATE ... IF NOT EXISTS
-- with constraints declared inline, so it changes nothing on a database
-- that already has the tables.
--
-- Constraints the code depends on:
--   * market_token and market_place_mapping are read with SELECT * into
--     models.MarketToken / models.MarketPlaceMapping, so they must have
--     exactly these columns. An extra column fails the scan
--     ("missing destination name").
--   * Non-pointer Go fields (string, int, time.Time, bool) need NOT NULL.
--     A NULL in them fails the scan.
--   * ecom_statement_transaction needs UNIQUE (market_place, order_id) for
--     the ON CONFLICT in ConnectorRepo.InsertEcomStatementTransaction.

-- Step 1: ecommerce.* -> public.*
DO $$
DECLARE
    t text;
BEGIN
    FOREACH t IN ARRAY ARRAY['market_token', 'market_place_mapping', 'log_market_place_order',
                             'log_order_status', 'ecom_statement_transaction'] LOOP
        IF to_regclass(format('ecommerce.%I', t)) IS NOT NULL THEN
            IF to_regclass(format('public.%I', t)) IS NOT NULL THEN
                RAISE EXCEPTION 'both ecommerce.% and public.% exist; merge them by hand, then re-run', t, t;
            END IF;
            EXECUTE format('ALTER TABLE ecommerce.%I SET SCHEMA public', t);
            RAISE NOTICE 'moved ecommerce.% to public.%', t, t;
        END IF;
    END LOOP;
END $$;

-- Step 2: create anything still missing (fresh database)

-- One row per connected shop. Tokens are refreshed in place by
-- TokenRepo.UpdateAccessToken. models.MarketToken.
CREATE TABLE IF NOT EXISTS public.market_token (
    id             BIGSERIAL    PRIMARY KEY,
    market_place   VARCHAR(32)  NOT NULL,            -- utils.Plaform: Tiktok | Lazada | Shopee
    shop_id        VARCHAR(128) NOT NULL,
    access_token   TEXT         NOT NULL DEFAULT '',
    refresh_token  TEXT         NOT NULL DEFAULT '',
    shop_reference VARCHAR(255),                      -- TikTok shop_cipher; nullable (*string)
    expire_time    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    update_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    enable         BOOLEAN      NOT NULL DEFAULT TRUE, -- false = skipped by every scan
    CONSTRAINT ux_market_token_shop UNIQUE (market_place, shop_id)
);

-- Lookup table keyed by type:
--   product               name = shop id, ecommerce_id = platform product id,
--                         tms_id = "<product name>|<sku>"
--   warehouse             ecommerce_id = shop id, tms_id = warehouse
--   status                status mapping per market_place
--   payment configuration ecommerce_id = shop id, tms_id = last processed
--                         od_ecom_order.id (a cursor, stored as text)
-- models.MarketPlaceMapping.
CREATE TABLE IF NOT EXISTS public.market_place_mapping (
    id           BIGSERIAL    PRIMARY KEY,
    tms_id       VARCHAR(255) NOT NULL DEFAULT '',
    ecommerce_id VARCHAR(255) NOT NULL DEFAULT '',
    quantity     INT          NOT NULL DEFAULT 0,
    name         VARCHAR(255) NOT NULL DEFAULT '',
    type         VARCHAR(64)  NOT NULL,
    market_place VARCHAR(32)  NOT NULL,
    org_id       INT          NOT NULL DEFAULT 0,
    CONSTRAINT ck_market_place_mapping_type
        CHECK (type IN ('product', 'warehouse', 'status', 'payment configuration'))
);

-- Every order seen on a marketplace, written before the order is created.
-- CheckOrderExists(market_place, order_id) makes the scan idempotent.
-- status is utils.LeadStatus: 1 pending (out of stock), 2 created, 3 refund, 4 cancel.
CREATE TABLE IF NOT EXISTS public.log_market_place_order (
    id           BIGSERIAL    PRIMARY KEY,
    market_place VARCHAR(32)  NOT NULL,
    shop_id      VARCHAR(128) NOT NULL DEFAULT '',
    order_id     VARCHAR(128) NOT NULL,
    status       INT          NOT NULL,
    payload      JSONB,                               -- raw platform order
    created_date TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_date TIMESTAMPTZ,                          -- nullable (*time.Time)
    org_id       INT          NOT NULL DEFAULT 0
);

-- Last known marketplace status per order. MAX(created_date) / MAX(updated_at)
-- per shop is the cursor for the next CronjobScanOrder run.
CREATE TABLE IF NOT EXISTS public.log_order_status (
    id           BIGSERIAL    PRIMARY KEY,
    market_place VARCHAR(32)  NOT NULL,
    shop_id      VARCHAR(128) NOT NULL DEFAULT '',
    order_id     VARCHAR(128) NOT NULL,
    status       VARCHAR(64)  NOT NULL DEFAULT '',     -- platform status string
    created_date TIMESTAMPTZ  NOT NULL,
    updated_at   TIMESTAMPTZ  NOT NULL
);

-- Settlement / fee breakdown per order (TikTok finance, Lazada transaction
-- details). Upserted. models.EcomStatementTransaction.
CREATE TABLE IF NOT EXISTS public.ecom_statement_transaction (
    id           BIGSERIAL    PRIMARY KEY,
    market_place VARCHAR(32)  NOT NULL,
    order_id     VARCHAR(128) NOT NULL,
    raw_data     JSONB        NOT NULL DEFAULT '{}'::jsonb,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT ux_ecom_statement_transaction_order UNIQUE (market_place, order_id)
);
