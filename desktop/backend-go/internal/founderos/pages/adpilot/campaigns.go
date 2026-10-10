// Package adpilot is the pure logic behind the /adpilot page: the campaign
// deck (FounderOS v1 lib/adpilot.ts, lib/adpilot-data.ts) and Adscout's ad
// library over the local Foreplay snapshot store (lib/foreplay/{store,wall,
// signals,mine,watchlist,saved,sync}.ts). The Foreplay API client and the
// store's read side live in connectors/foreplay; this package adds the
// writes, the derived facts and the user-triggered cycles.
package adpilot

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
)

// GeoPoint is one city the campaign's leads book from.
type GeoPoint struct {
	City     string  `json:"city"`
	Country  string  `json:"country"`
	Lat      float64 `json:"lat"`
	Lng      float64 `json:"lng"`
	Leads    float64 `json:"leads"`
	Bookings float64 `json:"bookings"`
}

// DailyPoint is one day of delivery.
type DailyPoint struct {
	Date     string  `json:"date"`
	Spend    float64 `json:"spend"`
	Leads    float64 `json:"leads"`
	Bookings float64 `json:"bookings"`
}

type Period struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// Audience shares (percent) by age band, gender and placement.
type Audience struct {
	Age        map[string]float64 `json:"age"`
	Gender     map[string]float64 `json:"gender"`
	Placements map[string]float64 `json:"placements"`
}

// Campaign mirrors CampaignSchema.
type Campaign struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Objective   string       `json:"objective"` // leads | bookings | purchases
	Status      string       `json:"status"`    // active | paused
	Platform    string       `json:"platform"`
	Period      Period       `json:"period"`
	Spend       float64      `json:"spend"`
	Impressions float64      `json:"impressions"`
	Clicks      float64      `json:"clicks"`
	Leads       float64      `json:"leads"`
	Bookings    float64      `json:"bookings"`
	Purchases   float64      `json:"purchases"`
	Revenue     float64      `json:"revenue"`
	Audience    Audience     `json:"audience"`
	Geo         []GeoPoint   `json:"geo"`
	Daily       []DailyPoint `json:"daily,omitempty"`
}

var campaignKeys = []string{"id", "name", "objective", "status", "platform", "period", "spend", "impressions",
	"clicks", "leads", "bookings", "purchases", "revenue", "audience", "geo"}

func (c *Campaign) UnmarshalJSON(b []byte) error {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(b, &m); err != nil || m == nil {
		return errors.New("campaign: expected an object")
	}
	for _, k := range campaignKeys {
		if v, ok := m[k]; !ok || string(v) == "null" {
			return fmt.Errorf("campaign: missing %q", k)
		}
	}
	type plain Campaign
	var p plain
	if err := json.Unmarshal(b, &p); err != nil {
		return fmt.Errorf("campaign %q: %w", m["id"], err)
	}
	switch p.Objective {
	case "leads", "bookings", "purchases":
	default:
		return fmt.Errorf("campaign %s: objective %q", p.ID, p.Objective)
	}
	if p.Status != "active" && p.Status != "paused" {
		return fmt.Errorf("campaign %s: status %q", p.ID, p.Status)
	}
	for _, g := range p.Geo {
		if g.Leads < 0 || g.Bookings < 0 || g.Leads != math.Trunc(g.Leads) || g.Bookings != math.Trunc(g.Bookings) {
			return fmt.Errorf("campaign %s: geo %s counts must be whole and >= 0", p.ID, g.City)
		}
	}
	*c = Campaign(p)
	return nil
}

// CampaignDataPath is ADPILOT_DATA_PATH, else data/adpilot-campaigns.json
// under the working directory, as in FounderOS v1.
func CampaignDataPath() string {
	if p := os.Getenv("ADPILOT_DATA_PATH"); p != "" {
		return p
	}
	return filepath.Join("data", "adpilot-campaigns.json")
}

// ReadCampaignFile reads the ad-account sync's campaign file. An absent file
// is the honest "no ad account connected" state (no campaigns, no error); a
// corrupt or invalid one is an error, never an empty deck.
func ReadCampaignFile(path string) ([]Campaign, *string, error) {
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return []Campaign{}, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	var f struct {
		Campaigns *[]Campaign `json:"campaigns"`
		SyncedAt  *string     `json:"syncedAt"`
	}
	if err := json.Unmarshal(raw, &f); err != nil {
		return nil, nil, fmt.Errorf("adpilot campaigns: %w", err)
	}
	if f.Campaigns == nil {
		return nil, nil, errors.New("adpilot campaigns: missing campaigns")
	}
	return *f.Campaigns, f.SyncedAt, nil
}

// Metrics is CampaignMetrics: sums plus ratios that are null (never 0 or
// Inf) when their denominator is zero.
type Metrics struct {
	Spend          float64  `json:"spend"`
	Impressions    float64  `json:"impressions"`
	Clicks         float64  `json:"clicks"`
	Leads          float64  `json:"leads"`
	Bookings       float64  `json:"bookings"`
	Purchases      float64  `json:"purchases"`
	Revenue        float64  `json:"revenue"`
	CTR            *float64 `json:"ctr"`
	CPL            *float64 `json:"cpl"`
	CostPerBooking *float64 `json:"costPerBooking"`
	CostPerResult  *float64 `json:"costPerResult"`
	ROAS           *float64 `json:"roas"`
}

