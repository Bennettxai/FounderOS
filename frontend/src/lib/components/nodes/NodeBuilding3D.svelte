<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import * as THREE from 'three';
	import { OrbitControls } from 'three/examples/jsm/controls/OrbitControls.js';
	import type { NodeTree } from '$lib/api/nodes/types';

	interface Props {
		nodes: NodeTree[];
		activeNodeId?: string | null;
		selectedId?: string | null;
		onSelect?: (node: NodeTree) => void;
		onNavigate?: (node: NodeTree) => void;
		onCreateRoom?: () => void;
	}

	let { nodes, activeNodeId = null, selectedId = null, onSelect, onNavigate, onCreateRoom }: Props = $props();

	let container: HTMLDivElement;
	let scene: THREE.Scene;
	let camera: THREE.PerspectiveCamera;
	let renderer: THREE.WebGLRenderer;
	let controls: OrbitControls;
	let animationId: number;
	let raycaster: THREE.Raycaster;
	let mouse: THREE.Vector2;
	let roomMeshes: Map<string, { group: THREE.Group; node: NodeTree }> = new Map();
	let clickTimeout: ReturnType<typeof setTimeout> | null = null;
	let isInitialized = false;

	// Animated agent tracking
	let agentMeshes: Map<string, {
		agent: THREE.Group;
		leftArm: THREE.Mesh;
		rightArm: THREE.Mesh;
		head: THREE.Mesh;
	}> = new Map();

	// Activity status messages
	const activityMessages = [
		'Analyzing data...',
		'Processing nodes...',
		'Checking files...',
		'Running tasks...',
		'Optimizing system...',
		'Syncing changes...',
		'Building reports...',
		'Evaluating metrics...'
	];
	let currentActivityIndex = $state(0);
	let activityInterval: ReturnType<typeof setInterval>;

	// Building config
	const FLOOR_HEIGHT = 2.5;
	const ROOM_SIZE = 2;
	const ROOM_GAP = 0.3;
	const ROOMS_PER_ROW = 4;

	// Colors matching 2D version
	const typeColors: Record<string, number> = {
		business: 0x3b82f6,
		project: 0x10b981,
		learning: 0x8b5cf6,
		operational: 0xf59e0b
	};

	const healthColors: Record<string, number> = {
		healthy: 0x22c55e,
		needs_attention: 0xeab308,
		critical: 0xef4444,
		not_started: 0x6b7280
	};

	function flattenWithDepth(nodeList: NodeTree[], depth: number = 0): { node: NodeTree; depth: number }[] {
		let result: { node: NodeTree; depth: number }[] = [];
		for (const n of nodeList) {
			result.push({ node: n, depth });
			if (n.children && n.children.length > 0) {
				result = result.concat(flattenWithDepth(n.children, depth + 1));
			}
		}
		return result;
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

	function clearScene() {
		if (!scene) return;
		scene.traverse((obj) => {
			disposeObject(obj);
		});
		while (scene.children.length > 0) {
			scene.remove(scene.children[0]);
		}
		roomMeshes.clear();
		agentMeshes.clear();
	}

	function createBuilding() {
		if (!scene) return;

		// Clear existing building with proper disposal
		clearScene();

		const flatNodes = flattenWithDepth(nodes);
		const floorMap = new Map<number, NodeTree[]>();

		for (const { node, depth } of flatNodes) {
			if (!floorMap.has(depth)) {
				floorMap.set(depth, []);
			}
			floorMap.get(depth)!.push(node);
		}

		const numFloors = Math.max(...Array.from(floorMap.keys()), 0) + 1;
		const maxRoomsOnFloor = Math.max(...Array.from(floorMap.values()).map(r => r.length), 1);
		const buildingWidth = Math.min(maxRoomsOnFloor, ROOMS_PER_ROW) * (ROOM_SIZE + ROOM_GAP) + 2;
		const buildingDepth = Math.ceil(maxRoomsOnFloor / ROOMS_PER_ROW) * (ROOM_SIZE + ROOM_GAP) + 2;

		// Add lights
		const ambientLight = new THREE.AmbientLight(0x404060, 0.6);
		scene.add(ambientLight);

		const moonLight = new THREE.DirectionalLight(0x6666aa, 0.4);
		moonLight.position.set(-10, 20, 10);
		moonLight.castShadow = true;
		scene.add(moonLight);

		const frontLight = new THREE.DirectionalLight(0xffffff, 0.3);
		frontLight.position.set(0, 5, 15);
		scene.add(frontLight);

		// Ground plane
		const groundGeo = new THREE.PlaneGeometry(60, 60);
		const groundMat = new THREE.MeshStandardMaterial({
			color: 0x1e3a1e,
			roughness: 1
		});
		const ground = new THREE.Mesh(groundGeo, groundMat);
		ground.rotation.x = -Math.PI / 2;
		ground.position.y = -0.1;
		ground.receiveShadow = true;
		scene.add(ground);

		// Building group
		const building = new THREE.Group();

		// Foundation
		const foundationGeo = new THREE.BoxGeometry(buildingWidth + 1, 0.5, buildingDepth + 1);
		const foundationMat = new THREE.MeshStandardMaterial({ color: 0x374151 });
		const foundation = new THREE.Mesh(foundationGeo, foundationMat);
		foundation.position.y = 0.25;
		foundation.receiveShadow = true;
		building.add(foundation);

		// Create floors
		for (let floor = 0; floor < numFloors; floor++) {
			const floorY = floor * FLOOR_HEIGHT + 0.5;
			const rooms = floorMap.get(floor) || [];

			// Floor slab
			const floorGeo = new THREE.BoxGeometry(buildingWidth, 0.2, buildingDepth);
			const floorMat = new THREE.MeshStandardMaterial({ color: 0x475569 });
			const floorMesh = new THREE.Mesh(floorGeo, floorMat);
			floorMesh.position.y = floorY;
			floorMesh.receiveShadow = true;
			building.add(floorMesh);

			// Floor walls (outer shell)
			const wallMat = new THREE.MeshStandardMaterial({
				color: 0x334155,
				transparent: true,
				opacity: 0.3,
				side: THREE.DoubleSide
			});

			// Back wall
			const backWallGeo = new THREE.PlaneGeometry(buildingWidth, FLOOR_HEIGHT - 0.2);
			const backWall = new THREE.Mesh(backWallGeo, wallMat);
			backWall.position.set(0, floorY + FLOOR_HEIGHT / 2, -buildingDepth / 2);
			building.add(backWall);

			// Side walls
			const sideWallGeo = new THREE.PlaneGeometry(buildingDepth, FLOOR_HEIGHT - 0.2);
			const leftWall = new THREE.Mesh(sideWallGeo, wallMat);
			leftWall.position.set(-buildingWidth / 2, floorY + FLOOR_HEIGHT / 2, 0);
			leftWall.rotation.y = Math.PI / 2;
			building.add(leftWall);

			const rightWall = new THREE.Mesh(sideWallGeo, wallMat);
			rightWall.position.set(buildingWidth / 2, floorY + FLOOR_HEIGHT / 2, 0);
			rightWall.rotation.y = -Math.PI / 2;
			building.add(rightWall);

			// Floor number indicator
			createFloorNumber(floor + 1, -buildingWidth / 2 - 0.5, floorY + FLOOR_HEIGHT / 2, 0, building);

			// Create rooms
			rooms.forEach((node, index) => {
				const col = index % ROOMS_PER_ROW;
				const row = Math.floor(index / ROOMS_PER_ROW);
				const x = (col - (Math.min(rooms.length, ROOMS_PER_ROW) - 1) / 2) * (ROOM_SIZE + ROOM_GAP);
				const z = row * (ROOM_SIZE + ROOM_GAP) - buildingDepth / 4;

				const room = createRoom(node, floor, index);
				room.position.set(x, floorY + 0.1, z);
				building.add(room);
			});
		}

		// Roof
		const roofGeo = new THREE.BoxGeometry(buildingWidth + 0.5, 0.3, buildingDepth + 0.5);
		const roofMat = new THREE.MeshStandardMaterial({ color: 0x1f2937 });
		const roof = new THREE.Mesh(roofGeo, roofMat);
		roof.position.y = numFloors * FLOOR_HEIGHT + 0.65;
		roof.castShadow = true;
		building.add(roof);

		// Antenna
		const antennaGeo = new THREE.CylinderGeometry(0.05, 0.05, 1.5, 8);
		const antennaMat = new THREE.MeshStandardMaterial({ color: 0x6b7280 });
		const antenna = new THREE.Mesh(antennaGeo, antennaMat);
		antenna.position.y = numFloors * FLOOR_HEIGHT + 1.5;
		building.add(antenna);

		// Antenna light
		const lightGeo = new THREE.SphereGeometry(0.1, 16, 16);
		const lightMat = new THREE.MeshStandardMaterial({
			color: 0xff0000,
			emissive: 0xff0000,
			emissiveIntensity: 1
		});
		const antennaLight = new THREE.Mesh(lightGeo, lightMat);
		antennaLight.position.y = numFloors * FLOOR_HEIGHT + 2.3;
		building.add(antennaLight);

		// Sign on roof
		createBuildingSign(0, numFloors * FLOOR_HEIGHT + 1, buildingDepth / 2 + 0.3, building);

		scene.add(building);

		// Update camera to fit building
		const targetY = (numFloors * FLOOR_HEIGHT) / 2 + 1;
		camera.position.set(buildingWidth * 1.5, targetY + 3, buildingDepth * 2);
		camera.lookAt(0, targetY, 0);
		controls.target.set(0, targetY, 0);
		controls.update();
	}

	function createRoom(node: NodeTree, floor: number, index: number): THREE.Group {
		const room = new THREE.Group();
		room.userData = { node, floor, index };

		const isActive = node.id === activeNodeId;
		const isSelected = node.id === selectedId;
		const typeColor = typeColors[node.type] || 0x64748b;
		const healthColor = healthColors[node.health] || 0x6b7280;

		// Room base (floor)
		const baseGeo = new THREE.BoxGeometry(ROOM_SIZE, 0.05, ROOM_SIZE);
		const baseMat = new THREE.MeshStandardMaterial({
			color: typeColor,
			transparent: true,
			opacity: 0.3
		});
		const base = new THREE.Mesh(baseGeo, baseMat);
		base.position.y = 0.025;
		room.add(base);

		// Room walls (low walls to see inside)
		const wallHeight = FLOOR_HEIGHT - 0.5;
		const wallMat = new THREE.MeshStandardMaterial({
			color: 0x1f2937,
			transparent: true,
			opacity: isSelected ? 0.6 : 0.4,
			side: THREE.DoubleSide
		});

		// Back wall with window
		const backWallGeo = new THREE.PlaneGeometry(ROOM_SIZE, wallHeight);
		const backWall = new THREE.Mesh(backWallGeo, wallMat);
		backWall.position.set(0, wallHeight / 2 + 0.05, -ROOM_SIZE / 2);
		room.add(backWall);

		// Side walls (partial)
		const sideWallGeo = new THREE.PlaneGeometry(ROOM_SIZE * 0.3, wallHeight);
		const leftWall = new THREE.Mesh(sideWallGeo, wallMat);
		leftWall.position.set(-ROOM_SIZE / 2, wallHeight / 2 + 0.05, -ROOM_SIZE * 0.35);
		leftWall.rotation.y = Math.PI / 2;
		room.add(leftWall);

		const rightWall = new THREE.Mesh(sideWallGeo, wallMat);
		rightWall.position.set(ROOM_SIZE / 2, wallHeight / 2 + 0.05, -ROOM_SIZE * 0.35);
		rightWall.rotation.y = -Math.PI / 2;
		room.add(rightWall);

		// Window (front - glows when active)
		const windowGeo = new THREE.PlaneGeometry(ROOM_SIZE * 0.8, wallHeight * 0.7);
		const windowMat = new THREE.MeshStandardMaterial({
			color: isActive ? typeColor : 0x334155,
			emissive: isActive ? typeColor : 0x000000,
			emissiveIntensity: isActive ? 0.8 : 0,
			transparent: true,
			opacity: 0.8,
			side: THREE.DoubleSide
		});
		const windowMesh = new THREE.Mesh(windowGeo, windowMat);
		windowMesh.position.set(0, wallHeight / 2 + 0.1, ROOM_SIZE / 2);
		room.add(windowMesh);

		// Window frame
		const frameMat = new THREE.LineBasicMaterial({ color: 0x4b5563 });
		const framePoints = [
			new THREE.Vector3(-ROOM_SIZE * 0.4, 0.2, ROOM_SIZE / 2 + 0.01),
			new THREE.Vector3(ROOM_SIZE * 0.4, 0.2, ROOM_SIZE / 2 + 0.01),
			new THREE.Vector3(ROOM_SIZE * 0.4, wallHeight * 0.9, ROOM_SIZE / 2 + 0.01),
			new THREE.Vector3(-ROOM_SIZE * 0.4, wallHeight * 0.9, ROOM_SIZE / 2 + 0.01),
			new THREE.Vector3(-ROOM_SIZE * 0.4, 0.2, ROOM_SIZE / 2 + 0.01)
		];
		const frameGeo = new THREE.BufferGeometry().setFromPoints(framePoints);
		const frame = new THREE.Line(frameGeo, frameMat);
		room.add(frame);

		// Desk
		const deskGeo = new THREE.BoxGeometry(0.8, 0.4, 0.4);
		const deskMat = new THREE.MeshStandardMaterial({ color: 0x78350f });
		const desk = new THREE.Mesh(deskGeo, deskMat);
		desk.position.set(0, 0.2, 0);
		desk.castShadow = true;
		room.add(desk);

		// Monitor
		const monitorGeo = new THREE.BoxGeometry(0.4, 0.25, 0.02);
		const monitorMat = new THREE.MeshStandardMaterial({
			color: isActive ? 0x60a5fa : 0x1f2937,
			emissive: isActive ? typeColor : 0x000000,
			emissiveIntensity: isActive ? 0.5 : 0
		});
		const monitor = new THREE.Mesh(monitorGeo, monitorMat);
		monitor.position.set(0, 0.53, -0.05);
		room.add(monitor);

		// Chair
		const chairGeo = new THREE.BoxGeometry(0.25, 0.25, 0.25);
		const chairMat = new THREE.MeshStandardMaterial({ color: 0x374151 });
		const chair = new THREE.Mesh(chairGeo, chairMat);
		chair.position.set(0, 0.2, 0.5);
		room.add(chair);

		// Agent figure when active
		if (isActive) {
			const agent = createAgent(typeColor, node.id);
			agent.position.set(0, 0.4, 0.45);
			room.add(agent);

			// Add point light for active room
			const roomLight = new THREE.PointLight(typeColor, 1, 4);
			roomLight.position.set(0, wallHeight - 0.3, 0);
			room.add(roomLight);

			// Add keyboard mesh in front of agent
			const keyboardGeo = new THREE.BoxGeometry(0.3, 0.02, 0.1);
			const keyboardMat = new THREE.MeshStandardMaterial({ color: 0x1f2937 });
			const keyboard = new THREE.Mesh(keyboardGeo, keyboardMat);
			keyboard.position.set(0, 0.42, 0.1);
			room.add(keyboard);
		}

		// Health indicator
		const healthGeo = new THREE.SphereGeometry(0.08, 16, 16);
		const healthMat = new THREE.MeshStandardMaterial({
			color: healthColor,
			emissive: healthColor,
			emissiveIntensity: 0.8
		});
		const healthIndicator = new THREE.Mesh(healthGeo, healthMat);
		healthIndicator.position.set(ROOM_SIZE / 2 - 0.15, wallHeight + 0.1, ROOM_SIZE / 2 - 0.15);
		room.add(healthIndicator);

		// Room label
		createRoomLabel(node.name, `${floor + 1}${String(index + 1).padStart(2, '0')}`, typeColor, 0, wallHeight + 0.3, ROOM_SIZE / 2, room);

		// Selection indicator
		if (isSelected) {
			const ringGeo = new THREE.RingGeometry(ROOM_SIZE * 0.6, ROOM_SIZE * 0.65, 32);
			const ringMat = new THREE.MeshBasicMaterial({
				color: 0x3b82f6,
				side: THREE.DoubleSide,
				transparent: true,
				opacity: 0.6
			});
			const ring = new THREE.Mesh(ringGeo, ringMat);
			ring.rotation.x = -Math.PI / 2;
			ring.position.y = 0.02;
			room.add(ring);
		}

		roomMeshes.set(node.id, { group: room, node });
		return room;
	}

	function createAgent(color: number, nodeId: string): THREE.Group {
		const agent = new THREE.Group();

		// Body (torso)
		const bodyGeo = new THREE.CylinderGeometry(0.08, 0.1, 0.2, 8);
		const bodyMat = new THREE.MeshStandardMaterial({ color });
		const body = new THREE.Mesh(bodyGeo, bodyMat);
		body.position.y = 0.15;
		agent.add(body);

		// Head
		const headGeo = new THREE.SphereGeometry(0.07, 16, 16);
		const headMat = new THREE.MeshStandardMaterial({ color: 0xfcd34d });
		const head = new THREE.Mesh(headGeo, headMat);
		head.position.y = 0.3;
		agent.add(head);

		// Left arm (typing)
		const armGeo = new THREE.CylinderGeometry(0.02, 0.02, 0.15, 6);
		const armMat = new THREE.MeshStandardMaterial({ color });
		const leftArm = new THREE.Mesh(armGeo, armMat);
		leftArm.position.set(-0.1, 0.12, 0.05);
		leftArm.rotation.x = Math.PI / 4;
		leftArm.rotation.z = Math.PI / 6;
		agent.add(leftArm);

		// Right arm (typing)
		const rightArm = new THREE.Mesh(armGeo, armMat);
		rightArm.position.set(0.1, 0.12, 0.05);
		rightArm.rotation.x = Math.PI / 4;
		rightArm.rotation.z = -Math.PI / 6;
		agent.add(rightArm);

		// Legs
		const legGeo = new THREE.CylinderGeometry(0.025, 0.025, 0.12, 6);
		const legMat = new THREE.MeshStandardMaterial({ color: 0x1f2937 });
		const leftLeg = new THREE.Mesh(legGeo, legMat);
		leftLeg.position.set(-0.04, 0, 0);
		agent.add(leftLeg);

		const rightLeg = new THREE.Mesh(legGeo, legMat);
		rightLeg.position.set(0.04, 0, 0);
		agent.add(rightLeg);

		// Store parts for animation
		agentMeshes.set(nodeId, {
			agent,
			leftArm,
			rightArm,
			head
		});

		return agent;
	}

	function createFloorNumber(num: number, x: number, y: number, z: number, parent: THREE.Group) {
		const canvas = document.createElement('canvas');
		canvas.width = 64;
		canvas.height = 64;
		const ctx = canvas.getContext('2d')!;
		ctx.fillStyle = '#374151';
		ctx.fillRect(0, 0, 64, 64);
		ctx.fillStyle = '#94a3b8';
		ctx.font = 'bold 36px Arial';
		ctx.textAlign = 'center';
		ctx.textBaseline = 'middle';
		ctx.fillText(num.toString(), 32, 32);

		const texture = new THREE.CanvasTexture(canvas);
		const mat = new THREE.MeshBasicMaterial({ map: texture });
		const geo = new THREE.PlaneGeometry(0.8, 0.8);
		const mesh = new THREE.Mesh(geo, mat);
		mesh.position.set(x, y, z);
		mesh.rotation.y = Math.PI / 2;
		parent.add(mesh);
	}

	function createRoomLabel(name: string, roomNum: string, color: number, x: number, y: number, z: number, parent: THREE.Group) {
		const canvas = document.createElement('canvas');
		canvas.width = 256;
		canvas.height = 64;
		const ctx = canvas.getContext('2d')!;

		// Background
		ctx.fillStyle = 'rgba(0,0,0,0.8)';
		ctx.fillRect(0, 0, 256, 64);

		// Room number badge
		ctx.fillStyle = `#${color.toString(16).padStart(6, '0')}`;
		ctx.fillRect(0, 0, 50, 64);
		ctx.fillStyle = '#ffffff';
		ctx.font = 'bold 20px Arial';
		ctx.textAlign = 'center';
		ctx.fillText(roomNum, 25, 38);

		// Name
		ctx.fillStyle = '#e2e8f0';
		ctx.font = '16px Arial';
		ctx.textAlign = 'left';
		const displayName = name.length > 18 ? name.slice(0, 18) + '...' : name;
		ctx.fillText(displayName, 60, 38);

		const texture = new THREE.CanvasTexture(canvas);
		const mat = new THREE.MeshBasicMaterial({
			map: texture,
			transparent: true,
			side: THREE.DoubleSide
		});
		const geo = new THREE.PlaneGeometry(1.5, 0.4);
		const mesh = new THREE.Mesh(geo, mat);
		mesh.position.set(x, y, z);
		parent.add(mesh);
	}

	function createBuildingSign(x: number, y: number, z: number, parent: THREE.Group) {
		const canvas = document.createElement('canvas');
		canvas.width = 256;
		canvas.height = 64;
		const ctx = canvas.getContext('2d')!;
		ctx.fillStyle = '#1f2937';
		ctx.fillRect(0, 0, 256, 64);
		ctx.fillStyle = '#fbbf24';
		ctx.font = 'bold 24px Arial';
		ctx.textAlign = 'center';
		ctx.fillText('BUSINESSOS', 128, 40);

		const texture = new THREE.CanvasTexture(canvas);
		const mat = new THREE.MeshBasicMaterial({ map: texture, side: THREE.DoubleSide });
		const geo = new THREE.PlaneGeometry(3, 0.75);
		const mesh = new THREE.Mesh(geo, mat);
		mesh.position.set(x, y, z);
		parent.add(mesh);
	}

	function init() {
		if (!container || isInitialized) return;
		isInitialized = true;

		// Scene
		scene = new THREE.Scene();
		scene.background = new THREE.Color(0x0f172a);

		// Camera
		camera = new THREE.PerspectiveCamera(
			50,
			container.clientWidth / container.clientHeight,
			0.1,
			1000
		);
		camera.position.set(15, 10, 15);

		// Renderer
		renderer = new THREE.WebGLRenderer({ antialias: true });
		renderer.setSize(container.clientWidth, container.clientHeight);
		renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));
		renderer.shadowMap.enabled = true;
		renderer.shadowMap.type = THREE.PCFSoftShadowMap;
		container.appendChild(renderer.domElement);

		// Controls
		controls = new OrbitControls(camera, renderer.domElement);
		controls.enableDamping = true;
		controls.dampingFactor = 0.05;
		controls.minDistance = 5;
		controls.maxDistance = 40;
		controls.maxPolarAngle = Math.PI / 2.1;
		controls.target.set(0, 3, 0);

		// Raycaster
		raycaster = new THREE.Raycaster();
		mouse = new THREE.Vector2();

		// Event listeners
		renderer.domElement.addEventListener('click', onMouseClick);
		renderer.domElement.addEventListener('dblclick', onMouseDoubleClick);
		window.addEventListener('resize', onWindowResize);

		// Build scene
		createBuilding();

		// Start animation
		animate();
	}

	function animate() {
		animationId = requestAnimationFrame(animate);

		const time = Date.now() * 0.001;

		// Animate agents
		agentMeshes.forEach(({ agent, leftArm, rightArm, head }) => {
			// Typing animation - alternating arm movement
			const typingSpeed = 8;
			leftArm.rotation.x = Math.PI / 4 + Math.sin(time * typingSpeed) * 0.15;
			rightArm.rotation.x = Math.PI / 4 + Math.sin(time * typingSpeed + Math.PI) * 0.15;

			// Subtle head bobbing
			head.position.y = 0.3 + Math.sin(time * 2) * 0.01;
			head.rotation.y = Math.sin(time * 0.5) * 0.1;

			// Slight body sway
			agent.rotation.z = Math.sin(time * 1.5) * 0.02;
		});

		// Pulse active room lights
		roomMeshes.forEach(({ group, node }) => {
			if (node.id === activeNodeId) {
				const light = group.children.find(c => c instanceof THREE.PointLight) as THREE.PointLight;
				if (light) {
					light.intensity = 0.8 + Math.sin(time * 3) * 0.4;
				}

				// Also pulse the window glow
				group.traverse((child) => {
					if (child instanceof THREE.Mesh && child.material instanceof THREE.MeshStandardMaterial) {
						if (child.material.emissiveIntensity > 0) {
							child.material.emissiveIntensity = 0.6 + Math.sin(time * 2) * 0.3;
						}
					}
				});
			}
		});

		controls.update();
		renderer.render(scene, camera);
	}

	function getIntersectedRoom(event: MouseEvent): NodeTree | null {
		const rect = container.getBoundingClientRect();
		mouse.x = ((event.clientX - rect.left) / container.clientWidth) * 2 - 1;
		mouse.y = -((event.clientY - rect.top) / container.clientHeight) * 2 + 1;

		raycaster.setFromCamera(mouse, camera);
		const intersects = raycaster.intersectObjects(scene.children, true);

		for (const intersect of intersects) {
			let obj: THREE.Object3D | null = intersect.object;
			while (obj) {
				if (obj.userData.node) {
					return obj.userData.node as NodeTree;
				}
				obj = obj.parent;
			}
		}
		return null;
	}

	function onMouseClick(event: MouseEvent) {
		if (clickTimeout) {
			clearTimeout(clickTimeout);
			clickTimeout = null;
			return;
		}

		clickTimeout = setTimeout(() => {
			const node = getIntersectedRoom(event);
			if (node) {
				onSelect?.(node);
			}
			clickTimeout = null;
		}, 200);
	}

	function onMouseDoubleClick(event: MouseEvent) {
		if (clickTimeout) {
			clearTimeout(clickTimeout);
			clickTimeout = null;
		}

		const node = getIntersectedRoom(event);
		if (node) {
			onNavigate?.(node);
		}
	}

	function onWindowResize() {
		if (!container || !camera || !renderer) return;
		camera.aspect = container.clientWidth / container.clientHeight;
		camera.updateProjectionMatrix();
		renderer.setSize(container.clientWidth, container.clientHeight);
	}

	function cleanup() {
		if (animationId) {
			cancelAnimationFrame(animationId);
		}
		// Properly dispose all Three.js resources
		clearScene();
		if (controls) {
			controls.dispose();
		}
		if (renderer && container) {
			renderer.domElement.removeEventListener('click', onMouseClick);
			renderer.domElement.removeEventListener('dblclick', onMouseDoubleClick);
			renderer.dispose();
			renderer.forceContextLoss();
			if (renderer.domElement.parentNode === container) {
				container.removeChild(renderer.domElement);
			}
		}
		window.removeEventListener('resize', onWindowResize);
		isInitialized = false;
	}

	// React to node changes
	$effect(() => {
		if (isInitialized && nodes) {
			createBuilding();
		}
	});

	// React to selection changes
	$effect(() => {
		if (isInitialized && (activeNodeId !== null || selectedId !== null)) {
			createBuilding();
		}
	});

	onMount(() => {
		// Small delay to ensure container is ready
		setTimeout(() => init(), 100);

		// Start activity message cycling
		activityInterval = setInterval(() => {
			currentActivityIndex = (currentActivityIndex + 1) % activityMessages.length;
		}, 2500);
	});

	onDestroy(() => {
		cleanup();
		if (activityInterval) {
			clearInterval(activityInterval);
		}
	});

	// Stats for HUD
	const totalRooms = $derived(flattenWithDepth(nodes).length);
	const floorCount = $derived(Math.max(...flattenWithDepth(nodes).map(n => n.depth), 0) + 1);
	const activeRoom = $derived(flattenWithDepth(nodes).find(n => n.node.id === activeNodeId)?.node);
