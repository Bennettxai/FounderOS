package blueprint

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
)

// Port of FounderOS v1 tests/blueprint.test.ts and tests/blueprint-infra.test.ts.

func fixtureRegistries() Registries {
	return Registries{
		Departments: []Department{
			{ID: "d-sales", Slug: "sales", Name: "Sales", Tagline: "pipeline"},
			{ID: "d-tech", Slug: "tech", Name: "TECH", Tagline: "the OS"},
		},
		Agents: []Agent{
			{ID: "conductor", DepartmentID: "d-tech", Name: "Conductor", Role: "Super agent", Status: "active", Tier: "lead", Description: "Routes everything.", Model: "claude"},
			{ID: "comms-agent", DepartmentID: "d-sales", Name: "Comms", Role: "Inbox", Status: "active", Tier: "worker", Tools: []string{"gmail", "slack", "fathom", "unknown-tool"}},
			{ID: "idle-agent", DepartmentID: "d-sales", Name: "Idle", Role: "Waits", Status: "idle", Tier: "worker", Tools: []string{"gbrain"}},
			{ID: "planned-agent", DepartmentID: "d-tech", Name: "Planned", Role: "Someday", Status: "planned", Tier: "worker"},
			{ID: "unwired-agent", DepartmentID: "d-tech", Name: "Unwired", Role: "No runtime", Status: "active", Tier: "worker"},
		},
		People: []Person{{ID: "koda", DepartmentID: "d-sales", Name: "Koda", Role: "Setter"}},
		Skills: []Skill{{ID: "voice", Name: "Voice", Description: "writing voice"}},
	}
}

var fixtureConnectors = []connectors.Status{
	{ID: "fathom", Name: "Fathom", Kind: "calls", State: connectors.StateConnected, Detail: "live"},
	{ID: "email", Name: "Email", Kind: "email", State: connectors.StateNotConfigured, Detail: "no inbox"},
	{ID: "slack", Name: "Slack", Kind: "slack", State: connectors.StateError, Detail: "HTTP 401"},
	{ID: "optimal-engine", Name: "Optimal Engine", Kind: "brain", State: connectors.StateConnected, Detail: "2/2 engines up · 5 workspaces"},
}

func fixtureOptions() Options {
	return Options{
		Operator:     "the operator",
		Runtime:      []string{"conductor", "comms-agent", "idle-agent"},
		RuntimeKnown: true,
		Connectors:   fixtureConnectors,
		Engines: []EngineInfo{
			{Name: "local", Tier: "device", URL: "http://127.0.0.1:4200", Workspaces: []string{"founderos", "vantage", "launchpad-cohort", "personal"}},
		},
		// this compile runs on the workstation where only the backend and Postgres answer
		HostID:    HostWorkstation,
		Probe:     func(port int) bool { return port == 8001 || port == 5432 },
		AppExists: func(string) bool { return false },
		Env:       func(string) string { return "" },
		Now:       time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC),
	}
}

