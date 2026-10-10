package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// ShareMemory shares a private memory with specific users
func (m *MemoryService) ShareMemory(ctx context.Context, memoryID uuid.UUID, ownerID string, shareWithUserIDs []string) error {
	_, err := m.pool.Exec(ctx, "SELECT share_memory($1, $2, $3)",
		memoryID, ownerID, shareWithUserIDs)
	if err != nil {
		return fmt.Errorf("failed to share memory: %w", err)
	}
	return nil
}

// UnshareMemory makes a shared memory private again
func (m *MemoryService) UnshareMemory(ctx context.Context, memoryID uuid.UUID, ownerID string) error {
	_, err := m.pool.Exec(ctx, "SELECT unshare_memory($1, $2)", memoryID, ownerID)
	if err != nil {
		return fmt.Errorf("failed to unshare memory: %w", err)
	}
	return nil
}

// TrackAccess increments access counter when memory is retrieved
func (m *MemoryService) TrackAccess(ctx context.Context, memoryID uuid.UUID) error {
	_, err := m.pool.Exec(ctx, "SELECT track_memory_access($1)", memoryID)
	if err != nil {
		return fmt.Errorf("failed to track memory access: %w", err)
	}
	return nil
}

// UpdateWorkspaceMemory updates an existing workspace memory
func (m *MemoryService) UpdateWorkspaceMemory(ctx context.Context, memoryID uuid.UUID, userID string, updates map[string]interface{}) error {
	// Check if user can access this memory
	var canAccess bool
	err := m.pool.QueryRow(ctx, "SELECT can_access_memory($1, $2)", userID, memoryID).Scan(&canAccess)
	if err != nil {
		return fmt.Errorf("failed to check memory access: %w", err)
	}
	if !canAccess {
		return fmt.Errorf("user does not have access to this memory")
	}

	// Build dynamic update query
	query := "UPDATE workspace_memories SET updated_at = NOW()"
	args := []interface{}{memoryID}
	argCount := 1

	for key, value := range updates {
		argCount++
		query += fmt.Sprintf(", %s = $%d", key, argCount)
		args = append(args, value)
	}

	query += " WHERE id = $1"

	_, err = m.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update workspace memory: %w", err)
	}

	return nil
}

// DeleteWorkspaceMemory soft-deletes a workspace memory
func (m *MemoryService) DeleteWorkspaceMemory(ctx context.Context, memoryID uuid.UUID, userID string) error {
	query := `
		UPDATE workspace_memories
		SET is_active = FALSE, updated_at = NOW()
		WHERE id = $1 AND (created_by = $2 OR owner_user_id = $2)
	`

	result, err := m.pool.Exec(ctx, query, memoryID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete workspace memory: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("memory not found or user not authorized to delete")
	}

	return nil
}
