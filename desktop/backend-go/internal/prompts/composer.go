package prompts

import (
	"strings"

	"github.com/rhl/businessos-backend/internal/prompts/core"
)

// PromptLayer represents a layer of the prompt system
type PromptLayer string

const (
	LayerIdentity      PromptLayer = "identity"
	LayerFormatting    PromptLayer = "formatting"
	LayerArtifacts     PromptLayer = "artifacts"
	LayerContext       PromptLayer = "context"
	LayerTools         PromptLayer = "tools"
	LayerErrors        PromptLayer = "errors"
	LayerAgentSpecific PromptLayer = "agent_specific"
)

// PromptComposer assembles prompts from layers
type PromptComposer struct {
	layers map[PromptLayer]string
}

// NewPromptComposer creates a new prompt composer with core layers
func NewPromptComposer() *PromptComposer {
	return &PromptComposer{
		layers: map[PromptLayer]string{
			LayerIdentity:   core.CoreIdentity,
			LayerFormatting: core.OutputFormattingStandards,
			LayerArtifacts:  core.ArtifactSystem,
			LayerContext:    core.ContextIntegration,
			LayerTools:      core.ToolUsagePatterns,
			LayerErrors:     core.ErrorHandling,
		},
	}
}

// ComposePrompt assembles a complete prompt from layers
func (c *PromptComposer) ComposePrompt(agentPrompt string, includeLayers ...PromptLayer) string {
	var parts []string

	if len(includeLayers) == 0 {
		includeLayers = []PromptLayer{
			LayerIdentity,
			LayerFormatting,
			LayerArtifacts,
			LayerContext,
			LayerTools,
			LayerErrors,
		}
	}

	for _, layer := range includeLayers {
		if content, ok := c.layers[layer]; ok {
			parts = append(parts, content)
		}
	}

	if agentPrompt != "" {
		parts = append(parts, agentPrompt)
	}

	return strings.Join(parts, "\n\n---\n\n")
}

// ComposeWithContext adds dynamic context to the prompt
func (c *PromptComposer) ComposeWithContext(agentPrompt string, userName string, projectName string, projectDesc string) string {
	basePrompt := c.ComposePrompt(agentPrompt)

	var contextParts []string

	if userName != "" {
		contextParts = append(contextParts, "**User:** "+userName)
	}

	if projectName != "" {
		contextParts = append(contextParts, "**Active Project:** "+projectName)
		if projectDesc != "" {
			contextParts = append(contextParts, projectDesc)
		}
	}

	if len(contextParts) > 0 {
		dynamicContext := "## CURRENT SESSION CONTEXT\n\n" + strings.Join(contextParts, "\n")
		return basePrompt + "\n\n---\n\n" + dynamicContext
	}

	return basePrompt
}

// ComposeMinimal creates a minimal prompt
func (c *PromptComposer) ComposeMinimal(agentPrompt string) string {
	return c.ComposePrompt(agentPrompt, LayerIdentity, LayerFormatting)
}

// ComposeForDocument creates a prompt optimized for document creation
func (c *PromptComposer) ComposeForDocument(agentPrompt string) string {
	return c.ComposePrompt(agentPrompt, LayerIdentity, LayerFormatting, LayerArtifacts, LayerContext)
}

// ComposeForAnalysis creates a prompt optimized for analysis
func (c *PromptComposer) ComposeForAnalysis(agentPrompt string) string {
	return c.ComposePrompt(agentPrompt, LayerIdentity, LayerFormatting, LayerContext, LayerTools)
}

// ComposeForProject creates a prompt optimized for project management
func (c *PromptComposer) ComposeForProject(agentPrompt string) string {
	return c.ComposePrompt(agentPrompt, LayerIdentity, LayerFormatting, LayerArtifacts, LayerContext, LayerTools)
}

// ComposeForClient creates a prompt optimized for client management
func (c *PromptComposer) ComposeForClient(agentPrompt string) string {
	return c.ComposePrompt(agentPrompt, LayerIdentity, LayerFormatting, LayerContext)
}

// GetLayer returns a specific layer content
func (c *PromptComposer) GetLayer(layer PromptLayer) string {
	return c.layers[layer]
}

// SetLayer allows overriding a layer
func (c *PromptComposer) SetLayer(layer PromptLayer, content string) {
	c.layers[layer] = content
}

// DefaultComposer is the default prompt composer instance
var DefaultComposer = NewPromptComposer()

// Compose is a convenience function using the default composer
func Compose(agentPrompt string, layers ...PromptLayer) string {
	return DefaultComposer.ComposePrompt(agentPrompt, layers...)
}

// ComposeWithUserContext is a convenience function for adding user context
func ComposeWithUserContext(agentPrompt string, userName string, projectName string, projectDesc string) string {
	return DefaultComposer.ComposeWithContext(agentPrompt, userName, projectName, projectDesc)
}
