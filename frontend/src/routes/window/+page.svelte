<script lang="ts">
	import { goto } from '$app/navigation';
	import { useSession } from '$lib/auth-client';
	import { getApps, type WorkspaceApp } from '$lib/api/apps';
	import { listWorkspaceDesktopSpaces, saveWorkspaceDesktopSpace } from '$lib/api/desktop-spaces';
	import { currentWorkspace, loadSavedWorkspace } from '$lib/stores/workspaces';
	import { get } from 'svelte/store';
	import { windowStore, visibleWindows, focusedWindow, type DesktopSpace, type SnapZone } from '$lib/stores/windowStore';
	import {
		connectDesktopPresenceRemote,
		desktopPresenceStore,
		followDesktopCursor,
		followedDesktopCursor,
		onDesktopSpaceUpdated,
		publishDesktopSpaceUpdated,
		publishDesktopCursor,
		setDesktopPresenceContext,
		startDesktopPresence,
		stopDesktopPresence,
		type DesktopPresenceCursor
	} from '$lib/stores/desktopPresenceStore';
	import { desktopSettings, getBackgroundCSS, isBackgroundDark } from '$lib/stores/desktopStore';
	import { deployedAppsStore } from '$lib/stores/deployedAppsStore';
	import { onMount, onDestroy } from 'svelte';
	import { browser } from '$app/environment';
	import { isElectron, isMacOS } from '$lib/utils/platform';
	import { onWorkspaceAppsUpdated } from '$lib/utils/workspaceAppsEvents';

	import MenuBar from '$lib/components/desktop/MenuBar.svelte';
	import DesktopIcon from '$lib/components/desktop/DesktopIcon.svelte';
	import Window from '$lib/components/desktop/Window.svelte';
	import Dock from '$lib/components/desktop/Dock.svelte';
	import SpotlightSearch from '$lib/components/desktop/SpotlightSearch.svelte';
	import IconPicker from '$lib/components/desktop/IconPicker.svelte';
	import AnimatedBackground from '$lib/components/desktop/AnimatedBackground.svelte';
	import Desktop3D from '$lib/components/desktop3d/Desktop3D.svelte';
	import OsaOrb from '$lib/components/osa/OsaOrb.svelte';
	import type { CustomIconConfig } from '$lib/stores/windowStore';

	import BootScreen from '$lib/components/window/BootScreen.svelte';
	import DesktopContextMenu from '$lib/components/window/DesktopContextMenu.svelte';
	import DesktopOnboarding from '$lib/components/window/DesktopOnboarding.svelte';
	import WindowContent from '$lib/components/window/WindowContent.svelte';

	const APP_VERSION = '0.0.1';
	const session = useSession();

	// Boot screen logic - show full loading on every visit
	let showBootScreen = $state(true);
	let bootComplete = $state(false);
	let workspaceDesktopApps = $state<WorkspaceApp[]>([]);
	let stopWorkspaceAppsListener: (() => void) | null = null;
	let workspaceAppsRefreshTimer: ReturnType<typeof setInterval> | null = null;
	let stopWorkspaceAppMessageListener: (() => void) | null = null;
	let stopDesktopSpaceUpdateListener: (() => void) | null = null;
	let desktopSpaceSyncTimer: ReturnType<typeof setTimeout> | null = null;
	let desktopSpaceRemoteApplyTimer: ReturnType<typeof setTimeout> | null = null;
	let canvasViewSaveTimer: ReturnType<typeof setTimeout> | null = null;
	let desktopSpacePollTimer: ReturnType<typeof setInterval> | null = null;
	let lastDesktopSpaceSyncKey = '';
	let lastDesktopSpaceLoadWorkspaceId = '';
	let lastAppliedRemoteDesktopSpaceKey = '';
	let applyingRemoteDesktopSpaces = $state(false);

	onMount(() => {
		// The desktop sits outside the (app) shell that restores the saved
		// workspace; restore it here so the 3D OS and workspace features follow it.
		if (!get(currentWorkspace)) loadSavedWorkspace().catch(() => {});
		// Initialize window store to load saved settings from localStorage
		windowStore.initialize();
		startDesktopPresence();

		// Start discovering deployed OSA apps and user-generated apps
		deployedAppsStore.startDiscovery();
		stopWorkspaceAppsListener = onWorkspaceAppsUpdated(loadWorkspaceDesktopApps);
		stopDesktopSpaceUpdateListener = onDesktopSpaceUpdated((event) => {
			if (event.workspaceId !== $currentWorkspace?.id || event.desktopId !== $windowStore.activeDesktopId) return;
			if (event.action === 'deleted') {
				windowStore.deleteDesktopSpace(event.desktopId);
				return;
			}
			void pullRemoteDesktopSpace(event.revision);
		});
		workspaceAppsRefreshTimer = setInterval(loadWorkspaceDesktopApps, 3000);
		loadWorkspaceDesktopApps();

		const handleWorkspaceAppMessage = async (event: MessageEvent) => {
			const allowedOrigins = new Set([
				window.location.origin,
				'http://localhost:5273',
				'http://127.0.0.1:5273'
			]);
			if (!allowedOrigins.has(event.origin)) return;
			const data = event.data as {
				type?: string;
				app?: {
					id?: string;
					name?: string;
					url?: string;
					launchMode?: 'iframe' | 'browser' | 'external';
					logoUrl?: string;
					color?: string;
					showOnDesktop?: boolean;
					showInDock?: boolean;
				};
			};
			if (data.type === 'businessos:workspace-apps-refresh') {
				await loadWorkspaceDesktopApps();
				return;
			}
			if ((data.type === 'businessos:workspace-app-installed' || data.type === 'businessos:workspace-app-upserted') && data.app?.id && data.app.url) {
				windowStore.placeWorkspaceApp({
					id: data.app.id,
					name: data.app.name || 'App',
					url: data.app.url,
					launch_mode: data.app.launchMode || 'iframe',
					icon: 'layout-grid',
					logo_url: data.app.logoUrl || '',
					color: data.app.color || '#111827',
					show_on_desktop: data.app.showOnDesktop ?? true,
					show_in_dock: data.app.showInDock ?? true
				});
				await loadWorkspaceDesktopApps();
				return;
			}
			if (data.type === 'businessos:workspace-app-removed' && data.app?.id) {
				const remaining = workspaceDesktopApps.filter((app) => app.id !== data.app?.id);
				workspaceDesktopApps = remaining;
				windowStore.syncWorkspaceApps(remaining);
				await loadWorkspaceDesktopApps();
				return;
			}
			if (data.type === 'businessos:open-workspace-app' && data.app?.id && data.app.url) {
				windowStore.openWindow(`workspace-app-${data.app.id}`, {
					title: data.app.name || 'App',
					data: {
						url: data.app.url,
						launchMode: data.app.launchMode || 'iframe'
					}
				});
				return;
			}
			if (data.type === 'businessos:preview-workspace-app' && data.app?.id && data.app.url) {
				windowStore.openWindow(`web-app-preview-${data.app.id}`, {
					title: data.app.name || 'App',
					data: {
						url: data.app.url,
						launchMode: data.app.launchMode || 'iframe'
					}
				});
			}
		};
		window.addEventListener('message', handleWorkspaceAppMessage);
		stopWorkspaceAppMessageListener = () =>
			window.removeEventListener('message', handleWorkspaceAppMessage);

		// Show loading screen for consistent duration (matches CSS animation)
		setTimeout(() => {
			showBootScreen = false;
			bootComplete = true;
		}, 1000); // 1 second for boot animation
	});

	onDestroy(() => {
		// Stop discovery when component unmounts
		deployedAppsStore.stopDiscovery();
		stopDesktopPresence();
		stopWorkspaceAppsListener?.();
		stopWorkspaceAppMessageListener?.();
		stopDesktopSpaceUpdateListener?.();
		if (workspaceAppsRefreshTimer) clearInterval(workspaceAppsRefreshTimer);
		if (desktopSpaceSyncTimer) clearTimeout(desktopSpaceSyncTimer);
		if (desktopSpaceRemoteApplyTimer) clearTimeout(desktopSpaceRemoteApplyTimer);
		if (canvasViewSaveTimer) clearTimeout(canvasViewSaveTimer);
		if (desktopSpacePollTimer) clearInterval(desktopSpacePollTimer);
	});

	$effect(() => {
		if (!$session.isPending && bootComplete) {
			showBootScreen = false;
		}
	});

	$effect(() => {
		if ($currentWorkspace?.id) {
			loadWorkspaceDesktopApps();
		}
	});

	$effect(() => {
		if (!browser) return;
		const saved = localStorage.getItem('businessos-dismissed-desktop-starters');
		if (!saved) return;
		try {
			const parsed = JSON.parse(saved);
			if (Array.isArray(parsed)) {
				dismissedDesktopStarters = parsed.filter((id) => typeof id === 'string');
			}
		} catch {
			dismissedDesktopStarters = [];
		}
	});

	function readSavedCanvasViews() {
		if (!browser) return {};
		try {
			const parsed = JSON.parse(localStorage.getItem(CANVAS_VIEW_STORAGE_KEY) || '{}');
			if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) return {};
			return parsed as Record<string, { zoom?: unknown; pan?: { x?: unknown; y?: unknown } }>;
		} catch {
			return {};
		}
	}

	function saveCanvasView() {
		if (!browser || !isCanvasDesktop) return;
		const views = readSavedCanvasViews();
		views[$windowStore.activeDesktopId] = {
			zoom: infinityZoom,
			pan: { x: infinityPan.x, y: infinityPan.y }
		};
		localStorage.setItem(CANVAS_VIEW_STORAGE_KEY, JSON.stringify(views));
	}

	function scheduleCanvasViewSave(delay = 180) {
		if (!browser || !isCanvasDesktop) return;
		if (canvasViewSaveTimer) clearTimeout(canvasViewSaveTimer);
		canvasViewSaveTimer = setTimeout(() => {
			canvasViewSaveTimer = null;
			saveCanvasView();
		}, delay);
	}

	function restoreCanvasView(desktopId: string) {
		if (!browser) return;
		const saved = readSavedCanvasViews()[desktopId];
		if (!saved) {
			resetInfinityView(false);
			return;
		}
		const zoom = typeof saved.zoom === 'number' ? saved.zoom : 1;
		const panX = typeof saved.pan?.x === 'number' ? saved.pan.x : 0;
		const panY = typeof saved.pan?.y === 'number' ? saved.pan.y : 0;
		infinityZoom = Number(clampCanvasZoom(zoom).toFixed(3));
		infinityPan = { x: panX, y: panY };
	}

	$effect(() => {
		if (!browser || !isCanvasDesktop) return;
		if (lastCanvasViewDesktopId !== $windowStore.activeDesktopId) {
			lastCanvasViewDesktopId = $windowStore.activeDesktopId;
			restoreCanvasView($windowStore.activeDesktopId);
		}
	});

	async function loadWorkspaceDesktopApps() {
		if (!$currentWorkspace?.id) {
			workspaceDesktopApps = [];
			return;
		}
		try {
			const res = await getApps(undefined, true);
			workspaceDesktopApps = res.apps;
			windowStore.syncWorkspaceApps(res.apps);
		} catch (err) {
			console.error('Failed to load workspace desktop apps:', err);
		}
	}

	// Onboarding state
	let showOnboarding = $state(false);
	let onboardingStep = $state(0);

	onMount(() => {
		const hasOnboarded = localStorage.getItem('businessos-onboarded');
		if (!hasOnboarded && !sessionStorage.getItem('businessos-booted')) {
			// Will show onboarding after boot completes
		}
	});

	$effect(() => {
		if (bootComplete && !showBootScreen) {
			const hasOnboarded = localStorage.getItem('businessos-onboarded');
			if (!hasOnboarded) {
				setTimeout(() => {
					showOnboarding = true;
				}, 500);
			}
		}
	});

	function completeOnboarding() {
		localStorage.setItem('businessos-onboarded', 'true');
		showOnboarding = false;
	}

	function nextOnboardingStep() {
		if (onboardingStep < 3) {
			onboardingStep++;
		} else {
			completeOnboarding();
		}
	}

	function skipOnboarding() {
		completeOnboarding();
	}

	// Detect Electron and macOS for traffic light handling
	const inElectron = $derived(browser && isElectron());
	const onMac = $derived(browser && isMacOS());
	const needsTrafficLightSpace = $derived(inElectron && onMac);
	// Menu bar height: 28px in Electron macOS, 26px otherwise
	const menuBarHeight = $derived(needsTrafficLightSpace ? 28 : 26);

	// Workspace dimensions (excluding menu bar and dock)
	let workspaceElement: HTMLDivElement | undefined = $state(undefined);
	let workspaceWidth = $state(0);
	let workspaceHeight = $state(0);

	// Grid settings for icons - dynamic based on icon size
	const ICON_PADDING = 16;
	const INFINITY_CANVAS_LIMIT = 50000;
	const CANVAS_MIN_ZOOM = 0.01;
	const CANVAS_MAX_ZOOM = 8;
	const CANVAS_FIT_PADDING = 96;
	const CANVAS_DEFAULT_CONTENT_WIDTH = 1400;
	const CANVAS_DEFAULT_CONTENT_HEIGHT = 900;
	const CANVAS_MIN_CURSOR_SCALE = 0.72;
	const CANVAS_MAX_CURSOR_SCALE = 24;
	const CANVAS_VIEW_STORAGE_KEY = 'businessos-canvas-views';
	// Grid size adjusts based on icon size to prevent overlap
	const GRID_SIZE = $derived(Math.max(96, $desktopSettings.iconSize + 40));
	const ICON_WIDTH = $derived(Math.max($desktopSettings.iconSize + 36, 90));
	const ICON_HEIGHT = $derived(Math.max($desktopSettings.iconSize + 50, 104));

	// Check if current background is dark (needs light text)
	const darkBackground = $derived(isBackgroundDark($desktopSettings.backgroundId));
	const activeDesktop = $derived(
		$windowStore.desktopSpaces.find((space) => space.id === $windowStore.activeDesktopId)
	);
	const activeDesktopName = $derived(activeDesktop?.name || 'Personal');
	const isInfinityDesktop = $derived(activeDesktopName.toLowerCase() === 'infinity desktop');
	const isSharedWorkspaceDesktop = $derived(
		(activeDesktop?.kind === 'workspace' || activeDesktop?.kind === 'team') && !isInfinityDesktop
	);
	// A shared workspace is still a bounded OS desktop. Only the explicitly
	// named Infinity Desktop gets unbounded pan and zoom canvas semantics.
	const isCanvasDesktop = $derived(isInfinityDesktop);
	const isWorkspaceDesktop = $derived(isUuid($windowStore.activeDesktopId));

	let infinityZoom = $state(1);
	let infinityPan = $state({ x: 0, y: 0 });
	let isPanningInfinity = $state(false);
	let isSpacePanningCanvas = $state(false);
	let infinityPanStart = $state({ x: 0, y: 0, panX: 0, panY: 0 });
	let dismissedDesktopStarters = $state<string[]>([]);
	let lastCanvasViewDesktopId = $state('');
	const canvasCursorScale = $derived(
		isCanvasDesktop && infinityZoom < CANVAS_MIN_CURSOR_SCALE
			? Number(Math.min(CANVAS_MAX_CURSOR_SCALE, CANVAS_MIN_CURSOR_SCALE / infinityZoom).toFixed(2))
			: 1
	);

	// Track icon positions (pixel-based for dragging)
	let iconPositions = $state<Record<string, { x: number; y: number }>>({});
	let lastActiveDesktopId = $state($windowStore.activeDesktopId);

	// Track which icon is being dragged
	let draggingIconId = $state<string | null>(null);

	// Selection box (lasso) state
	let isSelecting = $state(false);
	let selectionStart = $state({ x: 0, y: 0 });
	let selectionEnd = $state({ x: 0, y: 0 });
	let didSelectionDrag = $state(false);

	// Snap zone preview state
	let currentSnapZone = $state<SnapZone>(null);

	// Context menu state
	let showContextMenu = $state(false);
	let contextMenuPos = $state({ x: 0, y: 0 });
	let contextMenuType = $state<'desktop' | 'icon'>('desktop');
	let contextMenuIconId = $state<string | null>(null);

	// Rename state
	let renamingIconId = $state<string | null>(null);
	let renameValue = $state('');

	// Spotlight search state
	let showSpotlight = $state(false);

	// Icon picker state
	let showIconPicker = $state(false);
	let customizeIconId = $state<string | null>(null);
	let customizeIconCurrentConfig = $state<CustomIconConfig | undefined>(undefined);

	// Handler for icon customization
	function handleCustomizeIcon(iconId: string) {
		customizeIconId = iconId;
		const icon = $windowStore.desktopIcons.find(i => i.id === iconId);
		customizeIconCurrentConfig = icon?.customIcon;
		showIconPicker = true;
	}

	function handleIconPickerSelect(config: CustomIconConfig | undefined) {
		if (customizeIconId) {
			windowStore.updateIconCustomization(customizeIconId, config);
		}
		showIconPicker = false;
		customizeIconId = null;
		customizeIconCurrentConfig = undefined;
	}

	function handleIconPickerClose() {
		showIconPicker = false;
		customizeIconId = null;
		customizeIconCurrentConfig = undefined;
	}

	$effect(() => {
		const user = $session.data?.user;
		const activeWindow = $focusedWindow;
		setDesktopPresenceContext($windowStore.activeDesktopId, {
			userId: user?.id || 'local-user',
			name: user?.name || user?.email || 'Teammate',
			activeModule: activeWindow?.module,
			activeTitle: activeWindow?.title
		});
		connectDesktopPresenceRemote($currentWorkspace?.id || null, $windowStore.activeDesktopId);
	});

	$effect(() => {
		windowStore.updateActiveDesktopSettings($desktopSettings);
	});

	$effect(() => {
		if (!isCanvasDesktop || !$followedDesktopCursor || workspaceWidth <= 0 || workspaceHeight <= 0) return;
		const cursor = $desktopPresenceStore.find((item) => item.clientId === $followedDesktopCursor);
		if (!cursor) {
			followDesktopCursor(null);
			return;
		}
		infinityPan = {
			x: Math.round(workspaceWidth / 2 - cursor.x * infinityZoom),
			y: Math.round(workspaceHeight / 2 - cursor.y * infinityZoom)
		};
		scheduleCanvasViewSave();
	});

	function isUuid(value: string) {
		return /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(value);
	}

	function currentDesktopSpaceSnapshot(): DesktopSpace {
		const existing = $windowStore.desktopSpaces.find((space) => space.id === $windowStore.activeDesktopId);
		return {
			id: $windowStore.activeDesktopId,
			name: existing?.name || 'Desktop',
			kind: existing?.kind === 'team' || existing?.kind === 'workspace' ? existing.kind : 'personal',
			desktopSettings: $desktopSettings as unknown as Record<string, unknown>,
			desktopIcons: $windowStore.desktopIcons,
			dockPinnedItems: $windowStore.dockPinnedItems,
			folders: $windowStore.folders,
			windows: $windowStore.windows,
			windowOrder: $windowStore.windowOrder,
			focusedWindowId: $windowStore.focusedWindowId,
			createdAt: existing?.createdAt || new Date().toISOString(),
			updatedAt: existing?.updatedAt || new Date(0).toISOString()
		};
	}

	async function pullRemoteDesktopSpace(revision?: string) {
		const workspaceId = $currentWorkspace?.id;
		const desktopId = $windowStore.activeDesktopId;
		if (
			!workspaceId ||
			!isUuid(desktopId) ||
			applyingRemoteDesktopSpaces ||
			draggingIconId ||
			isPanningInfinity ||
			isSelecting ||
			$followedDesktopCursor
		) return;
		try {
			const res = await listWorkspaceDesktopSpaces(workspaceId);
			const remote = res.spaces.find((record) => record.config?.id === desktopId)?.config;
			if (!remote) return;
			const remoteKey = `${workspaceId}:${desktopId}:${revision || remote.updatedAt}:${JSON.stringify(remote)}`;
			const localKey = `${workspaceId}:${JSON.stringify(currentDesktopSpaceSnapshot())}`;
			if (remoteKey === lastAppliedRemoteDesktopSpaceKey || localKey === `${workspaceId}:${JSON.stringify(remote)}`) return;
			applyingRemoteDesktopSpaces = true;
			windowStore.applyRemoteDesktopSpace(remote);
			lastAppliedRemoteDesktopSpaceKey = remoteKey;
			lastDesktopSpaceSyncKey = `${workspaceId}:${JSON.stringify(remote)}`;
		} catch {
			// The periodic pull remains as a fallback.
		} finally {
			if (desktopSpaceRemoteApplyTimer) clearTimeout(desktopSpaceRemoteApplyTimer);
			desktopSpaceRemoteApplyTimer = setTimeout(() => {
				applyingRemoteDesktopSpaces = false;
			}, 250);
		}
	}

	$effect(() => {
		const workspaceId = $currentWorkspace?.id;
		if (!workspaceId || lastDesktopSpaceLoadWorkspaceId === workspaceId) return;
		lastDesktopSpaceLoadWorkspaceId = workspaceId;
		listWorkspaceDesktopSpaces(workspaceId)
			.then((res) => {
				const spaces = res.spaces
					.map((record) => record.config)
					.filter((space): space is DesktopSpace => Boolean(space?.id && space?.name));
				if (spaces.length > 0) {
					applyingRemoteDesktopSpaces = true;
					windowStore.mergeDesktopSpaces(spaces);
					if (desktopSpaceRemoteApplyTimer) clearTimeout(desktopSpaceRemoteApplyTimer);
					desktopSpaceRemoteApplyTimer = setTimeout(() => {
						if (isUuid($windowStore.activeDesktopId)) {
							lastDesktopSpaceSyncKey = `${workspaceId}:${JSON.stringify(currentDesktopSpaceSnapshot())}`;
						}
						applyingRemoteDesktopSpaces = false;
					}, 250);
				}
			})
			.catch(() => {
				applyingRemoteDesktopSpaces = false;
				lastDesktopSpaceLoadWorkspaceId = '';
			});
	});

	$effect(() => {
		const workspaceId = $currentWorkspace?.id;
		if (!workspaceId || applyingRemoteDesktopSpaces || !isUuid($windowStore.activeDesktopId)) return;
		const snapshot = currentDesktopSpaceSnapshot();
		const syncKey = `${workspaceId}:${JSON.stringify(snapshot)}`;
		if (syncKey === lastDesktopSpaceSyncKey) return;
		lastDesktopSpaceSyncKey = syncKey;
		if (desktopSpaceSyncTimer) clearTimeout(desktopSpaceSyncTimer);
		desktopSpaceSyncTimer = setTimeout(() => {
			saveWorkspaceDesktopSpace(workspaceId, snapshot)
				.then((record) => {
					publishDesktopSpaceUpdated(snapshot.id, record.updated_at);
				})
				.catch(() => {
					lastDesktopSpaceSyncKey = '';
				});
		}, 900);
	});

	$effect(() => {
		const workspaceId = $currentWorkspace?.id;
		const desktopId = $windowStore.activeDesktopId;
		if (desktopSpacePollTimer) {
			clearInterval(desktopSpacePollTimer);
			desktopSpacePollTimer = null;
		}
		if (!workspaceId || !isUuid(desktopId)) return;

		desktopSpacePollTimer = setInterval(() => void pullRemoteDesktopSpace(), 8000);
		void pullRemoteDesktopSpace();

		return () => {
			if (desktopSpacePollTimer) {
				clearInterval(desktopSpacePollTimer);
				desktopSpacePollTimer = null;
			}
		};
	});

	// Only show icons that are NOT inside a folder
	const visibleDesktopIcons = $derived(
		$windowStore.desktopIcons.filter(icon => !icon.folderId || icon.type === 'folder')
	);
	const showDesktopStarter = $derived(
		isWorkspaceDesktop &&
			visibleDesktopIcons.length === 0 &&
			$visibleWindows.length === 0 &&
			!dismissedDesktopStarters.includes($windowStore.activeDesktopId)
	);

	// Selection box computed bounds
	const selectionBox = $derived(() => {
		if (!isSelecting) return null;
		return {
			x: Math.min(selectionStart.x, selectionEnd.x),
			y: Math.min(selectionStart.y, selectionEnd.y),
			width: Math.abs(selectionEnd.x - selectionStart.x),
			height: Math.abs(selectionEnd.y - selectionStart.y)
		};
	});

	// Snap zone preview bounds
	const snapZonePreview = $derived(() => {
		if (!currentSnapZone || workspaceWidth === 0 || workspaceHeight === 0) return null;

		switch (currentSnapZone) {
			case 'left':
				return { x: 0, y: 0, width: workspaceWidth / 2, height: workspaceHeight };
			case 'right':
				return { x: workspaceWidth / 2, y: 0, width: workspaceWidth / 2, height: workspaceHeight };
			case 'top-left':
				return { x: 0, y: 0, width: workspaceWidth / 2, height: workspaceHeight / 2 };
			case 'top-right':
				return { x: workspaceWidth / 2, y: 0, width: workspaceWidth / 2, height: workspaceHeight / 2 };
			case 'bottom-left':
				return { x: 0, y: workspaceHeight / 2, width: workspaceWidth / 2, height: workspaceHeight / 2 };
			case 'bottom-right':
				return { x: workspaceWidth / 2, y: workspaceHeight / 2, width: workspaceWidth / 2, height: workspaceHeight / 2 };
			default:
				return null;
		}
	});

	// Handle snap zone change from window dragging
	function handleSnapZoneChange(zone: SnapZone) {
		currentSnapZone = zone;
	}

	$effect(() => {
		if (!$session.isPending && !$session.data) {
			goto('/login');
		}
	});

	// Update workspace dimensions when element is available
	$effect(() => {
		if (!workspaceElement) return;
		const clampActiveDesktopToViewport = (width: number, height: number) => {
			if (isCanvasDesktop) return;
			windowStore.clampActiveDesktopToViewport(width, height, {
				iconWidth: ICON_WIDTH,
				iconHeight: ICON_HEIGHT
			});
			iconPositions = {};
		};

		const measureDimensions = () => {
			if (workspaceElement) {
				const width = workspaceElement.clientWidth;
				const height = workspaceElement.clientHeight;
				if (width > 0 && height > 0) {
					workspaceWidth = width;
					workspaceHeight = height;
					clampActiveDesktopToViewport(width, height);
				}
			}
		};

		measureDimensions();

		const resizeObserver = new ResizeObserver((entries) => {
			for (const entry of entries) {
				workspaceWidth = entry.contentRect.width;
				workspaceHeight = entry.contentRect.height;
				clampActiveDesktopToViewport(entry.contentRect.width, entry.contentRect.height);
			}
		});
		resizeObserver.observe(workspaceElement);

		const handleResize = () => measureDimensions();
		window.addEventListener('resize', handleResize);

		requestAnimationFrame(measureDimensions);

		const delayedMeasure = setTimeout(measureDimensions, 100);

		return () => {
			resizeObserver.disconnect();
			window.removeEventListener('resize', handleResize);
			clearTimeout(delayedMeasure);
		};
	});

	// Re-measure when boot completes and session is ready
	$effect(() => {
		if (bootComplete && $session.data && workspaceElement) {
			requestAnimationFrame(() => {
				if (workspaceElement) {
					const width = workspaceElement.clientWidth;
					const height = workspaceElement.clientHeight;
					if (width > 0 && height > 0) {
						workspaceWidth = width;
						workspaceHeight = height;
					}
				}
			});
		}
	});

	// Get background style
	const backgroundStyle = $derived(() => {
		const bgCSS = getBackgroundCSS($desktopSettings.backgroundId, $desktopSettings.customBackgroundUrl);
		const isCustomImage = $desktopSettings.backgroundId === 'custom';

		if (isCustomImage && $desktopSettings.customBackgroundUrl) {
			const fitMap: Record<string, string> = {
				'cover': 'cover',
				'contain': 'contain',
				'fill': '100% 100%',
				'center': 'auto'
			};
			const bgSize = fitMap[$desktopSettings.backgroundFit] || 'cover';

			return `
				background-image: ${bgCSS.background};
				background-size: ${bgSize};
				background-position: center center;
				background-repeat: no-repeat;
				background-attachment: fixed;
				background-color: var(--dbg);
			`;
		} else if (bgCSS.backgroundSize) {
			return `background: ${bgCSS.background}; background-size: ${bgCSS.backgroundSize};`;
		}
		return `background: ${bgCSS.background};`;
	});

	onMount(() => {
		function updateDimensions() {
			if (workspaceElement) {
				workspaceWidth = workspaceElement.clientWidth;
				workspaceHeight = workspaceElement.clientHeight;
			}
		}

		window.addEventListener('resize', updateDimensions);

		function handleKeyDown(event: KeyboardEvent) {
			const activeElement = document.activeElement;
			if (activeElement?.tagName === 'IFRAME' ||
				activeElement?.tagName === 'INPUT' ||
				activeElement?.tagName === 'TEXTAREA') {
				return;
			}

			const isMeta = event.metaKey || event.ctrlKey;
			const isShift = event.shiftKey;
			const isCtrlAlt = event.ctrlKey && event.altKey;

			if (isCanvasDesktop && event.code === 'Space' && !isMeta) {
				event.preventDefault();
				isSpacePanningCanvas = true;
			} else if (isMeta && event.key === ' ') {
				event.preventDefault();
				showSpotlight = true;
			} else if (isMeta && event.key === 'w') {
				event.preventDefault();
				if ($focusedWindow) windowStore.closeWindow($focusedWindow.id);
			} else if (isMeta && event.key === 'm') {
				event.preventDefault();
				if ($focusedWindow) windowStore.minimizeWindow($focusedWindow.id);
			} else if (isMeta && isShift && event.key === 'F') {
				event.preventDefault();
				if (!isCanvasDesktop && $focusedWindow) windowStore.toggleMaximize($focusedWindow.id);
			} else if (isMeta && event.key === '`' && !isShift) {
				event.preventDefault();
				windowStore.cycleWindows();
			} else if (isMeta && isShift && event.key === '`') {
				event.preventDefault();
				openTerminalWindow();
			} else if (isCtrlAlt && event.key === 'ArrowLeft') {
				event.preventDefault();
				if (!isCanvasDesktop && $focusedWindow) windowStore.snapWindow($focusedWindow.id, 'left', workspaceWidth, workspaceHeight);
			} else if (isCtrlAlt && event.key === 'ArrowRight') {
				event.preventDefault();
				if (!isCanvasDesktop && $focusedWindow) windowStore.snapWindow($focusedWindow.id, 'right', workspaceWidth, workspaceHeight);
			} else if (isMeta && isShift && event.key === 'T') {
				event.preventDefault();
				windowStore.openWindow('tasks');
			} else if (isMeta && isShift && event.key === 'P') {
				event.preventDefault();
				windowStore.openWindow('projects');
			} else if (isMeta && isShift && event.key === 'N') {
				event.preventDefault();
				windowStore.openWindow('contexts');
			} else if (isMeta && event.key === '1') {
				event.preventDefault();
				windowStore.openWindow('dashboard');
			} else if (isMeta && event.key === '2') {
				event.preventDefault();
				windowStore.openWindow('chat');
			} else if (isMeta && event.key === '3') {
				event.preventDefault();
				windowStore.openWindow('tasks');
			} else if (isMeta && event.key === '4') {
				event.preventDefault();
				windowStore.openWindow('calendar');
			} else if (isMeta && event.key === '5') {
				event.preventDefault();
				windowStore.openWindow('projects');
			} else if (event.key === 'Escape') {
				windowStore.clearIconSelection();
				showSpotlight = false;
			}
		}

		function handleKeyUp(event: KeyboardEvent) {
			if (event.code === 'Space') {
				isSpacePanningCanvas = false;
			}
		}

		window.addEventListener('keydown', handleKeyDown);
		window.addEventListener('keyup', handleKeyUp);

		return () => {
			window.removeEventListener('resize', updateDimensions);
			window.removeEventListener('keydown', handleKeyDown);
			window.removeEventListener('keyup', handleKeyUp);
		};
	});

	function clampCanvasCoordinate(value: number) {
		return Math.max(-INFINITY_CANVAS_LIMIT, Math.min(value, INFINITY_CANVAS_LIMIT));
	}

	function clampIconPosition(x: number, y: number) {
		if (isCanvasDesktop) {
			return {
				x: clampCanvasCoordinate(x),
				y: clampCanvasCoordinate(y)
			};
		}
		const maxX = Math.max(0, workspaceWidth - ICON_WIDTH);
		const maxY = Math.max(0, workspaceHeight - ICON_HEIGHT);
		return {
			x: Math.max(0, Math.min(x, maxX)),
			y: Math.max(0, Math.min(y, maxY))
		};
	}

		// Calculate icon positions - use stored pixel position or calculate from grid
		function getIconPosition(icon: { id: string; x: number; y: number; positionMode?: 'grid' | 'pixel' }) {
			if (iconPositions[icon.id]) {
				const pos = iconPositions[icon.id];
				return clampIconPosition(pos.x, pos.y);
			}

		let x: number;
		let y: number;

		if (icon.positionMode === 'pixel') {
			x = icon.x;
		} else if (icon.x < 0) {
			const layoutWidth = isCanvasDesktop ? CANVAS_DEFAULT_CONTENT_WIDTH : workspaceWidth;
			x = layoutWidth + (icon.x * GRID_SIZE) - ICON_PADDING;
		} else {
			x = icon.x * GRID_SIZE + ICON_PADDING;
		}

		if (icon.positionMode === 'pixel') {
			y = icon.y;
		} else if (icon.y < 0) {
			const layoutHeight = isCanvasDesktop ? CANVAS_DEFAULT_CONTENT_HEIGHT : workspaceHeight;
			y = layoutHeight + (icon.y * GRID_SIZE) - ICON_PADDING;
		} else {
			y = icon.y * GRID_SIZE + ICON_PADDING;
		}

			return clampIconPosition(x, y);
		}

		$effect(() => {
			if (workspaceWidth <= 0 || workspaceHeight <= 0) return;
			if (lastActiveDesktopId !== $windowStore.activeDesktopId) {
				lastActiveDesktopId = $windowStore.activeDesktopId;
				iconPositions = {};
				return;
			}
			const nextPositions: Record<string, { x: number; y: number }> = {};
			let changed = false;

			for (const [iconId, pos] of Object.entries(iconPositions)) {
				const next = clampIconPosition(pos.x, pos.y);
				nextPositions[iconId] = next;
				if (next.x !== pos.x || next.y !== pos.y) changed = true;
			}

			if (changed) {
				iconPositions = nextPositions;
				for (const [iconId, pos] of Object.entries(nextPositions)) {
					windowStore.updateIconPosition(iconId, pos.x, pos.y);
				}
			}
		});

		$effect(() => {
			if (workspaceWidth <= 0 || workspaceHeight <= 0) return;

			for (const icon of visibleDesktopIcons) {
				if (icon.positionMode !== 'pixel') continue;
				const constrained = clampIconPosition(icon.x, icon.y);
				if (constrained.x !== icon.x || constrained.y !== icon.y) {
					windowStore.updateIconPosition(icon.id, constrained.x, constrained.y);
					iconPositions = {
						...iconPositions,
						[icon.id]: constrained
					};
				}
			}
		});

	function handleIconDragStart(iconId: string) {
		draggingIconId = iconId;
	}

	function handleIconDragMove(iconId: string, newX: number, newY: number) {
		const constrained = clampIconPosition(newX, newY);

		iconPositions = {
			...iconPositions,
			[iconId]: constrained
		};
	}

	function handleDesktopClick(event: MouseEvent) {
		if (didSelectionDrag) {
			didSelectionDrag = false;
			return;
		}
		const target = event.target as HTMLElement;
		const isDesktopOrBackground = target.classList.contains('desktop-workspace') ||
			target.classList.contains('animated-background') ||
			target.tagName === 'CANVAS';
		const isNotIconOrWindow = !target.closest('.desktop-icon') &&
			!target.closest('.window') &&
			!target.closest('.context-menu');

		if (isDesktopOrBackground || (target.closest('.desktop-workspace') && isNotIconOrWindow)) {
			windowStore.clearIconSelection();
		}
	}

	function handleDesktopMouseMove(event: MouseEvent) {
		if (!workspaceElement) return;
		const rect = workspaceElement.getBoundingClientRect();
		if (isPanningInfinity) {
			return;
		}
		if (isCanvasDesktop) {
			const canvasX = (event.clientX - rect.left - infinityPan.x) / infinityZoom;
			const canvasY = (event.clientY - rect.top - infinityPan.y) / infinityZoom;
			publishDesktopCursor(
				$windowStore.activeDesktopId,
				canvasX,
				canvasY,
				INFINITY_CANVAS_LIMIT * 2,
				INFINITY_CANVAS_LIMIT * 2
			);
			return;
		}
		const x = Math.max(0, Math.min(event.clientX - rect.left, workspaceWidth));
		const y = Math.max(0, Math.min(event.clientY - rect.top, workspaceHeight));
		publishDesktopCursor($windowStore.activeDesktopId, x, y, workspaceWidth, workspaceHeight);
	}

	function handleInfinityWheel(event: WheelEvent) {
		if (!isCanvasDesktop) return;
		event.preventDefault();
		const delta = canvasWheelDelta(event);
		if (event.metaKey || event.ctrlKey || event.altKey) {
			const { viewportX, viewportY } = canvasPointFromClient(event.clientX, event.clientY);
			setCanvasZoomAt(infinityZoom * Math.exp(-delta.y * 0.0018), viewportX, viewportY);
			return;
		}
		infinityPan = {
			x: infinityPan.x - (event.shiftKey ? delta.y : delta.x),
			y: infinityPan.y - (event.shiftKey ? 0 : delta.y)
		};
		scheduleCanvasViewSave();
	}

	function clampCanvasZoom(value: number) {
		return Math.max(CANVAS_MIN_ZOOM, Math.min(CANVAS_MAX_ZOOM, value));
	}

	function canvasPointFromClient(clientX: number, clientY: number) {
		const rect = workspaceElement?.getBoundingClientRect();
		const viewportX = rect ? clientX - rect.left : workspaceWidth / 2;
		const viewportY = rect ? clientY - rect.top : workspaceHeight / 2;
		return {
			viewportX,
			viewportY,
			canvasX: (viewportX - infinityPan.x) / infinityZoom,
			canvasY: (viewportY - infinityPan.y) / infinityZoom
		};
	}

	function canvasWheelDelta(event: WheelEvent) {
		const unit = event.deltaMode === WheelEvent.DOM_DELTA_LINE
			? 16
			: event.deltaMode === WheelEvent.DOM_DELTA_PAGE
				? 600
				: 1;
		return {
			x: event.deltaX * unit,
			y: event.deltaY * unit
		};
	}

	function setCanvasZoomAt(nextZoomValue: number, viewportX = workspaceWidth / 2, viewportY = workspaceHeight / 2) {
		const nextZoom = Number(clampCanvasZoom(nextZoomValue).toFixed(3));
		const anchorCanvasX = (viewportX - infinityPan.x) / infinityZoom;
		const anchorCanvasY = (viewportY - infinityPan.y) / infinityZoom;
		infinityZoom = nextZoom;
		infinityPan = {
			x: viewportX - anchorCanvasX * nextZoom,
			y: viewportY - anchorCanvasY * nextZoom
		};
		scheduleCanvasViewSave();
	}

	function updateCanvasPan(event: MouseEvent) {
		if (!workspaceElement) return;
		const rect = workspaceElement.getBoundingClientRect();
		const nextPan = {
			x: infinityPanStart.panX + event.clientX - infinityPanStart.x,
			y: infinityPanStart.panY + event.clientY - infinityPanStart.y
		};
		infinityPan = nextPan;
		scheduleCanvasViewSave();
		const canvasX = (event.clientX - rect.left - nextPan.x) / infinityZoom;
		const canvasY = (event.clientY - rect.top - nextPan.y) / infinityZoom;
		publishDesktopCursor(
			$windowStore.activeDesktopId,
			canvasX,
			canvasY,
			INFINITY_CANVAS_LIMIT * 2,
			INFINITY_CANVAS_LIMIT * 2
		);
	}

	function startInfinityPan(event: MouseEvent) {
		const target = event.target as HTMLElement;
		const canStartPan = event.button === 1 || event.button === 0;
		const shouldForcePan = isSpacePanningCanvas && event.button === 0;
		if (
			!isCanvasDesktop ||
			!canStartPan ||
			(!shouldForcePan && target.closest('button, input, textarea, select, .window, .desktop-icon, .desktop-starter, .infinity-toolbar'))
		) return false;
		event.preventDefault();
		isPanningInfinity = true;
		infinityPanStart = {
			x: event.clientX,
			y: event.clientY,
			panX: infinityPan.x,
			panY: infinityPan.y
		};
		document.addEventListener('mousemove', updateCanvasPan);
		document.addEventListener('mouseup', stopInfinityPan);
		window.addEventListener('blur', stopInfinityPan);
		return true;
	}

	function stopInfinityPan() {
		isPanningInfinity = false;
		isSpacePanningCanvas = false;
		document.removeEventListener('mousemove', updateCanvasPan);
		document.removeEventListener('mouseup', stopInfinityPan);
		window.removeEventListener('blur', stopInfinityPan);
		saveCanvasView();
	}

	function resetInfinityView(shouldSave = true) {
		infinityZoom = 1;
		infinityPan = {
			x: Math.round(workspaceWidth / 2 - CANVAS_DEFAULT_CONTENT_WIDTH / 2),
			y: Math.round(workspaceHeight / 2 - CANVAS_DEFAULT_CONTENT_HEIGHT / 2)
		};
		if (shouldSave) saveCanvasView();
	}

	function zoomInfinity(delta: number) {
		const factor = delta > 0 ? 1.2 : 1 / 1.2;
		setCanvasZoomAt(infinityZoom * factor);
	}

	function fitInfinityView() {
		const bounds = getCanvasContentBounds();
		const contentWidth = Math.max(1, bounds.maxX - bounds.minX);
		const contentHeight = Math.max(1, bounds.maxY - bounds.minY);
		const availableWidth = Math.max(320, workspaceWidth - CANVAS_FIT_PADDING * 2);
		const availableHeight = Math.max(240, workspaceHeight - CANVAS_FIT_PADDING * 2);
		const nextZoom = clampCanvasZoom(Math.min(availableWidth / contentWidth, availableHeight / contentHeight, 1));
		infinityZoom = Number(nextZoom.toFixed(3));
		infinityPan = {
			x: Math.round((workspaceWidth - contentWidth * infinityZoom) / 2 - bounds.minX * infinityZoom),
			y: Math.round((workspaceHeight - contentHeight * infinityZoom) / 2 - bounds.minY * infinityZoom)
		};
		saveCanvasView();
	}

	function centeredWindowPosition(width: number, height: number) {
		if (!isCanvasDesktop || workspaceWidth <= 0 || workspaceHeight <= 0) return {};
		return {
			x: Math.round((workspaceWidth / 2 - infinityPan.x) / infinityZoom - width / 2),
			y: Math.round((workspaceHeight / 2 - infinityPan.y) / infinityZoom - height / 2)
		};
	}

	function visibleCanvasWindowBounds() {
		if (!isCanvasDesktop || workspaceWidth <= 0 || workspaceHeight <= 0) return undefined;
		const margin = 18;
		return {
			x: Math.round((margin - infinityPan.x) / infinityZoom),
			y: Math.round((margin - infinityPan.y) / infinityZoom),
			width: Math.max(320, Math.round((workspaceWidth - margin * 2) / infinityZoom)),
			height: Math.max(220, Math.round((workspaceHeight - margin * 2) / infinityZoom))
		};
	}

	function openDesktopWindow(module: string, width: number, height: number, options: { title?: string; data?: Record<string, unknown> } = {}) {
		windowStore.openWindow(module, {
			...options,
			...centeredWindowPosition(width, height)
		});
	}

	function getCanvasContentBounds() {
		const bounds: Array<{ x: number; y: number; width: number; height: number }> = [];
		for (const icon of visibleDesktopIcons) {
			const pos = getIconPosition(icon);
			bounds.push({ x: pos.x, y: pos.y, width: ICON_WIDTH, height: ICON_HEIGHT });
		}
		for (const win of $visibleWindows) {
			if (win.minimized) continue;
			bounds.push({ x: win.x, y: win.y, width: win.width, height: win.height });
		}
		if (bounds.length === 0) {
			return {
				minX: 0,
				minY: 0,
				maxX: CANVAS_DEFAULT_CONTENT_WIDTH,
				maxY: CANVAS_DEFAULT_CONTENT_HEIGHT
			};
		}
		return bounds.reduce(
			(acc, item) => ({
				minX: Math.min(acc.minX, item.x),
				minY: Math.min(acc.minY, item.y),
				maxX: Math.max(acc.maxX, item.x + item.width),
				maxY: Math.max(acc.maxY, item.y + item.height)
			}),
			{
				minX: Number.POSITIVE_INFINITY,
				minY: Number.POSITIVE_INFINITY,
				maxX: Number.NEGATIVE_INFINITY,
				maxY: Number.NEGATIVE_INFINITY
			}
		);
	}

	function openAppsWindow() {
		openDesktopWindow('apps', 1050, 740);
	}

	function openCanvasNote() {
		const id = `canvas-note-${Date.now()}`;
		openDesktopWindow(id, 320, 240, {
			title: 'Note',
			data: { text: '' }
		});
	}

	function openTerminalWindow() {
		openDesktopWindow('terminal', 700, 500);
	}

	function dismissDesktopStarter() {
		const next = [...new Set([...dismissedDesktopStarters, $windowStore.activeDesktopId])];
		dismissedDesktopStarters = next;
		if (browser) {
			localStorage.setItem('businessos-dismissed-desktop-starters', JSON.stringify(next));
		}
	}

	function toggleFollowCursor(cursor: DesktopPresenceCursor) {
		const nextCursorId = $followedDesktopCursor === cursor.clientId ? null : cursor.clientId;
		followDesktopCursor(nextCursorId);
		if (nextCursorId && isCanvasDesktop) {
			infinityPan = {
				x: Math.round(workspaceWidth / 2 - cursor.x * infinityZoom),
				y: Math.round(workspaceHeight / 2 - cursor.y * infinityZoom)
			};
		}
	}

	function getPresenceCursorPosition(cursor: { x: number; y: number; viewportWidth: number; viewportHeight: number }) {
		if (isCanvasDesktop) {
			return {
				x: cursor.x,
				y: cursor.y
			};
		}
		const scaledX = cursor.viewportWidth > 0 ? (cursor.x / cursor.viewportWidth) * workspaceWidth : cursor.x;
		const scaledY = cursor.viewportHeight > 0 ? (cursor.y / cursor.viewportHeight) * workspaceHeight : cursor.y;
		return {
			x: Math.max(0, Math.min(scaledX, workspaceWidth)),
			y: Math.max(0, Math.min(scaledY, workspaceHeight))
		};
	}

	// Selection box handlers
	function handleDesktopMouseDown(event: MouseEvent) {
		if (startInfinityPan(event)) return;
		if (event.button !== 0) return;
		const target = event.target as HTMLElement;
		if (!target.closest('.desktop-workspace')) return;
		if (target.closest('.desktop-icon, .window, .context-menu, .desktop-starter, .infinity-toolbar')) return;
		if (!workspaceElement) return;

		const rect = workspaceElement.getBoundingClientRect();
		const point = canvasPointFromClient(event.clientX, event.clientY);
		const x = isCanvasDesktop ? point.canvasX : event.clientX - rect.left;
		const y = isCanvasDesktop ? point.canvasY : event.clientY - rect.top;

		selectionStart = { x, y };
		selectionEnd = { x, y };
		isSelecting = true;
		didSelectionDrag = false;

		if (!event.shiftKey) {
			windowStore.clearIconSelection();
		}

		document.addEventListener('mousemove', handleSelectionMove);
		document.addEventListener('mouseup', handleSelectionEnd);
	}

	function handleSelectionMove(event: MouseEvent) {
		if (!isSelecting || !workspaceElement) return;

		const rect = workspaceElement.getBoundingClientRect();
		const point = canvasPointFromClient(event.clientX, event.clientY);
		const x = isCanvasDesktop ? point.canvasX : Math.max(0, Math.min(event.clientX - rect.left, workspaceWidth));
		const y = isCanvasDesktop ? point.canvasY : Math.max(0, Math.min(event.clientY - rect.top, workspaceHeight));

		selectionEnd = { x, y };

		const box = selectionBox();
		if (box && box.width > 5 && box.height > 5) {
			didSelectionDrag = true;
			selectIconsInBox(box);
		}
	}

	function handleSelectionEnd() {
		isSelecting = false;
		document.removeEventListener('mousemove', handleSelectionMove);
		document.removeEventListener('mouseup', handleSelectionEnd);
	}

	function selectIconsInBox(box: { x: number; y: number; width: number; height: number }) {
		const selectedIds: string[] = [];

		for (const icon of $windowStore.desktopIcons) {
			const pos = getIconPosition(icon);
			const iconWidth = ICON_WIDTH;
			const iconHeight = ICON_HEIGHT;

			const iconRight = pos.x + iconWidth;
			const iconBottom = pos.y + iconHeight;
			const boxRight = box.x + box.width;
			const boxBottom = box.y + box.height;

			if (
				pos.x < boxRight &&
				iconRight > box.x &&
				pos.y < boxBottom &&
				iconBottom > box.y
			) {
				selectedIds.push(icon.id);
			}
		}

		if (selectedIds.length > 0) {
			windowStore.setSelectedIcons(selectedIds);
		}
	}

	function handleIconSelect(iconId: string, additive: boolean) {
		windowStore.selectIcon(iconId, additive);
	}

	function handleIconOpen(module: string) {
		const icon = $windowStore.desktopIcons.find((item) => item.module === module);
		if (module.startsWith('workspace-app-') && icon?.appUrl) {
			windowStore.openWindow(module, {
				title: icon.label,
				data: { url: icon.appUrl, launchMode: icon.launchMode || 'iframe' }
			});
			return;
		}
		windowStore.openWindow(module);
	}

	function handleIconDragEnd(iconId: string, finalX: number, finalY: number) {
		const folderIcon = visibleDesktopIcons.find(icon => {
			if (icon.type !== 'folder' || icon.id === iconId) return false;
			const folderPos = getIconPosition(icon);
			const inX = finalX >= folderPos.x - 20 && finalX <= folderPos.x + 80;
			const inY = finalY >= folderPos.y - 20 && finalY <= folderPos.y + 100;
			return inX && inY;
		});

		if (folderIcon && folderIcon.folderId) {
			windowStore.moveIconToFolder(iconId, folderIcon.folderId);
			const { [iconId]: _, ...rest } = iconPositions;
			iconPositions = rest;
		} else {
			const constrained = clampIconPosition(finalX, finalY);

			iconPositions = {
				...iconPositions,
				[iconId]: constrained
			};
			windowStore.updateIconPosition(iconId, constrained.x, constrained.y, {
				width: workspaceWidth,
				height: workspaceHeight,
				iconWidth: ICON_WIDTH,
				iconHeight: ICON_HEIGHT
			});
		}
		draggingIconId = null;
	}

	// Get z-index for a window based on its position in windowOrder
	function getWindowZIndex(windowId: string): number {
		const index = $windowStore.windowOrder.indexOf(windowId);
		return 100 + index;
	}

	// Context menu handlers
	function handleContextMenu(event: MouseEvent) {
		const target = event.target as HTMLElement;
		if (!target.closest('.desktop-workspace')) return;
		if (target.closest('.desktop-icon, .window, .context-menu, .desktop-starter, .infinity-toolbar')) return;

		event.preventDefault();
		contextMenuPos = { x: event.clientX, y: event.clientY };
		contextMenuType = 'desktop';
		contextMenuIconId = null;
		showContextMenu = true;
	}

	function handleIconContextMenu(event: MouseEvent, iconId: string) {
		event.preventDefault();
		event.stopPropagation();
		contextMenuPos = { x: event.clientX, y: event.clientY };
		contextMenuType = 'icon';
		contextMenuIconId = iconId;
		showContextMenu = true;
	}

	function closeContextMenu() {
		showContextMenu = false;
		contextMenuIconId = null;
	}

	function createNewFolder() {
		const point = isCanvasDesktop ? canvasPointFromClient(contextMenuPos.x, contextMenuPos.y) : null;
		const relativeX = point ? point.canvasX : contextMenuPos.x;
		const relativeY = point ? point.canvasY : contextMenuPos.y - menuBarHeight;

		const gridX = Math.floor(relativeX / GRID_SIZE);
		const gridY = Math.floor(relativeY / GRID_SIZE);

		windowStore.createFolder('New Folder', gridX, gridY);
		closeContextMenu();
	}

	function openDesktopSettings() {
		openDesktopWindow('desktop-settings', 550, 500);
		closeContextMenu();
	}

	function openAppsFromContextMenu() {
		openAppsWindow();
		closeContextMenu();
	}

	function openTerminalFromContextMenu() {
		openTerminalWindow();
		closeContextMenu();
	}

	function openNoteFromContextMenu() {
		openCanvasNote();
		closeContextMenu();
	}

	function fitCanvasFromContextMenu() {
		fitInfinityView();
		closeContextMenu();
	}

	function resetCanvasFromContextMenu() {
		resetInfinityView();
		closeContextMenu();
	}

	function arrangeIcons() {
		iconPositions = {};
		if (workspaceWidth > 0 && workspaceHeight > 0) {
			if (isCanvasDesktop) {
				const worldWidth = workspaceWidth / infinityZoom;
				const worldHeight = workspaceHeight / infinityZoom;
				windowStore.arrangeActiveDesktopIcons(worldWidth, worldHeight, {
					iconWidth: ICON_WIDTH,
					iconHeight: ICON_HEIGHT,
					gap: 12,
					padding: 24,
					originX: -infinityPan.x / infinityZoom,
					originY: -infinityPan.y / infinityZoom
				});
				requestAnimationFrame(fitInfinityView);
			} else {
				windowStore.arrangeActiveDesktopIcons(workspaceWidth, workspaceHeight, {
					iconWidth: ICON_WIDTH,
					iconHeight: ICON_HEIGHT,
					gap: 12,
					padding: 16
				});
			}
		}
		closeContextMenu();
	}

	function startRenameIcon() {
		if (!contextMenuIconId) return;
		const icon = $windowStore.desktopIcons.find(i => i.id === contextMenuIconId);
		if (icon) {
			renameValue = icon.label;
			renamingIconId = contextMenuIconId;
		}
		closeContextMenu();
	}

	function finishRename() {
		if (!renamingIconId || !renameValue.trim()) {
			renamingIconId = null;
			return;
		}

		const icon = $windowStore.desktopIcons.find(i => i.id === renamingIconId);
		if (icon?.type === 'folder' && icon.folderId) {
			windowStore.renameFolder(icon.folderId, renameValue.trim());
		}
		renamingIconId = null;
	}

	function pinToDock() {
		if (!contextMenuIconId) return;
		const icon = $windowStore.desktopIcons.find(i => i.id === contextMenuIconId);
		if (icon) {
			if (icon.type === 'folder' && icon.folderId) {
				windowStore.addToDock(`folder-${icon.folderId}`);
			} else {
				windowStore.addToDock(icon.module);
			}
		}
		closeContextMenu();
	}

	function deleteFolder() {
		if (!contextMenuIconId) return;
		const icon = $windowStore.desktopIcons.find(i => i.id === contextMenuIconId);
		if (icon?.type === 'folder' && icon.folderId) {
			windowStore.deleteFolder(icon.folderId);
		}
		closeContextMenu();
	}

	function openIcon() {
		if (!contextMenuIconId) return;
		const icon = $windowStore.desktopIcons.find(i => i.id === contextMenuIconId);
		if (icon) {
			if (icon.type === 'folder' && icon.folderId) {
				windowStore.openFolder(icon.folderId);
			} else {
				windowStore.openWindow(icon.module);
			}
		}
		closeContextMenu();
	}

	const contextMenuIcon = $derived(
		contextMenuIconId ? $windowStore.desktopIcons.find(i => i.id === contextMenuIconId) : null
	);
