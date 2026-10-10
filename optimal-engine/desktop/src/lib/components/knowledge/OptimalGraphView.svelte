<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import * as THREE from 'three';
	import { OrbitControls } from 'three/examples/jsm/controls/OrbitControls.js';
	import { getApiBaseUrl, getCSRFToken } from '$lib/api/base';

	// ── API types ────────────────────────────────────────────────────────────────
	interface OptimalEntity {
		name: string;
		type: string;
		connections: number;
	}

	interface OptimalEdge {
		source: string;
		target: string;
		relation: string;
		weight: number;
	}

	interface OptimalGraphData {
		entities: OptimalEntity[];
		edges: OptimalEdge[];
		stats: {
			entity_count: number;
			edge_count: number;
			edge_types: Record<string, number>;
		};
	}

	// ── Props ────────────────────────────────────────────────────────────────────
	interface Props {
		onSelectEntity?: (name: string, type: string) => void;
	}

	let { onSelectEntity }: Props = $props();

	// ── State ────────────────────────────────────────────────────────────────────
	let graphData = $state<OptimalGraphData | null>(null);
	let loading = $state(true);
	let fetchError = $state<string | null>(null);
	let showStats = $state(true);

	// ── Three.js refs ─────────────────────────────────────────────────────────────
	let container: HTMLDivElement;
	let scene: THREE.Scene;
	let camera: THREE.PerspectiveCamera;
	let renderer: THREE.WebGLRenderer;
	let controls: OrbitControls;
	let animationId: number;
	let raycaster: THREE.Raycaster;
	let mouse: THREE.Vector2;

	let nodeMeshes = new Map<string, THREE.Mesh>();
	let nodePositions = new Map<string, THREE.Vector3>();
	let sceneEdges = $state<THREE.Line[]>([]);
	let hoveredName = $state<string | null>(null);
	let tooltipPos = $state({ x: 0, y: 0 });
	let tooltipEntity = $state<OptimalEntity | null>(null);
	let labelPositions = $state<
		Map<string, { x: number; y: number; visible: boolean; name: string; type: string; connections: number }>
	>(new Map());

	// ── Color maps ────────────────────────────────────────────────────────────────
	// Entity type → hex color (spec: person=blue, org=purple, product=green, concept=yellow, operation=orange)
	const entityColors: Record<string, number> = {
		person:    0x3b82f6, // blue
		org:       0x8b5cf6, // purple
		product:   0x10b981, // green
		concept:   0xeab308, // yellow
		operation: 0xf97316, // orange
		default:   0x6b7280  // gray
	};

	// Relation type → hex edge color
	const relationColors: Record<string, number> = {
		manages:     0x3b82f6,
		works_with:  0x10b981,
		owns:        0x8b5cf6,
		part_of:     0xf97316,
		related_to:  0x6b7280,
		leads:       0xec4899,
		default:     0x374151
	};

	function entityColor(type: string): number {
		return entityColors[type?.toLowerCase()] ?? entityColors.default;
	}

	function relationColor(relation: string): number {
		return relationColors[relation?.toLowerCase()] ?? relationColors.default;
	}

	// ── Force simulation ──────────────────────────────────────────────────────────
	interface SimNode {
		key: string;
		x: number; y: number; z: number;
		vx: number; vy: number; vz: number;
		connections: number;
	}

	function runForceSimulation(simNodes: SimNode[], edges: OptimalEdge[], iterations = 120) {
		const repulsion = 600;
		const attraction = 0.06;
		const damping = 0.88;
		const center = 0.008;

		// Build adjacency for attraction
		const adj = new Map<string, string[]>();
		edges.forEach(e => {
			if (!adj.has(e.source)) adj.set(e.source, []);
			if (!adj.has(e.target)) adj.set(e.target, []);
			adj.get(e.source)!.push(e.target);
			adj.get(e.target)!.push(e.source);
		});

		for (let iter = 0; iter < iterations; iter++) {
			// Repulsion between all pairs
			for (let j = 0; j < simNodes.length; j++) {
				for (let k = j + 1; k < simNodes.length; k++) {
					const a = simNodes[j], b = simNodes[k];
					const dx = b.x - a.x, dy = b.y - a.y, dz = b.z - a.z;
					const dist = Math.sqrt(dx * dx + dy * dy + dz * dz) || 1;
					const f = repulsion / (dist * dist);
					const fx = (dx / dist) * f, fy = (dy / dist) * f, fz = (dz / dist) * f;
					a.vx -= fx; a.vy -= fy; a.vz -= fz;
					b.vx += fx; b.vy += fy; b.vz += fz;
				}
			}

			// Attraction along edges
			simNodes.forEach(node => {
				const neighbors = adj.get(node.key) ?? [];
				neighbors.forEach(nbrKey => {
					const nbr = simNodes.find(n => n.key === nbrKey);
					if (!nbr) return;
					const dx = nbr.x - node.x, dy = nbr.y - node.y, dz = nbr.z - node.z;
					node.vx += dx * attraction;
					node.vy += dy * attraction;
					node.vz += dz * attraction;
				});
			});

			// Center gravity
			simNodes.forEach(node => {
				node.vx -= node.x * center;
				node.vy -= node.y * center;
				node.vz -= node.z * center;
			});

			// Integrate
			simNodes.forEach(node => {
				node.x += node.vx; node.y += node.vy; node.z += node.vz;
				node.vx *= damping; node.vy *= damping; node.vz *= damping;
			});
		}
		return simNodes;
	}

	// ── Data fetch ────────────────────────────────────────────────────────────────
	async function fetchGraph() {
		loading = true;
		fetchError = null;
		try {
			const base = getApiBaseUrl();
			const csrf = getCSRFToken();
			const headers: Record<string, string> = { 'Content-Type': 'application/json' };
			if (csrf) headers['X-CSRF-Token'] = csrf;

			const res = await fetch(`${base}/optimal/graph`, {
				credentials: 'include',
				headers
			});

			if (!res.ok) throw new Error(`HTTP ${res.status}`);
			graphData = await res.json() as OptimalGraphData;
		} catch (e) {
			fetchError = e instanceof Error ? e.message : 'Failed to load graph';
		} finally {
			loading = false;
		}
	}

	// ── Three.js ──────────────────────────────────────────────────────────────────
	function disposeObj(obj: THREE.Object3D) {
		if (obj instanceof THREE.Mesh || obj instanceof THREE.Line) {
			if (obj.geometry) obj.geometry.dispose();
			const mats = Array.isArray(obj.material) ? obj.material : [obj.material];
			mats.forEach(m => m?.dispose());
		}
	}

	function clearGraph() {
		nodeMeshes.forEach(mesh => { mesh.traverse(disposeObj); scene.remove(mesh); });
		sceneEdges.forEach(line => { disposeObj(line); scene.remove(line); });
		nodeMeshes.clear();
		nodePositions.clear();
		sceneEdges = [];
	}

	function buildGraph() {
		clearGraph();
		if (!graphData || graphData.entities.length === 0) return;

		const { entities, edges } = graphData;
		const maxConns = Math.max(...entities.map(e => e.connections), 1);

		// Init sim nodes
		const simNodes: SimNode[] = entities.map(e => ({
			key: e.name,
			x: (Math.random() - 0.5) * 120,
			y: (Math.random() - 0.5) * 120,
			z: (Math.random() - 0.5) * 120,
			vx: 0, vy: 0, vz: 0,
			connections: e.connections
		}));

		runForceSimulation(simNodes, edges);

		// Create meshes
		simNodes.forEach(sim => {
			const entity = entities.find(e => e.name === sim.key);
			if (!entity) return;

			// Node size: 2.5–6 based on connection count
			const radius = 2.5 + (entity.connections / maxConns) * 3.5;
			const color = entityColor(entity.type);

			const geo = new THREE.SphereGeometry(radius, 32, 32);
			const mat = new THREE.MeshPhongMaterial({
				color,
				emissive: color,
				emissiveIntensity: 0.25,
				shininess: 80
			});
			const mesh = new THREE.Mesh(geo, mat);
			mesh.position.set(sim.x, sim.y, sim.z);
			mesh.userData = { name: entity.name, entity };

			// Glow shell
			const glowGeo = new THREE.SphereGeometry(radius + 1.2, 32, 32);
			const glowMat = new THREE.MeshBasicMaterial({ color, transparent: true, opacity: 0.12 });
			mesh.add(new THREE.Mesh(glowGeo, glowMat));

			scene.add(mesh);
			nodeMeshes.set(entity.name, mesh);
			nodePositions.set(entity.name, new THREE.Vector3(sim.x, sim.y, sim.z));
		});

		// Draw edges
		const newEdges: THREE.Line[] = [];
		edges.forEach(edge => {
			const src = nodePositions.get(edge.source);
			const tgt = nodePositions.get(edge.target);
			if (!src || !tgt) return;

			const geo = new THREE.BufferGeometry().setFromPoints([src, tgt]);
			const mat = new THREE.LineBasicMaterial({
				color: relationColor(edge.relation),
				transparent: true,
				opacity: 0.35 + Math.min(edge.weight ?? 0, 1) * 0.3
			});
			const line = new THREE.Line(geo, mat);
			scene.add(line);
			newEdges.push(line);
		});
		sceneEdges = newEdges;
	}

	function initScene() {
		if (!container) return;

		scene = new THREE.Scene();
		scene.background = new THREE.Color(0x0a0a0a);

		const aspect = container.clientWidth / container.clientHeight;
		camera = new THREE.PerspectiveCamera(60, aspect, 0.1, 1000);
		camera.position.set(0, 0, 160);

		renderer = new THREE.WebGLRenderer({ antialias: true });
		renderer.setSize(container.clientWidth, container.clientHeight);
		renderer.setPixelRatio(window.devicePixelRatio);
		container.appendChild(renderer.domElement);

		controls = new OrbitControls(camera, renderer.domElement);
		controls.enableDamping = true;
		controls.dampingFactor = 0.05;
		controls.minDistance = 40;
		controls.maxDistance = 350;

		raycaster = new THREE.Raycaster();
		mouse = new THREE.Vector2();

		scene.add(new THREE.AmbientLight(0xffffff, 0.5));
		const pt = new THREE.PointLight(0xffffff, 1, 600);
		pt.position.set(60, 60, 60);
		scene.add(pt);

		// Subtle grid
		const grid = new THREE.GridHelper(300, 25, 0x1a1a1a, 0x141414);
		grid.rotation.x = Math.PI / 2;
		scene.add(grid);

		renderer.domElement.addEventListener('mousemove', onMouseMove);
		renderer.domElement.addEventListener('click', onClick);
		window.addEventListener('resize', onResize);

		buildGraph();
		animate();
	}

	function updateLabelPositions() {
		if (!container || !camera) return;
		const map = new Map<string, { x: number; y: number; visible: boolean; name: string; type: string; connections: number }>();
		const tmp = new THREE.Vector3();

		nodeMeshes.forEach((mesh, name) => {
			const entity = mesh.userData.entity as OptimalEntity;
			mesh.getWorldPosition(tmp);
			tmp.project(camera);
			const x = (tmp.x * 0.5 + 0.5) * container.clientWidth;
			const y = (-tmp.y * 0.5 + 0.5) * container.clientHeight;
			map.set(name, { x, y, visible: tmp.z < 1, name, type: entity.type, connections: entity.connections });
		});
		labelPositions = map;
	}

	function animate() {
		animationId = requestAnimationFrame(animate);
		controls?.update();
		renderer?.render(scene, camera);
		updateLabelPositions();
	}

	function onMouseMove(e: MouseEvent) {
		if (!container || !renderer) return;
		const rect = renderer.domElement.getBoundingClientRect();
		mouse.x = ((e.clientX - rect.left) / rect.width) * 2 - 1;
		mouse.y = -((e.clientY - rect.top) / rect.height) * 2 + 1;
		tooltipPos = { x: e.clientX, y: e.clientY };

		raycaster.setFromCamera(mouse, camera);
		const hits = raycaster.intersectObjects(Array.from(nodeMeshes.values()));
		if (hits.length > 0) {
			const mesh = hits[0].object as THREE.Mesh;
			const name = mesh.userData.name as string;
			if (name !== hoveredName) {
				hoveredName = name;
				tooltipEntity = mesh.userData.entity as OptimalEntity;
				renderer.domElement.style.cursor = 'pointer';
				// Highlight hover
				const mat = mesh.material as THREE.MeshPhongMaterial;
				mat.emissiveIntensity = 0.7;
			}
		} else {
			if (hoveredName) {
				const prev = nodeMeshes.get(hoveredName);
				if (prev) (prev.material as THREE.MeshPhongMaterial).emissiveIntensity = 0.25;
			}
			hoveredName = null;
			tooltipEntity = null;
			renderer.domElement.style.cursor = 'grab';
		}
	}

	function onClick(e: MouseEvent) {
		if (!container || !renderer) return;
		const rect = renderer.domElement.getBoundingClientRect();
		mouse.x = ((e.clientX - rect.left) / rect.width) * 2 - 1;
		mouse.y = -((e.clientY - rect.top) / rect.height) * 2 + 1;
		raycaster.setFromCamera(mouse, camera);
		const hits = raycaster.intersectObjects(Array.from(nodeMeshes.values()));
		if (hits.length > 0) {
			const entity = hits[0].object.userData.entity as OptimalEntity;
			onSelectEntity?.(entity.name, entity.type);
		}
	}

	function onResize() {
		if (!container || !camera || !renderer) return;
		camera.aspect = container.clientWidth / container.clientHeight;
		camera.updateProjectionMatrix();
		renderer.setSize(container.clientWidth, container.clientHeight);
	}

	// Rebuild graph when data arrives
	$effect(() => {
		if (scene && graphData) buildGraph();
	});

	onMount(async () => {
		await fetchGraph();
		initScene();
	});

	onDestroy(() => {
		cancelAnimationFrame(animationId);
		clearGraph();
		controls?.dispose();
		if (renderer) {
			renderer.domElement.removeEventListener('mousemove', onMouseMove);
			renderer.domElement.removeEventListener('click', onClick);
			renderer.dispose();
			renderer.forceContextLoss();
		}
		window.removeEventListener('resize', onResize);
	});

	function colorHex(n: number) {
		return '#' + n.toString(16).padStart(6, '0');
	}

	function capitalise(s: string) {
		return s ? s.charAt(0).toUpperCase() + s.slice(1).toLowerCase() : '';
	}

	// Top edge types for the stats panel
	let topEdgeTypes = $derived(
		graphData
			? Object.entries(graphData.stats.edge_types)
					.sort((a, b) => b[1] - a[1])
					.slice(0, 6)
			: []
	);
