<!-- One stacked bar across the four lanes, one hue per lane (UsageBoard.tsx LaneBar). -->
<script lang="ts">
	import { BURN_SOURCES, LANE_HUE, SOURCE_LABEL, burnOf, fmtTokens, laneBurn, type PlanUsage, type UsageWindow } from './usage';
	import { TRACK } from './ui';

	let { plan, window, height = 8 }: { plan: PlanUsage; window: UsageWindow; height?: number } = $props();

	const lanes = $derived(plan.breakdown?.windows[window]);
	const total = $derived(lanes ? laneBurn(lanes) : 0);
</script>

<div data-part="lane-bar" class="flex w-full gap-[2px] overflow-hidden rounded-full" style="height: {height}px; {TRACK}">
	{#if lanes && total > 0}
		{#each BURN_SOURCES as s (s)}
			{@const w = (burnOf(lanes[s]) / total) * 100}
			{#if w > 0}
				<div class="h-full" style="width: {w}%; background: {LANE_HUE[s]}" title="{SOURCE_LABEL[s]} · {fmtTokens(burnOf(lanes[s]))}"></div>
			{/if}
		{/each}
	{/if}
</div>
