package adpilot

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/foreplay"
)

const day = 86400.0

func sp(s string) *string { return &s }
func bp(b bool) *bool     { return &b }

// ad builds a Foreplay ad from JSON the way the API/store deliver it.
func ad(t *testing.T, js string) foreplay.Ad {
	t.Helper()
	var a foreplay.Ad
	if err := json.Unmarshal([]byte(js), &a); err != nil {
		t.Fatal(err)
	}
	return a
}

func mk(id string, live bool, days float64, sentence string) foreplay.Ad {
	a := foreplay.Ad{ID: id, Live: bp(live), RunningDuration: &foreplay.RunningDuration{Seconds: days * day}}
	if sentence != "" {
		a.TimestampedTranscription = []foreplay.TranscriptLine{{StartTime: 0, Sentence: sentence}}
	}
	return a
}

func TestDaysRunningFloorsAndDefaultsToZero(t *testing.T) {
	if DaysRunning(foreplay.Ad{}) != 0 || DaysRunning(mk("a", true, 21.9, "")) != 21 {
		t.Fatal("days running")
	}
}

func TestHookLine(t *testing.T) {
	cases := []struct {
		ad   foreplay.Ad
		want *string
	}{
		{mk("a", true, 1, "  If your front desk misses calls  "), sp("If your front desk misses calls")},
		{mk("b", true, 1, "Thanks for watching!"), nil},
		{mk("c", true, 1, "short"), nil},
		{mk("d", true, 1, "..."), nil},
		{foreplay.Ad{ID: "e", FullTranscription: sp("Stop losing leads after hours. Book now")}, sp("Stop losing leads after hours")},
		{foreplay.Ad{ID: "f", TimestampedTranscription: []foreplay.TranscriptLine{{Sentence: "  "}}, FullTranscription: sp("Your agency needs this! yes")}, sp("Your agency needs this")},
		{foreplay.Ad{ID: "g"}, nil},
	}
	for _, c := range cases {
		got := HookLine(c.ad)
		if (got == nil) != (c.want == nil) || (got != nil && *got != *c.want) {
			t.Fatalf("%s: got %v want %v", c.ad.ID, got, c.want)
		}
	}
}

func TestBestHookFallsBackToTheTextOpener(t *testing.T) {
	a := foreplay.Ad{ID: "x", Description: sp("\n  It&#39;s &quot;time&quot; &amp; more<br/>second")}
	h, src := BestHook(a)
	if h == nil || *h != `It's "time" & more` || src == nil || *src != "text" {
		t.Fatalf("got %v %v", h, src)
	}
	h, src = BestHook(mk("y", true, 1, "Spoken hook line here"))
	if *h != "Spoken hook line here" || *src != "spoken" {
		t.Fatal("spoken")
	}
	h, src = BestHook(foreplay.Ad{ID: "z", Description: sp("tiny")})
	if h != nil || src != nil {
		t.Fatal("no hook")
	}
}

func TestToWallAdFlattensForTheUI(t *testing.T) {
	a := ad(t, `{"id":"ad_1","name":"Vantage page","brand_id":"br_1","live":true,"running_duration":{"seconds":2592000},
		"timestamped_transcription":[{"startTime":0,"endTime":2,"sentence":" If your front desk misses calls "},{"startTime":2,"sentence":"  "}],
		"emotional_drivers":{"urgency":8},"display_format":"video","thumbnail":"t.jpg","video":"v.mp4","image":null,"cta_type":"LEARN_MORE","link_url":"https://m"}`)
	w := ToWallAd(a, "Vantage")
	if w.Brand != "Vantage" || *w.BrandID != "br_1" || !w.Live || w.DaysRunning != 30 || *w.Hook != "If your front desk misses calls" ||
		*w.HookSource != "spoken" || len(w.Transcript) != 1 || w.Transcript[0].S != "If your front desk misses calls" || w.Drivers["urgency"] != 8 ||
		w.Image != nil || *w.CTAType != "LEARN_MORE" {
		t.Fatalf("wall = %+v", w)
	}
	if ToWallAd(a, "").Brand != "Vantage page" || ToWallAd(foreplay.Ad{ID: "q"}, "").Brand != "unknown" {
		t.Fatal("brand fallback")
	}
	raw, _ := json.Marshal(ToWallAd(foreplay.Ad{ID: "q"}, ""))
	for _, k := range []string{`"brandId":null`, `"hook":null`, `"hookSource":null`, `"transcript":[]`, `"drivers":{}`, `"live":false`} {
		if !strings.Contains(string(raw), k) {
			t.Fatalf("%s missing %s", raw, k)
		}
	}
}

