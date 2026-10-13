-- "Ghi nhớ đăng nhập" (remember me) for Google sign-in: the choice made on the
-- login page has to survive the round trip through Google, so it is stored on
-- the login attempt. Password login gets it straight from the request body.
-- Session length itself is config (auth.session-ttl-hours / auth.remember-me-days).
ALTER TABLE public.oauth_login_attempt
    ADD COLUMN IF NOT EXISTS remember_me BOOLEAN NOT NULL DEFAULT FALSE;
