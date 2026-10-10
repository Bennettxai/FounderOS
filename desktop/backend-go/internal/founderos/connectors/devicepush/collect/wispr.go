package collect

import (
	"context"
	"fmt"
	"math"
	"os"
	"sort"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/devicepush"
)

// WisprDetail is "7m ago" while the store is warm, "3h ago" and "10d ago"
// once it is not, with "stale on this box" past a day (lib/connectors/wispr.ts).
func WisprDetail(dictations, notes, todos, meetings int, minutesAgo float64) string {
	mins := int(math.Max(0, math.Floor(minutesAgo+0.5)))
	var age string
	switch {
	case mins < 60:
		age = fmt.Sprintf("%dm ago", mins)
	case mins < 1440:
		age = fmt.Sprintf("%dh ago", int(math.Floor(float64(mins)/60+0.5)))
	default:
		age = fmt.Sprintf("%dd ago", int(math.Floor(float64(mins)/1440+0.5)))
	}
	stale := ""
	if mins >= 1440 {
		stale = " · stale on this box"
	}
	return fmt.Sprintf("%s dictations · %d notes · %d todos · %d meetings · last activity %s%s",
		commas(dictations), notes, todos, meetings, age, stale)
}

// WisprStatus reads Wispr Flow's flow.sqlite read-only: History (dictations),
// Notes, Todos and Meetings.
func WisprStatus(ctx context.Context, dbPath string, now time.Time) connectors.Status {
	m := devicepush.Metas[devicepush.SourceWispr]
	s := connectors.Status{ID: m.ID, Name: m.Name, Kind: m.Kind}
	fi, err := os.Stat(dbPath)
	if err != nil {
		s.State = connectors.StateNotConfigured
		s.Detail = "flow.sqlite not found — is Wispr Flow installed?"
		return s
	}
	db, err := openReadOnly(ctx, dbPath)
	if err != nil {
		s.State = connectors.StateError
		s.Detail = "flow.sqlite exists but read failed: " + err.Error()
		return s
	}
	defer db.Close()
	count := func(table string) int {
		var n int
		// a table missing from an older schema counts as none, as in the TS
		if db.QueryRowContext(ctx, `SELECT COUNT(*) FROM "`+table+`"`).Scan(&n) != nil {
			return 0
		}
		return n
	}
	d, n, td, mt := count("History"), count("Notes"), count("Todos"), count("Meetings")
	s.State = connectors.StateConnected
	s.Detail = WisprDetail(d, n, td, mt, now.Sub(fi.ModTime()).Minutes())
	s.Meta = map[string]any{"dictations": d, "notes": n, "todos": td, "meetings": mt}
	return s
}

// RecentWisprNotes is the Comms "Notes" lane: dictations (the daily driver)
// with the app they went into, plus Notes and Todos when present. Each table
// read is guarded on its own, so an older schema skips a source instead of
// blanking the lane; a missing or unreadable file is an empty list.
func RecentWisprNotes(ctx context.Context, dbPath string, limit int) []devicepush.WisprNote {
	if limit <= 0 {
		limit = 40
	}
	out := []devicepush.WisprNote{}
	if _, err := os.Stat(dbPath); err != nil {
		return out
	}
	db, err := openReadOnly(ctx, dbPath)
	if err != nil {
		return out
	}
	defer db.Close()
	rows := func(query string) []map[string]any {
		r, err := rowMaps(ctx, db, query, limit)
		if err != nil {
			return nil
		}
		return r
	}

	for _, r := range rows(`SELECT transcriptEntityId, asrText, formattedText, editedText, timestamp, app, numWords, isArchived
	   FROM History WHERE COALESCE(isArchived,0)=0 ORDER BY timestamp DESC LIMIT ?`) {
		text := firstText(r["editedText"], r["formattedText"], r["asrText"])
		ts, ok := toISO(r["timestamp"])
		if text == "" || !ok {
			continue
		}
		n := devicepush.WisprNote{ID: jsString(r["transcriptEntityId"]), Kind: "dictation", Text: text, TS: ts}
		if app, isStr := asString(r["app"]); isStr && app != "" {
			n.App = &app
		}
		if w, isNum := asFloat(r["numWords"]); isNum {
			wc := int(w)
			n.WordCount = &wc
		}
		out = append(out, n)
	}

	for _, r := range rows(`SELECT id, title, contentPreview, createdAt, modifiedAt, isDeleted
	   FROM Notes WHERE COALESCE(isDeleted,0)=0 ORDER BY COALESCE(modifiedAt, createdAt) DESC LIMIT ?`) {
		text := firstText(r["title"], r["contentPreview"])
		at := r["modifiedAt"]
		if at == nil {
			at = r["createdAt"]
		}
		ts, ok := toISO(at)
		if text == "" || !ok {
			continue
		}
		out = append(out, devicepush.WisprNote{ID: jsString(r["id"]), Kind: "note", Text: text, TS: ts})
	}

	for _, r := range rows(`SELECT id, title, status, createdAt, isDeleted, isArchived
	   FROM Todos WHERE COALESCE(isDeleted,0)=0 AND COALESCE(isArchived,0)=0 ORDER BY createdAt DESC LIMIT ?`) {
		title := firstText(r["title"])
		ts, ok := toISO(r["createdAt"])
		if title == "" || !ok {
			continue
		}
		if status, _ := asString(r["status"]); status != "" && status != "open" {
			title += " · " + status
		}
		out = append(out, devicepush.WisprNote{ID: jsString(r["id"]), Kind: "todo", Text: title, TS: ts})
	}

	// WisprNoteSchema needs an id; rows without one are dropped like a failed parse.
	valid := out[:0]
	for _, n := range out {
		if n.ID != "" {
			valid = append(valid, n)
		}
	}
	sort.SliceStable(valid, func(i, j int) bool { return valid[i].TS > valid[j].TS })
	if len(valid) > limit {
		valid = valid[:limit]
	}
	return valid
}
