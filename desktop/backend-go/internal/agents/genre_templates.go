package agents

import (
	"github.com/rhl/businessos-backend/internal/signal"
)

// GenreStructureTemplates maps document types to their structural skeletons.
// Used by GenreEnricher to inject structure guidance into agent prompts.
var GenreStructureTemplates = map[string]string{
	"proposal": `Follow this proposal structure:
1. **Executive Summary** - Key points in 2-3 sentences
2. **Opportunity** - Problem/need being addressed
3. **Solution** - Proposed approach with specifics
4. **Implementation** - Timeline, milestones, phases
5. **Investment** - Cost breakdown, ROI projections
6. **Team** - Key personnel and qualifications
7. **Next Steps** - Clear action items with owners and dates

Minimum 1,500 words. Use real data from context. No placeholders.`,

	"sop": `Follow this SOP structure:
1. **Purpose** - Why this procedure exists
2. **Scope** - What it covers and what it doesn't
3. **Roles & Responsibilities** - Who does what
4. **Procedure** - Numbered steps with clear instructions
5. **Quality Checks** - Verification points
6. **Revision History** - Version tracking

Use imperative voice. Each step must be actionable.`,

	"report": `Follow this report structure:
1. **Executive Summary** - Key findings in 3-5 bullets
2. **Key Findings** - Detailed analysis with data
3. **Data Analysis** - Charts, tables, trends
4. **Recommendations** - Actionable next steps
5. **Appendix** - Supporting data and methodology

Lead with insights, not methodology. Use tables for comparisons.`,

	"brief": `Follow this brief structure:
1. **Objective** - What decision is needed
2. **Background** - Essential context only
3. **Key Points** - 3-5 critical items
4. **Recommendation** - Clear position with rationale
5. **Next Steps** - Immediate actions

Keep under 1,000 words. Decision-oriented.`,

	"framework": `Follow this framework structure:
1. **Problem Statement** - What gap this framework addresses
2. **Framework Components** - Core elements with relationships
3. **Application Guide** - How to apply in practice
4. **Evaluation Criteria** - How to measure effectiveness

Include a visual model description. Make it actionable.`,

	"guide": `Follow this guide structure:
1. **Overview** - What you'll learn
2. **Prerequisites** - What's needed before starting
3. **Step-by-Step Instructions** - Detailed walkthrough
4. **Common Issues** - Troubleshooting section
5. **Next Steps** - Where to go from here

Use examples throughout. Include code/config samples where relevant.`,

	"plan": `Follow this plan structure:
1. **Objective** - What success looks like
2. **Current State** - Where we are now
3. **Strategy** - How we'll get there
4. **Timeline** - Milestones with dates
5. **Resources** - People, tools, budget
6. **Risks** - What could go wrong and mitigations
7. **Success Criteria** - How we'll measure

Include dependencies between milestones.`,
}

// GetGenreStructureHint returns a genre-specific structural guidance string
// for non-document genres (DIRECT, INFORM, COMMIT, DECIDE, EXPRESS).
func GetGenreStructureHint(genre signal.Genre) string {
	switch genre {
	case signal.GenreDirect:
		return `## OUTPUT STRUCTURE GUIDANCE
Format as numbered action items with owners and deadlines where applicable.
Use imperative voice. Lead with the most important action.
If creating a document, follow the appropriate document template.`

	case signal.GenreInform:
		return `## OUTPUT STRUCTURE GUIDANCE
Lead with the key insight or answer first, then provide supporting evidence.
Use tables for comparisons. Use bullet points for lists.
Cite specific data from context when available.`

	case signal.GenreCommit:
		return `## OUTPUT STRUCTURE GUIDANCE
Structure as a commitment summary:
- **What**: Specific deliverable
- **Who**: Responsible parties
- **When**: Timeline/deadline
- **Dependencies**: What must happen first
- **Risks**: What could prevent delivery`

	case signal.GenreDecide:
		return `## OUTPUT STRUCTURE GUIDANCE
Lead with the recommended decision, then justify.
Structure as:
1. Recommendation (clear position)
2. Criteria used for evaluation
3. Alternatives considered with pros/cons
4. Impact analysis
5. Implementation path for chosen option`

	case signal.GenreExpress:
		return `## OUTPUT STRUCTURE GUIDANCE
Acknowledge the expressed sentiment first.
Then provide constructive perspective with concrete next steps.
Be empathetic but action-oriented.`

	default:
		return ""
	}
}
