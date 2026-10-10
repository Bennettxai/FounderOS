package plaud

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/guard"
)

// Fixtures mirror platform.plaud.ai's third-party API (the one @plaud-ai/mcp
// calls). Titles and content are invented except Plaud's own sample titles.
const filesBody = `{
  "type": "list",
  "data": [
    {"id": "84caa034a1b64ca0087585850c672cf7", "name": "Northgate Storage on-site walkthrough",
     "created_at": "2026-08-26T02:10:35", "start_at": "2026-08-26T02:10:35.468000", "duration": 4865796},
    {"id": "d434a33857b081647a8910922842ecf8", "name": "Welcome to Plaud.ai",
     "created_at": "2026-08-26T02:10:34", "duration": 253260},
    {"id": 42, "start_at": "2026-08-20T10:00:00+02:00"}
  ],
  "page": 1,
  "page_size": 20
}`

var fileBody = func() string {
	segs, _ := json.Marshal([]map[string]any{
		{"start_time": 0, "end_time": 31200, "content": "We walked the north lot first.", "speaker": "Alex"},
		{"start_time": 31200, "end_time": 40000, "content": "  ", "speaker": "Sam"},
		{"start_time": 40000, "end_time": 52000, "content": "Gate two needs a reader.", "speaker": " "},
	})
	body, _ := json.Marshal(map[string]any{
		"id": "f1", "name": "Site walk", "start_at": "2026-09-10T15:00:00", "duration": 600000,
		"source_list": []any{map[string]any{"data_type": "transaction", "data_content": string(segs)}},
		"note_list": []any{map[string]any{"data_type": "auto_sum_note", "data_title": "Summary",
			"data_content": "## Overview\nSite walk with Sam.\n\n## Action Items\n* Send the gate-two reader quote by **Friday**\n- Confirm the Sept 16 on-site window\n\n## Notes\n- Nothing else."}},
	})
	return string(body)
}()

const (
	goodRefresh = "rt_good"
	goodAccess  = "at_fresh"
)

type hit struct {
	method, path, rawQuery, auth, body string
}

type fake struct {
	mu       sync.Mutex
	hits     []hit
	rotateTo string
	srv      *httptest.Server
}

// newFake is a Plaud backend: the files endpoints want a Bearer token it
// knows; the refresh endpoint mints one from a refresh token it knows.
func newFake(t *testing.T) *fake {
	t.Helper()
	f := &fake{}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		f.mu.Lock()
		f.hits = append(f.hits, hit{r.Method, r.URL.Path, r.URL.RawQuery, r.Header.Get("Authorization"), string(b)})
		rotate := f.rotateTo
		f.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/oauth/third-party/access-token/refresh":
			form, _ := url.ParseQuery(string(b))
			if r.Method != http.MethodPost || r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" || form.Get("refresh_token") != goodRefresh {
				w.WriteHeader(401)
				_, _ = w.Write([]byte(`{"detail":"invalid refresh token"}`))
				return
			}
			resp := map[string]any{"access_token": goodAccess, "token_type": "Bearer", "expires_in": 3600}
			if rotate != "" {
				resp["refresh_token"] = rotate
			}
			_ = json.NewEncoder(w).Encode(resp)
		case r.Header.Get("Authorization") != "Bearer "+goodAccess:
			w.WriteHeader(401)
			_, _ = w.Write([]byte(`{}`))
		case r.URL.Path == "/open/third-party/files/":
			_, _ = w.Write([]byte(filesBody))
		case r.URL.Path == "/open/third-party/files/f1":
			_, _ = w.Write([]byte(fileBody))
		case r.URL.Path == "/open/third-party/files/gone":
			w.WriteHeader(404)
			_, _ = w.Write([]byte(`{}`))
		default:
			w.WriteHeader(500)
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fake) calls() []hit {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]hit(nil), f.hits...)
}

func (f *fake) refreshes() int {
	n := 0
	for _, h := range f.calls() {
		if strings.HasSuffix(h.path, "/refresh") {
			n++
		}
	}
	return n
}

var clock = time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)

type setup struct {
	envLocal, tokenFile string
}

