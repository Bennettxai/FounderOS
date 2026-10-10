<!--
  DEPRECATED: TaskDetailSlideOver is no longer the primary task detail UI.
  Task detail is now a full page at /tasks/[id]/+page.svelte.
  This component is kept for any embed/modal contexts that may still need it,
  but the tasks list page now navigates to the full page route instead.
-->
<script lang="ts">
	import { DropdownMenu } from 'bits-ui';
	import TaskCheckbox from './TaskCheckbox.svelte';
	import PriorityBadge from './PriorityBadge.svelte';
	import { team } from '$lib/stores/team';
	import { onMount } from 'svelte';

	type TaskStatus = 'todo' | 'in_progress' | 'in_review' | 'done' | 'blocked';
	type Priority = 'critical' | 'high' | 'medium' | 'low';

	interface Assignee {
		id: string;
		name: string;
		avatar?: string;
	}

	// Load team members on mount
	onMount(() => {
		team.loadMembers();
	});

	interface Subtask {
		id: string;
		title: string;
		completed: boolean;
	}

	interface Comment {
		id: string;
		authorId: string;
		authorName: string;
		authorAvatar?: string;
		content: string;
		createdAt: string;
	}

	interface Activity {
		id: string;
		type: string;
		description: string;
		createdAt: string;
	}

	interface Task {
		id: string;
		title: string;
		description?: string;
		status: TaskStatus;
		priority: Priority;
		projectId?: string;
		projectName?: string;
		projectColor?: string;
		assignee?: Assignee;
		dueDate?: string;
		subtasks?: Subtask[];
		comments?: Comment[];
		activity?: Activity[];
		createdAt?: string;
	}

	interface Props {
		open?: boolean;
		task?: Task | null;
		onClose?: () => void;
		onStatusChange?: (status: TaskStatus) => void;
		onPriorityChange?: (priority: Priority) => void;
		onAssigneeChange?: (assigneeId: string | null) => void;
		onDueDateChange?: (dueDate: string | null) => void;
		onDescriptionChange?: (description: string) => void;
		onSubtaskToggle?: (subtaskId: string) => void;
		onSubtaskAdd?: (title: string) => void;
		onCommentAdd?: (content: string) => void;
	}

	let {
		open = $bindable(false),
		task = null,
		onClose,
		onStatusChange,
		onPriorityChange,
		onAssigneeChange,
		onDueDateChange,
		onDescriptionChange,
		onSubtaskToggle,
		onSubtaskAdd,
		onCommentAdd
	}: Props = $props();

	let editingDescription = $state(false);
	let descriptionDraft = $state('');
	let newSubtask = $state('');
	let newComment = $state('');

	const statusOptions: { value: TaskStatus; label: string; color: string }[] = [
		{ value: 'todo', label: 'To Do', color: '#6B7280' },
		{ value: 'in_progress', label: 'In Progress', color: '#3B82F6' },
		{ value: 'in_review', label: 'In Review', color: '#8B5CF6' },
		{ value: 'done', label: 'Done', color: '#10B981' },
		{ value: 'blocked', label: 'Blocked', color: '#F59E0B' }
	];

	const priorityOptions: { value: Priority; label: string; color: string }[] = [
		{ value: 'critical', label: 'Critical', color: '#EF4444' },
		{ value: 'high', label: 'High', color: '#F97316' },
		{ value: 'medium', label: 'Medium', color: '#EAB308' },
		{ value: 'low', label: 'Low', color: '#6B7280' }
	];

	function handleClose() {
		open = false;
		onClose?.();
	}

	function startEditDescription() {
		descriptionDraft = task?.description || '';
		editingDescription = true;
	}

	function saveDescription() {
		onDescriptionChange?.(descriptionDraft);
		editingDescription = false;
	}

	function handleAddSubtask() {
		if (newSubtask.trim()) {
			onSubtaskAdd?.(newSubtask);
			newSubtask = '';
		}
	}

	function handleAddComment() {
		if (newComment.trim()) {
			onCommentAdd?.(newComment);
			newComment = '';
		}
	}

	function formatRelativeTime(dateStr: string) {
		const date = new Date(dateStr);
		const now = new Date();
		const diff = now.getTime() - date.getTime();
		const minutes = Math.floor(diff / 60000);
		const hours = Math.floor(diff / 3600000);
		const days = Math.floor(diff / 86400000);

		if (minutes < 1) return 'Just now';
		if (minutes < 60) return `${minutes}m ago`;
		if (hours < 24) return `${hours}h ago`;
		if (days < 7) return `${days}d ago`;
		return date.toLocaleDateString();
	}
