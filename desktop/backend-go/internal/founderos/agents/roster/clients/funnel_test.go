package clients

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/founderos/pages/pgtest"
)

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
	name := fmt.Sprintf("founderos_clients_test_%d", rand.New(rand.NewSource(time.Now().UnixNano())).Intn(1_000_000))
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

func TestPgFunnelWithoutAPoolIsAnErrorNotEmpty(t *testing.T) {
	_, err := (&PgFunnel{}).Contacts(context.Background())
	if err == nil {
		t.Fatal("a missing pool must fail the read")
	}
}

func TestPgFunnelWithoutTheVentureWorkspacesIsAnError(t *testing.T) {
	f := &PgFunnel{Pool: &pgxpool.Pool{}, Workspaces: map[string]string{"vantage": "x"}}
	_, err := f.Contacts(context.Background())
	if err == nil || !strings.Contains(err.Error(), "launchpad-cohort") {
		t.Fatalf("err = %v", err)
	}
}

func TestPgFunnelReadsBothVentureWorkspacesNewestFirst(t *testing.T) {
	pool := throwawayDB(t)
	ctx := context.Background()
	ws := map[string]string{}
	for _, slug := range []string{"vantage", "launchpad-cohort", "founderos"} {
		var id string
		if err := pool.QueryRow(ctx, `INSERT INTO workspaces (name, slug, owner_id) VALUES ($1,$1,'u1') RETURNING id::text`, slug).Scan(&id); err != nil {
			t.Fatal(err)
		}
		ws[slug] = id
	}
	ins := func(id, slug, venture, status string, amount any, at string) {
		t.Helper()
		if _, err := pool.Exec(ctx, `INSERT INTO founderos_funnel_contacts (id, workspace_id, name, venture, status, amount_usd, created_at)
			VALUES ($1, $2, $1, $3, $4, $5, $6)`, id, ws[slug], venture, status, amount, at); err != nil {
			t.Fatal(err)
		}
	}
	ins("m1", "vantage", "vantage", "converted", 5000.0, "2026-09-01T00:00:00Z")
	ins("a1", "launchpad-cohort", "launchpad-cohort", "engaged", nil, "2026-09-10T00:00:00Z")
	ins("a0", "launchpad-cohort", "launchpad-cohort", "converted", 997.0, "2026-09-01T00:00:00Z")
	ins("stray", "founderos", "vantage", "converted", 1.0, "2026-09-20T00:00:00Z") // outside the venture workspaces

	got, err := (&PgFunnel{Pool: pool, Workspaces: ws}).Contacts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	ids := []string{}
	for _, c := range got {
		ids = append(ids, c.ID)
	}
	if strings.Join(ids, ",") != "a1,a0,m1" {
		t.Fatalf("ids = %v", ids)
	}
	if got[0].AmountUSD != nil || got[2].AmountUSD == nil || *got[2].AmountUSD != 5000 || got[2].Venture != "vantage" || got[2].Status != "converted" {
		t.Fatalf("rows = %+v", got)
	}
}
