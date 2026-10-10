<script lang="ts">
	import type { CalendarEvent } from '$lib/api';
	import {
		weekDays,
		hours,
		isToday,
		formatHourFmt,
		formatTimeFmt,
		getEventColor,
		getTimedEventSegment,
		isEventOnDate
	} from './calendarUtils';

	interface Props {
		/** 'week' renders 7 columns; 'day' renders 1 column for currentDate */
		mode: 'week' | 'day';
		currentDate: Date;
		weekDates: Date[];
		events: CalendarEvent[];
		currentTime: Date;
		memberColors?: Record<string, string>;
		/** 24-hour time labels when true ("13:00"), 12-hour otherwise ("1 PM"). */
		use24h?: boolean;
		/** Hours outside [workingHoursStart, workingHoursEnd) are subtly shaded. */
		workingHoursStart?: number;
		workingHoursEnd?: number;
		onOpenEventModal: (event: CalendarEvent) => void;
		onOpenCreateModalAtHour: (date: Date, hour: number) => void;
	}

	let {
		mode,
		currentDate,
		weekDates,
		events,
		currentTime,
		memberColors = {},
		use24h = false,
		workingHoursStart = 9,
		workingHoursEnd = 17,
		onOpenEventModal,
		onOpenCreateModalAtHour
	}: Props = $props();

	// An hour is "working" if it falls within [start, end). Outside hours get shaded.
	function isWorkingHour(hour: number): boolean {
		return hour >= workingHoursStart && hour < workingHoursEnd;
	}

	const HOUR_PX = 56; // height of one hour row
	const DAY_PX = 24 * HOUR_PX;

	const currentTimePosition = $derived(
		((currentTime.getHours() * 60 + currentTime.getMinutes()) / 60) * HOUR_PX
	);

	function colorFor(ev: CalendarEvent): string {
		return ev.color_hex || memberColors[ev.user_id] || getEventColor(ev);
	}

	function minToPx(min: number): number {
		return (min / 60) * HOUR_PX;
	}

	interface LaidOut {
		event: CalendarEvent;
		segmentStart: Date;
		segmentEnd: Date;
		top: number;
		height: number;
		col: number;
		cols: number;
	}

	// Absolute-positioned layout for one day's timed events: top = minutes from
	// midnight, height = real duration, with overlapping events split into
	// side-by-side columns (the standard calendar layout algorithm).
	function layoutDay(evs: CalendarEvent[], date: Date): LaidOut[] {
		const items = evs
			.map((event) => ({ event, segment: getTimedEventSegment(event, date) }))
			.filter((item): item is { event: CalendarEvent; segment: NonNullable<typeof item.segment> } => item.segment !== null)
			.map(({ event, segment }) => ({
				event,
				segmentStart: segment.start,
				segmentEnd: segment.end,
				startMin: segment.startMinute,
				endMin: Math.max(segment.startMinute + 20, segment.endMinute)
			}))
			.sort((a, b) => a.startMin - b.startMin || a.endMin - b.endMin);

		const out: LaidOut[] = [];
		let cluster: typeof items = [];
		let clusterEnd = -1;

		const flush = () => {
			if (!cluster.length) return;
			const colEnds: number[] = [];
			const placement = new Map<(typeof cluster)[number], number>();
			for (const it of cluster) {
				let col = colEnds.findIndex((end) => end <= it.startMin);
				if (col === -1) {
					col = colEnds.length;
					colEnds.push(it.endMin);
				} else {
					colEnds[col] = it.endMin;
				}
				placement.set(it, col);
			}
			const cols = colEnds.length;
			for (const it of cluster) {
				out.push({
					event: it.event,
					segmentStart: it.segmentStart,
					segmentEnd: it.segmentEnd,
					top: minToPx(it.startMin),
					height: minToPx(it.endMin - it.startMin),
					col: placement.get(it) ?? 0,
					cols
				});
			}
			cluster = [];
			clusterEnd = -1;
		};

		for (const it of items) {
			if (cluster.length && it.startMin >= clusterEnd) flush();
			cluster.push(it);
			clusterEnd = Math.max(clusterEnd, it.endMin);
		}
		flush();
		return out;
	}

	function allDayFor(date: Date): CalendarEvent[] {
		return events.filter((e) => e.all_day && isEventOnDate(e, date));
	}

	const dayColumns = $derived(mode === 'week' ? weekDates : [currentDate]);
	const trackHasAllDay = $derived(dayColumns.some((d) => allDayFor(d).length > 0));
</script>

