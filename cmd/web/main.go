// Package main provides the entry point for the Go web server application.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alexedwards/scs/pgxstore"
	"github.com/alexedwards/scs/v2"
	"github.com/dunamismax/go-web-server/internal/buildinfo"
	"github.com/dunamismax/go-web-server/internal/config"
	"github.com/dunamismax/go-web-server/internal/handler"
	"github.com/dunamismax/go-web-server/internal/middleware"
	"github.com/dunamismax/go-web-server/internal/store"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
)

//go:generate go install github.com/a-h/templ/cmd/templ@v0.3.1001
//go:generate go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.30.0
//go:generate sh -c "cd ../../ && templ generate"
//go:generate sh -c "cd ../../ && sqlc generate"

func main() {
	if handleVersionCommand(os.Args[1:]) {
		return
	}

	// Load configuration
	cfg := config.New()

	// Setup structured logging
	var logger *slog.Logger
	if cfg.App.LogFormat == "json" {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: cfg.GetLogLevel(),
		}))
	} else {
		logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: cfg.GetLogLevel(),
		}))
	}

	slog.SetDefault(logger)

	slog.Info("Starting Go Web Server",
		"version", buildinfo.Version,
		"environment", cfg.App.Environment,
		"go_version", "1.26+",
		"port", cfg.Server.Port,
		"debug", cfg.App.Debug)

	// Create context for database operations
	ctx := context.Background()

	// Initialize database store with configurable pool settings
	poolConfig := store.PoolConfig{
		MaxConns:        cfg.Database.MaxConnections,
		MinConns:        cfg.Database.MinConnections,
		MaxConnLifetime: cfg.Database.MaxConnLifetime,
		MaxConnIdleTime: cfg.Database.MaxConnIdleTime,
	}

	store, err := store.NewStoreWithConfig(ctx, cfg.Database.URL, poolConfig)
	if err != nil {
		slog.Error("failed to connect to database", "error", err, "database_target", databaseTarget(cfg.Database.URL))
		return
	}

	defer func() {
		store.Close()
		slog.Info("Database connection pool closed")
	}()

	// Note: Database migrations are now managed by Atlas CLI
	// Run: atlas migrate apply --url $DATABASE_URL --dir file://migrations

	// Initialize schema for local bootstrap only when enabled.
	if cfg.Database.RunMigrations {
		if err := store.InitSchema(ctx); err != nil {
			slog.Error("failed to initialize schema", "error", err)
			return
		}
	} else {
		slog.Info("Skipping schema bootstrap", "database_target", databaseTarget(cfg.Database.URL))
	}

	// Create Echo instance
	e := echo.New()
	e.HideBanner = true
	e.Debug = cfg.App.Debug

	ipExtractor, err := configureIPExtractor(cfg.Security.TrustedProxies)
	if err != nil {
		slog.Error("failed to configure trusted proxies", "error", err)
		return
	}
	e.IPExtractor = ipExtractor

	// Configure custom error handler
	e.HTTPErrorHandler = middleware.ErrorHandler

	// Set custom 404 and 405 handlers
	e.RouteNotFound("/*", middleware.NotFoundHandler)
	e.Add("*", "/*", middleware.MethodNotAllowedHandler)

	// Configure timeouts
	e.Server.ReadTimeout = cfg.Server.ReadTimeout
	e.Server.WriteTimeout = cfg.Server.WriteTimeout

	// Middleware stack (order matters)

	// Custom recovery middleware (should be first)
	e.Use(middleware.RecoveryMiddleware())

	// Security headers middleware
	e.Use(middleware.SecurityHeadersMiddleware())

	// Input sanitization middleware
	e.Use(middleware.Sanitize())

	// CSRF protection middleware
	e.Use(middleware.CSRF())

	// Validation error middleware
	e.Use(middleware.ValidationErrorMiddleware())

	// Timeout error middleware
	e.Use(middleware.TimeoutErrorHandler())

	// Request ID middleware for tracing
	e.Use(echomiddleware.RequestID())

	// Structured logging middleware
	e.Use(echomiddleware.RequestLoggerWithConfig(echomiddleware.RequestLoggerConfig{
		LogStatus:    true,
		LogURI:       true,
		LogError:     true,
		LogMethod:    true,
		LogLatency:   true,
		LogRemoteIP:  true,
		LogUserAgent: cfg.App.Debug,
		LogValuesFunc: func(_ echo.Context, v echomiddleware.RequestLoggerValues) error {
			if v.Error == nil {
				slog.Info("request",
					"method", v.Method,
					"uri", v.URI,
					"status", v.Status,
					"latency", v.Latency.String(),
					"remote_ip", v.RemoteIP,
					"request_id", v.RequestID)
			} else {
				slog.Error("request error",
					"method", v.Method,
					"uri", v.URI,
					"status", v.Status,
					"latency", v.Latency.String(),
					"remote_ip", v.RemoteIP,
					"request_id", v.RequestID,
					"error", v.Error)
			}

			return nil
		},
	}))

	// Security middleware
	e.Use(echomiddleware.SecureWithConfig(echomiddleware.SecureConfig{
		XSSProtection:         "1; mode=block",
		ContentTypeNosniff:    "nosniff",
		XFrameOptions:         "DENY",
		HSTSMaxAge:            31536000,
		ContentSecurityPolicy: "default-src 'self'; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com https://fonts.gstatic.com; script-src 'self' 'unsafe-inline' 'unsafe-eval'; img-src 'self' data:; connect-src 'self'; font-src 'self' https://fonts.googleapis.com https://fonts.gstatic.com;",
	}))

	// CORS middleware
	if cfg.Security.EnableCORS {
		e.Use(echomiddleware.CORSWithConfig(echomiddleware.CORSConfig{
			AllowOrigins: cfg.Security.AllowedOrigins,
			AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete},
			AllowHeaders: []string{"*"},
			MaxAge:       86400,
		}))
	}

	// Rate limiting
	e.Use(echomiddleware.RateLimiterWithConfig(echomiddleware.RateLimiterConfig{
		Store: echomiddleware.NewRateLimiterMemoryStore(20),
		IdentifierExtractor: func(c echo.Context) (string, error) {
			return c.RealIP(), nil
		},
		ErrorHandler: func(_ echo.Context, err error) error {
			return middleware.ErrTooManyRequests.WithInternal(err)
		},
	}))

	// Request deadline middleware.
	// We avoid Echo's Timeout middleware here because it swaps the response writer
	// and breaks templ rendering for full-page HTML responses.
	e.Use(middleware.RequestTimeout(cfg.Server.ReadTimeout))

	// Add environment to context for error handling
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("environment", cfg.App.Environment)

			return next(c)
		}
	})

	// Initialize session manager
	sessionManager := scs.New()
	sessionManager.Store = pgxstore.New(store.DB())
	sessionManager.Lifetime = 24 * time.Hour
	sessionManager.Cookie.Name = cfg.Auth.CookieName
	sessionManager.Cookie.HttpOnly = true
	sessionManager.Cookie.Secure = cfg.Auth.CookieSecure
	sessionManager.Cookie.SameSite = http.SameSiteStrictMode

	// Initialize session-based authentication service
	authService := middleware.NewSessionAuthService(sessionManager)

	// Add session middleware to Echo
	e.Use(authService.SessionMiddleware())

	// Initialize handlers and register routes
	handlers := handler.NewHandlers(store, authService)
	if err := handler.RegisterRoutes(e, handlers); err != nil {
		slog.Error("failed to register routes", "error", err)
		return
	}

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Start server in goroutine
	go func() {
		address := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
		slog.Info("Server starting", "address", address)

		if err := e.Start(address); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("failed to start server", "error", err)
			return
		}
	}()

	// Wait for interrupt signal
	<-ctx.Done()

	slog.Info("Shutting down server...")

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := e.Shutdown(shutdownCtx); err != nil {
		slog.Error("failed to shutdown server gracefully", "error", err)
		return
	}

	slog.Info("Server shutdown complete")
}

