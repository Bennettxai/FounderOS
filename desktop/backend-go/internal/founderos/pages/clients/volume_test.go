package clients

import (
	"math"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

// Ported from FounderOS v1 tests/clients-volume.test.ts. The Request Volume
// card, the step line, the weekday rhythm and the one "Needs you" card come
// from this pure view-model; a request is in exactly one status, so the
// meters add up to the headline.

var chicago = func() *time.Location {
	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		panic(err)
	}
	return loc
}()

var volNow = time.Date(2026, 9, 24, 18, 0, 0, 0, chicago)

func req(over func(*Work)) Work {
	w := Work{ClientID: "silvio-big-mamas", Status: StatusSaved, CreatedAt: "2026-09-23T10:00:00.000Z", Brief: "Game day carousel"}
	if over != nil {
		over(&w)
	}
	return w
}

var volClients = []ClientRef{{ID: "silvio-big-mamas", Name: "Silvio / Big Mama's"}, {ID: "acme", Name: "Acme Roofing"}}

var volWork = []Work{
	req(nil),
	req(func(w *Work) { w.Status = StatusLaunched; w.CreatedAt = "2026-09-22T10:00:00.000Z" }),
	req(func(w *Work) { w.Status = StatusLaunched; w.CreatedAt = "2026-09-22T11:00:00.000Z" }),
	req(func(w *Work) {
		w.Status = StatusNeedsAttention
		w.Brief = "Menu refresh copy"
		w.CreatedAt = "2026-09-21T10:00:00.000Z"
	}),
	req(func(w *Work) {
		w.Status = StatusLaunching
		w.Brief = "Reel script"
		w.CreatedAt = "2026-09-20T10:00:00.000Z"
	}),
	// outside the 14-day window: still counts in the totals, not in the line
	req(func(w *Work) {
		w.ClientID = "acme"
		w.Status = StatusLaunched
		w.CreatedAt = "2026-08-01T10:00:00.000Z"
	}),
}

func TestVolumeHeadlineChipsCaption(t *testing.T) {
	v := ComputeVolume(volClients, volWork, volNow, 14)
	if v.Headline != 6 {
		t.Fatalf("headline %d", v.Headline)
	}
	want := map[Status]int{StatusSaved: 1, StatusLaunching: 1, StatusLaunched: 3, StatusNeedsAttention: 1}
	for k, n := range want {
		if v.Counts[k] != n {
			t.Fatalf("counts %v", v.Counts)
		}
	}
	var texts []string
	for _, c := range v.Chips {
		texts = append(texts, c.Text)
	}
	if strings.Join(texts, "|") != "3 launched|1 saved|1 launching|1 needs attention" {
		t.Fatalf("chips %v", texts)
	}
	if v.Caption != "across 2 confirmed clients · 2 with requests" {
		t.Fatalf("caption %q", v.Caption)
	}
}

func TestVolumeMetersSumToOne(t *testing.T) {
	v := ComputeVolume(volClients, volWork, volNow, 14)
	var labels []string
	sum := 0.0
	for _, m := range v.Meters {
		labels = append(labels, m.Label)
		sum += *m.Frac
		if !strings.HasPrefix(m.Hue, "var(--") {
			t.Fatalf("hue %q", m.Hue)
		}
	}
	if strings.Join(labels, "|") != "Launched (3)|Saved drafts (1)|Launching (1)|Needs attention (1)" {
		t.Fatalf("labels %v", labels)
	}
	if math.Abs(sum-1) > 1e-9 || v.Meters[0].Display != "50%" {
		t.Fatalf("sum %v display %q", sum, v.Meters[0].Display)
	}
	if v.Foot != "draft first · nothing publishes automatically" {
		t.Fatalf("foot %q", v.Foot)
	}
}

func TestVolumeStepLineIsPerLocalDayOldestFirst(t *testing.T) {
	v := ComputeVolume(volClients, volWork, volNow, 14)
	if len(v.Series) != 14 || v.Series[13].Label != "Sep 24" || v.Series[0].Label != "Sep 11" {
		t.Fatalf("series %v", v.Series)
	}
	total := 0
	for _, s := range v.Series {
		total += s.Count
	}
	if v.RequestsInWindow != 5 || total != 5 {
		t.Fatalf("window %d total %d", v.RequestsInWindow, total)
	}
}

