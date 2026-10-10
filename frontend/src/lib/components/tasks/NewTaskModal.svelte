<script lang="ts">
	import { Dialog } from 'bits-ui';
	import { DropdownMenu } from 'bits-ui';

	type TaskStatus = 'todo' | 'in_progress' | 'in_review' | 'done' | 'blocked';
	type Priority = 'critical' | 'high' | 'medium' | 'low';

	interface Project {
		id: string;
		name: string;
		color: string;
	}

	interface TeamMember {
		id: string;
		name: string;
		avatar?: string;
	}

	interface Props {
		open?: boolean;
		projects?: Project[];
		teamMembers?: TeamMember[];
		defaultStatus?: TaskStatus;
		defaultProjectId?: string;
		onClose?: () => void;
		onCreate?: (task: {
			title: string;
			description: string;
			projectId: string;
			status: TaskStatus;
			priority: Priority;
			assigneeId?: string;
			dueDate?: string;
			estimatedTime?: string;
			tags: string[];
		}) => void;
	}

	let {
		open = $bindable(false),
		projects = [],
		teamMembers = [],
		defaultStatus = 'todo',
		defaultProjectId,
		onClose,
		onCreate
	}: Props = $props();

	let title = $state('');
	let description = $state('');
	let projectId = $state(defaultProjectId || '');
	let status = $state<TaskStatus>(defaultStatus);
	let priority = $state<Priority>('medium');
	let assigneeId = $state<string>('');
	let dueDate = $state('');
	let estimatedTime = $state('');
	let tagInput = $state('');
	let tags = $state<string[]>([]);

	const statusOptions: { value: TaskStatus; label: string; color: string }[] = [
		{ value: 'todo', label: 'To Do', color: 'var(--dt3, #888)' },
		{ value: 'in_progress', label: 'In Progress', color: 'var(--dt2, #555)' },
		{ value: 'in_review', label: 'In Review', color: 'var(--dt2, #555)' },
		{ value: 'done', label: 'Done', color: 'var(--dt, #111)' },
		{ value: 'blocked', label: 'Blocked', color: 'var(--dt3, #888)' }
	];

	const priorityOptions: { value: Priority; label: string; color: string }[] = [
		{ value: 'critical', label: 'Critical', color: '#EF4444' },
		{ value: 'high', label: 'High', color: '#F97316' },
		{ value: 'medium', label: 'Medium', color: '#EAB308' },
		{ value: 'low', label: 'Low', color: '#9CA3AF' }
	];

	function handleSubmit() {
		if (!title.trim() || !projectId) return;

		onCreate?.({
			title,
			description,
			projectId,
			status,
			priority,
			assigneeId: assigneeId || undefined,
			dueDate: dueDate || undefined,
			estimatedTime: estimatedTime || undefined,
			tags
		});

		resetForm();
		open = false;
	}

	function resetForm() {
		title = '';
		description = '';
		projectId = defaultProjectId || '';
		status = defaultStatus;
		priority = 'medium';
		assigneeId = '';
		dueDate = '';
		estimatedTime = '';
		tagInput = '';
		tags = [];
	}

	function handleClose() {
		resetForm();
		open = false;
		onClose?.();
	}

	function addTag() {
		if (tagInput.trim() && !tags.includes(tagInput.trim())) {
			tags = [...tags, tagInput.trim()];
			tagInput = '';
		}
	}

	function removeTag(tag: string) {
		tags = tags.filter(t => t !== tag);
	}

	function handleTagKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter') {
			e.preventDefault();
			addTag();
		}
	}

	const selectedProject = $derived(projects.find(p => p.id === projectId));
	const selectedAssignee = $derived(teamMembers.find(m => m.id === assigneeId));
</script>

