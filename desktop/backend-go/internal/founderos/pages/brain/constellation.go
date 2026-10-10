package brain

import (
	"math"
	"sort"
)

// MemoryNode / MemoryEdge / MemoryGraph: FounderOS v1 lib/memory-core.ts, the
// constellation drawn at the core of the knowledge graph. On the bridge it is
// distilled from the engines' GET /api/graph (contexts + cross_ref edges):
// one folder hub per workspace, its most-linked pages around it.
type MemoryNode struct {
	ID      string  `json:"id"`
	Type    string  `json:"type"` // folder | page
	Label   string  `json:"label"`
	Folder  string  `json:"folder"`
	Genre   string  `json:"genre,omitempty"`
	Excerpt string  `json:"excerpt"`
	VX      float64 `json:"vx"` // layout coords in the unit disc
	VY      float64 `json:"vy"`
	Cluster int     `json:"cluster"`
	Links   int     `json:"links"`
}

type MemoryEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type"` // member | wikilink
}

type MemoryGraph struct {
	Nodes []MemoryNode `json:"nodes"`
	Edges []MemoryEdge `json:"edges"`
}

// DefaultMaxPages mirrors memory-core's DEFAULT_MAX_PAGES.
const DefaultMaxPages = 120

type pageRef struct {
	ws  string
	ctx EngineContext
	deg int
}

// BuildMemoryGraph keeps the maxPages most cross-referenced pages across the
// readable workspaces (then title, then id: fully deterministic; each
// workspace gets a share of the cap by page count), their
// workspace hubs, and only the edges between kept nodes. The layout replaces
// memory-core's force simulation with a deterministic uniform fill (below).
func BuildMemoryGraph(gs []WorkspaceGraph, maxPages int) MemoryGraph {
	if maxPages <= 0 {
		maxPages = DefaultMaxPages
	}
	deg := map[string]int{}
	exists := map[string]bool{}
	var pages []pageRef
	var wikilinks []MemoryEdge
	for _, g := range gs {
		if g.Err != "" {
			continue
		}
		for _, c := range g.Contexts {
			exists[g.Workspace+":"+c.ID] = true
		}
		for _, e := range g.Edges {
			s, t := g.Workspace+":"+e.Source, g.Workspace+":"+e.Target
			if !exists[s] || !exists[t] || s == t {
				continue
			}
			deg[s]++
			deg[t]++
			wikilinks = append(wikilinks, MemoryEdge{Source: s, Target: t, Type: "wikilink"})
		}
		for _, c := range g.Contexts {
			pages = append(pages, pageRef{ws: g.Workspace, ctx: c})
		}
	}
	for i := range pages {
		pages[i].deg = deg[pages[i].ws+":"+pages[i].ctx.ID]
	}
	sort.SliceStable(pages, func(i, j int) bool {
		a, b := pages[i], pages[j]
		if a.deg != b.deg {
			return a.deg > b.deg
		}
		if a.ctx.Title != b.ctx.Title {
			return a.ctx.Title < b.ctx.Title
		}
		return a.ws+":"+a.ctx.ID < b.ws+":"+b.ctx.ID
	})
	pages = fairShare(pages, maxPages)

	// hubs in workspace order of first appearance in gs
	var hubs []string
	perHub := map[string][]pageRef{}
	for _, g := range gs {
		for _, p := range pages {
			if p.ws == g.Workspace {
				if len(perHub[g.Workspace]) == 0 {
					hubs = append(hubs, g.Workspace)
				}
				perHub[g.Workspace] = append(perHub[g.Workspace], p)
			}
		}
	}

	kept := map[string]bool{}
	for _, p := range pages {
		kept[p.ws+":"+p.ctx.ID] = true
	}
	out := MemoryGraph{Nodes: []MemoryNode{}, Edges: []MemoryEdge{}}
	links := map[string]int{}
	for _, e := range wikilinks {
		if kept[e.Source] && kept[e.Target] {
			out.Edges = append(out.Edges, e)
			links[e.Source]++
			links[e.Target]++
		}
	}

	// Layout (FounderOS v1 lib/memory-core.ts's uniform fill): every workspace
	// owns an angular wedge sized by its kept pages, its hub sits mid-wedge,
	// its linked pages fill the wedge out to the disc edge at equal area per
	// note (r ∝ √i, angle on a golden-ratio walk across the wedge, so no ring
	// and no hub clumps), and its unlinked pages ring the rim as the orphan
	// halo. Deterministic: the order is the ranking above.
	total := len(pages)
	start := -math.Pi / 2
	for hi, ws := range hubs {
		members := perHub[ws]
		span := 2 * math.Pi
		if total > 0 && len(hubs) > 1 {
			span = 2 * math.Pi * float64(len(members)) / float64(total)
		}
		mid := start + span/2
		hubR := 0.55
		if len(hubs) == 1 {
			hubR = 0
		}
		hx, hy := clampDisc(hubR*math.Cos(mid), hubR*math.Sin(mid))
		out.Nodes = append(out.Nodes, MemoryNode{ID: "folder:" + ws, Type: "folder", Label: ws, Folder: ws, VX: hx, VY: hy, Cluster: hi, Links: len(members)})
		var linked, orphans []pageRef
		for _, p := range members {
			if links[ws+":"+p.ctx.ID] > 0 {
				linked = append(linked, p)
			} else {
				orphans = append(orphans, p)
			}
		}
		// keep a hair of clear water at each wedge edge so neighbours never touch
		pad := math.Min(0.04, span*0.06)
		inner := span - 2*pad
		place := func(p pageRef, x, y float64) {
			id := ws + ":" + p.ctx.ID
			out.Nodes = append(out.Nodes, MemoryNode{ID: id, Type: "page", Label: p.ctx.Title, Folder: ws, Genre: p.ctx.Genre, Excerpt: p.ctx.Abstract, VX: x, VY: y, Cluster: hi, Links: links[id]})
			out.Edges = append(out.Edges, MemoryEdge{Source: "folder:" + ws, Target: id, Type: "member"})
		}
		for i, p := range linked {
			r := 0.86 * math.Sqrt((float64(i)+0.5)/float64(len(linked)))
			u := math.Mod(float64(i)*0.6180339887498949+0.5, 1)
			a := start + pad + u*inner
			x, y := clampDisc(r*math.Cos(a), r*math.Sin(a))
			place(p, x, y)
		}
		for i, p := range orphans {
			a := start + pad + (float64(i)+0.5)/float64(len(orphans))*inner
			r := 0.92 + 0.05*float64(i%2)
			x, y := clampDisc(r*math.Cos(a), r*math.Sin(a))
			place(p, x, y)
		}
		start += span
	}
	return out
}

