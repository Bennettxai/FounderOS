<script lang="ts">
	import { windowStore } from '$lib/stores/windowStore';
	import { desktopSettings } from '$lib/stores/desktopStore';
	import OsaPill from './osa/OsaPill.svelte';
	import DockItem from './DockItem.svelte';
	import { iconPaths } from './iconPaths';
	import { onMount, onDestroy } from 'svelte';
	import { browser } from '$app/environment';
	import type { CustomIconConfig } from '$lib/stores/windowStore';

	let osaPillRef: OsaPill | undefined = $state(undefined);
	let viewportWidth = $state(1440);

	function measureViewport() {
		if (browser) viewportWidth = window.innerWidth;
	}

	const iconStyle = $derived($desktopSettings.iconStyle);
	const iconLibrary = $derived($desktopSettings.iconLibrary);

	// Different libraries have different visual styles
	const libraryStrokeWidth = $derived({
		lucide: 2,
		phosphor: 3,
		tabler: 1.2,
		heroicons: 2.5
	}[iconLibrary] || 2);

	const libraryLineCap = $derived<'round' | 'square' | 'butt'>(
		iconLibrary === 'tabler' ? 'square' : 'round'
	);

	const libraryLineJoin = $derived<'round' | 'miter' | 'bevel'>(
		iconLibrary === 'tabler' ? 'miter' : 'round'
	);

	const libraryIconScale = $derived({
		lucide: 1,
		phosphor: 1.25,
		tabler: 0.85,
		heroicons: 1.15
	}[iconLibrary] || 1);

	const librarySvgFilter = $derived({
		lucide: 'none',
		phosphor: 'drop-shadow(0 2px 3px rgba(0,0,0,0.25))',
		tabler: 'saturate(0.7)',
		heroicons: 'drop-shadow(0 1px 2px rgba(0,0,0,0.2)) saturate(1.2)'
	}[iconLibrary] || 'none');

	// Global keyboard handler for Ctrl+K OSA focus
	function handleGlobalKeydown(e: KeyboardEvent) {
		if (e.ctrlKey && e.key === 'k') {
			e.preventDefault();
			osaPillRef?.focusInput();
		}
	}

	onMount(() => {
		if (browser) {
			window.addEventListener('keydown', handleGlobalKeydown);
			measureViewport();
			window.addEventListener('resize', measureViewport);
		}
	});

	onDestroy(() => {
		if (browser) {
			window.removeEventListener('keydown', handleGlobalKeydown);
			window.removeEventListener('resize', measureViewport);
		}
	});

	interface DockItem {
		id: string;
		module: string;
		label: string;
		isOpen: boolean;
		isMinimized: boolean;
		windowId?: string;
		folderId?: string;
		folderColor?: string;
		customIcon?: CustomIconConfig;
		appUrl?: string;
		launchMode?: 'iframe' | 'browser' | 'external';
	}

	// Icon data for each module
	const moduleIcons: Record<string, { path: string; color: string; bgColor: string; isTerminal?: boolean; isFolder?: boolean; isFinder?: boolean; imageUrl?: string }> = {
		platform: {
			path: 'M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5',
			color: '#333333',
			bgColor: '#F5F5F5'
		},
		folder: {
			path: 'M3 7V17C3 18.1046 3.89543 19 5 19H19C20.1046 19 21 18.1046 21 17V9C21 7.89543 20.1046 7 19 7H12L10 5H5C3.89543 5 3 5.89543 3 7Z',
			color: '#3B82F6',
			bgColor: '#EFF6FF',
			isFolder: true
		},
		terminal: {
			path: 'M4 17l6-6-6-6M12 19h8',
			color: '#00FF00',
			bgColor: '#1E1E1E',
			isTerminal: true
		},
		dashboard: {
			path: 'M4 5a1 1 0 011-1h4a1 1 0 011 1v5a1 1 0 01-1 1H5a1 1 0 01-1-1V5zm10 0a1 1 0 011-1h4a1 1 0 011 1v2a1 1 0 01-1 1h-4a1 1 0 01-1-1V5zm0 6a1 1 0 011-1h4a1 1 0 011 1v5a1 1 0 01-1 1h-4a1 1 0 01-1-1v-5zm-10 1a1 1 0 011-1h4a1 1 0 011 1v3a1 1 0 01-1 1H5a1 1 0 01-1-1v-3z',
			color: '#1E88E5',
			bgColor: '#E3F2FD'
		},
		chat: {
			path: 'M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z',
			color: '#43A047',
			bgColor: '#E8F5E9'
		},
		tasks: {
			path: 'M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4',
			color: '#FB8C00',
			bgColor: '#FFF3E0'
		},
		projects: {
			path: 'M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z',
			color: '#8E24AA',
			bgColor: '#F3E5F5'
		},
		team: {
			path: 'M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z',
			color: '#00ACC1',
			bgColor: '#E0F7FA'
		},
		clients: {
			path: 'M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4',
			color: '#7B1FA2',
			bgColor: '#F3E5F5'
		},
		contexts: {
			path: 'M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10',
			color: '#5E35B1',
			bgColor: '#EDE7F6'
		},
		nodes: {
			path: 'M4 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2V6zM14 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V6zM4 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2v-2zM14 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z',
			color: '#E53935',
			bgColor: '#FFEBEE'
		},
		daily: {
			path: 'M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z',
			color: '#039BE5',
			bgColor: '#E1F5FE'
		},
		settings: {
			path: 'M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z M15 12a3 3 0 11-6 0 3 3 0 016 0z',
			color: '#546E7A',
			bgColor: '#ECEFF1'
		},
		trash: {
			path: 'M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16',
			color: '#78909C',
			bgColor: '#ECEFF1'
		},
		files: {
			path: 'M3 7V17C3 18.1046 3.89543 19 5 19H19C20.1046 19 21 18.1046 21 17V9C21 7.89543 20.1046 7 19 7H12L10 5H5C3.89543 5 3 5.89543 3 7Z M7 13h10M7 16h6',
			color: '#2196F3',
			bgColor: '#E3F2FD'
		},
		calendar: {
			path: 'M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z',
			color: '#E91E63',
			bgColor: '#FCE4EC'
		},
		'ai-settings': {
			path: 'M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z',
			color: '#9C27B0',
			bgColor: '#F3E5F5'
		},
		help: {
			path: 'M8.228 9c.549-1.165 2.03-2 3.772-2 2.21 0 4 1.343 4 3 0 1.4-1.278 2.575-3.006 2.907-.542.104-.994.54-.994 1.093m0 3h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z',
			color: '#0EA5E9',
			bgColor: '#E0F2FE'
		},
		finder: {
			path: 'M5 3h14a2 2 0 012 2v14a2 2 0 01-2 2H5a2 2 0 01-2-2V5a2 2 0 012-2z',
			color: '#1C9BF6',
			bgColor: 'linear-gradient(180deg, #3FBBF7 0%, #1C7FE6 100%)',
			isFinder: true
		}
	};

	const moduleLabels: Record<string, string> = {
		platform: 'Business OS',
		terminal: 'Terminal',
		dashboard: 'Command',
		chat: 'Chat',
		agents: 'Agents',
		knowledge: 'Knowledge',
		intelligence: 'Intelligence',
		inbox: 'Inbox',
		calendar: 'Calendar',
		relationships: 'Relationships',
		tasks: 'Tasks',
		projects: 'Projects',
		rhythm: 'Rhythm',
		pipelines: 'Pipelines',
		offers: 'Offers',
		campaigns: 'Campaigns',
		sites: 'Sites',
		personas: 'Personas',
		content: 'Content',
		apps: 'Apps',
		assets: 'Assets',
		deliverables: 'Deliverables',
		engines: 'Engines',
		builders: 'Builders',
		drive: 'Drive',
		finance: 'Finance',
		analytics: 'Analytics',
		data: 'Data',
		connectors: 'Connectors',
		resources: 'Resources',
		notifications: 'Notifications',
		admin: 'Admin',
		team: 'Team',
		clients: 'Clients',
		contexts: 'Contexts',
		nodes: 'Nodes',
		daily: 'Daily Log',
		settings: 'Settings',
		trash: 'Trash',
		files: 'Files',
		'ai-settings': 'AI Settings',
		help: 'Help',
		finder: 'Finder'
	};

	let hoveredIndex = $state<number | null>(null);
	let isDragOver = $state(false);

	function handleDragOver(event: DragEvent) {
		event.preventDefault();
		if (event.dataTransfer) {
			event.dataTransfer.dropEffect = 'copy';
		}
		isDragOver = true;
	}

	function handleDragLeave() {
		isDragOver = false;
	}

	function handleDrop(event: DragEvent) {
		event.preventDefault();
		isDragOver = false;
		const module = event.dataTransfer?.getData('text/plain');
		const iconId = event.dataTransfer?.getData('text/icon-id');

		if (module && module !== 'trash' && module !== 'separator') {
			let moduleToAdd = module;

			if (module === 'folder' && iconId) {
				const icon = $windowStore.desktopIcons.find(i => i.id === iconId);
				if (icon?.type === 'folder' && icon.folderId) {
					moduleToAdd = `folder-${icon.folderId}`;
				}
			}

			if (!$windowStore.dockPinnedItems.includes(moduleToAdd)) {
				windowStore.addToDock(moduleToAdd);
			}
		}
	}

	function isFolder(module: string): boolean {
		return module === 'folder' || module.startsWith('folder-');
	}

	function getFolderData(module: string) {
		if (module === 'folder') return null;
		const folderId = module.replace('folder-', '');
		const folderIcon = $windowStore.desktopIcons.find(
			i => i.type === 'folder' && i.folderId === folderId
		);
		const folder = $windowStore.folders.find(f => f.id === folderId);
		return {
			folderId,
			label: folder?.name || folderIcon?.label || 'Folder',
			color: folder?.color || folderIcon?.folderColor || '#3B82F6'
		};
	}

	const dockItems = $derived(() => {
		const items: DockItem[] = [];

		for (const module of $windowStore.dockPinnedItems) {
			const windows = $windowStore.windows.filter(w => w.module === module);
			const isOpen = windows.length > 0;
			const minimizedWindow = windows.find(w => w.minimized);
			const folderData = isFolder(module) ? getFolderData(module) : null;
			const desktopIcon = $windowStore.desktopIcons.find(i => i.module === module);

			items.push({
				id: `pinned-${module}`,
				module,
				label: folderData?.label || desktopIcon?.label || moduleLabels[module] || module,
				isOpen,
				isMinimized: !!minimizedWindow,
				windowId: minimizedWindow?.id || windows[0]?.id,
				folderId: folderData?.folderId,
				folderColor: folderData?.color,
				customIcon: desktopIcon?.customIcon,
				appUrl: desktopIcon?.appUrl,
				launchMode: desktopIcon?.launchMode
			});
		}

		items.push({
			id: 'separator',
			module: 'separator',
			label: '',
			isOpen: false,
			isMinimized: false
		});

		const pinnedModules = new Set($windowStore.dockPinnedItems);
		const openWindowModules = [...new Set($windowStore.windows.map(w => w.module))];

		for (const module of openWindowModules) {
			if (!pinnedModules.has(module)) {
				const windows = $windowStore.windows.filter(w => w.module === module);
				const minimizedWindow = windows.find(w => w.minimized);
				const desktopIcon = $windowStore.desktopIcons.find(i => i.module === module);

				items.push({
					id: `open-${module}`,
					module,
					label: desktopIcon?.label || moduleLabels[module] || module,
					isOpen: true,
					isMinimized: !!minimizedWindow,
					windowId: minimizedWindow?.id || windows[0]?.id,
					customIcon: desktopIcon?.customIcon,
					appUrl: desktopIcon?.appUrl,
					launchMode: desktopIcon?.launchMode
				});
			}
		}

		if (!pinnedModules.has('trash')) {
			items.push({
				id: 'separator-2',
				module: 'separator',
				label: '',
				isOpen: false,
				isMinimized: false
			});
			items.push({
				id: 'trash',
				module: 'trash',
				label: 'Trash',
				isOpen: $windowStore.windows.some(w => w.module === 'trash'),
				isMinimized: $windowStore.windows.some(w => w.module === 'trash' && w.minimized),
				windowId: $windowStore.windows.find(w => w.module === 'trash')?.id
			});
		}

		return items;
	});

	const dockIconSize = $derived.by(() => {
		const items = dockItems();
		const iconCount = Math.max(1, items.filter(item => item.module !== 'separator').length);
		const separatorCount = items.length - iconCount;
		const available = Math.max(240, viewportWidth - 32 - separatorCount * 12 - 20);
		return Math.max(20, Math.min(48, Math.floor(available / iconCount) - 8));
	});

	function handleItemClick(item: DockItem) {
		if (item.module === 'separator') return;

		if (item.isMinimized && item.windowId) {
			windowStore.restoreWindow(item.windowId);
		} else if (item.isOpen && item.windowId) {
			windowStore.focusWindow(item.windowId);
		} else if (item.folderId) {
			windowStore.openFolder(item.folderId);
		} else if (item.module.startsWith('workspace-app-') && item.appUrl) {
			windowStore.openWindow(item.module, {
				title: item.label,
				data: { url: item.appUrl, launchMode: item.launchMode || 'iframe' }
			});
		} else {
			windowStore.openWindow(item.module);
		}
	}

	function getItemIcon(item: DockItem) {
		if (item.folderId) {
			return {
				path: moduleIcons.folder.path,
				color: item.folderColor || '#3B82F6',
				bgColor: `${item.folderColor || '#3B82F6'}20`,
				isFolder: true
			};
		}
		if (item.customIcon?.type === 'image' && item.customIcon.imageUrl) {
			return {
				path: '',
				color: item.customIcon.foregroundColor || '#111827',
				bgColor: item.customIcon.backgroundColor || '#ffffff',
				imageUrl: item.customIcon.imageUrl
			};
		}
		return moduleIcons[item.module] || iconPaths[item.module] || moduleIcons.dashboard;
	}

	function getScale(index: number): number {
		if (hoveredIndex === null) return 1;
		const distance = Math.abs(index - hoveredIndex);
		if (distance === 0) return 1.4;
		if (distance === 1) return 1.2;
		if (distance === 2) return 1.1;
		return 1;
	}

	function getTranslateY(index: number): number {
		if (hoveredIndex === null) return 0;
		const distance = Math.abs(index - hoveredIndex);
		if (distance === 0) return -12;
		if (distance === 1) return -6;
		if (distance === 2) return -2;
		return 0;
	}
