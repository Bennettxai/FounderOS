<script lang="ts">
	import type { Task } from '$lib/api';

	interface Props {
		tasks: Task[];
	}

	let { tasks }: Props = $props();

	// Derive sprint-like groupings from tasks by status
	type SprintView = {
		id: string;
		name: string;
		tasks: Task[];
		done: number;
		total: number;
		velocity: number;
	};

	let sprints = $derived.by(() => {
		const todoTasks = tasks.filter(t => t.status === 'todo');
		const inProgressTasks = tasks.filter(t => t.status === 'in_progress');
		const doneTasks = tasks.filter(t => t.status === 'done');

		const views: SprintView[] = [
			{
				id: 'current',
				name: 'Current Sprint',
				tasks: [...inProgressTasks, ...todoTasks.slice(0, Math.max(0, 5 - inProgressTasks.length))],
				done: inProgressTasks.filter(t => t.status === 'done').length,
				total: inProgressTasks.length + Math.min(todoTasks.length, Math.max(0, 5 - inProgressTasks.length)),
				velocity: doneTasks.length
			},
			{
				id: 'backlog',
				name: 'Backlog',
				tasks: todoTasks.slice(Math.max(0, 5 - inProgressTasks.length)),
				done: 0,
				total: Math.max(0, todoTasks.length - Math.max(0, 5 - inProgressTasks.length)),
				velocity: 0
			}
		];
		return views;
	});

	let activeSprint = $state('current');
	let currentSprint = $derived(sprints.find(s => s.id === activeSprint) || sprints[0]);

	// Velocity bars (derived from completed tasks per-week approximation)
	let velocityBars = $derived.by(() => {
		const done = tasks.filter(t => t.status === 'done' && t.completed_at);
		if (done.length === 0) return [0];
		// Group by week
		const weeks = new Map<string, number>();
		for (const t of done) {
			const d = new Date(t.completed_at!);
			const weekKey = `${d.getFullYear()}-W${Math.ceil(d.getDate() / 7)}`;
			weeks.set(weekKey, (weeks.get(weekKey) || 0) + 1);
		}
		const vals = Array.from(weeks.values());
		return vals.length > 0 ? vals.slice(-8) : [0];
	});

	let maxVelocity = $derived(Math.max(...velocityBars, 1));

	function getTypeColor(type: string) {
		switch (type) {
			case 'todo': return 'prm-sp-type--story';
			case 'in_progress': return 'prm-sp-type--task';
			case 'done': return 'prm-sp-type--done';
			case 'cancelled': return 'prm-sp-type--bug';
			default: return 'prm-sp-type--chore';
		}
	}

	function getStatusLabel(status: string) {
		switch (status) {
			case 'todo': return 'To Do';
			case 'in_progress': return 'In Progress';
			case 'done': return 'Done';
			case 'cancelled': return 'Cancelled';
			default: return status;
		}
	}

	function getPriorityDots(priority: string) {
		switch (priority) {
			case 'critical': return 4;
			case 'high': return 3;
			case 'medium': return 2;
			case 'low': return 1;
			default: return 1;
		}
	}
</script>

