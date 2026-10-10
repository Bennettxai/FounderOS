<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { getUserAvatarUrl, useSession, signOut } from '$lib/auth-client';
	import { Separator } from 'bits-ui';
	import { browser } from '$app/environment';
	import { isElectron as checkElectron, isMacOS } from '$lib/utils/platform';
	import { desktopSettings } from '$lib/stores/desktopStore';
	import { onMount } from 'svelte';
	import { api } from '$lib/api';
	import { getBackendUrl, initCSRF } from '$lib/api/base';
	import { WorkspaceSwitcher } from '$lib/components/workspace';
	import { loadSavedWorkspace } from '$lib/stores/workspaces';
	import { notificationStore } from '$lib/stores/notifications';
	import { initializePush } from '$lib/services/pushService';
	import { initializeAppShellContext } from '$lib/services/appShellBootstrap';
	import { getInstallations, getModule, getWorkspaceModules } from '$lib/api/modules';
	import type { WorkspaceModuleItem } from '$lib/api/modules';
	import { listBoards } from '$lib/api/boards';
	import type { Board } from '$lib/api/boards';
	import { desktop3dStore, BUILTIN_MODULES, DYNAMIC_MODULE_COLORS } from '$lib/stores/desktop3dStore';
	import { windowStore } from '$lib/stores/windowStore';
	import { currentWorkspace, currentWorkspaceId } from '$lib/stores/workspaces';
	import {
		ChevronsLeft, Monitor, ChevronRight, ChevronDown, Search, Plus, Settings,
		LayoutDashboard, Bot, BookOpen, Sparkles, Inbox, Calendar, MessageSquare,
		Users, FolderKanban, ListChecks, Repeat, GitBranch, Tag,
		BriefcaseBusiness, TrendingUp,
		Megaphone, Globe, Contact, FileText,
		LayoutGrid, Image, Package, Cpu, Wrench, HardDrive,
		Wallet, BarChart3, Database, UsersRound, Plug, Shield,
		Library, Bell, HelpCircle, Circle, Boxes, LogOut, BookA,
		Home, Filter, Workflow, Share2, Clapperboard, Handshake, CandlestickChart, Crosshair,
		Network, Brain, Stethoscope, Waypoints, Gauge, Layers
	} from 'lucide-svelte';

	// Module icon registry — add a module = add a name here, no SVG paths.
	const ICONS: Record<string, any> = {
		command: LayoutDashboard, agents: Bot, knowledge: BookOpen, glossary: BookA, intelligence: Sparkles, inbox: Inbox, calendar: Calendar, communication: MessageSquare,
		relationships: Users, projects: FolderKanban, tasks: ListChecks, rhythm: Repeat, pipelines: GitBranch, offers: Tag,
		clients: BriefcaseBusiness, crm: TrendingUp,
		campaigns: Megaphone, sites: Globe, personas: Contact, content: FileText,
		apps: LayoutGrid, assets: Image, deliverables: Package, engines: Cpu, builders: Wrench, drive: HardDrive, boards: LayoutDashboard,
		finance: Wallet, analytics: BarChart3, data: Database, team: UsersRound, connectors: Plug, admin: Shield,
		resources: Library, notifications: Bell, help: HelpCircle,
		// FounderOS v1 modules
		home: Home, funnel: Filter, workflows: Workflow, social: Share2, clapperboard: Clapperboard,
		handshake: Handshake, trading: CandlestickChart, crosshair: Crosshair, org: Network, brain: Brain,
		doctor: Stethoscope, blueprint: Waypoints, usage: Gauge, layers: Layers
	};
	import { SearchPalette } from '@miosa/optimal-engine';
	import '@miosa/optimal-engine/styles/notion.css';
	import { getEngine, syncEngineConfig } from '$lib/optimal-engine/context';
	import { getWorkspaceSidebarGroups } from '$lib/config/workspaceModules';

	// ---------------------------------------------------------------------------
	// Engine Search Palette state
	// ---------------------------------------------------------------------------

	let engineSearchOpen = $state(false);

	function handleEngineSearchSelect(payload: { type: 'memory' | 'wiki' | 'signal'; id?: string; slug?: string }) {
		if (payload.type === 'wiki' && payload.slug) {
			goto(`/pages?view=knowledge-graph`);
		} else if (payload.type === 'memory' && payload.id) {
			goto(`/pages/${payload.id}`);
		} else if (payload.type === 'signal' && payload.id) {
			goto(`/nodes/${payload.id}`);
		}
	}

	function handleGlobalKeydown(e: KeyboardEvent) {
		if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
			// Don't steal focus from active inputs/textareas unless it's the palette itself
			const active = document.activeElement;
			if (
				active &&
				(active.tagName === 'INPUT' || active.tagName === 'TEXTAREA') &&
				!engineSearchOpen
			) {
				return;
			}
			e.preventDefault();
			engineSearchOpen = !engineSearchOpen;
		}
	}

	// Projects state for dropdown
	let projects = $state<Array<{id: string, name: string, status: string}>>([]);
	let showProjectsDropdown = $state(false);

	// Load projects for sidebar dropdown
	async function loadProjects() {
		try {
			const data = await api.getProjects('active');
			projects = data.slice(0, 5); // Show top 5 active projects
		} catch (e) {
			console.error('Failed to load projects:', e);
		}
	}

	// Workspace custom modules for sidebar "Your modules" section
	let workspaceModules = $state<WorkspaceModuleItem[]>([]);
	let workspaceModulesRequest = 0;

	async function loadWorkspaceModules(wsId: string) {
		const requestId = ++workspaceModulesRequest;
		try {
			const modules = await getWorkspaceModules(wsId);
			if (requestId === workspaceModulesRequest && wsId === $currentWorkspaceId) {
				workspaceModules = modules;
			}
		} catch {
			// API unavailable — show empty state, do not break the shell
			if (requestId === workspaceModulesRequest) workspaceModules = [];
		}
	}

	// Pinned boards for the sidebar "Boards" section (workspace-scoped;
	// the request helper injects X-Workspace-ID, wsId just keys the reload)
	let pinnedBoards = $state<Board[]>([]);

	async function loadPinnedBoards(wsId: string) {
		try {
			if (!wsId) return;
			pinnedBoards = await listBoards(true);
		} catch {
			// Boards API unavailable (table may not exist yet) - show nothing
			pinnedBoards = [];
		}
	}

	// Re-fetch whenever the active workspace changes
	$effect(() => {
		const wsId = $currentWorkspaceId;
		if (wsId) {
			loadWorkspaceModules(wsId);
			loadPinnedBoards(wsId);
		} else {
			workspaceModulesRequest += 1;
			workspaceModules = [];
			pinnedBoards = [];
		}
	});

	// Sync installed dynamic modules to 3D desktop + dock on startup
	async function syncDynamicModules() {
		try {
			const result = await getInstallations();
			const installations = result.installations ?? [];
			if (!installations.length) return;

			const builtinSet = new Set<string>(BUILTIN_MODULES);
			const dynamicInstalls = installations.filter(
				(inst) => inst.is_active && !builtinSet.has(inst.module_id)
			);
			if (!dynamicInstalls.length) return;

			// Fetch all module details in parallel — avoids N+1 sequential requests
			const results = await Promise.allSettled(
				dynamicInstalls.map((inst) => getModule(inst.module_id))
			);

			for (let idx = 0; idx < results.length; idx++) {
				const result = results[idx];
				const inst = dynamicInstalls[idx];

				if (result.status === 'fulfilled') {
					const mod = result.value;
					desktop3dStore.addModule({
						id: mod.id,
						title: mod.name,
						icon: mod.icon ?? 'box',
						color: DYNAMIC_MODULE_COLORS[mod.category] ?? DYNAMIC_MODULE_COLORS.general,
						category: mod.category,
					});
					windowStore.addToDock(mod.id);
				} else {
					// Module details unavailable — add with fallback
					desktop3dStore.addModule({
						id: inst.module_id,
						title: inst.module_id,
						icon: 'box',
						color: DYNAMIC_MODULE_COLORS.general,
					});
				}
			}
		} catch {
			// Module API not available yet — skip silently
		}
	}

	onMount(async () => {
		// First-run detection: only send an UNAUTHENTICATED desktop user to the
		// welcome / "Start Local vs Connect to Cloud" screen. A user who is already
		// signed in (e.g. via the web → desktop businessos://auth/callback handoff)
		// has a valid session here — do NOT bounce them back to onboarding.
		const isElectron = browser && 'electron' in window;
		const isAuthed = !!data?.user;
		if (isAuthed && !localStorage.getItem('businessos_setup_complete')) {
			// Signed-in = cloud mode. Mark complete so we never loop to /welcome.
			localStorage.setItem('businessos_setup_complete', 'cloud');
			localStorage.setItem('businessos_mode', 'cloud');
		}
		if (isElectron && !isAuthed && !localStorage.getItem('businessos_setup_complete')) {
			goto('/welcome');
			return;
		}
		// Auto-mark web users as cloud setup complete
		if (browser && !localStorage.getItem('businessos_setup_complete')) {
			localStorage.setItem('businessos_setup_complete', 'cloud');
			localStorage.setItem('businessos_mode', 'cloud');
		}

		// Embedded desktop modules run in their own iframe and therefore have their
		// own Svelte store instance. They still need the active workspace restored
		// before workspace-scoped pages can load data. Keep the heavier shell-only
		// systems disabled below, but do not leave embedded modules unscoped.
		const initializedContext = await initializeAppShellContext(isEmbedMode, {
			initCSRF,
			loadSavedWorkspace,
			loadProjects
		});
		if (initializedContext === 'embedded') return;

		// Guard: new users with no workspace at all go to onboarding.
		// Only check after loadSavedWorkspace() has had a chance to resolve.
		// Avoid redirecting from /onboarding itself to prevent an infinite loop.
		const { workspaces: wsStore } = await import('$lib/stores/workspaces');
		const { get } = await import('svelte/store');
		const loadedWorkspaces = get(wsStore);
		if (loadedWorkspaces.length === 0 && !window.location.pathname.startsWith('/onboarding')) {
			goto('/onboarding');
			return;
		}

		// Load the workspace's Optimal Engine connection so search/memory/context
		// use the shared engine app-wide.
		const wsId = localStorage.getItem('businessos_current_workspace_id');
		if (wsId) syncEngineConfig(wsId);

		// These don't return promises — fire immediately
		notificationStore.initialize();
		initializePush();

		// Sync dynamic modules after workspace is loaded
		syncDynamicModules();
	});

	const APP_VERSION = '0.0.1';

	let { children, data } = $props();

	const session = useSession();

	// Check if we're in embed mode (used by desktop windows)
	const isEmbedMode = $derived($page.url.searchParams.get('embed') === 'true');

	// No loading screen for app routes - instant load
	let bootComplete = $state(true);

	// Check if running in Electron (for native window styling)
	const inElectron = $derived(browser && checkElectron());
	const onMac = $derived(browser && isMacOS());
	const needsTrafficLightSpace = $derived(inElectron && onMac);

	// Sidebar collapsed state (persisted to localStorage)
	let isCollapsed = $state(false);

	$effect(() => {
		// Load collapsed state from localStorage
		const stored = localStorage.getItem('sidebar-collapsed');
		if (stored !== null) {
			isCollapsed = stored === 'true';
		}
	});

	function toggleSidebar() {
		isCollapsed = !isCollapsed;
		localStorage.setItem('sidebar-collapsed', String(isCollapsed));
	}

	// Mobile drawer state — off-canvas on <768px, ignored on desktop
	let drawerOpen = $state(false);

	function openDrawer() {
		drawerOpen = true;
	}

	function closeDrawer() {
		drawerOpen = false;
	}

	// Close drawer when a nav link is tapped on mobile, then route explicitly.
	// Electron occasionally misses SvelteKit's link interception after HMR;
	// explicit goto keeps sidebar navigation immediate instead of waiting for refresh.
	function handleNavClick(event: MouseEvent) {
		drawerOpen = false;
		if (event.metaKey || event.ctrlKey || event.shiftKey || event.altKey || event.button !== 0) return;
		const anchor = (event.currentTarget as HTMLElement).closest('a');
		const href = anchor?.getAttribute('href');
		if (!href || href.startsWith('http') || href.startsWith('mailto:') || href.startsWith('#')) return;
		event.preventDefault();
		void goto(href);
	}

	function isNavItemActive(item: NavItem): boolean {
		const target = new URL(item.href, $page.url.origin);
		if (!$page.url.pathname.startsWith(target.pathname)) return false;

		for (const [key, value] of target.searchParams.entries()) {
			if ($page.url.searchParams.get(key) !== value) return false;
		}

		return target.searchParams.size > 0 || $page.url.pathname.startsWith(item.href);
	}

	// Auth is handled server-side in +layout.server.ts — no duplicate client-side check needed.
	// The server redirects to /login if the session cookie is invalid, preventing double auth round-trips.

	interface NavItem {
		id?: string;
		href: string;
		label: string;
		icon: string;
		adminOnly?: boolean;
		shared?: boolean;
	}

	interface NavGroup {
		label: string;
		items: NavItem[];
	}

	let collapsedSections = $state<Set<string>>(new Set(['Growth', 'Build', 'Manage']));

	function toggleSection(label: string) {
		if (collapsedSections.has(label)) {
			collapsedSections.delete(label);
		} else {
			collapsedSections.add(label);
		}
		collapsedSections = new Set(collapsedSections);
	}

	// Finalized BusinessOS module model (workspace = container, module = interface type).
	// 5 groups · build order: Knowledge → Command → Agents → Projects/Tasks → …
	// item.icon is a key into ICONS. Modules route to real views; the rest 404 until built.
	const primaryNav: NavItem[] = [];

	const legacyNavGroups: NavGroup[] = [
		{
			label: 'Operate',
			items: [
				{ href: '/dashboard', label: 'Command', icon: 'command' },
				{ href: '/agents', label: 'Agents', icon: 'agents' },
				{ href: '/knowledge', label: 'Knowledge', icon: 'knowledge' },
				{ href: '/glossary', label: 'Glossary', icon: 'glossary' },
				{ href: '/intelligence', label: 'Intelligence', icon: 'intelligence' },
				{ href: '/inbox', label: 'Inbox', icon: 'inbox' },
				{ href: '/calendar', label: 'Calendar', icon: 'calendar' },
				{ href: '/communication', label: 'Communications', icon: 'communication' }
			]
		},
		{
			label: 'Business',
			items: [
				{ href: '/relationships', label: 'Relationships', icon: 'relationships' },
				{ href: '/clients', label: 'Clients', icon: 'clients' },
				{ href: '/crm', label: 'CRM', icon: 'crm' },
				{ href: '/projects', label: 'Projects', icon: 'projects' },
				{ href: '/tasks', label: 'Tasks', icon: 'tasks' },
				{ href: '/rhythm', label: 'Rhythm', icon: 'rhythm' },
				{ href: '/pipelines', label: 'Pipelines', icon: 'pipelines' },
				{ href: '/offers', label: 'Offers', icon: 'offers' }
			]
		},
		{
			label: 'Growth',
			items: [
				{ href: '/campaigns', label: 'Campaigns', icon: 'campaigns' },
				{ href: '/sites', label: 'Sites', icon: 'sites' },
				{ href: '/personas', label: 'Personas', icon: 'personas' }
			]
		},
		{
			label: 'Content',
			items: [
				{ href: '/content?scope=my', label: 'My Content', icon: 'content' },
				{ href: '/content?scope=clients', label: 'Client Content', icon: 'relationships' }
			]
		},
		{
			label: 'Build',
			items: [
				{ href: '/apps', label: 'Apps', icon: 'apps' },
				{ href: '/boards', label: 'Boards', icon: 'boards' },
				{ href: '/assets', label: 'Assets', icon: 'assets' },
				{ href: '/deliverables', label: 'Deliverables', icon: 'deliverables' },
				{ href: '/engines', label: 'Engines', icon: 'engines' },
				{ href: '/builders', label: 'Builders', icon: 'builders' },
				{ href: '/drive', label: 'Drive', icon: 'drive' }
			]
		},
		{
			label: 'Manage',
			items: [
				{ href: '/finance', label: 'Finance', icon: 'finance' },
				{ href: '/analytics', label: 'Analytics', icon: 'analytics' },
				{ href: '/data', label: 'Data', icon: 'data' },
				{ href: '/team', label: 'Team', icon: 'team' },
				{ href: '/connectors', label: 'Connectors', icon: 'connectors' },
				{ href: '/admin', label: 'Admin', icon: 'admin', adminOnly: true },
				{ href: '/resources', label: 'Resources', icon: 'resources' },
				{ href: '/notifications', label: 'Notifications', icon: 'notifications' },
				{ href: '/help', label: 'Help Center', icon: 'help' }
			]
		}
	];

	const navGroups = $derived.by(() => {
		const configuredGroups = getWorkspaceSidebarGroups(
			$currentWorkspace?.settings ?? {},
			workspaceModules
		);
		return configuredGroups.length > 0 ? configuredGroups : legacyNavGroups;
	});

	const canSeeAdminNav = $derived($session.data?.user?.platform_role === 'superadmin');

	// When the profile image fails to load (e.g. an expired Google avatar URL),
	// fall back to the initial circle instead of a broken <img> showing its alt
	// text crammed into the avatar box.
	let avatarError = $state(false);
	let avatarSource = $derived(getUserAvatarUrl($session.data?.user?.image));

	$effect(() => {
		avatarSource;
		avatarError = false;
	});

	let loggingOut = $state(false);
	async function handleLogout() {
		if (loggingOut) return;
		loggingOut = true;
		try {
			await signOut();
		} catch {
			loggingOut = false;
		}
	}

