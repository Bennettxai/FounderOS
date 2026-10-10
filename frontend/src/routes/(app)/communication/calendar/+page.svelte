<script lang="ts">
	import {
		api,
		apiClient,
		type CalendarEvent,
		type GoogleConnectionStatus,
		type MeetingType
	} from '$lib/api';
	import { getCalendarConnectionStatus, getCalendarAuthUrl } from '$lib/api/calendar';
	import { onMount, tick } from 'svelte';
	import { toast } from 'svelte-sonner';

	import { fade } from 'svelte/transition';

	import { bindShortcuts } from '$lib/components/comms/commsKeyboard';
	import PillButton from '$lib/components/ui/PillButton.svelte';
	import CalendarSidebar from '$lib/components/calendar/CalendarSidebar.svelte';
	import CalendarWeekDayView from '$lib/components/calendar/CalendarWeekDayView.svelte';
	import CalendarMonthView from '$lib/components/calendar/CalendarMonthView.svelte';
	import CalendarAgendaView from '$lib/components/calendar/CalendarAgendaView.svelte';
	import CalendarEventModal from '$lib/components/calendar/CalendarEventModal.svelte';
	import CalendarEventForm from '$lib/components/calendar/CalendarEventForm.svelte';
	import CalendarStatusBanner from '$lib/components/calendar/CalendarStatusBanner.svelte';
	import CalendarToolbar from '$lib/components/calendar/CalendarToolbar.svelte';

	import {
		type ViewMode,
		type SyncStats,
		type EventFormData,
		buildDateRange,
		isToday
	} from '$lib/components/calendar/calendarUtils';


	// ── View State ────────────────────────────────────────────────────────────
	let viewMode = $state<ViewMode>('week');
	let currentDate = $state(new Date());
	let selectedDay = $state(new Date());
	let showSidebar = $state(true);

	// ── Data ─────────────────────────────────────────────────────────────────
	let events = $state<CalendarEvent[]>([]);
	let upcomingEvents = $state<CalendarEvent[]>([]);
	let syncStats = $state<SyncStats | null>(null);
	let isLoading = $state(true);
	let isSyncing = $state(false);
	let connectionStatus = $state<GoogleConnectionStatus | null>(null);
	let selectedMeetingType = $state<MeetingType | ''>('');

	// ── Event Modal ───────────────────────────────────────────────────────────
	let selectedEvent = $state<CalendarEvent | null>(null);
	let showEventModal = $state(false);

	// ── Create/Edit Modal ─────────────────────────────────────────────────────
	let showCreateModal = $state(false);
	let editingEvent = $state<CalendarEvent | null>(null);
	let isSaving = $state(false);
	let formError = $state('');
	let formData = $state<EventFormData>({
		title: '',
		description: '',
		start_date: '',
		start_time: '09:00',
		end_date: '',
		end_time: '10:00',
		all_day: false,
		location: '',
		meeting_type: '',
		meeting_link: ''
	});

	// ── Time Indicator ────────────────────────────────────────────────────────
	let currentTime = $state(new Date());
	let timeGridRef: HTMLDivElement | null = $state(null);

	$effect(() => {
		const interval = setInterval(() => {
			currentTime = new Date();
		}, 60000);
		return () => clearInterval(interval);
	});

	// Scroll to current time on view change
	$effect(() => {
		if (timeGridRef && (viewMode === 'week' || viewMode === 'day') && !isLoading) {
			const scrollTop = Math.max(0, (new Date().getHours() - 2) * 60);
			setTimeout(() => {
				timeGridRef?.scrollTo({ top: scrollTop, behavior: 'smooth' });
			}, 100);
		}
	});

	// Keyboard shortcuts — Gmail/Linear-style. J/K navigate, T jumps to today,
	// 1-4 swap views, C creates an event. Skipped while typing in inputs.
	$effect(() =>
		bindShortcuts([
			{ key: 'j', description: 'Previous period', handler: () => { navigatePrev(); } },
			{ key: 'k', description: 'Next period', handler: () => { navigateNext(); } },
			{ key: 't', description: 'Jump to today', handler: () => { navigateToday(); } },
			{ key: '1', description: 'Day view', handler: () => { viewMode = 'day'; loadEvents(); } },
			{ key: '2', description: 'Week view', handler: () => { viewMode = 'week'; loadEvents(); } },
			{ key: '3', description: 'Month view', handler: () => { viewMode = 'month'; loadEvents(); } },
			{ key: '4', description: 'Agenda view', handler: () => { viewMode = 'agenda'; loadEvents(); } },
			{ key: 'c', description: 'Create event', handler: () => openCreateModal() },
		])
	);

	// ── Derived ───────────────────────────────────────────────────────────────
	const dateRange = $derived(buildDateRange(viewMode, currentDate));

	const weekDates = $derived((): Date[] => {
		if (viewMode !== 'week') return [];
		const dates: Date[] = [];
		const current = new Date(dateRange.start);
		for (let i = 0; i < 7; i++) {
			dates.push(new Date(current));
			current.setDate(current.getDate() + 1);
		}
		return dates;
	});

	const headerText = $derived((): string => {
		if (viewMode === 'day') {
			return currentDate.toLocaleString('default', {
				weekday: 'long',
				month: 'long',
				day: 'numeric',
				year: 'numeric'
			});
		}
		if (viewMode === 'week') {
			const start = dateRange.start;
			const end = dateRange.end;
			const startMonth = start.toLocaleString('default', { month: 'short' });
			const endMonth = end.toLocaleString('default', { month: 'short' });
			const year = start.getFullYear();
			if (startMonth === endMonth) {
				return `${startMonth} ${start.getDate()} - ${end.getDate()}, ${year}`;
			}
			return `${startMonth} ${start.getDate()} - ${endMonth} ${end.getDate()}, ${year}`;
		}
		if (viewMode === 'agenda') return 'Upcoming Events';
		return currentDate.toLocaleString('default', { month: 'long', year: 'numeric' });
	});

	// ── Bootstrap ─────────────────────────────────────────────────────────────
	onMount(async () => {
		currentDate = new Date();
		await Promise.all([
			loadConnectionStatus(),
			loadEvents(),
			loadSyncStats(),
			loadUpcomingEvents()
		]);
		isLoading = false;
	});

	// Reload events when filter changes
	$effect(() => {
		selectedMeetingType; // track dependency
		loadEvents();
	});

	// ── Data Loaders ──────────────────────────────────────────────────────────
	async function loadUpcomingEvents() {
		try {
			const res = await apiClient.get('/calendar/upcoming');
			const data = res.ok ? await res.json() : [];
			upcomingEvents = Array.isArray(data) ? data : [];
		} catch {
			upcomingEvents = [];
		}
	}

	async function loadSyncStats() {
		try {
			const res = await apiClient.get('/calendar/stats');
			if (res.ok) {
				const data = await res.json();
				syncStats = {
					totalEvents: data.total_events || 0,
					googleEvents: data.google_events || 0,
					localEvents: data.local_events || 0,
					dateRange: data.date_range || null,
					lastSync: data.last_sync || null
				};
			}
		} catch {
			// stats unavailable — non-critical
		}
	}

	async function loadConnectionStatus() {
		try {
			const status = await getCalendarConnectionStatus();
			connectionStatus = { connected: status.connected, email: status.email };
		} catch {
			connectionStatus = { connected: false };
		}
	}

	async function loadEvents() {
		try {
			const result = await api.getCalendarEvents({
				start: dateRange.start.toISOString(),
				end: dateRange.end.toISOString(),
				meetingType: selectedMeetingType || undefined
			});
			events = Array.isArray(result) ? result : [];
		} catch (e) {
			events = [];
			toast.error(e instanceof Error && e.message ? e.message : 'Failed to load calendar events');
		}
	}

	// ── Navigation ────────────────────────────────────────────────────────────
	async function navigatePrev() {
		const d = new Date(currentDate);
		if (viewMode === 'day') d.setDate(d.getDate() - 1);
		else if (viewMode === 'week') d.setDate(d.getDate() - 7);
		else d.setMonth(d.getMonth() - 1);
		currentDate = d;
		selectedDay = new Date(d);
		await tick();
		await loadEvents();
	}

	async function navigateNext() {
		const d = new Date(currentDate);
		if (viewMode === 'day') d.setDate(d.getDate() + 1);
		else if (viewMode === 'week') d.setDate(d.getDate() + 7);
		else d.setMonth(d.getMonth() + 1);
		currentDate = d;
		selectedDay = new Date(d);
		await tick();
		await loadEvents();
	}

	async function navigateToday() {
		currentDate = new Date();
		selectedDay = new Date();
		await tick();
		await loadEvents();
	}

	async function goToDayView(date: Date) {
		currentDate = new Date(date);
		selectedDay = new Date(date);
		viewMode = 'day';
		await tick();
		await loadEvents();
	}

	async function jumpToFirstEvent() {
		if (syncStats?.dateRange?.from) {
			currentDate = new Date(syncStats.dateRange.from);
			await tick();
			await loadEvents();
		}
	}

	async function jumpToLatestEvent() {
		if (syncStats?.dateRange?.to) {
			currentDate = new Date(syncStats.dateRange.to);
			await tick();
			await loadEvents();
		}
	}

	// ── Google Sync ───────────────────────────────────────────────────────────
	async function connectCalendar() {
		try {
			const result = await getCalendarAuthUrl();
			if (result.auth_url) window.location.href = result.auth_url;
		} catch (error) {
			console.error('Error initiating calendar auth:', error);
		}
	}

	async function syncCalendar() {
		isSyncing = true;
		try {
			const result = (await api.syncCalendar()) as {
				message: string;
				synced_count: number;
				details?: { total_events?: number; date_range?: string };
			};
			await loadEvents();
			await loadSyncStats();
			if (result?.details) {
				let dr: { from: string | null; to: string | null } | null = null;
				if (result.details.date_range) {
					const parts = result.details.date_range.split(' - ');
					dr = { from: parts[0] || null, to: parts[1] || null };
				}
				syncStats = {
					totalEvents: result.details.total_events || result.synced_count || 0,
					dateRange: dr,
					lastSync: new Date().toISOString(),
					googleEvents: 0,
					localEvents: 0
				};
			}
		} catch (error) {
			console.error('Error syncing calendar:', error);
		} finally {
			isSyncing = false;
		}
	}

	// ── Event Modal ───────────────────────────────────────────────────────────
	function openEventModal(event: CalendarEvent) {
		selectedEvent = event;
		showEventModal = true;
	}

	function closeEventModal() {
		selectedEvent = null;
		showEventModal = false;
	}

	// ── Create/Edit Modal ─────────────────────────────────────────────────────
	function openCreateModal(date?: Date) {
		editingEvent = null;
		const d = date || new Date();
		const dateStr = d.toISOString().split('T')[0];
		formData = {
			title: '',
			description: '',
			start_date: dateStr,
			start_time: '09:00',
			end_date: dateStr,
			end_time: '10:00',
			all_day: false,
			location: '',
			meeting_type: '',
			meeting_link: ''
		};
		formError = '';
		showCreateModal = true;
	}

	function openCreateModalAtHour(date: Date, hour: number) {
		editingEvent = null;
		const dateStr = date.toISOString().split('T')[0];
		const startTime = hour.toString().padStart(2, '0') + ':00';
		const endTime = ((hour + 1) % 24).toString().padStart(2, '0') + ':00';
		formData = {
			title: '',
			description: '',
			start_date: dateStr,
			start_time: startTime,
			end_date: dateStr,
			end_time: endTime,
			all_day: false,
			location: '',
			meeting_type: '',
			meeting_link: ''
		};
		formError = '';
		showCreateModal = true;
	}

	function openEditModal(event: CalendarEvent) {
		editingEvent = event;
		const start = new Date(event.start_time);
		const end = new Date(event.end_time);
		formData = {
			title: event.title || '',
			description: event.description || '',
			start_date: start.toISOString().split('T')[0],
			start_time: start.toTimeString().slice(0, 5),
			end_date: end.toISOString().split('T')[0],
			end_time: end.toTimeString().slice(0, 5),
			all_day: event.all_day || false,
			location: event.location || '',
			meeting_type: event.meeting_type || '',
			meeting_link: event.meeting_link || ''
		};
		formError = '';
		showCreateModal = true;
	}

	async function saveEvent() {
		if (!formData.title.trim()) {
			formError = 'Title is required';
			return;
		}
		isSaving = true;
		formError = '';
		try {
			const startTime = new Date(`${formData.start_date}T${formData.start_time}:00`).toISOString();
			const endTime = new Date(`${formData.end_date}T${formData.end_time}:00`).toISOString();
			const payload = {
				title: formData.title,
				description: formData.description || undefined,
				start_time: startTime,
				end_time: endTime,
				all_day: formData.all_day,
				location: formData.location || undefined,
				meeting_type: formData.meeting_type || undefined,
				meeting_link: formData.meeting_link || undefined,
				sync_to_google: connectionStatus?.connected || false
			};
			if (editingEvent) {
				const res = await apiClient.put(`/calendar/events/${editingEvent.id}`, payload);
				if (!res.ok) throw new Error('Failed to update event');
			} else {
				const res = await apiClient.post('/calendar/events', payload);
				if (!res.ok) throw new Error('Failed to create event');
			}
			showCreateModal = false;
			await loadEvents();
		} catch (error) {
			console.error('Error saving event:', error);
			formError = 'Failed to save event';
		} finally {
			isSaving = false;
		}
	}

	// ── Delete Confirmation ──────────────────────────────────────────────────
	let deleteTarget = $state<CalendarEvent | null>(null);
	let isDeleting = $state(false);

	function deleteEvent(event: CalendarEvent) {
		deleteTarget = event;
	}

	async function confirmDelete() {
		if (!deleteTarget) return;
		isDeleting = true;
		try {
			const res = await apiClient.delete(`/calendar/events/${deleteTarget.id}`);
			if (res.ok) {
				const wasOpen = selectedEvent?.id === deleteTarget.id;
				deleteTarget = null;
				if (wasOpen) closeEventModal();
				await loadEvents();
			}
		} catch (error) {
			console.error('Error deleting event:', error);
		} finally {
			isDeleting = false;
		}
	}

	function cancelDelete() {
		if (isDeleting) return;
		deleteTarget = null;
	}

	function handleEditFromModal(event: CalendarEvent) {
		closeEventModal();
		openEditModal(event);
	}
