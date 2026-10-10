package doctor

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// fakeEngine answers the six GETs the doctor reads, the way the staging hub
// did on 2026-09-30 (one failing audit check: no verified backup).
func fakeEngine(t *testing.T, key string, audit string) *httptest.Server {
	t.Helper()
	bodies := map[string]string{
		"/api/health":     `{"status":"up","checks":{"store":":ok","credential_key":":ok","migrations":":ok"},"degraded":[]}`,
		"/api/metrics":    `{"counters":{"optimal_engine.search.query":619,"optimal_engine.intake.ingested":0},"uptime_ms":2969924}`,
		"/api/workspaces": `{"workspaces":[{"slug":"launchpad-cohort"},{"slug":"default"},{"slug":"founderos"}]}`,
		"/api/stores": `{"count":3,"stores":[
			{"id":"relational","status":"available","technology":"SQLite","row_count":40266,"table_counts":{"claims":721,"contexts":747,"episodes":252,"events":37814,"facts":0}},
			{"id":"vector","status":"available","technology":"SQLite BLOB","row_count":38279,"table_counts":{"chunk_embeddings":37559,"vectors":720}},
			{"id":"graph","status":"available","technology":"RocksDB","row_count":2184,"table_counts":{"edges":1463}}]}`,
		"/api/stores/audit": audit,
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("doctor sent %s %s: it must only read", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer "+key {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		body, ok := bodies[r.URL.Path]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		// A failing audit answers 503 with its checks in the body, as the
		// staging engines do.
		if r.URL.Path == "/api/stores/audit" && strings.Contains(body, `"ok":false`) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		_, _ = w.Write([]byte(body))
	}))
}

const hubAudit = `{"ok":false,"checks":[
	{"name":"sqlite_integrity","ok":true,"detail":"ok"},
	{"name":"migrations","ok":true,"detail":{"expected":52,"applied":52}},
	{"name":"verified_backup","ok":false,"detail":":no_verified_backup"},
	{"name":"rlm_runtime","ok":true,"detail":{"available":false,"optional":true}}],"failures":1}`

const greenAudit = `{"ok":true,"checks":[{"name":"sqlite_integrity","ok":true,"detail":"ok"}]}`

func TestReadParsesEveryEngineRead(t *testing.T) {
	srv := fakeEngine(t, "k1", hubAudit)
	defer srv.Close()
	got := Read(context.Background(), srv.Client(), []Engine{{Name: "hub", URL: srv.URL, Key: "k1"}})
	if len(got) != 1 {
		t.Fatalf("readings = %d", len(got))
	}
	e := got[0]
	if !e.Reachable || e.Health != "up" || e.Error != "" {
		t.Fatalf("engine = %+v", e)
	}
	if strings.Join(e.Workspaces, ",") != "launchpad-cohort,founderos" {
		t.Fatalf("workspaces = %v (the legacy default is not a workspace)", e.Workspaces)
	}
	if len(e.Stores) != 3 || e.Stores[0].ID != "relational" || e.Stores[0].Tables["claims"] != 721 {
		t.Fatalf("stores = %+v", e.Stores)
	}
	if e.Searches == nil || *e.Searches != 619 {
		t.Fatalf("searches = %v", e.Searches)
	}
	// 3 health checks + 4 audit checks, each tagged with its engine.
	if len(e.Checks) != 7 {
		t.Fatalf("checks = %+v", e.Checks)
	}
	byName := map[string]Check{}
	for _, c := range e.Checks {
		if c.Engine != "hub" {
			t.Fatalf("check %q not tagged with its engine", c.Name)
		}
		byName[c.Name] = c
	}
	if byName["verified_backup"].Status != "error" || !strings.Contains(byName["verified_backup"].Message, "no_verified_backup") {
		t.Fatalf("verified_backup = %+v", byName["verified_backup"])
	}
	if byName["health.store"].Status != "ok" || byName["migrations"].Message != `{"applied":52,"expected":52}` {
		t.Fatalf("checks = %+v", byName)
	}
}

func TestAnUnreachableEngineIsAnErrorNeverAnEmptyBrain(t *testing.T) {
	srv := fakeEngine(t, "right", hubAudit)
	defer srv.Close()
	got := Read(context.Background(), srv.Client(), []Engine{
		{Name: "hub", URL: srv.URL, Key: "wrong"},
		{Name: "macbook", URL: "http://127.0.0.1:1", Key: "k"},
	})
	for _, e := range got {
		if e.Reachable || e.Error == "" {
			t.Fatalf("%s reads reachable: %+v", e.Name, e)
		}
		if e.Stores != nil || e.Workspaces != nil || e.Checks != nil {
			t.Fatalf("%s invented data while unreachable: %+v", e.Name, e)
		}
	}
	if !strings.Contains(got[0].Error, "401") {
		t.Fatalf("hub error = %q", got[0].Error)
	}
}

