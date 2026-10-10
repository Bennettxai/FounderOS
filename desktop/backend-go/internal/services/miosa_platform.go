package services

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	miosasdk "github.com/Miosa-osa/miosa-go"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/config"
)

const ProviderMIOSAPlatform = "miosa_platform"

type MIOSAPlatformService struct {
	pool *pgxpool.Pool
	cfg  *config.Config
}

type MIOSAPlatformKey struct {
	APIKey string
	Prefix string
	Source string
}

type MIOSAWorkspaceLink struct {
	WorkspaceID         string     `json:"workspace_id"`
	MIOSAWorkspaceID    string     `json:"miosa_workspace_id"`
	ExternalWorkspaceID string     `json:"external_workspace_id"`
	ExternalUserID      string     `json:"external_user_id"`
	Status              string     `json:"status"`
	LastSyncedAt        *time.Time `json:"last_synced_at,omitempty"`
}

type MIOSAWorkspaceEntitlement struct {
	WorkspaceID     string     `json:"workspace_id"`
	SandboxEnabled  bool       `json:"sandbox_enabled"`
	ComputerEnabled bool       `json:"computer_enabled"`
	DesktopEnabled  bool       `json:"desktop_enabled"`
	EnabledByUserID string     `json:"enabled_by_user_id,omitempty"`
	EnabledAt       *time.Time `json:"enabled_at,omitempty"`
}

type CreateMIOSASandboxInput struct {
	WorkspaceID string
	UserID      string
	UserEmail   string
	Name        string
	Cols        int
	Rows        int
	Shell       string
}

type MIOSASandboxTerminalSession struct {
	SandboxID           string `json:"sandbox_id"`
	MIOSAWorkspaceID    string `json:"miosa_workspace_id,omitempty"`
	ExternalWorkspaceID string `json:"external_workspace_id"`
	ExternalUserID      string `json:"external_user_id"`
	Status              string `json:"status"`
	PreviewURL          string `json:"preview_url,omitempty"`
	TerminalSessionID   string `json:"terminal_session_id,omitempty"`
	StreamAuth          string `json:"-"`
	ExpiresAt           int64  `json:"expires_at,omitempty"`
	WebSocketURL        string `json:"ws_url,omitempty"`
}

func NewMIOSAPlatformService(pool *pgxpool.Pool, cfg *config.Config) *MIOSAPlatformService {
	return &MIOSAPlatformService{pool: pool, cfg: cfg}
}

func (s *MIOSAPlatformService) APIBaseURL() string {
	baseURL := "https://api.miosa.ai/api/v1"
	if s.cfg != nil && s.cfg.MIOSACloudURL != "" {
		baseURL = s.cfg.MIOSACloudURL
	} else if s.cfg != nil && s.cfg.MIOSAAPIUrl != "" {
		baseURL = s.cfg.MIOSAAPIUrl
	}
	baseURL = strings.TrimRight(baseURL, "/")
	if !strings.HasSuffix(baseURL, "/api/v1") {
		baseURL += "/api/v1"
	}
	return baseURL
}

func (s *MIOSAPlatformService) NewClient(apiKey string) *miosasdk.Client {
	return miosasdk.NewClient(apiKey, miosasdk.WithBaseURL(s.APIBaseURL()))
}

