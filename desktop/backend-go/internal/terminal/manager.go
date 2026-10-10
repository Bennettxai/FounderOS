package terminal

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rhl/businessos-backend/internal/container"
)

// Manager handles terminal session lifecycle
type Manager struct {
	sessions       map[string]*Session
	mu             sync.RWMutex
	maxSessions    int
	cleanupTicker  *time.Ticker
	stopCleanup    chan struct{}
	containerMgr   *container.ContainerManager
	useContainers  bool
	securityConfig *SessionSecurityConfig
	pubsub         *TerminalPubSub // Optional pub/sub for horizontal scaling
}

// NewManager creates a new terminal session manager
// If containerMgr is provided, sessions will use Docker containers instead of local PTY
func NewManager(containerMgr *container.ContainerManager) *Manager {
	securityConfig := DefaultSessionSecurityConfig()

	m := &Manager{
		sessions:       make(map[string]*Session),
		maxSessions:    100,
		stopCleanup:    make(chan struct{}),
		containerMgr:   containerMgr,
		useContainers:  containerMgr != nil,
		securityConfig: securityConfig,
	}

	if m.useContainers {
		slog.Info("[Terminal] Manager initialized with Docker container support")
	} else {
		slog.Info("[Terminal] Manager initialized with local PTY support")
	}

	slog.Info("[Terminal] Security config",
		"max_duration", securityConfig.MaxSessionDuration,
		"idle_timeout", securityConfig.IdleTimeout,
		"ip_binding", securityConfig.EnableIPBinding)

	// Start cleanup goroutine
	m.startCleanup()

	return m
}

// SetPubSub configures optional pub/sub for horizontal scaling
func (m *Manager) SetPubSub(pubsub *TerminalPubSub) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pubsub = pubsub
	if pubsub != nil {
		slog.Info("[Terminal] Pub/sub enabled for horizontal scaling", "instance", pubsub.InstanceID())
	}
}

// GetPubSub returns the pub/sub manager (may be nil)
func (m *Manager) GetPubSub() *TerminalPubSub {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.pubsub
}

// GetSecurityConfig returns the current security configuration
func (m *Manager) GetSecurityConfig() *SessionSecurityConfig {
	return m.securityConfig
}

// UpdateSecurityConfig updates the security configuration
func (m *Manager) UpdateSecurityConfig(config *SessionSecurityConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.securityConfig = config
}

// CreateSession creates a new terminal session
// clientIP is optional but recommended for session hijacking protection
func (m *Manager) CreateSession(userID string, cols, rows int, shell, workingDir, environmentMode string, clientIP ...string) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check user session limit (max 5 per user)
	userSessions := m.getUserSessionCount(userID)
	if userSessions >= 5 {
		return nil, fmt.Errorf("session limit reached (max 5 per user)")
	}

	// Check global session limit
	if len(m.sessions) >= m.maxSessions {
		return nil, fmt.Errorf("maximum global session limit reached")
	}

	// Determine working directory
	if workingDir == "" {
		if m.useContainers {
			workingDir = "/workspace"
		} else {
			workingDir = getDefaultWorkingDir()
		}
	}

	// Determine environment mode
	if environmentMode == "" {
		if m.useContainers {
			environmentMode = "sandbox"
		} else {
			environmentMode = "local"
		}
	}

	// Calculate session expiration
	now := time.Now()
	var expiresAt time.Time
	if m.securityConfig.MaxSessionDuration > 0 {
		expiresAt = now.Add(m.securityConfig.MaxSessionDuration)
	}

	// Extract client IP for binding
	var ip, subnet string
	if len(clientIP) > 0 && clientIP[0] != "" {
		ip = clientIP[0]
		subnet = extractSubnet(ip)
	}

	// Create session with security fields.
	// WorkspaceID defaults to UserID so that OSA commands executed inside a
	// terminal session are always attributed to a real user, not a random UUID.
	// Callers that have an explicit workspace concept should set WorkspaceID
	// after creation before invoking any OSA operations.
	session := &Session{
		ID:              uuid.New().String(),
		UserID:          userID,
		WorkspaceID:     userID,
		CreatedAt:       now,
		LastActivity:    now,
		Cols:            cols,
		Rows:            rows,
		Shell:           shell,
		WorkingDir:      workingDir,
		Status:          StatusActive,
		EnvironmentMode: environmentMode,
		Environment:     m.buildEnvironment(userID, environmentMode),
		ClientIP:        ip,
		ClientSubnet:    subnet,
		ExpiresAt:       expiresAt,
	}

	// Start container or PTY based on configuration
	if m.useContainers && m.containerMgr != nil {
		if err := m.startContainer(session); err != nil {
			return nil, fmt.Errorf("failed to start container: %w", err)
		}
	} else {
		if err := startPTY(session); err != nil {
			return nil, fmt.Errorf("failed to start PTY: %w", err)
		}
	}

	// Store session
	m.sessions[session.ID] = session

	// Publish session created event for horizontal scaling
	if m.pubsub != nil {
		ctx := context.Background()
		if err := m.pubsub.PublishSessionEvent(ctx, "session_created", session.ID, session.UserID); err != nil {
			slog.Warn("[Terminal] Failed to publish session_created event", "error", err)
		}
	}

	return session, nil
}

