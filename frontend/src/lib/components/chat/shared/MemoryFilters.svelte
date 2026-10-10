<script lang="ts">
	import type { MemoryType } from '$lib/api/memory';

	interface Props {
		selectedType: MemoryType | 'all';
		showPinnedOnly: boolean;
		importanceMin: number;
		dateRange: { start: string; end: string } | null;
		onTypeChange: (type: MemoryType | 'all') => void;
		onPinnedToggle: () => void;
		onImportanceChange: (min: number) => void;
		onDateRangeChange: (range: { start: string; end: string } | null) => void;
	}

	let {
		selectedType,
		showPinnedOnly,
		importanceMin,
		dateRange,
		onTypeChange,
		onPinnedToggle,
		onImportanceChange,
		onDateRangeChange
	}: Props = $props();

	let expanded = $state(false);

	const memoryTypes: { value: MemoryType | 'all'; label: string }[] = [
		{ value: 'all', label: 'All Types' },
		{ value: 'fact', label: 'Facts' },
		{ value: 'preference', label: 'Preferences' },
		{ value: 'decision', label: 'Decisions' },
		{ value: 'event', label: 'Events' },
		{ value: 'learning', label: 'Learnings' },
		{ value: 'context', label: 'Context' },
		{ value: 'relationship', label: 'Relationships' }
	];

	function handleTypeSelect(type: MemoryType | 'all') {
		onTypeChange(type);
	}

	function handleImportanceInput(e: Event) {
		const target = e.target as HTMLInputElement;
		onImportanceChange(Number(target.value));
	}

	function handleStartDateChange(e: Event) {
		const target = e.target as HTMLInputElement;
		const start = target.value;
		if (start && dateRange?.end) {
			onDateRangeChange({ start, end: dateRange.end });
		} else if (start) {
			onDateRangeChange({ start, end: new Date().toISOString().split('T')[0] });
		}
	}

	function handleEndDateChange(e: Event) {
		const target = e.target as HTMLInputElement;
		const end = target.value;
		if (end && dateRange?.start) {
			onDateRangeChange({ start: dateRange.start, end });
		}
	}

	function clearDateRange() {
		onDateRangeChange(null);
	}

	function hasActiveFilters() {
		return showPinnedOnly || importanceMin > 0 || dateRange !== null;
	}
</script>

