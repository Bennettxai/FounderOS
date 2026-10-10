<script lang="ts">
	import { onMount } from 'svelte';
	import type { TerminalProvider } from '$lib/stores/terminal/terminalTypes';
	import { PROVIDER_CONFIGS } from '$lib/stores/terminal/terminalTypes';
	import { getThemeOptions } from './themes';
	import type { TerminalConfig } from '$lib/stores/terminal/terminalTypes';
	import { chatModelStore } from '$lib/stores/chat/chatModelStore.svelte';

	interface Props {
		activeProvider: TerminalProvider;
		config: TerminalConfig;
		onProviderChange: (provider: TerminalProvider) => void;
		onConfigChange: (config: Partial<TerminalConfig>) => void;
		environmentMode?: string;
		onLaunchAgent?: (agent: string) => void;
		onStopAgent?: () => void;
	}

	let { activeProvider, config, onProviderChange, onConfigChange, environmentMode, onLaunchAgent, onStopAgent }: Props = $props();

	let showSettings = $state(false);
	let agentLaunched = $state(false);
	const themeOptions = getThemeOptions();

	// Which agent CLIs are actually installed on the user's PATH. null = unknown
	// (web browser, or detection not run yet) so we never flag a CLI as missing
	// unless the desktop bridge actually told us it is absent.
	let cliDetected = $state<Record<string, boolean> | null>(null);

	// Install commands shown when a provider's CLI is not on PATH.
	const CLI_INSTALL: Record<string, string> = {
		claude: 'npm i -g @anthropic-ai/claude-code',
		codex: 'npm i -g @openai/codex',
		osa: 'npm i -g @osa-agent/cli',
		ollama: 'https://ollama.com/download'
	};

	onMount(async () => {
		try {
			const api = (window as unknown as {
				electron?: { agents?: { detect?: () => Promise<Record<string, boolean>> } };
			}).electron;
			if (api?.agents?.detect) {
				cliDetected = await api.agents.detect();
			}
		} catch {
			// leave cliDetected null - no bridge, no marking
		}
	});

	// A provider's CLI is "missing" only when detection ran and reported false.
	// 'shell' is not an agent CLI, so it is never flagged.
	function cliMissing(id: string): boolean {
		return cliDetected !== null && id !== 'shell' && cliDetected[id] === false;
	}

	const activeCliMissing = $derived(cliMissing(activeProvider));
	const activeInstallCmd = $derived(CLI_INSTALL[activeProvider] ?? '');
	const activeInstallIsUrl = $derived(activeInstallCmd.startsWith('http'));

	const canLaunchAgent = $derived(
		activeProvider !== 'shell' &&
		(environmentMode === 'sandbox' || environmentMode === 'local')
	);

	const agentLabel = $derived.by(() => {
		switch (activeProvider) {
			case 'claude': return 'Launch Claude Code';
			case 'codex': return 'Launch Codex';
			case 'ollama': return 'Launch Ollama';
			case 'osa': return 'Launch OSA';
			default: return '';
		}
	});

	// Provider → status: shell always available, others check chatModelStore
	const providerStatus = $derived.by(() => {
		const status: Record<string, boolean> = { shell: true };
		const configured = chatModelStore.configuredProviders;
		status.osa = configured.has('anthropic') || configured.has('ollama_local');
		status.claude = configured.has('anthropic');
		status.codex = configured.has('groq') || configured.has('openai');
		status.ollama = chatModelStore.installedModels.length > 0;
		return status;
	});

	// Filter models by current provider context
	const filteredModels = $derived.by(() => {
		const all = chatModelStore.models;
		if (activeProvider === 'shell') return [];
		if (activeProvider === 'ollama') return all.filter(m => m.type === 'local');
		if (activeProvider === 'claude' || activeProvider === 'osa') {
			return all.filter(m => m.id.includes('claude') || m.id.includes('sonnet') || m.id.includes('opus') || m.id.includes('haiku'));
		}
		if (activeProvider === 'codex') {
			return all.filter(m => m.id.includes('gpt') || m.id.includes('groq') || m.id.includes('qwen'));
		}
		return all;
	});