// GetSession retrieves a session by ID (without ownership validation)
// WARNING: Use GetSessionSecure for user-facing APIs to prevent unauthorized access
func (m *Manager) GetSession(sessionID string) (*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found")
	}

	return session, nil
}

// GetSessionSecure retrieves a session with full security validation
// Checks: ownership, expiration, and optionally IP binding
func (m *Manager) GetSessionSecure(sessionID, userID, clientIP string) (*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found")
	}

	// Validate ownership
	if session.UserID != userID {
		slog.Warn("[Security] Session ownership mismatch",
			"session_id", sessionID[:8], "expected_user", session.UserID[:8], "got_user", userID[:8])
		return nil, fmt.Errorf("session access denied")
	}

	// Check expiration
	if session.IsExpired() {
		slog.Warn("[Security] Session has expired", "session_id", sessionID[:8])
		return nil, fmt.Errorf("session expired")
	}

	// Check idle timeout
	if session.IsIdle(m.securityConfig.IdleTimeout) {
		slog.Warn("[Security] Session is idle", "session_id", sessionID[:8], "idle_timeout", m.securityConfig.IdleTimeout)
		return nil, fmt.Errorf("session timed out due to inactivity")
	}

	// Validate IP binding
	if clientIP != "" {
		valid, reason := session.ValidateIP(clientIP, m.securityConfig)
		if !valid {
			slog.Warn("[Security] Session IP validation failed",
				"session_id", sessionID[:8], "reason", reason,
				"expected_ip", session.ClientIP, "got_ip", clientIP)
			return nil, fmt.Errorf("session security violation: %s", reason)
		}
	}

	return session, nil
}

// ValidateSessionAccess performs security validation without retrieving the full session
// Returns (valid bool, reason string)
func (m *Manager) ValidateSessionAccess(sessionID, userID, clientIP string) (bool, string) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return false, "session not found"
	}

	if session.UserID != userID {
		return false, "unauthorized access"
	}

	if session.IsExpired() {
		return false, "session expired"
	}

	if session.IsIdle(m.securityConfig.IdleTimeout) {
		return false, "session timed out"
	}

	if clientIP != "" {
		valid, reason := session.ValidateIP(clientIP, m.securityConfig)
		if !valid {
			return false, reason
		}
	}

	return true, ""
}

// GetUserSessions retrieves all sessions for a user
func (m *Manager) GetUserSessions(userID string) []SessionInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var sessions []SessionInfo
	for _, session := range m.sessions {
		if session.UserID == userID {
			sessions = append(sessions, session.ToInfo())
		}
	}

	return sessions
}

// UpdateActivity updates the last activity time for a session
func (m *Manager) UpdateActivity(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if session, exists := m.sessions[sessionID]; exists {
		session.LastActivity = time.Now()
	}
}

// CloseSession closes a terminal session
func (m *Manager) CloseSession(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found")
	}

	// Publish session closed event before cleanup for horizontal scaling
	if m.pubsub != nil {
		ctx := context.Background()
		if err := m.pubsub.PublishSessionEvent(ctx, "session_closed", session.ID, session.UserID); err != nil {
			slog.Warn("[Terminal] Failed to publish session_closed event", "error", err)
		}
	}

	// Close container or PTY based on session type
	if session.IsContainerized() {
		m.closeContainer(session)
	} else {
		closePTY(session)
	}

	// Update status
	session.Status = StatusClosed

	// Remove from active sessions
	delete(m.sessions, sessionID)

	return nil
}

