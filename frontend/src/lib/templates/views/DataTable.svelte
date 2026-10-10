<script lang="ts">
	/**
	 * DataTable - Main table view component for app templates
	 * Features: Sorting, selection, pagination, inline editing
	 */

	import type { Field, RecordData, User } from '../types';
	import type { TableViewConfig, SortConfig } from '../types';
	import TextCell from '../cells/TextCell.svelte';
	import NumberCell from '../cells/NumberCell.svelte';
	import CurrencyCell from '../cells/CurrencyCell.svelte';
	import DateCell from '../cells/DateCell.svelte';
	import StatusBadge from '../cells/StatusBadge.svelte';
	import EmailCell from '../cells/EmailCell.svelte';
	import PhoneCell from '../cells/PhoneCell.svelte';
	import URLCell from '../cells/URLCell.svelte';
	import CheckboxCell from '../cells/CheckboxCell.svelte';
	import RatingCell from '../cells/RatingCell.svelte';
	import ProgressCell from '../cells/ProgressCell.svelte';
	import UserCell from '../cells/UserCell.svelte';
	import MultiSelectCell from '../cells/MultiSelectCell.svelte';

	interface Props {
		fields: Field[];
		data: RecordData[];
		config?: Partial<TableViewConfig>;
		selectedIds?: string[];
		sort?: SortConfig[];
		loading?: boolean;
		onselect?: (ids: string[]) => void;
		onsort?: (sort: SortConfig[]) => void;
		onrowclick?: (record: RecordData) => void;
		oncelledit?: (recordId: string, fieldId: string, value: unknown) => void;
	}

	let {
		fields,
		data,
		config = {},
		selectedIds = [],
		sort = [],
		loading = false,
		onselect,
		onsort,
		onrowclick,
		oncelledit
	}: Props = $props();

	// Reactive config values
	const density = $derived(config.density ?? 'comfortable');
	const showRowNumbers = $derived(config.showRowNumbers ?? false);
	const showCheckboxes = $derived(config.showCheckboxes ?? true);
	const frozenColumns = $derived(config.frozenColumns ?? 0);
	const columnWidths = $derived(config.columnWidths ?? {});
	const columnOrder = $derived(config.columnOrder);
	const enableInlineEdit = $derived(config.enableInlineEdit ?? true);
	const stripedRows = $derived(config.stripedRows ?? false);

	// Computed visible fields
	const visibleFields = $derived(() => {
		let ordered = columnOrder
			? columnOrder.map((id) => fields.find((f) => f.id === id)).filter((f): f is Field => !!f)
			: fields;
		return ordered.filter((f) => !f.hidden);
	});

	// Selection state
	const allSelected = $derived(data.length > 0 && selectedIds.length === data.length);
	const someSelected = $derived(selectedIds.length > 0 && selectedIds.length < data.length);

	// Row height based on density
	const rowHeights: Record<string, string> = {
		compact: 'var(--tpl-table-row-height-compact)',
		comfortable: 'var(--tpl-table-row-height)',
		spacious: 'var(--tpl-table-row-height-comfortable)'
	};

	function getRecordId(record: RecordData): string {
		return String(record.id ?? record._id ?? Math.random());
	}

	function isSelected(record: RecordData): boolean {
		return selectedIds.includes(getRecordId(record));
	}

	function toggleSelectAll() {
		if (allSelected) {
			onselect?.([]);
		} else {
			onselect?.(data.map(getRecordId));
		}
	}

	function toggleSelect(record: RecordData) {
		const id = getRecordId(record);
		if (isSelected(record)) {
			onselect?.(selectedIds.filter((s) => s !== id));
		} else {
			onselect?.([...selectedIds, id]);
		}
	}

	function handleSort(fieldId: string) {
		const existing = sort.find((s) => s.fieldId === fieldId);
		let newSort: SortConfig[];

		if (!existing) {
			newSort = [{ fieldId, direction: 'asc' }];
		} else if (existing.direction === 'asc') {
			newSort = [{ fieldId, direction: 'desc' }];
		} else {
			newSort = [];
		}

		onsort?.(newSort);
	}

	function getSortDirection(fieldId: string): 'asc' | 'desc' | null {
		const s = sort.find((s) => s.fieldId === fieldId);
		return s?.direction ?? null;
	}

	function handleCellEdit(record: RecordData, fieldId: string, value: unknown) {
		oncelledit?.(getRecordId(record), fieldId, value);
	}

	function asString(value: unknown): string | null {
		if (value == null) return null;
		return typeof value === 'string' ? value : String(value);
	}

	function asNumber(value: unknown): number | null {
		return typeof value === 'number' ? value : null;
	}

	function asBoolean(value: unknown): boolean | null {
		return typeof value === 'boolean' ? value : null;
	}

	function asDateValue(value: unknown): string | Date | null {
		return typeof value === 'string' || value instanceof Date ? value : null;
	}

	function asStringArray(value: unknown): string[] | null {
		return Array.isArray(value) ? value.filter((item): item is string => typeof item === 'string') : null;
	}

	function asUserValue(value: unknown): User | User[] | null {
		if (!value) return null;
		if (Array.isArray(value)) return value.filter(isUser);
		return isUser(value) ? value : null;
	}

	function isUser(value: unknown): value is User {
		if (!value || typeof value !== 'object') return false;
		const candidate = value as Record<string, unknown>;
		return typeof candidate.id === 'string' && typeof candidate.name === 'string';
	}

	function getColumnWidth(field: Field): string {
		if (columnWidths[field.id]) return `${columnWidths[field.id]}px`;
		if (field.width) return `${field.width}px`;
		if (field.minWidth) return `minmax(${field.minWidth}px, 1fr)`;
		return 'minmax(120px, 1fr)';
	}
