<script lang="ts">
	import { onMount } from 'svelte';
	import { adminApi, type AdminEngineStatus } from '$lib/api/admin';
	import { Loader2, Search, X, Cpu, CheckCircle2, Circle } from 'lucide-svelte';

	let { onForbidden }: { onForbidden?: () => void } = $props();

	let loading = $state(true);
	let error = $state<string | null>(null);
	let items = $state<AdminEngineStatus[]>([]);
	let q = $state('');
	// tracks which workspace ids have had a reminder sent this session
	let sentIds = $state<Set<string>>(new Set());
	let busyId = $state<string | null>(null);

	onMount(async () => {
		try {
			const res = await adminApi.listAllEngineStatus();
			items = res.data ?? [];
		} catch (e) {
			const msg = e instanceof Error ? e.message : 'Failed to load engine status';
			if (msg.includes('403') || msg.toLowerCase().includes('forbidden')) onForbidden?.();
			else error = msg;
		} finally {
			loading = false;
		}
	});

	const filtered = $derived.by(() => {
		const t = q.trim().toLowerCase();
		if (!t) return items;
		return items.filter(
			(w) =>
				w.name.toLowerCase().includes(t) ||
				(w.owner_email ?? '').toLowerCase().includes(t) ||
				(w.slug ?? '').toLowerCase().includes(t)
		);
	});

	const connectedCount = $derived(items.filter((w) => w.engine_configured && w.engine_enabled).length);
	const totalCount = $derived(items.length);

	async function sendReminder(w: AdminEngineStatus) {
		if (busyId) return;
		busyId = w.id;
		try {
			await adminApi.sendEngineReminder(w.id);
			sentIds = new Set([...sentIds, w.id]);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to send reminder';
		} finally {
			busyId = null;
		}
	}
</script>

