package microsoft

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

// ============================================================================
// MICROSOFT TEAMS — channels and messages
// ============================================================================
//
// Mirror of OutlookService for the Teams data path. Wave 2 supports:
//   - List joined teams + channels per team
//   - Sync channels into microsoft_teams_channels
//   - List/sync/send messages within a channel (microsoft_teams_messages)
//
// Out of scope for Wave 2 (deferred to Wave 3):
//   - Threaded replies (Graph: /messages/:id/replies)
//   - Reactions (Graph: /messages/:id/reactions toggle)
//   - DMs / private chats (Graph: /chats — different surface from team channels)
//   - Mention rendering (need to resolve aadObjectId → display)
//   - Realtime via Graph subscriptions (Wave 3)
//
// Engine sync: every saved message fires OnMessageSaved with the canonical
// DB row. See handlers/comms_engine_sync.go::newTeamsMessageEngineHook.

// ----------------------------------------------------------------------------
// Types
// ----------------------------------------------------------------------------

// TeamsChannel is the BO row representation returned by GetChannels.
type TeamsChannel struct {
	ID              string     `json:"id"`         // BO uuid
	UserID          string     `json:"user_id"`
	TeamID          string     `json:"team_id"`    // Graph team id
	ChannelID       string     `json:"channel_id"` // Graph channel id
	TeamName        string     `json:"team_name"`
	DisplayName     string     `json:"display_name"`
	Description     string     `json:"description,omitempty"`
	Email           string     `json:"email,omitempty"`
	WebURL          string     `json:"web_url,omitempty"`
	MembershipType  string     `json:"membership_type"` // standard|private|shared
	IsArchived      bool       `json:"is_archived"`
	MemberCount     int        `json:"member_count"`
	LastMessageAt   *time.Time `json:"last_message_at,omitempty"`
	CreatedDateTime *time.Time `json:"created_datetime,omitempty"`
	SyncedAt        time.Time  `json:"synced_at"`
}

// TeamsMessage is the BO row for a Teams chatMessage.
type TeamsMessage struct {
	ID                   string     `json:"id"` // BO uuid
	UserID               string     `json:"user_id"`
	MessageID            string     `json:"message_id"` // Graph chatMessage id
	TeamID               string     `json:"team_id"`
	ChannelID            string     `json:"channel_id"`
	ReplyToID            string     `json:"reply_to_id,omitempty"`
	MessageType          string     `json:"message_type,omitempty"`
	Importance           string     `json:"importance,omitempty"`
	Subject              string     `json:"subject,omitempty"`
	Summary              string     `json:"summary,omitempty"`
	BodyContent          string     `json:"body_content,omitempty"`
	BodyContentType      string     `json:"body_content_type,omitempty"`
	SenderID             string     `json:"sender_id,omitempty"`
	SenderName           string     `json:"sender_name,omitempty"`
	SenderType           string     `json:"sender_type,omitempty"`
	WebURL               string     `json:"web_url,omitempty"`
	IsDeleted            bool       `json:"is_deleted"`
	IsEdited             bool       `json:"is_edited"`
	CreatedDateTime      *time.Time `json:"created_datetime,omitempty"`
	LastModifiedDateTime *time.Time `json:"last_modified_datetime,omitempty"`
	SyncedAt             time.Time  `json:"synced_at"`
}

// SyncTeamsResult summarizes a channels-or-messages sync run.
type SyncTeamsResult struct {
	TotalTeams       int `json:"total_teams,omitempty"`
	SyncedChannels   int `json:"synced_channels,omitempty"`
	FailedChannels   int `json:"failed_channels,omitempty"`
	TotalMessages    int `json:"total_messages,omitempty"`
	SyncedMessages   int `json:"synced_messages,omitempty"`
	FailedMessages   int `json:"failed_messages,omitempty"`
}

// MessageSavedHook fires after a Teams message is upserted. Mirror of
// EmailSavedHook in shape; the engine-sync wiring lives in
// handlers/comms_engine_sync.go::newTeamsMessageEngineHook.
type MessageSavedHook func(ctx context.Context, msg *TeamsMessage, userID string)

// ----------------------------------------------------------------------------
// Service
// ----------------------------------------------------------------------------

// TeamsService handles Microsoft Teams operations (channels and messages).
type TeamsService struct {
	provider *Provider

	// OnMessageSaved is fired after a Teams message is upserted into
	// microsoft_teams_messages. nil = engine sync not wired.
	OnMessageSaved MessageSavedHook
}

