// Package usage is the /usage token-burn board's server half: FounderOS v1
// lib/usage.ts combineSeats / mergeBreakdowns / nameBoardLabels /
// combineOllama and app/api/usage/route.ts. One Claude plan, one ChatGPT
// (Codex) plan and one Ollama plan, each summed across every machine that
// reported it.
//
// On the bridge the backend never parses device files: every seat is a
// machine's push (devicepush collectors, or rows stored by /api/usage/push).
// A plan no machine reported is nil (unknown), never a zero plan, and an
// Ollama lane nobody reported is nil rather than "server down".
package usage

import (
	"sort"
	"strings"
	"time"

	dp "github.com/rhl/businessos-backend/internal/founderos/connectors/devicepush"
)

// StalePush matches FounderOS v1 STALE_PUSH_MS.
const StalePush = 24 * time.Hour

// Sources are the burn lanes, in board order.
var Sources = []string{"board", "sessions", "terminal", "automation"}

// Seat is one machine's reading of a plan. ReceiverStale is set when the
// receiver judged the push stale by its own clock.
type Seat struct {
	dp.SeatUsage
	ReceiverStale bool `json:"-"`
}

// Ollama is one machine's Ollama lane.
type Ollama struct {
	dp.OllamaSnapshot
	ReceiverStale bool `json:"-"`
}

// Machine is PlanMachine.
type Machine struct {
	ID           string  `json:"id"`
	Label        string  `json:"label"`
	Source       string  `json:"source"`
	CapturedAt   string  `json:"capturedAt"`
	LastActivity *string `json:"lastActivity"`
	Stale        bool    `json:"stale"`
}

// Plan is PlanUsage.
type Plan struct {
	Plan         *string           `json:"plan"`
	Official     *dp.Official      `json:"official"`
	Days         []dp.DayBucket    `json:"days"`
	ByModel      map[string]dp.Tot `json:"byModel"`
	Breakdown    *dp.Breakdown     `json:"breakdown,omitempty"`
	LastActivity *string           `json:"lastActivity"`
	Machines     []Machine         `json:"machines"`
	PlanConflict []string          `json:"planConflict,omitempty"`
	Note         string            `json:"note,omitempty"`
}

// OllamaModel is a model with the machine that lists it.
type OllamaModel struct {
	Name  string `json:"name"`
	Cloud bool   `json:"cloud"`
	Host  string `json:"host"`
}

// OllamaBoard is combineOllama's result.
type OllamaBoard struct {
	State    string             `json:"state"`
	Plan     *string            `json:"plan"`
	Models   []OllamaModel      `json:"models"`
	Requests *dp.RequestWindows `json:"requests"`
	Note     string             `json:"note"`
	Machines []Machine          `json:"machines"`
}

// Board is the whole /usage reading.
type Board struct {
	GeneratedAt string            `json:"generatedAt"`
	Claude      *Plan             `json:"claude"`
	Codex       *Plan             `json:"codex"`
	Ollama      *OllamaBoard      `json:"ollama"`
	Errors      map[string]string `json:"errors,omitempty"`
}

func addTot(a *dp.Tot, b dp.Tot) {
	a.In += b.In
	a.Out += b.Out
	a.CacheWrite += b.CacheWrite
	a.CacheRead += b.CacheRead
}

// Burn is input + output + cache writes; cache reads are re-read context.
func Burn(t dp.Tot) float64 { return t.In + t.Out + t.CacheWrite }

func lane(l *dp.LaneTots, source string) *dp.Tot {
	switch source {
	case "board":
		return &l.Board
	case "sessions":
		return &l.Sessions
	case "terminal":
		return &l.Terminal
	case "automation":
		return &l.Automation
	}
	return nil
}

func windows(w *dp.Windows) []*dp.LaneTots {
	return []*dp.LaneTots{&w.Hour, &w.Session, &w.Day, &w.Week}
}

func stale(source, capturedAt string, receiverStale bool, now time.Time) bool {
	if receiverStale {
		return true
	}
	if source != "push" {
		return false
	}
	t, err := time.Parse(time.RFC3339, capturedAt)
	if err != nil {
		return false
	}
	return now.Sub(t) > StalePush
}