<div class="sec-head">
	<div class="left-col">
		<div class="search">
			<Search size={15} />
			<input placeholder="Search workspaces…" bind:value={q} />
			{#if q}<button class="clr" onclick={() => (q = '')} aria-label="Clear"><X size={13} /></button>{/if}
		</div>
		{#if !loading}
			<span class="summary-pill" class:connected={connectedCount > 0}>
				<Cpu size={12} />
				{connectedCount} of {totalCount} workspaces connected
			</span>
		{/if}
	</div>
	<span class="count">{filtered.length} workspaces</span>
</div>

<div class="engine-desc">
	The Optimal Engine is each workspace's second brain — local-first, synced to cloud. Workspaces without it lose AI memory, semantic search, and context assembly.
</div>

{#if error}<div class="banner banner--error">{error}</div>{/if}

{#if loading}
	<div class="loading"><Loader2 class="spin" size={20} /> Loading engine status…</div>
{:else}
	<div class="table">
		<div class="thead trow"><span>Workspace</span><span>Owner</span><span>Members</span><span>Status</span><span>Engine Host</span><span></span></div>
		{#each filtered as w (w.id)}
			{@const connected = w.engine_configured && w.engine_enabled}
			<div class="trow">
				<span class="strong">{w.name}<span class="slug">/{w.slug}</span></span>
				<span class="muted">{w.owner_email ?? '—'}</span>
				<span>{w.member_count}</span>
				<span>
					{#if connected}
						<span class="pill pill--green"><CheckCircle2 size={11} /> Connected</span>
					{:else}
						<span class="pill pill--muted"><Circle size={11} /> Not connected</span>
					{/if}
				</span>
				<span class="muted mono">{w.engine_host ?? '—'}</span>
				<span class="action-cell">
					{#if !connected}
						{#if sentIds.has(w.id)}
							<span class="sent-label">Sent</span>
						{:else}
							<button
								class="remind-btn"
								onclick={() => sendReminder(w)}
								disabled={busyId === w.id}
								aria-label="Send engine setup reminder to {w.owner_email ?? w.name}"
							>
								{#if busyId === w.id}<Loader2 size={12} class="spin" />{/if}
								Remind
							</button>
						{/if}
					{/if}
				</span>
			</div>
			<!-- mobile card (hidden on desktop) -->
			<div class="mcard">
				<div class="mcard-top">
					<div class="mcard-name">{w.name}<span class="slug">/{w.slug}</span></div>
					{#if connected}
						<span class="pill pill--green"><CheckCircle2 size={11} /> Connected</span>
					{:else}
						<span class="pill pill--muted"><Circle size={11} /> Not connected</span>
					{/if}
				</div>
				<div class="mcard-fields">
					<div class="mfield"><span class="mfield-label">Owner</span><span class="muted">{w.owner_email ?? '—'}</span></div>
					<div class="mfield"><span class="mfield-label">Members</span><span>{w.member_count}</span></div>
					<div class="mfield mfield--wide"><span class="mfield-label">Engine Host</span><span class="muted mono">{w.engine_host ?? '—'}</span></div>
				</div>
				{#if !connected}
					<div class="mcard-action">
						{#if sentIds.has(w.id)}
							<span class="sent-label">Sent</span>
						{:else}
							<button
								class="remind-btn"
								onclick={() => sendReminder(w)}
								disabled={busyId === w.id}
								aria-label="Send engine setup reminder to {w.owner_email ?? w.name}"
							>
								{#if busyId === w.id}<Loader2 size={12} class="spin" />{/if}
								Remind
							</button>
						{/if}
					</div>
				{/if}
			</div>
		{/each}
		{#if filtered.length === 0}<div class="empty-row">No workspaces match.</div>{/if}
	</div>
{/if}

<style>
	.sec-head { display: flex; align-items: center; justify-content: space-between; gap: 14px; margin-bottom: 10px; }
	.left-col { display: flex; align-items: center; gap: 12px; }
	.search { display: flex; align-items: center; gap: 7px; border: 1px solid var(--dbd); border-radius: 9px; padding: 6px 11px; color: var(--dt3); min-width: 260px; }
	.search input { background: transparent; border: none; outline: none; color: var(--dt); font-size: 0.84rem; flex: 1; }
	.clr { display: inline-flex; background: transparent; border: none; color: var(--dt3); cursor: pointer; padding: 2px; }
	.count { font-size: 0.78rem; color: var(--dt3); }
	.summary-pill { display: inline-flex; align-items: center; gap: 5px; font-size: 0.75rem; color: var(--dt3); border: 1px solid var(--dbd); border-radius: 999px; padding: 3px 10px; }
	.summary-pill.connected { color: #4ade80; border-color: color-mix(in srgb, #4ade80 30%, transparent); }
	.engine-desc { font-size: 0.8rem; color: var(--dt3); margin-bottom: 16px; line-height: 1.5; max-width: 680px; }
	.banner--error { background: color-mix(in srgb, #ef4444 12%, transparent); color: #ef4444; padding: 11px 14px; border-radius: 10px; font-size: 0.83rem; margin-bottom: 12px; }
	.loading { display: flex; align-items: center; justify-content: center; gap: 8px; color: var(--dt3); padding: 40px; }
	.table { display: flex; flex-direction: column; border: 1px solid var(--dbd); border-radius: 13px; overflow: hidden; }
	.trow { display: grid; grid-template-columns: 2fr 2fr 0.7fr 1.1fr 1.6fr 90px; align-items: center; gap: 14px; padding: 12px 16px; border-bottom: 1px solid var(--dbd); font-size: 0.85rem; }
	.trow:last-child { border-bottom: none; }
	.thead { background: color-mix(in srgb, var(--dt) 4%, transparent); color: var(--dt3); font-size: 0.72rem; text-transform: uppercase; letter-spacing: 0.04em; font-weight: 600; }
	.strong { font-weight: 580; }
	.slug { color: var(--dt3); font-weight: 400; margin-left: 6px; font-size: 0.8rem; }
	.muted { color: var(--dt3); }
	.mono { font-family: ui-monospace, monospace; font-size: 0.78rem; }
	.pill { display: inline-flex; align-items: center; gap: 5px; font-size: 0.75rem; padding: 3px 9px; border-radius: 999px; font-weight: 560; }
	.pill--green { background: color-mix(in srgb, #4ade80 12%, transparent); color: #4ade80; }
	.pill--muted { background: color-mix(in srgb, var(--dt3) 10%, transparent); color: var(--dt3); }
	.action-cell { display: flex; justify-content: flex-end; }
	.remind-btn { background: color-mix(in srgb, var(--dt) 8%, transparent); border: 1px solid var(--dbd); border-radius: 8px; color: var(--dt); padding: 5px 12px; font-size: 0.78rem; cursor: pointer; display: inline-flex; align-items: center; gap: 5px; }
	.remind-btn:hover:not(:disabled) { background: color-mix(in srgb, var(--dt) 14%, transparent); }
	.remind-btn:disabled { opacity: 0.5; pointer-events: none; }
	.sent-label { font-size: 0.76rem; color: #4ade80; font-weight: 560; }
	.empty-row { padding: 28px; text-align: center; color: var(--dt3); font-size: 0.85rem; }
	/* mobile card - hidden by default */
	.mcard { display: none; }
	:global(.spin) { animation: spin 0.9s linear infinite; }
	@keyframes spin { to { transform: rotate(360deg); } }
	@media (max-width: 768px) {
		.sec-head { flex-wrap: wrap; gap: 8px; }
		.left-col { flex-wrap: wrap; gap: 8px; }
		.search { min-width: 0; flex: 1; }
		.summary-pill { flex-shrink: 0; }
		/* hide desktop rows */
		.thead { display: none; }
		.trow { display: none; }
		/* show mobile cards */
		.mcard { display: flex; flex-direction: column; gap: 10px; padding: 14px 16px; border-bottom: 1px solid var(--dbd); }
		.mcard:last-child { border-bottom: none; }
		.mcard-top { display: flex; align-items: center; justify-content: space-between; gap: 8px; }
		.mcard-name { font-weight: 580; font-size: 0.88rem; }
		.mcard-fields { display: grid; grid-template-columns: 1fr 1fr; gap: 8px; }
		.mfield { display: flex; flex-direction: column; gap: 3px; }
		.mfield--wide { grid-column: span 2; }
		.mfield-label { font-size: 0.68rem; text-transform: uppercase; letter-spacing: 0.04em; color: var(--dt3); font-weight: 600; }
		.mcard-action { display: flex; justify-content: flex-end; }
		.remind-btn { min-height: 40px; padding: 8px 16px; }
	}
	@media (max-width: 480px) {
		.mcard-fields { grid-template-columns: 1fr; }
		.mfield--wide { grid-column: span 1; }
	}
</style>
