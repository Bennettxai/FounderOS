package doctor

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

// Meter, Chip and SeriesPoint are the slab kit's shapes (kit/format.ts).
// A nil Frac is an unknown the meter draws hatched, never a zero.
type Meter struct {
	Label   string   `json:"label"`
	Frac    *float64 `json:"frac"`
	Display string   `json:"display"`
	Hue     string   `json:"hue"`
}

type Chip struct {
	Tone string `json:"tone,omitempty"`
	Text string `json:"text"`
}

type SeriesPoint struct {
	Label string `json:"label"`
	Count int64  `json:"count"`
}

type Counts struct {
	OK    int `json:"ok"`
	Warn  int `json:"warn"`
	Fail  int `json:"fail"`
	Total int `json:"total"`
}

type Insight struct {
	Value    *int    `json:"value"`
	Headline string  `json:"headline"`
	Body     string  `json:"body"`
	Frac     float64 `json:"frac"`
}

type StoreTop struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

type StoreShape struct {
	Cols []SeriesPoint `json:"cols"`
	// Tables counts every knowledge table some engine reported (Cols is capped).
	Tables int       `json:"tables"`
	Total  *int64    `json:"total"` // contexts across reachable engines
	Top    *StoreTop `json:"top"`
}

// RunLite is one memory-agent run for the Brain Runs step line.
type RunLite struct {
	FinishedAt time.Time
	OK         bool
}

type VolumeInput struct {
	Engines []EngineReading
	Axes    []PillarAxis
	Runs    []RunLite
	Now     time.Time
	Loc     *time.Location
	Days    int
}

// Volume is every number the slab shows (lib/doctor-volume.ts, on OE).
type Volume struct {
	Headline       *int          `json:"headline"`
	Counts         Counts        `json:"counts"`
	Chips          []Chip        `json:"chips"`
	Caption        string        `json:"caption"`
	Meters         []Meter       `json:"meters"`
	Foot           string        `json:"foot"`
	Series         []SeriesPoint `json:"series"`
	RunsInWindow   int           `json:"runsInWindow"`
	FailedInWindow int           `json:"failedInWindow"`
	Store          StoreShape    `json:"store"`
	Insight        Insight       `json:"insight"`
	EnginesUp      int           `json:"enginesUp"`
	EnginesTotal   int           `json:"enginesTotal"`
	Workspaces     int           `json:"workspaces"`
	Claims         *int64        `json:"claims"`
	Facts          *int64        `json:"facts"`
	Contexts       *int64        `json:"contexts"`
}

func ptr[T any](v T) *T { return &v }

func frac(n, d float64) float64 {
	if d <= 0 {
		return 0
	}
	return math.Max(0, math.Min(1, n/d))
}

func reachable(es []EngineReading) []EngineReading {
	var out []EngineReading
	for _, e := range es {
		if e.Reachable {
			out = append(out, e)
		}
	}
	return out
}

func allChecks(es []EngineReading) []Check {
	var out []Check
	for _, e := range reachable(es) {
		out = append(out, e.Checks...)
	}
	return out
}

// Score grades the engines 0..100: the share of checks passing on the
// engines that answered (a warn is half a pass), times the share of engines
// that answered at all. Nil when no engine answered: there is nothing to grade.
func Score(es []EngineReading) *int {
	up := reachable(es)
	if len(up) == 0 {
		return nil
	}
	checks := allChecks(es)
	pass := 1.0
	if len(checks) > 0 {
		var p float64
		for _, c := range checks {
			switch c.Status {
			case "ok":
				p++
			case "warn":
				p += 0.5
			}
		}
		pass = p / float64(len(checks))
	}
	s := int(math.Round(100 * pass * float64(len(up)) / float64(len(es))))
	return &s
}

// tableSum adds a relational/vector/graph table across reachable engines.
// Nil when no engine reported it.
func tableSum(es []EngineReading, table string) *int64 {
	var total int64
	seen := false
	for _, e := range reachable(es) {
		for _, s := range e.Stores {
			if n, ok := s.Tables[table]; ok {
				total += n
				seen = true
			}
		}
	}
	if !seen {
		return nil
	}
	return &total
}

// knowledgeTables are the store tables that hold memory, with short labels.
// Telemetry (events, model runs, jobs) is left out on purpose.
var knowledgeTables = []struct{ table, label string }{
	{"contexts", "contexts"},
	{"claims", "claims"},
	{"facts", "facts"},
	{"episodes", "episodes"},
	{"memory_objects", "memories"},
	{"chunk_embeddings", "chunks"},
	{"vectors", "vectors"},
	{"edges", "edges"},
}

// prod's matrix carries seven short folder names; table names run longer,
// so six columns keep it inside the card.
const storeCols = 6

func localDay(t time.Time) string { return t.Format("2006-01-02") }

