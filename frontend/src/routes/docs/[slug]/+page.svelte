<script lang="ts">
	import { page } from '$app/stores';
	import { fly, fade } from 'svelte/transition';
	import { onMount } from 'svelte';
	import { useSession } from '$lib/auth-client';

	const session = useSession();
	let showContent = $state(false);

	onMount(() => {
		setTimeout(() => (showContent = true), 100);
	});

	const features: Record<string, {
		name: string;
		tagline: string;
		description: string;
		icon: string;
		features: { title: string; desc: string }[];
		useCases: string[];
		techDetails: string[];
	}> = {
		'desktop': {
			name: 'Desktop',
			tagline: 'Native experience for your operating system',
			description: 'Business OS Desktop brings the full power of your workspace to a native application. Built with Electron, it runs on macOS, Windows, and Linux with system-level integration, global keyboard shortcuts, voice commands, and a seamless experience that feels like a true operating system.',
			icon: 'M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z',
			features: [
				{ title: 'voice_commands', desc: 'Speak to your AI assistant hands-free. Dictate messages, create tasks, and control your workspace with natural voice input.' },
				{ title: 'global_shortcuts', desc: 'Trigger actions from anywhere on your system. Open Spotlight with Cmd+Space, quick capture with customizable hotkeys.' },
				{ title: 'cross_platform', desc: 'Runs natively on macOS, Windows, and Linux. Same powerful experience across all your machines.' },
				{ title: 'system_tray', desc: 'Quick access from your system tray with status indicators, quick actions, and background operation.' },
				{ title: 'offline_support', desc: 'Continue working without internet - local AI models work offline, data syncs when back online.' },
				{ title: 'native_notifications', desc: 'System-level notifications that respect your OS settings and notification preferences.' }
			],
			useCases: [
				'Power users who want keyboard and voice-driven workflows',
				'Teams that need offline access to critical data',
				'Developers who prefer native app performance',
				'Users who work across macOS, Windows, and Linux'
			],
			techDetails: [
				'Built with Electron for macOS, Windows, and Linux',
				'Global shortcuts registered at OS level',
				'Web Speech API for voice recognition',
				'Local SQLite for offline data storage',
				'IPC for secure main/renderer communication',
				'Tauri support planned for smaller bundle size'
			]
		},
		'dashboard': {
			name: 'Dashboard',
			tagline: 'Your command center at a glance',
			description: 'The Dashboard is your home base in Business OS. It provides a customizable overview of your work, with widgets for tasks, calendar, metrics, and quick actions. Everything you need to start your day and stay on track.',
			icon: 'M4 5a1 1 0 011-1h14a1 1 0 011 1v2a1 1 0 01-1 1H5a1 1 0 01-1-1V5zM4 13a1 1 0 011-1h6a1 1 0 011 1v6a1 1 0 01-1 1H5a1 1 0 01-1-1v-6zM16 13a1 1 0 011-1h2a1 1 0 011 1v6a1 1 0 01-1 1h-2a1 1 0 01-1-1v-6z',
			features: [
				{ title: 'customizable_widgets', desc: 'Drag and drop widgets to create your perfect layout.' },
				{ title: 'realtime_metrics', desc: 'Live data from your projects, tasks, and integrations.' },
				{ title: 'quick_actions', desc: 'One-click access to common tasks and workflows.' },
				{ title: 'focus_mode', desc: 'Minimize distractions and focus on what matters.' },
				{ title: 'team_activity', desc: 'See what your team is working on in real-time.' },
				{ title: 'smart_suggestions', desc: 'AI-powered recommendations for your next actions.' }
			],
			useCases: [
				'Morning stand-ups and daily planning',
				'Monitoring team progress and blockers',
				'Quick access to frequently used features',
				'Executive overview of business metrics'
			],
			techDetails: [
				'Responsive grid layout with CSS Grid',
				'WebSocket connections for real-time updates',
				'Local storage for widget preferences',
				'Server-side rendering for fast initial load'
			]
		},
		'chat': {
			name: 'Chat',
			tagline: 'AI-powered conversations with context',
			description: 'Chat is your AI assistant that understands your business. It has access to your contexts, can execute tasks, and integrates with your tools via aMCP. Have natural conversations that get work done.',
			icon: 'M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z',
			features: [
				{ title: 'context_awareness', desc: 'AI understands your projects, clients, and documents.' },
				{ title: 'tool_integration', desc: 'Execute actions across your connected tools via aMCP.' },
				{ title: 'artifact_generation', desc: 'Create documents, code, and content inline.' },
				{ title: 'conversation_history', desc: 'Searchable history with context preservation.' },
				{ title: 'multiple_models', desc: 'Switch between Ollama, OpenAI, Anthropic, and more.' },
				{ title: 'streaming_responses', desc: 'Real-time streaming for fast, interactive conversations.' }
			],
			useCases: [
				'Research and analysis with AI assistance',
				'Drafting documents and communications',
				'Querying data across your integrations',
				'Automating repetitive tasks through conversation'
			],
			techDetails: [
				'Server-sent events for streaming',
				'Vector embeddings for context retrieval',
				'aMCP protocol for tool execution',
				'Multi-modal support (text, images, files)'
			]
		},
		'tasks': {
			name: 'Tasks',
			tagline: 'Track work with AI-powered management',
			description: 'Tasks helps you manage your work with intelligent prioritization, due date tracking, and AI assistance. Create tasks from conversations, link them to projects, and let the system help you stay on top of everything.',
			icon: 'M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4',
			features: [
				{ title: 'smart_prioritization', desc: 'AI suggests task priority based on context and deadlines.' },
				{ title: 'project_linking', desc: 'Connect tasks to projects for organized tracking.' },
				{ title: 'due_date_reminders', desc: 'Never miss a deadline with smart notifications.' },
				{ title: 'subtasks', desc: 'Break down complex work into manageable pieces.' },
				{ title: 'time_tracking', desc: 'Track time spent on tasks for billing and analysis.' },
				{ title: 'recurring_tasks', desc: 'Automate repeating work with flexible schedules.' }
			],
			useCases: [
				'Personal task management and GTD workflows',
				'Team task assignment and tracking',
				'Sprint planning and agile workflows',
				'Client deliverable tracking'
			],
			techDetails: [
				'Optimistic updates for instant feedback',
				'Drag-and-drop with touch support',
				'Keyboard navigation for power users',
				'Bulk operations for efficiency'
			]
		},
		'projects': {
			name: 'Projects',
			tagline: 'Organize work at scale',
			description: 'Projects brings structure to your work with milestones, team assignments, and progress tracking. Link projects to contexts for AI understanding, and manage complex initiatives with ease.',
			icon: 'M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10',
			features: [
				{ title: 'milestone_tracking', desc: 'Set and track major project milestones.' },
				{ title: 'team_assignments', desc: 'Assign team members with roles and permissions.' },
				{ title: 'progress_visualization', desc: 'See project health at a glance with charts.' },
				{ title: 'context_linking', desc: 'Connect projects to knowledge bases for AI context.' },
				{ title: 'client_association', desc: 'Link projects to clients for billing and reporting.' },
				{ title: 'template_system', desc: 'Create project templates for repeating work.' }
			],
			useCases: [
				'Managing client engagements',
				'Product development tracking',
				'Cross-functional initiatives',
				'Consulting projects with deliverables'
			],
			techDetails: [
				'Hierarchical data structure',
				'Real-time collaboration',
				'Audit logging for compliance',
				'Flexible permission system'
			]
		},
		'calendar': {
			name: 'Calendar',
			tagline: 'Schedule with intelligence',
			description: 'Calendar integrates with your workflow to schedule events, meetings, and deadlines. Connect external calendars, get smart scheduling suggestions, and never double-book again.',
			icon: 'M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z',
			features: [
				{ title: 'multi_calendar_sync', desc: 'Sync with Google, Outlook, and other calendars.' },
				{ title: 'smart_scheduling', desc: 'AI suggests optimal meeting times.' },
				{ title: 'task_integration', desc: 'See task due dates alongside events.' },
				{ title: 'availability_sharing', desc: 'Share your availability with external contacts.' },
				{ title: 'recurring_events', desc: 'Complex recurrence patterns supported.' },
				{ title: 'timezone_support', desc: 'Handle global teams with ease.' }
			],
			useCases: [
				'Team scheduling and coordination',
				'Client meeting management',
				'Deadline tracking and planning',
				'Resource allocation'
			],
			techDetails: [
				'iCal format support',
				'CalDAV integration',
				'Real-time sync with webhooks',
				'Conflict detection algorithm'
			]
		},
		'clients': {
			name: 'Clients',
			tagline: 'Manage relationships that matter',
			description: 'Clients is your CRM within Business OS. Track contacts, communication history, and link clients to projects and contexts. Build stronger relationships with full visibility.',
			icon: 'M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z',
			features: [
				{ title: 'contact_management', desc: 'Store and organize all client contacts.' },
				{ title: 'communication_history', desc: 'Track all interactions in one place.' },
				{ title: 'project_linking', desc: 'See all projects associated with a client.' },
				{ title: 'context_creation', desc: 'Auto-generate contexts from client data.' },
				{ title: 'import_tools', desc: 'Import from Salesforce, HubSpot, and more.' },
				{ title: 'custom_fields', desc: 'Add fields specific to your business.' }
			],
			useCases: [
				'Agency client management',
				'Sales pipeline tracking',
				'Account management',
				'Partner relationship management'
			],
			techDetails: [
				'aMCP integrations for CRM sync',
				'Full-text search on all fields',
				'Activity timeline with filters',
				'Export to CSV/JSON'
			]
		},
		'contexts': {
			name: 'Contexts',
			tagline: 'Knowledge that powers AI',
			description: 'Contexts are the knowledge bases that inform your AI agents. Upload documents, add notes, and connect data sources. When you chat or use agents, they understand your business through contexts.',
			icon: 'M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10',
			features: [
				{ title: 'document_upload', desc: 'PDF, Word, Markdown, and more.' },
				{ title: 'web_scraping', desc: 'Import content from URLs automatically.' },
				{ title: 'data_connectors', desc: 'Sync from databases and APIs via aMCP.' },
				{ title: 'vector_embeddings', desc: 'Semantic search across all content.' },
				{ title: 'version_history', desc: 'Track changes to context documents.' },
				{ title: 'access_control', desc: 'Control who can view and edit contexts.' }
			],
			useCases: [
				'Client-specific knowledge bases',
				'Product documentation',
				'Research repositories',
				'Training data for AI agents'
			],
			techDetails: [
				'Chunking with overlap for retrieval',
				'Multiple embedding models supported',
				'Background processing queue',
				'S3-compatible storage'
			]
		},
		'nodes': {
			name: 'Nodes',
			tagline: 'Visualize connections',
			description: 'Nodes provides a graph view of entities in your workspace. See how clients, projects, contexts, and team members connect. Discover relationships and navigate your data visually.',
			icon: 'M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1',
			features: [
				{ title: 'graph_visualization', desc: 'Interactive node-link diagram.' },
				{ title: 'relationship_types', desc: 'Define custom relationship types.' },
				{ title: 'filtering', desc: 'Filter by entity type, date, and more.' },
				{ title: 'clustering', desc: 'Auto-group related entities.' },
				{ title: 'path_finding', desc: 'Find connections between any two nodes.' },
				{ title: 'export', desc: 'Export graph data for analysis.' }
			],
			useCases: [
				'Understanding client ecosystems',
				'Mapping project dependencies',
				'Knowledge graph exploration',
				'Team structure visualization'
			],
			techDetails: [
				'Force-directed layout algorithm',
				'WebGL rendering for performance',
				'Incremental loading for large graphs',
				'Custom node/edge styling'
			]
		},
		'daily-log': {
			name: 'Daily Log',
			tagline: 'Journal your journey',
			description: 'Daily Log is your work journal. Capture thoughts, track accomplishments, and reflect on your day. Entries are timestamped and searchable, creating a record of your work over time.',
			icon: 'M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z',
			features: [
				{ title: 'quick_entry', desc: 'Fast capture with keyboard shortcuts.' },
				{ title: 'timestamps', desc: 'Automatic timestamping of all entries.' },
				{ title: 'tags', desc: 'Organize entries with custom tags.' },
				{ title: 'search', desc: 'Full-text search across all logs.' },
				{ title: 'markdown_support', desc: 'Rich formatting with Markdown.' },
				{ title: 'daily_summary', desc: 'AI-generated summary of your day.' }
			],
			useCases: [
				'Work journaling and reflection',
				'Meeting notes and action items',
				'Learning and idea capture',
				'Time tracking and accountability'
			],
			techDetails: [
				'Real-time auto-save',
				'Conflict-free replicated data types',
				'Offline-first architecture',
				'Export to Markdown/PDF'
			]
		}
	};

	const slug = $derived($page.params.slug);
	const feature = $derived(slug ? features[slug] || null : null);
