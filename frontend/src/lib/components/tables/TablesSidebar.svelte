<script lang="ts">
	/**
	 * TablesSidebar - NocoDB-style sidebar with bases and tables tree
	 * Features: Bases/folders, table tree, create new, favorites
	 */
	import {
		Plus,
		ChevronRight,
		ChevronDown,
		Folder,
		FolderOpen,
		Table2,
		Star,
		Upload,
		Database,
		Link,
		MoreHorizontal,
		Trash2,
		Edit3,
		Copy,
		Settings
	} from 'lucide-svelte';
	import type { TableListItem } from '$lib/api/tables/types';

	interface Base {
		id: string;
		name: string;
		icon: string;
		color: string;
		tables: TableListItem[];
		isExpanded: boolean;
	}

	interface Props {
		tables: TableListItem[];
		favorites: TableListItem[];
		selectedTableId?: string;
		onTableClick: (id: string) => void;
		onCreateTable: () => void;
		onCreateBase: () => void;
		onImport: () => void;
	}

	let {
		tables,
		favorites,
		selectedTableId,
		onTableClick,
		onCreateTable,
		onCreateBase,
		onImport
	}: Props = $props();

	// Group tables into bases (for now, we'll create virtual groups)
	let bases = $state<Base[]>([]);
	let expandedSections = $state({
		favorites: true,
		myData: true,
		shared: false,
		imports: true
	});

	// Context menu state
	let contextMenu = $state<{ x: number; y: number; tableId: string } | null>(null);

	// Organize tables by source
	const customTables = $derived(tables.filter((t) => t.source === 'custom'));
	const importedTables = $derived(tables.filter((t) => t.source === 'import'));
	const integrationTables = $derived(tables.filter((t) => t.source === 'integration'));

	function toggleSection(section: keyof typeof expandedSections) {
		expandedSections[section] = !expandedSections[section];
	}

	function handleContextMenu(e: MouseEvent, tableId: string) {
		e.preventDefault();
		contextMenu = { x: e.clientX, y: e.clientY, tableId };
	}

	function closeContextMenu() {
		contextMenu = null;
	}

	// Close context menu on click outside
	function handleWindowClick() {
		if (contextMenu) closeContextMenu();
	}
</script>

<svelte:window on:click={handleWindowClick} />

