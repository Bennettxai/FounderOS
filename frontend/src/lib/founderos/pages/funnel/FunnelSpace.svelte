<!-- The funnel as an open space (FounderOS v1 components/FunnelSpace.tsx): stage
     hubs left → right; every client enters at first touch, travels its real
     hub path, then orbits the stage it is in now. Size = likelihood, ring =
     relationship, hue = stage, fade-to-red = quiet decay, green = converted.
     Heavy: loaded lazily by FunnelGraph behind a same-aspect skeleton. -->
<script lang="ts">
	import FullscreenButton from './FullscreenButton.svelte';
	import FunnelNodeCard from './FunnelNodeCard.svelte';
	import type { FunnelNode, Summary } from './types';
	import { decayedColor, decayedOpacity, easeInOut, orbitSpread, rnd, smoothK, usd } from './viz';

	let {
		nodes,
		summary,
		stages,
		stageLabels,
		constants,
		initialLeadId = null
	}: {
		nodes: FunnelNode[];
		summary: Summary;
		stages: { id: string; label: string }[];
		stageLabels: Record<string, string>;
		constants: { stallDays: number; decayDays: number; decayFadeStart: number };
		initialLeadId?: string | null;
	} = $props();

	const W = 1100;
	// v1's open space (components/FunnelSpace.tsx): 1100×460, the spine
	// slightly above center; stage titles live above each hub.
	const H = 460;
	const CY = H / 2 - 14;
	const HUB_X0 = 100;
	const hubGap = $derived((W - 200) / (stages.length - 1));
	const hubX = (i: number) => HUB_X0 + i * hubGap;
	const hubR = (i: number) => (i === stages.length - 1 ? 34 : 24);
	const SEG = ['var(--fn-s0)', 'var(--fn-s1)', 'var(--fn-s2)', 'var(--fn-s3)', 'var(--fn-s4)'];
	const ENTER_DELAY = 500;
	const HOP_MS = 950;
	const DWELL_MS = 420;
	const PULSES = 9;
	const GOLDEN = 2.399963;
	const staggerMs = (count: number) => Math.min(340, 5000 / Math.max(1, count));
	type Pos = { x: number; y: number };

	let selectedId = $state<string | null>(null);
	let root = $state<HTMLDivElement>();
	let hoverId = $state<string | null>(null);
	let nodeEls: Record<string, SVGGElement> = {};
	let pulseEls: SVGCircleElement[] = [];
	const pos = new Map<string, Pos>();

	$effect(() => {
		if (initialLeadId && nodes.some((n) => n.id === initialLeadId)) selectedId = initialLeadId;
	});

	function orbitTarget(n: FunnelNode, i: number, t: number, spread: number): Pos {
		const speed = (0.0001 + rnd(i, 2) * 0.00012) * (rnd(i, 3) > 0.5 ? 1 : -1);
		const a = i * GOLDEN + rnd(i, 4) * 0.5 + t * speed;
		const r = hubR(n.currentHub) + 5 + ((1 - n.likelihood / 100) * 20 + rnd(i, 1) * 7) * spread;
		const yFlat = 0.85 / Math.sqrt(spread);
		return {
			x: hubX(n.currentHub) + Math.cos(a) * r + Math.sin(t * 0.0007 + i * 2.1) * 1.6,
			y: CY + Math.sin(a) * r * yFlat + Math.sin(t * 0.0009 + i * 1.3) * 2.2
		};
	}

	function replayPos(n: FunnelNode, i: number, tMs: number, stagger: number): Pos | null {
		let t = tMs - (ENTER_DELAY + i * stagger);
		if (t <= 0) return { x: hubX(n.hubs[0]) - 60 - rnd(i, 5) * 30, y: CY + (rnd(i, 6) - 0.5) * 80 };
		for (let leg = 0; leg < n.hubs.length - 1; leg++) {
			if (t < HOP_MS) {
				const u = easeInOut(t / HOP_MS);
				const x0 = hubX(n.hubs[leg]);
				const x1 = hubX(n.hubs[leg + 1]);
				return { x: x0 + (x1 - x0) * u, y: CY - Math.sin(u * Math.PI) * (18 + rnd(i, 7) * 22) * (rnd(i, 8) > 0.5 ? 1 : -1) };
			}
			t -= HOP_MS;
			if (t < DWELL_MS) return { x: hubX(n.hubs[leg + 1]), y: CY };
			t -= DWELL_MS;
		}
		return null;
	}

	$effect(() => {
		const list = nodes;
		const ids = new Set(list.map((n) => n.id));
		for (const k of [...pos.keys()]) if (!ids.has(k)) pos.delete(k);
		const counts = new Map<number, number>();
		for (const n of list) counts.set(n.currentHub, (counts.get(n.currentHub) ?? 0) + 1);
		const spread = (n: FunnelNode) => orbitSpread(counts.get(n.currentHub) ?? 0);
		const place = (el: SVGGElement | undefined, p: Pos) => el?.setAttribute('transform', `translate(${p.x.toFixed(1)}, ${p.y.toFixed(1)})`);
		const reduced = typeof window.matchMedia === 'function' && window.matchMedia('(prefers-reduced-motion: reduce)').matches;
		if (reduced || typeof requestAnimationFrame !== 'function') {
			list.forEach((n, i) => place(nodeEls[n.id], orbitTarget(n, i, 0, spread(n))));
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
				const target = replayPos(n, i, t, stagger) ?? orbitTarget(n, i, t, spread(n));
				const prev = pos.get(n.id) ?? target;
				const next = { x: prev.x + (target.x - prev.x) * k, y: prev.y + (target.y - prev.y) * k };
				pos.set(n.id, next);
				place(el, next);
			});
			for (let p = 0; p < PULSES; p++) {
				const el = pulseEls[p];
				if (!el) continue;
				const leg = p % (stages.length - 1);
				const u = (t * (0.00008 + rnd(p, 9) * 0.00005) + rnd(p, 10)) % 1;
				el.setAttribute('cx', String(hubX(leg) + (hubX(leg + 1) - hubX(leg)) * u));
				el.setAttribute('cy', String(CY + Math.sin(u * Math.PI * 2 + p) * 2));
				el.setAttribute('opacity', String(0.5 * Math.sin(u * Math.PI)));
			}
			raf = requestAnimationFrame(frame);
		};
		raf = requestAnimationFrame(frame);
		return () => cancelAnimationFrame(raf);
	});

	const selected = $derived(nodes.find((n) => n.id === selectedId) ?? null);
	const colorOf = (n: FunnelNode) => decayedColor(SEG[n.currentHub] ?? SEG[0], n.decay, n.state === 'converted');