func TestDegradedHealthFlagsAWarning(t *testing.T) {
	checks := healthChecks("hub", healthBody{Status: "up", Checks: map[string]string{"store": ":ok", "credential_key": ":missing"}, Degraded: []string{"embedder"}})
	st := map[string]string{}
	for _, c := range checks {
		st[c.Name] = c.Status
	}
	if st["health.store"] != "ok" || st["health.credential_key"] != "error" || st["degraded.embedder"] != "warn" {
		t.Fatalf("checks = %+v", checks)
	}
}

func TestScoreIsNullWithNoEngineAndGradedOtherwise(t *testing.T) {
	if s := Score(nil); s != nil {
		t.Fatalf("no engines scored %d", *s)
	}
	down := []EngineReading{{Name: "hub", Error: "dial refused"}}
	if s := Score(down); s != nil {
		t.Fatalf("unreachable engine scored %d", *s)
	}
	ok := func(n int) []Check {
		out := make([]Check, n)
		for i := range out {
			out[i] = Check{Status: "ok"}
		}
		return out
	}
	hub := EngineReading{Name: "hub", Reachable: true, Checks: append(ok(14), Check{Status: "error"})}
	if s := Score([]EngineReading{hub}); s == nil || *s != 93 {
		t.Fatalf("14/15 checks = %v, want 93", s)
	}
	// A second engine that is down halves the reach.
	if s := Score([]EngineReading{hub, down[0]}); s == nil || *s != 47 {
		t.Fatalf("one of two engines up = %v, want 47", s)
	}
	// A warn is half a pass.
	w := EngineReading{Reachable: true, Checks: []Check{{Status: "ok"}, {Status: "warn"}}}
	if s := Score([]EngineReading{w}); s == nil || *s != 75 {
		t.Fatalf("ok+warn = %v, want 75", s)
	}
}

func hubReading() EngineReading {
	return EngineReading{
		Name: "hub", URL: "http://127.0.0.1:4211", Reachable: true, Health: "up",
		Workspaces: []string{"launchpad-cohort", "founderos", "vantage", "personal"},
		Checks: []Check{
			{Engine: "hub", Name: "health.store", Status: "ok"},
			{Engine: "hub", Name: "sqlite_integrity", Status: "ok"},
			{Engine: "hub", Name: "verified_backup", Status: "error", Message: ":no_verified_backup"},
		},
		Stores: []Store{
			{ID: "relational", Status: "available", RowCount: 40266, Tables: map[string]int64{"claims": 721, "contexts": 747, "facts": 0, "episodes": 252, "events": 37814}},
			{ID: "full_text", Status: "available", RowCount: 747, Tables: map[string]int64{"contexts_fts": 747}},
			{ID: "vector", Status: "available", RowCount: 38279, Tables: map[string]int64{"chunk_embeddings": 37559, "vectors": 720}},
			{ID: "graph", Status: "available", RowCount: 2184, Tables: map[string]int64{"edges": 1463}},
		},
	}
}

var now = time.Date(2026, 9, 30, 15, 0, 0, 0, time.UTC)

func TestVolumeHeadlineChipsAndMetersAreHonest(t *testing.T) {
	axes := []PillarAxis{{ID: "a", Score: 60}, {ID: "b", Score: 80}}
	v := BuildVolume(VolumeInput{Engines: []EngineReading{hubReading()}, Axes: axes, Now: now, Loc: time.UTC})
	if v.Headline == nil || *v.Headline != 67 {
		t.Fatalf("headline = %v", v.Headline)
	}
	if v.Counts != (Counts{OK: 2, Warn: 0, Fail: 1, Total: 3}) {
		t.Fatalf("counts = %+v", v.Counts)
	}
	if len(v.Chips) != 2 || v.Chips[0].Text != "2 passing" || v.Chips[1].Text != "1 failing" {
		t.Fatalf("chips = %+v", v.Chips)
	}
	m := map[string]Meter{}
	for _, x := range v.Meters {
		m[strings.Split(x.Label, " (")[0]] = x
	}
	if f := m["Checks passing"].Frac; f == nil || *f < 0.66 || *f > 0.67 {
		t.Fatalf("checks meter = %+v", m["Checks passing"])
	}
	// Production's four meters, in its order: the storage layers meter
	// counts the engines plus their logical stores (1 engine + 4 stores).
	if len(v.Meters) != 4 {
		t.Fatalf("meters = %+v, want production's four", v.Meters)
	}
	sl := m["Storage layers live"]
	if sl.Frac == nil || *sl.Frac != 1 || sl.Label != "Storage layers live (5/5)" || sl.Display != "100%" {
		t.Fatalf("storage layers meter = %+v", sl)
	}
	if v.Meters[2].Label != sl.Label || v.Meters[3].Label != m["Pillar health"].Label {
		t.Fatalf("meter order = %+v", v.Meters)
	}
	if f := m["Pillar health"].Frac; f == nil || *f != 0.7 {
		t.Fatalf("pillar meter = %+v", m["Pillar health"])
	}
	if v.Claims == nil || *v.Claims != 721 || v.Facts == nil || *v.Facts != 0 || v.Contexts == nil || *v.Contexts != 747 {
		t.Fatalf("claims/facts/contexts = %v %v %v", v.Claims, v.Facts, v.Contexts)
	}
	// prod's one-line caption: "6 checks · 382 pages on disk"
	if v.Caption != "3 checks · 747 engine entries stored" { // v1: "N checks · N pages on disk"; the engine stores cross-reference copies too, so these are entries, not distinct pages (Brain counts those)
		t.Fatalf("caption = %q", v.Caption)
	}
	// prod: "18 folders · 6 pillars · 3/4 layers live"; the store's shape is its tables
	if v.Foot != "7 tables · 2 pillars · 5/5 layers live" {
		t.Fatalf("foot = %q", v.Foot)
	}
}

