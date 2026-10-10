<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { createTerminalService, getCloudTerminalSession, type TerminalService } from '$lib/services/terminal.service';
	import { createElectronTerminal, isElectronTerminalAvailable } from '$lib/services/electron-terminal.service';
	import { getBusinessOSHome } from '$lib/services/businessOSHome';
	import { Terminal } from '@xterm/xterm';
	import { FitAddon } from '@xterm/addon-fit';
	import { WebLinksAddon } from '@xterm/addon-web-links';
	import { SearchAddon } from '@xterm/addon-search';
	import '@xterm/xterm/css/xterm.css';
	import { getThemeColors } from './themes';
	import type { EnvironmentInfo, TerminalConfig } from '$lib/stores/terminal/terminalTypes';

	interface Props {
		paneId: string;
		resumeSessionId?: string;
		config: TerminalConfig;
		visible?: boolean;
		environmentInfo?: EnvironmentInfo;
		environmentMode?: string;
		onSessionCreated?: (paneId: string, sessionId: string) => void;
		onFocus?: (paneId: string) => void;
	}

	let { paneId, resumeSessionId, config, visible = true, environmentInfo, environmentMode = 'local', onSessionCreated, onFocus }: Props = $props();

	// Cloud computer URL - resolved when switching to production mode
	let cloudUrl = $state<string | null>(null);

	let terminalContainer = $state<HTMLDivElement | undefined>(undefined);
	let xterm: Terminal | null = null;
	let fitAddon: FitAddon | null = null;
	let service: TerminalService | null = null;
	let resizeObserver: ResizeObserver | null = null;
	let initTimers: ReturnType<typeof setTimeout>[] = [];
	let initialized = false;
	let isConnected = $state(false);
	let connectionError = $state<string | null>(null);

	async function initTerminal() {
		if (!terminalContainer || initialized) return;
		initialized = true;

		const themeColors = getThemeColors(config.theme);

		xterm = new Terminal({
			fontFamily: config.fontFamily,
			fontSize: config.fontSize,
			lineHeight: 1.1,
			letterSpacing: -0.25,
			cursorBlink: config.cursorBlink,
			cursorStyle: config.cursorStyle,
			convertEol: false,
			theme: themeColors,
			allowProposedApi: true
		});

		fitAddon = new FitAddon();
		xterm.loadAddon(fitAddon);
		xterm.loadAddon(new WebLinksAddon());
		xterm.loadAddon(new SearchAddon());

		xterm.open(terminalContainer);

		const t1 = setTimeout(() => xterm?.focus(), 0);
		const t2 = setTimeout(() => fitAddon?.fit(), 100);
		initTimers.push(t1, t2);

		// Input/resize handlers are registered AFTER the connection is established
		// to avoid duplicate handlers between cloud and local modes.

		// Remote modes use pre-authenticated WebSocket URLs minted by the backend.
		let cloudWsUrl: string | undefined = undefined;
		if (environmentMode === 'sandbox' && environmentInfo?.miosaTerminalWsUrl) {
			cloudWsUrl = environmentInfo.miosaTerminalWsUrl;
			xterm?.write(`\x1b[36m[Connected to BusinessOS MIOSA sandbox ${environmentInfo.miosaSandboxId?.slice(0, 8) ?? ''}]\x1b[0m\r\n`);
		} else if (environmentMode === 'sandbox' && environmentInfo?.sandboxRemoteState === 'remote-error') {
			xterm?.write(`\x1b[31m[MIOSA sandbox error: ${environmentInfo.miosaError ?? 'could not create sandbox'}]\x1b[0m\r\n`);
			xterm?.write('\x1b[33m[Opening local fallback sandbox]\x1b[0m\r\n');
		}
		if (environmentMode === 'production') {
			xterm?.write('\x1b[36m[Connecting to cloud computer...]\x1b[0m\r\n');
			try {
				const session = await getCloudTerminalSession();
				if (session?.mode === 'cloud' && session.ws_url) {
					cloudWsUrl = session.ws_url;
					cloudUrl = session.ws_url;
					xterm?.write(`\x1b[36m[Session: ${session.session_id?.slice(0, 8)}... on ${session.slug}]\x1b[0m\r\n`);
				} else {
					const msg = session?.message || 'No active cloud computer';
					xterm?.write(`\x1b[33m[${msg} - falling back to local]\x1b[0m\r\n`);
				}
			} catch {
				xterm?.write('\x1b[33m[Could not reach cloud - falling back to local]\x1b[0m\r\n');
			}
		}

		if (cloudWsUrl) {
			// Direct WebSocket to cloud VM - bypass the normal terminal service
			try {
				const ws = new WebSocket(cloudWsUrl);
				ws.binaryType = 'arraybuffer';
				ws.onopen = () => {
					isConnected = true;
					connectionError = null;
					xterm?.write(environmentMode === 'sandbox' ? '\x1b[32m[Sandbox terminal attached]\x1b[0m\r\n' : '\x1b[32m[Connected to cloud computer]\x1b[0m\r\n');
					onSessionCreated?.(paneId, environmentInfo?.miosaSandboxId ?? 'cloud');
				};
				ws.onmessage = (event) => {
					if (event.data instanceof ArrayBuffer) {
						xterm?.write(new Uint8Array(event.data));
					} else {
						xterm?.write(event.data);
					}
				};
				ws.onclose = () => {
					isConnected = false;
					xterm?.write('\r\n\x1b[31m[Cloud terminal disconnected]\x1b[0m\r\n');
				};
				ws.onerror = () => {
					connectionError = 'Cloud terminal connection failed';
					xterm?.write('\r\n\x1b[31m[Cloud terminal error]\x1b[0m\r\n');
				};
				// Store reference for cleanup (input handler registered below, after branching)
				service = {
					isConnected: () => ws.readyState === WebSocket.OPEN,
					sendInput: (data: string) => { if (ws.readyState === WebSocket.OPEN) ws.send(data); },
					resize: (cols: number, rows: number) => {
						if (ws.readyState === WebSocket.OPEN) {
							ws.send(JSON.stringify({ type: 'resize', cols, rows }));
						}
					},
					disconnect: () => ws.close(),
					connect: () => {},
				} as unknown as TerminalService;
			} catch (err) {
				xterm?.write(`\r\n\x1b[31m[Failed to connect: ${err}]\x1b[0m\r\n`);
			}
		} else {
			// Local mode. In the desktop app, connect to a real PTY on the user's
			// own machine via IPC (node-pty in the Electron main process). On the
			// web, fall back to the WebSocket terminal service.
			const handlers = {
				onData: (data: string) => {
					xterm?.write(data);
				},
				onConnect: (sessionId: string) => {
					isConnected = true;
					connectionError = null;
					onSessionCreated?.(paneId, sessionId);
					if (environmentMode === 'sandbox' && !resumeSessionId) {
						const sandboxId = environmentInfo?.sandboxId ?? `sbx-${paneId.slice(0, 8)}`;
						xterm?.write(`\x1b[33m[BusinessOS sandbox ${sandboxId} - isolated workspace, ready for MIOSA attach]\x1b[0m\r\n`);
					}

					const svc = currentService;
					const t3 = setTimeout(() => {
						if (xterm && fitAddon && svc?.isConnected()) {
							const dims = fitAddon.proposeDimensions();
							if (dims) svc.resize(dims.cols, dims.rows);
						}
					}, 150);
					initTimers.push(t3);
				},
				onDisconnect: () => {
					isConnected = false;
					xterm?.write('\r\n\x1b[31m[Disconnected]\x1b[0m\r\n');
				},
				onError: (error: string) => {
					// A local PTY failure (e.g. node-pty could not load - arch
					// mismatch / missing native dep) arrives here as an error
					// string. Render a clear, actionable message instead of a
					// blank pane or a cryptic [Error: ...].
					if (isLocalUnavailableReason(error)) {
						writeLocalTerminalUnavailable(error);
					} else {
						connectionError = error;
						xterm?.write(`\r\n\x1b[31m[Error: ${error}]\x1b[0m\r\n`);
					}
				}
			};

			// In the desktop app, verify the local terminal native module can
			// actually run on this machine before spawning a shell. If node-pty
			// failed to load (e.g. an arm64 build under Rosetta on an Intel Mac),
			// surface a clear message rather than a silent, dead pane.
			if (isElectronTerminalAvailable()) {
				const avail = await checkLocalTerminalAvailable();
				if (!avail.available) {
					writeLocalTerminalUnavailable(avail.reason);
					return;
				}
			}

			// Open the shell in the BusinessOS home dir (auto-detected, user
			// overridable) instead of the OS home, so the terminal lands where
			// the user's BusinessOS lives.
			const cwd = isElectronTerminalAvailable() ? await resolveTerminalCwd() : undefined;
			const dims = { cols: xterm.cols, rows: xterm.rows };
			const currentService = isElectronTerminalAvailable()
				? createElectronTerminal(handlers, {
					...dims,
					cwd,
					shell: environmentMode === 'local' ? 'auto' : undefined,
					resumeId: resumeSessionId,
					env: { ...buildTerminalEnv(), TMUX: '' }
				})
				: createTerminalService(handlers, { ...dims, shell: 'zsh' });

			service = currentService as unknown as TerminalService;
			service.connect();
		}

		// Register unified input/resize handlers (works for both cloud and local)
		xterm.onData((data) => {
			if (service?.isConnected()) {
				service.sendInput(data);
			}
		});

		xterm.onResize(({ cols, rows }) => {
			if (service?.isConnected()) {
				service.resize(cols, rows);
			}
		});
	}

	async function resolveTerminalCwd(): Promise<string> {
		const home = await getBusinessOSHome();
		if (environmentMode !== 'sandbox') return home;
		const sandboxId = environmentInfo?.sandboxId ?? `sbx-${paneId.slice(0, 8)}`;
		return environmentInfo?.workspacePath ?? `${home}/.businessos/sandboxes/${sandboxId}`;
	}

	function buildTerminalEnv(): Record<string, string> {
		const env: Record<string, string> = {
			BUSINESSOS_TERMINAL_ENV: environmentMode
		};

		if (environmentMode === 'sandbox') {
			const sandboxId = environmentInfo?.sandboxId ?? `sbx-${paneId.slice(0, 8)}`;
			env.BUSINESSOS_SANDBOX_ID = sandboxId;
			env.BUSINESSOS_SANDBOX_MODE = 'miosa';
			env.BUSINESSOS_SANDBOX_REMOTE_STATE = environmentInfo?.sandboxRemoteState ?? 'local-ready';
			env.BUSINESSOS_MIOSA_ACCOUNT_SOURCE = environmentInfo?.miosaAccountSource ?? 'local';
			if (environmentInfo?.workspacePath) {
				env.BUSINESSOS_SANDBOX_WORKSPACE = environmentInfo.workspacePath;
			}
			if (environmentInfo?.miosaSandboxId) {
				env.BUSINESSOS_MIOSA_SANDBOX_ID = environmentInfo.miosaSandboxId;
			}
			if (environmentInfo?.miosaWorkspaceId) {
				env.BUSINESSOS_MIOSA_WORKSPACE_ID = environmentInfo.miosaWorkspaceId;
			}
			if (environmentInfo?.miosaProjectId) {
				env.BUSINESSOS_MIOSA_PROJECT_ID = environmentInfo.miosaProjectId;
			}
			if (environmentInfo?.miosaPreviewUrl) {
				env.BUSINESSOS_MIOSA_PREVIEW_URL = environmentInfo.miosaPreviewUrl;
			}
			if (environmentInfo?.miosaDeploymentUrl) {
				env.BUSINESSOS_MIOSA_DEPLOYMENT_URL = environmentInfo.miosaDeploymentUrl;
			}
			if (environmentInfo?.miosaAttribution?.externalWorkspaceId) {
				env.BUSINESSOS_EXTERNAL_WORKSPACE_ID = environmentInfo.miosaAttribution.externalWorkspaceId;
			}
			if (environmentInfo?.miosaAttribution?.externalWorkspaceSlug) {
				env.BUSINESSOS_EXTERNAL_WORKSPACE_SLUG = environmentInfo.miosaAttribution.externalWorkspaceSlug;
			}
			if (environmentInfo?.miosaAttribution?.externalUserId) {
				env.BUSINESSOS_EXTERNAL_USER_ID = environmentInfo.miosaAttribution.externalUserId;
			}
			if (environmentInfo?.miosaAttribution?.externalProjectId) {
				env.BUSINESSOS_EXTERNAL_PROJECT_ID = environmentInfo.miosaAttribution.externalProjectId;
			}
		}

		return env;
	}

	// --- Local terminal (node-pty) availability + failure messaging ---
	// Keep the local-terminal failure path LOUD: a native-module load failure
	// (arch mismatch / missing dep) shows a clear message in the pane instead
	// of silently doing nothing.

	function isLocalUnavailableReason(reason: string): boolean {
		return /native module|arch|could not load|unavailable/i.test(reason);
	}

	// Probe the Electron main process for whether the local PTY can run here.
	// Defensive access: works whether or not the preload exposes the channel;
	// if it is absent we assume available and let create() surface real errors.
	async function checkLocalTerminalAvailable(): Promise<{ available: boolean; reason?: string }> {
		const api = (
			window as unknown as {
				electron?: {
					terminal?: {
						available?: () => Promise<{ available: boolean; reason?: string }>;
					};
				};
			}
		).electron?.terminal;
		if (typeof api?.available === "function") {
			try {
				return await api.available();
			} catch {
				// fall through - assume available
			}
		}
		return { available: true };
	}

	// Write a clear, actionable message into the xterm view and flag the error.
	function writeLocalTerminalUnavailable(reason?: string) {
		const detail =
			reason && reason.trim().length > 0
				? reason
				: "the local terminal native module could not load on this system";
		connectionError = "Local terminal unavailable";
		xterm?.write(`\r\n\x1b[31m[Local terminal unavailable on this system: ${detail}.]\x1b[0m\r\n`);
		xterm?.write("\x1b[33m[If you are on an Intel Mac, install the Intel build of BusinessOS.]\x1b[0m\r\n");
	}

	function handleResize() {
		if (fitAddon && xterm && visible) {
			fitAddon.fit();
		}
	}

	export function focus() {
		xterm?.focus();
		const textarea = terminalContainer?.querySelector('.xterm-helper-textarea') as HTMLTextAreaElement;
		if (textarea) textarea.focus();
	}

	export function writeData(data: string) {
		if (service?.isConnected()) {
			service.sendInput(data);
		}
	}

	// React to config changes (only theme/font, not cursor blink)
	$effect(() => {
		if (xterm) {
			const themeColors = getThemeColors(config.theme);
			xterm.options.theme = themeColors;
			xterm.options.fontFamily = config.fontFamily;
			xterm.options.fontSize = config.fontSize;
			xterm.options.cursorBlink = config.cursorBlink;
			xterm.options.cursorStyle = config.cursorStyle;
		}
	});

	// React to visibility changes
	$effect(() => {
		if (visible && fitAddon && xterm) {
			setTimeout(() => fitAddon?.fit(), 50);
		}
	});

	// Mode switching is handled by {#key environmentMode} in TerminalPane -
	// it destroys and recreates this component on mode change, so no $effect needed.

	onMount(() => {
		initTerminal();
		window.addEventListener('resize', handleResize);

		resizeObserver = new ResizeObserver(() => handleResize());
		if (terminalContainer) {
			resizeObserver.observe(terminalContainer);
		}
	});

	onDestroy(() => {
		// Clear all pending timers
		initTimers.forEach(clearTimeout);
		initTimers = [];

		// Remove global listener
		window.removeEventListener('resize', handleResize);

		// Disconnect resize observer
		resizeObserver?.disconnect();
		resizeObserver = null;

		// DETACH (do not kill) the local PTY on unmount so the shell survives a
		// refresh and can be reconnected. The session is only killed when the tab
		// is closed (store.closeTab). Cloud/WebSocket services have no detach, so
		// fall back to disconnect for those.
		if (service) {
			const svc = service as unknown as { detach?: () => void; disconnect: () => void };
			if (typeof svc.detach === 'function') svc.detach();
			else svc.disconnect();
			service = null;
		}

		// Dispose xterm
		if (xterm) {
			xterm.dispose();
			xterm = null;
		}

		fitAddon = null;
		initialized = false;
	});
