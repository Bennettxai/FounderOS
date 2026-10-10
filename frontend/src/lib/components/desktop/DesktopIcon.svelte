<script lang="ts">
	import { desktopSettings, type IconStyle, type IconLibrary } from '$lib/stores/desktopStore';
	import { windowStore, type CustomIconConfig } from '$lib/stores/windowStore';
	import { soundStore } from '$lib/stores/soundStore';
	import * as LucideIcons from 'lucide-svelte';
	import DesktopIconContextMenu from './DesktopIconContextMenu.svelte';
	import { iconPaths } from './iconPaths';

	interface Props {
		id: string;
		module: string;
		label: string;
		selected?: boolean;
		posX: number;
		posY: number;
		darkBackground?: boolean;
		iconType?: 'app' | 'folder';
		folderId?: string;
		folderColor?: string;
		customIcon?: CustomIconConfig;
		onSelect?: (id: string, additive: boolean) => void;
		onOpen?: (module: string) => void;
		onDragStart?: (id: string) => void;
		onDragMove?: (id: string, newX: number, newY: number) => void;
		onDragEnd?: (id: string, finalX: number, finalY: number) => void;
		onCustomizeIcon?: (id: string) => void;
		dragScale?: number;
	}

	let {
		id,
		module,
		label,
		selected = false,
		posX,
		posY,
		darkBackground = false,
		iconType = 'app',
		folderId,
		folderColor = '#3B82F6',
		customIcon,
		onSelect,
		onOpen,
		onDragStart,
		onDragMove,
		onDragEnd,
		onCustomizeIcon,
		dragScale = 1
	}: Props = $props();

	// Context menu state
	let showContextMenu = $state(false);
	let contextMenuX = $state(0);
	let contextMenuY = $state(0);

	// Get Lucide icon component by name
	function getLucideIcon(name: string): typeof import('lucide-svelte').Home | undefined {
		const icons = LucideIcons as unknown as Record<string, typeof import('lucide-svelte').Home>;
		return icons[name];
	}

	// Track if another icon is being dragged over this folder
	let isDragOver = $state(false);

	const iconStyle = $derived($desktopSettings.iconStyle);
	const iconSize = $derived($desktopSettings.iconSize);
	const showIconLabels = $derived($desktopSettings.showIconLabels);
	const iconLibrary = $derived($desktopSettings.iconLibrary);

	// Different libraries have EXTREMELY DRAMATIC different styles
	const libraryStrokeWidth = $derived({
		lucide: 2,        // Lucide - balanced, clean
		phosphor: 3,      // Phosphor - VERY bold, thick
		tabler: 1.2,      // Tabler - very thin, hairline
		heroicons: 2.5    // Heroicons - solid, medium-bold
	}[iconLibrary] || 2);

	// Some libraries have rounder vs sharper corners
	const libraryLineCap = $derived<'round' | 'square' | 'butt'>(
		iconLibrary === 'tabler' ? 'square' : iconLibrary === 'phosphor' ? 'round' : 'round'
	);

	const libraryLineJoin = $derived<'round' | 'miter' | 'bevel'>(
		iconLibrary === 'tabler' ? 'miter' : 'round'
	);

	// Icon scale varies EXTREMELY by library
	const libraryIconScale = $derived({
		lucide: 1,
		phosphor: 1.25,   // Phosphor icons 25% larger
		tabler: 0.85,     // Tabler icons 15% smaller
		heroicons: 1.15   // Heroicons 15% larger
	}[iconLibrary] || 1);

	// Different icon opacity per library
	const libraryOpacity = $derived({
		lucide: 1,        // Normal
		phosphor: 1,      // Full
		tabler: 0.7,      // More muted/faded
		heroicons: 1      // Full
	}[iconLibrary] || 1);

	// Different SVG filters per library for OBVIOUS visual differences
	const librarySvgFilter = $derived({
		lucide: 'none',
		phosphor: 'drop-shadow(0 2px 3px rgba(0,0,0,0.25))',    // Noticeable shadow
		tabler: 'saturate(0.7)',                                  // Desaturated look
		heroicons: 'drop-shadow(0 1px 2px rgba(0,0,0,0.2)) saturate(1.2)'  // Shadow + vivid
	}[iconLibrary] || 'none');

	// Calculate dimensions based on icon size - wider to accommodate labels
	const containerWidth = $derived(Math.max(iconSize + 36, 90));
	const imageSize = $derived(iconSize * 0.875); // Icon image is 87.5% of icon size
	const svgSize = $derived(iconSize * 0.4375); // SVG is about 50% of image
	const labelSize = $derived(Math.max(9, Math.min(13, iconSize * 0.17)));

	let clickCount = $state(0);
	let clickTimer: ReturnType<typeof setTimeout> | null = null;
	let isDragging = $state(false);
	let dragStartPos = { x: 0, y: 0 };
	let iconStartPos = { x: 0, y: 0 };
	let hasMoved = $state(false);

	function handleMouseDown(event: MouseEvent) {
		if (event.button !== 0) return; // Only left click
		event.preventDefault();

		dragStartPos = { x: event.clientX, y: event.clientY };
		iconStartPos = { x: posX, y: posY };
		hasMoved = false;

		// Start listening for drag
		document.addEventListener('mousemove', handleMouseMove);
		document.addEventListener('mouseup', handleMouseUp);
	}

	function handleMouseMove(event: MouseEvent) {
		const screenDeltaX = event.clientX - dragStartPos.x;
		const screenDeltaY = event.clientY - dragStartPos.y;
		const deltaX = screenDeltaX / dragScale;
		const deltaY = screenDeltaY / dragScale;

		// Only start dragging if moved more than 5px
		if (!isDragging && (Math.abs(screenDeltaX) > 5 || Math.abs(screenDeltaY) > 5)) {
			isDragging = true;
			hasMoved = true;
			onSelect?.(id, false); // Select when starting drag
			onDragStart?.(id);
		}

		if (isDragging) {
			onDragMove?.(id, iconStartPos.x + deltaX, iconStartPos.y + deltaY);
		}
	}

	function handleMouseUp(event: MouseEvent) {
		document.removeEventListener('mousemove', handleMouseMove);
		document.removeEventListener('mouseup', handleMouseUp);

		if (isDragging) {
			const deltaX = event.clientX - dragStartPos.x;
			const deltaY = event.clientY - dragStartPos.y;
			const finalX = iconStartPos.x + deltaX / dragScale;
			const finalY = iconStartPos.y + deltaY / dragScale;
			onDragEnd?.(id, finalX, finalY);
			isDragging = false;
		} else if (!hasMoved) {
			// Handle click
			handleClick(event);
		}
	}

	function handleClick(event: MouseEvent) {
		clickCount++;

		// Play click sound
		soundStore.playSound('click');

		if (clickCount === 1) {
			// Single click - select
			onSelect?.(id, event.metaKey || event.ctrlKey);
			clickTimer = setTimeout(() => {
				clickCount = 0;
			}, 300);
		} else if (clickCount === 2) {
			// Double click - open
			if (clickTimer) clearTimeout(clickTimer);
			clickCount = 0;
			if (iconType === 'folder' && folderId) {
				windowStore.openFolder(folderId);
			} else {
				onOpen?.(module);
			}
		}
	}

	// Context menu handlers
	function handleContextMenu(event: MouseEvent) {
		event.preventDefault();
		event.stopPropagation();

		// Position context menu
		contextMenuX = event.clientX;
		contextMenuY = event.clientY;
		showContextMenu = true;

		// Select this icon
		onSelect?.(id, false);

		// Close menu when clicking elsewhere
		document.addEventListener('click', closeContextMenu);
		document.addEventListener('contextmenu', closeContextMenu);
	}

	function closeContextMenu() {
		showContextMenu = false;
		document.removeEventListener('click', closeContextMenu);
		document.removeEventListener('contextmenu', closeContextMenu);
	}

	function handleCustomizeIcon() {
		closeContextMenu();
		onCustomizeIcon?.(id);
	}

	function handleResetIcon() {
		closeContextMenu();
		windowStore.resetIconCustomization(id);
	}

	// Folder drop handlers
	function handleFolderDragOver(event: DragEvent) {
		if (iconType !== 'folder') return;
		event.preventDefault();
		if (event.dataTransfer) {
			event.dataTransfer.dropEffect = 'move';
		}
		isDragOver = true;
	}

	function handleFolderDragLeave() {
		isDragOver = false;
	}

	function handleFolderDrop(event: DragEvent) {
		if (iconType !== 'folder' || !folderId) return;
		event.preventDefault();
		isDragOver = false;

		const droppedIconId = event.dataTransfer?.getData('text/icon-id');
		if (droppedIconId && droppedIconId !== id) {
			windowStore.moveIconToFolder(droppedIconId, folderId);
		}
	}

	const iconData = $derived(iconPaths[module] || iconPaths.dashboard);
	const isTerminal = module === 'terminal';
	const isPlatform = module === 'platform';

	// HTML5 drag start for dock pinning and folder dropping
	function handleNativeDragStart(event: DragEvent) {
		if (event.dataTransfer) {
			event.dataTransfer.setData('text/plain', module);
			event.dataTransfer.setData('text/icon-id', id);
			event.dataTransfer.effectAllowed = 'copyMove';
		}
	}

	// Use folder color for folder icons
	const effectiveIconData = $derived(() => {
		if (iconType === 'folder' && folderColor) {
			return {
				...iconPaths.folder,
				color: folderColor,
				bgColor: `${folderColor}20`
			};
		}
		return iconPaths[module] || iconPaths.dashboard;
	});
