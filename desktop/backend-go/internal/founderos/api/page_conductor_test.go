package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/paperclip"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/superset"
	"github.com/rhl/businessos-backend/internal/founderos/guard"
	"github.com/rhl/businessos-backend/internal/founderos/memory"
)

type conductorFakeBoard struct {
	thread    []paperclip.Comment
	threadErr error
	postErr   error
	issue     *paperclip.Issue
	agents    []paperclip.Agent
}

func (f *conductorFakeBoard) CockpitThread(context.Context, int) ([]paperclip.Comment, error) {
	if f.threadErr != nil {
		return nil, f.threadErr
	}
	return f.thread, nil
}
func (f *conductorFakeBoard) PostCockpitMessage(_ context.Context, m string) (*paperclip.Comment, error) {
	if f.postErr != nil {
		return nil, f.postErr
	}
	return &paperclip.Comment{ID: "c1", Body: m}, nil
}
func (f *conductorFakeBoard) CreateIssue(_ context.Context, title, _ string) (*paperclip.Issue, error) {
	if f.postErr != nil {
		return nil, f.postErr
	}
	f.issue = &paperclip.Issue{ID: "i1", Identifier: "FOS-77", Title: title}
	return f.issue, nil
}
func (f *conductorFakeBoard) Agents(context.Context) ([]paperclip.Agent, error) { return f.agents, nil }

type fakeDispatch struct{ err error }

func (f fakeDispatch) DispatchCodingTask(context.Context, string) (superset.DispatchResult, error) {
	return superset.DispatchResult{WorkspaceID: "w1", Branch: "ui/x"}, f.err
}

type fakeCapture struct{ got memory.Capture }

func (f *fakeCapture) Capture(_ context.Context, c memory.Capture) (string, error) {
	f.got = c
	return "sig-9", nil
}

func conductorRouter(t *testing.T, cd *ConductorDeps) http.Handler {
	d := &Deps{Board: connectors.NewRegistry(), Conductor: cd}
	return router(t, d)
}

func TestConductorChat(t *testing.T) {
	board := &conductorFakeBoard{thread: []paperclip.Comment{{ID: "c0", Body: "hi"}}}
	r := conductorRouter(t, &ConductorDeps{Board: board})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, authed(http.MethodGet, "/api/founderos/pages/conductor/chat", nil))
	if w.Code != 200 || !bytes.Contains(w.Body.Bytes(), []byte(`"messages":[{"id":"c0"`)) {
		t.Fatalf("GET chat: %d %s", w.Code, w.Body)
	}
	w = httptest.NewRecorder()
	r.ServeHTTP(w, authed(http.MethodPost, "/api/founderos/pages/conductor/chat", []byte(`{"message":"  "}`)))
	if w.Code != 400 {
		t.Fatalf("empty message: %d", w.Code)
	}
	w = httptest.NewRecorder()
	r.ServeHTTP(w, authed(http.MethodPost, "/api/founderos/pages/conductor/chat", []byte(`{"message":"ship it"}`)))
	if w.Code != 201 {
		t.Fatalf("POST chat: %d %s", w.Code, w.Body)
	}
	board.postErr = guard.ErrWritesDisabled
	w = httptest.NewRecorder()
	r.ServeHTTP(w, authed(http.MethodPost, "/api/founderos/pages/conductor/chat", []byte(`{"message":"ship it"}`)))
	if w.Code != http.StatusConflict || !bytes.Contains(w.Body.Bytes(), []byte(`"refused":true`)) || !bytes.Contains(w.Body.Bytes(), []byte(`"guarded":true`)) {
		t.Fatalf("refused write must be 409 refused, got %d %s", w.Code, w.Body)
	}
	board.postErr = errors.New("board unreachable")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, authed(http.MethodPost, "/api/founderos/pages/conductor/chat", []byte(`{"message":"ship it"}`)))
	if w.Code != http.StatusBadGateway {
		t.Fatalf("upstream failure: %d", w.Code)
	}
}

// FounderOS v1 /api/conductor/chat: with Paperclip unset (or unreachable) the
// thread read is an empty thread, never a 5xx; a send says so honestly
// without a 5xx either.
func TestConductorChatNotConfigured(t *testing.T) {
	board := &conductorFakeBoard{threadErr: errors.New("paperclip creds missing"), postErr: errors.New("paperclip creds missing")}
	r := conductorRouter(t, &ConductorDeps{Board: board})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, authed(http.MethodGet, "/api/founderos/pages/conductor/chat", nil))
	var got struct {
		Messages   []any `json:"messages"`
		Configured *bool `json:"configured"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil || w.Code != 200 {
		t.Fatalf("GET chat unconfigured must be 200: %d %s", w.Code, w.Body)
	}
	if got.Messages == nil || len(got.Messages) != 0 || got.Configured == nil || *got.Configured {
		t.Fatalf("want empty messages + configured:false, got %s", w.Body)
	}
	board.threadErr = errors.New("board unreachable")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, authed(http.MethodGet, "/api/founderos/pages/conductor/chat", nil))
	if w.Code != 200 || !bytes.Contains(w.Body.Bytes(), []byte(`"messages":[]`)) {
		t.Fatalf("unreachable board reads as an empty thread: %d %s", w.Code, w.Body)
	}
	w = httptest.NewRecorder()
	r.ServeHTTP(w, authed(http.MethodPost, "/api/founderos/pages/conductor/chat", []byte(`{"message":"ship it"}`)))
	if w.Code >= 500 || !bytes.Contains(w.Body.Bytes(), []byte(`"notConfigured":true`)) {
		t.Fatalf("send with Paperclip unset must not be 5xx: %d %s", w.Code, w.Body)
	}
}

func TestConductorContextAndDispatch(t *testing.T) {
	model := "opus-4.8"
	r := conductorRouter(t, &ConductorDeps{Board: &conductorFakeBoard{agents: []paperclip.Agent{{Name: "Conductor", Model: &model}}}, Dispatcher: fakeDispatch{}})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, authed(http.MethodGet, "/api/founderos/pages/conductor/context?path=/os/funnel", nil))
	var ctxBody struct {
		Title        string `json:"title"`
		Model        string `json:"model"`
		QuickActions []any  `json:"quickActions"`
	}
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &ctxBody) != nil || ctxBody.Title != "Funnel" || ctxBody.Model != "opus-4.8" || len(ctxBody.QuickActions) != 4 {
		t.Fatalf("context: %d %s", w.Code, w.Body)
	}
	w = httptest.NewRecorder()
	r.ServeHTTP(w, authed(http.MethodPost, "/api/founderos/pages/conductor/dispatch", []byte(`{"request":"make the funnel denser"}`)))
	if w.Code != 201 || !bytes.Contains(w.Body.Bytes(), []byte(`"branch":"ui/x"`)) {
		t.Fatalf("dispatch: %d %s", w.Code, w.Body)
	}
}
