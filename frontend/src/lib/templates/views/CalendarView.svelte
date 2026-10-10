<script lang="ts">
	/**
	 * CalendarView - Calendar view for app templates
	 */

	import type { Field } from '../types/field';
	import type { CalendarViewConfig } from '../types/view';
	import { TemplateButton, TemplateBadge, TemplateSkeleton } from '../primitives';

	interface Props {
		config: CalendarViewConfig;
		fields: Field[];
		data: Record<string, unknown>[];
		loading?: boolean;
		onrowclick?: (record: Record<string, unknown>) => void;
		ondatechange?: (date: Date) => void;
	}

	let {
		config,
		fields,
		data,
		loading = false,
		onrowclick,
		ondatechange
	}: Props = $props();

	let currentDate = $state(new Date());
	let viewMode = $state<'month' | 'week' | 'day'>('month');

	$effect(() => {
		viewMode = config.defaultView || 'month';
	});

	const weekDays = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];
	const weekStartsOn = $derived(config.weekStartsOn || 0);

	// Rotate weekdays based on start day
	const orderedWeekDays = $derived([
		...weekDays.slice(weekStartsOn),
		...weekDays.slice(0, weekStartsOn)
	]);

	const currentMonth = $derived(currentDate.getMonth());
	const currentYear = $derived(currentDate.getFullYear());

	// Get days for the current month view
	const calendarDays = $derived(() => {
		const firstDay = new Date(currentYear, currentMonth, 1);
		const lastDay = new Date(currentYear, currentMonth + 1, 0);
		const startDay = new Date(firstDay);
		startDay.setDate(startDay.getDate() - ((startDay.getDay() - weekStartsOn + 7) % 7));

		const days: Date[] = [];
		const current = new Date(startDay);

		while (days.length < 42) { // 6 weeks
			days.push(new Date(current));
			current.setDate(current.getDate() + 1);
		}

		return days;
	});

	// Group events by date
	const eventsByDate = $derived(() => {
		const map = new Map<string, Record<string, unknown>[]>();

		data.forEach(record => {
			const startDate = record[config.startDateField];
			if (!startDate) return;

			const date = new Date(startDate as string);
			const key = date.toDateString();

			if (!map.has(key)) {
				map.set(key, []);
			}
			map.get(key)!.push(record);
		});

		return map;
	});

	function getEventsForDate(date: Date): Record<string, unknown>[] {
		return eventsByDate().get(date.toDateString()) || [];
	}

	function isToday(date: Date): boolean {
		const today = new Date();
		return date.toDateString() === today.toDateString();
	}

	function isCurrentMonth(date: Date): boolean {
		return date.getMonth() === currentMonth;
	}

	function navigateMonth(delta: number) {
		const newDate = new Date(currentYear, currentMonth + delta, 1);
		currentDate = newDate;
		ondatechange?.(newDate);
	}

	function goToToday() {
		currentDate = new Date();
		ondatechange?.(currentDate);
	}

	function getEventColor(record: Record<string, unknown>): string {
		if (!config.colorField) return 'var(--tpl-accent-primary)';
		const colorValue = record[config.colorField];
		if (!colorValue) return 'var(--tpl-accent-primary)';

		const field = fields.find(f => f.id === config.colorField);
		if (field?.type === 'status') {
			const option = getFieldOptions(field).find((o) => o.value === colorValue);
			if (option?.color) {
				return `var(--tpl-status-${option.color}, ${option.color})`;
			}
		}
		return String(colorValue);
	}

	function getFieldOptions(field: Field): { value: string; label: string; color?: string }[] {
		if ('options' in field && Array.isArray(field.options)) {
			return field.options;
		}
		return field.config?.options ?? [];
	}

	const monthFormatter = new Intl.DateTimeFormat('en-US', { month: 'long', year: 'numeric' });
</script>

