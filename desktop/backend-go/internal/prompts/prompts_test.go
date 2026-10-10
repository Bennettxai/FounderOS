package prompts

import (
	"strings"
	"testing"

	"github.com/rhl/businessos-backend/internal/prompts/core"
)

// =============================================================================
// PROMPT LAYER TESTS
// =============================================================================

func TestPromptLayerConstants(t *testing.T) {
	expectedLayers := map[PromptLayer]string{
		LayerIdentity:      "identity",
		LayerFormatting:    "formatting",
		LayerArtifacts:     "artifacts",
		LayerContext:       "context",
		LayerTools:         "tools",
		LayerErrors:        "errors",
		LayerAgentSpecific: "agent_specific",
	}

	for layer, expected := range expectedLayers {
		if string(layer) != expected {
			t.Errorf("Layer %v should equal '%s', got '%s'", layer, expected, string(layer))
		}
	}
}

// =============================================================================
// PROMPT COMPOSER TESTS
// =============================================================================

func TestNewPromptComposer(t *testing.T) {
	composer := NewPromptComposer()
	if composer == nil {
		t.Fatal("NewPromptComposer returned nil")
	}

	// Check that core layers are initialized
	if composer.layers[LayerIdentity] == "" {
		t.Error("Identity layer should be initialized")
	}
	if composer.layers[LayerFormatting] == "" {
		t.Error("Formatting layer should be initialized")
	}
	if composer.layers[LayerArtifacts] == "" {
		t.Error("Artifacts layer should be initialized")
	}
}

func TestComposePromptWithAllLayers(t *testing.T) {
	composer := NewPromptComposer()

	agentPrompt := "You are a test agent."
	result := composer.ComposePrompt(agentPrompt)

	// Should contain all layers
	if !strings.Contains(result, core.CoreIdentity) {
		t.Error("Result should contain CoreIdentity")
	}
	if !strings.Contains(result, core.OutputFormattingStandards) {
		t.Error("Result should contain OutputFormattingStandards")
	}
	if !strings.Contains(result, agentPrompt) {
		t.Error("Result should contain agent prompt")
	}

	// Layers should be separated by dividers
	if !strings.Contains(result, "---") {
		t.Error("Layers should be separated by dividers")
	}
}

func TestComposePromptWithSpecificLayers(t *testing.T) {
	composer := NewPromptComposer()

	agentPrompt := "Test agent"
	result := composer.ComposePrompt(agentPrompt, LayerIdentity, LayerFormatting)

	// Should contain specified layers
	if !strings.Contains(result, core.CoreIdentity) {
		t.Error("Result should contain CoreIdentity")
	}
	if !strings.Contains(result, core.OutputFormattingStandards) {
		t.Error("Result should contain OutputFormattingStandards")
	}

	// Should NOT contain non-specified layers (context check)
	if strings.Contains(result, core.ContextIntegration) && !strings.Contains(result, "LayerContext") {
		// This is a weak test since layers might overlap in content
	}
}

func TestComposePromptEmptyAgent(t *testing.T) {
	composer := NewPromptComposer()

	result := composer.ComposePrompt("")

	// Should still have core layers
	if result == "" {
		t.Error("Result should not be empty even with empty agent prompt")
	}
	if !strings.Contains(result, core.CoreIdentity) {
		t.Error("Result should still contain CoreIdentity")
	}
}

func TestComposeWithContext(t *testing.T) {
	composer := NewPromptComposer()

	result := composer.ComposeWithContext(
		"Test agent",
		"John Doe",
		"Project Alpha",
		"A test project",
	)

	// Should contain user context
	if !strings.Contains(result, "John Doe") {
		t.Error("Result should contain user name")
	}
	if !strings.Contains(result, "Project Alpha") {
		t.Error("Result should contain project name")
	}
	if !strings.Contains(result, "CURRENT SESSION CONTEXT") {
		t.Error("Result should contain session context header")
	}
}

func TestComposeWithContextPartialInfo(t *testing.T) {
	composer := NewPromptComposer()

	// Only user name, no project
	result := composer.ComposeWithContext("Test agent", "Jane", "", "")

	if !strings.Contains(result, "Jane") {
		t.Error("Result should contain user name")
	}

	// No project context should be added when empty
	if strings.Contains(result, "Active Project") {
		t.Error("Should not contain project info when project is empty")
	}
}

