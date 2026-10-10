package collect

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/devicepush"
)

// The Codex plan lane, from the rollouts the codex CLI writes under
// ~/.codex/sessions/YYYY/MM/DD/rollout-*.jsonl (lib/connectors/codex-usage.ts).
// Its token_count events embed the plan's OFFICIAL gauge, as fresh as the
// last CLI run. Counters are cumulative per session, so a session adds its
// final total to the day of its last event. Files are parsed once and
// memoized by (size, mtime).

type sessionSummary struct {
	ts         string
	in, out    float64
	cacheRead  float64
	rateLimits *CodexRateLimits
	slots      SlotMap
}

type codexEntry struct {
	size    int64
	modTime time.Time
	summary *sessionSummary
}

// CodexCache memoizes parsed session files.
type CodexCache struct {
	mu    sync.Mutex
	files map[string]codexEntry
}

func NewCodexCache() *CodexCache { return &CodexCache{files: map[string]codexEntry{}} }

// PaperclipCodexHomes are the board seats' own CODEX_HOMEs:
// <root>/instances/<instance>/companies/<company>/codex-home/sessions.
func PaperclipCodexHomes(root string) []string {
	ls := func(d string) []string {
		entries, err := os.ReadDir(d)
		if err != nil {
			return nil
		}
		var out []string
		for _, e := range entries {
			if e.IsDir() {
				out = append(out, filepath.Join(d, e.Name()))
			}
		}
		return out
	}
	var out []string
	for _, inst := range ls(filepath.Join(root, "instances")) {
		for _, co := range ls(filepath.Join(inst, "companies")) {
			sessions := filepath.Join(co, "codex-home", "sessions")
			if _, err := os.Stat(sessions); err == nil {
				out = append(out, sessions)
			}
		}
	}
	return out
}

func (c *CodexCache) summarize(file string, board bool) *sessionSummary {
	fi, err := os.Stat(file)
	if err != nil {
		return nil
	}
	if hit, ok := c.files[file]; ok && hit.size == fi.Size() && hit.modTime.Equal(fi.ModTime()) {
		return hit.summary
	}
	raw, err := os.ReadFile(file)
	if err != nil {
		return nil
	}
	var events []*ParsedCodexLine
	var cwd, originator, source string
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.Contains(line, `"session_meta"`) {
			var d struct {
				Payload map[string]any `json:"payload"`
			}
			if json.Unmarshal([]byte(line), &d) == nil { // malformed meta: terminal lane
				cwd, _ = d.Payload["cwd"].(string)
				originator, _ = d.Payload["originator"].(string)
				source, _ = d.Payload["source"].(string)
			}
			continue
		}
		if !strings.Contains(line, "token_count") {
			continue
		}
		if p := ParseCodexLine(line); p != nil {
			events = append(events, p)
		}
	}
	if len(events) == 0 {
		return nil
	}
	s := &sessionSummary{slots: SlotMap{}}
	lane := ClassifyCodex(cwd, originator, source, board)
	var prev struct{ In, Out, CacheRead float64 }
	for _, e := range events {
		if e.TS != "" { // codexSessionFold: last ts, last totals, last gauge
			s.ts = e.TS
		}
		s.in, s.out, s.cacheRead = e.Cumulative.In, e.Cumulative.Out, e.Cumulative.CacheRead
		if e.RateLimits != nil {
			s.rateLimits = e.RateLimits
		}
		d := Tot4{
			In:        math.Max(0, e.Cumulative.In-prev.In),
			Out:       math.Max(0, e.Cumulative.Out-prev.Out),
			CacheRead: math.Max(0, e.Cumulative.CacheRead-prev.CacheRead),
		}
		prev = e.Cumulative
		if e.TS != "" && d.In+d.Out+d.CacheRead > 0 {
			AddToSlots(s.slots, e.TS, lane, d)
		}
	}
	c.files[file] = codexEntry{size: fi.Size(), modTime: fi.ModTime(), summary: s}
	return s
}

// CodexSeat is the whole Codex lane, or nil when no sessions exist at all.
// dirs are personal session roots; boardDirs are paperclip seats' homes.
func CodexSeat(dirs, boardDirs []string, now time.Time, cache *CodexCache, info SeatInfo) *devicepush.SeatUsage {
	type file struct {
		path  string
		board bool
	}
	var files []file
	for _, d := range dirs {
		for _, f := range listJSONL(d, 4, time.Time{}) {
			files = append(files, file{f, false})
		}
	}
	for _, d := range boardDirs {
		for _, f := range listJSONL(d, 4, time.Time{}) {
			files = append(files, file{f, true})
		}
	}
	if len(files) == 0 {
		return nil
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()

	days := EmptyDays(now, UsageWindowDays)
	var slotMaps []SlotMap
	var newest *sessionSummary
	var lastActivity string
	for _, f := range files {
		s := cache.summarize(f.path, f.board)
		if s == nil || s.ts == "" {
			continue
		}
		if s.ts > lastActivity {
			lastActivity = s.ts
		}
		slotMaps = append(slotMaps, s.slots)
		if s.rateLimits != nil && (newest == nil || s.ts > newest.ts) {
			newest = s
		}
		at, ok := parseTS(s.ts)
		if !ok {
			continue
		}
		for i := range days {
			if days[i].Day == DayKey(at) {
				days[i].In += s.in
				days[i].Out += s.out
				days[i].CacheRead += s.cacheRead
			}
		}
	}

	seat := &devicepush.SeatUsage{
		ID: info.ID, Kind: "codex", Label: info.Label, Source: "local",
		Days: days, ByModel: map[string]Tot4{},
		Note: "no rate-limit events found in recent sessions",
	}
	switch {
	case newest != nil:
		seat.CapturedAt = newest.ts
	case lastActivity != "":
		seat.CapturedAt = lastActivity
	default:
		seat.CapturedAt = isoMillis(now)
	}
	if lastActivity != "" {
		seat.LastActivity = &lastActivity
	}
	if newest != nil {
		seat.Official = &devicepush.Official{Session: newest.rateLimits.Session, Weekly: newest.rateLimits.Weekly}
		seat.Plan = CodexPlanName(newest.rateLimits.PlanType)
		seat.Note = "official plan gauge, as of the last codex run"
	}
	b := BreakdownFromSlots(slotMaps, now, days)
	seat.Breakdown = &b
	return seat
}
