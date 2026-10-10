package social

import (
	"math"
	"reflect"
	"testing"
)

func snap(p, at string, f int) Snapshot {
	return Snapshot{Platform: p, CapturedAt: at, Followers: f, Source: "test"}
}
func email(at string, n int, src string) EmailSnapshot {
	return EmailSnapshot{CapturedAt: at, Subscribers: n, Source: src, Quality: "ok"}
}
func acct(p string, order int) Account { return Account{Platform: p, Handle: "@" + p, Order: order} }

func approx(t *testing.T, got *float64, want float64) {
	t.Helper()
	if got == nil || math.Abs(*got-want) > 1e-4 {
		t.Fatalf("got %v, want %v", deref(got), want)
	}
}
func deref(f *float64) any {
	if f == nil {
		return nil
	}
	return *f
}

func TestSnapshotGrowthWindows(t *testing.T) {
	g := SnapshotGrowth([]Snapshot{snap("instagram", "2026-06-01", 1000), snap("instagram", "2026-06-05", 1050), snap("instagram", "2026-06-12", 1155)})
	approx(t, g.D7, 10)
	if g.D30 != nil {
		t.Fatal("d30 must be null without history")
	}
	approx(t, g.AllTime, 15.5)
}

func TestSyncRowsTrackedPlatformsWithCountsOnly(t *testing.T) {
	f := func(n float64) *float64 { return &n }
	rows := SyncRows([]LiveAccount{
		{Platform: "instagram", Followers: f(40061)},
		{Platform: "tiktok", Followers: f(9945)},
		{Platform: "facebook", Followers: f(100)},
		{Platform: "linkedin"},
	}, "2026-06-13", "zernio-live")
	want := []Snapshot{
		{Platform: "instagram", CapturedAt: "2026-06-13", Followers: 40061, Source: "zernio-live"},
		{Platform: "tiktok", CapturedAt: "2026-06-13", Followers: 9945, Source: "zernio-live"},
	}
	if !reflect.DeepEqual(rows, want) {
		t.Fatalf("rows = %+v", rows)
	}
}

func fixture() *Data {
	return &Data{
		Accounts: []Account{acct("instagram", 1), acct("tiktok", 2), acct("linkedin", 5)},
		Snapshots: map[string][]Snapshot{
			"instagram": {snap("instagram", "2026-04-13", 36000), snap("instagram", "2026-06-12", 40000)},
			"tiktok":    {snap("tiktok", "2026-06-12", 9000)},
		},
		Email: []EmailSnapshot{email("2026-04-13", 27000, "beehiiv"), email("2026-06-12", 30000, "beehiiv")},
	}
}

func TestDashboardInAccountOrderWithHonestNulls(t *testing.T) {
	d := BuildDashboard(fixture())
	if len(d.Platforms) != 3 || d.Platforms[0].Platform != "instagram" || d.Platforms[2].Platform != "linkedin" {
		t.Fatalf("platforms %+v", d.Platforms)
	}
	if *d.Platforms[0].Followers != 40000 || d.Platforms[2].Followers != nil {
		t.Fatal("followers")
	}
	if d.TotalFollowers != 49000 || d.AsOf == nil || *d.AsOf != "2026-06-12" {
		t.Fatalf("total %v asOf %v", d.TotalFollowers, d.AsOf)
	}
	approx(t, d.Platforms[0].Growth.D60, (40000.0-36000)/36000*100)
}

func TestAudienceTotalsAndGrowth(t *testing.T) {
	x := fixture()
	x.Snapshots = map[string][]Snapshot{"instagram": x.Snapshots["instagram"]}
	if AudienceTotal(x) != 70000 {
		t.Fatalf("total %v", AudienceTotal(x))
	}
	approx(t, AudienceGrowthPct(x, 60), (70000.0-63000)/63000*100)
	g := AudienceGrowth(x)
	if g.D60 == nil {
		t.Fatal("d60")
	}
	all, channels := AudienceSeries(x)
	if all.Points[len(all.Points)-1] != (SeriesPoint{Date: "2026-06-12", Value: 70000}) {
		t.Fatalf("all %+v", all.Points)
	}
	keys := []string{}
	for _, c := range channels {
		keys = append(keys, c.Key)
	}
	if !reflect.DeepEqual(keys, []string{"instagram", "email"}) {
		t.Fatalf("channels %v", keys)
	}
}

