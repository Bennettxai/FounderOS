<!--
	ApproveReject.svelte
	Approve/Reject action buttons for sandbox edit changes.
	Calls Pedro's sandbox edit API: apply (requires validated state) and reject.
-->
<script lang="ts">
	import { applyChanges, rejectChanges } from '$lib/api/sandbox-preview';

	interface Props {
		sandboxId: string;
		sandboxState?: string;
		onApprove?: () => void;
		onReject?: () => void;
		disabled?: boolean;
	}

	let { sandboxId, sandboxState, onApprove, onReject, disabled = false }: Props = $props();

	let isApplying = $state(false);
	let isRejecting = $state(false);
	let error = $state<string | null>(null);
	let showRejectConfirm = $state(false);

	let isBusy = $derived(isApplying || isRejecting);
	let needsValidation = $derived(sandboxState === 'pending');

	async function handleApprove() {
		isApplying = true;
		error = null;
		try {
			await applyChanges(sandboxId);
			onApprove?.();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to apply changes';
		} finally {
			isApplying = false;
		}
	}

	function handleRejectClick() {
		showRejectConfirm = true;
	}

	async function handleRejectConfirm() {
		showRejectConfirm = false;
		isRejecting = true;
		error = null;
		try {
			await rejectChanges(sandboxId);
			onReject?.();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to reject changes';
		} finally {
			isRejecting = false;
		}
	}
</script>

<div class="approve-reject flex flex-col gap-2">
	<!-- Validation hint -->
	{#if needsValidation}
		<div class="rounded-md bg-yellow-50 dark:bg-yellow-950/20 border border-yellow-200 dark:border-yellow-800 px-3 py-2">
			<p class="text-xs text-yellow-700 dark:text-yellow-400">
				Changes must be validated before applying. Validate first, then approve.
			</p>
		</div>
	{/if}

	<!-- Action buttons -->
	<div class="flex items-center gap-2">
		<button
			onclick={handleApprove}
			disabled={isBusy || disabled || needsValidation}
			aria-label="Approve proposed changes"
			aria-busy={isApplying}
			class="btn-pill btn-pill-success btn-pill-sm flex items-center gap-1.5"
		>
			{#if isApplying}
				<svg class="h-4 w-4 animate-spin" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<path d="M12 2v4m0 12v4m-7.07-3.93l2.83-2.83m8.48-8.48l2.83-2.83M2 12h4m12 0h4m-3.93 7.07l-2.83-2.83M7.76 7.76L4.93 4.93" />
				</svg>
				Applying...
			{:else}
				<svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<polyline points="20 6 9 17 4 12" />
				</svg>
				Approve
			{/if}
		</button>

		<button
			onclick={handleRejectClick}
			disabled={isBusy || disabled}
			aria-label="Reject proposed changes"
			aria-busy={isRejecting}
			class="btn-pill btn-pill-secondary btn-pill-sm flex items-center gap-1.5"
		>
			{#if isRejecting}
				<svg class="h-4 w-4 animate-spin" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<path d="M12 2v4m0 12v4m-7.07-3.93l2.83-2.83m8.48-8.48l2.83-2.83M2 12h4m12 0h4m-3.93 7.07l-2.83-2.83M7.76 7.76L4.93 4.93" />
				</svg>
				Rejecting...
			{:else}
				<svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<line x1="18" y1="6" x2="6" y2="18" />
					<line x1="6" y1="6" x2="18" y2="18" />
				</svg>
				Reject
			{/if}
		</button>
	</div>

	<!-- Reject confirmation inline dialog -->
	{#if showRejectConfirm}
		<div
			role="alertdialog"
			aria-labelledby="reject-confirm-title"
			class="rounded-lg border border-red-200 bg-red-50 dark:border-red-800 dark:bg-red-950/20 px-3 py-2"
		>
			<p id="reject-confirm-title" class="text-sm font-medium text-red-800 dark:text-red-300">
				Reject these changes?
			</p>
			<p class="text-xs text-red-600 dark:text-red-400 mt-0.5">
				This will discard all proposed changes. This cannot be undone.
			</p>
			<div class="flex items-center gap-2 mt-2">
				<button
					onclick={handleRejectConfirm}
					class="btn-pill btn-pill-danger btn-pill-xs"
				>
					Confirm Reject
				</button>
				<button
					onclick={() => (showRejectConfirm = false)}
					class="btn-pill btn-pill-ghost btn-pill-xs"
				>
					Cancel
				</button>
			</div>
		</div>
	{/if}

	<!-- Error message -->
	{#if error}
		<div class="rounded-md bg-red-50 dark:bg-red-950/20 border border-red-200 dark:border-red-800 px-3 py-2">
			<p class="text-xs text-red-600 dark:text-red-400">{error}</p>
		</div>
	{/if}
</div>
