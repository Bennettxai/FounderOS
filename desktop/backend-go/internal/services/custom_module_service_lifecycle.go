package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// DeleteModule deletes a module
func (s *CustomModuleService) DeleteModule(ctx context.Context, moduleID uuid.UUID, userID uuid.UUID) error {
	query := "DELETE FROM custom_modules WHERE id = $1 AND created_by = $2"
	result, err := s.pool.Exec(ctx, query, moduleID, userID)
	if err != nil {
		s.logger.Error("Failed to delete module", "error", err, "module_id", moduleID)
		return fmt.Errorf("failed to delete module: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("module not found or unauthorized")
	}

	s.logger.Info("Module deleted", "module_id", moduleID)
	return nil
}

// PublishModule publishes a module to the registry
func (s *CustomModuleService) PublishModule(ctx context.Context, moduleID uuid.UUID, userID uuid.UUID) error {
	query := `
		UPDATE custom_modules
		SET is_published = TRUE, is_public = TRUE, published_at = NOW()
		WHERE id = $1 AND created_by = $2
	`

	result, err := s.pool.Exec(ctx, query, moduleID, userID)
	if err != nil {
		s.logger.Error("Failed to publish module", "error", err, "module_id", moduleID)
		return fmt.Errorf("failed to publish module: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("module not found or unauthorized")
	}

	s.logger.Info("Module published", "module_id", moduleID)
	return nil
}
