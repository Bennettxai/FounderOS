package funnel

import (
	"regexp"
	"time"
)

// Radial funnel math (lib/funnel-radial.ts): leads enter at the rim in one
// acquisition wedge and travel inward ring by ring to the purchase core.

// Acquisition is one rim wedge.
type Acquisition struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// Acquisitions are the rim wedges in render order. X and LinkedIn share one.
var Acquisitions = []Acquisition{
	{"instagram", "Instagram"},
	{"youtube", "YouTube"},
	{"newsletter", "Newsletter"},
	{"x_linkedin", "X / LinkedIn"},
	{"form", "Forms"},
	{"word_of_mouth", "Word of mouth"},
}

var segmentIndex = func() map[string]int {
	m := map[string]int{}
	for i, a := range Acquisitions {
		m[a.ID] = i
	}
	return m
}()

// Keyword families, first match wins: youtube before form so "long-form"
// stays YouTube; explicit word of mouth before the form fallback.
var matchers = []struct {
	id string
	re *regexp.Regexp
}{
	{"youtube", regexp.MustCompile(`(?i)youtube|\byt\b|long-form`)},
	{"instagram", regexp.MustCompile(`(?i)instagram|\big\b|insta\b|reel|tiktok|meta ad|manychat|facebook|\bfb\b`)},
	{"newsletter", regexp.MustCompile(`(?i)newsletter|beehiiv`)},
	{"x_linkedin", regexp.MustCompile(`(?i)twitter|\bx thread|\bx post|\bx dm|\bon x\b|linkedin`)},
	{"word_of_mouth", regexp.MustCompile(`(?i)referr|word of mouth|recommend`)},
	{"form", regexp.MustCompile(`(?i)\bform\b|application|typeform|survey|landing page|opt.?in|waitlist`)},
}

// MatchAcquisition is the bare keyword pass; "" when nothing matches.
func MatchAcquisition(text string) string {
	for _, m := range matchers {
		if m.re.MatchString(text) {
			return m.id
		}
	}
	return ""
}

// AcquisitionFor: a stamped touch is trusted outright; otherwise classify
// the entry touch; paid traffic is the IG/FB machine; else word of mouth.
func AcquisitionFor(j Journey) string {
	if len(j.Touches) == 0 {
		return "word_of_mouth"
	}
	entry := j.Touches[0]
	if entry.Acquisition != "" {
		return entry.Acquisition
	}
	if m := MatchAcquisition(entry.Label); m != "" {
		return m
	}
	if entry.Channel == "ads" {
		return "instagram"
	}
	return "word_of_mouth"
}

// Origin is "where they came from" in words, for the dossier card.
type Origin struct {
	Segment string  `json:"segment"`
	Entry   *string `json:"entry"`
	Channel *string `json:"channel"`
	Source  *string `json:"source"`
	At      *string `json:"at"`
}

func OriginOf(j Journey) Origin {
	o := Origin{Segment: Acquisitions[segmentIndex[AcquisitionFor(j)]].Label}
	if len(j.Touches) > 0 {
		e := j.Touches[0]
		o.Entry, o.Channel, o.Source, o.At = sp(e.Label), sp(e.Channel), sp(e.Source), sp(e.At)
	}
	return o
}

// RadialNode is a space node placed on the circle.
type RadialNode struct {
	SpaceNode
	Segment     int    `json:"segment"`
	Rings       []int  `json:"rings"`
	CurrentRing int    `json:"currentRing"`
	Origin      Origin `json:"origin"`
}

// Segment is one wedge's tally.
type Segment struct {
	Acquisition
	Count     int `json:"count"`
	Converted int `json:"converted"`
}

type Radial struct {
	Nodes    []RadialNode `json:"nodes"`
	Segments []Segment    `json:"segments"`
}

// RadialModel: the same living nodes as the space, placed by wedge and ring.
// Every wedge is always present so the rim reads as a fixed compass.
func RadialModel(journeys []Journey, now time.Time) Radial {
	r := Radial{Nodes: []RadialNode{}, Segments: make([]Segment, len(Acquisitions))}
	for i, a := range Acquisitions {
		r.Segments[i] = Segment{Acquisition: a}
	}
	space := SpaceModel(journeys, now)
	for i, n := range space {
		seg := segmentIndex[AcquisitionFor(journeys[i])]
		r.Segments[seg].Count++
		if n.State == "converted" {
			r.Segments[seg].Converted++
		}
		r.Nodes = append(r.Nodes, RadialNode{SpaceNode: n, Segment: seg, Rings: n.Hubs, CurrentRing: n.CurrentHub, Origin: OriginOf(journeys[i])})
	}
	return r
}
