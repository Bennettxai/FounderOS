package org

import (
	"reflect"
	"testing"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/paperclip"
	"github.com/rhl/businessos-backend/internal/founderos/pages/osdata"
)

func dept(id string, order int) osdata.Department {
	return osdata.Department{ID: id, Name: id[5:], Slug: id[5:], Tagline: "t", Color: "#fff", Order: order}
}

func agent(id, dep, tier string, parent ...string) osdata.Agent {
	a := osdata.Agent{ID: id, DepartmentID: dep, Name: id, Role: "r", Status: "active", Tier: tier, Tools: []string{}, Instance: "builtin"}
	if len(parent) > 0 {
		p := parent[0]
		a.ParentID = &p
	}
	return a
}

func ids(nodes []AgentNode) []string {
	out := []string{}
	for _, n := range nodes {
		out = append(out, n.Agent.ID)
	}
	return out
}

// tests/hierarchy.test.ts
func TestBuildHierarchyNestsByParent(t *testing.T) {
	tree := BuildHierarchy([]osdata.Department{dept("dept-comms", 1)}, []osdata.Agent{
		agent("comms-agent", "dept-comms", "lead"),
		agent("gmail-worker", "dept-comms", "worker", "comms-agent"),
		agent("slack-worker", "dept-comms", "worker", "comms-agent"),
	})
	roots := tree.Departments[0].Roots
	if !reflect.DeepEqual(ids(roots), []string{"comms-agent"}) || !reflect.DeepEqual(ids(roots[0].Children), []string{"gmail-worker", "slack-worker"}) {
		t.Fatalf("tree: %+v", roots)
	}
}

func TestBuildHierarchyDeepNestingAndOrphans(t *testing.T) {
	tree := BuildHierarchy([]osdata.Department{dept("dept-x", 1)}, []osdata.Agent{
		agent("top", "dept-x", "lead"), agent("mid", "dept-x", "specialist", "top"), agent("leaf", "dept-x", "worker", "mid"),
	})
	if tree.Departments[0].Roots[0].Children[0].Children[0].Agent.ID != "leaf" {
		t.Fatal("deep nesting")
	}
	tree = BuildHierarchy([]osdata.Department{dept("dept-x", 1)}, []osdata.Agent{
		agent("solo", "dept-x", "lead"), agent("orphan", "dept-x", "worker", "gone-agent"),
	})
	if !reflect.DeepEqual(ids(tree.Departments[0].Roots), []string{"solo", "orphan"}) {
		t.Fatalf("orphan falls back to root: %v", ids(tree.Departments[0].Roots))
	}
}

func TestBuildHierarchyOrdersDepartmentsAndCounts(t *testing.T) {
	planned := agent("b1", "dept-b", "lead")
	planned.Status = "planned"
	tree := BuildHierarchy([]osdata.Department{dept("dept-b", 2), dept("dept-a", 1)}, []osdata.Agent{agent("a1", "dept-a", "lead"), planned})
	if tree.Departments[0].Department.ID != "dept-a" || tree.TotalAgents != 2 || tree.ActiveAgents != 1 {
		t.Fatalf("%+v", tree)
	}
	tree = BuildHierarchy([]osdata.Department{dept("dept-x", 1)}, []osdata.Agent{
		agent("top", "dept-x", "lead"), agent("child", "dept-x", "worker", "top"), agent("solo", "dept-x", "specialist"),
	})
	if got := ids(FlattenNodes(tree.Departments[0].Roots)); !reflect.DeepEqual(got, []string{"top", "child", "solo"}) {
		t.Fatalf("flatten depth-first: %v", got)
	}
}

// tests/org-live.test.ts
var depts = []osdata.Department{
	{ID: "dept-sales", Name: "Sales"},
	{ID: "dept-marketing-growth", Name: "Marketing/Growth"},
	{ID: "dept-tech", Name: "TECH"},
	{ID: "dept-clients", Name: "Clients"},
}

func live(name, status string, model string) paperclip.Agent {
	a := paperclip.Agent{ID: "id-" + name, Name: name, Status: status}
	if model != "" {
		a.Model = &model
	}
	return a
}

