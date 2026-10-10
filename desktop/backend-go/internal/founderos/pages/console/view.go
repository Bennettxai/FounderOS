package console

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
)

// Roster is the agent roster's live count (founderos_agents).
type Roster struct {
	Active int `json:"active"`
	Total  int `json:"total"`
}

// SourceState says whether one comms lane answered on this load.
type SourceState struct {
	Source string `json:"source"`
	State  string `json:"state"` // ok | error | not_configured | stale
	Detail string `json:"detail,omitempty"`
}

// Charge is a Stripe RecentCharge.
type Charge struct {
	Amount      int64  `json:"amount"`
	Currency    string `json:"currency"`
	Description string `json:"description"`
	Created     int64  `json:"created"`
}

// Input is everything the console reads on one load. A nil Roster / Recent /
// Charges with its error set means the source could not be read.
type Input struct {
	Now         time.Time
	Connections []connectors.Status
	Brain       Brain

	Roster    *Roster
	Recent    []Run // the 40 newest runs, newest first
	Window    []Run // every run started in the last 15 days, newest first
	RosterErr string

	Feed    []Item
	Sources []SourceState

	Charges    []Charge
	ChargesErr string
	// ChargesNotConfigured: no Stripe key is set. Not an error (FounderOS v1
	// home fails soft to no charges row): no error line, total unknown.
	ChargesNotConfigured bool
}

type SystemsTile struct {
	Connected int      `json:"connected"`
	Total     int      `json:"total"`
	Bars      []string `json:"bars"` // connector states, first 16, board order
}

type AgentsTile struct {
	Active *int  `json:"active"`
	Total  *int  `json:"total"`
	Spark  []int `json:"spark"`
}

type CommsTile struct {
	Inbound int   `json:"inbound"`
	Spark   []int `json:"spark"`
}

// DoneItem is one ledger line: a finished run (✓) or a charge ($).
type DoneItem struct {
	Key  string    `json:"key"`
	Head string    `json:"head"`
	Tone string    `json:"tone"`
	Body string    `json:"body"`
	At   time.Time `json:"at"`
}

// View is the whole console, composed.
type View struct {
	GeneratedAt       time.Time           `json:"generatedAt"`
	Hero              []Segment           `json:"hero"`
	Systems           SystemsTile         `json:"systems"`
	Agents            AgentsTile          `json:"agents"`
	Comms             CommsTile           `json:"comms"`
	Brain             Brain               `json:"brain"`
	Volume            Volume              `json:"volume"`
	ChargedTodayCents *int64              `json:"chargedTodayCents"`
	Activity          []Point             `json:"activity"`
	ActivityTotal     int                 `json:"activityTotal"`
	Mix               []Point             `json:"mix"`
	FeedCount         int                 `json:"feedCount"`
	Sources           []SourceState       `json:"sources"`
	Attention         Attention           `json:"attention"`
	DoneCount         int                 `json:"doneCount"`
	Done              []DoneItem          `json:"done"`
	Connections       []connectors.Status `json:"connections"`
	Errors            map[string]string   `json:"errors"`
}

func withCommas(n int64) string {
	s := fmt.Sprint(n)
	var b strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	return b.String()
}

// FormatMoney renders whole units the way the ledger does ("$6,000").
func FormatMoney(cents int64, currency string) string {
	whole := (cents + 50) / 100
	if strings.EqualFold(currency, "usd") || currency == "" {
		return "$" + withCommas(whole)
	}
	return withCommas(whole) + " " + strings.ToUpper(currency)
}