func BuildVolume(x VolumeInput) Volume {
	now := x.Now
	if now.IsZero() {
		now = time.Now()
	}
	loc := x.Loc
	if loc == nil {
		loc = time.Local
	}
	now = now.In(loc)
	days := x.Days
	if days <= 0 {
		days = 14
	}

	up := reachable(x.Engines)
	live := len(up) > 0
	checks := allChecks(x.Engines)
	var c Counts
	for _, ch := range checks {
		switch ch.Status {
		case "ok":
			c.OK++
		case "warn":
			c.Warn++
		}
	}
	c.Total = len(checks)
	c.Fail = c.Total - c.OK - c.Warn

	v := Volume{Counts: c, EnginesUp: len(up), EnginesTotal: len(x.Engines)}
	wsSeen := map[string]bool{}
	for _, e := range up {
		for _, w := range e.Workspaces {
			wsSeen[w] = true
		}
	}
	v.Workspaces = len(wsSeen)
	v.Claims, v.Facts, v.Contexts = tableSum(x.Engines, "claims"), tableSum(x.Engines, "facts"), tableSum(x.Engines, "contexts")

	if !live {
		v.Chips = []Chip{{Tone: "err", Text: "unreachable"}}
	} else {
		if c.OK > 0 {
			v.Chips = append(v.Chips, Chip{Tone: "ok", Text: fmt.Sprintf("%d passing", c.OK)})
		}
		if c.Warn > 0 {
			v.Chips = append(v.Chips, Chip{Tone: "warn", Text: fmt.Sprintf("%d warn", c.Warn)})
		}
		if c.Fail > 0 {
			v.Chips = append(v.Chips, Chip{Tone: "err", Text: fmt.Sprintf("%d failing", c.Fail)})
		}
	}
	if v.Chips == nil {
		v.Chips = []Chip{}
	}

	v.Headline = Score(x.Engines)
	if live {
		ctx := "no pages reported"
		if v.Contexts != nil {
			ctx = fmt.Sprintf("%d engine entries stored", *v.Contexts)
		}
		v.Caption = fmt.Sprintf("%d checks · %s", c.Total, ctx)
	} else if len(x.Engines) == 0 {
		v.Caption = "no engine in the topology has an endpoint"
	} else {
		v.Caption = fmt.Sprintf("0/%d engines answered · memory is unreachable, not empty", len(x.Engines))
	}

	// Meters: unknown stays nil (hatched), never a zero.
	health := Meter{Label: "Engine health (—/100)", Display: "unreachable", Hue: "var(--bn-accent)"}
	if v.Headline != nil {
		health.Label = fmt.Sprintf("Engine health (%d/100)", *v.Headline)
		health.Frac = ptr(frac(float64(*v.Headline), 100))
		health.Display = fmt.Sprintf("%d%%", *v.Headline)
	}
	checksM := Meter{Label: fmt.Sprintf("Checks passing (%d/%d)", c.OK, c.Total), Display: "no checks run", Hue: "var(--bn-warn)"}
	if c.Total > 0 {
		checksM.Frac = ptr(frac(float64(c.OK), float64(c.Total)))
		checksM.Display = fmt.Sprintf("%d ok · %d flagged", c.OK, c.Total-c.OK)
		if c.OK == c.Total {
			checksM.Hue = "var(--bn-ok)"
		}
	}
	// Production's "Storage layers live": every engine plus the logical
	// stores behind them. With no engine answering the stores were never
	// checked (UNKNOWN), and the label says how many, like prod's NOT CHECKED.
	layers := Layers(x.Engines)
	layersLive, layersChecked := 0, 0
	for _, l := range layers {
		if l.Val != notChecked {
			layersChecked++
		}
		if l.State == "connected" {
			layersLive++
		}
	}
	storage := Meter{Label: fmt.Sprintf("Storage layers live (%d/%d)", layersLive, len(layers)), Display: "no layers", Hue: "var(--bn-text-2)"}
	if unchecked := len(layers) - layersChecked; unchecked > 0 {
		storage.Label = fmt.Sprintf("Storage layers live (%d/%d checked · %d unchecked)", layersLive, layersChecked, unchecked)
	}
	if layersChecked > 0 {
		f := frac(float64(layersLive), float64(layersChecked))
		storage.Frac = ptr(f)
		storage.Display = fmt.Sprintf("%d%%", int(math.Round(f*100)))
	}
	pillar := Meter{Label: fmt.Sprintf("Pillar health (%d pillars)", len(x.Axes)), Display: "no pillars", Hue: "var(--bn-text)"}
	if len(x.Axes) > 0 {
		var sum int
		for _, a := range x.Axes {
			sum += a.Score
		}
		mean := int(math.Round(float64(sum) / float64(len(x.Axes))))
		pillar.Frac = ptr(frac(float64(mean), 100))
		pillar.Display = fmt.Sprintf("%d/100", mean)
	}
	v.Meters = []Meter{health, checksM, storage, pillar}

	// Memory-agent runs by local day over the window, oldest first.
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, -(days - 1))
	buckets := map[string]int64{}
	v.Series = make([]SeriesPoint, 0, days)
	keys := make([]string, 0, days)
	for i := 0; i < days; i++ {
		d := start.AddDate(0, 0, i)
		keys = append(keys, localDay(d))
		v.Series = append(v.Series, SeriesPoint{Label: d.Format("Jan 2")})
	}
	for _, r := range x.Runs {
		t := r.FinishedAt.In(loc)
		if t.Before(start) || t.After(now) {
			continue
		}
		buckets[localDay(t)]++
		v.RunsInWindow++
		if !r.OK {
			v.FailedInWindow++
		}
	}
	for i, k := range keys {
		v.Series[i].Count = buckets[k]
	}

	// The store: the knowledge tables, biggest first.
	v.Store = StoreShape{Cols: []SeriesPoint{}, Total: v.Contexts}
	for _, kt := range knowledgeTables {
		if n := tableSum(x.Engines, kt.table); n != nil {
			v.Store.Cols = append(v.Store.Cols, SeriesPoint{Label: kt.label, Count: *n})
		}
	}
	sort.SliceStable(v.Store.Cols, func(i, j int) bool { return v.Store.Cols[i].Count > v.Store.Cols[j].Count })
	v.Store.Tables = len(v.Store.Cols)
	// prod: "18 folders · 6 pillars · 3/4 layers live"; the store's shape is its tables
	v.Foot = fmt.Sprintf("%d tables · %d pillars · %d/%d layers live", v.Store.Tables, len(x.Axes), layersLive, len(layers))
	if len(v.Store.Cols) > storeCols {
		v.Store.Cols = v.Store.Cols[:storeCols]
	}
	if len(v.Store.Cols) > 0 {
		v.Store.Top = &StoreTop{Name: v.Store.Cols[0].Label, Count: v.Store.Cols[0].Count}
	}

	// The one gradient card: the checks asking for attention.
	var flagged []Check
	for _, ch := range checks {
		if ch.Status != "ok" {
			flagged = append(flagged, ch)
		}
	}
	switch {
	case !live:
		v.Insight = Insight{Headline: "No engine answered.", Body: "The Optimal Engine topology did not answer this read; memory is unreachable, not empty."}
	case len(flagged) > 0:
		// prod reads "resolver_health · connection": each flagged check once,
		// then the engines that flagged them
		names, engines := []string{}, []string{}
		seenName, seenEngine := map[string]bool{}, map[string]bool{}
		for _, f := range flagged {
			if !seenName[f.Name] && len(names) < 3 {
				seenName[f.Name] = true
				names = append(names, f.Name)
			}
			if !seenEngine[f.Engine] {
				seenEngine[f.Engine] = true
				engines = append(engines, f.Engine)
			}
		}
		body := strings.Join(names, " · ")
		if len(engines) > 0 {
			body += " · " + strings.Join(engines, ", ")
		}
		v.Insight = Insight{Value: ptr(len(flagged)), Headline: fmt.Sprintf("%d warn · %d failing.", c.Warn, c.Fail), Body: body, Frac: frac(float64(len(flagged)), float64(c.Total))}
	default:
		body := "The engines answered but ran no checks."
		if c.Total > 0 {
			body = fmt.Sprintf("%d checks green, nothing waiting on you.", c.Total)
		}
		v.Insight = Insight{Value: ptr(0), Headline: "Every check passed.", Body: body}
	}
	return v
}

