package personas

import (
	"context"
	"errors"
	"testing"

	"github.com/rhl/businessos-backend/internal/founderos/pages/catalogdb"
)

const insert = `INSERT INTO founderos_personas
	(id, workspace_id, ord, name, archetype, tagline, summary, accent, north_star, pillars, connectors, metrics, brain_use, signature_play)
	VALUES ($1, $2, $3, $4, 'Operator', 'tag', 'sum', '#3df08c', 'north', $5, '["stripe"]', '["mrr"]', 'brain', 'play')`

const pillars = `[{"name":"Sales","focus":"pipeline","agents":["Closer"]}]`

func TestLoadReturnsTheWorkspacePersonasInOrder(t *testing.T) {
	pool := catalogdb.TestDB(t)
	ctx := context.Background()
	ws := catalogdb.Workspace(t, pool, "founderos")
	other := catalogdb.Workspace(t, pool, "vantage")
	for _, r := range []struct {
		id, ws string
		ord    int
	}{{"b", ws, 2}, {"a", ws, 1}, {"x", other, 0}} {
		if _, err := pool.Exec(ctx, insert, r.id, r.ws, r.ord, "P "+r.id, pillars); err != nil {
			t.Fatal(err)
		}
	}
	got, err := Load(ctx, pool, "founderos")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != "a" || got[1].ID != "b" {
		t.Fatalf("got %+v, want a then b from founderos only", got)
	}
	p := got[0]
	if p.Order != 1 || p.NorthStar != "north" || p.BrainUse != "brain" || p.SignaturePlay != "play" || p.Accent != "#3df08c" {
		t.Fatalf("fields not mapped: %+v", p)
	}
	if len(p.Pillars) != 1 || p.Pillars[0].Name != "Sales" || p.Pillars[0].Agents[0] != "Closer" {
		t.Fatalf("pillars JSON not round-tripped: %+v", p.Pillars)
	}
	if len(p.Connectors) != 1 || p.Connectors[0] != "stripe" || p.Metrics[0] != "mrr" {
		t.Fatalf("connectors/metrics not round-tripped: %+v", p)
	}
}

func TestLoadWithoutTheWorkspaceIsAnErrorNotAnEmptyList(t *testing.T) {
	pool := catalogdb.TestDB(t)
	_, err := Load(context.Background(), pool, "founderos")
	if !errors.Is(err, ErrNoWorkspace) {
		t.Fatalf("err = %v, want ErrNoWorkspace", err)
	}
}

func TestLoadRejectsARowThatBreaksTheSchema(t *testing.T) {
	pool := catalogdb.TestDB(t)
	ctx := context.Background()
	ws := catalogdb.Workspace(t, pool, "founderos")
	// A pillar with no agents fails PersonaSchema (agents.min(1)).
	if _, err := pool.Exec(ctx, insert, "bad", ws, 1, "Bad", `[{"name":"Sales","focus":"x","agents":[]}]`); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(ctx, pool, "founderos"); err == nil {
		t.Fatal("want a validation error for a pillar with no agents")
	}
}

func TestValidate(t *testing.T) {
	ok := Persona{ID: "a", Name: "A", Archetype: "x", Tagline: "x", Summary: "x", Accent: "#fff", NorthStar: "x",
		Pillars: []Pillar{{Name: "S", Focus: "f", Agents: []string{"a"}}}, Connectors: []string{"c"}, Metrics: []string{"m"}, BrainUse: "b", SignaturePlay: "s"}
	if err := ok.Validate(); err != nil {
		t.Fatal(err)
	}
	for name, mut := range map[string]func(*Persona){
		"no name":       func(p *Persona) { p.Name = "" },
		"no pillars":    func(p *Persona) { p.Pillars = nil },
		"no connectors": func(p *Persona) { p.Connectors = nil },
		"no metrics":    func(p *Persona) { p.Metrics = nil },
		"no north star": func(p *Persona) { p.NorthStar = "" },
	} {
		p := ok
		mut(&p)
		if p.Validate() == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}

// G-Brain is retired: the seeded persona copy names the Optimal Engine's
// "Brain" instead, in prose, pillar text and connector chips alike.
func TestRebrandRetiresGBrain(t *testing.T) {
	p := Rebrand(Persona{
		ID: "p", Name: "Agency Owner", Summary: "lives in a G-Brain world",
		BrainUse:      "Every agent reads the shared G-Brain for each client's brand guidelines",
		SignaturePlay: "the transcript hits G-Brain, then a memo drafts itself",
		Pillars:       []Pillar{{Name: "Ops", Focus: "G-Brain hygiene", Agents: []string{"G-Brain Curator"}}},
		Connectors:    []string{"Slack", "G-Brain"},
		Metrics:       []string{"G-Brain pages"},
	})
	if p.BrainUse != "Every agent reads the shared Brain for each client's brand guidelines" {
		t.Errorf("brainUse = %q", p.BrainUse)
	}
	if p.SignaturePlay != "the transcript hits Brain, then a memo drafts itself" || p.Summary != "lives in a Brain world" {
		t.Errorf("prose = %q / %q", p.SignaturePlay, p.Summary)
	}
	if p.Connectors[1] != "Brain" || p.Metrics[0] != "Brain pages" || p.Pillars[0].Focus != "Brain hygiene" || p.Pillars[0].Agents[0] != "Brain Curator" {
		t.Errorf("lists = %+v", p)
	}
}
