<script lang="ts">
	import { fade, fly, scale } from 'svelte/transition';
	import { tweened } from 'svelte/motion';
	import { cubicOut } from 'svelte/easing';

	import { dashboardLayoutStore, availableWidgets, uniqueWidgetTypes } from '$lib/stores/dashboard/dashboardLayoutStore.svelte';
	import { dashboardAnalyticsStore } from '$lib/stores/dashboard/dashboardAnalyticsStore.svelte';
	import { dashboardDataStore } from '$lib/stores/dashboard/dashboardDataStore.svelte';
	import type { WidgetSize, AnalyticsTimeRange } from '$lib/stores/dashboard/types';

	interface Props {
		isOpen?: boolean;
		onToggle?: () => void;
	}

	let { isOpen = true, onToggle }: Props = $props();

	type TabId = 'widgets' | 'analytics' | 'activity';
	let activeTab = $state<TabId>('analytics');

	// Size selector state for widget picker
	let pickerSize = $state<WidgetSize>('medium');

	const layout = dashboardLayoutStore;
	const analytics = dashboardAnalyticsStore;
	const data = dashboardDataStore;

	// When edit mode activates, switch to widgets tab
	$effect(() => {
		if (layout.isEditMode) {
			activeTab = 'widgets';
		}
	});

	// Sync picker size into the store when it changes locally
	$effect(() => {
		layout.pickerSelectedSize = pickerSize;
	});

	// ── Animated numbers ──────────────────────────────────────────────────────────
	const animFocusRate      = tweened(0, { duration: 600, easing: cubicOut });
	const animTasksDone      = tweened(0, { duration: 600, easing: cubicOut });
	const animActiveProjects = tweened(0, { duration: 600, easing: cubicOut });
	const animTotalActivity  = tweened(0, { duration: 600, easing: cubicOut });

	$effect(() => {
		const s = analytics.seededAnalytics;
		if (s && isOpen && activeTab === 'analytics') {
			animFocusRate.set(s.focus.completionRate);
			animTasksDone.set(s.tasks.completedThisWeek);
			animActiveProjects.set(s.projects.active);
			animTotalActivity.set(s.activity.totalActions);
		}
	});

	$effect(() => {
		if (!isOpen) {
			animFocusRate.set(0, { duration: 0 });
			animTasksDone.set(0, { duration: 0 });
			animActiveProjects.set(0, { duration: 0 });
			animTotalActivity.set(0, { duration: 0 });
		}
	});

	// ── Time range labels ─────────────────────────────────────────────────────────
	const timeRangeOptions: { value: AnalyticsTimeRange; label: string }[] = [
		{ value: 'today', label: 'Today' },
		{ value: 'week',  label: 'Week' },
		{ value: 'month', label: 'Month' },
		{ value: '30days', label: '30d' },
	];

	function handleTimeRange(range: AnalyticsTimeRange): void {
		analytics.handleAnalyticsTimeRangeChange(range);
	}

	// ── Widget helpers ────────────────────────────────────────────────────────────
	const widgetSizes: { value: WidgetSize; label: string }[] = [
		{ value: 'small',  label: 'S' },
		{ value: 'medium', label: 'M' },
		{ value: 'large',  label: 'L' },
	];

	function isWidgetAdded(type: string): boolean {
		return uniqueWidgetTypes.includes(type as any) &&
			layout.addedUniqueTypes.has(type as any);
	}

	// ── Activity helpers ──────────────────────────────────────────────────────────
	function formatActivityTime(createdAt: string): string {
		const date = new Date(createdAt);
		const now = new Date();
		const diffMs = now.getTime() - date.getTime();
		const diffMins = Math.floor(diffMs / 60_000);
		if (diffMins < 1) return 'just now';
		if (diffMins < 60) return `${diffMins}m ago`;
		const diffHrs = Math.floor(diffMins / 60);
		if (diffHrs < 24) return `${diffHrs}h ago`;
		const diffDays = Math.floor(diffHrs / 24);
		return `${diffDays}d ago`;
	}

	function getActivityDotColor(type: string): string {
		switch (type) {
			case 'task_completed':   return 'var(--dt)';
			case 'task_started':     return 'var(--dt2)';
			case 'project_created':  return 'var(--dt)';
			case 'project_updated':  return 'var(--dt3)';
			case 'conversation':     return 'var(--dt2)';
			case 'team':             return 'var(--dt3)';
			case 'artifact':         return 'var(--dt2)';
			default:                 return 'var(--dt3)';
		}
	}

