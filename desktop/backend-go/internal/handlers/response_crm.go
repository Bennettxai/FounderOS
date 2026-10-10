package handlers

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/rhl/businessos-backend/internal/database/sqlc"
)

// Client response transformation
type ClientResponse struct {
	ID              string                 `json:"id"`
	UserID          string                 `json:"user_id"`
	Name            string                 `json:"name"`
	Type            string                 `json:"type"`
	Email           *string                `json:"email"`
	Phone           *string                `json:"phone"`
	Website         *string                `json:"website"`
	Industry        *string                `json:"industry"`
	CompanySize     *string                `json:"company_size"`
	Address         *string                `json:"address"`
	City            *string                `json:"city"`
	State           *string                `json:"state"`
	ZipCode         *string                `json:"zip_code"`
	Country         *string                `json:"country"`
	Status          string                 `json:"status"`
	Source          *string                `json:"source"`
	AssignedTo      *string                `json:"assigned_to"`
	LifetimeValue   *float64               `json:"lifetime_value"`
	Tags            []string               `json:"tags"`
	CustomFields    map[string]interface{} `json:"custom_fields"`
	Notes           *string                `json:"notes"`
	CreatedAt       string                 `json:"created_at"`
	UpdatedAt       string                 `json:"updated_at"`
	LastContactedAt *string                `json:"last_contacted_at"`
}

func TransformClient(c sqlc.Client) ClientResponse {
	clientType := "company"
	if c.Type.Valid {
		clientType = string(c.Type.Clienttype)
	}

	status := "lead"
	if c.Status.Valid {
		status = string(c.Status.Clientstatus)
	}

	var tags []string
	if c.Tags != nil {
		json.Unmarshal(c.Tags, &tags)
	}
	if tags == nil {
		tags = []string{}
	}

	var customFields map[string]interface{}
	if c.CustomFields != nil {
		json.Unmarshal(c.CustomFields, &customFields)
	}

	return ClientResponse{
		ID:              pgtypeUUIDToStringRequired(c.ID),
		UserID:          c.UserID,
		Name:            c.Name,
		Type:            clientType,
		Email:           c.Email,
		Phone:           c.Phone,
		Website:         c.Website,
		Industry:        c.Industry,
		CompanySize:     c.CompanySize,
		Address:         c.Address,
		City:            c.City,
		State:           c.State,
		ZipCode:         c.ZipCode,
		Country:         c.Country,
		Status:          status,
		Source:          c.Source,
		AssignedTo:      c.AssignedTo,
		LifetimeValue:   pgtypeNumericToFloat(c.LifetimeValue),
		Tags:            tags,
		CustomFields:    customFields,
		Notes:           c.Notes,
		CreatedAt:       c.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:       c.UpdatedAt.Time.Format(time.RFC3339),
		LastContactedAt: pgtypeTimestamptzToString(c.LastContactedAt),
	}
}

func TransformClients(clients []sqlc.Client) []ClientResponse {
	result := make([]ClientResponse, len(clients))
	for i, c := range clients {
		result[i] = TransformClient(c)
	}
	return result
}

