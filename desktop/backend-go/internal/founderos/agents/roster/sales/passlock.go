package sales

import (
	"errors"
	"sync"
)

// passLock lets one pass of a kind run at a time across the whole process.
// Each caller (the cron, a manual Run, POST /pages/plaud/ingest,
// POST /api/calls/archive) builds its own ingester or archiver, so a lock on
// the instance never saw the others; the engine gives every capture a fresh
// id, so two overlapping passes file the same recording twice.
type passLock struct {
	mu      sync.Mutex
	running bool
}

func (l *passLock) tryStart() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.running {
		return false
	}
	l.running = true
	return true
}

func (l *passLock) done() {
	l.mu.Lock()
	l.running = false
	l.mu.Unlock()
}

func (l *passLock) busy() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.running
}

var (
	plaudPass   passLock
	archivePass passLock
)

// ErrIngestRunning is returned while another Plaud ingest pass is in flight.
var ErrIngestRunning = errors.New("a Plaud ingest pass is already running")
