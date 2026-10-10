package handlers

import (
	"encoding/json"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rhl/businessos-backend/internal/middleware"
)

const desktopPresenceMaxMessageSize = 4096
const desktopPresenceMinCoordinate = -100000
const desktopPresenceMaxCoordinate = 100000

type desktopPresenceHandler struct {
	pool *pgxpool.Pool
	hub  *desktopPresenceHub
}

func newDesktopPresenceHandler(pool *pgxpool.Pool) *desktopPresenceHandler {
	return &desktopPresenceHandler{
		pool: pool,
		hub:  globalDesktopPresenceHub,
	}
}

type desktopPresenceMessage struct {
	Type           string  `json:"type"`
	X              float64 `json:"x,omitempty"`
	Y              float64 `json:"y,omitempty"`
	ViewportWidth  float64 `json:"viewport_width,omitempty"`
	ViewportHeight float64 `json:"viewport_height,omitempty"`
	ActiveModule   string  `json:"active_module,omitempty"`
	ActiveTitle    string  `json:"active_title,omitempty"`
}

type desktopPresenceEvent struct {
	Type           string  `json:"type"`
	ClientID       string  `json:"client_id"`
	UserID         string  `json:"user_id"`
	Name           string  `json:"name"`
	Color          string  `json:"color"`
	ActiveModule   string  `json:"active_module,omitempty"`
	ActiveTitle    string  `json:"active_title,omitempty"`
	WorkspaceID    string  `json:"workspace_id"`
	DesktopSpaceID string  `json:"desktop_space_id"`
	X              float64 `json:"x,omitempty"`
	Y              float64 `json:"y,omitempty"`
	ViewportWidth  float64 `json:"viewport_width,omitempty"`
	ViewportHeight float64 `json:"viewport_height,omitempty"`
	LastSeen       int64   `json:"last_seen"`
	Revision       string  `json:"revision,omitempty"`
	Action         string  `json:"action,omitempty"`
}

type desktopPresenceClient struct {
	id             string
	userID         string
	name           string
	color          string
	workspaceID    string
	desktopSpaceID string
	conn           *websocket.Conn
	writeMu        sync.Mutex
}

type desktopPresenceHub struct {
	mu    sync.RWMutex
	rooms map[string]map[string]*desktopPresenceClient
}

var globalDesktopPresenceHub = &desktopPresenceHub{
	rooms: make(map[string]map[string]*desktopPresenceClient),
}

func (h *desktopPresenceHub) join(client *desktopPresenceClient) {
	h.mu.Lock()
	room := h.rooms[client.roomKey()]
	if room == nil {
		room = make(map[string]*desktopPresenceClient)
		h.rooms[client.roomKey()] = room
	}
	room[client.id] = client
	h.mu.Unlock()

	h.broadcast(client.roomKey(), client.id, desktopPresenceEvent{
		Type:           "join",
		ClientID:       client.id,
		UserID:         client.userID,
		Name:           client.name,
		Color:          client.color,
		WorkspaceID:    client.workspaceID,
		DesktopSpaceID: client.desktopSpaceID,
		LastSeen:       time.Now().UnixMilli(),
	})
}

func (h *desktopPresenceHub) leave(client *desktopPresenceClient) {
	h.mu.Lock()
	room := h.rooms[client.roomKey()]
	if room != nil {
		delete(room, client.id)
		if len(room) == 0 {
			delete(h.rooms, client.roomKey())
		}
	}
	h.mu.Unlock()

	h.broadcast(client.roomKey(), client.id, desktopPresenceEvent{
		Type:           "leave",
		ClientID:       client.id,
		UserID:         client.userID,
		Name:           client.name,
		Color:          client.color,
		WorkspaceID:    client.workspaceID,
		DesktopSpaceID: client.desktopSpaceID,
		LastSeen:       time.Now().UnixMilli(),
	})
}

func (h *desktopPresenceHub) broadcast(roomKey string, senderID string, event desktopPresenceEvent) {
	h.mu.RLock()
	clients := make([]*desktopPresenceClient, 0, len(h.rooms[roomKey]))
	for _, client := range h.rooms[roomKey] {
		if client.id != senderID {
			clients = append(clients, client)
		}
	}
	h.mu.RUnlock()

	for _, client := range clients {
		client.write(event)
	}
}

func (h *desktopPresenceHub) broadcastDesktopSpaceUpdated(workspaceID string, desktopSpaceID string, revision string) {
	h.broadcast(workspaceID+":"+desktopSpaceID, "", desktopPresenceEvent{
		Type:           "desktop_space_updated",
		WorkspaceID:    workspaceID,
		DesktopSpaceID: desktopSpaceID,
		LastSeen:       time.Now().UnixMilli(),
		Revision:       revision,
		Action:         "updated",
	})
}

func (h *desktopPresenceHub) broadcastDesktopSpaceDeleted(workspaceID string, desktopSpaceID string) {
	h.broadcast(workspaceID+":"+desktopSpaceID, "", desktopPresenceEvent{
		Type:           "desktop_space_updated",
		WorkspaceID:    workspaceID,
		DesktopSpaceID: desktopSpaceID,
		LastSeen:       time.Now().UnixMilli(),
		Revision:       time.Now().UTC().Format(time.RFC3339Nano),
		Action:         "deleted",
	})
}

