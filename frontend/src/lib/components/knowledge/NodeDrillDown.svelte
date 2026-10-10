<script lang="ts">
	/**
	 * NodeDrillDown — Breadcrumb-based file explorer for the Pages "All Pages" view.
	 * Level 0: 13 node cards in a grid (slug, name, type, signal_count).
	 * Level 1+: current folder's children (folders + .md files) with breadcrumb trail.
	 * Clicking a folder drills deeper; clicking a .md file emits onOpenFile.
	 */

	import { getApiBaseUrl, getCSRFToken } from '$lib/api/base';

	// ─── Types ────────────────────────────────────────────────────────────────

	interface NodeInfo {
		slug: string;
		name: string;
		type: string;
		signal_count: number;
	}

	interface FileEntry {
		name: string;
		path: string;
		is_dir: boolean;
		size: number;
		children?: FileEntry[];
	}

	interface Props {
		nodes: NodeInfo[];
		onOpenFile?: (filePath: string) => void;
	}

	// ─── Props ────────────────────────────────────────────────────────────────

	let { nodes, onOpenFile }: Props = $props();

	// ─── State ────────────────────────────────────────────────────────────────

	// path[0] = node slug (e.g. '02-miosa'), path[1+] = sub-folder names
	let path = $state<string[]>([]);
	let currentItems = $state<FileEntry[]>([]);
	let loading = $state(false);
	let fetchError = $state<string | null>(null);

	// Cache: nodeSlug → full FileEntry tree (avoid re-fetching on back-nav)
	const treeCache = new Map<string, FileEntry[]>();

	// ─── Derived ─────────────────────────────────────────────────────────────

	// The node slug is always path[0] when we're inside a node
	let activeSlug = $derived(path[0] ?? null);

	// Breadcrumb labels: ['All Pages', 'MIOSA LLC', 'platform', 'projects']
	let breadcrumbs = $derived<string[]>([
		'All Pages',
		...path.map((seg, i) => {
			if (i === 0) {
				// Replace slug with human-readable node name
				return nodes.find((n) => n.slug === seg)?.name ?? seg;
			}
			return seg;
		})
	]);

	// ─── Helpers ─────────────────────────────────────────────────────────────

	function buildHeaders(): Record<string, string> {
		const h: Record<string, string> = {};
		const csrf = getCSRFToken();
		if (csrf) h['X-CSRF-Token'] = csrf;
		return h;
	}

	/** Fetch and cache the file tree for a node slug. */
	async function fetchTree(slug: string): Promise<FileEntry[]> {
		if (treeCache.has(slug)) return treeCache.get(slug)!;
		const res = await fetch(
			`${getApiBaseUrl()}/optimal/nodes/${encodeURIComponent(slug)}/files`,
			{ headers: buildHeaders(), credentials: 'include', signal: AbortSignal.timeout(8000) }
		);
		if (!res.ok) throw new Error(`HTTP ${res.status}`);
		const data: { files?: FileEntry[] } = await res.json();
		const files = Array.isArray(data.files) ? data.files : [];
		treeCache.set(slug, files);
		return files;
	}

	/**
	 * Walk the cached tree along path[1..] to find children at the current depth.
	 * path = ['02-miosa', 'platform', 'projects'] → find 'platform' at root, then
	 * 'projects' inside it, and return its children.
	 */
	function walkToCurrentDir(tree: FileEntry[], segments: string[]): FileEntry[] {
		let entries = tree;
		for (const seg of segments) {
			const dir = entries.find((e) => e.is_dir && e.name === seg);
			if (!dir || !dir.children) return [];
			entries = dir.children;
		}
		return entries;
	}

	// ─── Navigation ──────────────────────────────────────────────────────────

	/** Drill into a node from Level 0. */
	async function drillIntoNode(slug: string) {
		loading = true;
		fetchError = null;
		try {
			const tree = await fetchTree(slug);
			path = [slug];
			currentItems = tree;
		} catch (e) {
			fetchError = e instanceof Error ? e.message : 'Failed to load files';
		} finally {
			loading = false;
		}
	}

	/** Drill into a sub-folder at the current level. */
	async function drillIntoFolder(entry: FileEntry) {
		if (!activeSlug) return;
		loading = true;
		fetchError = null;
		try {
			const tree = await fetchTree(activeSlug);
			const newPath = [...path, entry.name];
			// Walk from path[1..] to the new depth
			const children = walkToCurrentDir(tree, newPath.slice(1));
			path = newPath;
			currentItems = children;
		} catch (e) {
			fetchError = e instanceof Error ? e.message : 'Failed to load folder';
		} finally {
			loading = false;
		}
	}

	/** Navigate to a breadcrumb index (0 = All Pages root). */
	async function navigateToBreadcrumb(index: number) {
		if (index === 0) {
			// Back to All Pages
			path = [];
			currentItems = [];
			fetchError = null;
			return;
		}
		// index 1 = node root, index 2 = first sub-folder, etc.
		// path segments: index 1 → path[0], index 2 → path[0..1], etc.
		const targetPath = path.slice(0, index);
		if (!targetPath[0]) return;
		loading = true;
		fetchError = null;
		try {
			const tree = await fetchTree(targetPath[0]);
			const children = targetPath.length > 1
				? walkToCurrentDir(tree, targetPath.slice(1))
				: tree;
			path = targetPath;
			currentItems = children;
		} catch (e) {
			fetchError = e instanceof Error ? e.message : 'Failed to load';
		} finally {
			loading = false;
		}
	}

	/** Go up one level. */
	function goBack() {
		navigateToBreadcrumb(path.length - 1);
	}

	/** Open a .md file — emit to parent. */
	function openFile(entry: FileEntry) {
		onOpenFile?.(entry.path);
	}

	// ─── Item count for folder entries ───────────────────────────────────────

	function childCount(entry: FileEntry): number {
		return entry.children?.length ?? 0;
	}

	// ─── Node type badge color ────────────────────────────────────────────────

	const TYPE_COLORS: Record<string, string> = {
		person: '#6366f1',
		entity: '#10b981',
		'operation:program': '#f59e0b',
		'operation:research': '#3b82f6',
		'operation:network': '#ec4899',
		'unit:community': '#8b5cf6',
		'domain:media': '#ef4444',
		'layer:command': '#64748b',
		'layer:operating': '#0ea5e9',
		'layer:tracking': '#f97316',
		'cross-cutting': '#84cc16',
		inbox: '#a8a29e',
	};

	function typeColor(type: string): string {
		return TYPE_COLORS[type] ?? '#6b7280';
	}