// NewTeamsService constructs a TeamsService.
func NewTeamsService(provider *Provider) *TeamsService {
	return &TeamsService{provider: provider}
}

// ----------------------------------------------------------------------------
// Graph API DTOs (only the fields we read)
// ----------------------------------------------------------------------------

type graphTeam struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
}

type graphChannel struct {
	ID              string `json:"id"`
	DisplayName     string `json:"displayName"`
	Description     string `json:"description"`
	Email           string `json:"email"`
	WebURL          string `json:"webUrl"`
	MembershipType  string `json:"membershipType"`
	CreatedDateTime string `json:"createdDateTime"`
}

type graphChatMessage struct {
	ID                   string `json:"id"`
	ReplyToID            string `json:"replyToId"`
	MessageType          string `json:"messageType"`
	Importance           string `json:"importance"`
	Subject              string `json:"subject"`
	Summary              string `json:"summary"`
	WebURL               string `json:"webUrl"`
	CreatedDateTime      string `json:"createdDateTime"`
	LastModifiedDateTime string `json:"lastModifiedDateTime"`
	LastEditedDateTime   string `json:"lastEditedDateTime"`
	DeletedDateTime      string `json:"deletedDateTime"`
	Body                 *struct {
		ContentType string `json:"contentType"`
		Content     string `json:"content"`
	} `json:"body"`
	From *struct {
		User *struct {
			ID                string `json:"id"`
			DisplayName       string `json:"displayName"`
			UserIdentityType  string `json:"userIdentityType"`
		} `json:"user"`
	} `json:"from"`
	Attachments []json.RawMessage `json:"attachments"`
	Mentions    []json.RawMessage `json:"mentions"`
	Reactions   []json.RawMessage `json:"reactions"`
}

// ----------------------------------------------------------------------------
// Sync — channels
// ----------------------------------------------------------------------------

// SyncChannels lists every team the user belongs to, then every channel in
// each team, upserting into microsoft_teams_channels. Returns counts so the
// caller can surface a "synced N channels across M teams" toast.
func (s *TeamsService) SyncChannels(ctx context.Context, userID string) (*SyncTeamsResult, error) {
	client, err := s.provider.GetHTTPClient(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get HTTP client: %w", err)
	}

	// 1) List joined teams.
	teamsResp, err := client.Get(GraphAPIBase + "/me/joinedTeams?$select=id,displayName")
	if err != nil {
		return nil, fmt.Errorf("failed to list teams: %w", err)
	}
	defer teamsResp.Body.Close()
	if teamsResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list teams: %s", teamsResp.Status)
	}
	var teamsBody struct {
		Value []graphTeam `json:"value"`
	}
	if err := json.NewDecoder(teamsResp.Body).Decode(&teamsBody); err != nil {
		return nil, fmt.Errorf("decode teams: %w", err)
	}

	result := &SyncTeamsResult{TotalTeams: len(teamsBody.Value)}

	// 2) For each team, list channels and upsert each.
	for _, t := range teamsBody.Value {
		channelsURL := fmt.Sprintf("%s/teams/%s/channels?$select=id,displayName,description,email,webUrl,membershipType,createdDateTime",
			GraphAPIBase, url.PathEscape(t.ID))
		chResp, err := client.Get(channelsURL)
		if err != nil {
			slog.Info("teams: channels fetch failed", "team_id", t.ID, "error", err)
			result.FailedChannels++
			continue
		}
		var chBody struct {
			Value []graphChannel `json:"value"`
		}
		decErr := json.NewDecoder(chResp.Body).Decode(&chBody)
		chResp.Body.Close()
		if decErr != nil {
			slog.Info("teams: channels decode failed", "team_id", t.ID, "error", decErr)
			result.FailedChannels++
			continue
		}
		for _, c := range chBody.Value {
			if err := s.saveChannel(ctx, userID, t, &c); err != nil {
				slog.Info("teams: save channel failed", "team_id", t.ID, "channel_id", c.ID, "error", err)
				result.FailedChannels++
			} else {
				result.SyncedChannels++
			}
		}
	}

	return result, nil
}