</script>

<div class="cal-page">
	<CalendarStatusBanner
		{upcomingEvents}
		{syncStats}
		{events}
		{connectionStatus}
		onOpenEventModal={openEventModal}
		onJumpToFirstEvent={jumpToFirstEvent}
		onJumpToLatestEvent={jumpToLatestEvent}
		onConnect={connectCalendar}
	/>

	<CalendarToolbar
		{viewMode}
		headerText={headerText()}
		bind:selectedMeetingType
		{showSidebar}
		onNavigatePrev={navigatePrev}
		onNavigateNext={navigateNext}
		onNavigateToday={navigateToday}
		onViewModeChange={(mode) => { viewMode = mode; loadEvents(); }}
		onMeetingTypeChange={(type) => { selectedMeetingType = type; }}
		onToggleSidebar={() => (showSidebar = !showSidebar)}
		onCreateEvent={() => openCreateModal()}
		onSync={syncCalendar}
		{isSyncing}
		isConnected={connectionStatus?.connected ?? false}
		lastSync={syncStats?.lastSync ?? null}
	/>

	<!-- Main Content -->
	<div class="cal-main">
		<div class="cal-sidebar-wrap" class:cal-sidebar-wrap--collapsed={!showSidebar}>
			<CalendarSidebar
				bind:selectedDay
				{events}
				onSelectDay={(date) => { selectedDay = new Date(date); }}
				onGoToDayView={goToDayView}
				onOpenCreateModal={openCreateModal}
				onOpenEventModal={openEventModal}
			/>
		</div>

		<!-- Calendar Grid -->
		<div class="cal-grid-scroll" bind:this={timeGridRef}>
			{#if isLoading}
				<div
					class="cal-skel"
					aria-busy="true"
					aria-label="Loading calendar"
					out:fade={{ duration: 180 }}
				>
					<div class="cal-skel__header">
						{#each Array(7) as _, i (i)}
							<div class="cal-skel__day-head"></div>
						{/each}
					</div>
					<div class="cal-skel__grid">
						{#each Array(8) as _, row (row)}
							<div class="cal-skel__row">
								<div class="cal-skel__gutter"></div>
								{#each Array(7) as _, col (col)}
									<div class="cal-skel__cell">
										{#if (row + col) % 4 === 0}
											<div class="cal-skel__chip cal-skel__chip--{(row + col) % 3}"></div>
										{/if}
									</div>
								{/each}
							</div>
						{/each}
					</div>
				</div>
			{:else if viewMode === 'week'}
				<CalendarWeekDayView
					mode="week"
					{currentDate}
					weekDates={weekDates()}
					{events}
					{currentTime}
					onOpenEventModal={openEventModal}
					onOpenCreateModalAtHour={openCreateModalAtHour}
				/>
			{:else if viewMode === 'day'}
				<CalendarWeekDayView
					mode="day"
					{currentDate}
					weekDates={[]}
					{events}
					{currentTime}
					onOpenEventModal={openEventModal}
					onOpenCreateModalAtHour={openCreateModalAtHour}
				/>
			{:else if viewMode === 'month'}
				<CalendarMonthView
					{currentDate}
					{selectedDay}
					{events}
					{dateRange}
					onSelectDay={(date) => { selectedDay = new Date(date); }}
					onGoToDayView={goToDayView}
					onOpenCreateModal={openCreateModal}
					onOpenEventModal={openEventModal}
				/>
			{:else}
				<CalendarAgendaView
					{events}
					{syncStats}
					onOpenEventModal={openEventModal}
					onJumpToFirstEvent={jumpToFirstEvent}
				/>
			{/if}
		</div>
	</div>
</div>

<!-- Event Detail Modal -->
{#if showEventModal && selectedEvent}
	<CalendarEventModal
		event={selectedEvent}
		onClose={closeEventModal}
		onEdit={handleEditFromModal}
		onDelete={deleteEvent}
	/>
{/if}

<!-- Create/Edit Event Modal -->
{#if showCreateModal}
	<CalendarEventForm
		{editingEvent}
		bind:formData
		{isSaving}
		{formError}
		{connectionStatus}
		onSave={saveEvent}
		onClose={() => (showCreateModal = false)}
	/>
{/if}

<!-- Delete Confirmation Modal -->
{#if deleteTarget}
	<div
		class="bos-modal-overlay"
		role="presentation"
		onclick={cancelDelete}
		onkeydown={(e) => { if (e.key === 'Escape') cancelDelete(); }}
	>
		<div
			class="bos-modal bos-modal--sm"
			role="dialog"
			aria-modal="true"
			aria-labelledby="cal-delete-title"
			onclick={(e) => e.stopPropagation()}
			onkeydown={(e) => e.stopPropagation()}
		>
			<header class="bos-modal-header">
				<div>
					<h2 id="cal-delete-title" class="bos-modal-title">Delete event?</h2>
					<p class="bos-modal-subtitle">This can't be undone.</p>
				</div>
			</header>
			<div class="bos-modal-body">
				<p style="color: var(--dt2); font-size: 0.875rem; margin: 0;">
					Delete <strong style="color: var(--dt);">{deleteTarget.title || 'Untitled event'}</strong>?
					{#if connectionStatus?.connected && deleteTarget.google_event_id}
						<br /><span style="color: var(--dt3); font-size: 0.8rem;">
							This will also remove it from Google Calendar.
						</span>
					{/if}
				</p>
			</div>
			<footer class="bos-modal-footer">
				<PillButton variant="ghost" size="sm" onclick={cancelDelete} disabled={isDeleting}>
					Cancel
				</PillButton>
				<PillButton variant="danger" size="sm" onclick={confirmDelete} loading={isDeleting}>
					Delete
				</PillButton>
			</footer>
		</div>
	</div>
{/if}

<style>
	/* ── Page shell ───────────────────────────────────────────────
	   Single scroll surface = .cal-grid-scroll. Banner + toolbar are
	   sticky at top of the main content area; sidebar scrolls inside
	   itself. Avoids the dual-scroll-container trap that hid the
	   bottom of the time grid on shorter viewports. */
	.cal-page {
		flex: 1 1 0%;
		min-height: 0;
		min-width: 0;
		display: flex;
		flex-direction: column;
		overflow: hidden;
	}

	.cal-main {
		flex: 1 1 0%;
		min-height: 0;
		display: flex;
	}

	.cal-grid-scroll {
		flex: 1 1 auto;
		min-width: 0;
		overflow: auto;
	}

	.cal-sidebar-wrap {
		width: 14rem;
		overflow: hidden;
		flex-shrink: 0;
		transition: width 0.25s cubic-bezier(0.4, 0, 0.2, 1);
	}

	.cal-sidebar-wrap--collapsed {
		width: 0;
	}

	/* ── Skeleton loader ─────────────────────────────────────────── */
	.cal-skel {
		padding: 0.5rem 0;
		min-width: 800px;
	}

	.cal-skel__header {
		display: grid;
		grid-template-columns: 3.5rem repeat(7, 1fr);
		padding: 0.5rem 0;
		border-bottom: 1px solid var(--dbd);
	}

	.cal-skel__header::before {
		content: '';
	}

	.cal-skel__day-head {
		height: 2.4rem;
		margin: 0 0.5rem;
		border-radius: 6px;
		background: var(--dbg2);
		animation: cal-pulse 1.4s ease-in-out infinite;
	}

	.cal-skel__grid {
		display: flex;
		flex-direction: column;
	}

	.cal-skel__row {
		display: grid;
		grid-template-columns: 3.5rem repeat(7, 1fr);
		height: 60px;
		border-bottom: 1px solid var(--dbd2);
	}

	.cal-skel__gutter {
		width: 3.5rem;
	}

	.cal-skel__cell {
		border-left: 1px solid var(--dbd2);
		padding: 4px;
		position: relative;
	}

	.cal-skel__chip {
		height: calc(100% - 8px);
		border-radius: 6px;
		background: var(--dbg2);
		animation: cal-pulse 1.4s ease-in-out infinite;
	}

	.cal-skel__chip--0 {
		width: 80%;
		animation-delay: 0s;
	}

	.cal-skel__chip--1 {
		width: 60%;
		animation-delay: 0.2s;
	}

	.cal-skel__chip--2 {
		width: 90%;
		animation-delay: 0.4s;
	}

	@keyframes cal-pulse {
		0%, 100% { opacity: 0.55; }
		50% { opacity: 0.95; }
	}
</style>
