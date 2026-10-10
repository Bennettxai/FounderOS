// Package content is the bridge port of FounderOS v1's /content and
// /content/lead-magnets view logic: lib/content.ts (the content crew),
// lib/content-volume.ts, lib/lead-magnet-volume.ts, lib/short-labels.ts and
// the lead-magnet validation from app/api/lead-magnets. Pure except store.go,
// which reads and writes the founderos_* tables.
package content

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/zernio"
	"github.com/rhl/businessos-backend/internal/founderos/pages/pagekit"
)

// DeptID is the Marketing/Growth pillar, where the content agent
// (social-agent) and its Zernio publisher and creative workers live.
const DeptID = "dept-marketing-growth"

// Agent is a founderos_agents row as the page needs it.
type Agent struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Role         string   `json:"role"`
	Status       string   `json:"status"`
	Tier         string   `json:"tier"`
	Description  string   `json:"description"`
	Model        string   `json:"model"`
	Tools        []string `json:"tools"`
	ParentID     *string  `json:"parentId"`
	DepartmentID string   `json:"departmentId"`
}

// Run is the slice of a founderos_agent_runs row the volume reads.
type Run struct {
	AgentID   string `json:"agentId"`
	StartedAt string `json:"startedAt"`
	OK        bool   `json:"ok"`
}

// ContentAgents is contentAgents(): the content pillar, lead first, then
// the workers by name.
func ContentAgents(all []Agent) []Agent {
	out := []Agent{}
	for _, a := range all {
		if a.DepartmentID == DeptID {
			out = append(out, a)
		}
	}
	isLead := func(a Agent) int {
		if a.Tier == "lead" || a.ParentID == nil {
			return 0
		}
		return 1
	}
	sort.SliceStable(out, func(i, j int) bool {
		if li, lj := isLead(out[i]), isLead(out[j]); li != lj {
			return li < lj
		}
		return localeLess(out[i].Name, out[j].Name)
	})
	return out
}

// localeLess approximates String.localeCompare for plain names: case-folded
// first, then byte order.
func localeLess(a, b string) bool {
	la, lb := strings.ToLower(a), strings.ToLower(b)
	if la != lb {
		return la < lb
	}
	return a < b
}

// ShortLabels is lib/short-labels.ts with its defaults (whitespace split, no
// cap, fallback "Agent"): the first word of each name, numbered only when
// two would collide.
func ShortLabels(names []string) []string {
	seen := map[string]int{}
	out := make([]string, len(names))
	for i, name := range names {
		trimmed := strings.TrimSpace(name)
		base := ""
		if f := strings.Fields(trimmed); len(f) > 0 {
			base = f[0]
		}
		if base == "" {
			base = trimmed
		}
		if base == "" {
			base = "Agent"
		}
		seen[base]++
		if n := seen[base]; n == 1 {
			out[i] = base
		} else {
			out[i] = fmt.Sprintf("%s %d", base, n)
		}
	}
	return out
}

// VolumeInput is ContentVolumeInput. PostsKnown false means Zernio gave no
// answer: the history is unknown, not empty.
type VolumeInput struct {
	Crew        []Agent
	Runs        []Run
	LeadMagnets []LeadMagnet
	RecentCount int
	PostDays    []zernio.PostDay
	PostsKnown  bool
	Today       string // YYYY-MM-DD (UTC)
	Days        int
}

type MagnetCounts struct {
	Total    int `json:"total"`
	Live     int `json:"live"`
	Draft    int `json:"draft"`
	Paused   int `json:"paused"`
	Archived int `json:"archived"`
}

// Insight is the one gradient card. Display empty means the card counts
// Value up; otherwise it prints Display.
type Insight struct {
	Value    int     `json:"value"`
	Display  string  `json:"display,omitempty"`
	Headline string  `json:"headline"`
	Body     string  `json:"body"`
	Frac     float64 `json:"frac"`
}

type ContentVolume struct {
	Headline      int             `json:"headline"`
	Chips         []pagekit.Chip  `json:"chips"`
	Caption       string          `json:"caption"`
	Meters        []pagekit.Meter `json:"meters"`
	Foot          string          `json:"foot"`
	Series        []pagekit.Point `json:"series"`
	PostsInWindow int             `json:"postsInWindow"`
	ActiveDays    int             `json:"activeDays"`
	CrewRuns      []pagekit.Point `json:"crewRuns"`
	RunsInWindow  int             `json:"runsInWindow"`
	Magnets       MagnetCounts    `json:"magnets"`
	Insight       Insight         `json:"insight"`
}

const (
	dayMs    = 24 * time.Hour
	crewCols = 6
)

func pctText(f float64) string { return fmt.Sprintf("%d%%", int(math.Round(f*100))) }

func utcDay(s string) (time.Time, bool) {
	t, err := time.Parse("2006-01-02", s)
	return t, err == nil
}

