<script lang="ts">
	import { T, useTask } from '@threlte/core';
	import { OrbitControls, interactivity } from '@threlte/extras';
	import * as THREE from 'three';
	import MemoryBubble from './MemoryBubble.svelte';
	import type { Memory } from '$lib/api/memory/types';
	import { onMount, onDestroy } from 'svelte';

	// Enable interactivity for click events on 3D objects - must be called inside Threlte context
	interactivity();

	// Electron pulse animation system - neural synapse effect
	interface ElectronPulse {
		id: number;
		from: [number, number, number];
		to: [number, number, number];
		progress: number; // 0 to 1
		color: string;
		opacity: number;
	}

	let pulses = $state<ElectronPulse[]>([]);
	let pulseIdCounter = $state(0);
	let pulseSpawnTimer: ReturnType<typeof setInterval> | null = null;

	interface Props {
		layoutResult: Array<{
			memory: Memory;
			position: [number, number, number];
			scale: number;
			nodeId: string | null;
		}>;
		selectedId: string | null;
		searchQuery: string;
		cameraPosition: [number, number, number];
		targetLookAt?: [number, number, number];
		autoRotate: boolean;
		nodeMap: Map<string, { name: string; type: string }>;
		getTypeColor: (type: string, memoryId?: string) => string;
		isHighlighted: (memory: Memory) => boolean;
		onBubbleClick: (memory: Memory) => void;
		onBackgroundClick?: () => void;
	}

	let {
		layoutResult,
		selectedId,
		searchQuery,
		cameraPosition,
		targetLookAt = [0, 0, 0],
		autoRotate,
		nodeMap,
		getTypeColor,
		isHighlighted,
		onBubbleClick,
		onBackgroundClick
	}: Props = $props();

	// Globe ALWAYS rotates - never stops, looks beautiful
	// User can still freely move camera while it rotates
	let effectiveAutoRotate = $derived(autoRotate);

	// Find nodes that are related (same nodeId/parent, or nearby)
	function findRelatedPairs(): Array<[number, number]> {
		const pairs: Array<[number, number]> = [];

		for (let i = 0; i < layoutResult.length; i++) {
			for (let j = i + 1; j < layoutResult.length; j++) {
				const nodeA = layoutResult[i];
				const nodeB = layoutResult[j];

				// Check if they share the same nodeId (parent context)
				if (nodeA.nodeId && nodeA.nodeId === nodeB.nodeId) {
					pairs.push([i, j]);
					continue;
				}

				// Check if they're nearby (within distance threshold)
				const dx = nodeA.position[0] - nodeB.position[0];
				const dy = nodeA.position[1] - nodeB.position[1];
				const dz = nodeA.position[2] - nodeB.position[2];
				const distance = Math.sqrt(dx*dx + dy*dy + dz*dz);

				// If close together, consider them related
				if (distance < 35) {
					pairs.push([i, j]);
				}
			}
		}

		return pairs;
	}

	// Spawn an electron pulse between RELATED nodes only
	function spawnPulse() {
		if (layoutResult.length < 2) return;

		const relatedPairs = findRelatedPairs();
		if (relatedPairs.length === 0) return;

		// Pick a random related pair
		const [idx1, idx2] = relatedPairs[Math.floor(Math.random() * relatedPairs.length)];

		// Randomly choose direction
		const [fromIdx, toIdx] = Math.random() > 0.5 ? [idx1, idx2] : [idx2, idx1];

		const from = layoutResult[fromIdx].position;
		const to = layoutResult[toIdx].position;

		// Random warm color for the pulse
		const colors = ['#C4A77D', '#7A9E7E', '#B8860B', '#9FA8B3', '#8B7355'];
		const color = colors[Math.floor(Math.random() * colors.length)];

		pulses = [...pulses, {
			id: pulseIdCounter++,
			from: [...from] as [number, number, number],
			to: [...to] as [number, number, number],
			progress: 0,
			color,
			opacity: 0.8
		}];
	}

	// Animate pulses - move them along their path
	useTask((delta) => {
		if (pulses.length === 0) return;

		pulses = pulses
			.map(pulse => ({
				...pulse,
				progress: pulse.progress + delta * 0.4, // Slower, more graceful travel
				opacity: pulse.progress > 0.7 ? (1 - pulse.progress) * 2.5 : 0.7 // Fade out at end
			}))
			.filter(pulse => pulse.progress < 1); // Remove completed pulses
	});

	// Start spawning pulses periodically
	onMount(() => {
		// Spawn a pulse every 1.5-3.5 seconds randomly - gentle rhythm
		const scheduleNextPulse = () => {
			const delay = 1500 + Math.random() * 2000;
			pulseSpawnTimer = setTimeout(() => {
				spawnPulse();
				scheduleNextPulse();
			}, delay);
		};
		scheduleNextPulse();
		// Spawn one after a brief delay
		setTimeout(spawnPulse, 1000);
	});

	onDestroy(() => {
		if (pulseSpawnTimer) clearTimeout(pulseSpawnTimer);
	});

	// Calculate pulse position along path
	function getPulsePosition(pulse: ElectronPulse): [number, number, number] {
		const t = pulse.progress;
		return [
			pulse.from[0] + (pulse.to[0] - pulse.from[0]) * t,
			pulse.from[1] + (pulse.to[1] - pulse.from[1]) * t,
			pulse.from[2] + (pulse.to[2] - pulse.from[2]) * t
		];
	}

	// Handle background click to deselect
	function handleBackgroundClick(e: Event) {
		// Only deselect if clicking the background, not a bubble
		if (selectedId) {
			onBackgroundClick?.();
		}
	}

	// Get connections for selected bubble
	let connections = $derived.by(() => {
		if (!selectedId) return [];

		const selectedNode = layoutResult.find(n => n.memory.id === selectedId);
		if (!selectedNode) return [];

		// Find related bubbles (same nodeId)
		const relatedNodes = layoutResult.filter(n => {
			if (n.memory.id === selectedId) return false;
			if (selectedNode.nodeId && n.nodeId === selectedNode.nodeId) return true;
			return false;
		});

		return relatedNodes.map(related => ({
			from: selectedNode.position,
			to: related.position,
			color: '#888888'
		}));
	});
