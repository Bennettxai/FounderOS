package terminal

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rhl/businessos-backend/internal/logging"
)

// AllowedOrigins contains the list of allowed origins for WebSocket connections
var AllowedOrigins = []string{
	"http://localhost:5173",      // Vite dev server
	"http://localhost:5273",      // BusinessOS local desktop runner
	"http://localhost:3000",      // Alternative dev server
	"https://localhost:5173",     // HTTPS dev server
	"https://app.businessos.com", // Production domain (update as needed)
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin:     checkWebSocketOrigin,
}

// MaxMessageSize sets the maximum message size for WebSocket connections
const MaxMessageSize = 16384 // 16KB

// checkWebSocketOrigin validates the origin of WebSocket connections
// This prevents Cross-Site WebSocket Hijacking (CSWSH) attacks
func checkWebSocketOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	maskedIP := logging.MaskIP(getClientIP(r))

	if origin == "" {
		// Log security event - missing origin header
		logging.Security("WebSocket connection denied - missing Origin header from %s", maskedIP)
		return false
	}

	// Parse origin URL
	originURL, err := url.Parse(origin)
	if err != nil {
		logging.Security("WebSocket connection denied - invalid Origin: %s from %s", origin, maskedIP)
		return false
	}

	if isAllowedLocalhostDevOrigin(originURL) {
		logging.Debug("WebSocket connection allowed from local dev origin: %s", origin)
		return true
	}

	// Check against allowed origins
	for _, allowedOrigin := range AllowedOrigins {
		allowedURL, err := url.Parse(allowedOrigin)
		if err != nil {
			continue // Skip malformed allowed origins
		}

		// Compare scheme, host, and port
		if originURL.Scheme == allowedURL.Scheme &&
			originURL.Host == allowedURL.Host {
			logging.Debug("WebSocket connection allowed from origin: %s", origin)
			return true
		}
	}

	// Log security event - unauthorized origin
	logging.Security("WebSocket connection denied - unauthorized origin: %s from %s", origin, maskedIP)
	return false
}

func isAllowedLocalhostDevOrigin(originURL *url.URL) bool {
	if originURL.Scheme != "http" && originURL.Scheme != "https" {
		return false
	}

	host := originURL.Hostname()
	if host != "localhost" && host != "127.0.0.1" {
		return false
	}

	port := originURL.Port()
	if port == "" {
		return false
	}

	parsedPort, err := strconv.Atoi(port)
	if err != nil {
		return false
	}

	return parsedPort >= 3000 && parsedPort <= 5999
}

// getClientIP extracts the client IP address from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (from proxies/load balancers)
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// Take the first IP in the chain
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header (from Nginx)
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	return r.RemoteAddr
}

// allowedShells is the set of shell binaries permitted for terminal sessions.
// Prevents arbitrary binary execution via the ?shell= WebSocket parameter.
var allowedShells = map[string]bool{
	"zsh":  true,
	"bash": true,
	"sh":   true,
	"fish": true,
}

// sanitizeShell validates the requested shell against the allowlist.
// Returns the shell name if allowed, empty string to fall back to auto-detect.
func sanitizeShell(requested string) string {
	if requested == "" {
		return ""
	}
	base := filepath.Base(requested)
	if allowedShells[base] {
		return base
	}
	logging.Security("Terminal shell rejected - not in allowlist: %s", requested)
	return ""
}

// sanitizeWorkingDir validates the requested working directory.
// Returns the path if it is safe, or empty string to fall back to home directory.
func sanitizeWorkingDir(requested string) string {
	if requested == "" {
		return ""
	}
	// Reject path traversal
	if strings.Contains(requested, "..") {
		logging.Security("Terminal cwd rejected - path traversal detected: %s", requested)
		return ""
	}
	// Must be absolute
	if !filepath.IsAbs(requested) {
		logging.Security("Terminal cwd rejected - relative path not allowed: %s", requested)
		return ""
	}
	return requested
}

