package connectors

import (
	"os"
	"path/filepath"
	"testing"
)

// FounderOS reads credentials from the process env, ~/.founderos/.env and the
// keys planted through /api/admin/keys. Nothing else in the home directory.
func TestDefaultResolverReadsOnlyTheFounderOSHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("FOUNDEROS_PLANTED_ENV", "")
	t.Setenv("FOUNDEROS_ENV_LOCAL", "")

	r := DefaultResolver()
	if want := filepath.Join(home, ".founderos", "planted.env"); r.Planted != want {
		t.Errorf("Planted = %q, want %q", r.Planted, want)
	}
	if want := filepath.Join(home, ".founderos", ".env"); r.EnvLocal != want {
		t.Errorf("EnvLocal = %q, want %q", r.EnvLocal, want)
	}
}

func TestDefaultResolverFindsAKeyInTheFounderOSEnvFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("FOUNDEROS_PLANTED_ENV", "")
	t.Setenv("FOUNDEROS_ENV_LOCAL", "")
	t.Setenv("SLACK_BOT_TOKEN", "")
	writeFile(t, filepath.Join(home, ".founderos", ".env"), "SLACK_BOT_TOKEN=xoxb-from-founderos-home\n")

	if got := DefaultResolver().Resolve("SLACK_BOT_TOKEN"); got != "xoxb-from-founderos-home" {
		t.Fatalf("Resolve = %q", got)
	}
}

// Other tools' credential files on the same machine are never read, even when
// they hold a key a connector wants.
func TestOtherToolsCredentialFilesAreNeverRead(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	keys := []string{"ZERNIO_API_KEY", "MIRO_ACCESS_TOKEN", "ARCADS_API_KEY", "TRAKYO_API_KEY", "MANYCHAT_API_KEY"}
	body := ""
	for _, k := range keys {
		t.Setenv(k, "")
		body += k + "=leaked\n"
	}
	for _, p := range []string{".social-media/.env", "clue-agent/.env.agents", "Projects/arcads-agent-skills/.env"} {
		writeFile(t, filepath.Join(home, p), body)
	}
	writeFile(t, filepath.Join(home, ".claude.json"), `{"mcpServers":{"manychat":{"env":{"MANYCHAT_API_KEY":"leaked"}}}}`)

	social, clue, arcads, claudeJSON := CredFiles()
	r := Resolver{}
	for _, k := range keys {
		if v := r.Resolve(k, social, clue, arcads); v != "" {
			t.Errorf("Resolve(%s) read another tool's credential file: %q", k, v)
		}
	}
	if v := McpEnvKey(claudeJSON, "manychat", "MANYCHAT_API_KEY"); v != "" {
		t.Errorf("McpEnvKey read ~/.claude.json: %q", v)
	}
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}
