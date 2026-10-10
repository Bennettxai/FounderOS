<script lang="ts">
	interface Props {
		x: number;
		y: number;
		hasCustomIcon: boolean;
		onCustomize: () => void;
		onReset: () => void;
		onOpen: () => void;
	}

	let { x, y, hasCustomIcon, onCustomize, onReset, onOpen }: Props = $props();
</script>

<div
	class="context-menu"
	style="left: {x}px; top: {y}px;"
>
	<button class="context-menu-item" onclick={onCustomize}>
		<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
			<circle cx="12" cy="12" r="3"/>
			<path d="M12 1v6m0 6v10M1 12h6m6 0h10"/>
		</svg>
		Change Icon...
	</button>
	{#if hasCustomIcon}
		<button class="context-menu-item" onclick={onReset}>
			<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/>
				<path d="M3 3v5h5"/>
			</svg>
			Reset to Default
		</button>
	{/if}
	<div class="context-menu-divider"></div>
	<button class="context-menu-item" onclick={onOpen}>
		<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
			<path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/>
			<polyline points="15 3 21 3 21 9"/>
			<line x1="10" y1="14" x2="21" y2="3"/>
		</svg>
		Open
	</button>
</div>

<style>
	.context-menu {
		position: fixed;
		z-index: 9999;
		background: white;
		border-radius: 8px;
		box-shadow: 0 4px 20px rgba(0, 0, 0, 0.15), 0 0 0 1px rgba(0, 0, 0, 0.05);
		padding: 4px;
		min-width: 180px;
		animation: contextMenuIn 0.15s ease-out;
	}

	@keyframes contextMenuIn {
		from {
			opacity: 0;
			transform: scale(0.95);
		}
		to {
			opacity: 1;
			transform: scale(1);
		}
	}

	.context-menu-item {
		display: flex;
		align-items: center;
		gap: 10px;
		width: 100%;
		padding: 8px 12px;
		background: none;
		border: none;
		border-radius: 6px;
		font-size: 13px;
		color: #374151;
		cursor: pointer;
		text-align: left;
		transition: background 0.1s;
	}

	.context-menu-item:hover {
		background: #f3f4f6;
	}

	.context-menu-item svg {
		flex-shrink: 0;
		color: #6b7280;
	}

	.context-menu-divider {
		height: 1px;
		background: #e5e7eb;
		margin: 4px 8px;
	}
</style>
