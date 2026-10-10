<script lang="ts">
	import { onMount } from 'svelte';
	import { Sparkles, CheckCircle2, Loader2, KeyRound, Server } from 'lucide-svelte';

	// The bundled Optimal Engine needs a model for its AI features. The user
	// brings their own: a cloud API key (Anthropic / OpenAI) or a local Ollama
	// server. This panel captures that choice and stores it on this machine via
	// the desktop main process (engine:setModelConfig / engine:getModelConfig).
	type Provider = 'anthropic' | 'openai' | 'ollama';
	interface ModelConfig {
		provider: Provider | '';
		apiKey?: string;
		ollamaUrl?: string;
	}

	interface ModelBridge {
		setModelConfig?: (cfg: ModelConfig) => Promise<{ ok: boolean; error?: string }>;
		getModelConfig?: () => Promise<ModelConfig>;
	}
	function bridge(): ModelBridge | undefined {
		return (globalThis as unknown as { electron?: { engine?: ModelBridge } }).electron?.engine;
	}

	const providers: { id: Provider; label: string; hint: string }[] = [
		{ id: 'anthropic', label: 'Anthropic API key', hint: 'Claude models via your Anthropic key.' },
		{ id: 'openai', label: 'OpenAI API key', hint: 'GPT models via your OpenAI key.' },
		{ id: 'ollama', label: 'Local Ollama', hint: 'A model running on this machine via Ollama.' }
	];

	let loading = $state(true);
	let saving = $state(false);
	let saved = $state(false);
	let error = $state<string | null>(null);
	let unavailable = $state(false);

	let provider = $state<Provider>('anthropic');
	let apiKey = $state('');
	let ollamaUrl = $state('');

	// Snapshot of what is currently stored, for the status line.
	let current = $state<ModelConfig | null>(null);

	onMount(async () => {
		const b = bridge();
		if (!b?.getModelConfig || !b?.setModelConfig) {
			unavailable = true;
			loading = false;
			return;
		}
		try {
			const cfg = await b.getModelConfig();
			current = cfg;
			if (cfg.provider) provider = cfg.provider;
			ollamaUrl = cfg.ollamaUrl ?? '';
			// The stored key is never echoed back into the input for safety.
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load model connection';
		} finally {
			loading = false;
		}
	});

	const needsKey = $derived(provider === 'anthropic' || provider === 'openai');
	const providerLabel = (p: Provider | '') => providers.find((x) => x.id === p)?.label ?? 'None';

	const canSave = $derived(
		provider === 'ollama' ? ollamaUrl.trim().length > 0 : apiKey.trim().length > 0
	);

	async function save() {
		const b = bridge();
		if (!b?.setModelConfig) {
			unavailable = true;
			return;
		}
		saving = true;
		saved = false;
		error = null;
		try {
			const cfg: ModelConfig = {
				provider,
				apiKey: needsKey ? apiKey.trim() : undefined,
				ollamaUrl: provider === 'ollama' ? ollamaUrl.trim() : undefined
			};
			const res = await b.setModelConfig(cfg);
			if (!res.ok) throw new Error(res.error ?? 'Save failed');
			// Reflect the new connection without keeping the key in memory.
			current = { provider, ollamaUrl: cfg.ollamaUrl };
			apiKey = '';
			saved = true;
			setTimeout(() => (saved = false), 2500);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to save';
		} finally {
			saving = false;
		}
	}
</script>

<header class="sec-head">
	<div class="sec-icon"><Sparkles size={20} strokeWidth={1.8} /></div>
	<div>
		<h2>Model connection</h2>
		<p>The bundled Optimal Engine uses a model for its AI features. Bring your own: a cloud API key, or a model running locally with Ollama.</p>
	</div>
</header>