func (s *TeamsService) saveChannel(ctx context.Context, userID string, team graphTeam, c *graphChannel) error {
	var createdAt *time.Time
	if c.CreatedDateTime != "" {
		if t, perr := time.Parse(time.RFC3339, c.CreatedDateTime); perr == nil {
			createdAt = &t
		}
	}

	_, err := s.provider.Pool().Exec(ctx, `
		INSERT INTO microsoft_teams_channels (
			user_id, team_id, channel_id, team_name,
			display_name, description, email, web_url, membership_type,
			created_datetime, synced_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
		ON CONFLICT (user_id, team_id, channel_id) DO UPDATE SET
			team_name = EXCLUDED.team_name,
			display_name = EXCLUDED.display_name,
			description = EXCLUDED.description,
			email = EXCLUDED.email,
			web_url = EXCLUDED.web_url,
			membership_type = EXCLUDED.membership_type,
			synced_at = NOW(),
			updated_at = NOW()
	`, userID, team.ID, c.ID, team.DisplayName,
		c.DisplayName, c.Description, c.Email, c.WebURL, c.MembershipType,
		createdAt)
	return err
}

// ----------------------------------------------------------------------------
// Read — channels
// ----------------------------------------------------------------------------

// GetChannels returns every Teams channel for a user, ordered by recent
// activity. The order matches Slack's sidebar convention.
func (s *TeamsService) GetChannels(ctx context.Context, userID string) ([]*TeamsChannel, error) {
	rows, err := s.provider.Pool().Query(ctx, `
		SELECT id, user_id, team_id, channel_id, team_name,
			display_name, description, email, web_url, membership_type,
			is_archived, member_count, last_message_at, created_datetime, synced_at
		FROM microsoft_teams_channels
		WHERE user_id = $1
		ORDER BY last_message_at DESC NULLS LAST, display_name
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*TeamsChannel
	for rows.Next() {
		var (
			ch              TeamsChannel
			teamName        *string
			displayName     *string
			description     *string
			email           *string
			webURL          *string
			membershipType  *string
			lastMessageAt   *time.Time
			createdDateTime *time.Time
		)
		if err := rows.Scan(
			&ch.ID, &ch.UserID, &ch.TeamID, &ch.ChannelID, &teamName,
			&displayName, &description, &email, &webURL, &membershipType,
			&ch.IsArchived, &ch.MemberCount, &lastMessageAt, &createdDateTime, &ch.SyncedAt,
		); err != nil {
			return nil, err
		}
		ch.TeamName = derefMS(teamName)
		ch.DisplayName = derefMS(displayName)
		ch.Description = derefMS(description)
		ch.Email = derefMS(email)
		ch.WebURL = derefMS(webURL)
		ch.MembershipType = derefMS(membershipType)
		ch.LastMessageAt = lastMessageAt
		ch.CreatedDateTime = createdDateTime
		out = append(out, &ch)
	}
	return out, rows.Err()
}

// GetChannelByID resolves a BO uuid to a TeamsChannel row. Used by the
// handler to translate the URL :id into the Graph (team_id, channel_id)
// pair before calling the Graph API.
func (s *TeamsService) GetChannelByID(ctx context.Context, userID, id string) (*TeamsChannel, error) {
	var (
		ch              TeamsChannel
		teamName        *string
		displayName     *string
		description     *string
		email           *string
		webURL          *string
		membershipType  *string
		lastMessageAt   *time.Time
		createdDateTime *time.Time
	)
	err := s.provider.Pool().QueryRow(ctx, `
		SELECT id, user_id, team_id, channel_id, team_name,
			display_name, description, email, web_url, membership_type,
			is_archived, member_count, last_message_at, created_datetime, synced_at
		FROM microsoft_teams_channels
		WHERE user_id = $1 AND id = $2
	`, userID, id).Scan(
		&ch.ID, &ch.UserID, &ch.TeamID, &ch.ChannelID, &teamName,
		&displayName, &description, &email, &webURL, &membershipType,
		&ch.IsArchived, &ch.MemberCount, &lastMessageAt, &createdDateTime, &ch.SyncedAt,
	)
	if err != nil {
		return nil, err
	}
	ch.TeamName = derefMS(teamName)
	ch.DisplayName = derefMS(displayName)
	ch.Description = derefMS(description)
	ch.Email = derefMS(email)
	ch.WebURL = derefMS(webURL)
	ch.MembershipType = derefMS(membershipType)
	ch.LastMessageAt = lastMessageAt
	ch.CreatedDateTime = createdDateTime
	return &ch, nil
}

// ----------------------------------------------------------------------------
// Sync — messages
// ----------------------------------------------------------------------------

// SyncChannelMessages fetches recent messages in a channel and upserts
// them into microsoft_teams_messages. Each saved row fires OnMessageSaved
// so the engine sees it.
func (s *TeamsService) SyncChannelMessages(ctx context.Context, userID, teamID, channelID string, maxResults int) (*SyncTeamsResult, error) {
	if maxResults <= 0 {
		maxResults = 50
	}
	client, err := s.provider.GetHTTPClient(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get HTTP client: %w", err)
	}

	apiURL := fmt.Sprintf("%s/teams/%s/channels/%s/messages?$top=%d",
		GraphAPIBase, url.PathEscape(teamID), url.PathEscape(channelID), maxResults)

	resp, err := client.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list messages: %s", resp.Status)
	}

	var body struct {
		Value []graphChatMessage `json:"value"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("decode messages: %w", err)
	}

	result := &SyncTeamsResult{TotalMessages: len(body.Value)}
	var latest time.Time
	for _, m := range body.Value {
		if err := s.saveMessage(ctx, userID, teamID, channelID, &m); err != nil {
			slog.Info("teams: save message failed", "id", m.ID, "error", err)
			result.FailedMessages++
			continue
		}
		result.SyncedMessages++
		if t := parseGraphTime(m.CreatedDateTime); t != nil && t.After(latest) {
			latest = *t
		}
	}

	// Best-effort: bump last_message_at on the channel row so the sidebar
	// orders correctly without a follow-up call.
	if !latest.IsZero() {
		_, _ = s.provider.Pool().Exec(ctx, `
			UPDATE microsoft_teams_channels
			SET last_message_at = $1, updated_at = NOW()
			WHERE user_id = $2 AND team_id = $3 AND channel_id = $4
			  AND (last_message_at IS NULL OR last_message_at < $1)
		`, latest, userID, teamID, channelID)
	}

	return result, nil
}

