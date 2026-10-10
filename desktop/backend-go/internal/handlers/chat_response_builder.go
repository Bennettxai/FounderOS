package handlers

import (
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/streaming"
)

// extractDocumentTitle extracts a title from the document content or user message
func extractDocumentTitle(content string, userMessage string) string {
	// Try to find first heading
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimPrefix(line, "# ")
		}
		if strings.HasPrefix(line, "## ") {
			return strings.TrimPrefix(line, "## ")
		}
	}

	// Fallback: use user message
	title := userMessage
	if len(title) > 60 {
		title = title[:60] + "..."
	}
	return title
}

// detectStructuredArtifact analyzes response content to detect if it should be an artifact.
// Works with models that don't follow ```artifact format (like Llama 3.3 70B).
func detectStructuredArtifact(content string, userMessage string) *streaming.Artifact {
	slog.Debug("detectStructuredArtifact: Called", "contentLen", len(content), "messagePreview", userMessage[:min(50, len(userMessage))])

	contentLower := strings.ToLower(content)
	msgLower := strings.ToLower(userMessage)
	lines := strings.Split(content, "\n")

	// Count structural elements
	headingCount := 0
	listItemCount := 0
	numberedListCount := 0
	codeBlockCount := 0
	tableRowCount := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			headingCount++
		}
		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
			listItemCount++
		}
		if len(trimmed) > 2 && trimmed[0] >= '0' && trimmed[0] <= '9' && (trimmed[1] == '.' || trimmed[1] == ')') {
			numberedListCount++
		}
		if strings.HasPrefix(trimmed, "```") {
			codeBlockCount++
		}
		if strings.HasPrefix(trimmed, "|") && strings.Contains(trimmed, "|") {
			tableRowCount++
		}
	}

	// Calculate structure score
	structureScore := headingCount*3 + listItemCount + numberedListCount*2 + codeBlockCount*2 + tableRowCount

	// Detect document type based on keywords and structure
	docType := ""

	// Plan detection - lower threshold
	planKeywords := []string{"fase", "phase", "etapa", "step", "milestone", "roadmap", "timeline",
		"cronograma", "plano", "plan", "objetivo", "goal", "meta", "sprint", "iteration",
		"week 1", "week 2", "semana 1", "semana 2", "day 1", "day 2", "dia 1", "dia 2"}
	for _, kw := range planKeywords {
		if strings.Contains(msgLower, kw) || strings.Contains(contentLower, kw) {
			docType = "plan"
			break
		}
	}

	// Proposal detection
	if docType == "" {
		proposalKeywords := []string{"proposal", "proposta", "orçamento", "budget", "escopo", "scope",
			"deliverables", "entregáveis", "investimento", "investment", "pricing", "preço"}
		for _, kw := range proposalKeywords {
			if strings.Contains(msgLower, kw) || strings.Contains(contentLower, kw) {
				docType = "proposal"
				break
			}
		}
	}

	// Report/Analysis detection
	if docType == "" {
		reportKeywords := []string{"analysis", "análise", "report", "relatório", "findings", "conclusões",
			"recommendations", "recomendações", "metrics", "métricas", "results", "resultados", "assessment"}
		for _, kw := range reportKeywords {
			if strings.Contains(msgLower, kw) || strings.Contains(contentLower, kw) {
				docType = "report"
				break
			}
		}
	}

	// Code detection
	if docType == "" && codeBlockCount >= 2 {
		codeKeywords := []string{"code", "código", "implement", "implementar", "function", "função",
			"class", "classe", "component", "componente", "api", "endpoint"}
		for _, kw := range codeKeywords {
			if strings.Contains(msgLower, kw) {
				docType = "code"
				break
			}
		}
	}

	// Document detection (generic)
	if docType == "" {
		docKeywords := []string{"document", "documento", "write", "escrever", "create", "criar",
			"draft", "rascunho", "template", "modelo", "guide", "guia", "manual", "tutorial"}
		for _, kw := range docKeywords {
			if strings.Contains(msgLower, kw) {
				docType = "document"
				break
			}
		}
	}

	// Determine if content qualifies as artifact
	// Criteria: sufficient length + structure OR explicit document type detected
	minLength := 300 // Lower threshold for structured content
	if structureScore >= 5 {
		minLength = 200 // Even lower if well-structured
	}

	shouldCreateArtifact := false

	if docType != "" && len(content) >= minLength {
		shouldCreateArtifact = true
	} else if len(content) >= 500 && structureScore >= 8 {
		// Long, well-structured content even without explicit keywords
		shouldCreateArtifact = true
		docType = "document"
	} else if len(content) >= 800 && headingCount >= 2 {
		// Fallback: long content with multiple sections
		shouldCreateArtifact = true
		docType = "document"
	}

	slog.Debug("detectStructuredArtifact: Analysis", "structureScore", structureScore, "docType", docType, "contentLen", len(content), "shouldCreate", shouldCreateArtifact)

	if !shouldCreateArtifact {
		slog.Debug("detectStructuredArtifact: Returning nil - not creating artifact")
		return nil
	}

	// Extract title
	title := extractDocumentTitle(content, userMessage)
	slog.Debug("detectStructuredArtifact: Creating artifact", "title", title, "type", docType)

	return &streaming.Artifact{
		Type:    docType,
		Title:   title,
		Content: content,
	}
}

// applyReasoningTemplate applies a reasoning template to LLM options
func applyReasoningTemplate(opts *services.LLMOptions, template sqlc.ReasoningTemplate) {
	// Apply thinking instruction from template
	if template.ThinkingInstruction != nil && *template.ThinkingInstruction != "" {
		opts.ThinkingInstruction = *template.ThinkingInstruction
		slog.Debug("ChatV2: Applied template thinking instruction", "len", len(*template.ThinkingInstruction))
	}

	// Apply max thinking tokens if set
	if template.MaxThinkingTokens != nil && *template.MaxThinkingTokens > 0 {
		opts.MaxThinkingTokens = int(*template.MaxThinkingTokens)
	}

	// Store template ID for tracing
	if template.ID.Valid {
		templateID := template.ID.Bytes
		opts.ReasoningTemplateID = uuid.UUID(templateID).String()
	}
}
