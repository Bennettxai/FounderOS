<script lang="ts">
	import { T, useTask } from '@threlte/core';
	import { HTML } from '@threlte/extras';
	import { spring } from 'svelte/motion';
	import type { Memory } from '$lib/api/memory/types';

	interface Props {
		memory: Memory;
		position: [number, number, number];
		scale?: number;
		color?: string;
		isSelected?: boolean;
		isHighlighted?: boolean;
		isDimmed?: boolean;
		onclick?: () => void;
		nodeInfo?: { name: string; type: string }; // Parent node info
	}

	let {
		memory,
		position,
		scale = 1,
		color = '#3b82f6',
		isSelected = false,
		isHighlighted = true,
		isDimmed = false,
		onclick,
		nodeInfo
	}: Props = $props();

	// Use memory's custom color if available, otherwise use passed color
	let bubbleColor = $derived(memory.color || color);

	// Check if memory has a cover image
	let hasCoverImage = $derived(!!memory.cover_image);

	// Animation state
	let hovered = $state(false);
	let showTooltip = $state(false);

	// Spring animation for scale
	const animatedScale = spring(1, {
		stiffness: 0.3,
		damping: 0.6
	});

	// Update scale based on state
	$effect(() => {
		if (isSelected) {
			animatedScale.set(scale * 1.3);
		} else if (hovered) {
			animatedScale.set(scale * 1.15);
		} else {
			animatedScale.set(scale);
		}
	});

	// Base bubble radius
	const baseRadius = 3;

	// Calculate opacity based on state - more subtle for marble look
	let opacity = $derived(
		isDimmed ? 0.3 : (isHighlighted ? 0.95 : 0.7)
	);

	// Soft glow for selection - smaller, more subtle
	let glowScale = $derived(
		isSelected ? 1.15 : (hovered ? 1.08 : 1.0)
	);

	// Subtle float animation
	let floatOffset = $state(0);
	useTask((delta) => {
		floatOffset = Math.sin(Date.now() * 0.001 + position[0]) * 0.3;
	});

	// Handle interaction
	function handlePointerEnter() {
		hovered = true;
		showTooltip = true;
	}

	function handlePointerLeave() {
		hovered = false;
		showTooltip = false;
	}

	function handleClick(e: Event) {
		e.stopPropagation();
		onclick?.();
	}

	// Get type initial (for labeling)
	function getTypeInitial(type: string): string {
		const icons: Record<string, string> = {
			'fact': 'F',
			'preference': 'P',
			'decision': 'D',
			'event': 'E',
			'learning': 'L',
			'context': 'C',
			'relationship': 'R'
		};
		return icons[type] || '?';
	}

	// Get node type icon based on the Node architecture
	function getNodeIcon(type: string): string {
		const icons: Record<string, string> = {
			'entity': '🏢',
			'department': '🏛️',
			'team': '👥',
			'project': '📁',
			'operational': '🔧',
			'learning': '📚',
			'person': '👤',
			'product': '🛠️',
			'partnership': '🤝',
			'context': '📋'
		};
		return icons[type.toLowerCase()] || '📌';
	}

	// Get SVG icon path based on memory type (Node Viewer style)
	function getMemoryIcon(type: string): string {
		const icons: Record<string, string> = {
			'fact': 'M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z', // document
			'preference': 'M4.318 6.318a4.5 4.5 0 000 6.364L12 20.364l7.682-7.682a4.5 4.5 0 00-6.364-6.364L12 7.636l-1.318-1.318a4.5 4.5 0 00-6.364 0z', // heart
			'decision': 'M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z', // key
			'event': 'M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z', // calendar
			'learning': 'M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253', // book
			'context': 'M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4', // building
			'relationship': 'M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z' // people
		};
		return icons[type] || 'M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z'; // default document
	}
</script>

