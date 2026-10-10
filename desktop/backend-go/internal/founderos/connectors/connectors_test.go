package connectors

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/guard"
)

func write(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestParseEnvFileMatchesFounderosOS(t *testing.T) {
	got := ParseEnvFile("# comment\nexport A=1\nB = \"two words\"\nC='x'\nbad key=1\n=nokey\nD=a=b\n")
	want := map[string]string{"A": "1", "B": "two words", "C": "x", "D": "a=b"}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %q, want %q", k, got[k], v)
		}
	}
}

func TestResolveOrderEnvLocalThenProcessThenFiles(t *testing.T) {
	dir := t.TempDir()
	envLocal := write(t, dir, "env.local", "TOKEN=from-env-local\n")
	social := write(t, dir, "social.env", "TOKEN=from-social\nONLY_FILE=file-value\n")
	clue := write(t, dir, "clue.env", "ONLY_FILE=second-file\nLATE=clue\n")
	r := Resolver{EnvLocal: envLocal}

	t.Setenv("TOKEN", "from-process")
	if got := r.Resolve("TOKEN", social); got != "from-env-local" {
		t.Errorf("env.local must win, got %q", got)
	}
	r.EnvLocal = filepath.Join(dir, "missing")
	if got := r.Resolve("TOKEN", social); got != "from-process" {
		t.Errorf("process env must beat files, got %q", got)
	}
	if got := r.Resolve("ONLY_FILE", social, clue); got != "file-value" {
		t.Errorf("first file must win, got %q", got)
	}
	if got := r.Resolve("LATE", social, clue); got != "clue" {
		t.Errorf("later files are consulted, got %q", got)
	}
	if got := r.Resolve("NOPE", social, filepath.Join(dir, "absent.env")); got != "" {
		t.Errorf("unknown key must be empty, got %q", got)
	}
}

func TestMcpEnvKeyFromClaudeJSON(t *testing.T) {
	dir := t.TempDir()
	p := write(t, dir, "claude.json", `{"mcpServers":{"manychat":{"env":{"MANYCHAT_API_KEY":"mc-123"}}}}`)
	if got := McpEnvKey(p, "manychat", "MANYCHAT_API_KEY"); got != "mc-123" {
		t.Errorf("got %q", got)
	}
	if got := McpEnvKey(p, "nope", "X"); got != "" {
		t.Errorf("missing server must be empty, got %q", got)
	}
}

type fake struct {
	st    Status
	delay time.Duration
}

func (f fake) Status(ctx context.Context) Status {
	select {
	case <-time.After(f.delay):
		return f.st
	case <-ctx.Done():
		return Status{}
	}
}

func TestRegistryStatusesAreHonestAndBounded(t *testing.T) {
	r := NewRegistry()
	r.Register(Meta{ID: "slack", Name: "Slack", Kind: KindSlack}, fake{st: Status{State: StateConnected, Detail: "3 channels"}})
	r.Register(Meta{ID: "stripe", Name: "Stripe", Kind: KindPayments}, fake{st: Status{State: StateNotConfigured, Detail: "STRIPE_SECRET_KEY missing"}})
	r.Register(Meta{ID: "slow", Name: "Slow", Kind: KindLocal}, fake{delay: time.Second})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	all := r.Statuses(ctx)
	if len(all) != 3 {
		t.Fatalf("got %d statuses", len(all))
	}
	byID := map[string]Status{}
	for _, s := range all {
		byID[s.ID] = s
	}
	if s := byID["slack"]; s.State != StateConnected || s.Name != "Slack" || s.Kind != KindSlack {
		t.Errorf("slack = %+v (meta must be filled from registration)", s)
	}
	if s := byID["slow"]; s.State != StateError || !strings.Contains(s.Detail, "timed out") {
		t.Errorf("a connector that does not answer must read as error/timed out, never connected or empty: %+v", s)
	}
	raw, _ := json.Marshal(byID["stripe"])
	if !strings.Contains(string(raw), `"state":"not_configured"`) || !strings.Contains(string(raw), `"kind":"payments"`) {
		t.Errorf("JSON shape must match FounderOS v1 /api/connections: %s", raw)
	}
}

