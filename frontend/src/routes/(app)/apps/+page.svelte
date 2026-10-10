<script lang="ts">
	import { currentWorkspace } from '$lib/stores/workspaces';
	import { notifyWorkspaceAppsUpdated } from '$lib/utils/workspaceAppsEvents';
	import {
		createApp,
		deleteApp,
		getAppCatalog,
		getApps,
		installCatalogApp,
		updateApp,
		type AppLaunchMode,
		type AppProvider,
		type AppStatus,
		type AppType,
		type AppUrlClass,
		type CatalogApp,
		type WorkspaceApp
	} from '$lib/api/apps';
	import {
		ArrowUpRight,
		Check,
		LayoutGrid,
		List,
		Loader2,
		Monitor,
		Pencil,
		Plus,
		Search,
		Store,
		Table2,
		Trash2,
		X
	} from 'lucide-svelte';

	let apps = $state<WorkspaceApp[]>([]);
	let catalogApps = $state<CatalogApp[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let query = $state('');
	let view = $state<'store' | 'installed'>('store');
	let display = $state<'grid' | 'list' | 'table'>('grid');
	let category = $state('all');

	let showEdit = $state(false);
	let saving = $state(false);
	let editing = $state<WorkspaceApp | null>(null);
	let failedLogos = $state<Set<string>>(new Set());

	const APP_TYPES: AppType[] = ['web_app', 'mini_app', 'internal_app', 'client_app', 'embedded_tool'];
	const PROVIDERS: AppProvider[] = [
		'custom',
		'miosa',
		'openai',
		'anthropic',
		'perplexity',
		'google',
		'gohighlevel',
		'hubspot',
		'vercel',
		'netlify',
		'render',
		'replit'
	];
	const LAUNCH_MODES: AppLaunchMode[] = ['iframe', 'browser', 'external'];
	const STATUSES: AppStatus[] = ['active', 'draft', 'archived'];
	const URL_CLASSES: AppUrlClass[] = [
		'temporary_preview',
		'always_on_preview',
		'stable_sandbox_embed',
		'durable_deployment',
		'custom_domain'
	];

	let form = $state({
		name: '',
		app_type: 'web_app' as AppType,
		provider: 'custom' as AppProvider,
		url: '',
		launch_mode: 'browser' as AppLaunchMode,
		status: 'active' as AppStatus,
		icon: 'layout-grid',
		logo_url: '',
		color: '#111827',
		category: 'general',
		notes: '',
		show_on_desktop: true,
		show_in_dock: true,
		position_index: 0,
		url_class: 'custom_domain' as AppUrlClass
	});

	let wsId = $state<string | null | undefined>(undefined);
	let loadGeneration = 0;
	$effect(() => {
		const id = $currentWorkspace?.id ?? null;
		if (id !== wsId) {
			wsId = id;
			void reload();
		}
	});

	let searchTimer: ReturnType<typeof setTimeout>;
	function onSearch() {
		clearTimeout(searchTimer);
		searchTimer = setTimeout(() => void reload(), 250);
	}

	function clearSearch() {
		query = '';
		clearTimeout(searchTimer);
		void reload();
	}

	function reload(): Promise<void> {
		const workspaceId = wsId;
		const generation = ++loadGeneration;
		if (!workspaceId || workspaceId.startsWith('kb:')) {
			apps = [];
			catalogApps = [];
			error = null;
			loading = false;
			return Promise.resolve();
		}
		return load(workspaceId, generation);
	}

	async function load(workspaceId: string, generation: number) {
		loading = true;
		error = null;
		try {
			const search = query.trim() || undefined;
			const [catalog, res] = await Promise.all([
				getAppCatalog(search, workspaceId),
				getApps(search, false, workspaceId)
			]);
			if (generation !== loadGeneration || workspaceId !== wsId) return;
			apps = res.apps;
			catalogApps = catalog.apps;
			if (category !== 'all' && !catalog.apps.some((app) => app.category === category) && !res.apps.some((app) => app.category === category)) {
				category = 'all';
			}
		} catch (e) {
			if (generation !== loadGeneration || workspaceId !== wsId) return;
			error = e instanceof Error ? e.message : 'Failed to load apps';
		} finally {
			if (generation === loadGeneration && workspaceId === wsId) {
				loading = false;
			}
		}
	}

	const CATEGORY_LABELS: Record<string, string> = {
		ai: 'AI',
		ads: 'Ads',
		automation: 'Automation',
		calendar: 'Calendar',
		commerce: 'Commerce',
		communication: 'Communication',
		creative: 'Creative',
		crm: 'CRM',
		database: 'Data',
		deployment: 'Deployment',
		design: 'Design',
		development: 'Development',
		docs: 'Docs',
		email: 'Email',
		files: 'Files',
		forms: 'Forms',
		general: 'General',
		meetings: 'Meetings',
		payments: 'Payments',
		'project-management': 'Project management',
		scheduling: 'Scheduling',
		sites: 'Sites'
	};

	function categoryLabel(value: string) {
		return CATEGORY_LABELS[value] ?? label(value);
	}

	const categoryOptions = $derived.by(() => {
		const counts = new Map<string, number>();
		const source = view === 'store' ? catalogApps : apps;
		for (const app of source) {
			const key = app.category || 'general';
			counts.set(key, (counts.get(key) ?? 0) + 1);
		}
		return Array.from(counts.entries())
			.sort((a, b) => categoryLabel(a[0]).localeCompare(categoryLabel(b[0])))
			.map(([id, count]) => ({ id, label: categoryLabel(id), count }));
	});

	const visibleCatalogApps = $derived.by(() => {
		if (category === 'all') return catalogApps;
		return catalogApps.filter((app) => app.category === category);
	});

	const visibleInstalledApps = $derived.by(() => {
		if (category === 'all') return apps;
		return apps.filter((app) => app.category === category);
	});

	$effect(() => {
		if (category === 'all') return;
		if (!categoryOptions.some((option) => option.id === category)) {
			category = 'all';
		}
	});

	const grouped = $derived.by(() => {
		const order = ['active', 'draft', 'archived'];
		const map = new Map<string, WorkspaceApp[]>();
		for (const app of visibleInstalledApps) {
			const key = app.status || 'active';
			if (!map.has(key)) map.set(key, []);
			map.get(key)!.push(app);
		}
		return Array.from(map.entries()).sort((a, b) => {
			const ai = order.indexOf(a[0]);
			const bi = order.indexOf(b[0]);
			return (ai === -1 ? 99 : ai) - (bi === -1 ? 99 : bi) || a[0].localeCompare(b[0]);
		});
	});

	const storeGroups = $derived.by(() => {
		const map = new Map<string, CatalogApp[]>();
		for (const app of visibleCatalogApps) {
			const key = app.category || 'general';
			if (!map.has(key)) map.set(key, []);
			map.get(key)!.push(app);
		}
		return Array.from(map.entries()).sort((a, b) => categoryLabel(a[0]).localeCompare(categoryLabel(b[0])));
	});

	function displayUrl(u: string) {
		return u.replace(/^https?:\/\//, '').replace(/\/$/, '');
	}

	function hrefFor(u: string) {
		if (!u) return '';
		return /^https?:\/\//.test(u) ? u : `https://${u}`;
	}

	function postDesktopMessage(payload: Record<string, unknown>) {
		const target = window.parent && window.parent !== window ? window.parent : window;
		target.postMessage(payload, window.location.origin);
	}

	function desktopMessageApp(app: WorkspaceApp) {
		return {
			id: app.id,
			name: app.name,
			url: hrefFor(app.url),
			launchMode: app.launch_mode,
			logoUrl: app.logo_url,
			color: app.color,
			showOnDesktop: app.show_on_desktop,
			showInDock: app.show_in_dock
		};
	}

	function notifyDesktopAppUpserted(app: WorkspaceApp) {
		postDesktopMessage({ type: 'businessos:workspace-app-upserted', app: desktopMessageApp(app) });
	}

	function notifyDesktopAppRemoved(app: WorkspaceApp) {
		postDesktopMessage({
			type: 'businessos:workspace-app-removed',
			app: desktopMessageApp(app)
		});
	}

	function notifyDesktopAppsRefresh() {
		postDesktopMessage({ type: 'businessos:workspace-apps-refresh' });
	}

	function openWorkspaceAppWindow(app: WorkspaceApp) {
		postDesktopMessage({
			type: 'businessos:open-workspace-app',
			app: desktopMessageApp(app)
		});
	}

	function previewWorkspaceAppWindow(app: WorkspaceApp | CatalogApp) {
		postDesktopMessage({
			type: 'businessos:preview-workspace-app',
			app: {
				id: app.id,
				name: app.name,
				url: hrefFor(app.url),
				launchMode: app.launch_mode,
				logoUrl: app.logo_url,
				color: app.color,
				showOnDesktop: false
			}
		});
	}

	function label(value: string) {
		return value.replaceAll('_', ' ');
	}

	function launchLabel(app: WorkspaceApp | CatalogApp) {
		if (app.launch_mode === 'browser') return 'Browser';
		if (app.launch_mode === 'external') return 'External';
		return 'BusinessOS iframe';
	}

	function launchHelp(app: WorkspaceApp | CatalogApp) {
		if (app.launch_mode === 'browser') return 'Opens in a browser tab for SaaS login compatibility.';
		if (app.launch_mode === 'external') return 'Opens outside BusinessOS and does not use an embedded session.';
		return 'Opens inside the BusinessOS desktop with an embedded app frame.';
	}

	function surfaceLabel(app: WorkspaceApp | CatalogApp) {
		if (app.provider === 'miosa' || app.url_class === 'stable_sandbox_embed' || app.url_class === 'durable_deployment') {
			return 'MIOSA/deployed app';
		}
		if (app.url_class === 'custom_domain' && app.launch_mode !== 'iframe') return 'SaaS link';
		return label(app.app_type);
	}

	function launchOptionLabel(mode: AppLaunchMode) {
		if (mode === 'iframe') return 'BusinessOS iframe';
		if (mode === 'browser') return 'Browser tab';
		return 'External system';
	}

	function fallbackInitial(app: WorkspaceApp | CatalogApp) {
		return app.name.trim().slice(0, 1).toUpperCase() || 'A';
	}

	function logoFor(app: WorkspaceApp | CatalogApp) {
		return app.logo_url?.trim() || faviconFor(app.url);
	}

	function faviconFor(rawUrl: string) {
		try {
			const normalized = hrefFor(rawUrl);
			const parsed = new URL(normalized);
			return `${parsed.origin}/favicon.ico`;
		} catch {
			return '';
		}
	}

	function useFavicon() {
		form.logo_url = faviconFor(form.url);
	}

	function logoVisible(app: WorkspaceApp | CatalogApp) {
		return Boolean(logoFor(app)) && !failedLogos.has(app.id);
	}

	function hideBrokenImage(app: WorkspaceApp | CatalogApp) {
		failedLogos = new Set([...failedLogos, app.id]);
	}

	function openNew() {
		editing = null;
		form = {
			name: '',
			app_type: 'web_app',
			provider: 'custom',
			url: '',
			launch_mode: 'browser',
			status: 'active',
			icon: 'layout-grid',
			logo_url: '',
			color: '#111827',
			category: 'general',
			notes: '',
			show_on_desktop: true,
			show_in_dock: true,
			position_index: apps.length,
			url_class: 'custom_domain'
		};
		showEdit = true;
	}

	function openEdit(app: WorkspaceApp) {
		editing = app;
		form = {
			name: app.name,
			app_type: app.app_type ?? 'web_app',
			provider: app.provider ?? 'custom',
			url: app.url ?? '',
			launch_mode: app.launch_mode ?? 'iframe',
			status: app.status ?? 'active',
			icon: app.icon ?? 'layout-grid',
			logo_url: app.logo_url ?? '',
			color: app.color ?? '#111827',
			category: app.category ?? 'general',
			notes: app.notes ?? '',
			show_on_desktop: app.show_on_desktop,
			show_in_dock: app.show_in_dock,
			position_index: app.position_index ?? 0,
			url_class: app.url_class ?? 'stable_sandbox_embed'
		};
		showEdit = true;
	}

	async function save(e: Event) {
		e.preventDefault();
		if (!form.name.trim() || !form.url.trim()) return;
		saving = true;
		error = null;
		try {
			const body = {
				...form,
				name: form.name.trim(),
				url: form.url.trim(),
				icon: form.icon.trim(),
				logo_url: form.logo_url.trim(),
				color: form.color.trim(),
				category: form.category.trim(),
				notes: form.notes.trim(),
				position_index: Number(form.position_index) || 0
			};
			const saved = editing ? await updateApp(editing.id, body) : await createApp(body);
			showEdit = false;
			notifyDesktopAppUpserted(saved);
			await reload();
			notifyWorkspaceAppsUpdated();
			notifyDesktopAppsRefresh();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to save app';
		} finally {
			saving = false;
		}
	}

	async function remove(app: WorkspaceApp) {
		if (!confirm(`Delete "${app.name}"?`)) return;
		try {
			await deleteApp(app.id);
			apps = apps.filter((x) => x.id !== app.id);
			notifyDesktopAppRemoved(app);
			notifyWorkspaceAppsUpdated();
			notifyDesktopAppsRefresh();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to delete app';
		}
	}

	async function install(app: CatalogApp) {
		if (app.installed) {
			view = 'installed';
			return;
		}
		saving = true;
		error = null;
		try {
			const installed = await installCatalogApp(app.id);
			notifyWorkspaceAppsUpdated();
			notifyDesktopAppUpserted(installed);
			await reload();
			view = 'installed';
			notifyDesktopAppsRefresh();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to install app';
		} finally {
			saving = false;
		}
	}

	async function toggleDesktop(app: WorkspaceApp) {
		if (app.read_only) return;
		const next = { ...app, show_on_desktop: !app.show_on_desktop };
		apps = apps.map((x) => (x.id === app.id ? next : x));
		try {
			await updateApp(app.id, {
				name: next.name,
				app_type: next.app_type,
				provider: next.provider,
				url: next.url,
				launch_mode: next.launch_mode,
				status: next.status,
				icon: next.icon,
				logo_url: next.logo_url,
				color: next.color,
				category: next.category,
				notes: next.notes,
				show_on_desktop: next.show_on_desktop,
				show_in_dock: next.show_in_dock,
				position_index: next.position_index,
				url_class: next.url_class
			});
			notifyDesktopAppUpserted(next);
			notifyWorkspaceAppsUpdated();
			notifyDesktopAppsRefresh();
		} catch (e) {
			apps = apps.map((x) => (x.id === app.id ? app : x));
			error = e instanceof Error ? e.message : 'Failed to update app';
		}
	}

	function launch(app: WorkspaceApp | CatalogApp) {
		if (!app.url) return;
		if ('source' in app) {
			openWorkspaceAppWindow(app);
			return;
		}
		const installed = apps.find(
			(item) =>
				item.catalog_app_id === app.id ||
				item.url === app.url ||
				item.name.toLowerCase() === app.name.toLowerCase()
		);
		if (installed) {
			openWorkspaceAppWindow(installed);
			return;
		}
		previewWorkspaceAppWindow(app);
	}

	function installedForCatalogApp(app: CatalogApp) {
		return apps.find(
			(item) =>
				item.catalog_app_id === app.id ||
				item.url === app.url ||
				item.name.toLowerCase() === app.name.toLowerCase()
		);
	}

	async function openOrInstall(app: CatalogApp) {
		const installed = installedForCatalogApp(app);
		if (installed) {
			openWorkspaceAppWindow(installed);
			return;
		}
		await install(app);
	}

</script>

<svelte:head>
	<title>Apps - BusinessOS</title>
</svelte:head>

<div class="apps-root">
	<header class="topbar">
		<div class="title-wrap">
			<div class="page-icon"><LayoutGrid size={18} strokeWidth={1.9} /></div>
			<div>
				<h1>Apps</h1>
				<div class="subline">{apps.length} installed · {catalogApps.length} store apps · BusinessOS-first launch</div>
			</div>
		</div>
		<div class="tools">
			<div class="search">
				<Search size={15} />
				<input bind:value={query} oninput={onSearch} placeholder="Search apps..." aria-label="Search apps" />
			</div>
			<button class="btn btn--primary" onclick={openNew}><Plus size={16} strokeWidth={2.4} />New app</button>
		</div>
	</header>

	<div class="controlbar">
		<div class="tabs" aria-label="App section">
			<button class:active={view === 'store'} onclick={() => (view = 'store')}><Store size={14} />Store</button>
			<button class:active={view === 'installed'} onclick={() => (view = 'installed')}><Check size={14} />Installed</button>
		</div>
		<div class="view-toggle" aria-label="Display mode">
			<button class:active={display === 'grid'} title="Grid view" onclick={() => (display = 'grid')}><LayoutGrid size={14} /></button>
			<button class:active={display === 'list'} title="List view" onclick={() => (display = 'list')}><List size={14} /></button>
			<button class:active={display === 'table'} title="Table view" onclick={() => (display = 'table')}><Table2 size={14} /></button>
		</div>
	</div>

	{#if error}<div class="banner banner--error">{error}</div>{/if}

	{#if loading}
		<div class="loading"><Loader2 class="spin" size={20} /> Loading apps...</div>
	{:else if view === 'store'}
		{#if catalogApps.length === 0 && query.trim()}
			<div class="empty">
				<Search size={26} strokeWidth={1.4} />
				<p>No store apps match "{query.trim()}".</p>
				<button class="btn btn--ghost" onclick={clearSearch}>Clear search</button>
			</div>
		{:else if catalogApps.length === 0}
			<div class="empty">
				<LayoutGrid size={30} strokeWidth={1.4} />
				<p>No catalog apps yet.</p>
				<button class="btn btn--primary" onclick={openNew}><Plus size={16} />Add custom app</button>
			</div>
		{:else}
			<div class="catalog-layout">
				<aside class="category-panel">
					<button class:active={category === 'all'} onclick={() => (category = 'all')}>
						<span>All apps</span><span>{catalogApps.length}</span>
					</button>
					{#each categoryOptions as item}
						<button class:active={category === item.id} onclick={() => (category = item.id)}>
							<span>{item.label}</span><span>{item.count}</span>
						</button>
					{/each}
				</aside>

				<div class="store-main">
					<div class="section-title">
						<div>
							<h2>{category === 'all' ? 'App Store' : categoryLabel(category)}</h2>
							<p>{visibleCatalogApps.length} apps ready to add to this workspace</p>
						</div>
						<span class="session-pill">Browser launch keeps SaaS logins</span>
					</div>

					{#if visibleCatalogApps.length === 0}
						<div class="empty empty--inline">
							<Search size={24} strokeWidth={1.4} />
							<p>No store apps in {categoryLabel(category)}.</p>
							<button class="btn btn--ghost" onclick={() => (category = 'all')}>Show all apps</button>
						</div>
					{:else if display === 'table'}
						<div class="data-table">
							<table>
								<thead>
									<tr>
										<th>App</th>
										<th>Type</th>
										<th>Launch</th>
										<th>URL</th>
										<th></th>
									</tr>
								</thead>
								<tbody>
									{#each visibleCatalogApps as app (app.id)}
										<tr>
											<td>
												<div class="table-app">
													<div class="app-icon app-icon--table">
														{#if logoVisible(app)}
															<img class="app-logo" src={logoFor(app)} alt="" onerror={() => hideBrokenImage(app)} />
														{:else}
															<span>{fallbackInitial(app)}</span>
														{/if}
													</div>
													<div><strong>{app.name}</strong><small>{label(app.provider)}</small></div>
												</div>
											</td>
											<td>{surfaceLabel(app)}</td>
											<td><span class="launch-chip" title={launchHelp(app)}>{launchLabel(app)}</span></td>
											<td><span class="app-url app-url--display">{displayUrl(app.url)}</span></td>
											<td class="table-actions">
												{#if !app.installed}<button class="btn btn--ghost" onclick={() => launch(app)}>Preview</button>{/if}
												<button class="btn btn--primary" onclick={() => openOrInstall(app)} disabled={saving}>{app.installed ? 'Open' : 'Add'}</button>
											</td>
										</tr>
									{/each}
								</tbody>
							</table>
						</div>
					{:else if display === 'list'}
						<div class="rows">
							{#each visibleCatalogApps as app (app.id)}
								<article class="app-row">
									<div class="app-icon">
										{#if logoVisible(app)}
											<img class="app-logo" src={logoFor(app)} alt="" onerror={() => hideBrokenImage(app)} />
										{:else}
											<span>{fallbackInitial(app)}</span>
										{/if}
									</div>
									<div class="row-main">
										<div class="row-title">
											<strong>{app.name}</strong>
											<span>{surfaceLabel(app)}</span>
											<span class="launch-chip" title={launchHelp(app)}>{launchLabel(app)}</span>
										</div>
										<p>{app.notes || displayUrl(app.url)}</p>
									</div>
									<div class="row-actions">
										{#if !app.installed}<button class="btn btn--ghost" onclick={() => launch(app)}><ArrowUpRight size={15} />Preview</button>{/if}
										<button class="btn btn--primary" onclick={() => openOrInstall(app)} disabled={saving}>{app.installed ? 'Open' : 'Add'}</button>
									</div>
								</article>
							{/each}
						</div>
					{:else}
						<div class="list">
							{#each storeGroups as [cat, items]}
								<section class="group">
									<div class="group-head">
										<span>{categoryLabel(cat)}</span>
										<span class="group-count">{items.length}</span>
									</div>
									<div class="grid">
										{#each items as app (app.id)}
											<article class="app-card">
												<div class="app-top">
													<div class="app-icon">
														{#if logoVisible(app)}
															<img class="app-logo" src={logoFor(app)} alt="" onerror={() => hideBrokenImage(app)} />
														{:else}
															<span>{fallbackInitial(app)}</span>
														{/if}
													</div>
													<div class="app-main">
														<div class="app-name">{app.name}</div>
														<span class="app-url app-url--display">{displayUrl(app.url)}</span>
													</div>
												</div>

												<div class="tags tags--clean">
													<span>{categoryLabel(app.category)}</span>
													<span>{surfaceLabel(app)}</span>
													<span class="launch-chip" title={launchHelp(app)}>{launchLabel(app)}</span>
													{#if app.installed}<span class="generated">installed</span>{/if}
												</div>

												{#if app.notes}<p class="notes">{app.notes}</p>{/if}

												<div class="card-actions">
													{#if !app.installed}
														<button class="btn btn--ghost" onclick={() => launch(app)}>
															<ArrowUpRight size={15} />Preview
														</button>
													{/if}
													<button class="btn btn--primary" onclick={() => openOrInstall(app)} disabled={saving}>
														{app.installed ? 'Open' : 'Add'}
													</button>
												</div>
											</article>
										{/each}
									</div>
								</section>
							{/each}
						</div>
					{/if}
				</div>
			</div>
		{/if}
	{:else if visibleInstalledApps.length === 0 && query.trim()}
		<div class="empty">
			<Search size={26} strokeWidth={1.4} />
			<p>No apps match "{query.trim()}".</p>
			<button class="btn btn--ghost" onclick={clearSearch}>Clear search</button>
		</div>
	{:else if visibleInstalledApps.length === 0 && category !== 'all'}
		<div class="empty">
			<Search size={26} strokeWidth={1.4} />
			<p>No installed apps in {categoryLabel(category)}.</p>
			<button class="btn btn--ghost" onclick={() => (category = 'all')}>Show all installed</button>
		</div>
	{:else if visibleInstalledApps.length === 0}
		<div class="empty">
			<LayoutGrid size={30} strokeWidth={1.4} />
			<p>No apps yet.</p>
			<button class="btn btn--primary" onclick={openNew}><Plus size={16} />Add app</button>
		</div>
	{:else}
		<div class="catalog-layout">
			<aside class="category-panel">
				<button class:active={category === 'all'} onclick={() => (category = 'all')}>
					<span>All installed</span><span>{apps.length}</span>
				</button>
				{#each categoryOptions as item}
					<button class:active={category === item.id} onclick={() => (category = item.id)}>
						<span>{item.label}</span><span>{item.count}</span>
					</button>
				{/each}
			</aside>
			<div class="store-main">
				<div class="section-title">
					<div>
						<h2>Installed apps</h2>
						<p>{visibleInstalledApps.length} apps saved to this workspace</p>
					</div>
					<button class="btn btn--ghost" onclick={openNew}><Plus size={15} />Custom app</button>
				</div>

				{#if display === 'table'}
					<div class="data-table">
						<table>
							<thead>
								<tr><th>App</th><th>Category</th><th>Launch</th><th>Desktop</th><th>Actions</th></tr>
							</thead>
							<tbody>
								{#each visibleInstalledApps as app (app.id)}
									<tr>
										<td>
											<div class="table-app">
												<div class="app-icon app-icon--table">
													{#if logoVisible(app)}
														<img class="app-logo" src={logoFor(app)} alt="" onerror={() => hideBrokenImage(app)} />
													{:else}
														<span>{fallbackInitial(app)}</span>
													{/if}
												</div>
												<div><strong>{app.name}</strong><small>{displayUrl(app.url)}</small></div>
											</div>
										</td>
										<td>{surfaceLabel(app)}</td>
										<td><span class="launch-chip" title={launchHelp(app)}>{launchLabel(app)}</span></td>
										<td><button class:active={app.show_on_desktop} class="switch" onclick={() => toggleDesktop(app)} disabled={app.read_only}><Monitor size={13} />Desktop</button></td>
										<td class="table-actions">
											<button class="btn btn--ghost" onclick={() => launch(app)} disabled={!app.url}>Open</button>
											{#if !app.read_only}<button class="icon-btn" title="Edit" onclick={() => openEdit(app)}><Pencil size={14} /></button><button class="icon-btn icon-btn--danger" title="Delete" onclick={() => remove(app)}><Trash2 size={14} /></button>{/if}
										</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				{:else}
					<div class={display === 'list' ? 'rows' : 'list'}>
						{#each grouped as [status, items]}
							<section class="group">
								<div class="group-head">
									<span>{status}</span>
									<span class="group-count">{items.length}</span>
								</div>
								<div class={display === 'list' ? 'rows' : 'grid'}>
									{#each items as app (app.id)}
										<article class={display === 'list' ? 'app-row' : 'app-card'}>
											<div class="app-top">
												<div class="app-icon">
													{#if logoVisible(app)}
														<img class="app-logo" src={logoFor(app)} alt="" onerror={() => hideBrokenImage(app)} />
													{:else}
														<span>{fallbackInitial(app)}</span>
													{/if}
												</div>
												<div class="app-main">
													<div class="app-name">{app.name}</div>
													{#if app.url}
														<span class="app-url app-url--display">{displayUrl(app.url)}</span>
													{:else}
														<div class="app-url app-url--empty">No deployment URL</div>
													{/if}
												</div>
												{#if !app.read_only}<button class="icon-btn" title="Edit" onclick={() => openEdit(app)}><Pencil size={14} /></button>{/if}
											</div>
											<div class="tags tags--clean">
												<span class:generated={app.source === 'generated'}>{app.source === 'generated' ? 'built app' : 'manual app'}</span>
												<span>{surfaceLabel(app)}</span>
												<span class="launch-chip" title={launchHelp(app)}>{launchLabel(app)}</span>
											</div>
											{#if app.notes}<p class="notes">{app.notes}</p>{/if}
											<div class="switch-row">
												<button class:active={app.show_on_desktop} class="switch" onclick={() => toggleDesktop(app)} disabled={app.read_only}><Monitor size={13} />Desktop</button>
												<span class:active={app.show_in_dock} class="dock-pill">Dock</span>
												<span class="position">#{app.position_index}</span>
											</div>
											<div class="card-actions">
												<button class="btn btn--ghost" onclick={() => launch(app)} disabled={!app.url}><ArrowUpRight size={15} />Open</button>
												{#if !app.read_only}<button class="icon-btn icon-btn--danger" title="Delete" onclick={() => remove(app)}><Trash2 size={14} /></button>{/if}
											</div>
										</article>
									{/each}
								</div>
							</section>
						{/each}
					</div>
				{/if}
			</div>
		</div>
	{/if}
</div>

{#if showEdit}
	<div class="overlay" role="button" tabindex="0" onclick={() => (showEdit = false)} onkeydown={(e) => e.key === 'Escape' && (showEdit = false)}>
		<div class="modal" role="dialog" tabindex="-1" onclick={(e) => e.stopPropagation()} onkeydown={() => {}}>
			<div class="modal-head">
				<h2>{editing ? 'Edit app' : 'New app'}</h2>
				<button class="card-x" onclick={() => (showEdit = false)} aria-label="Close"><X size={18} /></button>
			</div>
				<form onsubmit={save}>
					<label class="field"><span>Name</span><input bind:value={form.name} placeholder="e.g. Client portal" required /></label>
					<label class="field"><span>URL</span><input bind:value={form.url} placeholder="https://example.com" required /></label>
					<div class="field-row">
						<label class="field"><span>Type</span>
							<select bind:value={form.app_type}>
								{#each APP_TYPES as type}<option value={type}>{label(type)}</option>{/each}
							</select>
						</label>
						<label class="field"><span>Provider</span>
							<select bind:value={form.provider}>
								{#each PROVIDERS as provider}<option value={provider}>{label(provider)}</option>{/each}
							</select>
						</label>
					</div>
					<div class="field-row">
						<label class="field"><span>Launch</span>
							<select bind:value={form.launch_mode}>
								{#each LAUNCH_MODES as mode}<option value={mode}>{launchOptionLabel(mode)}</option>{/each}
							</select>
						</label>
						<label class="field"><span>Status</span>
							<select bind:value={form.status}>
								{#each STATUSES as st}<option value={st}>{label(st)}</option>{/each}
							</select>
						</label>
					</div>
					<div class="field-row">
						<label class="field"><span>URL class</span>
							<select bind:value={form.url_class}>
								{#each URL_CLASSES as cls}<option value={cls}>{label(cls)}</option>{/each}
							</select>
						</label>
						<label class="field"><span>Category</span><input bind:value={form.category} placeholder="general" /></label>
					</div>
					<div class="field-row">
						<label class="field"><span>Position</span><input bind:value={form.position_index} type="number" min="0" /></label>
						<label class="field"><span>Icon</span><input bind:value={form.icon} placeholder="layout-grid" /></label>
					</div>
					<div class="field-row">
						<label class="field"><span>Color</span><input bind:value={form.color} placeholder="#111827" /></label>
					</div>
				<div class="logo-row">
					<label class="field"><span>Logo URL</span><input bind:value={form.logo_url} placeholder="https://example.com/favicon.ico" /></label>
					<button type="button" class="btn btn--ghost logo-btn" onclick={useFavicon} disabled={!form.url.trim()}>Use favicon</button>
				</div>
				<div class="checks">
					<label><input type="checkbox" bind:checked={form.show_on_desktop} />Desktop</label>
					<label><input type="checkbox" bind:checked={form.show_in_dock} />Dock</label>
				</div>
				<label class="field"><span>Notes</span><textarea bind:value={form.notes} rows="3"></textarea></label>
				<div class="modal-actions">
					<button type="button" class="btn btn--ghost" onclick={() => (showEdit = false)}>Cancel</button>
					<button type="submit" class="btn btn--primary" disabled={saving || !form.name.trim() || !form.url.trim()}>
						{#if saving}<Loader2 class="spin" size={15} />{/if}{editing ? 'Save' : 'Add app'}
					</button>
				</div>
			</form>
		</div>
	</div>
{/if}

<style>
	.apps-root { height: 100%; display: flex; flex-direction: column; background: var(--dbg); color: var(--dt); overflow: hidden; }
	.topbar { display: flex; align-items: center; justify-content: space-between; gap: 16px; padding: 16px 20px 14px; border-bottom: 1px solid var(--dbd); flex-shrink: 0; }
	.title-wrap { display: flex; align-items: center; gap: 12px; min-width: 0; }
	.page-icon { width: 36px; height: 36px; border-radius: 8px; border: 1px solid var(--dbd); background: color-mix(in srgb, var(--dt) 5%, transparent); color: var(--dt2); display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
	h1 { font-size: 1.12rem; font-weight: 700; margin: 0; letter-spacing: 0; }
	.subline { font-size: 0.74rem; color: var(--dt3); margin-top: 2px; }
	.tools { display: flex; align-items: center; gap: 10px; }
	.search { display: flex; align-items: center; gap: 7px; padding: 7px 11px; border: 1px solid var(--dbd); border-radius: 8px; color: var(--dt3); }
	.search input { background: transparent; border: none; outline: none; color: var(--dt); font-size: 0.84rem; width: 190px; font-family: inherit; }
	.btn { display: inline-flex; align-items: center; justify-content: center; gap: 7px; min-height: 34px; padding: 8px 14px; border-radius: 8px; font-size: 0.83rem; font-weight: 620; cursor: pointer; border: 1px solid transparent; white-space: nowrap; font-family: inherit; }
	.btn--primary { background: var(--dt); color: var(--dbg); }
	.btn--ghost { background: transparent; border-color: var(--dbd); color: var(--dt2); text-decoration: none; }
	.btn:disabled { opacity: 0.55; cursor: not-allowed; }
	.controlbar { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 10px 20px; border-bottom: 1px solid var(--dbd); flex-shrink: 0; }
	.tabs, .view-toggle { display: inline-flex; align-items: center; gap: 3px; padding: 3px; border: 1px solid var(--dbd); border-radius: 8px; background: color-mix(in srgb, var(--dt) 2%, transparent); }
	.tabs button, .view-toggle button { display: inline-flex; align-items: center; justify-content: center; gap: 6px; min-height: 28px; border: 0; border-radius: 6px; padding: 6px 10px; background: transparent; color: var(--dt3); font: inherit; font-size: 0.78rem; font-weight: 620; cursor: pointer; }
	.view-toggle button { width: 30px; padding: 0; }
	.tabs button.active, .view-toggle button.active { background: var(--dt); color: var(--dbg); }
	.loading, .empty { flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 14px; color: var(--dt3); text-align: center; padding: 0 24px; }
	.empty--inline { min-height: 320px; border: 1px dashed var(--dbd); border-radius: 8px; }
	.empty p { max-width: 420px; line-height: 1.5; }
	.banner { margin: 14px 24px 0; padding: 11px 14px; border-radius: 8px; font-size: 0.83rem; }
	.banner--error { background: color-mix(in srgb, #ef4444 12%, transparent); color: #ef4444; }

	.catalog-layout { flex: 1; min-height: 0; display: grid; grid-template-columns: 190px minmax(0, 1fr); overflow: hidden; }
	.category-panel { border-right: 1px solid var(--dbd); padding: 14px 10px; overflow-y: auto; background: color-mix(in srgb, var(--dt) 1.5%, transparent); }
	.category-panel button { width: 100%; min-height: 32px; display: flex; align-items: center; justify-content: space-between; gap: 10px; padding: 7px 9px; border: 0; border-radius: 7px; background: transparent; color: var(--dt3); font: inherit; font-size: 0.78rem; cursor: pointer; }
	.category-panel button:hover { background: color-mix(in srgb, var(--dt) 5%, transparent); color: var(--dt); }
	.category-panel button.active { background: color-mix(in srgb, var(--dt) 9%, transparent); color: var(--dt); font-weight: 700; }
	.category-panel button span:last-child { color: var(--dt4); font-size: 0.7rem; }
	.store-main { min-width: 0; overflow-y: auto; padding: 18px 20px 80px; }
	.section-title { max-width: 1180px; margin: 0 auto 16px; display: flex; align-items: center; justify-content: space-between; gap: 12px; }
	.section-title h2 { margin: 0; font-size: 0.95rem; font-weight: 760; }
	.section-title p { margin: 3px 0 0; color: var(--dt3); font-size: 0.76rem; }
	.session-pill { display: inline-flex; align-items: center; min-height: 28px; border: 1px solid var(--dbd); border-radius: 999px; padding: 5px 10px; color: var(--dt3); font-size: 0.72rem; white-space: nowrap; }
	.list { width: 100%; }
	.group { max-width: 1180px; margin: 0 auto 24px; }
	.group-head { display: flex; align-items: center; gap: 8px; margin-bottom: 12px; padding-bottom: 8px; border-bottom: 1px solid var(--dbd); color: var(--dt3); font-size: 0.72rem; font-weight: 700; letter-spacing: 0.05em; text-transform: uppercase; }
	.group-count { display: inline-flex; align-items: center; justify-content: center; min-width: 22px; height: 20px; border-radius: 999px; background: color-mix(in srgb, var(--dt) 8%, transparent); color: var(--dt3); font-size: 0.68rem; }
	.grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(260px, 1fr)); gap: 10px; }
	.app-card { border: 1px solid var(--dbd); border-radius: 8px; padding: 12px; background: color-mix(in srgb, var(--dt) 1.8%, var(--dbg)); min-height: 188px; display: flex; flex-direction: column; gap: 10px; }
	.app-card:hover { border-color: color-mix(in srgb, var(--dt) 20%, var(--dbd)); }
	.app-top { display: flex; align-items: flex-start; gap: 11px; }
	.app-icon { width: 38px; height: 38px; border-radius: 8px; border: 1px solid color-mix(in srgb, var(--dt) 9%, var(--dbd)); background: #fff; color: #111827; display: flex; align-items: center; justify-content: center; flex-shrink: 0; box-shadow: inset 0 0 0 1px rgba(255,255,255,0.6); font-size: 0.82rem; font-weight: 760; }
	.app-icon--table { width: 30px; height: 30px; border-radius: 7px; }
	.app-logo { width: 24px; height: 24px; object-fit: contain; border-radius: 4px; }
	.app-main { flex: 1; min-width: 0; }
	.app-name { font-size: 0.94rem; font-weight: 700; color: var(--dt); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
	.app-url { display: inline-flex; align-items: center; gap: 5px; max-width: 100%; color: var(--dt3); text-decoration: none; font-size: 0.78rem; margin-top: 3px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
	.app-url:hover { color: var(--dt); text-decoration: underline; }
	.app-url--display:hover { color: var(--dt3); text-decoration: none; }
	.app-url--empty:hover { color: var(--dt3); text-decoration: none; }
	.tags { display: flex; flex-wrap: wrap; gap: 6px; }
	.tags span, .dock-pill, .position { border: 1px solid var(--dbd); border-radius: 999px; padding: 3px 8px; font-size: 0.68rem; color: var(--dt3); line-height: 1.2; }
	.tags--clean span:nth-child(n + 5) { display: none; }
	.tags span.generated { background: color-mix(in srgb, #22c55e 11%, transparent); border-color: color-mix(in srgb, #22c55e 28%, var(--dbd)); color: #16a34a; }
	.launch-chip { border-color: color-mix(in srgb, #0ea5e9 28%, var(--dbd)) !important; color: #0284c7 !important; background: color-mix(in srgb, #0ea5e9 8%, transparent); }
	.notes { margin: 0; color: var(--dt3); font-size: 0.8rem; line-height: 1.45; white-space: pre-wrap; min-height: 34px; }
	.switch-row { display: flex; align-items: center; gap: 7px; margin-top: auto; }
	.switch { display: inline-flex; align-items: center; gap: 6px; border: 1px solid var(--dbd); border-radius: 999px; padding: 4px 9px; background: transparent; color: var(--dt3); font-size: 0.72rem; cursor: pointer; font-family: inherit; }
	.switch:disabled { cursor: default; opacity: 0.7; }
	.switch.active, .dock-pill.active { background: color-mix(in srgb, #22c55e 12%, transparent); border-color: color-mix(in srgb, #22c55e 30%, var(--dbd)); color: #16a34a; }
	.card-actions { display: flex; align-items: center; justify-content: space-between; gap: 8px; padding-top: 2px; }

	.rows { max-width: 1180px; margin: 0 auto; display: flex; flex-direction: column; gap: 8px; }
	.app-row { border: 1px solid var(--dbd); border-radius: 8px; background: color-mix(in srgb, var(--dt) 1.5%, var(--dbg)); padding: 10px 12px; display: flex; align-items: center; gap: 12px; }
	.app-row .app-top { flex: 1; min-width: 0; }
	.row-main { flex: 1; min-width: 0; }
	.row-title { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
	.row-title strong { font-size: 0.9rem; }
	.row-title span { color: var(--dt3); font-size: 0.72rem; border: 1px solid var(--dbd); border-radius: 999px; padding: 2px 7px; }
	.row-main p { margin: 3px 0 0; color: var(--dt3); font-size: 0.78rem; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
	.row-actions { display: flex; align-items: center; gap: 8px; }

	.data-table { max-width: 1180px; margin: 0 auto; border: 1px solid var(--dbd); border-radius: 8px; overflow: hidden; background: color-mix(in srgb, var(--dt) 1%, var(--dbg)); }
	table { width: 100%; border-collapse: collapse; font-size: 0.8rem; }
	th { height: 36px; padding: 0 12px; border-bottom: 1px solid var(--dbd); text-align: left; color: var(--dt3); font-weight: 700; background: color-mix(in srgb, var(--dt) 2.5%, transparent); }
	td { padding: 10px 12px; border-bottom: 1px solid color-mix(in srgb, var(--dt) 6%, transparent); color: var(--dt2); vertical-align: middle; }
	tr:last-child td { border-bottom: 0; }
	.table-app { display: flex; align-items: center; gap: 10px; min-width: 0; }
	.table-app strong { display: block; color: var(--dt); font-size: 0.82rem; }
	.table-app small { display: block; color: var(--dt3); margin-top: 2px; max-width: 220px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
	.table-actions { display: flex; align-items: center; justify-content: flex-end; gap: 6px; white-space: nowrap; }

	.icon-btn { display: inline-flex; align-items: center; justify-content: center; width: 30px; height: 30px; border-radius: 7px; border: none; background: transparent; color: var(--dt3); cursor: pointer; flex-shrink: 0; }
	.icon-btn:hover { background: color-mix(in srgb, var(--dt) 8%, transparent); color: var(--dt); }
	.icon-btn--danger:hover { background: color-mix(in srgb, #ef4444 12%, transparent); color: #ef4444; }
	.overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.55); display: flex; align-items: center; justify-content: center; z-index: 100; padding: 20px; }
	.modal { width: 100%; max-width: 560px; max-height: calc(100vh - 48px); overflow-y: auto; background: var(--dbg); border: 1px solid var(--dbd); border-radius: 12px; padding: 22px; box-shadow: 0 24px 60px rgba(0,0,0,0.5); }
	.modal-head { display: flex; align-items: center; justify-content: space-between; margin-bottom: 18px; }
	.modal-head h2 { font-size: 1.05rem; font-weight: 700; margin: 0; }
	.card-x { display: inline-flex; align-items: center; justify-content: center; width: 30px; height: 30px; border-radius: 7px; border: none; background: transparent; color: var(--dt3); cursor: pointer; }
	.card-x:hover { background: color-mix(in srgb, var(--dt) 8%, transparent); color: var(--dt); }
	.field { display: flex; flex-direction: column; gap: 6px; margin-bottom: 14px; flex: 1; min-width: 0; }
	.field span { font-size: 0.78rem; font-weight: 620; color: var(--dt2); }
	.field input, .field textarea, .field select { width: 100%; padding: 9px 11px; border-radius: 8px; border: 1px solid var(--dbd); background: var(--dbg); color: var(--dt); font-size: 0.84rem; outline: none; font-family: inherit; resize: vertical; }
	.field-row { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; }
	.logo-row { display: grid; grid-template-columns: 1fr auto; gap: 12px; align-items: end; }
	.logo-btn { margin-bottom: 14px; }
	.checks { display: flex; align-items: center; gap: 16px; margin: 0 0 14px; color: var(--dt2); font-size: 0.82rem; }
	.checks label { display: inline-flex; align-items: center; gap: 7px; }
	.modal-actions { display: flex; justify-content: flex-end; gap: 10px; padding-top: 4px; }
	:global(.spin) { animation: spin 0.8s linear infinite; }
	@keyframes spin { to { transform: rotate(360deg); } }
	@media (max-width: 760px) {
		.topbar { align-items: stretch; flex-direction: column; }
		.tools, .controlbar { align-items: stretch; flex-direction: column; }
		.search { flex: 1; }
		.search input { width: 100%; }
		.catalog-layout { grid-template-columns: 1fr; }
		.category-panel { display: flex; gap: 6px; overflow-x: auto; border-right: 0; border-bottom: 1px solid var(--dbd); }
		.category-panel button { width: auto; flex: 0 0 auto; }
		.store-main { padding: 14px 12px 64px; }
		.section-title { align-items: flex-start; flex-direction: column; }
		.app-row { align-items: stretch; flex-direction: column; }
		.row-actions { justify-content: flex-end; }
		.data-table { overflow-x: auto; }
		table { min-width: 680px; }
		.field-row { grid-template-columns: 1fr; gap: 0; }
		.logo-row { grid-template-columns: 1fr; gap: 0; }
		.logo-btn { margin: 0 0 14px; }
	}
</style>
