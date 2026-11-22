package config

import (
	"log/slog"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const testDatabaseURL = "postgres://app:secret@db.internal:5432/app?sslmode=disable"

func TestLoadDefaults(t *testing.T) {
	t.Parallel()

	cfg, err := LoadFrom(map[string]string{"DATABASE_URL": testDatabaseURL})
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}

	if cfg.Env != EnvDevelopment || cfg.IsProduction() {
		t.Errorf("Env = %q; want development", cfg.Env)
	}
	if cfg.HTTPAddr != ":8080" {
		t.Errorf("HTTPAddr = %q; want :8080", cfg.HTTPAddr)
	}
	if cfg.LogFormat != "text" || cfg.LogLevel != slog.LevelInfo {
		t.Errorf("logging = %s/%s; want text/INFO", cfg.LogFormat, cfg.LogLevel)
	}
	if cfg.CookieSecure() {
		t.Error("CookieSecure = true in development; want false so plain-HTTP localhost works")
	}
	if !cfg.AutoMigrate {
		t.Error("AutoMigrate = false; want true by default")
	}
	if cfg.SessionLifetime != 7*24*time.Hour || cfg.ShutdownTimeout != 15*time.Second {
		t.Errorf("durations = %s, %s; want 168h, 15s", cfg.SessionLifetime, cfg.ShutdownTimeout)
	}
	if cfg.DatabaseMaxConns != 10 {
		t.Errorf("DatabaseMaxConns = %d; want 10", cfg.DatabaseMaxConns)
	}
	if len(cfg.TrustedProxies) != 0 || len(cfg.TrustedOrigins) != 0 {
		t.Errorf("trusted proxies/origins = %v/%v; want none", cfg.TrustedProxies, cfg.TrustedOrigins)
	}
}

func TestLoadProductionDefaults(t *testing.T) {
	t.Parallel()

	cfg, err := LoadFrom(map[string]string{"DATABASE_URL": testDatabaseURL, "APP_ENV": " Production "})
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}

	if !cfg.IsProduction() {
		t.Fatalf("Env = %q; want production", cfg.Env)
	}
	if cfg.LogFormat != "json" {
		t.Errorf("LogFormat = %q; want json in production", cfg.LogFormat)
	}
	if !cfg.CookieSecure() {
		t.Error("CookieSecure = false in production; want true")
	}
}

func TestLoadOverrides(t *testing.T) {
	t.Parallel()

	cfg, err := LoadFrom(map[string]string{
		"DATABASE_URL":          testDatabaseURL,
		"APP_ENV":               "production",
		"HTTP_ADDR":             "127.0.0.1:9000",
		"PORT":                  "3000", // HTTP_ADDR wins
		"DATABASE_MAX_CONNS":    "4",
		"DATABASE_AUTO_MIGRATE": "false",
		"LOG_LEVEL":             "debug",
		"LOG_FORMAT":            "TEXT",
		"SESSION_LIFETIME":      "12h",
		"SESSION_COOKIE_SECURE": "false",
		"TRUSTED_PROXIES":       "127.0.0.1, 10.0.0.0/8 ,::1,",
		"TRUSTED_ORIGINS":       "https://admin.example.com, ",
		"SHUTDOWN_TIMEOUT":      "5s",
	})
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}

	if cfg.HTTPAddr != "127.0.0.1:9000" {
		t.Errorf("HTTPAddr = %q; want 127.0.0.1:9000", cfg.HTTPAddr)
	}
	if cfg.DatabaseMaxConns != 4 || cfg.AutoMigrate {
		t.Errorf("database = %d conns, auto-migrate %v; want 4, false", cfg.DatabaseMaxConns, cfg.AutoMigrate)
	}
	if cfg.LogLevel != slog.LevelDebug || cfg.LogFormat != "text" {
		t.Errorf("logging = %s/%s; want text/DEBUG", cfg.LogFormat, cfg.LogLevel)
	}
	if cfg.CookieSecure() {
		t.Error("CookieSecure = true; an explicit SESSION_COOKIE_SECURE=false must win over the production default")
	}
	if cfg.SessionLifetime != 12*time.Hour || cfg.ShutdownTimeout != 5*time.Second {
		t.Errorf("durations = %s, %s; want 12h, 5s", cfg.SessionLifetime, cfg.ShutdownTimeout)
	}

	wantProxies := []netip.Prefix{
		netip.MustParsePrefix("127.0.0.1/32"),
		netip.MustParsePrefix("10.0.0.0/8"),
		netip.MustParsePrefix("::1/128"),
	}
	if len(cfg.TrustedProxies) != len(wantProxies) {
		t.Fatalf("TrustedProxies = %v; want %v", cfg.TrustedProxies, wantProxies)
	}
	for i, want := range wantProxies {
		if cfg.TrustedProxies[i] != want {
			t.Errorf("TrustedProxies[%d] = %v; want %v", i, cfg.TrustedProxies[i], want)
		}
	}
	if len(cfg.TrustedOrigins) != 1 || cfg.TrustedOrigins[0] != "https://admin.example.com" {
		t.Errorf("TrustedOrigins = %v; want [https://admin.example.com]", cfg.TrustedOrigins)
	}
}

