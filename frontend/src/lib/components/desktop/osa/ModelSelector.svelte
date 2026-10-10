<!--
	ModelSelector.svelte
	Compact model + provider picker for the OSA pill.
	Trigger: chip showing current model name.
	Dropdown: opens upward, glassmorphism — matches ModeSelector exactly.
-->
<script lang="ts">
	import { osaStore } from '$lib/stores/osa';
	import { getApiBaseUrl } from '$lib/api/base';

	interface Props {
		class?: string;
		compact?: boolean;
	}

	let { class: className = '', compact = false }: Props = $props();

	// ─── State ───────────────────────────────────────────────────────────────────

	let isOpen = $state(false);
	let dropdownElement: HTMLDivElement | undefined = $state(undefined);

	// Working copies — staged until Apply is pressed
	let stagedProvider = $state<string>('ollama');
	let stagedModel = $state<string>('');
	let stagedUrl = $state<string>('');

	// Apply test state: 'idle' | 'testing' | 'pass' | 'fail'
	let applyStatus = $state<'idle' | 'testing' | 'pass' | 'fail'>('idle');
	let applyError = $state<string>('');

	// ─── Derived ─────────────────────────────────────────────────────────────────

	let activeModel = $derived($osaStore.activeModel);
	let activeProvider = $derived($osaStore.activeProvider);

	let displayLabel = $derived(
		activeModel ? truncate(activeModel, 18) : 'Model'
	);

	// ─── Presets per provider ─────────────────────────────────────────────────────

	const CLOUD_PRESETS: Record<string, readonly string[]> = {
		'ollama-cloud': [
			'kimi-k2.6:cloud',
			'qwen3:235b', 'qwen3-coder-next:latest', 'deepseek-r1:671b', 'deepseek-v3.1-terminus:latest',
			'kimi-k2.5:latest', 'kimi-k2:latest', 'glm-5:latest', 'glm-4.7-flash:latest',
			'llama3.1:405b', 'nemotron-3-nano:latest',
		],
		anthropic: ['claude-opus-4-6', 'claude-sonnet-4-6', 'claude-sonnet-4-20250514', 'claude-haiku-4-5-20251001'],
		openai: ['gpt-5.4', 'gpt-5.3', 'gpt-5.3-codex', 'o3', 'o3-pro', 'o4-mini', 'gpt-4o'],
	} as const;

	// Local Ollama installed models — fetched dynamically
	let localOllamaModels = $state<string[]>([]);
	let ollamaFetchStatus = $state<'idle' | 'loading' | 'done' | 'error'>('idle');

	async function fetchLocalOllamaModels() {
		if (ollamaFetchStatus === 'loading' || ollamaFetchStatus === 'done') return;
		ollamaFetchStatus = 'loading';
		try {
			const res = await fetch('http://localhost:11434/api/tags', {
				signal: AbortSignal.timeout(3000),
			});
			if (!res.ok) throw new Error(`HTTP ${res.status}`);
			const data: { models?: { name: string; size: number }[] } = await res.json();
			if (data.models && Array.isArray(data.models)) {
				localOllamaModels = data.models.map((m) => m.name).sort();
			}
			ollamaFetchStatus = 'done';
		} catch {
			ollamaFetchStatus = 'error';
			// Fallback to static presets
			localOllamaModels = [];
		}
	}

	// Compute presets — local Ollama uses fetched models, others use static
	let currentPresets = $derived.by(() => {
		if (stagedProvider === 'ollama') {
			return localOllamaModels.length > 0 ? localOllamaModels : ['qwen3:32b', 'qwen3:8b', 'deepseek-r1:32b', 'llama3.1:8b'];
		}
		return CLOUD_PRESETS[stagedProvider] ?? [];
	});

	const PROVIDERS = ['ollama', 'ollama-cloud', 'anthropic', 'openai'] as const;

	const PROVIDER_LABELS: Record<string, string> = {
		ollama: 'Ollama (Local)',
		'ollama-cloud': 'Ollama (Cloud)',
		anthropic: 'Anthropic',
		openai: 'OpenAI',
	};

	// Whether the staged model matches the active store state
	let isApplied = $derived(
		stagedModel === (activeModel ?? '') &&
		stagedProvider === (activeProvider ?? 'ollama')
	);

	// ─── Helpers ─────────────────────────────────────────────────────────────────

	function truncate(str: string, max: number): string {
		return str.length > max ? str.slice(0, max - 1) + '…' : str;
	}

	function openDropdown() {
		// Seed staged values from store when opening
		stagedProvider = activeProvider ?? 'ollama';
		stagedModel = activeModel ?? '';
		applyStatus = 'idle';
		applyError = '';
		isOpen = true;
		// Fetch local models if ollama selected
		if (stagedProvider === 'ollama') fetchLocalOllamaModels();
	}

	function selectPreset(model: string) {
		stagedModel = model;
		applyStatus = 'idle';
		applyError = '';
	}

	async function applyConfig() {
		if (!stagedModel.trim()) return;
		applyStatus = 'testing';
		applyError = '';

		try {
			// 1. Apply the config to the store + POST to backend
			await osaStore.setModel(stagedProvider, stagedModel.trim(), stagedUrl.trim() || undefined);

			// 2. Verify by hitting the health endpoint — confirms OSA accepted the config
			const res = await fetch(`${getApiBaseUrl()}/osa/health`, {
				credentials: 'include',
				signal: AbortSignal.timeout(5000),
			});

			if (!res.ok) throw new Error(`Health check failed (HTTP ${res.status})`);

			const data = await res.json();
			if (data.status === 'unhealthy') {
				throw new Error(data.error || 'OSA is unhealthy');
			}

			applyStatus = 'pass';
			// Auto-close after showing green check
			setTimeout(() => { isOpen = false; applyStatus = 'idle'; }, 1200);
		} catch (err) {
			applyStatus = 'fail';
			applyError = err instanceof Error ? err.message : 'Connection test failed';
			// Reset to idle after a few seconds so user can retry
			setTimeout(() => { applyStatus = 'idle'; }, 4000);
		}
	}

	function handleKeyDown(e: KeyboardEvent) {
		if (e.key === 'Escape') isOpen = false;
		if (e.key === 'Enter' && isOpen) {
			e.preventDefault();
			applyConfig();
		}
	}

	function handleClickOutside(e: MouseEvent) {
		if (dropdownElement && !dropdownElement.contains(e.target as Node)) {
			isOpen = false;
		}
	}

	$effect(() => {
		if (isOpen) {
			document.addEventListener('click', handleClickOutside, true);
			return () => document.removeEventListener('click', handleClickOutside, true);
		}
	});
