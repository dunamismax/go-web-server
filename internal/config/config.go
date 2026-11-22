// Package config loads the application's settings from environment
// variables. There is no config file: one source keeps deployments easy to
// reason about, and every variable is listed in .env.example.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
)

// The values APP_ENV accepts.
const (
	EnvDevelopment = "development"
	EnvProduction  = "production"
)

// Config is the application's complete configuration.
type Config struct {
	// Env selects development conveniences or production hardening.
	Env string `env:"APP_ENV" envDefault:"development"`

	// HTTPAddr is the listen address. When it is unset and PORT is set (as
	// most container platforms do), the server listens on ":$PORT".
	HTTPAddr string `env:"HTTP_ADDR"`
	Port     string `env:"PORT"`

	DatabaseURL      string `env:"DATABASE_URL,required,notEmpty"`
	DatabaseMaxConns int32  `env:"DATABASE_MAX_CONNS" envDefault:"10"`
	// AutoMigrate applies pending migrations at startup. Turn it off when a
	// separate, more privileged role runs "server migrate up" during deploys.
	AutoMigrate bool `env:"DATABASE_AUTO_MIGRATE" envDefault:"true"`

	LogLevel slog.Level `env:"LOG_LEVEL" envDefault:"info"`
	// LogFormat is "text" or "json". Unset means text in development and
	// JSON in production.
	LogFormat string `env:"LOG_FORMAT"`

	SessionLifetime time.Duration `env:"SESSION_LIFETIME" envDefault:"168h"`
	// SessionCookieSecure marks the session cookie HTTPS-only. Unset means
	// true in production and false in development.
	SessionCookieSecure *bool `env:"SESSION_COOKIE_SECURE"`

	// TrustedProxies lists the IPs or CIDR ranges of reverse proxies whose
	// X-Forwarded-For header may be believed. Leave it empty unless the app
	// really sits behind proxies you control.
	TrustedProxies    []netip.Prefix `env:"-"`
	RawTrustedProxies []string       `env:"TRUSTED_PROXIES" envSeparator:","`

	// TrustedOrigins lists extra origins (scheme://host[:port]) allowed to
	// send state-changing requests, on top of the app's own origin.
	TrustedOrigins []string `env:"TRUSTED_ORIGINS" envSeparator:","`

	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"15s"`
}

// Load reads the configuration from the process environment.
func Load() (Config, error) {
	return parse(env.Options{})
}

// LoadFrom reads the configuration from environ instead of the process
// environment. Tests use it to stay hermetic.
func LoadFrom(environ map[string]string) (Config, error) {
	if environ == nil {
		environ = map[string]string{}
	}
	return parse(env.Options{Environment: environ})
}

func parse(opts env.Options) (Config, error) {
	cfg, err := env.ParseAsWithOptions[Config](opts)
	if err != nil {
		return Config{}, fmt.Errorf("config: %w", err)
	}
	if err := cfg.finalize(); err != nil {
		return Config{}, fmt.Errorf("config: %w", err)
	}
	return cfg, nil
}

// IsProduction reports whether the app runs with production hardening.
func (c Config) IsProduction() bool {
	return c.Env == EnvProduction
}

// CookieSecure reports whether cookies should be marked Secure.
func (c Config) CookieSecure() bool {
	if c.SessionCookieSecure != nil {
		return *c.SessionCookieSecure
	}
	return c.IsProduction()
}

// DatabaseTarget describes the database for logs without the credentials.
func (c Config) DatabaseTarget() string {
	u, err := url.Parse(c.DatabaseURL)
	if err != nil || u.Host == "" {
		return "configured database"
	}
	return u.Host + u.Path
}

func (c *Config) finalize() error {
	var errs []error

	c.Env = strings.ToLower(strings.TrimSpace(c.Env))
	if c.Env != EnvDevelopment && c.Env != EnvProduction {
		errs = append(errs, fmt.Errorf("APP_ENV must be %q or %q, got %q", EnvDevelopment, EnvProduction, c.Env))
	}

	if c.HTTPAddr == "" {
		c.HTTPAddr = ":8080"
		if c.Port != "" {
			c.HTTPAddr = net.JoinHostPort("", c.Port)
		}
	}

	c.LogFormat = strings.ToLower(strings.TrimSpace(c.LogFormat))
	switch c.LogFormat {
	case "":
		c.LogFormat = "text"
		if c.IsProduction() {
			c.LogFormat = "json"
		}
	case "text", "json":
	default:
		errs = append(errs, fmt.Errorf("LOG_FORMAT must be \"text\" or \"json\", got %q", c.LogFormat))
	}

	if c.DatabaseMaxConns < 1 {
		errs = append(errs, fmt.Errorf("DATABASE_MAX_CONNS must be at least 1, got %d", c.DatabaseMaxConns))
	}
	if c.SessionLifetime <= 0 {
		errs = append(errs, fmt.Errorf("SESSION_LIFETIME must be positive, got %s", c.SessionLifetime))
	}
	if c.ShutdownTimeout <= 0 {
		errs = append(errs, fmt.Errorf("SHUTDOWN_TIMEOUT must be positive, got %s", c.ShutdownTimeout))
	}

	c.TrustedProxies = c.TrustedProxies[:0]
	for _, raw := range c.RawTrustedProxies {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		prefix, err := parsePrefix(raw)
		if err != nil {
			errs = append(errs, fmt.Errorf("TRUSTED_PROXIES: %q is not an IP address or CIDR range", raw))
			continue
		}
		c.TrustedProxies = append(c.TrustedProxies, prefix)
	}

	origins := c.TrustedOrigins[:0]
	for _, origin := range c.TrustedOrigins {
		if origin = strings.TrimSpace(origin); origin != "" {
			origins = append(origins, origin)
		}
	}
	c.TrustedOrigins = origins

	return errors.Join(errs...)
}

// parsePrefix accepts either a CIDR range or a single address, which it
// widens to a one-address range.
func parsePrefix(s string) (netip.Prefix, error) {
	if prefix, err := netip.ParsePrefix(s); err == nil {
		return prefix.Masked(), nil
	}

	addr, err := netip.ParseAddr(s)
	if err != nil {
		return netip.Prefix{}, err
	}
	addr = addr.Unmap()
	return netip.PrefixFrom(addr, addr.BitLen()), nil
}
