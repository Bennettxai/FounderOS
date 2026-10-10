<script lang="ts">
	import { fly } from 'svelte/transition';
	import { goto } from '$app/navigation';

	type ProjectHealth = 'healthy' | 'at_risk' | 'critical';

	interface DashboardProject {
		id: string;
		name: string;
		clientName?: string;
		projectType: string;
		dueDate?: string;
		progress: number;
		health: ProjectHealth;
		teamCount: number;
	}

	interface Props {
		projects?: DashboardProject[];
		isLoading?: boolean;
		onViewAll?: () => void;
	}

	let { projects = [], isLoading = false, onViewAll }: Props = $props();

	const healthDotModifier: Record<ProjectHealth, string> = {
		healthy: 'dw-status-dot--green',
		at_risk: 'dw-status-dot--yellow',
		critical: 'dw-status-dot--red'
	};

	const healthLabels: Record<ProjectHealth, string> = {
		healthy: 'On Track',
		at_risk: 'At Risk',
		critical: 'Critical'
	};

	const visibleProjects = $derived(projects.slice(0, 4));
	const hasMore = $derived(projects.length > 4);

	function getDaysRemaining(dueDate?: string): string {
		if (!dueDate) return '';
		const due = new Date(dueDate);
		const now = new Date();
		const days = Math.ceil((due.getTime() - now.getTime()) / (1000 * 60 * 60 * 24));
		if (days < 0) return `${Math.abs(days)}d overdue`;
		if (days === 0) return 'Due today';
		if (days === 1) return 'Due tomorrow';
		return `${days} days left`;
	}

	function handleProjectClick(projectId: string) {
		goto(`/projects/${projectId}`);
	}
</script>

