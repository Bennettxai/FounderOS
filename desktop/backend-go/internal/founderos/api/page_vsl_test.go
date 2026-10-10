package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/pages/pgtest"
)

func TestVSLAssociationValidatesAndStoresInPostgres(t *testing.T) {
	post := func(r http.Handler, body string) (int, map[string]any) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, authed(http.MethodPost, "/api/founderos/pages/analytics/vsl/association", []byte(body)))
		var out map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &out)
		return w.Code, out
	}
	if code, _ := post(router(t, &Deps{Board: connectors.NewRegistry()}), `{"videoId":"x"}`); code != http.StatusServiceUnavailable {
		t.Fatalf("no Postgres: %d", code)
	}

	pool := pgtest.ThrowawayDB(t)
	ctx := context.Background()
	for _, slug := range []string{"launchpad-cohort", "founderos"} {
		if _, err := pool.Exec(ctx, `INSERT INTO workspaces (name, slug, owner_id) VALUES ($1,$1,'u1')`, slug); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pool.Exec(ctx, `INSERT INTO founderos_vsl_snapshots (video_id, date_from, date_to, workspace_id, captured_at, payload)
		SELECT 'vidA', '2025-08-10', '2026-09-24', id, now(), '{"videoId":"vidA"}' FROM workspaces WHERE slug = 'launchpad-cohort'`); err != nil {
		t.Fatal(err)
	}
	r := router(t, &Deps{Pool: pool, Board: connectors.NewRegistry()})
	for body, want := range map[string]int{`nope`: 400, `{}`: 400, `{"videoId":""}`: 400, `{"videoId":"missing"}`: 404} {
		if code, _ := post(r, body); code != want {
			t.Errorf("%s: %d, want %d", body, code, want)
		}
	}
	code, out := post(r, `{"videoId":"vidA"}`)
	if code != 200 || out["videoId"] != "vidA" {
		t.Fatalf("assign: %d %v", code, out)
	}
	var stored string
	if err := pool.QueryRow(ctx, `SELECT value FROM founderos_seed_meta WHERE key = 'vsl_ig_video'`).Scan(&stored); err != nil || stored != "vidA" {
		t.Fatalf("stored = %q %v", stored, err)
	}
	code, out = post(r, `{"videoId":null}`)
	if code != 200 || out["videoId"] != nil {
		t.Fatalf("clear: %d %v", code, out)
	}
}