func (s *TeamsService) saveMessage(ctx context.Context, userID, teamID, channelID string, m *graphChatMessage) error {
	var bodyContent, bodyContentType string
	if m.Body != nil {
		bodyContent = m.Body.Content
		bodyContentType = m.Body.ContentType
	}

	var senderID, senderName, senderType string
	if m.From != nil && m.From.User != nil {
		senderID = m.From.User.ID
		senderName = m.From.User.DisplayName
		senderType = m.From.User.UserIdentityType
	}

	createdAt := parseGraphTime(m.CreatedDateTime)
	lastModified := parseGraphTime(m.LastModifiedDateTime)
	lastEdited := parseGraphTime(m.LastEditedDateTime)
	deletedAt := parseGraphTime(m.DeletedDateTime)

	attachmentsJSON, _ := json.Marshal(m.Attachments)
	mentionsJSON, _ := json.Marshal(m.Mentions)
	reactionsJSON, _ := json.Marshal(m.Reactions)

	// Defaults so JSONB columns never receive NULL when the array was empty.
	if len(attachmentsJSON) == 0 {
		attachmentsJSON = []byte("[]")
	}
	if len(mentionsJSON) == 0 {
		mentionsJSON = []byte("[]")
	}
	if len(reactionsJSON) == 0 {
		reactionsJSON = []byte("[]")
	}

	_, err := s.provider.Pool().Exec(ctx, `
		INSERT INTO microsoft_teams_messages (
			user_id, message_id, team_id, channel_id, reply_to_id,
			message_type, importance, subject, summary,
			body_content, body_content_type,
			sender_id, sender_name, sender_type,
			attachments, mentions, reactions,
			web_url, is_deleted, is_edited,
			created_datetime, last_modified_datetime, last_edited_datetime, deleted_datetime,
			synced_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, NOW())
		ON CONFLICT (user_id, team_id, channel_id, message_id) DO UPDATE SET
			reply_to_id = EXCLUDED.reply_to_id,
			message_type = EXCLUDED.message_type,
			importance = EXCLUDED.importance,
			subject = EXCLUDED.subject,
			summary = EXCLUDED.summary,
			body_content = EXCLUDED.body_content,
			body_content_type = EXCLUDED.body_content_type,
			sender_id = EXCLUDED.sender_id,
			sender_name = EXCLUDED.sender_name,
			sender_type = EXCLUDED.sender_type,
			attachments = EXCLUDED.attachments,
			mentions = EXCLUDED.mentions,
			reactions = EXCLUDED.reactions,
			web_url = EXCLUDED.web_url,
			is_deleted = EXCLUDED.is_deleted,
			is_edited = EXCLUDED.is_edited,
			last_modified_datetime = EXCLUDED.last_modified_datetime,
			last_edited_datetime = EXCLUDED.last_edited_datetime,
			deleted_datetime = EXCLUDED.deleted_datetime,
			synced_at = NOW(),
			updated_at = NOW()
	`, userID, m.ID, teamID, channelID, m.ReplyToID,
		m.MessageType, m.Importance, m.Subject, m.Summary,
		bodyContent, bodyContentType,
		senderID, senderName, senderType,
		attachmentsJSON, mentionsJSON, reactionsJSON,
		m.WebURL, deletedAt != nil, lastEdited != nil,
		createdAt, lastModified, lastEdited, deletedAt)
	if err != nil {
		return err
	}

	// Engine-sync hook fires after upsert; refetch by canonical key for
	// the BO uuid. Mirrors google.GmailService.saveEmail and
	// microsoft.OutlookService.saveMessage.
	if s.OnMessageSaved != nil {
		if saved, ferr := s.getMessageByMessageID(ctx, userID, teamID, channelID, m.ID); ferr == nil && saved != nil {
			s.OnMessageSaved(ctx, saved, userID)
		}
	}

	return nil
}

