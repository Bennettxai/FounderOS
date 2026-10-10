<!-- The satellites around the graph (FounderOS v1 components/BrainSatellites.tsx):
     identity + page/folder counters top right, the ask bar top left with its
     retrieval notice, a footer ticker bottom right. Facts come from
     /pages/brain/satellites; the ask bar reads /pages/brain/query (engine pool of
     15, reranked when the cross-encoder answers, top 3). They sit inside the
     canvas box (below the view tabs, left of the directory aside) and fade out
     while a node card is open so nothing covers what is being read. -->
<script lang="ts">
	import { onDestroy, onMount } from 'svelte';
	import { founderosFetch } from '$lib/founderos/api';
	import type { BrainHit, BrainQueryBody, BrainSatellites } from './types';

	let { quiet = false }: { quiet?: boolean } = $props();

	type Phase = 'idle' | 'retrieve' | 'answer';

	let data = $state<BrainSatellites | null>(null);
	let failed = $state<string | null>(null);
	let q = $state('');
	let phase = $state<Phase>('idle');
	let hits = $state<BrainHit[]>([]);
	let ranked = $state<'rerank' | 'provider' | undefined>(undefined);
	let unsearched = $state<string[]>([]);
	let error = $state<string | null>(null);
	let timer: ReturnType<typeof setTimeout> | null = null;

	onMount(() => {
		founderosFetch<BrainSatellites>('/pages/brain/satellites')
			.then((d) => (data = d))
			.catch((e: unknown) => (failed = e instanceof Error ? e.message : 'unreachable'));
	});
	onDestroy(() => {
		if (timer) clearTimeout(timer);
	});

	// Only the newest ask may land; a slower earlier one is dropped.
	let askSeq = 0;
	async function ask(e: SubmitEvent) {
		e.preventDefault();
		const query = q.trim();
		if (query.length < 2) return;
		const seq = ++askSeq;
		phase = 'retrieve';
		error = null;
		try {
			const body = await founderosFetch<BrainQueryBody>(`/pages/brain/query?q=${encodeURIComponent(query)}`);
			if (seq !== askSeq) return;
			hits = body.results ?? [];
			unsearched = body.failedWorkspaces ?? [];
			ranked = body.ranked;
			phase = 'answer';
			if (timer) clearTimeout(timer);
			timer = setTimeout(() => (phase = 'idle'), 14_000);
		} catch (err) {
			if (seq !== askSeq) return;
			error = err instanceof Error ? err.message : String(err);
			phase = 'idle';
		}
	}

	const mounted = $derived(data?.mounted ?? false);
	const statusWord = $derived(
		phase === 'retrieve' ? 'retrieving' : phase === 'answer' ? 'answered' : mounted ? 'ready' : data ? 'not mounted' : failed ? 'unreachable' : 'loading'
	);
	const answer = $derived(hits[0]?.snippet || hits[0]?.title || (phase === 'answer' ? 'Nothing in the engines answers that.' : ''));
	const showNotice = $derived(phase === 'answer' || !!error);
</script>

<div
	data-part="satellites"
	data-quiet={quiet || undefined}
	class="pointer-events-none absolute inset-x-0 bottom-0 top-[47px] z-20 lg:right-[18.75rem]"
	style="opacity: {quiet ? 0 : 1}; transition: opacity 0.24s ease"