// ResizeSession resizes the terminal
func (m *Manager) ResizeSession(sessionID string, cols, rows int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found")
	}

	session.Cols = cols
	session.Rows = rows

	// Resize container exec or PTY based on session type
	var err error
	if session.IsContainerized() {
		if session.ExecID == "" {
			return fmt.Errorf("exec ID not set for containerized session")
		}
		err = m.containerMgr.ResizeExec(session.ExecID, uint(rows), uint(cols))
	} else {
		err = resizePTY(session, cols, rows)
	}

	// Publish resize event after successful resize for horizontal scaling
	if err == nil && m.pubsub != nil {
		ctx := context.Background()
		if pubErr := m.pubsub.PublishResize(ctx, sessionID, cols, rows); pubErr != nil {
			slog.Warn("[Terminal] Failed to publish resize event", "error", pubErr)
		}
	}

	return err
}

// Shutdown closes all sessions and stops cleanup
func (m *Manager) Shutdown() {
	close(m.stopCleanup)
	if m.cleanupTicker != nil {
		m.cleanupTicker.Stop()
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Close all sessions
	for sessionID, session := range m.sessions {
		if session.IsContainerized() {
			m.closeContainer(session)
		} else {
			closePTY(session)
		}
		delete(m.sessions, sessionID)
	}
}

// Helper functions

func (m *Manager) getUserSessionCount(userID string) int {
	count := 0
	for _, session := range m.sessions {
		if session.UserID == userID && session.Status == StatusActive {
			count++
		}
	}
	return count
}

func (m *Manager) buildEnvironment(userID, environmentMode string) map[string]string {
	env := make(map[string]string)
	env["TERM"] = "xterm-256color"
	env["LANG"] = "en_US.UTF-8"
	env["COLORTERM"] = "truecolor"

	// Set mode-aware prompts for both bash (PS1) and zsh (PROMPT).
	// xterm.js renders raw ANSI — bash PS1 needs \[ \] around non-printing chars,
	// but zsh uses %F{color} escapes via PROMPT.
	switch environmentMode {
	case "sandbox":
		env["PS1"] = "\\[\033[1;33m\\][sandbox]\\[\033[0m\\] \\[\033[1;36m\\]\\w\\[\033[0m\\] \\$ "
		env["PROMPT"] = "%F{yellow}%B[sandbox]%b%f %F{cyan}%~%f %# "
	case "production":
		env["PS1"] = "\\[\033[1;31m\\][prod]\\[\033[0m\\] \\[\033[1;36m\\]\\w\\[\033[0m\\] \\$ "
		env["PROMPT"] = "%F{red}%B[prod]%b%f %F{cyan}%~%f %# "
	default: // "local"
		env["PS1"] = "\\[\033[1;35m\\][local]\\[\033[0m\\] \\[\033[1;36m\\]\\w\\[\033[0m\\] \\$ "
		env["PROMPT"] = "%F{magenta}%B[local]%b%f %F{cyan}%~%f %# "
	}

	env["BUSINESSOS_ENV_MODE"] = environmentMode
	env["BUSINESSOS_OS"] = runtime.GOOS
	env["BUSINESSOS_USER_ID"] = userID // Pass user ID to container for internal API calls
	return env
}

func (m *Manager) startCleanup() {
	m.cleanupTicker = time.NewTicker(5 * time.Minute)
	go func() {
		for {
			select {
			case <-m.cleanupTicker.C:
				m.cleanupIdleSessions()
			case <-m.stopCleanup:
				return
			}
		}
	}()
}

func (m *Manager) cleanupIdleSessions() {
	m.mu.Lock()
	defer m.mu.Unlock()

	var expiredCount, idleCount int

	for sessionID, session := range m.sessions {
		shouldClose := false
		reason := ""

		// Check for hard expiration
		if session.IsExpired() {
			shouldClose = true
			reason = "expired"
			expiredCount++
		} else if session.IsIdle(m.securityConfig.IdleTimeout) {
			// Check for idle timeout
			shouldClose = true
			reason = "idle"
			idleCount++
		}

		if shouldClose {
			slog.Info("[Terminal] Closing session", "session_id", sessionID[:8], "reason", reason)
			if session.IsContainerized() {
				m.closeContainer(session)
			} else {
				closePTY(session)
			}
			delete(m.sessions, sessionID)
		}
	}

	if expiredCount > 0 || idleCount > 0 {
		slog.Info("[Terminal] Cleanup completed", "expired", expiredCount, "idle", idleCount)
	}
}

// startContainer creates and starts a Docker container for the session
func (m *Manager) startContainer(session *Session) error {
	slog.Info("[Terminal] Starting container for session", "session_id", session.ID, "user_id", session.UserID)

	// Create volume for user workspace
	volumeName, err := m.containerMgr.CreateVolume(session.UserID)
	if err != nil {
		return fmt.Errorf("failed to create volume: %w", err)
	}
	slog.Info("[Terminal] Volume created/verified", "volume", volumeName)

	// Create container with the default image (include session ID for unique naming)
	containerID, err := m.containerMgr.CreateContainer(session.UserID, session.ID, m.containerMgr.GetDefaultImage())
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}
	slog.Info("[Terminal] Container created", "container_id", containerID[:12])

	// Start container
	if err := m.containerMgr.StartContainer(containerID); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}
	slog.Info("[Terminal] Container started", "container_id", containerID[:12])

	// Determine shell command - load BusinessOS init script for osa command
	// The init script is bind-mounted at /etc/businessos/init.sh
	initScript := "/etc/businessos/init.sh"

	// Force bash in containers for reliable --rcfile support
	// zsh in containers has complex rc loading; bash is simpler and universal
	shell := "/bin/bash"
	shellCmd := []string{shell, "--rcfile", initScript}

	// Create and start exec session with shell and environment variables
	execID, hijacked, err := m.containerMgr.StartExecWithEnv(containerID, shellCmd, true, session.Environment)
	if err != nil {
		// If exec fails, try to stop the container
		m.containerMgr.StopContainer(containerID, 10)
		return fmt.Errorf("failed to start exec: %w", err)
	}
	slog.Info("[Terminal] Exec started", "exec_id", execID, "env", session.Environment)

	// Store container information in session
	session.ContainerID = containerID
	session.VolumeID = volumeName
	session.ExecID = execID
	session.ExecConn = &hijacked

	// Set initial terminal size
	if err := m.containerMgr.ResizeExec(execID, uint(session.Rows), uint(session.Cols)); err != nil {
		slog.Warn("[Terminal] Failed to set initial terminal size", "error", err)
	}

	slog.Info("[Terminal] Container session ready",
		"container_id", containerID[:12], "exec_id", execID[:12])
	return nil
}