func (s *TeamsService) getMessageByMessageID(ctx context.Context, userID, teamID, channelID, messageID string) (*TeamsMessage, error) {
	var (
		msg                  TeamsMessage
		replyToID            *string
		messageType          *string
		importance           *string
		subject              *string
		summary              *string
		bodyContent          *string
		bodyContentType      *string
		senderID             *string
		senderName           *string
		senderType           *string
		webURL               *string
		createdDateTime      *time.Time
		lastModifiedDateTime *time.Time
	)
	err := s.provider.Pool().QueryRow(ctx, `
		SELECT id, user_id, message_id, team_id, channel_id, reply_to_id,
			message_type, importance, subject, summary,
			body_content, body_content_type,
			sender_id, sender_name, sender_type,
			web_url, is_deleted, is_edited,
			created_datetime, last_modified_datetime, synced_at
		FROM microsoft_teams_messages
		WHERE user_id = $1 AND team_id = $2 AND channel_id = $3 AND message_id = $4
	`, userID, teamID, channelID, messageID).Scan(
		&msg.ID, &msg.UserID, &msg.MessageID, &msg.TeamID, &msg.ChannelID, &replyToID,
		&messageType, &importance, &subject, &summary,
		&bodyContent, &bodyContentType,
		&senderID, &senderName, &senderType,
		&webURL, &msg.IsDeleted, &msg.IsEdited,
		&createdDateTime, &lastModifiedDateTime, &msg.SyncedAt,
	)
	if err != nil {
		return nil, err
	}
	msg.ReplyToID = derefMS(replyToID)
	msg.MessageType = derefMS(messageType)
	msg.Importance = derefMS(importance)
	msg.Subject = derefMS(subject)
	msg.Summary = derefMS(summary)
	msg.BodyContent = derefMS(bodyContent)
	msg.BodyContentType = derefMS(bodyContentType)
	msg.SenderID = derefMS(senderID)
	msg.SenderName = derefMS(senderName)
	msg.SenderType = derefMS(senderType)
	msg.WebURL = derefMS(webURL)
	msg.CreatedDateTime = createdDateTime
	msg.LastModifiedDateTime = lastModifiedDateTime
	return &msg, nil
}

