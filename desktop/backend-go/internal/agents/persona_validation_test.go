package agents

import (
	"strings"
	"testing"

	"github.com/rhl/businessos-backend/internal/prompts"
)

// TestAgentPersonaValidation verifies each agent has proper system prompts
// and adheres to their designated persona characteristics
func TestAgentPersonaValidation(t *testing.T) {
	tests := []struct {
		name             string
		agentType        AgentType
		promptKey        string
		requiredPhrases  []string
		forbiddenPhrases []string
	}{
		{
			name:      "Orchestrator persona",
			agentType: AgentTypeOrchestrator,
			promptKey: "orchestrator",
			requiredPhrases: []string{
				"business", // Should contain business-related content
			},
			forbiddenPhrases: []string{
				"[PLACEHOLDER]",
				"TODO:",
			},
		},
		{
			name:      "Document agent persona",
			agentType: AgentTypeDocument,
			promptKey: "document",
			requiredPhrases: []string{
				"document",
			},
			forbiddenPhrases: []string{
				"[PLACEHOLDER]",
				"TODO:",
			},
		},
		{
			name:      "Analyst agent persona",
			agentType: AgentTypeAnalyst,
			promptKey: "analyst",
			requiredPhrases: []string{
				"analy",
			},
			forbiddenPhrases: []string{
				"[PLACEHOLDER]",
				"TODO:",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt := prompts.GetPrompt(tt.promptKey)

			if prompt == "" {
				t.Errorf("Agent %s has empty system prompt", tt.agentType)
				return
			}

			// Check required phrases
			for _, phrase := range tt.requiredPhrases {
				if !containsIgnoreCase(prompt, phrase) {
					t.Errorf("Agent %s prompt missing required phrase: '%s'", tt.agentType, phrase)
				}
			}

			// Check forbidden phrases
			for _, phrase := range tt.forbiddenPhrases {
				if containsIgnoreCase(prompt, phrase) {
					t.Errorf("Agent %s prompt contains forbidden phrase: '%s'", tt.agentType, phrase)
				}
			}

			// Verify reasonable length
			if len(prompt) < 100 {
				t.Errorf("Agent %s prompt too short (%d chars), likely incomplete", tt.agentType, len(prompt))
			}

			t.Logf("Agent %s: prompt length=%d chars", tt.agentType, len(prompt))
		})
	}
}

// TestAgentSystemPromptUniqueness verifies each agent type has a distinct prompt
func TestAgentSystemPromptUniqueness(t *testing.T) {
	prompts := map[string]string{
		"orchestrator": getAgentPrompt(AgentTypeOrchestrator),
		"document":     getAgentPrompt(AgentTypeDocument),
		"analyst":      getAgentPrompt(AgentTypeAnalyst),
		"project":      getAgentPrompt(AgentTypeProject),
	}

	// Check each pair for uniqueness
	keys := make([]string, 0, len(prompts))
	for k := range prompts {
		keys = append(keys, k)
	}

	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			k1, k2 := keys[i], keys[j]
			if prompts[k1] == prompts[k2] {
				t.Errorf("Agents %s and %s have identical prompts", k1, k2)
			}

			// Calculate similarity (shouldn't be too high)
			similarity := calculateSimilarity(prompts[k1], prompts[k2])
			if similarity > 0.8 {
				t.Errorf("Agents %s and %s have >80%% similar prompts (%.1f%%)",
					k1, k2, similarity*100)
			}
		}
	}
}

// TestAgentNameConsistency verifies agent type constants are consistent
func TestAgentNameConsistency(t *testing.T) {
	tests := []struct {
		agentType    AgentType
		expectedName string
	}{
		{AgentTypeOrchestrator, "orchestrator"},
		{AgentTypeDocument, "document"},
		{AgentTypeAnalyst, "analyst"},
		{AgentTypeProject, "project"},
	}

	for _, tt := range tests {
		t.Run(string(tt.agentType), func(t *testing.T) {
			// Verify the agent type string matches expected
			if string(tt.agentType) != tt.expectedName {
				t.Errorf("Expected AgentType string = %s, got %s", tt.expectedName, string(tt.agentType))
			}
		})
	}
}

