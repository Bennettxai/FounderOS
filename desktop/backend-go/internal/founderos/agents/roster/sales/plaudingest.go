package sales

import (
	"context"
	"errors"
	"fmt"
	"github.com/rhl/businessos-backend/internal/founderos/guard"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/unicode/norm"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/plaud"
	"github.com/rhl/businessos-backend/internal/founderos/memory"
)

// Plaud → memory: the port of FounderOS v1 lib/plaud-ingest.ts. Every
// recording Plaud has synced AND transcribed becomes one markdown page (facts
// header, Plaud's AI summary, the timestamped speaker transcript) captured
// into Optimal Engine through the memory router; the AI note's action items,
// verbatim, follow as a second "claims" capture. No LLM anywhere: Plaud did
// the transcription and the summary; this files them.
//
// Idempotent by Plaud file id through founderos_plaud_ingests. A recording that
// cannot be captured stays un-ingested and is retried next pass. There is no
// brain-store fallback: the engine is the only memory.

// Capturer is the memory write seam (*memory.Router satisfies it).
type Capturer interface {
	Capture(ctx context.Context, c memory.Capture) (string, error)
}

// PlaudRecordings is the Plaud read seam (*plaud.Connector satisfies it).
type PlaudRecordings interface {
	RecentRecordings(ctx context.Context, limit int) ([]plaud.Recording, error)
	File(ctx context.Context, id string) (*plaud.File, error)
}

// PlaudIngest is one founderos_plaud_ingests row.
type PlaudIngest struct {
	FileID     string
	Title      string
	RecordedAt string // ISO, "" when Plaud gave none
	IngestedAt time.Time
	Via        string // "oe" for every bridge write
	Slug       string
	Claims     int
	Workspace  string // the OE workspace the page went to
	SignalID   string
}

// PlaudLedger is the idempotency ledger.
type PlaudLedger interface {
	// Ready fails when the ledger cannot record a bridge ingest; the pass
	// then captures nothing, so no page can land unrecorded and be re-sent.
	Ready(ctx context.Context) error
	Ingested(ctx context.Context) (map[string]bool, error)
	Insert(ctx context.Context, row PlaudIngest) error
}

// ViaOE marks a recording filed into Optimal Engine by the bridge.
const ViaOE = "oe"

// plaudListLimit is how many recent recordings one pass scans (as the TS).
const plaudListLimit = 100

type IngestedRecording struct {
	FileID    string `json:"fileId"`
	Title     string `json:"title"`
	Workspace string `json:"workspace"`
	Slug      string `json:"slug"`
	SignalID  string `json:"signalId"`
	Claims    int    `json:"claims"`
}

type IngestFailure struct {
	FileID string `json:"fileId"`
	Error  string `json:"error"`
}

type IngestSkipped struct {
	AlreadyIngested []string `json:"alreadyIngested"`
	NotTranscribed  []string `json:"notTranscribed"`
	Sample          []string `json:"sample"`
}

type IngestResult struct {
	Scanned  int                 `json:"scanned"`
	Ingested []IngestedRecording `json:"ingested"`
	Skipped  IngestSkipped       `json:"skipped"`
	Failed   []IngestFailure     `json:"failed"`
	Claims   int                 `json:"claims"`
	// ClaimFailures are pages that landed whose action-item capture did not.
	ClaimFailures []IngestFailure `json:"claimFailures"`
}

// PlaudIngester runs one ingest pass.
type PlaudIngester struct {
	Plaud  PlaudRecordings
	Memory Capturer
	Ledger PlaudLedger
	// Route maps a brain-store path and page text to an OE workspace
	// (topology.WorkspaceForBrainPage).
	Route func(rel, text string) string
	Now   func() time.Time
}

