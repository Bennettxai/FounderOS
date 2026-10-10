package collect

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/devicepush"
)

// This machine's Claude seat, from the transcripts Claude Code already writes
// under ~/.claude/projects/**/*.jsonl (lib/connectors/claude-usage.ts). The
// scan is incremental: per file the byte offset of the last complete line is
// remembered with the running totals, and each pass reads only the appended
// tail. A file that SHRANK was rewritten and is rebuilt from zero.
//
// The official plan gauge is read separately (claude_official.go) and put on
// the seat by WithClaudeOfficial; without it the seat labels itself an
// estimate, as the TS does whenever no token is present.

type fileState struct {
	offset      int64
	size        int64
	perDay      map[string]*Tot4
	perDayModel map[string]map[string]*Tot4
	slots       SlotMap
	lastTS      string
}

func newFileState() *fileState {
	return &fileState{perDay: map[string]*Tot4{}, perDayModel: map[string]map[string]*Tot4{}, slots: SlotMap{}}
}

// ScanCache carries fold state between scans; a long-running collector keeps
// one so each tick reads only new bytes.
type ScanCache struct {
	mu    sync.Mutex
	files map[string]*fileState
}

func NewScanCache() *ScanCache { return &ScanCache{files: map[string]*fileState{}} }

// SeatInfo names a seat. ConfigJSON is ~/.claude.json, for the plan name.
type SeatInfo struct {
	ID         string
	Label      string
	ConfigJSON string
}

func listJSONL(dir string, maxDepth int, newerThan time.Time) []string {
	var out []string
	var walk func(d string, depth int)
	walk = func(d string, depth int) {
		if depth > maxDepth {
			return
		}
		entries, err := os.ReadDir(d)
		if err != nil {
			return
		}
		for _, e := range entries {
			p := filepath.Join(d, e.Name())
			if e.IsDir() {
				walk(p, depth+1)
			} else if e.Type().IsRegular() && strings.HasSuffix(e.Name(), ".jsonl") {
				if fi, err := e.Info(); err == nil && !fi.ModTime().Before(newerThan) {
					out = append(out, p)
				}
			}
		}
	}
	walk(dir, 0)
	return out
}

func foldTail(file string, st *fileState) error {
	fi, err := os.Stat(file)
	if err != nil {
		return err
	}
	size := fi.Size()
	if size < st.size { // rewritten shorter: everything folded is suspect
		*st = *newFileState()
	}
	if size == st.offset {
		st.size = size
		return nil
	}
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Seek(st.offset, io.SeekStart); err != nil {
		return err
	}
	r := bufio.NewReaderSize(io.LimitReader(f, size-st.offset), 1<<20)
	consumed := st.offset
	for {
		line, err := r.ReadString('\n')
		if !strings.HasSuffix(line, "\n") {
			break // partial final line: fold it next pass, once complete
		}
		consumed += int64(len(line))
		if strings.Contains(line, `"type":"assistant"`) { // cheap reject first
			if p := ParseClaudeLine(strings.TrimRight(line, "\r\n")); p != nil {
				foldLine(st, p)
			}
		}
		if err != nil {
			break
		}
	}
	st.offset = consumed
	st.size = size
	return nil
}

func foldLine(st *fileState, p *ParsedUsageLine) {
	at, ok := parseTS(p.TS)
	if !ok {
		return
	}
	dk := DayKey(at)
	if st.perDay[dk] == nil {
		st.perDay[dk] = &Tot4{}
	}
	addTot(st.perDay[dk], p.Tot4)
	if st.perDayModel[dk] == nil {
		st.perDayModel[dk] = map[string]*Tot4{}
	}
	if st.perDayModel[dk][p.Model] == nil {
		st.perDayModel[dk][p.Model] = &Tot4{}
	}
	addTot(st.perDayModel[dk][p.Model], p.Tot4)
	AddToSlots(st.slots, p.TS, ClassifyClaude(p.Cwd, p.Entrypoint), p.Tot4)
	if st.lastTS == "" || p.TS > st.lastTS {
		st.lastTS = p.TS
	}
}

// ScanClaudeProjects is the local Claude seat: burn by day and model across
// every project on this machine, incremental after the first pass.
func ScanClaudeProjects(dir string, now time.Time, cache *ScanCache, info SeatInfo) devicepush.SeatUsage {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	files := listJSONL(dir, 3, now.Add(-time.Duration(UsageWindowDays+1)*24*time.Hour))
	keep := map[string]bool{}
	for _, f := range files {
		keep[f] = true
		st := cache.files[f]
		if st == nil {
			st = newFileState()
		}
		if foldTail(f, st) == nil { // unreadable file: skip, keep prior state
			cache.files[f] = st
		}
	}
	for k := range cache.files { // files out of the mtime window are evicted
		if !keep[k] {
			delete(cache.files, k)
		}
	}

	days := EmptyDays(now, UsageWindowDays)
	byModel := map[string]Tot4{}
	var lastTS string
	var slotMaps []SlotMap
	for _, st := range cache.files {
		for i := range days {
			if b := st.perDay[days[i].Day]; b != nil {
				addTot(&days[i].Tot, *b)
			}
			// the model split honors the same window as the day columns
			for m, t := range st.perDayModel[days[i].Day] {
				if t.In+t.Out+t.CacheWrite+t.CacheRead == 0 {
					continue
				}
				agg := byModel[m]
				addTot(&agg, *t)
				byModel[m] = agg
			}
		}
		if st.lastTS != "" && st.lastTS > lastTS {
			lastTS = st.lastTS
		}
		PruneSlots(st.slots, now)
		slotMaps = append(slotMaps, st.slots)
	}

	seat := devicepush.SeatUsage{
		ID: info.ID, Kind: "claude", Label: info.Label, Source: "local",
		CapturedAt: isoMillis(now), Days: days, ByModel: byModel,
		Note: noteClaudeEstimate,
	}
	if lastTS != "" {
		seat.LastActivity = &lastTS
	}
	b := BreakdownFromSlots(slotMaps, now, days)
	seat.Breakdown = &b
	if info.ConfigJSON != "" {
		seat.Plan = ClaudeAccountPlan(info.ConfigJSON)
	}
	if len(files) == 0 {
		seat.Note = "no transcripts found — is this the right CLAUDE_PROJECTS_DIR?"
	}
	return seat
}

// ClaudeAccountPlan is the plan this machine's Claude login is on, from
// ~/.claude.json's oauthAccount; nil when there is no login here.
func ClaudeAccountPlan(file string) *string {
	raw, err := os.ReadFile(file)
	if err != nil {
		return nil
	}
	var doc struct {
		OauthAccount map[string]any `json:"oauthAccount"`
	}
	if json.Unmarshal(raw, &doc) != nil {
		return nil
	}
	return ClaudePlanName(doc.OauthAccount)
}
