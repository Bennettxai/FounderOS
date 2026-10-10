<script lang="ts">
	import { editor, blockTypes, filterBlockTypes, getBlockTypesBySection, type BlockType, type BlockTypeDefinition } from '$lib/stores/editor';
	import { fly } from 'svelte/transition';

	let selectedIndex = $state(0);
	let showIconPicker = $state(false);
	let selectedPageIcon = $state('document');

	// Icon presets - same as in SidebarPageItem
	const iconPresets = [
		{ id: 'document', path: 'M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z' },
		{ id: 'folder', path: 'M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z' },
		{ id: 'clipboard', path: 'M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2' },
		{ id: 'chart', path: 'M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z' },
		{ id: 'briefcase', path: 'M21 13.255A23.931 23.931 0 0112 15c-3.183 0-6.22-.62-9-1.745M16 6V4a2 2 0 00-2-2h-4a2 2 0 00-2 2v2m4 6h.01M5 20h14a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z' },
		{ id: 'user', path: 'M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z' },
		{ id: 'users', path: 'M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z' },
		{ id: 'building', path: 'M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4' },
		{ id: 'star', path: 'M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z' },
		{ id: 'target', path: 'M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z' },
		{ id: 'check', path: 'M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z' },
		{ id: 'heart', path: 'M4.318 6.318a4.5 4.5 0 000 6.364L12 20.364l7.682-7.682a4.5 4.5 0 00-6.364-6.364L12 7.636l-1.318-1.318a4.5 4.5 0 00-6.364 0z' },
		{ id: 'bookmark', path: 'M5 5a2 2 0 012-2h10a2 2 0 012 2v16l-7-3.5L5 21V5z' },
		{ id: 'flag', path: 'M3 21v-4m0 0V5a2 2 0 012-2h6.5l1 1H21l-3 6 3 6h-8.5l-1-1H5a2 2 0 00-2 2zm9-13.5V9' },
		{ id: 'home', path: 'M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6' },
		{ id: 'cog', path: 'M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z M15 12a3 3 0 11-6 0 3 3 0 016 0z' },
		{ id: 'lightbulb', path: 'M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z' },
		{ id: 'rocket', path: 'M15.59 14.37a6 6 0 01-5.84 7.38v-4.8m5.84-2.58a14.98 14.98 0 006.16-12.12A14.98 14.98 0 009.631 8.41m5.96 5.96a14.926 14.926 0 01-5.841 2.58m-.119-8.54a6 6 0 00-7.381 5.84h4.8m2.581-5.84a14.927 14.927 0 00-2.58 5.84m2.699 2.7c-.103.021-.207.041-.311.06a15.09 15.09 0 01-2.448-2.448 14.9 14.9 0 01.06-.312m-2.24 2.39a4.493 4.493 0 00-1.757 4.306 4.493 4.493 0 004.306-1.758M16.5 9a1.5 1.5 0 11-3 0 1.5 1.5 0 013 0z' },
		{ id: 'chat', path: 'M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z' },
		{ id: 'calendar', path: 'M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z' },
		{ id: 'code', path: 'M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4' },
		{ id: 'database', path: 'M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4m0 5c0 2.21-3.582 4-8 4s-8-1.79-8-4' },
		{ id: 'terminal', path: 'M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z' },
		{ id: 'pencil', path: 'M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z' },
	];

	// Filter block types based on query using priority-based filtering
	let filteredTypes = $derived(
		$editor.slashMenuQuery
			? filterBlockTypes($editor.slashMenuQuery)
			: blockTypes
	);

	// Get sections for display (only when not filtering)
	let sections = $derived(
		$editor.slashMenuQuery
			? null
			: getBlockTypesBySection(filteredTypes)
	);

	// Flat list for keyboard navigation
	let flatList = $derived(
		$editor.slashMenuQuery
			? filteredTypes
			: [...(sections?.suggested || []), ...(sections?.basic || [])]
	);

	// Reset selection when query changes
	$effect(() => {
		if ($editor.slashMenuQuery !== undefined) {
			selectedIndex = 0;
		}
	});

	function selectBlockType(type: BlockType) {
		// Use the store's selectBlockType which sets pendingBlockTypeSelection
		// Block.svelte will watch this and handle the actual selection (including page creation)
		// For page type, pass the selected icon
		if (type === 'page') {
			editor.selectBlockType(type, { icon: selectedPageIcon });
		} else {
			editor.selectBlockType(type);
		}
		// Reset icon picker state
		showIconPicker = false;
		selectedPageIcon = 'document';
	}

	function handleIconClick(e: MouseEvent, blockType: BlockType) {
		e.stopPropagation();
		if (blockType === 'page') {
			showIconPicker = !showIconPicker;
		}
	}

	function selectIcon(iconId: string) {
		selectedPageIcon = iconId;
		showIconPicker = false;
	}

	function getIconPath(iconId: string): string {
		const preset = iconPresets.find(p => p.id === iconId);
		return preset?.path || iconPresets[0].path;
	}

	function handleKeydown(e: KeyboardEvent) {
		if (!$editor.showSlashMenu) return;

		if (e.key === 'ArrowDown') {
			e.preventDefault();
			selectedIndex = Math.min(selectedIndex + 1, flatList.length - 1);
			scrollSelectedIntoView();
		} else if (e.key === 'ArrowUp') {
			e.preventDefault();
			selectedIndex = Math.max(selectedIndex - 1, 0);
			scrollSelectedIntoView();
		} else if (e.key === 'Enter' || e.key === 'Tab') {
			e.preventDefault();
			if (flatList[selectedIndex]) {
				selectBlockType(flatList[selectedIndex].type);
			}
		} else if (e.key === 'Escape') {
			e.preventDefault();
			editor.hideSlashMenu();
		}
	}

	function scrollSelectedIntoView() {
		// Use requestAnimationFrame to wait for DOM update
		requestAnimationFrame(() => {
			const menuEl = document.querySelector('[data-slash-menu]');
			const selectedEl = menuEl?.querySelector('.menu-item.bg-gray-100');
			if (selectedEl) {
				selectedEl.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
			}
		});
	}

	// Icon mapping to SVG paths
	function getIconSvg(iconName: string): string {
		const icons: Record<string, string> = {
			'file-text': 'M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z M14 2v6h6 M16 13H8 M16 17H8 M10 9H8',
			'minus': 'M5 12h14',
			'alert-circle': 'M12 22c5.523 0 10-4.477 10-10S17.523 2 12 2 2 6.477 2 12s4.477 10 10 10z M12 8v4 M12 16h.01',
			'type': 'M4 7V4h16v3 M9 20h6 M12 4v16',
			'heading-1': 'M4 12h8 M4 18V6 M12 18V6 M17 10v8 M17 10l3-2',
			'heading-2': 'M4 12h8 M4 18V6 M12 18V6 M21 18h-4c0-4 4-3 4-6 0-1.5-2-2.5-4-1',
			'heading-3': 'M4 12h8 M4 18V6 M12 18V6 M17.5 10.5c1.7-1 3.5 0 3.5 1.5a2 2 0 0 1-2 2 M17.5 17.5c1.7 1 3.5 0 3.5-1.5a2 2 0 0 0-2-2',
			'list': 'M8 6h13 M8 12h13 M8 18h13 M3 6h.01 M3 12h.01 M3 18h.01',
			'list-ordered': 'M10 6h11 M10 12h11 M10 18h11 M4 6h1v4 M4 10h2 M6 18H4c0-1 2-2 2-3s-1-1.5-2-1',
			'check-square': 'M9 11l3 3L22 4 M21 12v7a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11',
			'chevron-right': 'M9 18l6-6-6-6',
			'quote': 'M3 21c3 0 7-1 7-8V5c0-1.25-.756-2.017-2-2H4c-1.25 0-2 .75-2 1.972V11c0 1.25.75 2 2 2 1 0 1 0 1 1v1c0 1-1 2-2 2s-1 .008-1 1.031V21z M15 21c3 0 7-1 7-8V5c0-1.25-.757-2.017-2-2h-4c-1.25 0-2 .75-2 1.972V11c0 1.25.75 2 2 2h.75c0 2.25.25 4-2.75 4v3z',
			'code': 'M16 18l6-6-6-6 M8 6l-6 6 6 6',
			'columns': 'M9 4H5a1 1 0 0 0-1 1v14a1 1 0 0 0 1 1h4a1 1 0 0 0 1-1V5a1 1 0 0 0-1-1z M19 4h-4a1 1 0 0 0-1 1v14a1 1 0 0 0 1 1h4a1 1 0 0 0 1-1V5a1 1 0 0 0-1-1z',
			'link': 'M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71 M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71'
		};
		return icons[iconName] || icons['type'];
	}

	function getGlobalIndex(section: 'suggested' | 'basic', localIndex: number): number {
		if (section === 'suggested') return localIndex;
		return (sections?.suggested?.length || 0) + localIndex;
	}
