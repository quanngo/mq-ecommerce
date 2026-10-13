-- Login for the admin app (EC-1): Google OIDC and internal email/password.
-- New tables only; nothing existing is touched. Design: .workflow/EC-1/design.md.

-- Users who can sign in. Matched to Google by verified email; google_sub is
-- pinned on the first Google login so a later email change on the Google side
-- cannot take over another account. models.AppUser.
CREATE TABLE IF NOT EXISTS public.app_user (
    id                 BIGSERIAL    PRIMARY KEY,
    email              VARCHAR(320) NOT NULL,                   -- always lower-cased by the app
    display_name       VARCHAR(255) NOT NULL DEFAULT '',
    avatar_url         TEXT,                                    -- Google `picture`; nullable (*string)
    password_hash      TEXT,                                    -- bcrypt; NULL = Google-only account
    google_sub         VARCHAR(255),                            -- Google `sub`; NULL until first Google login
    status             VARCHAR(16)  NOT NULL DEFAULT 'active',
    failed_login_count INT          NOT NULL DEFAULT 0,         -- consecutive password failures
    locked_until       TIMESTAMPTZ,                             -- password login refused until then
    last_login_at      TIMESTAMPTZ,
    created_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT ux_app_user_email      UNIQUE (email),
    CONSTRAINT ux_app_user_google_sub UNIQUE (google_sub),
    CONSTRAINT ck_app_user_status     CHECK (status IN ('active', 'disabled')),
    CONSTRAINT ck_app_user_email_lc   CHECK (email = lower(email))
);

-- Server-side sessions. The browser holds a random 32-byte token in an
-- httpOnly cookie; only its sha256 (hex) is stored here.
CREATE TABLE IF NOT EXISTS public.user_session (
    token_hash  CHAR(64)    PRIMARY KEY,
    user_id     BIGINT      NOT NULL REFERENCES public.app_user (id) ON DELETE CASCADE,
    auth_method VARCHAR(16) NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at  TIMESTAMPTZ NOT NULL,
    revoked_at  TIMESTAMPTZ,
    user_agent  TEXT        NOT NULL DEFAULT '',
    ip          VARCHAR(64) NOT NULL DEFAULT '',
    CONSTRAINT ck_user_session_method CHECK (auth_method IN ('google', 'password'))
);
CREATE INDEX IF NOT EXISTS ix_user_session_user_id    ON public.user_session (user_id);
CREATE INDEX IF NOT EXISTS ix_user_session_expires_at ON public.user_session (expires_at);

-- One row per started Google login: CSRF state, OIDC nonce, PKCE verifier.
-- Deleted when the callback consumes it (single use) or by cleanupAuthSessions.
CREATE TABLE IF NOT EXISTS public.oauth_login_attempt (
    state         VARCHAR(64)  PRIMARY KEY,
    nonce         VARCHAR(64)  NOT NULL,
    code_verifier VARCHAR(128) NOT NULL,
    redirect_path VARCHAR(512) NOT NULL DEFAULT '/profile',     -- FE path to return to (sanitised)
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    expires_at    TIMESTAMPTZ  NOT NULL
);
CREATE INDEX IF NOT EXISTS ix_oauth_login_attempt_expires_at ON public.oauth_login_attempt (expires_at);
