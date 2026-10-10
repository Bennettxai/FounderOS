<script lang="ts">
	import { onMount } from 'svelte';
	import { adminApi, type AdminDashboard } from '$lib/api/admin';
	import { Loader2 } from 'lucide-svelte';

	let { onForbidden }: { onForbidden?: () => void } = $props();

	let loading = $state(true);
	let error = $state<string | null>(null);
	let dashboard = $state<AdminDashboard | null>(null);
	let orgCount = $state(0);
	let superadmins = $state(0);

	onMount(async () => {
		try {
			const [d, o, u] = await Promise.all([
				adminApi.dashboard(),
				adminApi.listOrganizations(),
				adminApi.listAllUsers()
			]);
			dashboard = d;
			orgCount = o.pagination?.total_items ?? o.data.length;
			superadmins = (u.data ?? []).filter((x) => x.platform_role === 'superadmin').length;
		} catch (e) {
			const msg = e instanceof Error ? e.message : 'Failed to load overview';
			if (msg.includes('403') || msg.toLowerCase().includes('forbidden')) onForbidden?.();
			else error = msg;
		} finally {
			loading = false;
		}
	});
</script>

{#if loading}
	<div class="loading"><Loader2 class="spin" size={20} /> Loading platform overview…</div>
{:else if error}
	<div class="banner banner--error">{error}</div>
{:else if dashboard}
	<div class="stat-grid">
		<div class="stat">
			<span class="stat-label">Total Users</span>
			<span class="stat-val">{dashboard.users.total}</span>
			<span class="stat-sub">+{dashboard.users.new_today} today</span>
		</div>
		<div class="stat">
			<span class="stat-label">Workspaces</span>
			<span class="stat-val">{dashboard.workspaces.total}</span>
			<span class="stat-sub">{dashboard.workspaces.active} active</span>
		</div>
		<div class="stat">
			<span class="stat-label">Organizations</span>
			<span class="stat-val">{orgCount}</span>
			<span class="stat-sub">across platform</span>
		</div>
		<div class="stat">
			<span class="stat-label">Computers</span>
			<span class="stat-val">{dashboard.computers.total}</span>
			<span class="stat-sub">{dashboard.computers.running} running</span>
		</div>
		<div class="stat">
			<span class="stat-label">Paid Subscriptions</span>
			<span class="stat-val">{dashboard.revenue.active_subscriptions}</span>
			<span class="stat-sub">{dashboard.revenue.status === 'unavailable' ? 'MRR n/a' : 'active'}</span>
		</div>
		<div class="stat">
			<span class="stat-label">Superadmins</span>
			<span class="stat-val">{superadmins}</span>
			<span class="stat-sub">platform owners</span>
		</div>
	</div>
{/if}

<style>
	.loading { display: flex; align-items: center; justify-content: center; gap: 8px; color: var(--dt3); padding: 40px; }
	.banner--error { background: color-mix(in srgb, #ef4444 12%, transparent); color: #ef4444; padding: 11px 14px; border-radius: 10px; font-size: 0.83rem; }
	.stat-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(190px, 1fr)); gap: 14px; }
	@media (max-width: 480px) {
		.stat-grid { grid-template-columns: 1fr 1fr; gap: 10px; }
	}
	.stat { border: 1px solid var(--dbd); border-radius: 13px; padding: 16px 18px; display: flex; flex-direction: column; gap: 4px; background: color-mix(in srgb, var(--dt) 2%, transparent); }
	.stat-label { font-size: 0.74rem; color: var(--dt3); text-transform: uppercase; letter-spacing: 0.04em; }
	.stat-val { font-size: 1.7rem; font-weight: 700; letter-spacing: -0.02em; }
	.stat-sub { font-size: 0.76rem; color: var(--dt3); }
	:global(.spin) { animation: spin 0.9s linear infinite; }
	@keyframes spin { to { transform: rotate(360deg); } }
</style>
