<script lang="ts">
	import type { Task, TeamMemberListResponse } from '$lib/api';
	import { api } from '$lib/api';
	import { formatDate, getPriorityDotVar } from '$lib/utils/project';

	interface Props {
		tasks: Task[];
		teamMembers: TeamMemberListResponse[];
		embedSuffix: string;
		onShowAddTask: () => void;
		onEditTask: (task: Task) => void;
		onTasksChanged: () => Promise<void>;
	}

	let { tasks, teamMembers, embedSuffix, onShowAddTask, onEditTask, onTasksChanged }: Props = $props();

	let completedTasks = $derived(tasks.filter((t) => t.status === 'done').length);
	let totalTasks = $derived(tasks.length);

	// Drag & Drop state
	let draggedTask = $state<Task | null>(null);
	let dragOverTask = $state<Task | null>(null);

	async function handleToggleTask(taskId: string) {
		try {
			await api.toggleTask(taskId);
			await onTasksChanged();
		} catch (err) {
			console.error('Error toggling task:', err);
		}
	}

	async function handleDeleteTask(taskId: string) {
		try {
			await api.deleteTask(taskId);
			await onTasksChanged();
		} catch (err) {
			console.error('Error deleting task:', err);
		}
	}

	function handleDragStart(e: DragEvent, task: Task) {
		draggedTask = task;
		if (e.dataTransfer) {
			e.dataTransfer.effectAllowed = 'move';
		}
	}

	function handleDragOver(e: DragEvent, task: Task) {
		e.preventDefault();
		if (e.dataTransfer) {
			e.dataTransfer.dropEffect = 'move';
		}
		if (draggedTask && draggedTask.id !== task.id && draggedTask.status === task.status) {
			dragOverTask = task;
		}
	}

	function handleDragLeave() {
		dragOverTask = null;
	}

	async function handleDrop(e: DragEvent, targetTask: Task) {
		e.preventDefault();
		dragOverTask = null;

		const currentDraggedTask = draggedTask;
		if (
			!currentDraggedTask ||
			currentDraggedTask.id === targetTask.id ||
			currentDraggedTask.status !== targetTask.status
		) {
			return;
		}

		try {
			const statusTasks = tasks.filter((t) => t.status === targetTask.status);
			const draggedIndex = statusTasks.findIndex((t) => t.id === currentDraggedTask.id);
			const targetIndex = statusTasks.findIndex((t) => t.id === targetTask.id);

			if (draggedIndex === -1 || targetIndex === -1) return;

			const newStatusTasks = [...statusTasks];
			const [removed] = newStatusTasks.splice(draggedIndex, 1);
			newStatusTasks.splice(targetIndex, 0, removed);

			const updatePromises = newStatusTasks.map((task, index) =>
				api.updateTask(task.id, { position: index })
			);

			await Promise.all(updatePromises);
			await onTasksChanged();
		} catch (err) {
			console.error('Error reordering tasks:', err);
			await onTasksChanged();
		}
	}

	function handleDragEnd() {
		draggedTask = null;
		dragOverTask = null;
	}
</script>