</script>

<div class="ndd">
	{#if path.length === 0}
		<!-- ─── Level 0: Node Grid ──────────────────────────────────────────── -->
		<div class="ndd__header">
			<h1 class="ndd__title">All Pages</h1>
			<span class="ndd__stats">{nodes.length} nodes</span>
		</div>

		{#if nodes.length === 0}
			<div class="ndd__empty">
				<p class="ndd__empty-desc">No nodes found. Check that the backend is running.</p>
			</div>
		{:else}
			<div class="ndd__grid">
				{#each nodes as node (node.slug)}
					<button
						class="ndd__node-card"
						onclick={() => drillIntoNode(node.slug)}
						aria-label="Open node {node.name}"
					>
						<div class="ndd__node-card__top">
							<span class="ndd__node-card__num">{node.slug.split('-')[0]}</span>
							<span
								class="ndd__node-card__badge"
								style="background: {typeColor(node.type)}22; color: {typeColor(node.type)}"
							>{node.type}</span>
						</div>
						<span class="ndd__node-card__name">{node.name}</span>
						{#if node.signal_count > 0}
							<span class="ndd__node-card__signals">{node.signal_count} signals</span>
						{/if}
						<!-- chevron -->
						<svg class="ndd__node-card__chevron" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M9 18l6-6-6-6"/></svg>
					</button>
				{/each}
			</div>
		{/if}

	{:else}
		<!-- ─── Level 1+: Breadcrumb + Children ────────────────────────────── -->

		<!-- Breadcrumb bar -->
		<div class="ndd__breadcrumb-bar">
			<button
				class="ndd__back-btn"
				onclick={goBack}
				aria-label="Go back"
			>
				<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M19 12H5M12 5l-7 7 7 7"/></svg>
			</button>
			<nav class="ndd__breadcrumb" aria-label="Navigation breadcrumb">
				{#each breadcrumbs as crumb, i (i)}
					{#if i < breadcrumbs.length - 1}
						<button
							class="ndd__crumb ndd__crumb--link"
							onclick={() => navigateToBreadcrumb(i)}
							aria-label="Go to {crumb}"
						>{crumb}</button>
						<span class="ndd__crumb-sep" aria-hidden="true">/</span>
					{:else}
						<span class="ndd__crumb ndd__crumb--active" aria-current="page">{crumb}</span>
					{/if}
				{/each}
			</nav>
		</div>

		<!-- Content -->
		{#if loading}
			<div class="ndd__loading">
				<div class="ndd__spinner" aria-hidden="true"></div>
				<span>Loading...</span>
			</div>
		{:else if fetchError}
			<div class="ndd__empty">
				<p class="ndd__empty-desc">{fetchError}</p>
				<button class="ndd__retry-btn" onclick={() => navigateToBreadcrumb(path.length)}>
					Retry
				</button>
			</div>
		{:else if currentItems.length === 0}
			<div class="ndd__empty">
				<p class="ndd__empty-desc">This folder is empty.</p>
			</div>
		{:else}
			<!-- Sort: directories first, then files, both alphabetical -->
			{@const sorted = [...currentItems].sort((a, b) => {
				if (a.is_dir !== b.is_dir) return a.is_dir ? -1 : 1;
				return a.name.localeCompare(b.name);
			})}
			<div class="ndd__list">
				{#each sorted as entry (entry.path)}
					{#if entry.is_dir}
						<button
							class="ndd__item ndd__item--dir"
							onclick={() => drillIntoFolder(entry)}
							aria-label="Open folder {entry.name}"
						>
							<span class="ndd__item__icon" aria-hidden="true">
								<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z"/></svg>
							</span>
							<span class="ndd__item__name">{entry.name}</span>
							{#if childCount(entry) > 0}
								<span class="ndd__item__count">{childCount(entry)}</span>
							{/if}
							<svg class="ndd__item__chevron" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M9 18l6-6-6-6"/></svg>
						</button>
					{:else if entry.name.endsWith('.md')}
						<button
							class="ndd__item ndd__item--file"
							onclick={() => openFile(entry)}
							aria-label="Open file {entry.name}"
						>
							<span class="ndd__item__icon" aria-hidden="true">
								<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/><polyline points="14,2 14,8 20,8"/></svg>
							</span>
							<span class="ndd__item__name">{entry.name}</span>
						</button>
					{/if}
				{/each}
			</div>
		{/if}
	{/if}
</div>

<style>
	/* ─── Container ─────────────────────────────────────────────────────────── */
	.ndd {
		flex: 1;
		display: flex;
		flex-direction: column;
		max-width: 900px;
		width: 100%;
		margin: 0 auto;
		padding: 40px 32px;
		font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
		color: var(--bos-v2-text-primary, #121212);
	}

	/* ─── Level-0 header ─────────────────────────────────────────────────────── */
	.ndd__header {
		display: flex;
		align-items: baseline;
		justify-content: space-between;
		margin-bottom: 28px;
	}

	.ndd__title {
		font-size: 24px;
		font-weight: 600;
		margin: 0;
		color: var(--bos-v2-text-primary, #121212);
	}

	.ndd__stats {
		font-size: 13px;
		color: var(--bos-v2-text-tertiary, #a1a1aa);
	}

	/* ─── Node card grid ─────────────────────────────────────────────────────── */
	.ndd__grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
		gap: 12px;
	}

	.ndd__node-card {
		position: relative;
		display: flex;
		flex-direction: column;
		gap: 6px;
		padding: 16px;
		border-radius: 10px;
		border: 1px solid var(--bos-v2-layer-insideBorder-border, rgba(0, 0, 0, 0.09));
		background: var(--bos-v2-layer-background-secondary, #f4f4f5);
		cursor: pointer;
		text-align: left;
		transition: background 0.15s, border-color 0.15s, box-shadow 0.15s;
	}

	.ndd__node-card:hover {
		background: var(--bos-v2-layer-background-tertiary, #eeeef0);
		border-color: var(--bos-brand-color, #1e96eb);
		box-shadow: 0 1px 6px rgba(30, 150, 235, 0.12);
	}

	.ndd__node-card__top {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 8px;
	}

	.ndd__node-card__num {
		font-size: 11px;
		font-weight: 600;
		color: var(--bos-v2-text-tertiary, #a1a1aa);
		letter-spacing: 0.04em;
	}

	.ndd__node-card__badge {
		font-size: 10px;
		font-weight: 500;
		padding: 2px 7px;
		border-radius: 99px;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
		max-width: 130px;
	}

	.ndd__node-card__name {
		font-size: 14px;
		font-weight: 600;
		color: var(--bos-v2-text-primary, #121212);
		line-height: 1.3;
	}

	.ndd__node-card__signals {
		font-size: 11px;
		color: var(--bos-v2-text-tertiary, #a1a1aa);
	}

	.ndd__node-card__chevron {
		position: absolute;
		bottom: 14px;
		right: 14px;
		color: var(--bos-v2-text-tertiary, #a1a1aa);
		opacity: 0;
		transition: opacity 0.15s;
	}

	.ndd__node-card:hover .ndd__node-card__chevron {
		opacity: 1;
	}

	/* ─── Breadcrumb bar ─────────────────────────────────────────────────────── */
	.ndd__breadcrumb-bar {
		display: flex;
		align-items: center;
		gap: 10px;
		margin-bottom: 24px;
		min-height: 36px;
	}

	.ndd__back-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 32px;
		height: 32px;
		border-radius: 8px;
		border: 1px solid var(--bos-v2-layer-insideBorder-border, rgba(0, 0, 0, 0.09));
		background: var(--bos-v2-layer-background-secondary, #f4f4f5);
		color: var(--bos-v2-text-secondary, #8e8d91);
		cursor: pointer;
		flex-shrink: 0;
		transition: background 0.15s, color 0.15s;
	}

	.ndd__back-btn:hover {
		background: var(--bos-v2-layer-background-tertiary, #eeeef0);
		color: var(--bos-v2-text-primary, #121212);
	}

	.ndd__breadcrumb {
		display: flex;
		align-items: center;
		gap: 4px;
		flex-wrap: wrap;
		min-width: 0;
	}

	.ndd__crumb {
		font-size: 13px;
		font-weight: 500;
		white-space: nowrap;
	}

	.ndd__crumb--link {
		color: var(--bos-brand-color, #1e96eb);
		background: none;
		border: none;
		padding: 2px 4px;
		border-radius: 4px;
		cursor: pointer;
		transition: background 0.12s;
	}

	.ndd__crumb--link:hover {
		background: rgba(30, 150, 235, 0.08);
	}

	.ndd__crumb--active {
		color: var(--bos-v2-text-primary, #121212);
		font-weight: 600;
	}

	.ndd__crumb-sep {
		color: var(--bos-v2-text-tertiary, #a1a1aa);
		font-size: 13px;
		user-select: none;
	}

	/* ─── File/folder list ───────────────────────────────────────────────────── */
	.ndd__list {
		display: flex;
		flex-direction: column;
		gap: 2px;
	}

	.ndd__item {
		display: flex;
		align-items: center;
		gap: 10px;
		padding: 9px 12px;
		border-radius: 8px;
		border: none;
		background: none;
		cursor: pointer;
		text-align: left;
		width: 100%;
		transition: background 0.12s;
	}

	.ndd__item:hover {
		background: var(--bos-v2-layer-background-secondary, #f4f4f5);
	}

	.ndd__item__icon {
		display: flex;
		align-items: center;
		flex-shrink: 0;
	}

	.ndd__item--dir .ndd__item__icon {
		color: var(--bos-brand-color, #1e96eb);
	}

	.ndd__item--file .ndd__item__icon {
		color: var(--bos-v2-text-tertiary, #a1a1aa);
	}

	.ndd__item__name {
		flex: 1;
		font-size: 14px;
		color: var(--bos-v2-text-primary, #121212);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.ndd__item--dir .ndd__item__name {
		font-weight: 500;
	}

	.ndd__item__count {
		font-size: 11px;
		color: var(--bos-v2-text-tertiary, #a1a1aa);
		background: var(--bos-v2-layer-background-secondary, #f4f4f5);
		padding: 2px 7px;
		border-radius: 99px;
		flex-shrink: 0;
	}

	.ndd__item__chevron {
		color: var(--bos-v2-text-tertiary, #a1a1aa);
		flex-shrink: 0;
		opacity: 0;
		transition: opacity 0.12s;
	}

	.ndd__item:hover .ndd__item__chevron {
		opacity: 1;
	}

	/* ─── Loading state ──────────────────────────────────────────────────────── */
	.ndd__loading {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 24px 12px;
		font-size: 13px;
		color: var(--bos-v2-text-secondary, #8e8d91);
	}

	.ndd__spinner {
		width: 20px;
		height: 20px;
		border: 2px solid var(--bos-v2-layer-insideBorder-border, rgba(0, 0, 0, 0.1));
		border-top-color: var(--bos-brand-color, #1e96eb);
		border-radius: 50%;
		animation: ndd-spin 0.7s linear infinite;
	}

	@keyframes ndd-spin {
		to { transform: rotate(360deg); }
	}

	/* ─── Empty / error state ────────────────────────────────────────────────── */
	.ndd__empty {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 12px;
		padding: 48px 24px;
	}

	.ndd__empty-desc {
		margin: 0;
		font-size: 14px;
		color: var(--bos-v2-text-secondary, #8e8d91);
	}

	.ndd__retry-btn {
		height: 30px;
		padding: 0 14px;
		font-size: 13px;
		font-weight: 500;
		border-radius: 8px;
		border: 1px solid var(--bos-v2-layer-insideBorder-border, rgba(0, 0, 0, 0.1));
		background: var(--bos-v2-layer-background-secondary, #f4f4f5);
		color: var(--bos-v2-text-primary, #121212);
		cursor: pointer;
		transition: background 0.15s;
	}

	.ndd__retry-btn:hover {
		background: var(--bos-v2-layer-background-tertiary, #eeeef0);
	}
</style>