func TestLoadUsesPortWhenAddrIsUnset(t *testing.T) {
	t.Parallel()

	cfg, err := LoadFrom(map[string]string{"DATABASE_URL": testDatabaseURL, "PORT": "3000"})
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if cfg.HTTPAddr != ":3000" {
		t.Errorf("HTTPAddr = %q; want :3000", cfg.HTTPAddr)
	}
}

func TestLoadRejectsBadValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		env  map[string]string
		want string // substring of the error
	}{
		{"missing database URL", map[string]string{}, "DATABASE_URL"},
		{"empty database URL", map[string]string{"DATABASE_URL": ""}, "DATABASE_URL"},
		{"unknown environment", map[string]string{"APP_ENV": "staging"}, "APP_ENV"},
		{"unknown log format", map[string]string{"LOG_FORMAT": "xml"}, "LOG_FORMAT"},
		{"unknown log level", map[string]string{"LOG_LEVEL": "loud"}, "LogLevel"},
		{"bad duration", map[string]string{"SESSION_LIFETIME": "a week"}, "SessionLifetime"},
		{"negative lifetime", map[string]string{"SESSION_LIFETIME": "-1h"}, "SESSION_LIFETIME"},
		{"zero pool", map[string]string{"DATABASE_MAX_CONNS": "0"}, "DATABASE_MAX_CONNS"},
		{"bad proxy", map[string]string{"TRUSTED_PROXIES": "127.0.0.1,proxy.internal"}, "proxy.internal"},
	}
	for _, tt := range tests {
		env := map[string]string{"DATABASE_URL": testDatabaseURL}
		if tt.name == "missing database URL" {
			env = map[string]string{}
		}
		for k, v := range tt.env {
			env[k] = v
		}

		_, err := LoadFrom(env)
		if err == nil {
			t.Errorf("%s: LoadFrom succeeded; want an error mentioning %q", tt.name, tt.want)
			continue
		}
		if !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: error %q does not mention %q", tt.name, err, tt.want)
		}
	}
}

func TestDatabaseTargetOmitsCredentials(t *testing.T) {
	t.Parallel()

	cfg := Config{DatabaseURL: testDatabaseURL}
	if got := cfg.DatabaseTarget(); got != "db.internal:5432/app" {
		t.Errorf("DatabaseTarget = %q; want db.internal:5432/app", got)
	}
	if got := (Config{DatabaseURL: "host=db user=app password=secret"}).DatabaseTarget(); strings.Contains(got, "secret") {
		t.Errorf("DatabaseTarget leaked the password: %q", got)
	}
}

func TestLoadDotEnv(t *testing.T) {
	// Not parallel: it changes the process environment.
	path := filepath.Join(t.TempDir(), ".env")
	contents := strings.Join([]string{
		"# a comment",
		"",
		"DOTENV_TEST_PLAIN=plain value",
		`DOTENV_TEST_DOUBLE="double quoted"`,
		"DOTENV_TEST_SINGLE='single quoted'",
		"export DOTENV_TEST_EXPORT=exported",
		"DOTENV_TEST_EQUALS=postgres://u:p@h/db?sslmode=disable",
		"DOTENV_TEST_EMPTY=",
		"DOTENV_TEST_PRESET=from file",
		"not an assignment",
		"=no key",
	}, "\n")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	keys := []string{
		"DOTENV_TEST_PLAIN", "DOTENV_TEST_DOUBLE", "DOTENV_TEST_SINGLE", "DOTENV_TEST_EXPORT",
		"DOTENV_TEST_EQUALS", "DOTENV_TEST_EMPTY", "DOTENV_TEST_PRESET",
	}
	t.Cleanup(func() {
		for _, key := range keys {
			_ = os.Unsetenv(key)
		}
	})
	t.Setenv("DOTENV_TEST_PRESET", "from environment")

	if err := LoadDotEnv(path); err != nil {
		t.Fatalf("LoadDotEnv: %v", err)
	}

	want := map[string]string{
		"DOTENV_TEST_PLAIN":  "plain value",
		"DOTENV_TEST_DOUBLE": "double quoted",
		"DOTENV_TEST_SINGLE": "single quoted",
		"DOTENV_TEST_EXPORT": "exported",
		"DOTENV_TEST_EQUALS": "postgres://u:p@h/db?sslmode=disable",
		"DOTENV_TEST_EMPTY":  "",
		// The real environment always beats the file.
		"DOTENV_TEST_PRESET": "from environment",
	}
	for key, value := range want {
		if got, ok := os.LookupEnv(key); !ok || got != value {
			t.Errorf("%s = %q (set: %v); want %q", key, got, ok, value)
		}
	}
}

func TestLoadDotEnvIgnoresMissingFile(t *testing.T) {
	t.Parallel()

	if err := LoadDotEnv(filepath.Join(t.TempDir(), "absent.env")); err != nil {
		t.Errorf("LoadDotEnv(missing file) = %v; want nil", err)
	}
}
