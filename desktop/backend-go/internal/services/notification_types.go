package services

import "time"

// Notification type constants
const (
	// Tasks
	NotifTaskAssigned      = "task.assigned"
	NotifTaskUnassigned    = "task.unassigned"
	NotifTaskDueSoon       = "task.due_soon"
	NotifTaskDueToday      = "task.due_today"
	NotifTaskOverdue       = "task.overdue"
	NotifTaskCompleted     = "task.completed"
	NotifTaskStatusChanged = "task.status_changed"
	NotifTaskComment       = "task.comment"

	// Projects
	NotifProjectAdded           = "project.added"
	NotifProjectRemoved         = "project.removed"
	NotifProjectStatusChanged   = "project.status_changed"
	NotifProjectDeadlineChanged = "project.deadline_changed"
	NotifProjectCompleted       = "project.completed"

	// Team
	NotifTeamMemberJoined = "team.member_joined"
	NotifTeamMemberLeft   = "team.member_left"
	NotifTeamRoleChanged  = "team.role_changed"

	// Clients
	NotifClientAssigned         = "client.assigned"
	NotifClientDealUpdate       = "client.deal_update"
	NotifClientMeetingScheduled = "client.meeting_scheduled"

	// Daily log
	NotifDailyLogReminder = "dailylog.reminder"
	NotifDailyLogMention  = "dailylog.mention"

	// Chat/AI
	NotifChatArtifactReady    = "chat.artifact_ready"
	NotifChatLongTaskComplete = "chat.long_task_complete"

	// Integrations
	NotifIntegrationConnected     = "integration.connected"
	NotifIntegrationDisconnected  = "integration.disconnected"
	NotifIntegrationTokenExpiring = "integration.token_expiring"
	NotifIntegrationSyncFailed    = "integration.sync_failed"
	NotifIntegrationCalendarEvent = "integration.calendar_event"

	// Mentions
	NotifMentionTask     = "mention.task"
	NotifMentionProject  = "mention.project"
	NotifMentionComment  = "mention.comment"
	NotifMentionDailyLog = "mention.dailylog"

	// System
	NotifSystemWelcome             = "system.welcome"
	NotifSystemMaintenance         = "system.maintenance"
	NotifSystemFeatureAnnouncement = "system.feature_announcement"
)

// Priority levels
const (
	PriorityLow    = "low"
	PriorityNormal = "normal"
	PriorityHigh   = "high"
	PriorityUrgent = "urgent"
)

// Delivery channels
const (
	ChannelInApp = "in_app"
	ChannelPush  = "push"
	ChannelEmail = "email"
)

// BatchConfig defines batching behavior
type BatchConfig struct {
	Window  time.Duration
	Max     int
	GroupBy string // "entity_id", "sender_id", "type"
}

// TypeConfig defines config for a notification type
type TypeConfig struct {
	Priority string
	Channels []string
	Batch    *BatchConfig
}

