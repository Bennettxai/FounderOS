<!-- One persona at a time: click (or ← / →) through all of them (FounderOS v1
     PersonasViewer). The card sits left, its knowledge graph right. -->
<script lang="ts">
	import { ChevronLeft, ChevronRight } from '$lib/founderos/icons';
	import { pad2, step } from './graph';
	import PersonaBrainGraph from './PersonaBrainGraph.svelte';
	import PersonaCard from './PersonaCard.svelte';
	import type { Persona } from './types';

	let { personas }: { personas: Persona[] } = $props();

	let i = $state(0);
	const n = $derived(personas.length);
	const current = $derived(personas[Math.min(i, Math.max(0, n - 1))]);
	const agentCount = $derived(current ? current.pillars.reduce((s, p) => s + p.agents.length, 0) : 0);
	const go = (dir: number) => (i = step(i, dir, n));

	function onKey(e: KeyboardEvent) {
		const tag = (e.target as HTMLElement | null)?.tagName;
		if (tag === 'INPUT' || tag === 'TEXTAREA') return;
		if (e.key === 'ArrowLeft') go(-1);
		else if (e.key === 'ArrowRight') go(1);
	}

	const btn = 'pv-step bn-pressable flex h-9 w-9 items-center justify-center rounded-[var(--bn-r-chip)] border';
</script>

<svelte:window onkeydown={onKey} />

{#if current}
	<div>
		<div class="mb-4 flex items-center gap-3">
			<div class="flex shrink-0 items-center gap-1.5">
				<button type="button" onclick={() => go(-1)} aria-label="Previous persona" class={btn}><ChevronLeft class="h-4 w-4" /></button>
				<button type="button" onclick={() => go(1)} aria-label="Next persona" class={btn}><ChevronRight class="h-4 w-4" /></button>
			</div>
			<div class="min-w-0">
				<div class="bn-dim font-mono text-[11px]"><span style={`color: ${current.accent}`}>{pad2(i + 1)}</span> / {pad2(n)}</div>
				<div class="bn-text truncate text-[13px] font-semibold">{current.name}</div>
			</div>
			<div class="ml-auto flex flex-wrap justify-end gap-1.5">
				{#each personas as p, idx (p.id)}
					<button
						type="button"
						onclick={() => (i = idx)}
						title={p.name}
						class="pv-rail bn-pressable flex items-center gap-1.5 rounded-[var(--bn-r-chip)] border px-2 py-1 font-mono text-[10px] {idx === i ? 'is-on' : ''}"
					>
						<span class="h-2 w-2 rounded-full" style={`background: ${idx === i ? p.accent : 'var(--bn-text-3)'}`}></span>
						{pad2(idx + 1)}
					</button>
				{/each}
			</div>
		</div>

		{#key current.id}
			<div class="persona-in grid gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(0,1.08fr)]">
				<PersonaCard p={current} />
				<div
					class="bn-card flex h-[600px] flex-col overflow-hidden rounded-[var(--bn-r-panel)] lg:sticky lg:top-4 lg:h-[calc(100dvh-11rem)] lg:min-h-[560px]"
					style={`border-top: 2px solid ${current.accent}`}
				>
					<div class="flex shrink-0 items-center justify-between border-b px-4 py-2.5 bn-border">
						<span class="bn-dim font-mono text-[9.5px] uppercase tracking-[0.16em]">Brain · knowledge graph</span>
						<span class="bn-dim font-mono text-[9.5px]">{current.pillars.length} depts · {agentCount} agents</span>
					</div>
					<div class="min-h-0 flex-1 p-2">
						<PersonaBrainGraph persona={current} />
					</div>
				</div>
			</div>
		{/key}
	</div>
{/if}

<style>
	/* Colours live here, not in Tailwind hover:/arbitrary utilities: the kit's
	   unlayered colour classes outrank the utilities layer, and border-[var()]
	   reads as a width. Prod: border -> border-strong + text on hover. */
	.pv-step {
		border-color: var(--bn-border);
		color: var(--bn-text-2);
	}
	.pv-step:hover {
		border-color: var(--bn-border-strong);
		color: var(--bn-text);
	}
	.pv-rail {
		border-color: var(--bn-border);
		color: var(--bn-text-3);
	}
	.pv-rail:hover {
		color: var(--bn-text-2);
	}
	.pv-rail.is-on {
		border-color: var(--bn-border-strong);
		color: var(--bn-text);
	}
	@keyframes persona-in {
		from {
			opacity: 0;
			transform: translateY(6px);
		}
		to {
			opacity: 1;
			transform: none;
		}
	}
	.persona-in {
		animation: persona-in 0.28s ease;
	}
	@media (prefers-reduced-motion: reduce) {
		.persona-in {
			animation: none;
		}
	}
</style>