func parseStamp(s string) (time.Time, bool) {
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04:05.000Z", "2006-01-02 15:04:05Z07:00", "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// Volume is contentVolume.
func Volume(x VolumeInput) ContentVolume {
	days := x.Days
	if days <= 0 {
		days = 30
	}
	end, _ := utcDay(x.Today)
	start := end.Add(-time.Duration(days-1) * dayMs)
	endOfToday := end.Add(dayMs)

	keys := make([]string, days)
	buckets := map[string]int{}
	series := make([]pagekit.Point, days)
	for i := 0; i < days; i++ {
		d := start.Add(time.Duration(i) * dayMs)
		keys[i] = d.Format("2006-01-02")
		buckets[keys[i]] = 0
		series[i].Label = pagekit.ShortDate(d)
	}
	lastPost := ""
	for _, p := range x.PostDays {
		key := p.Date
		if len(key) > 10 {
			key = key[:10]
		}
		if key <= x.Today && (lastPost == "" || key > lastPost) {
			lastPost = key
		}
		if _, ok := buckets[key]; ok {
			buckets[key]++
		}
	}
	postsInWindow, activeDays := 0, 0
	for i, k := range keys {
		series[i].Count = float64(buckets[k])
		postsInWindow += buckets[k]
		if buckets[k] > 0 {
			activeDays++
		}
	}

	crewIDs := map[string]bool{}
	for _, a := range x.Crew {
		crewIDs[a.ID] = true
	}
	perAgent := map[string]int{}
	runsInWindow, runsOK := 0, 0
	for _, r := range x.Runs {
		t, ok := parseStamp(r.StartedAt)
		if !ok || t.Before(start) || !t.Before(endOfToday) || !crewIDs[r.AgentID] {
			continue
		}
		runsInWindow++
		perAgent[r.AgentID]++
		if r.OK {
			runsOK++
		}
	}
	names := make([]string, len(x.Crew))
	for i, a := range x.Crew {
		names[i] = a.Name
	}
	labels := ShortLabels(names)
	crewRuns := []pagekit.Point{}
	for i, a := range x.Crew {
		if i >= crewCols {
			break
		}
		crewRuns = append(crewRuns, pagekit.Point{Label: labels[i], Count: float64(perAgent[a.ID])})
	}

	magnets := MagnetCounts{Total: len(x.LeadMagnets)}
	for _, m := range x.LeadMagnets {
		switch m.Status {
		case "live":
			magnets.Live++
		case "draft":
			magnets.Draft++
		case "paused":
			magnets.Paused++
		case "archived":
			magnets.Archived++
		}
	}
	live := magnets.Live
	activeCrew := 0
	for _, a := range x.Crew {
		if a.Status == "active" {
			activeCrew++
		}
	}

	chips := []pagekit.Chip{}
	if activeDays > 0 {
		chips = append(chips, pagekit.Chip{Tone: "accent", Text: pagekit.Plural(activeDays, "active day", "")})
	}
	if live > 0 {
		word := "magnets"
		if live == 1 {
			word = "magnet"
		}
		chips = append(chips, pagekit.Chip{Tone: "ok", Text: fmt.Sprintf("%d %s live", live, word)})
	}

	frac := func(n, d int) float64 { return pagekit.Clamp01(pagekit.Frac(float64(n), float64(d))) }
	meters := []pagekit.Meter{}
	if postsInWindow > 0 {
		f := frac(activeDays, days)
		meters = append(meters, pagekit.Known(fmt.Sprintf("Active posting days (%d/%d)", activeDays, days), f, pctText(f), pagekit.Accent))
	}
	if n := len(x.LeadMagnets); n > 0 {
		meters = append(meters, pagekit.Known(fmt.Sprintf("Lead magnets live (%d/%d)", live, n), frac(live, n), fmt.Sprintf("%d of %d", live, n), pagekit.OK))
	}
	if n := len(x.Crew); n > 0 {
		f := frac(activeCrew, n)
		meters = append(meters, pagekit.Known(fmt.Sprintf("Crew active (%d/%d)", activeCrew, n), f, pctText(f), pagekit.Ramp1))
	}
	if runsInWindow > 0 {
		meters = append(meters, pagekit.Known(fmt.Sprintf("Crew runs OK (%d/%d)", runsOK, runsInWindow), frac(runsOK, runsInWindow),
			fmt.Sprintf("%d ok · %d failed", runsOK, runsInWindow-runsOK), pagekit.Ramp4))
	}

	var insight Insight
	switch {
	case !x.PostsKnown:
		insight = Insight{Display: "—", Headline: "Posting history unavailable.", Body: "Zernio did not answer, so days since the last post are unknown."}
	case lastPost == "":
		insight = Insight{Display: "none", Headline: "Nothing posted on record yet.", Body: "Posts show here once Zernio reports its history."}
	default:
		lp, _ := utcDay(lastPost)
		quiet := int(math.Max(0, math.Round(end.Sub(lp).Hours()/24)))
		headline := "Posted today."
		if quiet != 0 {
			headline = pagekit.Plural(quiet, "day", "") + " since the last post went out."
		}
		insight = Insight{
			Value:    quiet,
			Headline: headline,
			Body:     fmt.Sprintf("%s in the last %d · ticks light with posting consistency.", pagekit.Plural(activeDays, "active day", ""), days),
			Frac:     frac(activeDays, days),
		}
	}

	caption := fmt.Sprintf("no posts on record in the last %d days", days)
	if !x.PostsKnown {
		caption = "posting history unavailable · Zernio not answering"
	} else if postsInWindow > 0 {
		caption = fmt.Sprintf("posts out through Zernio, last %d days", days)
	}

	return ContentVolume{
		Headline:      postsInWindow,
		Chips:         chips,
		Caption:       caption,
		Meters:        meters,
		Foot:          fmt.Sprintf("%s · %s · %s pulled", pagekit.Plural(len(x.Crew), "agent", ""), pagekit.Plural(len(x.LeadMagnets), "lead magnet", ""), pagekit.Plural(x.RecentCount, "recent post", "")),
		Series:        series,
		PostsInWindow: postsInWindow,
		ActiveDays:    activeDays,
		CrewRuns:      crewRuns,
		RunsInWindow:  runsInWindow,
		Magnets:       magnets,
		Insight:       insight,
	}
}
