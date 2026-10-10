<script lang="ts">
	import { slide, fade } from 'svelte/transition';
	import { goto } from '$app/navigation';
	import type { App } from '$lib/types/apps';
	import {
		Search,
		Command,
		Clock,
		Layers,
		CheckSquare,
		Users,
		Kanban,
		BookOpen,
		Calendar,
		BarChart3,
		Wallet,
		ArrowRight,
		Plus
	} from 'lucide-svelte';

	interface Props {
		apps: App[];
		recentApps: App[];
		isOpen: boolean;
		onClose: () => void;
		onSelect: (app: App) => void;
		onCreateApp?: () => void;
	}

	let { apps, recentApps, isOpen, onClose, onSelect, onCreateApp }: Props = $props();

	let query = $state('');
	let selectedIndex = $state(0);
	let inputElement: HTMLInputElement | null = $state(null);

	// Icon mapping
	const iconMap: Record<string, typeof Layers> = {
		task: CheckSquare,
		crm: Users,
		client: Users,
		project: Kanban,
		tracker: Kanban,
		journal: BookOpen,
		calendar: Calendar,
		analytics: BarChart3,
		report: BarChart3,
		dashboard: BarChart3,
		finance: Wallet,
		invoice: Wallet
	};

	function getAppIcon(name: string) {
		const lowerName = name.toLowerCase();
		for (const [keyword, icon] of Object.entries(iconMap)) {
			if (lowerName.includes(keyword)) {
				return icon;
			}
		}
		return Layers;
	}

	// Filter results based on query
	const filteredApps = $derived.by(() => {
		if (!query.trim()) {
			return [];
		}
		const q = query.toLowerCase();
		return apps.filter(
			(app) =>
				app.name.toLowerCase().includes(q) || app.description.toLowerCase().includes(q)
		);
	});

	// Show sections based on query
	const showRecent = $derived(!query.trim() && recentApps.length > 0);
	const showAllApps = $derived(!query.trim());
	const showSearchResults = $derived(query.trim().length > 0);

	// Build flat list for keyboard navigation
	const flatResults = $derived.by(() => {
		const items: { type: 'app' | 'action'; app?: App; action?: string }[] = [];

		if (showSearchResults) {
			filteredApps.forEach((app) => items.push({ type: 'app', app }));
		} else {
			if (showRecent) {
				recentApps.forEach((app) => items.push({ type: 'app', app }));
			}
			if (showAllApps) {
				apps.forEach((app) => items.push({ type: 'app', app }));
			}
		}

		// Always add "Create new app" action at the end
		items.push({ type: 'action', action: 'create' });

		return items;
	});

	// Focus input when opened
	$effect(() => {
		if (isOpen && inputElement) {
			setTimeout(() => inputElement?.focus(), 50);
		}
	});

	// Reset state when opened/closed
	$effect(() => {
		if (isOpen) {
			query = '';
			selectedIndex = 0;
		}
	});

	// Keep selectedIndex in bounds (handles empty arrays safely)
	$effect(() => {
		const maxIndex = flatResults.length - 1;
		if (flatResults.length === 0) {
			selectedIndex = 0;
		} else if (selectedIndex > maxIndex) {
			selectedIndex = maxIndex;
		} else if (selectedIndex < 0) {
			selectedIndex = 0;
		}
	});

	function handleKeydown(e: KeyboardEvent) {
		const items = flatResults;
		const itemCount = items.length;

		// Early return if no items (except for Escape which should still work)
		if (itemCount === 0 && e.key !== 'Escape') {
			return;
		}

		switch (e.key) {
			case 'Escape':
				e.preventDefault();
				onClose();
				break;
			case 'ArrowDown':
				e.preventDefault();
				if (itemCount > 0) {
					selectedIndex = (selectedIndex + 1) % itemCount;
				}
				break;
			case 'ArrowUp':
				e.preventDefault();
				if (itemCount > 0) {
					selectedIndex = (selectedIndex - 1 + itemCount) % itemCount;
				}
				break;
			case 'Enter':
				e.preventDefault();
				// Bounds check before accessing array
				if (selectedIndex >= 0 && selectedIndex < itemCount) {
					const item = items[selectedIndex];
					if (item) {
						if (item.type === 'app' && item.app) {
							onSelect(item.app);
							onClose();
						} else if (item.type === 'action' && item.action === 'create') {
							onCreateApp?.();
							onClose();
						}
					}
				}
				break;
		}
	}

	function handleBackdropClick(e: MouseEvent) {
		if (e.target === e.currentTarget) {
			onClose();
		}
	}

	function handleAppClick(app: App) {
		onSelect(app);
		onClose();
	}

	function handleCreateClick() {
		onCreateApp?.();
		onClose();
	}

	// Helper to check if item is selected
	function isSelected(index: number): boolean {
		return index === selectedIndex;
	}

	// Get running index for keyboard navigation
	let runningIndex = 0;
	function getAndIncrementIndex(): number {
		return runningIndex++;
	}
	function resetIndex() {
		runningIndex = 0;
	}
