<script lang="ts">
	import { onMount } from 'svelte';
	import { adminApi, type AdminUser } from '$lib/api/admin';
	import { Loader2, Search, X, Crown } from 'lucide-svelte';
	import { fmtDate, initials } from './util';

	let { onForbidden }: { onForbidden?: () => void } = $props();

	let loading = $state(true);
	let error = $state<string | null>(null);
	let users = $state<AdminUser[]>([]);
	let q = $state('');
	let busyId = $state<string | null>(null);

	const ROLES = ['user', 'admin', 'superadmin'];
	const PLANS = ['free', 'pro', 'team', 'enterprise'];

	onMount(async () => {
		try {
			const res = await adminApi.listAllUsers();
			users = res.data ?? [];
		} catch (e) {
			const msg = e instanceof Error ? e.message : 'Failed to load users';
			if (msg.includes('403') || msg.toLowerCase().includes('forbidden')) onForbidden?.();
			else error = msg;
		} finally {
			loading = false;
		}
	});

	const filtered = $derived.by(() => {
		const t = q.trim().toLowerCase();
		if (!t) return users;
		return users.filter((u) => u.email.toLowerCase().includes(t) || (u.name ?? '').toLowerCase().includes(t));
	});

	async function changeRole(u: AdminUser, role: string) {
		if (role === u.platform_role) return;
		busyId = u.id;
		try {
			await adminApi.setUserRole(u.id, role);
			users = users.map((x) => (x.id === u.id ? { ...x, platform_role: role } : x));
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to set role';
		} finally {
			busyId = null;
		}
	}
	async function changePlan(u: AdminUser, plan: string) {
		busyId = u.id;
		try {
			await adminApi.setUserPlan(u.id, plan);
			users = users.map((x) => (x.id === u.id ? { ...x, plan } : x));
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to set plan';
		} finally {
			busyId = null;
		}
	}
</script>

