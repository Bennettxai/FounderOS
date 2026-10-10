package api

import (
	"compress/gzip"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/rhl/businessos-backend/internal/founderos/agents"
	"github.com/rhl/businessos-backend/internal/founderos/connectors"
)

// FounderOS v1 lib/agents/real.ts, in registry order: the brandDealAgent and
// newsletterAgent constants, then every `id:` inside the realAgents array.
// All 32 must have a bridge implementation, and nothing else may.
var founderosOSAgents = []string{
	"brand-deal-agent", "newsletter-agent",
	"conductor", "comms-digest", "comms-agent", "gmail-worker", "whatsapp-worker", "slack-worker",
	"social-agent", "postly-publisher", "adsmith-creative", "reelkit-editor", "renderly-creative",
	"dmflow-mcp", "sales-agent", "launchpad-cohort-sales", "vantage-sales", "paykit-sales",
	"vantage-paykit", "stripe-sales", "processor-confirmation", "flexpay-financing", "sales-calls-data",
	"data-agent", "markdown-auditor", "vector-auditor", "payments-pulse", "crm-pulse", "client-roster",
	"client-onboarding", "client-success", "stack-monitor",
}

func TestRosterMapsOneToOneOntoFounderosOSAgents(t *testing.T) {
	var got []string
	for _, a := range AllAgents(&Deps{Resolver: connectors.Resolver{}}) {
		got = append(got, a.Meta().ID)
	}
	want := append([]string(nil), founderosOSAgents...)
	sort.Strings(got)
	sort.Strings(want)
	if len(got) != len(want) {
		t.Fatalf("roster has %d agents %v, want %d", len(got), got, len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("roster = %v\nwant    %v", got, want)
		}
	}
}

// The ETL fixture is FounderOS v1's own seed (lib/seed.ts): its agents table is
// the founderos_agents rows /agents and /org render. Every roster agent must
// be exactly one seeded row, carrying that row's name and department (the
// pillar it shows under), and every row must have an agent. Descriptions are
// not pinned here: several ported agents deliberately carry their real.ts
// runtime description rather than the seed's.
func TestRosterMetasAreTheSeededAgentRows(t *testing.T) {
	f, err := os.Open("../etl/testdata/founderos-os.seed.sql.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := io.ReadAll(gz)
	if err != nil {
		t.Fatal(err)
	}
	lit := `'((?:[^']|'')*)'`
	// agents(id, department_id, name, ...)
	re := regexp.MustCompile(`INSERT INTO agents VALUES\(` + strings.Repeat(lit+`,`, 2) + lit)
	unq := func(s string) string { return strings.ReplaceAll(s, "''", "'") }
	seeded := map[string]agents.Meta{}
	for _, m := range re.FindAllStringSubmatch(string(raw), -1) {
		seeded[m[1]] = agents.Meta{ID: unq(m[1]), DepartmentID: unq(m[2]), Name: unq(m[3])}
	}
	if len(seeded) != len(founderosOSAgents) {
		t.Fatalf("seed has %d agent rows, real.ts registers %d", len(seeded), len(founderosOSAgents))
	}
	roster := AllAgents(&Deps{Resolver: connectors.Resolver{}})
	for _, a := range roster {
		m := a.Meta()
		m.Description = ""
		row, ok := seeded[m.ID]
		if !ok {
			t.Errorf("%s: no seeded row", m.ID)
			continue
		}
		if m != row {
			t.Errorf("%s:\n meta %+v\n row  %+v", m.ID, m, row)
		}
		delete(seeded, m.ID)
	}
	for id := range seeded {
		t.Errorf("seeded row %s has no bridge agent", id)
	}
}
