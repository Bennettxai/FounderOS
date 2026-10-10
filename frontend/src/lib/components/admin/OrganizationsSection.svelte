<script lang="ts">
	import { onMount } from 'svelte';
	import { adminApi, type AdminOrg, type AdminOrgDetail } from '$lib/api/admin';
	import { Loader2, Search, X, ChevronRight, Building2 } from 'lucide-svelte';
	import { fmtDate, initials } from './util';
	import OrgActions from './OrgActions.svelte';

	let { onForbidden }: { onForbidden?: () => void } = $props();

	let loading = $state(true);
	let error = $state<string | null>(null);
	let orgs = $state<AdminOrg[]>([]);
	let q = $state('');

	let detail = $state<AdminOrgDetail | null>(null);
	let panelOpen = $state(false);
	let panelLoading = $state(false);

	onMount(async () => {
		try {
			const res = await adminApi.listAllOrganizations();
			orgs = res.data ?? [];
		} catch (e) {
			const msg = e instanceof Error ? e.message : 'Failed to load organizations';
			if (msg.includes('403') || msg.toLowerCase().includes('forbidden')) onForbidden?.();
			else error = msg;
		} finally {
			loading = false;
		}
	});

	const filtered = $derived.by(() => {
		const t = q.trim().toLowerCase();
		if (!t) return orgs;
		return orgs.filter((o) => o.name.toLowerCase().includes(t) || (o.owner_email ?? '').toLowerCase().includes(t));
	});

	async function open(o: AdminOrg) {
		panelOpen = true;
		panelLoading = true;
		detail = null;
		try {
			detail = await adminApi.getOrganization(o.id);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load organization';
			panelOpen = false;
		} finally {
			panelLoading = false;
		}
	}
	function close() {
		panelOpen = false;
		detail = null;
	}
	async function reloadList() {
		try {
			const res = await adminApi.listAllOrganizations();
			orgs = res.data ?? [];
		} catch { /* keep current list */ }
	}
	async function onActionDone() {
		await reloadList();
		if (detail) {
			try {
				detail = await adminApi.getOrganization(detail.id);
			} catch {
				close();
			}
		}
	}
</script>

