<script lang="ts">
	import type { Task, TeamMemberListResponse } from '$lib/api';

	interface Props {
		tasks: Task[];
		teamMembers: TeamMemberListResponse[];
	}

	let { tasks, teamMembers }: Props = $props();

	// Generate 14-day range centered around today
	const today = new Date();
	const startDate = new Date(today);
	startDate.setDate(startDate.getDate() - 3);

	let days = $derived(Array.from({ length: 14 }, (_, i) => {
		const d = new Date(startDate);
		d.setDate(d.getDate() + i);
		return d;
	}));

	let todayIndex = $derived(days.findIndex(d =>
		d.toDateString() === today.toDateString()
	));

	// Group tasks by assignee
	type TimelineRow = {
		group: string;
		tasks: {
			id: string;
			label: string;
			startCol: number;
			span: number;
			color: string;
			status: 'done' | 'active' | 'upcoming';
		}[];
	};

	let timelineRows = $derived.by(() => {
		const tasksByAssignee = new Map<string, Task[]>();

		for (const task of tasks) {
			if (!task.start_date && !task.due_date) continue;
			const key = task.assignee_id || 'unassigned';
			if (!tasksByAssignee.has(key)) tasksByAssignee.set(key, []);
			tasksByAssignee.get(key)!.push(task);
		}

		const rows: TimelineRow[] = [];
		const colors = ['var(--dt, #111)', 'var(--dt2, #555)', 'var(--dt3, #888)', 'var(--dt4, #bbb)', 'var(--dt2, #555)', 'var(--dt3, #888)'];
		let colorIdx = 0;

		for (const [assigneeId, assigneeTasks] of tasksByAssignee) {
			const member = teamMembers.find(m => m.id === assigneeId);
			const group = member ? member.name : (assigneeId === 'unassigned' ? 'Unassigned' : 'Unknown');
			const color = colors[colorIdx++ % colors.length];

			const mappedTasks = assigneeTasks.map(t => {
				const tStart = t.start_date ? new Date(t.start_date) : (t.due_date ? new Date(t.due_date) : today);
				const tEnd = t.due_date ? new Date(t.due_date) : tStart;

				const startCol = Math.max(0, Math.round((tStart.getTime() - startDate.getTime()) / (1000 * 60 * 60 * 24)));
				const endCol = Math.max(startCol + 1, Math.round((tEnd.getTime() - startDate.getTime()) / (1000 * 60 * 60 * 24)) + 1);

				return {
					id: t.id,
					label: t.title,
					startCol: Math.min(startCol, 13),
					span: Math.max(1, Math.min(endCol - startCol, 14 - startCol)),
					color,
					status: t.status === 'done' ? 'done' as const : (t.status === 'in_progress' ? 'active' as const : 'upcoming' as const)
				};
			});

			rows.push({ group, tasks: mappedTasks });
		}

		return rows;
	});

	// Tasks that have no dates — show separately
	let undatedTasks = $derived(tasks.filter(t => !t.start_date && !t.due_date));

	function formatDay(d: Date) {
		return d.toLocaleDateString(undefined, { weekday: 'short' });
	}

	function formatDayNum(d: Date) {
		return d.getDate();
	}

	function isWeekend(d: Date) {
		return d.getDay() === 0 || d.getDay() === 6;
	}
</script>

