<script lang="ts">
	interface DockItemData {
		id: string;
		module: string;
		label: string;
		isOpen: boolean;
		isMinimized: boolean;
		windowId?: string;
		folderId?: string;
		folderColor?: string;
	}

	interface IconData {
		path: string;
		color: string;
		bgColor: string;
		isTerminal?: boolean;
		isFolder?: boolean;
		isFinder?: boolean;
		imageUrl?: string;
	}

	interface Props {
		item: DockItemData;
		index: number;
		hoveredIndex: number | null;
		iconStyle: string;
		iconLibrary: string;
		libraryStrokeWidth: number;
		libraryLineCap: 'round' | 'square' | 'butt';
		libraryLineJoin: 'round' | 'miter' | 'bevel';
		libraryIconScale: number;
		librarySvgFilter: string;
		scale: number;
		baseSize: number;
		translateY: number;
		icon: IconData;
		onclick: () => void;
		onmouseenter: () => void;
		onmouseleave: () => void;
	}

	let {
		item,
		index,
		hoveredIndex,
		iconStyle,
		iconLibrary,
		libraryStrokeWidth,
		libraryLineCap,
		libraryLineJoin,
		libraryIconScale,
		librarySvgFilter,
		scale,
		baseSize,
		translateY,
		icon,
		onclick,
		onmouseenter,
		onmouseleave
	}: Props = $props();
</script>

<button
	class="dock-item style-{iconStyle}"
	class:has-indicator={item.isOpen}
	style="transform: scale({scale}) translateY({translateY}px); --dock-icon-size: {baseSize}px;"
	{onmouseenter}
	{onmouseleave}
	{onclick}
	aria-label={item.label}
