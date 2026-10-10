<script lang="ts">
	import type { GeneratedApp } from '$lib/stores/generatedAppsStore';

	interface Props {
		app: GeneratedApp;
		onDeploy?: (app: GeneratedApp) => Promise<void>;
		onStop?: (app: GeneratedApp) => Promise<void>;
		onDelete?: (app: GeneratedApp) => Promise<void>;
		onViewLogs?: (app: GeneratedApp) => void;
	}

	let { app, onDeploy, onStop, onDelete, onViewLogs }: Props = $props();

	let isDeploying = $state(false);
	let isStopping = $state(false);
	let showDeleteModal = $state(false);
	let error = $state<string | null>(null);

	async function handleDeploy() {
		if (!onDeploy) return;

		isDeploying = true;
		error = null;

		try {
			await onDeploy(app);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to deploy app';
		} finally {
			isDeploying = false;
		}
	}

	async function handleStop() {
		if (!onStop) return;

		isStopping = true;
		error = null;

		try {
			await onStop(app);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to stop app';
		} finally {
			isStopping = false;
		}
	}

	async function handleDelete() {
		if (!onDelete) return;

		error = null;

		try {
			await onDelete(app);
			showDeleteModal = false;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to delete app';
		}
	}

	function handleViewLogs() {
		if (onViewLogs) onViewLogs(app);
	}
</script>

<div class="flex flex-col gap-3">
	<!-- Primary Actions -->
	<div class="flex flex-wrap gap-2">
		<!-- Deploy Button (for generated apps) -->
		{#if app.status === 'generated' && onDeploy}
			<button
				onclick={handleDeploy}
				disabled={isDeploying}
				class="btn-pill btn-pill-success btn-pill-sm flex-1 min-w-[120px] disabled:bg-gray-400 flex items-center justify-center gap-2"
			>
				{#if isDeploying}
					<svg
						class="w-4 h-4 animate-spin"
						fill="none"
						stroke="currentColor"
						viewBox="0 0 24 24"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
						/>
					</svg>
					Deploying...
				{:else}
					<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12"
						/>
					</svg>
					Deploy
				{/if}
			</button>
		{/if}

		<!-- Stop Button (for deployed apps) -->
		{#if app.status === 'deployed' && onStop}
			<button
				onclick={handleStop}
				disabled={isStopping}
				class="btn-pill btn-pill-warning btn-pill-sm flex-1 min-w-[120px] disabled:bg-gray-400 flex items-center justify-center gap-2"
			>
				{#if isStopping}
					<svg
						class="w-4 h-4 animate-spin"
						fill="none"
						stroke="currentColor"
						viewBox="0 0 24 24"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
						/>
					</svg>
					Stopping...
				{:else}
					<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
						/>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M9 10a1 1 0 011-1h4a1 1 0 011 1v4a1 1 0 01-1 1h-4a1 1 0 01-1-1v-4z"
						/>
					</svg>
					Stop
				{/if}
			</button>
		{/if}

		<!-- View Logs Button -->
		{#if onViewLogs}
			<button
				onclick={handleViewLogs}
				class="flex-1 min-w-[120px] btn-pill btn-pill-primary flex items-center justify-center gap-2"
			>
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
					/>
				</svg>
				View Logs
			</button>
		{/if}
	</div>

	<!-- Secondary Actions -->
	<div class="flex gap-2">
		<!-- Delete Button -->
		{#if onDelete}
			<button
				onclick={() => (showDeleteModal = true)}
				class="flex-1 btn-pill btn-pill-danger flex items-center justify-center gap-2"
			>
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
					/>
				</svg>
				Delete
			</button>
		{/if}
	</div>

	<!-- Error Message -->
	{#if error}
		<div class="bg-red-50 dark:bg-red-900/30 border border-red-200 dark:border-red-800 rounded-lg p-3">
			<p class="text-sm text-red-700 dark:text-red-400 flex items-start gap-2">
				<svg class="w-4 h-4 flex-shrink-0 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
					/>
				</svg>
				<span>{error}</span>
			</p>
		</div>
	{/if}
</div>

<!-- Delete Confirmation Modal -->
{#if showDeleteModal}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
		onclick={() => (showDeleteModal = false)}
	>
		<div
			class="bg-white dark:bg-gray-800 rounded-xl p-6 max-w-md w-full mx-4 border border-gray-200 dark:border-gray-700 shadow-2xl"
			onclick={(e) => e.stopPropagation()}
		>
			<div class="flex items-start gap-4 mb-4">
				<div class="w-12 h-12 rounded-full bg-red-100 dark:bg-red-900/30 flex items-center justify-center flex-shrink-0">
					<svg
						class="w-6 h-6 text-red-600 dark:text-red-400"
						fill="none"
						stroke="currentColor"
						viewBox="0 0 24 24"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
						/>
					</svg>
				</div>
				<div class="flex-1">
					<h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-2">
						Delete App
					</h3>
					<p class="text-sm text-gray-600 dark:text-gray-400">
						Are you sure you want to delete <strong>{app.app_name}</strong>? This action cannot be
						undone.
					</p>
				</div>
			</div>

			<div class="flex gap-3 justify-end">
				<button
					onclick={() => (showDeleteModal = false)}
					class="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 rounded-lg transition-colors"
				>
					Cancel
				</button>
				<button
					onclick={handleDelete}
					class="btn-pill btn-pill-danger btn-pill-sm"
				>
					Delete App
				</button>
			</div>
		</div>
	</div>
{/if}
