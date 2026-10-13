package server

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"main/utils"
)

// Secrets are not written in config/*.yml. A value there may be a
// placeholder like "${TIKTOK_APP_SECRET}", which initConfig replaces with the
// environment variable of that name. For local runs the variables can live
// in config/.env (gitignored); real environment variables win over it.
//
// Placeholders are expanded after YAML decoding, on string fields only, so a
// secret containing ':', '#' or quotes cannot break the YAML.

const dotEnvFile = ".env"

var placeholder = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)

// loadDotEnv sets variables from a KEY=VALUE file without overriding ones that
// are already set. A missing file is fine. Values are never logged.
func loadDotEnv(path string) error {
	f, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(strings.TrimPrefix(sc.Text(), string(rune(0xFEFF)))) // tolerate a UTF-8 BOM
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, val, ok := strings.Cut(line, "=")
		key = strings.TrimSpace(key)
		if !ok || !placeholder.MatchString("${"+key+"}") {
			return fmt.Errorf("%s line %d: expected KEY=VALUE", path, n)
		}
		val = strings.TrimSpace(val)
		if len(val) >= 2 && (val[0] == '"' && val[len(val)-1] == '"' || val[0] == '\'' && val[len(val)-1] == '\'') {
			val = val[1 : len(val)-1]
		}
		if _, set := os.LookupEnv(key); !set {
			os.Setenv(key, val)
		}
	}
	return sc.Err()
}

// expandConfigEnv replaces ${VAR} in every string field of cfg and returns the
// names of referenced variables that are not set (they expand to "").
func expandConfigEnv(cfg *utils.Config) []string {
	missing := map[string]bool{}
	expandValue(reflect.ValueOf(cfg).Elem(), missing)
	names := make([]string, 0, len(missing))
	for name := range missing {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func expandValue(v reflect.Value, missing map[string]bool) {
	switch v.Kind() {
	case reflect.Ptr, reflect.Interface:
		if !v.IsNil() {
			expandValue(v.Elem(), missing)
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if v.Type().Field(i).IsExported() {
				expandValue(v.Field(i), missing)
			}
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			expandValue(v.Index(i), missing)
		}
	case reflect.String:
		if v.CanSet() && strings.Contains(v.String(), "${") {
			v.SetString(placeholder.ReplaceAllStringFunc(v.String(), func(m string) string {
				name := placeholder.FindStringSubmatch(m)[1]
				val, ok := os.LookupEnv(name)
				if !ok {
					missing[name] = true
				}
				return val
			}))
		}
	}
}

func dotEnvPath() string { return filepath.Join(configPath, dotEnvFile) }