// MergeBreakdowns sums lanes window by window and top burners by lane+label.
// No breakdowns at all is nil: no lane data, not a zero reading.
func MergeBreakdowns(list []*dp.Breakdown) *dp.Breakdown {
	var present []*dp.Breakdown
	for _, b := range list {
		if b != nil {
			present = append(present, b)
		}
	}
	if len(present) == 0 {
		return nil
	}
	out := &dp.Breakdown{Top: []dp.TopBurner{}}
	var order []string
	top := map[string]*dp.TopBurner{}
	for _, b := range present {
		src, dst := windows(&b.Windows), windows(&out.Windows)
		for i := range dst {
			for _, s := range Sources {
				addTot(lane(dst[i], s), *lane(src[i], s))
			}
		}
		for _, t := range b.Top {
			key := t.Source + "|" + t.Label
			if hit, ok := top[key]; ok {
				hit.Burn += t.Burn
				continue
			}
			c := t
			top[key] = &c
			order = append(order, key)
		}
	}
	for _, k := range order {
		out.Top = append(out.Top, *top[k])
	}
	sort.SliceStable(out.Top, func(i, j int) bool { return out.Top[i].Burn > out.Top[j].Burn })
	if len(out.Top) > 16 {
		out.Top = out.Top[:16]
	}
	return out
}