</script>

<svelte:window onkeydown={handleKeydown} />

{#if $editor.slashMenuPosition}
	<div
		data-slash-menu
		class="slash-menu fixed z-50 w-80 bg-white dark:bg-[var(--bos-v2-layer-background-overlayPanel)] rounded-xl shadow-2xl border border-gray-200 dark:border-[var(--bos-v2-layer-insideBorder-border)] overflow-hidden"
		style="left: {$editor.slashMenuPosition.x}px; top: {$editor.slashMenuPosition.y}px;"
		transition:fly={{ y: -8, duration: 150 }}
	>
		<!-- Menu Content -->
		<div class="max-h-96 overflow-y-auto">
			{#if !$editor.slashMenuQuery && sections}
				<!-- SUGGESTED SECTION -->
				{#if sections.suggested.length > 0}
					<div class="px-3 pt-3 pb-1.5">
						<p class="text-[11px] font-semibold text-gray-400 dark:text-gray-500 uppercase tracking-wider">Suggested</p>
					</div>
					{#each sections.suggested as blockType, idx}
						{@const globalIdx = getGlobalIndex('suggested', idx)}
						<div
							class="menu-item w-full px-3 py-2 flex items-center gap-3 text-left transition-colors cursor-pointer
								{globalIdx === selectedIndex ? 'bg-gray-100 dark:bg-[var(--bos-v2-layer-background-tertiary)]' : 'hover:bg-gray-50 dark:hover:bg-[var(--bos-v2-layer-background-secondary)]'}"
							onclick={() => selectBlockType(blockType.type)}
							onmouseenter={() => selectedIndex = globalIdx}
							role="button"
							tabindex="0"
						>
							{#if blockType.type === 'page'}
								<div class="relative">
									<button
										onclick={(e) => handleIconClick(e, blockType.type)}
										class="w-10 h-10 rounded-lg bg-blue-50 dark:bg-blue-900/30 border border-blue-200 dark:border-blue-700 flex items-center justify-center text-blue-600 dark:text-blue-400 hover:bg-blue-100 dark:hover:bg-blue-900/50 transition-colors"
										title="Click to change icon"
									>
										<svg class="w-5 h-5" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24">
											<path stroke-linecap="round" stroke-linejoin="round" d={getIconPath(selectedPageIcon)} />
										</svg>
									</button>
									{#if showIconPicker}
										<div class="absolute left-0 top-full mt-1 z-[100] w-48 max-h-48 bg-white dark:bg-[var(--bos-v2-layer-background-overlayPanel)] rounded-lg shadow-xl border border-gray-200 dark:border-[var(--bos-v2-layer-insideBorder-border)] overflow-y-auto p-2 grid grid-cols-5 gap-1">
											{#each iconPresets as preset}
												<button
													onclick={(e) => { e.stopPropagation(); selectIcon(preset.id); }}
													class="w-8 h-8 rounded-md flex items-center justify-center transition-colors
														{selectedPageIcon === preset.id ? 'bg-blue-100 dark:bg-blue-900/50 text-blue-600' : 'hover:bg-gray-100 dark:hover:bg-[var(--dbg2)] text-gray-500'}"
													title={preset.id}
												>
													<svg class="w-4 h-4" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24">
														<path stroke-linecap="round" stroke-linejoin="round" d={preset.path} />
													</svg>
												</button>
											{/each}
										</div>
									{/if}
								</div>
							{:else}
								<div class="w-10 h-10 rounded-lg bg-gray-100 dark:bg-[var(--bos-v2-layer-background-secondary)] border border-gray-200 dark:border-[var(--bos-v2-layer-insideBorder-border)] flex items-center justify-center text-gray-500 dark:text-gray-400">
									<svg class="w-5 h-5" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24">
										<path stroke-linecap="round" stroke-linejoin="round" d={getIconSvg(blockType.icon)} />
									</svg>
								</div>
							{/if}
							<div class="flex-1 min-w-0">
								<div class="text-sm font-medium text-gray-800 dark:text-gray-200">{blockType.label}</div>
								<div class="text-xs text-gray-500">{blockType.description}</div>
							</div>
							{#if blockType.keyboardShortcut}
								<kbd class="px-1.5 py-0.5 text-[10px] font-mono bg-gray-100 dark:bg-[var(--bos-v2-layer-background-tertiary)] border border-gray-200 dark:border-[var(--bos-v2-layer-background-tertiary)] rounded text-gray-500 dark:text-gray-400">
									{blockType.keyboardShortcut}
								</kbd>
							{/if}
						</div>
					{/each}
				{/if}

				<!-- BASIC BLOCKS SECTION -->
				{#if sections.basic.length > 0}
					<div class="px-3 pt-3 pb-1.5 {sections.suggested.length > 0 ? 'border-t border-gray-200 dark:border-[var(--bos-v2-layer-insideBorder-border)] mt-1' : ''}">
						<p class="text-[11px] font-semibold text-gray-400 dark:text-gray-500 uppercase tracking-wider">Basic blocks</p>
					</div>
					{#each sections.basic as blockType, idx}
						{@const globalIdx = getGlobalIndex('basic', idx)}
						<div
							class="menu-item w-full px-3 py-2 flex items-center gap-3 text-left transition-colors cursor-pointer
								{globalIdx === selectedIndex ? 'bg-gray-100 dark:bg-[var(--bos-v2-layer-background-tertiary)]' : 'hover:bg-gray-50 dark:hover:bg-[var(--bos-v2-layer-background-secondary)]'}"
							onclick={() => selectBlockType(blockType.type)}
							onmouseenter={() => selectedIndex = globalIdx}
							role="button"
							tabindex="0"
						>
							{#if blockType.type === 'page'}
								<div class="relative">
									<button
										onclick={(e) => handleIconClick(e, blockType.type)}
										class="w-10 h-10 rounded-lg bg-blue-50 dark:bg-blue-900/30 border border-blue-200 dark:border-blue-700 flex items-center justify-center text-blue-600 dark:text-blue-400 hover:bg-blue-100 dark:hover:bg-blue-900/50 transition-colors"
										title="Click to change icon"
									>
										<svg class="w-5 h-5" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24">
											<path stroke-linecap="round" stroke-linejoin="round" d={getIconPath(selectedPageIcon)} />
										</svg>
									</button>
									{#if showIconPicker}
										<div class="absolute left-0 top-full mt-1 z-[100] w-48 max-h-48 bg-white dark:bg-[var(--bos-v2-layer-background-overlayPanel)] rounded-lg shadow-xl border border-gray-200 dark:border-[var(--bos-v2-layer-insideBorder-border)] overflow-y-auto p-2 grid grid-cols-5 gap-1">
											{#each iconPresets as preset}
												<button
													onclick={(e) => { e.stopPropagation(); selectIcon(preset.id); }}
													class="w-8 h-8 rounded-md flex items-center justify-center transition-colors
														{selectedPageIcon === preset.id ? 'bg-blue-100 dark:bg-blue-900/50 text-blue-600' : 'hover:bg-gray-100 dark:hover:bg-[var(--dbg2)] text-gray-500'}"
													title={preset.id}
												>
													<svg class="w-4 h-4" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24">
														<path stroke-linecap="round" stroke-linejoin="round" d={preset.path} />
													</svg>
												</button>
											{/each}
										</div>
									{/if}
								</div>
							{:else}
								<div class="w-10 h-10 rounded-lg bg-gray-100 dark:bg-[var(--bos-v2-layer-background-secondary)] border border-gray-200 dark:border-[var(--bos-v2-layer-insideBorder-border)] flex items-center justify-center text-gray-500 dark:text-gray-400">
									<svg class="w-5 h-5" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24">
										<path stroke-linecap="round" stroke-linejoin="round" d={getIconSvg(blockType.icon)} />
									</svg>
								</div>
							{/if}
							<div class="flex-1 min-w-0">
								<div class="text-sm font-medium text-gray-800 dark:text-gray-200">{blockType.label}</div>
								<div class="text-xs text-gray-500">{blockType.description}</div>
							</div>
							{#if blockType.keyboardShortcut}
								<kbd class="px-1.5 py-0.5 text-[10px] font-mono bg-gray-100 dark:bg-[var(--bos-v2-layer-background-tertiary)] border border-gray-200 dark:border-[var(--bos-v2-layer-background-tertiary)] rounded text-gray-500 dark:text-gray-400">
									{blockType.keyboardShortcut}
								</kbd>
							{/if}
						</div>
					{/each}
				{/if}
			{:else}
				<!-- FILTERED RESULTS (flat list) -->
				{#if filteredTypes.length > 0}
					{#each filteredTypes as blockType, idx}
						<div
							class="menu-item w-full px-3 py-2 flex items-center gap-3 text-left transition-colors cursor-pointer
								{idx === selectedIndex ? 'bg-gray-100 dark:bg-[var(--bos-v2-layer-background-tertiary)]' : 'hover:bg-gray-50 dark:hover:bg-[var(--bos-v2-layer-background-secondary)]'}"
							onclick={() => selectBlockType(blockType.type)}
							onmouseenter={() => selectedIndex = idx}
							role="button"
							tabindex="0"
						>
							<!-- Clickable icon for page type -->
							{#if blockType.type === 'page'}
								<div class="relative">
									<button
										onclick={(e) => handleIconClick(e, blockType.type)}
										class="w-10 h-10 rounded-lg bg-blue-50 dark:bg-blue-900/30 border border-blue-200 dark:border-blue-700 flex items-center justify-center text-blue-600 dark:text-blue-400 hover:bg-blue-100 dark:hover:bg-blue-900/50 transition-colors"
										title="Click to change icon"
									>
										<svg class="w-5 h-5" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24">
											<path stroke-linecap="round" stroke-linejoin="round" d={getIconPath(selectedPageIcon)} />
										</svg>
									</button>
									<!-- Icon picker dropdown -->
									{#if showIconPicker}
										<div class="absolute left-0 top-full mt-1 z-[100] w-48 max-h-48 bg-white dark:bg-[var(--bos-v2-layer-background-overlayPanel)] rounded-lg shadow-xl border border-gray-200 dark:border-[var(--bos-v2-layer-insideBorder-border)] overflow-y-auto p-2 grid grid-cols-5 gap-1">
											{#each iconPresets as preset}
												<button
													onclick={(e) => { e.stopPropagation(); selectIcon(preset.id); }}
													class="w-8 h-8 rounded-md flex items-center justify-center transition-colors
														{selectedPageIcon === preset.id ? 'bg-blue-100 dark:bg-blue-900/50 text-blue-600' : 'hover:bg-gray-100 dark:hover:bg-[var(--dbg2)] text-gray-500'}"
													title={preset.id}
												>
													<svg class="w-4 h-4" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24">
														<path stroke-linecap="round" stroke-linejoin="round" d={preset.path} />
													</svg>
												</button>
											{/each}
										</div>
									{/if}
								</div>
							{:else}
								<div class="w-10 h-10 rounded-lg bg-gray-100 dark:bg-[var(--bos-v2-layer-background-secondary)] border border-gray-200 dark:border-[var(--bos-v2-layer-insideBorder-border)] flex items-center justify-center text-gray-500 dark:text-gray-400">
									<svg class="w-5 h-5" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24">
										<path stroke-linecap="round" stroke-linejoin="round" d={getIconSvg(blockType.icon)} />
									</svg>
								</div>
							{/if}
							<div class="flex-1 min-w-0">
								<div class="text-sm font-medium text-gray-800 dark:text-gray-200">{blockType.label}</div>
								<div class="text-xs text-gray-500">{blockType.description}</div>
							</div>
							{#if blockType.keyboardShortcut}
								<kbd class="px-1.5 py-0.5 text-[10px] font-mono bg-gray-100 dark:bg-[var(--bos-v2-layer-background-tertiary)] border border-gray-200 dark:border-[var(--bos-v2-layer-background-tertiary)] rounded text-gray-500 dark:text-gray-400">
									{blockType.keyboardShortcut}
								</kbd>
							{/if}
						</div>
					{/each}
				{:else}
					<!-- Empty state -->
					<div class="px-4 py-8 text-center">
						<p class="text-sm text-gray-500">No blocks found</p>
						<p class="text-xs text-gray-400 dark:text-gray-600 mt-1">Try a different search term</p>
					</div>
				{/if}
			{/if}
		</div>

		<!-- Filter Footer -->
		<div class="px-3 py-2 border-t border-gray-200 dark:border-[var(--bos-v2-layer-insideBorder-border)] bg-gray-50 dark:bg-[var(--bos-v2-layer-background-secondary)] flex items-center gap-2">
			<span class="text-gray-400 dark:text-gray-500 text-sm">/</span>
			<span class="flex-1 text-sm {$editor.slashMenuQuery ? 'text-gray-700 dark:text-gray-300' : 'text-gray-400 dark:text-gray-500'}">
				{$editor.slashMenuQuery || 'Filter...'}
			</span>
			<kbd class="px-1.5 py-0.5 text-[10px] font-mono bg-gray-100 dark:bg-[var(--bos-v2-layer-background-tertiary)] rounded text-gray-500 dark:text-gray-400">esc</kbd>
		</div>
	</div>
{/if}

<style>
	.slash-menu {
		/* Subtle shadow for light theme, darker for dark theme */
		box-shadow:
			0 0 0 1px rgba(0, 0, 0, 0.05),
			0 4px 8px rgba(0, 0, 0, 0.08),
			0 16px 24px rgba(0, 0, 0, 0.1),
			0 24px 32px rgba(0, 0, 0, 0.08);
	}

	:global(.dark) .slash-menu {
		box-shadow:
			0 0 0 1px rgba(0, 0, 0, 0.2),
			0 4px 8px rgba(0, 0, 0, 0.15),
			0 16px 24px rgba(0, 0, 0, 0.2),
			0 24px 32px rgba(0, 0, 0, 0.15);
	}

	/* Smooth scrollbar - light theme */
	.slash-menu > div:first-child::-webkit-scrollbar {
		width: 6px;
	}

	.slash-menu > div:first-child::-webkit-scrollbar-track {
		background: transparent;
	}

	.slash-menu > div:first-child::-webkit-scrollbar-thumb {
		background: var(--bos-status-neutral);
		border-radius: 3px;
	}

	.slash-menu > div:first-child::-webkit-scrollbar-thumb:hover {
		background: var(--bos-status-neutral);
		opacity: 0.8;
	}

	/* Dark theme scrollbar */
	:global(.dark) .slash-menu > div:first-child::-webkit-scrollbar-thumb {
		background: var(--dbd);
	}

	:global(.dark) .slash-menu > div:first-child::-webkit-scrollbar-thumb:hover {
		background: var(--dbd2);
	}
</style>
