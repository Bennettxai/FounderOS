<!-- A card header's real kebab (DealBoard CardHead): refresh the data or open the hub. -->
<script lang="ts">
	import { ExternalLink, RefreshCw } from '$lib/founderos/icons';

	let { title, onRefresh, hubUrl }: { title: string; onRefresh: () => void; hubUrl: string | null } = $props();
	let open = $state(false);
</script>

<div class="relative">
	<button
		type="button"
		aria-label="{title} menu"
		aria-expanded={open}
		onclick={() => (open = !open)}
		class="bn-pressable bn-border bn-dim grid h-8 w-8 place-items-center rounded-full border text-[13px] leading-none hover:text-[var(--bn-text)]"
	>
		···
	</button>
	{#if open}
		<button type="button" class="fixed inset-0 z-[30] cursor-default" aria-label="Close menu" onclick={() => (open = false)}></button>
		<!-- prod: right-5 top-14 of the card head, i.e. 4px under and 4px past the button -->
		<div class="bn-border absolute right-[-4px] top-9 z-[40] w-[210px] overflow-hidden rounded-[8px] border py-1 backdrop-blur" style="background: color-mix(in oklab, var(--bn-bg-2) 95%, transparent)">
			<button
				type="button"
				onclick={() => {
					open = false;
					onRefresh();
				}}
				class="bn-menu-item bn-pressable bn-muted flex w-full items-center gap-2.5 px-3.5 py-2 text-left text-[12.5px] hover:text-[var(--bn-text)]"
			>
				<RefreshCw size={13} strokeWidth={1.7} /> Refresh from Notion
			</button>
			{#if hubUrl}
				<a
					href={hubUrl}
					target="_blank"
					rel="noreferrer"
					onclick={() => (open = false)}
					class="bn-menu-item bn-muted flex w-full items-center gap-2.5 px-3.5 py-2 text-left text-[12.5px] hover:text-[var(--bn-text)]"
				>
					<ExternalLink size={13} strokeWidth={1.7} /> Open in Notion
				</a>
			{/if}
		</div>
	{/if}
</div>

<style>
	.bn-menu-item:hover {
		background: color-mix(in oklab, var(--bn-text) 6%, transparent);
	}
</style>
