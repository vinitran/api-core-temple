package userstore_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stephenafamo/bob"

	"api-core/internal/datastore"
	"api-core/internal/datastore/userstore"
	"api-core/internal/domain"
	"api-core/pkg/errorx"
)

// Run with a migrated database:
//
//	TEST_DATABASE_DSN=postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable go test ./internal/datastore/...
func newPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN not set")
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)

	return pool
}

// errRollback aborts the transaction so tests leave no data behind.
var errRollback = errorx.New(errorx.Other, "rollback")

func TestUpsertGoogleUser(t *testing.T) {
	pool := newPool(t)
	ctx := context.Background()

	err := datastore.RunInTx(ctx, pool, func(ctx context.Context, exec bob.Executor) error {
		store := userstore.NewWithExecutor(exec)
		name := "Alice"
		first := time.Now().UTC().Add(-time.Hour).Truncate(time.Microsecond)

		created, err := store.UpsertGoogleUser(ctx, domain.GoogleAccount{
			GoogleID: "it-google-1", Email: "it-alice@example.com", Name: &name, VerifiedEmail: true, LoginAt: first,
		})
		if err != nil {
			t.Fatalf("first upsert: %v", err)
		}

		renamed := "Alice B."
		second := first.Add(time.Hour)
		updated, err := store.UpsertGoogleUser(ctx, domain.GoogleAccount{
			GoogleID: "it-google-1", Email: "it-alice@example.com", Name: &renamed, VerifiedEmail: true, LoginAt: second,
		})
		if err != nil {
			t.Fatalf("second upsert (repeat login) must succeed: %v", err)
		}

		if updated.ID != created.ID {
			t.Errorf("repeat login created a new user: %d != %d", updated.ID, created.ID)
		}
		if updated.Name == nil || *updated.Name != renamed {
			t.Errorf("name = %v, want %q", updated.Name, renamed)
		}
		if updated.LastLoginAt == nil || !updated.LastLoginAt.Equal(second) {
			t.Errorf("last_login_at = %v, want %v", updated.LastLoginAt, second)
		}
		if !updated.CreatedAt.Equal(created.CreatedAt) {
			t.Errorf("created_at changed on upsert")
		}

		got, err := store.GetByID(ctx, created.ID)
		if err != nil || got.GoogleID != "it-google-1" {
			t.Errorf("GetByID() = %+v, %v", got, err)
		}

		_, err = store.GetByID(ctx, -1)
		if errorx.KindOf(err) != errorx.NotExist {
			t.Errorf("GetByID(missing) kind = %v, want NotExist", errorx.KindOf(err))
		}

		_, err = store.UpsertGoogleUser(ctx, domain.GoogleAccount{
			GoogleID: "it-google-2", Email: "it-alice@example.com", LoginAt: second,
		})
		if errorx.KindOf(err) != errorx.Exist {
			t.Errorf("duplicate email kind = %v (%v), want Exist", errorx.KindOf(err), err)
		}

		return errRollback
	})
	if !errors.Is(err, errRollback) {
		t.Fatalf("RunInTx() = %v", err)
	}
}
