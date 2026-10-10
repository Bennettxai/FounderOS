package console

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// Ports lib/interject.ts's contract (FounderOS v1 tests/interject*.test.ts).

func TestClassifyInterject(t *testing.T) {
	for text, want := range map[string]string{
		"task: ship the invoices":      "task",
		"remember the todo list":       "task",
		"tell comms-agent to hush":     "agent",
		"ask markets what moved":       "agent",
		"the offer lands better short": "note",
		"task ask bob":                 "task", // task patterns win
	} {
		if got := ClassifyInterject(text); got != want {
			t.Errorf("ClassifyInterject(%q) = %q, want %q", text, got, want)
		}
	}
}

type fakeInterject struct {
	title, desc string
	note        NoteInput
	taskErr     error
	noteErr     error
}

func (f *fakeInterject) CreateTask(_ context.Context, title, description string) (string, string, error) {
	f.title, f.desc = title, description
	if f.taskErr != nil {
		return "", "", f.taskErr
	}
	return "FOS-12", "http://board/issues/1", nil
}

func (f *fakeInterject) CaptureNote(_ context.Context, n NoteInput) (string, error) {
	f.note = n
	if f.noteErr != nil {
		return "", f.noteErr
	}
	return "sig-1", nil
}

func TestPerformInterjectRoutesTasksToTheBoard(t *testing.T) {
	f := &fakeInterject{}
	r := PerformInterject(context.Background(), "todo: call Vantage\nabout the retainer", "", f, time.Now())
	if !r.OK || r.Route != "task" || r.Ref != "FOS-12" || r.URL != "http://board/issues/1" {
		t.Fatalf("receipt = %+v", r)
	}
	if f.title != "call Vantage" || !strings.Contains(f.desc, "route: task") {
		t.Fatalf("task = %q / %q", f.title, f.desc)
	}
	f = &fakeInterject{}
	r = PerformInterject(context.Background(), "tell comms to hold replies", "", f, time.Now())
	if r.Route != "agent" || f.title != "tell comms to hold replies" {
		t.Fatalf("agent relay keeps its phrasing: %+v %q", r, f.title)
	}
}

func TestPerformInterjectNotesGoToTheOptimalEngine(t *testing.T) {
	f := &fakeInterject{}
	now := time.Date(2026, 9, 30, 10, 0, 0, 0, time.UTC)
	r := PerformInterject(context.Background(), "the offer lands better short", "", f, now)
	if !r.OK || r.Route != "note" || r.Slug != "inbox/2026-09-30-note" {
		t.Fatalf("receipt = %+v", r)
	}
	if f.note.Title != "Interject 2026-09-30" || f.note.Text != "the offer lands better short" {
		t.Fatalf("note = %+v", f.note)
	}
}

func TestPerformInterjectReportsFailuresInsteadOfFakeReceipts(t *testing.T) {
	f := &fakeInterject{taskErr: errors.New("bridge: outbound writes are disabled (FOUNDEROS_WRITES=0)")}
	r := PerformInterject(context.Background(), "anything", "task", f, time.Now())
	if r.OK || r.Route != "task" || !strings.Contains(r.Error, "FOUNDEROS_WRITES=0") {
		t.Fatalf("receipt = %+v", r)
	}
	f = &fakeInterject{noteErr: errors.New("engine hub unreachable")}
	r = PerformInterject(context.Background(), "anything", "note", f, time.Now())
	if r.OK || r.Error != "engine hub unreachable" {
		t.Fatalf("receipt = %+v", r)
	}
}