<!-- Main bubble group - Glass bubble like Node Viewer (no shadows) -->
<T.Group position={[position[0], position[1] + floatOffset, position[2]]}>
	<!-- Soft outer glow for selection/hover -->
	{#if isSelected || hovered}
		<T.Mesh scale={[$animatedScale * glowScale, $animatedScale * glowScale, $animatedScale * glowScale]}>
			<T.SphereGeometry args={[baseRadius * 1.15, 32, 32]} />
			<T.MeshBasicMaterial
				color="#ffffff"
				transparent
				opacity={0.3}
			/>
		</T.Mesh>
	{/if}

	<!-- Glass bubble sphere - Node Viewer marble style with soft transparency -->
	<T.Mesh
		scale={[$animatedScale, $animatedScale, $animatedScale]}
		onclick={handleClick}
		onpointerenter={handlePointerEnter}
		onpointerleave={handlePointerLeave}
	>
		<T.SphereGeometry args={[baseRadius, 64, 64]} />
		<T.MeshPhysicalMaterial
			color={bubbleColor}
			metalness={0.05}
			roughness={0.08}
			transmission={0.6}
			thickness={2.0}
			clearcoat={1}
			clearcoatRoughness={0.05}
			transparent
			opacity={opacity * 0.75}
			ior={1.45}
			reflectivity={0.5}
			envMapIntensity={0.8}
			sheen={0.3}
			sheenRoughness={0.2}
			sheenColor={bubbleColor}
		/>
	</T.Mesh>

	<!-- Icon or Cover Image inside bubble - Node Viewer style large centered image -->
	<HTML
		position={[0, 0, baseRadius * 0.5]}
		center
		transform
		occlude={false}
		pointerEvents="none"
		scale={$animatedScale * 1.2}
	>
		{#if hasCoverImage}
			<!-- Cover image display - larger for Node Viewer style -->
			<div class="bubble-image" style="opacity: {isDimmed ? 0.3 : 1};">
				<img src={memory.cover_image} alt={memory.title || 'Memory'} />
			</div>
		{:else}
			<!-- Default icon based on memory type -->
			<div class="bubble-icon" style="color: {bubbleColor}; opacity: {isDimmed ? 0.3 : 0.6};">
				<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
					<path d={getMemoryIcon(memory.memory_type || 'context')} />
				</svg>
			</div>
		{/if}
	</HTML>

	<!-- Primary highlight (top-left) - soft glass reflection -->
	<T.Mesh scale={[$animatedScale * 0.25, $animatedScale * 0.2, $animatedScale * 0.25]} position={[-1.2, 1.5, 1.8]}>
		<T.SphereGeometry args={[baseRadius, 16, 16]} />
		<T.MeshBasicMaterial
			color="#ffffff"
			transparent
			opacity={0.6}
		/>
	</T.Mesh>

	<!-- Secondary highlight (smaller, offset) -->
	<T.Mesh scale={[$animatedScale * 0.12, $animatedScale * 0.1, $animatedScale * 0.12]} position={[-0.3, 0.8, 2.2]}>
		<T.SphereGeometry args={[baseRadius, 12, 12]} />
		<T.MeshBasicMaterial
			color="#ffffff"
			transparent
			opacity={0.45}
		/>
	</T.Mesh>

	<!-- Bottom ambient reflection (subtle floor reflection) -->
	<T.Mesh scale={[$animatedScale * 0.6, $animatedScale * 0.15, $animatedScale * 0.6]} position={[0, -2.2, 0]}>
		<T.SphereGeometry args={[baseRadius, 16, 16]} />
		<T.MeshBasicMaterial
			color={bubbleColor}
			transparent
			opacity={0.15}
		/>
	</T.Mesh>

	<!-- Rim light effect (subtle edge glow) -->
	<T.Mesh scale={[$animatedScale * 1.03, $animatedScale * 1.03, $animatedScale * 1.03]}>
		<T.SphereGeometry args={[baseRadius, 32, 32]} />
		<T.MeshBasicMaterial
			color="#ffffff"
			transparent
			opacity={0.06}
			side={1}
		/>
	</T.Mesh>

	<!-- Persistent title label below bubble -->
	{#if !isDimmed}
		<HTML
			position={[0, -baseRadius * $animatedScale - 1.5, 0]}
			center
			pointerEvents="none"
		>
			<div class="bubble-title" style="opacity: {isSelected || hovered ? 1 : 0.7}">
				{(memory.title || memory.learning_summary || 'Memory').slice(0, 24)}{(memory.title || memory.learning_summary || '').length > 24 ? '...' : ''}
			</div>
		</HTML>
	{/if}

	<!-- HTML tooltip on hover -->
	{#if showTooltip && !isDimmed}
		<HTML
			position={[0, baseRadius * $animatedScale + 2, 0]}
			center
			pointerEvents="none"
		>
			<div class="bubble-tooltip">
				{#if nodeInfo}
					<div class="tooltip-node">
						<span class="node-icon">{getNodeIcon(nodeInfo.type)}</span>
						<span class="node-name">{nodeInfo.name}</span>
					</div>
				{/if}
				<div class="tooltip-type" style="background: {color}">
					{memory.memory_type || 'memory'}
				</div>
				<div class="tooltip-title">
					{memory.title || 'Untitled'}
				</div>
				{#if memory.summary}
					<div class="tooltip-summary">
						{memory.summary.slice(0, 100)}{memory.summary.length > 100 ? '...' : ''}
					</div>
				{/if}
				{#if memory.importance_score}
					<div class="tooltip-importance">
						Importance: {Math.round(memory.importance_score * 100)}%
					</div>
				{/if}
			</div>
		</HTML>
	{/if}
</T.Group>

<style>
	.bubble-tooltip {
		background: rgba(255, 255, 255, 0.95);
		backdrop-filter: blur(12px);
		border-radius: 12px;
		padding: 14px 18px;
		max-width: 260px;
		pointer-events: none;
		box-shadow: 0 4px 24px rgba(0, 0, 0, 0.12), 0 1px 3px rgba(0, 0, 0, 0.08);
		border: 1px solid rgba(0, 0, 0, 0.06);
	}

	.tooltip-type {
		display: inline-block;
		padding: 3px 10px;
		border-radius: 6px;
		font-size: 10px;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.5px;
		color: white;
		margin-bottom: 8px;
	}

	.tooltip-title {
		font-size: 14px;
		font-weight: 600;
		color: #333;
		line-height: 1.3;
		margin-bottom: 6px;
	}

	.tooltip-summary {
		font-size: 12px;
		color: #666;
		line-height: 1.5;
	}

	.tooltip-node {
		display: flex;
		align-items: center;
		gap: 6px;
		padding: 5px 10px;
		background: rgba(122, 158, 126, 0.12);
		border-radius: 6px;
		margin-bottom: 10px;
		border: 1px solid rgba(122, 158, 126, 0.25);
	}

	.node-icon {
		font-size: 12px;
	}

	.node-name {
		font-size: 11px;
		font-weight: 500;
		color: #5a7a5e;
	}

	.tooltip-importance {
		font-size: 10px;
		color: #888;
		margin-top: 8px;
		padding-top: 8px;
		border-top: 1px solid rgba(0, 0, 0, 0.08);
	}

	.bubble-icon {
		width: 32px;
		height: 32px;
		display: flex;
		align-items: center;
		justify-content: center;
		filter: drop-shadow(0 1px 2px rgba(0, 0, 0, 0.15));
	}

	.bubble-icon svg {
		width: 100%;
		height: 100%;
	}

	.bubble-image {
		width: 40px;
		height: 40px;
		border-radius: 50%;
		overflow: hidden;
		display: flex;
		align-items: center;
		justify-content: center;
		filter: drop-shadow(0 2px 4px rgba(0, 0, 0, 0.2));
	}

	.bubble-image img {
		width: 100%;
		height: 100%;
		object-fit: cover;
		border-radius: 50%;
	}

	.bubble-title {
		font-size: 10px;
		font-weight: 500;
		color: #555;
		text-align: center;
		white-space: nowrap;
		text-shadow: 0 1px 2px rgba(255, 255, 255, 0.8);
		transition: opacity 0.2s ease;
	}
</style>
