<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { Loader2, AlertCircle, ArrowLeft } from 'lucide-svelte';
	import { goto } from '$app/navigation';
	import { marked } from 'marked';
	import { installModule } from '$lib/api/osa/files';
	import { request } from '$lib/api/base';
	import { deploySandbox } from '$lib/api/sandbox';
	import OsaWindowHeader from './OsaWindowHeader.svelte';
	import OsaWindowSidebar from './OsaWindowSidebar.svelte';
	import OsaWindowFilePreview from './OsaWindowFilePreview.svelte';

	// Get workflow ID from URL params
	const workflowId = $derived($page.params.id);

	// State
	let workflow = $state<any>(null);
	let files = $state<any[]>([]);
	let selectedFile = $state<any>(null);
	let fileContent = $state<string | null>(null);
	let renderedMarkdown = $state<string>('');
	let activeTab = $state<string>('all');
	let loading = $state(true);
	let loadingContent = $state(false);
	let error = $state<string | null>(null);
	let copied = $state(false);
	let installing = $state(false);
	let installSuccess = $state(false);
	let installError = $state<string | null>(null);
	let deploying = $state(false);
	let deploymentUrl = $state('');
	let deployError = $state('');

	const filteredFiles = $derived(getFilesByCategory(activeTab));

	onMount(() => {
		loadWorkflow();
	});

	async function loadWorkflow() {
		loading = true;
		error = null;

		try {
			const workflowData = await request<any>(`/osa/workflows/${workflowId}`);
			workflow = workflowData;

			const filesData = await request<{ files: any[] }>(`/osa/workflows/${workflowId}/files`);
			files = filesData.files || [];

			if (files.length > 0) {
				selectFile(files[0]);
			}
		} catch (err: any) {
			console.error('Failed to load workflow:', err);
			error = err?.message || 'Failed to load workflow';
		} finally {
			loading = false;
		}
	}

	async function selectFile(file: any) {
		selectedFile = file;
		loadingContent = true;
		fileContent = null;
		renderedMarkdown = '';

		try {
			const response = await request<{ content: string; file: any }>(
				`/osa/files/${file.id}/content`
			);
			fileContent = response.content;

			if (file.type === 'markdown' || file.name.endsWith('.md')) {
				renderedMarkdown = await marked.parse(fileContent);
			}
		} catch (err: any) {
			console.error('Failed to load file content:', err);
			fileContent = `Error loading file: ${err?.message || 'Unknown error'}`;
		} finally {
			loadingContent = false;
		}
	}

	async function handleCopy() {
		if (!fileContent) return;

		try {
			await navigator.clipboard.writeText(fileContent);
			copied = true;
			setTimeout(() => {
				copied = false;
			}, 2000);
		} catch (err) {
			console.error('Failed to copy:', err);
		}
	}

	async function handleDownload() {
		if (!selectedFile || !fileContent) return;

		try {
			const blob = new Blob([fileContent], { type: 'text/plain' });
			const url = URL.createObjectURL(blob);
			const a = document.createElement('a');
			a.href = url;
			a.download = selectedFile.name;
			document.body.appendChild(a);
			a.click();
			document.body.removeChild(a);
			URL.revokeObjectURL(url);
		} catch (err) {
			console.error('Failed to download file:', err);
		}
	}

	async function handleInstallModule() {
		if (!workflow) return;

		installing = true;
		installError = null;
		installSuccess = false;

		try {
			await installModule(workflow.id, {
				module_name: workflow.name,
				install_path: undefined,
				file_ids: files.map((f) => f.id)
			});

			installSuccess = true;
			setTimeout(() => {
				installSuccess = false;
			}, 3000);
		} catch (err: any) {
			console.error('Failed to install module:', err);
			installError = err?.message || 'Failed to install module';
		} finally {
			installing = false;
		}
	}

	async function handleDeploy() {
		deploying = true;
		deployError = '';
		deploymentUrl = '';

		try {
			const result = await deploySandbox(workflow.id, workflow.display_name || workflow.name);
			deploymentUrl = result.url;
		} catch (err) {
			deployError = err instanceof Error ? err.message : 'Failed to deploy';
		} finally {
			deploying = false;
		}
	}

	function getFilesByCategory(category: string) {
		if (category === 'all') return files;
		return files.filter((f) => f.type === category);
	}
</script>

<div class="workflow-viewer">
	{#if loading}
		<div class="loading-state">
			<Loader2 size={48} class="animate-spin text-blue-400" />
			<p>Loading workflow...</p>
		</div>
	{:else if error}
		<div class="error-state">
			<AlertCircle size={48} class="text-red-400" />
			<p class="error-text">{error}</p>
			<button class="back-btn" onclick={() => goto('/window')} aria-label="Back to Desktop">
				<ArrowLeft size={16} />
				<span>Back to Desktop</span>
			</button>
		</div>
	{:else if workflow}
		<OsaWindowHeader
			{workflow}
			fileCount={files.length}
			{installing}
			{installSuccess}
			{installError}
			{deploying}
			{deploymentUrl}
			{deployError}
			onInstall={handleInstallModule}
			onDeploy={handleDeploy}
		/>

		<div class="workflow-content">
			<OsaWindowSidebar
				{files}
				{filteredFiles}
				{selectedFile}
				{activeTab}
				onSelectFile={selectFile}
				onSetActiveTab={(tab) => (activeTab = tab)}
			/>

			<OsaWindowFilePreview
				{selectedFile}
				{fileContent}
				{renderedMarkdown}
				{loadingContent}
				{copied}
				onCopy={handleCopy}
				onDownload={handleDownload}
			/>
		</div>
	{/if}
</div>

<style>
	.workflow-viewer {
		display: flex;
		flex-direction: column;
		height: 100vh;
		background: #0f172a;
		color: #e2e8f0;
		overflow: hidden;
	}

	/* Loading & Error States */
	.loading-state,
	.error-state {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		height: 100%;
		gap: 16px;
		color: #94a3b8;
	}

	.error-text {
		color: #ef4444;
		font-size: 16px;
	}

	.back-btn {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 8px 16px;
		background: #0f172a;
		border: 1px solid #334155;
		border-radius: 6px;
		color: #e2e8f0;
		font-size: 14px;
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.back-btn:hover {
		background: #1e293b;
		border-color: #60a5fa;
	}

	/* Main Content */
	.workflow-content {
		display: flex;
		flex: 1;
		overflow: hidden;
	}

	@media (max-width: 768px) {
		.workflow-viewer {
			height: auto;
			min-height: 100vh;
			overflow: auto;
		}
		.workflow-content {
			flex-direction: column;
			overflow: visible;
		}
	}
</style>
