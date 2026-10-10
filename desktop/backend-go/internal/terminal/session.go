package terminal

import (
	"os"
	"os/exec"
	"time"

	"github.com/docker/docker/api/types"
)

// SessionStatus represents the state of a terminal session
type SessionStatus string

const (
	StatusActive SessionStatus = "active"
	StatusIdle   SessionStatus = "idle"
	StatusClosed SessionStatus = "closed"
)

// SessionSecurityConfig holds security-related session settings
type SessionSecurityConfig struct {
	// MaxSessionDuration is the hard limit on session lifetime (0 = no limit)
	MaxSessionDuration time.Duration
	// IdleTimeout is how long a session can be idle before termination
	IdleTimeout time.Duration
	// EnableIPBinding validates client IP hasn't changed (prevents hijacking)
	EnableIPBinding bool
	// AllowIPMigration allows IP changes within the same subnet (for mobile)
	AllowIPMigration bool
}

// DefaultSessionSecurityConfig returns production-safe defaults
func DefaultSessionSecurityConfig() *SessionSecurityConfig {
	return &SessionSecurityConfig{
		MaxSessionDuration: 8 * time.Hour, // Hard limit: 8 hours max
		IdleTimeout:        30 * time.Minute,
		EnableIPBinding:    true,  // Detect session hijacking
		AllowIPMigration:   false, // Strict by default
	}
}

// Session represents an active terminal session
type Session struct {
	ID          string `json:"id"`
	UserID      string `json:"user_id"`
	WorkspaceID string `json:"workspace_id"` // OSA workspace ID; defaults to UserID when not explicitly set

	CreatedAt    time.Time         `json:"created_at"`
	LastActivity time.Time         `json:"last_activity"`
	Cols         int               `json:"cols"`
	Rows         int               `json:"rows"`
	Shell        string            `json:"shell"`
	WorkingDir   string            `json:"working_dir"`
	Environment  map[string]string `json:"-"`
	Status       SessionStatus     `json:"status"`

	// Environment mode
	EnvironmentMode string `json:"environment_mode"` // "sandbox" | "production" | "local"

	// Agent process tracking
	AgentProcess string `json:"-"` // "claude" | "codex" | ""
	AgentPID     int    `json:"-"` // PID of running agent CLI process

	// Security fields
	ClientIP     string    `json:"-"` // Original client IP for hijacking detection
	ClientSubnet string    `json:"-"` // Client subnet (first 3 octets for IPv4)
	ExpiresAt    time.Time `json:"-"` // Hard expiration time

	// Local PTY mode (backwards compatibility - unused in container mode)
	PTY *os.File  `json:"-"`
	Cmd *exec.Cmd `json:"-"`

	// Container isolation mode
	ContainerID string                  `json:"-"` // Docker container ID
	VolumeID    string                  `json:"-"` // Docker volume ID
	ExecID      string                  `json:"-"` // Docker exec instance ID
	ExecConn    *types.HijackedResponse `json:"-"` // Docker exec hijacked connection
}

// IsContainerized returns true if this session is running in a Docker container
func (s *Session) IsContainerized() bool {
	return s.ContainerID != ""
}

// IsExpired returns true if the session has exceeded its maximum duration
func (s *Session) IsExpired() bool {
	if s.ExpiresAt.IsZero() {
		return false // No expiration set
	}
	return time.Now().After(s.ExpiresAt)
}

// IsIdle returns true if the session has been idle too long
func (s *Session) IsIdle(timeout time.Duration) bool {
	return time.Now().Sub(s.LastActivity) > timeout
}

// ValidateIP checks if the provided IP matches the session's bound IP
// Returns (valid bool, reason string)
func (s *Session) ValidateIP(clientIP string, config *SessionSecurityConfig) (bool, string) {
	if !config.EnableIPBinding || s.ClientIP == "" {
		return true, ""
	}

	// Exact match
	if s.ClientIP == clientIP {
		return true, ""
	}

	// Allow subnet migration if enabled
	if config.AllowIPMigration && s.ClientSubnet != "" {
		clientSubnet := extractSubnet(clientIP)
		if clientSubnet == s.ClientSubnet {
			return true, ""
		}
	}

	return false, "IP address mismatch - possible session hijacking"
}

// extractSubnet extracts the subnet prefix from an IP address
// For IPv4: returns first 3 octets (e.g., "192.168.1" from "192.168.1.100")
// For IPv6: returns first 4 groups
func extractSubnet(ip string) string {
	// Handle IPv4
	parts := splitIP(ip)
	if len(parts) == 4 {
		return parts[0] + "." + parts[1] + "." + parts[2]
	}

	// Handle IPv6 - take first half
	if len(ip) > 19 {
		return ip[:19]
	}

	return ip
}

// splitIP splits an IP address string by dots
func splitIP(ip string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(ip); i++ {
		if ip[i] == '.' {
			parts = append(parts, ip[start:i])
			start = i + 1
		}
	}
	if start < len(ip) {
		parts = append(parts, ip[start:])
	}
	return parts
}

// Close cleans up session resources
func (s *Session) Close() error {
	if s.IsContainerized() {
		// Close the hijacked connection if present
		if s.ExecConn != nil && s.ExecConn.Conn != nil {
			s.ExecConn.Close()
		}
	} else {
		// Close local PTY if present
		if s.PTY != nil {
			s.PTY.Close()
		}
	}

	s.Status = StatusClosed
	return nil
}

// MessageType represents WebSocket message types
type MessageType string

const (
	MsgTypeInput     MessageType = "input"
	MsgTypeOutput    MessageType = "output"
	MsgTypeResize    MessageType = "resize"
	MsgTypeHeartbeat MessageType = "heartbeat"
	MsgTypeError     MessageType = "error"
	MsgTypeStatus    MessageType = "status"
)

// TerminalMessage represents a WebSocket message
type TerminalMessage struct {
	Type      MessageType            `json:"type"`
	SessionID string                 `json:"session_id,omitempty"`
	Data      string                 `json:"data,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// ResizeData for terminal resize events
type ResizeData struct {
	Cols int `json:"cols"`
	Rows int `json:"rows"`
}

// SessionInfo is the public session information
type SessionInfo struct {
	ID              string        `json:"id"`
	CreatedAt       time.Time     `json:"created_at"`
	LastActivity    time.Time     `json:"last_activity"`
	Cols            int           `json:"cols"`
	Rows            int           `json:"rows"`
	Shell           string        `json:"shell"`
	WorkingDir      string        `json:"working_dir"`
	Status          SessionStatus `json:"status"`
	EnvironmentMode string        `json:"environment_mode"`
}

// ToInfo converts a Session to public SessionInfo
func (s *Session) ToInfo() SessionInfo {
	return SessionInfo{
		ID:              s.ID,
		CreatedAt:       s.CreatedAt,
		LastActivity:    s.LastActivity,
		Cols:            s.Cols,
		Rows:            s.Rows,
		Shell:           s.Shell,
		WorkingDir:      s.WorkingDir,
		Status:          s.Status,
		EnvironmentMode: s.EnvironmentMode,
	}
}
