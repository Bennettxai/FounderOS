package collect

// Pure token-burn accounting, ported from FounderOS v1's lib/usage.ts. Sources
// are free and local (Claude transcripts, Codex rollouts, the Ollama server
// log); nothing here calls an LLM or a paid API.

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/devicepush"
)

// UsageWindowDays is the seat's day window (USAGE_WINDOW_DAYS).
const UsageWindowDays = 7

// Tot4 is one in/out/cacheWrite/cacheRead total.
type Tot4 = devicepush.Tot

// Lane is where tokens burned: board, sessions, terminal or automation.
type Lane struct {
	Source string
	Label  string
}

var burnSources = []string{"board", "sessions", "terminal", "automation"}

func isBurnSource(s string) bool {
	for _, b := range burnSources {
		if b == s {
			return true
		}
	}
	return false
}

func parseTS(ts string) (time.Time, bool) {
	t, err := time.Parse(time.RFC3339Nano, ts)
	return t, err == nil
}

// DayKey is the local calendar day of an ISO timestamp: the day the operator lived.
func DayKey(t time.Time) string { return t.In(local).Format("2006-01-02") }

// EmptyDays ends today and spans n days, oldest first.
func EmptyDays(now time.Time, n int) []devicepush.DayBucket {
	out := make([]devicepush.DayBucket, 0, n)
	for i := n - 1; i >= 0; i-- {
		out = append(out, devicepush.DayBucket{Day: DayKey(now.Add(-time.Duration(i) * 24 * time.Hour))})
	}
	return out
}

func addTot(a *Tot4, b Tot4) {
	a.In += b.In
	a.Out += b.Out
	a.CacheWrite += b.CacheWrite
	a.CacheRead += b.CacheRead
}

// BurnOf is fresh work for the provider: input + output + cache writes.
func BurnOf(t Tot4) float64 { return t.In + t.Out + t.CacheWrite }

// num is a positive finite JSON number, else 0: zeros are not evidence.
func num(v any) float64 {
	f, ok := v.(float64)
	if !ok || math.IsNaN(f) || math.IsInf(f, 0) || f <= 0 {
		return 0
	}
	return f
}

// ---- Claude transcript lines ---------------------------------------------------

type ParsedUsageLine struct {
	TS         string
	Model      string
	Cwd        string
	Entrypoint string
	Sidechain  bool
	Tot4
}

// ParseClaudeLine turns one transcript line into token counters, or nil for
// anything that is not an assistant message with a usage block.
func ParseClaudeLine(line string) *ParsedUsageLine {
	if strings.TrimSpace(line) == "" {
		return nil
	}
	var d map[string]any
	if json.Unmarshal([]byte(line), &d) != nil {
		return nil
	}
	ts, isStr := d["timestamp"].(string)
	if d["type"] != "assistant" || !isStr {
		return nil
	}
	msg, _ := d["message"].(map[string]any)
	usage, _ := msg["usage"].(map[string]any)
	if usage == nil {
		return nil
	}
	p := &ParsedUsageLine{TS: ts, Model: "unknown", Sidechain: d["isSidechain"] == true}
	if m, ok := msg["model"].(string); ok {
		p.Model = m
	}
	p.Cwd, _ = d["cwd"].(string)
	p.Entrypoint, _ = d["entrypoint"].(string)
	p.In = num(usage["input_tokens"])
	p.Out = num(usage["output_tokens"])
	p.CacheWrite = num(usage["cache_creation_input_tokens"])
	p.CacheRead = num(usage["cache_read_input_tokens"])
	return p
}

// ---- lanes -----------------------------------------------------------------

var (
	tmpDir      = regexp.MustCompile(`^/(private/)?(tmp|var/folders)(/|$)`)
	boardSeat   = regexp.MustCompile(`/workspaces/([^/]+)`)
	supersetWT  = regexp.MustCompile(`/\.superset/worktrees/(.+)$`)
	supersetPrj = regexp.MustCompile(`/\.superset/projects/([^/]+)`)
)

