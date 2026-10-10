package blueprint

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
)

// Registries is the roster as the bridge stores it (founderos_* tables in the
// founderos workspace).
type Registries struct {
	Departments []Department
	Agents      []Agent
	People      []Person
	Skills      []Skill
}

type Department struct{ ID, Slug, Name, Tagline string }

type Agent struct {
	ID, DepartmentID, Name, Role, Status, Tier, Description, Model string
	Tools                                                          []string
}

type Person struct{ ID, DepartmentID, Name, Role string }

type Skill struct{ ID, Name, Description string }

// EngineInfo is one Optimal Engine the topology routes workspaces to.
type EngineInfo struct {
	Name, Tier, URL string
	Workspaces      []string
}

// Surface is one /os page in the sidebar.
type Surface struct{ Href, Label string }

// Surfaces mirrors frontend src/lib/founderos/nav.ts (pinned by
// TestSurfacesMatchTheBridgeNav).
var Surfaces = []Surface{
	{"/os", "Home"}, {"/os/comms", "Comms"}, {"/os/funnel", "Funnel"}, {"/os/workflows", "Workflows"},
	{"/os/social", "Social"}, {"/os/content", "Content"}, {"/os/brand-deals", "Brand Deals"},
	{"/os/finances", "Finances"}, {"/os/trading", "Trading"}, {"/os/adpilot", "AdPilot"},
	{"/os/agents", "Agents"}, {"/os/chats", "Chats"}, {"/os/tasks", "Tasks"}, {"/os/skills", "Skills"},
	{"/os/org", "Org Chart"}, {"/os/blueprint", "Blueprint"}, {"/os/brain", "Brain"}, {"/os/doctor", "Doctor"},
	{"/os/integrations", "Connections"}, {"/os/usage", "Usage"}, {"/os/roadmap", "Roadmap"},
	{"/os/analytics", "Analytics"}, {"/os/reference", "Reference Model"}, {"/os/personas", "Personas"},
}

// Options carry everything live or environmental, injectable for tests.
type Options struct {
	Operator string
	// Runtime lists the agent ids the runtime registry can run. RuntimeKnown
	// is false when the registry could not be read, and then no agent is live.
	Runtime      []string
	RuntimeKnown bool
	Connectors   []connectors.Status
	Engines      []EngineInfo
	HostID       string // the compiling host; "" probes nothing
	Probe        func(port int) bool
	AppExists    func(bundle string) bool
	Env          func(string) string
	Now          time.Time
}

// Honest tool->connector matches only (FounderOS v1 TOOL_TO_CONNECTOR). GBrain
// is retired in v2, so the brain tools point at the Optimal Engine row.
var toolToConnector = map[string]string{
	"gmail": "email", "imap": "email", "calendar": "calendar", "zernio": "zernio", "beehiiv": "beehiiv",
	"stripe": "payments", "paypal": "payments", "square": "payments",
	"gbrain": "optimal-engine", "brain-store": "optimal-engine", "optimal-engine": "optimal-engine",
	"tmux": "local-stack", "slack": "slack", "whatsapp": "whatsapp", "trakyo": "trakyo", "fathom": "fathom",
	"typeform": "typeform", "vidalytics": "vidalytics", "loom": "loom", "plaud": "plaud", "arcads": "arcads", "wispr": "wispr",
}

var surfaceIcon = map[string]string{
	"/": "home", "/comms": "message-square", "/funnel": "filter", "/workflows": "workflow", "/social": "share-2",
	"/content": "clapperboard", "/brand-deals": "handshake", "/finances": "wallet", "/trading": "bar-chart-3",
	"/clients": "users", "/adpilot": "crosshair", "/agents": "users", "/tasks": "list-checks", "/skills": "sparkles",
	"/org": "network", "/blueprint": "waypoints", "/brain": "brain", "/doctor": "wrench", "/integrations": "plug",
	"/usage": "radar", "/analytics": "bar-chart-3", "/personas": "layers",
	"/chats": "messages-square", "/roadmap": "map", "/reference": "book-open",
}

