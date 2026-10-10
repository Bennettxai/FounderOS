<script lang="ts">
	/**
	 * ExportDialog - Export data UI for app templates
	 */

	import type { Field } from '../types/field';
	import { TemplateButton, TemplateModal, TemplateSwitch, TemplateSelect } from '../primitives';

	interface Props {
		open?: boolean;
		fields: Field[];
		data: Record<string, unknown>[];
		selectedIds?: Set<string>;
		onclose?: () => void;
	}

	let {
		open = $bindable(false),
		fields,
		data,
		selectedIds = new Set(),
		onclose
	}: Props = $props();

	let exportFormat = $state<'csv' | 'json' | 'xlsx'>('csv');
	let exportScope = $state<'all' | 'selected' | 'filtered'>('all');
	let selectedFields = $state<Set<string>>(new Set());
	let includeHeaders = $state(true);

	$effect(() => {
		selectedFields = new Set(fields.map(f => f.id));
	});

	function toggleField(fieldId: string) {
		if (selectedFields.has(fieldId)) {
			selectedFields.delete(fieldId);
		} else {
			selectedFields.add(fieldId);
		}
		selectedFields = new Set(selectedFields);
	}

	function selectAllFields() {
		selectedFields = new Set(fields.map(f => f.id));
	}

	function deselectAllFields() {
		selectedFields = new Set();
	}

	function getExportData(): Record<string, unknown>[] {
		if (exportScope === 'selected' && selectedIds.size > 0) {
			return data.filter(record => {
				const id = String(record.id || record._id || '');
				return selectedIds.has(id);
			});
		}
		return data;
	}

	function formatValue(value: unknown, field: Field): string {
		if (value === null || value === undefined) return '';

		switch (field.type) {
			case 'date':
			case 'datetime':
				return new Date(value as string).toISOString();
			case 'checkbox':
				return value ? 'true' : 'false';
			case 'multiselect':
				return Array.isArray(value) ? value.join(', ') : String(value);
			default:
				return String(value);
		}
	}

	function getFieldLabel(field: Field): string {
		return field.label ?? field.name ?? field.id;
	}

	function exportCSV() {
		const exportData = getExportData();
		const exportFields = fields.filter(f => selectedFields.has(f.id));

		let csv = '';

		if (includeHeaders) {
			csv += exportFields.map(f => `"${getFieldLabel(f).replace(/"/g, '""')}"`).join(',') + '\n';
		}

		exportData.forEach(record => {
			const row = exportFields.map(field => {
				const value = formatValue(record[field.id], field);
				return `"${value.replace(/"/g, '""')}"`;
			});
			csv += row.join(',') + '\n';
		});

		downloadFile(csv, 'export.csv', 'text/csv');
	}

	function exportJSON() {
		const exportData = getExportData();
		const exportFields = fields.filter(f => selectedFields.has(f.id));

		const jsonData = exportData.map(record => {
			const obj: Record<string, unknown> = {};
			exportFields.forEach(field => {
				obj[field.id] = record[field.id];
			});
			return obj;
		});

		downloadFile(JSON.stringify(jsonData, null, 2), 'export.json', 'application/json');
	}

	function downloadFile(content: string, filename: string, mimeType: string) {
		const blob = new Blob([content], { type: mimeType });
		const url = URL.createObjectURL(blob);
		const a = document.createElement('a');
		a.href = url;
		a.download = filename;
		document.body.appendChild(a);
		a.click();
		document.body.removeChild(a);
		URL.revokeObjectURL(url);
		open = false;
		onclose?.();
	}

	function handleExport() {
		if (exportFormat === 'csv') {
			exportCSV();
		} else if (exportFormat === 'json') {
			exportJSON();
		} else {
			// XLSX would need a library like SheetJS
			alert('Excel export requires additional setup');
		}
	}

	const recordCount = $derived(
		exportScope === 'selected' ? selectedIds.size : data.length
	);
</script>

