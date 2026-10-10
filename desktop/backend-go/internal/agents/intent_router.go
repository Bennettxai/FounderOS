package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/config"
	"github.com/rhl/businessos-backend/internal/services"
)

// SmartIntentRouter uses multi-layer analysis for intelligent routing
type SmartIntentRouter struct {
	pool   *pgxpool.Pool
	config *config.Config

	// Pattern matchers (Layer 1 - Fast)
	patterns map[AgentType][]*IntentPattern

	// Semantic signals (Layer 2 - Medium)
	signals map[AgentType][]SemanticSignal

	// LLM classifier enabled
	useLLMClassifier bool
}

// IntentPattern represents a regex pattern with metadata
type IntentPattern struct {
	Pattern     *regexp.Regexp
	Weight      float64
	Description string
	MustMatch   bool // If true, pattern alone triggers routing
}

// SemanticSignal represents semantic indicators
type SemanticSignal struct {
	Indicator   string
	Weight      float64
	Category    string   // "creation", "query", "update", "delete"
	RequiresAll []string // Other signals that must also be present
}

// IntentScore tracks scoring for each agent type
type IntentScore struct {
	Agent      AgentType
	Score      float64
	Confidence float64
	Signals    []string
	Category   string
}

// LLMClassification represents the LLM's classification response
type LLMClassification struct {
	Agent      string   `json:"agent"`
	Confidence float64  `json:"confidence"`
	Reasoning  string   `json:"reasoning"`
	Category   string   `json:"category"`
	Subtasks   []string `json:"subtasks,omitempty"`
}

// NewSmartIntentRouter creates an intelligent intent router
func NewSmartIntentRouter(pool *pgxpool.Pool, cfg *config.Config) *SmartIntentRouter {
	router := &SmartIntentRouter{
		pool:             pool,
		config:           cfg,
		useLLMClassifier: true,
		patterns:         make(map[AgentType][]*IntentPattern),
		signals:          make(map[AgentType][]SemanticSignal),
	}

	router.initializePatterns()
	router.initializeSignals()

	return router
}

