<!-- Waffle dot-matrix mini: up to six dots a column, the tallest at the max
     (FounderOS v1 slab-charts.tsx DotMatrix). -->
<script lang="ts">
	import './kit.css';
	import type { SeriesPoint } from './format';

	const EASE = 'cubic-bezier(.2,.7,.2,1)';

	let { cols, hue }: { cols: SeriesPoint[]; hue: string } = $props();

	const max = $derived(Math.max(...cols.map((c) => c.count), 1));
	const bars = $derived(
		cols.map((c) => ({
			label: c.label,
			dots: Math.max(c.count === 0 ? 0 : 1, Math.round((c.count / max) * 6)),
			strength: c.count === max ? 1 : c.count >= max * 0.6 ? 0.55 : 0.25
		}))
	);
</script>

<div class="flex items-end gap-3">
	{#each bars as c, ci (c.label)}
		<div class="flex flex-col items-center gap-1.5">
			<div class="flex flex-col-reverse gap-[3px]">
				{#each { length: c.dots } as _, i (i)}
					<span
						data-dot
						class="bn-fade block h-[7px] w-[7px] rounded-full"
						style="background: {hue}; opacity: {c.strength}; animation: bn-fade .3s {EASE} {900 + ci * 90 + i * 55}ms both"
					></span>
				{/each}
				{#if c.dots === 0}
					<span data-ghost class="block h-[7px] w-[7px] rounded-full" style="background: {hue}; opacity: 0.12"></span>
				{/if}
			</div>
			<span class="bn-dim whitespace-nowrap font-mono text-[10px]">{c.label}</span>
		</div>
	{/each}
</div>
