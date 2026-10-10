<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { terminalStore } from '$lib/stores/terminal';
	import { chatModelStore } from '$lib/stores/chat/chatModelStore.svelte';
	import TerminalTabBar from './TerminalTabBar.svelte';
	import TerminalProviderBar from './TerminalProviderBar.svelte';
	import TerminalFocusBar from './TerminalFocusBar.svelte';
	import TerminalSplitContainer from './TerminalSplitContainer.svelte';
	import TerminalEnvironmentBar from './TerminalEnvironmentBar.svelte';
	import TerminalSandboxAnalysis from './TerminalSandboxAnalysis.svelte';
	import { getBusinessOSHome } from '$lib/services/businessOSHome';
	import { currentWorkspace } from '$lib/stores/workspaces';
	import { createMIOSASandboxSession, getMIOSAStatus, type MIOSAConnectionStatus } from '$lib/api/miosa';
	import type {
		TerminalProvider,
		TerminalConfig,
		EnvironmentMode,
		EnvironmentInfo,
		MiosaAccountSource
	} from '$lib/stores/terminal/terminalTypes';
	import { getThemeColors } from './themes';

	// Current state from store
	const tabs = $derived(terminalStore.tabs);
	const activeTabId = $derived(terminalStore.activeTabId);
	const activeTab = $derived(terminalStore.activeTab);
	const config = $derived(terminalStore.config);
	const focusedPaneId = $derived(terminalStore.focusedPaneId);

	// Focus mode state
	let activeFocusMode = $state('general');

	// Shell write function registry - keyed by paneId
	const shellRefs = new Map<string, (data: string) => void>();

	function handleShellReady(paneId: string, write: (data: string) => void | null) {
		if (write) {
			shellRefs.set(paneId, write);

			// Auto-launch agent if this pane was queued for it
			if (pendingAgentLaunch.has(paneId)) {
				pendingAgentLaunch.delete(paneId);
				// Find the tab that owns this pane to get its provider
				const ownerTab = tabs.find(t => t.rootPaneId === paneId);
				const command = ownerTab ? agentCommand(ownerTab.provider) : null;
				if (command) {
					// Small delay to let the shell fully initialize
					setTimeout(() => write(command), 500);
				}
			}
		} else {
			shellRefs.delete(paneId);
		}
	}

	// Sandbox analysis state
	let showSandboxAnalysis = $state(false);
	let miosaStatus = $state<MIOSAConnectionStatus | null>(null);
	let creatingSandbox = $state(false);

	onMount(async () => {
		try {
			miosaStatus = await getMIOSAStatus();
		} catch {
			miosaStatus = null;
		}
	});

	// Background color follows theme
	const bgColor = $derived(getThemeColors(config.theme).background);

	// Current provider for focus bar
	const currentProvider = $derived(activeTab?.provider ?? 'shell');

	// Environment mode derived values
	const currentEnvMode = $derived(activeTab ? terminalStore.getEnvironmentInfo(activeTab.id).mode : 'local');
	const currentEnvInfo = $derived(activeTab ? terminalStore.getEnvironmentInfo(activeTab.id) : { mode: 'local' as EnvironmentMode });

	async function handleNewTab() {
		terminalStore.createTab('shell', await buildEnvironmentForMode(currentEnvMode));
	}

	function handleCloseTab(tabId: string) {
		terminalStore.closeTab(tabId);
	}

	function handleSwitchTab(tabId: string) {
		terminalStore.switchTab(tabId);
	}

	// Track which panes need auto-launch after shell connects
	const pendingAgentLaunch = new Set<string>();

	async function handleProviderChange(provider: TerminalProvider) {
		// Find an existing tab with this provider - switch to it instead of overwriting
		const existingTab = tabs.find(t => t.provider === provider && terminalStore.getEnvironmentInfo(t.id).mode === currentEnvMode);
		if (existingTab) {
			terminalStore.switchTab(existingTab.id);
			return;
		}
		// Create a new shell tab - all providers use real shell
		const tabId = terminalStore.createTab(provider, await buildEnvironmentForMode(currentEnvMode));
		if (tabId && provider !== 'shell') {
			// Find the root pane of the new tab and queue agent launch
			const tab = tabs.find(t => t.id === tabId);
			if (tab) {
				pendingAgentLaunch.add(tab.rootPaneId);
			}
		}
	}

	function handleConfigChange(partial: Partial<TerminalConfig>) {
		terminalStore.updateConfig(partial);
	}

	function handleSessionCreated(paneId: string, sessionId: string) {
		terminalStore.setPaneSessionId(paneId, sessionId);
	}

	function handlePaneFocus(paneId: string) {
		terminalStore.setFocusedPane(paneId);
	}

	function handleFocusModeChange(mode: string) {
		activeFocusMode = mode;
	}

	async function handleEnvironmentChange(mode: EnvironmentMode) {
		terminalStore.setLaunchEnvironmentMode(mode);
		if (currentEnvMode === mode) return;

		const existingTab = tabs.find(t => terminalStore.getEnvironmentInfo(t.id).mode === mode);
		if (existingTab) {
			terminalStore.switchTab(existingTab.id);
			return;
		}

		terminalStore.createTab('shell', await buildEnvironmentForMode(mode));
	}

	async function buildEnvironmentForMode(mode: EnvironmentMode): Promise<EnvironmentInfo> {
		if (mode !== 'sandbox') return { mode };

		const workspace = $currentWorkspace;
		const accountSource = resolveSandboxAccountSource();
		const baseEnv: EnvironmentInfo = {
			mode,
			sandboxRemoteState: 'local-ready',
			miosaAccountSource: accountSource,
			miosaAttribution: {
				externalWorkspaceId: workspace?.id,
				externalWorkspaceSlug: workspace?.slug,
			},
		};

		if (accountSource !== 'businessos') return baseEnv;

		creatingSandbox = true;
		try {
			const session = await createMIOSASandboxSession({
				workspace_id: workspace?.id,
				name: workspace?.name ? `${workspace.name} sandbox` : 'BusinessOS sandbox',
				cols: 120,
				rows: 30,
				shell: 'bash',
			});
			return {
				...baseEnv,
				sandboxId: session.sandbox_id,
				sandboxRemoteState: session.ws_url ? 'remote-attached' : 'remote-pending',
				miosaSandboxId: session.sandbox_id,
				miosaWorkspaceId: session.miosa_workspace_id,
				miosaTerminalWsUrl: session.ws_url,
				miosaPreviewUrl: session.preview_url,
				miosaAttribution: {
					...baseEnv.miosaAttribution,
					externalWorkspaceId: session.external_workspace_id,
					externalUserId: session.external_user_id,
				},
			};
		} catch (error) {
			return {
				...baseEnv,
				sandboxRemoteState: 'remote-error',
				miosaError: error instanceof Error ? error.message : 'Failed to create MIOSA sandbox',
			};
		} finally {
			creatingSandbox = false;
		}
	}

	function resolveSandboxAccountSource(): MiosaAccountSource {
		if ((miosaStatus?.businessos_tenant_available || miosaStatus?.capacity_provider === 'businessos') && miosaStatus?.businessos_sandbox_enabled) {
			return 'businessos';
		}
		if (miosaStatus?.user_key_available || miosaStatus?.capacity_provider === 'user') {
			return 'user';
		}
		return 'local';
	}

	function handleExitSandbox() {
		showSandboxAnalysis = true;
	}

	function handleSandboxMerge() {
		showSandboxAnalysis = false;
		if (activeTabId) {
			terminalStore.setEnvironmentMode(activeTabId, 'production');
		}
	}

	function handleSandboxStay() {
		showSandboxAnalysis = false;
	}

	// Agent launch commands - cd into the BusinessOS home first so agents read
	// CLAUDE.md. The home is auto-detected (dev repo root, or ~/BusinessOS for
	// packaged users) and user-overridable - never a hardcoded dev path.
	// Absolute BusinessOS home, or '' when unresolved. The shell PTY already
	// opens here, so the `cd` is only a safety net - and only emitted for a real
	// absolute path (never `cd "~"`, which fails because a quoted tilde does not
	// expand).
	let bosDir = $state('');
	onMount(async () => {
		try {
			bosDir = await getBusinessOSHome();
		} catch {
			bosDir = '';
		}
	});
	function cdPrefix(): string {
		return bosDir && bosDir !== '~' ? `cd ${JSON.stringify(bosDir)} && ` : '';
	}
	function agentCommand(agent: string): string | undefined {
		switch (agent) {
			case 'claude':
				return `${cdPrefix()}claude --dangerously-skip-permissions\n`;
			case 'codex':
				return `${cdPrefix()}codex --dangerously-bypass-approvals-and-sandbox -s danger-full-access -a never\n`;
			case 'ollama':
				return 'ollama run\n';
			case 'osa':
				return `${cdPrefix()}osa\n`;
			default:
				return undefined;
		}
	}

	function handleLaunchAgent(agent: string) {
		const command = agentCommand(agent);
		if (!command) return;

		// Use the focused pane, or fall back to the first leaf in the active tab
		const targetPaneId = focusedPaneId ?? getSplitTarget();
		if (!targetPaneId) return;

		const write = shellRefs.get(targetPaneId);
		if (write) {
			write(command);
		}
	}

	function handleStopAgent() {
		const targetPaneId = focusedPaneId ?? getSplitTarget();
		if (!targetPaneId) return;

		const write = shellRefs.get(targetPaneId);
		if (write) {
			// Send Ctrl+C (ASCII 0x03) to interrupt the running agent
			write('\x03');
		}
	}

	// Keyboard shortcuts
	function handleKeydown(e: KeyboardEvent) {
		const isMeta = e.metaKey || e.ctrlKey;

		if (isMeta && e.key === 't') {
			e.preventDefault();
			handleNewTab();
		} else if (isMeta && e.key === 'w') {
			e.preventDefault();
			if (activeTabId) handleCloseTab(activeTabId);
		} else if (isMeta && e.shiftKey && e.key === '[') {
			e.preventDefault();
			terminalStore.switchTabRelative(-1);
		} else if (isMeta && e.shiftKey && e.key === ']') {
			e.preventDefault();
			terminalStore.switchTabRelative(1);
		} else if (isMeta && !e.shiftKey && e.key === 'd') {
			e.preventDefault();
			const targetId = getSplitTarget();
			if (targetId) terminalStore.splitPane(targetId, 'horizontal');
		} else if (isMeta && e.shiftKey && (e.key === 'D' || e.key === 'd')) {
			if (e.shiftKey) {
				e.preventDefault();
				const targetId = getSplitTarget();
				if (targetId) terminalStore.splitPane(targetId, 'vertical');
			}
		} else if (isMeta && /^[1-5]$/.test(e.key)) {
			e.preventDefault();
			const providers: TerminalProvider[] = ['shell', 'osa', 'claude', 'codex', 'ollama'];
			const idx = parseInt(e.key) - 1;
			if (idx >= 0 && idx < providers.length) {
				handleProviderChange(providers[idx]);
			}
		}
	}

	function getSplitTarget(): string | null {
		if (!activeTab) return null;
		const rootPane = terminalStore.panes[activeTab.rootPaneId];
		if (!rootPane) return null;

		if (focusedPaneId) {
			const found = terminalStore.findPaneNode(rootPane, focusedPaneId);
			if (found && found.type === 'leaf') return focusedPaneId;
		}

		const firstLeaf = terminalStore.getFirstLeaf(rootPane);
		return firstLeaf?.id ?? null;
	}

	onMount(async () => {
		terminalStore.init();
		if (!/Nerd Font|MesloLGS NF/.test(terminalStore.config.fontFamily)) {
			terminalStore.updateConfig({
				fontFamily: `"Hack Nerd Font", "JetBrainsMono Nerd Font", "MesloLGS NF", "FiraCode Nerd Font", ${terminalStore.config.fontFamily}`
			});
		}
		// A persisted tab can be labelled "Shell" while its PTY is currently
		// running an agent process. Opening Terminal must always begin with a
		// fresh interactive shell, never a reused agent session.
		terminalStore.createTab('shell', { mode: 'local' });
		window.addEventListener('keydown', handleKeydown);

		// Initialize chatModelStore for AI providers
		await chatModelStore.loadUserSettings();
		await chatModelStore.loadModels();
	});

	onDestroy(() => {
		window.removeEventListener('keydown', handleKeydown);
	});
