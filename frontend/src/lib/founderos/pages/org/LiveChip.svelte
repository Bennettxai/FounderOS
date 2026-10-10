<!-- Tiny live chip: board status LED + model, worn by nodes the board matches. -->
<script lang="ts">
	import OrgDot from './OrgDot.svelte';
	import { liveDot } from './org';
	import type { BoardAgent } from './types';

	let { agent }: { agent: BoardAgent } = $props();
	const dot = $derived(liveDot(agent.status));
</script>

<span
	data-part="live-chip"
	title={`Paperclip board: ${agent.name} is ${agent.status}${agent.model ? ` on ${agent.model}` : ''}`}
	class="bn-border bn-muted flex items-center gap-1 rounded-full border px-1.5 py-0.5 font-mono text-[8.5px] uppercase tracking-[0.08em]"
	style="background: var(--bn-bg)"
>
	<OrgDot look={dot} />
	{agent.status}
	{#if agent.model}<span class="bn-dim normal-case">· {agent.model}</span>{/if}
</span>
