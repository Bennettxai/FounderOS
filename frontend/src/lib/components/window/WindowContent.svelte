<script lang="ts">
	import { TerminalApp } from '$lib/components/terminal';
	import DesktopSettingsContent from '$lib/components/desktop/DesktopSettingsContent.svelte';
	import FolderWindow from '$lib/components/desktop/FolderWindow.svelte';
	import FileBrowser from '$lib/components/desktop/FileBrowser.svelte';
	import type { DeployedApp } from '$lib/stores/deployedAppsStore';
	import type { WorkspaceApp } from '$lib/api/apps';
	import { windowStore } from '$lib/stores/windowStore';
	import { getModuleWindow } from './moduleWindowRegistry';

	interface Props {
		windowId: string;
		module: string;
		windowTitle: string;
		deployedApps: DeployedApp[];
		workspaceApps?: WorkspaceApp[];
		windowData?: Record<string, unknown>;
	}

	let { windowId, module, windowTitle, deployedApps, workspaceApps = [], windowData }: Props = $props();

	const isTerminal = $derived(module === 'terminal');
	const isDesktopSettings = $derived(module === 'desktop-settings');
	const isCanvasNote = $derived(module.startsWith('canvas-note'));
	const isFolder = $derived(module.startsWith('folder-'));
	const isFileBrowser = $derived(module === 'files' || module === 'finder');
	const isOsaApp = $derived(module.startsWith('osa-app-'));
	const isWorkspaceApp = $derived(module.startsWith('workspace-app-') || module.startsWith('web-app-preview-'));
	const isIframe = $derived(!isTerminal && !isDesktopSettings && !isCanvasNote && !isFolder && !isFileBrowser && !isOsaApp && !isWorkspaceApp);

	const folderId = $derived(isFolder ? module.replace('folder-', '') : '');
	const osaAppId = $derived(isOsaApp ? module.replace('osa-app-', '') : '');
	const workspaceAppId = $derived(
		module.startsWith('workspace-app-') ? module.replace('workspace-app-', '') : ''
	);
	const deployedApp = $derived(isOsaApp ? deployedApps.find(app => app.id === osaAppId) : undefined);
	const workspaceApp = $derived(isWorkspaceApp ? workspaceApps.find(app => app.id === workspaceAppId) : undefined);
	const workspaceAppUrl = $derived((windowData?.url as string | undefined) || workspaceApp?.url || null);
	const workspaceAppLaunchMode = $derived(
		((windowData?.launchMode as string | undefined) || workspaceApp?.launch_mode || 'iframe') as 'iframe' | 'browser' | 'external'
	);
	const workspaceAppNeedsBrowserSession = $derived(workspaceAppLaunchMode === 'browser' || workspaceAppLaunchMode === 'external');
	let workspaceAppError = $state('');
	let workspaceWebview = $state<HTMLElement | null>(null);
	let noteText = $state('');
	let hydratedNoteText = '';
	let noteSaveTimer: ReturnType<typeof setTimeout> | null = null;
	const chromeUserAgent =
		'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36';

	function handleWorkspaceAppLoadStart() {
		workspaceAppError = '';
	}

	function handleWorkspaceAppLoadFinish() {
		workspaceAppError = '';
	}

	function handleWorkspaceAppDomReady() {
		workspaceAppError = '';
	}

	function handleWorkspaceAppLoadFail(event: Event) {
		const detail = event as Event & { errorDescription?: string; errorCode?: number };
		if (detail.errorCode === -3) return;
		workspaceAppError = detail.errorDescription || 'The embedded browser could not load this URL.';
	}

	function navigateWorkspaceWebview(url: string) {
		const webview = workspaceWebview;
		if (!webview || !url) return;
		if ('loadURL' in webview) {
			(webview as HTMLElement & { loadURL: (nextUrl: string) => void }).loadURL(url);
			return;
		}
		(webview as HTMLElement & { src: string }).src = url;
	}

	function handleWorkspaceAppNewWindow(event: Event) {
		const detail = event as Event & { url?: string; preventDefault: () => void };
		if (!detail.url) return;
		detail.preventDefault();
		navigateWorkspaceWebview(detail.url);
	}

	function handleWorkspaceAppCreatedWindow(event: Event) {
		const detail = event as Event & { url?: string; preventDefault?: () => void };
		if (!detail.url) return;
		detail.preventDefault?.();
		navigateWorkspaceWebview(detail.url);
	}

	function reloadWorkspaceApp() {
		const webview = workspaceWebview;
		if (!webview || !('reload' in webview)) return;
		(webview as HTMLElement & { reload: () => void }).reload();
	}

	function openWorkspaceAppTab() {
		if (!workspaceAppUrl) return;
		window.open(workspaceAppUrl, '_blank', 'noopener,noreferrer');
	}

	$effect(() => {
		if (!isWorkspaceApp || !workspaceAppUrl) {
			workspaceAppError = '';
			return;
		}

		workspaceAppError = '';
	});

	$effect(() => {
		if (!isCanvasNote) return;
		const text = typeof windowData?.text === 'string' ? windowData.text : '';
		if (text === hydratedNoteText) return;
		hydratedNoteText = text;
		noteText = text;
	});

	function saveNoteText() {
		if (!isCanvasNote) return;
		if (noteSaveTimer) clearTimeout(noteSaveTimer);
		noteSaveTimer = setTimeout(() => {
			hydratedNoteText = noteText;
			windowStore.updateWindowData(windowId, { text: noteText });
		}, 180);
	}

	$effect(() => {
		return () => {
			if (noteSaveTimer) clearTimeout(noteSaveTimer);
		};
	});

	$effect(() => {
		const webview = workspaceWebview;
		if (!webview) return;

		webview.addEventListener('did-start-loading', handleWorkspaceAppLoadStart);
		webview.addEventListener('did-finish-load', handleWorkspaceAppLoadFinish);
		webview.addEventListener('dom-ready', handleWorkspaceAppDomReady);
		webview.addEventListener('did-fail-load', handleWorkspaceAppLoadFail);
		webview.addEventListener('new-window', handleWorkspaceAppNewWindow);
		webview.addEventListener('did-create-window', handleWorkspaceAppCreatedWindow);

		return () => {
			webview.removeEventListener('did-start-loading', handleWorkspaceAppLoadStart);
			webview.removeEventListener('did-finish-load', handleWorkspaceAppLoadFinish);
			webview.removeEventListener('dom-ready', handleWorkspaceAppDomReady);
			webview.removeEventListener('did-fail-load', handleWorkspaceAppLoadFail);
			webview.removeEventListener('new-window', handleWorkspaceAppNewWindow);
			webview.removeEventListener('did-create-window', handleWorkspaceAppCreatedWindow);
		};
	});

	const iframeUrl = $derived.by(() => {
		const base = getModuleWindow(module)?.url ?? null;
		if (!base) return null;
		if (module === 'platform') return base;
		return `${base}?embed=true`;
	});
	const iframeTitle = $derived(getModuleWindow(module)?.title ?? windowTitle);
