<script lang="ts">
	import { getTaskSuggestions } from '$lib/api/calendar';
	import type { TaskSuggestion, TaskSuggestionsResponse, SuggestionType, SuggestionPriority } from '$lib/api/calendar';
	import { fly, fade, scale } from 'svelte/transition';
	import { flip } from 'svelte/animate';

	/**
	 * TaskSuggestions - Calendar-based task recommendations
	 *
	 * PURPOSE:
	 * This component displays intelligent task suggestions based on calendar analysis:
	 * 1. "Prep" tasks - Things to do BEFORE upcoming meetings (review docs, prepare agenda)
	 * 2. "Follow-up" tasks - Things to do AFTER recent meetings (send notes, action items)
	 *
	 * IMPROVEMENTS:
	 * - Skeleton loaders for compact mode
	 * - Batch "Create All" button
	 * - Undo dismiss with snackbar
	 * - Animation transitions
	 * - Better empty states with helpful tips
	 */

	interface Props {
		/** Optional context ID to filter suggestions */
		contextId?: string;
		/** Optional project ID to filter suggestions */
		projectId?: string;
		/** Callback when user wants to create a task from suggestion */
		onCreateTask?: (suggestion: TaskSuggestion) => void;
		/** Display mode - full panel or compact widget */
		compact?: boolean;
		/** Maximum suggestions to show in compact mode */
		maxCompact?: number;
	}

	let {
		contextId,
		projectId,
		onCreateTask,
		compact = false,
		maxCompact = 3
	}: Props = $props();

	// State
	let loading = $state(true);
	let error = $state<string | null>(null);
	let response = $state<TaskSuggestionsResponse | null>(null);
	let dismissedIds = $state<Set<string>>(new Set());

	// Undo state
	let lastDismissed = $state<{ id: string; suggestion: TaskSuggestion } | null>(null);
	let undoTimeout = $state<ReturnType<typeof setTimeout> | null>(null);
	let undoCountdown = $state(5);
	let undoInterval = $state<ReturnType<typeof setInterval> | null>(null);

	// Batch creation
	let creating = $state(false);
	let createdCount = $state(0);

	// Derived: filter out dismissed suggestions
	const visibleSuggestions = $derived(
		response?.suggestions.filter((s: TaskSuggestion) =>
			!dismissedIds.has(getSuggestionId(s))
		) ?? []
	);

	const displaySuggestions = $derived(
		compact ? visibleSuggestions.slice(0, maxCompact) : visibleSuggestions
	);

	const prepSuggestions = $derived(
		displaySuggestions.filter((s: TaskSuggestion) => s.type === 'prep')
	);

	const followUpSuggestions = $derived(
		displaySuggestions.filter((s: TaskSuggestion) => s.type === 'follow_up')
	);

	const hasMoreSuggestions = $derived(
		compact && visibleSuggestions.length > maxCompact
	);

	// Load suggestions on mount and when filters change
	$effect(() => {
		loadSuggestions();
	});

	// Cleanup on unmount
	$effect(() => {
		return () => {
			if (undoTimeout) clearTimeout(undoTimeout);
			if (undoInterval) clearInterval(undoInterval);
		};
	});

	function getSuggestionId(suggestion: TaskSuggestion): string {
		return `${suggestion.type}-${suggestion.related_event_id}-${suggestion.title}`;
	}

	async function loadSuggestions() {
		loading = true;
		error = null;

		try {
			response = await getTaskSuggestions(contextId, projectId);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load suggestions';
		} finally {
			loading = false;
		}
	}

	function handleDismiss(suggestion: TaskSuggestion) {
		const id = getSuggestionId(suggestion);

		// Clear previous undo state
		if (undoTimeout) clearTimeout(undoTimeout);
		if (undoInterval) clearInterval(undoInterval);

		// Add to dismissed
		dismissedIds = new Set([...dismissedIds, id]);

		// Set up undo
		lastDismissed = { id, suggestion };
		undoCountdown = 5;

		undoInterval = setInterval(() => {
			undoCountdown -= 1;
			if (undoCountdown <= 0 && undoInterval) {
				clearInterval(undoInterval);
			}
		}, 1000);

		undoTimeout = setTimeout(() => {
			lastDismissed = null;
			if (undoInterval) clearInterval(undoInterval);
		}, 5000);
	}

	function handleUndo() {
		if (lastDismissed) {
			dismissedIds = new Set([...dismissedIds].filter(id => id !== lastDismissed?.id));
			lastDismissed = null;
			if (undoTimeout) clearTimeout(undoTimeout);
			if (undoInterval) clearInterval(undoInterval);
		}
	}

	function handleCreate(suggestion: TaskSuggestion) {
		if (onCreateTask) {
			onCreateTask(suggestion);
			// Also dismiss after creating
			const id = getSuggestionId(suggestion);
			dismissedIds = new Set([...dismissedIds, id]);
		}
	}

	async function handleCreateAll() {
		if (!onCreateTask || displaySuggestions.length === 0) return;

		creating = true;
		createdCount = 0;

		for (const suggestion of displaySuggestions) {
			onCreateTask(suggestion);
			createdCount++;
			const id = getSuggestionId(suggestion);
			dismissedIds = new Set([...dismissedIds, id]);
			// Small delay between creations for visual feedback
			await new Promise(resolve => setTimeout(resolve, 150));
		}

		creating = false;
		setTimeout(() => { createdCount = 0; }, 2000);
	}

	function getTypeLabel(type: SuggestionType): string {
		return type === 'prep' ? 'Prepare' : 'Follow-up';
	}

	function getPriorityStyle(priority: SuggestionPriority): string {
		const styles: Record<SuggestionPriority, string> = {
			high: 'background: var(--bos-priority-critical-bg); color: var(--bos-priority-critical); border-color: var(--dbd)',
			medium: 'background: var(--bos-priority-medium-bg); color: var(--bos-priority-medium); border-color: var(--dbd)',
			low: 'background: var(--bos-priority-low-bg); color: var(--bos-priority-low); border-color: var(--dbd)'
		};
		return styles[priority];
	}

	function getConfidenceBarStyle(confidence: number): string {
		if (confidence >= 0.8) return 'background: var(--bos-status-success)';
		if (confidence >= 0.6) return 'background: var(--bos-status-warning)';
		if (confidence >= 0.4) return 'background: var(--bos-status-warning)';
		return 'background: var(--bos-status-error)';
	}

	function getConfidenceWidth(confidence: number): string {
		return `${Math.round(confidence * 100)}%`;
	}

	function formatDueDate(isoString: string): string {
		const date = new Date(isoString);
		const now = new Date();
		const diffDays = Math.ceil((date.getTime() - now.getTime()) / (1000 * 60 * 60 * 24));

		if (diffDays === 0) return 'Today';
		if (diffDays === 1) return 'Tomorrow';
		if (diffDays === -1) return 'Yesterday';
		if (diffDays < 0) return `${Math.abs(diffDays)} days ago`;
		if (diffDays <= 7) return `In ${diffDays} days`;

		return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
	}

	function formatDueDateStyle(isoString: string): string {
		const date = new Date(isoString);
		const now = new Date();
		const diffDays = Math.ceil((date.getTime() - now.getTime()) / (1000 * 60 * 60 * 24));

		if (diffDays < 0) return 'color: var(--bos-priority-critical); background: var(--bos-priority-critical-bg)';
		if (diffDays === 0) return 'color: var(--bos-priority-high); background: var(--bos-priority-high-bg)';
		if (diffDays === 1) return 'color: var(--bos-priority-medium); background: var(--bos-priority-medium-bg)';
		return 'color: var(--dt2); background: var(--dbg2)';
	}
