package rest

import (
	"main/component/api"
	"main/component/models"
	"main/component/services"
	"main/middleware"
	"strconv"

	"github.com/gin-gonic/gin"
)

// OrderHandler serves /api/v1/orders (the Orders menu, EC-4). Read-only; the
// group is built with requireAuth. The list needs orders.read, the detail
// needs orders.detail.
type OrderHandler struct {
	Svc services.OrderService
}

func (h *OrderHandler) Router(r *gin.RouterGroup) {
	r.GET("", middleware.RequirePermission(models.PermOrdersRead), h.List)
	r.GET("/:id", middleware.RequirePermission(models.PermOrdersDetail), h.Get)
}

// List godoc
// @Summary  Orders, newest first, filtered and paginated
// @Tags     orders
// @Produce  json
// @Param    q            query string false "order_id, tracking_code, customer name or phone (contains)"
// @Param    market_place query string false "Tiktok | Lazada | Shopee"
// @Param    lead_status  query int    false "1-4 (utils.LeadStatus)"
// @Param    from         query string false "created from, YYYY-MM-DD (Vietnam time, inclusive)"
// @Param    to           query string false "created to, YYYY-MM-DD (Vietnam time, inclusive)"
// @Param    page         query int    false "default 1"
// @Param    page_size    query int    false "1-100, default 20"
// @Success  200 {object} models.OrderListResponse
// @Router   /v1/orders [get]
func (h *OrderHandler) List(c *gin.Context) {
	var q models.OrderListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		respondAuthError(c, services.ErrInvalidRequest)
		return
	}
	res, err := h.Svc.ListOrders(c.Request.Context(), q)
	if err != nil {
		respondAuthError(c, err)
		return
	}
	noStore(c)
	api.Ok(c, res)
}

// Get godoc
// @Summary  One order with its items
// @Tags     orders
// @Produce  json
// @Param    id path int true "ecom_order id"
// @Success  200 {object} models.OrderDetailResponse
// @Router   /v1/orders/{id} [get]
func (h *OrderHandler) Get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		respondAuthError(c, services.ErrNotFound)
		return
	}
	order, err := h.Svc.GetOrder(c.Request.Context(), id)
	if err != nil {
		respondAuthError(c, err)
		return
	}
	noStore(c)
	api.Ok(c, models.OrderDetailResponse{Order: *order})
}
