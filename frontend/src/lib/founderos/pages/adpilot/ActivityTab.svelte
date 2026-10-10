<!-- The wire: what changed across the watchlist, grouped by day. -->
<script lang="ts">
	import { CircleX, TrendingDown, TrendingUp, Trophy, Zap } from '$lib/founderos/icons';
	import Empty from './Empty.svelte';
	import { groupSignals } from './logic';
	import type { Signal, SignalType, WallAd } from './types';

	let { signals, wall, onOpen }: { signals: Signal[]; wall: WallAd[]; onOpen: (ad: WallAd) => void } = $props();

	const ICON: Record<SignalType, typeof Zap> = {
		new_launch: Zap,
		winner: Trophy,
		killed: CircleX,
		velocity_spike: TrendingUp,
		velocity_drop: TrendingDown
	};
	const byAd = $derived(new Map(wall.map((a) => [a.id, a])));
	const groups = $derived(groupSignals(signals));
</script>

{#if signals.length === 0}
	<Empty line1="Quiet." line2="After each refresh this lists what changed across your watchlist: launches, ads crossing 21/30/60 days, kills, and spend-velocity moves." />
{:else}
	<div class="mx-auto flex max-w-2xl flex-col gap-5" data-part="activity">
		<p class="bn-dim text-[12px]">What changed across your watchlist, newest first.</p>
		{#each groups as [dayKey, rows] (dayKey)}
			<div>
				<div class="bn-dim mb-2 text-[11.5px]">{dayKey}</div>
				<div class="flex flex-col gap-1.5">
					{#each rows as s, i (`${s.at}-${i}`)}
						{@const ad = s.adId ? byAd.get(s.adId) : undefined}
						{@const Icon = ICON[s.type] ?? Zap}
						<button
							type="button"
							disabled={!ad}
							onclick={() => ad && onOpen(ad)}
							class="bn-pressable flex items-center gap-3 rounded-[10px] border border-[rgba(255,69,87,0.1)] bg-black/20 px-3 py-2 text-left {ad ? 'hover:bg-black/35' : 'cursor-default'}"
						>
							<Icon class="h-4 w-4 shrink-0" style="color: {s.type === 'winner' ? 'var(--bn-accent)' : 'var(--bn-text-3)'}" strokeWidth={1.7} />
							<span class="bn-muted min-w-0 flex-1 truncate text-[12.5px]">{s.message}</span>
							{#if ad && (ad.thumbnail || ad.image)}
								<img src={ad.thumbnail ?? ad.image ?? ''} alt="" class="h-8 w-11 shrink-0 rounded-[6px] object-cover" />
							{/if}
						</button>
					{/each}
				</div>
			</div>
		{/each}
	</div>
{/if}
