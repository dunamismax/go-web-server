package store_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/dunamismax/go-web-server/internal/store"
	"github.com/dunamismax/go-web-server/internal/store/storetest"
)

func TestMigrationsApplyRollBackAndReapply(t *testing.T) {
	t.Parallel()

	pool := storetest.NewPool(t)
	ctx := t.Context()

	migrator, err := store.NewMigrator(ctx, pool)
	if err != nil {
		t.Fatalf("NewMigrator: %v", err)
	}
	t.Cleanup(func() { _ = migrator.Close() })

	applied, err := migrator.Up(ctx)
	if err != nil {
		t.Fatalf("Up: %v", err)
	}
	if len(applied) == 0 {
		t.Fatal("Up applied no migrations on an empty schema")
	}

	// A second run finds nothing to do.
	if again, err := migrator.Up(ctx); err != nil || len(again) != 0 {
		t.Fatalf("second Up = %v, %v; want no migrations and no error", again, err)
	}

	// Every migration must have a working Down.
	for range applied {
		if _, ok, err := migrator.Down(ctx); err != nil || !ok {
			t.Fatalf("Down = %v, %v; want a rolled-back migration", ok, err)
		}
	}
	if _, ok, err := migrator.Down(ctx); err != nil || ok {
		t.Fatalf("Down on an empty schema = %v, %v; want false, nil", ok, err)
	}

	status, err := migrator.Status(ctx)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	for _, m := range status {
		if m.Applied {
			t.Errorf("migration %s still applied after rolling everything back", m.Name)
		}
	}

	if reapplied, err := migrator.Up(ctx); err != nil || len(reapplied) != len(applied) {
		t.Fatalf("re-Up = %d migrations, %v; want %d, nil", len(reapplied), err, len(applied))
	}
}

func TestRegisterUserMakesOnlyTheFirstAccountAdmin(t *testing.T) {
	t.Parallel()

	db := storetest.New(t)
	ctx := t.Context()

	first, err := db.RegisterUser(ctx, store.RegisterUserParams{Email: "first@example.com", Name: "First", PasswordHash: "x"})
	if err != nil {
		t.Fatalf("RegisterUser(first): %v", err)
	}
	if first.Role != store.RoleAdmin || !first.IsAdmin() {
		t.Errorf("first user role = %q; want admin", first.Role)
	}
	if !first.IsActive {
		t.Error("first user is inactive; want active by default")
	}

	second, err := db.RegisterUser(ctx, store.RegisterUserParams{Email: "second@example.com", Name: "Second", PasswordHash: "x"})
	if err != nil {
		t.Fatalf("RegisterUser(second): %v", err)
	}
	if second.Role != store.RoleMember {
		t.Errorf("second user role = %q; want member", second.Role)
	}
}

func TestEmailConstraints(t *testing.T) {
	t.Parallel()

	db := storetest.New(t)
	ctx := t.Context()

	params := store.CreateUserParams{Email: "ada@example.com", Name: "Ada", PasswordHash: "x", Role: store.RoleMember}
	if _, err := db.CreateUser(ctx, params); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	_, err := db.CreateUser(ctx, params)
	if !store.IsUniqueViolation(err, store.ConstraintUsersEmail) {
		t.Errorf("duplicate email error = %v; want a %s unique violation", err, store.ConstraintUsersEmail)
	}
	if store.IsUniqueViolation(errors.New("boom"), store.ConstraintUsersEmail) {
		t.Error("IsUniqueViolation matched an unrelated error")
	}

	// The schema refuses mixed-case emails, so a code path that forgets to
	// normalize fails loudly instead of creating a second "Ada".
	params.Email = "Ada@Example.com"
	if _, err := db.CreateUser(ctx, params); err == nil {
		t.Error("CreateUser accepted a mixed-case email; want a check violation")
	}

	params.Email, params.Role = "eve@example.com", store.Role("superuser")
	if _, err := db.CreateUser(ctx, params); err == nil {
		t.Error("CreateUser accepted an unknown role; want a check violation")
	}
}

