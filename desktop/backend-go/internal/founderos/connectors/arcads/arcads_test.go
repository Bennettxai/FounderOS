package arcads

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

// Shaped like GET https://external-api.arcads.ai/v1/products.
const productsFixture = `[{"id":"prod_1","name":"Vantage","description":"AI intake agents","createdAt":"2026-06-01T10:00:00.000Z"},{"id":"prod_2","name":"Launchpad Cohort","createdAt":"2026-06-03T10:00:00.000Z"}]`

func newConnector(t *testing.T, envLocal, arcadsFile string) *Connector {
	t.Helper()
	t.Setenv("ARCADS_BASIC_AUTH", "")
	dir := t.TempDir()
	p := filepath.Join(dir, "env.local")
	os.WriteFile(p, []byte(envLocal), 0o600)
	c := New(connectors.Resolver{EnvLocal: p})
	c.Files = []string{filepath.Join(dir, "absent.env")}
	if arcadsFile != "" {
		f := filepath.Join(dir, ".env")
		os.WriteFile(f, []byte(arcadsFile), 0o600)
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
	if Meta.ID != "arcads" || Meta.Name != "Arcads (UGC Ads)" || Meta.Kind != connectors.KindCreative {
		t.Fatalf("Meta = %+v", Meta)
	}
}

func TestStatusNotConfigured(t *testing.T) {
	s := newConnector(t, "", "").Status(context.Background())
	if s.State != connectors.StateNotConfigured || s.Detail != "ARCADS_BASIC_AUTH not found in env or ~/.founderos/.env." {
		t.Fatalf("status = %+v", s)
	}
}

func TestStatusConnectedFromArcadsEnvFile(t *testing.T) {
	c := newConnector(t, "", "ARCADS_BASIC_AUTH=dXNlcjpwYXNz\n")
	c.BaseURL = serve(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/products" {
			t.Errorf("unexpected %s %s", r.Method, r.URL)
		}
		if r.Header.Get("Authorization") != "Basic dXNlcjpwYXNz" {
			t.Errorf("auth = %q", r.Header.Get("Authorization"))
		}
		io.WriteString(w, productsFixture)
	})
	s := c.Status(context.Background())
	if s.State != connectors.StateConnected || s.Detail != "Vantage workspace reachable · 2 products" || s.Meta["products"] != 2 {
		t.Fatalf("status = %+v", s)
	}
}

func TestStatusKeepsAnExistingBasicPrefixAndReadsResults(t *testing.T) {
	c := newConnector(t, "ARCADS_BASIC_AUTH=Basic abc\n", "")
	c.BaseURL = serve(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Basic abc" {
			t.Errorf("auth = %q", r.Header.Get("Authorization"))
		}
		io.WriteString(w, `{"results":[{"id":"prod_1","name":"Vantage"}]}`)
	})
	if s := c.Status(context.Background()); s.Detail != "Vantage workspace reachable · 1 product" || s.Meta["products"] != 1 {
		t.Fatalf("status = %+v", s)
	}
}

func TestStatusUnknownCountIsNotZero(t *testing.T) {
	c := newConnector(t, "ARCADS_BASIC_AUTH=x\n", "")
	c.BaseURL = serve(t, func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, `{"ok":true}`) })
	s := c.Status(context.Background())
	if s.State != connectors.StateConnected || s.Detail != "Vantage workspace reachable · product count unknown" {
		t.Fatalf("status = %+v", s)
	}
}

func TestStatusErrorOn403SaysRotate(t *testing.T) {
	c := newConnector(t, "ARCADS_BASIC_AUTH=x\n", "")
	c.BaseURL = serve(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusForbidden) })
	s := c.Status(context.Background())
	if s.State != connectors.StateError || s.Detail != "Creds found but API check failed: HTTP 403 — rotate creds in Arcads dashboard" {
		t.Fatalf("status = %+v", s)
	}
}

func TestStatusErrorWhenUnreachable(t *testing.T) {
	c := newConnector(t, "ARCADS_BASIC_AUTH=x\n", "")
	c.BaseURL = "http://127.0.0.1:1"
	s := c.Status(context.Background())
	if s.State != connectors.StateError || !strings.HasPrefix(s.Detail, "Creds found but API check failed: ") {
		t.Fatalf("status = %+v", s)
	}
}
