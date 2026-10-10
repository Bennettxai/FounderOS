package brain

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func fakeEngine(t *testing.T, key string) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+key {
			w.WriteHeader(401)
			return
		}
		switch r.URL.Path {
		case "/api/health":
			_, _ = w.Write([]byte(`{"status":"up","live":true,"ok?":true,"checks":{"store":":ok","migrations":":ok"},"degraded":[]}`))
		case "/api/graph":
			if r.URL.Query().Get("workspace") != "default:founderos" {
				w.WriteHeader(500)
				return
			}
			_, _ = w.Write([]byte(`{"contexts":[{"id":"a","node":"inbox","title":"Alpha","uri":"optimal://a","l0_abstract":"CHAT | a","genre":"chat","modified_at":"2026-09-30T06:40:38Z"},{"id":"b","node":"tech","title":"Beta","modified_at":"2026-09-30T06:40:26.169566Z"}],"edges":[{"source":"a","target":"b","relation":"cross_ref","weight":1.0}],"nodes":[]}`))
		default:
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestEngineClientGraphAndHealth(t *testing.T) {
	srv := fakeEngine(t, "k")
	c := NewEngineClient()
	e := Engine{Name: "hub", URL: srv.URL, Key: "k"}
	g := c.Graph(context.Background(), e, "founderos")
	if g.Err != "" || len(g.Contexts) != 2 || len(g.Edges) != 1 || g.Contexts[0].Abstract != "CHAT | a" || g.Contexts[1].Modified.IsZero() || g.Engine != "hub" {
		t.Fatalf("graph = %+v", g)
	}
	if bad := c.Graph(context.Background(), e, "vantage"); bad.Err == "" {
		t.Fatal("a 500 must surface as an error, never an empty workspace")
	}
	h := c.Health(context.Background(), e)
	if !h.Up || h.Err != "" || h.Checks["store"] != ":ok" {
		t.Fatalf("health = %+v", h)
	}
	wrong := c.Health(context.Background(), Engine{Name: "hub", URL: srv.URL, Key: "nope"})
	if wrong.Up || !strings.Contains(wrong.Err, "401") {
		t.Fatalf("auth failure = %+v", wrong)
	}
}

func TestBuildDoctor(t *testing.T) {
	d := BuildDoctor(
		[]EngineHealth{{Engine: "hub", Up: true, Checks: map[string]string{"store": ":ok"}}, {Engine: "macbook", Err: "dial tcp: refused"}},
		graphs(),
		[]string{"hermes"},
		RerankPassing,
	)
	if d.Connected || d.Status != "error" || d.HealthScore != nil {
		t.Fatalf("doctor = %+v (an engine down is an error; no invented score)", d)
	}
	find := func(name string) Check {
		for _, c := range d.Checks {
			if c.Name == name {
				return c
			}
		}
		t.Fatalf("no check %q in %+v", name, d.Checks)
		return Check{}
	}
	if c := find("hub engine"); c.Status != "ok" {
		t.Fatalf("hub = %+v", c)
	}
	if c := find("macbook engine"); c.Status != "error" || !strings.Contains(c.Message, "refused") {
		t.Fatalf("macbook = %+v", c)
	}
	if c := find("personal workspace"); c.Status != "error" {
		t.Fatalf("personal = %+v", c)
	}
	if c := find("hermes workspace"); c.Status != "warn" || !strings.Contains(c.Message, "not staged") {
		t.Fatalf("hermes = %+v", c)
	}
	if c := find("reranker"); c.Status != "ok" {
		t.Fatalf("reranker = %+v", c)
	}
	if !strings.Contains(d.Detail, "1/2 engines up") || !strings.Contains(d.Detail, "4 pages") {
		t.Fatalf("detail %q", d.Detail)
	}

	allUp := BuildDoctor([]EngineHealth{{Engine: "hub", Up: true}}, graphs()[:2], nil, RerankUnchecked)
	if !allUp.Connected || allUp.Status != "warn" {
		t.Fatalf("unknown reranker is a warning, not ok: %+v", allUp)
	}
	var rr Check
	for _, c := range allUp.Checks {
		if c.Name == "reranker" {
			rr = c
		}
	}
	if !strings.Contains(rr.Message, "engine order") {
		t.Fatalf("reranker check %+v", rr)
	}
}

// The engine rate-limits (429 with retry): a burst of per-workspace graph
// reads must back off and retry, not read the engine as unreachable.
func TestEngineClientRetriesARateLimit(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) <= 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":"rate_limited","retry_after_ms":10}`))
			return
		}
		_, _ = w.Write([]byte(`{"status":"up"}`))
	}))
	defer srv.Close()
	var out map[string]any
	if err := NewEngineClient().get(context.Background(), Engine{Name: "hub", URL: srv.URL}, "/api/health", &out); err != nil || out["status"] != "up" {
		t.Fatalf("err=%v out=%v calls=%d", err, out, calls)
	}
}