{#if loading}
	<div class="loading"><Loader2 class="spin" size={18} /> Loading connection…</div>
{:else if unavailable}
	<div class="banner banner--info">Model connection is managed by the desktop app. Open BusinessOS on the desktop to set it up.</div>
{:else}
	{#if error}<div class="banner banner--error">{error}</div>{/if}

	{#if current?.provider}
		<div class="status-banner">
			<CheckCircle2 size={16} />
			<span>
				Connected via <strong>{providerLabel(current.provider)}</strong>{#if current.provider === 'ollama' && current.ollamaUrl}<span class="mono"> · {current.ollamaUrl}</span>{/if}
			</span>
		</div>
	{:else}
		<div class="banner banner--info">No model connected yet. Choose a provider below.</div>
	{/if}

	<div class="card">
		<div class="field">
			<span class="field-label">Provider</span>
			<div class="provider-grid">
				{#each providers as p (p.id)}
					<button
						type="button"
						class="provider {provider === p.id ? 'provider--on' : ''}"
						aria-pressed={provider === p.id}
						onclick={() => (provider = p.id)}
					>
						<span class="provider-icon">
							{#if p.id === 'ollama'}<Server size={16} />{:else}<KeyRound size={16} />{/if}
						</span>
						<span class="provider-text">
							<span class="provider-label">{p.label}</span>
							<span class="provider-hint">{p.hint}</span>
						</span>
					</button>
				{/each}
			</div>
		</div>

		{#if needsKey}
			<label class="field">
				<span class="field-label">{providerLabel(provider)}</span>
				<input
					type="password"
					placeholder="Paste your API key"
					autocomplete="off"
					bind:value={apiKey}
				/>
				<span class="field-hint">Stored on this machine only. It never leaves your device except to call the model provider.</span>
			</label>
		{:else}
			<label class="field">
				<span class="field-label">Ollama URL</span>
				<input
					type="url"
					placeholder="http://localhost:11434"
					bind:value={ollamaUrl}
				/>
				<span class="field-hint">The base URL of your local Ollama server.</span>
			</label>
		{/if}

		<div class="actions">
			<button class="btn btn--primary" onclick={save} disabled={saving || !canSave}>
				{#if saving}<Loader2 class="spin" size={15} />{/if}
				{saved ? 'Saved' : 'Save connection'}
			</button>
		</div>
	</div>

	<div class="help-card">
		<h3>How the model connection works</h3>
		<ol>
			<li>Pick where your model comes from: <strong>Anthropic</strong>, <strong>OpenAI</strong>, or a local <strong>Ollama</strong> server.</li>
			<li>Paste your API key, or enter your Ollama URL (usually <code>http://localhost:11434</code>).</li>
			<li>Hit <strong>Save connection</strong>. The bundled engine uses it for its AI features.</li>
		</ol>
	</div>
{/if}

<style>
	.sec-head { display: flex; gap: 14px; align-items: flex-start; margin-bottom: 22px; }
	.sec-icon {
		width: 40px; height: 40px; border-radius: 11px; flex-shrink: 0;
		display: flex; align-items: center; justify-content: center;
		background: color-mix(in srgb, var(--dt) 8%, transparent); color: var(--dt);
	}
	.sec-head h2 { margin: 0; font-size: 1.15rem; font-weight: 660; letter-spacing: -0.02em; }
	.sec-head p { margin: 4px 0 0; font-size: 0.84rem; color: var(--dt3); line-height: 1.5; max-width: 56ch; }

	.card {
		border: 1px solid var(--dbd); border-radius: 14px; padding: 20px;
		display: flex; flex-direction: column; gap: 18px;
		background: color-mix(in srgb, var(--dt) 2%, transparent);
	}
	.field { display: flex; flex-direction: column; gap: 6px; }
	.field-label { font-size: 0.82rem; font-weight: 580; color: var(--dt); }
	.field-hint { font-size: 0.74rem; color: var(--dt3); }
	.field input {
		width: 100%; padding: 9px 12px; border-radius: 9px;
		border: 1px solid var(--dbd); background: var(--dbg); color: var(--dt);
		font-size: 0.86rem; outline: none; transition: border-color 140ms ease;
	}
	.field input:focus { border-color: color-mix(in srgb, var(--dt) 40%, transparent); }

	.provider-grid { display: flex; flex-direction: column; gap: 8px; }
	.provider {
		display: flex; align-items: center; gap: 11px; text-align: left; width: 100%;
		padding: 11px 13px; border-radius: 10px; cursor: pointer;
		border: 1px solid var(--dbd); background: var(--dbg); color: var(--dt2);
		transition: border-color 140ms ease, background 140ms ease;
	}
	.provider:hover { background: color-mix(in srgb, var(--dt) 5%, transparent); }
	.provider--on {
		border-color: color-mix(in srgb, var(--dt) 45%, transparent);
		background: color-mix(in srgb, var(--dt) 7%, transparent); color: var(--dt);
	}
	.provider-icon {
		width: 30px; height: 30px; border-radius: 8px; flex-shrink: 0;
		display: flex; align-items: center; justify-content: center;
		background: color-mix(in srgb, var(--dt) 8%, transparent); color: var(--dt);
	}
	.provider-text { display: flex; flex-direction: column; line-height: 1.25; min-width: 0; }
	.provider-label { font-size: 0.86rem; font-weight: 560; color: var(--dt); }
	.provider-hint { font-size: 0.74rem; color: var(--dt3); }

	.actions { display: flex; gap: 10px; justify-content: flex-end; }
	.btn {
		display: inline-flex; align-items: center; gap: 7px; padding: 9px 16px;
		border-radius: 9px; font-size: 0.84rem; font-weight: 560; cursor: pointer;
		border: 1px solid transparent; transition: background 140ms ease, opacity 140ms ease;
	}
	.btn:disabled { opacity: 0.55; cursor: not-allowed; }
	.btn--primary { background: var(--dt); color: var(--dbg); }

	.status-banner {
		display: flex; align-items: center; gap: 8px; margin-bottom: 16px;
		padding: 11px 14px; border-radius: 10px; font-size: 0.83rem;
		background: color-mix(in srgb, #22c55e 12%, transparent); color: #16a34a;
	}
	.status-banner strong { color: #16a34a; }
	.mono { font-family: ui-monospace, monospace; font-size: 0.8rem; }

	.help-card { margin-top: 18px; padding: 18px 20px; border-radius: 14px; border: 1px dashed var(--dbd); }
	.help-card h3 { margin: 0 0 10px; font-size: 0.86rem; font-weight: 620; }
	.help-card ol { margin: 0; padding-left: 18px; display: flex; flex-direction: column; gap: 6px; }
	.help-card li { font-size: 0.82rem; color: var(--dt2); line-height: 1.5; }
	.help-card code { background: color-mix(in srgb, var(--dt) 8%, transparent); padding: 1px 5px; border-radius: 4px; font-size: 0.78rem; }

	.banner { padding: 11px 14px; border-radius: 10px; font-size: 0.83rem; margin-bottom: 16px; }
	.banner--error { background: color-mix(in srgb, #ef4444 12%, transparent); color: #ef4444; }
	.banner--info { display: flex; align-items: center; gap: 8px; background: color-mix(in srgb, var(--dt) 5%, transparent); color: var(--dt2); }
	.loading { display: flex; align-items: center; gap: 8px; color: var(--dt3); font-size: 0.85rem; padding: 20px 0; }
	:global(.spin) { animation: spin 0.9s linear infinite; }
	@keyframes spin { to { transform: rotate(360deg); } }

	@media (max-width: 768px) {
		.card { padding: 16px; }
		.field input { font-size: 1rem; min-height: 40px; }
		.actions { flex-direction: column; }
		.actions .btn { width: 100%; justify-content: center; min-height: 40px; }
	}
</style>
