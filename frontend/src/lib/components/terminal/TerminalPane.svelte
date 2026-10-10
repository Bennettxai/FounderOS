<script lang="ts">
	import type { EnvironmentInfo, PaneLeaf, TerminalConfig } from '$lib/stores/terminal/terminalTypes';
	import TerminalShell from './TerminalShell.svelte';
	import TerminalAIChat from './TerminalAIChat.svelte';
	import MonacoEditor from '$lib/editor/MonacoEditor.svelte';
	import { terminalStore } from '$lib/stores/terminal';

	interface Props {
		pane: PaneLeaf;
		config: TerminalConfig;
		activeFocusMode?: string;
		environmentInfo?: EnvironmentInfo;
		environmentMode?: string;
		onSessionCreated?: (paneId: string, sessionId: string) => void;
		onFocus?: (paneId: string) => void;
		onShellReady?: (paneId: string, write: (data: string) => void) => void;
	}

	let { pane, config, activeFocusMode, environmentInfo, environmentMode = 'local', onSessionCreated, onFocus, onShellReady }: Props = $props();

	let shellRef: TerminalShell | undefined = $state();
	let aiChatRef: TerminalAIChat | undefined = $state();
	let monacoRef: MonacoEditor | undefined = $state();

	// Context menu state
	let showContextMenu = $state(false);
	let contextMenuX = $state(0);
	let contextMenuY = $state(0);

	// Shell stays mounted but hidden when in AI mode (keeps WebSocket alive)
	const showShell = $derived(pane.mode === 'shell');
	const showAI = $derived(pane.mode === 'ai');
	const showMonaco = $derived(pane.mode === 'monaco');

	// Keep shell alive if this pane was ever a shell (provider is shell or was shell)
	const shellMounted = $derived(pane.mode === 'shell' || pane.provider === 'shell');

	// Is this the root pane (can't close)?
	const isRootLeaf = $derived.by(() => {
		// Check if there's a parent split — if not, this is root
		const tree = Object.values(terminalStore.panes).find(t => {
			if (t.type === 'leaf') return t.id === pane.id;
			return !!findInTree(t, pane.id);
		});
		return tree?.type === 'leaf';
	});

	function findInTree(node: { type: string; id: string; children?: unknown[] }, id: string): boolean {
		if (node.id === id) return true;
		if (node.type === 'split' && Array.isArray(node.children)) {
			return node.children.some((c: unknown) => findInTree(c as typeof node, id));
		}
		return false;
	}

	function handleContextMenu(e: MouseEvent) {
		e.preventDefault();
		contextMenuX = e.clientX;
		contextMenuY = e.clientY;
		showContextMenu = true;

		// Close on next click anywhere
		const close = () => {
			showContextMenu = false;
			window.removeEventListener('click', close);
			window.removeEventListener('contextmenu', close);
		};
		setTimeout(() => {
			window.addEventListener('click', close);
			window.addEventListener('contextmenu', close);
		}, 0);
	}

	function ctxSplit(direction: 'horizontal' | 'vertical') {
		terminalStore.splitPane(pane.id, direction);
		showContextMenu = false;
	}

	function ctxSwitchMode(mode: 'shell' | 'ai' | 'monaco') {
		const provider = mode === 'shell' ? 'shell' : pane.provider === 'shell' ? 'osa' : pane.provider;
		terminalStore.setPaneMode(pane.id, mode, provider);
		showContextMenu = false;
	}

	function ctxClosePane() {
		terminalStore.closePaneInSplit(pane.id);
		showContextMenu = false;
	}

	function handleMonacoChange(newValue: string) {
		if (pane.filePath) {
			terminalStore.setPaneFile(pane.id, pane.filePath, newValue);
		}
	}

	function handleMonacoSave(savedValue: string) {
		console.log('[TerminalPane] Monaco save:', pane.filePath, savedValue.length, 'chars');
	}

	export function focus() {
		if (showShell) shellRef?.focus();
		else if (showAI) aiChatRef?.focus();
		else if (showMonaco) monacoRef?.focus();
	}

	export function writeData(data: string) {
		shellRef?.writeData(data);
	}

	// Notify parent when the shell ref is bound so it can register this pane's write function
	// Cleanup removes the stale entry when the pane unmounts or shell changes
	$effect(() => {
		if (shellRef && onShellReady) {
			onShellReady(pane.id, (data: string) => shellRef?.writeData(data));
			return () => {
				onShellReady(pane.id, null as unknown as (data: string) => void);
			};
		}
	});
</script>

<div
	class="terminal-pane"
	onclick={() => onFocus?.(pane.id)}
	onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') onFocus?.(pane.id); }}
	oncontextmenu={handleContextMenu}
	role="group"
	tabindex="-1"
	aria-label="Terminal pane"
