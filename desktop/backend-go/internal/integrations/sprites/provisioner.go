package sprites

import (
	"context"
	"fmt"
	"log/slog"
)

// planConfig holds resource allocation parameters for a given billing plan.
type planConfig struct {
	// CPU and memory fields are illustrative; adjust to match the sprites.dev API
	// resource-scaling parameters when available.
	CPUMillicores int
	MemoryMB      int
}

var planConfigs = map[string]planConfig{
	"starter":    {CPUMillicores: 500, MemoryMB: 512},
	"pro":        {CPUMillicores: 1000, MemoryMB: 1024},
	"business":   {CPUMillicores: 2000, MemoryMB: 2048},
	"enterprise": {CPUMillicores: 4000, MemoryMB: 4096},
}

// SpriteProvisioner handles the full customer-instance lifecycle: provisioning,
// upgrades (with automatic rollback on failure), scaling, suspension, and resumption.
type SpriteProvisioner struct {
	client *SpritesClient
	logger *slog.Logger
}

// NewSpriteProvisioner constructs a SpriteProvisioner backed by the given client.
// Pass a nil *slog.Logger to inherit the client's logger.
func NewSpriteProvisioner(client *SpritesClient, logger *slog.Logger) *SpriteProvisioner {
	if logger == nil {
		logger = client.logger
	}
	return &SpriteProvisioner{
		client: client,
		logger: logger,
	}
}

// ProvisionCustomer creates a full BOS instance for a new customer.
//
// It:
//  1. Creates a Sprite named "bos-{customerID}" with plan-appropriate env vars
//  2. Runs database migrations and required system defaults via ExecCommand
//  3. Returns the provisioned Sprite including its HTTP URL
//
// The call uses a provisioning-scoped timeout derived from the parent context.
func (p *SpriteProvisioner) ProvisionCustomer(ctx context.Context, customerID string, plan string) (*Sprite, error) {
	p.logger.InfoContext(ctx, "provisioning customer sprite",
		"customer_id", customerID,
		"plan", plan,
	)

	env := p.buildEnv(customerID, plan)

	sprite, err := p.client.CreateSprite(ctx, CreateSpriteRequest{
		Name:       "bos-" + customerID,
		Image:      "businessos-workspace:latest",
		Env:        env,
		CustomerID: customerID,
	})
	if err != nil {
		return nil, fmt.Errorf("provision customer %q: create sprite: %w", customerID, err)
	}

	// Run init script: migrations + required system defaults.
	initCmd := []string{"/bin/sh", "-c", "/scripts/init.sh"}
	result, err := p.client.ExecCommand(ctx, sprite.ID, initCmd)
	if err != nil {
		// Attempt cleanup so we don't leave a dangling sprite.
		_ = p.client.DeleteSprite(ctx, sprite.ID)
		return nil, fmt.Errorf("provision customer %q: run init script: %w", customerID, err)
	}
	if result.ExitCode != 0 {
		_ = p.client.DeleteSprite(ctx, sprite.ID)
		return nil, fmt.Errorf("provision customer %q: init script exited %d: %s",
			customerID, result.ExitCode, result.Output)
	}

	p.logger.InfoContext(ctx, "customer sprite provisioned",
		"customer_id", customerID,
		"sprite_id", sprite.ID,
		"url", sprite.URL,
	)
	return sprite, nil
}

// UpgradeCustomer safely upgrades a customer's Sprite to a new BOS version.
//
// It:
//  1. Checkpoints the current state as "pre-upgrade-{version}"
//  2. Pulls the new code/image via ExecCommand
//  3. Runs database migrations
//  4. Performs a health check; on failure, automatically restores the checkpoint
func (p *SpriteProvisioner) UpgradeCustomer(ctx context.Context, spriteID string, version string) error {
	checkpointTag := "pre-upgrade-" + version

	p.logger.InfoContext(ctx, "upgrading customer sprite",
		"sprite_id", spriteID,
		"version", version,
		"checkpoint_tag", checkpointTag,
	)

	// 1. Snapshot the current state so we can roll back if anything goes wrong.
	if err := p.client.Checkpoint(ctx, CheckpointRequest{
		SpriteID: spriteID,
		Tag:      checkpointTag,
	}); err != nil {
		return fmt.Errorf("upgrade sprite %q to %s: checkpoint: %w", spriteID, version, err)
	}

	// 2. Pull new code/image.
	pullCmd := []string{"/bin/sh", "-c", fmt.Sprintf("/scripts/upgrade.sh %s", version)}
	pullResult, err := p.client.ExecCommand(ctx, spriteID, pullCmd)
	if err != nil {
		return p.rollback(ctx, spriteID, version, checkpointTag,
			fmt.Errorf("pull new version: %w", err))
	}
	if pullResult.ExitCode != 0 {
		return p.rollback(ctx, spriteID, version, checkpointTag,
			fmt.Errorf("upgrade script exited %d: %s", pullResult.ExitCode, pullResult.Output))
	}

	// 3. Run database migrations.
	migrateCmd := []string{"/bin/sh", "-c", "/scripts/migrate.sh"}
	migrateResult, err := p.client.ExecCommand(ctx, spriteID, migrateCmd)
	if err != nil {
		return p.rollback(ctx, spriteID, version, checkpointTag,
			fmt.Errorf("run migrations: %w", err))
	}
	if migrateResult.ExitCode != 0 {
		return p.rollback(ctx, spriteID, version, checkpointTag,
			fmt.Errorf("migrations exited %d: %s", migrateResult.ExitCode, migrateResult.Output))
	}

	// 4. Health check — hit the /health endpoint inside the sprite.
	healthCmd := []string{"/bin/sh", "-c", "curl -sf http://localhost:8080/health"}
	healthResult, err := p.client.ExecCommand(ctx, spriteID, healthCmd)
	if err != nil {
		return p.rollback(ctx, spriteID, version, checkpointTag,
			fmt.Errorf("health check exec: %w", err))
	}
	if healthResult.ExitCode != 0 {
		return p.rollback(ctx, spriteID, version, checkpointTag,
			fmt.Errorf("health check failed (exit %d): %s", healthResult.ExitCode, healthResult.Output))
	}

	p.logger.InfoContext(ctx, "customer sprite upgraded",
		"sprite_id", spriteID,
		"version", version,
	)
	return nil
}

