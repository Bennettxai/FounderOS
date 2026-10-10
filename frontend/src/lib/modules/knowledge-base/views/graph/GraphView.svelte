<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import * as THREE from 'three';
	import { OrbitControls } from 'three/examples/jsm/controls/OrbitControls.js';
	import type { DocumentMeta } from '../../entities/types';

	// Internal node type for the graph
	interface GraphNode {
		id: string;
		name: string;
		type: string;
		parent_id: string | null;
		icon: string | null;
		word_count: number;
	}

	interface Props {
		documents: DocumentMeta[];
		selectedId?: string | null;
		onSelect?: (doc: DocumentMeta) => void;
		onNavigate?: (doc: DocumentMeta) => void;
	}

	let { documents, selectedId = null, onSelect, onNavigate }: Props = $props();

	// Convert DocumentMeta to internal graph node format
	function toGraphNode(doc: DocumentMeta): GraphNode {
		return {
			id: doc.id,
			name: doc.title || 'Untitled',
			type: doc.type || 'document',
			parent_id: doc.parent_id,
			icon: typeof doc.icon === 'string' ? doc.icon : doc.icon?.value || null,
			word_count: 0 // DocumentMeta doesn't have word_count
		};
	}

	// Get the original document from a graph node
	function getDocument(nodeId: string): DocumentMeta | undefined {
		return documents.find(d => d.id === nodeId);
	}

	// Derived graph nodes
	let graphNodes = $derived(documents.map(toGraphNode));

	let container: HTMLDivElement;
	let scene: THREE.Scene;
	let camera: THREE.PerspectiveCamera;
	let renderer: THREE.WebGLRenderer;
	let controls: OrbitControls;
	let animationId: number;
	let raycaster: THREE.Raycaster;
	let mouse: THREE.Vector2;

	// Node meshes and data
	let nodeMeshes: Map<string, THREE.Mesh> = new Map();
	let nodePositions: Map<string, THREE.Vector3> = new Map();
	let edges = $state<THREE.Line[]>([]);
	let hoveredNode: string | null = $state(null);
	let tooltipPosition = $state({ x: 0, y: 0 });
	let tooltipNode: GraphNode | null = $state(null);

	// Label positions for rendering
	let labelPositions = $state<Map<string, { x: number; y: number; visible: boolean; name: string; type: string }>>(new Map());

	// Type colors
	const typeColors: Record<string, number> = {
		business: 0x3b82f6, // Blue
		person: 0x10b981, // Green
		project: 0xf59e0b, // Amber
		document: 0x8b5cf6, // Purple
		profile: 0xec4899, // Pink
		default: 0x6b7280 // Gray
	};

	// Get color for node type
	function getNodeColor(node: GraphNode): number {
		const typeLower = (node.type || 'default').toLowerCase();
		return typeColors[typeLower] || typeColors.default;
	}

	// Force-directed layout simulation
	interface NodeSimulation {
		id: string;
		x: number;
		y: number;
		z: number;
		vx: number;
		vy: number;
		vz: number;
		parentId: string | null;
	}

	function runForceSimulation(nodes: NodeSimulation[], iterations: number = 100) {
		const repulsionForce = 500;
		const attractionForce = 0.05;
		const damping = 0.9;
		const centerForce = 0.01;

		// Build parent-child map
		const childMap = new Map<string, string[]>();
		nodes.forEach((node) => {
			if (node.parentId) {
				const children = childMap.get(node.parentId) || [];
				children.push(node.id);
				childMap.set(node.parentId, children);
			}
		});

		for (let i = 0; i < iterations; i++) {
			// Repulsion between all nodes
			for (let j = 0; j < nodes.length; j++) {
				for (let k = j + 1; k < nodes.length; k++) {
					const nodeA = nodes[j];
					const nodeB = nodes[k];

					const dx = nodeB.x - nodeA.x;
					const dy = nodeB.y - nodeA.y;
					const dz = nodeB.z - nodeA.z;
					const distance = Math.sqrt(dx * dx + dy * dy + dz * dz) || 1;

					const force = repulsionForce / (distance * distance);
					const fx = (dx / distance) * force;
					const fy = (dy / distance) * force;
					const fz = (dz / distance) * force;

					nodeA.vx -= fx;
					nodeA.vy -= fy;
					nodeA.vz -= fz;
					nodeB.vx += fx;
					nodeB.vy += fy;
					nodeB.vz += fz;
				}
			}

			// Attraction between connected nodes (parent-child)
			nodes.forEach((node) => {
				if (node.parentId) {
					const parent = nodes.find((n) => n.id === node.parentId);
					if (parent) {
						const dx = parent.x - node.x;
						const dy = parent.y - node.y;
						const dz = parent.z - node.z;

						node.vx += dx * attractionForce;
						node.vy += dy * attractionForce;
						node.vz += dz * attractionForce;
						parent.vx -= dx * attractionForce * 0.5;
						parent.vy -= dy * attractionForce * 0.5;
						parent.vz -= dz * attractionForce * 0.5;
					}
				}
			});

			// Center gravity
			nodes.forEach((node) => {
				node.vx -= node.x * centerForce;
				node.vy -= node.y * centerForce;
				node.vz -= node.z * centerForce;
			});

			// Apply velocities with damping
			nodes.forEach((node) => {
				node.x += node.vx;
				node.y += node.vy;
				node.z += node.vz;
				node.vx *= damping;
				node.vy *= damping;
				node.vz *= damping;
			});
		}

		return nodes;
	}

	function initScene() {
		if (!container) return;

		// Scene
		scene = new THREE.Scene();
		scene.background = new THREE.Color(0x0f0f11);

		// Camera
		const aspect = container.clientWidth / container.clientHeight;
		camera = new THREE.PerspectiveCamera(60, aspect, 0.1, 1000);
		camera.position.set(0, 0, 150);

		// Renderer
		renderer = new THREE.WebGLRenderer({ antialias: true });
		renderer.setSize(container.clientWidth, container.clientHeight);
		renderer.setPixelRatio(window.devicePixelRatio);
		container.appendChild(renderer.domElement);

		// Controls
		controls = new OrbitControls(camera, renderer.domElement);
		controls.enableDamping = true;
		controls.dampingFactor = 0.05;
		controls.minDistance = 50;
		controls.maxDistance = 300;

		// Raycaster for mouse picking
		raycaster = new THREE.Raycaster();
		mouse = new THREE.Vector2();

		// Lights
		const ambientLight = new THREE.AmbientLight(0xffffff, 0.6);
		scene.add(ambientLight);

		const pointLight = new THREE.PointLight(0xffffff, 1, 500);
		pointLight.position.set(50, 50, 50);
		scene.add(pointLight);

		// Add grid helper for depth perception
		const gridHelper = new THREE.GridHelper(200, 20, 0x333333, 0x222222);
		gridHelper.rotation.x = Math.PI / 2;
		scene.add(gridHelper);

		// Create nodes and edges
		createGraph();

		// Event listeners
		renderer.domElement.addEventListener('mousemove', onMouseMove);
		renderer.domElement.addEventListener('click', onClick);
		renderer.domElement.addEventListener('dblclick', onDoubleClick);
		window.addEventListener('resize', onResize);

		// Start animation
		animate();
	}

	function createGraph() {
		// Clear existing
		nodeMeshes.forEach((mesh) => scene.remove(mesh));
		edges.forEach((edge) => scene.remove(edge));
		nodeMeshes.clear();
		nodePositions.clear();
		edges = [];

		if (graphNodes.length === 0) return;

		// Initialize simulation nodes with random positions
		const simNodes: NodeSimulation[] = graphNodes.map((node) => ({
			id: node.id,
			x: (Math.random() - 0.5) * 100,
			y: (Math.random() - 0.5) * 100,
			z: (Math.random() - 0.5) * 100,
			vx: 0,
			vy: 0,
			vz: 0,
			parentId: node.parent_id || null
		}));

		// Run force simulation
		const simulatedNodes = runForceSimulation(simNodes, 150);

		// Create node meshes
		simulatedNodes.forEach((simNode) => {
			const node = graphNodes.find((n) => n.id === simNode.id);
			if (!node) return;

			const geometry = new THREE.SphereGeometry(3, 32, 32);
			const color = getNodeColor(node);
			const material = new THREE.MeshPhongMaterial({
				color,
				emissive: color,
				emissiveIntensity: 0.2,
				shininess: 100
			});

			const mesh = new THREE.Mesh(geometry, material);
			mesh.position.set(simNode.x, simNode.y, simNode.z);
			mesh.userData = { nodeId: node.id, node };

			scene.add(mesh);
			nodeMeshes.set(node.id, mesh);
			nodePositions.set(node.id, new THREE.Vector3(simNode.x, simNode.y, simNode.z));

			// Add glow effect
			const glowGeometry = new THREE.SphereGeometry(4, 32, 32);
			const glowMaterial = new THREE.MeshBasicMaterial({
				color,
				transparent: true,
				opacity: 0.15
			});
			const glow = new THREE.Mesh(glowGeometry, glowMaterial);
			mesh.add(glow);
		});

		// Create edges for parent-child relationships
		graphNodes.forEach((node) => {
			if (node.parent_id) {
				const parentPos = nodePositions.get(node.parent_id);
				const childPos = nodePositions.get(node.id);

				if (parentPos && childPos) {
					const points = [parentPos, childPos];
					const geometry = new THREE.BufferGeometry().setFromPoints(points);
					const material = new THREE.LineBasicMaterial({
						color: 0x444444,
						transparent: true,
						opacity: 0.5
					});
					const line = new THREE.Line(geometry, material);
					scene.add(line);
					edges.push(line);
				}
			}
		});

		// Highlight selected node
		updateSelectedNode();
	}

	function updateSelectedNode() {
		nodeMeshes.forEach((mesh, id) => {
			const material = mesh.material as THREE.MeshPhongMaterial;
			if (id === selectedId) {
				material.emissiveIntensity = 0.8;
				mesh.scale.setScalar(1.3);
			} else {
				material.emissiveIntensity = 0.2;
				mesh.scale.setScalar(1);
			}
		});
	}

	function onMouseMove(event: MouseEvent) {
		if (!container || !renderer) return;

		const rect = renderer.domElement.getBoundingClientRect();
		mouse.x = ((event.clientX - rect.left) / rect.width) * 2 - 1;
		mouse.y = -((event.clientY - rect.top) / rect.height) * 2 + 1;

		tooltipPosition = { x: event.clientX, y: event.clientY };

		// Check for hover
		raycaster.setFromCamera(mouse, camera);
		const meshArray = Array.from(nodeMeshes.values());
		const intersects = raycaster.intersectObjects(meshArray);

		if (intersects.length > 0) {
			const hit = intersects[0].object as THREE.Mesh;
			const nodeId = hit.userData.nodeId;
			if (nodeId !== hoveredNode) {
				hoveredNode = nodeId;
				tooltipNode = hit.userData.node;
				renderer.domElement.style.cursor = 'pointer';
			}
		} else {
			hoveredNode = null;
			tooltipNode = null;
			renderer.domElement.style.cursor = 'grab';
		}
	}

	function onClick(event: MouseEvent) {
		if (!container || !renderer) return;

		const rect = renderer.domElement.getBoundingClientRect();
		mouse.x = ((event.clientX - rect.left) / rect.width) * 2 - 1;
		mouse.y = -((event.clientY - rect.top) / rect.height) * 2 + 1;

		raycaster.setFromCamera(mouse, camera);
		const meshArray = Array.from(nodeMeshes.values());
		const intersects = raycaster.intersectObjects(meshArray);

		if (intersects.length > 0) {
			const hit = intersects[0].object as THREE.Mesh;
			const nodeId = hit.userData.nodeId as string;
			const doc = getDocument(nodeId);
			if (doc) onSelect?.(doc);
		}
	}

	function onDoubleClick(event: MouseEvent) {
		if (!container || !renderer) return;

		const rect = renderer.domElement.getBoundingClientRect();
		mouse.x = ((event.clientX - rect.left) / rect.width) * 2 - 1;
		mouse.y = -((event.clientY - rect.top) / rect.height) * 2 + 1;

		raycaster.setFromCamera(mouse, camera);
		const meshArray = Array.from(nodeMeshes.values());
		const intersects = raycaster.intersectObjects(meshArray);

		if (intersects.length > 0) {
			const hit = intersects[0].object as THREE.Mesh;
			const nodeId = hit.userData.nodeId as string;
			const doc = getDocument(nodeId);
			if (doc) onNavigate?.(doc);
		}
	}

	function onResize() {
		if (!container || !camera || !renderer) return;

		camera.aspect = container.clientWidth / container.clientHeight;
		camera.updateProjectionMatrix();
		renderer.setSize(container.clientWidth, container.clientHeight);
	}

	function updateLabelPositions() {
		if (!container || !camera || !renderer) return;

		const newPositions = new Map<string, { x: number; y: number; visible: boolean; name: string; type: string }>();
		const tempVector = new THREE.Vector3();

		nodeMeshes.forEach((mesh, id) => {
			const node = mesh.userData.node as GraphNode;
			if (!node) return;

			// Get world position
			mesh.getWorldPosition(tempVector);

			// Project to screen coordinates
			tempVector.project(camera);

			// Convert to CSS coordinates
			const x = (tempVector.x * 0.5 + 0.5) * container.clientWidth;
			const y = (-tempVector.y * 0.5 + 0.5) * container.clientHeight;

			// Check if in front of camera
			const visible = tempVector.z < 1;

			newPositions.set(id, {
				x,
				y,
				visible,
				name: node.name || 'Untitled',
				type: node.type || 'document'
			});
		});

		labelPositions = newPositions;
	}

	function animate() {
		animationId = requestAnimationFrame(animate);
		controls?.update();
		renderer?.render(scene, camera);
		updateLabelPositions();
	}

	// Watch for document changes
	$effect(() => {
		if (scene && documents) {
			createGraph();
		}
	});

	// Watch for selection changes
	$effect(() => {
		if (scene && selectedId !== undefined) {
			updateSelectedNode();
		}
	});

	onMount(() => {
		initScene();
	});

	onDestroy(() => {
		if (animationId) {
			cancelAnimationFrame(animationId);
		}
		if (renderer) {
			renderer.domElement.removeEventListener('mousemove', onMouseMove);
			renderer.domElement.removeEventListener('click', onClick);
			renderer.domElement.removeEventListener('dblclick', onDoubleClick);
			renderer.dispose();
		}
		window.removeEventListener('resize', onResize);
	});

	// Get type label for display
	function getTypeLabel(type: string | null | undefined): string {
		if (!type) return 'Document';
		return type.charAt(0).toUpperCase() + type.slice(1).toLowerCase();
	}
