<script lang="ts">
	/**
	 * TimelineView - Timeline/Gantt view for app templates
	 */

	import type { Field } from '../types/field';
	import type { TimelineViewConfig } from '../types/view';
	import { TemplateSkeleton, TemplateButton } from '../primitives';

	interface Props {
		config: TimelineViewConfig;
		fields: Field[];
		data: Record<string, unknown>[];
		loading?: boolean;
		onrowclick?: (record: Record<string, unknown>) => void;
	}

	let {
		config,
		fields,
		data,
		loading = false,
		onrowclick
	}: Props = $props();

	let viewStart = $state(new Date());
	let zoomLevel = $state<'day' | 'week' | 'month'>('week');

	// Calculate date range from data
	const dateRange = $derived(() => {
		let minDate = new Date();
		let maxDate = new Date();

		data.forEach(record => {
			const start = record[config.startDateField];
			const end = record[config.endDateField];
			if (start) {
				const startDate = new Date(start as string);
				if (startDate < minDate) minDate = startDate;
			}
			if (end) {
				const endDate = new Date(end as string);
				if (endDate > maxDate) maxDate = endDate;
			}
		});

		// Add padding
		minDate.setDate(minDate.getDate() - 7);
		maxDate.setDate(maxDate.getDate() + 7);

		return { min: minDate, max: maxDate };
	});

	// Generate time columns based on zoom level
	const timeColumns = $derived(() => {
		const columns: Date[] = [];
		const range = dateRange();
		const current = new Date(range.min);

		while (current <= range.max) {
			columns.push(new Date(current));
			if (zoomLevel === 'day') {
				current.setDate(current.getDate() + 1);
			} else if (zoomLevel === 'week') {
				current.setDate(current.getDate() + 7);
			} else {
				current.setMonth(current.getMonth() + 1);
			}
		}
		return columns;
	});

	// Group data by groupBy field if specified
	const groupedData = $derived(() => {
		if (!config.groupByField) {
			return { '_all': data };
		}

		const groups: Record<string, Record<string, unknown>[]> = {};
		data.forEach(record => {
			const groupValue = String(record[config.groupByField!] || '_ungrouped');
			if (!groups[groupValue]) {
				groups[groupValue] = [];
			}
			groups[groupValue].push(record);
		});
		return groups;
	});

	function getBarPosition(record: Record<string, unknown>): { left: string; width: string } {
		const range = dateRange();
		const totalDays = (range.max.getTime() - range.min.getTime()) / (1000 * 60 * 60 * 24);

		const start = new Date(record[config.startDateField] as string);
		const end = new Date(record[config.endDateField] as string);

		const startOffset = (start.getTime() - range.min.getTime()) / (1000 * 60 * 60 * 24);
		const duration = (end.getTime() - start.getTime()) / (1000 * 60 * 60 * 24);

		const leftPercent = (startOffset / totalDays) * 100;
		const widthPercent = (duration / totalDays) * 100;

		return {
			left: `${Math.max(0, leftPercent)}%`,
			width: `${Math.min(100, Math.max(2, widthPercent))}%`
		};
	}

	function getBarColor(record: Record<string, unknown>): string {
		if (!config.colorField) return 'var(--tpl-accent-primary)';
		const colorValue = record[config.colorField];
		if (!colorValue) return 'var(--tpl-accent-primary)';

		const field = fields.find(f => f.id === config.colorField);
		if (field?.type === 'status') {
			const option = getFieldOptions(field).find((o) => o.value === colorValue);
			if (option?.color) {
				return `var(--tpl-status-${option.color}, ${option.color})`;
			}
		}
		return String(colorValue);
	}

	function getFieldOptions(field: Field): { value: string; label: string; color?: string }[] {
		if ('options' in field && Array.isArray(field.options)) {
			return field.options;
		}
		return field.config?.options ?? [];
	}

	function formatColumnHeader(date: Date): string {
		if (zoomLevel === 'day') {
			return date.toLocaleDateString('en-US', { weekday: 'short', day: 'numeric' });
		} else if (zoomLevel === 'week') {
			return `W${Math.ceil(date.getDate() / 7)} ${date.toLocaleDateString('en-US', { month: 'short' })}`;
		} else {
			return date.toLocaleDateString('en-US', { month: 'short', year: '2-digit' });
		}
	}

	function navigateTime(direction: number) {
		const newStart = new Date(viewStart);
		if (zoomLevel === 'day') {
			newStart.setDate(newStart.getDate() + direction * 7);
		} else if (zoomLevel === 'week') {
			newStart.setDate(newStart.getDate() + direction * 28);
		} else {
			newStart.setMonth(newStart.getMonth() + direction * 3);
		}
		viewStart = newStart;
	}