</script>

<div class="tpl-table-container" style="--row-height: {rowHeights[density]}">
	<table class="tpl-table" class:tpl-table-striped={stripedRows}>
		<thead class="tpl-table-header">
			<tr>
				{#if showCheckboxes}
					<th class="tpl-table-th tpl-table-checkbox-col">
						<button
							type="button"
							class="tpl-table-checkbox"
							class:checked={allSelected}
							class:indeterminate={someSelected}
							onclick={toggleSelectAll}
							aria-label="Select all"
						>
							{#if allSelected}
								<svg viewBox="0 0 16 16" fill="currentColor">
									<path d="M12.207 4.793a1 1 0 010 1.414l-5 5a1 1 0 01-1.414 0l-2-2a1 1 0 011.414-1.414L6.5 9.086l4.293-4.293a1 1 0 011.414 0z" />
								</svg>
							{:else if someSelected}
								<svg viewBox="0 0 16 16" fill="currentColor">
									<path d="M4 8h8" stroke="currentColor" stroke-width="2" />
								</svg>
							{/if}
						</button>
					</th>
				{/if}
				{#if showRowNumbers}
					<th class="tpl-table-th tpl-table-row-num">#</th>
				{/if}
				{#each visibleFields() as field, i}
					<th
						class="tpl-table-th"
						class:tpl-table-frozen={i < frozenColumns}
						style="width: {getColumnWidth(field)}"
					>
						<button
							type="button"
							class="tpl-table-th-content"
							onclick={() => handleSort(field.id)}
						>
							<span class="tpl-table-th-label">{field.name}</span>
							{#if getSortDirection(field.id)}
								<span class="tpl-table-sort-icon">
									{#if getSortDirection(field.id) === 'asc'}
										<svg viewBox="0 0 20 20" fill="currentColor">
											<path fill-rule="evenodd" d="M14.707 12.707a1 1 0 01-1.414 0L10 9.414l-3.293 3.293a1 1 0 01-1.414-1.414l4-4a1 1 0 011.414 0l4 4a1 1 0 010 1.414z" clip-rule="evenodd" />
										</svg>
									{:else}
										<svg viewBox="0 0 20 20" fill="currentColor">
											<path fill-rule="evenodd" d="M5.293 7.293a1 1 0 011.414 0L10 10.586l3.293-3.293a1 1 0 111.414 1.414l-4 4a1 1 0 01-1.414 0l-4-4a1 1 0 010-1.414z" clip-rule="evenodd" />
										</svg>
									{/if}
								</span>
							{/if}
						</button>
					</th>
				{/each}
			</tr>
		</thead>
		<tbody class="tpl-table-body">
			{#if loading}
				{#each Array(5) as _}
					<tr class="tpl-table-row tpl-table-row-loading">
						{#if showCheckboxes}
							<td class="tpl-table-td"><div class="tpl-skeleton tpl-skeleton-checkbox"></div></td>
						{/if}
						{#if showRowNumbers}
							<td class="tpl-table-td"><div class="tpl-skeleton tpl-skeleton-num"></div></td>
						{/if}
						{#each visibleFields() as _}
							<td class="tpl-table-td"><div class="tpl-skeleton tpl-shimmer"></div></td>
						{/each}
					</tr>
				{/each}
			{:else if data.length === 0}
				<tr>
					<td colspan={visibleFields().length + (showCheckboxes ? 1 : 0) + (showRowNumbers ? 1 : 0)} class="tpl-table-empty">
						<div class="tpl-table-empty-content">
							<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
								<path d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4" />
							</svg>
							<p>No records found</p>
						</div>
					</td>
				</tr>
			{:else}
				{#each data as record, rowIndex}
					<tr
						class="tpl-table-row tpl-row-highlight"
						class:tpl-table-row-selected={isSelected(record)}
						onclick={() => onrowclick?.(record)}
					>
						{#if showCheckboxes}
							<td class="tpl-table-td tpl-table-checkbox-col">
								<button
									type="button"
									class="tpl-table-checkbox"
									class:checked={isSelected(record)}
									onclick={(e) => { e.stopPropagation(); toggleSelect(record); }}
								>
									{#if isSelected(record)}
										<svg viewBox="0 0 16 16" fill="currentColor">
											<path d="M12.207 4.793a1 1 0 010 1.414l-5 5a1 1 0 01-1.414 0l-2-2a1 1 0 011.414-1.414L6.5 9.086l4.293-4.293a1 1 0 011.414 0z" />
										</svg>
									{/if}
								</button>
							</td>
						{/if}
						{#if showRowNumbers}
							<td class="tpl-table-td tpl-table-row-num">{rowIndex + 1}</td>
						{/if}
						{#each visibleFields() as field, i}
							{@const value = record[field.id]}
							{@const editable = enableInlineEdit && !field.readonly}
							<td class="tpl-table-td" class:tpl-table-frozen={i < frozenColumns}>

								{#if field.type === 'text'}
									<TextCell
										value={asString(value)}
										{editable}
										onchange={(v) => handleCellEdit(record, field.id, v)}
									/>
								{:else if field.type === 'number'}
									<NumberCell
										value={asNumber(value)}
										{editable}
										precision={field.precision}
										format={field.format}
										prefix={field.prefix}
										suffix={field.suffix}
										onchange={(v) => handleCellEdit(record, field.id, v)}
									/>
								{:else if field.type === 'currency'}
									<CurrencyCell
										value={asNumber(value)}
										{editable}
										currency={field.currency}
										onchange={(v) => handleCellEdit(record, field.id, v)}
									/>
								{:else if field.type === 'date' || field.type === 'datetime'}
									<DateCell
										value={asDateValue(value)}
										{editable}
										includeTime={field.type === 'datetime'}
										onchange={(v) => handleCellEdit(record, field.id, v)}
									/>
								{:else if field.type === 'status'}
									<StatusBadge
										value={asString(value)}
										options={field.options}
										{editable}
										onchange={(v) => handleCellEdit(record, field.id, v)}
									/>
								{:else if field.type === 'email'}
									<EmailCell
										value={asString(value)}
										{editable}
										onchange={(v) => handleCellEdit(record, field.id, v)}
									/>
								{:else if field.type === 'phone'}
									<PhoneCell
										value={asString(value)}
										{editable}
										onchange={(v) => handleCellEdit(record, field.id, v)}
									/>
								{:else if field.type === 'url'}
									<URLCell
										value={asString(value)}
										{editable}
										onchange={(v) => handleCellEdit(record, field.id, v)}
									/>
								{:else if field.type === 'checkbox'}
									<CheckboxCell
										value={asBoolean(value)}
										{editable}
										onchange={(v) => handleCellEdit(record, field.id, v)}
									/>
								{:else if field.type === 'rating'}
									<RatingCell
										value={asNumber(value)}
										{editable}
										max={field.max}
										onchange={(v) => handleCellEdit(record, field.id, v)}
									/>
								{:else if field.type === 'progress'}
									<ProgressCell value={asNumber(value)} />
								{:else if field.type === 'user'}
									<UserCell value={asUserValue(value)} />
								{:else if field.type === 'select'}
									<StatusBadge
										value={asString(value)}
										options={field.options}
										{editable}
										onchange={(v) => handleCellEdit(record, field.id, v)}
									/>
								{:else if field.type === 'multiselect'}
									<MultiSelectCell
										value={asStringArray(value)}
										options={field.options}
										{editable}
										onchange={(v) => handleCellEdit(record, field.id, v)}
									/>
								{:else}
									<TextCell value={asString(value)} {editable} onchange={(v) => handleCellEdit(record, field.id, v)} />
								{/if}
							</td>
						{/each}
					</tr>
				{/each}
			{/if}
		</tbody>
	</table>
</div>

<style>
	.tpl-table-container {
		width: 100%;
		overflow-x: auto;
		border: 1px solid var(--tpl-table-border-color);
		border-radius: var(--tpl-radius-lg);
		background: var(--tpl-bg-primary);
	}

	.tpl-table {
		width: 100%;
		border-collapse: collapse;
		font-family: var(--tpl-font-sans);
	}

	.tpl-table-header {
		position: sticky;
		top: 0;
		z-index: 10;
		background: var(--tpl-table-header-bg);
	}

	.tpl-table-th {
		height: var(--tpl-table-header-height);
		padding: 0;
		text-align: left;
		font-size: var(--tpl-text-2xs);
		font-weight: var(--tpl-font-semibold);
		color: var(--tpl-text-muted);
		text-transform: uppercase;
		letter-spacing: 0.05em;
		border-bottom: 1px solid var(--tpl-table-border-color);
		white-space: nowrap;
	}

	.tpl-table-th-content {
		display: flex;
		align-items: center;
		gap: var(--tpl-space-1);
		width: 100%;
		height: 100%;
		padding: var(--tpl-table-cell-padding);
		background: transparent;
		border: none;
		font: inherit;
		color: inherit;
		text-transform: inherit;
		letter-spacing: inherit;
		cursor: pointer;
		transition: all var(--tpl-transition-fast);
	}

	.tpl-table-th-content:hover {
		background: var(--tpl-bg-hover);
		color: var(--tpl-text-primary);
	}

	.tpl-table-th-content:focus-visible {
		outline: none;
		box-shadow: inset var(--tpl-shadow-focus);
	}

	.tpl-table-sort-icon {
		display: flex;
		align-items: center;
		color: var(--tpl-accent-primary);
	}

	.tpl-table-sort-icon svg {
		width: var(--tpl-icon-xs);
		height: var(--tpl-icon-xs);
	}

	.tpl-table-row {
		height: var(--row-height);
		cursor: pointer;
		transition: background var(--tpl-transition-fast);
	}

	.tpl-table-row:hover {
		background: var(--tpl-table-row-hover);
	}

	.tpl-table-row-selected {
		background: var(--tpl-table-row-selected) !important;
	}

	.tpl-table-row-selected:hover {
		background: var(--tpl-table-row-selected-hover) !important;
	}

	.tpl-table-striped .tpl-table-row:nth-child(even):not(:hover):not(.tpl-table-row-selected) {
		background: var(--tpl-table-stripe-bg);
	}

	.tpl-table-td {
		height: var(--row-height);
		padding: 0;
		border-bottom: 1px solid var(--tpl-border-subtle);
		vertical-align: middle;
	}

	.tpl-table-checkbox-col {
		width: 40px;
		text-align: center;
	}

	.tpl-table-row-num {
		width: 48px;
		padding: var(--tpl-table-cell-padding);
		text-align: center;
		font-size: var(--tpl-text-xs);
		color: var(--tpl-text-muted);
	}

	.tpl-table-checkbox {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 16px;
		height: 16px;
		margin: 0 auto;
		padding: 0;
		background: var(--tpl-bg-primary);
		border: 1.5px solid var(--tpl-border-default);
		border-radius: var(--tpl-radius-xs);
		cursor: pointer;
		transition: all var(--tpl-transition-fast);
	}

	.tpl-table-checkbox:hover {
		border-color: var(--tpl-accent-primary);
	}

	.tpl-table-checkbox:focus-visible {
		outline: none;
		box-shadow: var(--tpl-shadow-focus);
	}

	.tpl-table-checkbox.checked {
		background: var(--tpl-accent-primary);
		border-color: var(--tpl-accent-primary);
		color: white;
	}

	.tpl-table-checkbox.indeterminate {
		background: var(--tpl-accent-primary);
		border-color: var(--tpl-accent-primary);
		color: white;
	}

	.tpl-table-checkbox svg {
		width: 10px;
		height: 10px;
	}

	.tpl-table-frozen {
		position: sticky;
		left: 0;
		background: inherit;
		z-index: 5;
	}

	.tpl-table-empty {
		height: 200px;
		text-align: center;
	}

	.tpl-table-empty-content {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		height: 100%;
		color: var(--tpl-text-muted);
	}

	.tpl-table-empty-content svg {
		width: 48px;
		height: 48px;
		margin-bottom: var(--tpl-space-3);
		opacity: 0.5;
	}

	.tpl-table-empty-content p {
		margin: 0;
		font-size: var(--tpl-text-sm);
	}

	/* Loading skeleton */
	.tpl-skeleton {
		height: 20px;
		background: var(--tpl-bg-tertiary);
		border-radius: var(--tpl-radius-sm);
		margin: var(--tpl-space-2) var(--tpl-space-3);
	}

	.tpl-skeleton-checkbox {
		width: 18px;
		height: 18px;
		margin: 0 auto;
		border-radius: var(--tpl-radius-sm);
	}

	.tpl-skeleton-num {
		width: 24px;
		margin: 0 auto;
	}
</style>
