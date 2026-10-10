package services

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// SandboxEditState represents the lifecycle state of a sandbox edit session.
type SandboxEditState string

const (
	SandboxEditStatePending   SandboxEditState = "pending"
	SandboxEditStateValidated SandboxEditState = "validated"
	SandboxEditStateApplied   SandboxEditState = "applied"
	SandboxEditStateRejected  SandboxEditState = "rejected"
)

// DiffEntry represents a file-level change between original and edited content.
type DiffEntry struct {
	Filename string `json:"filename"`
	Added    int    `json:"lines_added"`
	Removed  int    `json:"lines_removed"`
	Diff     string `json:"diff"` // simplified unified diff (changed lines only)
}

// SandboxEdit is a forked module edit session held in memory.
type SandboxEdit struct {
	ID         string            `json:"id"`
	TenantID   string            `json:"tenant_id"`
	UserID     string            `json:"user_id"`
	ModuleID   string            `json:"module_id"`
	ModuleName string            `json:"module_name"`
	State      SandboxEditState  `json:"state"`
	Files      map[string]string `json:"files"`              // filename -> current content
	OrigFiles  map[string]string `json:"orig_files"`         // filename -> original content
	Diff       []DiffEntry       `json:"diff,omitempty"`
	Errors     []string          `json:"errors,omitempty"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
}

// SandboxEditService manages in-memory sandbox edit sessions.
// It is safe for concurrent use.
type SandboxEditService struct {
	mu     sync.RWMutex
	edits  map[string]*SandboxEdit
	logger *slog.Logger
}

// NewSandboxEditService creates a new SandboxEditService with the given logger.
func NewSandboxEditService(logger *slog.Logger) *SandboxEditService {
	return &SandboxEditService{
		edits:  make(map[string]*SandboxEdit),
		logger: logger.With("service", "sandbox_edit"),
	}
}

// Fork creates a new sandbox edit session for a module.
// The session starts with a placeholder file so the editor has something to show.
func (s *SandboxEditService) Fork(ctx context.Context, tenantID, userID, moduleID, moduleName string) (*SandboxEdit, error) {
	id := uuid.New().String()

	// Placeholder file so the session is immediately usable.
	placeholder := fmt.Sprintf("// Module: %s\n// Edit this file in the Monaco editor.\n", moduleName)
	placeholderName := "main.go"

	files := map[string]string{
		placeholderName: placeholder,
	}
	origFiles := map[string]string{
		placeholderName: placeholder,
	}

	now := time.Now().UTC()
	edit := &SandboxEdit{
		ID:         id,
		TenantID:   tenantID,
		UserID:     userID,
		ModuleID:   moduleID,
		ModuleName: moduleName,
		State:      SandboxEditStatePending,
		Files:      files,
		OrigFiles:  origFiles,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	s.mu.Lock()
	s.edits[id] = edit
	s.mu.Unlock()

	s.logger.InfoContext(ctx, "sandbox edit forked",
		"id", id,
		"tenant_id", tenantID,
		"user_id", userID,
		"module_id", moduleID,
		"module_name", moduleName,
	)

	return edit, nil
}

// Get retrieves a sandbox edit by ID and verifies tenant ownership.
// Returns an error if the edit does not exist or belongs to a different tenant.
func (s *SandboxEditService) Get(ctx context.Context, id, tenantID string) (*SandboxEdit, error) {
	s.mu.RLock()
	edit, ok := s.edits[id]
	s.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("sandbox edit %q not found", id)
	}
	if edit.TenantID != tenantID {
		return nil, fmt.Errorf("sandbox edit %q: access denied", id)
	}

	return edit, nil
}

// UpdateFile replaces the content of a file in the sandbox.
// Only allowed while the edit is in the "pending" state.
func (s *SandboxEditService) UpdateFile(ctx context.Context, id, tenantID, filename, content string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	edit, ok := s.edits[id]
	if !ok {
		return fmt.Errorf("sandbox edit %q not found", id)
	}
	if edit.TenantID != tenantID {
		return fmt.Errorf("sandbox edit %q: access denied", id)
	}
	if edit.State != SandboxEditStatePending {
		return fmt.Errorf("sandbox edit %q: cannot update file in state %q (must be %q)",
			id, edit.State, SandboxEditStatePending)
	}

	edit.Files[filename] = content
	edit.UpdatedAt = time.Now().UTC()

	s.logger.InfoContext(ctx, "sandbox edit file updated",
		"id", id,
		"filename", filename,
		"content_len", len(content),
	)

	return nil
}

// Validate checks all files for basic issues (empty content).
// On success the state transitions to "validated"; on failure it stays
// "pending" and the Errors field is populated.
func (s *SandboxEditService) Validate(ctx context.Context, id, tenantID string) (*SandboxEdit, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	edit, ok := s.edits[id]
	if !ok {
		return nil, fmt.Errorf("sandbox edit %q not found", id)
	}
	if edit.TenantID != tenantID {
		return nil, fmt.Errorf("sandbox edit %q: access denied", id)
	}

	var errs []string
	for name, content := range edit.Files {
		if strings.TrimSpace(content) == "" {
			errs = append(errs, fmt.Sprintf("file %q is empty", name))
		}
	}

	edit.UpdatedAt = time.Now().UTC()
	if len(errs) > 0 {
		edit.Errors = errs
		// Keep state pending so the user can correct and re-validate.
		s.logger.WarnContext(ctx, "sandbox edit validation failed",
			"id", id,
			"errors", errs,
		)
	} else {
		edit.Errors = nil
		edit.State = SandboxEditStateValidated
		s.logger.InfoContext(ctx, "sandbox edit validated",
			"id", id,
		)
	}

	return edit, nil
}

// Preview generates a line-level diff between the original files and the
// current edited files, and stores the result in edit.Diff.
func (s *SandboxEditService) Preview(ctx context.Context, id, tenantID string) (*SandboxEdit, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	edit, ok := s.edits[id]
	if !ok {
		return nil, fmt.Errorf("sandbox edit %q not found", id)
	}
	if edit.TenantID != tenantID {
		return nil, fmt.Errorf("sandbox edit %q: access denied", id)
	}

	edit.Diff = computeDiff(edit.OrigFiles, edit.Files)
	edit.UpdatedAt = time.Now().UTC()

	s.logger.InfoContext(ctx, "sandbox edit preview generated",
		"id", id,
		"files_changed", len(edit.Diff),
	)

	return edit, nil
}

// Apply marks the sandbox as applied.
// Only allowed when the state is "validated".
func (s *SandboxEditService) Apply(ctx context.Context, id, tenantID, userID string) (*SandboxEdit, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	edit, ok := s.edits[id]
	if !ok {
		return nil, fmt.Errorf("sandbox edit %q not found", id)
	}
	if edit.TenantID != tenantID {
		return nil, fmt.Errorf("sandbox edit %q: access denied", id)
	}
	if edit.State != SandboxEditStateValidated {
		return nil, fmt.Errorf("sandbox edit %q: cannot apply in state %q (must be %q)",
			id, edit.State, SandboxEditStateValidated)
	}

	edit.State = SandboxEditStateApplied
	edit.UpdatedAt = time.Now().UTC()

	s.logger.InfoContext(ctx, "sandbox edit applied",
		"id", id,
		"user_id", userID,
		"module_id", edit.ModuleID,
	)

	return edit, nil
}

// Reject discards the sandbox edit by marking it as rejected.
func (s *SandboxEditService) Reject(ctx context.Context, id, tenantID, userID string) (*SandboxEdit, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	edit, ok := s.edits[id]
	if !ok {
		return nil, fmt.Errorf("sandbox edit %q not found", id)
	}
	if edit.TenantID != tenantID {
		return nil, fmt.Errorf("sandbox edit %q: access denied", id)
	}

	edit.State = SandboxEditStateRejected
	edit.UpdatedAt = time.Now().UTC()

	s.logger.InfoContext(ctx, "sandbox edit rejected",
		"id", id,
		"user_id", userID,
		"module_id", edit.ModuleID,
	)

	return edit, nil
}

// computeDiff produces a DiffEntry per file that changed between orig and
// current. It uses a simple line-count approach: added = lines in current not
// in orig, removed = lines in orig not in current.
func computeDiff(orig, current map[string]string) []DiffEntry {
	// Collect the union of all filenames.
	seen := make(map[string]struct{})
	for k := range orig {
		seen[k] = struct{}{}
	}
	for k := range current {
		seen[k] = struct{}{}
	}

	var entries []DiffEntry
	for filename := range seen {
		origContent := orig[filename]
		curContent := current[filename]
		if origContent == curContent {
			continue
		}

		origLines := splitLines(origContent)
		curLines := splitLines(curContent)

		added, removed, diffText := lineDiff(origLines, curLines)
		entries = append(entries, DiffEntry{
			Filename: filename,
			Added:    added,
			Removed:  removed,
			Diff:     diffText,
		})
	}
	return entries
}

// splitLines splits content into non-empty lines, normalising CRLF.
func splitLines(content string) []string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	raw := strings.Split(content, "\n")
	out := make([]string, 0, len(raw))
	for _, l := range raw {
		out = append(out, l)
	}
	return out
}

// lineDiff computes a minimal line-level diff and returns (added, removed,
// unified-style diff text). It uses a simple set-difference approach: lines
// present in current but not orig are "added"; lines present in orig but not
// current are "removed". This is intentionally simple — no Myers algorithm.
func lineDiff(origLines, curLines []string) (added, removed int, diff string) {
	origSet := make(map[string]int) // line text -> count in orig
	for _, l := range origLines {
		origSet[l]++
	}
	curSet := make(map[string]int)
	for _, l := range curLines {
		curSet[l]++
	}

	var sb strings.Builder

	// Lines removed (in orig but reduced/missing in current).
	for _, l := range origLines {
		if curSet[l] < origSet[l] {
			removed++
			curSet[l]++ // consume to avoid double-counting within this pass
			sb.WriteString("- ")
			sb.WriteString(l)
			sb.WriteByte('\n')
		}
	}

	// Reset to compute additions.
	origSet2 := make(map[string]int)
	for _, l := range origLines {
		origSet2[l]++
	}

	for _, l := range curLines {
		if origSet2[l] < 1 {
			added++
			sb.WriteString("+ ")
			sb.WriteString(l)
			sb.WriteByte('\n')
		} else {
			origSet2[l]--
		}
	}

	return added, removed, sb.String()
}
