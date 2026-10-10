<script lang="ts">
	import {
		Grid3X3,
		PieChart,
		MessageSquare,
		CheckSquare,
		FolderKanban,
		Users,
		Calendar,
		Terminal,
		Activity,
		TrendingUp,
		Settings
	} from 'lucide-svelte';
</script>

<!-- Core Modules -->
<section id="modules" class="help-section">
	<div class="section-header">
		<div class="section-icon gradient-orange">
			<Grid3X3 size={24} />
		</div>
		<div>
			<h1>Core Modules</h1>
			<p class="section-subtitle">Everything you need to run your business</p>
		</div>
	</div>

	<div class="modules-showcase">
		<div id="dashboard-mod" class="module-detail">
			<div class="module-header">
				<div class="module-icon" style="background: var(--bos-status-info-bg); color: var(--bos-status-info);">
					<PieChart size={24} />
				</div>
				<div>
					<h3>Dashboard</h3>
					<span class="module-tagline">Your daily command center</span>
				</div>
			</div>
			<p>Get a complete overview of your business at a glance. See today's tasks, upcoming
			deadlines, project progress, and key metrics. Quick actions let you create tasks,
			start a chat, or jump to any module.</p>
			<div class="module-features">
				<span>Today's Focus</span>
				<span>Task Summary</span>
				<span>Project Status</span>
				<span>Quick Actions</span>
			</div>
		</div>

		<div id="chat-mod" class="module-detail">
			<div class="module-header">
				<div class="module-icon" style="background: var(--bos-status-success-bg); color: var(--bos-status-success);">
					<MessageSquare size={24} />
				</div>
				<div>
					<h3>Chat</h3>
					<span class="module-tagline">AI-powered conversations</span>
				</div>
			</div>
			<p>Your intelligent assistant for any task. Use Focus Modes for specialized help,
			save conversation threads, and let AI access your business context for relevant answers.</p>
			<div class="module-features">
				<span>Focus Modes</span>
				<span>Voice Input</span>
				<span>Artifacts</span>
				<span>Context Awareness</span>
			</div>
		</div>

		<div id="tasks-mod" class="module-detail">
			<div class="module-header">
				<div class="module-icon" style="background: var(--bos-status-warning-bg); color: var(--bos-status-warning);">
					<CheckSquare size={24} />
				</div>
				<div>
					<h3>Tasks</h3>
					<span class="module-tagline">Get things done</span>
				</div>
			</div>
			<p>Powerful task management with multiple views. Kanban boards for visual workflow,
			list view for quick scanning, calendar view for deadline management. Set priorities,
			assignees, and due dates.</p>
			<div class="module-features">
				<span>Kanban Board</span>
				<span>List View</span>
				<span>Calendar View</span>
				<span>Priorities</span>
				<span>Filters</span>
			</div>
		</div>

		<div id="projects-mod" class="module-detail">
			<div class="module-header">
				<div class="module-icon" style="background: color-mix(in srgb, var(--bos-category-ai) 12%, transparent); color: var(--bos-category-ai);">
					<FolderKanban size={24} />
				</div>
				<div>
					<h3>Projects</h3>
					<span class="module-tagline">Organize your work</span>
				</div>
			</div>
			<p>Group related tasks and track progress. Each project has a status, description,
			and linked tasks. See completion percentages and manage project-specific notes.</p>
			<div class="module-features">
				<span>Project Status</span>
				<span>Task Linking</span>
				<span>Progress Tracking</span>
				<span>Notes</span>
			</div>
		</div>

		<div id="team-mod" class="module-detail">
			<div class="module-header">
				<div class="module-icon" style="background: color-mix(in srgb, var(--bos-category-automation) 12%, transparent); color: var(--bos-category-automation);">
					<Users size={24} />
				</div>
				<div>
					<h3>Team</h3>
					<span class="module-tagline">Manage your people</span>
				</div>
			</div>
			<p>Visualize your organization with an interactive org chart. Track team member
			capacity, assign tasks, and see workload distribution across your team.</p>
			<div class="module-features">
				<span>Org Chart</span>
				<span>Capacity Planning</span>
				<span>Workload View</span>
				<span>Team Directory</span>
			</div>
		</div>

		<div id="calendar-mod" class="module-detail">
			<div class="module-header">
				<div class="module-icon" style="background: color-mix(in srgb, var(--bos-category-communication) 12%, transparent); color: var(--bos-category-communication);">
					<Calendar size={24} />
				</div>
				<div>
					<h3>Calendar</h3>
					<span class="module-tagline">Time management</span>
				</div>
			</div>
			<p>Schedule events, meetings, and deadlines. Integrates with Google Calendar for
			sync. Multiple views including day, week, and month layouts.</p>
			<div class="module-features">
				<span>Event Creation</span>
				<span>Google Sync</span>
				<span>Multiple Views</span>
				<span>Reminders</span>
			</div>
		</div>
	</div>

	<div class="more-modules">
		<h3>Additional Modules</h3>
		<div class="mini-modules-grid">
			<div class="mini-module">
				<div class="mini-icon" style="background: var(--dbg3); color: var(--bos-status-success);">
					<Terminal size={18} />
				</div>
				<div>
					<h5>Terminal</h5>
					<p>Command-line interface for power users and OS Agent interaction</p>
				</div>
			</div>
			<div class="mini-module">
				<div class="mini-icon" style="background: var(--dbg2); color: var(--dt2);">
					<Activity size={18} />
				</div>
				<div>
					<h5>Daily Log</h5>
					<p>Track daily activities, patterns, and reflections over time</p>
				</div>
			</div>
			<div class="mini-module">
				<div class="mini-icon" style="background: var(--bos-status-success-bg); color: var(--bos-status-success);">
					<TrendingUp size={18} />
				</div>
				<div>
					<h5>Usage</h5>
					<p>Analytics on AI usage, tokens consumed, and costs by provider</p>
				</div>
			</div>
			<div class="mini-module">
				<div class="mini-icon" style="background: var(--dbg2); color: var(--dt2);">
					<Settings size={18} />
				</div>
				<div>
					<h5>Settings</h5>
					<p>Configure AI models, API keys, integrations, and preferences</p>
				</div>
			</div>
		</div>
	</div>
</section>
