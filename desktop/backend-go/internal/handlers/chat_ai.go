package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rhl/businessos-backend/internal/agents"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/tools"
	"github.com/rhl/businessos-backend/internal/utils"
)

// DocumentAI handles document writing assistance using the Document agent (V2)
func (h *ChatHandler) DocumentAI(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	var req struct {
		Prompt    string  `json:"prompt" binding:"required"`
		Model     *string `json:"model"`
		ContextID *string `json:"context_id"`
		ProjectID *string `json:"project_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	model := h.cfg.DefaultModel
	if req.Model != nil && *req.Model != "" {
		model = *req.Model
	}

	// Parse IDs
	var contextID, projectID *uuid.UUID
	if req.ContextID != nil {
		if parsed, err := uuid.Parse(*req.ContextID); err == nil {
			contextID = &parsed
		}
	}
	if req.ProjectID != nil {
		if parsed, err := uuid.Parse(*req.ProjectID); err == nil {
			projectID = &parsed
		}
	}

	// Use Document agent V2
	registry := agents.NewAgentRegistry(h.pool, h.cfg, nil, nil, nil)
	agent := registry.GetAgent(agents.AgentTypeDocument, user.ID, user.Name, nil, nil)
	agent.SetModel(model)

	messages := []services.ChatMessage{
		{Role: "user", Content: req.Prompt},
	}

	llm := services.NewLLMService(h.cfg, model)
	response, err := llm.ChatComplete(c.Request.Context(), messages, agent.GetSystemPrompt())
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "complete AI request", err)
		return
	}

	parsed, _ := tools.SaveArtifactsFromResponse(c.Request.Context(), h.pool, user.ID, nil, contextID, response)

	// Link to project if provided
	if projectID != nil && len(parsed.Artifacts) > 0 {
		queries := sqlc.New(h.pool)
		for _, artifactData := range parsed.Artifacts {
			if artifactData.Summary != "" {
				if artifactID, err := uuid.Parse(artifactData.Summary); err == nil {
					queries.LinkArtifact(c.Request.Context(), sqlc.LinkArtifactParams{
						ID:        pgtype.UUID{Bytes: artifactID, Valid: true},
						ProjectID: pgtype.UUID{Bytes: *projectID, Valid: true},
					})
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"content":   parsed.CleanResponse,
		"artifacts": parsed.Artifacts,
	})
}

// AnalyzeContent handles data analysis using the Analyst agent (V2)
func (h *ChatHandler) AnalyzeContent(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	var req struct {
		Content string  `json:"content" binding:"required"`
		Model   *string `json:"model"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	model := h.cfg.DefaultModel
	if req.Model != nil && *req.Model != "" {
		model = *req.Model
	}

	// Use Analyst agent V2
	registry := agents.NewAgentRegistry(h.pool, h.cfg, nil, nil, nil)
	agent := registry.GetAgent(agents.AgentTypeAnalyst, user.ID, user.Name, nil, nil)
	agent.SetModel(model)

	messages := []services.ChatMessage{
		{Role: "user", Content: "Analyze the following content and provide insights:\n\n" + req.Content},
	}

	llm := services.NewLLMService(h.cfg, model)
	response, err := llm.ChatComplete(c.Request.Context(), messages, agent.GetSystemPrompt())
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "complete AI request", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"analysis": response})
}