// initializePatterns sets up regex patterns for fast matching
func (r *SmartIntentRouter) initializePatterns() {
	// Document Agent patterns - formal document creation
	r.patterns[AgentTypeDocument] = []*IntentPattern{
		// High confidence - explicit document requests
		{Pattern: regexp.MustCompile(`(?i)(create|write|draft|generate)\s+(a\s+)?(formal\s+)?(proposal|sop|report|framework|playbook|manual|guide|brief|agenda|policy|contract)`), Weight: 1.0, MustMatch: true, Description: "Explicit document creation"},
		{Pattern: regexp.MustCompile(`(?i)(criar|escrever|redigir)\s+(uma?\s+)?(proposta|relatório|documento|manual|guia)`), Weight: 1.0, MustMatch: true, Description: "Portuguese document creation"},
		{Pattern: regexp.MustCompile(`(?i)standard\s+operating\s+procedure`), Weight: 1.0, MustMatch: true, Description: "SOP request"},
		{Pattern: regexp.MustCompile(`(?i)(pitch|investor|slide)\s+deck`), Weight: 0.95, MustMatch: true, Description: "Presentation deck"},

		// Medium confidence - document-related terms
		{Pattern: regexp.MustCompile(`(?i)(documentation|template|outline)\s+(for|of|about)`), Weight: 0.7, Description: "Documentation request"},
		{Pattern: regexp.MustCompile(`(?i)formal\s+(document|letter|communication)`), Weight: 0.8, Description: "Formal document"},
	}

	// Project Agent patterns - project/task management
	r.patterns[AgentTypeProject] = []*IntentPattern{
		// High confidence - explicit project requests
		{Pattern: regexp.MustCompile(`(?i)(create|start|setup|initialize)\s+(a\s+)?(new\s+)?project`), Weight: 1.0, MustMatch: true, Description: "Project creation"},
		{Pattern: regexp.MustCompile(`(?i)(plan|prioritize|organize)\s+(my\s+)?(tasks?|day|week|sprint)`), Weight: 0.95, MustMatch: true, Description: "Task planning"},
		{Pattern: regexp.MustCompile(`(?i)(what('s|s)?|which)\s+(should\s+i|to)\s+(work\s+on|do\s+next|prioritize)`), Weight: 0.9, MustMatch: true, Description: "Next task query"},
		{Pattern: regexp.MustCompile(`(?i)(break\s+down|decompose|split)\s+(this\s+)?(task|work|project)`), Weight: 0.9, MustMatch: true, Description: "Task breakdown"},
		{Pattern: regexp.MustCompile(`(?i)(criar|planejar|organizar)\s+(um\s+)?(projeto|tarefas?)`), Weight: 1.0, MustMatch: true, Description: "Portuguese project"},

		// Medium confidence
		{Pattern: regexp.MustCompile(`(?i)(project|sprint|milestone)\s+(status|progress|update|timeline|roadmap)`), Weight: 0.75, Description: "Project status"},
		{Pattern: regexp.MustCompile(`(?i)(assign|allocate|delegate)\s+(to|resources|team)`), Weight: 0.7, Description: "Resource allocation"},
		{Pattern: regexp.MustCompile(`(?i)(backlog|user\s+stor(y|ies)|story\s+points)`), Weight: 0.8, Description: "Agile terms"},
	}

	// Client Agent patterns - CRM/client management
	r.patterns[AgentTypeClient] = []*IntentPattern{
		// High confidence - explicit client requests
		{Pattern: regexp.MustCompile(`(?i)(add|create|onboard)\s+(a\s+)?(new\s+)?client`), Weight: 1.0, MustMatch: true, Description: "Client creation"},
		{Pattern: regexp.MustCompile(`(?i)(move|update|change)\s+(client|lead|prospect|deal)\s+(to|in|through)\s+(pipeline|stage)`), Weight: 1.0, MustMatch: true, Description: "Pipeline update"},
		{Pattern: regexp.MustCompile(`(?i)(log|record|add)\s+(a\s+)?(client\s+)?(interaction|meeting|call|note)`), Weight: 0.95, MustMatch: true, Description: "Interaction logging"},
		{Pattern: regexp.MustCompile(`(?i)(follow\s*up|reach\s+out|contact)\s+(with\s+)?(client|lead|prospect)`), Weight: 0.9, MustMatch: true, Description: "Client follow-up"},
		{Pattern: regexp.MustCompile(`(?i)(cliente|lead|prospecto)\s+(novo|pipeline|contato)`), Weight: 1.0, MustMatch: true, Description: "Portuguese client"},

		// Medium confidence
		{Pattern: regexp.MustCompile(`(?i)client\s+(history|profile|details|status|notes|info)`), Weight: 0.75, Description: "Client info query"},
		{Pattern: regexp.MustCompile(`(?i)(crm|sales\s+pipeline|deal\s+pipeline|opportunity)`), Weight: 0.7, Description: "CRM terms"},
	}

	// Analyst Agent patterns - data analysis AND research queries
	r.patterns[AgentTypeAnalyst] = []*IntentPattern{
		// High confidence - explicit analysis requests
		{Pattern: regexp.MustCompile(`(?i)(analyze|analyse|examine|evaluate|assess)\s+(the\s+)?(data|metrics|performance|trends|numbers|results)`), Weight: 1.0, MustMatch: true, Description: "Data analysis"},
		{Pattern: regexp.MustCompile(`(?i)(how\s+(are|did)\s+we\s+(doing|do)|what('s|s)?\s+(working|not\s+working))`), Weight: 0.95, MustMatch: true, Description: "Performance query"},
		{Pattern: regexp.MustCompile(`(?i)(compare|benchmark|contrast)\s+.*(vs|versus|against|with|to)`), Weight: 0.9, MustMatch: true, Description: "Comparison request"},
		{Pattern: regexp.MustCompile(`(?i)(forecast|predict|project|estimate)\s+.*(revenue|sales|growth|trend)`), Weight: 0.95, MustMatch: true, Description: "Forecasting"},
		{Pattern: regexp.MustCompile(`(?i)(analisar|análise|métricas|desempenho|tendências)`), Weight: 1.0, MustMatch: true, Description: "Portuguese analysis"},

		// Research queries - analyst handles these
		{Pattern: regexp.MustCompile(`(?i)^(how|what|why|when|where|who)\s+(does|do|is|are|was|were|did|can|could|would|should)\s+.+\??\s*$`), Weight: 0.85, Description: "Research question"},
		{Pattern: regexp.MustCompile(`(?i)(explain|describe|tell\s+me\s+about|what\s+is|how\s+does)\s+.+`), Weight: 0.9, MustMatch: true, Description: "Explanation request"},
		{Pattern: regexp.MustCompile(`(?i)(research|investigate|look\s+into|find\s+out|learn\s+about)`), Weight: 0.95, MustMatch: true, Description: "Research request"},
		{Pattern: regexp.MustCompile(`(?i)(como|o\s+que|por\s+que|explique|pesquise)`), Weight: 0.9, MustMatch: true, Description: "Portuguese research"},

		// Medium confidence
		{Pattern: regexp.MustCompile(`(?i)(kpi|roi|okr|metric|dashboard)`), Weight: 0.7, Description: "KPI terms"},
		{Pattern: regexp.MustCompile(`(?i)(trend|pattern|insight|finding)`), Weight: 0.6, Description: "Analysis terms"},
		{Pattern: regexp.MustCompile(`(?i)(swot|competitive\s+analysis|market\s+analysis)`), Weight: 0.85, Description: "Strategic analysis"},
	}

}

