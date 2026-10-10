package usage

import (
	"testing"
	"time"

	dp "github.com/rhl/businessos-backend/internal/founderos/connectors/devicepush"
)

func sp(s string) *string { return &s }

func tot(in, out, cw, cr float64) dp.Tot {
	return dp.Tot{In: in, Out: out, CacheWrite: cw, CacheRead: cr}
}

func days(vals ...float64) []dp.DayBucket {
	out := []dp.DayBucket{}
	for i, v := range vals {
		out = append(out, dp.DayBucket{Day: "2026-09-2" + string(rune('4'+i)), Tot: tot(v, 0, 0, 1)})
	}
	return out
}

func seat(id, label, captured string, plan *string, d []dp.DayBucket) Seat {
	return Seat{SeatUsage: dp.SeatUsage{
		ID: id, Kind: "claude", Label: label, Source: "push", CapturedAt: captured,
		Days: d, ByModel: map[string]dp.Tot{"claude-opus-5": tot(10, 5, 1, 100)}, Plan: plan,
	}}
}

var now = time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)

func TestCombineSeatsWithNoSeatsIsUnknown(t *testing.T) {
	if p := CombineSeats(nil, now); p != nil {
		t.Fatalf("no seats must read unknown (nil), got %+v", p)
	}
}

func TestCombineSeatsSumsDaysAndModelsAcrossMachines(t *testing.T) {
	a := seat("claude-macbook", "Claude · MacBook", "2026-09-30T11:59:00Z", sp("Claude Max 20x"), days(1, 2, 3))
	b := seat("claude-mini", "Claude · mini", "2026-09-30T11:00:00Z", nil, days(10, 20, 30))
	b.LastActivity = sp("2026-09-30T10:00:00Z")
	a.LastActivity = sp("2026-09-30T09:00:00Z")
	p := CombineSeats([]Seat{a, b}, now)
	if p == nil {
		t.Fatal("nil plan")
	}
	if got := p.Days[2].In; got != 33 {
		t.Fatalf("day 3 in = %v, want 33", got)
	}
	if got := p.ByModel["claude-opus-5"].In; got != 20 {
		t.Fatalf("model in = %v, want 20", got)
	}
	if p.Plan == nil || *p.Plan != "Claude Max 20x" {
		t.Fatalf("plan = %v", p.Plan)
	}
	if p.LastActivity == nil || *p.LastActivity != "2026-09-30T10:00:00Z" {
		t.Fatalf("lastActivity = %v", p.LastActivity)
	}
	if len(p.Machines) != 2 || p.Machines[0].Stale || p.Machines[1].Stale {
		t.Fatalf("machines = %+v", p.Machines)
	}
	if p.PlanConflict != nil {
		t.Fatalf("one plan name must not conflict: %v", p.PlanConflict)
	}
}

func TestCombineSeatsFlagsStaleAndConflictingPlans(t *testing.T) {
	a := seat("claude-macbook", "Claude · MacBook", "2026-09-30T11:59:00Z", sp("Claude Max 20x"), days(1))
	b := seat("claude-mini", "Claude · mini", "2026-09-28T11:00:00Z", sp("Claude Max 5x"), days(1))
	c := seat("claude-x", "Claude · x", "2026-09-30T11:59:00Z", nil, days(1))
	c.ReceiverStale = true
	p := CombineSeats([]Seat{a, b, c}, now)
	if p.Machines[0].Stale || !p.Machines[1].Stale || !p.Machines[2].Stale {
		t.Fatalf("stale flags = %+v", p.Machines)
	}
	if len(p.PlanConflict) != 2 || p.PlanConflict[0] != "Claude Max 20x (Claude · MacBook)" {
		t.Fatalf("planConflict = %v", p.PlanConflict)
	}
}

func TestCombineSeatsTakesTheFreshestOfficialGauge(t *testing.T) {
	a := seat("a", "A", "2026-09-30T10:00:00Z", nil, days(1))
	a.Official = &dp.Official{Session: &dp.OfficialWindow{UsedPercent: 10, WindowMinutes: 300}}
	b := seat("b", "B", "2026-09-30T11:00:00Z", nil, days(1))
	b.Official = &dp.Official{Session: &dp.OfficialWindow{UsedPercent: 40, WindowMinutes: 300}}
	p := CombineSeats([]Seat{a, b}, now)
	if p.Official == nil || p.Official.Session.UsedPercent != 40 {
		t.Fatalf("official = %+v", p.Official)
	}
}

