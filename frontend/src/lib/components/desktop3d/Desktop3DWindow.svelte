<script lang="ts">
	import { T } from '@threlte/core';
	import { HTML } from '@threlte/extras';
	import { spring } from 'svelte/motion';
	import { desktop3dStore, type Window3DState, type ViewMode } from '$lib/stores/desktop3dStore';

	interface Props {
		window: Window3DState;
		isFocused: boolean;
		isNextWindow: boolean;
		isPrevWindow: boolean;
		isHovered: boolean;
		viewMode: ViewMode;
		onClick?: () => void;
		onResize?: (widthDelta: number, heightDelta: number) => void;
		onHover?: (hovered: boolean) => void;
	}

	let {
		window,
		isFocused = false,
		isNextWindow = false,
		isPrevWindow = false,
		isHovered = false,
		viewMode = 'orb',
		onClick,
		onResize,
		onHover
	}: Props = $props();

	// Track pointer down position to distinguish click vs drag
	let pointerDownPos: { x: number; y: number } | null = null;
	const DRAG_THRESHOLD = 15; // pixels - increased for better click detection

	// Spring animation for smooth position transitions
	type Vec3 = [number, number, number];
	let animatedPosition = spring<Vec3>(window.position, {
		stiffness: 0.08,
		damping: 0.7
	});
	// Separate springs for Y rotation (facing) and X rotation (tilt)
	// This allows proper rotation order: Y first, then X
	let animatedYRotation = spring(0, {
		stiffness: 0.08,
		damping: 0.7
	});
	let animatedXRotation = spring(0, {
		stiffness: 0.08,
		damping: 0.7
	});
	let animatedScale = spring(1, {
		stiffness: 0.1,
		damping: 0.6
	});

	// Handle resize button clicks - call store DIRECTLY to bypass prop chain issues in 3D HTML
	function handleSizeIncrease(e: MouseEvent | Event) {
		e.stopPropagation();
		if ('stopImmediatePropagation' in e) e.stopImmediatePropagation();
		e.preventDefault();
		desktop3dStore.resizeFocusedWindow(100, 75);
	}
	function handleSizeDecrease(e: MouseEvent | Event) {
		e.stopPropagation();
		if ('stopImmediatePropagation' in e) e.stopImmediatePropagation();
		e.preventDefault();
		desktop3dStore.resizeFocusedWindow(-100, -75);
	}

	// Current opacity (not animated)
	// All windows always visible, just dimmed in focus mode
	let currentOpacity = $derived.by(() => {
		if (isFocused) return 1;
		if (isNextWindow || isPrevWindow) return 0.85; // Side previews clearly visible
		if (viewMode === 'focused') return 0.15; // Others very faded, orb in background
		return 1;
	});

	// Always render all windows
	let shouldRender = true;

	// Track if we're dragging to prevent click on drag end
	let isDragging = $state(false);
	let pointerDownTime = 0;

	// Handle pointer down - track start position
	function handlePointerDown(e: any) {
		isDragging = false;
		pointerDownTime = Date.now();
		// Store the pointer position when pressed
		const event = e.nativeEvent || e;
		if (event.clientX !== undefined) {
			pointerDownPos = { x: event.clientX, y: event.clientY };
		}
	}

	// Handle pointer move - detect if dragging
	function handlePointerMove(e: any) {
		if (!pointerDownPos) return;
		const event = e.nativeEvent || e;
		if (event.clientX !== undefined) {
			const dx = Math.abs(event.clientX - pointerDownPos.x);
			const dy = Math.abs(event.clientY - pointerDownPos.y);
			if (dx > DRAG_THRESHOLD || dy > DRAG_THRESHOLD) {
				isDragging = true;
			}
		}
	}

	// Handle click - only trigger if it wasn't a drag
	function handleClick(e: any) {
		const clickDuration = Date.now() - pointerDownTime;

		// Don't trigger if: was dragging, held too long, or no pointer down recorded
		if (isDragging || clickDuration > 500 || !pointerDownPos) {
			pointerDownPos = null;
			isDragging = false;
			return;
		}

		pointerDownPos = null;
		isDragging = false;
		e.stopPropagation();
		onClick?.();
	}

	// Calculate target position based on state
	function getTargetPosition(): Vec3 {
		if (isFocused) {
			// Focused window: always come to front center, facing camera
			return [0, 10, 200];
		}
		if (isPrevWindow) {
			// Previous window: to the left
			return [-200, 10, 150];
		}
		if (isNextWindow) {
			// Next window: to the right
			return [200, 10, 150];
		}
		// Other windows: stay in orb positions
		return window.position;
	}

	// Calculate Y rotation - face outward from sphere center
	// This is applied FIRST (outer group) so tilt works correctly
	function getTargetYRotation(): number {
		const [x, , z] = window.position;

		// When focused or adjacent, face the camera (no Y rotation)
		if (isFocused || isNextWindow || isPrevWindow) {
			return 0;
		}

		// Face outward from sphere center
		return Math.atan2(x, z);
	}

	// Calculate X rotation (tilt) - applied AFTER Y rotation
	// Top windows tilt UP (face upward), bottom windows tilt DOWN (face downward)
	function getTargetXRotation(): number {
		const [, y] = window.position;

		// When focused or adjacent, no tilt
		if (isFocused || isNextWindow || isPrevWindow) {
			return 0;
		}

		// Normalize Y position based on sphere radius
		const normalizedY = y / 95;

		// Top windows (positive Y) = negative tilt = face DOWNWARD ~45 degrees
		// Bottom windows (negative Y) = positive tilt = face UPWARD ~45 degrees
		// Middle windows (y near 0) = minimal tilt
		// 0.785 radians = 45 degrees
		return -normalizedY * 0.785;
	}

	// Calculate target scale
	function getTargetScale(): number {
		if (isFocused) return 2.2; // Large focused window
		if (isNextWindow || isPrevWindow) return 0.9; // Side previews visible
		if (viewMode === 'focused') return 0.5; // Background windows much smaller
		return 1;
	}

	// Update springs when state changes
	$effect(() => {
		const targetPos = getTargetPosition();
		animatedPosition.set(targetPos);
	});

	$effect(() => {
		const targetY = getTargetYRotation();
		animatedYRotation.set(targetY);
	});

	$effect(() => {
		const targetX = getTargetXRotation();
		animatedXRotation.set(targetX);
	});

	$effect(() => {
		const targetScale = getTargetScale();
		animatedScale.set(targetScale);
	});

	// HUGE scale for visible windows - BIGGER for better visibility
	const htmlScale = 2.0;

	// Route mapping for modules where route differs from module name
	function getModuleRoute(module: string): string {
		switch (module) {
			case 'pages':
				return '/knowledge';  // pages redirects to knowledge
			case 'contexts':
				return '/knowledge';  // contexts also redirects to knowledge
			case 'communication':
				return '/communication/calendar';  // communication redirects to calendar tab
			default:
				return `/${module}`;
		}
	}

	// Track iframe element for focus management
	let iframeElement = $state<HTMLIFrameElement | null>(null);

	// Track previous focus state to only focus on transition
	let wasFocused = $state(false);

	// CRITICAL: Focus iframe content when window BECOMES focused (transition from unfocused to focused)
	// This ensures keyboard input (arrow keys, Enter, etc.) reaches the iframe content
	// BUT only runs ONCE when focus changes, not continuously
	$effect(() => {
		// Only focus when transitioning from unfocused to focused
		if (isFocused && !wasFocused && iframeElement) {
			if (import.meta.env.DEV) console.log('[Window] Focusing iframe for window:', window.title);
			// Focus the iframe element itself
			iframeElement.focus();

			// Try to focus content inside iframe (works for same-origin iframes)
			try {
				iframeElement.contentWindow?.focus();
			} catch (e) {
				// Cross-origin iframe - can't access contentWindow
				if (import.meta.env.DEV) console.log('[Window] Cannot focus cross-origin iframe content');
			}
		}
		// Update previous state
		wasFocused = isFocused;
	});
