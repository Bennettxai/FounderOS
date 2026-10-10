<!-- The stage: containers, frames and cards as positioned HTML over two SVG
     layers (structural traces below, the selection's relations on top), a
     pan/zoom camera that mutates the world transform directly (no render per
     frame), and the perf rules learned on the concept: the dot grid lives in
     screen space, will-change only on the world, blur and backdrop filters
     drop while the camera moves. Port of FounderOS v1
     components/blueprint/HierarchyCanvas.tsx. -->
<script lang="ts" module>
	export type FocusMode =
		| { mode: 'none' }
		| { mode: 'filter'; match: Set<string> }
		| { mode: 'focus'; selected: string; hot: Set<string> };
</script>

<script lang="ts">
	import { onMount } from 'svelte';
	import { ChevronDown } from '$lib/founderos/icons';
	import { KIND_ICON, type HContainer, type HGroup, type Hierarchy, type HierarchyIndex } from './hierarchy';
	import { fitAllCam, fitTopCam, flyToCam, GRID, rep, roundedPath, zoomCam, type Box, type Cam, type DrawnEdge, type Layout } from './hierarchy-layout';
	import HierarchyNode from './HierarchyNode.svelte';
	import { iconFor } from './icons';

	let {
		idx,
		layout,
		spine,
		focusEdges,
		focusColor,
		focus,
		lit,
		onselect,
		ontoggle,
		onfly
	}: {
		h: Hierarchy;
		idx: HierarchyIndex;
		layout: Layout;
		spine: DrawnEdge[];
		focusEdges: DrawnEdge[];
		focusColor: string | null;
		focus: FocusMode;
		lit: Set<string>;
		onselect: (id: string | null) => void;
		ontoggle: (id: string) => void;
		onfly: (id: string) => void;
	} = $props();

	const EASE = (t: number) => 1 - Math.pow(1 - t, 5);

	let canvasEl: HTMLDivElement | undefined = $state();
	let worldEl: HTMLDivElement | undefined = $state();
	let flareEl: SVGGElement | undefined = $state();
	let spotEl: HTMLDivElement | undefined = $state();
	let pctEl: HTMLDivElement | undefined = $state();
	let cam: Cam = { x: 0, y: 0, s: 1 };
	let anim: number | null = null;
	let moveTimer: ReturnType<typeof setTimeout> | null = null;
	let panning: { x: number; y: number; cx: number; cy: number; moved: boolean } | null = null;
	const mounted = new Set<string>();
	// plain object, not $state: the first-paint stagger is read once per card, never re-rendered
	const paint = { first: true };

	// jsdom (and a canvas not laid out yet) reports 0: fall back to a sane stage
	const view = () => ({ w: canvasEl?.clientWidth || 1200, h: canvasEl?.clientHeight || 800 });

	function applyCam() {
		const c = canvasEl;
		const w = worldEl;
		if (!c || !w) return;
		const { x, y, s } = cam;
		w.style.transform = `translate(${x}px, ${y}px) scale(${s})`;
		if (pctEl) pctEl.textContent = `${Math.round(s * 100)}%`;
		let pitch = GRID * s;
		while (pitch < 18) pitch *= 2; // double the pitch when dots would crowd
		c.style.setProperty('--gs', `${pitch}px`);
		c.style.setProperty('--gox', `${((x % pitch) + pitch) % pitch}px`);
		c.style.setProperty('--goy', `${((y % pitch) + pitch) % pitch}px`);
		c.style.setProperty('--ga', String(Math.max(0.35, Math.min(1, (s - 0.15) / 0.5))));
		c.classList.add('is-moving');
		if (moveTimer) clearTimeout(moveTimer);
		moveTimer = setTimeout(() => c.classList.remove('is-moving'), 180);
	}

	function tween(to: Cam, dur = 620) {
		if (anim) cancelAnimationFrame(anim);
		const from = { ...cam };
		const t0 = performance.now();
		const step = (now: number) => {
			const t = Math.min(1, (now - t0) / dur);
			const k = EASE(t);
			cam = { x: from.x + (to.x - from.x) * k, y: from.y + (to.y - from.y) * k, s: from.s + (to.s - from.s) * k };
			applyCam();
			if (t < 1) anim = requestAnimationFrame(step);
		};
		anim = requestAnimationFrame(step);
	}

	export function fitAll(animate = true) {
		const { w, h: hh } = view();
		const to = fitAllCam(layout, w, hh);
		if (animate) tween(to);
		else {
			cam = to;
			applyCam();
		}
	}
	export function fitTop() {
		cam = fitTopCam(layout, view().w);
		applyCam();
	}
	export function flyTo(id: string, s: number) {
		const b = rep(layout, idx, id);
		if (!b) return;
		const { w, h: hh } = view();
		tween(flyToCam(b, s, w, hh));
	}
	export function zoomBy(f: number) {
		const { w, h: hh } = view();
		tween(zoomCam(cam, f, w / 2, hh / 2), 260);
	}
	/** Run a flare along a structural edge; resolves when it arrives. */
	export function runFlare(edgeId: string, ms = 560): Promise<void> {
		return new Promise<void>((resolve) => {
			const layer = flareEl;
			const path = worldEl?.querySelector<SVGPathElement>(`.bh-edge[data-id="${edgeId}"] .line`);
			if (!layer || !path || typeof path.getTotalLength !== 'function') return resolve();
			const len = path.getTotalLength();
			const c = document.createElementNS('http://www.w3.org/2000/svg', 'circle');
			c.setAttribute('class', 'bh-flare');
			c.setAttribute('r', '9');
			layer.appendChild(c);
			const t0 = performance.now();
			const step = (now: number) => {
				const t = Math.min(1, (now - t0) / ms);
				const p = path.getPointAtLength(len * t);
				c.setAttribute('cx', String(p.x));
				c.setAttribute('cy', String(p.y));
				if (t < 1) requestAnimationFrame(step);
				else {
					c.remove();
					resolve();
				}
			};
			requestAnimationFrame(step);
		});
	}

	// first view + resize: fit the width, anchor the top. The wheel listener
	// must be non-passive to stop the page scrolling under the map.
	onMount(() => {
		fitTop();
		const onResize = () => fitTop();
		const stop = (e: WheelEvent) => e.preventDefault();
		window.addEventListener('resize', onResize);
		canvasEl?.addEventListener('wheel', stop, { passive: false });
		queueMicrotask(() => (paint.first = false));
		return () => {
			window.removeEventListener('resize', onResize);
			canvasEl?.removeEventListener('wheel', stop);
			if (anim) cancelAnimationFrame(anim);
			if (moveTimer) clearTimeout(moveTimer);
		};
	});

	// ── pointer: pan, zoom, glow spot ───────────────────────────────────
	const isChrome = (t: EventTarget | null) => !!(t as HTMLElement | null)?.closest?.('.bh-node, .bh-frame-head, .bh-ch');
	function onpointerdown(ev: PointerEvent) {
		if (ev.button !== 0 || isChrome(ev.target)) return;
		panning = { x: ev.clientX, y: ev.clientY, cx: cam.x, cy: cam.y, moved: false };
		canvasEl?.classList.add('is-panning');
		canvasEl?.setPointerCapture?.(ev.pointerId);
	}
	function onpointermove(ev: PointerEvent) {
		const c = canvasEl;
		if (!c) return;
		const r = c.getBoundingClientRect();
		if (spotEl) spotEl.style.transform = `translate(${ev.clientX - r.left}px, ${ev.clientY - r.top}px)`;
		const p = panning;
		if (!p) return;
		const dx = ev.clientX - p.x;
		const dy = ev.clientY - p.y;
		if (Math.abs(dx) + Math.abs(dy) > 3) p.moved = true;
		cam.x = p.cx + dx;
		cam.y = p.cy + dy;
		applyCam();
	}
	function onpointerup(ev: PointerEvent) {
		const p = panning;
		if (p && !p.moved && !isChrome(ev.target)) onselect(null);
		panning = null;
		canvasEl?.classList.remove('is-panning');
	}
	function onwheel(ev: WheelEvent) {
		const c = canvasEl;
		if (!c) return;
		const r = c.getBoundingClientRect();
		if (ev.ctrlKey || ev.metaKey) cam = zoomCam(cam, Math.exp(-ev.deltaY * 0.01), ev.clientX - r.left, ev.clientY - r.top);
		else {
			cam.x -= ev.deltaX;
			cam.y -= ev.deltaY;
		}
		applyCam();
	}
	function ondblclick(ev: MouseEvent) {
		const n = (ev.target as HTMLElement).closest<HTMLElement>('.bh-node');
		if (n?.dataset.id) onfly(n.dataset.id);
	}
	function onNodesClick(ev: MouseEvent) {
		const n = (ev.target as HTMLElement).closest<HTMLElement>('.bh-node');
		if (!n?.dataset.id) return;
		if (n.dataset.mode === 'g') ontoggle(n.dataset.id);
		else onselect(n.dataset.id);
	}
	function onFramesClick(ev: MouseEvent) {
		const head = (ev.target as HTMLElement).closest<HTMLElement>('.bh-frame-head');
		const id = head?.parentElement?.dataset.id;
		if (!id) return;
		if ((ev.target as HTMLElement).closest('.bh-fc')) ontoggle(id);
		else onselect(id);
	}
	function onFramesDoubleClick(ev: MouseEvent) {
		const id = (ev.target as HTMLElement).closest<HTMLElement>('.bh-frame-head')?.parentElement?.dataset.id;
		if (id) ontoggle(id);
	}
	function onContainersClick(ev: MouseEvent) {
		const id = (ev.target as HTMLElement).closest<HTMLElement>('.bh-ch')?.parentElement?.dataset.id;
		if (id) onselect(id);
	}

	// ── focus classes ───────────────────────────────────────────────────
	function stateClass(id: string, isContainer: boolean): string {
		const parts: string[] = [];
		if (lit.has(id)) parts.push('is-lit');
		if (focus.mode === 'filter') {
			const hit = isContainer || focus.match.has(id);
			if (!hit) parts.push('is-dim');
			else if (!isContainer) parts.push('is-match');
		} else if (focus.mode === 'focus') {
			if (id === focus.selected) parts.push('is-selected');
			else if (focus.hot.has(id)) parts.push('is-hot');
			else parts.push('is-dim');
		}
		return parts.join(' ');
	}
	const edgesDim = $derived(focus.mode !== 'none');

	function onAnimEnd(e: AnimationEvent) {
		const el = e.currentTarget as HTMLElement;
		const id = el.dataset.id;
		if (!id) return;
		mounted.add(id);
		el.classList.remove('is-enter', 'is-new');
	}

	const split = $derived.by(() => {
		const containers: Box[] = [];
		const frames: Box[] = [];
		const nodes: Box[] = [];
		for (const b of layout.boxes.values()) {
			if (b.kind === 'container') containers.push(b);
			else if (b.kind === 'frame') frames.push(b);
			else nodes.push(b);
		}
		return { containers, frames, nodes };
	});

	function enterClass(id: string): string {
		if (mounted.has(id)) return '';
		return paint.first ? 'is-enter' : 'is-new';
	}

	const mid = (e: DrawnEdge) => {
		const pn = e.pts[e.pts.length - 1];
		const a = e.pts[e.pts.length - 2] ?? e.pts[0];
		return { x: (a.x + pn.x) / 2, y: (a.y + pn.y) / 2 };
	};
	const focusStyle = $derived(focusColor ? `--ec: ${focusColor}` : undefined);
</script>

{#snippet edge(e: DrawnEdge, extra: string)}
	{@const d = roundedPath(e.pts)}
	{#if e.bare}
		<g class="bh-edge {e.cls} {extra}" data-id={e.id}>
			<path class="glow" {d} />
			<path class="line" {d} />
		</g>
	{:else}
		{@const p0 = e.pts[0]}
		{@const pn = e.pts[e.pts.length - 1]}
		{@const m = mid(e)}
		<g class="bh-edge {e.cls} {extra}" data-id={e.id} style={e.color ? `--ec: ${e.color}` : undefined}>
			<path class="glow" {d} />
			<path class="line" {d} />
			<path class="flow" {d} />
			<circle class="port" cx={p0.x} cy={p0.y} r="4" />
			<circle class="port" cx={pn.x} cy={pn.y} r="4" />
			{#if e.plus}
				<circle class="port" cx={m.x} cy={m.y} r="6" />
				<path class="port-plus" d="M {m.x - 3} {m.y} H {m.x + 3} M {m.x} {m.y - 3} V {m.y + 3}" />
			{/if}
		</g>
	{/if}
{/snippet}

<!-- svelte-ignore a11y_no_noninteractive_tabindex -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
	bind:this={canvasEl}
	class="bh-canvas"
	tabindex="0"
	{onpointerdown}
	{onpointermove}
	{onpointerup}
	{onwheel}
	{ondblclick}
>
	<div bind:this={spotEl} class="bh-glow-spot"></div>
	<div bind:this={worldEl} class="bh-world">
		<svg class="bh-edges" xmlns="http://www.w3.org/2000/svg">
			<defs>
				<radialGradient id="bh-flare">
					<stop offset="0" stop-color="var(--bh-flare)" />
					<stop offset="0.35" stop-color="var(--accent)" stop-opacity="0.85" />
					<stop offset="1" stop-color="var(--accent)" stop-opacity="0" />
				</radialGradient>
			</defs>
			<g>
				{#each spine as e, i (`${e.id}-${i}`)}
					{@render edge(e, `${edgesDim ? 'is-dim' : ''} ${lit.has(e.id) ? 'is-lit' : ''}`)}
				{/each}
			</g>
			<g bind:this={flareEl}></g>
		</svg>
		<!-- svelte-ignore a11y_click_events_have_key_events -->
		<div class="bh-layer" onclick={onContainersClick}>
			{#each split.containers as b (b.id)}
				{@const c = b.it as HContainer}
				{@const Icon = iconFor(c.icon || 'server')}
				<div class="bh-container k-{c.kind} {stateClass(b.id, true)}" data-id={b.id} style="width: {b.w}px; height: {b.h}px; transform: translate({b.x}px, {b.y}px)">
					{#if c.kind !== 'operator'}
						<div class="bh-ch">
							<span class="bh-ci"><Icon /></span>
							<div>
								<div class="bh-ct">{c.name}</div>
								<div class="bh-cs">{c.sub}</div>
							</div>
						</div>
						{#if Object.keys(c.facts).length > 0}
							<div class="bh-cf">
								{#each Object.entries(c.facts) as [k, v] (k)}
									<span>{k} · {v}</span>
								{/each}
							</div>
						{/if}
					{/if}
				</div>
			{/each}
		</div>
		<!-- svelte-ignore a11y_click_events_have_key_events -->
		<div class="bh-layer" onclick={onFramesClick} ondblclick={onFramesDoubleClick}>
			{#each split.frames as b (b.id)}
				{@const g = b.it as HGroup}
				{@const Icon = iconFor(g.icon || KIND_ICON[g.kind])}
				<div class="bh-frame k-{g.kind} {stateClass(b.id, false)}" data-id={b.id} style="width: {b.w}px; height: {b.h}px; transform: translate({b.x}px, {b.y}px); --c: var(--bh-k-{g.kind}, var(--text-2))">
					<div class="bh-frame-head">
						<span class="bh-fi"><Icon /></span>
						<span class="bh-ft">{g.name}</span>
						<span class="bh-fs">{g.sub}</span>
						<span class="bh-fc">
							{#if g.badge}<span class="bh-gb s-{g.status ?? ''}">{g.badge}</span>{:else}<span class="n">{idx.countLeaves(g)}</span>{/if}
							<ChevronDown />
						</span>
					</div>
					<div class="bh-frame-band"></div>
				</div>
			{/each}
		</div>
		<!-- svelte-ignore a11y_click_events_have_key_events -->
		<div class="bh-layer" onclick={onNodesClick}>
			{#each split.nodes as b, i (b.id)}
				{@const fresh = !mounted.has(b.id)}
				<HierarchyNode
					box={b}
					className="{enterClass(b.id)} {stateClass(b.id, false)}"
					enterDelay={fresh && paint.first ? `${Math.min(i * 12, 600)}ms` : undefined}
					onanimationend={onAnimEnd}
				/>
			{/each}
		</div>
		<svg class="bh-edges bh-edges-top" xmlns="http://www.w3.org/2000/svg" style={focusStyle}>
			<g>
				{#each focusEdges as e, i (`${e.id}-${i}`)}
					{@render edge({ ...e, color: e.color ?? focusColor ?? undefined }, 'is-anim')}
				{/each}
			</g>
		</svg>
		<div class="bh-layer bh-labels">
			{#each spine as e, i (`${e.id}-${i}`)}
				{#if e.label && e.labelAt}
					<div class="bh-pill {e.cls} {edgesDim ? 'is-dim' : ''} {lit.has(e.id) ? 'hot' : ''}" data-edge={e.id} style="left: {e.labelAt.x}px; top: {e.labelAt.y}px">{e.label}</div>
				{/if}
			{/each}
			{#each focusEdges as e, i (`${e.id}-${i}`)}
				{#if e.label && e.labelAt}
					<div class="bh-pill dyn" style="left: {e.labelAt.x}px; top: {e.labelAt.y}px; {focusStyle ?? ''}">{e.label}</div>
				{/if}
			{/each}
		</div>
	</div>
	<div class="bh-hud bh-hud-zoom">
		<button type="button" class="bn-pressable bh-hud-btn" aria-label="Zoom in" onclick={() => zoomBy(1.25)}><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"><path d="M12 5v14M5 12h14" /></svg></button>
		<button type="button" class="bn-pressable bh-hud-btn" aria-label="Zoom out" onclick={() => zoomBy(0.8)}><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"><path d="M5 12h14" /></svg></button>
		<div bind:this={pctEl} class="bh-hud-pct">100%</div>
	</div>
</div>
