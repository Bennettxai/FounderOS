package social

import (
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/rhl/businessos-backend/internal/founderos/pages/pagekit"
)

func ip(n int) *int         { return &n }
func fp(f float64) *float64 { return &f }

func frac(t *testing.T, m pagekit.Meter, want float64) {
	t.Helper()
	if m.Frac == nil || math.Abs(*m.Frac-want) > 1e-9 {
		t.Fatalf("%s frac = %v, want %v", m.Label, deref(m.Frac), want)
	}
}

func socialFixture() SocialVolume {
	return BuildSocialVolume(SocialVolumeInput{
		Channels: []Channel{
			{Key: "instagram", Label: "Instagram", Value: ip(6000)},
			{Key: "tiktok", Label: "TikTok", Value: ip(2000)},
			{Key: "twitter", Label: "X", Value: ip(500)},
			{Key: "youtube", Label: "YouTube", Value: ip(1000)},
			{Key: "linkedin", Label: "LinkedIn"},
			{Key: "email", Label: "Email list", Value: ip(500)},
		},
		Total:   10000,
		Growth7: fp(1.234),
		Leader:  "TikTok",
		Threads: []ThreadState{{Name: "Ava", Unreplied: true}, {Name: "Ben"}, {Name: "Cy", Unreplied: true}, {Name: "Dee"}},
		Queued:  2,
		PostDays: []PostDay{
			{Date: "2026-09-24", Platforms: []string{"instagram", "tiktok"}},
			{Date: "2026-09-24", Platforms: []string{"instagram"}},
			{Date: "2026-09-20", Platforms: []string{"twitter", "mastodon"}},
			{Date: "2026-07-01", Platforms: []string{"youtube"}},
		},
		Today: "2026-09-24",
	})
}

func TestSocialVolumeHeadlineChipsCaption(t *testing.T) {
	v := socialFixture()
	if v.Headline != 10000 {
		t.Fatal(v.Headline)
	}
	// FounderOS v1 lib/social-volume.ts: growth, the owed DM replies, queued.
	want := []pagekit.Chip{{Tone: "ok", Text: "+1.23% 7d"}, {Tone: "warn", Text: "2 need reply"}, {Tone: "accent", Text: "2 queued"}}
	if !reflect.DeepEqual(v.Chips, want) {
		t.Fatalf("chips %+v", v.Chips)
	}
	if v.Caption != "across 5 live channels · TikTok leads 7-day growth" {
		t.Fatal(v.Caption)
	}
}

func TestSocialVolumeMetersAreTheBiggestFourChannels(t *testing.T) {
	v := socialFixture()
	labels := []string{}
	for _, m := range v.Meters {
		labels = append(labels, m.Label)
	}
	if !reflect.DeepEqual(labels, []string{"Instagram", "TikTok", "YouTube", "X"}) {
		t.Fatal(labels)
	}
	frac(t, v.Meters[0], 0.6)
	if v.Meters[0].Display != "6,000 · 60%" || v.Meters[0].Hue != pagekit.Ramp1 || v.Meters[3].Display != "500 · 5%" || v.Meters[3].Hue != pagekit.Ramp4 {
		t.Fatalf("%+v", v.Meters)
	}
	if v.Foot != "share of total reach · 2 smaller channels" {
		t.Fatal(v.Foot)
	}
}

func TestSocialVolumePostingSeriesAndMix(t *testing.T) {
	v := socialFixture()
	if len(v.Series) != 30 || v.Series[29] != (pagekit.Point{Label: "Sep 24", Count: 2}) || v.Series[0].Label != "Aug 26" || v.PostsInWindow != 3 {
		t.Fatalf("%+v", v.Series)
	}
	for _, p := range v.Series {
		if p.Label == "Sep 20" && p.Count != 1 {
			t.Fatal("sep 20")
		}
	}
	want := []pagekit.Point{{Label: "IG", Count: 2}, {Label: "TT", Count: 1}, {Label: "X", Count: 1}, {Label: "YT", Count: 0}, {Label: "LI", Count: 0}}
	if !reflect.DeepEqual(v.Mix, want) {
		t.Fatalf("%+v", v.Mix)
	}
}

