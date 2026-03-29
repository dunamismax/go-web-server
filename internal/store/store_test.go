package store

import (
	"strings"
	"testing"
)

func TestCanonicalSchemaBootstrapUsesCurrentSchemaFile(t *testing.T) {
	t.Parallel()

	checks := []string{
		"CREATE TABLE IF NOT EXISTS users",
		"password_hash TEXT NOT NULL",
		"CREATE TABLE IF NOT EXISTS sessions",
		"CREATE INDEX IF NOT EXISTS idx_sessions_expiry ON sessions(expiry)",
	}

	for _, check := range checks {
		if !strings.Contains(canonicalSchemaSQL, check) {
			t.Fatalf("canonicalSchemaSQL missing %q", check)
		}
	}
}

func TestLegacyBootstrapReconciliationStillProtectsPasswordInvariant(t *testing.T) {
	t.Parallel()

	checks := []string{
		"ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash TEXT",
		"WHERE password_hash IS NULL",
		"ALTER TABLE users ALTER COLUMN password_hash SET NOT NULL",
	}

	for _, check := range checks {
		if !strings.Contains(legacyBootstrapReconciliationSQL, check) {
			t.Fatalf("legacyBootstrapReconciliationSQL missing %q", check)
		}
	}
}
