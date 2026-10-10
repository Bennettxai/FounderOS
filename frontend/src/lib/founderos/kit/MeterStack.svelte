<!-- The Deal Volume bars, staggered 150ms (FounderOS v1 slab.tsx MeterStack).
     `dense` is the tighter rhythm for a volume card sharing its row with a
     shorter page (/workflows). -->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import './kit.css';
	import Render from './Render.svelte';
	import VolumeMeter from './VolumeMeter.svelte';
	import type { Meter } from './format';

	let {
		meters,
		foot,
		empty = 'nothing to measure yet',
		baseDelay = 500,
		dense = false
	}: { meters: Meter[]; foot?: string | Snippet; empty?: string; baseDelay?: number; dense?: boolean } = $props();
</script>

<div class="{dense ? 'mt-4' : 'mt-5'} flex flex-1 flex-col">
	<div data-part="meters" class="bn-border flex flex-1 flex-col justify-around border-t {dense ? 'gap-2.5 pt-3.5' : 'gap-6 pt-5'}">
		{#if meters.length === 0}
			<div class="bn-dim text-[12.5px]">{empty}</div>
		{:else}
			{#each meters as m, i (m.label)}
				<VolumeMeter {...m} delay={baseDelay + i * 150} />
			{/each}
		{/if}
	</div>
	{#if foot}
		<div data-part="foot" class="bn-border bn-dim {dense ? 'mt-3 pt-2.5' : 'mt-5 pt-3'} border-t text-center font-mono text-[10.5px] tracking-[0.1em]"><Render value={foot} /></div>
	{/if}
</div>
