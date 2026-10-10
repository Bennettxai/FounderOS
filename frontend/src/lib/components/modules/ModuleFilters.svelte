<script lang="ts">
	import { Search, ArrowUpDown } from 'lucide-svelte';
	import type { ModuleCategory, ModuleFilters } from '$lib/types/modules';
	import { categoryLabels } from '$lib/types/modules';
	import { getCategoryColor } from '$lib/constants/colors';

	interface Props {
		filters: ModuleFilters;
		onFiltersChange: (filters: Partial<ModuleFilters>) => void;
	}

	let { filters, onFiltersChange }: Props = $props();

	let searchInput = $state(filters.search);

	// Debounce search
	let debounceTimer: number;
	function handleSearchInput(value: string) {
		searchInput = value;
		clearTimeout(debounceTimer);
		debounceTimer = setTimeout(() => {
			onFiltersChange({ search: value });
		}, 300) as unknown as number;
	}

	const categories: (ModuleCategory | null)[] = [
		null,
		'productivity',
		'communication',
		'finance',
		'analytics',
		'automation',
		'integration',
		'utilities',
		'custom'
	];

	const sortOptions: Array<{ value: 'popular' | 'newest' | 'name' | 'installs'; label: string }> = [
		{ value: 'popular', label: 'Popular' },
		{ value: 'newest', label: 'Newest' },
		{ value: 'name', label: 'Name' },
		{ value: 'installs', label: 'Most Installed' }
	];

	let showSortDropdown = $state(false);

	function handleSortSelect(value: ModuleFilters['sort']) {
		onFiltersChange({ sort: value });
		showSortDropdown = false;
	}
</script>

<div class="am-filters">
	<!-- Search Input -->
	<div class="am-filters__search">
		<Search class="am-filters__search-icon" />
		<input
			type="text"
			placeholder="Search modules..."
			value={searchInput}
			oninput={(e) => handleSearchInput(e.currentTarget.value)}
			class="am-filters__search-input"
		/>
	</div>

	<!-- Category Pills + Sort -->
	<div class="am-filters__row">
		<div class="am-filters__pills">
			{#each categories as cat}
				{@const isActive = filters.category === cat}
				{@const catColor = cat ? getCategoryColor(cat) : null}
				<button
					class="am-pill"
					class:am-pill--active={isActive}
					style={isActive && catColor ? `background: ${catColor}; color: var(--bos-surface-on-color); border-color: ${catColor};` : ''}
					onclick={() => onFiltersChange({ category: cat })}
				>
					{cat ? categoryLabels[cat] : 'All'}
				</button>
			{/each}
		</div>

		<!-- Sort Button -->
		<div class="am-filters__sort-wrap">
			<button
				class="am-sort-btn"
				onclick={() => showSortDropdown = !showSortDropdown}
				aria-label="Sort modules"
			>
				<ArrowUpDown class="w-3.5 h-3.5" />
				<span>{sortOptions.find(o => o.value === filters.sort)?.label ?? 'Sort'}</span>
			</button>
			{#if showSortDropdown}
				<!-- svelte-ignore a11y_no_static_element_interactions -->
				<div class="am-sort-dropdown" onmouseleave={() => showSortDropdown = false}>
					{#each sortOptions as option}
						<button
							class="am-sort-dropdown__item"
							class:am-sort-dropdown__item--active={filters.sort === option.value}
							onclick={() => handleSortSelect(option.value)}
						>
							{option.label}
						</button>
					{/each}
				</div>
			{/if}
		</div>
	</div>
</div>

<style>
	/* ══════════════════════════════════════════════════════════════ */
	/*  MODULE FILTERS v2 (am-filters-) — Foundation Design Tokens  */
	/* ══════════════════════════════════════════════════════════════ */
	.am-filters {
		display: flex;
		flex-direction: column;
		gap: 12px;
	}

	/* Search */
	.am-filters__search {
		position: relative;
	}
	.am-filters__search :global(.am-filters__search-icon) {
		position: absolute;
		left: 12px;
		top: 50%;
		transform: translateY(-50%);
		width: 16px;
		height: 16px;
		color: var(--dt3);
		pointer-events: none;
	}
	.am-filters__search-input {
		width: 100%;
		padding: 9px 14px 9px 36px;
		border-radius: 10px;
		border: 1px solid var(--dbd);
		background: var(--dbg);
		color: var(--dt);
		font-size: 13px;
		transition: border-color 0.15s, box-shadow 0.15s;
	}
	.am-filters__search-input::placeholder {
		color: var(--dt4);
	}
	.am-filters__search-input:focus {
		outline: none;
		border-color: var(--dt3);
		box-shadow: 0 0 0 2px rgba(0, 0, 0, 0.04);
	}

	/* Pills + Sort Row */
	.am-filters__row {
		display: flex;
		align-items: center;
		gap: 12px;
	}

	/* Category Pills */
	.am-filters__pills {
		display: flex;
		align-items: center;
		gap: 6px;
		overflow-x: auto;
		scrollbar-width: none;
		-ms-overflow-style: none;
		flex: 1;
		min-width: 0;
	}
	.am-filters__pills::-webkit-scrollbar {
		display: none;
	}

	.am-pill {
		display: inline-flex;
		align-items: center;
		white-space: nowrap;
		padding: 5px 14px;
		border-radius: 999px;
		font-size: 13px;
		font-weight: 500;
		border: 1px solid var(--dbd);
		background: var(--dbg2);
		color: var(--dt2);
		cursor: pointer;
		transition: all 0.15s;
		flex-shrink: 0;
	}
	.am-pill:hover {
		border-color: var(--dt3);
		color: var(--dt);
	}
	.am-pill--active {
		background: var(--dt);
		color: var(--bos-surface-on-color);
		border-color: var(--dt);
	}

	/* Sort */
	.am-filters__sort-wrap {
		position: relative;
		flex-shrink: 0;
	}
	.am-sort-btn {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		padding: 5px 12px;
		border-radius: 999px;
		font-size: 12px;
		font-weight: 500;
		border: 1px solid var(--dbd);
		background: transparent;
		color: var(--dt2);
		cursor: pointer;
		transition: all 0.15s;
		white-space: nowrap;
	}
	.am-sort-btn:hover {
		border-color: var(--dt3);
		color: var(--dt);
	}

	.am-sort-dropdown {
		position: absolute;
		top: calc(100% + 4px);
		right: 0;
		min-width: 150px;
		padding: 4px;
		border-radius: 10px;
		border: 1px solid var(--dbd);
		background: var(--dbg);
		box-shadow: 0 8px 24px rgba(0, 0, 0, 0.1);
		z-index: 50;
	}
	.am-sort-dropdown__item {
		display: block;
		width: 100%;
		text-align: left;
		padding: 8px 12px;
		border-radius: 6px;
		font-size: 12px;
		font-weight: 500;
		color: var(--dt2);
		background: transparent;
		border: none;
		cursor: pointer;
		transition: all 0.1s;
	}
	.am-sort-dropdown__item:hover {
		background: var(--dbg2);
		color: var(--dt);
	}
	.am-sort-dropdown__item--active {
		color: var(--dt);
		font-weight: 600;
	}
</style>
