package handlers

import (
	"time"

	"github.com/google/uuid"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
)

// =============================================================================
// MOBILE RESPONSE TYPES
// =============================================================================
// These types are optimized for mobile:
// 1. Flat structure (no nested objects in lists)
// 2. Smaller field names where possible
// 3. Unix timestamps instead of ISO strings (smaller, easier to parse)
// 4. Optional fields use omitempty to skip null values
// 5. Only essential fields - heavy data fetched on-demand
//
// Size comparison:
// - Web API TaskResponse: ~2KB (with nested assignee, project, comments)
// - Mobile MobileTaskResponse: ~100 bytes (flat, essential fields only)
// =============================================================================

// =============================================================================
// USER / ME RESPONSE
// =============================================================================

// MobileUserResponse is the lean user profile returned by GET /me
type MobileUserResponse struct {
	ID        string  `json:"id"`
	Email     string  `json:"email"`
	Name      string  `json:"name"`
	AvatarURL *string `json:"avatar_url,omitempty"`
	Timezone  string  `json:"timezone,omitempty"`
}

// MobileWorkspaceResponse represents workspace context (for future multi-tenant)
type MobileWorkspaceResponse struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
	Role string `json:"role,omitempty"`
}

// MobilePreferencesResponse represents user notification preferences
type MobilePreferencesResponse struct {
	NotificationsEnabled bool    `json:"notifications_enabled"`
	QuietHoursStart      *string `json:"quiet_hours_start,omitempty"`
	QuietHoursEnd        *string `json:"quiet_hours_end,omitempty"`
}

// MobileMeResponse is the full response for GET /api/mobile/v1/me
type MobileMeResponse struct {
	User        MobileUserResponse         `json:"user"`
	Workspace   *MobileWorkspaceResponse   `json:"workspace,omitempty"`
	Preferences *MobilePreferencesResponse `json:"preferences,omitempty"`
}

// =============================================================================
// TASK RESPONSES
// =============================================================================

// MobileTaskResponse is the lean task for list views (~100 bytes)
// Used in GET /tasks list responses
type MobileTaskResponse struct {
	ID        uuid.UUID `json:"id"`
	Title     string    `json:"title"`
	Status    string    `json:"status"`
	Priority  string    `json:"priority"`
	DueDate   *string   `json:"due_date,omitempty"` // "2026-01-05" format
	Assignee  *string   `json:"assignee,omitempty"` // Just the name, not full object
	Project   *string   `json:"project,omitempty"`  // Just the name, not full object
	UpdatedAt int64     `json:"updated_at"`         // Unix timestamp
}

// MobileTaskDetailResponse is the full task for detail views (~2KB)
// Used in GET /tasks/:id responses - fetched on-demand when user taps a task
type MobileTaskDetailResponse struct {
	ID          uuid.UUID                 `json:"id"`
	Title       string                    `json:"title"`
	Description *string                   `json:"description,omitempty"`
	Status      string                    `json:"status"`
	Priority    string                    `json:"priority"`
	DueDate     *string                   `json:"due_date,omitempty"`
	StartDate   *string                   `json:"start_date,omitempty"`
	CompletedAt *string                   `json:"completed_at,omitempty"`
	Assignee    *MobileAssigneeResponse   `json:"assignee,omitempty"`
	Project     *MobileProjectRefResponse `json:"project,omitempty"`
	Tags        []string                  `json:"tags,omitempty"`
	// Counts instead of full arrays - fetch separately if needed
	CommentsCount    int   `json:"comments_count"`
	AttachmentsCount int   `json:"attachments_count"`
	SubtasksCount    int   `json:"subtasks_count"`
	CreatedAt        int64 `json:"created_at"`
	UpdatedAt        int64 `json:"updated_at"`
}

// MobileAssigneeResponse is a minimal user reference
type MobileAssigneeResponse struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	AvatarURL *string `json:"avatar_url,omitempty"`
}

// MobileProjectRefResponse is a minimal project reference
type MobileProjectRefResponse struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

// MobileTaskListResponse wraps the task list with pagination
type MobileTaskListResponse struct {
	Tasks   []interface{} `json:"tasks"` // Can be MobileTaskResponse or filtered map
	Cursor  string        `json:"cursor,omitempty"`
	HasMore bool          `json:"has_more"`
	Total   *int          `json:"total,omitempty"` // Optional, expensive to compute
}

// MobileTaskQuickCreateRequest is the minimal input for quick task creation
type MobileTaskQuickCreateRequest struct {
	Title    string  `json:"title" binding:"required,min=1,max=500"`
	DueDate  *string `json:"due_date,omitempty"` // "2026-01-05" format
	Priority *string `json:"priority,omitempty"` // "low", "medium", "high", "critical"
}

