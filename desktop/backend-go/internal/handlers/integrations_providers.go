package handlers

// ============================================================================
// Provider Data — OAuth mapping and default provider catalogue
// ============================================================================

// getOAuthProvider maps provider IDs to their OAuth endpoint provider.
// e.g., gmail and google_calendar both use the "google" OAuth endpoint.
func getOAuthProvider(providerID string) string {
	oauthMapping := map[string]string{
		"gmail":           "google",
		"google_calendar": "google",
		"google_drive":    "google",
		"gemini":          "google",
		"outlook":         "microsoft",
		"teams":           "microsoft",
		// These use their own OAuth endpoints
		"slack":      "slack",
		"notion":     "notion",
		"hubspot":    "hubspot",
		"salesforce": "salesforce",
		"linear":     "linear",
		"asana":      "asana",
		"github":     "github",
		"gitlab":     "gitlab",
		"zoom":       "zoom",
		"discord":    "discord",
		"dropbox":    "dropbox",
		"clickup":    "clickup",
		"jira":       "jira",
		"trello":     "trello",
		"pipedrive":  "pipedrive",
		"fathom":     "fathom",
		"fireflies":  "fireflies",
		"instagram":  "instagram",
	}

	if oauth, ok := oauthMapping[providerID]; ok {
		return oauth
	}
	// Default: empty string (e.g., chatgpt, claude — file import only)
	return ""
}

