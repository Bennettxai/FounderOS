package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/pages/catalogdb"
	"github.com/rhl/businessos-backend/internal/founderos/pages/reference"
	"github.com/rhl/businessos-backend/internal/founderos/pages/roadmap"
)

func roadmapPatch(t *testing.T, r *gin.Engine, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPatch, "/api/founderos/pages/roadmap", strings.NewReader(body))
	req.Header.Set("Cookie", "session=ok")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestRoadmapPageServesTheBoardAndMarksRowsDone(t *testing.T) {
	pool := catalogdb.TestDB(t)
	ws := catalogdb.Workspace(t, pool, "founderos")
	ctx := context.Background()
	for _, q := range []string{
		`INSERT INTO founderos_departments (id, workspace_id, name, slug, color, ord) VALUES ('dept-tech', $1, 'TECH', 'tech', '#fff', 1)`,
		`INSERT INTO founderos_phases (id, workspace_id, number, title, items) VALUES ('phase-4', $1, 4, 'Dedicated Host', '[]')`,
		`INSERT INTO founderos_roadmap_items (id, workspace_id, title, quarter, status, department_id, description, phase_id)
			VALUES ('rm-auth', $1, 'Auth + remote access', '2026-Q4', 'next', 'dept-tech', 'Reach it from anywhere.', 'phase-4')`,
	} {
		if _, err := pool.Exec(ctx, q, ws); err != nil {
			t.Fatal(err)
		}
	}
	r := router(t, &Deps{Pool: pool, Board: connectors.NewRegistry()})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/founderos/pages/roadmap", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("no session: %d", w.Code)
	}
	w = catalogGet(t, r, "/api/founderos/pages/roadmap")
	var page roadmap.Page
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &page) != nil {
		t.Fatalf("got %d %s", w.Code, w.Body)
	}
	if len(page.Items) != 1 || page.Items[0].Title != "Auth + remote access" || page.Departments["dept-tech"] != "TECH" ||
		len(page.Phases) != 1 || page.Phases[0].Total != 1 || page.Shipped != 0 {
		t.Fatalf("page = %+v", page)
	}

	// v1 PATCH /api/roadmap: the row moves and the whole board comes back.
	w = roadmapPatch(t, r, `{"id":"rm-auth","status":"done"}`)
	var patched struct {
		Item  roadmap.Item `json:"item"`
		Board roadmap.Page `json:"board"`
	}
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &patched) != nil {
		t.Fatalf("patch: %d %s", w.Code, w.Body)
	}
	if patched.Item.Status != "done" || patched.Board.Shipped != 1 || patched.Board.Phases[0].Pct != 100 {
		t.Fatalf("patched = %+v", patched)
	}
	if w := roadmapPatch(t, r, `{"id":"rm-auth","status":"planned"}`); w.Code != http.StatusBadRequest {
		t.Fatalf("bad status: %d %s", w.Code, w.Body)
	}
	if w := roadmapPatch(t, r, `not json`); w.Code != http.StatusBadRequest {
		t.Fatalf("bad body: %d %s", w.Code, w.Body)
	}
	if w := roadmapPatch(t, r, `{"id":"nope","status":"done"}`); w.Code != http.StatusNotFound {
		t.Fatalf("unknown id: %d %s", w.Code, w.Body)
	}
}

func TestReferencePageServesTheDomains(t *testing.T) {
	pool := catalogdb.TestDB(t)
	ws := catalogdb.Workspace(t, pool, "founderos")
	if _, err := pool.Exec(context.Background(), `INSERT INTO founderos_domains (id, workspace_id, number, title, color, items)
		VALUES ('brm-8', $1, 8, 'Security', '#525252', '["No keys in repo"]')`, ws); err != nil {
		t.Fatal(err)
	}
	r := router(t, &Deps{Pool: pool, Board: connectors.NewRegistry()})
	w := catalogGet(t, r, "/api/founderos/pages/reference")
	var body struct {
		Domains []reference.Domain `json:"domains"`
	}
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &body) != nil {
		t.Fatalf("got %d %s", w.Code, w.Body)
	}
	if len(body.Domains) != 1 || body.Domains[0].Title != "Security" || body.Domains[0].Items[0] != "No keys in repo" {
		t.Fatalf("domains = %+v", body.Domains)
	}
}

func TestRoadmapAndReferenceAreHonestWhenTheDataIsMissing(t *testing.T) {
	r := router(t, &Deps{Board: connectors.NewRegistry()})
	for _, p := range []string{"/api/founderos/pages/roadmap", "/api/founderos/pages/reference"} {
		if w := catalogGet(t, r, p); w.Code != http.StatusServiceUnavailable {
			t.Fatalf("%s no pool: %d %s", p, w.Code, w.Body)
		}
	}
	if w := roadmapPatch(t, r, `{"id":"x","status":"done"}`); w.Code != http.StatusServiceUnavailable {
		t.Fatalf("patch no pool: %d %s", w.Code, w.Body)
	}
	pool := catalogdb.TestDB(t)
	r = router(t, &Deps{Pool: pool, Board: connectors.NewRegistry()})
	for _, p := range []string{"/api/founderos/pages/roadmap", "/api/founderos/pages/reference"} {
		if w := catalogGet(t, r, p); w.Code != http.StatusServiceUnavailable {
			t.Fatalf("%s no workspace: %d %s", p, w.Code, w.Body)
		}
	}
}
