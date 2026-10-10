// Package instagram provides the Instagram/Meta integration used by ContentOS.
package instagram

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/config"
	"github.com/rhl/businessos-backend/internal/integrations"
	"github.com/rhl/businessos-backend/internal/security"
)

const (
	ProviderID    = "instagram"
	ProviderName  = "Instagram"
	Category      = "social"
	AuthURL       = "https://www.facebook.com/v20.0/dialog/oauth"
	TokenURL      = "https://graph.facebook.com/v20.0/oauth/access_token"
	GraphAPIURL   = "https://graph.facebook.com/v20.0"
	oauthStateTTL = 10 * time.Minute
)

var (
	ErrWorkspaceAccess       = errors.New("active workspace membership required")
	ErrWorkspaceManagement   = errors.New("workspace owner or admin required")
	ErrEncryptionUnavailable = errors.New("Instagram credential encryption is not configured")
	ErrInvalidOAuthState     = errors.New("invalid or expired Instagram OAuth state")
)

var DefaultScopes = []string{
	"instagram_basic",
	"instagram_manage_insights",
	"instagram_content_publish",
	"pages_show_list",
	"pages_read_engagement",
}

type Provider struct {
	pool         *pgxpool.Pool
	encryption   *security.TokenEncryption
	clientID     string
	clientSecret string
	redirectURI  string
	scopes       []string
}

type InstagramAccount struct {
	ID                string `json:"id"`
	Username          string `json:"username"`
	Name              string `json:"name,omitempty"`
	ProfilePictureURL string `json:"profile_picture_url,omitempty"`
	FollowersCount    int    `json:"followers_count,omitempty"`
	MediaCount        int    `json:"media_count,omitempty"`
	PageID            string `json:"page_id,omitempty"`
	PageName          string `json:"page_name,omitempty"`
}

type MediaItem struct {
	ID               string `json:"id"`
	Caption          string `json:"caption,omitempty"`
	MediaType        string `json:"media_type,omitempty"`
	MediaProductType string `json:"media_product_type,omitempty"`
	Permalink        string `json:"permalink,omitempty"`
	Timestamp        string `json:"timestamp,omitempty"`
	LikeCount        int    `json:"like_count,omitempty"`
	CommentsCount    int    `json:"comments_count,omitempty"`
}

type AuditPreview struct {
	Account            *InstagramAccount `json:"account,omitempty"`
	RecentMedia        []MediaItem       `json:"recent_media"`
	Reels              []MediaItem       `json:"reels"`
	MediaCountChecked  int               `json:"media_count_checked"`
	ReelsCountChecked  int               `json:"reels_count_checked"`
	InsightsReady      bool              `json:"insights_ready"`
	PublishingReady    bool              `json:"publishing_ready"`
	NextCursor         string            `json:"next_cursor,omitempty"`
	ImplementationNote string            `json:"implementation_note"`
}

func NewProvider(pool *pgxpool.Pool) *Provider {
	return newProvider(pool, security.GetGlobalEncryption())
}

func newProvider(pool *pgxpool.Pool, encryption *security.TokenEncryption) *Provider {
	cfg := config.AppConfig

	redirectURI := cfg.InstagramRedirectURI
	if redirectURI == "" {
		redirectURI = cfg.MetaRedirectURI
	}
	if redirectURI == "" {
		redirectURI = fmt.Sprintf("%s/api/integrations/instagram/callback", cfg.BaseURL)
	}

	clientID := cfg.InstagramClientID
	if clientID == "" {
		clientID = cfg.MetaAppID
	}
	clientSecret := cfg.InstagramClientSecret
	if clientSecret == "" {
		clientSecret = cfg.MetaAppSecret
	}

	return &Provider{
		pool:         pool,
		encryption:   encryption,
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURI:  redirectURI,
		scopes:       DefaultScopes,
	}
}

