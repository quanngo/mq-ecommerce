package server

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"main/utils"
)

func TestParseCreateUserArgs(t *testing.T) {
	in, err := parseCreateUserArgs([]string{"--email", "an@mq.com", "--name", "An Nguyen", "--role", "admin", "--password-stdin"}, strings.NewReader("s3cret-pass-1\r\nignored"))
	if err != nil || in.Email != "an@mq.com" || in.DisplayName != "An Nguyen" || in.Role != "admin" || in.Password == nil || *in.Password != "s3cret-pass-1" {
		t.Fatalf("got (%+v, %v)", in, err)
	}

	in, err = parseCreateUserArgs([]string{"--email", "g@mq.com"}, strings.NewReader("should not be read"))
	if err != nil || in.Password != nil || in.Role != "" {
		t.Errorf("google-only: %+v err=%v (stdin must be ignored without --password-stdin; role defaults later)", in, err)
	}

	in, err = parseCreateUserArgs([]string{"--email", "a@mq.com", "--password-stdin"}, strings.NewReader("no-newline-at-eof"))
	if err != nil || in.Password == nil || *in.Password != "no-newline-at-eof" {
		t.Errorf("password without trailing newline: %+v err=%v", in, err)
	}

	for name, tc := range map[string]struct {
		args  []string
		stdin string
	}{
		"missing email": {[]string{"--name", "x"}, ""},
		"blank email":   {[]string{"--email", "  "}, ""},
		"empty stdin":   {[]string{"--email", "a@mq.com", "--password-stdin"}, ""},
		"blank line":    {[]string{"--email", "a@mq.com", "--password-stdin"}, "\n"},
		"unknown flag":  {[]string{"--email", "a@mq.com", "--password", "inline"}, ""},
	} {
		if _, err := parseCreateUserArgs(tc.args, strings.NewReader(tc.stdin)); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestNewGoogleClient_DisabledOrMisconfiguredIsUntypedNil(t *testing.T) {
	// Must be a nil interface (not a nil *googleOIDC inside it), or
	// AuthService.Providers() would report Google as available.
	if c := newGoogleClient(utils.GoogleAuthConfig{Enable: false, ClientID: "x", ClientSecret: "y", RedirectURI: "z"}); c != nil {
		t.Errorf("disabled: got %#v, want nil", c)
	}
	if c := newGoogleClient(utils.GoogleAuthConfig{Enable: true}); c != nil {
		t.Errorf("enabled without client-id/secret: got %#v, want nil", c)
	}
}

func TestNewAuthService_WithoutGoogle(t *testing.T) {
	svc := newAuthService(nil, utils.AuthConfig{}, nil)
	if p := svc.Providers(); p.Google || !p.Password {
		t.Errorf("Providers() = %+v, want google=false password=true", p)
	}
}

func TestInitConfig_ReadsAuthBlock(t *testing.T) {
	dir := t.TempDir()
	yml := `
auth:
  frontend-url: https://admin.mq.com
  cookie-secure: true
  session-ttl-hours: 8
  google:
    enable: true
    client-id: cid
    client-secret: csecret
    redirect-uri: https://api.mq.com/api/v1/auth/google/callback
    allowed-domains: ["mq.com"]
    auto-provision: true
`
	if err := os.WriteFile(filepath.Join(dir, "application.yml"), []byte(yml), 0o600); err != nil {
		t.Fatal(err)
	}
	old := configPath
	configPath = dir
	t.Cleanup(func() { configPath = old })

	cfg, err := initConfig()
	if err != nil {
		t.Fatal(err)
	}
	a := cfg.Auth
	if a.FrontendURL != "https://admin.mq.com" || !a.CookieSecure || a.SessionTTLHours != 8 {
		t.Errorf("auth = %+v", a)
	}
	g := a.Google
	if !g.Enable || g.ClientID != "cid" || g.ClientSecret != "csecret" || g.RedirectURI == "" ||
		len(g.AllowedDomains) != 1 || g.AllowedDomains[0] != "mq.com" || !g.AutoProvision {
		t.Errorf("google = %+v", g)
	}
	if d := a.WithDefaults(); d.CookieName != "ecom_session" || d.MaxFailedLogins != 5 {
		t.Errorf("missing keys did not fall back to defaults: %+v", d)
	}
}
