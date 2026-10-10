<!-- The AI Head card (components/ConductorCard.tsx): the Conductor with its
     chat pill. Send broadcasts to every agent via POST /agents/broadcast;
     the replies expand below. -->
<script lang="ts">
	import { founderosFetch } from '$lib/founderos/api';
	import ConductorEmblem from '$lib/founderos/kit/ConductorEmblem.svelte';
	import OrgAsyncButton from './OrgAsyncButton.svelte';
	import type { Agent, Broadcast } from './types';

	let {
		conductor,
		agentNames,
		initialBroadcast
	}: { conductor: Agent; agentNames: Record<string, string>; initialBroadcast: Broadcast | null } = $props();

	let message = $state('');
	let sending = $state(false);
	let error = $state<string | null>(null);
	let broadcast = $state<Broadcast | null>(null);
	let showReplies = $state(false);
	const shown = $derived(broadcast ?? initialBroadcast);
	const okCount = $derived(shown ? shown.replies.filter((r) => r.ok).length : 0);

	async function send() {
		const text = message.trim();
		if (!text || sending) return;
		sending = true;
		error = null;
		try {
			broadcast = await founderosFetch<Broadcast>('/agents/broadcast', { method: 'POST', json: { message: text } });
			showReplies = true;
			message = '';
		} catch (err) {
			error = err instanceof Error ? err.message : String(err);
		} finally {
			sending = false;
		}
	}
</script>

<div data-part="conductor-card" data-lens="r" class="bn-pressable is-row bn-surface w-[340px] rounded-[12px] border p-4" style="border-color: var(--bn-border-strong)">
	<div class="bn-dim text-center text-[10px] uppercase tracking-[0.25em]">AI Head</div>
	<div class="mt-2 flex justify-center">
		<ConductorEmblem size={62} thinking={sending} />
	</div>
	<div class="bn-text mt-2 text-center text-sm font-bold tracking-[0.2em]">CONDUCTOR</div>
	<div class="bn-dim text-center text-[10px]">super agent · {conductor.instance} runtime until the dedicated host lands</div>

	<div class="mt-3 flex gap-1.5">
		<input
			bind:value={message}
			onkeydown={(e) => e.key === 'Enter' && send()}
			placeholder="Chat with Conductor — reaches every agent"
			disabled={sending}
			class="bn-border bn-text min-w-0 flex-1 rounded-full border px-3 py-1.5 text-xs focus:outline-none"
			style="background: var(--bn-bg)"
		/>
		<!-- three-state broadcast control: Send → spinner → ✓ (✗ if it threw) -->
		<OrgAsyncButton run={send} disabled={!message.trim()} failed={error !== null} doneLabel="" failLabel="" class="shrink-0 rounded-full!">Send</OrgAsyncButton>
	</div>
	{#if error}<p class="bn-muted mt-1.5 text-[11px]">⚠ {error}</p>{/if}

	<div class="mt-3 grid grid-cols-3 gap-1">
		{#each ['Broadcast', 'Orchestration', 'Instances'] as cap (cap)}
			<span class="rounded-full px-2 py-1 text-center text-[9px] font-semibold uppercase tracking-wider" style="background: var(--bn-text); color: var(--bn-bg)">{cap}</span>
		{/each}
	</div>
	<div class="bn-muted mt-2 rounded-[6px] px-2 py-1 text-center text-[9px] uppercase tracking-[0.2em]" style="background: var(--bn-surface-2)">Agent Tools</div>
	<div data-part="conductor-tools" class="mt-1.5 flex justify-center gap-1">
		{#each conductor.tools as tool (tool)}
			<span class="bn-border bn-muted rounded-[4px] border px-1.5 py-0.5 font-mono text-[9px]">{tool}</span>
		{/each}
	</div>

	{#if shown}
		<div data-part="broadcast-footer" class="bn-border mt-3 border-t pt-2">
			<button type="button" onclick={() => (showReplies = !showReplies)} class="bn-pressable flex w-full items-baseline justify-between gap-2 text-left">
				<span class="bn-muted truncate text-[11px]">«{shown.message}»</span>
				<span class="shrink-0 text-[10px]" style="color: var(--bn-ok)"
					>{okCount}/{shown.replies.length} ok <span class="bn-dim">{showReplies ? '▾' : '▸'}</span></span
				>
			</button>
			{#if showReplies}
				<ul class="bn-enter mt-2 max-h-56 space-y-1 overflow-y-auto pr-1">
					{#each shown.replies as reply, i (reply.id ?? `${reply.agentId}-${i}`)}
						<li class="flex items-start gap-1.5 rounded-[6px] px-2 py-1.5" style="background: var(--bn-surface-2)">
							<span
								class="mt-1 h-1 w-1 shrink-0 rounded-full"
								style={reply.ok ? 'background: var(--bn-text)' : 'border: 1px solid var(--bn-text-3)'}
							></span>
							<div class="min-w-0">
								<span class="bn-text text-[10px] font-semibold">{agentNames[reply.agentId] ?? reply.agentId}</span>
								<span class="bn-muted break-words text-[10px] leading-relaxed"> — {reply.reply}</span>
							</div>
						</li>
					{/each}
				</ul>
			{/if}
		</div>
	{/if}
</div>