<div class="dt2-sidebar flex h-full w-64 flex-col">
	<!-- Header -->
	<div class="dt2-sidebar__header flex items-center justify-between px-4 py-3">
		<div class="flex items-center gap-2">
			<Database class="h-5 w-5 text-blue-600" />
			<span class="font-semibold" style="color: var(--dt);">Tables</span>
		</div>
		<div class="flex items-center gap-1">
			<button
				type="button"
				class="btn-pill btn-pill-ghost btn-pill-icon"
				onclick={onImport}
				title="Import data"
			>
				<Upload class="h-4 w-4" />
			</button>
			<button
				type="button"
				class="btn-pill btn-pill-ghost btn-pill-icon"
				onclick={onCreateTable}
				title="Create table"
			>
				<Plus class="h-4 w-4" />
			</button>
		</div>
	</div>

	<!-- Scrollable Content -->
	<div class="flex-1 overflow-y-auto">
		<!-- Quick Actions -->
		<div class="border-b border-gray-200 p-3">
			<button
				type="button"
				class="flex w-full items-center gap-2 px-3 py-2 rounded-lg text-sm hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors border border-dashed border-gray-300"
				onclick={onCreateBase}
			>
				<Plus class="h-4 w-4" />
				<span>New Base</span>
			</button>
		</div>

		<!-- Favorites Section -->
		{#if favorites.length > 0}
			<div class="border-b border-gray-200">
				<button
					type="button"
					class="flex w-full items-center gap-2 text-left uppercase tracking-wider px-3 py-1.5 text-xs text-gray-500 hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors"
					onclick={() => toggleSection('favorites')}
				>
					{#if expandedSections.favorites}
						<ChevronDown class="h-3 w-3" />
					{:else}
						<ChevronRight class="h-3 w-3" />
					{/if}
					<Star class="h-3 w-3" />
					Favorites
					<span class="ml-auto rounded-full bg-gray-200 px-1.5 py-0.5 text-xs text-gray-600">
						{favorites.length}
					</span>
				</button>
				{#if expandedSections.favorites}
					<div class="pb-2">
						{#each favorites as table}
							<button
								type="button"
								class="w-full flex items-center gap-2 text-left px-3 py-2 rounded-lg text-sm transition-colors {selectedTableId === table.id ? 'bg-gray-100 dark:bg-gray-800 font-medium' : 'hover:bg-gray-50 dark:hover:bg-gray-900'}"
								onclick={() => onTableClick(table.id)}
								oncontextmenu={(e) => handleContextMenu(e, table.id)}
							>
								<Table2 class="h-4 w-4 shrink-0 text-gray-400" />
								<span class="truncate">{table.name}</span>
							</button>
						{/each}
					</div>
				{/if}
			</div>
		{/if}

		<!-- My Data Section (Custom Tables) -->
		<div class="border-b border-gray-200">
			<button
				type="button"
				class="flex w-full items-center gap-2 text-left uppercase tracking-wider px-3 py-1.5 text-xs text-gray-500 hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors"
				onclick={() => toggleSection('myData')}
			>
				{#if expandedSections.myData}
					<ChevronDown class="h-3 w-3" />
				{:else}
					<ChevronRight class="h-3 w-3" />
				{/if}
				<FolderOpen class="h-3 w-3" />
				My Tables
				<span class="ml-auto rounded-full bg-gray-200 px-1.5 py-0.5 text-xs text-gray-600">
					{customTables.length}
				</span>
			</button>
			{#if expandedSections.myData}
				<div class="pb-2">
					{#if customTables.length === 0}
						<div class="px-4 py-2 text-xs text-gray-400">No custom tables yet</div>
					{:else}
						{#each customTables as table}
							<button
								type="button"
								class="group w-full flex items-center gap-2 text-left px-3 py-2 rounded-lg text-sm transition-colors {selectedTableId === table.id ? 'bg-gray-100 dark:bg-gray-800 font-medium' : 'hover:bg-gray-50 dark:hover:bg-gray-900'}"
								onclick={() => onTableClick(table.id)}
								oncontextmenu={(e) => handleContextMenu(e, table.id)}
							>
								<Table2 class="h-4 w-4 shrink-0 text-gray-400" />
								<span class="truncate flex-1">{table.name}</span>
								<span class="text-xs text-gray-400">{table.row_count}</span>
							</button>
						{/each}
					{/if}
				</div>
			{/if}
		</div>

		<!-- Imports Section -->
		<div class="border-b border-gray-200">
			<button
				type="button"
				class="flex w-full items-center gap-2 text-left uppercase tracking-wider px-3 py-1.5 text-xs text-gray-500 hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors"
				onclick={() => toggleSection('imports')}
			>
				{#if expandedSections.imports}
					<ChevronDown class="h-3 w-3" />
				{:else}
					<ChevronRight class="h-3 w-3" />
				{/if}
				<Upload class="h-3 w-3" />
				Imported
				<span class="ml-auto rounded-full bg-gray-200 px-1.5 py-0.5 text-xs text-gray-600">
					{importedTables.length}
				</span>
			</button>
			{#if expandedSections.imports}
				<div class="pb-2">
					{#if importedTables.length === 0}
						<button
							type="button"
							class="flex w-full items-center gap-2 text-left px-3 py-1.5 rounded-lg text-xs hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
							onclick={onImport}
						>
							<Plus class="h-3 w-3" />
							Import CSV or Excel
						</button>
					{:else}
						{#each importedTables as table}
							<button
								type="button"
								class="group w-full flex items-center gap-2 text-left px-3 py-2 rounded-lg text-sm transition-colors {selectedTableId === table.id ? 'bg-gray-100 dark:bg-gray-800 font-medium' : 'hover:bg-gray-50 dark:hover:bg-gray-900'}"
								onclick={() => onTableClick(table.id)}
							>
								<Upload class="h-4 w-4 shrink-0 text-orange-400" />
								<span class="truncate flex-1">{table.name}</span>
								<span class="text-xs text-gray-400">{table.row_count}</span>
							</button>
						{/each}
					{/if}
				</div>
			{/if}
		</div>

		<!-- Integrations Section -->
		<div>
			<button
				type="button"
				class="flex w-full items-center gap-2 text-left uppercase tracking-wider px-3 py-1.5 text-xs text-gray-500 hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors"
				onclick={() => toggleSection('shared')}
			>
				{#if expandedSections.shared}
					<ChevronDown class="h-3 w-3" />
				{:else}
					<ChevronRight class="h-3 w-3" />
				{/if}
				<Link class="h-3 w-3" />
				Connected
				<span class="ml-auto rounded-full bg-gray-200 px-1.5 py-0.5 text-xs text-gray-600">
					{integrationTables.length}
				</span>
			</button>
			{#if expandedSections.shared}
				<div class="pb-2">
					{#if integrationTables.length === 0}
						<div class="px-4 py-2 text-xs text-gray-400">No connected sources</div>
					{:else}
						{#each integrationTables as table}
							<button
								type="button"
								class="group w-full flex items-center gap-2 text-left px-3 py-2 rounded-lg text-sm transition-colors {selectedTableId === table.id ? 'bg-gray-100 dark:bg-gray-800 font-medium' : 'hover:bg-gray-50 dark:hover:bg-gray-900'}"
								onclick={() => onTableClick(table.id)}
							>
								<Database class="h-4 w-4 shrink-0 text-green-500" />
								<span class="truncate flex-1">{table.name}</span>
								<span class="text-xs text-gray-400">{table.row_count}</span>
							</button>
						{/each}
					{/if}
				</div>
			{/if}
		</div>
	</div>

	<!-- Footer -->
	<div class="border-t border-gray-200 p-3">
		<button
			type="button"
			class="flex w-full items-center gap-2 px-3 py-2 rounded-lg text-sm hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
		>
			<Settings class="h-4 w-4" />
			<span>Settings</span>
		</button>
	</div>
</div>

<!-- Context Menu -->
{#if contextMenu}
	<div
		class="dt2-context-menu fixed w-48 rounded-lg py-1"
		style="left: {contextMenu.x}px; top: {contextMenu.y}px"
	>
		<button
			type="button"
			class="dt2-context-menu__item flex w-full items-center gap-2 px-3 py-2 text-sm transition-colors"
		>
			<Edit3 class="h-4 w-4" />
			Rename
		</button>
		<button
			type="button"
			class="dt2-context-menu__item flex w-full items-center gap-2 px-3 py-2 text-sm transition-colors"
		>
			<Copy class="h-4 w-4" />
			Duplicate
		</button>
		<button
			type="button"
			class="dt2-context-menu__item flex w-full items-center gap-2 px-3 py-2 text-sm transition-colors"
		>
			<Star class="h-4 w-4" />
			Add to favorites
		</button>
		<div class="my-1" style="border-top: 1px solid var(--dbd);"></div>
		<button
			type="button"
			class="dt2-context-menu__item flex w-full items-center gap-2 px-3 py-2 text-sm transition-colors"
		>
			<Trash2 class="h-4 w-4" />
			Delete
		</button>
	</div>
{/if}

<style>
	.dt2-sidebar {
		background: var(--dbg2);
		border-right: 1px solid var(--dbd);
		color: var(--dt);
	}

	.dt2-sidebar__header {
		border-bottom: 1px solid var(--dbd);
	}

	.dt2-context-menu {
		z-index: var(--bos-z-index-popover, 1001);
		background: var(--dbg);
		border: 1px solid var(--dbd);
		box-shadow: var(--shadow-lg);
	}

	.dt2-context-menu__item {
		color: var(--dt);
		border-radius: 6px;
	}

	.dt2-context-menu__item:hover {
		background: var(--dbg3);
	}
</style>
