package sprites

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// RunCodeRequest describes a code-execution job for an ephemeral Sprite sandbox.
type RunCodeRequest struct {
	// Language is a hint used to select the execution command when Command is empty.
	// Supported values: "go", "python", "node", "bash".
	Language string

	// Files is a map of filename -> file content that will be written into the
	// sandbox before the execution command is run.
	Files map[string]string

	// Command is the shell command to execute. If empty, a default is derived
	// from Language. The command is executed as: /bin/sh -c "<Command>".
	Command string

	// Timeout caps the total execution time. Defaults to 30 s if zero.
	Timeout time.Duration

	// CustomerID is used for audit logging and Sprite naming; it is not sent
	// to the sprites.dev API.
	CustomerID string
}

// RunCodeResult holds the output of an ephemeral sandbox execution.
type RunCodeResult struct {
	Output   string
	ExitCode int
	Duration time.Duration
	Error    string
}

// defaultCommands maps language identifiers to their default execution commands.
var defaultCommands = map[string]string{
	"go":     "go run .",
	"python": "python3 main.py",
	"node":   "node index.js",
	"bash":   "bash main.sh",
}

// Sandbox provides ephemeral Sprite execution for OSA BUILD mode.
// Each RunCode call creates a fresh Sprite, executes the code, captures the
// output, and then destroys the Sprite — leaving no persistent state.
type Sandbox struct {
	client *SpritesClient
	logger *slog.Logger
}

// NewSandbox constructs a Sandbox backed by the given client.
// Pass a nil *slog.Logger to inherit the client's logger.
func NewSandbox(client *SpritesClient, logger *slog.Logger) *Sandbox {
	if logger == nil {
		logger = client.logger
	}
	return &Sandbox{
		client: client,
		logger: logger,
	}
}

// RunCode provisions an ephemeral Sprite, writes the supplied files, executes
// the command, captures stdout/stderr, cleans up the Sprite, and returns the
// result. The Sprite is always deleted regardless of execution outcome.
func (s *Sandbox) RunCode(ctx context.Context, req RunCodeRequest) (*RunCodeResult, error) {
	if req.Timeout == 0 {
		req.Timeout = defaultTimeout
	}

	execCtx, cancel := context.WithTimeout(ctx, req.Timeout)
	defer cancel()

	cmd, err := s.resolveCommand(req)
	if err != nil {
		return nil, fmt.Errorf("run code: %w", err)
	}

	s.logger.InfoContext(ctx, "creating sandbox sprite",
		"language", req.Language,
		"customer_id", req.CustomerID,
		"file_count", len(req.Files),
	)

	sprite, err := s.client.CreateSprite(execCtx, CreateSpriteRequest{
		Name:       s.sandboxName(req.CustomerID),
		CustomerID: req.CustomerID,
		// Ephemeral sandboxes use the default workspace image.
	})
	if err != nil {
		return nil, fmt.Errorf("run code: create sandbox sprite: %w", err)
	}

	// Always clean up the ephemeral sprite, even if subsequent steps fail.
	defer func() {
		if delErr := s.client.DeleteSprite(ctx, sprite.ID); delErr != nil {
			s.logger.WarnContext(ctx, "failed to delete sandbox sprite",
				"sprite_id", sprite.ID,
				"error", delErr,
			)
		}
	}()

	// Write files into the sandbox.
	if err := s.writeFiles(execCtx, sprite.ID, req.Files); err != nil {
		return &RunCodeResult{
			ExitCode: -1,
			Error:    err.Error(),
		}, fmt.Errorf("run code: write files: %w", err)
	}

	// Execute the command and measure wall-clock duration.
	start := time.Now()
	result, err := s.client.ExecCommand(execCtx, sprite.ID, []string{"/bin/sh", "-c", cmd})
	elapsed := time.Since(start)

	if err != nil {
		return &RunCodeResult{
			ExitCode: -1,
			Duration: elapsed,
			Error:    err.Error(),
		}, fmt.Errorf("run code: exec command: %w", err)
	}

	s.logger.InfoContext(ctx, "sandbox execution complete",
		"sprite_id", sprite.ID,
		"exit_code", result.ExitCode,
		"duration_ms", elapsed.Milliseconds(),
	)

	return &RunCodeResult{
		Output:   result.Output,
		ExitCode: result.ExitCode,
		Duration: elapsed,
		Error:    result.Error,
	}, nil
}

// ---- internal helpers --------------------------------------------------------

// resolveCommand returns the command to run. If req.Command is set, it is used
// directly. Otherwise the language is looked up in defaultCommands.
func (s *Sandbox) resolveCommand(req RunCodeRequest) (string, error) {
	if req.Command != "" {
		return req.Command, nil
	}
	if req.Language == "" {
		return "", fmt.Errorf("either Command or Language must be specified")
	}
	cmd, ok := defaultCommands[req.Language]
	if !ok {
		return "", fmt.Errorf("unsupported language %q (supported: go, python, node, bash)", req.Language)
	}
	return cmd, nil
}

// sandboxName generates a short, unique-ish name for an ephemeral sprite.
// The sprites.dev API requires names to be unique within an account, so we
// append a nanosecond timestamp.
func (s *Sandbox) sandboxName(customerID string) string {
	ts := time.Now().UnixNano()
	if customerID == "" {
		return fmt.Sprintf("sandbox-%d", ts)
	}
	return fmt.Sprintf("sandbox-%s-%d", customerID, ts)
}

// writeFiles writes each file from the map into the sandbox sprite using a
// here-doc shell command. Each file is created relative to /workspace.
func (s *Sandbox) writeFiles(ctx context.Context, spriteID string, files map[string]string) error {
	for name, content := range files {
		// Use printf to avoid issues with special characters in content.
		// For production use, a dedicated file-upload endpoint would be preferable.
		writeCmd := fmt.Sprintf(
			"mkdir -p /workspace/$(dirname %q) && printf '%%s' %q > /workspace/%s",
			name, content, name,
		)
		result, err := s.client.ExecCommand(ctx, spriteID, []string{"/bin/sh", "-c", writeCmd})
		if err != nil {
			return fmt.Errorf("write file %q: %w", name, err)
		}
		if result.ExitCode != 0 {
			return fmt.Errorf("write file %q: command exited %d: %s",
				name, result.ExitCode, result.Output)
		}
	}
	return nil
}
