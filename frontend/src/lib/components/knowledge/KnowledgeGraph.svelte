<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { Application, Graphics, Container, Text, TextStyle } from 'pixi.js';
	import type { FederatedPointerEvent } from 'pixi.js';
	import { forceSimulation, forceCenter, forceManyBody, forceLink } from 'd3-force';
	import type { SimulationNodeDatum, SimulationLinkDatum } from 'd3-force';
	import { getApiBaseUrl } from '$lib/api/base';
	import type { Memory } from '$lib/api/memory/types';

	// ── Props ─────────────────────────────────────────────────────────────────
	interface Props {
		memories?: Memory[];
		onSelect?: (memory: Memory) => void;
		onDeselect?: () => void;
		selectedId?: string | null;
		searchQuery?: string;
		highlightedIds?: string[];
		nodes?: Map<string, { name: string; type: string }>;
		zoomLevel?: number;
		onZoomChange?: (level: number) => void;
	}
	let {
		memories = [], onSelect, onDeselect,
		selectedId = null,          // eslint-disable-line @typescript-eslint/no-unused-vars
		searchQuery = '',           // eslint-disable-line @typescript-eslint/no-unused-vars
		highlightedIds = [],        // eslint-disable-line @typescript-eslint/no-unused-vars
		nodes: nodeMap = new Map(), // eslint-disable-line @typescript-eslint/no-unused-vars
		zoomLevel = 50,             // eslint-disable-line @typescript-eslint/no-unused-vars
		onZoomChange
	}: Props = $props();

	// ── API types ─────────────────────────────────────────────────────────────
	interface ApiNode   { slug: string; name: string; type: string; signal_count: number; }
	interface ApiFile   { name: string; path: string; is_dir: boolean; size: number; children?: ApiFile[]; }
	interface ApiEntity { name: string; type: string; connections: number; }
	interface ApiEdge   { source: string; target: string; relation: string; weight?: number; }
	type GNodeType = 'core' | 'folder' | 'document' | 'entity';

	interface GNodeRaw {
		id: string; label: string; nodeType: GNodeType;
		connections?: number; entityType?: string; _maxConn?: number;
	}
	interface GNodeDatum extends SimulationNodeDatum, GNodeRaw { gfx: Graphics; radius: number; }
	interface GLinkDatum extends SimulationLinkDatum<GNodeDatum> { relation: string; gfx: Graphics; }

	// ── Design constants ──────────────────────────────────────────────────────
	const BG_COLOR = 0xfafafa, C_CORE = 0x1a1a1a, C_ENTITY = 0x3b82f6;
	const C_FOLDER = 0x6b7280, C_DOC  = 0x9ca3af, C_HOV    = 0x7c3aed;
	const LINK_COLOR = 0x334466, LINK_ALPHA = 0.25, LINK_HOV_ALPHA = 0.7;
	const LINK_FADE_ALPHA = 0.03, NODE_FADE_ALPHA = 0.06;

	function nodeColor(t: GNodeType): number {
		return t === 'core' ? C_CORE : t === 'entity' ? C_ENTITY : t === 'folder' ? C_FOLDER : C_DOC;
	}
	function nodeRadius(n: GNodeRaw): number {
		const base = 2 + ((n.connections ?? 0) / Math.max(n._maxConn ?? 10, 1)) * 5;
		return n.nodeType === 'core' ? base + 1 : base;
	}

	// ── Component state ───────────────────────────────────────────────────────
	let container = $state<HTMLDivElement | null>(null);
	let loading   = $state(true);
	let error     = $state<string | null>(null);
	let statsText = $state('');

	// ── PixiJS + sim handles ──────────────────────────────────────────────────
	let app: Application | null = null;
	let sim: ReturnType<typeof forceSimulation<GNodeDatum>> | null = null;
	let ro:  ResizeObserver | null = null;

	// ── Draw helpers ──────────────────────────────────────────────────────────
	function drawNode(n: GNodeDatum, highlighted: boolean, dimmed: boolean) {
		const color = nodeColor(n.nodeType), r = n.radius;
		const alpha = dimmed ? NODE_FADE_ALPHA : 1;
		const glow  = dimmed ? 0 : highlighted ? 0.45 : 0.18;
		n.gfx.clear();
		n.gfx.circle(0, 0, r + (highlighted ? 6 : 3));
		n.gfx.fill({ color: highlighted ? C_HOV : color, alpha: glow });
		n.gfx.circle(0, 0, r);
		n.gfx.fill({ color: highlighted ? C_HOV : color, alpha });
		n.gfx.circle(-r * 0.28, -r * 0.28, r * 0.32);
		n.gfx.fill({ color: 0xffffff, alpha: dimmed ? 0 : highlighted ? 0.55 : 0.22 });
	}

	function drawLink(l: GLinkDatum, highlighted: boolean, dimmed: boolean) {
		const s = l.source as GNodeDatum, t = l.target as GNodeDatum;
		if (s.x == null || t.x == null) return;
		l.gfx.clear();
		l.gfx.moveTo(s.x!, s.y!);
		l.gfx.lineTo(t.x!, t.y!);
		if (highlighted) {
			l.gfx.stroke({ color: C_HOV, alpha: LINK_HOV_ALPHA, width: 1.5 });
		} else {
			l.gfx.stroke({ color: LINK_COLOR, alpha: dimmed ? LINK_FADE_ALPHA : LINK_ALPHA, width: 1 });
		}
	}

	// ── Core renderer ─────────────────────────────────────────────────────────
	async function buildPixiGraph(
		gNodesRaw: GNodeRaw[],
		gLinksRaw: { source: string; target: string; relation: string }[]
	) {
		if (!container) return;
		destroyGraph();
		container.innerHTML = '';

		const w = container.clientWidth  || 800;
		const h = container.clientHeight || 600;

		app = new Application();
		await app.init({ width: w, height: h, backgroundColor: BG_COLOR, antialias: true,
			resolution: window.devicePixelRatio || 1, autoDensity: true });
		container.appendChild(app.canvas as HTMLCanvasElement);

		const world = new Container();
		app.stage.addChild(world);
		world.x = w / 2; world.y = h / 2;

		const linkLayer = new Container(), nodeLayer = new Container(), labelLayer = new Container();
		world.addChild(linkLayer); world.addChild(nodeLayer); world.addChild(labelLayer);

		// Create link graphics
		const linkData: GLinkDatum[] = gLinksRaw.map(l => {
			const gfx = new Graphics(); linkLayer.addChild(gfx);
			return { source: l.source as unknown as GNodeDatum,
				target: l.target as unknown as GNodeDatum, relation: l.relation, gfx };
		});

		// Create node graphics + labels
		const labelStyle = new TextStyle({ fontFamily: '-apple-system, system-ui, sans-serif', fontSize: 10, fill: '#374151' });
		const coreLabelStyle = new TextStyle({ fontFamily: '-apple-system, system-ui, sans-serif', fontSize: 11, fontWeight: '600', fill: '#111827' });

		const nodeData: GNodeDatum[] = gNodesRaw.map(raw => {
			const gfx = new Graphics();
			gfx.eventMode = 'static'; gfx.cursor = 'pointer';
			nodeLayer.addChild(gfx);

			// Add text label
			const txt = new Text({ text: raw.label ?? raw.id, style: raw.nodeType === 'core' ? coreLabelStyle : labelStyle });
			txt.anchor.set(0, 0.5);
			txt.visible = false; // Labels only on hover or zoom
			labelLayer.addChild(txt);

			const datum: GNodeDatum = { ...raw, radius: nodeRadius(raw), gfx, _label: txt } as GNodeDatum & { _label: Text };
			drawNode(datum, false, false);
			return datum;
		});

		// Adjacency maps (populated after sim resolves references)
		const neighbors = new Map<string, Set<string>>();
		const nodeLinks = new Map<string, Set<GLinkDatum>>();
		for (const n of nodeData) { neighbors.set(n.id, new Set()); nodeLinks.set(n.id, new Set()); }

		// D3-force simulation — exact physics from demo
		// Depth helper — deeper nodes get tighter links
		const nodeDepth = new Map<string, number>();
		for (const n of nodeData) {
			nodeDepth.set(n.id, (n.id.match(/\//g) || []).length); // count path segments
		}

		sim = forceSimulation<GNodeDatum>(nodeData)
			.force('center', forceCenter(0, 0))
			.force('charge', forceManyBody<GNodeDatum>().strength(-120).distanceMax(400))
			.force('link', forceLink<GNodeDatum, GLinkDatum>(linkData).id(d => d.id)
				.distance((l: GLinkDatum) => {
					const rel = l.relation;
					if (rel === 'contains') {
						// Hierarchy depth: core→domain=60, domain→project=40, project→doc=20
						const tgt = typeof l.target === 'object' ? l.target : null;
						const depth = tgt ? (nodeDepth.get(tgt.id) ?? 1) : 1;
						if (depth <= 1) return 60;   // core → domain folder (platform, agency)
						if (depth === 2) return 40;  // domain → project/subfolder
						return 20;                    // deep docs — very tight
					}
					if (rel === 'sibling')     return 250;  // core→core: WIDE
					if (rel === 'belongs_to')  return 100;  // entity→core
					if (rel === 'cross_ref')   return 150;  // cross-references: bridges between clusters
					return 60;
				})
				.strength((l: GLinkDatum) => {
					const rel = l.relation;
					if (rel === 'contains')    return 0.7;  // strong pull keeps clusters tight
					if (rel === 'sibling')     return 0.03; // very weak — just hints
					if (rel === 'belongs_to')  return 0.2;
					if (rel === 'cross_ref')   return 0.08;
					return 0.4;
				})
			)
			.alphaDecay(0.02).velocityDecay(0.4);

		// Build adjacency after d3 resolves source/target references
		setTimeout(() => {
			for (const l of linkData) {
				const s = (l.source as GNodeDatum).id, t = (l.target as GNodeDatum).id;
				if (s && t) {
					neighbors.get(s)?.add(t); neighbors.get(t)?.add(s);
					nodeLinks.get(s)?.add(l); nodeLinks.get(t)?.add(l);
				}
			}
		}, 0);

		// Hover state (plain JS — runs at 60fps, no reactivity needed)
		let hoveredNode: GNodeDatum | null = null;

		function applyHover(node: GNodeDatum | null) {
			if (hoveredNode === node) return;
			hoveredNode = node;
			if (!node) {
				for (const n of nodeData) drawNode(n, false, false);
				for (const l of linkData) drawLink(l, false, false);
				return;
			}
			const cn = new Set([node.id, ...(neighbors.get(node.id) ?? [])]);
			const cl = nodeLinks.get(node.id) ?? new Set<GLinkDatum>();
			for (const n of nodeData) drawNode(n, n.id === node.id, !cn.has(n.id));
			for (const l of linkData) drawLink(l, cl.has(l), !cl.has(l));
		}

		// Simulation tick — update positions, labels, redraw links
		sim.on('tick', () => {
			const z = scale;
			for (const n of nodeData) {
				if (n.x != null) {
					n.gfx.x = n.x!; n.gfx.y = n.y!;
					// Position label next to node
					const lbl = (n as any)._label as Text | undefined;
					if (lbl) {
						lbl.x = n.x! + n.radius + 3;
						lbl.y = n.y!;
						// Show labels: core always, entity at zoom>1.5, all at zoom>3, hovered always
						const isHov = n.id === hoveredNode?.id;
						const isCon = hoveredNode && neighbors.get(hoveredNode.id)?.has(n.id);
						lbl.visible = isHov || !!isCon || n.nodeType === 'core' || (n.nodeType === 'entity' && z > 1.5) || z > 3;
						lbl.alpha = isHov ? 1 : isCon ? 0.8 : 0.6;
						lbl.scale.set(1 / z); // Keep label size constant regardless of zoom
					}
				}
			}
			const hn = hoveredNode;
			if (hn) {
				const cl = nodeLinks.get(hn.id)!;
				for (const l of linkData) drawLink(l, cl.has(l), !cl.has(l));
			} else {
				for (const l of linkData) drawLink(l, false, false);
			}
		});

		// Node drag + click
		let dragNode: GNodeDatum | null = null;
		for (const node of nodeData) {
			node.gfx.on('pointerover', () => applyHover(node));
			node.gfx.on('pointerout',  () => { if (!dragNode) applyHover(null); });
			node.gfx.on('pointerdown', (e: FederatedPointerEvent) => {
				e.stopPropagation();
				dragNode = node; node.fx = node.x; node.fy = node.y;
				sim!.alphaTarget(0.3).restart();
			});
			node.gfx.on('pointerup', () => {
				if (dragNode === node) {
					const mem = buildMemoryFromNode(node);
					if (mem) onSelect?.(mem);
				}
			});
		}

		// Stage pan
		let isPanning = false, panStart = { x: 0, y: 0 }, worldStart = { x: 0, y: 0 }, scale = 1;
		app.stage.eventMode = 'static'; app.stage.hitArea = app.screen;

		app.stage.on('pointerdown', (e: FederatedPointerEvent) => {
			if (e.target === app!.stage) {
				isPanning = true; panStart = { x: e.globalX, y: e.globalY };
				worldStart = { x: world.x, y: world.y };
			}
		});
		app.stage.on('pointermove', (e: FederatedPointerEvent) => {
			if (isPanning) { world.x = worldStart.x + (e.globalX - panStart.x); world.y = worldStart.y + (e.globalY - panStart.y); }
			if (dragNode) { const local = world.toLocal(e.global); dragNode.fx = local.x; dragNode.fy = local.y; }
		});
		app.stage.on('pointerup', (e: FederatedPointerEvent) => {
			isPanning = false;
			if (dragNode) { dragNode.fx = null; dragNode.fy = null; sim!.alphaTarget(0); dragNode = null; applyHover(null); }
			if (e.target === app!.stage) onDeselect?.();
		});
		app.stage.on('pointerupoutside', () => { isPanning = false; });

		// Wheel zoom toward cursor
		(app.canvas as HTMLCanvasElement).addEventListener('wheel', (e: WheelEvent) => {
			e.preventDefault();
			const factor   = e.deltaY < 0 ? 1.1 : 0.91;
			const newScale = Math.min(Math.max(scale * factor, 0.1), 8);
			const wx = (e.offsetX - world.x) / scale, wy = (e.offsetY - world.y) / scale;
			scale = newScale; world.scale.set(scale);
			world.x = e.offsetX - wx * scale; world.y = e.offsetY - wy * scale;
			onZoomChange?.(Math.max(0, Math.min(100, Math.round(((scale - 0.1) / 7.9) * 100))));
		}, { passive: false });

		// Resize
		ro = new ResizeObserver(() => {
			if (!app || !container) return;
			app.renderer.resize(container.clientWidth, container.clientHeight);
			app.stage.hitArea = app.screen;
		});
		ro.observe(container);
	}

	// ── Map node back to Memory for callbacks ─────────────────────────────────
	function buildMemoryFromNode(n: GNodeDatum): Memory | null {
		const found = memories.find(m => m.id === n.id);
		if (found) return found;
		const docId = n.nodeType === 'core' ? `${n.id}/context.md` : n.id;
		return {
			id: docId, user_id: 'graph', title: n.label ?? n.id,
			summary: `${n.nodeType} — ${n.connections ?? 0} connections`, content: '',
			memory_type: 'context', importance_score: (n.connections ?? 0) / 200,
			is_pinned: false, is_active: true, tags: [],
			metadata: { entity_type: n.entityType ?? n.nodeType },
			source_type: null, source_id: null, project_id: null, node_id: null,
			expires_at: null, access_count: 0, last_accessed_at: null, created_at: '', updated_at: ''
		};
	}

	// ── File tree walker ──────────────────────────────────────────────────────
	function walkTree(
		files: ApiFile[], parentId: string,
		gNodes: GNodeRaw[], gLinks: { source: string; target: string; relation: string }[]
	) {
		for (const f of files) {
			gNodes.push({ id: f.path, label: f.name, nodeType: f.is_dir ? 'folder' : 'document' });
			gLinks.push({ source: parentId, target: f.path, relation: 'contains' });
			if (f.is_dir && f.children?.length) walkTree(f.children, f.path, gNodes, gLinks);
		}
	}

	// ── Main API fetch + render ───────────────────────────────────────────────
	async function initGraph() {
		if (!container) return;
		loading = true; error = null;
		try {
			const base = getApiBaseUrl();
			const [nodesRes, graphRes] = await Promise.all([
				fetch(`${base}/optimal/nodes`), fetch(`${base}/optimal/graph`)
			]);
			if (!nodesRes.ok) throw new Error(`/optimal/nodes → HTTP ${nodesRes.status}`);
			if (!graphRes.ok) throw new Error(`/optimal/graph → HTTP ${graphRes.status}`);

			const nodesData = await nodesRes.json();
			const coreNodes: ApiNode[] = Array.isArray(nodesData) ? nodesData : (nodesData.nodes ?? []);
			const graphRaw  = await graphRes.json();
			const graphData: { entities: ApiEntity[]; edges: ApiEdge[] } = {
				entities: graphRaw.entities ?? [], edges: graphRaw.edges ?? []
			};

			const fileTreeResults = await Promise.allSettled(
				coreNodes.map(n =>
					fetch(`${base}/optimal/nodes/${n.slug}/files`)
						.then(r => r.ok ? r.json() : Promise.resolve({ files: [] }))
						.then((d: { files: ApiFile[] }) => ({ slug: n.slug, files: d.files ?? [] }))
				)
			);

			const gNodes: GNodeRaw[] = [];
			const gLinks: { source: string; target: string; relation: string }[] = [];
			const slugSet = new Set(coreNodes.map(n => n.slug));

			for (const n of coreNodes) gNodes.push({ id: n.slug, label: n.name, nodeType: 'core' });
			for (const res of fileTreeResults)
				if (res.status === 'fulfilled') walkTree(res.value.files, res.value.slug, gNodes, gLinks);

			const entityIds = new Set<string>();
			let maxConnections = 1;
			for (const e of graphData.entities) {
				if (!entityIds.has(e.name)) {
					entityIds.add(e.name);
					if (e.connections > maxConnections) maxConnections = e.connections;
					gNodes.push({ id: e.name, label: e.name, nodeType: 'entity', connections: e.connections, entityType: e.type });
				}
			}

			// NO sibling links — let cores spread via repulsion, each gets its own space

			// Entity→core: only keep ONE strongest connection per entity (prevents center collapse)
			const entityBestCore = new Map<string, string>();
			for (const e of graphData.edges) {
				if (entityIds.has(e.source) && slugSet.has(e.target)) {
					if (!entityBestCore.has(e.source)) entityBestCore.set(e.source, e.target);
				}
				if (entityIds.has(e.target) && slugSet.has(e.source)) {
					if (!entityBestCore.has(e.target)) entityBestCore.set(e.target, e.source);
				}
			}
			for (const [entity, core] of entityBestCore) {
				gLinks.push({ source: entity, target: core, relation: 'belongs_to' });
			}
			const seenLinks = new Set<string>();
			function addLink(s: string, t: string, rel: string) {
				const key = `${s}||${t}||${rel}`;
				if (!seenLinks.has(key)) { seenLinks.add(key); gLinks.push({ source: s, target: t, relation: rel }); }
			}
			for (const e of graphData.edges) {
				const sc = slugSet.has(e.source), tc = slugSet.has(e.target);
				const se = entityIds.has(e.source), te = entityIds.has(e.target);
				if (e.relation === 'cross_ref') {
					if (sc && tc) addLink(e.source, e.target, 'cross_ref');
					if (se && te) addLink(e.source, e.target, 'cross_ref');
					if (se && tc) addLink(e.source, e.target, 'mentioned_in');
					if (te && sc) addLink(e.target, e.source, 'mentioned_in');
				} else if (e.relation === 'mentioned_in' || e.relation === 'lives_in') {
					if (se && tc) addLink(e.source, e.target, 'mentioned_in');
					if (te && sc) addLink(e.target, e.source, 'mentioned_in');
				}
			}

			// Remove isolated nodes
			const connectedIds = new Set<string>();
			for (const lk of gLinks) { connectedIds.add(lk.source); connectedIds.add(lk.target); }
			const filteredNodes = gNodes.filter(n => connectedIds.has(n.id));
			for (const n of filteredNodes) n._maxConn = maxConnections;

			statsText = `${filteredNodes.length} nodes · ${gLinks.length} edges`;
			await buildPixiGraph(filteredNodes, gLinks);
			loading = false;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load graph';
			loading = false;
		}
	}

	// ── Legacy mode: build from memories prop ────────────────────────────────
	async function buildFromMemories() {
		if (!container) return;
		loading = true;
		const maxConn = Math.max(...memories.map(m => Math.round((m.importance_score ?? 0.5) * 100)), 1);
		const gNodes: GNodeRaw[] = memories.map(m => ({
			id: m.id, label: m.title ?? m.id, nodeType: 'document' as GNodeType,
			connections: Math.round((m.importance_score ?? 0.5) * 100), _maxConn: maxConn
		}));
		const gLinks: { source: string; target: string; relation: string }[] = [];
		for (let i = 0; i < memories.length; i++)
			for (let j = i + 1; j < memories.length; j++)
				if (memories[i].node_id && memories[i].node_id === memories[j].node_id)
					gLinks.push({ source: memories[i].id, target: memories[j].id, relation: 'co_node' });
		statsText = `${memories.length} memories · ${gLinks.length} edges`;
		await buildPixiGraph(gNodes, gLinks);
		loading = false;
	}

	// ── Cleanup ───────────────────────────────────────────────────────────────
	function destroyGraph() {
		ro?.disconnect(); ro = null;
		sim?.stop(); sim = null;
		if (app) { try { app.destroy(true, { children: true }); } catch { /* ignore */ } app = null; }
	}

	// ── Lifecycle ─────────────────────────────────────────────────────────────
	onMount(() => { memories.length === 0 ? initGraph() : buildFromMemories(); });
	onDestroy(() => destroyGraph());
</script>

<div class="kg-root" bind:this={container}></div>

{#if statsText && !loading}
	<div class="kg-stats" aria-label="Graph statistics">
		<span class="kg-stat-item">{statsText}</span>
		<span class="kg-stat-sep">·</span>
		<span class="kg-stat-legend">
			<span class="kg-dot" style="background:#1a1a1a"></span>core
			<span class="kg-dot" style="background:#3b82f6"></span>entity
			<span class="kg-dot" style="background:#6b7280"></span>folder
			<span class="kg-dot" style="background:#9ca3af"></span>doc
		</span>
	</div>
{/if}

{#if loading && !error}
	<div class="kg-overlay">
		<div class="kg-spinner" aria-label="Loading graph"></div>
		<span class="kg-overlay-label">Building full hierarchy…</span>
	</div>
{/if}

{#if error}
	<div class="kg-overlay">
		<span class="kg-error-label">{error}</span>
		<button class="kg-retry" onclick={() => initGraph()}>Retry</button>
	</div>
{/if}

<style>
	.kg-root {
		position: relative; width: 100%; height: 100%;
		min-height: 500px; background: #fafafa; overflow: hidden;
	}
	.kg-stats {
		position: absolute; bottom: 16px; left: 16px;
		display: flex; align-items: center; gap: 8px;
		padding: 6px 12px; background: rgba(255,255,255,0.9); backdrop-filter: blur(8px);
		border: 1px solid rgba(0,0,0,0.08); border-radius: 8px;
		font-size: 11px; color: rgba(0,0,0,0.5); pointer-events: none; z-index: 10;
	}
	.kg-stat-item { white-space: nowrap; }
	.kg-stat-sep  { color: rgba(0,0,0,0.3); }
	.kg-stat-legend { display: flex; align-items: center; gap: 5px; font-size: 10px; color: rgba(0,0,0,0.4); }
	.kg-dot { display: inline-block; width: 7px; height: 7px; border-radius: 50%; margin-left: 4px; }
	.kg-overlay {
		position: absolute; inset: 0;
		display: flex; flex-direction: column; align-items: center; justify-content: center;
		gap: 12px; background: #fafafa; z-index: 20; pointer-events: none;
	}
	.kg-overlay-label { font-size: 13px; color: rgba(0,0,0,0.4); letter-spacing: 0.03em; }
	.kg-error-label   { font-size: 13px; color: rgba(180,30,30,0.7); letter-spacing: 0.03em; }
	.kg-spinner {
		width: 24px; height: 24px;
		border: 2px solid rgba(0,0,0,0.08); border-top-color: #7c3aed;
		border-radius: 50%; animation: kg-spin 0.8s linear infinite;
	}
	@keyframes kg-spin { to { transform: rotate(360deg); } }
	.kg-retry {
		pointer-events: all; padding: 6px 16px; font-size: 12px;
		border: 1px solid rgba(0,0,0,0.15); border-radius: 6px;
		background: white; color: #1a1a1a; cursor: pointer; transition: background 0.15s;
	}
	.kg-retry:hover { background: rgba(124,58,237,0.06); border-color: rgba(124,58,237,0.3); }
</style>
