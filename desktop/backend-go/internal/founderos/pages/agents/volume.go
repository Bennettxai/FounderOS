// Package agentspage is the /agents page's pure logic, ported from FounderOS v1
// lib/board-live.ts (the live board payload and its stats), lib/agents-volume.ts
// (the slab hero), lib/agent-failover.ts (the model failover plan),
// lib/agents/activity.ts and the deliverables queue (lib/board-deliverables,
// deliverable-brief, deliverable-preview, deliverable-revision,
// board-approvals, deliverable-decisions).
package agentspage

import (
	"sort"
	"strconv"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/paperclip"
	"github.com/rhl/businessos-backend/internal/founderos/pages/osdata"
	"github.com/rhl/businessos-backend/internal/founderos/pages/shared"
)

// Board is BoardLivePayload: one snapshot of the Paperclip board. When the
// board is unreachable Connected is false, the lists are empty and Error
// says why (never fake data).
type Board struct {
	Connected bool              `json:"connected"`
	Error     string            `json:"error,omitempty"`
	Agents    []paperclip.Agent `json:"agents"`
	Issues    []paperclip.Issue `json:"issues"`
	Runs      []paperclip.Run   `json:"runs"`
	CheckedAt string            `json:"checkedAt"`
	Decisions []osdata.Decision `json:"decisions"`
}

// RunsFetched / IssuesFetched are the depths /agents and /board/live read.
const (
	RunsFetched   = 120
	IssuesFetched = 250
)

// RunOK are the statuses that mean a run ended well.
var RunOK = map[string]bool{"succeeded": true, "completed": true, "success": true, "done": true}

var pending = map[string]bool{"running": true, "queued": true, "pending": true}

type BoardStats struct {
	Seats         int `json:"seats"`
	Running       int `json:"running"`
	OpenTasks     int `json:"openTasks"`
	Runs24h       int `json:"runs24h"`
	Heartbeats24h int `json:"heartbeats24h"`
}

func within(iso *string, since time.Time) bool {
	if iso == nil {
		return false
	}
	t, ok := shared.ParseISO(*iso)
	return ok && t.After(since)
}

// Stats is boardStats: every figure from live board data only.
func Stats(b Board, now time.Time) BoardStats {
	dayAgo := now.Add(-24 * time.Hour)
	s := BoardStats{Seats: len(b.Agents)}
	for _, a := range b.Agents {
		if a.Status == "running" {
			s.Running++
		}
		if within(a.LastHeartbeatAt, dayAgo) {
			s.Heartbeats24h++
		}
	}
	for _, i := range b.Issues {
		if i.Status != "done" && i.Status != "cancelled" {
			s.OpenTasks++
		}
	}
	for _, r := range b.Runs {
		if within(r.StartedAt, dayAgo) {
			s.Runs24h++
		}
	}
	return s
}

type Counts struct {
	Running int `json:"running"`
	Idle    int `json:"idle"`
	Paused  int `json:"paused"`
	Error   int `json:"error"`
}

// Volume is AgentsVolume: every number the /agents slab hero shows.
type Volume struct {
	Headline       int                  `json:"headline"`
	Counts         Counts               `json:"counts"`
	Chips          []shared.Chip        `json:"chips"`
	Caption        string               `json:"caption"`
	Meters         []shared.Meter       `json:"meters"`
	Foot           string               `json:"foot"`
	OpenTasks      int                  `json:"openTasks"`
	Series         []shared.SeriesPoint `json:"series"`
	Window         string               `json:"window"`
	RunsInWindow   int                  `json:"runsInWindow"`
	FailedInWindow int                  `json:"failedInWindow"`
	BySeat         []shared.SeriesPoint `json:"bySeat"`
	Lanes          []shared.SeriesPoint `json:"lanes"`
	Insight        shared.Insight       `json:"insight"`
}