// Run files every newly transcribed recording. A pass that cannot run at all
// (no memory, no ledger, Plaud unreadable) is an error, never an empty result.
func (p *PlaudIngester) Run(ctx context.Context) (IngestResult, error) {
	res := IngestResult{
		Ingested: []IngestedRecording{}, Failed: []IngestFailure{}, ClaimFailures: []IngestFailure{},
		Skipped: IngestSkipped{AlreadyIngested: []string{}, NotTranscribed: []string{}, Sample: []string{}},
	}
	// Until the flip prod owns what is filed from Plaud: engine captures
	// and ledger rows made here would fight the cutover ETL and brain import
	// (duplicates). FOUNDEROS_WRITES=0 refuses the whole pass, and logs it.
	if err := guard.Outbound("plaud.ingest", func() error { return nil }); err != nil {
		return res, err
	}
	if !plaudPass.tryStart() {
		return res, ErrIngestRunning
	}
	defer plaudPass.done()
	switch {
	case p.Memory == nil:
		return res, errors.New("no memory router on this bridge")
	case p.Ledger == nil:
		return res, errors.New("no plaud ingest ledger (Postgres) on this bridge")
	case p.Route == nil:
		return res, errors.New("no engine topology loaded")
	}
	if err := p.Ledger.Ready(ctx); err != nil {
		return res, err
	}
	done, err := p.Ledger.Ingested(ctx)
	if err != nil {
		return res, fmt.Errorf("read plaud ingest ledger: %w", err)
	}
	recordings, err := p.Plaud.RecentRecordings(ctx, plaudListLimit)
	if err != nil {
		return res, fmt.Errorf("list Plaud recordings: %w", err)
	}
	res.Scanned = len(recordings)
	now := time.Now
	if p.Now != nil {
		now = p.Now
	}

	for _, rec := range recordings {
		if plaud.IsSample(rec.Title) {
			res.Skipped.Sample = append(res.Skipped.Sample, rec.ID)
			continue
		}
		if done[rec.ID] {
			res.Skipped.AlreadyIngested = append(res.Skipped.AlreadyIngested, rec.ID)
			continue
		}
		file, err := p.Plaud.File(ctx, rec.ID)
		if err != nil {
			res.Failed = append(res.Failed, IngestFailure{FileID: rec.ID, Error: err.Error()})
			continue
		}
		if file == nil || !file.Transcribed {
			res.Skipped.NotTranscribed = append(res.Skipped.NotTranscribed, rec.ID)
			continue
		}

		slug := RecordingSlug(file.Title, file.At, file.ID)
		md := RenderRecordingMarkdown(file)
		ws := p.Route("inbox/recordings/"+slug+".md", md)
		signal, err := p.Memory.Capture(ctx, memory.Capture{Workspace: ws, Title: file.Title, Text: md, Genre: "transcript", Node: "inbox"})
		if err != nil {
			res.Failed = append(res.Failed, IngestFailure{FileID: file.ID, Error: err.Error()})
			continue
		}

		claims := 0
		if items := plaud.ActionItems(file.Note); len(items) > 0 {
			lines := []string{fmt.Sprintf("Action items from %q (Plaud recording, %s):", file.Title, day(file.At))}
			for _, it := range items {
				lines = append(lines, "- "+it)
			}
			_, cerr := p.Memory.Capture(ctx, memory.Capture{Workspace: ws, Title: "Action items: " + file.Title, Text: strings.Join(lines, "\n"), Genre: "claims", Node: "inbox"})
			if cerr != nil {
				res.ClaimFailures = append(res.ClaimFailures, IngestFailure{FileID: file.ID, Error: cerr.Error()})
			} else {
				claims = len(items)
			}
		}

		row := PlaudIngest{
			FileID: file.ID, Title: file.Title, RecordedAt: file.At, IngestedAt: now().UTC(),
			Via: ViaOE, Slug: slug, Claims: claims, Workspace: ws, SignalID: signal,
		}
		if err := p.Ledger.Insert(ctx, row); err != nil {
			// The page is in the engine but not recorded: say exactly that.
			res.Failed = append(res.Failed, IngestFailure{FileID: file.ID, Error: fmt.Sprintf("captured as %s but the ingest ledger refused it: %v", signal, err)})
			continue
		}
		res.Ingested = append(res.Ingested, IngestedRecording{FileID: file.ID, Title: file.Title, Workspace: ws, Slug: slug, SignalID: signal, Claims: claims})
		res.Claims += claims
	}
	return res, nil
}

var isoDay = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}`)

func day(at string) string {
	if isoDay.MatchString(at) {
		return at[:10]
	}
	return ""
}

// RecordingSlug is "<YYYY-MM-DD|undated>-<slugified title>".
func RecordingSlug(title, at, id string) string {
	d := day(at)
	if d == "" {
		d = "undated"
	}
	s := Slugify(title)
	if s == "" {
		s = id[:min(8, len(id))]
	}
	return d + "-" + s
}

// Slugify ports lib/brain-dump.ts slugifyTitle: lowercase, NFKD (so accents
// fall away), keep ASCII word characters, spaces and dashes, dash-join.
func Slugify(title string) string {
	var b strings.Builder
	for _, r := range norm.NFKD.String(strings.ToLower(title)) {
		switch {
		case r < unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-'):
			b.WriteRune(r)
		case unicode.IsSpace(r):
			b.WriteRune(' ')
		}
	}
	var out strings.Builder
	sep := false
	for _, r := range strings.TrimSpace(b.String()) {
		if r == ' ' || r == '_' || r == '-' {
			sep = true
			continue
		}
		if sep && out.Len() > 0 {
			out.WriteByte('-')
		}
		sep = false
		out.WriteRune(r)
	}
	if out.Len() == 0 {
		return "untitled"
	}
	return out.String()
}

func mmss(ms float64) string {
	s := max(0, int(ms/1000))
	return fmt.Sprintf("%02d:%02d", s/60, s%60)
}

// RenderRecordingMarkdown is the page for one recording: facts header, the
// AI summary, then the timestamped speaker transcript.
func RenderRecordingMarkdown(f *plaud.File) string {
	lines := []string{"# " + f.Title, "", "source: plaud", "plaud_file_id: " + f.ID}
	if f.At != "" {
		lines = append(lines, "recorded: "+f.At[:min(10, len(f.At))])
	}
	if f.DurationMinutes != nil {
		lines = append(lines, "duration: "+strconv.FormatFloat(*f.DurationMinutes, 'f', -1, 64)+"m")
	}
	lines = append(lines, "")
	if f.Note != nil && strings.TrimSpace(*f.Note) != "" {
		lines = append(lines, "## Summary", "", strings.TrimSpace(*f.Note), "")
	}
	if len(f.Transcript) > 0 {
		lines = append(lines, "## Transcript", "")
		for _, seg := range f.Transcript {
			who := ""
			if seg.Speaker != nil {
				who = *seg.Speaker + ": "
			}
			lines = append(lines, fmt.Sprintf("[%s - %s] %s%s", mmss(seg.StartMs), mmss(seg.EndMs), who, seg.Text))
		}
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}
