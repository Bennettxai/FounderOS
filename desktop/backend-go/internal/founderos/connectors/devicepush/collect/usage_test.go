package collect

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	local = time.UTC // day keys are the Mac's local days; pin them for the suite
	os.Exit(m.Run())
}

var usageNow = time.Date(2026, 9, 5, 15, 0, 0, 0, time.UTC)

func aline(ts string, tokens int, extra ...string) string {
	cwd := ""
	if len(extra) > 0 {
		cwd = fmt.Sprintf(`,"cwd":%q,"entrypoint":%q`, extra[0], extra[1])
	}
	return fmt.Sprintf(`{"type":"assistant","timestamp":%q%s,"message":{"model":"claude-opus-5","usage":{"input_tokens":%d,"output_tokens":10,"cache_creation_input_tokens":0,"cache_read_input_tokens":5}}}`+"\n", ts, cwd, tokens)
}

const noise = `{"type":"user","timestamp":"2026-09-05T13:00:00.000Z","message":{}}` + "\n"

func write(t *testing.T, path, body string) {
	t.Helper()
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func appendTo(t *testing.T, path, body string) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	_, _ = f.WriteString(body)
}

// ---- pure helpers -------------------------------------------------------------

func TestParseClaudeLine(t *testing.T) {
	p := ParseClaudeLine(strings.TrimSpace(aline("2026-09-05T14:00:00.000Z", 100)))
	if p == nil || p.Model != "claude-opus-5" || p.In != 100 || p.Out != 10 || p.CacheRead != 5 || p.TS != "2026-09-05T14:00:00.000Z" {
		t.Fatalf("p = %+v", p)
	}
	for _, bad := range []string{"", "nope", noise, `{"type":"assistant","timestamp":"t","message":{"model":"m"}}`} {
		if ParseClaudeLine(bad) != nil {
			t.Errorf("%q parsed", bad)
		}
	}
	if p := ParseClaudeLine(`{"type":"assistant","timestamp":"t","message":{"usage":{"input_tokens":-3,"output_tokens":"x"}}}`); p == nil || p.In != 0 || p.Out != 0 || p.Model != "unknown" {
		t.Errorf("bad counters must read 0, model unknown: %+v", p)
	}
}

func TestEmptyDaysEndsToday(t *testing.T) {
	days := EmptyDays(usageNow, 7)
	if len(days) != 7 || days[6].Day != "2026-09-05" || days[0].Day != "2026-08-30" {
		t.Fatalf("days = %+v", days)
	}
}

func TestClassify(t *testing.T) {
	home := "/Users/alex"
	cases := []struct {
		got  Lane
		want Lane
	}{
		{ClassifyClaude(home+"/.paperclip/instances/default/workspaces/678ab7fb", "sdk-cli"), Lane{"board", "678ab7fb"}},
		{ClassifyClaude(home+"/.paperclip/instances/default/projects/abc/def/_default", "cli"), Lane{"board", "board project"}},
		{ClassifyClaude(home+"/.superset/worktrees/FounderOS-v1/example-org/awake-quality", "cli"), Lane{"sessions", "FounderOS-v1 · awake-quality"}},
		{ClassifyClaude(home+"/.superset/projects/Silvio-Big-Mamas", "cli"), Lane{"sessions", "Silvio-Big-Mamas"}},
		{ClassifyClaude("/private/tmp", "sdk-cli"), Lane{"automation", "tmp"}},
		{ClassifyClaude("/private/var/folders/2c/abc/T", "sdk-cli"), Lane{"automation", "tmp"}},
		{ClassifyClaude(home+"/Desktop/FounderOS-v1", "cli"), Lane{"terminal", "FounderOS-v1"}},
		{ClassifyClaude("", ""), Lane{"terminal", "unknown"}},
		{ClassifyCodex(home+"/.paperclip/instances/default/workspaces/95ee", "codex_exec", "", false), Lane{"board", "95ee"}},
		{ClassifyCodex(home+"/Projects/x", "codex_exec", "", false), Lane{"automation", "x"}},
		{ClassifyCodex(home+"/Projects/x", "codex-tui", "", false), Lane{"terminal", "x"}},
		{ClassifyCodex(home+"/Projects/x", "codex-tui", "", true), Lane{"board", "x"}},
	}
	for i, c := range cases {
		if c.got != c.want {
			t.Errorf("%d: %+v, want %+v", i, c.got, c.want)
		}
	}
}

