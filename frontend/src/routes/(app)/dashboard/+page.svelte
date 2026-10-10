<script lang="ts">
	import { onMount } from 'svelte';
	import { currentWorkspace } from '$lib/stores/workspaces';
	import { useSession } from '$lib/auth-client';
	import { getDashboardSummary, type DashboardSummary } from '$lib/api/dashboard';
	import { founderosCommandGroup } from '$lib/founderos/commandModules';
	import { CheckSquare, FolderKanban, Users, CircleCheck, Loader2 } from 'lucide-svelte';

	const session = useSession();

	// At-a-glance: live operating numbers pulled from the real dashboard summary.
	let summary = $state<DashboardSummary | null>(null);
	let glanceLoading = $state(true);
	let glanceError = $state<string | null>(null);

	let wsId = $state<string | null | undefined>(undefined);
	$effect(() => {
		const id = $currentWorkspace?.id ?? null;
		if (id !== wsId) {
			wsId = id;
			loadGlance();
		}
	});

	onMount(loadGlance);

	async function loadGlance(): Promise<void> {
		glanceLoading = true;
		glanceError = null;
		try {
			summary = await getDashboardSummary();
		} catch (e) {
			summary = null;
			glanceError = e instanceof Error ? e.message : 'Could not load your at-a-glance.';
		} finally {
			glanceLoading = false;
		}
	}

	const focusToday = $derived(summary?.focus_items ?? []);
	const focusDone = $derived(focusToday.filter((f) => f.completed).length);

	type Stat = { label: string; value: string; icon: typeof CheckSquare };
	const stats = $derived<Stat[]>(
		summary
			? [
					{
						label: focusToday.length ? 'Focus done today' : 'Focus today',
						value: focusToday.length ? `${focusDone}/${focusToday.length}` : '0',
						icon: CircleCheck
					},
					{ label: 'Open tasks', value: String(summary.task_count ?? 0), icon: CheckSquare },
					{ label: 'Projects', value: String(summary.project_count ?? 0), icon: FolderKanban },
					{ label: 'Clients', value: String(summary.client_count ?? 0), icon: Users }
				]
			: []
	);

	const moduleGroups = $derived([
		{
			label: 'Operate',
			purpose: 'The daily control layer for attention, memory, agents, communication, and calendar rhythm.',
			modules: [
				{ href: '/dashboard', label: 'Command', desc: 'Daily command center for focus, priorities, decisions, blockers, and the current operating picture.', live: true },
				{ href: '/agents', label: 'Agents', desc: 'AI workers, skills, runs, delegation, approvals, and execution history across the workspace.', live: true },
				{ href: '/knowledge', label: 'Knowledge', desc: 'Workspace wiki, operating docs, source-linked memory, notes, files, and reusable context.', live: true },
				{ href: '/intelligence', label: 'Intelligence', desc: 'Signals, risks, recommendations, weak spots, opportunities, and what needs attention next.', live: true },
				{ href: '/inbox', label: 'Inbox', desc: 'Incoming work, messages, documents, reminders, requests, and items that need triage.', live: true },
				{ href: '/calendar', label: 'Calendar', desc: 'Meetings, time blocks, deadlines, team cadence, and the schedule that drives execution.', live: true }
			]
		},
		founderosCommandGroup($currentWorkspace?.settings ?? {}),
		{
			label: 'Business',
			purpose: 'The operating structure for who exists, what work exists, what is owed, and how the company moves.',
			modules: [
				{ href: '/relationships', label: 'Relationships', desc: 'People, clients, partners, vendors, team members, stakeholders, and relationship history.', live: true },
				{ href: '/projects', label: 'Projects', desc: 'Bounded initiatives, milestones, scopes, owners, timelines, and delivery state.', live: true },
				{ href: '/tasks', label: 'Tasks', desc: 'Assignments, next actions, delegated work, follow-ups, due dates, and completion state.', live: true },
				{ href: '/rhythm', label: 'Rhythm', desc: 'Daily, weekly, and monthly operating cadence, reviews, planning cycles, and accountability loops.', live: true },
				{ href: '/pipelines', label: 'Pipelines', desc: 'Leads, opportunities, deals, retainers, client stages, and revenue movement.', live: true },
				{ href: '/offers', label: 'Offers', desc: 'What the business sells, packaging, pricing logic, positioning, promises, and sales assets.', live: true }
			]
		},
		{
			label: 'Growth',
			purpose: 'The market-facing layer for demand, sites, audiences, messaging, and published content.',
			modules: [
				{ href: '/campaigns', label: 'Campaigns', desc: 'Marketing pushes, launch plans, ad campaigns, outreach sequences, and performance tracking.', live: true },
				{ href: '/sites', label: 'Sites', desc: 'Public websites, landing pages, funnels, forms, pages, and deployment targets.', live: true },
				{ href: '/personas', label: 'Personas', desc: 'Ideal customers, audience segments, buyer pain, objections, language, and market context.', live: true },
				{ href: '/content', label: 'Content', desc: 'Posts, scripts, newsletters, podcasts, clips, ideas, publishing calendar, and content assets.', live: true }
			]
		},
		{
			label: 'Build',
			purpose: 'The creation layer for apps, assets, deliverables, automation engines, builders, and files.',
			modules: [
				{ href: '/boards', label: 'Boards', desc: 'Compose views of your modules onto one surface, filter to a client, pin to the sidebar.', live: true },
				{ href: '/apps', label: 'Apps', desc: 'Internal tools, generated apps, client apps, app shells, and app lifecycle state.', live: true },
				{ href: '/assets', label: 'Assets', desc: 'Brand files, images, videos, recordings, creative assets, and reusable media.', live: true },
				{ href: '/deliverables', label: 'Deliverables', desc: 'Client-ready outputs, documents, playbooks, reports, builds, and handoff packages.', live: true },
				{ href: '/engines', label: 'Engines', desc: 'Automation systems, workflows, AI engines, data flows, and runtime capabilities.', live: true },
				{ href: '/builders', label: 'Builders', desc: 'No-code and code builders, templates, app creation tools, and repeatable build systems.', live: true },
				{ href: '/drive', label: 'Drive', desc: 'Workspace files, folders, uploads, exports, attachments, and file organization.', live: true }
			]
		},
		{
			label: 'Manage',
			purpose: 'The administrative layer for money, data, teams, integrations, resources, notifications, and controls.',
			modules: [
				{ href: '/finance', label: 'Finance', desc: 'Pricing, revenue, expenses, splits, billing, compensation, and financial visibility.', live: true },
				{ href: '/analytics', label: 'Analytics', desc: 'Metrics, dashboards, performance trends, scorecards, and reporting.', live: true },
				{ href: '/data', label: 'Data', desc: 'Tables, records, datasets, schemas, imports, exports, and structured business data.', live: true },
				{ href: '/team', label: 'Team', desc: 'Team members, roles, responsibilities, permissions, capacity, hiring, and accountability.', live: true },
				{ href: '/connectors', label: 'Connectors', desc: 'Google, Slack, GHL, Fathom, email, calendar, CRM, webhooks, and third-party tools.', live: true },
				{ href: '/resources', label: 'Resources', desc: 'Reference materials, templates, SOPs, training, saved examples, and reusable material.', live: true },
				{ href: '/notifications', label: 'Notifications', desc: 'Alerts, reminders, system messages, approvals, and user-facing updates.', live: true },
				{ href: '/admin', label: 'Admin', desc: 'Workspace settings, access control, billing controls, audit state, and platform management.', live: true },
				{ href: '/help', label: 'Help Center', desc: 'Guides, product help, support references, onboarding material, and operating instructions.', live: true }
			]
		}
	]);

	const modules = $derived(moduleGroups.flatMap((group) => group.modules));

	const liveCount = $derived(modules.filter((m) => m.live).length);

	function greeting(): string {
		const h = new Date().getHours();
		if (h < 12) return 'Good morning';
		if (h < 18) return 'Good afternoon';
		return 'Good evening';
	}

	const firstName = $derived(($session.data?.user?.name ?? '').split(' ')[0] || 'there');