>
	<div class="bn-rise pointer-events-none absolute right-[124px] top-3 z-20 flex items-center gap-1.5" style="--rise-i: 1" data-part="sat-chips">
		<span class="bn-sat-chip" title={data?.statusLine ?? failed ?? 'reading the engines'}>
			<span class="h-1.5 w-1.5 rounded-full" style="background: {mounted ? 'var(--bn-accent)' : data || failed ? 'var(--bn-warn)' : 'var(--bn-text-3)'}"></span>
			Optimal Engine
			<span class="bn-dim">{data ? (mounted ? 'connected' : 'not mounted') : failed ? 'unreachable' : 'reading'}</span>
		</span>
		{#if data && mounted}
			<span class="bn-sat-chip" data-part="pages"><span class="bn-text tabular-nums">{data.pages.toLocaleString('en-US')}</span><span class="bn-dim">pages</span></span>
			<span class="bn-sat-chip" data-part="folders"><span class="bn-text tabular-nums">{data.folders}</span><span class="bn-dim">{data.folders === 1 ? 'workspace' : 'workspaces'}</span></span>
		{/if}
	</div>

	<form onsubmit={ask} class="bn-rise absolute left-3 top-3 z-20 w-[340px] max-w-[60%]" class:pointer-events-none={quiet} class:pointer-events-auto={!quiet} style="--rise-i: 2">
		<div class="bn-glass flex items-center gap-2 rounded-[5px] border px-2.5 py-1" class:is-live={phase === 'retrieve'}>
			<span class="bn-accent shrink-0 font-mono text-[10.5px]">brain ›</span>
			<input
				bind:value={q}
				disabled={data !== null && !mounted}
				placeholder={mounted || !data ? 'ask the brain' : 'no engine pages mounted'}
				spellcheck="false"
				aria-label="Ask the brain"
				class="bn-text min-h-[16px] w-full flex-1 border-0 bg-transparent font-mono text-[10.5px] outline-none"
			/>
			<span class="bn-dim shrink-0 font-mono text-[9.5px]" data-part="ask-status">{statusWord}</span>
		</div>
		<div
			class="bn-glass mt-1.5 rounded-[5px] border px-2.5 py-1.5"
			aria-live="polite"
			data-part="notice"
			style="opacity: {showNotice ? 1 : 0}; transform: translateY({showNotice ? 0 : -6}px); pointer-events: {showNotice ? 'auto' : 'none'}; transition: opacity 0.24s ease, transform 0.24s ease"
		>
			{#if error}
				<div class="bn-err-text font-mono text-[10px]">{error}</div>
			{:else if showNotice}
				<div class="flex items-baseline gap-2">
					<span class="bn-notice-dot h-1.5 w-1.5 shrink-0 rounded-full"></span>
					<span class="bn-muted shrink-0 font-mono text-[10px]">{hits.length > 0 ? `retrieved from ${hits.length} notes${ranked === 'rerank' ? ' · reranked' : ''}` : 'retrieval'}</span>
					<span class="bn-text min-w-0 flex-1 truncate text-[12px]">{answer}</span>
				</div>
				{#if unsearched.length > 0}
					<div class="bn-warn-text mt-1 pl-[14px] font-mono text-[9.5px]" title="those engines did not answer">partial · {unsearched.join(', ')} not searched</div>
				{/if}
				{#if hits.length > 0}
					<div class="mt-1 flex gap-1 overflow-hidden pl-[14px]">
						{#each hits.slice(0, 3) as h (`${h.source}-${h.title}`)}
							<span class="bn-hit-pill bn-muted shrink-0 truncate rounded-[5px] border px-1.5 py-0.5 font-mono text-[9px]" title={`${h.workspace} · ${h.engine}`}>{h.title || h.source}</span>
						{/each}
					</div>
				{/if}
			{/if}
		</div>
	</form>

	<div class="bn-dim pointer-events-none absolute bottom-2 right-7 z-20 font-mono text-[9px]" title="pool 15 → rerank → top 3">live retrieval</div>
</div>

<style>
	.bn-sat-chip {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		border: 1px solid var(--bn-border-strong);
		border-radius: 5px;
		background: color-mix(in oklab, var(--bn-bg) 80%, transparent);
		padding: 4px 8px;
		font-family: var(--bn-font);
		font-size: 10.5px;
		color: var(--bn-text-2);
		backdrop-filter: blur(6px);
	}
	.bn-glass {
		background: color-mix(in oklab, var(--bn-bg) 84%, transparent);
		border-color: var(--bn-border-strong);
		backdrop-filter: blur(10px);
	}
	.bn-glass.is-live {
		border-color: color-mix(in oklab, var(--bn-accent) 55%, transparent);
	}
	.bn-glass input::placeholder {
		color: var(--bn-text-3);
	}
	.bn-notice-dot {
		background: var(--bn-accent);
		transform: translateY(-2px);
	}
	.bn-hit-pill {
		border-color: color-mix(in oklab, var(--bn-text) 14%, transparent);
	}
	.bn-warn-text {
		color: var(--bn-warn);
	}
	.bn-err-text {
		color: var(--bn-err);
	}
</style>