</script>

<!-- Toggle pill button (always visible, sits on the left edge of the panel) -->
<button
	class="dw-rp-toggle-btn"
	onclick={onToggle}
	aria-label={isOpen ? 'Collapse right panel' : 'Expand right panel'}
	aria-expanded={isOpen}
>
	<svg
		class="dw-rp-toggle-icon"
		class:dw-rp-toggle-icon--flipped={isOpen}
		fill="none"
		stroke="currentColor"
		viewBox="0 0 24 24"
		aria-hidden="true"
	>
		<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
	</svg>
</button>

{#if isOpen}
	<!-- Panel -->
	<aside
		class="dw-rp-panel"
		transition:fly={{ x: 320, duration: 240 }}
		aria-label="Dashboard right panel"
		role="complementary"
	>
		<!-- Tab bar -->
		<div class="dw-rp-tabbar" role="tablist" aria-label="Panel sections">
			{#each (['widgets', 'analytics', 'activity'] as TabId[]) as tab}
				<button
					class="dw-rp-tab"
					class:dw-rp-tab--active={activeTab === tab}
					onclick={() => (activeTab = tab)}
					role="tab"
					aria-selected={activeTab === tab}
					aria-controls="dw-rp-panel-{tab}"
					id="dw-rp-tab-{tab}"
				>
					{tab.charAt(0).toUpperCase() + tab.slice(1)}
				</button>
			{/each}
		</div>

		<!-- ── TAB PANELS ─────────────────────────────────────────────────────── -->
		<div class="dw-rp-content" role="tabpanel" id="dw-rp-panel-{activeTab}" aria-labelledby="dw-rp-tab-{activeTab}">

			<!-- ── WIDGETS TAB ──────────────────────────────────────────────────── -->
			{#if activeTab === 'widgets'}
				<div class="dw-rp-section" in:fade={{ duration: 150 }}>

					<!-- Header row -->
					<div class="dw-rp-section-header">
						<span class="dw-rp-section-title">Add Widgets</span>
						{#if layout.isEditMode}
							<span class="dw-rp-edit-badge">Edit Mode</span>
						{/if}
					</div>

					<!-- Size selector -->
					<div class="dw-rp-size-row" role="group" aria-label="Default widget size">
						<span class="dw-rp-size-label">Default size</span>
						<div class="dw-rp-seg-ctrl">
							{#each widgetSizes as s}
								<button
									class="dw-rp-seg-btn"
									class:dw-rp-seg-btn--active={pickerSize === s.value}
									onclick={() => (pickerSize = s.value)}
									aria-pressed={pickerSize === s.value}
									aria-label="Size {s.label}"
								>
									{s.label}
								</button>
							{/each}
						</div>
					</div>

					<!-- Widget list -->
					<ul class="dw-rp-widget-list" role="list">
						{#each availableWidgets as widget (widget.type)}
							{@const added = isWidgetAdded(widget.type)}
							<li>
								<button
									class="dw-rp-action-card"
									class:dw-rp-action-card--disabled={added}
									onclick={() => !added && layout.addWidget(widget.type)}
									disabled={added}
									aria-label="{added ? 'Already added' : 'Add'} {widget.title} widget"
									aria-disabled={added}
								>
									<!-- Icon -->
									<div class="dw-rp-widget-icon" aria-hidden="true">
										<svg viewBox="0 0 24 24" fill="currentColor" class="w-4 h-4">
											<path d={widget.icon} />
										</svg>
									</div>

									<!-- Text -->
									<div class="dw-rp-widget-text">
										<span class="dw-rp-widget-title">{widget.title}</span>
										<span class="dw-rp-widget-desc">{widget.description}</span>
									</div>

									<!-- State indicator -->
									{#if added}
										<span class="dw-rp-added-badge" aria-label="Already on dashboard">Added</span>
									{:else}
										<svg class="dw-rp-add-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
										</svg>
									{/if}
								</button>
							</li>
						{/each}
					</ul>

					{#if !layout.isEditMode}
						<p class="dw-rp-tip">
							Enable edit mode (press <kbd class="dw-rp-kbd">E</kbd>) to rearrange widgets on your dashboard.
						</p>
					{/if}
				</div>

			<!-- ── ANALYTICS TAB ───────────────────────────────────────────────── -->
			{:else if activeTab === 'analytics'}
				<div class="dw-rp-section" in:fade={{ duration: 150 }}>

					<!-- Time range selector -->
					<div class="dw-rp-range-row" role="group" aria-label="Time range">
						{#each timeRangeOptions as opt}
							<button
								class="dw-rp-range-pill"
								class:dw-rp-range-pill--active={analytics.analyticsTimeRange === opt.value}
								onclick={() => handleTimeRange(opt.value)}
								aria-pressed={analytics.analyticsTimeRange === opt.value}
							>
								{opt.label}
							</button>
						{/each}
					</div>

					{#if analytics.analyticsLoading}
						<!-- Loading skeleton -->
						<div class="dw-rp-skeleton-grid" aria-busy="true" aria-label="Loading analytics">
							{#each [1, 2, 3, 4] as _}
								<div class="dw-rp-metric-card dw-rp-skeleton-card">
									<div class="dw-rp-skel dw-rp-skel--val animate-pulse"></div>
									<div class="dw-rp-skel dw-rp-skel--lbl animate-pulse"></div>
								</div>
							{/each}
						</div>
					{:else if analytics.seededAnalytics}
						{@const s = analytics.seededAnalytics}

						<!-- Stat cards grid -->
						<div class="dw-rp-metric-grid">
							<div class="dw-rp-metric-card" transition:scale={{ delay: 40, duration: 180 }}>
								<div class="dw-rp-metric-val tabular-nums">{Math.round($animFocusRate)}%</div>
								<div class="dw-rp-metric-lbl">Focus completion</div>
								<div class="dw-rp-metric-trend dw-rp-metric-trend--up">+12%</div>
							</div>

							<div class="dw-rp-metric-card" transition:scale={{ delay: 80, duration: 180 }}>
								<div class="dw-rp-metric-val tabular-nums">{Math.round($animTasksDone)}</div>
								<div class="dw-rp-metric-lbl">Tasks done</div>
								{#if s.tasks.completionRate > 0}
									<div class="dw-rp-metric-trend dw-rp-metric-trend--up">{s.tasks.completionRate}%</div>
								{/if}
							</div>

							<div class="dw-rp-metric-card" transition:scale={{ delay: 120, duration: 180 }}>
								<div class="dw-rp-metric-val tabular-nums">{Math.round($animActiveProjects)}</div>
								<div class="dw-rp-metric-lbl">Active projects</div>
								{#if s.projects.atRisk > 0}
									<div class="dw-rp-metric-trend dw-rp-metric-trend--warn">{s.projects.atRisk} at risk</div>
								{/if}
							</div>

							<div class="dw-rp-metric-card" transition:scale={{ delay: 160, duration: 180 }}>
								<div class="dw-rp-metric-val tabular-nums">{Math.round($animTotalActivity)}</div>
								<div class="dw-rp-metric-lbl">Total activity</div>
							</div>
						</div>

						<!-- Divider -->
						<div class="dw-rp-divider" role="separator"></div>

						<!-- Tasks bar chart -->
						<div class="dw-rp-chart-section">
							<div class="dw-rp-chart-header">
								<span class="dw-rp-chart-title">Tasks this week</span>
								<span class="dw-rp-chart-meta">{s.tasks.completedThisWeek} completed</span>
							</div>
							<div class="dw-rp-bar-chart" aria-label="Weekly task completion chart" role="img">
								{#each s.tasks.weeklyData as val, i}
									{@const max = Math.max(...s.tasks.weeklyData, 1)}
									{@const pct = Math.max(8, Math.round((val / max) * 100))}
									{@const isToday = i === ((new Date().getDay() + 6) % 7)}
									<div class="dw-rp-bar-col">
										<span class="dw-rp-bar-num">{val > 0 ? val : ''}</span>
										<div
											class="dw-rp-bar"
											class:dw-rp-bar--today={isToday}
											style="height: {pct}%;"
											aria-label="{['Mon','Tue','Wed','Thu','Fri','Sat','Sun'][i]}: {val}"
											role="presentation"
										></div>
									</div>
								{/each}
							</div>
							<div class="dw-rp-day-labels" aria-hidden="true">
								<span>M</span><span>T</span><span>W</span><span>T</span><span>F</span><span>S</span><span>S</span>
							</div>
						</div>

						<div class="dw-rp-divider" role="separator"></div>

						<!-- Summary stats list -->
						<dl class="dw-rp-stat-list">
							<div class="dw-rp-stat-row">
								<dt class="dw-rp-stat-label">On-time delivery</dt>
								<dd class="dw-rp-stat-val">{s.projects.onTimeRate}%</dd>
							</div>
							<div class="dw-rp-stat-row">
								<dt class="dw-rp-stat-label">Tasks due today</dt>
								<dd class="dw-rp-stat-val">{s.tasks.dueToday}</dd>
							</div>
							<div class="dw-rp-stat-row">
								<dt class="dw-rp-stat-label">Overdue</dt>
								<dd class="dw-rp-stat-val dw-rp-stat-val--danger">{s.tasks.overdue}</dd>
							</div>
							<div class="dw-rp-stat-row">
								<dt class="dw-rp-stat-label">Focus streak</dt>
								<dd class="dw-rp-stat-val">{s.focus.streak} days</dd>
							</div>
							<div class="dw-rp-stat-row">
								<dt class="dw-rp-stat-label">Most active day</dt>
								<dd class="dw-rp-stat-val">{s.activity.mostActiveDay}</dd>
							</div>
						</dl>
					{:else}
						<!-- Empty analytics state -->
						<div class="dw-rp-empty">
							<div class="dw-rp-empty-icon" aria-hidden="true">
								<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" class="w-6 h-6">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
								</svg>
							</div>
							<p class="dw-rp-empty-text">No analytics data yet. Complete tasks to see your stats.</p>
						</div>
					{/if}
				</div>

			<!-- ── ACTIVITY TAB ────────────────────────────────────────────────── -->
			{:else if activeTab === 'activity'}
				<div class="dw-rp-section" in:fade={{ duration: 150 }}>

					<div class="dw-rp-section-header">
						<span class="dw-rp-section-title">Recent Activity</span>
					</div>

					{#if data.activities.length === 0}
						<div class="dw-rp-empty">
							<div class="dw-rp-empty-icon" aria-hidden="true">
								<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" class="w-6 h-6">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
								</svg>
							</div>
							<p class="dw-rp-empty-text">No recent activity. Start working to see updates here.</p>
						</div>
					{:else}
						<ol class="dw-rp-feed" aria-label="Activity feed">
							{#each data.activities.slice(0, 10) as item (item.id)}
								<li class="dw-rp-feed-item" in:fade={{ duration: 100 }}>
									<!-- Avatar dot -->
									<div
										class="dw-rp-feed-dot"
										style="background-color: {getActivityDotColor(item.type)};"
										aria-hidden="true"
									></div>

									<!-- Content -->
									<div class="dw-rp-feed-body">
										<p class="dw-rp-feed-desc">{item.description}</p>
										<div class="dw-rp-feed-meta">
											{#if item.actorName}
												<span class="dw-rp-feed-actor">{item.actorName}</span>
												<span class="dw-rp-feed-sep" aria-hidden="true">·</span>
											{/if}
											<time
												class="dw-rp-feed-time"
												datetime={item.createdAt}
												title={new Date(item.createdAt).toLocaleString()}
											>
												{formatActivityTime(item.createdAt)}
											</time>
										</div>
									</div>
								</li>
							{/each}
						</ol>

						{#if data.activities.length > 10}
							<div class="dw-rp-feed-footer">
								<a href="/activity" class="dw-rp-view-all-link">
									View all activity
									<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" class="w-3.5 h-3.5" aria-hidden="true">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
									</svg>
								</a>
							</div>
						{/if}
					{/if}
				</div>
			{/if}

		</div>
	</aside>
{/if}

<style>
	/* ── Toggle button ──────────────────────────────────────────────────────────── */
	.dw-rp-toggle-btn {
		position: absolute;
		left: -16px;
		top: 50%;
		transform: translateY(-50%);
		z-index: 10;
		display: flex;
		align-items: center;
		justify-content: center;
		width: 20px;
		height: 48px;
		background: var(--dbg);
		border: 1px solid var(--dbd);
		border-right: none;
		border-radius: var(--radius-sm) 0 0 var(--radius-sm);
		color: var(--dt3);
		cursor: pointer;
		transition: color 150ms, background 150ms;
		box-shadow: var(--shadow-sm);
	}
	.dw-rp-toggle-btn:hover {
		color: var(--dt);
		background: var(--dbg2);
	}
	.dw-rp-toggle-btn:focus-visible {
		outline: 2px solid var(--dt);
		outline-offset: 2px;
	}

	.dw-rp-toggle-icon {
		width: 14px;
		height: 14px;
		transition: transform 200ms;
	}
	.dw-rp-toggle-icon--flipped {
		transform: rotate(180deg);
	}

	/* ── Panel shell ────────────────────────────────────────────────────────────── */
	.dw-rp-panel {
		position: relative;
		width: 320px;
		flex-shrink: 0;
		height: 100%;
		display: flex;
		flex-direction: column;
		background: var(--dbg);
		border-left: 1px solid var(--dbd);
		overflow: hidden;
		box-shadow: var(--shadow-md);
	}

	/* ── Tab bar ────────────────────────────────────────────────────────────────── */
	.dw-rp-tabbar {
		display: flex;
		align-items: stretch;
		border-bottom: 1px solid var(--dbd);
		background: var(--dbg);
		flex-shrink: 0;
	}

	.dw-rp-tab {
		flex: 1;
		padding: var(--space-3) var(--space-2);
		font-size: var(--text-xs);
		font-weight: var(--font-medium);
		color: var(--dt3);
		background: transparent;
		border: none;
		border-bottom: 2px solid transparent;
		margin-bottom: -1px;
		cursor: pointer;
		transition: color 150ms, border-color 150ms;
		letter-spacing: 0.02em;
	}
	.dw-rp-tab:hover {
		color: var(--dt2);
	}
	.dw-rp-tab--active {
		color: var(--dt);
		border-bottom-color: var(--dt);
	}
	.dw-rp-tab:focus-visible {
		outline: 2px solid var(--dt);
		outline-offset: -2px;
	}

	/* ── Scrollable content area ────────────────────────────────────────────────── */
	.dw-rp-content {
		flex: 1;
		overflow-y: auto;
		overflow-x: hidden;
		scrollbar-width: thin;
		scrollbar-color: var(--dbd) transparent;
	}
	.dw-rp-content::-webkit-scrollbar {
		width: 4px;
	}
	.dw-rp-content::-webkit-scrollbar-thumb {
		background: var(--dbd);
		border-radius: 2px;
	}

	/* ── Section wrapper ────────────────────────────────────────────────────────── */
	.dw-rp-section {
		padding: var(--space-4) var(--space-4) var(--space-6);
		display: flex;
		flex-direction: column;
		gap: var(--space-4);
	}

	.dw-rp-section-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
	}
	.dw-rp-section-title {
		font-size: var(--text-sm);
		font-weight: var(--font-semibold);
		color: var(--dt);
	}

	.dw-rp-edit-badge {
		font-size: var(--text-xs);
		font-weight: var(--font-medium);
		color: var(--dt2);
		background: var(--dbg3);
		border: 1px solid var(--dbd);
		border-radius: var(--radius-full);
		padding: 2px var(--space-2);
	}

	.dw-rp-divider {
		height: 1px;
		background: var(--dbd2);
		margin: 0 calc(-1 * var(--space-4));
	}

	/* ── Widget size selector ───────────────────────────────────────────────────── */
	.dw-rp-size-row {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: var(--space-3);
	}
	.dw-rp-size-label {
		font-size: var(--text-xs);
		color: var(--dt3);
	}
	.dw-rp-seg-ctrl {
		display: flex;
		background: var(--dbg2);
		border: 1px solid var(--dbd);
		border-radius: var(--radius-sm);
		padding: 2px;
		gap: 2px;
	}
	.dw-rp-seg-btn {
		padding: 3px var(--space-3);
		font-size: var(--text-xs);
		font-weight: var(--font-medium);
		color: var(--dt3);
		background: transparent;
		border: none;
		border-radius: calc(var(--radius-sm) - 2px);
		cursor: pointer;
		transition: color 120ms, background 120ms, box-shadow 120ms;
	}
	.dw-rp-seg-btn:hover {
		color: var(--dt2);
	}
	.dw-rp-seg-btn--active {
		color: var(--dt);
		background: var(--dbg);
		box-shadow: var(--shadow-sm);
	}

	/* ── Widget action cards ────────────────────────────────────────────────────── */
	.dw-rp-widget-list {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: var(--space-2);
	}

	.dw-rp-action-card {
		width: 100%;
		display: flex;
		align-items: center;
		gap: var(--space-3);
		padding: var(--space-3);
		background: var(--dbg2);
		border: 1px solid var(--dbd2);
		border-radius: var(--radius-sm);
		cursor: pointer;
		text-align: left;
		transition: background 140ms, border-color 140ms, box-shadow 140ms;
	}
	.dw-rp-action-card:hover:not(.dw-rp-action-card--disabled) {
		background: var(--dbg3);
		border-color: var(--dbd);
		box-shadow: var(--shadow-sm);
	}
	.dw-rp-action-card:focus-visible {
		outline: 2px solid var(--dt);
		outline-offset: 2px;
	}
	.dw-rp-action-card--disabled {
		opacity: 0.55;
		cursor: default;
	}

	.dw-rp-widget-icon {
		width: 32px;
		height: 32px;
		flex-shrink: 0;
		display: flex;
		align-items: center;
		justify-content: center;
		background: var(--dbg3);
		border: 1px solid var(--dbd);
		border-radius: var(--radius-sm);
		color: var(--dt2);
	}

	.dw-rp-widget-text {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: 2px;
	}
	.dw-rp-widget-title {
		font-size: var(--text-sm);
		font-weight: var(--font-medium);
		color: var(--dt);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	.dw-rp-widget-desc {
		font-size: var(--text-xs);
		color: var(--dt3);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.dw-rp-added-badge {
		flex-shrink: 0;
		font-size: 10px;
		font-weight: var(--font-medium);
		color: var(--dt3);
		background: var(--dbg3);
		border: 1px solid var(--dbd);
		border-radius: var(--radius-full);
		padding: 2px 6px;
		white-space: nowrap;
	}

	.dw-rp-add-icon {
		flex-shrink: 0;
		width: 16px;
		height: 16px;
		color: var(--dt3);
		transition: color 120ms;
	}
	.dw-rp-action-card:hover:not(.dw-rp-action-card--disabled) .dw-rp-add-icon {
		color: var(--dt);
	}

	.dw-rp-tip {
		font-size: var(--text-xs);
		color: var(--dt3);
		line-height: 1.5;
		margin: 0;
	}
	.dw-rp-kbd {
		display: inline-block;
		padding: 1px 5px;
		font-size: 11px;
		font-family: inherit;
		font-weight: var(--font-medium);
		color: var(--dt2);
		background: var(--dbg3);
		border: 1px solid var(--dbd);
		border-radius: var(--radius-xs);
	}

	/* ── Analytics: time range pills ────────────────────────────────────────────── */
	.dw-rp-range-row {
		display: flex;
		gap: var(--space-1);
		background: var(--dbg2);
		border: 1px solid var(--dbd2);
		border-radius: var(--radius-md);
		padding: 3px;
	}
	.dw-rp-range-pill {
		flex: 1;
		padding: 4px var(--space-2);
		font-size: var(--text-xs);
		font-weight: var(--font-medium);
		color: var(--dt3);
		background: transparent;
		border: none;
		border-radius: calc(var(--radius-md) - 3px);
		cursor: pointer;
		transition: color 120ms, background 120ms, box-shadow 120ms;
		white-space: nowrap;
	}
	.dw-rp-range-pill:hover {
		color: var(--dt2);
	}
	.dw-rp-range-pill--active {
		color: var(--dt);
		background: var(--dbg);
		box-shadow: var(--shadow-sm);
	}
	.dw-rp-range-pill:focus-visible {
		outline: 2px solid var(--dt);
		outline-offset: 2px;
	}

	/* ── Analytics: metric grid ─────────────────────────────────────────────────── */
	.dw-rp-skeleton-grid,
	.dw-rp-metric-grid {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: var(--space-3);
	}

	.dw-rp-metric-card {
		background: var(--dbg2);
		padding: var(--space-3);
		display: flex;
		flex-direction: column;
		gap: 3px;
	}
	.dw-rp-metric-val {
		font-size: var(--text-xl);
		font-weight: var(--font-semibold);
		color: var(--dt);
		line-height: 1;
	}
	.dw-rp-metric-lbl {
		font-size: var(--text-xs);
		color: var(--dt3);
	}
	.dw-rp-metric-trend {
		font-size: 10px;
		font-weight: var(--font-medium);
		margin-top: 2px;
	}
	.dw-rp-metric-trend--up {
		color: var(--color-success, #059669);
	}
	.dw-rp-metric-trend--down {
		color: var(--color-error, #dc2626);
	}
	.dw-rp-metric-trend--warn {
		color: var(--accent-orange, #d97706);
	}

	/* Skeleton state */
	.dw-rp-skeleton-card {
		pointer-events: none;
	}
	.dw-rp-skel {
		border-radius: var(--radius-xs);
		background: var(--dbg3);
	}
	.dw-rp-skel--val {
		width: 48px;
		height: 24px;
		margin-bottom: 4px;
	}
	.dw-rp-skel--lbl {
		width: 72px;
		height: 12px;
	}

	/* ── Analytics: bar chart ───────────────────────────────────────────────────── */
	.dw-rp-chart-section {
		display: flex;
		flex-direction: column;
		gap: var(--space-2);
	}
	.dw-rp-chart-header {
		display: flex;
		align-items: baseline;
		justify-content: space-between;
	}
	.dw-rp-chart-title {
		font-size: var(--text-xs);
		font-weight: var(--font-semibold);
		color: var(--dt);
	}
	.dw-rp-chart-meta {
		font-size: var(--text-xs);
		color: var(--dt3);
	}

	.dw-rp-bar-chart {
		display: flex;
		align-items: flex-end;
		gap: var(--space-1);
		height: 56px;
	}
	.dw-rp-bar-col {
		flex: 1;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: flex-end;
		height: 100%;
		gap: 2px;
	}
	.dw-rp-bar-num {
		font-size: 9px;
		color: var(--dt3);
		line-height: 1;
	}
	.dw-rp-bar {
		width: 100%;
		max-width: 28px;
		border-radius: var(--radius-xs) var(--radius-xs) 0 0;
		background: var(--dt4);
		transition: height 450ms cubic-bezier(0.4, 0, 0.2, 1), background 150ms;
	}
	.dw-rp-bar:hover {
		background: var(--dt2);
	}
	.dw-rp-bar--today {
		background: var(--dt);
	}

	.dw-rp-day-labels {
		display: flex;
		justify-content: space-between;
		font-size: 10px;
		color: var(--dt3);
		font-weight: var(--font-medium);
	}
	.dw-rp-day-labels span {
		flex: 1;
		text-align: center;
	}

	/* ── Analytics: stat list ───────────────────────────────────────────────────── */
	.dw-rp-stat-list {
		display: flex;
		flex-direction: column;
		margin: 0;
		padding: 0;
	}
	.dw-rp-stat-row {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: var(--space-2) 0;
		border-bottom: 1px solid var(--dbd2);
	}
	.dw-rp-stat-row:last-child {
		border-bottom: none;
	}
	.dw-rp-stat-label {
		font-size: var(--text-sm);
		color: var(--dt2);
	}
	.dw-rp-stat-val {
		font-size: var(--text-sm);
		font-weight: var(--font-medium);
		color: var(--dt);
	}
	.dw-rp-stat-val--danger {
		color: var(--color-error, #dc2626);
	}

	/* ── Activity feed ──────────────────────────────────────────────────────────── */
	.dw-rp-feed {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 0;
	}

	.dw-rp-feed-item {
		display: flex;
		gap: var(--space-3);
		padding: var(--space-3) 0;
		border-bottom: 1px solid var(--dbd2);
	}
	.dw-rp-feed-item:last-child {
		border-bottom: none;
	}

	.dw-rp-feed-dot {
		width: 8px;
		height: 8px;
		border-radius: var(--radius-full);
		flex-shrink: 0;
		margin-top: 5px;
	}

	.dw-rp-feed-body {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: 3px;
	}
	.dw-rp-feed-desc {
		font-size: var(--text-sm);
		color: var(--dt);
		line-height: 1.4;
		margin: 0;
		/* Limit to 2 lines */
		display: -webkit-box;
		-webkit-line-clamp: 2;
		-webkit-box-orient: vertical;
		overflow: hidden;
	}
	.dw-rp-feed-meta {
		display: flex;
		align-items: center;
		gap: var(--space-1);
	}
	.dw-rp-feed-actor {
		font-size: var(--text-xs);
		font-weight: var(--font-medium);
		color: var(--dt2);
	}
	.dw-rp-feed-sep {
		font-size: var(--text-xs);
		color: var(--dt4);
	}
	.dw-rp-feed-time {
		font-size: var(--text-xs);
		color: var(--dt3);
	}

	.dw-rp-feed-footer {
		padding-top: var(--space-3);
		text-align: center;
	}
	.dw-rp-view-all-link {
		display: inline-flex;
		align-items: center;
		gap: var(--space-1);
		font-size: var(--text-xs);
		font-weight: var(--font-medium);
		color: var(--dt2);
		text-decoration: none;
		transition: color 120ms;
	}
	.dw-rp-view-all-link:hover {
		color: var(--dt);
	}
	.dw-rp-view-all-link:focus-visible {
		outline: 2px solid var(--dt);
		outline-offset: 2px;
		border-radius: var(--radius-xs);
	}

	/* ── Empty states ───────────────────────────────────────────────────────────── */
	.dw-rp-empty {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: var(--space-3);
		padding: var(--space-8) var(--space-4);
		text-align: center;
	}
	.dw-rp-empty-icon {
		width: 40px;
		height: 40px;
		display: flex;
		align-items: center;
		justify-content: center;
		background: var(--dbg2);
		border: 1px solid var(--dbd);
		border-radius: var(--radius-md);
		color: var(--dt3);
	}
	.dw-rp-empty-text {
		font-size: var(--text-sm);
		color: var(--dt3);
		line-height: 1.5;
		max-width: 220px;
		margin: 0;
	}
</style>