// Prod 8abea0c: the note belongs with the gauge, so a seat that read the
// official % speaks first, even when an estimate-only seat comes earlier.
func TestCombineSeatsNoteComesFromTheSeatWithTheGauge(t *testing.T) {
	a := seat("a", "A", "2026-09-30T11:00:00Z", nil, days(1))
	a.Note = "local burn estimate; no Claude login on this box to read the official limit %"
	b := seat("b", "B", "2026-09-30T10:00:00Z", nil, days(1))
	b.Official = &dp.Official{Session: &dp.OfficialWindow{UsedPercent: 8, WindowMinutes: 300}}
	b.Note = "burn measured from transcripts; limit % is the official gauge from this login"
	if p := CombineSeats([]Seat{a, b}, now); p.Note != b.Note {
		t.Fatalf("note = %q, want the gauge seat's", p.Note)
	}
	if p := CombineSeats([]Seat{a}, now); p.Note != a.Note {
		t.Fatalf("no gauge anywhere: note = %q, want the first seat's", p.Note)
	}
}

func bd(board float64, label string) *dp.Breakdown {
	l := dp.LaneTots{Board: tot(board, 0, 0, 0)}
	return &dp.Breakdown{Windows: dp.Windows{Hour: l, Session: l, Day: l, Week: l}, Top: []dp.TopBurner{{Source: "board", Label: label, Burn: board}}}
}

func TestMergeBreakdownsSumsLanesAndTopBurners(t *testing.T) {
	m := MergeBreakdowns([]*dp.Breakdown{bd(5, "seat-1"), bd(7, "seat-1"), bd(3, "seat-2")})
	if m == nil || m.Windows.Week.Board.In != 15 {
		t.Fatalf("merged = %+v", m)
	}
	if len(m.Top) != 2 || m.Top[0].Label != "seat-1" || m.Top[0].Burn != 12 {
		t.Fatalf("top = %+v", m.Top)
	}
	if MergeBreakdowns(nil) != nil {
		t.Fatal("no breakdowns must be nil (no lane data), not zeros")
	}
}

func TestNameBoardLabels(t *testing.T) {
	b := bd(5, "abc123")
	b.Top = append(b.Top, dp.TopBurner{Source: "terminal", Label: "abc123", Burn: 1})
	got := NameBoardLabels(b, map[string]string{"abc123-full-id": "Conductor"})
	if got.Top[0].Label != "Conductor" || got.Top[1].Label != "abc123" {
		t.Fatalf("top = %+v", got.Top)
	}
}

func ollama(id, label, captured, state string, plan *string, chat int) Ollama {
	return Ollama{OllamaSnapshot: dp.OllamaSnapshot{Kind: "ollama", ID: id, Label: label, CapturedAt: captured, Lane: dp.OllamaLane{
		State: state, Plan: plan, Models: []dp.OllamaModel{{Name: "gpt-oss:120b-cloud", Cloud: true}},
		Requests: &dp.RequestWindows{Day: dp.RequestCounts{Chat: chat, Embed: 1}, Week: dp.RequestCounts{Chat: chat, Embed: 1}}, Note: "log",
	}}}
}

func TestCombineOllama(t *testing.T) {
	if CombineOllama(nil, now) != nil {
		t.Fatal("no machine reporting Ollama must read unknown (nil), not a down server")
	}
	b := CombineOllama([]Ollama{
		ollama("ollama-mini", "Ollama · mini", "2026-09-30T11:00:00Z", "down", nil, 2),
		ollama("ollama-mbp", "Ollama · MacBook", "2026-09-30T11:30:00Z", "up", sp("Ollama Pro"), 3),
	}, now)
	if b.State != "up" || b.Plan == nil || *b.Plan != "Ollama Pro" {
		t.Fatalf("board = %+v", b)
	}
	if b.Requests == nil || b.Requests.Day.Chat != 5 || b.Requests.Day.Embed != 2 {
		t.Fatalf("requests = %+v", b.Requests)
	}
	if len(b.Models) != 2 || b.Models[1].Host != "Ollama · MacBook" {
		t.Fatalf("models = %+v", b.Models)
	}
	if len(b.Machines) != 2 {
		t.Fatalf("machines = %+v", b.Machines)
	}
}

func TestCombineOllamaWithoutLogsHasNoRequests(t *testing.T) {
	o := ollama("a", "A", "2026-09-30T11:00:00Z", "up", nil, 0)
	o.Lane.Requests = nil
	if b := CombineOllama([]Ollama{o}, now); b.Requests != nil {
		t.Fatalf("requests must be unknown without a log: %+v", b.Requests)
	}
}

