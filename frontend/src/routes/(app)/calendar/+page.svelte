<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { api } from '$lib/api';
	import { currentWorkspace } from '$lib/stores/workspaces';
	import type { CalendarEvent } from '$lib/api/calendar/types';
	import { getCalendarConnectionStatus, getCalendarAuthUrl } from '$lib/api/calendar';
	import {
		deleteContentItem,
		getContent,
		updateContentItem,
		type ContentItem,
		type ContentItemInput,
		type ContentStatus,
		type ContentType
	} from '$lib/api/content';
	import { listMembers, type WorkspaceMember } from '$lib/api/workspace-admin';
	import ContentCalendarEditor from '$lib/components/calendar/ContentCalendarEditor.svelte';
	import CalendarMonthView from '$lib/components/calendar/CalendarMonthView.svelte';
	import CalendarWeekDayView from '$lib/components/calendar/CalendarWeekDayView.svelte';
	import {
		buildDateRange,
		filterEventsBySelectedMembers,
		getEventDisplayStart,
		getEventsForDate,
		sanitizeHtml,
		formatTimeFmt,
		type ViewMode
	} from '$lib/components/calendar/calendarUtils';
	import {
		contentCalendarColors,
		contentCalendarSources,
		type ContentCalendarSource,
		contentProfiles as deriveContentProfiles,
		contentToCalendarEvents,
		contentTypeLabel,
		filterCalendarContent,
		missedContent
	} from '$lib/modules/content/calendarProjection';
	import { getPreferences, updatePreferences, type WorkspacePreferences } from '$lib/api/preferences';
	import { Plus, Loader2, X, RefreshCw, ChevronLeft, ChevronRight, Trash2, MapPin, Clock, Link2, Check, Settings } from 'lucide-svelte';

	let view = $state<ViewMode>('month');
	let calendarMode = $state<'appointments' | 'content'>('appointments');
	let events = $state<CalendarEvent[]>([]);
	let contentItems = $state<ContentItem[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let syncing = $state(false);

	let currentDate = $state(new Date());
	let selectedDay = $state(new Date());
	let currentTime = $state(new Date());

	let googleConnected = $state(false);
	let googleAuthorized = $state(false);
	let googleStatusLoaded = $state(false);
	let googleAccount = $state('');
	let connecting = $state(false);
	let lastSynced = $state<Date | null>(null);
	const syncedLabel = $derived.by(() => {
		if (syncing) return 'Syncing…';
		if (!lastSynced) return googleConnected && googleAuthorized ? 'Connected' : '';
		const s = Math.floor((currentTime.getTime() - lastSynced.getTime()) / 1000);
		if (s < 60) return 'Synced just now';
		if (s < 3600) return `Synced ${Math.floor(s / 60)}m ago`;
		if (s < 86400) return `Synced ${Math.floor(s / 3600)}h ago`;
		return `Synced ${Math.floor(s / 86400)}d ago`;
	});

	// Team calendars: see other workspace members' calendars side-by-side, color-coded per person.
	// NOTE: Google Calendar connection is PER-USER. Each member connects their own Google account
	// (Connect Google authorizes only the signed-in user); OAuth tokens are stored per user_id
	// server-side, so this feed only surfaces events the workspace already shares — it never
	// reuses one member's Google token to read another member's calendar.
	const PALETTE = ['#6366f1', '#34d399', '#f59e0b', '#ec4899', '#06b6d4', '#f43f5e', '#8b5cf6', '#14b8a6', '#eab308', '#3b82f6'];
	let members = $state<WorkspaceMember[]>([]);
	let selectedMembers = $state<Set<string>>(new Set());
	const memberColors = $derived.by(() => {
		const m: Record<string, string> = {};
		members.forEach((mem, i) => { m[mem.user_id] = PALETTE[i % PALETTE.length]; });
		return m;
	});
	function memberName(uid: string): string {
		const mem = members.find((x) => x.user_id === uid);
		return mem?.name || mem?.email || 'Member';
	}
	function toggleMember(uid: string) {
		const s = new Set(selectedMembers);
		if (s.has(uid)) s.delete(uid); else s.add(uid);
		selectedMembers = s;
	}
	// Events filtered to the picked members (all picked by default).
	const visibleEvents = $derived(
		filterEventsBySelectedMembers(events, selectedMembers, members.length > 0)
	);

	let contentProfile = $state('all');
	let activeContentSources = $state<Set<ContentCalendarSource>>(new Set(contentCalendarSources.map((source) => source.id)));
	let contentColors = $state<Record<string, string>>({ ...contentCalendarColors });
	let colorWorkspaceKey = '';
	const CONTENT_STAGES: ContentStatus[] = ['idea', 'scripting', 'to_film', 'to_edit', 'client_review', 'approved', 'to_post', 'posted', 'archive'];
	const CONTENT_STAGE_LABELS: Record<string, string> = {
		idea: 'Idea',
		scripting: 'Scripting',
		to_film: 'To Film',
		to_edit: 'To Edit',
		client_review: 'Client Review',
		approved: 'Approved',
		to_post: 'To Post',
		posted: 'Posted',
		archive: 'Archive',
		draft: 'Draft',
		scheduled: 'Scheduled',
		published: 'Published'
	};
	const CONTENT_TYPES: ContentType[] = ['script', 'copywriting', 'carousel', 'image', 'video', 'post', 'reel', 'story', 'newsletter', 'podcast', 'thread', 'article', 'other'];
	const CONTENT_CHANNELS = ['Instagram', 'Instagram Stories', 'LinkedIn', 'TikTok', 'YouTube Shorts', 'YouTube', 'Email', 'Paid Ads', 'Website'];
	const CONTENT_WORKSTREAMS = ['Organic Content', 'LinkedIn Posts', 'Paid Ads', 'Trial Reels', 'Long Form', 'Email / Newsletter', 'Case Studies / Proof'];

	$effect(() => {
		const mode = $page.url.searchParams.get('mode');
		calendarMode = mode === 'content' ? 'content' : 'appointments';
	});
	const contentProfiles = $derived(deriveContentProfiles(contentItems));
	const visibleContentItems = $derived(filterCalendarContent(contentItems, contentProfile, Array.from(activeContentSources)));
	const missedContentItems = $derived(missedContent(visibleContentItems));
	const contentEvents = $derived(contentToCalendarEvents(visibleContentItems));
	const activeEvents = $derived(calendarMode === 'content' ? contentEvents : visibleEvents);

	const dateRange = $derived(buildDateRange(view, currentDate));
	const weekDates = $derived.by(() => {
		const weekStart = prefs?.week_start ?? 0; // 0=Sun, 1=Mon
		const start = new Date(currentDate);
		const offset = (currentDate.getDay() - weekStart + 7) % 7;
		start.setDate(currentDate.getDate() - offset);
		start.setHours(0, 0, 0, 0);
		return Array.from({ length: 7 }, (_, i) => {
			const x = new Date(start);
			x.setDate(start.getDate() + i);
			return x;
		});
	});

	const title = $derived.by(() => {
		if (view === 'month') return currentDate.toLocaleDateString(undefined, { month: 'long', year: 'numeric' });
		if (view === 'day') return currentDate.toLocaleDateString(undefined, { weekday: 'long', month: 'long', day: 'numeric' });
		if (view === 'week') {
			const a = weekDates[0], b = weekDates[6];
			const sameMonth = a.getMonth() === b.getMonth();
			return `${a.toLocaleDateString(undefined, { month: 'short', day: 'numeric' })} – ${b.toLocaleDateString(undefined, { month: sameMonth ? undefined : 'short', day: 'numeric', year: 'numeric' })}`;
		}
		return 'Upcoming';
	});

	const views: { id: ViewMode; label: string }[] = [
		{ id: 'month', label: 'Month' },
		{ id: 'week', label: 'Week' },
		{ id: 'day', label: 'Day' },
		{ id: 'agenda', label: 'Agenda' }
	];

	const agendaGroups = $derived.by(() => {
		const map = new Map<string, CalendarEvent[]>();
		for (const ev of [...activeEvents].sort((a, b) => getEventDisplayStart(a).getTime() - getEventDisplayStart(b).getTime())) {
			const key = getEventDisplayStart(ev).toDateString();
			if (!map.has(key)) map.set(key, []);
			map.get(key)!.push(ev);
		}
		return Array.from(map.entries());
	});

	// Create / detail modals
	let showCreate = $state(false);
	let creating = $state(false);
	let form = $state({ title: '', date: '', start: '09:00', end: '10:00', location: '', description: '' });
	let selectedEvent = $state<CalendarEvent | null>(null);
	let busyDelete = $state(false);
	let editingContent = $state<ContentItem | null>(null);
	let contentForm = $state<ContentItemInput>(emptyContentForm());
	let savingContent = $state(false);
	let deletingContent = $state(false);

	// Day preview popover: clicking a day in Month shows that whole day's events.
	let previewDay = $state<Date | null>(null);
	const previewEvents = $derived.by(() =>
		previewDay
			? getEventsForDate(activeEvents, previewDay).sort(
					(a, b) => new Date(a.start_time).getTime() - new Date(b.start_time).getTime()
				)
			: []
	);

	let lastKey = $state('');
	$effect(() => {
		const r = dateRange;
		const key = `${$currentWorkspace?.id ?? ''}|${view}|${calendarMode}|${r.start.toISOString()}|${r.end.toISOString()}`;
		if (key !== lastKey) { lastKey = key; load(); }
	});

	onMount(async () => {
		await loadPrefs();
		try {
			const s = await getCalendarConnectionStatus();
			googleConnected = !!s.connected;
			googleAuthorized = !!s.authorized;
			googleAccount = s.email ?? '';
			lastSynced = s.last_sync ? new Date(s.last_sync) : null;
		} catch {
			googleConnected = false;
			googleAuthorized = false;
			googleAccount = '';
		} finally {
			googleStatusLoaded = true;
		}
		await load();
		try {
			const res = await listMembers($currentWorkspace?.id ?? '');
			members = Array.isArray(res) ? res : ((res as { members?: WorkspaceMember[] })?.members ?? []);
			selectedMembers = new Set(members.map((m) => m.user_id));
		} catch { members = []; }
	});

	async function load() {
		loading = true; error = null;
		try {
			const r = dateRange;
			if (calendarMode === 'appointments') {
				events = await api.getCalendarEvents({ start: r.start.toISOString(), end: r.end.toISOString() });
			} else {
				const res = await getContent(undefined, $currentWorkspace?.id);
				contentItems = res.items;
			}
		} catch (e) { error = e instanceof Error ? e.message : 'Failed to load calendar'; }
		finally { loading = false; }
	}

	function step(dir: number) {
		const d = new Date(currentDate);
		if (view === 'month') d.setMonth(d.getMonth() + dir);
		else if (view === 'week') d.setDate(d.getDate() + 7 * dir);
		else if (view === 'day') d.setDate(d.getDate() + dir);
		else d.setDate(d.getDate() + 30 * dir);
		currentDate = d;
	}
	function goToday() { currentDate = new Date(); selectedDay = new Date(); }
	function isContentEvent(ev: CalendarEvent): boolean { return ev.user_id.startsWith('contentos') || ev.id.startsWith('content-'); }
	function eventColor(ev: CalendarEvent): string {
		if (calendarMode === 'content') return contentColors[ev.user_id] || contentColors['contentos-organic-content'] || contentCalendarColors['contentos-organic-content'];
		return ev.color_hex || memberColors[ev.user_id] || (ev.source === 'google' ? '#34a853' : '#6366f1');
	}

	function fmtDateInput(d: Date): string {
		return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
	}
	function contentColorStorageKey() {
		return `contentos-calendar-colors:${$currentWorkspace?.id ?? 'no-workspace'}`;
	}
	function loadContentColors() {
		if (typeof localStorage === 'undefined') return;
		try {
			const parsed = JSON.parse(localStorage.getItem(contentColorStorageKey()) || '{}') as Record<string, string>;
			contentColors = { ...contentCalendarColors, ...parsed };
		} catch {
			contentColors = { ...contentCalendarColors };
		}
	}
	function setContentColor(source: ContentCalendarSource, color: string) {
		contentColors = { ...contentColors, [source]: color };
		if (typeof localStorage !== 'undefined') localStorage.setItem(contentColorStorageKey(), JSON.stringify(contentColors));
	}
	function toggleContentSource(source: ContentCalendarSource) {
		const next = new Set(activeContentSources);
		if (next.has(source)) next.delete(source);
		else next.add(source);
		activeContentSources = next;
	}
	async function moveCalendarContentEvent(ev: CalendarEvent, date: Date) {
		if (!ev.context_id) return;
		const item = contentItems.find((candidate) => candidate.id === ev.context_id);
		if (!item) return;
		const field = ev.id.startsWith('content-film-') ? 'due_date' : 'publish_date';
		error = null;
		try {
			const updated = await updateContentItem(item.id, { ...contentFormFromItem(item), [field]: fmtDateInput(date) }, $currentWorkspace?.id);
			contentItems = contentItems.map((candidate) => (candidate.id === updated.id ? updated : candidate));
			selectedDay = date;
			previewDay = null;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to move content date';
		}
	}
	function openCreate(date?: Date, hour?: number) {
		const d = date ?? selectedDay ?? new Date();
		const dur = prefs?.default_event_minutes ?? 60;
		const startH = hour != null ? hour : 9;
		const startMin = startH * 60;
		const endMin = startMin + dur;
		const hhmm = (m: number) => `${String(Math.floor(m / 60) % 24).padStart(2, '0')}:${String(m % 60).padStart(2, '0')}`;
		form = { title: '', date: fmtDateInput(d), start: hhmm(startMin), end: hhmm(endMin), location: '', description: '' };
		showCreate = true;
	}

	// Manual refresh button. With auto-sync running this is rarely needed, but kept
	// as an explicit "pull now" affordance; surfaces errors since the user asked.
	async function syncGoogle() {
		if (syncing) return;
		syncing = true; error = null;
		try {
			const s = await getCalendarConnectionStatus();
			googleConnected = !!s.connected;
			googleAuthorized = !!s.authorized;
			googleAccount = s.email ?? '';
			if (!googleConnected) throw new Error('Google Calendar is not connected on this BusinessOS instance.');
			if (!googleAuthorized) throw new Error('Google Calendar permission is required. Reconnect Google Calendar and approve calendar access.');
			await api.syncCalendar();
			await load();
			lastSynced = new Date();
		}
		catch (e) { error = e instanceof Error ? e.message : 'Sync failed'; }
		finally { syncing = false; }
	}

	// Best-effort background sync: never blocks the UI, never surfaces errors.
	// Reuses the `syncing` flag so the manual "Sync Google" button shows its
	// spinner while a background pull is in flight, but the user need not click it.
	async function backgroundSync() {
		if (!googleConnected || !googleAuthorized || syncing) return;
		syncing = true;
		try { await api.syncCalendar(); await load(); lastSynced = new Date(); }
		catch { /* background sync is best-effort — swallow errors */ }
		finally { syncing = false; }
	}

	// Automatic Google sync — the user never has to click "Sync Google":
	//  • periodic background pull (~every 5 min) while the page is mounted
	//  • one-shot pull whenever the Google connection flips true (e.g. just connected)
	// The interval is torn down on destroy via the $effect cleanup.
	const AUTO_SYNC_MS = 5 * 60 * 1000;
	let prevConnected = false;
	$effect(() => {
		const connected = googleConnected && googleAuthorized;
		if (connected && !prevConnected) backgroundSync();
		prevConnected = connected;

		if (!connected) return;
		const id = setInterval(backgroundSync, AUTO_SYNC_MS);
		return () => clearInterval(id);
	});

	// Tick the clock every 30s so the "Synced Xm ago" label counts up between
	// background syncs and the week/day current-time line stays accurate.
	$effect(() => {
		const id = setInterval(() => (currentTime = new Date()), 30_000);
		return () => clearInterval(id);
	});

	// ── Settings (workspace preferences primitive) ───────────────────────
	let prefs = $state<WorkspacePreferences | null>(null);
	let showSettings = $state(false);
	let savingPrefs = $state(false);
	let prefsForm = $state({
		timezone: 'America/New_York', date_format: 'MM/DD/YYYY', time_format: '12h' as '12h' | '24h',
		week_start: 0, working_hours_start: 9, working_hours_end: 17, default_event_minutes: 60,
		default_view: 'month' as ViewMode
	});
	let prefsApplied = false;

	async function loadPrefs() {
		try {
			const p = await getPreferences();
			prefs = p;
			const cal = (p.calendar ?? {}) as { default_view?: ViewMode };
			prefsForm = {
				timezone: p.timezone, date_format: p.date_format, time_format: p.time_format,
				week_start: p.week_start, working_hours_start: p.working_hours_start,
				working_hours_end: p.working_hours_end, default_event_minutes: p.default_event_minutes,
				default_view: cal.default_view ?? 'month'
			};
			// Apply default view once, only if the user hasn't navigated yet.
			if (!prefsApplied) { prefsApplied = true; if (cal.default_view) view = cal.default_view; }
		} catch { /* no workspace / not critical */ }
	}

	async function savePrefs() {
		savingPrefs = true;
		try {
			const updated = await updatePreferences({
				timezone: prefsForm.timezone, date_format: prefsForm.date_format,
				time_format: prefsForm.time_format, week_start: prefsForm.week_start,
				working_hours_start: prefsForm.working_hours_start, working_hours_end: prefsForm.working_hours_end,
				default_event_minutes: prefsForm.default_event_minutes,
				calendar: { ...(prefs?.calendar ?? {}), default_view: prefsForm.default_view }
			});
			prefs = updated;
			showSettings = false;
		} catch (e) { error = e instanceof Error ? e.message : 'Failed to save settings'; }
		finally { savingPrefs = false; }
	}

	const use24h = $derived(prefs?.time_format === '24h');

	$effect(() => {
		const key = contentColorStorageKey();
		if (key === colorWorkspaceKey) return;
		colorWorkspaceKey = key;
		loadContentColors();
	});

	async function connectGoogle() {
		connecting = true; error = null;
		try {
			const { auth_url } = await getCalendarAuthUrl();
			const el = (window as unknown as { electron?: { shell?: { openExternal?: (u: string) => Promise<void> } } }).electron;
			if (el?.shell?.openExternal) await el.shell.openExternal(auth_url);
			else window.open(auth_url, '_blank');
		} catch (e) { error = e instanceof Error ? e.message : 'Could not start Google connect'; }
		finally { connecting = false; }
	}

	async function create(e: Event) {
		e.preventDefault();
		if (!form.title.trim() || !form.date) return;
		creating = true; error = null;
		try {
			const startISO = new Date(`${form.date}T${form.start}`).toISOString();
			const endISO = new Date(`${form.date}T${form.end}`).toISOString();
			await api.createCalendarEvent({
				title: form.title.trim(), start_time: startISO, end_time: endISO,
				location: form.location.trim() || undefined, description: form.description.trim() || undefined
			});
			showCreate = false;
			await load();
		} catch (e) { error = e instanceof Error ? e.message : 'Failed to create event'; }
		finally { creating = false; }
	}

	async function removeEvent() {
		if (!selectedEvent) return;
		busyDelete = true;
		try {
			await api.deleteCalendarEvent(selectedEvent.id);
			events = events.filter((x) => x.id !== selectedEvent!.id);
			selectedEvent = null;
		} catch (e) { error = e instanceof Error ? e.message : 'Failed to delete'; }
		finally { busyDelete = false; }
	}

	function emptyContentForm(): ContentItemInput {
		return {
			title: '',
			content_type: 'post',
			status: 'idea',
			hook: '',
			body: '',
			caption: '',
			cta: '',
			channel: '',
			link: '',
			category: 'Organic Content',
			theme: '',
			client: '',
			campaign: '',
			owner: '',
			editor: '',
			priority: 'normal',
			due_date: '',
			publish_date: '',
			asset_link: '',
			review_link: '',
			revision_notes: '',
			notes: '',
			views: 0,
			reach: 0,
			likes: 0,
			comments: 0,
			saves: 0,
			shares: 0,
			reposts: 0,
			follows: 0,
			profile_activity: 0,
			accounts_engaged: 0,
			avg_watch_time_seconds: 0,
			retention_rate: 0,
			analytics_notes: ''
		};
	}

	function contentFormFromItem(item: ContentItem): ContentItemInput {
		return {
			title: item.title || '',
			content_type: item.content_type || 'post',
			status: item.status || 'idea',
			hook: item.hook || '',
			body: item.body || '',
			caption: item.caption || '',
			cta: item.cta || '',
			channel: item.channel || '',
			link: item.link || '',
			category: item.category || contentTypeLabel(item),
			theme: item.theme || '',
			client: item.client || '',
			campaign: item.campaign || '',
			owner: item.owner || '',
			editor: item.editor || '',
			priority: item.priority || 'normal',
			due_date: item.due_date || '',
			publish_date: item.publish_date || '',
			asset_link: item.asset_link || '',
			review_link: item.review_link || '',
			revision_notes: item.revision_notes || '',
			notes: item.notes || '',
			views: item.views || 0,
			reach: item.reach || 0,
			likes: item.likes || 0,
			comments: item.comments || 0,
			saves: item.saves || 0,
			shares: item.shares || 0,
			reposts: item.reposts || 0,
			follows: item.follows || 0,
			profile_activity: item.profile_activity || 0,
			accounts_engaged: item.accounts_engaged || 0,
			avg_watch_time_seconds: item.avg_watch_time_seconds || 0,
			retention_rate: item.retention_rate || 0,
			analytics_notes: item.analytics_notes || ''
		};
	}

	function openContentEditor(item: ContentItem) {
		editingContent = item;
		contentForm = contentFormFromItem(item);
		selectedEvent = null;
		previewDay = null;
	}

	function openEvent(ev: CalendarEvent) {
		if (isContentEvent(ev) && ev.context_id) {
			const item = contentItems.find((x) => x.id === ev.context_id);
			if (item) {
				openContentEditor(item);
				return;
			}
		}
		selectedEvent = ev;
	}

	async function saveContent(e: Event) {
		e.preventDefault();
		if (!editingContent || !contentForm.title.trim()) return;
		savingContent = true;
		error = null;
		try {
			const updated = await updateContentItem(editingContent.id, contentForm, $currentWorkspace?.id);
			contentItems = contentItems.map((item) => (item.id === updated.id ? updated : item));
			editingContent = null;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to save content card';
		} finally {
			savingContent = false;
		}
	}

	async function deleteEditingContent() {
		if (!editingContent) return;
		if (!confirm(`Delete "${editingContent.title}" from ContentOS? This removes it from the calendar too.`)) return;
		deletingContent = true;
		error = null;
		try {
			await deleteContentItem(editingContent.id, $currentWorkspace?.id);
			contentItems = contentItems.filter((item) => item.id !== editingContent?.id);
			editingContent = null;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to delete content card';
		} finally {
			deletingContent = false;
		}
	}

	function fmtTime(s: string): string { return formatTimeFmt(s, use24h); }
	function fmtDayLabel(s: string): string {
		const d = new Date(s); const t = new Date().toDateString();
		const tm = new Date(Date.now() + 86400000).toDateString();
		if (d.toDateString() === t) return 'Today';
		if (d.toDateString() === tm) return 'Tomorrow';
		return d.toLocaleDateString(undefined, { weekday: 'long', month: 'short', day: 'numeric' });
	}
</script>

<div class="cal-root">
	<header class="topbar">
		<div class="left">
			<h1>{title}</h1>
			<div class="nav-btns">
				<button class="icon-btn" onclick={() => step(-1)} aria-label="Previous"><ChevronLeft size={18} /></button>
				<button class="today-btn" onclick={goToday}>Today</button>
				<button class="icon-btn" onclick={() => step(1)} aria-label="Next"><ChevronRight size={18} /></button>
			</div>
		</div>

		<div class="center">
			<div class="seg seg--mode">
				<button class="seg-btn" class:seg-btn--on={calendarMode === 'appointments'} onclick={() => (calendarMode = 'appointments')}>Appointments</button>
				<button class="seg-btn" class:seg-btn--on={calendarMode === 'content'} onclick={() => (calendarMode = 'content')}>Content Calendar</button>
			</div>
			<div class="seg">
				{#each views as v}
					<button class="seg-btn" class:seg-btn--on={view === v.id} onclick={() => (view = v.id)}>{v.label}</button>
				{/each}
			</div>
		</div>

		<div class="tools">
			{#if calendarMode === 'appointments' && googleConnected && googleAuthorized && syncedLabel}
				<button class="sync-status" onclick={syncGoogle} disabled={syncing} title="Auto-syncs with Google. Click to refresh now.">
					<RefreshCw size={12} class={syncing ? 'spin' : ''} />{syncedLabel}
				</button>
			{:else if calendarMode === 'appointments'}
				<button class="sync-status" onclick={connectGoogle} disabled={connecting} title={googleConnected ? 'Reconnect Google Calendar permissions' : 'Connect Google Calendar'}>
					<Link2 size={12} />{connecting ? 'Connecting...' : googleConnected ? 'Reconnect Google' : 'Connect Google'}
				</button>
			{/if}
			<button class="icon-btn" onclick={() => { loadPrefs(); showSettings = true; }} aria-label="Calendar settings" title="Calendar settings"><Settings size={17} /></button>
			{#if calendarMode === 'appointments'}
				<button class="btn btn--primary" onclick={() => openCreate()}><Plus size={16} strokeWidth={2.4} />New event</button>
			{:else}
				<a class="btn btn--primary" href="/content"><Plus size={16} strokeWidth={2.4} />Open ContentOS</a>
			{/if}
		</div>
	</header>

	{#if error}<div class="banner banner--error">{error}</div>{/if}
	{#if calendarMode === 'appointments' && googleStatusLoaded && !googleConnected}
		<div class="banner banner--google">
			<div>
				<strong>Google Calendar is not connected here.</strong>
				<span>Connect a Google account to pull appointments into this workspace calendar.</span>
			</div>
			<button class="btn btn--primary" onclick={connectGoogle} disabled={connecting}>
				<Link2 size={14} />{connecting ? 'Connecting...' : 'Connect Google'}
			</button>
		</div>
	{:else if calendarMode === 'appointments' && googleStatusLoaded && googleConnected && !googleAuthorized}
		<div class="banner banner--google banner--warning">
			<div>
				<strong>Google Calendar permission is required.</strong>
				<span>The account is signed in, but Google has not granted Calendar access. Reconnect and approve the requested permission.</span>
			</div>
			<button class="btn btn--primary" onclick={connectGoogle} disabled={connecting}>
				<Link2 size={14} />{connecting ? 'Connecting...' : 'Reconnect Google'}
			</button>
		</div>
	{/if}

	{#if calendarMode === 'content'}
		<div class="content-cal-filters">
			<span class="members-label">Content view:</span>
			<select class="content-profile-select" bind:value={contentProfile} aria-label="Content profile">
				<option value="all">All profiles</option>
				{#each contentProfiles as profile}<option value={profile}>{profile}</option>{/each}
			</select>
			<div class="content-cal-legend" aria-label="Content calendar sources">
				{#each contentCalendarSources as source (source.id)}
					<button
						type="button"
						class="content-source-chip"
						class:content-source-chip--off={!activeContentSources.has(source.id)}
						aria-pressed={activeContentSources.has(source.id)}
						onclick={() => toggleContentSource(source.id)}
					>
						<input
							class="source-color"
							type="color"
							value={contentColors[source.id] ?? contentCalendarColors[source.id]}
							aria-label={`Change ${source.label} color`}
							onclick={(event) => event.stopPropagation()}
							oninput={(event) => setContentColor(source.id, (event.currentTarget as HTMLInputElement).value)}
						/>
						<span>{source.label}</span>
					</button>
				{/each}
			</div>
			<span class="content-cal-note">{contentEvents.length} scheduled dates · {missedContentItems.length} missed</span>
		</div>
	{/if}

	{#if calendarMode === 'appointments' && members.length > 0}
		<!-- Member picker: always shown when the workspace has members so team
		     calendars can be toggled side-by-side, color-coded per person. -->
		<div class="members">
			<span class="members-label">Calendars:</span>
			{#each members as m (m.user_id)}
				<button class="mchip" class:mchip--off={!selectedMembers.has(m.user_id)} onclick={() => toggleMember(m.user_id)} title={memberName(m.user_id)} aria-pressed={selectedMembers.has(m.user_id)}>
					<span class="mdot" style="background:{memberColors[m.user_id]}"></span>
					{memberName(m.user_id)}
				</button>
			{/each}
			{#if selectedMembers.size > 0 && selectedMembers.size < members.length}
				<!-- Legend: which members are currently showing in week/day/month. -->
				<span class="legend">
					Showing:
					{#each members.filter((m) => selectedMembers.has(m.user_id)) as m (m.user_id)}
						<span class="legend-item"><span class="mdot" style="background:{memberColors[m.user_id]}"></span>{memberName(m.user_id)}</span>
					{/each}
				</span>
			{/if}
		</div>
	{/if}

	{#if loading}
		<div class="loading"><Loader2 class="spin" size={20} /> Loading…</div>
	{:else}
		<div class="body">
			{#if view === 'month'}
				<CalendarMonthView
					{currentDate} {selectedDay} events={activeEvents} {dateRange} memberColors={calendarMode === 'content' ? contentColors : memberColors} {use24h}
					weekStart={prefs?.week_start ?? 0}
					allowCreate={calendarMode === 'appointments'}
					allowEventDrag={calendarMode === 'content'}
					onSelectDay={(d) => { selectedDay = d; previewDay = d; }}
					onGoToDayView={(d) => { currentDate = d; view = 'day'; previewDay = null; }}
					onOpenCreateModal={(d) => { if (calendarMode === 'appointments') openCreate(d); }}
					onOpenEventModal={openEvent}
					onMoveEventDate={moveCalendarContentEvent}
				/>
			{:else if view === 'week' || view === 'day'}
				<CalendarWeekDayView
					mode={view}
					{currentDate} {weekDates} events={activeEvents} {currentTime} memberColors={calendarMode === 'content' ? contentColors : memberColors} {use24h}
					workingHoursStart={prefs?.working_hours_start ?? 9}
					workingHoursEnd={prefs?.working_hours_end ?? 17}
					onOpenEventModal={openEvent}
					onOpenCreateModalAtHour={(d, h) => { if (calendarMode === 'appointments') openCreate(d, h); }}
				/>
			{:else}
				<!-- Agenda -->
				<div class="agenda">
					{#if agendaGroups.length === 0}
						<div class="empty">
							<Clock size={24} strokeWidth={1.4} />
							<p>{calendarMode === 'content' ? 'No content dates scheduled in this range.' : 'Nothing scheduled.'}</p>
							{#if calendarMode === 'appointments'}<button class="btn btn--primary" onclick={() => openCreate()}><Plus size={15} />New event</button>{/if}
						</div>
					{:else}
						{#each agendaGroups as [day, evs] (day)}
							{@const isTodayGroup = new Date(day).toDateString() === new Date().toDateString()}
							<div class="ag-day" class:ag-day--today={isTodayGroup}>
								<div class="ag-head">
									<span class="ag-head-label">{fmtDayLabel(day)}</span>
									<span class="ag-head-date">{new Date(day).toLocaleDateString(undefined, { month: 'short', day: 'numeric' })}</span>
									{#if isTodayGroup}<span class="ag-head-today">Today</span>{/if}
									<span class="ag-head-count">{evs.length} event{evs.length === 1 ? '' : 's'}</span>
								</div>
								{#each evs as ev (ev.id)}
									<button class="ag-ev" onclick={() => openEvent(ev)}>
										<span class="ag-dot" style="background:{eventColor(ev)}"></span>
										<span class="ag-time">{ev.all_day ? 'All day' : `${fmtTime(ev.start_time)} - ${fmtTime(ev.end_time)}`}</span>
										<span class="ag-title">{ev.title ?? 'Untitled'}</span>
										{#if ev.location}<span class="ag-loc"><MapPin size={11} />{ev.location}</span>{/if}
										{#if isContentEvent(ev)}<span class="ag-src">ContentOS</span>{:else if ev.source === 'google'}<span class="ag-src">Google</span>{/if}
									</button>
								{/each}
							</div>
						{/each}
					{/if}
				</div>
			{/if}
		</div>
	{/if}
</div>

<!-- Create modal -->
{#if showCreate}
	<div class="overlay" role="button" tabindex="0" onclick={() => (showCreate = false)} onkeydown={(e) => e.key === 'Escape' && (showCreate = false)}>
		<div class="modal" role="dialog" tabindex="-1" onclick={(e) => e.stopPropagation()} onkeydown={() => {}}>
			<div class="modal-head"><h2>New event</h2><button class="card-x" onclick={() => (showCreate = false)} aria-label="Close"><X size={18} /></button></div>
			<form onsubmit={create}>
				<label class="field"><span>Title</span><input bind:value={form.title} placeholder="Event title" required /></label>
				<label class="field"><span>Date</span><input type="date" bind:value={form.date} required /></label>
				<div class="field-row">
					<label class="field"><span>Start</span><input type="time" bind:value={form.start} /></label>
					<label class="field"><span>End</span><input type="time" bind:value={form.end} /></label>
				</div>
				<label class="field"><span>Location</span><input bind:value={form.location} placeholder="Location or link (optional)" /></label>
				<div class="modal-actions">
					<button type="button" class="btn btn--ghost" onclick={() => (showCreate = false)}>Cancel</button>
					<button type="submit" class="btn btn--primary" disabled={creating || !form.title.trim() || !form.date}>{#if creating}<Loader2 class="spin" size={15} />{/if}Create event</button>
				</div>
			</form>
		</div>
	</div>
{/if}

<!-- Settings modal -->
{#if showSettings}
	<div class="overlay" role="button" tabindex="0" onclick={() => (showSettings = false)} onkeydown={(e) => e.key === 'Escape' && (showSettings = false)}>
		<div class="modal modal--wide" role="dialog" tabindex="-1" onclick={(e) => e.stopPropagation()} onkeydown={() => {}}>
			<div class="modal-head"><h2>Calendar settings</h2><button class="card-x" onclick={() => (showSettings = false)} aria-label="Close"><X size={18} /></button></div>

			<p class="set-section">Google Calendar</p>
			<div class="gcal-row">
				{#if googleConnected && googleAuthorized}
					<span class="gcal-status gcal-status--on"><Check size={14} /> Connected{googleAccount ? ` · ${googleAccount}` : ''}{syncedLabel ? ` · ${syncedLabel.toLowerCase()}` : ''}</span>
					<button class="btn btn--ghost" onclick={syncGoogle} disabled={syncing}><RefreshCw size={14} class={syncing ? 'spin' : ''} />Sync now</button>
				{:else if googleConnected}
					<span class="gcal-status gcal-status--warning">Permission required{googleAccount ? ` · ${googleAccount}` : ''}</span>
					<button class="btn btn--primary" onclick={connectGoogle} disabled={connecting}><Link2 size={14} />{connecting ? 'Connecting...' : 'Reconnect Google'}</button>
				{:else}
					<span class="gcal-status">Not connected</span>
					<button class="btn btn--primary" onclick={connectGoogle} disabled={connecting}><Link2 size={14} />{connecting ? 'Connecting…' : 'Connect Google'}</button>
				{/if}
			</div>
			<p class="gcal-hint">Events sync automatically in the background. Each person connects their own Google account.</p>

			<p class="set-section">Calendar</p>
			<div class="field-row">
				<label class="field"><span>Default view</span>
					<select bind:value={prefsForm.default_view}>
						<option value="month">Month</option><option value="week">Week</option>
						<option value="day">Day</option><option value="agenda">Agenda</option>
					</select>
				</label>
				<label class="field"><span>Default event length</span>
					<select bind:value={prefsForm.default_event_minutes}>
						<option value={15}>15 min</option><option value={30}>30 min</option>
						<option value={45}>45 min</option><option value={60}>60 min</option>
						<option value={90}>90 min</option><option value={120}>2 hours</option>
					</select>
				</label>
			</div>

			<p class="set-section">Business defaults <span class="set-hint">shared across all modules</span></p>
			<div class="field-row">
				<label class="field"><span>Time zone</span>
					<select bind:value={prefsForm.timezone}>
						<option value="America/New_York">Eastern (New York)</option>
						<option value="America/Chicago">Central (Chicago)</option>
						<option value="America/Denver">Mountain (Denver)</option>
						<option value="America/Los_Angeles">Pacific (Los Angeles)</option>
						<option value="America/Sao_Paulo">Brazil (São Paulo)</option>
						<option value="Europe/London">London</option>
						<option value="Asia/Bangkok">Bangkok</option>
					</select>
				</label>
				<label class="field"><span>Week starts on</span>
					<select bind:value={prefsForm.week_start}>
						<option value={0}>Sunday</option><option value={1}>Monday</option>
					</select>
				</label>
			</div>
			<div class="field-row">
				<label class="field"><span>Date format</span>
					<select bind:value={prefsForm.date_format}>
						<option value="MM/DD/YYYY">MM/DD/YYYY</option>
						<option value="DD/MM/YYYY">DD/MM/YYYY</option>
						<option value="YYYY-MM-DD">YYYY-MM-DD</option>
					</select>
				</label>
				<label class="field"><span>Time format</span>
					<select bind:value={prefsForm.time_format}>
						<option value="12h">12-hour (1:00 PM)</option>
						<option value="24h">24-hour (13:00)</option>
					</select>
				</label>
			</div>
			<div class="field-row">
				<label class="field"><span>Working hours start</span>
					<select bind:value={prefsForm.working_hours_start}>
						{#each Array.from({ length: 24 }, (_, i) => i) as h}<option value={h}>{h}:00</option>{/each}
					</select>
				</label>
				<label class="field"><span>Working hours end</span>
					<select bind:value={prefsForm.working_hours_end}>
						{#each Array.from({ length: 24 }, (_, i) => i) as h}<option value={h}>{h}:00</option>{/each}
					</select>
				</label>
			</div>

			<div class="modal-actions">
				<button type="button" class="btn btn--ghost" onclick={() => (showSettings = false)}>Cancel</button>
				<button type="button" class="btn btn--primary" onclick={savePrefs} disabled={savingPrefs}>{#if savingPrefs}<Loader2 class="spin" size={15} />{/if}Save settings</button>
			</div>
		</div>
	</div>
{/if}

<!-- ContentOS card editor -->
{#if editingContent}
	<ContentCalendarEditor
		bind:form={contentForm}
		contentProfiles={contentProfiles}
		workstreams={CONTENT_WORKSTREAMS}
		stages={CONTENT_STAGES}
		stageLabels={CONTENT_STAGE_LABELS}
		types={CONTENT_TYPES}
		channels={CONTENT_CHANNELS}
		saving={savingContent}
		deleting={deletingContent}
		onClose={() => (editingContent = null)}
		onSave={saveContent}
		onDelete={deleteEditingContent}
	/>
{/if}

<!-- Detail modal -->
{#if selectedEvent}
	<div class="overlay" role="button" tabindex="0" onclick={() => (selectedEvent = null)} onkeydown={(e) => e.key === 'Escape' && (selectedEvent = null)}>
		<div class="modal" role="dialog" tabindex="-1" onclick={(e) => e.stopPropagation()} onkeydown={() => {}}>
			<div class="modal-head"><h2>{selectedEvent.title ?? 'Untitled'}</h2><button class="card-x" onclick={() => (selectedEvent = null)} aria-label="Close"><X size={18} /></button></div>
			<div class="detail">
				<div class="detail-row"><Clock size={15} /><span>{fmtDayLabel(selectedEvent.start_time)} · {fmtTime(selectedEvent.start_time)} – {fmtTime(selectedEvent.end_time)}</span></div>
				{#if selectedEvent.location}<div class="detail-row"><MapPin size={15} /><span>{selectedEvent.location}</span></div>{/if}
				{#if isContentEvent(selectedEvent)}
					<div class="detail-row"><span class="src">ContentOS</span><span>This date comes from the ContentOS card.</span></div>
				{:else if selectedEvent.source === 'google'}<div class="detail-row"><span class="src">Google</span></div>{/if}
				{#if selectedEvent.description}<div class="detail-desc">{@html sanitizeHtml(selectedEvent.description)}</div>{/if}
			</div>
			<div class="modal-actions">
				{#if isContentEvent(selectedEvent)}
					<a class="btn btn--primary" href="/content">Open ContentOS</a>
				{:else}
					<button class="btn btn--danger" onclick={removeEvent} disabled={busyDelete}>{#if busyDelete}<Loader2 class="spin" size={15} />{:else}<Trash2 size={15} />{/if}Delete</button>
				{/if}
			</div>
		</div>
	</div>
{/if}

<!-- Day preview popover -->
{#if previewDay}
	<div class="overlay" role="button" tabindex="0" onclick={() => (previewDay = null)} onkeydown={(e) => e.key === 'Escape' && (previewDay = null)}>
		<div class="modal" role="dialog" tabindex="-1" onclick={(e) => e.stopPropagation()} onkeydown={() => {}}>
			<div class="modal-head">
				<h2>{previewDay.toLocaleDateString(undefined, { weekday: 'long', month: 'long', day: 'numeric' })}</h2>
				<button class="card-x" onclick={() => (previewDay = null)} aria-label="Close"><X size={18} /></button>
			</div>
			<div class="preview-list">
				{#if previewEvents.length === 0}
					<p class="preview-empty">Nothing scheduled this day.</p>
				{:else}
					{#each previewEvents as ev (ev.id)}
						<button class="preview-ev" onclick={() => openEvent(ev)}>
							<span class="preview-dot" style="background:{eventColor(ev)}"></span>
							<span class="preview-time">{fmtTime(ev.start_time)} – {fmtTime(ev.end_time)}</span>
							<span class="preview-title">{ev.title ?? 'Untitled'}</span>
						</button>
					{/each}
				{/if}
			</div>
			<div class="modal-actions">
				{#if calendarMode === 'appointments'}
					<button class="btn btn--primary" onclick={() => { const d = previewDay; previewDay = null; openCreate(d ?? undefined); }}><Plus size={15} />New event this day</button>
				{:else}
					<a class="btn btn--primary" href="/content">Open ContentOS</a>
				{/if}
			</div>
		</div>
	</div>
{/if}

<style>
	.cal-root { height: 100%; display: flex; flex-direction: column; background: var(--dbg); color: var(--dt); overflow: hidden; }
	.preview-list { display: flex; flex-direction: column; gap: 6px; margin-bottom: 16px; max-height: 50vh; overflow-y: auto; }
	.preview-empty { color: var(--dt3); font-size: 0.85rem; padding: 12px 2px; }
	.preview-ev { display: flex; align-items: center; gap: 10px; width: 100%; padding: 9px 11px; border: 1px solid var(--dbd); border-radius: 9px; background: color-mix(in srgb, var(--dt) 2%, transparent); color: var(--dt); cursor: pointer; text-align: left; }
	.preview-ev:hover { border-color: color-mix(in srgb, var(--dt) 22%, transparent); }
	.preview-dot { width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0; }
	.preview-time { font-size: 0.74rem; color: var(--dt3); width: 120px; flex-shrink: 0; font-variant-numeric: tabular-nums; }
	.preview-title { font-size: 0.86rem; font-weight: 540; flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
	.topbar { display: flex; align-items: center; justify-content: space-between; gap: 16px; padding: 14px 22px; border-bottom: 1px solid var(--dbd); flex-shrink: 0; }
	.left { display: flex; align-items: center; gap: 14px; }
	.left h1 { font-size: 1.15rem; font-weight: 680; letter-spacing: -0.02em; margin: 0; min-width: 150px; }
	.nav-btns { display: flex; align-items: center; gap: 4px; }
	.center { display: flex; align-items: center; justify-content: center; gap: 8px; min-width: 0; }
	.icon-btn { display: inline-flex; align-items: center; justify-content: center; width: 30px; height: 30px; border-radius: 8px; border: 1px solid var(--dbd); background: transparent; color: var(--dt2); cursor: pointer; }
	.icon-btn:hover { background: color-mix(in srgb, var(--dt) 8%, transparent); }
	.today-btn { padding: 6px 12px; border-radius: 8px; border: 1px solid var(--dbd); background: transparent; color: var(--dt2); font-size: 0.8rem; font-weight: 560; cursor: pointer; }
	.today-btn:hover { background: color-mix(in srgb, var(--dt) 8%, transparent); }
	.seg { display: inline-flex; padding: 3px; border-radius: 10px; background: color-mix(in srgb, var(--dt) 6%, transparent); gap: 2px; }
	.seg--mode { background: color-mix(in srgb, #0f766e 8%, transparent); }
	.seg-btn { padding: 5px 13px; border-radius: 7px; border: none; background: transparent; color: var(--dt3); font-size: 0.8rem; font-weight: 560; cursor: pointer; transition: background 0.14s, color 0.14s; }
	.seg-btn--on { background: var(--dbg); color: var(--dt); box-shadow: 0 1px 3px rgba(0,0,0,0.18); }
	.tools { display: flex; align-items: center; gap: 10px; }
	.btn { display: inline-flex; align-items: center; gap: 7px; padding: 8px 15px; border-radius: 9px; font-size: 0.83rem; font-weight: 580; cursor: pointer; border: 1px solid transparent; white-space: nowrap; }
	a.btn { text-decoration: none; }
	.btn--primary { background: var(--dt); color: var(--dbg); }
	.btn--ghost { background: transparent; border-color: var(--dbd); color: var(--dt2); }
	.btn--danger { background: color-mix(in srgb, #ef4444 14%, transparent); color: #ef4444; }
	.btn:disabled { opacity: 0.55; cursor: not-allowed; }
	.body { flex: 1; overflow-y: auto; }
	.loading { flex: 1; display: flex; align-items: center; justify-content: center; gap: 8px; color: var(--dt3); }
	.banner { margin: 12px 22px 0; padding: 11px 14px; border-radius: 10px; font-size: 0.83rem; }
	.banner--error { background: color-mix(in srgb, #ef4444 12%, transparent); color: #ef4444; }
	.banner--google { display: flex; align-items: center; justify-content: space-between; gap: 16px; border: 1px solid color-mix(in srgb, #2563eb 24%, transparent); background: color-mix(in srgb, #2563eb 7%, transparent); color: var(--dt2); }
	.banner--warning { border-color: color-mix(in srgb, #d97706 28%, transparent); background: color-mix(in srgb, #d97706 8%, transparent); }
	.banner--google div { display: flex; flex-direction: column; gap: 3px; min-width: 0; }
	.banner--google strong { color: var(--dt); font-weight: 650; }
	.banner--google span { color: var(--dt3); line-height: 1.35; }
	.banner--google .btn { flex-shrink: 0; }
	.content-cal-filters { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; padding: 12px 22px 0; }
	.content-profile-select { min-height: 30px; padding: 5px 10px; border: 1px solid var(--dbd); border-radius: 8px; background: var(--dbg); color: var(--dt2); font: inherit; font-size: 0.78rem; outline: none; }
	.content-cal-legend { display: inline-flex; align-items: center; gap: 6px; flex-wrap: wrap; min-width: 0; }
	.content-source-chip { display: inline-flex; align-items: center; gap: 6px; min-height: 28px; padding: 3px 8px 3px 5px; border: 1px solid var(--dbd); border-radius: 8px; background: color-mix(in srgb, var(--dt) 3%, transparent); color: var(--dt2); font: inherit; font-size: 0.72rem; font-weight: 620; white-space: nowrap; cursor: pointer; }
	.content-source-chip:hover { border-color: color-mix(in srgb, var(--dt) 24%, transparent); }
	.content-source-chip--off { opacity: 0.45; }
	.content-source-chip span { pointer-events: none; }
	.source-color { width: 18px; height: 18px; padding: 0; border: 0; border-radius: 999px; background: transparent; cursor: pointer; }
	.source-color::-webkit-color-swatch-wrapper { padding: 0; }
	.source-color::-webkit-color-swatch { border: 1px solid rgb(0 0 0 / 0.18); border-radius: 999px; }
	.content-cal-note { color: var(--dt3); font-size: 0.74rem; margin-left: auto; }
	.sync-status { display: inline-flex; align-items: center; gap: 5px; font-size: 0.74rem; color: var(--dt3); background: transparent; border: none; cursor: pointer; padding: 4px 6px; border-radius: 7px; }
	.sync-status:hover { color: var(--dt2); background: color-mix(in srgb, var(--dt) 5%, transparent); }
	.gcal-row { display: flex; align-items: center; justify-content: space-between; gap: 10px; margin-bottom: 6px; }
	.gcal-status { display: inline-flex; align-items: center; gap: 6px; font-size: 0.84rem; color: var(--dt2); }
	.gcal-status--on { color: #34d399; }
	.gcal-status--warning { color: #d97706; }
	.gcal-hint { font-size: 0.74rem; color: var(--dt4); margin: 0 0 16px; }
	.members { display: flex; align-items: center; gap: 7px; flex-wrap: wrap; padding: 12px 22px 0; }
	.members-label { font-size: 0.74rem; color: var(--dt3); font-weight: 560; }
	.mchip { display: inline-flex; align-items: center; gap: 6px; padding: 4px 10px; border-radius: 999px; border: 1px solid var(--dbd); background: color-mix(in srgb, var(--dt) 4%, transparent); color: var(--dt2); font-size: 0.76rem; font-weight: 540; cursor: pointer; transition: opacity 0.14s; }
	.mchip--off { opacity: 0.4; }
	.mchip:hover { border-color: color-mix(in srgb, var(--dt) 25%, transparent); }
	.mdot { width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0; }
	.legend { display: inline-flex; align-items: center; gap: 9px; flex-wrap: wrap; margin-left: 6px; padding-left: 10px; border-left: 1px solid var(--dbd); font-size: 0.72rem; color: var(--dt3); font-weight: 540; }
	.legend-item { display: inline-flex; align-items: center; gap: 5px; }
	.agenda { padding: 18px 24px; max-width: 720px; margin: 0 auto; }
	.empty { display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 12px; color: var(--dt3); padding: 60px 0; }
	.ag-day { margin-bottom: 26px; }
	.ag-day--today { position: relative; }
	.ag-head { display: flex; align-items: baseline; gap: 9px; margin-bottom: 11px; padding-bottom: 9px; border-bottom: 1px solid var(--dbd); }
	.ag-head-label { font-size: 0.84rem; font-weight: 660; color: var(--dt); letter-spacing: -0.01em; }
	.ag-head-date { font-size: 0.74rem; color: var(--dt3); font-variant-numeric: tabular-nums; }
	.ag-day--today .ag-head-label { color: var(--bos-nav-active, #6aa3ff); }
	.ag-head-today { font-size: 0.62rem; font-weight: 700; letter-spacing: 0.05em; text-transform: uppercase; color: var(--dbg); background: var(--bos-nav-active, #6aa3ff); padding: 2px 7px; border-radius: 999px; }
	.ag-head-count { margin-left: auto; font-size: 0.7rem; color: var(--dt4); font-weight: 540; }
	.ag-ev { display: flex; align-items: center; gap: 12px; width: 100%; padding: 10px 13px; margin-bottom: 6px; border: 1px solid var(--dbd); border-radius: 10px; background: color-mix(in srgb, var(--dt) 2%, transparent); color: var(--dt); cursor: pointer; text-align: left; transition: border-color 0.14s, background 0.14s; }
	.ag-ev:hover { border-color: color-mix(in srgb, var(--dt) 22%, transparent); background: color-mix(in srgb, var(--dt) 5%, transparent); }
	.ag-dot { width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0; }
	.ag-time { font-size: 0.76rem; color: var(--dt3); width: 150px; flex-shrink: 0; font-variant-numeric: tabular-nums; white-space: nowrap; }
	.ag-title { font-size: 0.88rem; font-weight: 540; flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
	.ag-loc { display: inline-flex; align-items: center; gap: 4px; font-size: 0.73rem; color: var(--dt3); flex-shrink: 0; max-width: 180px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
	.ag-src { font-size: 0.66rem; color: var(--dt3); background: color-mix(in srgb, var(--dt) 6%, transparent); padding: 1px 6px; border-radius: 5px; flex-shrink: 0; }
	.overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.55); display: flex; align-items: center; justify-content: center; z-index: 100; padding: 20px; }
	.modal { width: 100%; max-width: 440px; background: var(--dbg); border: 1px solid var(--dbd); border-radius: 16px; padding: 22px; box-shadow: 0 24px 60px rgba(0,0,0,0.5); }
	.modal-head { display: flex; align-items: center; justify-content: space-between; margin-bottom: 18px; gap: 12px; }
	.modal-head h2 { font-size: 1.05rem; font-weight: 640; margin: 0; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
	.modal-head p { margin: 4px 0 0; color: var(--dt3); font-size: 0.8rem; line-height: 1.35; }
	.card-x { display: inline-flex; align-items: center; justify-content: center; width: 28px; height: 28px; border-radius: 7px; border: none; background: transparent; color: var(--dt3); cursor: pointer; flex-shrink: 0; }
	.card-x:hover { background: color-mix(in srgb, var(--dt) 8%, transparent); }
	.field { display: flex; flex-direction: column; gap: 6px; margin-bottom: 14px; }
	.field span { font-size: 0.8rem; font-weight: 560; color: var(--dt2); }
	.field input { padding: 9px 12px; border-radius: 9px; border: 1px solid var(--dbd); background: var(--dbg); color: var(--dt); font-size: 0.86rem; outline: none; font-family: inherit; }
	.field input:focus { border-color: color-mix(in srgb, var(--dt) 40%, transparent); }
	.field-row { display: flex; gap: 12px; }
	.field-row .field { flex: 1; }
	.field select { padding: 9px 12px; border-radius: 9px; border: 1px solid var(--dbd); background: var(--dbg); color: var(--dt); font-size: 0.86rem; outline: none; font-family: inherit; cursor: pointer; }
	.field select:focus { border-color: color-mix(in srgb, var(--dt) 40%, transparent); }
	.modal--wide { max-width: 560px; }
	.set-section { font-size: 0.74rem; font-weight: 700; letter-spacing: 0.04em; text-transform: uppercase; color: var(--dt3); margin: 6px 0 10px; }
	.set-hint { font-weight: 500; text-transform: none; letter-spacing: 0; color: var(--dt4); margin-left: 6px; }
	.modal-actions { display: flex; justify-content: flex-end; gap: 10px; margin-top: 6px; }
	.detail { display: flex; flex-direction: column; gap: 10px; margin-bottom: 18px; }
	.detail-row { display: flex; align-items: center; gap: 9px; font-size: 0.86rem; color: var(--dt2); }
	.detail-desc { font-size: 0.85rem; color: var(--dt2); margin: 4px 0 0; line-height: 1.55; max-height: 320px; overflow-y: auto; padding-right: 4px; }
	.detail-desc :global(p) { margin: 0 0 8px; }
	.detail-desc :global(ul), .detail-desc :global(ol) { margin: 4px 0 10px; padding-left: 18px; }
	.detail-desc :global(li) { margin: 2px 0; }
	.detail-desc :global(h1), .detail-desc :global(h2), .detail-desc :global(h3) { font-size: 0.9rem; font-weight: 650; color: var(--dt); margin: 10px 0 4px; }
	.detail-desc :global(strong) { color: var(--dt); font-weight: 600; }
	.detail-desc :global(a) { color: var(--bos-accent, #6aa3ff); text-decoration: underline; }
	.src { font-size: 0.7rem; color: var(--dt3); background: color-mix(in srgb, var(--dt) 6%, transparent); padding: 2px 8px; border-radius: 6px; }
	:global(.spin) { animation: spin 0.9s linear infinite; }
	@keyframes spin { to { transform: rotate(360deg); } }
	@media (max-width: 760px) { .center { display: none; } .left h1 { min-width: 0; font-size: 1rem; } }
</style>
