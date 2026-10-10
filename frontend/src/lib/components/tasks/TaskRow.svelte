<script lang="ts">
	import { DropdownMenu } from 'bits-ui';
	import { goto } from '$app/navigation';
	import TaskCheckbox from './TaskCheckbox.svelte';

	type TaskStatus = 'todo' | 'in_progress' | 'in_review' | 'done' | 'blocked';
	type Priority = 'critical' | 'high' | 'medium' | 'low';

	interface Assignee {
		id: string;
		name: string;
		avatar?: string;
	}

	interface Props {
		id: string;
		title: string;
		status: TaskStatus;
		priority: Priority;
		projectId?: string;
		projectName?: string;
		projectColor?: string;
		assignee?: Assignee;
		dueDate?: string;
		tags?: string[];
		onClick?: () => void;
		onStatusChange?: (status: TaskStatus) => void;
		onEdit?: () => void;
		onDuplicate?: () => void;
		onDelete?: () => void;
		onAssign?: () => void;
		onSetDueDate?: () => void;
	}

	let {
		id,
		title,
		status,
		priority,
		projectId,
		projectName,
		projectColor = '#6B7280',
		assignee,
		dueDate,
		tags = [],
		onClick,
		onStatusChange,
		onEdit,
		onDuplicate,
		onDelete,
		onAssign,
		onSetDueDate
	}: Props = $props();

	function navigateToProject(e: MouseEvent) {
		if (projectId) {
			e.stopPropagation();
			goto(`/projects/${projectId}`);
		}
	}

	let menuOpen = $state(false);

	function formatDueDate(dateStr: string) {
		const date = new Date(dateStr);
		const now = new Date();
		const diff = date.getTime() - now.getTime();
		const days = Math.ceil(diff / (1000 * 60 * 60 * 24));

		if (days < 0) return { text: `${Math.abs(days)}d overdue`, isOverdue: true };
		if (days === 0) return { text: 'Due today', isOverdue: false };
		if (days === 1) return { text: 'Due tomorrow', isOverdue: false };
		if (days < 7) return { text: `Due in ${days}d`, isOverdue: false };
		return { text: date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' }), isOverdue: false };
	}

	const dueDateInfo = $derived(dueDate ? formatDueDate(dueDate) : null);
	const isDone = $derived(status === 'done');
	const isBlocked = $derived(status === 'blocked');

	const priorityDotColor = $derived(() => {
		switch (priority) {
			case 'critical': return '#EF4444';
			case 'high': return '#F97316';
			case 'medium': return '#EAB308';
			case 'low': return '#6B7280';
		}
	});

	function getTagColor(_tag: string): string {
		return 'var(--dt3, #888)';
	}
</script>

<div
	class="group relative flex items-center px-4 py-2.5 tb-row transition-all duration-150
		{isBlocked ? 'tb-row--blocked' : ''}
		{isDone ? 'opacity-50' : ''}"
>
	<!-- Checkbox + Priority dot + Title -->
	<div class="flex items-center gap-3 flex-1 min-w-0">
		<TaskCheckbox {status} onStatusChange={onStatusChange} />

		<!-- Priority dot -->
		<span
			class="tb-row-priority-dot"
			style="background-color: {priorityDotColor()}"
			title="{priority} priority"
		></span>

		<!-- Title -->
		<button
			onclick={onClick}
			class="flex-1 min-w-0 text-left"
			aria-label="View task: {title}"
		>
			<span class="text-sm tb-row-title truncate block {isDone ? 'line-through tb-row-title--done' : ''}">
				{title}
			</span>
		</button>
	</div>

	<!-- Deadline column -->
	<div class="tb-col-deadline">
		{#if dueDateInfo}
			<span class="tb-row-date {dueDateInfo.isOverdue ? 'tb-row-overdue' : ''}">
				{dueDateInfo.text}
			</span>
		{:else}
			<span class="tb-row-date-empty">—</span>
		{/if}
	</div>

	<!-- Project column -->
	<div class="tb-col-project">
		{#if projectName}
			<button
				onclick={navigateToProject}
				class="tb-row-project-pill"
				aria-label="Go to project: {projectName}"
			>
				<span class="tb-row-project-dot" style="background-color: {projectColor}"></span>
				<span class="truncate">{projectName}</span>
			</button>
		{:else}
			<span class="tb-row-date-empty">—</span>
		{/if}
	</div>

	<!-- Labels column -->
	<div class="tb-col-labels">
		<div class="flex items-center gap-1 flex-wrap">
			{#each (tags || []).slice(0, 2) as tag}
				<span class="tb-row-label-pill">
					{tag}
				</span>
			{/each}
			{#if (tags || []).length > 2}
				<span class="tb-row-label-overflow">+{(tags || []).length - 2}</span>
			{/if}
		</div>
	</div>

	<!-- Menu -->
	<div class="tb-col-actions opacity-0 group-hover:opacity-100 transition-opacity {menuOpen ? '!opacity-100' : ''}">
		<DropdownMenu.Root bind:open={menuOpen}>
			<DropdownMenu.Trigger
				class="tb-row-menu-trigger"
				onclick={(e: MouseEvent) => e.stopPropagation()}
				aria-label="Task actions"
			>
				<svg class="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
					<path d="M10 6a2 2 0 110-4 2 2 0 010 4zM10 12a2 2 0 110-4 2 2 0 010 4zM10 18a2 2 0 110-4 2 2 0 010 4z" />
				</svg>
			</DropdownMenu.Trigger>
			<DropdownMenu.Portal>
				<DropdownMenu.Content
					class="z-50 min-w-[180px] tb-row-menu rounded-xl shadow-lg p-1 animate-in fade-in-0 zoom-in-95"
					sideOffset={4}
				>
					<DropdownMenu.Item
						class="flex items-center gap-3 px-3 py-2 text-sm tb-row-menu-item rounded-lg cursor-pointer transition-colors"
						onclick={onClick}
					>
						<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
						</svg>
						View Details
					</DropdownMenu.Item>
					<DropdownMenu.Item
						class="flex items-center gap-3 px-3 py-2 text-sm tb-row-menu-item rounded-lg cursor-pointer transition-colors"
						onclick={onEdit}
					>
						<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
						</svg>
						Edit
					</DropdownMenu.Item>
					<DropdownMenu.Item
						class="flex items-center gap-3 px-3 py-2 text-sm tb-row-menu-item rounded-lg cursor-pointer transition-colors"
						onclick={onDuplicate}
					>
						<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
						</svg>
						Duplicate
					</DropdownMenu.Item>
					{#if projectId && projectName}
						<DropdownMenu.Item
							class="flex items-center gap-3 px-3 py-2 text-sm tb-row-menu-item rounded-lg cursor-pointer transition-colors"
							onclick={() => goto(`/projects/${projectId}`)}
						>
							<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
							</svg>
							Go to Project
						</DropdownMenu.Item>
					{/if}

					<DropdownMenu.Separator class="h-px tb-row-menu-sep my-1" />

					<DropdownMenu.Item
						class="flex items-center gap-3 px-3 py-2 text-sm tb-row-menu-item rounded-lg cursor-pointer transition-colors"
						onclick={() => onStatusChange?.('done')}
					>
						<svg class="w-4 h-4" style="color: var(--dt2, #555)" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
						</svg>
						Mark Done
					</DropdownMenu.Item>

					<DropdownMenu.Separator class="h-px tb-row-menu-sep my-1" />

					<DropdownMenu.Item
						class="flex items-center gap-3 px-3 py-2 text-sm tb-row-menu-danger rounded-lg cursor-pointer transition-colors"
						onclick={onDelete}
					>
						<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
						</svg>
						Delete
					</DropdownMenu.Item>
				</DropdownMenu.Content>
			</DropdownMenu.Portal>
		</DropdownMenu.Root>
	</div>
</div>

<style>
	.tb-row {
		border-bottom: 1px solid var(--dbd2, #f0f0f0);
	}
	.tb-row:hover {
		background: var(--dbg2, #f5f5f5);
	}
	.tb-row--blocked {
		background: color-mix(in srgb, var(--color-warning, #f59e0b) 5%, transparent);
	}

	/* Priority dot */
	.tb-row-priority-dot {
		width: 7px;
		height: 7px;
		border-radius: 50%;
		flex-shrink: 0;
	}

	/* Title */
	.tb-row-title {
		color: var(--dt, #111);
		font-weight: 500;
	}
	.tb-row-title--done {
		color: var(--dt3, #888);
		font-weight: 400;
	}

	/* Column widths — must match TaskListView headers */
	.tb-col-deadline {
		width: 110px;
		flex-shrink: 0;
	}
	.tb-col-project {
		width: 130px;
		flex-shrink: 0;
	}
	.tb-col-labels {
		width: 160px;
		flex-shrink: 0;
	}
	.tb-col-actions {
		width: 36px;
		flex-shrink: 0;
		display: flex;
		justify-content: center;
	}

	/* Deadline */
	.tb-row-date {
		font-size: 0.8125rem;
		color: var(--dt2, #555);
	}
	.tb-row-overdue {
		color: var(--color-error, #ef4444);
		font-weight: 500;
	}
	.tb-row-date-empty {
		font-size: 0.8125rem;
		color: var(--dt4, #bbb);
	}

	/* Project pill */
	.tb-row-project-pill {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		max-width: 100%;
		font-size: 0.8125rem;
		color: var(--dt2, #555);
		transition: color 150ms ease;
	}
	.tb-row-project-pill:hover {
		color: var(--dt, #111);
	}
	.tb-row-project-dot {
		width: 8px;
		height: 8px;
		border-radius: 3px;
		flex-shrink: 0;
	}

	/* Label pills */
	.tb-row-label-pill {
		display: inline-flex;
		align-items: center;
		padding: 1px 8px;
		border-radius: 9999px;
		font-size: 0.6875rem;
		font-weight: 500;
		white-space: nowrap;
		background: var(--dbg3, #eee);
		color: var(--dt2, #555);
	}
	.tb-row-label-overflow {
		font-size: 0.6875rem;
		color: var(--dt3, #888);
		font-weight: 500;
	}

	/* Menu trigger */
	.tb-row-menu-trigger {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 28px;
		height: 28px;
		border-radius: 6px;
		color: var(--dt3, #888);
		transition: all 150ms ease;
	}
	.tb-row-menu-trigger:hover {
		background: var(--dbg3, #eee);
		color: var(--dt, #111);
	}

	/* Menu dropdown */
	.tb-row-menu {
		background: var(--dbg, #fff);
		border: 1px solid var(--dbd, #e0e0e0);
	}
	.tb-row-menu-item {
		color: var(--dt, #111);
	}
	.tb-row-menu-item:hover {
		background: var(--dbg2, #f5f5f5);
	}
	.tb-row-menu-sep {
		background: var(--dbd2, #f0f0f0);
	}
	.tb-row-menu-danger {
		color: var(--color-error, #ef4444);
	}
	.tb-row-menu-danger:hover {
		background: color-mix(in srgb, var(--color-error, #ef4444) 5%, transparent);
	}
</style>
