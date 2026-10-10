package agents

import (
	"fmt"
	"strings"

	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/signal"
)

// BuildSignalAnnotation produces a model-agnostic signal annotation for the system prompt.
// Designed to work with ANY LLM — local 7B models, Groq, Anthropic, OpenAI, etc.
// The annotation is explicit enough that even a weak model can follow the structure.
//
// This is ADDITIVE — it tells the LLM what we detected and what data it has.
// The LLM decides what to do with it. No gating, no filtering, no routing.
func BuildSignalAnnotation(envelope *signal.SignalEnvelope, tieredCtx *services.TieredContext) string {
	if envelope == nil {
		return ""
	}

	var sb strings.Builder

	// 1. Signal classification — what kind of message this is
	sb.WriteString("## SIGNAL DETECTION\n")
	sb.WriteString(fmt.Sprintf("**Type**: %s (%s)\n", envelope.Genre, describeGenre(envelope.Genre)))
	if envelope.DocType != "" {
		sb.WriteString(fmt.Sprintf("**Document**: %s detected — structure template provided below\n", envelope.DocType))
	}

	// 2. Available context inventory — what data the LLM has to work with.
	// Not filtering. Just telling the model: "here's what's loaded, use what you need."
	if tieredCtx != nil && tieredCtx.Level1 != nil {
		inventory := buildContextInventory(tieredCtx.Level1)
		if inventory != "" {
			sb.WriteString("\n### Available Context\n")
			sb.WriteString(inventory)
		}
	}

	// 3. Structure template — if document type detected, give detailed per-section guidance.
	// Explicit enough that a local model can follow it without "knowing" the genre.
	if envelope.DocType != "" {
		if template, ok := GenreStructureTemplates[envelope.DocType]; ok {
			sb.WriteString("\n### Document Structure\n")
			sb.WriteString(template)
			sb.WriteString("\n")
		}
	}

	// 4. Writing style — always included, genre-specific.
	// Tells the model HOW to write, not just WHAT sections to include.
	style := getWritingStyle(envelope.Genre, envelope.DocType)
	if style != "" {
		sb.WriteString("\n### Writing Style\n")
		sb.WriteString(style)
		sb.WriteString("\n")
	}

	return sb.String()
}

// buildContextInventory lists what context data is available to the LLM.
// This is informational — helps any model (especially weaker ones) know what to reference.
func buildContextInventory(l1 *services.FullContext) string {
	var lines []string

	if l1.Project != nil && l1.Project.Name != "" {
		line := fmt.Sprintf("- **Project**: %s (status: %s", l1.Project.Name, l1.Project.Status)
		if l1.Project.Priority != "" {
			line += fmt.Sprintf(", priority: %s", l1.Project.Priority)
		}
		line += ")"
		lines = append(lines, line)
	}

	if l1.LinkedClient != nil && l1.LinkedClient.Name != "" {
		line := fmt.Sprintf("- **Client**: %s", l1.LinkedClient.Name)
		if l1.LinkedClient.Industry != "" {
			line += fmt.Sprintf(" (%s)", l1.LinkedClient.Industry)
		}
		lines = append(lines, line)
	}

	if len(l1.Tasks) > 0 {
		lines = append(lines, fmt.Sprintf("- **Tasks**: %d active tasks loaded", len(l1.Tasks)))
	}

	if len(l1.TeamMembers) > 0 {
		lines = append(lines, fmt.Sprintf("- **Team**: %d members loaded", len(l1.TeamMembers)))
	}

	if len(l1.Contexts) > 0 {
		lines = append(lines, fmt.Sprintf("- **Context profiles**: %d loaded", len(l1.Contexts)))
	}

	if len(l1.Documents) > 0 {
		lines = append(lines, fmt.Sprintf("- **Documents**: %d attached", len(l1.Documents)))
	}

	if len(l1.RelevantRAG) > 0 {
		lines = append(lines, fmt.Sprintf("- **Knowledge**: %d relevant blocks from RAG", len(l1.RelevantRAG)))
	}

	if len(l1.Memories) > 0 {
		lines = append(lines, fmt.Sprintf("- **Memories**: %d workspace memories", len(l1.Memories)))
	}

	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n") + "\n"
}

