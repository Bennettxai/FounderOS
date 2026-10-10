package etl

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// wantFounderosTables is the 55 target tables of docs/founderos/table-map.md
// (the 52 of migration 160, plus migration 165's roadmap, phases and domains).
var wantFounderosTables = []string{
	"founderos_departments", "founderos_agents", "founderos_people", "founderos_sop_tasks",
	"founderos_tools", "founderos_skills", "founderos_personas", "founderos_workflows",
	"founderos_metrics", "founderos_agent_runs", "founderos_agent_messages", "founderos_agent_tasks",
	"founderos_agent_crons", "founderos_cron_runs", "founderos_broadcasts", "founderos_broadcast_replies",
	"founderos_comms_digests", "founderos_digest_reads", "founderos_contact_tags", "founderos_plaud_ingests",
	"founderos_social_accounts", "founderos_social_snapshots", "founderos_social_dms",
	"founderos_social_dm_snapshots", "founderos_social_dm_messages", "founderos_social_posts",
	"founderos_email_list_snapshots", "founderos_email_list_sync_misses", "founderos_lead_magnets",
	"founderos_brand_deals", "founderos_funnel_contacts", "founderos_funnel_touches", "founderos_funnel_archive",
	"founderos_proposals", "founderos_deliverable_decisions", "founderos_client_work",
	"founderos_vsl_snapshots", "founderos_trading_snapshots", "founderos_trading_positions",
	"founderos_trading_orders", "founderos_trading_analysis", "founderos_trading_activity",
	"founderos_trading_limits", "founderos_metric_snapshots", "founderos_usage_snapshots",
	"founderos_ollama_snapshots", "founderos_seed_meta", "founderos_bank_summaries",
	"founderos_ledger_rows", "founderos_paykit_customer_snapshots",
	"founderos_slack_bridge_jobs", "founderos_slack_bridge_sessions",
	"founderos_phases", "founderos_roadmap_items", "founderos_domains",
}

// bridgeNativeTables are founderos_* tables the bridge creates for itself
// (not ETL targets from FounderOS v1), e.g. migration 161's call archive.
var bridgeNativeTables = []string{"founderos_call_archive", "founderos_device_pushes"}

// pendingETLTables are founderos_* tables a page reads whose ETL copy from
// FounderOS v1 is not in spec.go yet (none now: the funnel archive landed).
var pendingETLTables = []string{}

func dbDir() string { return filepath.Join("..", "..", "database") }

func TestMigration160IsMirroredInSchemaSQL(t *testing.T) {
	mig, err := os.ReadFile(filepath.Join(dbDir(), "migrations", "160_founderos_tables.sql"))
	if err != nil {
		t.Fatal(err)
	}
	schema, err := os.ReadFile(filepath.Join(dbDir(), "schema.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(schema), strings.TrimSpace(string(mig))) {
		t.Fatal("schema.sql does not contain migrations/160_founderos_tables.sql verbatim; fold it in so a fresh bootstrap and a migrated DB agree")
	}
}

func TestWantFounderosTablesHas55Names(t *testing.T) {
	seen := map[string]bool{}
	for _, n := range wantFounderosTables {
		if seen[n] {
			t.Fatalf("duplicate %s", n)
		}
		seen[n] = true
	}
	if len(seen) != 55 {
		t.Fatalf("table map names 55 targets, list has %d", len(seen))
	}
}

func TestFounderosTablesExistWithWorkspaceFK(t *testing.T) {
	pool, _ := newTestDB(t)
	ctx := context.Background()

	rows, err := pool.Query(ctx, `SELECT table_name FROM information_schema.tables
		WHERE table_schema = 'public' AND table_name LIKE 'founderos\_%' ORDER BY 1`)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			t.Fatal(err)
		}
		got = append(got, n)
	}
	rows.Close()
	want := append(append(append([]string(nil), wantFounderosTables...), bridgeNativeTables...), pendingETLTables...)
	sort.Strings(want)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("founderos tables:\n got %v\nwant %v", got, want)
	}

	for _, tbl := range wantFounderosTables {
		var n int
		err := pool.QueryRow(ctx, `
			SELECT count(*) FROM pg_constraint c
			JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = ANY(c.conkey)
			WHERE c.contype = 'f' AND c.conrelid = $1::regclass
			  AND c.confrelid = 'workspaces'::regclass AND a.attname = 'workspace_id'
			  AND a.attnotnull AND c.confdeltype = 'c'`, tbl).Scan(&n)
		if err != nil {
			t.Fatal(err)
		}
		if n != 1 {
			t.Errorf("%s: want workspace_id NOT NULL REFERENCES workspaces ON DELETE CASCADE, found %d", tbl, n)
		}
		var imported bool
		if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM information_schema.columns
			WHERE table_name = $1 AND column_name = 'imported_at')`, tbl).Scan(&imported); err != nil {
			t.Fatal(err)
		}
		if !imported {
			t.Errorf("%s: missing imported_at", tbl)
		}
	}

	// The migration is recorded by the runner and re-applying it is a no-op.
	var recorded bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = '160_founderos_tables.sql')`).Scan(&recorded); err != nil {
		t.Fatal(err)
	}
	if !recorded {
		t.Error("160_founderos_tables.sql not recorded in schema_migrations")
	}
	body, err := os.ReadFile(filepath.Join(dbDir(), "migrations", "160_founderos_tables.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, string(body)); err != nil {
		t.Fatalf("re-applying 160 is not idempotent: %v", err)
	}
}
