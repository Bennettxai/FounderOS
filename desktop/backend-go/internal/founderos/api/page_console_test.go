package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/stripe"
	"github.com/rhl/businessos-backend/internal/founderos/pages/console"
)

type consoleFakeInterject struct {
	tasks int
	notes []console.NoteInput
	err   error
}

func (f *consoleFakeInterject) CreateTask(context.Context, string, string) (string, string, error) {
	f.tasks++
	if f.err != nil {
		return "", "", f.err
	}
	return "FOS-1", "", nil
}

func (f *consoleFakeInterject) CaptureNote(_ context.Context, n console.NoteInput) (string, error) {
	f.notes = append(f.notes, n)
	return "sig", f.err
}

func consoleStub(t *testing.T, s consoleSources) {
	t.Helper()
	prev := consoleSourcesFor
	consoleSourcesFor = func(*Deps) consoleSources { return s }
	t.Cleanup(func() { consoleSourcesFor = prev })
}

func consoleHealthySources(ij console.InterjectDeps) consoleSources {
	ws := 4
	return consoleSources{
		statuses: func(context.Context) []connectors.Status {
			return []connectors.Status{{ID: "optimal-engine", State: connectors.StateConnected}, {ID: "slack", State: connectors.StateError}}
		},
		brain: func(context.Context) console.Brain {
			return console.Brain{Connected: true, EnginesUp: 2, EnginesTotal: 2, Workspaces: &ws, Engines: []console.EngineReading{{Name: "hub", State: "connected", Up: true, Workspaces: &ws, Homes: []string{"founderos"}}}}
		},
		roster: func(_ context.Context, now time.Time) (*console.Roster, []console.Run, []console.Run, error) {
			runs := []console.Run{{ID: "r1", AgentID: "a", OK: true, Summary: "done", StartedAt: now.Add(-time.Minute), FinishedAt: now.Add(-time.Minute)}}
			return &console.Roster{Active: 1, Total: 2}, runs, runs, nil
		},
		feed: func(context.Context) ([]console.Item, []console.SourceState) {
			return []console.Item{{Source: "email", Title: "hi", TS: time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)}},
				[]console.SourceState{{Source: "email", State: "ok"}, {Source: "slack", State: "not_configured"}}
		},
		charges:   func(context.Context) ([]console.Charge, error) { return nil, errors.New("stripe: no key") },
		interject: ij,
	}
}

func TestConsoleRequiresASession(t *testing.T) {
	consoleStub(t, consoleHealthySources(&consoleFakeInterject{}))
	r := router(t, &Deps{})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/founderos/pages/console", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("code = %d", w.Code)
	}
}