</script>

<div class="ogv-root">
	<!-- ── Loading ── -->
	{#if loading}
		<div class="ogv-center">
			<div class="ogv-spinner"></div>
			<p class="ogv-hint">Loading OptimalOS graph…</p>
		</div>
	{:else if fetchError}
		<div class="ogv-center">
			<p class="ogv-error">{fetchError}</p>
			<button class="ogv-btn" onclick={fetchGraph}>Retry</button>
		</div>
	{:else}
		<!-- ── 3D canvas ── -->
		<div bind:this={container} class="ogv-canvas"></div>

		<!-- ── Node labels ── -->
		<div class="ogv-labels">
			{#each [...labelPositions] as [name, lbl] (name)}
				{#if lbl.visible}
					<div
						class="ogv-label"
						class:ogv-label--hover={hoveredName === name}
						style="left:{lbl.x}px; top:{lbl.y + 13}px"
					>
						{lbl.name}
					</div>
				{/if}
			{/each}
		</div>

		<!-- ── Tooltip ── -->
		{#if tooltipEntity && hoveredName}
			<div class="ogv-tooltip" style="left:{tooltipPos.x + 14}px; top:{tooltipPos.y + 14}px">
				<div class="ogv-tooltip__header">
					<span
						class="ogv-tooltip__dot"
						style="background:{colorHex(entityColor(tooltipEntity.type))}"
					></span>
					<span class="ogv-tooltip__type">{capitalise(tooltipEntity.type)}</span>
				</div>
				<div class="ogv-tooltip__name">{tooltipEntity.name}</div>
				<div class="ogv-tooltip__meta">{tooltipEntity.connections} connection{tooltipEntity.connections !== 1 ? 's' : ''}</div>
				<div class="ogv-tooltip__hint">
					<kbd>Click</kbd> select
				</div>
			</div>
		{/if}

		<!-- ── Stats panel ── -->
		{#if showStats && graphData}
			<div class="ogv-stats">
				<div class="ogv-stats__title">
					Optimal Graph
					<button class="ogv-stats__close" onclick={() => showStats = false} aria-label="Hide stats">×</button>
				</div>
				<div class="ogv-stats__row">
					<span class="ogv-stats__num">{graphData.stats.entity_count}</span> entities
				</div>
				<div class="ogv-stats__row">
					<span class="ogv-stats__num">{graphData.stats.edge_count}</span> edges
				</div>
				{#if topEdgeTypes.length > 0}
					<div class="ogv-stats__sep"></div>
					<div class="ogv-stats__label">Edge types</div>
					{#each topEdgeTypes as [rel, count]}
						<div class="ogv-stats__edge-row">
							<span class="ogv-stats__edge-dot" style="background:{colorHex(relationColor(rel))}"></span>
							<span class="ogv-stats__edge-name">{rel}</span>
							<span class="ogv-stats__edge-count">{count}</span>
						</div>
					{/each}
				{/if}
			</div>
		{:else if !showStats}
			<button class="ogv-stats-toggle" onclick={() => showStats = true} title="Show stats">
				<svg width="14" height="14" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"/>
				</svg>
			</button>
		{/if}

		<!-- ── Entity type legend ── -->
		<div class="ogv-legend">
			<div class="ogv-legend__title">Entity types</div>
			{#each Object.entries(entityColors).filter(([k]) => k !== 'default') as [type, color]}
				<div class="ogv-legend__row">
					<span class="ogv-legend__dot" style="background:{colorHex(color)}"></span>
					<span>{capitalise(type)}</span>
				</div>
			{/each}
		</div>

		<!-- ── Camera controls ── -->
		<div class="ogv-controls">
			<button
				onclick={() => { if (camera && controls) { camera.position.set(0, 0, 160); controls.reset(); }}}
				class="ogv-ctrl-btn" title="Reset view"
			>
				<svg width="14" height="14" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
				</svg>
			</button>
			<button
				onclick={() => { if (camera) camera.position.z = Math.max(40, camera.position.z - 25); }}
				class="ogv-ctrl-btn" title="Zoom in"
			>
				<svg width="14" height="14" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0zM10 7v3m0 0v3m0-3h3m-3 0H7"/>
				</svg>
			</button>
			<button
				onclick={() => { if (camera) camera.position.z = Math.min(350, camera.position.z + 25); }}
				class="ogv-ctrl-btn" title="Zoom out"
			>
				<svg width="14" height="14" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0zM13 10H7"/>
				</svg>
			</button>
			<button
				onclick={fetchGraph}
				class="ogv-ctrl-btn" title="Refresh data"
			>
				<svg width="14" height="14" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
				</svg>
			</button>
		</div>

		<!-- ── Controls hint ── -->
		<div class="ogv-hint-bar">
			<kbd>Drag</kbd> rotate
			<kbd>Scroll</kbd> zoom
			<kbd>Right-drag</kbd> pan
		</div>

		<!-- ── Empty state ── -->
		{#if graphData && graphData.entities.length === 0}
			<div class="ogv-center">
				<div class="ogv-empty-icon">
					<svg width="40" height="40" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<circle cx="12" cy="12" r="3"/><path d="M12 3v3m0 12v3M3 12h3m12 0h3"/>
					</svg>
				</div>
				<p class="ogv-hint">No entities in the graph yet</p>
			</div>
		{/if}
	{/if}
</div>

<style>
	.ogv-root {
		position: relative;
		width: 100%;
		height: 100%;
		min-height: 500px;
		background: #0a0a0a;
		border-radius: 12px;
		overflow: hidden;
		font-family: inherit;
	}

	.ogv-canvas {
		width: 100%;
		height: 100%;
	}

	/* ── Loading / error center ── */
	.ogv-center {
		position: absolute;
		inset: 0;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 12px;
	}

	.ogv-spinner {
		width: 28px;
		height: 28px;
		border: 2px solid rgba(255,255,255,0.1);
		border-top-color: #3b82f6;
		border-radius: 50%;
		animation: spin 0.7s linear infinite;
	}

	@keyframes spin { to { transform: rotate(360deg); } }

	.ogv-error {
		color: #f87171;
		font-size: 13px;
	}

	.ogv-btn {
		padding: 6px 16px;
		font-size: 12px;
		background: rgba(255,255,255,0.08);
		border: 1px solid rgba(255,255,255,0.12);
		border-radius: 6px;
		color: #e2e8f0;
		cursor: pointer;
	}

	.ogv-btn:hover { background: rgba(255,255,255,0.14); }

	/* ── Labels overlay ── */
	.ogv-labels {
		position: absolute;
		inset: 0;
		pointer-events: none;
		overflow: hidden;
	}

	.ogv-label {
		position: absolute;
		transform: translateX(-50%);
		white-space: nowrap;
		font-size: 10px;
		font-weight: 500;
		color: #9ca3af;
		background: rgba(0,0,0,0.5);
		padding: 1px 5px;
		border-radius: 4px;
		backdrop-filter: blur(4px);
		text-shadow: 0 1px 2px rgba(0,0,0,0.8);
		opacity: 0.8;
		transition: opacity 0.15s;
	}

	.ogv-label--hover {
		color: #f1f5f9;
		background: rgba(255,255,255,0.15);
		opacity: 1;
	}

	/* ── Tooltip ── */
	.ogv-tooltip {
		position: fixed;
		z-index: 50;
		pointer-events: none;
		background: rgba(15,15,15,0.96);
		border: 1px solid rgba(255,255,255,0.1);
		border-radius: 8px;
		padding: 10px 12px;
		max-width: 200px;
		backdrop-filter: blur(8px);
		box-shadow: 0 4px 20px rgba(0,0,0,0.6);
	}

	.ogv-tooltip__header {
		display: flex;
		align-items: center;
		gap: 6px;
		margin-bottom: 4px;
	}

	.ogv-tooltip__dot {
		width: 10px;
		height: 10px;
		border-radius: 50%;
		flex-shrink: 0;
		box-shadow: 0 0 6px currentColor;
	}

	.ogv-tooltip__type {
		font-size: 10px;
		color: #9ca3af;
		font-weight: 500;
	}

	.ogv-tooltip__name {
		font-size: 13px;
		font-weight: 600;
		color: #f1f5f9;
		word-break: break-word;
	}

	.ogv-tooltip__meta {
		font-size: 11px;
		color: #6b7280;
		margin-top: 3px;
	}

	.ogv-tooltip__hint {
		font-size: 10px;
		color: #4b5563;
		margin-top: 8px;
		padding-top: 8px;
		border-top: 1px solid rgba(255,255,255,0.06);
		display: flex;
		align-items: center;
		gap: 4px;
	}

	.ogv-tooltip__hint kbd {
		padding: 1px 4px;
		font-size: 9px;
		background: rgba(255,255,255,0.06);
		border-radius: 3px;
	}

	/* ── Stats panel ── */
	.ogv-stats {
		position: absolute;
		top: 16px;
		left: 16px;
		background: rgba(10,10,10,0.92);
		border: 1px solid rgba(255,255,255,0.08);
		border-radius: 8px;
		padding: 10px 12px;
		min-width: 160px;
		backdrop-filter: blur(8px);
	}

	.ogv-stats__title {
		font-size: 10px;
		text-transform: uppercase;
		letter-spacing: 0.08em;
		color: #4b5563;
		margin-bottom: 8px;
		display: flex;
		align-items: center;
		justify-content: space-between;
	}

	.ogv-stats__close {
		background: none;
		border: none;
		color: #4b5563;
		cursor: pointer;
		font-size: 14px;
		line-height: 1;
		padding: 0;
	}

	.ogv-stats__close:hover { color: #9ca3af; }

	.ogv-stats__row {
		font-size: 12px;
		color: #9ca3af;
		margin-bottom: 3px;
	}

	.ogv-stats__num {
		font-size: 14px;
		font-weight: 600;
		color: #f1f5f9;
		margin-right: 2px;
	}

	.ogv-stats__sep {
		height: 1px;
		background: rgba(255,255,255,0.06);
		margin: 8px 0;
	}

	.ogv-stats__label {
		font-size: 9px;
		text-transform: uppercase;
		letter-spacing: 0.08em;
		color: #374151;
		margin-bottom: 6px;
	}

	.ogv-stats__edge-row {
		display: flex;
		align-items: center;
		gap: 6px;
		margin-bottom: 4px;
	}

	.ogv-stats__edge-dot {
		width: 6px;
		height: 6px;
		border-radius: 50%;
		flex-shrink: 0;
	}

	.ogv-stats__edge-name {
		flex: 1;
		font-size: 11px;
		color: #6b7280;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.ogv-stats__edge-count {
		font-size: 10px;
		color: #4b5563;
	}

	.ogv-stats-toggle {
		position: absolute;
		top: 16px;
		left: 16px;
		width: 32px;
		height: 32px;
		display: flex;
		align-items: center;
		justify-content: center;
		background: rgba(10,10,10,0.92);
		border: 1px solid rgba(255,255,255,0.08);
		border-radius: 6px;
		color: #4b5563;
		cursor: pointer;
	}

	.ogv-stats-toggle:hover { color: #9ca3af; }

	/* ── Legend ── */
	.ogv-legend {
		position: absolute;
		bottom: 56px;
		left: 16px;
		background: rgba(10,10,10,0.92);
		border: 1px solid rgba(255,255,255,0.08);
		border-radius: 8px;
		padding: 8px 10px;
		backdrop-filter: blur(8px);
	}

	.ogv-legend__title {
		font-size: 9px;
		text-transform: uppercase;
		letter-spacing: 0.08em;
		color: #374151;
		margin-bottom: 6px;
	}

	.ogv-legend__row {
		display: flex;
		align-items: center;
		gap: 6px;
		margin-bottom: 4px;
		font-size: 11px;
		color: #6b7280;
	}

	.ogv-legend__dot {
		width: 8px;
		height: 8px;
		border-radius: 50%;
		flex-shrink: 0;
	}

	/* ── Camera controls ── */
	.ogv-controls {
		position: absolute;
		bottom: 16px;
		right: 16px;
		display: flex;
		flex-direction: column;
		gap: 4px;
	}

	.ogv-ctrl-btn {
		width: 32px;
		height: 32px;
		display: flex;
		align-items: center;
		justify-content: center;
		background: rgba(10,10,10,0.92);
		border: 1px solid rgba(255,255,255,0.08);
		border-radius: 6px;
		color: #4b5563;
		cursor: pointer;
		transition: color 0.15s;
	}

	.ogv-ctrl-btn:hover { color: #e2e8f0; }

	/* ── Controls hint bar ── */
	.ogv-hint-bar {
		position: absolute;
		top: 16px;
		right: 16px;
		background: rgba(10,10,10,0.92);
		border: 1px solid rgba(255,255,255,0.08);
		border-radius: 6px;
		padding: 6px 10px;
		font-size: 11px;
		color: #4b5563;
		display: flex;
		align-items: center;
		gap: 8px;
		backdrop-filter: blur(8px);
	}

	.ogv-hint-bar kbd {
		padding: 1px 5px;
		font-size: 9px;
		background: rgba(255,255,255,0.06);
		border-radius: 3px;
		color: #6b7280;
	}

	.ogv-hint {
		font-size: 12px;
		color: #4b5563;
	}

	.ogv-empty-icon {
		width: 72px;
		height: 72px;
		display: flex;
		align-items: center;
		justify-content: center;
		background: rgba(255,255,255,0.04);
		border-radius: 50%;
		color: #374151;
		margin-bottom: 8px;
	}
</style>