// initializeSignals sets up semantic signals for nuanced detection
func (r *SmartIntentRouter) initializeSignals() {
	// Document signals - look for document-like output expectations
	r.signals[AgentTypeDocument] = []SemanticSignal{
		{Indicator: "formal", Weight: 0.3, Category: "tone"},
		{Indicator: "professional", Weight: 0.3, Category: "tone"},
		{Indicator: "structured", Weight: 0.25, Category: "format"},
		{Indicator: "sections", Weight: 0.2, Category: "format"},
		{Indicator: "deliverable", Weight: 0.4, Category: "output"},
		{Indicator: "send to", Weight: 0.3, Category: "action"},
		{Indicator: "present to", Weight: 0.3, Category: "action"},
		{Indicator: "submit", Weight: 0.3, Category: "action"},
	}

	// Project signals - look for planning/organization intent
	r.signals[AgentTypeProject] = []SemanticSignal{
		{Indicator: "deadline", Weight: 0.4, Category: "time"},
		{Indicator: "due date", Weight: 0.4, Category: "time"},
		{Indicator: "by when", Weight: 0.3, Category: "time"},
		{Indicator: "schedule", Weight: 0.35, Category: "planning"},
		{Indicator: "timeline", Weight: 0.35, Category: "planning"},
		{Indicator: "priority", Weight: 0.4, Category: "organization"},
		{Indicator: "urgent", Weight: 0.3, Category: "organization"},
		{Indicator: "blocked", Weight: 0.4, Category: "status"},
		{Indicator: "dependency", Weight: 0.35, Category: "status"},
	}

	// Client signals - look for relationship/CRM intent
	r.signals[AgentTypeClient] = []SemanticSignal{
		{Indicator: "relationship", Weight: 0.4, Category: "crm"},
		{Indicator: "pipeline", Weight: 0.5, Category: "crm"},
		{Indicator: "deal", Weight: 0.4, Category: "sales"},
		{Indicator: "prospect", Weight: 0.4, Category: "sales"},
		{Indicator: "lead", Weight: 0.35, Category: "sales"},
		{Indicator: "meeting with", Weight: 0.3, Category: "interaction"},
		{Indicator: "called", Weight: 0.25, Category: "interaction"},
		{Indicator: "emailed", Weight: 0.25, Category: "interaction"},
	}

	// Analyst signals - look for data/metrics intent
	r.signals[AgentTypeAnalyst] = []SemanticSignal{
		{Indicator: "data", Weight: 0.3, Category: "data"},
		{Indicator: "numbers", Weight: 0.3, Category: "data"},
		{Indicator: "statistics", Weight: 0.4, Category: "data"},
		{Indicator: "percentage", Weight: 0.3, Category: "metrics"},
		{Indicator: "growth", Weight: 0.35, Category: "metrics"},
		{Indicator: "decline", Weight: 0.35, Category: "metrics"},
		{Indicator: "trend", Weight: 0.4, Category: "analysis"},
		{Indicator: "pattern", Weight: 0.35, Category: "analysis"},
		{Indicator: "why", Weight: 0.2, Category: "inquiry"},
		{Indicator: "reason", Weight: 0.2, Category: "inquiry"},
	}

}

