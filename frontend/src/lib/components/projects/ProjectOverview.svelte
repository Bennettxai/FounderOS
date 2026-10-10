<script lang="ts">
	import type { Project, Task, TeamMemberListResponse, ClientListResponse } from '$lib/api';
	import { api } from '$lib/api';
	import { getTypeLabel, formatDate, getPriorityDotVar } from '$lib/utils/project';

	interface Props {
		project: Project;
		tasks: Task[];
		teamMembers: TeamMemberListResponse[];
		clients: ClientListResponse[];
		embedSuffix: string;
		onProjectUpdate: () => Promise<void>;
		onNavigateToTasks: () => void;
		onShowAddTask: () => void;
		onShowAssignTeam: () => void;
	}

	let {
		project,
		tasks,
		teamMembers,
		clients,
		embedSuffix,
		onProjectUpdate,
		onNavigateToTasks,
		onShowAddTask,
		onShowAssignTeam
	}: Props = $props();

	let completedTasks = $derived(tasks.filter((t) => t.status === 'done').length);
	let totalTasks = $derived(tasks.length);
	let completionPct = $derived(totalTasks > 0 ? Math.round((completedTasks / totalTasks) * 100) : 0);

	// Metrics
	let todoCount = $derived(tasks.filter(t => t.status === 'todo').length);
	let inProgressCount = $derived(tasks.filter(t => t.status === 'in_progress').length);
	let cancelledCount = $derived(tasks.filter(t => t.status === 'cancelled').length);

	let criticalCount = $derived(tasks.filter(t => t.priority === 'critical').length);
	let highCount = $derived(tasks.filter(t => t.priority === 'high').length);

	// SVG donut for completion ring
	let ringRadius = 36;
	let ringCircumference = $derived(2 * Math.PI * ringRadius);
	let ringOffset = $derived(ringCircumference - (completionPct / 100) * ringCircumference);

	function handleToggleTask(taskId: string) {
		api.toggleTask(taskId).then(() => {});
	}
</script>

