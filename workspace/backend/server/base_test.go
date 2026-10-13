package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestInitConfig_ReadsConnectorBlock(t *testing.T) {
	dir := t.TempDir()
	yml := `
connector:
  enable: true
  host: https://connector.example/
  callback: https://callback.example/
`
	if err := os.WriteFile(filepath.Join(dir, "application.yml"), []byte(yml), 0o600); err != nil {
		t.Fatal(err)
	}
	// Must be skipped: initConfig only loads *.yml.
	if err := os.WriteFile(filepath.Join(dir, "application.yml.example"), []byte("connector:\n  host: wrong\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	old := configPath
	configPath = dir
	t.Cleanup(func() { configPath = old })

	cfg, err := initConfig()
	if err != nil {
		t.Fatalf("initConfig() error = %v", err)
	}
	if !cfg.Connector.Enable {
		t.Error("Connector.Enable = false, want true")
	}
	if cfg.Connector.Host != "https://connector.example/" {
		t.Errorf("Connector.Host = %q", cfg.Connector.Host)
	}
	if cfg.Connector.Callback != "https://callback.example/" {
		t.Errorf("Connector.Callback = %q", cfg.Connector.Callback)
	}
}

func TestRequestLogger_DropsQueryString(t *testing.T) {
	var buf bytes.Buffer
	gin.DefaultWriter = &buf
	t.Cleanup(func() { gin.DefaultWriter = os.Stdout })

	gin.SetMode(gin.TestMode)
	app := gin.New()
	app.Use(requestLogger())
	app.GET("/api/v1/auth/google/callback", func(c *gin.Context) { c.Status(http.StatusFound) })
	app.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/auth/google/callback?code=SECRETCODE&state=SECRETSTATE", nil))

	out := buf.String()
	if !strings.Contains(out, "/api/v1/auth/google/callback") {
		t.Fatalf("path not logged: %q", out)
	}
	if strings.Contains(out, "SECRETCODE") || strings.Contains(out, "SECRETSTATE") {
		t.Errorf("query string logged: %q", out)
	}
}
