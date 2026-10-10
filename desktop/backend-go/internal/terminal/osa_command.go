package terminal

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/rhl/businessos-backend/internal/integrations/osa"
)

// OSACommand handles `osa` CLI commands in terminal
type OSACommand struct {
	client      *osa.ResilientClient
	userID      uuid.UUID
	workspaceID uuid.UUID
}

// NewOSACommand creates an OSA command handler bound to the authenticated
// terminal session. userIDStr and workspaceIDStr must be the real user/workspace
// UUIDs from the session — never random UUIDs.
func NewOSACommand(client *osa.ResilientClient, userIDStr, workspaceIDStr string) *OSACommand {
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		// userIDStr is not a UUID (e.g., opaque string IDs from some auth
		// providers). Derive a stable, deterministic UUID from the raw string
		// using UUID v5 with the DNS namespace so OSA always sees a valid UUID
		// that round-trips consistently for the same user.
		userID = uuid.NewSHA1(uuid.NameSpaceDNS, []byte(userIDStr))
	}

	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		workspaceID = uuid.NewSHA1(uuid.NameSpaceDNS, []byte(workspaceIDStr))
	}

	return &OSACommand{
		client:      client,
		userID:      userID,
		workspaceID: workspaceID,
	}
}

// Execute handles: osa <subcommand> [args]
func (c *OSACommand) Execute(ctx context.Context, args []string) (string, error) {
	if len(args) == 0 {
		return c.help(), nil
	}

	subcommand := args[0]
	switch subcommand {
	case "generate", "gen":
		return c.generate(ctx, args[1:])
	case "status":
		return c.status(ctx, args[1:])
	case "list":
		return c.list(ctx)
	case "health":
		return c.health(ctx)
	case "help", "-h", "--help":
		return c.help(), nil
	default:
		return "", fmt.Errorf("unknown subcommand: %s (try 'osa help')", subcommand)
	}
}

func (c *OSACommand) generate(ctx context.Context, args []string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("usage: osa generate <description>\nExample: osa generate \"Express.js todo app with SQLite\"")
	}

	description := strings.Join(args, " ")

	req := &osa.AppGenerationRequest{
		Name:        "Generated App",
		Description: description,
		Type:        "full-stack",
		UserID:      c.userID,
		WorkspaceID: c.workspaceID,
	}

	resp, err := c.client.GenerateApp(ctx, req)
	if err != nil {
		return "", fmt.Errorf("generation failed: %w", err)
	}

	output := fmt.Sprintf(`
🎯 App Generation Started
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
App ID:     %s
Workspace:  %s
Status:     %s

⏳ OSA-5 is running 21-agent workflow...
   Use 'osa status %s' to check progress
`, resp.AppID, resp.WorkspaceID, resp.Status, resp.AppID)

	return output, nil
}

func (c *OSACommand) status(ctx context.Context, args []string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("usage: osa status <app-id>")
	}

	appID := args[0]
	status, err := c.client.GetAppStatus(ctx, appID, c.userID)
	if err != nil {
		return "", err
	}

	output := fmt.Sprintf(`
📊 App Status
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
App ID:       %s
Status:       %s
Current Step: %s
Progress:     %.0f%%

%s
%s
`, status.AppID, status.Status, status.CurrentStep,
		status.Progress*100, status.Output, status.Error)

	return output, nil
}

func (c *OSACommand) list(ctx context.Context) (string, error) {
	// List recent apps (requires OSA API extension)
	return "📦 Recent Apps:\n(Not yet implemented - requires OSA API extension)", nil
}

func (c *OSACommand) health(ctx context.Context) (string, error) {
	health, err := c.client.HealthCheck(ctx)
	if err != nil {
		return fmt.Sprintf("❌ OSA-5 is DOWN: %v", err), err
	}
	return fmt.Sprintf("✅ OSA-5 is healthy\n   Status: %s\n   Version: %s", health.Status, health.Version), nil
}

func (c *OSACommand) help() string {
	return `
OSA CLI - Control the 21-Agent Orchestration System
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Commands:
  osa generate <description>  Generate a new application
  osa gen <description>        Alias for 'generate'
  osa status <app-id>          Check app generation status
  osa list                     List recent apps
  osa health                   Check OSA-5 health
  osa help                     Show this help

Examples:
  osa gen "Express.js REST API with authentication"
  osa gen "React dashboard with charts"
  osa status app-abc-123
  osa health

Documentation: https://docs.businessos.ai/osa-integration
`
}
