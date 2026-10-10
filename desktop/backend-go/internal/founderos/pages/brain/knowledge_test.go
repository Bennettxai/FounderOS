package brain

import (
	"sort"
	"testing"
)

// Ported from FounderOS v1 tests/knowledge-graph.test.ts.

func dept(id, name string) Department {
	return Department{ID: id, Name: name, Slug: id, Color: "#fff", Order: 1}
}

func agent(id, deptID, name string, tools []string, parent string) Agent {
	a := Agent{ID: id, DepartmentID: deptID, Name: name, Status: "active", Tier: "worker", Tools: tools, Instance: "builtin"}
	if parent != "" {
		a.ParentID = &parent
	}
	return a
}

func task(id, deptID, kind, assignee string) SopTask {
	return SopTask{ID: id, DepartmentID: deptID, Title: "Do " + id, Steps: []string{"one", "two", "three"}, AssigneeKind: kind, AssigneeID: assignee}
}

var (
	kgDepts  = []Department{dept("dept-tech", "TECH"), dept("dept-sales", "Sales")}
	kgAgents = []Agent{
		agent("conductor", "dept-tech", "Conductor", []string{"openclaw"}, ""),
		agent("data-agent", "dept-tech", "Data Agent", []string{"openclaw", "comms-feed"}, "conductor"),
		agent("sales-agent", "dept-sales", "Sales Agent", []string{"attio"}, ""),
	}
	kgPeople = []Person{{ID: "person-len", DepartmentID: "dept-sales", Name: "person-len", Role: "Human", Tools: []string{"fathom"}}}
	kgTasks  = []SopTask{
		task("sop-conductor", "dept-tech", "agent", "conductor"),
		task("sop-data", "dept-tech", "agent", "data-agent"),
		task("sop-sales", "dept-sales", "agent", "sales-agent"),
		task("sop-len", "dept-sales", "person", "person-len"),
	}
)

func build() KnowledgeGraph { return BuildKnowledgeGraph(kgAgents, kgDepts, kgPeople, kgTasks, nil) }

func nodesOf(g KnowledgeGraph, kind string) []KGNode {
	var out []KGNode
	for _, n := range g.Nodes {
		if n.Kind == kind {
			out = append(out, n)
		}
	}
	return out
}

func edgesOf(g KnowledgeGraph, kind string) []KGEdge {
	var out []KGEdge
	for _, e := range g.Edges {
		if e.Kind == kind {
			out = append(out, e)
		}
	}
	return out
}

func hasEdge(g KnowledgeGraph, want KGEdge) bool {
	for _, e := range g.Edges {
		if e == want {
			return true
		}
	}
	return false
}

func noDangling(t *testing.T, g KnowledgeGraph) {
	t.Helper()
	ids := map[string]bool{}
	for _, n := range g.Nodes {
		ids[n.ID] = true
	}
	for _, e := range g.Edges {
		if !ids[e.Source] || !ids[e.Target] {
			t.Fatalf("dangling edge %+v", e)
		}
	}
}

func TestKnowledgeGraphCoreAndPillars(t *testing.T) {
	g := build()
	selfs := nodesOf(g, "self")
	if len(selfs) != 1 || selfs[0].ID != SelfID || selfs[0].Ring != 0 || selfs[0].Label != "the operator" {
		t.Fatalf("self = %+v", selfs)
	}
	teams := nodesOf(g, "team")
	if len(teams) != 2 || teams[0].Ring != 1 {
		t.Fatalf("teams = %+v", teams)
	}
	for _, tm := range teams {
		if tm.Color == "" {
			t.Fatalf("team %s has no life-area tint", tm.ID)
		}
		if !hasEdge(g, KGEdge{Source: SelfID, Target: tm.ID, Kind: "pillar"}) {
			t.Fatalf("no pillar edge to %s", tm.ID)
		}
	}
	if c := LifeAreaColor("dept-sales"); c != "#ef4444" {
		t.Fatalf("sales tint = %q", c)
	}
}

