<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { Loader2, Play, RefreshCw, AlertCircle, ExternalLink, Square } from 'lucide-svelte';
	import { generatedAppsStore, type GeneratedApp } from '$lib/stores/generatedAppsStore';
	import { getSandboxInfo, type SandboxInfo } from '$lib/api/sandbox';
	import SandboxStatusBadge from '$lib/components/osa/SandboxStatusBadge.svelte';
	import SandboxPreview from './sandbox/SandboxPreview.svelte';
	import type { SandboxStatus } from '$lib/types/sandbox';

	interface Props {
		appId: string;
		sandboxEditId?: string;
	}

	let { appId, sandboxEditId }: Props = $props();

	let showReviewPanel = $state(false);

	let app = $state<GeneratedApp | null>(null);
	let sandboxInfo = $state<SandboxInfo | null>(null);
	let isLoading = $state(true);
	let actionLoading = $state(false);
	let error = $state<string | null>(null);
	let iframeRef = $state<HTMLIFrameElement | null>(null);
	let pollInterval: ReturnType<typeof setInterval> | null = null;

	// Derive sandbox status
	let sandboxStatus = $derived<SandboxStatus>(
		sandboxInfo?.status ?? app?.sandbox?.status ?? 'pending'
	);
	let sandboxUrl = $derived(sandboxInfo?.url ?? app?.sandbox?.url);
	let isRunning = $derived(sandboxStatus === 'running');
	let isBuilding = $derived(sandboxStatus === 'deploying');
	let isStopped = $derived(sandboxStatus === 'stopped');
	let isError = $derived(sandboxStatus === 'failed');

	onMount(async () => {
		showReviewPanel = Boolean(sandboxEditId);
		await loadApp();
		// Poll for status updates while building
		pollInterval = setInterval(async () => {
			if (isBuilding) {
				await refreshStatus();
			}
		}, 3000);
	});

	onDestroy(() => {
		if (pollInterval) clearInterval(pollInterval);
	});

	async function loadApp() {
		isLoading = true;
		error = null;
		try {
			app = await generatedAppsStore.getAppById(appId);
			if (!app) {
				error = 'App not found';
				return;
			}
			await refreshStatus();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load app';
		} finally {
			isLoading = false;
		}
	}

	async function refreshStatus() {
		try {
			sandboxInfo = await getSandboxInfo(appId);
		} catch {
			// Sandbox may not exist yet
			sandboxInfo = null;
		}
	}

	async function handleDeploy() {
		actionLoading = true;
		error = null;
		try {
			await generatedAppsStore.deployApp(appId);
			await refreshStatus();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to deploy';
		} finally {
			actionLoading = false;
		}
	}

	async function handleStart() {
		actionLoading = true;
		error = null;
		try {
			await generatedAppsStore.startSandbox(appId);
			await refreshStatus();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to start sandbox';
		} finally {
			actionLoading = false;
		}
	}

	async function handleStop() {
		actionLoading = true;
		error = null;
		try {
			await generatedAppsStore.stopSandbox(appId);
			await refreshStatus();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to stop sandbox';
		} finally {
			actionLoading = false;
		}
	}

	function handleOpenExternal() {
		if (sandboxUrl) {
			window.open(sandboxUrl, '_blank', 'noopener,noreferrer');
		}
	}

	function handleRefreshIframe() {
		if (iframeRef && sandboxUrl) {
			iframeRef.src = sandboxUrl;
		}
	}
</script>

