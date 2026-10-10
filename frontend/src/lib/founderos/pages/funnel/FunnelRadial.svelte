<!-- The funnel as a circle (FounderOS v1 components/FunnelRadial.tsx): the
     journey runs outside → in. Acquisition wedges around the rim, stage rings
     pull leads inward, the core is the purchase. Same node language as the
     space; converted nodes leave their wedge for the shared core. Heavy:
     loaded lazily by FunnelGraph behind a same-aspect skeleton. -->
<script lang="ts">
	import FullscreenButton from './FullscreenButton.svelte';
	import FunnelNodeCard from './FunnelNodeCard.svelte';
	import type { FunnelNode, Segment } from './types';
	import { decayedColor, decayedOpacity, easeInOut, rnd, smoothK } from './viz';

	let {
		nodes,
		segments,
		stages,
		stageLabels,
		constants,
		initialLeadId = null
	}: {
		nodes: FunnelNode[];
		segments: Segment[];
		stages: { id: string; label: string }[];
		stageLabels: Record<string, string>;
		constants: { stallDays: number; decayDays: number; decayFadeStart: number };
		initialLeadId?: string | null;
	} = $props();

	const W = 1100;
	const H = 680;
	const CX = W / 2;
	const CY = H / 2;
	const TAU = Math.PI * 2;
	const SEG_INSET = 0.13;
	const TOP = -Math.PI / 2;
	const WEDGE = ['var(--fn-s0)', 'var(--fn-s1)', 'var(--fn-s2)', 'var(--fn-s3)', 'var(--fn-s5)', 'var(--fn-s6)'];
	const ENTER_DELAY = 500;
	const HOP_MS = 950;
	const DWELL_MS = 420;
	const GOLDEN = 2.399963;
	const staggerMs = (count: number) => Math.min(340, 5000 / Math.max(1, count));
	type Pos = { x: number; y: number };

	const segSpan = $derived(TAU / Math.max(1, segments.length));
	/** One radius per stage, outside → in; the last is the Closed core. */
	const RING = $derived(stages.map((_, i, all) => (i === all.length - 1 ? 48 : Math.round(288 - (i * (288 - 114)) / Math.max(1, all.length - 2)))));
	const CORE = $derived(RING[RING.length - 1]);

	const polar = (a: number, r: number): Pos => ({ x: Math.round((CX + Math.cos(a) * r) * 100) / 100, y: Math.round((CY + Math.sin(a) * r) * 100) / 100 });
	const wedgeAngle = (n: FunnelNode, i: number) => TOP + n.segment * segSpan + SEG_INSET + (segSpan - 2 * SEG_INSET) * rnd(i, 11);

	function bandRadius(n: FunnelNode, i: number, ring: number): number {
		if (ring >= RING.length - 1) return 6 + rnd(i, 12) * (CORE - 16);
		const outer = RING[ring] - 8;
		const inner = RING[ring + 1] + 12;
		const depth = 0.15 + 0.6 * (n.likelihood / 100) + rnd(i, 1) * 0.2;
		return outer - (outer - inner) * Math.min(0.95, depth);
	}

	function orbitTarget(n: FunnelNode, i: number, t: number): Pos {
		if (n.currentRing >= RING.length - 1) {
			const a = i * GOLDEN + t * 0.00005 * (rnd(i, 3) > 0.5 ? 1 : -1);
			return polar(a, bandRadius(n, i, RING.length - 1));
		}
		const wobble = Math.sin(t * (0.00018 + rnd(i, 2) * 0.00014) + rnd(i, 4) * TAU) * 0.09;
		const breath = Math.sin(t * 0.0009 + i * 1.3) * 2.4;
		return polar(wedgeAngle(n, i) + wobble, bandRadius(n, i, n.currentRing) + breath);
	}

	function replayPos(n: FunnelNode, i: number, tMs: number, stagger: number): Pos | null {
		const a = wedgeAngle(n, i);
		let t = tMs - (ENTER_DELAY + i * stagger);
		if (t <= 0) return polar(a, RING[0] + 46 + rnd(i, 5) * 26);
		const stops = [RING[0] + 46, ...n.rings.map((ring) => bandRadius(n, i, ring))];
		const twist = (rnd(i, 8) > 0.5 ? 1 : -1) * 0.07;
		const coreDelta = ((((i * GOLDEN) % TAU) - a + TAU * 1.5) % TAU) - Math.PI;
		for (let leg = 0; leg < stops.length - 1; leg++) {
			if (t < HOP_MS) {
				const u = easeInOut(t / HOP_MS);
				const toCore = n.rings[leg] >= RING.length - 1;
				const angle = toCore ? a + coreDelta * u : a + Math.sin(u * Math.PI) * twist;
				return polar(angle, stops[leg] + (stops[leg + 1] - stops[leg]) * u);
			}
			t -= HOP_MS;
			if (t < DWELL_MS) return polar(a, stops[leg + 1]);
			t -= DWELL_MS;
		}
		return null;
	}

	let selectedId = $state<string | null>(null);
	let root = $state<HTMLDivElement>();
	let hoverId = $state<string | null>(null);
	let nodeEls: Record<string, SVGGElement> = {};
	const pos = new Map<string, Pos>();

	$effect(() => {
		if (initialLeadId && nodes.some((n) => n.id === initialLeadId)) selectedId = initialLeadId;
	});

	$effect(() => {
		const list = nodes;
		const ids = new Set(list.map((n) => n.id));
		for (const k of [...pos.keys()]) if (!ids.has(k)) pos.delete(k);
		const place = (el: SVGGElement | undefined, p: Pos) => el?.setAttribute('transform', `translate(${p.x.toFixed(1)}, ${p.y.toFixed(1)})`);
		const reduced = typeof window.matchMedia === 'function' && window.matchMedia('(prefers-reduced-motion: reduce)').matches;
		if (reduced || typeof requestAnimationFrame !== 'function') {
			list.forEach((n, i) => place(nodeEls[n.id], orbitTarget(n, i, 0)));
			return;
		}
		let raf = 0;
		const t0 = performance.now();
		let last = t0;
		const stagger = staggerMs(list.length);
		const frame = (nowMs: number) => {
			const t = nowMs - t0;
			const k = smoothK(nowMs - last);
			last = nowMs;
			list.forEach((n, i) => {
				const el = nodeEls[n.id];
				if (!el) return;
				const target = replayPos(n, i, t, stagger) ?? orbitTarget(n, i, t);
				const prev = pos.get(n.id) ?? target;
				const next = { x: prev.x + (target.x - prev.x) * k, y: prev.y + (target.y - prev.y) * k };
				pos.set(n.id, next);
				place(el, next);
			});
			raf = requestAnimationFrame(frame);
		};
		raf = requestAnimationFrame(frame);
		return () => cancelAnimationFrame(raf);
	});

	const selected = $derived(nodes.find((n) => n.id === selectedId) ?? null);
	const anchor = $derived(selected ?? (hoverId ? (nodes.find((n) => n.id === hoverId) ?? null) : null));
	const convertedTotal = $derived(segments.reduce((s, x) => s + x.converted, 0));
