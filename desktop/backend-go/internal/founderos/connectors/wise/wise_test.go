package wise

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/guard"
)

// resolver returns a Resolver whose env.local holds exactly body, with the
// process env for the Wise key blanked so a developer's shell cannot leak in.
func resolver(t *testing.T, body string) connectors.Resolver {
	t.Helper()
	t.Setenv("WISE_1_TOKEN", "")
	p := filepath.Join(t.TempDir(), "env.local")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return connectors.Resolver{EnvLocal: p}
}

// A /v1/transfers payload shaped like Wise's real response (trimmed).
const transfersFixture = `[
  {"id": 101, "user": 7, "targetAccount": 9, "status": "outgoing_payment_sent",
   "reference": "rent", "rate": 1, "created": "2026-09-10 12:00:00",
   "sourceCurrency": "USD", "sourceValue": 250.5,
   "targetCurrency": "USD", "targetValue": 250.5},
  {"id": 102, "status": "processing", "created": 1757500000,
   "targetCurrency": "EUR", "targetValue": 10.25},
  {"id": 103, "status": "cancelled", "targetCurrency": "EUR"},
  {"id": 104, "targetValue": "12", "targetCurrency": "EUR", "created": "x"}
]`

func server(t *testing.T, h http.HandlerFunc) *httptest.Server {
	t.Helper()
	s := httptest.NewServer(h)
	t.Cleanup(s.Close)
	return s
}

func TestMetaMatchesFounderosOS(t *testing.T) {
	if Meta.ID != "wise-1" || Meta.Name != "Wise" || Meta.Kind != connectors.KindPayments {
		t.Fatalf("Meta = %+v", Meta)
	}
}

func TestParseTransfersMapsAndSkipsMalformed(t *testing.T) {
	got := ParseTransfers([]byte(transfersFixture))
	if len(got) != 2 {
		t.Fatalf("want 2 transfers, got %d: %+v", len(got), got)
	}
	ref := "rent"
	if got[0].AmountCents != 25050 || got[0].Currency != "USD" || got[0].Status != "outgoing_payment_sent" ||
		got[0].Reference == nil || *got[0].Reference != ref {
		t.Errorf("first = %+v", got[0])
	}
	if got[1].AmountCents != 1025 || got[1].Reference != nil {
		t.Errorf("second = %+v", got[1])
	}
	// created keeps its JSON type: a string stays a string, a number a number.
	out, _ := json.Marshal(got)
	if !strings.Contains(string(out), `"created":"2026-09-10 12:00:00"`) || !strings.Contains(string(out), `"created":1757500000`) {
		t.Errorf("created lost its type: %s", out)
	}
	if strings.Contains(string(out), `"reference":null`) {
		t.Errorf("absent reference must be omitted: %s", out)
	}
}

func TestParseTransfersAcceptsWrappedAndRejectsJunk(t *testing.T) {
	wrapped := ParseTransfers([]byte(`{"transfers":[{"targetValue":10,"targetCurrency":"EUR","status":"s","created":"x"},{"nope":1}]}`))
	if len(wrapped) != 1 || wrapped[0].AmountCents != 1000 {
		t.Fatalf("wrapped = %+v", wrapped)
	}
	noStatus := ParseTransfers([]byte(`[{"targetValue":1,"targetCurrency":"EUR","created":"x"}]`))
	if len(noStatus) != 1 || noStatus[0].Status != "unknown" {
		t.Fatalf("missing status must read unknown: %+v", noStatus)
	}
	for _, junk := range []string{`null`, `{}`, `"x"`, `not json`} {
		if got := ParseTransfers([]byte(junk)); len(got) != 0 {
			t.Errorf("%s → %+v, want empty", junk, got)
		}
	}
}

func TestOutgoingUnkeyedReturnsNil(t *testing.T) {
	c := New(resolver(t, ""))
	got, err := c.Outgoing(context.Background())
	if err != nil || got != nil {
		t.Fatalf("unkeyed must be (nil, nil) so the page hides the section, got %v, %v", got, err)
	}
}

