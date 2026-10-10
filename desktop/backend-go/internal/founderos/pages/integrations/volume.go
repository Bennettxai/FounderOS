package integrations

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
)

// Chip, Meter and Point match the FounderOS kit's StatChip, Meter and SeriesPoint.
type Chip struct {
	Tone string `json:"tone,omitempty"`
	Text string `json:"text"`
}

type Meter struct {
	Label   string   `json:"label"`
	Frac    *float64 `json:"frac"`
	Display string   `json:"display"`
	Hue     string   `json:"hue"`
}

type Point struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

type Counts struct {
	Connected     int `json:"connected"`
	NotConfigured int `json:"notConfigured"`
	Error         int `json:"error"`
	Total         int `json:"total"`
}

type TopCategory struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type Insight struct {
	Value    int     `json:"value"`
	Headline string  `json:"headline"`
	Body     string  `json:"body"`
	Frac     float64 `json:"frac"`
}

// VolumeModel is every number on the slab (lib/integrations-volume.ts).
type VolumeModel struct {
	Headline    int          `json:"headline"`
	Counts      Counts       `json:"counts"`
	Chips       []Chip       `json:"chips"`
	Caption     string       `json:"caption"`
	Meters      []Meter      `json:"meters"`
	Foot        string       `json:"foot"`
	ByCategory  []Point      `json:"byCategory"`
	TopCategory *TopCategory `json:"topCategory"`
	Health      []Point      `json:"health"`
	Insight     Insight      `json:"insight"`
}

// Dot-matrix column labels: short enough that five sit beside a 30px stat.
var short = map[string]string{
	"Productivity": "Prod", "Communication": "Comm", "CRM & Sales": "CRM", "Developer": "Dev",
	"Scheduling": "Sched", "Finance": "Fin", "Marketing": "Mkt", "Storage": "Store",
	"Knowledge": "Know", "AI & Automation": "AI", "Creative": "Create",
}

func frac(n, d int) float64 {
	if d <= 0 {
		return 0
	}
	return math.Max(0, math.Min(1, float64(n)/float64(d)))
}

func pct(f float64) string { return fmt.Sprintf("%d%%", int(math.Round(f*100))) }

func plural(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}

func meter(label string, n, d int, hue string) Meter {
	f := frac(n, d)
	return Meter{Label: fmt.Sprintf("%s (%d/%d)", label, n, d), Frac: &f, Display: pct(f), Hue: hue}
}

// Volume computes the Connection Volume card, the category and health dot
// matrices and the one insight from the live checks and the merged catalog.
func Volume(statuses []connectors.Status, catalog []Entry) VolumeModel {
	var c Counts
	var erroring []string
	c.Total = len(statuses)
	for _, s := range statuses {
		switch s.State {
		case connectors.StateConnected:
			c.Connected++
		case connectors.StateError:
			c.Error++
			erroring = append(erroring, s.Name)
		}
	}
	c.NotConfigured = c.Total - c.Connected - c.Error

	chips := []Chip{}
	if c.Connected > 0 {
		chips = append(chips, Chip{Tone: "ok", Text: fmt.Sprintf("%d live", c.Connected)})
	}
	if c.NotConfigured > 0 {
		chips = append(chips, Chip{Text: fmt.Sprintf("%d not configured", c.NotConfigured)})
	}
	if c.Error > 0 {
		chips = append(chips, Chip{Tone: "err", Text: fmt.Sprintf("%d erroring", c.Error)})
	}

	tools := len(catalog)
	connectedTools, keyed, saved, popular, popularLive, savedNotLive := 0, 0, 0, 0, 0, 0
	for _, e := range catalog {
		if e.Connected {
			connectedTools++
		}
		if len(ConnectKeysFor(e.Integration)) > 0 {
			keyed++
			if e.KeySaved {
				saved++
			}
		}
		if e.Popular {
			popular++
			if e.Connected {
				popularLive++
			}
		}
		if e.KeySaved && !e.Connected {
			savedNotLive++
		}
	}

	meters := []Meter{}
	if c.Total > 0 {
		meters = append(meters, meter("Connectors live", c.Connected, c.Total, "var(--bn-ok)"))
	}
	if tools > 0 {
		meters = append(meters, meter("Catalog tools connected", connectedTools, tools, "var(--bn-accent)"))
	}
	if keyed > 0 {
		meters = append(meters, meter("Keys saved", saved, keyed, "var(--bn-text-2)"))
	}
	if popular > 0 {
		meters = append(meters, meter("Popular connected", popularLive, popular, "var(--bn-text-3)"))
	}

	// Connected tools per category, busiest first; ties keep catalog order.
	type cat struct {
		name         string
		order, count int
	}
	var cats []cat
	for _, name := range Categories {
		present, n := false, 0
		for _, e := range catalog {
			if e.Category == name {
				present = true
				if e.Connected {
					n++
				}
			}
		}
		if present {
			cats = append(cats, cat{name, len(cats), n})
		}
	}
	ranked := append([]cat(nil), cats...)
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].count != ranked[j].count {
			return ranked[i].count > ranked[j].count
		}
		return ranked[i].order < ranked[j].order
	})
	byCategory := []Point{}
	for i, r := range ranked {
		if i == 5 {
			break
		}
		label := short[r.name]
		if label == "" {
			label = r.name
		}
		byCategory = append(byCategory, Point{label, r.count})
	}
	var top *TopCategory
	if len(ranked) > 0 && ranked[0].count > 0 {
		top = &TopCategory{Name: ranked[0].name, Count: ranked[0].count}
	}

	in := Insight{Value: c.Error, Frac: frac(c.Error, c.Total)}
	switch {
	case c.Error > 0:
		in.Headline = plural(c.Error, "connector") + " erroring."
		if len(erroring) > 3 {
			erroring = erroring[:3]
		}
		in.Body = strings.Join(erroring, " · ")
	default:
		in.Headline = "No connector is erroring."
		switch {
		case savedNotLive > 0:
			in.Body = plural(savedNotLive, "saved key") + " not live yet."
		case c.Total > 0:
			in.Body = "Every configured connector answers."
		default:
			in.Body = "No connector checks ran."
		}
	}

	catWord := "categories"
	if len(cats) == 1 {
		catWord = "category"
	}
	foot := fmt.Sprintf("%s · %d %s", plural(tools, "tool"), len(cats), catWord)
	if savedNotLive > 0 {
		foot += " · " + plural(savedNotLive, "saved key") + " not live"
	}

	return VolumeModel{
		Headline:    c.Connected,
		Counts:      c,
		Chips:       chips,
		Caption:     "of " + plural(c.Total, "connector check") + " · live status, never a stored key alone",
		Meters:      meters,
		Foot:        foot,
		ByCategory:  byCategory,
		TopCategory: top,
		Health:      []Point{{"live", c.Connected}, {"unset", c.NotConfigured}, {"error", c.Error}},
		Insight:     in,
	}
}
