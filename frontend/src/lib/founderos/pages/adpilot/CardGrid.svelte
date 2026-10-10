<!-- Ad cards: the first ad with a still takes the 2x2 lead slot; every card
     carries the save toggle (the swipe-file affordance). -->
<script lang="ts">
	import { Bookmark } from '$lib/founderos/icons';
	import Empty from './Empty.svelte';
	import { leadWithImage } from './logic';
	import type { WallAd } from './types';

	let {
		ads,
		savedIds,
		onOpen,
		onSave
	}: { ads: WallAd[]; savedIds: Set<string>; onOpen: (ad: WallAd) => void; onSave: (ad: WallAd) => void } = $props();

	const display = $derived(leadWithImage(ads));
</script>

{#if display.length === 0}
	<Empty line1="Nothing here yet." line2="Add brands to the watchlist or run a search: the library builds itself." />
{:else}
	<div class="grid grid-cols-2 gap-2 sm:grid-cols-3 xl:grid-cols-4" data-part="card-grid">
		{#each display as ad, i (ad.id)}
			{@const saved = savedIds.has(ad.id)}
			<div class="group relative {i === 0 ? 'col-span-2 row-span-2' : ''}" data-part="ad-card">
				<button
					type="button"
					onclick={() => onOpen(ad)}
					class="bn-pressable is-row block w-full overflow-hidden rounded-[12px] border border-[rgba(255,69,87,0.13)] bg-[#0a0608] text-left"
				>
					<div class="relative w-full {i === 0 ? 'aspect-[4/3.05]' : 'aspect-[4/3]'} bg-[#0a0d0f]">
						{#if ad.thumbnail || ad.image}
							<img src={ad.thumbnail ?? ad.image ?? ''} alt="{ad.brand} ad creative" class="h-full w-full object-cover opacity-90 transition-opacity group-hover:opacity-100" loading="lazy" />
						{:else}
							<div class="bn-dim flex h-full items-center justify-center font-mono text-[11px]">{ad.format ?? 'ad'}</div>
						{/if}
						<div class="absolute inset-x-0 bottom-0 h-16 bg-gradient-to-t from-[#04070ae8] to-transparent"></div>
						<div class="absolute bottom-2 left-2.5 right-2.5 flex items-end justify-between gap-2">
							<span class="bn-muted truncate text-[11.5px]">{ad.brand}</span>
							<span class="flex shrink-0 items-baseline gap-1">
								<span class="text-[19px] font-light tabular-nums" style="color: {ad.live ? 'var(--bn-accent)' : 'var(--bn-text-3)'}">{ad.daysRunning}</span>
								<span class="bn-dim text-[10px]">days</span>
							</span>
						</div>
						{#if ad.live}<span class="absolute left-2 top-2 h-2 w-2 rounded-full" style="background: var(--bn-accent)"></span>{/if}
					</div>
					{#if ad.hook}
						<p class="bn-muted px-2.5 py-2 text-[12px] leading-snug {i === 0 ? 'line-clamp-3' : 'line-clamp-2'}">“{ad.hook}”</p>
					{/if}
				</button>
				<button
					type="button"
					aria-label={saved ? 'unsave ad' : 'save ad'}
					onclick={(e) => {
						e.stopPropagation();
						onSave(ad);
					}}
					class="bn-pressable absolute right-2 top-2 rounded-full border p-1.5 backdrop-blur {saved
						? 'border-[var(--bn-accent-line)] bg-[var(--bn-accent-soft)] text-[var(--bn-accent)]'
						: 'bn-muted ap-hover-text border-[rgba(255,255,255,0.14)] bg-black/45 opacity-0 group-hover:opacity-100 focus:opacity-100'}"
				>
					<Bookmark class="h-3.5 w-3.5" strokeWidth={1.7} fill={saved ? 'currentColor' : 'none'} />
				</button>
			</div>
		{/each}
	</div>
{/if}