</script>

<div class="provider-bar">
	<div class="providers">
		{#each PROVIDER_CONFIGS as p (p.id)}
			<button
				class="provider-pill"
				class:active={p.id === activeProvider}
				style="--pill-color: {p.color}"
				onclick={() => { agentLaunched = false; onProviderChange(p.id); }}
				title="{p.label}{p.shortcut ? ` (Ctrl+${p.shortcut})` : ''}"
			>
				<span class="status-dot" class:available={providerStatus[p.id]} class:unavailable={!providerStatus[p.id]}></span>
				{p.label}
				{#if cliMissing(p.id)}
					<span class="cli-warn" title="{p.label} CLI not installed">!</span>
				{/if}
			</button>
		{/each}
	</div>

	<div class="actions">
		{#if canLaunchAgent && agentLabel}
			<button
				class="launch-agent-btn"
				class:launched={agentLaunched}
				onclick={() => {
					if (agentLaunched) {
						agentLaunched = false;
						onStopAgent?.();
						return;
					}
					onLaunchAgent?.(activeProvider);
					agentLaunched = true;
				}}
				aria-label={agentLabel}
			>
				{#if agentLaunched}
					<span class="agent-dot"></span> Stop Agent
				{:else}
					{agentLabel}
				{/if}
			</button>
		{/if}
		<button
			class="settings-btn"
			class:active={showSettings}
			onclick={() => showSettings = !showSettings}
			aria-label="Terminal settings"
		>
			&#9881;
		</button>
	</div>
</div>

{#if activeCliMissing && activeInstallCmd}
	<div class="cli-hint">
		<span class="cli-hint-dot"></span>
		<span class="cli-hint-text">
			The <strong>{activeProvider}</strong> CLI is not installed on this machine.
		</span>
		{#if activeInstallIsUrl}
			<span class="cli-hint-label">Download:</span>
			<a class="cli-hint-cmd" href={activeInstallCmd} target="_blank" rel="noreferrer">{activeInstallCmd}</a>
		{:else}
			<span class="cli-hint-label">Install:</span>
			<code class="cli-hint-cmd">{activeInstallCmd}</code>
		{/if}
	</div>
{/if}

{#if showSettings}
	<div class="settings-panel">
		<!-- Terminal Settings -->
		<div class="settings-section">
			<div class="setting-row">
				<label for="term-theme">Theme</label>
				<select
					id="term-theme"
					value={config.theme}
					onchange={(e) => onConfigChange({ theme: (e.target as HTMLSelectElement).value })}
				>
					{#each themeOptions as opt (opt.id)}
						<option value={opt.id}>{opt.name}</option>
					{/each}
				</select>
			</div>

			<div class="setting-row">
				<label for="term-fontsize">Font</label>
				<input
					id="term-fontsize"
					type="range"
					min="10"
					max="24"
					step="1"
					value={config.fontSize}
					oninput={(e) => onConfigChange({ fontSize: Number((e.target as HTMLInputElement).value) })}
				/>
				<span class="value">{config.fontSize}px</span>
			</div>

			<div class="setting-row">
				<label for="term-cursor">Cursor</label>
				<select
					id="term-cursor"
					value={config.cursorStyle}
					onchange={(e) => onConfigChange({ cursorStyle: (e.target as HTMLSelectElement).value as 'block' | 'underline' | 'bar' })}
				>
					<option value="block">Block</option>
					<option value="underline">Underline</option>
					<option value="bar">Bar</option>
				</select>
			</div>

			<div class="setting-row">
				<label for="term-blink">Blink</label>
				<input
					id="term-blink"
					type="checkbox"
					checked={config.cursorBlink}
					onchange={(e) => onConfigChange({ cursorBlink: (e.target as HTMLInputElement).checked })}
				/>
			</div>
		</div>

		<!-- AI Settings (only when not in shell mode) -->
		{#if activeProvider !== 'shell'}
			<div class="settings-divider"></div>
			<div class="settings-section">
				<!-- Model Selector -->
				{#if filteredModels.length > 0}
					<div class="setting-row">
						<label for="ai-model">Model</label>
						<select
							id="ai-model"
							value={chatModelStore.selectedModel}
							onchange={(e) => chatModelStore.selectModel((e.target as HTMLSelectElement).value)}
						>
							{#each filteredModels as model (model.id)}
								<option value={model.id}>
									{model.name}{model.type === 'local' ? ' (local)' : ''}
								</option>
							{/each}
						</select>
					</div>
				{/if}

				<!-- Temperature -->
				<div class="setting-row">
					<label for="ai-temp">Temp</label>
					<input
						id="ai-temp"
						type="range"
						min="0"
						max="2"
						step="0.1"
						value={chatModelStore.aiTemperature}
						oninput={(e) => chatModelStore.aiTemperature = Number((e.target as HTMLInputElement).value)}
					/>
					<span class="value">{chatModelStore.aiTemperature.toFixed(1)}</span>
				</div>

				<!-- Max Tokens -->
				<div class="setting-row">
					<label for="ai-tokens">Tokens</label>
					<input
						id="ai-tokens"
						type="range"
						min="256"
						max="32768"
						step="256"
						value={chatModelStore.aiMaxTokens}
						oninput={(e) => chatModelStore.aiMaxTokens = Number((e.target as HTMLInputElement).value)}
					/>
					<span class="value">{chatModelStore.aiMaxTokens >= 1024 ? `${(chatModelStore.aiMaxTokens / 1024).toFixed(0)}K` : chatModelStore.aiMaxTokens}</span>
				</div>

				<!-- Top-P -->
				<div class="setting-row">
					<label for="ai-topp">Top-P</label>
					<input
						id="ai-topp"
						type="range"
						min="0"
						max="1"
						step="0.05"
						value={chatModelStore.aiTopP}
						oninput={(e) => chatModelStore.aiTopP = Number((e.target as HTMLInputElement).value)}
					/>
					<span class="value">{chatModelStore.aiTopP.toFixed(2)}</span>
				</div>

				<!-- COT Toggle -->
				<div class="setting-row">
					<label for="ai-cot">COT</label>
					<input
						id="ai-cot"
						type="checkbox"
						checked={chatModelStore.useCOT}
						onchange={() => chatModelStore.toggleCOT()}
					/>
				</div>

				<!-- Show Usage Toggle -->
				<div class="setting-row">
					<label for="ai-usage">Usage</label>
					<input
						id="ai-usage"
						type="checkbox"
						checked={chatModelStore.showUsageInChat}
						onchange={(e) => chatModelStore.showUsageInChat = (e.target as HTMLInputElement).checked}
					/>
				</div>
			</div>
		{/if}
	</div>
{/if}

<style>
	.provider-bar {
		display: flex;
		align-items: center;
		justify-content: space-between;
		height: 30px;
		padding: 0 8px;
		background: #0d0d0d;
		border-bottom: 1px solid #1a1a1a;
		flex-shrink: 0;
	}

	.providers {
		display: flex;
		gap: 4px;
		align-items: center;
	}

	.provider-pill {
		display: flex;
		align-items: center;
		gap: 4px;
		padding: 3px 10px;
		border-radius: 12px;
		border: 1px solid #333;
		background: transparent;
		color: #888;
		font-family: 'SF Mono', monospace;
		font-size: 10px;
		font-weight: 500;
		cursor: pointer;
		transition: all 0.15s ease;
		text-transform: uppercase;
		letter-spacing: 0.5px;
	}

	.provider-pill:hover {
		border-color: var(--pill-color, #00ff00);
		color: var(--pill-color, #00ff00);
	}

	.provider-pill.active {
		background: var(--pill-color, #00ff00);
		color: #000;
		border-color: var(--pill-color, #00ff00);
		font-weight: 700;
	}

	.status-dot {
		width: 5px;
		height: 5px;
		border-radius: 50%;
		flex-shrink: 0;
	}

	.status-dot.available { background: #22c55e; }
	.status-dot.unavailable { background: #555; }

	.provider-pill.active .status-dot.available { background: #000; }
	.provider-pill.active .status-dot.unavailable { background: #00000066; }

	.cli-warn {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 11px;
		height: 11px;
		border-radius: 50%;
		background: #f59e0b;
		color: #000;
		font-size: 8px;
		font-weight: 700;
		line-height: 1;
		flex-shrink: 0;
	}

	.provider-pill.active .cli-warn { background: #7c4a03; color: #f59e0b; }

	.cli-hint {
		display: flex;
		align-items: center;
		gap: 8px;
		background: #111;
		border-bottom: 1px solid #222;
		padding: 5px 12px;
		font-family: 'SF Mono', monospace;
		font-size: 10px;
		color: #999;
		flex-shrink: 0;
		flex-wrap: wrap;
	}

	.cli-hint-dot {
		width: 6px;
		height: 6px;
		border-radius: 50%;
		background: #f59e0b;
		flex-shrink: 0;
	}

	.cli-hint-text strong { color: #ccc; text-transform: uppercase; }

	.cli-hint-label { color: #666; }

	.cli-hint-cmd {
		background: #1a1a1a;
		border: 1px solid #333;
		border-radius: 3px;
		padding: 1px 6px;
		color: #f59e0b;
		font-family: inherit;
		font-size: 10px;
		text-decoration: none;
	}

	.cli-hint-cmd:hover { border-color: #f59e0b; }

	.actions {
		display: flex;
		gap: 4px;
	}

	.settings-btn {
		background: transparent;
		border: none;
		color: #555;
		font-size: 14px;
		cursor: pointer;
		padding: 2px 6px;
		border-radius: 3px;
	}

	.settings-btn:hover,
	.settings-btn.active {
		background: #1a1a1a;
		color: #ccc;
	}

	.settings-panel {
		background: #111;
		border-bottom: 1px solid #222;
		padding: 8px 12px;
		display: flex;
		gap: 8px;
		flex-wrap: wrap;
		flex-shrink: 0;
	}

	.settings-section {
		display: flex;
		gap: 12px;
		flex-wrap: wrap;
	}

	.settings-divider {
		width: 1px;
		background: #333;
		align-self: stretch;
		margin: 0 4px;
	}

	.setting-row {
		display: flex;
		align-items: center;
		gap: 6px;
		font-family: 'SF Mono', monospace;
		font-size: 11px;
		color: #888;
	}

	.setting-row label {
		white-space: nowrap;
		color: #666;
		min-width: 32px;
	}

	.setting-row select {
		background: #1a1a1a;
		border: 1px solid #333;
		color: #ccc;
		border-radius: 3px;
		padding: 2px 4px;
		font-family: inherit;
		font-size: inherit;
		max-width: 140px;
	}

	.setting-row input[type="range"] {
		width: 70px;
		accent-color: #00ff00;
	}

	.setting-row input[type="checkbox"] {
		accent-color: #00ff00;
	}

	.value {
		min-width: 30px;
		text-align: right;
		color: #ccc;
		font-size: 10px;
	}

	.launch-agent-btn {
		display: flex;
		align-items: center;
		gap: 4px;
		padding: 3px 8px;
		border-radius: 4px;
		border: 1px solid #333;
		background: transparent;
		color: #888;
		font-family: 'SF Mono', monospace;
		font-size: 9px;
		font-weight: 500;
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.launch-agent-btn:hover {
		border-color: #22c55e;
		color: #22c55e;
	}

	.launch-agent-btn.launched {
		border-color: #22c55e55;
		color: #22c55e;
	}

	.agent-dot {
		width: 5px;
		height: 5px;
		border-radius: 50%;
		background: #22c55e;
		animation: pulse-agent 1.5s ease-in-out infinite;
	}

	@keyframes pulse-agent {
		0%, 100% { opacity: 0.4; }
		50% { opacity: 1; }
	}
</style>