// closeContainer closes the container session and removes the container
func (m *Manager) closeContainer(session *Session) {
	slog.Info("[Terminal] Closing container session", "session_id", session.ID)

	// Close exec connection first
	if session.ExecConn != nil {
		session.ExecConn.Close()
		slog.Info("[Terminal] Exec connection closed", "session_id", session.ID)
	}

	// Stop and remove container for immediate cleanup
	// The ContainerMonitor acts as a safety net for any missed containers
	if session.ContainerID != "" {
		containerID := session.ContainerID[:12]

		// Stop container first (with graceful timeout)
		if err := m.containerMgr.StopContainer(session.ContainerID, 5); err != nil {
			slog.Warn("[Terminal] Failed to stop container", "container_id", containerID, "error", err)
		} else {
			slog.Info("[Terminal] Container stopped", "container_id", containerID)
		}

		// Remove container immediately (force=true handles any remaining state)
		if err := m.containerMgr.RemoveContainer(session.ContainerID, true); err != nil {
			// Not critical - ContainerMonitor will clean up orphaned containers
			slog.Warn("[Terminal] Failed to remove container (will be cleaned by monitor)",
				"container_id", containerID, "error", err)
		} else {
			slog.Info("[Terminal] Container removed", "container_id", containerID)
		}
	}

	slog.Info("[Terminal] Container session closed", "session_id", session.ID)
}

// SandboxChanges holds the git diff summary for a sandbox session
type SandboxChanges struct {
	FileCount  int      `json:"file_count"`
	Files      []string `json:"files"`
	Insertions int      `json:"insertions"`
	Deletions  int      `json:"deletions"`
	Summary    string   `json:"summary"`
}

// GetSandboxChanges returns git change summary for a sandbox session
func (m *Manager) GetSandboxChanges(sessionID string) (*SandboxChanges, error) {
	m.mu.RLock()
	session, exists := m.sessions[sessionID]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("session not found")
	}

	if session.EnvironmentMode != "sandbox" {
		return nil, fmt.Errorf("session is not in sandbox mode")
	}

	// Return empty changes for now - actual git integration would
	// exec git commands inside the container/PTY
	return &SandboxChanges{
		FileCount:  0,
		Files:      []string{},
		Insertions: 0,
		Deletions:  0,
		Summary:    "No changes tracked",
	}, nil
}
