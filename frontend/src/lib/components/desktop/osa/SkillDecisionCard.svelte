<!--
	SkillDecisionCard.svelte
	SORX temperature-gated decision card shown when OSA wants to execute a skill.
	COLD skills auto-run. WARM skills need a quick confirm. HOT skills need full approval.
	Follows the ApproveReject.svelte pattern for consistency.
-->
<script lang="ts">
	import type { SkillExecution } from '$lib/types/skills';
	import {
		tierLabels,
		tierColors,
		temperatureLabels,
		temperatureColors,
		categoryIcons
	} from '$lib/types/skills';

	interface Props {
		execution: SkillExecution;
		onApprove?: (execution: SkillExecution) => void | Promise<void>;
		onReject?: (execution: SkillExecution) => void | Promise<void>;
		disabled?: boolean;
	}

	let { execution, onApprove, onReject, disabled = false }: Props = $props();

	let isApproving = $state(false);
	let isRejecting = $state(false);
	let showRejectConfirm = $state(false);
	let error = $state<string | null>(null);

	let isBusy = $derived(isApproving || isRejecting);
	let isCold = $derived(execution.temperature === 'cold');
	let isHot = $derived(execution.temperature === 'hot');

	async function handleApprove() {
		isApproving = true;
		error = null;
		try {
			await onApprove?.(execution);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to approve';
		} finally {
			isApproving = false;
		}
	}

	function handleRejectClick() {
		if (isHot) {
			showRejectConfirm = true;
		} else {
			handleRejectConfirm();
		}
	}

	async function handleRejectConfirm() {
		showRejectConfirm = false;
		isRejecting = true;
		error = null;
		try {
			await onReject?.(execution);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to reject';
		} finally {
			isRejecting = false;
		}
	}
</script>

<div
	class="skill-decision-card rounded-xl border bg-white p-4 shadow-sm dark:bg-gray-900
		{isHot
			? 'border-red-200 dark:border-red-800'
			: execution.temperature === 'warm'
				? 'border-amber-200 dark:border-amber-800'
				: 'border-gray-200 dark:border-gray-700'}"
	role="alertdialog"
	aria-labelledby="decision-title-{execution.id}"