</script>

<div bind:this={dropdownElement} class="model-selector">

	<!-- Trigger chip -->
	<button
		class="model-trigger"
		class:has-model={!!activeModel}
		role="combobox"
		aria-expanded={isOpen}
		aria-controls="osa-model-panel"
		aria-label="Select model: {activeModel ?? 'none selected'}"
		aria-haspopup="dialog"
		title={activeModel ?? 'Select model'}
		onclick={(e) => { e.stopPropagation(); isOpen ? (isOpen = false) : openDropdown(); }}
		onkeydown={handleKeyDown}
	>
		<!-- CPU/chip icon to differentiate from ModeSelector dot -->
		<svg class="model-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
			<rect x="4" y="4" width="16" height="16" rx="2" />
			<rect x="9" y="9" width="6" height="6" />
			<line x1="9" y1="1" x2="9" y2="4" /><line x1="15" y1="1" x2="15" y2="4" />
			<line x1="9" y1="20" x2="9" y2="23" /><line x1="15" y1="20" x2="15" y2="23" />
			<line x1="20" y1="9" x2="23" y2="9" /><line x1="20" y1="15" x2="23" y2="15" />
			<line x1="1" y1="9" x2="4" y2="9" /><line x1="1" y1="15" x2="4" y2="15" />
		</svg>
		{#if !compact}
			<span class="model-label">{displayLabel}</span>
			<svg
				class="model-chevron"
				class:open={isOpen}
				viewBox="0 0 24 24"
				fill="none"
				stroke="currentColor"
				stroke-width="2.5"
				aria-hidden="true"
			>
				<polyline points="6 9 12 15 18 9" />
			</svg>
		{/if}
	</button>

	<!-- Dropdown — opens upward -->
	{#if isOpen}
		<div
			id="osa-model-panel"
			class="model-dropdown"
			role="dialog"
			aria-label="Model configuration"
		>
			<!-- Provider section -->
			<div class="section-label">Provider</div>
			<div class="provider-group" role="radiogroup" aria-label="Select provider">
				{#each PROVIDERS as provider}
					<label class="provider-option" class:active={stagedProvider === provider}>
						<input
							type="radio"
							name="osa-provider"
							value={provider}
							checked={stagedProvider === provider}
							class="sr-only"
							onchange={() => { stagedProvider = provider; stagedModel = ''; applyStatus = 'idle'; applyError = ''; if (provider === 'ollama') fetchLocalOllamaModels(); }}
						/>
						<span class="provider-dot" style="background-color: {provider === 'ollama' ? '#22c55e' : provider === 'ollama-cloud' ? '#06b6d4' : provider === 'anthropic' ? '#f59e0b' : '#3b82f6'}" aria-hidden="true"></span>
						<span class="provider-name">{PROVIDER_LABELS[provider]}</span>
						{#if provider === 'ollama'}
							<span class="provider-badge local">local</span>
						{:else}
							<span class="provider-badge cloud">cloud</span>
						{/if}
						{#if stagedProvider === provider}
							<svg class="provider-check" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" aria-hidden="true">
								<polyline points="20 6 9 17 4 12" />
							</svg>
						{/if}
					</label>
				{/each}
			</div>

			<!-- API URL for cloud providers -->
			{#if stagedProvider === 'ollama-cloud'}
				<div class="section-label" style="margin-top:4px">Ollama Cloud URL</div>
				<div class="model-input-wrap">
					<input
						type="text"
						class="model-input"
						placeholder="https://your-ollama-cloud.example.com"
						value={stagedUrl}
						aria-label="Ollama cloud URL"
						oninput={(e) => { stagedUrl = (e.target as HTMLInputElement).value; }}
					/>
				</div>
			{/if}

			<!-- Divider -->
			<div class="divider" role="separator"></div>

			<!-- Model name input -->
			<div class="section-label">Model name</div>
			<div class="model-input-wrap">
				<input
					type="text"
					class="model-input"
					placeholder={currentPresets[0] ?? 'e.g. gpt-4o'}
					value={stagedModel}
					aria-label="Model name"
					oninput={(e) => { stagedModel = (e.target as HTMLInputElement).value; }}
					onkeydown={handleKeyDown}
				/>
			</div>

			<!-- Quick presets / installed models -->
			{#if currentPresets.length > 0}
				<div class="presets-label">
					{#if stagedProvider === 'ollama' && localOllamaModels.length > 0}
						Installed models
					{:else if stagedProvider === 'ollama' && ollamaFetchStatus === 'loading'}
						Loading models…
					{:else}
						Quick presets
					{/if}
				</div>
				<div class="presets-list">
					{#each currentPresets as preset}
						<button
							class="preset-chip"
							class:selected={stagedModel === preset}
							type="button"
							onclick={() => selectPreset(preset)}
							aria-pressed={stagedModel === preset}
						>
							{preset}
						</button>
					{/each}
				</div>
			{:else if stagedProvider === 'ollama' && ollamaFetchStatus === 'error'}
				<div class="apply-error" style="margin-top:2px">Ollama not running — type a model name manually</div>
			{/if}

			<!-- Apply button with test status -->
			<button
				class="apply-btn"
				class:testing={applyStatus === 'testing'}
				class:pass={applyStatus === 'pass'}
				class:fail={applyStatus === 'fail'}
				type="button"
				disabled={!stagedModel.trim() || applyStatus === 'testing' || applyStatus === 'pass'}
				onclick={applyConfig}
				aria-label="Apply model configuration"
			>
				{#if applyStatus === 'testing'}
					<span class="apply-spinner" aria-hidden="true"></span>
					Testing…
				{:else if applyStatus === 'pass'}
					<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" class="apply-icon" aria-hidden="true">
						<polyline points="20 6 9 17 4 12" />
					</svg>
					Connected
				{:else if applyStatus === 'fail'}
					<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" class="apply-icon" aria-hidden="true">
						<line x1="18" y1="6" x2="6" y2="18" /><line x1="6" y1="6" x2="18" y2="18" />
					</svg>
					Failed
				{:else}
					Apply & Test
				{/if}
			</button>
			{#if applyStatus === 'fail' && applyError}
				<div class="apply-error">{applyError}</div>
			{/if}
		</div>
	{/if}
</div>

<style>
	.model-selector {
		position: relative;
		flex-shrink: 0;
	}

	/* ===== TRIGGER ===== */
	.model-trigger {
		display: flex;
		align-items: center;
		gap: 4px;
		padding: 4px 8px 4px 6px;
		background: rgba(0, 0, 0, 0.04);
		border: 1px solid rgba(0, 0, 0, 0.06);
		border-radius: 8px;
		cursor: pointer;
		transition: all 0.15s ease;
		max-width: 160px;
	}

	.model-trigger:hover {
		background: rgba(0, 0, 0, 0.07);
	}

	:global(.dark) .model-trigger {
		background: rgba(255, 255, 255, 0.06);
		border-color: rgba(255, 255, 255, 0.08);
	}

	:global(.dark) .model-trigger:hover {
		background: rgba(255, 255, 255, 0.1);
	}

	.model-icon {
		width: 14px;
		height: 14px;
		color: #8e8e93;
		flex-shrink: 0;
	}

	:global(.dark) .model-icon {
		color: #636366;
	}

	.model-label {
		font-size: 11px;
		font-weight: 500;
		font-family: ui-monospace, 'SF Mono', Menlo, monospace;
		color: #636366;
		letter-spacing: 0.01em;
		max-width: 100px;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	:global(.dark) .model-label {
		color: #98989d;
	}

	.model-trigger.has-model .model-label {
		color: #1c1c1e;
	}

	:global(.dark) .model-trigger.has-model .model-label {
		color: #f5f5f7;
	}

	.model-chevron {
		width: 10px;
		height: 10px;
		color: #aeaeb2;
		transition: transform 0.2s ease;
		flex-shrink: 0;
	}

	.model-chevron.open {
		transform: rotate(180deg);
	}

	/* ===== DROPDOWN ===== */
	.model-dropdown {
		position: absolute;
		bottom: calc(100% + 8px);
		left: 0;
		z-index: 10100;
		min-width: 256px;
		padding: 10px 8px 8px;
		background: rgba(255, 255, 255, 0.85);
		backdrop-filter: blur(32px) saturate(1.6);
		-webkit-backdrop-filter: blur(32px) saturate(1.6);
		border: 1px solid rgba(255, 255, 255, 0.7);
		border-radius: 16px;
		box-shadow:
			0 16px 48px rgba(0, 0, 0, 0.12),
			0 4px 12px rgba(0, 0, 0, 0.06),
			inset 0 1px 0 rgba(255, 255, 255, 0.8);
		animation: dropdownIn 0.15s ease-out;
		display: flex;
		flex-direction: column;
		gap: 4px;
	}

	:global(.dark) .model-dropdown {
		background: rgba(36, 36, 38, 0.9);
		border-color: rgba(255, 255, 255, 0.1);
		box-shadow:
			0 16px 48px rgba(0, 0, 0, 0.5),
			0 4px 12px rgba(0, 0, 0, 0.3),
			inset 0 1px 0 rgba(255, 255, 255, 0.06);
	}

	@keyframes dropdownIn {
		from {
			opacity: 0;
			transform: translateY(4px) scale(0.97);
		}
		to {
			opacity: 1;
			transform: translateY(0) scale(1);
		}
	}

	/* ===== SECTION LABELS ===== */
	.section-label,
	.presets-label {
		font-size: 10px;
		font-weight: 600;
		letter-spacing: 0.06em;
		text-transform: uppercase;
		color: #8e8e93;
		padding: 0 4px;
		margin-top: 2px;
	}

	:global(.dark) .section-label,
	:global(.dark) .presets-label {
		color: #636366;
	}

	/* ===== PROVIDER RADIO GROUP ===== */
	.provider-group {
		display: flex;
		flex-direction: column;
		gap: 1px;
	}

	.provider-option {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 8px 10px;
		border-radius: 10px;
		cursor: pointer;
		transition: background 0.12s ease;
		user-select: none;
	}

	.provider-option:hover {
		background: rgba(0, 0, 0, 0.05);
	}

	.provider-option.active {
		background: rgba(0, 0, 0, 0.06);
	}

	:global(.dark) .provider-option:hover {
		background: rgba(255, 255, 255, 0.07);
	}

	:global(.dark) .provider-option.active {
		background: rgba(255, 255, 255, 0.08);
	}

	.provider-dot {
		width: 6px;
		height: 6px;
		border-radius: 50%;
		flex-shrink: 0;
	}

	.provider-name {
		font-size: 13px;
		font-weight: 600;
		color: #1c1c1e;
		flex: 1;
	}

	:global(.dark) .provider-name {
		color: #f5f5f7;
	}

	.provider-badge {
		font-size: 9px;
		font-weight: 600;
		letter-spacing: 0.04em;
		padding: 2px 5px;
		border-radius: 4px;
		flex-shrink: 0;
	}

	.provider-badge.local {
		background: rgba(34, 197, 94, 0.1);
		color: #16a34a;
	}

	.provider-badge.cloud {
		background: rgba(59, 130, 246, 0.1);
		color: #2563eb;
	}

	:global(.dark) .provider-badge.local {
		background: rgba(34, 197, 94, 0.15);
		color: #4ade80;
	}

	:global(.dark) .provider-badge.cloud {
		background: rgba(59, 130, 246, 0.15);
		color: #60a5fa;
	}

	.provider-check {
		width: 14px;
		height: 14px;
		color: #007aff;
		flex-shrink: 0;
	}

	:global(.dark) .provider-check {
		color: #0a84ff;
	}

	/* ===== DIVIDER ===== */
	.divider {
		height: 1px;
		background: rgba(0, 0, 0, 0.06);
		margin: 4px 2px;
	}

	:global(.dark) .divider {
		background: rgba(255, 255, 255, 0.07);
	}

	/* ===== MODEL TEXT INPUT ===== */
	.model-input-wrap {
		padding: 0 2px;
	}

	.model-input {
		width: 100%;
		padding: 8px 10px;
		font-size: 12px;
		font-family: ui-monospace, 'SF Mono', Menlo, monospace;
		color: #1c1c1e;
		background: rgba(0, 0, 0, 0.04);
		border: 1px solid rgba(0, 0, 0, 0.08);
		border-radius: 8px;
		outline: none;
		transition: border-color 0.15s ease, box-shadow 0.15s ease;
		box-sizing: border-box;
	}

	.model-input::placeholder {
		color: #aeaeb2;
		font-style: italic;
	}

	.model-input:focus {
		border-color: rgba(0, 122, 255, 0.35);
		box-shadow: 0 0 0 3px rgba(0, 122, 255, 0.08);
	}

	:global(.dark) .model-input {
		color: #f5f5f7;
		background: rgba(255, 255, 255, 0.06);
		border-color: rgba(255, 255, 255, 0.1);
	}

	:global(.dark) .model-input::placeholder {
		color: #636366;
	}

	:global(.dark) .model-input:focus {
		border-color: rgba(10, 132, 255, 0.4);
		box-shadow: 0 0 0 3px rgba(10, 132, 255, 0.1);
	}

	/* ===== PRESET CHIPS ===== */
	.presets-list {
		display: flex;
		flex-wrap: wrap;
		gap: 4px;
		padding: 0 2px;
	}

	.preset-chip {
		padding: 4px 8px;
		font-size: 11px;
		font-family: ui-monospace, 'SF Mono', Menlo, monospace;
		color: #3a3a3c;
		background: rgba(0, 0, 0, 0.04);
		border: 1px solid rgba(0, 0, 0, 0.06);
		border-radius: 6px;
		cursor: pointer;
		transition: background 0.12s ease, border-color 0.12s ease, color 0.12s ease;
		white-space: nowrap;
	}

	.preset-chip:hover {
		background: rgba(0, 0, 0, 0.08);
	}

	.preset-chip.selected {
		background: rgba(0, 122, 255, 0.1);
		border-color: rgba(0, 122, 255, 0.25);
		color: #007aff;
	}

	:global(.dark) .preset-chip {
		color: #d1d1d6;
		background: rgba(255, 255, 255, 0.06);
		border-color: rgba(255, 255, 255, 0.08);
	}

	:global(.dark) .preset-chip:hover {
		background: rgba(255, 255, 255, 0.1);
	}

	:global(.dark) .preset-chip.selected {
		background: rgba(10, 132, 255, 0.15);
		border-color: rgba(10, 132, 255, 0.3);
		color: #0a84ff;
	}

	/* ===== APPLY BUTTON ===== */
	.apply-btn {
		margin-top: 4px;
		padding: 8px 12px;
		font-size: 12px;
		font-weight: 600;
		color: white;
		background: #1c1c1e;
		border: none;
		border-radius: 8px;
		cursor: pointer;
		transition: background 0.15s ease, opacity 0.15s ease, color 0.15s ease;
		width: 100%;
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 6px;
	}

	.apply-btn:hover:not(:disabled) {
		background: #000;
	}

	.apply-btn:disabled {
		opacity: 0.38;
		cursor: default;
	}

	.apply-btn.testing {
		background: rgba(59, 130, 246, 0.1);
		color: #3b82f6;
		border: 1px solid rgba(59, 130, 246, 0.2);
	}

	.apply-btn.pass {
		background: rgba(34, 197, 94, 0.12);
		color: #16a34a;
		border: 1px solid rgba(34, 197, 94, 0.2);
	}

	.apply-btn.fail {
		background: rgba(239, 68, 68, 0.1);
		color: #dc2626;
		border: 1px solid rgba(239, 68, 68, 0.2);
	}

	:global(.dark) .apply-btn {
		background: #f5f5f7;
		color: #1c1c1e;
	}

	:global(.dark) .apply-btn:hover:not(:disabled) {
		background: #fff;
	}

	:global(.dark) .apply-btn.testing {
		background: rgba(59, 130, 246, 0.15);
		color: #60a5fa;
		border-color: rgba(59, 130, 246, 0.25);
	}

	:global(.dark) .apply-btn.pass {
		background: rgba(34, 197, 94, 0.15);
		color: #4ade80;
		border-color: rgba(34, 197, 94, 0.25);
	}

	:global(.dark) .apply-btn.fail {
		background: rgba(239, 68, 68, 0.15);
		color: #fca5a5;
		border-color: rgba(239, 68, 68, 0.25);
	}

	.apply-icon {
		width: 14px;
		height: 14px;
		flex-shrink: 0;
	}

	.apply-spinner {
		width: 14px;
		height: 14px;
		border: 2px solid currentColor;
		border-top-color: transparent;
		border-radius: 50%;
		animation: spin 0.6s linear infinite;
	}

	@keyframes spin {
		to { transform: rotate(360deg); }
	}

	.apply-error {
		font-size: 10px;
		color: #dc2626;
		padding: 2px 4px;
		line-height: 1.3;
	}

	:global(.dark) .apply-error {
		color: #fca5a5;
	}

	/* ===== SCREEN-READER ONLY ===== */
	.sr-only {
		position: absolute;
		width: 1px;
		height: 1px;
		padding: 0;
		margin: -1px;
		overflow: hidden;
		clip: rect(0, 0, 0, 0);
		white-space: nowrap;
		border: 0;
	}
</style>