<Dialog.Root bind:open>
	<Dialog.Portal>
		<Dialog.Overlay
			class="fixed inset-0 bg-black/50 z-50 animate-in fade-in-0"
		/>
		<Dialog.Content
			class="fixed left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 z-50 w-full max-w-lg animate-in fade-in-0 zoom-in-95"
			style="background: var(--bos-modal-bg); border: 1px solid var(--bos-modal-border); border-radius: var(--bos-modal-radius); box-shadow: var(--bos-modal-shadow);"
		>
			<!-- Header -->
			<div
				class="flex items-center justify-between px-6 py-4 border-b"
				style="border-color: var(--bos-modal-border);"
			>
				<Dialog.Title
					class="text-lg font-semibold"
					style="color: var(--bos-text-primary-color);"
				>New Task</Dialog.Title>
				<Dialog.Close
					class="btn-pill btn-pill-ghost btn-pill-icon"
					onclick={handleClose}
				>
					<svg
						class="w-5 h-5"
						style="color: var(--bos-text-tertiary-color);"
						fill="none"
						stroke="currentColor"
						viewBox="0 0 24 24"
					>
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
					</svg>
				</Dialog.Close>
			</div>

			<!-- Body -->
			<div class="px-6 py-4 space-y-4 max-h-[60vh] overflow-y-auto">
				<!-- Task Name -->
				<div>
					<label for="task-name" class="bos-label bos-label--required">
						Task name
					</label>
					<input
						id="task-name"
						type="text"
						bind:value={title}
						placeholder="What needs to be done?"
						class="bos-input"
					/>
				</div>

				<!-- Description -->
				<div>
					<label for="task-desc" class="bos-label">
						Description
					</label>
					<textarea
						id="task-desc"
						bind:value={description}
						placeholder="Add details, notes, or context..."
						rows={3}
						class="bos-input resize-none"
					></textarea>
				</div>

				<!-- Project & Status Row -->
				<div class="grid grid-cols-2 gap-3">
					<!-- Project -->
					<div>
						<label class="bos-label bos-label--required">
							Project
						</label>
						<DropdownMenu.Root>
							<DropdownMenu.Trigger
								class="btn-pill btn-pill-secondary w-full flex items-center justify-between text-left"
							>
								{#if selectedProject}
									<span class="flex items-center gap-2">
										<span class="w-2 h-2 rounded-full" style="background-color: {selectedProject.color}"></span>
										{selectedProject.name}
									</span>
								{:else}
									<span style="color: var(--bos-text-tertiary-color);">Select project</span>
								{/if}
								<svg
									class="w-4 h-4"
									style="color: var(--bos-text-tertiary-color);"
									fill="none"
									stroke="currentColor"
									viewBox="0 0 24 24"
								>
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
								</svg>
							</DropdownMenu.Trigger>
							<DropdownMenu.Portal>
								<DropdownMenu.Content
									class="bos-dropdown z-[60] min-w-[200px] animate-in fade-in-0 zoom-in-95"
									sideOffset={4}
								>
									{#each projects as project}
										<DropdownMenu.Item
											class="bos-dropdown-item"
											onclick={() => projectId = project.id}
										>
											<span class="w-2 h-2 rounded-full" style="background-color: {project.color}"></span>
											{project.name}
										</DropdownMenu.Item>
									{/each}
									{#if projects.length === 0}
										<p
											class="px-3 py-2 text-sm"
											style="color: var(--bos-text-tertiary-color);"
										>No projects available</p>
									{/if}
								</DropdownMenu.Content>
							</DropdownMenu.Portal>
						</DropdownMenu.Root>
					</div>

					<!-- Status -->
					<div>
						<label class="bos-label">Status</label>
						<DropdownMenu.Root>
							<DropdownMenu.Trigger
								class="btn-pill btn-pill-secondary w-full flex items-center justify-between text-left"
							>
								<span class="flex items-center gap-2">
									<span class="w-2 h-2 rounded-full" style="background-color: {statusOptions.find(s => s.value === status)?.color}"></span>
									{statusOptions.find(s => s.value === status)?.label}
								</span>
								<svg
									class="w-4 h-4"
									style="color: var(--bos-text-tertiary-color);"
									fill="none"
									stroke="currentColor"
									viewBox="0 0 24 24"
								>
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
								</svg>
							</DropdownMenu.Trigger>
							<DropdownMenu.Portal>
								<DropdownMenu.Content
									class="bos-dropdown z-[60] min-w-[160px] animate-in fade-in-0 zoom-in-95"
									sideOffset={4}
								>
									{#each statusOptions as option}
										<DropdownMenu.Item
											class="bos-dropdown-item"
											onclick={() => status = option.value}
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

				<!-- Priority & Assignee Row -->
				<div class="grid grid-cols-2 gap-3">
					<!-- Priority -->
					<div>
						<label class="bos-label">Priority</label>
						<DropdownMenu.Root>
							<DropdownMenu.Trigger
								class="btn-pill btn-pill-secondary w-full flex items-center justify-between text-left"
							>
								<span class="flex items-center gap-2">
									<span class="w-2 h-2 rounded-full" style="background-color: {priorityOptions.find(p => p.value === priority)?.color}"></span>
									{priorityOptions.find(p => p.value === priority)?.label}
								</span>
								<svg
									class="w-4 h-4"
									style="color: var(--bos-text-tertiary-color);"
									fill="none"
									stroke="currentColor"
									viewBox="0 0 24 24"
								>
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
								</svg>
							</DropdownMenu.Trigger>
							<DropdownMenu.Portal>
								<DropdownMenu.Content
									class="bos-dropdown z-[60] min-w-[140px] animate-in fade-in-0 zoom-in-95"
									sideOffset={4}
								>
									{#each priorityOptions as option}
										<DropdownMenu.Item
											class="bos-dropdown-item"
											onclick={() => priority = option.value}
										>
											<span class="w-2 h-2 rounded-full" style="background-color: {option.color}"></span>
											{option.label}
										</DropdownMenu.Item>
									{/each}
								</DropdownMenu.Content>
							</DropdownMenu.Portal>
						</DropdownMenu.Root>
					</div>

					<!-- Assignee -->
					<div>
						<label class="bos-label">Assignee</label>
						<DropdownMenu.Root>
							<DropdownMenu.Trigger
								class="btn-pill btn-pill-secondary w-full flex items-center justify-between text-left"
							>
								{#if selectedAssignee}
									<span class="flex items-center gap-2">
										<div
											class="w-5 h-5 rounded-full flex items-center justify-center text-xs"
											style="background: var(--bos-hover-color); color: var(--bos-text-primary-color);"
										>
											{selectedAssignee.name.charAt(0)}
										</div>
										{selectedAssignee.name}
									</span>
								{:else}
									<span style="color: var(--bos-text-tertiary-color);">Unassigned</span>
								{/if}
								<svg
									class="w-4 h-4"
									style="color: var(--bos-text-tertiary-color);"
									fill="none"
									stroke="currentColor"
									viewBox="0 0 24 24"
								>
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
								</svg>
							</DropdownMenu.Trigger>
							<DropdownMenu.Portal>
								<DropdownMenu.Content
									class="bos-dropdown z-[60] min-w-[180px] animate-in fade-in-0 zoom-in-95"
									sideOffset={4}
								>
									<DropdownMenu.Item
										class="bos-dropdown-item"
										style="color: var(--bos-text-secondary-color);"
										onclick={() => assigneeId = ''}
									>
										Unassigned
									</DropdownMenu.Item>
									{#each teamMembers as member}
										<DropdownMenu.Item
											class="bos-dropdown-item"
											onclick={() => assigneeId = member.id}
										>
											<div
												class="w-5 h-5 rounded-full flex items-center justify-center text-xs"
												style="background: var(--bos-hover-color); color: var(--bos-text-primary-color);"
											>
												{member.name.charAt(0)}
											</div>
											{member.name}
										</DropdownMenu.Item>
									{/each}
								</DropdownMenu.Content>
							</DropdownMenu.Portal>
						</DropdownMenu.Root>
					</div>
				</div>

				<!-- Due Date & Estimated Time Row -->
				<div class="grid grid-cols-2 gap-3">
					<!-- Due Date -->
					<div>
						<label for="due-date" class="bos-label">Due date</label>
						<input
							id="due-date"
							type="date"
							bind:value={dueDate}
							class="bos-input"
						/>
					</div>

					<!-- Estimated Time -->
					<div>
						<label for="est-time" class="bos-label">Estimated time</label>
						<input
							id="est-time"
							type="text"
							bind:value={estimatedTime}
							placeholder="e.g., 2h, 1d"
							class="bos-input"
						/>
					</div>
				</div>

				<!-- Tags -->
				<div>
					<label class="bos-label">Tags</label>
					<div
						class="flex flex-wrap gap-2 p-2 border rounded-xl min-h-[44px]"
						style="border-color: var(--bos-border-color);"
					>
						{#each tags as tag}
							<span
								class="flex items-center gap-1 px-2 py-1 text-sm rounded-lg"
								style="background: var(--bos-hover-color); color: var(--bos-text-primary-color);"
							>
								{tag}
								<button
									onclick={() => removeTag(tag)}
									style="color: var(--bos-text-secondary-color);"
									class="hover:opacity-75 transition-opacity"
								>
									<svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
									</svg>
								</button>
							</span>
						{/each}
						<input
							type="text"
							bind:value={tagInput}
							onkeydown={handleTagKeydown}
							placeholder={tags.length === 0 ? '+ Add tags...' : ''}
							class="flex-1 min-w-[100px] px-2 py-1 text-sm focus:outline-none bg-transparent"
							style="color: var(--bos-text-primary-color);"
						/>
					</div>
				</div>
			</div>

			<!-- Footer -->
			<div
				class="flex items-center justify-end gap-3 px-6 py-4 border-t"
				style="border-color: var(--bos-modal-border);"
			>
				<button
					onclick={handleClose}
					class="btn-pill btn-pill-ghost btn-pill-sm"
				>
					Cancel
				</button>
				<button
					onclick={handleSubmit}
					disabled={!title.trim() || !projectId}
					class="btn-pill btn-pill-primary btn-pill-sm"
				>
					Create Task
				</button>
			</div>
		</Dialog.Content>
	</Dialog.Portal>
</Dialog.Root>
