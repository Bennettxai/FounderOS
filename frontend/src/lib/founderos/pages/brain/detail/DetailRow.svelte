<!-- A clickable row used for agents, people, SOPs and skills in the detail cards. -->
<script lang="ts">
	import { ChevronRight } from '$lib/founderos/icons';
	let { color, title, sub, badge, onclick }: { color: string; title: string; sub?: string; badge?: string; onclick?: () => void } = $props();
</script>

<button
	type="button"
	{onclick}
	disabled={!onclick}
	class="bn-pressable kg-b kg-surf group flex w-full items-center gap-2 rounded-[8px] border px-2.5 py-1.5 text-left {onclick ? 'kg-h-bs' : 'cursor-default'}"
>
	<span class="h-2 w-2 shrink-0 rounded-full" style="background: {color}"></span>
	<span class="min-w-0 flex-1">
		<span class="bn-text block truncate text-[11.5px] font-semibold">{title}</span>
		{#if sub}<span class="bn-dim block truncate font-mono text-[9.5px]">{sub}</span>{/if}
	</span>
	{#if badge}<span class="bn-dim shrink-0 font-mono text-[9px] uppercase tracking-wide">{badge}</span>{/if}
	{#if onclick}<span class="bn-dim kg-gh-show shrink-0 opacity-0 transition-opacity"><ChevronRight size={14} /></span>{/if}
</button>
