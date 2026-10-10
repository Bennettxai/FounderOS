// Package catalogdb is the test harness shared by the /personas and /skills
// ports: a throwaway Postgres database on the test server (pgtest.AdminURL),
// created and dropped per test and never businessos_dev.
package catalogdb

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/founderos/pages/pgtest"
)

// TestDB returns a migrated, empty database, skipping when the bridge
// Postgres is not running.
func TestDB(t testing.TB) *pgxpool.Pool {
	t.Helper()
	admin := pgtest.AdminURL()
	ctx := context.Background()
	ap, err := pgxpool.New(ctx, admin)
	if err == nil {
		err = ap.Ping(ctx)
	}
	if err != nil {
		t.Skipf("bridge Postgres unreachable (%v)", err)
	}
	name := fmt.Sprintf("founderos_catalog_test_%d", rand.New(rand.NewSource(time.Now().UnixNano())).Intn(1_000_000_000))
	if _, err := ap.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = ap.Exec(context.Background(), "DROP DATABASE IF EXISTS "+name+" WITH (FORCE)")
		ap.Close()
	})
	pool, err := pgxpool.New(ctx, pgtest.DatabaseURL(name))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err := pgtest.RunMigrations(ctx, pool); err != nil {
		t.Fatal(err)
	}
	return pool
}

// Workspace inserts a workspace and returns its id.
func Workspace(t testing.TB, pool *pgxpool.Pool, slug string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO workspaces (name, slug, owner_id) VALUES ($1, $1, 'u1') RETURNING id::text`, slug).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}
