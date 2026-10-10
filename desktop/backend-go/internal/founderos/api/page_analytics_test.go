package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/pages/analytics"
	"github.com/rhl/businessos-backend/internal/founderos/pages/metrics"
	"github.com/rhl/businessos-backend/internal/founderos/pages/pgtest"
)

func fakeAnalyticsReads(t *testing.T, r analyticsReads) {
	t.Helper()
	old := analyticsConnectorReads
	t.Cleanup(func() { analyticsConnectorReads = old })
	analyticsConnectorReads = func(context.Context, *Deps, time.Time) analyticsReads { return r }
}

type analyticsBody struct {
	RunVolume []analytics.DayCount   `json:"runVolume"`
	RunsKnown bool                   `json:"runsKnown"`
	Volume    analytics.Volume       `json:"volume"`
	Live      []analyticsTile        `json:"live"`
	Pending   []analytics.MetricTile `json:"pending"`
	Audience  struct {
		Known          bool                `json:"known"`
		TotalFollowers int64               `json:"totalFollowers"`
		Platforms      []analyticsPlatform `json:"platforms"`
	} `json:"audience"`
	Errors map[string]string `json:"errors"`
}

func TestAnalyticsWithoutPostgresSaysSoAndKeepsConnectorTiles(t *testing.T) {
	stubBrainPages(t, errors.New("no engines in the test"))
	leads := 12.0
	fakeAnalyticsReads(t, analyticsReads{Leads30d: &leads})
	var body analyticsBody
	if code := funnelGetJSON(t, "/api/founderos/pages/analytics", &body); code != 200 {
		t.Fatalf("code %d", code)
	}
	if body.RunsKnown || body.Audience.Known || body.Errors["runs"] == "" || body.Errors["social"] == "" {
		t.Fatalf("an unreadable store is surfaced: %+v", body)
	}
	if len(body.RunVolume) != 30 || len(body.Live) != 1 || body.Live[0].ID != "leads" || len(body.Pending) != 7 {
		t.Fatalf("tiles = %+v / %+v", body.Live, body.Pending)
	}
	if body.Volume.Insight.Value != 7 || body.Volume.Caption != "no audience snapshots yet" {
		t.Fatalf("volume = %+v", body.Volume)
	}
}

