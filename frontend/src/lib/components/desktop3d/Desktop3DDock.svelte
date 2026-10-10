<script lang="ts">
	import { MODULE_INFO, ALL_MODULES, CORE_MODULES, type ModuleId, type Window3DState } from '$lib/stores/desktop3dStore';
	import { desktopSettings } from '$lib/stores/desktopStore';

	interface Props {
		windows: Window3DState[];
		focusedWindowId: string | null;
		onSelect?: (module: ModuleId) => void;
		onUnfocus?: () => void;
	}

	let {
		windows = [],
		focusedWindowId = null,
		onSelect,
		onUnfocus
	}: Props = $props();

	// Get icon style from desktop settings
	const iconStyle = $derived($desktopSettings.iconStyle);

	// Hover state for tooltips
	let hoveredModule: ModuleId | null = $state(null);

	// Icon SVG paths for each module
	const moduleIcons: Record<string, { path: string; bgColor: string }> = {
		dashboard: {
			path: 'M4 5a1 1 0 011-1h4a1 1 0 011 1v5a1 1 0 01-1 1H5a1 1 0 01-1-1V5zm10 0a1 1 0 011-1h4a1 1 0 011 1v2a1 1 0 01-1 1h-4a1 1 0 01-1-1V5zm0 6a1 1 0 011-1h4a1 1 0 011 1v5a1 1 0 01-1 1h-4a1 1 0 01-1-1v-5zm-10 1a1 1 0 011-1h4a1 1 0 011 1v3a1 1 0 01-1 1H5a1 1 0 01-1-1v-3z',
			bgColor: 'var(--bos-status-info)'
		},
		chat: {
			path: 'M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z',
			bgColor: 'var(--bos-status-success)'
		},
		tasks: {
			path: 'M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4',
			bgColor: 'var(--bos-status-warning)'
		},
		projects: {
			path: 'M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z',
			bgColor: 'var(--bos-category-ai)'
		},
		team: {
			path: 'M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z',
			bgColor: 'var(--bos-status-info)'
		},
		clients: {
			path: 'M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4',
			bgColor: 'var(--bos-category-ai)'
		},
		calendar: {
			path: 'M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z',
			bgColor: 'var(--bos-category-communication)'
		},
		contexts: {
			path: 'M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10',
			bgColor: 'var(--bos-category-ai)'
		},
		nodes: {
			path: 'M4 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2V6zM14 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V6zM4 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2v-2zM14 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z',
			bgColor: 'var(--bos-status-error)'
		},
		daily: {
			path: 'M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z',
			bgColor: 'var(--bos-status-info)'
		},
		settings: {
			path: 'M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z M15 12a3 3 0 11-6 0 3 3 0 016 0z',
			bgColor: 'var(--dt2)'
		},
		knowledge: {
			path: 'M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z',
			bgColor: 'var(--bos-category-ai)'
		},
		terminal: {
			path: 'M4 17l6-6-6-6M12 19h8',
			bgColor: 'var(--dt2)'
		},
		files: {
			path: 'M3 7V17C3 18.1046 3.89543 19 5 19H19C20.1046 19 21 18.1046 21 17V9C21 7.89543 20.1046 7 19 7H12L10 5H5C3.89543 5 3 5.89543 3 7Z M7 13h10M7 16h6',
			bgColor: 'var(--bos-status-info)'
		},
		help: {
			path: 'M8.228 9c.549-1.165 2.03-2 3.772-2 2.21 0 4 1.343 4 3 0 1.4-1.278 2.575-3.006 2.907-.542.104-.994.54-.994 1.093m0 3h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z',
			bgColor: 'var(--bos-status-info)'
		},
		agents: {
			path: 'M12 7c-1.657 0-3 .895-3 2s1.343 2 3 2 3-.895 3-2-1.343-2-3-2zm0 0V4m0 3v3M7.5 17.5c0-1.38 2.015-2.5 4.5-2.5s4.5 1.12 4.5 2.5V21H7.5v-3.5z',
			bgColor: 'var(--bos-category-ai)'
		},
		crm: {
			path: 'M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4',
			bgColor: 'var(--bos-category-automation)'
		},
		integrations: {
			path: 'M8 12l-4-4m0 0l4-4m-4 4h16m-8 8l4-4m0 0l-4-4',
			bgColor: 'var(--bos-category-productivity)'
		},
		notifications: {
			path: 'M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9',
			bgColor: 'var(--bos-status-error)'
		},
		profile: {
			path: 'M5.121 17.804A13.937 13.937 0 0112 16c2.5 0 4.847.655 6.879 1.804M15 10a3 3 0 11-6 0 3 3 0 016 0zm6 2a9 9 0 11-18 0 9 9 0 0118 0z',
			bgColor: 'var(--bos-status-info)'
		},
		'voice-notes': {
			path: 'M19 11a7 7 0 01-7 7m0 0a7 7 0 01-7-7m7 7v4m0 0H8m4 0h4m-4-8a3 3 0 01-3-3V5a3 3 0 116 0v6a3 3 0 01-3 3z',
			bgColor: 'var(--bos-category-communication)'
		},
		usage: {
			path: 'M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z',
			bgColor: 'var(--dt2)'
		},
		tables: {
			path: 'M3 10h18M3 14h18m-9-4v8m-7 0h14a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z',
			bgColor: 'var(--bos-category-productivity)'
		},
		communication: {
			path: 'M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z',
			bgColor: 'var(--bos-status-error)'
		},
		pages: {
			path: 'M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z',
			bgColor: 'var(--bos-status-success)'
		}
	};

	// Smart dock: Show 8 modules (core + recently focused)
	const MAX_DOCK_MODULES = 8;
	const ALWAYS_SHOW: ModuleId[] = ['dashboard', 'knowledge', 'tasks', 'settings'];

	let dockModules = $derived.by(() => {
		// Start with always-show modules
		const modules: ModuleId[] = [...ALWAYS_SHOW];

		// Get recently focused modules (sorted by lastFocused time)
		const recentModules = [...windows]
			.filter(w => w.lastFocused > 0 && !ALWAYS_SHOW.includes(w.module))
			.sort((a, b) => b.lastFocused - a.lastFocused)
			.slice(0, MAX_DOCK_MODULES - ALWAYS_SHOW.length)
			.map(w => w.module);

		// Add recent modules
		recentModules.forEach(m => {
			if (!modules.includes(m)) {
				modules.push(m);
			}
		});

		// If we don't have enough, fill with core modules
		if (modules.length < MAX_DOCK_MODULES) {
			CORE_MODULES.forEach(m => {
				if (modules.length < MAX_DOCK_MODULES && !modules.includes(m)) {
					modules.push(m);
				}
			});
		}

		return modules.slice(0, MAX_DOCK_MODULES);
	});

	// Check if a module is currently focused
	function isModuleFocused(module: ModuleId): boolean {
		const window = windows.find(w => w.module === module);
		return window?.id === focusedWindowId;
	}

	// Handle dock item click
	function handleClick(module: ModuleId) {
		onSelect?.(module);
	}

	// Get icon for module
	function getIcon(module: ModuleId) {
		return moduleIcons[module] || { path: 'M12 4v16m8-8H4', bgColor: 'var(--dt2)' };
	}
