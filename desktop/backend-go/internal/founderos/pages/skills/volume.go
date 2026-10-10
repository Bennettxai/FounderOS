package skills

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
)

// Meter hues, as the bridge's Monolith tokens (FounderOS v1 used --ok,
// --ramp-1, --accent and --ramp-4; the mono theme has no ramp, so the ramp
// steps become the text greys: color means status only).
const (
	HueOK     = "var(--bn-ok)"
	HueRamp1  = "var(--bn-text)"
	HueAccent = "var(--bn-accent)"
	HueRamp4  = "var(--bn-text-2)"
)

// OperatorSkill is one row of founderos_skills (FounderOS v1 SkillSchema).
type OperatorSkill struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Category     string   `json:"category"`
	Description  string   `json:"description"`
	OwnerAgentID *string  `json:"ownerAgentId"`
	Status       string   `json:"status"` // live | learning | planned
	Tools        []string `json:"tools"`
	Markdown     string   `json:"markdown"`
}

type VolumeInput struct {
	Claude   []CatalogSkill
	Operator []OperatorSkill
	// OperatorKnown is false when the operator table could not be read:
	// its numbers are then unknown, never zero.
	OperatorKnown bool
	AgentNames    map[string]string
}

type Counts struct {
	Claude   int `json:"claude"`
	User     int `json:"user"`
	Plugin   int `json:"plugin"`
	Operator int `json:"operator"`
	Live     int `json:"live"`
	Learning int `json:"learning"`
	Planned  int `json:"planned"`
}

type Chip struct {
	Tone string `json:"tone,omitempty"`
	Text string `json:"text"`
}

// Meter is the kit's Meter: Frac nil means unknown (no bar drawn).
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

type Insight struct {
	Value    int      `json:"value"`
	Headline string   `json:"headline"`
	Body     string   `json:"body"`
	Frac     *float64 `json:"frac"`
}

type VolumeResult struct {
	Headline   int     `json:"headline"`
	Counts     Counts  `json:"counts"`
	Chips      []Chip  `json:"chips"`
	Caption    string  `json:"caption"`
	Meters     []Meter `json:"meters"`
	Foot       string  `json:"foot"`
	Groups     []Point `json:"groups"`
	GroupCount int     `json:"groupCount"`
	Categories []Point `json:"categories"`
	Owners     []Point `json:"owners"`
	Unassigned int     `json:"unassigned"`
	Insight    Insight `json:"insight"`
}

const (
	groupCols = 8
	ownerCols = 6
)

func frac(n, d int) *float64 {
	f := 0.0
	if d > 0 {
		f = math.Max(0, math.Min(1, float64(n)/float64(d)))
	}
	return &f
}

func pct(f *float64) string { return fmt.Sprintf("%d%%", int(math.Round(*f*100))) }

func plural(n int, one string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, one)
	}
	return fmt.Sprintf("%d %ss", n, one)
}

var (
	splitSpace    = regexp.MustCompile(`\s+`)
	splitGroup    = regexp.MustCompile(`\s*·\s*|\s+`)
	splitCategory = regexp.MustCompile(`\s*·\s*|\s+&\s+|\s+`)
)

