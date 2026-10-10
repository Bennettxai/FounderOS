<script lang="ts">
	import { fade, slide } from 'svelte/transition';
	import TaskRow from './TaskRow.svelte';
	import TaskGroupHeader from './TaskGroupHeader.svelte';
	import TaskInlineAdd from './TaskInlineAdd.svelte';

	type TaskStatus = 'todo' | 'in_progress' | 'in_review' | 'done' | 'blocked';
	type Priority = 'critical' | 'high' | 'medium' | 'low';
	type GroupBy = 'status' | 'priority' | 'project' | 'assignee' | 'none';

	interface Assignee {
		id: string;
		name: string;
		avatar?: string;
	}

	interface Task {
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
		// Allow TaskRow to receive projectId for navigation
	}

	interface Props {
		tasks: Task[];
		groupBy?: GroupBy;
		showDoneTasks?: boolean;
		onTaskClick?: (taskId: string) => void;
		onTaskStatusChange?: (taskId: string, status: TaskStatus) => void;
		onTaskEdit?: (taskId: string) => void;
		onTaskDuplicate?: (taskId: string) => void;
		onTaskDelete?: (taskId: string) => void;
		onAddTask?: (task: { title: string; priority: Priority; status?: string }) => void;
	}

	let {
		tasks,
		groupBy = 'status',
		showDoneTasks = true,
		onTaskClick,
		onTaskStatusChange,
		onTaskEdit,
		onTaskDuplicate,
		onTaskDelete,
		onAddTask
	}: Props = $props();

	let collapsedGroups = $state<Set<string>>(new Set());
	let showInlineAdd = $state<string | null>(null);

	const statusConfig: Record<TaskStatus, { label: string; color: string; order: number }> = {
		todo: { label: 'To Do', color: 'var(--status-todo)', order: 0 },
		in_progress: { label: 'In Progress', color: 'var(--status-in-progress)', order: 1 },
		in_review: { label: 'In Review', color: 'var(--status-in-review)', order: 2 },
		done: { label: 'Done', color: 'var(--status-done)', order: 3 },
		blocked: { label: 'Blocked', color: 'var(--status-blocked)', order: 4 }
	};

	const priorityConfig: Record<Priority, { label: string; color: string; order: number }> = {
		critical: { label: 'Critical', color: 'var(--priority-critical)', order: 0 },
		high: { label: 'High', color: 'var(--priority-high)', order: 1 },
		medium: { label: 'Medium', color: 'var(--priority-medium)', order: 2 },
		low: { label: 'Low', color: 'var(--priority-low)', order: 3 }
	};

	interface GroupedTasks {
		key: string;
		label: string;
		color: string;
		tasks: Task[];
		order: number;
	}

	const groupedTasks = $derived((): GroupedTasks[] => {
		let filtered = showDoneTasks ? tasks : tasks.filter(t => t.status !== 'done');

		if (groupBy === 'none') {
			return [{ key: 'all', label: 'All Tasks', color: 'var(--status-todo)', tasks: filtered, order: 0 }];
		}

		const groups: Map<string, GroupedTasks> = new Map();

		filtered.forEach(task => {
			let key: string;
			let label: string;
			let color: string;
			let order: number;

			switch (groupBy) {
				case 'status':
					key = task.status;
					label = statusConfig[task.status].label;
					color = statusConfig[task.status].color;
					order = statusConfig[task.status].order;
					break;
				case 'priority':
					key = task.priority;
					label = priorityConfig[task.priority].label;
					color = priorityConfig[task.priority].color;
					order = priorityConfig[task.priority].order;
					break;
				case 'project':
					key = task.projectId || 'no-project';
					label = task.projectName || 'No Project';
					color = task.projectColor || 'var(--status-todo)';
					order = task.projectName ? 0 : 999;
					break;
				case 'assignee':
					key = task.assignee?.id || 'unassigned';
					label = task.assignee?.name || 'Unassigned';
					color = 'var(--status-todo)';
					order = task.assignee ? 0 : 999;
					break;
				default:
					key = 'all';
					label = 'All Tasks';
					color = 'var(--status-todo)';
					order = 0;
			}

			if (!groups.has(key)) {
				groups.set(key, { key, label, color, tasks: [], order });
			}
			groups.get(key)!.tasks.push(task);
		});

		// Sort groups by order
		return Array.from(groups.values()).sort((a, b) => a.order - b.order);
	});

	function toggleGroup(key: string) {
		if (collapsedGroups.has(key)) {
			collapsedGroups.delete(key);
		} else {
			collapsedGroups.add(key);
		}
		collapsedGroups = new Set(collapsedGroups);
	}

	function handleAddTask(groupKey: string, task: { title: string; priority: Priority }) {
		onAddTask?.({ ...task, status: groupBy === 'status' ? groupKey : undefined });
		showInlineAdd = null;
	}
