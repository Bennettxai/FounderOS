<!-- The front of /comms (FounderOS v1 CommsTabs.tsx): a swappable view between
     the three-pane messaging view, the 7-day meetings calendar and the
     recordings list (Plaud in the room + Fathom on calls). -->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { CommsPage } from './model';
	import RecordingsBoard from './RecordingsBoard.svelte';
	import ThreePane from './ThreePane.svelte';
	import SlidingTabs from './SlidingTabs.svelte';
	import WeekCalendar from './WeekCalendar.svelte';

	/** lead: an optional heading on the tab row's left, so the page needs no separate title bar. */
	let { page, nowMs, lead }: { page: CommsPage; nowMs: number; lead?: Snippet } = $props();

	type Tab = 'messaging' | 'meetings' | 'recordings';
	let tab = $state<Tab>('messaging');
	const unread = $derived(page.lanes.reduce((n, l) => n + l.unread, 0));
	const tabs = $derived<Array<{ id: Tab; label: string; count: number | string | null }>>([
		{ id: 'messaging', label: 'Messaging', count: unread },
		{ id: 'meetings', label: 'Meetings', count: page.calendar.error ? '?' : page.calendar.events.length },
		{ id: 'recordings', label: 'Recordings', count: page.recordings.recordings.length }
	]);
</script>

<div>
	<div data-part="tab-row" class="pc-hair mb-5 flex flex-wrap items-center gap-2 border-b pb-3">
		{@render lead?.()}
		<SlidingTabs {tabs} value={tab} onchange={(id) => (tab = id)} tabWidth={120} />
		<span class="bn-dim ml-auto font-mono text-[10px] uppercase tracking-[0.15em]">
			{tab === 'messaging' ? `${unread} unread` : tab === 'meetings' ? 'next 7 days' : 'plaud + fathom · newest first'}
		</span>
	</div>

	{#if tab === 'messaging'}
		<ThreePane lanes={page.lanes} slackCards={page.slackCards} slackRoster={page.slackRoster} channels={page.channels} channelsError={page.channelsError} {nowMs} />
	{:else if tab === 'meetings'}
		<WeekCalendar events={page.calendar.events} accounts={page.calendar.accounts} nowISO={page.now} error={page.calendar.error} />
	{:else}
		<RecordingsBoard recordings={page.recordings.recordings} sources={page.recordings.sources} errors={page.recordings.errors} {nowMs} />
	{/if}
</div>