func TestKnowledgeGraphTasksWorkersTools(t *testing.T) {
	g := build()
	tasks := nodesOf(g, "task")
	if len(tasks) != 4 || tasks[0].Ring != 2 || tasks[0].Label != "Do sop-conductor" {
		t.Fatalf("tasks = %+v", tasks)
	}
	if emp, per := nodesOf(g, "employee"), nodesOf(g, "person"); len(emp) != 3 || len(per) != 1 || emp[0].Ring != 3 || per[0].Ring != 3 {
		t.Fatalf("workers = %+v %+v", emp, per)
	}
	tools := nodesOf(g, "tool")
	var labels []string
	for _, tl := range tools {
		labels = append(labels, tl.Label)
		if tl.Ring != 4 {
			t.Fatalf("tool ring %+v", tl)
		}
	}
	if want := []string{"Attio", "Comms Feed", "Fathom", "Openclaw"}; !equal(labels, want) {
		t.Fatalf("tool labels = %v, want %v (deduped, sorted, prettified)", labels, want)
	}
	if len(edgesOf(g, "sop")) != 4 {
		t.Fatal("one sop edge per task")
	}
	does := edgesOf(g, "does")
	workers := map[string]int{}
	for _, e := range does {
		workers[e.Target]++
	}
	if len(does) != 4 || len(workers) != 4 {
		t.Fatalf("does edges must be monogamous: %+v", does)
	}
	if len(edgesOf(g, "member")) != 0 {
		t.Fatal("assigned workers reach their team through the task, not a member edge")
	}
	if !hasEdge(g, KGEdge{Source: "person:person-len", Target: "tool:fathom", Kind: "uses"}) || !hasEdge(g, KGEdge{Source: "emp:sales-agent", Target: "tool:attio", Kind: "uses"}) {
		t.Fatal("uses edges run worker→tool for agents and humans")
	}
	reports := edgesOf(g, "reports")
	if len(reports) != 1 || reports[0] != (KGEdge{Source: "emp:data-agent", Target: "emp:conductor", Kind: "reports"}) {
		t.Fatalf("reports = %+v", reports)
	}
	noDangling(t, g)
}

func TestUnassignedWorkerFallsBackToMember(t *testing.T) {
	agents := append([]Agent{}, kgAgents...)
	agents = append(agents, agent("loner", "dept-tech", "Loner", nil, ""))
	g := BuildKnowledgeGraph(agents, kgDepts, kgPeople, kgTasks, nil)
	if !hasEdge(g, KGEdge{Source: "emp:loner", Target: "team:dept-tech", Kind: "member"}) {
		t.Fatal("an unassigned worker falls back to a member edge")
	}
}

func TestBoardAgentsOrbitFounderos(t *testing.T) {
	g := BuildKnowledgeGraph(kgAgents, kgDepts, kgPeople, kgTasks, []BoardAgent{{ID: "b2", Name: "Forge"}, {ID: "b1", Name: "Conductor"}})
	board := nodesOf(g, "board")
	if len(board) != 2 || board[0].Label != "Conductor" || board[0].Ring != 1 {
		t.Fatalf("board = %+v (sorted by name, ring 1)", board)
	}
	if !hasEdge(g, KGEdge{Source: SelfID, Target: "board:b1", Kind: "board"}) {
		t.Fatal("board edge from the operator")
	}
}

var (
	multiDepts  = append(append([]Department{}, kgDepts...), dept("dept-clients", "Clients"))
	multiAgents = append(append([]Agent{}, kgAgents...), agent("client-roster", "dept-clients", "Client Roster", []string{"attio"}, ""))
	multiTasks  = append(append([]SopTask{}, kgTasks...), task("sop-roster", "dept-clients", "agent", "client-roster"))
)

