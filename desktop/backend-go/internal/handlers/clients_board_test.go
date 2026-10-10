package handlers

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

// ─── Helper: clientBoardTaskCounts ───────────────────────────────────────────

func boardStrPtr(s string) *string { return &s }

func TestClientBoardTaskCounts_Empty(t *testing.T) {
	open, done := clientBoardTaskCounts([]gin.H{})
	if open != 0 || done != 0 {
		t.Errorf("expected 0/0, got open=%d done=%d", open, done)
	}
}

func TestClientBoardTaskCounts_MixedStatuses(t *testing.T) {
	tasks := []gin.H{
		{"status": boardStrPtr("todo")},
		{"status": boardStrPtr("in_progress")},
		{"status": boardStrPtr("done")},
		{"status": boardStrPtr("completed")},
		{"status": boardStrPtr("cancelled")},
	}
	open, done := clientBoardTaskCounts(tasks)
	if open != 2 {
		t.Errorf("expected open=2, got %d", open)
	}
	if done != 2 {
		t.Errorf("expected done=2, got %d", done)
	}
}

func TestClientBoardTaskCounts_CaseInsensitive(t *testing.T) {
	tasks := []gin.H{
		{"status": boardStrPtr("DONE")},
		{"status": boardStrPtr("Cancelled")},
		{"status": boardStrPtr("TODO")},
	}
	open, done := clientBoardTaskCounts(tasks)
	if open != 1 {
		t.Errorf("expected open=1, got %d", open)
	}
	if done != 1 {
		t.Errorf("expected done=1, got %d", done)
	}
}

func TestClientBoardTaskCounts_NilAndMissingStatus(t *testing.T) {
	// A nil or missing status is treated as open (not done, not cancelled).
	tasks := []gin.H{
		{"status": (*string)(nil)},
		{},
	}
	open, done := clientBoardTaskCounts(tasks)
	if open != 2 {
		t.Errorf("expected open=2, got %d", open)
	}
	if done != 0 {
		t.Errorf("expected done=0, got %d", done)
	}
}

// ─── Helper: boardTextPtr ────────────────────────────────────────────────────

func TestBoardTextPtr_Null(t *testing.T) {
	if got := boardTextPtr(pgtype.Text{Valid: false}); got != nil {
		t.Errorf("expected nil for invalid text, got %v", *got)
	}
}

func TestBoardTextPtr_Value(t *testing.T) {
	got := boardTextPtr(pgtype.Text{String: "active", Valid: true})
	if got == nil || *got != "active" {
		t.Errorf("expected 'active', got %v", got)
	}
}
