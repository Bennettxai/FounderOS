<script lang="ts">
	import type { MeetingType } from '$lib/api';
	import type { ViewMode } from './calendarUtils';

	interface Props {
		viewMode: ViewMode;
		headerText: string;
		selectedMeetingType: MeetingType | '';
		showSidebar: boolean;
		onNavigatePrev: () => void;
		onNavigateNext: () => void;
		onNavigateToday: () => void;
		onViewModeChange: (mode: ViewMode) => void;
		onMeetingTypeChange: (type: MeetingType | '') => void;
		onToggleSidebar: () => void;
		onCreateEvent: () => void;
		onSync: () => void;
		isSyncing: boolean;
		isConnected: boolean;
		lastSync?: string | null;
	}

	let {
		viewMode,
		headerText,
		selectedMeetingType = $bindable(),
		showSidebar,
		onNavigatePrev,
		onNavigateNext,
		onNavigateToday,
		onViewModeChange,
		onMeetingTypeChange,
		onToggleSidebar,
		onCreateEvent,
		onSync,
		isSyncing,
		isConnected,
		lastSync
	}: Props = $props();

	const syncTooltip = $derived(() => {
		if (isSyncing) return 'Syncing…';
		if (!lastSync) return 'Sync calendar (never synced)';
		const ts = new Date(lastSync);
		if (Number.isNaN(ts.getTime())) return 'Sync calendar';
		const diffMs = Date.now() - ts.getTime();
		const mins = Math.floor(diffMs / 60_000);
		if (mins < 1) return 'Sync calendar — last synced just now';
		if (mins < 60) return `Sync calendar — last synced ${mins}m ago`;
		const hrs = Math.floor(mins / 60);
		if (hrs < 24) return `Sync calendar — last synced ${hrs}h ago`;
		return `Sync calendar — last synced ${ts.toLocaleString()}`;
	});

	const meetingTypes: Array<{ value: MeetingType | ''; label: string }> = [
		{ value: '', label: 'All types' },
		{ value: 'team', label: 'Team' },
		{ value: 'sales', label: 'Sales' },
		{ value: 'client', label: 'Client' },
		{ value: 'onboarding', label: 'Onboarding' },
		{ value: 'kickoff', label: 'Kickoff' },
		{ value: 'implementation', label: 'Implementation' },
		{ value: 'standup', label: 'Standup' },
		{ value: 'planning', label: 'Planning' },
		{ value: 'review', label: 'Review' },
		{ value: 'one_on_one', label: '1:1' },
		{ value: 'retrospective', label: 'Retrospective' },
		{ value: 'internal', label: 'Internal' },
		{ value: 'external', label: 'External' },
		{ value: 'other', label: 'Other' }
	];

	const viewModes: ViewMode[] = ['day', 'week', 'month', 'agenda'];
</script>

