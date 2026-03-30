// Package store provides database access and query execution functionality.
package store

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed schema.sql
var canonicalSchemaSQL string

const legacyBootstrapReconciliationSQL = `
	-- Legacy reconciliation for older local databases created before password hashes were required.
	ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash TEXT;
	UPDATE users
	SET password_hash = 'account-disabled-no-password',
		is_active = false,
		updated_at = CURRENT_TIMESTAMP
	WHERE password_hash IS NULL;
	ALTER TABLE users ALTER COLUMN password_hash SET NOT NULL;
`

// Store provides all functions to execute db queries.
type Store struct {
	*Queries // Embed sqlc-generated queries

	db *pgxpool.Pool
}

// PoolConfig holds database connection pool configuration.
type PoolConfig struct {
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
}

// NewStore creates a new store instance with database connection pool.
func NewStore(ctx context.Context, databaseURL string) (*Store, error) {
	// Default pool configuration
	poolConfig := PoolConfig{
		MaxConns:        25,
		MinConns:        5,
		MaxConnLifetime: 0,
		MaxConnIdleTime: 0,
	}

	return NewStoreWithConfig(ctx, databaseURL, poolConfig)
}

// NewStoreWithConfig creates a new store instance with custom pool configuration.
func NewStoreWithConfig(ctx context.Context, databaseURL string, poolConfig PoolConfig) (*Store, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	// Set connection pool settings from config
	config.MaxConns = poolConfig.MaxConns
	config.MinConns = poolConfig.MinConns
	config.MaxConnLifetime = poolConfig.MaxConnLifetime
	config.MaxConnIdleTime = poolConfig.MaxConnIdleTime

	db, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := db.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Store{
		db:      db,
		Queries: New(db),
	}, nil
}

// NewStoreWithDB creates a new store instance with an existing database pool.
func NewStoreWithDB(db *pgxpool.Pool) *Store {
	return &Store{
		db:      db,
		Queries: New(db),
	}
}

// Close closes the database connection pool.
func (s *Store) Close() {
	s.db.Close()
}

// DB returns the underlying database connection pool for advanced operations.
func (s *Store) DB() *pgxpool.Pool {
	return s.db
}

// BeginTx starts a new transaction.
func (s *Store) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return s.db.Begin(ctx)
}

// WithTx returns a new Store that will execute queries within the given transaction.
func (s *Store) WithTx(tx pgx.Tx) *Store {
	return &Store{
		db:      s.db,
		Queries: s.Queries.WithTx(tx),
	}
}

// DeactivateUserChecked deactivates a user and reports whether a row was updated.
func (s *Store) DeactivateUserChecked(ctx context.Context, id int64) (bool, error) {
	tag, err := s.db.Exec(ctx, deactivateUser, id)
	if err != nil {
		return false, err
	}

	return tag.RowsAffected() > 0, nil
}

// DeleteUserChecked deletes a user and reports whether a row was removed.
func (s *Store) DeleteUserChecked(ctx context.Context, id int64) (bool, error) {
	tag, err := s.db.Exec(ctx, deleteUser, id)
	if err != nil {
		return false, err
	}

	return tag.RowsAffected() > 0, nil
}

// InitSchema initializes the database schema using the canonical schema.sql file.
// This is kept here for compatibility, but migrations are preferred.
func (s *Store) InitSchema(ctx context.Context) error {
	bootstrapSQL := canonicalSchemaSQL + "\n" + legacyBootstrapReconciliationSQL

	if _, err := s.db.Exec(ctx, bootstrapSQL); err != nil {
		return fmt.Errorf("failed to initialize schema: %w", err)
	}

	return nil
}