</script>

<div class="flex-1 overflow-y-auto">
	<!-- Column Headers -->
	{#if tasks.length > 0}
		<div class="tb-col-header flex items-center px-4 py-2">
			<div class="flex-1 min-w-0">
				<span class="tb-col-label">Task</span>
			</div>
			<div class="tb-col-deadline">
				<span class="tb-col-label">Deadline</span>
			</div>
			<div class="tb-col-project">
				<span class="tb-col-label">Project</span>
			</div>
			<div class="tb-col-labels">
				<span class="tb-col-label">Labels</span>
			</div>
			<div class="tb-col-actions"></div>
		</div>
	{/if}

	{#each groupedTasks() as group (group.key)}
		<div in:fade={{ duration: 150 }}>
			{#if groupBy !== 'none'}
				<TaskGroupHeader
					title={group.label}
					count={group.tasks.length}
					color={group.color}
					collapsed={collapsedGroups.has(group.key)}
					showAddButton={group.key !== 'done'}
					onToggle={() => toggleGroup(group.key)}
					onAdd={() => showInlineAdd = group.key}
				/>
			{/if}

			{#if !collapsedGroups.has(group.key)}
				<div transition:slide={{ duration: 200 }}>
					{#each group.tasks as task (task.id)}
						<TaskRow
							{...task}
							onClick={() => onTaskClick?.(task.id)}
							onStatusChange={(status) => onTaskStatusChange?.(task.id, status)}
							onEdit={() => onTaskEdit?.(task.id)}
							onDuplicate={() => onTaskDuplicate?.(task.id)}
							onDelete={() => onTaskDelete?.(task.id)}
						/>
					{/each}

					{#if showInlineAdd === group.key}
						<TaskInlineAdd
							status={groupBy === 'status' ? group.key : undefined}
							onAdd={(task) => handleAddTask(group.key, task)}
							onCancel={() => showInlineAdd = null}
						/>
					{/if}
				</div>
			{/if}
		</div>
	{/each}

	{#if tasks.length === 0}
		<div class="flex flex-col items-center justify-center py-16" in:fade={{ duration: 200 }}>
			{#if showInlineAdd !== 'all'}
				<div class="w-16 h-16 rounded-full tb-empty-icon flex items-center justify-center mb-4">
					<svg class="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4" />
					</svg>
				</div>
				<h3 class="text-lg font-medium tb-empty-title mb-1">No tasks yet</h3>
				<p class="tb-empty-text mb-4">Create your first task to start tracking work</p>
				<button
					onclick={() => showInlineAdd = 'all'}
					class="btn-cta"
				>
					<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
					</svg>
					Create your first task
				</button>
			{:else}
				<div class="w-full max-w-2xl">
					<TaskInlineAdd
						onAdd={(task) => handleAddTask('all', task)}
						onCancel={() => showInlineAdd = null}
					/>
				</div>
			{/if}
		</div>
	{/if}
</div>

<style>
	.tb-col-header {
		background: var(--dbg, #fff);
		border-bottom: 1px solid var(--dbd, #e0e0e0);
		position: sticky;
		top: 0;
		z-index: 20;
	}
	.tb-col-label {
		font-size: 0.6875rem;
		font-weight: 500;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		color: var(--dt3, #888);
	}
	.tb-col-deadline {
		width: 110px;
		flex-shrink: 0;
		text-align: left;
	}
	.tb-col-project {
		width: 130px;
		flex-shrink: 0;
		text-align: left;
	}
	.tb-col-labels {
		width: 160px;
		flex-shrink: 0;
		text-align: left;
	}
	.tb-col-actions {
		width: 36px;
		flex-shrink: 0;
	}
	.tb-empty-icon {
		background: var(--dbg2, #f5f5f5);
		color: var(--dt3, #888);
	}
	.tb-empty-title {
		color: var(--dt, #111);
	}
	.tb-empty-text {
		color: var(--dt2, #555);
	}
</style>
