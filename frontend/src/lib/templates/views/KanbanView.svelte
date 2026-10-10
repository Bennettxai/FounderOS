<script lang="ts">
	/**
	 * KanbanView - Kanban board view for app templates
	 */

	import type { Field, SelectOption } from '../types/field';
	import type { KanbanViewConfig } from '../types/view';
	import { TemplateCard, TemplateBadge, TemplateAvatar, TemplateSkeleton } from '../primitives';

	interface Props {
		config: KanbanViewConfig;
		fields: Field[];
		data: Record<string, unknown>[];
		loading?: boolean;
		selectedIds?: Set<string>;
		onselect?: (id: string, selected: boolean) => void;
		onrowclick?: (record: Record<string, unknown>) => void;
		ondrop?: (recordId: string, newColumnValue: string) => void;
	}

	let {
		config,
		fields,
		data,
		loading = false,
		selectedIds = new Set(),
		onselect,
		onrowclick,
		ondrop
	}: Props = $props();

	let draggedItem: string | null = $state(null);
	let dragOverColumn: string | null = $state(null);
	let collapsedColumns = $state(new Set<string>());

	$effect(() => {
		collapsedColumns = new Set(config.collapsedColumns || []);
	});

	// Get the group-by field
	const groupField = $derived(fields.find(f => f.id === config.groupByField));

	// Get column options from the status/select field
	const columns = $derived(() => {
		if (!groupField) return [];
		if (groupField.type === 'status' || groupField.type === 'select') {
			const options = getFieldOptions(groupField);
			if (options.length > 0) return options;
		}
		// Generate columns from unique values in data
		const uniqueValues = new Set<string>();
		data.forEach(record => {
			const value = record[config.groupByField];
			if (value !== null && value !== undefined) {
				uniqueValues.add(String(value));
			}
		});
		return Array.from(uniqueValues).map<SelectOption>(value => ({ value, label: value, color: 'gray' }));
	});

	// Group data by column
	const groupedData = $derived(() => {
		const groups: Record<string, Record<string, unknown>[]> = {};
		const cols = columns();
		cols.forEach((col) => {
			groups[col.value] = [];
		});
		// Add uncategorized column
		groups['_uncategorized'] = [];

		data.forEach(record => {
			const value = record[config.groupByField];
			const key = value !== null && value !== undefined ? String(value) : '_uncategorized';
			if (groups[key]) {
				groups[key].push(record);
			} else {
				groups['_uncategorized'].push(record);
			}
		});
		return groups;
	});

	function handleDragStart(e: DragEvent, recordId: string) {
		if (!config.allowDragDrop) return;
		draggedItem = recordId;
		if (e.dataTransfer) {
			e.dataTransfer.effectAllowed = 'move';
			e.dataTransfer.setData('text/plain', recordId);
		}
	}

	function handleDragOver(e: DragEvent, columnValue: string) {
		if (!config.allowDragDrop || !draggedItem) return;
		e.preventDefault();
		dragOverColumn = columnValue;
	}

	function handleDragLeave() {
		dragOverColumn = null;
	}

	function handleDrop(e: DragEvent, columnValue: string) {
		if (!config.allowDragDrop || !draggedItem) return;
		e.preventDefault();
		ondrop?.(draggedItem, columnValue);
		draggedItem = null;
		dragOverColumn = null;
	}

	function handleDragEnd() {
		draggedItem = null;
		dragOverColumn = null;
	}

	function toggleColumnCollapse(columnValue: string) {
		if (collapsedColumns.has(columnValue)) {
			collapsedColumns.delete(columnValue);
		} else {
			collapsedColumns.add(columnValue);
		}
		collapsedColumns = new Set(collapsedColumns);
	}

	function getFieldValue(record: Record<string, unknown>, fieldId: string): unknown {
		return record[fieldId];
	}

	function getColumnColor(value: string): string {
		if (config.columnColors?.[value]) {
			return config.columnColors[value];
		}
		const col = columns().find((c) => c.value === value);
		return col?.color || 'gray';
	}

	function getFieldLabel(field: Field): string {
		return ((field as Field & { label?: string }).label) ?? field.name ?? field.id;
	}

	function getFieldOptions(field: Field): SelectOption[] {
		if ('options' in field && Array.isArray(field.options)) {
			return field.options;
		}
		return ((field as Field & { config?: { options?: SelectOption[] } }).config?.options) ?? [];
	}

	function isWipLimitExceeded(columnValue: string): boolean {
		if (!config.wipLimits?.[columnValue]) return false;
		const grouped = groupedData();
		return grouped[columnValue].length > config.wipLimits[columnValue];
	}
</script>