func TestOverlayLiveOrgMatchesPillarsByName(t *testing.T) {
	o := OverlayLiveOrg(depts, []paperclip.Agent{
		live("Conductor", "running", "claude-fable-5"),
		live("sales", "idle", "claude-sonnet-4-6"),
		live("TECH", "error", "gpt-5.5"),
		live("Forge", "running", ""),
		live("Hermes Workers", "idle", ""),
	})
	if o.Conductor == nil || o.Conductor.Status != "running" {
		t.Fatal("conductor")
	}
	if *o.ByDepartment["dept-sales"].Model != "claude-sonnet-4-6" || o.ByDepartment["dept-tech"].Status != "error" {
		t.Fatalf("by dept: %+v", o.ByDepartment)
	}
	if _, has := o.ByDepartment["dept-clients"]; has {
		t.Fatal("clients has no board counterpart")
	}
	if len(o.Extras) != 2 || o.Extras[0].Name != "Forge" || o.Extras[1].Name != "Hermes Workers" {
		t.Fatalf("extras: %+v", o.Extras)
	}
}

func TestOverlayLiveOrgEmptyIsInert(t *testing.T) {
	o := OverlayLiveOrg(depts, nil)
	if o.Conductor != nil || len(o.ByDepartment) != 0 || o.Extras == nil || len(o.Extras) != 0 {
		t.Fatalf("%+v", o)
	}
}

// tests/ventures.test.ts
func TestVenturesAreTheTwoIncomeSources(t *testing.T) {
	if len(Ventures) != 2 || Ventures[0].ID != "vantage" || Ventures[1].ID != "launchpad-cohort" {
		t.Fatalf("%+v", Ventures)
	}
	if Ventures[0].Color != "#00ffaa" || Ventures[1].Color != "#d9263f" {
		t.Fatal("brand colors")
	}
	areas := map[string]bool{}
	colors := map[string]bool{}
	for _, a := range LifeAreas {
		areas[a.ID], colors[a.Color] = true, true
	}
	for _, v := range Ventures {
		if colors[v.Color] || len(v.Focus) == 0 {
			t.Fatalf("%s collides or has no focus", v.ID)
		}
		for area, list := range v.AreaAgents {
			if !areas[area] {
				t.Fatalf("unknown area %s", area)
			}
			_ = list
		}
		for _, req := range []string{"marketing", "communication", "finances"} {
			if len(v.AreaAgents[req]) == 0 {
				t.Fatalf("%s has no %s", v.ID, req)
			}
		}
	}
	set := VentureAgentSet("vantage")
	if !set["conductor"] || !set["vantage-sales"] || set["whatsapp-worker"] {
		t.Fatalf("vantage set: %v", set)
	}
	if len(VentureAgentSet("nope")) != 0 {
		t.Fatal("unknown venture")
	}
}

func TestLifeAreaForDepartmentTakesFirstMatch(t *testing.T) {
	if a := LifeAreaForDepartment("dept-tech"); a == nil || a.ID != "knowledge" {
		t.Fatalf("dept-tech → %+v", a)
	}
	if LifeAreaForDepartment("dept-none") != nil {
		t.Fatal("unknown dept")
	}
}

func TestCrewsSplitLeadsFromPillsAndCollectTools(t *testing.T) {
	sales := agent("sales-agent", "dept-sales", "lead")
	sales.Tools = []string{"typeform", "stripe"}
	worker := agent("crm-pulse", "dept-sales", "worker", "sales-agent")
	worker.Tools = []string{"attio", "stripe"}
	solo := agent("rogue", "dept-sales", "specialist")
	tree := BuildHierarchy([]osdata.Department{
		{ID: "dept-sales", Name: "Sales", Color: "#ffd166", Order: 1},
		{ID: "dept-clients", Name: "Clients", Color: "#14b8a6", Order: 2},
	}, []osdata.Agent{sales, worker, solo})
	ov := OverlayLiveOrg(depts, []paperclip.Agent{live("Sales", "running", "")})
	crews := Crews(tree, ov)
	c := crews[0]
	if !reflect.DeepEqual(idsOf(c.Leads), []string{"sales-agent"}) || !reflect.DeepEqual(ids(c.Pills), []string{"crm-pulse", "rogue"}) {
		t.Fatalf("leads %v pills %v", idsOf(c.Leads), ids(c.Pills))
	}
	if !reflect.DeepEqual(c.Tools, []string{"typeform", "stripe", "attio"}) || c.Area == nil || c.Area.ID != "sales" || c.Live == nil || c.Empty {
		t.Fatalf("crew: %+v", c)
	}
	if !crews[1].Empty || crews[1].Live != nil || len(crews[1].Tools) != 0 {
		t.Fatalf("empty crew: %+v", crews[1])
	}
}

func idsOf(list []osdata.Agent) []string {
	out := []string{}
	for _, a := range list {
		out = append(out, a.ID)
	}
	return out
}