func TestDiffBrandAds(t *testing.T) {
	at := "2026-09-30T00:00:00Z"
	prev := []foreplay.Ad{mk("old", true, 20, "Old winner hook here"), mk("dying", true, 40, "Dying ad hook line")}
	next := []foreplay.Ad{
		mk("old", true, 21, "Old winner hook here"),     // crosses 21
		mk("dying", false, 45, "Dying ad hook line"),    // killed
		mk("young", true, 3, "Brand new launch hook"),   // new launch
		mk("catalog", true, 90, "Back catalog ad hook"), // first sight but old: not a launch
	}
	sig := DiffBrandAds("br", "Acme", prev, next, at)
	if len(sig) != 3 {
		t.Fatalf("signals = %+v", sig)
	}
	if sig[0].Type != "winner" || sig[0].Message != `Acme: "Old winner hook here" crossed 21 days and is still live` {
		t.Fatalf("winner %+v", sig[0])
	}
	if sig[1].Type != "killed" || sig[1].Message != `Acme killed "Dying ad hook line" after 45 days` {
		t.Fatalf("killed %+v", sig[1])
	}
	if sig[2].Type != "new_launch" || sig[2].AdID == nil || *sig[2].AdID != "young" || sig[2].At != at {
		t.Fatalf("launch %+v", sig[2])
	}
	// label fallbacks: empty headline falls through to the format, then "ad"
	h := foreplay.Ad{ID: "h", Live: bp(true), Headline: sp("  "), DisplayFormat: sp("image")}
	if s := DiffBrandAds("b", "B", nil, []foreplay.Ad{h}, at); s[0].Message != "B launched image" {
		t.Fatal(s[0].Message)
	}
}

func TestVelocitySignal(t *testing.T) {
	series := func(counts ...float64) []foreplay.AnalyticsDay {
		var out []foreplay.AnalyticsDay
		for i, c := range counts {
			out = append(out, foreplay.AnalyticsDay{Date: "2026-09-" + string(rune('1'+i/10)) + string(rune('0'+i%10)), ActiveCount: c})
		}
		return out
	}
	if VelocitySignal("b", "B", series(10, 10, 10), "at") != nil {
		t.Fatal("needs 4 days")
	}
	if VelocitySignal("b", "B", series(2, 2, 2, 9), "at") != nil {
		t.Fatal("tiny base")
	}
	if VelocitySignal("b", "B", series(10, 10, 10, 12), "at") != nil {
		t.Fatal("under 30%")
	}
	s := VelocitySignal("b", "B", series(10, 10, 10, 10, 15), "at")
	if s == nil || s.Type != "velocity_spike" || s.Message != "B active ads +50% vs 7-day mean (15 live on 2026-09-14)" {
		t.Fatalf("spike %+v", s)
	}
	d := VelocitySignal("b", "B", series(10, 10, 10, 10, 5), "at")
	if d == nil || d.Type != "velocity_drop" || !strings.Contains(d.Message, "-50%") {
		t.Fatalf("drop %+v", d)
	}
}

func TestDigestAndNaiveProbes(t *testing.T) {
	if Digest(nil) != "No changes across the watchlist." {
		t.Fatal("empty digest")
	}
	if Digest([]Signal{{Message: "a"}, {Message: "b"}}) != "a · b" {
		t.Fatal("digest")
	}
	p := NaiveProbes("  founders replacing staff with AI agents ")
	if len(p) != 3 || p[0] != "founders replacing staff with AI agents" || p[1] != "founders replacing staff" || p[2] != "staff with agents" {
		t.Fatalf("probes = %q", p)
	}
	if p := NaiveProbes("ai ads"); len(p) != 1 {
		t.Fatalf("short = %q", p)
	}
}

// ---- store ------------------------------------------------------------------

func TestStoreWallRanksLiveFirstThenLongevity(t *testing.T) {
	s := Store{Dir: t.TempDir()}
	now := time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)
	if _, err := s.AddWatchEntry(WatchEntry{ID: "br_1", Name: "Acme"}, now); err != nil {
		t.Fatal(err)
	}
	if err := s.WriteBrandAds("br_1", []foreplay.Ad{mk("dead", false, 200, ""), mk("a", true, 10, ""), mk("b", true, 50, "")}); err != nil {
		t.Fatal(err)
	}
	w, err := s.Wall(2)
	if err != nil || len(w) != 2 || w[0].ID != "b" || w[1].ID != "a" || w[0].Brand != "Acme" {
		t.Fatalf("wall = %+v %v", w, err)
	}
}

