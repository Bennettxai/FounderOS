<script lang="ts">
	/**
	 * TableToolbar - Filter, sort, hide columns, search
	 */
	import {
		Search,
		Filter,
		ArrowUpDown,
		EyeOff,
		Plus,
		Download,
		Upload,
		MoreHorizontal
	} from 'lucide-svelte';
	import type { Column, Filter as FilterType, Sort } from '$lib/api/tables/types';

	interface Props {
		columns: Column[];
		filters: FilterType[];
		sorts: Sort[];
		searchQuery: string;
		selectedCount: number;
		onSearchChange: (query: string) => void;
		onAddFilter?: () => void;
		onAddSort?: () => void;
		onHideFields?: () => void;
		onAddRow: () => void;
		onDeleteSelected?: () => void;
		onExport?: () => void;
		onImport?: () => void;
	}

	let {
		columns,
		filters,
		sorts,
		searchQuery,
		selectedCount,
		onSearchChange,
		onAddFilter,
		onAddSort,
		onHideFields,
		onAddRow,
		onDeleteSelected,
		onExport,
		onImport
	}: Props = $props();

	let showMoreMenu = $state(false);

	function handleClickOutside() {
		showMoreMenu = false;
	}
</script>

<svelte:window onclick={handleClickOutside} />

<div class="dt2-toolbar flex items-center justify-between px-4 py-2">
	<!-- Left: Search and Filters -->
	<div class="flex items-center gap-2">
		<!-- Search -->
		<div class="relative">
			<Search class="absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2" style="color: var(--dt3);" />
			<input
				type="text"
				placeholder="Search..."
				value={searchQuery}
				oninput={(e) => onSearchChange((e.target as HTMLInputElement).value)}
				class="dt2-search w-48 rounded-lg py-1.5 pl-8 pr-3 text-sm focus:outline-none"
			/>
		</div>

		<!-- Filter Button -->
		{#if onAddFilter}
			<button
				type="button"
				class="btn-pill btn-pill-sm flex items-center gap-1.5 {filters.length > 0 ? 'btn-pill-soft' : 'btn-pill-secondary'}"
				onclick={onAddFilter}
			>
				<Filter class="h-4 w-4" />
				Filter
				{#if filters.length > 0}
					<span class="rounded bg-blue-100 px-1.5 py-0.5 text-xs font-medium text-blue-600">
						{filters.length}
					</span>
				{/if}
			</button>
		{/if}

		<!-- Sort Button -->
		{#if onAddSort}
			<button
				type="button"
				class="btn-pill btn-pill-sm flex items-center gap-1.5 {sorts.length > 0 ? 'btn-pill-soft' : 'btn-pill-secondary'}"
				onclick={onAddSort}
			>
				<ArrowUpDown class="h-4 w-4" />
				Sort
				{#if sorts.length > 0}
					<span class="rounded bg-blue-100 px-1.5 py-0.5 text-xs font-medium text-blue-600">
						{sorts.length}
					</span>
				{/if}
			</button>
		{/if}

		<!-- Hide Fields Button -->
		{#if onHideFields}
			<button
				type="button"
				class="btn-pill btn-pill-secondary btn-pill-sm flex items-center gap-1.5"
				onclick={onHideFields}
			>
				<EyeOff class="h-4 w-4" />
				Hide fields
			</button>
		{/if}
	</div>

	<!-- Right: Actions -->
	<div class="flex items-center gap-2">
		<!-- Selection Actions -->
		{#if selectedCount > 0}
			<div class="flex items-center gap-2 pr-2" style="border-right: 1px solid var(--dbd);">
				<span class="text-sm" style="color: var(--dt2);">{selectedCount} selected</span>
				{#if onDeleteSelected}
					<button
						type="button"
						class="btn-pill btn-pill-ghost btn-pill-xs"
						onclick={onDeleteSelected}
					>
						Delete
					</button>
				{/if}
			</div>
		{/if}

		<!-- Add Row -->
		<button
			type="button"
			class="btn-cta flex items-center gap-1.5"
			onclick={onAddRow}
		>
			<Plus class="h-4 w-4" />
			Add row
		</button>

		<!-- More Menu -->
		<div class="relative">
			<button
				type="button"
				class="btn-pill btn-pill-ghost btn-pill-icon"
				onclick={(e) => {
					e.stopPropagation();
					showMoreMenu = !showMoreMenu;
				}}
			>
				<MoreHorizontal class="h-5 w-5" />
			</button>

			{#if showMoreMenu}
				<div
					class="dt2-dropdown absolute right-0 top-full mt-1 w-40 rounded-lg py-1"
				>
					{#if onExport}
						<button
							type="button"
							class="dt2-dropdown__item flex w-full items-center gap-2 px-3 py-2 text-sm transition-colors"
							onclick={() => {
								onExport();
								showMoreMenu = false;
							}}
						>
							<Download class="h-4 w-4" />
							Export
						</button>
					{/if}
					{#if onImport}
						<button
							type="button"
							class="dt2-dropdown__item flex w-full items-center gap-2 px-3 py-2 text-sm transition-colors"
							onclick={() => {
								onImport();
								showMoreMenu = false;
							}}
						>
							<Upload class="h-4 w-4" />
							Import
						</button>
					{/if}
				</div>
			{/if}
		</div>
	</div>
</div>

<style>
	.dt2-toolbar {
		background: var(--dbg2);
		border-bottom: 1px solid var(--dbd2);
		color: var(--dt);
	}

	.dt2-search {
		background: var(--dbg);
		border: 1px solid var(--dbd);
		color: var(--dt);
	}

	.dt2-search:focus {
		border-color: var(--dt2-accent, #3b82f6);
		box-shadow: 0 0 0 1px var(--dt2-accent, #3b82f6);
	}

	.dt2-dropdown {
		z-index: var(--bos-z-index-popover, 1001);
		background: var(--dbg);
		border: 1px solid var(--dbd);
		box-shadow: var(--shadow-lg);
	}

	.dt2-dropdown__item {
		color: var(--dt);
		border-radius: 6px;
	}

	.dt2-dropdown__item:hover {
		background: var(--dbg3);
	}
</style>
