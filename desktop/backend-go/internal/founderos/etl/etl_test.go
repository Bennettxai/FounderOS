package etl

import (
	"compress/gzip"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/security"
)

// fixtureKey is a throwaway TOKEN_ENCRYPTION_KEY for tests only.
var fixtureKey = base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))

func execSQLite(t *testing.T, path string, stmts ...string) {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("sqlite %s: %v\n%.200s", filepath.Base(path), err, s)
		}
	}
}

// buildFixture materialises FounderOS v1's seed (testdata/founderos-os.seed.sql.gz,
// see testdata/build-fixture.sh) and adds a handful of synthetic rows to the
// tables the seed leaves empty, plus synthetic side databases.
func buildFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	f, err := os.Open(filepath.Join("testdata", "founderos-os.seed.sql.gz"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	dump, err := io.ReadAll(zr)
	if err != nil {
		t.Fatal(err)
	}
	osDB := filepath.Join(dir, FileOS)
	execSQLite(t, osDB, string(dump))
	// FounderOS v1's funnel_archive (2026-09-26) postdates the seed dump: the
	// schema of lib/db.ts plus one journey per venture.
	execSQLite(t, osDB,
		`CREATE TABLE IF NOT EXISTS funnel_archive (id TEXT PRIMARY KEY, venture TEXT NOT NULL CHECK (venture IN ('vantage','launchpad-cohort')), journey TEXT NOT NULL)`,
		`INSERT INTO funnel_archive VALUES ('attio-1','launchpad-cohort','{"id":"attio-1","venture":"launchpad-cohort","name":"Ada Example","touches":[]}')`,
		`INSERT INTO funnel_archive VALUES ('ghl-1','vantage','{"id":"ghl-1","venture":"vantage","name":"Bo Example","touches":[]}')`,
	)
	execSQLite(t, osDB,
		`INSERT INTO comms_digests VALUES ('dig-1','2026-09-23T17:28:42.826Z','{"digest":{"generatedAt":"2026-09-23T17:28:42.826Z","windowHours":24,"entries":[]}}')`,
		`INSERT INTO comms_digests VALUES ('dig-0','2026-09-23T17:28:42.826Z','{"digest":{"entries":[]}}')`,
		`INSERT INTO digest_reads VALUES ('gmail:abc','2026-09-24T09:00:00.000Z')`,
		`INSERT INTO contact_tags VALUES ('Ada Example','whatsapp','friend',1)`,
		`INSERT INTO plaud_ingests VALUES ('of_1','Sync','', '2026-09-23T17:28:34.696Z','gbrain','2026-09-04-sync',2)`,
		`INSERT INTO cron_runs VALUES ('cr-1','cron-plaud-ingest-30m','sales-calls-data','2026-09-23T17:28:34.271Z',NULL,1,'running')`,
		`INSERT INTO broadcasts VALUES ('bc-1','status?','2026-09-20T10:00:00.000Z')`,
		`INSERT INTO broadcast_replies VALUES ('br-1','bc-1','conductor',1,'all good','2026-09-20T10:00:05.000Z')`,
		`INSERT INTO metric_snapshots VALUES ('agent-runs','2026-09-23T17:26:27.331Z',12)`,
		`INSERT INTO usage_snapshots VALUES ('seat-macbook','2026-09-24T12:00:00.000Z','{"seat":"macbook","plans":[]}')`,
		`INSERT INTO ollama_snapshots VALUES ('ollama-macbook','2026-09-24T12:00:00.000Z','{"models":[]}')`,
		`INSERT INTO vsl_snapshots VALUES ('vid1','2025-08-10','2026-09-24','2026-09-24T19:09:45.450Z','{"videoId":"vid1","plays":10}')`,
		`INSERT INTO trading_limits VALUES (1,50,10,1,3,5,100,500,1,'2026-09-18T12:00:00.000Z')`,
		`INSERT INTO trading_orders VALUES ('ord-1','agentic','AAPL','buy','limit','queued',1,0,NULL,190.5,'markets','2026-09-24T14:00:00.000Z')`,
		`INSERT INTO trading_analysis VALUES ('ta-1','2026-09-24T14:00:00.000Z','agentic','markets',50,2,'','[{"symbol":"AAPL"}]')`,
		`INSERT INTO deliverable_decisions VALUES ('proposal:vantage-nora','approved','2026-09-01T12:00:00.000Z','r1','')`,
		`INSERT INTO deliverable_decisions VALUES ('ws-abc/brief.md','dismissed','2026-09-02T12:00:00.000Z','','meh')`,
		`INSERT INTO client_work VALUES ('cw-1','client-a','Build the thing','saved','2026-09-10T12:00:00.000Z',NULL,NULL)`,
		`INSERT INTO brand_deals (id,brand,status,paid_in_full,deadline,notion_url,last_edited,seeded) VALUES ('bd-1','Acme','Pitched',0,'2026-10-01','https://notion.so/x','2026-09-20T10:00:00.000Z',1)`,
		`INSERT INTO agent_messages VALUES ('am-1','conductor','user','hi','[]','2026-09-01T10:00:00')`,
		`INSERT INTO email_list_sync_misses VALUES ('2026-09-24','no reading from beehiiv')`,
		`INSERT INTO workflows VALUES ('wf-user-1','Custom','',0,9,'[]')`,
		`INSERT INTO phases VALUES ('phase-1',1,'Real Connections','["Slack","Brain"]')`,
		`INSERT INTO roadmap_items (id,title,quarter,status,phase_id) VALUES ('r1','x','2026-Q3','done','phase-1'),('r2','y','2026-Q4','next',NULL)`,
		`INSERT INTO domains (id,number,title,color) VALUES ('d1',1,'x','#fff')`,
		// The founderos-os.db copies of the Slack tables are expected empty, but
		// a stray row must lose to state.db on a key collision.
		`INSERT INTO slack_bridge_jobs VALUES ('Ev-dup','{}','done')`,
	)

	execSQLite(t, filepath.Join(dir, FileBank),
		`CREATE TABLE bank_summaries (account TEXT NOT NULL, business TEXT NOT NULL, month TEXT NOT NULL, credits_cents INTEGER NOT NULL, debits_cents INTEGER NOT NULL, net_cents INTEGER NOT NULL, PRIMARY KEY (account, month))`,
		`INSERT INTO bank_summaries VALUES ('Checking 1','Vantage','2026-08',500000,200000,300000),('Checking 2','General Operations','2026-08',100,50,50)`)
	execSQLite(t, filepath.Join(dir, FileLedger),
		`CREATE TABLE ledger_rows (hash TEXT PRIMARY KEY, date TEXT NOT NULL, description TEXT NOT NULL, amount_cents INTEGER NOT NULL, direction TEXT NOT NULL, category TEXT NOT NULL, card TEXT NOT NULL DEFAULT 'platinum')`,
		`INSERT INTO ledger_rows VALUES ('gold|2026-08-01|CAFE|500|out','2026-08-01','CAFE',500,'out','Food','gold'),
			('blue|2026-08-02|ADS|9900|out','2026-08-02','ADS',9900,'out','Ads','blue'),
			('business|2026-08-03|SAAS|1200|out','2026-08-03','SAAS',1200,'out','Software','business')`)
	execSQLite(t, filepath.Join(dir, FilePaykit),
		`CREATE TABLE paykit_customer_snapshots (account TEXT NOT NULL, captured_on TEXT NOT NULL, customer_id TEXT NOT NULL, total_spent_cents INTEGER NOT NULL, total_transactions INTEGER NOT NULL, last_transaction_date TEXT, source TEXT NOT NULL, PRIMARY KEY (account, captured_on, customer_id))`,
		`INSERT INTO paykit_customer_snapshots VALUES ('paykit-lc','2026-09-20','c-1',250000,2,'2026-08-09T15:15:50-05:00','live'),('paykit-lc','2026-09-20','c-2',100000,1,NULL,'live')`)
	state := filepath.Join(dir, "slack-bridge", FileState)
	if err := os.MkdirAll(filepath.Dir(state), 0o755); err != nil {
		t.Fatal(err)
	}
	execSQLite(t, state,
		`CREATE TABLE slack_bridge_jobs (id TEXT PRIMARY KEY, payload TEXT NOT NULL, state TEXT NOT NULL)`,
		`CREATE TABLE slack_bridge_sessions (key TEXT PRIMARY KEY, payload TEXT NOT NULL)`,
		`CREATE TABLE agents (id TEXT PRIMARY KEY)`, // openDb() creates the rest; they stay empty
		`INSERT INTO slack_bridge_jobs VALUES ('Ev-1','{"id":"Ev-1","key":"T:C:1","channel":"C","thread":"1","text":"run the tests"}','queued')`,
		`INSERT INTO slack_bridge_jobs VALUES ('Ev-dup','{}','processing')`,
		`INSERT INTO slack_bridge_sessions VALUES ('T:C:1','{"workspace":"ws-1","terminal":"term-1","pending":{"code":"123456","text":"deploy","expires":1790000000000}}')`,
	)
	return dir
}

// seedWorkspaces creates the operator's user and the four workspaces (what
// cmd/founderos-bootstrap, criterion 2.2, will do for real).
func seedWorkspaces(t *testing.T, pool *pgxpool.Pool, skip ...string) {
	t.Helper()
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `INSERT INTO "user" (id, name, email) VALUES ('founderos-test', 'Alex', 'alex@example.test') ON CONFLICT DO NOTHING`); err != nil {
		t.Fatal(err)
	}
	for _, s := range Slugs {
		if contains(skip, s) {
			continue
		}
		if _, err := pool.Exec(ctx, `INSERT INTO workspaces (name, slug, owner_id) VALUES ($1, $1, 'founderos-test')`, s); err != nil {
			t.Fatal(err)
		}
	}
}

