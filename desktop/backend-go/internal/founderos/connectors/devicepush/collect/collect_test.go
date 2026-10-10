package collect

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/devicepush"
)

var collectEnv = []string{
	"FOUNDEROS_COLLECTOR_DEVICE", "FOUNDEROS_COLLECTOR_LABEL", "OBSIDIAN_VAULT", "CLAUDE_PROJECTS_DIR", "CLAUDE_CONFIG_JSON",
	"FOUNDEROS_OS_SEAT_ID", "FOUNDEROS_OS_SEAT_LABEL", "CODEX_SESSIONS_DIR", "PAPERCLIP_HOME", "OLLAMA_BASE_URL",
	"OLLAMA_LOG_DIR", "PAPERCLIP_API_URL", "HERMES_GATEWAY_URL", "GBRAIN_BIN",
	"CLAUDE_OAUTH_TOKEN", "CLAUDE_OAUTH_TOKEN_2", "CLAUDE_OAUTH_TOKEN_3", "CLAUDE_OAUTH_TOKEN_4", "CLAUDE_OAUTH_TOKEN_5",
	"CLAUDE_OAUTH_TOKEN_6", "CLAUDE_OAUTH_TOKEN_7", "CLAUDE_OAUTH_TOKEN_8", "CLAUDE_OAUTH_TOKEN_9", "CLAUDE_OAUTH_TOKEN_10",
	"CLAUDE_OAUTH_TOKEN_11", "CLAUDE_OAUTH_TOKEN_12",
}

func envResolver(t *testing.T, lines ...string) connectors.Resolver {
	t.Helper()
	for _, k := range collectEnv {
		t.Setenv(k, "")
	}
	p := filepath.Join(t.TempDir(), "env.local")
	if err := os.WriteFile(p, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return connectors.Resolver{EnvLocal: p}
}

func TestDeviceSlug(t *testing.T) {
	if got := DeviceSlug("Alexs-MacBook-Pro.local"); got != "alexs-macbook-pro" {
		t.Errorf("got %q", got)
	}
	if got := DeviceSlug("alex's Mac mini"); got != "alex-s-mac-mini" {
		t.Errorf("got %q", got)
	}
}

func TestDefaultConfigUsesHomeAndOverrides(t *testing.T) {
	home := "/Users/x"
	c := DefaultConfig(envResolver(t), home, "Alexs-MacBook-Pro.local")
	if c.Device != "alexs-macbook-pro" || c.Label != "Alexs-MacBook-Pro" {
		t.Errorf("device = %q label = %q", c.Device, c.Label)
	}
	if c.WisprDB != "/Users/x/Library/Application Support/Wispr Flow/flow.sqlite" || c.ObsidianVault != "/Users/x/Documents/Obsidian Vault" {
		t.Errorf("paths = %+v", c)
	}
	if c.Claude.ID != "claude-alexs-macbook-pro" || c.Claude.Label != "Claude · Alexs-MacBook-Pro" || c.CodexBoardRoot != "/Users/x/.paperclip" {
		t.Errorf("seat = %+v board=%q", c.Claude, c.CodexBoardRoot)
	}
	if c.Stack.PaperclipURL != "" || c.OllamaBaseURL != "http://localhost:11434" {
		t.Errorf("stack = %+v ollama=%q", c.Stack, c.OllamaBaseURL)
	}

	c = DefaultConfig(envResolver(t, "OBSIDIAN_VAULT=/v", "CODEX_SESSIONS_DIR=/c", "FOUNDEROS_OS_SEAT_ID=claude-main", "PAPERCLIP_API_URL=http://mini:3100", "FOUNDEROS_COLLECTOR_DEVICE=mini"), home, "h")
	if c.ObsidianVault != "/v" || c.CodexDir != "/c" || c.CodexBoardRoot != "" || c.Claude.ID != "claude-main" || c.Stack.PaperclipURL != "http://mini:3100" || c.Device != "mini" {
		t.Errorf("overrides = %+v", c)
	}
}

// A whole Mac in a temp dir: every source present, one board ping answering.
func fakeMac(t *testing.T) Config {
	t.Helper()
	home := t.TempDir()
	chatDB(t, filepath.Join(home, "Library", "Group Containers"))
	flowDB(t, filepath.Join(home, "Library", "Application Support", "Wispr Flow"))
	write(t, filepath.Join(home, "Documents", "Obsidian Vault", "note.md"), "# hi")
	write(t, filepath.Join(home, ".claude", "projects", "p", "s.jsonl"), aline(isoMillis(now.Add(-time.Hour)), 100))
	write(t, filepath.Join(home, ".claude.json"), `{"oauthAccount":{"organizationType":"claude_max","organizationRateLimitTier":"x_20x"}}`)
	write(t, filepath.Join(home, ".codex", "sessions", "2026", "09", "29", "r.jsonl"), codexLine(isoMillis(now.Add(-time.Hour)), 50, true))
	board := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))
	t.Cleanup(board.Close)
	cfg := DefaultConfig(envResolver(t, "PAPERCLIP_API_URL="+board.URL, "HERMES_GATEWAY_URL=http://127.0.0.1:1", "OLLAMA_BASE_URL=http://127.0.0.1:1"), home, "Test-Mac.local")
	cfg.Stack.BrewDir, cfg.Stack.UsrLocalBin = t.TempDir(), t.TempDir()
	return cfg
}

