// Package i18n holds the backend's translated text (EC-7) and works out which
// language a request wants.
//
// Catalogs are flat JSON files in locales/<locale>.json, embedded in the
// binary. Error codes in the response envelope stay language-neutral: only
// human-readable text (error_text, role and permission names) is translated.
// To add a language, add locales/<code>.json with the same keys as vi.json and
// append the code to Supported.
package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	VI = "vi"
	EN = "en"

	// Default is used when nothing else matches; the admin app was Vietnamese-only before EC-7.
	Default = VI

	// ContextKey holds the request's locale in the gin context (see Locale / Set).
	ContextKey = "i18n.locale"
)

// Supported lists the locales with a catalog, in display order.
var Supported = []string{VI, EN}

//go:embed locales/*.json
var localeFiles embed.FS

var catalogs = mustLoad()

func mustLoad() map[string]map[string]string {
	out := make(map[string]map[string]string, len(Supported))
	for _, l := range Supported {
		raw, err := localeFiles.ReadFile("locales/" + l + ".json")
		if err != nil {
			panic(fmt.Sprintf("i18n: missing catalog %s: %v", l, err))
		}
		m := map[string]string{}
		if err := json.Unmarshal(raw, &m); err != nil {
			panic(fmt.Sprintf("i18n: invalid catalog %s: %v", l, err))
		}
		out[l] = m
	}
	return out
}

// Normalize returns the supported locale for a tag like "en", "en-US" or
// "VI_vn", or "" when it isn't supported.
func Normalize(tag string) string {
	tag = strings.ToLower(strings.TrimSpace(tag))
	if i := strings.IndexAny(tag, "-_"); i >= 0 {
		tag = tag[:i]
	}
	if _, ok := catalogs[tag]; ok {
		return tag
	}
	return ""
}

// IsSupported reports whether locale is exactly one of Supported.
func IsSupported(locale string) bool {
	_, ok := catalogs[locale]
	return ok
}

// FromAcceptLanguage picks the supported locale with the highest q-value in
// an Accept-Language header ("en-US,en;q=0.9,vi;q=0.8"), or Default.
func FromAcceptLanguage(header string) string {
	best, bestQ := "", -1.0
	for _, part := range strings.Split(header, ",") {
		fields := strings.Split(part, ";")
		locale := Normalize(fields[0])
		if locale == "" {
			continue
		}
		q := 1.0
		for _, f := range fields[1:] {
			f = strings.TrimSpace(f)
			if strings.HasPrefix(f, "q=") {
				if _, err := fmt.Sscanf(f[2:], "%g", &q); err != nil {
					q = 0
				}
			}
		}
		if q > bestQ && q > 0 {
			best, bestQ = locale, q
		}
	}
	if best == "" {
		return Default
	}
	return best
}

// Lookup returns the text for key in locale, falling back to Default.
func Lookup(locale, key string) (string, bool) {
	if s, ok := catalogs[locale][key]; ok {
		return s, true
	}
	s, ok := catalogs[Default][key]
	return s, ok
}

// T is Lookup that returns the key itself when no catalog has it.
func T(locale, key string) string {
	if s, ok := Lookup(locale, key); ok {
		return s
	}
	return key
}

// Keys returns the sorted keys of one catalog (for parity tests).
func Keys(locale string) []string {
	keys := make([]string, 0, len(catalogs[locale]))
	for k := range catalogs[locale] {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Locale returns the request's locale as set by middleware.Locale /
// middleware.RequireAuth, or Default.
func Locale(c *gin.Context) string {
	if c != nil {
		if v, ok := c.Get(ContextKey); ok {
			if s, _ := v.(string); IsSupported(s) {
				return s
			}
		}
	}
	return Default
}

// Set stores locale for the rest of the request and announces it in Content-Language.
func Set(c *gin.Context, locale string) {
	if !IsSupported(locale) {
		return
	}
	c.Set(ContextKey, locale)
	c.Header("Content-Language", locale)
}

// RoleName / RoleDescription translate a role by code; custom roles that have
// no catalog entry keep the text stored in the DB.
func RoleName(locale, code, fallback string) string {
	if s, ok := Lookup(locale, "role."+code+".name"); ok {
		return s
	}
	return fallback
}

func RoleDescription(locale, code, fallback string) string {
	if s, ok := Lookup(locale, "role."+code+".description"); ok {
		return s
	}
	return fallback
}

// PermissionName translates a permission code, or returns the code itself.
func PermissionName(locale, code string) string {
	if s, ok := Lookup(locale, "permission."+code); ok {
		return s
	}
	return code
}

// PermissionNames maps each code to its translated name.
func PermissionNames(locale string, codes []string) map[string]string {
	out := make(map[string]string, len(codes))
	for _, c := range codes {
		out[c] = PermissionName(locale, c)
	}
	return out
}
