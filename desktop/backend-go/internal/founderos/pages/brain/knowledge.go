// Package brain is the pure logic behind the /os/brain page, ported from
// FounderOS v1 lib/knowledge-graph.ts, lib/brain-viz.ts, lib/brain-satellites.ts,
// lib/brain-dump.ts and lib/memory-core.ts. The knowledge core now reads the
// Optimal Engine instead of the GBrain brain-store.
package brain

import (
	"sort"
	"strconv"
	"strings"
)

type Department struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Slug    string `json:"slug"`
	Tagline string `json:"tagline"`
	Color   string `json:"color"`
	Order   int    `json:"order"`
}

type Agent struct {
	ID           string   `json:"id"`
	DepartmentID string   `json:"departmentId"`
	Name         string   `json:"name"`
	Role         string   `json:"role"`
	Status       string   `json:"status"`
	Tier         string   `json:"tier"`
	Description  string   `json:"description"`
	Model        string   `json:"model"`
	Tools        []string `json:"tools"`
	ParentID     *string  `json:"parentId"`
	Instance     string   `json:"instance"`
}

type Person struct {
	ID           string   `json:"id"`
	DepartmentID string   `json:"departmentId"`
	Name         string   `json:"name"`
	Role         string   `json:"role"`
	Tools        []string `json:"tools"`
}

type SopTask struct {
	ID           string   `json:"id"`
	DepartmentID string   `json:"departmentId"`
	Title        string   `json:"title"`
	Summary      string   `json:"summary"`
	Steps        []string `json:"steps"`
	AssigneeKind string   `json:"assigneeKind"` // agent | person
	AssigneeID   string   `json:"assigneeId"`
}

// BoardAgent is a live Paperclip seat that is not a department lead.
type BoardAgent struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type KGNode struct {
	ID    string `json:"id"`
	Kind  string `json:"kind"` // self | board | team | task | employee | person | tool
	Label string `json:"label"`
	Ring  int    `json:"ring"` // 0 = the operator core → 4 = tools
	Color string `json:"color,omitempty"`
}

type KGEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Kind   string `json:"kind"` // pillar | sop | does | member | uses | reports | board
}

type KnowledgeGraph struct {
	Nodes []KGNode `json:"nodes"`
	Edges []KGEdge `json:"edges"`
}

const SelfID = "self"

var ring = map[string]int{"self": 0, "board": 1, "team": 1, "task": 2, "employee": 3, "person": 3, "tool": 4}

// DeptExecTitles: the pillar node IS the department-head agent.
var DeptExecTitles = map[string]string{
	"dept-sales":            "CRO",
	"dept-marketing-growth": "CMO",
	"dept-tech":             "CTO",
	"dept-finance":          "CFO",
	"dept-comms":            "CCO",
	"dept-clients":          "COO",
}

// GraphDeptOrder is the graph-only display order: Finances rides next to Sales.
var GraphDeptOrder = []string{"dept-sales", "dept-finance", "dept-clients", "dept-marketing-growth", "dept-tech", "dept-comms"}

// lifeAreaColors: lib/life-map.ts LIFE_AREAS, first area per department.
var lifeAreaColors = map[string]string{
	"dept-marketing-growth": "#f59e0b",
	"dept-sales":            "#ef4444",
	"dept-finance":          "#22c55e",
	"dept-comms":            "#3b82f6",
	"dept-clients":          "#14b8a6",
	"dept-tech":             "#a855f7",
}

// LifeAreaColor is a department's life-area tint, "" when it has none.
func LifeAreaColor(deptID string) string { return lifeAreaColors[deptID] }

// GraphDeptRank ranks a department for layout; unknown ids sort last.
func GraphDeptRank(deptID string) int {
	for i, id := range GraphDeptOrder {
		if id == deptID {
			return i
		}
	}
	return len(GraphDeptOrder) + 1
}

func OrderGraphDepartments(ds []Department) []Department {
	out := append([]Department(nil), ds...)
	sort.SliceStable(out, func(i, j int) bool { return GraphDeptRank(out[i].ID) < GraphDeptRank(out[j].ID) })
	return out
}

// prettify: 'comms-feed' → 'Comms Feed'.
func prettify(slug string) string {
	parts := strings.FieldsFunc(slug, func(r rune) bool { return r == '-' || r == '_' })
	for i, p := range parts {
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, " ")
}

// WorkerNodeID: agents are emp:, humans are person:.
func WorkerNodeID(kind, assigneeID string) string {
	if kind == "agent" {
		return "emp:" + assigneeID
	}
	return "person:" + assigneeID
}