</script>

<svelte:window
	onkeydown={(e) => {
		// Global Cmd+K handler
		if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
			e.preventDefault();
			// Parent handles toggling
		}
	}}
/>

{#if isOpen}
	<!-- Backdrop -->
	<div
		class="fixed inset-0 z-50 bg-black/50 backdrop-blur-sm"
		transition:fade={{ duration: 150 }}
		onclick={handleBackdropClick}
		onkeydown={handleKeydown}
		role="dialog"
		aria-modal="true"
		aria-label="App launcher"
		tabindex="-1"
	>
		<!-- Modal -->
		<div
			class="fixed top-[15%] left-1/2 -translate-x-1/2 w-full max-w-xl bg-white dark:bg-gray-800 rounded-2xl shadow-2xl overflow-hidden border border-gray-200 dark:border-gray-700"
			transition:slide={{ duration: 200 }}
		>
			<!-- Search Input -->
			<div class="flex items-center gap-3 px-4 py-3 border-b border-gray-200 dark:border-gray-700">
				<Search size={20} class="text-gray-400 flex-shrink-0" strokeWidth={2} aria-hidden="true" />
				<input
					bind:this={inputElement}
					type="text"
					bind:value={query}
					placeholder="Search apps or type to filter..."
					aria-label="Search apps"
					aria-autocomplete="list"
					aria-controls="app-search-results"
					class="flex-1 bg-transparent border-0 outline-none text-gray-900 dark:text-gray-100 placeholder:text-gray-400 dark:placeholder:text-gray-500 text-base"
					onkeydown={handleKeydown}
				/>
				<div class="flex items-center gap-1">
					<kbd class="px-1.5 py-0.5 text-xs text-gray-400 bg-gray-100 dark:bg-gray-700 rounded font-mono">
						<Command class="w-3 h-3 inline" />K
					</kbd>
				</div>
			</div>

			<!-- Results -->
			<div class="max-h-[400px] overflow-y-auto" id="app-search-results" aria-live="polite" aria-atomic="false">
				{#if showSearchResults}
					<!-- Search Results -->
					{@const _ = resetIndex()}
					{#if filteredApps.length === 0}
						<div class="px-4 py-8 text-center text-gray-500 dark:text-gray-400">
							<p>No apps found for "{query}"</p>
						</div>
					{:else}
						<div class="py-2">
							<div class="px-3 py-1.5 text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
								Results
							</div>
							{#each filteredApps as app}
								{@const idx = getAndIncrementIndex()}
								<button
									onclick={() => handleAppClick(app)}
									class="w-full flex items-center gap-3 px-3 py-2.5 text-left transition-colors
										{isSelected(idx) ? 'bg-blue-50 dark:bg-blue-900/30' : 'hover:bg-gray-50 dark:hover:bg-gray-700/50'}"
								>
									<div class="w-9 h-9 rounded-lg bg-gradient-to-br from-gray-100 to-gray-200 dark:from-gray-700 dark:to-gray-600 flex items-center justify-center text-gray-600 dark:text-gray-300 flex-shrink-0">
										<svelte:component this={getAppIcon(app.name)} size={20} strokeWidth={2} />
									</div>
									<div class="flex-1 min-w-0">
										<div class="text-sm font-medium text-gray-900 dark:text-gray-100 truncate">
											{app.name}
										</div>
										<div class="text-xs text-gray-500 dark:text-gray-400 truncate">
											{app.description}
										</div>
									</div>
									{#if isSelected(idx)}
										<ArrowRight size={20} class="text-blue-500 flex-shrink-0" />
									{/if}
								</button>
							{/each}
						</div>
					{/if}
				{:else}
					<!-- Default View: Recent + All Apps -->
					{@const _ = resetIndex()}

					<!-- Recently Opened -->
					{#if showRecent}
						<div class="py-2">
							<div class="px-3 py-1.5 text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider flex items-center gap-2">
								<Clock class="w-3 h-3" />
								Recently Opened
							</div>
							{#each recentApps as app}
								{@const idx = getAndIncrementIndex()}
								<button
									onclick={() => handleAppClick(app)}
									class="w-full flex items-center gap-3 px-3 py-2.5 text-left transition-colors
										{isSelected(idx) ? 'bg-blue-50 dark:bg-blue-900/30' : 'hover:bg-gray-50 dark:hover:bg-gray-700/50'}"
								>
									<div class="w-9 h-9 rounded-lg bg-gradient-to-br from-gray-100 to-gray-200 dark:from-gray-700 dark:to-gray-600 flex items-center justify-center text-gray-600 dark:text-gray-300 flex-shrink-0">
										<svelte:component this={getAppIcon(app.name)} size={20} strokeWidth={2} />
									</div>
									<div class="flex-1 min-w-0">
										<div class="text-sm font-medium text-gray-900 dark:text-gray-100 truncate">
											{app.name}
										</div>
										<div class="text-xs text-gray-500 dark:text-gray-400 truncate">
											{app.description}
										</div>
									</div>
									{#if isSelected(idx)}
										<ArrowRight size={20} class="text-blue-500 flex-shrink-0" />
									{/if}
								</button>
							{/each}
						</div>
					{/if}

					<!-- All Apps -->
					{#if showAllApps && apps.length > 0}
						<div class="py-2 {showRecent ? 'border-t border-gray-100 dark:border-gray-700' : ''}">
							<div class="px-3 py-1.5 text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider flex items-center gap-2">
								<Layers class="w-3 h-3" />
								All Apps
							</div>
							{#each apps as app}
								{@const idx = getAndIncrementIndex()}
								<button
									onclick={() => handleAppClick(app)}
									class="w-full flex items-center gap-3 px-3 py-2.5 text-left transition-colors
										{isSelected(idx) ? 'bg-blue-50 dark:bg-blue-900/30' : 'hover:bg-gray-50 dark:hover:bg-gray-700/50'}"
								>
									<div class="w-9 h-9 rounded-lg bg-gradient-to-br from-gray-100 to-gray-200 dark:from-gray-700 dark:to-gray-600 flex items-center justify-center text-gray-600 dark:text-gray-300 flex-shrink-0">
										<svelte:component this={getAppIcon(app.name)} size={20} strokeWidth={2} />
									</div>
									<div class="flex-1 min-w-0">
										<div class="text-sm font-medium text-gray-900 dark:text-gray-100 truncate">
											{app.name}
										</div>
										<div class="text-xs text-gray-500 dark:text-gray-400 truncate">
											{app.description}
										</div>
									</div>
									{#if isSelected(idx)}
										<ArrowRight size={20} class="text-blue-500 flex-shrink-0" />
									{/if}
								</button>
							{/each}
						</div>
					{/if}
				{/if}

				<!-- Create New App Action -->
				<div class="py-2 border-t border-gray-100 dark:border-gray-700">
					<button
						onclick={handleCreateClick}
						class="w-full flex items-center gap-3 px-3 py-2.5 text-left transition-colors
							{flatResults.length > 0 && isSelected(flatResults.length - 1) ? 'bg-blue-50 dark:bg-blue-900/30' : 'hover:bg-gray-50 dark:hover:bg-gray-700/50'}"
					>
						<div class="w-9 h-9 rounded-lg border-2 border-dashed border-gray-300 dark:border-gray-600 flex items-center justify-center text-gray-400 dark:text-gray-500 flex-shrink-0">
							<Plus size={20} strokeWidth={2} />
						</div>
						<div class="flex-1 min-w-0">
							<div class="text-sm font-medium text-gray-900 dark:text-gray-100">
								Create new app
							</div>
							<div class="text-xs text-gray-500 dark:text-gray-400">
								Start from scratch or use a template
							</div>
						</div>
						{#if flatResults.length > 0 && isSelected(flatResults.length - 1)}
							<ArrowRight size={20} class="text-blue-500 flex-shrink-0" />
						{/if}
					</button>
				</div>
			</div>

			<!-- Footer with keyboard shortcuts -->
			<div class="px-4 py-2 border-t border-gray-200 dark:border-gray-700 flex items-center gap-4 text-xs text-gray-500 dark:text-gray-400 bg-gray-50 dark:bg-gray-800/50">
				<span class="flex items-center gap-1">
					<kbd class="px-1.5 py-0.5 bg-white dark:bg-gray-700 rounded border border-gray-200 dark:border-gray-600">↑</kbd>
					<kbd class="px-1.5 py-0.5 bg-white dark:bg-gray-700 rounded border border-gray-200 dark:border-gray-600">↓</kbd>
					<span>navigate</span>
				</span>
				<span class="flex items-center gap-1">
					<kbd class="px-1.5 py-0.5 bg-white dark:bg-gray-700 rounded border border-gray-200 dark:border-gray-600">↵</kbd>
					<span>open</span>
				</span>
				<span class="flex items-center gap-1">
					<kbd class="px-1.5 py-0.5 bg-white dark:bg-gray-700 rounded border border-gray-200 dark:border-gray-600">esc</kbd>
					<span>close</span>
				</span>
			</div>
		</div>
	</div>
{/if}