// ShortLabels is FounderOS v1 lib/short-labels.ts: the first token of each name,
// cut to max runes, numbered only when two would collide.
func ShortLabels(names []string, split *regexp.Regexp, max int, fallback string) []string {
	seen := map[string]int{}
	out := make([]string, len(names))
	for i, name := range names {
		t := strings.TrimSpace(name)
		base := split.Split(t, -1)[0]
		if base == "" {
			base = t
		}
		if base == "" {
			base = fallback
		}
		if r := []rune(base); len(r) > max {
			base = string(r[:max])
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

type tallied struct {
	key   string
	count int
}

// tally counts by key, first-seen order kept for ties after the sort.
func tally(keys []string) []tallied {
	idx := map[string]int{}
	var out []tallied
	for _, k := range keys {
		if i, ok := idx[k]; ok {
			out[i].count++
			continue
		}
		idx[k] = len(out)
		out = append(out, tallied{k, 1})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].count > out[j].count })
	return out
}

func toSeries(ranked []tallied, split *regexp.Regexp) []Point {
	names := make([]string, len(ranked))
	for i, r := range ranked {
		names[i] = r.key
	}
	labels := ShortLabels(names, split, 12, "Agent")
	out := make([]Point, len(ranked))
	for i, r := range ranked {
		out[i] = Point{labels[i], r.count}
	}
	return out
}

func isPlugin(s CatalogSkill) bool { return strings.Contains(s.Slug, ":") }

// Volume is one pure pass over the page's own rows: the Skill Volume
// headline and meters, skills per source group, operator skills per category
// and per owner, and the one "Drafts" card. Skills carry no dates, so nothing
// here draws a timeline.
func Volume(x VolumeInput) VolumeResult {
	plugin := 0
	for _, s := range x.Claude {
		if isPlugin(s) {
			plugin++
		}
	}
	claude := len(x.Claude)
	operator := len(x.Operator)
	learning, planned, opLive, owned, withTools := 0, 0, 0, 0, 0
	for _, s := range x.Operator {
		switch s.Status {
		case "learning":
			learning++
		case "planned":
			planned++
		case "live":
			opLive++
		}
		if s.OwnerAgentID != nil {
			owned++
		}
		if len(s.Tools) > 0 {
			withTools++
		}
	}
	// Claude Code skills on disk are live by definition.
	live := claude + opLive
	total := claude + operator
	v := VolumeResult{
		Headline: total,
		Counts:   Counts{Claude: claude, User: claude - plugin, Plugin: plugin, Operator: operator, Live: live, Learning: learning, Planned: planned},
		Chips:    []Chip{},
	}
	if live > 0 {
		v.Chips = append(v.Chips, Chip{Tone: "ok", Text: fmt.Sprintf("%d live", live)})
	}
	if learning > 0 {
		v.Chips = append(v.Chips, Chip{Tone: "warn", Text: fmt.Sprintf("%d learning", learning)})
	}
	if planned > 0 {
		v.Chips = append(v.Chips, Chip{Text: fmt.Sprintf("%d planned", planned)})
	}

	opCaption := fmt.Sprintf("%d operator", operator)
	if !x.OperatorKnown {
		opCaption = "operator unavailable"
	}
	v.Caption = fmt.Sprintf("%d user · %d plugin · %s", claude-plugin, plugin, opCaption)

	liveF, claudeF := frac(live, total), frac(claude, total)
	liveD, claudeD := "no skills yet", "none on disk"
	if total > 0 {
		liveD = pct(liveF)
	}
	if claude > 0 {
		claudeD = pct(claudeF)
	}
	v.Meters = []Meter{
		{Label: fmt.Sprintf("Live (%d/%d)", live, total), Frac: liveF, Display: liveD, Hue: HueOK},
		{Label: fmt.Sprintf("Claude Code on disk (%d/%d)", claude, total), Frac: claudeF, Display: claudeD, Hue: HueRamp1},
	}
	if x.OperatorKnown {
		ownedF, toolsF := frac(owned, operator), frac(withTools, operator)
		ownedD, toolsD := "no operator skills", "no operator skills"
		if operator > 0 {
			ownedD, toolsD = pct(ownedF), pct(toolsF)
		}
		v.Meters = append(v.Meters,
			Meter{Label: fmt.Sprintf("Operator skills owned (%d/%d)", owned, operator), Frac: ownedF, Display: ownedD, Hue: HueAccent},
			Meter{Label: fmt.Sprintf("Operator skills with tools (%d/%d)", withTools, operator), Frac: toolsF, Display: toolsD, Hue: HueRamp4},
		)
	} else {
		v.Meters = append(v.Meters,
			Meter{Label: "Operator skills owned", Display: "unavailable", Hue: HueAccent},
			Meter{Label: "Operator skills with tools", Display: "unavailable", Hue: HueRamp4},
		)
	}

	// Source groups: each plugin is its own group; operator skills are one.
	var groupKeys []string
	pluginNames := map[string]bool{}
	for _, s := range x.Claude {
		groupKeys = append(groupKeys, strings.TrimPrefix(s.Group, "Plugin · "))
		if isPlugin(s) {
			pluginNames[strings.SplitN(s.Slug, ":", 2)[0]] = true
		}
	}
	for range x.Operator {
		groupKeys = append(groupKeys, "Operator")
	}
	rankedGroups := tally(groupKeys)
	top := rankedGroups
	if len(top) > groupCols {
		top = top[:groupCols]
	}
	v.Groups = toSeries(top, splitGroup)
	v.GroupCount = len(rankedGroups)

	tools := map[string]bool{}
	var cats, owners []string
	for _, s := range x.Operator {
		for _, t := range s.Tools {
			tools[t] = true
		}
		cats = append(cats, s.Category)
		if s.OwnerAgentID != nil {
			name, ok := x.AgentNames[*s.OwnerAgentID]
			if !ok {
				name = *s.OwnerAgentID
			}
			owners = append(owners, name)
		}
	}
	v.Foot = fmt.Sprintf("%s · %s · %s wired", plural(len(rankedGroups), "group"), plural(len(pluginNames), "plugin"), plural(len(tools), "tool"))
	v.Categories = toSeries(tally(cats), splitCategory)
	rankedOwners := tally(owners)
	if len(rankedOwners) > ownerCols {
		rankedOwners = rankedOwners[:ownerCols]
	}
	v.Owners = toSeries(rankedOwners, splitSpace)
	v.Unassigned = operator - owned

	// The one gradient card: the operator drafts still to finish.
	if !x.OperatorKnown {
		v.Insight = Insight{Headline: "Operator skills are unavailable.", Body: "The operator skills table could not be read."}
		return v
	}
	var drafts []string
	for _, s := range x.Operator {
		if s.Status != "live" {
			drafts = append(drafts, s.Name)
		}
	}
	var parts []string
	if learning > 0 {
		parts = append(parts, fmt.Sprintf("%d learning", learning))
	}
	if planned > 0 {
		parts = append(parts, fmt.Sprintf("%d planned", planned))
	}
	headline := "Every operator skill is live."
	if len(parts) > 0 {
		headline = strings.Join(parts, " · ") + "."
	}
	body := "No operator skills yet."
	switch {
	case len(drafts) > 0:
		shown := drafts
		if len(shown) > 3 {
			shown = shown[:3]
		}
		body = strings.Join(shown, " · ")
	case operator > 0:
		body = plural(operator, "operator skill") + ", all live."
	}
	v.Insight = Insight{Value: len(drafts), Headline: headline, Body: body, Frac: frac(len(drafts), operator)}
	return v
}
