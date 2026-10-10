<!-- The worker pill (components/OrgWorkerPill.tsx, mock 1f): collapsed it is a
     task pill; clicking opens the role, a run control and a link to the
     roster. Run posts to the bridge runtime; a failure is shown, not hidden. -->
<script lang="ts">
	import OrgDot from './OrgDot.svelte';
	import { rosterDot } from './org';
	import { founderosFetch } from '$lib/founderos/api';
	import OrgAsyncButton from './OrgAsyncButton.svelte';
	import VentureDots from './VentureDots.svelte';
	import type { Agent, Venture } from './types';

	let { agent, dim = false, ventures }: { agent: Agent; dim?: boolean; ventures: Venture[] } = $props();

	let open = $state(false);
	let error = $state<string | null>(null);
	let summary = $state<string | null>(null);

	/** POST the run; a refusal or a failed run throws so the button reads ✗. */
	async function go() {
		error = null;
		summary = null;
		try {
			const res = await founderosFetch<{ ok: boolean; summary: string }>(`/agents/${agent.id}/run`, { method: 'POST' });
			if (!res?.ok) throw new Error(res?.summary ?? 'run failed');
			summary = res.summary ?? null;
		} catch (err) {
			error = err instanceof Error ? err.message : String(err);
			throw err;
		}
	}

	function toggle() {
		open = !open;
	}
	function onKey(e: KeyboardEvent) {
		if (e.key === 'Enter' || e.key === ' ') {
			e.preventDefault();
			toggle();
		}
	}
</script>

<div
	role="button"
	tabindex="0"
	aria-expanded={open}
	data-agent={agent.id}
	data-dim={dim}
	title={`${agent.role} — ${agent.description}`}
	onclick={toggle}
	onkeydown={onKey}
	data-lens="r"
	class="bn-pressable is-row bn-border border text-left {open ? 'rounded-[8px]' : 'rounded-full'}"
	class:opacity-20={dim}
	style="background: var(--bn-bg)"
>
	<div class="flex items-center gap-1.5 px-2.5 py-1.5">
		<OrgDot look={rosterDot(agent.status)} />
		<span class="bn-text truncate text-[10px] font-medium">{agent.name}</span>
		<VentureDots {ventures} agentId={agent.id} />
	</div>
	{#if open}
		<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
		<div class="bn-enter bn-muted px-2.5 pb-2 pl-5 text-[9.5px] leading-relaxed" onclick={(e) => e.stopPropagation()}>
			{agent.role}
			<div class="mt-1.5 flex items-center gap-1.5">
				<OrgAsyncButton run={go} tone="secondary" busyLabel="running" doneLabel="ok" showElapsed>▸ run</OrgAsyncButton>
				<a href="/os/agents" data-lens="c" class="bn-pressable is-dark bn-border bn-muted inline-flex h-[20px] items-center rounded-[6px] border px-2 font-mono text-[9.5px] font-semibold" style="background: var(--bn-bg)">chat</a>
			</div>
			{#if error}
				<p class="mt-1 font-mono text-[9px]" style="color: var(--bn-err)">✕ {error}</p>
			{:else if summary}
				<p class="bn-dim mt-1 font-mono text-[9px]">{summary}</p>
			{/if}
		</div>
	{/if}
</div>
