package schemahealth

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestLiveSchemaStartupBudget(t *testing.T) {
	url := os.Getenv("SCHEMA_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("read-only live database check requires SCHEMA_TEST_DATABASE_URL")
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	start := time.Now()
	report, err := Check(ctx, pool)
	t.Logf("schema check elapsed=%s missing=%v", time.Since(start), report.Missing)
	if err != nil {
		t.Fatal(err)
	}
}