func compileFixture(t *testing.T) Graph {
	t.Helper()
	g, err := Compile(fixtureRegistries(), fixtureOptions())
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func byID(g Graph) map[string]Node {
	m := map[string]Node{}
	for _, n := range g.Nodes {
		m[n.ID] = n
	}
	return m
}

func hasEdge(g Graph, from, to, kind string) bool {
	for _, e := range g.Edges {
		if e.From == from && e.To == to && (kind == "" || e.Kind == kind) {
			return true
		}
	}
	return false
}

func runsOn(g Graph, id string) string {
	for _, e := range g.Edges {
		if e.From == id && e.Kind == "runs-on" && strings.HasPrefix(e.To, "host-") {
			return e.To
		}
	}
	return ""
}

func TestEveryEdgeLandsOnARealNodeAndIDsAreUnique(t *testing.T) {
	g := compileFixture(t)
	if p := Validate(g); len(p) > 0 {
		t.Fatalf("invalid graph: %v", p)
	}
	if g.CompiledAt != "2026-09-30T12:00:00Z" {
		t.Fatalf("compiledAt = %q", g.CompiledAt)
	}
}

func TestValidateCatchesDanglingEdgesAndDuplicates(t *testing.T) {
	g := Graph{Nodes: []Node{{ID: "a"}, {ID: "a"}}, Edges: []Edge{{From: "a", To: "ghost", Kind: "uses"}}}
	p := Validate(g)
	if len(p) != 2 {
		t.Fatalf("problems = %v, want a dangling edge and a duplicate", p)
	}
}

func TestEveryAgentAppearsOnceWithTheConductorAsTheSpine(t *testing.T) {
	g := compileFixture(t)
	n := byID(g)
	c := n["agent-conductor"]
	if c.Kind != "wizard" || c.Name != "Conductor" || c.Layer != 1 {
		t.Fatalf("conductor = %+v", c)
	}
	if !hasEdge(g, "operator", "agent-conductor", "commands") {
		t.Fatal("operator does not command the Conductor")
	}
	commanded := 0
	for _, e := range g.Edges {
		if e.From == "agent-conductor" && e.Kind == "commands" && strings.HasPrefix(e.To, "agent-") {
			commanded++
		}
	}
	if commanded != len(fixtureRegistries().Agents)-1 {
		t.Fatalf("conductor commands %d agents", commanded)
	}
}

func TestLiveMeansWiredAndActivePlannedIsDesigned(t *testing.T) {
	n := byID(compileFixture(t))
	want := map[string]string{
		"agent-comms-agent":   "live",
		"agent-idle-agent":    "configured",
		"agent-planned-agent": "designed",
		"agent-unwired-agent": "designed",
	}
	for id, s := range want {
		if n[id].Status != s {
			t.Errorf("%s status = %q, want %q", id, n[id].Status, s)
		}
	}
	if n["agent-comms-agent"].Facts.Get("roster") != "active" || n["agent-comms-agent"].Facts.Get("tools") == "" {
		t.Errorf("agent facts = %v", n["agent-comms-agent"].Facts)
	}
}

func TestUnknownRuntimeNeverClaimsLive(t *testing.T) {
	o := fixtureOptions()
	o.Runtime, o.RuntimeKnown = nil, false
	g, err := Compile(fixtureRegistries(), o)
	if err != nil {
		t.Fatal(err)
	}
	for _, nd := range g.Nodes {
		if nd.Kind == "agent" && nd.Status == "live" {
			t.Fatalf("%s is live with no runtime registry", nd.ID)
		}
	}
	if byID(g)["agent-comms-agent"].Facts.Get("runtime") == "" {
		t.Fatal("an unknown runtime must say so on the node")
	}
}

func TestConnectorsAppearWithHonestStatus(t *testing.T) {
	g := compileFixture(t)
	n := byID(g)
	for id, s := range map[string]string{"connector-fathom": "live", "connector-email": "not-configured", "connector-slack": "configured"} {
		if n[id].Status != s || n[id].Layer != 3 {
			t.Errorf("%s = %+v, want status %s", id, n[id], s)
		}
	}
	count := 0
	for _, nd := range g.Nodes {
		if nd.Kind == "connector" {
			count++
		}
	}
	if count != len(fixtureConnectors) {
		t.Fatalf("%d connector nodes, want %d", count, len(fixtureConnectors))
	}
}

func TestAgentConnectorEdgesOnlyForRealConnectors(t *testing.T) {
	g := compileFixture(t)
	if !hasEdge(g, "agent-comms-agent", "connector-fathom", "uses") || !hasEdge(g, "agent-comms-agent", "connector-email", "uses") {
		t.Fatal("tool->connector edges missing")
	}
	// gbrain is retired on the bridge: the tool maps to the Optimal Engine row
	if !hasEdge(g, "agent-idle-agent", "connector-optimal-engine", "uses") {
		t.Fatal("gbrain tool should map to the optimal-engine connector")
	}
	for _, e := range g.Edges {
		if strings.Contains(e.To, "unknown-tool") || e.To == "connector-gbrain" {
			t.Fatalf("fabricated edge %+v", e)
		}
	}
}

func TestLayersZeroThroughFour(t *testing.T) {
	g := compileFixture(t)
	for layer := 0; layer <= 4; layer++ {
		found := false
		for _, n := range g.Nodes {
			if n.Layer == layer {
				found = true
			}
		}
		if !found {
			t.Errorf("layer %d empty", layer)
		}
	}
}

func TestEveryNavSurfaceIsANode(t *testing.T) {
	g := compileFixture(t)
	for _, s := range Surfaces {
		found := false
		for _, n := range g.Nodes {
			if n.Kind == "surface" && n.Facts.Get("route") == s.Href && n.Layer == 3 {
				found = true
			}
		}
		if !found {
			t.Errorf("surface %s missing", s.Href)
		}
	}
	for from, to := range map[string]string{"surface-brain": "store-brain", "surface-finances": "store-ledger", "surface-trading": "store-candles", "surface-brand-deals": "store-notion-deals", "surface-adpilot": "store-ad-intel"} {
		if !hasEdge(g, from, to, "uses") {
			t.Errorf("missing %s -> %s", from, to)
		}
	}
	if !hasEdge(g, "surface-finances", "store-bank", "uses") {
		t.Error("finances reads the bank store")
	}
}

// The Go surface list must match the bridge's sidebar (frontend nav.ts), so a
// new page shows up on the map the moment it lands in the nav.
func TestSurfacesMatchTheBridgeNav(t *testing.T) {
	dir, _ := os.Getwd()
	var raw []byte
	for {
		b, err := os.ReadFile(filepath.Join(dir, "frontend", "src", "lib", "founderos", "nav.ts"))
		if err == nil {
			raw = b
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Skip("frontend nav.ts not found above the working directory")
		}
		dir = parent
	}
	re := regexp.MustCompile(`href: '([^']+)', label: '([^']+)'`)
	var nav, ours []string
	for _, m := range re.FindAllStringSubmatch(string(raw), -1) {
		// Only the operator's own /os pages are surfaces; the sidebar's BusinessOS
		// group (/dashboard, /knowledge, ...) is not part of the blueprint.
		if m[1] != "/os" && !strings.HasPrefix(m[1], "/os/") {
			continue
		}
		nav = append(nav, m[1]+"|"+m[2])
	}
	for _, s := range Surfaces {
		ours = append(ours, s.Href+"|"+s.Label)
	}
	sort.Strings(nav)
	sort.Strings(ours)
	if strings.Join(nav, ",") != strings.Join(ours, ",") {
		t.Fatalf("Surfaces drifted from nav.ts:\n nav  %v\n ours %v", nav, ours)
	}
}

func TestTheDealRouterResolves(t *testing.T) {
	g := compileFixture(t)
	if byID(g)["router-deals"].Kind != "router" {
		t.Fatal("no deal router")
	}
	n := 0
	for _, e := range g.Edges {
		if e.From == "router-deals" || e.To == "router-deals" {
			n++
		}
	}
	if n < 3 || !hasEdge(g, "router-deals", "store-notion-deals", "uses") {
		t.Fatalf("deal router edges = %d", n)
	}
}

func TestDaemonsRunOnAHostAndNeverClaimLive(t *testing.T) {
	g := compileFixture(t)
	var ids []string
	for _, n := range g.Nodes {
		if n.Kind != "daemon" {
			continue
		}
		ids = append(ids, n.ID)
		if n.Status == "live" {
			t.Errorf("daemon %s claims live", n.ID)
		}
		if runsOn(g, n.ID) != HostWorkstation {
			t.Errorf("daemon %s runs on %q", n.ID, runsOn(g, n.ID))
		}
	}
	// v1's four daemons, no more
	sort.Strings(ids)
	if strings.Join(ids, ",") != "daemon-analytics-refresh,daemon-cron,daemon-hermes,daemon-telegram" {
		t.Fatalf("daemons = %v", ids)
	}
}

// The cron scheduler is the Go backend's own; it fires only with FOUNDEROS_CRONS=1.
func TestCronDaemonReflectsTheGuard(t *testing.T) {
	g := compileFixture(t)
	if s := byID(g)["daemon-cron"].Status; s != "not-configured" {
		t.Fatalf("with FOUNDEROS_CRONS unset the scheduler is %q", s)
	}
	if !hasEdge(g, "daemon-cron", "service-backend", "uses") {
		t.Fatal("the scheduler runs inside the backend")
	}
	o := fixtureOptions()
	o.Env = func(k string) string {
		if k == "FOUNDEROS_CRONS" {
			return "1"
		}
		return ""
	}
	g2, _ := Compile(fixtureRegistries(), o)
	if s := byID(g2)["daemon-cron"].Status; s != "configured" {
		t.Fatalf("with FOUNDEROS_CRONS=1 the scheduler is %q", s)
	}
}

// The machines are the demo stack's, generic: a workstation where the demo
// runs and the production host docker-compose.production.yml targets.
func TestHostsAreTheDemoStacksMachines(t *testing.T) {
	g := compileFixture(t)
	var hosts []string
	for _, n := range g.Nodes {
		if n.Kind == "host" {
			hosts = append(hosts, n.ID)
		}
	}
	sort.Strings(hosts)
	if strings.Join(hosts, ",") != "host-server,host-workstation" {
		t.Fatalf("hosts = %v", hosts)
	}
	n := byID(g)
	if n[HostWorkstation].Name != "Workstation" || n[HostWorkstation].Status != "live" {
		t.Fatalf("workstation = %+v", n[HostWorkstation])
	}
	if n[HostServer].Status != "designed" {
		t.Fatalf("the production host is not deployed from the demo: %+v", n[HostServer])
	}
}

func TestTheV2StackRunsOnTheWorkstation(t *testing.T) {
	g := compileFixture(t)
	n := byID(g)
	want := map[string]string{
		"service-frontend":       "SvelteKit frontend",
		"service-backend":        "Go backend",
		"service-postgres":       "Postgres",
		"service-redis":          "Redis",
		"service-optimal-engine": "Optimal Engine",
	}
	for id, name := range want {
		if n[id].Kind != "service" || n[id].Name != name {
			t.Errorf("%s = %+v, want service %q", id, n[id], name)
		}
		if runsOn(g, id) != HostWorkstation {
			t.Errorf("%s runs on %q", id, runsOn(g, id))
		}
	}
	for _, id := range []string{"store-db", "store-ledger", "store-bank", "store-candles"} {
		if runsOn(g, id) != HostWorkstation {
			t.Errorf("%s runs on %q", id, runsOn(g, id))
		}
	}
	for _, e := range [][3]string{
		{"service-frontend", "service-backend", "uses"},
		{"service-backend", "service-postgres", "reads"},
		{"service-backend", "service-redis", "uses"},
		{"service-backend", "store-brain", "reads"},
		{"store-brain", "service-optimal-engine", "reads"},
		{"store-db", "service-postgres", "runs-on"},
	} {
		if !hasEdge(g, e[0], e[1], e[2]) {
			t.Errorf("missing %s -%s-> %s", e[0], e[2], e[1])
		}
	}
	// no containers, dev tools or managed cloud: the demo stack has none
	for _, nd := range g.Nodes {
		if nd.Kind == "container" || nd.Kind == "tool" || (nd.Kind == "service" && runsOn(g, nd.ID) == "") {
			t.Errorf("unexpected %s node %s", nd.Kind, nd.ID)
		}
	}
}

func TestServicePortsComeFromTheBackendEnv(t *testing.T) {
	o := fixtureOptions()
	env := map[string]string{
		"FRONTEND_URL":       "http://localhost:5373",
		"SERVER_PORT":        "8901",
		"DATABASE_URL":       "postgres://u:pw@127.0.0.1:25532/db?sslmode=disable",
		"REDIS_URL":          "redis://127.0.0.1:26479/0",
		"OPTIMAL_ENGINE_URL": "http://127.0.0.1:4221",
	}
	o.Env = func(k string) string { return env[k] }
	o.Probe = func(port int) bool { return port == 8901 || port == 4221 }
	g, err := Compile(fixtureRegistries(), o)
	if err != nil {
		t.Fatal(err)
	}
	n := byID(g)
	ports := map[string]string{"service-frontend": "5373", "service-backend": "8901", "service-postgres": "25532", "service-redis": "26479", "service-optimal-engine": "4221"}
	for id, port := range ports {
		if n[id].Facts.Get("port") != port {
			t.Errorf("%s port = %q, want %s", id, n[id].Facts.Get("port"), port)
		}
	}
	if n["service-backend"].Status != "live" || n["service-optimal-engine"].Status != "live" || n["service-postgres"].Status != "configured" {
		t.Fatalf("probe statuses: backend %s, engine %s, postgres %s", n["service-backend"].Status, n["service-optimal-engine"].Status, n["service-postgres"].Status)
	}
	if n["service-backend"].Facts.Get("probe") != ":8901 answered on this host" {
		t.Fatalf("probe fact = %q", n["service-backend"].Facts.Get("probe"))
	}
	raw, _ := json.Marshal(g)
	if strings.Contains(string(raw), "pw@") {
		t.Fatal("a database credential leaked into the graph")
	}
}

func TestOnlyTheCompilingHostIsProbed(t *testing.T) {
	n := byID(compileFixture(t))
	if n["service-backend"].Status != "live" || n["service-postgres"].Status != "live" {
		t.Fatal("services that answered on this host are live")
	}
	if n["service-redis"].Status != "configured" || n["service-redis"].Facts.Get("probe") != ":6379 did not answer on this host" {
		t.Fatalf("a probed service that did not answer is configured, not live: %+v", n["service-redis"])
	}
	o := fixtureOptions()
	o.HostID = HostServer
	g, _ := Compile(fixtureRegistries(), o)
	if r := byID(g)["service-backend"]; r.Status != "configured" || r.Facts.Get("verified") != "not from this host" {
		t.Fatalf("another host's services are never claimed live: %+v", r)
	}
}

func TestNoHostKnownProbesNothing(t *testing.T) {
	o := fixtureOptions()
	o.HostID = ""
	probed := false
	o.Probe = func(int) bool { probed = true; return true }
	g, err := Compile(fixtureRegistries(), o)
	if err != nil {
		t.Fatal(err)
	}
	if probed {
		t.Fatal("probed with no host known")
	}
	for _, nd := range g.Nodes {
		if nd.Kind == "service" && nd.Status == "live" {
			t.Fatalf("%s live with no host known", nd.ID)
		}
	}
}

func TestTheOptimalEngineIsTheBrain(t *testing.T) {
	g := compileFixture(t)
	n := byID(g)
	brain := n["store-brain"]
	if brain.Name != "Optimal Engine" || brain.Status != "live" {
		t.Fatalf("store-brain = %+v", brain)
	}
	if brain.Facts.Get("engines") != "local" || brain.Facts.Get("workspaces") != "4" {
		t.Fatalf("brain facts = %v", brain.Facts)
	}
	// the brain without an OE row on the board is configured, not live
	o := fixtureOptions()
	o.Connectors = fixtureConnectors[:3]
	g2, _ := Compile(fixtureRegistries(), o)
	if s := byID(g2)["store-brain"].Status; s != "configured" {
		t.Fatalf("brain with no OE status = %q", s)
	}
}

// v1 draws the engine the Conductor runs on beside it: the AI Gateway, honest
// about a missing key, plus the model chain it walks.
func TestTheAIGatewaySitsBesideTheConductor(t *testing.T) {
	g := compileFixture(t)
	gw := byID(g)["model-claude-engine"]
	if gw.Kind != "model" || gw.Name != "AI Gateway" || gw.Status != "not-configured" || !strings.Contains(gw.Blurb, "AI_GATEWAY_API_KEY") {
		t.Fatalf("gateway = %+v", gw)
	}
	if !hasEdge(g, "agent-conductor", "model-claude-engine", "runs-on") {
		t.Fatal("the Conductor runs on the gateway")
	}
	if gw.Facts.Get("chain") != "anthropic/claude-sonnet-5 > openai/gpt-oss-20b > alibaba/qwen-3-14b" {
		t.Fatalf("chain = %q", gw.Facts.Get("chain"))
	}
	models := 0
	for _, nd := range g.Nodes {
		if nd.Kind == "model" && nd.ID != "model-claude-engine" {
			models++
			if nd.Status != "not-configured" {
				t.Errorf("%s claims %s with no key", nd.ID, nd.Status)
			}
		}
	}
	if models != 3 {
		t.Fatalf("%d chain models", models)
	}
	o := fixtureOptions()
	o.Env = func(k string) string {
		switch k {
		case "AI_GATEWAY_API_KEY":
			return "set"
		case "LLM_MODEL":
			return "openai/gpt-oss-20b"
		}
		return ""
	}
	g2, _ := Compile(fixtureRegistries(), o)
	gw2 := byID(g2)["model-claude-engine"]
	if gw2.Status != "live" || gw2.Blurb != "Vercel AI Gateway · default model openai/gpt-oss-20b" || gw2.Facts.Get("chain") != "openai/gpt-oss-20b > alibaba/qwen-3-14b" {
		t.Fatalf("configured gateway = %+v", gw2)
	}
}

func TestHostDetection(t *testing.T) {
	if got := CurrentHostID(func(string) string { return "" }); got != HostWorkstation {
		t.Errorf("the compiling backend is on the workstation by default, got %q", got)
	}
	if got := CurrentHostID(func(string) string { return "host-server" }); got != HostServer {
		t.Errorf("override ignored: %q", got)
	}
}

// No private machines, hosts or addresses in the map, and G-Brain is gone:
// the seeded roster's v1 wording reads as the Brain (the Optimal Engine).
func TestNoPrivateInfrastructure(t *testing.T) {
	reg := fixtureRegistries()
	reg.Departments[1].Tagline = "AI & automations · G-Brain."
	reg.Agents[2].Role = "G-Brain Analyst"
	reg.Agents[2].Description = "Runs gbrain doctor."
	reg.Agents[2].Model = "gbrain CLI"
	reg.Skills[0].Description = "Hybrid search over G-Brain."
	g, err := Compile(reg, fixtureOptions())
	if err != nil {
		t.Fatal(err)
	}
	n := byID(g)
	if n["agent-idle-agent"].Facts.Get("role") != "Brain Analyst" || n["agent-idle-agent"].Facts.Get("tools") != "optimal-engine" || n["dept-tech"].Blurb != "AI & automations · Brain." || n["skill-voice"].Blurb != "Hybrid search over Brain." {
		t.Fatalf("G-Brain wording not retired: %+v %+v %+v", n["agent-idle-agent"], n["dept-tech"], n["skill-voice"])
	}
	raw, _ := json.Marshal(g)
	raw = []byte(strings.ReplaceAll(string(raw), "Telegram bridge", "Telegram"))
	if m := regexp.MustCompile(`(?i)mac ?mini|macbook|tailnet|tailscale|100\.64\.|paperclip board|bridge|4210|4211|superset|iphone|the mini|G-Brain|gbrain`).Find(raw); m != nil {
		t.Fatalf("private build leftover in the graph: %q", m)
	}
}

func TestNoSecretsAndNoDonorNames(t *testing.T) {
	raw, _ := json.Marshal(compileFixture(t))
	if regexp.MustCompile(`(?i)glados`).Match(raw) {
		t.Fatal("graph says GladOS")
	}
	inv, _ := json.Marshal(infraNodes(func(string) string { return "" }))
	if regexp.MustCompile(`(?i)sk-|ghp_|xox[bp]-|secret`).Match(inv) {
		t.Fatal("secret-looking value in the static inventory")
	}
}

// Every icon the compiler names must exist in the frontend's icon map
// (src/lib/founderos/pages/blueprint/icons.ts), or the card falls back to Box.
func TestEveryIconHasAFrontendGlyph(t *testing.T) {
	dir, _ := os.Getwd()
	var raw []byte
	for {
		b, err := os.ReadFile(filepath.Join(dir, "frontend", "src", "lib", "founderos", "pages", "blueprint", "icons.ts"))
		if err == nil {
			raw = b
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Skip("frontend icons.ts not found above the working directory")
		}
		dir = parent
	}
	known := map[string]bool{}
	for _, m := range regexp.MustCompile(`(?m)^\s*'?([a-z0-9-]+)'?: c\(`).FindAllStringSubmatch(string(raw), -1) {
		known[m[1]] = true
	}
	if len(known) < 40 {
		t.Fatalf("parsed only %d icons from icons.ts", len(known))
	}
	for _, n := range compileFixture(t).Nodes {
		if !known[n.Icon] {
			t.Errorf("%s uses icon %q, missing from icons.ts", n.ID, n.Icon)
		}
	}
}
