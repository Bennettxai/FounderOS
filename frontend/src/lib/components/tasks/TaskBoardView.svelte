<script lang="ts">
	import { fade, fly } from 'svelte/transition';
	import { flip } from 'svelte/animate';
	import TaskBoardCard from './TaskBoardCard.svelte';

	type TaskStatus = 'todo' | 'in_progress' | 'in_review' | 'done' | 'blocked';
	type Priority = 'critical' | 'high' | 'medium' | 'low';

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
		subtaskCount?: number;
		subtaskCompleted?: number;
		commentCount?: number;
	}

	interface Props {
		tasks: Task[];
		onTaskClick?: (taskId: string) => void;
		onTaskStatusChange?: (taskId: string, status: TaskStatus) => void;
		onAddTask?: (status: TaskStatus) => void;
	}

	let { tasks, onTaskClick, onTaskStatusChange, onAddTask }: Props = $props();

	const columns: { status: TaskStatus; label: string; color: string }[] = [
		{ status: 'todo', label: 'To Do', color: 'var(--status-todo)' },
		{ status: 'in_progress', label: 'In Progress', color: 'var(--status-in-progress)' },
		{ status: 'in_review', label: 'In Review', color: 'var(--status-in-review)' },
		{ status: 'done', label: 'Done', color: 'var(--status-done)' },
		{ status: 'blocked', label: 'Blocked', color: 'var(--status-blocked)' }
	];

	let draggedTask = $state<Task | null>(null);
	let dragOverColumn = $state<TaskStatus | null>(null);

	const tasksByStatus = $derived((): Record<TaskStatus, Task[]> => {
		const grouped: Record<TaskStatus, Task[]> = {
			todo: [],
			in_progress: [],
			in_review: [],
			done: [],
			blocked: []
		};

		tasks.forEach(task => {
			grouped[task.status].push(task);
		});

		return grouped;
	});

	function handleDragStart(e: DragEvent, task: Task) {
		draggedTask = task;
		if (e.dataTransfer) {
			e.dataTransfer.effectAllowed = 'move';
			e.dataTransfer.setData('text/plain', task.id);
		}
	}

	function handleDragEnd() {
		draggedTask = null;
		dragOverColumn = null;
	}

	function handleDragOver(e: DragEvent, status: TaskStatus) {
		e.preventDefault();
		if (e.dataTransfer) {
			e.dataTransfer.dropEffect = 'move';
		}
		dragOverColumn = status;
	}

	function handleDragLeave() {
		dragOverColumn = null;
	}

	function handleDrop(e: DragEvent, status: TaskStatus) {
		e.preventDefault();
		if (draggedTask && draggedTask.status !== status) {
			onTaskStatusChange?.(draggedTask.id, status);
		}
		draggedTask = null;
		dragOverColumn = null;
	}
</script>

