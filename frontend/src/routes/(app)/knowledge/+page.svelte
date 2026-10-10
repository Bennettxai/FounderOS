<script lang="ts">
	import { onMount } from 'svelte';
	import {
		fetchWorkspaces,
		fetchTree,
		fetchFile,
		saveFile,
		syncToCloud,
		fetchSources,
		getStorage,
		activateCloudSync,
		formatBytes,
		slugify,
		splitFrontmatter,
		parseFrontmatter,
		type KBWorkspace,
		type StorageUsage
	} from '$lib/kb/client';
	import type { KBTreeNode } from '$lib/kb/types';
	import { getEngineKnowledge, type EngineKnowledgeItem } from '$lib/api/knowledge';
	import { getDeliverables, type Deliverable } from '$lib/api/deliverables';
	import { currentWorkspace } from '$lib/stores/workspaces';
	import { marked } from 'marked';
	import {
		FileText, Folder, ChevronRight, Loader2, Hash, Clock, Sparkles, Download,
		BookOpen, Users, GitBranch, ClipboardList, Link2, Package, Bot,
		Pencil, Plus, Check, X, PanelLeft, PanelRight, CloudUpload, ExternalLink, ShieldCheck
	} from 'lucide-svelte';

	let workspaces = $state<KBWorkspace[]>([]);
	let slug = $state('');
	let lastShellSlug = $state('');

	let rawTree = $state<KBTreeNode[]>([]);
	// Per-doc source: "engine" (your local), "cloud" (shared copy), "synced" (both).
	let sources = $state<Record<string, string>>({});
	let sourceCounts = $state<{ engine: number; cloud: number; synced: number }>({ engine: 0, cloud: 0, synced: 0 });
	let sourceFilter = $state<'all' | 'engine' | 'cloud' | 'synced'>('all');
	let loadingTree = $state(false);
	let treeError = $state<string | null>(null);

	let tab = $state('docs');
	let filter = $state('');

	let selectedPath = $state('');
	let selectedTitle = $state('');
	let bodyHtml = $state('');
	let properties = $state<Record<string, string>>({});
	let modified = $state('');
	let toc = $state<{ level: number; text: string; id: string }[]>([]);
	let loadingDoc = $state(false);

	// Editing
	let rawContent = $state('');
	let editing = $state(false);
	let draft = $state('');
	let saving = $state(false);
	let saveError = $state<string | null>(null);

	// Cloud sync opt-in gate. Local knowledge is always on + free; cloud sync is
	// the opt-in, paid-later layer. Nothing syncs until the user activates it.
	let cloudActivated = $state(false);
	let overLimit = $state(false);
	let storage = $state<StorageUsage | null>(null);
	let activating = $state(false);
	let activateErr = $state<string | null>(null);

	// Load cloud-sync status (activated + usage) for the current workspace.
	async function loadCloudStatus(ws: string) {
		try {
			const s = await getStorage(ws);
			if (slug !== ws) return; // stale
			storage = s;
			cloudActivated = s.activated;
			overLimit = s.over_limit;
		} catch {
			if (slug === ws) {
				storage = null;
				cloudActivated = false;
				overLimit = false;
			}
		}
	}

	async function doActivate() {
		if (activating || !slug) return;
		activating = true;
		activateErr = null;
		try {
			await activateCloudSync(slug);
			await loadCloudStatus(slug);
		} catch (e) {
			activateErr = e instanceof Error ? e.message : 'Activation failed';
		} finally {
			activating = false;
		}
	}

	// Sync this workspace's local knowledge up to the shared cloud copy.
	let syncing = $state(false);
	let syncMsg = $state<string | null>(null);

	async function doSync() {
		if (syncing || !slug) return;
		syncing = true;
		syncMsg = null;
		try {
			const res = await syncToCloud(slug, $currentWorkspace?.id);
			syncMsg = `Synced ${res.documents} docs to the team`;
			if (res.storage) {
				storage = { ...res.storage, activated: true, over_limit: res.storage.over_limit ?? false };
				overLimit = res.storage.over_limit ?? false;
			}
			void loadCloudStatus(slug);
		} catch (e) {
			syncMsg = e instanceof Error ? e.message : 'Sync failed';
		} finally {
			syncing = false;
			setTimeout(() => (syncMsg = null), 6000);
		}
	}

	// ── Sources tab: canonical engine-indexed sources for this workspace ──
	let engineItems = $state<EngineKnowledgeItem[]>([]);
	let engineLinked = $state(true);
	let loadingEngine = $state(false);
	let engineError = $state<string | null>(null);
	let engineLoadedSlug = $state('');

	// ── Packages tab: generated deliverables/exports for this workspace ──
	let deliverables = $state<Deliverable[]>([]);
	let loadingDeliverables = $state(false);
	let deliverablesError = $state<string | null>(null);
	let deliverablesLoadedSlug = $state('');

	// Load the workspace's canonical engine sources (Sources + Agent Context tabs).
	// Scoped by the X-Workspace-ID header inside request<T>(); guarded against a
	// stale response arriving after the slug changed.
	async function loadEngine() {
		const requested = slug;
		loadingEngine = true;
		engineError = null;
		try {
			const res = await getEngineKnowledge();
			if (slug !== requested) return;
			engineItems = res.items ?? [];
			engineLinked = res.engine_linked ?? false;
		} catch (e) {
			if (slug === requested) engineError = (e as Error).message;
		} finally {
			if (slug === requested) loadingEngine = false;
		}
	}

	// Load the workspace's deliverables/packages (Packages + Agent Context tabs).
	async function loadDeliverables() {
		const requested = slug;
		loadingDeliverables = true;
		deliverablesError = null;
		try {
			const res = await getDeliverables();
			if (slug !== requested) return;
			deliverables = res.deliverables ?? [];
		} catch (e) {
			if (slug === requested) deliverablesError = (e as Error).message;
		} finally {
			if (slug === requested) loadingDeliverables = false;
		}
	}

	// Lazy-load each real layer the first time its tab (or Agent Context, which
	// summarizes both) is shown for the current workspace. Guarded by the loaded
	// slug so switching back and forth doesn't refetch, and so a workspace change
	// (slug !== loaded) transparently reloads.
	$effect(() => {
		const t = tab;
		const s = slug;
		const mounted = workspaceMounted;
		if (!s) return;
		const wantEngine = (t === 'sources' || t === 'agent') && mounted;
		const wantDeliverables = t === 'packages' || t === 'agent';
		if (wantEngine && engineLoadedSlug !== s) {
			engineLoadedSlug = s;
			void loadEngine();
		}
		if (wantDeliverables && deliverablesLoadedSlug !== s) {
			deliverablesLoadedSlug = s;
			void loadDeliverables();
		}
	});

	let expanded = $state<Set<string>>(new Set());

	// Collapsible panels
	let treeOpen = $state(true);
	let railOpen = $state(true);

	// Mobile tree drawer — on <=768px the tree becomes an overlay
	let mobileTreeOpen = $state(false);

	const TABS = [
		{ id: 'docs', label: 'Docs', icon: BookOpen },
		{ id: 'nodes', label: 'Nodes', icon: Users },
		{ id: 'decisions', label: 'Decisions', icon: GitBranch },
		{ id: 'sops', label: 'SOPs', icon: ClipboardList },
		{ id: 'sources', label: 'Sources', icon: Link2 },
		{ id: 'packages', label: 'Packages', icon: Package },
		{ id: 'agent', label: 'Agent Context', icon: Bot }
	];

	const PLACEHOLDERS: Record<string, { title: string; body: string }> = {
		sources: { title: 'Sources', body: 'Meetings (Fathom), uploads, transcripts, and citations that back this workspace. Wire connectors to populate this layer.' },
		packages: { title: 'Packages', body: 'Generated deliverables and exports (PDFs, proposals, briefs) produced from this workspace. Built in the Deliverables flow.' },
		agent: { title: 'Agent Context', body: 'What the agent is allowed to read, what is trusted, what changed recently. The control layer for safe agent operation.' }
	};

	const HIDE = new Set(['readme.md', 'agents.md', 'schema.md']);

	// Curate raw filesystem: folder README becomes the folder's page; hide plumbing.
	function curate(nodes: KBTreeNode[]): KBTreeNode[] {
		const out: KBTreeNode[] = [];
		for (const n of nodes) {
			if (n.type === 'file') {
				if (HIDE.has(n.name.toLowerCase())) continue;
				out.push(n);
				continue;
			}
			const kids = curate(n.children ?? []);
			const readme = (n.children ?? []).find((c) => c.type === 'file' && c.name.toLowerCase() === 'readme.md');
			const dir: KBTreeNode = { ...n, children: kids, indexPath: readme?.path };
			if (kids.length > 0 || dir.indexPath) out.push(dir);
		}
		return out;
	}

	function flattenFiles(nodes: KBTreeNode[], pred: (n: KBTreeNode) => boolean): KBTreeNode[] {
		const out: KBTreeNode[] = [];
		for (const n of nodes) {
			if (n.type === 'file') {
				if (pred(n)) out.push(n);
			} else out.push(...flattenFiles(n.children ?? [], pred));
		}
		return out;
	}

	// Nodes layer: nodes/ subfolders become entity pages (context.md = body, signal.md = sub).
	function buildNodesTree(nodes: KBTreeNode[]): KBTreeNode[] {
		const dir = nodes.find((n) => n.type === 'dir' && n.name === 'nodes');
		if (!dir?.children) return [];
		return dir.children
			.filter((e) => e.type === 'dir')
			.map((e) => {
				const ctx = (e.children ?? []).find((c) => c.name.toLowerCase() === 'context.md');
				const others = (e.children ?? []).filter((c) => c.name.toLowerCase() !== 'context.md');
				return { ...e, indexPath: ctx?.path, children: others } as KBTreeNode;
			});
	}

	// The tree shown for a given tab (re-slices the same files by meaning).
	function treeFor(id: string, src: KBTreeNode[]): KBTreeNode[] {
		const curated = curate(src);
		if (id === 'docs') return curated.filter((n) => n.name !== 'nodes');
		if (id === 'nodes') return buildNodesTree(src);
		if (id === 'decisions') return flattenFiles(src, (n) => /decision/i.test(n.name));
		if (id === 'sops') return flattenFiles(src, (n) => /(sop|playbook|checklist)/i.test(n.name));
		return [];
	}

	const activeTree = $derived(treeFor(tab, rawTree));
	const isPlaceholder = $derived(!!PLACEHOLDERS[tab]);

	// Agent Context scope: what the agent can read, built entirely from data
	// already loaded on the page (tree + source map) plus the source/deliverable
	// counts. No extra backend.
	const topFolders = $derived(curate(rawTree).filter((n) => n.type === 'dir'));
	const totalDocs = $derived(flattenFiles(rawTree, () => true).length);
	const totalSources = $derived(sourceCounts.engine + sourceCounts.cloud + sourceCounts.synced);
	const trustedDocs = $derived(sourceCounts.engine + sourceCounts.synced);
	const workspaceMounted = $derived(workspaces.some((w) => w.slug === slug));
	const workspaceOptions = $derived.by(() => {
		const options = [...workspaces];
		if (slug && !options.some((w) => w.slug === slug)) {
			options.unshift({
				slug,
				name: $currentWorkspace?.name ?? titleize(slug)
			});
		}
		return options;
	});
	const selectedWorkspaceLabel = $derived(
		workspaceOptions.find((w) => w.slug === slug)?.name ?? titleize(slug)
	);

	onMount(async () => {
		try {
			const ws = await fetchWorkspaces();
			workspaces = ws;
			const shellSlug = $currentWorkspace?.slug;
			const preferred =
				(shellSlug ? { slug: shellSlug, name: $currentWorkspace?.name ?? titleize(shellSlug) } : null) ??
				ws.find((w) => w.slug === 'businessos') ??
				ws[0];
			if (preferred) {
				slug = preferred.slug;
				lastShellSlug = slug;
				await loadTree();
			}
		} catch (e) {
			treeError = (e as Error).message;
		}
	});

	$effect(() => {
		const shellSlug = $currentWorkspace?.slug;
		if (!shellSlug || shellSlug === lastShellSlug || workspaces.length === 0) return;
		lastShellSlug = shellSlug;
		slug = shellSlug;
		tab = 'docs';
		void loadTree();
	});

	async function loadTree() {
		if (!slug) return;
		// Workspace-scoped load: clear the previous workspace's tree AND source map
		// up front, and drop any response that arrives after the slug has changed,
		// so nothing from workspace A ever renders while workspace B is selected.
		const requested = slug;
		loadingTree = true;
		treeError = null;
		rawTree = [];
		sources = {};
		sourceCounts = { engine: 0, cloud: 0, synced: 0 };
		storage = null;
		cloudActivated = false;
		overLimit = false;
		activateErr = null;
		// Clear the real-layer caches so workspace B never shows workspace A's
		// sources/packages; the $effect reloads them lazily per active tab.
		engineItems = [];
		engineLinked = true;
		engineError = null;
		engineLoadedSlug = '';
		deliverables = [];
		deliverablesError = null;
		deliverablesLoadedSlug = '';
		resetDoc();
		// Cloud-sync status is independent of whether local files are mounted.
		void loadCloudStatus(requested);
		if (!workspaceMounted) {
			loadingTree = false;
			return;
		}
		try {
			const tree = await fetchTree(requested);
			if (slug !== requested) return; // workspace changed mid-flight; stale response
			rawTree = tree;
			expandActive();
			// Tag each doc with its source (engine / cloud / synced). Best-effort.
			fetchSources(requested)
				.then((s) => {
					if (slug !== requested) return;
					sources = s.sources ?? {};
					sourceCounts = s.counts ?? { engine: 0, cloud: 0, synced: 0 };
				})
				.catch(() => {});
		} catch (e) {
			if (slug === requested) treeError = (e as Error).message;
		} finally {
			if (slug === requested) loadingTree = false;
		}
	}

	function expandActive() {
		const t = treeFor(tab, rawTree);
		expanded = new Set(t.filter((n) => n.type === 'dir').map((n) => n.path));
	}

	function switchTab(id: string) {
		tab = id;
		filter = '';
		resetDoc();
		expandActive();
	}

	function resetDoc() {
		selectedPath = '';
		selectedTitle = '';
		bodyHtml = '';
		properties = {};
		toc = [];
		modified = '';
	}

	function onWorkspaceChange(e: Event) {
		slug = (e.target as HTMLSelectElement).value;
		tab = 'docs';
		loadTree();
	}

	function toggle(path: string) {
		const next = new Set(expanded);
		next.has(path) ? next.delete(path) : next.add(path);
		expanded = next;
	}

	async function renderFromContent(content: string) {
		const { frontmatter, body } = splitFrontmatter(content);
		properties = parseFrontmatter(frontmatter);
		let html = await marked.parse(body);
		const items: { level: number; text: string; id: string }[] = [];
		html = html.replace(/<h([1-3])>([\s\S]*?)<\/h\1>/g, (_m, lvl, inner) => {
			const text = String(inner).replace(/<[^>]+>/g, '').trim();
			const id = 'h-' + text.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '');
			items.push({ level: Number(lvl), text, id });
			return `<h${lvl} id="${id}">${inner}</h${lvl}>`;
		});
		toc = items;
		bodyHtml = html;
	}

	async function openDoc(path: string, title: string) {
		selectedPath = path;
		selectedTitle = title;
		editing = false;
		saveError = null;
		loadingDoc = true;
		bodyHtml = '';
		properties = {};
		toc = [];
		modified = '';
		mobileTreeOpen = false; // close mobile drawer on doc select
		try {
			const file = await fetchFile(slug, path);
			rawContent = file.content;
			modified = file.modified ?? '';
			await renderFromContent(file.content);
		} catch {
			bodyHtml = `<p class="kb-error">Could not load this document.</p>`;
		} finally {
			loadingDoc = false;
		}
	}

	function startEdit() {
		draft = rawContent;
		saveError = null;
		editing = true;
	}

	function cancelEdit() {
		editing = false;
		saveError = null;
	}

	async function save() {
		if (!selectedPath) return;
		saving = true;
		saveError = null;
		try {
			const res = await saveFile(slug, selectedPath, draft);
			rawContent = draft;
			modified = res.modified ?? modified;
			await renderFromContent(draft);
			editing = false;
		} catch (e) {
			saveError = (e as Error).message;
		} finally {
			saving = false;
		}
	}

	async function createPage() {
		if (!workspaceMounted) {
			saveError = 'This workspace does not have a mounted Knowledge folder yet.';
			return;
		}
		const title = window.prompt('New page title');
		if (!title) return;
		const path = `pages/${slugify(title)}.md`;
		const content = `---\ntitle: ${title}\nstatus: draft\n---\n\n# ${title}\n\n`;
		try {
			await saveFile(slug, path, content);
			await loadTree();
			tab = 'docs';
			expandActive();
			await openDoc(path, title);
			startEdit();
		} catch (e) {
			saveError = (e as Error).message;
		}
	}

	function onDirClick(node: KBTreeNode) {
		toggle(node.path);
		if (node.indexPath) openDoc(node.indexPath, node.title ?? node.name);
	}

	function titleize(s: string): string {
		return s
			.replace(/\.md$/i, '')
			.replace(/^\d+[-_]/, '')
			.replace(/[-_]/g, ' ')
			.replace(/\b\w/g, (c) => c.toUpperCase());
	}

	// Current workspace display name (for the breadcrumb root).
	const currentName = $derived(workspaces.find((w) => w.slug === slug)?.name ?? '');

	// Breadcrumb segments (folders only; the file itself is the page title).
	const crumbs = $derived(selectedPath ? selectedPath.split('/').slice(0, -1).map(titleize) : []);

	function fmtDate(s: string): string {
		if (!s) return '';
		const d = new Date(s);
		if (isNaN(+d)) return '';
		return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' });
	}

	const propOrder = ['status', 'owner', 'type', 'layer', 'updated', 'tags'];
	const shownProps = $derived(
		Object.entries(properties)
			.filter(([k]) => k.toLowerCase() !== 'title')
			.sort((a, b) => {
				const ia = propOrder.indexOf(a[0].toLowerCase());
				const ib = propOrder.indexOf(b[0].toLowerCase());
				return (ia < 0 ? 99 : ia) - (ib < 0 ? 99 : ib);
			})
	);

	function matches(node: KBTreeNode, q: string): boolean {
		if (!q) return true;
		const label = (node.title ?? node.name).toLowerCase();
		if (label.includes(q)) return true;
		return (node.children ?? []).some((c) => matches(c, q));
	}

	// Source filter: show only engine / cloud / synced docs (or all). A folder
	// shows if any descendant file matches. No source data => never hide.
	function sourceOk(node: KBTreeNode): boolean {
		if (sourceFilter === 'all' || Object.keys(sources).length === 0) return true;
		if (node.type === 'file') return (sources[node.path] ?? '') === sourceFilter;
		return (node.children ?? []).some(sourceOk);
	}
