<!-- The card kebab (FounderOS v1 TradingBoard CardHead): a round button that
     opens a small glass menu: refresh from the feed, or open the broker. -->
<script lang="ts">
	import { ExternalLink, RefreshCw } from '$lib/founderos/icons';
	import './trading.css';

	let { title, onRefresh }: { title: string; onRefresh: () => void } = $props();
	let open = $state(false);
</script>

<div class="relative">
	<button
		type="button"
		aria-label="{title} menu"
		aria-haspopup="menu"
		aria-expanded={open}
		onclick={() => (open = !open)}
		data-lens="c"
		class="bn-pressable bn-dim bn-border tb-hover grid h-8 w-8 place-items-center rounded-full border text-[13px] leading-none">···</button
	>
	{#if open}
		<div class="fixed inset-0 z-[30]" role="presentation" onclick={() => (open = false)}></div>
		<div role="menu" aria-label="{title} actions" class="tb-menu absolute -right-1 top-9 z-[40] w-[220px] overflow-hidden rounded-[8px] py-1">
			<button
				type="button"
				role="menuitem"
				data-lens="c"
				class="tb-menu-item bn-pressable flex w-full items-center gap-2.5 px-3.5 py-2 text-left text-[12.5px]"
				onclick={() => {
					open = false;
					onRefresh();
				}}><RefreshCw size={13} strokeWidth={1.7} /> Refresh from the feed</button
			>
			<a
				href="https://robinhood.com/"
				target="_blank"
				rel="noreferrer"
				role="menuitem"
				onclick={() => (open = false)}
				class="tb-menu-item flex w-full items-center gap-2.5 px-3.5 py-2 text-left text-[12.5px]"><ExternalLink size={13} strokeWidth={1.7} /> Open Robinhood</a
			>
		</div>
	{/if}
</div>
