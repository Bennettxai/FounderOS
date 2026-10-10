package adpilot

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// Ported from FounderOS v1 tests/adpilot.test.ts.

func campaign(id string, mod func(*Campaign)) Campaign {
	c := Campaign{
		ID: id, Name: id, Objective: "leads", Status: "active", Platform: "Meta",
		Period: Period{From: "2026-08-15", To: "2026-09-14"},
		Spend:  1000, Impressions: 100000, Clicks: 2000, Leads: 100, Bookings: 10, Purchases: 5, Revenue: 3000,
		Audience: Audience{
			Age:        map[string]float64{"25-34": 50, "35-44": 50},
			Gender:     map[string]float64{"male": 80, "female": 20},
			Placements: map[string]float64{"Reels": 100},
		},
		Geo: []GeoPoint{{City: "Austin", Country: "US", Lat: 30.27, Lng: -97.74, Leads: 100, Bookings: 10}},
	}
	if mod != nil {
		mod(&c)
	}
	return c
}

func eqp(t *testing.T, name string, got *float64, want float64) {
	t.Helper()
	if got == nil || *got != want {
		t.Fatalf("%s = %v, want %v", name, got, want)
	}
}

func TestMetricsDeriveEveryRatioAndHonorTheObjective(t *testing.T) {
	m := MetricsOf(campaign("a", nil))
	eqp(t, "cpl", m.CPL, 10)
	eqp(t, "costPerBooking", m.CostPerBooking, 100)
	eqp(t, "costPerResult", m.CostPerResult, 10) // objective=leads → per lead
	eqp(t, "roas", m.ROAS, 3)
	eqp(t, "ctr", m.CTR, 0.02)

	b := MetricsOf(campaign("b", func(c *Campaign) { c.Objective = "purchases" }))
	eqp(t, "costPerResult purchases", b.CostPerResult, 200)
}

func TestMetricsAreNullInsteadOfDividingByZero(t *testing.T) {
	m := MetricsOf(campaign("a", func(c *Campaign) { c.Leads, c.Bookings, c.Impressions, c.Spend = 0, 0, 0, 0 }))
	if m.CPL != nil || m.CTR != nil || m.ROAS != nil {
		t.Fatalf("want null ratios, got %+v", m)
	}
}

func TestAggregateSumsBeforeDeriving(t *testing.T) {
	m := AggregateCampaigns([]Campaign{
		campaign("a", func(c *Campaign) { c.Spend, c.Leads = 1000, 100 }),
		campaign("b", func(c *Campaign) { c.Spend, c.Leads = 3000, 100 }),
	})
	if m.Spend != 4000 {
		t.Fatalf("spend %v", m.Spend)
	}
	eqp(t, "cpl", m.CPL, 20) // 4000/200, not the mean of 10 and 30
	eqp(t, "costPerResult (aggregate: per booking)", m.CostPerResult, 200)
}

func TestAggregateGeoMergesCitiesAndSortsByLeads(t *testing.T) {
	merged := AggregateGeo([]Campaign{
		campaign("a", nil),
		campaign("b", func(c *Campaign) {
			c.Geo = []GeoPoint{
				{City: "Austin", Country: "US", Lat: 30.27, Lng: -97.74, Leads: 50, Bookings: 5},
				{City: "London", Country: "GB", Lat: 51.51, Lng: -0.13, Leads: 200, Bookings: 9},
			}
		}),
	})
	if merged[0].City != "London" || merged[1].City != "Austin" || merged[1].Leads != 150 || merged[1].Bookings != 15 {
		t.Fatalf("merged = %+v", merged)
	}
}

func TestAggregateAudienceWeightsBySpend(t *testing.T) {
	a := AggregateAudience([]Campaign{
		campaign("a", func(c *Campaign) {
			c.Spend = 1000
			c.Audience = Audience{Age: map[string]float64{"x": 100}, Gender: map[string]float64{"male": 100}, Placements: map[string]float64{"Reels": 100}}
		}),
		campaign("b", func(c *Campaign) {
			c.Spend = 3000
			c.Audience = Audience{Age: map[string]float64{"x": 0}, Gender: map[string]float64{"male": 100}, Placements: map[string]float64{"Reels": 100}}
		}),
	})
	if a.Age["x"] != 25 || a.Gender["male"] != 100 {
		t.Fatalf("audience = %+v", a)
	}
}

