<!-- Blueprint (spec 6.19): the system, drawn from itself. The Go compiler
     assembles the graph from the bridge's live registries on every request
     (GET /api/founderos/pages/blueprint); the workspace reads it as machines →
     groups → things. The workspace is interaction-heavy, so it loads after
     first paint behind a dimension-matched skeleton, like FounderOS v1's
     BlueprintCanvasLazy. -->
<script lang="ts">
	import { onMount, type Component } from 'svelte';
	import { founderosFetch } from '$lib/founderos/api';
	import { blueprintSubline, type BlueprintBody, type BlueprintGraph } from '$lib/founderos/pages/blueprint/graph';
	import '$lib/founderos/pages/blueprint/blueprint.css';

	type WorkspaceProps = { graph: BlueprintGraph; subline: string };

	let graph = $state<BlueprintGraph | null>(null);
	let failure = $state<string | null>(null);
	let Workspace = $state<Component<WorkspaceProps> | null>(null);

	onMount(() => {
		const load = import('$lib/founderos/pages/blueprint/HierarchyWorkspace.svelte');
		founderosFetch<BlueprintBody>('/pages/blueprint')
			.then(async (body) => {
				if (!body?.graph) throw new Error(body?.error || 'the compiler returned no graph');
				Workspace = (await load).default as unknown as Component<WorkspaceProps>;
				graph = body.graph;
			})
			.catch((e: unknown) => {
				failure = e instanceof Error ? e.message : 'unreachable';
			});
	});

	const subline = $derived(graph ? blueprintSubline(graph) : '');
</script>

{#if failure}
	<div class="bh-page">
		<header class="bh-top">
			<div class="bh-title">
				<div class="bh-h1">Blueprint</div>
				<div class="bh-sub">the map could not compile</div>
			</div>
		</header>
		<div class="bh-state" role="alert">
			<span class="bh-state-k">unreachable</span>
			<span>Blueprint could not compile: {failure}</span>
		</div>
	</div>
{:else if graph && graph.nodes.length === 0}
	<div class="bh-page">
		<header class="bh-top">
			<div class="bh-title">
				<div class="bh-h1">Blueprint</div>
				<div class="bh-sub">{subline}</div>
			</div>
		</header>
		<div class="bh-state">
			<span class="bh-state-k">empty</span>
			<span>Nothing compiled: the registries answered with no agents, connectors or infrastructure.</span>
		</div>
	</div>
{:else if graph && Workspace}
	<Workspace {graph} {subline} />
{:else}
	<div class="bh-skeleton animate-pulse" aria-busy="true" aria-label="compiling the blueprint"></div>
{/if}

<style>
	.bh-state {
		flex: 1;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 10px;
		padding: 24px;
		text-align: center;
		font-family: var(--bn-font);
		font-size: 12.5px;
		color: var(--bn-text-2);
	}
	.bh-state-k {
		font-size: 10px;
		font-weight: 700;
		letter-spacing: 0.26em;
		text-transform: uppercase;
		color: var(--bn-err);
	}
</style>
