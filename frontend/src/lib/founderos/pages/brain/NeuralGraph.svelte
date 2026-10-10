<!-- The horizontal neural-network view of the same knowledge graph (the operator
     OS components/NeuralGraph.tsx): tools as the labelled input layer, then
     workers → SOP tasks → pillars → the memory core as the single output
     neuron. Deep navy canvas, bundles of signed strands (green positive, red
     negative, opacity by magnitude, a cheap two-pass bloom), a hover spotlight
     that zooms the camera onto the neuron under the cursor, the directory on
     the right, and the radial view's detail cards on click. Pure SVG. -->
<script lang="ts">
	import './kg.css';
	import GraphDirectory from './GraphDirectory.svelte';
	import NeuralDetail from './NeuralDetail.svelte';
	import { toolSlugOf } from './kg';
	import { neuralLayout, NEURAL_H, NEURAL_W, type NeuralStrand } from './neural-layout';
	import type { BrainPage, DirectoryGroup } from './types';

	let { page }: { page: BrainPage } = $props();

	const NAVY_BG = '#0a1024';
	const POS_COLOR = '#5ee8a2';
	const NEG_COLOR = '#ff5f6b';
	const DOT_COLOR = '#e8ecf9';
	const SHIMMER = [
		'animation: ng-shimmer 7s ease-in-out infinite alternate',
		'animation: ng-shimmer 9s ease-in-out -3s infinite alternate',
		'animation: ng-shimmer 11s ease-in-out -6s infinite alternate'
	];
	const STRANDS_PER_EDGE = 5;
	const hashStr = (s: string) => {
		let h = 2166136261;
		for (let i = 0; i < s.length; i++) {
			h ^= s.charCodeAt(i);
			h = Math.imul(h, 16777619);
		}
		return h >>> 0;
	};
	const sparkPts = (id: string, x: number, y: number) => {
		const h = hashStr(id);
		return Array.from({ length: 5 }, (_, i) => `${x + i * 3},${y - (((h >> (i * 4)) % 7) - 3)}`).join(' ');
	};
	const hoverKind = (id: string): string => {
		if (id.startsWith('emp:')) return 'AI agent';
		if (id.startsWith('person:')) return 'human';
		if (id.startsWith('task:')) return 'SOP task';
		if (id.startsWith('tool:')) return 'tool';
		if (id.startsWith('team:')) return 'pillar';
		return 'memory core';
	};

	const graph = $derived(page.graph);
	const layout = $derived(neuralLayout(graph));
	const labelById = $derived(new Map(graph.nodes.map((n) => [n.id, n.label])));
	let hoverId = $state<string | null>(null);
	let selectedId = $state<string | null>(null);

	const nodeIdForRow = (kind: DirectoryGroup['kind'], id: string) =>
		kind === 'tool' ? graph.nodes.find((n) => n.kind === 'tool' && toolSlugOf(n.id) === id)?.id ?? null : id;
	const hoverFromDirectory = (kind: DirectoryGroup['kind'], id: string | null) => (hoverId = id ? nodeIdForRow(kind, id) : null);
	const pickFromDirectory = (kind: DirectoryGroup['kind'], id: string) => (selectedId = nodeIdForRow(kind, id));

	const incident = $derived.by(() => {
		if (!hoverId) return null;
		const set = new Set<string>([hoverId]);
		for (const s of layout.strands) {
			if (s.source === hoverId) set.add(s.target);
			if (s.target === hoverId) set.add(s.source);
		}
		return set;
	});
	const hoverPos = $derived(hoverId ? layout.pos.get(hoverId) ?? null : null);
	const camera = $derived(
		hoverPos ? `transform: scale(1.55); transform-origin: ${hoverPos.x}px ${hoverPos.y}px` : `transform: scale(1); transform-origin: ${NEURAL_W / 2}px ${NEURAL_H / 2}px`
	);

	function strandPath(s: NeuralStrand, k: number) {
		const a = layout.pos.get(s.source)!;
		const b = layout.pos.get(s.target)!;
		const h = hashStr(`${s.source}→${s.target}#${k}`);
		const fray = ((h % 15) - 7) * 0.9;
		const bow = ((((h >> 4) % 61) - 30) / 30) * 42;
		const mx = (a.x + b.x) / 2 + (((h >> 10) % 31) - 15);
		return `M ${a.x} ${a.y + fray * 0.4} C ${mx} ${a.y + bow}, ${mx} ${b.y - bow}, ${b.x} ${b.y + fray * 0.25}`;
	}
	// the strand bundles never change with hover; only their group opacity does
	const bundles = $derived(
		layout.strands.map((s, i) => {
			const a = layout.pos.get(s.source)!;
			const layerFade = 1 - (a.x / NEURAL_W) * 0.45;
			return {
				s,
				gi: i % SHIMMER.length,
				base: (0.05 + Math.abs(s.weight) * 0.16) * layerFade,
				tint: s.weight >= 0 ? POS_COLOR : NEG_COLOR,
				hot: Math.abs(s.weight) > 0.7,
				paths: Array.from({ length: STRANDS_PER_EDGE }, (_, k) => strandPath(s, k))
			};
		})
	);
	const strandOpacity = (s: NeuralStrand, base: number) =>
		!incident ? base : s.source === hoverId || s.target === hoverId ? Math.min(1, base * 5 + 0.3) : base * 0.15;
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="bn-kg kg-b relative w-full overflow-hidden rounded-[10px] border" style="background: {NAVY_BG}" onmouseleave={() => (hoverId = null)} data-part="neural-graph" data-view="neural">
	<svg viewBox="0 0 {NEURAL_W} {NEURAL_H}" class="block w-full" role="img" aria-label="Brain neural view">
		<defs>
			<radialGradient id="ngGlow" cx="50%" cy="50%" r="62%">
				<stop offset="0%" stop-color="#1b2a56" stop-opacity="0.9" />
				<stop offset="55%" stop-color="#111a38" stop-opacity="0.55" />
				<stop offset="100%" stop-color={NAVY_BG} stop-opacity="0" />
			</radialGradient>
		</defs>
		<rect width={NEURAL_W} height={NEURAL_H} fill={NAVY_BG} />
		<rect width={NEURAL_W} height={NEURAL_H} fill="url(#ngGlow)" />
		{#each [0.62, 0.78, 0.94] as k (k)}
			<ellipse cx={NEURAL_W / 2} cy={NEURAL_H / 2} rx={(NEURAL_W / 2) * k} ry={(NEURAL_H / 2) * (k - 0.14)} fill="none" stroke="#31406e" stroke-width="0.7" opacity="0.35" />
		{/each}

		<g style="{camera}; transition: transform 700ms cubic-bezier(0.22, 1, 0.36, 1), transform-origin 700ms cubic-bezier(0.22, 1, 0.36, 1)">
			{#each SHIMMER as style, gi (gi)}
				<g {style}>
					{#each bundles as b (`${b.s.source}→${b.s.target}`)}
						{#if b.gi === gi}
							<g opacity={strandOpacity(b.s, b.base)} style="transition: opacity 300ms" data-strand={`${b.s.source}→${b.s.target}`}>
								{#each b.paths as d, k (k)}
									{#if k === 0}<path {d} fill="none" stroke={b.tint} stroke-width="2.6" opacity="0.3" />{/if}
									<path {d} fill="none" stroke={b.tint} stroke-width="0.55" opacity="0.8" />
									{#if b.hot && k === 2}<path {d} fill="none" stroke="#f4f7ff" stroke-width="0.4" opacity="0.55" />{/if}
								{/each}
							</g>
						{/if}
					{/each}
				</g>
			{/each}

			{#each layout.reports as r (`${r.source}→${r.target}`)}
				{@const a = layout.pos.get(r.source)}
				{@const b = layout.pos.get(r.target)}
				{#if a && b}
					<path
						d="M {a.x} {a.y} C {a.x - 34} {(a.y + b.y) / 2}, {b.x - 34} {(a.y + b.y) / 2}, {b.x} {b.y}"
						fill="none"
						stroke="#8fa3d9"
						stroke-width="0.6"
						opacity={incident && !(incident.has(r.source) && incident.has(r.target)) ? 0.05 : 0.22}
						style="transition: opacity 300ms"
					/>
				{/if}
			{/each}

			{#each layout.layers as layer (layer.kind)}
				{#each layer.nodeIds as id (id)}
					{@const p = layout.pos.get(id)!}
					{@const isSelf = layer.kind === 'self'}
					{@const r = isSelf ? 10 : layer.kind === 'team' ? 5 : 2.6}
					<!-- svelte-ignore a11y_click_events_have_key_events -->
					<g
						transform="translate({p.x},{p.y})"
						opacity={incident && !incident.has(id) ? 0.15 : 1}
						style="transition: opacity 300ms; cursor: pointer; outline: none"
						role="button"
						tabindex="-1"
						aria-label={labelById.get(id) ?? id}
						data-neuron={id}
						onmouseenter={() => (hoverId = id)}
						onclick={() => (selectedId = selectedId === id ? null : id)}
					>
						<title>{labelById.get(id) ?? id}</title>
						<circle r={Math.max(8, r + 5)} fill="transparent" />
						<circle {r} fill={isSelf ? 'var(--bn-kg-mem, #e35c35)' : DOT_COLOR} fill-opacity={isSelf ? 1 : 0.92} />
						{#if isSelf}<circle r={r + 5} fill="none" stroke="var(--bn-kg-mem, #e35c35)" stroke-width="0.8" opacity="0.5" />{/if}
						{#if layer.kind === 'tool'}
							<text x="-10" text-anchor="end" dominant-baseline="middle" font-family="var(--bn-font)" font-size="7.5" fill="#93a3cf">
								{(labelById.get(id) ?? id).toUpperCase().replace(/[^A-Z0-9]+/g, '_').slice(0, 16)}
							</text>
							<polyline points={sparkPts(id, 6, 0)} fill="none" stroke="#5b6d9e" stroke-width="0.8" />
						{/if}
						{#if layer.kind === 'team'}
							<text x="9" dominant-baseline="middle" font-family="var(--bn-font)" font-size="8.5" fill="#aeb9e2">{labelById.get(id)}</text>
						{/if}
						{#if isSelf}
							<text y="24" text-anchor="middle" font-family="var(--bn-font)" font-size="9.5" font-weight="600" fill="var(--bn-kg-mem, #e35c35)">Engine</text>
						{/if}
					</g>
				{/each}
			{/each}
		</g>
	</svg>

	<div class="pointer-events-none absolute inset-x-0 top-3 flex justify-around px-6 transition-opacity duration-200 {hoverId ? 'opacity-20' : 'opacity-100'}" data-part="layer-names">
		{#each layout.layers as layer (layer.kind)}
			<div class="font-mono text-[10px] font-semibold uppercase tracking-[0.16em]" style="color: #c7d2f2; text-shadow: 0 1px 6px #0a1024, 0 0 3px #0a1024">{layer.name}</div>
		{/each}
	</div>

	{#if hoverId}
		<div class="pointer-events-none absolute inset-x-0 top-2.5 z-30 flex justify-center" data-part="neural-readout">
			<div class="flex items-baseline gap-2 font-mono" style="text-shadow: 0 1px 6px #0a1024, 0 0 3px #0a1024">
				<span class="text-[13px] font-semibold" style="color: #e8ecf9">{labelById.get(hoverId) ?? hoverId}</span>
				<span class="text-[9.5px] uppercase tracking-[0.16em]" style="color: #7d8cbd">{hoverKind(hoverId)}</span>
			</div>
		</div>
	{/if}

	<div class="kg-neural-dir absolute bottom-3 right-3 top-16 z-[6] flex w-72">
		<GraphDirectory groups={page.directory} onPick={pickFromDirectory} onHover={hoverFromDirectory} class="h-full" />
	</div>

	{#if selectedId}
		<div class="kg-bs kg-bg95 kg-blur absolute inset-y-0 right-0 z-10 w-80 border-l" data-part="neural-detail">
			<NeuralDetail
				nodeId={selectedId}
				{graph}
				agents={page.agents}
				departments={page.departments}
				people={page.people}
				tasks={page.tasks}
				runsByAgent={page.runsByAgent}
				onSelect={(id) => (selectedId = id)}
				onClose={() => (selectedId = null)}
			/>
		</div>
	{/if}
</div>

<style>
	@media (max-width: 820px) {
		.kg-neural-dir {
			display: none;
		}
	}
</style>