</script>

{#if open && task}
	<!-- Overlay -->
	<div
		class="fixed inset-0 bg-black/30 z-40 animate-in fade-in-0"
		onclick={handleClose}
	></div>

	<!-- Slide-over Panel -->
	<div
		class="fixed right-0 top-0 bottom-0 w-full max-w-md bg-white shadow-xl z-50 flex flex-col animate-in slide-in-from-right"
	>
		<!-- Header -->
		<div class="flex items-center justify-between px-6 py-4 border-b border-gray-100">
			<h2 class="text-lg font-semibold text-gray-900">Task Details</h2>
			<button
				onclick={handleClose}
				class="btn-pill btn-pill-ghost btn-pill-icon btn-pill-xs"
				aria-label="Close task details"
			>
				<svg class="w-5 h-5 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
				</svg>
			</button>
		</div>

		<!-- Content -->
		<div class="flex-1 overflow-y-auto">
			<div class="px-6 py-4 space-y-6">
				<!-- Title with Checkbox -->
				<div class="flex items-start gap-3">
					<TaskCheckbox
						status={task.status}
						size="lg"
						onStatusChange={onStatusChange}
					/>
					<h3 class="text-xl font-medium text-gray-900 {task.status === 'done' ? 'line-through text-gray-500' : ''}">
						{task.title}
					</h3>
				</div>

				<!-- Status & Priority -->
				<div class="grid grid-cols-2 gap-3">
					<div>
						<label class="block text-xs font-medium text-gray-500 uppercase mb-1.5">Status</label>
						<DropdownMenu.Root>
							<DropdownMenu.Trigger
								class="w-full flex items-center justify-between px-3 py-2 text-sm border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors"
							>
								<span class="flex items-center gap-2">
									<span class="w-2 h-2 rounded-full" style="background-color: {statusOptions.find(s => s.value === task.status)?.color}"></span>
									{statusOptions.find(s => s.value === task.status)?.label}
								</span>
								<svg class="w-4 h-4 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
								</svg>
							</DropdownMenu.Trigger>
							<DropdownMenu.Portal>
								<DropdownMenu.Content
									class="z-[60] min-w-[160px] bg-white border border-gray-200 rounded-xl shadow-lg p-1 animate-in fade-in-0 zoom-in-95"
									sideOffset={4}
								>
									{#each statusOptions as option}
										<DropdownMenu.Item
											class="flex items-center gap-2 px-3 py-2 text-sm text-gray-700 hover:bg-gray-100 rounded-lg cursor-pointer"
											onclick={() => onStatusChange?.(option.value)}
										>
											<span class="w-2 h-2 rounded-full" style="background-color: {option.color}"></span>
											{option.label}
										</DropdownMenu.Item>
									{/each}
								</DropdownMenu.Content>
							</DropdownMenu.Portal>
						</DropdownMenu.Root>
					</div>

					<div>
						<label class="block text-xs font-medium text-gray-500 uppercase mb-1.5">Priority</label>
						<DropdownMenu.Root>
							<DropdownMenu.Trigger
								class="w-full flex items-center justify-between px-3 py-2 text-sm border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors"
							>
								<span class="flex items-center gap-2">
									<span class="w-2 h-2 rounded-full" style="background-color: {priorityOptions.find(p => p.value === task.priority)?.color}"></span>
									{priorityOptions.find(p => p.value === task.priority)?.label}
								</span>
								<svg class="w-4 h-4 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
								</svg>
							</DropdownMenu.Trigger>
							<DropdownMenu.Portal>
								<DropdownMenu.Content
									class="z-[60] min-w-[140px] bg-white border border-gray-200 rounded-xl shadow-lg p-1 animate-in fade-in-0 zoom-in-95"
									sideOffset={4}
								>
									{#each priorityOptions as option}
										<DropdownMenu.Item
											class="flex items-center gap-2 px-3 py-2 text-sm text-gray-700 hover:bg-gray-100 rounded-lg cursor-pointer"
											onclick={() => onPriorityChange?.(option.value)}
										>
											<span class="w-2 h-2 rounded-full" style="background-color: {option.color}"></span>
											{option.label}
										</DropdownMenu.Item>
									{/each}
								</DropdownMenu.Content>
							</DropdownMenu.Portal>
						</DropdownMenu.Root>
					</div>
				</div>

				<!-- Assignee & Due Date -->
				<div class="grid grid-cols-2 gap-3">
					<div>
						<label class="block text-xs font-medium text-gray-500 uppercase mb-1.5">Assignee</label>
						<DropdownMenu.Root>
							<DropdownMenu.Trigger
								class="w-full flex items-center justify-between px-3 py-2 text-sm border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors"
								aria-label="Select assignee"
							>
								<span class="flex items-center gap-2">
									{#if task.assignee}
										<div class="w-6 h-6 rounded-full bg-gray-200 flex items-center justify-center text-xs font-medium">
											{task.assignee.name.charAt(0)}
										</div>
										<span>{task.assignee.name}</span>
									{:else}
										<svg class="w-4 h-4 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
										</svg>
										<span class="text-gray-400">Unassigned</span>
									{/if}
								</span>
								<svg class="w-4 h-4 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
								</svg>
							</DropdownMenu.Trigger>
							<DropdownMenu.Portal>
								<DropdownMenu.Content
									class="z-[60] min-w-[200px] bg-white border border-gray-200 rounded-xl shadow-lg p-1 animate-in fade-in-0 zoom-in-95"
									sideOffset={4}
								>
									<DropdownMenu.Item
										class="flex items-center gap-2 px-3 py-2 text-sm text-gray-700 hover:bg-gray-100 rounded-lg cursor-pointer"
										onclick={() => onAssigneeChange?.(null)}
									>
										<svg class="w-5 h-5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
										</svg>
										<span>Unassigned</span>
										{#if !task.assignee}
											<svg class="w-4 h-4 ml-auto text-gray-900" fill="none" stroke="currentColor" viewBox="0 0 24 24">
												<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
											</svg>
										{/if}
									</DropdownMenu.Item>
									{#if $team.members.length > 0}
										<DropdownMenu.Separator class="my-1 h-px bg-gray-200" />
										{#each $team.members as member}
											<DropdownMenu.Item
												class="flex items-center gap-2 px-3 py-2 text-sm text-gray-700 hover:bg-gray-100 rounded-lg cursor-pointer"
												onclick={() => onAssigneeChange?.(member.id)}
											>
												<div class="w-6 h-6 rounded-full bg-gray-200 flex items-center justify-center text-xs font-medium flex-shrink-0">
													{member.name?.charAt(0) || '?'}
												</div>
												<span class="truncate">{member.name}</span>
												{#if task.assignee?.id === member.id}
													<svg class="w-4 h-4 ml-auto text-gray-900 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
														<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
													</svg>
												{/if}
											</DropdownMenu.Item>
										{/each}
									{:else if $team.loading}
										<div class="px-3 py-2 text-sm text-gray-500">Loading team members...</div>
									{:else}
										<div class="px-3 py-2 text-sm text-gray-500">No team members found</div>
									{/if}
								</DropdownMenu.Content>
							</DropdownMenu.Portal>
						</DropdownMenu.Root>
					</div>

					<div>
						<label class="block text-xs font-medium text-gray-500 uppercase mb-1.5">Due Date</label>
						<input
							type="date"
							value={task.dueDate?.split('T')[0] || ''}
							onchange={(e) => onDueDateChange?.((e.target as HTMLInputElement).value)}
							class="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-gray-900 focus:border-transparent"
						/>
					</div>
				</div>

				<!-- Project -->
				{#if task.projectName}
					<div>
						<label class="block text-xs font-medium text-gray-500 uppercase mb-1.5">Project</label>
						<div class="flex items-center gap-2 px-3 py-2 bg-gray-50 rounded-lg">
							<span class="w-3 h-3 rounded" style="background-color: {task.projectColor || '#6B7280'}"></span>
							<span class="text-sm text-gray-700">{task.projectName}</span>
						</div>
					</div>
				{/if}

				<hr class="border-gray-200" />

				<!-- Description -->
				<div>
					<div class="flex items-center justify-between mb-2">
						<label class="text-sm font-medium text-gray-700">Description</label>
						{#if !editingDescription}
							<button onclick={startEditDescription} class="btn-pill btn-pill-ghost btn-pill-xs">
								Edit
							</button>
						{/if}
					</div>
					{#if editingDescription}
						<div>
							<textarea
								bind:value={descriptionDraft}
								rows={4}
								class="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-gray-900 focus:border-transparent resize-none"
								placeholder="Add a description..."
							></textarea>
							<div class="flex justify-end gap-2 mt-2">
								<button onclick={() => editingDescription = false} class="btn-pill btn-pill-soft btn-pill-xs">
									Cancel
								</button>
								<button onclick={saveDescription} class="btn-pill btn-pill-primary btn-pill-xs">
									Save
								</button>
							</div>
						</div>
					{:else}
						<p class="text-sm text-gray-600 bg-gray-50 rounded-lg px-3 py-2 min-h-[60px]">
							{task.description || 'No description'}
						</p>
					{/if}
				</div>

				<hr class="border-gray-200" />

				<!-- Subtasks -->
				<div>
					<div class="flex items-center justify-between mb-2">
						<label class="text-sm font-medium text-gray-700">Subtasks</label>
						<span class="text-xs text-gray-400">
							{task.subtasks?.filter(s => s.completed).length || 0}/{task.subtasks?.length || 0}
						</span>
					</div>
					<div class="space-y-1">
						{#each task.subtasks || [] as subtask}
							<label class="flex items-center gap-2 px-2 py-1.5 hover:bg-gray-50 rounded-lg cursor-pointer">
								<input
									type="checkbox"
									checked={subtask.completed}
									onchange={() => onSubtaskToggle?.(subtask.id)}
									class="rounded border-gray-300"
								/>
								<span class="text-sm {subtask.completed ? 'line-through text-gray-400' : 'text-gray-700'}">
									{subtask.title}
								</span>
							</label>
						{/each}
						<div class="flex items-center gap-2 px-2 py-1.5">
							<svg class="w-4 h-4 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
							</svg>
							<input
								type="text"
								bind:value={newSubtask}
								onkeydown={(e) => e.key === 'Enter' && handleAddSubtask()}
								placeholder="Add subtask..."
								class="flex-1 text-sm focus:outline-none"
							/>
						</div>
					</div>
				</div>

				<hr class="border-gray-200" />

				<!-- Comments -->
				<div>
					<label class="text-sm font-medium text-gray-700 mb-2 block">Comments</label>
					<div class="space-y-3">
						{#each task.comments || [] as comment}
							<div class="bg-gray-50 rounded-lg px-3 py-2">
								<div class="flex items-center gap-2 mb-1">
									<div class="w-6 h-6 rounded-full bg-gray-200 flex items-center justify-center text-xs font-medium">
										{comment.authorName.charAt(0)}
									</div>
									<span class="text-sm font-medium text-gray-700">{comment.authorName}</span>
									<span class="text-xs text-gray-400">{formatRelativeTime(comment.createdAt)}</span>
								</div>
								<p class="text-sm text-gray-600">{comment.content}</p>
							</div>
						{/each}
						<div class="flex items-start gap-2">
							<textarea
								bind:value={newComment}
								rows={2}
								placeholder="Add a comment..."
								class="flex-1 px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-gray-900 focus:border-transparent resize-none"
							></textarea>
							<button
								onclick={handleAddComment}
								disabled={!newComment.trim()}
								class="btn-pill btn-pill-primary btn-pill-sm disabled:opacity-50 disabled:cursor-not-allowed"
							>
								Post
							</button>
						</div>
					</div>
				</div>

				<hr class="border-gray-200" />

				<!-- Activity -->
				<div>
					<label class="text-sm font-medium text-gray-700 mb-2 block">Activity</label>
					<div class="space-y-2">
						{#each task.activity || [] as item}
							<div class="flex items-start gap-2 text-sm">
								<div class="w-1.5 h-1.5 rounded-full bg-gray-400 mt-2"></div>
								<div>
									<span class="text-gray-600">{item.description}</span>
									<span class="text-gray-400"> - {formatRelativeTime(item.createdAt)}</span>
								</div>
							</div>
						{/each}
						{#if !task.activity?.length}
							<p class="text-sm text-gray-400">No activity yet</p>
						{/if}
					</div>
				</div>
			</div>
		</div>
	</div>
{/if}