</script>

<div class="dock-container">
	<!-- OSA Interface - always visible above dock -->
	<OsaPill bind:this={osaPillRef} />

	<div
		class="dock"
		class:drag-over={isDragOver}
		ondragover={handleDragOver}
		ondragleave={handleDragLeave}
		ondrop={handleDrop}
		role="toolbar"
		aria-label="Application dock"
		tabindex="0"
	>
		{#each dockItems() as item, index (item.id)}
			{#if item.module === 'separator'}
				<div class="dock-separator"></div>
			{:else}
				{@const icon = getItemIcon(item)}
				<DockItem
					{item}
					{index}
					{hoveredIndex}
					{iconStyle}
					{iconLibrary}
					{libraryStrokeWidth}
					{libraryLineCap}
					{libraryLineJoin}
					{libraryIconScale}
					{librarySvgFilter}
					baseSize={dockIconSize}
					scale={getScale(index)}
					translateY={getTranslateY(index)}
					{icon}
					onclick={() => handleItemClick(item)}
					onmouseenter={() => (hoveredIndex = index)}
					onmouseleave={() => (hoveredIndex = null)}
				/>
			{/if}
		{/each}
	</div>
</div>

<style>
	.dock-container {
		position: fixed;
		bottom: 8px;
		left: 50%;
		transform: translateX(-50%);
		z-index: 9999;
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 16px;
		pointer-events: none;
		max-width: calc(100vw - 16px);
	}

	.dock {
		display: flex;
		align-items: flex-end;
		gap: 4px;
		padding: 6px 10px 8px;
		background: rgba(255, 255, 255, 0.75);
		backdrop-filter: blur(20px);
		-webkit-backdrop-filter: blur(20px);
		border: 1px solid rgba(255, 255, 255, 0.6);
		border-radius: 16px;
		pointer-events: auto;
		box-shadow:
			0 0 0 0.5px rgba(0, 0, 0, 0.1),
			0 8px 32px rgba(0, 0, 0, 0.12);
		transition: all 0.2s ease;
		position: relative;
		z-index: 10;
		max-width: calc(100vw - 16px);
	}

	.dock.drag-over {
		background: rgba(0, 102, 255, 0.15);
		border-color: rgba(0, 102, 255, 0.5);
		box-shadow:
			0 0 0 2px rgba(0, 102, 255, 0.3),
			0 8px 32px rgba(0, 0, 0, 0.15);
	}

	.dock-separator {
		width: 1px;
		height: 40px;
		background: rgba(0, 0, 0, 0.15);
		margin: 0 4px;
		align-self: center;
	}

	/* ===== DARK MODE ===== */
	:global(.dark) .dock {
		background: rgba(44, 44, 46, 0.85);
		border-color: rgba(255, 255, 255, 0.12);
		box-shadow:
			0 0 0 0.5px rgba(255, 255, 255, 0.08),
			0 8px 32px rgba(0, 0, 0, 0.4);
	}

	:global(.dark) .dock.drag-over {
		background: rgba(59, 130, 246, 0.15);
		border-color: rgba(59, 130, 246, 0.5);
	}

	:global(.dark) .dock-separator {
		background: rgba(255, 255, 255, 0.15);
	}
</style>
