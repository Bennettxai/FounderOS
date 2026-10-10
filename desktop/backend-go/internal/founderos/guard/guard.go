// Package guard keeps the FounderOS bridge from double-firing while live prod
// still runs. Until cutover, FOUNDEROS_WRITES and FOUNDEROS_CRONS are off: no
// message, post, DM, trade, payment or calendar write leaves the bridge, and
// no scheduled agent runs. Every refusal is recorded so the bridge can show
// what it would have done.
package guard

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// ErrWritesDisabled is returned for any side effect refused by the guard.
var ErrWritesDisabled = errors.New("bridge: outbound writes are disabled (FOUNDEROS_WRITES=0)")

// WritesEnabled reports whether outbound side effects may leave the bridge.
func WritesEnabled() bool { return os.Getenv("FOUNDEROS_WRITES") == "1" }

// CronsEnabled reports whether scheduled agents may run.
func CronsEnabled() bool { return os.Getenv("FOUNDEROS_CRONS") == "1" }

type Refusal struct {
	Action string    `json:"action"`
	At     time.Time `json:"at"`
}

const maxRefusals = 500

var (
	mu      sync.Mutex
	refused []Refusal
)

func record(action string) {
	mu.Lock()
	defer mu.Unlock()
	refused = append(refused, Refusal{Action: action, At: time.Now().UTC()})
	if len(refused) > maxRefusals {
		refused = refused[len(refused)-maxRefusals:]
	}
	slog.Info("bridge guard refused", "action", action)
}

// Refused returns the most recent refusals, oldest first.
func Refused() []Refusal {
	mu.Lock()
	defer mu.Unlock()
	return append([]Refusal(nil), refused...)
}

// Reset clears the refusal log.
func Reset() {
	mu.Lock()
	refused = nil
	mu.Unlock()
}

// Outbound runs fn only when writes are enabled.
func Outbound(action string, fn func() error) error {
	if !WritesEnabled() {
		record(action)
		return ErrWritesDisabled
	}
	return fn()
}

// RunCron runs a scheduled job only when crons are enabled and reports
// whether it ran. The job's own error is logged, not returned: a cron tick
// never fails its scheduler.
func RunCron(name string, fn func() error) bool {
	if !CronsEnabled() {
		record("cron:" + name)
		return false
	}
	if err := fn(); err != nil {
		slog.Error("bridge cron failed", "cron", name, "err", err)
	}
	return true
}

// Transport wraps base (http.DefaultTransport when nil) so that, while writes
// are off, any mutating request to a non-local host is refused before it is
// sent. readPOSTs lists "host/path-prefix" entries for read APIs that happen
// to use POST (query and search endpoints).
func Transport(base http.RoundTripper, readPOSTs ...string) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return &transport{base: base, readPOSTs: readPOSTs}
}

type transport struct {
	base      http.RoundTripper
	readPOSTs []string
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	if WritesEnabled() || safeMethod(req.Method) || isLocal(req.URL.Hostname()) || t.allowedRead(req) {
		return t.base.RoundTrip(req)
	}
	record(fmt.Sprintf("%s %s%s", req.Method, req.URL.Host, req.URL.Path))
	if req.Body != nil {
		req.Body.Close()
	}
	return nil, ErrWritesDisabled
}

func (t *transport) allowedRead(req *http.Request) bool {
	if req.Method != http.MethodPost {
		return false
	}
	target := req.URL.Host + req.URL.Path
	for _, prefix := range t.readPOSTs {
		// An empty prefix (e.g. from an unparsable URL) must allow nothing.
		if prefix != "" && strings.HasPrefix(target, prefix) {
			return true
		}
	}
	return false
}

func safeMethod(m string) bool {
	// PROPFIND and REPORT are WebDAV/CalDAV reads.
	return m == http.MethodGet || m == http.MethodHead || m == http.MethodOptions || m == "PROPFIND" || m == "REPORT"
}

func isLocal(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