func baseName(p string) string {
	if p == "" {
		return "unknown"
	}
	if tmpDir.MatchString(p) { // macOS temp dirs end in a bare "T"
		return "tmp"
	}
	parts := strings.Split(strings.TrimRight(p, "/"), "/")
	if last := parts[len(parts)-1]; last != "" {
		return last
	}
	return "unknown"
}

func laneFromCwd(cwd string) *Lane {
	if cwd == "" {
		return nil
	}
	if strings.Contains(cwd, "/.paperclip/") {
		if m := boardSeat.FindStringSubmatch(cwd); m != nil {
			return &Lane{"board", m[1]}
		}
		return &Lane{"board", "board project"}
	}
	if m := supersetWT.FindStringSubmatch(cwd); m != nil {
		var wt []string
		for _, s := range strings.Split(m[1], "/") {
			if s != "" {
				wt = append(wt, s)
			}
		}
		if len(wt) > 1 {
			return &Lane{"sessions", wt[0] + " · " + wt[len(wt)-1]}
		}
		if len(wt) == 1 {
			return &Lane{"sessions", wt[0]}
		}
	}
	if m := supersetPrj.FindStringSubmatch(cwd); m != nil {
		return &Lane{"sessions", m[1]}
	}
	return nil
}

// ClassifyClaude: headless (sdk-*) runs outside the board are crons and
// scripts; anything else interactive is a terminal.
func ClassifyClaude(cwd, entrypoint string) Lane {
	if l := laneFromCwd(cwd); l != nil {
		return *l
	}
	if strings.HasPrefix(entrypoint, "sdk") {
		return Lane{"automation", baseName(cwd)}
	}
	return Lane{"terminal", baseName(cwd)}
}

// ClassifyCodex classifies a session from session_meta; a seat's own
// CODEX_HOME is the board whatever tree it was pointed at.
func ClassifyCodex(cwd, originator, source string, board bool) Lane {
	l := laneFromCwd(cwd)
	if board {
		if l != nil && l.Source == "board" {
			return *l
		}
		return Lane{"board", baseName(cwd)}
	}
	if l != nil {
		return *l
	}
	if originator == "codex_exec" || source == "exec" {
		return Lane{"automation", baseName(cwd)}
	}
	return Lane{"terminal", baseName(cwd)}
}

// ---- 10-minute slots -----------------------------------------------------------

const slotMS int64 = 10 * 60_000

// SlotMap is slot index (floor(ms / 10min)) → "source|label" → tokens.
type SlotMap map[int64]map[string]Tot4

func AddToSlots(slots SlotMap, ts string, lane Lane, t Tot4) {
	at, ok := parseTS(ts)
	if !ok {
		return
	}
	k := floorDiv(at.UnixMilli(), slotMS)
	if slots[k] == nil {
		slots[k] = map[string]Tot4{}
	}
	key := lane.Source + "|" + lane.Label
	cur := slots[k][key]
	addTot(&cur, t)
	slots[k][key] = cur
}

func floorDiv(a, b int64) int64 {
	q := a / b
	if (a%b != 0) && ((a < 0) != (b < 0)) {
		q--
	}
	return q
}

// PruneSlots drops slots older than the week (plus a day).
func PruneSlots(slots SlotMap, now time.Time) {
	floor := floorDiv(now.UnixMilli()-int64(UsageWindowDays+1)*86400_000, slotMS)
	for k := range slots {
		if k < floor {
			delete(slots, k)
		}
	}
}

func laneTot(l *devicepush.LaneTots, source string) *Tot4 {
	switch source {
	case "board":
		return &l.Board
	case "sessions":
		return &l.Sessions
	case "terminal":
		return &l.Terminal
	default:
		return &l.Automation
	}
}

