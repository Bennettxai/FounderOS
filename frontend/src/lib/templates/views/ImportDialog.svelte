<script lang="ts">
	/**
	 * ImportDialog - Import data UI for app templates
	 */

	import type { Field } from '../types/field';
	import { TemplateButton, TemplateModal, TemplateSelect, TemplateProgress } from '../primitives';

	interface Props {
		open?: boolean;
		fields: Field[];
		onimport?: (data: Record<string, unknown>[]) => void;
		onclose?: () => void;
	}

	let {
		open = $bindable(false),
		fields,
		onimport,
		onclose
	}: Props = $props();

	type ImportStep = 'upload' | 'mapping' | 'preview' | 'complete';

	let currentStep = $state<ImportStep>('upload');
	let file = $state<File | null>(null);
	let parsedData = $state<Record<string, unknown>[]>([]);
	let headers = $state<string[]>([]);
	let fieldMapping = $state<Record<string, string>>({});
	let importing = $state(false);
	let importProgress = $state(0);
	let dragOver = $state(false);

	function handleFileSelect(e: Event) {
		const input = e.target as HTMLInputElement;
		if (input.files?.[0]) {
			processFile(input.files[0]);
		}
	}

	function handleDrop(e: DragEvent) {
		e.preventDefault();
		dragOver = false;
		if (e.dataTransfer?.files?.[0]) {
			processFile(e.dataTransfer.files[0]);
		}
	}

	function handleDragOver(e: DragEvent) {
		e.preventDefault();
		dragOver = true;
	}

	function handleDragLeave() {
		dragOver = false;
	}

	function getFieldLabel(field: Field): string {
		return field.label ?? field.name ?? field.id;
	}

	async function processFile(f: File) {
		file = f;

		if (f.name.endsWith('.json')) {
			const text = await f.text();
			parsedData = JSON.parse(text);
			if (parsedData.length > 0) {
				headers = Object.keys(parsedData[0]);
			}
		} else if (f.name.endsWith('.csv')) {
			const text = await f.text();
			const lines = text.split('\n').filter(line => line.trim());
			headers = parseCSVLine(lines[0]);
			parsedData = lines.slice(1).map(line => {
				const values = parseCSVLine(line);
				const obj: Record<string, unknown> = {};
				headers.forEach((header, i) => {
					obj[header] = values[i] || '';
				});
				return obj;
			});
		}

		// Auto-map fields by matching names
		const mapping: Record<string, string> = {};
		headers.forEach(header => {
			const normalizedHeader = header.toLowerCase().replace(/[^a-z0-9]/g, '');
			const matchingField = fields.find(f => {
				const normalizedLabel = getFieldLabel(f).toLowerCase().replace(/[^a-z0-9]/g, '');
				const normalizedId = f.id.toLowerCase().replace(/[^a-z0-9]/g, '');
				return normalizedLabel === normalizedHeader || normalizedId === normalizedHeader;
			});
			if (matchingField) {
				mapping[header] = matchingField.id;
			}
		});
		fieldMapping = mapping;

		currentStep = 'mapping';
	}

	function parseCSVLine(line: string): string[] {
		const result: string[] = [];
		let current = '';
		let inQuotes = false;

		for (let i = 0; i < line.length; i++) {
			const char = line[i];
			if (char === '"') {
				if (inQuotes && line[i + 1] === '"') {
					current += '"';
					i++;
				} else {
					inQuotes = !inQuotes;
				}
			} else if (char === ',' && !inQuotes) {
				result.push(current.trim());
				current = '';
			} else {
				current += char;
			}
		}
		result.push(current.trim());
		return result;
	}

	function getMappedData(): Record<string, unknown>[] {
		return parsedData.map(row => {
			const mapped: Record<string, unknown> = {};
			Object.entries(fieldMapping).forEach(([source, target]) => {
				if (target && row[source] !== undefined) {
					mapped[target] = row[source];
				}
			});
			return mapped;
		});
	}

	async function handleImport() {
		importing = true;
		importProgress = 0;

		const mappedData = getMappedData();

		// Simulate import progress
		for (let i = 0; i <= 100; i += 10) {
			importProgress = i;
			await new Promise(resolve => setTimeout(resolve, 100));
		}

		onimport?.(mappedData);
		importing = false;
		currentStep = 'complete';
	}

	function reset() {
		currentStep = 'upload';
		file = null;
		parsedData = [];
		headers = [];
		fieldMapping = {};
		importing = false;
		importProgress = 0;
	}

	function close() {
		reset();
		open = false;
		onclose?.();
	}

	const mappedFieldCount = $derived(Object.values(fieldMapping).filter(Boolean).length);
	const previewData = $derived(getMappedData().slice(0, 5));
