package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/founderos/pages/pgtest"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/zernio"
)

// contentThrowawayDB: a fresh migrated database on the bridge Postgres,
// dropped afterwards (never businessos_dev); skipped when Postgres is down.
func contentThrowawayDB(t *testing.T) *pgxpool.Pool {
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
	name := fmt.Sprintf("founderos_content_test_%d", rand.New(rand.NewSource(time.Now().UnixNano())).Intn(1_000_000))
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

func contentTestWorkspace(t *testing.T, pool *pgxpool.Pool, slug string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO workspaces (name, slug, owner_id) VALUES ($1, $1, 'u1') RETURNING id::text`, slug).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

type contentFakeZernio struct {
	posts []zernio.Post
	days  []zernio.PostDay
	err   error
}

func (f contentFakeZernio) RecentPosts(context.Context, int) ([]zernio.Post, error) {
	return f.posts, f.err
}
func (f contentFakeZernio) PostDays(context.Context) ([]zernio.PostDay, error) { return f.days, f.err }

func contentFixture(t *testing.T, z contentFakeZernio) (*pgxpool.Pool, http.Handler) {
	t.Helper()
	pool := contentThrowawayDB(t)
	ctx := context.Background()
	fo, pb := contentTestWorkspace(t, pool, "founderos"), contentTestWorkspace(t, pool, "personal")
	mustExec := func(q string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, q, args...); err != nil {
			t.Fatal(err)
		}
	}
	mustExec(`INSERT INTO founderos_departments (id, workspace_id, name, slug, color, ord) VALUES
		('dept-marketing-growth', $1, 'Marketing', 'marketing', '#fff', 1), ('dept-sales', $1, 'Sales', 'sales', '#fff', 2)`, fo)
	mustExec(`INSERT INTO founderos_agents (id, workspace_id, department_id, name, role, status, tier, description, model, tools, parent_id) VALUES
		('postly-publisher', $1, 'dept-marketing-growth', 'Zernio Publisher', 'Publisher', 'active', 'worker', 'posts', 'haiku', '["zernio"]', 'social-agent'),
		('social-agent', $1, 'dept-marketing-growth', 'Social Agent', 'Lead', 'active', 'lead', 'runs social', 'sonnet', '["zernio","arcads"]', NULL),
		('sales-agent', $1, 'dept-sales', 'Sales Agent', 'Lead', 'active', 'lead', '', '', '[]', NULL)`, fo)
	mustExec(`INSERT INTO founderos_agent_runs (id, workspace_id, agent_id, started_at, finished_at, ok) VALUES
		('r1', $1, 'social-agent', '2026-09-23T10:00:00Z', '2026-09-23T10:00:01Z', true),
		('r2', $1, 'social-agent', '2026-09-20T10:00:00Z', '2026-09-20T10:00:01Z', false),
		('r3', $1, 'sales-agent', '2026-09-22T10:00:00Z', '2026-09-22T10:00:01Z', true),
		('r4', $1, 'postly-publisher', '2026-06-01T10:00:00Z', '2026-06-01T10:00:01Z', true)`, fo)
	mustExec(`INSERT INTO founderos_lead_magnets (id, workspace_id, name, offer, url, status, captures, destination, source, launched_at, notes, origin) VALUES
		('agent-stack', $1, 'Agent Stack', 'tools', 'https://founderos-agent-stack.vercel.app', 'live', 'email', 'Beehiiv · newsletter', 'IG reel', '2026-08-12', '', 'seed'),
		('offer-doc', $1, 'Offer Doc', '', 'https://example.com/offer', 'draft', 'none', 'Beehiiv', 'IG reel', '2026-09-20', '', 'os')`, pb)

	prevZ, prevNow := contentZernio, contentNow
	contentZernio = func(*Deps) contentPostSource { return z }
	contentNow = func() time.Time { return time.Date(2026, 9, 24, 15, 0, 0, 0, time.UTC) }
	t.Cleanup(func() { contentZernio, contentNow = prevZ, prevNow })
	return pool, router(t, &Deps{Pool: pool})
}

func contentDo(t *testing.T, h http.Handler, method, path, body string) (int, map[string]any) {
	t.Helper()
	var rd *bytes.Reader
	if body != "" {
		rd = bytes.NewReader([]byte(body))
	} else {
		rd = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, rd)
	req.Header.Set("Cookie", "session=ok")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w.Code, out
}

func TestContentPageNeedsASession(t *testing.T) {
	r := router(t, &Deps{})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/founderos/pages/content", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("got %d", w.Code)
	}
}

func TestContentPageWithoutWorkspaceIs503NotEmpty(t *testing.T) {
	code, body := contentDo(t, router(t, &Deps{}), http.MethodGet, "/api/founderos/pages/content", "")
	if code != http.StatusServiceUnavailable || body["error"] == nil {
		t.Fatalf("got %d %v", code, body)
	}
}

func TestContentPageModel(t *testing.T) {
	_, h := contentFixture(t, contentFakeZernio{
		posts: []zernio.Post{{Platform: "instagram", Caption: "hi", URL: "https://x", Status: "published"}},
		days:  []zernio.PostDay{{Date: "2026-09-22", Platforms: []string{"instagram"}}, {Date: "2026-09-22", Platforms: []string{"tiktok"}}, {Date: "2026-05-01", Platforms: []string{"x"}}},
	})
	code, body := contentDo(t, h, http.MethodGet, "/api/founderos/pages/content", "")
	if code != 200 {
		t.Fatalf("%d %v", code, body)
	}
	crew := body["crew"].([]any)
	if len(crew) != 2 || crew[0].(map[string]any)["id"] != "social-agent" || crew[0].(map[string]any)["runnable"] != false {
		t.Fatalf("crew %v", crew)
	}
	if tools := crew[0].(map[string]any)["tools"].([]any); len(tools) != 2 {
		t.Fatalf("tools %v", tools)
	}
	if body["postsKnown"] != true || body["postsError"] != nil || len(body["posts"].([]any)) != 1 || body["pipelineActiveDays"] != 2.0 {
		t.Fatalf("posts %v", body)
	}
	if lm := body["leadMagnets"].([]any); len(lm) != 2 || lm[0].(map[string]any)["id"] != "offer-doc" {
		t.Fatalf("lead magnets newest first %v", lm)
	}
	v := body["volume"].(map[string]any)
	if v["postsInWindow"] != 2.0 || v["runsInWindow"] != 2.0 {
		t.Fatalf("volume %v", v)
	}
	if v["magnets"].(map[string]any)["live"] != 1.0 || body["today"] != "2026-09-24" {
		t.Fatalf("magnets %v", v["magnets"])
	}
}

func TestContentPageZernioDownIsUnknown(t *testing.T) {
	_, h := contentFixture(t, contentFakeZernio{err: errors.New("HTTP 503")})
	code, body := contentDo(t, h, http.MethodGet, "/api/founderos/pages/content", "")
	if code != 200 || body["postsKnown"] != false || body["postsError"] != "HTTP 503" || body["pipelineActiveDays"] != nil {
		t.Fatalf("%d %v", code, body)
	}
	if ins := body["volume"].(map[string]any)["insight"].(map[string]any); ins["display"] != "—" {
		t.Fatalf("insight %v", ins)
	}
	if posts := body["posts"].([]any); len(posts) != 0 {
		t.Fatalf("posts %v", posts)
	}
}

func TestContentLeadMagnetsListFiltersRowsButMeasuresAll(t *testing.T) {
	_, h := contentFixture(t, contentFakeZernio{})
	code, body := contentDo(t, h, http.MethodGet, "/api/founderos/pages/content/lead-magnets?status=live", "")
	if code != 200 || body["filter"] != "live" || len(body["rows"].([]any)) != 1 || body["volume"].(map[string]any)["total"] != 2.0 {
		t.Fatalf("%d %v", code, body)
	}
	_, body = contentDo(t, h, http.MethodGet, "/api/founderos/pages/content/lead-magnets?status=bogus", "")
	if body["filter"] != "all" || len(body["rows"].([]any)) != 2 {
		t.Fatalf("%v", body)
	}
}

func TestContentLeadMagnetCreatePatchDelete(t *testing.T) {
	pool, h := contentFixture(t, contentFakeZernio{})
	code, body := contentDo(t, h, http.MethodPost, "/api/founderos/pages/content/lead-magnets",
		`{"name":"Agent Stack","url":"https://founderos-agent-stack.vercel.app/v2","source":"IG reel"}`)
	if code != http.StatusCreated {
		t.Fatalf("%d %v", code, body)
	}
	lm := body["leadMagnet"].(map[string]any)
	if lm["id"] != "agent-stack-2" || lm["origin"] != "os" || lm["status"] != "live" || lm["captures"] != "email" || lm["launchedAt"] != "2026-09-24" {
		t.Fatalf("%v", lm)
	}
	var ws string
	_ = pool.QueryRow(context.Background(), `SELECT w.slug FROM founderos_lead_magnets m JOIN workspaces w ON w.id = m.workspace_id WHERE m.id = 'agent-stack-2'`).Scan(&ws)
	if ws != "personal" {
		t.Fatalf("stored in %q", ws)
	}
	if code, _ := contentDo(t, h, http.MethodPost, "/api/founderos/pages/content/lead-magnets", `{"name":"Broken","url":"not-a-url"}`); code != 400 {
		t.Fatalf("bad url %d", code)
	}

	code, body = contentDo(t, h, http.MethodPatch, "/api/founderos/pages/lead-magnets/agent-stack-2", `{"status":"archived","origin":"seed","id":"x"}`)
	if code != 200 || body["leadMagnet"].(map[string]any)["status"] != "archived" || body["leadMagnet"].(map[string]any)["origin"] != "os" || body["leadMagnet"].(map[string]any)["name"] != "Agent Stack" {
		t.Fatalf("%d %v", code, body)
	}
	if code, _ := contentDo(t, h, http.MethodPatch, "/api/founderos/pages/lead-magnets/agent-stack-2", `{"status":"sideways"}`); code != 400 {
		t.Fatalf("bad status %d", code)
	}
	if code, _ := contentDo(t, h, http.MethodPatch, "/api/founderos/pages/lead-magnets/nope", `{"status":"paused"}`); code != 404 {
		t.Fatalf("missing %d", code)
	}

	if code, body := contentDo(t, h, http.MethodDelete, "/api/founderos/pages/lead-magnets/agent-stack-2", ""); code != 200 || body["ok"] != true {
		t.Fatalf("%d %v", code, body)
	}
	if code, _ := contentDo(t, h, http.MethodDelete, "/api/founderos/pages/lead-magnets/agent-stack-2", ""); code != 404 {
		t.Fatalf("second delete %d", code)
	}
	if code, _ := contentDo(t, h, http.MethodPatch, "/api/founderos/pages/lead-magnets/agent-stack-2", `{"status":"live"}`); code != 404 {
		t.Fatalf("patch after delete %d", code)
	}
}
