package tools

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
)

// ArtifactData represents parsed artifact data from LLM response
type ArtifactData struct {
	Type    string `json:"type"`
	Title   string `json:"title"`
	Content string `json:"content"`
	Summary string `json:"summary,omitempty"`
}

// ParsedArtifacts holds artifacts extracted from a response
type ParsedArtifacts struct {
	Artifacts     []ArtifactData
	CleanResponse string
}

// ArtifactTypeToEnum converts string type to sqlc enum
func ArtifactTypeToEnum(t string) sqlc.Artifacttype {
	typeMap := map[string]sqlc.Artifacttype{
		"code":     sqlc.ArtifacttypeCODE,
		"document": sqlc.ArtifacttypeDOCUMENT,
		"markdown": sqlc.ArtifacttypeMARKDOWN,
		"react":    sqlc.ArtifacttypeREACT,
		"html":     sqlc.ArtifacttypeHTML,
		"svg":      sqlc.ArtifacttypeSVG,
		// Map old types to DOCUMENT
		"proposal":  sqlc.ArtifacttypeDOCUMENT,
		"sop":       sqlc.ArtifacttypeDOCUMENT,
		"framework": sqlc.ArtifacttypeDOCUMENT,
		"agenda":    sqlc.ArtifacttypeDOCUMENT,
		"report":    sqlc.ArtifacttypeDOCUMENT,
		"plan":      sqlc.ArtifacttypeDOCUMENT,
		"other":     sqlc.ArtifacttypeDOCUMENT,
	}

	if enum, ok := typeMap[strings.ToLower(t)]; ok {
		return enum
	}
	return sqlc.ArtifacttypeDOCUMENT
}

// ParseArtifactsFromResponse extracts artifact blocks from LLM response
func ParseArtifactsFromResponse(response string) ParsedArtifacts {
	result := ParsedArtifacts{
		Artifacts:     []ArtifactData{},
		CleanResponse: response,
	}

	// Pattern to match artifact blocks
	// ```artifact\n{json}\n```
	pattern := regexp.MustCompile("(?s)```artifact\\s*\\n([\\s\\S]*?)\\n```")

	matches := pattern.FindAllStringSubmatch(response, -1)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		jsonContent := strings.TrimSpace(match[1])

		var artifact ArtifactData
		if err := json.Unmarshal([]byte(jsonContent), &artifact); err != nil {
			// Try to be more lenient with JSON parsing
			continue
		}

		// Validate required fields
		if artifact.Title == "" || artifact.Content == "" || artifact.Type == "" {
			continue
		}

		result.Artifacts = append(result.Artifacts, artifact)

		// Replace artifact block with reference in clean response
		replacement := "\n\n[Artifact Created: " + artifact.Title + "]\n\n"
		result.CleanResponse = strings.Replace(result.CleanResponse, match[0], replacement, 1)
	}

	return result
}

// CreateArtifact creates an artifact in the database
func CreateArtifact(
	ctx context.Context,
	pool *pgxpool.Pool,
	userID string,
	conversationID *uuid.UUID,
	contextID *uuid.UUID,
	projectID *uuid.UUID,
	data ArtifactData,
) (*sqlc.Artifact, error) {
	queries := sqlc.New(pool)

	var convID pgtype.UUID
	if conversationID != nil {
		convID = pgtype.UUID{Bytes: *conversationID, Valid: true}
	}

	var ctxID pgtype.UUID
	if contextID != nil {
		ctxID = pgtype.UUID{Bytes: *contextID, Valid: true}
	}

	var projID pgtype.UUID
	if projectID != nil {
		projID = pgtype.UUID{Bytes: *projectID, Valid: true}
	}

	var summary *string
	if data.Summary != "" {
		summary = &data.Summary
	}

	artifact, err := queries.CreateArtifact(ctx, sqlc.CreateArtifactParams{
		UserID:         userID,
		ConversationID: convID,
		ContextID:      ctxID,
		ProjectID:      projID,
		Title:          data.Title,
		Type:           ArtifactTypeToEnum(data.Type),
		Content:        data.Content,
		Summary:        summary,
	})
	if err != nil {
		return nil, err
	}

	return &artifact, nil
}

// SaveArtifactsFromResponse parses and saves artifacts from an LLM response
func SaveArtifactsFromResponse(
	ctx context.Context,
	pool *pgxpool.Pool,
	userID string,
	conversationID *uuid.UUID,
	contextID *uuid.UUID,
	response string,
) (ParsedArtifacts, error) {
	parsed := ParseArtifactsFromResponse(response)

	for i, artifactData := range parsed.Artifacts {
		artifact, err := CreateArtifact(ctx, pool, userID, conversationID, contextID, nil, artifactData)
		if err != nil {
			// Log error but continue with other artifacts
			continue
		}
		// Update artifact data with created ID
		parsed.Artifacts[i].Summary = uuid.UUID(artifact.ID.Bytes).String()
	}

	return parsed, nil
}

// Tool definitions for LLM tool calling
var ToolDefinitions = []map[string]interface{}{
	{
		"type": "function",
		"function": map[string]interface{}{
			"name":        "create_artifact",
			"description": "Create a business artifact (document, proposal, SOP, etc.) that will be saved and displayed to the user",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"title": map[string]interface{}{
						"type":        "string",
						"description": "Title of the artifact",
					},
					"content": map[string]interface{}{
						"type":        "string",
						"description": "Full content of the artifact in markdown format",
					},
					"artifact_type": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"proposal", "sop", "framework", "agenda", "report", "plan", "code", "document", "markdown", "other"},
						"description": "Type of artifact being created",
					},
					"summary": map[string]interface{}{
						"type":        "string",
						"description": "Brief summary of the artifact (optional)",
					},
				},
				"required": []string{"title", "content", "artifact_type"},
			},
		},
	},
	{
		"type": "function",
		"function": map[string]interface{}{
			"name":        "list_artifacts",
			"description": "List existing artifacts, optionally filtered by type",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"artifact_type": map[string]interface{}{
						"type":        "string",
						"description": "Filter by artifact type (optional)",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of artifacts to return (default 10)",
					},
				},
			},
		},
	},
}
