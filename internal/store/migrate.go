package store

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"hash/fnv"
	"io/fs"
	"path"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// Migration describes one schema migration and whether it has been applied.
type Migration struct {
	Version int64
	Name    string
	Applied bool
}

// Migrator applies the embedded goose migrations. It holds its own
// database/sql handle on top of the pool, so Close it when finished.
type Migrator struct {
	provider *goose.Provider
}

// NewMigrator prepares the embedded migrations to run against pool. A
// PostgreSQL advisory lock serializes concurrent runs, so several instances
// can start at once without racing each other.
func NewMigrator(ctx context.Context, pool *pgxpool.Pool) (*Migrator, error) {
	files, err := fs.Sub(migrationFiles, "migrations")
	if err != nil {
		return nil, fmt.Errorf("locate embedded migrations: %w", err)
	}

	lockID, err := migrationLockID(ctx, pool)
	if err != nil {
		return nil, err
	}
	// Retry once a second for up to two minutes while another instance
	// holds the lock.
	locker, err := lock.NewPostgresSessionLocker(lock.WithLockID(lockID), lock.WithLockTimeout(1, 120))
	if err != nil {
		return nil, fmt.Errorf("create migration lock: %w", err)
	}

	provider, err := goose.NewProvider(
		goose.DialectPostgres,
		stdlib.OpenDBFromPool(pool),
		files,
		goose.WithSessionLocker(locker),
	)
	if err != nil {
		return nil, fmt.Errorf("load migrations: %w", err)
	}

	return &Migrator{provider: provider}, nil
}

// migrationLockID returns the advisory lock key for the schema the pool
// writes to. Advisory locks are database-wide, so a per-schema key lets
// independent schemas in one database (the test suite creates many) migrate
// at the same time. The default "public" schema keeps goose's standard key,
// which the goose CLI also uses.
func migrationLockID(ctx context.Context, pool *pgxpool.Pool) (int64, error) {
	var schema string
	if err := pool.QueryRow(ctx, "SELECT coalesce(current_schema(), '')").Scan(&schema); err != nil {
		return 0, fmt.Errorf("read current schema: %w", err)
	}
	if schema == "" || schema == "public" {
		return lock.DefaultLockID, nil
	}

	h := fnv.New64a()
	_, _ = h.Write([]byte(schema))
	return int64(h.Sum64()), nil //nolint:gosec // any 64-bit pattern is a valid lock key
}

// Close releases the migrator's database handle. The pool stays open.
func (m *Migrator) Close() error {
	return m.provider.Close()
}

// Up applies every pending migration and returns the ones it ran.
func (m *Migrator) Up(ctx context.Context) ([]Migration, error) {
	results, err := m.provider.Up(ctx)
	if err != nil {
		return nil, fmt.Errorf("apply migrations: %w", err)
	}

	applied := make([]Migration, 0, len(results))
	for _, r := range results {
		applied = append(applied, Migration{
			Version: r.Source.Version,
			Name:    migrationName(r.Source.Path),
			Applied: true,
		})
	}
	return applied, nil
}

// Down rolls back the most recently applied migration. It returns false when
// there was nothing to roll back.
func (m *Migrator) Down(ctx context.Context) (Migration, bool, error) {
	result, err := m.provider.Down(ctx)
	if errors.Is(err, goose.ErrNoNextVersion) {
		return Migration{}, false, nil
	}
	if err != nil {
		return Migration{}, false, fmt.Errorf("roll back migration: %w", err)
	}

	return Migration{
		Version: result.Source.Version,
		Name:    migrationName(result.Source.Path),
	}, true, nil
}

// Status lists every known migration in version order.
func (m *Migrator) Status(ctx context.Context) ([]Migration, error) {
	statuses, err := m.provider.Status(ctx)
	if err != nil {
		return nil, fmt.Errorf("read migration status: %w", err)
	}

	migrations := make([]Migration, 0, len(statuses))
	for _, s := range statuses {
		migrations = append(migrations, Migration{
			Version: s.Source.Version,
			Name:    migrationName(s.Source.Path),
			Applied: s.State == goose.StateApplied,
		})
	}
	return migrations, nil
}

// Migrate is the one-call form used at startup: apply everything pending.
func Migrate(ctx context.Context, pool *pgxpool.Pool) ([]Migration, error) {
	m, err := NewMigrator(ctx, pool)
	if err != nil {
		return nil, err
	}
	applied, err := m.Up(ctx)
	return applied, errors.Join(err, m.Close())
}

func migrationName(file string) string {
	return path.Base(file)
}