</script>

{#if nodes.length === 0}
	<p class="bn-dim py-6 text-center font-mono text-[11.5px]">No journeys for this filter yet.</p>
{:else}
	<div class="fn-space-root relative" data-testid="funnel-radial" bind:this={root}>
		<FullscreenButton target={() => root} />
		{#if anchor}
			<div class="pointer-events-none absolute left-2 top-1.5 z-20 flex items-baseline gap-2 font-mono">
				<span class="bn-text text-[12px] font-semibold">{anchor.name}</span>
				<span class="bn-dim text-[9.5px] uppercase tracking-[0.12em]">{anchor.likelihood}% · {anchor.daysSinceLastTouch}d quiet</span>
			</div>
		{/if}
		<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_noninteractive_element_interactions -->
		<svg viewBox="0 0 {W} {H}" class="fn-radial-canvas block w-full" role="img" aria-label="Clients spiralling from their acquisition source into the conversion core" onclick={() => (selectedId = null)}>
			{#each RING.slice(0, -1) as r, s (s)}
				<g>
					<circle cx={CX} cy={CY} {r} fill="none" stroke="var(--bn-border)" stroke-dasharray="2 6" />
					<text x={CX + 9} y={CY - r + 13} fill="var(--bn-text-3)" font-size="9" font-family="var(--bn-font, monospace)" style="text-transform: uppercase; letter-spacing: 0.14em">{stages[s].label}</text>
				</g>
			{/each}
			{#each segments as seg, sIdx (seg.id)}
				{@const boundary = TOP + sIdx * segSpan}
				{@const mid = boundary + segSpan / 2}
				{@const lp = polar(mid, RING[0] + 24)}
				{@const cos = Math.cos(mid)}
				<g>
					<line x1={polar(boundary, CORE + 12).x} y1={polar(boundary, CORE + 12).y} x2={polar(boundary, RING[0]).x} y2={polar(boundary, RING[0]).y} stroke="var(--bn-border)" stroke-opacity="0.7" />
					<circle cx={polar(mid, RING[0]).x} cy={polar(mid, RING[0]).y} r="2.5" fill={WEDGE[sIdx]} />
					<text x={lp.x} y={Math.round((lp.y + Math.sin(mid) * 7 + 3) * 100) / 100} text-anchor={cos > 0.35 ? 'start' : cos < -0.35 ? 'end' : 'middle'} fill={seg.count > 0 ? 'var(--bn-text-2)' : 'var(--bn-text-3)'} font-size="10.5" font-family="var(--bn-font, monospace)" style="text-transform: uppercase; letter-spacing: 0.14em">
						{seg.label} · {seg.count}{#if seg.converted > 0}<tspan fill="var(--bn-ok)"> ✓{seg.converted}</tspan>{/if}
					</text>
				</g>
			{/each}
			<circle class="fn-hub-ring" cx={CX} cy={CY} r={CORE + 10} fill="none" stroke="var(--bn-ok)" stroke-opacity="0.45" stroke-dasharray="3 7" />
			<circle cx={CX} cy={CY} r={CORE} fill="var(--bn-surface-2)" stroke="var(--bn-ok)" stroke-opacity="0.55" stroke-width="1.2" />
			<text x={CX} y={CY - 2} text-anchor="middle" fill="var(--bn-text-2)" font-size="10" font-family="var(--bn-font, monospace)" style="text-transform: uppercase; letter-spacing: 0.18em">converted</text>
			<text x={CX} y={CY + 14} text-anchor="middle" fill="var(--bn-ok)" font-size="13" font-family="var(--bn-font, monospace)">{convertedTotal}</text>
			{#each nodes as n, i (n.id)}
				{@const color = decayedColor(WEDGE[n.segment] ?? WEDGE[0], n.decay, n.state === 'converted')}
				{@const emphasized = hoverId === n.id || selectedId === n.id}
				{@const pulses = n.state === 'converted' || (n.relationship === 'hot' && n.decay < 0.5)}
				{@const start = polar(wedgeAngle(n, i), RING[0] + 46)}
				<g
					bind:this={() => nodeEls[n.id], (el) => (nodeEls[n.id] = el)}
					data-node={n.id}
					role="button"
					tabindex="0"
					aria-label={n.person ?? n.name}
					aria-pressed={selectedId === n.id}
					onkeydown={(e) => {
						if (e.key !== 'Enter' && e.key !== ' ') return;
						e.preventDefault();
						e.stopPropagation();
						selectedId = selectedId === n.id ? null : n.id;
					}}
					transform="translate({start.x.toFixed(1)}, {start.y.toFixed(1)})"
					style="cursor: pointer"
					onmouseenter={() => (hoverId = n.id)}
					onmouseleave={() => (hoverId = null)}
					onclick={(e) => {
						e.stopPropagation();
						selectedId = selectedId === n.id ? null : n.id;
					}}
				>
					{#if pulses}
						<circle class="fn-halo" r={n.radius + 3} fill="none" stroke={color} style="animation-delay: {(i % 6) * 0.4}s; animation-duration: {n.state === 'converted' ? '3.4s' : '2.6s'}" />
					{/if}
					<circle r={n.radius + 2.5} fill="none" stroke={color} stroke-width="0.8" opacity={emphasized ? 0.9 : 0.35 * decayedOpacity(n.decay)} stroke-dasharray={n.relationship === 'cold' ? '2 3' : undefined} />
					{#if n.relationship === 'hot'}<circle r={n.radius + 5} fill="none" stroke={color} stroke-width="0.6" opacity={0.25 * decayedOpacity(n.decay)} />{/if}
					<circle r={n.radius} fill={color} stroke="var(--bn-bg)" stroke-width="1" opacity={emphasized ? 1 : decayedOpacity(n.decay)} />
					{#if emphasized}
						<text y={-n.radius - 9} text-anchor="middle" fill="var(--bn-text)" font-size="11" font-family="var(--bn-font, monospace)" style="pointer-events: none">{n.name}</text>
					{/if}
				</g>
			{/each}
		</svg>
		{#if selected}
			<FunnelNodeCard node={selected} {stageLabels} stallDays={constants.stallDays} onclose={() => (selectedId = null)} />
		{/if}
		<div class="bn-dim mt-1 flex flex-wrap items-center gap-x-4 gap-y-1 font-mono text-[10px] uppercase tracking-wide">
			<span class="flex items-center gap-1.5"><span class="h-2 w-2 rounded-full" style="background: var(--fn-s1)"></span> hue = where they came from</span>
			<span>rings run outside → in · center = purchase</span>
			<span class="flex items-center gap-1.5"><span class="h-2 w-2 rounded-full" style="background: color-mix(in oklab, var(--bn-err) 70%, var(--fn-s1)); opacity: 0.6"></span> fades red after {constants.decayFadeStart}d quiet → archive at {constants.decayDays}d</span>
			<span class="ml-auto">wedge = Trakyo first touch when attributed · untracked = word of mouth · click a node</span>
		</div>
	</div>
{/if}
