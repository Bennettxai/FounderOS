<!-- Accent sparkline with a 10%-opacity fill (FounderOS v1 terminal.tsx Spark). -->
<script lang="ts">
	import { sparkPoints } from './format';

	let { data, w = 72, h = 22 }: { data: number[]; w?: number; h?: number } = $props();
	const pts = $derived(sparkPoints(data, w, h));
</script>

{#if pts}
	<svg width={w} height={h} viewBox="0 0 {w} {h}" aria-hidden="true">
		<polygon points="0,{h} {pts.join(' ')} {w},{h}" fill="var(--bn-accent)" opacity="0.1" />
		<polyline points={pts.join(' ')} fill="none" stroke="var(--bn-accent)" stroke-width="1.5" />
	</svg>
{/if}