</script>

<TemplateModal bind:open onclose={close}>
	<div class="tpl-import-dialog">
		{#if currentStep === 'upload'}
			<div class="tpl-import-header">
				<h2 class="tpl-import-title">Import Data</h2>
				<p class="tpl-import-subtitle">Upload a CSV or JSON file to import records</p>
			</div>

			<div
				class="tpl-import-dropzone"
				class:tpl-import-dropzone-active={dragOver}
				role="button"
				tabindex="0"
				ondrop={handleDrop}
				ondragover={handleDragOver}
				ondragleave={handleDragLeave}
			>
				<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
					<path d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12" />
				</svg>
				<p>Drag and drop your file here, or</p>
				<label class="tpl-import-browse-btn">
					Browse files
					<input type="file" accept=".csv,.json" onchange={handleFileSelect} hidden />
				</label>
				<span class="tpl-import-formats">Supported formats: CSV, JSON</span>
			</div>

		{:else if currentStep === 'mapping'}
			<div class="tpl-import-header">
				<h2 class="tpl-import-title">Map Fields</h2>
				<p class="tpl-import-subtitle">
					{file?.name} - {parsedData.length} records found
				</p>
			</div>

			<div class="tpl-import-mapping">
				<div class="tpl-import-mapping-header">
					<span>Source Column</span>
					<span>Target Field</span>
				</div>
				{#each headers as header}
					<div class="tpl-import-mapping-row">
						<span class="tpl-import-mapping-source">{header}</span>
						<TemplateSelect
							options={[
								{ value: '', label: 'Skip this column' },
								...fields.map(f => ({ value: f.id, label: getFieldLabel(f) }))
							]}
							value={fieldMapping[header] || ''}
							size="sm"
							onchange={(e) => {
								fieldMapping[header] = (e.target as HTMLSelectElement).value;
								fieldMapping = { ...fieldMapping };
							}}
						/>
					</div>
				{/each}
			</div>

			<div class="tpl-import-footer">
				<div class="tpl-import-status">
					{mappedFieldCount} of {headers.length} columns mapped
				</div>
				<div class="tpl-import-actions">
					<TemplateButton variant="outline" onclick={() => { reset(); }}>Back</TemplateButton>
					<TemplateButton
						variant="primary"
						disabled={mappedFieldCount === 0}
						onclick={() => currentStep = 'preview'}
					>
						Preview
					</TemplateButton>
				</div>
			</div>

		{:else if currentStep === 'preview'}
			<div class="tpl-import-header">
				<h2 class="tpl-import-title">Preview Import</h2>
				<p class="tpl-import-subtitle">
					Review the first 5 records before importing
				</p>
			</div>

			<div class="tpl-import-preview">
				<table class="tpl-import-preview-table">
					<thead>
						<tr>
							{#each fields.filter(f => Object.values(fieldMapping).includes(f.id)) as field}
								<th>{getFieldLabel(field)}</th>
							{/each}
						</tr>
					</thead>
					<tbody>
						{#each previewData as row}
							<tr>
								{#each fields.filter(f => Object.values(fieldMapping).includes(f.id)) as field}
									<td>{row[field.id] ?? '-'}</td>
								{/each}
							</tr>
						{/each}
					</tbody>
				</table>
			</div>

			{#if importing}
				<div class="tpl-import-progress">
					<TemplateProgress value={importProgress} showValue label="Importing..." />
				</div>
			{/if}

			<div class="tpl-import-footer">
				<div class="tpl-import-status">
					{parsedData.length} records ready to import
				</div>
				<div class="tpl-import-actions">
					<TemplateButton variant="outline" onclick={() => currentStep = 'mapping'} disabled={importing}>
						Back
					</TemplateButton>
					<TemplateButton variant="primary" onclick={handleImport} disabled={importing}>
						{importing ? 'Importing...' : `Import ${parsedData.length} records`}
					</TemplateButton>
				</div>
			</div>

		{:else if currentStep === 'complete'}
			<div class="tpl-import-complete">
				<div class="tpl-import-complete-icon">
					<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<circle cx="12" cy="12" r="10" />
						<path d="M9 12l2 2 4-4" />
					</svg>
				</div>
				<h2 class="tpl-import-title">Import Complete</h2>
				<p class="tpl-import-subtitle">
					Successfully imported {parsedData.length} records
				</p>
				<TemplateButton variant="primary" onclick={close}>Done</TemplateButton>
			</div>
		{/if}
	</div>
</TemplateModal>

<style>
	.tpl-import-dialog {
		width: 560px;
		padding: var(--tpl-space-6);
	}

	.tpl-import-header {
		margin-bottom: var(--tpl-space-6);
	}

	.tpl-import-title {
		margin: 0 0 var(--tpl-space-1);
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-lg);
		font-weight: var(--tpl-font-semibold);
		color: var(--tpl-text-primary);
	}

	.tpl-import-subtitle {
		margin: 0;
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-sm);
		color: var(--tpl-text-muted);
	}

	.tpl-import-dropzone {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: var(--tpl-space-10);
		border: 2px dashed var(--tpl-border-default);
		border-radius: var(--tpl-radius-lg);
		background: var(--tpl-bg-secondary);
		text-align: center;
		transition: all var(--tpl-transition-fast);
	}

	.tpl-import-dropzone-active {
		border-color: var(--tpl-accent-primary);
		background: var(--tpl-accent-primary-light);
	}

	.tpl-import-dropzone svg {
		width: 48px;
		height: 48px;
		color: var(--tpl-text-muted);
		margin-bottom: var(--tpl-space-4);
	}

	.tpl-import-dropzone p {
		margin: 0 0 var(--tpl-space-3);
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-sm);
		color: var(--tpl-text-secondary);
	}

	.tpl-import-browse-btn {
		padding: var(--tpl-space-2) var(--tpl-space-4);
		background: var(--tpl-accent-primary);
		border-radius: var(--tpl-radius-md);
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-sm);
		font-weight: var(--tpl-font-medium);
		color: var(--tpl-text-inverted);
		cursor: pointer;
		transition: background var(--tpl-transition-fast);
	}

	.tpl-import-browse-btn:hover {
		background: var(--tpl-accent-primary-hover);
	}

	.tpl-import-formats {
		margin-top: var(--tpl-space-3);
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-xs);
		color: var(--tpl-text-muted);
	}

	.tpl-import-mapping {
		max-height: 300px;
		overflow-y: auto;
		border: 1px solid var(--tpl-border-default);
		border-radius: var(--tpl-radius-md);
	}

	.tpl-import-mapping-header {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: var(--tpl-space-4);
		padding: var(--tpl-space-3);
		background: var(--tpl-bg-secondary);
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-xs);
		font-weight: var(--tpl-font-semibold);
		color: var(--tpl-text-muted);
		text-transform: uppercase;
		letter-spacing: var(--tpl-tracking-wide);
		border-bottom: 1px solid var(--tpl-border-default);
		position: sticky;
		top: 0;
	}

	.tpl-import-mapping-row {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: var(--tpl-space-4);
		align-items: center;
		padding: var(--tpl-space-2) var(--tpl-space-3);
		border-bottom: 1px solid var(--tpl-border-subtle);
	}

	.tpl-import-mapping-source {
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-sm);
		color: var(--tpl-text-primary);
	}

	.tpl-import-preview {
		max-height: 250px;
		overflow: auto;
		border: 1px solid var(--tpl-border-default);
		border-radius: var(--tpl-radius-md);
	}

	.tpl-import-preview-table {
		width: 100%;
		border-collapse: collapse;
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-sm);
	}

	.tpl-import-preview-table th {
		padding: var(--tpl-space-2) var(--tpl-space-3);
		background: var(--tpl-bg-secondary);
		font-weight: var(--tpl-font-medium);
		color: var(--tpl-text-secondary);
		text-align: left;
		border-bottom: 1px solid var(--tpl-border-default);
		position: sticky;
		top: 0;
	}

	.tpl-import-preview-table td {
		padding: var(--tpl-space-2) var(--tpl-space-3);
		color: var(--tpl-text-primary);
		border-bottom: 1px solid var(--tpl-border-subtle);
	}

	.tpl-import-progress {
		margin-top: var(--tpl-space-4);
	}

	.tpl-import-footer {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-top: var(--tpl-space-6);
		padding-top: var(--tpl-space-4);
		border-top: 1px solid var(--tpl-border-subtle);
	}

	.tpl-import-status {
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-sm);
		color: var(--tpl-text-muted);
	}

	.tpl-import-actions {
		display: flex;
		gap: var(--tpl-space-2);
	}

	.tpl-import-complete {
		display: flex;
		flex-direction: column;
		align-items: center;
		text-align: center;
		padding: var(--tpl-space-8) 0;
	}

	.tpl-import-complete-icon {
		width: 64px;
		height: 64px;
		color: var(--tpl-status-success);
		margin-bottom: var(--tpl-space-4);
	}

	.tpl-import-complete-icon svg {
		width: 100%;
		height: 100%;
	}

	.tpl-import-complete .tpl-import-title {
		margin-bottom: var(--tpl-space-1);
	}

	.tpl-import-complete .tpl-import-subtitle {
		margin-bottom: var(--tpl-space-6);
	}
</style>
