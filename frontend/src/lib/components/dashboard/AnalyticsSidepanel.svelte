<script lang="ts">
	import { fly, fade, scale } from 'svelte/transition';
	import { tweened } from 'svelte/motion';
	import { cubicOut } from 'svelte/easing';
	
	interface Props {
		isOpen: boolean;
		onClose: () => void;
		analytics: DashboardAnalytics | null;
		isLoading?: boolean;
		onTimeRangeChange?: (range: TimeRange) => void;
	}
	
	type TimeRange = 'today' | 'week' | 'month' | '30days';
	
	interface DashboardAnalytics {
		focus: {
			completionRate: number;
			completedToday: number;
			totalToday: number;
			streak: number;
			avgCompletionTime: string;
			weeklyData: number[];
		};
		tasks: {
			completedThisWeek: number;
			dueToday: number;
			overdue: number;
			completionRate: number;
			byPriority: { critical: number; high: number; medium: number; low: number };
			weeklyData: number[];
		};
		projects: {
			active: number;
			completed: number;
			atRisk: number;
			onTimeRate: number;
			avgProgress: number;
		};
		activity: {
			totalActions: number;
			mostActiveDay: string;
			topActivityType: string;
			weeklyData: number[];
		};
	}
	
	let { isOpen, onClose, analytics, isLoading = false, onTimeRangeChange }: Props = $props();
	
	// Time range selector
	let timeRange = $state<TimeRange>('week');
	
	const timeRanges = [
		{ value: 'today', label: 'Today' },
		{ value: 'week', label: 'Week' },
		{ value: 'month', label: 'Month' },
		{ value: '30days', label: '30d' },
	];
	
	// Animated number stores
	const animatedFocusRate = tweened(0, { duration: 600, easing: cubicOut });
	const animatedTasksCompleted = tweened(0, { duration: 600, easing: cubicOut });
	const animatedActiveProjects = tweened(0, { duration: 600, easing: cubicOut });
	const animatedStreak = tweened(0, { duration: 600, easing: cubicOut });
	const animatedAvgProgress = tweened(0, { duration: 800, easing: cubicOut });
	const animatedTotalActions = tweened(0, { duration: 600, easing: cubicOut });
	
	// Update animated values when analytics change
	$effect(() => {
		if (analytics && isOpen) {
			animatedFocusRate.set(analytics.focus.completionRate);
			animatedTasksCompleted.set(analytics.tasks.completedThisWeek);
			animatedActiveProjects.set(analytics.projects.active);
			animatedStreak.set(analytics.focus.streak);
			animatedAvgProgress.set(analytics.projects.avgProgress);
			animatedTotalActions.set(analytics.activity.totalActions);
		}
	});
	
	// Reset animations when panel closes
	$effect(() => {
		if (!isOpen) {
			animatedFocusRate.set(0, { duration: 0 });
			animatedTasksCompleted.set(0, { duration: 0 });
			animatedActiveProjects.set(0, { duration: 0 });
			animatedStreak.set(0, { duration: 0 });
			animatedAvgProgress.set(0, { duration: 0 });
			animatedTotalActions.set(0, { duration: 0 });
		}
	});
	
	function handleTimeRangeChange(range: TimeRange) {
		timeRange = range;
		onTimeRangeChange?.(range);
	}
	
	// Check if data is empty
	function hasNoData(data: DashboardAnalytics | null): boolean {
		if (!data) return true;
		return (
			data.focus.completionRate === 0 &&
			data.tasks.completedThisWeek === 0 &&
			data.projects.active === 0 &&
			data.activity.totalActions === 0
		);
	}
	
	// Get time range label
	function getTimeRangeLabel(range: TimeRange): string {
		switch (range) {
			case 'today': return 'Today';
			case 'week': return 'This Week';
			case 'month': return 'This Month';
			case '30days': return 'Last 30 Days';
		}
	}
</script>