func TestSocialVolumeInsightIsDMsOwed(t *testing.T) {
	v := socialFixture()
	if v.Insight.Value != 2 || v.Insight.Headline != "2 of 4 Instagram threads need a reply." || v.Insight.Body != "Ava · Cy" || v.Insight.Frac != 0.5 {
		t.Fatalf("%+v", v.Insight)
	}
	d := BuildSocialVolume(SocialVolumeInput{Channels: socialFixtureChannels(), Total: 10000, Growth7: fp(-12.5), Today: "2026-09-24"})
	if !reflect.DeepEqual(d.Chips, []pagekit.Chip{{Tone: "err", Text: "-12.5% 7d"}}) || d.Caption != "across 5 live channels" {
		t.Fatalf("%+v %s", d.Chips, d.Caption)
	}
}

func socialFixtureChannels() []Channel {
	return []Channel{{Key: "instagram", Label: "Instagram", Value: ip(6000)}, {Key: "tiktok", Label: "TikTok", Value: ip(2000)}, {Key: "twitter", Label: "X", Value: ip(500)}, {Key: "youtube", Label: "YouTube", Value: ip(1000)}, {Key: "linkedin", Label: "LinkedIn"}, {Key: "email", Label: "Email list", Value: ip(500)}}
}

func TestSocialVolumeEmptyAndOffline(t *testing.T) {
	e := BuildSocialVolume(SocialVolumeInput{Today: "2026-09-24"})
	if e.Headline != 0 || len(e.Chips) != 0 || len(e.Meters) != 0 || e.Caption != "no channels reporting yet" {
		t.Fatalf("%+v", e)
	}
	if e.Insight.Headline != "Inbox clear." || e.Insight.Body != "No Instagram threads yet." || e.Insight.Frac != 0 {
		t.Fatalf("%+v", e.Insight)
	}
	// Bridge rule: a channel with no reading is unknown (null frac), not 0.
	o := BuildSocialVolume(SocialVolumeInput{Channels: []Channel{{Key: "linkedin", Label: "LinkedIn"}}, Today: "2026-09-24"})
	if len(o.Meters) != 1 || o.Meters[0].Frac != nil || o.Meters[0].Display != "offline" {
		t.Fatalf("%+v", o.Meters)
	}
}

var platSnaps = []Snapshot{
	{CapturedAt: "2026-06-01", Followers: 1500, Source: "zernio-config"},
	{CapturedAt: "2026-08-20", Followers: 900, Source: "zernio-config"},
	{CapturedAt: "2026-08-26", Followers: 1000, Source: "zernio-config"},
	{CapturedAt: "2026-09-10", Followers: 1100, Source: "zernio-config"},
	{CapturedAt: "2026-09-20", Followers: 1050, Source: "zernio-config"},
	{CapturedAt: "2026-09-23", Followers: 1050, Source: "zernio-config"},
	{CapturedAt: "2026-09-24", Followers: 1250, Source: "zernio-live"},
}