func TestCollectBuildsAValidPayload(t *testing.T) {
	cfg := fakeMac(t)
	p := NewCollector(cfg).Collect(context.Background(), now)
	if err := devicepush.Validate(p); err != nil {
		t.Fatalf("payload invalid: %v", err)
	}
	if p.Device != "test-mac" || p.Label != "Test-Mac" || p.CapturedAt != isoMillis(now) {
		t.Errorf("identity = %q %q %q", p.Device, p.Label, p.CapturedAt)
	}
	states := map[string]connectors.State{}
	for _, s := range p.Statuses {
		states[s.ID] = s.State
	}
	for _, id := range []string{"whatsapp", "wispr", "obsidian", "local-stack"} {
		if states[id] != connectors.StateConnected {
			t.Errorf("%s = %s", id, states[id])
		}
	}
	if p.WhatsApp == nil || len(p.WhatsApp.Chats) != 2 || p.Wispr == nil || len(p.Wispr.Notes) == 0 || p.Obsidian == nil || len(p.Obsidian.Notes) != 1 {
		t.Fatalf("data = %+v %+v %+v", p.WhatsApp, p.Wispr, p.Obsidian)
	}
	if len(p.Usage) != 2 || p.Usage[0].Kind != "claude" || p.Usage[0].ID != "claude-test-mac" || *p.Usage[0].Plan != "Claude Max 20x" ||
		p.Usage[1].Kind != "codex" || p.Usage[1].ID != "codex-test-mac" || p.Usage[1].Label != "Codex · Test-Mac" {
		t.Fatalf("usage = %+v", p.Usage)
	}
	if p.Ollama != nil {
		t.Errorf("a down Ollama with no log is not pushed: %+v", p.Ollama)
	}
}

func TestCollectLeavesOutDataItCouldNotRead(t *testing.T) {
	cfg := DefaultConfig(envResolver(t, "PAPERCLIP_API_URL=http://127.0.0.1:1", "HERMES_GATEWAY_URL=http://127.0.0.1:1", "OLLAMA_BASE_URL=http://127.0.0.1:1"), t.TempDir(), "empty")
	cfg.Stack.BrewDir, cfg.Stack.UsrLocalBin = t.TempDir(), t.TempDir()
	p := NewCollector(cfg).Collect(context.Background(), now)
	if err := devicepush.Validate(p); err != nil {
		t.Fatal(err)
	}
	if p.WhatsApp != nil || p.Wispr != nil || p.Obsidian != nil {
		t.Fatalf("absent sources must not push empty data that reads as 'nothing': %+v", p)
	}
	for _, s := range p.Statuses {
		if s.State == connectors.StateConnected {
			t.Errorf("%s connected on an empty Mac", s.ID)
		}
	}
}

func TestPushPostsJSONAndReportsRefusals(t *testing.T) {
	receiver := devicepush.NewReceiver(devicepush.NewMemStore())
	var auth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		if r.Method != http.MethodPost || r.Header.Get("Content-Type") != "application/json" {
			w.WriteHeader(405)
			return
		}
		var p devicepush.Payload
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			w.WriteHeader(400)
			return
		}
		if err := receiver.Accept(r.Context(), p); err != nil {
			w.WriteHeader(400)
			_, _ = w.Write([]byte(`{"ok":false,"error":"` + err.Error() + `"}`))
			return
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	p := NewCollector(fakeMac(t)).Collect(context.Background(), now)
	if err := Push(context.Background(), nil, srv.URL+"/ingest", "tok", p); err != nil {
		t.Fatal(err)
	}
	if auth != "Bearer tok" {
		t.Errorf("auth = %q", auth)
	}
	s := receiver.Connector("whatsapp").Status(context.Background())
	if s.State != connectors.StateConnected || !strings.Contains(s.Detail, "pushed by Test-Mac") {
		t.Fatalf("status after push = %+v", s)
	}

	err := Push(context.Background(), nil, srv.URL, "", devicepush.Payload{Device: "BAD!"})
	if err == nil || !strings.Contains(err.Error(), "400") || !strings.Contains(err.Error(), "slug") {
		t.Fatalf("err = %v", err)
	}
}