<TemplateModal bind:open onclose={() => onclose?.()}>
	<div class="tpl-export-dialog">
		<div class="tpl-export-header">
			<h2 class="tpl-export-title">Export Data</h2>
			<p class="tpl-export-subtitle">{recordCount} record{recordCount !== 1 ? 's' : ''} will be exported</p>
		</div>

		<div class="tpl-export-section">
			<h3 class="tpl-export-section-title">Format</h3>
			<div class="tpl-export-format-options">
				<button
					class="tpl-export-format-btn"
					class:tpl-export-format-btn-active={exportFormat === 'csv'}
					onclick={() => exportFormat = 'csv'}
				>
					<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
						<polyline points="14,2 14,8 20,8" />
						<line x1="8" y1="13" x2="16" y2="13" />
						<line x1="8" y1="17" x2="16" y2="17" />
					</svg>
					<span>CSV</span>
				</button>
				<button
					class="tpl-export-format-btn"
					class:tpl-export-format-btn-active={exportFormat === 'json'}
					onclick={() => exportFormat = 'json'}
				>
					<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
						<polyline points="14,2 14,8 20,8" />
						<path d="M8 13h2" />
						<path d="M8 17h2" />
					</svg>
					<span>JSON</span>
				</button>
				<button
					class="tpl-export-format-btn"
					class:tpl-export-format-btn-active={exportFormat === 'xlsx'}
					onclick={() => exportFormat = 'xlsx'}
				>
					<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
						<polyline points="14,2 14,8 20,8" />
						<path d="M8 13h8" />
						<path d="M8 17h8" />
						<path d="M12 13v8" />
					</svg>
					<span>Excel</span>
				</button>
			</div>
		</div>

		<div class="tpl-export-section">
			<h3 class="tpl-export-section-title">Scope</h3>
			<TemplateSelect
				options={[
					{ value: 'all', label: `All records (${data.length})` },
					{ value: 'selected', label: `Selected only (${selectedIds.size})`, disabled: selectedIds.size === 0 }
				]}
				value={exportScope}
				onchange={(e) => exportScope = (e.target as HTMLSelectElement).value as 'all' | 'selected'}
			/>
		</div>

		<div class="tpl-export-section">
			<div class="tpl-export-section-header">
				<h3 class="tpl-export-section-title">Fields</h3>
				<div class="tpl-export-field-actions">
					<button class="tpl-export-link-btn" onclick={selectAllFields}>Select all</button>
					<button class="tpl-export-link-btn" onclick={deselectAllFields}>Deselect all</button>
				</div>
			</div>
			<div class="tpl-export-fields">
				{#each fields as field}
					<label class="tpl-export-field">
						<input
							type="checkbox"
							checked={selectedFields.has(field.id)}
							onchange={() => toggleField(field.id)}
						/>
						<span>{getFieldLabel(field)}</span>
					</label>
				{/each}
			</div>
		</div>

		{#if exportFormat === 'csv'}
			<div class="tpl-export-section">
				<TemplateSwitch
					bind:checked={includeHeaders}
					label="Include column headers"
				/>
			</div>
		{/if}

		<div class="tpl-export-footer">
			<TemplateButton variant="outline" onclick={() => { open = false; onclose?.(); }}>
				Cancel
			</TemplateButton>
			<TemplateButton
				variant="primary"
				disabled={selectedFields.size === 0}
				onclick={handleExport}
			>
				Export {recordCount} record{recordCount !== 1 ? 's' : ''}
			</TemplateButton>
		</div>
	</div>
</TemplateModal>

<style>
	.tpl-export-dialog {
		width: 480px;
		padding: var(--tpl-space-6);
	}

	.tpl-export-header {
		margin-bottom: var(--tpl-space-6);
	}

	.tpl-export-title {
		margin: 0 0 var(--tpl-space-1);
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-lg);
		font-weight: var(--tpl-font-semibold);
		color: var(--tpl-text-primary);
	}

	.tpl-export-subtitle {
		margin: 0;
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-sm);
		color: var(--tpl-text-muted);
	}

	.tpl-export-section {
		margin-bottom: var(--tpl-space-5);
	}

	.tpl-export-section-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-bottom: var(--tpl-space-2);
	}

	.tpl-export-section-title {
		margin: 0 0 var(--tpl-space-2);
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-sm);
		font-weight: var(--tpl-font-medium);
		color: var(--tpl-text-primary);
	}

	.tpl-export-section-header .tpl-export-section-title {
		margin-bottom: 0;
	}

	.tpl-export-format-options {
		display: flex;
		gap: var(--tpl-space-2);
	}

	.tpl-export-format-btn {
		flex: 1;
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: var(--tpl-space-2);
		padding: var(--tpl-space-4);
		background: var(--tpl-bg-secondary);
		border: 2px solid var(--tpl-border-default);
		border-radius: var(--tpl-radius-lg);
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-sm);
		font-weight: var(--tpl-font-medium);
		color: var(--tpl-text-secondary);
		cursor: pointer;
		transition: all var(--tpl-transition-fast);
	}

	.tpl-export-format-btn:hover {
		border-color: var(--tpl-border-hover);
		background: var(--tpl-bg-tertiary);
	}

	.tpl-export-format-btn-active {
		border-color: var(--tpl-accent-primary);
		background: var(--tpl-accent-primary-light);
		color: var(--tpl-accent-primary);
	}

	.tpl-export-format-btn svg {
		width: 32px;
		height: 32px;
	}

	.tpl-export-field-actions {
		display: flex;
		gap: var(--tpl-space-3);
	}

	.tpl-export-link-btn {
		padding: 0;
		background: none;
		border: none;
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-xs);
		color: var(--tpl-text-link);
		cursor: pointer;
	}

	.tpl-export-link-btn:hover {
		text-decoration: underline;
	}

	.tpl-export-fields {
		display: grid;
		grid-template-columns: repeat(2, 1fr);
		gap: var(--tpl-space-2);
		max-height: 200px;
		overflow-y: auto;
		padding: var(--tpl-space-3);
		background: var(--tpl-bg-secondary);
		border-radius: var(--tpl-radius-md);
	}

	.tpl-export-field {
		display: flex;
		align-items: center;
		gap: var(--tpl-space-2);
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-sm);
		color: var(--tpl-text-primary);
		cursor: pointer;
	}

	.tpl-export-field input {
		accent-color: var(--tpl-accent-primary);
	}

	.tpl-export-footer {
		display: flex;
		justify-content: flex-end;
		gap: var(--tpl-space-2);
		padding-top: var(--tpl-space-4);
		border-top: 1px solid var(--tpl-border-subtle);
	}
</style>