func handleVersionCommand(args []string) bool {
	if len(args) == 0 {
		return false
	}

	switch args[0] {
	case "--version", "version":
		fmt.Printf("server %s\n", buildinfo.Version)
		return true
	default:
		return false
	}
}

func configureIPExtractor(trustedProxies []string) (echo.IPExtractor, error) {
	if len(trustedProxies) == 0 {
		return echo.ExtractIPDirect(), nil
	}

	options := []echo.TrustOption{
		echo.TrustLoopback(false),
		echo.TrustLinkLocal(false),
		echo.TrustPrivateNet(false),
	}

	for _, proxy := range trustedProxies {
		if _, network, err := net.ParseCIDR(proxy); err == nil {
			options = append(options, echo.TrustIPRange(network))
			continue
		}

		ip := net.ParseIP(proxy)
		if ip == nil {
			return nil, fmt.Errorf("invalid trusted proxy %q", proxy)
		}

		maskBits := 32
		if ip.To4() == nil {
			maskBits = 128
		}

		options = append(options, echo.TrustIPRange(&net.IPNet{
			IP:   ip,
			Mask: net.CIDRMask(maskBits, maskBits),
		}))
	}

	xffExtractor := echo.ExtractIPFromXFFHeader(options...)
	realIPExtractor := echo.ExtractIPFromRealIPHeader(options...)
	directExtractor := echo.ExtractIPDirect()

	return func(r *http.Request) string {
		directIP := directExtractor(r)
		if ip := xffExtractor(r); ip != directIP {
			return ip
		}
		return realIPExtractor(r)
	}, nil
}

func databaseTarget(databaseURL string) string {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return "configured-database"
	}

	return fmt.Sprintf("%s:%d/%s", cfg.ConnConfig.Host, cfg.ConnConfig.Port, cfg.ConnConfig.Database)
}
