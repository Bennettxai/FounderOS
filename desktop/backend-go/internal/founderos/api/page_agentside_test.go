package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/paperclip"
	"github.com/rhl/businessos-backend/internal/founderos/guard"
	"github.com/rhl/businessos-backend/internal/founderos/pages/osdata/osdatatest"
)

// fakeBoard is a scripted Paperclip board for the agent-side page tests.
// down makes every read fail the way an unreachable board does; writes
// honour guard.Outbound exactly like the real connector.
type fakeBoard struct {
	mu       sync.Mutex
	down     bool
	agents   []paperclip.Agent
	issues   []paperclip.Issue
	runs     []paperclip.Run
	failRuns []paperclip.FailoverRun
	calls    []string
}

var errBoardDown = errors.New("board unreachable (dial tcp: timeout)")

func (f *fakeBoard) log(s string) { f.mu.Lock(); f.calls = append(f.calls, s); f.mu.Unlock() }

func (f *fakeBoard) Agents(context.Context) ([]paperclip.Agent, error) {
	if f.down {
		return nil, errBoardDown
	}
	return f.agents, nil
}
func (f *fakeBoard) Issues(_ context.Context, limit int) ([]paperclip.Issue, error) {
	if f.down {
		return nil, errBoardDown
	}
	f.log("issues")
	return f.issues, nil
}
func (f *fakeBoard) Runs(context.Context, int) ([]paperclip.Run, error) {
	if f.down {
		return nil, errBoardDown
	}
	return f.runs, nil
}
func (f *fakeBoard) FailoverRuns(context.Context, int) ([]paperclip.FailoverRun, error) {
	if f.down {
		return nil, errBoardDown
	}
	return f.failRuns, nil
}
func (f *fakeBoard) write(name string) error {
	return guard.Outbound("paperclip."+name, func() error {
		if f.down {
			return errBoardDown
		}
		f.log(name)
		return nil
	})
}
func (f *fakeBoard) CreateIssue(_ context.Context, title, _ string) (*paperclip.Issue, error) {
	if err := f.write("issue.create"); err != nil {
		return nil, err
	}
	return &paperclip.Issue{ID: "new", Identifier: "FOS-1", Title: title, Status: "todo"}, nil
}
func (f *fakeBoard) InvokeHeartbeat(_ context.Context, id string) (bool, error) {
	err := f.write("heartbeat:" + id)
	return err == nil, err
}
func (f *fakeBoard) SetAgentModel(_ context.Context, id, model string) (bool, error) {
	err := f.write("model:" + id + ":" + model)
	return err == nil, err
}
func (f *fakeBoard) ClearAgentError(_ context.Context, id string) (bool, error) {
	err := f.write("clear:" + id)
	return err == nil, err
}
func (f *fakeBoard) ReassignCockpitIssue(_ context.Context, id string) (bool, error) {
	err := f.write("reassign:" + id)
	return err == nil, err
}
func (f *fakeBoard) RepairCockpitIssue(context.Context, string) string {
	if f.write("repair") != nil {
		return "failed"
	}
	return "ok"
}
func (f *fakeBoard) PostOnce(_ context.Context, marker, _ string) string {
	if f.write("post:"+marker) != nil {
		return "failed"
	}
	return "posted"
}

func sp(s string) *string { return &s }

// pageDeps is Deps on a seeded throwaway DB with a fake board.
func pageDeps(t *testing.T, board *fakeBoard) *Deps {
	t.Helper()
	pool, ws := osdatatest.ThrowawayDB(t)
	osdatatest.Seed(t, pool, ws, nowForTests())
	d := &Deps{Pool: pool, Board: connectors.NewRegistry()}
	if board != nil {
		SetBoardSource(d, board)
	}
	return d
}

// getJSON performs an authed request and decodes the body into out.
func getJSON(t *testing.T, d *Deps, method, path string, body []byte, out any) int {
	t.Helper()
	w := httptest.NewRecorder()
	router(t, d).ServeHTTP(w, authed(method, path, body))
	if out != nil && w.Body.Len() > 0 {
		if err := json.Unmarshal(w.Body.Bytes(), out); err != nil {
			t.Fatalf("%s %s: bad json %v: %s", method, path, err, w.Body)
		}
	}
	return w.Code
}

func TestBoardSourceIsBuiltOncePerDeps(t *testing.T) {
	d := &Deps{}
	if boardFor(d) != boardFor(d) {
		t.Fatal("board rebuilt per call: the breaker state would be lost")
	}
	fb := &fakeBoard{}
	SetBoardSource(d, fb)
	if boardFor(d) != BoardSource(fb) {
		t.Fatal("swap ignored")
	}
}

func TestDataErrMapsMissingWorkspaceTo503(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := ginTestContext(w)
	dataErr(c, errors.New("no database"))
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("code %d", w.Code)
	}
}

// nowForTests is the fixed-ish clock the seed hangs off (second precision).
var testNow = time.Now().UTC().Truncate(time.Second)

func nowForTests() time.Time { return testNow }

func ginTestContext(w *httptest.ResponseRecorder) (*gin.Context, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	return gin.CreateTestContext(w)
}
