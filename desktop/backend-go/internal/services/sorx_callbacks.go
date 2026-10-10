// Package services provides business logic for BusinessOS.
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// CallbackRequest represents a callback from Sorx.
type CallbackRequest struct {
	ID          uuid.UUID   `json:"id"`
	Type        string      `json:"type"`
	ExecutionID string      `json:"execution_id"`
	SkillID     string      `json:"skill_id"`
	StepID      string      `json:"step_id"`
	UserID      string      `json:"user_id"`
	Timestamp   time.Time   `json:"timestamp"`
	Payload     interface{} `json:"payload"`
}

// CallbackResponse is sent back to Sorx.
type CallbackResponse struct {
	ID        uuid.UUID   `json:"id"`
	RequestID uuid.UUID   `json:"request_id"`
	Success   bool        `json:"success"`
	Error     string      `json:"error,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
	Result    interface{} `json:"result,omitempty"`
}

// HandleCallback processes a callback from Sorx.
func (s *SorxService) HandleCallback(ctx context.Context, req CallbackRequest) (*CallbackResponse, error) {
	response := &CallbackResponse{
		ID:        uuid.New(),
		RequestID: req.ID,
		Timestamp: time.Now().UTC(),
		Success:   true,
	}

	var err error
	switch req.Type {
	case "request_skill":
		response.Result, err = s.handleSkillCallback(ctx, req)
	case "request_agent":
		response.Result, err = s.handleAgentCallback(ctx, req)
	case "request_decision":
		response.Result, err = s.handleDecisionCallback(ctx, req)
	case "request_data":
		response.Result, err = s.handleDataCallback(ctx, req)
	case "return_result":
		err = s.handleResultCallback(ctx, req)
	case "update_progress":
		err = s.handleProgressCallback(ctx, req)
	case "log_event":
		err = s.handleLogCallback(ctx, req)
	default:
		err = fmt.Errorf("unknown callback type: %s", req.Type)
	}

	if err != nil {
		response.Success = false
		response.Error = err.Error()
	}

	return response, nil
}

func (s *SorxService) handleSkillCallback(ctx context.Context, req CallbackRequest) (interface{}, error) {
	payload, ok := req.Payload.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid skill callback payload")
	}

	skillID, _ := payload["skill_id"].(string)
	params, _ := payload["params"].(map[string]interface{})

	return map[string]interface{}{
		"execution_id": uuid.New().String(),
		"skill_id":     skillID,
		"status":       "queued",
		"params":       params,
	}, nil
}

func (s *SorxService) handleAgentCallback(ctx context.Context, req CallbackRequest) (interface{}, error) {
	payload, ok := req.Payload.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid agent callback payload")
	}

	agentRole, _ := payload["agent_role"].(string)
	task, _ := payload["task"].(string)

	return map[string]interface{}{
		"response":  fmt.Sprintf("Agent %s processed task: %s", agentRole, task),
		"reasoning": "Placeholder reasoning",
	}, nil
}

func (s *SorxService) handleDecisionCallback(ctx context.Context, req CallbackRequest) (interface{}, error) {
	payload, ok := req.Payload.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid decision callback payload")
	}

	question, _ := payload["question"].(string)
	options, _ := payload["options"].([]interface{})
	priority, _ := payload["priority"].(string)
	if priority == "" {
		priority = "medium"
	}

	var optionStrs []string
	for _, opt := range options {
		if s, ok := opt.(string); ok {
			optionStrs = append(optionStrs, s)
		}
	}

	var decisionID uuid.UUID
	err := s.pool.QueryRow(ctx, `
		INSERT INTO pending_decisions (
			execution_id, skill_id, step_id, user_id,
			question, options, priority, context
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`, req.ExecutionID, req.SkillID, req.StepID, req.UserID,
		question, optionStrs, priority, payload).Scan(&decisionID)

	if err != nil {
		return nil, fmt.Errorf("failed to create pending decision: %w", err)
	}

	return map[string]interface{}{
		"id":           decisionID,
		"execution_id": req.ExecutionID,
		"skill_id":     req.SkillID,
		"status":       "pending",
	}, nil
}

func (s *SorxService) handleDataCallback(ctx context.Context, req CallbackRequest) (interface{}, error) {
	payload, ok := req.Payload.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid data callback payload")
	}

	dataType, _ := payload["data_type"].(string)
	query, _ := payload["query"].(map[string]interface{})

	switch dataType {
	case "client":
		return s.handleClientDataRequest(ctx, req.UserID, query)
	case "project":
		return s.handleProjectDataRequest(ctx, req.UserID, query)
	case "task":
		return s.handleTaskDataRequest(ctx, req.UserID, query)
	case "context":
		return s.handleContextDataRequest(ctx, req.UserID, query)
	case "daily_log":
		return s.handleDailyLogDataRequest(ctx, req.UserID, query)
	default:
		return nil, fmt.Errorf("unknown data type: %s", dataType)
	}
}

func (s *SorxService) handleResultCallback(ctx context.Context, req CallbackRequest) error {
	payload, ok := req.Payload.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid result callback payload")
	}

	status, _ := payload["status"].(string)
	result, _ := payload["result"].(map[string]interface{})
	errorMsg, _ := payload["error"].(string)
	metrics, _ := payload["metrics"].(map[string]interface{})

	resultJSON, _ := json.Marshal(result)
	metricsJSON, _ := json.Marshal(metrics)

	_, err := s.pool.Exec(ctx, `
		UPDATE skill_executions SET
			status = $2,
			result = $3,
			error = $4,
			metrics = $5,
			completed_at = NOW()
		WHERE id = $1::uuid
	`, req.ExecutionID, status, resultJSON, errorMsg, metricsJSON)

	return err
}

func (s *SorxService) handleProgressCallback(ctx context.Context, req CallbackRequest) error {
	return nil
}

func (s *SorxService) handleLogCallback(ctx context.Context, req CallbackRequest) error {
	return nil
}
