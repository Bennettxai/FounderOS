<script lang="ts">
	import { DropdownMenu } from 'bits-ui';
	import { fly } from 'svelte/transition';

	type ViewMode = 'list' | 'board';
	type GroupBy = 'status' | 'priority' | 'project' | 'assignee' | 'none';

	interface Props {
		view?: ViewMode;
		groupBy?: GroupBy;
		searchQuery?: string;
		onViewChange?: (view: ViewMode) => void;
		onGroupByChange?: (groupBy: GroupBy) => void;
		onSearchChange?: (query: string) => void;
		onFilterChange?: (filters: Record<string, string[]>) => void;
		onNewTask?: () => void;
	}

	let {
		view = $bindable('list'),
		groupBy = $bindable('status'),
		searchQuery = $bindable(''),
		onViewChange,
		onGroupByChange,
		onSearchChange,
		onFilterChange,
		onNewTask
	}: Props = $props();

	let filterOpen = $state(false);
	let groupByOpen = $state(false);

	const groupByOptions: { value: GroupBy; label: string }[] = [
		{ value: 'none', label: 'None' },
		{ value: 'status', label: 'Status' },
		{ value: 'priority', label: 'Priority' },
		{ value: 'project', label: 'Project' },
		{ value: 'assignee', label: 'Assignee' }
	];

	function handleViewChange(newView: ViewMode) {
		view = newView;
		onViewChange?.(newView);
	}

	function handleGroupByChange(newGroupBy: GroupBy) {
		groupBy = newGroupBy;
		onGroupByChange?.(newGroupBy);
		groupByOpen = false;
	}

	function handleSearchInput(e: Event) {
		const target = e.target as HTMLInputElement;
		searchQuery = target.value;
		onSearchChange?.(searchQuery);
	}
</script>

