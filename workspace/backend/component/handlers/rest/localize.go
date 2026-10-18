package rest

import (
	"main/component/models"
	"main/i18n"

	"github.com/gin-gonic/gin"
)

// Responses carry role and permission names in the request's language (EC-7).
// The DB keeps the Vietnamese text as the fallback for roles that have no
// catalog entry (i18n/locales/*.json).

func userResponse(c *gin.Context, u models.SessionUser) models.UserResponse {
	locale := i18n.Locale(c)
	resp := models.NewUserResponse(u)
	resp.Role.Name = i18n.RoleName(locale, resp.Role.Code, resp.Role.Name)
	resp.PermissionNames = i18n.PermissionNames(locale, resp.Permissions)
	return resp
}

func localizeRoles(c *gin.Context, roles []models.Role) []models.Role {
	locale := i18n.Locale(c)
	for i := range roles {
		r := &roles[i]
		r.Name = i18n.RoleName(locale, r.Code, r.Name)
		r.Description = i18n.RoleDescription(locale, r.Code, r.Description)
		r.PermissionNames = i18n.PermissionNames(locale, r.Permissions)
	}
	return roles
}

func localizeManagedUser(c *gin.Context, u *models.ManagedUser) *models.ManagedUser {
	if u != nil {
		u.Role.Name = i18n.RoleName(i18n.Locale(c), u.Role.Code, u.Role.Name)
	}
	return u
}

func localizeManagedUsers(c *gin.Context, users []models.ManagedUser) []models.ManagedUser {
	for i := range users {
		localizeManagedUser(c, &users[i])
	}
	return users
}