<div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
	<!-- Main Content -->
	<div class="lg:col-span-2 space-y-6">
		<!-- Description -->
		<div class="prm-ov-card">
			<h2 class="prm-ov-heading">
				<svg class="w-5 h-5 prm-ov-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h7" />
				</svg>
				Description
			</h2>
			{#if project.description}
				<p class="prm-ov-desc">{project.description}</p>
			{:else}
				<p class="prm-ov-empty">No description added yet. Click Edit to add one.</p>
			{/if}
		</div>

		<!-- Recent Tasks -->
		<div class="prm-ov-card">
			<div class="flex items-center justify-between mb-4">
				<h2 class="prm-ov-heading" style="margin-bottom:0">
					<svg class="w-5 h-5 prm-ov-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
					</svg>
					Tasks
				</h2>
				<button
					onclick={() => { onNavigateToTasks(); onShowAddTask(); }}
					class="btn-pill btn-pill-ghost btn-pill-sm"
				>
					+ Add Task
				</button>
			</div>
			{#if tasks.length === 0}
				<div class="text-center py-8">
					<div class="w-12 h-12 rounded-full prm-ov-empty-circle flex items-center justify-center mx-auto mb-3">
						<svg class="w-6 h-6 prm-ov-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
						</svg>
					</div>
					<p class="prm-ov-muted mb-2">No tasks yet</p>
					<button
						onclick={() => { onNavigateToTasks(); onShowAddTask(); }}
						class="btn-pill btn-pill-primary btn-pill-sm"
					>
						Add First Task
					</button>
				</div>
			{:else}
				<div class="space-y-2">
					{#each tasks.slice(0, 5) as task}
						<div class="flex items-center gap-3 p-3 rounded-lg prm-ov-task-row group">
							<button
								onclick={() => handleToggleTask(task.id)}
								class="w-5 h-5 rounded border-2 flex items-center justify-center flex-shrink-0 transition-colors {task.status === 'done' ? 'prm-ov-checkbox--done' : 'prm-ov-checkbox'}"
								aria-label="Toggle task complete"
							>
								{#if task.status === 'done'}
									<svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="3" d="M5 13l4 4L19 7" />
									</svg>
								{/if}
							</button>
							<div class="flex-1 min-w-0">
								<p class="text-sm {task.status === 'done' ? 'prm-ov-done' : 'prm-ov-text'}">{task.title}</p>
								{#if task.due_date}
									<p class="text-xs prm-ov-meta">Due {formatDate(task.due_date)}</p>
								{/if}
							</div>
							<span class="prm-ov-priority-badge">
							<span class="prm-ov-priority-dot" style="background: {getPriorityDotVar(task.priority)}"></span>
							{task.priority}
						</span>
						</div>
					{/each}
					{#if tasks.length > 5}
						<button
							onclick={onNavigateToTasks}
							class="btn-pill btn-pill-ghost btn-pill-sm w-full text-center"
						>
							View all {tasks.length} tasks
						</button>
					{/if}
				</div>
			{/if}
		</div>

		<!-- Metrics Dashboard -->
		{#if totalTasks > 0}
			<div class="prm-ov-metrics">
				<h2 class="prm-ov-heading">
					<svg class="w-5 h-5 prm-ov-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
					</svg>
					Metrics
				</h2>
				<div class="prm-ov-metrics-grid">
					<!-- Completion Ring -->
					<div class="prm-ov-metric-card prm-ov-metric-card--ring">
						<svg viewBox="0 0 80 80" class="prm-ov-ring-svg">
							<circle cx="40" cy="40" r={ringRadius} fill="none" stroke="var(--dbg3, #e5e7eb)" stroke-width="6" />
							<circle cx="40" cy="40" r={ringRadius} fill="none" stroke="var(--dt3, #888)" stroke-width="6"
								stroke-dasharray={ringCircumference} stroke-dashoffset={ringOffset}
								stroke-linecap="round" transform="rotate(-90 40 40)"
								style="transition: stroke-dashoffset 0.5s"
							/>
							<text x="40" y="38" text-anchor="middle" class="prm-ov-ring-pct">{completionPct}%</text>
							<text x="40" y="50" text-anchor="middle" class="prm-ov-ring-label">done</text>
						</svg>
					</div>

					<!-- Status Breakdown -->
					<div class="prm-ov-metric-card">
						<span class="prm-ov-metric-title">Status</span>
						<div class="prm-ov-metric-rows">
							<div class="prm-ov-metric-row">
								<span class="prm-ov-metric-dot" style="background: var(--bos-status-success)"></span>
								<span class="prm-ov-metric-name">Done</span>
								<span class="prm-ov-metric-val">{completedTasks}</span>
							</div>
							<div class="prm-ov-metric-row">
								<span class="prm-ov-metric-dot" style="background: var(--bos-status-info)"></span>
								<span class="prm-ov-metric-name">In Progress</span>
								<span class="prm-ov-metric-val">{inProgressCount}</span>
							</div>
							<div class="prm-ov-metric-row">
								<span class="prm-ov-metric-dot" style="background: var(--bos-status-neutral)"></span>
								<span class="prm-ov-metric-name">To Do</span>
								<span class="prm-ov-metric-val">{todoCount}</span>
							</div>
							{#if cancelledCount > 0}
								<div class="prm-ov-metric-row">
									<span class="prm-ov-metric-dot" style="background: var(--bos-status-error)"></span>
									<span class="prm-ov-metric-name">Cancelled</span>
									<span class="prm-ov-metric-val">{cancelledCount}</span>
								</div>
							{/if}
						</div>
					</div>

					<!-- Priority Breakdown -->
					<div class="prm-ov-metric-card">
						<span class="prm-ov-metric-title">Priority</span>
						<div class="prm-ov-metric-rows">
							{#if criticalCount > 0}
								<div class="prm-ov-metric-row">
									<span class="prm-ov-metric-dot" style="background: var(--bos-status-error)"></span>
									<span class="prm-ov-metric-name">Critical</span>
									<span class="prm-ov-metric-val">{criticalCount}</span>
								</div>
							{/if}
							{#if highCount > 0}
								<div class="prm-ov-metric-row">
									<span class="prm-ov-metric-dot" style="background: var(--bos-priority-high)"></span>
									<span class="prm-ov-metric-name">High</span>
									<span class="prm-ov-metric-val">{highCount}</span>
								</div>
							{/if}
							<div class="prm-ov-metric-row">
								<span class="prm-ov-metric-dot" style="background: var(--bos-priority-medium)"></span>
								<span class="prm-ov-metric-name">Medium</span>
								<span class="prm-ov-metric-val">{tasks.filter(t => t.priority === 'medium').length}</span>
							</div>
							<div class="prm-ov-metric-row">
								<span class="prm-ov-metric-dot" style="background: var(--bos-status-success)"></span>
								<span class="prm-ov-metric-name">Low</span>
								<span class="prm-ov-metric-val">{tasks.filter(t => t.priority === 'low').length}</span>
							</div>
						</div>
					</div>
				</div>
			</div>
		{/if}
	</div>

	<!-- Sidebar -->
	<div class="space-y-6">
		<!-- Quick Actions -->
		<div class="prm-ov-card">
			<h2 class="prm-ov-heading">Quick Actions</h2>
			<div class="space-y-2">
				{#if project.status !== 'completed'}
					<button
						onclick={async () => {
							await api.updateProject(project.id, { status: 'completed' });
							await onProjectUpdate();
						}}
						class="btn-pill btn-pill-ghost btn-pill-sm w-full justify-start"
					>
						<svg class="w-4 h-4 mr-2 prm-ov-action-icon--green" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
						</svg>
						Mark Complete
					</button>
				{/if}
				{#if project.status === 'active'}
					<button
						onclick={async () => {
							await api.updateProject(project.id, { status: 'paused' });
							await onProjectUpdate();
						}}
						class="btn-pill btn-pill-ghost btn-pill-sm w-full justify-start"
					>
						<svg class="w-4 h-4 mr-2 prm-ov-action-icon--amber" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 9v6m4-6v6m7-3a9 9 0 11-18 0 9 9 0 0118 0z" />
						</svg>
						Pause Project
					</button>
				{:else if project.status === 'paused'}
					<button
						onclick={async () => {
							await api.updateProject(project.id, { status: 'active' });
							await onProjectUpdate();
						}}
						class="btn-pill btn-pill-ghost btn-pill-sm w-full justify-start"
					>
						<svg class="w-4 h-4 mr-2 prm-ov-action-icon--green" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z" />
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
						</svg>
						Resume Project
					</button>
				{/if}
				<button
					onclick={() => { onNavigateToTasks(); onShowAddTask(); }}
					class="btn-pill btn-pill-ghost btn-pill-sm w-full justify-start"
				>
						<svg class="w-4 h-4 mr-2 prm-ov-action-icon--purple" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
					</svg>
					Add Task
				</button>
				<a href="/knowledge{embedSuffix}" class="btn-pill btn-pill-ghost btn-pill-sm w-full justify-start">
					<svg class="w-4 h-4 mr-2 prm-ov-action-icon--blue" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
					</svg>
					View Documents
				</a>
				{#if project.status !== 'archived'}
					<button
						onclick={async () => {
							await api.updateProject(project.id, { status: 'archived' });
							await onProjectUpdate();
						}}
						class="btn-pill btn-pill-ghost btn-pill-sm w-full justify-start prm-ov-muted"
					>
						<svg class="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 8h14M5 8a2 2 0 110-4h14a2 2 0 110 4M5 8v10a2 2 0 002 2h10a2 2 0 002-2V8m-9 4h4" />
						</svg>
						Archive
					</button>
				{/if}
			</div>
		</div>

		<!-- Details -->
		<div class="prm-ov-card">
			<h2 class="prm-ov-heading">Details</h2>
			<dl class="space-y-3">
				<div>
					<dt class="prm-ov-dt">Status</dt>
					<dd class="text-sm font-medium capitalize">{project.status}</dd>
				</div>
				<div>
					<dt class="prm-ov-dt">Priority</dt>
					<dd class="prm-ov-priority-badge" style="font-size: 0.875rem;">
						<span class="prm-ov-priority-dot" style="background: {getPriorityDotVar(project.priority)}"></span>
						{project.priority}
					</dd>
				</div>
				<div>
					<dt class="prm-ov-dt">Type</dt>
					<dd class="prm-ov-dd">{getTypeLabel(project.project_type)}</dd>
				</div>
				{#if project.client_name}
					<div>
						<dt class="prm-ov-dt">Client</dt>
						<dd class="prm-ov-dd">{project.client_name}</dd>
					</div>
				{/if}
				<div>
					<dt class="prm-ov-dt">Created</dt>
					<dd class="prm-ov-dd">{formatDate(project.created_at)}</dd>
				</div>
				<div>
					<dt class="prm-ov-dt">Last Updated</dt>
					<dd class="prm-ov-dd">{formatDate(project.updated_at)}</dd>
				</div>
			</dl>
		</div>

		<!-- Team Members -->
		{#if teamMembers.length > 0}
			<div class="prm-ov-card">
				<div class="flex items-center justify-between mb-3">
					<h2 class="prm-ov-heading" style="margin-bottom:0">Team</h2>
					<button onclick={onShowAssignTeam} class="btn-pill btn-pill-ghost btn-pill-sm">
						+ Assign
					</button>
				</div>
				<div class="space-y-2">
					{#each teamMembers.slice(0, 3) as member}
						<div class="flex items-center gap-2 p-2 rounded-lg prm-ov-task-row">
							<div class="prm-ov-avatar">
								{member.name.split(' ').map((n: string) => n[0]).join('').slice(0, 2)}
							</div>
							<div class="flex-1 min-w-0">
								<p class="text-sm font-medium prm-ov-text truncate">{member.name}</p>
								<p class="text-xs prm-ov-meta truncate">{member.role}</p>
							</div>
						</div>
					{/each}
					{#if teamMembers.length > 3}
						<p class="text-xs prm-ov-meta text-center">+{teamMembers.length - 3} more</p>
					{/if}
				</div>
			</div>
		{/if}
	</div>
</div>

<style>
	.prm-ov-card {
		background: var(--dbg, #fff);
		border: 1px solid var(--dbd, #e0e0e0);
		border-radius: 0.75rem;
		padding: 1.5rem;
	}
	.prm-ov-heading {
		font-size: 1.125rem;
		font-weight: 500;
		color: var(--dt, #111);
		margin-bottom: 0.75rem;
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}
	.prm-ov-icon {
		color: var(--dt3, #888);
	}
	.prm-ov-desc {
		color: var(--dt2, #555);
		white-space: pre-wrap;
	}
	.prm-ov-empty {
		color: var(--dt3, #888);
		font-style: italic;
	}
	.prm-ov-text {
		color: var(--dt, #111);
	}
	.prm-ov-muted {
		color: var(--dt2, #555);
	}
	.prm-ov-meta {
		color: var(--dt3, #888);
	}
	.prm-ov-done {
		color: var(--dt3, #888);
		text-decoration: line-through;
	}
	.prm-ov-task-row:hover {
		background: var(--dbg2, #f5f5f5);
	}
	.prm-ov-checkbox {
		border-color: var(--dbd, #ccc);
	}
	.prm-ov-checkbox:hover {
		border-color: var(--dt, #111);
	}
	.prm-ov-empty-circle {
		background: var(--dbg2, #f5f5f5);
	}
	.prm-ov-checkbox--done { background: var(--dt, #111); border-color: var(--dt, #111); color: var(--bos-surface-on-color, #fff); }
	.prm-ov-action-icon--green { color: var(--dt2, #555); }
	.prm-ov-action-icon--amber { color: var(--dt2, #555); }
	.prm-ov-action-icon--purple { color: var(--dt2, #555); }
	.prm-ov-action-icon--blue { color: var(--dt2, #555); }
	.prm-ov-dt {
		font-size: 0.75rem;
		color: var(--dt2, #555);
		text-transform: uppercase;
	}
	.prm-ov-dd {
		font-size: 0.875rem;
		color: var(--dt, #111);
	}
	.prm-ov-avatar {
		width: 2rem;
		height: 2rem;
		border-radius: 50%;
		background: var(--dt3, #888);
		color: #fff;
		font-size: 0.6875rem;
		font-weight: 600;
		line-height: 2rem;
		text-align: center;
		flex-shrink: 0;
		letter-spacing: 0.02em;
	}

	/* Priority badge */
	.prm-ov-priority-badge {
		display: inline-flex;
		align-items: center;
		gap: 0.375rem;
		font-size: 0.75rem;
		font-weight: 500;
		color: var(--dt2, #555);
		text-transform: capitalize;
	}
	.prm-ov-priority-dot {
		width: 6px;
		height: 6px;
		border-radius: 50%;
		flex-shrink: 0;
	}

	/* Metrics Dashboard */
	.prm-ov-metrics { background: var(--dbg, #fff); border: 1px solid var(--dbd, #e0e0e0); border-radius: 0.75rem; padding: 1.5rem; }
	.prm-ov-metrics-grid { display: grid; grid-template-columns: 5rem 1fr 1fr; gap: 1rem; align-items: start; }
	@media (max-width: 640px) { .prm-ov-metrics-grid { grid-template-columns: 1fr; } }
	.prm-ov-metric-card { padding: 0.5rem; }
	.prm-ov-metric-card--ring { display: flex; align-items: center; justify-content: center; }
	.prm-ov-ring-svg { width: 5rem; height: 5rem; }
	.prm-ov-ring-pct { font-size: 0.875rem; font-weight: 700; fill: var(--dt, #111); }
	.prm-ov-ring-label { font-size: 0.5rem; fill: var(--dt3, #6b7280); text-transform: uppercase; }
	.prm-ov-metric-title { display: block; font-size: 0.6875rem; font-weight: 600; color: var(--dt3, #6b7280); text-transform: uppercase; margin-bottom: 0.5rem; }
	.prm-ov-metric-rows { display: flex; flex-direction: column; gap: 0.375rem; }
	.prm-ov-metric-row { display: flex; align-items: center; gap: 0.5rem; font-size: 0.75rem; }
	.prm-ov-metric-dot { width: 0.5rem; height: 0.5rem; border-radius: 50%; flex-shrink: 0; }
	.prm-ov-metric-name { color: var(--dt2, #555); flex: 1; }
	.prm-ov-metric-val { font-weight: 600; color: var(--dt, #111); }
</style>
