<script lang="ts">
	/**
	 * ImportModal - CSV/Excel import wizard
	 * Features: File upload, preview, column mapping, type detection
	 */
	import {
		X,
		Upload,
		FileSpreadsheet,
		FileText,
		Check,
		AlertCircle,
		ChevronRight,
		ChevronLeft,
		Table2,
		RefreshCw,
		Trash2
	} from 'lucide-svelte';
	import type { ColumnType } from '$lib/api/tables/types';

	interface PreviewColumn {
		name: string;
		detectedType: ColumnType;
		selectedType: ColumnType;
		sampleValues: string[];
		include: boolean;
	}

	interface Props {
		isOpen: boolean;
		onClose: () => void;
		onImport: (data: ImportData) => void;
	}

	interface ImportData {
		tableName: string;
		columns: PreviewColumn[];
		data: Record<string, unknown>[];
		hasHeaderRow: boolean;
	}

	let { isOpen, onClose, onImport }: Props = $props();

	// Wizard state
	let step = $state(1);
	let file = $state<File | null>(null);
	let isDragging = $state(false);
	let isProcessing = $state(false);
	let error = $state<string | null>(null);

	// Import settings
	let tableName = $state('');
	let hasHeaderRow = $state(true);
	let delimiter = $state(',');

	// Preview data
	let previewColumns = $state<PreviewColumn[]>([]);
	let previewData = $state<string[][]>([]);
	let rawData = $state<string[][]>([]);

	const columnTypes: { value: ColumnType; label: string }[] = [
		{ value: 'text', label: 'Text' },
		{ value: 'long_text', label: 'Long Text' },
		{ value: 'number', label: 'Number' },
		{ value: 'currency', label: 'Currency' },
		{ value: 'percent', label: 'Percent' },
		{ value: 'date', label: 'Date' },
		{ value: 'datetime', label: 'Date & Time' },
		{ value: 'checkbox', label: 'Checkbox' },
		{ value: 'email', label: 'Email' },
		{ value: 'url', label: 'URL' },
		{ value: 'phone', label: 'Phone' },
		{ value: 'single_select', label: 'Single Select' }
	];

	// Handle file drop
	function handleDrop(e: DragEvent) {
		e.preventDefault();
		isDragging = false;
		const files = e.dataTransfer?.files;
		if (files && files.length > 0) {
			handleFile(files[0]);
		}
	}

	function handleDragOver(e: DragEvent) {
		e.preventDefault();
		isDragging = true;
	}

	function handleDragLeave() {
		isDragging = false;
	}

	function handleFileSelect(e: Event) {
		const input = e.target as HTMLInputElement;
		if (input.files && input.files.length > 0) {
			handleFile(input.files[0]);
		}
	}

	async function handleFile(selectedFile: File) {
		// Validate file type
		const validTypes = [
			'text/csv',
			'application/vnd.ms-excel',
			'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet'
		];
		const validExtensions = ['.csv', '.xls', '.xlsx'];

		const hasValidType = validTypes.includes(selectedFile.type);
		const hasValidExtension = validExtensions.some((ext) =>
			selectedFile.name.toLowerCase().endsWith(ext)
		);

		if (!hasValidType && !hasValidExtension) {
			error = 'Please upload a CSV or Excel file';
			return;
		}

		file = selectedFile;
		tableName = selectedFile.name.replace(/\.[^/.]+$/, '').replace(/[_-]/g, ' ');
		error = null;
		isProcessing = true;

		try {
			// For now, only handle CSV (Excel would need a library)
			if (selectedFile.name.toLowerCase().endsWith('.csv')) {
				const text = await selectedFile.text();
				parseCSV(text);
			} else {
				error = 'Excel files require additional processing. Please use CSV for now.';
				isProcessing = false;
				return;
			}

			step = 2;
		} catch (err) {
			error = 'Failed to read file. Please try again.';
		}
		isProcessing = false;
	}

	function parseCSV(text: string) {
		// Simple CSV parser (handles quoted values)
		const lines = text.split(/\r?\n/).filter((line) => line.trim());
		const parsedData: string[][] = [];

		for (const line of lines) {
			const row: string[] = [];
			let current = '';
			let inQuotes = false;

			for (let i = 0; i < line.length; i++) {
				const char = line[i];
				if (char === '"') {
					inQuotes = !inQuotes;
				} else if (char === delimiter && !inQuotes) {
					row.push(current.trim());
					current = '';
				} else {
					current += char;
				}
			}
			row.push(current.trim());
			parsedData.push(row);
		}

		rawData = parsedData;
		processData();
	}

	function processData() {
		if (rawData.length === 0) return;

		const headers = hasHeaderRow ? rawData[0] : rawData[0].map((_, i) => `Column ${i + 1}`);
		const dataRows = hasHeaderRow ? rawData.slice(1) : rawData;

		// Detect column types based on sample values
		previewColumns = headers.map((name, i) => {
			const sampleValues = dataRows.slice(0, 5).map((row) => row[i] || '');
			const detectedType = detectColumnType(sampleValues);
			return {
				name: name || `Column ${i + 1}`,
				detectedType,
				selectedType: detectedType,
				sampleValues,
				include: true
			};
		});

		previewData = dataRows.slice(0, 10);
	}

	function detectColumnType(values: string[]): ColumnType {
		const nonEmpty = values.filter((v) => v && v.trim());
		if (nonEmpty.length === 0) return 'text';

		// Check for numbers
		if (nonEmpty.every((v) => /^-?\d*\.?\d+$/.test(v.replace(/,/g, '')))) {
			// Check if likely currency
			if (nonEmpty.some((v) => /^\$|€|£/.test(v) || parseFloat(v.replace(/,/g, '')) > 100)) {
				return 'currency';
			}
			// Check if likely percent
			if (nonEmpty.every((v) => parseFloat(v) >= 0 && parseFloat(v) <= 100)) {
				return 'percent';
			}
			return 'number';
		}

		// Check for dates
		if (nonEmpty.every((v) => !isNaN(Date.parse(v)))) {
			// Check if has time component
			if (nonEmpty.some((v) => /\d{1,2}:\d{2}/.test(v))) {
				return 'datetime';
			}
			return 'date';
		}

		// Check for emails
		if (nonEmpty.every((v) => /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(v))) {
			return 'email';
		}

		// Check for URLs
		if (nonEmpty.every((v) => /^https?:\/\//i.test(v))) {
			return 'url';
		}

		// Check for booleans
		if (
			nonEmpty.every((v) =>
				['true', 'false', 'yes', 'no', '1', '0'].includes(v.toLowerCase())
			)
		) {
			return 'checkbox';
		}

		// Check if likely single select (few unique values)
		const unique = new Set(nonEmpty);
		if (unique.size <= 5 && nonEmpty.length >= 3) {
			return 'single_select';
		}

		// Default to text, or long_text if long
		if (nonEmpty.some((v) => v.length > 100)) {
			return 'long_text';
		}

		return 'text';
	}

	function handleImport() {
		const includedColumns = previewColumns.filter((c) => c.include);
		const dataRows = hasHeaderRow ? rawData.slice(1) : rawData;

		const mappedData = dataRows.map((row) => {
			const obj: Record<string, unknown> = {};
			includedColumns.forEach((col, i) => {
				const originalIndex = previewColumns.indexOf(col);
				let value: unknown = row[originalIndex] || null;

				// Type conversion
				if (value) {
					switch (col.selectedType) {
						case 'number':
						case 'currency':
						case 'percent':
							value = parseFloat(String(value).replace(/[,$%]/g, ''));
							break;
						case 'checkbox':
							value = ['true', 'yes', '1'].includes(String(value).toLowerCase());
							break;
						case 'date':
						case 'datetime':
							value = new Date(String(value)).toISOString();
							break;
					}
				}

				obj[col.name] = value;
			});
			return obj;
		});

		onImport({
			tableName,
			columns: includedColumns,
			data: mappedData,
			hasHeaderRow
		});
	}

	function reset() {
		step = 1;
		file = null;
		error = null;
		tableName = '';
		previewColumns = [];
		previewData = [];
		rawData = [];
	}

	function handleClose() {
		reset();
		onClose();
	}

	// Re-process when header setting changes
	$effect(() => {
		if (rawData.length > 0) {
			processData();
		}
	});
</script>

{#if isOpen}
	<div class="fixed inset-0 z-50 flex items-center justify-center" style="background: var(--bos-modal-backdrop);">
		<div
			class="flex max-h-[90vh] w-full max-w-4xl flex-col rounded-xl shadow-2xl"
			style="background: var(--bos-modal-bg);"
			onclick={(e) => e.stopPropagation()}
		>
			<!-- Header -->
			<div class="import-modal__header" style="border-bottom: 1px solid var(--dbd);">
				<div class="import-modal__title">
					<h2 class="text-lg font-semibold" style="color: var(--dt);">Import Data</h2>
					<p class="text-sm" style="color: var(--dt2);">
						{#if step === 1}
							Upload a CSV or Excel file
						{:else if step === 2}
							Configure columns and data types
						{:else}
							Review and confirm import
						{/if}
					</p>
				</div>

				<!-- Progress Steps (absolutely centered) -->
				<div class="import-modal__steps">
					{#each [1, 2, 3] as s}
						<div
							class="import-modal__step"
							style={s <= step
								? 'background: var(--bos-nav-active); color: var(--bos-surface-on-color);'
								: 'background: var(--dbg3); color: var(--dt3);'}
						>
							{#if s < step}
								<Check class="h-4 w-4" />
							{:else}
								{s}
							{/if}
						</div>
						{#if s < 3}
							<ChevronRight class="h-4 w-4" style="color: var(--dt4);" />
						{/if}
					{/each}
				</div>

				<button type="button" onclick={handleClose} class="btn-pill btn-pill-ghost btn-pill-icon">
					<X class="h-5 w-5" />
				</button>
			</div>

			<!-- Content -->
			<div class="flex-1 overflow-y-auto p-6">
				{#if step === 1}
					<!-- Step 1: File Upload -->
					<div
						class="flex flex-col items-center justify-center rounded-xl border-2 border-dashed p-12 transition-colors"
						style={isDragging
							? 'border-color: var(--bos-nav-active); background: var(--bos-nav-active-bg);'
							: 'border-color: var(--dbd);'}
						ondrop={handleDrop}
						ondragover={handleDragOver}
						ondragleave={handleDragLeave}
					>
						{#if isProcessing}
							<RefreshCw class="mb-4 h-12 w-12 animate-spin" style="color: var(--bos-nav-active);" />
							<p style="color: var(--dt2);">Processing file...</p>
						{:else if file}
							<div class="flex items-center gap-3">
								{#if file.name.endsWith('.csv')}
									<FileText class="h-12 w-12" style="color: var(--bos-status-success);" />
								{:else}
									<FileSpreadsheet class="h-12 w-12" style="color: var(--bos-status-success);" />
								{/if}
								<div>
									<p class="font-medium" style="color: var(--dt);">{file.name}</p>
									<p class="text-sm" style="color: var(--dt2);">
										{(file.size / 1024).toFixed(1)} KB
									</p>
								</div>
								<button
									type="button"
									class="btn-pill btn-pill-ghost btn-pill-icon"
									onclick={() => {
										file = null;
										error = null;
									}}
								>
									<Trash2 class="h-5 w-5" />
								</button>
							</div>
						{:else}
							<Upload class="mb-4 h-12 w-12" style="color: var(--dt3);" />
							<p class="mb-2" style="color: var(--dt2);">
								Drag and drop your file here, or
								<label class="cursor-pointer hover:underline" style="color: var(--bos-nav-active);">
									browse
									<input
										type="file"
										accept=".csv,.xls,.xlsx"
										class="hidden"
										onchange={handleFileSelect}
									/>
								</label>
							</p>
							<p class="text-sm" style="color: var(--dt3);">Supports CSV and Excel files</p>
						{/if}
					</div>

					{#if error}
						<div class="mt-4 flex items-center gap-2 rounded-lg px-4 py-3" style="background: var(--bos-status-error-bg); color: var(--bos-status-error-text);">
							<AlertCircle class="h-5 w-5 shrink-0" />
							<p class="text-sm">{error}</p>
						</div>
					{/if}
				{:else if step === 2}
					<!-- Step 2: Column Configuration -->
					<div class="space-y-6">
						<!-- Table Name -->
						<div>
							<label class="block text-sm font-medium" style="color: var(--dt2);">Table Name</label>
							<input
								type="text"
								bind:value={tableName}
								class="mt-1 w-full rounded-lg px-3 py-2 focus:outline-none"
								style="border: 1px solid var(--dbd); background: var(--dbg2); color: var(--dt);"
								placeholder="My Table"
							/>
						</div>

						<!-- Header Row Toggle -->
						<label class="flex items-center gap-3">
							<input
								type="checkbox"
								bind:checked={hasHeaderRow}
								class="h-4 w-4 rounded"
								style="border-color: var(--dbd); accent-color: var(--bos-brand-color);"
							/>
							<span class="text-sm" style="color: var(--dt2);">First row contains column headers</span>
						</label>

						<!-- Column Configuration Table -->
						<div class="overflow-x-auto rounded-lg" style="border: 1px solid var(--dbd);">
							<table class="w-full text-sm">
								<thead style="background: var(--dbg2);">
									<tr>
										<th class="w-10 px-4 py-3 text-left font-medium" style="color: var(--dt2);">
											<input
												type="checkbox"
												checked={previewColumns.every((c) => c.include)}
												onchange={(e) => {
													const checked = (e.target as HTMLInputElement).checked;
													previewColumns = previewColumns.map((c) => ({
														...c,
														include: checked
													}));
												}}
												class="h-4 w-4 rounded"
												style="border-color: var(--dbd); accent-color: var(--bos-brand-color);"
											/>
										</th>
										<th class="px-4 py-3 text-left font-medium" style="color: var(--dt2);">Column Name</th>
										<th class="px-4 py-3 text-left font-medium" style="color: var(--dt2);">Type</th>
										<th class="px-4 py-3 text-left font-medium" style="color: var(--dt2);">Sample Values</th>
									</tr>
								</thead>
								<tbody style="border-top: 1px solid var(--dbd);">
									{#each previewColumns as col, i}
										<tr
											style={col.include
												? 'border-bottom: 1px solid var(--dbd);'
												: 'border-bottom: 1px solid var(--dbd); background: var(--dbg2); opacity: 0.5;'}
										>
											<td class="px-4 py-3">
												<input
													type="checkbox"
													bind:checked={col.include}
													class="h-4 w-4 rounded"
													style="border-color: var(--dbd); accent-color: var(--bos-brand-color);"
												/>
											</td>
											<td class="px-4 py-3">
												<input
													type="text"
													bind:value={col.name}
													disabled={!col.include}
													class="w-full rounded px-2 py-1 text-sm focus:outline-none"
													style="border: 1px solid var(--dbd); background: var(--dbg2); color: var(--dt);"
												/>
											</td>
											<td class="px-4 py-3">
												<select
													bind:value={col.selectedType}
													disabled={!col.include}
													class="w-full rounded px-2 py-1 text-sm focus:outline-none"
													style="border: 1px solid var(--dbd); background: var(--dbg2); color: var(--dt);"
												>
													{#each columnTypes as type}
														<option value={type.value}>{type.label}</option>
													{/each}
												</select>
											</td>
											<td class="px-4 py-3">
												<div class="flex flex-wrap gap-1">
													{#each col.sampleValues.slice(0, 3) as val}
														{#if val}
															<span class="rounded px-2 py-0.5 text-xs" style="background: var(--dbg3); color: var(--dt2);">
																{val.length > 20 ? val.slice(0, 20) + '...' : val}
															</span>
														{/if}
													{/each}
												</div>
											</td>
										</tr>
									{/each}
								</tbody>
							</table>
						</div>
					</div>
				{:else}
					<!-- Step 3: Preview & Confirm -->
					<div class="space-y-6">
						<!-- Summary -->
						<div class="rounded-lg p-4" style="background: var(--bos-nav-active-bg);">
							<div class="flex items-center gap-3">
								<Table2 class="h-8 w-8" style="color: var(--bos-nav-active);" />
								<div>
									<h3 class="font-medium" style="color: var(--dt);">{tableName}</h3>
									<p class="text-sm" style="color: var(--dt2);">
										{previewColumns.filter((c) => c.include).length} columns,
										{rawData.length - (hasHeaderRow ? 1 : 0)} rows
									</p>
								</div>
							</div>
						</div>

						<!-- Data Preview -->
						<div>
							<h4 class="mb-2 text-sm font-medium" style="color: var(--dt2);">Data Preview</h4>
							<div class="overflow-x-auto rounded-lg" style="border: 1px solid var(--dbd);">
								<table class="w-full text-sm">
									<thead style="background: var(--dbg2);">
										<tr>
											{#each previewColumns.filter((c) => c.include) as col}
												<th class="whitespace-nowrap px-4 py-2 text-left font-medium" style="color: var(--dt2);">
													{col.name}
												</th>
											{/each}
										</tr>
									</thead>
									<tbody style="border-top: 1px solid var(--dbd);">
										{#each previewData.slice(0, 5) as row}
											<tr style="border-bottom: 1px solid var(--dbd);">
												{#each previewColumns.filter((c) => c.include) as col}
													{@const originalIndex = previewColumns.indexOf(col)}
													<td class="whitespace-nowrap px-4 py-2" style="color: var(--dt);">
														{row[originalIndex] || '-'}
													</td>
												{/each}
											</tr>
										{/each}
									</tbody>
								</table>
							</div>
							{#if previewData.length > 5}
								<p class="mt-2 text-center text-xs" style="color: var(--dt3);">
									Showing first 5 of {previewData.length} rows
								</p>
							{/if}
						</div>
					</div>
				{/if}
			</div>

			<!-- Footer -->
			<div class="flex items-center justify-between px-6 py-4" style="border-top: 1px solid var(--dbd);">
				<button
					type="button"
					class="btn-pill btn-pill-ghost btn-pill-sm"
					onclick={handleClose}
				>
					Cancel
				</button>

				<div class="flex items-center gap-3">
					{#if step > 1}
						<button
							type="button"
							class="btn-pill btn-pill-ghost btn-pill-sm flex items-center gap-2"
							onclick={() => step--}
						>
							<ChevronLeft class="h-4 w-4" />
							Back
						</button>
					{/if}

					{#if step < 3}
						<button
							type="button"
							class="btn-cta flex items-center gap-2"
							disabled={step === 1 && !file}
							onclick={() => step++}
						>
							Next
							<ChevronRight class="h-4 w-4" />
						</button>
					{:else}
						<button
							type="button"
							class="btn-cta flex items-center gap-2"
							onclick={handleImport}
						>
							<Upload class="h-4 w-4" />
							Import {rawData.length - (hasHeaderRow ? 1 : 0)} rows
						</button>
					{/if}
				</div>
			</div>
		</div>
	</div>
{/if}

<style>
	.import-modal__header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 1rem 1.5rem;
		position: relative;
	}
	.import-modal__title {
		flex-shrink: 0;
	}
	.import-modal__steps {
		position: absolute;
		left: 50%;
		transform: translateX(-50%);
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}
	.import-modal__step {
		display: flex;
		width: 2rem;
		height: 2rem;
		align-items: center;
		justify-content: center;
		border-radius: 9999px;
		font-size: 0.875rem;
		font-weight: 600;
		transition: background 0.15s, color 0.15s;
	}
</style>
