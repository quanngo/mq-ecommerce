package services

import (
	"errors"
	"net/http"
)

// AuthError carries a stable Code that the frontend switches on and the
// status that goes into the api response `code`. The cause is for server logs
// only and never reaches the client.
type AuthError struct {
	Code   string
	Status int
	cause  error
}

func (e *AuthError) Error() string {
	if e.cause != nil {
		return e.Code + ": " + e.cause.Error()
	}
	return e.Code
}

func (e *AuthError) Unwrap() error { return e.cause }

// Is matches on Code, so errors.Is(wrapAuth(ErrInvalidState, x), ErrInvalidState) is true.
func (e *AuthError) Is(target error) bool {
	t, ok := target.(*AuthError)
	return ok && t.Code == e.Code
}

var (
	ErrInvalidRequest     = &AuthError{Code: "INVALID_REQUEST", Status: http.StatusBadRequest}
	ErrInvalidCredentials = &AuthError{Code: "INVALID_CREDENTIALS", Status: http.StatusUnauthorized}
	ErrAccountLocked      = &AuthError{Code: "ACCOUNT_LOCKED", Status: http.StatusLocked}
	ErrUnauthenticated    = &AuthError{Code: "UNAUTHENTICATED", Status: http.StatusUnauthorized}
	ErrForbiddenOrigin    = &AuthError{Code: "FORBIDDEN_ORIGIN", Status: http.StatusForbidden}
	ErrGoogleDisabled     = &AuthError{Code: "GOOGLE_DISABLED", Status: http.StatusServiceUnavailable}
	ErrConsentDenied      = &AuthError{Code: "CONSENT_DENIED", Status: http.StatusBadRequest}
	ErrInvalidState       = &AuthError{Code: "INVALID_STATE", Status: http.StatusBadRequest}
	ErrExchangeFailed     = &AuthError{Code: "EXCHANGE_FAILED", Status: http.StatusBadGateway}
	ErrInvalidIDToken     = &AuthError{Code: "INVALID_ID_TOKEN", Status: http.StatusUnauthorized}
	ErrEmailNotVerified   = &AuthError{Code: "EMAIL_NOT_VERIFIED", Status: http.StatusForbidden}
	ErrDomainNotAllowed   = &AuthError{Code: "DOMAIN_NOT_ALLOWED", Status: http.StatusForbidden}
	ErrUserNotAllowed     = &AuthError{Code: "USER_NOT_ALLOWED", Status: http.StatusForbidden}
	ErrInternal           = &AuthError{Code: "INTERNAL_ERROR", Status: http.StatusInternalServerError}

	// Roles, user management, password changes.
	ErrForbidden     = &AuthError{Code: "FORBIDDEN", Status: http.StatusForbidden}
	ErrNotFound      = &AuthError{Code: "NOT_FOUND", Status: http.StatusNotFound}
	ErrWeakPassword  = &AuthError{Code: "WEAK_PASSWORD", Status: http.StatusBadRequest}
	ErrWrongPassword = &AuthError{Code: "WRONG_PASSWORD", Status: http.StatusBadRequest} // current password on change
	ErrEmailTaken    = &AuthError{Code: "EMAIL_TAKEN", Status: http.StatusConflict}
	ErrInvalidRole   = &AuthError{Code: "INVALID_ROLE", Status: http.StatusBadRequest}
	ErrLastAdmin     = &AuthError{Code: "LAST_ADMIN", Status: http.StatusConflict}         // would leave nobody able to manage users
	ErrSelfModify    = &AuthError{Code: "CANNOT_MODIFY_SELF", Status: http.StatusConflict} // own role/status/reset via user management

	// Product images (EC-10).
	ErrInvalidImage = &AuthError{Code: "INVALID_IMAGE", Status: http.StatusBadRequest}     // not a JPG/PNG, unreadable, empty or too large
	ErrMarketplace  = &AuthError{Code: "MARKETPLACE_ERROR", Status: http.StatusBadGateway} // the marketplace refused the call or was unreachable
)

// wrapAuth keeps base's Code and Status and attaches cause for the logs.
func wrapAuth(base *AuthError, cause error) *AuthError {
	return &AuthError{Code: base.Code, Status: base.Status, cause: cause}
}

// AsAuthError returns err as an *AuthError; anything else becomes INTERNAL_ERROR.
func AsAuthError(err error) *AuthError {
	var ae *AuthError
	if errors.As(err, &ae) {
		return ae
	}
	return wrapAuth(ErrInternal, err)
}
