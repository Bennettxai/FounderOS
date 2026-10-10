<!-- View switch for the /os/brain knowledge graph (FounderOS v1
     components/BrainGraphView.tsx): the radial wheel (default) or the neural
     network projection, pillar chips on the radial view only. Both engines
     are the page's heaviest bundles, so each loads with a dynamic import
     behind a skeleton matching its settled footprint: the Svelte form of
     FounderOS v1's next/dynamic({ ssr: false }) contract. -->
<script lang="ts">
	import { ToggleChip } from '$lib/founderos/kit';
	import type { BrainPage, MemoryGraph } from './types';

	let {
		page,
		memory = null,
		memoryError = null,
		onPanel
	}: { page: BrainPage; memory?: MemoryGraph | null; memoryError?: string | null; onPanel?: (open: boolean) => void } = $props();

	let view = $state<'radial' | 'neural'>('radial');
	// chips read in department order (FounderOS v1 passes db.departments.all(),
	// ORDER BY "order"); the wheel keeps its own graph order (kg.ts GRAPH_DEPT_ORDER)
	const pillars = $derived(
		page.departments.filter((d) => page.graph.nodes.some((n) => n.id === `team:${d.id}`)).sort((a, b) => a.order - b.order)
	);
	const TAB_W = 84;
	const TABS = [
		['radial', 'Radial'],
		['neural', 'Neural']
	] as const;
	let off = $state<string[]>([]);
	const activePillars = $derived(pillars.map((d) => d.id).filter((id) => !off.includes(id)));
	// all on = no filter (production passes every department id, which reads the same)
	const activeArg = $derived(off.length === 0 ? undefined : activePillars);
	const toggle = (id: string) => (off = off.includes(id) ? off.filter((x) => x !== id) : [...off, id]);

	const graphModule = import('./KnowledgeGraph.svelte');
	let neuralModule: Promise<typeof import('./NeuralGraph.svelte')> | null = $state(null);
	$effect(() => {
		if (view === 'neural' && !neuralModule) neuralModule = import('./NeuralGraph.svelte');
	});
</script>

<div class="flex h-full flex-col" data-part="graph-view">
	<div class="mb-2 flex shrink-0 flex-wrap items-center gap-1" style="min-height: 39px">
		<div
			class="bn-tabs relative grid rounded-[8px] border p-[3px]"
			style="grid-template-columns: repeat({TABS.length}, {TAB_W}px)"
			role="tablist"
			data-variant="pill"
		>
			<span
				aria-hidden="true"
				data-part="tab-indicator"
				class="bn-tab-ind pointer-events-none absolute bottom-[3px] left-[3px] top-[3px] rounded-[5px]"
				style="width: {TAB_W}px; transform: translateX({TABS.findIndex(([id]) => id === view) * TAB_W}px)"
			></span>
			{#each TABS as [id, label] (id)}
				<button
					type="button"
					role="tab"
					aria-selected={view === id}
					data-lens="c"
					class="bn-pressable bn-tab relative inline-flex h-[30px] items-center justify-center bg-transparent font-mono text-[10.5px] font-bold uppercase tracking-[.18em]"
					class:is-on={view === id}
					onclick={() => (view = id)}>{label}</button
				>
			{/each}
		</div>
		{#if view === 'radial' && pillars.length > 0}
			<span aria-hidden="true" class="bn-sep mx-1.5 h-[18px] w-px shrink-0"></span>
			<div class="flex flex-wrap items-center gap-1" data-part="pillar-chips">
				{#each pillars as d (d.id)}
					<ToggleChip on={!off.includes(d.id)} title={`Filter ${d.name}`} aria-pressed={!off.includes(d.id)} onclick={() => toggle(d.id)}>{d.name}</ToggleChip>
				{/each}
			</div>
		{/if}
	</div>
	<div class="min-h-0 flex-1">
		{#if view === 'radial'}
			{#await graphModule}
				<div class="flex h-full flex-col gap-3 lg:flex-row" data-part="graph-skeleton" aria-busy="true">
					<div class="bn-skel min-h-[420px] min-w-0 flex-1 rounded-[10px] border lg:min-h-0"></div>
					<div class="bn-skel hidden shrink-0 rounded-[10px] border lg:block lg:w-72"></div>
				</div>
			{:then mod}
				<mod.default {page} {memory} {memoryError} activePillars={activeArg} {onPanel} onShowAllPillars={() => (off = [])} />
			{:catch e}
				<div class="bn-err-text font-mono text-[11px]">the graph failed to load: {e instanceof Error ? e.message : String(e)}</div>
			{/await}
		{:else if neuralModule}
			{#await neuralModule}
				<div class="bn-skel w-full overflow-hidden rounded-[10px] border" style="aspect-ratio: 1200 / 640" data-part="neural-skeleton" aria-busy="true"></div>
			{:then mod}
				<mod.default {page} />
			{:catch e}
				<div class="bn-err-text font-mono text-[11px]">the neural view failed to load: {e instanceof Error ? e.message : String(e)}</div>
			{/await}
		{/if}
	</div>
</div>

<style>
	/* SlidingTabs variant="pill": the marker slides between fixed-width tabs */
	.bn-tabs {
		border-color: var(--bn-border);
		background: var(--bn-bg);
	}
	.bn-tab-ind {
		background: var(--bn-surface-2);
		transition: transform var(--bn-dur-lens, 0.2s) var(--bn-ease-lens, ease);
	}
	.bn-tab {
		color: var(--bn-text-3);
	}
	.bn-tab:hover {
		color: var(--bn-text-2);
	}
	.bn-tab.is-on {
		color: var(--bn-text);
	}
	@media (prefers-reduced-motion: reduce) {
		.bn-tab-ind {
			transition: none;
		}
	}
	.bn-sep {
		background: var(--bn-border);
	}
	.bn-skel {
		border-color: var(--bn-border);
		background: var(--bn-surface);
		animation: bn-skel-pulse 1.6s ease-in-out infinite;
	}
	@keyframes bn-skel-pulse {
		50% {
			opacity: 0.55;
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.bn-skel {
			animation: none;
		}
	}
	.bn-err-text {
		color: var(--bn-err);
	}
</style>
