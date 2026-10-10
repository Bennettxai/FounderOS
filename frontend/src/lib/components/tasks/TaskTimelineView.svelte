<script lang="ts">
	type TaskStatus = 'todo' | 'in_progress' | 'in_review' | 'done' | 'blocked';
	type Priority = 'critical' | 'high' | 'medium' | 'low';

	interface Task {
		id: string;
		title: string;
		status: TaskStatus;
		priority: Priority;
		dueDate?: string;
	}

	interface Props {
		tasks: Task[];
		onTaskClick?: (taskId: string) => void;
	}

	let { tasks, onTaskClick }: Props = $props();

	let windowOffset = $state(0);

	const STATUS_COLORS: Record<TaskStatus, string> = {
		in_progress: '#4A90E2',
		done: '#10B981',
		todo: '#888888',
		in_review: '#A855F7',
		blocked: '#FF6B35'
	};

	const STATUS_LABELS: Record<TaskStatus, string> = {
		todo: 'To Do',
		in_progress: 'In Progress',
		in_review: 'In Review',
		done: 'Done',
		blocked: 'Blocked'
	};

	const PRIORITY_LABELS: Record<Priority, string> = {
		critical: 'Critical',
		high: 'High',
		medium: 'Medium',
		low: 'Low'
	};

	function getBaseDate(): Date {
		const today = new Date();
		today.setHours(0, 0, 0, 0);
		return today;
	}

	function getDayColumns(offset: number): Date[] {
		const base = getBaseDate();
		const startDay = new Date(base);
		startDay.setDate(startDay.getDate() - 3 + offset);
		const days: Date[] = [];
		for (let i = 0; i < 14; i++) {
			const d = new Date(startDay);
			d.setDate(d.getDate() + i);
			days.push(d);
		}
		return days;
	}

	const days = $derived(getDayColumns(windowOffset));

	const today = $derived(() => {
		const t = new Date();
		t.setHours(0, 0, 0, 0);
		return t;
	});

	function isSameDay(a: Date, b: Date): boolean {
		return a.getFullYear() === b.getFullYear() &&
			a.getMonth() === b.getMonth() &&
			a.getDate() === b.getDate();
	}

	function isWeekend(date: Date): boolean {
		const day = date.getDay();
		return day === 0 || day === 6;
	}

	function formatDayHeader(date: Date): { weekday: string; dateNum: number } {
		const weekdays = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];
		return {
			weekday: weekdays[date.getDay()],
			dateNum: date.getDate()
		};
	}

	function parseDueDate(dueDate: string): Date {
		const d = new Date(dueDate);
		d.setHours(0, 0, 0, 0);
		return d;
	}

	function getDayIndex(date: Date): number {
		const startDay = days[0];
		const diff = Math.round((date.getTime() - startDay.getTime()) / (1000 * 60 * 60 * 24));
		return diff;
	}

	const datedTasks = $derived(
		tasks.filter((t) => t.dueDate).map((t) => ({
			...t,
			parsedDate: parseDueDate(t.dueDate!)
		}))
	);

	const visibleTasks = $derived(
		datedTasks.filter((t) => {
			const idx = getDayIndex(t.parsedDate);
			return idx >= 0 && idx < 14;
		})
	);

	const undatedTasks = $derived(tasks.filter((t) => !t.dueDate));

	const monthYearLabel = $derived(() => {
		if (days.length === 0) return '';
		const first = days[0];
		const last = days[days.length - 1];
		const opts: Intl.DateTimeFormatOptions = { month: 'long', year: 'numeric' };
		if (first.getMonth() === last.getMonth()) {
			return first.toLocaleDateString('en-US', opts);
		}
		const fMonth = first.toLocaleDateString('en-US', { month: 'short' });
		const lFull = last.toLocaleDateString('en-US', opts);
		return `${fMonth} - ${lFull}`;
	});

	const todayIndex = $derived(() => {
		const t = getBaseDate();
		return getDayIndex(t);
	});

	function navigatePrev() {
		windowOffset -= 7;
	}

	function navigateNext() {
		windowOffset += 7;
	}

	function goToToday() {
		windowOffset = 0;
	}

	const hasTasks = $derived(visibleTasks.length > 0 || undatedTasks.length > 0);
</script>

