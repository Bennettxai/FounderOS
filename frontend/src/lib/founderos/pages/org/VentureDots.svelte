<!-- Tiny colored dots: which ventures an agent serves. Venture colors are data. -->
<script lang="ts">
	import { venturesForAgent } from './org';
	import type { Venture } from './types';

	let { ventures, agentId }: { ventures: Venture[]; agentId: string } = $props();
	const serving = $derived(venturesForAgent(ventures, agentId));
</script>

{#if serving.length > 0}
	<span class="flex shrink-0 items-center gap-0.5">
		{#each serving as v (v.id)}
			<span title={v.label} class="h-1 w-1 rounded-full" style="background: {v.color}"></span>
		{/each}
	</span>
{/if}