func TestBreakdownWindowsAndTop(t *testing.T) {
	slots := SlotMap{}
	tot := func(n float64) Tot4 { return Tot4{In: n} }
	AddToSlots(slots, "2026-09-05T14:30:00Z", Lane{"board", "seat-1"}, tot(10))       // hour
	AddToSlots(slots, "2026-09-05T11:00:00Z", Lane{"sessions", "repo · wt"}, tot(20)) // session
	AddToSlots(slots, "2026-09-05T02:00:00Z", Lane{"terminal", "x"}, tot(40))         // day
	AddToSlots(slots, "2026-09-02T02:00:00Z", Lane{"terminal", "x"}, tot(80))         // week
	AddToSlots(slots, "2026-08-01T02:00:00Z", Lane{"terminal", "x"}, tot(1000))       // outside
	AddToSlots(slots, "not a time", Lane{"terminal", "x"}, tot(5))
	b := BreakdownFromSlots([]SlotMap{slots}, usageNow, EmptyDays(usageNow, 7))
	w := b.Windows
	if w.Hour.Board.In != 10 || w.Session.Sessions.In != 20 || w.Session.Board.In != 10 || w.Day.Terminal.In != 40 || w.Week.Terminal.In != 120 {
		t.Fatalf("windows = %+v", w)
	}
	if len(b.Top) != 3 || b.Top[0].Label != "x" || b.Top[0].Burn != 120 || b.Top[2].Source != "board" {
		t.Fatalf("top = %+v", b.Top)
	}
	PruneSlots(slots, usageNow)
	for k := range slots {
		if k < (usageNow.UnixMilli()-8*86400_000)/slotMS {
			t.Errorf("slot %d survived the prune", k)
		}
	}
}

func TestPlanNames(t *testing.T) {
	if got := ClaudePlanName(map[string]any{"organizationType": "claude_max", "organizationRateLimitTier": "default_claude_max_20x"}); got == nil || *got != "Claude Max 20x" {
		t.Errorf("got %v", got)
	}
	if got := ClaudePlanName(map[string]any{"organizationType": "claude_team"}); got == nil || *got != "Claude Team" {
		t.Errorf("got %v", got)
	}
	if ClaudePlanName(nil) != nil {
		t.Error("no login, no plan")
	}
	if got := CodexPlanName("prolite"); got == nil || *got != "ChatGPT Pro Lite" {
		t.Errorf("got %v", got)
	}
	if got := CodexPlanName("some_new-tier"); got == nil || *got != "ChatGPT Some New Tier" {
		t.Errorf("got %v", got)
	}
	if got := OllamaPlanName("pro"); got == nil || *got != "Ollama Pro" {
		t.Errorf("got %v", got)
	}
}

func codexLine(ts string, totalIn int, rl bool) string {
	limits := ""
	if rl {
		limits = `,"rate_limits":{"primary":{"used_percent":11,"window_minutes":300,"resets_at":1788274423},"secondary":{"used_percent":42,"window_minutes":10080,"resets_at":1788274423},"plan_type":"pro"}`
	}
	return fmt.Sprintf(`{"timestamp":%q,"type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":%d,"cached_input_tokens":0,"output_tokens":5}}%s}}`+"\n", ts, totalIn, limits)
}

func TestParseCodexLineAssignsWindowsByLength(t *testing.T) {
	p := ParseCodexLine(strings.TrimSpace(codexLine("2026-09-05T10:00:00.000Z", 500, true)))
	if p == nil || p.Cumulative.In != 500 || p.RateLimits == nil || p.RateLimits.Weekly.UsedPercent != 42 || p.RateLimits.Session.WindowMinutes != 300 || p.RateLimits.PlanType != "pro" {
		t.Fatalf("p = %+v", p)
	}
	if *p.RateLimits.Weekly.ResetsAt != time.Unix(1788274423, 0).UTC().Format("2006-01-02T15:04:05.000Z") {
		t.Errorf("resetsAt = %s", *p.RateLimits.Weekly.ResetsAt)
	}
	weeklyAsPrimary := `{"timestamp":"t","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":1}},"rate_limits":{"primary":{"used_percent":9,"window_minutes":10080}}}}`
	if p := ParseCodexLine(weeklyAsPrimary); p.RateLimits.Weekly == nil || p.RateLimits.Session != nil {
		t.Errorf("a weekly window reported as primary is the weekly gauge: %+v", p.RateLimits)
	}
	if ParseCodexLine(`{"type":"response_item","payload":{"type":"function_call"}}`) != nil || ParseCodexLine("nope") != nil {
		t.Error("other shapes must not parse")
	}
}

func TestCountOllamaRequests(t *testing.T) {
	gin := func(ago time.Duration, path string) string {
		return fmt.Sprintf(`[GIN] %s | 200 |  1.2ms |  127.0.0.1 | POST     "%s"`, usageNow.Add(-ago).In(local).Format("2006/01/02 - 15:04:05"), path)
	}
	lines := []string{gin(5*time.Minute, "/api/chat"), gin(6*time.Minute, "/api/embed"), gin(2*time.Hour, "/v1/chat/completions"),
		gin(72*time.Hour, "/api/embed"), "[GIN] junk", gin(time.Minute, "/api/tags"), gin(-time.Hour, "/api/chat")}
	c := CountOllamaRequests(lines, usageNow)
	if c.Hour.Chat != 1 || c.Hour.Embed != 1 || c.Session.Chat != 2 || c.Day.Chat != 2 || c.Week.Embed != 2 || c.Week.Chat != 2 {
		t.Fatalf("counts = %+v", c)
	}
}