// Build composes the console from one load's readings (app/page.tsx).
func Build(in Input) View {
	now := in.Now
	dayStart := LocalMidnight(now)
	v := View{GeneratedAt: now, Brain: in.Brain, Connections: in.Connections, Sources: in.Sources, Errors: map[string]string{}}
	if v.Connections == nil {
		v.Connections = []connectors.Status{}
	}
	if v.Sources == nil {
		v.Sources = []SourceState{}
	}

	down := 0
	v.Systems.Bars = []string{}
	for i, c := range in.Connections {
		if c.State == connectors.StateConnected {
			v.Systems.Connected++
		}
		if c.State == connectors.StateError {
			down++
		}
		if i < 16 {
			v.Systems.Bars = append(v.Systems.Bars, string(c.State))
		}
	}
	v.Systems.Total = len(in.Connections)

	starts := make([]string, 0, len(in.Window))
	startTimes := make([]time.Time, 0, len(in.Window))
	for _, r := range in.Window {
		starts = append(starts, r.StartedAt.UTC().Format(time.RFC3339))
		startTimes = append(startTimes, r.StartedAt)
	}
	var active, total *int
	if in.Roster != nil {
		a, t := in.Roster.Active, in.Roster.Total
		active, total = &a, &t
		v.Agents = AgentsTile{Active: active, Total: total, Spark: PerDay(starts, 7, now)}
	}
	if in.RosterErr != "" {
		v.Errors["roster"] = in.RosterErr
	}

	feedStamps := make([]string, len(in.Feed))
	for i, it := range in.Feed {
		feedStamps[i] = it.TS
	}
	inbound := InboundLast24h(in.Feed, now)
	v.Comms = CommsTile{Inbound: inbound, Spark: PerDay(feedStamps, 7, now)}
	v.FeedCount = len(in.Feed)
	v.Mix = SourceMix(in.Feed)

	failedRecent := 0
	for _, r := range in.Recent {
		if !r.OK {
			failedRecent++
		}
	}
	facts := PulseFacts{ConnectorsDown: down, Inbound: inbound, BrainConnected: in.Brain.Connected, EnginesUp: in.Brain.EnginesUp, EnginesTotal: in.Brain.EnginesTotal, Health: in.Brain.Health, FailedRuns: failedRecent}
	if in.Roster != nil {
		facts.ActiveAgents, facts.TotalAgents = in.Roster.Active, in.Roster.Total
	}
	v.Hero = StateOfWorld(facts)
	if in.Roster == nil {
		// "0 agents live · 0 idle" would be a lie; the roster is unknown.
		kept := v.Hero[:0]
		for _, s := range v.Hero {
			if strings.HasSuffix(s.Text, "agents live") || strings.HasSuffix(s.Text, " idle") {
				continue
			}
			kept = append(kept, s)
		}
		v.Hero = append(kept, Segment{"agent roster unknown", "warn"})
	}

	v.Volume = OperatingVolume(VolumeInput{
		Connected: v.Systems.Connected, TotalConnections: v.Systems.Total,
		ActiveAgents: active, TotalAgents: total,
		EnginesUp: in.Brain.EnginesUp, EnginesTotal: in.Brain.EnginesTotal, Health: in.Brain.Health,
		Runs: in.Window, Now: now,
	})
	v.Activity = DailySeries(startTimes, 14, now)
	for _, p := range v.Activity {
		v.ActivityTotal += p.Count
	}

	// Done today: collapsed OK runs since local midnight + today's charges.
	var done []DoneItem
	for _, r := range CollapseRuns(in.Recent) {
		if !r.OK || r.FinishedAt.Before(dayStart) {
			continue
		}
		body := r.AgentID + " · " + r.Summary
		if r.Repeat > 1 {
			body += fmt.Sprintf(" ×%d", r.Repeat)
		}
		done = append(done, DoneItem{Key: "run-" + r.ID, Head: "✓", Tone: "ok", Body: body, At: r.FinishedAt})
	}
	chargesToday := 0
	if in.ChargesErr != "" {
		v.Errors["charges"] = in.ChargesErr
	} else if !in.ChargesNotConfigured {
		var cents int64
		for _, c := range in.Charges {
			at := time.Unix(c.Created, 0)
			if at.Before(dayStart) {
				continue
			}
			chargesToday++
			cents += c.Amount
			desc := c.Description
			if desc == "" {
				desc = "charge"
			}
			done = append(done, DoneItem{Key: fmt.Sprintf("charge-%d-%d", c.Created, c.Amount), Head: "$", Tone: "accent", Body: FormatMoney(c.Amount, c.Currency) + " · " + desc, At: at})
		}
		v.ChargedTodayCents = &cents
	}
	sort.SliceStable(done, func(i, j int) bool { return done[i].At.After(done[j].At) })
	if len(done) > 8 {
		done = done[:8]
	}
	if done == nil {
		done = []DoneItem{}
	}
	v.Done = done
	v.DoneCount = v.Volume.RunsToday - v.Volume.FailedToday + chargesToday
	v.Attention = HomeAttention(inbound, v.Volume.FailedToday, down, v.DoneCount)
	return v
}
