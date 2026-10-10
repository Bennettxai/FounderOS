<!-- Agent-run volume from the real run log (FounderOS v1 components/RunVolumeCard.tsx):
     a count-up headline, range chips, and per-day hatched bars; hover answers
     with the day's number, zero days keep a 2px stub instead of vanishing. -->
<script lang="ts">
	import { BigStat, SlabCard, chipClass } from '$lib/founderos/kit';
	import { fmtShort } from './format';

	let { data, known = true, i = 1 }: { data: { date: string; count: number }[]; known?: boolean; i?: number } = $props();

	const RANGES = [7, 14, 30] as const;
	const HUE = 'var(--send-activity)';
	let range = $state<7 | 14 | 30>(14);
	let hovered = $state<string | null>(null);
	const shown = $derived(data.slice(-range));
	const windowRuns = $derived(shown.reduce((s, p) => s + p.count, 0));
	const activeDays = $derived(shown.filter((p) => p.count > 0).length);
	const max = $derived(Math.max(1, ...shown.map((p) => p.count)));
	const hot = $derived(hovered ? shown.find((p) => p.date === hovered) : null);
</script>

<SlabCard {i} title="Agent Run Volume" sub="{range}d" class="flex flex-col">
	{#snippet action()}
		{#each RANGES as r (r)}
			<button data-lens="c" class={chipClass(range === r)} aria-pressed={range === r} onclick={() => (range = r)}>{r}d</button>
		{/each}
	{/snippet}
	<div class="flex flex-1 flex-col px-6 pb-5 pt-3" data-testid="run-volume">
		{#if known}
			{#key range}<BigStat value={windowRuns} chips={[{ tone: 'accent', text: `${activeDays} of ${shown.length} days active` }]} caption="agent runs in the last {range} days, from the real run log" />{/key}
		{:else}
			<BigStat value={null} caption="the run log could not be read" />
		{/if}
		<div class="mt-5 flex flex-1 items-end gap-[3px] border-t pt-5" style="min-height: 190px; border-color: var(--bn-border)">
			{#each shown as p, k (`${range}-${p.date}`)}
				{@const on = hovered === p.date}
				<!-- svelte-ignore a11y_no_static_element_interactions -->
				<div class="flex h-full flex-1 items-end self-stretch" onmouseenter={() => (hovered = p.date)} onmouseleave={() => (hovered = null)}>
					<div
						data-part="run-bar"
						class="bn-grow w-full rounded-t-[4px] transition-opacity duration-150"
						style="height: {Math.max(2, (p.count / max) * 170)}px; opacity: {on ? 1 : hovered ? 0.35 : p.count === 0 ? 0.3 : 0.85}; background: linear-gradient(0deg, transparent 60%, color-mix(in oklab, {HUE} 60%, white) 100%), repeating-linear-gradient(45deg, {HUE}, {HUE} 5px, color-mix(in oklab, {HUE} 45%, transparent) 5px, color-mix(in oklab, {HUE} 45%, transparent) 10px);{p.count > 0 ? ` box-shadow: 0 0 12px color-mix(in oklab, ${HUE} 40%, transparent);` : ''} animation-delay: {400 + k * 25}ms"
					></div>
				</div>
			{/each}
		</div>
		{#if shown.length > 0}
			<div class="bn-dim mt-2 flex justify-between font-mono text-[10.5px]">
				<span>{fmtShort(shown[0].date)}</span>
				<span class={hot ? 'bn-text' : ''}>{hot ? `${hot.count} runs · ${fmtShort(hot.date)}` : 'hover a day'}</span>
				<span>today</span>
			</div>
		{/if}
	</div>
</SlabCard>
