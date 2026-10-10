<script lang="ts">
	interface Props {
		isOpen: boolean;
		onToggle: () => void;
	}

	let { isOpen, onToggle }: Props = $props();

	let currentTime = $state(new Date());
	let calendarMonth = $state(new Date());

	$effect(() => {
		const interval = setInterval(() => {
			currentTime = new Date();
		}, 1000);
		return () => clearInterval(interval);
	});

	function formatTime(date: Date): string {
		return date.toLocaleDateString('en-US', {
			weekday: 'short',
			month: 'short',
			day: 'numeric',
			hour: 'numeric',
			minute: '2-digit',
			hour12: true
		});
	}

	function getDaysInMonth(date: Date): number {
		return new Date(date.getFullYear(), date.getMonth() + 1, 0).getDate();
	}

	function getFirstDayOfMonth(date: Date): number {
		return new Date(date.getFullYear(), date.getMonth(), 1).getDay();
	}

	function getCalendarDays(date: Date): (number | null)[] {
		const daysInMonth = getDaysInMonth(date);
		const firstDay = getFirstDayOfMonth(date);
		const days: (number | null)[] = [];

		for (let i = 0; i < firstDay; i++) {
			days.push(null);
		}
		for (let i = 1; i <= daysInMonth; i++) {
			days.push(i);
		}

		return days;
	}

	function prevMonth() {
		calendarMonth = new Date(calendarMonth.getFullYear(), calendarMonth.getMonth() - 1, 1);
	}

	function nextMonth() {
		calendarMonth = new Date(calendarMonth.getFullYear(), calendarMonth.getMonth() + 1, 1);
	}

	function isToday(day: number | null): boolean {
		if (day === null) return false;
		const today = new Date();
		return (
			day === today.getDate() &&
			calendarMonth.getMonth() === today.getMonth() &&
			calendarMonth.getFullYear() === today.getFullYear()
		);
	}

	const calendarDays = $derived(getCalendarDays(calendarMonth));
	const monthYearLabel = $derived(
		calendarMonth.toLocaleDateString('en-US', { month: 'long', year: 'numeric' })
	);
</script>

<div class="menu-bar-item-wrapper">
	<button
		class="menu-bar-clock"
		onclick={onToggle}
		aria-label="Open calendar"
		aria-haspopup="dialog"
		aria-expanded={isOpen}
	>
		{formatTime(currentTime)}
	</button>

	{#if isOpen}
		<div class="menu-dropdown calendar-menu" role="dialog" aria-label="Calendar">
			<div class="calendar-header">
				<button class="calendar-nav" onclick={prevMonth} aria-label="Previous month">
					<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
						<path d="M15 18l-6-6 6-6" />
					</svg>
				</button>
				<span class="calendar-month-year">{monthYearLabel}</span>
				<button class="calendar-nav" onclick={nextMonth} aria-label="Next month">
					<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
						<path d="M9 18l6-6-6-6" />
					</svg>
				</button>
			</div>

			<div class="calendar-weekdays" aria-hidden="true">
				<span>Su</span><span>Mo</span><span>Tu</span><span>We</span><span>Th</span><span>Fr</span><span>Sa</span>
			</div>

			<div class="calendar-days">
				{#each calendarDays as day}
					<span
						class="calendar-day"
						class:empty={day === null}
						class:today={isToday(day)}
						aria-label={day !== null ? String(day) : undefined}
						aria-current={isToday(day) ? 'date' : undefined}
					>
						{day ?? ''}
					</span>
				{/each}
			</div>

			<div class="calendar-today-btn-wrapper">
				<button class="calendar-today-btn" onclick={() => { calendarMonth = new Date(); }}>
					Today
				</button>
			</div>
		</div>
	{/if}
</div>

<style>
	.menu-bar-item-wrapper {
		position: relative;
	}

	.menu-bar-clock {
		color: #333;
		font-size: 13px;
		background: none;
		border: none;
		cursor: pointer;
		padding: 4px 8px;
		border-radius: 4px;
	}

	.menu-bar-clock:hover {
		background: rgba(0, 0, 0, 0.08);
	}

	.calendar-menu {
		right: 0;
		left: auto;
		min-width: 260px;
		padding: 12px;
	}

	.calendar-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-bottom: 12px;
	}

	.calendar-nav {
		background: none;
		border: none;
		cursor: pointer;
		padding: 4px;
		border-radius: 4px;
		color: #666;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.calendar-nav:hover {
		background: rgba(0, 0, 0, 0.08);
		color: #333;
	}

	.calendar-nav svg {
		width: 16px;
		height: 16px;
	}

	.calendar-month-year {
		font-weight: 600;
		font-size: 14px;
		color: #333;
	}

	.calendar-weekdays {
		display: grid;
		grid-template-columns: repeat(7, 1fr);
		gap: 2px;
		margin-bottom: 4px;
	}

	.calendar-weekdays span {
		text-align: center;
		font-size: 10px;
		font-weight: 600;
		color: #999;
		padding: 4px;
	}

	.calendar-days {
		display: grid;
		grid-template-columns: repeat(7, 1fr);
		gap: 2px;
	}

	.calendar-day {
		text-align: center;
		font-size: 12px;
		padding: 6px;
		border-radius: 4px;
		cursor: pointer;
		color: #333;
	}

	.calendar-day:hover:not(.empty) {
		background: rgba(0, 0, 0, 0.08);
	}

	.calendar-day.empty {
		cursor: default;
	}

	.calendar-day.today {
		background: #0066FF;
		color: white;
		font-weight: 600;
	}

	.calendar-day.today:hover {
		background: #0055DD;
	}

	.calendar-today-btn-wrapper {
		margin-top: 12px;
		padding-top: 8px;
		border-top: 1px solid rgba(0, 0, 0, 0.1);
		display: flex;
		justify-content: center;
	}

	.calendar-today-btn {
		background: none;
		border: 1px solid rgba(0, 0, 0, 0.15);
		padding: 4px 16px;
		border-radius: 4px;
		font-size: 12px;
		cursor: pointer;
		color: #0066FF;
		font-weight: 500;
	}

	.calendar-today-btn:hover {
		background: rgba(0, 102, 255, 0.08);
	}

	/* Dark mode */
	:global(.dark) .menu-bar-clock {
		color: #f5f5f7;
	}

	:global(.dark) .menu-bar-clock:hover {
		background: rgba(255, 255, 255, 0.1);
	}

	:global(.dark) .calendar-nav {
		color: #a1a1a6;
	}

	:global(.dark) .calendar-nav:hover {
		background: rgba(255, 255, 255, 0.1);
		color: #f5f5f7;
	}

	:global(.dark) .calendar-month-year {
		color: #f5f5f7;
	}

	:global(.dark) .calendar-weekdays span {
		color: #6e6e73;
	}

	:global(.dark) .calendar-day {
		color: #f5f5f7;
	}

	:global(.dark) .calendar-day:hover:not(.empty) {
		background: rgba(255, 255, 255, 0.1);
	}

	:global(.dark) .calendar-today-btn-wrapper {
		border-top-color: rgba(255, 255, 255, 0.1);
	}

	:global(.dark) .calendar-today-btn {
		border-color: rgba(255, 255, 255, 0.15);
		color: #0A84FF;
	}

	:global(.dark) .calendar-today-btn:hover {
		background: rgba(10, 132, 255, 0.15);
	}
</style>
