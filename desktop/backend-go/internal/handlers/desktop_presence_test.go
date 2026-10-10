package handlers

import (
	"math"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDesktopPresenceRoomKeyIncludesWorkspaceAndDesktop(t *testing.T) {
	client := &desktopPresenceClient{
		workspaceID:    "workspace-a",
		desktopSpaceID: "desktop-a",
	}

	require.Equal(t, "workspace-a:desktop-a", client.roomKey())
}

func TestDesktopPresenceHubLeaveRemovesEmptyRoom(t *testing.T) {
	hub := &desktopPresenceHub{rooms: map[string]map[string]*desktopPresenceClient{}}
	client := &desktopPresenceClient{
		id:             "client-a",
		userID:         "user-a",
		name:           "Alice",
		color:          "#0ea5e9",
		workspaceID:    "workspace-a",
		desktopSpaceID: "desktop-a",
	}

	hub.mu.Lock()
	hub.rooms[client.roomKey()] = map[string]*desktopPresenceClient{client.id: client}
	hub.mu.Unlock()

	hub.leave(client)

	hub.mu.RLock()
	defer hub.mu.RUnlock()
	require.Empty(t, hub.rooms)
}

func TestValidPresenceValuesRejectInvalidNumbers(t *testing.T) {
	require.True(t, validPresenceCoordinate(0))
	require.True(t, validPresenceCoordinate(-100000))
	require.True(t, validPresenceCoordinate(100000))
	require.False(t, validPresenceCoordinate(-100001))
	require.False(t, validPresenceCoordinate(100001))
	require.False(t, validPresenceCoordinate(math.NaN()))
	require.False(t, validPresenceCoordinate(math.Inf(1)))

	require.True(t, validPresenceViewport(1440))
	require.True(t, validPresenceViewport(0))
	require.False(t, validPresenceViewport(-1))
	require.False(t, validPresenceViewport(100001))
	require.False(t, validPresenceViewport(math.NaN()))
	require.False(t, validPresenceViewport(math.Inf(-1)))
}

func TestSanitizePresenceLabel(t *testing.T) {
	require.Equal(t, "Claude", sanitizePresenceLabel("  Claude  "))
	require.Len(t, sanitizePresenceLabel("abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz"), 80)
}

func TestCheckDesktopPresenceOrigin(t *testing.T) {
	t.Run("requires origin", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "/presence/ws", nil)
		require.NoError(t, err)

		require.False(t, checkDesktopPresenceOrigin(req))
	})

	t.Run("allows local dev ports", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "/presence/ws", nil)
		require.NoError(t, err)
		req.Header.Set("Origin", "http://localhost:5273")

		require.True(t, checkDesktopPresenceOrigin(req))
	})

	t.Run("rejects local ports outside dev range", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "/presence/ws", nil)
		require.NoError(t, err)
		req.Header.Set("Origin", "http://localhost:8801")

		require.False(t, checkDesktopPresenceOrigin(req))
	})
}