// ToolSlugOf maps a tool node id (tool:slug or tool:slug@dept) to its slug.
func ToolSlugOf(nodeID string) string {
	s := strings.TrimPrefix(nodeID, "tool:")
	if i := strings.Index(s, "@"); i >= 0 {
		s = s[:i]
	}
	return s
}

type workerRow struct {
	nodeID, kind, label, deptID string
	tools                       []string
}

// BuildKnowledgeGraph: the operator at the core, board seats and pillars on ring 1,
// SOP tasks on ring 2, workers (agents and humans) on ring 3, tools on ring 4.
// A tool used by several departments is duplicated, one copy per department.
func BuildKnowledgeGraph(agents []Agent, departments []Department, people []Person, tasks []SopTask, boardAgents []BoardAgent) KnowledgeGraph {
	g := KnowledgeGraph{Nodes: []KGNode{{ID: SelfID, Kind: "self", Label: "the operator", Ring: 0}}, Edges: []KGEdge{}}

	board := append([]BoardAgent(nil), boardAgents...)
	sort.SliceStable(board, func(i, j int) bool { return board[i].Name < board[j].Name })
	for _, b := range board {
		g.Nodes = append(g.Nodes, KGNode{ID: "board:" + b.ID, Kind: "board", Label: b.Name, Ring: ring["board"]})
		g.Edges = append(g.Edges, KGEdge{Source: SelfID, Target: "board:" + b.ID, Kind: "board"})
	}

	used := map[string]bool{}
	for _, a := range agents {
		used[a.DepartmentID] = true
	}
	for _, p := range people {
		used[p.DepartmentID] = true
	}
	for _, d := range departments {
		if !used[d.ID] {
			continue
		}
		g.Nodes = append(g.Nodes, KGNode{ID: "team:" + d.ID, Kind: "team", Label: d.Name, Ring: ring["team"], Color: LifeAreaColor(d.ID)})
		g.Edges = append(g.Edges, KGEdge{Source: SelfID, Target: "team:" + d.ID, Kind: "pillar"})
	}

	assigned := map[string]bool{}
	for _, t := range tasks {
		if !used[t.DepartmentID] {
			continue
		}
		g.Nodes = append(g.Nodes, KGNode{ID: "task:" + t.ID, Kind: "task", Label: t.Title, Ring: ring["task"]})
		g.Edges = append(g.Edges, KGEdge{Source: "team:" + t.DepartmentID, Target: "task:" + t.ID, Kind: "sop"})
		w := WorkerNodeID(t.AssigneeKind, t.AssigneeID)
		g.Edges = append(g.Edges, KGEdge{Source: "task:" + t.ID, Target: w, Kind: "does"})
		assigned[w] = true
	}

	var workers []workerRow
	for _, a := range agents {
		workers = append(workers, workerRow{"emp:" + a.ID, "employee", a.Name, a.DepartmentID, a.Tools})
	}
	for _, p := range people {
		workers = append(workers, workerRow{"person:" + p.ID, "person", p.Name, p.DepartmentID, p.Tools})
	}
	deptsOfTool := map[string]map[string]bool{}
	for _, w := range workers {
		for _, slug := range w.tools {
			if deptsOfTool[slug] == nil {
				deptsOfTool[slug] = map[string]bool{}
			}
			deptsOfTool[slug][w.deptID] = true
		}
	}
	toolNodeID := func(slug, deptID string) string {
		if len(deptsOfTool[slug]) > 1 {
			return "tool:" + slug + "@" + deptID
		}
		return "tool:" + slug
	}

	for _, w := range workers {
		g.Nodes = append(g.Nodes, KGNode{ID: w.nodeID, Kind: w.kind, Label: w.label, Ring: ring[w.kind]})
		if !assigned[w.nodeID] && used[w.deptID] {
			g.Edges = append(g.Edges, KGEdge{Source: w.nodeID, Target: "team:" + w.deptID, Kind: "member"})
		}
		for _, slug := range w.tools {
			g.Edges = append(g.Edges, KGEdge{Source: w.nodeID, Target: toolNodeID(slug, w.deptID), Kind: "uses"})
		}
	}
	agentIDs := map[string]bool{}
	for _, a := range agents {
		agentIDs[a.ID] = true
	}
	for _, a := range agents {
		if a.ParentID != nil && agentIDs[*a.ParentID] {
			g.Edges = append(g.Edges, KGEdge{Source: "emp:" + a.ID, Target: "emp:" + *a.ParentID, Kind: "reports"})
		}
	}

	slugs := make([]string, 0, len(deptsOfTool))
	for s := range deptsOfTool {
		slugs = append(slugs, s)
	}
	sort.Strings(slugs)
	for _, slug := range slugs {
		depts := keys(deptsOfTool[slug])
		if len(depts) > 1 {
			for _, d := range depts {
				g.Nodes = append(g.Nodes, KGNode{ID: "tool:" + slug + "@" + d, Kind: "tool", Label: prettify(slug), Ring: ring["tool"]})
			}
		} else {
			g.Nodes = append(g.Nodes, KGNode{ID: "tool:" + slug, Kind: "tool", Label: prettify(slug), Ring: ring["tool"]})
		}
	}
	return g
}