func TestMonthlyAudienceGrowthQualifiesOnlyChannelsWithABaseline(t *testing.T) {
	x := &Data{Accounts: []Account{acct("instagram", 1)}, Snapshots: map[string][]Snapshot{}}
	if AudienceGrowthPct(x, 30) != nil {
		t.Fatal("empty must be null")
	}
	x.Email = []EmailSnapshot{email("2026-05-13", 28000, "beehiiv"), email("2026-06-12", 30000, "beehiiv")}
	x.Snapshots["instagram"] = []Snapshot{snap("instagram", "2026-06-12", 40000)}
	approx(t, AudienceGrowthPct(x, 30), (30000.0-28000)/28000*100)
	x.Snapshots["instagram"] = []Snapshot{snap("instagram", "2026-05-13", 38000), snap("instagram", "2026-06-12", 40000)}
	approx(t, AudienceGrowthPct(x, 30), (70000.0-66000)/66000*100)
}

// FounderOS v1 lib/email-list.ts: the seeded Beehiiv snapshots are the email
// list's history until a live reading lands, so they count like any other row.
func TestEmailListReadsEverySnapshotLikeV1(t *testing.T) {
	x := &Data{Email: []EmailSnapshot{
		email("2026-05-28", 1840, "seed-beehiiv"), email("2026-06-05", 1849, "seed-beehiiv"), email("2026-06-12", 1850, "seed-beehiiv"),
	}}
	e := BuildEmailList(x)
	if e.Subscribers == nil || *e.Subscribers != 1850 || *e.AsOf != "2026-06-12" || len(e.Series) != 3 {
		t.Fatalf("%+v", e)
	}
	approx(t, e.Growth.D7, (1850.0-1849)/1849*100)
	if e.Growth.D30 != nil {
		t.Fatal("d30 has no baseline that far back")
	}
	approx(t, e.Growth.AllTime, (1850.0-1840)/1840*100)
	empty := BuildEmailList(&Data{})
	if empty.Subscribers != nil || empty.Growth.D30 != nil {
		t.Fatal("empty list must be null")
	}
	if AudienceTotal(&Data{Email: x.Email}) != 1850 {
		t.Fatal("the seeded list counts toward reach, as in v1")
	}
}

func TestDMSeriesAndGrowth(t *testing.T) {
	x := &Data{DMSnapshots: []DMSnapshot{
		{Platform: "instagram", CapturedAt: "2026-05-13", Count: 1000},
		{Platform: "instagram", CapturedAt: "2026-06-12", Count: 1240},
		{Platform: "tiktok", CapturedAt: "2026-06-12", Count: 360},
	}, DMs: []DM{{Platform: "instagram", Count: 1240}, {Platform: "tiktok", Count: 360}}}
	s := DMSeries(x)
	if s[len(s)-1] != (SeriesPoint{Date: "2026-06-12", Value: 1600}) {
		t.Fatalf("%+v", s)
	}
	approx(t, DMGrowthPct(x, 30), 60)
	if TotalDMs(x) != 1600 {
		t.Fatal("total dms")
	}
}

func TestDMThreadsNewestFirstUnrepliedWhenLastIsInbound(t *testing.T) {
	h := "ava"
	x := &Data{DMMessages: []DMMessage{ // newest first, as the store returns
		{ID: "3", Platform: "instagram", SubscriberID: "a", Name: "Ava", Handle: &h, Text: "hi?", Direction: "in", TS: "2026-09-24T10:00:00.000Z"},
		{ID: "2", Platform: "instagram", SubscriberID: "b", Name: "Ben", Text: "thanks", Direction: "out", TS: "2026-09-23T10:00:00.000Z"},
		{ID: "1", Platform: "instagram", SubscriberID: "a", Name: "Ava", Handle: &h, Text: "hello", Direction: "out", TS: "2026-09-22T10:00:00.000Z"},
		{ID: "0", Platform: "tiktok", SubscriberID: "c", Name: "Cy", Text: "x", Direction: "in", TS: "2026-09-25T10:00:00.000Z"},
	}}
	th := DMThreads(x, "instagram")
	if len(th) != 2 || th[0].SubscriberID != "a" || !th[0].Unreplied || th[1].Unreplied {
		t.Fatalf("%+v", th)
	}
	if th[0].Messages[0].ID != "1" || th[0].Last.ID != "3" {
		t.Fatal("messages must be chronological")
	}
}

func TestPlatformDetail(t *testing.T) {
	x := fixture()
	d := PlatformDetail(x, "instagram")
	if d == nil || d.Account.Handle != "@instagram" || *d.Followers != 40000 || len(d.Snapshots) != 2 {
		t.Fatalf("%+v", d)
	}
	if PlatformDetail(x, "myspace") != nil || PlatformDetail(x, "twitter") != nil {
		t.Fatal("untracked platforms are nil")
	}
	if l := PlatformDetail(x, "linkedin"); l == nil || l.Followers != nil {
		t.Fatal("linkedin without snapshots reads null followers")
	}
}
