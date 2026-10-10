package miro

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
)

// Shaped like GET https://api.miro.com/v2/boards?limit=10.
const boardsFixture = `{"data":[{"id":"uXjVKabc=","type":"board","name":"Vantage funnel map"},{"id":"uXjVKdef=","type":"board","name":"Offer stack"}],"total":14,"size":2,"offset":0,"limit":10,"links":{"self":"https://api.miro.com/v2/boards?limit=10&offset=0"},"type":"list"}`

func newConnector(t *testing.T, envLocal, clueFile string) *Connector {
	t.Helper()
	t.Setenv("MIRO_ACCESS_TOKEN", "")
	dir := t.TempDir()
	p := filepath.Join(dir, "env.local")
	os.WriteFile(p, []byte(envLocal), 0o600)
	c := New(connectors.Resolver{EnvLocal: p})
	c.Files = []string{filepath.Join(dir, "absent.env")}
	if clueFile != "" {
		f := filepath.Join(dir, ".env.agents")
		os.WriteFile(f, []byte(clueFile), 0o600)
		c.Files = []string{f}
	}
	return c
}

func serve(t *testing.T, h http.HandlerFunc) string {
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return srv.URL
}

func TestMetaMatchesFounderosOS(t *testing.T) {
	if Meta.ID != "miro" || Meta.Name != "Miro" || Meta.Kind != connectors.KindCreative {
		t.Fatalf("Meta = %+v", Meta)
	}
}

func TestStatusNotConfigured(t *testing.T) {
	s := newConnector(t, "", "").Status(context.Background())
	if s.State != connectors.StateNotConfigured || s.Detail != "MIRO_ACCESS_TOKEN not found in env or ~/.founderos/.env." {
		t.Fatalf("status = %+v", s)
	}
}

func TestStatusConnectedCountsBoards(t *testing.T) {
	c := newConnector(t, "", "MIRO_ACCESS_TOKEN=miro-tok\n") // from the clue-agent fallback file
	c.BaseURL = serve(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/boards" || r.URL.Query().Get("limit") != "10" {
			t.Errorf("unexpected %s", r.URL)
		}
		if r.Header.Get("Authorization") != "Bearer miro-tok" {
			t.Errorf("auth = %q", r.Header.Get("Authorization"))
		}
		io.WriteString(w, boardsFixture)
	})
	s := c.Status(context.Background())
	if s.State != connectors.StateConnected || s.Detail != "Boards API reachable · 14 boards" || s.Meta["boards"] != 14 {
		t.Fatalf("status = %+v", s)
	}
}

func TestStatusFallsBackToPageSizeWithoutTotal(t *testing.T) {
	c := newConnector(t, "MIRO_ACCESS_TOKEN=t\n", "")
	c.BaseURL = serve(t, func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"data":[{"id":"a"}],"type":"list"}`)
	})
	if s := c.Status(context.Background()); s.Detail != "Boards API reachable · 1 boards" || s.Meta["boards"] != 1 {
		t.Fatalf("status = %+v", s)
	}
}

// Unknown reads unknown: a reachable API that reports no count is not "0 boards".
func TestStatusUnknownCountIsNotZero(t *testing.T) {
	c := newConnector(t, "MIRO_ACCESS_TOKEN=t\n", "")
	c.BaseURL = serve(t, func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, `{"type":"list"}`) })
	s := c.Status(context.Background())
	if s.State != connectors.StateConnected || s.Detail != "Boards API reachable · board count unknown" {
		t.Fatalf("status = %+v", s)
	}
	if _, has := s.Meta["boards"]; has {
		t.Errorf("meta must not claim a count: %v", s.Meta)
	}
}

func TestStatusErrorOnRejectedToken(t *testing.T) {
	c := newConnector(t, "MIRO_ACCESS_TOKEN=t\n", "")
	c.BaseURL = serve(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusUnauthorized) })
	s := c.Status(context.Background())
	if s.State != connectors.StateError || s.Detail != "Token found but API check failed: HTTP 401" {
		t.Fatalf("status = %+v", s)
	}
}

func TestStatusErrorWhenUnreachable(t *testing.T) {
	c := newConnector(t, "MIRO_ACCESS_TOKEN=t\n", "")
	c.BaseURL = "http://127.0.0.1:1"
	s := c.Status(context.Background())
	if s.State != connectors.StateError || !strings.HasPrefix(s.Detail, "Token found but API check failed: ") {
		t.Fatalf("status = %+v", s)
	}
}
