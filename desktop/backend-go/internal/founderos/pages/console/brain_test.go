package console

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func fakeEngine(t *testing.T, key string, slugs ...string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("the console only reads engines, got %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer "+key {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/api/health":
			_, _ = w.Write([]byte(`{"status":"up"}`))
		case "/api/workspaces":
			body := `{"workspaces":[{"slug":"default"}`
			for _, s := range slugs {
				body += `,{"slug":"` + s + `"}`
			}
			_, _ = w.Write([]byte(body + `]}`))
		default:
			http.NotFound(w, r)
		}
	}))
}

func TestProbeEnginesReadsEachEngineAndItsWorkspaces(t *testing.T) {
	hub := fakeEngine(t, "k1", "founderos", "vantage")
	defer hub.Close()
	down := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusBadGateway) }))
	defer down.Close()

	b := ProbeEngines(context.Background(), []EngineRef{
		{Name: "hub", URL: hub.URL, Key: "k1", Homes: []string{"founderos", "vantage"}},
		{Name: "macbook", URL: down.URL, Key: "k2", Homes: []string{"personal"}},
		{Name: "mini", Homes: []string{"hermes"}}, // not staged here
	})
	if b.EnginesTotal != 2 || b.EnginesUp != 1 || b.Connected {
		t.Fatalf("counts = up %d / %d connected %v", b.EnginesUp, b.EnginesTotal, b.Connected)
	}
	if b.Workspaces == nil || *b.Workspaces != 2 {
		t.Fatalf("workspaces = %v (the engine's default catch-all is not a workspace)", b.Workspaces)
	}
	if len(b.Engines) != 3 {
		t.Fatalf("engines = %+v", b.Engines)
	}
	hubR, mbR, miniR := b.Engines[0], b.Engines[1], b.Engines[2]
	if !hubR.Up || hubR.Workspaces == nil || *hubR.Workspaces != 2 || len(hubR.Homes) != 2 {
		t.Fatalf("hub = %+v", hubR)
	}
	if mbR.Up || mbR.Workspaces != nil || mbR.State != "error" || mbR.Detail == "" {
		t.Fatalf("an unreachable engine is an error with unknown workspaces, not an empty brain: %+v", mbR)
	}
	if miniR.State != "not_configured" || miniR.Up {
		t.Fatalf("unstaged engine = %+v", miniR)
	}
}

func TestProbeEnginesAllUpIsConnected(t *testing.T) {
	hub := fakeEngine(t, "k1", "founderos")
	defer hub.Close()
	b := ProbeEngines(context.Background(), []EngineRef{{Name: "hub", URL: hub.URL, Key: "k1"}})
	if !b.Connected || b.EnginesUp != 1 || b.EnginesTotal != 1 {
		t.Fatalf("brain = %+v", b)
	}
	none := ProbeEngines(context.Background(), nil)
	if none.Connected || none.Workspaces != nil || none.EnginesTotal != 0 {
		t.Fatalf("no engines = %+v", none)
	}
}

func TestProbeEnginesRejectedKeyIsAnError(t *testing.T) {
	hub := fakeEngine(t, "right")
	defer hub.Close()
	b := ProbeEngines(context.Background(), []EngineRef{{Name: "hub", URL: hub.URL, Key: "wrong"}})
	if b.Engines[0].Up || b.Engines[0].State != "error" {
		t.Fatalf("rejected key = %+v", b.Engines[0])
	}
}

func healthEngine(t *testing.T, health string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/health":
			_, _ = w.Write([]byte(health))
		case "/api/workspaces":
			_, _ = w.Write([]byte(`{"workspaces":[{"slug":"a"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
}

// The engine's health checks stand where G-Brain's doctor score stood:
// the share of checks passing, averaged over the staged engines (a down
// engine scores 0), with the doctor's status words.
func TestProbeEnginesScoresHealthFromTheEnginesOwnChecks(t *testing.T) {
	good := healthEngine(t, `{"status":"up","checks":{"store":":ok","credential_key":":ok","migrations":":ok"},"degraded":[]}`)
	defer good.Close()
	b := ProbeEngines(context.Background(), []EngineRef{{Name: "hub", URL: good.URL}})
	if b.Health == nil || *b.Health != 100 || b.Status != "ok" || b.Engines[0].Health == nil || *b.Engines[0].Health != 100 {
		t.Fatalf("all checks ok = %+v", b)
	}

	warn := healthEngine(t, `{"status":"up","checks":{"store":":ok","credential_key":":missing","migrations":":ok","embedder":":ok"},"degraded":["credential_key"]}`)
	defer warn.Close()
	b = ProbeEngines(context.Background(), []EngineRef{{Name: "hub", URL: good.URL}, {Name: "mb", URL: warn.URL}})
	// (100 + 75) / 2 = 87.5 → 88
	if b.Health == nil || *b.Health != 88 || b.Status != "warnings" {
		t.Fatalf("one failing check = %v %q", b.Health, b.Status)
	}

	degradedOnly := healthEngine(t, `{"status":"up","checks":{"store":":ok"},"degraded":["rerank"]}`)
	defer degradedOnly.Close()
	b = ProbeEngines(context.Background(), []EngineRef{{Name: "hub", URL: degradedOnly.URL}})
	if b.Status != "warnings" {
		t.Fatalf("a degraded subsystem is a warning: %+v", b)
	}

	down := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusBadGateway) }))
	defer down.Close()
	b = ProbeEngines(context.Background(), []EngineRef{{Name: "hub", URL: good.URL}, {Name: "mb", URL: down.URL}})
	if b.Health == nil || *b.Health != 50 || b.Status != "warnings" || b.Engines[1].Health != nil {
		t.Fatalf("a down engine scores 0 in the mean: %v %q", b.Health, b.Status)
	}

	b = ProbeEngines(context.Background(), []EngineRef{{Name: "mb", URL: down.URL}})
	if b.Health != nil || b.Status != "offline" {
		t.Fatalf("every engine down = %v %q", b.Health, b.Status)
	}
	b = ProbeEngines(context.Background(), []EngineRef{{Name: "mini"}})
	if b.Health != nil || b.Status != "not configured" {
		t.Fatalf("nothing staged = %v %q", b.Health, b.Status)
	}

	// An engine that answers "up" with no checks at all is healthy, not unknown.
	bare := fakeEngine(t, "k", "x")
	defer bare.Close()
	b = ProbeEngines(context.Background(), []EngineRef{{Name: "hub", URL: bare.URL, Key: "k"}})
	if b.Health == nil || *b.Health != 100 || b.Status != "ok" {
		t.Fatalf("bare up = %v %q", b.Health, b.Status)
	}
}