// newConn builds a connector over temp files: envLines go into env.local, and
// tokenJSON (when non-empty) is the MCP token file.
func newConn(t *testing.T, base, envLines, tokenJSON string) (*Connector, setup) {
	t.Helper()
	dir := t.TempDir()
	s := setup{envLocal: filepath.Join(dir, "env.local"), tokenFile: filepath.Join(dir, "tokens-mcp.json")}
	if envLines != "" {
		if err := os.WriteFile(s.envLocal, []byte(envLines), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if tokenJSON != "" {
		if err := os.WriteFile(s.tokenFile, []byte(tokenJSON), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PLAUD_REFRESH_TOKEN", "")
	t.Setenv("PLAUD_ACCESS_TOKEN", "")
	t.Setenv("PLAUD_TOKEN_FILE", "")
	c := New(connectors.Resolver{EnvLocal: s.envLocal})
	c.BaseURL = base
	c.TokenFile = s.tokenFile
	c.Now = func() time.Time { return clock }
	return c, s
}

func tokenJSON(access, refresh string, expiresAt time.Time) string {
	b, _ := json.Marshal(map[string]any{"access_token": access, "refresh_token": refresh, "token_type": "Bearer", "expires_at": expiresAt.UnixMilli()})
	return string(b)
}

func TestMetaMatchesFounderosOS(t *testing.T) {
	if Meta.ID != "plaud" || Meta.Name != "Plaud" || Meta.Kind != connectors.KindKnowledge {
		t.Fatalf("meta = %+v", Meta)
	}
}

func TestParseFiles(t *testing.T) {
	rows := ParseFiles([]byte(filesBody))
	if len(rows) != 3 {
		t.Fatalf("rows = %+v", rows)
	}
	r := rows[0]
	if r.ID != "84caa034a1b64ca0087585850c672cf7" || r.Title != "Northgate Storage on-site walkthrough" ||
		r.At != "2026-08-26T02:10:35Z" || r.DurationMinutes == nil || *r.DurationMinutes != 81 {
		t.Errorf("row0 = %+v", r)
	}
	if rows[1].At != "2026-08-26T02:10:34Z" || *rows[1].DurationMinutes != 4 {
		t.Errorf("row1 falls back to created_at: %+v", rows[1])
	}
	if rows[2].ID != "42" || rows[2].Title != "Untitled recording" || rows[2].At != "2026-08-20T10:00:00+02:00" || rows[2].DurationMinutes != nil {
		t.Errorf("row2 = %+v", rows[2])
	}
	if len(ParseFiles([]byte(`{}`))) != 0 || len(ParseFiles([]byte(`nope`))) != 0 {
		t.Error("unknown shapes are empty")
	}
}

func TestParseFile(t *testing.T) {
	f := ParseFile([]byte(fileBody))
	if f == nil || !f.Transcribed || len(f.Transcript) != 2 {
		t.Fatalf("file = %+v", f)
	}
	if f.Transcript[0].Speaker == nil || *f.Transcript[0].Speaker != "Alex" || f.Transcript[0].EndMs != 31200 ||
		f.Transcript[1].Speaker != nil || f.Transcript[1].Text != "Gate two needs a reader." {
		t.Errorf("transcript = %+v", f.Transcript)
	}
	if f.Note == nil || !strings.HasPrefix(*f.Note, "## Overview") || f.At != "2026-09-10T15:00:00Z" || *f.DurationMinutes != 10 {
		t.Errorf("file = %+v", f)
	}
	bare := ParseFile([]byte(`{"id":"x","name":"n","source_list":[],"note_list":[]}`))
	if bare == nil || bare.Transcribed || len(bare.Transcript) != 0 || bare.Note != nil {
		t.Errorf("bare = %+v", bare)
	}
	if ParseFile([]byte(`null`)) != nil {
		t.Error("null is no file")
	}
	odd := ParseFile([]byte(`{"source_list":[{"data_type":"transaction","data_content":"not json"}]}`))
	if odd == nil || odd.Transcribed {
		t.Errorf("unreadable transcript is not transcribed: %+v", odd)
	}
}

func TestActionItems(t *testing.T) {
	f := ParseFile([]byte(fileBody))
	got := ActionItems(f.Note)
	if strings.Join(got, "|") != "Send the gate-two reader quote by Friday|Confirm the Sept 16 on-site window" {
		t.Errorf("got %q", got)
	}
	got = ActionItems(strPtr("## Next steps\n1. Call back\n2) Send deck\n## Other\n- no"))
	if strings.Join(got, "|") != "Call back|Send deck" {
		t.Errorf("got %q", got)
	}
	if ActionItems(nil) != nil {
		t.Error("no note, no items")
	}
}

func TestIsSample(t *testing.T) {
	for _, title := range []string{"Welcome to Plaud.ai", " how to use plaud ", "Steve Jobs & Bill Gates: A Conversation That Shaped Technology"} {
		if !IsSample(title) {
			t.Errorf("%q is a bundled sample", title)
		}
	}
	if IsSample("Northgate Storage on-site walkthrough") {
		t.Error("a real recording is not a sample")
	}
}

func TestStatusNotConfigured(t *testing.T) {
	f := newFake(t)
	c, _ := newConn(t, f.srv.URL, "", "")
	s := c.Status(context.Background())
	if s.State != connectors.StateNotConfigured || !strings.Contains(s.Detail, "PLAUD_REFRESH_TOKEN") || len(f.calls()) != 0 {
		t.Fatalf("status = %+v calls=%d", s, len(f.calls()))
	}
	if c.Configured() {
		t.Error("not configured")
	}
}

func TestStatusConnectedWithFreshTokenFileMakesNoRefresh(t *testing.T) {
	f := newFake(t)
	c, _ := newConn(t, f.srv.URL, "", tokenJSON(goodAccess, goodRefresh, clock.Add(time.Hour)))
	s := c.Status(context.Background())
	if s.State != connectors.StateConnected || s.Detail != "Plaud reachable · 3 recordings visible to this account" || s.Meta["recordings"] != 3 {
		t.Fatalf("status = %+v", s)
	}
	if f.refreshes() != 0 {
		t.Error("a fresh access token needs no refresh")
	}
	if h := f.calls()[0]; h.rawQuery != "page=1&page_size=100" || h.auth != "Bearer "+goodAccess {
		t.Errorf("list call = %+v", h)
	}
}

func TestStaleTokenFileRefreshesAndWritesRotatedTokenBack(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "1") // after cutover the bridge owns the token
	f := newFake(t)
	f.rotateTo = "rt_rotated"
	c, s := newConn(t, f.srv.URL, "", tokenJSON("at_stale", goodRefresh, clock.Add(-time.Minute)))
	if st := c.Status(context.Background()); st.State != connectors.StateConnected {
		t.Fatalf("status = %+v", st)
	}
	if f.refreshes() != 1 {
		t.Fatalf("refreshes = %d", f.refreshes())
	}
	raw, err := os.ReadFile(s.tokenFile)
	if err != nil {
		t.Fatal(err)
	}
	var saved map[string]any
	if err := json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	if saved["refresh_token"] != "rt_rotated" || saved["access_token"] != goodAccess || saved["token_type"] != "Bearer" ||
		saved["expires_at"] != float64(clock.Add(time.Hour).UnixMilli()) {
		t.Errorf("saved = %v", saved)
	}
	if !strings.HasPrefix(string(raw), "{\n  \"access_token\"") {
		t.Errorf("written like JSON.stringify(set, null, 2): %s", raw)
	}
	// The in-memory token carries the next read: no second refresh.
	if _, err := c.RecentRecordings(context.Background(), 5); err != nil {
		t.Fatal(err)
	}
	if f.refreshes() != 1 {
		t.Errorf("cached access token reused, refreshes = %d", f.refreshes())
	}
}

func TestEnvRefreshTokenRotationUpsertsEnvLocal(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "1") // after cutover the bridge owns the token
	f := newFake(t)
	f.rotateTo = "rt_next"
	c, s := newConn(t, f.srv.URL, "# keep me\nOTHER=1\nPLAUD_REFRESH_TOKEN="+goodRefresh+"\nPLAUD_ACCESS_TOKEN=at_env\n", tokenJSON(goodAccess, "rt_file", clock.Add(time.Hour)))
	if st := c.Status(context.Background()); st.State != connectors.StateConnected {
		t.Fatalf("status = %+v", st)
	}
	// The env access token has no expiry, so it is never trusted: one refresh.
	if f.refreshes() != 1 {
		t.Errorf("refreshes = %d", f.refreshes())
	}
	raw, _ := os.ReadFile(s.envLocal)
	want := "# keep me\nOTHER=1\nPLAUD_REFRESH_TOKEN=rt_next\nPLAUD_ACCESS_TOKEN=at_env\n"
	if string(raw) != want {
		t.Errorf("env.local =\n%s\nwant\n%s", raw, want)
	}
	info, _ := os.Stat(s.envLocal)
	if info.Mode().Perm() != 0o600 {
		t.Errorf("env.local mode = %v", info.Mode())
	}
	tok, _ := os.ReadFile(s.tokenFile)
	if !strings.Contains(string(tok), "rt_file") {
		t.Error("an env-sourced rotation must not touch the MCP token file")
	}
}

func TestRefreshRejectedIsError(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "1") // after cutover the bridge owns the token
	f := newFake(t)
	c, _ := newConn(t, f.srv.URL, "PLAUD_REFRESH_TOKEN=rt_dead\n", "")
	s := c.Status(context.Background())
	if s.State != connectors.StateError || s.Detail != "Plaud credential is set but the call failed: token refresh HTTP 401" {
		t.Fatalf("status = %+v", s)
	}
	if _, err := c.RecentRecordings(context.Background(), 5); err == nil {
		t.Error("a rejected refresh is an error, never an empty list")
	}
}

func TestRecentRecordingsClampsPageSize(t *testing.T) {
	f := newFake(t)
	c, _ := newConn(t, f.srv.URL, "", tokenJSON(goodAccess, goodRefresh, clock.Add(time.Hour)))
	rows, err := c.RecentRecordings(context.Background(), 2)
	if err != nil || len(rows) != 2 {
		t.Fatalf("rows=%+v err=%v", rows, err)
	}
	if q := f.calls()[0].rawQuery; q != "page=1&page_size=10" {
		t.Errorf("the API floors page_size at 10: %q", q)
	}
	none, _ := newConn(t, f.srv.URL, "", "")
	if _, err := none.RecentRecordings(context.Background(), 5); !errors.Is(err, ErrNotConfigured) {
		t.Errorf("err = %v", err)
	}
}

func TestFile(t *testing.T) {
	f := newFake(t)
	c, _ := newConn(t, f.srv.URL, "", tokenJSON(goodAccess, goodRefresh, clock.Add(time.Hour)))
	file, err := c.File(context.Background(), "f1")
	if err != nil || file == nil || !file.Transcribed || file.Note == nil {
		t.Fatalf("file=%+v err=%v", file, err)
	}
	gone, err := c.File(context.Background(), "gone")
	if err != nil || gone != nil {
		t.Errorf("404 is no such file: %+v %v", gone, err)
	}
	if _, err := c.File(context.Background(), "boom"); err == nil || !strings.Contains(err.Error(), "HTTP 500") {
		t.Errorf("err = %v", err)
	}
}

func TestPlaudTokenFileEnvOverride(t *testing.T) {
	f := newFake(t)
	c, s := newConn(t, f.srv.URL, "", tokenJSON(goodAccess, goodRefresh, clock.Add(time.Hour)))
	c.TokenFile = ""
	t.Setenv("PLAUD_TOKEN_FILE", s.tokenFile)
	if st := c.Status(context.Background()); st.State != connectors.StateConnected {
		t.Fatalf("PLAUD_TOKEN_FILE must be honoured: %+v", st)
	}
}

type recordRT struct{ reqs []*http.Request }

func (r *recordRT) RoundTrip(req *http.Request) (*http.Response, error) {
	r.reqs = append(r.reqs, req)
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{}`)), Header: http.Header{}, Request: req}, nil
}

// The OAuth refresh is a POST, but it is a read (it mints a token, sends
// nothing anywhere), so it must pass the guard while FOUNDEROS_WRITES=0.
func TestRefreshPOSTIsAllowedThroughGuard(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "0")
	base := &recordRT{}
	rt := guard.Transport(base, ReadPOSTs...)
	req, _ := http.NewRequest(http.MethodPost, API+"/oauth/third-party/access-token/refresh", strings.NewReader("refresh_token=x"))
	if _, err := rt.RoundTrip(req); err != nil {
		t.Fatalf("refresh refused by guard: %v", err)
	}
	other, _ := http.NewRequest(http.MethodPost, API+"/open/third-party/files/", strings.NewReader("{}"))
	if _, err := rt.RoundTrip(other); !errors.Is(err, guard.ErrWritesDisabled) {
		t.Errorf("only the refresh is a read POST, got %v", err)
	}
}

func strPtr(s string) *string { return &s }

// Prod on the mini shares Plaud's refresh-token lineage. Rotating it from the
// bridge would invalidate prod's copy, so while FOUNDEROS_WRITES=0 a stale token
// is reported, never refreshed.
func TestBridgeNeverRotatesTheSharedRefreshTokenWhileWritesAreOff(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "0")
	f := newFake(t)
	f.rotateTo = "rt_rotated"
	c, s := newConn(t, f.srv.URL, "", tokenJSON("at_stale", goodRefresh, clock.Add(-time.Minute)))
	before, _ := os.ReadFile(s.tokenFile)
	st := c.Status(context.Background())
	if st.State != connectors.StateError || !strings.Contains(st.Detail, "rotate") {
		t.Fatalf("status = %+v, want an error explaining the refresh was withheld", st)
	}
	if f.refreshes() != 0 {
		t.Fatalf("refreshes = %d, want 0", f.refreshes())
	}
	after, _ := os.ReadFile(s.tokenFile)
	if string(before) != string(after) {
		t.Fatal("token file changed while writes were off")
	}
}
