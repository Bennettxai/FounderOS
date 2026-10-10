package guard

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWritesAndCronsAreOffUnlessExplicitlyOne(t *testing.T) {
	for _, v := range []string{"", "0", "true", "yes", " 1"} {
		t.Setenv("FOUNDEROS_WRITES", v)
		t.Setenv("FOUNDEROS_CRONS", v)
		if WritesEnabled() || CronsEnabled() {
			t.Errorf("value %q must leave writes and crons off", v)
		}
	}
	t.Setenv("FOUNDEROS_WRITES", "1")
	t.Setenv("FOUNDEROS_CRONS", "1")
	if !WritesEnabled() || !CronsEnabled() {
		t.Error("value 1 must turn writes and crons on")
	}
}

func TestOutboundRefusesAndRecordsWhenWritesOff(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "0")
	Reset()
	ran := false
	err := Outbound("slack.chat.postMessage", func() error { ran = true; return nil })
	if !errors.Is(err, ErrWritesDisabled) {
		t.Fatalf("err = %v, want ErrWritesDisabled", err)
	}
	if ran {
		t.Fatal("side effect ran while writes were off")
	}
	refused := Refused()
	if len(refused) != 1 || refused[0].Action != "slack.chat.postMessage" {
		t.Fatalf("refusal log = %+v", refused)
	}

	t.Setenv("FOUNDEROS_WRITES", "1")
	if err := Outbound("slack.chat.postMessage", func() error { ran = true; return nil }); err != nil || !ran {
		t.Fatalf("with writes on: err=%v ran=%v", err, ran)
	}
}

func TestTransportBlocksMutatingRequestsToExternalHosts(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "0")
	Reset()
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits++ }))
	defer srv.Close()
	// httptest listens on 127.0.0.1, which the guard treats as local; route
	// through a fake external name to exercise the external path.
	client := &http.Client{Transport: Transport(&rewrite{to: srv.URL})}

	resp, err := client.Get("https://api.slack.com/api/conversations.list")
	if err != nil {
		t.Fatalf("GET must pass: %v", err)
	}
	resp.Body.Close()

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		req, _ := http.NewRequest(method, "https://api.slack.com/api/chat.postMessage", strings.NewReader("{}"))
		_, err := client.Do(req)
		if !errors.Is(err, ErrWritesDisabled) {
			t.Errorf("%s: err = %v, want ErrWritesDisabled", method, err)
		}
	}
	if hits != 1 {
		t.Fatalf("server saw %d requests, want only the GET", hits)
	}
	if n := len(Refused()); n != 4 {
		t.Fatalf("refusals recorded = %d, want 4", n)
	}
}

func TestTransportAllowsReadOnlyPostsToAllowlistedEndpoints(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "0")
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits++ }))
	defer srv.Close()
	// Some read APIs are POST-shaped (GraphQL queries, search). They must be
	// listed explicitly; everything else stays blocked.
	client := &http.Client{Transport: Transport(&rewrite{to: srv.URL}, "api.notion.com/v1/databases/")}
	req, _ := http.NewRequest(http.MethodPost, "https://api.notion.com/v1/databases/abc/query", strings.NewReader("{}"))
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("allowlisted read POST refused: %v", err)
	}
	resp.Body.Close()
	req, _ = http.NewRequest(http.MethodPost, "https://api.notion.com/v1/pages", strings.NewReader("{}"))
	if _, err := client.Do(req); !errors.Is(err, ErrWritesDisabled) {
		t.Fatalf("non-allowlisted POST: err = %v", err)
	}
}

func TestTransportLeavesLocalhostAlone(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "0")
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits++ }))
	defer srv.Close()
	client := &http.Client{Transport: Transport(nil)}
	resp, err := client.Post(srv.URL+"/api/ingest", "application/json", strings.NewReader("{}"))
	if err != nil {
		t.Fatalf("POST to the local staging engine must pass: %v", err)
	}
	resp.Body.Close()
	if hits != 1 {
		t.Fatal("local POST did not arrive")
	}
}

func TestCronGate(t *testing.T) {
	t.Setenv("FOUNDEROS_CRONS", "0")
	Reset()
	ran := false
	if RunCron("comms-digest", func() error { ran = true; return nil }) || ran {
		t.Fatal("cron ran while crons were off")
	}
	if r := Refused(); len(r) != 1 || r[0].Action != "cron:comms-digest" {
		t.Fatalf("refusal log = %+v", r)
	}
	t.Setenv("FOUNDEROS_CRONS", "1")
	if !RunCron("comms-digest", func() error { ran = true; return nil }) || !ran {
		t.Fatal("cron did not run with crons on")
	}
}

// rewrite sends every request to a test server while keeping the original
// URL visible to the guard.
type rewrite struct{ to string }

func (r *rewrite) RoundTrip(req *http.Request) (*http.Response, error) {
	out := req.Clone(req.Context())
	u, _ := out.URL.Parse(r.to + req.URL.Path)
	out.URL = u
	out.Host = u.Host
	return http.DefaultTransport.RoundTrip(out)
}

func TestTransportTreatsWebDAVReadMethodsAsReads(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "0")
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits++ }))
	defer srv.Close()
	client := &http.Client{Transport: Transport(&rewrite{to: srv.URL})}
	for _, m := range []string{"PROPFIND", "REPORT"} {
		req, _ := http.NewRequest(m, "https://apidata.googleusercontent.com/caldav/v2/x/events", strings.NewReader("<q/>"))
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("%s refused: %v", m, err)
		}
		resp.Body.Close()
	}
	req, _ := http.NewRequest("MKCALENDAR", "https://apidata.googleusercontent.com/caldav/v2/x", nil)
	if _, err := client.Do(req); !errors.Is(err, ErrWritesDisabled) {
		t.Fatalf("MKCALENDAR must stay blocked, got %v", err)
	}
	if hits != 2 {
		t.Fatalf("hits = %d", hits)
	}
}

func TestAnEmptyAllowlistEntryAllowsNothing(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "0")
	client := &http.Client{Transport: Transport(&rewrite{to: "http://127.0.0.1:1"}, "")}
	req, _ := http.NewRequest(http.MethodPost, "https://api.example.com/send", strings.NewReader("{}"))
	if _, err := client.Do(req); !errors.Is(err, ErrWritesDisabled) {
		t.Fatalf("an empty read-POST prefix must not open every POST, got %v", err)
	}
}
