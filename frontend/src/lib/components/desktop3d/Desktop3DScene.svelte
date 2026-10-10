<script lang="ts">
	import { T } from '@threlte/core';
	import { OrbitControls, interactivity } from '@threlte/extras';
	import * as THREE from 'three';
	import Desktop3DWindow from './Desktop3DWindow.svelte';
	import type { Window3DState, ViewMode } from '$lib/stores/desktop3dStore';

	// Enable pointer events on 3D objects
	interactivity();

	interface Props {
		windows: Window3DState[];
		viewMode: ViewMode;
		focusedWindowId: string | null;
		autoRotate: boolean;
		sphereRadius: number;
		cameraDistance: number; // Camera distance for zoom control
		cameraRotationDelta?: { x: number; y: number }; // NEW: Gesture rotation delta
		gestureDragging?: boolean; // NEW: Is gesture dragging active
		orbitControlsRef?: any; // Bindable ref to OrbitControls for gesture control
		onWindowClick?: (id: string) => void;
		onBackgroundClick?: () => void;
		onResize?: (widthDelta: number, heightDelta: number) => void;
		onZoomOut?: () => void; // Called when user zooms out in focus mode
	}

	let {
		windows = [],
		viewMode = 'orb',
		focusedWindowId = null,
		autoRotate = true,
		sphereRadius = 60,
		cameraDistance = 400,
		cameraRotationDelta = { x: 0, y: 0 }, // NEW: Default rotation delta
		gestureDragging = false, // NEW: Default not dragging
		orbitControlsRef = $bindable(), // BINDABLE: Expose OrbitControls ref to parent
		onWindowClick,
		onBackgroundClick,
		onResize,
		onZoomOut
	}: Props = $props();

	// OrbitControls ref is now exposed as bindable prop (line 39)
	// No need for local variable - use orbitControlsRef directly

	// Track which window is hovered (only ONE at a time)
	let hoveredWindowId: string | null = $state(null);

	// Handle zoom out while focused - exit focus mode
	// Only trigger on significant zoom out, not rotation
	function handleControlsChange(e: any) {
		if (!focusedWindowId || !orbitControlsRef) return;

		// Get current camera distance from target
		const controls = orbitControlsRef;
		if (controls?.object) {
			const distance = controls.object.position.distanceTo(controls.target);
			// Only unfocus if zoomed WAY out (past the orb view distance)
			if (distance > 600) {
				onZoomOut?.();
			}
		}
	}
	// NO camera reset - keep free-form rotation at all times

	// Camera position - INITIAL ONLY (OrbitControls manages it after)
	// Don't make this reactive or it will fight with gesture control!
	let initialCameraPosition: [number, number, number] = [0, 40, cameraDistance];

	// Effective auto-rotate (disabled when focused)
	let effectiveAutoRotate = $derived(autoRotate && !focusedWindowId);

	// Reset camera to front view when focusing a window
	$effect(() => {
		if (focusedWindowId && orbitControlsRef) {
			// Smoothly move camera to face the focused window
			const controls = orbitControlsRef;
			if (controls?.object) {
				// Reset camera to front position using current cameraDistance
				controls.object.position.set(0, 40, cameraDistance);
				controls.target.set(0, 0, 0);
				controls.update();
			}
		}
	});

	// NEW: Apply gesture rotation delta to camera
	$effect(() => {
		if (!orbitControlsRef) return;

		const controls = orbitControlsRef;
		const hasRotation = cameraRotationDelta && (Math.abs(cameraRotationDelta.x) > 0.001 || Math.abs(cameraRotationDelta.y) > 0.001);

		if (gestureDragging && hasRotation && controls) {
			// Get current spherical coordinates
			const offset = new THREE.Vector3();
			offset.copy(controls.object.position).sub(controls.target);

			// Convert to spherical
			const spherical = new THREE.Spherical();
			spherical.setFromVector3(offset);

			// Apply rotation deltas with higher sensitivity
			// X delta = azimuthal angle (horizontal rotation)
			// Y delta = polar angle (vertical rotation)
			spherical.theta -= cameraRotationDelta.x * 1.0; // Increased from 0.5
			spherical.phi -= cameraRotationDelta.y * 1.0; // Increased from 0.5

			// Clamp polar angle to prevent flipping
			spherical.phi = Math.max(0.1, Math.min(Math.PI - 0.1, spherical.phi));

			// Convert back to cartesian
			offset.setFromSpherical(spherical);

			// Update camera position
			controls.object.position.copy(controls.target).add(offset);
			controls.update();
		}
	});

	// Calculate indices for prev/next windows
	let focusedIndex = $derived(windows.findIndex(w => w.id === focusedWindowId));
	let prevIndex = $derived(focusedIndex > 0 ? focusedIndex - 1 : windows.length - 1);
	let nextIndex = $derived(focusedIndex >= 0 && focusedIndex < windows.length - 1 ? focusedIndex + 1 : 0);

	// Handle background click
	function handleBackgroundClick() {
		onBackgroundClick?.();
	}

</script>

<!-- Camera - OrbitControls manages position (initial only, then OrbitControls takes over) -->
<T.PerspectiveCamera
	makeDefault
	position={initialCameraPosition}
	fov={50}
>
	<OrbitControls
		bind:ref={orbitControlsRef}
		enableDamping={true}
		dampingFactor={0.08}
		autoRotate={effectiveAutoRotate}
		autoRotateSpeed={0.3}
		minDistance={150}
		maxDistance={800}
		minPolarAngle={0.1}
		maxPolarAngle={Math.PI * 0.9}
		enablePan={false}
		enableZoom={true}
		enableRotate={true}
		onchange={handleControlsChange}
	/>
</T.PerspectiveCamera>

<!-- Lighting -->
<T.AmbientLight intensity={0.6} />
<T.DirectionalLight position={[50, 100, 50]} intensity={1} />
<T.DirectionalLight position={[-50, 50, -50]} intensity={0.5} />
<T.PointLight position={[0, 0, 0]} intensity={2} color="#4a9eff" distance={100} />

<!-- Background click catcher -->
<T.Mesh
	onclick={handleBackgroundClick}
>
	<T.SphereGeometry args={[500, 32, 32]} />
	<T.MeshBasicMaterial
		visible={false}
		side={THREE.BackSide}
	/>
</T.Mesh>

<!-- NO center orb - windows are the focus -->
<!-- NO connection lines -->

<!-- Module Windows -->
{#each windows as window, index (window.id)}
	<Desktop3DWindow
		{window}
		isFocused={window.id === focusedWindowId}
		isPrevWindow={focusedWindowId !== null && index === prevIndex}
		isNextWindow={focusedWindowId !== null && index === nextIndex}
		isHovered={window.id === hoveredWindowId}
		viewMode={viewMode}
		onClick={() => onWindowClick?.(window.id)}
		onResize={onResize}
		onHover={(hovered) => {
			if (hovered) {
				hoveredWindowId = window.id;
			} else if (hoveredWindowId === window.id) {
				hoveredWindowId = null;
			}
		}}
	/>
{/each}

<!-- Background handled by parent container gradient -->
