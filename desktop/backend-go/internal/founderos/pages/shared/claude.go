package shared

import (
	"context"
	"os"
	"os/exec"
)

// ClaudeCommand builds a `claude -p` call that can only produce text: no
// built-in tools (--tools ""), no MCP servers (--strict-mcp-config with none
// configured), run from a fresh empty directory. Prompts on the bridge carry
// third-party text, so the model must not be able to act (security review #2).
// Call cleanup when the command is done.
func ClaudeCommand(ctx context.Context, bin, prompt string) (*exec.Cmd, func(), error) {
	dir, err := os.MkdirTemp("", "founderos-claude-")
	if err != nil {
		return nil, func() {}, err
	}
	cmd := exec.CommandContext(ctx, bin, "-p", prompt, "--output-format", "json", "--tools", "", "--strict-mcp-config")
	cmd.Dir = dir
	return cmd, func() { _ = os.RemoveAll(dir) }, nil
}
