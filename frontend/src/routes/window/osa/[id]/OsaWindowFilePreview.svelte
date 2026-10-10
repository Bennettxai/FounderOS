<script lang="ts">
	import {
		FileCode,
		FileText,
		Database,
		Settings,
		Rocket,
		Copy,
		Check,
		Download,
		Loader2
	} from 'lucide-svelte';

	interface WorkflowFile {
		id: string;
		name: string;
		type: string;
		size: number;
		updated_at: string;
		language?: string;
	}

	interface Props {
		selectedFile: WorkflowFile | null;
		fileContent: string | null;
		renderedMarkdown: string;
		loadingContent: boolean;
		copied: boolean;
		onCopy: () => void;
		onDownload: () => void;
	}

	let {
		selectedFile,
		fileContent,
		renderedMarkdown,
		loadingContent,
		copied,
		onCopy,
		onDownload
	}: Props = $props();

	const fileCategories: Record<string, { label: string; icon: typeof FileCode }> = {
		all: { label: 'All Files', icon: FileCode },
		code: { label: 'Code', icon: FileCode },
		schema: { label: 'Schema', icon: Database },
		config: { label: 'Config', icon: Settings },
		documentation: { label: 'Docs', icon: FileText },
		deployment: { label: 'Deploy', icon: Rocket }
	};

	function getLanguageFromFile(file: WorkflowFile): string {
		if (file.language) return file.language;

		const ext = file.name.split('.').pop()?.toLowerCase() || '';
		const langMap: Record<string, string> = {
			js: 'javascript',
			ts: 'typescript',
			tsx: 'typescript',
			jsx: 'javascript',
			py: 'python',
			go: 'go',
			rs: 'rust',
			java: 'java',
			cpp: 'cpp',
			c: 'c',
			cs: 'csharp',
			rb: 'ruby',
			php: 'php',
			sh: 'bash',
			yaml: 'yaml',
			yml: 'yaml',
			json: 'json',
			xml: 'xml',
			html: 'html',
			css: 'css',
			scss: 'scss',
			md: 'markdown',
			sql: 'sql',
			dockerfile: 'dockerfile'
		};

		return langMap[ext] || 'plaintext';
	}

	function formatFileSize(bytes: number): string {
		if (bytes === 0) return '0 B';
		const k = 1024;
		const sizes = ['B', 'KB', 'MB', 'GB'];
		const i = Math.floor(Math.log(bytes) / Math.log(k));
		return Math.round((bytes / Math.pow(k, i)) * 10) / 10 + ' ' + sizes[i];
	}

	function formatDate(dateString: string): string {
		const date = new Date(dateString);
		return date.toLocaleString('en-US', {
			year: 'numeric',
			month: 'short',
			day: 'numeric',
			hour: '2-digit',
			minute: '2-digit'
		});
	}
</script>

