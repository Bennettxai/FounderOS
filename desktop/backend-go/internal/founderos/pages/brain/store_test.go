package brain

import (
	"fmt"
	"math"
	"strings"
	"testing"
	"time"
)

// Ported from tests/brain-viz.test.ts (foldersToClusters) and
// tests/brain-satellites.test.ts (summarizeSatellites).

func TestFoldersToClusters(t *testing.T) {
	got := FoldersToClusters([]Folder{{"inbox", 2}, {"sales", 3}, {"empty", 0}, {"tech", 1}})
	if len(got) != 3 || got[0] != (Cluster{"sales", 3}) || got[1] != (Cluster{"inbox", 2}) || got[2] != (Cluster{"tech", 1}) {
		t.Fatalf("clusters = %+v (sorted by pages, empties dropped)", got)
	}
	var many []Folder
	for i := 0; i < 8; i++ {
		many = append(many, Folder{fmt.Sprintf("f%d", i), 10 - i})
	}
	capped := FoldersToClusters(many)
	if len(capped) != 6 || capped[5].Label != "misc" || capped[5].Pages != 5+4+3 {
		t.Fatalf("capped = %+v (six clusters, tail folds into misc)", capped)
	}
	if len(FoldersToClusters(nil)) != 0 {
		t.Fatal("empty store, no clusters")
	}
}

var now = time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)

const day = 24 * time.Hour

func TestSummarizeSatellites(t *testing.T) {
	store := Store{Path: "optimal-engine", TotalFiles: 6, Folders: []Folder{{"sales", 3}, {"inbox", 2}, {"tech", 1}}}
	mtimes := []time.Time{now.Add(-1 * day), now.Add(-10 * day), now.Add(-40 * day), now.Add(-100 * day), now.Add(-200 * day), now.Add(-365 * day)}
	s := SummarizeSatellites(store, mtimes, ProviderStatus{Provider: "optimal-engine", Connected: true, Detail: "2/2 engines up"}, now)
	if !s.Mounted || s.Pages != 6 || s.Folders != 3 || len(s.Clusters) != 3 || s.Clusters[0].Label != "sales" {
		t.Fatalf("satellites = %+v", s)
	}
	if s.FreshPct == nil || *s.FreshPct != 33 || s.StalePct == nil || *s.StalePct != 50 {
		t.Fatalf("fresh/stale = %v/%v", s.FreshPct, s.StalePct)
	}
	if !strings.Contains(s.StatusLine, "optimal-engine · connected") {
		t.Fatalf("status line %q", s.StatusLine)
	}
}

func TestSatellitesNoNotesMeansNoPercentages(t *testing.T) {
	s := SummarizeSatellites(Store{Path: "x"}, nil, ProviderStatus{Provider: "optimal-engine", Detail: "no engines"}, now)
	if s.Mounted || s.Pages != 0 || s.FreshPct != nil || s.StalePct != nil || len(s.Clusters) != 0 {
		t.Fatalf("empty satellites = %+v (never 0%% dressed as a fact)", s)
	}
	if !strings.Contains(s.StatusLine, "no notes mounted") {
		t.Fatalf("status line %q", s.StatusLine)
	}
}

func graphs() []WorkspaceGraph {
	return []WorkspaceGraph{
		{Workspace: "founderos", Engine: "hub", Contexts: []EngineContext{
			{ID: "a", Node: "inbox", Title: "Alpha", Genre: "sop", Modified: now.Add(-day)},
			{ID: "b", Node: "inbox", Title: "Beta", Genre: "note", Modified: now.Add(-100 * day)},
			{ID: "c", Node: "tech", Title: "Gamma", Genre: "note", Modified: now},
		}, Edges: []EngineEdge{{Source: "a", Target: "b", Relation: "cross_ref"}, {Source: "a", Target: "c", Relation: "cross_ref"}, {Source: "a", Target: "zzz", Relation: "cross_ref"}}},
		{Workspace: "vantage", Engine: "hub", Contexts: []EngineContext{{ID: "m", Node: "team", Title: "Vantage SOP", Modified: now}}},
		{Workspace: "personal", Engine: "macbook", Err: "engine macbook unreachable"},
	}
}

func TestStoreFromEngineGraphs(t *testing.T) {
	st := StoreFrom(graphs())
	if st.Path != "optimal-engine" || st.TotalFiles != 4 || len(st.Folders) != 2 || st.Folders[0] != (Folder{"founderos", 3}) {
		t.Fatalf("store = %+v (a workspace is a folder; the unreachable one is left out, not zero)", st)
	}
	if got := ModifiedTimes(graphs()); len(got) != 4 {
		t.Fatalf("mtimes = %v", got)
	}
}

