package terminal

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/creack/pty"
)

// startPTY creates a new PTY and starts the shell process
func startPTY(session *Session) error {
	slog.Info("[PTY] startPTY called")
	// Determine shell based on OS
	shell := getShellPath(session.Shell)
	slog.Info("[PTY] Using shell", "shell", shell)

	// Create command with BusinessOS init script
	var cmd *exec.Cmd
	if shell == "zsh" || shell == "/bin/zsh" {
		// Source BusinessOS init script in zsh
		// Try to find the init script relative to the executable or use env var
		initScript := os.Getenv("BUSINESSOS_INIT_SCRIPT")
		if initScript == "" {
			// Try relative path from working directory
			possiblePaths := []string{
				"./internal/terminal/businessos_init.sh",
				"internal/terminal/businessos_init.sh",
			}
			for _, p := range possiblePaths {
				if abs, err := filepath.Abs(p); err == nil {
					if _, err := os.Stat(abs); err == nil {
						initScript = abs
						break
					}
				}
			}
		}
		// Start zsh and source the init script (functions will persist in this shell)
		cmd = exec.Command(shell, "--no-rcs", "-c", fmt.Sprintf("source %s; exec zsh --no-rcs", initScript))
	} else {
		cmd = exec.Command(shell)
	}
	slog.Info("[PTY] Command created")

	// Set working directory
	if session.WorkingDir != "" {
		cmd.Dir = session.WorkingDir
	} else {
		cmd.Dir = getDefaultWorkingDir()
	}
	slog.Info("[PTY] Working dir", "dir", cmd.Dir)

	// Set environment variables
	cmd.Env = buildEnvArray(session.Environment)
	slog.Info("[PTY] Starting PTY")

	// Start with PTY
	ptmx, err := pty.Start(cmd)
	if err != nil {
		slog.Error("[PTY] pty.Start failed", "error", err)
		return fmt.Errorf("failed to start PTY: %w", err)
	}
	slog.Info("[PTY] PTY started successfully")

	// Set PTY size
	slog.Info("[PTY] Setting PTY size", "cols", session.Cols, "rows", session.Rows)
	if err := pty.Setsize(ptmx, &pty.Winsize{
		Rows: uint16(session.Rows),
		Cols: uint16(session.Cols),
	}); err != nil {
		ptmx.Close()
		return fmt.Errorf("failed to set PTY size: %w", err)
	}
	slog.Info("[PTY] PTY size set")

	// Store in session
	session.PTY = ptmx
	session.Cmd = cmd
	slog.Info("[PTY] Session PTY assigned")

	// NOTE: Welcome banner is now sent via WebSocket, not PTY
	// Writing to PTY master would go to shell stdin, not output!
	slog.Info("[PTY] PTY ready (banner sent via WebSocket)")

	slog.Info("[PTY] startPTY complete")
	return nil
}

// closePTY closes the PTY and kills the process
func closePTY(session *Session) {
	if session.PTY != nil {
		session.PTY.Close()
	}

	if session.Cmd != nil && session.Cmd.Process != nil {
		// Send SIGTERM first
		session.Cmd.Process.Signal(syscall.SIGTERM)

		// Wait briefly for graceful shutdown
		done := make(chan error, 1)
		go func() {
			done <- session.Cmd.Wait()
		}()

		select {
		case <-done:
			// Process exited gracefully
		case <-time.After(2 * time.Second):
			// Force kill if still running
			session.Cmd.Process.Kill()
		}
	}
}

// resizePTY resizes the PTY
func resizePTY(session *Session, cols, rows int) error {
	if session.PTY == nil {
		return fmt.Errorf("PTY not initialized")
	}

	return pty.Setsize(session.PTY, &pty.Winsize{
		Rows: uint16(rows),
		Cols: uint16(cols),
	})
}

// ReadPTY reads from PTY (output from shell)
func ReadPTY(session *Session, buffer []byte) (int, error) {
	if session.PTY == nil {
		return 0, fmt.Errorf("PTY not initialized")
	}
	return session.PTY.Read(buffer)
}

// WritePTY writes to PTY (input to shell)
func WritePTY(session *Session, data []byte) (int, error) {
	if session.PTY == nil {
		return 0, fmt.Errorf("PTY not initialized")
	}
	return session.PTY.Write(data)
}

// getShellPath determines the appropriate shell path
func getShellPath(preferredShell string) string {
	switch runtime.GOOS {
	case "windows":
		if preferredShell == "powershell" {
			return "powershell.exe"
		}
		return "cmd.exe"
	case "darwin", "linux":
		if preferredShell != "" {
			return preferredShell
		}
		// Try zsh first (macOS default), then bash
		if _, err := exec.LookPath("zsh"); err == nil {
			return "zsh"
		}
		return "bash"
	default:
		return "sh"
	}
}

// getDefaultWorkingDir returns the user's home directory
func getDefaultWorkingDir() string {
	if home, err := os.UserHomeDir(); err == nil {
		return home
	}
	return "/"
}

// safeEnvVars is the allowlist of environment variables that may be inherited
// by terminal sessions. Server secrets (DATABASE_URL, API keys, crypto keys)
// are explicitly excluded to prevent credential leakage via the `env` command.
var safeEnvVars = map[string]bool{
	"HOME":            true,
	"USER":            true,
	"SHELL":           true,
	"TERM":            true,
	"LANG":            true,
	"LC_ALL":          true,
	"LC_CTYPE":        true,
	"COLORTERM":       true,
	"EDITOR":          true,
	"VISUAL":          true,
	"PAGER":           true,
	"LESS":            true,
	"PATH":            true,
	"TMPDIR":          true,
	"XDG_RUNTIME_DIR": true,
}

// buildEnvArray converts environment map to array format.
// Only inherits variables from the safeEnvVars allowlist so that server secrets
// (DATABASE_URL, API keys, crypto keys) are never visible inside PTY sessions.
func buildEnvArray(envMap map[string]string) []string {
	var env []string
	// Only inherit safe environment variables — never leak server secrets.
	for _, e := range os.Environ() {
		key, _, _ := strings.Cut(e, "=")
		if safeEnvVars[key] {
			env = append(env, e)
		}
	}
	// Add session-specific overrides.
	for key, value := range envMap {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}
	return env
}
