<!-- FounderOS v1 SparkBars: flex cells, 3px floor, 15px of travel. -->
<script lang="ts">
	import { sparkBarHeights } from './console';
	let { data, height = 18 }: { data: number[]; height?: number } = $props();
	const bars = $derived(sparkBarHeights(data));
</script>

{#if data.length > 0}
	<div class="flex items-end gap-[2px]" style:height="{height}px" aria-hidden="true">
		{#each bars as h, i (i)}
			<span class="flex-1" style:height="{h}px" style:background="var(--bn-text-2)"></span>
		{/each}
	</div>
{/if}
