<script lang="ts">
	import { Dialog } from 'bits-ui';
	import { uploadDocument, type DocumentMetadata, type DocumentUploadResponse } from '$lib/api/pedro-documents';

	interface Props {
		open?: boolean;
		onClose?: () => void;
		onUploadComplete?: (doc: DocumentUploadResponse) => void;
	}

	let {
		open = $bindable(false),
		onClose,
		onUploadComplete
	}: Props = $props();

	let files = $state<File[]>([]);
	let uploading = $state(false);
	let uploadProgress = $state(0);
	let error = $state<string | null>(null);
	let dragOver = $state(false);

	// Metadata fields
	let documentTitle = $state('');
	let documentDescription = $state('');
	let selectedTags = $state<string[]>([]);
	let tagInput = $state('');

	const acceptedTypes = [
		'application/pdf',
		'text/plain',
		'text/markdown',
		'application/json',
		'text/csv',
		'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
		'application/msword'
	];

	const acceptedExtensions = ['.pdf', '.txt', '.md', '.json', '.csv', '.docx', '.doc'];

	function handleDragOver(e: DragEvent) {
		e.preventDefault();
		dragOver = true;
	}

	function handleDragLeave(e: DragEvent) {
		e.preventDefault();
		dragOver = false;
	}

	function handleDrop(e: DragEvent) {
		e.preventDefault();
		dragOver = false;

		const droppedFiles = e.dataTransfer?.files;
		if (droppedFiles) {
			addFiles(Array.from(droppedFiles));
		}
	}

	function handleFileSelect(e: Event) {
		const input = e.target as HTMLInputElement;
		if (input.files) {
			addFiles(Array.from(input.files));
		}
		input.value = '';
	}

	function addFiles(newFiles: File[]) {
		const validFiles = newFiles.filter(file => {
			const ext = '.' + file.name.split('.').pop()?.toLowerCase();
			return acceptedTypes.includes(file.type) || acceptedExtensions.includes(ext);
		});

		if (validFiles.length < newFiles.length) {
			error = 'Some files were skipped (unsupported format)';
			setTimeout(() => error = null, 3000);
		}

		files = [...files, ...validFiles];

		// Auto-fill title from first file if empty
		if (!documentTitle && files.length > 0) {
			documentTitle = files[0].name.replace(/\.[^/.]+$/, '');
		}
	}

	function removeFile(index: number) {
		files = files.filter((_, i) => i !== index);
	}

	function formatFileSize(bytes: number): string {
		if (bytes === 0) return '0 Bytes';
		const k = 1024;
		const sizes = ['Bytes', 'KB', 'MB', 'GB'];
		const i = Math.floor(Math.log(bytes) / Math.log(k));
		return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
	}

	function getFileIcon(file: File): string {
		const ext = file.name.split('.').pop()?.toLowerCase();
		switch (ext) {
			case 'pdf':
				return `<path stroke-linecap="round" stroke-linejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 0 0-3.375-3.375h-1.5A1.125 1.125 0 0 1 13.5 7.125v-1.5a3.375 3.375 0 0 0-3.375-3.375H8.25m0 12.75h7.5m-7.5 3H12M10.5 2.25H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 0 0-9-9Z" />`;
			case 'txt':
			case 'md':
				return `<path stroke-linecap="round" stroke-linejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 0 0-3.375-3.375h-1.5A1.125 1.125 0 0 1 13.5 7.125v-1.5a3.375 3.375 0 0 0-3.375-3.375H8.25m2.25 0H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 0 0-9-9Z" />`;
			case 'json':
			case 'csv':
				return `<path stroke-linecap="round" stroke-linejoin="round" d="M3.375 19.5h17.25m-17.25 0a1.125 1.125 0 0 1-1.125-1.125M3.375 19.5h7.5c.621 0 1.125-.504 1.125-1.125m-9.75 0V5.625m0 12.75v-1.5c0-.621.504-1.125 1.125-1.125m18.375 2.625V5.625m0 12.75c0 .621-.504 1.125-1.125 1.125m1.125-1.125v-1.5c0-.621-.504-1.125-1.125-1.125m0 3.75h-7.5A1.125 1.125 0 0 1 12 18.375m9.75-12.75c0-.621-.504-1.125-1.125-1.125H3.375c-.621 0-1.125.504-1.125 1.125m19.5 0v1.5c0 .621-.504 1.125-1.125 1.125M2.25 5.625v1.5c0 .621.504 1.125 1.125 1.125m0 0h17.25m-17.25 0h7.5c.621 0 1.125.504 1.125 1.125M3.375 8.25c-.621 0-1.125.504-1.125 1.125v1.5c0 .621.504 1.125 1.125 1.125m17.25-3.75h-7.5c-.621 0-1.125.504-1.125 1.125m8.625-1.125c.621 0 1.125.504 1.125 1.125v1.5c0 .621-.504 1.125-1.125 1.125m-17.25 0h7.5m-7.5 0c-.621 0-1.125.504-1.125 1.125v1.5c0 .621.504 1.125 1.125 1.125M12 10.875v-1.5m0 1.5c0 .621-.504 1.125-1.125 1.125M12 10.875c0 .621.504 1.125 1.125 1.125m-2.25 0c.621 0 1.125.504 1.125 1.125M13.125 12h7.5m-7.5 0c-.621 0-1.125.504-1.125 1.125M20.625 12c.621 0 1.125.504 1.125 1.125v1.5c0 .621-.504 1.125-1.125 1.125m-17.25 0h7.5M12 14.625v-1.5m0 1.5c0 .621-.504 1.125-1.125 1.125M12 14.625c0 .621.504 1.125 1.125 1.125m-2.25 0c.621 0 1.125.504 1.125 1.125m0 1.5v-1.5m0 0c0-.621.504-1.125 1.125-1.125m0 0h7.5" />`;
			default:
				return `<path stroke-linecap="round" stroke-linejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 0 0-3.375-3.375h-1.5A1.125 1.125 0 0 1 13.5 7.125v-1.5a3.375 3.375 0 0 0-3.375-3.375H8.25m2.25 0H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 0 0-9-9Z" />`;
		}
	}

	function addTag() {
		if (tagInput.trim() && !selectedTags.includes(tagInput.trim())) {
			selectedTags = [...selectedTags, tagInput.trim()];
			tagInput = '';
		}
	}

	function removeTag(tag: string) {
		selectedTags = selectedTags.filter(t => t !== tag);
	}

	function handleTagKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter') {
			e.preventDefault();
			addTag();
		}
	}

	async function handleUpload() {
		if (files.length === 0) return;

		uploading = true;
		uploadProgress = 0;
		error = null;

		try {
			for (let i = 0; i < files.length; i++) {
				const file = files[i];
				const metadata: DocumentMetadata = {
					title: files.length === 1 ? documentTitle || file.name : file.name,
					description: documentDescription || undefined,
					tags: selectedTags.length > 0 ? selectedTags : undefined
				};

				const result = await uploadDocument(file, metadata);
				uploadProgress = ((i + 1) / files.length) * 100;
				onUploadComplete?.(result);
			}

			resetForm();
			open = false;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Upload failed';
		} finally {
			uploading = false;
		}
	}

	function resetForm() {
		files = [];
		documentTitle = '';
		documentDescription = '';
		selectedTags = [];
		tagInput = '';
		uploadProgress = 0;
		error = null;
	}

	function handleClose() {
		resetForm();
		open = false;
		onClose?.();
	}
