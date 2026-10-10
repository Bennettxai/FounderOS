package sales

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/rhl/businessos-backend/internal/founderos/agents/roster"
	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/fathomcalls"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/plaud"
	"github.com/rhl/businessos-backend/internal/founderos/memory"
)

func callsData(t *testing.T, fathomOn bool, plaudStatus connectors.Status) (*CallsData, *ingestFixture) {
	t.Helper()
	f := newIngestFixture(t)
	return &CallsData{
		FathomConfigured: func() bool { return fathomOn },
		Plaud:            fakeStatus(plaudStatus),
		Ingest:           f.ing,
	}, f
}

func connected(n int) connectors.Status {
	return connectors.Status{State: connectors.StateConnected, Detail: "Plaud reachable", Meta: map[string]any{"recordings": n}}
}

func TestCallsDataFilesPlaudRecordings(t *testing.T) {
	c, f := callsData(t, true, connected(2))
	res := run(t, c)
	want := "Recorders: Fathom configured · Plaud connected (2 recordings, 1 in brain, 1 filed this pass, 1 awaiting transcription, 2 claims to OptimalEngine)"
	if !res.OK || res.Summary != want {
		t.Fatalf("got  %q\nwant %q (ok=%v)", res.Summary, want, res.OK)
	}
	d := res.Data.(map[string]any)
	if *d["filed"].(*int) != 1 || *d["inBrain"].(*int) != 1 || d["fathom"] != "configured" || d["plaud"] != "connected" {
		t.Fatalf("data %#v", d)
	}
	if len(f.ledger.rows) != 1 {
		t.Fatal("run must record the ingest")
	}

	// Second run: nothing new, still honest.
	res = run(t, c)
	if !res.OK || !strings.Contains(res.Summary, "1 in brain, 0 filed this pass") || strings.Contains(res.Summary, "claims to OptimalEngine") {
		t.Fatalf("%q", res.Summary)
	}
}

func TestCallsDataWithAFailedRecordingIsNotOK(t *testing.T) {
	c, f := callsData(t, true, connected(1))
	f.mem.fail = func(memory.Capture) error { return errors.New("engine hub unreachable") }
	res := run(t, c)
	if res.OK || !strings.Contains(res.Summary, "1 FAILED: engine hub unreachable") {
		t.Fatalf("%+v", res)
	}
}

func TestCallsDataPlaudRefreshRefusedIsAnHonestFailure(t *testing.T) {
	// With FOUNDEROS_WRITES=0 the Plaud connector will not rotate an expired
	// token that prod shares; that reads as a failed run, even with Fathom up.
	detail := "Plaud credential is set but the call failed: Plaud access token expired; the bridge will not rotate the refresh token prod shares (FOUNDEROS_WRITES=0)"
	c, f := callsData(t, true, connectors.Status{State: connectors.StateError, Detail: detail})
	res := run(t, c)
	if res.OK || res.Summary != "Recorders: Fathom configured · Plaud error ("+detail+")" {
		t.Fatalf("%+v", res)
	}
	if d := res.Data.(map[string]any); d["recordings"].(*int) != nil || d["inBrain"].(*int) != nil {
		t.Fatalf("unknown counts must read unknown, not 0: %#v", d)
	}
	if len(f.mem.captured) != 0 {
		t.Fatal("no ingest without a connected Plaud")
	}
}

func TestCallsDataNothingConfigured(t *testing.T) {
	c, _ := callsData(t, false, connectors.Status{State: connectors.StateNotConfigured, Detail: "Set PLAUD_REFRESH_TOKEN"})
	res := run(t, c)
	if res.OK || res.Summary != "Recorders: Fathom not_configured · Plaud not_configured — set FATHOM_API_KEY and PLAUD_REFRESH_TOKEN to capture calls and in-person meetings" {
		t.Fatalf("%+v", res)
	}
}

func TestCallsDataFathomOnlyIsOK(t *testing.T) {
	c, _ := callsData(t, true, connectors.Status{State: connectors.StateNotConfigured})
	res := run(t, c)
	if !res.OK || res.Summary != "Recorders: Fathom configured · Plaud not_configured" {
		t.Fatalf("%+v", res)
	}
}

func TestCallsDataIngestPassErrorIsNotOK(t *testing.T) {
	c, f := callsData(t, true, connected(2))
	f.ledger.notReady = errors.New("founderos_plaud_ingests refuses via='oe'")
	res := run(t, c)
	if res.OK || !strings.Contains(res.Summary, "ingest FAILED: founderos_plaud_ingests refuses via='oe'") {
		t.Fatalf("%+v", res)
	}
	if d := res.Data.(map[string]any); d["inBrain"].(*int) != nil {
		t.Fatalf("in-brain is unknown when the pass could not run: %#v", d)
	}
}

func TestCallsDataArchiveCalls(t *testing.T) {
	c, _ := callsData(t, true, connected(0))
	if _, err := c.ArchiveCalls(context.Background()); err == nil {
		t.Fatal("no archiver wired must be an error")
	}
	a := newArchiveFixture(t)
	c.Archive = a.arch
	r, err := c.ArchiveCalls(context.Background())
	if err != nil || len(r.Exported) != 1 {
		t.Fatalf("r=%+v err=%v", r, err)
	}
}

func TestNewCallsDataWithoutMemoryOrPostgresFailsHonestly(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "1") // past the guard, to the missing seams
	p := plaud.New(connectors.Resolver{})
	c := NewCallsData(roster.Deps{}, p, fathomcalls.New(connectors.Resolver{}))
	c.Plaud = fakeStatus(connected(1))
	c.Ingest.Plaud = &fakePlaud{}
	res := run(t, c)
	if res.OK || !strings.Contains(res.Summary, "ingest FAILED: no memory router") {
		t.Fatalf("%+v", res)
	}
	if _, err := c.ArchiveCalls(context.Background()); err == nil || !strings.Contains(err.Error(), "memory") {
		t.Fatalf("archive without memory: %v", err)
	}
}

func TestCallsDataSaysTheIngestIsRefusedWhileWritesAreOff(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "0")
	c := NewCallsData(roster.Deps{}, plaud.New(connectors.Resolver{}), fathomcalls.New(connectors.Resolver{}))
	c.Plaud = fakeStatus(connected(1))
	c.Ingest.Plaud = &fakePlaud{}
	res := run(t, c)
	if !strings.Contains(res.Summary, "ingest refused: writes are off") || strings.Contains(res.Summary, "FAILED") {
		t.Fatalf("%+v", res.Summary)
	}
}