func (p *Provider) Name() string        { return ProviderID }
func (p *Provider) DisplayName() string { return ProviderName }
func (p *Provider) Category() string    { return Category }
func (p *Provider) Icon() string        { return "/logos/integrations/instagram.svg" }
func (p *Provider) Configured() bool {
	return p.clientID != "" && p.clientSecret != "" && p.redirectURI != "" && p.encryption != nil
}

func (p *Provider) GetAuthURL(state string) string {
	params := url.Values{}
	params.Set("client_id", p.clientID)
	params.Set("redirect_uri", p.redirectURI)
	params.Set("state", state)
	params.Set("response_type", "code")
	params.Set("scope", strings.Join(p.scopes, ","))
	return fmt.Sprintf("%s?%s", AuthURL, params.Encode())
}

func (p *Provider) ExchangeCode(ctx context.Context, code string) (*integrations.TokenResponse, error) {
	params := url.Values{}
	params.Set("client_id", p.clientID)
	params.Set("client_secret", p.clientSecret)
	params.Set("redirect_uri", p.redirectURI)
	params.Set("code", code)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, TokenURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange Instagram code: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("Instagram token exchange failed: %s", string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse Instagram token response: %w", err)
	}

	expiresAt := time.Now().Add(60 * 24 * time.Hour)
	if tokenResp.ExpiresIn > 0 {
		expiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	}

	account, err := p.GetPrimaryAccount(ctx, tokenResp.AccessToken)
	if err != nil {
		account = &InstagramAccount{ID: "unknown", Username: "Instagram account"}
	}

	return &integrations.TokenResponse{
		AccessToken: tokenResp.AccessToken,
		ExpiresAt:   expiresAt,
		Scopes:      p.scopes,
		AccountID:   account.ID,
		AccountName: account.Username,
		Metadata: map[string]interface{}{
			"username":            account.Username,
			"name":                account.Name,
			"profile_picture_url": account.ProfilePictureURL,
			"followers_count":     account.FollowersCount,
			"media_count":         account.MediaCount,
			"page_id":             account.PageID,
			"page_name":           account.PageName,
		},
	}, nil
}

func (p *Provider) RefreshToken(ctx context.Context, refreshToken string) (*integrations.TokenResponse, error) {
	return nil, fmt.Errorf("Instagram token refresh is not implemented yet")
}

type encryptedTokenData struct {
	AccessToken string `json:"access_token"`
}

func (p *Provider) RequireWorkspaceMembership(ctx context.Context, userID, workspaceID string) error {
	if userID == "" || workspaceID == "" {
		return ErrWorkspaceAccess
	}
	var active bool
	if err := p.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM workspace_members
			WHERE workspace_id = $1 AND user_id = $2 AND status = 'active'
		)
	`, workspaceID, userID).Scan(&active); err != nil {
		return fmt.Errorf("failed to verify Instagram workspace access: %w", err)
	}
	if !active {
		return ErrWorkspaceAccess
	}
	return nil
}

func (p *Provider) RequireWorkspaceManagement(ctx context.Context, userID, workspaceID string) error {
	if userID == "" || workspaceID == "" {
		return ErrWorkspaceManagement
	}
	var allowed bool
	if err := p.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM workspace_members
			WHERE workspace_id = $1 AND user_id = $2 AND status = 'active'
			  AND LOWER(COALESCE(NULLIF(role_name, ''), NULLIF(role, ''), '')) IN ('owner', 'admin')
		)
	`, workspaceID, userID).Scan(&allowed); err != nil {
		return fmt.Errorf("failed to verify Instagram workspace management access: %w", err)
	}
	if !allowed {
		return ErrWorkspaceManagement
	}
	return nil
}