</script>

<div class="dock-container">
	<div class="dock">
		{#each dockModules as module (module)}
			{@const info = MODULE_INFO[module]}
			{@const icon = getIcon(module)}
			{@const focused = isModuleFocused(module)}
			<button
				class="dock-item style- {iconStyle}"
				class:focused={focused}
				onclick={() => handleClick(module)}
				onmouseenter={() => hoveredModule = module}
				onmouseleave={() => hoveredModule = null}
			>
				<div class="dock-icon" style="background-color: {icon.bgColor}">
					<svg class="dock-icon-svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
						<path d={icon.path} />
					</svg>
				</div>
				{#if focused}
					<div class="dock-indicator"></div>
				{/if}
				<!-- Tooltip -->
				{#if hoveredModule === module}
					<div class="dock-tooltip">{info.title}</div>
				{/if}
			</button>
		{/each}
	</div>
</div>

<style>
	.dock-container {
		position: fixed;
		bottom: 20px;
		left: 50%;
		transform: translateX(-50%);
		z-index: 100;
	}

	.dock {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 10px 16px;
		background: rgba(255, 255, 255, 0.85);
		backdrop-filter: blur(20px);
		border: 1px solid rgba(0, 0, 0, 0.08);
		border-radius: 20px;
		box-shadow: 0 10px 40px rgba(0, 0, 0, 0.1);
	}

	.dock-item {
		position: relative;
		display: flex;
		flex-direction: column;
		align-items: center;
		padding: 0;
		background: none;
		border: none;
		cursor: pointer;
		transition: transform 0.2s;
	}

	.dock-item:hover {
		transform: translateY(-8px) scale(1.1);
	}

	.dock-item.focused {
		transform: translateY(-4px);
	}

	.dock-icon {
		width: 48px;
		height: 48px;
		border-radius: 12px;
		display: flex;
		align-items: center;
		justify-content: center;
		box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
		transition: box-shadow 0.2s;
	}

	.dock-icon-svg {
		width: 24px;
		height: 24px;
		stroke: white;
	}

	.dock-item:hover .dock-icon {
		box-shadow: 0 8px 24px rgba(0, 0, 0, 0.2);
	}

	.dock-item.focused .dock-icon {
		box-shadow: 0 4px 20px rgba(74, 158, 255, 0.5);
	}

	.dock-indicator {
		position: absolute;
		bottom: -6px;
		width: 6px;
		height: 6px;
		background: var(--dt);
		border-radius: 50%;
	}

	/* Tooltip */
	.dock-tooltip {
		position: absolute;
		bottom: 100%;
		left: 50%;
		transform: translateX(-50%);
		margin-bottom: 12px;
		padding: 8px 14px;
		background: rgba(30, 30, 30, 0.95);
		color: white;
		font-size: 13px;
		font-weight: 500;
		border-radius: 8px;
		white-space: nowrap;
		pointer-events: none;
		box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
		z-index: 1000;
		animation: tooltipFadeIn 0.15s ease;
	}

	.dock-tooltip::after {
		content: '';
		position: absolute;
		top: 100%;
		left: 50%;
		transform: translateX(-50%);
		border: 6px solid transparent;
		border-top-color: rgba(30, 30, 30, 0.95);
	}

	@keyframes tooltipFadeIn {
		from { opacity: 0; transform: translateX(-50%) translateY(4px); }
		to { opacity: 1; transform: translateX(-50%) translateY(0); }
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

	.dock-item.style-minimal .dock-icon-svg {
		stroke: var(--dt);
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
		width: 52px;
		height: 52px;
	}

	.dock-item.style-macos .dock-icon-svg {
		width: 28px;
		height: 28px;
	}

	/* Outlined */
	.dock-item.style-outlined .dock-icon {
		box-shadow: none;
		background: transparent !important;
		border: 2px solid currentColor;
	}

	.dock-item.style-outlined .dock-icon-svg {
		stroke: var(--dt);
	}

	/* Glassmorphism */
	.dock-item.style-glassmorphism .dock-icon {
		background: rgba(255, 255, 255, 0.2) !important;
		backdrop-filter: blur(8px);
		border: 1px solid rgba(255, 255, 255, 0.3);
	}

	/* Paper */
	.dock-item.style-paper .dock-icon {
		background: white !important;
		border: 1px solid rgba(0, 0, 0, 0.1);
		box-shadow: 2px 2px 0 rgba(0, 0, 0, 0.1);
	}

	.dock-item.style-paper .dock-icon-svg {
		stroke: var(--dt);
	}

	/* ===== DARK MODE STYLES ===== */
	:global(.dark) .dock {
		background: rgba(44, 44, 46, 0.85);
		border-color: rgba(255, 255, 255, 0.12);
		box-shadow:
			0 0 0 0.5px rgba(255, 255, 255, 0.08),
			0 8px 32px rgba(0, 0, 0, 0.4);
	}

	:global(.dark) .dock-indicator {
		background: var(--dt);
	}

	:global(.dark) .dock-tooltip {
		background: rgba(44, 44, 46, 0.98);
		border: 1px solid rgba(255, 255, 255, 0.12);
	}

	:global(.dark) .dock-tooltip::after {
		border-top-color: rgba(44, 44, 46, 0.98);
	}

	/* Dark mode style variants */
	:global(.dark) .dock-item.style-minimal .dock-icon-svg {
		stroke: var(--dt);
	}

	:global(.dark) .dock-item.style-minimal:hover .dock-icon {
		background: rgba(255, 255, 255, 0.1) !important;
	}

	:global(.dark) .dock-item.style-outlined .dock-icon-svg {
		stroke: var(--dt);
	}

	:global(.dark) .dock-item.style-glassmorphism .dock-icon {
		background: rgba(255, 255, 255, 0.1) !important;
		border-color: rgba(255, 255, 255, 0.2);
	}

	:global(.dark) .dock-item.style-paper .dock-icon {
		background: var(--dbg2) !important;
		border-color: rgba(255, 255, 255, 0.1);
		box-shadow: 2px 2px 0 rgba(0, 0, 0, 0.3);
	}

	:global(.dark) .dock-item.style-paper .dock-icon-svg {
		stroke: var(--dt);
	}
</style>
