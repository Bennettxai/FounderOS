package sorx

func (e *Engine) registerBuiltinSkills() {
	// ========================================================================
	// Register command-based skills (migrated from legacy commands)
	// ========================================================================
	RegisterCommandSkills(e)

	// ========================================================================
	// INTEGRATION SKILLS
	// ========================================================================

	// Email processing skill - Tier 3 (Reasoning AI)
	e.RegisterSkill(&SkillDefinition{
		ID:                   "email.process_inbox",
		Name:                 "Process Email Inbox",
		Description:          "Scans inbox and extracts actionable items using AI analysis",
		Category:             "communication",
		Tier:                 TierReasoningAI,
		RoleAffinity:         []Role{RoleAny, RoleOperations},
		RequiredIntegrations: []string{"gmail"},
		DataPointsSatisfied:  []string{"inbox.processed", "tasks.extracted"},
		RequiresApprovalAt:   TemperatureWarm,
		Steps: []Step{
			{
				ID:          "fetch_emails",
				Type:        StepTypeAction,
				Action:      "gmail.list_messages",
				Integration: "gmail",
				Params:      map[string]interface{}{"max_results": 50, "label": "INBOX"},
			},
			{
				ID:     "analyze_with_agent",
				Type:   StepTypeAction,
				Action: "agent.analyst",
				Params: map[string]interface{}{
					"task": "Analyze these emails and extract: 1) Action items that should become tasks 2) Important dates/deadlines 3) Key information to remember. Format as structured JSON.",
					"from": "fetch_emails",
				},
			},
			{
				ID:     "create_tasks",
				Type:   StepTypeAction,
				Action: "businessos.create_tasks",
				Params: map[string]interface{}{"from": "analyze_with_agent"},
			},
		},
	})

	// CRM sync skill
	e.RegisterSkill(&SkillDefinition{
		ID:                   "crm.sync_contacts",
		Name:                 "Sync CRM Contacts",
		Description:          "Syncs contacts from CRM to BusinessOS",
		Category:             "crm",
		RequiredIntegrations: []string{"hubspot"},
		Steps: []Step{
			{
				ID:          "fetch_contacts",
				Type:        StepTypeAction,
				Action:      "hubspot.list_contacts",
				Integration: "hubspot",
			},
			{
				ID:     "map_contacts",
				Type:   StepTypeAction,
				Action: "transform.map_fields",
				Params: map[string]interface{}{"mapping": "hubspot_to_client"},
			},
			{
				ID:     "upsert_clients",
				Type:   StepTypeAction,
				Action: "businessos.upsert_clients",
			},
		},
	})

	// Task sync skill with decision
	e.RegisterSkill(&SkillDefinition{
		ID:                   "tasks.import_with_review",
		Name:                 "Import Tasks with Review",
		Description:          "Imports tasks from external source with human review",
		Category:             "tasks",
		RequiredIntegrations: []string{},
		Steps: []Step{
			{
				ID:     "fetch_tasks",
				Type:   StepTypeAction,
				Action: "linear.list_issues",
			},
			{
				ID:               "review_tasks",
				Type:             StepTypeDecision,
				RequiresDecision: true,
				DecisionQuestion: "Which tasks should be imported?",
				DecisionOptions:  []string{"Import all", "Import assigned only", "Skip"},
				Priority:         "medium",
			},
			{
				ID:     "import_tasks",
				Type:   StepTypeAction,
				Action: "businessos.import_tasks",
			},
		},
	})

	// Calendar sync skill
	e.RegisterSkill(&SkillDefinition{
		ID:                   "calendar.sync_events",
		Name:                 "Sync Calendar Events",
		Description:          "Syncs calendar events and creates daily log entries",
		Category:             "calendar",
		RequiredIntegrations: []string{"google_calendar"},
		Steps: []Step{
			{
				ID:          "fetch_events",
				Type:        StepTypeAction,
				Action:      "google_calendar.list_events",
				Integration: "google_calendar",
				Params:      map[string]interface{}{"days_ahead": 7},
			},
			{
				ID:     "create_log_entries",
				Type:   StepTypeAction,
				Action: "businessos.create_daily_log",
			},
		},
	})

	// Daily Brief skill - aggregates multiple sources with AI summarization
	e.RegisterSkill(&SkillDefinition{
		ID:                   "daily.brief",
		Name:                 "Generate Daily Brief",
		Description:          "Creates a daily brief from calendar, tasks, and emails",
		Category:             "automation",
		RequiredIntegrations: []string{}, // Works with whatever is connected
		Steps: []Step{
			{
				ID:      "gather_calendar",
				Type:    StepTypeAction,
				Action:  "google_calendar.list_events",
				Params:  map[string]interface{}{"days_ahead": 1},
				OnError: "continue", // Continue even if not connected
			},
			{
				ID:      "gather_tasks",
				Type:    StepTypeAction,
				Action:  "businessos.list_pending_tasks",
				OnError: "continue",
			},
			{
				ID:      "gather_emails",
				Type:    StepTypeAction,
				Action:  "gmail.list_messages",
				Params:  map[string]interface{}{"max_results": 20, "label": "INBOX"},
				OnError: "continue",
			},
			{
				ID:     "synthesize_brief",
				Type:   StepTypeAction,
				Action: "agent.orchestrator",
				Params: map[string]interface{}{
					"task": `Based on the gathered data, create a daily brief that includes:
1. **Today's Schedule** - Key meetings and events
2. **Priority Tasks** - Most important tasks to complete today
3. **Unread Emails** - Summary of important unread emails
4. **Key Reminders** - Any deadlines or important dates coming up

Format as a well-structured markdown document suitable for the user to review each morning.`,
				},
			},
			{
				ID:     "save_to_daily_log",
				Type:   StepTypeAction,
				Action: "businessos.create_daily_log",
				Params: map[string]interface{}{"from": "synthesize_brief", "type": "daily_brief"},
			},
		},
	})

	// Knowledge extraction skill - uses document agent
	e.RegisterSkill(&SkillDefinition{
		ID:                   "knowledge.extract_and_build",
		Name:                 "Extract and Build Knowledge",
		Description:          "Extracts key information from sources and creates knowledge nodes",
		Category:             "knowledge",
		RequiredIntegrations: []string{},
		Steps: []Step{
			{
				ID:     "analyze_source",
				Type:   StepTypeAction,
				Action: "agent.analyst",
				Params: map[string]interface{}{
					"task": `Analyze the provided content and extract:
1. Key concepts and definitions
2. Important facts and figures
3. Relationships between entities
4. Actionable insights

Format each item as a structured knowledge node with: title, type, content, tags, and related_to fields.`,
				},
			},
			{
				ID:     "create_nodes",
				Type:   StepTypeAction,
				Action: "businessos.create_nodes",
				Params: map[string]interface{}{"from": "analyze_source", "type": "knowledge"},
			},
		},
	})

	// Client analysis skill - uses analyst agent
	e.RegisterSkill(&SkillDefinition{
		ID:                   "analysis.client_health",
		Name:                 "Client Health Analysis",
		Description:          "Analyzes client data and generates health report",
		Category:             "analysis",
		RequiredIntegrations: []string{},
		Steps: []Step{
			{
				ID:     "gather_client_data",
				Type:   StepTypeAction,
				Action: "businessos.get_client_summary",
			},
			{
				ID:     "analyze_health",
				Type:   StepTypeAction,
				Action: "agent.analyst",
				Params: map[string]interface{}{
					"task": `Analyze this client data and provide a health assessment including:
1. **Health Score** (1-10) with reasoning
2. **Strengths** - What's going well
3. **Risks** - Potential issues to address
4. **Recommendations** - Suggested actions
5. **Next Steps** - Specific tasks to improve the relationship`,
					"from": "gather_client_data",
				},
			},
		},
	})

	// Pipeline analysis skill
	e.RegisterSkill(&SkillDefinition{
		ID:                   "analysis.pipeline",
		Name:                 "Sales Pipeline Analysis",
		Description:          "Analyzes sales pipeline and generates insights",
		Category:             "analysis",
		RequiredIntegrations: []string{},
		Steps: []Step{
			{
				ID:     "gather_pipeline",
				Type:   StepTypeAction,
				Action: "businessos.get_pipeline_summary",
			},
			{
				ID:     "analyze_pipeline",
				Type:   StepTypeAction,
				Action: "agent.analyst",
				Params: map[string]interface{}{
					"task": `Analyze this pipeline data and provide insights including:
1. **Pipeline Health** - Overall status and value
2. **Stage Distribution** - Where deals are concentrated
3. **Velocity Analysis** - How fast deals are moving
4. **At-Risk Deals** - Deals that need attention
5. **Forecast** - Projected outcomes based on current data
6. **Recommendations** - Strategic actions to improve pipeline`,
					"from": "gather_pipeline",
				},
			},
		},
	})

	// Meeting prep skill - uses orchestrator for comprehensive prep
	e.RegisterSkill(&SkillDefinition{
		ID:                   "meeting.prepare",
		Name:                 "Meeting Preparation",
		Description:          "Prepares comprehensive brief for an upcoming meeting",
		Category:             "productivity",
		RequiredIntegrations: []string{},
		Steps: []Step{
			{
				ID:      "get_meeting_details",
				Type:    StepTypeAction,
				Action:  "google_calendar.get_event",
				OnError: "continue",
			},
			{
				ID:     "gather_context",
				Type:   StepTypeAction,
				Action: "businessos.get_meeting_context",
				Params: map[string]interface{}{"from": "get_meeting_details"},
			},
			{
				ID:     "prepare_brief",
				Type:   StepTypeAction,
				Action: "agent.orchestrator",
				Params: map[string]interface{}{
					"task": `Prepare a comprehensive meeting brief including:
1. **Meeting Overview** - Purpose, attendees, timing
2. **Attendee Profiles** - Relevant background on each attendee
3. **Historical Context** - Previous interactions and discussions
4. **Talking Points** - Key topics to cover
5. **Questions to Ask** - Strategic questions for the meeting
6. **Potential Objections** - Issues that might come up and how to address
7. **Desired Outcomes** - What success looks like for this meeting`,
					"from": "gather_context",
				},
			},
		},
	})
}