func TestBuildMemoryGraph(t *testing.T) {
	g := BuildMemoryGraph(graphs(), 120)
	byID := map[string]MemoryNode{}
	for _, n := range g.Nodes {
		byID[n.ID] = n
		if math.Hypot(n.VX, n.VY) > 1.0001 {
			t.Fatalf("node %s outside the unit disc: %v,%v", n.ID, n.VX, n.VY)
		}
	}
	if byID["folder:founderos"].Type != "folder" || byID["folder:vantage"].Type != "folder" {
		t.Fatalf("hubs missing: %+v", g.Nodes)
	}
	a := byID["founderos:a"]
	if a.Type != "page" || a.Label != "Alpha" || a.Folder != "founderos" || a.Links != 2 {
		t.Fatalf("page a = %+v (links count kept cross-refs)", a)
	}
	for _, e := range g.Edges {
		if _, ok := byID[e.Source]; !ok {
			t.Fatalf("dangling %+v", e)
		}
		if _, ok := byID[e.Target]; !ok {
			t.Fatalf("dangling %+v", e)
		}
	}
	var wl, mem int
	for _, e := range g.Edges {
		switch e.Type {
		case "wikilink":
			wl++
		case "member":
			mem++
		}
	}
	if wl != 2 || mem != 4 {
		t.Fatalf("edges: %d wikilink, %d member", wl, mem)
	}
	// capped to the most-linked pages
	small := BuildMemoryGraph(graphs(), 1)
	pages := 0
	for _, n := range small.Nodes {
		if n.Type == "page" {
			pages++
			if n.ID != "founderos:a" {
				t.Fatalf("kept %s, want the most-linked page", n.ID)
			}
		}
	}
	if pages != 1 {
		t.Fatalf("cap ignored: %d pages", pages)
	}
	if again := BuildMemoryGraph(graphs(), 120); fmt.Sprint(again) != fmt.Sprint(g) {
		t.Fatal("layout must be deterministic")
	}
}

// A workspace dense with cross-refs must not crowd the rest out of the core:
// each readable workspace keeps a share of the cap in proportion to its pages.
func TestMemoryGraphSharesTheCapAcrossWorkspaces(t *testing.T) {
	dense := WorkspaceGraph{Workspace: "vantage", Engine: "hub"}
	for i := 0; i < 20; i++ {
		dense.Contexts = append(dense.Contexts, EngineContext{ID: fmt.Sprintf("m%d", i), Title: fmt.Sprintf("M%02d", i)})
		if i > 0 {
			dense.Edges = append(dense.Edges, EngineEdge{Source: "m0", Target: fmt.Sprintf("m%d", i)})
		}
	}
	sparse := WorkspaceGraph{Workspace: "personal", Engine: "macbook"}
	for i := 0; i < 60; i++ {
		sparse.Contexts = append(sparse.Contexts, EngineContext{ID: fmt.Sprintf("p%d", i), Title: fmt.Sprintf("P%02d", i)})
	}
	g := BuildMemoryGraph([]WorkspaceGraph{dense, sparse}, 8)
	per := map[string]int{}
	for _, n := range g.Nodes {
		if n.Type == "page" {
			per[n.Folder]++
		}
	}
	if per["vantage"] != 2 || per["personal"] != 6 {
		t.Fatalf("pages per workspace = %v, want 2/6 (a share of 8 by page count)", per)
	}
}

// FounderOS v1's memory core (lib/memory-core.ts) fills the disc uniformly:
// equal-area rings each hold their share of the linked notes (no clump around
// a hub), each workspace keeps its own angular wedge, and unlinked notes ring
// the rim as the orphan halo. The port lays the engines out the same way.
func TestMemoryGraphFillsTheDiscUniformly(t *testing.T) {
	var gs []WorkspaceGraph
	for _, ws := range []string{"founderos", "vantage", "personal"} {
		g := WorkspaceGraph{Workspace: ws, Engine: "hub"}
		for i := 0; i < 40; i++ {
			g.Contexts = append(g.Contexts, EngineContext{ID: fmt.Sprintf("c%d", i), Title: fmt.Sprintf("%s %02d", ws, i)})
			if i > 0 && i < 34 { // 34 linked notes, 6 orphans per workspace
				g.Edges = append(g.Edges, EngineEdge{Source: "c0", Target: fmt.Sprintf("c%d", i)})
			}
		}
		gs = append(gs, g)
	}
	g := BuildMemoryGraph(gs, 120)
	var linked, inner int
	wedge := map[string][2]float64{} // min/max angle per workspace, unwrapped from the hub
	hubAngle := map[string]float64{}
	for _, n := range g.Nodes {
		if n.Type == "folder" {
			hubAngle[n.Folder] = math.Atan2(n.VY, n.VX)
		}
	}
	for _, n := range g.Nodes {
		if n.Type != "page" {
			continue
		}
		r := math.Hypot(n.VX, n.VY)
		if n.Links == 0 {
			if r < 0.88 {
				t.Fatalf("orphan %s at r=%.3f, want it on the rim halo", n.ID, r)
			}
			continue
		}
		linked++
		if r < 0.86*math.Sqrt(0.5) {
			inner++
		}
		d := math.Remainder(math.Atan2(n.VY, n.VX)-hubAngle[n.Folder], 2*math.Pi)
		w := wedge[n.Folder]
		wedge[n.Folder] = [2]float64{math.Min(w[0], d), math.Max(w[1], d)}
	}
	// the inner half of the disc's area holds about half the linked notes
	if frac := float64(inner) / float64(linked); frac < 0.35 || frac > 0.65 {
		t.Fatalf("inner half-area holds %.2f of the linked notes, want ~0.5 (uniform fill, no centre clump)", frac)
	}
	// each workspace stays inside its own third of the disc
	for ws, w := range wedge {
		if w[0] < -math.Pi/3-0.01 || w[1] > math.Pi/3+0.01 {
			t.Fatalf("%s spills out of its wedge: %.2f..%.2f rad from its hub", ws, w[0], w[1])
		}
	}
}