</script>

<!-- Camera - position only set initially, then OrbitControls takes over -->
<T.PerspectiveCamera
	makeDefault
	position={cameraPosition}
	fov={50}
>
	<OrbitControls
		enableDamping
		dampingFactor={0.05}
		autoRotate={effectiveAutoRotate}
		autoRotateSpeed={0.3}
		minDistance={15}
		maxDistance={300}
		minPolarAngle={0.1}
		maxPolarAngle={Math.PI * 0.85}
		target={targetLookAt}
	/>
</T.PerspectiveCamera>

<!-- Invisible sphere surrounding scene for click-to-deselect -->
<!-- Using a large inverted sphere so clicks anywhere in empty space deselect -->
<T.Mesh
	onclick={handleBackgroundClick}
>
	<T.SphereGeometry args={[400, 32, 32]} />
	<T.MeshBasicMaterial visible={false} side={THREE.BackSide} />
</T.Mesh>

<!-- Lighting - soft ambient for Node Viewer floating room aesthetic -->
<T.AmbientLight intensity={1.1} />
<T.DirectionalLight position={[10, 50, 20]} intensity={0.6} color="#ffffff" />
<T.DirectionalLight position={[-15, 30, -10]} intensity={0.3} color="#ffffff" />
<T.DirectionalLight position={[0, -30, 0]} intensity={0.2} color="#e8e8e8" />

<!-- Background - white (gradient is handled by CSS container) -->
<T.Color args={['#ffffff']} attach="background" />

<!-- Connection lines (only when selected) - Node Viewer style thin lines -->
{#each connections as conn}
	{@const points = [
		new THREE.Vector3(...conn.from),
		new THREE.Vector3(...conn.to)
	]}
	{@const geometry = new THREE.BufferGeometry().setFromPoints(points)}
	<T.Line geometry={geometry}>
		<T.LineBasicMaterial color={conn.color} transparent opacity={0.4} linewidth={1} />
	</T.Line>
{/each}

<!-- Electron pulse animations - neural synapse effect -->
{#each pulses as pulse (pulse.id)}
	{@const pos = getPulsePosition(pulse)}
	{@const linePoints = [
		new THREE.Vector3(...pulse.from),
		new THREE.Vector3(...pulse.to)
	]}
	{@const lineGeometry = new THREE.BufferGeometry().setFromPoints(linePoints)}

	<!-- Connection line for pulse (fades with progress) -->
	<T.Line geometry={lineGeometry}>
		<T.LineBasicMaterial
			color={pulse.color}
			transparent
			opacity={pulse.opacity * 0.3}
			linewidth={1}
		/>
	</T.Line>

	<!-- Glowing electron particle -->
	<T.Mesh position={pos}>
		<T.SphereGeometry args={[1.2, 16, 16]} />
		<T.MeshBasicMaterial
			color={pulse.color}
			transparent
			opacity={pulse.opacity}
		/>
	</T.Mesh>

	<!-- Outer glow -->
	<T.Mesh position={pos}>
		<T.SphereGeometry args={[2.5, 12, 12]} />
		<T.MeshBasicMaterial
			color={pulse.color}
			transparent
			opacity={pulse.opacity * 0.3}
		/>
	</T.Mesh>
{/each}

<!-- Memory bubbles -->
{#each layoutResult as node (node.memory.id)}
	<MemoryBubble
		memory={node.memory}
		position={node.position}
		scale={node.scale}
		color={getTypeColor(node.memory.memory_type || 'context', node.memory.id)}
		isSelected={selectedId === node.memory.id}
		isHighlighted={isHighlighted(node.memory)}
		isDimmed={searchQuery !== '' && !isHighlighted(node.memory)}
		onclick={() => onBubbleClick(node.memory)}
		nodeInfo={node.nodeId ? nodeMap.get(node.nodeId) : undefined}
	/>
{/each}