func contains(xs []string, x string) bool {
	for _, y := range xs {
		if y == x {
			return true
		}
	}
	return false
}

func sqliteCount(t *testing.T, path, table string) int {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var n int
	if err := db.QueryRow(`SELECT count(*) FROM ` + table).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func pgScalar[T any](t *testing.T, pool *pgxpool.Pool, q string, args ...any) T {
	t.Helper()
	var v T
	if err := pool.QueryRow(context.Background(), q, args...).Scan(&v); err != nil {
		t.Fatalf("%s: %v", q, err)
	}
	return v
}

func runETL(t *testing.T, pool *pgxpool.Pool, dir string) *Report {
	t.Helper()
	rep, err := Run(context.Background(), pool, Options{From: dir, EncryptionKey: fixtureKey, DatabaseLabel: "test"})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	return rep
}

func TestETLLoadsSeedFixtureIdempotently(t *testing.T) {
	pool, _ := newTestDB(t)
	seedWorkspaces(t, pool)
	dir := buildFixture(t)
	osDB := filepath.Join(dir, FileOS)

	rep := runETL(t, pool, dir)
	if !rep.OK {
		t.Fatalf("report not OK: %v", rep.Problems)
	}
	if len(rep.Tables) != 55 {
		t.Fatalf("report has %d tables, want 55", len(rep.Tables))
	}
	for _, tr := range rep.Tables {
		if tr.Skipped != "" {
			t.Errorf("%s skipped: %s", tr.Target, tr.Skipped)
		}
		if tr.SourceRows != tr.TargetRows {
			t.Errorf("%s: source %d, target %d", tr.Target, tr.SourceRows, tr.TargetRows)
		}
		if tr.Inserted != tr.SourceRows {
			t.Errorf("%s: first run inserted %d of %d", tr.Target, tr.Inserted, tr.SourceRows)
		}
	}

	// Source counts equal the SQLite counts, table by table.
	for _, sp := range Specs() {
		if sp.sources[0].file != FileOS || len(sp.sources) > 1 || sp.target == "founderos_trading_limits" {
			continue
		}
		want := sqliteCount(t, osDB, sp.sources[0].table)
		if got := rep.Table(sp.target).TargetRows; got != want {
			t.Errorf("%s: target %d, sqlite %s has %d", sp.target, got, sp.sources[0].table, want)
		}
	}
	for tbl, want := range map[string]int{"founderos_departments": 6, "founderos_agents": 32, "founderos_social_snapshots": 455,
		"founderos_funnel_touches": 61, "founderos_proposals": 4, "founderos_trading_limits": 1,
		"founderos_bank_summaries": 2, "founderos_ledger_rows": 3, "founderos_paykit_customer_snapshots": 2,
		"founderos_slack_bridge_jobs": 2, "founderos_slack_bridge_sessions": 1} {
		if got := pgScalar[int](t, pool, "SELECT count(*) FROM "+tbl); got != want {
			t.Errorf("%s rows = %d, want %d", tbl, got, want)
		}
	}
	// roadmap_items, phases and domains are no longer dropped: /os/roadmap and
	// /os/reference read them (migration 165), in the FounderOS HQ workspace.
	if len(rep.Dropped) != 0 {
		t.Errorf("dropped = %+v, want none", rep.Dropped)
	}
	for tbl, want := range map[string]int{"founderos_roadmap_items": 2, "founderos_phases": 1, "founderos_domains": 1} {
		if got := pgScalar[int](t, pool, "SELECT count(*) FROM "+tbl+" WHERE workspace_id = $1", rep.Workspaces[WSFounderOS]); got != want {
			t.Errorf("%s rows in founderos = %d, want %d", tbl, got, want)
		}
	}
	if got := pgScalar[string](t, pool, `SELECT coalesce(phase_id, '-') || ' ' || quarter || ' ' || status FROM founderos_roadmap_items WHERE id = 'r1'`); got != "phase-1 2026-Q3 done" {
		t.Errorf("roadmap row r1 = %q", got)
	}
	if got := pgScalar[string](t, pool, `SELECT items::text FROM founderos_phases WHERE id = 'phase-1'`); got != `["Slack", "Brain"]` {
		t.Errorf("phase items = %s, want a JSONB string array", got)
	}

	// Transform rules.
	ws := rep.Workspaces
	if got := pgScalar[bool](t, pool, `SELECT autopilot FROM founderos_trading_limits`); got {
		t.Error("autopilot imported ON")
	}
	if rep.TradingLimits.SourceAutopilot == nil || !*rep.TradingLimits.SourceAutopilot {
		t.Error("report must state the source autopilot was ON")
	}
	if got := pgScalar[string](t, pool, `SELECT workspace_id::text FROM founderos_trading_limits`); got != ws[WSPersonal] {
		t.Error("trading_limits not in personal")
	}
	if got := pgScalar[bool](t, pool, `SELECT recorded_at IS NULL FROM founderos_plaud_ingests WHERE file_id = 'of_1'`); !got {
		t.Error("blank recorded_at must become NULL (D6)")
	}
	if rep.Table("founderos_plaud_ingests").D6BlankToNull != 1 {
		t.Error("D6 not counted")
	}
	if rep.Table("founderos_social_dms").D2DateOnly != 5 {
		t.Errorf("social_dms D2 = %d, want 5", rep.Table("founderos_social_dms").D2DateOnly)
	}
	if rep.Table("founderos_agent_messages").NoOffsetAsUTC != 1 {
		t.Error("no-offset instant not counted")
	}
	if got := pgScalar[string](t, pool, `SELECT to_char(created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI') FROM founderos_agent_messages`); got != "2026-09-01 10:00" {
		t.Errorf("no-offset instant read as %s, want UTC", got)
	}
	if got := pgScalar[string](t, pool, `SELECT captured_on::text FROM founderos_social_snapshots ORDER BY captured_on LIMIT 1`); got != "2026-03-14" {
		t.Errorf("first social day = %s", got)
	}
	if got := pgScalar[string](t, pool, `SELECT month::text FROM founderos_bank_summaries WHERE business = 'Vantage'`); got != "2026-08-01" {
		t.Errorf("D4 month = %s", got)
	}
	if got := pgScalar[string](t, pool, `SELECT jsonb_typeof(tools) FROM founderos_agents LIMIT 1`); got != "array" {
		t.Errorf("agents.tools is %s", got)
	}
	if got := pgScalar[int](t, pool, `SELECT count(*) FROM founderos_funnel_touches t JOIN founderos_funnel_contacts c ON c.id = t.contact_id WHERE t.workspace_id <> c.workspace_id`); got != 0 {
		t.Errorf("%d touches not in their contact's workspace", got)
	}
	if got := pgScalar[string](t, pool, `SELECT workspace_id::text FROM founderos_funnel_contacts WHERE id = 'fc-jake-moreau'`); got != ws[WSLaunchpadCohort] {
		t.Error("launchpad-cohort contact misrouted")
	}
	if got := pgScalar[string](t, pool, `SELECT workspace_id::text FROM founderos_workflows WHERE id = 'wf-vantage-sales'`); got != ws[WSVantage] {
		t.Error("vantage workflow misrouted")
	}
	if got := pgScalar[string](t, pool, `SELECT workspace_id::text FROM founderos_deliverable_decisions WHERE id = 'proposal:vantage-nora'`); got != ws[WSVantage] {
		t.Error("proposal decision not in the proposal's workspace")
	}
	if got := pgScalar[string](t, pool, `SELECT workspace_id::text FROM founderos_deliverable_decisions WHERE id = 'ws-abc/brief.md'`); got != ws[WSFounderOS] {
		t.Error("superset decision not in founderos")
	}
	// PayKit dates are copied verbatim, as FounderOS v1 stores them.
	if got := pgScalar[string](t, pool, `SELECT last_transaction_date FROM founderos_paykit_customer_snapshots WHERE customer_id = 'c-1'`); got != "2026-08-09T15:15:50-05:00" {
		t.Errorf("paykit last_transaction_date = %q, want the raw source text", got)
	}
	if got := pgScalar[string](t, pool, `SELECT card || ':' || workspace_id::text FROM founderos_ledger_rows WHERE description = 'SAAS'`); got != "platinum:"+ws[WSLaunchpadCohort] {
		t.Errorf("legacy business card row = %s", got)
	}
	if got := pgScalar[string](t, pool, `SELECT workspace_id::text FROM founderos_ledger_rows WHERE card = 'gold'`); got != ws[WSPersonal] {
		t.Error("gold card not personal")
	}
	if got := pgScalar[string](t, pool, `SELECT workspace_id::text FROM founderos_bank_summaries WHERE business = 'General Operations'`); got != ws[WSLaunchpadCohort] {
		t.Error("General Operations not launchpad-cohort")
	}
	if got := pgScalar[bool](t, pool, `SELECT payload->'pending' = 'null'::jsonb FROM founderos_slack_bridge_sessions`); !got {
		t.Error("slack pending nonce copied")
	}
	if rep.PendingNulled != 1 {
		t.Errorf("pending nulled = %d", rep.PendingNulled)
	}
	if got := pgScalar[string](t, pool, `SELECT state FROM founderos_slack_bridge_jobs WHERE id = 'Ev-dup'`); got != "processing" {
		t.Errorf("state.db must win a key collision, got %s", got)
	}
	if got := pgScalar[int64](t, pool, `SELECT seq FROM founderos_slack_bridge_jobs WHERE id = 'Ev-1'`); got != 1 {
		t.Errorf("slack job seq = %d, want the source rowid 1", got)
	}
	if got := pgScalar[int](t, pool, `SELECT count(*) FROM information_schema.columns WHERE table_name = 'founderos_proposals' AND column_name = 'access_code'`); got != 0 {
		t.Error("founderos_proposals must not carry access_code")
	}
	if got := pgScalar[int](t, pool, `SELECT count(*) FROM founderos_proposals WHERE has_access_code`); got != 4 {
		t.Errorf("has_access_code on %d proposals, want 4", got)
	}

	// Access codes: encrypted in credential_vault, readable with the key.
	if rep.Vault.Action != "inserted" || rep.Vault.Codes != 4 {
		t.Fatalf("vault = %+v", rep.Vault)
	}
	cipher := pgScalar[[]byte](t, pool, `SELECT encrypted_data FROM credential_vault WHERE user_id = 'founderos-test' AND provider_id = $1`, VaultProvider)
	if strings.Contains(string(cipher), "fixture-code") {
		t.Fatal("access codes stored in plain text")
	}
	enc, err := security.NewTokenEncryption(fixtureKey)
	if err != nil {
		t.Fatal(err)
	}
	plain, err := enc.DecryptBytes(cipher)
	if err != nil {
		t.Fatal(err)
	}
	var codes map[string]string
	if err := json.Unmarshal([]byte(plain), &codes); err != nil || len(codes) != 4 || !strings.HasPrefix(codes["vantage-nora"], "fixture-code-") {
		t.Fatalf("vault payload = %v, %v", codes, err)
	}

	// Reports.
	out := t.TempDir()
	jp, mp, err := rep.Write(out, "etl-report")
	if err != nil {
		t.Fatal(err)
	}
	md, _ := os.ReadFile(mp)
	if !strings.Contains(string(md), "founderos_agents") || !strings.Contains(string(md), "GREEN") || strings.Contains(string(md), "fixture-code") {
		t.Errorf("markdown report:\n%s", md)
	}
	var back Report
	if b, _ := os.ReadFile(jp); json.Unmarshal(b, &back) != nil || len(back.Tables) != 55 {
		t.Error("JSON report does not round-trip")
	}

	// Second run: identical counts, zero changes, imported_at untouched.
	stamp := pgScalar[time.Time](t, pool, `SELECT max(imported_at) FROM founderos_agents`)
	rep2 := runETL(t, pool, dir)
	if rep2.Changes() != 0 {
		for _, tr := range rep2.Tables {
			if tr.Inserted+tr.Updated+tr.Deleted > 0 {
				t.Errorf("re-run changed %s: +%d ~%d -%d", tr.Target, tr.Inserted, tr.Updated, tr.Deleted)
			}
		}
		t.Fatalf("re-run made %d changes (vault %s)", rep2.Changes(), rep2.Vault.Action)
	}
	for i, tr := range rep2.Tables {
		if tr.TargetRows != rep.Tables[i].TargetRows || tr.SourceRows != rep.Tables[i].SourceRows {
			t.Errorf("%s counts moved on re-run", tr.Target)
		}
	}
	if rep2.Vault.Action != "unchanged" {
		t.Errorf("vault on re-run: %s", rep2.Vault.Action)
	}
	if got := pgScalar[time.Time](t, pool, `SELECT max(imported_at) FROM founderos_agents`); !got.Equal(stamp) {
		t.Error("re-run rewrote unchanged rows")
	}

	// Mirror: source prunes an agent, edits a tool, drops a touch -> the
	// target follows exactly.
	execSQLite(t, osDB,
		`DELETE FROM agents WHERE rowid = (SELECT max(rowid) FROM agents)`,
		`UPDATE tools SET name = name || ' (renamed)' WHERE rowid = 1`,
		`DELETE FROM funnel_touches WHERE id = 'fc-jake-moreau-t2'`,
		`UPDATE proposals SET access_code = '' WHERE id = 'vantage-reese'`,
	)
	rep3 := runETL(t, pool, dir)
	if !rep3.OK {
		t.Fatalf("mirror run not OK: %v", rep3.Problems)
	}
	if a := rep3.Table("founderos_agents"); a.Deleted != 1 || a.TargetRows != 31 {
		t.Errorf("agents after prune: %+v", a)
	}
	if tl := rep3.Table("founderos_tools"); tl.Updated != 1 || tl.Inserted != 0 {
		t.Errorf("tools after edit: %+v", tl)
	}
	if ft := rep3.Table("founderos_funnel_touches"); ft.Deleted != 1 {
		t.Errorf("touches after delete: %+v", ft)
	}
	if rep3.Vault.Action != "updated" || rep3.Vault.Codes != 3 {
		t.Errorf("vault after code removal: %+v", rep3.Vault)
	}
	if rep3.Changes() != 1+1+1+1+1 { // agent, tool, touch, proposal flag, vault
		t.Errorf("mirror run changes = %d", rep3.Changes())
	}
}

func TestETLFailsClearlyWhenAWorkspaceIsMissing(t *testing.T) {
	pool, _ := newTestDB(t)
	seedWorkspaces(t, pool, WSPersonal)
	_, err := Run(context.Background(), pool, Options{From: buildFixture(t), EncryptionKey: fixtureKey})
	if err == nil || !strings.Contains(err.Error(), "personal") || !strings.Contains(err.Error(), "founderos-bootstrap") {
		t.Fatalf("err = %v", err)
	}
	if n := pgScalar[int](t, pool, `SELECT count(*) FROM founderos_departments`); n != 0 {
		t.Fatalf("wrote %d rows despite the failed preflight", n)
	}
}

func TestETLWorkspaceMapAndAbsentSideFiles(t *testing.T) {
	pool, _ := newTestDB(t)
	seedWorkspaces(t, pool, WSPersonal)
	if _, err := pool.Exec(context.Background(), `INSERT INTO workspaces (name, slug, owner_id) VALUES ('Alex Personal', 'founderos-personal', 'founderos-test')`); err != nil {
		t.Fatal(err)
	}
	dir := buildFixture(t)
	for _, f := range []string{FileBank, FileLedger, FilePaykit} {
		os.Remove(filepath.Join(dir, f))
	}
	os.RemoveAll(filepath.Join(dir, "slack-bridge"))

	rep, err := Run(context.Background(), pool, Options{From: dir, EncryptionKey: fixtureKey,
		WorkspaceMap: map[string]string{WSPersonal: "founderos-personal"}})
	if err != nil {
		t.Fatal(err)
	}
	if !rep.OK {
		t.Fatalf("problems: %v", rep.Problems)
	}
	want := pgScalar[string](t, pool, `SELECT id::text FROM workspaces WHERE slug = 'founderos-personal'`)
	if rep.Workspaces[WSPersonal] != want {
		t.Error("workspace map ignored")
	}
	for _, tbl := range []string{"founderos_bank_summaries", "founderos_ledger_rows", "founderos_paykit_customer_snapshots"} {
		if tr := rep.Table(tbl); tr.Skipped == "" {
			t.Errorf("%s: absent source must be skipped, not treated as empty", tbl)
		}
	}
	// founderos-os.db still has its (stray) Slack job, so that table loads.
	if tr := rep.Table("founderos_slack_bridge_jobs"); tr.Skipped != "" || tr.TargetRows != 1 {
		t.Errorf("slack jobs from founderos-os.db alone: %+v", tr)
	}
}

func TestETLRequiresKeyForAccessCodes(t *testing.T) {
	pool, _ := newTestDB(t)
	seedWorkspaces(t, pool)
	_, err := Run(context.Background(), pool, Options{From: buildFixture(t)})
	if err == nil || !strings.Contains(err.Error(), "TOKEN_ENCRYPTION_KEY") {
		t.Fatalf("err = %v", err)
	}
	if n := pgScalar[int](t, pool, `SELECT count(*) FROM credential_vault`); n != 0 {
		t.Fatal("vault written without a key")
	}
}