// lanes: the open stages in board order; review is spelled two ways.
var lanes = []struct {
	label    string
	statuses []string
}{
	{"doing", []string{"in_progress"}},
	{"review", []string{"in_review", "review"}},
	{"todo", []string{"todo"}},
	{"blocked", []string{"blocked"}},
	{"backlog", []string{"backlog"}},
}

// AgentsVolume is one pure pass over the board payload.
func AgentsVolume(b Board, now time.Time, days int) Volume {
	stats := Stats(b, now)
	var counts Counts
	for _, a := range b.Agents {
		switch a.Status {
		case "running":
			counts.Running++
		case "idle":
			counts.Idle++
		case "paused":
			counts.Paused++
		case "error":
			counts.Error++
		}
	}

	stamps := make([]string, 0, len(b.Runs))
	for _, r := range b.Runs {
		if r.StartedAt != nil {
			stamps = append(stamps, *r.StartedAt)
		}
	}
	series := shared.DailySeries(stamps, days, now)
	start := shared.WindowStart(now, days)

	type stamped struct {
		r paperclip.Run
		t time.Time
	}
	var inWindow []stamped
	oldest := time.Time{}
	for _, r := range b.Runs {
		if r.StartedAt == nil {
			continue
		}
		t, ok := shared.ParseISO(*r.StartedAt)
		if !ok {
			continue
		}
		if oldest.IsZero() || t.Before(oldest) {
			oldest = t
		}
		if !t.Before(start) && !t.After(now) {
			inWindow = append(inWindow, stamped{r, t})
		}
	}
	window := itoa(days) + "d"
	if len(b.Runs) >= RunsFetched && !oldest.IsZero() && oldest.After(start) {
		window = "since " + oldest.In(time.Local).Format("Jan 2")
	}
	failed := func(s string) bool { return !RunOK[s] && !pending[s] }
	finished, okRuns, failedInWindow, failed24h := 0, 0, 0, 0
	dayAgo := now.Add(-24 * time.Hour)
	for _, x := range inWindow {
		if !pending[x.r.Status] {
			finished++
			if RunOK[x.r.Status] {
				okRuns++
			}
			if failed(x.r.Status) {
				failedInWindow++
			}
		}
		if failed(x.r.Status) && x.t.After(dayAgo) {
			failed24h++
		}
	}

	// runs by seat: busiest first, first word of the seat's name
	nameByID := map[string]string{}
	for _, a := range b.Agents {
		nameByID[a.ID] = a.Name
	}
	perSeat := map[string]int{}
	var order []string
	for _, x := range inWindow {
		name := ""
		switch {
		case x.r.AgentName != nil:
			name = *x.r.AgentName
		case nameByID[x.r.AgentID] != "":
			name = nameByID[x.r.AgentID]
		default:
			name = x.r.AgentID
			if len(name) > 8 {
				name = name[:8]
			}
		}
		if _, seen := perSeat[name]; !seen {
			order = append(order, name)
		}
		perSeat[name]++
	}
	sort.SliceStable(order, func(i, j int) bool { return perSeat[order[i]] > perSeat[order[j]] })
	if len(order) > 6 {
		order = order[:6]
	}
	seatLabels := shared.ShortLabels(order, 0)
	bySeat := make([]shared.SeriesPoint, len(order))
	for i, name := range order {
		bySeat[i] = shared.SeriesPoint{Label: seatLabels[i], Count: perSeat[name]}
	}

	live, done := 0, 0
	laneCounts := make([]shared.SeriesPoint, len(lanes))
	for i, l := range lanes {
		laneCounts[i].Label = l.label
	}
	for _, is := range b.Issues {
		if is.Status == "cancelled" {
			continue
		}
		live++
		if is.Status == "done" {
			done++
		}
		for i, l := range lanes {
			for _, s := range l.statuses {
				if is.Status == s {
					laneCounts[i].Count++
				}
			}
		}
	}
	models := map[string]bool{}
	for _, a := range b.Agents {
		if a.Model != nil && *a.Model != "" {
			models[*a.Model] = true
		}
	}

	if !b.Connected {
		return Volume{
			Counts:  counts,
			Chips:   []shared.Chip{{Tone: "err", Text: "board unreachable"}},
			Caption: "Paperclip is not answering, so there is nothing to count",
			Meters:  []shared.Meter{}, Foot: "no live board", Series: series, Window: window,
			BySeat: []shared.SeriesPoint{}, Lanes: laneCounts,
			Insight: shared.Insight{Headline: "Board unreachable.", Body: "No numbers until Paperclip answers on the tailnet."},
		}
	}

	chips := []shared.Chip{}
	if counts.Running > 0 {
		chips = append(chips, shared.Chip{Tone: "ok", Text: itoa(counts.Running) + " running"})
	}
	if counts.Idle > 0 {
		chips = append(chips, shared.Chip{Text: itoa(counts.Idle) + " idle"})
	}
	if counts.Paused > 0 {
		chips = append(chips, shared.Chip{Tone: "warn", Text: itoa(counts.Paused) + " paused"})
	}
	if counts.Error > 0 {
		chips = append(chips, shared.Chip{Tone: "err", Text: itoa(counts.Error) + " in error"})
	}

	seats := stats.Seats
	runF := shared.Frac(counts.Running, seats)
	beatF := shared.Frac(stats.Heartbeats24h, seats)
	okF := shared.Frac(okRuns, finished)
	doneF := shared.Frac(done, live)
	pick := func(cond bool, a, b string) string {
		if cond {
			return a
		}
		return b
	}
	meters := []shared.Meter{
		{Label: "Seats running (" + itoa(counts.Running) + "/" + itoa(seats) + ")", Frac: shared.F(runF), Display: pick(counts.Running > 0, shared.Pct(runF), "none running"), Hue: "var(--bn-ok)"},
		{Label: "Heartbeat in 24h (" + itoa(stats.Heartbeats24h) + "/" + itoa(seats) + ")", Frac: shared.F(beatF), Display: pick(seats > 0, shared.Pct(beatF), "no seats"), Hue: "var(--bn-accent)"},
		{Label: "Runs OK · " + window + " (" + itoa(okRuns) + "/" + itoa(finished) + ")", Frac: shared.F(okF), Display: pick(finished > 0, itoa(okRuns)+" ok · "+itoa(failedInWindow)+" failed", "no finished runs"), Hue: "var(--bn-text-2)"},
		{Label: "Tasks done (" + itoa(done) + "/" + itoa(live) + ")", Frac: shared.F(doneF), Display: pick(live > 0, shared.Pct(doneF), "no tasks"), Hue: "var(--bn-text)"},
	}

	waiting := laneCounts[1].Count + laneCounts[3].Count
	headline := "Nothing on the board is waiting on you."
	if waiting > 0 {
		headline = shared.Plural(waiting, "task") + " waiting on review or unblocking."
	}
	return Volume{
		Headline: seats, Counts: counts, Chips: chips,
		Caption:   "seats on the Paperclip board · " + shared.Plural(len(models), "model"),
		Meters:    meters,
		Foot:      shared.Plural(stats.OpenTasks, "open task") + " · " + shared.Plural(stats.Runs24h, "run") + " in 24h",
		OpenTasks: stats.OpenTasks, Series: series, Window: window,
		RunsInWindow: len(inWindow), FailedInWindow: failedInWindow,
		BySeat: bySeat, Lanes: laneCounts,
		Insight: shared.Insight{
			Value: waiting, Headline: headline,
			Body: shared.Plural(counts.Error, "seat") + " in error · " + shared.Plural(failed24h, "failed run") + " in 24h",
			Frac: shared.Frac(waiting, stats.OpenTasks),
		},
	}
}

func itoa(n int) string { return strconv.Itoa(n) }
