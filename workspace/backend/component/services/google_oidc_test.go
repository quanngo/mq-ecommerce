package services

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"main/utils"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"golang.org/x/oauth2"
)

// fakeIdP is a minimal OpenID provider: discovery, JWKS and a token endpoint
// that returns an ID token signed with a test RSA key.
type fakeIdP struct {
	srv         *httptest.Server
	key         *rsa.PrivateKey
	claims      map[string]interface{}
	tokenStatus int
	gotVerifier string
	signWithKey *rsa.PrivateKey // nil = the published key
	omitIDToken bool
}

func newFakeIdP(t *testing.T) *fakeIdP {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeIdP{key: key, tokenStatus: http.StatusOK}
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"issuer":                                f.srv.URL,
			"authorization_endpoint":                f.srv.URL + "/auth",
			"token_endpoint":                        f.srv.URL + "/token",
			"jwks_uri":                              f.srv.URL + "/jwks",
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	})
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: &key.PublicKey, KeyID: "k1", Algorithm: "RS256", Use: "sig"}}})
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		f.gotVerifier = r.Form.Get("code_verifier")
		w.Header().Set("Content-Type", "application/json")
		if f.tokenStatus != http.StatusOK {
			w.WriteHeader(f.tokenStatus)
			w.Write([]byte(`{"error":"invalid_grant","error_description":"echo of code ` + r.Form.Get("code") + `"}`))
			return
		}
		resp := map[string]interface{}{"access_token": "google-access-token", "token_type": "Bearer", "expires_in": 3600}
		if !f.omitIDToken {
			resp["id_token"] = f.sign(t)
		}
		json.NewEncoder(w).Encode(resp)
	})
	f.srv = httptest.NewServer(mux)
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeIdP) sign(t *testing.T) string {
	key := f.key
	if f.signWithKey != nil {
		key = f.signWithKey
	}
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: key},
		(&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", "k1"))
	if err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(f.claims)
	obj, err := signer.Sign(payload)
	if err != nil {
		t.Fatal(err)
	}
	s, _ := obj.CompactSerialize()
	return s
}

func (f *fakeIdP) baseClaims(clientID string) map[string]interface{} {
	return map[string]interface{}{
		"iss": f.srv.URL, "aud": clientID, "sub": "g-1",
		"exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Unix(),
		"email": "an.nguyen@mq.com", "email_verified": true, "hd": "mq.com",
		"name": "An Nguyen", "nonce": "n-1",
	}
}

func testGoogleCfg() utils.GoogleAuthConfig {
	return utils.GoogleAuthConfig{
		Enable: true, ClientID: "client-1", ClientSecret: "super-secret",
		RedirectURI: "http://localhost:9091/api/v1/auth/google/callback", AllowedDomains: []string{"mq.com"},
	}
}

func TestGoogleOIDC_AuthCodeURL(t *testing.T) {
	idp := newFakeIdP(t)
	g, err := newOIDCClient(context.Background(), testGoogleCfg(), idp.srv.URL, idp.srv.Client())
	if err != nil {
		t.Fatal(err)
	}
	raw := g.AuthCodeURL("st-1", "n-1", "verifier-verifier-verifier-verifier-verifie")
	u, _ := url.Parse(raw)
	q := u.Query()
	want := map[string]string{
		"client_id": "client-1", "state": "st-1", "nonce": "n-1", "response_type": "code",
		"redirect_uri": "http://localhost:9091/api/v1/auth/google/callback", "code_challenge_method": "S256",
		"code_challenge": oauth2.S256ChallengeFromVerifier("verifier-verifier-verifier-verifier-verifie"),
		"scope":          "openid email profile", "hd": "mq.com",
	}
	for k, v := range want {
		if q.Get(k) != v {
			t.Errorf("%s = %q, want %q", k, q.Get(k), v)
		}
	}
	if q.Has("client_secret") || q.Has("code_verifier") {
		t.Error("secret or verifier leaked into the authorize URL")
	}
}

func TestGoogleOIDC_Exchange(t *testing.T) {
	cfg := testGoogleCfg()
	otherKey, _ := rsa.GenerateKey(rand.Reader, 2048)

	cases := []struct {
		name   string
		mutate func(f *fakeIdP)
		want   *AuthError // nil = success
	}{
		{"valid", func(f *fakeIdP) {}, nil},
		{"token endpoint 400", func(f *fakeIdP) { f.tokenStatus = http.StatusBadRequest }, ErrExchangeFailed},
		{"no id_token", func(f *fakeIdP) { f.omitIDToken = true }, ErrExchangeFailed},
		{"wrong signing key", func(f *fakeIdP) { f.signWithKey = otherKey }, ErrInvalidIDToken},
		{"wrong audience", func(f *fakeIdP) { f.claims["aud"] = "someone-else" }, ErrInvalidIDToken},
		{"wrong issuer", func(f *fakeIdP) { f.claims["iss"] = "https://evil.example" }, ErrInvalidIDToken},
		{"expired", func(f *fakeIdP) { f.claims["exp"] = time.Now().Add(-time.Hour).Unix() }, ErrInvalidIDToken},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			idp := newFakeIdP(t)
			idp.claims = idp.baseClaims(cfg.ClientID)
			tc.mutate(idp)
			g, err := newOIDCClient(context.Background(), cfg, idp.srv.URL, idp.srv.Client())
			if err != nil {
				t.Fatal(err)
			}
			claims, err := g.Exchange(context.Background(), "the-auth-code", "my-verifier")
			if tc.want == nil {
				if err != nil {
					t.Fatalf("Exchange: %v", err)
				}
				if claims.Email != "an.nguyen@mq.com" || !claims.EmailVerified || claims.HostedDomain != "mq.com" || claims.Nonce != "n-1" {
					t.Errorf("claims = %+v", claims)
				}
				if idp.gotVerifier != "my-verifier" {
					t.Errorf("token endpoint got code_verifier %q", idp.gotVerifier)
				}
				return
			}
			if !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %s", err, tc.want.Code)
			}
			// The provider echoed the code in its error body; it must not survive into our error.
			if msg := err.Error(); containsAny(msg, "the-auth-code", "super-secret") {
				t.Errorf("error leaks secrets: %s", msg)
			}
		})
	}
}

func TestNewGoogleOIDC_RequiresConfig(t *testing.T) {
	g, err := NewGoogleOIDC(context.Background(), utils.GoogleAuthConfig{Enable: true})
	if err == nil || g != nil {
		t.Fatalf("got (%v, %v), want (nil, error)", g, err)
	}
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if sub != "" && strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