func TestStoreEmptyIsEmptyAndCorruptIsAnError(t *testing.T) {
	s := Store{Dir: filepath.Join(t.TempDir(), "never-synced")}
	w, err := s.Wall(80)
	sig, err2 := s.ReadSignals()
	saved, err3 := s.ReadSaved()
	if err != nil || err2 != nil || err3 != nil || len(w) != 0 || len(sig) != 0 || len(saved) != 0 {
		t.Fatal("an unsynced store is empty, not an error")
	}
	if err := os.MkdirAll(s.Dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"watchlist.json", "signals.json", "saved.json"} {
		_ = os.WriteFile(filepath.Join(s.Dir, f), []byte("{nope"), 0o644)
	}
	if _, err := s.Wall(80); err == nil {
		t.Fatal("corrupt watchlist must surface")
	}
	if _, err := s.ReadSignals(); err == nil {
		t.Fatal("corrupt signals must surface")
	}
	if _, err := s.ReadSaved(); err == nil {
		t.Fatal("corrupt saved must surface")
	}
}

func TestWatchlistSkipsInvalidRowsAddsOnceAndRemoves(t *testing.T) {
	s := Store{Dir: t.TempDir()}
	_ = os.WriteFile(filepath.Join(s.Dir, "watchlist.json"), []byte(`[{"id":"","name":"x","addedAt":"t"},{"id":"br_1","name":"Acme","addedAt":"t"},{"name":"no id"}]`), 0o644)
	list, err := s.ReadWatchEntries()
	if err != nil || len(list) != 1 || list[0].ID != "br_1" {
		t.Fatalf("list = %+v %v", list, err)
	}
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	list, _ = s.AddWatchEntry(WatchEntry{ID: "br_1", Name: "dupe"}, now)
	if len(list) != 1 || list[0].Name != "Acme" {
		t.Fatal("duplicate id must not add")
	}
	list, _ = s.AddWatchEntry(WatchEntry{ID: "br_2", Name: "Beta", Avatar: sp("a.png")}, now)
	if len(list) != 2 || list[1].AddedAt != "2026-09-30T12:00:00.000Z" {
		t.Fatalf("added = %+v", list)
	}
	list, _ = s.RemoveWatchEntry("br_1")
	if len(list) != 1 || list[0].ID != "br_2" {
		t.Fatalf("removed = %+v", list)
	}
}

func TestSavedAdsNewestFirstNoDupesCapped(t *testing.T) {
	s := Store{Dir: t.TempDir()}
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	w1, w2 := ToWallAd(mk("a", true, 1, ""), "A"), ToWallAd(mk("b", true, 1, ""), "B")
	_, _ = s.SaveAd(w1, now)
	list, _ := s.SaveAd(w2, now)
	list2, _ := s.SaveAd(w1, now)
	if len(list) != 2 || list[0].Ad.ID != "b" || len(list2) != 2 {
		t.Fatalf("saved = %+v", list)
	}
	list, _ = s.UnsaveAd("b")
	if len(list) != 1 || list[0].Ad.ID != "a" || list[0].SavedAt != "2026-09-30T12:00:00.000Z" {
		t.Fatalf("unsaved = %+v", list)
	}
	for i := 0; i < 205; i++ {
		_, _ = s.SaveAd(ToWallAd(mk("x"+strings.Repeat("y", i), true, 1, ""), "X"), now)
	}
	list, _ = s.ReadSaved()
	if len(list) != 200 {
		t.Fatalf("cap: %d", len(list))
	}
}

func TestAppendSignalsNewestFirstCapped(t *testing.T) {
	s := Store{Dir: t.TempDir()}
	_ = s.AppendSignals([]Signal{{Type: "winner", Message: "old"}}, 2)
	_ = s.AppendSignals(nil, 2) // no-op, no file churn
	_ = s.AppendSignals([]Signal{{Type: "killed", Message: "new1"}, {Type: "killed", Message: "new2"}}, 2)
	got, _ := s.ReadSignals()
	if len(got) != 2 || got[0].Message != "new1" || got[1].Message != "new2" {
		t.Fatalf("signals = %+v", got)
	}
}