// A reranker that answers but fails the canary is noise, and the doctor says
// so: "up" alone was what let the broken :8081 model look healthy.
func TestDoctorSaysWhyResultsStayInEngineOrder(t *testing.T) {
	msg := func(st RerankState) Check {
		d := BuildDoctor([]EngineHealth{{Engine: "hub", Up: true}}, graphs()[:2], nil, st)
		for _, c := range d.Checks {
			if c.Name == "reranker" {
				return c
			}
		}
		t.Fatal("no reranker check")
		return Check{}
	}
	if c := msg(RerankFailing); c.Status != "warn" || !strings.Contains(c.Message, "fails the canary") {
		t.Fatalf("failing: %+v", c)
	}
	if c := msg(RerankUnreachable); c.Status != "warn" || !strings.Contains(c.Message, "offline") {
		t.Fatalf("unreachable: %+v", c)
	}
	if c := msg(RerankOff); c.Status != "warn" || !strings.Contains(c.Message, "BRAIN_RERANK=0") {
		t.Fatalf("off: %+v", c)
	}
	if c := msg(RerankPassing); c.Status != "ok" || !strings.Contains(c.Message, "passes the canary") {
		t.Fatalf("passing: %+v", c)
	}
}

// The engine copies every page that mentions an entity into its `team` node
// (and financial genres into `operation-revenue`): pipeline/router.ex's
// cross-cutting rules. The copy is indexed as a second context with the same
// title and file name; it is the same page, so the brain reads it once.
func TestEngineClientGraphReadsCrossReferenceCopiesOnce(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"contexts":[
			{"id":"p1","node":"tech","title":"Stripe","uri":"optimal://inbox/founderos/inbox/signals/2026-10-09-stripe.md"},
			{"id":"c1","node":"team","title":"Stripe","uri":"optimal://inbox/founderos/team/signals/2026-10-09-stripe.md"},
			{"id":"p2","node":"tech","title":"Stripe","uri":"optimal://inbox/founderos/inbox/signals/2026-10-09-stripe.md"},
			{"id":"p3","node":"knowledge-base","title":"Dana","uri":"optimal://inbox/founderos/inbox/signals/2026-10-09-dana.md"},
			{"id":"c3","node":"team","title":"Dana","uri":"optimal://inbox/founderos/team/signals/2026-10-09-dana.md"},
			{"id":"c4","node":"operation-revenue","title":"Invoice","uri":"optimal://x/operation-revenue/signals/2026-10-09-invoice.md"},
			{"id":"p4","node":"finance","title":"Invoice","uri":"optimal://x/inbox/signals/2026-10-09-invoice.md"},
			{"id":"t1","node":"team","title":"Standup","uri":"optimal://x/team/signals/2026-10-09-standup.md"}],
			"edges":[{"source":"p1","target":"p3","relation":"shared_entity"},{"source":"p1","target":"c3","relation":"cross_ref"}]}`))
	}))
	t.Cleanup(srv.Close)
	g := NewEngineClient().Graph(context.Background(), Engine{Name: "local", URL: srv.URL}, "founderos")
	var ids []string
	for _, c := range g.Contexts {
		ids = append(ids, c.ID)
	}
	// Two distinct pages may share a title (an agent and a tool both called
	// Stripe); a team page with no primary elsewhere is a page of its own.
	if got := strings.Join(ids, ","); got != "p1,p2,p3,p4,t1" {
		t.Fatalf("contexts = %s, want the copies c1, c3, c4 dropped", got)
	}
	if len(g.Edges) != 2 {
		t.Fatalf("edges = %v (edges to dropped copies fall away when the graph is built)", g.Edges)
	}
}
