<script lang="ts">
	import type { CalendarEvent } from '$lib/api';
	import { getEventDisplayStart, type SyncStats } from './calendarUtils';

	interface Props {
		events: CalendarEvent[];
		syncStats: SyncStats | null;
		onOpenEventModal: (event: CalendarEvent) => void;
		onJumpToFirstEvent: () => void;
	}

	let { events, syncStats, onOpenEventModal, onJumpToFirstEvent }: Props = $props();

	const sortedEvents = $derived(
		[...events].sort((a, b) => getEventDisplayStart(a).getTime() - getEventDisplayStart(b).getTime())
	);
</script>

<div class="p-6">
	{#if events.length === 0}
		<div class="text-center py-12">
			<svg class="mx-auto h-12 w-12" style="color: var(--dt4)" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
			</svg>
			<h3 class="mt-2 text-sm font-semibold" style="color: var(--dt)">No events in this period</h3>
			<p class="mt-1 text-sm" style="color: var(--dt3)">
				{#if syncStats && syncStats.totalEvents > 0}
					You have {syncStats.totalEvents} events synced from
					{syncStats.dateRange?.from
						? new Date(syncStats.dateRange.from).toLocaleDateString()
						: 'N/A'} to
					{syncStats.dateRange?.to
						? new Date(syncStats.dateRange.to).toLocaleDateString()
						: 'N/A'}.
					<button onclick={onJumpToFirstEvent} class="btn-pill btn-pill-ghost btn-pill-sm ml-1">
						Jump to events
					</button>
				{:else}
					No events found. Create one or sync from Google Calendar.
				{/if}
			</p>
		</div>
	{:else}
		<div class="space-y-3">
			{#each sortedEvents as event (event.id)}
				<button
					type="button"
					onclick={() => onOpenEventModal(event)}
					class="cav-row w-full text-left p-4 rounded-xl border transition-all"
				>
					<div class="flex items-start gap-4">
						<!-- Date/Time column -->
						<div class="flex-shrink-0 text-center w-16">
							<p class="text-xs uppercase" style="color: var(--dt3)">
								{getEventDisplayStart(event).toLocaleDateString('en-US', { weekday: 'short' })}
							</p>
							<p class="text-2xl font-bold" style="color: var(--dt)">{getEventDisplayStart(event).getDate()}</p>
							<p class="text-xs" style="color: var(--dt3)">
								{getEventDisplayStart(event).toLocaleDateString('en-US', { month: 'short' })}
							</p>
						</div>

						<!-- Event Details -->
						<div class="flex-1 min-w-0">
							<div class="flex items-center gap-2">
								<h3 class="text-base font-semibold truncate" style="color: var(--dt)">
									{event.title || 'Untitled Event'}
								</h3>
								{#if event.source === 'google'}
									<svg class="w-4 h-4 flex-shrink-0" viewBox="0 0 24 24" aria-label="Synced from Google Calendar">
										<title>Synced from Google Calendar</title>
										<path fill="#4285F4" d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"/>
										<path fill="#34A853" d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"/>
										<path fill="#FBBC05" d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"/>
										<path fill="#EA4335" d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"/>
									</svg>
								{/if}
								{#if event.meeting_type && event.meeting_type !== 'other'}
									<span class="inline-block px-2 py-0.5 text-xs font-medium rounded-full" style="background: var(--dbg2); color: var(--dt2)">
										{event.meeting_type.replace('_', ' ')}
									</span>
								{/if}
							</div>

							<p class="text-sm mt-1" style="color: var(--dt3)">
								{#if event.all_day}
									All day
								{:else}
									{new Date(event.start_time).toLocaleTimeString('en-US', {
										hour: 'numeric',
										minute: '2-digit'
									})} - {new Date(event.end_time).toLocaleTimeString('en-US', {
										hour: 'numeric',
										minute: '2-digit'
									})}
								{/if}
							</p>

							{#if event.location}
								<p class="text-sm mt-1 flex items-center gap-1" style="color: var(--dt3)">
									<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z" />
									</svg>
									{event.location}
								</p>
							{/if}

							{#if event.description}
								<p class="text-sm mt-1 line-clamp-2" style="color: var(--dt4)">{event.description}</p>
							{/if}
						</div>

						<!-- Attendees -->
						{#if event.attendees && event.attendees.length > 0}
							<div class="flex-shrink-0 flex -space-x-2">
								{#each event.attendees.slice(0, 3) as attendee}
									<div
										class="w-8 h-8 rounded-full border-2 flex items-center justify-center text-xs font-medium"
										style="background: var(--dbg3); color: var(--dt2); border-color: var(--dbg);"
										title={attendee.email}
									>
										{(attendee.name || attendee.email || '?').charAt(0).toUpperCase()}
									</div>
								{/each}
								{#if event.attendees.length > 3}
									<div
										class="w-8 h-8 rounded-full border-2 flex items-center justify-center text-xs font-medium"
										style="background: var(--dbg2); color: var(--dt3); border-color: var(--dbg);"
									>
										+{event.attendees.length - 3}
									</div>
								{/if}
							</div>
						{/if}
					</div>
				</button>
			{/each}
		</div>
	{/if}
</div>

<style>
	.cav-row {
		background: var(--dbg);
		border-color: var(--dbd2);
	}

	.cav-row:hover {
		background: var(--dbg2);
		border-color: var(--dbd);
		transform: translateY(-1px);
		box-shadow: 0 4px 14px rgba(0, 0, 0, 0.06);
	}

	.cav-row:focus-visible {
		outline: none;
		border-color: var(--bos-accent-blue);
		box-shadow: 0 0 0 2px rgba(var(--bos-accent-blue-rgb), 0.25);
	}
</style>
