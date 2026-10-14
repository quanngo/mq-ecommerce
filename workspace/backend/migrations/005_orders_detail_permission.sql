-- Orders menu (EC-4): the order list needs orders.read (seeded by 004); the
-- order detail page needs the new orders.detail. Requested for every seeded
-- role, so viewers can open details too; take it away from a role with
--   DELETE FROM public.role_permission WHERE permission_code = 'orders.detail' AND role_id = (...);
-- Code checks models.PermOrdersDetail; keep the code in sync with that constant.

INSERT INTO public.permission (code, description) VALUES
    ('orders.detail', 'Xem chi tiết đơn hàng')
ON CONFLICT (code) DO NOTHING;

INSERT INTO public.role_permission (role_id, permission_code)
SELECT r.id, 'orders.detail'
FROM public.role r
WHERE r.code IN ('admin', 'operator', 'viewer')
ON CONFLICT DO NOTHING;

-- The order list is sorted newest first and filtered by creation date.
CREATE INDEX IF NOT EXISTS ix_ecom_order_created_at
    ON public.ecom_order (created_at DESC, id DESC);
