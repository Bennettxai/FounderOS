<script lang="ts">
	import { onMount } from 'svelte';
	import { currentWorkspace } from '$lib/stores/workspaces';
	import {
		getFinanceSummary,
		getFinanceDeals,
		type FinanceSummary,
		type FinanceDeal,
		type FinanceDealStatus
	} from '$lib/api/finance';
	import { Wallet, Loader2, RefreshCw, TrendingUp, CircleDollarSign, Clock, XCircle } from 'lucide-svelte';

	let summary = $state<FinanceSummary | null>(null);
	let deals = $state<FinanceDeal[]>([]);
	let loading = $state(true);
	let refreshing = $state(false);
	let error = $state<string | null>(null);
	let filter = $state<FinanceDealStatus | 'all'>('won');

	const FILTERS: { id: FinanceDealStatus | 'all'; label: string }[] = [
		{ id: 'won', label: 'Booked' },
		{ id: 'open', label: 'Pipeline' },
		{ id: 'lost', label: 'Lost' },
		{ id: 'all', label: 'All' }
	];

	// Reload when the active workspace changes (finance is workspace-scoped).
	let wsId = $state<string | null | undefined>(null);
	$effect(() => {
		const id = $currentWorkspace?.id ?? null;
		if (id !== wsId) {
			wsId = id;
			load();
		}
	});

	onMount(load);

	async function fetchAll() {
		[summary, deals] = await Promise.all([getFinanceSummary(), getFinanceDeals({ limit: 200 })]);
	}

	async function load() {
		loading = true;
		error = null;
		try {
			await fetchAll();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load finance data';
		} finally {
			loading = false;
		}
	}

	async function refresh() {
		refreshing = true;
		error = null;
		try {
			await fetchAll();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to refresh finance data';
		} finally {
			refreshing = false;
		}
	}

	// Currency: use the dominant currency across deals, default to USD.
	const currency = $derived.by(() => {
		const counts = new Map<string, number>();
		for (const d of deals) {
			const c = (d.currency || 'USD').toUpperCase();
			counts.set(c, (counts.get(c) ?? 0) + 1);
		}
		let best = 'USD';
		let max = 0;
		for (const [c, n] of counts) {
			if (n > max) {
				max = n;
				best = c;
			}
		}
		return best;
	});

	function fmtMoney(value: number | undefined): string {
		const n = value ?? 0;
		try {
			return new Intl.NumberFormat(undefined, {
				style: 'currency',
				currency,
				maximumFractionDigits: n % 1 === 0 ? 0 : 2
			}).format(n);
		} catch {
			return `$${n.toLocaleString()}`;
		}
	}

	function fmtDate(d: string | undefined): string {
		if (!d) return '—';
		const parsed = new Date(d);
		if (Number.isNaN(parsed.getTime())) return '—';
		return parsed.toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' });
	}

	const winRate = $derived.by(() => {
		if (!summary) return null;
		if (typeof summary.win_rate === 'number') return summary.win_rate;
		const closed = summary.won_deals + summary.lost_deals;
		if (closed === 0) return null;
		return (summary.won_deals / closed) * 100;
	});

	const filteredDeals = $derived.by(() => {
		const list = filter === 'all' ? deals : deals.filter((d) => (d.status ?? 'open') === filter);
		return [...list].sort((a, b) => (b.amount ?? 0) - (a.amount ?? 0));
	});

	// Booked revenue grouped by client (company), highest first.
	const revenueByClient = $derived.by(() => {
		const won = deals.filter((d) => d.status === 'won');
		const byClient = new Map<string, { total: number; count: number }>();
		for (const d of won) {
			const name = d.company_name?.trim() || 'Unassigned';
			const cur = byClient.get(name) ?? { total: 0, count: 0 };
			cur.total += d.amount ?? 0;
			cur.count += 1;
			byClient.set(name, cur);
		}
		return [...byClient.entries()]
			.map(([name, v]) => ({ name, ...v }))
			.sort((a, b) => b.total - a.total)
			.slice(0, 6);
	});

	const maxClientTotal = $derived(revenueByClient.reduce((m, c) => Math.max(m, c.total), 0));

	function statusLabel(s: FinanceDealStatus | undefined): string {
		if (s === 'won') return 'Booked';
		if (s === 'lost') return 'Lost';
		return 'Open';
	}
