<script lang="ts">
	import { onMount } from 'svelte';
	import { browser } from '$app/environment';
	import { filesystemService, type FileItem, type QuickAccessPath } from '$lib/services/filesystem.service';

	// Check if running in Electron
	const isElectron = browser && typeof window !== 'undefined' && !!(window as any).electron;

	// State
	let isLoading = $state(true);
	let error = $state<string | null>(null);
	let currentPath = $state('~');
	let parentDir = $state<string | null>(null);
	let items = $state<FileItem[]>([]);
	let quickAccessPaths = $state<QuickAccessPath[]>([]);
	let selectedItems = $state<Set<string>>(new Set());
	let viewMode = $state<'grid' | 'list'>('list');
	let searchQuery = $state('');
	let showHidden = $state(false);

	// Filtered items based on search
	const filteredItems = $derived(() => {
		if (!searchQuery.trim()) return items;
		const query = searchQuery.toLowerCase();
		return items.filter(item => item.name.toLowerCase().includes(query));
	});

	// Load directory contents
	async function loadDirectory(path: string = '~') {
		isLoading = true;
		error = null;
		try {
			const response = await filesystemService.listDirectory(path, showHidden);
			items = response.items;
			currentPath = response.path;
			parentDir = response.parentDir || null;
			selectedItems = new Set();
		} catch (err: any) {
			console.error('Failed to load directory:', err);
			error = err?.response?.data?.error || err?.message || 'Failed to load directory';
		} finally {
			isLoading = false;
		}
	}

	// Load quick access paths
	async function loadQuickAccessPaths() {
		try {
			const response = await filesystemService.getQuickAccessPaths();
			quickAccessPaths = response.paths;
		} catch (err) {
			console.error('Failed to load quick access paths:', err);
		}
	}

	// Navigation functions
	function navigateTo(path: string) {
		loadDirectory(path);
	}

	function openItem(item: FileItem) {
		if (item.type === 'folder') {
			loadDirectory(item.path);
		} else {
			// For files, could open preview or download
			if (import.meta.env.DEV) console.log('Opening file:', item.name);
			// Could implement file preview modal here
		}
	}

	function goBack() {
		if (parentDir) {
			loadDirectory(parentDir);
		}
	}

	function goHome() {
		loadDirectory('~');
	}

	function toggleSelect(itemId: string, event: MouseEvent) {
		if (event.metaKey || event.ctrlKey) {
			const newSelected = new Set(selectedItems);
			if (newSelected.has(itemId)) {
				newSelected.delete(itemId);
			} else {
				newSelected.add(itemId);
			}
			selectedItems = newSelected;
		} else {
			selectedItems = new Set([itemId]);
		}
	}

	function toggleHidden() {
		showHidden = !showHidden;
		loadDirectory(currentPath);
	}

	// Create new folder
	async function createNewFolder() {
		const name = prompt('Enter folder name:');
		if (!name) return;

		try {
			await filesystemService.createDirectory(currentPath, name);
			loadDirectory(currentPath); // Refresh
		} catch (err: any) {
			alert(err?.response?.data?.error || 'Failed to create folder');
		}
	}

	// Delete selected items
	async function deleteSelected() {
		if (selectedItems.size === 0) return;
		if (!confirm(`Delete ${selectedItems.size} item(s)?`)) return;

		for (const id of selectedItems) {
			const item = items.find(i => i.id === id);
			if (item) {
				try {
					await filesystemService.delete(item.path);
				} catch (err: any) {
					alert(`Failed to delete ${item.name}: ${err?.response?.data?.error || 'Unknown error'}`);
				}
			}
		}
		loadDirectory(currentPath); // Refresh
	}

	// Get breadcrumb segments from path
	function getBreadcrumbs(path: string): { name: string; path: string }[] {
		const parts = path.split('/').filter(Boolean);
		const crumbs: { name: string; path: string }[] = [];
		let accumulated = '';

		for (const part of parts) {
			accumulated += '/' + part;
			crumbs.push({ name: part, path: accumulated });
		}

		return crumbs;
	}

	const breadcrumbs = $derived(getBreadcrumbs(currentPath));

	// Quick access icon mapping
	function getQuickAccessIcon(iconName: string): string {
		const icons: Record<string, string> = {
			home: 'M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6',
			desktop: 'M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z',
			document: 'M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z',
			download: 'M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4',
			image: 'M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z',
			music: 'M9 19V6l12-3v13M9 19c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zm12-3c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zM9 10l12-3',
			video: 'M15 10l4.553-2.276A1 1 0 0121 8.618v6.764a1 1 0 01-1.447.894L15 14M5 18h8a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z',
		};
		return icons[iconName] || icons.document;
	}

	onMount(() => {
		loadQuickAccessPaths();
		loadDirectory('~');
	});