func TestWriteAnalyticsMergesByDate(t *testing.T) {
	s := Store{Dir: t.TempDir()}
	_ = s.WriteAnalytics("br", []foreplay.AnalyticsDay{{Date: "2026-09-02", ActiveCount: 2}, {Date: "2026-09-01", ActiveCount: 1}})
	_ = s.WriteAnalytics("br", []foreplay.AnalyticsDay{{Date: "2026-09-02", ActiveCount: 5}, {Date: "2026-09-03", ActiveCount: 3}})
	got, err := s.ReadAnalytics("br")
	if err != nil || len(got) != 3 || got[0].Date != "2026-09-01" || got[1].ActiveCount != 5 {
		t.Fatalf("analytics = %+v %v", got, err)
	}
}

func TestBrandAdsRoundTripKeepsUnknownFields(t *testing.T) {
	s := Store{Dir: t.TempDir()}
	a := ad(t, `{"id":"ad_1","live":true,"extra_field":"kept"}`)
	a.Live = bp(false)
	if err := s.WriteBrandAds("br/1", []foreplay.Ad{a}); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(filepath.Join(s.Dir, "ads-br_1.json"))
	if !strings.Contains(string(raw), `"extra_field": "kept"`) || !strings.Contains(string(raw), `"live": false`) {
		t.Fatalf("store file = %s", raw)
	}
}

// ---- API cycles (no network: a fake Foreplay) -----------------------------

type fakeAPI struct {
	calls      int64
	brandAds   map[string][]foreplay.Ad // "<id>|<order>"
	analytics  map[string][]foreplay.AnalyticsDay
	domains    map[string][]foreplay.SpyderBrand
	discovery  map[string][]foreplay.Ad
	failProbe  string
	usage      *foreplay.Usage
	usageErr   error
	seenFilter []foreplay.AdFilters
}

func (f *fakeAPI) CallCount() int64 { return f.calls }
func (f *fakeAPI) AdsByBrandID(_ context.Context, ids string, flt foreplay.AdFilters) ([]foreplay.Ad, error) {
	f.calls++
	f.seenFilter = append(f.seenFilter, flt)
	return f.brandAds[ids+"|"+flt.Order], nil
}
func (f *fakeAPI) BrandAnalytics(_ context.Context, id, _, _ string) ([]foreplay.AnalyticsDay, error) {
	f.calls++
	return f.analytics[id], nil
}
func (f *fakeAPI) Usage(context.Context) (*foreplay.Usage, error) {
	f.calls++
	return f.usage, f.usageErr
}
func (f *fakeAPI) BrandsByDomain(_ context.Context, domain string, _ int) ([]foreplay.SpyderBrand, error) {
	f.calls++
	return f.domains[domain], nil
}
func (f *fakeAPI) DiscoveryAds(_ context.Context, q string, flt foreplay.AdFilters) ([]foreplay.Ad, error) {
	f.calls++
	f.seenFilter = append(f.seenFilter, flt)
	if q == f.failProbe {
		return nil, errors.New("boom")
	}
	return f.discovery[q], nil
}

func TestRunSyncCycle(t *testing.T) {
	s := Store{Dir: t.TempDir()}
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	_, _ = s.AddWatchEntry(WatchEntry{ID: "br_1", Name: "Acme"}, now)
	_ = s.WriteBrandAds("br_1", []foreplay.Ad{mk("gone", true, 30, "This one got switched off"), mk("keep", true, 20, "Still running strong here")})
	api := &fakeAPI{
		brandAds: map[string][]foreplay.Ad{
			"br_1|":       {mk("keep", true, 21, "Still running strong here"), mk("fresh", true, 2, "A brand new launch today")},
			"br_1|oldest": {mk("keep", true, 21, "Still running strong here")},
		},
		analytics: map[string][]foreplay.AnalyticsDay{"br_1": {{Date: "2026-09-29", ActiveCount: 4}}},
		usage:     &foreplay.Usage{TotalCredits: 10000, RemainingCredits: 9640},
	}
	res, err := RunSyncCycle(context.Background(), api, s, now)
	if err != nil {
		t.Fatal(err)
	}
	if res.Brands != 1 || res.AdsSeen != 2 || res.APICalls != 4 || res.RemainingCredits == nil || *res.RemainingCredits != 9640 {
		t.Fatalf("result = %+v", res)
	}
	types := map[string]bool{}
	for _, sg := range res.Signals {
		types[sg.Type] = true
	}
	if !types["winner"] || !types["killed"] || !types["new_launch"] || len(res.Signals) != 3 {
		t.Fatalf("signals = %+v", res.Signals)
	}
	if f := api.seenFilter[0]; f.Live == nil || !*f.Live || f.Limit == nil || *f.Limit != 75 {
		t.Fatalf("filter = %+v", f)
	}
	ads, _ := s.ReadBrandAds("br_1")
	if len(ads) != 3 {
		t.Fatalf("store keeps dead ads for history: %d", len(ads))
	}
	for _, a := range ads {
		if a.ID == "gone" && (a.Live == nil || *a.Live) {
			t.Fatal("the vanished ad is stored as not live")
		}
	}
	meta, _ := s.Foreplay().ReadMeta()
	if meta.LastSyncAt == nil || *meta.LastSyncAt != "2026-09-30T12:00:00.000Z" || meta.LastSyncCalls != 4 {
		t.Fatalf("meta = %+v", meta)
	}
	u, _ := s.Foreplay().ReadUsage()
	if u == nil || u.RemainingCredits != 9640 {
		t.Fatal("usage stored")
	}
	sig, _ := s.ReadSignals()
	if len(sig) != 3 || !strings.Contains(res.Digest, " · ") {
		t.Fatal("signals appended, digest joined")
	}

	// a failed usage read never fails the cycle, and credits stay unknown
	api.usage, api.usageErr = nil, errors.New("down")
	res, err = RunSyncCycle(context.Background(), api, s, now)
	if err != nil || res.RemainingCredits != nil {
		t.Fatalf("usage failure: %+v %v", res, err)
	}
}