<div class="sec-head">
	<div class="search">
		<Search size={15} />
		<input placeholder="Search users…" bind:value={q} />
		{#if q}<button class="clr" onclick={() => (q = '')} aria-label="Clear"><X size={13} /></button>{/if}
	</div>
	<span class="count">{filtered.length} users</span>
</div>

{#if error}<div class="banner banner--error">{error}</div>{/if}

{#if loading}
	<div class="loading"><Loader2 class="spin" size={20} /> Loading users…</div>
{:else}
	<div class="table">
		<div class="thead trow"><span>User</span><span>Role</span><span>Plan</span><span>Workspace</span><span>Joined</span></div>
		{#each filtered as u (u.id)}
			<div class="trow" class:busy={busyId === u.id}>
				<div class="cell-user">
					<div class="avatar">{initials(u.name, u.email)}</div>
					<div class="u-meta">
						<span class="u-name">{u.name || '(no name)'}{#if u.platform_role === 'superadmin'}<Crown size={12} class="mini-crown" />{/if}</span>
						<span class="u-email">{u.email}</span>
					</div>
				</div>
				<select value={u.platform_role} onchange={(e) => changeRole(u, e.currentTarget.value)}>
					{#each ROLES as r}<option value={r}>{r}</option>{/each}
				</select>
				<select value={u.plan ?? 'free'} onchange={(e) => changePlan(u, e.currentTarget.value)}>
					{#each PLANS as p}<option value={p}>{p}</option>{/each}
				</select>
				<span class="muted">{u.workspace_name ?? '—'}</span>
				<span class="muted">{fmtDate(u.created_at)}</span>
			</div>
			<!-- mobile card (hidden on desktop) -->
			<div class="mcard" class:busy={busyId === u.id}>
				<div class="mcard-head">
					<div class="avatar">{initials(u.name, u.email)}</div>
					<div class="u-meta">
						<span class="u-name">{u.name || '(no name)'}{#if u.platform_role === 'superadmin'}<Crown size={12} class="mini-crown" />{/if}</span>
						<span class="u-email">{u.email}</span>
					</div>
				</div>
				<div class="mcard-fields">
					<div class="mfield">
						<span class="mfield-label">Role</span>
						<select value={u.platform_role} onchange={(e) => changeRole(u, e.currentTarget.value)}>
							{#each ROLES as r}<option value={r}>{r}</option>{/each}
						</select>
					</div>
					<div class="mfield">
						<span class="mfield-label">Plan</span>
						<select value={u.plan ?? 'free'} onchange={(e) => changePlan(u, e.currentTarget.value)}>
							{#each PLANS as p}<option value={p}>{p}</option>{/each}
						</select>
					</div>
					<div class="mfield">
						<span class="mfield-label">Workspace</span>
						<span class="muted">{u.workspace_name ?? '—'}</span>
					</div>
					<div class="mfield">
						<span class="mfield-label">Joined</span>
						<span class="muted">{fmtDate(u.created_at)}</span>
					</div>
				</div>
			</div>
		{/each}
		{#if filtered.length === 0}<div class="empty-row">No users match.</div>{/if}
	</div>
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
	.trow { display: grid; grid-template-columns: 2.4fr 1fr 1fr 1.4fr 1fr; align-items: center; gap: 14px; padding: 12px 16px; border-bottom: 1px solid var(--dbd); font-size: 0.85rem; }
	.trow:last-child { border-bottom: none; }
	.thead { background: color-mix(in srgb, var(--dt) 4%, transparent); color: var(--dt3); font-size: 0.72rem; text-transform: uppercase; letter-spacing: 0.04em; font-weight: 600; }
	.trow.busy { opacity: 0.55; pointer-events: none; }
	.cell-user { display: flex; align-items: center; gap: 10px; min-width: 0; }
	.avatar { width: 30px; height: 30px; border-radius: 50%; display: flex; align-items: center; justify-content: center; background: linear-gradient(135deg, #6366f1, #8b5cf6); color: #fff; font-size: 0.78rem; font-weight: 650; flex-shrink: 0; }
	.u-meta { display: flex; flex-direction: column; min-width: 0; }
	.u-name { font-weight: 560; display: inline-flex; align-items: center; gap: 5px; }
	:global(.mini-crown) { color: #fbbf24; }
	.u-email { font-size: 0.76rem; color: var(--dt3); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
	.muted { color: var(--dt3); }
	select { background: var(--dbg2, #111); border: 1px solid var(--dbd); border-radius: 8px; color: var(--dt); padding: 5px 8px; font-size: 0.8rem; cursor: pointer; min-height: 40px; }
	.empty-row { padding: 28px; text-align: center; color: var(--dt3); font-size: 0.85rem; }
	/* mobile card - hidden by default */
	.mcard { display: none; }
	:global(.spin) { animation: spin 0.9s linear infinite; }
	@keyframes spin { to { transform: rotate(360deg); } }
	@media (max-width: 768px) {
		.sec-head { flex-wrap: wrap; gap: 8px; }
		.search { min-width: 0; flex: 1; }
		/* hide desktop table rows and header */
		.thead { display: none; }
		.trow { display: none; }
		/* show mobile cards */
		.mcard { display: flex; flex-direction: column; gap: 10px; padding: 14px 16px; border-bottom: 1px solid var(--dbd); }
		.mcard:last-child { border-bottom: none; }
		.mcard.busy { opacity: 0.55; pointer-events: none; }
		.mcard-head { display: flex; align-items: center; gap: 10px; }
		.mcard-fields { display: grid; grid-template-columns: 1fr 1fr; gap: 8px; }
		.mfield { display: flex; flex-direction: column; gap: 4px; }
		.mfield-label { font-size: 0.68rem; text-transform: uppercase; letter-spacing: 0.04em; color: var(--dt3); font-weight: 600; }
		.mfield select { width: 100%; min-height: 40px; }
	}
	@media (max-width: 480px) {
		.mcard-fields { grid-template-columns: 1fr; }
	}
</style>
