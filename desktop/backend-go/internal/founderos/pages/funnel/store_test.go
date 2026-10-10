package funnel

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rhl/businessos-backend/internal/founderos/pages/pgtest"
)

func TestPgSeedWithoutAPoolOrWorkspacesIsAnError(t *testing.T) {
	if _, err := (&PgSeed{}).Journeys(context.Background(), ""); err == nil {
		t.Fatal("a missing pool must fail the read")
	}
	s := &PgSeed{Pool: &pgxpool.Pool{}, Workspaces: map[string]string{"vantage": "x"}}
	if _, err := s.Journeys(context.Background(), ""); err == nil || !strings.Contains(err.Error(), "launchpad-cohort") {
		t.Fatalf("err = %v", err)
	}
}

func TestPgSeedReadsJourneysWithTouches(t *testing.T) {
	pool := pgtest.ThrowawayDB(t)
	ctx := context.Background()
	ws := map[string]string{}
	for _, slug := range []string{"vantage", "launchpad-cohort"} {
		var id string
		if err := pool.QueryRow(ctx, `INSERT INTO workspaces (name, slug, owner_id) VALUES ($1,$1,'u1') RETURNING id::text`, slug).Scan(&id); err != nil {
			t.Fatal(err)
		}
		ws[slug] = id
	}
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`INSERT INTO founderos_funnel_contacts (id, workspace_id, name, venture, status, product, amount_usd, relationship, likelihood, email, person, created_at)
		VALUES ('fc-m', $1, 'Vantage Co', 'vantage', 'converted', 'Build', 5000, 'hot', 90, 'm@x.com', 'Mia', '2026-06-01T00:00:00Z'),
		       ('fc-a', $2, 'LC Lead', 'launchpad-cohort', 'engaged', NULL, NULL, 'warm', 50, NULL, NULL, '2026-06-05T00:00:00Z')`, ws["vantage"], ws["launchpad-cohort"])
	exec(`INSERT INTO founderos_funnel_touches (id, workspace_id, contact_id, seq, stage, channel, label, source, at) VALUES
		('t2', $1, 'fc-m', 2, 'converted', 'checkout', 'Paid', 'stripe', '2026-06-20'),
		('t1', $1, 'fc-m', 1, 'first_touch', 'organic', 'IG reel', 'trakyo', '2026-06-01')`, ws["vantage"])

	s := &PgSeed{Pool: pool, Workspaces: ws}
	all, err := s.Journeys(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 || all[0].ID != "fc-a" || all[1].ID != "fc-m" {
		t.Fatalf("journeys = %+v", all)
	}
	m := all[1]
	if m.CreatedAt != "2026-06-01" || *m.AmountUSD != 5000 || *m.Person != "Mia" || m.Phone != nil || len(m.Touches) != 2 || m.Touches[0].ID != "t1" || m.Touches[1].At != "2026-06-20" {
		t.Fatalf("vantage = %+v", m)
	}
	if all[0].AmountUSD != nil || len(all[0].Touches) != 0 || all[0].Touches == nil {
		t.Fatalf("aa = %+v", all[0])
	}
	only, err := s.Journeys(ctx, "vantage")
	if err != nil || len(only) != 1 || only[0].ID != "fc-m" {
		t.Fatalf("venture filter = %+v %v", only, err)
	}
}

// Ported from FounderOS v1 tests/funnel-archive.test.ts: the restored Attio /
// GoHighLevel journeys (db.funnel.archivedJourneys) read back whole, venture-
// filtered, ordered by id, only from the venture workspaces.
func TestPgArchiveReadsCompleteJourneys(t *testing.T) {
	if _, err := (&PgArchive{}).Journeys(context.Background(), ""); err == nil {
		t.Fatal("a missing pool must fail the read")
	}
	pool := pgtest.ThrowawayDB(t)
	ctx := context.Background()
	ws := map[string]string{}
	for _, slug := range []string{"vantage", "launchpad-cohort", "personal"} {
		var id string
		if err := pool.QueryRow(ctx, `INSERT INTO workspaces (name, slug, owner_id) VALUES ($1,$1,'u1') RETURNING id::text`, slug).Scan(&id); err != nil {
			t.Fatal(err)
		}
		ws[slug] = id
	}
	j := `{"id":"%s","name":"Historic lead","venture":"%s","status":"engaged","product":null,"amountUsd":null,"relationship":"warm","likelihood":50,
		"url":"https://app.attio.com/old","email":null,"phone":null,"person":null,"company":null,"role":null,"linkedin":null,"createdAt":"2026-09-20",
		"touches":[{"id":"%[1]s-t1","contactId":"%[1]s","seq":1,"stage":"engaged","channel":"crm","label":"Historic booked call","source":"attio","at":"2026-09-20"}]}`
	for _, r := range [][3]string{{"attio-b", "vantage", ws["vantage"]}, {"ghl-a", "launchpad-cohort", ws["launchpad-cohort"]}, {"stray", "vantage", ws["personal"]}} {
		if _, err := pool.Exec(ctx, `INSERT INTO founderos_funnel_archive (id, workspace_id, venture, journey) VALUES ($1, $2, $3, $4::jsonb)`,
			r[0], r[2], r[1], fmt.Sprintf(j, r[0], r[1])); err != nil {
			t.Fatal(err)
		}
	}
	a := &PgArchive{Pool: pool, Workspaces: ws}
	all, err := a.Journeys(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 || all[0].ID != "attio-b" || all[1].ID != "ghl-a" {
		t.Fatalf("archive = %+v", all)
	}
	if u := all[0].URL; u == nil || *u != "https://app.attio.com/old" || len(all[0].Touches) != 1 || all[0].Touches[0].Source != "attio" {
		t.Fatalf("journey = %+v", all[0])
	}
	only, err := a.Journeys(ctx, "launchpad-cohort")
	if err != nil || len(only) != 1 || only[0].ID != "ghl-a" {
		t.Fatalf("venture filter = %+v %v", only, err)
	}
}
