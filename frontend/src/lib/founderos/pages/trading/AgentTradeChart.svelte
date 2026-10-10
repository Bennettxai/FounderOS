<!-- The agent graph (FounderOS v1 AgentTradeChart.tsx): one account's value over
     time with its trades pinned to the line. Monolith: one series, hairline
     axes, square marks; buys and sells differ by fill, never by hue alone. -->
<script lang="ts">
	import { chartGeometry, clock, usd } from './view';
	import type { TradeActivity, TradingAccountSnapshot } from './types';

	const W = 720;
	const H = 168;
	const GUTTER = 6;

	let { history, trades }: { history: TradingAccountSnapshot[]; trades: TradeActivity[] } = $props();

	const geo = $derived(chartGeometry(history, trades, { w: W, h: H, pad: 10 }));
	let hover = $state<number | null>(null);
	const active = $derived(geo && hover !== null ? geo.points[hover] : null);
	const up = $derived(geo ? geo.changeUsd >= 0 : true);

	function move(e: MouseEvent) {
		if (!geo) return;
		const box = (e.currentTarget as SVGElement).getBoundingClientRect();
		const x = ((e.clientX - box.left) / (box.width || 1)) * (W + GUTTER * 2) - GUTTER;
		let best = 0;
		geo.points.forEach((p, i) => {
			if (Math.abs(p.x - x) < Math.abs(geo.points[best].x - x)) best = i;
		});
		hover = best;
	}
</script>

{#if !geo}
	<div class="flex items-center justify-center border border-dashed px-6 text-center" style="aspect-ratio: {W + GUTTER * 2} / {H}; border-color: var(--bn-border)">
		<p class="bn-dim font-mono text-[11px] leading-relaxed">No value series yet. The graph draws itself once the agent has pushed a second snapshot.</p>
	</div>
{:else}
	<div class="relative">
		<svg
			viewBox="{-GUTTER} 0 {W + GUTTER * 2} {H}"
			class="w-full"
			style="aspect-ratio: {W + GUTTER * 2} / {H}"
			role="img"
			aria-label="Account value, {usd(geo.firstUsd)} to {usd(geo.lastUsd)} across {geo.points.length} samples"
			onmousemove={move}
			onmouseleave={() => (hover = null)}
		>
			{#each [0.5, 1] as f (f)}
				<line x1={-GUTTER} x2={W + GUTTER} y1={H * f} y2={H * f} stroke="var(--bn-border)" stroke-width="1" vector-effect="non-scaling-stroke" />
			{/each}
			<path d={geo.area} fill="var(--bn-accent)" opacity="0.07" />
			<path
				class="bn-draw"
				d={geo.line}
				fill="none"
				stroke="var(--bn-accent)"
				stroke-width="2"
				stroke-linejoin="round"
				vector-effect="non-scaling-stroke"
				pathLength={1}
				style="stroke-dasharray: 1; stroke-dashoffset: 1; animation: bn-draw 1.6s cubic-bezier(.2,.7,.2,1) .4s both"
			/>
			{#each geo.markers as m (m.trade.id)}
				<rect
					data-marker={m.trade.action}
					x={m.x - 4}
					y={m.y - 4}
					width="8"
					height="8"
					fill={m.trade.action === 'buy' ? 'var(--bn-accent)' : 'var(--bn-surface)'}
					stroke="var(--bn-accent)"
					stroke-width="1.5"
					vector-effect="non-scaling-stroke"
				>
					<title>{`${m.trade.action.toUpperCase()} ${m.trade.quantity} ${m.trade.symbol} @ ${usd(m.trade.priceUsd)}`}</title>
				</rect>
			{/each}
			{#if active}
				<line x1={active.x} x2={active.x} y1="0" y2={H} stroke="var(--bn-border-strong)" stroke-width="1" vector-effect="non-scaling-stroke" />
				<circle cx={active.x} cy={active.y} r="3.5" fill="var(--bn-accent)" />
			{/if}
		</svg>
		<div class="bn-dim mt-1 flex items-baseline justify-between font-mono text-[9.5px] uppercase tracking-[0.18em]">
			<span>{clock(geo.points[0].at)}</span>
			<span style="color: {up ? 'var(--bn-ok)' : 'var(--bn-err)'}">{up ? '+' : '-'}{usd(Math.abs(geo.changeUsd))} ({up ? '+' : ''}{geo.changePct}%)</span>
			<span>{clock(geo.points[geo.points.length - 1].at)}</span>
		</div>
		{#if active}
			<div
				class="bn-text pointer-events-none absolute -top-1 z-10 -translate-x-1/2 border px-2 py-1 font-mono text-[10px] tabular-nums"
				style="left: {((active.x + GUTTER) / (W + GUTTER * 2)) * 100}%; border-color: var(--bn-border-strong); background: var(--bn-surface)"
			>
				{usd(active.valueUsd)} · <span class="bn-dim">{clock(active.at)}</span>
			</div>
		{/if}
	</div>
{/if}