<div class="space-y-4">
	<!-- Task Stats Cards -->
	<div class="grid grid-cols-4 gap-3">
		<div class="prm-tk-card">
			<p class="prm-tk-stat-label">To Do</p>
			<p class="prm-tk-stat-value">{tasks.filter((t) => t.status === 'todo').length}</p>
		</div>
		<div class="prm-tk-card">
			<p class="prm-tk-stat-label">In Progress</p>
			<p class="prm-tk-stat-value prm-tk-stat-value--blue mt-1">{tasks.filter((t) => t.status === 'in_progress').length}</p>
		</div>
		<div class="prm-tk-card">
			<p class="prm-tk-stat-label">Done</p>
			<p class="prm-tk-stat-value prm-tk-stat-value--green mt-1">{tasks.filter((t) => t.status === 'done').length}</p>
		</div>
		<div class="prm-tk-card">
			<p class="prm-tk-stat-label">Completion</p>
			<p class="prm-tk-stat-value prm-tk-stat-value--purple mt-1">
				{totalTasks > 0 ? Math.round((completedTasks / totalTasks) * 100) : 0}%
			</p>
		</div>
	</div>

	<!-- Tasks List -->
	<div class="prm-tk-list">
		<div class="p-4 prm-tk-list__header flex items-center justify-between">
			<div class="flex items-center gap-2">
				<h2 class="prm-tk-list__title">All Tasks</h2>
				<span class="prm-tk-meta">({totalTasks})</span>
			</div>
			<div class="flex items-center gap-2">
				<a href="/tasks{embedSuffix}" class="btn-pill btn-pill-secondary btn-pill-sm">
					<svg class="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
					</svg>
					Open Tasks
				</a>
				<button onclick={onShowAddTask} class="btn-pill btn-pill-primary btn-pill-sm">
					<svg class="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
					</svg>
					Add Task
				</button>
			</div>
		</div>

		{#if tasks.length === 0}
			<div class="text-center py-16">
				<div class="w-16 h-16 rounded-full prm-tk-empty-circle flex items-center justify-center mx-auto mb-4">
					<svg class="w-8 h-8 prm-tk-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
					</svg>
				</div>
				<h3 class="prm-tk-list__title mb-1">No tasks yet</h3>
				<p class="prm-tk-muted mb-4">Break down your project into manageable tasks</p>
				<button onclick={onShowAddTask} class="btn-pill btn-pill-primary">Add First Task</button>
			</div>
		{:else}
			{#each [
				{ status: 'todo', label: 'To Do', dotClass: 'prm-tk-dot--gray', items: tasks.filter((t) => t.status === 'todo') },
				{ status: 'in_progress', label: 'In Progress', dotClass: 'prm-tk-dot--blue', items: tasks.filter((t) => t.status === 'in_progress') },
				{ status: 'done', label: 'Done', dotClass: 'prm-tk-dot--green', items: tasks.filter((t) => t.status === 'done') }
			].filter((g) => g.items.length > 0) as group}
				<div class="prm-tk-group">
					<div class="px-4 py-2 prm-tk-group__header flex items-center gap-2">
						<span class="prm-tk-dot {group.dotClass}"></span>
						<span class="prm-tk-group__label">{group.label}</span>
						<span class="prm-tk-meta">({group.items.length})</span>
					</div>
					<div class="prm-tk-group__items">
						{#each group.items as task}
							{@const isSubtask = !!task.parent_task_id}
							{@const assignee = task.assignee_id ? teamMembers.find((m) => m.id === task.assignee_id) : null}
							<div
								class="flex items-center gap-4 p-4 prm-tk-row group/task transition-all {dragOverTask?.id === task.id ? 'prm-tk-row--dragover' : ''} {draggedTask?.id === task.id ? 'opacity-50' : 'opacity-100'} {isSubtask ? 'pl-12 prm-tk-row--subtask' : ''}"
								draggable="true"
								ondragstart={(e) => handleDragStart(e, task)}
								ondragover={(e) => handleDragOver(e, task)}
								ondragleave={handleDragLeave}
								ondrop={(e) => handleDrop(e, task)}
								ondragend={handleDragEnd}
							>
								<!-- Subtask Indicator -->
								{#if isSubtask}
									<div class="absolute left-6 w-4 h-4 flex items-center justify-center">
										<svg class="w-3 h-3 prm-tk-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
										</svg>
									</div>
								{/if}

								<!-- Drag Handle -->
								<div class="flex-shrink-0 cursor-move prm-tk-drag opacity-0 group-hover/task:opacity-100 transition-opacity">
									<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 8h16M4 16h16" />
									</svg>
								</div>

								<button
									onclick={() => handleToggleTask(task.id)}
									class="w-6 h-6 rounded-full border-2 flex items-center justify-center flex-shrink-0 transition-colors {task.status === 'done' ? 'prm-tk-checkbox--done' : 'prm-tk-checkbox'}"
									aria-label="Toggle task complete"
								>
									{#if task.status === 'done'}
										<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="3" d="M5 13l4 4L19 7" />
										</svg>
									{/if}
								</button>

								<div class="flex-1 min-w-0">
									<p class="font-medium {task.status === 'done' ? 'prm-tk-done' : 'prm-tk-text'}">{task.title}</p>
									{#if task.description}
										<p class="text-sm prm-tk-muted mt-0.5 line-clamp-1">{task.description}</p>
									{/if}
									<div class="flex items-center gap-3 mt-1 text-xs prm-tk-meta">
										{#if task.start_date}
											<span class="flex items-center gap-1">
												<svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
													<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
												</svg>
												Start: {formatDate(task.start_date)}
											</span>
										{/if}
										{#if task.due_date}
											<span class="flex items-center gap-1 {new Date(task.due_date) < new Date() && task.status !== 'done' ? 'prm-tk-overdue' : ''}">
												<svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
													<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
												</svg>
												Due: {formatDate(task.due_date)}
											</span>
										{/if}
										{#if task.estimated_hours}
											<span class="flex items-center gap-1">
												<svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
													<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
												</svg>
												{task.estimated_hours}h
											</span>
										{/if}
									</div>
								</div>

								<div class="flex items-center gap-2">
									<span class="prm-tk-priority-label"><span class="prm-tk-priority-dot" style="background: {getPriorityDotVar(task.priority)}"></span>{task.priority}</span>
									{#if assignee}
											<span class="text-xs px-2 py-1 rounded prm-tk-assignee font-medium flex items-center gap-1">
											<svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
												<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
											</svg>
											{assignee.name}
										</span>
									{/if}
								</div>

								<div class="flex items-center gap-1 opacity-0 group-hover/task:opacity-100 transition-opacity">
									<button
										onclick={() => onEditTask(task)}
										class="btn-pill btn-pill-ghost btn-pill-icon"
										aria-label="Edit task"
									>
										<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
										</svg>
									</button>
									<button
										onclick={() => handleDeleteTask(task.id)}
										class="btn-pill btn-pill-danger btn-pill-icon"
										aria-label="Delete task"
									>
										<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
										</svg>
									</button>
								</div>
							</div>
						{/each}
					</div>
				</div>
			{/each}
		{/if}
	</div>
</div>

<style>
	.prm-tk-card {
		background: var(--dbg, #fff);
		border: 1px solid var(--dbd, #e0e0e0);
		border-radius: 0.75rem;
		padding: 1rem;
	}
	.prm-tk-stat-label {
		font-size: 0.75rem;
		font-weight: 500;
		color: var(--dt2, #555);
		text-transform: uppercase;
		letter-spacing: 0.05em;
	}
	.prm-tk-stat-value {
		font-size: 1.5rem;
		font-weight: 700;
		color: var(--dt, #111);
		margin-top: 0.25rem;
	}
	.prm-tk-list {
		background: var(--dbg, #fff);
		border: 1px solid var(--dbd, #e0e0e0);
		border-radius: 0.75rem;
	}
	.prm-tk-list__header {
		border-bottom: 1px solid var(--dbd2, #f0f0f0);
	}
	.prm-tk-list__title {
		font-size: 1.125rem;
		font-weight: 500;
		color: var(--dt, #111);
	}
	.prm-tk-text {
		color: var(--dt, #111);
	}
	.prm-tk-muted {
		color: var(--dt2, #555);
	}
	.prm-tk-meta {
		color: var(--dt3, #888);
	}
	.prm-tk-done {
		color: var(--dt3, #888);
		text-decoration: line-through;
	}
	.prm-tk-icon {
		color: var(--dt3, #888);
	}
	.prm-tk-drag {
		color: var(--dt3, #888);
	}
	.prm-tk-drag:hover {
		color: var(--dt2, #555);
	}
	.prm-tk-checkbox {
		border-color: var(--dbd, #ccc);
	}
	.prm-tk-checkbox:hover {
		border-color: var(--dt, #111);
	}
	.prm-tk-empty-circle {
		background: var(--dbg2, #f5f5f5);
	}
	.prm-tk-group {
		border-bottom: 1px solid var(--dbd2, #f0f0f0);
	}
	.prm-tk-group:last-child {
		border-bottom: none;
	}
	.prm-tk-group__header {
		background: var(--dbg2, rgba(245,245,245,0.5));
	}
	.prm-tk-group__label {
		font-size: 0.75rem;
		font-weight: 500;
		color: var(--dt2, #555);
	}
	.prm-tk-group__items {
		border-top: 1px solid var(--dbd2, #f0f0f0);
	}
	.prm-tk-row:hover {
		background: var(--dbg2, #f5f5f5);
	}
	.prm-tk-row + .prm-tk-row {
		border-top: 1px solid var(--dbd2, #f0f0f0);
	}
	.prm-tk-row--subtask {
		background: var(--dbg2, rgba(245,245,245,0.5));
	}
	.prm-tk-stat-value--blue { color: var(--dt, #111); }
	.prm-tk-stat-value--green { color: var(--dt, #111); }
	.prm-tk-stat-value--purple { color: var(--dt, #111); }
	.prm-tk-dot { width: 0.5rem; height: 0.5rem; border-radius: 50%; flex-shrink: 0; }
	.prm-tk-priority-label { display: inline-flex; align-items: center; gap: 0.375rem; font-size: 0.75rem; font-weight: 500; color: var(--dt2, #555); text-transform: capitalize; }
	.prm-tk-priority-dot { width: 6px; height: 6px; border-radius: 50%; flex-shrink: 0; }
	.prm-tk-dot--gray { background: var(--dt3, #888); }
	.prm-tk-dot--blue { background: var(--dt2, #555); }
	.prm-tk-dot--green { background: var(--dt3, #888); }
	.prm-tk-row--dragover { border-top: 2px solid var(--dt, #111); }
	.prm-tk-checkbox--done { background: var(--dt, #111); border-color: var(--dt, #111); color: var(--bos-surface-on-color, #fff); }
	.prm-tk-overdue { color: var(--bos-status-error, #ef4444); }
	.prm-tk-assignee { background: var(--dbg3, #eee); color: var(--dt, #111); }
</style>
