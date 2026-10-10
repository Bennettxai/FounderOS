<script lang="ts">
	import { onMount } from 'svelte';
	import { adminApi, type AdminConnection } from '$lib/api/admin';
	import { Loader2, Search, X, Plug, CheckCircle2, Circle } from 'lucide-svelte';

	let { onForbidden }: { onForbidden?: () => void } = $props();

	let loading = $state(true);
	let error = $state<string | null>(null);
	let items = $state<AdminConnection[]>([]);
	let q = $state('');

	onMount(async () => {
		try {
			const res = await adminApi.listAllConnections();
			items = res.data ?? [];
		} catch (e) {
			const msg = e instanceof Error ? e.message : 'Failed to load connections';
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
			(u) => u.email.toLowerCase().includes(t) || (u.name ?? '').toLowerCase().includes(t)
		);
	});

	const googleCount = $derived(items.filter((u) => u.google_connected).length);
	const totalCount = $derived(items.length);
</script>

<div class="sec-head">
	<div class="left-col">
		<div class="search">
			<Search size={15} />
			<input placeholder="Search users…" bind:value={q} />
			{#if q}<button class="clr" onclick={() => (q = '')} aria-label="Clear"><X size={13} /></button>{/if}
		</div>
		{#if !loading}
			<span class="summary-pill" class:connected={googleCount > 0}>
				<Plug size={12} />
				{googleCount} of {totalCount} connected Google
			</span>
		{/if}
	</div>
	<span class="count">{filtered.length} users</span>
</div>

<p class="desc">
	Every user connects their own accounts (Google, Slack, Notion). Tokens are stored per user, so one
	person connecting never exposes another's account. This is who has wired up what.
</p>

{#if error}<div class="banner banner--error">{error}</div>{/if}

{#if loading}
	<div class="loading"><Loader2 class="spin" size={20} /> Loading connections…</div>
{:else if filtered.length === 0}
	<div class="empty"><Plug size={24} strokeWidth={1.4} /><p>No users match.</p></div>
{:else}
	<div class="table-wrap">
		<table class="tbl">
			<thead>
				<tr><th>User</th><th>Google</th><th>Slack</th><th>Notion</th><th>Other</th></tr>
			</thead>
			<tbody>
				{#each filtered as u (u.id)}
					<tr>
						<td>
							<div class="u-name">{u.name || u.email}</div>
							{#if u.name}<div class="u-email">{u.email}</div>{/if}
						</td>
						<td>
							{#if u.google_connected}
								<span class="pill pill--ok"><CheckCircle2 size={12} /> {u.google_email ?? 'Connected'}</span>
							{:else}
								<span class="pill pill--muted"><Circle size={11} /> Not connected</span>
							{/if}
						</td>
						<td>
							{#if u.slack_connected}<span class="pill pill--ok"><CheckCircle2 size={12} /> Connected</span>
							{:else}<span class="pill pill--muted"><Circle size={11} /> —</span>{/if}
						</td>
						<td>
							{#if u.notion_connected}<span class="pill pill--ok"><CheckCircle2 size={12} /> Connected</span>
							{:else}<span class="pill pill--muted"><Circle size={11} /> —</span>{/if}
						</td>
						<td>
							{#if u.other_connections > 0}<span class="pill pill--ok">{u.other_connections}</span>
							{:else}<span class="pill pill--muted">0</span>{/if}
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
{/if}

<style>
	.sec-head { display: flex; align-items: center; justify-content: space-between; gap: 12px; flex-wrap: wrap; margin-bottom: 6px; }
	.left-col { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
	.search { display: flex; align-items: center; gap: 7px; padding: 7px 11px; border: 1px solid var(--dbd); border-radius: 9px; color: var(--dt3); }
	.search input { background: transparent; border: none; outline: none; color: var(--dt); font-size: 0.84rem; width: 180px; font-family: inherit; }
	.clr { display: inline-flex; background: transparent; border: none; color: var(--dt3); cursor: pointer; }
	.summary-pill { display: inline-flex; align-items: center; gap: 6px; font-size: 0.75rem; color: var(--dt3); border: 1px solid var(--dbd); padding: 4px 10px; border-radius: 999px; }
	.summary-pill.connected { color: #22c55e; border-color: color-mix(in srgb, #22c55e 40%, transparent); }
	.count { font-size: 0.76rem; color: var(--dt3); }
	.desc { font-size: 0.82rem; color: var(--dt3); line-height: 1.5; margin: 4px 0 16px; max-width: 640px; }
	.banner { padding: 11px 14px; border-radius: 10px; font-size: 0.83rem; margin-bottom: 12px; }
	.banner--error { background: color-mix(in srgb, #ef4444 12%, transparent); color: #ef4444; }
	.loading, .empty { display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 12px; color: var(--dt3); padding: 50px 0; }
	.table-wrap { overflow-x: auto; border: 1px solid var(--dbd); border-radius: 12px; }
	.tbl { width: 100%; border-collapse: collapse; font-size: 0.85rem; }
	.tbl th { text-align: left; font-size: 0.72rem; text-transform: uppercase; letter-spacing: 0.05em; color: var(--dt3); font-weight: 620; padding: 11px 14px; border-bottom: 1px solid var(--dbd); white-space: nowrap; }
	.tbl td { padding: 11px 14px; border-bottom: 1px solid color-mix(in srgb, var(--dt) 5%, transparent); vertical-align: top; }
	.tbl tr:last-child td { border-bottom: none; }
	.u-name { font-weight: 580; color: var(--dt); }
	.u-email { font-size: 0.78rem; color: var(--dt3); margin-top: 2px; }
	.pill { display: inline-flex; align-items: center; gap: 5px; font-size: 0.74rem; font-weight: 520; padding: 3px 9px; border-radius: 999px; white-space: nowrap; }
	.pill--ok { color: #22c55e; background: color-mix(in srgb, #22c55e 13%, transparent); }
	.pill--muted { color: var(--dt3); background: color-mix(in srgb, var(--dt) 6%, transparent); }
	:global(.spin) { animation: spin 0.9s linear infinite; }
	@keyframes spin { to { transform: rotate(360deg); } }
</style>