// ---- Claude scanner -------------------------------------------------------------

func TestScanClaudeProjectsAggregatesIncrementally(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "proj-a", "s1.jsonl")
	write(t, f, noise+aline("2026-09-05T12:00:00.000Z", 100)+aline("2026-09-04T12:00:00.000Z", 200))
	write(t, filepath.Join(dir, "ignore.txt"), "not a transcript")
	cache := NewScanCache()
	seat := ScanClaudeProjects(dir, usageNow, cache, SeatInfo{ID: "claude-mac", Label: "Claude · mac"})
	if seat.Days[6].In != 100 || seat.Days[5].In != 200 || seat.ByModel["claude-opus-5"].Out != 20 || *seat.LastActivity != "2026-09-05T12:00:00.000Z" {
		t.Fatalf("seat = %+v", seat)
	}
	if seat.Kind != "claude" || seat.Source != "local" || seat.ID != "claude-mac" || seat.Official != nil {
		t.Errorf("seat = %+v", seat)
	}
	before := cache.files[f].offset
	appendTo(t, f, aline("2026-09-05T14:00:00.000Z", 50))
	seat = ScanClaudeProjects(dir, usageNow, cache, SeatInfo{ID: "claude-mac", Label: "x"})
	if seat.Days[6].In != 150 || cache.files[f].offset <= before {
		t.Fatalf("appended bytes: in=%v offset=%d", seat.Days[6].In, cache.files[f].offset)
	}
	write(t, f, aline("2026-09-05T12:00:00.000Z", 30)) // shorter rewrite
	if seat = ScanClaudeProjects(dir, usageNow, cache, SeatInfo{ID: "x", Label: "x"}); seat.Days[6].In != 30 {
		t.Fatalf("a rewritten file is rebuilt from zero: %v", seat.Days[6].In)
	}
}

func TestScanClaudeHoldsBackAPartialLine(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "s.jsonl")
	half := aline("2026-09-05T13:00:00.000Z", 40)
	write(t, f, aline("2026-09-05T12:00:00.000Z", 100)+half[:50])
	cache := NewScanCache()
	if seat := ScanClaudeProjects(dir, usageNow, cache, SeatInfo{ID: "x", Label: "x"}); seat.Days[6].In != 100 {
		t.Fatalf("in = %v", seat.Days[6].In)
	}
	appendTo(t, f, half[50:])
	if seat := ScanClaudeProjects(dir, usageNow, cache, SeatInfo{ID: "x", Label: "x"}); seat.Days[6].In != 140 {
		t.Fatalf("in = %v", seat.Days[6].In)
	}
}

func TestScanClaudeWindowsModelsAndLanes(t *testing.T) {
	dir := t.TempDir()
	board := "/Users/b/.paperclip/instances/default/workspaces/seat-1"
	write(t, filepath.Join(dir, "p", "s.jsonl"),
		aline("2026-08-01T12:00:00.000Z", 999)+
			aline("2026-09-05T14:30:00.000Z", 100, board, "sdk-cli")+
			aline("2026-09-05T09:00:00.000Z", 50, "/Users/b/Desktop/x", "cli"))
	seat := ScanClaudeProjects(dir, usageNow, NewScanCache(), SeatInfo{ID: "x", Label: "x"})
	if seat.ByModel["claude-opus-5"].In != 150 {
		t.Errorf("byModel must respect the window: %+v", seat.ByModel)
	}
	if seat.Breakdown == nil || seat.Breakdown.Windows.Hour.Board.In != 100 || seat.Breakdown.Windows.Day.Terminal.In != 50 {
		t.Fatalf("breakdown = %+v", seat.Breakdown)
	}
	if seat.Note != "local burn estimate; no Claude login on this box to read the official limit %" {
		t.Errorf("note = %q", seat.Note)
	}
	empty := ScanClaudeProjects(filepath.Join(dir, "nope"), usageNow, NewScanCache(), SeatInfo{ID: "x", Label: "x"})
	if !strings.Contains(empty.Note, "no transcripts") || empty.LastActivity != nil {
		t.Errorf("empty = %+v", empty)
	}
}

func TestClaudeAccountPlan(t *testing.T) {
	f := filepath.Join(t.TempDir(), ".claude.json")
	write(t, f, `{"oauthAccount":{"organizationType":"claude_max","organizationRateLimitTier":"default_claude_max_5x"},"mcpServers":{}}`)
	if got := ClaudeAccountPlan(f); got == nil || *got != "Claude Max 5x" {
		t.Errorf("got %v", got)
	}
	if ClaudeAccountPlan(filepath.Join(t.TempDir(), "missing.json")) != nil {
		t.Error("no file, no plan")
	}
}

