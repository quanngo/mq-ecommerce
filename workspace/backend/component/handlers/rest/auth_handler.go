package rest

import (
	"main/component/api"
	"main/component/models"
	"main/component/services"
	"main/middleware"
	"main/utils"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// AuthHandler serves /api/v1/auth. JSON endpoints follow the api.Ok /
// api.ErrorWithMessage convention (HTTP 200, real status in `code`, error code
// in `message`). The two Google endpoints are browser redirects.
type AuthHandler struct {
	AuthSvc services.AuthService
	Cfg     utils.AuthConfig // after WithDefaults()
}

func (h *AuthHandler) Router(r *gin.RouterGroup, requireAuth, sameOrigin gin.HandlerFunc) *gin.RouterGroup {
	r.GET("providers", h.Providers)
	r.GET("google/login", h.GoogleLogin)
	r.GET("google/callback", h.GoogleCallback)
	r.POST("login", sameOrigin, h.PasswordLogin)
	r.GET("me", requireAuth, h.Me)
	r.POST("logout", sameOrigin, h.Logout)
	r.POST("logout-all", sameOrigin, requireAuth, h.LogoutAll)
	r.POST("change-password", sameOrigin, requireAuth, h.ChangePassword)
	return r
}

// Providers godoc
// @Summary  Login methods that are currently available
// @Tags     auth
// @Produce  json
// @Success  200 {object} models.ProvidersResponse
// @Router   /v1/auth/providers [get]
func (h *AuthHandler) Providers(c *gin.Context) {
	api.Ok(c, h.AuthSvc.Providers())
}

// GoogleLogin godoc
// @Summary  Start Google sign-in (302 to Google)
// @Tags     auth
// @Param    next query string false "FE path to return to after login"
// @Param    remember query string false "1 = remember me (long session, persistent cookie)"
// @Success  302
// @Router   /v1/auth/google/login [get]
func (h *AuthHandler) GoogleLogin(c *gin.Context) {
	authURL, err := h.AuthSvc.BeginGoogleLogin(c.Request.Context(), c.Query("next"), c.Query("remember") == "1")
	if err != nil {
		h.redirectWithError(c, err)
		return
	}
	noStore(c)
	c.Redirect(http.StatusFound, authURL)
}

// GoogleCallback godoc
// @Summary  Google redirect target: sets the session cookie and 302s to the FE
// @Tags     auth
// @Success  302
// @Router   /v1/auth/google/callback [get]
func (h *AuthHandler) GoogleCallback(c *gin.Context) {
	session, redirectPath, err := h.AuthSvc.CompleteGoogleLogin(c.Request.Context(), services.GoogleCallbackInput{
		Code:          c.Query("code"),
		State:         c.Query("state"),
		ProviderError: c.Query("error"),
		Meta:          clientMeta(c),
	})
	if err != nil {
		h.redirectWithError(c, err)
		return
	}
	h.setSessionCookie(c, session)
	noStore(c)
	c.Redirect(http.StatusFound, h.Cfg.FrontendURL+redirectPath)
}

// PasswordLogin godoc
// @Summary  Sign in with an internal email and password
// @Tags     auth
// @Accept   json
// @Produce  json
// @Param    body body models.PasswordLoginRequest true "credentials"
// @Success  200 {object} models.UserResponse
// @Router   /v1/auth/login [post]
func (h *AuthHandler) PasswordLogin(c *gin.Context) {
	var req models.PasswordLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Never echo err.Error(): it can contain the submitted body.
		h.respondError(c, services.ErrInvalidRequest)
		return
	}
	meta := clientMeta(c)
	meta.RememberMe = req.RememberMe
	session, err := h.AuthSvc.LoginWithPassword(c.Request.Context(), req.Email, req.Password, meta)
	if err != nil {
		h.respondError(c, err)
		return
	}
	h.setSessionCookie(c, session)
	noStore(c)
	api.Ok(c, models.NewUserResponse(session.User))
}

