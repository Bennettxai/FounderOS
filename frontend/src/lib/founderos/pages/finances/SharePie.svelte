<!-- Share donut (FounderOS v1 SharePie.tsx): one slice per item, the total at
     center, a legend with values and shares. Slices ride the Monolith greys;
     colour stays reserved for status. -->
<script lang="ts">
	import { pieSlices, type PieItem } from './spend-report';

	const SLICES = ['var(--bn-text)', 'var(--bn-text-2)', 'var(--bn-text-3)', 'var(--bn-border-strong)', 'var(--bn-accent)', 'var(--bn-surface-3)'];
	const S = 168;
	const C = S / 2;
	const R = 62;

	let {
		items,
		total,
		centerLabel,
		format,
		donutPx = 170,
		legend = true,
		ariaLabel
	}: {
		items: PieItem[];
		total: number;
		centerLabel: string;
		format: (v: number) => string;
		donutPx?: number;
		/** off when the caller lists the shares itself (the /finances meters) */
		legend?: boolean;
		ariaLabel: string;
	} = $props();

	const slices = $derived(pieSlices(items));
	const rad = (deg: number) => (deg * Math.PI) / 180;
	function arc(a0: number, a1: number): string {
		const sweep = Math.min(a1 - a0, 359.98);
		const x0 = C + R * Math.cos(rad(a0));
		const y0 = C + R * Math.sin(rad(a0));
		const x1 = C + R * Math.cos(rad(a0 + sweep));
		const y1 = C + R * Math.sin(rad(a0 + sweep));
		return `M ${x0.toFixed(2)} ${y0.toFixed(2)} A ${R} ${R} 0 ${sweep > 180 ? 1 : 0} 1 ${x1.toFixed(2)} ${y1.toFixed(2)}`;
	}
</script>

{#if slices.length === 0}
	<p class="bn-dim py-4 text-center font-mono text-[10.5px]">Nothing to chart yet.</p>
{:else}
	<div class="flex flex-col items-center gap-3">
		<svg viewBox="0 0 {S} {S}" class="shrink-0" style="width: {donutPx}px; height: {donutPx}px" role="img" aria-label={ariaLabel}>
			{#each slices as s, i (s.key)}
				<path d={arc(s.startAngle + 0.6, s.endAngle - 0.6)} fill="none" stroke-width="17" style="stroke: {SLICES[i % SLICES.length]}">
					<title>{`${s.label} · ${format(s.value)} · ${(s.share * 100).toFixed(1)}%`}</title>
				</path>
			{/each}
			<text x={C} y={C - 3} text-anchor="middle" font-size="15" font-weight="700" style="fill: var(--bn-text)">{format(total)}</text>
			<text x={C} y={C + 13} text-anchor="middle" font-size="7.5" letter-spacing="0.18em" style="fill: var(--bn-text-3); text-transform: uppercase">{centerLabel}</text>
		</svg>
		{#if legend}
		<div class="flex w-full flex-col gap-1">
			{#each slices as s, i (s.key)}
				<div class="flex items-center gap-2 font-mono text-[10px]">
					<span class="h-2 w-2 shrink-0" style="background: {SLICES[i % SLICES.length]}"></span>
					<span class="bn-dim min-w-0 flex-1 truncate uppercase tracking-[0.08em]">{s.label}</span>
					<span class="bn-muted shrink-0">{format(s.value)}</span>
					<span class="bn-text w-11 shrink-0 text-right">{(s.share * 100).toFixed(1)}%</span>
				</div>
			{/each}
		</div>
		{/if}
	</div>
{/if}