// typeConfigs - internal lookup
var typeConfigs = map[string]TypeConfig{
	// High priority - no batching
	NotifTaskDueSoon:              {PriorityHigh, []string{ChannelInApp, ChannelPush, ChannelEmail}, nil},
	NotifTaskDueToday:             {PriorityHigh, []string{ChannelInApp, ChannelPush}, nil},
	NotifTaskOverdue:              {PriorityUrgent, []string{ChannelInApp, ChannelPush, ChannelEmail}, nil},
	NotifIntegrationDisconnected:  {PriorityHigh, []string{ChannelInApp, ChannelEmail}, nil},
	NotifIntegrationTokenExpiring: {PriorityHigh, []string{ChannelInApp, ChannelPush, ChannelEmail}, nil},
	NotifIntegrationSyncFailed:    {PriorityHigh, []string{ChannelInApp, ChannelEmail}, nil},
	NotifSystemMaintenance:        {PriorityHigh, []string{ChannelInApp, ChannelPush, ChannelEmail}, nil},

	// Normal priority - with batching
	NotifTaskAssigned:   {PriorityNormal, []string{ChannelInApp, ChannelPush}, &BatchConfig{30 * time.Second, 10, "type"}},
	NotifTaskCompleted:  {PriorityNormal, []string{ChannelInApp}, &BatchConfig{60 * time.Second, 20, "type"}},
	NotifTaskComment:    {PriorityNormal, []string{ChannelInApp, ChannelPush}, &BatchConfig{5 * time.Minute, 20, "entity_id"}},
	NotifProjectAdded:   {PriorityNormal, []string{ChannelInApp, ChannelPush}, &BatchConfig{30 * time.Second, 5, "type"}},
	NotifClientDealUpdate: {PriorityNormal, []string{ChannelInApp}, &BatchConfig{60 * time.Second, 10, "entity_id"}},
	NotifMentionTask:    {PriorityNormal, []string{ChannelInApp, ChannelPush}, &BatchConfig{5 * time.Minute, 10, "entity_id"}},
	NotifMentionProject: {PriorityNormal, []string{ChannelInApp, ChannelPush}, &BatchConfig{5 * time.Minute, 10, "entity_id"}},
	NotifMentionComment: {PriorityNormal, []string{ChannelInApp, ChannelPush}, &BatchConfig{5 * time.Minute, 10, "entity_id"}},

	// Normal priority - no batching
	NotifProjectRemoved:         {PriorityNormal, []string{ChannelInApp}, nil},
	NotifProjectStatusChanged:   {PriorityNormal, []string{ChannelInApp}, nil},
	NotifProjectDeadlineChanged: {PriorityNormal, []string{ChannelInApp, ChannelEmail}, nil},
	NotifProjectCompleted:       {PriorityNormal, []string{ChannelInApp, ChannelPush}, nil},
	NotifTeamRoleChanged:        {PriorityNormal, []string{ChannelInApp, ChannelEmail}, nil},
	NotifClientAssigned:         {PriorityNormal, []string{ChannelInApp, ChannelPush}, nil},
	NotifClientMeetingScheduled: {PriorityNormal, []string{ChannelInApp, ChannelPush, ChannelEmail}, nil},
	NotifDailyLogReminder:       {PriorityNormal, []string{ChannelInApp, ChannelPush}, nil},
	NotifChatArtifactReady:      {PriorityNormal, []string{ChannelInApp, ChannelPush}, nil},
	NotifChatLongTaskComplete:   {PriorityNormal, []string{ChannelInApp, ChannelPush}, nil},
	NotifIntegrationCalendarEvent: {PriorityNormal, []string{ChannelInApp, ChannelPush}, nil},

	// Low priority
	NotifTaskUnassigned:            {PriorityLow, []string{ChannelInApp}, &BatchConfig{30 * time.Second, 10, "type"}},
	NotifTaskStatusChanged:         {PriorityLow, []string{ChannelInApp}, &BatchConfig{60 * time.Second, 20, "entity_id"}},
	NotifTeamMemberJoined:          {PriorityLow, []string{ChannelInApp}, &BatchConfig{60 * time.Second, 10, "type"}},
	NotifTeamMemberLeft:            {PriorityLow, []string{ChannelInApp}, nil},
	NotifDailyLogMention:           {PriorityLow, []string{ChannelInApp}, &BatchConfig{5 * time.Minute, 10, "entity_id"}},
	NotifMentionDailyLog:           {PriorityLow, []string{ChannelInApp}, &BatchConfig{5 * time.Minute, 10, "entity_id"}},
	NotifIntegrationConnected:      {PriorityLow, []string{ChannelInApp}, nil},
	NotifSystemWelcome:             {PriorityLow, []string{ChannelInApp}, nil},
	NotifSystemFeatureAnnouncement: {PriorityLow, []string{ChannelInApp}, nil},
}

// GetTypeConfig returns config for a notification type
func GetTypeConfig(notifType string) (TypeConfig, bool) {
	cfg, ok := typeConfigs[notifType]
	return cfg, ok
}

// IsBatchable returns true if the notification type supports batching
func IsBatchable(notifType string) bool {
	cfg, ok := typeConfigs[notifType]
	return ok && cfg.Batch != nil
}