</script>

<div
	class="desktop-icon style-{iconStyle}"
	class:selected
	class:dragging={isDragging}
	class:dark-bg={darkBackground}
	class:drag-over={isDragOver}
	class:is-folder={iconType === 'folder'}
	style="width: {containerWidth}px;"
	onmousedown={handleMouseDown}
	oncontextmenu={handleContextMenu}
	ondragstart={handleNativeDragStart}
	ondragover={handleFolderDragOver}
	ondragleave={handleFolderDragLeave}
	ondrop={handleFolderDrop}
	draggable="true"
	role="button"
	tabindex="0"
	aria-label={label}
>
	<div
		class="icon-image"
		class:terminal={isTerminal}
		style="
			width: {imageSize}px;
			height: {imageSize}px;
			--base-radius: {Math.max(8, imageSize * 0.2)}px;
			background-color: {iconStyle === 'minimal' ? 'transparent' : (customIcon?.backgroundColor || effectiveIconData().bgColor)};
			{iconStyle === 'outlined' ? `border: 2px solid ${customIcon?.foregroundColor || effectiveIconData().color}; background-color: transparent;` : ''}
			{iconStyle === 'neon' ? `color: ${customIcon?.foregroundColor || effectiveIconData().color};` : ''}
			{iconStyle === 'gradient' ? `--gradient-start: ${customIcon?.foregroundColor || effectiveIconData().color}; --gradient-end: ${customIcon?.backgroundColor || effectiveIconData().bgColor};` : ''}
		"
	>
		{#if customIcon?.type === 'lucide' && customIcon.lucideName}
			<!-- Custom Lucide icon -->
			{@const LucideIcon = getLucideIcon(customIcon.lucideName)}
			{#if LucideIcon}
				<svelte:component
					this={LucideIcon}
					size={svgSize}
					color={customIcon.foregroundColor || effectiveIconData().color}
					strokeWidth={libraryStrokeWidth}
				/>
			{/if}
		{:else if customIcon?.type === 'custom' && customIcon.customSvg}
			<!-- Custom SVG -->
			<div
				class="custom-svg-container"
				style="width: {svgSize}px; height: {svgSize}px; color: {customIcon.foregroundColor || effectiveIconData().color};"
			>
				{@html customIcon.customSvg}
			</div>
		{:else if customIcon?.type === 'image' && customIcon.imageUrl}
			<img class="custom-image-icon" src={customIcon.imageUrl} alt="" draggable="false" />
		{:else if isTerminal}
			<div class="terminal-icon">
				<span class="terminal-prompt" style="font-size: {svgSize * 0.65}px;">&gt;_</span>
			</div>
		{:else if iconType === 'folder'}
			<!-- Folder icon with fill -->
			<svg
				class="icon-svg"
				viewBox="0 0 24 24"
				fill={effectiveIconData().color}
				stroke="none"
				style="width: {svgSize * 1.2}px; height: {svgSize * 1.2}px;"
			>
				<path d={effectiveIconData().path} />
			</svg>
		{:else}
			<svg
				class="icon-svg library-{iconLibrary}"
				viewBox="0 0 24 24"
				fill="none"
				stroke={effectiveIconData().color}
				stroke-width={libraryStrokeWidth}
				stroke-linecap={libraryLineCap}
				stroke-linejoin={libraryLineJoin}
				style="
					width: {svgSize * libraryIconScale}px;
					height: {svgSize * libraryIconScale}px;
					opacity: {libraryOpacity};
					filter: {librarySvgFilter};
				"
			>
				<path d={effectiveIconData().path} />
			</svg>
		{/if}
	</div>
	{#if showIconLabels}
		<span class="icon-label" class:selected style="font-size: {labelSize}px; max-width: {containerWidth - 8}px;">{label}</span>
	{/if}
</div>

<!-- Context Menu -->
{#if showContextMenu}
	<DesktopIconContextMenu
		x={contextMenuX}
		y={contextMenuY}
		hasCustomIcon={!!customIcon}
		onCustomize={handleCustomizeIcon}
		onReset={handleResetIcon}
		onOpen={() => { closeContextMenu(); onOpen?.(module); }}
	/>
{/if}

<style>
	.desktop-icon {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 6px;
		padding: 8px;
		border: none;
		background: transparent;
		cursor: pointer;
		border-radius: 8px;
		transition: transform 0.15s ease;
		width: 80px;
		user-select: none;
	}

	.desktop-icon:hover:not(.dragging) {
		transform: scale(1.05);
	}

	.desktop-icon.dragging {
		opacity: 0.8;
		cursor: grabbing;
		transition: none;
	}

	.desktop-icon:hover .icon-image {
		box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
	}

	.desktop-icon.selected .icon-image {
		box-shadow: 0 0 0 2px #0066FF;
	}

	/* Folder drag-over highlight */
	.desktop-icon.is-folder.drag-over .icon-image {
		box-shadow: 0 0 0 3px #3B82F6, 0 8px 20px rgba(59, 130, 246, 0.4);
		transform: scale(1.1);
	}

	.desktop-icon.is-folder.drag-over {
		transform: scale(1.05);
	}

	.icon-image {
		width: 56px;
		height: 56px;
		border-radius: var(--base-radius, 12px);
		display: flex;
		align-items: center;
		justify-content: center;
		box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
		transition: box-shadow 0.15s ease;
	}

	.icon-image.terminal {
		background: #1E1E1E !important;
	}

	.terminal-icon {
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.terminal-prompt {
		font-family: 'SF Mono', 'Monaco', 'Inconsolata', 'Fira Code', 'Courier New', monospace;
		font-size: 18px;
		font-weight: bold;
		color: #00FF00;
		text-shadow: 0 0 10px rgba(0, 255, 0, 0.5);
	}

	.icon-svg {
		width: 28px;
		height: 28px;
		transition: all 0.2s ease;
	}

	/* Icon Library-specific styles for DRAMATIC visual differences */
	.icon-svg.library-lucide {
		/* Lucide: Clean, balanced - default look */
	}

	.icon-svg.library-phosphor {
		/* Phosphor: Bold, duotone-inspired look */
		stroke-dasharray: none;
		stroke-opacity: 1;
	}

	.icon-svg.library-tabler {
		/* Tabler: Thin, minimal, geometric */
		stroke-dasharray: none;
		stroke-opacity: 0.8;
	}

	.icon-svg.library-heroicons {
		/* Heroicons: Solid, professional */
		stroke-dasharray: none;
		stroke-opacity: 1;
	}

	.icon-label {
		font-size: 11px;
		font-weight: 500;
		color: #333;
		text-align: center;
		max-width: 90px;
		overflow: hidden;
		display: -webkit-box;
		-webkit-line-clamp: 2;
		-webkit-box-orient: vertical;
		line-height: 1.3;
		padding: 2px 6px;
		border-radius: 4px;
		text-shadow: 0 1px 2px rgba(255, 255, 255, 0.8);
		word-break: break-word;
	}

	/* Icon Style Variants */

	/* Minimal - no backgrounds, just icons */
	.desktop-icon.style-minimal .icon-image {
		box-shadow: none;
		background: transparent !important;
	}

	.desktop-icon.style-minimal:hover .icon-image {
		box-shadow: none;
		background: rgba(0, 0, 0, 0.05) !important;
	}

	.desktop-icon.style-minimal.selected .icon-image {
		box-shadow: none;
		background: rgba(0, 102, 255, 0.1) !important;
	}

	.desktop-icon.style-minimal .icon-svg {
		width: 36px;
		height: 36px;
	}

	/* Rounded - circular backgrounds */
	.desktop-icon.style-rounded .icon-image {
		border-radius: 50%;
	}

	/* Square - sharp corners */
	.desktop-icon.style-square .icon-image {
		border-radius: 4px;
	}

	/* macOS - squircle style */
	.desktop-icon.style-macos .icon-image {
		border-radius: 22%;
		width: 60px;
		height: 60px;
	}

	.desktop-icon.style-macos .icon-svg {
		width: 32px;
		height: 32px;
	}

	/* Outlined - border outline style */
	.desktop-icon.style-outlined .icon-image {
		box-shadow: none;
	}

	.desktop-icon.style-outlined:hover .icon-image {
		box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
	}

	.desktop-icon.style-outlined.selected .icon-image {
		box-shadow: 0 0 0 2px #0066FF;
	}

	/* Retro - classic pixelated computer style */
	.desktop-icon.style-retro .icon-image {
		border-radius: 0;
		box-shadow:
			4px 4px 0 rgba(0, 0, 0, 0.3),
			inset -2px -2px 0 rgba(0, 0, 0, 0.2),
			inset 2px 2px 0 rgba(255, 255, 255, 0.3);
		image-rendering: pixelated;
	}

	.desktop-icon.style-retro .icon-label {
		font-family: 'Courier New', monospace;
		text-shadow: 1px 1px 0 rgba(0, 0, 0, 0.3);
	}

	.desktop-icon.style-retro.selected .icon-image {
		box-shadow:
			4px 4px 0 rgba(0, 0, 0, 0.3),
			0 0 0 2px #0066FF;
	}

	/* Win95 - Windows 95 style 3D borders */
	.desktop-icon.style-win95 .icon-image {
		border-radius: 0;
		box-shadow: none;
		border: 2px solid;
		border-color: #DFDFDF #808080 #808080 #DFDFDF;
		background: #C0C0C0 !important;
	}

	.desktop-icon.style-win95:hover .icon-image {
		border-color: #808080 #DFDFDF #DFDFDF #808080;
	}

	.desktop-icon.style-win95.selected .icon-image {
		border-color: #808080 #DFDFDF #DFDFDF #808080;
		background: #000080 !important;
	}

	.desktop-icon.style-win95 .icon-label {
		font-family: 'MS Sans Serif', 'Segoe UI', sans-serif;
		font-size: 11px;
	}

	.desktop-icon.style-win95.selected .icon-label {
		background: #000080;
		color: white;
		text-shadow: none;
	}

	/* Glassmorphism - frosted glass effect */
	.desktop-icon.style-glassmorphism .icon-image {
		background: rgba(255, 255, 255, 0.2) !important;
		backdrop-filter: blur(10px);
		-webkit-backdrop-filter: blur(10px);
		border: 1px solid rgba(255, 255, 255, 0.3);
		box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1);
	}

	.desktop-icon.style-glassmorphism:hover .icon-image {
		background: rgba(255, 255, 255, 0.3) !important;
		box-shadow: 0 8px 32px rgba(0, 0, 0, 0.15);
	}

	.desktop-icon.style-glassmorphism.selected .icon-image {
		border-color: #0066FF;
		box-shadow: 0 0 0 2px rgba(0, 102, 255, 0.3), 0 8px 32px rgba(0, 0, 0, 0.15);
	}

	/* Neon - glowing neon style */
	.desktop-icon.style-neon .icon-image {
		background: #1a1a2e !important;
		border-radius: 12px;
		box-shadow:
			0 0 10px currentColor,
			0 0 20px currentColor,
			inset 0 0 10px rgba(255, 255, 255, 0.1);
		border: 1px solid currentColor;
	}

	.desktop-icon.style-neon:hover .icon-image {
		box-shadow:
			0 0 15px currentColor,
			0 0 30px currentColor,
			0 0 45px currentColor,
			inset 0 0 10px rgba(255, 255, 255, 0.1);
	}

	.desktop-icon.style-neon .icon-svg {
		filter: drop-shadow(0 0 3px currentColor);
	}

	.desktop-icon.style-neon .icon-label {
		color: var(--bos-surface-on-color);
		text-shadow: 0 0 10px currentColor;
	}

	.desktop-icon.style-neon.selected .icon-image {
		box-shadow:
			0 0 10px #0066FF,
			0 0 20px #0066FF,
			0 0 30px #0066FF,
			inset 0 0 10px rgba(255, 255, 255, 0.1);
		border-color: #0066FF;
	}

	/* Flat - flat design with no shadows */
	.desktop-icon.style-flat .icon-image {
		box-shadow: none;
		border-radius: 8px;
	}

	.desktop-icon.style-flat:hover .icon-image {
		box-shadow: none;
		filter: brightness(0.95);
	}

	.desktop-icon.style-flat.selected .icon-image {
		box-shadow: none;
		outline: 2px solid #0066FF;
		outline-offset: 2px;
	}

	/* Gradient - gradient background style */
	.desktop-icon.style-gradient .icon-image {
		background: linear-gradient(135deg, var(--gradient-start, #667eea) 0%, var(--gradient-end, #764ba2) 100%) !important;
		box-shadow: 0 4px 15px rgba(0, 0, 0, 0.2);
	}

	.desktop-icon.style-gradient .icon-svg {
		stroke: white !important;
		filter: drop-shadow(0 1px 2px rgba(0, 0, 0, 0.2));
	}

	.desktop-icon.style-gradient:hover .icon-image {
		box-shadow: 0 6px 20px rgba(0, 0, 0, 0.25);
		transform: translateY(-2px);
	}

	.desktop-icon.style-gradient.selected .icon-image {
		box-shadow: 0 0 0 2px #0066FF, 0 4px 15px rgba(0, 0, 0, 0.2);
	}

	.icon-label.selected {
		background: #0066FF;
		color: white;
		text-shadow: none;
	}

	/* macOS Classic - Mac OS 9 platinum style */
	.desktop-icon.style-macos-classic .icon-image {
		border-radius: 4px;
		background: linear-gradient(180deg, #EAEAEA 0%, #D4D4D4 50%, #C4C4C4 100%) !important;
		border: 1px solid;
		border-color: #FFFFFF #888888 #888888 #FFFFFF;
		box-shadow:
			1px 1px 0 #666666,
			inset 1px 1px 0 rgba(255, 255, 255, 0.8);
	}

	.desktop-icon.style-macos-classic:hover .icon-image {
		background: linear-gradient(180deg, #F0F0F0 0%, #E0E0E0 50%, #D0D0D0 100%) !important;
	}

	.desktop-icon.style-macos-classic.selected .icon-image {
		background: linear-gradient(180deg, #3366CC 0%, #2255BB 50%, #1144AA 100%) !important;
		border-color: #1144AA #000033 #000033 #1144AA;
	}

	.desktop-icon.style-macos-classic.selected .icon-svg {
		stroke: white !important;
	}

	.desktop-icon.style-macos-classic .icon-label {
		font-family: 'Chicago', 'Charcoal', 'Geneva', 'Helvetica', sans-serif;
		font-size: 10px;
		font-weight: normal;
		text-shadow: none;
		color: #000;
	}

	.desktop-icon.style-macos-classic.selected .icon-label {
		background: #3366CC;
		color: white;
		text-shadow: none;
	}

	/* Paper - card style with soft shadows */
	.desktop-icon.style-paper .icon-image {
		background: #FFFFFF !important;
		border-radius: 8px;
		box-shadow:
			0 1px 3px rgba(0, 0, 0, 0.08),
			0 4px 12px rgba(0, 0, 0, 0.05);
		border: 1px solid rgba(0, 0, 0, 0.06);
	}

	.desktop-icon.style-paper:hover .icon-image {
		box-shadow:
			0 2px 8px rgba(0, 0, 0, 0.1),
			0 8px 24px rgba(0, 0, 0, 0.08);
		transform: translateY(-2px);
	}

	.desktop-icon.style-paper.selected .icon-image {
		box-shadow:
			0 0 0 2px #0066FF,
			0 2px 8px rgba(0, 0, 0, 0.1);
	}

	.desktop-icon.style-paper .icon-label {
		background: rgba(255, 255, 255, 0.9);
		padding: 3px 8px;
		border-radius: 4px;
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.06);
	}

	/* Pixel - 8-bit pixel art style */
	.desktop-icon.style-pixel .icon-image {
		border-radius: 0;
		image-rendering: pixelated;
		box-shadow:
			4px 0 0 #000,
			-4px 0 0 #000,
			0 4px 0 #000,
			0 -4px 0 #000;
		border: none;
	}

	.desktop-icon.style-pixel:hover .icon-image {
		box-shadow:
			4px 0 0 #333,
			-4px 0 0 #333,
			0 4px 0 #333,
			0 -4px 0 #333;
		filter: brightness(1.1);
	}

	.desktop-icon.style-pixel.selected .icon-image {
		box-shadow:
			4px 0 0 #0066FF,
			-4px 0 0 #0066FF,
			0 4px 0 #0066FF,
			0 -4px 0 #0066FF;
	}

	.desktop-icon.style-pixel .icon-svg {
		image-rendering: pixelated;
	}

	.desktop-icon.style-pixel .icon-label {
		font-family: 'Press Start 2P', 'Courier New', monospace;
		font-size: 8px;
		letter-spacing: 0.5px;
		text-transform: uppercase;
	}

	/* Frosted - clean frosted glass with blur */
	.desktop-icon.style-frosted .icon-image {
		background: rgba(255, 255, 255, 0.6) !important;
		backdrop-filter: blur(12px) saturate(180%);
		-webkit-backdrop-filter: blur(12px) saturate(180%);
		border-radius: 14px;
		border: 1px solid rgba(255, 255, 255, 0.4);
		box-shadow: 0 4px 16px rgba(0, 0, 0, 0.08);
	}

	.desktop-icon.style-frosted:hover .icon-image {
		background: rgba(255, 255, 255, 0.75) !important;
		box-shadow: 0 6px 20px rgba(0, 0, 0, 0.12);
	}

	.desktop-icon.style-frosted.selected .icon-image {
		border-color: #0066FF;
		box-shadow: 0 0 0 2px rgba(0, 102, 255, 0.3), 0 6px 20px rgba(0, 0, 0, 0.12);
	}

	/* Terminal - green on black hacker aesthetic */
	.desktop-icon.style-terminal .icon-image {
		background: #0a0a0a !important;
		border-radius: 4px;
		border: 1px solid #00ff00;
		box-shadow: 0 0 10px rgba(0, 255, 0, 0.3), inset 0 0 20px rgba(0, 255, 0, 0.05);
	}

	.desktop-icon.style-terminal:hover .icon-image {
		box-shadow: 0 0 15px rgba(0, 255, 0, 0.5), inset 0 0 30px rgba(0, 255, 0, 0.1);
		border-color: #00ff00;
	}

	.desktop-icon.style-terminal.selected .icon-image {
		box-shadow: 0 0 20px rgba(0, 255, 0, 0.7), 0 0 40px rgba(0, 255, 0, 0.3);
	}

	.desktop-icon.style-terminal .icon-svg {
		color: #00ff00 !important;
		filter: drop-shadow(0 0 2px #00ff00);
	}

	.desktop-icon.style-terminal .icon-label {
		font-family: 'Courier New', monospace;
		color: #00ff00;
		text-shadow: 0 0 5px rgba(0, 255, 0, 0.5);
	}

	/* Glow - soft colored glow aura effect */
	.desktop-icon.style-glow .icon-image {
		border-radius: 14px;
		box-shadow:
			0 0 20px currentColor,
			0 0 40px rgba(100, 100, 255, 0.3);
		border: none;
	}

	.desktop-icon.style-glow:hover .icon-image {
		box-shadow:
			0 0 25px currentColor,
			0 0 50px rgba(100, 100, 255, 0.4);
		transform: scale(1.02);
	}

	.desktop-icon.style-glow.selected .icon-image {
		box-shadow:
			0 0 30px #0066FF,
			0 0 60px rgba(0, 102, 255, 0.5);
	}

	.desktop-icon.style-glow .icon-svg {
		filter: drop-shadow(0 0 4px currentColor);
	}

	/* Brutalist - bold raw design with thick borders */
	.desktop-icon.style-brutalist .icon-image {
		background: var(--dbg) !important;
		border-radius: 0;
		border: 4px solid #000;
		box-shadow: 6px 6px 0 #000;
	}

	.desktop-icon.style-brutalist:hover .icon-image {
		transform: translate(-2px, -2px);
		box-shadow: 8px 8px 0 #000;
	}

	.desktop-icon.style-brutalist.selected .icon-image {
		background: #ff0 !important;
		box-shadow: 4px 4px 0 #000;
	}

	.desktop-icon.style-brutalist .icon-svg {
		color: #000 !important;
	}

	.desktop-icon.style-brutalist .icon-label {
		font-weight: 900;
		text-transform: uppercase;
		letter-spacing: 1px;
	}

	/* Depth - layered 3D depth shadows */
	.desktop-icon.style-depth .icon-image {
		border-radius: 12px;
		border: none;
		box-shadow:
			0 2px 4px rgba(0, 0, 0, 0.1),
			0 4px 8px rgba(0, 0, 0, 0.1),
			0 8px 16px rgba(0, 0, 0, 0.1),
			0 16px 32px rgba(0, 0, 0, 0.08);
	}

	.desktop-icon.style-depth:hover .icon-image {
		transform: translateY(-4px);
		box-shadow:
			0 4px 8px rgba(0, 0, 0, 0.12),
			0 8px 16px rgba(0, 0, 0, 0.12),
			0 16px 32px rgba(0, 0, 0, 0.1),
			0 24px 48px rgba(0, 0, 0, 0.08);
	}

	.desktop-icon.style-depth.selected .icon-image {
		box-shadow:
			0 2px 4px rgba(0, 102, 255, 0.2),
			0 4px 8px rgba(0, 102, 255, 0.15),
			0 8px 16px rgba(0, 102, 255, 0.1),
			0 16px 32px rgba(0, 102, 255, 0.08);
	}

	/* Neumorphism - soft 3D with inset/outset shadows */
	.desktop-icon.style-neumorphism .icon-image {
		background: #e0e0e0 !important;
		border-radius: 16px;
		border: none;
		box-shadow:
			8px 8px 16px #bebebe,
			-8px -8px 16px #ffffff;
	}

	.desktop-icon.style-neumorphism:hover .icon-image {
		box-shadow:
			4px 4px 8px #bebebe,
			-4px -4px 8px #ffffff;
	}

	.desktop-icon.style-neumorphism.selected .icon-image {
		box-shadow:
			inset 4px 4px 8px #bebebe,
			inset -4px -4px 8px #ffffff;
	}

	/* Material - Google Material Design elevation */
	.desktop-icon.style-material .icon-image {
		border-radius: 12px;
		box-shadow: 0 2px 4px rgba(0,0,0,0.2), 0 4px 8px rgba(0,0,0,0.1);
		border: none;
	}

	.desktop-icon.style-material:hover .icon-image {
		box-shadow: 0 4px 8px rgba(0,0,0,0.25), 0 8px 16px rgba(0,0,0,0.15);
		transform: translateY(-2px);
	}

	.desktop-icon.style-material.selected .icon-image {
		box-shadow: 0 0 0 2px #1976D2, 0 4px 8px rgba(25,118,210,0.3);
	}

	/* Fluent - Microsoft Fluent Design acrylic */
	.desktop-icon.style-fluent .icon-image {
		background: rgba(255, 255, 255, 0.7) !important;
		backdrop-filter: blur(20px) saturate(150%);
		-webkit-backdrop-filter: blur(20px) saturate(150%);
		border-radius: 8px;
		border: 1px solid rgba(255, 255, 255, 0.5);
		box-shadow: 0 2px 8px rgba(0,0,0,0.08);
	}

	.desktop-icon.style-fluent:hover .icon-image {
		background: rgba(255, 255, 255, 0.85) !important;
		box-shadow: 0 4px 12px rgba(0,0,0,0.12);
	}

	.desktop-icon.style-fluent.selected .icon-image {
		border-color: #0078D4;
		box-shadow: 0 0 0 2px rgba(0,120,212,0.3);
	}

	/* Aero - Windows Vista/7 glass effect */
	.desktop-icon.style-aero .icon-image {
		background: linear-gradient(180deg, rgba(255,255,255,0.8) 0%, rgba(200,220,240,0.6) 100%) !important;
		backdrop-filter: blur(8px);
		-webkit-backdrop-filter: blur(8px);
		border-radius: 6px;
		border: 1px solid rgba(255,255,255,0.7);
		box-shadow: 0 1px 4px rgba(0,0,0,0.2), inset 0 1px 0 rgba(255,255,255,0.8);
	}

	.desktop-icon.style-aero:hover .icon-image {
		background: linear-gradient(180deg, rgba(255,255,255,0.9) 0%, rgba(200,220,240,0.7) 100%) !important;
		box-shadow: 0 2px 8px rgba(0,0,0,0.25), inset 0 1px 0 rgba(255,255,255,0.9);
	}

	.desktop-icon.style-aero.selected .icon-image {
		border-color: #3399FF;
		box-shadow: 0 0 0 2px rgba(51,153,255,0.4), 0 2px 8px rgba(0,0,0,0.2);
	}

	/* Aurora - animated gradient shimmer */
	.desktop-icon.style-aurora .icon-image {
		background: linear-gradient(135deg, #667eea, #764ba2, #f093fb, #667eea) !important;
		background-size: 300% 300% !important;
		animation: auroraShift 4s ease infinite;
		border-radius: 14px;
		border: none;
		box-shadow: 0 4px 16px rgba(102,126,234,0.3);
	}

	@keyframes auroraShift {
		0% { background-position: 0% 50%; }
		50% { background-position: 100% 50%; }
		100% { background-position: 0% 50%; }
	}

	.desktop-icon.style-aurora .icon-svg {
		stroke: white !important;
		filter: drop-shadow(0 1px 2px rgba(0,0,0,0.3));
	}

	.desktop-icon.style-aurora.selected .icon-image {
		box-shadow: 0 0 0 2px #fff, 0 4px 16px rgba(102,126,234,0.5);
	}

	/* Crystal - gem-like faceted appearance */
	.desktop-icon.style-crystal .icon-image {
		background: linear-gradient(135deg, rgba(255,255,255,0.9) 0%, rgba(200,220,255,0.6) 50%, rgba(255,255,255,0.8) 100%) !important;
		border-radius: 12px;
		border: 1px solid rgba(255,255,255,0.8);
		box-shadow:
			0 4px 16px rgba(0,0,0,0.1),
			inset 0 2px 4px rgba(255,255,255,0.8),
			inset 0 -2px 4px rgba(0,0,0,0.05);
	}

	.desktop-icon.style-crystal:hover .icon-image {
		box-shadow:
			0 6px 20px rgba(0,0,0,0.15),
			inset 0 2px 4px rgba(255,255,255,0.9),
			inset 0 -2px 4px rgba(0,0,0,0.08);
		transform: scale(1.02);
	}

	.desktop-icon.style-crystal.selected .icon-image {
		border-color: #60A5FA;
		box-shadow: 0 0 0 2px rgba(96,165,250,0.4), 0 4px 16px rgba(0,0,0,0.1);
	}

	/* Holographic - rainbow shifting iridescent */
	.desktop-icon.style-holographic .icon-image {
		background: linear-gradient(135deg, #ff0080, #ff8c00, #40e0d0, #8000ff, #ff0080) !important;
		background-size: 400% 400% !important;
		animation: holoShift 3s ease infinite;
		border-radius: 12px;
		border: none;
		box-shadow: 0 4px 16px rgba(128,0,255,0.2);
	}

	@keyframes holoShift {
		0% { background-position: 0% 50%; }
		50% { background-position: 100% 50%; }
		100% { background-position: 0% 50%; }
	}

	.desktop-icon.style-holographic .icon-svg {
		stroke: white !important;
		filter: drop-shadow(0 1px 3px rgba(0,0,0,0.4));
	}

	.desktop-icon.style-holographic.selected .icon-image {
		box-shadow: 0 0 0 2px #fff, 0 4px 20px rgba(128,0,255,0.4);
	}

	/* Vaporwave - 80s/90s pink and cyan */
	.desktop-icon.style-vaporwave .icon-image {
		background: linear-gradient(135deg, #FF71CE, #01CDFE) !important;
		border-radius: 8px;
		border: 2px solid #B967FF;
		box-shadow: 4px 4px 0 #05FFA1;
	}

	.desktop-icon.style-vaporwave:hover .icon-image {
		transform: translate(-2px, -2px);
		box-shadow: 6px 6px 0 #05FFA1;
	}

	.desktop-icon.style-vaporwave .icon-svg {
		stroke: white !important;
	}

	.desktop-icon.style-vaporwave .icon-label {
		color: #FF71CE;
		text-shadow: 1px 1px 0 #01CDFE;
	}

	.desktop-icon.style-vaporwave.selected .icon-image {
		border-color: #FFFB96;
		box-shadow: 4px 4px 0 #FFFB96;
	}

	/* Cyberpunk - neon with scan lines */
	.desktop-icon.style-cyberpunk .icon-image {
		background: #0a0a0a !important;
		border-radius: 4px;
		border: 1px solid #00f0ff;
		box-shadow: 0 0 8px #00f0ff, inset 0 0 20px rgba(0,240,255,0.05);
		background-image: repeating-linear-gradient(
			0deg,
			transparent,
			transparent 2px,
			rgba(0,240,255,0.03) 2px,
			rgba(0,240,255,0.03) 4px
		) !important;
	}

	.desktop-icon.style-cyberpunk:hover .icon-image {
		box-shadow: 0 0 12px #00f0ff, 0 0 24px rgba(0,240,255,0.3);
	}

	.desktop-icon.style-cyberpunk .icon-svg {
		stroke: #00f0ff !important;
		filter: drop-shadow(0 0 3px #00f0ff);
	}

	.desktop-icon.style-cyberpunk .icon-label {
		color: #00f0ff;
		font-family: 'Courier New', monospace;
		text-shadow: 0 0 5px #00f0ff;
	}

	.desktop-icon.style-cyberpunk.selected .icon-image {
		border-color: #ff2a6d;
		box-shadow: 0 0 8px #ff2a6d, 0 0 16px rgba(255,42,109,0.3);
	}

	/* Synthwave - retro futuristic purple/pink */
	.desktop-icon.style-synthwave .icon-image {
		background: linear-gradient(180deg, #2d1b69, #1a0533) !important;
		border-radius: 8px;
		border: 1px solid #e040fb;
		box-shadow: 0 0 10px rgba(224,64,251,0.3), 0 4px 16px rgba(0,0,0,0.3);
	}

	.desktop-icon.style-synthwave:hover .icon-image {
		box-shadow: 0 0 15px rgba(224,64,251,0.5), 0 6px 20px rgba(0,0,0,0.4);
	}

	.desktop-icon.style-synthwave .icon-svg {
		stroke: #e040fb !important;
		filter: drop-shadow(0 0 3px #e040fb);
	}

	.desktop-icon.style-synthwave .icon-label {
		color: #e040fb;
		text-shadow: 0 0 5px rgba(224,64,251,0.5);
	}

	.desktop-icon.style-synthwave.selected .icon-image {
		border-color: #00e5ff;
		box-shadow: 0 0 10px rgba(0,229,255,0.5);
	}

	/* Matrix - green code rain style */
	.desktop-icon.style-matrix .icon-image {
		background: #000 !important;
		border-radius: 4px;
		border: 1px solid #00ff41;
		box-shadow: 0 0 10px rgba(0,255,65,0.2), inset 0 0 20px rgba(0,255,65,0.05);
	}

	.desktop-icon.style-matrix:hover .icon-image {
		box-shadow: 0 0 15px rgba(0,255,65,0.4), inset 0 0 30px rgba(0,255,65,0.1);
	}

	.desktop-icon.style-matrix .icon-svg {
		stroke: #00ff41 !important;
		filter: drop-shadow(0 0 2px #00ff41);
	}

	.desktop-icon.style-matrix .icon-label {
		color: #00ff41;
		font-family: 'Courier New', monospace;
		text-shadow: 0 0 5px rgba(0,255,65,0.5);
	}

	.desktop-icon.style-matrix.selected .icon-image {
		box-shadow: 0 0 20px rgba(0,255,65,0.6), inset 0 0 20px rgba(0,255,65,0.1);
	}

	/* Glitch - digital glitch distortion */
	.desktop-icon.style-glitch .icon-image {
		border-radius: 8px;
		box-shadow: 3px 0 #ff0000, -3px 0 #00ffff;
		animation: glitchShake 3s infinite;
	}

	@keyframes glitchShake {
		0%, 95%, 100% { transform: translate(0); }
		96% { transform: translate(-2px, 1px); box-shadow: 4px 0 #ff0000, -4px 0 #00ffff; }
		97% { transform: translate(2px, -1px); box-shadow: -3px 0 #ff0000, 3px 0 #00ffff; }
		98% { transform: translate(-1px, 2px); }
		99% { transform: translate(1px, -2px); }
	}

	.desktop-icon.style-glitch:hover .icon-image {
		animation-duration: 0.5s;
	}

	.desktop-icon.style-glitch.selected .icon-image {
		box-shadow: 3px 0 #ff0000, -3px 0 #00ffff, 0 0 0 2px #0066FF;
	}

	/* Chrome - metallic reflective surface */
	.desktop-icon.style-chrome .icon-image {
		background: linear-gradient(180deg, #e8e8e8, #c0c0c0, #e8e8e8, #a0a0a0) !important;
		border-radius: 10px;
		border: 1px solid #999;
		box-shadow: 0 2px 8px rgba(0,0,0,0.2), inset 0 1px 0 rgba(255,255,255,0.8);
	}

	.desktop-icon.style-chrome:hover .icon-image {
		background: linear-gradient(180deg, #f0f0f0, #d0d0d0, #f0f0f0, #b0b0b0) !important;
		box-shadow: 0 4px 12px rgba(0,0,0,0.25);
	}

	.desktop-icon.style-chrome .icon-svg {
		stroke: #444 !important;
		filter: drop-shadow(0 1px 0 rgba(255,255,255,0.5));
	}

	.desktop-icon.style-chrome.selected .icon-image {
		border-color: #0066FF;
		box-shadow: 0 0 0 2px rgba(0,102,255,0.3), 0 2px 8px rgba(0,0,0,0.2);
	}

	/* Rainbow - animated rainbow spectrum */
	.desktop-icon.style-rainbow .icon-image {
		background: linear-gradient(135deg, #ff0000, #ff7700, #ffff00, #00ff00, #0000ff, #8b00ff, #ff0000) !important;
		background-size: 400% 400% !important;
		animation: rainbowShift 3s linear infinite;
		border-radius: 12px;
		border: none;
	}

	@keyframes rainbowShift {
		0% { background-position: 0% 50%; }
		100% { background-position: 400% 50%; }
	}

	.desktop-icon.style-rainbow .icon-svg {
		stroke: white !important;
		filter: drop-shadow(0 1px 2px rgba(0,0,0,0.3));
	}

	.desktop-icon.style-rainbow.selected .icon-image {
		box-shadow: 0 0 0 2px #fff, 0 4px 16px rgba(0,0,0,0.2);
	}

	/* Sketch - hand-drawn outline style */
	.desktop-icon.style-sketch .icon-image {
		background: #FEFCE8 !important;
		border-radius: 8px;
		border: 2px solid #333;
		box-shadow: 3px 3px 0 #333;
		transform: rotate(-1deg);
	}

	.desktop-icon.style-sketch:hover .icon-image {
		transform: rotate(0deg);
		box-shadow: 2px 2px 0 #333;
	}

	.desktop-icon.style-sketch .icon-svg {
		stroke: #333 !important;
		stroke-dasharray: 4 2;
	}

	.desktop-icon.style-sketch .icon-label {
		font-family: 'Comic Sans MS', 'Chalkboard SE', cursive;
	}

	.desktop-icon.style-sketch.selected .icon-image {
		border-color: #0066FF;
		box-shadow: 3px 3px 0 #0066FF;
	}

	/* Comic - comic book thick black borders */
	.desktop-icon.style-comic .icon-image {
		background: #FFF !important;
		border-radius: 8px;
		border: 3px solid #000;
		box-shadow: 4px 4px 0 #000;
	}

	.desktop-icon.style-comic:hover .icon-image {
		transform: scale(1.05) rotate(-2deg);
		box-shadow: 5px 5px 0 #000;
	}

	.desktop-icon.style-comic .icon-label {
		font-family: 'Comic Sans MS', 'Chalkboard SE', cursive;
		font-weight: bold;
		text-transform: uppercase;
	}

	.desktop-icon.style-comic.selected .icon-image {
		background: #FFFF00 !important;
		box-shadow: 4px 4px 0 #0066FF;
		border-color: #0066FF;
	}

	/* Watercolor - soft blurred watercolor paint */
	.desktop-icon.style-watercolor .icon-image {
		border-radius: 50%;
		box-shadow: 0 0 20px rgba(0,0,0,0.08);
		border: none;
		filter: blur(0.5px) saturate(1.3);
	}

	.desktop-icon.style-watercolor:hover .icon-image {
		filter: blur(0px) saturate(1.5);
		box-shadow: 0 0 25px rgba(0,0,0,0.12);
	}

	.desktop-icon.style-watercolor .icon-svg {
		opacity: 0.85;
	}

	.desktop-icon.style-watercolor.selected .icon-image {
		box-shadow: 0 0 0 3px rgba(0,102,255,0.3), 0 0 20px rgba(0,0,0,0.08);
	}

	/* iOS - iOS app icon rounded square */
	.desktop-icon.style-ios .icon-image {
		border-radius: 22%;
		box-shadow: 0 2px 8px rgba(0,0,0,0.15);
		border: none;
		overflow: hidden;
	}

	.desktop-icon.style-ios:hover .icon-image {
		box-shadow: 0 4px 16px rgba(0,0,0,0.2);
		transform: scale(1.05);
	}

	.desktop-icon.style-ios.selected .icon-image {
		box-shadow: 0 0 0 3px #007AFF, 0 2px 8px rgba(0,0,0,0.15);
	}

	.desktop-icon.style-ios .icon-label {
		font-family: -apple-system, 'SF Pro', system-ui, sans-serif;
		font-weight: 500;
	}

	/* Android - Material You rounded square */
	.desktop-icon.style-android .icon-image {
		border-radius: 28%;
		box-shadow: 0 1px 3px rgba(0,0,0,0.12), 0 4px 8px rgba(0,0,0,0.08);
		border: none;
	}

	.desktop-icon.style-android:hover .icon-image {
		box-shadow: 0 2px 6px rgba(0,0,0,0.16), 0 8px 16px rgba(0,0,0,0.12);
		transform: translateY(-1px);
	}

	.desktop-icon.style-android.selected .icon-image {
		box-shadow: 0 0 0 3px #4285F4, 0 2px 6px rgba(0,0,0,0.12);
	}

	.desktop-icon.style-android .icon-label {
		font-family: 'Roboto', 'Google Sans', sans-serif;
	}

	/* Windows 11 - modern Windows 11 rounded */
	.desktop-icon.style-windows11 .icon-image {
		border-radius: 8px;
		background: rgba(255,255,255,0.8) !important;
		backdrop-filter: blur(16px);
		-webkit-backdrop-filter: blur(16px);
		border: 1px solid rgba(0,0,0,0.06);
		box-shadow: 0 2px 8px rgba(0,0,0,0.06);
	}

	.desktop-icon.style-windows11:hover .icon-image {
		background: rgba(255,255,255,0.9) !important;
		box-shadow: 0 4px 12px rgba(0,0,0,0.1);
	}

	.desktop-icon.style-windows11.selected .icon-image {
		border-color: #0078D4;
		box-shadow: 0 0 0 2px rgba(0,120,212,0.2);
	}

	.desktop-icon.style-windows11 .icon-label {
		font-family: 'Segoe UI Variable', 'Segoe UI', sans-serif;
	}

	/* Amiga - Amiga Workbench retro style */
	.desktop-icon.style-amiga .icon-image {
		border-radius: 0;
		background: #0055AA !important;
		border: 2px solid;
		border-color: #FFFFFF #000000 #000000 #FFFFFF;
		box-shadow: none;
	}

	.desktop-icon.style-amiga:hover .icon-image {
		background: #0066CC !important;
	}

	.desktop-icon.style-amiga .icon-svg {
		stroke: #FF8800 !important;
	}

	.desktop-icon.style-amiga .icon-label {
		font-family: 'Courier New', monospace;
		font-size: 10px;
		color: var(--bos-surface-on-color);
		background: #0055AA;
		padding: 1px 4px;
		text-shadow: none;
	}

	.desktop-icon.style-amiga.selected .icon-image {
		background: #FF8800 !important;
		border-color: #000000 #FFFFFF #FFFFFF #000000;
	}

	.desktop-icon.style-amiga.selected .icon-svg {
		stroke: #0055AA !important;
	}

	.desktop-icon.style-amiga.selected .icon-label {
		background: #FF8800;
		color: #000;
	}

	/* Dark background mode - light text */
	.desktop-icon.dark-bg .icon-label {
		color: #FFFFFF;
		text-shadow: 0 1px 3px rgba(0, 0, 0, 0.8), 0 0 8px rgba(0, 0, 0, 0.5);
	}

	.desktop-icon.dark-bg.selected .icon-label,
	.desktop-icon.dark-bg .icon-label.selected {
		background: rgba(0, 102, 255, 0.9);
		color: white;
		text-shadow: none;
	}

	/* Dark background specific style overrides */
	.desktop-icon.dark-bg.style-win95.selected .icon-label {
		background: #000080;
		color: white;
	}

	.desktop-icon.dark-bg.style-macos-classic .icon-label {
		color: #FFFFFF;
		text-shadow: 0 1px 3px rgba(0, 0, 0, 0.8);
	}

	.desktop-icon.dark-bg.style-macos-classic.selected .icon-label {
		background: #3366CC;
		color: white;
		text-shadow: none;
	}

	.desktop-icon.dark-bg.style-paper .icon-label {
		background: rgba(0, 0, 0, 0.6);
		color: white;
		text-shadow: none;
	}

	.desktop-icon.dark-bg.style-retro .icon-label {
		color: #00FF00;
		text-shadow: 0 0 10px rgba(0, 255, 0, 0.5), 1px 1px 0 rgba(0, 0, 0, 0.5);
	}

	/* Custom SVG container */
	.custom-svg-container {
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.custom-svg-container :global(svg) {
		width: 100%;
		height: 100%;
	}

	/* Context menu styles are in DesktopIconContextMenu.svelte */
	.custom-image-icon {
		width: 62%;
		height: 62%;
		object-fit: contain;
		border-radius: 7px;
		pointer-events: none;
		user-select: none;
	}
</style>
