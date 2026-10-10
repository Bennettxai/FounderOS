<!-- Card header (FounderOS v1 TradingBoard CardHead): the title, its meta line
     under it, and the kebab (refresh from the feed, open the broker). Its own
     data-part: the kit's Home-card rules on [data-part='card-head'] (10px
     foot, pill buttons) are not this head. -->
<script lang="ts">
	import CardMenu from './CardMenu.svelte';

	let { title, meta, onRefresh }: { title: string; meta?: string; onRefresh: () => void } = $props();
</script>

<div data-part="tb-head" class="relative flex items-start justify-between gap-3 px-6 pt-5">
	<div class="min-w-0">
		<h2 class="bn-text text-[19px] font-semibold tracking-[-0.01em]">{title}</h2>
		{#if meta}<div data-part="card-meta" class="bn-dim mt-0.5 truncate font-mono text-[10.5px]">{meta}</div>{/if}
	</div>
	<div class="shrink-0"><CardMenu {title} {onRefresh} /></div>
</div>
