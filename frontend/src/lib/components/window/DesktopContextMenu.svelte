<script lang="ts">
	import type { DesktopIcon } from '$lib/stores/windowStore';

	interface Props {
		x: number;
		y: number;
		type: 'desktop' | 'icon';
		icon: DesktopIcon | null | undefined;
		onClose: () => void;
		onOpenIcon: () => void;
		onRenameIcon: () => void;
		onPinToDock: () => void;
		onDeleteFolder: () => void;
		onCreateNewFolder: () => void;
		onArrangeIcons: () => void;
		onOpenApps: () => void;
		onOpenTerminal: () => void;
		onCreateNote: () => void;
		onFitCanvas: () => void;
		onResetCanvas: () => void;
		onOpenDesktopSettings: () => void;
		isCanvasDesktop?: boolean;
	}

	let {
		x,
		y,
		type,
		icon,
		onClose,
		onOpenIcon,
		onRenameIcon,
		onPinToDock,
		onDeleteFolder,
		onCreateNewFolder,
		onArrangeIcons,
		onOpenApps,
		onOpenTerminal,
		onCreateNote,
		onFitCanvas,
		onResetCanvas,
		onOpenDesktopSettings,
		isCanvasDesktop = false
	}: Props = $props();
</script>

<div
	class="context-menu-overlay"
	onclick={onClose}
	oncontextmenu={(e) => { e.preventDefault(); onClose(); }}
	role="presentation"
></div>
<div
	class="context-menu"
	style="left: {x}px; top: {y}px;"
>
	{#if type === 'icon' && icon}
		<!-- Icon Context Menu -->
		<button class="context-menu-item" onclick={onOpenIcon}>
			<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
				<path d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/>
				<path d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"/>
			</svg>
			Open
		</button>
		{#if icon.type === 'folder'}
			<button class="context-menu-item" onclick={onRenameIcon}>
				<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
					<path d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"/>
				</svg>
				Rename
			</button>
		{/if}
		<div class="context-menu-separator"></div>
		<button class="context-menu-item" onclick={onPinToDock}>
			<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
				<path d="M5 5a2 2 0 012-2h10a2 2 0 012 2v16l-7-3.5L5 21V5z"/>
			</svg>
			Add to Dock
		</button>
		{#if icon.type === 'folder'}
			<div class="context-menu-separator"></div>
			<button class="context-menu-item context-menu-item--danger" onclick={onDeleteFolder}>
				<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
					<path d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/>
				</svg>
				Delete Folder
			</button>
		{/if}
	{:else}
		<!-- Desktop Context Menu -->
		<button class="context-menu-item" onclick={onCreateNewFolder}>
			<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
				<path d="M3 7V17C3 18.1046 3.89543 19 5 19H19C20.1046 19 21 18.1046 21 17V9C21 7.89543 20.1046 7 19 7H12L10 5H5C3.89543 5 3 5.89543 3 7Z"/>
			</svg>
			New Folder
		</button>
		<button class="context-menu-item" onclick={onOpenApps}>
			<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
				<rect x="3" y="3" width="7" height="7" rx="1"/>
				<rect x="14" y="3" width="7" height="7" rx="1"/>
				<rect x="3" y="14" width="7" height="7" rx="1"/>
				<rect x="14" y="14" width="7" height="7" rx="1"/>
			</svg>
			Open Apps
		</button>
		<button class="context-menu-item" onclick={onOpenTerminal}>
			<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
				<path d="M4 17l6-6-6-6"/>
				<path d="M12 19h8"/>
			</svg>
			Open Terminal
		</button>
		<div class="context-menu-separator"></div>
		{#if isCanvasDesktop}
			<button class="context-menu-item" onclick={onCreateNote}>
				<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
					<path d="M4 4h16v16H4z"/>
					<path d="M8 8h8M8 12h6"/>
				</svg>
				New Note
			</button>
			<button class="context-menu-item" onclick={onFitCanvas}>
				<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
					<path d="M4 9V4h5M20 9V4h-5M4 15v5h5M20 15v5h-5"/>
				</svg>
				Fit Canvas
			</button>
			<button class="context-menu-item" onclick={onResetCanvas}>
				<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
					<path d="M3 12a9 9 0 109-9"/>
					<path d="M3 4v6h6"/>
				</svg>
				Reset View
			</button>
			<div class="context-menu-separator"></div>
		{/if}
		<button class="context-menu-item" onclick={onArrangeIcons}>
			<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
				<path d="M4 6h4M4 10h4M4 14h4M4 18h4M10 6h10M10 10h10M10 14h10M10 18h10"/>
			</svg>
			Arrange Icons
		</button>
		<button class="context-menu-item" onclick={onOpenDesktopSettings}>
			<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
				<path d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"/>
				<circle cx="12" cy="12" r="3"/>
			</svg>
			Desktop Settings
		</button>
	{/if}
</div>

<style>
	.context-menu-overlay {
		position: fixed;
		inset: 0;
		z-index: 9998;
	}

	.context-menu {
		position: fixed;
		z-index: 9999;
		background: rgba(255, 255, 255, 0.95);
		backdrop-filter: blur(20px);
		-webkit-backdrop-filter: blur(20px);
		border-radius: 8px;
		box-shadow: 0 4px 20px rgba(0, 0, 0, 0.15), 0 0 0 1px rgba(0, 0, 0, 0.05);
		padding: 4px;
		min-width: 180px;
	}

	.context-menu-item {
		display: flex;
		align-items: center;
		gap: 10px;
		width: 100%;
		padding: 8px 12px;
		border: none;
		background: none;
		cursor: pointer;
		font-size: 13px;
		color: #333;
		border-radius: 4px;
		text-align: left;
	}

	.context-menu-item:hover {
		background: rgba(0, 102, 255, 0.1);
		color: #0066FF;
	}

	.context-menu-item svg {
		width: 16px;
		height: 16px;
		flex-shrink: 0;
	}

	.context-menu-separator {
		height: 1px;
		background: rgba(0, 0, 0, 0.1);
		margin: 4px 8px;
	}

	.context-menu-item--danger {
		color: #DC2626;
	}

	.context-menu-item--danger:hover {
		background: rgba(220, 38, 38, 0.1);
		color: #B91C1C;
	}

	/* ===== DARK MODE ===== */
	:global(.dark) .context-menu {
		background: rgba(44, 44, 46, 0.95);
		box-shadow: 0 4px 20px rgba(0, 0, 0, 0.4), 0 0 0 1px rgba(255, 255, 255, 0.08);
	}

	:global(.dark) .context-menu-item {
		color: #f5f5f7;
	}

	:global(.dark) .context-menu-item:hover {
		background: rgba(10, 132, 255, 0.2);
		color: #0A84FF;
	}

	:global(.dark) .context-menu-separator {
		background: rgba(255, 255, 255, 0.1);
	}

	:global(.dark) .context-menu-item--danger {
		color: #FF453A;
	}

	:global(.dark) .context-menu-item--danger:hover {
		background: rgba(255, 69, 58, 0.15);
		color: #FF453A;
	}
</style>
