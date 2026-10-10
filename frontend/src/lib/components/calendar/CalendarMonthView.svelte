<script lang="ts">
	import type { CalendarEvent } from '$lib/api';
	import { getWeekDays, isToday, getEventsForDate, buildMonthData, formatTimeFmt, getEventColor } from './calendarUtils';

	interface Props {
		currentDate: Date;
		selectedDay: Date;
		events: CalendarEvent[];
		dateRange: { start: Date; end: Date };
		memberColors?: Record<string, string>;
		/** 24-hour event times when true ("13:00"), 12-hour otherwise ("1:00 PM"). */
		use24h?: boolean;
		weekStart?: number;
		allowCreate?: boolean;
		allowEventDrag?: boolean;
		onSelectDay: (date: Date) => void;
		onGoToDayView: (date: Date) => void;
		onOpenCreateModal: (date: Date) => void;
		onOpenEventModal: (event: CalendarEvent) => void;
		onMoveEventDate?: (event: CalendarEvent, date: Date) => void | Promise<void>;
	}

	let {
		currentDate,
		selectedDay,
		events,
		dateRange,
		memberColors = {},
		use24h = false,
		weekStart = 0,
		allowCreate = true,
		allowEventDrag = false,
		onSelectDay,
		onGoToDayView,
		onOpenCreateModal,
		onOpenEventModal,
		onMoveEventDate = () => {}
	}: Props = $props();
	let draggingEventId = $state<string | null>(null);
	let dropTargetKey = $state<string | null>(null);

	function dotColor(ev: CalendarEvent): string {
		return ev.color_hex || memberColors[ev.user_id] || getEventColor(ev);
	}

	const monthWeekDays = $derived(getWeekDays(weekStart));
	const monthData = $derived(buildMonthData(currentDate, dateRange, weekStart));

	function isCurrentMonth(date: Date): boolean {
		return date.getMonth() === currentDate.getMonth();
	}
	function isSelected(date: Date): boolean {
		return (
			selectedDay.getDate() === date.getDate() &&
			selectedDay.getMonth() === date.getMonth() &&
			selectedDay.getFullYear() === date.getFullYear()
		);
	}
	function timeLabel(s: string): string {
		return formatTimeFmt(s, use24h);
	}
	function dateKey(date: Date): string {
		return `${date.getFullYear()}-${date.getMonth()}-${date.getDate()}`;
	}
	function startEventDrag(e: DragEvent, event: CalendarEvent) {
		if (!allowEventDrag) return;
		draggingEventId = event.id;
		e.dataTransfer?.setData('application/x-calendar-event', event.id);
		e.dataTransfer?.setData('text/plain', event.id);
		if (e.dataTransfer) e.dataTransfer.effectAllowed = 'move';
		e.stopPropagation();
	}
	function allowEventDrop(e: DragEvent, date: Date) {
		if (!allowEventDrag) return;
		e.preventDefault();
		e.stopPropagation();
		dropTargetKey = dateKey(date);
		if (e.dataTransfer) e.dataTransfer.dropEffect = 'move';
	}
	function dropEvent(e: DragEvent, date: Date) {
		if (!allowEventDrag) return;
		e.preventDefault();
		e.stopPropagation();
		const id = e.dataTransfer?.getData('application/x-calendar-event') || draggingEventId;
		const event = events.find((candidate) => candidate.id === id);
		draggingEventId = null;
		dropTargetKey = null;
		if (event) void onMoveEventDate(event, date);
	}
</script>

