package sales

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"testing"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/plaud"
	"github.com/rhl/businessos-backend/internal/founderos/memory"
	"github.com/rhl/businessos-backend/internal/founderos/topology"
)

// fakeMemory records every capture; fail, when set, decides per capture.
type fakeMemory struct {
	mu       sync.Mutex
	captured []memory.Capture
	fail     func(memory.Capture) error
}

func (m *fakeMemory) Capture(_ context.Context, c memory.Capture) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.fail != nil {
		if err := m.fail(c); err != nil {
			return "", err
		}
	}
	m.captured = append(m.captured, c)
	return fmt.Sprintf("sig_%d", len(m.captured)), nil
}

func (m *fakeMemory) genre(g string) []memory.Capture {
	var out []memory.Capture
	for _, c := range m.captured {
		if c.Genre == g {
			out = append(out, c)
		}
	}
	return out
}

type fakePlaudLedger struct {
	rows     map[string]PlaudIngest
	notReady error
	insErr   error
	readErr  error
}

func newPlaudLedger() *fakePlaudLedger { return &fakePlaudLedger{rows: map[string]PlaudIngest{}} }

func (l *fakePlaudLedger) Ready(context.Context) error { return l.notReady }

func (l *fakePlaudLedger) Ingested(context.Context) (map[string]bool, error) {
	if l.readErr != nil {
		return nil, l.readErr
	}
	out := map[string]bool{}
	for id := range l.rows {
		out[id] = true
	}
	return out, nil
}

func (l *fakePlaudLedger) Insert(_ context.Context, r PlaudIngest) error {
	if l.insErr != nil {
		return l.insErr
	}
	l.rows[r.FileID] = r
	return nil
}

type fakeArchiveLedger struct {
	rows    map[string]ArchiveEntry
	readErr error
	recErr  error
}

func newArchiveLedger() *fakeArchiveLedger {
	return &fakeArchiveLedger{rows: map[string]ArchiveEntry{}}
}

func (l *fakeArchiveLedger) Archived(context.Context) (map[string]ArchiveEntry, error) {
	if l.readErr != nil {
		return nil, l.readErr
	}
	out := map[string]ArchiveEntry{}
	for k, v := range l.rows {
		out[k] = v
	}
	return out, nil
}

func (l *fakeArchiveLedger) Record(_ context.Context, e ArchiveEntry) error {
	if l.recErr != nil {
		return l.recErr
	}
	l.rows[e.RecordingID] = e
	return nil
}

func (l *fakeArchiveLedger) ids() []string {
	var out []string
	for k := range l.rows {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// fakePlaud serves a fixed list and files; listErr/fileErr inject failures.
type fakePlaud struct {
	list    []plaud.Recording
	files   map[string]*plaud.File
	listErr error
	fileErr map[string]error
	gate    chan struct{} // when set, listing waits for it
	entered chan struct{} // when set, closed as listing starts
}

func (p *fakePlaud) RecentRecordings(_ context.Context, limit int) ([]plaud.Recording, error) {
	if p.entered != nil {
		close(p.entered)
		p.entered = nil
	}
	if p.gate != nil {
		<-p.gate
	}
	if p.listErr != nil {
		return nil, p.listErr
	}
	if len(p.list) > limit {
		return p.list[:limit], nil
	}
	return p.list, nil
}

func (p *fakePlaud) File(_ context.Context, id string) (*plaud.File, error) {
	if err := p.fileErr[id]; err != nil {
		return nil, err
	}
	return p.files[id], nil
}

func repoTopology(t *testing.T) *topology.Topology {
	t.Helper()
	topo, err := topology.LoadRepo()
	if err != nil {
		t.Fatal(err)
	}
	return topo
}

var errBoom = errors.New("boom")