func TestAggregateDailyMergesByDateOldestFirst(t *testing.T) {
	d := AggregateDaily([]Campaign{
		campaign("a", func(c *Campaign) {
			c.Daily = []DailyPoint{{Date: "2026-09-02", Spend: 10, Leads: 1, Bookings: 0}, {Date: "2026-09-01", Spend: 5, Leads: 1, Bookings: 1}}
		}),
		campaign("b", func(c *Campaign) { c.Daily = []DailyPoint{{Date: "2026-09-02", Spend: 20, Leads: 2, Bookings: 1}} }),
	})
	if len(d) != 2 || d[0].Date != "2026-09-01" || d[1].Spend != 30 || d[1].Leads != 3 || d[1].Bookings != 1 {
		t.Fatalf("daily = %+v", d)
	}
}

func TestDeckViewsCoverAllAndEachCampaign(t *testing.T) {
	cs := []Campaign{campaign("a", nil), campaign("b", func(c *Campaign) { c.Status = "paused"; c.Spend = 3000 })}
	d := BuildDeck(cs, nil)
	if d.LiveCount != 1 || len(d.Views) != 3 {
		t.Fatalf("deck = %+v", d)
	}
	if d.Views["all"].Metrics.Spend != 4000 || d.Views["b"].Metrics.Spend != 3000 {
		t.Fatalf("views = %+v", d.Views)
	}
	// a single campaign keeps its own audience, not a re-weighted one
	if d.Views["a"].Audience.Age["25-34"] != 50 {
		t.Fatalf("single audience = %+v", d.Views["a"].Audience)
	}
}

func TestDeckWithNoCampaignsHasNoFakeTotals(t *testing.T) {
	d := BuildDeck(nil, nil)
	if d.Views != nil || d.LiveCount != 0 || d.Campaigns == nil {
		t.Fatalf("empty deck = %+v", d)
	}
	raw, _ := json.Marshal(d)
	if string(raw) != `{"campaigns":[],"syncedAt":null,"liveCount":0,"views":null}` {
		t.Fatalf("json = %s", raw)
	}
}

func TestReadCampaignFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "campaigns.json")
	raw, _ := json.Marshal(map[string]any{"campaigns": []Campaign{campaign("a", nil)}, "syncedAt": "2026-09-20T10:00:00Z"})
	if err := os.WriteFile(file, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	cs, synced, err := ReadCampaignFile(file)
	if err != nil || len(cs) != 1 || synced == nil || *synced != "2026-09-20T10:00:00Z" {
		t.Fatalf("got %v %v %v", cs, synced, err)
	}

	// absent is the live-repo state: no ad account connected, not an error
	cs, synced, err = ReadCampaignFile(filepath.Join(dir, "missing.json"))
	if err != nil || len(cs) != 0 || synced != nil {
		t.Fatalf("missing: %v %v %v", cs, synced, err)
	}

	// a corrupt or invalid file is surfaced, never read as empty
	for _, bad := range []string{`{`, `{"campaigns":[{"id":"x"}]}`, `{"campaigns":[{"id":"x","name":"x","objective":"reach","status":"active","platform":"Meta","period":{"from":"a","to":"b"},"spend":1,"impressions":1,"clicks":1,"leads":1,"bookings":1,"purchases":1,"revenue":1,"audience":{"age":{},"gender":{},"placements":{}},"geo":[]}]}`} {
		if err := os.WriteFile(file, []byte(bad), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, _, err := ReadCampaignFile(file); err == nil {
			t.Fatalf("want an error for %s", bad)
		}
	}
}

func TestCampaignDataPath(t *testing.T) {
	t.Setenv("ADPILOT_DATA_PATH", "/x/c.json")
	if CampaignDataPath() != "/x/c.json" {
		t.Fatal(CampaignDataPath())
	}
	t.Setenv("ADPILOT_DATA_PATH", "")
	if CampaignDataPath() != filepath.Join("data", "adpilot-campaigns.json") {
		t.Fatal(CampaignDataPath())
	}
}