<div class="ct-bar">
	<!-- Left: Title + Navigation -->
	<div class="ct-nav">
		<h1 class="ct-title">Calendar</h1>

		<span class="ct-divider"></span>

		<div class="ct-nav__arrows">
			<button
				onclick={onNavigatePrev}
				class="ct-arrow-btn"
				aria-label="Previous period"
			>
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M15 19l-7-7 7-7" />
				</svg>
			</button>
			<button
				onclick={onNavigateNext}
				class="ct-arrow-btn"
				aria-label="Next period"
			>
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M9 5l7 7-7 7" />
				</svg>
			</button>
		</div>

		<button
			onclick={onNavigateToday}
			class="ct-today-btn"
		>
			Today
		</button>

		<h2 class="ct-heading">{headerText}</h2>
	</div>

	<!-- Right: Actions + Filters + View Toggle + Sidebar -->
	<div class="ct-controls">
		<button onclick={onCreateEvent} class="ct-create-btn" aria-label="Create new event">
			<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M12 4v16m8-8H4" />
			</svg>
			<span>Event</span>
		</button>

		{#if isConnected}
			<button
				onclick={onSync}
				disabled={isSyncing}
				class="ct-icon-btn"
				aria-label={syncTooltip()}
				title={syncTooltip()}
			>
				<svg class="w-4 h-4" class:ct-spinning={isSyncing} fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
				</svg>
			</button>
		{/if}

		<span class="ct-divider"></span>

		<div class="ct-select-wrap">
			<select
				bind:value={selectedMeetingType}
				onchange={(e) => onMeetingTypeChange((e.currentTarget as HTMLSelectElement).value as MeetingType | '')}
				class="ct-select"
				aria-label="Filter events by meeting type"
			>
				{#each meetingTypes as type (type.value)}
					<option value={type.value}>{type.label}</option>
				{/each}
			</select>
			<svg class="ct-select__chevron" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
			</svg>
		</div>

		<span class="ct-divider"></span>

		<div class="ct-view-toggle" role="group" aria-label="Calendar view mode">
			{#each viewModes as mode (mode)}
				<button
					onclick={() => onViewModeChange(mode)}
					class="ct-view-btn {viewMode === mode ? 'ct-view-btn--active' : ''}"
					aria-pressed={viewMode === mode}
				>
					{mode.charAt(0).toUpperCase() + mode.slice(1)}
				</button>
			{/each}
		</div>

		<button
			onclick={onToggleSidebar}
			class="ct-icon-btn"
			aria-label={showSidebar ? 'Hide sidebar' : 'Show sidebar'}
		>
			<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h7" />
			</svg>
		</button>
	</div>
</div>

<style>
	.ct-bar {
		padding: 0.75rem 1.5rem;
		border-bottom: 1px solid var(--dbd2);
		display: flex;
		align-items: center;
		justify-content: space-between;
		flex-shrink: 0;
		background: var(--dbg);
		gap: 0.75rem;
		flex-wrap: wrap;
	}

	@media (max-width: 900px) {
		.ct-bar {
			padding: 0.65rem 1rem;
			row-gap: 0.5rem;
		}

		.ct-heading {
			display: none;
		}

		.ct-create-btn span {
			display: none;
		}

		.ct-create-btn {
			padding: 0 0.65rem;
		}

		.ct-select {
			min-width: 6.5rem;
			font-size: 0.72rem;
		}

		.ct-controls {
			gap: 0.5rem;
		}
	}

	@media (max-width: 640px) {
		.ct-view-toggle {
			display: none;
		}

		.ct-nav {
			gap: 0.5rem;
		}
	}

	/* ─── Left Nav ─── */

	.ct-nav {
		display: flex;
		align-items: center;
		gap: 0.75rem;
	}

	.ct-title {
		font-size: 0.9rem;
		font-weight: 700;
		color: var(--dt);
		margin: 0;
		white-space: nowrap;
		letter-spacing: -0.01em;
	}

	.ct-heading {
		font-size: 0.9rem;
		font-weight: 600;
		color: var(--dt);
		margin: 0;
		white-space: nowrap;
		letter-spacing: -0.01em;
	}

	/* ─── Divider ─── */

	.ct-divider {
		width: 1px;
		height: 20px;
		background: var(--dbd2);
		flex-shrink: 0;
		opacity: 0.6;
	}

	/* ─── Arrow buttons ─── */

	.ct-nav__arrows {
		display: flex;
		align-items: center;
		gap: 0.125rem;
	}

	.ct-arrow-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 32px;
		height: 32px;
		min-width: 32px;
		border-radius: 7px;
		border: 1px solid transparent;
		background: transparent;
		color: var(--dt3);
		cursor: pointer;
		transition: background 0.12s, border-color 0.12s, color 0.12s;
		padding: 0;
	}

	.ct-arrow-btn:hover {
		background: var(--dbg2);
		border-color: var(--dbd);
		color: var(--dt);
	}

	.ct-arrow-btn:active {
		background: var(--dbg3);
		color: var(--dt);
	}

	/* ─── Today button ─── */

	.ct-today-btn {
		display: inline-flex;
		align-items: center;
		height: 30px;
		padding: 0 0.75rem;
		border-radius: 7px;
		border: 1px solid var(--dbd);
		background: var(--dbg2);
		color: var(--dt2);
		font-size: 0.75rem;
		font-weight: 600;
		font-family: inherit;
		cursor: pointer;
		white-space: nowrap;
		transition: background 0.12s, border-color 0.12s, color 0.12s;
		letter-spacing: 0.01em;
		border-left: 2px solid var(--bos-nav-active, var(--dt3));
	}

	.ct-today-btn:hover {
		background: var(--dbg3);
		border-color: var(--dt3);
		border-left-color: var(--bos-nav-active, var(--dt2));
		color: var(--dt);
	}

	.ct-today-btn:active {
		background: var(--dbg3);
		color: var(--dt);
	}

	/* ─── Right controls ─── */

	.ct-controls {
		display: flex;
		align-items: center;
		gap: 0.75rem;
	}

	/* ─── Create Event CTA button ─── */
	/* Blue accent — matches PillButton variant="cta" used across comms. */

	.ct-create-btn {
		display: inline-flex;
		align-items: center;
		gap: 0.375rem;
		height: 32px;
		padding: 0 0.875rem;
		border-radius: 7px;
		border: 1px solid rgba(59, 130, 246, 0.55);
		background: linear-gradient(180deg, #5b9cff 0%, var(--bos-accent-blue) 100%);
		color: #ffffff;
		font-size: 0.75rem;
		font-weight: 600;
		font-family: inherit;
		cursor: pointer;
		white-space: nowrap;
		transition: background 0.15s, box-shadow 0.15s, transform 0.15s, border-color 0.15s;
		letter-spacing: 0.01em;
		box-shadow:
			0 1px 0 0 rgba(255, 255, 255, 0.22) inset,
			0 4px 12px 0 rgba(59, 130, 246, 0.28),
			0 8px 20px 0 rgba(59, 130, 246, 0.15);
	}

	.ct-create-btn:hover {
		transform: translateY(-1px);
		background: linear-gradient(180deg, #6daaff 0%, #2f73ed 100%);
		border-color: rgba(59, 130, 246, 0.7);
		box-shadow:
			0 1px 0 0 rgba(255, 255, 255, 0.3) inset,
			0 6px 18px 0 rgba(59, 130, 246, 0.42),
			0 10px 26px 0 rgba(59, 130, 246, 0.22);
	}

	.ct-create-btn:active {
		transform: translateY(0);
		background: linear-gradient(180deg, #4a90f5 0%, #2563eb 100%);
		box-shadow:
			0 1px 0 0 rgba(255, 255, 255, 0.18) inset,
			0 2px 8px 0 rgba(59, 130, 246, 0.3);
	}

	/* ─── Generic icon button ─── */

	.ct-icon-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 32px;
		height: 32px;
		min-width: 32px;
		border-radius: 7px;
		border: 1px solid transparent;
		background: transparent;
		color: var(--dt3);
		cursor: pointer;
		transition: background 0.12s, border-color 0.12s, color 0.12s;
		padding: 0;
	}

	.ct-icon-btn:hover {
		background: var(--dbg2);
		border-color: var(--dbd);
		color: var(--dt2);
	}

	.ct-icon-btn:active {
		background: var(--dbg3);
	}

	.ct-icon-btn:disabled {
		opacity: 0.45;
		cursor: not-allowed;
	}

	/* ─── Select dropdown ─── */

	.ct-select-wrap {
		position: relative;
		display: flex;
		align-items: center;
	}

	.ct-select {
		appearance: none;
		-webkit-appearance: none;
		background: var(--dbg2);
		border: 1px solid var(--dbd);
		border-radius: 7px;
		color: var(--dt2);
		font-size: 0.775rem;
		font-family: inherit;
		font-weight: 500;
		padding: 0 1.9rem 0 0.75rem;
		cursor: pointer;
		outline: none;
		transition: border-color 0.12s, box-shadow 0.12s;
		min-width: 8.5rem;
		height: 32px;
		letter-spacing: 0.01em;
	}

	.ct-select:hover {
		border-color: var(--dt4);
	}

	.ct-select:focus {
		border-color: var(--dt3);
		box-shadow: 0 0 0 2px color-mix(in srgb, var(--dt3) 15%, transparent);
	}

	.ct-select option {
		background: var(--dbg);
		color: var(--dt);
	}

	.ct-select__chevron {
		position: absolute;
		right: 0.55rem;
		top: 50%;
		transform: translateY(-50%);
		width: 13px;
		height: 13px;
		color: var(--dt4);
		pointer-events: none;
		flex-shrink: 0;
	}

	/* ─── View mode toggle group ─── */

	.ct-view-toggle {
		display: flex;
		align-items: center;
		background: var(--dbg2);
		border: 1px solid var(--dbd);
		border-radius: 8px;
		padding: 3px;
		gap: 2px;
	}

	.ct-view-btn {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		height: 24px;
		padding: 0 0.6rem;
		border-radius: 5px;
		border: 1px solid transparent;
		background: transparent;
		color: var(--dt4);
		font-size: 0.7rem;
		font-weight: 500;
		font-family: inherit;
		cursor: pointer;
		white-space: nowrap;
		transition: background 0.12s, border-color 0.12s, color 0.12s;
		letter-spacing: 0.01em;
	}

	.ct-view-btn:hover:not(.ct-view-btn--active) {
		background: var(--dbg3);
		color: var(--dt2);
	}

	.ct-view-btn--active {
		background: var(--dbg3);
		border-color: var(--dbd2);
		color: var(--dt);
		font-weight: 600;
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.25);
	}

	/* ─── Sync spinner ─── */

	.ct-spinning {
		animation: ct-spin 0.8s linear infinite;
	}

	@keyframes ct-spin {
		to {
			transform: rotate(360deg);
		}
	}
</style>
