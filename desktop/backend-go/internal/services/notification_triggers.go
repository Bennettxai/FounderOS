package services

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

// NotificationTriggers provides methods to trigger notifications for various events
type NotificationTriggers struct {
	notifService *NotificationService
}

// NewNotificationTriggers creates a new triggers instance
func NewNotificationTriggers(notifService *NotificationService) *NotificationTriggers {
	return &NotificationTriggers{notifService: notifService}
}

// ========== TASK TRIGGERS ==========

// TaskAssignedInput contains data for task assignment notification
type TaskAssignedInput struct {
	TaskID       uuid.UUID
	TaskTitle    string
	AssigneeID   string // User being assigned
	AssignerID   string // User who assigned
	AssignerName string
	ProjectID    *uuid.UUID
	ProjectName  string
}

// OnTaskAssigned triggers notification when a task is assigned to someone
func (t *NotificationTriggers) OnTaskAssigned(ctx context.Context, input TaskAssignedInput) {
	// Don't notify if assigning to yourself
	if input.AssigneeID == input.AssignerID {
		return
	}

	_, err := t.notifService.Create(ctx, CreateInput{
		UserID:     input.AssigneeID,
		Type:       NotifTaskAssigned,
		Title:      fmt.Sprintf("%s assigned you a task", input.AssignerName),
		Body:       input.TaskTitle,
		EntityType: "task",
		EntityID:   &input.TaskID,
		SenderID:   input.AssignerID,
		SenderName: input.AssignerName,
		Metadata: map[string]interface{}{
			"task_id":      input.TaskID.String(),
			"task_title":   input.TaskTitle,
			"project_id":   uuidPtrToString(input.ProjectID),
			"project_name": input.ProjectName,
		},
	})
	if err != nil {
		slog.Info("[NotificationTriggers] OnTaskAssigned error", "error", err)
	}
}

// TaskCompletedInput contains data for task completion notification
type TaskCompletedInput struct {
	TaskID        uuid.UUID
	TaskTitle     string
	CompletedByID string
	CompletedBy   string
	OwnerID       string // Task creator/owner to notify
	ProjectID     *uuid.UUID
}

// OnTaskCompleted triggers notification when a task is marked complete
func (t *NotificationTriggers) OnTaskCompleted(ctx context.Context, input TaskCompletedInput) {
	// Only notify task owner if someone else completed it
	if input.OwnerID == input.CompletedByID {
		return
	}

	_, err := t.notifService.Create(ctx, CreateInput{
		UserID:     input.OwnerID,
		Type:       NotifTaskCompleted,
		Title:      fmt.Sprintf("%s completed a task", input.CompletedBy),
		Body:       input.TaskTitle,
		EntityType: "task",
		EntityID:   &input.TaskID,
		SenderID:   input.CompletedByID,
		SenderName: input.CompletedBy,
		Metadata: map[string]interface{}{
			"task_id":    input.TaskID.String(),
			"task_title": input.TaskTitle,
			"project_id": uuidPtrToString(input.ProjectID),
		},
	})
	if err != nil {
		slog.Info("[NotificationTriggers] OnTaskCompleted error", "error", err)
	}
}

// TaskStatusChangedInput contains data for task status change notification
type TaskStatusChangedInput struct {
	TaskID      uuid.UUID
	TaskTitle   string
	OldStatus   string
	NewStatus   string
	ChangedByID string
	ChangedBy   string
	AssigneeID  string // Person assigned to the task
}

// OnTaskStatusChanged triggers notification when task status changes
func (t *NotificationTriggers) OnTaskStatusChanged(ctx context.Context, input TaskStatusChangedInput) {
	// Don't notify if assignee made the change
	if input.AssigneeID == input.ChangedByID || input.AssigneeID == "" {
		return
	}

	_, err := t.notifService.Create(ctx, CreateInput{
		UserID:     input.AssigneeID,
		Type:       NotifTaskStatusChanged,
		Title:      fmt.Sprintf("Task status changed to %s", input.NewStatus),
		Body:       input.TaskTitle,
		EntityType: "task",
		EntityID:   &input.TaskID,
		SenderID:   input.ChangedByID,
		SenderName: input.ChangedBy,
		Metadata: map[string]interface{}{
			"task_id":    input.TaskID.String(),
			"task_title": input.TaskTitle,
			"old_status": input.OldStatus,
			"new_status": input.NewStatus,
		},
	})
	if err != nil {
		slog.Info("[NotificationTriggers] OnTaskStatusChanged error", "error", err)
	}
}