func TestVolumeRhythmMondayFirst(t *testing.T) {
	v := ComputeVolume(volClients, volWork, volNow, 14)
	var labels []string
	total := 0
	for _, d := range v.Rhythm {
		labels = append(labels, d.Label)
		total += d.Count
	}
	if strings.Join(labels, ",") != "Mon,Tue,Wed,Thu,Fri,Sat,Sun" || total != 5 {
		t.Fatalf("rhythm %v", v.Rhythm)
	}
	// Sep 20 2026 is a Sunday, Sep 21 a Monday (Chicago local)
	if v.Rhythm[6].Count != 1 || v.Rhythm[0].Count != 1 || v.Rhythm[1].Count != 2 || v.Rhythm[2].Count != 1 {
		t.Fatalf("rhythm %v", v.Rhythm)
	}
}

func TestVolumePerClientKeepsConfirmedOrder(t *testing.T) {
	v := ComputeVolume(volClients, volWork, volNow, 14)
	if len(v.PerClient) != 2 || v.PerClient[0] != (ClientCount{ID: "silvio-big-mamas", Count: 5}) || v.PerClient[1] != (ClientCount{ID: "acme", Count: 1}) {
		t.Fatalf("perClient %v", v.PerClient)
	}
}

func TestVolumeInsightNeedsYou(t *testing.T) {
	v := ComputeVolume(volClients, volWork, volNow, 14)
	if v.Insight.Value != 2 || v.Insight.Headline != "1 needs attention · 1 launch unconfirmed." || v.Insight.Body != "Menu refresh copy · Reel script" {
		t.Fatalf("insight %+v", v.Insight)
	}
	if math.Abs(v.Insight.Frac-2.0/6) > 1e-9 {
		t.Fatalf("frac %v", v.Insight.Frac)
	}
}

func TestVolumeAllClear(t *testing.T) {
	v := ComputeVolume(volClients, []Work{req(func(w *Work) { w.Status = StatusLaunched })}, volNow, 14)
	if v.Insight.Value != 0 || v.Insight.Headline != "Nothing waiting on you." || v.Insight.Body != "1 request launched, 0 saved as drafts." {
		t.Fatalf("insight %+v", v.Insight)
	}
}

func TestVolumeEmptyStaysHonest(t *testing.T) {
	v := ComputeVolume(volClients, nil, volNow, 14)
	if v.Headline != 0 || len(v.Chips) != 0 || len(v.Meters) != 0 || v.RequestsInWindow != 0 {
		t.Fatalf("%+v", v)
	}
	if v.Chips == nil || v.Meters == nil {
		t.Fatal("empty lists must serialize as [], not null")
	}
	if v.Caption != "across 2 confirmed clients · no requests yet" || v.Insight.Body != "No requests yet. Save a brief below to start one." {
		t.Fatalf("%q %q", v.Caption, v.Insight.Body)
	}
}

func TestVolumeClipsLongBriefs(t *testing.T) {
	long := strings.Repeat("x", 200)
	v := ComputeVolume(volClients, []Work{req(func(w *Work) { w.Status = StatusNeedsAttention; w.Brief = long })}, volNow, 14)
	if utf8.RuneCountInString(v.Insight.Body) > 60 || !strings.HasSuffix(v.Insight.Body, "…") {
		t.Fatalf("body %q", v.Insight.Body)
	}
	if got := clip("  a   b \n c  "); got != "a b c" {
		t.Fatalf("clip collapses whitespace: %q", got)
	}
}

func TestVolumeSkipsUnparseableAndFutureRows(t *testing.T) {
	v := ComputeVolume(volClients, []Work{
		req(func(w *Work) { w.CreatedAt = "not a date" }),
		req(func(w *Work) { w.CreatedAt = "2026-09-30T10:00:00.000Z" }),
	}, volNow, 14)
	if v.Headline != 2 || v.RequestsInWindow != 0 {
		t.Fatalf("%+v", v)
	}
}

func TestProjectsAreTheConfirmedRoster(t *testing.T) {
	if len(Projects) != 1 || Projects[0].ID != "silvio-big-mamas" || Projects[0].ProjectID != "2ffe9fec-9041-4577-83c3-72902a6aae98" || Projects[0].ProjectName != "Silvio-Big Mamas" {
		t.Fatalf("%+v", Projects)
	}
	if ProjectByID("silvio-big-mamas") == nil || ProjectByID("nobody") != nil {
		t.Fatal("lookup")
	}
}
