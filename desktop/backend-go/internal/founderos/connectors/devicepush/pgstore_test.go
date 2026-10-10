package devicepush

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/founderos/pages/pgtest"
)

func throwaway(t *testing.T) (*pgxpool.Pool, string) {
	t.Helper()
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, pgtest.AdminURL())
	if err == nil {
		err = admin.Ping(ctx)
	}
	if err != nil {
		t.Skipf("bridge Postgres unreachable: %v", err)
	}
	name := fmt.Sprintf("founderos_devicepush_test_%d", rand.New(rand.NewSource(time.Now().UnixNano())).Intn(1_000_000))
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = admin.Exec(context.Background(), "DROP DATABASE IF EXISTS "+name+" WITH (FORCE)")
		admin.Close()
	})
	pool, err := pgxpool.New(ctx, pgtest.DatabaseURL(name))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err := pgtest.RunMigrations(ctx, pool); err != nil {
		t.Fatal(err)
	}
	var ws string
	if err := pool.QueryRow(ctx, `INSERT INTO workspaces (name, slug, owner_id) VALUES ('FounderOS','founderos','u1') RETURNING id::text`).Scan(&ws); err != nil {
		t.Fatal(err)
	}
	return pool, ws
}

// Device pushes must survive a bridge restart: a restart that forgets them
// shows every device source as "not configured" until the next push.
func TestPgStoreKeepsTheLatestPushPerDeviceAcrossRestarts(t *testing.T) {
	pool, ws := throwaway(t)
	ctx := context.Background()
	at := time.Date(2026, 9, 30, 8, 0, 0, 0, time.UTC)
	s := NewPgStore(pool, ws)
	for i, dev := range []string{"alexs-macbook-pro", "mini", "alexs-macbook-pro"} {
		r := Received{Payload: Payload{Device: dev, Label: fmt.Sprintf("v%d", i), CapturedAt: at.Format(time.RFC3339)}, ReceivedAt: at.Add(time.Duration(i) * time.Minute)}
		if err := s.Save(ctx, r); err != nil {
			t.Fatal(err)
		}
	}
	restarted := NewPgStore(pool, ws) // a fresh process
	got, err := restarted.Latest(ctx)
	if err != nil || len(got) != 2 {
		t.Fatalf("latest = %+v err=%v", got, err)
	}
	if got[0].Payload.Device != "alexs-macbook-pro" || got[0].Payload.Label != "v2" || !got[0].ReceivedAt.Equal(at.Add(2*time.Minute)) {
		t.Fatalf("macbook = %+v (newest push wins)", got[0])
	}
	pool.Close()
	if _, err := restarted.Latest(ctx); err == nil {
		t.Fatal("an unreachable store must be an error, never an empty list")
	}
}
