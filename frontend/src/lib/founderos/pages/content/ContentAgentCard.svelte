<!-- A content agent you can run in place (FounderOS v1 ContentAgentCard, mock
     5c): the card is a lens row, the run control and the tool chips are
     controls. Run is FounderOS v1's AsyncButton: idle → spinner + elapsed →
     ✓ ok (or ✗ failed) → idle after 1.4s. Runs go to the same agent runtime
     /os/agents uses; an agent the bridge has no implementation for says so,
     and a refused run (FOUNDEROS_WRITES) shows its reason under the card. -->
<script lang="ts">
	import { onDestroy } from 'svelte';
	import { Wrench } from '$lib/founderos/icons';
	import { founderosFetch } from '$lib/founderos/api';
	import { Dot, Pressable } from '$lib/founderos/kit';
	import { prettyTool, type CrewMember } from './types';

	let { agent, lead = false }: { agent: CrewMember; lead?: boolean } = $props();

	let phase = $state<'idle' | 'busy' | 'done'>('idle');
	let failed = $state(false);
	let error = $state<string | null>(null);
	let since = 0;
	let elapsed = $state(0);
	let tick: ReturnType<typeof setInterval> | null = null;
	let back: ReturnType<typeof setTimeout> | null = null;

	const stopTick = () => {
		if (tick) clearInterval(tick);
		tick = null;
	};

	async function run() {
		if (!agent.runnable || phase !== 'idle') return;
		phase = 'busy';
		failed = false;
		error = null;
		since = Date.now();
		elapsed = 0;
		tick = setInterval(() => (elapsed = Math.max(0, Math.floor((Date.now() - since) / 1000))), 1000);
		try {
			await founderosFetch(`/agents/${encodeURIComponent(agent.id)}/run`, { method: 'POST' });
		} catch (e) {
			failed = true;
			error = e instanceof Error ? e.message : String(e);
		} finally {
			stopTick();
			phase = 'done';
			back = setTimeout(() => (phase = 'idle'), 1400);
		}
	}

	onDestroy(() => {
		stopTick();
		if (back) clearTimeout(back);
	});

	const active = $derived(agent.status === 'active');
</script>

<div data-part="agent-card" data-lens="r" class="ca-card bn-pressable is-row rounded-[10px] border p-4" class:ca-lead={lead}>
	<div class="flex items-start justify-between gap-3">
		<div class="min-w-0">
			<div class="flex items-center gap-2">
				<Dot state={active ? 'ok' : 'available'} pulse={active} />
				<span class="bn-text truncate text-[14px] font-bold">{agent.name}</span>
				{#if lead}
					<span class="ca-lead-tag bn-accent rounded-full border px-1.5 py-px font-mono text-[9px] uppercase tracking-wide">lead</span>
				{/if}
			</div>
			<div class="bn-dim mt-0.5 font-mono text-[10.5px]">{agent.role} · {agent.model}</div>
		</div>
		<div class="flex shrink-0 items-center gap-2">
			<span class="font-mono text-[10px] uppercase tracking-wide" style="color: {active ? 'var(--bn-ok)' : 'var(--bn-warn)'}">{agent.status}</span>
			<Pressable
				tone="secondary"
				onclick={run}
				disabled={!agent.runnable}
				aria-busy={phase === 'busy'}
				title={agent.runnable ? `Run ${agent.name}` : `${agent.name} is not on the bridge runtime yet`}
				class="disabled:cursor-not-allowed disabled:opacity-30"
			>
				{#if phase === 'busy'}
					<span class="ca-spin inline-block h-[10px] w-[10px] rounded-full border-[1.5px]"></span>
					running <span class="tabular-nums">{elapsed}s</span>
				{:else if phase === 'done'}
					<span class="bn-pop" style="color: {failed ? 'var(--bn-err)' : 'var(--bn-ok)'}">{failed ? '✗' : '✓'}</span>
					{failed ? 'failed' : 'ok'}
				{:else}▸ run{/if}
			</Pressable>
		</div>
	</div>
	<p class="bn-muted mt-2.5 text-[12px] leading-relaxed [text-wrap:pretty]">{agent.description}</p>
	{#if !agent.runnable}
		<p class="bn-dim mt-1.5 font-mono text-[10px]">not on the bridge runtime yet · run it from FounderOS v1</p>
	{/if}
	{#if failed && error}
		<p class="mt-1.5 font-mono text-[10.5px]" style="color: var(--bn-err)">{error}</p>
	{/if}
	{#if agent.tools.length > 0}
		<div class="mt-3 flex flex-wrap gap-1.5">
			{#each agent.tools as t (t)}
				<span data-lens="c" class="ca-tool bn-pressable is-dark bn-muted inline-flex items-center gap-1 rounded-full border px-2 py-0.5 font-mono text-[10px]">
					<Wrench size={10} />
					{prettyTool(t)}
				</span>
			{/each}
		</div>
	{/if}
</div>

<style>
	.ca-card {
		border-color: var(--bn-border);
		background: var(--bn-surface);
	}
	.ca-lead {
		border-color: var(--bn-accent-line, color-mix(in oklab, var(--bn-accent) 25%, transparent));
	}
	.ca-lead-tag {
		border-color: var(--bn-accent-line, color-mix(in oklab, var(--bn-accent) 25%, transparent));
	}
	.ca-tool {
		border-color: var(--bn-border);
		background: var(--bn-surface-2);
	}
	.ca-spin {
		border-color: var(--bn-ok);
		border-right-color: transparent;
		animation: bn-om-spin 0.8s linear infinite;
	}
</style>