<div class="memory-filters">
	<!-- Quick filters bar -->
	<div class="quick-filters">
		<div class="filter-group">
			<label class="filter-label">Type</label>
			<select class="type-select" value={selectedType} onchange={(e) => handleTypeSelect((e.target as HTMLSelectElement).value as MemoryType | 'all')}>
				{#each memoryTypes as type}
					<option value={type.value}>{type.label}</option>
				{/each}
			</select>
		</div>

		<button
			class="filter-btn"
			class:active={showPinnedOnly}
			onclick={onPinnedToggle}
			title={showPinnedOnly ? 'Show all memories' : 'Show pinned only'}
		>
			<svg xmlns="http://www.w3.org/2000/svg" fill={showPinnedOnly ? 'currentColor' : 'none'} viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" width="16" height="16">
				<path stroke-linecap="round" stroke-linejoin="round" d="M16.5 3.75V16.5L12 14.25 7.5 16.5V3.75m9 0H18A2.25 2.25 0 0 1 20.25 6v12A2.25 2.25 0 0 1 18 20.25H6A2.25 2.25 0 0 1 3.75 18V6A2.25 2.25 0 0 1 6 3.75h1.5m9 0h-9" />
			</svg>
			Pinned
		</button>

		<button
			class="filter-btn advanced-toggle"
			class:active={expanded || hasActiveFilters()}
			onclick={() => expanded = !expanded}
		>
			<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" width="16" height="16">
				<path stroke-linecap="round" stroke-linejoin="round" d="M10.5 6h9.75M10.5 6a1.5 1.5 0 1 1-3 0m3 0a1.5 1.5 0 1 0-3 0M3.75 6H7.5m3 12h9.75m-9.75 0a1.5 1.5 0 0 1-3 0m3 0a1.5 1.5 0 0 0-3 0m-3.75 0H7.5m9-6h3.75m-3.75 0a1.5 1.5 0 0 1-3 0m3 0a1.5 1.5 0 0 0-3 0m-9.75 0h9.75" />
			</svg>
			Advanced
			{#if hasActiveFilters()}
				<span class="filter-badge">{[showPinnedOnly, importanceMin > 0, dateRange !== null].filter(Boolean).length}</span>
			{/if}
		</button>
	</div>

	<!-- Advanced filters panel -->
	{#if expanded}
		<div class="advanced-filters">
			<div class="filter-row">
				<label class="filter-label">
					Minimum Importance: {importanceMin}%
				</label>
				<input
					type="range"
					min="0"
					max="100"
					step="10"
					value={importanceMin}
					oninput={handleImportanceInput}
					class="importance-slider"
				/>
				<div class="slider-marks">
					<span>0%</span>
					<span>50%</span>
					<span>100%</span>
				</div>
			</div>

			<div class="filter-row">
				<label class="filter-label">Date Range</label>
				<div class="date-inputs">
					<input
						type="date"
						value={dateRange?.start || ''}
						onchange={handleStartDateChange}
						class="date-input"
						placeholder="Start date"
					/>
					<span class="date-separator">to</span>
					<input
						type="date"
						value={dateRange?.end || ''}
						onchange={handleEndDateChange}
						class="date-input"
						placeholder="End date"
					/>
					{#if dateRange}
						<button class="btn-pill btn-pill-ghost clear-date-btn" onclick={clearDateRange} title="Clear date range">
							<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" width="14" height="14">
								<path stroke-linecap="round" stroke-linejoin="round" d="M6 18 18 6M6 6l12 12" />
							</svg>
						</button>
					{/if}
				</div>
			</div>
		</div>
	{/if}
</div>

<style>
	.memory-filters {
		display: flex;
		flex-direction: column;
		gap: 8px;
		padding: 12px 16px;
		border-bottom: 1px solid var(--color-border);
		background: var(--color-bg);
	}

	:global(.dark) .memory-filters {
		border-bottom-color: rgba(255, 255, 255, 0.06);
		background: var(--dbg);
	}

	.quick-filters {
		display: flex;
		align-items: center;
		gap: 8px;
		flex-wrap: wrap;
	}

	.filter-group {
		display: flex;
		align-items: center;
		gap: 6px;
	}

	.filter-label {
		font-size: 12px;
		font-weight: 500;
		color: var(--color-text-muted);
		white-space: nowrap;
	}

	:global(.dark) .filter-label {
		color: var(--dt3);
	}

	.type-select {
		padding: 6px 10px;
		font-size: 13px;
		border: 1px solid var(--color-border);
		background: var(--color-bg-secondary);
		color: var(--color-text);
		border-radius: 6px;
		cursor: pointer;
		outline: none;
		transition: all 0.15s ease;
	}

	.type-select:hover {
		border-color: var(--color-text-muted);
	}

	:global(.dark) .type-select {
		background: var(--dbg2);
		color: var(--dt);
		border-color: rgba(255, 255, 255, 0.1);
	}

	.filter-btn {
		display: flex;
		align-items: center;
		gap: 6px;
		padding: 6px 12px;
		font-size: 12px;
		font-weight: 500;
		color: var(--color-text-muted);
		background: var(--color-bg-secondary);
		border: 1px solid var(--color-border);
		border-radius: 6px;
		cursor: pointer;
		transition: all 0.15s ease;
		white-space: nowrap;
	}

	.filter-btn:hover {
		color: var(--color-text);
		border-color: var(--color-text-muted);
	}

	.filter-btn.active {
		color: var(--bos-primary-color);
		background: var(--bos-secondary-color);
		border-color: var(--bos-primary-color);
	}

	:global(.dark) .filter-btn {
		background: var(--dbg2);
		color: var(--dt3);
		border-color: rgba(255, 255, 255, 0.1);
	}

	:global(.dark) .filter-btn:hover {
		color: var(--dt);
		border-color: rgba(255, 255, 255, 0.2);
	}

	:global(.dark) .filter-btn.active {
		background: color-mix(in srgb, var(--bos-primary-color) 15%, transparent);
	}

	.advanced-toggle {
		margin-left: auto;
		position: relative;
	}

	.filter-badge {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 18px;
		height: 18px;
		font-size: 10px;
		font-weight: 700;
		color: white;
		background: var(--bos-primary-color);
		border-radius: 50%;
	}

	.advanced-filters {
		display: flex;
		flex-direction: column;
		gap: 16px;
		padding: 12px;
		background: var(--color-bg-secondary);
		border-radius: 8px;
		animation: slideDown 0.2s ease-out;
	}

	:global(.dark) .advanced-filters {
		background: var(--dbg2);
	}

	@keyframes slideDown {
		from {
			opacity: 0;
			transform: translateY(-8px);
		}
		to {
			opacity: 1;
			transform: translateY(0);
		}
	}

	.filter-row {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.importance-slider {
		width: 100%;
		height: 6px;
		border-radius: 3px;
		background: var(--color-border);
		outline: none;
		-webkit-appearance: none;
		cursor: pointer;
	}

	.importance-slider::-webkit-slider-thumb {
		-webkit-appearance: none;
		appearance: none;
		width: 16px;
		height: 16px;
		border-radius: 50%;
		background: var(--bos-primary-color);
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.importance-slider::-webkit-slider-thumb:hover {
		transform: scale(1.1);
	}

	.importance-slider::-moz-range-thumb {
		width: 16px;
		height: 16px;
		border-radius: 50%;
		background: var(--bos-primary-color);
		cursor: pointer;
		border: none;
		transition: all 0.15s ease;
	}

	.importance-slider::-moz-range-thumb:hover {
		transform: scale(1.1);
	}

	:global(.dark) .importance-slider {
		background: var(--dbg3);
	}

	.slider-marks {
		display: flex;
		justify-content: space-between;
		font-size: 10px;
		color: var(--color-text-muted);
		margin-top: -4px;
	}

	:global(.dark) .slider-marks {
		color: var(--dt3);
	}

	.date-inputs {
		display: flex;
		align-items: center;
		gap: 8px;
		flex-wrap: wrap;
	}

	.date-input {
		flex: 1;
		min-width: 120px;
		padding: 6px 10px;
		font-size: 13px;
		border: 1px solid var(--color-border);
		background: var(--color-bg);
		color: var(--color-text);
		border-radius: 6px;
		outline: none;
		transition: all 0.15s ease;
	}

	.date-input:focus {
		border-color: var(--bos-primary-color);
	}

	:global(.dark) .date-input {
		background: var(--dbg);
		color: var(--dt);
		border-color: rgba(255, 255, 255, 0.1);
	}

	.date-separator {
		font-size: 12px;
		color: var(--color-text-muted);
	}

	:global(.dark) .date-separator {
		color: var(--dt3);
	}

	.clear-date-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 6px;
		border: none;
		background: transparent;
		color: var(--color-text-muted);
		cursor: pointer;
		border-radius: 4px;
		transition: all 0.15s ease;
	}

	.clear-date-btn:hover {
		color: var(--color-text);
		background: var(--color-bg-secondary);
	}

	:global(.dark) .clear-date-btn:hover {
		background: var(--dbg3);
	}

	@media (max-width: 600px) {
		.filter-group {
			flex: 1;
			min-width: 140px;
		}

		.type-select {
			flex: 1;
		}
	}
</style>