// Contact response transformation
type ContactResponse struct {
	ID        string  `json:"id"`
	ClientID  string  `json:"client_id"`
	Name      string  `json:"name"`
	Email     *string `json:"email"`
	Phone     *string `json:"phone"`
	Role      *string `json:"role"`
	IsPrimary bool    `json:"is_primary"`
	Notes     *string `json:"notes"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

func TransformContact(c sqlc.ClientContact) ContactResponse {
	isPrimary := false
	if c.IsPrimary != nil {
		isPrimary = *c.IsPrimary
	}

	return ContactResponse{
		ID:        pgtypeUUIDToStringRequired(c.ID),
		ClientID:  pgtypeUUIDToStringRequired(c.ClientID),
		Name:      c.Name,
		Email:     c.Email,
		Phone:     c.Phone,
		Role:      c.Role,
		IsPrimary: isPrimary,
		Notes:     c.Notes,
		CreatedAt: c.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt: c.UpdatedAt.Time.Format(time.RFC3339),
	}
}

func TransformContacts(contacts []sqlc.ClientContact) []ContactResponse {
	result := make([]ContactResponse, len(contacts))
	for i, c := range contacts {
		result[i] = TransformContact(c)
	}
	return result
}

// Interaction response transformation
type InteractionResponse struct {
	ID          string  `json:"id"`
	ClientID    string  `json:"client_id"`
	ContactID   *string `json:"contact_id"`
	Type        string  `json:"type"`
	Subject     string  `json:"subject"`
	Description *string `json:"description"`
	Outcome     *string `json:"outcome"`
	OccurredAt  string  `json:"occurred_at"`
	CreatedAt   string  `json:"created_at"`
}

func TransformInteraction(i sqlc.ClientInteraction) InteractionResponse {
	// InteractionType is not nullable in the model
	interactionType := string(i.Type)

	return InteractionResponse{
		ID:          pgtypeUUIDToStringRequired(i.ID),
		ClientID:    pgtypeUUIDToStringRequired(i.ClientID),
		ContactID:   pgtypeUUIDToString(i.ContactID),
		Type:        interactionType,
		Subject:     i.Subject,
		Description: i.Description,
		Outcome:     i.Outcome,
		OccurredAt:  i.OccurredAt.Time.Format(time.RFC3339),
		CreatedAt:   i.CreatedAt.Time.Format(time.RFC3339),
	}
}

func TransformInteractions(interactions []sqlc.ClientInteraction) []InteractionResponse {
	result := make([]InteractionResponse, len(interactions))
	for i, inter := range interactions {
		result[i] = TransformInteraction(inter)
	}
	return result
}

// Calendar Event Response
type CalendarEventResponse struct {
	ID            string           `json:"id"`
	UserID        string           `json:"user_id"`
	GoogleEventID *string          `json:"google_event_id"`
	CalendarID    *string          `json:"calendar_id"`
	Title         *string          `json:"title"`
	Description   *string          `json:"description"`
	StartTime     string           `json:"start_time"`
	EndTime       string           `json:"end_time"`
	AllDay        bool             `json:"all_day"`
	Location      *string          `json:"location"`
	Attendees     []map[string]any `json:"attendees"`
	Status        *string          `json:"status"`
	Visibility    *string          `json:"visibility"`
	HtmlLink      *string          `json:"html_link"`
	Source        *string          `json:"source"`
	ColorID       *string          `json:"color_id"`
	ColorHex      *string          `json:"color_hex"`
	MeetingType   string           `json:"meeting_type"`
	ContextID     *string          `json:"context_id"`
	ProjectID     *string          `json:"project_id"`
	ClientID      *string          `json:"client_id"`
	RecordingURL  *string          `json:"recording_url"`
	MeetingLink   *string          `json:"meeting_link"`
	ExternalLinks []string         `json:"external_links"`
	MeetingNotes  *string          `json:"meeting_notes"`
	ActionItems   []string         `json:"action_items"`
	SyncedAt      string           `json:"synced_at"`
	CreatedAt     string           `json:"created_at"`
	UpdatedAt     string           `json:"updated_at"`
}

func TransformCalendarEvent(e sqlc.CalendarEvent) CalendarEventResponse {
	meetingType := "other"
	if e.MeetingType.Valid {
		meetingType = strings.ToLower(string(e.MeetingType.Meetingtype))
	}

	allDay := false
	if e.AllDay != nil {
		allDay = *e.AllDay
	}

	var attendees []map[string]any
	if e.Attendees != nil {
		json.Unmarshal(e.Attendees, &attendees)
	}
	if attendees == nil {
		attendees = []map[string]any{}
	}

	var externalLinks []string
	if e.ExternalLinks != nil {
		json.Unmarshal(e.ExternalLinks, &externalLinks)
	}
	if externalLinks == nil {
		externalLinks = []string{}
	}

	var actionItems []string
	if e.ActionItems != nil {
		json.Unmarshal(e.ActionItems, &actionItems)
	}
	if actionItems == nil {
		actionItems = []string{}
	}

	return CalendarEventResponse{
		ID:            pgtypeUUIDToStringRequired(e.ID),
		UserID:        e.UserID,
		GoogleEventID: e.GoogleEventID,
		CalendarID:    e.CalendarID,
		Title:         e.Title,
		Description:   e.Description,
		StartTime:     e.StartTime.Time.Format(time.RFC3339),
		EndTime:       e.EndTime.Time.Format(time.RFC3339),
		AllDay:        allDay,
		Location:      e.Location,
		Attendees:     attendees,
		Status:        e.Status,
		Visibility:    e.Visibility,
		HtmlLink:      e.HtmlLink,
		Source:        e.Source,
		MeetingType:   meetingType,
		ContextID:     pgtypeUUIDToString(e.ContextID),
		ProjectID:     pgtypeUUIDToString(e.ProjectID),
		ClientID:      pgtypeUUIDToString(e.ClientID),
		RecordingURL:  e.RecordingUrl,
		MeetingLink:   e.MeetingLink,
		ExternalLinks: externalLinks,
		MeetingNotes:  e.MeetingNotes,
		ActionItems:   actionItems,
		SyncedAt:      e.SyncedAt.Time.Format(time.RFC3339),
		CreatedAt:     e.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:     e.UpdatedAt.Time.Format(time.RFC3339),
	}
}

func TransformCalendarEvents(events []sqlc.CalendarEvent) []CalendarEventResponse {
	result := make([]CalendarEventResponse, len(events))
	for i, e := range events {
		result[i] = TransformCalendarEvent(e)
	}
	return result
}
