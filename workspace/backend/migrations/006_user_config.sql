-- Multi-language admin app (EC-7): per-user settings live in public.user_config,
-- one row per app_user (1-1, user_id is both PK and FK), starting with the
-- preferred UI language so it follows the user across browsers and devices.
-- The FE switches it with PUT /api/v1/auth/me/locale; the backend uses it to
-- translate error_text and role/permission names (package i18n).
-- Supported: vi (default), en. A user without a row reads as 'vi'.
-- To add a language: add workspace/backend/i18n/locales/<code>.json and the
-- FE catalog, then widen ck_user_config_locale in a new migration.

CREATE TABLE IF NOT EXISTS public.user_config (
    user_id    BIGINT      PRIMARY KEY REFERENCES public.app_user (id) ON DELETE CASCADE,
    locale     VARCHAR(8)  NOT NULL DEFAULT 'vi',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ck_user_config_locale CHECK (locale IN ('vi', 'en'))
);

-- Existing users get the default settings.
INSERT INTO public.user_config (user_id)
SELECT id FROM public.app_user
ON CONFLICT (user_id) DO NOTHING;
