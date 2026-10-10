<!-- Concept mining (POST /pages/adscout/mine): probes expand the concept,
     Discovery answers with proven ads. Explicit credit spend on submit only. -->
<script lang="ts">
	import { LoaderCircle, Search } from '$lib/founderos/icons';
	import { founderosFetch } from '$lib/founderos/api';
	import CardGrid from './CardGrid.svelte';
	import Empty from './Empty.svelte';
	import type { MineResponse, WallAd } from './types';

	let {
		savedIds,
		onOpen,
		onSave,
		onResults
	}: { savedIds: Set<string>; onOpen: (ad: WallAd) => void; onSave: (ad: WallAd) => void; onResults: (ads: WallAd[]) => void } = $props();

	let concept = $state('');
	let minDays = $state(21);
	let format = $state<'all' | 'video' | 'image'>('all');
	let st = $state<{ phase: 'idle' } | { phase: 'running' } | { phase: 'error'; message: string } | ({ phase: 'done' } & MineResponse)>({ phase: 'idle' });

	async function run() {
		const q = concept.trim();
		if (q.length < 3 || st.phase === 'running') return;
		st = { phase: 'running' };
		try {
			const body = await founderosFetch<MineResponse>('/pages/adscout/mine', {
				method: 'POST',
				json: { concept: q, minDays, ...(format !== 'all' ? { format } : {}) }
			});
			st = { phase: 'done', ...body };
			onResults(body.winners);
		} catch (err) {
			st = { phase: 'error', message: err instanceof Error ? err.message : 'search failed' };
		}
	}
</script>

{#snippet chip(on: boolean, label: string, pick: () => void)}
	<button
		type="button"
		onclick={pick}
		aria-pressed={on}
		class="bn-pressable rounded-full border px-2.5 py-1 text-[11.5px] {on
			? 'border-[var(--bn-accent-line)] bg-[var(--bn-accent-soft)] text-[var(--bn-accent)]'
			: 'bn-dim ap-hover-muted border-[rgba(255,69,87,0.14)]'}">{label}</button
	>
{/snippet}

<div data-part="search">
	<label class="bn-muted block text-[12.5px]" for="ap-concept">Describe a concept: probes expand it, the database answers with proven ads</label>
	<div class="ap-glow mt-2 flex items-center gap-2 rounded-[12px] border border-[rgba(255,69,87,0.3)] bg-black/45 p-1.5">
		<Search class="ml-2 h-4 w-4 shrink-0" style="color: var(--bn-accent)" strokeWidth={1.7} />
		<input
			id="ap-concept"
			bind:value={concept}
			onkeydown={(e) => e.key === 'Enter' && run()}
			placeholder="founders replacing staff with AI agents"
			class="bn-text ap-ph min-w-0 flex-1 bg-transparent px-1 py-2 text-[14px] outline-none"
		/>
		<button type="button" onclick={run} disabled={st.phase === 'running' || concept.trim().length < 3} class="bn-pressable ap-btn ap-btn-primary flex items-center gap-1.5 text-[12.5px]">
			{#if st.phase === 'running'}<LoaderCircle class="ap-spin h-4 w-4" strokeWidth={1.7} />{:else}<Search class="h-4 w-4" strokeWidth={1.7} />{/if}
			Search ads
		</button>
	</div>
	<div class="mt-2 flex flex-wrap items-center gap-1.5">
		<span class="bn-dim mr-1 text-[11px]">Proven for</span>
		{#each [21, 30, 60] as d (d)}{@render chip(minDays === d, `${d}+ days`, () => (minDays = d))}{/each}
		<span class="bn-dim ml-3 mr-1 text-[11px]">Format</span>
		{#each ['all', 'video', 'image'] as const as f (f)}{@render chip(format === f, f, () => (format = f))}{/each}
		<span class="bn-dim ml-auto text-[11px]">~5 API credits per search</span>
	</div>

	<div class="mt-4">
		{#if st.phase === 'idle'}
			<Empty line1="Search the 100M-ad database by idea, not by brand." line2="Results are filtered to ads still paying off after your chosen threshold: the honest proof filter." />
		{:else if st.phase === 'running'}
			<p class="bn-dim py-8 text-center text-[12.5px]">expanding concept → probing discovery → filtering to proven ads…</p>
		{:else if st.phase === 'error'}
			<p class="py-8 text-center text-[12px] text-[#f04e52]">{st.message}</p>
		{:else}
			<div class="bn-dim mb-3 text-[11.5px] leading-relaxed">
				{#each st.probes as p (p.query)}<span class="mr-3">“{p.query}” → {p.results}</span>{/each}
				<span class="bn-muted">· {st.pooled} pooled · {st.apiCalls} credits</span>
			</div>
			<CardGrid ads={st.winners} {savedIds} {onOpen} {onSave} />
		{/if}
	</div>
</div>
