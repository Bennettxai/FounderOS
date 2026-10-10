<script lang="ts">
	import { signalStore, genreDistribution, modeDistribution } from '$lib/stores/signal';
	import {
		getSignalHealth,
		type SignalHealthResponse,
		GENRE_LABELS,
		MODE_LABELS
	} from '$lib/api/signal';
	import { onMount } from 'svelte';

	let health = $state<SignalHealthResponse | null>(null);
	let loading = $state(true);
	let error = $state<string | null>(null);

	const genres = $derived($genreDistribution);
	const modes = $derived($modeDistribution);
	const totalEvents = $derived($signalStore.history.length);

	onMount(async () => {
		try {
			health = await getSignalHealth();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load signal health';
		} finally {
			loading = false;
		}
	});

	const statusVariant = $derived((): 'healthy' | 'degraded' | 'unknown' => {
		if (!health) return 'unknown';
		if (health.status === 'healthy') return 'healthy';
		if (health.status === 'degraded') return 'degraded';
		return 'unknown';
	});

	const passingMetrics = $derived((): number | null => {
		if (!health) return null;
		const m = health.metrics;
		return [
			m.action_completion,
			m.re_encoding,
			m.signal_bounce,
			m.genre_recognition,
			m.feedback_closure,
			m.time_to_decide
		].filter(Boolean).length;
	});
</script>

