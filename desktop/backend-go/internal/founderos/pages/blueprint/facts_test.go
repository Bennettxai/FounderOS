package blueprint

import (
	"encoding/json"
	"strings"
	"testing"
)

// FounderOS v1 writes each node's facts in the order its compiler lists them
// (a JS object keeps insertion order), and the blueprint shows them in that
// order: the container chips and the inspector's Facts list. A Go map would
// sort them alphabetically, so Facts keeps its insertion order on the wire.
func TestFactsKeepInsertionOrderOnTheWire(t *testing.T) {
	f := facts("chip", "M4 Pro", "memory", "24 GB", "tailnet", "mini", "role", "production")
	f.Set("verified", "not from this host")
	f.Set("chip", "M5") // an overwrite keeps its slot
	b, err := json.Marshal(f)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"chip":"M5","memory":"24 GB","tailnet":"mini","role":"production","verified":"not from this host"}`
	if string(b) != want {
		t.Fatalf("facts json = %s, want %s", b, want)
	}
	var back Facts
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if again, _ := json.Marshal(back); string(again) != want {
		t.Fatalf("round trip = %s", again)
	}
	if back.Get("role") != "production" || back.Get("missing") != "" || back.Len() != 5 {
		t.Fatalf("get/len = %q %q %d", back.Get("role"), back.Get("missing"), back.Len())
	}
	if empty, _ := json.Marshal(Facts{}); string(empty) != "{}" {
		t.Fatalf("empty facts = %s, want {}", empty)
	}
}

// The workstation container's facts keep their written order and stay short:
// a long row pushed past the container's left edge on the map.
func TestTheWorkstationFactsKeepTheirOrder(t *testing.T) {
	g := compileFixture(t)
	b, err := json.Marshal(byID(g)[HostWorkstation].Facts)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"start":"scripts/dev-local.sh","role":"demo + development"}`
	if string(b) != want {
		t.Fatalf("workstation facts = %s\nwant %s", b, want)
	}
	agent, _ := json.Marshal(byID(g)["agent-comms-agent"].Facts)
	if !strings.HasPrefix(string(agent), `{"role":`) {
		t.Fatalf("agent facts open with role in prod, got %s", agent)
	}
}
