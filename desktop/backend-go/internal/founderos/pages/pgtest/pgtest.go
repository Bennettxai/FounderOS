// Package pgtest gives tests a throwaway, migrated database on a test
// Postgres server, created and dropped per test. It never touches
// businessos_dev.
//
// The server is FOUNDEROS_PG_ADMIN_URL (default: the local stack's
// 127.0.0.1:25432). Every per-test database URL is derived from that admin URL
// via DatabaseURL, so pointing the env var at another server moves every test
// helper with it; no helper may hardcode a host or port.
package pgtest

import (
	"context"
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rhl/businessos-backend/internal/database"
)

// DefaultAdminURL is the admin connection used when FOUNDEROS_PG_ADMIN_URL is
// unset: the postgres database on the local stack's Postgres.
const DefaultAdminURL = "postgres://postgres@127.0.0.1:25432/postgres?sslmode=disable"

// AdminURL is the admin connection string tests create and drop their
// throwaway databases through: FOUNDEROS_PG_ADMIN_URL, else the repo .env.dev
// POSTGRES_PORT, else DefaultAdminURL.
func AdminURL() string {
	if u := os.Getenv("FOUNDEROS_PG_ADMIN_URL"); u != "" {
		return u
	}
	wd, _ := os.Getwd()
	return adminURLFrom(wd)
}

// adminURLFrom finds the repo's .env.dev above dir and uses its POSTGRES_PORT
// (the dev Postgres dev-local.sh runs for this checkout), else DefaultAdminURL.
func adminURLFrom(dir string) string {
	for {
		raw, err := os.ReadFile(filepath.Join(dir, ".env.dev"))
		if err == nil {
			for _, line := range strings.Split(string(raw), "\n") {
				if v, ok := strings.CutPrefix(strings.TrimSpace(line), "POSTGRES_PORT="); ok && strings.TrimSpace(v) != "" {
					return "postgres://postgres@127.0.0.1:" + strings.TrimSpace(v) + "/postgres?sslmode=disable"
				}
			}
			return DefaultAdminURL
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return DefaultAdminURL
		}
		dir = parent
	}
}

// DatabaseURL is AdminURL with its database swapped for name: same host,
// port, user and options.
func DatabaseURL(name string) string {
	u, err := url.Parse(AdminURL())
	if err != nil {
		panic(fmt.Sprintf("pgtest: unparsable admin url: %v", err))
	}
	u.Path = "/" + name
	return u.String()
}

// bootstrapLockKey is the advisory-lock key (on the admin database) that
// serializes schema bootstraps across every test process.
const bootstrapLockKey int64 = 0x464f535f54455354 // "FOS_TEST"

// RunMigrations applies the BusinessOS migrations to a throwaway database,
// one test process at a time. A fresh bootstrap is a single transaction that
// holds ~3,200 locks, so with Postgres' default lock table (64 per
// connection x 100 connections) two concurrent bootstraps from parallel test
// packages exhaust it ("out of shared memory", SQLSTATE 53200). A
// session-level advisory lock on the admin database serializes them; it is
// released when the admin connection closes, even if the test process dies.
func RunMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	if conn, err := pgx.Connect(ctx, AdminURL()); err == nil {
		defer conn.Close(context.Background())
		if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock($1)", bootstrapLockKey); err != nil {
			return fmt.Errorf("bootstrap lock: %w", err)
		}
	}
	return database.RunMigrations(ctx, pool)
}

// ThrowawayDB creates and drops a migrated database on the bridge Postgres
// (never businessos_dev). Skips when the bridge Postgres is not running.
func ThrowawayDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	admin := AdminURL()
	ctx := context.Background()
	ap, err := pgxpool.New(ctx, admin)
	if err == nil {
		err = ap.Ping(ctx)
	}
	if err != nil {
		t.Skipf("bridge Postgres unreachable (%v)", err)
	}
	name := fmt.Sprintf("founderos_pages_test_%d", rand.New(rand.NewSource(time.Now().UnixNano())).Intn(1_000_000))
	if _, err := ap.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = ap.Exec(context.Background(), "DROP DATABASE IF EXISTS "+name+" WITH (FORCE)")
		ap.Close()
	})
	pool, err := pgxpool.New(ctx, DatabaseURL(name))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err := RunMigrations(ctx, pool); err != nil {
		t.Fatal(err)
	}
	return pool
}
