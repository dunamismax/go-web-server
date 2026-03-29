package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/v2"
)

func TestApplyDerivedDefaultsSetsCookieSecureByEnvironment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		environment string
		wantSecure  bool
	}{
		{
			name:        "development defaults to insecure cookies",
			environment: "development",
			wantSecure:  false,
		},
		{
			name:        "production defaults to secure cookies",
			environment: "production",
			wantSecure:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			k := koanf.New(".")
			cfg := Config{}
			cfg.App.Environment = tt.environment

			applyDerivedDefaults(k, &cfg)

			if cfg.Auth.CookieSecure != tt.wantSecure {
				t.Fatalf("CookieSecure = %t, want %t", cfg.Auth.CookieSecure, tt.wantSecure)
			}
		})
	}
}

func TestApplyDerivedDefaultsPreservesExplicitCookieSecure(t *testing.T) {
	t.Parallel()

	k := koanf.New(".")
	if err := k.Load(confmap.Provider(map[string]interface{}{
		"auth.cookie_secure": false,
	}, "."), nil); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	cfg := Config{}
	cfg.App.Environment = "production"
	cfg.Auth.CookieSecure = false

	applyDerivedDefaults(k, &cfg)

	if cfg.Auth.CookieSecure {
		t.Fatal("CookieSecure was overridden despite explicit configuration")
	}
}

func TestBuildDatabaseURLEscapesReservedCharacters(t *testing.T) {
	t.Parallel()

	got := buildDatabaseURL("app-user", "p@ss:/word", "localhost", "5432", "go-web-server", "disable")
	want := "postgres://app-user:p%40ss%3A%2Fword@localhost:5432/go-web-server?sslmode=disable"

	if got != want {
		t.Fatalf("buildDatabaseURL() = %q, want %q", got, want)
	}
}

func TestEnvKeyToKoanfKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "database url", in: "DATABASE_URL", want: "database.url"},
		{name: "nested underscore key", in: "AUTH_COOKIE_SECURE", want: "auth.cookie_secure"},
		{name: "duration key", in: "SERVER_READ_TIMEOUT", want: "server.read_timeout"},
		{name: "single token", in: "PORT", want: "port"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := envKeyToKoanfKey(tt.in); got != tt.want {
				t.Fatalf("envKeyToKoanfKey(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestLoadDotEnvFileMapsCommonKeys(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	content := "DATABASE_URL=postgres://gowebserver:secret@localhost:5432/gowebserver?sslmode=disable\n" +
		"AUTH_COOKIE_SECURE=false\n" +
		"SERVER_READ_TIMEOUT=17s\n" +
		"DATABASE_RUN_MIGRATIONS=false\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	k := koanf.New(".")
	if err := loadDotEnvFile(k, path); err != nil {
		t.Fatalf("loadDotEnvFile() error = %v", err)
	}

	if got := k.String("database.url"); got != "postgres://gowebserver:secret@localhost:5432/gowebserver?sslmode=disable" {
		t.Fatalf("database.url = %q", got)
	}
	if got := k.String("auth.cookie_secure"); got != "false" {
		t.Fatalf("auth.cookie_secure = %q, want %q", got, "false")
	}
	if got := k.String("server.read_timeout"); got != "17s" {
		t.Fatalf("server.read_timeout = %q, want %q", got, "17s")
	}
	if got := k.String("database.run_migrations"); got != "false" {
		t.Fatalf("database.run_migrations = %q, want %q", got, "false")
	}
}

func TestNewHonorsMappedEnvironmentVariables(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://gowebserver:secret@localhost:5432/gowebserver?sslmode=disable")
	t.Setenv("APP_ENVIRONMENT", "development")
	t.Setenv("AUTH_COOKIE_SECURE", "false")
	t.Setenv("DATABASE_RUN_MIGRATIONS", "false")
	t.Setenv("SERVER_READ_TIMEOUT", "17s")

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	t.Cleanup(func() {
		if chdirErr := os.Chdir(cwd); chdirErr != nil {
			t.Fatalf("Chdir() cleanup error = %v", chdirErr)
		}
	})

	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	cfg := New()

	if cfg.Auth.CookieSecure {
		t.Fatal("CookieSecure = true, want false from AUTH_COOKIE_SECURE")
	}
	if cfg.Database.RunMigrations {
		t.Fatal("RunMigrations = true, want false from DATABASE_RUN_MIGRATIONS")
	}
	if cfg.Server.ReadTimeout != 17*time.Second {
		t.Fatalf("ReadTimeout = %s, want %s", cfg.Server.ReadTimeout, 17*time.Second)
	}
}

func TestNewLoadsMappedDotEnvValues(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	t.Cleanup(func() {
		if chdirErr := os.Chdir(cwd); chdirErr != nil {
			t.Fatalf("Chdir() cleanup error = %v", chdirErr)
		}
	})

	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	content := "DATABASE_URL=postgres://gowebserver:secret@localhost:5432/gowebserver?sslmode=disable\n" +
		"AUTH_COOKIE_SECURE=false\n" +
		"SERVER_READ_TIMEOUT=17s\n" +
		"DATABASE_RUN_MIGRATIONS=false\n"
	if err := os.WriteFile(filepath.Join(tmp, ".env"), []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg := New()

	if cfg.Auth.CookieSecure {
		t.Fatal("CookieSecure = true, want false from .env")
	}
	if cfg.Database.RunMigrations {
		t.Fatal("RunMigrations = true, want false from .env")
	}
	if cfg.Server.ReadTimeout != 17*time.Second {
		t.Fatalf("ReadTimeout = %s, want %s", cfg.Server.ReadTimeout, 17*time.Second)
	}
}