// TaskCommentInput contains data for task comment notification
type TaskCommentInput struct {
	TaskID       uuid.UUID
	TaskTitle    string
	CommentID    uuid.UUID
	CommentText  string
	CommenterID  string
	CommenterName string
	TaskOwnerID  string
	AssigneeID   string
}

// OnTaskComment triggers notification when someone comments on a task
func (t *NotificationTriggers) OnTaskComment(ctx context.Context, input TaskCommentInput) {
	// Collect users to notify (owner + assignee, excluding commenter)
	usersToNotify := make(map[string]bool)
	if input.TaskOwnerID != "" && input.TaskOwnerID != input.CommenterID {
		usersToNotify[input.TaskOwnerID] = true
	}
	if input.AssigneeID != "" && input.AssigneeID != input.CommenterID {
		usersToNotify[input.AssigneeID] = true
	}

	for userID := range usersToNotify {
		_, err := t.notifService.Create(ctx, CreateInput{
			UserID:     userID,
			Type:       NotifTaskComment,
			Title:      fmt.Sprintf("%s commented on a task", input.CommenterName),
			Body:       truncateString(input.CommentText, 100),
			EntityType: "task",
			EntityID:   &input.TaskID,
			SenderID:   input.CommenterID,
			SenderName: input.CommenterName,
			Metadata: map[string]interface{}{
				"task_id":    input.TaskID.String(),
				"task_title": input.TaskTitle,
				"comment_id": input.CommentID.String(),
			},
		})
		if err != nil {
			slog.Info("[NotificationTriggers] OnTaskComment error for user", "value", userID, "error", err)
		}
	}
}

// ========== PROJECT TRIGGERS ==========

// ProjectMemberAddedInput contains data for project member addition
type ProjectMemberAddedInput struct {
	ProjectID    uuid.UUID
	ProjectName  string
	AddedUserID  string
	AddedByID    string
	AddedByName  string
	Role         string
}

// OnProjectMemberAdded triggers notification when someone is added to a project
func (t *NotificationTriggers) OnProjectMemberAdded(ctx context.Context, input ProjectMemberAddedInput) {
	if input.AddedUserID == input.AddedByID {
		return
	}

	_, err := t.notifService.Create(ctx, CreateInput{
		UserID:     input.AddedUserID,
		Type:       NotifProjectAdded,
		Title:      fmt.Sprintf("You were added to %s", input.ProjectName),
		Body:       fmt.Sprintf("%s added you as %s", input.AddedByName, input.Role),
		EntityType: "project",
		EntityID:   &input.ProjectID,
		SenderID:   input.AddedByID,
		SenderName: input.AddedByName,
		Metadata: map[string]interface{}{
			"project_id":   input.ProjectID.String(),
			"project_name": input.ProjectName,
			"role":         input.Role,
		},
	})
	if err != nil {
		slog.Info("[NotificationTriggers] OnProjectMemberAdded error", "error", err)
	}
}

// ProjectStatusChangedInput contains data for project status change
type ProjectStatusChangedInput struct {
	ProjectID    uuid.UUID
	ProjectName  string
	OldStatus    string
	NewStatus    string
	ChangedByID  string
	ChangedBy    string
	MemberIDs    []string // All project members to notify
}