func TestOutgoingReadsTransfers(t *testing.T) {
	s := server(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/transfers" || r.URL.Query().Get("limit") != "10" {
			t.Errorf("unexpected %s %s", r.Method, r.URL)
		}
		if r.Header.Get("Authorization") != "Bearer tok" {
			t.Errorf("auth header = %q", r.Header.Get("Authorization"))
		}
		w.Write([]byte(transfersFixture))
	})
	c := New(resolver(t, "WISE_1_TOKEN=tok\n"))
	c.BaseURL = s.URL
	got, err := c.Outgoing(context.Background())
	if err != nil || len(got) != 2 {
		t.Fatalf("got %v, %v", got, err)
	}
}

func TestOutgoingKeyedButEmptyIsEmptyNotNil(t *testing.T) {
	s := server(t, func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`[]`)) })
	c := New(resolver(t, "WISE_1_TOKEN=tok\n"))
	c.BaseURL = s.URL
	got, err := c.Outgoing(context.Background())
	if err != nil || got == nil || len(got) != 0 {
		t.Fatalf("keyed + nothing sent must be an empty list, got %#v, %v", got, err)
	}
}

// FounderOS v1 skipped a non-OK response and returned [], which renders a dead
// token as "nothing sent". The bridge reports it as an error instead.
func TestOutgoingAuthFailureIsAnErrorNotEmpty(t *testing.T) {
	s := server(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"invalid_token","error_description":"Invalid token"}`))
	})
	c := New(resolver(t, "WISE_1_TOKEN=bad\n"))
	c.BaseURL = s.URL
	got, err := c.Outgoing(context.Background())
	if err == nil || got != nil {
		t.Fatalf("want error, got %v, %v", got, err)
	}
}

func TestStatusNotConfigured(t *testing.T) {
	st := New(resolver(t, "")).Status(context.Background())
	if st.State != connectors.StateNotConfigured || !strings.Contains(st.Detail, "WISE_1_TOKEN") {
		t.Fatalf("%+v", st)
	}
}

func TestStatusConnected(t *testing.T) {
	s := server(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/profiles" {
			t.Errorf("status must read profiles, got %s", r.URL.Path)
		}
		w.Write([]byte(`[{"id":1,"type":"personal"},{"id":2,"type":"business"}]`))
	})
	c := New(resolver(t, "WISE_1_TOKEN=tok\n"))
	c.BaseURL = s.URL
	st := c.Status(context.Background())
	if st.State != connectors.StateConnected || st.Detail != "Wise · 2 profiles" {
		t.Fatalf("%+v", st)
	}
	if st.ID != Meta.ID || st.Name != Meta.Name || st.Kind != Meta.Kind {
		t.Fatalf("identity not stamped: %+v", st)
	}
}

func TestStatusErrorOnAuthFailureAndUnreachable(t *testing.T) {
	s := server(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"invalid_token"}`))
	})
	c := New(resolver(t, "WISE_1_TOKEN=bad\n"))
	c.BaseURL = s.URL
	if st := c.Status(context.Background()); st.State != connectors.StateError || !strings.Contains(st.Detail, "401") {
		t.Fatalf("auth failure: %+v", st)
	}
	s.Close()
	if st := c.Status(context.Background()); st.State != connectors.StateError {
		t.Fatalf("unreachable: %+v", st)
	}
}

// The connector has no write path (FounderOS v1 never moves money through Wise),
// and its client is the guarded one: a mutating call is refused before it
// leaves the process while FOUNDEROS_WRITES is off.
func TestClientRefusesWritesWhenDisabled(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "0")
	c := New(resolver(t, "WISE_1_TOKEN=tok\n"))
	req, _ := http.NewRequest(http.MethodPost, "https://api.wise.com/v3/profiles/1/transfers", strings.NewReader("{}"))
	if _, err := c.client.Do(req); !errors.Is(err, guard.ErrWritesDisabled) {
		t.Fatalf("want ErrWritesDisabled, got %v", err)
	}
}