// FetchAndSaveChannelMessage fetches a single Teams channel message
// from Graph by its identifiers, upserts it into
// microsoft_teams_messages, and fires OnMessageSaved. Used by the Graph
// webhook handler — every notification carries only resourceData.id, so
// we GET the resource ourselves before persisting.
func (s *TeamsService) FetchAndSaveChannelMessage(ctx context.Context, userID, teamID, channelID, messageID string) error {
	client, err := s.provider.GetHTTPClient(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get HTTP client: %w", err)
	}

	apiURL := fmt.Sprintf("%s/teams/%s/channels/%s/messages/%s",
		GraphAPIBase, url.PathEscape(teamID), url.PathEscape(channelID), url.PathEscape(messageID))

	resp, err := client.Get(apiURL)
	if err != nil {
		return fmt.Errorf("graph fetch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("graph fetch: %s", resp.Status)
	}

	var msg graphChatMessage
	if err := json.NewDecoder(resp.Body).Decode(&msg); err != nil {
		return fmt.Errorf("decode message: %w", err)
	}
	return s.saveMessage(ctx, userID, teamID, channelID, &msg)
}

// ----------------------------------------------------------------------------
// Read — messages
// ----------------------------------------------------------------------------

// GetChannelMessages returns the most recent messages for a channel, by
// created_datetime descending. limit defaults to 50.
func (s *TeamsService) GetChannelMessages(ctx context.Context, userID, teamID, channelID string, limit, offset int) ([]*TeamsMessage, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.provider.Pool().Query(ctx, `
		SELECT id, user_id, message_id, team_id, channel_id, reply_to_id,
			message_type, importance, subject, summary,
			body_content, body_content_type,
			sender_id, sender_name, sender_type,
			web_url, is_deleted, is_edited,
			created_datetime, last_modified_datetime, synced_at
		FROM microsoft_teams_messages
		WHERE user_id = $1 AND team_id = $2 AND channel_id = $3
		ORDER BY created_datetime DESC NULLS LAST
		LIMIT $4 OFFSET $5
	`, userID, teamID, channelID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*TeamsMessage
	for rows.Next() {
		var (
			msg                  TeamsMessage
			replyToID            *string
			messageType          *string
			importance           *string
			subject              *string
			summary              *string
			bodyContent          *string
			bodyContentType      *string
			senderID             *string
			senderName           *string
			senderType           *string
			webURL               *string
			createdDateTime      *time.Time
			lastModifiedDateTime *time.Time
		)
		if err := rows.Scan(
			&msg.ID, &msg.UserID, &msg.MessageID, &msg.TeamID, &msg.ChannelID, &replyToID,
			&messageType, &importance, &subject, &summary,
			&bodyContent, &bodyContentType,
			&senderID, &senderName, &senderType,
			&webURL, &msg.IsDeleted, &msg.IsEdited,
			&createdDateTime, &lastModifiedDateTime, &msg.SyncedAt,
		); err != nil {
			return nil, err
		}
		msg.ReplyToID = derefMS(replyToID)
		msg.MessageType = derefMS(messageType)
		msg.Importance = derefMS(importance)
		msg.Subject = derefMS(subject)
		msg.Summary = derefMS(summary)
		msg.BodyContent = derefMS(bodyContent)
		msg.BodyContentType = derefMS(bodyContentType)
		msg.SenderID = derefMS(senderID)
		msg.SenderName = derefMS(senderName)
		msg.SenderType = derefMS(senderType)
		msg.WebURL = derefMS(webURL)
		msg.CreatedDateTime = createdDateTime
		msg.LastModifiedDateTime = lastModifiedDateTime
		out = append(out, &msg)
	}
	return out, rows.Err()
}

// ----------------------------------------------------------------------------
// Send
// ----------------------------------------------------------------------------

// SendChannelMessage posts a message to a Teams channel. Body is plain
// text; Graph wraps it as bodyContentType=text. Returns the Graph
// chatMessage id of the posted message.
func (s *TeamsService) SendChannelMessage(ctx context.Context, userID, teamID, channelID, content string, isHTML bool) (string, error) {
	client, err := s.provider.GetHTTPClient(ctx, userID)
	if err != nil {
		return "", err
	}

	contentType := "text"
	if isHTML {
		contentType = "html"
	}
	payload := map[string]interface{}{
		"body": map[string]string{
			"contentType": contentType,
			"content":     content,
		},
	}
	jsonBody, _ := json.Marshal(payload)

	apiURL := fmt.Sprintf("%s/teams/%s/channels/%s/messages",
		GraphAPIBase, url.PathEscape(teamID), url.PathEscape(channelID))

	resp, err := client.Post(apiURL, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("send message: %s", resp.Status)
	}

	var posted graphChatMessage
	_ = json.NewDecoder(resp.Body).Decode(&posted)
	return posted.ID, nil
}

// ----------------------------------------------------------------------------
// helpers
// ----------------------------------------------------------------------------

func derefMS(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func parseGraphTime(s string) *time.Time {
	if s == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil
	}
	return &t
}
