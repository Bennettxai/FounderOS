<script lang="ts">
	import { onMount } from 'svelte';
	import { currentWorkspace } from '$lib/stores/workspaces';
	import { getAnalyticsSummary, type AnalyticsSummary, type StatCount } from '$lib/api/analytics';
	import {
		BarChart3,
		Loader2,
		RefreshCw,
		CheckSquare,
		FolderKanban,
		Users,
		Package,
		FileText,
		Megaphone,
		TrendingUp,
		AlertTriangle
	} from 'lucide-svelte';

	let summary = $state<AnalyticsSummary | null>(null);
	let loading = $state(true);
	let error = $state<string | null>(null);

	let wsId = $state<string | null | undefined>(undefined);
	$effect(() => {
		const id = $currentWorkspace?.id ?? null;
		if (id !== wsId) {
			wsId = id;
			load();
		}
	});
	onMount(load);

	async function load() {
		loading = true;
		error = null;
		try {
			summary = await getAnalyticsSummary();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load analytics';
		} finally {
			loading = false;
		}
	}

	const tiles = $derived(
		summary
			? [
					{ label: 'Tasks', value: summary.totals.tasks, icon: CheckSquare },
					{ label: 'Projects', value: summary.totals.projects, icon: FolderKanban },
					{ label: 'Clients', value: summary.totals.clients, icon: Users },
					{ label: 'Offers', value: summary.totals.offers, icon: Package },
					{ label: 'Content', value: summary.totals.content, icon: FileText },
					{ label: 'Campaigns', value: summary.totals.campaigns, icon: Megaphone }
				]
			: []
	);

	// Bar chart helpers — scale each bucket against the max in its group.
	function maxOf(buckets: StatCount[]): number {
		return buckets.reduce((m, b) => Math.max(m, b.count), 0);
	}
	function pct(count: number, max: number): number {
		if (max <= 0) return 0;
		return Math.max(2, Math.round((count / max) * 100));
	}
	function prettyLabel(s: string): string {
		return s
			.replace(/_/g, ' ')
			.toLowerCase()
			.replace(/\b\w/g, (c) => c.toUpperCase());
	}

	const anyData = $derived(!!summary && tiles.some((t) => t.value > 0));
</script>

<svelte:head><title>Analytics - BusinessOS</title></svelte:head>