func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

type DirectoryRow struct {
	ID      string   `json:"id"`
	Label   string   `json:"label"`
	Sub     string   `json:"sub"`
	DeptIDs []string `json:"deptIds"`
}

type DirectoryGroup struct {
	Kind  string         `json:"kind"` // employee | person | task | tool
	Title string         `json:"title"`
	Rows  []DirectoryRow `json:"rows"`
}

// GraphDirectory is the scrollable everything-index beside the graph: AI
// agents, humans, SOPs, tools, each alphabetized. Tool rows carry the slug.
func GraphDirectory(agents []Agent, departments []Department, people []Person, tasks []SopTask, g KnowledgeGraph) []DirectoryGroup {
	deptName := map[string]string{}
	for _, d := range departments {
		deptName[d.ID] = d.Name
	}
	nameOf := func(id string) string {
		if n, ok := deptName[id]; ok {
			return n
		}
		return id
	}
	deptOfNode := map[string]string{}
	for _, a := range agents {
		deptOfNode["emp:"+a.ID] = a.DepartmentID
	}
	for _, p := range people {
		deptOfNode["person:"+p.ID] = p.DepartmentID
	}

	toolRows := map[string]DirectoryRow{}
	for _, n := range g.Nodes {
		if n.Kind != "tool" {
			continue
		}
		slug := ToolSlugOf(n.ID)
		if _, seen := toolRows[slug]; seen {
			continue
		}
		users := map[string]bool{}
		for _, e := range g.Edges {
			if e.Kind == "uses" && ToolSlugOf(e.Target) == slug {
				users[e.Source] = true
			}
		}
		depts := map[string]bool{}
		for u := range users {
			if d, ok := deptOfNode[u]; ok {
				depts[d] = true
			}
		}
		sub := "1 user"
		if len(users) != 1 {
			sub = strconv.Itoa(len(users)) + " users"
		}
		toolRows[slug] = DirectoryRow{ID: slug, Label: n.Label, Sub: sub, DeptIDs: keys(depts)}
	}

	byLabel := func(rows []DirectoryRow) []DirectoryRow {
		sort.SliceStable(rows, func(i, j int) bool { return rows[i].Label < rows[j].Label })
		return rows
	}
	var agentRows, personRows, taskRows, tools []DirectoryRow
	for _, a := range agents {
		agentRows = append(agentRows, DirectoryRow{ID: "emp:" + a.ID, Label: a.Name, Sub: nameOf(a.DepartmentID), DeptIDs: []string{a.DepartmentID}})
	}
	for _, p := range people {
		personRows = append(personRows, DirectoryRow{ID: "person:" + p.ID, Label: p.Name, Sub: nameOf(p.DepartmentID), DeptIDs: []string{p.DepartmentID}})
	}
	for _, t := range tasks {
		taskRows = append(taskRows, DirectoryRow{ID: "task:" + t.ID, Label: t.Title, Sub: nameOf(t.DepartmentID), DeptIDs: []string{t.DepartmentID}})
	}
	for _, r := range toolRows {
		tools = append(tools, r)
	}
	return []DirectoryGroup{
		{Kind: "employee", Title: "AI agents", Rows: nonNil(byLabel(agentRows))},
		{Kind: "person", Title: "Humans", Rows: nonNil(byLabel(personRows))},
		{Kind: "task", Title: "SOPs", Rows: nonNil(byLabel(taskRows))},
		{Kind: "tool", Title: "Tools", Rows: nonNil(byLabel(tools))},
	}
}

func nonNil(r []DirectoryRow) []DirectoryRow {
	if r == nil {
		return []DirectoryRow{}
	}
	return r
}
