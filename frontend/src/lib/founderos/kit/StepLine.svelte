<!-- Stepped line over fading pinstripes that draws itself and pins its peak
     (FounderOS v1 slab-charts.tsx StepLine). -->
<script lang="ts">
	import './kit.css';
	import type { SeriesPoint } from './format';

	const EASE = 'cubic-bezier(.2,.7,.2,1)';
	const W = 600;

	let {
		series,
		hue,
		empty = 'No activity in this window.',
		unit = '',
		height = 150
	}: {
		series: SeriesPoint[];
		hue: string;
		empty?: string;
		/** Appended to the peak callout's number ("12 runs on Sep 3"). */
		unit?: string;
		/** viewBox height against a 600-wide box; smaller draws a flatter strip (/workflows uses 56). */
		height?: number;
	} = $props();
	const H = $derived(height);

	const uid = $props.id();
	const pins = `bn-pins-${uid}`;
	const fade = `bn-grad-${uid}`;
	const mask = `bn-mask-${uid}`;

	const total = $derived(series.reduce((n, s) => n + s.count, 0));
	const geo = $derived.by(() => {
		if (series.length === 0 || total === 0) return null;
		const max = Math.max(...series.map((b) => b.count), 1);
		const stepW = W / series.length;
		// headroom for the peak callout scales with the box, 60 of 150 at the default
		const pad = Math.round(H * 0.4);
		const y = (c: number) => H - Math.round(H * 0.09) - (c / max) * (H - pad);
		let path = `M 0 ${y(series[0].count)}`;
		series.forEach((_, i) => {
			path += ` H ${(i + 1) * stepW}`;
			if (i < series.length - 1) path += ` V ${y(series[i + 1].count)}`;
		});
		const peakIdx = series.reduce((bi, b, i) => (b.count > series[bi].count ? i : bi), 0);
		const peak = series[peakIdx];
		return { path, area: `${path} V ${H} H 0 Z`, peakIdx, peak, stepW, py: y(peak.count) };
	});
</script>

{#if !geo}
	<div class="bn-dim px-6 py-8 text-[12px]">{empty}</div>
{:else}
	<div class="relative px-6 pb-5">
		<svg viewBox="0 0 {W} {H}" class="block w-full" aria-hidden="true">
			<defs>
				<pattern id={pins} width="5" height="8" patternUnits="userSpaceOnUse">
					<rect width="1.2" height="8" style="fill: {hue}" opacity="0.28" />
				</pattern>
				<linearGradient id={fade} x1="0" y1="0" x2="0" y2="1">
					<stop offset="0" style="stop-color: var(--bn-text); stop-opacity: 0.9" />
					<stop offset="1" style="stop-color: var(--bn-text); stop-opacity: 0.05" />
				</linearGradient>
				<mask id={mask}>
					<rect width={W} height={H} fill="url(#{fade})" />
				</mask>
			</defs>
			<path d={geo.area} fill="url(#{pins})" mask="url(#{mask})" />
			<path
				data-part="line"
				class="bn-draw"
				d={geo.path}
				fill="none"
				stroke-width="3"
				stroke-linejoin="round"
				pathLength={1}
				style="stroke: {hue}; stroke-dasharray: 1; stroke-dashoffset: 1; animation: bn-draw 1.6s {EASE} .5s both"
			/>
			<circle class="bn-fade" cx={(geo.peakIdx + 0.5) * geo.stepW} cy={geo.py} r="4.5" style="fill: {hue}; animation: bn-fade .4s {EASE} 1.9s both" />
		</svg>
		<div
			data-part="peak"
			class="bn-fade bn-border pointer-events-none absolute whitespace-nowrap rounded-full border px-2.5 py-1 text-[11px] backdrop-blur"
			style="left: calc(24px + (100% - 48px) * {(geo.peakIdx + 0.5) / series.length}); top: {(geo.py / H) * 100}%; transform: translate(-50%, -140%); background: color-mix(in oklab, var(--bn-bg) 75%, transparent); animation: bn-fade .5s {EASE} 2s both"
		>
			<span class="font-semibold tabular-nums" style="color: {hue}">{geo.peak.count.toLocaleString('en-US')}{unit}</span>
			<span class="bn-muted">on {geo.peak.label}</span>
		</div>
		<div class="bn-dim mt-1 flex justify-between font-mono text-[10.5px]">
			<span>{series[0].label}</span>
			<span>{series[series.length - 1].label}</span>
		</div>
	</div>
{/if}
