<script lang="ts">
	import { DropdownMenu } from 'bits-ui';
	import { team } from '$lib/stores/team';
	import { onMount } from 'svelte';

	type Priority = 'critical' | 'high' | 'medium' | 'low';

	interface Props {
		projectId?: string;
		projectName?: string;
		status?: string;
		onAdd?: (task: { title: string; priority: Priority; assigneeId?: string; dueDate?: string }) => void;
		onCancel?: () => void;
	}

	let { projectId, projectName, status, onAdd, onCancel }: Props = $props();

	// Load team members on mount
	onMount(() => {
		team.loadMembers();
	});

	let title = $state('');
	let priority: Priority = $state('medium');
	let assigneeId = $state<string | undefined>(undefined);
	let dueDate = $state<string | undefined>(undefined);
	let isExpanded = $state(false);
	let inputRef: HTMLInputElement | undefined = $state(undefined);

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter' && title.trim()) {
			e.preventDefault();
			handleSubmit();
		} else if (e.key === 'Escape') {
			handleCancel();
		}
	}

	function handleSubmit() {
		if (!title.trim()) return;
		onAdd?.({ title, priority, assigneeId, dueDate });
		title = '';
		priority = 'medium';
		assigneeId = undefined;
		dueDate = undefined;
		isExpanded = false;
	}

	function handleCancel() {
		title = '';
		isExpanded = false;
		onCancel?.();
	}

	function handleFocus() {
		isExpanded = true;
	}

	const priorityDotColors: Record<Priority, string> = {
		critical: '#ef4444',
		high: '#f97316',
		medium: '#eab308',
		low: '#9ca3af'
	};

	const priorityOptions: { value: Priority; label: string }[] = [
		{ value: 'critical', label: 'Critical' },
		{ value: 'high', label: 'High' },
		{ value: 'medium', label: 'Medium' },
		{ value: 'low', label: 'Low' }
	];
</script>

