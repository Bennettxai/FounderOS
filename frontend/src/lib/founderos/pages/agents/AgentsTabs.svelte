<!-- /agents tab shell (FounderOS v1 components/AgentsTabs.tsx): Roster (the
     live board + Conductor rail, always mounted), Needs You, Deliverables, and
     the stock Hermes worker-pool dashboard embedded from the host (the iframe
     mounts on first activation only). Deliverables are fetched here so the
     Needs You badge counts while its list is closed. The active tab mirrors
     into ?tab= so a link can open it. -->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import { onMount } from 'svelte';
	import { ArrowUpRight } from '$lib/founderos/icons';
	import { replaceState } from '$app/navigation';
	import { founderosFetch } from '$lib/founderos/api';
	import DeliverablesPanel from './DeliverablesPanel.svelte';
	import NeedsYouList from './NeedsYouList.svelte';
	import { AGENTS_TABS, tabHref, type AgentsTabId } from './agents-tabs';
	import { SEEN_KEY, markSeenMap, parseSeenMap, revisionMap, tabBadges, type SeenMap } from './deliverables-seen';
	import {
		OPENED_KEY,
		markManyOpened,
		markOpened,
		parseOpenedMap,
		pruneOpened,
		readStore,
		unseenStateOf,
		writeStore,
		type OpenedMap
	} from '../console/needs-queue';
	import type { DeliverableItem, DeliverablesBody, PaperclipIssue } from './types';

	let {
		hermesUrl = '',
		boardUrl = null,
		initialTab = 'roster',
		issues = [],
		children
	}: {
		/** '' while the board view is still loading: the Hermes tab waits. */
		hermesUrl?: string;
		boardUrl?: string | null;
		initialTab?: AgentsTabId;
		/** The live board's issues; the finished ones join Deliverables. */
		issues?: PaperclipIssue[];
		children?: Snippet;
	} = $props();

	// svelte-ignore state_referenced_locally
	let tab = $state<AgentsTabId>(initialTab);
	// svelte-ignore state_referenced_locally
	let visited = $state(initialTab === 'hermes');
	let data = $state<DeliverablesBody | null>(null);
	let error = $state<string | null>(null);
	let refreshing = $state(false);
	/** Only a successful response may move the seen store. */
	let loadedOk = $state(false);
	let hydrated = $state(false);
	/** The tab badges' store: what he has looked at, per revision. */
	let seen = $state<SeenMap | null>(null);
	/** The per-row dots' store: what he has OPENED, per revision. */
	let opened = $state<OpenedMap | null>(null);

	async function reload() {
		refreshing = true;
		try {
			data = await founderosFetch<DeliverablesBody>('/pages/board/deliverables');
			error = null;
			loadedOk = true;
			adopt(data);
		} catch (err) {
			error = err instanceof Error ? err.message : String(err);
		} finally {
			refreshing = false;
		}
	}

	/** First successful load in a fresh browser sets both baselines silently:
	    only what lands AFTER today ever badges or dots. After that, prune the
	    opened store so it cannot grow forever as agents churn files. */
	function adopt(body: DeliverablesBody) {
		if (!hydrated) return;
		const all = (body.groups ?? []).flatMap((g) => g.items ?? []);
		if (seen === null) {
			seen = markSeenMap(null, revisionMap(body.groups ?? []));
			writeStore(SEEN_KEY, seen);
		}
		if (opened === null) {
			opened = markManyOpened({}, all.map((i) => ({ id: i.id, revision: i.revision })));
			writeStore(OPENED_KEY, opened);
			return;
		}
		const pruned = pruneOpened(opened, all.map((i) => i.id));
		if (Object.keys(pruned).length !== Object.keys(opened).length) {
			opened = pruned;
			writeStore(OPENED_KEY, pruned);
		}
	}

	onMount(() => {
		seen = parseSeenMap(readStore(SEEN_KEY));
		opened = parseOpenedMap(readStore(OPENED_KEY));
		hydrated = true;
		void reload();
	});

	const current = $derived(data ? revisionMap(data.groups ?? []) : {});
	const badges = $derived(tabBadges(hydrated, current, seen));

	// Looking at the tab is what clears its badges, whether he switched to it or
	// was already sitting on it when something new landed.
	$effect(() => {
		if (tab !== 'deliverables' || !loadedOk || badges.unseen + badges.changed === 0) return;
		seen = markSeenMap(seen, current);
		writeStore(SEEN_KEY, seen);
	});

	const unseenOf = (item: DeliverableItem) => unseenStateOf(opened, item.id, item.revision);
	function markItemOpened(item: DeliverableItem) {
		opened = markOpened(opened ?? {}, item.id, item.revision);
		writeStore(OPENED_KEY, opened);
	}

	// The waiting count is NOT an unread badge: it clears only when the agents
	// stop asking or he makes the call himself.
	const waiting = $derived(data?.needsYou.open.length ?? 0);
	const TAB_WIDTH = 150;
	const ids = Object.keys(AGENTS_TABS) as AgentsTabId[];
	const index = $derived(Math.max(0, ids.indexOf(tab)));

	function switchTo(id: AgentsTabId) {
		tab = id;
		if (id === 'hermes') visited = true;
		if (id === 'needsyou' || id === 'deliverables') reload();
		const href = tabHref(window.location.href, id);
		try {
			replaceState(href, {});
		} catch {
			// the router is not ready (tests): keep the link tracking the tab anyway
			try {
				window.history.replaceState(window.history.state, '', href);
			} catch {
				/* the tab still switches; only the link stops tracking it */
			}
		}
	}