<div class="sec-head">
	<div class="search">
		<Search size={15} />
		<input placeholder="Search organizations…" bind:value={q} />
		{#if q}<button class="clr" onclick={() => (q = '')} aria-label="Clear"><X size={13} /></button>{/if}
	</div>
	<span class="count">{filtered.length} organizations</span>
</div>

{#if error}<div class="banner banner--error">{error}</div>{/if}

{#if loading}
	<div class="loading"><Loader2 class="spin" size={20} /> Loading organizations…</div>
{:else}
	<div class="table">
		<div class="thead trow"><span>Organization</span><span>Owner</span><span>Members</span><span>Workspaces</span><span>Created</span><span></span></div>
		{#each filtered as o (o.id)}
			<button class="trow clickable" onclick={() => open(o)}>
				<span class="strong">{o.name}<span class="slug">/{o.slug}</span></span>
				<span class="muted">{o.owner_email ?? '—'}</span>
				<span>{o.member_count}</span>
				<span>{o.workspace_count}</span>
				<span class="muted">{fmtDate(o.created_at)}</span>
				<ChevronRight size={15} class="chev" />
			</button>
			<!-- mobile card (hidden on desktop) -->
			<button class="mcard" onclick={() => open(o)}>
				<div class="mcard-body">
					<div class="mcard-name">{o.name}<span class="slug">/{o.slug}</span></div>
					<div class="mcard-meta">
						<span>{o.owner_email ?? '—'}</span>
						<span class="dot">·</span>
						<span>{o.member_count} members</span>
						<span class="dot">·</span>
						<span>{o.workspace_count} workspaces</span>
					</div>
					<div class="mcard-sub muted">{fmtDate(o.created_at)}</div>
				</div>
				<ChevronRight size={15} class="chev" />
			</button>
		{/each}
		{#if filtered.length === 0}<div class="empty-row">No organizations match.</div>{/if}
	</div>
{/if}

{#if panelOpen}
	<div class="overlay" role="button" tabindex="0" onclick={close} onkeydown={(e) => e.key === 'Escape' && close()}></div>
	<aside class="drawer">
		<header class="drawer-head">
			<h2>{detail?.name ?? 'Loading…'}</h2>
			<button class="clr" onclick={close} aria-label="Close"><X size={16} /></button>
		</header>
		{#if panelLoading}<div class="loading"><Loader2 class="spin" size={18} /></div>
		{:else if detail}
			<div class="drawer-body">
				<div class="d-meta"><span>Owner</span><b>{detail.owner_email ?? '—'}</b></div>
				<div class="d-meta"><span>Plan</span><b>{detail.plan ?? 'free'}</b></div>
				<div class="d-meta"><span>Created</span><b>{fmtDate(detail.created_at)}</b></div>
				<h3>Workspaces ({detail.workspaces.length})</h3>
				{#each detail.workspaces as w}
					<div class="m-row"><Building2 size={15} /><div class="m-meta"><span>{w.name}</span><span class="u-email">{w.member_count} members</span></div></div>
				{/each}
				<h3>Members ({detail.members.length})</h3>
				{#each detail.members as m}
					<div class="m-row"><div class="avatar sm">{initials(m.name, m.email)}</div><div class="m-meta"><span>{m.name || m.email}</span><span class="u-email">{m.email}</span></div><span class="role-tag">{m.role}</span></div>
				{/each}
				<OrgActions org={detail} onDone={onActionDone} />
			</div>
		{/if}
	</aside>
{/if}

<style>
	.sec-head { display: flex; align-items: center; justify-content: space-between; gap: 14px; margin-bottom: 16px; }
	.search { display: flex; align-items: center; gap: 7px; border: 1px solid var(--dbd); border-radius: 9px; padding: 6px 11px; color: var(--dt3); min-width: 260px; }
	.search input { background: transparent; border: none; outline: none; color: var(--dt); font-size: 0.84rem; flex: 1; }
	.clr { display: inline-flex; background: transparent; border: none; color: var(--dt3); cursor: pointer; padding: 2px; }
	.count { font-size: 0.78rem; color: var(--dt3); }
	.banner--error { background: color-mix(in srgb, #ef4444 12%, transparent); color: #ef4444; padding: 11px 14px; border-radius: 10px; font-size: 0.83rem; margin-bottom: 12px; }
	.loading { display: flex; align-items: center; justify-content: center; gap: 8px; color: var(--dt3); padding: 40px; }
	.table { display: flex; flex-direction: column; border: 1px solid var(--dbd); border-radius: 13px; overflow: hidden; }
	.trow { display: grid; grid-template-columns: 2fr 2fr 0.9fr 1fr 1.1fr 24px; align-items: center; gap: 14px; padding: 12px 16px; border-bottom: 1px solid var(--dbd); font-size: 0.85rem; text-align: left; }
	.trow:last-child { border-bottom: none; }
	.thead { background: color-mix(in srgb, var(--dt) 4%, transparent); color: var(--dt3); font-size: 0.72rem; text-transform: uppercase; letter-spacing: 0.04em; font-weight: 600; }
	.clickable { background: transparent; border: none; border-bottom: 1px solid var(--dbd); width: 100%; cursor: pointer; color: var(--dt); }
	.clickable:hover { background: color-mix(in srgb, var(--dt) 4%, transparent); }
	.strong { font-weight: 580; }
	.slug { color: var(--dt3); font-weight: 400; margin-left: 6px; font-size: 0.8rem; }
	.muted { color: var(--dt3); }
	:global(.chev) { color: var(--dt3); }
	.empty-row { padding: 28px; text-align: center; color: var(--dt3); font-size: 0.85rem; }
	.avatar { width: 30px; height: 30px; border-radius: 50%; display: flex; align-items: center; justify-content: center; background: linear-gradient(135deg, #6366f1, #8b5cf6); color: #fff; font-size: 0.78rem; font-weight: 650; flex-shrink: 0; }
	.avatar.sm { width: 26px; height: 26px; font-size: 0.7rem; }
	.u-email { font-size: 0.76rem; color: var(--dt3); }
	.overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.5); z-index: 40; }
	.drawer { position: fixed; top: 0; right: 0; bottom: 0; width: 420px; max-width: 92vw; background: var(--dbg); border-left: 1px solid var(--dbd); z-index: 41; display: flex; flex-direction: column; box-shadow: -20px 0 50px rgba(0,0,0,0.4); }
	.drawer-head { display: flex; align-items: center; justify-content: space-between; padding: 18px 20px; border-bottom: 1px solid var(--dbd); }
	.drawer-head h2 { font-size: 1.02rem; font-weight: 640; margin: 0; }
	.drawer-body { padding: 18px 20px; overflow-y: auto; flex: 1; }
	.d-meta { display: flex; justify-content: space-between; padding: 8px 0; border-bottom: 1px solid var(--dbd); font-size: 0.85rem; }
	.d-meta span { color: var(--dt3); }
	.drawer-body h3 { font-size: 0.74rem; text-transform: uppercase; letter-spacing: 0.05em; color: var(--dt3); margin: 20px 0 10px; }
	.m-row { display: flex; align-items: center; gap: 10px; padding: 8px 0; font-size: 0.84rem; }
	.m-meta { display: flex; flex-direction: column; flex: 1; min-width: 0; }
	.role-tag { font-size: 0.72rem; color: var(--dt3); border: 1px solid var(--dbd); padding: 2px 8px; border-radius: 999px; text-transform: capitalize; }
	/* mobile card - hidden by default */
	.mcard { display: none; }
	:global(.spin) { animation: spin 0.9s linear infinite; }
	@keyframes spin { to { transform: rotate(360deg); } }
	@media (max-width: 768px) {
		.sec-head { flex-wrap: wrap; gap: 8px; }
		.search { min-width: 0; flex: 1; }
		/* hide desktop rows */
		.thead { display: none; }
		.trow { display: none; }
		/* show mobile cards */
		.mcard { display: flex; align-items: center; justify-content: space-between; gap: 10px; padding: 14px 16px; border-bottom: 1px solid var(--dbd); background: transparent; border-left: none; border-right: none; border-top: none; border-radius: 0; width: 100%; cursor: pointer; color: var(--dt); text-align: left; }
		.mcard:hover { background: color-mix(in srgb, var(--dt) 4%, transparent); }
		.mcard-body { display: flex; flex-direction: column; gap: 3px; min-width: 0; flex: 1; }
		.mcard-name { font-weight: 580; font-size: 0.88rem; }
		.mcard-meta { font-size: 0.76rem; color: var(--dt3); display: flex; flex-wrap: wrap; gap: 4px; align-items: center; }
		.dot { color: var(--dbd); }
		.mcard-sub { font-size: 0.74rem; color: var(--dt3); }
		/* full-width drawer on mobile */
		.drawer { width: 100vw; max-width: 100vw; border-left: none; border-top: 1px solid var(--dbd); top: 0; }
	}
</style>