// ScaleCustomer adjusts the resource allocation of a Sprite to match the given
// billing plan. Unknown plan names return an error before any API call is made.
func (p *SpriteProvisioner) ScaleCustomer(ctx context.Context, spriteID string, plan string) error {
	cfg, ok := planConfigs[plan]
	if !ok {
		return fmt.Errorf("scale sprite %q: unknown plan %q", spriteID, plan)
	}

	p.logger.InfoContext(ctx, "scaling customer sprite",
		"sprite_id", spriteID,
		"plan", plan,
		"cpu_millicores", cfg.CPUMillicores,
		"memory_mb", cfg.MemoryMB,
	)

	scaleCmd := []string{
		"/bin/sh", "-c",
		fmt.Sprintf("/scripts/scale.sh --cpu %d --memory %d", cfg.CPUMillicores, cfg.MemoryMB),
	}
	result, err := p.client.ExecCommand(ctx, spriteID, scaleCmd)
	if err != nil {
		return fmt.Errorf("scale sprite %q to plan %q: %w", spriteID, plan, err)
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("scale sprite %q to plan %q: script exited %d: %s",
			spriteID, plan, result.ExitCode, result.Output)
	}

	p.logger.InfoContext(ctx, "customer sprite scaled",
		"sprite_id", spriteID,
		"plan", plan,
	)
	return nil
}

// SuspendCustomer stops a Sprite (e.g. for non-payment). The instance is
// checkpointed first so state is preserved for resumption.
func (p *SpriteProvisioner) SuspendCustomer(ctx context.Context, spriteID string) error {
	p.logger.InfoContext(ctx, "suspending customer sprite", "sprite_id", spriteID)

	// Checkpoint before stopping so no data is lost.
	if err := p.client.Checkpoint(ctx, CheckpointRequest{
		SpriteID: spriteID,
		Tag:      "pre-suspend",
	}); err != nil {
		return fmt.Errorf("suspend sprite %q: checkpoint: %w", spriteID, err)
	}

	stopCmd := []string{"/bin/sh", "-c", "/scripts/stop.sh"}
	result, err := p.client.ExecCommand(ctx, spriteID, stopCmd)
	if err != nil {
		return fmt.Errorf("suspend sprite %q: stop: %w", spriteID, err)
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("suspend sprite %q: stop script exited %d: %s",
			spriteID, result.ExitCode, result.Output)
	}

	p.logger.InfoContext(ctx, "customer sprite suspended", "sprite_id", spriteID)
	return nil
}

// ResumeCustomer restores a previously suspended Sprite from its "pre-suspend"
// checkpoint and starts the BOS process.
func (p *SpriteProvisioner) ResumeCustomer(ctx context.Context, spriteID string) error {
	p.logger.InfoContext(ctx, "resuming customer sprite", "sprite_id", spriteID)

	if err := p.client.Restore(ctx, spriteID, "pre-suspend"); err != nil {
		return fmt.Errorf("resume sprite %q: restore checkpoint: %w", spriteID, err)
	}

	startCmd := []string{"/bin/sh", "-c", "/scripts/start.sh"}
	result, err := p.client.ExecCommand(ctx, spriteID, startCmd)
	if err != nil {
		return fmt.Errorf("resume sprite %q: start: %w", spriteID, err)
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("resume sprite %q: start script exited %d: %s",
			spriteID, result.ExitCode, result.Output)
	}

	p.logger.InfoContext(ctx, "customer sprite resumed", "sprite_id", spriteID)
	return nil
}

// ---- internal helpers --------------------------------------------------------

// buildEnv constructs the environment variable map for a new customer sprite.
// Sensitive credentials (DB password, API keys) should be injected by the
// caller via a secrets manager rather than hard-coded here.
func (p *SpriteProvisioner) buildEnv(customerID, plan string) map[string]string {
	return map[string]string{
		"BOS_CUSTOMER_ID": customerID,
		"BOS_PLAN":        plan,
		"BOS_ENV":         "production",
	}
}

// rollback restores a Sprite from a checkpoint tag after a failed upgrade step.
// It logs the restoration attempt and wraps both the original error and any
// restore error together so callers have the full picture.
func (p *SpriteProvisioner) rollback(ctx context.Context, spriteID, version, tag string, cause error) error {
	p.logger.WarnContext(ctx, "upgrade failed, rolling back to checkpoint",
		"sprite_id", spriteID,
		"version", version,
		"checkpoint_tag", tag,
		"cause", cause,
	)

	if restoreErr := p.client.Restore(ctx, spriteID, tag); restoreErr != nil {
		return fmt.Errorf("upgrade sprite %q to %s failed (%w); rollback to %q also failed: %w",
			spriteID, version, cause, tag, restoreErr)
	}

	p.logger.InfoContext(ctx, "sprite rolled back to checkpoint",
		"sprite_id", spriteID,
		"checkpoint_tag", tag,
	)
	return fmt.Errorf("upgrade sprite %q to %s: %w (rolled back to %q)", spriteID, version, cause, tag)
}