func TestMineConcept(t *testing.T) {
	api := &fakeAPI{
		failProbe: "broken probe",
		discovery: map[string][]foreplay.Ad{
			"ai agents": {mk("a", true, 40, "AI agents replace your staff"), mk("b", true, 90, "Hire AI not people today")},
			"ai staff":  {mk("a", true, 40, "AI agents replace your staff")},
		},
	}
	res := MineConcept(context.Background(), api, "ai employees", []string{"ai agents", "ai staff", "broken probe"}, MineOpts{Format: "video"})
	if res.Pooled != 2 || res.APICalls != 3 || len(res.Probes) != 3 || res.Probes[2].Results != 0 {
		t.Fatalf("mine = %+v", res)
	}
	if res.Winners[0].Ad.ID != "b" || len(res.Winners[1].Probes) != 2 {
		t.Fatalf("winners = %+v", res.Winners)
	}
	f := api.seenFilter[0]
	if *f.RunningDurationMinDays != 21 || *f.Limit != 15 || !*f.Live || f.DisplayFormat != "video" {
		t.Fatalf("filters = %+v", f)
	}
}

func TestAddWatchDomainPicksTheBiggestPage(t *testing.T) {
	s := Store{Dir: t.TempDir()}
	n1, n2 := 5, 40
	api := &fakeAPI{domains: map[string][]foreplay.SpyderBrand{"skool.com": {{ID: "small", Name: "Skool mini", AdsCount: &n1}, {ID: "big", Name: "Skool", AdsCount: &n2}}}}
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	e, err := AddWatchDomain(context.Background(), api, s, " HTTPS://Skool.com/about ", now)
	if err != nil || e == nil || e.ID != "big" || *e.Domain != "skool.com" {
		t.Fatalf("entry = %+v %v", e, err)
	}
	list, _ := s.ReadWatchEntries()
	if len(list) != 1 {
		t.Fatal("added to the watchlist")
	}
	if e, _ := AddWatchDomain(context.Background(), api, s, "nobody.io", now); e != nil {
		t.Fatal("unknown domain adds nothing")
	}
}

func TestExtractReplyText(t *testing.T) {
	if ExtractReplyText(`{"result":"hi"}`) != "hi" || ExtractReplyText(`{"message":"m"}`) != "m" || ExtractReplyText("  plain ") != "plain" {
		t.Fatal("extract")
	}
}

func TestParseWallAdRequiresTheFullSnapshot(t *testing.T) {
	good, _ := json.Marshal(ToWallAd(mk("a", true, 3, "A good hook line here"), "A"))
	w, err := ParseWallAd(good)
	if err != nil || w.ID != "a" || w.HookSource == nil || *w.HookSource != "spoken" {
		t.Fatalf("good: %+v %v", w, err)
	}
	for _, bad := range []string{
		`null`, `{"id":"a"}`,
		strings.Replace(string(good), `"id":"a"`, `"id":""`, 1),
		strings.Replace(string(good), `"live":true`, `"live":"yes"`, 1),
		strings.Replace(string(good), `"hookSource":"spoken"`, `"hookSource":"sung"`, 1),
		strings.Replace(string(good), `,"linkUrl":null`, ``, 1),
	} {
		if _, err := ParseWallAd([]byte(bad)); err == nil {
			t.Fatalf("want an error for %s", bad)
		}
	}
}