</script>

<div class="flex min-h-0 flex-1 flex-col">
	<!-- One sliding underline across fixed 150px tabs (FounderOS v1 SlidingTabs):
	     the marker travels between them, so nothing jumps on a switch. -->
	<div class="mb-4 flex shrink-0 items-center gap-1">
		<div class="bn-tabs relative grid border-b" style="grid-template-columns: repeat({ids.length}, {TAB_WIDTH}px)" role="tablist" aria-label="Agents views">
			<span aria-hidden="true" class="bn-tab-marker pointer-events-none absolute -bottom-px left-0 h-[2px]" style="width: {TAB_WIDTH}px; transform: translateX({index * TAB_WIDTH}px)"></span>
			{#each ids as id (id)}
				<button
					type="button"
					role="tab"
					aria-selected={tab === id}
					data-lens="c"
					onclick={() => switchTo(id)}
					class="bn-pressable bn-tab relative inline-flex h-[30px] items-center justify-center gap-1.5 bg-transparent font-mono text-[10.5px] font-bold uppercase tracking-[.18em]"
					class:is-active={tab === id}
				>
					{AGENTS_TABS[id]}
					{#if id === 'needsyou' && waiting > 0}
						<span data-badge="warn" class="bn-tab-badge" title="{waiting} agent files are waiting on you">{waiting}</span>
					{/if}
					{#if id === 'deliverables'}
						{#if badges.changed > 0}
							<span data-badge="err" class="bn-tab-badge" title="{badges.changed} changed since you last looked">{badges.changed}</span>
						{/if}
						{#if badges.unseen > 0}
							<span data-badge="ok" class="bn-tab-badge" title="{badges.unseen} new since you last looked">{badges.unseen}</span>
						{/if}
					{/if}
				</button>
			{/each}
		</div>
		{#if tab === 'hermes' && hermesUrl}
			<a
				href={hermesUrl}
				target="_blank"
				rel="noreferrer"
				data-lens="c"
				class="bn-pressable is-dark bn-dim bn-hermes-link ml-auto flex items-center gap-1 px-2 py-1 font-mono text-[10px]"
			>
				Open full dashboard <ArrowUpRight class="h-3 w-3" />
			</a>
		{/if}
	</div>

	<!-- The roster stays mounted always -->
	<div class={tab === 'roster' ? 'min-h-0 flex-1' : 'hidden'}>{@render children?.()}</div>

	<!-- The queue of work asking for his decision, from the same payload -->
	{#if tab === 'needsyou'}
		<NeedsYouList {data} {error} {boardUrl} {refreshing} {opened} onOpen={markItemOpened} onReload={reload} />
	{/if}

	<!-- Agent deliverables: files the board agents actually produced -->
	{#if tab === 'deliverables'}
		<DeliverablesPanel {data} {error} {issues} {refreshing} {unseenOf} onOpen={markItemOpened} onReload={reload} />
	{/if}

	{#if visited && !hermesUrl && tab === 'hermes'}
		<p class="bn-dim font-mono text-[10px]">reading the board…</p>
	{/if}
	{#if visited && hermesUrl}
		<div class={tab === 'hermes' ? '' : 'hidden'}>
			<iframe src={hermesUrl} title="Hermes worker-pool dashboard" class="bn-hermes h-[calc(100dvh-14rem)] min-h-[480px] w-full border"></iframe>
			<div class="bn-dim mt-1.5 font-mono text-[9.5px]">
				Stock Hermes dashboard, embedded live from the host. Blank or erroring? The dashboard process may be down on os-host (the VPN serve answers 502).
			</div>
		</div>
	{/if}
</div>

<style>
	.bn-tabs {
		border-color: var(--bn-border);
	}
	.bn-tab-marker {
		background: var(--bn-text);
		transition: transform var(--bn-dur-lens) var(--bn-ease-lens);
	}
	/* prod SlidingTabs: hover:scale-[1.04] hover:shadow-none over the lens */
	.bn-tab.bn-pressable:hover {
		transform: scale(1.04);
		box-shadow: none;
	}
	.bn-hermes-link {
		border-radius: var(--bn-r-ctl, 6px);
	}
	.bn-hermes-link:hover {
		color: var(--bn-text);
	}
	.bn-tab {
		color: var(--bn-text-3);
	}
	.bn-tab:hover {
		color: var(--bn-text-2);
	}
	.bn-tab.is-active {
		color: var(--bn-text);
	}
	/* A round count riding inside the tab label: a status dot that grew. */
	.bn-tab-badge {
		display: inline-flex;
		min-width: 1rem;
		height: 1rem;
		align-items: center;
		justify-content: center;
		padding: 0 4px;
		border-radius: 9999px;
		background: var(--bn-warn);
		color: var(--bn-bg);
		font-size: 9px;
		font-weight: 700;
		line-height: 1;
		letter-spacing: normal;
	}
	.bn-tab-badge[data-badge='err'] {
		background: var(--bn-err);
	}
	.bn-tab-badge[data-badge='ok'] {
		background: var(--bn-ok);
	}
	.bn-hermes {
		border-color: var(--bn-border);
		border-radius: var(--bn-r-panel, 10px);
		background: var(--bn-bg);
	}
	@media (prefers-reduced-motion: reduce) {
		.bn-tab-marker {
			transition: none;
		}
	}
</style>
