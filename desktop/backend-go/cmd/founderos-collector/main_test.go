package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/devicepush"
)

func isolate(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("FOUNDEROS_ENV_LOCAL", t.TempDir()+"/env.local")
	for _, k := range []string{"FOUNDEROS_COLLECTOR_URL", "FOUNDEROS_COLLECTOR_TOKEN", "OBSIDIAN_VAULT", "CLAUDE_PROJECTS_DIR", "CODEX_SESSIONS_DIR"} {
		t.Setenv(k, "")
	}
	t.Setenv("PAPERCLIP_API_URL", "http://127.0.0.1:1")
	t.Setenv("HERMES_GATEWAY_URL", "http://127.0.0.1:1")
	t.Setenv("OLLAMA_BASE_URL", "http://127.0.0.1:1")
	t.Setenv("FOUNDEROS_COLLECTOR_DEVICE", "test-mac")
}

func TestNoURLWithoutDryRunIsAUsageError(t *testing.T) {
	isolate(t)
	var out, errOut bytes.Buffer
	if code := run([]string{}, &out, &errOut); code != 2 || !strings.Contains(errOut.String(), "FOUNDEROS_COLLECTOR_URL") {
		t.Fatalf("code = %d, stderr = %s", code, errOut.String())
	}
}

func TestDryRunPrintsAValidPayloadAndPostsNothing(t *testing.T) {
	isolate(t)
	var out, errOut bytes.Buffer
	if code := run([]string{"-dry-run"}, &out, &errOut); code != 0 {
		t.Fatalf("code = %d, stderr = %s", code, errOut.String())
	}
	var p devicepush.Payload
	if err := json.Unmarshal(out.Bytes(), &p); err != nil {
		t.Fatalf("stdout is not a payload: %v", err)
	}
	if err := devicepush.Validate(p); err != nil || p.Device != "test-mac" || len(p.Statuses) != 4 {
		t.Fatalf("payload = %+v, %v", p, err)
	}
}

// The official Claude gauge's OAuth token is read on this Mac and used for
// one GET; nothing the collector prints or pushes may carry it.
func TestTheClaudeOAuthTokenNeverReachesTheOutput(t *testing.T) {
	isolate(t)
	const token = "sk-ant-oat01-NEVER-LEAVE-THIS-MAC-cmd-test"
	t.Setenv("CLAUDE_OAUTH_TOKEN", token)
	var out, errOut bytes.Buffer
	if code := run([]string{"-dry-run"}, &out, &errOut); code != 0 {
		t.Fatalf("code = %d, stderr = %s", code, errOut.String())
	}
	if strings.Contains(out.String()+errOut.String(), token) {
		t.Fatal("the Claude OAuth token reached the collector's output")
	}
}

func TestOncePushesToTheURL(t *testing.T) {
	isolate(t)
	var got devicepush.Payload
	var auth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&got)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()
	t.Setenv("FOUNDEROS_COLLECTOR_TOKEN", "secret")
	var out, errOut bytes.Buffer
	if code := run([]string{"-url", srv.URL + "/ingest"}, &out, &errOut); code != 0 {
		t.Fatalf("code = %d, stderr = %s", code, errOut.String())
	}
	if got.Device != "test-mac" || auth != "Bearer secret" || !strings.Contains(out.String(), "pushed test-mac") {
		t.Fatalf("got = %+v auth=%q out=%q", got, auth, out.String())
	}

	srv.Close()
	if code := run([]string{"-url", srv.URL}, &out, &errOut); code != 1 {
		t.Fatalf("an unreachable backend must fail the run, code = %d", code)
	}
}

// stuckWispr puts a named pipe where Wispr's flow.sqlite lives under the
// isolated HOME: opening it blocks, as the mini's collector did for 19h.
func stuckWispr(t *testing.T) {
	t.Helper()
	dir := filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "Wispr Flow")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "flow.sqlite")
	if err := syscall.Mkfifo(p, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if f, err := os.OpenFile(p, os.O_WRONLY|syscall.O_NONBLOCK, 0); err == nil {
			f.Close()
		}
	})
}

func runWithin(t *testing.T, limit time.Duration, args ...string) (int, string, string) {
	t.Helper()
	var out, errOut bytes.Buffer
	done := make(chan int, 1)
	go func() { done <- run(args, &out, &errOut) }()
	select {
	case code := <-done:
		return code, out.String(), errOut.String()
	case <-time.After(limit):
		t.Fatalf("run(%v) did not return within %s", args, limit)
		return 0, "", ""
	}
}

func TestAStuckSourceTimesOutAndTheTickStillPushes(t *testing.T) {
	isolate(t)
	stuckWispr(t)
	code, out, errOut := runWithin(t, 10*time.Second, "-dry-run", "-source-timeout", "300ms")
	if code != 0 || !strings.Contains(errOut, "wispr timed out") {
		t.Fatalf("code = %d, stderr = %s", code, errOut)
	}
	var p devicepush.Payload
	if err := json.Unmarshal([]byte(out), &p); err != nil || len(p.Statuses) != 4 {
		t.Fatalf("payload = %+v, %v", p, err)
	}
}

// launchd's StartInterval never starts a run while the last is alive, so a
// tick past its deadline must end the process, naming what it waited on.
func TestATickPastItsDeadlineExits(t *testing.T) {
	isolate(t)
	stuckWispr(t)
	for _, args := range [][]string{
		{"-dry-run", "-source-timeout", "1m", "-deadline", "300ms"},
		{"-dry-run", "-source-timeout", "1m", "-deadline", "300ms", "-every", "10m"},
	} {
		code, _, errOut := runWithin(t, 10*time.Second, args...)
		if code != 1 || !strings.Contains(errOut, "deadline") || !strings.Contains(errOut, "wispr") {
			t.Fatalf("%v: code = %d, stderr = %s", args, code, errOut)
		}
	}
}

func TestTheDefaultDeadlineFitsInsideLaunchdsInterval(t *testing.T) {
	if defaultDeadline >= 600*time.Second || defaultSourceTimeout >= defaultDeadline {
		t.Fatalf("deadline %s must be under the 600s StartInterval and above the per-source %s", defaultDeadline, defaultSourceTimeout)
	}
}
