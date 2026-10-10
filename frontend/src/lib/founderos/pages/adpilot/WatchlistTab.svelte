<!-- The local Adscout watchlist: track a brand by website (one Foreplay
     lookup) and stop watching it. -->
<script lang="ts">
	import { LoaderCircle, Plus, X } from '$lib/founderos/icons';
	import { founderosFetch } from '$lib/founderos/api';
	import Empty from './Empty.svelte';
	import { brandCounts } from './logic';
	import type { WallAd, WatchEntry } from './types';

	let {
		watchlist,
		wall,
		setWatchlist,
		setNotice
	}: { watchlist: WatchEntry[]; wall: WallAd[]; setWatchlist: (w: WatchEntry[]) => void; setNotice: (n: string | null) => void } = $props();

	let domain = $state('');
	let busy = $state(false);
	const counts = $derived(brandCounts(wall));

	async function add() {
		const d = domain.trim();
		if (!d) return;
		busy = true;
		try {
			const body = await founderosFetch<{ added: WatchEntry; watchlist: WatchEntry[] }>('/pages/adscout/watchlist', { method: 'POST', json: { domain: d } });
			setWatchlist(body.watchlist);
			domain = '';
			setNotice(`Watching ${body.added.name}: refresh data to pull its ads`);
		} catch (err) {
			setNotice(err instanceof Error ? err.message : 'add failed');
		} finally {
			busy = false;
		}
	}

	async function remove(brandId: string) {
		try {
			const body = await founderosFetch<{ watchlist: WatchEntry[] }>('/pages/adscout/watchlist', { method: 'DELETE', json: { brandId } });
			setWatchlist(body.watchlist);
		} catch (err) {
			setNotice(err instanceof Error ? err.message : 'remove failed');
		}
	}
</script>

<div class="mx-auto max-w-2xl" data-part="watchlist">
	<label class="bn-muted block text-[12px]" for="ap-domain">Track a brand: enter its website, Adscout finds its ad pages</label>
	<div class="mt-1.5 flex items-center gap-2">
		<input
			id="ap-domain"
			bind:value={domain}
			onkeydown={(e) => e.key === 'Enter' && add()}
			placeholder="skool.com"
			class="bn-text ap-ph min-w-0 flex-1 rounded-[10px] border border-[rgba(255,69,87,0.18)] bg-black/35 px-3 py-2.5 text-[13px] outline-none"
		/>
		<button type="button" onclick={add} disabled={busy} class="bn-pressable ap-btn flex items-center gap-1.5 text-[12.5px]">
			{#if busy}<LoaderCircle class="ap-spin h-4 w-4" strokeWidth={1.7} />{:else}<Plus class="h-4 w-4" strokeWidth={1.7} />{/if}
			Track brand
		</button>
	</div>
	<div class="mt-4 flex flex-col gap-2">
		{#if watchlist.length === 0}
			<Empty line1="No brands tracked." line2="Tracked brands get synced on each refresh: new launches, longevity winners, kills, velocity moves." />
		{/if}
		{#each watchlist as w (w.id)}
			{@const c = counts.get(w.id)}
			<div class="flex items-center gap-3 rounded-[12px] border border-[rgba(255,69,87,0.13)] bg-black/25 px-3.5 py-2.5">
				{#if w.avatar}
					<img src={w.avatar} alt="" class="h-8 w-8 rounded-full object-cover" />
				{:else}
					<span class="flex h-8 w-8 items-center justify-center rounded-full bg-[var(--bn-accent-soft)] text-[12px] text-[var(--bn-accent)]">{w.name.slice(0, 1)}</span>
				{/if}
				<div class="min-w-0 flex-1">
					<div class="bn-text truncate text-[13px]">{w.name}</div>
					<div class="bn-dim text-[11px]">{w.domain ?? 'added from a result'}</div>
				</div>
				{#if c}<div class="bn-muted text-right text-[11.5px] tabular-nums">{c.live} live · longest {c.top}d</div>{/if}
				<button type="button" aria-label="stop watching {w.name}" onclick={() => remove(w.id)} class="bn-pressable ap-btn p-1.5"><X class="h-3.5 w-3.5" strokeWidth={1.7} /></button>
			</div>
		{/each}
	</div>
</div>