</script>

<div class="tpl-timeline-view">
	<div class="tpl-timeline-header">
		<div class="tpl-timeline-nav">
			<TemplateButton variant="ghost" size="sm" onclick={() => navigateTime(-1)}>
				<svg viewBox="0 0 20 20" fill="currentColor" width="16" height="16">
					<path fill-rule="evenodd" d="M12.707 5.293a1 1 0 010 1.414L9.414 10l3.293 3.293a1 1 0 01-1.414 1.414l-4-4a1 1 0 010-1.414l4-4a1 1 0 011.414 0z" clip-rule="evenodd" />
				</svg>
			</TemplateButton>
			<TemplateButton variant="outline" size="sm" onclick={() => viewStart = new Date()}>Today</TemplateButton>
			<TemplateButton variant="ghost" size="sm" onclick={() => navigateTime(1)}>
				<svg viewBox="0 0 20 20" fill="currentColor" width="16" height="16">
					<path fill-rule="evenodd" d="M7.293 14.707a1 1 0 010-1.414L10.586 10 7.293 6.707a1 1 0 011.414-1.414l4 4a1 1 0 010 1.414l-4 4a1 1 0 01-1.414 0z" clip-rule="evenodd" />
				</svg>
			</TemplateButton>
		</div>
		<div class="tpl-timeline-zoom">
			<span class="tpl-timeline-zoom-label">Zoom:</span>
			<div class="tpl-timeline-zoom-buttons">
				<button
					class="tpl-timeline-zoom-btn"
					class:tpl-timeline-zoom-btn-active={zoomLevel === 'day'}
					onclick={() => zoomLevel = 'day'}
				>Day</button>
				<button
					class="tpl-timeline-zoom-btn"
					class:tpl-timeline-zoom-btn-active={zoomLevel === 'week'}
					onclick={() => zoomLevel = 'week'}
				>Week</button>
				<button
					class="tpl-timeline-zoom-btn"
					class:tpl-timeline-zoom-btn-active={zoomLevel === 'month'}
					onclick={() => zoomLevel = 'month'}
				>Month</button>
			</div>
		</div>
	</div>

	{#if loading}
		<div class="tpl-timeline-loading">
			<TemplateSkeleton variant="rectangular" width="100%" height="300px" />
		</div>
	{:else}
		<div class="tpl-timeline-container">
			<div class="tpl-timeline-sidebar">
				<div class="tpl-timeline-sidebar-header">Tasks</div>
				{#each Object.entries(groupedData()) as [group, items]}
					{#if config.groupByField && group !== '_all'}
						<div class="tpl-timeline-group-header">{group === '_ungrouped' ? 'Ungrouped' : group}</div>
					{/if}
					{#each items as record}
						<div class="tpl-timeline-row-label">
							{record[config.titleField]}
						</div>
					{/each}
				{/each}
			</div>
			<div class="tpl-timeline-chart">
				<div class="tpl-timeline-time-header">
					{#each timeColumns() as column}
						<div class="tpl-timeline-time-column">
							{formatColumnHeader(column)}
						</div>
					{/each}
				</div>
				<div class="tpl-timeline-rows">
					{#each Object.entries(groupedData()) as [group, items]}
						{#if config.groupByField && group !== '_all'}
							<div class="tpl-timeline-group-row"></div>
						{/if}
						{#each items as record}
							{@const position = getBarPosition(record)}
							{@const color = getBarColor(record)}
							<div class="tpl-timeline-row">
								<div class="tpl-timeline-grid">
									{#each timeColumns() as _}
										<div class="tpl-timeline-grid-cell"></div>
									{/each}
								</div>
								<button
									class="tpl-timeline-bar"
									style:left={position.left}
									style:width={position.width}
									style:background={color}
									onclick={() => onrowclick?.(record)}
								>
									<span class="tpl-timeline-bar-label">{record[config.titleField]}</span>
								</button>
							</div>
						{/each}
					{/each}
				</div>
			</div>
		</div>
	{/if}
</div>

<style>
	.tpl-timeline-view {
		display: flex;
		flex-direction: column;
		height: 100%;
		background: var(--tpl-bg-primary);
	}

	.tpl-timeline-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: var(--tpl-space-4);
		border-bottom: 1px solid var(--tpl-border-default);
	}

	.tpl-timeline-nav {
		display: flex;
		align-items: center;
		gap: var(--tpl-space-2);
	}

	.tpl-timeline-zoom {
		display: flex;
		align-items: center;
		gap: var(--tpl-space-2);
	}

	.tpl-timeline-zoom-label {
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-sm);
		color: var(--tpl-text-muted);
	}

	.tpl-timeline-zoom-buttons {
		display: flex;
		background: var(--tpl-bg-secondary);
		border-radius: var(--tpl-radius-md);
		padding: 2px;
	}

	.tpl-timeline-zoom-btn {
		padding: var(--tpl-space-1-5) var(--tpl-space-3);
		background: transparent;
		border: none;
		border-radius: var(--tpl-radius-sm);
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-xs);
		font-weight: var(--tpl-font-medium);
		color: var(--tpl-text-secondary);
		cursor: pointer;
		transition: all var(--tpl-transition-fast);
	}

	.tpl-timeline-zoom-btn:hover {
		color: var(--tpl-text-primary);
	}

	.tpl-timeline-zoom-btn-active {
		background: var(--tpl-bg-primary);
		color: var(--tpl-text-primary);
		box-shadow: var(--tpl-shadow-xs);
	}

	.tpl-timeline-loading {
		padding: var(--tpl-space-4);
	}

	.tpl-timeline-container {
		flex: 1;
		display: flex;
		overflow: hidden;
	}

	.tpl-timeline-sidebar {
		width: 200px;
		flex-shrink: 0;
		border-right: 1px solid var(--tpl-border-default);
		background: var(--tpl-bg-secondary);
	}

	.tpl-timeline-sidebar-header {
		height: 40px;
		display: flex;
		align-items: center;
		padding: 0 var(--tpl-space-3);
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-xs);
		font-weight: var(--tpl-font-semibold);
		color: var(--tpl-text-muted);
		text-transform: uppercase;
		letter-spacing: var(--tpl-tracking-wide);
		border-bottom: 1px solid var(--tpl-border-default);
	}

	.tpl-timeline-group-header {
		height: 32px;
		display: flex;
		align-items: center;
		padding: 0 var(--tpl-space-3);
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-xs);
		font-weight: var(--tpl-font-semibold);
		color: var(--tpl-text-secondary);
		background: var(--tpl-bg-tertiary);
	}

	.tpl-timeline-row-label {
		height: 40px;
		display: flex;
		align-items: center;
		padding: 0 var(--tpl-space-3);
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-sm);
		color: var(--tpl-text-primary);
		border-bottom: 1px solid var(--tpl-border-subtle);
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.tpl-timeline-chart {
		flex: 1;
		overflow-x: auto;
		display: flex;
		flex-direction: column;
	}

	.tpl-timeline-time-header {
		display: flex;
		height: 40px;
		border-bottom: 1px solid var(--tpl-border-default);
		background: var(--tpl-bg-secondary);
		position: sticky;
		top: 0;
	}

	.tpl-timeline-time-column {
		min-width: 80px;
		flex: 1;
		display: flex;
		align-items: center;
		justify-content: center;
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-xs);
		color: var(--tpl-text-muted);
		border-right: 1px solid var(--tpl-border-subtle);
	}

	.tpl-timeline-rows {
		flex: 1;
	}

	.tpl-timeline-group-row {
		height: 32px;
		background: var(--tpl-bg-tertiary);
	}

	.tpl-timeline-row {
		position: relative;
		height: 40px;
		border-bottom: 1px solid var(--tpl-border-subtle);
	}

	.tpl-timeline-grid {
		position: absolute;
		inset: 0;
		display: flex;
	}

	.tpl-timeline-grid-cell {
		min-width: 80px;
		flex: 1;
		border-right: 1px solid var(--tpl-border-subtle);
	}

	.tpl-timeline-bar {
		position: absolute;
		top: 6px;
		height: 28px;
		min-width: 24px;
		padding: 0 var(--tpl-space-2);
		background: var(--tpl-accent-primary);
		border: none;
		border-radius: var(--tpl-radius-sm);
		cursor: pointer;
		display: flex;
		align-items: center;
		overflow: hidden;
		transition: opacity var(--tpl-transition-fast);
	}

	.tpl-timeline-bar:hover {
		opacity: 0.9;
	}

	.tpl-timeline-bar-label {
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-xs);
		font-weight: var(--tpl-font-medium);
		color: white;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
</style>