<div class="dw-sh-widget">
	<!-- Header -->
	<div class="dw-sh-header">
		<div class="dw-sh-header-left">
			<div class="dw-sh-icon-wrap" aria-hidden="true">
				<svg class="dw-sh-icon-svg" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
				</svg>
			</div>
			<h2 class="dw-sh-title">Signal Health</h2>
		</div>

		<!-- Status badge -->
		{#if loading}
			<span class="dw-sh-status-badge dw-sh-status-badge--unknown">
				<span class="dw-sh-dot"></span>
				Loading
			</span>
		{:else if health}
			<span class="dw-sh-status-badge dw-sh-status-badge--{statusVariant()}">
				<span class="dw-sh-dot"></span>
				{health.status}
			</span>
		{/if}
	</div>

	<!-- Error state -->
	{#if error}
		<div class="dw-sh-error">
			<svg class="dw-sh-error-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L3.07 16.5c-.77.833.192 2.5 1.732 2.5z" />
			</svg>
			<p class="dw-sh-error-text">{error}</p>
		</div>

	<!-- Loading skeleton -->
	{:else if loading}
		<div class="dw-sh-skeleton-group" aria-busy="true" aria-label="Loading signal health data">
			<div class="dw-sh-metric-grid">
				{#each [0, 1, 2] as _}
					<div class="dw-sh-skeleton dw-sh-skeleton--card"></div>
				{/each}
			</div>
			<div class="dw-sh-skeleton dw-sh-skeleton--line dw-sh-skeleton--w75"></div>
			<div class="dw-sh-skeleton dw-sh-skeleton--line"></div>
			<div class="dw-sh-skeleton dw-sh-skeleton--line dw-sh-skeleton--w83"></div>
		</div>

	{:else}
		<!-- Summary metric cards -->
		{#if health}
			<div class="dw-sh-metric-grid">
				<div class="dw-metric-card dw-metric-card--center">
					<span class="dw-metric-card__value">{passingMetrics()}/6</span>
					<span class="dw-metric-card__label">Metrics</span>
				</div>
				<div class="dw-metric-card dw-metric-card--center">
					<span
						class="dw-metric-card__value"
						class:dw-metric-card__value--on={health.feedback_loop.homeostatic_loop}
						class:dw-metric-card__value--off={!health.feedback_loop.homeostatic_loop}
					>
						{health.feedback_loop.homeostatic_loop ? 'ON' : 'OFF'}
					</span>
					<span class="dw-metric-card__label">Feedback</span>
				</div>
				<div class="dw-metric-card dw-metric-card--center">
					<span class="dw-metric-card__value">{totalEvents}</span>
					<span class="dw-metric-card__label">Signals</span>
				</div>
			</div>
		{/if}

		<!-- Genre distribution -->
		{#if genres.length > 0}
			<div class="dw-sh-section">
				<p class="dw-sh-section-label">Genre Distribution</p>
				<div class="dw-sh-bar-rows">
					{#each genres as g}
						<div class="dw-sh-bar-row">
							<span class="dw-sh-bar-name">{GENRE_LABELS[g.genre] ?? g.genre}</span>
							<div
								class="dw-sh-bar-track"
								role="meter"
								aria-label="{GENRE_LABELS[g.genre] ?? g.genre}: {g.percentage}%"
								aria-valuenow={g.percentage}
								aria-valuemin={0}
								aria-valuemax={100}
							>
								<div
									class="dw-sh-bar-fill dw-sh-bar-fill--genre-{g.genre.toLowerCase()}"
									style="width: {g.percentage}%"
								></div>
							</div>
							<span class="dw-sh-bar-pct">{g.percentage}%</span>
						</div>
					{/each}
				</div>
			</div>
		{/if}

		<!-- Mode distribution stacked bar -->
		{#if modes.length > 0}
			<div class="dw-sh-section">
				<p class="dw-sh-section-label">Mode Distribution</p>
				<div
					class="dw-sh-stacked-bar"
					role="group"
					aria-label="Mode distribution"
				>
					{#each modes as m}
						<div
							class="dw-sh-stacked-segment dw-sh-stacked-segment--mode-{m.mode.toLowerCase()}"
							style="width: {m.percentage}%"
							title="{MODE_LABELS[m.mode] ?? m.mode}: {m.percentage}%"
						></div>
					{/each}
				</div>
				<div class="dw-sh-legend">
					{#each modes as m}
						<div class="dw-sh-legend-item">
							<span class="dw-sh-legend-dot dw-sh-legend-dot--mode-{m.mode.toLowerCase()}"></span>
							<span class="dw-sh-legend-label">{MODE_LABELS[m.mode] ?? m.mode}</span>
						</div>
					{/each}
				</div>
			</div>
		{/if}

		<!-- Empty state -->
		{#if genres.length === 0 && modes.length === 0}
			<div class="dw-sh-empty">
				<svg class="dw-sh-empty-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
				</svg>
				<p class="dw-sh-empty-title">No signal data yet</p>
				<a href="/chat" class="dw-sh-empty-action">Start a conversation</a>
			</div>
		{/if}
	{/if}
</div>

<style>
	/* ─── Signal color tokens — CSS vars with hex fallbacks ─── */
	:root {
		/* Genre colors */
		--dw-signal-direct:   var(--accent-orange, #fb923c);
		--dw-signal-inform:   var(--accent-blue, #60a5fa);
		--dw-signal-commit:   var(--color-success, #34d399);
		--dw-signal-decide:   var(--color-error, #fb7185);
		--dw-signal-express:  var(--accent-pink, #f472b6);

		/* Mode colors */
		--dw-signal-build:    var(--accent-indigo, #6366f1);
		--dw-signal-assist:   var(--color-success, #22c55e);
		--dw-signal-analyze:  var(--accent-purple, #8b5cf6);
		--dw-signal-execute:  var(--accent-orange, #f59e0b);
		--dw-signal-maintain: var(--dt3, #94a3b8);
	}

	/* ─── Widget shell ─── */
	.dw-sh-widget {
		background: var(--dbg);
		border: 1px solid var(--dbd);
		border-radius: 0.75rem;
		padding: 1.25rem;
		box-shadow: var(--shadow-sm, 0 1px 2px rgba(0, 0, 0, 0.05));
		transition: box-shadow 0.3s;
		display: flex;
		flex-direction: column;
		gap: 1rem;
	}

	.dw-sh-widget:hover {
		box-shadow: var(--shadow-md, 0 4px 6px rgba(0, 0, 0, 0.07));
	}

	/* ─── Header ─── */
	.dw-sh-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
	}

	.dw-sh-header-left {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}

	.dw-sh-icon-wrap {
		width: 2rem;
		height: 2rem;
		border-radius: 0.5rem;
		background: linear-gradient(135deg, var(--accent-purple, #7c3aed), var(--accent-indigo, #4338ca));
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.12);
	}

	.dw-sh-icon-svg {
		width: 1rem;
		height: 1rem;
		color: #fff;
	}

	.dw-sh-title {
		font-size: 1rem;
		font-weight: 600;
		color: var(--dt);
		line-height: 1.25;
	}

	/* ─── Status badge ─── */
	.dw-sh-status-badge {
		display: inline-flex;
		align-items: center;
		gap: 0.375rem;
		font-size: 0.72rem;
		font-weight: 500;
		padding: 0.2rem 0.6rem;
		border-radius: 999px;
		border: 1px solid var(--dbd2);
		background: var(--dbg2);
		text-transform: capitalize;
	}

	.dw-sh-dot {
		width: 0.5rem;
		height: 0.5rem;
		border-radius: 50%;
		flex-shrink: 0;
	}

	.dw-sh-status-badge--healthy {
		color: var(--color-success, #16a34a);
	}
	.dw-sh-status-badge--healthy .dw-sh-dot {
		background: var(--color-success, #22c55e);
	}

	.dw-sh-status-badge--degraded {
		color: var(--accent-orange, #d97706);
	}
	.dw-sh-status-badge--degraded .dw-sh-dot {
		background: var(--accent-orange, #f59e0b);
	}

	.dw-sh-status-badge--unknown {
		color: var(--dt3);
	}
	.dw-sh-status-badge--unknown .dw-sh-dot {
		background: var(--dt4);
	}

	/* ─── Metric cards grid ─── */
	.dw-sh-metric-grid {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: var(--space-3, 0.75rem);
	}

	.dw-metric-card {
		background: var(--dbg2);
		padding: 0.5rem 0.25rem;
		display: flex;
		flex-direction: column;
		gap: 0.2rem;
	}

	.dw-metric-card--center {
		align-items: center;
		text-align: center;
	}

	.dw-metric-card__value {
		font-size: 1.1rem;
		font-weight: 700;
		color: var(--dt);
		line-height: 1.2;
	}

	.dw-metric-card__value--on {
		color: var(--color-success, #16a34a);
	}

	.dw-metric-card__value--off {
		color: var(--dt3);
	}

	.dw-metric-card__label {
		font-size: 0.65rem;
		font-weight: 500;
		text-transform: uppercase;
		letter-spacing: 0.06em;
		color: var(--dt3);
	}

	/* ─── Sections (genre + mode) ─── */
	.dw-sh-section {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}

	.dw-sh-section-label {
		font-size: 0.65rem;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.07em;
		color: var(--dt3);
	}

	/* ─── Genre bar rows ─── */
	.dw-sh-bar-rows {
		display: flex;
		flex-direction: column;
		gap: 0.4rem;
	}

	.dw-sh-bar-row {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}

	.dw-sh-bar-name {
		font-size: 0.67rem;
		color: var(--dt2);
		width: 4rem;
		text-align: right;
		flex-shrink: 0;
	}

	.dw-sh-bar-track {
		flex: 1;
		height: 0.5rem;
		border-radius: 999px;
		background: var(--dbg3);
		overflow: hidden;
	}

	.dw-sh-bar-fill {
		height: 100%;
		border-radius: 999px;
		transition: width 0.5s ease;
	}

	/* Genre fill colors via CSS custom properties */
	.dw-sh-bar-fill--genre-direct   { background: var(--dw-signal-direct); }
	.dw-sh-bar-fill--genre-inform   { background: var(--dw-signal-inform); }
	.dw-sh-bar-fill--genre-commit   { background: var(--dw-signal-commit); }
	.dw-sh-bar-fill--genre-decide   { background: var(--dw-signal-decide); }
	.dw-sh-bar-fill--genre-express  { background: var(--dw-signal-express); }

	.dw-sh-bar-pct {
		font-size: 0.67rem;
		color: var(--dt3);
		width: 2rem;
		text-align: right;
		flex-shrink: 0;
	}

	/* ─── Mode stacked bar ─── */
	.dw-sh-stacked-bar {
		display: flex;
		gap: 2px;
		height: 0.75rem;
		border-radius: 999px;
		overflow: hidden;
		background: var(--dbg3);
	}

	.dw-sh-stacked-segment {
		height: 100%;
		transition: width 0.5s ease;
	}

	/* Mode segment colors */
	.dw-sh-stacked-segment--mode-build    { background: var(--dw-signal-build); }
	.dw-sh-stacked-segment--mode-assist   { background: var(--dw-signal-assist); }
	.dw-sh-stacked-segment--mode-analyze  { background: var(--dw-signal-analyze); }
	.dw-sh-stacked-segment--mode-execute  { background: var(--dw-signal-execute); }
	.dw-sh-stacked-segment--mode-maintain { background: var(--dw-signal-maintain); }

	/* ─── Mode legend ─── */
	.dw-sh-legend {
		display: flex;
		flex-wrap: wrap;
		gap: 0.3rem 0.75rem;
	}

	.dw-sh-legend-item {
		display: flex;
		align-items: center;
		gap: 0.3rem;
	}

	.dw-sh-legend-dot {
		width: 0.5rem;
		height: 0.5rem;
		border-radius: 50%;
		flex-shrink: 0;
	}

	/* Mode dot colors */
	.dw-sh-legend-dot--mode-build    { background: var(--dw-signal-build); }
	.dw-sh-legend-dot--mode-assist   { background: var(--dw-signal-assist); }
	.dw-sh-legend-dot--mode-analyze  { background: var(--dw-signal-analyze); }
	.dw-sh-legend-dot--mode-execute  { background: var(--dw-signal-execute); }
	.dw-sh-legend-dot--mode-maintain { background: var(--dw-signal-maintain); }

	.dw-sh-legend-label {
		font-size: 0.67rem;
		color: var(--dt3);
	}

	/* ─── Skeleton loading ─── */
	.dw-sh-skeleton-group {
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
		animation: dw-sh-pulse 1.6s ease-in-out infinite;
	}

	@keyframes dw-sh-pulse {
		0%, 100% { opacity: 1; }
		50%       { opacity: 0.45; }
	}

	.dw-sh-skeleton {
		background: var(--dbg3);
		border-radius: 0.375rem;
	}

	.dw-sh-skeleton--card {
		height: 3rem;
	}

	.dw-sh-skeleton--line {
		height: 0.75rem;
		width: 100%;
	}

	.dw-sh-skeleton--w75 { width: 75%; }
	.dw-sh-skeleton--w83 { width: 83%; }

	/* ─── Error state ─── */
	.dw-sh-error {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		padding: 0.75rem;
		background: color-mix(in srgb, var(--color-error, #dc2626) 8%, transparent);
		border: 1px solid color-mix(in srgb, var(--color-error, #dc2626) 20%, transparent);
		border-radius: 0.5rem;
	}

	.dw-sh-error-icon {
		width: 1rem;
		height: 1rem;
		color: var(--color-error, #dc2626);
		flex-shrink: 0;
	}

	.dw-sh-error-text {
		font-size: 0.8rem;
		color: var(--color-error, #dc2626);
		line-height: 1.4;
	}

	/* ─── Empty state ─── */
	.dw-sh-empty {
		display: flex;
		flex-direction: column;
		align-items: center;
		padding: 2rem 1rem;
		gap: 0.75rem;
		text-align: center;
	}

	.dw-sh-empty-icon {
		width: 1.5rem;
		height: 1.5rem;
		color: var(--dt3);
		flex-shrink: 0;
	}

	.dw-sh-empty-title {
		font-size: 0.85rem;
		color: var(--dt2);
		margin: 0;
	}

	.dw-sh-empty-action {
		display: inline-flex;
		align-items: center;
		font-size: 0.8rem;
		color: var(--dt2);
		background: transparent;
		border: 1px solid var(--dbd2);
		border-radius: 6px;
		padding: 0.2rem 0.75rem;
		cursor: pointer;
		transition: border-color 0.15s, color 0.15s, background 0.15s;
		height: 28px;
		text-decoration: none;
	}

	.dw-sh-empty-action:hover {
		border-color: var(--dt4);
		color: var(--dt);
		background: var(--dbg2);
	}
</style>
