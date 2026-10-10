<script lang="ts">
	import { onMount } from 'svelte';
	import { RefreshCw, HardDrive, Cloud, Loader2, CalendarDays } from 'lucide-svelte';
	import {
		getSyncPolicies,
		setSyncPolicy,
		setAllSync,
		type SyncPolicy
	} from '$lib/api/workspace-admin';
	import { getCalendarConnectionStatus } from '$lib/api/calendar';
	import { getStorage, formatBytes } from '$lib/kb/client';
	import { currentWorkspace } from '$lib/stores/workspaces';

	let { workspaceId, canManage }: { workspaceId: string; canManage: boolean } = $props();

	// Cloud storage usage meter. Workspace-scoped by slug (same source the rest
	// of the app uses for knowledge calls). Defaults mirror the backend: 0 used,
	// 1 GB limit when nothing has synced yet.
	let storageLoaded = $state(false);
	let bytesUsed = $state(0);
	let bytesLimit = $state(1073741824);
	const storagePct = $derived(bytesLimit > 0 ? (bytesUsed / bytesLimit) * 100 : 0);

	let policies = $state<SyncPolicy[]>([]);
	let autoSyncAll = $state(false);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let busy = $state<string | null>(null); // module key (or '__all__') currently saving

	// Google Calendar connection indicator. Read-only status here; the
	// connect/disconnect flow lives in the Calendar module's settings.
	// The status endpoint returns `account` (the Google email); the older
	// frontend type only declares `email`, so read both.
	let gcalLoaded = $state(false);
	let gcalConnected = $state(false);
	let gcalEmail = $state<string | null>(null);

	onMount(() => {
		load();
		loadGcal();
		loadStorage();
	});

	async function loadStorage() {
		const slug = $currentWorkspace?.slug;
		if (!slug) {
			storageLoaded = true;
			return;
		}
		try {
			const s = await getStorage(slug);
			bytesUsed = s.bytes_used;
			bytesLimit = s.bytes_limit;
		} catch {
			// Non-fatal: leave defaults, just don't show the meter as loaded data.
		} finally {
			storageLoaded = true;
		}
	}

	async function loadGcal() {
		try {
			const s: { connected: boolean; email?: string; account?: string } =
				await getCalendarConnectionStatus();
			gcalConnected = !!s.connected;
			gcalEmail = s.email ?? s.account ?? null;
		} catch {
			gcalConnected = false;
		} finally {
			gcalLoaded = true;
		}
	}

	async function load() {
		loading = true;
		error = null;
		try {
			const res = await getSyncPolicies(workspaceId);
			policies = res.policies;
			autoSyncAll = res.auto_sync_all;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load sync settings';
		} finally {
			loading = false;
		}
	}

	async function toggleModule(p: SyncPolicy) {
		if (!canManage || busy) return;
		const next = p.sync_mode === 'workspace' ? 'local' : 'workspace';
		busy = p.module;
		try {
			await setSyncPolicy(workspaceId, p.module, next);
			policies = policies.map((x) => (x.module === p.module ? { ...x, sync_mode: next } : x));
			autoSyncAll = policies.every((x) => x.sync_mode === 'workspace');
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to update';
		} finally {
			busy = null;
		}
	}

	async function toggleAll() {
		if (!canManage || busy) return;
		const next = autoSyncAll ? 'local' : 'workspace';
		busy = '__all__';
		try {
			await setAllSync(workspaceId, next);
			autoSyncAll = next === 'workspace';
			policies = policies.map((x) => ({ ...x, sync_mode: next }));
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to update';
		} finally {
			busy = null;
		}
	}
</script>