<div class="tb-timeline-container glass-card">
	<!-- Header -->
	<div class="tb-timeline-header">
		<div class="tb-timeline-nav">
			<button class="tb-timeline-nav-btn" onclick={navigatePrev} aria-label="Previous week">
				<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
					<path d="M15 19l-7-7 7-7" />
				</svg>
			</button>
			<button class="tb-timeline-today-btn" onclick={goToToday}>Today</button>
			<span class="tb-timeline-label">{monthYearLabel()}</span>
			<button class="tb-timeline-nav-btn" onclick={navigateNext} aria-label="Next week">
				<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
					<path d="M9 5l7 7-7 7" />
				</svg>
			</button>
		</div>

		<!-- Legend -->
		<div class="tb-timeline-legend">
			{#each Object.entries(STATUS_LABELS) as [status, label]}
				<div class="tb-timeline-legend-item">
					<span class="tb-timeline-legend-dot" style="background: {STATUS_COLORS[status as TaskStatus]}"></span>
					<span class="tb-timeline-legend-label">{label}</span>
				</div>
			{/each}
		</div>
	</div>

	{#if !hasTasks && tasks.length === 0}
		<!-- Empty state -->
		<div class="tb-timeline-empty">
			<div class="tb-timeline-empty-icon">
				<svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
					<rect x="3" y="4" width="18" height="18" rx="2" ry="2" />
					<line x1="16" y1="2" x2="16" y2="6" />
					<line x1="8" y1="2" x2="8" y2="6" />
					<line x1="3" y1="10" x2="21" y2="10" />
				</svg>
			</div>
			<h3 class="tb-timeline-empty-title">No timeline data</h3>
			<p class="tb-timeline-empty-desc">Add due dates to tasks to see them on the timeline</p>
		</div>
	{:else}
		<!-- Timeline grid -->
		<div class="tb-timeline-grid glass-surface">
			<!-- Day headers row -->
			<div class="tb-timeline-day-headers">
				<div class="tb-timeline-task-label-header">Task</div>
				{#each days as day, i}
					{@const header = formatDayHeader(day)}
					{@const isT = isSameDay(day, today())}
					<div
						class="tb-timeline-day-header"
						class:tb-timeline-weekend={isWeekend(day)}
						class:tb-timeline-today-header={isT}
					>
						<span class="tb-timeline-day-weekday">{header.weekday}</span>
						<span class="tb-timeline-day-num" class:tb-timeline-today-num={isT}>{header.dateNum}</span>
					</div>
				{/each}
			</div>

			<!-- Today marker overlay -->
			{#if todayIndex() >= 0 && todayIndex() < 14}
				<div
					class="tb-timeline-today-marker"
					style="left: calc(140px + ({todayIndex()} * ((100% - 140px) / 14)) + ((100% - 140px) / 28))"
				></div>
			{/if}

			<!-- Task rows -->
			<div class="tb-timeline-rows">
				{#each visibleTasks as task (task.id)}
					{@const colIdx = getDayIndex(task.parsedDate)}
					<div class="tb-timeline-row">
						<button
							class="tb-timeline-task-label"
							onclick={() => onTaskClick?.(task.id)}
							title="{task.title} ({PRIORITY_LABELS[task.priority]})"
						>
							<span class="tb-timeline-task-title">{task.title}</span>
						</button>
						<div class="tb-timeline-task-cells">
							{#each days as _, i}
								<div
									class="tb-timeline-cell"
									class:tb-timeline-weekend={isWeekend(days[i])}
								>
									{#if i === colIdx}
										<button
											class="tb-timeline-bar"
											style="background: {STATUS_COLORS[task.status]}"
											onclick={() => onTaskClick?.(task.id)}
											title="{task.title} - {STATUS_LABELS[task.status]}"
										></button>
									{/if}
								</div>
							{/each}
						</div>
					</div>
				{/each}

				{#if visibleTasks.length === 0 && undatedTasks.length === 0}
					<div class="tb-timeline-no-visible">
						<p>No tasks in this date range. Use the navigation arrows to explore other dates.</p>
					</div>
				{/if}
			</div>
		</div>

		<!-- Undated section -->
		{#if undatedTasks.length > 0}
			<div class="tb-timeline-undated">
				<h4 class="tb-timeline-undated-title">Undated Tasks</h4>
				<div class="tb-timeline-undated-list">
					{#each undatedTasks as task (task.id)}
						<button
							class="tb-timeline-undated-item"
							onclick={() => onTaskClick?.(task.id)}
						>
							<span class="tb-timeline-undated-dot" style="background: {STATUS_COLORS[task.status]}"></span>
							<span class="tb-timeline-undated-label">{task.title}</span>
							<span class="tb-timeline-undated-status">{STATUS_LABELS[task.status]}</span>
						</button>
					{/each}
				</div>
			</div>
		{/if}
	{/if}
</div>

<style>
	.tb-timeline-container {
		display: flex;
		flex-direction: column;
		gap: var(--space-4, 16px);
		padding: var(--space-5, 20px);
		border-radius: var(--radius-md, 12px);
		overflow: hidden;
	}

	/* Header */
	.tb-timeline-header {
		display: flex;
		flex-direction: column;
		gap: var(--space-3, 12px);
	}

	.tb-timeline-nav {
		display: flex;
		align-items: center;
		gap: var(--space-3, 12px);
	}

	.tb-timeline-nav-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 32px;
		height: 32px;
		border: 1px solid var(--dbd, #e5e7eb);
		border-radius: var(--radius-sm, 8px);
		background: var(--dbg, #ffffff);
		color: var(--dt2, #6b7280);
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.tb-timeline-nav-btn:hover {
		background: var(--dbg2, #f3f4f6);
		color: var(--dt, #111827);
		border-color: var(--dbd2, #d1d5db);
	}

	.tb-timeline-today-btn {
		padding: var(--space-1, 4px) var(--space-3, 12px);
		font-size: var(--text-sm, 14px);
		font-weight: var(--font-medium, 500);
		border: 1px solid var(--dbd, #e5e7eb);
		border-radius: var(--radius-sm, 8px);
		background: var(--dbg, #ffffff);
		color: var(--dt2, #6b7280);
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.tb-timeline-today-btn:hover {
		background: var(--dbg2, #f3f4f6);
		color: var(--dt, #111827);
	}

	.tb-timeline-label {
		font-size: var(--text-base, 16px);
		font-weight: var(--font-semibold, 600);
		color: var(--dt, #111827);
		min-width: 180px;
	}

	/* Legend */
	.tb-timeline-legend {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-4, 16px);
	}

	.tb-timeline-legend-item {
		display: flex;
		align-items: center;
		gap: var(--space-1, 4px);
	}

	.tb-timeline-legend-dot {
		width: 8px;
		height: 8px;
		border-radius: 50%;
		flex-shrink: 0;
	}

	.tb-timeline-legend-label {
		font-size: var(--text-xs, 12px);
		color: var(--dt3, #888888);
	}

	/* Grid */
	.tb-timeline-grid {
		position: relative;
		border-radius: var(--radius-sm, 8px);
		overflow: hidden;
		border: 1px solid var(--dbd, #e5e7eb);
	}

	/* Day headers */
	.tb-timeline-day-headers {
		display: flex;
		border-bottom: 1px solid var(--dbd, #e5e7eb);
	}

	.tb-timeline-task-label-header {
		width: 140px;
		min-width: 140px;
		padding: var(--space-2, 8px) var(--space-3, 12px);
		font-size: var(--text-xs, 12px);
		font-weight: var(--font-semibold, 600);
		color: var(--dt3, #888888);
		text-transform: uppercase;
		letter-spacing: 0.05em;
		border-right: 1px solid var(--dbd, #e5e7eb);
	}

	.tb-timeline-day-header {
		flex: 1;
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 2px;
		padding: var(--space-2, 8px) var(--space-1, 4px);
		border-right: 1px solid var(--dbd, #e5e7eb);
	}

	.tb-timeline-day-header:last-child {
		border-right: none;
	}

	.tb-timeline-day-weekday {
		font-size: var(--text-xs, 12px);
		font-weight: var(--font-medium, 500);
		color: var(--dt3, #888888);
		text-transform: uppercase;
	}

	.tb-timeline-day-num {
		font-size: var(--text-sm, 14px);
		font-weight: var(--font-semibold, 600);
		color: var(--dt2, #6b7280);
		width: 28px;
		height: 28px;
		display: flex;
		align-items: center;
		justify-content: center;
		border-radius: 50%;
	}

	.tb-timeline-today-num {
		background: #ef4444;
		color: #ffffff;
	}

	.tb-timeline-today-header {
		background: rgba(239, 68, 68, 0.05);
	}

	.tb-timeline-weekend {
		background: var(--dbg2, #f3f4f6);
	}

	/* Today marker */
	.tb-timeline-today-marker {
		position: absolute;
		top: 0;
		bottom: 0;
		width: 2px;
		background: #ef4444;
		z-index: 10;
		pointer-events: none;
		opacity: 0.7;
	}

	/* Task rows */
	.tb-timeline-rows {
		display: flex;
		flex-direction: column;
	}

	.tb-timeline-row {
		display: flex;
		border-bottom: 1px solid var(--dbd, #e5e7eb);
		min-height: 44px;
	}

	.tb-timeline-row:last-child {
		border-bottom: none;
	}

	.tb-timeline-task-label {
		width: 140px;
		min-width: 140px;
		padding: var(--space-2, 8px) var(--space-3, 12px);
		display: flex;
		align-items: center;
		border-right: 1px solid var(--dbd, #e5e7eb);
		background: transparent;
		border-top: none;
		border-bottom: none;
		border-left: none;
		cursor: pointer;
		text-align: left;
		transition: background 0.15s ease;
	}

	.tb-timeline-task-label:hover {
		background: var(--dbg2, #f3f4f6);
	}

	.tb-timeline-task-title {
		font-size: var(--text-sm, 14px);
		font-weight: var(--font-medium, 500);
		color: var(--dt, #111827);
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		max-width: 100%;
	}

	.tb-timeline-task-cells {
		flex: 1;
		display: flex;
	}

	.tb-timeline-cell {
		flex: 1;
		display: flex;
		align-items: center;
		justify-content: center;
		padding: var(--space-1, 4px) 2px;
		border-right: 1px solid var(--dbd, #e5e7eb);
	}

	.tb-timeline-cell:last-child {
		border-right: none;
	}

	.tb-timeline-bar {
		width: 100%;
		height: 24px;
		border-radius: 4px;
		border: none;
		cursor: pointer;
		transition: opacity 0.15s ease, transform 0.15s ease;
		box-shadow: var(--shadow-sm, 0 1px 2px rgba(0, 0, 0, 0.05));
	}

	.tb-timeline-bar:hover {
		opacity: 0.85;
		transform: scaleY(1.15);
	}

	.tb-timeline-no-visible {
		padding: var(--space-6, 24px);
		text-align: center;
	}

	.tb-timeline-no-visible p {
		font-size: var(--text-sm, 14px);
		color: var(--dt3, #888888);
	}

	/* Undated section */
	.tb-timeline-undated {
		border-top: 1px solid var(--dbd, #e5e7eb);
		padding-top: var(--space-4, 16px);
	}

	.tb-timeline-undated-title {
		font-size: var(--text-sm, 14px);
		font-weight: var(--font-semibold, 600);
		color: var(--dt2, #6b7280);
		margin-bottom: var(--space-3, 12px);
	}

	.tb-timeline-undated-list {
		display: flex;
		flex-direction: column;
		gap: var(--space-1, 4px);
	}

	.tb-timeline-undated-item {
		display: flex;
		align-items: center;
		gap: var(--space-2, 8px);
		padding: var(--space-2, 8px) var(--space-3, 12px);
		border-radius: var(--radius-sm, 8px);
		background: transparent;
		border: none;
		cursor: pointer;
		width: 100%;
		text-align: left;
		transition: background 0.15s ease;
	}

	.tb-timeline-undated-item:hover {
		background: var(--dbg2, #f3f4f6);
	}

	.tb-timeline-undated-dot {
		width: 8px;
		height: 8px;
		border-radius: 50%;
		flex-shrink: 0;
	}

	.tb-timeline-undated-label {
		flex: 1;
		font-size: var(--text-sm, 14px);
		font-weight: var(--font-medium, 500);
		color: var(--dt, #111827);
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.tb-timeline-undated-status {
		font-size: var(--text-xs, 12px);
		color: var(--dt3, #888888);
		flex-shrink: 0;
	}

	/* Empty state */
	.tb-timeline-empty {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: var(--space-6, 24px) var(--space-4, 16px);
		gap: var(--space-3, 12px);
		min-height: 240px;
	}

	.tb-timeline-empty-icon {
		width: 72px;
		height: 72px;
		border-radius: 50%;
		background: var(--dbg2, #f3f4f6);
		display: flex;
		align-items: center;
		justify-content: center;
		color: var(--dt3, #888888);
	}

	.tb-timeline-empty-title {
		font-size: var(--text-base, 16px);
		font-weight: var(--font-semibold, 600);
		color: var(--dt, #111827);
		margin: 0;
	}

	.tb-timeline-empty-desc {
		font-size: var(--text-sm, 14px);
		color: var(--dt3, #888888);
		margin: 0;
		text-align: center;
	}

	/* Dark mode */
	:global(.dark) .tb-timeline-nav-btn {
		background: var(--dbg2, #1f2937);
		border-color: var(--dbd, #374151);
		color: var(--dt2, #9ca3af);
	}

	:global(.dark) .tb-timeline-nav-btn:hover {
		background: var(--dbg3, #374151);
		color: var(--dt, #f9fafb);
	}

	:global(.dark) .tb-timeline-today-btn {
		background: var(--dbg2, #1f2937);
		border-color: var(--dbd, #374151);
		color: var(--dt2, #9ca3af);
	}

	:global(.dark) .tb-timeline-today-btn:hover {
		background: var(--dbg3, #374151);
		color: var(--dt, #f9fafb);
	}

	:global(.dark) .tb-timeline-weekend {
		background: var(--dbg2, #1f2937);
	}

	:global(.dark) .tb-timeline-task-label:hover {
		background: var(--dbg3, #374151);
	}

	:global(.dark) .tb-timeline-undated-item:hover {
		background: var(--dbg3, #374151);
	}

	:global(.dark) .tb-timeline-empty-icon {
		background: var(--dbg3, #374151);
		color: var(--dt3, #6b7280);
	}

	:global(.dark) .tb-timeline-today-header {
		background: rgba(239, 68, 68, 0.1);
	}
</style>
