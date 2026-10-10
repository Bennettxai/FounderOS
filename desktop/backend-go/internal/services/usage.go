package services

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
)

// UsageService handles usage logging
type UsageService struct {
	pool *pgxpool.Pool
}

// NewUsageService creates a new UsageService
func NewUsageService(pool *pgxpool.Pool) *UsageService {
	return &UsageService{pool: pool}
}

// LogAIUsageParams contains parameters for logging AI usage
type LogAIUsageParams struct {
	UserID         string
	ConversationID *uuid.UUID
	Provider       string
	Model          string
	InputTokens    int
	OutputTokens   int
	TotalTokens    int
	ThinkingTokens int // Tokens used for Chain of Thought reasoning
	AgentName      string
	DelegatedTo    string
	RequestType    string
	NodeID         *uuid.UUID
	ProjectID      *uuid.UUID
	DurationMs     int
	StartedAt      time.Time
	CompletedAt    time.Time
	EstimatedCost  float64
}

// LogAIUsage logs an AI usage event
func (s *UsageService) LogAIUsage(ctx context.Context, params LogAIUsageParams) error {
	queries := sqlc.New(s.pool)

	var conversationID pgtype.UUID
	if params.ConversationID != nil {
		conversationID = pgtype.UUID{Bytes: *params.ConversationID, Valid: true}
	}

	var nodeID pgtype.UUID
	if params.NodeID != nil {
		nodeID = pgtype.UUID{Bytes: *params.NodeID, Valid: true}
	}

	var projectID pgtype.UUID
	if params.ProjectID != nil {
		projectID = pgtype.UUID{Bytes: *params.ProjectID, Valid: true}
	}

	inputTokens := int32(params.InputTokens)
	outputTokens := int32(params.OutputTokens)
	totalTokens := int32(params.TotalTokens)
	thinkingTokens := int32(params.ThinkingTokens)
	durationMs := int32(params.DurationMs)

	var agentName *string
	if params.AgentName != "" {
		agentName = &params.AgentName
	}

	var delegatedTo *string
	if params.DelegatedTo != "" {
		delegatedTo = &params.DelegatedTo
	}

	var requestType *string
	if params.RequestType != "" {
		requestType = &params.RequestType
	}

	// Convert cost to pgtype.Numeric (storing as dollars with decimal precision)
	// Use the Scan method to properly set the numeric value
	cost := pgtype.Numeric{}
	_ = cost.Scan(params.EstimatedCost)

	_, err := queries.CreateAIUsageLog(ctx, sqlc.CreateAIUsageLogParams{
		UserID:         params.UserID,
		ConversationID: conversationID,
		Provider:       params.Provider,
		Model:          params.Model,
		InputTokens:    &inputTokens,
		OutputTokens:   &outputTokens,
		TotalTokens:    &totalTokens,
		ThinkingTokens: &thinkingTokens,
		AgentName:      agentName,
		DelegatedTo:    delegatedTo,
		RequestType:    requestType,
		NodeID:         nodeID,
		ProjectID:      projectID,
		DurationMs:     &durationMs,
		StartedAt:      pgtype.Timestamptz{Time: params.StartedAt, Valid: true},
		CompletedAt:    pgtype.Timestamptz{Time: params.CompletedAt, Valid: true},
		EstimatedCost:  cost,
	})

	return err
}

// LogMCPUsageParams contains parameters for logging MCP usage
type LogMCPUsageParams struct {
	UserID         string
	ToolName       string
	ServerName     string
	InputParams    []byte
	OutputResult   []byte
	Success        bool
	ErrorMessage   string
	DurationMs     int
	ConversationID *uuid.UUID
	AIRequestID    *uuid.UUID
}

// LogMCPUsage logs an MCP tool usage event
func (s *UsageService) LogMCPUsage(ctx context.Context, params LogMCPUsageParams) error {
	queries := sqlc.New(s.pool)

	var conversationID pgtype.UUID
	if params.ConversationID != nil {
		conversationID = pgtype.UUID{Bytes: *params.ConversationID, Valid: true}
	}

	var aiRequestID pgtype.UUID
	if params.AIRequestID != nil {
		aiRequestID = pgtype.UUID{Bytes: *params.AIRequestID, Valid: true}
	}

	durationMs := int32(params.DurationMs)
	success := params.Success

	var serverName *string
	if params.ServerName != "" {
		serverName = &params.ServerName
	}

	var errorMessage *string
	if params.ErrorMessage != "" {
		errorMessage = &params.ErrorMessage
	}

	_, err := queries.CreateMCPUsageLog(ctx, sqlc.CreateMCPUsageLogParams{
		UserID:         params.UserID,
		ToolName:       params.ToolName,
		ServerName:     serverName,
		InputParams:    params.InputParams,
		OutputResult:   params.OutputResult,
		Success:        &success,
		ErrorMessage:   errorMessage,
		DurationMs:     &durationMs,
		ConversationID: conversationID,
		AiRequestID:    aiRequestID,
	})

	return err
}

// CalculateEstimatedCost calculates the estimated cost for an AI request
// Prices are approximate and should be updated regularly
func CalculateEstimatedCost(provider, model string, inputTokens, outputTokens int) float64 {
	// Price per 1K tokens (in dollars)
	type pricing struct {
		input  float64
		output float64
	}

	prices := map[string]map[string]pricing{
		"anthropic": {
			"claude-sonnet-4-20250514":   {input: 0.003, output: 0.015},
			"claude-opus-4-1-20250805":     {input: 0.015, output: 0.075},
			"claude-3-7-sonnet-20250219": {input: 0.003, output: 0.015},
			"claude-3-5-sonnet-20241022": {input: 0.003, output: 0.015},
			"claude-haiku-4-5-20251001":  {input: 0.001, output: 0.005},
			"claude-3-opus-20240229":     {input: 0.015, output: 0.075},
			"claude-3-sonnet-20240229":   {input: 0.003, output: 0.015},
			"claude-3-haiku-20240307":    {input: 0.00025, output: 0.00125},
		},
		"groq": {
			"llama-3.3-70b-versatile": {input: 0.00059, output: 0.00079},
			"llama-3.1-8b-instant":    {input: 0.00005, output: 0.00008},
			"mixtral-8x7b-32768":      {input: 0.00024, output: 0.00024},
		},
		"ollama_local": {
			// Local models are free
			"default": {input: 0, output: 0},
		},
		"ollama_cloud": {
			// Approximate cloud Ollama pricing
			"default": {input: 0.0001, output: 0.0002},
		},
	}

	providerPrices, ok := prices[provider]
	if !ok {
		return 0
	}

	price, ok := providerPrices[model]
	if !ok {
		// Use default if model not found
		price, ok = providerPrices["default"]
		if !ok {
			return 0
		}
	}

	inputCost := float64(inputTokens) / 1000 * price.input
	outputCost := float64(outputTokens) / 1000 * price.output

	return inputCost + outputCost
}
