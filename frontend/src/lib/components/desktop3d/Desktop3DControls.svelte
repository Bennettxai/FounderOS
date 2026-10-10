<script lang="ts">
	import type { ViewMode } from '$lib/stores/desktop3dStore';

	interface Props {
		viewMode: ViewMode;
		autoRotate: boolean;
		hasFocusedWindow: boolean;
		onToggleView?: () => void;
		onToggleAutoRotate?: () => void;
		onExit?: () => void;
	}

	let {
		viewMode = 'orb',
		autoRotate = true,
		hasFocusedWindow = false,
		onToggleView,
		onToggleAutoRotate,
		onExit
	}: Props = $props();

	let showHelp = $state(false);
</script>

<div class="controls-overlay">
	<!-- Top Left: Exit Button -->
	<div class="controls-top-left">
		<button class="btn-pill btn-pill-ghost control-btn exit-btn" onclick={onExit} title="Exit 3D Desktop (Esc)">
			<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
			</svg>
			<span>Exit</span>
		</button>
	</div>

	<!-- Top Right: View Controls -->
	<div class="controls-top-right">
		<!-- Auto-Rotate Toggle -->
		<button
			class="btn-pill btn-pill-ghost control-btn"
			class:active={autoRotate}
			onclick={onToggleAutoRotate}
			title="Toggle Auto-Rotate"
		>
			<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
			</svg>
		</button>

		<!-- View Mode Toggle -->
		<button
			class="btn-pill btn-pill-ghost control-btn view-toggle"
			onclick={onToggleView}
			disabled={hasFocusedWindow}
			title={viewMode === 'orb' ? 'Spread to Grid (Space)' : 'Collapse to Orb (Space)'}
		>
			{#if viewMode === 'orb'}
				<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2V6zM14 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V6zM4 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2v-2zM14 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z" />
				</svg>
				<span>Grid</span>
			{:else}
				<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 10a1 1 0 011-1h4a1 1 0 011 1v4a1 1 0 01-1 1h-4a1 1 0 01-1-1v-4z" />
				</svg>
				<span>Orb</span>
			{/if}
		</button>
	</div>

	<!-- Help Button (replaced bottom instructions) -->
	<div class="help-button-container">
		<button class="btn-pill btn-pill-ghost help-btn" title="Keyboard Shortcuts" onclick={() => showHelp = !showHelp}>
			<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8.228 9c.549-1.165 2.03-2 3.772-2 2.21 0 4 1.343 4 3 0 1.4-1.278 2.575-3.006 2.907-.542.104-.994.54-.994 1.093m0 3h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
			</svg>
		</button>

		{#if showHelp}
			<div class="help-tooltip">
				<div class="help-header">Keyboard Shortcuts</div>
				<div class="help-content">
					<div class="help-item"><kbd>Drag</kbd> Rotate view</div>
					<div class="help-item"><kbd>Scroll</kbd> Zoom</div>
					<div class="help-item"><kbd>Click</kbd> Focus window</div>
					<div class="help-item"><kbd>Space</kbd> Toggle view</div>
					<div class="help-item"><kbd>←/→</kbd> Navigate windows</div>
					<div class="help-item"><kbd>Esc</kbd> Exit / Unfocus</div>
				</div>
			</div>
		{/if}
	</div>
</div>

<style>
	.controls-overlay {
		position: fixed;
		inset: 0;
		pointer-events: none;
		z-index: 50;
	}

	.controls-top-left {
		position: absolute;
		top: 60px; /* Below MenuBar */
		left: 20px;
		display: flex;
		gap: 10px;
		pointer-events: auto;
	}

	.controls-top-right {
		position: absolute;
		top: 60px; /* Below MenuBar */
		right: 20px;
		display: flex;
		gap: 10px;
		pointer-events: auto;
	}

	.help-button-container {
		position: absolute;
		top: 60px;
		left: 140px; /* Next to exit button */
		pointer-events: auto;
	}

	.help-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 10px;
		background: rgba(255, 255, 255, 0.85);
		backdrop-filter: blur(12px);
		border: 1px solid rgba(0, 0, 0, 0.08);
		border-radius: 10px;
		color: var(--dt2);
		cursor: pointer;
		transition: all 0.2s;
		box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
	}

	.help-btn:hover {
		background: rgba(255, 255, 255, 0.95);
		border-color: rgba(0, 0, 0, 0.12);
		color: var(--dt);
		box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
	}

	.help-tooltip {
		position: absolute;
		top: 55px;
		right: 0;
		min-width: 250px;
		background: rgba(255, 255, 255, 0.95);
		backdrop-filter: blur(16px);
		border: 1px solid rgba(0, 0, 0, 0.08);
		border-radius: 12px;
		box-shadow: 0 8px 24px rgba(0, 0, 0, 0.12);
		overflow: hidden;
		z-index: 100;
	}

	.help-header {
		padding: 12px 16px;
		background: rgba(0, 0, 0, 0.03);
		border-bottom: 1px solid rgba(0, 0, 0, 0.06);
		font-weight: 600;
		font-size: 13px;
		color: var(--dt);
	}

	.help-content {
		padding: 8px;
	}

	.help-item {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 8px 10px;
		font-size: 13px;
		color: var(--dt2);
		border-radius: 6px;
		transition: background 0.15s;
	}

	.help-item:hover {
		background: rgba(0, 0, 0, 0.03);
	}

	.help-item kbd {
		min-width: 60px;
		padding: 4px 8px;
		background: rgba(0, 0, 0, 0.05);
		border: 1px solid rgba(0, 0, 0, 0.1);
		border-radius: 4px;
		font-family: inherit;
		font-size: 11px;
		font-weight: 500;
		color: var(--dt);
		text-align: center;
	}

	.control-btn {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 10px 16px;
		background: rgba(255, 255, 255, 0.85);
		backdrop-filter: blur(12px);
		border: 1px solid rgba(0, 0, 0, 0.08);
		border-radius: 10px;
		color: var(--dt);
		font-size: 14px;
		font-weight: 500;
		cursor: pointer;
		transition: all 0.2s;
		box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
	}

	.control-btn:hover {
		background: rgba(255, 255, 255, 0.95);
		border-color: rgba(0, 0, 0, 0.12);
		box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
	}

	.control-btn.active {
		background: rgba(74, 158, 255, 0.15);
		border-color: rgba(74, 158, 255, 0.3);
		color: var(--bos-status-info);
	}

	.control-btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.exit-btn {
		background: rgba(255, 240, 240, 0.9);
		border-color: rgba(200, 100, 100, 0.2);
		color: var(--bos-status-error);
	}

	.exit-btn:hover {
		background: rgba(255, 230, 230, 0.95);
		border-color: rgba(200, 100, 100, 0.3);
	}

	.voice-btn.active {
		background: rgba(59, 130, 246, 0.15);
		border-color: rgba(59, 130, 246, 0.3);
		color: var(--bos-status-info);
		animation: voicePulse 1.5s ease-in-out infinite;
	}

	@keyframes voicePulse {
		0%, 100% {
			box-shadow: 0 2px 8px rgba(59, 130, 246, 0.2);
		}
		50% {
			box-shadow: 0 4px 16px rgba(59, 130, 246, 0.4);
		}
	}

	/* ===== DARK MODE STYLES ===== */
	:global(.dark) .control-btn {
		background: rgba(44, 44, 46, 0.85);
		border-color: rgba(255, 255, 255, 0.12);
		color: var(--dt);
		box-shadow: 0 2px 8px rgba(0, 0, 0, 0.3);
	}

	:global(.dark) .control-btn:hover {
		background: rgba(58, 58, 60, 0.95);
		border-color: rgba(255, 255, 255, 0.18);
		box-shadow: 0 4px 12px rgba(0, 0, 0, 0.4);
	}

	:global(.dark) .control-btn.active {
		background: rgba(74, 158, 255, 0.25);
		border-color: rgba(74, 158, 255, 0.5);
		color: var(--bos-status-info);
	}

	:global(.dark) .control-btn svg {
		stroke: currentColor;
	}

	:global(.dark) .control-btn.active svg {
		stroke: var(--bos-status-info);
	}

	:global(.dark) .exit-btn {
		background: rgba(80, 30, 30, 0.85);
		border-color: rgba(255, 100, 100, 0.3);
		color: var(--bos-status-error);
	}

	:global(.dark) .exit-btn:hover {
		background: rgba(100, 40, 40, 0.95);
		border-color: rgba(255, 100, 100, 0.5);
	}

	:global(.dark) .exit-btn svg {
		stroke: var(--bos-status-error);
	}

	:global(.dark) .voice-btn.active {
		background: rgba(59, 130, 246, 0.25);
		border-color: rgba(59, 130, 246, 0.5);
		color: var(--bos-status-info);
	}

	:global(.dark) .voice-btn.active svg {
		stroke: var(--bos-status-info);
	}

	:global(.dark) .help-btn {
		background: rgba(44, 44, 46, 0.85);
		border-color: rgba(255, 255, 255, 0.12);
		color: var(--dt3);
	}

	:global(.dark) .help-btn:hover {
		background: rgba(58, 58, 60, 0.95);
		border-color: rgba(255, 255, 255, 0.18);
		color: var(--dt);
	}

	:global(.dark) .help-tooltip {
		background: rgba(30, 30, 30, 0.95);
		border-color: rgba(255, 255, 255, 0.12);
		box-shadow: 0 8px 24px rgba(0, 0, 0, 0.6);
	}

	:global(.dark) .help-header {
		background: rgba(255, 255, 255, 0.05);
		border-color: rgba(255, 255, 255, 0.08);
		color: var(--dt);
	}

	:global(.dark) .help-item {
		color: var(--dt3);
	}

	:global(.dark) .help-item:hover {
		background: rgba(255, 255, 255, 0.05);
	}

	:global(.dark) .help-item kbd {
		background: rgba(255, 255, 255, 0.1);
		border-color: rgba(255, 255, 255, 0.15);
		color: var(--dt);
	}
</style>