>
	<div
		class="dock-icon"
		class:terminal={icon.isTerminal}
		class:finder={icon.isFinder}
		style="
			{icon.isFinder ? `background: ${icon.bgColor};` : `background-color: ${iconStyle === 'minimal' ? 'transparent' : icon.bgColor};`}
			{iconStyle === 'outlined' && !icon.isFinder ? `border: 2px solid ${icon.color}; background-color: transparent;` : ''}
			{iconStyle === 'neon' ? `color: ${icon.color};` : ''}
			{iconStyle === 'gradient' ? `--gradient-start: ${icon.color}; --gradient-end: ${icon.bgColor};` : ''}
		"
	>
		{#if icon.isTerminal}
			<span class="terminal-prompt">&gt;_</span>
		{:else if icon.isFinder}
			<svg class="dock-icon-svg finder-face" viewBox="0 0 24 24" fill="none">
				<rect x="6" y="7" width="4" height="6" rx="2" fill="white"/>
				<rect x="14" y="7" width="4" height="6" rx="2" fill="white"/>
				<path d="M6 17 C8 20, 16 20, 18 17" stroke="white" stroke-width="2.5" stroke-linecap="round" fill="none"/>
			</svg>
		{:else if icon.imageUrl}
			<img class="dock-icon-image" src={icon.imageUrl} alt="" />
		{:else if icon.isFolder}
			<svg
				class="dock-icon-svg"
				viewBox="0 0 24 24"
				fill={icon.color}
				stroke="none"
			>
				<path d={icon.path} />
			</svg>
		{:else}
			<svg
				class="dock-icon-svg library-{iconLibrary}"
				viewBox="0 0 24 24"
				fill="none"
				stroke={icon.color}
				stroke-width={libraryStrokeWidth}
				stroke-linecap={libraryLineCap}
				stroke-linejoin={libraryLineJoin}
				style="transform: scale({libraryIconScale}); filter: {librarySvgFilter};"
			>
				<path d={icon.path} />
			</svg>
		{/if}
	</div>

	<!-- Open indicator dot -->
	{#if item.isOpen}
		<div class="dock-indicator"></div>
	{/if}

	<!-- Tooltip -->
	{#if hoveredIndex === index}
		<div class="dock-tooltip">{item.label}</div>
	{/if}
</button>

<style>
	.dock-item {
		position: relative;
		display: flex;
		flex-direction: column;
		align-items: center;
		background: none;
		border: none;
		cursor: pointer;
		padding: 4px;
		transition: transform 0.15s ease-out;
		transform-origin: bottom center;
	}

	.dock-icon {
		width: var(--dock-icon-size, 48px);
		height: var(--dock-icon-size, 48px);
		border-radius: calc(var(--dock-icon-size, 48px) * 0.25);
		display: flex;
		align-items: center;
		justify-content: center;
		box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
		transition: box-shadow 0.15s ease;
	}

	.dock-item:hover .dock-icon {
		box-shadow: 0 4px 16px rgba(0, 0, 0, 0.15);
	}

	.dock-icon-svg {
		width: calc(var(--dock-icon-size, 48px) * 0.5);
		height: calc(var(--dock-icon-size, 48px) * 0.5);
	}

	.dock-icon-image {
		width: calc(var(--dock-icon-size, 48px) * 0.625);
		height: calc(var(--dock-icon-size, 48px) * 0.625);
		object-fit: contain;
		border-radius: 7px;
	}

	.dock-icon.terminal {
		background: #1E1E1E !important;
	}

	.dock-icon.finder {
		border-radius: 10px;
		box-shadow:
			0 2px 8px rgba(0, 0, 0, 0.15),
			inset 0 1px 0 rgba(255, 255, 255, 0.2);
	}

	.finder-face {
		width: 32px;
		height: 32px;
	}

	.terminal-prompt {
		font-family: 'SF Mono', 'Monaco', 'Inconsolata', 'Fira Code', 'Courier New', monospace;
		font-size: 16px;
		font-weight: bold;
		color: #00FF00;
		text-shadow: 0 0 8px rgba(0, 255, 0, 0.5);
	}

	.dock-indicator {
		width: 4px;
		height: 4px;
		border-radius: 50%;
		background: #333;
		margin-top: 4px;
	}

	.dock-tooltip {
		position: absolute;
		bottom: 100%;
		left: 50%;
		transform: translateX(-50%);
		margin-bottom: 8px;
		padding: 6px 12px;
		background: rgba(30, 30, 30, 0.95);
		color: white;
		font-size: 12px;
		font-weight: 500;
		border-radius: 6px;
		white-space: nowrap;
		pointer-events: none;
		box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
		z-index: 10010;
	}

	.dock-tooltip::after {
		content: '';
		position: absolute;
		top: 100%;
		left: 50%;
		transform: translateX(-50%);
		border: 6px solid transparent;
		border-top-color: rgba(30, 30, 30, 0.9);
	}

	/* ===== ICON STYLE VARIANTS ===== */

	/* Minimal */
	.dock-item.style-minimal .dock-icon {
		box-shadow: none;
		background: transparent !important;
	}

	.dock-item.style-minimal:hover .dock-icon {
		background: rgba(0, 0, 0, 0.08) !important;
	}

	/* Rounded */
	.dock-item.style-rounded .dock-icon {
		border-radius: 50%;
	}

	/* Square */
	.dock-item.style-square .dock-icon {
		border-radius: 4px;
	}

	/* macOS */
	.dock-item.style-macos .dock-icon {
		border-radius: 22%;
	}

	.dock-item.style-macos .dock-icon-svg {
		width: 28px;
		height: 28px;
	}

	/* Outlined */
	.dock-item.style-outlined .dock-icon {
		box-shadow: none;
	}

	.dock-item.style-outlined:hover .dock-icon {
		box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
	}

	/* Retro */
	.dock-item.style-retro .dock-icon {
		border-radius: 0;
		box-shadow:
			3px 3px 0 rgba(0, 0, 0, 0.3),
			inset -2px -2px 0 rgba(0, 0, 0, 0.2),
			inset 2px 2px 0 rgba(255, 255, 255, 0.3);
	}

	/* Win95 */
	.dock-item.style-win95 .dock-icon {
		border-radius: 0;
		box-shadow: none;
		border: 2px solid;
		border-color: #DFDFDF #808080 #808080 #DFDFDF;
		background: #C0C0C0 !important;
	}

	.dock-item.style-win95:hover .dock-icon {
		border-color: #808080 #DFDFDF #DFDFDF #808080;
	}

	/* Glassmorphism */
	.dock-item.style-glassmorphism .dock-icon {
		background: rgba(255, 255, 255, 0.2) !important;
		backdrop-filter: blur(10px);
		-webkit-backdrop-filter: blur(10px);
		border: 1px solid rgba(255, 255, 255, 0.3);
		box-shadow: 0 4px 16px rgba(0, 0, 0, 0.1);
	}

	.dock-item.style-glassmorphism:hover .dock-icon {
		background: rgba(255, 255, 255, 0.3) !important;
	}

	/* Neon */
	.dock-item.style-neon .dock-icon {
		background: #1a1a2e !important;
		box-shadow:
			0 0 8px currentColor,
			0 0 16px currentColor,
			inset 0 0 8px rgba(255, 255, 255, 0.1);
		border: 1px solid currentColor;
	}

	.dock-item.style-neon:hover .dock-icon {
		box-shadow:
			0 0 12px currentColor,
			0 0 24px currentColor,
			0 0 36px currentColor,
			inset 0 0 8px rgba(255, 255, 255, 0.1);
	}

	.dock-item.style-neon .dock-icon-svg {
		filter: drop-shadow(0 0 3px currentColor);
	}

	/* Flat */
	.dock-item.style-flat .dock-icon {
		box-shadow: none;
		border-radius: 8px;
	}

	.dock-item.style-flat:hover .dock-icon {
		filter: brightness(0.95);
	}

	/* Gradient */
	.dock-item.style-gradient .dock-icon {
		background: linear-gradient(135deg, var(--gradient-start, #667eea) 0%, var(--gradient-end, #764ba2) 100%) !important;
		box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2);
	}

	.dock-item.style-gradient .dock-icon-svg {
		stroke: white !important;
		filter: drop-shadow(0 1px 2px rgba(0, 0, 0, 0.2));
	}

	.dock-item.style-gradient:hover .dock-icon {
		box-shadow: 0 6px 16px rgba(0, 0, 0, 0.25);
	}

	/* macOS Classic */
	.dock-item.style-macos-classic .dock-icon {
		border-radius: 4px;
		background: linear-gradient(180deg, #EAEAEA 0%, #D4D4D4 50%, #C4C4C4 100%) !important;
		border: 1px solid;
		border-color: #FFFFFF #888888 #888888 #FFFFFF;
		box-shadow:
			1px 1px 0 #666666,
			inset 1px 1px 0 rgba(255, 255, 255, 0.8);
	}

	.dock-item.style-macos-classic:hover .dock-icon {
		background: linear-gradient(180deg, #F0F0F0 0%, #E0E0E0 50%, #D0D0D0 100%) !important;
	}

	/* Paper */
	.dock-item.style-paper .dock-icon {
		background: var(--dbg) !important;
		border-radius: 8px;
		box-shadow:
			0 1px 3px rgba(0, 0, 0, 0.08),
			0 4px 12px rgba(0, 0, 0, 0.05);
		border: 1px solid rgba(0, 0, 0, 0.06);
	}

	.dock-item.style-paper:hover .dock-icon {
		box-shadow:
			0 2px 8px rgba(0, 0, 0, 0.1),
			0 8px 24px rgba(0, 0, 0, 0.08);
		transform: translateY(-2px);
	}

	/* Pixel */
	.dock-item.style-pixel .dock-icon {
		border-radius: 0;
		image-rendering: pixelated;
		box-shadow:
			3px 0 0 #000,
			-3px 0 0 #000,
			0 3px 0 #000,
			0 -3px 0 #000;
		border: none;
	}

	.dock-item.style-pixel:hover .dock-icon {
		box-shadow:
			3px 0 0 #333,
			-3px 0 0 #333,
			0 3px 0 #333,
			0 -3px 0 #333;
		filter: brightness(1.1);
	}

	/* Frosted */
	.dock-item.style-frosted .dock-icon {
		background: rgba(255, 255, 255, 0.6) !important;
		backdrop-filter: blur(12px) saturate(180%);
		-webkit-backdrop-filter: blur(12px) saturate(180%);
		border-radius: 14px;
		border: 1px solid rgba(255, 255, 255, 0.4);
		box-shadow: 0 4px 16px rgba(0, 0, 0, 0.08);
	}

	.dock-item.style-frosted:hover .dock-icon {
		background: rgba(255, 255, 255, 0.75) !important;
		box-shadow: 0 6px 20px rgba(0, 0, 0, 0.12);
	}

	/* Terminal */
	.dock-item.style-terminal .dock-icon {
		background: #0a0a0a !important;
		border-radius: 4px;
		border: 1px solid #00ff00;
		box-shadow: 0 0 8px rgba(0, 255, 0, 0.3);
	}

	.dock-item.style-terminal:hover .dock-icon {
		box-shadow: 0 0 12px rgba(0, 255, 0, 0.5);
	}

	.dock-item.style-terminal .dock-icon-svg {
		color: #00ff00 !important;
		filter: drop-shadow(0 0 2px #00ff00);
	}

	/* Glow */
	.dock-item.style-glow .dock-icon {
		border-radius: 14px;
		box-shadow:
			0 0 15px currentColor,
			0 0 30px rgba(100, 100, 255, 0.3);
		border: none;
	}

	.dock-item.style-glow:hover .dock-icon {
		box-shadow:
			0 0 20px currentColor,
			0 0 40px rgba(100, 100, 255, 0.4);
		transform: scale(1.02);
	}

	.dock-item.style-glow .dock-icon-svg {
		filter: drop-shadow(0 0 3px currentColor);
	}

	/* Brutalist */
	.dock-item.style-brutalist .dock-icon {
		background: var(--dbg) !important;
		border-radius: 0;
		border: 3px solid #000;
		box-shadow: 4px 4px 0 #000;
	}

	.dock-item.style-brutalist:hover .dock-icon {
		transform: translate(-2px, -2px);
		box-shadow: 6px 6px 0 #000;
	}

	.dock-item.style-brutalist .dock-icon-svg {
		color: #000 !important;
	}

	/* Depth */
	.dock-item.style-depth .dock-icon {
		border-radius: 12px;
		border: none;
		box-shadow:
			0 2px 4px rgba(0, 0, 0, 0.1),
			0 4px 8px rgba(0, 0, 0, 0.1),
			0 8px 16px rgba(0, 0, 0, 0.08);
	}

	.dock-item.style-depth:hover .dock-icon {
		transform: translateY(-3px);
		box-shadow:
			0 4px 8px rgba(0, 0, 0, 0.12),
			0 8px 16px rgba(0, 0, 0, 0.1),
			0 16px 32px rgba(0, 0, 0, 0.08);
	}

	/* ===== DARK MODE ===== */
	:global(.dark) .dock-indicator {
		background: #fff;
	}

	:global(.dark) .dock-tooltip {
		background: rgba(44, 44, 46, 0.98);
		border: 1px solid rgba(255, 255, 255, 0.12);
	}

	:global(.dark) .dock-tooltip::after {
		border-top-color: rgba(44, 44, 46, 0.98);
	}

	:global(.dark) .dock-item.style-glassmorphism .dock-icon {
		background: rgba(255, 255, 255, 0.1) !important;
		border-color: rgba(255, 255, 255, 0.15);
	}

	:global(.dark) .dock-item.style-glassmorphism:hover .dock-icon {
		background: rgba(255, 255, 255, 0.15) !important;
	}

	:global(.dark) .dock-item.style-minimal:hover .dock-icon {
		background: rgba(255, 255, 255, 0.1) !important;
	}

	:global(.dark) .dock-item.style-paper .dock-icon {
		background: #3a3a3c !important;
		border-color: rgba(255, 255, 255, 0.12);
	}
</style>
