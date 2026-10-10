<script lang="ts">
	import { fade } from 'svelte/transition';
	import PriorityBadge from './PriorityBadge.svelte';

	type TaskStatus = 'todo' | 'in_progress' | 'in_review' | 'done' | 'blocked';
	type Priority = 'critical' | 'high' | 'medium' | 'low';
	type CalendarView = 'month' | 'week' | 'day';

	interface Task {
		id: string;
		title: string;
		status: TaskStatus;
		priority: Priority;
		dueDate?: string;
	}

	interface Props {
		tasks: Task[];
		view?: CalendarView;
		onTaskClick?: (taskId: string) => void;
		onDateClick?: (date: Date) => void;
	}

	let { tasks, view = 'month', onTaskClick, onDateClick }: Props = $props();

	let currentDate = $state(new Date());
	let selectedView = $state<CalendarView>(view);

	const weekDays = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];

	const priorityDotHex: Record<Priority, string> = {
		critical: '#ef4444',
		high: '#f97316',
		medium: '#eab308',
		low: '#9ca3af'
	};

	function getMonthDays(date: Date): (Date | null)[] {
		const year = date.getFullYear();
		const month = date.getMonth();

		const firstDay = new Date(year, month, 1);
		const lastDay = new Date(year, month + 1, 0);

		const days: (Date | null)[] = [];

		// Add empty days for the start of the week
		for (let i = 0; i < firstDay.getDay(); i++) {
			days.push(null);
		}

		// Add all days of the month
		for (let i = 1; i <= lastDay.getDate(); i++) {
			days.push(new Date(year, month, i));
		}

		return days;
	}

	function getTasksForDate(date: Date): Task[] {
		const dateStr = date.toISOString().split('T')[0];
		return tasks.filter(task => {
			if (!task.dueDate) return false;
			return task.dueDate.split('T')[0] === dateStr;
		});
	}

	function isToday(date: Date): boolean {
		const today = new Date();
		return date.toDateString() === today.toDateString();
	}

	function navigateMonth(direction: number) {
		currentDate = new Date(currentDate.getFullYear(), currentDate.getMonth() + direction, 1);
	}

	function goToToday() {
		currentDate = new Date();
	}

	const monthDays = $derived(getMonthDays(currentDate));

	const monthName = $derived(
		currentDate.toLocaleDateString('en-US', { month: 'long', year: 'numeric' })
	);
</script>

<div class="flex-1 flex flex-col overflow-hidden p-6">
	<!-- Calendar Header -->
	<div class="flex items-center justify-between mb-6">
		<div class="flex items-center gap-4">
			<button
				onclick={() => navigateMonth(-1)}
				class="btn-pill btn-pill-ghost btn-pill-icon"
			>
				<svg class="w-5 h-5 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
				</svg>
			</button>

			<h2 class="text-lg font-semibold text-gray-900 min-w-[180px] text-center">
				{monthName}
			</h2>

			<button
				onclick={() => navigateMonth(1)}
				class="btn-pill btn-pill-ghost btn-pill-icon"
			>
				<svg class="w-5 h-5 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
				</svg>
			</button>

			<button
				onclick={goToToday}
				class="btn-pill btn-pill-ghost btn-pill-sm"
			>
				Today
			</button>
		</div>

		<div class="flex items-center gap-1 bg-gray-100 rounded-lg p-1">
			<button
				onclick={() => selectedView = 'month'}
				class="btn-pill btn-pill-sm {selectedView === 'month' ? 'btn-pill-soft' : 'btn-pill-ghost'}"
			>
				Month
			</button>
			<button
				onclick={() => selectedView = 'week'}
				class="btn-pill btn-pill-sm {selectedView === 'week' ? 'btn-pill-soft' : 'btn-pill-ghost'}"
			>
				Week
			</button>
			<button
				onclick={() => selectedView = 'day'}
				class="btn-pill btn-pill-sm {selectedView === 'day' ? 'btn-pill-soft' : 'btn-pill-ghost'}"
			>
				Day
			</button>
		</div>
	</div>

	<!-- Calendar Grid -->
	<div class="flex-1 flex flex-col border border-gray-200 rounded-xl overflow-hidden bg-white">
		<!-- Week Day Headers -->
		<div class="grid grid-cols-7 border-b border-gray-200 bg-gray-50">
			{#each weekDays as day}
				<div class="py-3 text-center text-xs font-medium text-gray-500 uppercase">
					{day}
				</div>
			{/each}
		</div>

		<!-- Calendar Days -->
		<div class="flex-1 grid grid-cols-7">
			{#each monthDays as date, i}
				{@const dayTasks = date ? getTasksForDate(date) : []}
				<button
					onclick={() => date && onDateClick?.(date)}
					class="min-h-[100px] p-2 border-b border-r border-gray-100 text-left hover:bg-gray-50 transition-colors
						{!date ? 'bg-gray-50' : ''}
						{(i + 1) % 7 === 0 ? 'border-r-0' : ''}"
					disabled={!date}
				>
					{#if date}
						<div class="flex items-center justify-between mb-1">
							<span
								class="w-7 h-7 flex items-center justify-center text-sm rounded-full
									{isToday(date) ? 'bg-gray-900 text-white font-medium' : 'text-gray-700'}"
							>
								{date.getDate()}
							</span>
							{#if dayTasks.length > 3}
								<span class="text-xs text-gray-400">+{dayTasks.length - 3}</span>
							{/if}
						</div>

						<div class="space-y-1">
							{#each dayTasks.slice(0, 3) as task (task.id)}
								<button
									onclick={(e) => {
										e.stopPropagation();
										onTaskClick?.(task.id);
									}}
									class="btn-pill btn-pill-soft btn-pill-xs w-full flex items-center gap-1.5 truncate"
								>
									<span class="w-1.5 h-1.5 rounded-full shrink-0" style="background: {priorityDotHex[task.priority]}"></span>
									<span class="truncate {task.status === 'done' ? 'line-through text-gray-400' : 'text-gray-700'}">
										{task.title}
									</span>
								</button>
							{/each}
						</div>
					{/if}
				</button>
			{/each}
		</div>
	</div>
</div>
