package brainimport

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/rhl/businessos-backend/internal/founderos/topology"
)

const fixtureTopology = `
version: 1
engines:
  macbook: {tier: device}
  hub: {tier: shared}
workspaces:
  - {slug: founderos, home: hub, class: business}
  - {slug: vantage, name: Vantage, home: hub, class: business}
  - {slug: personal, name: Personal, home: macbook, class: personal}
brain_store:
  default: founderos
  routes: {conversations: personal, vantage: vantage}
retired: {default: founderos}
`

var long = strings.Repeat("real content line. ", 20) // > MinChars

func writeStore(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"conversations/2026-09-01-chat.md": "# Chat\n" + long,
		"meetings/2026-09-02-call.md":      "# Call\n" + long,
		"sops/onboarding.md":               "# SOP\n" + long,
		"vantage/offer.md":                 "# Offer\n" + long,
		"README.md":                        "# Readme\n" + long,
		"tools/empty.md":                   "# stub",
		".git/HEAD.md":                     long,
		"notes.txt":                        long,
	}
	for rel, body := range files {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func topo(t *testing.T) *topology.Topology {
	t.Helper()
	tp, err := topology.Parse([]byte(fixtureTopology))
	if err != nil {
		t.Fatal(err)
	}
	return tp
}

func TestPlanRoutesPagesByTopology(t *testing.T) {
	items, skipped, err := Plan(writeStore(t), topo(t))
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]Item{}
	for _, it := range items {
		got[it.Rel] = it
	}
	want := map[string]struct{ ws, engine, genre string }{
		"conversations/2026-09-01-chat.md": {"personal", "macbook", "note"},
		"meetings/2026-09-02-call.md":      {"founderos", "hub", "transcript"},
		"sops/onboarding.md":               {"founderos", "hub", "sop"},
		"vantage/offer.md":                 {"vantage", "hub", "note"},
		"README.md":                        {"founderos", "hub", "note"},
	}
	if len(got) != len(want) {
		t.Fatalf("planned %d pages, want %d: %v", len(got), len(want), items)
	}
	for rel, w := range want {
		it, ok := got[rel]
		if !ok {
			t.Errorf("%s not planned", rel)
			continue
		}
		if it.Workspace != w.ws || it.Engine != w.engine || it.Genre != w.genre {
			t.Errorf("%s → ws=%s engine=%s genre=%s, want %s/%s/%s", rel, it.Workspace, it.Engine, it.Genre, w.ws, w.engine, w.genre)
		}
		if len(it.Hash) != 64 {
			t.Errorf("%s hash %q", rel, it.Hash)
		}
	}
	if skipped != 1 {
		t.Errorf("skipped short pages = %d, want 1 (tools/empty.md)", skipped)
	}
}

type fakeEngine struct {
	mu       sync.Mutex
	requests []map[string]any
	auth     []string
	fail     string // title substring that returns 500
}

