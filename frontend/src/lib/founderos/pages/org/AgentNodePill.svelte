<!-- A worker pill with its sub-agents nested under it (app/org AgentNodePill). -->
<script lang="ts">
	import AgentNodePill from './AgentNodePill.svelte';
	import WorkerPill from './WorkerPill.svelte';
	import { dimFor } from './org';
	import type { AgentNode, Venture } from './types';

	let { node, depth = 0, venture, ventures }: { node: AgentNode; depth?: number; venture: Venture | null; ventures: Venture[] } = $props();
</script>

<div class="space-y-1.5" style={depth ? `padding-left: ${depth * 10}px` : undefined}>
	<WorkerPill agent={node.agent} dim={dimFor(venture, node.agent.id)} {ventures} />
	{#each node.children as child (child.agent.id)}
		<AgentNodePill node={child} depth={depth + 1} {venture} {ventures} />
	{/each}
</div>