// WebSocketHandler handles terminal WebSocket connections
type WebSocketHandler struct {
	manager *Manager
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(manager *Manager) *WebSocketHandler {
	return &WebSocketHandler{
		manager: manager,
	}
}

// HandleConnection handles a WebSocket terminal connection
func (h *WebSocketHandler) HandleConnection(w http.ResponseWriter, r *http.Request, userID string) {
	logging.Info("[Terminal] HandleConnection starting for user: %s", logging.MaskSessionID(userID))

	// Check connection limit before upgrading
	rateLimiter := GetRateLimiter()
	if !rateLimiter.AddConnection(userID) {
		HTTP429Handler(w, "Too many concurrent connections")
		return
	}

	// Upgrade to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logging.Error("[Terminal] Failed to upgrade WebSocket connection: %v", err)
		rateLimiter.RemoveConnection(userID)
		return
	}

	// Set message size limit (prevents DoS via large frames)
	conn.SetReadLimit(MaxMessageSize)

	logging.Debug("[Terminal] WebSocket upgraded successfully")
	defer func() {
		conn.Close()
		rateLimiter.RemoveConnection(userID)
	}()

	// Parse query parameters for terminal configuration
	query := r.URL.Query()
	cols := parseIntParam(query.Get("cols"), 80)
	rows := parseIntParam(query.Get("rows"), 24)
	// Validate shell against allowlist and reject arbitrary binary paths.
	// Empty result means getShellPath will auto-detect (zsh on macOS, bash on Linux).
	shell := sanitizeShell(query.Get("shell"))
	// Validate cwd and reject relative paths and path traversal attempts.
	workingDir := sanitizeWorkingDir(query.Get("cwd"))
	environmentMode := query.Get("environment_mode")
	logging.Debug("[Terminal] Config: cols=%d, rows=%d, shell=%s", cols, rows, shell)

	// Get client IP for session binding (hijacking protection)
	clientIP := getClientIP(r)

	// Create terminal session with security binding
	session, err := h.manager.CreateSession(userID, cols, rows, shell, workingDir, environmentMode, clientIP)
	if err != nil {
		logging.Error("[Terminal] CreateSession error: %v", err)
		h.sendError(conn, err.Error())
		return
	}
	maskedSessionID := logging.MaskSessionID(session.ID)
	maskedUserID := logging.MaskSessionID(userID)
	logging.Info("[Terminal] Session created: %s for user %s", maskedSessionID, maskedUserID)
	defer h.manager.CloseSession(session.ID)

	// Send connected status with session ID
	h.sendStatus(conn, "connected", map[string]interface{}{
		"session_id":    session.ID,
		"cols":          cols,
		"rows":          rows,
		"shell":         shell,
		"containerized": session.IsContainerized(),
	})

	// Send welcome banner via WebSocket (not PTY!)
	banner := GetWelcomeBanner()
	h.sendOutput(conn, session.ID, banner)

	// Start bi-directional streaming
	errChan := make(chan error, 2)

	// Read from WebSocket, write to PTY or Docker (user input)
	go h.handleInput(conn, session, errChan)

	// Read from PTY or Docker, write to WebSocket (terminal output)
	go h.handleOutput(conn, session, errChan)

	// Wait for error or disconnect
	err = <-errChan
	if err != nil && err != io.EOF {
		logging.Info("Terminal session %s ended: %v", logging.MaskSessionID(session.ID), err)
	}
}

// handleInput reads from WebSocket and writes to PTY or Docker container
func (h *WebSocketHandler) handleInput(conn *websocket.Conn, session *Session, errChan chan error) {
	// Get sanitizer and rate limiter instances (singleton, thread-safe)
	sanitizer := GetSanitizer()
	rateLimiter := GetRateLimiter()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			errChan <- err
			return
		}

		// Check message rate limit
		if !rateLimiter.AllowMessage(session.UserID) {
			logging.Security("Rate limit exceeded for user %s", logging.MaskSessionID(session.UserID))
			h.sendError(conn, "Rate limit exceeded - please slow down")
			continue // Don't terminate session, just skip this message
		}

		var msg TerminalMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			// Treat as raw input if not JSON - validate raw bytes
			inputStr := string(message)

			// Fast-path check first, then full validation if needed
			if !QuickValidate(inputStr) {
				result := sanitizer.ValidateInput(inputStr, session.UserID)
				if result.Blocked {
					// Send error to client but don't terminate session
					h.sendError(conn, "Input blocked: "+result.Reason)
					continue
				}
				// Use sanitized input
				message = []byte(result.Sanitized)
			}

			if session.IsContainerized() {
				// Write to Docker container exec connection
				_, writeErr := session.ExecConn.Conn.Write(message)
				if writeErr != nil {
					errChan <- writeErr
					return
				}
			} else {
				// Write to local PTY
				_, writeErr := WritePTY(session, message)
				if writeErr != nil {
					errChan <- writeErr
					return
				}
			}
			continue
		}

		switch msg.Type {
		case MsgTypeInput:
			// DEBUG: Log IMMEDIATELY when input arrives, before any processing
			hexBytes := make([]string, len(msg.Data))
			for i, b := range []byte(msg.Data) {
				hexBytes[i] = fmt.Sprintf("%02x", b)
			}
			logging.Info("[Terminal] 🔵 INPUT RECEIVED: %q (hex: %s)", msg.Data, strings.Join(hexBytes, " "))

			// Log if arrow keys are detected
			if strings.Contains(msg.Data, "\x1b[A") || strings.Contains(msg.Data, "\x1b[B") ||
				strings.Contains(msg.Data, "\x1b[C") || strings.Contains(msg.Data, "\x1b[D") {
				logging.Info("[Terminal] 🎯 ARROW KEY DETECTED in input")
			}

			// Validate and sanitize user input before execution
			inputData := msg.Data

			// Fast-path check first, then full validation if needed
			if !QuickValidate(inputData) {
				logging.Info("[Terminal] ⚠️  QuickValidate FAILED, running full sanitizer...")
				result := sanitizer.ValidateInput(inputData, session.UserID)
				if result.Blocked {
					logging.Info("[Terminal] ❌ Input BLOCKED: %s", result.Reason)
					// Send error to client but don't terminate session
					h.sendError(conn, "Input blocked: "+result.Reason)
					continue
				}
				logging.Info("[Terminal] ✅ Sanitizer PASSED")
				// Use sanitized input
				inputData = result.Sanitized
			} else {
				logging.Info("[Terminal] ✅ QuickValidate PASSED (no sanitizer needed)")
			}

			// After sanitization, verify arrow keys passed through
			if strings.Contains(msg.Data, "\x1b[A") || strings.Contains(msg.Data, "\x1b[B") ||
				strings.Contains(msg.Data, "\x1b[C") || strings.Contains(msg.Data, "\x1b[D") {
				if strings.Contains(inputData, "\x1b[A") || strings.Contains(inputData, "\x1b[B") ||
					strings.Contains(inputData, "\x1b[C") || strings.Contains(inputData, "\x1b[D") {
					logging.Info("[Terminal] ✅ ARROW KEY PASSED sanitizer")
				} else {
					logging.Warn("[Terminal] ⚠️  Escape sequence was STRIPPED by sanitizer")
				}
			}

			// DEBUG: Log what we're sending to PTY (use Info so it always shows)
			// Convert to hex for arrow keys
			hexBytes = make([]string, len(inputData))
			for i, b := range []byte(inputData) {
				hexBytes[i] = fmt.Sprintf("%02x", b)
			}
			logging.Info("[Terminal] 🔑 Sending to PTY: %q (hex: %s, len=%d)", inputData, strings.Join(hexBytes, " "), len(inputData))

			// Write validated input to PTY or Docker container
			if session.IsContainerized() {
				// Write to Docker container exec connection
				_, err := session.ExecConn.Conn.Write([]byte(inputData))
				if err != nil {
					errChan <- err
					return
				}
			} else {
				// Write to local PTY
				_, err := WritePTY(session, []byte(inputData))
				if err != nil {
					errChan <- err
					return
				}
			}
			h.manager.UpdateActivity(session.ID)

		case MsgTypeResize:
			// Handle terminal resize
			var resizeData ResizeData
			if err := json.Unmarshal([]byte(msg.Data), &resizeData); err == nil {
				h.manager.ResizeSession(session.ID, resizeData.Cols, resizeData.Rows)
			}

		case MsgTypeHeartbeat:
			// Update last activity
			h.manager.UpdateActivity(session.ID)
		}
	}
}

