// Package blueprint is the bridge port of FounderOS v1 lib/blueprint: the
// system, drawn from itself. Compile assembles the graph from the bridge's
// live registries (the Postgres roster, the agent runtime, the Connections
// board, the Optimal Engine topology) plus the hand-listed infrastructure, so
// the map changes when the system does. Layout stays client-side, like
// FounderOS v1 (frontend src/lib/founderos/pages/blueprint).
package blueprint

import "fmt"

// Node kinds (lib/blueprint/graph.ts NODE_KINDS).
const (
	KindOperator   = "operator"
	KindWizard     = "wizard"
	KindDepartment = "department"
	KindAgent      = "agent"
	KindPerson     = "person"
	KindConnector  = "connector"
	KindModel      = "model"
	KindSkillpack  = "skillpack"
	KindStore      = "store"
	KindDaemon     = "daemon"
	KindHost       = "host"
	KindService    = "service"
	KindContainer  = "container"
	KindTool       = "tool"
	KindSurface    = "surface"
	KindRouter     = "router"
)

// Honest lifecycle: live (wired + working), configured (wired, unverified),
// not-configured (wired in code, missing its credential/host), designed
// (specified but not wired).
const (
	StatusLive          = "live"
	StatusConfigured    = "configured"
	StatusNotConfigured = "not-configured"
	StatusDesigned      = "designed"
)

// Node mirrors BlueprintNodeSchema. Layers, top-down: 0 operator, 1 wizard,
// 2 departments + agents, 3 capabilities, 4 infrastructure.
type Node struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	Name   string `json:"name"`
	Layer  int    `json:"layer"`
	Status string `json:"status"`
	Blurb  string `json:"blurb"`
	Facts  Facts  `json:"facts"`
	Icon   string `json:"icon"`
}

// Edge mirrors BlueprintEdgeSchema. Kinds: commands, member-of, uses,
// runs-on, reads, writes, delivers-to, deploys-to.
type Edge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Kind string `json:"kind"`
	Via  string `json:"via,omitempty"`
}

type Graph struct {
	CompiledAt string `json:"compiledAt"`
	Nodes      []Node `json:"nodes"`
	Edges      []Edge `json:"edges"`
}

// Validate is the compiler's own honesty check: every edge endpoint exists
// and node ids are unique.
func Validate(g Graph) []string {
	ids := map[string]bool{}
	for _, n := range g.Nodes {
		ids[n.ID] = true
	}
	var problems []string
	for _, e := range g.Edges {
		if !ids[e.From] {
			problems = append(problems, "edge from missing node: "+e.From)
		}
		if !ids[e.To] {
			problems = append(problems, "edge to missing node: "+e.To)
		}
	}
	seen := map[string]bool{}
	for _, n := range g.Nodes {
		if seen[n.ID] {
			problems = append(problems, fmt.Sprintf("duplicate node id: %s", n.ID))
		}
		seen[n.ID] = true
	}
	return problems
}
