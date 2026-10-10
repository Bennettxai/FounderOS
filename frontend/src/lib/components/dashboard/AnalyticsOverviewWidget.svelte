<script lang="ts">
	import { dashboardAnalyticsStore } from '$lib/stores/dashboard/dashboardAnalyticsStore.svelte';

	interface Props {
		isLoading?: boolean;
	}

	let { isLoading = false }: Props = $props();

	const analytics = dashboardAnalyticsStore;

	// Compute trend direction from weekly activity data
	function weeklyTrend(data: number[]): { pct: number; up: boolean } {
		const recent = data.slice(-2);
		if (recent.length < 2 || recent[0] === 0) return { pct: 0, up: true };
		const pct = Math.round(((recent[1] - recent[0]) / recent[0]) * 100);
		return { pct: Math.abs(pct), up: pct >= 0 };
	}

	// Derived metrics from the seeded analytics store
	let metrics = $derived.by(() => {
		const a = analytics.seededAnalytics;
		if (!a) return null;
		const tasksTrend = weeklyTrend(a.tasks.weeklyData);
		return [
			{
				key: 'tasks',
				label: 'Tasks Done',
				value: a.tasks.completedThisWeek,
				suffix: '',
				trend: tasksTrend,
			},
			{
				key: 'focus',
				label: 'Focus Rate',
				value: a.focus.completionRate,
				suffix: '%',
				trend: { pct: 12, up: true },
			},
			{
				key: 'projects',
				label: 'Active Projects',
				value: a.projects.active,
				suffix: '',
				trend: { pct: 0, up: true },
			},
			{
				key: 'streak',
				label: 'Day Streak',
				value: a.focus.streak,
				suffix: 'd',
				trend: { pct: 0, up: true },
			},
		];
	});

	// Sparkline from tasks weekly data — normalised to 40px tall
	let sparkline = $derived.by(() => {
		const a = analytics.seededAnalytics;
		if (!a) return [];
		const data = a.tasks.weeklyData;
		const max = Math.max(...data, 1);
		return data.map((v, i) => ({
			x: i,
			h: Math.max(4, Math.round((v / max) * 32)),
			isToday: i === new Date().getDay() - 1 || (i === 6 && new Date().getDay() === 0),
		}));
	});
</script>

