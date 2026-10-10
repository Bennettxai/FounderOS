package sales

import (
	"context"
	"errors"
	"fmt"
	"github.com/rhl/businessos-backend/internal/founderos/guard"
	"strings"

	"github.com/rhl/businessos-backend/internal/founderos/agents"
	"github.com/rhl/businessos-backend/internal/founderos/agents/roster"
	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/fathomcalls"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/plaud"
	"github.com/rhl/businessos-backend/internal/founderos/topology"
)

// CallsData is Sales Calls Data: Fathom on the calls, Plaud in the room. A
// run reports both recorders and files every newly transcribed Plaud
// recording into memory (the cron-plaud-ingest-30m work). The Fathom call
// archive is a separate, long pass (ArchiveCalls), as in FounderOS v1 where it
// runs from POST /api/calls/archive rather than on every agent run.
type CallsData struct {
	FathomConfigured func() bool
	Plaud            connectors.Connector
	Ingest           *PlaudIngester
	Archive          *CallArchiver
	// setupErr explains a missing seam (no topology), for the summary.
	setupErr error
}

// NewCallsData wires the live seams from Deps.
func NewCallsData(d roster.Deps, p *plaud.Connector, f *fathomcalls.Connector) *CallsData {
	social, clue, _, _ := connectors.CredFiles()
	c := &CallsData{
		FathomConfigured: func() bool { return d.Res.Resolve("FATHOM_API_KEY", clue, social) != "" },
		Plaud:            p,
		Ingest:           &PlaudIngester{Plaud: p},
		Archive:          &CallArchiver{Fathom: f},
	}
	if d.Memory != nil { // never store a typed nil in the interface
		c.Ingest.Memory, c.Archive.Memory = d.Memory, d.Memory
	}
	if d.Pool != nil {
		ws := d.Workspaces[LedgerWorkspace]
		c.Ingest.Ledger = NewPgPlaudLedger(d.Pool, ws)
		c.Archive.Ledger = NewPgArchiveLedger(d.Pool, ws)
	}
	if topo, err := topology.LoadRepo(); err != nil {
		c.setupErr = fmt.Errorf("engine topology: %w", err)
	} else {
		c.Ingest.Route, c.Archive.Route = topo.WorkspaceForBrainPage, topo.WorkspaceForBrainPage
	}
	return c
}

func (c *CallsData) Meta() agents.Meta { return metaCallsData }

func intp(n int) *int { return &n }

func (c *CallsData) Run(ctx context.Context) (agents.Result, error) {
	fathom := "not_configured"
	if c.FathomConfigured() {
		fathom = "configured"
	}
	ps := c.Plaud.Status(ctx)
	plaudUp := ps.State == connectors.StateConnected
	liveCount := 0
	if fathom == "configured" {
		liveCount++
	}
	if plaudUp {
		liveCount++
	}

	// Unknown stays nil in Data: a count nobody read is not 0.
	var recordings, filed, inBrain, waiting, failed, claims *int
	var ingestErr error
	var detail string
	switch {
	case plaudUp:
		if n, ok := ps.Meta["recordings"].(int); ok {
			recordings = intp(n)
		}
		r, err := c.Ingest.Run(ctx)
		if err != nil {
			ingestErr = err
			if c.setupErr != nil {
				ingestErr = fmt.Errorf("%w (%v)", err, c.setupErr)
			}
		} else {
			filed, waiting, failed, claims = intp(len(r.Ingested)), intp(len(r.Skipped.NotTranscribed)), intp(len(r.Failed)), intp(r.Claims)
			inBrain = intp(len(r.Skipped.AlreadyIngested) + len(r.Ingested))
		}
		detail = plaudDetail(recordings, r, err == nil, ingestErr)
	case ps.State == connectors.StateError:
		detail = " (" + ps.Detail + ")"
	}

	summary := fmt.Sprintf("Recorders: Fathom %s · Plaud %s%s", fathom, ps.State, detail)
	if liveCount == 0 {
		summary += " — set FATHOM_API_KEY and PLAUD_REFRESH_TOKEN to capture calls and in-person meetings"
	}
	ok := liveCount > 0 && ps.State != connectors.StateError && ingestErr == nil && (failed == nil || *failed == 0)
	data := map[string]any{
		"fathom": fathom, "plaud": string(ps.State), "recordings": recordings,
		"filed": filed, "inBrain": inBrain, "waiting": waiting, "failed": failed, "claims": claims,
	}
	if ingestErr != nil {
		data["ingestError"] = ingestErr.Error()
	}
	return agents.Result{OK: ok, Summary: summary, Data: data}, nil
}

func plaudDetail(recordings *int, r IngestResult, ran bool, ingestErr error) string {
	var parts []string
	if recordings != nil {
		s := "s"
		if *recordings == 1 {
			s = ""
		}
		parts = append(parts, fmt.Sprintf("%d recording%s", *recordings, s))
	}
	if ran {
		parts = append(parts,
			fmt.Sprintf("%d in brain", len(r.Skipped.AlreadyIngested)+len(r.Ingested)),
			fmt.Sprintf("%d filed this pass", len(r.Ingested)),
			fmt.Sprintf("%d awaiting transcription", len(r.Skipped.NotTranscribed)))
		if r.Claims > 0 {
			parts = append(parts, fmt.Sprintf("%d claims to OptimalEngine", r.Claims))
		}
		if len(r.Failed) > 0 {
			errs := make([]string, len(r.Failed))
			for i, f := range r.Failed {
				errs[i] = f.Error
			}
			parts = append(parts, fmt.Sprintf("%d FAILED: %s", len(r.Failed), strings.Join(errs, "; ")))
		}
		if len(r.ClaimFailures) > 0 {
			parts = append(parts, fmt.Sprintf("%d claim captures failed", len(r.ClaimFailures)))
		}
	}
	if ingestErr != nil {
		if errors.Is(ingestErr, guard.ErrWritesDisabled) {
			parts = append(parts, "ingest refused: writes are off on the bridge (prod files recordings until the flip)")
		} else {
			parts = append(parts, "ingest FAILED: "+ingestErr.Error())
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return " (" + strings.Join(parts, ", ") + ")"
}

// ArchiveCalls runs one Fathom call-archive pass (POST /api/calls/archive).
func (c *CallsData) ArchiveCalls(ctx context.Context) (ArchiveResult, error) {
	if c.Archive == nil {
		return emptyArchive(), errors.New("no call archiver wired")
	}
	res, err := c.Archive.Run(ctx)
	if err != nil && c.setupErr != nil && c.Archive.Route == nil {
		err = fmt.Errorf("%w (%v)", err, c.setupErr)
	}
	return res, err
}