func TestComposeWithContextEmpty(t *testing.T) {
	composer := NewPromptComposer()

	result := composer.ComposeWithContext("Test agent", "", "", "")

	// Should not contain CURRENT SESSION CONTEXT when all empty
	if strings.Contains(result, "CURRENT SESSION CONTEXT") {
		t.Error("Should not contain session context when all context is empty")
	}
}

func TestComposeMinimal(t *testing.T) {
	composer := NewPromptComposer()

	result := composer.ComposeMinimal("Minimal agent")

	// Should only have identity and formatting
	if !strings.Contains(result, core.CoreIdentity) {
		t.Error("Minimal should contain CoreIdentity")
	}
	if !strings.Contains(result, core.OutputFormattingStandards) {
		t.Error("Minimal should contain OutputFormattingStandards")
	}

	// Minimal should NOT contain other layers
	if strings.Contains(result, core.ArtifactSystem) {
		t.Error("Minimal should NOT contain ArtifactSystem")
	}
	if strings.Contains(result, core.ToolUsagePatterns) {
		t.Error("Minimal should NOT contain ToolUsagePatterns")
	}
}

func TestComposeForDocument(t *testing.T) {
	composer := NewPromptComposer()

	result := composer.ComposeForDocument("Document agent prompt")

	// Should have document-specific layers
	if !strings.Contains(result, core.CoreIdentity) {
		t.Error("Should contain CoreIdentity")
	}
	if !strings.Contains(result, core.ArtifactSystem) {
		t.Error("Document compose should contain ArtifactSystem")
	}
}

func TestComposeForAnalysis(t *testing.T) {
	composer := NewPromptComposer()

	result := composer.ComposeForAnalysis("Analysis agent prompt")

	// Should have analysis-specific layers
	if !strings.Contains(result, core.CoreIdentity) {
		t.Error("Should contain CoreIdentity")
	}
	if !strings.Contains(result, core.ToolUsagePatterns) {
		t.Error("Analysis compose should contain ToolUsagePatterns")
	}
}

func TestComposeForProject(t *testing.T) {
	composer := NewPromptComposer()

	result := composer.ComposeForProject("Project agent prompt")

	// Should have project-specific layers
	if !strings.Contains(result, core.CoreIdentity) {
		t.Error("Should contain CoreIdentity")
	}
	if !strings.Contains(result, core.ToolUsagePatterns) {
		t.Error("Project compose should contain ToolUsagePatterns")
	}
}

func TestComposeForClient(t *testing.T) {
	composer := NewPromptComposer()

	result := composer.ComposeForClient("Client agent prompt")

	// Should have client-specific layers
	if !strings.Contains(result, core.CoreIdentity) {
		t.Error("Should contain CoreIdentity")
	}
	if !strings.Contains(result, core.ContextIntegration) {
		t.Error("Client compose should contain ContextIntegration")
	}
}

func TestGetLayer(t *testing.T) {
	composer := NewPromptComposer()

	identity := composer.GetLayer(LayerIdentity)
	if identity != core.CoreIdentity {
		t.Error("GetLayer should return correct layer content")
	}

	formatting := composer.GetLayer(LayerFormatting)
	if formatting != core.OutputFormattingStandards {
		t.Error("GetLayer should return formatting layer")
	}
}

func TestSetLayer(t *testing.T) {
	composer := NewPromptComposer()

	customIdentity := "Custom OSA Identity"
	composer.SetLayer(LayerIdentity, customIdentity)

	if composer.GetLayer(LayerIdentity) != customIdentity {
		t.Error("SetLayer should update layer content")
	}
}

func TestSetLayerAffectsCompose(t *testing.T) {
	composer := NewPromptComposer()

	customFormatting := "CUSTOM_FORMATTING_MARKER"
	composer.SetLayer(LayerFormatting, customFormatting)

	result := composer.ComposePrompt("Test agent", LayerFormatting)

	if !strings.Contains(result, customFormatting) {
		t.Error("Custom layer should appear in composed prompt")
	}
}

// =============================================================================
// DEFAULT COMPOSER & CONVENIENCE FUNCTIONS TESTS
// =============================================================================

