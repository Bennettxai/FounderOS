<!-- The pillar spider chart (FounderOS v1 components/PillarRadar.tsx): one axis
     per department over a three-level grid, the overall-health polygon plus
     its three signals as layers. Hovering sifts between layers: the nearest
     isolates, the rest recede. Monolith: layers differ by dash, not hue. -->
<script lang="ts">
	import { nearestPillarLayer, radarPoint, type PillarLayerKey } from './radar';
	import type { PillarAxis } from './types';

	let { axes, health, warnings }: { axes: PillarAxis[]; health: number | null; warnings: number } = $props();

	const S = 520;
	const C = S / 2;
	const R = 172;
	const LABEL_R = R + 36;
	const PAD_X = 76;
	const VB_MIN_X = -PAD_X;
	const VB_W = S + PAD_X * 2;

	type Layer = { key: PillarLayerKey; label: string; color: string; dash?: string; fill?: boolean };
	const LAYERS: Layer[] = [
		{ key: 'score', label: 'Overall health', color: 'var(--bn-text)', fill: true },
		// prod --funnel-s0/s1/s2: a grey ramp on Monolith, hues on Terminal (style below)
		{ key: 'roster', label: 'Roster active', color: 'var(--pr-s0)' },
		{ key: 'freshness', label: 'Run recency', color: 'var(--pr-s1)', dash: '6 4' },
		{ key: 'sop', label: 'SOP coverage', color: 'var(--pr-s2)', dash: '2 4' }
	];

	let svgEl = $state<SVGSVGElement | null>(null);
	let active = $state<PillarLayerKey | null>(null);

	const n = $derived(Math.max(1, axes.length));
	const ring = (k: number) =>
		Array.from({ length: n }, (_, i) => radarPoint(i, n, R * k, C).join(',')).join(' ');
	const point = (i: number, v: number, extra = 0) => radarPoint(i, n, (Math.max(5, v) / 100) * R + extra, C);
	const polyFor = (key: PillarLayerKey) => axes.map((a, i) => point(i, a[key]).join(',')).join(' ');
	const ordered = $derived([...LAYERS].sort((a, b) => Number(a.key === active) - Number(b.key === active)));
	const activeLayer = $derived(LAYERS.find((l) => l.key === active) ?? null);
	const opacity = (l: Layer) => (active === null ? (l.fill ? 1 : 0.85) : active === l.key ? 1 : 0.08);

	function onMove(e: MouseEvent) {
		if (!svgEl) return;
		const rect = svgEl.getBoundingClientRect();
		if (rect.width === 0) return;
		const x = VB_MIN_X + ((e.clientX - rect.left) / rect.width) * VB_W;
		const y = ((e.clientY - rect.top) / rect.height) * S;
		active = nearestPillarLayer({ x, y }, axes, R, C);
	}

	function anchor(x: number): 'middle' | 'start' | 'end' {
		return Math.abs(x - C) < 8 ? 'middle' : x > C ? 'start' : 'end';
	}
</script>