func TestListAndCountUsersSearchAndPaginate(t *testing.T) {
	t.Parallel()

	db := storetest.New(t)
	ctx := t.Context()

	for i := range 5 {
		_, err := db.CreateUser(ctx, store.CreateUserParams{
			Email:        fmt.Sprintf("user%d@example.com", i),
			Name:         fmt.Sprintf("User %d", i),
			PasswordHash: "x",
			Role:         store.RoleMember,
		})
		if err != nil {
			t.Fatalf("CreateUser: %v", err)
		}
	}
	if _, err := db.CreateUser(ctx, store.CreateUserParams{
		Email: "grace@navy.example", Name: "Grace Hopper", PasswordHash: "x", Role: store.RoleAdmin,
	}); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	tests := []struct {
		search string
		want   int64
	}{
		{"", 6},
		{"grace", 1},        // name, case-insensitive
		{"HOPPER", 1},       // name, case-insensitive
		{"navy.example", 1}, // email
		{"user", 5},
		{"100%", 0}, // LIKE wildcards are plain text
		{"_", 0},
	}
	for _, tt := range tests {
		count, err := db.CountUsers(ctx, tt.search)
		if err != nil {
			t.Fatalf("CountUsers(%q): %v", tt.search, err)
		}
		if count != tt.want {
			t.Errorf("CountUsers(%q) = %d; want %d", tt.search, count, tt.want)
		}
	}

	page1, err := db.ListUsers(ctx, store.ListUsersParams{Search: "user", PageSize: 2, PageOffset: 0})
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	page3, err := db.ListUsers(ctx, store.ListUsersParams{Search: "user", PageSize: 2, PageOffset: 4})
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if len(page1) != 2 || len(page3) != 1 {
		t.Fatalf("page sizes = %d, %d; want 2, 1", len(page1), len(page3))
	}
	// Newest first.
	if page1[0].Name != "User 4" || page3[0].Name != "User 0" {
		t.Errorf("order = %q … %q; want User 4 … User 0", page1[0].Name, page3[0].Name)
	}

	none, err := db.ListUsers(ctx, store.ListUsersParams{Search: "nobody", PageSize: 10})
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if none == nil || len(none) != 0 {
		t.Errorf("ListUsers with no matches = %v; want an empty, non-nil slice", none)
	}
}

func TestUpdateUserPasswordMovesPasswordChangedAt(t *testing.T) {
	t.Parallel()

	db := storetest.New(t)
	ctx := t.Context()

	user, err := db.CreateUser(ctx, store.CreateUserParams{Email: "a@example.com", Name: "A", PasswordHash: "old", Role: store.RoleMember})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	// Upgrading the hash format is not a password change.
	if err := db.UpgradeUserPasswordHash(ctx, store.UpgradeUserPasswordHashParams{ID: user.ID, PasswordHash: "rehashed"}); err != nil {
		t.Fatalf("UpgradeUserPasswordHash: %v", err)
	}
	upgraded, err := db.GetUser(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	if !upgraded.PasswordChangedAt.Equal(user.PasswordChangedAt) || upgraded.PasswordHash != "rehashed" {
		t.Errorf("after upgrade: hash %q, changed_at moved = %v; want rehashed, false",
			upgraded.PasswordHash, !upgraded.PasswordChangedAt.Equal(user.PasswordChangedAt))
	}

	changed, err := db.UpdateUserPassword(ctx, store.UpdateUserPasswordParams{ID: user.ID, PasswordHash: "new"})
	if err != nil {
		t.Fatalf("UpdateUserPassword: %v", err)
	}
	if !changed.PasswordChangedAt.After(user.PasswordChangedAt) {
		t.Error("UpdateUserPassword did not advance password_changed_at")
	}

	if _, err := db.GetUser(ctx, user.ID+1000); !store.IsNotFound(err) {
		t.Errorf("GetUser(missing) error = %v; want not found", err)
	}
}

func TestInTxRollsBackOnError(t *testing.T) {
	t.Parallel()

	db := storetest.New(t)
	ctx := t.Context()

	boom := errors.New("boom")
	err := db.InTx(ctx, func(q *store.Queries) error {
		if _, err := q.CreateUser(ctx, store.CreateUserParams{Email: "tx@example.com", Name: "Tx", PasswordHash: "x", Role: store.RoleMember}); err != nil {
			return err
		}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("InTx error = %v; want boom", err)
	}

	if count, err := db.CountUsers(ctx, ""); err != nil || count != 0 {
		t.Errorf("CountUsers after rollback = %d, %v; want 0, nil", count, err)
	}
}
