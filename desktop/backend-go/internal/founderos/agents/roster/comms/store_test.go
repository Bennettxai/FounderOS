package comms

import (
	"context"
	"fmt"
	"math/rand"
	"slices"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/founderos/pages/pgtest"
)

// throwawayDB mirrors agents/pgstore_test.go: a fresh migrated database on
// the bridge Postgres, dropped afterwards; skipped when Postgres is down.
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
	name := fmt.Sprintf("founderos_comms_test_%d", rand.New(rand.NewSource(time.Now().UnixNano())).Intn(1_000_000))
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

func mkWorkspace(t *testing.T, pool *pgxpool.Pool, slug string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO workspaces (name, slug, owner_id) VALUES ($1, $1, 'u1') RETURNING id::text`, slug).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func TestPgDigestStoreIsScopedAndReadsNewest(t *testing.T) {
	pool := throwawayDB(t)
	ctx := context.Background()
	personal, other := mkWorkspace(t, pool, "personal"), mkWorkspace(t, pool, "founderos")
	st, elsewhere := NewPgDigestStore(pool, personal), NewPgDigestStore(pool, other)

	if _, found, err := st.Latest(ctx); err != nil || found {
		t.Fatalf("empty table: found %v err %v", found, err)
	}
	t0 := time.Date(2026, 9, 29, 9, 0, 0, 0, time.UTC)
	for i, at := range []time.Time{t0, t0.Add(24 * time.Hour), t0.Add(-24 * time.Hour)} {
		if err := st.Insert(ctx, fmt.Sprintf("d%d", i), at, []byte(fmt.Sprintf(`{"n":%d}`, i))); err != nil {
			t.Fatal(err)
		}
	}
	if err := elsewhere.Insert(ctx, "x", t0.Add(48*time.Hour), []byte(`{"n":99}`)); err != nil {
		t.Fatal(err)
	}
	payload, found, err := st.Latest(ctx)
	if err != nil || !found || string(payload) != `{"n": 1}` {
		t.Fatalf("latest = %s found %v err %v", payload, found, err)
	}

	if _, err := pool.Exec(ctx, `INSERT INTO founderos_digest_reads (key, workspace_id, read_at) VALUES
		('email|a|1', $1, now() - interval '1 day'), ('email|b|2', $1, now()), ('email|c|3', $2, now())`, personal, other); err != nil {
		t.Fatal(err)
	}
	keys, err := st.ClearedKeys(ctx)
	if err != nil || !slices.Equal(keys, []string{"email|b|2", "email|a|1"}) {
		t.Fatalf("keys = %v err %v", keys, err)
	}

	if _, err := pool.Exec(ctx, `INSERT INTO founderos_contact_tags (person, channel, workspace_id, tag, tier) VALUES
		('Riley', 'whatsapp', $1, 'student', 1), ('Mom', 'whatsapp', $1, 'family', 1), ('Nope', 'email', $2, 'family', 2)`, personal, other); err != nil {
		t.Fatal(err)
	}
	tags, err := st.ContactTags(ctx)
	if err != nil || len(tags) != 2 {
		t.Fatalf("tags = %+v err %v", tags, err)
	}
}

func TestPgStoresWithoutAWorkspaceRefuseHonestly(t *testing.T) {
	ctx := context.Background()
	st := NewPgDigestStore(nil, "")
	if _, _, err := st.Latest(ctx); err == nil {
		t.Fatal("no pool must be an error, not an empty backlog")
	}
	if err := st.Insert(ctx, "a", time.Now(), []byte(`{}`)); err == nil {
		t.Fatal("no pool must refuse the insert")
	}
	if _, err := NewPgSocialQueue(nil, "").QueuedPosts(ctx); err == nil {
		t.Fatal("no pool must be an error, not zero queued posts")
	}
}

func TestPgSocialQueueCountsQueuedOnly(t *testing.T) {
	pool := throwawayDB(t)
	ctx := context.Background()
	brand, other := mkWorkspace(t, pool, "personal"), mkWorkspace(t, pool, "founderos")
	if _, err := pool.Exec(ctx, `INSERT INTO founderos_social_posts (id, workspace_id, caption, platforms, status, created_at) VALUES
		('p1', $1, 'a', '["instagram"]', 'queued', now()),
		('p2', $1, 'b', '["tiktok"]', 'queued', now()),
		('p3', $1, 'c', '["x"]', 'published', now()),
		('p4', $2, 'd', '["x"]', 'queued', now())`, brand, other); err != nil {
		t.Fatal(err)
	}
	n, err := NewPgSocialQueue(pool, brand).QueuedPosts(ctx)
	if err != nil || n != 2 {
		t.Fatalf("queued = %d err %v", n, err)
	}
}