<div class="sandbox-window flex flex-col h-full bg-white dark:bg-gray-900">
	<!-- Toolbar -->
	<div class="flex items-center justify-between px-4 py-2 border-b border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800">
		<div class="flex items-center gap-3">
			<span class="text-sm font-medium text-gray-900 dark:text-white truncate max-w-[200px]">
				{app?.app_name ?? 'Loading...'}
			</span>
			<SandboxStatusBadge status={sandboxStatus} size="sm" />
		</div>

		<div class="flex items-center gap-2">
			{#if sandboxEditId}
				<button
					onclick={() => (showReviewPanel = !showReviewPanel)}
					class="btn-pill btn-pill-xs {showReviewPanel
						? 'btn-pill-soft'
						: 'btn-pill-ghost'}"
					title="Review proposed changes"
				>
					Review Changes
				</button>
			{/if}
			{#if isRunning}
				<button
					onclick={handleRefreshIframe}
					class="p-1.5 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors text-gray-500 hover:text-gray-900 dark:text-gray-400 dark:hover:text-white"
					title="Refresh preview"
				>
					<RefreshCw class="w-4 h-4" />
				</button>
				<button
					onclick={handleOpenExternal}
					class="p-1.5 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors text-gray-500 hover:text-gray-900 dark:text-gray-400 dark:hover:text-white"
					title="Open in new tab"
				>
					<ExternalLink class="w-4 h-4" />
				</button>
				<button
					onclick={handleStop}
					disabled={actionLoading}
					class="p-1.5 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors text-gray-500 hover:text-gray-900 dark:text-gray-400 dark:hover:text-white"
					title="Stop sandbox"
				>
					{#if actionLoading}
						<Loader2 class="w-4 h-4 animate-spin" />
					{:else}
						<Square class="w-4 h-4" />
					{/if}
				</button>
			{/if}
		</div>
	</div>

	<!-- Content -->
	<div class="flex-1 relative">
		{#if showReviewPanel && sandboxEditId}
			<!-- Sandbox Edit Review Panel -->
			<SandboxPreview
				sandboxId={sandboxEditId}
				onClose={() => {
					showReviewPanel = false;
					refreshStatus();
				}}
				class="absolute inset-0"
			/>

		{:else if isLoading}
			<!-- Loading State -->
			<div class="absolute inset-0 flex items-center justify-center bg-gray-50 dark:bg-gray-900">
				<div class="text-center">
					<Loader2 class="w-8 h-8 animate-spin text-blue-500 mx-auto mb-3" />
					<p class="text-sm text-gray-500 dark:text-gray-400">Loading app...</p>
				</div>
			</div>

		{:else if error}
			<!-- Error State -->
			<div class="absolute inset-0 flex items-center justify-center bg-gray-50 dark:bg-gray-900">
				<div class="text-center max-w-sm px-4">
					<AlertCircle class="w-12 h-12 text-red-500 mx-auto mb-3" />
					<p class="text-sm font-medium text-gray-900 dark:text-white mb-2">Error</p>
					<p class="text-sm text-gray-500 dark:text-gray-400 mb-4">{error}</p>
					<button
						onclick={loadApp}
						class="btn-pill btn-pill-primary btn-pill-sm"
					>
						Retry
					</button>
				</div>
			</div>

		{:else if !sandboxInfo && !app?.sandbox}
			<!-- Not Deployed State -->
			<div class="absolute inset-0 flex items-center justify-center bg-gray-50 dark:bg-gray-900">
				<div class="text-center max-w-sm px-4">
					<div class="w-16 h-16 bg-blue-100 dark:bg-blue-900/30 rounded-full flex items-center justify-center mx-auto mb-4">
						<Play class="w-8 h-8 text-blue-600 dark:text-blue-400" />
					</div>
					<p class="text-lg font-medium text-gray-900 dark:text-white mb-2">Ready to Deploy</p>
					<p class="text-sm text-gray-500 dark:text-gray-400 mb-4">
						Deploy this app to a Docker sandbox to preview it.
					</p>
					<button
						onclick={handleDeploy}
						disabled={actionLoading}
						class="btn-pill btn-pill-primary btn-pill-sm flex items-center gap-2 mx-auto"
					>
						{#if actionLoading}
							<Loader2 class="w-4 h-4 animate-spin" />
							Deploying...
						{:else}
							<Play class="w-4 h-4" />
							Deploy to Sandbox
						{/if}
					</button>
				</div>
			</div>

		{:else if isBuilding}
			<!-- Building State -->
			<div class="absolute inset-0 flex items-center justify-center bg-gray-50 dark:bg-gray-900">
				<div class="text-center">
					<Loader2 class="w-12 h-12 animate-spin text-yellow-500 mx-auto mb-4" />
					<p class="text-lg font-medium text-gray-900 dark:text-white mb-2">Building Sandbox</p>
					<p class="text-sm text-gray-500 dark:text-gray-400">
						Installing dependencies and starting the app...
					</p>
				</div>
			</div>

		{:else if isStopped}
			<!-- Stopped State -->
			<div class="absolute inset-0 flex items-center justify-center bg-gray-50 dark:bg-gray-900">
				<div class="text-center max-w-sm px-4">
					<div class="w-16 h-16 bg-gray-200 dark:bg-gray-700 rounded-full flex items-center justify-center mx-auto mb-4">
						<Square class="w-8 h-8 text-gray-500" />
					</div>
					<p class="text-lg font-medium text-gray-900 dark:text-white mb-2">Sandbox Stopped</p>
					<p class="text-sm text-gray-500 dark:text-gray-400 mb-4">
						Start the sandbox to preview your app.
					</p>
					<button
						onclick={handleStart}
						disabled={actionLoading}
						class="btn-pill btn-pill-primary btn-pill-sm flex items-center gap-2 mx-auto"
					>
						{#if actionLoading}
							<Loader2 class="w-4 h-4 animate-spin" />
							Starting...
						{:else}
							<Play class="w-4 h-4" />
							Start Sandbox
						{/if}
					</button>
				</div>
			</div>

		{:else if isError}
			<!-- Error State -->
			<div class="absolute inset-0 flex items-center justify-center bg-gray-50 dark:bg-gray-900">
				<div class="text-center max-w-sm px-4">
					<AlertCircle class="w-12 h-12 text-red-500 mx-auto mb-3" />
					<p class="text-lg font-medium text-gray-900 dark:text-white mb-2">Sandbox Error</p>
					<p class="text-sm text-gray-500 dark:text-gray-400 mb-4">
						{sandboxInfo?.health_status === 'unhealthy' ? 'The sandbox is unhealthy.' : 'Something went wrong with the sandbox.'}
					</p>
					<div class="flex gap-2 justify-center">
						<button
							onclick={handleStart}
							disabled={actionLoading}
							class="btn-pill btn-pill-primary btn-pill-sm"
						>
							Restart
						</button>
					</div>
				</div>
			</div>

		{:else if isRunning && sandboxUrl}
			<!-- Running State - Show iframe -->
			<iframe
				bind:this={iframeRef}
				src={sandboxUrl}
				title="{app?.app_name ?? 'Sandbox'} Preview"
				class="w-full h-full border-0"
				sandbox="allow-scripts allow-same-origin allow-forms allow-popups allow-popups-to-escape-sandbox"
			></iframe>
		{/if}
	</div>
</div>