// NameBoardLabels swaps board seat ids for agent names when the board answered.
func NameBoardLabels(b *dp.Breakdown, names map[string]string) *dp.Breakdown {
	if b == nil || len(names) == 0 {
		return b
	}
	ids := make([]string, 0, len(names))
	for id := range names {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	nameOf := func(label string) (string, bool) {
		if n, ok := names[label]; ok {
			return n, true
		}
		for _, id := range ids {
			if strings.HasPrefix(id, label) || strings.HasPrefix(label, id) {
				return names[id], true
			}
		}
		return "", false
	}
	out := *b
	out.Top = make([]dp.TopBurner, len(b.Top))
	for i, t := range b.Top {
		if t.Source == "board" {
			if n, ok := nameOf(t.Label); ok {
				t.Label = n
			}
		}
		out.Top[i] = t
	}
	return &out
}

// windowDays is the seat day window (collect.UsageWindowDays).
const windowDays = 7

// dayWindow is the n local days ending today, oldest first: the same keys the
// collector buckets by (the Mac's local calendar day). It is anchored to now,
// never to whichever seat happens to sort first.
func dayWindow(now time.Time, n int) []dp.DayBucket {
	out := make([]dp.DayBucket, 0, n)
	for i := n - 1; i >= 0; i-- {
		out = append(out, dp.DayBucket{Day: now.Add(-time.Duration(i) * 24 * time.Hour).In(time.Local).Format("2006-01-02")})
	}
	return out
}

// CombineSeats sums every machine's reading of the SAME plan. The gauge comes
// from the freshest reading that has one; every machine stays listed.
func CombineSeats(seats []Seat, now time.Time) *Plan {
	if len(seats) == 0 {
		return nil
	}
	p := &Plan{ByModel: map[string]dp.Tot{}, Machines: []Machine{}, Days: dayWindow(now, windowDays)}
	idx := map[string]int{}
	for i, d := range p.Days {
		idx[d.Day] = i
	}
	var breakdowns []*dp.Breakdown
	var withGauge []Seat
	var plans []string
	planWho := map[string]string{}
	for _, s := range seats {
		for _, d := range s.Days {
			if i, ok := idx[d.Day]; ok {
				addTot(&p.Days[i].Tot, d.Tot)
			}
		}
		for m, t := range s.ByModel {
			cur := p.ByModel[m]
			addTot(&cur, t)
			p.ByModel[m] = cur
		}
		b := s.Breakdown
		if b != nil && stale(s.Source, s.CapturedAt, s.ReceiverStale, now) {
			// a stale seat's last hour / 5h session describe then, not now
			cp := *b
			cp.Windows.Hour, cp.Windows.Session = dp.LaneTots{}, dp.LaneTots{}
			b = &cp
		}
		breakdowns = append(breakdowns, b)
		if s.Official != nil {
			withGauge = append(withGauge, s)
		}
		if s.Plan != nil && *s.Plan != "" {
			if _, seen := planWho[*s.Plan]; !seen {
				planWho[*s.Plan] = s.Label
				plans = append(plans, *s.Plan)
			}
		}
		if s.LastActivity != nil && (p.LastActivity == nil || *s.LastActivity > *p.LastActivity) {
			v := *s.LastActivity
			p.LastActivity = &v
		}
		if p.Note == "" && s.Note != "" {
			p.Note = s.Note
		}
		p.Machines = append(p.Machines, Machine{
			ID: s.ID, Label: s.Label, Source: s.Source, CapturedAt: s.CapturedAt, LastActivity: s.LastActivity,
			Stale: stale(s.Source, s.CapturedAt, s.ReceiverStale, now),
		})
	}
	sort.SliceStable(withGauge, func(i, j int) bool { return withGauge[i].CapturedAt > withGauge[j].CapturedAt })
	if len(withGauge) > 0 {
		p.Official = withGauge[0].Official
	}
	// The note belongs with the gauge: a seat that read the official % speaks
	// first (FounderOS v1 8abea0c); otherwise the first seat with a note.
	for _, s := range withGauge {
		if s.Note != "" {
			p.Note = s.Note
			break
		}
	}
	if len(plans) > 0 {
		v := plans[0]
		p.Plan = &v
	}
	if len(plans) > 1 {
		for _, name := range plans {
			p.PlanConflict = append(p.PlanConflict, name+" ("+planWho[name]+")")
		}
	}
	p.Breakdown = MergeBreakdowns(breakdowns)
	return p
}

// CombineOllama folds every machine's Ollama lane: plan from whoever is signed
// in, requests summed over machines that have a server log, models per host.
func CombineOllama(all []Ollama, now time.Time) *OllamaBoard {
	if len(all) == 0 {
		return nil
	}
	b := &OllamaBoard{State: "down", Models: []OllamaModel{}, Machines: []Machine{}, Note: all[0].Lane.Note}
	for _, o := range all {
		if o.Lane.State == "up" {
			b.State = "up"
		}
		if b.Plan == nil && o.Lane.Plan != nil && *o.Lane.Plan != "" {
			v := *o.Lane.Plan
			b.Plan = &v
		}
		for _, m := range o.Lane.Models {
			b.Models = append(b.Models, OllamaModel{Name: m.Name, Cloud: m.Cloud, Host: o.Label})
		}
		if r := o.Lane.Requests; r != nil {
			if b.Requests == nil {
				b.Requests = &dp.RequestWindows{}
			}
			add := func(a *dp.RequestCounts, c dp.RequestCounts) { a.Chat += c.Chat; a.Embed += c.Embed }
			add(&b.Requests.Hour, r.Hour)
			add(&b.Requests.Session, r.Session)
			add(&b.Requests.Day, r.Day)
			add(&b.Requests.Week, r.Week)
		}
		b.Machines = append(b.Machines, Machine{
			ID: o.ID, Label: o.Label, Source: "push", CapturedAt: o.CapturedAt,
			Stale: stale("push", o.CapturedAt, o.ReceiverStale, now),
		})
	}
	return b
}

// DedupeSeats keeps the newest reading per seat id (a machine can appear both
// as a live device push and as a stored /api/usage/push row), in first-seen
// id order.
func DedupeSeats(seats []Seat) []Seat {
	best := map[string]int{}
	var out []Seat
	for _, s := range seats {
		if i, ok := best[s.ID]; ok {
			if s.CapturedAt > out[i].CapturedAt {
				out[i] = s
			}
			continue
		}
		best[s.ID] = len(out)
		out = append(out, s)
	}
	return out
}

// DedupeOllama keeps the newest lane per id.
func DedupeOllama(all []Ollama) []Ollama {
	best := map[string]int{}
	var out []Ollama
	for _, o := range all {
		if i, ok := best[o.ID]; ok {
			if o.CapturedAt > out[i].CapturedAt {
				out[i] = o
			}
			continue
		}
		best[o.ID] = len(out)
		out = append(out, o)
	}
	return out
}

// Build assembles the board: seats split by kind into one plan each, board
// seat ids named when names are known.
func Build(seats []Seat, lanes []Ollama, names map[string]string, now time.Time) Board {
	var claude, codex []Seat
	for _, s := range DedupeSeats(seats) {
		switch s.Kind {
		case "claude":
			claude = append(claude, s)
		case "codex":
			codex = append(codex, s)
		}
	}
	named := func(p *Plan) *Plan {
		if p != nil && p.Breakdown != nil {
			p.Breakdown = NameBoardLabels(p.Breakdown, names)
		}
		return p
	}
	return Board{
		GeneratedAt: now.UTC().Format(time.RFC3339),
		Claude:      named(CombineSeats(claude, now)),
		Codex:       named(CombineSeats(codex, now)),
		Ollama:      CombineOllama(DedupeOllama(lanes), now),
	}
}
