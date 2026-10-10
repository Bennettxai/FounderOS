// Package connectors is the FounderOS connector base: the honest status
// contract of FounderOS v1's lib/connectors, its credential resolution order
// (lib/creds.ts), and an HTTP client that goes through the bridge guard.
package connectors

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/guard"
)

type State string

const (
	StateConnected     State = "connected"
	StateNotConfigured State = "not_configured"
	StateError         State = "error"
)

type Kind string

const (
	KindEmail         Kind = "email"
	KindCalendar      Kind = "calendar"
	KindSlack         Kind = "slack"
	KindPayments      Kind = "payments"
	KindNotion        Kind = "notion"
	KindBrain         Kind = "brain"
	KindSocial        Kind = "social"
	KindCRM           Kind = "crm"
	KindAds           Kind = "ads"
	KindCreative      Kind = "creative"
	KindKnowledge     Kind = "knowledge"
	KindLocal         Kind = "local"
	KindOrchestration Kind = "orchestration"
)

// Status is one row of the Connections board; the JSON shape matches the operator
// OS's GET /api/connections.
type Status struct {
	ID     string         `json:"id"`
	Name   string         `json:"name"`
	Kind   Kind           `json:"kind"`
	State  State          `json:"state"`
	Detail string         `json:"detail"`
	Meta   map[string]any `json:"meta,omitempty"`
}

// Meta identifies a connector at registration.
type Meta struct {
	ID   string
	Name string
	Kind Kind
}

// Connector reports its status. Status must never write anywhere and must
// report unreachable as StateError, never as connected or empty.
type Connector interface {
	Status(ctx context.Context) Status
}

type entry struct {
	meta Meta
	c    Connector
}

type Registry struct {
	mu      sync.RWMutex
	entries []entry
}

func NewRegistry() *Registry { return &Registry{} }

// Register adds a connector; a duplicate id is a programming error.
func (r *Registry) Register(m Meta, c Connector) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, e := range r.entries {
		if e.meta.ID == m.ID {
			panic(fmt.Sprintf("connectors: duplicate id %q", m.ID))
		}
	}
	r.entries = append(r.entries, entry{meta: m, c: c})
}

// Statuses checks every connector concurrently. A connector that has not
// answered when ctx ends reads as error / timed out.
func (r *Registry) Statuses(ctx context.Context) []Status {
	r.mu.RLock()
	entries := append([]entry(nil), r.entries...)
	r.mu.RUnlock()

	out := make([]Status, len(entries))
	var wg sync.WaitGroup
	for i, e := range entries {
		wg.Add(1)
		go func(i int, e entry) {
			defer wg.Done()
			out[i] = check(ctx, e)
		}(i, e)
	}
	wg.Wait()
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// StatusOf checks one connector by id (FounderOS v1 connectorStatusById), with
// the same timeout and honesty rules as Statuses.
func (r *Registry) StatusOf(ctx context.Context, id string) (Status, bool) {
	r.mu.RLock()
	var found *entry
	for i := range r.entries {
		if r.entries[i].meta.ID == id {
			e := r.entries[i]
			found = &e
			break
		}
	}
	r.mu.RUnlock()
	if found == nil {
		return Status{}, false
	}
	return check(ctx, *found), true
}

func check(ctx context.Context, e entry) Status {
	done := make(chan Status, 1)
	go func() {
		// This goroutine is out of reach of gin's recovery: a panicking
		// connector must be one red row, not a crashed backend.
		defer func() {
			if p := recover(); p != nil {
				done <- Status{State: StateError, Detail: fmt.Sprintf("status check panicked: %v", p)}
			}
		}()
		done <- e.c.Status(ctx)
	}()
	var s Status
	select {
	case s = <-done:
	case <-ctx.Done():
		s = Status{State: StateError, Detail: "status check timed out"}
	}
	if s.State == "" {
		s = Status{State: StateError, Detail: "status check returned nothing"}
	}
	s.ID, s.Name, s.Kind = e.meta.ID, e.meta.Name, e.meta.Kind
	return s
}

// HTTPClient returns a client whose transport refuses outbound writes while
// FOUNDEROS_WRITES is off. readPOSTs lists read-only POST endpoints to allow.
func HTTPClient(timeout time.Duration, readPOSTs ...string) *http.Client {
	return &http.Client{Timeout: timeout, Transport: guard.Transport(nil, readPOSTs...)}
}

// ---- credentials (mirrors lib/creds.ts) -----------------------------------

var envKey = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// ParseEnvFile parses KEY=value lines the way FounderOS v1 does.
func ParseEnvFile(content string) map[string]string {
	out := map[string]string{}
	for _, raw := range strings.Split(content, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		eq := strings.Index(line, "=")
		if eq <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:eq])
		if !envKey.MatchString(key) {
			continue
		}
		val := strings.TrimSpace(line[eq+1:])
		if len(val) >= 2 && ((val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'')) {
			val = val[1 : len(val)-1]
		}
		out[key] = val
	}
	return out
}

func readEnvFile(path string) map[string]string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return map[string]string{}
	}
	return ParseEnvFile(string(raw))
}

// Resolver finds a credential: keys planted on the bridge (Planted, via
// /api/admin/keys) first, then a fresh read of EnvLocal (so a key added while
// running takes effect at once), then the process env, then each file in
// order. Secrets are read at runtime and never copied.
type Resolver struct {
	Planted  string
	EnvLocal string
}

func (r Resolver) Resolve(name string, files ...string) string {
	if r.Planted != "" {
		if v := readEnvFile(r.Planted)[name]; v != "" {
			return v
		}
	}
	if r.EnvLocal != "" {
		if v := readEnvFile(r.EnvLocal)[name]; v != "" {
			return v
		}
	}
	if v := os.Getenv(name); v != "" {
		return v
	}
	for _, f := range files {
		if v := readEnvFile(f)[name]; v != "" {
			return v
		}
	}
	return ""
}

// DefaultResolver reads planted keys (FOUNDEROS_PLANTED_ENV, else
// ~/.founderos/planted.env) and the operator's env file (FOUNDEROS_ENV_LOCAL,
// else ~/.founderos/.env). Nothing else in the home directory is read.
func DefaultResolver() Resolver {
	home, _ := os.UserHomeDir()
	r := Resolver{
		Planted:  filepath.Join(home, ".founderos", "planted.env"),
		EnvLocal: filepath.Join(home, ".founderos", ".env"),
	}
	if p := os.Getenv("FOUNDEROS_PLANTED_ENV"); p != "" {
		r.Planted = p
	}
	if p := os.Getenv("FOUNDEROS_ENV_LOCAL"); p != "" {
		r.EnvLocal = p
	}
	return r
}

// CredFiles names extra credential files a connector may fall back to. FounderOS
// never reads other tools' credential files (their .env files, ~/.claude.json),
// so there are none: every key comes through the Resolver.
func CredFiles() (socialMedia, clueAgent, arcads, claudeJSON string) {
	return "", "", "", ""
}

// McpEnvKey reads mcpServers.<server>.env.<key> from a ~/.claude.json.
func McpEnvKey(claudeJSON, server, key string) string {
	if claudeJSON == "" {
		return ""
	}
	raw, err := os.ReadFile(claudeJSON)
	if err != nil {
		return ""
	}
	var doc struct {
		McpServers map[string]struct {
			Env map[string]string `json:"env"`
		} `json:"mcpServers"`
	}
	if json.Unmarshal(raw, &doc) != nil {
		return ""
	}
	return doc.McpServers[server].Env[key]
}
