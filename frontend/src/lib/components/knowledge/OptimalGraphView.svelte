<script lang="ts">
	import { onMount } from 'svelte';
	import { optimalStore, type OptimalNode } from '$lib/stores/optimal';

	interface Props {
		onNodeSelect?: (node: { slug: string; name: string }) => void;
	}

	let { onNodeSelect }: Props = $props();

	let nodes: OptimalNode[] = $state([]);
	let selectedSlug: string | null = $state(null);
	let loading = $state(true);
	let svgEl: SVGSVGElement | undefined = $state();
	let width = $state(800);
	let height = $state(600);

	interface GraphNode {
		slug: string;
		name: string;
		signalCount: number;
		x: number;
		y: number;
		vx: number;
		vy: number;
		radius: number;
		color: string;
	}

	interface GraphEdge {
		source: string;
		target: string;
	}

	let graphNodes: GraphNode[] = $state([]);
	let graphEdges: GraphEdge[] = $state([]);
	let simRunning = $state(false);

	const COLORS = [
		'#6366f1', '#8b5cf6', '#ec4899', '#f43f5e', '#f97316',
		'#eab308', '#22c55e', '#14b8a6', '#06b6d4', '#3b82f6',
		'#a855f7', '#e879f9', '#fb923c', '#4ade80', '#2dd4bf',
	];

	function buildGraph(nodeList: OptimalNode[]) {
		const real = nodeList.filter(n => n.signal_count > 0 || n.slug === '00-getting-started');
		const cx = width / 2;
		const cy = height / 2;

		graphNodes = real.map((n, i) => {
			const angle = (2 * Math.PI * i) / real.length;
			const spread = Math.min(width, height) * 0.3;
			return {
				slug: n.slug,
				name: n.name || n.slug,
				signalCount: n.signal_count,
				x: cx + Math.cos(angle) * spread + (Math.random() - 0.5) * 40,
				y: cy + Math.sin(angle) * spread + (Math.random() - 0.5) * 40,
				vx: 0,
				vy: 0,
				radius: Math.max(20, Math.min(45, 15 + n.signal_count * 3)),
				color: COLORS[i % COLORS.length],
			};
		});

		graphEdges = [];
		for (let i = 0; i < graphNodes.length; i++) {
			for (let j = i + 1; j < graphNodes.length; j++) {
				const a = nodeList.find(n => n.slug === graphNodes[i].slug);
				const b = nodeList.find(n => n.slug === graphNodes[j].slug);
				if (!a || !b) continue;
				const aText = (a.context_md || '').toLowerCase();
				const bName = (b.name || b.slug).toLowerCase();
				const bText = (b.context_md || '').toLowerCase();
				const aName = (a.name || a.slug).toLowerCase();
				if (aText.includes(bName) || bText.includes(aName)) {
					graphEdges.push({ source: graphNodes[i].slug, target: graphNodes[j].slug });
				}
			}
		}

		runSimulation();
	}

	function runSimulation() {
		simRunning = true;
		let iterations = 0;
		const maxIter = 120;
		const cx = width / 2;
		const cy = height / 2;

		function tick() {
			if (iterations >= maxIter) { simRunning = false; return; }
			iterations++;
			const alpha = 1 - iterations / maxIter;

			for (const node of graphNodes) {
				let fx = 0, fy = 0;

				for (const other of graphNodes) {
					if (other === node) continue;
					const dx = node.x - other.x;
					const dy = node.y - other.y;
					const dist = Math.max(Math.sqrt(dx * dx + dy * dy), 1);
					const repulsion = 8000 / (dist * dist);
					fx += (dx / dist) * repulsion * alpha;
					fy += (dy / dist) * repulsion * alpha;
				}

				for (const edge of graphEdges) {
					let other: GraphNode | undefined;
					if (edge.source === node.slug) other = graphNodes.find(n => n.slug === edge.target);
					if (edge.target === node.slug) other = graphNodes.find(n => n.slug === edge.source);
					if (!other) continue;
					const dx = other.x - node.x;
					const dy = other.y - node.y;
					const dist = Math.sqrt(dx * dx + dy * dy);
					const attraction = dist * 0.005 * alpha;
					fx += (dx / dist) * attraction;
					fy += (dy / dist) * attraction;
				}

				fx += (cx - node.x) * 0.01 * alpha;
				fy += (cy - node.y) * 0.01 * alpha;

				node.vx = (node.vx + fx) * 0.6;
				node.vy = (node.vy + fy) * 0.6;
				node.x += node.vx;
				node.y += node.vy;

				node.x = Math.max(node.radius + 10, Math.min(width - node.radius - 10, node.x));
				node.y = Math.max(node.radius + 10, Math.min(height - node.radius - 10, node.y));
			}

			graphNodes = [...graphNodes];
			requestAnimationFrame(tick);
		}

		requestAnimationFrame(tick);
	}

	function getNodePos(slug: string): { x: number; y: number } {
		const n = graphNodes.find(gn => gn.slug === slug);
		return n ? { x: n.x, y: n.y } : { x: 0, y: 0 };
	}

	$effect(() => {
		if (svgEl) {
			const rect = svgEl.getBoundingClientRect();
			width = rect.width || 800;
			height = rect.height || 600;
		}
	});

	onMount(() => {
		const unsub = optimalStore.subscribe((s) => {
			nodes = s.nodes;
			loading = s.loading;
			if (s.nodes.length > 0 && graphNodes.length === 0) {
				buildGraph(s.nodes);
			}
		});
		optimalStore.loadNodes();
		return unsub;
	});

	let selectedNode = $derived(nodes.find(n => n.slug === selectedSlug));