func (s *MIOSAPlatformService) GetTenantKey(ctx context.Context) (*MIOSAPlatformKey, error) {
	if s.pool != nil {
		var userID string
		err := s.pool.QueryRow(ctx, `
			SELECT user_id
			FROM credential_vault
			WHERE provider_id = $1
			ORDER BY updated_at DESC
			LIMIT 1
		`, ProviderMIOSAPlatform).Scan(&userID)
		if err == nil {
			keyData, err := NewCredentialVaultService(s.pool).GetAPIKey(ctx, userID, ProviderMIOSAPlatform)
			if err == nil && keyData != nil && keyData.APIKey != "" {
				return &MIOSAPlatformKey{APIKey: keyData.APIKey, Prefix: KeyPrefix(keyData.APIKey), Source: "vault"}, nil
			}
			if err != nil {
				return nil, err
			}
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
	}

	if s.cfg != nil && s.cfg.MIOSAAPIKey != "" {
		return &MIOSAPlatformKey{APIKey: s.cfg.MIOSAAPIKey, Prefix: KeyPrefix(s.cfg.MIOSAAPIKey), Source: "env"}, nil
	}

	return nil, nil
}

func (s *MIOSAPlatformService) SaveTenantKey(ctx context.Context, userID, apiKey string) (*miosasdk.TenantPlan, string, error) {
	apiKey = strings.TrimSpace(apiKey)
	if !strings.HasPrefix(apiKey, "msk_") {
		return nil, "", fmt.Errorf("MIOSA API keys must start with msk_")
	}

	client := s.NewClient(apiKey)
	tenant, err := client.Tenant.Current(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("MIOSA tenant validation failed: %w", err)
	}

	if s.pool == nil {
		return tenant, KeyPrefix(apiKey), nil
	}

	_, err = NewCredentialVaultService(s.pool).StoreAPIKeyCredential(ctx, StoreAPIKeyInput{
		UserID:               userID,
		ProviderID:           ProviderMIOSAPlatform,
		APIKey:               apiKey,
		ExternalAccountID:    tenant.ID,
		ExternalAccountEmail: tenant.Name,
		Metadata: map[string]interface{}{
			"role":              "businessos_miosa_tenant",
			"capacity_provider": "businessos",
			"tenant_id":         tenant.ID,
			"tenant_name":       tenant.Name,
			"tenant_plan":       tenant.Plan,
		},
	})
	if err != nil {
		return nil, "", fmt.Errorf("failed to store MIOSA key: %w", err)
	}

	return tenant, KeyPrefix(apiKey), nil
}

func (s *MIOSAPlatformService) EnsureWorkspaceLink(ctx context.Context, workspaceID, createdByUserID string) (*MIOSAWorkspaceLink, error) {
	if s.pool == nil {
		return nil, fmt.Errorf("database is not configured")
	}

	if link, err := s.GetWorkspaceLink(ctx, workspaceID); err == nil && link != nil {
		return link, nil
	} else if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	key, err := s.GetTenantKey(ctx)
	if err != nil {
		return nil, err
	}
	if key == nil || key.APIKey == "" {
		return nil, fmt.Errorf("BusinessOS MIOSA tenant key is not configured")
	}

	var name, slug string
	err = s.pool.QueryRow(ctx, `
		SELECT name, slug
		FROM workspaces
		WHERE id = $1
	`, workspaceID).Scan(&name, &slug)
	if err != nil {
		return nil, err
	}

	externalWorkspaceID := workspaceID
	externalUserID := workspaceID
	remote, err := s.NewClient(key.APIKey).Workspaces.Create(ctx, miosasdk.CreateWorkspaceInput{
		Name:        name,
		Description: "BusinessOS workspace managed by the BusinessOS MIOSA tenant",
		Metadata: map[string]string{
			"businessos_workspace_id":   workspaceID,
			"businessos_workspace_slug": slug,
			"external_workspace_id":     externalWorkspaceID,
			"external_user_id":          externalUserID,
			"managed_by":                "businessos",
		},
	})
	if err != nil {
		return nil, err
	}

	return s.UpsertWorkspaceLink(ctx, MIOSAWorkspaceLink{
		WorkspaceID:         workspaceID,
		MIOSAWorkspaceID:    remote.ID,
		ExternalWorkspaceID: externalWorkspaceID,
		ExternalUserID:      externalUserID,
		Status:              "linked",
	}, createdByUserID)
}

func (s *MIOSAPlatformService) CreateSandboxTerminalSession(ctx context.Context, input CreateMIOSASandboxInput) (*MIOSASandboxTerminalSession, error) {
	enabled, err := s.IsWorkspaceSandboxEnabled(ctx, input.WorkspaceID)
	if err != nil {
		return nil, err
	}
	if !enabled {
		return nil, fmt.Errorf("BusinessOS MIOSA sandboxes are not enabled for this workspace")
	}

	key, err := s.GetTenantKey(ctx)
	if err != nil {
		return nil, err
	}
	if key == nil || key.APIKey == "" {
		return nil, fmt.Errorf("BusinessOS MIOSA tenant key is not configured")
	}

	link, err := s.EnsureWorkspaceLink(ctx, input.WorkspaceID, input.UserID)
	if err != nil {
		return nil, err
	}

	client := s.NewClient(key.APIKey)
	name := strings.TrimSpace(input.Name)
	if name == "" {
		name = "BusinessOS sandbox"
	}
	sandbox, err := client.Sandboxes.Create(ctx, miosasdk.CreateSandboxInput{
		Name: name,
		Size: miosasdk.SizeSmall,
		Metadata: map[string]string{
			"managed_by":                  "businessos",
			"businessos_workspace_id":     input.WorkspaceID,
			"businessos_user_id":          input.UserID,
			"businessos_user_email":       input.UserEmail,
			"businessos_miosa_workspace":  link.MIOSAWorkspaceID,
			"businessos_external_subject": link.ExternalUserID,
		},
		ExternalWorkspaceID: link.ExternalWorkspaceID,
		ExternalUserID:      link.ExternalUserID,
		ExternalProjectID:   input.WorkspaceID,
	})
	if err != nil {
		return nil, err
	}

	comp := sandbox.ComputerData
	compStatus := string(comp.Status)
	if compStatus == "creating" || compStatus == "starting" || compStatus == "" {
		for i := 0; i < 20; i++ {
			time.Sleep(3 * time.Second)
			updated, pollErr := client.Sandboxes.Get(ctx, comp.ID)
			if pollErr != nil {
				continue
			}
			comp = updated.ComputerData
			compStatus = string(comp.Status)
			if compStatus == "running" || compStatus == "active" {
				break
			}
		}
	}

	session := &MIOSASandboxTerminalSession{
		SandboxID:           comp.ID,
		MIOSAWorkspaceID:    link.MIOSAWorkspaceID,
		ExternalWorkspaceID: link.ExternalWorkspaceID,
		ExternalUserID:      link.ExternalUserID,
		Status:              compStatus,
		PreviewURL:          publicSandboxURL(comp),
	}

	handle := client.Sandboxes.GetHandle(comp.ID)
	cols := input.Cols
	if cols <= 0 {
		cols = 120
	}
	rows := input.Rows
	if rows <= 0 {
		rows = 30
	}
	shell := strings.TrimSpace(input.Shell)
	if shell == "" {
		shell = "bash"
	}

	var raw map[string]interface{}
	var termErr error
	for attempt := 0; attempt < 3; attempt++ {
		raw, termErr = handle.Terminal.Create(ctx, miosasdk.SandboxCreateTerminalInput{
			Cols:  cols,
			Rows:  rows,
			Shell: shell,
			Env: map[string]string{
				"BUSINESSOS_TERMINAL_ENV":          "sandbox",
				"BUSINESSOS_MIOSA_ACCOUNT_SOURCE":  "businessos",
				"BUSINESSOS_MIOSA_SANDBOX_ID":      comp.ID,
				"BUSINESSOS_MIOSA_WORKSPACE_ID":    link.MIOSAWorkspaceID,
				"BUSINESSOS_EXTERNAL_WORKSPACE_ID": link.ExternalWorkspaceID,
				"BUSINESSOS_EXTERNAL_USER_ID":      link.ExternalUserID,
			},
		})
		if termErr == nil {
			break
		}
		time.Sleep(3 * time.Second)
	}
	if termErr != nil {
		return nil, termErr
	}

	session.TerminalSessionID = stringValue(raw, "session_id", "id")
	session.StreamAuth = stringValue(raw, "stream_auth", "auth", "token")
	session.ExpiresAt = int64Value(raw, "expires_at")
	session.WebSocketURL = sandboxTerminalWebSocketURL(comp, session.TerminalSessionID, session.StreamAuth)

	if s.pool != nil {
		_, _ = s.pool.Exec(ctx, `
			INSERT INTO miosa_sandbox_sessions (
				workspace_id,
				user_id,
				miosa_sandbox_id,
				miosa_workspace_id,
				external_workspace_id,
				external_user_id,
				terminal_session_id,
				status,
				preview_url
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`, input.WorkspaceID, input.UserID, session.SandboxID, session.MIOSAWorkspaceID, session.ExternalWorkspaceID, session.ExternalUserID, session.TerminalSessionID, session.Status, session.PreviewURL)
	}

	return session, nil
}

func (s *MIOSAPlatformService) IsWorkspaceSandboxEnabled(ctx context.Context, workspaceID string) (bool, error) {
	if s.pool == nil || workspaceID == "" {
		return false, nil
	}
	var enabled bool
	err := s.pool.QueryRow(ctx, `
		SELECT sandbox_enabled
		FROM miosa_workspace_entitlements
		WHERE workspace_id = $1
	`, workspaceID).Scan(&enabled)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return enabled, nil
}

func (s *MIOSAPlatformService) GetWorkspaceEntitlement(ctx context.Context, workspaceID string) (*MIOSAWorkspaceEntitlement, error) {
	if s.pool == nil || workspaceID == "" {
		return nil, pgx.ErrNoRows
	}
	var entitlement MIOSAWorkspaceEntitlement
	err := s.pool.QueryRow(ctx, `
		SELECT
			workspace_id::text,
			sandbox_enabled,
			computer_enabled,
			desktop_enabled,
			COALESCE(enabled_by_user_id, ''),
			enabled_at
		FROM miosa_workspace_entitlements
		WHERE workspace_id = $1
	`, workspaceID).Scan(
		&entitlement.WorkspaceID,
		&entitlement.SandboxEnabled,
		&entitlement.ComputerEnabled,
		&entitlement.DesktopEnabled,
		&entitlement.EnabledByUserID,
		&entitlement.EnabledAt,
	)
	if err != nil {
		return nil, err
	}
	return &entitlement, nil
}

type UpdateMIOSAWorkspaceEntitlementInput struct {
	SandboxEnabled  *bool
	ComputerEnabled *bool
	DesktopEnabled  *bool
}

func (s *MIOSAPlatformService) SetWorkspaceSandboxEnabled(ctx context.Context, workspaceID, actorUserID string, enabled bool) (*MIOSAWorkspaceEntitlement, error) {
	return s.UpdateWorkspaceEntitlement(ctx, workspaceID, actorUserID, UpdateMIOSAWorkspaceEntitlementInput{
		SandboxEnabled: &enabled,
	})
}

func (s *MIOSAPlatformService) UpdateWorkspaceEntitlement(ctx context.Context, workspaceID, actorUserID string, input UpdateMIOSAWorkspaceEntitlementInput) (*MIOSAWorkspaceEntitlement, error) {
	if s.pool == nil {
		return nil, fmt.Errorf("database is not configured")
	}
	var entitlement MIOSAWorkspaceEntitlement
	err := s.pool.QueryRow(ctx, `
		INSERT INTO miosa_workspace_entitlements (
			workspace_id,
			sandbox_enabled,
			computer_enabled,
			desktop_enabled,
			enabled_by_user_id,
			enabled_at,
			updated_at
		)
		VALUES (
			$1,
			COALESCE($2, FALSE),
			COALESCE($3, FALSE),
			COALESCE($4, FALSE),
			$5,
			CASE WHEN COALESCE($2, FALSE) OR COALESCE($3, FALSE) OR COALESCE($4, FALSE) THEN NOW() ELSE NULL END,
			NOW()
		)
		ON CONFLICT (workspace_id) DO UPDATE SET
			sandbox_enabled = COALESCE($2, miosa_workspace_entitlements.sandbox_enabled),
			computer_enabled = COALESCE($3, miosa_workspace_entitlements.computer_enabled),
			desktop_enabled = COALESCE($4, miosa_workspace_entitlements.desktop_enabled),
			enabled_by_user_id = EXCLUDED.enabled_by_user_id,
			enabled_at = CASE
				WHEN COALESCE($2, miosa_workspace_entitlements.sandbox_enabled)
					OR COALESCE($3, miosa_workspace_entitlements.computer_enabled)
					OR COALESCE($4, miosa_workspace_entitlements.desktop_enabled)
				THEN COALESCE(miosa_workspace_entitlements.enabled_at, NOW())
				ELSE NULL
			END,
			updated_at = NOW()
		RETURNING
			workspace_id::text,
			sandbox_enabled,
			computer_enabled,
			desktop_enabled,
			COALESCE(enabled_by_user_id, ''),
			enabled_at
	`, workspaceID, input.SandboxEnabled, input.ComputerEnabled, input.DesktopEnabled, actorUserID).Scan(
		&entitlement.WorkspaceID,
		&entitlement.SandboxEnabled,
		&entitlement.ComputerEnabled,
		&entitlement.DesktopEnabled,
		&entitlement.EnabledByUserID,
		&entitlement.EnabledAt,
	)
	if err != nil {
		return nil, err
	}
	return &entitlement, nil
}

func (s *MIOSAPlatformService) GetWorkspaceLink(ctx context.Context, workspaceID string) (*MIOSAWorkspaceLink, error) {
	var link MIOSAWorkspaceLink
	err := s.pool.QueryRow(ctx, `
		SELECT
			workspace_id::text,
			miosa_workspace_id,
			external_workspace_id,
			external_user_id,
			status,
			last_synced_at
		FROM miosa_workspace_links
		WHERE workspace_id = $1
	`, workspaceID).Scan(
		&link.WorkspaceID,
		&link.MIOSAWorkspaceID,
		&link.ExternalWorkspaceID,
		&link.ExternalUserID,
		&link.Status,
		&link.LastSyncedAt,
	)
	if err != nil {
		return nil, err
	}
	return &link, nil
}

func (s *MIOSAPlatformService) UpsertWorkspaceLink(ctx context.Context, link MIOSAWorkspaceLink, createdByUserID string) (*MIOSAWorkspaceLink, error) {
	var saved MIOSAWorkspaceLink
	err := s.pool.QueryRow(ctx, `
		INSERT INTO miosa_workspace_links (
			workspace_id,
			miosa_workspace_id,
			external_workspace_id,
			external_user_id,
			status,
			created_by_user_id,
			last_synced_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (workspace_id) DO UPDATE SET
			miosa_workspace_id = EXCLUDED.miosa_workspace_id,
			external_workspace_id = EXCLUDED.external_workspace_id,
			external_user_id = EXCLUDED.external_user_id,
			status = EXCLUDED.status,
			last_synced_at = NOW(),
			updated_at = NOW()
		RETURNING
			workspace_id::text,
			miosa_workspace_id,
			external_workspace_id,
			external_user_id,
			status,
			last_synced_at
	`, link.WorkspaceID, link.MIOSAWorkspaceID, link.ExternalWorkspaceID, link.ExternalUserID, link.Status, createdByUserID).Scan(
		&saved.WorkspaceID,
		&saved.MIOSAWorkspaceID,
		&saved.ExternalWorkspaceID,
		&saved.ExternalUserID,
		&saved.Status,
		&saved.LastSyncedAt,
	)
	if err != nil {
		return nil, err
	}
	return &saved, nil
}

func KeyPrefix(apiKey string) string {
	if len(apiKey) <= 12 {
		return apiKey
	}
	return apiKey[:12]
}

func publicSandboxURL(comp miosasdk.ComputerData) string {
	if comp.PreviewDomain != "" {
		if strings.HasPrefix(comp.PreviewDomain, "http://") || strings.HasPrefix(comp.PreviewDomain, "https://") {
			return comp.PreviewDomain
		}
		return "https://" + comp.PreviewDomain
	}
	if comp.Slug != "" {
		return "https://" + comp.Slug + ".sandbox.miosa.ai"
	}
	return ""
}

func sandboxTerminalWebSocketURL(comp miosasdk.ComputerData, sessionID, streamAuth string) string {
	if sessionID == "" {
		return ""
	}
	host := comp.Slug + ".sandbox.miosa.ai"
	if comp.PreviewDomain != "" {
		parsed, err := url.Parse(publicSandboxURL(comp))
		if err == nil && parsed.Host != "" {
			host = parsed.Host
		}
	}
	if comp.Slug == "" && host == ".sandbox.miosa.ai" && len(comp.ID) >= 8 {
		host = comp.ID[:8] + ".sandbox.miosa.ai"
	}

	wsURL := "wss://" + host + "/ws/terminal/" + comp.ID + "/" + sessionID
	if streamAuth != "" {
		wsURL += "?auth=" + url.QueryEscape(streamAuth)
	}
	return wsURL
}

func stringValue(raw map[string]interface{}, keys ...string) string {
	candidates := []map[string]interface{}{raw}
	for _, key := range []string{"session", "data", "terminal_session"} {
		if nested, ok := raw[key].(map[string]interface{}); ok {
			candidates = append(candidates, nested)
		}
	}
	for _, candidate := range candidates {
		for _, key := range keys {
			if value, ok := candidate[key].(string); ok && value != "" {
				return value
			}
		}
	}
	return ""
}

func int64Value(raw map[string]interface{}, keys ...string) int64 {
	candidates := []map[string]interface{}{raw}
	for _, key := range []string{"session", "data", "terminal_session"} {
		if nested, ok := raw[key].(map[string]interface{}); ok {
			candidates = append(candidates, nested)
		}
	}
	for _, candidate := range candidates {
		for _, key := range keys {
			switch value := candidate[key].(type) {
			case int64:
				return value
			case int:
				return int64(value)
			case float64:
				return int64(value)
			case string:
				var parsed int64
				if _, err := fmt.Sscan(value, &parsed); err == nil {
					return parsed
				}
			}
		}
	}
	return 0
}
