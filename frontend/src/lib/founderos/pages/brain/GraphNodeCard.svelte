<!-- Mock 1d, the node hover card (FounderOS v1 components/GraphNodeCard.tsx):
     240px, pinned bottom-left of the graph, naming the node the pointer last
     touched. "open note" jumps the graph to that node's detail; "brain › query"
     really asks the engines (GET /pages/brain/query?q=<label>) and lists what
     came back. The query button is AsyncButton's idle → busy → done ride. -->
<script lang="ts">
	import { X } from '$lib/founderos/icons';
	import { Pressable } from '$lib/founderos/kit';
	import { founderosFetch } from '$lib/founderos/api';
	import type { BrainHit, BrainQueryBody } from './types';

	let { label, kind, links, sub, onOpen, onDismiss }: { label: string; kind: string; links: number; sub?: string; onOpen: () => void; onDismiss: () => void } = $props();

	let hits = $state<BrainHit[] | null>(null);
	let failed = $state(false);
	let phase = $state<'idle' | 'busy' | 'done'>('idle');

	async function query() {
		if (phase !== 'idle') return;
		phase = 'busy';
		failed = false;
		hits = null;
		try {
			const body = await founderosFetch<BrainQueryBody>(`/pages/brain/query?q=${encodeURIComponent(label)}`);
			hits = body.results ?? [];
		} catch {
			failed = true;
		} finally {
			phase = 'done';
			setTimeout(() => (phase = 'idle'), 1400);
		}
	}
</script>

<div class="kg-enter kg-bs kg-bg w-60 rounded-[10px] border px-3 py-2.5" data-part="node-card">
	<div class="flex items-start gap-2">
		<div class="min-w-0 flex-1">
			<div class="bn-dim font-mono text-[9px] uppercase tracking-[0.28em]">node · hover card</div>
			<div class="bn-text mt-1 truncate text-[12px] font-bold">{label}</div>
			<div class="bn-dim mt-0.5 truncate font-mono text-[10px]">{sub ? `${sub} · ` : ''}{kind} · {links} link{links === 1 ? '' : 's'}</div>
		</div>
		<button type="button" onclick={onDismiss} title="Dismiss" aria-label="Dismiss node card" data-lens="c" class="bn-pressable is-dark bn-dim -mr-1 -mt-1 grid h-5 w-5 shrink-0 place-items-center rounded-[6px]">
			<X size={12} />
		</button>
	</div>

	<div class="mt-2.5 flex items-center gap-1.5">
		<button type="button" onclick={onOpen} data-lens="c" class="bn-pressable is-dark kg-b kg-bg bn-muted inline-flex h-[22px] items-center rounded-[6px] border px-2 font-mono text-[10px] font-semibold">open note</button>
		<Pressable tone="secondary" class="!h-[22px] !px-2 !text-[10px]" onclick={query} aria-busy={phase === 'busy'} data-part="node-query">
			{#if phase === 'idle'}brain › query{:else if phase === 'busy'}<span class="kg-spin inline-block h-[10px] w-[10px] rounded-full border-[1.5px]"></span>querying{:else if failed}<span class="kg-err">✗</span> failed{:else}<span style="color: var(--bn-ok)">✓</span> hits{/if}
		</Pressable>
	</div>

	{#if hits}
		<div class="kg-b mt-2 border-t pt-2">
			{#if hits.length === 0}
				<p class="bn-dim font-mono text-[10px]">no hits in the engines</p>
			{:else}
				<ul class="space-y-1">
					{#each hits.slice(0, 3) as h (h.uri + h.title)}<li class="bn-muted truncate font-mono text-[10px]">· {h.title}</li>{/each}
				</ul>
			{/if}
		</div>
	{/if}
</div>

<style>
	.kg-spin {
		border-color: var(--bn-ok);
		border-right-color: transparent;
		animation: kg-spin 0.8s linear infinite;
	}
	@keyframes kg-spin {
		to {
			transform: rotate(360deg);
		}
	}
</style>
