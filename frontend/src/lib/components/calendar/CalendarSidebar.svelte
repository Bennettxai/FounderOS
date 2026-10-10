<script lang="ts">
	import type { CalendarEvent } from '$lib/api';
	import {
		isToday,
		getEventColors,
		getEventColor,
		getEventsForDate,
		getTimedEventSegment
	} from './calendarUtils';

	interface Props {
		selectedDay: Date;
		events: CalendarEvent[];
		onSelectDay: (date: Date) => void;
		onGoToDayView: (date: Date) => void;
		onOpenCreateModal: (date: Date) => void;
		onOpenEventModal: (event: CalendarEvent) => void;
	}

	let {
		selectedDay = $bindable(),
		events,
		onSelectDay,
		onGoToDayView,
		onOpenCreateModal,
		onOpenEventModal
	}: Props = $props();

	const selectedDayEvents = $derived(
		getEventsForDate(events, selectedDay).sort((a, b) => {
			const aStart = getTimedEventSegment(a, selectedDay)?.start.getTime() ?? 0;
			const bStart = getTimedEventSegment(b, selectedDay)?.start.getTime() ?? 0;
			return aStart - bStart;
		})
	);

	const isSelectedDayToday = $derived(isToday(selectedDay));

	const miniCalDaysInMonth = $derived(
		new Date(selectedDay.getFullYear(), selectedDay.getMonth() + 1, 0).getDate()
	);
	const miniCalFirstDayOffset = $derived(
		new Date(selectedDay.getFullYear(), selectedDay.getMonth(), 1).getDay()
	);

	function hasEventsOnDate(date: Date): boolean {
		return getEventsForDate(events, date).length > 0;
	}

	function eventTimeForSelectedDay(event: CalendarEvent): string {
		const start = getTimedEventSegment(event, selectedDay)?.start ?? new Date(event.start_time);
		return start.toLocaleTimeString('en-US', { hour: 'numeric', minute: '2-digit' });
	}

	function prevMonth() {
		const d = new Date(selectedDay);
		d.setMonth(d.getMonth() - 1);
		selectedDay = d;
	}

	function nextMonth() {
		const d = new Date(selectedDay);
		d.setMonth(d.getMonth() + 1);
		selectedDay = d;
	}
</script>

