// Package tasks is the /tasks page's pure logic: FounderOS v1
// lib/tasks-volume.ts, one pass over the local kanban (founderos_agent_tasks),
// the live Paperclip board issues, the scheduled jobs and the real cron runs.
package tasks

import (
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/pages/shared"
)

type Task struct {
	AgentID   string
	Status    string
	UpdatedAt string
}

type Issue struct {
	Status    string
	UpdatedAt *string
}

// Job is the slice of shared.JobRow the volume reads.
type Job struct {
	Description  string
	Enabled      bool
	UnknownAgent bool
	Overdue      bool
	LastOK       *bool
	Runs         int
	OK           int
}

type Run struct {
	StartedAt string
	OK        bool
}

type Input struct {
	Tasks  []Task
	Issues []Issue
	// BoardConnected is false when Paperclip did not answer. As in v1, the
	// board then reads empty ("board empty or offline"); only BoardOnline says which.
	BoardConnected bool
	Jobs           []Job
	Runs           []Run
	AgentNames     map[string]string
	Now            time.Time
	Days           int
}

type Counts struct {
	Open   int `json:"open"`
	Doing  int `json:"doing"`
	Review int `json:"review"`
	Done   int `json:"done"`
}

type BoardCounts struct {
	Total      int `json:"total"`
	InProgress int `json:"inProgress"`
	Blocked    int `json:"blocked"`
	Done       int `json:"done"`
}

type Owner struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type Cron struct {
	RunsInWindow   int                  `json:"runsInWindow"`
	FailedInWindow int                  `json:"failedInWindow"`
	Rhythm         []shared.SeriesPoint `json:"rhythm"`
}

type Result struct {
	Headline        int                  `json:"headline"`
	Counts          Counts               `json:"counts"`
	Board           BoardCounts          `json:"board"`
	BoardOnline     bool                 `json:"boardOnline"`
	Chips           []shared.Chip        `json:"chips"`
	Caption         string               `json:"caption"`
	Meters          []shared.Meter       `json:"meters"`
	Foot            string               `json:"foot"`
	Series          []shared.SeriesPoint `json:"series"`
	TouchesInWindow int                  `json:"touchesInWindow"`
	Cron            Cron                 `json:"cron"`
	Owners          []shared.SeriesPoint `json:"owners"`
	BusiestOwner    *Owner               `json:"busiestOwner"`
	Insight         shared.Insight       `json:"insight"`
}

const ownerCols = 6

var weekdays = []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}

func healthy(j Job) bool {
	return j.Enabled && !j.UnknownAgent && !j.Overdue && (j.LastOK == nil || *j.LastOK)
}

func itoa(n int) string { return strconv.Itoa(n) }

func pick(c bool, a, b string) string {
	if c {
		return a
	}
	return b
}

