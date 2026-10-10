package etl

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/founderos/pages/pgtest"
)

// testServerURL is the test Postgres admin URL (FOUNDEROS_ETL_TEST_PG, else
// the shared FOUNDEROS_PG_ADMIN_URL via pgtest). Tests
// never touch an existing database: each run creates founderos_etl_test_<n>,
// applies the real migration runner to it, and drops it afterwards.
func testServerURL() string {
	if u := os.Getenv("FOUNDEROS_ETL_TEST_PG"); u != "" {
		return u
	}
	return pgtest.AdminURL()
}

// newTestDB returns a pool on a fresh throwaway database with every
// BusinessOS migration applied (schema.sql bootstrap + post-baseline files,
// including 160_founderos_tables.sql). It skips when the server is unreachable.
func newTestDB(t *testing.T) (*pgxpool.Pool, string) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping Postgres integration test in -short mode")
	}
	cfg, err := pgx.ParseConfig(testServerURL())
	if err != nil {
		t.Fatalf("parse test server url: %v", err)
	}
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", cfg.Host, cfg.Port), 2*time.Second)
	if err != nil {
		t.Skipf("bridge Postgres %s:%d unreachable (%v); start it with `make bridge-up` to run this test", cfg.Host, cfg.Port, err)
	}
	conn.Close()

	ctx := context.Background()
	admin, err := pgx.Connect(ctx, testServerURL())
	if err != nil {
		t.Skipf("bridge Postgres refused a connection (%v)", err)
	}
	name := fmt.Sprintf("founderos_etl_test_%d_%d", os.Getpid(), rand.Intn(1_000_000))
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		admin.Close(ctx)
		t.Fatalf("create throwaway db: %v", err)
	}
	admin.Close(ctx)

	dbCfg := cfg.Copy()
	dbCfg.Database = name
	dbURL := dbCfg.ConnString()
	poolCfg, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		t.Fatal(err)
	}
	poolCfg.ConnConfig.Database = name
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool.Close()
		a, err := pgx.Connect(context.Background(), testServerURL())
		if err != nil {
			t.Logf("drop %s: %v", name, err)
			return
		}
		defer a.Close(context.Background())
		if _, err := a.Exec(context.Background(), "DROP DATABASE IF EXISTS "+name+" WITH (FORCE)"); err != nil {
			t.Logf("drop %s: %v", name, err)
		}
	})
	if err := pgtest.RunMigrations(ctx, pool); err != nil {
		t.Fatalf("run migrations on %s: %v", name, err)
	}
	return pool, name
}