// ClassifyIntent performs multi-layer intent classification
func (r *SmartIntentRouter) ClassifyIntent(ctx context.Context, messages []services.ChatMessage, tieredCtx *services.TieredContext) Intent {
	if len(messages) == 0 {
		return Intent{Category: "general", ShouldDelegate: false, TargetAgent: AgentTypeOrchestrator, Confidence: 0.5}
	}

	// Extract the last user message
	var lastUserMsg string
	var conversationContext []string
	for i := len(messages) - 1; i >= 0; i-- {
		if strings.ToLower(messages[i].Role) == "user" {
			if lastUserMsg == "" {
				lastUserMsg = messages[i].Content
			}
			conversationContext = append(conversationContext, messages[i].Content)
			if len(conversationContext) >= 3 {
				break
			}
		}
	}

	if lastUserMsg == "" {
		return Intent{Category: "general", ShouldDelegate: false, TargetAgent: AgentTypeOrchestrator, Confidence: 0.5}
	}

	msgLower := strings.ToLower(lastUserMsg)

	// Layer 1: Pattern matching (fast, high precision)
	patternIntent := r.classifyByPatterns(msgLower)
	if patternIntent.Confidence >= 0.9 {
		return patternIntent
	}

	// Layer 2: Semantic signal analysis
	signalScores := r.analyzeSemanticSignals(msgLower)

	// Layer 3: Context-aware boosting
	contextScores := r.analyzeContext(tieredCtx, conversationContext)

	// Combine scores
	finalScores := r.combineScores(patternIntent, signalScores, contextScores)

	// Find best match
	bestIntent := r.selectBestIntent(finalScores, msgLower)

	// Layer 4: LLM classification for ambiguous cases
	if bestIntent.Confidence < 0.7 && r.useLLMClassifier && r.config != nil {
		llmIntent := r.classifyWithLLM(ctx, lastUserMsg, conversationContext)
		if llmIntent.Confidence > bestIntent.Confidence {
			return llmIntent
		}
	}

	return bestIntent
}

// classifyByPatterns performs regex pattern matching
func (r *SmartIntentRouter) classifyByPatterns(msg string) Intent {
	var bestMatch Intent
	bestMatch.Confidence = 0

	for agentType, patterns := range r.patterns {
		for _, p := range patterns {
			if p.Pattern.MatchString(msg) {
				score := p.Weight
				if p.MustMatch && score > bestMatch.Confidence {
					bestMatch = Intent{
						Category:       string(agentType),
						ShouldDelegate: true,
						TargetAgent:    agentType,
						Confidence:     score,
						Reasoning:      fmt.Sprintf("Pattern match: %s", p.Description),
					}
				}
			}
		}
	}

	return bestMatch
}