</script>

<svelte:head>
	<title>{feature?.name || 'Feature'} - Business OS Documentation</title>
</svelte:head>

<div class="min-h-screen bg-white">
	<!-- Grid background -->
	{#if showContent}
		<div class="fixed inset-0 pointer-events-none" in:fade={{ duration: 1000 }}>
			<div
				class="w-full h-full opacity-[0.04]"
				style="
					background-image:
						linear-gradient(rgba(0,0,0,0.3) 1px, transparent 1px),
						linear-gradient(90deg, rgba(0,0,0,0.3) 1px, transparent 1px);
					background-size: 60px 60px;
				"
			></div>
		</div>
	{/if}

	<!-- Header -->
	<header class="border-b border-gray-200 bg-white/95 backdrop-blur-sm sticky top-0 z-50">
		<div class="max-w-4xl mx-auto px-6 h-14 flex items-center justify-between">
			<div class="flex items-center gap-3">
				<a href="/" class="flex items-baseline gap-0.5">
					<span class="text-black text-lg font-extrabold tracking-[0.15em] font-mono">BUSINESS</span>
					<span class="text-black/30 text-base font-light font-mono">OS</span>
				</a>
				<span class="text-gray-300 font-mono">/</span>
				<a href="/docs" class="font-mono text-xs text-gray-500 hover:text-black transition-colors">docs</a>
				{#if feature}
					<span class="text-gray-300 font-mono">/</span>
					<span class="font-mono text-xs text-gray-900">{feature.name.toLowerCase()}</span>
				{/if}
			</div>
			{#if $session.data}
				<a href="/window" class="bg-black text-white px-4 py-2 rounded-lg text-xs font-medium hover:bg-gray-800 transition-colors font-mono flex items-center gap-2">
					<svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 19l-7-7m0 0l7-7m-7 7h18" />
					</svg>
					Back to Desktop
				</a>
			{:else}
				<a href="/register" class="bg-black text-white px-4 py-2 rounded-lg text-xs font-medium hover:bg-gray-800 transition-colors font-mono">
					Get Started
				</a>
			{/if}
		</div>
	</header>

	{#if showContent && feature}
		<div class="max-w-4xl mx-auto px-6 py-12">
			<!-- Hero -->
			<div class="mb-12" in:fly={{ y: 20, duration: 500 }}>
				<a href="/docs" class="inline-flex items-center gap-2 text-xs text-gray-400 hover:text-gray-900 transition-colors font-mono mb-6 group">
					<span class="group-hover:-translate-x-1 transition-transform">←</span>
					back to docs
				</a>

				<div class="flex items-start gap-4 mb-6">
					<div class="w-12 h-12 border border-gray-200 flex items-center justify-center flex-shrink-0">
						<svg class="w-6 h-6 text-black" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d={feature.icon} />
						</svg>
					</div>
					<div>
						<div class="font-mono text-xs text-gray-400 mb-1">> {feature.name.toLowerCase()}</div>
						<h1 class="text-2xl font-bold text-black font-mono">{feature.name}</h1>
						<p class="text-sm text-gray-500 font-mono mt-1">{feature.tagline}</p>
					</div>
				</div>

				<p class="text-gray-600 leading-relaxed font-mono text-sm">
					{feature.description}
				</p>
			</div>

			<!-- Features -->
			<section class="mb-12" in:fly={{ y: 20, duration: 500, delay: 100 }}>
				<div class="font-mono text-xs text-gray-400 mb-4 tracking-wider">> FEATURES</div>
				<div class="grid md:grid-cols-2 gap-3">
					{#each feature.features as feat, i}
						<div class="border border-gray-200 p-4 hover:border-gray-400 transition-colors">
							<h3 class="font-mono text-sm text-black mb-1">{feat.title}</h3>
							<p class="text-xs text-gray-500 font-mono leading-relaxed">{feat.desc}</p>
						</div>
					{/each}
				</div>
			</section>

			<!-- Use Cases -->
			<section class="mb-12" in:fly={{ y: 20, duration: 500, delay: 150 }}>
				<div class="font-mono text-xs text-gray-400 mb-4 tracking-wider">> USE_CASES</div>
				<div class="border border-gray-200 p-4">
					<ul class="space-y-2">
						{#each feature.useCases as useCase}
							<li class="flex items-center gap-3 font-mono text-sm">
								<span class="text-gray-400">-</span>
								<span class="text-gray-600">{useCase}</span>
							</li>
						{/each}
					</ul>
				</div>
			</section>

			<!-- Technical Details -->
			<section class="mb-12" in:fly={{ y: 20, duration: 500, delay: 200 }}>
				<div class="font-mono text-xs text-gray-400 mb-4 tracking-wider">> TECH_DETAILS</div>
				<div class="bg-gray-50 border border-gray-200 p-4">
					<ul class="space-y-2">
						{#each feature.techDetails as detail}
							<li class="flex items-center gap-2 font-mono text-xs text-gray-600">
								<span class="text-gray-400">$</span>
								{detail}
							</li>
						{/each}
					</ul>
				</div>
			</section>

			<!-- CTA -->
			<section class="border-t border-gray-200 pt-12" in:fly={{ y: 20, duration: 500, delay: 250 }}>
				<div class="text-center">
					<div class="font-mono text-xs text-gray-400 mb-2">> ready?</div>
					<h3 class="text-lg font-bold text-black mb-2 font-mono">Try {feature.name}</h3>
					<p class="text-gray-500 mb-6 font-mono text-sm">Get started with Business OS today.</p>
					<a href="/register" class="inline-flex items-center gap-2 bg-black text-white px-6 py-3 text-xs font-medium hover:bg-gray-800 transition-colors font-mono">
						initialize_workspace
						<span>→</span>
					</a>
				</div>
			</section>
		</div>
	{:else if showContent}
		<div class="max-w-4xl mx-auto px-6 py-12 text-center">
			<div class="font-mono text-xs text-gray-400 mb-4">> error: 404</div>
			<h1 class="text-xl font-bold text-black mb-4 font-mono">feature_not_found</h1>
			<p class="text-gray-500 mb-6 font-mono text-sm">The documentation page you're looking for doesn't exist.</p>
			<a href="/docs" class="text-black hover:underline font-mono text-sm">← back to docs</a>
		</div>
	{/if}
</div>
