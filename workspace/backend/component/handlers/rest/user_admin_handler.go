package rest

import (
	"main/component/api"
	"main/component/models"
	"main/component/services"
	"main/middleware"
	"strconv"

	"github.com/gin-gonic/gin"
)

// UserAdminHandler serves /api/v1/users and /api/v1/roles (the user-management
// page). Every route needs a session; reading needs users.read, changes need
// users.manage and must come from the frontend's origin.
type UserAdminHandler struct {
	Svc services.UserAdminService
}

func (h *UserAdminHandler) Router(api *gin.RouterGroup, requireAuth, sameOrigin gin.HandlerFunc) {
	read := middleware.RequirePermission(models.PermUsersRead)
	manage := middleware.RequirePermission(models.PermUsersManage)

	users := api.Group("/users", requireAuth)
	users.GET("", read, h.List)
	users.POST("", sameOrigin, manage, h.Create)
	users.PATCH("/:id", sameOrigin, manage, h.Update)
	users.POST("/:id/reset-password", sameOrigin, manage, h.ResetPassword)

	api.GET("/roles", requireAuth, read, h.Roles)
}

// List godoc
// @Summary  All users (user management)
// @Tags     users
// @Produce  json
// @Success  200 {object} models.UsersListResponse
// @Router   /v1/users [get]
func (h *UserAdminHandler) List(c *gin.Context) {
	users, err := h.Svc.ListUsers(c.Request.Context())
	if err != nil {
		respondAuthError(c, err)
		return
	}
	noStore(c)
	api.Ok(c, models.UsersListResponse{Users: localizeManagedUsers(c, users)})
}

// Roles godoc
// @Summary  Roles and their permissions
// @Tags     users
// @Produce  json
// @Success  200 {object} models.RolesListResponse
// @Router   /v1/roles [get]
func (h *UserAdminHandler) Roles(c *gin.Context) {
	roles, err := h.Svc.ListRoles(c.Request.Context())
	if err != nil {
		respondAuthError(c, err)
		return
	}
	api.Ok(c, models.RolesListResponse{Roles: localizeRoles(c, roles)})
}

// Create godoc
// @Summary  Create a user (password optional: without it the account is Google-only)
// @Tags     users
// @Accept   json
// @Produce  json
// @Param    body body models.CreateUserRequest true "new user"
// @Success  200 {object} models.ManagedUser
// @Router   /v1/users [post]
func (h *UserAdminHandler) Create(c *gin.Context) {
	var req models.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondAuthError(c, services.ErrInvalidRequest) // never echo the body (it may hold a password)
		return
	}
	u, err := h.Svc.CreateUser(c.Request.Context(), middleware.CurrentUser(c), req)
	if err != nil {
		respondAuthError(c, err)
		return
	}
	api.Ok(c, localizeManagedUser(c, u))
}

// Update godoc
// @Summary  Change a user's name, role or status (active/disabled)
// @Tags     users
// @Accept   json
// @Produce  json
// @Param    id   path int true "user id"
// @Param    body body models.UpdateUserRequest true "fields to change"
// @Success  200 {object} models.ManagedUser
// @Router   /v1/users/{id} [patch]
func (h *UserAdminHandler) Update(c *gin.Context) {
	id, ok := userID(c)
	if !ok {
		return
	}
	var req models.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondAuthError(c, services.ErrInvalidRequest)
		return
	}
	u, err := h.Svc.UpdateUser(c.Request.Context(), middleware.CurrentUser(c), id, req)
	if err != nil {
		respondAuthError(c, err)
		return
	}
	api.Ok(c, localizeManagedUser(c, u))
}

// ResetPassword godoc
// @Summary  Set a new password for another user and sign them out everywhere
// @Tags     users
// @Accept   json
// @Produce  json
// @Param    id   path int true "user id"
// @Param    body body models.ResetPasswordRequest true "new password"
// @Success  200
// @Router   /v1/users/{id}/reset-password [post]
func (h *UserAdminHandler) ResetPassword(c *gin.Context) {
	id, ok := userID(c)
	if !ok {
		return
	}
	var req models.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondAuthError(c, services.ErrInvalidRequest)
		return
	}
	if err := h.Svc.ResetPassword(c.Request.Context(), middleware.CurrentUser(c), id, req.Password); err != nil {
		respondAuthError(c, err)
		return
	}
	api.Ok(c)
}

func userID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		respondAuthError(c, services.ErrNotFound)
		return 0, false
	}
	return id, true
}
