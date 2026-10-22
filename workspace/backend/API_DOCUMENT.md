# MQ E-commerce Integration Service API Documentation

Base URL: `http://<host>:9091/api/v1` (port from `config/application.yml` → `server`; the admin app calls it as relative `/api/v1/...` through the Vite proxy or a reverse proxy, same origin).

Go 1.24 + Gin. Routes are registered in `component/handlers/rest/rest.go` (`APIRoutes`). Most of the service's work is done by cron workers (`config/workers.yml`), not HTTP endpoints.

## Contents

- [Conventions](#conventions)
- [Auth](#auth) — `/auth/*`
- [Users and roles](#users-and-roles) — `/users`, `/roles`
- [Orders](#orders) — `/orders`
- [Product images](#product-images) — `/product-images`
- [Marketplaces](#marketplaces) — `/tiktok`, `/lazada`
- [Other](#other) — swagger

## Conventions

- **Envelope.** JSON endpoints always answer **HTTP 200** with `{code, message, error_text?, data}` (`component/api/response.go`). `code` is the real status, `message` is `"Success"` or a stable error code (e.g. `FORBIDDEN`), and `data` is **an array**: the payload is `data[0]`. Errors have no `data`; they carry `error_text`, the error in the request's language (omitted when there is no text for the code or status). Clients switch on `message`, never on `error_text`.
- **Language (EC-7).** Supported: `vi` (default) and `en`. A request's language is the signed-in user's saved `locale`; without a session, the best match in `Accept-Language`; otherwise `vi`. Responses send `Content-Language`. It translates `error_text` and role/permission names (`role.name`, `role.description`, `permission_names`); error codes and data never change. Catalogs: `i18n/locales/*.json`.
- **Session.** Login sets the httpOnly cookie `ecom_session` (`auth.cookie-name`, `SameSite=Lax`). Endpoints marked *session* use `middleware.RequireAuth`; without a valid session they return `401 UNAUTHENTICATED`.
- **Permissions.** Endpoints marked *permission X* use `middleware.RequirePermission`; a user whose role lacks X gets `403 FORBIDDEN`. Permission codes: `users.read`, `users.manage`, `orders.read`, `orders.detail`, `orders.write` (`migrations/004`, `005`).
- **Same origin.** State-changing endpoints marked *same origin* use `middleware.RequireSameOrigin`: a request with an `Origin` header other than the origin of `auth.frontend-url` gets `403 FORBIDDEN_ORIGIN` (requests without `Origin` pass; `SameSite=Lax` covers those).
- **Errors common to all JSON endpoints:** `400 INVALID_REQUEST` (body/query can't be bound or validated), `500 INTERNAL_ERROR`. Error responses never include the cause.

## Auth

### GET /auth/providers
- **Description**: Login methods currently available.
- **Auth**: public
- **Request**: none
- **Response**: 200, `data[0]` = `{google: bool, password: bool}`
- **Errors**: none

### GET /auth/google/login
- **Description**: Starts Google sign-in (OIDC with state, nonce, PKCE). Browser redirect, not JSON.
- **Auth**: public
- **Request**: query `next` (FE path to return to, sanitised), `remember=1` (long session)
- **Response**: 302 to Google
- **Errors**: 302 to `<frontend-url>/login?error=CODE` (e.g. `GOOGLE_DISABLED`)

### GET /auth/google/callback
- **Description**: Google's redirect target. Verifies the ID token, sets `ecom_session`, redirects to the FE.
- **Auth**: public
- **Request**: query `code`, `state`, or `error` (from Google)
- **Response**: 302 to `<frontend-url><next>`
- **Errors**: 302 to `<frontend-url>/login?error=CODE`: `CONSENT_DENIED`, `INVALID_STATE`, `EXCHANGE_FAILED`, `INVALID_ID_TOKEN`, `EMAIL_NOT_VERIFIED`, `DOMAIN_NOT_ALLOWED`, `USER_NOT_ALLOWED`, `INTERNAL_ERROR`

### POST /auth/login
- **Description**: Sign in with an internal email and password; sets `ecom_session`.
- **Auth**: public, same origin
- **Request**: body `{email: string (email, ≤320), password: string (≤256), remember_me?: bool}`
- **Response**: 200, `data[0]` = `UserResponse` `{id, email, display_name, avatar_url, auth_method: "password"|"google", last_login_at, role: {code, name}, permissions: string[], permission_names: {code: name}, locale: "vi"|"en"}`. Role and permission names are in the account's `locale` (the response sends `Content-Language`).
- **Errors**: `400 INVALID_REQUEST`, `401 INVALID_CREDENTIALS` (unknown email, wrong password, Google-only or disabled account), `423 ACCOUNT_LOCKED`, `403 FORBIDDEN_ORIGIN`

### GET /auth/me
- **Description**: The signed-in user.
- **Auth**: session
- **Response**: 200, `data[0]` = `UserResponse` (as above)
- **Errors**: `401 UNAUTHENTICATED`

### PUT /auth/me/locale
- **Description**: Saves the signed-in user's UI language (`user_config.locale`, a 1-1 table next to `app_user`, migration `006`). From the next request on, the backend answers in it.
- **Auth**: session, same origin
- **Request**: body `{locale: "vi" | "en"}` (exact code)
- **Response**: 200, `data[0]` = `UserResponse` already in the new language
- **Errors**: `400 INVALID_REQUEST` (missing or unsupported locale), `401 UNAUTHENTICATED`, `403 FORBIDDEN_ORIGIN`

### POST /auth/logout
- **Description**: Revokes this browser's session and clears the cookie.
- **Auth**: same origin (cookie optional)
- **Response**: 200, no data
- **Errors**: `401 UNAUTHENTICATED` (no cookie; the cookie is cleared anyway), `403 FORBIDDEN_ORIGIN`

### POST /auth/logout-all
- **Description**: Revokes every session of the signed-in user and clears this cookie.
- **Auth**: session, same origin
- **Response**: 200, `data[0]` = `{revoked_sessions: number}`
- **Errors**: `401 UNAUTHENTICATED`, `403 FORBIDDEN_ORIGIN`

### POST /auth/change-password
- **Description**: Changes your own password and signs out your other sessions.
- **Auth**: session, same origin
- **Request**: body `{current_password: string (≤256; may be empty for an account that never had a password), new_password: string (≤256)}`
- **Response**: 200, no data
- **Errors**: `400 WRONG_PASSWORD`, `400 WEAK_PASSWORD` (shorter than `auth.min-password-length`, default 10), `401 UNAUTHENTICATED`, `403 FORBIDDEN_ORIGIN`

## Users and roles

Mounted only when `UserAdminService` is wired (`server/base.go`).

### GET /users
- **Description**: All users for the user-management page.
- **Auth**: session, permission `users.read`
- **Response**: 200, `data[0]` = `{users: ManagedUser[]}`; `ManagedUser` = `{id, email, display_name, avatar_url, status: "active"|"disabled", role: {code, name}, has_password, google_linked, locked_until, last_login_at, created_at}`; `role.name` is in the request's language
- **Errors**: `401 UNAUTHENTICATED`, `403 FORBIDDEN`

### GET /roles
- **Description**: Roles and their permission codes.
- **Auth**: session, permission `users.read`
- **Response**: 200, `data[0]` = `{roles: [{code, name, description, is_system, permissions: string[], permission_names: {code: name}}]}`; names and descriptions are in the request's language (roles without a catalog entry keep the text stored in the DB)
- **Errors**: `401 UNAUTHENTICATED`, `403 FORBIDDEN`

### POST /users
- **Description**: Creates a user. Without `password` the account can only sign in with Google.
- **Auth**: session, same origin, permission `users.manage`
- **Request**: body `{email: string (email, ≤320), display_name?: string (≤255), role: string, password?: string (≤256)}`
- **Response**: 200, `data[0]` = `ManagedUser`
- **Errors**: `400 INVALID_REQUEST`, `400 INVALID_ROLE`, `400 WEAK_PASSWORD`, `409 EMAIL_TAKEN`, `401`, `403 FORBIDDEN` / `FORBIDDEN_ORIGIN`

### PATCH /users/:id
- **Description**: Changes a user's name, role or status. Disabling revokes their sessions.
- **Auth**: session, same origin, permission `users.manage`
- **Request**: path `id`; body `{display_name?: string, role?: string, status?: "active"|"disabled"}` (only present fields change)
- **Response**: 200, `data[0]` = `ManagedUser`
- **Errors**: `404 NOT_FOUND`, `400 INVALID_ROLE`, `409 CANNOT_MODIFY_SELF` (own role/status), `409 LAST_ADMIN` (would leave no active user with `users.manage`), `401`, `403`

### POST /users/:id/reset-password
- **Description**: Sets another user's password and signs them out everywhere.
- **Auth**: session, same origin, permission `users.manage`
- **Request**: path `id`; body `{password: string (≤256)}`
- **Response**: 200, no data
- **Errors**: `404 NOT_FOUND`, `400 WEAK_PASSWORD`, `409 CANNOT_MODIFY_SELF`, `401`, `403`

## Orders

Read-only views of `public.ecom_order` / `ecom_order_item` for the admin app's Orders menu (EC-4). Mounted only when `OrderService` is wired (`server/base.go`). `raw_order` and `org_id` are never returned.

### GET /orders
- **Description**: Orders, newest first (`created_at DESC, id DESC`), filtered and paginated.
- **Auth**: session, permission `orders.read`
- **Request**: query (all optional)

  | Param | Type | Notes |
  |---|---|---|
  | `q` | string, ≤100 chars | Contains match (case-insensitive) on `order_id`, `tracking_code`, `customer_name` or `phone`. `%` and `_` match literally. |
  | `market_place` | `Tiktok` \| `Lazada` \| `Shopee` | |
  | `lead_status` | 1–4 | `utils.LeadStatus`: 1 pending (out of stock), 2 created, 3 refund, 4 cancel |
  | `from`, `to` | `YYYY-MM-DD` | Creation date in Vietnam time (UTC+7), both inclusive; `from` ≤ `to` |
  | `page` | int ≥ 1 | default 1 |
  | `page_size` | 1–100 | default 20 |

- **Response**: 200, `data[0]` = `{orders: OrderSummary[], total: number, page: number, page_size: number}`; `OrderSummary` = `{id, market_place, shop_id, order_id, tracking_code, lead_status, do_status, so_status: number|null, customer_name, phone, total_amount, payment_method, item_count, created_at, updated_at}`. Response has `Cache-Control: no-store`.
- **Errors**: `400 INVALID_REQUEST` (any invalid parameter), `401 UNAUTHENTICATED`, `403 FORBIDDEN`

### GET /orders/:id
- **Description**: One order with its items.
- **Auth**: session, permission `orders.detail`
- **Request**: path `id` (the `ecom_order.id` from the list, not the marketplace `order_id`)
- **Response**: 200, `data[0]` = `{order: OrderDetail}`; `OrderDetail` = every `OrderSummary` field plus `{address, province, district, sub_district, neighborhood, warehouse, shipping_label, comment, discount, shipping_fee, items: [{id, name, sku, quantity, price}]}` (`items` is `[]` when there are none). Response has `Cache-Control: no-store`.
- **Errors**: `404 NOT_FOUND` (unknown or non-numeric id), `401 UNAUTHENTICATED`, `403 FORBIDDEN`

## Product images

Product images uploaded from the admin app straight to TikTok Shop or Lazada (EC-10). The file is **not** stored by this service: it is sent to the marketplace and only what the marketplace returns is kept in `public.ecom_product_image` (migration `007`), for later product listings. Mounted only when `ProductImageService` is wired (`server/base.go`). Every route needs a session and permission `orders.write`.

### GET /product-images/shops
- **Description**: Shops images can be uploaded to: `market_token` rows of TikTok and Lazada that are enabled and have an access token. Also lists TikTok's image use cases.
- **Auth**: session, permission `orders.write`
- **Response**: 200, `data[0]` = `{shops: [{market_place: "Tiktok"|"Lazada", shop_id}], tiktok_use_cases: ["MAIN_IMAGE", "ATTRIBUTE_IMAGE", "DESCRIPTION_IMAGE", "CERTIFICATION_IMAGE", "SIZE_CHART_IMAGE"]}`
- **Errors**: `401 UNAUTHENTICATED`, `403 FORBIDDEN`

### GET /product-images
- **Description**: Uploaded images, newest first (`created_at DESC, id DESC`).
- **Auth**: session, permission `orders.write`
- **Request**: query (all optional) `market_place` (`Tiktok` | `Lazada`), `shop_id` (≤128), `page` (≥1, default 1), `page_size` (1–100, default 24)
- **Response**: 200, `data[0]` = `{images: ProductImage[], total, page, page_size}`; `ProductImage` = `{id, market_place, shop_id, use_case (TikTok, '' for Lazada), uri (TikTok image uri, '' for Lazada), url (marketplace CDN URL), width, height, file_name, size_bytes, uploaded_by (email | null), created_at}`. Response has `Cache-Control: no-store`.
- **Errors**: `400 INVALID_REQUEST`, `401 UNAUTHENTICATED`, `403 FORBIDDEN`

### POST /product-images
- **Description**: Uploads **one** image to a shop and stores the result. TikTok: `POST /product/202309/images/upload` (multipart `data` + `use_case`, header `x-tts-access-token`). Lazada: `POST /image/upload` (multipart `image`; system params and `sign` in the query). Both use the shop's access token from `market_token`; no retry on an expired token.
- **Auth**: session, same origin, permission `orders.write`
- **Request**: `multipart/form-data`: `file` (JPG or PNG, judged by its bytes, ≤ 5 MB, must decode), `market_place` (`Tiktok` | `Lazada`), `shop_id`, `use_case` (TikTok only, default `MAIN_IMAGE`; ignored for Lazada). The whole body is limited to 6 MB.
- **Response**: 200, `data[0]` = `ProductImage` (width/height from TikTok, or read from the file for Lazada). Response has `Cache-Control: no-store`.
- **Errors**: `400 INVALID_IMAGE` (not JPG/PNG, unreadable, empty, over 5 MB — nothing is sent to the marketplace), `400 INVALID_REQUEST` (no file, unknown marketplace or use case), `404 NOT_FOUND` (unknown or disabled shop, or a shop without a token), `502 MARKETPLACE_ERROR` (the marketplace refused or didn't answer; its code and message are logged, nothing is stored), `401`, `403 FORBIDDEN` / `FORBIDDEN_ORIGIN`

## Marketplaces

### /tiktok, /lazada
- **Description**: Route groups built with `requireAuth` (`TikTokHandler.Router`, `LazadaHandler.Router`), but neither registers any endpoint yet. The marketplace work runs in cron workers.

### POST /shopee/webhook
- **Description**: `ShopeeHandler.ShopeeWebhook` exists but is **not mounted** in `APIRoutes`, so it is not reachable.

## Other

### GET /swagger/*any
- **Description**: Swagger UI for the annotated handlers (`docs/`, regenerate with `swag init`). Served at `/swagger/index.html`, outside `/api/v1`.
- **Auth**: public
