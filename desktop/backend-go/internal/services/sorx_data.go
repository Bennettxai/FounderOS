// Package services provides business logic for BusinessOS.
package services

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

func (s *SorxService) handleClientDataRequest(ctx context.Context, userID string, query map[string]interface{}) (interface{}, error) {
	action, _ := query["action"].(string)
	data, _ := query["data"].(map[string]interface{})

	switch action {
	case "create":
		var clientID uuid.UUID
		err := s.pool.QueryRow(ctx, `
			INSERT INTO clients (user_id, name, email, company_name, status, metadata)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id
		`, userID, data["name"], data["email"], data["company_name"], data["status"], data).Scan(&clientID)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"id": clientID}, nil

	case "find":
		filter, _ := query["filter"].(map[string]interface{})
		if email, ok := filter["email"].(string); ok {
			var client map[string]interface{}
			err := s.pool.QueryRow(ctx, `
				SELECT id, name, email, company_name FROM clients
				WHERE user_id = $1 AND email = $2
			`, userID, email).Scan(&client)
			if err != nil {
				return nil, nil // Not found
			}
			return client, nil
		}
		return nil, nil

	default:
		return nil, fmt.Errorf("unknown action: %s", action)
	}
}

func (s *SorxService) handleProjectDataRequest(ctx context.Context, userID string, query map[string]interface{}) (interface{}, error) {
	action, _ := query["action"].(string)
	data, _ := query["data"].(map[string]interface{})

	switch action {
	case "create":
		var projectID uuid.UUID
		err := s.pool.QueryRow(ctx, `
			INSERT INTO projects (user_id, name, description, status, client_id)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id
		`, userID, data["name"], data["description"], data["status"], data["client_id"]).Scan(&projectID)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"id": projectID}, nil

	default:
		return nil, fmt.Errorf("unknown action: %s", action)
	}
}

func (s *SorxService) handleTaskDataRequest(ctx context.Context, userID string, query map[string]interface{}) (interface{}, error) {
	action, _ := query["action"].(string)
	data, _ := query["data"].(map[string]interface{})

	switch action {
	case "create":
		var taskID uuid.UUID
		err := s.pool.QueryRow(ctx, `
			INSERT INTO tasks (user_id, title, description, priority, status, source)
			VALUES ($1, $2, $3, $4, 'pending', $5)
			RETURNING id
		`, userID, data["title"], data["description"], data["priority"], data["source"]).Scan(&taskID)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"id": taskID}, nil

	default:
		return nil, fmt.Errorf("unknown action: %s", action)
	}
}

func (s *SorxService) handleContextDataRequest(ctx context.Context, userID string, query map[string]interface{}) (interface{}, error) {
	action, _ := query["action"].(string)
	data, _ := query["data"].(map[string]interface{})

	switch action {
	case "create":
		contentJSON, _ := json.Marshal(data["content"])
		sourceJSON, _ := json.Marshal(data["source"])

		var contextID uuid.UUID
		err := s.pool.QueryRow(ctx, `
			INSERT INTO contexts (user_id, title, type, content, source, tags)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id
		`, userID, data["title"], data["type"], contentJSON, sourceJSON, data["tags"]).Scan(&contextID)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"id": contextID}, nil

	default:
		return nil, fmt.Errorf("unknown action: %s", action)
	}
}

func (s *SorxService) handleDailyLogDataRequest(ctx context.Context, userID string, query map[string]interface{}) (interface{}, error) {
	return nil, fmt.Errorf("daily_log operations not yet implemented")
}