</script>

{#if nodes.length === 0}
	<p class="bn-dim py-6 text-center font-mono text-[11.5px]">No journeys for this filter yet.</p>
{:else}
	<div class="fn-space-root relative" data-testid="funnel-space" bind:this={root}>
		<!-- fullscreen: the space is built to be filmed -->
		<FullscreenButton target={() => root} />
		<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_noninteractive_element_interactions -->
		<svg viewBox="0 0 {W} {H}" class="block w-full" role="img" aria-label="Clients orbiting their funnel stage from first touch to conversion" onclick={() => (selectedId = null)}>
			<line x1={hubX(0)} x2={hubX(stages.length - 1)} y1={CY} y2={CY} stroke="var(--bn-border)" />
			{#each Array.from({ length: PULSES }) as _, p (p)}
				<circle bind:this={() => pulseEls[p], (el) => (pulseEls[p] = el)} r="1.8" fill="var(--fn-warm)" opacity="0" />
			{/each}
			{#each stages as s, i (s.id)}
				{@const r = hubR(i)}
				{@const row = summary.stages[i]}
				<g>
					<circle class="fn-hub-ring" cx={hubX(i)} cy={CY} r={r + 10} fill="none" stroke={SEG[i]} stroke-opacity="0.45" stroke-dasharray="3 7" style="animation-delay: {i * 0.6}s" />
					<circle cx={hubX(i)} cy={CY} r={r} fill="var(--bn-surface-2)" stroke={SEG[i]} stroke-opacity="0.6" stroke-width="1.2" />
					<circle cx={hubX(i)} cy={CY} r="2.5" fill={SEG[i]} />
					<text x={hubX(i)} y={CY - r - 52} text-anchor="middle" fill="var(--bn-text-2)" font-size="10.5" font-family="var(--bn-font, monospace)" style="text-transform: uppercase; letter-spacing: 0.16em">{s.label}</text>
					<text x={hubX(i)} y={CY - r - 36} text-anchor="middle" fill="var(--bn-text-3)" font-size="10" font-family="var(--bn-font, monospace)">
						{row ? `${row.total}${row.conversionFromPrev != null ? ` · ${row.conversionFromPrev}%` : ''}` : ''}
					</text>
				</g>
			{/each}
			{#each nodes as n, i (n.id)}
				{@const color = colorOf(n)}
				{@const emphasized = hoverId === n.id || selectedId === n.id}
				{@const pulses = n.state === 'converted' || (n.relationship === 'hot' && n.decay < 0.5)}
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
					transform="translate({hubX(n.hubs[0]) - 60}, {CY})"
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
						<text y={-n.radius - 9} text-anchor="middle" fill="var(--bn-text)" font-size="11" font-family="var(--bn-font, monospace)" style="pointer-events: none">
							{n.name}{n.state === 'converted' && n.amountUsd != null ? ` · ${usd(n.amountUsd)}` : ''}
						</text>
					{/if}
				</g>
			{/each}
		</svg>
		{#if selected}
			<FunnelNodeCard node={selected} {stageLabels} stallDays={constants.stallDays} onclose={() => (selectedId = null)} />
		{/if}
		<div class="bn-dim mt-1 flex flex-wrap items-center gap-x-4 gap-y-1 font-mono text-[10px] uppercase tracking-wide">
			<span class="flex items-center gap-1.5"><span class="h-2 w-2 rounded-full" style="background: var(--fn-s1)"></span> hue = its stage</span>
			<span class="flex items-center gap-1.5"><span class="h-2 w-2 rounded-full" style="background: color-mix(in oklab, var(--bn-err) 70%, var(--fn-s1)); opacity: 0.6"></span> fades red after {constants.decayFadeStart}d quiet → archive at {constants.decayDays}d</span>
			<span class="flex items-center gap-1.5"><span class="h-2 w-2 rounded-full" style="background: var(--bn-ok)"></span> converted</span>
			<span class="ml-auto">size + closeness = ICP fit · movement resets the clock · click a node</span>
		</div>
	</div>
{/if}
