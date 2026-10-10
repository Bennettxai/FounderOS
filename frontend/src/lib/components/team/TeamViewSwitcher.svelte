<script lang="ts">
	type ViewMode = 'directory' | 'orgchart' | 'capacity';

	interface Props {
		view?: ViewMode;
		searchQuery?: string;
		onViewChange?: (view: ViewMode) => void;
		onSearchChange?: (query: string) => void;
	}

	let {
		view = $bindable('directory'),
		searchQuery = $bindable(''),
		onViewChange,
		onSearchChange
	}: Props = $props();

	const viewOptions: { value: ViewMode; label: string; icon: string }[] = [
		{
			value: 'directory',
			label: 'Directory',
			icon: `<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
			</svg>`
		},
		{
			value: 'orgchart',
			label: 'Org Chart',
			icon: `<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4" />
			</svg>`
		},
		{
			value: 'capacity',
			label: 'Capacity',
			icon: `<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
			</svg>`
		}
	];

	function handleViewChange(newView: ViewMode) {
		view = newView;
		onViewChange?.(newView);
	}

	function handleSearchInput(e: Event) {
		const target = e.target as HTMLInputElement;
		searchQuery = target.value;
		onSearchChange?.(searchQuery);
	}
</script>

<div class="td-view-bar">
	<!-- View Switcher -->
	<div class="td-view-tabs">
		{#each viewOptions as option}
			<button
				onclick={() => handleViewChange(option.value)}
				class="td-view-tab {view === option.value ? 'td-view-tab--active' : ''}"
			>
				{@html option.icon}
				<span>{option.label}</span>
			</button>
		{/each}
	</div>

	<!-- Search -->
	<div class="td-search-wrap">
		<svg class="td-search-wrap__icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
		</svg>
		<input
			type="text"
			placeholder="Search team..."
			value={searchQuery}
			oninput={handleSearchInput}
			class="td-search-input"
		/>
	</div>
</div>

<style>
	.td-view-bar {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 10px 20px;
		border-bottom: 1px solid var(--dbd, #e0e0e0);
		background: var(--dbg, #fff);
	}

	.td-view-tabs {
		display: flex;
		align-items: center;
		gap: 2px;
		background: var(--dbg2, #f5f5f5);
		border-radius: 9px;
		padding: 3px;
	}

	.td-view-tab {
		display: flex;
		align-items: center;
		gap: 6px;
		padding: 5px 12px;
		border-radius: 7px;
		font-size: 13px;
		font-weight: 500;
		color: var(--dt3, #888);
		background: transparent;
		border: none;
		cursor: pointer;
		transition: all 0.15s;
		white-space: nowrap;
	}
	.td-view-tab:hover { color: var(--dt, #111); }
	.td-view-tab--active {
		background: var(--dbg, #fff);
		color: var(--dt, #111);
		font-weight: 600;
		box-shadow: 0 1px 3px rgba(0,0,0,0.08);
	}
	.td-view-tab :global(svg) { width: 15px; height: 15px; }

	.td-search-wrap {
		position: relative;
	}
	.td-search-wrap__icon {
		position: absolute;
		left: 10px;
		top: 50%;
		transform: translateY(-50%);
		width: 15px;
		height: 15px;
		color: var(--dt4, #bbb);
		pointer-events: none;
	}
	.td-search-input {
		width: 220px;
		padding: 7px 12px 7px 34px;
		font-size: 13px;
		border: 1px solid var(--dbd, #e0e0e0);
		border-radius: 9px;
		background: var(--dbg2, #f5f5f5);
		color: var(--dt, #111);
		outline: none;
		transition: border-color 0.15s, box-shadow 0.15s;
	}
	.td-search-input::placeholder { color: var(--dt4, #bbb); }
	.td-search-input:focus {
		border-color: var(--dt, #111);
		box-shadow: 0 0 0 2px rgba(0,0,0,0.06);
	}
</style>