</script>

<svelte:head><title>Finance - BusinessOS</title></svelte:head>

<div class="fin-root">
	<header class="topbar">
		<div class="title-wrap">
			<div class="page-icon"><Wallet size={18} strokeWidth={1.9} /></div>
			<div>
				<h1>Finance</h1>
				<p class="sub">Revenue booked and in pipeline across clients and projects.</p>
			</div>
		</div>
		<button class="btn btn--ghost" onclick={refresh} disabled={loading || refreshing} aria-label="Refresh finance data">
			{#if refreshing}<Loader2 class="spin" size={15} />{:else}<RefreshCw size={15} />{/if}
			Refresh
		</button>
	</header>

	{#if error}<div class="banner banner--error">{error}</div>{/if}

	{#if loading}
		<div class="loading"><Loader2 class="spin" size={20} /> Loading finance…</div>
	{:else if summary && summary.total_deals === 0}
		<div class="empty">
			<Wallet size={38} strokeWidth={1.4} />
			<p class="empty-title">No financial data yet</p>
			<p class="empty-body">
				Revenue appears here as you add deals in the pipeline. Book a deal to start tracking income.
			</p>
			<a class="btn btn--primary" href="/pipelines">Go to pipeline</a>
		</div>
	{:else}
		<div class="scroll">
			<!-- Summary tiles -->
			<section class="tiles">
				<div class="tile tile--accent">
					<div class="tile-top"><CircleDollarSign size={16} /><span>Booked revenue</span></div>
					<div class="tile-value">{fmtMoney(summary?.won_value)}</div>
					<div class="tile-meta">{summary?.won_deals ?? 0} won deal{(summary?.won_deals ?? 0) === 1 ? '' : 's'}</div>
				</div>
				<div class="tile">
					<div class="tile-top"><Clock size={16} /><span>Open pipeline</span></div>
					<div class="tile-value">{fmtMoney(summary?.open_value)}</div>
					<div class="tile-meta">{summary?.open_deals ?? 0} open deal{(summary?.open_deals ?? 0) === 1 ? '' : 's'}</div>
				</div>
				<div class="tile">
					<div class="tile-top"><TrendingUp size={16} /><span>Win rate</span></div>
					<div class="tile-value">{winRate === null ? '—' : `${winRate.toFixed(0)}%`}</div>
					<div class="tile-meta">{summary?.won_deals ?? 0} won · {summary?.lost_deals ?? 0} lost</div>
				</div>
				<div class="tile">
					<div class="tile-top"><XCircle size={16} /><span>Lost value</span></div>
					<div class="tile-value">{fmtMoney(summary?.lost_value)}</div>
					<div class="tile-meta">{summary?.lost_deals ?? 0} lost deal{(summary?.lost_deals ?? 0) === 1 ? '' : 's'}</div>
				</div>
			</section>

			<!-- Revenue by client -->
			{#if revenueByClient.length > 0}
				<section class="panel">
					<div class="panel-head">
						<h2>Booked revenue by client</h2>
						<span class="panel-note">Top {revenueByClient.length}</span>
					</div>
					<div class="bars">
						{#each revenueByClient as c (c.name)}
							<div class="bar-row">
								<span class="bar-name" title={c.name}>{c.name}</span>
								<div class="bar-track">
									<div
										class="bar-fill"
										style="width:{maxClientTotal > 0 ? Math.max(4, (c.total / maxClientTotal) * 100) : 0}%"
									></div>
								</div>
								<span class="bar-value">{fmtMoney(c.total)}</span>
							</div>
						{/each}
					</div>
				</section>
			{/if}

			<!-- Records -->
			<section class="panel">
				<div class="panel-head">
					<h2>Revenue records</h2>
					<div class="seg">
						{#each FILTERS as f}
							<button class:active={filter === f.id} onclick={() => (filter = f.id)}>{f.label}</button>
						{/each}
					</div>
				</div>

				{#if filteredDeals.length === 0}
					<div class="mini-empty">
						No {filter === 'all' ? '' : statusLabel(filter as FinanceDealStatus).toLowerCase()} records.
					</div>
				{:else}
					<div class="table">
						<div class="thead">
							<span class="c-name">Deal</span>
							<span class="c-client">Client</span>
							<span class="c-stage">Stage</span>
							<span class="c-status">Status</span>
							<span class="c-date">Close date</span>
							<span class="c-amt">Amount</span>
						</div>
						{#each filteredDeals as d (d.id)}
							<div class="trow">
								<span class="c-name" title={d.name}>{d.name}</span>
								<span class="c-client">{d.company_name?.trim() || '—'}</span>
								<span class="c-stage">{d.stage_name || '—'}</span>
								<span class="c-status">
									<span class="pill pill--{d.status ?? 'open'}">{statusLabel(d.status)}</span>
								</span>
								<span class="c-date">{fmtDate(d.actual_close_date || d.expected_close_date)}</span>
								<span class="c-amt">{fmtMoney(d.amount)}</span>
							</div>
						{/each}
					</div>
				{/if}
			</section>
		</div>
	{/if}
</div>

<style>
	.fin-root { height: 100%; display: flex; flex-direction: column; background: var(--dbg); color: var(--dt); overflow: hidden; }
	.topbar { display: flex; align-items: center; justify-content: space-between; gap: 16px; padding: 18px 24px; border-bottom: 1px solid var(--dbd); flex-shrink: 0; }
	.title-wrap { display: flex; align-items: center; gap: 12px; }
	.page-icon { width: 38px; height: 38px; border-radius: 10px; border: 1px solid var(--dbd); background: color-mix(in srgb, var(--dt) 5%, transparent); color: var(--dt2); display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
	.title-wrap h1 { font-size: 1.15rem; font-weight: 680; letter-spacing: -0.02em; margin: 0; }
	.sub { font-size: 0.8rem; color: var(--dt3); margin: 2px 0 0; }

	.btn { display: inline-flex; align-items: center; gap: 7px; padding: 8px 15px; border-radius: 9px; font-size: 0.83rem; font-weight: 580; cursor: pointer; border: 1px solid transparent; white-space: nowrap; text-decoration: none; }
	.btn--primary { background: var(--dt); color: var(--dbg); }
	.btn--ghost { background: transparent; border-color: var(--dbd); color: var(--dt2); }
	.btn:disabled { opacity: 0.55; cursor: not-allowed; }

	.banner { margin: 16px 24px 0; padding: 11px 14px; border-radius: 10px; font-size: 0.83rem; }
	.banner--error { background: color-mix(in srgb, #ef4444 12%, transparent); color: #ef4444; }

	.loading, .empty { flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 12px; color: var(--dt3); text-align: center; padding: 32px; }
	.loading { flex-direction: row; gap: 8px; }
	.empty-title { font-size: 0.98rem; font-weight: 600; color: var(--dt2); margin: 2px 0 0; }
	.empty-body { font-size: 0.85rem; color: var(--dt3); max-width: 380px; margin: 0 0 6px; line-height: 1.5; }

	.scroll { flex: 1; overflow-y: auto; padding: 20px 24px 28px; display: flex; flex-direction: column; gap: 20px; }

	.tiles { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 12px; }
	.tile { border: 1px solid var(--dbd); border-radius: 13px; padding: 15px 16px; background: color-mix(in srgb, var(--dt) 2%, transparent); display: flex; flex-direction: column; gap: 7px; }
	.tile--accent { border-color: color-mix(in srgb, var(--dt) 22%, transparent); background: color-mix(in srgb, var(--dt) 5%, transparent); }
	.tile-top { display: flex; align-items: center; gap: 7px; font-size: 0.76rem; font-weight: 600; color: var(--dt3); text-transform: uppercase; letter-spacing: 0.04em; }
	.tile-value { font-size: 1.45rem; font-weight: 680; letter-spacing: -0.02em; color: var(--dt); }
	.tile-meta { font-size: 0.76rem; color: var(--dt3); }

	.panel { border: 1px solid var(--dbd); border-radius: 14px; background: color-mix(in srgb, var(--dt) 1.5%, transparent); padding: 16px 18px; }
	.panel-head { display: flex; align-items: center; justify-content: space-between; gap: 12px; margin-bottom: 14px; }
	.panel-head h2 { font-size: 0.9rem; font-weight: 640; margin: 0; }
	.panel-note { font-size: 0.72rem; color: var(--dt3); }

	.bars { display: flex; flex-direction: column; gap: 10px; }
	.bar-row { display: grid; grid-template-columns: minmax(90px, 1.4fr) 3fr auto; align-items: center; gap: 12px; }
	.bar-name { font-size: 0.82rem; color: var(--dt2); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
	.bar-track { height: 8px; border-radius: 999px; background: color-mix(in srgb, var(--dt) 7%, transparent); overflow: hidden; }
	.bar-fill { height: 100%; border-radius: 999px; background: color-mix(in srgb, var(--dt) 55%, transparent); }
	.bar-value { font-size: 0.82rem; font-weight: 580; color: var(--dt); font-variant-numeric: tabular-nums; }

	.seg { display: flex; border: 1px solid var(--dbd); border-radius: 9px; overflow: hidden; }
	.seg button { padding: 6px 12px; font-size: 0.76rem; font-weight: 560; background: transparent; border: none; color: var(--dt3); cursor: pointer; }
	.seg button.active { background: color-mix(in srgb, var(--dt) 10%, transparent); color: var(--dt); }

	.table { display: flex; flex-direction: column; }
	.thead, .trow { display: grid; grid-template-columns: 2.2fr 1.5fr 1.2fr 0.9fr 1fr 1fr; align-items: center; gap: 12px; }
	.thead { padding: 8px 10px; font-size: 0.68rem; text-transform: uppercase; letter-spacing: 0.06em; color: var(--dt3); border-bottom: 1px solid var(--dbd); }
	.trow { padding: 11px 10px; border-bottom: 1px solid color-mix(in srgb, var(--dt) 6%, transparent); font-size: 0.84rem; }
	.trow:last-child { border-bottom: none; }
	.c-name { font-weight: 560; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
	.c-client, .c-stage, .c-date { color: var(--dt2); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
	.c-amt { text-align: right; font-weight: 580; font-variant-numeric: tabular-nums; }
	.pill { display: inline-flex; font-size: 0.68rem; font-weight: 620; padding: 2px 9px; border-radius: 999px; letter-spacing: 0.02em; }
	.pill--won { background: rgba(34, 197, 94, 0.15); color: #16a34a; }
	:global(.dark) .pill--won { color: #4ade80; }
	.pill--open { background: color-mix(in srgb, var(--dt) 10%, transparent); color: var(--dt2); }
	.pill--lost { background: color-mix(in srgb, #ef4444 14%, transparent); color: #ef4444; }

	.mini-empty { padding: 22px 6px; text-align: center; color: var(--dt3); font-size: 0.83rem; }

	:global(.spin) { animation: spin 0.9s linear infinite; }
	@keyframes spin { to { transform: rotate(360deg); } }

	@media (max-width: 768px) {
		.topbar { padding: 14px 16px; }
		.scroll { padding: 16px; gap: 16px; }
		.thead { display: none; }
		.trow { grid-template-columns: 1fr auto; gap: 6px 10px; padding: 12px 8px; }
		.c-name { grid-column: 1; }
		.c-amt { grid-column: 2; grid-row: 1; }
		.c-client { grid-column: 1; font-size: 0.76rem; }
		.c-status { grid-column: 2; grid-row: 2; justify-self: end; }
		.c-stage, .c-date { display: none; }
	}

	@media (max-width: 480px) {
		.topbar { flex-wrap: wrap; gap: 10px; }
		.bar-row { grid-template-columns: minmax(70px, 1fr) 2fr auto; gap: 8px; }
	}
</style>