</script>

<div class="file-browser">
	<!-- Sidebar -->
	<aside class="file-sidebar">
		<div class="sidebar-header">
			<h3>Locations</h3>
		</div>

		<div class="sidebar-section">
			{#each quickAccessPaths as loc}
				<button
					class="sidebar-item"
					class:active={currentPath === loc.path}
					onclick={() => navigateTo(loc.path)}
				>
					<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
						<path d={getQuickAccessIcon(loc.icon)} />
					</svg>
					{loc.name}
				</button>
			{/each}
		</div>

		<div class="sidebar-divider"></div>

		<div class="sidebar-section">
			<label class="sidebar-checkbox">
				<input type="checkbox" checked={showHidden} onchange={toggleHidden} />
				<span>Show Hidden Files</span>
			</label>
		</div>
	</aside>

	<!-- Main content -->
	<main class="file-main">
		<!-- Toolbar -->
		<div class="file-toolbar">
			<div class="toolbar-nav">
				<button class="toolbar-btn" onclick={goBack} disabled={!parentDir} title="Go Back">
					<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<path d="M15 19l-7-7 7-7" />
					</svg>
				</button>
				<button class="toolbar-btn" onclick={goHome} title="Go Home">
					<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<path d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6" />
					</svg>
				</button>
				<button class="toolbar-btn" onclick={() => loadDirectory(currentPath)} title="Refresh">
					<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<path d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
					</svg>
				</button>
			</div>

			<!-- Breadcrumb -->
			<div class="file-breadcrumb">
				<button onclick={goHome}>~</button>
				{#each breadcrumbs as crumb, i}
					<span class="breadcrumb-sep">/</span>
					<button onclick={() => navigateTo(crumb.path)}>{crumb.name}</button>
				{/each}
			</div>

			<!-- Search & View toggle -->
			<div class="toolbar-right">
				<div class="search-box">
					<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<circle cx="11" cy="11" r="8" />
						<path d="m21 21-4.35-4.35" />
					</svg>
					<input type="text" placeholder="Search..." bind:value={searchQuery} />
				</div>

				<button class="toolbar-btn" onclick={createNewFolder} title="New Folder">
					<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<path d="M9 13h6m-3-3v6m5 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
					</svg>
				</button>

				{#if selectedItems.size > 0}
					<button class="toolbar-btn danger" onclick={deleteSelected} title="Delete Selected">
						<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<path d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
						</svg>
					</button>
				{/if}

				<div class="view-toggle">
					<button class:active={viewMode === 'list'} onclick={() => viewMode = 'list'} title="List View">
						<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<path d="M4 6h16M4 12h16M4 18h16" />
						</svg>
					</button>
					<button class:active={viewMode === 'grid'} onclick={() => viewMode = 'grid'} title="Grid View">
						<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<rect x="3" y="3" width="7" height="7" />
							<rect x="14" y="3" width="7" height="7" />
							<rect x="3" y="14" width="7" height="7" />
							<rect x="14" y="14" width="7" height="7" />
						</svg>
					</button>
				</div>
			</div>
		</div>

		<!-- Loading State -->
		{#if isLoading}
			<div class="loading-state">
				<div class="spinner"></div>
				<p>Loading...</p>
			</div>
		{:else if error}
			<div class="error-state">
				<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
					<path d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
				</svg>
				<p>{error}</p>
				<button onclick={() => loadDirectory(currentPath)}>Retry</button>
			</div>
		{:else if viewMode === 'list'}
			<!-- List View -->
			<div class="file-list">
				<div class="file-list-header">
					<span class="col-name">Name</span>
					<span class="col-date">Date Modified</span>
					<span class="col-size">Size</span>
				</div>
				{#each filteredItems() as item (item.id)}
					{@const fileIcon = item.type === 'file' ? filesystemService.getFileIcon(item.extension) : null}
					<button
						class="file-list-item"
						class:selected={selectedItems.has(item.id)}
						class:hidden-file={item.isHidden}
						onclick={(e) => toggleSelect(item.id, e)}
						ondblclick={() => openItem(item)}
					>
						<span class="col-name">
							{#if item.type === 'folder'}
								<svg class="item-icon folder" viewBox="0 0 24 24" fill="#3B82F6" stroke="none">
									<path d="M3 7V17C3 18.1046 3.89543 19 5 19H19C20.1046 19 21 18.1046 21 17V9C21 7.89543 20.1046 7 19 7H12L10 5H5C3.89543 5 3 5.89543 3 7Z"/>
								</svg>
							{:else}
								<svg class="item-icon" viewBox="0 0 24 24" fill="none" stroke={fileIcon?.color} stroke-width="1.5">
									<path d={fileIcon?.icon} />
								</svg>
							{/if}
							{item.name}
						</span>
						<span class="col-date">{filesystemService.formatDate(item.modified)}</span>
						<span class="col-size">{item.type === 'folder' ? '-' : filesystemService.formatSize(item.size)}</span>
					</button>
				{/each}

				{#if filteredItems().length === 0}
					<div class="empty-folder">
						<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
							<path d="M3 7V17C3 18.1046 3.89543 19 5 19H19C20.1046 19 21 18.1046 21 17V9C21 7.89543 20.1046 7 19 7H12L10 5H5C3.89543 5 3 5.89543 3 7Z"/>
						</svg>
						<p>{searchQuery ? 'No matching files' : 'This folder is empty'}</p>
					</div>
				{/if}
			</div>
		{:else}
			<!-- Grid View -->
			<div class="file-grid">
				{#each filteredItems() as item (item.id)}
					{@const fileIcon = item.type === 'file' ? filesystemService.getFileIcon(item.extension) : null}
					<button
						class="file-grid-item"
						class:selected={selectedItems.has(item.id)}
						class:hidden-file={item.isHidden}
						onclick={(e) => toggleSelect(item.id, e)}
						ondblclick={() => openItem(item)}
					>
						{#if item.type === 'folder'}
							<svg class="grid-icon folder" viewBox="0 0 24 24" fill="#3B82F6" stroke="none">
								<path d="M3 7V17C3 18.1046 3.89543 19 5 19H19C20.1046 19 21 18.1046 21 17V9C21 7.89543 20.1046 7 19 7H12L10 5H5C3.89543 5 3 5.89543 3 7Z"/>
							</svg>
						{:else}
							<svg class="grid-icon" viewBox="0 0 24 24" fill="none" stroke={fileIcon?.color} stroke-width="1.5">
								<path d={fileIcon?.icon} />
							</svg>
						{/if}
						<span class="grid-name">{item.name}</span>
					</button>
				{/each}
			</div>
		{/if}

		<!-- Status Bar -->
		<div class="file-status">
			<span>{currentPath}</span>
			<span>{filteredItems().length} items</span>
			{#if selectedItems.size > 0}
				<span>{selectedItems.size} selected</span>
			{/if}
		</div>
	</main>
</div>

<style>
	/* Light mode (default) */
	.file-browser {
		display: flex;
		height: 100%;
		background: var(--dbg);
		font-size: 13px;
		color: var(--dt);
	}

	/* Sidebar */
	.file-sidebar {
		width: 200px;
		background: var(--dbg2);
		border-right: 1px solid var(--dbd);
		padding: 8px 0;
		flex-shrink: 0;
		display: flex;
		flex-direction: column;
	}

	.sidebar-header h3 {
		font-size: 11px;
		font-weight: 600;
		color: var(--dt2);
		text-transform: uppercase;
		padding: 8px 16px;
		margin: 0;
	}

	.sidebar-section {
		padding: 0;
	}

	.sidebar-item {
		display: flex;
		align-items: center;
		gap: 8px;
		width: 100%;
		padding: 6px 16px;
		border: none;
		background: none;
		cursor: pointer;
		color: var(--dt);
		text-align: left;
		font-size: 13px;
	}

	.sidebar-item:hover {
		background: rgba(0, 0, 0, 0.05);
	}

	.sidebar-item.active {
		background: rgba(59, 130, 246, 0.1);
		color: #2563eb;
	}

	.sidebar-item svg {
		width: 16px;
		height: 16px;
		flex-shrink: 0;
	}

	.sidebar-divider {
		height: 1px;
		background: var(--dbd);
		margin: 8px 16px;
	}

	.sidebar-checkbox {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 6px 16px;
		font-size: 12px;
		color: var(--dt2);
		cursor: pointer;
	}

	.sidebar-checkbox input {
		margin: 0;
		accent-color: #3B82F6;
	}

	/* Main */
	.file-main {
		flex: 1;
		display: flex;
		flex-direction: column;
		min-width: 0;
		background: var(--dbg);
	}

	/* Toolbar */
	.file-toolbar {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 8px 12px;
		border-bottom: 1px solid var(--dbd);
		background: var(--dbg2);
	}

	.toolbar-nav {
		display: flex;
		gap: 4px;
	}

	.toolbar-btn {
		width: 28px;
		height: 28px;
		border: none;
		background: none;
		border-radius: 6px;
		cursor: pointer;
		display: flex;
		align-items: center;
		justify-content: center;
		color: var(--dt);
	}

	.toolbar-btn:hover:not(:disabled) {
		background: rgba(0, 0, 0, 0.05);
	}

	.toolbar-btn:disabled {
		opacity: 0.3;
		cursor: not-allowed;
	}

	.toolbar-btn.danger {
		color: #ef4444;
	}

	.toolbar-btn.danger:hover {
		background: rgba(239, 68, 68, 0.1);
	}

	.toolbar-btn svg {
		width: 16px;
		height: 16px;
	}

	.file-breadcrumb {
		flex: 1;
		display: flex;
		align-items: center;
		gap: 4px;
		font-size: 13px;
		min-width: 0;
		overflow-x: auto;
	}

	.file-breadcrumb button {
		border: none;
		background: none;
		cursor: pointer;
		padding: 4px 8px;
		border-radius: 4px;
		color: var(--dt);
		white-space: nowrap;
	}

	.file-breadcrumb button:hover {
		background: rgba(0, 0, 0, 0.05);
	}

	.breadcrumb-sep {
		color: #9ca3af;
	}

	.toolbar-right {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.search-box {
		display: flex;
		align-items: center;
		gap: 6px;
		background: var(--dbg);
		border: 1px solid var(--dbd);
		border-radius: 6px;
		padding: 4px 8px;
	}

	.search-box svg {
		width: 14px;
		height: 14px;
		color: #9ca3af;
	}

	.search-box input {
		border: none;
		background: none;
		outline: none;
		width: 120px;
		font-size: 12px;
		color: var(--dt);
	}

	.search-box input::placeholder {
		color: #9ca3af;
	}

	.view-toggle {
		display: flex;
		border: 1px solid var(--dbd);
		border-radius: 6px;
		overflow: hidden;
	}

	.view-toggle button {
		width: 28px;
		height: 26px;
		border: none;
		background: var(--dbg);
		cursor: pointer;
		display: flex;
		align-items: center;
		justify-content: center;
		color: var(--dt2);
	}

	.view-toggle button:not(:last-child) {
		border-right: 1px solid var(--dbd);
	}

	.view-toggle button.active {
		background: var(--dbg2);
		color: var(--dt);
	}

	.view-toggle button svg {
		width: 14px;
		height: 14px;
	}

	/* Loading and Error States */
	.loading-state, .error-state {
		flex: 1;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 16px;
		color: var(--dt2);
	}

	.spinner {
		width: 32px;
		height: 32px;
		border: 3px solid var(--dbd);
		border-top-color: #3B82F6;
		border-radius: 50%;
		animation: spin 1s linear infinite;
	}

	@keyframes spin {
		to { transform: rotate(360deg); }
	}

	.error-state svg {
		width: 48px;
		height: 48px;
		color: #ef4444;
	}

	.error-state button {
		padding: 8px 16px;
		background: #3B82F6;
		color: white;
		border: none;
		border-radius: 6px;
		cursor: pointer;
	}

	/* File List */
	.file-list {
		flex: 1;
		overflow-y: auto;
	}

	.file-list-header {
		display: flex;
		padding: 8px 16px;
		border-bottom: 1px solid var(--dbd);
		font-size: 11px;
		font-weight: 600;
		color: var(--dt2);
		text-transform: uppercase;
		background: var(--dbg2);
		position: sticky;
		top: 0;
	}

	.file-list-item {
		display: flex;
		align-items: center;
		padding: 8px 16px;
		border: none;
		background: none;
		width: 100%;
		cursor: pointer;
		text-align: left;
		border-bottom: 1px solid var(--dbd);
		color: var(--dt);
	}

	.file-list-item:hover {
		background: var(--dbg2);
	}

	.file-list-item.selected {
		background: rgba(59, 130, 246, 0.1);
	}

	.file-list-item.hidden-file {
		opacity: 0.6;
	}

	.col-name {
		flex: 1;
		display: flex;
		align-items: center;
		gap: 10px;
		min-width: 0;
	}

	.col-date {
		width: 140px;
		color: var(--dt2);
		flex-shrink: 0;
	}

	.col-size {
		width: 80px;
		text-align: right;
		color: var(--dt2);
		flex-shrink: 0;
	}

	.item-icon {
		width: 20px;
		height: 20px;
		flex-shrink: 0;
	}

	.item-icon.folder {
		color: #3B82F6;
	}

	/* Grid View */
	.file-grid {
		flex: 1;
		overflow-y: auto;
		padding: 16px;
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(90px, 1fr));
		gap: 16px;
		align-content: start;
	}

	.file-grid-item {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 8px;
		padding: 12px 8px;
		border: none;
		background: none;
		border-radius: 8px;
		cursor: pointer;
		color: var(--dt);
	}

	.file-grid-item:hover {
		background: var(--dbg2);
	}

	.file-grid-item.selected {
		background: rgba(59, 130, 246, 0.1);
	}

	.file-grid-item.hidden-file {
		opacity: 0.6;
	}

	.grid-icon {
		width: 48px;
		height: 48px;
	}

	.grid-name {
		font-size: 12px;
		text-align: center;
		max-width: 80px;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	/* Empty state */
	.empty-folder {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: 60px 20px;
		color: #9ca3af;
	}

	.empty-folder svg {
		width: 64px;
		height: 64px;
		opacity: 0.3;
		margin-bottom: 16px;
	}

	/* Status Bar */
	.file-status {
		display: flex;
		gap: 16px;
		padding: 6px 16px;
		border-top: 1px solid var(--dbd);
		background: var(--dbg2);
		font-size: 11px;
		color: var(--dt2);
	}

	/* Dark mode overrides */
	:global(.dark) .file-browser {
		background: #1c1c1e;
		color: #f5f5f7;
	}

	:global(.dark) .file-sidebar {
		background: #2c2c2e;
		border-right-color: rgba(255, 255, 255, 0.1);
	}

	:global(.dark) .sidebar-header h3 {
		color: #8e8e93;
	}

	:global(.dark) .sidebar-item {
		color: #e5e5ea;
	}

	:global(.dark) .sidebar-item:hover {
		background: rgba(255, 255, 255, 0.08);
	}

	:global(.dark) .sidebar-item.active {
		background: rgba(59, 130, 246, 0.2);
		color: #60a5fa;
	}

	:global(.dark) .sidebar-divider {
		background: rgba(255, 255, 255, 0.1);
	}

	:global(.dark) .sidebar-checkbox {
		color: #a1a1a6;
	}

	:global(.dark) .file-main {
		background: #1c1c1e;
	}

	:global(.dark) .file-toolbar {
		border-bottom-color: rgba(255, 255, 255, 0.1);
		background: #2c2c2e;
	}

	:global(.dark) .toolbar-btn {
		color: #e5e5ea;
	}

	:global(.dark) .toolbar-btn:hover:not(:disabled) {
		background: rgba(255, 255, 255, 0.1);
	}

	:global(.dark) .toolbar-btn.danger:hover {
		background: rgba(239, 68, 68, 0.2);
	}

	:global(.dark) .file-breadcrumb button {
		color: #e5e5ea;
	}

	:global(.dark) .file-breadcrumb button:hover {
		background: rgba(255, 255, 255, 0.1);
	}

	:global(.dark) .breadcrumb-sep {
		color: #6e6e73;
	}

	:global(.dark) .search-box {
		background: #3a3a3c;
		border-color: rgba(255, 255, 255, 0.1);
	}

	:global(.dark) .search-box svg {
		color: #8e8e93;
	}

	:global(.dark) .search-box input {
		color: #f5f5f7;
	}

	:global(.dark) .search-box input::placeholder {
		color: #6e6e73;
	}

	:global(.dark) .view-toggle {
		border-color: rgba(255, 255, 255, 0.1);
	}

	:global(.dark) .view-toggle button {
		background: #3a3a3c;
		color: #a1a1a6;
	}

	:global(.dark) .view-toggle button:not(:last-child) {
		border-right-color: rgba(255, 255, 255, 0.1);
	}

	:global(.dark) .view-toggle button.active {
		background: #4a4a4c;
		color: #f5f5f7;
	}

	:global(.dark) .loading-state,
	:global(.dark) .error-state {
		color: #a1a1a6;
	}

	:global(.dark) .spinner {
		border-color: #3a3a3c;
		border-top-color: #3B82F6;
	}

	:global(.dark) .file-list-header {
		border-bottom-color: rgba(255, 255, 255, 0.1);
		color: #8e8e93;
		background: #2c2c2e;
	}

	:global(.dark) .file-list-item {
		border-bottom-color: rgba(255, 255, 255, 0.05);
		color: #f5f5f7;
	}

	:global(.dark) .file-list-item:hover {
		background: rgba(255, 255, 255, 0.05);
	}

	:global(.dark) .file-list-item.selected {
		background: rgba(59, 130, 246, 0.2);
	}

	:global(.dark) .col-date,
	:global(.dark) .col-size {
		color: #a1a1a6;
	}

	:global(.dark) .file-grid-item {
		color: #f5f5f7;
	}

	:global(.dark) .file-grid-item:hover {
		background: rgba(255, 255, 255, 0.08);
	}

	:global(.dark) .file-grid-item.selected {
		background: rgba(59, 130, 246, 0.2);
	}

	:global(.dark) .empty-folder {
		color: #6e6e73;
	}

	:global(.dark) .file-status {
		border-top-color: rgba(255, 255, 255, 0.1);
		background: #2c2c2e;
		color: #8e8e93;
	}
</style>