// Volume is tasksVolume.
func Volume(x Input) Result {
	now := x.Now
	if now.IsZero() {
		now = time.Now()
	}
	days := x.Days
	if days <= 0 {
		days = 14
	}

	var c Counts
	for _, t := range x.Tasks {
		switch t.Status {
		case "open":
			c.Open++
		case "doing":
			c.Doing++
		case "review":
			c.Review++
		case "done":
			c.Done++
		}
	}
	inFlight := c.Open + c.Doing + c.Review

	b := BoardCounts{Total: len(x.Issues)}
	for _, i := range x.Issues {
		switch i.Status {
		case "in_progress":
			b.InProgress++
		case "blocked":
			b.Blocked++
		case "done":
			b.Done++
		}
	}

	chips := []shared.Chip{}
	if c.Open > 0 {
		chips = append(chips, shared.Chip{Text: itoa(c.Open) + " to do"})
	}
	if c.Doing > 0 {
		chips = append(chips, shared.Chip{Tone: "warn", Text: itoa(c.Doing) + " in progress"})
	}
	if c.Review > 0 {
		chips = append(chips, shared.Chip{Tone: "accent", Text: itoa(c.Review) + " in review"})
	}
	if c.Done > 0 {
		chips = append(chips, shared.Chip{Tone: "ok", Text: itoa(c.Done) + " done"})
	}

	enabled, healthyN, runsAll := 0, 0, 0
	for _, j := range x.Jobs {
		if j.Enabled {
			enabled++
		}
		if healthy(j) {
			healthyN++
		}
		runsAll += j.Runs
	}

	shippedF := shared.Frac(c.Done, len(x.Tasks))
	reviewF := shared.Frac(c.Review, inFlight)
	boardF := shared.Frac(b.InProgress, b.Total)
	healthyF := shared.Frac(healthyN, enabled)
	boardMeter := shared.Meter{Label: "Board issues in progress (" + itoa(b.InProgress) + "/" + itoa(b.Total) + ")", Frac: shared.F(boardF),
		Display: pick(b.Total > 0, shared.Pct(boardF), "board empty or offline"), Hue: "var(--bn-warn)"}
	meters := []shared.Meter{
		{Label: "Kanban shipped (" + itoa(c.Done) + "/" + itoa(len(x.Tasks)) + ")", Frac: shared.F(shippedF), Display: pick(len(x.Tasks) > 0, shared.Pct(shippedF), "no tasks yet"), Hue: "var(--bn-ok)"},
		{Label: "Waiting on review (" + itoa(c.Review) + "/" + itoa(inFlight) + " in flight)", Frac: shared.F(reviewF), Display: pick(inFlight > 0, shared.Pct(reviewF), "nothing in flight"), Hue: "var(--bn-accent)"},
		boardMeter,
		{Label: "Crons healthy (" + itoa(healthyN) + "/" + itoa(enabled) + ")", Frac: shared.F(healthyF), Display: pick(enabled > 0, shared.Pct(healthyF), "none enabled"),
			Hue: pick(enabled > 0 && healthyN == enabled, "var(--bn-ok)", "var(--bn-warn)")},
	}

	// window: local days, oldest first, ending today
	start := shared.WindowStart(now, days)
	inWindow := func(iso string) (time.Time, bool) {
		t, ok := shared.ParseISO(iso)
		if !ok || t.Before(start) || t.After(now) {
			return time.Time{}, false
		}
		return t, true
	}
	var stamps []string
	touches := 0
	for _, t := range x.Tasks {
		if _, ok := inWindow(t.UpdatedAt); ok {
			stamps = append(stamps, t.UpdatedAt)
			touches++
		}
	}
	for _, i := range x.Issues {
		if i.UpdatedAt == nil {
			continue
		}
		if _, ok := inWindow(*i.UpdatedAt); ok {
			stamps = append(stamps, *i.UpdatedAt)
			touches++
		}
	}
	series := shared.DailySeries(stamps, days, now)

	rhythm := make([]shared.SeriesPoint, 7)
	for i, l := range weekdays {
		rhythm[i].Label = l
	}
	runsIn, failedIn := 0, 0
	for _, r := range x.Runs {
		t, ok := inWindow(r.StartedAt)
		if !ok {
			continue
		}
		rhythm[(int(t.In(time.Local).Weekday())+6)%7].Count++
		runsIn++
		if !r.OK {
			failedIn++
		}
	}

	perAgent := map[string]int{}
	var order []string
	for _, t := range x.Tasks {
		if t.Status == "done" {
			continue
		}
		if _, seen := perAgent[t.AgentID]; !seen {
			order = append(order, t.AgentID)
		}
		perAgent[t.AgentID]++
	}
	sort.SliceStable(order, func(i, j int) bool { return perAgent[order[i]] > perAgent[order[j]] })
	if len(order) > ownerCols {
		order = order[:ownerCols]
	}
	names := make([]string, len(order))
	for i, id := range order {
		names[i] = id
		if n, ok := x.AgentNames[id]; ok {
			names[i] = n
		}
	}
	short := shared.ShortLabels(names, 0)
	owners := make([]shared.SeriesPoint, len(order))
	for i, id := range order {
		owners[i] = shared.SeriesPoint{Label: short[i], Count: perAgent[id]}
	}
	var busiest *Owner
	if len(order) > 0 {
		busiest = &Owner{Name: names[0], Count: perAgent[order[0]]}
	}
	agentsOnKanban := map[string]bool{}
	for _, t := range x.Tasks {
		agentsOnKanban[t.AgentID] = true
	}

	var late []Job
	for _, j := range x.Jobs {
		if j.Enabled && !healthy(j) {
			late = append(late, j)
		}
	}
	value := c.Review + b.Blocked + len(late)
	var parts []string
	if c.Review > 0 {
		parts = append(parts, itoa(c.Review)+" in review")
	}
	if b.Blocked > 0 {
		parts = append(parts, itoa(b.Blocked)+" blocked")
	}
	if len(late) > 0 {
		parts = append(parts, shared.Plural(len(late), "cron")+" late or failing")
	}
	whole := inFlight + (b.Total - b.Done) + enabled
	headline := "Nothing is waiting on you."
	if len(parts) > 0 {
		headline = strings.Join(parts, " · ") + "."
	}
	var body string
	switch {
	case len(late) > 0:
		var ds []string
		for i, j := range late {
			if i == 3 {
				break
			}
			ds = append(ds, j.Description)
		}
		body = strings.Join(ds, " · ")
	case value > 0:
		body = shared.Plural(inFlight, "task") + " in flight on the kanban."
	case whole > 0:
		body = shared.Plural(inFlight, "task") + " open, nothing waiting on you."
	default:
		body = "No tasks, issues or crons yet."
	}

	caption := itoa(len(x.Tasks)) + " on the local kanban · " + itoa(len(x.Issues)) + " on the board"
	return Result{
		Headline: len(x.Tasks) + len(x.Issues), Counts: c, Board: b, BoardOnline: x.BoardConnected, Chips: chips,
		Caption: caption, Meters: meters,
		Foot:   shared.Plural(len(x.Jobs), "cron") + " · " + itoa(runsAll) + " runs recorded · " + shared.Plural(len(agentsOnKanban), "agent") + " on the kanban",
		Series: series, TouchesInWindow: touches,
		Cron:   Cron{RunsInWindow: runsIn, FailedInWindow: failedIn, Rhythm: rhythm},
		Owners: owners, BusiestOwner: busiest,
		Insight: shared.Insight{Value: value, Headline: headline, Body: body, Frac: shared.Frac(value, whole)},
	}
}
