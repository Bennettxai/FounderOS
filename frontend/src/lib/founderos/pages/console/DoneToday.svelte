<!-- What the OS actually finished since local midnight: OK agent runs (collapsed
     so a 30-minute cron costs one line) plus Stripe charges that landed today. -->
<script lang="ts">
	import { Label } from '$lib/founderos/kit';
	import { TONE_COLOR, relativeTime, type DoneItem } from './console';

	let { items, count }: { items: DoneItem[]; count: number } = $props();
</script>

<div data-part="card" class="bn-card relative overflow-hidden">
	<div data-part="card-head" class="flex items-center gap-3 border-b px-4 py-2.5" style:border-color="var(--bn-border)">
		<Label>Done today</Label>
		<span class="bn-dim font-mono text-[10px]">{count} since midnight</span>
		<a href="/os/agents" class="bn-linky bn-dim ml-auto font-mono text-[10px]">runs →</a>
	</div>
	{#if items.length === 0}
		<div class="bn-dim px-4 py-5 font-mono text-[11px]">nothing finished yet today</div>
	{:else}
		<ul class="flex flex-col">
			{#each items as d (d.key)}
				<li data-part="row" class="flex items-baseline gap-2.5 border-b px-4 py-2 font-mono text-[11px] last:border-b-0" style:border-color="var(--bn-hairline)">
					<span class="shrink-0 font-bold" style:color={TONE_COLOR[d.tone]}>{d.head}</span>
					<span class="bn-dim min-w-0 flex-1 truncate">{d.body}</span>
					<span class="bn-dim shrink-0">{relativeTime(d.at)}</span>
				</li>
			{/each}
		</ul>
	{/if}
</div>
