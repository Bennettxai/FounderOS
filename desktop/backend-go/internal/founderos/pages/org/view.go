package org

import (
	"sort"
	"strings"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/paperclip"
	"github.com/rhl/businessos-backend/internal/founderos/pages/osdata"
)

// Live is the board strip: the overlay plus whether the board answered.
// An unreachable board is connected=false with its error, never an empty
// "all quiet" board.
type Live struct {
	Connected bool   `json:"connected"`
	Error     string `json:"error,omitempty"`
	LiveOverlay
}

// View is everything GET /pages/org serves.
type View struct {
	Departments   []osdata.Department `json:"departments"`
	Agents        []osdata.Agent      `json:"agents"`
	AgentNames    map[string]string   `json:"agentNames"`
	Conductor     *osdata.Agent       `json:"conductor"`
	Tree          Hierarchy           `json:"tree"`
	Crews         []Crew              `json:"crews"`
	Live          Live                `json:"live"`
	Ventures      []Venture           `json:"ventures"`
	LifeAreas     []LifeArea          `json:"lifeAreas"`
	LastBroadcast *osdata.Broadcast   `json:"lastBroadcast"`
}

// BuildView composes the page: the Conductor sits in the AI Head slot and
// every other agent goes into the department columns.
func BuildView(departments []osdata.Department, agents []osdata.Agent, board []paperclip.Agent, boardErr error, last *osdata.Broadcast) View {
	if departments == nil {
		departments = []osdata.Department{}
	}
	if agents == nil {
		agents = []osdata.Agent{}
	}
	departments, agents = retireGBrain(departments, agents)
	v := View{Departments: departments, Agents: agents, AgentNames: map[string]string{}, LifeAreas: LifeAreas, LastBroadcast: last}
	var columns []osdata.Agent
	for i, a := range agents {
		v.AgentNames[a.ID] = a.Name
		if a.ID == "conductor" {
			c := agents[i]
			v.Conductor = &c
			continue
		}
		columns = append(columns, a)
	}
	v.Tree = BuildHierarchy(departments, columns)

	switch {
	case boardErr != nil:
		v.Live = Live{Error: boardErr.Error(), LiveOverlay: OverlayLiveOrg(departments, nil)}
	case len(board) == 0:
		v.Live = Live{Error: "board answered with no agents", LiveOverlay: OverlayLiveOrg(departments, nil)}
	default:
		v.Live = Live{Connected: true, LiveOverlay: OverlayLiveOrg(departments, board)}
	}
	v.Crews = Crews(v.Tree, v.Live.LiveOverlay)

	for _, vt := range Ventures {
		set := VentureAgentSet(vt.ID)
		vt.AgentIDs = make([]string, 0, len(set))
		for id := range set {
			vt.AgentIDs = append(vt.AgentIDs, id)
		}
		sort.Strings(vt.AgentIDs)
		v.Ventures = append(v.Ventures, vt)
	}
	return v
}

// brainWording maps the seeded roster's v1 G-Brain wording onto v2's Brain
// (the bundled Optimal Engine). Longest phrases first.
var brainWording = strings.NewReplacer(
	"gbrain CLI", "Optimal Engine",
	"G-Brain", "Brain",
	"gbrain", "optimal-engine",
)

// retireGBrain returns copies of the roster with G-Brain retired from every
// visible string (roles, descriptions, models, taglines, tool chips).
func retireGBrain(departments []osdata.Department, agents []osdata.Agent) ([]osdata.Department, []osdata.Agent) {
	ds := make([]osdata.Department, len(departments))
	for i, d := range departments {
		d.Tagline = brainWording.Replace(d.Tagline)
		ds[i] = d
	}
	as := make([]osdata.Agent, len(agents))
	for i, a := range agents {
		a.Role = brainWording.Replace(a.Role)
		a.Description = brainWording.Replace(a.Description)
		a.Model = brainWording.Replace(a.Model)
		tools := make([]string, 0, len(a.Tools))
		seen := map[string]bool{}
		for _, t := range a.Tools {
			t = brainWording.Replace(t)
			if !seen[t] {
				seen[t] = true
				tools = append(tools, t)
			}
		}
		a.Tools = tools
		as[i] = a
	}
	return ds, as
}