<div class="page">
	<header class="page-header">
		<div class="page-icon"><BarChart3 size={22} strokeWidth={1.8} /></div>
		<div class="head-text">
			<h1 class="page-title">Analytics</h1>
			<p class="page-desc">Counts and trends computed across your workspace activity.</p>
		</div>
		<button class="btn btn-ghost" onclick={load} disabled={loading}>
			{#if loading}<Loader2 size={15} class="spin" />{:else}<RefreshCw size={15} />{/if} Refresh
		</button>
	</header>

	{#if error}<div class="error-bar">{error}</div>{/if}

	{#if loading}
		<div class="loading"><Loader2 size={22} class="spin" /> Crunching your workspace…</div>
	{:else if summary}
		<!-- Stat tiles -->
		<div class="tiles">
			{#each tiles as t}
				{@const Ico = t.icon}
				<div class="tile">
					<div class="tile-top"><Ico size={16} /><span class="tile-label">{t.label}</span></div>
					<span class="tile-value">{t.value.toLocaleString()}</span>
				</div>
			{/each}
		</div>

		<!-- Trends row -->
		<section>
			<h2 class="sec-title"><TrendingUp size={16} /> Last 30 days</h2>
			<div class="trends">
				<div class="trend">
					<span class="trend-value">{summary.trends.tasks_completed_30d}</span>
					<span class="trend-label">Tasks completed</span>
				</div>
				<div class="trend">
					<span class="trend-value">{summary.trends.tasks_created_30d}</span>
					<span class="trend-label">Tasks created</span>
				</div>
				<div class="trend">
					<span class="trend-value">{summary.trends.tasks_open}</span>
					<span class="trend-label">Tasks open now</span>
				</div>
				<div class="trend {summary.trends.tasks_overdue > 0 ? 'trend-warn' : ''}">
					<span class="trend-value">
						{#if summary.trends.tasks_overdue > 0}<AlertTriangle size={14} />{/if}
						{summary.trends.tasks_overdue}
					</span>
					<span class="trend-label">Tasks overdue</span>
				</div>
				<div class="trend">
					<span class="trend-value">{summary.trends.projects_active}</span>
					<span class="trend-label">Active projects</span>
				</div>
				<div class="trend">
					<span class="trend-value">{summary.trends.clients_active}</span>
					<span class="trend-label">Active clients</span>
				</div>
			</div>
		</section>

		<!-- Bar charts -->
		<div class="charts">
			{#each [{ title: 'Tasks by status', icon: CheckSquare, data: summary.tasks_by_status }, { title: 'Projects by status', icon: FolderKanban, data: summary.projects_by_status }, { title: 'Clients by status', icon: Users, data: summary.clients_by_status }] as chart}
				{@const Ico = chart.icon}
				{@const mx = maxOf(chart.data)}
				<section class="chart-card">
					<h2 class="sec-title"><Ico size={16} /> {chart.title}</h2>
					{#if chart.data.length}
						<div class="bars">
							{#each chart.data as b (b.label)}
								<div class="bar-row">
									<span class="bar-label" title={prettyLabel(b.label)}>{prettyLabel(b.label)}</span>
									<div class="bar-track">
										<div class="bar-fill" style="width: {pct(b.count, mx)}%"></div>
									</div>
									<span class="bar-count">{b.count}</span>
								</div>
							{/each}
						</div>
					{:else}
						<p class="mini-empty">No records yet.</p>
					{/if}
				</section>
			{/each}
		</div>

		{#if !anyData}
			<div class="empty-state">
				<BarChart3 size={40} strokeWidth={1.4} class="empty-icon" />
				<p class="empty-title">No data to analyze yet</p>
				<p class="empty-body">Analytics populate as activity accumulates across projects, tasks, and campaigns.</p>
			</div>
		{/if}
	{/if}
</div>

<style>
	.page { display: flex; flex-direction: column; gap: 22px; padding: 28px 32px; height: 100%; overflow-y: auto; }
	.page-header { display: flex; align-items: flex-start; gap: 14px; }
	.page-icon { width: 42px; height: 42px; border-radius: 10px; border: 1px solid var(--dbd); background: color-mix(in srgb, var(--dt) 5%, transparent); color: var(--dt2); display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
	.head-text { flex: 1; min-width: 0; }
	.page-title { font-size: 1.25rem; font-weight: 650; color: var(--dt); letter-spacing: -0.02em; margin: 0; }
	.page-desc { font-size: 0.875rem; color: var(--dt3); margin: 2px 0 0; }

	.btn { display: inline-flex; align-items: center; gap: 6px; padding: 8px 14px; border-radius: 8px; font-size: 0.84rem; font-weight: 550; cursor: pointer; border: 1px solid transparent; white-space: nowrap; }
	.btn-ghost { background: transparent; border-color: var(--dbd); color: var(--dt2); }
	.btn-ghost:hover { background: color-mix(in srgb, var(--dt) 5%, transparent); }
	.btn:disabled { opacity: 0.6; cursor: default; }

	.error-bar { padding: 10px 14px; border-radius: 8px; background: color-mix(in srgb, #ef4444 12%, transparent); color: #ef4444; font-size: 0.83rem; }
	.loading { display: flex; align-items: center; gap: 8px; padding: 48px; justify-content: center; color: var(--dt3); font-size: 0.9rem; }

	.tiles { display: grid; grid-template-columns: repeat(auto-fill, minmax(150px, 1fr)); gap: 12px; }
	.tile { border: 1px solid var(--dbd); border-radius: 12px; padding: 14px 16px; display: flex; flex-direction: column; gap: 8px; background: color-mix(in srgb, var(--dt) 2%, transparent); }
	.tile-top { display: flex; align-items: center; gap: 7px; color: var(--dt3); }
	.tile-label { font-size: 0.78rem; font-weight: 550; }
	.tile-value { font-size: 1.5rem; font-weight: 680; color: var(--dt); letter-spacing: -0.02em; }

	section { display: flex; flex-direction: column; gap: 12px; }
	.sec-title { display: flex; align-items: center; gap: 7px; font-size: 0.82rem; font-weight: 650; text-transform: uppercase; letter-spacing: 0.05em; color: var(--dt2); margin: 0; }

	.trends { display: grid; grid-template-columns: repeat(auto-fill, minmax(150px, 1fr)); gap: 12px; }
	.trend { border: 1px solid var(--dbd); border-radius: 10px; padding: 12px 14px; display: flex; flex-direction: column; gap: 4px; background: color-mix(in srgb, var(--dt) 2%, transparent); }
	.trend-value { display: flex; align-items: center; gap: 5px; font-size: 1.2rem; font-weight: 650; color: var(--dt); }
	.trend-label { font-size: 0.76rem; color: var(--dt3); }
	.trend-warn { border-left: 3px solid #f59e0b; }
	.trend-warn .trend-value { color: #f59e0b; }

	.charts { display: grid; grid-template-columns: repeat(auto-fill, minmax(300px, 1fr)); gap: 16px; }
	.chart-card { border: 1px solid var(--dbd); border-radius: 12px; padding: 16px 18px; background: color-mix(in srgb, var(--dt) 2%, transparent); display: flex; flex-direction: column; gap: 12px; }
	.bars { display: flex; flex-direction: column; gap: 10px; }
	.bar-row { display: grid; grid-template-columns: 90px 1fr 32px; align-items: center; gap: 10px; }
	.bar-label { font-size: 0.78rem; color: var(--dt2); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
	.bar-track { height: 10px; border-radius: 5px; background: color-mix(in srgb, var(--dt) 7%, transparent); overflow: hidden; }
	.bar-fill { height: 100%; border-radius: 5px; background: var(--dt2); transition: width 0.3s ease; }
	.bar-count { font-size: 0.8rem; color: var(--dt3); text-align: right; font-variant-numeric: tabular-nums; }
	.mini-empty { padding: 12px 0; color: var(--dt3); font-size: 0.83rem; margin: 0; }

	.empty-state { flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 10px; padding: 48px 24px; text-align: center; }
	.empty-state :global(.empty-icon) { color: var(--dt4, var(--dt3)); opacity: 0.5; }
	.empty-title { font-size: 0.95rem; font-weight: 600; color: var(--dt2); margin: 4px 0 0; }
	.empty-body { font-size: 0.84rem; color: var(--dt3); max-width: 360px; margin: 0; }

	:global(.spin) { animation: spin 0.8s linear infinite; }
	@keyframes spin { to { transform: rotate(360deg); } }
	@media (max-width: 768px) { .page { padding: 16px 18px; } .charts { grid-template-columns: 1fr; } }
	@media (max-width: 480px) { .page { padding: 12px 14px; } }
</style>
