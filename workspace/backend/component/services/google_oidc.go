package services

import (
	"context"
	"errors"
	"fmt"
	"main/utils"
	"net/http"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

const googleIssuer = "https://accounts.google.com"

// OIDCClient is the Google side of the login, behind an interface so
// AuthService can be tested without Google.
type OIDCClient interface {
	// AuthCodeURL builds the authorize URL; verifier is the PKCE code verifier
	// (the S256 challenge is derived from it).
	AuthCodeURL(state, nonce, verifier string) string
	// Exchange swaps the code (with the PKCE verifier) for tokens and verifies
	// the ID token's signature, iss, aud and exp. Errors are *AuthError
	// (EXCHANGE_FAILED or INVALID_ID_TOKEN).
	Exchange(ctx context.Context, code, verifier string) (*GoogleClaims, error)
}

type GoogleClaims struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	HostedDomain  string `json:"hd"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	Nonce         string `json:"nonce"`
}

type googleOIDC struct {
	oauth    oauth2.Config
	verifier *oidc.IDTokenVerifier
	client   *http.Client
	hdHint   string
}

// NewGoogleOIDC discovers Google's endpoints and keys. It makes a network call.
func NewGoogleOIDC(ctx context.Context, cfg utils.GoogleAuthConfig) (OIDCClient, error) {
	g, err := newOIDCClient(ctx, cfg, googleIssuer, &http.Client{Timeout: 10 * time.Second})
	if err != nil {
		return nil, err // untyped nil, never a nil *googleOIDC inside the interface
	}
	return g, nil
}

func newOIDCClient(ctx context.Context, cfg utils.GoogleAuthConfig, issuer string, client *http.Client) (*googleOIDC, error) {
	if cfg.ClientID == "" || cfg.ClientSecret == "" || cfg.RedirectURI == "" {
		return nil, errors.New("auth.google client-id, client-secret and redirect-uri are required")
	}
	provider, err := oidc.NewProvider(oidc.ClientContext(ctx, client), issuer)
	if err != nil {
		return nil, fmt.Errorf("oidc discovery %s: %w", issuer, err)
	}
	g := &googleOIDC{
		oauth: oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURI,
			Endpoint:     provider.Endpoint(),
			Scopes:       []string{oidc.ScopeOpenID, "email", "profile"},
		},
		verifier: provider.Verifier(&oidc.Config{ClientID: cfg.ClientID}),
		client:   client,
	}
	if len(cfg.AllowedDomains) == 1 {
		g.hdHint = cfg.AllowedDomains[0] // UI hint only; the real check is in AuthService
	}
	return g, nil
}

func (g *googleOIDC) AuthCodeURL(state, nonce, verifier string) string {
	opts := []oauth2.AuthCodeOption{
		oidc.Nonce(nonce),
		oauth2.S256ChallengeOption(verifier),
		oauth2.SetAuthURLParam("prompt", "select_account"),
	}
	if g.hdHint != "" {
		opts = append(opts, oauth2.SetAuthURLParam("hd", g.hdHint))
	}
	return g.oauth.AuthCodeURL(state, opts...)
}

func (g *googleOIDC) Exchange(ctx context.Context, code, verifier string) (*GoogleClaims, error) {
	ctx = oidc.ClientContext(ctx, g.client)
	token, err := g.oauth.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return nil, wrapAuth(ErrExchangeFailed, sanitizeOAuthErr(err))
	}
	rawID, ok := token.Extra("id_token").(string)
	if !ok || rawID == "" {
		return nil, wrapAuth(ErrExchangeFailed, errors.New("token response has no id_token"))
	}
	idToken, err := g.verifier.Verify(ctx, rawID)
	if err != nil {
		return nil, wrapAuth(ErrInvalidIDToken, err)
	}
	var claims GoogleClaims
	if err := idToken.Claims(&claims); err != nil {
		return nil, wrapAuth(ErrInvalidIDToken, err)
	}
	// Google's access/refresh tokens are dropped here on purpose: only identity is needed.
	return &claims, nil
}

// sanitizeOAuthErr keeps the OAuth error code and HTTP status only, so a
// provider response body never ends up in the logs.
func sanitizeOAuthErr(err error) error {
	var re *oauth2.RetrieveError
	if errors.As(err, &re) {
		status := 0
		if re.Response != nil {
			status = re.Response.StatusCode
		}
		return fmt.Errorf("token endpoint returned %d %s", status, re.ErrorCode)
	}
	return errors.New("token endpoint request failed")
}