</script>

<div
	class="terminal-shell"
	class:hidden={!visible}
	onclick={() => { focus(); onFocus?.(paneId); }}
	onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') focus(); }}
	role="application"
	tabindex="-1"
	aria-label="Terminal shell"
>
	{#if connectionError}
		<div class="connection-status error">
			{connectionError}
		</div>
	{:else if !isConnected}
		<div class="connection-status connecting">
			Connecting...
		</div>
	{/if}
	<div
		class="xterm-container"
		bind:this={terminalContainer}
		tabindex="-1"
		aria-label="Terminal content"
	></div>
</div>

<style>
	.terminal-shell {
		width: 100%;
		height: 100%;
		position: relative;
		overflow: hidden;
	}

	.terminal-shell.hidden {
		display: none;
	}

	.xterm-container {
		width: 100%;
		height: 100%;
		padding: 12px 18px;
		box-sizing: border-box;
		outline: none;
		cursor: text;
	}

	.xterm-container :global(.xterm) {
		height: 100%;
	}

	.xterm-container :global(.xterm-viewport) {
		overflow-y: auto !important;
	}

	.xterm-container :global(.xterm-viewport::-webkit-scrollbar) {
		width: 6px;
	}

	.xterm-container :global(.xterm-viewport::-webkit-scrollbar-track) {
		background: transparent;
	}

	.xterm-container :global(.xterm-viewport::-webkit-scrollbar-thumb) {
		background: #333;
		border-radius: 3px;
	}

	.connection-status {
		position: absolute;
		top: 4px;
		right: 8px;
		padding: 2px 8px;
		border-radius: 3px;
		font-size: 11px;
		font-family: 'SF Mono', monospace;
		z-index: 10;
	}

	.connection-status.connecting {
		background: #333;
		color: #ffcc00;
	}

	.connection-status.error {
		background: #ff555533;
		color: #ff5555;
	}
</style>