<div class="month">
	<div class="weekdays">
		{#each monthWeekDays as day (day)}
			<div class="weekday">{day}</div>
		{/each}
	</div>

	<div class="grid">
		{#each monthData.flat() as date (date.toISOString())}
			{@const todayDate = isToday(date)}
			{@const selectedDate = isSelected(date)}
			{@const inMonth = isCurrentMonth(date)}
			{@const dateEvents = getEventsForDate(events, date)}
			<div
				role="button"
				tabindex="0"
				onclick={() => onSelectDay(date)}
				ondblclick={() => onOpenCreateModal(date)}
				onkeydown={(e) => { if (e.key === 'Enter') onSelectDay(date); }}
				ondragover={(e) => allowEventDrop(e, date)}
				ondragleave={() => { if (dropTargetKey === dateKey(date)) dropTargetKey = null; }}
				ondrop={(e) => dropEvent(e, date)}
				class="cell"
				class:cell--out={!inMonth}
				class:cell--today={todayDate}
				class:cell--selected={selectedDate}
				class:cell--drop={dropTargetKey === dateKey(date)}
			>
				<div class="cell-head">
					<span class="daynum" class:daynum--today={todayDate}>{date.getDate()}</span>
					{#if allowCreate}
						<button
							class="add"
							onclick={(e) => { e.stopPropagation(); onOpenCreateModal(date); }}
							title="Add event"
							aria-label="Add event on {date.toLocaleDateString()}"
						>+</button>
					{/if}
				</div>

				<div class="events">
					{#each dateEvents.slice(0, 3) as event (event.id)}
						<button
							class="ev"
							class:ev--google={event.source === 'google'}
							class:ev--dragging={draggingEventId === event.id}
							style="--c: {dotColor(event)};"
							draggable={allowEventDrag}
							ondragstart={(e) => startEventDrag(e, event)}
							ondragend={() => { draggingEventId = null; dropTargetKey = null; }}
							onclick={(e) => { e.stopPropagation(); onOpenEventModal(event); }}
							title={event.title ?? 'Untitled'}
						>
							<span class="ev-dot" style="background:{dotColor(event)}"></span>
							<span class="ev-time">{event.all_day ? 'All day' : timeLabel(event.start_time)}</span>
							<span class="ev-title">{event.title ?? 'Untitled'}</span>
						</button>
					{/each}
					{#if dateEvents.length > 3}
						<button class="more" onclick={(e) => { e.stopPropagation(); onSelectDay(date); }}>
							+{dateEvents.length - 3} more
						</button>
					{/if}
				</div>
			</div>
		{/each}
	</div>
</div>

<style>
	.month { width: 100%; min-width: 0; box-sizing: border-box; overflow: hidden; padding: 14px 18px 18px; }
	.weekdays { display: grid; grid-template-columns: repeat(7, minmax(0, 1fr)); gap: 6px; width: 100%; min-width: 0; margin-bottom: 8px; }
	.weekday { text-align: left; padding: 0 4px; font-size: 0.7rem; font-weight: 640; letter-spacing: 0.05em; text-transform: uppercase; color: var(--dt4); }
	.grid { display: grid; grid-template-columns: repeat(7, minmax(0, 1fr)); gap: 6px; width: 100%; min-width: 0; }

	.cell {
		min-width: 0;
		box-sizing: border-box;
		min-height: 104px;
		display: flex;
		flex-direction: column;
		gap: 4px;
		padding: 6px 7px;
		border: 1px solid var(--dbd);
		border-radius: 10px;
		background: var(--dbg);
		text-align: left;
		cursor: pointer;
		transition: border-color 0.14s ease, background 0.14s ease;
	}
	.cell:hover { border-color: color-mix(in srgb, var(--dt) 26%, transparent); }
	.cell--drop { border-color: color-mix(in srgb, #2563eb 70%, transparent); background: color-mix(in srgb, #2563eb 7%, var(--dbg)); }
	.cell--out { background: color-mix(in srgb, var(--dt) 2.5%, transparent); }
	.cell--today { border-color: color-mix(in srgb, var(--dt) 40%, transparent); }
	.cell--selected { border-color: var(--bos-nav-active, #6366f1); box-shadow: 0 0 0 1px var(--bos-nav-active, #6366f1) inset; }

	.cell-head { display: flex; align-items: center; justify-content: space-between; }
	.daynum { font-size: 0.78rem; font-weight: 560; color: var(--dt2); }
	.cell--out .daynum { color: var(--dt4); }
	.daynum--today {
		display: inline-flex; align-items: center; justify-content: center;
		min-width: 20px; height: 20px; padding: 0 5px; border-radius: 999px;
		background: var(--dt); color: var(--dbg); font-weight: 680;
	}
	.add {
		width: 18px; height: 18px; border-radius: 5px; border: none; background: transparent;
		color: var(--dt3); font-size: 14px; line-height: 1; cursor: pointer; opacity: 0;
		display: inline-flex; align-items: center; justify-content: center; transition: opacity 0.12s ease, background 0.12s ease;
	}
	.cell:hover .add { opacity: 1; }
	.add:hover { background: color-mix(in srgb, var(--dt) 12%, transparent); color: var(--dt); }

	.events { display: flex; flex-direction: column; gap: 3px; min-width: 0; overflow: hidden; }
	.ev {
		display: flex; align-items: center; gap: 5px; width: 100%;
		min-width: 0;
		min-height: 18px; padding: 2px 6px; border-radius: 5px; border: 1px solid color-mix(in srgb, var(--c) 72%, black 8%); cursor: pointer;
		background: linear-gradient(180deg, color-mix(in srgb, var(--c) 94%, white 6%), color-mix(in srgb, var(--c) 90%, black 10%));
		color: white; font-size: 0.7rem; font-weight: 680; text-align: left; overflow: hidden;
		box-shadow: inset 0 1px 0 rgba(255,255,255,0.16);
	}
	.ev:hover { filter: brightness(1.05); box-shadow: inset 0 1px 0 rgba(255,255,255,0.2), 0 1px 3px rgba(15,23,42,0.16); }
	.ev--dragging { opacity: .55; }
	.ev-dot { width: 5px; height: 5px; border-radius: 50%; background: rgba(255,255,255,0.92); flex-shrink: 0; box-shadow: 0 0 0 1px rgba(0,0,0,0.08); }
	.ev--google .ev-dot { background: rgba(255,255,255,0.92); }
	.ev-time { color: rgba(255,255,255,0.74); flex-shrink: 0; font-variant-numeric: tabular-nums; font-weight: 640; }
	.ev-title { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
	.more { border: none; background: transparent; color: var(--dt3); font-size: 0.68rem; padding: 1px 5px; text-align: left; cursor: pointer; }
	.more:hover { color: var(--dt); }
</style>