<div class="dw-widget">
	<!-- Header -->
	<div class="dw-widget-header">
		<div class="dw-widget-title-group">
			<div class="dw-widget-icon">
				<svg class="dw-widget-icon-svg" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
				</svg>
			</div>
			<h2 class="dw-widget-title">Active Projects</h2>
			{#if projects.length > 0}
				<span class="dw-count-badge">{projects.length}</span>
			{/if}
		</div>
	</div>

	<!-- Skeleton loading state -->
	{#if isLoading}
		<div class="dw-proj-skeleton" aria-hidden="true">
			{#each [1, 2, 3] as _}
				<div class="dw-proj-sk-row">
					<div class="dw-proj-sk dw-proj-sk--dot"></div>
					<div class="dw-proj-sk-info">
						<div class="dw-proj-sk dw-proj-sk--name"></div>
						<div class="dw-proj-sk dw-proj-sk--meta"></div>
						<div class="dw-proj-sk dw-proj-sk--bar"></div>
					</div>
					<div class="dw-proj-sk dw-proj-sk--pct"></div>
				</div>
			{/each}
		</div>

	<!-- Empty state -->
	{:else if projects.length === 0}
		<div class="dw-empty-state">
			<svg class="dw-empty-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
			</svg>
			<p class="dw-empty-text">No active projects</p>
			<button onclick={() => goto('/projects')} class="dw-empty-action">
				Create a project
			</button>
		</div>
	{:else}
		<!-- Project list -->
		<div class="dw-projects-list">
			{#each visibleProjects as project, index (project.id)}
				<button
					onclick={() => handleProjectClick(project.id)}
					class="dw-project-row"
					in:fly={{ y: 12, duration: 300, delay: index * 60 }}
					aria-label="Open project {project.name}"
				>
					<!-- Status dot -->
					<span class="dw-status-dot {healthDotModifier[project.health]}" title={healthLabels[project.health]}></span>

					<!-- Project info -->
					<div class="dw-project-info">
						<span class="dw-project-name">{project.name}</span>
						<div class="dw-project-meta">
							<span class="dw-project-meta-label">
								{project.clientName ? project.clientName : project.projectType}
							</span>
							{#if project.dueDate}
								<span class="dw-project-meta-due {project.health === 'critical' ? 'dw-project-meta-due--critical' : ''}">
									{getDaysRemaining(project.dueDate)}
								</span>
							{/if}
						</div>
						<!-- Progress bar -->
						<div class="dw-project-bar-wrap" role="progressbar" aria-valuenow={project.progress} aria-valuemin={0} aria-valuemax={100} aria-label="{project.progress}% complete">
							<div class="dw-project-bar" style="width: {project.progress}%"></div>
						</div>
					</div>

					<!-- Percentage -->
					<span class="dw-project-pct">{project.progress}%</span>
				</button>
			{/each}
		</div>

		<!-- View all footer -->
		{#if hasMore || onViewAll}
			<div class="dw-widget-footer">
				<button onclick={() => onViewAll?.()} class="dw-view-all-link">
					View all projects
					<svg class="dw-chevron-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
					</svg>
				</button>
			</div>
		{/if}
	{/if}
</div>

<style>
	/* ── Widget shell ── */
	.dw-widget {
		background: var(--dbg);
		border: 1px solid var(--dbd);
		border-radius: 12px;
		padding: var(--space-4);
		display: flex;
		flex-direction: column;
		gap: var(--space-4);
	}

	/* ── Header ── */
	.dw-widget-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
	}

	.dw-widget-title-group {
		display: flex;
		align-items: center;
		gap: var(--space-2);
	}

	.dw-widget-icon {
		width: 2rem;
		height: 2rem;
		border-radius: 8px;
		background: var(--dbg3);
		border: 1px solid var(--dbd);
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
	}

	.dw-widget-icon-svg {
		width: 1rem;
		height: 1rem;
		color: var(--dt2);
	}

	.dw-widget-title {
		font-size: 0.9rem;
		font-weight: 600;
		color: var(--dt);
		margin: 0;
	}

	.dw-count-badge {
		font-size: 0.72rem;
		font-weight: 700;
		color: var(--dt3);
		background: var(--dbg3);
		border: 1px solid var(--dbd);
		border-radius: 999px;
		padding: 0.1rem 0.5rem;
		line-height: 1.4;
	}

	/* ── Empty state ── */
	.dw-empty-state {
		display: flex;
		flex-direction: column;
		align-items: center;
		padding: 2rem var(--space-4);
		gap: var(--space-3);
		text-align: center;
	}

	.dw-empty-icon {
		width: 1.5rem;
		height: 1.5rem;
		color: var(--dt3);
		flex-shrink: 0;
	}

	.dw-empty-text {
		font-size: 0.85rem;
		color: var(--dt2);
		margin: 0;
	}

	.dw-empty-action {
		display: inline-flex;
		align-items: center;
		font-size: 0.8rem;
		color: var(--dt2);
		background: transparent;
		border: 1px solid var(--dbd);
		border-radius: 6px;
		padding: var(--space-1) var(--space-3);
		cursor: pointer;
		transition: border-color 0.15s, color 0.15s, background 0.15s;
		height: 28px;
	}

	.dw-empty-action:hover {
		border-color: var(--dt4);
		color: var(--dt);
		background: var(--dbg2);
	}

	/* ── Skeleton ── */
	.dw-proj-skeleton {
		display: flex;
		flex-direction: column;
		gap: var(--space-3);
	}

	.dw-proj-sk-row {
		display: flex;
		align-items: center;
		gap: var(--space-3);
		padding: var(--space-3);
		background: var(--dbg2);
		border: 1px solid var(--dbd);
		border-radius: 8px;
	}

	.dw-proj-sk-info {
		flex: 1;
		display: flex;
		flex-direction: column;
		gap: 0.35rem;
		min-width: 0;
	}

	@keyframes dw-proj-pulse {
		50% { opacity: 0.5; }
	}

	.dw-proj-sk {
		background: var(--dbg3, color-mix(in srgb, var(--dt) 8%, transparent));
		animation: dw-proj-pulse 1.5s ease-in-out infinite;
		border-radius: 4px;
		flex-shrink: 0;
	}

	.dw-proj-sk--dot {
		width: 10px;
		height: 10px;
		border-radius: 50%;
	}

	.dw-proj-sk--name {
		height: 12px;
		width: 70%;
		border-radius: 4px;
	}

	.dw-proj-sk--meta {
		height: 10px;
		width: 45%;
		border-radius: 3px;
	}

	.dw-proj-sk--bar {
		height: 6px;
		width: 100%;
		border-radius: 3px;
	}

	.dw-proj-sk--pct {
		width: 28px;
		height: 12px;
		border-radius: 4px;
	}

	/* ── Project list ── */
	.dw-projects-list {
		display: flex;
		flex-direction: column;
		gap: var(--space-3);
	}

	.dw-project-row {
		display: flex;
		align-items: center;
		gap: var(--space-3);
		padding: var(--space-3);
		background: var(--dbg2);
		border: 1px solid var(--dbd);
		border-radius: 8px;
		cursor: pointer;
		text-align: left;
		width: 100%;
		transition: border-color 0.15s, background 0.15s;
	}

	.dw-project-row:hover {
		border-color: var(--dt4);
		background: var(--dbg3);
	}

	/* ── Status dot ── */
	.dw-status-dot {
		width: 10px;
		height: 10px;
		border-radius: 50%;
		flex-shrink: 0;
		background: var(--dt4);
	}

	.dw-status-dot--green  { background: #22c55e; }
	.dw-status-dot--yellow { background: #eab308; }
	.dw-status-dot--red    { background: #ef4444; }

	/* ── Project info ── */
	.dw-project-info {
		display: flex;
		flex-direction: column;
		gap: 0.35rem;
		flex: 1;
		min-width: 0;
	}

	.dw-project-name {
		font-size: 0.85rem;
		font-weight: 600;
		color: var(--dt);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.dw-project-meta {
		display: flex;
		align-items: center;
		gap: var(--space-2);
	}

	.dw-project-meta-label {
		font-size: 0.75rem;
		color: var(--dt3);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.dw-project-meta-due {
		font-size: 0.72rem;
		color: var(--dt3);
		white-space: nowrap;
	}

	.dw-project-meta-due--critical {
		color: var(--color-error);
	}

	.dw-project-bar-wrap {
		width: 100%;
		height: 6px;
		background: var(--dbd);
		border-radius: 3px;
		overflow: hidden;
	}

	.dw-project-bar {
		height: 100%;
		background: var(--dt2);
		border-radius: 3px;
		transition: width 0.4s ease;
	}

	/* ── Percentage label ── */
	.dw-project-pct {
		font-size: 0.78rem;
		font-weight: 700;
		color: var(--dt3);
		white-space: nowrap;
		flex-shrink: 0;
	}

	/* ── Footer ── */
	.dw-widget-footer {
		display: flex;
		justify-content: center;
		padding-top: var(--space-1);
		border-top: 1px solid var(--dbd2);
	}

	.dw-view-all-link {
		display: inline-flex;
		align-items: center;
		gap: var(--space-1);
		font-size: 0.8rem;
		color: var(--dt3);
		background: none;
		border: none;
		cursor: pointer;
		padding: var(--space-1) var(--space-2);
		border-radius: 4px;
		transition: color 0.15s;
	}

	.dw-view-all-link:hover {
		color: var(--dt);
	}

	.dw-chevron-icon {
		width: 0.8rem;
		height: 0.8rem;
	}
</style>