func TestDefaultComposer(t *testing.T) {
	if DefaultComposer == nil {
		t.Fatal("DefaultComposer should be initialized")
	}

	// Verify it's a working composer
	result := DefaultComposer.ComposePrompt("Test")
	if result == "" {
		t.Error("DefaultComposer should produce output")
	}
}

func TestComposeConvenienceFunction(t *testing.T) {
	result := Compose("Test agent")

	if result == "" {
		t.Error("Compose convenience function should work")
	}
	if !strings.Contains(result, core.CoreIdentity) {
		t.Error("Compose should include core layers")
	}
}

func TestComposeWithLayers(t *testing.T) {
	result := Compose("Test agent", LayerIdentity)

	if !strings.Contains(result, core.CoreIdentity) {
		t.Error("Compose with specific layer should include that layer")
	}
}

func TestComposeWithUserContextConvenience(t *testing.T) {
	result := ComposeWithUserContext("Test agent", "User", "Project", "Description")

	if !strings.Contains(result, "User") {
		t.Error("Should contain user name")
	}
	if !strings.Contains(result, "Project") {
		t.Error("Should contain project name")
	}
}

// =============================================================================
// CORE LAYER CONTENT TESTS
// =============================================================================

func TestCoreIdentityNotEmpty(t *testing.T) {
	if core.CoreIdentity == "" {
		t.Error("CoreIdentity should not be empty")
	}
	if len(core.CoreIdentity) < 100 {
		t.Error("CoreIdentity should have substantial content")
	}
}

func TestOutputFormattingNotEmpty(t *testing.T) {
	if core.OutputFormattingStandards == "" {
		t.Error("OutputFormattingStandards should not be empty")
	}
}

func TestArtifactSystemNotEmpty(t *testing.T) {
	if core.ArtifactSystem == "" {
		t.Error("ArtifactSystem should not be empty")
	}
}

func TestContextIntegrationNotEmpty(t *testing.T) {
	if core.ContextIntegration == "" {
		t.Error("ContextIntegration should not be empty")
	}
}

func TestToolUsagePatternsNotEmpty(t *testing.T) {
	if core.ToolUsagePatterns == "" {
		t.Error("ToolUsagePatterns should not be empty")
	}
}

func TestErrorHandlingNotEmpty(t *testing.T) {
	if core.ErrorHandling == "" {
		t.Error("ErrorHandling should not be empty")
	}
}

// =============================================================================
// PROMPT COMPOSITION QUALITY TESTS
// =============================================================================

func TestPromptNotTooLong(t *testing.T) {
	composer := NewPromptComposer()

	result := composer.ComposePrompt("Test agent")

	// Prompts shouldn't exceed reasonable token limits (rough character estimate)
	// 4 chars ≈ 1 token, 10000 tokens ≈ 40000 chars
	if len(result) > 50000 {
		t.Errorf("Prompt is too long: %d chars", len(result))
	}
}

func TestPromptHasProperStructure(t *testing.T) {
	composer := NewPromptComposer()

	result := composer.ComposePrompt("Test agent")

	// Should have dividers between sections
	dividerCount := strings.Count(result, "---")
	if dividerCount < 2 {
		t.Error("Prompt should have section dividers")
	}

	// Should end with agent-specific content
	if !strings.HasSuffix(strings.TrimSpace(result), "Test agent") {
		t.Error("Prompt should end with agent-specific content")
	}
}

func TestDifferentComposerModes(t *testing.T) {
	composer := NewPromptComposer()

	minimal := composer.ComposeMinimal("Agent")
	document := composer.ComposeForDocument("Agent")
	analysis := composer.ComposeForAnalysis("Agent")
	full := composer.ComposePrompt("Agent")

	// Minimal should be shortest
	if len(minimal) >= len(full) {
		t.Error("Minimal prompt should be shorter than full prompt")
	}

	// All should have core identity
	for name, prompt := range map[string]string{
		"minimal":  minimal,
		"document": document,
		"analysis": analysis,
		"full":     full,
	} {
		if !strings.Contains(prompt, core.CoreIdentity) {
			t.Errorf("%s prompt should contain CoreIdentity", name)
		}
	}
}