// describeGenre returns a plain-English description of the genre.
// Helps weaker models understand what the genre means without prior training on our taxonomy.
func describeGenre(genre signal.Genre) string {
	switch genre {
	case signal.GenreDirect:
		return "user wants something created or done"
	case signal.GenreInform:
		return "user is asking a question or seeking information"
	case signal.GenreCommit:
		return "user is making a commitment or planning an action"
	case signal.GenreDecide:
		return "user needs help choosing between options"
	case signal.GenreExpress:
		return "user is expressing a feeling or giving feedback"
	default:
		return "general message"
	}
}

// getWritingStyle returns explicit writing guidance for any model.
// This compensates for models that don't inherently know genre conventions.
func getWritingStyle(genre signal.Genre, docType string) string {
	// Document-specific style overrides genre default
	if docType != "" {
		switch docType {
		case "proposal":
			return `- Professional, confident tone. Write as the user's organization addressing the client.
- Use active voice and concrete language. No hedging ("we believe", "perhaps").
- Reference specific data from the loaded context — project details, client info, tasks, knowledge blocks.
- Numbers and specifics over generalities. If budget data exists, use it. If not, provide a realistic framework.
- Minimum 1,500 words. Each section should be substantive, not placeholder text.`
		case "sop":
			return `- Imperative voice ("Do X", not "You should do X").
- Every step must be a single concrete action that can be checked off.
- Include expected outcomes after key steps so the reader can verify.
- Use numbered lists for sequences, bullet lists for non-ordered items.
- If team members are loaded, assign roles by name or function.`
		case "report":
			return `- Lead with findings, not methodology. The reader wants answers first.
- Use tables for any comparisons (3+ items). Use bullet points for key takeaways.
- Reference specific data from knowledge blocks and documents.
- Include "So what?" after each finding — why does this matter?
- Executive summary must stand alone — someone reading only that section gets the full picture.`
		case "brief":
			return `- Maximum 1,000 words. Every sentence must earn its place.
- Decision-oriented: what needs to happen and why.
- Use bold for key terms and recommendations.
- One page when printed. If it's longer, cut.`
		case "framework":
			return `- Abstract enough to be reusable, concrete enough to be actionable.
- Include a text description of the visual model (boxes, arrows, relationships).
- Each component should have: definition, purpose, inputs, outputs.
- End with a practical application example using the loaded context.`
		case "guide":
			return `- Write for someone doing this for the first time.
- Every instruction should be copy-pasteable or directly actionable.
- Include what success looks like after each major step.
- Anticipate common mistakes and address them inline.`
		case "plan":
			return `- Concrete timelines with dates, not vague phases.
- Each milestone has: deliverable, owner (from team if loaded), dependency, and success criteria.
- Include a risk table: risk, likelihood, impact, mitigation.
- Use the project and task data to ground the plan in reality.`
		}
	}

	// Genre-level style (non-document or unknown document type)
	switch genre {
	case signal.GenreDirect:
		return `- Lead with the action. First sentence should be what you're doing or creating.
- If creating a document, follow the structure template above.
- If executing a task, confirm what you'll do, then do it.
- Use data from loaded context — don't make things up.`
	case signal.GenreInform:
		return `- Answer the question directly in the first sentence.
- Then provide supporting evidence from loaded context and knowledge.
- Use tables for comparisons, bullet points for lists.
- Cite specific sources when referencing context data.`
	case signal.GenreCommit:
		return `- State the commitment clearly: what, who, when.
- List dependencies and prerequisites.
- Identify risks to delivery.
- Use task and project data to ground commitments.`
	case signal.GenreDecide:
		return `- State your recommendation first, then justify.
- Compare options in a table: option, pros, cons, effort, risk.
- Use loaded metrics and data to support the recommendation.
- End with clear next steps for the chosen option.`
	case signal.GenreExpress:
		return `- Acknowledge the sentiment before responding to content.
- Be direct but empathetic.
- Pivot to constructive action: what can be done next.`
	default:
		return ""
	}
}