{#if isOpen}
	<!-- Backdrop -->
	<button
		class="fixed inset-0 bg-black/20 z-40"
		onclick={onClose}
		transition:fade={{ duration: 150 }}
		aria-label="Close analytics"
	></button>
	
	<!-- Sidepanel -->
	<div 
		class="dw-ap-panel fixed inset-0 sm:inset-auto sm:top-0 sm:right-0 sm:bottom-0 sm:w-[420px] sm:border-l z-50 flex flex-col"
		transition:fly={{ x: 420, duration: 300 }}
	>
		<!-- Header -->
		<div class="dw-ap-divider flex items-center justify-between px-5 sm:px-6 py-4 sm:py-5 border-b">
			<div>
				<h2 class="dw-ap-title text-lg font-semibold">Analytics</h2>
				<p class="dw-ap-subtitle text-sm mt-0.5">Dashboard performance overview</p>
			</div>
			<button
				onclick={onClose}
				class="btn-pill btn-pill-ghost btn-pill-icon btn-pill-xs"
				aria-label="Close"
			>
				<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
				</svg>
			</button>
		</div>
		
		<!-- Time Range Selector -->
		<div class="dw-ap-divider px-5 sm:px-6 py-4 border-b">
			<div class="dw-ap-range-bg flex items-center gap-1 p-1 rounded-lg">
				{#each timeRanges as range}
					<button
						onclick={() => handleTimeRangeChange(range.value as TimeRange)}
						class="btn-pill btn-pill-xs flex-1 {timeRange === range.value ? 'btn-pill-soft shadow-sm' : 'btn-pill-ghost'}"
					>
						{range.label}
					</button>
				{/each}
			</div>
		</div>
		
		<!-- Analytics Content -->
		<div class="flex-1 overflow-y-auto">
			{#if isLoading}
				<!-- Loading Skeleton -->
				<div class="dw-ap-divider px-5 sm:px-6 py-5 border-b">
					<div class="grid grid-cols-2 gap-3">
						{#each [1, 2, 3, 4] as _}
							<div class="dw-ap-card p-4 rounded-xl border animate-pulse">
								<div class="dw-ap-skeleton h-7 w-16 rounded mb-2"></div>
								<div class="dw-ap-skeleton h-4 w-24 rounded"></div>
							</div>
						{/each}
					</div>
				</div>
				
				<div class="dw-ap-divider px-5 sm:px-6 py-5 border-b">
					<div class="dw-ap-skeleton h-4 w-12 rounded mb-4 animate-pulse"></div>
					<div class="flex items-end gap-1.5 h-20 mb-4">
						{#each [1, 2, 3, 4, 5, 6, 7] as _, i}
							<div class="flex-1 dw-ap-skeleton rounded-t animate-pulse" style="height: {20 + Math.random() * 60}%"></div>
						{/each}
					</div>
					<div class="space-y-3">
						{#each [1, 2, 3] as _}
							<div class="flex justify-between animate-pulse">
								<div class="dw-ap-skeleton h-4 w-20 rounded"></div>
								<div class="dw-ap-skeleton h-4 w-8 rounded"></div>
							</div>
						{/each}
					</div>
				</div>
			{:else if hasNoData(analytics)}
				<!-- Empty State -->
				<div class="flex-1 flex flex-col items-center justify-center px-8 py-16 text-center">
					<div class="dw-ap-empty-icon w-16 h-16 rounded-2xl flex items-center justify-center mb-4">
						<svg class="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
						</svg>
					</div>
					<h3 class="dw-ap-title text-base font-medium mb-1">No activity yet</h3>
					<p class="dw-ap-subtitle text-sm max-w-[240px]">
						Complete your first focus item or task to start tracking your productivity.
					</p>
					<button
						onclick={onClose}
						class="btn-pill btn-pill-primary btn-pill-sm mt-6"
					>
						Get Started
					</button>
				</div>
			{:else if analytics}
				<!-- Summary Cards -->
				<div class="dw-ap-divider px-5 sm:px-6 py-5 border-b">
					<div class="grid grid-cols-2 gap-3">
						<div class="dw-ap-card p-4 rounded-xl border" transition:scale={{ delay: 50, duration: 200 }}>
							<div class="dw-ap-value text-2xl font-semibold tabular-nums">{Math.round($animatedFocusRate)}%</div>
							<div class="dw-ap-label text-sm mt-1">Focus Completion</div>
						</div>
						<div class="dw-ap-card p-4 rounded-xl border" transition:scale={{ delay: 100, duration: 200 }}>
							<div class="dw-ap-value text-2xl font-semibold tabular-nums">{Math.round($animatedTasksCompleted)}</div>
							<div class="dw-ap-label text-sm mt-1">Tasks Completed</div>
						</div>
						<div class="dw-ap-card p-4 rounded-xl border" transition:scale={{ delay: 150, duration: 200 }}>
							<div class="dw-ap-value text-2xl font-semibold tabular-nums">{Math.round($animatedActiveProjects)}</div>
							<div class="dw-ap-label text-sm mt-1">Active Projects</div>
						</div>
						<div class="dw-ap-card p-4 rounded-xl border" transition:scale={{ delay: 200, duration: 200 }}>
							<div class="flex items-center gap-1.5">
								<span class="dw-ap-value text-2xl font-semibold tabular-nums">{Math.round($animatedStreak)}</span>
								<svg class="dw-ap-streak-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" aria-hidden="true">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.75" d="M13 7h8m0 0l-3-3m3 3l-3 3M3 17h8m0 0l-3-3m3 3l-3 3M8 12h8" />
								</svg>
							</div>
							<div class="dw-ap-label text-sm mt-1">Day Streak</div>
						</div>
					</div>
				</div>
				
				<!-- Tasks Section -->
				<div class="dw-ap-divider px-5 sm:px-6 py-5 border-b">
					<div class="flex items-center justify-between mb-4">
						<h3 class="dw-ap-section-title text-sm font-semibold">Tasks</h3>
						<span class="dw-ap-meta text-xs">{getTimeRangeLabel(timeRange)}</span>
					</div>
					
					<!-- Bar Chart -->
					<div class="dw-ap-card mb-5 p-4 rounded-xl border">
						<div class="flex items-end justify-between gap-2" style="height: 80px;">
							{#each analytics.tasks.weeklyData as value, i}
								{@const max = Math.max(...analytics.tasks.weeklyData, 1)}
								{@const heightPx = Math.max(6, Math.round((value / max) * 60))}
								{@const isToday = i === new Date().getDay() - 1 || (i === 6 && new Date().getDay() === 0)}
								<div class="flex-1 flex flex-col items-center justify-end h-full">
									<span class="dw-ap-meta text-[10px] tabular-nums mb-1">{value > 0 ? value : ''}</span>
									<div 
										class="dw-ap-bar w-full max-w-[32px] rounded-t-md transition-all duration-500 ease-out"
										class:dw-ap-bar--active={isToday}
										style="height: {heightPx}px; transition-delay: {i * 50}ms"
									></div>
								</div>
							{/each}
						</div>
						<div class="dw-ap-day-labels flex justify-between text-[10px] mt-3 font-medium">
							<span>Mon</span><span>Tue</span><span>Wed</span><span>Thu</span><span>Fri</span><span>Sat</span><span>Sun</span>
						</div>
					</div>
					
					<!-- Task Stats -->
					<div class="space-y-0.5">
						<div class="dw-ap-stat-row flex items-center justify-between py-2.5 border-b">
							<span class="dw-ap-stat-label text-sm">Due Today</span>
							<span class="dw-ap-stat-value text-sm font-medium tabular-nums">{analytics.tasks.dueToday}</span>
						</div>
						<div class="dw-ap-stat-row flex items-center justify-between py-2.5 border-b">
							<span class="dw-ap-stat-label text-sm">Overdue</span>
							<div class="flex items-center gap-2">
								{#if analytics.tasks.overdue > 0}
									<span class="dw-ap-dot-danger w-1.5 h-1.5 rounded-full animate-pulse"></span>
								{/if}
								<span class="dw-ap-stat-value text-sm font-medium tabular-nums">{analytics.tasks.overdue}</span>
							</div>
						</div>
						<div class="flex items-center justify-between py-2.5">
							<span class="dw-ap-stat-label text-sm">Completion Rate</span>
							<span class="dw-ap-stat-value text-sm font-medium tabular-nums">{analytics.tasks.completionRate}%</span>
						</div>
					</div>
				</div>
				
				<!-- Projects Section -->
				<div class="dw-ap-divider px-5 sm:px-6 py-5 border-b">
					<div class="flex items-center justify-between mb-4">
						<h3 class="dw-ap-section-title text-sm font-semibold">Projects</h3>
					</div>
					
					<!-- Progress Ring -->
					<div class="dw-ap-card flex items-center gap-4 mb-5 p-4 rounded-xl border">
						<div class="relative w-16 h-16 flex-shrink-0">
							<svg class="w-16 h-16 transform -rotate-90">
								<circle class="dw-ap-ring-track" cx="32" cy="32" r="28" stroke-width="5" fill="none" />
								<circle 
									class="dw-ap-ring-fill transition-all duration-700"
									cx="32" cy="32" r="28" 
									stroke-width="5" 
									fill="none"
									stroke-dasharray="{$animatedAvgProgress * 1.76} 176"
									stroke-linecap="round"
								/>
							</svg>
							<div class="absolute inset-0 flex items-center justify-center">
								<span class="dw-ap-value text-sm font-semibold tabular-nums">{Math.round($animatedAvgProgress)}%</span>
							</div>
						</div>
						<div>
							<div class="dw-ap-value text-sm font-medium">Average Progress</div>
							<div class="dw-ap-label text-xs mt-0.5">Across {analytics.projects.active} active projects</div>
						</div>
					</div>
					
					<!-- Project Stats -->
					<div class="space-y-0.5">
						<div class="dw-ap-stat-row flex items-center justify-between py-2.5 border-b">
							<span class="dw-ap-stat-label text-sm">Active</span>
							<span class="dw-ap-stat-value text-sm font-medium tabular-nums">{analytics.projects.active}</span>
						</div>
						<div class="dw-ap-stat-row flex items-center justify-between py-2.5 border-b">
							<span class="dw-ap-stat-label text-sm">Completed</span>
							<span class="dw-ap-stat-value text-sm font-medium tabular-nums">{analytics.projects.completed}</span>
						</div>
						<div class="dw-ap-stat-row flex items-center justify-between py-2.5 border-b">
							<span class="dw-ap-stat-label text-sm">At Risk</span>
							<div class="flex items-center gap-2">
								{#if analytics.projects.atRisk > 0}
									<span class="dw-ap-dot-warning w-1.5 h-1.5 rounded-full"></span>
								{/if}
								<span class="dw-ap-stat-value text-sm font-medium tabular-nums">{analytics.projects.atRisk}</span>
							</div>
						</div>
						<div class="flex items-center justify-between py-2.5">
							<span class="dw-ap-stat-label text-sm">On-time Delivery</span>
							<span class="dw-ap-stat-value text-sm font-medium tabular-nums">{analytics.projects.onTimeRate}%</span>
						</div>
					</div>
				</div>
				
				<!-- Activity Section -->
				<div class="px-5 sm:px-6 py-5">
					<div class="flex items-center justify-between mb-4">
						<h3 class="dw-ap-section-title text-sm font-semibold">Activity</h3>
					</div>
					
					<!-- Activity Chart -->
					<div class="dw-ap-card mb-5 p-4 rounded-xl border">
						<div class="flex items-end justify-between gap-2" style="height: 56px;">
							{#each analytics.activity.weeklyData as value, i}
								{@const max = Math.max(...analytics.activity.weeklyData, 1)}
								{@const heightPx = Math.max(4, Math.round((value / max) * 48))}
								{@const intensity = max > 0 ? value / max : 0}
								<div class="flex-1 flex flex-col items-center justify-end h-full">
									<div 
										class="dw-ap-bar dw-ap-bar--active w-full max-w-[28px] rounded-t-md transition-all duration-500 ease-out"
										style="height: {heightPx}px; opacity: {0.25 + intensity * 0.75}; transition-delay: {i * 50}ms"
									></div>
								</div>
							{/each}
						</div>
						<div class="dw-ap-day-labels flex justify-between text-[10px] mt-3 font-medium">
							<span>M</span><span>T</span><span>W</span><span>T</span><span>F</span><span>S</span><span>S</span>
						</div>
					</div>
					
					<!-- Activity Stats -->
					<div class="space-y-0.5">
						<div class="dw-ap-stat-row flex items-center justify-between py-2.5 border-b">
							<span class="dw-ap-stat-label text-sm">Total Actions</span>
							<span class="dw-ap-stat-value text-sm font-medium tabular-nums">{Math.round($animatedTotalActions)}</span>
						</div>
						<div class="flex items-center justify-between py-2.5">
							<span class="dw-ap-stat-label text-sm">Most Active Day</span>
							<span class="dw-ap-stat-value text-sm font-medium">{analytics.activity.mostActiveDay}</span>
						</div>
					</div>
				</div>
			{/if}
		</div>
		
		<!-- Footer -->
		{#if !isLoading && !hasNoData(analytics)}
			<div class="dw-ap-divider px-5 sm:px-6 py-4 border-t">
				<button
					class="btn-pill btn-pill-primary btn-pill-block"
				>
					<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
					</svg>
					Ask AI About This Data
				</button>
			</div>
		{/if}
	</div>
{/if}

<style>
	/* Analytics Sidepanel — Foundation tokens with BOS fallbacks */
	.dw-ap-panel {
		background: var(--dbg, var(--bos-card, #fff));
		border-color: var(--dbd, var(--bos-border, #e0e0e0));
	}

	.dw-ap-divider {
		border-color: var(--dbd2, var(--bos-border, #f0f0f0));
	}

	.dw-ap-title {
		color: var(--dt, var(--bos-text-primary, #111));
	}

	.dw-ap-subtitle {
		color: var(--dt2, var(--bos-text-secondary, #555));
	}

	.dw-ap-range-bg {
		background: var(--dbg2, var(--bos-hover, #f5f5f5));
	}

	/* Cards */
	.dw-ap-card {
		background: var(--dbg2, var(--bos-hover, #f5f5f5));
		border-color: var(--dbd2, var(--bos-border, #f0f0f0));
	}

	.dw-ap-value {
		color: var(--dt, var(--bos-text-primary, #111));
	}

	.dw-ap-label {
		color: var(--dt2, var(--bos-text-secondary, #555));
	}

	/* Section headings & meta */
	.dw-ap-section-title {
		color: var(--dt, var(--bos-text-primary, #111));
	}

	.dw-ap-meta {
		color: var(--dt3, var(--bos-text-tertiary, #888));
	}

	/* Chart bars */
	.dw-ap-bar {
		background-color: var(--dt3, #888);
	}
	.dw-ap-bar:hover {
		background-color: var(--dt2, #555);
	}
	.dw-ap-bar--active {
		background-color: var(--dt, var(--bos-text-primary, #111));
	}

	.dw-ap-day-labels {
		color: var(--dt3, var(--bos-text-tertiary, #888));
	}

	/* Stats rows */
	.dw-ap-stat-row {
		border-color: var(--dbd2, var(--bos-border, #f0f0f0));
	}

	.dw-ap-stat-label {
		color: var(--dt2, var(--bos-text-secondary, #666));
	}

	.dw-ap-stat-value {
		color: var(--dt, var(--bos-text-primary, #111));
	}

	/* Progress ring */
	.dw-ap-ring-track {
		stroke: var(--dbd, var(--bos-border, #e5e7eb));
	}
	.dw-ap-ring-fill {
		stroke: var(--dt, var(--bos-text-primary, #111));
	}

	/* Indicator dots */
	.dw-ap-dot-danger {
		background-color: #ef4444;
	}
	.dw-ap-dot-warning {
		background-color: #f59e0b;
	}

	/* Skeleton loading */
	.dw-ap-skeleton {
		background: var(--dbg3, var(--bos-hover, #eee));
	}

	/* Empty state */
	.dw-ap-empty-icon {
		background: var(--dbg2, var(--bos-hover, #f5f5f5));
		color: var(--dt3, var(--bos-text-tertiary, #888));
	}

	/* Streak trend icon */
	.dw-ap-streak-icon {
		width: 1rem;
		height: 1rem;
		color: var(--dt2, var(--bos-text-secondary, #555));
		flex-shrink: 0;
	}
</style>