<div class="sync-section">
	<div class="head">
		<div class="icon"><RefreshCw size={20} /></div>
		<div>
			<h2>Sync</h2>
			<p class="sub">
				Your data is <strong>local-first</strong> - it lives on this machine. Turn a module on
				to sync its whole dataset to the cloud so the web app and your teammates can see it.
			</p>
		</div>
	</div>

	{#if error}
		<div class="banner">{error}</div>
	{/if}

	{#if loading}
		<div class="loading"><Loader2 class="spin" size={18} /> Loading sync settings…</div>
	{:else}
		<!-- Master auto-sync -->
		<button
			class="row row--master"
			class:on={autoSyncAll}
			onclick={toggleAll}
			disabled={!canManage || busy === '__all__'}
		>
			<div class="row-left">
				<Cloud size={18} />
				<div class="row-text">
					<span class="row-title">Auto-sync everything</span>
					<span class="row-desc">Sync all modules to the cloud automatically.</span>
				</div>
			</div>
			<span class="switch" class:switch--on={autoSyncAll}>
				{#if busy === '__all__'}<Loader2 class="spin" size={13} />{/if}
			</span>
		</button>

		<div class="list">
			{#each policies as p (p.module)}
				{@const synced = p.sync_mode === 'workspace'}
				<button
					class="row"
					onclick={() => toggleModule(p)}
					disabled={!canManage || busy === p.module}
				>
					<div class="row-left">
						{#if synced}<Cloud size={17} class="ic-synced" />{:else}<HardDrive size={17} class="ic-local" />{/if}
						<div class="row-text">
							<span class="row-title">{p.label}</span>
							<span class="row-desc">{synced ? 'Synced to cloud' : 'Local only'}</span>
						</div>
					</div>
					<span class="switch" class:switch--on={synced}>
						{#if busy === p.module}<Loader2 class="spin" size={13} />{/if}
					</span>
				</button>
			{/each}
		</div>

		{#if !canManage}
			<p class="note">Only owners, admins, and managers can change sync settings.</p>
		{/if}
	{/if}

	<!-- Cloud storage usage: how much of this workspace's cloud quota the synced
	     copy is using. Byte counts come from GET /api/knowledge/storage. -->
	<div class="storage">
		<div class="storage-head">
			<span class="storage-title">Cloud storage</span>
			{#if storageLoaded}
				<span class="storage-usage" class:storage-usage--warn={storagePct >= 90}>
					{formatBytes(bytesUsed)} of {formatBytes(bytesLimit)} used
				</span>
			{/if}
		</div>
		<div class="meter">
			<div
				class="meter-fill"
				class:meter-fill--warn={storagePct >= 90 && storagePct < 100}
				class:meter-fill--full={storagePct >= 100}
				style="width: {Math.min(100, Math.max(storageLoaded ? 2 : 0, storagePct))}%"
			></div>
		</div>
	</div>

	<!-- Connected accounts: read-only status. Connecting happens in the
	     Calendar module's settings; events auto-sync in the background. -->
	<div class="accounts">
		<p class="accounts-title">Connected accounts</p>
		<div class="row row--static">
			<div class="row-left">
				<CalendarDays size={17} class={gcalConnected ? 'ic-synced' : 'ic-local'} />
				<div class="row-text">
					<span class="row-title">Google Calendar</span>
					<span class="row-desc">
						{#if !gcalLoaded}
							Checking connection…
						{:else if gcalConnected}
							{gcalEmail ?? 'Google account connected'} · syncs automatically
						{:else}
							Connect from the Calendar module's settings
						{/if}
					</span>
				</div>
			</div>
			{#if gcalLoaded}
				<span class="status-pill" class:status-pill--on={gcalConnected}>
					{gcalConnected ? 'Connected' : 'Not connected'}
				</span>
			{/if}
		</div>
	</div>
</div>

<style>
	.sync-section { max-width: 640px; }
	.head { display: flex; gap: 14px; align-items: flex-start; margin-bottom: 20px; }
	.icon { width: 40px; height: 40px; border-radius: 10px; background: color-mix(in srgb, var(--dt) 8%, transparent); display: flex; align-items: center; justify-content: center; color: var(--dt); flex-shrink: 0; }
	h2 { font-size: 1.15rem; font-weight: 680; margin: 0 0 4px; color: var(--dt); }
	.sub { font-size: 0.84rem; color: var(--dt3); margin: 0; line-height: 1.5; }
	.sub strong { color: var(--dt2); }
	.banner { margin-bottom: 14px; padding: 10px 13px; border-radius: 9px; font-size: 0.82rem; background: color-mix(in srgb, #ef4444 12%, transparent); color: #ef4444; }
	.loading { display: flex; align-items: center; gap: 8px; color: var(--dt3); padding: 20px 0; font-size: 0.85rem; }
	.row { display: flex; align-items: center; justify-content: space-between; width: 100%; gap: 14px; padding: 13px 15px; border: 1px solid var(--dbd); border-radius: 11px; background: var(--dbg); color: var(--dt); cursor: pointer; text-align: left; transition: border-color 0.15s ease, background 0.15s ease; }
	.row:hover:not(:disabled) { border-color: color-mix(in srgb, var(--dt) 22%, transparent); }
	.row:disabled { cursor: default; opacity: 0.7; }
	.row--master { margin-bottom: 16px; background: color-mix(in srgb, var(--dt) 3%, transparent); }
	.list { display: flex; flex-direction: column; gap: 8px; }
	.row-left { display: flex; align-items: center; gap: 12px; min-width: 0; }
	.row-text { display: flex; flex-direction: column; gap: 2px; min-width: 0; }
	.row-title { font-size: 0.88rem; font-weight: 580; }
	.row-desc { font-size: 0.74rem; color: var(--dt3); }
	:global(.ic-synced) { color: #34d399; }
	:global(.ic-local) { color: var(--dt3); }
	.switch { position: relative; width: 38px; height: 22px; border-radius: 999px; background: color-mix(in srgb, var(--dt) 16%, transparent); flex-shrink: 0; display: flex; align-items: center; justify-content: center; color: #fff; transition: background 0.18s ease; }
	.switch::after { content: ''; position: absolute; left: 3px; top: 3px; width: 16px; height: 16px; border-radius: 50%; background: #fff; transition: transform 0.18s ease; }
	.switch--on { background: #34d399; }
	.switch--on::after { transform: translateX(16px); }
	.note { margin-top: 14px; font-size: 0.78rem; color: var(--dt4); }
	.storage { margin-top: 24px; }
	.storage-head { display: flex; align-items: baseline; justify-content: space-between; gap: 12px; margin-bottom: 8px; }
	.storage-title { font-size: 0.72rem; font-weight: 640; letter-spacing: 0.05em; text-transform: uppercase; color: var(--dt3); }
	.storage-usage { font-size: 0.78rem; font-weight: 560; color: var(--dt2); white-space: nowrap; }
	.storage-usage--warn { color: #f59e0b; }
	.meter { width: 100%; height: 8px; border-radius: 999px; background: var(--dbg2); border: 1px solid var(--dbd); overflow: hidden; }
	.meter-fill { height: 100%; border-radius: 999px; background: var(--accent-blue); transition: width 0.25s ease, background 0.18s ease; }
	.meter-fill--warn { background: #f59e0b; }
	.meter-fill--full { background: #ef4444; }
	.accounts { margin-top: 24px; }
	.accounts-title { font-size: 0.72rem; font-weight: 640; letter-spacing: 0.05em; text-transform: uppercase; color: var(--dt3); margin: 0 0 8px; }
	.row--static { cursor: default; }
	.row.row--static:hover { border-color: var(--dbd); }
	.status-pill { font-size: 0.72rem; font-weight: 580; padding: 3px 10px; border-radius: 999px; background: color-mix(in srgb, var(--dt) 8%, transparent); color: var(--dt3); flex-shrink: 0; white-space: nowrap; }
	.status-pill--on { background: color-mix(in srgb, #34d399 14%, transparent); color: #34d399; }
	:global(.spin) { animation: spin 0.9s linear infinite; }
	@keyframes spin { to { transform: rotate(360deg); } }
</style>
