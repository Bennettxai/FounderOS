package seed

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/rhl/businessos-backend/internal/founderos/brainimport"
	"github.com/rhl/businessos-backend/internal/founderos/topology"
)

// brain/ is the demo's knowledge: the markdown v1 generates from its seed
// (demo-v1 scripts/generate-brain-docs.ts → lib/brain-docs.ts: one page per
// agent, SOP, tool, human and pillar, wikilinked), captured on DumpedOn. The
// only edit is the rename v2 applies everywhere: G-Brain is the Optimal
// Engine ("the Brain"), so tools/gbrain.md ships as tools/optimal-engine.md
// (with the engine's own one-line description).
//
//go:embed brain
var brainFS embed.FS

// BrainDocs is how many knowledge pages the demo ships.
const BrainDocs = 114

// MaterializeBrain writes the knowledge pages into dir as a brain-store
// (agents/, sops/, tools/, people/, org/) and returns how many it wrote.
func MaterializeBrain(dir string) (int, error) {
	n := 0
	err := fs.WalkDir(brainFS, "brain", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		raw, err := brainFS.ReadFile(p)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel("brain", filepath.FromSlash(p))
		out := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return err
		}
		n++
		return os.WriteFile(out, raw, 0o644)
	})
	return n, err
}

// BrainReport is what one SeedBrain run did.
type BrainReport struct {
	Planned  int                   `json:"planned"`  // pages routed to an engine
	Skipped  int                   `json:"skipped"`  // too short to be worth a page (brainimport.MinChars)
	Landed   int                   `json:"landed"`   // already in the engine: not sent again
	Ingested int                   `json:"ingested"` // sent this run
	Errors   []brainimport.Failure `json:"errors,omitempty"`
}

// SeedBrain imports the demo's knowledge into the Optimal Engine, routed by
// the engine topology (every page lands in HQ, the founderos workspace). It
// creates the topology's workspaces, registers each page's slug as an entity
// so the wikilinks become graph edges, and sends only the pages the engine
// does not already hold: a re-run ingests nothing.
func SeedBrain(topo *topology.Topology, engines map[string]brainimport.Endpoint) (*BrainReport, error) {
	dir, err := os.MkdirTemp("", "founderos-brain-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	store := filepath.Join(dir, "store")
	if _, err := MaterializeBrain(store); err != nil {
		return nil, err
	}
	items, skipped, err := brainimport.Plan(store, topo)
	if err != nil {
		return nil, err
	}
	rep := &BrainReport{Planned: len(items), Skipped: skipped}
	if err := brainimport.EnsureWorkspaces(topo, engines); err != nil {
		return rep, err
	}
	if err := brainimport.EnsureEntities(items, engines); err != nil {
		return rep, err
	}
	rest, landed, err := brainimport.WithoutLanded(items, engines)
	if err != nil {
		return rep, err
	}
	rep.Landed = landed
	// The engine is the record (WithoutLanded); the ledger is per run only.
	run, err := brainimport.Run(rest, engines, filepath.Join(dir, "ledger.json"), false)
	if run != nil {
		rep.Ingested, rep.Errors = run.Ingested, run.Errors
	}
	return rep, err
}
