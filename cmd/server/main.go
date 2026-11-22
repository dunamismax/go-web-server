// Command server runs the web application.
//
//	server                 start the HTTP server (the default)
//	server migrate up      apply pending database migrations
//	server migrate down    roll back the most recent migration
//	server migrate status  list migrations and whether they are applied
//	server healthcheck     exit 0 if a running server answers /healthz
//	server version         print the build version
//
// Configuration comes from environment variables; see .env.example.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dunamismax/go-web-server/internal/assets"
	"github.com/dunamismax/go-web-server/internal/config"
	"github.com/dunamismax/go-web-server/internal/store"
	"github.com/dunamismax/go-web-server/internal/version"
	"github.com/dunamismax/go-web-server/internal/web"
)

func main() {
	os.Exit(mainExitCode())
}

// mainExitCode exists so deferred cleanup runs before the process exits.
func mainExitCode() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	return 0
}

func run(ctx context.Context, args []string, stdout io.Writer) error {
	command := "serve"
	if len(args) > 0 {
		command, args = args[0], args[1:]
	}

	switch command {
	case "version", "--version", "-v":
		_, err := fmt.Fprintln(stdout, version.String())
		return err
	case "help", "--help", "-h":
		_, err := fmt.Fprintln(stdout, usage)
		return err
	case "serve", "migrate", "healthcheck":
	default:
		return fmt.Errorf("unknown command %q\n\n%s", command, usage)
	}

	// A .env file is a development convenience. Real environment variables
	// always take precedence over it, and production does not need one.
	if err := config.LoadDotEnv(".env"); err != nil {
		return err
	}

	// The health check needs only the listen address, so it works even when
	// the rest of the configuration is not visible to the probing process.
	if command == "healthcheck" {
		return healthcheck(ctx)
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if command == "migrate" {
		return migrate(ctx, cfg, args, stdout)
	}
	return serve(ctx, cfg, stdout)
}

const usage = `Usage: server [command]

Commands:
  serve            Start the HTTP server (default)
  migrate up       Apply pending database migrations
  migrate down     Roll back the most recent migration
  migrate status   List migrations and whether they are applied
  healthcheck      Exit 0 if a running server answers /healthz
  version          Print the build version`

func serve(ctx context.Context, cfg config.Config, stdout io.Writer) error {
	logger := newLogger(cfg, stdout)
	slog.SetDefault(logger)

	logger.Info("starting",
		"version", version.String(),
		"env", cfg.Env,
		"addr", cfg.HTTPAddr,
		"database", cfg.DatabaseTarget())

	db, err := store.Open(ctx, cfg.DatabaseURL, cfg.DatabaseMaxConns)
	if err != nil {
		return err
	}
	defer db.Close()

	if cfg.AutoMigrate {
		applied, err := store.Migrate(ctx, db.Pool())
		if err != nil {
			return err
		}
		for _, m := range applied {
			logger.Info("applied migration", "version", m.Version, "name", m.Name)
		}
	}

	if !cfg.IsProduction() && assets.UseLiveFiles() {
		logger.Debug("serving static files from disk")
	}

	app, err := web.New(web.Options{Config: cfg, Logger: logger, Store: db})
	if err != nil {
		return err
	}
	defer app.Close()

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           app.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       2 * time.Minute,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelWarn),
	}

	listener, err := new(net.ListenConfig).Listen(ctx, "tcp", cfg.HTTPAddr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", cfg.HTTPAddr, err)
	}
	logger.Info("listening", "addr", listener.Addr().String())

	serveErr := make(chan error, 1)
	go func() { serveErr <- server.Serve(listener) }()

	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
	}

	// Stop accepting connections and let in-flight requests finish.
	logger.Info("shutting down", "timeout", cfg.ShutdownTimeout)
	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cfg.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}
	if err := <-serveErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	logger.Info("stopped")
	return nil
}

func migrate(ctx context.Context, cfg config.Config, args []string, stdout io.Writer) error {
	action := "up"
	if len(args) > 0 {
		action = args[0]
	}

	db, err := store.Open(ctx, cfg.DatabaseURL, cfg.DatabaseMaxConns)
	if err != nil {
		return err
	}
	defer db.Close()

	migrator, err := store.NewMigrator(ctx, db.Pool())
	if err != nil {
		return err
	}
	defer migrator.Close() //nolint:errcheck // nothing useful to do with a close error on exit

	switch action {
	case "up":
		applied, err := migrator.Up(ctx)
		if err != nil {
			return err
		}
		if len(applied) == 0 {
			fmt.Fprintln(stdout, "No pending migrations.")
		}
		for _, m := range applied {
			fmt.Fprintf(stdout, "Applied %s\n", m.Name)
		}
	case "down":
		m, ok, err := migrator.Down(ctx)
		if err != nil {
			return err
		}
		if !ok {
			fmt.Fprintln(stdout, "Nothing to roll back.")
			break
		}
		fmt.Fprintf(stdout, "Rolled back %s\n", m.Name)
	case "status":
		migrations, err := migrator.Status(ctx)
		if err != nil {
			return err
		}
		for _, m := range migrations {
			state := "pending"
			if m.Applied {
				state = "applied"
			}
			fmt.Fprintf(stdout, "%-8s %s\n", state, m.Name)
		}
	default:
		return fmt.Errorf("unknown migrate action %q (want up, down or status)", action)
	}
	return nil
}

// healthcheck asks the local server for /healthz. Container images without a
// shell or curl use it as their HEALTHCHECK command.
func healthcheck(ctx context.Context) error {
	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
		if port := os.Getenv("PORT"); port != "" {
			addr = ":" + port
		}
	}
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("parse HTTP_ADDR %q: %w", addr, err)
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	target := "http://" + net.JoinHostPort(host, port) + "/healthz"
	// The target is this process's own listen address, taken from its own
	// configuration, never from a request.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, http.NoBody) //nolint:gosec // see above
	if err != nil {
		return err
	}
	// A one-shot probe has no use for a pooled connection, and leaving one
	// open would make the server wait on it during a graceful shutdown.
	client := &http.Client{Transport: &http.Transport{DisableKeepAlives: true}}
	resp, err := client.Do(req) //nolint:gosec // see above
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck // response body is not read

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s returned %s", target, resp.Status)
	}
	return nil
}

func newLogger(cfg config.Config, w io.Writer) *slog.Logger {
	opts := &slog.HandlerOptions{Level: cfg.LogLevel}
	if cfg.LogFormat == "json" {
		return slog.New(slog.NewJSONHandler(w, opts))
	}
	return slog.New(slog.NewTextHandler(w, opts))
}