</script>

{#if $session.data}
	{#if isEmbedMode}
		<!-- Embed mode: no sidebar, just content -->
		<div class="h-screen w-screen overflow-hidden sb-main-bg">
			{@render children()}
		</div>
	{:else}
	<div class="h-screen flex overflow-hidden p-2 gap-2 sb-canvas">

		<!-- Mobile top bar — visible only below 768px -->
		<div class="sb-topbar">
			<button
				class="sb-topbar-hamburger"
				onclick={openDrawer}
				aria-label="Open navigation"
			>
				<span class="sb-hamburger-line"></span>
				<span class="sb-hamburger-line"></span>
				<span class="sb-hamburger-line"></span>
			</button>
			<span class="sb-topbar-title">
				{$currentWorkspace?.name || 'BusinessOS'}
			</span>
			<button
				class="sb-topbar-search"
				onclick={() => (engineSearchOpen = true)}
				aria-label="Search"
			>
				<Search size={18} strokeWidth={2} />
			</button>
		</div>

		<!-- Mobile backdrop — tapping closes the drawer -->
		{#if drawerOpen}
			<!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
			<div class="sb-backdrop" onclick={closeDrawer} aria-hidden="true"></div>
		{/if}

		<!-- Sidebar — persistent on desktop, off-canvas drawer on mobile -->
		<aside
			class="sb-sidebar flex flex-col flex-shrink-0 transition-all duration-300 ease-in-out rounded-[14px] overflow-hidden {isCollapsed ? (needsTrafficLightSpace ? 'w-20' : 'w-16') : 'w-64'} {drawerOpen ? 'sb-drawer--open' : ''}"
		>
			<!-- Draggable titlebar region for Electron (traffic light area) -->
			{#if needsTrafficLightSpace}
				<div
					class="h-12 flex-shrink-0 drag-region"
					style="-webkit-app-region: drag;"
				>
					<!-- Traffic light spacer - this area is for the macOS window controls -->
				</div>
			{:else}
				<div class="h-4 flex-shrink-0"></div>
			{/if}

			<!-- Header: workspace switcher + collapse -->
			<div class="sb-head {isCollapsed ? 'sb-head--collapsed' : ''}" style="-webkit-app-region: no-drag;">
				{#if !isCollapsed}
					<div class="sb-head-ws"><WorkspaceSwitcher /></div>
					<!-- Mobile: close drawer; Desktop: collapse sidebar -->
					<button
						onclick={() => { closeDrawer(); toggleSidebar(); }}
						class="sb-icon-btn sb-collapse-btn flex-shrink-0"
						title="Collapse sidebar"
						aria-label="Collapse sidebar"
					>
						<ChevronsLeft size={18} strokeWidth={2} />
					</button>
					<button
						onclick={closeDrawer}
						class="sb-icon-btn sb-drawer-close-btn flex-shrink-0"
						aria-label="Close navigation"
					>
						<ChevronsLeft size={18} strokeWidth={2} />
					</button>
				{:else}
					<button
						class="sb-workspace-badge"
						title={$currentWorkspace?.name || 'Workspace'}
						onclick={toggleSidebar}
						aria-label="Expand sidebar to switch workspace"
					>
						{$currentWorkspace?.name?.charAt(0).toUpperCase() || 'W'}
					</button>
				{/if}
			</div>

			<!-- Search (opens ⌘K palette) -->
			<div class="px-2 pb-1">
				<button
					class="sb-search {isCollapsed ? 'sb-search--collapsed' : ''}"
					onclick={() => (engineSearchOpen = true)}
					title="Search (⌘K)"
					aria-label="Search"
				>
					<Search size={16} strokeWidth={2} class="flex-shrink-0" />
					{#if !isCollapsed}
						<span class="sb-search-label">Search</span>
						<kbd class="sb-kbd">⌘K</kbd>
					{/if}
				</button>
			</div>

			<!-- Navigation -->
			<nav class="flex-1 px-2 pt-1 overflow-y-auto sb-nav-scroll">
				<!-- Primary (flat, no header) -->
				<div class="sb-section-items">
					{#each primaryNav as item}
						<a
							href={item.href}
							onclick={handleNavClick}
							class="sb-nav-item flex items-center gap-2.5 px-2.5 py-1.5 rounded-lg text-[0.86rem]
								{isNavItemActive(item) ? 'sb-nav-item--active' : ''}
								{isCollapsed ? 'justify-center' : ''}"
							title={isCollapsed ? item.label : ''}
						>
							<svg class="w-[18px] h-[18px] flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.6" d={item.icon} />
							</svg>
									{#if !isCollapsed}<span>{item.label}</span>{/if}
									{#if !isCollapsed && item.shared}
										<span class="sb-module-shared-badge">shared</span>
									{/if}
								</a>
					{/each}
				</div>

				<!-- Grouped modules -->
				{#each navGroups as group, gi}
					{#if !isCollapsed}
						<button
							class="sb-section-header"
							onclick={() => toggleSection(group.label)}
							aria-label="Toggle {group.label} section"
						>
							<span class="sb-section-label">{group.label}</span>
							<ChevronRight size={13} strokeWidth={2.2} class="sb-section-chevron {collapsedSections.has(group.label) ? '' : 'sb-section-chevron--open'}" />
						</button>
					{:else if gi >= 0}
						<div class="sb-section-dot"></div>
					{/if}

					{#if !collapsedSections.has(group.label)}
						<div class="sb-section-items">
							{#each group.items as item}
								{#if !item.adminOnly || canSeeAdminNav}
									{@const Ico = ICONS[item.icon] ?? Circle}
									<a
										href={item.href}
										onclick={handleNavClick}
										class="sb-nav-item flex items-center gap-2.5 px-2.5 py-1.5 rounded-lg text-[0.86rem]
											{isNavItemActive(item) ? 'sb-nav-item--active' : ''}
											{isCollapsed ? 'justify-center' : ''}"
										title={isCollapsed ? item.label : ''}
									>
										<Ico size={18} strokeWidth={1.8} class="flex-shrink-0" />
										{#if !isCollapsed}<span>{item.label}</span>{/if}
									</a>
								{/if}
							{/each}
						</div>
					{/if}
				{/each}

				<!-- Boards - pinned workspace boards -->
				{#if pinnedBoards.length > 0}
					{#if !isCollapsed}
						<button
							class="sb-section-header"
							onclick={() => toggleSection('Boards')}
							aria-label="Toggle Boards section"
						>
							<span class="sb-section-label">Boards</span>
							<ChevronRight size={13} strokeWidth={2.2} class="sb-section-chevron {collapsedSections.has('Boards') ? '' : 'sb-section-chevron--open'}" />
						</button>
					{:else}
						<div class="sb-section-dot"></div>
					{/if}

					{#if !collapsedSections.has('Boards')}
						<div class="sb-section-items">
							{#each pinnedBoards as board (board.id)}
								<a
									href="/boards/{board.id}"
									onclick={handleNavClick}
									class="sb-nav-item flex items-center gap-2.5 px-2.5 py-1.5 rounded-lg text-[0.86rem]
										{$page.url.pathname.startsWith(`/boards/${board.id}`) ? 'sb-nav-item--active' : ''}
										{isCollapsed ? 'justify-center' : ''}"
									title={isCollapsed ? board.name : ''}
								>
									<LayoutDashboard size={18} strokeWidth={1.8} class="flex-shrink-0" />
									{#if !isCollapsed}<span class="flex-1 truncate">{board.name}</span>{/if}
								</a>
							{/each}
						</div>
					{/if}
				{/if}

				<!-- All modules / New module CTA -->
				<a
					href="/modules"
					onclick={handleNavClick}
					class="sb-add {isCollapsed ? 'sb-add--collapsed' : ''}"
					title="All modules"
				>
					<Plus size={16} strokeWidth={2.2} class="flex-shrink-0" />
					{#if !isCollapsed}<span>{workspaceModules.length > 0 ? 'All modules' : 'New module'}</span>{/if}
				</a>
			</nav>

			<Separator.Root class="sb-sep h-px" />

			<!-- Footer: Desktop + Settings -->
			<div class="px-2 py-1.5 flex flex-col gap-0.5">
				<a
					href="/window"
					onclick={handleNavClick}
					class="sb-nav-item flex items-center gap-2.5 px-2.5 py-1.5 rounded-lg text-[0.86rem]
						{$page.url.pathname.startsWith('/window') ? 'sb-nav-item--active' : ''}
						{isCollapsed ? 'justify-center' : ''}"
					title={isCollapsed ? 'Desktop' : ''}
				>
					<Monitor size={18} strokeWidth={1.8} class="flex-shrink-0" />
					{#if !isCollapsed}
						<span>Desktop</span>
						<span class="ml-auto sb-desktop-badge">View</span>
					{/if}
				</a>
				<a
					href="/settings"
					onclick={handleNavClick}
					class="sb-nav-item flex items-center gap-2.5 px-2.5 py-1.5 rounded-lg text-[0.86rem]
						{$page.url.pathname.startsWith('/settings') ? 'sb-nav-item--active' : ''}
						{isCollapsed ? 'justify-center' : ''}"
					title={isCollapsed ? 'Settings' : ''}
				>
					<Settings size={18} strokeWidth={1.8} class="flex-shrink-0" />
					{#if !isCollapsed}<span>Settings</span>{/if}
				</a>
			</div>

			<Separator.Root class="sb-sep h-px" />

			<!-- User Section - account link + sign out -->
			<div class="p-2 flex items-center gap-1">
				<a
					href="/dashboard"
					onclick={handleNavClick}
					class="sb-user-link flex items-center gap-2.5 p-1.5 rounded-lg transition-colors flex-1 min-w-0 {$page.url.pathname === '/dashboard' ? 'sb-user-link--active' : ''} {isCollapsed ? 'justify-center' : ''}"
					title={isCollapsed ? 'Dashboard' : ''}
				>
					{#if avatarSource && !avatarError}
						<img
							src={avatarSource}
							alt={$session.data.user?.name || 'Profile'}
							onerror={() => (avatarError = true)}
							class="w-8 h-8 rounded-full object-cover flex-shrink-0 sb-avatar-border"
						/>
					{:else}
						<div class="w-8 h-8 rounded-full bg-gradient-to-br from-indigo-500 to-purple-600 text-white flex items-center justify-center text-xs font-semibold flex-shrink-0">
							{$session.data.user?.name?.charAt(0).toUpperCase() || 'U'}
						</div>
					{/if}
					{#if !isCollapsed}
						<div class="flex-1 min-w-0">
							<p class="text-[0.82rem] font-medium sb-user-name truncate">{$session.data.user?.name}</p>
							<p class="text-[0.72rem] sb-user-email truncate">{$session.data.user?.email}</p>
						</div>
						<ChevronRight size={15} strokeWidth={2} class="sb-user-chevron" />
					{/if}
				</a>
				{#if !isCollapsed}
					<button
						class="sb-logout-btn"
						onclick={handleLogout}
						disabled={loggingOut}
						title="Sign out"
						aria-label="Sign out"
					>
						<LogOut size={16} strokeWidth={2} />
					</button>
				{/if}
			</div>
		</aside>

		<!-- Main Content -->
		<main class="flex-1 flex flex-col min-w-0 overflow-hidden rounded-[14px] sb-panel">
			{#if needsTrafficLightSpace}
				<!-- Draggable titlebar region for main content area (Electron macOS only) -->
				<div
					class="h-12 flex-shrink-0 drag-region sb-main-border-b"
					style="-webkit-app-region: drag;"
				>
					<!-- This provides a drag area across the top of the main content -->
				</div>
				<!-- min-h-0 + flex-col are load-bearing: without min-h-0 a flex
				     item defaults to min-height:auto and refuses to shrink
				     below its content, letting child pages push their header
				     above the viewport. -->
				<div class="flex-1 min-h-0 overflow-hidden -mt-12 pt-12 flex flex-col">
					{@render children()}
				</div>
			{:else}
				<div class="flex-1 min-h-0 overflow-hidden flex flex-col">
					{@render children()}
				</div>
			{/if}
		</main>

	</div>
	{/if}

	<!-- Engine Search Palette — Cmd+K / Ctrl+K global search across memories, wiki, signals -->
	{#if browser}
		<SearchPalette
			engine={getEngine()}
			open={engineSearchOpen}
			onclose={() => (engineSearchOpen = false)}
			onselect={handleEngineSearchSelect}
		/>
	{/if}
{:else if !bootComplete}
	<!-- Loading state only during initial auth check -->
	<div class="h-screen flex items-center justify-center sb-main-bg">
		<div class="text-center">
			<div class="w-8 h-8 border-2 border-gray-300 border-t-blue-500 rounded-full animate-spin mx-auto mb-4"></div>
			<p class="sb-muted">Loading...</p>
		</div>
	</div>
{/if}

<svelte:window onkeydown={handleGlobalKeydown} />

<style>
	/* ══════════════════════════════════════════════════════════════ */
	/*  SIDEBAR (sb-) — Foundation Design Tokens                    */
	/* ══════════════════════════════════════════════════════════════ */

	/* ══════════════════════════════════════════════════════════════ */
	/*  MIOSA shell — theme-aware (light + dark) via design tokens.     */
	/*  Accent overlays use color-mix(--dt) so they adapt per theme.    */
	/* ══════════════════════════════════════════════════════════════ */

	/* Canvas — the background layer behind floating panels */
	.sb-canvas {
		background: var(--dbg2);
	}
	:global(.dark) .sb-canvas {
		background: #000000;
	}

	/* Panel — main content surface */
	.sb-panel {
		background: var(--dbg);
		border: 1px solid var(--dbd);
	}
	.sb-main-bg {
		background: var(--dbg);
	}
	.sb-main-border-b {
		border-bottom: 1px solid var(--dbd);
	}

	/* Sidebar container */
	.sb-sidebar {
		background: var(--dbg);
		border: 1px solid var(--dbd);
	}

	/* Title */
	.sb-title {
		color: var(--dt);
		font-weight: 650;
		letter-spacing: -0.015em;
	}

	/* Separator */
	.sb-sep {
		background: var(--dbd);
		opacity: 0.7;
	}

	/* Header row — workspace switcher + collapse */
	.sb-head {
		display: flex;
		align-items: center;
		gap: 6px;
		padding: 0 8px 8px;
	}
	.sb-head--collapsed {
		justify-content: center;
	}
	.sb-head-ws {
		flex: 1;
		min-width: 0;
	}
	.sb-icon-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 28px;
		height: 28px;
		border-radius: 8px;
		border: none;
		background: transparent;
		color: var(--dt3);
		cursor: pointer;
		transition: background 150ms ease, color 150ms ease;
	}
	.sb-icon-btn:hover {
		background: color-mix(in srgb, var(--dt) 8%, transparent);
		color: var(--dt);
	}

	/* Search field-button */
	.sb-search {
		display: flex;
		align-items: center;
		gap: 8px;
		width: 100%;
		padding: 7px 10px;
		border-radius: 9px;
		border: 1px solid var(--dbd);
		background: color-mix(in srgb, var(--dt) 3%, transparent);
		color: var(--dt3);
		cursor: pointer;
		transition: background 150ms ease, border-color 150ms ease, color 150ms ease;
	}
	.sb-search:hover {
		background: color-mix(in srgb, var(--dt) 6%, transparent);
		border-color: color-mix(in srgb, var(--dt) 18%, transparent);
		color: var(--dt2);
	}
	.sb-search--collapsed {
		justify-content: center;
		padding: 7px;
	}
	.sb-search-label {
		font-size: 0.84rem;
		font-weight: 500;
	}
	.sb-kbd {
		margin-left: auto;
		font-family: inherit;
		font-size: 0.66rem;
		font-weight: 600;
		letter-spacing: 0.02em;
		padding: 2px 6px;
		border-radius: 5px;
		color: var(--dt3);
		background: color-mix(in srgb, var(--dt) 7%, transparent);
		border: 1px solid var(--dbd);
	}

	/* Add module CTA */
	.sb-add {
		display: flex;
		align-items: center;
		gap: 8px;
		margin-top: 10px;
		padding: 8px 10px;
		border-radius: 9px;
		border: 1px dashed color-mix(in srgb, var(--dt) 18%, transparent);
		background: transparent;
		color: var(--dt3);
		font-size: 0.84rem;
		font-weight: 500;
		text-decoration: none;
		transition: background 150ms ease, border-color 150ms ease, color 150ms ease;
	}
	.sb-add:hover {
		background: color-mix(in srgb, var(--dt) 5%, transparent);
		border-color: color-mix(in srgb, var(--dt) 30%, transparent);
		color: var(--dt);
	}
	.sb-add--collapsed {
		justify-content: center;
		padding: 8px;
	}

	/* Shared-scope badge on workspace modules */
	.sb-module-shared-badge {
		font-size: 0.62rem;
		font-weight: 600;
		letter-spacing: 0.04em;
		text-transform: uppercase;
		padding: 1px 5px;
		border-radius: 4px;
		color: var(--dt3);
		background: color-mix(in srgb, var(--dt) 7%, transparent);
		border: 1px solid var(--dbd);
		flex-shrink: 0;
	}

	/* Desktop button */
	.sb-desktop-btn {
		background: color-mix(in srgb, var(--dt) 4%, transparent);
		border: 1px solid var(--dbd);
		color: var(--dt2);
		font-weight: 500;
	}
	.sb-desktop-btn:hover {
		background: color-mix(in srgb, var(--dt) 8%, transparent);
		color: var(--dt);
	}
	.sb-desktop-badge {
		color: var(--dt3);
		font-size: 0.65rem;
		letter-spacing: 0.04em;
		text-transform: uppercase;
	}

	/* ── Nav Items ──────────────────────────────────────────────── */
	.sb-nav-item {
		color: var(--dt2);
		position: relative;
		font-weight: 500;
		letter-spacing: -0.005em;
		transition: background 150ms ease, color 150ms ease;
	}
	.sb-nav-item :global(svg) {
		color: var(--dt3);
		transition: color 150ms ease;
	}
	.sb-nav-item::after {
		content: '';
		position: absolute;
		left: 0;
		top: 50%;
		transform: translateY(-50%);
		width: 3px;
		height: 0%;
		border-radius: 0 3px 3px 0;
		background: var(--dt);
		opacity: 0;
		transition: height 220ms cubic-bezier(0.4, 0, 0.2, 1), opacity 180ms ease;
	}
	.sb-nav-item:hover {
		background: color-mix(in srgb, var(--dt) 6%, transparent);
		color: var(--dt);
	}
	.sb-nav-item:hover :global(svg) {
		color: var(--dt2);
	}
	.sb-nav-item--active {
		background: color-mix(in srgb, var(--dt) 9%, transparent);
		color: var(--dt);
		font-weight: 600;
	}
	.sb-nav-item--active :global(svg) {
		color: var(--dt);
	}
	.sb-nav-item--active::after {
		height: 56%;
		opacity: 1;
	}
	.sb-nav-item--active:hover {
		background: color-mix(in srgb, var(--dt) 12%, transparent);
		color: var(--dt);
	}

	/* ── Section Headers ────────────────────────────────────────── */
	.sb-section-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		width: 100%;
		padding: 6px 12px 4px;
		margin-top: 14px;
		border: none;
		background: none;
		cursor: pointer;
	}
	.sb-section-header:first-child {
		margin-top: 2px;
	}
	.sb-section-label {
		font-size: 0.65rem;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.1em;
		color: var(--dt3);
	}
	.sb-section-chevron {
		width: 12px;
		height: 12px;
		color: var(--dt4);
		transition: transform 200ms ease;
	}
	.sb-section-chevron--open {
		transform: rotate(90deg);
	}
	.sb-section-dot {
		width: 4px;
		height: 4px;
		border-radius: 50%;
		background: var(--dbd);
		margin: 8px auto;
	}
	.sb-section-items {
		display: flex;
		flex-direction: column;
		gap: 2px;
	}
	.sb-nav-scroll {
		scrollbar-width: none;
	}
	.sb-nav-scroll::-webkit-scrollbar {
		display: none;
	}

	/* ── Workspace Badge (collapsed) ────────────────────────────── */
	.sb-workspace-badge {
		width: 34px;
		height: 34px;
		border-radius: 9px;
		background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%);
		border: none;
		color: #ffffff;
		font-size: 0.8rem;
		font-weight: 700;
		display: flex;
		align-items: center;
		justify-content: center;
		cursor: pointer;
		transition: transform 150ms ease, box-shadow 200ms ease;
	}
	.sb-workspace-badge:hover {
		transform: scale(1.06);
		box-shadow: 0 0 0 2px var(--dbg), 0 0 0 4px rgba(99, 102, 241, 0.4);
	}

	/* Sub-items (projects dropdown) */
	.sb-sub-item {
		color: var(--dt3);
	}
	.sb-sub-item:hover {
		background: color-mix(in srgb, var(--dt) 6%, transparent);
		color: var(--dt);
	}
	.sb-sub-item--active {
		background: color-mix(in srgb, var(--dt) 9%, transparent);
		color: var(--dt);
	}
	.sb-dot-inactive {
		background: var(--dt4);
	}

	/* Chevron toggle for projects dropdown */
	.sb-chevron-btn:hover {
		background: color-mix(in srgb, var(--dt) 8%, transparent);
	}
	.sb-chevron-icon {
		color: var(--dt3);
	}

	/* ── User Section ───────────────────────────────────────────── */
	.sb-user-link:hover {
		background: color-mix(in srgb, var(--dt) 6%, transparent);
	}
	.sb-logout-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 32px;
		height: 32px;
		flex-shrink: 0;
		border-radius: 8px;
		color: var(--dt3);
		background: transparent;
		transition: background 0.15s ease, color 0.15s ease;
	}
	.sb-logout-btn:hover {
		background: color-mix(in srgb, var(--dt) 8%, transparent);
		color: var(--dt);
	}
	.sb-logout-btn:disabled {
		opacity: 0.5;
		cursor: default;
	}
	.sb-user-link--active {
		background: color-mix(in srgb, var(--dt) 6%, transparent);
	}
	.sb-avatar-border {
		border: 2px solid var(--dbd);
	}
	.sb-user-name {
		color: var(--dt);
		font-weight: 550;
	}
	.sb-user-email {
		color: var(--dt3);
	}
	.sb-user-chevron {
		color: var(--dt4);
	}

	/* ══════════════════════════════════════════════════════════════ */
	/*  MOBILE — off-canvas drawer + top bar (<768px)               */
	/* ══════════════════════════════════════════════════════════════ */

	/* Top bar: hidden on desktop, visible on mobile */
	.sb-topbar {
		display: none;
	}

	/* Drawer close button: hidden on desktop (desktop uses collapse) */
	.sb-drawer-close-btn {
		display: none;
	}

	@media (max-width: 767px) {
		/* Remove desktop padding/gap — full-bleed on mobile */
		:global(.sb-canvas) {
			padding: 0 !important;
			gap: 0 !important;
		}

		/* Top bar — slim 48px bar across the top */
		.sb-topbar {
			display: flex;
			position: fixed;
			top: 0;
			left: 0;
			right: 0;
			z-index: 40;
			height: 48px;
			align-items: center;
			gap: 8px;
			padding: 0 12px;
			background: var(--dbg);
			border-bottom: 1px solid var(--dbd);
		}
		.sb-topbar-hamburger {
			display: flex;
			flex-direction: column;
			justify-content: center;
			gap: 4px;
			width: 40px;
			height: 40px;
			padding: 8px;
			border: none;
			background: transparent;
			cursor: pointer;
			border-radius: 8px;
			flex-shrink: 0;
		}
		.sb-topbar-hamburger:hover {
			background: color-mix(in srgb, var(--dt) 6%, transparent);
		}
		.sb-hamburger-line {
			display: block;
			width: 18px;
			height: 2px;
			border-radius: 2px;
			background: var(--dt2);
		}
		.sb-topbar-title {
			flex: 1;
			font-size: 0.9rem;
			font-weight: 600;
			color: var(--dt);
			letter-spacing: -0.01em;
			white-space: nowrap;
			overflow: hidden;
			text-overflow: ellipsis;
		}
		.sb-topbar-search {
			display: flex;
			align-items: center;
			justify-content: center;
			width: 40px;
			height: 40px;
			border: none;
			background: transparent;
			color: var(--dt3);
			cursor: pointer;
			border-radius: 8px;
			flex-shrink: 0;
		}
		.sb-topbar-search:hover {
			background: color-mix(in srgb, var(--dt) 6%, transparent);
			color: var(--dt);
		}

		/* Dim backdrop */
		.sb-backdrop {
			position: fixed;
			inset: 0;
			z-index: 49;
			background: rgba(0, 0, 0, 0.5);
		}

		/* Sidebar: off-canvas drawer, slides in from left */
		.sb-sidebar {
			position: fixed !important;
			top: 0;
			left: 0;
			bottom: 0;
			z-index: 50;
			width: 280px !important;
			border-radius: 0 14px 14px 0 !important;
			transform: translateX(-100%);
			transition: transform 280ms cubic-bezier(0.4, 0, 0.2, 1);
		}
		.sb-sidebar.sb-drawer--open {
			transform: translateX(0);
		}

		/* Main content: full-width, pushed down by top bar */
		.sb-panel {
			border-radius: 0 !important;
			border-left: none !important;
			border-right: none !important;
			border-bottom: none !important;
			margin-top: 48px;
			width: 100% !important;
			max-width: 100vw;
		}

		/* Ensure no horizontal overflow */
		:global(body) {
			overflow-x: hidden;
		}

		/* Hide desktop collapse button, show drawer close button */
		.sb-collapse-btn {
			display: none;
		}
		.sb-drawer-close-btn {
			display: flex;
		}

		/* Ensure nav touch targets meet 40px minimum */
		.sb-nav-item {
			min-height: 40px;
		}
		.sb-section-header {
			min-height: 40px;
		}
		.sb-search {
			min-height: 40px;
		}
	}
</style>