// OnProjectStatusChanged triggers notification when project status changes
func (t *NotificationTriggers) OnProjectStatusChanged(ctx context.Context, input ProjectStatusChangedInput) {
	for _, memberID := range input.MemberIDs {
		if memberID == input.ChangedByID {
			continue
		}

		_, err := t.notifService.Create(ctx, CreateInput{
			UserID:     memberID,
			Type:       NotifProjectStatusChanged,
			Title:      fmt.Sprintf("%s status changed to %s", input.ProjectName, input.NewStatus),
			Body:       fmt.Sprintf("Changed by %s", input.ChangedBy),
			EntityType: "project",
			EntityID:   &input.ProjectID,
			SenderID:   input.ChangedByID,
			SenderName: input.ChangedBy,
			Metadata: map[string]interface{}{
				"project_id":   input.ProjectID.String(),
				"project_name": input.ProjectName,
				"old_status":   input.OldStatus,
				"new_status":   input.NewStatus,
			},
		})
		if err != nil {
			slog.Info("[NotificationTriggers] OnProjectStatusChanged error for user", "value", memberID, "error", err)
		}
	}
}

// ========== MENTION TRIGGERS ==========

// MentionRegex matches @username patterns
var MentionRegex = regexp.MustCompile(`@(\w+)`)

// MentionInput contains data for mention notifications
type MentionInput struct {
	Text          string
	EntityType    string // "task", "project", "comment", "dailylog"
	EntityID      uuid.UUID
	EntityTitle   string
	MentionerID   string
	MentionerName string
	// UserLookup maps usernames to user IDs
	UserLookup    map[string]string
}

// OnMention triggers notifications for @mentions in text
func (t *NotificationTriggers) OnMention(ctx context.Context, input MentionInput) {
	matches := MentionRegex.FindAllStringSubmatch(input.Text, -1)
	if len(matches) == 0 {
		return
	}

	// Deduplicate mentions
	mentioned := make(map[string]bool)
	for _, match := range matches {
		if len(match) > 1 {
			username := strings.ToLower(match[1])
			mentioned[username] = true
		}
	}

	// Get notification type based on entity
	notifType := NotifMentionComment
	switch input.EntityType {
	case "task":
		notifType = NotifMentionTask
	case "project":
		notifType = NotifMentionProject
	case "dailylog":
		notifType = NotifMentionDailyLog
	}

	// Send notifications
	for username := range mentioned {
		userID, ok := input.UserLookup[username]
		if !ok || userID == input.MentionerID {
			continue
		}

		_, err := t.notifService.Create(ctx, CreateInput{
			UserID:     userID,
			Type:       notifType,
			Title:      fmt.Sprintf("%s mentioned you", input.MentionerName),
			Body:       truncateString(input.Text, 100),
			EntityType: input.EntityType,
			EntityID:   &input.EntityID,
			SenderID:   input.MentionerID,
			SenderName: input.MentionerName,
			Metadata: map[string]interface{}{
				"entity_type":  input.EntityType,
				"entity_id":    input.EntityID.String(),
				"entity_title": input.EntityTitle,
			},
		})
		if err != nil {
			slog.Info("[NotificationTriggers] OnMention error for user", "value", userID, "error", err)
		}
	}
}

// ========== COMMENT & MENTION TRIGGERS ==========

// OnTaskCommentEvent - event struct for comment service
type OnTaskCommentEvent struct {
	TaskID        uuid.UUID
	TaskTitle     string
	CommentID     uuid.UUID
	CommentText   string
	CommenterID   string
	CommenterName string
	TaskOwnerID   string
}

// OnMentionEvent - event struct for mention notifications
type OnMentionEvent struct {
	MentionedUserID string
	MentionerID     string
	MentionerName   string
	SourceType      string // "comment", "task", "note"
	SourceID        uuid.UUID
	EntityType      string // "task", "project"
	EntityID        uuid.UUID
	EntityTitle     string
	Context         string // Text snippet with the mention
}

// OnCommentReplyEvent - event struct for reply notifications
type OnCommentReplyEvent struct {
	ParentCommentID  uuid.UUID
	ParentAuthorID   string
	ReplyID          uuid.UUID
	ReplyText        string
	ReplierID        string
	ReplierName      string
	EntityType       string
	EntityID         uuid.UUID
	EntityTitle      string
}

