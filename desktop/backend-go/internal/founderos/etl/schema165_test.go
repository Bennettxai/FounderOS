package etl

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMigration165IsMirroredInSchemaSQL(t *testing.T) {
	mig, err := os.ReadFile(filepath.Join(dbDir(), "migrations", "165_founderos_roadmap.sql"))
	if err != nil {
		t.Fatal(err)
	}
	schema, _ := os.ReadFile(filepath.Join(dbDir(), "schema.sql"))
	if !strings.Contains(string(schema), strings.TrimSpace(string(mig))) {
		t.Fatal("schema.sql must contain migrations/165_founderos_roadmap.sql verbatim")
	}
}

// The CHECKs mirror FounderOS v1's Zod (lib/schemas.ts): RoadmapStatusSchema
// is done|now|next|later and a quarter must look like 2026-Q2.
func TestMigration165ChecksMirrorV1Schemas(t *testing.T) {
	pool, _ := newTestDB(t)
	seedWorkspaces(t, pool)
	ctx := context.Background()
	var ws string
	if err := pool.QueryRow(ctx, `SELECT id::text FROM workspaces WHERE slug = 'founderos'`).Scan(&ws); err != nil {
		t.Fatal(err)
	}
	ins := `INSERT INTO founderos_roadmap_items (id, workspace_id, title, quarter, status) VALUES ($1, $2, 't', $3, $4)`
	if _, err := pool.Exec(ctx, ins, "ok", ws, "2026-Q4", "later"); err != nil {
		t.Fatalf("a valid row was refused: %v", err)
	}
	for _, bad := range []struct{ id, quarter, status string }{
		{"bad-status", "2026-Q3", "planned"},
		{"bad-quarter", "Q3", "done"},
		{"bad-q5", "2026-Q5", "done"},
	} {
		if _, err := pool.Exec(ctx, ins, bad.id, ws, bad.quarter, bad.status); err == nil {
			t.Errorf("%s: want a CHECK violation", bad.id)
		}
	}
	// Re-applying the migration is a no-op.
	body, err := os.ReadFile(filepath.Join(dbDir(), "migrations", "165_founderos_roadmap.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, string(body)); err != nil {
		t.Fatalf("re-applying 165 is not idempotent: %v", err)
	}
}