<div class="dw-aow-card">
	<!-- Header -->
	<div class="dw-aow-header">
		<div class="dw-aow-title-group">
			<div class="dw-aow-icon" aria-hidden="true">
				<svg width="14" height="14" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
						d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
				</svg>
			</div>
			<h2 class="dw-aow-title">Analytics</h2>
		</div>
		<span class="dw-aow-range">This Week</span>
	</div>

	{#if isLoading || analytics.analyticsLoading}
		<!-- Skeleton -->
		<div class="dw-aow-skeleton-wrap" aria-hidden="true">
			<div class="dw-aow-metrics-row">
				{#each [1, 2, 3, 4] as _}
					<div class="dw-aow-metric-cell">
						<div class="dw-aow-sk dw-aow-sk--value"></div>
						<div class="dw-aow-sk dw-aow-sk--label"></div>
						<div class="dw-aow-sk dw-aow-sk--trend"></div>
					</div>
				{/each}
			</div>
			<div class="dw-aow-sparkline-wrap">
				<div class="dw-aow-sk dw-aow-sk--spark"></div>
			</div>
		</div>
	{:else if metrics}
		<!-- Metrics row -->
		<div class="dw-aow-metrics-row" role="list">
			{#each metrics as metric (metric.key)}
				<div class="dw-aow-metric-cell" role="listitem">
					<div class="dw-aow-metric-value">
						{metric.value}<span class="dw-aow-metric-suffix">{metric.suffix}</span>
					</div>
					<div class="dw-aow-metric-label">{metric.label}</div>
					{#if metric.trend.pct > 0}
						<div class="dw-aow-trend {metric.trend.up ? 'dw-aow-trend--up' : 'dw-aow-trend--down'}">
							<svg width="10" height="10" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
								{#if metric.trend.up}
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M5 15l7-7 7 7" />
								{:else}
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M19 9l-7 7-7-7" />
								{/if}
							</svg>
							{metric.trend.pct}%
						</div>
					{:else}
						<div class="dw-aow-trend-neutral">—</div>
					{/if}
				</div>
			{/each}
		</div>

		<!-- Sparkline -->
		<div class="dw-aow-sparkline-wrap" aria-label="7-day task activity">
			<div class="dw-aow-sparkline" role="img">
				{#each sparkline as bar (bar.x)}
					<div class="dw-aow-spark-col">
						<div
							class="dw-aow-spark-bar {bar.isToday ? 'dw-aow-spark-bar--today' : ''}"
							style="height: {bar.h}px"
						></div>
					</div>
				{/each}
			</div>
			<div class="dw-aow-day-labels" aria-hidden="true">
				<span>M</span><span>T</span><span>W</span><span>T</span><span>F</span><span>S</span><span>S</span>
			</div>
		</div>
	{:else}
		<!-- Empty -->
		<div class="dw-aow-empty">
			<p class="dw-aow-empty-text">No data yet</p>
		</div>
	{/if}
</div>

<style>
	/* ── Analytics Overview Widget (dw-aow-*) — Foundation tokens ── */

	.dw-aow-card {
		background: var(--dbg);
		border: 1px solid var(--dbd);
		border-radius: 12px;
		padding: var(--space-4);
		display: flex;
		flex-direction: column;
		gap: var(--space-3);
		box-shadow: var(--shadow-sm);
	}

	/* ── Header ── */
	.dw-aow-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
	}

	.dw-aow-title-group {
		display: flex;
		align-items: center;
		gap: var(--space-2);
	}

	.dw-aow-icon {
		width: 2rem;
		height: 2rem;
		border-radius: 8px;
		background: var(--dbg3);
		border: 1px solid var(--dbd);
		display: flex;
		align-items: center;
		justify-content: center;
		color: var(--dt2);
		flex-shrink: 0;
	}

	.dw-aow-title {
		font-size: 0.9rem;
		font-weight: 600;
		color: var(--dt);
		margin: 0;
	}

	.dw-aow-range {
		font-size: 0.72rem;
		color: var(--dt4);
		font-weight: 500;
	}

	/* ── Metrics row ── */
	.dw-aow-metrics-row {
		display: grid;
		grid-template-columns: repeat(4, 1fr);
		gap: var(--space-2);
	}

	.dw-aow-metric-cell {
		display: flex;
		flex-direction: column;
		gap: 2px;
		padding: var(--space-2) var(--space-2);
		background: var(--dbg2);
		border: 1px solid var(--dbd);
		border-radius: 8px;
		min-width: 0;
	}

	.dw-aow-metric-value {
		font-size: 1.35rem;
		font-weight: 700;
		color: var(--dt);
		line-height: 1;
		font-variant-numeric: tabular-nums;
	}

	.dw-aow-metric-suffix {
		font-size: 0.8rem;
		font-weight: 500;
		color: var(--dt3);
	}

	.dw-aow-metric-label {
		font-size: 0.7rem;
		color: var(--dt3);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.dw-aow-trend {
		display: inline-flex;
		align-items: center;
		gap: 2px;
		font-size: 0.65rem;
		font-weight: 600;
		margin-top: 2px;
	}

	.dw-aow-trend--up {
		color: #22c55e;
	}

	.dw-aow-trend--down {
		color: #ef4444;
	}

	.dw-aow-trend-neutral {
		font-size: 0.65rem;
		color: var(--dt4);
		margin-top: 2px;
	}

	/* ── Sparkline ── */
	.dw-aow-sparkline-wrap {
		display: flex;
		flex-direction: column;
		gap: 4px;
	}

	.dw-aow-sparkline {
		display: flex;
		align-items: flex-end;
		gap: 3px;
		height: 40px;
	}

	.dw-aow-spark-col {
		flex: 1;
		display: flex;
		align-items: flex-end;
		height: 100%;
	}

	.dw-aow-spark-bar {
		width: 100%;
		border-radius: 3px 3px 0 0;
		background: var(--dt3);
		min-height: 4px;
		transition: height 0.4s ease;
	}

	.dw-aow-spark-bar--today {
		background: var(--dt);
	}

	.dw-aow-day-labels {
		display: flex;
		justify-content: space-between;
		font-size: 0.62rem;
		color: var(--dt4);
		font-weight: 500;
		padding: 0 1px;
	}

	/* ── Skeleton ── */
	.dw-aow-skeleton-wrap {
		display: flex;
		flex-direction: column;
		gap: var(--space-3);
	}

	@keyframes dw-aow-pulse {
		50% { opacity: 0.5; }
	}

	.dw-aow-sk {
		background: var(--dbg3, color-mix(in srgb, var(--dt) 8%, transparent));
		border-radius: 4px;
		animation: dw-aow-pulse 1.5s ease-in-out infinite;
	}

	.dw-aow-sk--value {
		width: 48px;
		height: 22px;
		border-radius: 4px;
	}

	.dw-aow-sk--label {
		width: 56px;
		height: 10px;
		border-radius: 3px;
	}

	.dw-aow-sk--trend {
		width: 28px;
		height: 8px;
		border-radius: 3px;
	}

	.dw-aow-sk--spark {
		width: 100%;
		height: 40px;
		border-radius: 4px;
	}

	/* ── Empty ── */
	.dw-aow-empty {
		display: flex;
		align-items: center;
		justify-content: center;
		padding: var(--space-6) 0;
	}

	.dw-aow-empty-text {
		font-size: 0.83rem;
		color: var(--dt4);
	}
</style>