<div class="prm-sp-container">
	<!-- Sprint Tabs -->
	<div class="prm-sp-tabs">
		{#each sprints as sprint}
			<button
				class="prm-sp-tab {activeSprint === sprint.id ? 'prm-sp-tab--active' : ''}"
				onclick={() => activeSprint = sprint.id}
			>
				{sprint.name}
				<span class="prm-sp-tab-count">{sprint.total}</span>
			</button>
		{/each}
	</div>

	{#if currentSprint}
		<div class="prm-sp-layout">
			<!-- Sprint Card -->
			<div class="prm-sp-card">
				<div class="prm-sp-card-header">
					<h3 class="prm-sp-card-title">{currentSprint.name}</h3>
				</div>
				<div class="prm-sp-card-stats">
					<div class="prm-sp-stat">
						<span class="prm-sp-stat-value">{currentSprint.total}</span>
						<span class="prm-sp-stat-label">Tasks</span>
					</div>
					<div class="prm-sp-stat">
						<span class="prm-sp-stat-value">{currentSprint.done}</span>
						<span class="prm-sp-stat-label">Done</span>
					</div>
					<div class="prm-sp-stat">
						<span class="prm-sp-stat-value">{currentSprint.velocity}</span>
						<span class="prm-sp-stat-label">Completed (all)</span>
					</div>
				</div>

				<!-- Velocity Chart -->
				{#if velocityBars.length > 1}
					<div class="prm-sp-velocity">
						<span class="prm-sp-velocity-label">Velocity</span>
						<div class="prm-sp-velocity-chart">
							{#each velocityBars as bar}
								<div class="prm-sp-velocity-col">
									<div
										class="prm-sp-velocity-bar"
										style="height: {(bar / maxVelocity) * 100}%"
									></div>
								</div>
							{/each}
						</div>
					</div>
				{/if}
			</div>

			<!-- Task List -->
			<div class="prm-sp-backlog">
				<h4 class="prm-sp-backlog-title">Tasks ({currentSprint.tasks.length})</h4>
				{#if currentSprint.tasks.length === 0}
					<div class="prm-sp-empty">
						<p class="prm-sp-empty-text">No tasks in this sprint</p>
					</div>
				{:else}
					<div class="prm-sp-items">
						{#each currentSprint.tasks as task}
							<div class="prm-sp-item">
								<div class="prm-sp-item-drag">
									<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2">
										<path d="M8 6h.01M8 12h.01M8 18h.01M12 6h.01M12 12h.01M12 18h.01" stroke-linecap="round" />
									</svg>
								</div>
								<span class="prm-sp-item-type {getTypeColor(task.status)}">{getStatusLabel(task.status)}</span>
								<span class="prm-sp-item-title">{task.title}</span>
								<span class="prm-sp-item-priority">
									{'●'.repeat(getPriorityDots(task.priority))}
								</span>
							</div>
						{/each}
					</div>
				{/if}
			</div>
		</div>
	{/if}
</div>

<style>
	.prm-sp-container { display: flex; flex-direction: column; gap: 1rem; }

	/* Sprint Tabs */
	.prm-sp-tabs { display: flex; gap: 0.25rem; padding-bottom: 0.75rem; border-bottom: 1px solid var(--dbd2, #f3f4f6); }
	.prm-sp-tab {
		position: relative;
		padding: 0.5rem 0.75rem;
		font-size: 0.8125rem;
		font-weight: 500;
		color: var(--dt3, #6b7280);
		background: none;
		border: none;
		cursor: pointer;
		display: flex;
		align-items: center;
		gap: 0.375rem;
		border-radius: 0.375rem;
		transition: color 0.15s, background 0.15s;
	}
	.prm-sp-tab:hover { color: var(--dt, #111); background: var(--dbg2, #f9fafb); }
	.prm-sp-tab--active { color: var(--dt, #111); font-weight: 600; background: var(--dbg3, #f3f4f6); }
	.prm-sp-tab-count {
		font-size: 0.6875rem; font-weight: 600; padding: 0.0625rem 0.375rem;
		border-radius: 9999px; background: var(--dbg3, #f3f4f6); color: var(--dt3, #6b7280);
	}
	.prm-sp-tab--active .prm-sp-tab-count { background: var(--dbg, #fff); }

	/* Layout */
	.prm-sp-layout { display: grid; grid-template-columns: 16rem 1fr; gap: 1rem; }
	@media (max-width: 768px) {
		.prm-sp-layout { grid-template-columns: 1fr; }
	}

	/* Sprint Card */
	.prm-sp-card { background: var(--dbg, #fff); border: 1px solid var(--dbd, #e5e7eb); border-radius: 0.75rem; padding: 1rem; }
	.prm-sp-card-header { margin-bottom: 0.75rem; }
	.prm-sp-card-title { font-size: 0.875rem; font-weight: 600; color: var(--dt, #111); }
	.prm-sp-card-stats { display: grid; grid-template-columns: repeat(3, 1fr); gap: 0.5rem; margin-bottom: 1rem; }
	.prm-sp-stat { text-align: center; }
	.prm-sp-stat-value { display: block; font-size: 1.125rem; font-weight: 700; color: var(--dt, #111); }
	.prm-sp-stat-label { display: block; font-size: 0.625rem; color: var(--dt4, #9ca3af); }

	/* Velocity Chart */
	.prm-sp-velocity { margin-top: 0.5rem; }
	.prm-sp-velocity-label { display: block; font-size: 0.625rem; font-weight: 600; color: var(--dt3, #6b7280); text-transform: uppercase; margin-bottom: 0.375rem; }
	.prm-sp-velocity-chart { display: flex; align-items: flex-end; gap: 3px; height: 3rem; }
	.prm-sp-velocity-col { flex: 1; display: flex; align-items: flex-end; height: 100%; }
	.prm-sp-velocity-bar { width: 100%; background: var(--dt3, #888); border-radius: 2px 2px 0 0; min-height: 2px; transition: height 0.3s; }

	/* Backlog */
	.prm-sp-backlog { background: var(--dbg, #fff); border: 1px solid var(--dbd, #e5e7eb); border-radius: 0.75rem; padding: 0.75rem; }
	.prm-sp-backlog-title { font-size: 0.8125rem; font-weight: 600; color: var(--dt, #111); padding-bottom: 0.5rem; border-bottom: 1px solid var(--dbd2, #f3f4f6); margin-bottom: 0.5rem; }
	.prm-sp-items { display: flex; flex-direction: column; gap: 0.25rem; }
	.prm-sp-item {
		display: flex; align-items: center; gap: 0.5rem; padding: 0.5rem;
		border-radius: 0.375rem; transition: background 0.15s;
	}
	.prm-sp-item:hover { background: var(--dbg2, #f9fafb); }
	.prm-sp-item-drag { color: var(--dt4, #d1d5db); cursor: grab; flex-shrink: 0; }
	.prm-sp-item-type { font-size: 0.625rem; font-weight: 600; padding: 0.125rem 0.375rem; border-radius: 4px; flex-shrink: 0; text-transform: uppercase; }
	.prm-sp-type--story { background: var(--dbg3, #eee); color: var(--dt, #111); }
	.prm-sp-type--task { background: var(--dbg3, #eee); color: var(--dt2, #555); }
	.prm-sp-type--done { background: var(--dbg3, #eee); color: var(--dt3, #888); }
	.prm-sp-type--bug { background: var(--dbg3, #eee); color: var(--dt, #111); font-weight: 700; }
	.prm-sp-type--chore { background: var(--dbg3, #f3f4f6); color: var(--dt3, #6b7280); }
	.prm-sp-item-title { font-size: 0.8125rem; color: var(--dt, #111); flex: 1; min-width: 0; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
	.prm-sp-item-priority { font-size: 0.625rem; color: var(--dt4, #9ca3af); flex-shrink: 0; }

	/* Empty State */
	.prm-sp-empty { text-align: center; padding: 2rem 1rem; }
	.prm-sp-empty-text { font-size: 0.75rem; color: var(--dt3, #6b7280); }
</style>