func (p *Provider) CreateOAuthState(ctx context.Context, userID, workspaceID string) (string, error) {
	if err := p.RequireWorkspaceManagement(ctx, userID, workspaceID); err != nil {
		return "", err
	}
	random := make([]byte, 32)
	if _, err := rand.Read(random); err != nil {
		return "", fmt.Errorf("failed to create Instagram OAuth state: %w", err)
	}
	state := base64.RawURLEncoding.EncodeToString(random)
	hash := sha256.Sum256([]byte(state))
	_, err := p.pool.Exec(ctx, `
		INSERT INTO instagram_oauth_states (state_hash, user_id, workspace_id, expires_at)
		VALUES ($1::bytea, $2::varchar, $3::uuid, $4::timestamptz)
	`, hash[:], userID, workspaceID, time.Now().Add(oauthStateTTL))
	if err != nil {
		return "", fmt.Errorf("failed to store Instagram OAuth state: %w", err)
	}
	_, _ = p.pool.Exec(ctx, `DELETE FROM instagram_oauth_states WHERE expires_at < NOW()`)
	return state, nil
}

func (p *Provider) ConsumeOAuthState(ctx context.Context, state string) (string, string, error) {
	if state == "" {
		return "", "", ErrInvalidOAuthState
	}
	hash := sha256.Sum256([]byte(state))
	var userID, workspaceID string
	err := p.pool.QueryRow(ctx, `
		DELETE FROM instagram_oauth_states
		WHERE state_hash = $1 AND expires_at >= NOW()
		RETURNING user_id, workspace_id
	`, hash[:]).Scan(&userID, &workspaceID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", ErrInvalidOAuthState
	}
	if err != nil {
		return "", "", fmt.Errorf("failed to consume Instagram OAuth state: %w", err)
	}
	if err := p.RequireWorkspaceMembership(ctx, userID, workspaceID); err != nil {
		return "", "", err
	}
	return userID, workspaceID, nil
}