<div class="tpl-kanban-view">
	{#if loading}
		<div class="tpl-kanban-loading">
			{#each Array(4) as _}
				<div class="tpl-kanban-column-skeleton">
					<TemplateSkeleton variant="text" width="60%" height="20px" />
					<TemplateSkeleton variant="rounded" height="100px" />
					<TemplateSkeleton variant="rounded" height="80px" />
					<TemplateSkeleton variant="rounded" height="90px" />
				</div>
			{/each}
		</div>
	{:else}
		<div class="tpl-kanban-columns">
			{#each columns() as column}
				{@const items = groupedData()[column.value] || []}
				{@const isCollapsed = collapsedColumns.has(column.value)}
				{@const isDragOver = dragOverColumn === column.value}
				{@const wipExceeded = isWipLimitExceeded(column.value)}

				<div
					class="tpl-kanban-column"
					class:tpl-kanban-column-collapsed={isCollapsed}
					class:tpl-kanban-column-drag-over={isDragOver}
					class:tpl-kanban-column-wip-exceeded={wipExceeded}
					role="list"
					ondragover={(e) => handleDragOver(e, column.value)}
					ondragleave={handleDragLeave}
					ondrop={(e) => handleDrop(e, column.value)}
				>
					<button
						class="tpl-kanban-column-header"
						onclick={() => toggleColumnCollapse(column.value)}
					>
						<span class="tpl-kanban-column-color" style:background={`var(--tpl-status-${getColumnColor(column.value)}, ${getColumnColor(column.value)})`}></span>
						<span class="tpl-kanban-column-title">{column.label}</span>
						{#if config.showColumnCounts}
							<span class="tpl-kanban-column-count">{items.length}</span>
						{/if}
						{#if config.wipLimits?.[column.value]}
							<span class="tpl-kanban-column-wip" class:tpl-kanban-wip-exceeded={wipExceeded}>
								/ {config.wipLimits[column.value]}
							</span>
						{/if}
						<svg class="tpl-kanban-collapse-icon" viewBox="0 0 20 20" fill="currentColor">
							<path fill-rule="evenodd" d="M5.293 7.293a1 1 0 011.414 0L10 10.586l3.293-3.293a1 1 0 111.414 1.414l-4 4a1 1 0 01-1.414 0l-4-4a1 1 0 010-1.414z" clip-rule="evenodd" />
						</svg>
					</button>

					{#if !isCollapsed}
						<div class="tpl-kanban-cards">
							{#each items as record}
								{@const id = String(record.id || record._id || '')}
								{@const isSelected = selectedIds.has(id)}
								{@const title = getFieldValue(record, config.titleField)}
								{@const subtitle = config.subtitleField ? getFieldValue(record, config.subtitleField) : null}

								<div
									class="tpl-kanban-card"
									class:tpl-kanban-card-selected={isSelected}
									class:tpl-kanban-card-dragging={draggedItem === id}
									draggable={config.allowDragDrop ? 'true' : 'false'}
									ondragstart={(e) => handleDragStart(e, id)}
									ondragend={handleDragEnd}
									onclick={() => onrowclick?.(record)}
									onkeydown={(e) => e.key === 'Enter' && onrowclick?.(record)}
									role="button"
									tabindex="0"
								>
									<div class="tpl-kanban-card-title">{title}</div>
									{#if subtitle}
										<div class="tpl-kanban-card-subtitle">{subtitle}</div>
									{/if}
									{#if config.cardFields}
										<div class="tpl-kanban-card-fields">
											{#each config.cardFields as fieldId}
												{@const field = fields.find(f => f.id === fieldId)}
												{@const value = getFieldValue(record, fieldId)}
												{#if field && value !== null && value !== undefined}
													<div class="tpl-kanban-card-field">
														<span class="tpl-kanban-card-field-label">{getFieldLabel(field)}:</span>
														<span class="tpl-kanban-card-field-value">{value}</span>
													</div>
												{/if}
											{/each}
										</div>
									{/if}
									{#if onselect}
										<input
											type="checkbox"
											class="tpl-kanban-card-checkbox"
											checked={isSelected}
											onchange={(e) => onselect?.(id, e.currentTarget.checked)}
											onclick={(e) => e.stopPropagation()}
										/>
									{/if}
								</div>
							{/each}
							{#if items.length === 0}
								<div class="tpl-kanban-empty">No items</div>
							{/if}
						</div>
					{/if}
				</div>
			{/each}
		</div>
	{/if}
</div>

<style>
	.tpl-kanban-view {
		height: 100%;
		overflow-x: auto;
		padding: var(--tpl-space-4);
	}

	.tpl-kanban-loading,
	.tpl-kanban-columns {
		display: flex;
		gap: var(--tpl-space-4);
		min-height: 400px;
	}

	.tpl-kanban-column-skeleton {
		flex: 0 0 280px;
		display: flex;
		flex-direction: column;
		gap: var(--tpl-space-3);
		padding: var(--tpl-space-3);
		background: var(--tpl-bg-secondary);
		border-radius: var(--tpl-radius-lg);
	}

	.tpl-kanban-column {
		flex: 0 0 280px;
		display: flex;
		flex-direction: column;
		background: var(--tpl-bg-secondary);
		border-radius: var(--tpl-radius-lg);
		max-height: calc(100vh - 200px);
		transition: all var(--tpl-transition-fast);
	}

	.tpl-kanban-column-collapsed {
		flex: 0 0 48px;
	}

	.tpl-kanban-column-drag-over {
		background: var(--tpl-bg-selected);
		border: 2px dashed var(--tpl-accent-primary);
	}

	.tpl-kanban-column-wip-exceeded {
		border: 2px solid var(--tpl-status-warning);
	}

	.tpl-kanban-column-header {
		display: flex;
		align-items: center;
		gap: var(--tpl-space-2);
		padding: var(--tpl-space-3);
		background: transparent;
		border: none;
		font-family: var(--tpl-font-sans);
		cursor: pointer;
		text-align: left;
		width: 100%;
		border-radius: var(--tpl-radius-lg) var(--tpl-radius-lg) 0 0;
		transition: background var(--tpl-transition-fast);
	}

	.tpl-kanban-column-header:hover {
		background: var(--tpl-bg-hover);
	}

	.tpl-kanban-column-color {
		width: 12px;
		height: 12px;
		border-radius: var(--tpl-radius-full);
		flex-shrink: 0;
	}

	.tpl-kanban-column-title {
		flex: 1;
		font-size: var(--tpl-text-sm);
		font-weight: var(--tpl-font-semibold);
		color: var(--tpl-text-primary);
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.tpl-kanban-column-count {
		font-size: var(--tpl-text-xs);
		font-weight: var(--tpl-font-medium);
		color: var(--tpl-text-muted);
		background: var(--tpl-bg-tertiary);
		padding: var(--tpl-space-0-5) var(--tpl-space-2);
		border-radius: var(--tpl-radius-full);
	}

	.tpl-kanban-column-wip {
		font-size: var(--tpl-text-xs);
		color: var(--tpl-text-muted);
	}

	.tpl-kanban-wip-exceeded {
		color: var(--tpl-status-warning);
		font-weight: var(--tpl-font-semibold);
	}

	.tpl-kanban-collapse-icon {
		width: 16px;
		height: 16px;
		color: var(--tpl-text-muted);
		transition: transform var(--tpl-transition-fast);
	}

	.tpl-kanban-column-collapsed .tpl-kanban-collapse-icon {
		transform: rotate(-90deg);
	}

	.tpl-kanban-cards {
		flex: 1;
		overflow-y: auto;
		padding: 0 var(--tpl-space-3) var(--tpl-space-3);
		display: flex;
		flex-direction: column;
		gap: var(--tpl-space-2);
	}

	.tpl-kanban-card {
		position: relative;
		background: var(--tpl-bg-primary);
		border: 1px solid var(--tpl-border-default);
		border-radius: var(--tpl-radius-md);
		padding: var(--tpl-space-3);
		cursor: pointer;
		transition: all var(--tpl-transition-fast);
	}

	.tpl-kanban-card:hover {
		border-color: var(--tpl-border-hover);
		box-shadow: var(--tpl-shadow-sm);
	}

	.tpl-kanban-card:focus-visible {
		outline: none;
		box-shadow: var(--tpl-shadow-focus);
	}

	.tpl-kanban-card-selected {
		border-color: var(--tpl-accent-primary);
		background: var(--tpl-bg-selected);
	}

	.tpl-kanban-card-dragging {
		opacity: 0.5;
		transform: rotate(2deg);
	}

	.tpl-kanban-card-title {
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-sm);
		font-weight: var(--tpl-font-medium);
		color: var(--tpl-text-primary);
		margin-bottom: var(--tpl-space-1);
	}

	.tpl-kanban-card-subtitle {
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-xs);
		color: var(--tpl-text-muted);
	}

	.tpl-kanban-card-fields {
		margin-top: var(--tpl-space-2);
		padding-top: var(--tpl-space-2);
		border-top: 1px solid var(--tpl-border-subtle);
		display: flex;
		flex-direction: column;
		gap: var(--tpl-space-1);
	}

	.tpl-kanban-card-field {
		display: flex;
		justify-content: space-between;
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-2xs);
	}

	.tpl-kanban-card-field-label {
		color: var(--tpl-text-muted);
	}

	.tpl-kanban-card-field-value {
		color: var(--tpl-text-secondary);
	}

	.tpl-kanban-card-checkbox {
		position: absolute;
		top: var(--tpl-space-2);
		right: var(--tpl-space-2);
		width: 16px;
		height: 16px;
		cursor: pointer;
		accent-color: var(--tpl-accent-primary);
	}

	.tpl-kanban-empty {
		padding: var(--tpl-space-4);
		text-align: center;
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-xs);
		color: var(--tpl-text-muted);
		font-style: italic;
	}
</style>
