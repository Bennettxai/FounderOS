package skills

import (
	"math"
	"reflect"
	"testing"
)

func ptr(s string) *string { return &s }

// Ported from FounderOS v1 tests/skills-volume.test.ts.
var volInput = VolumeInput{
	Claude: []CatalogSkill{
		{Slug: "codex", Group: "Engineering"},
		{Slug: "mcp-builder", Group: "Engineering"},
		{Slug: "build", Group: "Spec · build · review"},
		{Slug: "superpowers:brainstorming", Group: "Plugin · superpowers"},
		{Slug: "superpowers:tdd", Group: "Plugin · superpowers"},
		{Slug: "superpowers:debug", Group: "Plugin · superpowers"},
		{Slug: "slack:digest", Group: "Plugin · slack"},
	},
	Operator: []OperatorSkill{
		{Name: "Cold opener", Category: "Sales", Status: "live", OwnerAgentID: ptr("closer"), Tools: []string{"gmail"}},
		{Name: "Objection map", Category: "Sales", Status: "learning", OwnerAgentID: ptr("closer"), Tools: []string{}},
		{Name: "Hook writer", Category: "Content", Status: "live", OwnerAgentID: ptr("scribe"), Tools: []string{"zernio", "gmail"}},
		{Name: "Ops audit", Category: "Ops", Status: "planned", Tools: []string{}},
	},
	OperatorKnown: true,
	AgentNames:    map[string]string{"closer": "Closer Agent", "scribe": "Scribe"},
}

func TestVolumeHeadlineCountsAndCaption(t *testing.T) {
	v := Volume(volInput)
	if v.Headline != 11 {
		t.Errorf("headline %d", v.Headline)
	}
	want := Counts{Claude: 7, User: 3, Plugin: 4, Operator: 4, Live: 9, Learning: 1, Planned: 1}
	if v.Counts != want {
		t.Errorf("counts %+v", v.Counts)
	}
	if v.Caption != "3 user · 4 plugin · 4 operator" {
		t.Errorf("caption %q", v.Caption)
	}
	if !reflect.DeepEqual(v.Chips, []Chip{{Tone: "ok", Text: "9 live"}, {Tone: "warn", Text: "1 learning"}, {Text: "1 planned"}}) {
		t.Errorf("chips %+v", v.Chips)
	}
}

func near(a *float64, b float64) bool { return a != nil && math.Abs(*a-b) < 1e-9 }

func TestVolumeMeters(t *testing.T) {
	v := Volume(volInput)
	labels := []string{}
	for _, m := range v.Meters {
		labels = append(labels, m.Label)
	}
	if !reflect.DeepEqual(labels, []string{"Live (9/11)", "Claude Code on disk (7/11)", "Operator skills owned (3/4)", "Operator skills with tools (2/4)"}) {
		t.Errorf("labels %v", labels)
	}
	if !near(v.Meters[0].Frac, 9.0/11) || v.Meters[0].Display != "82%" || v.Meters[0].Hue != HueOK {
		t.Errorf("m0 %+v", v.Meters[0])
	}
	if v.Meters[1].Display != "64%" || v.Meters[1].Hue != HueRamp1 {
		t.Errorf("m1 %+v", v.Meters[1])
	}
	if !near(v.Meters[2].Frac, 0.75) || v.Meters[2].Display != "75%" || v.Meters[2].Hue != HueAccent {
		t.Errorf("m2 %+v", v.Meters[2])
	}
	if !near(v.Meters[3].Frac, 0.5) || v.Meters[3].Display != "50%" || v.Meters[3].Hue != HueRamp4 {
		t.Errorf("m3 %+v", v.Meters[3])
	}
	if v.Foot != "5 groups · 2 plugins · 2 tools wired" {
		t.Errorf("foot %q", v.Foot)
	}
}

func TestVolumeSeries(t *testing.T) {
	v := Volume(volInput)
	wantGroups := []Point{{"Operator", 4}, {"superpowers", 3}, {"Engineering", 2}, {"Spec", 1}, {"slack", 1}}
	if !reflect.DeepEqual(v.Groups, wantGroups) || v.GroupCount != 5 {
		t.Errorf("groups %+v (%d)", v.Groups, v.GroupCount)
	}
	if !reflect.DeepEqual(v.Categories, []Point{{"Sales", 2}, {"Content", 1}, {"Ops", 1}}) {
		t.Errorf("categories %+v", v.Categories)
	}
	if !reflect.DeepEqual(v.Owners, []Point{{"Closer", 2}, {"Scribe", 1}}) || v.Unassigned != 1 {
		t.Errorf("owners %+v unassigned %d", v.Owners, v.Unassigned)
	}
	if v.Insight.Value != 2 || v.Insight.Headline != "1 learning · 1 planned." || v.Insight.Body != "Objection map · Ops audit" || !near(v.Insight.Frac, 0.5) {
		t.Errorf("insight %+v", v.Insight)
	}
}

func TestVolumeEmptyIsHonest(t *testing.T) {
	e := Volume(VolumeInput{OperatorKnown: true, AgentNames: map[string]string{}})
	if e.Headline != 0 || len(e.Chips) != 0 || len(e.Groups) != 0 {
		t.Errorf("empty %+v", e)
	}
	displays := []string{}
	for _, m := range e.Meters {
		if m.Frac == nil || *m.Frac != 0 {
			t.Errorf("empty meter frac %v", m.Frac)
		}
		displays = append(displays, m.Display)
	}
	if !reflect.DeepEqual(displays, []string{"no skills yet", "none on disk", "no operator skills", "no operator skills"}) {
		t.Errorf("displays %v", displays)
	}
	if e.Insight.Value != 0 || *e.Insight.Frac != 0 || e.Insight.Headline != "Every operator skill is live." || e.Insight.Body != "No operator skills yet." {
		t.Errorf("insight %+v", e.Insight)
	}
	g := Volume(VolumeInput{OperatorKnown: true, Operator: []OperatorSkill{{Name: "A", Category: "Ops", Status: "live", Tools: []string{}}}})
	if g.Insight.Body != "1 operator skill, all live." || g.Unassigned != 1 {
		t.Errorf("all live %+v", g.Insight)
	}
}

func TestVolumeWithOperatorSkillsUnreachableReadsUnknown(t *testing.T) {
	in := volInput
	in.Operator = nil
	in.OperatorKnown = false
	v := Volume(in)
	// the operator meters cannot be measured: unknown, never 0
	for _, m := range v.Meters[2:] {
		if m.Frac != nil || m.Display != "unavailable" {
			t.Errorf("operator meter %+v should be unknown", m)
		}
	}
	// the catalog-wide meters only count what is known
	if v.Meters[0].Label != "Live (7/7)" {
		t.Errorf("live meter %+v", v.Meters[0])
	}
	if v.Insight.Frac != nil || v.Insight.Headline != "Operator skills are unavailable." {
		t.Errorf("insight %+v", v.Insight)
	}
	if v.Caption != "3 user · 4 plugin · operator unavailable" {
		t.Errorf("caption %q", v.Caption)
	}
}

func TestShortLabels(t *testing.T) {
	got := ShortLabels([]string{"Closer Agent", "Closer Bot", "", "Scribe"}, splitSpace, 12, "Agent")
	if !reflect.DeepEqual(got, []string{"Closer", "Closer 2", "Agent", "Scribe"}) {
		t.Errorf("got %v", got)
	}
	if got := ShortLabels([]string{"superlongpluginname"}, splitSpace, 12, "Agent"); got[0] != "superlongplu" {
		t.Errorf("cut %v", got)
	}
}
