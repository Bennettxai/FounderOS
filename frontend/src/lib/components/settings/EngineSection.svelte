<script lang="ts">
	import { onMount } from 'svelte';
	import { Cpu, CheckCircle2, XCircle, Loader2, Plug, BookOpen } from 'lucide-svelte';
	import { goto } from '$app/navigation';
	import {
		getEngineConfig,
		updateEngineConfig,
		type EngineConfig,
		type EngineTestResult
	} from '$lib/api/workspace-admin';
	import { setCachedEngineConfig } from '$lib/optimal-engine/context';
	import { testEngine, listEngineWorkspaces, detectLocalEngine, ensureEngineWorkspace, type EngineWorkspace } from '$lib/optimal-engine/connect';
	import { workspaces, createWorkspace } from '$lib/stores/workspaces';

	let { workspaceId, canManage }: { workspaceId: string; canManage: boolean } = $props();

	let loading = $state(true);
	let saving = $state(false);
	let testing = $state(false);
	let error = $state<string | null>(null);
	let saved = $state(false);
	let testResult = $state<EngineTestResult | null>(null);

	let detecting = $state(false);
	let engineWorkspaces = $state<EngineWorkspace[] | null>(null);
	let creatingSlug = $state<string | null>(null);
	let probing = $state(false);
	let detectedLocal = $state<string | null>(null);

	let form = $state<EngineConfig>({
		enabled: false,
		base_url: '',
		api_key: '',
		workspace: ''
	});

	onMount(async () => {
		await load();
		// If nothing is connected yet, look for an engine running on this machine
		// so the user can connect it in one click instead of typing a URL.
		if (!form.base_url) {
			probing = true;
			try {
				detectedLocal = await detectLocalEngine();
			} catch {
				detectedLocal = null;
			} finally {
				probing = false;
			}
		}
	});

	// One-click connect to the auto-detected local engine: prefill, verify, and
	// pull its workspaces so the user can create the ones they need.
	async function useDetected() {
		if (!detectedLocal) return;
		const current = $workspaces.find((workspace) => workspace.id === workspaceId);
		if (!current) {
			error = 'Current BusinessOS workspace is unavailable';
			return;
		}
		form.base_url = detectedLocal;
		form.enabled = true;
		form.workspace = current.slug;
		detectedLocal = null;
		try {
			await ensureEngineWorkspace(form.base_url, form.api_key, {
				slug: current.slug,
				name: current.name,
				description: current.description ?? undefined,
			});
			await test();
			await save();
			await detect();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Could not provision engine workspace';
		}
	}

	async function useEngineWorkspace(ws: EngineWorkspace) {
		form.workspace = ws.slug;
		form.enabled = true;
		await save();
	}

	async function load() {
		loading = true;
		error = null;
		try {
			const cfg = await getEngineConfig(workspaceId);
			form = {
				enabled: cfg.enabled ?? false,
				base_url: cfg.base_url ?? '',
				api_key: cfg.api_key ?? '',
				workspace: cfg.workspace ?? ''
			};
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load engine config';
		} finally {
			loading = false;
		}
	}

	async function save() {
		if (!canManage) return;
		saving = true;
		error = null;
		saved = false;
		try {
			const cfg = await updateEngineConfig(workspaceId, {
				enabled: form.enabled,
				base_url: form.base_url.trim(),
				api_key: (form.api_key ?? '').trim(),
				workspace: form.workspace.trim()
			});
			form = {
				enabled: cfg.enabled,
				base_url: cfg.base_url,
				api_key: cfg.api_key ?? '',
				workspace: cfg.workspace
			};
			// Cache + rebuild the engine client so modules use the new connection.
			setCachedEngineConfig({
				enabled: cfg.enabled,
				base_url: cfg.base_url,
				api_key: cfg.api_key ?? '',
				workspace: cfg.workspace
			});
			// Desktop: cache the full connection (incl. the key the user entered)
			// on this machine so the desktop can write to a LOCAL engine that the
			// cloud backend cannot reach. Uses form.api_key since the server never
			// returns the stored key.
			const el = (globalThis as unknown as { electron?: { engine?: { setConfig?: (id: string, c: unknown) => Promise<unknown> } } }).electron;
			if (el?.engine?.setConfig) {
				await el.engine.setConfig(workspaceId, {
					enabled: form.enabled,
					base_url: form.base_url.trim(),
					api_key: (form.api_key ?? '').trim(),
					workspace: form.workspace.trim()
				});
			}
			saved = true;
			setTimeout(() => (saved = false), 2500);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to save';
		} finally {
			saving = false;
		}
	}

	async function test() {
		testing = true;
		testResult = null;
		error = null;
		try {
			// Reach the engine from the user's machine (browser or desktop) so a
			// local engine (http://localhost:4200) works regardless of whether
			// BusinessOS runs live, as the desktop app, or in local dev.
			testResult = await testEngine(form.base_url, form.api_key);
		} catch (e) {
			testResult = {
				reachable: false,
				message: e instanceof Error ? e.message : 'Test failed'
			};
		} finally {
			testing = false;
		}
	}

	async function detect() {
		detecting = true;
		error = null;
		engineWorkspaces = null;
		try {
			engineWorkspaces = await listEngineWorkspaces(form.base_url, form.api_key);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Could not list engine workspaces';
		} finally {
			detecting = false;
		}
	}

	function existsInBusinessOS(ws: EngineWorkspace): boolean {
		const slug = ws.slug.toLowerCase();
		const name = ws.name.toLowerCase();
		return $workspaces.some(
			(w) => w.slug?.toLowerCase() === slug || w.name?.toLowerCase() === name
		);
	}

	// Engine detects workspaces; you create the ones missing from BusinessOS so
	// they land in the database and are mapped to that engine workspace.
	async function createFromEngine(ws: EngineWorkspace) {
		if (!canManage) return;
		creatingSlug = ws.slug;
		error = null;
		try {
			const created = await createWorkspace({ name: ws.name });
			await updateEngineConfig(created.id, {
				enabled: true,
				base_url: form.base_url.trim(),
				api_key: (form.api_key ?? '').trim(),
				workspace: ws.slug
			});
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to create workspace';
		} finally {
			creatingSlug = null;
		}
	}
