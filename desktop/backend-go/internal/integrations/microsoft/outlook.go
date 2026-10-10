package microsoft

import (
	"context"
	"time"
)

// ============================================================================
// OUTLOOK MAIL TYPES
// ============================================================================

// OutlookMessage represents an Outlook email message.
type OutlookMessage struct {
	ID               string           `json:"id"`
	UserID           string           `json:"user_id"`
	MessageID        string           `json:"message_id"`
	ConversationID   string           `json:"conversation_id,omitempty"`
	Subject          string           `json:"subject"`
	BodyPreview      string           `json:"body_preview,omitempty"`
	BodyContent      string           `json:"body_content,omitempty"`
	BodyContentType  string           `json:"body_content_type,omitempty"`
	Importance       string           `json:"importance,omitempty"`
	FromEmail        string           `json:"from_email"`
	FromName         string           `json:"from_name"`
	ToRecipients     []EmailRecipient `json:"to_recipients,omitempty"`
	CcRecipients     []EmailRecipient `json:"cc_recipients,omitempty"`
	BccRecipients    []EmailRecipient `json:"bcc_recipients,omitempty"`
	IsRead           bool             `json:"is_read"`
	IsDraft          bool             `json:"is_draft"`
	HasAttachments   bool             `json:"has_attachments"`
	FolderID         string           `json:"folder_id,omitempty"`
	FolderName       string           `json:"folder_name,omitempty"`
	Categories       []string         `json:"categories,omitempty"`
	FlagStatus       string           `json:"flag_status,omitempty"`
	ReceivedDateTime time.Time        `json:"received_datetime,omitempty"`
	SentDateTime     time.Time        `json:"sent_datetime,omitempty"`
	SyncedAt         time.Time        `json:"synced_at"`
}

// EmailRecipient represents an email recipient.
type EmailRecipient struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

// EmailSavedHook is called after an Outlook email is upserted into the
// `microsoft_mail_messages` table. Used by handlers to mirror the email
// into the OptimalEngine knowledge graph as Signal{Module: "email",
// metadata.provider: "outlook"}. nil = no-op.
//
// Mirrors google.EmailSavedHook in shape; the engine-sync wiring lives
// in handlers/comms_engine_sync.go (newOutlookEngineHook).
type EmailSavedHook func(ctx context.Context, email *OutlookMessage, userID string)

// ============================================================================
// OUTLOOK CALENDAR TYPES
// ============================================================================

// OutlookEvent represents an Outlook calendar event.
type OutlookEvent struct {
	ID                    string          `json:"id"`
	UserID                string          `json:"user_id"`
	EventID               string          `json:"event_id"`
	CalendarID            string          `json:"calendar_id,omitempty"`
	Subject               string          `json:"subject"`
	BodyPreview           string          `json:"body_preview,omitempty"`
	BodyContent           string          `json:"body_content,omitempty"`
	LocationDisplayName   string          `json:"location_display_name,omitempty"`
	StartDateTime         time.Time       `json:"start_datetime"`
	StartTimeZone         string          `json:"start_timezone,omitempty"`
	EndDateTime           time.Time       `json:"end_datetime"`
	EndTimeZone           string          `json:"end_timezone,omitempty"`
	IsAllDay              bool            `json:"is_all_day"`
	Attendees             []EventAttendee `json:"attendees,omitempty"`
	OrganizerEmail        string          `json:"organizer_email,omitempty"`
	OrganizerName         string          `json:"organizer_name,omitempty"`
	IsOnlineMeeting       bool            `json:"is_online_meeting"`
	OnlineMeetingProvider string          `json:"online_meeting_provider,omitempty"`
	OnlineMeetingURL      string          `json:"online_meeting_url,omitempty"`
	ResponseStatus        string          `json:"response_status,omitempty"`
	Importance            string          `json:"importance,omitempty"`
	ShowAs                string          `json:"show_as,omitempty"`
	IsCancelled           bool            `json:"is_cancelled"`
	IsReminderOn          bool            `json:"is_reminder_on"`
	ReminderMinutesBefore int             `json:"reminder_minutes_before"`
	Categories            []string        `json:"categories,omitempty"`
	CreatedDateTime       time.Time       `json:"created_datetime,omitempty"`
	LastModifiedDateTime  time.Time       `json:"last_modified_datetime,omitempty"`
	SyncedAt              time.Time       `json:"synced_at"`
}

// EventAttendee represents a calendar event attendee.
type EventAttendee struct {
	Email          string `json:"email"`
	Name           string `json:"name,omitempty"`
	Type           string `json:"type,omitempty"` // required, optional, resource
	ResponseStatus string `json:"response_status,omitempty"`
}

// ============================================================================
// OUTLOOK SERVICE
// ============================================================================

// OutlookService handles Microsoft Outlook operations (mail and calendar).
type OutlookService struct {
	provider *Provider

	// OnEmailSaved is fired after an email is upserted into the
	// `microsoft_mail_messages` table. Used by handlers to push the row
	// into the OptimalEngine knowledge graph as a Signal{Module: "email"}.
	// Set externally (typically in routes_integrations.go) — nil when
	// the engine isn't wired, in which case sync continues to work
	// without engine indexing.
	OnEmailSaved EmailSavedHook
}

// NewOutlookService creates a new Outlook service.
func NewOutlookService(provider *Provider) *OutlookService {
	return &OutlookService{provider: provider}
}