</script>

<Dialog.Root bind:open>
	<Dialog.Portal>
		<Dialog.Overlay
			class="fixed inset-0 bg-black/50 z-50 animate-in fade-in-0"
		/>
		<Dialog.Content
			class="fixed left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 z-50 w-full max-w-lg bg-white dark:bg-zinc-900 rounded-2xl shadow-xl animate-in fade-in-0 zoom-in-95"
		>
			<!-- Header -->
			<div class="flex items-center justify-between px-6 py-4 border-b border-gray-100 dark:border-zinc-800">
				<Dialog.Title class="text-lg font-semibold text-gray-900 dark:text-gray-100">Upload Documents</Dialog.Title>
				<Dialog.Close
					class="w-8 h-8 flex items-center justify-center rounded-lg hover:bg-gray-100 dark:hover:bg-zinc-800 transition-colors"
					onclick={handleClose}
				>
					<svg class="w-5 h-5 text-gray-500 dark:text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
					</svg>
				</Dialog.Close>
			</div>

			<!-- Body -->
			<div class="px-6 py-4 space-y-4 max-h-[60vh] overflow-y-auto">
				<!-- Drop Zone -->
				<div
					class="drop-zone"
					class:drag-over={dragOver}
					class:has-files={files.length > 0}
					ondragover={handleDragOver}
					ondragleave={handleDragLeave}
					ondrop={handleDrop}
					role="button"
					tabindex="0"
				>
					{#if files.length === 0}
						<div class="drop-content">
							<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="drop-icon">
								<path stroke-linecap="round" stroke-linejoin="round" d="M12 16.5V9.75m0 0 3 3m-3-3-3 3M6.75 19.5a4.5 4.5 0 0 1-1.41-8.775 5.25 5.25 0 0 1 10.233-2.33 3 3 0 0 1 3.758 3.848A3.752 3.752 0 0 1 18 19.5H6.75Z" />
							</svg>
							<p class="drop-text">Drag and drop files here</p>
							<p class="drop-hint">or</p>
							<label class="browse-btn">
								Browse Files
								<input
									type="file"
									multiple
									accept={acceptedExtensions.join(',')}
									onchange={handleFileSelect}
									class="hidden"
								/>
							</label>
							<p class="formats-hint">Supported: PDF, TXT, MD, JSON, CSV, DOCX</p>
						</div>
					{:else}
						<div class="files-list">
							{#each files as file, index}
								<div class="file-item">
									<div class="file-icon">
										<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" width="20" height="20">
											{@html getFileIcon(file)}
										</svg>
									</div>
									<div class="file-info">
										<span class="file-name">{file.name}</span>
										<span class="file-size">{formatFileSize(file.size)}</span>
									</div>
									<button
										class="remove-file-btn"
										onclick={() => removeFile(index)}
										disabled={uploading}
									>
										<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" width="16" height="16">
											<path stroke-linecap="round" stroke-linejoin="round" d="M6 18 18 6M6 6l12 12" />
										</svg>
									</button>
								</div>
							{/each}
							<label class="add-more-btn">
								<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" width="16" height="16">
									<path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
								</svg>
								Add more files
								<input
									type="file"
									multiple
									accept={acceptedExtensions.join(',')}
									onchange={handleFileSelect}
									class="hidden"
								/>
							</label>
						</div>
					{/if}
				</div>

				{#if files.length > 0}
					<!-- Document Title (for single file) -->
					{#if files.length === 1}
						<div>
							<label for="doc-title" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
								Document Title
							</label>
							<input
								id="doc-title"
								type="text"
								bind:value={documentTitle}
								placeholder={files[0]?.name || 'Enter title...'}
								class="w-full px-4 py-2.5 text-sm border border-gray-200 dark:border-zinc-700 dark:bg-zinc-800 dark:text-white rounded-xl focus:outline-none focus:ring-2 focus:ring-gray-900 dark:focus:ring-zinc-600 focus:border-transparent transition-all"
							/>
						</div>
					{/if}

					<!-- Description -->
					<div>
						<label for="doc-desc" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
							Description (optional)
						</label>
						<textarea
							id="doc-desc"
							bind:value={documentDescription}
							placeholder="Add a description for context..."
							rows={2}
							class="w-full px-4 py-2.5 text-sm border border-gray-200 dark:border-zinc-700 dark:bg-zinc-800 dark:text-white rounded-xl focus:outline-none focus:ring-2 focus:ring-gray-900 dark:focus:ring-zinc-600 focus:border-transparent transition-all resize-none"
						></textarea>
					</div>

					<!-- Tags -->
					<div>
						<label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Tags (optional)</label>
						<div class="flex flex-wrap gap-2 p-2 border border-gray-200 dark:border-zinc-700 dark:bg-zinc-800 rounded-xl min-h-[44px]">
							{#each selectedTags as tag}
								<span class="flex items-center gap-1 px-2 py-1 bg-gray-100 dark:bg-zinc-700 text-gray-700 dark:text-gray-300 text-sm rounded-lg">
									{tag}
									<button onclick={() => removeTag(tag)} class="hover:text-gray-900 dark:hover:text-white">
										<svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
										</svg>
									</button>
								</span>
							{/each}
							<input
								type="text"
								bind:value={tagInput}
								onkeydown={handleTagKeydown}
								placeholder={selectedTags.length === 0 ? '+ Add tags...' : ''}
								class="flex-1 min-w-[100px] px-2 py-1 text-sm focus:outline-none bg-transparent dark:text-white"
							/>
						</div>
					</div>
				{/if}

				<!-- Error message -->
				{#if error}
					<div class="error-message">
						<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" width="16" height="16">
							<path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m9-.75a9 9 0 1 1-18 0 9 9 0 0 1 18 0Zm-9 3.75h.008v.008H12v-.008Z" />
						</svg>
						{error}
					</div>
				{/if}

				<!-- Upload Progress -->
				{#if uploading}
					<div class="upload-progress">
						<div class="progress-bar">
							<div class="progress-fill" style="width: {uploadProgress}%"></div>
						</div>
						<span class="progress-text">{Math.round(uploadProgress)}%</span>
					</div>
				{/if}
			</div>

			<!-- Footer -->
			<div class="flex items-center justify-end gap-3 px-6 py-4 border-t border-gray-100 dark:border-zinc-800">
				<button
					onclick={handleClose}
					disabled={uploading}
					class="btn-pill btn-pill-ghost btn-pill-sm"
				>
					Cancel
				</button>
				<button
					onclick={handleUpload}
					disabled={files.length === 0 || uploading}
					class="btn-pill btn-pill-primary btn-pill-sm"
				>
					{uploading ? 'Uploading...' : `Upload ${files.length} file${files.length !== 1 ? 's' : ''}`}
				</button>
			</div>
		</Dialog.Content>
	</Dialog.Portal>
</Dialog.Root>

<style>
	.drop-zone {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		min-height: 200px;
		padding: 24px;
		border: 2px dashed var(--color-border, #e5e7eb);
		border-radius: 16px;
		background: var(--color-bg-secondary, #f9fafb);
		transition: all 0.2s ease;
	}

	:global(.dark) .drop-zone {
		border-color: rgba(255, 255, 255, 0.1);
		background: #1c1c1e;
	}

	.drop-zone.drag-over {
		border-color: #3b82f6;
		background: rgba(59, 130, 246, 0.05);
	}

	.drop-zone.has-files {
		padding: 12px;
		min-height: auto;
	}

	.drop-content {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 8px;
		text-align: center;
	}

	.drop-icon {
		width: 48px;
		height: 48px;
		color: var(--color-text-muted, #6b7280);
	}

	:global(.dark) .drop-icon {
		color: #6e6e73;
	}

	.drop-text {
		font-size: 14px;
		font-weight: 500;
		color: var(--color-text, #111827);
		margin: 0;
	}

	:global(.dark) .drop-text {
		color: #f5f5f7;
	}

	.drop-hint {
		font-size: 12px;
		color: var(--color-text-muted, #6b7280);
		margin: 0;
	}

	.browse-btn {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		padding: 8px 16px;
		font-size: 13px;
		font-weight: 500;
		color: #3b82f6;
		background: rgba(59, 130, 246, 0.1);
		border-radius: 8px;
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.browse-btn:hover {
		background: rgba(59, 130, 246, 0.2);
	}

	.formats-hint {
		font-size: 11px;
		color: var(--color-text-muted, #9ca3af);
		margin: 8px 0 0;
	}

	.files-list {
		display: flex;
		flex-direction: column;
		gap: 8px;
		width: 100%;
	}

	.file-item {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 10px 12px;
		background: var(--color-bg, white);
		border: 1px solid var(--color-border, #e5e7eb);
		border-radius: 10px;
	}

	:global(.dark) .file-item {
		background: #2c2c2e;
		border-color: rgba(255, 255, 255, 0.1);
	}

	.file-icon {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 36px;
		height: 36px;
		background: rgba(59, 130, 246, 0.1);
		color: #3b82f6;
		border-radius: 8px;
		flex-shrink: 0;
	}

	.file-info {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: 2px;
	}

	.file-name {
		font-size: 13px;
		font-weight: 500;
		color: var(--color-text, #111827);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	:global(.dark) .file-name {
		color: #f5f5f7;
	}

	.file-size {
		font-size: 11px;
		color: var(--color-text-muted, #6b7280);
	}

	:global(.dark) .file-size {
		color: #6e6e73;
	}

	.remove-file-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 28px;
		height: 28px;
		border: none;
		background: transparent;
		color: var(--color-text-muted, #6b7280);
		cursor: pointer;
		border-radius: 6px;
		transition: all 0.15s ease;
	}

	.remove-file-btn:hover:not(:disabled) {
		background: rgba(239, 68, 68, 0.1);
		color: #ef4444;
	}

	.remove-file-btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.add-more-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 6px;
		padding: 10px;
		font-size: 13px;
		font-weight: 500;
		color: #3b82f6;
		background: transparent;
		border: 1px dashed #3b82f6;
		border-radius: 10px;
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.add-more-btn:hover {
		background: rgba(59, 130, 246, 0.05);
	}

	.error-message {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 10px 12px;
		font-size: 13px;
		color: #ef4444;
		background: rgba(239, 68, 68, 0.1);
		border-radius: 8px;
	}

	.upload-progress {
		display: flex;
		align-items: center;
		gap: 12px;
	}

	.progress-bar {
		flex: 1;
		height: 8px;
		background: var(--color-bg-tertiary, #e5e7eb);
		border-radius: 4px;
		overflow: hidden;
	}

	:global(.dark) .progress-bar {
		background: #3a3a3c;
	}

	.progress-fill {
		height: 100%;
		background: #3b82f6;
		border-radius: 4px;
		transition: width 0.3s ease;
	}

	.progress-text {
		font-size: 12px;
		font-weight: 500;
		color: var(--color-text, #111827);
		min-width: 40px;
		text-align: right;
	}

	:global(.dark) .progress-text {
		color: #f5f5f7;
	}
</style>
