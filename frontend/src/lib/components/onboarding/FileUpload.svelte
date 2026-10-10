<!--
  FileUpload.svelte
  Drag and drop file upload for CSV/Excel/JSON
-->
<script lang="ts">
	interface Props {
		accept?: string;
		maxSizeMB?: number;
		multiple?: boolean;
		files?: File[];
		onFilesChange?: (files: File[]) => void;
		class?: string;
	}

	let {
		accept = '.csv,.xlsx,.xls,.json',
		maxSizeMB = 10,
		multiple = false,
		files = $bindable([]),
		onFilesChange,
		class: className = ''
	}: Props = $props();

	let isDragging = $state(false);
	let errorMessage = $state('');
	let fileInputRef: HTMLInputElement;

	function validateFile(file: File): string | null {
		const maxBytes = maxSizeMB * 1024 * 1024;
		if (file.size > maxBytes) {
			return `File "${file.name}" exceeds ${maxSizeMB}MB limit`;
		}
		return null;
	}

	function handleFiles(fileList: FileList | null) {
		if (!fileList) return;
		errorMessage = '';

		const newFiles: File[] = [];
		for (const file of fileList) {
			const error = validateFile(file);
			if (error) {
				errorMessage = error;
				return;
			}
			newFiles.push(file);
		}

		if (multiple) {
			files = [...files, ...newFiles];
		} else {
			files = newFiles.slice(0, 1);
		}
		onFilesChange?.(files);
	}

	function handleDrop(e: DragEvent) {
		e.preventDefault();
		isDragging = false;
		handleFiles(e.dataTransfer?.files ?? null);
	}

	function handleDragOver(e: DragEvent) {
		e.preventDefault();
		isDragging = true;
	}

	function handleDragLeave() {
		isDragging = false;
	}

	function handleInputChange(e: Event) {
		const target = e.target as HTMLInputElement;
		handleFiles(target.files);
		target.value = '';
	}

	function removeFile(index: number) {
		files = files.filter((_, i) => i !== index);
		onFilesChange?.(files);
	}

	function formatFileSize(bytes: number): string {
		if (bytes < 1024) return bytes + ' B';
		if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
		return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
	}
</script>

<div class="file-upload {className}">
	<div
		class="drop-zone"
		class:is-dragging={isDragging}
		class:has-files={files.length > 0}
		role="button"
		tabindex="0"
		ondrop={handleDrop}
		ondragover={handleDragOver}
		ondragleave={handleDragLeave}
		onclick={() => fileInputRef.click()}
		onkeydown={(e) => e.key === 'Enter' && fileInputRef.click()}
	>
		<input
			bind:this={fileInputRef}
			type="file"
			{accept}
			{multiple}
			onchange={handleInputChange}
			class="hidden-input"
		/>

		<div class="drop-content">
			<div class="icon">
				<svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
					<path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
					<polyline points="17 8 12 3 7 8" />
					<line x1="12" x2="12" y1="3" y2="15" />
				</svg>
			</div>
			<p class="title">
				{#if isDragging}
					Drop files here
				{:else}
					Drag & drop files or <span class="browse">browse</span>
				{/if}
			</p>
			<p class="hint">
				Supported: CSV, Excel, JSON (max {maxSizeMB}MB)
			</p>
		</div>
	</div>

	{#if errorMessage}
		<p class="error">{errorMessage}</p>
	{/if}

	{#if files.length > 0}
		<div class="file-list">
			{#each files as file, index (file.name + index)}
				<div class="file-item">
					<div class="file-info">
						<span class="file-name">{file.name}</span>
						<span class="file-size">{formatFileSize(file.size)}</span>
					</div>
					<button
						type="button"
						class="btn-pill btn-pill-ghost remove-btn"
						onclick={() => removeFile(index)}
						aria-label="Remove {file.name}"
					>
						<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
							<path d="M18 6 6 18" />
							<path d="m6 6 12 12" />
						</svg>
					</button>
				</div>
			{/each}
		</div>
	{/if}
</div>

<style>
	.file-upload {
		display: flex;
		flex-direction: column;
		gap: 12px;
	}

	.drop-zone {
		display: flex;
		align-items: center;
		justify-content: center;
		min-height: 180px;
		padding: 24px;
		border: 2px dashed var(--border, #e5e7eb);
		border-radius: 12px;
		background-color: var(--secondary, #f9fafb);
		cursor: pointer;
		transition: all 0.2s ease;
	}

	.drop-zone:hover {
		border-color: var(--primary, #000000);
		background-color: var(--accent, #f3f4f6);
	}

	.drop-zone.is-dragging {
		border-color: var(--primary, #000000);
		background-color: var(--accent, #f3f4f6);
	}

	.hidden-input {
		display: none;
	}

	.drop-content {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 8px;
		text-align: center;
	}

	.icon {
		color: var(--muted-foreground, #6b7280);
	}

	.title {
		font-size: 15px;
		color: var(--foreground, #1f2937);
		margin: 0;
	}

	.browse {
		color: var(--primary, #000000);
		font-weight: 500;
		text-decoration: underline;
	}

	.hint {
		font-size: 13px;
		color: var(--muted-foreground, #6b7280);
		margin: 0;
	}

	.error {
		font-size: 13px;
		color: var(--error, #ef4444);
		margin: 0;
	}

	.file-list {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.file-item {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 12px 16px;
		background-color: var(--background, #ffffff);
		border: 1px solid var(--border, #e5e7eb);
		border-radius: 8px;
	}

	.file-info {
		display: flex;
		flex-direction: column;
		gap: 2px;
	}

	.file-name {
		font-size: 14px;
		font-weight: 500;
		color: var(--foreground, #1f2937);
	}

	.file-size {
		font-size: 12px;
		color: var(--muted-foreground, #6b7280);
	}

	.remove-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 28px;
		height: 28px;
		padding: 0;
		border: none;
		border-radius: 50%;
		background-color: transparent;
		color: var(--muted-foreground, #6b7280);
		cursor: pointer;
		transition: all 0.2s ease;
	}

	.remove-btn:hover {
		background-color: var(--error, #ef4444);
		color: white;
	}

	/* Dark mode */
	:global(.dark) .drop-zone {
		background-color: var(--secondary, #1a1a1a);
		border-color: var(--border, #2a2a2a);
	}

	:global(.dark) .drop-zone:hover,
	:global(.dark) .drop-zone.is-dragging {
		border-color: var(--primary, #ffffff);
		background-color: var(--accent, #2a2a2a);
	}

	:global(.dark) .title {
		color: var(--foreground, #f9fafb);
	}

	:global(.dark) .browse {
		color: var(--primary, #ffffff);
	}

	:global(.dark) .file-item {
		background-color: var(--background, #0a0a0a);
		border-color: var(--border, #2a2a2a);
	}

	:global(.dark) .file-name {
		color: var(--foreground, #f9fafb);
	}
</style>
