// Package funnel is the /funnel page's pure logic, ported from FounderOS v1
// lib/funnel.ts, lib/funnel-leads.ts, lib/funnel-stripe.ts,
// lib/funnel-trakyo.ts, lib/funnel-radial.ts, lib/funnel-volume.ts,
// lib/funnel-contact.ts and lib/funnel-compose.ts. The network lives in the
// connectors; everything here is source-agnostic and testable.
package funnel

import (
	"math"
	"sort"
	"time"
)

// Touch is one step of a journey (FunnelTouchSchema). Acquisition is set
// only when the source system (Trakyo) attributed the touch to a rim wedge.
type Touch struct {
	ID          string `json:"id"`
	ContactID   string `json:"contactId"`
	Seq         int    `json:"seq"`
	Stage       string `json:"stage"`
	Channel     string `json:"channel"`
	Label       string `json:"label"`
	Source      string `json:"source"`
	At          string `json:"at"` // YYYY-MM-DD
	Acquisition string `json:"acquisition,omitempty"`
}

// Journey is one client's path through the funnel (FunnelJourneySchema).
// Pointers are the schema's nullables and read null.
type Journey struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Venture      string   `json:"venture"`
	Status       string   `json:"status"`
	Product      *string  `json:"product"`
	AmountUSD    *float64 `json:"amountUsd"`
	Relationship string   `json:"relationship"`
	Likelihood   int      `json:"likelihood"`
	URL          *string  `json:"url"`
	Email        *string  `json:"email"`
	Phone        *string  `json:"phone"`
	Person       *string  `json:"person"`
	Company      *string  `json:"company"`
	Role         *string  `json:"role"`
	LinkedIn     *string  `json:"linkedin"`
	CreatedAt    string   `json:"createdAt"`
	Touches      []Touch  `json:"touches"`
}

func sp(s string) *string { return &s }

// Stage is one of the five steps a lead moves through.
type Stage struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// Stages is FounderOS v1 lib/funnel.ts FUNNEL_STAGES: the canonical order with
// display labels. Every live lane maps onto these ids (a Typeform submission
// is a first touch, a booked call engages, a held call nurtures, a payment
// converts).
var Stages = []Stage{
	{"first_touch", "First touch"},
	{"engaged", "Engaged"},
	{"nurtured", "Nurtured"},
	{"opted_in", "Opted in"},
	{"converted", "Converted"},
}

var stageIndex = map[string]int{"first_touch": 0, "engaged": 1, "nurtured": 2, "opted_in": 3, "converted": 4}

// StageLabel is the display label for every stored stage id.
var StageLabel = func() map[string]string {
	m := make(map[string]string, len(Stages))
	for _, s := range Stages {
		m[s.ID] = s.Label
	}
	return m
}()

// ValidStage reports whether s is a stored stage id (FunnelStageSchema).
func ValidStage(s string) bool { _, ok := stageIndex[s]; return ok }

// ValidVenture reports whether v is a funnel venture (FunnelVentureSchema).
func ValidVenture(v string) bool { return v == "vantage" || v == "launchpad-cohort" }

const (
	// StallDays: quiet for more than this before converting runs red.
	StallDays = 7
	// DecayDays: quiet past this decays the lead into the archive.
	DecayDays = 90
	// DecayFadeStart: nodes stay neutral until here, then fade toward red.
	DecayFadeStart = 21
	// AttentionCap bounds each attention rail.
	AttentionCap = 4
	// PushLikelihood is the hot-lead bar for the push rail.
	PushLikelihood = 70
)

// DecayFactor is 0 through DecayFadeStart, ramping to 1 at DecayDays.
// Converted never decays.
func DecayFactor(days int, status string) float64 {
	if status == "converted" {
		return 0
	}
	return math.Min(1, math.Max(0, float64(days-DecayFadeStart)/float64(DecayDays-DecayFadeStart)))
}

// Meta is a journey's liveness at a moment.
type Meta struct {
	DaysSinceLastTouch int    `json:"daysSinceLastTouch"`
	State              string `json:"state"` // converted | stalled | active | decayed
}

// MetaOf: converted is green, a pre-conversion lead quiet past StallDays is
// stalled (first_touch never stalls), past DecayDays it is decayed.
func MetaOf(j Journey, now time.Time) Meta {
	last := j.CreatedAt
	if n := len(j.Touches); n > 0 {
		last = j.Touches[n-1].At
	}
	days := 0
	if at, err := time.Parse("2006-01-02", last); err == nil {
		days = int(math.Floor(float64(now.Sub(at).Milliseconds()) / 86_400_000))
		if days < 0 {
			days = 0
		}
	}
	canStall := j.Status != "converted" && j.Status != "first_touch"
	state := "active"
	switch {
	case j.Status == "converted":
		state = "converted"
	case days > DecayDays:
		state = "decayed"
	case canStall && days > StallDays:
		state = "stalled"
	}
	return Meta{DaysSinceLastTouch: days, State: state}
}

// Queue is what the operator should act on today.
type Queue struct {
	PushNow []Journey `json:"pushNow"`
	SaveNow []Journey `json:"saveNow"`
}

