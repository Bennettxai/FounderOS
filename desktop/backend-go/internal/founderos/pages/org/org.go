// Package org is the /org hierarchy board's pure logic: FounderOS v1
// lib/hierarchy.ts (the org tree), lib/org-live.ts (the Paperclip board
// overlay, matched by name), the venture lens (lib/ventures.ts) and the
// life-area tints (lib/life-map.ts LIFE_AREAS), plus the per-crew shaping the
// frozen app/org/page.tsx markup does inline.
package org

import (
	"sort"
	"strings"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/paperclip"
	"github.com/rhl/businessos-backend/internal/founderos/pages/osdata"
)

// ---- lib/hierarchy.ts -------------------------------------------------------

type AgentNode struct {
	Agent    osdata.Agent `json:"agent"`
	Children []AgentNode  `json:"children"`
}

type DepartmentNode struct {
	Department osdata.Department `json:"department"`
	Roots      []AgentNode       `json:"roots"`
}

type Hierarchy struct {
	Departments  []DepartmentNode `json:"departments"`
	TotalAgents  int              `json:"totalAgents"`
	ActiveAgents int              `json:"activeAgents"`
}

var tierOrder = map[string]int{"lead": 0, "specialist": 1, "worker": 2}

func buildNodes(agents []osdata.Agent) []AgentNode {
	ids := map[string]bool{}
	for _, a := range agents {
		ids[a.ID] = true
	}
	sorted := append([]osdata.Agent(nil), agents...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if ti, tj := tierOrder[sorted[i].Tier], tierOrder[sorted[j].Tier]; ti != tj {
			return ti < tj
		}
		return sorted[i].Name < sorted[j].Name
	})
	// an agent whose parent left the roster surfaces at the root
	parentOf := func(a osdata.Agent) string {
		if a.ParentID != nil && ids[*a.ParentID] {
			return *a.ParentID
		}
		return ""
	}
	var childrenOf func(parent string) []AgentNode
	childrenOf = func(parent string) []AgentNode {
		out := []AgentNode{}
		for _, a := range sorted {
			if parentOf(a) == parent {
				out = append(out, AgentNode{Agent: a, Children: childrenOf(a.ID)})
			}
		}
		return out
	}
	return childrenOf("")
}

// FlattenNodes walks the tree depth-first.
func FlattenNodes(nodes []AgentNode) []AgentNode {
	out := []AgentNode{}
	for _, n := range nodes {
		out = append(out, n)
		out = append(out, FlattenNodes(n.Children)...)
	}
	return out
}

// BuildHierarchy shapes departments/agents into operator → department →
// instance agents → workers (via parentId).
func BuildHierarchy(departments []osdata.Department, agents []osdata.Agent) Hierarchy {
	deps := append([]osdata.Department(nil), departments...)
	sort.SliceStable(deps, func(i, j int) bool { return deps[i].Order < deps[j].Order })
	h := Hierarchy{Departments: []DepartmentNode{}, TotalAgents: len(agents)}
	for _, d := range deps {
		var mine []osdata.Agent
		for _, a := range agents {
			if a.DepartmentID == d.ID {
				mine = append(mine, a)
			}
		}
		h.Departments = append(h.Departments, DepartmentNode{Department: d, Roots: buildNodes(mine)})
	}
	for _, a := range agents {
		if a.Status == "active" {
			h.ActiveAgents++
		}
	}
	return h
}

// ---- lib/org-live.ts --------------------------------------------------------

// LiveOverlay matches the board's agents onto the pillars by name.
type LiveOverlay struct {
	Conductor    *paperclip.Agent           `json:"conductor"`
	ByDepartment map[string]paperclip.Agent `json:"byDepartment"`
	Extras       []paperclip.Agent          `json:"extras"`
}

func OverlayLiveOrg(departments []osdata.Department, live []paperclip.Agent) LiveOverlay {
	byName := map[string]string{}
	for _, d := range departments {
		byName[strings.ToLower(d.Name)] = d.ID
	}
	o := LiveOverlay{ByDepartment: map[string]paperclip.Agent{}, Extras: []paperclip.Agent{}}
	for _, a := range live {
		key := strings.ToLower(a.Name)
		if key == "conductor" {
			c := a
			o.Conductor = &c
			continue
		}
		if id, ok := byName[key]; ok {
			o.ByDepartment[id] = a
		} else {
			o.Extras = append(o.Extras, a)
		}
	}
	return o
}

// ---- crews (the department column in app/org/page.tsx) ----------------------

// Crew is one department column: the instance leads at the root, every other
// node as a pill, the union of tools, its life-area tint and its live lead.
type Crew struct {
	Department osdata.Department `json:"department"`
	Area       *LifeArea         `json:"area"`
	Live       *paperclip.Agent  `json:"live"`
	Leads      []osdata.Agent    `json:"leads"`
	Pills      []AgentNode       `json:"pills"`
	Tools      []string          `json:"tools"`
	Empty      bool              `json:"empty"`
}

func Crews(h Hierarchy, live LiveOverlay) []Crew {
	out := []Crew{}
	for _, dn := range h.Departments {
		c := Crew{Department: dn.Department, Area: LifeAreaForDepartment(dn.Department.ID), Leads: []osdata.Agent{}, Pills: []AgentNode{}, Tools: []string{}}
		if l, ok := live.ByDepartment[dn.Department.ID]; ok {
			c.Live = &l
		}
		for _, r := range dn.Roots {
			if r.Agent.Tier == "lead" {
				c.Leads = append(c.Leads, r.Agent)
				c.Pills = append(c.Pills, r.Children...)
			} else {
				c.Pills = append(c.Pills, r)
			}
		}
		all := FlattenNodes(dn.Roots)
		seen := map[string]bool{}
		for _, n := range all {
			for _, tool := range n.Agent.Tools {
				if !seen[tool] {
					seen[tool] = true
					c.Tools = append(c.Tools, tool)
				}
			}
		}
		c.Empty = len(all) == 0
		out = append(out, c)
	}
	return out
}