<div class="prm-tl-container">
	{#if timelineRows.length === 0}
		<!-- Empty State -->
		<div class="prm-tl-empty">
			<div class="prm-tl-empty-circle">
				<svg class="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
				</svg>
			</div>
			<h3 class="prm-tl-empty-title">No timeline data</h3>
			<p class="prm-tl-empty-desc">Add start and due dates to tasks to see them on the timeline.</p>
		</div>
	{:else}
		<!-- Legend -->
		<div class="prm-tl-legend">
			<div class="prm-tl-legend-item">
				<span class="prm-tl-legend-dot prm-tl-legend-dot--done"></span>
				<span>Done</span>
			</div>
			<div class="prm-tl-legend-item">
				<span class="prm-tl-legend-dot prm-tl-legend-dot--active"></span>
				<span>In Progress</span>
			</div>
			<div class="prm-tl-legend-item">
				<span class="prm-tl-legend-dot prm-tl-legend-dot--upcoming"></span>
				<span>Upcoming</span>
			</div>
			<div class="prm-tl-legend-item prm-tl-legend-item--today">
				<span class="prm-tl-legend-line"></span>
				<span>Today</span>
			</div>
		</div>

		<!-- Gantt Grid -->
		<div class="prm-tl-grid">
			<!-- Header Row: Day Labels -->
			<div class="prm-tl-row-label prm-tl-header-corner"></div>
			{#each days as day, i}
				<div class="prm-tl-day-header {isWeekend(day) ? 'prm-tl-day-header--weekend' : ''} {i === todayIndex ? 'prm-tl-day-header--today' : ''}">
					<span class="prm-tl-day-name">{formatDay(day)}</span>
					<span class="prm-tl-day-num">{formatDayNum(day)}</span>
				</div>
			{/each}

			<!-- Task Rows -->
			{#each timelineRows as row}
				<div class="prm-tl-row-label">{row.group}</div>
				<div class="prm-tl-row" style="grid-column: 2 / -1;">
					<!-- Day cells -->
					{#each days as _, i}
						<div class="prm-tl-cell {isWeekend(days[i]) ? 'prm-tl-cell--weekend' : ''} {i === todayIndex ? 'prm-tl-cell--today' : ''}"></div>
					{/each}
					<!-- Task bars -->
					{#each row.tasks as task}
						<div
							class="prm-tl-bar prm-tl-bar--{task.status}"
							style="grid-column: {task.startCol + 1} / span {task.span}; background-color: {task.status === 'done' ? 'color-mix(in srgb, ' + task.color + ' 40%, var(--dbg3, #e5e7eb))' : task.color};"
							title={task.label}
						>
							<span class="prm-tl-bar-label">{task.label}</span>
						</div>
					{/each}
				</div>
			{/each}

			<!-- Today Marker -->
			{#if todayIndex >= 0}
				<div class="prm-tl-today-marker" style="grid-column: {todayIndex + 2}; grid-row: 1 / -1;"></div>
			{/if}
		</div>
	{/if}

	<!-- Undated Tasks -->
	{#if undatedTasks.length > 0}
		<div class="prm-tl-undated">
			<h4 class="prm-tl-undated-title">Tasks without dates ({undatedTasks.length})</h4>
			<div class="prm-tl-undated-list">
				{#each undatedTasks.slice(0, 8) as task}
					<div class="prm-tl-undated-item">
						<span class="prm-tl-undated-dot prm-tl-undated-dot--{task.status}"></span>
						<span class="prm-tl-undated-label">{task.title}</span>
					</div>
				{/each}
				{#if undatedTasks.length > 8}
					<span class="prm-tl-undated-more">+{undatedTasks.length - 8} more</span>
				{/if}
			</div>
		</div>
	{/if}
</div>

<style>
	.prm-tl-container { display: flex; flex-direction: column; gap: 1rem; }

	/* Legend */
	.prm-tl-legend { display: flex; align-items: center; gap: 1rem; font-size: 0.6875rem; color: var(--dt3, #6b7280); flex-wrap: wrap; }
	.prm-tl-legend-item { display: flex; align-items: center; gap: 0.375rem; }
	.prm-tl-legend-dot { width: 0.5rem; height: 0.5rem; border-radius: 2px; }
	.prm-tl-legend-dot--done { background: var(--dt4, #bbb); }
	.prm-tl-legend-dot--active { background: var(--dt, #111); }
	.prm-tl-legend-dot--upcoming { background: var(--dbg3, #e5e7eb); border: 1px solid var(--dbd, #d1d5db); }
	.prm-tl-legend-line { width: 0.75rem; height: 2px; background: var(--dt3, #888); }
	.prm-tl-legend-item--today { margin-left: auto; }

	/* Gantt Grid */
	.prm-tl-grid {
		display: grid;
		grid-template-columns: 7rem repeat(14, 1fr);
		position: relative;
		background: var(--dbg, #fff);
		border: 1px solid var(--dbd, #e5e7eb);
		border-radius: 0.75rem;
		overflow: hidden;
	}
	.prm-tl-header-corner { background: var(--dbg, #fff); border-bottom: 1px solid var(--dbd, #e5e7eb); border-right: 1px solid var(--dbd2, #f3f4f6); }
	.prm-tl-day-header {
		padding: 0.375rem 0.25rem;
		text-align: center;
		border-bottom: 1px solid var(--dbd, #e5e7eb);
		border-right: 1px solid var(--dbd2, #f3f4f6);
		background: var(--dbg, #fff);
	}
	.prm-tl-day-header--weekend { background: var(--dbg2, #f9fafb); }
	.prm-tl-day-header--today { background: var(--dbg2, #f5f5f5); }
	.prm-tl-day-name { display: block; font-size: 0.625rem; font-weight: 500; color: var(--dt4, #9ca3af); text-transform: uppercase; }
	.prm-tl-day-num { display: block; font-size: 0.8125rem; font-weight: 600; color: var(--dt, #111); line-height: 1.2; }

	/* Row Labels */
	.prm-tl-row-label {
		padding: 0.5rem 0.75rem;
		font-size: 0.75rem;
		font-weight: 600;
		color: var(--dt2, #4b5563);
		border-right: 1px solid var(--dbd2, #f3f4f6);
		border-bottom: 1px solid var(--dbd2, #f3f4f6);
		display: flex;
		align-items: center;
		background: var(--dbg, #fff);
	}

	/* Task Row */
	.prm-tl-row {
		display: grid;
		grid-template-columns: repeat(14, 1fr);
		position: relative;
		min-height: 2.5rem;
		border-bottom: 1px solid var(--dbd2, #f3f4f6);
	}
	.prm-tl-cell { border-right: 1px solid var(--dbd2, #f3f4f6); }
	.prm-tl-cell--weekend { background: var(--dbg2, #f9fafb); }
	.prm-tl-cell--today { background: var(--dbg2, #f5f5f5); }

	/* Task Bars */
	.prm-tl-bar {
		position: absolute;
		top: 0.375rem;
		bottom: 0.375rem;
		border-radius: 4px;
		display: flex;
		align-items: center;
		padding: 0 0.5rem;
		z-index: 1;
		overflow: hidden;
	}
	.prm-tl-bar--done { opacity: 0.6; }
	.prm-tl-bar--upcoming { background: var(--dbg3, #e5e7eb) !important; border: 1px dashed var(--dbd, #d1d5db); }
	.prm-tl-bar-label { font-size: 0.625rem; font-weight: 600; color: var(--bos-surface-on-color, #fff); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
	.prm-tl-bar--upcoming .prm-tl-bar-label { color: var(--dt3, #6b7280); }
	.prm-tl-bar--done .prm-tl-bar-label { color: var(--dt3, #6b7280); }

	/* Today Marker */
	.prm-tl-today-marker { position: absolute; width: 2px; background: var(--bos-status-error, #ef4444); z-index: 2; pointer-events: none; }

	/* Empty State */
	.prm-tl-empty { display: flex; flex-direction: column; align-items: center; justify-content: center; padding: 3rem 1rem; text-align: center; }
	.prm-tl-empty-circle { width: 4rem; height: 4rem; border-radius: 50%; background: var(--dbg3, #f3f4f6); display: flex; align-items: center; justify-content: center; color: var(--dt4, #9ca3af); margin-bottom: 0.75rem; }
	.prm-tl-empty-title { font-size: 0.875rem; font-weight: 600; color: var(--dt, #111); margin-bottom: 0.25rem; }
	.prm-tl-empty-desc { font-size: 0.75rem; color: var(--dt3, #6b7280); max-width: 20rem; }

	/* Undated Tasks */
	.prm-tl-undated { padding: 0.75rem; background: var(--dbg, #fff); border: 1px solid var(--dbd, #e5e7eb); border-radius: 0.75rem; }
	.prm-tl-undated-title { font-size: 0.75rem; font-weight: 600; color: var(--dt2, #4b5563); margin-bottom: 0.5rem; }
	.prm-tl-undated-list { display: flex; flex-wrap: wrap; gap: 0.375rem; }
	.prm-tl-undated-item { display: flex; align-items: center; gap: 0.375rem; font-size: 0.6875rem; color: var(--dt2, #4b5563); padding: 0.25rem 0.5rem; background: var(--dbg2, #f9fafb); border-radius: 4px; }
	.prm-tl-undated-dot { width: 0.375rem; height: 0.375rem; border-radius: 50%; flex-shrink: 0; }
	.prm-tl-undated-dot--todo { background: var(--dt4, #9ca3af); }
	.prm-tl-undated-dot--in_progress { background: var(--bos-status-info, #3b82f6); }
	.prm-tl-undated-dot--done { background: var(--bos-status-success, #22c55e); }
	.prm-tl-undated-dot--cancelled { background: var(--bos-status-error, #ef4444); }
	.prm-tl-undated-label { white-space: nowrap; overflow: hidden; text-overflow: ellipsis; max-width: 12rem; }
	.prm-tl-undated-more { font-size: 0.6875rem; color: var(--dt4, #9ca3af); padding: 0.25rem 0.5rem; }
</style>