// handleOutput reads from PTY or Docker container and writes to WebSocket
func (h *WebSocketHandler) handleOutput(conn *websocket.Conn, session *Session, errChan chan error) {
	buffer := make([]byte, 4096)

	for {
		var n int
		var err error

		if session.IsContainerized() {
			// Read from Docker container exec connection
			n, err = session.ExecConn.Reader.Read(buffer)
		} else {
			// Read from local PTY
			n, err = ReadPTY(session, buffer)
		}

		if err != nil {
			errChan <- err
			return
		}

		if n > 0 {
			// Send output to client
			msg := TerminalMessage{
				Type:      MsgTypeOutput,
				SessionID: session.ID,
				Data:      string(buffer[:n]),
			}

			msgBytes, _ := json.Marshal(msg)
			if err := conn.WriteMessage(websocket.TextMessage, msgBytes); err != nil {
				errChan <- err
				return
			}
		}
	}
}

// sendError sends an error message to the client
func (h *WebSocketHandler) sendError(conn *websocket.Conn, message string) {
	msg := TerminalMessage{
		Type: MsgTypeError,
		Data: message,
	}
	msgBytes, _ := json.Marshal(msg)
	conn.WriteMessage(websocket.TextMessage, msgBytes)
}

// sendOutput sends output data to the client
func (h *WebSocketHandler) sendOutput(conn *websocket.Conn, sessionID string, data string) {
	msg := TerminalMessage{
		Type:      MsgTypeOutput,
		SessionID: sessionID,
		Data:      data,
	}
	msgBytes, _ := json.Marshal(msg)
	if err := conn.WriteMessage(websocket.TextMessage, msgBytes); err != nil {
		logging.Error("[Terminal] ERROR writing output message: %v", err)
	}
}

// sendStatus sends a status message to the client
func (h *WebSocketHandler) sendStatus(conn *websocket.Conn, status string, metadata map[string]interface{}) {
	msg := TerminalMessage{
		Type:     MsgTypeStatus,
		Data:     status,
		Metadata: logging.SafeLogFields(metadata),
	}
	msgBytes, _ := json.Marshal(msg)
	logging.Debug("[Terminal] Sending status: %s", status)
	if err := conn.WriteMessage(websocket.TextMessage, msgBytes); err != nil {
		logging.Error("[Terminal] ERROR writing status message: %v", err)
	}
}

// parseIntParam parses an integer from string with default value
func parseIntParam(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return val
}

// SetPongHandler sets up WebSocket keep-alive
func SetupKeepAlive(conn *websocket.Conn) {
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
}
