<!--
	SandboxPreview.svelte
	Main container for the sandbox edit preview flow.
	Fetches sandbox state from Pedro's sandbox edit API, shows the diff viewer,
	and provides approve/reject controls with full lifecycle management.

	Flow: fork → edit files → validate → preview diff → approve/reject
-->
<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import {
		getSandboxEdit,
		getPreview,
		validateSandbox,
		type SandboxEdit
	} from '$lib/api/sandbox-preview';
	import DiffViewer from './DiffViewer.svelte';
	import ApproveReject from './ApproveReject.svelte';

	interface Props {
		sandboxId: string;
		onClose?: () => void;
		class?: string;
	}

	let { sandboxId, onClose, class: className }: Props = $props();

	let sandbox = $state<SandboxEdit | null>(null);
	let isLoading = $state(true);
	let isValidating = $state(false);
	let error = $state<string | null>(null);
	let successMessage = $state<string | null>(null);

	let hasDiffs = $derived((sandbox?.diff?.length ?? 0) > 0);
	let hasErrors = $derived((sandbox?.errors?.length ?? 0) > 0);

	// Track timeouts for cleanup on destroy
	let timeoutIds: ReturnType<typeof setTimeout>[] = [];

	onDestroy(() => {
		timeoutIds.forEach(clearTimeout);
	});

	onMount(async () => {
		await loadPreview();
	});

	async function loadPreview() {
		isLoading = true;
		error = null;
		successMessage = null;
		try {
			// First get current state
			sandbox = await getSandboxEdit(sandboxId);

			// If state is pending or validated, also fetch the diff preview
			if (sandbox.state === 'pending' || sandbox.state === 'validated') {
				const withDiff = await getPreview(sandboxId);
				sandbox = withDiff;
			}
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load preview';
		} finally {
			isLoading = false;
		}
	}

	async function handleValidate() {
		isValidating = true;
		error = null;
		try {
			sandbox = await validateSandbox(sandboxId);
			if (sandbox.state === 'validated') {
				// Refresh with diff preview
				const withDiff = await getPreview(sandboxId);
				sandbox = withDiff;
			}
		} catch (err) {
			error = err instanceof Error ? err.message : 'Validation failed';
		} finally {
			isValidating = false;
		}
	}

	function handleApproved() {
		successMessage = 'Changes applied successfully.';
		timeoutIds.push(setTimeout(() => {
			onClose?.();
		}, 1500));
	}

	function handleRejected() {
		successMessage = 'Changes rejected and discarded.';
		timeoutIds.push(setTimeout(() => {
			onClose?.();
		}, 1500));
	}
</script>

<section
	class="sandbox-preview flex flex-col h-full {className ?? ''}"
	aria-labelledby="preview-title"
