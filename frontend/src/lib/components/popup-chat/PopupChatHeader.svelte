<script lang="ts">
	type PopupSize = 'small' | 'medium' | 'large' | 'full';

	interface Props {
		isMeetingMode: boolean;
		currentSize: PopupSize;
		onSetSize: (size: PopupSize) => void;
		onOpenMainWindow: () => void;
		onHidePopup: () => void;
	}

	let {
		isMeetingMode,
		currentSize,
		onSetSize,
		onOpenMainWindow,
		onHidePopup,
	}: Props = $props();
</script>

<div class="popup-header">
	<div class="header-drag-region"></div>
	<div class="header-content">
		<div class="header-title">
			{#if isMeetingMode}
				<span class="recording-indicator"></span>
				<span>Meeting Mode</span>
			{:else}
				<svg class="header-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
				</svg>
				<span>Quick Chat</span>
			{/if}
		</div>
		<div class="header-actions">
			<!-- Size toggle -->
			<div class="size-toggle">
				<button
					class="size-btn"
					class:active={currentSize === 'small'}
					onclick={() => onSetSize('small')}
					title="Small"
				>S</button>
				<button
					class="size-btn"
					class:active={currentSize === 'medium'}
					onclick={() => onSetSize('medium')}
					title="Medium"
				>M</button>
				<button
					class="size-btn"
					class:active={currentSize === 'large'}
					onclick={() => onSetSize('large')}
					title="Large"
				>L</button>
			</div>
			<button class="header-btn" onclick={onOpenMainWindow} title="Open full app (Cmd+Enter)" aria-label="Open full app">
				<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/>
					<polyline points="15 3 21 3 21 9"/>
					<line x1="10" y1="14" x2="21" y2="3"/>
				</svg>
			</button>
			<button class="header-btn" onclick={onHidePopup} title="Close (Esc)" aria-label="Close popup">
				<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<line x1="18" y1="6" x2="6" y2="18"/>
					<line x1="6" y1="6" x2="18" y2="18"/>
				</svg>
			</button>
		</div>
	</div>
</div>

<style>
	.popup-header {
		position: relative;
		background: rgba(249, 250, 251, 0.9);
		border-bottom: 1px solid rgba(0, 0, 0, 0.08);
	}

	.header-drag-region {
		position: absolute;
		top: 0;
		left: 0;
		right: 0;
		height: 32px;
		-webkit-app-region: drag;
	}

	.header-content {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 12px 16px;
		-webkit-app-region: no-drag;
	}

	.header-title {
		display: flex;
		align-items: center;
		gap: 8px;
		font-weight: 600;
		font-size: 14px;
		color: #111;
	}

	.header-icon {
		width: 18px;
		height: 18px;
	}

	.recording-indicator {
		width: 8px;
		height: 8px;
		background: #ef4444;
		border-radius: 50%;
		animation: pulse 1.5s infinite;
	}

	@keyframes pulse {
		0%, 100% { opacity: 1; }
		50% { opacity: 0.5; }
	}

	.header-actions {
		display: flex;
		gap: 4px;
	}

	.header-btn {
		width: 28px;
		height: 28px;
		border: none;
		background: none;
		border-radius: 6px;
		cursor: pointer;
		display: flex;
		align-items: center;
		justify-content: center;
		color: #666;
		transition: all 0.15s;
	}

	.header-btn:hover {
		background: rgba(0, 0, 0, 0.08);
		color: #111;
	}

	.header-btn svg {
		width: 16px;
		height: 16px;
	}

	.size-toggle {
		display: flex;
		background: rgba(0, 0, 0, 0.05);
		border-radius: 6px;
		padding: 2px;
		gap: 1px;
	}

	.size-btn {
		width: 22px;
		height: 22px;
		border: none;
		background: none;
		border-radius: 4px;
		cursor: pointer;
		font-size: 10px;
		font-weight: 600;
		color: #999;
		transition: all 0.15s;
	}

	.size-btn:hover {
		color: #666;
		background: rgba(0, 0, 0, 0.05);
	}

	.size-btn.active {
		background: white;
		color: #111;
		box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
	}

	/* Dark mode */
	:global(.dark) .popup-header {
		background: rgba(44, 44, 46, 0.9);
		border-bottom-color: rgba(255, 255, 255, 0.08);
	}

	:global(.dark) .header-title {
		color: #f5f5f7;
	}

	:global(.dark) .header-btn {
		color: #a1a1a6;
	}

	:global(.dark) .header-btn:hover {
		background: rgba(255, 255, 255, 0.1);
		color: #f5f5f7;
	}

	:global(.dark) .size-toggle {
		background: rgba(255, 255, 255, 0.08);
	}

	:global(.dark) .size-btn {
		color: #6e6e73;
	}

	:global(.dark) .size-btn:hover {
		color: #a1a1a6;
		background: rgba(255, 255, 255, 0.08);
	}

	:global(.dark) .size-btn.active {
		background: #3a3a3c;
		color: #f5f5f7;
		box-shadow: 0 1px 2px rgba(0, 0, 0, 0.3);
	}
</style>
