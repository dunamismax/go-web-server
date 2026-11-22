// Package storetest gives tests a real, isolated PostgreSQL database.
//
// Each call creates a uniquely named schema inside the database that
// TEST_DATABASE_URL points at and drops it when the test ends. Tests can
// therefore run in parallel, and can share a server with your development
// data without touching it.
package storetest

import (
	"context"
	"crypto/rand"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dunamismax/go-web-server/internal/store"
)

// EnvVar names the environment variable holding the test database URL.
const EnvVar = "TEST_DATABASE_URL"

// New returns a migrated Store on a throwaway schema. It skips the test when
// TEST_DATABASE_URL is unset, so "go test ./..." works without a database.
func New(t testing.TB) *store.Store {
	t.Helper()

	pool := NewPool(t)
	if _, err := store.Migrate(t.Context(), pool); err != nil {
		t.Fatalf("storetest: migrate: %v", err)
	}
	return store.FromPool(pool)
}

// NewPool is New without the migrations, for tests of the migrations
// themselves. The pool is closed when the test ends.
func NewPool(t testing.TB) *pgxpool.Pool {
	t.Helper()

	cfg, err := pgxpool.ParseConfig(NewURL(t))
	if err != nil {
		t.Fatalf("storetest: parse database URL: %v", err)
	}
	cfg.MaxConns = 4

	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("storetest: create pool: %v", err)
	}
	// Cleanups run last-in, first-out: the pool closes before NewURL's
	// cleanup drops the schema.
	t.Cleanup(pool.Close)

	return pool
}

// NewURL creates an empty throwaway schema and returns a connection string
// whose connections see only that schema. Use it for code that opens its own
// connections, such as the server's entry point. The schema is dropped when
// the test ends.
func NewURL(t testing.TB) string {
	t.Helper()

	databaseURL := os.Getenv(EnvVar)
	if databaseURL == "" {
		t.Skipf("%s is not set; skipping test that needs PostgreSQL (run \"make test\")", EnvVar)
	}

	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()

	schema := "test_" + strings.ToLower(rand.Text())
	quoted := pgx.Identifier{schema}.Sanitize()

	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("storetest: connect to %s: %v", EnvVar, err)
	}
	defer conn.Close(ctx) //nolint:errcheck // best-effort close of a setup connection

	if _, err := conn.Exec(ctx, "CREATE SCHEMA "+quoted); err != nil {
		t.Fatalf("storetest: create schema: %v", err)
	}

	t.Cleanup(func() {
		// t.Context is already canceled during cleanup.
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		conn, err := pgx.Connect(ctx, databaseURL)
		if err != nil {
			t.Logf("storetest: reconnect to drop schema %s: %v", schema, err)
			return
		}
		defer conn.Close(ctx) //nolint:errcheck // best-effort close of a teardown connection

		if _, err := conn.Exec(ctx, "DROP SCHEMA "+quoted+" CASCADE"); err != nil {
			t.Logf("storetest: drop schema %s: %v", schema, err)
		}
	})

	// pgx sends unrecognized URL parameters to the server as session
	// settings, so every connection resolves unqualified names in the test
	// schema only.
	u, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatalf("storetest: parse %s: %v", EnvVar, err)
	}
	query := u.Query()
	query.Set("search_path", schema)
	u.RawQuery = query.Encode()

	return u.String()
}