</script>

<div class="relative w-full h-full bg-slate-900">
	<div bind:this={container} class="w-full h-full"></div>

	<!-- HUD Overlay -->
	<div class="absolute top-4 left-4 bg-slate-800/90 backdrop-blur border border-slate-600 rounded-xl px-4 py-3">
		<div class="flex items-center gap-3 text-sm">
			<svg class="w-5 h-5 text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5" />
			</svg>
			<span class="text-white font-semibold">BusinessOS Tower</span>
			<span class="text-slate-500">|</span>
			<span class="text-slate-400"><span class="text-white">{floorCount}</span> Floors</span>
			<span class="text-slate-400"><span class="text-white">{totalRooms}</span> Rooms</span>
			<span class="text-slate-500">|</span>
			<button
				onclick={() => onCreateRoom?.()}
				class="btn-pill btn-pill-ghost btn-pill-sm flex items-center gap-1 bg-blue-500/20 border border-blue-400/50 text-blue-300 hover:bg-blue-500/30 hover:text-blue-200"
			>
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
				</svg>
				Add Room
			</button>
		</div>
	</div>

	<!-- Active Room Indicator -->
	{#if activeRoom}
		<div class="absolute top-4 right-4 bg-blue-500/20 border border-blue-400/50 rounded-xl px-4 py-3 flex flex-col gap-2">
			<div class="flex items-center gap-2">
				<div class="w-2 h-2 rounded-full bg-blue-400 animate-pulse"></div>
				<span class="text-sm text-blue-300">Agent in: <span class="font-medium text-blue-200">{activeRoom.name}</span></span>
			</div>
			<div class="flex items-center gap-2 text-xs">
				<!-- Animated typing indicator -->
				<div class="flex gap-0.5">
					<div class="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-bounce" style="animation-delay: 0ms;"></div>
					<div class="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-bounce" style="animation-delay: 150ms;"></div>
					<div class="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-bounce" style="animation-delay: 300ms;"></div>
				</div>
				<span class="text-emerald-300">{activityMessages[currentActivityIndex]}</span>
			</div>
		</div>
	{/if}

	<!-- Controls Hint -->
	<div class="absolute bottom-4 left-4 bg-slate-800/80 backdrop-blur border border-slate-600 rounded-lg px-3 py-2">
		<p class="text-xs text-slate-400">Drag to orbit | Scroll to zoom | Click to select</p>
	</div>

	<!-- Legend -->
	<div class="absolute bottom-4 right-4 bg-slate-800/80 backdrop-blur border border-slate-600 rounded-lg px-4 py-3">
		<div class="flex gap-4 text-xs">
			<div class="flex items-center gap-1.5">
				<div class="w-3 h-3 rounded bg-blue-500"></div>
				<span class="text-slate-400">Business</span>
			</div>
			<div class="flex items-center gap-1.5">
				<div class="w-3 h-3 rounded bg-emerald-500"></div>
				<span class="text-slate-400">Project</span>
			</div>
			<div class="flex items-center gap-1.5">
				<div class="w-3 h-3 rounded bg-violet-500"></div>
				<span class="text-slate-400">Learning</span>
			</div>
			<div class="flex items-center gap-1.5">
				<div class="w-3 h-3 rounded bg-amber-500"></div>
				<span class="text-slate-400">Operational</span>
			</div>
		</div>
	</div>
</div>