<div class="px-4 py-2 animate-in fade-in-0">
	<div class="tia-container">
		<input
			bind:this={inputRef}
			bind:value={title}
			onkeydown={handleKeydown}
			onfocus={handleFocus}
			type="text"
			placeholder="+ Add a task..."
			class="tia-input"
		/>

		{#if isExpanded}
			<div class="tia-actions-bar animate-in slide-in-from-top-2">
				<div class="flex items-center gap-2">
					<!-- Project (if not already set) -->
					{#if !projectId}
						<DropdownMenu.Root>
							<DropdownMenu.Trigger class="btn-pill btn-pill-ghost btn-pill-xs flex items-center gap-1">
								<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
								</svg>
								{projectName || 'Project'}
							</DropdownMenu.Trigger>
							<DropdownMenu.Portal>
								<DropdownMenu.Content
									class="z-50 min-w-[160px] tia-dropdown rounded-xl p-1 animate-in fade-in-0 zoom-in-95"
									sideOffset={4}
								>
									<DropdownMenu.Item class="px-3 py-2 text-sm tia-dropdown-item rounded-lg cursor-pointer">
										No projects yet
									</DropdownMenu.Item>
								</DropdownMenu.Content>
							</DropdownMenu.Portal>
						</DropdownMenu.Root>
					{/if}

					<!-- Priority -->
					<DropdownMenu.Root>
						<DropdownMenu.Trigger class="btn-pill btn-pill-ghost btn-pill-xs flex items-center gap-1">
							<span class="w-2 h-2 rounded-full" style="background: {priorityDotColors[priority]}"></span>
							{priorityOptions.find(p => p.value === priority)?.label}
						</DropdownMenu.Trigger>
						<DropdownMenu.Portal>
							<DropdownMenu.Content
								class="z-50 min-w-[140px] tia-dropdown rounded-xl p-1 animate-in fade-in-0 zoom-in-95"
								sideOffset={4}
							>
								{#each priorityOptions as option}
									<DropdownMenu.Item
										class="flex items-center gap-2 px-3 py-2 text-sm tia-dropdown-item rounded-lg cursor-pointer"
										onclick={() => priority = option.value}
									>
										<span class="w-2 h-2 rounded-full" style="background: {priorityDotColors[option.value]}"></span>
										{option.label}
									</DropdownMenu.Item>
								{/each}
							</DropdownMenu.Content>
						</DropdownMenu.Portal>
					</DropdownMenu.Root>

					<!-- Assignee -->
					<DropdownMenu.Root>
						<DropdownMenu.Trigger
							class="btn-pill btn-pill-ghost btn-pill-xs flex items-center gap-1"
							aria-label="Assign task to team member"
						>
							{#if assigneeId}
								{@const selectedMember = $team.members.find(m => m.id === assigneeId)}
								<div class="w-4 h-4 rounded-full flex items-center justify-center text-[10px] font-medium" style="background: var(--dbg3, #eee); color: var(--dt, #111)">
									{selectedMember?.name?.charAt(0) || '?'}
								</div>
								<span class="max-w-16 truncate">{selectedMember?.name || 'Assigned'}</span>
							{:else}
								<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
								</svg>
								<span>Assign</span>
							{/if}
						</DropdownMenu.Trigger>
						<DropdownMenu.Portal>
							<DropdownMenu.Content
								class="z-50 min-w-[180px] tia-dropdown rounded-xl p-1 animate-in fade-in-0 zoom-in-95"
								sideOffset={4}
							>
								<DropdownMenu.Item
									class="flex items-center gap-2 px-3 py-2 text-sm tia-dropdown-item rounded-lg cursor-pointer"
									onclick={() => assigneeId = undefined}
								>
									<svg class="w-4 h-4"  style="color: var(--dt3, #888)" fill="none" stroke="currentColor" viewBox="0 0 24 24">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
									</svg>
									<span>Unassigned</span>
									{#if !assigneeId}
										<svg class="w-4 h-4 ml-auto" style="color: var(--dt, #111)" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
										</svg>
									{/if}
								</DropdownMenu.Item>
								{#if $team.members.length > 0}
									<DropdownMenu.Separator class="my-1 h-px" style="background: var(--dbd, #e0e0e0)" />
									{#each $team.members as member}
										<DropdownMenu.Item
											class="flex items-center gap-2 px-3 py-2 text-sm tia-dropdown-item rounded-lg cursor-pointer"
											onclick={() => assigneeId = member.id}
										>
											<div class="w-5 h-5 rounded-full flex items-center justify-center text-xs font-medium shrink-0" style="background: var(--dbg3, #eee); color: var(--dt, #111)">
												{member.name?.charAt(0) || '?'}
											</div>
											<span class="truncate">{member.name}</span>
											{#if assigneeId === member.id}
												<svg class="w-4 h-4 ml-auto text-gray-900 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
													<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
												</svg>
											{/if}
										</DropdownMenu.Item>
									{/each}
								{:else if $team.loading}
									<div class="px-3 py-2 text-sm" style="color: var(--dt3, #888)">Loading...</div>
								{:else}
									<div class="px-3 py-2 text-sm" style="color: var(--dt3, #888)">No team members</div>
								{/if}
							</DropdownMenu.Content>
						</DropdownMenu.Portal>
					</DropdownMenu.Root>

					<!-- Due Date -->
					<button class="btn-pill btn-pill-ghost btn-pill-xs flex items-center gap-1">
						<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
						</svg>
						Due
					</button>
				</div>

				<div class="flex items-center gap-2">
					<button
						onclick={handleCancel}
						class="btn-pill btn-pill-soft btn-pill-xs"
					>
						Cancel
					</button>
					<button
						onclick={handleSubmit}
						disabled={!title.trim()}
						class="btn-pill btn-pill-primary btn-pill-xs disabled:opacity-50 disabled:cursor-not-allowed"
					>
						Add
					</button>
				</div>
			</div>
		{/if}
	</div>
</div>

<style>
	.tia-container {
		border: 1px solid var(--dbd, #e0e0e0);
		border-radius: 0.75rem;
		background: var(--dbg, #fff);
		overflow: hidden;
		transition: all 0.15s;
	}
	.tia-container:focus-within {
		border-color: var(--dt3, #888);
		box-shadow: 0 1px 3px rgba(0,0,0,0.06);
	}
	.tia-input {
		width: 100%;
		padding: 0.75rem 1rem;
		font-size: 0.875rem;
		color: var(--dt, #111);
		background: transparent;
		border: none;
		outline: none;
	}
	.tia-input::placeholder {
		color: var(--dt4, #bbb);
	}
	.tia-actions-bar {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 0.5rem 0.75rem;
		border-top: 1px solid var(--dbd2, #f0f0f0);
		background: var(--dbg2, #f5f5f5);
	}
	:global(.tia-dropdown) {
		background: var(--dbg, #fff);
		border: 1px solid var(--dbd, #e0e0e0);
		box-shadow: 0 4px 16px rgba(0,0,0,0.1);
	}
	:global(.tia-dropdown-item) {
		color: var(--dt, #111);
	}
	:global(.tia-dropdown-item:hover) {
		background: var(--dbg2, #f5f5f5);
	}
</style>