<div class="bn-pillar-radar flex h-full flex-col">
	<div class="grid flex-1 place-items-center">
		<svg
			bind:this={svgEl}
			viewBox="{VB_MIN_X} 0 {VB_W} {S}"
			class="block w-full max-w-[560px]"
			style="max-width: 560px"
			role="img"
			aria-label="Pillar health radar · hover to sift between layers"
			onmousemove={onMove}
			onmouseleave={() => (active = null)}
		>
			{#each [1 / 3, 2 / 3, 1] as k (k)}
				<polygon points={ring(k)} fill="none" stroke="var(--bn-border-strong)" stroke-width="1" opacity={k === 1 ? 0.9 : 0.5} />
			{/each}
			{#each axes as a, i (a.id)}
				{@const p = radarPoint(i, n, R, C)}
				<line x1={C} y1={C} x2={p[0]} y2={p[1]} stroke="var(--bn-border-strong)" stroke-width="1" opacity="0.4" />
			{/each}

			{#each ordered as layer (layer.key)}
				{@const on = active === layer.key}
				<g style="transition: opacity 0.15s" opacity={opacity(layer)}>
					<polygon
						points={polyFor(layer.key)}
						fill={(layer.fill && active === null) || on ? layer.color : 'none'}
						fill-opacity={layer.fill && active === null ? 0.07 : on ? 0.1 : 0}
						stroke={layer.color}
						stroke-width={on ? 3 : layer.fill ? 2.4 : 1.8}
						stroke-dasharray={on ? undefined : layer.dash}
						stroke-linejoin="round"
					/>
					{#each axes as a, i (a.id)}
						{@const p = point(i, a[layer.key])}
						<circle cx={p[0]} cy={p[1]} r={on ? 5.5 : layer.fill ? 5 : 3.4} fill={layer.color} />
					{/each}
					{#if on}
						{#each axes as a, i (a.id)}
							{@const o = point(i, a[layer.key], 15)}
							<text x={o[0]} y={o[1]} text-anchor="middle" dominant-baseline="middle" font-family="var(--bn-font)" font-size="12" font-weight="700" fill={layer.color}>{a[layer.key]}</text>
						{/each}
					{/if}
				</g>
			{/each}

			{#each axes as a, i (a.id)}
				{@const p = radarPoint(i, n, LABEL_R, C)}
				<text x={p[0]} y={p[1]} text-anchor={anchor(p[0])} dominant-baseline="middle" font-family="var(--bn-font)" font-size="13" letter-spacing="0.14em" fill="var(--bn-text-2)">{a.label.toUpperCase().replace('/GROWTH', '')}</text>
			{/each}
		</svg>
	</div>

	<div class="flex flex-wrap items-center justify-center gap-x-4 gap-y-1.5 px-4 font-mono text-[9.5px] uppercase tracking-[0.14em]">
		{#each LAYERS as layer (layer.key)}
			{@const on = active === layer.key}
			<button
				type="button"
				onmouseenter={() => (active = layer.key)}
				onmouseleave={() => (active = null)}
				data-lens="c"
				class="bn-pressable flex items-center gap-1.5 transition-opacity {active !== null && !on ? 'opacity-30' : ''} {on ? 'bn-text' : 'bn-dim'}"
			>
				<svg width="18" height="6" aria-hidden="true">
					<line x1="0" y1="3" x2="18" y2="3" stroke={layer.color} stroke-width={on ? 3 : layer.fill ? 2.6 : 1.8} stroke-dasharray={on ? undefined : layer.dash} />
				</svg>
				{layer.label}
			</button>
		{/each}
	</div>

	<div class="flex items-baseline justify-center gap-2 pb-4 pt-2 font-mono">
		{#if activeLayer}
			<span class="text-[11px] uppercase tracking-[0.16em]" style="color: {activeLayer.color}">{activeLayer.label} · per pillar, 0 to 100</span>
		{:else}
			<span class="text-2xl font-semibold" style="color: var(--bn-ok)">{health ?? '—'}</span>
			<span class="bn-dim text-[11px]">/ 100 health</span>
			{#if warnings > 0}
				<span style="color: var(--bn-border-strong)">·</span>
				<span class="text-[11px] font-semibold" style="color: var(--bn-warn)">{warnings} warning{warnings === 1 ? '' : 's'}</span>
			{/if}
		{/if}
	</div>
</div>

<style>
	.bn-pillar-radar {
		--pr-s0: #f2f2f2;
		--pr-s1: #bdbdbd;
		--pr-s2: #8a8a8a;
	}
	:global(:root[data-founderos-theme='terminal']) .bn-pillar-radar {
		--pr-s0: #5ec9f8;
		--pr-s1: #8b7cff;
		--pr-s2: #e3d872;
	}
</style>