// analyzeSemanticSignals scores based on semantic indicators
func (r *SmartIntentRouter) analyzeSemanticSignals(msg string) map[AgentType]*IntentScore {
	scores := make(map[AgentType]*IntentScore)

	for agentType, signals := range r.signals {
		score := &IntentScore{Agent: agentType, Score: 0, Signals: []string{}}

		for _, signal := range signals {
			if strings.Contains(msg, signal.Indicator) {
				score.Score += signal.Weight
				score.Signals = append(score.Signals, signal.Indicator)
				if score.Category == "" {
					score.Category = signal.Category
				}
			}
		}

		// Normalize score (max 1.0)
		if score.Score > 1.0 {
			score.Score = 1.0
		}
		score.Confidence = score.Score * 0.7 // Signals alone max 70% confidence

		scores[agentType] = score
	}

	return scores
}

// analyzeContext boosts scores based on tiered context
func (r *SmartIntentRouter) analyzeContext(tieredCtx *services.TieredContext, recentMessages []string) map[AgentType]float64 {
	boosts := make(map[AgentType]float64)

	if tieredCtx == nil {
		return boosts
	}

	// If user has a project selected, boost project agent
	if tieredCtx.Level1.Project != nil {
		boosts[AgentTypeProject] += 0.15
		boosts[AgentTypeDocument] += 0.1 // Documents often relate to projects
	}

	// If there's a linked client in context, boost client agent
	if tieredCtx.Level1.LinkedClient != nil {
		boosts[AgentTypeClient] += 0.2
	}

	// Analyze recent conversation for continuity
	for _, msg := range recentMessages {
		msgLower := strings.ToLower(msg)
		if strings.Contains(msgLower, "document") || strings.Contains(msgLower, "proposal") {
			boosts[AgentTypeDocument] += 0.1
		}
		if strings.Contains(msgLower, "task") || strings.Contains(msgLower, "project") {
			boosts[AgentTypeProject] += 0.1
		}
		if strings.Contains(msgLower, "client") || strings.Contains(msgLower, "lead") {
			boosts[AgentTypeClient] += 0.1
		}
		if strings.Contains(msgLower, "analyze") || strings.Contains(msgLower, "metric") {
			boosts[AgentTypeAnalyst] += 0.1
		}
	}

	return boosts
}

// combineScores merges all scoring layers
func (r *SmartIntentRouter) combineScores(patternIntent Intent, signalScores map[AgentType]*IntentScore, contextBoosts map[AgentType]float64) map[AgentType]float64 {
	combined := make(map[AgentType]float64)

	// Add pattern score
	if patternIntent.Confidence > 0 {
		combined[patternIntent.TargetAgent] = patternIntent.Confidence * 0.5
	}

	// Add signal scores
	for agent, score := range signalScores {
		combined[agent] += score.Confidence * 0.3
	}

	// Add context boosts
	for agent, boost := range contextBoosts {
		combined[agent] += boost * 0.2
	}

	return combined
}

// selectBestIntent picks the highest scoring agent
func (r *SmartIntentRouter) selectBestIntent(scores map[AgentType]float64, _ string) Intent {
	var bestAgent AgentType = AgentTypeOrchestrator
	var bestScore float64 = 0

	for agent, score := range scores {
		if score > bestScore {
			bestScore = score
			bestAgent = agent
		}
	}

	// Minimum threshold for delegation
	if bestScore < 0.4 {
		return Intent{
			Category:       "general",
			ShouldDelegate: false,
			TargetAgent:    AgentTypeOrchestrator,
			Confidence:     0.6,
			Reasoning:      "No strong signal detected, using orchestrator",
		}
	}

	return Intent{
		Category:       string(bestAgent),
		ShouldDelegate: true,
		TargetAgent:    bestAgent,
		Confidence:     bestScore,
		Reasoning:      fmt.Sprintf("Multi-layer analysis score: %.2f", bestScore),
	}
}