// Me godoc
// @Summary  The signed-in user
// @Tags     auth
// @Produce  json
// @Success  200 {object} models.UserResponse
// @Router   /v1/auth/me [get]
func (h *AuthHandler) Me(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		h.respondError(c, services.ErrUnauthenticated)
		return
	}
	noStore(c)
	api.Ok(c, models.NewUserResponse(*user))
}

// Logout godoc
// @Summary  Revoke the current session and clear the cookie
// @Tags     auth
// @Produce  json
// @Success  200
// @Router   /v1/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	token, _ := c.Cookie(h.Cfg.CookieName)
	h.clearSessionCookie(c)
	if token == "" {
		h.respondError(c, services.ErrUnauthenticated)
		return
	}
	if err := h.AuthSvc.Logout(c.Request.Context(), token); err != nil {
		h.respondError(c, err)
		return
	}
	api.Ok(c)
}

// LogoutAll godoc
// @Summary  Revoke every session of the signed-in user (all devices) and clear this cookie
// @Tags     auth
// @Produce  json
// @Success  200 {object} models.LogoutAllResponse
// @Router   /v1/auth/logout-all [post]
func (h *AuthHandler) LogoutAll(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		h.respondError(c, services.ErrUnauthenticated)
		return
	}
	n, err := h.AuthSvc.LogoutAll(c.Request.Context(), user.ID)
	if err != nil {
		h.respondError(c, err)
		return
	}
	h.clearSessionCookie(c)
	noStore(c)
	api.Ok(c, models.LogoutAllResponse{RevokedSessions: n})
}

func (h *AuthHandler) setSessionCookie(c *gin.Context, s *models.NewSession) {
	// Without "remember me" the cookie has no Max-Age, so it dies with the
	// browser; the server-side expiry (session-ttl-hours) still applies.
	maxAge := 0
	if s.Persistent {
		maxAge = int(time.Until(s.ExpiresAt).Seconds())
	}
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(h.Cfg.CookieName, s.Token, maxAge, "/", "", h.Cfg.CookieSecure, true)
}

func (h *AuthHandler) clearSessionCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(h.Cfg.CookieName, "", -1, "/", "", h.Cfg.CookieSecure, true)
}

// respondError sends only the stable error code; the cause goes to the logs.
// ChangePassword godoc
// @Summary  Change your own password (current password required if you have one); signs out your other sessions
// @Tags     auth
// @Accept   json
// @Produce  json
// @Param    body body models.ChangePasswordRequest true "passwords"
// @Success  200
// @Router   /v1/auth/change-password [post]
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req models.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, services.ErrInvalidRequest) // never echo the body
		return
	}
	if err := h.AuthSvc.ChangePassword(c.Request.Context(), middleware.CurrentUser(c), req.CurrentPassword, req.NewPassword); err != nil {
		h.respondError(c, err)
		return
	}
	noStore(c)
	api.Ok(c)
}

func (h *AuthHandler) respondError(c *gin.Context, err error) { respondAuthError(c, err) }

// respondAuthError sends only the stable error code; the cause goes to the
// logs (for server errors only, so expected 4xx don't flood them).
func respondAuthError(c *gin.Context, err error) {
	ae := services.AsAuthError(err)
	if ae.Status >= http.StatusInternalServerError {
		utils.ShowErrorLogs(err)
	}
	api.ErrorWithMessage(c, ae.Status, ae.Code)
}

// redirectWithError sends the browser back to the FE login page with ?error=CODE.
func (h *AuthHandler) redirectWithError(c *gin.Context, err error) {
	ae := services.AsAuthError(err)
	utils.ShowErrorLogs(err)
	noStore(c)
	c.Redirect(http.StatusFound, h.Cfg.FrontendURL+"/login?error="+url.QueryEscape(ae.Code))
}

// noStore keeps auth responses (and the code in the callback URL) out of caches and Referer headers.
func noStore(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("Referrer-Policy", "no-referrer")
}

func clientMeta(c *gin.Context) services.ClientMeta {
	return services.ClientMeta{UserAgent: c.Request.UserAgent(), IP: c.ClientIP()}
}

// trimTrailingSlash lets frontend-url be configured with or without a trailing "/".
func trimTrailingSlash(s string) string { return strings.TrimRight(s, "/") }
