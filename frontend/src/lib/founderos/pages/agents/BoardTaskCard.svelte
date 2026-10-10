<!-- One task in a board lane (FounderOS v1 components/BoardTaskCard.tsx). Click
     it to Approve or Dismiss: a real write to the decision store, keyed
     board:<issueId> and bound to the task's updatedAt. -->
<script lang="ts">
	import { founderosFetch } from '$lib/founderos/api';
	import { boardTaskDecisionId } from './board';
	import type { Decision, PaperclipIssue } from './types';

	let { issue, working = false, decision }: { issue: PaperclipIssue; working?: boolean; decision?: Decision } = $props();

	let open = $state(false);
	let busy = $state(false);
	let override = $state<{ kind: Decision['decision'] | null; at: string | undefined } | null>(null);
	// a fresh decision from the poll wins over a stale local override
	const local = $derived(override && override.at === decision?.decidedAt ? override.kind : (decision?.decision ?? null));

	async function decide(kind: Decision['decision']) {
		if (busy) return;
		busy = true;
		override = { kind, at: decision?.decidedAt };
		try {
			await founderosFetch('/pages/board/deliverables/decision', {
				method: 'POST',
				json: { id: boardTaskDecisionId(issue.id), decision: kind, decidedRevision: issue.updatedAt ?? '' }
			});
		} catch {
			override = null;
		} finally {
			busy = false;
			open = false;
		}
	}

	const approved = $derived(local === 'approved');
	const dismissed = $derived(local === 'dismissed');
	const toggle = () => (open = !open);
</script>

<div
	role="button"
	tabindex="0"
	aria-expanded={open}
	title={issue.title}
	onclick={toggle}
	onkeydown={(e) => {
		if (e.key === 'Enter' || e.key === ' ') {
			e.preventDefault();
			toggle();
		}
	}}
	data-lens="r"
	class="bn-pressable is-row bn-enter bn-task cursor-pointer border px-2 py-1.5 text-left"
	class:bn-task-live={working && !approved}
	style:border-color={approved ? 'color-mix(in oklab, var(--bn-ok) 35%, transparent)' : working ? undefined : 'var(--bn-border)'}
	style:opacity={dismissed ? 0.45 : 1}
>
	<div class="bn-text line-clamp-2 text-[10.5px] leading-snug">{issue.title}</div>
	<div class="bn-dim mt-0.5 flex items-center gap-1 font-mono text-[9px]">
		{#if issue.assigneeName}
			<span class="h-1 w-1 shrink-0" class:bn-task-live-dot={working && !approved} style:background={working && !approved ? 'var(--bn-ok)' : 'var(--bn-text-2)'}></span>
			<span class="truncate">{issue.assigneeName}</span>
		{/if}
		{#if approved}<span class="ml-auto shrink-0" style:color="var(--bn-ok)">✓ approved</span>{/if}
		{#if dismissed}<span class="ml-auto shrink-0">dismissed</span>{/if}
	</div>
	{#if open}
		<!-- svelte-ignore a11y_no_static_element_interactions, a11y_click_events_have_key_events -->
		<div class="bn-enter mt-1.5 flex gap-1" onclick={(e) => e.stopPropagation()}>
			<button type="button" data-lens="c" disabled={busy} onclick={() => decide('approved')} class="bn-pressable is-primary bn-btn-primary h-[22px] flex-1 border font-mono text-[9.5px] font-bold">Approve</button>
			<button type="button" data-lens="c" disabled={busy} onclick={() => decide('dismissed')} class="bn-pressable is-dark bn-btn-ghost h-[22px] flex-1 border font-mono text-[9.5px] font-semibold">Dismiss</button>
		</div>
	{/if}
</div>

<style>
	.bn-task {
		border-radius: var(--bn-r-ctl, 6px);
		background: var(--bn-bg);
	}
	.bn-btn-primary {
		border-color: var(--bn-text);
		border-radius: var(--bn-r-ctl, 6px);
		color: var(--bn-accent-ink);
	}
	.bn-btn-ghost {
		border-color: var(--bn-border);
		border-radius: var(--bn-r-ctl, 6px);
		background: var(--bn-bg);
		color: var(--bn-text-2);
	}
	button:disabled {
		opacity: 0.4;
	}
</style>
