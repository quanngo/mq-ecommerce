package i18n

import (
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCatalogsHaveSameKeys(t *testing.T) {
	want := Keys(Default)
	if len(want) == 0 {
		t.Fatal("default catalog is empty")
	}
	for _, l := range Supported {
		if got := Keys(l); !reflect.DeepEqual(got, want) {
			t.Errorf("catalog %s keys differ from %s:\n got  %v\n want %v", l, Default, got, want)
		}
		for _, k := range Keys(l) {
			if catalogs[l][k] == "" {
				t.Errorf("catalog %s: empty text for %s", l, k)
			}
		}
	}
}

func TestNormalize(t *testing.T) {
	cases := map[string]string{"vi": VI, "VI-vn": VI, "en": EN, "en-US": EN, " en_GB ": EN, "fr": "", "": "", "e": ""}
	for in, want := range cases {
		if got := Normalize(in); got != want {
			t.Errorf("Normalize(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFromAcceptLanguage(t *testing.T) {
	cases := map[string]string{
		"":                            VI,
		"en-US,en;q=0.9":              EN,
		"vi-VN,vi;q=0.9,en;q=0.8":     VI,
		"fr-FR,en;q=0.5":              EN,
		"fr,de":                       VI,
		"vi;q=0.2, en;q=0.7":          EN,
		"en;q=0":                      VI,
		"en;q=abc,vi;q=0.1":           VI,
		"*":                           VI,
		"en-GB;q=0.8 , vi-VN ; q=0.9": VI,
	}
	for in, want := range cases {
		if got := FromAcceptLanguage(in); got != want {
			t.Errorf("FromAcceptLanguage(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestTFallbacks(t *testing.T) {
	if got := T(EN, "error.FORBIDDEN"); got != "You don't have permission to do this." {
		t.Errorf("en text = %q", got)
	}
	if got := T("fr", "error.FORBIDDEN"); got != T(VI, "error.FORBIDDEN") {
		t.Errorf("unsupported locale should fall back to %s, got %q", Default, got)
	}
	if got := T(EN, "no.such.key"); got != "no.such.key" {
		t.Errorf("missing key = %q, want the key", got)
	}
}

func TestRoleAndPermissionNames(t *testing.T) {
	if got := RoleName(EN, "admin", "Quản trị viên"); got != "Administrator" {
		t.Errorf("RoleName = %q", got)
	}
	if got := RoleName(EN, "custom", "Kho"); got != "Kho" {
		t.Errorf("custom role should keep DB name, got %q", got)
	}
	if got := RoleDescription(VI, "viewer", ""); got != "Chỉ xem dữ liệu" {
		t.Errorf("RoleDescription = %q", got)
	}
	got := PermissionNames(EN, []string{"orders.read", "x.y"})
	want := map[string]string{"orders.read": "View orders", "x.y": "x.y"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("PermissionNames = %v, want %v", got, want)
	}
}

func TestLocaleContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	if got := Locale(c); got != Default {
		t.Errorf("empty context = %q", got)
	}
	Set(c, "fr") // ignored
	if got := Locale(c); got != Default {
		t.Errorf("unsupported Set changed locale to %q", got)
	}
	Set(c, EN)
	if got := Locale(c); got != EN {
		t.Errorf("Locale = %q, want en", got)
	}
	if got := w.Header().Get("Content-Language"); got != EN {
		t.Errorf("Content-Language = %q", got)
	}
	if got := Locale(nil); got != Default {
		t.Errorf("nil context = %q", got)
	}
}
