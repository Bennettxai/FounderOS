// Package miosa implements the BusinessOS <-> MIOSA Cloud sync service.
//
// Sync is always opt-in and user-initiated (or periodic if the user enables
// auto-sync in settings). The payload is a WorkspaceManifest that contains
// only configuration objects — never raw business data rows.
//
// Architecture: the Go backend is the single authority for all MIOSA cloud
// interactions. The SvelteKit frontend never contacts api.miosa.ai directly.
package miosa

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// SyncMode mirrors the OSA_MODE environment variable.
type SyncMode string

const (
	ModeLocal SyncMode = "local"
	ModeCloud SyncMode = "cloud"
)

// ConnectionStatus is returned by GET /api/miosa/status.
type ConnectionStatus struct {
	Mode      SyncMode  `json:"mode"`
	Connected bool      `json:"connected"`
	APIKeySet bool      `json:"api_key_set"`
	LastSync  time.Time `json:"last_sync,omitempty"`
	Error     string    `json:"error,omitempty"`
}

// WorkspaceManifest is the sync payload. It contains only configuration
// objects. Raw business data (tasks, contacts, deals, conversations) is
// deliberately excluded.
type WorkspaceManifest struct {
	Version     string               `json:"version"`
	ExportedAt  time.Time            `json:"exported_at"`
	WorkspaceID string               `json:"workspace_id"`
	Settings    WorkspaceSettings    `json:"settings"`
	Agents      []AgentConfig        `json:"agents"`
	Apps        []AppDefinition      `json:"apps"`
	Templates   []TemplateDefinition `json:"templates"`
}

// WorkspaceSettings contains the workspace-level configuration (non-data).
type WorkspaceSettings struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Icon        string          `json:"icon,omitempty"`
	Theme       string          `json:"theme"`
	Features    map[string]bool `json:"features"`
	OSAModel    string          `json:"osa_model,omitempty"`
	Custom      map[string]any  `json:"custom,omitempty"`
}

// AgentConfig is an agent definition (no conversation history).
type AgentConfig struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	SystemPrompt string         `json:"system_prompt"`
	Model        string         `json:"model"`
	Temperature  float64        `json:"temperature"`
	Tools        []string       `json:"tools"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

// AppDefinition is a custom app schema/layout (no data rows).
type AppDefinition struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Type        string         `json:"type"`
	Schema      map[string]any `json:"schema"`
	Layout      map[string]any `json:"layout"`
	Permissions map[string]any `json:"permissions,omitempty"`
}

// TemplateDefinition is a template structure (no documents created from it).
type TemplateDefinition struct {
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	Category string         `json:"category"`
	Body     string         `json:"body"`
	Vars     []string       `json:"vars,omitempty"`
	Meta     map[string]any `json:"meta,omitempty"`
}

// SyncResult is returned by POST /api/miosa/sync.
type SyncResult struct {
	Success    bool      `json:"success"`
	SyncedAt   time.Time `json:"synced_at"`
	ManifestID string    `json:"manifest_id,omitempty"`
	Error      string    `json:"error,omitempty"`
}

// ManifestProvider is implemented by the workspace service to build a manifest
// for a given workspace. This keeps the sync service free of direct database
// dependencies.
type ManifestProvider interface {
	BuildManifest(ctx context.Context, workspaceID string) (*WorkspaceManifest, error)
}

// SyncService pushes a WorkspaceManifest to MIOSA Cloud.
// It is only active when OSA_MODE=cloud and MIOSA_API_KEY is set.
type SyncService struct {
	mode     SyncMode
	apiKey   string
	cloudURL string
	provider ManifestProvider
	logger   *slog.Logger
	client   *http.Client
}

// NewSyncService constructs the service. If mode is ModeLocal or apiKey is
// empty, all cloud operations return a no-op success so callers need not
// branch on mode.
func NewSyncService(
	mode SyncMode,
	apiKey string,
	cloudURL string,
	provider ManifestProvider,
	logger *slog.Logger,
) *SyncService {
	if cloudURL == "" {
		cloudURL = "https://api.miosa.ai"
	}
	return &SyncService{
		mode:     mode,
		apiKey:   apiKey,
		cloudURL: cloudURL,
		provider: provider,
		logger:   logger,
		client:   &http.Client{Timeout: 30 * time.Second},
	}
}

// Status returns the current cloud connection status without performing any
// network calls (safe to call frequently).
func (s *SyncService) Status(ctx context.Context) ConnectionStatus {
	return ConnectionStatus{
		Mode:      s.mode,
		Connected: s.mode == ModeCloud && s.apiKey != "",
		APIKeySet: s.apiKey != "",
	}
}

// Ping verifies the API key is valid by calling the MIOSA Cloud health
// endpoint. Use this when the user first enters their API key.
func (s *SyncService) Ping(ctx context.Context) error {
	if s.mode != ModeCloud || s.apiKey == "" {
		return fmt.Errorf("cloud mode is not configured (OSA_MODE must be 'cloud' and MIOSA_API_KEY must be set)")
	}

	var health map[string]interface{}
	if err := s.doCloudJSON(ctx, http.MethodGet, "/health", nil, &health); err != nil {
		return fmt.Errorf("MIOSA Cloud ping failed: %w", err)
	}
	return nil
}

// Sync builds the workspace manifest and pushes it to MIOSA Cloud.
// It is safe to call in local mode; the function returns a no-op success.
func (s *SyncService) Sync(ctx context.Context, workspaceID string) (*SyncResult, error) {
	if s.mode != ModeCloud || s.apiKey == "" {
		s.logger.DebugContext(ctx, "sync skipped: not in cloud mode")
		return &SyncResult{Success: true, SyncedAt: time.Now()}, nil
	}

	manifest, err := s.provider.BuildManifest(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to build workspace manifest: %w", err)
	}

	payload, err := json.Marshal(manifest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal manifest: %w", err)
	}

	var resp struct {
		SessionID string `json:"session_id"`
		ID        string `json:"id"`
	}
	err = s.doCloudJSON(ctx, http.MethodPost, "/api/v1/orchestrate", map[string]interface{}{
		"input":        string(payload),
		"user_id":      "bos-sync",
		"workspace_id": workspaceID,
		"phase":        "manifest-sync",
	}, &resp)
	if err != nil {
		return &SyncResult{
			Success: false,
			Error:   err.Error(),
		}, fmt.Errorf("MIOSA Cloud sync failed: %w", err)
	}

	manifestID := resp.SessionID
	if manifestID == "" {
		manifestID = resp.ID
	}

	s.logger.InfoContext(ctx, "workspace manifest synced to MIOSA Cloud",
		slog.String("workspace_id", workspaceID),
		slog.String("manifest_id", manifestID),
	)

	return &SyncResult{
		Success:    true,
		SyncedAt:   time.Now(),
		ManifestID: manifestID,
	}, nil
}

func (s *SyncService) doCloudJSON(ctx context.Context, method, path string, payload interface{}, out interface{}) error {
	var body io.Reader
	if payload != nil {
		var buf bytes.Buffer
		if err := json.NewEncoder(&buf).Encode(payload); err != nil {
			return err
		}
		body = &buf
	}

	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(s.cloudURL, "/")+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		var errBody struct {
			Error   string `json:"error"`
			Details string `json:"details"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		msg := errBody.Details
		if msg == "" {
			msg = errBody.Error
		}
		if msg == "" {
			msg = resp.Status
		}
		return fmt.Errorf("status %d: %s", resp.StatusCode, msg)
	}

	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
