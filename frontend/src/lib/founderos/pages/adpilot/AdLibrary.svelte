<!-- The Ad library (FounderOS v1 components/adpilot/AdLibrary.tsx): Ask Adscout
     on top, then the band (open library · API credits · refresh), then one
     bounded module with tabs: For you, Search, Saved, Watchlist, Activity.
     Reads the synced store only; refresh, search and track-by-domain are
     the explicit, user-triggered Foreplay spends. -->
<script lang="ts">
	import { untrack } from 'svelte';
	import { ChevronDown, Library as LibraryIcon, LoaderCircle, RefreshCw } from '$lib/founderos/icons';
	import { founderosFetch } from '$lib/founderos/api';
	import ActivityTab from './ActivityTab.svelte';
	import AskWidget from './AskWidget.svelte';
	import CardGrid from './CardGrid.svelte';
	import Dossier from './Dossier.svelte';
	import Empty from './Empty.svelte';
	import SearchTab from './SearchTab.svelte';
	import WatchlistTab from './WatchlistTab.svelte';
	import { ago, creditTicks, forYou, num } from './logic';
	import type { Library, SavedAd, WallAd, WatchEntry } from './types';

	let { library, onRefreshed }: { library: Library; onRefreshed?: () => void } = $props();

	type Tab = 'foryou' | 'search' | 'saved' | 'watchlist' | 'activity';

	let tab = $state<Tab>('foryou');
	// Local copies the tabs mutate, seeded from the payload once (the view
	// remounts the library after a refresh).
	let watchlist = $state<WatchEntry[]>(untrack(() => library.watchlist));
	let saved = $state<SavedAd[]>(untrack(() => library.saved));
	let syncing = $state(false);
	let notice = $state<string | null>(null);
	let selected = $state<WallAd | null>(null);
	let mineResults = $state<WallAd[] | null>(null);
	let libOpen = $state(false);

	const savedIds = $derived(new Set(saved.map((s) => s.ad.id)));
	const savedAds = $derived(saved.map((s) => s.ad));
	const feed = $derived.by(() => {
		const pool = new Map<string, WallAd>();
		for (const ad of library.wall) pool.set(ad.id, ad);
		for (const ad of mineResults ?? []) pool.set(ad.id, ad);
		return forYou([...pool.values()], savedAds).slice(0, 30);
	});
	const credits = $derived(library.credits);
	const lit = $derived(creditTicks(credits));

	const tabs = $derived<{ key: Tab; label: string; count?: number }[]>([
		{ key: 'foryou', label: 'For you' },
		{ key: 'search', label: 'Search' },
		{ key: 'saved', label: 'Saved', count: saved.length },
		{ key: 'watchlist', label: 'Watchlist', count: watchlist.length },
		{ key: 'activity', label: 'Activity', count: library.signals.length }
	]);

	async function toggleSave(ad: WallAd) {
		const isSaved = savedIds.has(ad.id);
		try {
			const body = await founderosFetch<{ saved: SavedAd[] }>('/pages/adscout/saved', {
				method: isSaved ? 'DELETE' : 'POST',
				json: isSaved ? { adId: ad.id } : ad
			});
			saved = body.saved;
		} catch (err) {
			notice = err instanceof Error ? err.message : 'save failed';
		}
	}

	async function runSync() {
		syncing = true;
		notice = null;
		try {
			const body = await founderosFetch<{ brands: number; signals: unknown[] }>('/pages/adscout/sync', { method: 'POST' });
			notice = `Refreshed: ${body.brands} brands · ${body.signals.length} new signals`;
			onRefreshed?.();
		} catch (err) {
			notice = err instanceof Error ? err.message : 'refresh failed';
		} finally {
			syncing = false;
		}
	}

	async function watch(ad: WallAd) {
		if (!ad.brandId) return;
		try {
			const body = await founderosFetch<{ watchlist: WatchEntry[] }>('/pages/adscout/watchlist', {
				method: 'POST',
				json: { brandId: ad.brandId, name: ad.brand, avatar: ad.thumbnail }
			});
			watchlist = body.watchlist;
			notice = `Watching ${ad.brand}`;
		} catch (err) {
			notice = err instanceof Error ? err.message : 'track failed';
		}
	}
</script>