// MobileTaskStatusUpdateRequest is for PUT /tasks/:id/status
type MobileTaskStatusUpdateRequest struct {
	Status string `json:"status" binding:"required,oneof=todo in_progress done cancelled"`
}

// MobileTaskStatusResponse is the lean response after status updates
type MobileTaskStatusResponse struct {
	ID          uuid.UUID `json:"id"`
	Status      string    `json:"status"`
	CompletedAt *string   `json:"completed_at,omitempty"`
	UpdatedAt   int64     `json:"updated_at"`
}

// =============================================================================
// TRANSFORMERS
// =============================================================================
// These functions convert from sqlc database types to mobile response types.
// They handle null checking, format conversion, and field mapping.
// =============================================================================

// TransformToMobileTask converts a sqlc Task to MobileTaskResponse
func TransformToMobileTask(task sqlc.Task, projectName *string, assigneeName *string) MobileTaskResponse {
	return MobileTaskResponse{
		ID:        task.ID.Bytes,
		Title:     task.Title,
		Status:    string(task.Status.Taskstatus),
		Priority:  string(task.Priority.Taskpriority),
		DueDate:   formatNullableTimestamp(task.DueDate),
		Assignee:  assigneeName,
		Project:   projectName,
		UpdatedAt: task.UpdatedAt.Time.Unix(),
	}
}

// TransformToMobileTaskDetail converts a sqlc Task to full detail response
func TransformToMobileTaskDetail(
	task sqlc.Task,
	project *sqlc.Project,
	assignee *sqlc.TeamMember,
	subtaskCount int,
) MobileTaskDetailResponse {
	resp := MobileTaskDetailResponse{
		ID:        task.ID.Bytes,
		Title:     task.Title,
		Status:    string(task.Status.Taskstatus),
		Priority:  string(task.Priority.Taskpriority),
		DueDate:   formatNullableTimestamp(task.DueDate),
		StartDate: formatNullableTimestamp(task.StartDate),
		Tags:      []string{}, // Tags feature not yet implemented
		// Counts - fetched separately for efficiency
		CommentsCount:    0, // Populated by caller when available
		AttachmentsCount: 0, // Attachments feature not yet implemented
		SubtasksCount:    subtaskCount,
		CreatedAt:        task.CreatedAt.Time.Unix(),
		UpdatedAt:        task.UpdatedAt.Time.Unix(),
	}

	// Handle nullable description (Task.Description is *string not pgtype)
	if task.Description != nil {
		resp.Description = task.Description
	}

	// Handle nullable completed_at
	if task.CompletedAt.Valid {
		s := task.CompletedAt.Time.Format(time.RFC3339)
		resp.CompletedAt = &s
	}

	// Handle project reference
	if project != nil {
		resp.Project = &MobileProjectRefResponse{
			ID:   project.ID.Bytes,
			Name: project.Name,
		}
	}

	// Handle assignee reference
	if assignee != nil {
		var avatarURL *string
		if assignee.AvatarUrl != nil {
			avatarURL = assignee.AvatarUrl
		}
		resp.Assignee = &MobileAssigneeResponse{
			ID:        uuid.UUID(assignee.ID.Bytes).String(),
			Name:      assignee.Name,
			AvatarURL: avatarURL,
		}
	}

	return resp
}

// =============================================================================
// NOTIFICATION TRANSFORMERS
// =============================================================================

func TransformToMobileNotification(row sqlc.ListNotificationsForMobileRow) MobileNotificationResponse {
	resp := MobileNotificationResponse{
		ID:         uuid.UUID(row.ID.Bytes),
		Type:       row.Type,
		Title:      row.Title,
		Body:       row.Body,
		EntityType: row.EntityType,
		Priority:   "normal",
		IsRead:     false,
		CreatedAt:  row.CreatedAt.Time.Unix(),
	}

	if row.Priority != nil {
		resp.Priority = *row.Priority
	}
	if row.IsRead != nil {
		resp.IsRead = *row.IsRead
	}
	if row.EntityID.Valid {
		entityStr := uuid.UUID(row.EntityID.Bytes).String()
		resp.EntityID = &entityStr
	}

	return resp
}

// =============================================================================
// NOTIFICATION RESPONSES (Phase 2 - defined here for consistency)
// =============================================================================

