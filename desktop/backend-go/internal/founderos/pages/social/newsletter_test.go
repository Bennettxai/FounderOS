package social

import (
	"reflect"
	"testing"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/beehiiv"
	"github.com/rhl/businessos-backend/internal/founderos/pages/pagekit"
)

func issues() []beehiiv.Newsletter {
	return []beehiiv.Newsletter{
		{ID: "p2", Title: "Second", PublishedAt: "2026-07-14T15:00:00.000Z", Recipients: 1100, Delivered: 1020, Opens: 206, OpenRate: 20.2, Clicks: 20, Unsubscribes: 5},
		{ID: "p1", Title: "First", PublishedAt: "2026-07-07T15:00:00.000Z", Recipients: 1000, Delivered: 980, Opens: 294, OpenRate: 30, Clicks: 30, Unsubscribes: 5, SpamReports: 1},
	}
}

func TestNewsletterVolume(t *testing.T) {
	v := BuildNewsletterVolume(issues(), ip(5400))
	if *v.Headline != 5400 || !reflect.DeepEqual(v.Chips, []pagekit.Chip{{Tone: "accent", Text: "2 issues"}, {Tone: "ok", Text: "25.1% avg open"}}) || v.Caption != "subscribers, live via Beehiiv" {
		t.Fatalf("%+v", v)
	}
	disp := []string{}
	for _, m := range v.Meters {
		disp = append(disp, m.Label+"="+m.Display)
	}
	if !reflect.DeepEqual(disp, []string{"Delivered=95.2% of sends", "Opened=25.0% of delivered", "Clicked=10.0% of opens", "Unsubscribed=0.50% of delivered"}) {
		t.Fatal(disp)
	}
	frac(t, v.Meters[0], 2000.0/2100)
	if v.Foot != "2 issues · 2,100 sends" {
		t.Fatal(v.Foot)
	}
	if !reflect.DeepEqual(v.Sends, []pagekit.Point{{Label: "Jul 7", Count: 1000}, {Label: "Jul 14", Count: 1100}}) || !reflect.DeepEqual(v.OpenRates, []pagekit.Point{{Label: "Jul 7", Count: 30}, {Label: "Jul 14", Count: 20.2}}) {
		t.Fatalf("%+v %+v", v.Sends, v.OpenRates)
	}
	if v.TotalRecipients != 2100 || v.TotalClicks != 50 || v.Unsubscribes != 10 || v.SpamReports != 1 || *v.AvgOpenRate != 25.1 {
		t.Fatalf("%+v", v)
	}
	want := NewsletterInsight{Display: "30.0%", Headline: "Best open rate: First.", Body: "25.1% average across 2 issues · Jul 7", Frac: 0.3}
	if v.Insight != want {
		t.Fatalf("%+v", v.Insight)
	}
}

func TestNewsletterVolumeSeededAndEmpty(t *testing.T) {
	s := BuildNewsletterVolume([]beehiiv.Newsletter{{ID: "seed-1", PublishedAt: "2026-07-01T15:00:00.000Z", Recipients: 10, Delivered: 10}}, nil)
	if s.Headline != nil || s.Caption != "no live subscriber count · add BEEHIIV_API_KEY" || s.Foot != "1 issue · 10 sends · seeded preview" {
		t.Fatalf("%+v", s)
	}
	e := BuildNewsletterVolume(nil, nil)
	if len(e.Chips) != 0 || e.AvgOpenRate != nil || len(e.Meters) != 0 || len(e.Sends) != 0 || e.Foot != "no issues yet" || e.Insight.Display != "none" || e.Insight.Headline != "No issues sent yet." {
		t.Fatalf("%+v", e)
	}
	if len(SeedNewsletters) != 4 || SeedNewsletters[0].ID != "seed-4" {
		t.Fatal("seed")
	}
}

func TestNewsletterOpenRateMatrixKeepsSix(t *testing.T) {
	var many []beehiiv.Newsletter
	for i := 7; i >= 0; i-- {
		many = append(many, beehiiv.Newsletter{ID: "x", PublishedAt: "2026-08-" + pad2(10+i) + "T15:00:00.000Z", OpenRate: float64(20 + i)})
	}
	m := BuildNewsletterVolume(many, nil)
	if len(m.OpenRates) != 6 || m.OpenRates[5] != (pagekit.Point{Label: "Aug 17", Count: 27}) || len(m.Sends) != 8 {
		t.Fatalf("%+v", m.OpenRates)
	}
}

func pad2(n int) string { return string(rune('0'+n/10)) + string(rune('0'+n%10)) }

func TestNewsletterOpenRateColumnsNeverShareALabel(t *testing.T) {
	same := []beehiiv.Newsletter{
		{ID: "a", Title: "a", PublishedAt: "2026-09-03T10:00:00Z", Recipients: 10, Delivered: 10, Opens: 5, OpenRate: 50},
		{ID: "b", Title: "b", PublishedAt: "2026-09-03T18:00:00Z", Recipients: 10, Delivered: 10, Opens: 4, OpenRate: 40},
	}
	v := BuildNewsletterVolume(same, nil)
	if len(v.OpenRates) != 2 || v.OpenRates[0].Label == v.OpenRates[1].Label {
		t.Fatalf("open-rate labels collide: %+v", v.OpenRates)
	}
}