<div data-part="library">
	<AskWidget wall={library.wall} onOpen={(ad) => (selected = ad)} />

	<div class="mt-6 grid gap-4 lg:grid-cols-12">
		<button
			type="button"
			onclick={() => (libOpen = !libOpen)}
			aria-expanded={libOpen}
			class="bn-pressable ap-bright flex items-center justify-between p-5 text-left hover:-translate-y-px lg:col-span-5"
		>
			<span>
				<span class="flex items-center gap-2 text-[16px] font-medium text-white">
					<LibraryIcon class="h-5 w-5" strokeWidth={1.7} />
					{libOpen ? 'Close ad library' : 'Open ad library'}
				</span>
				<span class="mt-1 block text-[12.5px] text-white/75">
					{num(library.wall.length)} tracked ads · {saved.length} saved · search by concept, save winners, remake
				</span>
			</span>
			<ChevronDown class="h-5 w-5 shrink-0 text-white/85 transition-transform {libOpen ? 'rotate-180' : ''}" strokeWidth={1.9} />
		</button>
		<div class="ap-panel p-5 lg:col-span-4" data-part="credits">
			<div class="bn-dim text-[12.5px]">API credits</div>
			<div class="bn-text mt-1 text-[25px] font-light tabular-nums">
				{#if credits}
					{num(credits.remaining)} <span class="bn-dim text-[14px]">/ {num(credits.total)}</span>
				{:else if library.configured}
					<span class="bn-dim" data-unknown>unknown until first sync</span>
				{:else}
					not connected
				{/if}
			</div>
			<div class="mt-2.5 flex gap-[3px]">
				{#each { length: 20 } as _, i (i)}
					<span class="h-[10px] flex-1 rounded-[2px]" style="background: {i < lit ? 'rgba(255,69,87,0.85)' : 'rgba(255,255,255,0.07)'}"></span>
				{/each}
			</div>
		</div>
		<div class="ap-panel flex flex-col justify-center p-5 lg:col-span-3">
			<button type="button" onclick={runSync} disabled={syncing} class="bn-pressable ap-btn ap-btn-primary flex items-center justify-center gap-2 py-2.5 text-[13px]">
				{#if syncing}<LoaderCircle class="ap-spin h-4 w-4" strokeWidth={1.7} />{:else}<RefreshCw class="h-4 w-4" strokeWidth={1.7} />{/if}
				Refresh data
			</button>
			<div class="bn-dim mt-2 text-center text-[12px]" data-part="sync-note">{notice ?? `last synced ${ago(library.lastSyncAt)}`}</div>
		</div>
	</div>

	{#if libOpen}
		<div class="ap-panel mt-4 flex h-[80vh] flex-col overflow-hidden" data-part="library-module">
			<div class="flex gap-1 border-b border-[rgba(255,69,87,0.14)] px-3" role="tablist">
				{#each tabs as t (t.key)}
					<button
						type="button"
						role="tab"
						aria-selected={tab === t.key}
						onclick={() => (tab = t.key)}
						class="bn-pressable flex items-center gap-1.5 border-b-2 px-2.5 py-2.5 text-[12.5px] {tab === t.key ? 'bn-text border-[var(--bn-accent)]' : 'bn-dim ap-hover-muted border-transparent'}"
					>
						{t.label}
						{#if t.count !== undefined && t.count > 0}<span class="bn-dim text-[10.5px] tabular-nums">{t.count}</span>{/if}
					</button>
				{/each}
			</div>
			<div class="min-h-0 flex-1 overflow-y-auto p-4">
				{#if tab === 'foryou'}
					<CardGrid ads={feed.map((f) => f.ad)} {savedIds} onOpen={(ad) => (selected = ad)} onSave={toggleSave} />
					<p class="bn-dim mt-3 text-center text-[11px]">ranked by fit to your niche + your saves + proven longevity: saves teach this feed</p>
				{:else if tab === 'search'}
					<SearchTab {savedIds} onOpen={(ad) => (selected = ad)} onSave={toggleSave} onResults={(ads) => (mineResults = ads)} />
				{:else if tab === 'saved'}
					{#if saved.length === 0}
						<Empty line1="Nothing saved yet." line2="The bookmark on any ad saves it here: and teaches the For-you feed." />
					{:else}
						<CardGrid ads={savedAds} {savedIds} onOpen={(ad) => (selected = ad)} onSave={toggleSave} />
					{/if}
				{:else if tab === 'watchlist'}
					<WatchlistTab {watchlist} wall={library.wall} setWatchlist={(w) => (watchlist = w)} setNotice={(n) => (notice = n)} />
				{:else}
					<ActivityTab signals={library.signals} wall={library.wall} onOpen={(ad) => (selected = ad)} />
				{/if}
			</div>
		</div>
	{/if}

	{#if selected}
		<Dossier ad={selected} isSaved={savedIds.has(selected.id)} onClose={() => (selected = null)} onSave={toggleSave} onWatch={watch} />
	{/if}
</div>