// MobileNotificationResponse is the lean notification for list views
type MobileNotificationResponse struct {
	ID         uuid.UUID `json:"id"`
	Type       string    `json:"type"`
	Title      string    `json:"title"`
	Body       *string   `json:"body,omitempty"`
	EntityType *string   `json:"entity_type,omitempty"`
	EntityID   *string   `json:"entity_id,omitempty"`
	Priority   string    `json:"priority"`
	IsRead     bool      `json:"is_read"`
	CreatedAt  int64     `json:"created_at"`
}

// MobileNotificationListResponse wraps notification list with metadata
type MobileNotificationListResponse struct {
	Notifications []MobileNotificationResponse `json:"notifications"`
	Cursor        string                       `json:"cursor,omitempty"`
	HasMore       bool                         `json:"has_more"`
	UnreadCount   int                          `json:"unread_count"`
}

// MobileNotificationCountResponse is the minimal badge count response (~20 bytes)
type MobileNotificationCountResponse struct {
	UnreadCount int `json:"unread_count"`
}

// MobileNotificationReadRequest is for batch marking as read
type MobileNotificationReadRequest struct {
	IDs []uuid.UUID `json:"ids,omitempty"`
	All bool        `json:"all,omitempty"`
}

// MobileNotificationReadResponse is the response after marking read
type MobileNotificationReadResponse struct {
	MarkedCount int `json:"marked_count"`
	UnreadCount int `json:"unread_count"`
}

// =============================================================================
// DAILY LOG RESPONSES (Phase 2 - defined here for consistency)
// =============================================================================

// MobileDailyLogEntryResponse is a single log entry
type MobileDailyLogEntryResponse struct {
	ID           uuid.UUID  `json:"id"`
	Content      string     `json:"content"`
	Type         string     `json:"type,omitempty"` // "note", "accomplishment", etc.
	LinkedTaskID *uuid.UUID `json:"linked_task_id,omitempty"`
	CreatedAt    int64      `json:"created_at"`
}

// MobileDailyLogTodayResponse is the response for GET /dailylog/today
type MobileDailyLogTodayResponse struct {
	Date        string                        `json:"date"`
	Content     string                        `json:"content,omitempty"`
	Entries     []MobileDailyLogEntryResponse `json:"entries,omitempty"`
	EnergyLevel *int32                        `json:"energy_level,omitempty"`
	Summary     *string                       `json:"summary,omitempty"`
	Mood        *string                       `json:"mood,omitempty"`
}

// MobileDailyLogHistoryItem is a summary of one day's log
type MobileDailyLogHistoryItem struct {
	Date       string  `json:"date"`
	EntryCount int     `json:"entry_count"`
	Summary    *string `json:"summary,omitempty"`
	Mood       *string `json:"mood,omitempty"`
}

// MobileDailyLogHistoryResponse is the response for GET /dailylog/history
type MobileDailyLogHistoryResponse struct {
	Logs    []MobileDailyLogHistoryItem `json:"logs"`
	HasMore bool                        `json:"has_more"`
	Before  string                      `json:"before,omitempty"`
}

// MobileDailyLogTodayResponse updated to include content directly
type MobileDailyLogTodayResponseV2 struct {
	Date        string `json:"date"`
	Content     string `json:"content,omitempty"`
	EnergyLevel *int32 `json:"energy_level,omitempty"`
}

// MobileDailyLogEntryRequest is for creating a new entry
type MobileDailyLogEntryRequest struct {
	Content      string     `json:"content" binding:"required,min=1"`
	Type         string     `json:"type,omitempty"`
	LinkedTaskID *uuid.UUID `json:"linked_task_id,omitempty"`
}

// =============================================================================
// SYNC RESPONSES
// =============================================================================

type MobileSyncResponse struct {
	Tasks         []MobileTaskResponse         `json:"tasks"`
	Notifications []MobileNotificationResponse `json:"notifications"`
	ServerTime    int64                        `json:"server_time"`
	HasMore       bool                         `json:"has_more"`
}

// =============================================================================
// SYNC TRANSFORMERS
// =============================================================================

func TransformSyncTaskRow(row sqlc.GetTaskChangesSinceRow) MobileTaskResponse {
	resp := MobileTaskResponse{
		ID:        uuid.UUID(row.ID.Bytes),
		Title:     row.Title,
		UpdatedAt: row.UpdatedAt.Time.Unix(),
	}

	if row.Status.Valid {
		resp.Status = string(row.Status.Taskstatus)
	}
	if row.Priority.Valid {
		resp.Priority = string(row.Priority.Taskpriority)
	}
	if row.DueDate.Valid {
		dateStr := row.DueDate.Time.Format("2006-01-02")
		resp.DueDate = &dateStr
	}
	if row.ProjectName != nil {
		resp.Project = row.ProjectName
	}
	if row.AssigneeName != nil {
		resp.Assignee = row.AssigneeName
	}

	return resp
}

