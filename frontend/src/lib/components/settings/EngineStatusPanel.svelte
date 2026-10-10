<script lang="ts">
	import { onMount } from 'svelte';
	import { HardDrive, FolderOpen, Loader2, Play, RefreshCw, Server, Square } from 'lucide-svelte';
	import { isElectron } from '$lib/utils/platform';

	// Shape of the electron.engine bridge we use here. Kept narrow and optional so
	// the component compiles and runs on the web, where window.electron is absent.
	interface EngineStatus {
		running: boolean;
		available: boolean;
		url: string;
		port: number;
		dataDir: string;
	}
	interface EngineBridge {
		status?: () => Promise<EngineStatus>;
		start?: () => Promise<{ ok: boolean; message: string }>;
		stop?: () => Promise<{ ok: boolean; message: string }>;
		revealData?: () => Promise<{ ok: boolean; error?: string; dataDir?: string }>;
	}
	function engineBridge(): EngineBridge | null {
		const el = (globalThis as unknown as { electron?: { engine?: EngineBridge } }).electron;
		return el?.engine ?? null;
	}

	const desktop = isElectron();

	let loading = $state(true);
	let refreshing = $state(false);
	let revealing = $state(false);
	let changingState = $state(false);
	let error = $state<string | null>(null);
	let status = $state<EngineStatus | null>(null);

	async function load() {
		const bridge = engineBridge();
		if (!bridge?.status) {
			loading = false;
			return;
		}
		error = null;
		try {
			status = await bridge.status();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Could not read engine status';
		} finally {
			loading = false;
			refreshing = false;
		}
	}

	async function refresh() {
		refreshing = true;
		await load();
	}

	async function reveal() {
		const bridge = engineBridge();
		if (!bridge?.revealData) return;
		revealing = true;
		error = null;
		try {
			const res = await bridge.revealData();
			if (!res.ok && res.error) error = res.error;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Could not open the data folder';
		} finally {
			revealing = false;
		}
	}

	async function start() {
		const bridge = engineBridge();
		if (!bridge?.start) return;
		changingState = true;
		error = null;
		try {
			const result = await bridge.start();
			if (!result.ok) error = result.message;
			await load();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Could not start the built-in engine';
		} finally {
			changingState = false;
		}
	}

	async function stop() {
		const bridge = engineBridge();
		if (!bridge?.stop) return;
		changingState = true;
		error = null;
		try {
			const result = await bridge.stop();
			if (!result.ok) error = result.message;
			await load();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Could not stop the built-in engine';
		} finally {
			changingState = false;
		}
	}

	onMount(() => {
		if (desktop) load();
		else loading = false;
	});
</script>

<header class="sec-head">
	<div class="sec-icon"><Server size={20} strokeWidth={1.8} /></div>
	<div>
		<h2>Built-in engine</h2>
		<p>Use the private engine that ships with BusinessOS when you want a separate on-device brain. It is independent from any Optimal Engine connected in the other settings section.</p>
	</div>
</header>

