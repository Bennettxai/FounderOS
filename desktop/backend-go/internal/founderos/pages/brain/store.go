package brain

import (
	"math"
	"sort"
	"time"
)

// EngineContext is one engine context (a page) from the engine's GET /api/graph.
type EngineContext struct {
	ID       string    `json:"id"`
	Node     string    `json:"node"`
	Title    string    `json:"title"`
	URI      string    `json:"uri"`
	Abstract string    `json:"abstract"`
	Genre    string    `json:"genre"`
	Modified time.Time `json:"modified"`
}

type EngineEdge struct {
	Source   string `json:"source"`
	Target   string `json:"target"`
	Relation string `json:"relation"`
}

// WorkspaceGraph is one workspace's graph as its home engine returned it, or
// the reason it could not be read (Err).
type WorkspaceGraph struct {
	Workspace string          `json:"workspace"`
	Engine    string          `json:"engine"`
	Contexts  []EngineContext `json:"-"`
	Edges     []EngineEdge    `json:"-"`
	Err       string          `json:"error,omitempty"`
}

type Folder struct {
	Name  string `json:"name"`
	Files int    `json:"files"`
}

// Store is FounderOS v1's BrainOverview.store. On the bridge the "store" is the
// set of engine workspaces and a workspace is a top-level folder.
type Store struct {
	Path       string   `json:"path"`
	TotalFiles int      `json:"totalFiles"`
	Folders    []Folder `json:"folders"`
}

// StoreFrom counts the readable workspaces' pages. An unreadable workspace is
// left out (and surfaced by the caller), never counted as zero pages.
func StoreFrom(gs []WorkspaceGraph) Store {
	st := Store{Path: "optimal-engine", Folders: []Folder{}}
	for _, g := range gs {
		if g.Err != "" || len(g.Contexts) == 0 {
			continue
		}
		st.TotalFiles += len(g.Contexts)
		st.Folders = append(st.Folders, Folder{Name: g.Workspace, Files: len(g.Contexts)})
	}
	sort.SliceStable(st.Folders, func(i, j int) bool { return st.Folders[i].Files > st.Folders[j].Files })
	return st
}

// ModifiedTimes lists every readable page's engine modified_at. The engine
// stamps a page when it is ingested, so for imported pages this is the import
// time, not the original note's age.
func ModifiedTimes(gs []WorkspaceGraph) []time.Time {
	var out []time.Time
	for _, g := range gs {
		if g.Err != "" {
			continue
		}
		for _, c := range g.Contexts {
			if !c.Modified.IsZero() {
				out = append(out, c.Modified)
			}
		}
	}
	return out
}

type Cluster struct {
	Label string `json:"label"`
	Pages int    `json:"pages"`
}

const maxClusters = 6

// FoldersToClusters: folders → up to six clusters, the tail folds into misc.
func FoldersToClusters(folders []Folder) []Cluster {
	var nonEmpty []Folder
	for _, f := range folders {
		if f.Files > 0 {
			nonEmpty = append(nonEmpty, f)
		}
	}
	sort.SliceStable(nonEmpty, func(i, j int) bool {
		if nonEmpty[i].Files != nonEmpty[j].Files {
			return nonEmpty[i].Files > nonEmpty[j].Files
		}
		return nonEmpty[i].Name < nonEmpty[j].Name
	})
	out := []Cluster{}
	if len(nonEmpty) <= maxClusters {
		for _, f := range nonEmpty {
			out = append(out, Cluster{f.Name, f.Files})
		}
		return out
	}
	misc := 0
	for i, f := range nonEmpty {
		if i < maxClusters-1 {
			out = append(out, Cluster{f.Name, f.Files})
		} else {
			misc += f.Files
		}
	}
	return append(out, Cluster{"misc", misc})
}

type ProviderStatus struct {
	Provider  string `json:"provider"`
	Connected bool   `json:"connected"`
	Detail    string `json:"detail"`
}

// Satellites is BrainSatelliteData: the facts the panels around the graph show.
type Satellites struct {
	Mounted    bool      `json:"mounted"`
	StatusLine string    `json:"statusLine"`
	Pages      int       `json:"pages"`
	Folders    int       `json:"folders"`
	Clusters   []Cluster `json:"clusters"`
	FreshPct   *int      `json:"freshPct"` // share touched within 30 days; null with no notes
	StalePct   *int      `json:"stalePct"` // share untouched for 90+ days; null with no notes
}

func SummarizeSatellites(store Store, mtimes []time.Time, status ProviderStatus, now time.Time) Satellites {
	s := Satellites{Pages: store.TotalFiles, Folders: len(store.Folders), Clusters: FoldersToClusters(store.Folders)}
	s.Mounted = s.Pages > 0
	if s.Mounted {
		state := "offline"
		if status.Connected {
			state = "connected"
		}
		s.StatusLine = status.Provider + " · " + state + " · " + status.Detail
		if r := []rune(s.StatusLine); len(r) > 120 {
			s.StatusLine = string(r[:120])
		}
	} else {
		s.StatusLine = status.Provider + " · no notes mounted"
	}
	if n := len(mtimes); n > 0 {
		fresh, stale := 0, 0
		for _, t := range mtimes {
			age := now.Sub(t)
			if age <= 30*24*time.Hour {
				fresh++
			}
			if age >= 90*24*time.Hour {
				stale++
			}
		}
		f := int(math.Round(float64(fresh) / float64(n) * 100))
		st := int(math.Round(float64(stale) / float64(n) * 100))
		s.FreshPct, s.StalePct = &f, &st
	}
	return s
}
