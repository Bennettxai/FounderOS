<script lang="ts">
	import type { CalendarEvent } from '$lib/api/calendar';
	import type { GoogleConnectionStatus } from '$lib/api/integrations';
	import { getTodayEvents } from '$lib/api/calendar';
	import { getGoogleConnectionStatus } from '$lib/api/integrations';
	import { onMount } from 'svelte';

	interface Props {
		maxEvents?: number;
		showHeader?: boolean;
		onViewAll?: () => void;
	}

	let { maxEvents = 5, showHeader = true, onViewAll }: Props = $props();

	let events = $state<CalendarEvent[]>([]);
	let isLoading = $state(true);
	let error = $state<string | null>(null);
	let isConnected = $state(false);

	onMount(async () => {
		await checkConnectionAndLoadEvents();
	});

	async function checkConnectionAndLoadEvents() {
		try {
			// Always load native events — the calendar base works with no integration.
			const todayEvents = await getTodayEvents();
			events = todayEvents.slice(0, maxEvents);
			// Google connection is optional context (for an offer-to-sync hint), not a gate.
			try {
				const status = await getGoogleConnectionStatus();
				isConnected = status.connected;
			} catch {
				isConnected = false;
			}
		} catch (err) {
			console.error('Error loading calendar events:', err);
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

	function formatTimeRange(start: string, end: string, allDay: boolean): string {
		if (allDay) return 'All day';
		return `${formatTime(start)} - ${formatTime(end)}`;
	}

	function getMeetingTypeStyle(): string {
		return 'background: var(--dbg2); color: var(--dt2)';
	}

	function isEventNow(event: CalendarEvent): boolean {
		const now = new Date();
		const start = new Date(event.start_time);
		const end = new Date(event.end_time);
		return now >= start && now <= end;
	}
</script>

<div class="card">
	{#if showHeader}
		<div class="flex items-center justify-between mb-4">
			<h2 class="text-lg font-medium" style="color: var(--dt)">Today's Schedule</h2>
			{#if onViewAll}
				<button
					onclick={onViewAll}
					class="btn-pill btn-pill-ghost btn-pill-sm"
				>
					View all
				</button>
			{/if}
		</div>
	{/if}

	{#if isLoading}
		<div class="flex items-center justify-center py-8">
			<div class="animate-spin h-6 w-6 border-2 border-t-transparent rounded-full" style="border-color: var(--dt); border-top-color: transparent"></div>
		</div>
	{:else if error}
		<div class="text-center py-6 text-red-500">
			<p class="text-sm">{error}</p>
		</div>
	{:else if events.length === 0}
		<div class="text-center py-6">
			<svg class="w-12 h-12 mx-auto mb-3" style="color: var(--dt4)" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
			</svg>
			<p class="text-sm" style="color: var(--dt3)">No events scheduled for today</p>
			{#if !isConnected}
				<a href="/settings?tab=integrations" class="text-xs font-medium mt-2 inline-block" style="color: var(--bos-nav-active)">
					Connect Google Calendar to sync
				</a>
			{/if}
		</div>
	{:else}
		<div class="space-y-3">
			{#each events as event (event.id)}
				<div
					class="flex items-start gap-3 p-3 rounded-lg transition-colors"
					style="{isEventNow(event) ? 'background: var(--bos-nav-active-bg); border: 1px solid var(--dbd)' : 'background: var(--dbg2)'}"
				>
					<div
						class="flex-shrink-0 w-1 h-full min-h-[40px] rounded-full"
						style="{isEventNow(event) ? 'background: var(--bos-nav-active)' : 'background: var(--dbd)'}"
					></div>
					<div class="flex-1 min-w-0">
						<div class="flex items-start justify-between gap-2">
							<p class="font-medium truncate" style="color: var(--dt)">
								{event.title || 'Untitled Event'}
							</p>
							{#if isEventNow(event)}
								<span class="flex-shrink-0 px-1.5 py-0.5 text-xs font-medium rounded" style="background: var(--bos-nav-active); color: var(--bos-surface-on-color)">
									Now
								</span>
							{/if}
						</div>
						<p class="text-sm" style="color: var(--dt3)">
							{formatTimeRange(event.start_time, event.end_time, event.all_day)}
						</p>
						{#if event.location}
							<p class="text-xs mt-1 truncate" style="color: var(--dt4)">
								{event.location}
							</p>
						{/if}
						{#if event.meeting_type && event.meeting_type !== 'other'}
							<span class="inline-block mt-1.5 px-2 py-0.5 text-xs font-medium rounded-full" style="{getMeetingTypeStyle()}">
								{event.meeting_type.replace('_', ' ')}
							</span>
						{/if}
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>