{#if !desktop}
	<div class="card">
		<div class="banner banner--info">
			<Server size={15} />
			<span>The built-in engine runs inside the BusinessOS desktop app. Open the desktop app to see its status and reach your data files.</span>
		</div>
	</div>
{:else if loading}
	<div class="loading"><Loader2 class="spin" size={18} /> Reading engine status…</div>
{:else}
	{#if error}<div class="banner banner--error">{error}</div>{/if}

	<div class="card">
		<div class="status-row">
			<div class="status-left">
				<span class="dot {status?.running ? 'dot--on' : 'dot--off'}"></span>
				<div>
					<div class="status-title">
						Built-in engine: {status?.running ? 'running' : 'stopped'}
					</div>
					<div class="status-sub">
						{status?.running
							? 'Your private on-device engine is up and available to connect.'
							: status?.available
								? 'This private engine is stopped. Start it to expose its local URL, then connect a workspace to it from Optimal Engine.'
								: 'This source development build does not include the packaged engine runtime. Connect a development engine, such as http://localhost:4200, from Optimal Engine instead.'}
					</div>
				</div>
			</div>
			<div class="status-actions">
				<button class="btn btn--ghost btn--sm" onclick={refresh} disabled={refreshing || changingState}>
					{#if refreshing}<Loader2 class="spin" size={14} />{:else}<RefreshCw size={14} />{/if}
					Refresh
				</button>
				{#if status?.running}
					<button class="btn btn--ghost btn--sm" onclick={stop} disabled={changingState}>
						{#if changingState}<Loader2 class="spin" size={14} />{:else}<Square size={13} />{/if}
						Stop engine
					</button>
				{:else if status?.available}
					<button class="btn btn--primary btn--sm" onclick={start} disabled={changingState}>
						{#if changingState}<Loader2 class="spin" size={14} />{:else}<Play size={13} />{/if}
						Start engine
					</button>
				{/if}
			</div>
		</div>

		{#if status?.url}
			<div class="kv">
				<span class="kv-label">Local URL</span>
				<span class="mono">{status.url}</span>
			</div>
		{/if}

		<div class="kv kv--stack">
			<span class="kv-label"><HardDrive size={13} /> Data directory</span>
			<span class="mono mono--path">{status?.dataDir ?? ''}</span>
		</div>

		<div class="actions">
			<button class="btn btn--primary" onclick={reveal} disabled={revealing || !status?.dataDir}>
				{#if revealing}<Loader2 class="spin" size={15} />{:else}<FolderOpen size={15} />{/if}
				Reveal in Finder
			</button>
		</div>
	</div>

	<div class="help-card">
		<h3>How this differs from Optimal Engine</h3>
		<ol>
			<li>This engine is private to this device and stores its own independent data folder.</li>
			<li>Start it here, then use its local URL in Optimal Engine if you want a BusinessOS workspace to use it.</li>
			<li>An external or self-hosted Optimal Engine can be connected separately for a shared or existing brain.</li>
			<li>Everything this built-in engine stores lives in the data directory above. Back it up like any folder.</li>
		</ol>
	</div>
{/if}

<style>
	.sec-head {
		display: flex;
		gap: 14px;
		align-items: flex-start;
		margin-bottom: 22px;
	}
	.sec-icon {
		width: 40px;
		height: 40px;
		border-radius: 11px;
		flex-shrink: 0;
		display: flex;
		align-items: center;
		justify-content: center;
		background: color-mix(in srgb, var(--dt) 8%, transparent);
		color: var(--dt);
	}
	.sec-head h2 {
		margin: 0;
		font-size: 1.15rem;
		font-weight: 660;
		letter-spacing: -0.02em;
	}
	.sec-head p {
		margin: 4px 0 0;
		font-size: 0.84rem;
		color: var(--dt3);
		line-height: 1.5;
		max-width: 56ch;
	}

	.card {
		border: 1px solid var(--dbd);
		border-radius: 14px;
		padding: 20px;
		display: flex;
		flex-direction: column;
		gap: 18px;
		background: color-mix(in srgb, var(--dt) 2%, transparent);
	}

	.status-row {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 16px;
	}
	.status-left {
		display: flex;
		align-items: flex-start;
		gap: 12px;
		min-width: 0;
	}
	.status-actions { display: flex; align-items: center; gap: 8px; flex-shrink: 0; }
	.dot {
		width: 10px;
		height: 10px;
		border-radius: 50%;
		flex-shrink: 0;
		margin-top: 5px;
	}
	.dot--on {
		background: #22c55e;
		box-shadow: 0 0 0 3px color-mix(in srgb, #22c55e 22%, transparent);
	}
	.dot--off {
		background: color-mix(in srgb, var(--dt) 35%, transparent);
	}
	.status-title {
		font-size: 0.9rem;
		font-weight: 600;
		color: var(--dt);
	}
	.status-sub {
		font-size: 0.76rem;
		color: var(--dt3);
		margin-top: 2px;
		line-height: 1.45;
	}

	.kv {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 12px;
	}
	.kv--stack {
		flex-direction: column;
		align-items: flex-start;
		gap: 6px;
	}
	.kv-label {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		font-size: 0.78rem;
		font-weight: 560;
		color: var(--dt2);
	}
	.mono {
		font-family: ui-monospace, monospace;
		font-size: 0.8rem;
		color: var(--dt);
	}
	.mono--path {
		word-break: break-all;
		background: color-mix(in srgb, var(--dt) 5%, transparent);
		border: 1px solid var(--dbd);
		border-radius: 8px;
		padding: 8px 10px;
		width: 100%;
		box-sizing: border-box;
	}

	.actions {
		display: flex;
		gap: 10px;
		justify-content: flex-end;
	}
	.btn {
		display: inline-flex;
		align-items: center;
		gap: 7px;
		padding: 9px 16px;
		border-radius: 9px;
		font-size: 0.84rem;
		font-weight: 560;
		cursor: pointer;
		border: 1px solid transparent;
		transition:
			background 140ms ease,
			opacity 140ms ease;
	}
	.btn:disabled {
		opacity: 0.55;
		cursor: not-allowed;
	}
	.btn--primary {
		background: var(--dt);
		color: var(--dbg);
	}
	.btn--ghost {
		background: transparent;
		border-color: var(--dbd);
		color: var(--dt2);
	}
	.btn--ghost:hover {
		background: color-mix(in srgb, var(--dt) 6%, transparent);
	}
	.btn--sm {
		padding: 6px 11px;
		font-size: 0.78rem;
	}

	.help-card {
		margin-top: 18px;
		padding: 18px 20px;
		border-radius: 14px;
		border: 1px dashed var(--dbd);
	}
	.help-card h3 {
		margin: 0 0 10px;
		font-size: 0.86rem;
		font-weight: 620;
	}
	.help-card ol {
		margin: 0;
		padding-left: 18px;
		display: flex;
		flex-direction: column;
		gap: 6px;
	}
	.help-card li {
		font-size: 0.82rem;
		color: var(--dt2);
		line-height: 1.5;
	}

	.banner {
		padding: 11px 14px;
		border-radius: 10px;
		font-size: 0.83rem;
		margin-bottom: 16px;
	}
	.banner--error {
		background: color-mix(in srgb, #ef4444 12%, transparent);
		color: #ef4444;
	}
	.banner--info {
		display: flex;
		align-items: center;
		gap: 8px;
		background: color-mix(in srgb, var(--dt) 5%, transparent);
		color: var(--dt2);
		margin-bottom: 0;
	}
	.loading {
		display: flex;
		align-items: center;
		gap: 8px;
		color: var(--dt3);
		font-size: 0.85rem;
		padding: 20px 0;
	}
	:global(.spin) {
		animation: spin 0.9s linear infinite;
	}
	@keyframes spin {
		to {
			transform: rotate(360deg);
		}
	}

	@media (max-width: 768px) {
		.card {
			padding: 16px;
		}
		.status-row {
			flex-direction: column;
			align-items: stretch;
			gap: 12px;
		}
		.status-row .btn {
			justify-content: center;
			min-height: 40px;
		}
		.actions {
			flex-direction: column;
		}
		.actions .btn {
			width: 100%;
			justify-content: center;
			min-height: 40px;
		}
	}
</style>