func TestSharedToolsSplitPerDepartment(t *testing.T) {
	g := BuildKnowledgeGraph(multiAgents, multiDepts, kgPeople, multiTasks, nil)
	var attios []string
	for _, n := range nodesOf(g, "tool") {
		if ToolSlugOf(n.ID) == "attio" {
			attios = append(attios, n.ID)
			if n.Label != "Attio" {
				t.Fatalf("label %q", n.Label)
			}
		}
	}
	sort.Strings(attios)
	if !equal(attios, []string{"tool:attio@dept-clients", "tool:attio@dept-sales"}) {
		t.Fatalf("attio copies = %v", attios)
	}
	if !hasEdge(g, KGEdge{Source: "emp:sales-agent", Target: "tool:attio@dept-sales", Kind: "uses"}) || !hasEdge(g, KGEdge{Source: "emp:client-roster", Target: "tool:attio@dept-clients", Kind: "uses"}) {
		t.Fatal("uses edges point at the copy in the worker's own department")
	}
	if !hasEdge(g, KGEdge{Source: "emp:data-agent", Target: "tool:openclaw", Kind: "uses"}) {
		t.Fatal("single-department tools keep their plain id")
	}
	noDangling(t, g)
	for in, want := range map[string]string{"tool:attio@dept-sales": "attio", "tool:attio": "attio", "tool:comms-feed@dept-tech": "comms-feed"} {
		if got := ToolSlugOf(in); got != want {
			t.Fatalf("ToolSlugOf(%q) = %q", in, got)
		}
	}
}

func TestGraphDirectory(t *testing.T) {
	g := BuildKnowledgeGraph(multiAgents, multiDepts, kgPeople, multiTasks, nil)
	dir := GraphDirectory(multiAgents, multiDepts, kgPeople, multiTasks, g)
	var kinds []string
	for _, gr := range dir {
		kinds = append(kinds, gr.Kind)
		for i := 1; i < len(gr.Rows); i++ {
			if gr.Rows[i-1].Label > gr.Rows[i].Label {
				t.Fatalf("%s rows not alphabetized", gr.Kind)
			}
		}
	}
	if !equal(kinds, []string{"employee", "person", "task", "tool"}) {
		t.Fatalf("groups = %v", kinds)
	}
	if len(dir[0].Rows) != 4 || len(dir[1].Rows) != 1 || len(dir[2].Rows) != 5 || len(dir[3].Rows) != 4 {
		t.Fatalf("row counts %d %d %d %d", len(dir[0].Rows), len(dir[1].Rows), len(dir[2].Rows), len(dir[3].Rows))
	}
	var roster DirectoryRow
	for _, r := range dir[0].Rows {
		if r.Label == "Client Roster" {
			roster = r
		}
	}
	if roster.ID != "emp:client-roster" || roster.Sub != "Clients" || !equal(roster.DeptIDs, []string{"dept-clients"}) {
		t.Fatalf("roster row = %+v", roster)
	}
	for _, r := range dir[3].Rows {
		if r.ID == "attio" && (!equal(r.DeptIDs, []string{"dept-clients", "dept-sales"}) || r.Sub != "2 users") {
			t.Fatalf("attio row = %+v", r)
		}
	}
}

func TestGraphDeptOrder(t *testing.T) {
	if GraphDeptRank("dept-finance") != GraphDeptRank("dept-sales")+1 {
		t.Fatal("Finances sits immediately next to Sales")
	}
	if GraphDeptRank("dept-mystery") <= GraphDeptRank("dept-comms") {
		t.Fatal("unknown departments rank after the known ones")
	}
	if len(DeptExecTitles) != 6 || DeptExecTitles["dept-sales"] != "CRO" || DeptExecTitles["dept-clients"] != "COO" {
		t.Fatalf("exec titles = %v", DeptExecTitles)
	}
	ordered := OrderGraphDepartments([]Department{dept("dept-comms", "C"), dept("dept-finance", "F"), dept("dept-sales", "S")})
	if ordered[0].ID != "dept-sales" || ordered[1].ID != "dept-finance" {
		t.Fatalf("ordered = %+v", ordered)
	}
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