</script>

<div class="dash">
	<header class="dash-head">
		<div>
			<p class="dash-eyebrow">{$currentWorkspace?.name ?? 'Workspace'}</p>
			<h1 class="dash-title">{greeting()}, {firstName}.</h1>
			<p class="dash-sub">Rebuild in progress - {liveCount} of {modules.length} modules live across {moduleGroups.length} operating groups.</p>
		</div>
		<a href="/window" class="dash-desktop-btn">Open Desktop</a>
	</header>

	<section class="dash-glance" aria-label="At a glance">
		{#if glanceLoading}
			<div class="dash-glance-state">
				<Loader2 size={15} class="dash-spin" aria-hidden="true" />
				<span>Loading your at-a-glance</span>
			</div>
		{:else if glanceError}
			<div class="dash-glance-state dash-glance-state--error">
				<span>{glanceError}</span>
				<button type="button" class="dash-glance-retry" onclick={loadGlance}>Retry</button>
			</div>
		{:else}
			<div class="dash-stats">
				{#each stats as s}
					<div class="dash-stat">
						<s.icon size={16} class="dash-stat-icon" aria-hidden="true" />
						<div class="dash-stat-body">
							<span class="dash-stat-value">{s.value}</span>
							<span class="dash-stat-label">{s.label}</span>
						</div>
					</div>
				{/each}
			</div>
		{/if}
	</section>

	<section class="dash-groups">
		{#each moduleGroups as group}
			<section class="dash-group">
				<div class="dash-group-head">
					<h2>{group.label}</h2>
					<p>{group.purpose}</p>
				</div>
				<div class="dash-grid">
					{#each group.modules as m}
						{#if m.live}
							<a href={m.href} class="dash-card dash-card--live">
								<div class="dash-card-top">
									<span class="dash-card-label">{m.label}</span>
									<span class="dash-badge dash-badge--live">Live</span>
								</div>
								<p class="dash-card-desc">{m.desc}</p>
							</a>
						{:else}
							<div class="dash-card dash-card--soon" aria-disabled="true">
								<div class="dash-card-top">
									<span class="dash-card-label">{m.label}</span>
									<span class="dash-badge dash-badge--soon">Soon</span>
								</div>
								<p class="dash-card-desc">{m.desc}</p>
							</div>
						{/if}
					{/each}
				</div>
			</section>
		{/each}
	</section>
</div>

<style>
	.dash {
		height: 100%;
		overflow-y: auto;
		padding: 2.5rem;
		max-width: 1100px;
		margin: 0 auto;
		width: 100%;
	}
	.dash-head {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: 1rem;
		margin-bottom: 2rem;
	}
	.dash-eyebrow {
		font-size: 0.7rem;
		letter-spacing: 0.1em;
		text-transform: uppercase;
		color: var(--dt3);
		margin: 0 0 0.4rem;
		font-weight: 600;
	}
	.dash-title {
		font-size: 1.7rem;
		font-weight: 650;
		letter-spacing: -0.02em;
		color: var(--dt);
		margin: 0;
		line-height: 1.15;
	}
	.dash-sub {
		font-size: 0.88rem;
		color: var(--dt2);
		margin: 0.5rem 0 0;
	}
	.dash-desktop-btn {
		flex-shrink: 0;
		font-size: 0.84rem;
		font-weight: 600;
		padding: 0.6rem 1.1rem;
		border-radius: 10px;
		border: 1px solid var(--dbd);
		background: color-mix(in srgb, var(--dt) 5%, transparent);
		color: var(--dt);
		text-decoration: none;
		transition: background 0.15s ease, border-color 0.15s ease;
	}
	.dash-desktop-btn:hover {
		background: color-mix(in srgb, var(--dt) 10%, transparent);
		border-color: color-mix(in srgb, var(--dt) 20%, transparent);
	}
	.dash-glance {
		margin-bottom: 2rem;
	}
	.dash-stats {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
		gap: 0.8rem;
	}
	.dash-stat {
		display: flex;
		align-items: center;
		gap: 0.7rem;
		border: 1px solid var(--dbd);
		border-radius: 14px;
		padding: 0.9rem 1.05rem;
		background: var(--dbg2);
	}
	.dash-stat :global(.dash-stat-icon) {
		flex-shrink: 0;
		color: var(--dt3);
	}
	.dash-stat-body {
		display: grid;
		gap: 0.1rem;
		min-width: 0;
	}
	.dash-stat-value {
		font-size: 1.35rem;
		font-weight: 650;
		letter-spacing: -0.02em;
		line-height: 1.1;
		color: var(--dt);
	}
	.dash-stat-label {
		font-size: 0.74rem;
		color: var(--dt2);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	.dash-glance-state {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		font-size: 0.82rem;
		color: var(--dt2);
		padding: 0.4rem 0;
	}
	.dash-glance-state--error {
		color: var(--dt2);
	}
	.dash-glance-retry {
		font-size: 0.78rem;
		font-weight: 600;
		color: var(--dt);
		background: none;
		border: none;
		padding: 0;
		cursor: pointer;
		text-decoration: underline;
		text-underline-offset: 2px;
	}
	.dash-glance-state :global(.dash-spin) {
		animation: dash-spin 0.8s linear infinite;
	}
	@keyframes dash-spin {
		to {
			transform: rotate(360deg);
		}
	}

	.dash-groups {
		display: grid;
		gap: 1.4rem;
	}
	.dash-group {
		display: grid;
		gap: 0.72rem;
	}
	.dash-group-head {
		display: grid;
		gap: 0.22rem;
	}
	.dash-group-head h2 {
		margin: 0;
		font-size: 0.82rem;
		font-weight: 700;
		letter-spacing: 0.08em;
		text-transform: uppercase;
		color: var(--dt);
	}
	.dash-group-head p {
		margin: 0;
		max-width: 760px;
		font-size: 0.78rem;
		line-height: 1.45;
		color: var(--dt2);
	}
	.dash-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
		gap: 0.8rem;
	}

	@media (max-width: 768px) {
		.dash {
			padding: 1.5rem 1.25rem;
		}
		.dash-head {
			flex-direction: column;
			align-items: flex-start;
			gap: 0.75rem;
		}
		.dash-desktop-btn {
			width: 100%;
			text-align: center;
		}
		.dash-title {
			font-size: 1.45rem;
		}
		.dash-grid {
			grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
		}
	}

	@media (max-width: 480px) {
		.dash {
			padding: 1rem;
		}
		.dash-title {
			font-size: 1.3rem;
		}
		.dash-grid {
			grid-template-columns: 1fr;
		}
		.dash-group-head p {
			max-width: 100%;
		}
	}
	.dash-card {
		display: block;
		text-decoration: none;
		border: 1px solid var(--dbd);
		border-radius: 14px;
		padding: 1.05rem 1.15rem;
		background: var(--dbg2);
		transition: border-color 0.15s ease, transform 0.15s ease, background 0.15s ease;
	}
	.dash-card--live:hover {
		border-color: color-mix(in srgb, var(--dt) 28%, transparent);
		transform: translateY(-2px);
		background: var(--dbg3);
	}
	.dash-card--soon {
		opacity: 0.5;
		cursor: default;
	}
	.dash-card-top {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-bottom: 0.5rem;
	}
	.dash-card-label {
		font-size: 0.95rem;
		font-weight: 600;
		color: var(--dt);
	}
	.dash-card-desc {
		font-size: 0.8rem;
		color: var(--dt2);
		margin: 0;
		line-height: 1.45;
	}
	.dash-badge {
		font-size: 0.6rem;
		font-weight: 700;
		letter-spacing: 0.06em;
		text-transform: uppercase;
		padding: 0.18rem 0.5rem;
		border-radius: 100px;
	}
	.dash-badge--live {
		background: rgba(34, 197, 94, 0.15);
		color: #16a34a;
	}
	:global(.dark) .dash-badge--live {
		color: #4ade80;
	}
	.dash-badge--soon {
		background: color-mix(in srgb, var(--dt) 9%, transparent);
		color: var(--dt3);
	}
</style>
