package handlers

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestNormalizeDesktopSpaceKind(t *testing.T) {
	require.Equal(t, "personal", normalizeDesktopSpaceKind(""))
	require.Equal(t, "personal", normalizeDesktopSpaceKind("personal"))
	require.Equal(t, "team", normalizeDesktopSpaceKind(" TEAM "))
	require.Equal(t, "workspace", normalizeDesktopSpaceKind("workspace"))
	require.Empty(t, normalizeDesktopSpaceKind("public"))
}

func TestValidateDesktopSpaceConfigRequiresObject(t *testing.T) {
	_, err := validateDesktopSpaceConfig(json.RawMessage(`[]`), uuid.New(), "Shared", "workspace")
	require.Error(t, err)

	_, err = validateDesktopSpaceConfig(json.RawMessage(`{"id": "not-canonical"}`), uuid.New(), "Shared", "workspace")
	require.NoError(t, err)
}

func TestValidateDesktopSpaceConfigCanonicalizesIdentity(t *testing.T) {
	id := uuid.New()
	config, err := validateDesktopSpaceConfig(
		json.RawMessage(`{"id":"wrong","name":"Wrong","kind":"personal","desktopIcons":[]}`),
		id,
		"Team Desk",
		"team",
	)
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(config, &payload))
	require.Equal(t, id.String(), payload["id"])
	require.Equal(t, "Team Desk", payload["name"])
	require.Equal(t, "team", payload["kind"])
	require.Equal(t, []any{}, payload["desktopIcons"])
}

func TestDesktopSpaceIDFromConfig(t *testing.T) {
	id := uuid.New()
	parsed, ok := desktopSpaceIDFromConfig(json.RawMessage(`{"id":"` + id.String() + `"}`))
	require.True(t, ok)
	require.Equal(t, id, parsed)

	_, ok = desktopSpaceIDFromConfig(json.RawMessage(`{"id":"not-a-uuid"}`))
	require.False(t, ok)
}
