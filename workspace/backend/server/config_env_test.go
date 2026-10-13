package server

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func withConfigDir(t *testing.T, files map[string]string) {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	old := configPath
	configPath = dir
	t.Cleanup(func() { configPath = old })
}

func TestInitConfig_ExpandsPlaceholdersFromEnvAndDotEnv(t *testing.T) {
	t.Setenv("TEST_TIKTOK_SECRET", "from-real-env")
	os.Unsetenv("TEST_DB_PASSWORD") // set only via .env below
	os.Unsetenv("TEST_CONNECTOR_PASSWORD")
	t.Cleanup(func() {
		os.Unsetenv("TEST_DB_PASSWORD")
		os.Unsetenv("TEST_CONNECTOR_PASSWORD")
	})

	withConfigDir(t, map[string]string{
		"application.yml": `
database:
  host: 127.0.0.1
  port: 5555
  password: "${TEST_DB_PASSWORD}"
ecommerce:
  tiktok:
    app_secret: "${TEST_TIKTOK_SECRET}"
connector:
  password: "${TEST_CONNECTOR_PASSWORD}"
  host: "https://${TEST_UNSET_HOST}/api"
auth:
  google:
    allowed-domains: ["${TEST_UNSET_DOMAIN}", "mq.com"]
`,
		".env": `
# comment
TEST_DB_PASSWORD=123456
export TEST_TIKTOK_SECRET=from-dotenv-must-not-win
TEST_CONNECTOR_PASSWORD="p@ss: #with \"quotes\" and $dollar"
`,
	})

	cfg, err := initConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Database.Password != "123456" {
		t.Errorf("db password from .env = %q", cfg.Database.Password)
	}
	if cfg.Database.Port != 5555 || cfg.Database.Host != "127.0.0.1" {
		t.Error("non-secret values changed")
	}
	if cfg.Ecommerce.TiktokConfig.AppSecret != "from-real-env" {
		t.Errorf("real env must win over .env, got %q", cfg.Ecommerce.TiktokConfig.AppSecret)
	}
	if want := `p@ss: #with \"quotes\" and $dollar`; cfg.Connector.Password != want {
		t.Errorf("special characters mangled: %q, want %q", cfg.Connector.Password, want)
	}
	if cfg.Connector.Host != "https:///api" {
		t.Errorf("unset var should expand to empty: %q", cfg.Connector.Host)
	}
	if d := cfg.Auth.Google.AllowedDomains; len(d) != 2 || d[0] != "" || d[1] != "mq.com" {
		t.Errorf("slices not expanded: %q", d)
	}
}

func TestInitConfig_NoDotEnvIsFine(t *testing.T) {
	withConfigDir(t, map[string]string{"application.yml": "server:\n  port: 9091\n"})
	cfg, err := initConfig()
	if err != nil || cfg.Server.Port != 9091 {
		t.Fatalf("cfg=%+v err=%v", cfg, err)
	}
}

func TestInitConfig_DotEnvIsNotReadAsYAML(t *testing.T) {
	// initConfig decodes *.yml only; the .env file must never be parsed as YAML.
	withConfigDir(t, map[string]string{
		"application.yml": "server:\n  port: 9091\n",
		".env":            "SOME_KEY=: not yaml : at all\n",
	})
	t.Cleanup(func() { os.Unsetenv("SOME_KEY") })
	if _, err := initConfig(); err != nil {
		t.Fatal(err)
	}
}

func TestLoadDotEnv_RejectsMalformedLines(t *testing.T) {
	for _, bad := range []string{"NO_EQUALS_SIGN\n", "1BAD=x\n", "BAD-NAME=x\n"} {
		dir := t.TempDir()
		p := filepath.Join(dir, ".env")
		os.WriteFile(p, []byte(bad), 0o600)
		err := loadDotEnv(p)
		if err == nil {
			t.Errorf("%q accepted", bad)
			continue
		}
		if strings.Contains(err.Error(), "=x") {
			t.Errorf("error message echoes the value: %v", err)
		}
	}
}