<div class="csb">
	<!-- Mini Calendar Navigator -->
	<div class="csb-cal">
		<div class="csb-cal__header">
			<h3 class="csb-cal__month">
				{selectedDay.toLocaleString('default', { month: 'long', year: 'numeric' })}
			</h3>
			<div class="csb-cal__nav">
				<button onclick={prevMonth} class="csb-cal__nav-btn" aria-label="Previous month">
					<svg width="14" height="14" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
					</svg>
				</button>
				<button onclick={nextMonth} class="csb-cal__nav-btn" aria-label="Next month">
					<svg width="14" height="14" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
					</svg>
				</button>
			</div>
		</div>

		<!-- Mini Month Grid -->
		<div class="csb-grid">
			{#each ['S', 'M', 'T', 'W', 'T', 'F', 'S'] as dayLabel}
				<div class="csb-grid__label">{dayLabel}</div>
			{/each}
			{#each Array(miniCalFirstDayOffset) as _, i (i)}
				<div class="csb-grid__empty"></div>
			{/each}
			{#each Array(miniCalDaysInMonth) as _, i (i)}
				{@const day = i + 1}
				{@const date = new Date(selectedDay.getFullYear(), selectedDay.getMonth(), day)}
				{@const todayDate = isToday(date)}
				{@const isSelected =
					selectedDay.getDate() === day && selectedDay.getMonth() === date.getMonth()}
				{@const dotVisible = hasEventsOnDate(date) && !todayDate}
				<button
					onclick={() => { onSelectDay(date); onGoToDayView(date); }}
					class="csb-grid__day"
					class:csb-grid__day--today={todayDate}
					class:csb-grid__day--selected={isSelected && !todayDate}
				>
					{day}
					{#if dotVisible}
						<span class="csb-grid__dot"></span>
					{/if}
				</button>
			{/each}
		</div>
	</div>

	<!-- Divider -->
	<div class="csb-divider"></div>

	<!-- Daily Agenda -->
	<div class="csb-agenda">
		<div class="csb-agenda__header">
			<h3 class="csb-agenda__title">
				{isSelectedDayToday
					? "Today's Agenda"
					: selectedDay.toLocaleDateString('en-US', {
							weekday: 'short',
							month: 'short',
							day: 'numeric'
					  })}
			</h3>
			<span class="csb-agenda__count">
				{selectedDayEvents.length}
			</span>
		</div>

		{#if selectedDayEvents.length === 0}
			<div class="csb-agenda__empty">
				<svg class="csb-agenda__empty-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
				</svg>
				<p class="csb-agenda__empty-text">No events scheduled</p>
				<button
					onclick={() => onOpenCreateModal(selectedDay)}
					class="csb-agenda__add-btn"
					aria-label="Add event"
				>
					<svg width="11" height="11" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M12 5v14M5 12h14" />
					</svg>
					Add event
				</button>
			</div>
		{:else}
			<div class="csb-agenda__list">
				{#each selectedDayEvents as event (event.id)}
					{@const colors = getEventColors(event)}
					{@const accentColor = getEventColor(event)}
					<button
						onclick={() => onOpenEventModal(event)}
						class="csb-agenda__event {colors.bg}"
						style="border-left: 3px solid {accentColor};"
					>
						<p class="csb-agenda__event-time {colors.text}">
							{#if event.all_day}
								All day
							{:else}
								{eventTimeForSelectedDay(event)}
							{/if}
						</p>
						<p class="csb-agenda__event-title">
							{event.title || 'Untitled'}
						</p>
						{#if event.location}
							<p class="csb-agenda__event-loc">{event.location}</p>
						{/if}
					</button>
				{/each}
			</div>
		{/if}
	</div>
</div>

<style>
	/* ── Calendar Sidebar ────────────────────────────────────────── */
	.csb {
		width: 14rem;
		min-width: 14rem;
		border-right: 1px solid var(--dbd2);
		display: flex;
		flex-direction: column;
		background: var(--dbg2);
		flex-shrink: 0;
		height: 100%;
	}

	/* Mini Calendar */
	.csb-cal {
		padding: 0.75rem;
	}

	.csb-cal__header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-bottom: 0.5rem;
	}

	.csb-cal__month {
		font-size: 0.75rem;
		font-weight: 600;
		color: var(--dt);
		margin: 0;
		letter-spacing: -0.01em;
	}

	.csb-cal__nav {
		display: flex;
		align-items: center;
		gap: 0.15rem;
	}

	.csb-cal__nav-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 26px;
		height: 26px;
		min-width: 26px;
		padding: 0.25rem;
		border: none;
		border-radius: 6px;
		background: transparent;
		color: var(--dt3);
		cursor: pointer;
		transition: background 0.12s, color 0.12s;
	}

	.csb-cal__nav-btn:hover {
		background: var(--dbg3);
		color: var(--dt);
	}

	/* Mini calendar grid */
	.csb-grid {
		display: grid;
		grid-template-columns: repeat(7, 1fr);
		gap: 2px;
		text-align: center;
	}

	.csb-grid__label {
		font-size: 0.62rem;
		font-weight: 600;
		color: var(--dt3);
		padding: 0.2rem 0;
		padding-bottom: 0.3rem;
	}

	.csb-grid__empty {
		width: 100%;
		aspect-ratio: 1;
	}

	.csb-grid__day {
		position: relative;
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 0.7rem;
		color: var(--dt2);
		border-radius: 6px;
		cursor: pointer;
		border: none;
		background: transparent;
		padding: 0.15rem;
		aspect-ratio: 1;
		transition: background 0.12s, color 0.12s;
		line-height: 1;
	}

	.csb-grid__day:hover {
		background: var(--dbg3);
		color: var(--dt);
	}

	.csb-grid__day--today {
		background: var(--bos-accent-blue);
		color: #ffffff;
		font-weight: 700;
		box-shadow:
			0 0 0 2px rgba(var(--bos-accent-blue-rgb), 0.18),
			0 1px 4px rgba(var(--bos-accent-blue-rgb), 0.3);
	}

	.csb-grid__day--today:hover {
		background: var(--bos-accent-blue);
		color: #ffffff;
		filter: brightness(1.05);
	}

	.csb-grid__day--selected {
		background: var(--bos-nav-active-bg);
		color: var(--bos-nav-active);
		font-weight: 700;
		box-shadow: inset 0 0 0 1px var(--bos-nav-active);
	}

	.csb-grid__dot {
		position: absolute;
		bottom: 2px;
		left: 50%;
		transform: translateX(-50%);
		width: 4px;
		height: 4px;
		background: var(--dt3);
		border-radius: 50%;
	}

	/* Section divider */
	.csb-divider {
		height: 1px;
		background: var(--dbd2);
		margin: 0 0.75rem;
		flex-shrink: 0;
	}

	/* Daily Agenda */
	.csb-agenda {
		flex: 1;
		overflow: auto;
		padding: 0.65rem 0.75rem 0.75rem;
	}

	.csb-agenda__header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-bottom: 0.5rem;
	}

	.csb-agenda__title {
		font-size: 0.72rem;
		font-weight: 600;
		color: var(--dt);
		margin: 0;
		letter-spacing: -0.01em;
	}

	.csb-agenda__count {
		font-size: 0.6rem;
		font-weight: 600;
		color: var(--dt3);
		background: var(--dbg3);
		border: 1px solid var(--dbd2);
		padding: 0.05rem 0.38rem;
		border-radius: 999px;
		min-width: 1.25rem;
		text-align: center;
	}

	/* Empty state */
	.csb-agenda__empty {
		display: flex;
		flex-direction: column;
		align-items: center;
		text-align: center;
		padding: 1rem 0.5rem 0.75rem;
		gap: 0.35rem;
	}

	.csb-agenda__empty-icon {
		width: 1.6rem;
		height: 1.6rem;
		color: var(--dt4);
		flex-shrink: 0;
	}

	.csb-agenda__empty-text {
		font-size: 0.7rem;
		color: var(--dt3);
		margin: 0;
	}

	.csb-agenda__add-btn {
		display: inline-flex;
		align-items: center;
		gap: 0.3rem;
		margin-top: 0.15rem;
		padding: 0.3rem 0.65rem;
		font-size: 0.68rem;
		font-weight: 500;
		color: var(--dt2);
		background: transparent;
		border: 1.5px dashed var(--dbd);
		border-radius: 6px;
		cursor: pointer;
		transition: border-color 0.14s, color 0.14s, background 0.14s;
		white-space: nowrap;
	}

	.csb-agenda__add-btn:hover {
		border-color: var(--dt3);
		color: var(--dt);
		background: var(--dbg3);
	}

	/* Event list */
	.csb-agenda__list {
		display: flex;
		flex-direction: column;
		gap: 0.3rem;
	}

	.csb-agenda__event {
		width: 100%;
		text-align: left;
		padding: 0.42rem 0.55rem;
		border-radius: 7px;
		cursor: pointer;
		border: 1px solid transparent;
		transition: filter 0.12s, transform 0.1s, box-shadow 0.12s;
	}

	.csb-agenda__event:hover {
		filter: brightness(0.94);
		transform: translateX(1px);
		box-shadow: 0 1px 4px rgba(0, 0, 0, 0.08);
	}

	.csb-agenda__event-time {
		font-size: 0.62rem;
		font-weight: 600;
		margin: 0;
		letter-spacing: 0.01em;
	}

	.csb-agenda__event-title {
		font-size: 0.72rem;
		font-weight: 500;
		color: var(--dt);
		margin: 0.1rem 0 0;
		display: -webkit-box;
		-webkit-line-clamp: 1;
		line-clamp: 1;
		-webkit-box-orient: vertical;
		overflow: hidden;
	}

	.csb-agenda__event-loc {
		font-size: 0.62rem;
		color: var(--dt3);
		margin: 0.1rem 0 0;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
</style>
