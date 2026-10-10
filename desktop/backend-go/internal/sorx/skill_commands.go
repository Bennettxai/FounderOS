// Package sorx provides skill-based commands for the Sorx engine.
package sorx

// SkillCommand represents a slash command that triggers a Sorx skill.
type SkillCommand struct {
	Name        string   `json:"name"`         // Command name (without /)
	DisplayName string   `json:"display_name"` // User-friendly name
	Description string   `json:"description"`  // What this command does
	Icon        string   `json:"icon"`         // Lucide icon name
	Category    string   `json:"category"`     // integration, automation, sync
	SkillID     string   `json:"skill_id"`     // ID of the Sorx skill to trigger
	Params      []string `json:"params"`       // Parameter names that can be passed
}

// BuiltInSkillCommands are commands that trigger Sorx skills.
// These bridge the command system to the Sorx skill execution engine.
var BuiltInSkillCommands = map[string]SkillCommand{
	// Communication Skills
	"process-inbox": {
		Name:        "process-inbox",
		DisplayName: "Process Inbox",
		Description: "Scan email inbox and extract actionable items into tasks",
		Icon:        "mail-open",
		Category:    "integration",
		SkillID:     "email.process_inbox",
		Params:      []string{"max_emails", "label"},
	},
	"send-email": {
		Name:        "send-email",
		DisplayName: "Send Email",
		Description: "Compose and send an email using connected Gmail",
		Icon:        "send",
		Category:    "integration",
		SkillID:     "email.send",
		Params:      []string{"to", "subject", "body"},
	},

	// CRM Skills
	"sync-crm": {
		Name:        "sync-crm",
		DisplayName: "Sync CRM",
		Description: "Sync contacts from HubSpot to BusinessOS clients",
		Icon:        "users",
		Category:    "sync",
		SkillID:     "crm.sync_contacts",
		Params:      []string{},
	},
	"import-contacts": {
		Name:        "import-contacts",
		DisplayName: "Import Contacts",
		Description: "Import new contacts from CRM with review",
		Icon:        "user-plus",
		Category:    "integration",
		SkillID:     "crm.import_contacts",
		Params:      []string{"source"},
	},

	// Calendar Skills
	"sync-calendar": {
		Name:        "sync-calendar",
		DisplayName: "Sync Calendar",
		Description: "Sync Google Calendar events to daily log",
		Icon:        "calendar",
		Category:    "sync",
		SkillID:     "calendar.sync_events",
		Params:      []string{"days_ahead"},
	},

	// Task Skills
	"import-tasks": {
		Name:        "import-tasks",
		DisplayName: "Import Tasks",
		Description: "Import tasks from Linear/external source with review",
		Icon:        "list-checks",
		Category:    "integration",
		SkillID:     "tasks.import_with_review",
		Params:      []string{"source"},
	},

	// Knowledge Skills
	"build-knowledge": {
		Name:        "build-knowledge",
		DisplayName: "Build Knowledge",
		Description: "Extract knowledge from conversations and build context nodes",
		Icon:        "brain",
		Category:    "automation",
		SkillID:     "knowledge.extract_and_build",
		Params:      []string{"source", "type"},
	},

	// Analysis Skills
	"analyze-pipeline": {
		Name:        "analyze-pipeline",
		DisplayName: "Analyze Pipeline",
		Description: "Analyze sales pipeline and generate insights",
		Icon:        "bar-chart-2",
		Category:    "automation",
		SkillID:     "analysis.pipeline",
		Params:      []string{},
	},
	"daily-brief": {
		Name:        "daily-brief",
		DisplayName: "Daily Brief",
		Description: "Generate a daily brief from calendar, tasks, and emails",
		Icon:        "newspaper",
		Category:    "automation",
		SkillID:     "daily.brief",
		Params:      []string{},
	},

	// Notification Skills
	"slack-notify": {
		Name:        "slack-notify",
		DisplayName: "Slack Notify",
		Description: "Send a notification to Slack channel",
		Icon:        "message-square",
		Category:    "integration",
		SkillID:     "slack.send_message",
		Params:      []string{"channel", "message"},
	},
}

// GetSkillCommand returns a skill command by name.
func GetSkillCommand(name string) (*SkillCommand, bool) {
	cmd, ok := BuiltInSkillCommands[name]
	if !ok {
		return nil, false
	}
	return &cmd, true
}

// ListSkillCommands returns all available skill commands.
func ListSkillCommands() []SkillCommand {
	commands := make([]SkillCommand, 0, len(BuiltInSkillCommands))
	for _, cmd := range BuiltInSkillCommands {
		commands = append(commands, cmd)
	}
	return commands
}

// GetSkillCommandsByCategory returns skill commands filtered by category.
func GetSkillCommandsByCategory(category string) []SkillCommand {
	commands := make([]SkillCommand, 0)
	for _, cmd := range BuiltInSkillCommands {
		if cmd.Category == category {
			commands = append(commands, cmd)
		}
	}
	return commands
}