func (c *desktopPresenceClient) roomKey() string {
	return c.workspaceID + ":" + c.desktopSpaceID
}

func (c *desktopPresenceClient) write(event desktopPresenceEvent) {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	_ = c.conn.WriteJSON(event)
}

var desktopPresenceUpgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin:     checkDesktopPresenceOrigin,
}

func checkDesktopPresenceOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return false
	}
	originURL, err := url.Parse(origin)
	if err != nil {
		return false
	}
	if originURL.Scheme != "http" && originURL.Scheme != "https" {
		return false
	}
	host := originURL.Hostname()
	if host == "localhost" || host == "127.0.0.1" {
		port, err := strconv.Atoi(originURL.Port())
		return err == nil && port >= 3000 && port <= 5999
	}
	return strings.HasSuffix(host, ".businessos.dev") || host == "businessos.dev" || host == "app.businessos.com"
}

func (h *desktopPresenceHandler) handleWebSocket(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid workspace id"})
		return
	}
	desktopSpaceID := strings.TrimSpace(c.Param("desktopSpaceId"))
	desktopSpaceUUID, err := uuid.Parse(desktopSpaceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid desktop space id"})
		return
	}

	var allowed bool
	err = h.pool.QueryRow(c.Request.Context(),
		`SELECT EXISTS(
			SELECT 1
			FROM workspace_members wm
			JOIN workspace_desktop_spaces wds ON wds.workspace_id = wm.workspace_id
			WHERE wm.workspace_id=$1
			  AND wm.user_id=$2
			  AND wm.status='active'
			  AND wds.id=$3
		)`,
		workspaceID, user.ID, desktopSpaceUUID).Scan(&allowed)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "desktop presence membership check failed",
			"workspace_id", workspaceID,
			"user_id", user.ID,
			"desktop_space_id", desktopSpaceID,
			"error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not verify workspace membership"})
		return
	}
	if !allowed {
		c.JSON(http.StatusForbidden, gin.H{"error": "Workspace access denied"})
		return
	}

	conn, err := desktopPresenceUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		slog.WarnContext(c.Request.Context(), "desktop presence websocket upgrade failed", "error", err)
		return
	}
	defer conn.Close()
	conn.SetReadLimit(desktopPresenceMaxMessageSize)

	client := &desktopPresenceClient{
		id:             uuid.NewString(),
		userID:         user.ID,
		name:           displayName(user.Name, user.Email),
		color:          colorForPresence(user.ID),
		workspaceID:    workspaceID.String(),
		desktopSpaceID: desktopSpaceID,
		conn:           conn,
	}

	h.hub.join(client)
	defer h.hub.leave(client)

	for {
		var msg desktopPresenceMessage
		if err := conn.ReadJSON(&msg); err != nil {
			return
		}
		switch msg.Type {
		case "cursor":
			if !validPresenceCoordinate(msg.X) || !validPresenceCoordinate(msg.Y) ||
				!validPresenceViewport(msg.ViewportWidth) || !validPresenceViewport(msg.ViewportHeight) {
				continue
			}
			h.hub.broadcast(client.roomKey(), client.id, desktopPresenceEvent{
				Type:           "cursor",
				ClientID:       client.id,
				UserID:         client.userID,
				Name:           client.name,
				Color:          client.color,
				WorkspaceID:    client.workspaceID,
				DesktopSpaceID: client.desktopSpaceID,
				X:              msg.X,
				Y:              msg.Y,
				ViewportWidth:  msg.ViewportWidth,
				ViewportHeight: msg.ViewportHeight,
				ActiveModule:   sanitizePresenceLabel(msg.ActiveModule),
				ActiveTitle:    sanitizePresenceLabel(msg.ActiveTitle),
				LastSeen:       time.Now().UnixMilli(),
			})
		case "heartbeat":
			client.write(desktopPresenceEvent{
				Type:           "heartbeat",
				ClientID:       client.id,
				UserID:         client.userID,
				Name:           client.name,
				Color:          client.color,
				WorkspaceID:    client.workspaceID,
				DesktopSpaceID: client.desktopSpaceID,
				LastSeen:       time.Now().UnixMilli(),
			})
		case "leave":
			return
		default:
			_ = conn.WriteMessage(websocket.TextMessage, mustJSON(gin.H{"type": "error", "error": "Unknown message type"}))
		}
	}
}

func validPresenceCoordinate(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= desktopPresenceMinCoordinate && value <= desktopPresenceMaxCoordinate
}

func validPresenceViewport(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0 && value <= desktopPresenceMaxCoordinate
}

func displayName(name string, email string) string {
	if trimmed := strings.TrimSpace(name); trimmed != "" {
		return trimmed
	}
	if trimmed := strings.TrimSpace(email); trimmed != "" {
		return trimmed
	}
	return "Teammate"
}

func sanitizePresenceLabel(value string) string {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) > 80 {
		return trimmed[:80]
	}
	return trimmed
}

func colorForPresence(value string) string {
	palette := []string{"#0ea5e9", "#22c55e", "#f97316", "#a855f7", "#ef4444", "#14b8a6"}
	hash := uint32(2166136261)
	for _, b := range []byte(value) {
		hash ^= uint32(b)
		hash *= 16777619
	}
	return palette[int(hash)%len(palette)]
}

func mustJSON(value any) []byte {
	b, _ := json.Marshal(value)
	return b
}