// BreakdownFromSlots folds slot maps into the four windows by lane, plus the
// sixteen biggest burners of the week.
func BreakdownFromSlots(maps []SlotMap, now time.Time, days []devicepush.DayBucket) devicepush.Breakdown {
	var w devicepush.Windows
	hourFloor := floorDiv(now.UnixMilli()-3600_000, slotMS)
	sessionFloor := floorDiv(now.UnixMilli()-5*3600_000, slotMS)
	today := DayKey(now)
	week := map[string]bool{}
	for _, d := range days {
		week[d.Day] = true
	}
	byKey := map[string]*Tot4{}
	for _, m := range maps {
		for idx, lanes := range m {
			day := DayKey(time.UnixMilli(idx * slotMS))
			if !week[day] {
				continue
			}
			for key, t := range lanes {
				source := key[:strings.Index(key, "|")]
				if !isBurnSource(source) {
					continue
				}
				addTot(laneTot(&w.Week, source), t)
				if byKey[key] == nil {
					byKey[key] = &Tot4{}
				}
				addTot(byKey[key], t)
				if day == today {
					addTot(laneTot(&w.Day, source), t)
				}
				if idx >= sessionFloor {
					addTot(laneTot(&w.Session, source), t)
				}
				if idx >= hourFloor {
					addTot(laneTot(&w.Hour, source), t)
				}
			}
		}
	}
	top := []devicepush.TopBurner{}
	for key, t := range byKey {
		if b := BurnOf(*t); b > 0 {
			i := strings.Index(key, "|")
			top = append(top, devicepush.TopBurner{Source: key[:i], Label: key[i+1:], Burn: b})
		}
	}
	sort.SliceStable(top, func(i, j int) bool {
		if top[i].Burn != top[j].Burn {
			return top[i].Burn > top[j].Burn
		}
		return top[i].Source+top[i].Label < top[j].Source+top[j].Label
	})
	if len(top) > 16 {
		top = top[:16]
	}
	return devicepush.Breakdown{Windows: w, Top: top}
}

// ---- Codex session events -------------------------------------------------------

type CodexRateLimits struct {
	Session  *devicepush.OfficialWindow
	Weekly   *devicepush.OfficialWindow
	PlanType string
}

type ParsedCodexLine struct {
	TS string
	// Cumulative for the session, as Codex reports it: NOT a delta.
	Cumulative struct{ In, Out, CacheRead float64 }
	RateLimits *CodexRateLimits
}

func codexWindow(v any) *devicepush.OfficialWindow {
	o, _ := v.(map[string]any)
	pct, ok1 := o["used_percent"].(float64)
	mins, ok2 := o["window_minutes"].(float64)
	if !ok1 || !ok2 {
		return nil
	}
	w := &devicepush.OfficialWindow{UsedPercent: pct, WindowMinutes: int(mins)}
	if r, ok := o["resets_at"].(float64); ok {
		s := isoMillis(time.Unix(int64(r), 0))
		w.ResetsAt = &s
	}
	return w
}

// ParseCodexLine reads a token_count event: cumulative totals plus the plan's
// official gauge, assigned by window length (Codex has reported the weekly
// window as `primary` since 2026-09).
func ParseCodexLine(line string) *ParsedCodexLine {
	if strings.TrimSpace(line) == "" {
		return nil
	}
	var d map[string]any
	if json.Unmarshal([]byte(line), &d) != nil {
		return nil
	}
	payload, _ := d["payload"].(map[string]any)
	if d["type"] != "event_msg" || payload["type"] != "token_count" {
		return nil
	}
	info, _ := payload["info"].(map[string]any)
	t, _ := info["total_token_usage"].(map[string]any)
	if t == nil {
		return nil
	}
	p := &ParsedCodexLine{}
	p.TS, _ = d["timestamp"].(string)
	p.Cumulative.In, p.Cumulative.Out, p.Cumulative.CacheRead = num(t["input_tokens"]), num(t["output_tokens"]), num(t["cached_input_tokens"])
	if rl, ok := payload["rate_limits"].(map[string]any); ok {
		lim := &CodexRateLimits{}
		for _, w := range []*devicepush.OfficialWindow{codexWindow(rl["primary"]), codexWindow(rl["secondary"])} {
			if w == nil {
				continue
			}
			if float64(w.WindowMinutes) >= 7*24*60*0.9 {
				lim.Weekly = w
			} else {
				lim.Session = w
			}
		}
		lim.PlanType, _ = rl["plan_type"].(string)
		if lim.Session != nil || lim.Weekly != nil {
			p.RateLimits = lim
		}
	}
	return p
}