func TestConsoleComposesTheView(t *testing.T) {
	consoleStub(t, consoleHealthySources(&consoleFakeInterject{}))
	r := router(t, &Deps{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/founderos/pages/console", nil)
	req.Header.Set("Cookie", "session=ok")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("code = %d body %s", w.Code, w.Body)
	}
	var v struct {
		Systems           struct{ Connected, Total int }
		Agents            struct{ Active, Total *int }
		Brain             console.Brain
		Comms             struct{ Inbound int }
		Volume            console.Volume
		ChargedTodayCents *int64
		Done              []console.DoneItem
		Sources           []console.SourceState
		Errors            map[string]string
	}
	if err := json.Unmarshal(w.Body.Bytes(), &v); err != nil {
		t.Fatal(err)
	}
	if v.Systems.Connected != 1 || v.Systems.Total != 2 || *v.Agents.Active != 1 || v.Brain.EnginesUp != 2 || v.Comms.Inbound != 1 {
		t.Fatalf("view = %+v", v)
	}
	if v.ChargedTodayCents != nil || v.Errors["charges"] != "stripe: no key" {
		t.Fatalf("an unreadable Stripe is unknown, not $0: %+v %v", v.Errors, v.ChargedTodayCents)
	}
	if len(v.Done) != 1 || len(v.Sources) != 2 {
		t.Fatalf("done %+v sources %+v", v.Done, v.Sources)
	}
}

func TestConsoleUnkeyedStripeIsNotAnError(t *testing.T) {
	src := consoleHealthySources(&consoleFakeInterject{})
	src.charges = func(context.Context) ([]console.Charge, error) {
		return nil, fmt.Errorf("%w: STRIPE_SECRET_KEY is not set", stripe.ErrNotConfigured)
	}
	consoleStub(t, src)
	r := router(t, &Deps{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/founderos/pages/console", nil)
	req.Header.Set("Cookie", "session=ok")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("code = %d body %s", w.Code, w.Body)
	}
	var v struct {
		ChargedTodayCents *int64
		Errors            map[string]string
	}
	if err := json.Unmarshal(w.Body.Bytes(), &v); err != nil {
		t.Fatal(err)
	}
	if _, ok := v.Errors["charges"]; ok || v.ChargedTodayCents != nil {
		t.Fatalf("unkeyed Stripe must be quiet and unknown: %+v %v", v.Errors, v.ChargedTodayCents)
	}
}

func TestConsoleRosterFailureIsSurfaced(t *testing.T) {
	s := consoleHealthySources(&consoleFakeInterject{})
	s.roster = func(context.Context, time.Time) (*console.Roster, []console.Run, []console.Run, error) {
		return nil, nil, nil, errors.New("founderos workspace missing")
	}
	consoleStub(t, s)
	r := router(t, &Deps{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/founderos/pages/console", nil)
	req.Header.Set("Cookie", "session=ok")
	r.ServeHTTP(w, req)
	var v struct {
		Agents struct{ Active *int }
		Errors map[string]string
	}
	_ = json.Unmarshal(w.Body.Bytes(), &v)
	if w.Code != http.StatusOK || v.Agents.Active != nil || v.Errors["roster"] == "" {
		t.Fatalf("code %d view %+v", w.Code, v)
	}
}

func consolePost(t *testing.T, r http.Handler, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
	req.Header.Set("Cookie", "session=ok")
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

func TestInterjectRoutesAndValidates(t *testing.T) {
	ij := &consoleFakeInterject{}
	consoleStub(t, consoleHealthySources(ij))
	r := router(t, &Deps{})

	if w := consolePost(t, r, "/api/founderos/pages/interject", `{"text":"  "}`); w.Code != http.StatusBadRequest {
		t.Fatalf("blank text = %d", w.Code)
	}
	if w := consolePost(t, r, "/api/founderos/pages/interject", `{"text":"x","route":"email"}`); w.Code != http.StatusBadRequest {
		t.Fatalf("bad route = %d", w.Code)
	}
	if w := consolePost(t, r, "/api/founderos/pages/interject", `nope`); w.Code != http.StatusBadRequest {
		t.Fatalf("bad json = %d", w.Code)
	}
	w := consolePost(t, r, "/api/founderos/pages/interject", `{"text":"the offer lands better short"}`)
	var rc console.Receipt
	_ = json.Unmarshal(w.Body.Bytes(), &rc)
	if w.Code != http.StatusOK || !rc.OK || rc.Route != "note" || len(ij.notes) != 1 {
		t.Fatalf("note = %d %+v", w.Code, rc)
	}
	w = consolePost(t, r, "/api/founderos/pages/interject", `{"text":"anything","route":"task"}`)
	_ = json.Unmarshal(w.Body.Bytes(), &rc)
	if w.Code != http.StatusOK || rc.Ref != "FOS-1" || ij.tasks != 1 {
		t.Fatalf("task = %d %+v", w.Code, rc)
	}
}

func TestInterjectThatCannotLandIsA502(t *testing.T) {
	ij := &consoleFakeInterject{err: errors.New("board unreachable")}
	consoleStub(t, consoleHealthySources(ij))
	r := router(t, &Deps{})
	w := consolePost(t, r, "/api/founderos/pages/interject", `{"text":"todo: x"}`)
	var rc console.Receipt
	_ = json.Unmarshal(w.Body.Bytes(), &rc)
	if w.Code != http.StatusBadGateway || rc.OK || rc.Error == "" {
		t.Fatalf("code %d receipt %+v", w.Code, rc)
	}
}

// A task the bridge guard refused is not an upstream failure: 409, so the
// composer can say "writes are off on the bridge".
func TestInterjectRefusedByTheGuardIsA409(t *testing.T) {
	ij := &consoleFakeInterject{err: errors.New("bridge: outbound writes are disabled (FOUNDEROS_WRITES=0)")}
	consoleStub(t, consoleHealthySources(ij))
	r := router(t, &Deps{})
	w := consolePost(t, r, "/api/founderos/pages/interject", `{"text":"todo: x"}`)
	if w.Code != http.StatusConflict {
		t.Fatalf("code %d", w.Code)
	}
}