>
	{#if shellMounted}
		{#key environmentMode}
			<TerminalShell
				bind:this={shellRef}
				paneId={pane.id}
				resumeSessionId={pane.sessionId ?? undefined}
				{config}
				visible={showShell}
				{environmentInfo}
				{environmentMode}
				{onSessionCreated}
				onFocus={onFocus}
			/>
		{/key}
	{/if}

	{#if showAI}
		<TerminalAIChat
			bind:this={aiChatRef}
			paneId={pane.id}
			provider={pane.provider}
			themeId={config.theme}
			visible={showAI}
			focusMode={activeFocusMode}
			onFocus={onFocus}
		/>
	{/if}

	{#if showMonaco}
		<div class="monaco-wrapper">
			{#if pane.filePath}
				<div class="monaco-header">
					<span class="file-path">{pane.filePath}</span>
					<button
						class="close-editor-btn"
						onclick={() => terminalStore.setPaneMode(pane.id, 'shell', 'shell')}
						aria-label="Close editor"
					>
						&times;
					</button>
				</div>
				<div class="monaco-body">
					<MonacoEditor
						bind:this={monacoRef}
						value={pane.fileContent ?? ''}
						filename={pane.filePath}
						readonly={false}
						onChange={handleMonacoChange}
						onSave={handleMonacoSave}
					/>
				</div>
			{:else}
				<div class="monaco-empty">
					<p class="empty-title">Code Editor</p>
					<p class="empty-hint">No file open. Use the terminal to open a file:</p>
					<code class="empty-cmd">edit &lt;filepath&gt;</code>
				</div>
			{/if}
		</div>
	{/if}

	<!-- Context Menu -->
	{#if showContextMenu}
		<div
			class="context-menu"
			style="left: {contextMenuX}px; top: {contextMenuY}px;"
			role="menu"
		>
			<button class="ctx-item" role="menuitem" onclick={() => ctxSplit('horizontal')}>Split Horizontal</button>
			<button class="ctx-item" role="menuitem" onclick={() => ctxSplit('vertical')}>Split Vertical</button>
			<div class="ctx-divider"></div>
			<button class="ctx-item" role="menuitem" onclick={() => ctxSwitchMode('shell')} disabled={pane.mode === 'shell'}>Switch to Shell</button>
			<button class="ctx-item" role="menuitem" onclick={() => ctxSwitchMode('ai')} disabled={pane.mode === 'ai'}>Switch to AI</button>
			<button class="ctx-item" role="menuitem" onclick={() => ctxSwitchMode('monaco')} disabled={pane.mode === 'monaco'}>Switch to Editor</button>
			<div class="ctx-divider"></div>
			<button class="ctx-item danger" role="menuitem" onclick={ctxClosePane} disabled={isRootLeaf}>Close Pane</button>
		</div>
	{/if}
</div>

<style>
	.terminal-pane {
		width: 100%;
		height: 100%;
		position: relative;
		overflow: hidden;
		outline: none;
	}

	/* ─── Context Menu ──────────────────────────────────────────────────── */
	.context-menu {
		position: fixed;
		background: #1a1a1a;
		border: 1px solid #333;
		border-radius: 6px;
		padding: 4px;
		min-width: 160px;
		z-index: 100;
		box-shadow: 0 4px 16px rgba(0,0,0,0.5);
	}

	.ctx-item {
		display: block;
		width: 100%;
		padding: 6px 12px;
		background: transparent;
		border: none;
		color: #ccc;
		font-family: 'SF Mono', monospace;
		font-size: 11px;
		text-align: left;
		cursor: pointer;
		border-radius: 3px;
	}

	.ctx-item:hover:not(:disabled) { background: #2a2a2a; }
	.ctx-item:disabled { color: #555; cursor: default; }
	.ctx-item.danger:hover:not(:disabled) { background: #ff555522; color: #ff5555; }

	.ctx-divider {
		height: 1px;
		background: #333;
		margin: 3px 4px;
	}

	/* ─── Monaco ────────────────────────────────────────────────────────── */
	.monaco-wrapper {
		display: flex;
		flex-direction: column;
		width: 100%;
		height: 100%;
		background: #1e1e1e;
	}

	.monaco-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 4px 12px;
		background: #252526;
		border-bottom: 1px solid #333;
		flex-shrink: 0;
		height: 28px;
	}

	.file-path {
		font-family: 'SF Mono', 'Monaco', 'Fira Code', monospace;
		font-size: 11px;
		color: #ccc;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.close-editor-btn {
		background: transparent;
		border: none;
		color: #666;
		font-size: 16px;
		cursor: pointer;
		padding: 0 4px;
		border-radius: 3px;
		line-height: 1;
	}

	.close-editor-btn:hover {
		background: #333;
		color: #ff5555;
	}

	.monaco-body {
		flex: 1;
		overflow: hidden;
		min-height: 0;
	}

	.monaco-empty {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		height: 100%;
		color: #555;
		font-family: 'SF Mono', monospace;
		font-size: 13px;
		gap: 8px;
	}

	.monaco-empty .empty-title {
		margin: 0;
		font-size: 16px;
		font-weight: 600;
		color: #888;
	}

	.monaco-empty .empty-hint {
		margin: 0;
		font-size: 12px;
		opacity: 0.7;
	}

	.monaco-empty .empty-cmd {
		background: #2a2a2a;
		padding: 4px 12px;
		border-radius: 4px;
		color: #00ff00;
		font-size: 12px;
	}
</style>
