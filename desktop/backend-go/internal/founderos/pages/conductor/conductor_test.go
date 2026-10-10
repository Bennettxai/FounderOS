package conductor

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestScreenTitleAcceptsFounderosAndOsPaths(t *testing.T) {
	cases := map[string]string{"/": "Home", "/os": "Home", "/funnel": "Funnel", "/os/funnel?x=1": "Funnel", "/os/social/instagram": "Social", "/os/brain": "Brain", "/nowhere": "/nowhere"}
	for in, want := range cases {
		if got := ScreenTitle(in); got != want {
			t.Errorf("ScreenTitle(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestQuickActionsPerRouteWithDefault(t *testing.T) {
	if qa := QuickActions("/os/funnel"); len(qa) != 4 || qa[0].Label != "Who is going cold?" {
		t.Fatalf("funnel actions = %+v", qa)
	}
	if qa := QuickActions("/os/somewhere"); qa[0].Label != "Summarize this screen" {
		t.Fatalf("default actions = %+v", qa)
	}
	for _, a := range QuickActions("/os/brain") {
		if strings.Contains(a.Prompt, "G-Brain") {
			t.Fatalf("GBrain is retired; brain prompts must not name it: %q", a.Prompt)
		}
	}
}

func TestClassifyInterject(t *testing.T) {
	cases := map[string]Route{"task: ship the funnel": RouteTask, "remember the todo list": RouteTask, "tell sales to call Logan": RouteAgent, "Ask comms about Rae": RouteAgent, "Logan wants a proposal": RouteNote}
	for in, want := range cases {
		if got := Classify(in); got != want {
			t.Errorf("Classify(%q) = %s, want %s", in, got, want)
		}
	}
}

type fakeInterject struct {
	title, desc string
	note        Note
	failTask    error
	failNote    error
}

func (f *fakeInterject) CreateTask(_ context.Context, title, desc string) (string, string, error) {
	f.title, f.desc = title, desc
	return "FOS-42", "http://board/issues/1", f.failTask
}
func (f *fakeInterject) CaptureNote(_ context.Context, n Note) (string, error) {
	f.note = n
	return n.Slug, f.failNote
}

func TestPerformInterject(t *testing.T) {
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	f := &fakeInterject{}
	r := Perform(context.Background(), "task: - ship the funnel\nmore detail", "", f, now)
	if !r.OK || r.Route != RouteTask || r.Ref != "FOS-42" || f.title != "ship the funnel" || !strings.Contains(f.desc, "interjected from the OS home (route: task)") {
		t.Fatalf("task receipt = %+v title=%q", r, f.title)
	}
	r = Perform(context.Background(), "Logan wants a proposal", "", f, now)
	if !r.OK || r.Route != RouteNote || f.note.Slug != "inbox/2026-09-30-note" || f.note.Title != "Interject 2026-09-30" {
		t.Fatalf("note receipt = %+v note=%+v", r, f.note)
	}
	f.failTask = errors.New("bridge: outbound writes are disabled (FOUNDEROS_WRITES=0)")
	r = Perform(context.Background(), "tell sales to call", RouteAgent, f, now)
	if r.OK || !strings.Contains(r.Error, "writes are disabled") {
		t.Fatalf("refused task must be an honest failure: %+v", r)
	}
}
