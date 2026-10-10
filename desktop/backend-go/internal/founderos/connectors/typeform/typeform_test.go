package typeform

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
)

const formsBody = `{"total_items":3,"page_count":1,"items":[
 {"id":"frmA","title":"Apply to work with us","last_updated_at":"2026-09-20T10:00:00Z"},
 {"id":"frmB","title":"","last_updated_at":null},
 {"id":"frmC","title":"Old quiz"},
 {"title":"no id"}
]}`

const defA = `{"id":"frmA","fields":[
 {"id":"f1","title":"What's your first name?","type":"short_text","ref":"first_name"},
 {"id":"g1","title":"About you","type":"group","properties":{"fields":[
   {"id":"f2","title":"Last name","type":"short_text"},
   {"id":"f3","title":"Company name","type":"short_text"}
 ]}},
 {"id":"f4","title":"Email","type":"email"},
 {"id":"f5","title":"Anything else?","type":"long_text"}
]}`

const respA = `{"total_items":4,"page_count":1,"items":[
 {"response_id":"r1","submitted_at":"2026-09-25T10:00:00Z","hidden":{"utm_source":"ig","blank":"  "},"answers":[
   {"field":{"id":"f1","type":"short_text","ref":"first_name"},"type":"text","text":"Ada"},
   {"field":{"id":"f2","type":"short_text"},"type":"text","text":"Lovelace"},
   {"field":{"id":"f3","type":"short_text"},"type":"text","text":"Acme Agency"},
   {"field":{"id":"f4","type":"email"},"type":"email","email":" Ada@Example.COM "},
   {"field":{"id":"f5","type":"long_text"},"type":"text","text":"name drop"},
   {"field":{"id":"f6","type":"phone_number"},"type":"phone_number","phone_number":"+15555550100"}
 ]},
 {"token":"r2","submitted_at":"2026-09-26T10:00:00Z","answers":[{"field":{"id":"f5"},"type":"text","text":"just a note"}]},
 {"response_id":"r3","submitted_at":"2026-09-27T10:00:00Z","hidden":{"email":"Hidden@X.io","name":"Grace H"},"answers":[]},
 {"response_id":"r4","answers":[{"type":"email","email":"no-submit@x.io"}]}
]}`

func resolver(t *testing.T, body string) connectors.Resolver {
	t.Helper()
	t.Setenv("TYPEFORM_API_KEY", "")
	t.Setenv("TYPEFORM_FORM_IDS", "")
	p := filepath.Join(t.TempDir(), "env.local")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return connectors.Resolver{EnvLocal: p}
}

type fake struct {
	mu        sync.Mutex
	paths     []string
	status    int
	failForms map[string]bool
}

