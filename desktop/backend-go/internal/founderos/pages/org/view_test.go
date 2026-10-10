package org

import (
	"errors"
	"testing"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/paperclip"
	"github.com/rhl/businessos-backend/internal/founderos/pages/osdata"
)

func TestBuildViewPutsConductorInTheHeadSlot(t *testing.T) {
	ds := []osdata.Department{{ID: "dept-tech", Name: "TECH", Order: 1}}
	as := []osdata.Agent{agent("conductor", "dept-tech", "lead"), agent("stack-monitor", "dept-tech", "specialist")}
	v := BuildView(ds, as, []paperclip.Agent{live("Conductor", "running", ""), live("Forge", "idle", "")}, nil, nil)
	if v.Conductor == nil || v.Conductor.ID != "conductor" {
		t.Fatal("conductor slot")
	}
	if v.Tree.TotalAgents != 1 || len(v.Crews) != 1 || v.Crews[0].Pills[0].Agent.ID != "stack-monitor" {
		t.Fatalf("conductor must leave the columns: %+v", v.Crews)
	}
	if !v.Live.Connected || v.Live.Error != "" || v.Live.Conductor == nil || len(v.Live.Extras) != 1 {
		t.Fatalf("live: %+v", v.Live)
	}
	if v.AgentNames["stack-monitor"] != "stack-monitor" || len(v.Ventures) != 2 || len(v.Ventures[0].AgentIDs) == 0 || len(v.LifeAreas) != 7 {
		t.Fatalf("view: %+v", v)
	}
	if v.LastBroadcast != nil {
		t.Fatal("no broadcast yet")
	}
}

func TestBuildViewBoardDownIsHonest(t *testing.T) {
	v := BuildView(nil, nil, nil, errors.New("board unreachable (timeout)"), nil)
	if v.Live.Connected || v.Live.Error == "" || v.Live.Conductor != nil || len(v.Live.Extras) != 0 {
		t.Fatalf("%+v", v.Live)
	}
	if v.Crews == nil || v.Agents == nil || v.Departments == nil {
		t.Fatal("empty lists must serialize as []")
	}
	v = BuildView(nil, nil, []paperclip.Agent{}, nil, nil)
	if v.Live.Connected || v.Live.Error == "" {
		t.Fatalf("a board with no agents is not connected: %+v", v.Live)
	}
}

// G-Brain is retired in v2: the seeded roster still carries v1's G-Brain
// wording, so the board shows it as the Brain (the Optimal Engine).
func TestBuildViewRetiresGBrainWording(t *testing.T) {
	ds := []osdata.Department{{ID: "dept-tech", Name: "TECH", Tagline: "AI & automations · G-Brain.", Order: 1}}
	data := agent("data-agent", "dept-tech", "lead")
	data.Role = "G-Brain Analyst"
	data.Description = "Bound to the G-Brain instance. Runs gbrain doctor."
	data.Model = "gbrain CLI"
	data.Tools = []string{"gbrain", "brain-store", "ollama"}
	cond := agent("conductor", "dept-tech", "lead")
	cond.Tools = []string{"gbrain", "tmux"}
	v := BuildView(ds, []osdata.Agent{cond, data}, nil, errors.New("down"), nil)

	got := v.Crews[0].Leads[0]
	if got.Role != "Brain Analyst" || got.Description != "Bound to the Brain instance. Runs optimal-engine doctor." || got.Model != "Optimal Engine" {
		t.Fatalf("lead wording: %+v", got)
	}
	for _, tool := range v.Crews[0].Tools {
		if tool == "gbrain" {
			t.Fatalf("gbrain tool chip survived: %v", v.Crews[0].Tools)
		}
	}
	if v.Crews[0].Tools[0] != "optimal-engine" {
		t.Fatalf("tools: %v", v.Crews[0].Tools)
	}
	if v.Conductor.Tools[0] != "optimal-engine" || v.Departments[0].Tagline != "AI & automations · Brain." {
		t.Fatalf("conductor/department: %+v %+v", v.Conductor, v.Departments[0])
	}
	if v.Agents[1].Role != "Brain Analyst" {
		t.Fatalf("agents list: %+v", v.Agents[1])
	}
}