func TestVolumeWithNoEngineInventsNothing(t *testing.T) {
	v := BuildVolume(VolumeInput{Engines: []EngineReading{{Name: "hub", Error: "dial refused"}}, Now: now, Loc: time.UTC})
	if v.Headline != nil || v.Claims != nil || v.Facts != nil || v.Contexts != nil {
		t.Fatalf("invented numbers: %+v", v)
	}
	if len(v.Chips) != 1 || v.Chips[0].Text != "unreachable" || v.Chips[0].Tone != "err" {
		t.Fatalf("chips = %+v", v.Chips)
	}
	for _, m := range v.Meters {
		// The engine row was checked and is down, so 0 layers live is measured;
		// the stores behind it were never checked and say so in the label.
		if strings.HasPrefix(m.Label, "Storage layers live") {
			if m.Frac == nil || *m.Frac != 0 || m.Label != "Storage layers live (0/1 checked · 4 unchecked)" {
				t.Fatalf("storage layers meter = %+v", m)
			}
			continue
		}
		if m.Frac != nil {
			t.Fatalf("meter %q has a value with nothing measured: %+v", m.Label, m)
		}
	}
	if v.Insight.Value != nil || !strings.Contains(v.Insight.Headline, "No engine answered") {
		t.Fatalf("insight = %+v", v.Insight)
	}
	if v.Store.Total != nil || len(v.Store.Cols) != 0 {
		t.Fatalf("store = %+v", v.Store)
	}
}

func TestInsightCountsFlaggedChecksAndSaysAllGreen(t *testing.T) {
	v := BuildVolume(VolumeInput{Engines: []EngineReading{hubReading()}, Now: now, Loc: time.UTC})
	if v.Insight.Value == nil || *v.Insight.Value != 1 || v.Insight.Body != "verified_backup · hub" {
		t.Fatalf("insight = %+v", v.Insight)
	}
	g := hubReading()
	g.Checks = g.Checks[:2]
	v = BuildVolume(VolumeInput{Engines: []EngineReading{g}, Now: now, Loc: time.UTC})
	if v.Insight.Value == nil || *v.Insight.Value != 0 || v.Insight.Headline != "Every check passed." {
		t.Fatalf("insight = %+v", v.Insight)
	}
}

func TestStepLineBucketsBrainRunsByLocalDayOldestFirst(t *testing.T) {
	runs := []RunLite{
		{FinishedAt: now.Add(-1 * time.Hour), OK: true},
		{FinishedAt: now.Add(-2 * time.Hour), OK: false},
		{FinishedAt: now.Add(-49 * time.Hour), OK: true},
		{FinishedAt: now.Add(-30 * 24 * time.Hour), OK: true}, // outside the window
	}
	v := BuildVolume(VolumeInput{Runs: runs, Now: now, Loc: time.UTC, Days: 14})
	if len(v.Series) != 14 || v.Series[13].Label != "Sep 30" || v.Series[13].Count != 2 || v.Series[11].Count != 1 {
		t.Fatalf("series = %+v", v.Series)
	}
	if v.RunsInWindow != 3 || v.FailedInWindow != 1 {
		t.Fatalf("window = %d runs, %d failed", v.RunsInWindow, v.FailedInWindow)
	}
}