func TestPlatformVolume(t *testing.T) {
	v := BuildPlatformVolume("Instagram", ip(1250), Growth{D7: fp(1.234), D30: fp(-2.5), D60: fp(4), AllTime: fp(-16.7)}, platSnaps, "2026-09-24", 30)
	if *v.Headline != 1250 || !reflect.DeepEqual(v.Chips, []pagekit.Chip{{Tone: "ok", Text: "+1.23% 7d"}, {Tone: "err", Text: "-2.50% 30d"}}) {
		t.Fatalf("%+v", v.Chips)
	}
	if v.Caption != "Instagram followers · 7 snapshots since Jun 1" {
		t.Fatal(v.Caption)
	}
	wantLabels := []string{"Days gained (3/5)", "Days dipped (1/5)", "Tracked days (5/30)", "Of all-time peak"}
	for i, m := range v.Meters {
		if m.Label != wantLabels[i] {
			t.Fatalf("%+v", v.Meters)
		}
	}
	frac(t, v.Meters[0], 0.6)
	frac(t, v.Meters[2], 5.0/30)
	if v.Meters[2].Display != "17%" || v.Meters[3].Display != "1,250 of 1,500" || v.Meters[0].Hue != pagekit.OK || v.Meters[3].Hue != pagekit.Accent {
		t.Fatalf("%+v", v.Meters)
	}
	if v.Foot != "7 snapshots · latest Sep 24 · zernio-live" {
		t.Fatal(v.Foot)
	}
	if len(v.Series) != 30 || v.Series[0] != (pagekit.Point{Label: "Aug 26", Count: 100}) || v.Series[29] != (pagekit.Point{Label: "Sep 24", Count: 200}) || v.GainedDays != 3 || v.Intervals != 5 || *v.Net != 350 {
		t.Fatalf("%+v", v.Series)
	}
	want := PlatformInsight{Value: 350, Display: "+350", Headline: "Gained 350 followers in the last 30 days.", Body: "up on 3 of 5 tracked days · best day Sep 24 (+200)", Frac: 0.6}
	if v.Insight != want {
		t.Fatalf("%+v", v.Insight)
	}
	loss := BuildPlatformVolume("X", ip(950), Growth{}, []Snapshot{{CapturedAt: "2026-09-22", Followers: 1000, Source: "s"}, {CapturedAt: "2026-09-24", Followers: 950, Source: "s"}}, "2026-09-24", 30)
	if len(loss.Chips) != 0 || loss.Insight.Display != "-50" || loss.Insight.Headline != "Lost 50 followers in the last 30 days." || loss.Insight.Body != "up on 0 of 1 tracked days" {
		t.Fatalf("%+v", loss.Insight)
	}
}

func TestPlatformVolumeThin(t *testing.T) {
	e := BuildPlatformVolume("LinkedIn", nil, Growth{}, nil, "2026-09-24", 30)
	if e.Headline != nil || e.Caption != "no follower reading yet" || len(e.Meters) != 0 || e.Foot != "no snapshots yet" || e.Net != nil || e.Insight.Display != "none" || len(e.Series) != 30 {
		t.Fatalf("%+v", e)
	}
	one := BuildPlatformVolume("TikTok", ip(400), Growth{}, []Snapshot{{CapturedAt: "2026-09-24", Followers: 400, Source: "s"}}, "2026-09-24", 30)
	if len(one.Meters) != 2 || one.Meters[0].Label != "Tracked days (1/30)" || one.Meters[1].Label != "Of all-time peak" || one.Intervals != 0 {
		t.Fatalf("%+v", one.Meters)
	}
}

// Seeded demo threads (source seed-dummy, until the ManyChat webhook feeds the
// inbox) read exactly as FounderOS v1 reads them: the demo world is seeded
// throughout, so the copy carries no extra label.
func TestSocialVolumeDemoThreadsReadAsV1(t *testing.T) {
	v := BuildSocialVolume(SocialVolumeInput{Channels: socialFixtureChannels(), Total: 10000, Today: "2026-09-24",
		Threads: []ThreadState{{Name: "Sam", Unreplied: true, Demo: true}, {Name: "Alex", Unreplied: true, Demo: true}}})
	if v.Insight.Headline != "2 of 2 Instagram threads need a reply." {
		t.Fatalf("headline %q", v.Insight.Headline)
	}
	found := false
	for _, c := range v.Chips {
		if c.Text == "2 need reply" && c.Tone == "warn" {
			found = true
		}
		if strings.Contains(c.Text, "demo") {
			t.Fatalf("chips %+v", v.Chips)
		}
	}
	if !found {
		t.Fatalf("chips %+v", v.Chips)
	}
}

func TestDMThreadsMarkAllSeededThreadsDemo(t *testing.T) {
	ts := DMThreads(&Data{DMMessages: []DMMessage{
		{Platform: "instagram", SubscriberID: "a", Name: "Sam", Direction: "in", TS: "2026-07-18T12:00:00Z", Source: "seed-dummy"},
		{Platform: "instagram", SubscriberID: "b", Name: "Real", Direction: "in", TS: "2026-09-29T12:00:00Z", Source: "manychat"},
	}}, "instagram")
	demo := map[string]bool{}
	for _, t := range ts {
		demo[t.Name] = t.Demo
	}
	if !demo["Sam"] || demo["Real"] {
		t.Fatalf("demo = %v", demo)
	}
}