// ---- plan names ------------------------------------------------------------

var separators = regexp.MustCompile(`[_-]+`)

func isWord(r rune) bool {
	return r == '_' || r < unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r))
}

// titleCase is s.replace(/[_-]+/g,' ').replace(/\b\w/g, upper).
func titleCase(s string) string {
	s = separators.ReplaceAllString(s, " ")
	out := []rune(s)
	for i, r := range out {
		if isWord(r) && (i == 0 || !isWord(out[i-1])) {
			out[i] = unicode.ToUpper(r)
		}
	}
	return string(out)
}

var tierX = regexp.MustCompile(`(\d+)x`)

func strPtr(s string) *string { return &s }

// ClaudePlanName reads ~/.claude.json's oauthAccount (no token involved).
func ClaudePlanName(acc map[string]any) *string {
	typ, _ := acc["organizationType"].(string)
	if typ == "" {
		return nil
	}
	tier, _ := acc["organizationRateLimitTier"].(string)
	if typ == "claude_max" {
		if m := tierX.FindStringSubmatch(tier); m != nil {
			return strPtr(fmt.Sprintf("Claude Max %sx", m[1]))
		}
		return strPtr("Claude Max")
	}
	return strPtr("Claude " + titleCase(strings.TrimPrefix(typ, "claude_")))
}

var chatGPTPlans = map[string]string{"plus": "Plus", "pro": "Pro", "prolite": "Pro Lite", "team": "Team", "business": "Business", "enterprise": "Enterprise", "edu": "Edu", "free": "Free"}

// CodexPlanName reads Codex's rate_limits.plan_type.
func CodexPlanName(planType string) *string {
	if planType == "" {
		return nil
	}
	if n, ok := chatGPTPlans[planType]; ok {
		return strPtr("ChatGPT " + n)
	}
	return strPtr("ChatGPT " + titleCase(planType))
}

// OllamaPlanName reads the local server's POST /api/me `plan`.
func OllamaPlanName(plan string) *string {
	if plan == "" {
		return nil
	}
	return strPtr("Ollama " + titleCase(plan))
}

// ---- Ollama request counts -----------------------------------------------------

var (
	ginLine    = regexp.MustCompile(`^\[GIN\] (\d{4})/(\d{2})/(\d{2}) - (\d{2}):(\d{2}):(\d{2}) \|.*?"([^"?]+)`)
	chatPaths  = map[string]bool{"/api/chat": true, "/api/generate": true, "/v1/chat/completions": true, "/v1/completions": true, "/v1/responses": true}
	embedPaths = map[string]bool{"/api/embed": true, "/api/embeddings": true, "/v1/embeddings": true}
)

// CountOllamaRequests counts chat vs embedding requests per window from the
// server log's [GIN] lines (stamped in local time; Ollama logs no tokens).
func CountOllamaRequests(lines []string, now time.Time) devicepush.RequestWindows {
	var out devicepush.RequestWindows
	today := DayKey(now)
	for _, line := range lines {
		m := ginLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		var kind string
		switch {
		case chatPaths[m[7]]:
			kind = "chat"
		case embedPaths[m[7]]:
			kind = "embed"
		default:
			continue
		}
		when, err := time.ParseInLocation("2006/01/02 15:04:05", fmt.Sprintf("%s/%s/%s %s:%s:%s", m[1], m[2], m[3], m[4], m[5], m[6]), local)
		if err != nil {
			continue
		}
		age := now.Sub(when)
		if age < 0 || age > UsageWindowDays*24*time.Hour {
			continue
		}
		bump := func(c *devicepush.RequestCounts) {
			if kind == "chat" {
				c.Chat++
			} else {
				c.Embed++
			}
		}
		bump(&out.Week)
		if DayKey(when) == today {
			bump(&out.Day)
		}
		if age <= 5*time.Hour {
			bump(&out.Session)
		}
		if age <= time.Hour {
			bump(&out.Hour)
		}
	}
	return out
}
