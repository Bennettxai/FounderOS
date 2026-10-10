<script lang="ts">
	import { getUpcomingEvents } from '$lib/api/calendar';
	import type { CalendarEvent } from '$lib/api/calendar';
	import { onMount } from 'svelte';
	import { getEventDisplayStart } from './calendarUtils';

	interface Props {
		limit?: number;
		showHeader?: boolean;
		onEventClick?: (event: CalendarEvent) => void;
	}

	let { limit = 10, showHeader = true, onEventClick }: Props = $props();

	let events = $state<CalendarEvent[]>([]);
	let isLoading = $state(true);
	let error = $state<string | null>(null);

	onMount(async () => {
		await loadEvents();
	});

	async function loadEvents() {
		try {
			events = await getUpcomingEvents(limit);
		} catch (err) {
			console.error('Error loading upcoming events:', err);
			error = 'Failed to load events';
		} finally {
			isLoading = false;
		}
	}

	function formatTime(dateString: string): string {
		return new Date(dateString).toLocaleTimeString('en-US', {
			hour: 'numeric',
			minute: '2-digit',
			hour12: true
		});
	}

	function getMeetingTypeColor(type: string): string {
		const colors: Record<string, string> = {
			team: 'border-l-blue-500',
			sales: 'border-l-green-500',
			onboarding: 'border-l-purple-500',
			kickoff: 'border-l-orange-500',
			implementation: 'border-l-cyan-500',
			standup: 'border-l-indigo-500',
			retrospective: 'border-l-pink-500',
			planning: 'border-l-yellow-500',
			review: 'border-l-teal-500',
			one_on_one: 'border-l-rose-500',
			client: 'border-l-emerald-500',
			internal: 'border-l-slate-500',
			external: 'border-l-amber-500',
			other: 'border-l-gray-400'
		};
		return colors[type] || colors.other;
	}

	// Group events by date
	const groupedEvents = $derived(() => {
		const groups: Record<string, CalendarEvent[]> = {};
		for (const event of events) {
			const dateKey = getEventDisplayStart(event).toDateString();
			if (!groups[dateKey]) {
				groups[dateKey] = [];
			}
			groups[dateKey].push(event);
		}
		return groups;
	});
</script>

<div class="card">
	{#if showHeader}
		<h2 class="text-lg font-medium mb-4" style="color: var(--dt)">Upcoming Events</h2>
	{/if}

	{#if isLoading}
		<div class="flex items-center justify-center py-8">
			<div class="animate-spin h-6 w-6 border-2 border-t-transparent rounded-full" style="border-color: var(--dt)"></div>
		</div>
	{:else if error}
		<div class="text-center py-6" style="color: var(--bos-status-error-text)">
			<p class="text-sm">{error}</p>
		</div>
	{:else if events.length === 0}
		<div class="text-center py-6">
			<svg class="w-12 h-12 mx-auto mb-3" style="color: var(--dt4)" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
			</svg>
			<p class="text-sm" style="color: var(--dt3)">No upcoming events</p>
		</div>
	{:else}
		<div class="space-y-4">
			{#each Object.entries(groupedEvents()) as [dateKey, dateEvents]}
				<div>
					<h3 class="text-xs font-semibold uppercase tracking-wider mb-2" style="color: var(--dt3)">
						{getEventDisplayStart(dateEvents[0]).toLocaleDateString('en-US', {
							weekday: 'short',
							month: 'short',
							day: 'numeric'
						})}
					</h3>
					<div class="space-y-2">
						{#each dateEvents as event (event.id)}
							<button
								class="btn-pill btn-pill-soft w-full text-left p-3 border-l-4 {getMeetingTypeColor(event.meeting_type)}"
								onclick={() => onEventClick?.(event)}
							>
								<div class="flex items-start justify-between gap-2">
									<p class="font-medium truncate" style="color: var(--dt)">
										{event.title || 'Untitled Event'}
									</p>
									<span class="text-xs shrink-0" style="color: var(--dt3)">
										{event.all_day ? 'All day' : formatTime(event.start_time)}
									</span>
								</div>
								{#if event.location}
									<p class="text-xs mt-1 truncate" style="color: var(--dt4)">
										{event.location}
									</p>
								{/if}
							</button>
						{/each}
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>