>
	<!-- Header: Skill Name + Temperature Badge -->
	<div class="mb-3 flex items-start justify-between">
		<div class="flex items-center gap-2">
			<!-- Skill icon -->
			<div class="flex h-8 w-8 items-center justify-center rounded-lg
				{isHot ? 'bg-red-100 dark:bg-red-900/30' : execution.temperature === 'warm' ? 'bg-amber-100 dark:bg-amber-900/30' : 'bg-gray-100 dark:bg-gray-800'}">
				<svg class="h-4 w-4 {isHot ? 'text-red-600 dark:text-red-400' : execution.temperature === 'warm' ? 'text-amber-600 dark:text-amber-400' : 'text-gray-600 dark:text-gray-400'}" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
					<path d={categoryIcons[execution.skill.category]} />
				</svg>
			</div>
			<div>
				<h3 id="decision-title-{execution.id}" class="font-mono text-sm font-semibold text-gray-900 dark:text-white">
					{execution.skill.name}
				</h3>
				<span class="text-[10px] text-gray-500 dark:text-gray-400">
					{execution.action}
				</span>
			</div>
		</div>

		<!-- Temperature + Tier badges -->
		<div class="flex items-center gap-1.5">
			<span class="rounded-full px-2 py-0.5 text-[10px] font-semibold {temperatureColors[execution.temperature]}">
				{temperatureLabels[execution.temperature]}
			</span>
			<span class="rounded-full border px-2 py-0.5 text-[10px] font-medium {tierColors[execution.skill.tier]}">
				{tierLabels[execution.skill.tier]}
			</span>
		</div>
	</div>

	<!-- Reasoning (why OSA wants to execute this) -->
	{#if execution.reasoning}
		<div class="mb-3 rounded-lg bg-gray-50 px-3 py-2 dark:bg-gray-800/50">
			<p class="text-xs leading-relaxed text-gray-600 dark:text-gray-300">
				{execution.reasoning}
			</p>
		</div>
	{/if}

	<!-- HOT warning -->
	{#if isHot}
		<div class="mb-3 rounded-md border border-red-200 bg-red-50 px-3 py-2 dark:border-red-800 dark:bg-red-950/20">
			<p class="text-xs font-medium text-red-700 dark:text-red-300">
				This action sends data externally and cannot be undone.
			</p>
		</div>
	{/if}

	<!-- Action Buttons -->
	{#if !isCold}
		<div class="flex flex-col gap-2">
			<div class="flex items-center gap-2">
				<button
					onclick={handleApprove}
					disabled={isBusy || disabled}
					aria-label="Approve skill execution"
					aria-busy={isApproving}
					class="flex flex-1 items-center justify-center gap-1.5 btn-pill btn-pill-xs disabled:opacity-50
						{isHot
							? 'btn-pill-danger'
							: 'btn-pill-primary'}"
				>
					{#if isApproving}
						<svg class="h-3.5 w-3.5 animate-spin" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<path d="M12 2v4m0 12v4m-7.07-3.93l2.83-2.83m8.48-8.48l2.83-2.83M2 12h4m12 0h4m-3.93 7.07l-2.83-2.83M7.76 7.76L4.93 4.93" />
						</svg>
						Running...
					{:else}
						<svg class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<polyline points="20 6 9 17 4 12" />
						</svg>
						{isHot ? 'Approve & Execute' : 'Confirm'}
					{/if}
				</button>

				<button
					onclick={handleRejectClick}
					disabled={isBusy || disabled}
					aria-label="Reject skill execution"
					aria-busy={isRejecting}
					class="btn-pill btn-pill-secondary btn-pill-xs flex items-center gap-1.5"
				>
					{#if isRejecting}
						<svg class="h-3.5 w-3.5 animate-spin" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<path d="M12 2v4m0 12v4m-7.07-3.93l2.83-2.83m8.48-8.48l2.83-2.83M2 12h4m12 0h4m-3.93 7.07l-2.83-2.83M7.76 7.76L4.93 4.93" />
						</svg>
					{:else}
						<svg class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<line x1="18" y1="6" x2="6" y2="18" />
							<line x1="6" y1="6" x2="18" y2="18" />
						</svg>
					{/if}
					Deny
				</button>
			</div>

			<!-- Reject confirmation (HOT skills only) -->
			{#if showRejectConfirm}
				<div
					role="alertdialog"
					aria-labelledby="reject-confirm-{execution.id}"
					class="rounded-lg border border-gray-200 bg-gray-50 px-3 py-2 dark:border-gray-700 dark:bg-gray-800"
				>
					<p id="reject-confirm-{execution.id}" class="text-xs font-medium text-gray-700 dark:text-gray-300">
						Deny this skill execution?
					</p>
					<p class="mt-0.5 text-[10px] text-gray-500 dark:text-gray-400">
						OSA will not run {execution.skill.name} for this request.
					</p>
					<div class="mt-2 flex items-center gap-2">
						<button
							onclick={handleRejectConfirm}
							class="btn-pill btn-pill-primary btn-pill-xs"
						>
							Confirm Deny
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

			<!-- Error -->
			{#if error}
				<div class="rounded-md border border-red-200 bg-red-50 px-3 py-2 dark:border-red-800 dark:bg-red-950/20">
					<p class="text-xs text-red-600 dark:text-red-400">{error}</p>
				</div>
			{/if}
		</div>
	{:else}
		<!-- COLD: auto-executed, show status -->
		<div class="flex items-center gap-2 rounded-lg bg-emerald-50 px-3 py-2 dark:bg-emerald-900/20">
			<svg class="h-4 w-4 text-emerald-600 dark:text-emerald-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<polyline points="20 6 9 17 4 12" />
			</svg>
			<span class="text-xs font-medium text-emerald-700 dark:text-emerald-300">
				Auto-executed (cold temperature)
			</span>
		</div>
	{/if}
</div>
