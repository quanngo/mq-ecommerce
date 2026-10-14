-- Roles & permissions for the admin app, plus the default admin account.
--
-- Model: one role per user (app_user.role_id). A role grants a set of
-- permission codes (role_permission). Code checks permissions, never role
-- names, so roles can be re-shaped in the DB without a deploy. The seeded
-- roles are is_system = TRUE.
--
-- !!! DEFAULT ADMIN: admin@mq.com / 123456 (requested for every environment).
-- !!! Change it right after the first login (Profile -> Đổi mật khẩu). The
-- !!! backend logs a warning on every start while the default is in use.

CREATE TABLE IF NOT EXISTS public.role (
    id          BIGSERIAL    PRIMARY KEY,
    code        VARCHAR(64)  NOT NULL,
    name        VARCHAR(128) NOT NULL,
    description TEXT         NOT NULL DEFAULT '',
    is_system   BOOLEAN      NOT NULL DEFAULT FALSE,   -- seeded; don't delete
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT ux_role_code UNIQUE (code)
);

CREATE TABLE IF NOT EXISTS public.permission (
    code        VARCHAR(64) PRIMARY KEY,                -- "<area>.<action>", e.g. users.manage
    description TEXT        NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS public.role_permission (
    role_id         BIGINT      NOT NULL REFERENCES public.role (id) ON DELETE CASCADE,
    permission_code VARCHAR(64) NOT NULL REFERENCES public.permission (code) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_code)
);

INSERT INTO public.permission (code, description) VALUES
    ('users.read',   'Xem danh sách người dùng'),
    ('users.manage', 'Tạo người dùng, đổi role, khoá/mở khoá, đặt lại mật khẩu'),
    ('orders.read',  'Xem đơn hàng'),
    ('orders.write', 'Cập nhật đơn hàng')
ON CONFLICT (code) DO NOTHING;

INSERT INTO public.role (code, name, description, is_system) VALUES
    ('admin',    'Quản trị viên', 'Toàn quyền, gồm quản lý người dùng', TRUE),
    ('operator', 'Vận hành',      'Xem và xử lý đơn hàng',              TRUE),
    ('viewer',   'Chỉ xem',       'Chỉ xem dữ liệu',                    TRUE)
ON CONFLICT (code) DO NOTHING;

INSERT INTO public.role_permission (role_id, permission_code)
SELECT r.id, p.code
FROM public.role r
JOIN (VALUES
    ('admin',    'users.read'),
    ('admin',    'users.manage'),
    ('admin',    'orders.read'),
    ('admin',    'orders.write'),
    ('operator', 'orders.read'),
    ('operator', 'orders.write'),
    ('viewer',   'orders.read')
) AS p (role_code, code) ON p.role_code = r.code
ON CONFLICT DO NOTHING;

-- Every user has exactly one role. Users created before this migration get
-- the least privileged role; promote them in the UI or with SQL.
ALTER TABLE public.app_user ADD COLUMN IF NOT EXISTS role_id BIGINT REFERENCES public.role (id);
UPDATE public.app_user SET role_id = (SELECT id FROM public.role WHERE code = 'viewer') WHERE role_id IS NULL;
ALTER TABLE public.app_user ALTER COLUMN role_id SET NOT NULL;
CREATE INDEX IF NOT EXISTS ix_app_user_role_id ON public.app_user (role_id);

-- Default admin: admin@mq.com / 123456 (bcrypt, cost 12). Not touched if the
-- account already exists, so a changed password is never reset by re-running.
INSERT INTO public.app_user (email, display_name, password_hash, status, role_id)
SELECT 'admin@mq.com', 'Administrator',
       '$2a$12$CLCrb/SA2eaOO4WBU9AJxOPnaYAUgIY43plRnfZv/kDXLUzbm7ZQK',
       'active', r.id
FROM public.role r
WHERE r.code = 'admin'
ON CONFLICT (email) DO NOTHING;
