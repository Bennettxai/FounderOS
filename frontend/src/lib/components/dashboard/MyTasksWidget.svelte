<script lang="ts">
	import { fly } from 'svelte/transition';
	import { goto } from '$app/navigation';

	type TaskPriority = 'critical' | 'high' | 'medium' | 'low';
	type TaskBucket = 'overdue' | 'today' | 'upcoming';

	interface DashboardTask {
		id: string;
		title: string;
		projectName?: string;
		dueDate?: string;
		priority: TaskPriority;
		completed: boolean;
	}

	interface Props {
		tasks?: DashboardTask[];
		isLoading?: boolean;
		onToggle?: (id: string) => void;
		onViewAll?: () => void;
	}

	let { tasks = [], isLoading = false, onToggle, onViewAll }: Props = $props();

	// Sort and cap at 5: overdue → today → upcoming
	const visibleTasks = $derived((): Array<DashboardTask & { bucket: TaskBucket }> => {
		const today = new Date();
		today.setHours(0, 0, 0, 0);

		const overdue: Array<DashboardTask & { bucket: TaskBucket }> = [];
		const dueToday: Array<DashboardTask & { bucket: TaskBucket }> = [];
		const upcoming: Array<DashboardTask & { bucket: TaskBucket }> = [];

		for (const task of tasks.filter((t) => !t.completed)) {
			if (!task.dueDate) {
				upcoming.push({ ...task, bucket: 'upcoming' });
				continue;
			}
			const due = new Date(task.dueDate);
			due.setHours(0, 0, 0, 0);

			if (due < today) {
				overdue.push({ ...task, bucket: 'overdue' });
			} else if (due.getTime() === today.getTime()) {
				dueToday.push({ ...task, bucket: 'today' });
			} else {
				upcoming.push({ ...task, bucket: 'upcoming' });
			}
		}

		return [...overdue, ...dueToday, ...upcoming].slice(0, 5);
	});

	const incompleteTotalCount = $derived(tasks.filter((t) => !t.completed).length);
	const hasMore = $derived(incompleteTotalCount > 5);

	function formatDueDate(dueDate?: string): string {
		if (!dueDate) return '';
		const due = new Date(dueDate);
		const today = new Date();
		today.setHours(0, 0, 0, 0);
		const diff = Math.ceil((due.getTime() - today.getTime()) / (1000 * 60 * 60 * 24));

		if (diff < 0) return `${Math.abs(diff)}d overdue`;
		if (diff === 0) return 'Today';
		if (diff === 1) return 'Tomorrow';
		return due.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
	}

	function handleToggle(id: string) {
		onToggle?.(id);
	}
</script>