func TransformSyncNotificationRow(row sqlc.GetNotificationChangesSinceRow) MobileNotificationResponse {
	resp := MobileNotificationResponse{
		ID:         uuid.UUID(row.ID.Bytes),
		Type:       row.Type,
		Title:      row.Title,
		Body:       row.Body,
		EntityType: row.EntityType,
		Priority:   "normal",
		IsRead:     false,
		CreatedAt:  row.CreatedAt.Time.Unix(),
	}

	if row.Priority != nil {
		resp.Priority = *row.Priority
	}
	if row.IsRead != nil {
		resp.IsRead = *row.IsRead
	}
	if row.EntityID.Valid {
		entityStr := uuid.UUID(row.EntityID.Bytes).String()
		resp.EntityID = &entityStr
	}

	return resp
}

func TransformToDailyLogHistoryItem(row sqlc.GetDailyLogHistoryForMobileRow) MobileDailyLogHistoryItem {
	item := MobileDailyLogHistoryItem{
		Date: row.Date.Time.Format("2006-01-02"),
	}
	if row.EnergyLevel != nil {
		energy := int(*row.EnergyLevel)
		item.EntryCount = energy
	}
	return item
}

// =============================================================================
// CHAT TYPES
// =============================================================================

type MobileChatThreadResponse struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	LastMessage string    `json:"last_message,omitempty"`
	UpdatedAt   int64     `json:"updated_at"`
}

type MobileChatThreadListResponse struct {
	Threads []MobileChatThreadResponse `json:"threads"`
}

type MobileChatMessageResponse struct {
	ID        uuid.UUID `json:"id"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	CreatedAt int64     `json:"created_at"`
}

type MobileChatHistoryResponse struct {
	Messages []MobileChatMessageResponse `json:"messages"`
	Cursor   string                      `json:"cursor,omitempty"`
	HasMore  bool                        `json:"has_more"`
}

type MobileChatMessageRequest struct {
	ConversationID uuid.UUID `json:"conversation_id" binding:"required"`
	Content        string    `json:"content" binding:"required"`
}

// =============================================================================
// SMART CAPTURE TYPES
// =============================================================================

type MobileSmartCaptureRequest struct {
	Text string `json:"text" binding:"required"`
}

type MobileSmartCaptureResponse struct {
	Type      string     `json:"type"`
	Title     string     `json:"title"`
	DueDate   *time.Time `json:"due_date,omitempty"`
	Priority  string     `json:"priority,omitempty"`
	CreatedID *uuid.UUID `json:"created_id,omitempty"`
}

type SmartCaptureParsed struct {
	Type     string
	Title    string
	DueDate  *time.Time
	Priority string
}

// =============================================================================
// PUSH REGISTRATION TYPES
// =============================================================================

type MobilePushRegisterRequest struct {
	DeviceID    string `json:"device_id"`
	Platform    string `json:"platform" binding:"required,oneof=ios android web"`
	PushToken   string `json:"push_token" binding:"required"`
	AppVersion  string `json:"app_version"`
	OsVersion   string `json:"os_version"`
	DeviceModel string `json:"device_model"`
}

// =============================================================================
// CHAT TRANSFORMERS
// =============================================================================

func TransformToChatThread(row sqlc.ListConversationsForMobileRow) MobileChatThreadResponse {
	resp := MobileChatThreadResponse{
		ID: uuid.UUID(row.ID.Bytes),
	}
	if row.Title != nil {
		resp.Title = *row.Title
	} else {
		resp.Title = "Untitled"
	}
	if row.UpdatedAt.Valid {
		resp.UpdatedAt = row.UpdatedAt.Time.Unix()
	}
	return resp
}

func TransformToChatMessage(row sqlc.GetMessagesForMobileRow) MobileChatMessageResponse {
	return MobileChatMessageResponse{
		ID:        uuid.UUID(row.ID.Bytes),
		Role:      string(row.Role),
		Content:   row.Content,
		CreatedAt: row.CreatedAt.Time.Unix(),
	}
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// formatNullableTimestamp converts pgtype.Timestamp to *string in date format
func formatNullableTimestamp(ts interface{}) *string {
	// Handle different pgtype timestamp types
	switch t := ts.(type) {
	case struct {
		Time  time.Time
		Valid bool
	}:
		if !t.Valid {
			return nil
		}
		s := t.Time.Format("2006-01-02")
		return &s
	default:
		return nil
	}
}