func TestDedupeSeatsKeepsTheNewestReadingPerID(t *testing.T) {
	old := seat("claude-mini", "Claude · mini", "2026-09-29T11:00:00Z", nil, days(1))
	fresh := seat("claude-mini", "Claude · mini", "2026-09-30T11:00:00Z", nil, days(9))
	other := seat("claude-mbp", "Claude · MBP", "2026-09-30T11:00:00Z", nil, days(1))
	got := DedupeSeats([]Seat{old, other, fresh})
	if len(got) != 2 || got[0].ID != "claude-mini" || got[0].Days[0].In != 9 || got[1].ID != "claude-mbp" {
		t.Fatalf("deduped = %+v", got)
	}
}

func TestBuildSplitsSeatsByKind(t *testing.T) {
	cl := seat("claude-mbp", "Claude · MBP", "2026-09-30T11:00:00Z", nil, days(1))
	cx := seat("codex-mbp", "Codex · MBP", "2026-09-30T11:00:00Z", sp("ChatGPT Pro"), days(2))
	cx.Kind = "codex"
	b := Build([]Seat{cl, cx}, nil, nil, now)
	if b.Claude == nil || b.Codex == nil || b.Ollama != nil {
		t.Fatalf("board = %+v", b)
	}
	if b.Codex.Plan == nil || *b.Codex.Plan != "ChatGPT Pro" {
		t.Fatalf("codex plan = %v", b.Codex.Plan)
	}
	if b.GeneratedAt != "2026-09-30T12:00:00Z" {
		t.Fatalf("generatedAt = %s", b.GeneratedAt)
	}
	empty := Build(nil, nil, nil, now)
	if empty.Claude != nil || empty.Codex != nil || empty.Ollama != nil {
		t.Fatalf("no pushes must be all unknown: %+v", empty)
	}
}

// The day window ends today, whatever machine sorts first: seats come back
// ORDER BY device, so a MacBook that last pushed on the 28th must not drop the
// mini's 29th and 30th or read "today" as the 28th.
func TestCombineSeatsWindowEndsTodayNotAtTheFirstSeat(t *testing.T) {
	mk := func(from int, vals ...float64) []dp.DayBucket {
		out := []dp.DayBucket{}
		for i, v := range vals {
			out = append(out, dp.DayBucket{Day: time.Date(2026, 9, from+i, 12, 0, 0, 0, time.UTC).Format("2006-01-02"), Tot: tot(v, 0, 0, 0)})
		}
		return out
	}
	stale := seat("claude-a-macbook", "Claude · MacBook", "2026-09-28T20:00:00Z", nil, mk(22, 1, 1, 1, 1, 1, 1, 1))
	fresh := seat("claude-b-mini", "Claude · mini", "2026-09-30T11:55:00Z", nil, mk(24, 2, 2, 2, 2, 2, 2, 9))
	p := CombineSeats([]Seat{stale, fresh}, now)
	if len(p.Days) != 7 || p.Days[0].Day != "2026-09-24" || p.Days[6].Day != "2026-09-30" {
		t.Fatalf("window = %v … %v (%d days)", p.Days[0].Day, p.Days[len(p.Days)-1].Day, len(p.Days))
	}
	if p.Days[6].In != 9 {
		t.Fatalf("today's burn from the mini = %v, want 9", p.Days[6].In)
	}
	if p.Days[4].In != 3 { // the 28th: both machines
		t.Fatalf("the 28th = %v, want 3", p.Days[4].In)
	}
}

// A stale seat's "last hour" and "5h session" describe then, not now: they are
// left out of the plan's windows (day and week still count, they are history).
func TestStaleSeatsDoNotCountTowardTheLastHourOrSession(t *testing.T) {
	burn := func(n float64) dp.Breakdown {
		l := dp.LaneTots{Sessions: tot(n, 0, 0, 0)}
		return dp.Breakdown{Windows: dp.Windows{Hour: l, Session: l, Day: l, Week: l}, Top: []dp.TopBurner{}}
	}
	fresh := seat("claude-a", "A", "2026-09-30T11:59:00Z", nil, days(1))
	fb := burn(10)
	fresh.Breakdown = &fb
	old := seat("claude-b", "B", "2026-09-29T08:00:00Z", nil, days(1)) // a day old: stale
	ob := burn(100)
	old.Breakdown = &ob
	p := CombineSeats([]Seat{fresh, old}, now)
	w := p.Breakdown.Windows
	if w.Hour.Sessions.In != 10 || w.Session.Sessions.In != 10 {
		t.Fatalf("hour/session = %v/%v, want only the fresh seat's 10", w.Hour.Sessions.In, w.Session.Sessions.In)
	}
	if w.Week.Sessions.In != 110 {
		t.Fatalf("week = %v, want 110 (history counts)", w.Week.Sessions.In)
	}
}
