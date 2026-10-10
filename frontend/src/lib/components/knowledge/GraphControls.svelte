<script lang="ts">
	interface Props {
		autoRotate?: boolean;
		viewMode?: 'graph' | 'list' | 'calendar';
		onZoomIn?: () => void;
		onZoomOut?: () => void;
		onReset?: () => void;
		onToggleRotate?: () => void;
		onViewModeChange?: (mode: 'graph' | 'list' | 'calendar') => void;
		onShowHistory?: () => void;
		onSearch?: () => void;
	}

	let {
		autoRotate = false,
		viewMode = 'graph',
		onZoomIn,
		onZoomOut,
		onReset,
		onToggleRotate,
		onViewModeChange,
		onShowHistory,
		onSearch
	}: Props = $props();
</script>

<div class="graph-controls">
	<!-- History -->
	<button
		class="btn-pill btn-pill-ghost control-btn"
		onclick={onShowHistory}
		title="Chat history"
	>
		<svg fill="none" stroke="currentColor" viewBox="0 0 24 24" width="18" height="18">
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
		</svg>
	</button>

	<div class="control-divider"></div>

	<!-- View modes -->
	<button
		class="btn-pill btn-pill-ghost control-btn"
		class:active={viewMode === 'graph'}
		onclick={() => onViewModeChange?.('graph')}
		title="Graph view"
	>
		<svg fill="none" stroke="currentColor" viewBox="0 0 24 24" width="18" height="18">
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
		</svg>
	</button>

	<button
		class="btn-pill btn-pill-ghost control-btn"
		class:active={viewMode === 'list'}
		onclick={() => onViewModeChange?.('list')}
		title="List view"
	>
		<svg fill="none" stroke="currentColor" viewBox="0 0 24 24" width="18" height="18">
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M4 6h16M4 10h16M4 14h16M4 18h16" />
		</svg>
	</button>

	<button
		class="btn-pill btn-pill-ghost control-btn"
		class:active={viewMode === 'calendar'}
		onclick={() => onViewModeChange?.('calendar')}
		title="Calendar view"
	>
		<svg fill="none" stroke="currentColor" viewBox="0 0 24 24" width="18" height="18">
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
		</svg>
	</button>

	<div class="control-divider"></div>

	<!-- Zoom controls -->
	<button
		class="btn-pill btn-pill-ghost control-btn"
		onclick={onZoomIn}
		title="Zoom in"
	>
		<svg fill="none" stroke="currentColor" viewBox="0 0 24 24" width="18" height="18">
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0zM10 7v6m3-3H7" />
		</svg>
	</button>

	<button
		class="btn-pill btn-pill-ghost control-btn"
		onclick={onZoomOut}
		title="Zoom out"
	>
		<svg fill="none" stroke="currentColor" viewBox="0 0 24 24" width="18" height="18">
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0zM13 10H7" />
		</svg>
	</button>

	<div class="control-divider"></div>

	<button
		class="btn-pill btn-pill-ghost control-btn"
		class:active={autoRotate}
		onclick={onToggleRotate}
		title={autoRotate ? 'Stop rotation' : 'Auto rotate'}
	>
		<svg fill="none" stroke="currentColor" viewBox="0 0 24 24" width="18" height="18">
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
		</svg>
	</button>

	<button
		class="btn-pill btn-pill-ghost control-btn"
		onclick={onReset}
		title="Reset view"
	>
		<svg fill="none" stroke="currentColor" viewBox="0 0 24 24" width="18" height="18">
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M4 8V4m0 0h4M4 4l5 5m11-1V4m0 0h-4m4 0l-5 5M4 16v4m0 0h4m-4 0l5-5m11 5l-5-5m5 5v-4m0 4h-4" />
		</svg>
	</button>

	<div class="control-divider"></div>

	<!-- Search -->
	<button
		class="btn-pill btn-pill-ghost control-btn"
		onclick={onSearch}
		title="Search"
	>
		<svg fill="none" stroke="currentColor" viewBox="0 0 24 24" width="18" height="18">
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
		</svg>
	</button>
</div>

<style>
	.graph-controls {
		position: absolute;
		top: 16px;
		right: 16px;
		display: flex;
		flex-direction: column;
		gap: 2px;
		padding: 8px;
		background: rgba(255, 255, 255, 0.9);
		backdrop-filter: blur(12px);
		border-radius: 14px;
		border: 1px solid rgba(0, 0, 0, 0.08);
		box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
	}

	.control-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 34px;
		height: 34px;
		padding: 0;
		border: none;
		background: transparent;
		color: #666;
		cursor: pointer;
		border-radius: 8px;
		transition: all 0.15s ease;
	}

	.control-btn:hover {
		background: rgba(0, 0, 0, 0.06);
		color: #333;
	}

	.control-btn.active {
		background: rgba(139, 115, 85, 0.15);
		color: #8B7355;
	}

	.control-divider {
		height: 1px;
		background: rgba(0, 0, 0, 0.08);
		margin: 4px 0;
	}
</style>
