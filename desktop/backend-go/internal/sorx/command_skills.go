// Package sorx provides command-based skills that migrate from the legacy command system.
// These skills wrap the old command patterns into proper Sorx skill definitions.
package sorx

// RegisterCommandSkills registers all skills that were migrated from commands.
// These are Tier 3-4 skills as they all involve AI reasoning/generation.
func RegisterCommandSkills(e *Engine) {
	// ========================================================================
	// GENERAL SKILLS (from general commands)
	// ========================================================================

	e.RegisterSkill(&SkillDefinition{
		ID:          "general.analyze",
		Name:        "Analyze Content",
		Description: "Deep analysis of content, data, or patterns with structured insights",
		Category:    "general",
		Tier:        TierReasoningAI,
		RoleAffinity: []Role{RoleAny, RoleAnalyst, RoleExecutive},
		DataPointsSatisfied: []string{"analysis.completed"},
		RequiresApprovalAt: TemperatureCold,
		Steps: []Step{
			{
				ID:        "gather_context",
				Type:      StepTypeAction,
				Action:    "businessos.gather_context",
				Params:    map[string]interface{}{"sources": []string{"documents", "conversations", "artifacts"}},
			},
			{
				ID:        "analyze",
				Type:      StepTypeAgent,
				AgentType: "analyst",
				AgentPrompt: `Perform a deep analysis of the provided content and context.

ANALYSIS FRAMEWORK:
1. **Overview**: Provide a high-level summary of what you're analyzing
2. **Key Findings**: Identify the most important patterns, trends, or insights
3. **Deep Dive**: Examine specific details that warrant attention
4. **Implications**: What do these findings mean for the user?
5. **Recommendations**: Based on your analysis, what actions should be considered?

Be thorough, objective, and data-driven in your analysis. Support conclusions with evidence from the provided context.`,
			},
		},
	})

	e.RegisterSkill(&SkillDefinition{
		ID:          "general.summarize",
		Name:        "Summarize Content",
		Description: "Create concise summaries that capture essential information",
		Category:    "general",
		Tier:        TierStructuredAI,
		RoleAffinity: []Role{RoleAny},
		DataPointsSatisfied: []string{"summary.created"},
		RequiresApprovalAt: TemperatureCold,
		Steps: []Step{
			{
				ID:        "gather_context",
				Type:      StepTypeAction,
				Action:    "businessos.gather_context",
				Params:    map[string]interface{}{"sources": []string{"documents", "conversations", "artifacts"}},
			},
			{
				ID:        "summarize",
				Type:      StepTypeAgent,
				AgentType: "document",
				AgentPrompt: `Create a clear, concise summary that captures essential information.

SUMMARY STRUCTURE:
- **Executive Summary**: 2-3 sentence overview
- **Key Points**: Bullet points of the most important information
- **Details**: Brief elaboration on significant items if needed
- **Action Items**: Any tasks or next steps identified

Keep summaries focused and actionable. Prioritize information by relevance and importance.`,
			},
		},
	})

	e.RegisterSkill(&SkillDefinition{
		ID:          "general.explain",
		Name:        "Explain Concepts",
		Description: "Explain concepts, code, or content clearly at appropriate complexity",
		Category:    "general",
		Tier:        TierReasoningAI,
		RoleAffinity: []Role{RoleAny, RoleDocument},
		DataPointsSatisfied: []string{"explanation.provided"},
		Steps: []Step{
			{
				ID:        "explain",
				Type:      StepTypeAgent,
				AgentType: "document",
				AgentPrompt: `Explain the requested concept or content clearly.

EXPLANATION APPROACH:
1. Start with a simple, jargon-free overview
2. Build up complexity gradually
3. Use analogies and examples when helpful
4. Anticipate and address common questions
5. Connect to practical applications

Adapt your explanation to the apparent expertise level of the question. For technical topics, include both conceptual understanding and practical details.`,
			},
		},
	})

	e.RegisterSkill(&SkillDefinition{
		ID:          "general.review",
		Name:        "Review Content",
		Description: "Review and provide constructive feedback on content",
		Category:    "general",
		Tier:        TierReasoningAI,
		RoleAffinity: []Role{RoleAny, RoleAnalyst},
		DataPointsSatisfied: []string{"review.completed"},
		Steps: []Step{
			{
				ID:        "review",
				Type:      StepTypeAgent,
				AgentType: "analyst",
				AgentPrompt: `Provide expert review and constructive feedback.

REVIEW FRAMEWORK:
1. **Strengths**: What's working well
2. **Areas for Improvement**: Specific, actionable suggestions
3. **Critical Issues**: Any problems that need immediate attention
4. **Recommendations**: Prioritized list of improvements
5. **Summary**: Overall assessment

Be constructive, specific, and balanced in your feedback. Provide examples and explanations for suggestions.`,
			},
		},
	})

	e.RegisterSkill(&SkillDefinition{
		ID:          "general.brainstorm",
		Name:        "Brainstorm Ideas",
		Description: "Generate creative ideas and explore possibilities",
		Category:    "creative",
		Tier:        TierGenerativeAI,
		RoleAffinity: []Role{RoleAny, RoleMarketing, RoleExecutive},
		DataPointsSatisfied: []string{"brainstorm.completed"},
		Steps: []Step{
			{
				ID:        "brainstorm",
				Type:      StepTypeAgent,
				AgentType: "orchestrator",
				AgentPrompt: `Generate innovative ideas as a creative brainstorming partner.

BRAINSTORMING APPROACH:
1. Generate multiple diverse ideas (aim for 5-10)
2. Include both conventional and unconventional options
3. Build on provided context and constraints
4. Consider different perspectives and approaches
5. Briefly explain the rationale for each idea

Don't filter ideas too early - include creative possibilities even if they seem ambitious. Group related ideas together.`,
			},
		},
	})

	e.RegisterSkill(&SkillDefinition{
		ID:          "general.compare",
		Name:        "Compare Options",
		Description: "Compare documents, options, or data with structured analysis",
		Category:    "general",
		Tier:        TierReasoningAI,
		RoleAffinity: []Role{RoleAny, RoleAnalyst},
		DataPointsSatisfied: []string{"comparison.completed"},
		Steps: []Step{
			{
				ID:        "compare",
				Type:      StepTypeAgent,
				AgentType: "analyst",
				AgentPrompt: `Create a comprehensive comparison analysis.

COMPARISON FRAMEWORK:
1. **Overview**: What is being compared
2. **Criteria**: Key dimensions for comparison
3. **Side-by-Side**: Clear comparison table/list
4. **Analysis**: Key differences and similarities
5. **Recommendation**: Which option is best for what scenario

Present comparisons in a clear, scannable format. Highlight the most important differences.`,
			},
		},
	})

	// ========================================================================
	// BUSINESS SKILLS (from business commands)
	// ========================================================================

	e.RegisterSkill(&SkillDefinition{
		ID:          "business.proposal",
		Name:        "Generate Proposal",
		Description: "Generate professional business proposals from context",
		Category:    "business",
		Tier:        TierReasoningAI,
		RoleAffinity: []Role{RoleSales, RoleExecutive},
		DataPointsSatisfied: []string{"proposal.drafted"},
		RequiresApprovalAt: TemperatureWarm, // Proposals should be reviewed before sending
		Steps: []Step{
			{
				ID:        "gather_context",
				Type:      StepTypeAction,
				Action:    "businessos.gather_context",
				Params:    map[string]interface{}{"sources": []string{"documents", "clients", "projects", "conversations"}},
			},
			{
				ID:        "generate_proposal",
				Type:      StepTypeAgent,
				AgentType: "document",
				AgentPrompt: `Create a professional business proposal.

PROPOSAL STRUCTURE:
1. **Executive Summary**: Brief overview of the proposal
2. **Understanding**: Demonstrate understanding of the client's needs
3. **Proposed Solution**: Clear description of what you're proposing
4. **Approach/Methodology**: How you'll deliver the solution
5. **Timeline**: Key milestones and deliverables
6. **Investment**: Pricing and terms (if applicable)
7. **Next Steps**: Clear call to action

Use professional language, be specific about deliverables, and reference relevant context from the project/client data.`,
			},
			{
				ID:               "review_proposal",
				Type:             StepTypeDecision,
				RequiresDecision: true,
				DecisionQuestion: "Review the generated proposal before sending",
				DecisionOptions:  []string{"approve", "edit", "reject"},
				InputFields: map[string]InputField{
					"feedback": {
						Type:        "text",
						Label:       "Feedback or edits",
						Required:    false,
						Placeholder: "Any changes needed?",
					},
				},
			},
		},
	})

	e.RegisterSkill(&SkillDefinition{
		ID:          "business.report",
		Name:        "Create Business Report",
		Description: "Create comprehensive business reports from data and context",
		Category:    "business",
		Tier:        TierReasoningAI,
		RoleAffinity: []Role{RoleAnalyst, RoleExecutive, RoleFinance},
		DataPointsSatisfied: []string{"report.generated"},
		Steps: []Step{
			{
				ID:        "gather_data",
				Type:      StepTypeAction,
				Action:    "businessos.gather_context",
				Params:    map[string]interface{}{"sources": []string{"documents", "projects", "conversations", "artifacts"}},
			},
			{
				ID:        "generate_report",
				Type:      StepTypeAgent,
				AgentType: "analyst",
				AgentPrompt: `Create a comprehensive business report.

REPORT STRUCTURE:
1. **Title & Date**
2. **Executive Summary**: Key findings and recommendations
3. **Background**: Context and purpose of the report
4. **Methodology**: How data was gathered/analyzed
5. **Findings**: Detailed results with supporting data
6. **Analysis**: Interpretation of findings
7. **Recommendations**: Actionable next steps
8. **Appendix**: Supporting data if needed

Use data from the provided context. Include specific numbers and metrics where available.`,
			},
		},
	})

	e.RegisterSkill(&SkillDefinition{
		ID:          "business.email",
		Name:        "Draft Email",
		Description: "Draft professional emails based on context",
		Category:    "communication",
		Tier:        TierStructuredAI,
		RoleAffinity: []Role{RoleAny, RoleSales, RoleSupport},
		DataPointsSatisfied: []string{"email.drafted"},
		RequiresApprovalAt: TemperatureWarm, // External communications need review
		RequiredIntegrations: []string{}, // Can draft without gmail connected
		Steps: []Step{
			{
				ID:        "gather_context",
				Type:      StepTypeAction,
				Action:    "businessos.gather_context",
				Params:    map[string]interface{}{"sources": []string{"conversations", "clients", "projects"}},
			},
			{
				ID:        "draft_email",
				Type:      StepTypeAgent,
				AgentType: "document",
				AgentPrompt: `Draft a professional email communication.

EMAIL PRINCIPLES:
1. Clear subject line that summarizes the email
2. Appropriate greeting based on relationship
3. Concise, well-structured body
4. Clear call to action or next steps
5. Professional closing

Adapt tone based on the context (formal for clients, friendly for team). Reference relevant details from the provided context.`,
			},
			{
				ID:               "review_email",
				Type:             StepTypeDecision,
				RequiresDecision: true,
				DecisionQuestion: "Review the email before sending",
				DecisionOptions:  []string{"send", "edit", "cancel"},
				InputFields: map[string]InputField{
					"to": {
						Type:     "text",
						Label:    "Recipient",
						Required: true,
					},
					"subject": {
						Type:     "text",
						Label:    "Subject",
						Required: true,
					},
				},
			},
		},
	})

	e.RegisterSkill(&SkillDefinition{
		ID:          "business.meeting_notes",
		Name:        "Meeting Notes",
		Description: "Create meeting notes or agenda from context",
		Category:    "productivity",
		Tier:        TierStructuredAI,
		RoleAffinity: []Role{RoleAny, RoleOperations},
		DataPointsSatisfied: []string{"meeting_notes.created"},
		Steps: []Step{
			{
				ID:        "gather_context",
				Type:      StepTypeAction,
				Action:    "businessos.gather_context",
				Params:    map[string]interface{}{"sources": []string{"conversations", "documents", "projects"}},
			},
			{
				ID:        "generate_notes",
				Type:      StepTypeAgent,
				AgentType: "document",
				AgentPrompt: `Create clear meeting documentation.

For MEETING NOTES:
- Date, attendees, and purpose
- Key discussion points
- Decisions made
- Action items with owners and deadlines
- Next steps

For MEETING AGENDA:
- Meeting objective
- Agenda items with time allocations
- Required preparation
- Expected outcomes

Extract relevant information from the provided context to populate the notes/agenda.`,
			},
			{
				ID:     "save_to_daily_log",
				Type:   StepTypeAction,
				Action: "businessos.create_daily_log",
				Params: map[string]interface{}{"from": "generate_notes", "type": "meeting_notes"},
			},
		},
	})

	e.RegisterSkill(&SkillDefinition{
		ID:          "business.swot",
		Name:        "SWOT Analysis",
		Description: "Create SWOT analysis from context",
		Category:    "analysis",
		Tier:        TierReasoningAI,
		RoleAffinity: []Role{RoleAnalyst, RoleExecutive, RoleSales},
		DataPointsSatisfied: []string{"swot.completed"},
		Steps: []Step{
			{
				ID:        "gather_context",
				Type:      StepTypeAction,
				Action:    "businessos.gather_context",
				Params:    map[string]interface{}{"sources": []string{"documents", "projects", "clients"}},
			},
			{
				ID:        "generate_swot",
				Type:      StepTypeAgent,
				AgentType: "analyst",
				AgentPrompt: `Perform a SWOT analysis.

SWOT FRAMEWORK:
**Strengths** (Internal, Positive)
- What advantages exist?
- What is done well?
- What unique resources are available?

**Weaknesses** (Internal, Negative)
- What could be improved?
- What should be avoided?
- What limitations exist?

**Opportunities** (External, Positive)
- What trends could be leveraged?
- What opportunities are emerging?
- What could be done that isn't being done?

**Threats** (External, Negative)
- What obstacles exist?
- What is the competition doing?
- What risks are present?

Provide specific, actionable insights based on the provided context.`,
			},
		},
	})

	e.RegisterSkill(&SkillDefinition{
		ID:          "business.forecast",
		Name:        "Generate Forecast",
		Description: "Generate forecasts from historical data",
		Category:    "analysis",
		Tier:        TierReasoningAI,
		RoleAffinity: []Role{RoleAnalyst, RoleFinance, RoleExecutive},
		DataPointsSatisfied: []string{"forecast.generated"},
		Steps: []Step{
			{
				ID:        "gather_data",
				Type:      StepTypeAction,
				Action:    "businessos.gather_context",
				Params:    map[string]interface{}{"sources": []string{"documents", "projects"}},
			},
			{
				ID:        "generate_forecast",
				Type:      StepTypeAgent,
				AgentType: "analyst",
				AgentPrompt: `Create a data-driven forecast.

FORECAST APPROACH:
1. **Current State**: Summary of historical data
2. **Trends**: Key patterns identified
3. **Assumptions**: What assumptions underlie the forecast
4. **Projections**: Detailed forecasts with ranges
5. **Scenarios**: Best/expected/worst case
6. **Risks**: Factors that could affect accuracy
7. **Recommendations**: Actions based on forecast

Be clear about confidence levels and the basis for projections.`,
			},
		},
	})

	e.RegisterSkill(&SkillDefinition{
		ID:          "business.pitch",
		Name:        "Create Pitch Deck",
		Description: "Create pitch deck content from context",
		Category:    "business",
		Tier:        TierReasoningAI,
		RoleAffinity: []Role{RoleSales, RoleExecutive, RoleMarketing},
		DataPointsSatisfied: []string{"pitch.created"},
		RequiresApprovalAt: TemperatureWarm,
		Steps: []Step{
			{
				ID:        "gather_context",
				Type:      StepTypeAction,
				Action:    "businessos.gather_context",
				Params:    map[string]interface{}{"sources": []string{"projects", "clients", "documents"}},
			},
			{
				ID:        "generate_pitch",
				Type:      StepTypeAgent,
				AgentType: "document",
				AgentPrompt: `Create compelling pitch presentation content.

PITCH STRUCTURE:
1. **Hook**: Attention-grabbing opening
2. **Problem**: What pain point are you solving?
3. **Solution**: Your unique approach
4. **Value Proposition**: Why choose this solution?
5. **How It Works**: Brief explanation
6. **Traction/Proof**: Evidence of success
7. **Team**: Why you're qualified (if applicable)
8. **Ask**: What do you need?

Create slide-by-slide content with key points and talking notes. Keep each slide focused on one main idea.`,
			},
		},
	})

	// ========================================================================
	// CREATIVE SKILLS
	// ========================================================================

	e.RegisterSkill(&SkillDefinition{
		ID:          "creative.generate",
		Name:        "Generate Content",
		Description: "Generate high-quality content based on context and requirements",
		Category:    "creative",
		Tier:        TierGenerativeAI,
		RoleAffinity: []Role{RoleAny, RoleMarketing, RoleDocument},
		DataPointsSatisfied: []string{"content.generated"},
		Steps: []Step{
			{
				ID:        "gather_context",
				Type:      StepTypeAction,
				Action:    "businessos.gather_context",
				Params:    map[string]interface{}{"sources": []string{"documents", "conversations", "artifacts"}},
			},
			{
				ID:        "generate",
				Type:      StepTypeAgent,
				AgentType: "document",
				AgentPrompt: `Create high-quality content based on the provided context and requirements.

GENERATION PRINCIPLES:
- Match the tone and style appropriate for the use case
- Ensure accuracy when referencing provided context
- Be creative while staying relevant
- Structure content logically
- Make content actionable when appropriate

Consider the context provided to inform your generation. Reference specific details from the context when relevant.`,
			},
		},
	})

	// ========================================================================
	// TASK SKILLS
	// ========================================================================

	e.RegisterSkill(&SkillDefinition{
		ID:          "task.create_from_input",
		Name:        "Create Tasks",
		Description: "Parse input and create actionable tasks",
		Category:    "productivity",
		Tier:        TierStructuredAI,
		RoleAffinity: []Role{RoleAny, RoleOperations},
		DataPointsSatisfied: []string{"tasks.created"},
		Steps: []Step{
			{
				ID:        "parse_input",
				Type:      StepTypeAgent,
				AgentType: "task",
				AgentPrompt: `Parse the input to identify and create clear, actionable tasks.

TASK CREATION:
1. Parse the user's input to identify distinct tasks
2. For each task, provide:
   - Clear title (action-oriented, starts with verb)
   - Brief description if needed
   - Priority suggestion (high/medium/low)
   - Any relevant tags or categories
3. Group related tasks together
4. Identify dependencies between tasks if any

Format tasks as JSON for easy processing.`,
			},
			{
				ID:     "create_tasks",
				Type:   StepTypeAction,
				Action: "businessos.create_tasks",
				Params: map[string]interface{}{"from": "parse_input"},
			},
		},
	})
}
