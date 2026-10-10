<!-- Audience share donut (components/SharePie.tsx, stacked + fluid as /social
     uses it): one slice per channel drawn by one clockwise sweep hand, total
     reach at the centre, legend with value and share. Slices ride prod's
     mono funnel ramp (--funnel-s0,1,2,3,5,6 as --soc-s*, set on .soc-root in
     social.css): on Monolith, color is for status only. -->
<script lang="ts">
	import { arcPath, formatFollowers, pieSlices, type PieItem } from './lib';

	let {
		items,
		total,
		centerLabel = 'total reach',
		donutMax = 230
	}: { items: PieItem[]; total: number; centerLabel?: string; donutMax?: number } = $props();

	const SLICES = ['--soc-s0', '--soc-s1', '--soc-s2', '--soc-s3', '--soc-s5', '--soc-s6'].map((v) => `var(${v})`);
	/** One full turn of the sweep hand, plus the lead-in before it starts. */
	const SWEEP_MS = 1100;
	const SWEEP_LEAD_MS = 200;
	const S = 168;
	const C = S / 2;
	const R = 62;
	const slices = $derived(pieSlices(items));
	// Each slice draws for a time proportional to its share, starting where the
	// previous one ended: one hand at constant angular speed.
	const timing = $derived(
		slices.map((s, i) => {
			const before = slices.slice(0, i).reduce((acc, x) => acc + x.share, 0);
			return { delay: Math.round(SWEEP_LEAD_MS + before * SWEEP_MS), dur: Math.max(60, Math.round(s.share * SWEEP_MS)) };
		})
	);
</script>

{#if slices.length === 0}
	<p class="bn-dim py-4 text-center font-mono text-[10.5px]">Nothing to chart yet.</p>
{:else}
	<div class="flex flex-col items-center gap-3" data-part="audience-pie">
		<svg viewBox="0 0 {S} {S}" class="shrink-0" style="width: 100%; max-width: {donutMax}px; height: auto; aspect-ratio: 1" role="img" aria-label="Share of total reach by channel">
			{#each slices as s, i (s.key)}
				<path
					class="sp-sweep"
					d={arcPath(C, C, R, s.startAngle + 0.6, s.endAngle - 0.6)}
					pathLength="1"
					fill="none"
					stroke={SLICES[i % SLICES.length]}
					stroke-width="17"
					style="animation-duration: {timing[i].dur}ms; animation-delay: {timing[i].delay}ms"
				>
					<title>{`${s.label} · ${formatFollowers(s.value)} · ${(s.share * 100).toFixed(1)}%`}</title>
				</path>
			{/each}
			<text x={C} y={C - 3} text-anchor="middle" fill="var(--bn-text)" font-size="15" font-weight="700" font-family="var(--bn-font)" letter-spacing="-0.02em">{formatFollowers(total)}</text>
			<text x={C} y={C + 13} text-anchor="middle" fill="var(--bn-text-3)" font-size="7.5" font-family="var(--bn-font)" letter-spacing="0.18em" style="text-transform: uppercase">{centerLabel}</text>
		</svg>
		<div class="flex w-full flex-col gap-1">
			{#each slices as s, i (s.key)}
				<div class="flex items-center gap-2 font-mono text-[10px]" data-part="legend-row">
					<span class="h-2 w-2 shrink-0" style="background: {SLICES[i % SLICES.length]}"></span>
					<span class="bn-dim min-w-0 flex-1 truncate uppercase tracking-[0.08em]">{s.label}</span>
					<span class="bn-muted shrink-0">{formatFollowers(s.value)}</span>
					<span class="bn-text w-11 shrink-0 text-right">{(s.share * 100).toFixed(1)}%</span>
				</div>
			{/each}
		</div>
	</div>
{/if}

<style>
	.sp-sweep {
		stroke-dasharray: 1;
		stroke-dashoffset: 1;
		animation-name: sp-sweep;
		animation-timing-function: linear;
		animation-fill-mode: both;
	}
	@keyframes sp-sweep {
		to {
			stroke-dashoffset: 0;
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.sp-sweep {
			animation: none;
			stroke-dashoffset: 0;
		}
	}
</style>
