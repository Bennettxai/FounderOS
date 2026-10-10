package tech

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rhl/businessos-backend/internal/founderos/agents"
	"github.com/rhl/businessos-backend/internal/founderos/agents/roster"
	"github.com/rhl/businessos-backend/internal/founderos/topology"
)

func TestAgentsRoster(t *testing.T) {
	list := Agents(roster.Deps{})
	want := []string{"data-agent", "markdown-auditor", "vector-auditor", "stack-monitor"}
	if len(list) != len(want) {
		t.Fatalf("got %d agents", len(list))
	}
	for i, a := range list {
		m := a.Meta()
		if m.ID != want[i] || m.DepartmentID != "dept-tech" || m.Name == "" || m.Description == "" {
			t.Errorf("agent %d meta = %+v", i, m)
		}
	}
	if _, ok := list[0].(agents.Responder); !ok {
		t.Error("data-agent answers broadcasts by querying memory")
	}
	for _, a := range list[1:] {
		if _, ok := a.(agents.Responder); ok {
			t.Errorf("%s has no broadcast voice in FounderOS v1", a.Meta().ID)
		}
	}
	// Registering them must not panic on duplicate ids.
	agents.New(agents.NewMemStore(), list...)
}

func TestStagedEnginesFromEnvAndKeys(t *testing.T) {
	topo := `
version: 1
engines:
  hub: {tier: shared, url_env: OE_HUB_URL, key_env: OE_HUB_KEY}
  macbook: {tier: device, url_env: OE_MACBOOK_URL, key_env: OE_MACBOOK_KEY}
  mini: {tier: device, url_env: OE_MINI_URL, key_env: OE_MINI_KEY}
  cloud: {tier: shared, deferred: true}
workspaces:
  - {slug: founderos, home: hub, class: business}
  - {slug: personal, home: macbook, class: personal}
  - {slug: hermes, home: mini, class: device}
brain_store: {default: founderos, routes: {}}
`
	home := t.TempDir()
	t.Setenv("OE_HUB_URL", "http://hub.example:4211")
	t.Setenv("OE_HUB_KEY", "hub-key")
	t.Setenv("OE_MACBOOK_URL", "")
	t.Setenv("OE_MACBOOK_KEY", "")
	t.Setenv("OE_MINI_URL", "")
	t.Setenv("OE_MINI_KEY", "")
	writeKey(t, home, "oe-macbook.key", "mac-key\n")

	got := stagedEngines(mustParse(t, topo), home)
	if len(got) != 2 {
		t.Fatalf("want hub (env) and macbook (key file), got %+v", got)
	}
	if got[0].Name != "hub" || got[0].URL != "http://hub.example:4211" || got[0].Key != "hub-key" {
		t.Errorf("hub = %+v", got[0])
	}
	if got[1].Name != "macbook" || got[1].URL != "http://127.0.0.1:4210" || got[1].Key != "mac-key" {
		t.Errorf("macbook = %+v", got[1])
	}
}

func writeKey(t *testing.T, home, name, value string) {
	t.Helper()
	dir := filepath.Join(home, ".founderos-bridge", "keys")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(value), 0o600); err != nil {
		t.Fatal(err)
	}
}

func mustParse(t *testing.T, doc string) *topology.Topology {
	t.Helper()
	topo, err := topology.Parse([]byte(doc))
	if err != nil {
		t.Fatal(err)
	}
	return topo
}