func ratio(num, den float64) *float64 {
	if den > 0 {
		v := num / den
		return &v
	}
	return nil
}

func derive(c Campaign) Metrics {
	result := c.Bookings // aggregate (no objective): per booking
	switch c.Objective {
	case "leads":
		result = c.Leads
	case "purchases":
		result = c.Purchases
	}
	return Metrics{
		Spend: c.Spend, Impressions: c.Impressions, Clicks: c.Clicks, Leads: c.Leads, Bookings: c.Bookings,
		Purchases: c.Purchases, Revenue: c.Revenue,
		CTR: ratio(c.Clicks, c.Impressions), CPL: ratio(c.Spend, c.Leads), CostPerBooking: ratio(c.Spend, c.Bookings),
		CostPerResult: ratio(c.Spend, result), ROAS: ratio(c.Revenue, c.Spend),
	}
}

// MetricsOf is one campaign's metrics, honouring its objective.
func MetricsOf(c Campaign) Metrics { return derive(c) }

// AggregateCampaigns sums first, then derives (never averages ratios).
func AggregateCampaigns(cs []Campaign) Metrics {
	var s Campaign
	for _, c := range cs {
		s.Spend += c.Spend
		s.Impressions += c.Impressions
		s.Clicks += c.Clicks
		s.Leads += c.Leads
		s.Bookings += c.Bookings
		s.Purchases += c.Purchases
		s.Revenue += c.Revenue
	}
	return derive(s)
}

// AggregateGeo merges same city+country rows, most leads first.
func AggregateGeo(cs []Campaign) []GeoPoint {
	idx := map[string]int{}
	out := []GeoPoint{}
	for _, c := range cs {
		for _, g := range c.Geo {
			k := g.City + "|" + g.Country
			if i, ok := idx[k]; ok {
				out[i].Leads += g.Leads
				out[i].Bookings += g.Bookings
				continue
			}
			idx[k] = len(out)
			out = append(out, g)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Leads > out[j].Leads })
	return out
}

// AggregateAudience is the spend-weighted merge of audience shares, rounded
// to one decimal.
func AggregateAudience(cs []Campaign) Audience {
	total := 0.0
	for _, c := range cs {
		total += c.Spend
	}
	if total == 0 {
		total = 1
	}
	merge := func(pick func(Campaign) map[string]float64) map[string]float64 {
		out := map[string]float64{}
		for _, c := range cs {
			w := c.Spend / total
			for k, v := range pick(c) {
				out[k] += v * w
			}
		}
		for k, v := range out {
			out[k] = math.Floor(v*10+0.5) / 10
		}
		return out
	}
	return Audience{
		Age:        merge(func(c Campaign) map[string]float64 { return c.Audience.Age }),
		Gender:     merge(func(c Campaign) map[string]float64 { return c.Audience.Gender }),
		Placements: merge(func(c Campaign) map[string]float64 { return c.Audience.Placements }),
	}
}

// AggregateDaily merges per-campaign dailies by date, oldest first.
func AggregateDaily(cs []Campaign) []DailyPoint {
	idx := map[string]int{}
	out := []DailyPoint{}
	for _, c := range cs {
		for _, d := range c.Daily {
			if i, ok := idx[d.Date]; ok {
				out[i].Spend += d.Spend
				out[i].Leads += d.Leads
				out[i].Bookings += d.Bookings
				continue
			}
			idx[d.Date] = len(out)
			out = append(out, d)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Date < out[j].Date })
	return out
}

// View is everything the deck's widgets read for one selection.
type View struct {
	Metrics  Metrics      `json:"metrics"`
	Geo      []GeoPoint   `json:"geo"`
	Audience Audience     `json:"audience"`
	Daily    []DailyPoint `json:"daily"`
}

// Deck is the campaign half of the page: the campaigns plus a precomputed
// view per selection ("all" and each campaign id). With no campaigns there
// are no views at all, so the page shows "No ad account connected" rather
// than a row of zeros.
type Deck struct {
	Campaigns []Campaign      `json:"campaigns"`
	SyncedAt  *string         `json:"syncedAt"`
	LiveCount int             `json:"liveCount"`
	Views     map[string]View `json:"views"`
}

func viewOf(cs []Campaign) View {
	if len(cs) == 1 {
		return View{Metrics: MetricsOf(cs[0]), Geo: AggregateGeo(cs), Audience: cs[0].Audience, Daily: AggregateDaily(cs)}
	}
	return View{Metrics: AggregateCampaigns(cs), Geo: AggregateGeo(cs), Audience: AggregateAudience(cs), Daily: AggregateDaily(cs)}
}

func BuildDeck(cs []Campaign, syncedAt *string) Deck {
	d := Deck{Campaigns: cs, SyncedAt: syncedAt}
	if d.Campaigns == nil {
		d.Campaigns = []Campaign{}
	}
	if len(cs) == 0 {
		return d
	}
	d.Views = map[string]View{"all": viewOf(cs)}
	for _, c := range cs {
		if c.Status == "active" {
			d.LiveCount++
		}
		d.Views[c.ID] = viewOf([]Campaign{c})
	}
	return d
}