// Layer is a Storage layers row: each engine, then the logical stores.
type Layer struct {
	Name  string `json:"name"`
	Sub   string `json:"sub"`
	Val   string `json:"val"`
	State string `json:"state"` // connected | available | error
}

var layerStores = []struct{ id, sub string }{
	{"relational", "contexts, claims, facts, episodes"},
	{"full_text", "FTS5 exact-text retrieval"},
	{"vector", "chunk embeddings, cosine search"},
	{"graph", "edges and relationship traversal"},
}

func commas(n int64) string {
	s := fmt.Sprint(n)
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	var b strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}

// notChecked marks a store layer no engine could vouch for (none answered).
const notChecked = "UNKNOWN"

func Layers(es []EngineReading) []Layer {
	out := []Layer{}
	for _, e := range es {
		l := Layer{Name: e.Name + " engine", Sub: e.URL}
		switch {
		case !e.Reachable:
			l.Val, l.State, l.Sub = "UNREACHABLE", "error", e.URL+" · "+e.Error
		default:
			l.Sub = fmt.Sprintf("%s · %d workspaces · health %s", e.URL, len(e.Workspaces), e.Health)
			l.Val, l.State = "LIVE", "connected"
			if e.Health != "up" {
				l.Val, l.State = strings.ToUpper(e.Health), "available"
			}
		}
		out = append(out, l)
	}
	up := reachable(es)
	for _, ls := range layerStores {
		var rows int64
		have := 0
		for _, e := range up {
			for _, s := range e.Stores {
				if s.ID == ls.id && s.Status == "available" {
					rows += s.RowCount
					have++
				}
			}
		}
		l := Layer{Name: ls.id, Sub: fmt.Sprintf("%s · on %d of %d engines", ls.sub, have, len(es))}
		switch {
		case len(up) == 0:
			l.Val, l.State = notChecked, "available"
		case have == 0:
			l.Val, l.State = "MISSING", "error"
		default:
			l.Val, l.State = commas(rows)+" rows", "connected"
		}
		out = append(out, l)
	}
	return out
}