func TestAnalyticsReadsTheRunLogAndAudienceFromPostgres(t *testing.T) {
	pool := pgtest.ThrowawayDB(t)
	ctx := context.Background()
	ids := map[string]string{}
	for _, slug := range []string{"founderos", "personal", "launchpad-cohort"} {
		var id string
		if err := pool.QueryRow(ctx, `INSERT INTO workspaces (name, slug, owner_id) VALUES ($1,$1,'u1') RETURNING id::text`, slug).Scan(&id); err != nil {
			t.Fatal(err)
		}
		ids[slug] = id
	}
	now := time.Now().UTC()
	day := func(back int) string { return now.AddDate(0, 0, -back).Format("2006-01-02") }
	for _, q := range []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO founderos_agent_runs (id, workspace_id, agent_id, started_at, finished_at, ok) VALUES ('r1', $1, 'scout', now(), now(), true), ('r2', $1, 'scout', now() - interval '2 days', now(), false)`, []any{ids["founderos"]}},
		{`INSERT INTO founderos_social_accounts (platform, workspace_id, handle, ord) VALUES ('instagram', $1, '@b', 1), ('tiktok', $1, '@b', 2)`, []any{ids["personal"]}},
		{`INSERT INTO founderos_social_snapshots (platform, captured_on, workspace_id, followers, source) VALUES ('instagram', $2, $1, 1000, 'z'), ('instagram', $3, $1, 1100, 'z')`, []any{ids["personal"], day(10), day(0)}},
	} {
		if _, err := pool.Exec(ctx, q.sql, q.args...); err != nil {
			t.Fatal(err)
		}
	}
	fakeAnalyticsReads(t, analyticsReads{})
	w := httptest.NewRecorder()
	router(t, &Deps{Pool: pool, Board: connectors.NewRegistry()}).ServeHTTP(w, authed(http.MethodGet, "/api/founderos/pages/analytics", nil))
	var body analyticsBody
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &body) != nil {
		t.Fatalf("got %d %s", w.Code, w.Body)
	}
	if !body.RunsKnown || body.Volume.Runs.Total != 2 || body.Volume.Runs.Failed != 1 || body.RunVolume[29].Count != 1 {
		t.Fatalf("runs = %+v", body.Volume.Runs)
	}
	if !body.Audience.Known || body.Audience.TotalFollowers != 1100 || len(body.Audience.Platforms) != 2 {
		t.Fatalf("audience = %+v", body.Audience)
	}
	ig, tt := body.Audience.Platforms[0], body.Audience.Platforms[1]
	if ig.D7 == nil || *ig.D7 < 9.9 || *ig.D7 > 10.1 || len(ig.Bars) != 2 || *ig.Share != 100 {
		t.Fatalf("instagram = %+v", ig)
	}
	if tt.Followers != nil || tt.Bars != nil || tt.D7 != nil {
		t.Fatalf("an unmeasured platform is unknown, not zero: %+v", tt)
	}
	if body.Volume.Reach != 1100 || body.Live[0].ID != "audience" {
		t.Fatalf("reach = %+v live = %+v", body.Volume.Reach, body.Live)
	}
}

// stubBrainPages keeps analytics tests off the real engines.
func stubBrainPages(t *testing.T, err error) {
	t.Helper()
	prev := readBrainPages
	readBrainPages = func(context.Context, *Deps) (metrics.Read, error) { return metrics.Read{}, err }
	t.Cleanup(func() { readBrainPages = prev })
}

func TestAnalyticsMemoryPagesComeFromTheEngines(t *testing.T) {
	v := 1423.0
	prev := readBrainPages
	readBrainPages = func(context.Context, *Deps) (metrics.Read, error) {
		return metrics.Read{Value: &v, Source: "Optimal Engine · 2 engines · source packages"}, nil
	}
	t.Cleanup(func() { readBrainPages = prev })
	if r, err := readBrainPages(context.Background(), nil); err != nil || *r.Value != 1423 {
		t.Fatalf("stub: %v %v", r, err)
	}
}

// Prod's Brain-store Pages tile names one source ("GBrain") under its spark;
// the Optimal Engine tile keeps that one-line caption, not the engine detail.
func TestAnalyticsMemoryPagesTileNamesOneSource(t *testing.T) {
	v := 1423.0
	prev := readBrainPages
	readBrainPages = func(context.Context, *Deps) (metrics.Read, error) {
		return metrics.Read{Value: &v, Source: "Optimal Engine · 2 engines · source packages"}, nil
	}
	t.Cleanup(func() { readBrainPages = prev })
	val, src := brainTile(context.Background(), nil)
	if val == nil || *val != 1423 {
		t.Fatalf("value: %v", val)
	}
	if src != "Optimal Engine" {
		t.Fatalf("source = %q, want %q", src, "Optimal Engine")
	}
}

// v1's eight operating metrics: the engine tile is v1's Brain-store Pages as
// "Brain Pages" (G-Brain is retired), and the page carries no private-build
// VSL import (v1's analytics has no VSL card).
func TestAnalyticsTilesAndBodyMatchV1(t *testing.T) {
	v := 114.0
	in := operatingInputs(0, nil, 0, 0, analyticsReads{}, &v, "Optimal Engine")
	labels := map[string]string{}
	for _, m := range in {
		labels[m.ID] = m.Label
	}
	if len(in) != 8 || labels["brain"] != "Brain Pages" {
		t.Fatalf("tiles = %v", labels)
	}
	stubBrainPages(t, errors.New("no engines in the test"))
	fakeAnalyticsReads(t, analyticsReads{})
	var raw map[string]json.RawMessage
	if code := funnelGetJSON(t, "/api/founderos/pages/analytics", &raw); code != 200 {
		t.Fatalf("code %d", code)
	}
	if _, ok := raw["vsl"]; ok {
		t.Fatal("analytics body still carries the VSL import")
	}
}