>
	<!-- Header -->
	<div class="flex items-center justify-between px-4 py-3 border-b border-gray-200 dark:border-gray-700">
		<div class="flex items-center gap-2">
			<h2 id="preview-title" class="text-sm font-semibold text-gray-900 dark:text-white">
				Review Proposed Changes
			</h2>
			{#if sandbox}
				<span class="rounded-full px-2 py-0.5 text-[10px] font-semibold uppercase
					{sandbox.state === 'validated' ? 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400' :
					sandbox.state === 'applied' ? 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400' :
					sandbox.state === 'rejected' ? 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400' :
					'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400'}"
				>
					{sandbox.state}
				</span>
			{/if}
		</div>

		{#if onClose}
			<button
				onclick={onClose}
				class="btn-pill btn-pill-ghost btn-pill-icon"
				aria-label="Close preview"
			>
				<svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<line x1="18" y1="6" x2="6" y2="18" />
					<line x1="6" y1="6" x2="18" y2="18" />
				</svg>
			</button>
		{/if}
	</div>

	<!-- Content -->
	<div class="flex-1 overflow-y-auto p-4">
		{#if isLoading}
			<!-- Loading State -->
			<div class="flex flex-col items-center justify-center py-12">
				<svg class="h-8 w-8 animate-spin text-blue-500 mb-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<path d="M12 2v4m0 12v4m-7.07-3.93l2.83-2.83m8.48-8.48l2.83-2.83M2 12h4m12 0h4m-3.93 7.07l-2.83-2.83M7.76 7.76L4.93 4.93" />
				</svg>
				<p class="text-sm text-gray-500 dark:text-gray-400">Loading preview...</p>
			</div>

		{:else if error}
			<!-- Error State -->
			<div class="flex flex-col items-center justify-center py-12">
				<svg class="h-10 w-10 text-red-500 mb-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<circle cx="12" cy="12" r="10" />
					<line x1="12" y1="8" x2="12" y2="12" />
					<line x1="12" y1="16" x2="12.01" y2="16" />
				</svg>
				<p class="text-sm font-medium text-gray-900 dark:text-white mb-1">Error</p>
				<p class="text-xs text-gray-500 dark:text-gray-400 mb-3">{error}</p>
				<button
					onclick={loadPreview}
					class="btn-pill btn-pill-primary btn-pill-xs"
				>
					Retry
				</button>
			</div>

		{:else if successMessage}
			<!-- Success State -->
			<div class="flex flex-col items-center justify-center py-12">
				<svg class="h-10 w-10 text-green-500 mb-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<circle cx="12" cy="12" r="10" />
					<polyline points="16 10 11 15 8 12" />
				</svg>
				<p class="text-sm font-medium text-gray-900 dark:text-white">{successMessage}</p>
			</div>

		{:else if sandbox?.state === 'applied'}
			<!-- Already applied -->
			<div class="flex flex-col items-center justify-center py-12">
				<svg class="h-10 w-10 text-blue-500 mb-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<circle cx="12" cy="12" r="10" />
					<polyline points="16 10 11 15 8 12" />
				</svg>
				<p class="text-sm font-medium text-gray-900 dark:text-white">Changes Already Applied</p>
				<p class="text-xs text-gray-500 dark:text-gray-400 mt-1">These changes have been applied to the module.</p>
			</div>

		{:else if sandbox?.state === 'rejected'}
			<!-- Already rejected -->
			<div class="flex flex-col items-center justify-center py-12">
				<svg class="h-10 w-10 text-gray-400 mb-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<circle cx="12" cy="12" r="10" />
					<line x1="15" y1="9" x2="9" y2="15" />
					<line x1="9" y1="9" x2="15" y2="15" />
				</svg>
				<p class="text-sm font-medium text-gray-900 dark:text-white">Changes Rejected</p>
				<p class="text-xs text-gray-500 dark:text-gray-400 mt-1">These changes have been discarded.</p>
			</div>

		{:else}
			<!-- Active preview: pending or validated -->

			<!-- Validation errors -->
			{#if hasErrors}
				<div class="mb-4 rounded-lg border border-red-200 bg-red-50 dark:border-red-800 dark:bg-red-950/20 px-3 py-2">
					<p class="text-xs font-medium text-red-800 dark:text-red-300 mb-1">Validation Errors</p>
					<ul class="list-disc list-inside text-xs text-red-600 dark:text-red-400 space-y-0.5">
						{#each sandbox?.errors ?? [] as err}
							<li>{err}</li>
						{/each}
					</ul>
				</div>
			{/if}

			<!-- Validate button (when pending) -->
			{#if sandbox?.state === 'pending'}
				<div class="mb-4 flex items-center gap-2">
					<button
						onclick={handleValidate}
						disabled={isValidating}
						class="btn-pill btn-pill-primary btn-pill-xs flex items-center gap-1.5"
					>
						{#if isValidating}
							<svg class="h-3 w-3 animate-spin" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
								<path d="M12 2v4m0 12v4m-7.07-3.93l2.83-2.83m8.48-8.48l2.83-2.83M2 12h4m12 0h4m-3.93 7.07l-2.83-2.83M7.76 7.76L4.93 4.93" />
							</svg>
							Validating...
						{:else}
							Validate Changes
						{/if}
					</button>
					<span class="text-xs text-gray-500 dark:text-gray-400">
						Validate before approving
					</span>
				</div>
			{/if}

			<!-- Diff viewer -->
			{#if hasDiffs}
				<DiffViewer diffs={sandbox?.diff ?? []} moduleName={sandbox?.module_name} />
			{:else}
				<div class="rounded-lg border border-gray-200 dark:border-gray-700 px-4 py-8 text-center">
					<p class="text-sm text-gray-500 dark:text-gray-400">No changes to preview</p>
					<p class="text-xs text-gray-400 dark:text-gray-500 mt-1">
						The sandbox has no file differences to show.
					</p>
				</div>
			{/if}
		{/if}
	</div>

	<!-- Footer with approve/reject (only for active states) -->
	{#if sandbox && (sandbox.state === 'pending' || sandbox.state === 'validated') && !isLoading && !error && !successMessage}
		<div class="border-t border-gray-200 dark:border-gray-700 px-4 py-3">
			<ApproveReject
				{sandboxId}
				sandboxState={sandbox.state}
				onApprove={handleApproved}
				onReject={handleRejected}
				disabled={isLoading || isValidating}
			/>
		</div>
	{/if}
</section>
