package finances

import (
	"context"
	"fmt"
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/founderos/pages/pgtest"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/paykit"
)

// Ports of tests/ledger.test.ts and tests/bank.test.ts against Postgres,
// plus the paykit.db history port, on a throwaway database.

func throwawayDB(t *testing.T) *pgxpool.Pool {
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
	name := fmt.Sprintf("founderos_finances_test_%d", rand.New(rand.NewSource(time.Now().UnixNano())).Intn(1_000_000))
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

// SeedWorkspaces creates the FounderOS workspaces the per-row rules route to.
func seedWorkspaces(t *testing.T, pool *pgxpool.Pool, slugs ...string) map[string]string {
	t.Helper()
	ws := map[string]string{}
	for _, slug := range slugs {
		var id string
		if err := pool.QueryRow(context.Background(), `INSERT INTO workspaces (name, slug, owner_id) VALUES ($1,$1,'u1') RETURNING id::text`, slug).Scan(&id); err != nil {
			t.Fatal(err)
		}
		ws[slug] = id
	}
	return ws
}

var allSlugs = []string{"launchpad-cohort", "vantage", "personal"}

func lrow(date, desc string, cents int64, dir, cat, card string) LedgerRow {
	return LedgerRow{ParsedRow: ParsedRow{Date: date, Description: desc, AmountCents: cents, Direction: dir}, Category: cat, Card: card}
}

func TestPgLedgerInsertDedupesAndRoutesByCard(t *testing.T) {
	pool := throwawayDB(t)
	ws := seedWorkspaces(t, pool, allSlugs...)
	ctx := context.Background()
	s := &PgStore{Pool: pool}
	rows := []LedgerRow{
		lrow("2026-06-01", "AWS", 5700, "out", "Infrastructure", "platinum"),
		lrow("2026-06-02", "Facebook Ads", 150000, "out", "Advertising", "gold"),
		lrow("2026-06-03", "Notion", 4000, "out", "Software", "blue"),
		lrow("2026-06-03", "Notion", 4000, "out", "Software", "platinum"), // same charge, two cards
		lrow("2026-06-04", "Client", 500000, "in", "Income", ""),          // no card → the default lane
	}
	n, err := s.InsertLedgerRows(ctx, rows)
	if err != nil || n != 5 {
		t.Fatalf("insert = %d, %v", n, err)
	}
	if n, err = s.InsertLedgerRows(ctx, rows); err != nil || n != 0 {
		t.Fatalf("re-upload inserted %d, %v", n, err)
	}
	got, err := s.LedgerRows(ctx)
	if err != nil || len(got) != 5 {
		t.Fatalf("rows = %v, %v", got, err)
	}
	if got[0].Date != "2026-06-01" || got[len(got)-1].Card != "platinum" {
		t.Fatalf("oldest first, default card: %+v", got)
	}
	var wsOf = map[string]string{}
	r, err := pool.Query(ctx, `SELECT hash, workspace_id::text FROM founderos_ledger_rows`)
	if err != nil {
		t.Fatal(err)
	}
	for r.Next() {
		var h, w string
		_ = r.Scan(&h, &w)
		wsOf[h] = w
	}
	if wsOf["gold|2026-06-02|Facebook Ads|150000|out"] != ws["personal"] ||
		wsOf["blue|2026-06-03|Notion|4000|out"] != ws["vantage"] ||
		wsOf["platinum|2026-06-01|AWS|5700|out"] != ws["launchpad-cohort"] {
		t.Fatalf("per-row workspaces / verbatim hashes: %v", wsOf)
	}
}

func TestPgBankUpsertUpdatesInPlace(t *testing.T) {
	pool := throwawayDB(t)
	ws := seedWorkspaces(t, pool, allSlugs...)
	ctx := context.Background()
	s := &PgStore{Pool: pool}
	for _, b := range []BankSummary{
		bsum("4219", "General Operations", "2026-04", 1000, 500),
		bsum("5630", "Vantage", "2026-04", 2086597, 2498023),
		bsum("4219", "General Operations", "2026-03", 1800000, 1500000),
		bsum("4219", "General Operations", "2026-04", 2589928, 2169558),
	} {
		if err := s.UpsertBankSummary(ctx, b); err != nil {
			t.Fatal(err)
		}
	}
	all, err := s.BankSummaries(ctx)
	if err != nil || len(all) != 3 || all[0].Month != "2026-03" {
		t.Fatalf("all = %+v, %v", all, err)
	}
	for _, b := range all {
		if b.Account == "4219" && b.Month == "2026-04" && (b.CreditsCents != 2589928 || b.NetCents != 2589928-2169558) {
			t.Fatalf("re-upload must update: %+v", b)
		}
	}
	var mer string
	if err := pool.QueryRow(ctx, `SELECT workspace_id::text FROM founderos_bank_summaries WHERE account='5630'`).Scan(&mer); err != nil || mer != ws["vantage"] {
		t.Fatalf("vantage row in %s (%v)", mer, err)
	}
}

func TestPgStoreWithoutTheWorkspacesIsAnError(t *testing.T) {
	pool := throwawayDB(t)
	seedWorkspaces(t, pool, "vantage")
	_, err := (&PgStore{Pool: pool}).LedgerRows(context.Background())
	if err == nil || !strings.Contains(err.Error(), "launchpad-cohort") {
		t.Fatalf("err = %v", err)
	}
	if _, err := (&PgStore{}).BankSummaries(context.Background()); err == nil {
		t.Fatal("no pool must fail the read, not read empty")
	}
}

func TestPgPaykitHistoryReplacesTheDayAndCarriesTheSeed(t *testing.T) {
	pool := throwawayDB(t)
	ws := seedWorkspaces(t, pool, allSlugs...)
	ctx := context.Background()
	h := &PgPaykitHistory{Store: &PgStore{Pool: pool}, Account: paykit.LaunchpadCohort.ID}

	snaps, err := h.Snapshots()
	if err != nil || len(snaps) != 1 || snaps[0].CapturedOn != "2026-08-20" || snaps[0].Source != paykit.SourceReconstructed {
		t.Fatalf("seed = %+v, %v", snaps, err)
	}
	last := "2026-09-02T09:00:00-05:00"
	day := paykit.Snapshot{CapturedOn: "2026-09-03", Source: paykit.SourceLive, Customers: []paykit.Customer{
		{ID: "a", TotalSpentCents: 1000, Transactions: 1, LastTransactionDate: &last},
		{ID: "", TotalSpentCents: 999, Transactions: 1}, // no id: skipped
	}}
	if err := h.Record(day); err != nil {
		t.Fatal(err)
	}
	day.Customers = []paykit.Customer{{ID: "b", TotalSpentCents: 2000, Transactions: 2}}
	if err := h.Record(day); err != nil { // same day again: replaced wholesale
		t.Fatal(err)
	}
	snaps, err = h.Snapshots()
	if err != nil || len(snaps) != 2 {
		t.Fatalf("snaps = %+v, %v", snaps, err)
	}
	var ids []string
	for _, c := range snaps[1].Customers {
		ids = append(ids, c.ID)
	}
	if !reflect.DeepEqual(ids, []string{"b"}) {
		t.Fatalf("day not replaced: %v", ids)
	}
	var w string
	if err := pool.QueryRow(ctx, `SELECT DISTINCT workspace_id::text FROM founderos_paykit_customer_snapshots`).Scan(&w); err != nil || w != ws["launchpad-cohort"] {
		t.Fatalf("workspace %s (%v)", w, err)
	}

	// A recorded offset reads back in the source's zone, so the month slice holds.
	if err := h.Record(paykit.Snapshot{CapturedOn: "2026-09-04", Source: paykit.SourceLive, Customers: []paykit.Customer{{ID: "c", TotalSpentCents: 1, Transactions: 1, LastTransactionDate: &last}}}); err != nil {
		t.Fatal(err)
	}
	snaps, _ = h.Snapshots()
	got := snaps[len(snaps)-1].Customers[0].LastTransactionDate
	if got == nil || !strings.HasPrefix(*got, "2026-09-02T09:00:00") {
		t.Fatalf("last transaction date = %v", got)
	}
}

// PayKit sends last_transaction_date in its own shapes ("2026-10-01
// 09:00:00" has no offset and is not RFC 3339). FounderOS v1 keeps the raw text,
// so the bridge must too: never NULL, never rewritten into another zone.
func TestPgPaykitHistoryKeepsTheRawLastTransactionDate(t *testing.T) {
	pool := throwawayDB(t)
	seedWorkspaces(t, pool, allSlugs...)
	h := &PgPaykitHistory{Store: &PgStore{Pool: pool}, Account: paykit.LaunchpadCohort.ID}
	raws := map[string]string{"a": "2026-10-01 09:00:00", "b": "2026-08-09T15:15:50-05:00", "c": "2026-09-02T14:00:00Z"}
	var cs []paykit.Customer
	for _, id := range []string{"a", "b", "c"} {
		v := raws[id]
		cs = append(cs, paykit.Customer{ID: id, TotalSpentCents: 100, Transactions: 1, LastTransactionDate: &v})
	}
	if err := h.Record(paykit.Snapshot{CapturedOn: "2026-10-02", Source: paykit.SourceLive, Customers: cs}); err != nil {
		t.Fatal(err)
	}
	snaps, err := h.Snapshots()
	if err != nil {
		t.Fatal(err)
	}
	day := snaps[len(snaps)-1]
	if day.CapturedOn != "2026-10-02" || len(day.Customers) != 3 {
		t.Fatalf("day = %+v", day)
	}
	for _, c := range day.Customers {
		if c.LastTransactionDate == nil || *c.LastTransactionDate != raws[c.ID] {
			got := "<nil>"
			if c.LastTransactionDate != nil {
				got = *c.LastTransactionDate
			}
			t.Errorf("customer %s last_transaction_date = %q, want raw %q", c.ID, got, raws[c.ID])
		}
	}
}