// OnTaskComment triggers notification when someone comments on a task (called by CommentService)
func (ns *NotificationService) OnTaskComment(ctx context.Context, event OnTaskCommentEvent) {
	if event.TaskOwnerID == "" || event.TaskOwnerID == event.CommenterID {
		return
	}

	_, err := ns.Create(ctx, CreateInput{
		UserID:     event.TaskOwnerID,
		Type:       NotifTaskComment,
		Title:      fmt.Sprintf("%s commented on %s", event.CommenterName, event.TaskTitle),
		Body:       truncateString(event.CommentText, 100),
		EntityType: "task",
		EntityID:   &event.TaskID,
		SenderID:   event.CommenterID,
		SenderName: event.CommenterName,
		Metadata: map[string]interface{}{
			"task_id":    event.TaskID.String(),
			"task_title": event.TaskTitle,
			"comment_id": event.CommentID.String(),
		},
	})
	if err != nil {
		slog.Info("[NotificationService] OnTaskComment error", "error", err)
	}
}

// OnMention triggers notification when a user is @mentioned
func (ns *NotificationService) OnMention(ctx context.Context, event OnMentionEvent) {
	if event.MentionedUserID == event.MentionerID {
		return
	}

	notifType := NotifMentionComment
	switch event.EntityType {
	case "task":
		notifType = NotifMentionTask
	case "project":
		notifType = NotifMentionProject
	}

	title := fmt.Sprintf("%s mentioned you", event.MentionerName)
	if event.EntityTitle != "" {
		title = fmt.Sprintf("%s mentioned you in %s", event.MentionerName, event.EntityTitle)
	}

	_, err := ns.Create(ctx, CreateInput{
		UserID:     event.MentionedUserID,
		Type:       notifType,
		Title:      title,
		Body:       truncateString(event.Context, 100),
		EntityType: event.EntityType,
		EntityID:   &event.EntityID,
		SenderID:   event.MentionerID,
		SenderName: event.MentionerName,
		Metadata: map[string]interface{}{
			"source_type":  event.SourceType,
			"source_id":    event.SourceID.String(),
			"entity_type":  event.EntityType,
			"entity_id":    event.EntityID.String(),
			"entity_title": event.EntityTitle,
		},
	})
	if err != nil {
		slog.Info("[NotificationService] OnMention error", "error", err)
	}
}

// OnCommentReply triggers notification when someone replies to a comment
func (ns *NotificationService) OnCommentReply(ctx context.Context, event OnCommentReplyEvent) {
	if event.ParentAuthorID == event.ReplierID {
		return
	}

	_, err := ns.Create(ctx, CreateInput{
		UserID:     event.ParentAuthorID,
		Type:       NotifTaskComment, // Reuse task comment type for replies
		Title:      fmt.Sprintf("%s replied to your comment", event.ReplierName),
		Body:       truncateString(event.ReplyText, 100),
		EntityType: event.EntityType,
		EntityID:   &event.EntityID,
		SenderID:   event.ReplierID,
		SenderName: event.ReplierName,
		Metadata: map[string]interface{}{
			"parent_comment_id": event.ParentCommentID.String(),
			"reply_id":          event.ReplyID.String(),
			"entity_type":       event.EntityType,
			"entity_id":         event.EntityID.String(),
			"entity_title":      event.EntityTitle,
		},
	})
	if err != nil {
		slog.Info("[NotificationService] OnCommentReply error", "error", err)
	}
}

// ========== SYSTEM TRIGGERS ==========

// OnWelcome sends a welcome notification to new users
func (t *NotificationTriggers) OnWelcome(ctx context.Context, userID string, userName string) {
	_, err := t.notifService.Create(ctx, CreateInput{
		UserID:     userID,
		Type:       NotifSystemWelcome,
		Title:      fmt.Sprintf("Welcome to BusinessOS, %s!", userName),
		Body:       "Get started by creating your first project or task.",
		EntityType: "system",
		Metadata: map[string]interface{}{
			"action": "onboarding",
		},
	})
	if err != nil {
		slog.Info("[NotificationTriggers] OnWelcome error", "error", err)
	}
}

// ========== HELPERS ==========

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func uuidPtrToString(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
}
