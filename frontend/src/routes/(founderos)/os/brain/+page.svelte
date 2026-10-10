<!-- /os/brain (spec 6.6), FounderOS v1 app/brain/page.tsx on the Optimal Engine.
     One screen, no scroll: the header carries the compact capture in its right
     slot, the knowledge graph fills everything under it, the satellites float
     over the graph. Like production, no health readouts here: they live on
     /os/doctor. -->
<script lang="ts">
	import { onMount } from 'svelte';
	import { page as appPage } from '$app/state';
	import { operatorName } from '$lib/founderos/operator';
	import { founderosFetch } from '$lib/founderos/api';
	import { PageHeader } from '$lib/founderos/kit';
	import BrainDump from '$lib/founderos/pages/brain/BrainDump.svelte';
	import BrainGraphView from '$lib/founderos/pages/brain/BrainGraphView.svelte';
	import BrainSatellites from '$lib/founderos/pages/brain/BrainSatellites.svelte';
	import type { BrainGraphBody, BrainPage } from '$lib/founderos/pages/brain/types';

	let page = $state<BrainPage | null>(null);
	let pageError = $state<string | null>(null);
	let graph = $state<BrainGraphBody | null>(null);
	let graphError = $state<string | null>(null);
	let panelOpen = $state(false);

	const msg = (e: unknown) => (e instanceof Error ? e.message : 'unreachable');

	// v1 draws the operator at the core by name ("Alex"): the signed-in user's
	// first name, not the API's generic label.
	const operator = $derived(operatorName(appPage.data?.user as { name?: unknown; email?: unknown } | undefined));
	const view = $derived(
		page ? { ...page, graph: { ...page.graph, nodes: page.graph.nodes.map((n) => (n.kind === 'self' ? { ...n, label: operator } : n)) } } : null
	);

	function loadGraph() {
		founderosFetch<BrainGraphBody>('/pages/brain/graph')
			.then((g) => {
				graph = g;
				graphError = null;
			})
			.catch((e: unknown) => (graphError = msg(e)));
	}

	onMount(() => {
		founderosFetch<BrainPage>('/pages/brain')
			.then((p) => (page = p))
			.catch((e: unknown) => (pageError = msg(e)));
		loadGraph();
	});

	const unreadable = $derived(graph ? graph.workspaces.filter((w) => w.error) : []);
	const memoryError = $derived(
		graphError ?? (unreadable.length ? unreadable.map((w) => `${w.workspace}: ${w.error}`).join(' · ') : null)
	);
</script>

<div class="flex flex-col" style="height: calc(100dvh - 9.25rem); min-height: 520px" data-part="brain-page">
	<PageHeader eyebrow="knowledge core" title="Brain" caret rightWide>
		{#snippet right()}
			<BrainDump onSaved={loadGraph} />
		{/snippet}
	</PageHeader>

	<div class="relative -mt-3 min-h-0 flex-1 bn-rise" style="--rise-i: 1">
		{#if view}
			<BrainGraphView page={view} memory={graph?.constellation ?? null} {memoryError} onPanel={(quiet) => (panelOpen = quiet)} />
		{:else if pageError}
			<div class="bn-card flex h-full items-center justify-center p-6" data-part="graph-error">
				<p class="bn-muted max-w-[60ch] text-center font-mono text-[11.5px]">The org graph is unavailable: <span class="bn-err-text">{pageError}</span></p>
			</div>
		{:else}
			<div class="flex h-full flex-col gap-3 lg:flex-row" style="padding-top: 47px" aria-busy="true" data-part="page-skeleton">
				<div class="bn-card min-h-[420px] min-w-0 flex-1 lg:min-h-0"></div>
				<div class="bn-card hidden shrink-0 lg:block lg:w-72"></div>
			</div>
		{/if}
		<BrainSatellites quiet={panelOpen} />
	</div>
</div>

<style>
	.bn-err-text {
		color: var(--bn-err);
	}
</style>