</script>

<div class="window-module-content">
	{#if isTerminal}
		<TerminalApp />
	{:else if isDesktopSettings}
		<DesktopSettingsContent />
	{:else if isCanvasNote}
		<div class="canvas-note">
			<textarea
				bind:value={noteText}
				oninput={saveNoteText}
				placeholder="Write a note..."
				aria-label="Canvas note"
			></textarea>
		</div>
	{:else if isFolder}
		<FolderWindow {folderId} />
	{:else if isFileBrowser}
		<FileBrowser />
	{:else if isOsaApp}
		{#if deployedApp && deployedApp.status === 'running'}
			<iframe
				src={deployedApp.url}
				title={deployedApp.name}
				class="module-iframe osa-app-iframe"
				sandbox="allow-scripts allow-same-origin allow-forms allow-popups"
			></iframe>
		{:else}
			<div class="module-placeholder">
				<span class="placeholder-icon">
					<svg class="w-12 h-12" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
					</svg>
				</span>
				<span class="placeholder-text">App {deployedApp?.status === 'crashed' ? 'Crashed' : 'Not Running'}</span>
			</div>
		{/if}
	{:else if isWorkspaceApp}
		{#if workspaceAppUrl}
			<div class="workspace-app-shell">
				{#if workspaceAppNeedsBrowserSession}
					<div class="workspace-app-toolbar">
						<span>{workspaceAppLaunchMode === 'external' ? 'External app' : 'Browser session'}</span>
						<button type="button" onclick={openWorkspaceAppTab}>Open tab</button>
					</div>
				{/if}
				<webview
					bind:this={workspaceWebview}
					src={workspaceAppUrl}
					class="module-webview workspace-app-webview"
					partition="persist:businessos-webapps"
					useragent={chromeUserAgent}
					allowpopups
				></webview>
				{#if workspaceAppError}
					<div class="workspace-app-state workspace-app-state--error">
						<span>{workspaceAppError}</span>
						<button type="button" onclick={reloadWorkspaceApp}>Reload</button>
						<button type="button" onclick={openWorkspaceAppTab}>Open tab</button>
					</div>
				{/if}
			</div>
		{:else}
			<div class="module-placeholder">
				<span class="placeholder-icon">
					<svg class="w-12 h-12" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M13.5 6H5.75A1.75 1.75 0 004 7.75v10.5C4 19.216 4.784 20 5.75 20h10.5A1.75 1.75 0 0018 18.25V10.5M15 4h5m0 0v5m0-5L10 14" />
					</svg>
				</span>
				<span class="placeholder-text">App URL missing</span>
			</div>
		{/if}
	{:else if isIframe && iframeUrl}
		<iframe src={iframeUrl} title={iframeTitle} class="module-iframe"></iframe>
	{:else}
		<div class="module-placeholder">
			<span class="placeholder-icon">
				<svg class="w-12 h-12" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
				</svg>
			</span>
			<span class="placeholder-text">{windowTitle}</span>
		</div>
	{/if}
</div>

<style>
	.window-module-content {
		width: 100%;
		height: 100%;
		display: flex;
		flex-direction: column;
	}

	.module-iframe {
		width: 100%;
		height: 100%;
		border: none;
		background: white;
	}

	.module-webview {
		display: flex;
		width: 100%;
		flex: 1 1 auto;
		min-height: 0;
		border: none;
		background: white;
	}

	.workspace-app-shell {
		position: relative;
		flex: 1;
		min-height: 0;
		background: white;
		display: flex;
		flex-direction: column;
	}

	.workspace-app-toolbar {
		height: 34px;
		flex: 0 0 auto;
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 12px;
		padding: 0 12px;
		border-bottom: 1px solid rgba(17,24,39,0.08);
		background: #f8fafc;
		color: #64748b;
		font-size: 0.74rem;
		font-weight: 650;
	}

	.workspace-app-toolbar button {
		border: 1px solid rgba(17,24,39,0.12);
		border-radius: 7px;
		background: #fff;
		color: #111827;
		cursor: pointer;
		font: inherit;
		font-size: 0.72rem;
		font-weight: 700;
		padding: 4px 9px;
	}

	.workspace-app-toolbar button:hover {
		background: #f1f5f9;
	}

	.workspace-app-state {
		position: absolute;
		left: 12px;
		right: 12px;
		bottom: 44px;
		z-index: 3;
		display: inline-flex;
		align-items: center;
		justify-content: center;
		max-width: min(560px, calc(100% - 48px));
		gap: 12px;
		margin: 0 auto;
		padding: 10px 14px;
		border-radius: 8px;
		background: rgba(255,255,255,0.94);
		border: 1px solid rgba(17,24,39,0.1);
		color: #374151;
		font-size: 0.82rem;
		font-weight: 650;
		box-shadow: 0 16px 40px rgba(0,0,0,0.12);
	}

	.workspace-app-state--error {
		color: #991b1b;
	}

	.workspace-app-state button {
		border: 0;
		border-radius: 6px;
		background: #111827;
		color: white;
		cursor: pointer;
		font: inherit;
		font-size: 0.78rem;
		font-weight: 700;
		padding: 6px 10px;
	}

	.canvas-note {
		flex: 1;
		min-height: 0;
		background:
			linear-gradient(180deg, rgba(255,255,255,0.42), rgba(255,255,255,0)),
			#fff7b8;
	}

	.canvas-note textarea {
		width: 100%;
		height: 100%;
		resize: none;
		border: 0;
		outline: none;
		padding: 18px 18px 22px;
		background: transparent;
		color: #2f2a12;
		font: 500 15px/1.45 ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
	}

	.canvas-note textarea::placeholder {
		color: rgba(47, 42, 18, 0.42);
	}

	.module-placeholder {
		flex: 1;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 16px;
		color: #999;
		background: #fafafa;
	}

	.placeholder-icon {
		color: #ccc;
	}

	.placeholder-text {
		font-size: 14px;
		font-weight: 500;
	}
</style>