func (f *fakeEngine) server(t *testing.T) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/ingest" {
			http.NotFound(w, r)
			return
		}
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		f.mu.Lock()
		f.requests = append(f.requests, body)
		f.auth = append(f.auth, r.Header.Get("Authorization"))
		f.mu.Unlock()
		if f.fail != "" && strings.Contains(body["title"].(string), f.fail) {
			http.Error(w, `{"error":"boom"}`, http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(`{"ok":true,"signal_id":"s1","source_package_id":"p1"}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestRunIngestsToHomeEngineAndIsIdempotent(t *testing.T) {
	store := writeStore(t)
	items, _, err := Plan(store, topo(t))
	if err != nil {
		t.Fatal(err)
	}
	hub, mac := &fakeEngine{}, &fakeEngine{}
	hubSrv, macSrv := hub.server(t), mac.server(t)
	engines := map[string]Endpoint{
		"hub":     {URL: hubSrv.URL, Key: "hub-key"},
		"macbook": {URL: macSrv.URL, Key: "mac-key"},
	}
	ledgerPath := filepath.Join(t.TempDir(), "ledger.json")

	rep, err := Run(items, engines, ledgerPath, false)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Ingested != 5 || rep.Duplicates != 0 || len(rep.Errors) != 0 {
		t.Fatalf("first run report = %+v", rep)
	}
	if len(mac.requests) != 1 || len(hub.requests) != 4 {
		t.Fatalf("mac got %d, hub got %d; want 1 and 4", len(mac.requests), len(hub.requests))
	}
	if mac.requests[0]["workspace"] != "default:personal" {
		t.Errorf("conversation went to workspace %v", mac.requests[0]["workspace"])
	}
	if mac.auth[0] != "Bearer mac-key" || hub.auth[0] != "Bearer hub-key" {
		t.Errorf("auth headers = %v / %v", mac.auth, hub.auth)
	}
	if mac.requests[0]["extract_claims"] != true {
		t.Errorf("extract_claims not set: %v", mac.requests[0])
	}

	again, err := Run(items, engines, ledgerPath, false)
	if err != nil {
		t.Fatal(err)
	}
	if again.Ingested != 0 || again.Duplicates != 5 {
		t.Fatalf("re-run report = %+v, want 0 ingested / 5 duplicates", again)
	}
	if len(mac.requests)+len(hub.requests) != 5 {
		t.Fatalf("re-run sent %d more requests", len(mac.requests)+len(hub.requests)-5)
	}
}

func TestRunDryRunSendsNothingAndRecordsNothing(t *testing.T) {
	items, _, _ := Plan(writeStore(t), topo(t))
	hub := &fakeEngine{}
	srv := hub.server(t)
	ledgerPath := filepath.Join(t.TempDir(), "ledger.json")
	engines := map[string]Endpoint{"hub": {URL: srv.URL}, "macbook": {URL: srv.URL}}

	rep, err := Run(items, engines, ledgerPath, true)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Planned != 5 || rep.Ingested != 0 {
		t.Fatalf("dry-run report = %+v", rep)
	}
	if len(hub.requests) != 0 {
		t.Fatalf("dry run sent %d requests", len(hub.requests))
	}
	if _, err := os.Stat(ledgerPath); !os.IsNotExist(err) {
		t.Fatalf("dry run wrote a ledger: %v", err)
	}
}

func TestRunReportsFailuresAndRetriesThemNextTime(t *testing.T) {
	items, _, _ := Plan(writeStore(t), topo(t))
	hub, mac := &fakeEngine{fail: "Offer"}, &fakeEngine{}
	hubSrv, macSrv := hub.server(t), mac.server(t)
	engines := map[string]Endpoint{"hub": {URL: hubSrv.URL}, "macbook": {URL: macSrv.URL}}
	ledgerPath := filepath.Join(t.TempDir(), "ledger.json")

	rep, err := Run(items, engines, ledgerPath, false)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Ingested != 4 || len(rep.Errors) != 1 || rep.Errors[0].Rel != "vantage/offer.md" {
		t.Fatalf("report = %+v", rep)
	}
	hub.fail = ""
	rep, _ = Run(items, engines, ledgerPath, false)
	if rep.Ingested != 1 || rep.Duplicates != 4 {
		t.Fatalf("retry report = %+v, want the failed page retried", rep)
	}
}

func TestRunRefusesAnEngineWithNoEndpoint(t *testing.T) {
	items, _, _ := Plan(writeStore(t), topo(t))
	_, err := Run(items, map[string]Endpoint{"hub": {URL: "http://127.0.0.1:1"}}, filepath.Join(t.TempDir(), "l.json"), false)
	if err == nil || !strings.Contains(err.Error(), "macbook") {
		t.Fatalf("want an error naming the missing macbook endpoint, got %v", err)
	}
}

func TestRunRetriesWhenTheEngineRateLimits(t *testing.T) {
	items, _, _ := Plan(writeStore(t), topo(t))
	var mu sync.Mutex
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls++
		n := calls
		mu.Unlock()
		if n%2 == 1 { // every first attempt is throttled
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":"rate_limited","retry_after_ms":5}`))
			return
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(srv.Close)
	engines := map[string]Endpoint{"hub": {URL: srv.URL}, "macbook": {URL: srv.URL}}

	rep, err := Run(items, engines, filepath.Join(t.TempDir(), "l.json"), false)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Ingested != 5 || len(rep.Errors) != 0 {
		t.Fatalf("report = %+v, want all 5 ingested after retries", rep)
	}
	if calls != 10 {
		t.Fatalf("calls = %d, want 10 (one throttled + one ok per page)", calls)
	}
}