func TestRegistryRejectsDuplicateIDs(t *testing.T) {
	r := NewRegistry()
	r.Register(Meta{ID: "x", Name: "X", Kind: KindLocal}, fake{})
	defer func() {
		if recover() == nil {
			t.Fatal("duplicate id must panic at registration")
		}
	}()
	r.Register(Meta{ID: "x", Name: "X2", Kind: KindLocal}, fake{})
}

func TestHTTPClientIsGuarded(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "0")
	c := HTTPClient(5 * time.Second)
	req, _ := http.NewRequest(http.MethodPost, "https://api.example.com/send", strings.NewReader("{}"))
	if _, err := c.Do(req); !errors.Is(err, guard.ErrWritesDisabled) {
		t.Fatalf("connector client must refuse external POSTs while writes are off, got %v", err)
	}
}

func TestPlantedKeysWinOverEnvLocal(t *testing.T) {
	dir := t.TempDir()
	planted := write(t, dir, "planted.env", "TOKEN=planted\n")
	envLocal := write(t, dir, "env.local", "TOKEN=from-env-local\nOTHER=local\n")
	r := Resolver{Planted: planted, EnvLocal: envLocal}
	if got := r.Resolve("TOKEN"); got != "planted" {
		t.Errorf("planted must win, got %q", got)
	}
	if got := r.Resolve("OTHER"); got != "local" {
		t.Errorf("unplanted keys fall through to env.local, got %q", got)
	}
}

func TestDefaultResolverPlantedPath(t *testing.T) {
	t.Setenv("FOUNDEROS_PLANTED_ENV", "/tmp/x-planted.env")
	if got := DefaultResolver().Planted; got != "/tmp/x-planted.env" {
		t.Fatalf("Planted = %q", got)
	}
	t.Setenv("FOUNDEROS_PLANTED_ENV", "")
	if got := DefaultResolver().Planted; !strings.HasSuffix(got, "/.founderos/planted.env") {
		t.Fatalf("default Planted = %q", got)
	}
}

// One key's "test" runs one connector, not the whole board (FounderOS v1
// connectorStatusById).
func TestRegistryStatusOfChecksOneConnector(t *testing.T) {
	r := NewRegistry()
	r.Register(Meta{ID: "slack", Name: "Slack", Kind: KindSlack}, fake{st: Status{State: StateConnected, Detail: "ok"}})
	r.Register(Meta{ID: "slow", Name: "Slow", Kind: KindLocal}, fake{delay: time.Second})
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	s, ok := r.StatusOf(ctx, "slack")
	if !ok || s.ID != "slack" || s.Name != "Slack" || s.State != StateConnected {
		t.Fatalf("got %+v %v", s, ok)
	}
	if _, ok := r.StatusOf(ctx, "nope"); ok {
		t.Fatal("unknown id resolved")
	}
	s, _ = r.StatusOf(ctx, "slow")
	if s.State != StateError {
		t.Fatalf("a check that outlives ctx reads error, got %+v", s)
	}
}

type panicky struct{}

func (panicky) Status(context.Context) Status { var m map[string]int; m["x"]++; return Status{} }

// A connector that panics is one red row, never a crashed backend: the check
// runs on its own goroutine, out of reach of gin's recovery.
func TestAPanickingConnectorIsAnErrorRow(t *testing.T) {
	r := NewRegistry()
	r.Register(Meta{ID: "bad", Name: "Bad", Kind: KindLocal}, panicky{})
	r.Register(Meta{ID: "good", Name: "Good", Kind: KindLocal}, fake{st: Status{State: StateConnected}})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	got := r.Statuses(ctx)
	if len(got) != 2 || got[0].ID != "bad" || got[0].State != StateError || !strings.Contains(got[0].Detail, "panicked") {
		t.Fatalf("statuses = %+v", got)
	}
	if got[1].State != StateConnected {
		t.Fatalf("the healthy row must survive: %+v", got[1])
	}
	if s, ok := r.StatusOf(ctx, "bad"); !ok || s.State != StateError {
		t.Fatalf("StatusOf = %+v", s)
	}
}
