<script lang="ts">
	import { DropdownMenu } from 'bits-ui';
	import type { ViewMode } from '$lib/stores/clients';
	import type { ClientStatus, ClientType } from '$lib/api';

	interface Props {
		view: ViewMode;
		searchQuery: string;
		statusFilter: ClientStatus | null;
		typeFilter: ClientType | null;
		onViewChange: (view: ViewMode) => void;
		onSearchChange: (query: string) => void;
		onStatusChange: (status: ClientStatus | null) => void;
		onTypeChange: (type: ClientType | null) => void;
	}

	let {
		view = $bindable(),
		searchQuery = $bindable(),
		statusFilter,
		typeFilter,
		onViewChange,
		onSearchChange,
		onStatusChange,
		onTypeChange
	}: Props = $props();

	const views: { id: ViewMode; label: string; icon: string }[] = [
		{ id: 'table', label: 'Table', icon: 'M3 10h18M3 14h18M3 18h18M3 6h18' },
		{
			id: 'cards',
			label: 'Cards',
			icon: 'M4 5a1 1 0 011-1h4a1 1 0 011 1v4a1 1 0 01-1 1H5a1 1 0 01-1-1V5zm10 0a1 1 0 011-1h4a1 1 0 011 1v4a1 1 0 01-1 1h-4a1 1 0 01-1-1V5zM4 15a1 1 0 011-1h4a1 1 0 011 1v4a1 1 0 01-1 1H5a1 1 0 01-1-1v-4zm10 0a1 1 0 011-1h4a1 1 0 011 1v4a1 1 0 01-1 1h-4a1 1 0 01-1-1v-4z'
		},
		{
			id: 'kanban',
			label: 'Pipeline',
			icon: 'M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2'
		}
	];

	const statuses: { id: ClientStatus | ''; label: string }[] = [
		{ id: '', label: 'All Statuses' },
		{ id: 'lead', label: 'Leads' },
		{ id: 'prospect', label: 'Prospects' },
		{ id: 'active', label: 'Active' },
		{ id: 'inactive', label: 'Inactive' },
		{ id: 'churned', label: 'Churned' }
	];

	const types: { id: ClientType | ''; label: string }[] = [
		{ id: '', label: 'All Types' },
		{ id: 'company', label: 'Companies' },
		{ id: 'individual', label: 'Individuals' }
	];

	// Get display label for current filter
	const getStatusLabel = $derived(
		statuses.find((s) => s.id === (statusFilter || ''))?.label || 'All Statuses'
	);

	const getTypeLabel = $derived(
		types.find((t) => t.id === (typeFilter || ''))?.label || 'All Types'
	);
</script>