</script>

<svelte:head>
	<title>Business OS - Desktop</title>
</svelte:head>

{#if showBootScreen}
	<BootScreen
		companyName={$desktopSettings.companyName}
		appVersion={APP_VERSION}
	/>
{:else if $session.data}
	<!-- 3D Desktop Mode -->
	{#if $desktopSettings.enable3DDesktop}
		<Desktop3D onExit={() => desktopSettings.set3DDesktop(false)} />
	{:else}
	<div class="desktop-environment" style={backgroundStyle()}>
		<!-- Animated Background Effect -->
		{#if $desktopSettings.animatedBackground.effect !== 'none'}
			<AnimatedBackground
				effectType={$desktopSettings.animatedBackground.effect}
				intensity={$desktopSettings.animatedBackground.intensity}
				colors={$desktopSettings.animatedBackground.colors}
				speed={$desktopSettings.animatedBackground.speed}
			/>
		{/if}

		<!-- Noise texture overlay -->
		{#if $desktopSettings.showNoise}
			<div class="noise-overlay"></div>
		{/if}

			<!-- Menu Bar -->
			<MenuBar />

				{#if isCanvasDesktop}
					<div class="infinity-toolbar" style="top: {menuBarHeight + 12}px;">
						<span>{isInfinityDesktop ? 'Infinity Canvas' : activeDesktopName}</span>
						<button type="button" onclick={openAppsWindow}>Apps</button>
						<button type="button" onclick={openDesktopSettings}>Modules</button>
						<button type="button" onclick={openCanvasNote}>Note</button>
						<button type="button" onclick={() => zoomInfinity(-0.1)} aria-label="Zoom out">-</button>
						<strong>{Math.round(infinityZoom * 100)}%</strong>
						<button type="button" onclick={() => zoomInfinity(0.1)} aria-label="Zoom in">+</button>
						<button type="button" onclick={fitInfinityView}>Fit</button>
						<button type="button" onclick={() => resetInfinityView()}>Reset</button>
					</div>
				{/if}

			<!-- Desktop Workspace -->
			<div
				bind:this={workspaceElement}
				class="desktop-workspace"
				class:infinity-workspace={isCanvasDesktop}
				class:panning={isPanningInfinity}
				class:space-panning={isSpacePanningCanvas}
					style="top: {menuBarHeight}px;"
					onclick={handleDesktopClick}
					onmousedown={handleDesktopMouseDown}
					onmousemove={handleDesktopMouseMove}
					onwheel={handleInfinityWheel}
					oncontextmenu={handleContextMenu}
					role="application"
				aria-label="Desktop workspace"
				>
					<div
						class="infinity-canvas-layer"
						style={isCanvasDesktop ? `transform: translate(${infinityPan.x}px, ${infinityPan.y}px) scale(${infinityZoom});` : ''}
					>
							{#if showDesktopStarter}
								<div class="desktop-starter" class:desktop-starter--infinity={isInfinityDesktop}>
									<button
										type="button"
										class="starter-close"
										onclick={dismissDesktopStarter}
										aria-label="Hide desktop starter"
									>
										x
									</button>
									<div class="starter-kicker">{isInfinityDesktop ? 'Canvas' : 'Workspace Desktop'}</div>
									<h1>{isInfinityDesktop ? 'Infinity Desktop' : activeDesktopName}</h1>
									<p>
										{isInfinityDesktop
											? 'Open modules, arrange windows, pan the canvas with the trackpad, and zoom with Command-scroll.'
											: 'This shared desktop is a team canvas. Add modules and apps, arrange windows, and follow teammates as they work.'}
									</p>
								<div class="starter-actions">
									<button type="button" onclick={openAppsWindow}>Open Apps</button>
									<button type="button" onclick={openTerminalWindow}>Open Terminal</button>
									<button type="button" onclick={openDesktopSettings}>Desktop Settings</button>
								</div>
								{#if isCanvasDesktop}
									<div class="starter-shortcuts">
										<span>Scroll pans</span>
										<span>Command-scroll zooms</span>
										<span>Middle-drag moves canvas</span>
									</div>
								{/if}
							</div>
						{/if}

						<!-- Desktop Icons - only render when workspace dimensions are known -->
						{#if workspaceWidth > 0 && workspaceHeight > 0}
							{#each visibleDesktopIcons as icon (icon.id)}
								{@const pos = getIconPosition(icon)}
									<div
										class="desktop-icon-wrapper"
										class:dragging={draggingIconId === icon.id}
										style="
											position: absolute;
											left: {pos.x}px;
											top: {pos.y}px;
										"
										oncontextmenu={(e) => handleIconContextMenu(e, icon.id)}
									>
									{#if renamingIconId === icon.id}
										<!-- Rename input -->
										<div class="rename-container">
											<div class="rename-icon-preview" style="background: {icon.folderColor || '#3B82F6'}20">
												<svg viewBox="0 0 24 24" fill={icon.folderColor || '#3B82F6'}>
													<path d="M3 7V17C3 18.1046 3.89543 19 5 19H19C20.1046 19 21 18.1046 21 17V9C21 7.89543 20.1046 7 19 7H12L10 5H5C3.89543 5 3 5.89543 3 7Z"/>
												</svg>
											</div>
											<input
												type="text"
												class="rename-input"
												bind:value={renameValue}
												onblur={finishRename}
												onkeydown={(e) => {
													if (e.key === 'Enter') finishRename();
													if (e.key === 'Escape') { renamingIconId = null; }
												}}
												autofocus
											/>
										</div>
									{:else}
										<DesktopIcon
											id={icon.id}
											module={icon.module}
											label={icon.label}
											selected={$windowStore.selectedIconIds.includes(icon.id)}
											posX={pos.x}
											posY={pos.y}
											darkBackground={darkBackground}
											iconType={icon.type || 'app'}
											folderId={icon.type === 'folder' ? icon.folderId : undefined}
											folderColor={icon.folderColor}
											customIcon={icon.customIcon}
											onSelect={handleIconSelect}
											onOpen={handleIconOpen}
											onDragStart={handleIconDragStart}
											onDragMove={handleIconDragMove}
											onDragEnd={handleIconDragEnd}
											onCustomizeIcon={handleCustomizeIcon}
											dragScale={isCanvasDesktop ? infinityZoom : 1}
										/>
									{/if}
								</div>
							{/each}
						{/if}

						<!-- Snap Zone Preview Overlay -->
						{#if currentSnapZone}
							{@const preview = snapZonePreview()}
							{#if preview}
								<div
									class="snap-zone-preview"
									style="
										left: {preview.x}px;
										top: {preview.y}px;
										width: {preview.width}px;
										height: {preview.height}px;
									"
								></div>
							{/if}
						{/if}

						<!-- Windows -->
						{#each $visibleWindows as win (win.id)}
							<Window
								window={win}
								focused={$focusedWindow?.id === win.id}
								zIndex={getWindowZIndex(win.id)}
								workspaceHeight={isCanvasDesktop ? INFINITY_CANVAS_LIMIT * 2 : workspaceHeight}
								workspaceWidth={isCanvasDesktop ? INFINITY_CANVAS_LIMIT * 2 : workspaceWidth}
								viewportScale={isCanvasDesktop ? infinityZoom : 1}
								readableScale={1}
								unbounded={isCanvasDesktop}
								maximizeBounds={visibleCanvasWindowBounds()}
								onsnapZoneChange={handleSnapZoneChange}
								>
									{#snippet children()}
										<svelte:boundary>
											<WindowContent
												module={win.module}
												windowTitle={win.title}
												deployedApps={$deployedAppsStore.apps}
												workspaceApps={workspaceDesktopApps}
												windowData={win.data}
												windowId={win.id}
											/>
											{#snippet failed(error)}
												<div class="window-content-error">
													<strong>{win.title} crashed</strong>
													<span>{error instanceof Error ? error.message : 'This window could not render.'}</span>
													<button type="button" onclick={() => windowStore.closeWindow(win.id)}>Close window</button>
												</div>
											{/snippet}
										</svelte:boundary>
									{/snippet}
								</Window>
							{/each}

						{#each $desktopPresenceStore as cursor (cursor.clientId)}
							{@const cursorPos = getPresenceCursorPosition(cursor)}
							<div
								class="presence-cursor"
								style="
									left: {cursorPos.x}px;
									top: {cursorPos.y}px;
									--presence-color: {cursor.color};
									--presence-readable-scale: {canvasCursorScale};
								"
							>
								<svg class="presence-cursor-arrow" viewBox="0 0 18 22" aria-hidden="true">
									<path d="M1 1L16 11L9 13L6 21L1 1Z" />
								</svg>
								<span>
									<strong>{cursor.name}</strong>
									{#if cursor.activeTitle}
										<small>Using {cursor.activeTitle}</small>
									{/if}
									<button
										type="button"
										class="presence-follow"
										class:active={$followedDesktopCursor === cursor.clientId}
										onclick={() => toggleFollowCursor(cursor)}
									>
										{$followedDesktopCursor === cursor.clientId ? 'Following' : 'Follow'}
									</button>
								</span>
							</div>
						{/each}

						<!-- Selection box -->
						{#if isSelecting}
							{@const box = selectionBox()}
							{#if box && box.width > 2 && box.height > 2}
								<div
									class="selection-box"
									style="
										left: {box.x}px;
										top: {box.y}px;
										width: {box.width}px;
										height: {box.height}px;
									"
								></div>
							{/if}
						{/if}
					</div>
			</div>

		<!-- Context Menu -->
		{#if showContextMenu}
			<DesktopContextMenu
				x={contextMenuPos.x}
				y={contextMenuPos.y}
				type={contextMenuType}
				icon={contextMenuIcon}
				onClose={closeContextMenu}
				onOpenIcon={openIcon}
				onRenameIcon={startRenameIcon}
				onPinToDock={pinToDock}
					onDeleteFolder={deleteFolder}
					onCreateNewFolder={createNewFolder}
					onArrangeIcons={arrangeIcons}
					onOpenApps={openAppsFromContextMenu}
					onOpenTerminal={openTerminalFromContextMenu}
					onCreateNote={openNoteFromContextMenu}
					onFitCanvas={fitCanvasFromContextMenu}
					onResetCanvas={resetCanvasFromContextMenu}
					onOpenDesktopSettings={openDesktopSettings}
					{isCanvasDesktop}
				/>
		{/if}

		<!-- Dock -->
		<Dock />

			<!-- OSA Orb - floating, draggable on window desktop -->
		<OsaOrb />

		<!-- Spotlight Search -->
		<SpotlightSearch open={showSpotlight} onClose={() => showSpotlight = false} />

		<!-- Icon Picker Modal -->
		{#if showIconPicker}
			<div class="icon-picker-overlay">
				<button class="icon-picker-backdrop" onclick={handleIconPickerClose}></button>
				<IconPicker
					currentIcon={customizeIconCurrentConfig}
					onSelect={handleIconPickerSelect}
					onClose={handleIconPickerClose}
				/>
			</div>
		{/if}

		<!-- Onboarding Overlay -->
		{#if showOnboarding}
			<DesktopOnboarding
				step={onboardingStep}
				onNext={nextOnboardingStep}
				onSkip={skipOnboarding}
				onComplete={completeOnboarding}
			/>
		{/if}
	</div>
	{/if}
{/if}

<style>
	.desktop-environment {
		position: fixed;
		inset: 0;
		overflow: hidden;
	}

	/* Noise texture overlay */
	.noise-overlay {
		position: absolute;
		inset: 0;
		opacity: 0.03;
		pointer-events: none;
		z-index: 1;
		background-image: url("data:image/svg+xml,%3Csvg viewBox='0 0 256 256' xmlns='http://www.w3.org/2000/svg'%3E%3Cfilter id='noiseFilter'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.9' numOctaves='4' stitchTiles='stitch'/%3E%3C/filter%3E%3Crect width='100%25' height='100%25' filter='url(%23noiseFilter)'/%3E%3C/svg%3E");
	}

		.desktop-workspace {
			position: absolute;
			/* top is set dynamically via inline style */
			left: 0;
			right: 0;
			bottom: 80px; /* Dock area */
			overflow: hidden;
			z-index: 2;
		}

		.desktop-workspace.infinity-workspace {
			cursor: grab;
			background: transparent;
		}

		.desktop-workspace.infinity-workspace:active {
			cursor: grabbing;
		}

		.desktop-workspace.infinity-workspace.panning,
		.desktop-workspace.infinity-workspace.space-panning {
			cursor: grabbing;
		}

		.infinity-canvas-layer {
			position: absolute;
			inset: 0;
			transform-origin: 0 0;
			will-change: transform;
		}

		.desktop-starter {
			position: absolute;
			left: 50%;
			top: 44%;
			z-index: 70;
			width: min(440px, calc(100vw - 48px));
			padding: 22px;
			transform: translate(-50%, -50%);
			border: 1px solid rgba(255, 255, 255, 0.44);
			border-radius: 22px;
			background: rgba(255, 255, 255, 0.78);
			backdrop-filter: blur(24px) saturate(1.35);
			-webkit-backdrop-filter: blur(24px) saturate(1.35);
			box-shadow:
				0 24px 70px rgba(15, 23, 42, 0.18),
				inset 0 1px 0 rgba(255, 255, 255, 0.7);
			color: #111827;
			text-align: left;
			pointer-events: auto;
		}

		.desktop-starter--infinity {
			border-color: rgba(37, 99, 235, 0.24);
			background: rgba(255, 255, 255, 0.84);
		}

		.starter-close {
			position: absolute;
			top: 12px;
			right: 12px;
			width: 24px;
			height: 24px;
			border: 1px solid rgba(15, 23, 42, 0.1);
			border-radius: 999px;
			background: rgba(255, 255, 255, 0.66);
			color: #64748b;
			font-size: 13px;
			font-weight: 850;
			line-height: 1;
			cursor: pointer;
		}

		.starter-close:hover {
			background: #111827;
			color: white;
			transform: translateY(-1px);
		}

		.starter-kicker {
			margin-bottom: 10px;
			color: #2563eb;
			font-size: 11px;
			font-weight: 850;
			letter-spacing: 0.08em;
			text-transform: uppercase;
		}

		.desktop-starter h1 {
			margin: 0;
			color: #0f172a;
			font-size: 26px;
			font-weight: 850;
			letter-spacing: 0;
			line-height: 1.05;
		}

		.desktop-starter p {
			margin: 10px 0 18px;
			color: #475569;
			font-size: 13px;
			line-height: 1.5;
		}

		.starter-actions {
			display: flex;
			flex-wrap: wrap;
			gap: 8px;
		}

		.starter-actions button {
			height: 34px;
			padding: 0 13px;
			border: 1px solid rgba(15, 23, 42, 0.12);
			border-radius: 10px;
			background: #111827;
			color: white;
			font-size: 12px;
			font-weight: 750;
			cursor: pointer;
		}

		.starter-actions button:nth-child(n + 2) {
			background: rgba(255, 255, 255, 0.74);
			color: #111827;
		}

		.starter-actions button:hover {
			transform: translateY(-1px);
			box-shadow: 0 10px 22px rgba(15, 23, 42, 0.14);
		}

		.starter-shortcuts {
			display: flex;
			flex-wrap: wrap;
			gap: 6px;
			margin-top: 14px;
		}

		.starter-shortcuts span {
			padding: 4px 8px;
			border: 1px solid rgba(37, 99, 235, 0.16);
			border-radius: 999px;
			background: rgba(37, 99, 235, 0.07);
			color: #1e40af;
			font-size: 11px;
			font-weight: 700;
		}

		.infinity-toolbar {
			position: absolute;
			right: 18px;
			z-index: 10002;
			display: flex;
			align-items: center;
			gap: 8px;
			height: 34px;
			padding: 0 8px 0 12px;
			border: 1px solid rgba(255, 255, 255, 0.38);
			border-radius: 999px;
			background: rgba(17, 24, 39, 0.72);
			color: white;
			backdrop-filter: blur(18px);
			-webkit-backdrop-filter: blur(18px);
			box-shadow: 0 14px 34px rgba(0, 0, 0, 0.24);
			-webkit-app-region: no-drag;
		}

		.infinity-toolbar span,
		.infinity-toolbar strong {
			font-size: 11px;
			font-weight: 800;
			white-space: nowrap;
		}

		.infinity-toolbar strong {
			min-width: 42px;
			text-align: center;
			color: rgba(255, 255, 255, 0.76);
		}

		.infinity-toolbar button {
			height: 24px;
			min-width: 24px;
			padding: 0 8px;
			border: 1px solid rgba(255, 255, 255, 0.18);
			border-radius: 999px;
			background: rgba(255, 255, 255, 0.12);
			color: white;
			font-size: 12px;
			font-weight: 800;
			cursor: pointer;
		}

		.infinity-toolbar button:hover {
			background: rgba(255, 255, 255, 0.2);
		}

		@media (max-width: 768px) {
		.desktop-environment {
			/* Ensure no horizontal overflow on mobile */
			overflow: hidden;
			touch-action: none;
		}
		.desktop-workspace {
			bottom: 72px;
		}
	}

	.desktop-icon-wrapper {
		pointer-events: auto;
	}

	.desktop-icon-wrapper.dragging {
		z-index: 9998;
	}

	/* Selection box (lasso) */
	.selection-box {
		position: absolute;
		background: rgba(0, 102, 255, 0.1);
		border: 1px solid rgba(0, 102, 255, 0.5);
		border-radius: 2px;
		pointer-events: none;
		z-index: 50;
	}

	/* Snap zone preview overlay */
	.snap-zone-preview {
		position: absolute;
		background: rgba(100, 150, 255, 0.15);
		border: 2px solid rgba(100, 150, 255, 0.5);
		border-radius: 8px;
		pointer-events: none;
		z-index: 99;
		transition: all 0.15s ease-out;
		box-shadow: inset 0 0 30px rgba(100, 150, 255, 0.1);
	}

	.window-content-error {
		height: 100%;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 10px;
		padding: 24px;
		background: color-mix(in srgb, var(--dbg, #ffffff) 96%, #ef4444 4%);
		color: var(--dt, #111827);
		text-align: center;
	}

	.window-content-error strong {
		font-size: 14px;
		font-weight: 800;
	}

	.window-content-error span {
		max-width: 420px;
		color: var(--dt3, #6b7280);
		font-size: 12px;
		line-height: 1.45;
	}

	.window-content-error button {
		height: 30px;
		padding: 0 12px;
		border: 1px solid var(--dbd, #d1d5db);
		border-radius: 8px;
		background: var(--dt, #111827);
		color: var(--dbg, #ffffff);
		font-size: 12px;
		font-weight: 750;
		cursor: pointer;
	}

		.presence-cursor {
			position: absolute;
			z-index: 10000;
			display: flex;
			align-items: center;
			gap: 6px;
			pointer-events: auto;
			transform: translate(2px, 2px) scale(var(--presence-readable-scale, 1));
			transform-origin: top left;
			filter: drop-shadow(0 8px 18px rgba(0, 0, 0, 0.28));
		}

	.presence-cursor-arrow {
		width: 18px;
		height: 22px;
		fill: var(--presence-color);
		stroke: white;
		stroke-width: 1.5;
	}

	.presence-cursor span {
		max-width: 140px;
		overflow: hidden;
		padding: 4px 8px;
		border-radius: 999px;
		background: var(--presence-color);
		color: white;
		font-size: 11px;
		font-weight: 700;
		line-height: 1;
	}

	.presence-cursor strong,
	.presence-cursor small {
		display: block;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

		.presence-cursor small {
			margin-top: 3px;
			opacity: 0.82;
			font-size: 10px;
			font-weight: 650;
		}

		.presence-follow {
			display: block;
			width: 100%;
			margin-top: 5px;
			padding: 3px 6px;
			border: 1px solid rgba(255, 255, 255, 0.32);
			border-radius: 999px;
			background: rgba(255, 255, 255, 0.18);
			color: white;
			font-size: 10px;
			font-weight: 800;
			cursor: pointer;
		}

		.presence-follow:hover,
		.presence-follow.active {
			background: white;
			color: var(--presence-color);
		}

	/* Rename container */
	.rename-container {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 6px;
		padding: 8px;
		width: 100px;
	}

	.rename-icon-preview {
		width: 56px;
		height: 56px;
		border-radius: 12px;
		display: flex;
		align-items: center;
		justify-content: center;
		box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
	}

	.rename-icon-preview svg {
		width: 28px;
		height: 28px;
	}

	.rename-input {
		width: 100%;
		padding: 4px 6px;
		font-size: 11px;
		border: 1px solid #3B82F6;
		border-radius: 4px;
		text-align: center;
		outline: none;
		background: white;
	}

	.rename-input:focus {
		box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.3);
	}

	/* Icon Picker Modal */
	.icon-picker-overlay {
		position: fixed;
		inset: 0;
		z-index: 10000;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.icon-picker-backdrop {
		position: absolute;
		inset: 0;
		background: rgba(0, 0, 0, 0.5);
		backdrop-filter: blur(4px);
		-webkit-backdrop-filter: blur(4px);
		border: none;
		cursor: pointer;
	}
</style>