<style>
	/* #63 — Hover lift + drag ghost */
	.tbc-drag-wrapper {
		transition: transform 0.15s ease, box-shadow 0.15s ease, opacity 0.15s ease;
		border-radius: 0.625rem;
	}
	.tbc-drag-wrapper:hover {
		transform: translateY(-1px);
		box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
	}
	@media (prefers-reduced-motion: reduce) {
		.tbc-drag-wrapper {
			transition: opacity 0.15s ease;
		}
		.tbc-drag-wrapper:hover {
			transform: none;
		}
	}
	/* When actively dragging this specific card */
	.tbc-drag-ghost {
		opacity: 0.45;
		transform: rotate(2deg);
	}

	/* Board outer scroll container (#61) */
	.board-outer {
		flex: 1;
		overflow-x: auto;
		padding: 1.25rem 1.5rem;
	}

	/* Fluid column track (#61) */
	.board-track {
		display: flex;
		gap: 1rem;
		min-width: max-content;
		align-items: flex-start;
	}

	/* Column — fluid: flex:1, min 260px, max 340px (#61) */
	.board-column {
		flex: 1;
		min-width: 260px;
		max-width: 340px;
		display: flex;
		flex-direction: column;
		border-radius: 0.625rem;
		background: var(--dbg2, #f7f7f7);
		border: 1px solid var(--dbd, #e4e4e4);
		transition: background 0.15s ease-out, border-color 0.15s ease-out, box-shadow 0.15s ease-out;
	}

	/* Drag-over: visible indicator using design tokens (#61, #63) */
	.board-column--drag-over {
		background: color-mix(in srgb, var(--dt) 4%, var(--dbg2, #f7f7f7));
		border-color: color-mix(in srgb, var(--dt) 30%, transparent);
		box-shadow: inset 0 0 0 2px color-mix(in srgb, var(--dt) 12%, transparent);
	}

	/* Column header — bolder weight + bottom border (#61) */
	.board-column__header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 0.625rem 0.75rem;
		border-bottom: 1px solid var(--dbd, #e4e4e4);
	}
	.board-column__header-left {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}
	.board-column__dot {
		width: 0.5625rem;
		height: 0.5625rem;
		border-radius: 50%;
		flex-shrink: 0;
	}
	.board-column__label {
		font-size: 0.8125rem;
		font-weight: 600;
		color: var(--dt, #111);
	}
	.board-column__count {
		font-size: 0.6875rem;
		padding: 0.125rem 0.4375rem;
		border-radius: 999px;
		color: var(--dt3, #888);
		background: var(--dbg3, #ededed);
		font-weight: 500;
	}

	/* Column body — 12px gap between cards (#61) */
	.board-column__body {
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
		padding: 0.625rem;
		min-height: 200px;
	}

	/* Empty state — dashed border + icon + centered text (#61) */
	.board-column__empty {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 0.4375rem;
		min-height: 96px;
		border: 1.5px dashed var(--dbd2, #d9d9d9);
		border-radius: 0.5rem;
		color: var(--dt3, #aaa);
		font-size: 0.8125rem;
	}
	.board-column__empty-icon {
		width: 1.125rem;
		height: 1.125rem;
		opacity: 0.5;
	}
</style>

<div class="board-outer">
	<div class="board-track">
		{#each columns as column (column.status)}
			<div
				class="board-column {dragOverColumn === column.status ? 'board-column--drag-over' : ''}"
				ondragover={(e) => handleDragOver(e, column.status)}
				ondragleave={handleDragLeave}
				ondrop={(e) => handleDrop(e, column.status)}
			>
				<!-- Column Header: bolder font-weight, bottom border (#61) -->
				<div class="board-column__header">
					<div class="board-column__header-left">
						<span class="board-column__dot" style="background-color: {column.color};"></span>
						<span class="board-column__label">{column.label}</span>
						<span class="board-column__count">{tasksByStatus()[column.status].length}</span>
					</div>
					{#if column.status !== 'done'}
						<button
							onclick={() => onAddTask?.(column.status)}
							class="btn-pill btn-pill-ghost btn-pill-icon"
							aria-label="Add task to {column.label}"
						>
							<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
							</svg>
						</button>
					{/if}
				</div>

				<!-- Column Body: 12px gap between cards (#61) -->
				<div class="board-column__body">
					{#each tasksByStatus()[column.status] as task (task.id)}
						<div
							draggable="true"
							ondragstart={(e) => handleDragStart(e, task)}
							ondragend={handleDragEnd}
							class="tbc-drag-wrapper cursor-grab active:cursor-grabbing {draggedTask?.id === task.id ? 'tbc-drag-ghost' : ''}"
							animate:flip={{ duration: 200 }}
						>
							<TaskBoardCard
								{...task}
								onClick={() => onTaskClick?.(task.id)}
							/>
						</div>
					{/each}

					<!-- Empty column state: dashed outline + icon + "No tasks" (#61) -->
					{#if tasksByStatus()[column.status].length === 0 && dragOverColumn !== column.status}
						<div class="board-column__empty" in:fade={{ duration: 150 }}>
							<svg class="board-column__empty-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
							</svg>
							<span>No tasks</span>
						</div>
					{/if}
				</div>
			</div>
		{/each}
	</div>
</div>
