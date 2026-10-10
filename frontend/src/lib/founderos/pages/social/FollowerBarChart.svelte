<!-- Follower history as bars (components/FollowerBarChart.tsx): one bar per
     snapshot on round y ticks; the tip is green on a gain, red on a dip, and
     the newest bar runs full accent. Hover a bar for the exact count. -->
<script lang="ts">
	import { followerBarModel, formatFollowers, type FollowerPoint } from './lib';

	let { series }: { series: FollowerPoint[] } = $props();

	const W = 920;
	const H = 240;
	const AXIS_L = 58;
	const AXIS_B = 22;
	const plotW = W - AXIS_L - 8;
	const plotH = H - AXIS_B - 10;
	const model = $derived(followerBarModel(series));
	const n = $derived(model.bars.length);
	const slot = $derived(plotW / Math.max(1, n));
	const barW = $derived(Math.max(3, Math.min(26, slot * 0.62)));
	const span = $derived(Math.max(1, model.ceil - model.floor));
</script>

{#if n === 0}
	<p class="bn-dim mt-4 text-xs" data-part="bars-empty">No snapshots yet — counts appear once a sync records this account.</p>
{:else}
	<svg viewBox="0 0 {W} {H}" class="mt-2 block w-full" role="img" aria-label="Follower count over time as a bar chart" data-part="follower-bars">
		{#each model.yTicks as v (v)}
			{@const y = 10 + plotH - ((v - model.floor) / span) * plotH}
			{#if y >= 6 && y <= 10 + plotH + 1}
				<line x1={AXIS_L} x2={W - 8} y1={y} y2={y} stroke="var(--bn-border)" stroke-dasharray="2 5" />
				<text x={AXIS_L - 8} y={y + 3} text-anchor="end" fill="var(--bn-text-3)" font-size="9" font-family="var(--bn-font)">{formatFollowers(v)}</text>
			{/if}
		{/each}
		{#each model.bars as b, i (b.date)}
			{@const x = AXIS_L + i * slot + (slot - barW) / 2}
			{@const h = b.h * plotH}
			{@const y = 10 + plotH - h}
			{@const newest = i === n - 1}
			<g>
				<rect {x} {y} width={barW} height={h} fill="var(--bn-accent)" opacity={newest ? 0.95 : 0.28 + b.h * 0.3}>
					<title>{`${b.date} · ${formatFollowers(b.followers)} followers${b.delta == null ? '' : ` · ${b.delta >= 0 ? '+' : ''}${b.delta.toLocaleString('en-US')}`}`}</title>
				</rect>
				<rect {x} y={y - 2} width={barW} height="2" fill={b.delta == null ? 'var(--bn-accent)' : b.delta >= 0 ? 'var(--bn-ok)' : 'var(--bn-err)'} opacity={newest ? 1 : 0.8} />
				{#if b.xLabel}
					<text x={x + barW / 2} y={H - 6} text-anchor="middle" fill="var(--bn-text-3)" font-size="8.5" font-family="var(--bn-font)">{b.xLabel}</text>
				{/if}
			</g>
		{/each}
		<line x1={AXIS_L} x2={W - 8} y1={10 + plotH} y2={10 + plotH} stroke="var(--bn-border-strong)" />
	</svg>
{/if}
