<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import * as THREE from 'three';
	import { OrbitControls } from 'three/examples/jsm/controls/OrbitControls.js';
	import type { NodeTree, NodeType, NodeHealth } from '$lib/api/nodes/types';

	interface Props {
		nodes: NodeTree[];
		activeNodeId?: string | null;
		selectedId?: string | null;
		onSelect?: (node: NodeTree) => void;
		onNavigate?: (node: NodeTree) => void;
	}

	let { nodes, activeNodeId = null, selectedId = null, onSelect, onNavigate }: Props = $props();

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
	let tooltipNode: NodeTree | null = $state(null);

	// Flattened nodes for graph
	let flatNodes = $state<NodeTree[]>([]);

	// Label positions for rendering
	let labelPositions = $state<Map<string, { x: number; y: number; visible: boolean; name: string; type: NodeType; health: NodeHealth; isActive: boolean }>>(new Map());

	// Type colors - matching node types
	const typeColors: Record<NodeType | 'default', number> = {
		business: 0x3b82f6,    // Blue
		project: 0xf59e0b,     // Amber
		learning: 0x8b5cf6,    // Purple
		operational: 0x10b981, // Green
		default: 0x6b7280      // Gray
	};

	// Health glow intensities
	const healthGlowIntensity: Record<NodeHealth, number> = {
		healthy: 0.4,
		needs_attention: 0.25,
		critical: 0.15,
		not_started: 0.1
	};

	// Get color for node type
	function getNodeColor(node: NodeTree): number {
		return typeColors[node.type] || typeColors.default;
	}

	// Flatten the tree for graph layout
	function flattenTree(treeNodes: NodeTree[]): NodeTree[] {
		const flat: NodeTree[] = [];
		function traverse(node: NodeTree) {
			flat.push(node);
			if (node.children) {
				node.children.forEach(traverse);
			}
		}
		treeNodes.forEach(traverse);
		return flat;
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
		depth: number;
	}

	function runForceSimulation(simNodes: NodeSimulation[], iterations: number = 100) {
		const repulsionForce = 600;
		const attractionForce = 0.08;
		const damping = 0.9;
		const centerForce = 0.01;
		const depthSpacing = 30; // Vertical spacing by depth

		// Build parent-child map
		const childMap = new Map<string, string[]>();
		simNodes.forEach((node) => {
			if (node.parentId) {
				const children = childMap.get(node.parentId) || [];
				children.push(node.id);
				childMap.set(node.parentId, children);
			}
		});

		// Initial positioning by depth (layers)
		simNodes.forEach((node) => {
			node.y = -node.depth * depthSpacing;
		});

		for (let i = 0; i < iterations; i++) {
			// Repulsion between all nodes
			for (let j = 0; j < simNodes.length; j++) {
				for (let k = j + 1; k < simNodes.length; k++) {
					const nodeA = simNodes[j];
					const nodeB = simNodes[k];

					const dx = nodeB.x - nodeA.x;
					const dy = nodeB.y - nodeA.y;
					const dz = nodeB.z - nodeA.z;
					const distance = Math.sqrt(dx * dx + dy * dy + dz * dz) || 1;

					const force = repulsionForce / (distance * distance);
					const fx = (dx / distance) * force;
					const fy = (dy / distance) * force * 0.3; // Less vertical repulsion
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
			simNodes.forEach((node) => {
				if (node.parentId) {
					const parent = simNodes.find((n) => n.id === node.parentId);
					if (parent) {
						const dx = parent.x - node.x;
						const dy = parent.y - node.y;
						const dz = parent.z - node.z;

						node.vx += dx * attractionForce;
						node.vy += dy * attractionForce * 0.5;
						node.vz += dz * attractionForce;
						parent.vx -= dx * attractionForce * 0.3;
						parent.vz -= dz * attractionForce * 0.3;
					}
				}
			});

			// Center gravity (horizontal only)
			simNodes.forEach((node) => {
				node.vx -= node.x * centerForce;
				node.vz -= node.z * centerForce;
			});

			// Apply velocities with damping
			simNodes.forEach((node) => {
				node.x += node.vx;
				node.y += node.vy;
				node.z += node.vz;
				node.vx *= damping;
				node.vy *= damping;
				node.vz *= damping;
			});
		}

		return simNodes;
	}

	function getNodeDepth(node: NodeTree, allNodes: NodeTree[]): number {
		let depth = 0;
		let current = node;
		while (current.parent_id) {
			const parent = allNodes.find(n => n.id === current.parent_id);
			if (!parent) break;
			current = parent;
			depth++;
		}
		return depth;
	}

	function initScene() {
		if (!container) return;

		// Scene
		scene = new THREE.Scene();
		scene.background = new THREE.Color(0x0f0f11);

		// Camera
		const aspect = container.clientWidth / container.clientHeight;
		camera = new THREE.PerspectiveCamera(60, aspect, 0.1, 1000);
		camera.position.set(0, 80, 180);
		camera.lookAt(0, 0, 0);

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
		controls.maxDistance = 400;

		// Raycaster for mouse picking
		raycaster = new THREE.Raycaster();
		mouse = new THREE.Vector2();

		// Lights
		const ambientLight = new THREE.AmbientLight(0xffffff, 0.6);
		scene.add(ambientLight);

		const pointLight = new THREE.PointLight(0xffffff, 1, 500);
		pointLight.position.set(50, 100, 50);
		scene.add(pointLight);

		const pointLight2 = new THREE.PointLight(0xffffff, 0.5, 500);
		pointLight2.position.set(-50, -50, -50);
		scene.add(pointLight2);

		// Add grid helper for depth perception
		const gridHelper = new THREE.GridHelper(200, 20, 0x333333, 0x222222);
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

	// Properly dispose Three.js objects to prevent memory leaks
	function disposeObject(obj: THREE.Object3D) {
		if (obj instanceof THREE.Mesh) {
			if (obj.geometry) obj.geometry.dispose();
			if (obj.material) {
				if (Array.isArray(obj.material)) {
					obj.material.forEach(m => m.dispose());
				} else {
					obj.material.dispose();
				}
			}
		}
		if (obj instanceof THREE.Line) {
			if (obj.geometry) obj.geometry.dispose();
			if (obj.material) {
				if (Array.isArray(obj.material)) {
					obj.material.forEach(m => m.dispose());
				} else {
					obj.material.dispose();
				}
			}
		}
	}

	function clearGraph() {
		// Dispose and remove node meshes
		nodeMeshes.forEach((mesh) => {
			mesh.traverse(disposeObject);
			scene.remove(mesh);
		});
		// Dispose and remove edges
		edges.forEach((edge) => {
			disposeObject(edge);
			scene.remove(edge);
		});
		nodeMeshes.clear();
		nodePositions.clear();
		edges = [];
	}

	function createGraph() {
		// Clear existing with proper disposal
		clearGraph();

		// Flatten the tree
		flatNodes = flattenTree(nodes);

		if (flatNodes.length === 0) return;

		// Initialize simulation nodes with random positions
		const simNodes: NodeSimulation[] = flatNodes.map((node) => ({
			id: node.id,
			x: (Math.random() - 0.5) * 100,
			y: 0,
			z: (Math.random() - 0.5) * 100,
			vx: 0,
			vy: 0,
			vz: 0,
			parentId: node.parent_id || null,
			depth: getNodeDepth(node, flatNodes)
		}));

		// Run force simulation
		const simulatedNodes = runForceSimulation(simNodes, 150);

		// Create node meshes
		simulatedNodes.forEach((simNode) => {
			const node = flatNodes.find((n) => n.id === simNode.id);
			if (!node) return;

			// Base size varies by whether it has children
			const baseSize = node.children_count > 0 ? 4.5 : 3.5;
			const geometry = new THREE.SphereGeometry(baseSize, 32, 32);
			const color = getNodeColor(node);
			const glowIntensity = healthGlowIntensity[node.health] || 0.2;

			const material = new THREE.MeshPhongMaterial({
				color,
				emissive: color,
				emissiveIntensity: glowIntensity,
				shininess: 100
			});

			const mesh = new THREE.Mesh(geometry, material);
			mesh.position.set(simNode.x, simNode.y, simNode.z);
			mesh.userData = { nodeId: node.id, node };

			scene.add(mesh);
			nodeMeshes.set(node.id, mesh);
			nodePositions.set(node.id, new THREE.Vector3(simNode.x, simNode.y, simNode.z));

			// Add outer glow effect
			const glowSize = baseSize + 1.5;
			const glowGeometry = new THREE.SphereGeometry(glowSize, 32, 32);
			const glowMaterial = new THREE.MeshBasicMaterial({
				color,
				transparent: true,
				opacity: 0.12
			});
			const glow = new THREE.Mesh(glowGeometry, glowMaterial);
			mesh.add(glow);

			// Add pulsing ring for active node
			if (node.is_active || node.id === activeNodeId) {
				const ringGeometry = new THREE.RingGeometry(baseSize + 2, baseSize + 3, 32);
				const ringMaterial = new THREE.MeshBasicMaterial({
					color: 0xffffff,
					transparent: true,
					opacity: 0.6,
					side: THREE.DoubleSide
				});
				const ring = new THREE.Mesh(ringGeometry, ringMaterial);
				ring.userData.isActiveRing = true;
				mesh.add(ring);
			}
		});

		// Create edges for parent-child relationships
		flatNodes.forEach((node) => {
			if (node.parent_id) {
				const parentPos = nodePositions.get(node.parent_id);
				const childPos = nodePositions.get(node.id);

				if (parentPos && childPos) {
					const points = [parentPos, childPos];
					const geometry = new THREE.BufferGeometry().setFromPoints(points);
					const material = new THREE.LineBasicMaterial({
						color: 0x555555,
						transparent: true,
						opacity: 0.6
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
			const node = mesh.userData.node as NodeTree;
			const baseGlow = healthGlowIntensity[node.health] || 0.2;

			if (id === selectedId) {
				material.emissiveIntensity = 0.8;
				mesh.scale.setScalar(1.3);
			} else if (id === activeNodeId || node.is_active) {
				material.emissiveIntensity = baseGlow + 0.3;
				mesh.scale.setScalar(1.15);
			} else {
				material.emissiveIntensity = baseGlow;
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
			const node = hit.userData?.node as NodeTree | undefined;
			if (node) {
				onSelect?.(node);
			}
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
			const node = hit.userData?.node as NodeTree | undefined;
			if (node) {
				onNavigate?.(node);
			}
		}
	}

	function onResize() {
		if (!container || !camera || !renderer) return;

		camera.aspect = container.clientWidth / container.clientHeight;
		camera.updateProjectionMatrix();
		renderer.setSize(container.clientWidth, container.clientHeight);
	}

	// Animate active node rings
	let pulsePhase = 0;
	function animateActiveRings() {
		pulsePhase += 0.05;
		nodeMeshes.forEach((mesh) => {
			mesh.children.forEach((child) => {
				if (child.userData.isActiveRing) {
					const ring = child as THREE.Mesh;
					const mat = ring.material as THREE.MeshBasicMaterial;
					mat.opacity = 0.3 + Math.sin(pulsePhase) * 0.3;
					ring.rotation.z += 0.01;
				}
			});
		});
	}

	function updateLabelPositions() {
		if (!container || !camera || !renderer) return;

		const newPositions = new Map<string, { x: number; y: number; visible: boolean; name: string; type: NodeType; health: NodeHealth; isActive: boolean }>();
		const tempVector = new THREE.Vector3();

		nodeMeshes.forEach((mesh, id) => {
			const node = mesh.userData.node as NodeTree;
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
				name: node.name,
				type: node.type,
				health: node.health,
				isActive: node.is_active || node.id === activeNodeId
			});
		});

		labelPositions = newPositions;
	}

	function animate() {
		animationId = requestAnimationFrame(animate);
		controls?.update();
		animateActiveRings();
		renderer?.render(scene, camera);
		updateLabelPositions();
	}

	// Watch for node changes
	$effect(() => {
		if (scene && nodes) {
			createGraph();
		}
	});

	// Watch for selection changes
	$effect(() => {
		if (scene && (selectedId !== undefined || activeNodeId !== undefined)) {
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
		// Properly dispose all Three.js resources
		clearGraph();
		if (controls) {
			controls.dispose();
		}
		if (renderer) {
			renderer.domElement.removeEventListener('mousemove', onMouseMove);
			renderer.domElement.removeEventListener('click', onClick);
			renderer.domElement.removeEventListener('dblclick', onDoubleClick);
			renderer.dispose();
			renderer.forceContextLoss();
		}
		window.removeEventListener('resize', onResize);
	});

	// Get health status display
	function getHealthDisplay(health: NodeHealth): { label: string; color: string } {
		const displays: Record<NodeHealth, { label: string; color: string }> = {
			healthy: { label: 'Healthy', color: 'text-green-400' },
			needs_attention: { label: 'Needs Attention', color: 'text-yellow-400' },
			critical: { label: 'Critical', color: 'text-red-400' },
			not_started: { label: 'Not Started', color: 'text-gray-400' }
		};
		return displays[health] || displays.not_started;
	}

	// Get type display name
	function getTypeDisplay(type: NodeType): string {
		const displays: Record<NodeType, string> = {
			business: 'Business',
			project: 'Project',
			learning: 'Learning',
			operational: 'Operational'
		};
		return displays[type] || type;
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
					style="left: {label.x}px; top: {label.y + 14}px; opacity: {hoveredNode === id ? 1 : 0.85};"
				>
					<div class="flex flex-col items-center gap-0.5">
						<span
							class="text-xs font-medium px-2 py-0.5 rounded-md backdrop-blur-sm max-w-[140px] truncate
								{hoveredNode === id ? 'bg-white/20 text-white' : 'bg-black/50 text-gray-200'}
								{label.isActive ? 'ring-1 ring-white/50' : ''}"
							style="text-shadow: 0 1px 2px rgba(0,0,0,0.8);"
						>
							{label.name}
						</span>
						{#if label.isActive}
							<span class="text-[9px] text-green-400 font-medium">ACTIVE</span>
						{/if}
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
					<span class="text-xs font-medium text-gray-300">{getTypeDisplay(tooltipNode.type)}</span>
					{#if tooltipNode.is_active || tooltipNode.id === activeNodeId}
						<span class="px-1.5 py-0.5 text-[10px] bg-green-500/20 text-green-400 rounded font-medium">Active</span>
					{/if}
				</div>
				<div class="text-sm font-semibold text-white truncate">
					{tooltipNode.name}
				</div>
				<div class="text-xs {getHealthDisplay(tooltipNode.health).color} mt-1">
					{getHealthDisplay(tooltipNode.health).label}
				</div>
				{#if tooltipNode.children_count > 0}
					<div class="text-xs text-gray-400 mt-1">
						{tooltipNode.children_count} child node{tooltipNode.children_count !== 1 ? 's' : ''}
					</div>
				{/if}
				{#if tooltipNode.purpose}
					<div class="text-xs text-gray-500 mt-2 pt-2 border-t border-gray-700 line-clamp-2">
						{tooltipNode.purpose}
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
			onclick={() => { if (camera && controls) { camera.position.set(0, 80, 180); camera.lookAt(0, 0, 0); controls.reset(); }}}
			class="btn-pill btn-pill-ghost btn-pill-icon bg-gray-900/90 border border-gray-700 text-gray-400 hover:bg-gray-800 hover:text-white"
			title="Reset view"
		>
			<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
			</svg>
		</button>
		<!-- Zoom In -->
		<button
			onclick={() => { if (camera) { camera.position.z = Math.max(50, camera.position.z - 30); }}}
			class="btn-pill btn-pill-ghost btn-pill-icon bg-gray-900/90 border border-gray-700 text-gray-400 hover:bg-gray-800 hover:text-white"
			title="Zoom in"
		>
			<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0zM10 7v3m0 0v3m0-3h3m-3 0H7" />
			</svg>
		</button>
		<!-- Zoom Out -->
		<button
			onclick={() => { if (camera) { camera.position.z = Math.min(400, camera.position.z + 30); }}}
			class="btn-pill btn-pill-ghost btn-pill-icon bg-gray-900/90 border border-gray-700 text-gray-400 hover:bg-gray-800 hover:text-white"
			title="Zoom out"
		>
			<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0zM13 10H7" />
			</svg>
		</button>
	</div>

	<!-- Legend -->
	<div class="absolute bottom-4 left-4 bg-gray-900/90 backdrop-blur-sm border border-gray-700 rounded-lg p-3">
		<div class="text-[10px] text-gray-500 uppercase tracking-wider mb-2">Node Types</div>
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
		<div class="border-t border-gray-700 mt-2 pt-2">
			<div class="text-[10px] text-gray-500 uppercase tracking-wider mb-1">Health</div>
			<div class="flex items-center gap-3 text-[10px]">
				<span class="text-green-400">Bright = Healthy</span>
				<span class="text-gray-500">Dim = Critical</span>
			</div>
		</div>
	</div>

	<!-- Stats -->
	<div class="absolute top-4 left-4 bg-gray-900/90 backdrop-blur-sm border border-gray-700 rounded-lg px-3 py-2">
		<div class="text-[10px] text-gray-500 uppercase tracking-wider mb-1">Node Graph</div>
		<div class="text-xs text-gray-300">
			<span class="text-white font-semibold">{flatNodes.length}</span> nodes
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
	{#if flatNodes.length === 0}
		<div class="absolute inset-0 flex flex-col items-center justify-center bg-gray-50 dark:bg-[#0f0f11]">
			<div class="w-20 h-20 rounded-full bg-gray-800 flex items-center justify-center mb-4">
				<svg class="w-10 h-10 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9" />
				</svg>
			</div>
			<h3 class="text-lg font-medium text-gray-300 mb-1">No nodes to visualize</h3>
			<p class="text-sm text-gray-500">Create some nodes to see them in the graph</p>
		</div>
	{/if}
</div>