<div class="cr-toolbar">
	<div class="cr-toolbar__inner">
		<!-- Left: Search & Filters -->
		<div class="cr-toolbar__left">
			<!-- Search -->
			<div class="cr-toolbar__search-wrap">
				<svg
					class="cr-toolbar__search-icon"
					fill="none"
					stroke="currentColor"
					viewBox="0 0 24 24"
					aria-hidden="true"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
					/>
				</svg>
				<input
					type="text"
					placeholder="Search clients..."
					bind:value={searchQuery}
					oninput={() => onSearchChange(searchQuery)}
					class="cr-toolbar__search-input"
					aria-label="Search clients"
				/>
			</div>

			<!-- Status Filter -->
			<DropdownMenu.Root>
				<DropdownMenu.Trigger class="cr-toolbar__filter-btn" aria-label="Filter by status">
					<span>{getStatusLabel}</span>
					<svg class="cr-toolbar__chevron" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
					</svg>
				</DropdownMenu.Trigger>
				<DropdownMenu.Portal>
					<DropdownMenu.Content class="cr-dropdown" sideOffset={4}>
						{#each statuses as status}
							<DropdownMenu.Item
								class="cr-dropdown__item {status.id === (statusFilter || '') ? 'cr-dropdown__item--active' : ''}"
								onclick={() => onStatusChange(status.id || null)}
							>
								{status.label}
							</DropdownMenu.Item>
						{/each}
					</DropdownMenu.Content>
				</DropdownMenu.Portal>
			</DropdownMenu.Root>

			<!-- Type Filter -->
			<DropdownMenu.Root>
				<DropdownMenu.Trigger class="cr-toolbar__filter-btn" aria-label="Filter by type">
					<span>{getTypeLabel}</span>
					<svg class="cr-toolbar__chevron" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
					</svg>
				</DropdownMenu.Trigger>
				<DropdownMenu.Portal>
					<DropdownMenu.Content class="cr-dropdown" sideOffset={4}>
						{#each types as type}
							<DropdownMenu.Item
								class="cr-dropdown__item {type.id === (typeFilter || '') ? 'cr-dropdown__item--active' : ''}"
								onclick={() => onTypeChange(type.id || null)}
							>
								{type.label}
							</DropdownMenu.Item>
						{/each}
					</DropdownMenu.Content>
				</DropdownMenu.Portal>
			</DropdownMenu.Root>
		</div>

		<!-- Right: View Switcher -->
		<div class="cr-view-switcher">
			{#each views as v}
				<button
					onclick={() => {
						view = v.id;
						onViewChange(v.id);
					}}
					class="cr-view-switcher__tab {view === v.id ? 'cr-view-switcher__tab--active' : ''}"
					aria-label="Switch to {v.label} view"
					aria-pressed={view === v.id}
				>
					<svg class="cr-view-switcher__icon" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d={v.icon} />
					</svg>
					{v.label}
				</button>
			{/each}
		</div>
	</div>
</div>

<style>
	/* ─── Toolbar (cr- prefix, Foundation tokens) ──────────────── */
	.cr-toolbar {
		padding: 10px 24px;
		background: var(--dbg2, #f5f5f5);
	}
	.cr-toolbar__inner {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 16px;
	}
	.cr-toolbar__left {
		display: flex;
		align-items: center;
		gap: 10px;
		flex: 1;
	}

	/* Search */
	.cr-toolbar__search-wrap {
		position: relative;
		flex: 1;
		max-width: 280px;
	}
	.cr-toolbar__search-icon {
		position: absolute;
		left: 10px;
		top: 50%;
		transform: translateY(-50%);
		width: 16px;
		height: 16px;
		color: var(--dt3, #888);
		pointer-events: none;
	}
	.cr-toolbar__search-input {
		width: 100%;
		padding: 7px 12px 7px 32px;
		font-size: 13px;
		border: 1px solid var(--dbd, #e0e0e0);
		border-radius: 8px;
		background: var(--dbg, #fff);
		color: var(--dt, #111);
		outline: none;
		transition: border-color 0.12s, box-shadow 0.12s;
	}
	.cr-toolbar__search-input::placeholder {
		color: var(--dt4, #bbb);
	}
	.cr-toolbar__search-input:focus {
		border-color: var(--dt3, #888);
		box-shadow: 0 0 0 2px rgba(0, 0, 0, 0.06);
	}

	/* Filter buttons — :global because bits-ui DropdownMenu.Trigger renders its own element */
	:global(.cr-toolbar__filter-btn) {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		padding: 7px 12px;
		font-size: 13px;
		font-weight: 500;
		border: 1px solid var(--dbd, #e0e0e0);
		border-radius: 8px;
		background: var(--dbg, #fff);
		color: var(--dt, #111);
		cursor: pointer;
		transition: background 0.12s, border-color 0.12s;
		white-space: nowrap;
	}
	:global(.cr-toolbar__filter-btn:hover) {
		background: var(--dbg2, #f5f5f5);
		border-color: var(--dbd, #e0e0e0);
	}
	.cr-toolbar__chevron {
		width: 14px;
		height: 14px;
		color: var(--dt3, #888);
		flex-shrink: 0;
	}

	/* Dropdown (Foundation CRM pattern) — :global because bits-ui portals outside component */
	:global(.cr-dropdown) {
		z-index: 50;
		min-width: 160px;
		border-radius: 10px;
		border: 1px solid var(--dbd2, #f0f0f0);
		background: var(--dbg, #fff);
		box-shadow: 0 8px 24px rgba(0, 0, 0, 0.12);
		overflow: hidden;
		padding: 4px;
	}
	:global(.cr-dropdown__item) {
		display: block;
		width: 100%;
		padding: 7px 10px;
		border: none;
		background: transparent;
		color: var(--dt, #111);
		font-size: 13px;
		font-weight: 500;
		text-align: left;
		cursor: pointer;
		border-radius: 7px;
		transition: background 0.1s;
	}
	:global(.cr-dropdown__item:hover),
	:global(.cr-dropdown__item[data-highlighted]) {
		background: var(--dbg2, #f5f5f5);
	}
	:global(.cr-dropdown__item--active) {
		background: var(--dbg2, #f5f5f5);
		font-weight: 600;
	}

	/* View Switcher (Foundation CRM pattern) */
	.cr-view-switcher {
		display: inline-flex;
		align-items: center;
		gap: 2px;
		padding: 3px;
		border-radius: 10px;
		border: 1px solid var(--dbd, #e0e0e0);
		background: var(--dbg2, #f5f5f5);
	}
	.cr-view-switcher__tab {
		display: inline-flex;
		align-items: center;
		gap: 5px;
		padding: 5px 10px;
		font-size: 13px;
		font-weight: 500;
		border: none;
		border-radius: 7px;
		background: transparent;
		color: var(--dt3, #888);
		cursor: pointer;
		transition: background 0.12s, color 0.12s, box-shadow 0.12s;
		white-space: nowrap;
	}
	.cr-view-switcher__tab:hover {
		color: var(--dt, #111);
	}
	.cr-view-switcher__tab--active {
		background: var(--dbg, #fff);
		color: var(--dt, #111);
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
	}
	.cr-view-switcher__icon {
		width: 16px;
		height: 16px;
		flex-shrink: 0;
	}
</style>