func (f *fake) serve(t *testing.T) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.paths = append(f.paths, r.URL.Path+"?"+r.URL.RawQuery)
		status := f.status
		f.mu.Unlock()
		if r.Header.Get("Authorization") != "Bearer tfp_test" || r.Header.Get("Accept") != "application/json" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		if status != 0 {
			w.WriteHeader(status)
			return
		}
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		switch {
		case len(parts) == 1 && parts[0] == "forms":
			fmt.Fprint(w, formsBody)
		case len(parts) >= 2 && f.failForms[parts[1]]:
			w.WriteHeader(http.StatusInternalServerError)
		case len(parts) == 2 && parts[1] == "frmA":
			fmt.Fprint(w, defA)
		case len(parts) == 3 && parts[1] == "frmA" && parts[2] == "responses":
			fmt.Fprint(w, respA)
		case len(parts) == 2:
			fmt.Fprint(w, `{"fields":[]}`)
		case len(parts) == 3:
			fmt.Fprint(w, `{"items":[]}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func newTest(t *testing.T, f *fake, env string, files ...string) *Connector {
	c := New(resolver(t, env))
	c.baseURL = f.serve(t).URL
	c.credFiles = files
	return c
}

func TestMetaMatchesFounderosOS(t *testing.T) {
	if Meta != (connectors.Meta{ID: "typeform", Name: "Typeform", Kind: connectors.KindCRM}) {
		t.Fatalf("Meta = %+v", Meta)
	}
}

func TestStatusNotConfigured(t *testing.T) {
	f := &fake{}
	c := newTest(t, f, "")
	s := c.Status(context.Background())
	if s.State != connectors.StateNotConfigured || !strings.HasPrefix(s.Detail, "Set TYPEFORM_API_KEY to pull lead-form submissions into the funnel.") {
		t.Fatalf("status = %+v", s)
	}
	if len(f.paths) != 0 {
		t.Fatal("unconfigured must not call out")
	}
}

func TestKeyFallsBackToClueAgentThenSocialFiles(t *testing.T) {
	dir := t.TempDir()
	clue := filepath.Join(dir, "clue.env")
	social := filepath.Join(dir, "social.env")
	os.WriteFile(social, []byte("TYPEFORM_API_KEY=tfp_test\n"), 0o600)
	os.WriteFile(clue, []byte("OTHER=1\n"), 0o600)
	c := newTest(t, &fake{}, "", clue, social)
	if s := c.Status(context.Background()); s.State != connectors.StateConnected {
		t.Fatalf("file fallback: %+v", s)
	}
}

func TestStatusConnected(t *testing.T) {
	f := &fake{}
	c := newTest(t, f, "TYPEFORM_API_KEY=tfp_test\n")
	s := c.Status(context.Background())
	if s.State != connectors.StateConnected || s.Detail != "Typeform reachable · 3 forms visible to this token" || s.Meta["forms"] != 3 {
		t.Fatalf("status = %+v", s)
	}
	if f.paths[0] != "/forms?page_size=1" {
		t.Fatalf("status path = %s", f.paths[0])
	}
}

func TestStatusError(t *testing.T) {
	f := &fake{status: 401}
	c := newTest(t, f, "TYPEFORM_API_KEY=tfp_test\n")
	if s := c.Status(context.Background()); s.State != connectors.StateError || s.Detail != "TYPEFORM_API_KEY is set but the call failed: HTTP 401" {
		t.Fatalf("status = %+v", s)
	}
	c.baseURL = "http://127.0.0.1:1"
	if s := c.Status(context.Background()); s.State != connectors.StateError {
		t.Fatalf("unreachable: %+v", s)
	}
}

func TestParseFormsAndFieldTitles(t *testing.T) {
	forms := ParseForms([]byte(formsBody))
	if len(forms) != 3 || forms[1].Title != "Untitled form" || forms[1].LastUpdatedAt != "" || forms[0].LastUpdatedAt != "2026-09-20T10:00:00Z" {
		t.Fatalf("forms = %+v", forms)
	}
	titles := FieldTitles([]byte(defA))
	if titles["f2"] != "Last name" || titles["f3"] != "Company name" || len(titles) != 6 {
		t.Fatalf("titles = %+v", titles)
	}
}

func TestParseResponsesPicksThePerson(t *testing.T) {
	leads := ParseResponses([]byte(respA), Form{ID: "frmA", Title: "Apply"}, FieldTitles([]byte(defA)))
	if len(leads) != 2 {
		t.Fatalf("leads = %+v", leads)
	}
	a := leads[0]
	if a.ResponseID != "r1" || a.Name != "Ada Lovelace" || a.Email != "ada@example.com" || a.Phone != "+15555550100" ||
		a.FormID != "frmA" || a.FormTitle != "Apply" || a.SubmittedAt != "2026-09-25T10:00:00Z" {
		t.Fatalf("lead r1 = %+v", a)
	}
	if len(a.Hidden) != 1 || a.Hidden["utm_source"] != "ig" {
		t.Fatalf("hidden = %+v", a.Hidden)
	}
	g := leads[1]
	if g.ResponseID != "r3" || g.Name != "Grace H" || g.Email != "hidden@x.io" || g.Phone != "" {
		t.Fatalf("lead r3 = %+v", g)
	}
}

func TestLeadsNotConfiguredIsNoSourceNotNoLeads(t *testing.T) {
	c := newTest(t, &fake{}, "")
	if _, err := c.Leads(context.Background(), LeadOptions{}); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("err = %v", err)
	}
	f := &fake{status: 500}
	c = newTest(t, f, "TYPEFORM_API_KEY=tfp_test\n")
	if leads, err := c.Leads(context.Background(), LeadOptions{}); err == nil || leads != nil {
		t.Fatalf("forms failure must be an error, got %v %v", leads, err)
	}
}

func TestLeadsReadsMostRecentFormsSequentiallyWithSince(t *testing.T) {
	f := &fake{failForms: map[string]bool{"frmB": true}}
	c := newTest(t, f, "TYPEFORM_API_KEY=tfp_test\n")
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	leads, err := c.Leads(context.Background(), LeadOptions{Now: now, Days: 30, MaxForms: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(leads) != 2 {
		t.Fatalf("frmB failing must cost only its own leads: %+v", leads)
	}
	want := []string{
		"/forms?page_size=200&sort_by=last_updated_at&order_by=desc",
		"/forms/frmA?",
		"/forms/frmA/responses?page_size=200&response_type=completed&since=2026-08-30T12%3A00%3A00.000Z",
		"/forms/frmB?",
	}
	if fmt.Sprint(f.paths) != fmt.Sprint(want) {
		t.Fatalf("paths:\n%v\nwant\n%v", f.paths, want)
	}
}

func TestLeadsPinnedFormIDs(t *testing.T) {
	f := &fake{}
	c := newTest(t, f, "TYPEFORM_API_KEY=tfp_test\nTYPEFORM_FORM_IDS= frmC , frmA,\n")
	leads, err := c.Leads(context.Background(), LeadOptions{Now: time.Date(2026, 9, 29, 0, 0, 0, 0, time.UTC)})
	if err != nil || len(leads) != 2 {
		t.Fatalf("leads=%v err=%v", leads, err)
	}
	var read []string
	for _, p := range f.paths[1:] {
		if !strings.Contains(p, "/responses") {
			read = append(read, strings.TrimSuffix(p, "?"))
		}
	}
	// Pinned forms keep the listing's order (most recently updated first).
	if fmt.Sprint(read) != "[/forms/frmA /forms/frmC]" {
		t.Fatalf("read = %v", read)
	}
	// Default window is 90 days.
	if !strings.Contains(f.paths[2], "since=2026-07-01T00%3A00%3A00.000Z") {
		t.Fatalf("since = %s", f.paths[2])
	}
}

func TestLeadsEmptyIsEmptyNotNil(t *testing.T) {
	f := &fake{}
	c := newTest(t, f, "TYPEFORM_API_KEY=tfp_test\nTYPEFORM_FORM_IDS=frmC\n")
	leads, err := c.Leads(context.Background(), LeadOptions{})
	if err != nil || leads == nil || len(leads) != 0 {
		t.Fatalf("a keyed read with no leads must be an empty list: %v %v", leads, err)
	}
}
