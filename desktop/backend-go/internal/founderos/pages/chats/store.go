package chats

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rhl/businessos-backend/internal/founderos/pages/osdata"
)

// Store reads and writes the FounderOS workspace's founderos_agent_messages.
type Store struct{ Pool *pgxpool.Pool }

func (s Store) workspace(ctx context.Context) (string, error) {
	var id string
	err := s.Pool.QueryRow(ctx, `SELECT id::text FROM workspaces WHERE slug = 'founderos'`).Scan(&id)
	if err == pgx.ErrNoRows {
		return "", osdata.ErrNoWorkspace
	}
	return id, err
}

// Thread is one agent's conversation, oldest first.
func (s Store) Thread(ctx context.Context, agentID string) ([]osdata.AgentMessage, error) {
	ws, err := s.workspace(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.Pool.Query(ctx, `SELECT id, agent_id, role, content, tool_calls, created_at FROM founderos_agent_messages
		WHERE workspace_id = $1 AND agent_id = $2 ORDER BY created_at, id`, ws, agentID)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (osdata.AgentMessage, error) {
		var m osdata.AgentMessage
		var raw []byte
		var at time.Time
		if err := r.Scan(&m.ID, &m.AgentID, &m.Role, &m.Content, &raw, &at); err != nil {
			return m, err
		}
		m.CreatedAt, m.ToolCalls = at.UTC().Format("2006-01-02T15:04:05.000Z"), []osdata.ToolCallBrief{}
		if len(raw) > 0 {
			_ = json.Unmarshal(raw, &m.ToolCalls)
		}
		return m, nil
	})
	if out == nil {
		out = []osdata.AgentMessage{}
	}
	return out, err
}

// Append stores one turn.
func (s Store) Append(ctx context.Context, agentID, role, content string, at time.Time) error {
	ws, err := s.workspace(ctx)
	if err != nil {
		return err
	}
	_, err = s.Pool.Exec(ctx, `INSERT INTO founderos_agent_messages (id, workspace_id, agent_id, role, content, tool_calls, created_at)
		VALUES ($1, $2, $3, $4, $5, '[]', $6)`, uuid.NewString(), ws, agentID, role, content, at)
	return err
}