// ExtractTasks extracts actionable tasks from content using Task agent (V2)
func (h *ChatHandler) ExtractTasks(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	var req struct {
		Content         string                   `json:"content"`
		ArtifactContent string                   `json:"artifact_content"`
		ArtifactTitle   string                   `json:"artifact_title"`
		ArtifactType    string                   `json:"artifact_type"`
		Model           *string                  `json:"model"`
		TeamMembers     []map[string]interface{} `json:"team_members"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.Info("[ExtractTasks] Bind error", "error", err)
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}
	slog.Info("[ExtractTasks] Received:,,", "title", req.ArtifactTitle, "title", req.ArtifactType, "content_len", len(req.ArtifactContent))

	// Use artifact_content if content is empty
	content := req.Content
	if content == "" {
		content = req.ArtifactContent
	}
	if content == "" {
		utils.RespondBadRequest(c, slog.Default(), "content or artifact_content is required")
		return
	}

	model := h.cfg.DefaultModel
	if req.Model != nil && *req.Model != "" {
		model = *req.Model
	}

	// Use Task agent V2 for task extraction
	registry := agents.NewAgentRegistry(h.pool, h.cfg, nil, nil, nil)
	agent := registry.GetAgent(agents.AgentTypeTask, user.ID, user.Name, nil, nil)
	agent.SetModel(model)

	prompt := fmt.Sprintf(`Extract actionable tasks from the following %s titled "%s".
Return them as a JSON array of objects with "title", "description", and "priority" (high/medium/low) fields.

Focus on concrete, actionable items that can be assigned to team members.

Content:
%s

Return ONLY a valid JSON array, no other text. Example format:
[{"title": "Task name", "description": "What needs to be done", "priority": "high"}]`,
		req.ArtifactType, req.ArtifactTitle, content)

	messages := []services.ChatMessage{
		{Role: "user", Content: prompt},
	}

	llm := services.NewLLMService(h.cfg, model)
	response, err := llm.ChatComplete(c.Request.Context(), messages, agent.GetSystemPrompt())
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "complete AI request", err)
		return
	}

	// Try to parse the response as JSON array
	var tasks []map[string]interface{}
	if err := json.Unmarshal([]byte(response), &tasks); err != nil {
		// Try to extract JSON from response
		start := strings.Index(response, "[")
		end := strings.LastIndex(response, "]")
		if start >= 0 && end > start {
			jsonStr := response[start : end+1]
			if err := json.Unmarshal([]byte(jsonStr), &tasks); err != nil {
				slog.Info("[ExtractTasks] Failed to parse tasks JSON", "error", err)
				c.JSON(http.StatusOK, gin.H{"tasks": []interface{}{}})
				return
			}
		} else {
			slog.Info("[ExtractTasks] No JSON array found in response")
			c.JSON(http.StatusOK, gin.H{"tasks": []interface{}{}})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"tasks": tasks})
}

// CreatePlan creates a strategic plan using the Project agent (V2)
func (h *ChatHandler) CreatePlan(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	var req struct {
		Goal      string  `json:"goal" binding:"required"`
		Timeframe *string `json:"timeframe"`
		Model     *string `json:"model"`
		ProjectID *string `json:"project_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	model := h.cfg.DefaultModel
	if req.Model != nil && *req.Model != "" {
		model = *req.Model
	}

	// Use Project agent V2 for planning
	registry := agents.NewAgentRegistry(h.pool, h.cfg, nil, nil, nil)
	agent := registry.GetAgent(agents.AgentTypeProject, user.ID, user.Name, nil, nil)
	agent.SetModel(model)

	prompt := "Create a detailed strategic plan for the following goal:\n\n" + req.Goal
	if req.Timeframe != nil {
		prompt += "\n\nTimeframe: " + *req.Timeframe
	}
	prompt += "\n\nInclude milestones, success criteria, and potential risks."

	messages := []services.ChatMessage{
		{Role: "user", Content: prompt},
	}

	llm := services.NewLLMService(h.cfg, model)
	response, err := llm.ChatComplete(c.Request.Context(), messages, agent.GetSystemPrompt())
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "complete AI request", err)
		return
	}

	// Parse and save any artifacts (plans often generate artifacts)
	var projectID *uuid.UUID
	if req.ProjectID != nil {
		if parsed, err := uuid.Parse(*req.ProjectID); err == nil {
			projectID = &parsed
		}
	}

	parsed, _ := tools.SaveArtifactsFromResponse(c.Request.Context(), h.pool, user.ID, nil, nil, response)

	// Link to project if provided
	if projectID != nil && len(parsed.Artifacts) > 0 {
		queries := sqlc.New(h.pool)
		for _, artifactData := range parsed.Artifacts {
			if artifactData.Summary != "" {
				if artifactID, err := uuid.Parse(artifactData.Summary); err == nil {
					queries.LinkArtifact(c.Request.Context(), sqlc.LinkArtifactParams{
						ID:        pgtype.UUID{Bytes: artifactID, Valid: true},
						ProjectID: pgtype.UUID{Bytes: *projectID, Valid: true},
					})
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"plan":      parsed.CleanResponse,
		"artifacts": parsed.Artifacts,
	})
}