// ---- Codex seat ----------------------------------------------------------------

func TestCodexSeat(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "2026", "09", "05", "rollout-a.jsonl"),
		`{"type":"session_meta","payload":{"cwd":"/Users/b/.superset/worktrees/FounderOS-v1/x/wt","originator":"codex-tui"}}`+"\n"+
			codexLine("2026-09-05T14:10:00.000Z", 100, false)+codexLine("2026-09-05T14:20:00.000Z", 300, true))
	seat := CodexSeat([]string{dir}, nil, usageNow, NewCodexCache(), SeatInfo{ID: "codex-mac", Label: "Codex · mac"})
	if seat == nil || seat.Days[6].In != 300 || seat.Official.Weekly.UsedPercent != 42 || seat.CapturedAt != "2026-09-05T14:20:00.000Z" {
		t.Fatalf("seat = %+v", seat)
	}
	if seat.Plan == nil || *seat.Plan != "ChatGPT Pro" || seat.Note != "official plan gauge, as of the last codex run" {
		t.Errorf("plan/note = %v %q", seat.Plan, seat.Note)
	}
	if seat.Breakdown.Windows.Hour.Sessions.In != 300 {
		t.Errorf("deltas land on the session lane: %+v", seat.Breakdown.Windows.Hour)
	}

	paperclipRoot := t.TempDir()
	boardHome := filepath.Join(paperclipRoot, "instances", "default", "companies", "c1", "codex-home", "sessions")
	write(t, filepath.Join(boardHome, "2026", "09", "05", "rollout-b.jsonl"), codexLine("2026-09-05T14:30:00.000Z", 70, false))
	_ = os.MkdirAll(filepath.Join(paperclipRoot, "instances", "default", "companies", "c2", "no-codex"), 0o755)
	homes := PaperclipCodexHomes(paperclipRoot)
	if len(homes) != 1 || homes[0] != boardHome {
		t.Fatalf("homes = %v", homes)
	}
	seat = CodexSeat([]string{dir}, homes, usageNow, NewCodexCache(), SeatInfo{ID: "codex-mac", Label: "Codex · mac"})
	if seat.Breakdown.Windows.Hour.Board.In != 70 || seat.Days[6].In != 370 {
		t.Fatalf("board sessions: %+v", seat.Breakdown.Windows.Hour)
	}
	if CodexSeat([]string{filepath.Join(dir, "missing")}, nil, usageNow, NewCodexCache(), SeatInfo{ID: "c", Label: "c"}) != nil {
		t.Error("no sessions means no codex lane")
	}
}

// ---- Ollama lane ---------------------------------------------------------------

func TestOllamaLane(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/tags" && r.Method == http.MethodGet:
			_, _ = io.WriteString(w, `{"models":[{"name":"bge-m3:latest"},{"name":"gpt-oss:120b-cloud","remote_host":"https://ollama.com:443"},{"name":"kimi:cloud"},{"nope":1}]}`)
		case r.URL.Path == "/api/me" && r.Method == http.MethodPost:
			_, _ = io.WriteString(w, `{"name":"founderos","plan":"pro"}`)
		default:
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()
	logs := t.TempDir()
	write(t, filepath.Join(logs, "server.log"), fmt.Sprintf(`[GIN] %s | 200 | 1ms | 127.0.0.1 | POST     "/api/embed"`+"\n", usageNow.Add(-10*time.Minute).In(local).Format("2006/01/02 - 15:04:05")))

	lane := ReadOllamaLane(context.Background(), nil, srv.URL+"/v1", logs, usageNow)
	if lane.State != "up" || lane.Plan == nil || *lane.Plan != "Ollama Pro" || len(lane.Models) != 3 {
		t.Fatalf("lane = %+v", lane)
	}
	if lane.Models[0].Cloud || !lane.Models[1].Cloud || !lane.Models[2].Cloud {
		t.Errorf("models = %+v", lane.Models)
	}
	if lane.Requests == nil || lane.Requests.Hour.Embed != 1 {
		t.Errorf("requests = %+v", lane.Requests)
	}
	raw, _ := json.Marshal(lane)
	if !strings.Contains(string(raw), `"note":"Ollama logs requests, not tokens`) {
		t.Errorf("lane json = %s", raw)
	}

	srv.Close()
	down := ReadOllamaLane(context.Background(), nil, srv.URL, t.TempDir(), usageNow)
	if down.State != "down" || down.Plan != nil || len(down.Models) != 0 || down.Requests != nil {
		t.Fatalf("down = %+v", down)
	}
}
