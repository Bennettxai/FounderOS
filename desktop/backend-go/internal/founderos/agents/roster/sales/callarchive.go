package sales

import (
	"context"
	"errors"
	"fmt"
	"github.com/rhl/businessos-backend/internal/founderos/guard"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/fathomcalls"
	"github.com/rhl/businessos-backend/internal/founderos/memory"
)

// The call archive: the port of FounderOS v1 lib/call-archive.ts. Every
// recorded Fathom call becomes one markdown page (the same page the TS wrote
// into brain-store meetings/), captured into Optimal Engine through the
// memory router and routed by the topology's meeting rules, so the knowledge
// base keeps the conversations after the recorder's subscription ends. Pure
// code, no LLM.
//
// Idempotent by recording id through an ArchiveLedger: a call whose entry
// carries the current ArchiveVersion is never re-sent, a call without a
// transcript is left for the next pass, and Fathom's demo call is skipped.
// The TS also deleted a demo page an earlier pass had written; the memory
// router has no delete, and the bridge never writes one, so there is nothing
// to remove here.

// ArchiveVersion is the page layout version; ledger entries with an older
// one are re-captured (the TS rewrote such pages).
const ArchiveVersion = 3

var versionLine = fmt.Sprintf("archive_version: %d", ArchiveVersion)

// ErrArchiveRunning is returned while another archive pass holds the lock
// (a full pass is minutes of paged API calls).
var ErrArchiveRunning = errors.New("a call archive pass is already running")

// FathomMeetings is the Fathom read seam (*fathomcalls.Connector satisfies it).
type FathomMeetings interface {
	MeetingsFull(ctx context.Context, visit func(page []fathomcalls.MeetingFull) error) error
}

// ArchiveEntry is one archived call.
type ArchiveEntry struct {
	RecordingID string    `json:"-"`
	Version     int       `json:"v"`
	Workspace   string    `json:"workspace"`
	SignalID    string    `json:"signal,omitempty"` // "" when brain import landed it
	Page        string    `json:"page"`             // the brain-store path it would have had
	Title       string    `json:"title,omitempty"`
	ArchivedAt  time.Time `json:"archivedAt"`
}

// ArchiveLedger records which calls are archived.
type ArchiveLedger interface {
	Archived(ctx context.Context) (map[string]ArchiveEntry, error)
	Record(ctx context.Context, e ArchiveEntry) error
}

type ArchiveFailure struct {
	ID    string `json:"id"`
	Error string `json:"error"`
}

// ArchiveResult matches the TS ArchiveResult; Exported holds brain-store
// style paths so the counts and names read the same as prod's.
type ArchiveResult struct {
	Source          string           `json:"source"`
	MeetingsScanned int              `json:"meetingsScanned"`
	Found           int              `json:"found"`
	Exported        []string         `json:"exported"`
	SkippedExisting []string         `json:"skippedExisting"`
	SkippedSample   []string         `json:"skippedSample"`
	NoTranscript    []string         `json:"noTranscript"`
	Failed          []ArchiveFailure `json:"failed"`
}

// CallArchiver runs archive passes, one at a time.
type CallArchiver struct {
	Fathom FathomMeetings
	Memory Capturer
	Ledger ArchiveLedger
	Route  func(rel, text string) string
	Now    func() time.Time
}

// Running reports whether an archive pass is in flight anywhere in the
// process (archivePass).
func (a *CallArchiver) Running() bool { return archivePass.busy() }

func emptyArchive() ArchiveResult {
	return ArchiveResult{Source: "fathom", Exported: []string{}, SkippedExisting: []string{}, SkippedSample: []string{}, NoTranscript: []string{}, Failed: []ArchiveFailure{}}
}

// Run archives every new transcribed call. A pass that cannot start is an
// error; a walk that fails part-way returns what landed plus the error.
func (a *CallArchiver) Run(ctx context.Context) (ArchiveResult, error) {
	res := emptyArchive()
	// As the Plaud ingest: prod owns the archive until the flip.
	if err := guard.Outbound("calls.archive", func() error { return nil }); err != nil {
		return res, err
	}
	if !archivePass.tryStart() {
		return res, ErrArchiveRunning
	}
	defer archivePass.done()

	switch {
	case a.Memory == nil:
		return res, errors.New("no memory router on this bridge")
	case a.Ledger == nil:
		return res, errors.New("no call archive ledger (Postgres) on this bridge")
	case a.Route == nil:
		return res, errors.New("no engine topology loaded")
	}
	have, err := a.Ledger.Archived(ctx)
	if err != nil {
		return res, fmt.Errorf("read call archive ledger: %w", err)
	}
	now := time.Now
	if a.Now != nil {
		now = a.Now
	}

	err = a.Fathom.MeetingsFull(ctx, func(page []fathomcalls.MeetingFull) error {
		res.MeetingsScanned += len(page)
		for _, m := range page {
			id := m.ID()
			if id == "" {
				ref := m.URL
				if ref == "" {
					ref = "?"
				}
				res.Failed = append(res.Failed, ArchiveFailure{ID: ref, Error: "no recording id"})
				continue
			}
			res.Found++
			if fathomcalls.IsSample(m) {
				res.SkippedSample = append(res.SkippedSample, id)
				continue
			}
			if e, ok := have[id]; ok && e.Version == ArchiveVersion {
				res.SkippedExisting = append(res.SkippedExisting, id)
				continue
			}
			if !m.HasTranscript() {
				res.NoTranscript = append(res.NoTranscript, id)
				continue
			}
			text := RenderFathomCallPage(m)
			rel := "meetings/" + archivePageName(m.At(), m.Title(), id)
			ws := a.Route(rel, text)
			signal, err := a.Memory.Capture(ctx, memory.Capture{Workspace: ws, Title: m.Title(), Text: text, Genre: "transcript", Node: "knowledge-base"})
			if err != nil {
				res.Failed = append(res.Failed, ArchiveFailure{ID: id, Error: err.Error()})
				continue
			}
			entry := ArchiveEntry{RecordingID: id, Version: ArchiveVersion, Workspace: ws, SignalID: signal, Page: rel, Title: m.Title(), ArchivedAt: now().UTC()}
			if err := a.Ledger.Record(ctx, entry); err != nil {
				res.Failed = append(res.Failed, ArchiveFailure{ID: id, Error: fmt.Sprintf("captured as %s but the archive ledger refused it: %v", signal, err)})
				continue
			}
			have[id] = entry
			res.Exported = append(res.Exported, rel)
		}
		return nil
	})
	return res, err
}