func TestStoreMatrixIsTheKnowledgeTablesSummedAcrossEngines(t *testing.T) {
	mb := hubReading()
	mb.Name = "macbook"
	mb.Stores[0].Tables = map[string]int64{"claims": 672, "contexts": 674, "facts": 0, "events": 175296}
	v := BuildVolume(VolumeInput{Engines: []EngineReading{hubReading(), mb}, Now: now, Loc: time.UTC})
	if v.Store.Total == nil || *v.Store.Total != 747+674 {
		t.Fatalf("store total = %v", v.Store.Total)
	}
	labels := []string{}
	for _, c := range v.Store.Cols {
		labels = append(labels, c.Label)
		if c.Label == "events" {
			t.Fatal("telemetry events are not knowledge")
		}
	}
	if labels[0] != "chunks" || v.Store.Cols[0].Count != 2*37559 {
		t.Fatalf("cols = %+v", v.Store.Cols)
	}
	// six columns keep the longer table names inside the card
	if len(v.Store.Cols) > 6 {
		t.Fatalf("more than six columns: %d", len(v.Store.Cols))
	}
	if v.Store.Tables != 7 {
		t.Fatalf("tables = %d, want every knowledge table reported (7)", v.Store.Tables)
	}
	if v.Store.Top == nil || v.Store.Top.Name != "chunks" {
		t.Fatalf("top = %+v", v.Store.Top)
	}
}

func TestLayersListEnginesThenTheLogicalStores(t *testing.T) {
	down := EngineReading{Name: "macbook", URL: "http://127.0.0.1:4210", Error: "dial refused"}
	ls := Layers([]EngineReading{hubReading(), down})
	if len(ls) != 6 {
		t.Fatalf("layers = %+v", ls)
	}
	if ls[0].Name != "hub engine" || ls[0].State != "connected" || ls[0].Val != "LIVE" {
		t.Fatalf("hub = %+v", ls[0])
	}
	if ls[1].State != "error" || ls[1].Val != "UNREACHABLE" {
		t.Fatalf("macbook = %+v", ls[1])
	}
	if ls[2].Name != "relational" || ls[2].Val != "40,266 rows" || ls[2].State != "connected" {
		t.Fatalf("relational = %+v", ls[2])
	}
	if ls[4].Name != "vector" || !strings.Contains(ls[4].Sub, "1 of 2 engines") {
		t.Fatalf("vector = %+v", ls[4])
	}
	none := Layers([]EngineReading{down})
	for _, l := range none[1:] {
		if l.State == "connected" {
			t.Fatalf("store %q reads live with no engine answering", l.Name)
		}
	}
}

func TestPillarAxesFollowGraphOrderAndClamp(t *testing.T) {
	depts := []Department{{ID: "dept-comms", Name: "Communications"}, {ID: "dept-sales", Name: "Sales"}, {ID: "dept-x", Name: "X"}}
	agents := []Agent{
		{ID: "s1", DepartmentID: "dept-sales", Status: "active"},
		{ID: "s2", DepartmentID: "dept-sales", Status: "active"},
		{ID: "c1", DepartmentID: "dept-comms", Status: "idle"},
	}
	tasks := []SopTask{{DepartmentID: "dept-sales"}, {DepartmentID: "dept-sales"}}
	latest := map[string]time.Time{"s1": now.Add(-30 * time.Minute), "c1": now.Add(-10 * 24 * time.Hour)}
	axes := PillarAxes(depts, agents, tasks, latest, now)
	if len(axes) != 3 || axes[0].ID != "dept-sales" || axes[1].ID != "dept-comms" || axes[2].ID != "dept-x" {
		t.Fatalf("order = %+v", axes)
	}
	if axes[0].Score != 100 || axes[0].Roster != 100 || axes[0].Freshness != 100 || axes[0].SOP != 100 {
		t.Fatalf("sales = %+v", axes[0])
	}
	// idle roster, week-plus-old run, no SOPs: 30*0.15 = 4.5 → clamped to 15.
	if axes[1].Score != 15 || axes[1].Freshness != 15 {
		t.Fatalf("comms = %+v", axes[1])
	}
	again := PillarAxes(depts, agents, tasks, latest, now)
	b1, _ := json.Marshal(axes)
	b2, _ := json.Marshal(again)
	if string(b1) != string(b2) {
		t.Fatal("not deterministic")
	}
}

func TestInsightFoldsOneCheckFlaggedOnEveryEngine(t *testing.T) {
	a, b := hubReading(), hubReading()
	b.Name = "macbook"
	for i := range b.Checks {
		b.Checks[i].Engine = "macbook"
	}
	v := BuildVolume(VolumeInput{Engines: []EngineReading{a, b}, Now: now, Loc: time.UTC})
	// prod reads "resolver_health · connection": the check names, once each
	if v.Insight.Body != "verified_backup · hub, macbook" {
		t.Fatalf("insight body = %q", v.Insight.Body)
	}
	if v.Insight.Value == nil || *v.Insight.Value != 2 {
		t.Fatalf("insight value = %v", v.Insight.Value)
	}
}