<div class="file-preview">
	{#if !selectedFile}
		<div class="empty-preview">
			<FileCode size={48} class="text-gray-400" />
			<p>Select a file to preview</p>
		</div>
	{:else}
		<!-- File Header -->
		<div class="preview-header">
			<div class="file-info">
				<svelte:component
					this={fileCategories[selectedFile.type as keyof typeof fileCategories]?.icon || FileCode}
					size={20}
					class="text-blue-400"
				/>
				<div class="file-details">
					<h3 class="file-header-name">{selectedFile.name}</h3>
					<p class="file-header-meta">
						{formatFileSize(selectedFile.size)} • {selectedFile.type} • Updated {formatDate(
							selectedFile.updated_at
						)}
					</p>
				</div>
			</div>

			<div class="preview-actions">
				<button
					class="action-btn"
					onclick={onCopy}
					disabled={!fileContent || loadingContent}
					aria-label={copied ? 'Copied' : 'Copy file content'}
				>
					{#if copied}
						<Check size={16} class="text-green-400" />
					{:else}
						<Copy size={16} />
					{/if}
					<span>{copied ? 'Copied!' : 'Copy'}</span>
				</button>

				<button
					class="action-btn"
					onclick={onDownload}
					disabled={loadingContent}
					aria-label="Download file"
				>
					<Download size={16} />
					<span>Download</span>
				</button>
			</div>
		</div>

		<!-- File Content -->
		<div class="preview-content">
			{#if loadingContent}
				<div class="loading-content">
					<Loader2 size={32} class="animate-spin text-blue-400" />
					<p>Loading file content...</p>
				</div>
			{:else if fileContent}
				{#if renderedMarkdown}
					<div class="markdown-preview">
						{@html renderedMarkdown}
					</div>
				{:else}
					<pre class="code-preview"><code class="language-{getLanguageFromFile(selectedFile)}">{fileContent}</code></pre>
				{/if}
			{:else}
				<div class="empty-content">
					<FileText size={32} class="text-gray-400" />
					<p>No content available</p>
				</div>
			{/if}
		</div>
	{/if}
</div>

<style>
	.file-preview {
		flex: 1;
		display: flex;
		flex-direction: column;
		background: #0f172a;
		overflow: hidden;
	}

	.empty-preview {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		height: 100%;
		gap: 16px;
		color: #64748b;
	}

	.preview-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 16px 24px;
		background: #1e293b;
		border-bottom: 1px solid #334155;
	}

	.file-info {
		display: flex;
		align-items: center;
		gap: 12px;
		flex: 1;
		min-width: 0;
	}

	.file-details {
		flex: 1;
		min-width: 0;
	}

	.file-header-name {
		font-size: 16px;
		font-weight: 600;
		color: #f1f5f9;
		margin: 0;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.file-header-meta {
		font-size: 12px;
		color: #64748b;
		margin: 4px 0 0 0;
	}

	.preview-actions {
		display: flex;
		gap: 8px;
	}

	.action-btn {
		display: flex;
		align-items: center;
		gap: 6px;
		padding: 8px 12px;
		background: #0f172a;
		border: 1px solid #334155;
		border-radius: 6px;
		color: #e2e8f0;
		font-size: 14px;
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.action-btn:hover:not(:disabled) {
		background: #1e293b;
		border-color: #60a5fa;
	}

	.action-btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.preview-content {
		flex: 1;
		overflow: auto;
		padding: 24px;
	}

	.loading-content,
	.empty-content {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		height: 100%;
		gap: 16px;
		color: #64748b;
	}

	.code-preview {
		margin: 0;
		padding: 20px;
		background: #020617;
		border: 1px solid #1e293b;
		border-radius: 8px;
		overflow-x: auto;
		font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
		font-size: 13px;
		line-height: 1.7;
		color: #e2e8f0;
	}

	.code-preview code {
		display: block;
	}

	/* Markdown Styling */
	.markdown-preview {
		color: #e2e8f0;
		line-height: 1.7;
	}

	.markdown-preview :global(h1),
	.markdown-preview :global(h2),
	.markdown-preview :global(h3),
	.markdown-preview :global(h4),
	.markdown-preview :global(h5),
	.markdown-preview :global(h6) {
		color: #f1f5f9;
		margin-top: 24px;
		margin-bottom: 16px;
		font-weight: 600;
	}

	.markdown-preview :global(h1) {
		font-size: 28px;
		border-bottom: 1px solid #334155;
		padding-bottom: 8px;
	}

	.markdown-preview :global(h2) {
		font-size: 24px;
		border-bottom: 1px solid #1e293b;
		padding-bottom: 6px;
	}

	.markdown-preview :global(h3) {
		font-size: 20px;
	}

	.markdown-preview :global(code) {
		background: #020617;
		padding: 2px 8px;
		border-radius: 4px;
		font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
		font-size: 13px;
		border: 1px solid #1e293b;
	}

	.markdown-preview :global(pre) {
		background: #020617;
		padding: 20px;
		border: 1px solid #1e293b;
		border-radius: 8px;
		overflow-x: auto;
		margin: 16px 0;
	}

	.markdown-preview :global(pre code) {
		background: transparent;
		padding: 0;
		border: none;
	}

	.markdown-preview :global(a) {
		color: #60a5fa;
		text-decoration: none;
	}

	.markdown-preview :global(a:hover) {
		text-decoration: underline;
	}

	.markdown-preview :global(ul),
	.markdown-preview :global(ol) {
		padding-left: 24px;
		margin: 16px 0;
	}

	.markdown-preview :global(li) {
		margin: 8px 0;
	}

	.markdown-preview :global(blockquote) {
		border-left: 4px solid #334155;
		padding-left: 16px;
		margin: 16px 0;
		color: #94a3b8;
	}

	.markdown-preview :global(table) {
		border-collapse: collapse;
		width: 100%;
		margin: 16px 0;
	}

	.markdown-preview :global(th),
	.markdown-preview :global(td) {
		border: 1px solid #334155;
		padding: 8px 12px;
		text-align: left;
	}

	.markdown-preview :global(th) {
		background: #1e293b;
		font-weight: 600;
	}

	.markdown-preview :global(img) {
		max-width: 100%;
		border-radius: 8px;
		margin: 16px 0;
	}
</style>
