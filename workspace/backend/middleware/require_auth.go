package middleware

import (
	"context"
	"main/component/api"
	"main/component/models"
	"main/i18n"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

// Authenticator is the slice of services.AuthService this package needs.
// services imports middleware, so middleware can't import services.
type Authenticator interface {
	Authenticate(ctx context.Context, rawToken string) (*models.SessionUser, error)
}

const ContextUserKey = "auth.user"

// RequireAuth rejects requests without a valid session cookie with
// {code: 401, message: "UNAUTHENTICATED"} and puts the user in the context.
func RequireAuth(auth Authenticator, cookieName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, _ := c.Cookie(cookieName)
		user, err := auth.Authenticate(c.Request.Context(), token)
		if err != nil || user == nil {
			api.ErrorWithMessage(c, http.StatusUnauthorized, "UNAUTHENTICATED")
			c.Abort()
			return
		}
		c.Set(ContextUserKey, user)
		// The account's saved language beats Accept-Language (see Locale).
		i18n.Set(c, user.Locale)
		c.Next()
	}
}

// Locale picks the request's language from Accept-Language (i18n.Default when
// nothing supported matches) for everything that runs after it, including
// error_text in the response envelope. Mount it on the whole API; RequireAuth
// later switches to the signed-in user's saved language.
func Locale() gin.HandlerFunc {
	return func(c *gin.Context) {
		i18n.Set(c, i18n.FromAcceptLanguage(c.GetHeader("Accept-Language")))
		c.Next()
	}
}

// RequirePermission must run after RequireAuth. It rejects users whose role
// lacks the permission with {code: 403, message: "FORBIDDEN"}. Check
// permissions, never role names, so roles can change without a deploy.
func RequirePermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !CurrentUser(c).Can(permission) {
			api.ErrorWithMessage(c, http.StatusForbidden, "FORBIDDEN")
			c.Abort()
			return
		}
		c.Next()
	}
}

// CurrentUser returns the user set by RequireAuth, or nil.
func CurrentUser(c *gin.Context) *models.SessionUser {
	v, ok := c.Get(ContextUserKey)
	if !ok {
		return nil
	}
	u, _ := v.(*models.SessionUser)
	return u
}

// RequireSameOrigin rejects state-changing requests whose Origin header is
// present and is not the frontend's origin. Defence in depth on top of
// SameSite=Lax against CSRF.
func RequireSameOrigin(frontendURL string) gin.HandlerFunc {
	allowed := originOf(frontendURL)
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" && !strings.EqualFold(strings.TrimRight(origin, "/"), allowed) {
			api.ErrorWithMessage(c, http.StatusForbidden, "FORBIDDEN_ORIGIN")
			c.Abort()
			return
		}
		c.Next()
	}
}

func originOf(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host
}