// classifyWithLLM uses the LLM for complex/ambiguous cases
func (r *SmartIntentRouter) classifyWithLLM(ctx context.Context, message string, recentContext []string) Intent {
	if r.config == nil {
		return Intent{Category: "general", ShouldDelegate: false, TargetAgent: AgentTypeOrchestrator, Confidence: 0.5}
	}

	// Use a fast model for classification
	llm := services.NewLLMService(r.config, r.config.DefaultModel)

	classificationPrompt := fmt.Sprintf(`Classify this user request into ONE of these categories:
- document: Creating formal documents (proposals, SOPs, reports, frameworks, guides)
- project: Project/task management, planning, scheduling, prioritization
- client: Client/CRM management, pipeline, interactions, follow-ups
- analyst: Data analysis, metrics, trends, comparisons, forecasting
- general: General questions, conversation, unclear intent

User message: "%s"

Recent context: %s

Respond ONLY with valid JSON:
{"agent": "category_name", "confidence": 0.0-1.0, "reasoning": "brief explanation", "category": "subcategory"}`,
		message,
		strings.Join(recentContext, " | "))

	// Set short timeout for classification
	classifyCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	response, err := llm.ChatComplete(classifyCtx, []services.ChatMessage{
		{Role: "user", Content: classificationPrompt},
	}, "You are an intent classifier. Respond only with JSON.")

	if err != nil {
		slog.Warn("intent router LLM classification failed", "error", err)
		return Intent{Category: "general", ShouldDelegate: false, TargetAgent: AgentTypeOrchestrator, Confidence: 0.5}
	}

	// Parse LLM response
	var classification LLMClassification

	// Extract JSON from response (handle markdown code blocks)
	jsonStr := response
	if idx := strings.Index(response, "{"); idx >= 0 {
		if endIdx := strings.LastIndex(response, "}"); endIdx > idx {
			jsonStr = response[idx : endIdx+1]
		}
	}

	if err := json.Unmarshal([]byte(jsonStr), &classification); err != nil {
		slog.Warn("intent router failed to parse LLM response", "error", err)
		return Intent{Category: "general", ShouldDelegate: false, TargetAgent: AgentTypeOrchestrator, Confidence: 0.5}
	}

	// Map to agent type
	agentMap := map[string]AgentType{
		"document": AgentTypeDocument,
		"project":  AgentTypeProject,
		"client":   AgentTypeClient,
		"analyst":  AgentTypeAnalyst,
		"analysis": AgentTypeAnalyst,
		"general":  AgentTypeOrchestrator,
	}

	targetAgent, ok := agentMap[strings.ToLower(classification.Agent)]
	if !ok {
		targetAgent = AgentTypeOrchestrator
	}

	shouldDelegate := targetAgent != AgentTypeOrchestrator && classification.Confidence >= 0.6

	return Intent{
		Category:       classification.Agent,
		ShouldDelegate: shouldDelegate,
		TargetAgent:    targetAgent,
		Confidence:     classification.Confidence,
		Reasoning:      fmt.Sprintf("[LLM] %s", classification.Reasoning),
	}
}

// EnableLLMClassifier enables or disables LLM-based classification
func (r *SmartIntentRouter) EnableLLMClassifier(enabled bool) {
	r.useLLMClassifier = enabled
}

// NewIntentRouter creates a basic intent router (compatibility wrapper)
func NewIntentRouter() *SmartIntentRouter {
	return NewSmartIntentRouter(nil, nil)
}

// ClassifyIntentBasic provides backward compatibility for simple classification
func (r *SmartIntentRouter) ClassifyIntentBasic(messages []services.ChatMessage, tieredCtx *services.TieredContext) Intent {
	return r.ClassifyIntent(context.Background(), messages, tieredCtx)
}

// ShouldDelegateForFocusMode checks if focus mode requires delegation
func ShouldDelegateForFocusMode(focusMode string) (bool, AgentType) {
	switch focusMode {
	case "write":
		return true, AgentTypeDocument
	case "analyze", "research":
		return true, AgentTypeAnalyst
	case "plan", "build":
		return true, AgentTypeProject
	default:
		return false, AgentTypeOrchestrator
	}
}