</script>

<div class="ogv-root">
	<aside class="ogv-sidebar">
		<div class="ogv-sidebar__header">
			<span class="ogv-sidebar__title">NODES</span>
			<span class="ogv-sidebar__count">{nodes.length}</span>
		</div>
		<div class="ogv-sidebar__list">
			{#each nodes as node}
				<button
					class="ogv-node-item"
					class:active={selectedSlug === node.slug}
					onclick={() => { selectedSlug = node.slug; onNodeSelect?.({ slug: node.slug, name: node.name }); }}
				>
					<span class="ogv-node-name">{node.name || node.slug}</span>
					{#if node.signal_count > 0}
						<span class="ogv-node-badge">{node.signal_count}</span>
					{/if}
				</button>
			{/each}
		</div>
	</aside>

	<section class="ogv-main">
		{#if loading && graphNodes.length === 0}
			<div class="ogv-loading">Loading knowledge graph...</div>
		{:else if graphNodes.length === 0}
			<div class="ogv-empty">No nodes available. Add content to your workspace to see the knowledge graph.</div>
		{:else}
			<svg bind:this={svgEl} class="ogv-svg" viewBox="0 0 {width} {height}">
				<defs>
					<filter id="glow">
						<feGaussianBlur stdDeviation="3" result="blur" />
						<feMerge>
							<feMergeNode in="blur" />
							<feMergeNode in="SourceGraphic" />
						</feMerge>
					</filter>
				</defs>

				{#each graphEdges as edge}
					{@const s = getNodePos(edge.source)}
					{@const t = getNodePos(edge.target)}
					<line
						x1={s.x} y1={s.y} x2={t.x} y2={t.y}
						stroke="rgba(255,255,255,0.08)"
						stroke-width="1"
					/>
				{/each}

				{#each graphNodes as node}
					<g
						class="ogv-graph-node"
						class:selected={selectedSlug === node.slug}
						onclick={() => { selectedSlug = node.slug; onNodeSelect?.({ slug: node.slug, name: node.name }); }}
						role="button"
						tabindex="0"
						aria-label="Node: {node.name}"
					>
						<circle
							cx={node.x} cy={node.y} r={node.radius}
							fill={node.color}
							fill-opacity="0.15"
							stroke={node.color}
							stroke-width={selectedSlug === node.slug ? 2.5 : 1}
							stroke-opacity={selectedSlug === node.slug ? 1 : 0.4}
							filter={selectedSlug === node.slug ? 'url(#glow)' : undefined}
						/>
						<text
							x={node.x} y={node.y}
							text-anchor="middle"
							dominant-baseline="central"
							fill="white"
							font-size={node.radius > 30 ? '11' : '9'}
							font-weight="500"
							pointer-events="none"
						>{node.name}</text>
						{#if node.signalCount > 0}
							<text
								x={node.x} y={node.y + (node.radius > 30 ? 14 : 11)}
								text-anchor="middle"
								fill="rgba(255,255,255,0.5)"
								font-size="8"
								pointer-events="none"
							>{node.signalCount} signals</text>
						{/if}
					</g>
				{/each}
			</svg>
		{/if}
	</section>

	{#if selectedNode}
		<aside class="ogv-detail">
			<div class="ogv-detail__header">
				<h3>{selectedNode.name}</h3>
				<button class="ogv-detail__close" onclick={() => selectedSlug = null} aria-label="Close">x</button>
			</div>
			<div class="ogv-detail__content">
				{#if selectedNode.signal_count > 0}
					<div class="ogv-detail__stat">{selectedNode.signal_count} signals</div>
				{/if}
				{#if selectedNode.context_md}
					<pre class="ogv-detail__context">{selectedNode.context_md}</pre>
				{/if}
			</div>
		</aside>
	{/if}
</div>

<style>
	.ogv-root {
		display: flex;
		height: 100%;
		width: 100%;
		overflow: hidden;
		background: #0d1117;
		color: #e6edf3;
	}

	.ogv-sidebar {
		width: 240px;
		flex-shrink: 0;
		display: flex;
		flex-direction: column;
		border-right: 1px solid rgba(255,255,255,0.06);
		overflow: hidden;
	}

	.ogv-sidebar__header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 14px 16px 10px;
		border-bottom: 1px solid rgba(255,255,255,0.06);
	}

	.ogv-sidebar__title {
		font-size: 11px;
		font-weight: 600;
		letter-spacing: 0.08em;
		color: rgba(255,255,255,0.4);
	}

	.ogv-sidebar__count {
		font-size: 11px;
		color: rgba(255,255,255,0.3);
	}

	.ogv-sidebar__list {
		flex: 1;
		overflow-y: auto;
		padding: 4px 8px;
	}

	.ogv-node-item {
		display: flex;
		align-items: center;
		justify-content: space-between;
		width: 100%;
		padding: 8px 10px;
		border: none;
		background: transparent;
		color: #c9d1d9;
		font-size: 13px;
		border-radius: 6px;
		cursor: pointer;
		text-align: left;
		transition: background 0.15s;
	}

	.ogv-node-item:hover {
		background: rgba(255,255,255,0.05);
	}

	.ogv-node-item.active {
		background: rgba(99,102,241,0.15);
		color: #a5b4fc;
	}

	.ogv-node-badge {
		font-size: 11px;
		color: rgba(255,255,255,0.35);
		background: rgba(255,255,255,0.06);
		padding: 1px 6px;
		border-radius: 10px;
	}

	.ogv-main {
		flex: 1;
		min-width: 0;
		position: relative;
		overflow: hidden;
	}

	.ogv-svg {
		width: 100%;
		height: 100%;
		display: block;
	}

	.ogv-loading, .ogv-empty {
		display: flex;
		align-items: center;
		justify-content: center;
		height: 100%;
		color: rgba(255,255,255,0.3);
		font-size: 14px;
	}

	.ogv-graph-node {
		cursor: pointer;
	}

	.ogv-graph-node:hover circle {
		fill-opacity: 0.25;
		stroke-opacity: 0.8;
	}

	.ogv-detail {
		width: 320px;
		flex-shrink: 0;
		border-left: 1px solid rgba(255,255,255,0.06);
		display: flex;
		flex-direction: column;
		overflow: hidden;
		background: rgba(13,17,23,0.95);
	}

	.ogv-detail__header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 14px 16px;
		border-bottom: 1px solid rgba(255,255,255,0.06);
	}

	.ogv-detail__header h3 {
		margin: 0;
		font-size: 15px;
		font-weight: 600;
	}

	.ogv-detail__close {
		background: none;
		border: none;
		color: rgba(255,255,255,0.4);
		font-size: 16px;
		cursor: pointer;
		padding: 2px 6px;
		border-radius: 4px;
	}

	.ogv-detail__close:hover {
		background: rgba(255,255,255,0.1);
		color: white;
	}

	.ogv-detail__content {
		flex: 1;
		overflow-y: auto;
		padding: 12px 16px;
	}

	.ogv-detail__stat {
		font-size: 12px;
		color: rgba(255,255,255,0.4);
		margin-bottom: 12px;
	}

	.ogv-detail__context {
		font-size: 12px;
		line-height: 1.6;
		color: rgba(255,255,255,0.65);
		white-space: pre-wrap;
		word-break: break-word;
		margin: 0;
		font-family: inherit;
	}
</style>