</script>

{#if compact}
	<!-- Compact Widget Mode -->
	<div style="background: var(--dbg); border-color: var(--dbd)" class="rounded-lg border overflow-hidden shadow-sm">
		<div style="border-color: var(--dbd)" class="px-4 py-3 border-b flex items-center justify-between">
			<h3 style="color: var(--dt1)" class="text-sm font-medium flex items-center gap-1.5">
				<!-- Lightbulb icon -->
				<svg class="w-4 h-4 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z" />
				</svg>
				Suggested Tasks
			</h3>
			<div class="flex items-center gap-2">
				{#if hasMoreSuggestions}
					<span style="color: var(--dt2); background: var(--dbg2)" class="text-xs px-2 py-0.5 rounded-full">
						+{visibleSuggestions.length - maxCompact} more
					</span>
				{/if}
				<button
					onclick={loadSuggestions}
					disabled={loading}
					aria-label="Refresh suggestions"
					class="btn-pill btn-pill-ghost btn-pill-icon"
				>
					<svg class="w-4 h-4 {loading ? 'animate-spin' : ''}" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
					</svg>
				</button>
			</div>
		</div>

		{#if loading}
			<!-- Skeleton Loaders -->
			<div class="divide-y" style="border-color: var(--dbd)">
				{#each Array(maxCompact) as _, i}
					<div class="p-3 animate-pulse" transition:fade={{ duration: 150, delay: i * 50 }}>
						<div class="flex items-start gap-2">
							<div style="background: var(--dbg2)" class="w-6 h-6 rounded-full"></div>
							<div class="flex-1 space-y-2">
								<div style="background: var(--dbg2)" class="h-4 rounded w-3/4"></div>
								<div style="background: var(--dbg3)" class="h-3 rounded w-1/2"></div>
							</div>
						</div>
					</div>
				{/each}
			</div>
		{:else if error}
			<div class="p-4 text-center">
				<p style="color: var(--bos-status-error)" class="text-sm mb-2">{error}</p>
				<button
					onclick={loadSuggestions}
					class="btn-pill btn-pill-ghost btn-pill-xs hover:underline"
				>
					Try again
				</button>
			</div>
		{:else if displaySuggestions.length === 0}
			<div class="p-4 text-center" transition:fade>
				<!-- Sparkles icon -->
				<svg class="w-8 h-8 mx-auto mb-1" style="color: var(--dt3)" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 3v4M3 5h4M6 17v4m-2-2h4m5-16l2.286 6.857L21 12l-5.714 2.143L13 21l-2.286-6.857L5 12l5.714-2.143L13 3z" />
				</svg>
				<p style="color: var(--dt2)" class="text-sm">No suggestions right now</p>
				<p style="color: var(--dt3)" class="text-xs mt-1">Check back after your next meeting</p>
			</div>
		{:else}
			<div class="divide-y" style="border-color: var(--dbd)">
				{#each displaySuggestions as suggestion (getSuggestionId(suggestion))}
					<div
						style="--hover-bg: var(--dbg2)"
						class="p-3 group transition-colors hover:[background:var(--dbg2)]"
						animate:flip={{ duration: 200 }}
						transition:fly={{ x: -20, duration: 200 }}
					>
						<div class="flex items-start gap-2">
							<!-- Type icon: clipboard for prep, send for follow_up -->
							{#if suggestion.type === 'prep'}
								<svg class="w-4 h-4 flex-shrink-0 mt-0.5" style="color: var(--dt2)" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
								</svg>
							{:else}
								<svg class="w-4 h-4 flex-shrink-0 mt-0.5" style="color: var(--dt2)" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8" />
								</svg>
							{/if}
							<div class="flex-1 min-w-0">
								<p style="color: var(--dt1)" class="text-sm font-medium truncate">
									{suggestion.title}
								</p>
								<div class="flex items-center gap-2 mt-0.5">
									<span
										style="{formatDueDateStyle(suggestion.suggested_due_date)}"
										class="text-xs px-1.5 py-0.5 rounded"
									>
										{formatDueDate(suggestion.suggested_due_date)}
									</span>
									<span style="color: var(--dt3)" class="text-xs truncate">
										{suggestion.related_event_title}
									</span>
								</div>
							</div>
							<div class="flex items-center gap-1">
								<button
									onclick={() => handleCreate(suggestion)}
									class="btn-pill btn-pill-ghost btn-pill-xs opacity-0 group-hover:opacity-100"
									aria-label="Create task: {suggestion.title}"
								>
									+ Add
								</button>
								<button
									onclick={() => handleDismiss(suggestion)}
									class="btn-pill btn-pill-ghost btn-pill-icon opacity-0 group-hover:opacity-100"
									aria-label="Dismiss suggestion"
								>
									<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
									</svg>
								</button>
							</div>
						</div>
					</div>
				{/each}
			</div>

			<!-- Batch create button for compact mode -->
			{#if displaySuggestions.length > 1 && onCreateTask}
				<div style="background: var(--dbg2); border-color: var(--dbd)" class="px-3 py-2 border-t">
					<button
						onclick={handleCreateAll}
						disabled={creating}
						class="btn-pill btn-pill-ghost btn-pill-xs w-full"
					>
						{#if creating}
							Creating... ({createdCount}/{displaySuggestions.length})
						{:else}
							+ Create All ({displaySuggestions.length})
						{/if}
					</button>
				</div>
			{/if}
		{/if}
	</div>
{:else}
	<!-- Full Panel Mode -->
	<div class="space-y-6">
		<!-- Header -->
		<div class="flex items-center justify-between">
			<div>
				<h2 style="color: var(--dt1)" class="text-lg font-semibold flex items-center gap-2">
					<!-- Lightbulb icon -->
					<svg class="w-5 h-5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z" />
					</svg>
					Task Suggestions
				</h2>
				<p style="color: var(--dt2)" class="text-sm mt-0.5">
					AI-powered recommendations based on your calendar
				</p>
			</div>
			<div class="flex items-center gap-2">
				{#if displaySuggestions.length > 1 && onCreateTask}
					<button
						onclick={handleCreateAll}
						disabled={creating}
						class="btn-pill btn-pill-success btn-pill-sm flex items-center gap-1.5"
					>
						{#if creating}
							<svg class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
								<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
								<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
							</svg>
							Creating ({createdCount})
						{:else}
							<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
							</svg>
							Create All ({displaySuggestions.length})
						{/if}
					</button>
				{/if}
				<button
					onclick={loadSuggestions}
					disabled={loading}
					aria-label="Refresh suggestions"
					class="btn-pill btn-pill-ghost btn-pill-icon"
				>
					<svg class="w-5 h-5 {loading ? 'animate-spin' : ''}" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
					</svg>
				</button>
			</div>
		</div>

		{#if loading}
			<!-- Full skeleton loader -->
			<div class="space-y-4">
				{#each Array(3) as _, i}
					<div
						style="background: var(--dbg); border-color: var(--dbd)"
						class="border rounded-lg p-4 animate-pulse"
						transition:fade={{ duration: 150, delay: i * 100 }}
					>
						<div class="flex items-start gap-3">
							<div style="background: var(--dbg2)" class="w-10 h-10 rounded-full"></div>
							<div class="flex-1 space-y-3">
								<div class="flex gap-2">
									<div style="background: var(--dbg2)" class="h-5 rounded w-16"></div>
									<div style="background: var(--dbg2)" class="h-5 rounded w-20"></div>
								</div>
								<div style="background: var(--dbg2)" class="h-5 rounded w-3/4"></div>
								<div style="background: var(--dbg3)" class="h-4 rounded w-1/2"></div>
								<div style="background: var(--dbg3)" class="h-3 rounded w-1/3"></div>
							</div>
						</div>
					</div>
				{/each}
			</div>
		{:else if error}
			<div
				style="background: var(--bos-priority-critical-bg); border-color: var(--dbd)"
				class="p-4 border rounded-lg flex items-start gap-3"
				transition:fly={{ y: -10, duration: 200 }}
			>
				<svg style="color: var(--bos-status-error)" class="w-5 h-5 flex-shrink-0 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
				</svg>
				<div>
					<p style="color: var(--bos-priority-critical)" class="font-medium">Failed to load suggestions</p>
					<p style="color: var(--bos-priority-critical)" class="text-sm mt-1">{error}</p>
					<button onclick={loadSuggestions} class="btn-pill btn-pill-ghost btn-pill-sm mt-2 underline hover:no-underline">
						Try again
					</button>
				</div>
			</div>
		{:else if visibleSuggestions.length === 0}
			<div
				style="background: var(--dbg2); border-color: var(--dbd)"
				class="text-center py-16 rounded-xl border"
				transition:scale={{ duration: 200, start: 0.95 }}
			>
				<!-- Sparkles icon -->
				<svg class="w-12 h-12 mx-auto mb-4" style="color: var(--dt3)" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 3v4M3 5h4M6 17v4m-2-2h4m5-16l2.286 6.857L21 12l-5.714 2.143L13 21l-2.286-6.857L5 12l5.714-2.143L13 3z" />
				</svg>
				<h3 style="color: var(--dt1)" class="text-lg font-medium mb-2">All caught up!</h3>
				<p style="color: var(--dt2)" class="max-w-sm mx-auto">
					No task suggestions based on your recent calendar activity.
				</p>
				<div style="color: var(--dt3)" class="mt-6 text-sm space-y-1">
					<p>Suggestions appear when you have:</p>
					<ul class="list-none space-y-1">
						<li class="flex items-center justify-center gap-1.5">
							<!-- Calendar icon -->
							<svg class="w-3.5 h-3.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
							</svg>
							Upcoming meetings that need prep
						</li>
						<li class="flex items-center justify-center gap-1.5">
							<!-- Check icon -->
							<svg class="w-3.5 h-3.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
							</svg>
							Recent meetings with follow-ups
						</li>
					</ul>
				</div>
			</div>
		{:else}
			<!-- Analysis Info -->
			{#if response}
				<div
					style="color: var(--dt3)"
					class="text-xs flex items-center gap-4 px-2"
					transition:fade={{ duration: 150 }}
				>
					<span class="flex items-center gap-1">
						<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
						</svg>
						{response.events_analyzed} events analyzed
					</span>
					<span style="background: var(--dt3)" class="w-1 h-1 rounded-full"></span>
					<span>
						{new Date(response.analysis_period.start).toLocaleDateString('en-US', { month: 'short', day: 'numeric' })} –
						{new Date(response.analysis_period.end).toLocaleDateString('en-US', { month: 'short', day: 'numeric' })}
					</span>
				</div>
			{/if}

			<!-- Prep Tasks Section -->
			{#if prepSuggestions.length > 0}
				<div transition:fly={{ y: 10, duration: 200 }}>
					<h3 style="color: var(--dt2)" class="text-sm font-medium mb-3 flex items-center gap-2">
						<span style="background: var(--dbg2)" class="w-7 h-7 rounded-full flex items-center justify-center">
							<!-- Clipboard icon -->
							<svg class="w-4 h-4" style="color: var(--dt2)" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
							</svg>
						</span>
						<span>Prep Tasks</span>
						<span style="color: var(--dt3)" class="text-xs font-normal">— Before upcoming meetings</span>
						<span style="background: var(--dbg2); color: var(--dt2)" class="ml-auto text-xs px-2 py-0.5 rounded-full">
							{prepSuggestions.length}
						</span>
					</h3>
					<div class="space-y-3">
						{#each prepSuggestions as suggestion (getSuggestionId(suggestion))}
							<div
								style="background: var(--dbg); border-color: var(--dbd)"
								class="border rounded-xl p-4 hover:shadow-md transition-all duration-200 group"
								animate:flip={{ duration: 200 }}
								transition:fly={{ x: -20, duration: 200 }}
							>
								<div class="flex items-start justify-between gap-4">
									<div class="flex-1 min-w-0">
										<div class="flex items-center gap-2 mb-2 flex-wrap">
											<span
												style="{getPriorityStyle(suggestion.priority)}"
												class="px-2.5 py-1 text-xs font-medium rounded-full border flex items-center gap-1"
											>
												{suggestion.priority}
											</span>
											<span
												style="{formatDueDateStyle(suggestion.suggested_due_date)}"
												class="text-xs px-2 py-1 rounded-full"
											>
												Due: {formatDueDate(suggestion.suggested_due_date)}
											</span>
										</div>
										<h4 style="color: var(--dt1)" class="font-semibold text-base">{suggestion.title}</h4>
										<p style="color: var(--dt2)" class="text-sm mt-1.5 leading-relaxed">{suggestion.description}</p>
										<div style="color: var(--dt3)" class="mt-3 flex items-center gap-3 text-xs">
											<span class="flex items-center gap-1">
												<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
													<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
												</svg>
												{suggestion.related_event_title}
											</span>
										</div>
										<!-- Confidence Bar -->
										<div class="mt-3 flex items-center gap-2">
											<span style="color: var(--dt3)" class="text-xs w-20">Confidence:</span>
											<div style="background: var(--dbg2)" class="flex-1 h-2 rounded-full overflow-hidden max-w-[120px]">
												<div
													class="h-full rounded-full transition-all duration-500"
													style="{getConfidenceBarStyle(suggestion.confidence)}; width: {getConfidenceWidth(suggestion.confidence)}"
												></div>
											</div>
											<span style="color: var(--dt2)" class="text-xs font-medium w-10 text-right">
												{Math.round(suggestion.confidence * 100)}%
											</span>
										</div>
									</div>
									<div class="flex flex-col gap-2">
										<button
											onclick={() => handleCreate(suggestion)}
											class="btn-pill btn-pill-primary btn-pill-sm flex items-center gap-1.5"
										>
											<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
												<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
											</svg>
											Create
										</button>
										<button
											onclick={() => handleDismiss(suggestion)}
											class="btn-pill btn-pill-ghost btn-pill-sm opacity-0 group-hover:opacity-100"
										>
											Dismiss
										</button>
									</div>
								</div>
							</div>
						{/each}
					</div>
				</div>
			{/if}

			<!-- Follow-up Tasks Section -->
			{#if followUpSuggestions.length > 0}
				<div transition:fly={{ y: 10, duration: 200, delay: 100 }}>
					<h3 style="color: var(--dt2)" class="text-sm font-medium mb-3 flex items-center gap-2">
						<span style="background: var(--dbg2)" class="w-7 h-7 rounded-full flex items-center justify-center">
							<!-- Send/arrow icon for follow-up -->
							<svg class="w-4 h-4" style="color: var(--dt2)" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8" />
							</svg>
						</span>
						<span>Follow-up Tasks</span>
						<span style="color: var(--dt3)" class="text-xs font-normal">— After recent meetings</span>
						<span style="background: var(--dbg2); color: var(--dt2)" class="ml-auto text-xs px-2 py-0.5 rounded-full">
							{followUpSuggestions.length}
						</span>
					</h3>
					<div class="space-y-3">
						{#each followUpSuggestions as suggestion (getSuggestionId(suggestion))}
							<div
								style="background: var(--dbg); border-color: var(--dbd)"
								class="border rounded-xl p-4 hover:shadow-md transition-all duration-200 group"
								animate:flip={{ duration: 200 }}
								transition:fly={{ x: -20, duration: 200 }}
							>
								<div class="flex items-start justify-between gap-4">
									<div class="flex-1 min-w-0">
										<div class="flex items-center gap-2 mb-2 flex-wrap">
											<span
												style="{getPriorityStyle(suggestion.priority)}"
												class="px-2.5 py-1 text-xs font-medium rounded-full border flex items-center gap-1"
											>
												{suggestion.priority}
											</span>
											<span
												style="{formatDueDateStyle(suggestion.suggested_due_date)}"
												class="text-xs px-2 py-1 rounded-full"
											>
												Due: {formatDueDate(suggestion.suggested_due_date)}
											</span>
										</div>
										<h4 style="color: var(--dt1)" class="font-semibold text-base">{suggestion.title}</h4>
										<p style="color: var(--dt2)" class="text-sm mt-1.5 leading-relaxed">{suggestion.description}</p>
										<div style="color: var(--dt3)" class="mt-3 flex items-center gap-3 text-xs">
											<span class="flex items-center gap-1">
												<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
													<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
												</svg>
												{suggestion.related_event_title}
											</span>
										</div>
										<!-- Confidence Bar -->
										<div class="mt-3 flex items-center gap-2">
											<span style="color: var(--dt3)" class="text-xs w-20">Confidence:</span>
											<div style="background: var(--dbg2)" class="flex-1 h-2 rounded-full overflow-hidden max-w-[120px]">
												<div
													class="h-full rounded-full transition-all duration-500"
													style="{getConfidenceBarStyle(suggestion.confidence)}; width: {getConfidenceWidth(suggestion.confidence)}"
												></div>
											</div>
											<span style="color: var(--dt2)" class="text-xs font-medium w-10 text-right">
												{Math.round(suggestion.confidence * 100)}%
											</span>
										</div>
									</div>
									<div class="flex flex-col gap-2">
										<button
											onclick={() => handleCreate(suggestion)}
											class="btn-pill btn-pill-primary btn-pill-sm flex items-center gap-1.5"
										>
											<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
												<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
											</svg>
											Create
										</button>
										<button
											onclick={() => handleDismiss(suggestion)}
											class="btn-pill btn-pill-ghost btn-pill-sm opacity-0 group-hover:opacity-100"
										>
											Dismiss
										</button>
									</div>
								</div>
							</div>
						{/each}
					</div>
				</div>
			{/if}
		{/if}
	</div>
{/if}

<!-- Undo Snackbar -->
{#if lastDismissed}
	<div
		class="fixed bottom-4 left-1/2 -translate-x-1/2 z-50"
		transition:fly={{ y: 20, duration: 200 }}
	>
		<div style="background: var(--dt1); color: var(--dbg)" class="flex items-center gap-3 px-4 py-3 rounded-lg shadow-lg">
			<span class="text-sm">Suggestion dismissed</span>
			<button
				onclick={handleUndo}
				class="btn-pill btn-pill-ghost btn-pill-sm flex items-center gap-1"
			>
				Undo
				<span style="color: var(--dt3)" class="text-xs">({undoCountdown}s)</span>
			</button>
		</div>
	</div>
{/if}

<!-- Success Toast for batch creation -->
{#if createdCount > 0 && !creating}
	<div
		class="fixed bottom-4 right-4 z-50"
		transition:fly={{ y: 20, duration: 200 }}
	>
		<div style="background: var(--bos-status-success)" class="flex items-center gap-2 px-4 py-3 text-white rounded-lg shadow-lg">
			<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
			</svg>
			<span class="text-sm font-medium">Created {createdCount} task(s)</span>
		</div>
	</div>
{/if}