func archivePageName(at, title, id string) string {
	d := day(at)
	if d == "" {
		d = "undated"
	}
	return fmt.Sprintf("%s-%s--fathom-%s.md", d, Slugify(title), id)
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// RenderFathomCallPage is the call's page: YAML frontmatter, facts, summary,
// action items, then the merged transcript.
func RenderFathomCallPage(m fathomcalls.MeetingFull) string {
	title, at := m.Title(), m.At()
	safe := strings.ReplaceAll(strings.ReplaceAll(title, `\`, `\\`), `"`, `\"`)
	lines := []string{"---", `title: "` + safe + `"`, "type: meeting"}
	if d := day(at); d != "" {
		lines = append(lines, "date: "+d)
	}
	lines = append(lines, "source: fathom", "---", "")
	lines = append(lines, "# "+title, "", versionLine, "source: fathom", "fathom_recording_id: "+m.ID())
	if m.URL != "" {
		lines = append(lines, "url: "+m.URL)
	}
	if at != "" {
		lines = append(lines, "recorded: "+at)
	}
	if d := m.DurationMinutes(); d != nil {
		lines = append(lines, fmt.Sprintf("duration: %dm", *d))
	}
	if rb := m.RecordedBy; rb != nil && (deref(rb.Name) != "" || deref(rb.Email) != "") {
		lines = append(lines, strings.Replace(fmt.Sprintf("recorded_by: %s <%s>", deref(rb.Name), deref(rb.Email)), " <>", "", 1))
	}
	var invitees []string
	for _, i := range m.CalendarInvitees {
		switch {
		case deref(i.Name) != "":
			invitees = append(invitees, fmt.Sprintf("%s <%s>", *i.Name, deref(i.Email)))
		case deref(i.Email) != "":
			invitees = append(invitees, *i.Email)
		}
	}
	if len(invitees) > 0 {
		lines = append(lines, "invitees: "+strings.Join(invitees, ", "))
	}
	lines = append(lines, "")
	if m.DefaultSummary != nil {
		if s := strings.TrimSpace(m.DefaultSummary.MarkdownFormatted); s != "" {
			lines = append(lines, "## Summary", "", s, "")
		}
	}
	var actions []string
	for _, ai := range m.ActionItems {
		desc := strings.TrimSpace(ai.Description)
		if desc == "" {
			continue
		}
		var who []string
		if ai.Assignee != nil && deref(ai.Assignee.Name) != "" {
			who = append(who, *ai.Assignee.Name)
		}
		if ai.RecordingTimestamp != "" {
			who = append(who, ai.RecordingTimestamp)
		}
		box := " "
		if ai.Completed {
			box = "x"
		}
		line := fmt.Sprintf("- [%s] %s", box, desc)
		if len(who) > 0 {
			line += " (" + strings.Join(who, ", ") + ")"
		}
		actions = append(actions, line)
	}
	if len(actions) > 0 {
		lines = append(lines, "## Action items", "")
		lines = append(lines, actions...)
		lines = append(lines, "")
	}
	if t := fathomcalls.TranscriptLines(m); len(t) > 0 {
		lines = append(lines, "## Transcript", "")
		lines = append(lines, t...)
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

var fathomPageRe = regexp.MustCompile(`--fathom-(.+)\.md$`)

// SeedArchiveLedgerFromStore records the Fathom pages a brain-store already
// holds at the current archive version, so a bridge pass never re-captures a
// call the brain import already landed in the engine. storeDir is passed in
// (the backend knows no home paths). It returns how many entries it added.
func SeedArchiveLedgerFromStore(ctx context.Context, ledger ArchiveLedger, storeDir string, route func(rel, text string) string, now time.Time) (int, error) {
	dir := filepath.Join(storeDir, "meetings")
	names, err := os.ReadDir(dir)
	if err != nil {
		return 0, fmt.Errorf("read %s: %w", dir, err)
	}
	have, err := ledger.Archived(ctx)
	if err != nil {
		return 0, err
	}
	added := 0
	for _, n := range names {
		sub := fathomPageRe.FindStringSubmatch(n.Name())
		if n.IsDir() || sub == nil {
			continue
		}
		id := sub[1]
		if e, ok := have[id]; ok && e.Version == ArchiveVersion {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, n.Name()))
		if err != nil || !strings.Contains(string(raw), "\n"+versionLine+"\n") {
			continue
		}
		rel := "meetings/" + n.Name()
		e := ArchiveEntry{RecordingID: id, Version: ArchiveVersion, Workspace: route(rel, string(raw)), Page: rel, ArchivedAt: now.UTC()}
		if err := ledger.Record(ctx, e); err != nil {
			return added, err
		}
		added++
	}
	return added, nil
}