// Obvious surface->store reads; every other surface is honestly bare.
var surfaceStoreEdges = map[string][]string{
	"/brain":       {"store-brain"},
	"/doctor":      {"store-brain"},
	"/finances":    {"store-ledger", "store-bank"},
	"/trading":     {"store-candles"},
	"/brand-deals": {"store-notion-deals"},
	"/adpilot":     {"store-ad-intel"},
}

// founderosPath maps a bridge href to its FounderOS v1 route ("/os/comms" → "/comms").
func founderosPath(href string) string {
	p := strings.TrimPrefix(href, "/os")
	if p == "" {
		return "/"
	}
	return p
}

func surfaceSlug(path string) string {
	if path == "/" {
		return "home"
	}
	return strings.ReplaceAll(strings.TrimPrefix(path, "/"), "/", "-")
}

func clip(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

func connectorNode(s connectors.Status) Node {
	status := map[connectors.State]string{
		connectors.StateConnected:     StatusLive,
		connectors.StateNotConfigured: StatusNotConfigured,
		connectors.StateError:         StatusConfigured,
	}[s.State]
	if status == "" {
		status = StatusConfigured
	}
	return Node{ID: "connector-" + s.ID, Kind: KindConnector, Name: s.Name, Layer: 3, Status: status, Blurb: clip(s.Detail, 140), Facts: facts("kind", string(s.Kind)), Icon: "plug"}
}

func stores(o Options) []Node {
	brainStatus, brainDetail := StatusConfigured, "no Optimal Engine row on the Connections board"
	for _, c := range o.Connectors {
		if c.ID == "optimal-engine" {
			brainStatus, brainDetail = connectorNode(c).Status, c.Detail
		}
	}
	var names []string
	ws := map[string]bool{}
	for _, e := range o.Engines {
		names = append(names, e.Name)
		for _, w := range e.Workspaces {
			ws[w] = true
		}
	}
	sort.Strings(names)
	engines := "no engine configured"
	if len(names) > 0 {
		engines = strings.Join(names, ", ")
	}
	return []Node{
		{ID: "store-db", Kind: KindStore, Name: "OS database", Layer: 4, Status: StatusLive, Blurb: "Agents, runs, departments, workflows, crons, metrics: the founderos_* tables in Postgres.", Facts: facts("path", "founderos_*"), Icon: "database"},
		{ID: "store-ledger", Kind: KindStore, Name: "Ledger", Layer: 4, Status: StatusLive, Blurb: "Statement events and income by business (Postgres).", Facts: facts("path", "founderos_ledger_rows"), Icon: "database"},
		{ID: "store-bank", Kind: KindStore, Name: "Bank", Layer: 4, Status: StatusLive, Blurb: "Uploaded statements and the bank line items (Postgres).", Facts: facts("path", "founderos_bank_summaries"), Icon: "database"},
		{ID: "store-candles", Kind: KindStore, Name: "Candles", Layer: 4, Status: StatusLive, Blurb: "Crypto and ETF candles the trading backtests read.", Facts: facts("path", "data/candles/"), Icon: "list"},
		{ID: "store-brain", Kind: KindStore, Name: "Optimal Engine", Layer: 4, Status: brainStatus,
			Blurb: "The OS memory layer: each workspace lives in its home engine per the engine topology.",
			Facts: facts("engines", engines, "workspaces", fmt.Sprint(len(ws)), "status", clip(brainDetail, 120), "topology", "config/founderos/engine-topology.yaml"), Icon: "brain"},
		{ID: "store-notion-deals", Kind: KindStore, Name: "Notion Brand Deals Hub", Layer: 4, Status: StatusLive, Blurb: "The sponsorship pipeline. Notion stays the source of truth; the OS reads it.", Facts: facts("source", "Notion data source"), Icon: "handshake"},
		{ID: "store-ad-intel", Kind: KindStore, Name: "Ad intel", Layer: 4, Status: StatusLive, Blurb: "Adscout's synced ads, signals and saves.", Facts: facts("path", "data/ad-intel/"), Icon: "radar"},
	}
}

// daemons are the background processes the OS names. None is verified from a
// compile, so none claims live.
func daemons(o Options) []Node {
	has := func(k string) bool { return strings.TrimSpace(o.Env(k)) != "" }
	pick := func(ok bool, yes, no string) string {
		if ok {
			return yes
		}
		return no
	}
	crons := o.Env("FOUNDEROS_CRONS") == "1"
	return []Node{
		{ID: "daemon-cron", Kind: KindDaemon, Name: "Cron scheduler", Layer: 4, Status: pick(crons, StatusConfigured, StatusNotConfigured),
			Blurb: pick(crons, "Runs the agent crons on schedule inside the backend and records every run in cron_runs. Whether it is firing is visible on /workflows, not here.", "FOUNDEROS_CRONS is off: the backend's scheduler is registered and fires nothing."),
			Facts: facts("guard", "FOUNDEROS_CRONS", "runs in", "the Go backend"), Icon: "sunrise"},
		{ID: "daemon-analytics-refresh", Kind: KindDaemon, Name: "Analytics refresh", Layer: 4, Status: StatusConfigured, Blurb: "The 15-minute sweep that snapshots metrics so /analytics never pays a live call per render.", Facts: facts("cadence", "every 15 min"), Icon: "radar"},
		{ID: "daemon-hermes", Kind: KindDaemon, Name: "Hermes gateway", Layer: 4, Status: pick(has("HERMES_GATEWAY_URL"), StatusConfigured, StatusNotConfigured),
			Blurb: pick(has("HERMES_GATEWAY_URL"), "Gateway URL set; the dashboard tab on /agents talks to it. Not probed at compile time.", "HERMES_GATEWAY_URL is unset on this host."), Facts: facts(), Icon: "send"},
		{ID: "daemon-telegram", Kind: KindDaemon, Name: "Telegram bridge", Layer: 4, Status: pick(has("TELEGRAM_BOT_TOKEN"), StatusConfigured, StatusNotConfigured),
			Blurb: pick(has("TELEGRAM_BOT_TOKEN"), "The front door from a phone: a message becomes a comment on the cockpit issue and wakes the Conductor.", "TELEGRAM_BOT_TOKEN is unset on this host."), Facts: facts(), Icon: "message-circle"},
	}
}

// The engine the Conductor and every chat run on (FounderOS v1
// lib/connectors/llm.ts): the Vercel AI Gateway, with the free-tier models it
// falls back to when the preferred one is refused.
const (
	gatewayKey    = "AI_GATEWAY_API_KEY"
	fallbackModel = "anthropic/claude-sonnet-5"
)

var freeTierModels = []string{"openai/gpt-oss-20b", "alibaba/qwen-3-14b"}

func modelChain(env func(string) string) []string {
	preferred := strings.TrimSpace(env("LLM_MODEL"))
	if preferred == "" {
		preferred = fallbackModel
	}
	seen := map[string]bool{}
	var chain []string
	for _, m := range append([]string{preferred}, freeTierModels...) {
		if !seen[m] {
			seen[m] = true
			chain = append(chain, m)
		}
	}
	return chain
}

var modelIDUnsafe = regexp.MustCompile(`[^a-zA-Z0-9]+`)

func gatewayNodes(env func(string) string) []Node {
	chain := modelChain(env)
	up := strings.TrimSpace(env(gatewayKey)) != ""
	status, detail := StatusNotConfigured, "Set "+gatewayKey+" in the backend environment to enable agent chat via the Vercel AI Gateway."
	if up {
		status, detail = StatusLive, "Vercel AI Gateway · default model "+chain[0]
	}
	nodes := []Node{{ID: "model-claude-engine", Kind: KindModel, Name: "AI Gateway", Layer: 3, Status: status, Blurb: clip(detail, 140),
		Facts: facts("via", "Vercel AI Gateway", "chain", strings.Join(chain, " > ")), Icon: "cpu"}}
	for _, id := range chain {
		nodes = append(nodes, Node{ID: "model-" + modelIDUnsafe.ReplaceAllString(id, "-"), Kind: KindModel, Name: id, Layer: 3, Status: status,
			Blurb: "In the gateway fallback chain: tried in order when the preferred model is refused.", Facts: facts("id", id), Icon: "cpu"})
	}
	return nodes
}

// brainWording retires v1's G-Brain from the seeded roster's text: in v2 the
// brain is the bundled Optimal Engine. Longest phrases first.
var brainWording = strings.NewReplacer("gbrain CLI", "Optimal Engine", "G-Brain", "Brain", "gbrain", "optimal-engine")

func retireGBrain(reg Registries) Registries {
	out := Registries{People: reg.People}
	for _, d := range reg.Departments {
		d.Tagline = brainWording.Replace(d.Tagline)
		out.Departments = append(out.Departments, d)
	}
	for _, a := range reg.Agents {
		a.Role, a.Description, a.Model = brainWording.Replace(a.Role), brainWording.Replace(a.Description), brainWording.Replace(a.Model)
		var tools []string
		seen := map[string]bool{}
		for _, t := range a.Tools {
			if t = brainWording.Replace(t); !seen[t] {
				seen[t] = true
				tools = append(tools, t)
			}
		}
		a.Tools = tools
		out.Agents = append(out.Agents, a)
	}
	for _, sk := range reg.Skills {
		sk.Description = brainWording.Replace(sk.Description)
		out.Skills = append(out.Skills, sk)
	}
	return out
}

var deptIcon, agentIcon, personIcon, skillIcon = "folder", "bot", "user-round", "sparkles"

// Compile assembles the Blueprint graph (FounderOS v1 compileBlueprint).
func Compile(reg Registries, o Options) (Graph, error) {
	if o.Env == nil {
		o.Env = func(string) string { return "" }
	}
	if o.Now.IsZero() {
		o.Now = time.Now()
	}
	reg = retireGBrain(reg)
	operator := strings.TrimSpace(o.Operator)
	if operator == "" {
		operator = "the operator"
	}
	var nodes []Node
	var edges []Edge
	runtime := map[string]bool{}
	for _, id := range o.Runtime {
		runtime[id] = true
	}
	connectorIDs := map[string]bool{}
	for _, c := range o.Connectors {
		connectorIDs["connector-"+c.ID] = true
	}
	deptByID := map[string]Department{}
	for _, d := range reg.Departments {
		deptByID[d.ID] = d
	}

	// L0 + L1: the spine.
	nodes = append(nodes, Node{ID: "operator", Kind: KindOperator, Name: operator, Layer: 0, Status: StatusLive, Blurb: "The operator. Everything answers upward to here.", Facts: facts(), Icon: "user-round"})
	conductor := Node{ID: "agent-conductor", Kind: KindWizard, Name: "Conductor", Layer: 1, Status: StatusLive, Blurb: "The super agent: talks to everything, routes to everyone.", Facts: facts(), Icon: "wand-2"}
	for _, a := range reg.Agents {
		if a.ID == "conductor" {
			conductor.Blurb = clip(a.Description, 140)
			conductor.Facts = facts("role", a.Role, "model", a.Model)
		}
	}
	nodes = append(nodes, conductor)
	edges = append(edges, Edge{From: "operator", To: "agent-conductor", Kind: "commands"})

	// L2: departments + agents, bound against the runtime registry. A roster
	// agent with no runtime is honestly designed; a wired one not active on
	// the roster is configured, never live.
	for _, d := range reg.Departments {
		nodes = append(nodes, Node{ID: "dept-" + d.Slug, Kind: KindDepartment, Name: d.Name, Layer: 2, Status: StatusLive, Blurb: d.Tagline, Facts: facts(), Icon: deptIcon})
	}
	for _, a := range reg.Agents {
		if a.ID == "conductor" {
			continue
		}
		wired := runtime[a.ID]
		status := StatusConfigured
		switch {
		case a.Status == "planned" || !wired:
			status = StatusDesigned
		case a.Status == "active":
			status = StatusLive
		}
		f := facts("role", a.Role, "model", a.Model, "tools", strings.Join(a.Tools, ", "), "tier", a.Tier, "roster", a.Status)
		if !o.RuntimeKnown {
			f.Set("runtime", "registry unavailable on this load")
		}
		nodes = append(nodes, Node{ID: "agent-" + a.ID, Kind: KindAgent, Name: a.Name, Layer: 2, Status: status, Blurb: clip(a.Description, 140), Facts: f, Icon: agentIcon})
		edges = append(edges, Edge{From: "agent-conductor", To: "agent-" + a.ID, Kind: "commands"})
		if d, ok := deptByID[a.DepartmentID]; ok {
			edges = append(edges, Edge{From: "agent-" + a.ID, To: "dept-" + d.Slug, Kind: "member-of"})
		}
		seen := map[string]bool{}
		for _, t := range a.Tools {
			id := toolToConnector[t]
			nid := "connector-" + id
			if id != "" && connectorIDs[nid] && !seen[nid] {
				seen[nid] = true
				edges = append(edges, Edge{From: "agent-" + a.ID, To: nid, Kind: "uses"})
			}
		}
	}
	for _, p := range reg.People {
		nodes = append(nodes, Node{ID: "person-" + p.ID, Kind: KindPerson, Name: p.Name, Layer: 2, Status: StatusLive, Blurb: p.Role, Facts: facts(), Icon: personIcon})
		if d, ok := deptByID[p.DepartmentID]; ok {
			edges = append(edges, Edge{From: "person-" + p.ID, To: "dept-" + d.Slug, Kind: "member-of"})
		}
	}

	// The engine the Conductor runs on, and the model chain it walks.
	nodes = append(nodes, gatewayNodes(o.Env)...)
	edges = append(edges, Edge{From: "agent-conductor", To: "model-claude-engine", Kind: "runs-on"})

	// L3: skills, connectors (live statuses), surfaces.
	for _, s := range reg.Skills {
		nodes = append(nodes, Node{ID: "skill-" + s.ID, Kind: KindSkillpack, Name: s.Name, Layer: 3, Status: StatusLive, Blurb: clip(s.Description, 120), Facts: facts(), Icon: skillIcon})
	}
	for _, c := range o.Connectors {
		nodes = append(nodes, connectorNode(c))
	}
	for _, s := range Surfaces {
		path := founderosPath(s.Href)
		slug := surfaceSlug(path)
		icon := surfaceIcon[path]
		if icon == "" {
			icon = "square"
		}
		nodes = append(nodes, Node{ID: "surface-" + slug, Kind: KindSurface, Name: s.Label, Layer: 3, Status: StatusLive, Blurb: "OS page at " + s.Href + ".", Facts: facts("route", s.Href), Icon: icon})
		for _, st := range surfaceStoreEdges[path] {
			edges = append(edges, Edge{From: "surface-" + slug, To: st, Kind: "uses"})
		}
	}

	// L4: infrastructure, stores, daemons.
	ds := daemons(o)
	infraN, infraE := resolveInfrastructure(o)
	nodes = append(nodes, infraHosts()...)
	nodes = append(nodes, stores(o)...)
	nodes = append(nodes, ds...)
	nodes = append(nodes, infraN...)
	for _, s := range []string{"store-db", "store-ledger", "store-bank", "store-candles"} {
		edges = append(edges, Edge{From: s, To: HostWorkstation, Kind: "runs-on"})
	}
	for _, s := range []string{"store-db", "store-ledger", "store-bank"} {
		edges = append(edges, Edge{From: s, To: "service-postgres", Kind: "runs-on"})
	}
	for _, d := range ds {
		edges = append(edges, Edge{From: d.ID, To: HostWorkstation, Kind: "runs-on"})
	}
	edges = append(edges, infraE...)
	edges = append(edges, Edge{From: "daemon-cron", To: "service-backend", Kind: "uses", Via: "in-process"})

	// The one routing diamond with a real rule behind it: sponsorship deals.
	nodes = append(nodes, Node{ID: "router-deals", Kind: KindRouter, Name: "Deal routing", Layer: 3, Status: StatusLive,
		Blurb: "Deal routing: inbound sponsor messages land in the Notion hub, the brand-deal agents draft there, the board reads it back.",
		Facts: facts("rule", "Notion is the source of truth; the OS never writes a deal, only reads the hub"), Icon: "route"})
	edges = append(edges,
		Edge{From: "surface-comms", To: "router-deals", Kind: "uses", Via: "inbound"},
		Edge{From: "surface-brand-deals", To: "router-deals", Kind: "uses", Via: "board"},
		Edge{From: "router-deals", To: "store-notion-deals", Kind: "uses", Via: "Notion hub"},
	)

	g := Graph{CompiledAt: o.Now.UTC().Format(time.RFC3339), Nodes: nodes, Edges: edges}
	if p := Validate(g); len(p) > 0 {
		return Graph{}, fmt.Errorf("blueprint graph invalid: %s", strings.Join(p, "; "))
	}
	return g, nil
}