</script>

<div class="terminal-app" style="background: {bgColor}">
	<!-- Environment Mode Bar (top) -->
	<TerminalEnvironmentBar
		currentMode={currentEnvMode}
		environmentInfo={currentEnvInfo}
		onModeChange={handleEnvironmentChange}
		onExitSandbox={handleExitSandbox}
		{creatingSandbox}
	/>

	<!-- Provider Bar -->
	<TerminalProviderBar
		activeProvider={currentProvider}
		{config}
		environmentMode={currentEnvMode}
		onProviderChange={handleProviderChange}
		onConfigChange={handleConfigChange}
		onLaunchAgent={handleLaunchAgent}
		onStopAgent={handleStopAgent}
	/>

	<!-- Focus Mode Bar (only when AI provider active) -->
	<TerminalFocusBar
		provider={currentProvider}
		{activeFocusMode}
		onFocusModeChange={handleFocusModeChange}
	/>

	<!-- Tab Bar -->
	<TerminalTabBar
		{tabs}
		{activeTabId}
		onSwitchTab={handleSwitchTab}
		onCloseTab={handleCloseTab}
		onNewTab={handleNewTab}
	/>

	<!-- Pane Area - one pane tree per tab, only active shown -->
	<div class="pane-area">
		{#each tabs as tab (tab.id)}
			{@const paneTree = terminalStore.panes[tab.rootPaneId]}
			{@const tabEnvInfo = terminalStore.getEnvironmentInfo(tab.id)}
			{#if paneTree}
				<div
					class="tab-pane"
					class:active={tab.id === activeTabId}
				>
					<TerminalSplitContainer
						node={paneTree}
						{config}
						{activeFocusMode}
						environmentInfo={tabEnvInfo}
						environmentMode={tabEnvInfo.mode}
						onSessionCreated={handleSessionCreated}
						onFocus={handlePaneFocus}
						onShellReady={handleShellReady}
					/>
				</div>
			{/if}
		{/each}
	</div>

	<!-- Sandbox Analysis Modal -->
	<TerminalSandboxAnalysis
		sessionId={activeTab?.sessionId ?? ''}
		visible={showSandboxAnalysis}
		onMerge={handleSandboxMerge}
		onStay={handleSandboxStay}
	/>
</div>

<style>
	.terminal-app {
		display: flex;
		flex-direction: column;
		width: 100%;
		height: 100%;
		overflow: hidden;
	}

	.pane-area {
		flex: 1;
		position: relative;
		overflow: hidden;
	}

	.tab-pane {
		position: absolute;
		inset: 0;
		display: none;
	}

	.tab-pane.active {
		display: flex;
	}
</style>
