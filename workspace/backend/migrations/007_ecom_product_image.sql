-- Product images uploaded to the marketplaces from the admin app (EC-10).
-- Product listings on TikTok Shop and Lazada must point at images that were
-- uploaded to that marketplace first. The file itself is NOT kept by us: we
-- send it to the marketplace and store what it returns (url, and TikTok's
-- uri, which TikTok's product APIs reference), so the product-creation step
-- can use it later.

CREATE TABLE IF NOT EXISTS public.ecom_product_image (
    id           BIGSERIAL    PRIMARY KEY,
    market_place VARCHAR(32)  NOT NULL,              -- utils.Plaform: Tiktok | Lazada
    shop_id      VARCHAR(128) NOT NULL,              -- market_token.shop_id
    use_case     VARCHAR(32)  NOT NULL DEFAULT '',   -- TikTok: MAIN_IMAGE, ATTRIBUTE_IMAGE, DESCRIPTION_IMAGE, ...; '' for Lazada
    uri          VARCHAR(512) NOT NULL DEFAULT '',   -- TikTok image uri; '' for Lazada
    url          TEXT         NOT NULL,              -- image URL on the marketplace CDN
    width        INT          NOT NULL DEFAULT 0,
    height       INT          NOT NULL DEFAULT 0,
    file_name    VARCHAR(255) NOT NULL DEFAULT '',   -- original name, for people to recognise it
    size_bytes   INT          NOT NULL DEFAULT 0,
    uploaded_by  BIGINT       REFERENCES public.app_user (id) ON DELETE SET NULL,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT ck_ecom_product_image_market_place CHECK (market_place IN ('Tiktok', 'Lazada'))
);

CREATE INDEX IF NOT EXISTS ix_ecom_product_image_shop
    ON public.ecom_product_image (market_place, shop_id, created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS ix_ecom_product_image_created_at
    ON public.ecom_product_image (created_at DESC, id DESC);