</script>

<div class="kb-shell">
	<!-- Layer tabs -->
	<nav class="kb-tabs">
		<!-- Desktop panel toggle -->
		<button class="kb-paneltoggle kb-paneltoggle--desktop {treeOpen ? 'kb-paneltoggle--on' : ''}" title="Toggle pages panel" aria-label="Toggle pages panel" onclick={() => (treeOpen = !treeOpen)}>
			<PanelLeft size={16} strokeWidth={1.9} />
		</button>
		<!-- Mobile pages drawer toggle -->
		<button class="kb-paneltoggle kb-paneltoggle--mobile {mobileTreeOpen ? 'kb-paneltoggle--on' : ''}" title="Toggle pages" aria-label="Toggle pages panel" onclick={() => (mobileTreeOpen = !mobileTreeOpen)}>
			<PanelLeft size={16} strokeWidth={1.9} />
		</button>
		{#each TABS as t}
			<button class="kb-tab {tab === t.id ? 'kb-tab--active' : ''}" onclick={() => switchTab(t.id)}>
				<t.icon size={15} strokeWidth={1.9} />
				<span>{t.label}</span>
			</button>
		{/each}
		<div class="kb-tabs-spacer"></div>
		<button
			class="kb-paneltoggle {railOpen ? 'kb-paneltoggle--on' : ''}"
			title="Toggle properties panel"
			aria-label="Toggle properties panel"
			onclick={() => (railOpen = !railOpen)}
		>
			<PanelRight size={16} strokeWidth={1.9} />
		</button>
	</nav>

	<div class="kb">
		<!-- Mobile backdrop — tapping outside closes the tree drawer -->
		{#if mobileTreeOpen}
			<!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
			<div class="kb-mobile-backdrop" onclick={() => (mobileTreeOpen = false)} aria-hidden="true"></div>
		{/if}

		<!-- Left: tree -->
		<aside class="kb-tree {treeOpen ? '' : 'kb-tree--closed'} {mobileTreeOpen ? 'kb-tree--mobile-open' : ''} kb-tree--mobile">
			<div class="kb-tree-head">
				<div class="kb-tree-titlerow">
					<span class="kb-tree-title">{TABS.find((t) => t.id === tab)?.label}</span>
					{#if workspaceOptions.length > 0}
						<select class="kb-ws-select" value={slug} onchange={onWorkspaceChange} aria-label="Workspace">
							{#each workspaceOptions as w}
								<option value={w.slug}>{w.name}</option>
							{/each}
						</select>
					{/if}
				</div>
				{#if !isPlaceholder && workspaceMounted}
					<input class="kb-filter" placeholder="Filter…" bind:value={filter} />
					<div class="kb-tree-actions">
						<button class="kb-newpage" onclick={createPage}><Plus size={14} strokeWidth={2.2} /> New page</button>
						{#if cloudActivated}
							<button class="kb-sync" onclick={doSync} disabled={syncing} title="Push this workspace's knowledge to the shared cloud copy so the team can view it">
								{#if syncing}<Loader2 size={13} class="kb-spin" />{:else}<CloudUpload size={13} strokeWidth={2.1} />{/if}
								Sync to team
							</button>
						{/if}
					</div>
					{#if syncMsg}<div class="kb-sync-msg">{syncMsg}</div>{/if}
					{#if !cloudActivated}
						<div class="kb-cloud-gate">
							<div class="kb-cloud-gate-icon"><CloudUpload size={16} strokeWidth={2} /></div>
							<div class="kb-cloud-gate-title">Activate cloud sync</div>
							<p class="kb-cloud-gate-body">Your local knowledge is always on and free. Cloud sync is opt-in: it syncs this workspace's knowledge to the cloud so your team and your other devices can access it.</p>
							<button class="kb-cloud-gate-btn" onclick={doActivate} disabled={activating}>
								{#if activating}<Loader2 size={13} class="kb-spin" />{:else}<CloudUpload size={13} strokeWidth={2.1} />{/if}
								Activate cloud sync
							</button>
							{#if activateErr}<div class="kb-cloud-gate-err">{activateErr}</div>{/if}
						</div>
					{:else}
						{#if overLimit}<div class="kb-cloud-overlimit">You're over your free cloud storage - upgrade coming soon.</div>{/if}
						{#if storage}<div class="kb-cloud-usage">{formatBytes(storage.bytes_used)} of {formatBytes(storage.bytes_limit)} used</div>{/if}
					{/if}
					{#if sourceCounts.engine + sourceCounts.cloud + sourceCounts.synced > 0}
						<div class="kb-srcfilter">
							<button class="kb-srcchip {sourceFilter === 'all' ? 'on' : ''}" onclick={() => (sourceFilter = 'all')}>All</button>
							<button class="kb-srcchip kb-srcchip--engine {sourceFilter === 'engine' ? 'on' : ''}" onclick={() => (sourceFilter = 'engine')} title="In your engine only, not synced">Engine {sourceCounts.engine}</button>
							<button class="kb-srcchip kb-srcchip--synced {sourceFilter === 'synced' ? 'on' : ''}" onclick={() => (sourceFilter = 'synced')} title="In your engine and the cloud">Synced {sourceCounts.synced}</button>
							<button class="kb-srcchip kb-srcchip--cloud {sourceFilter === 'cloud' ? 'on' : ''}" onclick={() => (sourceFilter = 'cloud')} title="In the shared cloud only">Cloud {sourceCounts.cloud}</button>
						</div>
					{/if}
				{/if}
			</div>
			<div class="kb-tree-body">
				{#if isPlaceholder}
					<div class="kb-layerhint">{PLACEHOLDERS[tab].body}</div>
				{:else if loadingTree}
					<div class="kb-muted kb-center"><Loader2 size={15} class="kb-spin" /> Loading…</div>
				{:else if !workspaceMounted}
					<div class="kb-mount-state">
						<BookOpen size={17} strokeWidth={1.8} />
						<div>
							<div class="kb-mount-title">Knowledge is not mounted here yet.</div>
							<p>{selectedWorkspaceLabel} exists as a BusinessOS workspace, but no local Optimal Engine knowledge folder is attached for this slug.</p>
						</div>
					</div>
				{:else if treeError}
					<div class="kb-muted">Couldn’t load ({treeError}).</div>
				{:else if activeTree.length === 0}
					<div class="kb-muted">Nothing in this layer yet.</div>
				{:else}
					{#each activeTree as node}
						{#if matches(node, filter.toLowerCase()) && sourceOk(node)}
							{@render treeItem(node, 0)}
						{/if}
					{/each}
				{/if}
			</div>
		</aside>

		<!-- Center -->
		<main class="kb-doc">
			{#if tab === 'sources'}
				{@render sourcesView()}
			{:else if tab === 'packages'}
				{@render packagesView()}
			{:else if tab === 'agent'}
				{@render agentView()}
			{:else if !workspaceMounted}
				<div class="kb-empty">
					<BookOpen size={30} strokeWidth={1.4} />
					<h2 class="kb-ph-title">No Knowledge mount for {selectedWorkspaceLabel}</h2>
					<p class="kb-ph-body">Create or sync a local Optimal Engine folder named <code>{slug}</code> to make this workspace readable in the Knowledge module.</p>
				</div>
			{:else if !selectedPath}
				<div class="kb-empty">
					<FileText size={26} strokeWidth={1.5} />
					<p>Select a page to read it.</p>
				</div>
			{:else}
				<div class="kb-doc-inner">
					<div class="kb-doc-head">
						<div class="kb-doc-headrow">
							<nav class="kb-crumbs">
								<span>{currentName}</span>
								{#each crumbs as c}<span class="kb-crumb-sep">/</span><span>{c}</span>{/each}
							</nav>
							<div class="kb-doc-actions">
								{#if editing}
									<button class="kb-btn" onclick={cancelEdit} disabled={saving}><X size={14} /> Cancel</button>
									<button class="kb-btn kb-btn--primary" onclick={save} disabled={saving}>
										{#if saving}<Loader2 size={14} class="kb-spin" />{:else}<Check size={14} />{/if} Save
									</button>
								{:else}
									<button class="kb-btn" onclick={startEdit}><Pencil size={14} /> Edit</button>
								{/if}
							</div>
						</div>
						<div class="kb-pageicon"><FileText size={42} strokeWidth={1.25} /></div>
						<h1 class="kb-doc-title">{properties.title ?? selectedTitle}</h1>
						{#if saveError}<p class="kb-save-error">Couldn’t save: {saveError}</p>{/if}
					</div>
					{#if loadingDoc}
						<div class="kb-muted kb-center"><Loader2 size={15} class="kb-spin" /> Loading…</div>
					{:else if editing}
						<textarea class="kb-editor" bind:value={draft} spellcheck="false"></textarea>
					{:else}
						<article class="kb-prose">{@html bodyHtml}</article>
					{/if}
				</div>
			{/if}
		</main>

		<!-- Right rail -->
		{#if selectedPath && !loadingDoc && !isPlaceholder && railOpen}
			<aside class="kb-rail">
				{#if shownProps.length > 0}
					<div class="kb-rail-sec">
						<div class="kb-rail-label">Properties</div>
						{#each shownProps as [k, v]}
							<div class="kb-prop"><span class="kb-prop-k">{k}</span><span class="kb-prop-v">{v}</span></div>
						{/each}
					</div>
				{/if}
				{#if toc.length > 0}
					<div class="kb-rail-sec">
						<div class="kb-rail-label">On this page</div>
						{#each toc as h}
							<a class="kb-toc kb-toc--l{h.level}" href={`#${h.id}`}><Hash size={11} class="kb-toc-icon" />{h.text}</a>
						{/each}
					</div>
				{/if}
				<div class="kb-rail-sec">
					<div class="kb-rail-label">Actions</div>
					<button class="kb-action" disabled><Sparkles size={14} /> Ask agent <span class="kb-soon">soon</span></button>
					<button class="kb-action" disabled><Download size={14} /> Export PDF <span class="kb-soon">soon</span></button>
				</div>
				{#if modified}
					<div class="kb-rail-meta"><Clock size={12} /> Updated {fmtDate(modified)}</div>
				{/if}
			</aside>
		{/if}
	</div>
</div>

{#snippet treeItem(node: KBTreeNode, depth: number)}
	{#if node.type === 'dir'}
		<button
			class="kb-row kb-dir {selectedPath && node.indexPath === selectedPath ? 'kb-file--active' : ''}"
			style="padding-left: {8 + depth * 13}px"
			onclick={() => onDirClick(node)}
		>
			<ChevronRight size={13} class="kb-chev {expanded.has(node.path) ? 'kb-chev--open' : ''}" />
			<Folder size={14} class="kb-file-icon" />
			<span class="kb-row-label">{node.title ?? node.name}</span>
		</button>
		{#if expanded.has(node.path) && node.children}
			{#each node.children as child}
				{#if matches(child, filter.toLowerCase())}
					{@render treeItem(child, depth + 1)}
				{/if}
			{/each}
		{/if}
	{:else}
		<button
			class="kb-row kb-file {selectedPath === node.path ? 'kb-file--active' : ''}"
			style="padding-left: {8 + depth * 13 + 17}px"
			onclick={() => openDoc(node.path, node.title ?? node.name)}
		>
			<FileText size={14} class="kb-file-icon" />
			<span class="kb-row-label">{node.title ?? node.name}</span>
			{#if sources[node.path]}
				<span class="kb-src kb-src--{sources[node.path]}" title="{sources[node.path] === 'engine' ? 'In your engine only - not yet synced' : sources[node.path] === 'cloud' ? 'In the shared cloud only' : 'Synced: in your engine and the cloud'}">{sources[node.path]}</span>
			{/if}
		</button>
	{/if}
{/snippet}

<!-- ── Sources: canonical + connected sources the workspace's knowledge rests on ── -->
{#snippet sourcesView()}
	<div class="kb-doc-inner">
		<div class="kb-doc-head">
			<nav class="kb-crumbs"><span>{currentName || selectedWorkspaceLabel}</span><span class="kb-crumb-sep">/</span><span>Sources</span></nav>
			<div class="kb-pageicon"><Link2 size={42} strokeWidth={1.25} /></div>
			<h1 class="kb-doc-title">Sources</h1>
			<p class="kb-head-sub">Meetings, uploads, transcripts, and citations the agent treats as evidence for this workspace.</p>
		</div>
		<div class="kb-panel">
			{#if totalSources > 0}
				<div class="kb-statrow">
					<div class="kb-stat"><span class="kb-stat-n">{totalDocs}</span><span class="kb-stat-l">Documents</span></div>
					<div class="kb-stat"><span class="kb-stat-n kb-stat-n--engine">{sourceCounts.engine}</span><span class="kb-stat-l">Engine only</span></div>
					<div class="kb-stat"><span class="kb-stat-n kb-stat-n--synced">{sourceCounts.synced}</span><span class="kb-stat-l">Synced</span></div>
					<div class="kb-stat"><span class="kb-stat-n kb-stat-n--cloud">{sourceCounts.cloud}</span><span class="kb-stat-l">Cloud only</span></div>
				</div>
			{/if}

			<div class="kb-sec-label">Indexed sources{#if engineItems.length > 0} <span class="kb-count">{engineItems.length}</span>{/if}</div>
			{#if loadingEngine}
				<div class="kb-muted kb-center"><Loader2 size={15} class="kb-spin" /> Loading sources…</div>
			{:else if engineError}
				<div class="kb-muted">Couldn’t load sources ({engineError}).</div>
			{:else if !engineLinked}
				<div class="kb-mount-state">
					<Link2 size={17} strokeWidth={1.8} />
					<div>
						<div class="kb-mount-title">No engine linked for this workspace.</div>
						<p>Sources appear once {selectedWorkspaceLabel} is connected to a local Optimal Engine. Its documents and citations then become searchable evidence for the agent.</p>
					</div>
				</div>
			{:else if engineItems.length === 0}
				<div class="kb-mount-state">
					<Link2 size={17} strokeWidth={1.8} />
					<div>
						<div class="kb-mount-title">No sources ingested yet.</div>
						<p>Connect Fathom for meeting transcripts, upload documents, or write pages in the Docs tab. Once ingested into the engine they surface here as citable sources.</p>
					</div>
				</div>
			{:else}
				<div class="kb-cards">
					{#each engineItems as item}
						<div class="kb-card">
							<div class="kb-card-top">
								<FileText size={15} class="kb-file-icon" />
								<span class="kb-card-title">{item.title || 'Untitled source'}</span>
								{#if item.canonical}<span class="kb-tag kb-tag--good">canonical</span>{/if}
								{#if item.is_external}<span class="kb-tag">external</span>{/if}
							</div>
							{#if item.abstract}<p class="kb-card-abs">{item.abstract}</p>{/if}
							<div class="kb-card-meta">
								{#if item.source}<span class="kb-meta-pill"><Link2 size={11} /> {item.source}</span>{/if}
								{#if item.target_module}<span class="kb-meta-pill">{item.target_module}</span>{/if}
								{#if item.status}<span class="kb-tag">{item.status}</span>{/if}
								{#if item.updated_at}<span class="kb-meta-date"><Clock size={11} /> {fmtDate(item.updated_at)}</span>{/if}
							</div>
						</div>
					{/each}
				</div>
			{/if}
		</div>
	</div>
{/snippet}

<!-- ── Packages: generated deliverables/exports produced from this workspace ── -->
{#snippet packagesView()}
	<div class="kb-doc-inner">
		<div class="kb-doc-head">
			<nav class="kb-crumbs"><span>{currentName || selectedWorkspaceLabel}</span><span class="kb-crumb-sep">/</span><span>Packages</span></nav>
			<div class="kb-pageicon"><Package size={42} strokeWidth={1.25} /></div>
			<h1 class="kb-doc-title">Packages</h1>
			<p class="kb-head-sub">Deliverables and packaged exports produced from this workspace’s knowledge.</p>
		</div>
		<div class="kb-panel">
			{#if loadingDeliverables}
				<div class="kb-muted kb-center"><Loader2 size={15} class="kb-spin" /> Loading packages…</div>
			{:else if deliverablesError}
				<div class="kb-muted">Couldn’t load packages ({deliverablesError}).</div>
			{:else if deliverables.length === 0}
				<div class="kb-mount-state">
					<Package size={17} strokeWidth={1.8} />
					<div>
						<div class="kb-mount-title">No packages yet.</div>
						<p>Generated deliverables — proposals, briefs, decks, scripts, reports — appear here once produced in the Deliverables flow. Each links back to the artifact it shipped.</p>
					</div>
				</div>
			{:else}
				<div class="kb-sec-label">Deliverables <span class="kb-count">{deliverables.length}</span></div>
				<div class="kb-cards">
					{#each deliverables as d}
						<div class="kb-card">
							<div class="kb-card-top">
								<Package size={15} class="kb-file-icon" />
								<span class="kb-card-title">{d.title || 'Untitled package'}</span>
								<span class="kb-tag">{d.kind}</span>
								<span class="kb-tag kb-tag--{d.status === 'delivered' ? 'good' : d.status === 'in_progress' ? 'warn' : ''}">{d.status.replace('_', ' ')}</span>
							</div>
							{#if d.description}<p class="kb-card-abs">{d.description}</p>{/if}
							<div class="kb-card-meta">
								{#if d.client}<span class="kb-meta-pill">{d.client}</span>{/if}
								{#if d.project}<span class="kb-meta-pill">{d.project}</span>{/if}
								{#if d.updated_at}<span class="kb-meta-date"><Clock size={11} /> {fmtDate(d.updated_at)}</span>{/if}
								{#if d.link}<a class="kb-card-link" href={d.link} target="_blank" rel="noopener noreferrer"><ExternalLink size={12} /> Open</a>{/if}
							</div>
						</div>
					{/each}
				</div>
			{/if}
		</div>
	</div>
{/snippet}

<!-- ── Agent Context: what the agent may read + trust for this workspace ── -->
{#snippet agentView()}
	<div class="kb-doc-inner">
		<div class="kb-doc-head">
			<nav class="kb-crumbs"><span>{currentName || selectedWorkspaceLabel}</span><span class="kb-crumb-sep">/</span><span>Agent Context</span></nav>
			<div class="kb-pageicon"><Bot size={42} strokeWidth={1.25} /></div>
			<h1 class="kb-doc-title">Agent Context</h1>
			<p class="kb-head-sub">The exact knowledge scope the AI agent is allowed to read and trust when working on this workspace.</p>
		</div>
		<div class="kb-panel">
			<div class="kb-scope">
				<div class="kb-scope-k">Mounted workspace</div>
				<div class="kb-scope-v">
					<span class="kb-scope-name">{selectedWorkspaceLabel}</span>
					<code class="kb-scope-slug">{slug}</code>
					{#if workspaceMounted}
						<span class="kb-tag kb-tag--good">engine mounted</span>
					{:else}
						<span class="kb-tag kb-tag--warn">not mounted</span>
					{/if}
				</div>
			</div>

			<div class="kb-statrow">
				<div class="kb-stat"><span class="kb-stat-n">{totalDocs}</span><span class="kb-stat-l">Readable docs</span></div>
				<div class="kb-stat"><span class="kb-stat-n">{topFolders.length}</span><span class="kb-stat-l">Folders</span></div>
				<div class="kb-stat"><span class="kb-stat-n">{engineItems.length}</span><span class="kb-stat-l">Indexed sources</span></div>
				<div class="kb-stat"><span class="kb-stat-n">{deliverables.length}</span><span class="kb-stat-l">Packages</span></div>
			</div>

			{#if topFolders.length > 0}
				<div class="kb-sec-label">Knowledge folders in scope</div>
				<div class="kb-chiprow">
					{#each topFolders as f}
						<span class="kb-folderchip"><Folder size={12} /> {f.title ?? f.name}</span>
					{/each}
				</div>
			{/if}

			<div class="kb-sec-label">Trust</div>
			<div class="kb-trustrow">
				<span class="kb-trust"><ShieldCheck size={13} /> {trustedDocs} trusted (engine + synced)</span>
				<span class="kb-trust kb-trust--muted">{sourceCounts.cloud} cloud-only</span>
			</div>

			<div class="kb-accessgrid">
				<div class="kb-access kb-access--yes">
					<div class="kb-access-h"><Check size={14} /> Can access</div>
					<ul>
						<li>Every canonical document in <strong>{selectedWorkspaceLabel}</strong>’s knowledge base ({totalDocs} pages across {topFolders.length} folders).</li>
						<li>{engineItems.length} engine-indexed sources — meeting transcripts, uploads, and citations.</li>
						<li>{deliverables.length} packaged deliverables produced from this workspace.</li>
						<li>Documents marked <strong>engine</strong> or <strong>synced</strong> are treated as trusted evidence.</li>
					</ul>
				</div>
				<div class="kb-access kb-access--no">
					<div class="kb-access-h"><X size={14} /> Cannot access</div>
					<ul>
						<li>Any other workspace’s knowledge — scope is locked to <code>{slug}</code>.</li>
						<li>External systems you haven’t connected (Drive, Gmail, CRM) unless explicitly wired in.</li>
						{#if !workspaceMounted}<li>Local engine documents — no Optimal Engine is mounted for this slug yet.</li>{/if}
						<li>Unsynced private notes living outside the knowledge base.</li>
					</ul>
				</div>
			</div>
		</div>
	</div>
{/snippet}

<style>
	.kb-shell { display: flex; flex-direction: column; height: 100%; overflow: hidden; }

	/* Tabs */
	.kb-tabs {
		display: flex;
		gap: 2px;
		padding: 8px 12px 0;
		border-bottom: 1px solid var(--dbd);
		background: var(--dbg);
		overflow-x: auto;
		scrollbar-width: none;
	}
	.kb-tabs::-webkit-scrollbar { display: none; }
	.kb-tab {
		display: flex; align-items: center; gap: 7px;
		padding: 9px 13px; border: none; background: none; cursor: pointer;
		color: var(--dt3); font-size: 0.84rem; font-weight: 500;
		border-bottom: 2px solid transparent; white-space: nowrap;
		transition: color 140ms ease, border-color 140ms ease;
	}
	.kb-tab:hover { color: var(--dt2); }
	.kb-tab--active { color: var(--dt); border-bottom-color: #6366f1; }
	.kb-tabs-spacer { flex: 1; }
	.kb-paneltoggle {
		display: flex; align-items: center; justify-content: center; flex-shrink: 0;
		width: 30px; height: 30px; margin: 0 2px 6px; border-radius: 7px;
		border: none; background: none; color: var(--dt3); cursor: pointer;
		transition: color 140ms ease, background 140ms ease;
	}
	.kb-paneltoggle:hover { color: var(--dt); background: color-mix(in srgb, var(--dt) 7%, transparent); }
	.kb-paneltoggle--on { color: var(--dt2); }

	.kb { display: flex; flex: 1; min-height: 0; }

	/* Tree (Notion two-tone: sidebar slightly off, main content clean) */
	.kb-tree {
		width: 280px; flex-shrink: 0; border-right: 1px solid var(--dbd);
		display: flex; flex-direction: column; min-height: 0; background: var(--dbg2);
		transition: width 180ms ease;
	}
	.kb-tree--closed { width: 0; overflow: hidden; border-right: none; }
	.kb-tree-head { padding: 16px 14px 10px; border-bottom: 1px solid var(--dbd); }
	.kb-tree-titlerow { display: flex; align-items: center; justify-content: space-between; gap: 8px; }
	.kb-tree-title { font-size: 0.95rem; font-weight: 650; color: var(--dt); letter-spacing: -0.01em; }
	.kb-ws-select {
		max-width: 150px; padding: 4px 6px; font-size: 0.74rem; color: var(--dt2);
		background: var(--dbg2); border: 1px solid var(--dbd); border-radius: 7px; cursor: pointer; outline: none;
	}
	.kb-filter {
		margin-top: 10px; width: 100%; padding: 6px 9px; font-size: 0.8rem; color: var(--dt);
		background: var(--dbg2); border: 1px solid var(--dbd); border-radius: 8px; outline: none;
	}
	.kb-filter::placeholder { color: var(--dt3); }
	.kb-filter:focus { border-color: color-mix(in srgb, var(--dt) 25%, transparent); }
	.kb-newpage {
		display: flex; align-items: center; justify-content: center; gap: 6px;
		margin-top: 8px; width: 100%; padding: 6px 9px; font-size: 0.78rem; font-weight: 500;
		color: var(--dt2); background: transparent; border: 1px dashed color-mix(in srgb, var(--dt) 20%, transparent);
		border-radius: 8px; cursor: pointer; transition: background 140ms ease, color 140ms ease, border-color 140ms ease;
	}
	.kb-newpage:hover { color: var(--dt); border-color: color-mix(in srgb, var(--dt) 32%, transparent); background: color-mix(in srgb, var(--dt) 4%, transparent); }
	.kb-tree-actions { display: flex; gap: 6px; margin-top: 8px; }
	.kb-tree-actions .kb-newpage { margin-top: 0; flex: 1; }
	.kb-sync {
		display: flex; align-items: center; justify-content: center; gap: 6px; white-space: nowrap;
		padding: 6px 9px; font-size: 0.78rem; font-weight: 500;
		color: var(--dt2); background: transparent; border: 1px solid color-mix(in srgb, var(--dt) 20%, transparent);
		border-radius: 8px; cursor: pointer; transition: background 140ms ease, color 140ms ease, border-color 140ms ease;
	}
	.kb-sync:hover:not(:disabled) { color: var(--dt); border-color: color-mix(in srgb, var(--dt) 32%, transparent); background: color-mix(in srgb, var(--dt) 4%, transparent); }
	.kb-sync:disabled { opacity: 0.55; cursor: default; }
	.kb-sync-msg { margin-top: 6px; font-size: 0.72rem; color: var(--dt2); }

	/* Cloud sync opt-in gate: local is always on + free, cloud is opt-in. */
	.kb-cloud-gate {
		margin-top: 10px; padding: 12px; border-radius: 10px;
		background: var(--dbg2); border: 1px solid var(--dbd);
		display: flex; flex-direction: column; gap: 7px;
	}
	.kb-cloud-gate-icon {
		display: inline-flex; align-items: center; justify-content: center;
		width: 30px; height: 30px; border-radius: 8px;
		color: var(--dt); background: color-mix(in srgb, var(--dt) 8%, transparent);
	}
	.kb-cloud-gate-title { font-size: 0.82rem; font-weight: 600; color: var(--dt); }
	.kb-cloud-gate-body { margin: 0; font-size: 0.73rem; line-height: 1.45; color: var(--dt2); }
	.kb-cloud-gate-btn {
		display: flex; align-items: center; justify-content: center; gap: 6px;
		margin-top: 2px; padding: 7px 10px; font-size: 0.78rem; font-weight: 500;
		color: var(--bos-v2-button-pureWhiteText); background: var(--bos-v2-button-primary);
		border: 1px solid transparent; border-radius: 8px; cursor: pointer;
		transition: filter 140ms ease;
	}
	.kb-cloud-gate-btn:hover:not(:disabled) { filter: brightness(1.08); }
	.kb-cloud-gate-btn:disabled { opacity: 0.6; cursor: default; }
	.kb-cloud-gate-err { font-size: 0.72rem; color: #ef4444; }
	.kb-cloud-usage { margin-top: 8px; font-size: 0.72rem; color: var(--dt3, var(--dt2)); }
	.kb-cloud-overlimit {
		margin-top: 8px; padding: 7px 9px; border-radius: 8px; font-size: 0.72rem; line-height: 1.4;
		color: #b45309; background: color-mix(in srgb, #f59e0b 12%, transparent);
		border: 1px solid color-mix(in srgb, #f59e0b 32%, transparent);
	}
	.kb-srcfilter { display: flex; flex-wrap: wrap; gap: 5px; margin-top: 8px; }
	.kb-srcchip { font-size: 0.68rem; font-weight: 540; padding: 3px 8px; border-radius: 999px; border: 1px solid var(--dbd); background: transparent; color: var(--dt3); cursor: pointer; white-space: nowrap; }
	.kb-srcchip.on { color: var(--dt); border-color: color-mix(in srgb, var(--dt) 34%, transparent); background: color-mix(in srgb, var(--dt) 6%, transparent); }
	.kb-srcchip--engine.on { color: #f59e0b; border-color: color-mix(in srgb, #f59e0b 45%, transparent); background: color-mix(in srgb, #f59e0b 12%, transparent); }
	.kb-srcchip--synced.on { color: #22c55e; border-color: color-mix(in srgb, #22c55e 45%, transparent); background: color-mix(in srgb, #22c55e 12%, transparent); }
	.kb-srcchip--cloud.on { color: #3b82f6; border-color: color-mix(in srgb, #3b82f6 45%, transparent); background: color-mix(in srgb, #3b82f6 12%, transparent); }
	.kb-src { margin-left: auto; font-size: 0.6rem; font-weight: 640; text-transform: uppercase; letter-spacing: 0.03em; padding: 1px 5px; border-radius: 5px; flex-shrink: 0; }
	.kb-src--engine { color: #f59e0b; background: color-mix(in srgb, #f59e0b 15%, transparent); }
	.kb-src--synced { color: #22c55e; background: color-mix(in srgb, #22c55e 15%, transparent); }
	.kb-src--cloud { color: #3b82f6; background: color-mix(in srgb, #3b82f6 15%, transparent); }
	.kb-tree-body { flex: 1; overflow-y: auto; padding: 8px 8px 24px; }
	.kb-row {
		display: flex; align-items: center; gap: 6px; width: 100%; padding: 5px 8px;
		border: none; background: none; border-radius: 7px; cursor: pointer; text-align: left;
		color: var(--dt2); font-size: 0.82rem; transition: background 120ms ease, color 120ms ease;
	}
	.kb-row:hover { background: color-mix(in srgb, var(--dt) 6%, transparent); color: var(--dt); }
	.kb-dir { font-weight: 550; color: var(--dt); }
	.kb-file--active { background: color-mix(in srgb, var(--dt) 9%, transparent); color: var(--dt); font-weight: 550; }
	.kb-row-label { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
	:global(.kb-chev) { flex-shrink: 0; color: var(--dt3); transition: transform 160ms ease; }
	:global(.kb-chev--open) { transform: rotate(90deg); }
	:global(.kb-file-icon) { flex-shrink: 0; color: var(--dt3); }

	/* Document */
	.kb-doc { flex: 1; min-width: 0; overflow-y: auto; background: var(--dbg); }
	.kb-doc-inner { max-width: 720px; margin: 0 auto; }
	.kb-empty {
		height: 100%; display: flex; flex-direction: column; align-items: center; justify-content: center;
		gap: 10px; color: var(--dt3); font-size: 0.9rem; text-align: center; padding: 0 40px;
	}
	.kb-ph-title { font-size: 1.1rem; font-weight: 640; color: var(--dt); margin: 4px 0 0; }
	.kb-ph-body { max-width: 380px; line-height: 1.6; color: var(--dt3); margin: 0; }
	.kb-ph-body code {
		font-family: ui-monospace, SFMono-Regular, monospace;
		font-size: 0.82em;
		color: var(--dt);
		background: color-mix(in srgb, var(--dt) 8%, transparent);
		border-radius: 5px;
		padding: 1px 5px;
	}
	.kb-doc-head { padding: 28px 56px 0; }
	.kb-doc-headrow { display: flex; align-items: center; justify-content: space-between; gap: 12px; min-height: 28px; }
	.kb-crumbs { display: flex; align-items: center; gap: 6px; font-size: 0.78rem; color: var(--dt3); overflow: hidden; }
	.kb-crumbs span { white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
	.kb-crumb-sep { opacity: 0.5; }
	.kb-doc-actions { display: flex; gap: 6px; flex-shrink: 0; }
	.kb-pageicon { margin: 28px 0 14px; color: var(--dt3); }
	.kb-btn {
		display: flex; align-items: center; gap: 6px; padding: 5px 10px; font-size: 0.78rem; font-weight: 500;
		color: var(--dt2); background: var(--dbg2); border: 1px solid var(--dbd); border-radius: 8px; cursor: pointer;
		transition: background 140ms ease, color 140ms ease;
	}
	.kb-btn:hover:not(:disabled) { color: var(--dt); background: color-mix(in srgb, var(--dt) 8%, transparent); }
	.kb-btn:disabled { opacity: 0.6; cursor: default; }
	.kb-btn--primary { color: #fff; background: #6366f1; border-color: #6366f1; }
	.kb-btn--primary:hover:not(:disabled) { background: #5457e5; color: #fff; }
	.kb-doc-title { font-size: 2.5rem; font-weight: 700; letter-spacing: -0.03em; color: var(--dt); margin: 0; line-height: 1.1; }
	.kb-save-error { color: #ef4444; font-size: 0.78rem; margin: 8px 0 0; }
	.kb-editor {
		width: calc(100% - 80px); margin: 18px 40px 80px; min-height: 60vh; resize: vertical;
		font-family: ui-monospace, SFMono-Regular, monospace; font-size: 0.86rem; line-height: 1.65;
		color: var(--dt); background: var(--dbg2); border: 1px solid var(--dbd); border-radius: 10px;
		padding: 16px 18px; outline: none;
	}
	.kb-editor:focus { border-color: color-mix(in srgb, var(--dt) 25%, transparent); }
	.kb-prose { padding: 22px 56px 90px; color: var(--dt2); font-size: 0.95rem; line-height: 1.75; }
	.kb-prose :global(h1), .kb-prose :global(h2), .kb-prose :global(h3) {
		color: var(--dt); font-weight: 650; letter-spacing: -0.01em; margin: 1.6em 0 0.5em; line-height: 1.3; scroll-margin-top: 20px;
	}
	.kb-prose :global(h1) { font-size: 1.45rem; }
	.kb-prose :global(h2) { font-size: 1.2rem; }
	.kb-prose :global(h3) { font-size: 1.02rem; }
	.kb-prose :global(p) { margin: 0.7em 0; }
	.kb-prose :global(a) { color: #6366f1; text-decoration: none; }
	.kb-prose :global(a:hover) { text-decoration: underline; }
	.kb-prose :global(ul), .kb-prose :global(ol) { margin: 0.7em 0; padding-left: 1.4em; }
	.kb-prose :global(li) { margin: 0.3em 0; }
	.kb-prose :global(code) {
		font-family: ui-monospace, monospace; font-size: 0.85em; padding: 1px 5px; border-radius: 5px;
		background: color-mix(in srgb, var(--dt) 8%, transparent); color: var(--dt);
	}
	.kb-prose :global(pre) { background: var(--dbg2); border: 1px solid var(--dbd); border-radius: 10px; padding: 14px 16px; overflow-x: auto; margin: 1em 0; }
	.kb-prose :global(pre code) { background: none; padding: 0; }
	.kb-prose :global(blockquote) { border-left: 3px solid var(--dbd); margin: 1em 0; padding: 0.2em 0 0.2em 1em; color: var(--dt3); }
	.kb-prose :global(table) { border-collapse: collapse; width: 100%; margin: 1em 0; font-size: 0.88rem; }
	.kb-prose :global(th), .kb-prose :global(td) { border: 1px solid var(--dbd); padding: 7px 11px; text-align: left; }
	.kb-prose :global(th) { background: color-mix(in srgb, var(--dt) 5%, transparent); color: var(--dt); font-weight: 600; }
	.kb-prose :global(hr) { border: none; border-top: 1px solid var(--dbd); margin: 1.6em 0; }

	.kb-btn--icon { padding: 5px 7px; }

	/* Right rail */
	.kb-rail { width: 240px; flex-shrink: 0; border-left: 1px solid var(--dbd); background: var(--dbg); overflow-y: auto; padding: 28px 18px; display: flex; flex-direction: column; gap: 22px; }
	.kb-rail-sec { display: flex; flex-direction: column; gap: 6px; }
	.kb-rail-label { font-size: 0.64rem; font-weight: 600; text-transform: uppercase; letter-spacing: 0.09em; color: var(--dt3); margin-bottom: 2px; }
	.kb-prop { display: flex; justify-content: space-between; gap: 10px; font-size: 0.78rem; padding: 2px 0; }
	.kb-prop-k { color: var(--dt3); text-transform: capitalize; }
	.kb-prop-v { color: var(--dt); text-align: right; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
	.kb-toc { display: flex; align-items: center; gap: 5px; font-size: 0.78rem; color: var(--dt2); text-decoration: none; padding: 2px 0; line-height: 1.35; }
	.kb-toc:hover { color: var(--dt); }
	.kb-toc--l2 { padding-left: 12px; }
	.kb-toc--l3 { padding-left: 24px; color: var(--dt3); }
	:global(.kb-toc-icon) { flex-shrink: 0; color: var(--dt3); }
	.kb-action { display: flex; align-items: center; gap: 7px; width: 100%; padding: 7px 9px; border-radius: 8px; border: 1px solid var(--dbd); background: var(--dbg2); color: var(--dt2); font-size: 0.8rem; cursor: not-allowed; }
	.kb-soon { margin-left: auto; font-size: 0.6rem; text-transform: uppercase; color: var(--dt3); }
	.kb-rail-meta { display: flex; align-items: center; gap: 6px; font-size: 0.72rem; color: var(--dt3); margin-top: auto; }

	.kb-muted { color: var(--dt3); font-size: 0.84rem; padding: 8px; }
	.kb-center { display: flex; align-items: center; gap: 8px; }
	.kb-mount-state {
		display: grid;
		grid-template-columns: 18px minmax(0, 1fr);
		gap: 9px;
		margin: 8px 4px;
		padding: 10px;
		color: var(--dt3);
		background: color-mix(in srgb, var(--dt) 4%, transparent);
		border: 1px solid color-mix(in srgb, var(--dt) 8%, transparent);
		border-radius: 9px;
	}
	.kb-mount-title {
		margin-bottom: 4px;
		color: var(--dt);
		font-size: 0.8rem;
		font-weight: 620;
		line-height: 1.25;
	}
	.kb-mount-state p {
		margin: 0;
		font-size: 0.76rem;
		line-height: 1.45;
	}
	:global(.kb-spin) { animation: kb-spin 1s linear infinite; }
	@keyframes kb-spin { to { transform: rotate(360deg); } }

	@media (max-width: 1100px) {
		.kb-rail { display: none; }
	}

	/* ── Mobile (<=768px): tree becomes an off-canvas overlay drawer ── */
	@media (max-width: 768px) {
		/* Hide the desktop panel toggle, show the mobile one */
		.kb-paneltoggle--desktop { display: none; }
		.kb-paneltoggle--mobile { display: flex; }

		/* Doc head padding reduced */
		.kb-doc-head { padding: 16px 18px 0; }
		.kb-doc-inner { max-width: 100%; }

		/* Prose and editor fill the viewport */
		.kb-prose { padding: 16px 18px 60px; }
		.kb-editor {
			width: calc(100% - 36px);
			margin: 12px 18px 60px;
		}

		/* Tree: hidden off-screen by default on mobile */
		.kb-tree--mobile {
			position: fixed;
			top: 0;
			left: 0;
			bottom: 0;
			z-index: 200;
			width: 280px;
			transform: translateX(-100%);
			transition: transform 240ms cubic-bezier(0.4, 0, 0.2, 1);
			border-right: 1px solid var(--dbd);
			box-shadow: none;
		}
		/* Override the desktop closed state on mobile — position handles visibility */
		.kb-tree--mobile.kb-tree--closed {
			width: 280px;
			overflow: visible;
			border-right: 1px solid var(--dbd);
		}
		/* Slide in when open */
		.kb-tree--mobile.kb-tree--mobile-open {
			transform: translateX(0);
			box-shadow: 4px 0 24px rgba(0, 0, 0, 0.18);
		}

		/* Backdrop behind the drawer */
		.kb-mobile-backdrop {
			position: fixed;
			inset: 0;
			z-index: 199;
			background: rgba(0, 0, 0, 0.4);
		}

		/* Title in doc head: allow wrapping */
		.kb-doc-title { font-size: 1.8rem; }

		/* Page icon smaller on mobile */
		.kb-pageicon { margin: 16px 0 10px; }

		/* Actions row: allow wrap */
		.kb-doc-headrow { flex-wrap: wrap; gap: 8px; }

		/* Tabs: tighten spacing */
		.kb-tabs { padding: 6px 8px 0; gap: 0; }
		.kb-tab { padding: 8px 9px; gap: 5px; font-size: 0.78rem; }
	}

	/* ── Very small (<=480px): further reduce chrome ── */
	@media (max-width: 480px) {
		.kb-tab span { display: none; }
		.kb-tab { padding: 8px 10px; }
		.kb-doc-head { padding: 12px 14px 0; }
		.kb-prose { padding: 12px 14px 48px; }
		.kb-editor {
			width: calc(100% - 28px);
			margin: 10px 14px 48px;
		}
		.kb-doc-title { font-size: 1.5rem; }
	}

	/* Hide mobile toggle on desktop */
	.kb-paneltoggle--mobile { display: none; }

	/* ── Real layer views (Sources / Packages / Agent Context) ── */
	.kb-layerhint { color: var(--dt3); font-size: 0.8rem; line-height: 1.5; padding: 10px 8px; }
	.kb-head-sub { max-width: 640px; margin: 10px 0 0; color: var(--dt3); font-size: 0.9rem; line-height: 1.6; }
	.kb-panel { padding: 24px 56px 90px; max-width: 820px; }

	.kb-sec-label {
		display: flex; align-items: center; gap: 7px;
		font-size: 0.66rem; font-weight: 640; text-transform: uppercase; letter-spacing: 0.09em;
		color: var(--dt3); margin: 26px 0 12px;
	}
	.kb-count {
		font-size: 0.68rem; font-weight: 600; letter-spacing: 0; text-transform: none;
		color: var(--dt2); background: color-mix(in srgb, var(--dt) 8%, transparent);
		border-radius: 999px; padding: 1px 8px;
	}

	/* Stat row */
	.kb-statrow { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 10px; }
	.kb-stat {
		display: flex; flex-direction: column; gap: 3px; padding: 14px 16px;
		background: var(--dbg2); border: 1px solid var(--dbd); border-radius: 12px;
	}
	.kb-stat-n { font-size: 1.5rem; font-weight: 700; letter-spacing: -0.02em; color: var(--dt); line-height: 1; }
	.kb-stat-n--engine { color: #f59e0b; }
	.kb-stat-n--synced { color: #22c55e; }
	.kb-stat-n--cloud { color: #3b82f6; }
	.kb-stat-l { font-size: 0.72rem; color: var(--dt3); }

	/* Cards (sources + packages) */
	.kb-cards { display: flex; flex-direction: column; gap: 10px; }
	.kb-card {
		display: flex; flex-direction: column; gap: 9px; padding: 14px 16px;
		background: var(--dbg2); border: 1px solid var(--dbd); border-radius: 12px;
		transition: border-color 140ms ease;
	}
	.kb-card:hover { border-color: color-mix(in srgb, var(--dt) 20%, transparent); }
	.kb-card-top { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
	.kb-card-title { font-size: 0.92rem; font-weight: 600; color: var(--dt); }
	.kb-card-abs { margin: 0; font-size: 0.84rem; line-height: 1.55; color: var(--dt2); }
	.kb-card-meta { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; font-size: 0.74rem; color: var(--dt3); }
	.kb-meta-pill {
		display: inline-flex; align-items: center; gap: 4px;
		padding: 2px 8px; border-radius: 999px; border: 1px solid var(--dbd);
		color: var(--dt2); background: transparent;
	}
	.kb-meta-date { display: inline-flex; align-items: center; gap: 4px; }
	.kb-card-link {
		display: inline-flex; align-items: center; gap: 5px; margin-left: auto;
		padding: 3px 10px; border-radius: 8px; font-size: 0.75rem; font-weight: 550;
		color: #6366f1; text-decoration: none; border: 1px solid color-mix(in srgb, #6366f1 40%, transparent);
	}
	.kb-card-link:hover { background: color-mix(in srgb, #6366f1 12%, transparent); }

	/* Small tags */
	.kb-tag {
		font-size: 0.66rem; font-weight: 620; text-transform: capitalize; letter-spacing: 0.01em;
		padding: 2px 8px; border-radius: 6px; white-space: nowrap;
		color: var(--dt2); background: color-mix(in srgb, var(--dt) 8%, transparent);
	}
	.kb-tag--good { color: #22c55e; background: color-mix(in srgb, #22c55e 14%, transparent); }
	.kb-tag--warn { color: #f59e0b; background: color-mix(in srgb, #f59e0b 14%, transparent); }

	/* Agent Context scope */
	.kb-scope {
		display: flex; flex-wrap: wrap; align-items: baseline; gap: 6px 12px;
		padding: 14px 16px; background: var(--dbg2); border: 1px solid var(--dbd); border-radius: 12px; margin-bottom: 20px;
	}
	.kb-scope-k { font-size: 0.66rem; font-weight: 640; text-transform: uppercase; letter-spacing: 0.09em; color: var(--dt3); }
	.kb-scope-v { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
	.kb-scope-name { font-size: 0.95rem; font-weight: 620; color: var(--dt); }
	.kb-scope-slug {
		font-family: ui-monospace, SFMono-Regular, monospace; font-size: 0.78rem; color: var(--dt2);
		background: color-mix(in srgb, var(--dt) 8%, transparent); border-radius: 6px; padding: 1px 7px;
	}
	.kb-chiprow { display: flex; flex-wrap: wrap; gap: 7px; }
	.kb-folderchip {
		display: inline-flex; align-items: center; gap: 5px;
		padding: 4px 10px; border-radius: 999px; border: 1px solid var(--dbd);
		font-size: 0.78rem; color: var(--dt2); background: var(--dbg2);
	}
	.kb-trustrow { display: flex; flex-wrap: wrap; gap: 10px; }
	.kb-trust { display: inline-flex; align-items: center; gap: 6px; font-size: 0.82rem; color: #22c55e; font-weight: 540; }
	.kb-trust--muted { color: var(--dt3); font-weight: 500; }

	.kb-accessgrid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 12px; margin-top: 26px; }
	.kb-access { padding: 16px 18px; border-radius: 12px; border: 1px solid var(--dbd); background: var(--dbg2); }
	.kb-access--yes { border-color: color-mix(in srgb, #22c55e 30%, var(--dbd)); }
	.kb-access--no { border-color: color-mix(in srgb, #ef4444 26%, var(--dbd)); }
	.kb-access-h { display: flex; align-items: center; gap: 7px; font-size: 0.82rem; font-weight: 640; margin-bottom: 8px; }
	.kb-access--yes .kb-access-h { color: #22c55e; }
	.kb-access--no .kb-access-h { color: #ef4444; }
	.kb-access ul { margin: 0; padding-left: 18px; display: flex; flex-direction: column; gap: 6px; }
	.kb-access li { font-size: 0.83rem; line-height: 1.5; color: var(--dt2); }
	.kb-access code {
		font-family: ui-monospace, SFMono-Regular, monospace; font-size: 0.8em; color: var(--dt);
		background: color-mix(in srgb, var(--dt) 8%, transparent); border-radius: 5px; padding: 0 4px;
	}

	@media (max-width: 768px) {
		.kb-panel { padding: 16px 18px 60px; }
		.kb-statrow { grid-template-columns: repeat(2, minmax(0, 1fr)); }
		.kb-accessgrid { grid-template-columns: 1fr; }
	}
	@media (max-width: 480px) {
		.kb-panel { padding: 12px 14px 48px; }
	}
</style>