<div class="dw-widget">
	<!-- Header -->
	<div class="dw-widget-header">
		<div class="dw-widget-title-group">
			<div class="dw-widget-icon">
				<svg class="dw-widget-icon-svg" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4" />
				</svg>
			</div>
			<h2 class="dw-widget-title">My Tasks</h2>
			{#if incompleteTotalCount > 0}
				<span class="dw-count-badge">{incompleteTotalCount}</span>
			{/if}
		</div>
	</div>

	<!-- Skeleton loading state -->
	{#if isLoading}
		<div class="dw-tasks-skeleton" aria-hidden="true">
			{#each [1, 2, 3, 4] as _}
				<div class="dw-tasks-sk-row">
					<div class="dw-tasks-sk dw-tasks-sk--check"></div>
					<div class="dw-tasks-sk-info">
						<div class="dw-tasks-sk dw-tasks-sk--title"></div>
						<div class="dw-tasks-sk dw-tasks-sk--sub"></div>
					</div>
					<div class="dw-tasks-sk dw-tasks-sk--date"></div>
				</div>
			{/each}
		</div>

	<!-- Empty state -->
	{:else if tasks.length === 0 || incompleteTotalCount === 0}
		<div class="dw-empty-state">
			<svg class="dw-empty-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M5 13l4 4L19 7" />
			</svg>
			<p class="dw-empty-text">All caught up — no tasks due soon</p>
			<button onclick={() => goto('/tasks')} class="dw-empty-action">
				Add a task
			</button>
		</div>
	{:else}
		<!-- Task list -->
		<div class="dw-tasks-list">
			{#each visibleTasks() as task, index (task.id)}
				<div
					class="dw-task-row"
					in:fly={{ x: -8, duration: 200, delay: index * 40 }}
				>
					<!-- Checkbox -->
					<button
						onclick={() => handleToggle(task.id)}
						class="dw-task-checkbox"
						aria-label="Mark '{task.title}' complete"
					>
						<svg class="dw-task-checkbox-check" fill="none" stroke="currentColor" viewBox="0 0 12 12" aria-hidden="true">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2 6l3 3 5-5" />
						</svg>
					</button>

					<!-- Task content -->
					<div class="dw-task-info">
						<span class="dw-task-title">{task.title}</span>
						{#if task.projectName}
							<span class="dw-task-project">{task.projectName}</span>
						{/if}
					</div>

					<!-- Right side: due date + priority dot -->
					<div class="dw-task-meta">
						{#if task.dueDate}
							<span class="dw-task-due {task.bucket === 'overdue' ? 'dw-task-due--overdue' : ''}">
								{formatDueDate(task.dueDate)}
							</span>
						{/if}
						<span class="dw-priority-dot dw-priority-dot--{task.priority}" title={task.priority}></span>
					</div>
				</div>
			{/each}
		</div>

		<!-- View all footer -->
		{#if hasMore || onViewAll}
			<div class="dw-widget-footer">
				<button onclick={() => onViewAll?.()} class="dw-view-all-link">
					View all tasks
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
	.dw-tasks-skeleton {
		display: flex;
		flex-direction: column;
		gap: var(--space-2);
	}

	.dw-tasks-sk-row {
		display: flex;
		align-items: center;
		gap: var(--space-3);
		padding: var(--space-2) var(--space-3);
		background: var(--dbg2);
		border: 1px solid var(--dbd);
		border-radius: 8px;
	}

	.dw-tasks-sk-info {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: 0.2rem;
	}

	@keyframes dw-tasks-pulse {
		50% { opacity: 0.5; }
	}

	.dw-tasks-sk {
		background: var(--dbg3, color-mix(in srgb, var(--dt) 8%, transparent));
		animation: dw-tasks-pulse 1.5s ease-in-out infinite;
		border-radius: 4px;
		flex-shrink: 0;
	}

	.dw-tasks-sk--check {
		width: 1.1rem;
		height: 1.1rem;
		border-radius: 4px;
	}

	.dw-tasks-sk--title {
		height: 12px;
		width: 65%;
		border-radius: 4px;
	}

	.dw-tasks-sk--sub {
		height: 10px;
		width: 40%;
		border-radius: 3px;
	}

	.dw-tasks-sk--date {
		width: 36px;
		height: 10px;
		border-radius: 3px;
	}

	/* ── Task list ── */
	.dw-tasks-list {
		display: flex;
		flex-direction: column;
		gap: var(--space-2);
	}

	.dw-task-row {
		display: flex;
		align-items: center;
		gap: var(--space-3);
		padding: var(--space-2) var(--space-3);
		background: var(--dbg2);
		border: 1px solid var(--dbd);
		border-radius: 8px;
		transition: border-color 0.15s, background 0.15s;
	}

	.dw-task-row:hover {
		border-color: var(--dt4);
		background: var(--dbg3);
	}

	/* ── Checkbox ── */
	.dw-task-checkbox {
		flex-shrink: 0;
		width: 1.1rem;
		height: 1.1rem;
		border-radius: 4px;
		border: 1.5px solid var(--dbd);
		background: var(--dbg);
		display: flex;
		align-items: center;
		justify-content: center;
		cursor: pointer;
		transition: border-color 0.15s, background 0.15s;
		padding: 0;
	}

	.dw-task-checkbox:hover {
		border-color: var(--dt3);
		background: var(--dbg3);
	}

	.dw-task-checkbox-check {
		width: 0.65rem;
		height: 0.65rem;
		color: transparent;
	}

	.dw-task-checkbox:hover .dw-task-checkbox-check {
		color: var(--dt4);
	}

	/* ── Task info ── */
	.dw-task-info {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: 0.2rem;
	}

	.dw-task-title {
		font-size: 0.83rem;
		font-weight: 500;
		color: var(--dt);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.dw-task-project {
		font-size: 0.72rem;
		color: var(--dt3);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	/* ── Task meta (right side) ── */
	.dw-task-meta {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		flex-shrink: 0;
	}

	.dw-task-due {
		font-size: 0.72rem;
		color: var(--dt3);
		white-space: nowrap;
	}

	.dw-task-due--overdue {
		color: var(--color-error);
	}

	/* ── Priority dot ── */
	.dw-priority-dot {
		width: 7px;
		height: 7px;
		border-radius: 50%;
		flex-shrink: 0;
	}

	.dw-priority-dot--critical { background: var(--color-error); }
	.dw-priority-dot--high     { background: var(--accent-orange); }
	.dw-priority-dot--medium   { background: var(--accent-blue); }
	.dw-priority-dot--low      { background: var(--dt3); }

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