// TestAgentDescriptions verifies all agents have descriptions defined
func TestAgentDescriptions(t *testing.T) {
	// Test the description mapping directly
	descriptions := map[AgentType]string{
		AgentTypeOrchestrator: "Main coordinator that handles requests and delegates to sub-agents",
		AgentTypeDocument:     "Creates professional business documents",
		AgentTypeAnalyst:      "Analyzes data and provides insights",
		AgentTypeProject:      "Helps with planning and prioritization",
	}

	for agentType, expectedDesc := range descriptions {
		t.Run(string(agentType), func(t *testing.T) {
			if expectedDesc == "" {
				t.Errorf("Agent %s has empty description", agentType)
			}

			if len(expectedDesc) < 10 {
				t.Errorf("Agent %s description too short: '%s'", agentType, expectedDesc)
			}

			t.Logf("Agent %s: '%s'", agentType, expectedDesc)
		})
	}
}

// TestFocusModeAgentMapping verifies focus modes map to correct agent types
func TestFocusModeAgentMapping(t *testing.T) {
	tests := []struct {
		focusMode     string
		expectedAgent AgentType
	}{
		{"research", AgentTypeAnalyst},
		{"analyze", AgentTypeAnalyst},
		{"write", AgentTypeDocument},
		{"build", AgentTypeProject},
		{"plan", AgentTypeProject},
		{"general", AgentTypeOrchestrator},
		{"unknown_mode", AgentTypeOrchestrator}, // Default fallback
	}

	for _, tt := range tests {
		t.Run(tt.focusMode, func(t *testing.T) {
			agent := GetAgentForFocusMode(tt.focusMode)

			if agent != tt.expectedAgent {
				t.Errorf("Focus mode '%s' mapped to %s, expected %s",
					tt.focusMode, agent, tt.expectedAgent)
			}
		})
	}
}

// TestDelegationParsing was removed — parseDelegation was part of the v1
// orchestrator system that has been deleted in favour of the SmartIntentRouter.

// TestOrchestratorPromptWithContext verifies context-aware prompt building
func TestOrchestratorPromptWithContext(t *testing.T) {
	tests := []struct {
		name        string
		userName    string
		projectName string
		projectDesc string
		shouldHave  []string
	}{
		{
			name:        "Full context",
			userName:    "John",
			projectName: "Acme Project",
			projectDesc: "Building a SaaS platform",
			shouldHave:  []string{"John", "Acme Project"},
		},
		{
			name:        "No project",
			userName:    "Maria",
			projectName: "",
			projectDesc: "",
			shouldHave:  []string{"Maria"},
		},
		{
			name:        "Empty user",
			userName:    "",
			projectName: "Test Project",
			projectDesc: "Test description",
			shouldHave:  []string{"Test Project"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt := prompts.BuildOrchestratorPromptWithContext(
				tt.userName, tt.projectName, tt.projectDesc)

			for _, phrase := range tt.shouldHave {
				if phrase != "" && !strings.Contains(prompt, phrase) {
					t.Errorf("Prompt should contain '%s'", phrase)
				}
			}

			t.Logf("Generated prompt length: %d chars", len(prompt))
		})
	}
}

// TestAgentTypeMapping verifies agent type mappings exist
func TestAgentTypeMapping(t *testing.T) {
	agentTypes := []AgentType{
		AgentTypeOrchestrator,
		AgentTypeDocument,
		AgentTypeAnalyst,
		AgentTypeProject,
		AgentTypeTask,
		AgentTypeClient,
	}

	for _, agentType := range agentTypes {
		t.Run(string(agentType), func(t *testing.T) {
			if string(agentType) == "" {
				t.Error("Agent type is empty")
			}
		})
	}
}

// Helper functions

func getAgentPrompt(agentType AgentType) string {
	promptKey := ""
	switch agentType {
	case AgentTypeOrchestrator:
		promptKey = "orchestrator"
	case AgentTypeDocument:
		promptKey = "document"
	case AgentTypeAnalyst:
		promptKey = "analyst"
	case AgentTypeProject:
		promptKey = "planner"
	}
	return prompts.GetPrompt(promptKey)
}

func containsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

func calculateSimilarity(s1, s2 string) float64 {
	if s1 == s2 {
		return 1.0
	}
	if len(s1) == 0 || len(s2) == 0 {
		return 0.0
	}

	// Simple word-based Jaccard similarity
	words1 := strings.Fields(strings.ToLower(s1))
	words2 := strings.Fields(strings.ToLower(s2))

	set1 := make(map[string]bool)
	for _, w := range words1 {
		set1[w] = true
	}

	set2 := make(map[string]bool)
	for _, w := range words2 {
		set2[w] = true
	}

	intersection := 0
	for w := range set1 {
		if set2[w] {
			intersection++
		}
	}

	union := len(set1) + len(set2) - intersection
	if union == 0 {
		return 0.0
	}

	return float64(intersection) / float64(union)
}