// getDefaultProviders returns core providers when database is empty.
// Icon URLs use local /logos/integrations/ where available, authjs.dev as fallback.
// Includes est_nodes, initial_sync, auto_live_sync, and tooltip for rich UI display.
func getDefaultProviders() []map[string]interface{} {
	providers := []map[string]interface{}{
		// Productivity - Email & Calendar
		{"id": "gmail", "name": "Gmail", "description": "Import project details and track the context of important conversations.", "category": "communication", "icon_url": "/logos/integrations/gmail.svg", "modules": []string{"chat", "daily_log"}, "skills": []string{"gmail.send_email", "gmail.search"}, "status": "available", "auto_live_sync": true, "est_nodes": "50-200", "initial_sync": "15-30m", "tooltip": "Your new emails are processed into nodes every day."},
		{"id": "google_calendar", "name": "Google Calendar", "description": "Sync your events so BusinessOS stays on top of meetings, plans, and deadlines.", "category": "calendar", "icon_url": "/logos/integrations/calendar.svg", "modules": []string{"calendar", "daily_log"}, "skills": []string{"google_calendar.sync_daily_log", "google_calendar.create_event"}, "status": "available", "auto_live_sync": true, "est_nodes": "20-100", "initial_sync": "5-10m", "tooltip": "Your calendar events are automatically synced to keep your schedule updated."},
		{"id": "notion", "name": "Notion", "description": "Sync your workspace pages, project roadmaps, and structured knowledge.", "category": "storage", "icon_url": "/logos/integrations/notion.svg", "modules": []string{"contexts", "projects"}, "skills": []string{"notion.sync_database", "notion.create_page"}, "status": "available", "auto_live_sync": true, "est_nodes": "30-150", "initial_sync": "10-20m", "tooltip": "Your Notion updates are processed into nodes every day."},
		{"id": "google_drive", "name": "Google Drive", "description": "Sync your documents, spreadsheets, and presentations into your knowledge base.", "category": "storage", "icon_url": "https://authjs.dev/img/providers/google.svg", "modules": []string{"contexts"}, "skills": []string{}, "status": "available", "auto_live_sync": true, "est_nodes": "50-300", "initial_sync": "20-40m", "tooltip": "Your Drive files are indexed and searchable within your knowledge base."},
		{"id": "dropbox", "name": "Dropbox", "description": "Import your files and folders to make them searchable and connected.", "category": "storage", "icon_url": "https://authjs.dev/img/providers/dropbox.svg", "modules": []string{"contexts"}, "skills": []string{}, "status": "available", "auto_live_sync": true, "est_nodes": "30-200", "initial_sync": "15-30m", "tooltip": "Your Dropbox files are continuously synced."},
		// Communication
		{"id": "slack", "name": "Slack", "description": "Extract key insights and memories from your team channels and DMs.", "category": "communication", "icon_url": "/logos/integrations/slack.svg", "modules": []string{"chat", "tasks", "team"}, "skills": []string{"slack.send_message", "slack.message_to_task"}, "status": "available", "auto_live_sync": true, "est_nodes": "150-300", "initial_sync": "30-45m", "tooltip": "Your Slack messages are analyzed for important insights and decisions."},
		{"id": "instagram", "name": "Instagram", "description": "Connect posts, Reels, profile metrics, publishing status, and ContentOS performance audits.", "category": "social", "icon_url": "/logos/integrations/instagram.svg", "modules": []string{"content"}, "skills": []string{"instagram.audit_profile", "instagram.sync_reels", "instagram.publish_media"}, "status": "available", "auto_live_sync": true, "est_nodes": "100-500", "initial_sync": "10-30m", "tooltip": "Your Instagram account can feed ContentOS audits, post tracking, and performance insights."},
		{"id": "teams", "name": "Microsoft Teams", "description": "Sync your Teams conversations, channels, and shared files.", "category": "communication", "icon_url": "https://authjs.dev/img/providers/azure-ad.svg", "modules": []string{"chat", "team"}, "skills": []string{}, "status": "available", "auto_live_sync": true, "est_nodes": "100-250", "initial_sync": "25-40m", "tooltip": "Your Teams messages and files are synced automatically."},
		{"id": "discord", "name": "Discord", "description": "Import conversations from your Discord servers and DMs.", "category": "communication", "icon_url": "https://authjs.dev/img/providers/discord.svg", "modules": []string{"chat"}, "skills": []string{}, "status": "coming_soon", "auto_live_sync": true, "est_nodes": "100-300", "initial_sync": "20-35m"},
		// AI Assistants (Manual sync)
		{"id": "chatgpt", "name": "ChatGPT", "description": "Capture your brainstorming sessions, creative ideas, and problem-solving history.", "category": "ai", "icon_url": "/logos/integrations/openai.svg", "modules": []string{"contexts"}, "skills": []string{"chatgpt.import_history"}, "status": "available", "auto_live_sync": false, "est_nodes": "80-120", "initial_sync": "30m"},
		{"id": "claude", "name": "Claude", "description": "Preserve your Claude in-depth discussions, research analysis, and writing drafts.", "category": "ai", "icon_url": "/logos/integrations/claude.svg", "modules": []string{"contexts"}, "skills": []string{"claude.import_history"}, "status": "available", "auto_live_sync": false, "est_nodes": "80-120", "initial_sync": "10-15m"},
		{"id": "perplexity", "name": "Perplexity", "description": "Import your research queries, sources, and discovered insights.", "category": "ai", "icon_url": "https://authjs.dev/img/providers/perplexity.svg", "modules": []string{"contexts"}, "skills": []string{}, "status": "available", "auto_live_sync": false, "est_nodes": "40-80", "initial_sync": "10-15m"},
		{"id": "gemini", "name": "Google Gemini", "description": "Sync your Gemini conversations and generated content.", "category": "ai", "icon_url": "https://authjs.dev/img/providers/google.svg", "modules": []string{"contexts"}, "skills": []string{}, "status": "coming_soon", "auto_live_sync": false, "est_nodes": "60-100", "initial_sync": "15-20m"},
		// Meetings
		{"id": "fireflies", "name": "Fireflies.ai", "description": "Turn meeting transcripts, summaries, and action items into memories.", "category": "meetings", "icon_url": "/logos/integrations/fireflies.svg", "modules": []string{"daily_log", "contexts"}, "skills": []string{"fireflies.get_transcripts"}, "status": "available", "auto_live_sync": true, "est_nodes": "20-50", "initial_sync": "10-15m", "tooltip": "Your meeting transcripts are processed into memories automatically."},
		{"id": "fathom", "name": "Fathom", "description": "Turn meeting transcripts, summaries, and action items into memories.", "category": "meetings", "icon_url": "/logos/integrations/fathom.svg", "modules": []string{"daily_log"}, "skills": []string{"fathom.get_summaries"}, "status": "available", "auto_live_sync": true, "est_nodes": "20-50", "initial_sync": "10-15m", "tooltip": "Your meeting transcripts and summaries are processed automatically."},
		{"id": "tldv", "name": "tl;dv", "description": "Turn meeting transcripts, summaries, and action items into memories.", "category": "meetings", "icon_url": "https://authjs.dev/img/providers/google.svg", "modules": []string{"daily_log", "contexts"}, "skills": []string{}, "status": "available", "auto_live_sync": true, "est_nodes": "20-50", "initial_sync": "10-15m", "tooltip": "Your meeting recordings are transcribed and processed automatically."},
		{"id": "granola", "name": "Granola", "description": "Upload meeting notes to turn transcripts into memories.", "category": "meetings", "icon_url": "https://authjs.dev/img/providers/google.svg", "modules": []string{"daily_log"}, "skills": []string{}, "status": "available", "auto_live_sync": false, "est_nodes": "20-50", "initial_sync": "10-15m"},
		{"id": "zoom", "name": "Zoom", "description": "Import meeting recordings, transcripts, and chat history.", "category": "meetings", "icon_url": "https://authjs.dev/img/providers/zoom.svg", "modules": []string{"calendar", "daily_log"}, "skills": []string{}, "status": "available", "auto_live_sync": true, "est_nodes": "30-80", "initial_sync": "15-25m", "tooltip": "Your Zoom recordings are automatically transcribed and imported."},
		{"id": "loom", "name": "Loom", "description": "Import your video messages and their transcripts.", "category": "meetings", "icon_url": "https://authjs.dev/img/providers/loom.svg", "modules": []string{"daily_log", "contexts"}, "skills": []string{}, "status": "available", "auto_live_sync": true, "est_nodes": "15-40", "initial_sync": "10-15m", "tooltip": "Your Loom videos are transcribed and added automatically."},
		// Project Management
		{"id": "linear", "name": "Linear", "description": "Sync your issues, projects, and roadmaps for full context.", "category": "tasks", "icon_url": "https://authjs.dev/img/providers/linear.svg", "modules": []string{"tasks", "projects"}, "skills": []string{"linear.sync_issues"}, "status": "available", "auto_live_sync": true, "est_nodes": "50-150", "initial_sync": "10-20m", "tooltip": "Your Linear issues and updates are synced in real-time."},
		{"id": "asana", "name": "Asana", "description": "Import your tasks, projects, and team workflows.", "category": "tasks", "icon_url": "https://authjs.dev/img/providers/asana.svg", "modules": []string{"tasks", "projects"}, "skills": []string{"asana.sync_tasks"}, "status": "available", "auto_live_sync": true, "est_nodes": "40-120", "initial_sync": "15-25m", "tooltip": "Your Asana tasks and projects are synced automatically."},
		{"id": "monday", "name": "Monday.com", "description": "Sync your boards, items, and updates into your knowledge base.", "category": "tasks", "icon_url": "https://authjs.dev/img/providers/monday.svg", "modules": []string{"tasks", "projects"}, "skills": []string{}, "status": "available", "auto_live_sync": true, "est_nodes": "40-100", "initial_sync": "15-20m", "tooltip": "Your Monday boards are synced and updated automatically."},
		{"id": "trello", "name": "Trello", "description": "Import your boards, cards, and checklists.", "category": "tasks", "icon_url": "https://authjs.dev/img/providers/trello.svg", "modules": []string{"tasks"}, "skills": []string{}, "status": "available", "auto_live_sync": true, "est_nodes": "30-80", "initial_sync": "10-15m", "tooltip": "Your Trello boards are synced in real-time."},
		{"id": "jira", "name": "Jira", "description": "Sync your issues, sprints, and project documentation.", "category": "tasks", "icon_url": "https://authjs.dev/img/providers/atlassian.svg", "modules": []string{"tasks", "projects"}, "skills": []string{}, "status": "available", "auto_live_sync": true, "est_nodes": "60-200", "initial_sync": "20-35m", "tooltip": "Your Jira issues and sprints are synced automatically."},
		{"id": "clickup", "name": "ClickUp", "description": "Import your tasks, docs, and workspace data.", "category": "tasks", "icon_url": "https://authjs.dev/img/providers/click-up.svg", "modules": []string{"tasks", "projects"}, "skills": []string{}, "status": "available", "auto_live_sync": true, "est_nodes": "50-150", "initial_sync": "15-25m", "tooltip": "Your ClickUp workspace is synced automatically."},
		// CRM
		{"id": "hubspot", "name": "HubSpot", "description": "Sync your CRM contacts, deals, and customer interactions into your knowledge base.", "category": "crm", "icon_url": "/logos/integrations/hubspot.svg", "modules": []string{"clients", "projects"}, "skills": []string{"hubspot.qualify_lead", "hubspot.sync_contacts"}, "status": "available", "auto_live_sync": true, "est_nodes": "100-500", "initial_sync": "20-40m", "tooltip": "Your HubSpot contacts and deals are synced and analyzed for insights."},
		{"id": "gohighlevel", "name": "GoHighLevel", "description": "Import your marketing funnels, contacts, and automation data.", "category": "crm", "icon_url": "https://authjs.dev/img/providers/google.svg", "modules": []string{"clients", "projects"}, "skills": []string{}, "status": "available", "auto_live_sync": true, "est_nodes": "150-400", "initial_sync": "25-45m", "tooltip": "Your GHL contacts, funnels, and campaigns are synced automatically."},
		{"id": "salesforce", "name": "Salesforce", "description": "Sync your accounts, opportunities, and customer data.", "category": "crm", "icon_url": "https://authjs.dev/img/providers/salesforce.svg", "modules": []string{"clients"}, "skills": []string{}, "status": "available", "auto_live_sync": true, "est_nodes": "200-600", "initial_sync": "30-60m", "tooltip": "Your Salesforce data is synced and enriched automatically."},
		{"id": "pipedrive", "name": "Pipedrive", "description": "Import your deals, contacts, and sales pipeline.", "category": "crm", "icon_url": "https://authjs.dev/img/providers/pipedrive.svg", "modules": []string{"clients"}, "skills": []string{}, "status": "available", "auto_live_sync": true, "est_nodes": "80-250", "initial_sync": "15-30m", "tooltip": "Your Pipedrive pipeline is synced in real-time."},
		// Storage
		{"id": "airtable", "name": "Airtable", "description": "Sync your bases, tables, and automation data.", "category": "storage", "icon_url": "/logos/integrations/airtable.webp", "modules": []string{"contexts", "projects"}, "skills": []string{"airtable.sync_base"}, "status": "available", "auto_live_sync": true, "est_nodes": "50-200", "initial_sync": "15-30m", "tooltip": "Your Airtable bases are continuously synced."},
		// Notes (Manual sync)
		{"id": "evernote", "name": "Evernote", "description": "Import your notes, notebooks, and web clips.", "category": "storage", "icon_url": "https://authjs.dev/img/providers/evernote.svg", "modules": []string{"contexts"}, "skills": []string{}, "status": "available", "auto_live_sync": false, "est_nodes": "100-300", "initial_sync": "15-30m"},
		{"id": "obsidian", "name": "Obsidian", "description": "Sync your vault, notes, and knowledge graph connections.", "category": "storage", "icon_url": "https://authjs.dev/img/providers/google.svg", "modules": []string{"contexts"}, "skills": []string{}, "status": "available", "auto_live_sync": false, "est_nodes": "50-200", "initial_sync": "10-20m"},
		{"id": "roam", "name": "Roam Research", "description": "Import your daily notes, linked references, and graph structure.", "category": "storage", "icon_url": "https://authjs.dev/img/providers/google.svg", "modules": []string{"contexts"}, "skills": []string{}, "status": "available", "auto_live_sync": false, "est_nodes": "60-180", "initial_sync": "15-25m"},
		// Calendar
		{"id": "outlook", "name": "Microsoft Outlook", "description": "Sync your Outlook calendar, events, and email.", "category": "calendar", "icon_url": "https://authjs.dev/img/providers/azure-ad.svg", "modules": []string{"calendar", "daily_log"}, "skills": []string{}, "status": "available", "auto_live_sync": true, "est_nodes": "30-150", "initial_sync": "10-20m", "tooltip": "Your Outlook events are automatically synced."},
		{"id": "calendly", "name": "Calendly", "description": "Sync your scheduled meetings and availability.", "category": "calendar", "icon_url": "https://authjs.dev/img/providers/calendly.svg", "modules": []string{"calendar"}, "skills": []string{}, "status": "available", "auto_live_sync": true, "est_nodes": "10-50", "initial_sync": "5-10m", "tooltip": "Your Calendly bookings are synced automatically."},
	}

	// Attach oauth_provider to each entry
	for _, p := range providers {
		if id, ok := p["id"].(string); ok {
			p["oauth_provider"] = getOAuthProvider(id)
		}
	}

	return providers
}
