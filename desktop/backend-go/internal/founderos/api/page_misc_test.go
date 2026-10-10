package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/founderos/pages/pgtest"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
)

func throwawayPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, pgtest.AdminURL())
	if err == nil {
		err = admin.Ping(ctx)
	}
	if err != nil {
		t.Skipf("bridge Postgres unreachable: %v", err)
	}
	name := fmt.Sprintf("founderos_api_test_%d", rand.New(rand.NewSource(time.Now().UnixNano())).Intn(1_000_000))
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
	for _, slug := range []string{"founderos", "personal"} {
		if _, err := pool.Exec(ctx, `INSERT INTO workspaces (name, slug, owner_id) VALUES ($1, $1, 'u1')`, slug); err != nil {
			t.Fatal(err)
		}
	}
	return pool
}

func wsID(t *testing.T, pool *pgxpool.Pool, slug string) string {
	var id string
	if err := pool.QueryRow(context.Background(), `SELECT id::text FROM workspaces WHERE slug=$1`, slug).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func TestDepartmentsToolsAndContactTags(t *testing.T) {
	pool := throwawayPool(t)
	ctx := context.Background()
	fo := wsID(t, pool, "founderos")
	_, _ = pool.Exec(ctx, `INSERT INTO founderos_departments (id, workspace_id, name, slug, color, ord) VALUES ('d2',$1,'Sales','sales','#f00',2),('d1',$1,'Tech','tech','#0f0',1)`, fo)
	_, _ = pool.Exec(ctx, `INSERT INTO founderos_tools (id, workspace_id, name, category, status, color) VALUES ('t1',$1,'Stripe','payments','connected','#fff')`, fo)
	r := router(t, &Deps{Board: connectors.NewRegistry(), Pool: pool})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, authed(http.MethodGet, "/api/founderos/pages/tools", nil))
	if w.Code != 200 || !bytes.Contains(w.Body.Bytes(), []byte(`"name":"Stripe"`)) {
		t.Fatalf("tools: %d %s", w.Code, w.Body)
	}

	for _, body := range []string{`{"person":"Rae","channel":"whatsapp","tag":"client","tier":4}`, `{"person":"","channel":"x","tag":"y","tier":1}`} {
		w = httptest.NewRecorder()
		r.ServeHTTP(w, authed(http.MethodPost, "/api/founderos/pages/contacts/tags", []byte(body)))
		if w.Code != 400 {
			t.Fatalf("invalid tag %s: %d", body, w.Code)
		}
	}
	for _, tier := range []int{1, 2} { // second POST upserts
		w = httptest.NewRecorder()
		r.ServeHTTP(w, authed(http.MethodPost, "/api/founderos/pages/contacts/tags", []byte(fmt.Sprintf(`{"person":"Rae","channel":"whatsapp","tag":"client","tier":%d}`, tier))))
		if w.Code != 200 {
			t.Fatalf("upsert: %d %s", w.Code, w.Body)
		}
	}
	w = httptest.NewRecorder()
	r.ServeHTTP(w, authed(http.MethodGet, "/api/founderos/pages/contacts/tags", nil))
	var tags struct {
		Tiers []struct{ Tier int } `json:"tiers"`
		Tags  []struct {
			Person string
			Tier   int
		} `json:"tags"`
	}
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &tags) != nil || len(tags.Tiers) != 3 || len(tags.Tags) != 1 || tags.Tags[0].Tier != 2 {
		t.Fatalf("tags: %d %s", w.Code, w.Body)
	}
	w = httptest.NewRecorder()
	r.ServeHTTP(w, authed(http.MethodDelete, "/api/founderos/pages/contacts/tags", []byte(`{"person":"Rae","channel":"whatsapp"}`)))
	var n int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM founderos_contact_tags`).Scan(&n)
	if w.Code != 200 || n != 0 {
		t.Fatalf("delete: %d rows=%d", w.Code, n)
	}
}