</script>

<div class="relative w-full h-full min-h-[500px] bg-gray-50 dark:bg-[#0f0f11] rounded-xl overflow-hidden">
	<!-- 3D Canvas Container -->
	<div bind:this={container} class="w-full h-full"></div>

	<!-- Node Labels -->
	<div class="absolute inset-0 pointer-events-none overflow-hidden">
		{#each [...labelPositions] as [id, label] (id)}
			{#if label.visible}
				<div
					class="absolute transform -translate-x-1/2 whitespace-nowrap transition-opacity duration-150"
					style="left: {label.x}px; top: {label.y + 12}px; opacity: {hoveredNode === id ? 1 : 0.85};"
				>
					<div class="flex flex-col items-center">
						<span
							class="text-xs font-medium px-2 py-0.5 rounded-md backdrop-blur-sm max-w-[120px] truncate
								{hoveredNode === id ? 'bg-white/20 text-white' : 'bg-black/40 text-gray-200'}"
							style="text-shadow: 0 1px 2px rgba(0,0,0,0.8);"
						>
							{label.name}
						</span>
					</div>
				</div>
			{/if}
		{/each}
	</div>

	<!-- Tooltip -->
	{#if tooltipNode && hoveredNode}
		<div
			class="fixed z-50 pointer-events-none"
			style="left: {tooltipPosition.x + 12}px; top: {tooltipPosition.y + 12}px;"
		>
			<div
				class="bg-gray-900/95 border border-gray-600 rounded-lg shadow-2xl p-3 max-w-xs backdrop-blur-sm"
			>
				<div class="flex items-center gap-2 mb-1">
					<div
						class="w-3 h-3 rounded-full ring-2 ring-white/20"
						style="background-color: #{getNodeColor(tooltipNode).toString(16).padStart(6, '0')}"
					></div>
					<span class="text-xs font-medium text-gray-300 capitalize">{getTypeLabel(tooltipNode.type)}</span>
				</div>
				<div class="text-sm font-semibold text-white truncate">
					{tooltipNode.name || 'Untitled'}
				</div>
				{#if tooltipNode.word_count > 0}
					<div class="text-xs text-gray-400 mt-1">
						{tooltipNode.word_count.toLocaleString()} words
					</div>
				{/if}
				<div class="flex items-center gap-2 text-[10px] text-gray-500 mt-2 pt-2 border-t border-gray-700">
					<span class="px-1.5 py-0.5 bg-gray-800 rounded">Click</span> select
					<span class="px-1.5 py-0.5 bg-gray-800 rounded">Dbl-click</span> open
				</div>
			</div>
		</div>
	{/if}

	<!-- Control Buttons -->
	<div class="absolute bottom-4 right-4 flex flex-col gap-2">
		<!-- Reset View -->
		<button
			onclick={() => { if (camera && controls) { camera.position.set(0, 0, 150); controls.reset(); }}}
			class="w-9 h-9 flex items-center justify-center bg-gray-900/90 hover:bg-gray-800 border border-gray-700 rounded-lg text-gray-400 hover:text-white transition-colors"
			title="Reset view"
		>
			<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
			</svg>
		</button>
		<!-- Zoom In -->
		<button
			onclick={() => { if (camera) { camera.position.z = Math.max(50, camera.position.z - 20); }}}
			class="w-9 h-9 flex items-center justify-center bg-gray-900/90 hover:bg-gray-800 border border-gray-700 rounded-lg text-gray-400 hover:text-white transition-colors"
			title="Zoom in"
		>
			<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0zM10 7v3m0 0v3m0-3h3m-3 0H7" />
			</svg>
		</button>
		<!-- Zoom Out -->
		<button
			onclick={() => { if (camera) { camera.position.z = Math.min(300, camera.position.z + 20); }}}
			class="w-9 h-9 flex items-center justify-center bg-gray-900/90 hover:bg-gray-800 border border-gray-700 rounded-lg text-gray-400 hover:text-white transition-colors"
			title="Zoom out"
		>
			<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0zM13 10H7" />
			</svg>
		</button>
	</div>

	<!-- Legend -->
	<div class="absolute bottom-4 left-4 bg-gray-900/90 backdrop-blur-sm border border-gray-700 rounded-lg p-3">
		<div class="text-[10px] text-gray-500 uppercase tracking-wider mb-2">Legend</div>
		<div class="grid grid-cols-2 gap-x-4 gap-y-1.5">
			{#each Object.entries(typeColors).filter(([key]) => key !== 'default') as [type, color]}
				<div class="flex items-center gap-1.5">
					<div
						class="w-2.5 h-2.5 rounded-full"
						style="background-color: #{color.toString(16).padStart(6, '0')}"
					></div>
					<span class="text-xs text-gray-300 capitalize">{type}</span>
				</div>
			{/each}
		</div>
	</div>

	<!-- Stats -->
	<div class="absolute top-4 left-4 bg-gray-900/90 backdrop-blur-sm border border-gray-700 rounded-lg px-3 py-2">
		<div class="text-[10px] text-gray-500 uppercase tracking-wider mb-1">Page Graph</div>
		<div class="text-xs text-gray-300">
			<span class="text-white font-semibold">{graphNodes.length}</span> pages
			<span class="mx-1.5 text-gray-600">|</span>
			<span class="text-white font-semibold">{edges.length}</span> connections
		</div>
	</div>

	<!-- Controls hint -->
	<div class="absolute top-4 right-4 bg-gray-900/90 backdrop-blur-sm border border-gray-700 rounded-lg px-3 py-2">
		<div class="text-xs text-gray-400 flex items-center gap-3">
			<span><kbd class="px-1 py-0.5 text-[10px] bg-gray-800 rounded">Drag</kbd> rotate</span>
			<span><kbd class="px-1 py-0.5 text-[10px] bg-gray-800 rounded">Scroll</kbd> zoom</span>
			<span><kbd class="px-1 py-0.5 text-[10px] bg-gray-800 rounded">Right-drag</kbd> pan</span>
		</div>
	</div>

	<!-- Empty state -->
	{#if graphNodes.length === 0}
		<div class="absolute inset-0 flex flex-col items-center justify-center">
			<div class="w-20 h-20 rounded-full bg-gray-800 flex items-center justify-center mb-4">
				<svg class="w-10 h-10 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
				</svg>
			</div>
			<h3 class="text-lg font-medium text-gray-300 mb-1">No pages to visualize</h3>
			<p class="text-sm text-gray-500">Create some pages to see them connected in the graph</p>
		</div>
	{/if}
</div>