<div class="tb-toolbar">
	<!-- View Switcher -->
	<div class="tb-view-toggle">
		<button
			onclick={() => handleViewChange('list')}
			class="tb-view-btn {view === 'list' ? 'tb-view-btn--active' : ''}"
			aria-label="List view"
		>
			<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 10h16M4 14h16M4 18h16" />
			</svg>
			<span>List</span>
		</button>
		<button
			onclick={() => handleViewChange('board')}
			class="tb-view-btn {view === 'board' ? 'tb-view-btn--active' : ''}"
			aria-label="Board view"
		>
			<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 17V7m0 10a2 2 0 01-2 2H5a2 2 0 01-2-2V7a2 2 0 012-2h2a2 2 0 012 2m0 10a2 2 0 002 2h2a2 2 0 002-2M9 7a2 2 0 012-2h2a2 2 0 012 2m0 10V7m0 10a2 2 0 002 2h2a2 2 0 002-2V7a2 2 0 00-2-2h-2a2 2 0 00-2 2" />
			</svg>
			<span>Board</span>
		</button>
	</div>

	<!-- Center Actions -->
	<div class="flex items-center gap-2">
		<!-- Filter Dropdown -->
		<DropdownMenu.Root bind:open={filterOpen}>
			<DropdownMenu.Trigger class="tb-action-btn">
				<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 4a1 1 0 011-1h16a1 1 0 011 1v2.586a1 1 0 01-.293.707l-6.414 6.414a1 1 0 00-.293.707V17l-4 4v-6.586a1 1 0 00-.293-.707L3.293 7.293A1 1 0 013 6.586V4z" />
				</svg>
				Filter
				<svg class="w-3 h-3 tb-chevron" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
				</svg>
			</DropdownMenu.Trigger>
			<DropdownMenu.Portal>
				<DropdownMenu.Content
					class="z-50 w-64 tb-dropdown rounded-xl p-3"
					sideOffset={4}
				>
					<div class="space-y-4">
						<div>
							<p class="tb-dropdown-heading">Status</p>
							<div class="space-y-1">
								{#each ['To Do', 'In Progress', 'In Review', 'Done', 'Blocked'] as status}
									<label class="tb-dropdown-check-row">
										<span class="tb-custom-check">
											<svg class="w-2.5 h-2.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
												<path stroke-linecap="round" stroke-linejoin="round" stroke-width="3" d="M5 13l4 4L19 7" />
											</svg>
										</span>
										<input type="checkbox" class="sr-only" checked />
										<span class="tb-dropdown-label">{status}</span>
									</label>
								{/each}
							</div>
						</div>

						<div>
							<p class="tb-dropdown-heading">Priority</p>
							<div class="space-y-1">
								{#each ['Critical', 'High', 'Medium', 'Low'] as prio}
									<label class="tb-dropdown-check-row">
										<span class="tb-custom-check">
											<svg class="w-2.5 h-2.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
												<path stroke-linecap="round" stroke-linejoin="round" stroke-width="3" d="M5 13l4 4L19 7" />
											</svg>
										</span>
										<input type="checkbox" class="sr-only" checked />
										<span class="tb-dropdown-label">{prio}</span>
									</label>
								{/each}
							</div>
						</div>

						<div class="flex items-center justify-between pt-2 tb-dropdown-footer">
							<button class="tb-dropdown-text-btn">Clear All</button>
							<button class="tb-dropdown-apply-btn">Apply</button>
						</div>
					</div>
				</DropdownMenu.Content>
			</DropdownMenu.Portal>
		</DropdownMenu.Root>

		<!-- Group By Dropdown -->
		{#if view === 'list'}
			<DropdownMenu.Root bind:open={groupByOpen}>
				<DropdownMenu.Trigger class="tb-action-btn">
					<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
					</svg>
					Group
					<svg class="w-3 h-3 tb-chevron" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
					</svg>
				</DropdownMenu.Trigger>
				<DropdownMenu.Portal>
					<DropdownMenu.Content
						class="z-50 min-w-[160px] tb-dropdown rounded-xl p-1"
						sideOffset={4}
					>
						{#each groupByOptions as option}
							<DropdownMenu.Item
								class="tb-dropdown-item {groupBy === option.value ? 'tb-dropdown-item--active' : ''}"
								onclick={() => handleGroupByChange(option.value)}
							>
								{#if groupBy === option.value}
									<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
									</svg>
								{:else}
									<span class="w-4"></span>
								{/if}
								{option.label}
							</DropdownMenu.Item>
						{/each}
					</DropdownMenu.Content>
				</DropdownMenu.Portal>
			</DropdownMenu.Root>
		{/if}
	</div>

	<!-- Search -->
	<div class="tb-search-wrap">
		<svg class="tb-search-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
		</svg>
		<input
			type="text"
			placeholder="Search tasks..."
			value={searchQuery}
			oninput={handleSearchInput}
			class="tb-search-input"
		/>
	</div>

	<!-- New Task -->
	{#if onNewTask}
		<button
			onclick={onNewTask}
			class="btn-cta tb-new-task-btn"
			aria-label="Create new task"
		>
			<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
			</svg>
			<span class="tb-new-task-label">New Task</span>
		</button>
	{/if}
</div>

<style>
	.tb-toolbar {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		justify-content: space-between;
		gap: 0.75rem;
		padding: 0.625rem 1.5rem;
		background: var(--dbg, #fff);
		border-bottom: 1px solid var(--dbd, #e0e0e0);
	}

	/* ── View toggle (compact segmented) ── */
	.tb-view-toggle {
		display: inline-flex;
		align-items: center;
		gap: 1px;
		border: 1px solid var(--dbd, #e5e7eb);
		border-radius: 0.5rem;
		padding: 2px;
		background: var(--dbg2, #f9fafb);
	}
	.tb-view-btn {
		display: flex;
		align-items: center;
		gap: 0.375rem;
		padding: 0.25rem 0.625rem;
		border-radius: 0.375rem;
		font-size: 0.8125rem;
		color: var(--dt3, #888);
		background: transparent;
		border: none;
		cursor: pointer;
		transition: all 0.15s;
		white-space: nowrap;
	}
	.tb-view-btn:hover { color: var(--dt, #111); }
	.tb-view-btn--active {
		color: var(--dt);
		background: color-mix(in srgb, var(--dt) 8%, transparent);
		font-weight: 600;
	}
	:global(.dark) .tb-view-btn--active {
		color: var(--dt);
		background: color-mix(in srgb, var(--dt) 8%, transparent);
	}

	/* ── Action buttons (Filter, Group) — :global because DropdownMenu.Trigger renders its own element ── */
	:global(.tb-action-btn) {
		display: inline-flex;
		align-items: center;
		gap: 0.375rem;
		padding: 0.375rem 0.625rem;
		font-size: 0.8125rem;
		font-weight: 500;
		color: var(--dt2, #555);
		background: var(--dbg, #fff);
		border: 1px solid var(--dbd, #e0e0e0);
		border-radius: 0.5rem;
		cursor: pointer;
		transition: all 0.15s;
		white-space: nowrap;
	}
	:global(.tb-action-btn:hover) {
		color: var(--dt, #111);
		background: var(--dbg2, #f5f5f5);
		border-color: var(--dt3, #888);
	}
	:global(.tb-chevron) {
		color: var(--dt4, #bbb);
		margin-left: -0.125rem;
	}

	/* ── Dropdown ── */
	:global(.tb-dropdown) {
		background: var(--dbg, #fff);
		border: 1px solid var(--dbd, #e0e0e0);
		box-shadow: 0 4px 16px rgba(0,0,0,0.1);
	}
	:global(.tb-dropdown-heading) {
		font-size: 0.6875rem;
		font-weight: 600;
		color: var(--dt3, #888);
		text-transform: uppercase;
		letter-spacing: 0.04em;
		margin-bottom: 0.5rem;
	}
	:global(.tb-dropdown-check-row) {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		padding: 0.375rem 0.5rem;
		border-radius: 0.375rem;
		cursor: pointer;
		transition: background 0.12s;
	}
	:global(.tb-dropdown-check-row:hover) {
		background: var(--dbg2, #f5f5f5);
	}
	:global(.tb-custom-check) {
		width: 0.875rem;
		height: 0.875rem;
		border-radius: 0.1875rem;
		background: var(--dt);
		color: var(--dbg);
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
	}
	:global(.tb-dropdown-label) {
		font-size: 0.8125rem;
		color: var(--dt, #111);
	}
	:global(.tb-dropdown-footer) {
		border-top: 1px solid var(--dbd2, #f0f0f0);
	}
	:global(.tb-dropdown-text-btn) {
		font-size: 0.75rem;
		color: var(--dt3, #888);
		background: none;
		border: none;
		cursor: pointer;
		padding: 0.25rem 0;
	}
	:global(.tb-dropdown-text-btn:hover) { color: var(--dt, #111); }
	:global(.tb-dropdown-apply-btn) {
		font-size: 0.75rem;
		font-weight: 600;
		color: var(--dbg);
		background: var(--dt);
		border: none;
		border-radius: 0.375rem;
		padding: 0.3rem 0.75rem;
		cursor: pointer;
		transition: all 0.15s;
	}
	:global(.tb-dropdown-apply-btn:hover) { background: var(--dt2); }
	:global(.tb-dropdown-item) {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		padding: 0.5rem 0.75rem;
		font-size: 0.8125rem;
		color: var(--dt2, #555);
		border-radius: 0.5rem;
		cursor: pointer;
		transition: background 0.12s;
	}
	:global(.tb-dropdown-item:hover) { background: var(--dbg2, #f5f5f5); }
	:global(.tb-dropdown-item--active) {
		color: var(--dt);
		font-weight: 600;
		background: color-mix(in srgb, var(--dt) 8%, transparent);
	}
	:global(.dark .tb-dropdown-item--active) {
		color: var(--dt);
		background: color-mix(in srgb, var(--dt) 8%, transparent);
	}

	/* ── Search ── */
	.tb-search-wrap {
		position: relative;
		flex: 1;
		min-width: 0;
		max-width: 16rem;
	}
	.tb-search-icon {
		position: absolute;
		left: 0.75rem;
		top: 50%;
		transform: translateY(-50%);
		width: 1rem;
		height: 1rem;
		color: var(--dt4, #bbb);
	}
	.tb-search-input {
		width: 100%;
		padding: 0.375rem 0.75rem 0.375rem 2.25rem;
		font-size: 0.8125rem;
		background: var(--dbg2, #f5f5f5);
		border: 1px solid var(--dbd, #e0e0e0);
		border-radius: 0.5rem;
		color: var(--dt, #111);
		outline: none;
		transition: border-color 0.15s;
	}
	.tb-search-input:focus {
		border-color: var(--dbd);
		box-shadow: 0 0 0 2px color-mix(in srgb, var(--dt) 10%, transparent);
	}

	/* ── New Task button (inline with toolbar) ── */
	.tb-new-task-btn {
		flex-shrink: 0;
		white-space: nowrap;
	}
	/* Hide label text on very small viewports; icon-only still readable */
	@media (max-width: 480px) {
		.tb-new-task-label {
			display: none;
		}
	}
</style>