</script>

<header class="sec-head">
	<div class="sec-icon"><Cpu size={20} strokeWidth={1.8} /></div>
	<div>
		<h2>Optimal Engine</h2>
		<p>Connect this workspace to an existing local, self-hosted, or shared Optimal Engine. This controls the engine this workspace actively uses for knowledge, search, memory, and context.</p>
	</div>
</header>

{#if loading}
	<div class="loading"><Loader2 class="spin" size={18} /> Loading connection…</div>
{:else}
	{#if error}<div class="banner banner--error">{error}</div>{/if}

	{#if probing}
		<div class="banner banner--info"><Loader2 class="spin" size={14} /> Looking for an Optimal Engine running on this machine…</div>
	{:else if detectedLocal}
		<div class="detected-banner">
			<div class="detected-text">
				<strong>Found an Optimal Engine</strong> running on this machine at <span class="mono">{detectedLocal}</span>
			</div>
			<button class="btn btn--primary btn--sm" onclick={useDetected}>Use this engine</button>
		</div>
	{/if}

	<div class="connection-state {form.enabled && form.base_url ? 'connection-state--on' : ''}">
		<span class="state-dot"></span>
		<div>
			<strong>{form.enabled && form.base_url ? 'This workspace is using an Optimal Engine' : 'No Optimal Engine is active for this workspace'}</strong>
			<span>{form.enabled && form.base_url ? `${form.base_url}${form.workspace ? ` · workspace: ${form.workspace}` : ''}` : 'Configure and save a connection below, or detect one running on this machine.'}</span>
		</div>
		{#if form.enabled && form.base_url}
			<button class="btn btn--ghost btn--sm" onclick={() => goto('/knowledge')}><BookOpen size={14} /> Open knowledge</button>
		{/if}
	</div>

	<div class="card">
		<div class="toggle-row">
			<div>
				<div class="field-label">Enable connection</div>
				<div class="field-hint">When on, modules use this engine for search, memory, and context.</div>
			</div>
			<button
				class="switch {form.enabled ? 'switch--on' : ''}"
				role="switch"
				aria-checked={form.enabled}
				aria-label="Enable Optimal Engine"
				disabled={!canManage}
				onclick={() => (form.enabled = !form.enabled)}
			>
				<span class="knob"></span>
			</button>
		</div>

		<label class="field">
			<span class="field-label">Engine URL</span>
			<input
				type="url"
				placeholder="https://engine.yourcompany.com  or  http://localhost:4200"
				bind:value={form.base_url}
				disabled={!canManage}
			/>
			<span class="field-hint">The base URL of the engine you want this workspace to use. It can be local or hosted.</span>
		</label>

		<label class="field">
			<span class="field-label">API key</span>
			<input
				type="password"
				placeholder="Paste the engine API key"
				autocomplete="off"
				bind:value={form.api_key}
				disabled={!canManage}
			/>
			<span class="field-hint">Optional for a local development engine. Required only when the engine you connect is protected.</span>
		</label>

		<label class="field">
			<span class="field-label">Engine workspace</span>
			<input
				type="text"
				placeholder="e.g. acme  (the workspace slug inside the engine)"
				bind:value={form.workspace}
				disabled={!canManage}
			/>
			<span class="field-hint">The workspace slug inside that engine. Use Detect to inspect available engine workspaces.</span>
		</label>

		{#if testResult}
			<div class="test-result {testResult.reachable ? 'test-result--ok' : 'test-result--bad'}">
				{#if testResult.reachable}
					<CheckCircle2 size={16} />
				{:else}
					<XCircle size={16} />
				{/if}
				<span>{testResult.message}{testResult.status ? ` (HTTP ${testResult.status})` : ''}</span>
			</div>
		{/if}

		<div class="actions">
			<button class="btn btn--ghost" onclick={test} disabled={testing || !form.base_url}>
				{#if testing}<Loader2 class="spin" size={15} />{:else}<Plug size={15} />{/if}
				Test connection
			</button>
			{#if canManage}
				<button class="btn btn--primary" onclick={save} disabled={saving}>
					{#if saving}<Loader2 class="spin" size={15} />{/if}
					{saved ? 'Connection saved' : 'Save connection'}
				</button>
			{/if}
		</div>
		{#if !canManage}
			<p class="readonly-note">Only owners, admins, and managers can change the engine connection.</p>
		{/if}
	</div>

	<div class="card">
		<div class="detect-head">
			<div>
				<div class="field-label">Workspaces in this engine</div>
				<div class="field-hint">Inspect the workspaces and data partitions this engine already has. Create any missing BusinessOS workspace from a detected engine workspace.</div>
			</div>
			<button class="btn btn--ghost" onclick={detect} disabled={detecting || !form.base_url}>
				{#if detecting}<Loader2 class="spin" size={15} />{:else}<Cpu size={15} />{/if}
				Detect
			</button>
		</div>

		{#if engineWorkspaces}
			{#if engineWorkspaces.length === 0}
				<p class="field-hint">No workspaces found in the engine.</p>
			{:else}
				<ul class="ws-list">
					{#each engineWorkspaces as ws (ws.id)}
						<li class="ws-row">
							<div class="ws-info">
								<span class="ws-name">{ws.name}</span>
								<span class="ws-slug">/{ws.slug}</span>
							</div>
							{#if existsInBusinessOS(ws)}
								{#if form.workspace === ws.slug}
									<span class="ws-tag ws-tag--ok"><CheckCircle2 size={13} /> Connected</span>
								{:else if canManage}
									<button class="btn btn--ghost btn--sm" onclick={() => useEngineWorkspace(ws)}>Use for this workspace</button>
								{:else}
									<span class="ws-tag ws-tag--ok"><CheckCircle2 size={13} /> In BusinessOS</span>
								{/if}
							{:else if canManage}
								<button class="btn btn--ghost btn--sm" onclick={() => createFromEngine(ws)} disabled={creatingSlug === ws.slug}>
									{#if creatingSlug === ws.slug}<Loader2 class="spin" size={13} />{/if}
									Create workspace
								</button>
							{:else}
								<span class="ws-tag">Not in BusinessOS</span>
							{/if}
						</li>
					{/each}
				</ul>
			{/if}
		{/if}
	</div>

	<div class="help-card">
		<h3>Connection model</h3>
		<ol>
			<li>Use <strong>Built-in Engine</strong> if you want a separate private brain on this device, or run/connect any existing Optimal Engine here.</li>
			<li>Use <strong>Detect</strong> to inspect the engine before choosing the workspace slug.</li>
			<li>Use <strong>Test connection</strong>, then <strong>Save connection</strong> to make this workspace actively use it.</li>
			<li>Turning this connection off stops this workspace from using that engine. It does not delete data from the engine.</li>
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
	.connection-state { display: flex; align-items: center; gap: 10px; border: 1px solid var(--dbd); border-radius: 11px; padding: 12px 14px; margin-bottom: 16px; background: color-mix(in srgb, var(--dt) 2%, transparent); }
	.connection-state--on { border-color: color-mix(in srgb, #22c55e 35%, transparent); background: color-mix(in srgb, #22c55e 7%, transparent); }
	.state-dot { width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0; background: color-mix(in srgb, var(--dt) 35%, transparent); }
	.connection-state--on .state-dot { background: #22c55e; box-shadow: 0 0 0 3px color-mix(in srgb, #22c55e 18%, transparent); }
	.connection-state div { min-width: 0; display: flex; flex: 1; flex-direction: column; gap: 2px; }
	.connection-state strong { font-size: 0.82rem; color: var(--dt); }
	.connection-state span:not(.state-dot) { font-size: 0.74rem; color: var(--dt3); overflow-wrap: anywhere; }
	@media (max-width: 560px) { .connection-state { align-items: flex-start; flex-wrap: wrap; } .connection-state .btn { width: 100%; justify-content: center; } }
	.toggle-row { display: flex; align-items: center; justify-content: space-between; gap: 16px; }
	.field { display: flex; flex-direction: column; gap: 6px; }
	.field-label { font-size: 0.82rem; font-weight: 580; color: var(--dt); }
	.field-hint { font-size: 0.74rem; color: var(--dt3); }
	.field input {
		width: 100%; padding: 9px 12px; border-radius: 9px;
		border: 1px solid var(--dbd); background: var(--dbg); color: var(--dt);
		font-size: 0.86rem; outline: none; transition: border-color 140ms ease;
	}
	.field input:focus { border-color: color-mix(in srgb, var(--dt) 40%, transparent); }
	.field input:disabled { opacity: 0.6; cursor: not-allowed; }

	.switch {
		width: 42px; height: 24px; border-radius: 999px; border: none; cursor: pointer;
		background: color-mix(in srgb, var(--dt) 20%, transparent); position: relative;
		transition: background 160ms ease; flex-shrink: 0;
	}
	.switch--on { background: #22c55e; }
	.switch:disabled { opacity: 0.5; cursor: not-allowed; }
	.knob {
		position: absolute; top: 3px; left: 3px; width: 18px; height: 18px;
		border-radius: 50%; background: #fff; transition: transform 160ms ease;
	}
	.switch--on .knob { transform: translateX(18px); }

	.actions { display: flex; gap: 10px; justify-content: flex-end; }
	.btn {
		display: inline-flex; align-items: center; gap: 7px; padding: 9px 16px;
		border-radius: 9px; font-size: 0.84rem; font-weight: 560; cursor: pointer;
		border: 1px solid transparent; transition: background 140ms ease, opacity 140ms ease;
	}
	.btn:disabled { opacity: 0.55; cursor: not-allowed; }
	.btn--primary { background: var(--dt); color: var(--dbg); }
	.btn--ghost { background: transparent; border-color: var(--dbd); color: var(--dt2); }
	.btn--ghost:hover { background: color-mix(in srgb, var(--dt) 6%, transparent); }

	@media (max-width: 768px) {
		.card { padding: 16px; }
		.field input { font-size: 1rem; /* prevent iOS zoom */ min-height: 40px; }
		/* Toggle row: keep side-by-side but allow text to wrap */
		.toggle-row { align-items: flex-start; }
		/* Actions: stack buttons full-width */
		.actions { flex-direction: column; }
		.actions .btn { width: 100%; justify-content: center; min-height: 40px; }
	}

	.test-result {
		display: flex; align-items: center; gap: 8px; padding: 10px 12px;
		border-radius: 9px; font-size: 0.82rem;
	}
	.test-result--ok { background: color-mix(in srgb, #22c55e 12%, transparent); color: #16a34a; }
	.test-result--bad { background: color-mix(in srgb, #ef4444 12%, transparent); color: #ef4444; }
	.readonly-note { font-size: 0.74rem; color: var(--dt3); margin: -6px 0 0; text-align: right; }

	.help-card { margin-top: 18px; padding: 18px 20px; border-radius: 14px; border: 1px dashed var(--dbd); }
	.help-card h3 { margin: 0 0 10px; font-size: 0.86rem; font-weight: 620; }
	.help-card ol { margin: 0; padding-left: 18px; display: flex; flex-direction: column; gap: 6px; }
	.help-card li { font-size: 0.82rem; color: var(--dt2); line-height: 1.5; }
	.help-card code { background: color-mix(in srgb, var(--dt) 8%, transparent); padding: 1px 5px; border-radius: 4px; font-size: 0.78rem; }

	.banner { padding: 11px 14px; border-radius: 10px; font-size: 0.83rem; margin-bottom: 16px; }
	.banner--error { background: color-mix(in srgb, #ef4444 12%, transparent); color: #ef4444; }
	.loading { display: flex; align-items: center; gap: 8px; color: var(--dt3); font-size: 0.85rem; padding: 20px 0; }
	:global(.spin) { animation: spin 0.9s linear infinite; }
	@keyframes spin { to { transform: rotate(360deg); } }

	.banner--info { display: flex; align-items: center; gap: 8px; background: color-mix(in srgb, var(--dt) 5%, transparent); color: var(--dt2); padding: 11px 14px; border-radius: 10px; font-size: 0.85rem; }
	.detected-banner { display: flex; align-items: center; justify-content: space-between; gap: 14px; padding: 13px 15px; border: 1px solid color-mix(in srgb, #3ad389 35%, transparent); background: color-mix(in srgb, #3ad389 8%, transparent); border-radius: 11px; margin-bottom: 4px; }
	.detected-text { font-size: 0.86rem; color: var(--dt2); }
	.detected-text strong { color: var(--dt); }
	.mono { font-family: ui-monospace, monospace; font-size: 0.82rem; color: var(--dt); }
	@media (max-width: 480px) { .detected-banner { flex-direction: column; align-items: stretch; } .detected-banner .btn { width: 100%; justify-content: center; min-height: 40px; } }
	.detect-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 14px; }
	.btn--sm { padding: 5px 10px; font-size: 0.78rem; min-height: 34px; }
	.ws-list { list-style: none; margin: 14px 0 0; padding: 0; display: flex; flex-direction: column; gap: 6px; }
	.ws-row { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 9px 11px; border: 1px solid var(--dbd); border-radius: 9px; }
	.ws-info { display: flex; align-items: baseline; gap: 7px; min-width: 0; }
	.ws-name { font-size: 0.86rem; font-weight: 560; color: var(--dt); }
	.ws-slug { font-size: 0.76rem; color: var(--dt3); }
	.ws-tag { display: inline-flex; align-items: center; gap: 5px; font-size: 0.74rem; color: var(--dt3); border: 1px solid var(--dbd); padding: 3px 9px; border-radius: 999px; }
	.ws-tag--ok { color: #3ad389; border-color: color-mix(in srgb, #3ad389 35%, transparent); }
	@media (max-width: 480px) {
		.detect-head { flex-direction: column; }
		.ws-row { flex-direction: column; align-items: stretch; gap: 8px; }
		.ws-row .btn { width: 100%; justify-content: center; min-height: 40px; }
	}
</style>