<div class="tpl-calendar-view">
	<div class="tpl-calendar-header">
		<div class="tpl-calendar-nav">
			<TemplateButton variant="ghost" size="sm" onclick={() => navigateMonth(-1)}>
				<svg viewBox="0 0 20 20" fill="currentColor" width="16" height="16">
					<path fill-rule="evenodd" d="M12.707 5.293a1 1 0 010 1.414L9.414 10l3.293 3.293a1 1 0 01-1.414 1.414l-4-4a1 1 0 010-1.414l4-4a1 1 0 011.414 0z" clip-rule="evenodd" />
				</svg>
			</TemplateButton>
			<h2 class="tpl-calendar-title">{monthFormatter.format(currentDate)}</h2>
			<TemplateButton variant="ghost" size="sm" onclick={() => navigateMonth(1)}>
				<svg viewBox="0 0 20 20" fill="currentColor" width="16" height="16">
					<path fill-rule="evenodd" d="M7.293 14.707a1 1 0 010-1.414L10.586 10 7.293 6.707a1 1 0 011.414-1.414l4 4a1 1 0 010 1.414l-4 4a1 1 0 01-1.414 0z" clip-rule="evenodd" />
				</svg>
			</TemplateButton>
		</div>
		<div class="tpl-calendar-actions">
			<TemplateButton variant="outline" size="sm" onclick={goToToday}>Today</TemplateButton>
			<div class="tpl-calendar-view-switcher">
				<button
					class="tpl-calendar-view-btn"
					class:tpl-calendar-view-btn-active={viewMode === 'month'}
					onclick={() => viewMode = 'month'}
				>Month</button>
				<button
					class="tpl-calendar-view-btn"
					class:tpl-calendar-view-btn-active={viewMode === 'week'}
					onclick={() => viewMode = 'week'}
				>Week</button>
				<button
					class="tpl-calendar-view-btn"
					class:tpl-calendar-view-btn-active={viewMode === 'day'}
					onclick={() => viewMode = 'day'}
				>Day</button>
			</div>
		</div>
	</div>

	{#if loading}
		<div class="tpl-calendar-loading">
			<TemplateSkeleton variant="rectangular" width="100%" height="400px" />
		</div>
	{:else if viewMode === 'month'}
		<div class="tpl-calendar-grid">
			<div class="tpl-calendar-weekdays">
				{#each orderedWeekDays as day}
					<div class="tpl-calendar-weekday">{day}</div>
				{/each}
			</div>
			<div class="tpl-calendar-days">
				{#each calendarDays() as day}
					{@const events = getEventsForDate(day)}
					{@const dayIsToday = isToday(day)}
					{@const dayInMonth = isCurrentMonth(day)}

					<div
						class="tpl-calendar-day"
						class:tpl-calendar-day-today={dayIsToday}
						class:tpl-calendar-day-other-month={!dayInMonth}
					>
						<div class="tpl-calendar-day-number" class:tpl-calendar-day-number-today={dayIsToday}>
							{day.getDate()}
						</div>
						<div class="tpl-calendar-events">
							{#each events.slice(0, 3) as event}
								<button
									class="tpl-calendar-event"
									style:background={getEventColor(event)}
									onclick={() => onrowclick?.(event)}
								>
									{event[config.titleField]}
								</button>
							{/each}
							{#if events.length > 3}
								<span class="tpl-calendar-more">+{events.length - 3} more</span>
							{/if}
						</div>
					</div>
				{/each}
			</div>
		</div>
	{:else}
		<div class="tpl-calendar-placeholder">
			<p>{viewMode === 'week' ? 'Week' : 'Day'} view coming soon</p>
		</div>
	{/if}
</div>

<style>
	.tpl-calendar-view {
		display: flex;
		flex-direction: column;
		height: 100%;
		background: var(--tpl-bg-primary);
	}

	.tpl-calendar-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: var(--tpl-space-4);
		border-bottom: 1px solid var(--tpl-border-default);
	}

	.tpl-calendar-nav {
		display: flex;
		align-items: center;
		gap: var(--tpl-space-2);
	}

	.tpl-calendar-title {
		margin: 0;
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-lg);
		font-weight: var(--tpl-font-semibold);
		color: var(--tpl-text-primary);
		min-width: 200px;
		text-align: center;
	}

	.tpl-calendar-actions {
		display: flex;
		align-items: center;
		gap: var(--tpl-space-3);
	}

	.tpl-calendar-view-switcher {
		display: flex;
		background: var(--tpl-bg-secondary);
		border-radius: var(--tpl-radius-md);
		padding: 2px;
	}

	.tpl-calendar-view-btn {
		padding: var(--tpl-space-1-5) var(--tpl-space-3);
		background: transparent;
		border: none;
		border-radius: var(--tpl-radius-sm);
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-xs);
		font-weight: var(--tpl-font-medium);
		color: var(--tpl-text-secondary);
		cursor: pointer;
		transition: all var(--tpl-transition-fast);
	}

	.tpl-calendar-view-btn:hover {
		color: var(--tpl-text-primary);
	}

	.tpl-calendar-view-btn-active {
		background: var(--tpl-bg-primary);
		color: var(--tpl-text-primary);
		box-shadow: var(--tpl-shadow-xs);
	}

	.tpl-calendar-loading {
		padding: var(--tpl-space-4);
	}

	.tpl-calendar-grid {
		flex: 1;
		display: flex;
		flex-direction: column;
	}

	.tpl-calendar-weekdays {
		display: grid;
		grid-template-columns: repeat(7, 1fr);
		border-bottom: 1px solid var(--tpl-border-default);
	}

	.tpl-calendar-weekday {
		padding: var(--tpl-space-2);
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-xs);
		font-weight: var(--tpl-font-semibold);
		color: var(--tpl-text-muted);
		text-align: center;
		text-transform: uppercase;
		letter-spacing: var(--tpl-tracking-wide);
	}

	.tpl-calendar-days {
		flex: 1;
		display: grid;
		grid-template-columns: repeat(7, 1fr);
		grid-template-rows: repeat(6, 1fr);
	}

	.tpl-calendar-day {
		min-height: 100px;
		padding: var(--tpl-space-1);
		border-right: 1px solid var(--tpl-border-subtle);
		border-bottom: 1px solid var(--tpl-border-subtle);
		display: flex;
		flex-direction: column;
	}

	.tpl-calendar-day:nth-child(7n) {
		border-right: none;
	}

	.tpl-calendar-day-other-month {
		background: var(--tpl-bg-secondary);
	}

	.tpl-calendar-day-other-month .tpl-calendar-day-number {
		color: var(--tpl-text-muted);
	}

	.tpl-calendar-day-today {
		background: var(--tpl-accent-primary-light);
	}

	.tpl-calendar-day-number {
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-sm);
		font-weight: var(--tpl-font-medium);
		color: var(--tpl-text-primary);
		padding: var(--tpl-space-1);
		text-align: right;
	}

	.tpl-calendar-day-number-today {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 28px;
		height: 28px;
		background: var(--tpl-accent-primary);
		color: var(--tpl-text-inverted);
		border-radius: var(--tpl-radius-full);
		margin-left: auto;
	}

	.tpl-calendar-events {
		flex: 1;
		display: flex;
		flex-direction: column;
		gap: 2px;
		overflow: hidden;
	}

	.tpl-calendar-event {
		padding: 2px var(--tpl-space-1-5);
		background: var(--tpl-accent-primary);
		border: none;
		border-radius: var(--tpl-radius-xs);
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-2xs);
		color: white;
		text-align: left;
		cursor: pointer;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		transition: opacity var(--tpl-transition-fast);
	}

	.tpl-calendar-event:hover {
		opacity: 0.9;
	}

	.tpl-calendar-more {
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-2xs);
		color: var(--tpl-text-muted);
		padding: 2px var(--tpl-space-1);
	}

	.tpl-calendar-placeholder {
		flex: 1;
		display: flex;
		align-items: center;
		justify-content: center;
		color: var(--tpl-text-muted);
		font-family: var(--tpl-font-sans);
	}
</style>