// fairShare keeps up to max pages, each workspace's share in proportion to
// its page count (largest remainder), its best-ranked pages first. Without it
// the densest-linked workspace crowds every other one out of the core.
func fairShare(ranked []pageRef, max int) []pageRef {
	if len(ranked) <= max {
		return ranked
	}
	count := map[string]int{}
	var order []string
	for _, p := range ranked {
		if count[p.ws] == 0 {
			order = append(order, p.ws)
		}
		count[p.ws]++
	}
	sort.Strings(order)
	quota := map[string]int{}
	type rem struct {
		ws   string
		frac float64
	}
	var rems []rem
	given := 0
	for _, ws := range order {
		exact := float64(max) * float64(count[ws]) / float64(len(ranked))
		quota[ws] = int(exact)
		given += quota[ws]
		rems = append(rems, rem{ws, exact - float64(quota[ws])})
	}
	sort.SliceStable(rems, func(i, j int) bool { return rems[i].frac > rems[j].frac })
	for i := 0; given < max && i < len(rems); i++ {
		quota[rems[i].ws]++
		given++
	}
	out := make([]pageRef, 0, max)
	for _, p := range ranked {
		if quota[p.ws] > 0 {
			out = append(out, p)
			quota[p.ws]--
		}
	}
	return out
}

func clampDisc(x, y float64) (float64, float64) {
	if d := math.Hypot(x, y); d > 0.98 {
		x, y = x/d*0.98, y/d*0.98
	}
	return round3(x), round3(y)
}

func round3(v float64) float64 { return math.Round(v*1000) / 1000 }