<div class="cwdv" class:cwdv--day={mode === 'day'}>
	<div class="cwdv-scroll">
		<!-- Sticky header stack (date row + all-day row) shares the scroll width
		     with the grid below so every column separator lines up exactly. -->
		<div class="cwdv-head-stack">
			<div class="cwdv-row cwdv-header">
				<div class="cwdv-gutter"></div>
				{#each dayColumns as date (date.toISOString())}
					<div class="cwdv-col cwdv-col-head">
						<span class="cwdv-col-day">{weekDays[date.getDay()]}</span>
						<span class="cwdv-col-num" class:cwdv-col-num--today={isToday(date)}>{date.getDate()}</span>
					</div>
				{/each}
			</div>
			{#if trackHasAllDay}
				<div class="cwdv-row cwdv-allday">
					<div class="cwdv-gutter cwdv-allday-label">all day</div>
					{#each dayColumns as date (date.toISOString())}
						<div class="cwdv-col cwdv-allday-col">
							{#each allDayFor(date) as ev (ev.id)}
								<button
									class="cwdv-allday-pill"
									style="--c: {colorFor(ev)};"
									onclick={() => onOpenEventModal(ev)}
									title={ev.title || 'Untitled'}
								>
									{ev.title || 'Untitled'}
								</button>
							{/each}
						</div>
					{/each}
				</div>
			{/if}
		</div>

		<!-- Time grid: gutter labels + lines + day tracks -->
		<div class="cwdv-grid" style="height: {DAY_PX}px;">
			<div class="cwdv-gutter cwdv-time-gutter">
				{#each hours as hour (hour)}
					{#if hour > 0}
						<span class="cwdv-time-label" style="top: {hour * HOUR_PX}px;">{formatHourFmt(hour, use24h)}</span>
					{/if}
				{/each}
			</div>

			<div class="cwdv-grid-body">
				<!-- Subtle shading for non-working hours, behind the grid lines + events. -->
				{#each hours as hour (hour)}
					{#if !isWorkingHour(hour)}
						<div class="cwdv-offhours" style="top: {hour * HOUR_PX}px; height: {HOUR_PX}px;"></div>
					{/if}
				{/each}
				{#each hours as hour (hour)}
					<div class="cwdv-line" style="top: {hour * HOUR_PX}px;"></div>
				{/each}

				<div class="cwdv-tracks">
					{#each dayColumns as date (date.toISOString())}
						<div class="cwdv-col cwdv-track">
							{#each hours as hour (hour)}
								<button
									class="cwdv-cell"
									style="top: {hour * HOUR_PX}px; height: {HOUR_PX}px;"
									aria-label="Add event at {date.toLocaleDateString()} {formatHourFmt(hour, use24h)}"
									onclick={() => onOpenCreateModalAtHour(date, hour)}
								></button>
							{/each}

							{#each layoutDay(events, date) as it (it.event.id)}
								<button
									class="cwdv-event"
									style="top: {it.top}px; height: {it.height}px; left: calc({(it.col / it.cols) * 100}% + 1px); width: calc({100 / it.cols}% - 3px); --c: {colorFor(it.event)};"
									onclick={(e) => { e.stopPropagation(); onOpenEventModal(it.event); }}
									title={it.event.title || 'Untitled'}
								>
									<span class="cwdv-event-title">{it.event.title || 'Untitled'}</span>
									{#if it.height >= 32}
										<span class="cwdv-event-time">{formatTimeFmt(it.segmentStart, use24h)} - {formatTimeFmt(it.segmentEnd, use24h)}</span>
									{/if}
								</button>
							{/each}

							{#if isToday(date)}
								<div class="cwdv-now" style="top: {currentTimePosition}px;">
									<span class="cwdv-now-dot"></span>
								</div>
							{/if}
						</div>
					{/each}
				</div>
			</div>
		</div>
	</div>
</div>

<style>
	.cwdv {
		display: flex;
		flex-direction: column;
		height: 100%;
		min-width: 760px;
	}
	.cwdv--day {
		min-width: 420px;
	}
	.cwdv-scroll {
		flex: 1;
		overflow-y: auto;
		scrollbar-gutter: stable;
	}

	/* shared row layout: gutter + N equal columns */
	.cwdv-row {
		display: flex;
	}
	.cwdv-gutter {
		width: 60px;
		flex-shrink: 0;
	}
	.cwdv-col {
		flex: 1 1 0;
		min-width: 0;
		border-left: 1px solid var(--dbd2);
	}

	/* sticky header stack */
	.cwdv-head-stack {
		position: sticky;
		top: 0;
		z-index: 6;
		background: var(--dbg);
	}
	.cwdv-header {
		border-bottom: 1px solid var(--dbd);
	}
	.cwdv-col-head {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 3px;
		padding: 8px 4px;
	}
	.cwdv-col-day {
		font-size: 0.7rem;
		font-weight: 600;
		letter-spacing: 0.04em;
		text-transform: uppercase;
		color: var(--dt3);
	}
	.cwdv-col-num {
		font-size: 1.05rem;
		font-weight: 650;
		color: var(--dt);
		width: 1.8rem;
		height: 1.8rem;
		display: flex;
		align-items: center;
		justify-content: center;
		line-height: 1;
	}
	.cwdv-col-num--today {
		background: var(--dt);
		color: var(--dbg);
		border-radius: 50%;
		font-size: 0.9rem;
		font-weight: 700;
	}

	/* all-day strip */
	.cwdv-allday {
		border-bottom: 1px solid var(--dbd);
		background: var(--dbg2);
		max-height: 84px;
		overflow-y: auto;
	}
	.cwdv-allday-label {
		font-size: 0.6rem;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		color: var(--dt4);
		padding: 7px 6px 0 0;
		text-align: right;
		align-self: flex-start;
	}
	.cwdv-allday-col {
		padding: 4px;
		display: flex;
		flex-direction: column;
		gap: 3px;
	}
	.cwdv-allday-pill {
		text-align: left;
		font-size: 0.72rem;
		font-weight: 680;
		padding: 4px 8px;
		border-radius: 5px;
		background: linear-gradient(180deg, color-mix(in srgb, var(--c) 94%, white 6%), color-mix(in srgb, var(--c) 90%, black 10%));
		border: 1px solid color-mix(in srgb, var(--c) 72%, black 8%);
		color: white;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
		cursor: pointer;
		box-shadow: inset 0 1px 0 rgba(255,255,255,0.16);
	}
	.cwdv-allday-pill:hover {
		filter: brightness(1.05);
	}

	/* time grid */
	.cwdv-grid {
		position: relative;
		display: flex;
	}
	.cwdv-time-gutter {
		position: relative;
	}
	.cwdv-time-label {
		position: absolute;
		right: 7px;
		transform: translateY(-50%);
		font-size: 0.66rem;
		color: var(--dt4);
		white-space: nowrap;
		font-variant-numeric: tabular-nums;
	}
	.cwdv-grid-body {
		position: relative;
		flex: 1;
		min-width: 0;
	}
	.cwdv-line {
		position: absolute;
		left: 0;
		right: 0;
		border-top: 1px solid var(--dbd2);
	}
	.cwdv-offhours {
		position: absolute;
		left: 0;
		right: 0;
		background: color-mix(in srgb, var(--dt) 3.5%, transparent);
		pointer-events: none;
		z-index: 0;
	}
	.cwdv-tracks {
		position: absolute;
		inset: 0;
		display: flex;
	}
	.cwdv-track {
		position: relative;
	}
	.cwdv-cell {
		position: absolute;
		left: 0;
		right: 0;
		padding: 0;
		background: transparent;
		border: none;
		cursor: pointer;
	}
	.cwdv-cell:hover {
		background: color-mix(in srgb, var(--dt) 4%, transparent);
	}
	.cwdv-event {
		position: absolute;
		border-radius: 6px;
		padding: 4px 7px;
		background: linear-gradient(180deg, color-mix(in srgb, var(--c) 94%, white 6%), color-mix(in srgb, var(--c) 90%, black 10%));
		border: 1px solid color-mix(in srgb, var(--c) 72%, black 8%);
		color: white;
		overflow: hidden;
		text-align: left;
		display: flex;
		flex-direction: column;
		gap: 1px;
		cursor: pointer;
		min-height: 16px;
		box-shadow: inset 0 1px 0 rgba(255,255,255,0.16);
		transition: filter 0.1s ease, box-shadow 0.1s ease;
		z-index: 2;
	}
	.cwdv-event:hover {
		filter: brightness(1.05);
		box-shadow: inset 0 1px 0 rgba(255,255,255,0.2), 0 1px 3px rgba(15,23,42,0.16);
		z-index: 3;
	}
	.cwdv-event-title {
		font-size: 0.72rem;
		font-weight: 700;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
		line-height: 1.25;
	}
	.cwdv-event-time {
		font-size: 0.63rem;
		color: rgba(255,255,255,0.76);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	.cwdv-now {
		position: absolute;
		left: 0;
		right: 0;
		height: 0;
		border-top: 2px solid #ef4444;
		z-index: 4;
		pointer-events: none;
	}
	.cwdv-now-dot {
		position: absolute;
		left: -4px;
		top: -4px;
		width: 8px;
		height: 8px;
		border-radius: 50%;
		background: #ef4444;
	}
</style>