func (p *Provider) SaveToken(ctx context.Context, userID, workspaceID string, token *integrations.TokenResponse) error {
	if p.encryption == nil {
		return ErrEncryptionUnavailable
	}
	plaintext, err := json.Marshal(encryptedTokenData{AccessToken: token.AccessToken})
	if err != nil {
		return fmt.Errorf("failed to serialize Instagram credentials: %w", err)
	}
	encryptedData, err := p.encryption.EncryptBytes(string(plaintext))
	if err != nil {
		return fmt.Errorf("failed to encrypt Instagram credentials: %w", err)
	}

	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var membership int
	if err := tx.QueryRow(ctx, `
		SELECT 1 FROM workspace_members
		WHERE workspace_id = $1 AND user_id = $2 AND status = 'active'
		  AND LOWER(COALESCE(NULLIF(role_name, ''), NULLIF(role, ''), '')) IN ('owner', 'admin')
		FOR UPDATE
	`, workspaceID, userID).Scan(&membership); errors.Is(err, pgx.ErrNoRows) {
		return ErrWorkspaceManagement
	} else if err != nil {
		return fmt.Errorf("failed to verify Instagram workspace access: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO instagram_workspace_credentials (
			workspace_id, connected_by_user_id, encrypted_data, encryption_version,
			expires_at, external_account_id, external_account_name, scopes, metadata
		) VALUES ($1, $2, $3, 1, $4, $5, $6, $7, $8)
		ON CONFLICT (workspace_id) DO UPDATE SET
			connected_by_user_id = EXCLUDED.connected_by_user_id,
			encrypted_data = EXCLUDED.encrypted_data,
			encryption_version = EXCLUDED.encryption_version,
			expires_at = EXCLUDED.expires_at,
			external_account_id = EXCLUDED.external_account_id,
			external_account_name = EXCLUDED.external_account_name,
			scopes = EXCLUDED.scopes,
			metadata = EXCLUDED.metadata,
			connected_at = NOW(),
			updated_at = NOW()
	`, workspaceID, userID, encryptedData, token.ExpiresAt, token.AccountID, token.AccountName, token.Scopes, token.Metadata)
	if err != nil {
		return fmt.Errorf("failed to save Instagram credentials: %w", err)
	}

	return tx.Commit(ctx)
}

func (p *Provider) GetToken(ctx context.Context, userID, workspaceID string) (*integrations.Token, error) {
	if p.encryption == nil {
		return nil, ErrEncryptionUnavailable
	}
	var token integrations.Token
	var encryptedData []byte
	err := p.pool.QueryRow(ctx, `
		SELECT iwc.encrypted_data, iwc.expires_at, iwc.scopes
		FROM instagram_workspace_credentials iwc
		JOIN workspace_members wm ON wm.workspace_id = iwc.workspace_id
		WHERE iwc.workspace_id = $1 AND wm.user_id = $2 AND wm.status = 'active'
	`, workspaceID, userID).Scan(&encryptedData, &token.ExpiresAt, &token.Scopes)
	if errors.Is(err, pgx.ErrNoRows) {
		if membershipErr := p.RequireWorkspaceMembership(ctx, userID, workspaceID); membershipErr != nil {
			return nil, membershipErr
		}
		return nil, fmt.Errorf("Instagram token not found: %w", pgx.ErrNoRows)
	}
	if err != nil {
		return nil, fmt.Errorf("Instagram token not found: %w", err)
	}
	plaintext, err := p.encryption.DecryptBytes(encryptedData)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt Instagram credentials: %w", err)
	}
	var tokenData encryptedTokenData
	if err := json.Unmarshal([]byte(plaintext), &tokenData); err != nil {
		return nil, fmt.Errorf("failed to parse Instagram credentials: %w", err)
	}
	token.AccessToken = tokenData.AccessToken
	return &token, nil
}

func (p *Provider) GetConnectionStatus(ctx context.Context, userID, workspaceID string) (*integrations.ConnectionStatus, error) {
	var status integrations.ConnectionStatus
	err := p.pool.QueryRow(ctx, `
			SELECT
				true, iwc.connected_at, iwc.external_account_id,
				iwc.external_account_name, iwc.scopes, iwc.updated_at
			FROM instagram_workspace_credentials iwc
			JOIN workspace_members wm ON wm.workspace_id = iwc.workspace_id
			WHERE iwc.workspace_id = $1 AND wm.user_id = $2 AND wm.status = 'active'
		`, workspaceID, userID).Scan(
		&status.Connected,
		&status.ConnectedAt,
		&status.AccountID,
		&status.AccountName,
		&status.Scopes,
		&status.LastSyncAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		if membershipErr := p.RequireWorkspaceMembership(ctx, userID, workspaceID); membershipErr != nil {
			return nil, membershipErr
		}
		return &integrations.ConnectionStatus{
			Connected:  false,
			SyncStatus: "idle",
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get Instagram connection status: %w", err)
	}
	status.SyncStatus = "idle"
	return &status, nil
}

func (p *Provider) Disconnect(ctx context.Context, userID, workspaceID string) error {
	tag, err := p.pool.Exec(ctx, `
		DELETE FROM instagram_workspace_credentials iwc
		USING workspace_members wm
		WHERE iwc.workspace_id = $1
			AND wm.workspace_id = iwc.workspace_id
			AND wm.user_id = $2
			AND wm.status = 'active'
			AND LOWER(COALESCE(NULLIF(wm.role_name, ''), NULLIF(wm.role, ''), '')) IN ('owner', 'admin')
	`, workspaceID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete Instagram credentials: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return p.RequireWorkspaceManagement(ctx, userID, workspaceID)
	}
	return nil
}

func (p *Provider) SupportsSync() bool { return true }

func (p *Provider) Sync(ctx context.Context, userID string, options integrations.SyncOptions) (*integrations.SyncResult, error) {
	return nil, errors.New("Instagram sync requires an explicit workspace")
}

func (p *Provider) SyncWorkspace(ctx context.Context, userID, workspaceID string, options integrations.SyncOptions) (*integrations.SyncResult, error) {
	start := time.Now()
	_, err := p.BuildAuditPreview(ctx, userID, workspaceID, 50)
	result := &integrations.SyncResult{
		Success:  err == nil,
		Duration: time.Since(start),
	}
	if err != nil {
		result.Errors = append(result.Errors, err.Error())
	}
	return result, err
}

func (p *Provider) BuildAuditPreview(ctx context.Context, userID, workspaceID string, limit int) (*AuditPreview, error) {
	token, err := p.GetToken(ctx, userID, workspaceID)
	if err != nil {
		return nil, err
	}
	account, err := p.GetPrimaryAccount(ctx, token.AccessToken)
	if err != nil {
		return nil, err
	}
	media, nextCursor, err := p.GetRecentMedia(ctx, token.AccessToken, account.ID, limit)
	if err != nil {
		return nil, err
	}

	reels := make([]MediaItem, 0)
	for _, item := range media {
		if strings.EqualFold(item.MediaProductType, "REELS") || strings.Contains(strings.ToUpper(item.Permalink), "/REEL/") {
			reels = append(reels, item)
		}
	}

	return &AuditPreview{
		Account:            account,
		RecentMedia:        media,
		Reels:              reels,
		MediaCountChecked:  len(media),
		ReelsCountChecked:  len(reels),
		InsightsReady:      true,
		PublishingReady:    true,
		NextCursor:         nextCursor,
		ImplementationNote: "This preview reads the connected Instagram professional account and recent media. A full audit will paginate all media, pull per-post insights, transcribe videos, and send the findings into ContentOS.",
	}, nil
}

func (p *Provider) GetPrimaryAccount(ctx context.Context, accessToken string) (*InstagramAccount, error) {
	endpoint := GraphAPIURL + "/me/accounts"
	params := url.Values{}
	params.Set("access_token", accessToken)
	params.Set("fields", "id,name,instagram_business_account{id,username,name,profile_picture_url,followers_count,media_count}")

	var payload struct {
		Data []struct {
			ID                       string `json:"id"`
			Name                     string `json:"name"`
			InstagramBusinessAccount struct {
				ID                string `json:"id"`
				Username          string `json:"username"`
				Name              string `json:"name"`
				ProfilePictureURL string `json:"profile_picture_url"`
				FollowersCount    int    `json:"followers_count"`
				MediaCount        int    `json:"media_count"`
			} `json:"instagram_business_account"`
		} `json:"data"`
	}
	if err := p.getJSON(ctx, endpoint, params, &payload); err != nil {
		return nil, err
	}
	for _, page := range payload.Data {
		ig := page.InstagramBusinessAccount
		if ig.ID == "" {
			continue
		}
		return &InstagramAccount{
			ID:                ig.ID,
			Username:          ig.Username,
			Name:              ig.Name,
			ProfilePictureURL: ig.ProfilePictureURL,
			FollowersCount:    ig.FollowersCount,
			MediaCount:        ig.MediaCount,
			PageID:            page.ID,
			PageName:          page.Name,
		}, nil
	}
	return nil, fmt.Errorf("no Instagram professional account found for connected Meta user")
}

func (p *Provider) GetRecentMedia(ctx context.Context, accessToken, accountID string, limit int) ([]MediaItem, string, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	endpoint := fmt.Sprintf("%s/%s/media", GraphAPIURL, accountID)
	params := url.Values{}
	params.Set("access_token", accessToken)
	params.Set("limit", fmt.Sprintf("%d", limit))
	params.Set("fields", "id,caption,media_type,media_product_type,permalink,timestamp,like_count,comments_count")

	var payload struct {
		Data   []MediaItem `json:"data"`
		Paging struct {
			Cursors struct {
				After string `json:"after"`
			} `json:"cursors"`
		} `json:"paging"`
	}
	if err := p.getJSON(ctx, endpoint, params, &payload); err != nil {
		return nil, "", err
	}
	return payload.Data, payload.Paging.Cursors.After, nil
}

func (p *Provider) getJSON(ctx context.Context, endpoint string, params url.Values, target interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+params.Encode(), nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("Instagram API error %d: %s", resp.StatusCode, string(body))
	}
	return json.Unmarshal(body, target)
}

func (p *Provider) Pool() *pgxpool.Pool {
	return p.pool
}
