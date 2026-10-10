<script lang="ts">
	import type { CalendarEvent, MeetingType } from '$lib/api/calendar';

	interface Props {
		event: CalendarEvent;
		compact?: boolean;
		onClick?: () => void;
	}

	let { event, compact = false, onClick }: Props = $props();

	function formatTime(dateString: string): string {
		return new Date(dateString).toLocaleTimeString('en-US', {
			hour: 'numeric',
			minute: '2-digit',
			hour12: true
		});
	}

	function getMeetingTypeStyle(type: string): { bg: string; text: string; border: string } {
		const styles: Record<string, { bg: string; text: string; border: string }> = {
			team: { bg: 'var(--dbg2)', text: 'var(--dt)', border: 'var(--dbd)' },
			sales: { bg: 'var(--dbg2)', text: 'var(--dt)', border: 'var(--dbd)' },
			client: { bg: 'var(--dbg2)', text: 'var(--dt)', border: 'var(--dbd)' },
			onboarding: { bg: 'var(--dbg3)', text: 'var(--dt)', border: 'var(--dbd)' },
			kickoff: { bg: 'var(--dbg3)', text: 'var(--dt)', border: 'var(--dbd)' },
			implementation: { bg: 'var(--dbg2)', text: 'var(--dt2)', border: 'var(--dbd2)' },
			standup: { bg: 'var(--dbg2)', text: 'var(--dt2)', border: 'var(--dbd2)' },
			planning: { bg: 'var(--dbg3)', text: 'var(--dt2)', border: 'var(--dbd)' },
			review: { bg: 'var(--dbg2)', text: 'var(--dt)', border: 'var(--dbd)' },
			one_on_one: { bg: 'var(--dbg3)', text: 'var(--dt)', border: 'var(--dbd)' },
			retrospective: { bg: 'var(--dbg2)', text: 'var(--dt2)', border: 'var(--dbd2)' },
			internal: { bg: 'var(--dbg2)', text: 'var(--dt2)', border: 'var(--dbd2)' },
			external: { bg: 'var(--dbg3)', text: 'var(--dt)', border: 'var(--dbd)' },
			other: { bg: 'var(--dbg2)', text: 'var(--dt3)', border: 'var(--dbd2)' }
		};
		return styles[type] || styles.other;
	}

	const style = $derived(getMeetingTypeStyle(event.meeting_type));
</script>

{#if compact}
	<button
		class="cec-card cec-card--compact btn-pill btn-pill-soft btn-pill-xs w-full text-left truncate"
		style="background: {style.bg}; color: {style.text};"
		onclick={onClick}
	>
		{event.all_day ? '' : formatTime(event.start_time) + ' '}{event.title || 'Untitled'}
	</button>
{:else}
	<button
		class="cec-card btn-pill btn-pill-soft w-full text-left p-3 border"
		style="background: {style.bg}; color: {style.text}; border-color: {style.border};"
		onclick={onClick}
	>
		<div class="flex items-start justify-between gap-2">
			<div class="flex-1 min-w-0">
				<p class="font-medium truncate">
					{event.title || 'Untitled Event'}
				</p>
				<p class="text-sm opacity-75">
					{#if event.all_day}
						All day
					{:else}
						{formatTime(event.start_time)} - {formatTime(event.end_time)}
					{/if}
				</p>
				{#if event.location}
					<p class="text-xs mt-1 opacity-60 truncate flex items-center gap-1">
						<svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z" />
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 11a3 3 0 11-6 0 3 3 0 016 0z" />
						</svg>
						{event.location}
					</p>
				{/if}
			</div>
			{#if event.source === 'google'}
				<svg class="w-4 h-4 shrink-0 opacity-50" viewBox="0 0 24 24">
					<path fill="#4285F4" d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"/>
					<path fill="#34A853" d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"/>
					<path fill="#FBBC05" d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"/>
					<path fill="#EA4335" d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"/>
				</svg>
			{/if}
		</div>
		{#if event.attendees && event.attendees.length > 0}
			<div class="mt-2 flex items-center gap-1">
				<svg class="w-3 h-3 opacity-50" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z" />
				</svg>
				<span class="text-xs opacity-60">
					{event.attendees.length} attendee{event.attendees.length > 1 ? 's' : ''}
				</span>
			</div>
		{/if}
	</button>
{/if}

<style>
	.cec-card {
		transition: transform 150ms cubic-bezier(0.4, 0, 0.2, 1),
			box-shadow 150ms cubic-bezier(0.4, 0, 0.2, 1),
			filter 150ms ease;
	}

	.cec-card:hover {
		transform: translateY(-1px);
		filter: brightness(0.97);
		box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08), 0 1px 3px rgba(0, 0, 0, 0.04);
	}

	.cec-card--compact:hover {
		transform: translateY(-1px) scale(1.01);
	}

	.cec-card:focus-visible {
		outline: none;
		box-shadow: 0 0 0 2px var(--bos-accent-blue);
	}

	@media (prefers-reduced-motion: reduce) {
		.cec-card {
			transition: none;
		}
		.cec-card:hover {
			transform: none;
		}
	}
</style>