</script>

<!-- Window using HTML component for DOM content in 3D space -->
<!-- Only render if should be visible -->
{#if shouldRender}
	<!-- Invisible click mesh - ONLY for non-focused, non-adjacent windows in orb mode -->
	{#if !isFocused && !isNextWindow && !isPrevWindow}
		<!-- Click mesh - positioned at window center, faces camera (no rotation) -->
		<T.Mesh
			position={$animatedPosition}
			onpointerdown={(e: { stopPropagation: () => void }) => { e.stopPropagation(); handlePointerDown(e); }}
			onpointermove={(e: unknown) => { handlePointerMove(e); }}
			onclick={(e: { stopPropagation: () => void }) => { e.stopPropagation(); handleClick(e); }}
			onpointerenter={(e: { stopPropagation: () => void }) => { e.stopPropagation(); onHover?.(true); }}
			onpointerleave={(e: { stopPropagation: () => void }) => { e.stopPropagation(); onHover?.(false); }}
		>
			<T.SphereGeometry args={[50, 12, 12]} />
			<T.MeshBasicMaterial visible={false} transparent opacity={0} />
		</T.Mesh>
	{:else if isFocused}
		<!-- Blocker mesh when focused - prevents background clicks -->
		<T.Mesh
			position={$animatedPosition}
			onclick={(e: { stopPropagation: () => void }) => e.stopPropagation()}
		>
			<T.PlaneGeometry args={[window.width * 0.6, window.height * 0.6]} />
			<T.MeshBasicMaterial visible={false} transparent opacity={0} />
		</T.Mesh>
	{/if}
	<!-- NOTE: No click mesh for prev/next windows - they use HTML pointer events -->

	<!-- Nested groups for proper rotation order: Y first (face outward), then X (tilt) -->
	<T.Group position={$animatedPosition}>
	<T.Group rotation={[0, $animatedYRotation, 0]}>
	<T.Group rotation={[$animatedXRotation, 0, 0]}>
	<HTML
		transform
		scale={htmlScale * $animatedScale}
		pointerEvents={isFocused || isNextWindow || isPrevWindow ? 'auto' : 'none'}
		zIndexRange={[100, 0]}
	>
		<!-- svelte-ignore a11y_no_static_element_interactions -->
		<div
			class="window-wrapper"
			class:focused={isFocused}
			class:dimmed={!isFocused && viewMode === 'focused'}
			class:clickable={isNextWindow || isPrevWindow}
			class:hovered={isHovered}
			style="opacity: {currentOpacity};"
			onclick={(isNextWindow || isPrevWindow) ? onClick : undefined}
		>
			<div
				class="window-3d"
				style="--window-width: {window.width}px; --window-height: {window.height}px;"
			>

				<!-- Title bar -->
				<div class="window-titlebar" style="border-left: 4px solid {window.color};">
					<span class="module-title">{window.title}</span>
					<div class="titlebar-right">
						{#if isFocused}
							<!-- Size controls when focused -->
							<!-- svelte-ignore a11y_no_static_element_interactions -->
							<!-- svelte-ignore a11y_click_events_have_key_events -->
							<div
								class="size-controls"
								onmousedown={(e) => e.stopPropagation()}
								onclick={(e) => e.stopPropagation()}
							>
								<button
									type="button"
									class="btn-pill btn-pill-ghost size-btn decrease-btn"
									onclick={(e) => { e.stopPropagation(); e.preventDefault(); handleSizeDecrease(e); }}
									onpointerdown={(e) => { e.stopPropagation(); e.preventDefault(); }}
									onmousedown={(e) => { e.stopPropagation(); e.preventDefault(); }}
									title="Click to make smaller"
								>
									<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3">
										<path d="M5 12h14" />
									</svg>
								</button>
								<span class="size-label">{window.width}x{window.height}</span>
								<button
									type="button"
									class="btn-pill btn-pill-ghost size-btn increase-btn"
									onclick={(e) => { e.stopPropagation(); e.preventDefault(); handleSizeIncrease(e); }}
									onpointerdown={(e) => { e.stopPropagation(); e.preventDefault(); }}
									onmousedown={(e) => { e.stopPropagation(); e.preventDefault(); }}
									title="Click to make larger"
								>
									<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3">
										<path d="M12 5v14M5 12h14" />
									</svg>
								</button>
							</div>
						{/if}
						<div class="titlebar-controls">
							<span class="control-dot" style="background: #febc2e;"></span>
							<span class="control-dot" style="background: #28c840;"></span>
							<span class="control-dot" style="background: #ff5f57;"></span>
						</div>
					</div>
				</div>

				<!-- LIVE Content - Always show iframe -->
				<div class="window-content">
					<iframe
						bind:this={iframeElement}
						src="{getModuleRoute(window.module)}?embed=true"
						title={window.title}
						class="window-iframe"
						sandbox="allow-same-origin allow-scripts allow-forms allow-popups allow-modals"
						loading="eager"
						tabindex="0"
					></iframe>
				</div>
			</div>

			<!-- Label under window (hidden when focused) -->
			{#if !isFocused}
				<div class="window-label" style="background-color: {window.color};">
					{window.title}
				</div>
			{/if}
		</div>
	</HTML>
	</T.Group>
	</T.Group>
</T.Group>
{/if}

<style>
	.window-wrapper {
		display: flex;
		flex-direction: column;
		align-items: center;
		cursor: pointer;
		backface-visibility: visible;
		-webkit-backface-visibility: visible;
	}

	.window-wrapper.dimmed {
		opacity: 0.5;
	}

	.window-wrapper.clickable {
		cursor: pointer;
	}

	.window-wrapper.clickable:hover {
		transform: scale(1.02);
	}

	/* Hover effect - shows which window you're about to click */
	.window-wrapper.hovered {
		transform: scale(1.08);
	}

	.window-wrapper.hovered .window-3d {
		box-shadow:
			0 0 0 4px rgba(74, 158, 255, 0.6),
			0 12px 48px rgba(74, 158, 255, 0.3),
			0 4px 12px rgba(0, 0, 0, 0.15);
	}

	.window-wrapper.hovered .window-label {
		transform: scale(1.1);
		box-shadow: 0 6px 20px rgba(0, 0, 0, 0.3);
	}

	.window-3d {
		position: relative;
		/* Width/height set dynamically via style attribute */
		width: var(--window-width, 700px);
		height: var(--window-height, 450px);
		background: var(--dbg);
		border-radius: 10px;
		overflow: hidden;
		box-shadow:
			0 8px 32px rgba(0, 0, 0, 0.15),
			0 2px 8px rgba(0, 0, 0, 0.1);
		display: flex;
		flex-direction: column;
		transition: box-shadow 0.3s ease, transform 0.3s ease;
		border: 1px solid rgba(0, 0, 0, 0.1);
		backface-visibility: visible;
		-webkit-backface-visibility: visible;
	}

	.window-label {
		margin-top: 12px;
		padding: 8px 20px;
		border-radius: 20px;
		color: white;
		font-size: 16px;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 1px;
		box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2);
	}

	.window-3d:hover {
		box-shadow:
			0 12px 48px rgba(0, 0, 0, 0.2),
			0 4px 12px rgba(0, 0, 0, 0.15);
	}

	.window-wrapper.focused .window-3d {
		cursor: default;
		box-shadow:
			0 20px 60px rgba(0, 0, 0, 0.25),
			0 8px 24px rgba(0, 0, 0, 0.2);
	}

	.window-titlebar {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 8px 12px;
		background: var(--dbg2);
		border-bottom: 1px solid var(--dbd);
		user-select: none;
		flex-shrink: 0;
		pointer-events: auto !important;
		position: relative;
		z-index: 10;
	}

	.module-title {
		font-size: 13px;
		font-weight: 600;
		color: var(--dt);
	}

	.titlebar-right {
		display: flex;
		align-items: center;
		gap: 12px;
	}

	.titlebar-controls {
		display: flex;
		gap: 6px;
	}

	.control-dot {
		width: 10px;
		height: 10px;
		border-radius: 50%;
	}

	/* Size controls in titlebar */
	.size-controls {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 4px 12px;
		background: rgba(74, 158, 255, 0.1);
		border: 1px solid rgba(74, 158, 255, 0.2);
		border-radius: 8px;
		position: relative;
		z-index: 10000;
		pointer-events: auto !important;
		touch-action: manipulation;
		user-select: none;
	}

	.size-btn {
		width: 28px;
		height: 28px;
		padding: 4px;
		background: rgba(74, 158, 255, 0.2);
		border: 1px solid rgba(74, 158, 255, 0.4);
		border-radius: 6px;
		cursor: pointer;
		display: flex;
		align-items: center;
		justify-content: center;
		transition: all 0.2s;
		pointer-events: auto !important;
		position: relative;
		z-index: 10000;
		touch-action: manipulation;
		user-select: none;
		-webkit-user-select: none;
	}

	.size-btn:hover {
		background: rgba(74, 158, 255, 0.4);
		border-color: rgba(74, 158, 255, 0.6);
		transform: scale(1.1);
	}

	.size-btn:active {
		background: rgba(74, 158, 255, 0.6);
		transform: scale(0.95);
	}

	.size-btn svg {
		width: 12px;
		height: 12px;
		stroke: var(--dt);
	}

	.size-label {
		font-size: 10px;
		color: var(--dt2);
		min-width: 60px;
		text-align: center;
	}

	.window-content {
		flex: 1;
		overflow: hidden;
		background: var(--dbg);
	}

	.window-iframe {
		width: 100%;
		height: 100%;
		border: none;
		pointer-events: none;
		outline: none;
	}

	.window-wrapper.focused .window-iframe {
		pointer-events: auto;
		/* Ensure iframe can receive keyboard events */
		user-select: auto;
	}

	/* Focus indicator for iframe */
	.window-iframe:focus {
		outline: none;
	}
</style>
