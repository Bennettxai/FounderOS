package doctor

import (
	"math"
	"sort"
	"time"
)

// Pillar radar (lib/pillar-radar.ts): one axis per department, scored
// 15..100 from real signals: 55% of the roster active, 30% run recency,
// 15% SOP coverage. The three signals are exposed as their own layers.

type Department struct {
	ID    string
	Name  string
	Color string
}

type Agent struct {
	ID           string
	DepartmentID string
	Status       string
}

type SopTask struct {
	DepartmentID string
}

type PillarAxis struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Color     string `json:"color"`
	Score     int    `json:"score"`
	Roster    int    `json:"roster"`
	Freshness int    `json:"freshness"`
	SOP       int    `json:"sop"`
}

// graphDeptOrder is lib/knowledge-graph.ts GRAPH_DEPT_ORDER: Finances rides
// beside Sales. Unknown departments sort after these.
var graphDeptOrder = []string{"dept-sales", "dept-finance", "dept-clients", "dept-marketing-growth", "dept-tech", "dept-comms"}

func deptRank(id string) int {
	for i, d := range graphDeptOrder {
		if d == id {
			return i
		}
	}
	return len(graphDeptOrder) + 1
}

func runRecency(latest time.Time, ok bool, now time.Time) float64 {
	if !ok {
		return 0
	}
	h := now.Sub(latest).Hours()
	switch {
	case h < 0:
		return 0
	case h <= 1:
		return 1
	case h <= 24:
		return 0.7
	case h <= 24*7:
		return 0.4
	default:
		return 0.15
	}
}

// PillarAxes scores each department. latest maps agent id → its newest run's
// finish time.
func PillarAxes(depts []Department, agents []Agent, tasks []SopTask, latest map[string]time.Time, now time.Time) []PillarAxis {
	ordered := append([]Department(nil), depts...)
	sort.SliceStable(ordered, func(i, j int) bool { return deptRank(ordered[i].ID) < deptRank(ordered[j].ID) })
	out := make([]PillarAxis, 0, len(ordered))
	for _, d := range ordered {
		var roster, active int
		var newest time.Time
		haveRun := false
		for _, a := range agents {
			if a.DepartmentID != d.ID {
				continue
			}
			roster++
			if a.Status == "active" {
				active++
			}
			if t, ok := latest[a.ID]; ok && (!haveRun || t.After(newest)) {
				newest, haveRun = t, true
			}
		}
		share := 0.0
		if roster > 0 {
			share = float64(active) / float64(roster)
		}
		rec := runRecency(newest, haveRun, now)
		sops := 0
		for _, t := range tasks {
			if t.DepartmentID == d.ID {
				sops++
			}
		}
		cov := 0.0
		switch {
		case roster > 0:
			cov = math.Min(1, float64(sops)/float64(roster))
		case sops > 0:
			cov = 1
		}
		score := int(math.Round(55*share + 30*rec + 15*cov))
		out = append(out, PillarAxis{
			ID: d.ID, Label: d.Name, Color: d.Color,
			Score:     min(100, max(15, score)),
			Roster:    int(math.Round(share * 100)),
			Freshness: int(math.Round(rec * 100)),
			SOP:       int(math.Round(cov * 100)),
		})
	}
	return out
}
