<script lang="ts">
	/**
	 * AppRenderer - Main orchestrator that renders an entire app from config
	 * Takes an AppConfig and renders the complete UI
	 */

	import type { AppConfig, AppState, RecordData } from '../types';
	import type { SortConfig } from '../types';
	import AppShell from './AppShell.svelte';
	import { DataTable } from '../views';

	interface Props {
		config: AppConfig;
		data: RecordData[];
		loading?: boolean;
		onrecordclick?: (record: RecordData) => void;
		onrecordcreate?: () => void;
		onrecordedit?: (recordId: string, fieldId: string, value: unknown) => void;
		onselectionchange?: (ids: string[]) => void;
	}

	let {
		config,
		data,
		loading = false,
		onrecordclick,
		onrecordcreate,
		onrecordedit,
		onselectionchange
	}: Props = $props();

	// App state
	let selectedIds = $state<string[]>([]);
	let sort = $state<SortConfig[]>([]);
	let searchQuery = $state('');

	// Default view ID derived from config
	const defaultViewId = $derived(config.defaultViewId ?? config.views[0]?.id ?? '');
	let currentViewId = $state('');

	// Initialize view ID from config
	$effect(() => {
		if (!currentViewId && defaultViewId) {
			currentViewId = defaultViewId;
		}
	});

	// Derived state
	const currentView = $derived(config.views.find((v) => v.id === currentViewId) ?? config.views[0]);

	const filteredData = $derived(() => {
		let result = [...data];

		// Search filter
		if (searchQuery) {
			const query = searchQuery.toLowerCase();
			result = result.filter((record) => {
				return config.fields.some((field) => {
					const value = record[field.id];
					if (value == null) return false;
					return String(value).toLowerCase().includes(query);
				});
			});
		}

		// Sort
		if (sort.length > 0) {
			result.sort((a, b) => {
				for (const s of sort) {
					const aVal = a[s.fieldId];
					const bVal = b[s.fieldId];

					if (aVal == null && bVal == null) continue;
					if (aVal == null) return s.direction === 'asc' ? 1 : -1;
					if (bVal == null) return s.direction === 'asc' ? -1 : 1;

					if (aVal < bVal) return s.direction === 'asc' ? -1 : 1;
					if (aVal > bVal) return s.direction === 'asc' ? 1 : -1;
				}
				return 0;
			});
		}

		return result;
	});

	const views = $derived(
		config.views.map((v) => ({
			id: v.id,
			name: v.name,
			type: v.type
		}))
	);

	function handleSearch(query: string) {
		searchQuery = query;
	}

	function handleViewChange(viewId: string) {
		currentViewId = viewId;
	}

	function handleSelect(ids: string[]) {
		selectedIds = ids;
		onselectionchange?.(ids);
	}

	function handleSort(newSort: SortConfig[]) {
		sort = newSort;
	}

	function handleRowClick(record: RecordData) {
		onrecordclick?.(record);
	}

	function handleCellEdit(recordId: string, fieldId: string, value: unknown) {
		onrecordedit?.(recordId, fieldId, value);
	}
</script>

<div class="tpl-app-renderer">
	<AppShell
		branding={config.branding}
		toolbar={config.toolbar}
		selectedCount={selectedIds.length}
		currentView={currentViewId}
		{views}
		onsearch={handleSearch}
		onviewchange={handleViewChange}
		oncreate={onrecordcreate}
	>
		{#if currentView?.type === 'table'}
			<DataTable
				fields={config.fields}
				data={filteredData()}
				config={currentView}
				{selectedIds}
				{sort}
				{loading}
				onselect={handleSelect}
				onsort={handleSort}
				onrowclick={handleRowClick}
				oncelledit={handleCellEdit}
			/>
		{:else}
			<!-- Placeholder for other view types -->
			<div class="tpl-view-placeholder">
				<p>View type "{currentView?.type}" coming soon</p>
			</div>
		{/if}
	</AppShell>
</div>

<style>
	.tpl-app-renderer {
		height: 100%;
		min-height: 500px;
	}

	.tpl-view-placeholder {
		display: flex;
		align-items: center;
		justify-content: center;
		height: 300px;
		background: var(--tpl-bg-primary);
		border: 2px dashed var(--tpl-border-default);
		border-radius: var(--tpl-radius-lg);
		color: var(--tpl-text-muted);
	}
</style>