// Attention: pushNow = hot leads still in active motion, freshest first;
// saveNow = stalled or fading leads, highest likelihood first. Disjoint.
func Attention(journeys []Journey, now time.Time) Queue {
	type jm struct {
		j Journey
		m Meta
	}
	var push, save []jm
	for _, j := range journeys {
		m := MetaOf(j, now)
		decay := DecayFactor(m.DaysSinceLastTouch, j.Status)
		if m.State == "active" && j.Status != "converted" && j.Likelihood >= PushLikelihood && decay == 0 {
			push = append(push, jm{j, m})
		}
		if m.State != "decayed" && j.Status != "converted" && (m.State == "stalled" || decay > 0) {
			save = append(save, jm{j, m})
		}
	}
	sort.SliceStable(push, func(a, b int) bool {
		if push[a].m.DaysSinceLastTouch != push[b].m.DaysSinceLastTouch {
			return push[a].m.DaysSinceLastTouch < push[b].m.DaysSinceLastTouch
		}
		return push[a].j.Likelihood > push[b].j.Likelihood
	})
	sort.SliceStable(save, func(a, b int) bool {
		if save[a].j.Likelihood != save[b].j.Likelihood {
			return save[a].j.Likelihood > save[b].j.Likelihood
		}
		return save[a].m.DaysSinceLastTouch > save[b].m.DaysSinceLastTouch
	})
	q := Queue{PushNow: []Journey{}, SaveNow: []Journey{}}
	for i := 0; i < len(push) && i < AttentionCap; i++ {
		q.PushNow = append(q.PushNow, push[i].j)
	}
	for i := 0; i < len(save) && i < AttentionCap; i++ {
		q.SaveNow = append(q.SaveNow, save[i].j)
	}
	return q
}

// Split: the space renders actives; the archive lists what decayed.
func Split(journeys []Journey, now time.Time) (active, archived []Journey) {
	active, archived = []Journey{}, []Journey{}
	for _, j := range journeys {
		if MetaOf(j, now).State == "decayed" {
			archived = append(archived, j)
		} else {
			active = append(active, j)
		}
	}
	return active, archived
}

// SpaceNode is one client in the open funnel space.
type SpaceNode struct {
	ID                 string   `json:"id"`
	Name               string   `json:"name"`
	Venture            string   `json:"venture"`
	Status             string   `json:"status"`
	Relationship       string   `json:"relationship"`
	Likelihood         int      `json:"likelihood"`
	State              string   `json:"state"`
	DaysSinceLastTouch int      `json:"daysSinceLastTouch"`
	Product            *string  `json:"product"`
	AmountUSD          *float64 `json:"amountUsd"`
	Hubs               []int    `json:"hubs"`
	CurrentHub         int      `json:"currentHub"`
	Radius             float64  `json:"radius"`
	Decay              float64  `json:"decay"`
	URL                *string  `json:"url"`
	Email              *string  `json:"email"`
	Phone              *string  `json:"phone"`
	Person             *string  `json:"person"`
	Company            *string  `json:"company"`
	Role               *string  `json:"role"`
	LinkedIn           *string  `json:"linkedin"`
	Touches            []Touch  `json:"touches"`
}

// SpaceModel: each journey becomes one node; repeated touches inside a
// stage collapse into one hub visit.
func SpaceModel(journeys []Journey, now time.Time) []SpaceNode {
	out := make([]SpaceNode, 0, len(journeys))
	for _, j := range journeys {
		hubs := []int{}
		for _, t := range j.Touches {
			col := stageIndex[t.Stage]
			if len(hubs) == 0 || hubs[len(hubs)-1] != col {
				hubs = append(hubs, col)
			}
		}
		if len(hubs) == 0 {
			hubs = []int{0}
		}
		m := MetaOf(j, now)
		touches := j.Touches
		if touches == nil {
			touches = []Touch{}
		}
		out = append(out, SpaceNode{
			ID: j.ID, Name: j.Name, Venture: j.Venture, Status: j.Status, Relationship: j.Relationship,
			Likelihood: j.Likelihood, State: m.State, DaysSinceLastTouch: m.DaysSinceLastTouch,
			Product: j.Product, AmountUSD: j.AmountUSD, Hubs: hubs, CurrentHub: hubs[len(hubs)-1],
			Radius: 2.5 + float64(j.Likelihood)/100*3, Decay: DecayFactor(m.DaysSinceLastTouch, j.Status),
			URL: j.URL, Email: j.Email, Phone: j.Phone, Person: j.Person, Company: j.Company, Role: j.Role,
			LinkedIn: j.LinkedIn, Touches: touches,
		})
	}
	return out
}

// StageRow is one bar of the funnel: journeys that progressed at least this far.
type StageRow struct {
	Stage              string   `json:"stage"`
	Total              int      `json:"total"`
	Organic            int      `json:"organic"`
	Ads                int      `json:"ads"`
	ConversionFromPrev *float64 `json:"conversionFromPrev"`
}

// Summary is FunnelSummarySchema.
type Summary struct {
	Clients    int        `json:"clients"`
	Converted  int        `json:"converted"`
	RevenueUSD float64    `json:"revenueUsd"`
	Stages     []StageRow `json:"stages"`
}

// Summarize: per-stage reached counts (furthest stage at or past the bar),
// the organic/ads split by first touch, and stage-to-stage conversion.
func Summarize(journeys []Journey) Summary {
	s := Summary{Stages: make([]StageRow, 0, len(Stages))}
	for _, j := range journeys {
		if j.Status == "converted" {
			s.Converted++
			if j.AmountUSD != nil {
				s.RevenueUSD += *j.AmountUSD
			}
		}
	}
	s.Clients = len(journeys)
	for i, st := range Stages {
		row := StageRow{Stage: st.ID}
		for _, j := range journeys {
			if stageIndex[j.Status] < i {
				continue
			}
			row.Total++
			if len(j.Touches) > 0 {
				switch j.Touches[0].Channel {
				case "organic":
					row.Organic++
				case "ads":
					row.Ads++
				}
			}
		}
		if i > 0 && s.Stages[i-1].Total > 0 {
			v := math.Floor(float64(row.Total)/float64(s.Stages[i-1].Total)*1000+0.5) / 10
			row.ConversionFromPrev = &v
		}
		s.Stages = append(s.Stages, row)
	}
	return s
}
