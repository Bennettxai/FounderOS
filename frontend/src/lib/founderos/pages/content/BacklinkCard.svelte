<!-- One of the three Content intelligence surfaces (FounderOS v1 BacklinkCard):
     the in-OS lead magnet index, or a Vantage intelligence link out. -->
<script lang="ts">
	import type { Component } from 'svelte';
	import { ArrowUpRight, ExternalLink } from '$lib/founderos/icons';

	let {
		href,
		icon,
		mark,
		title,
		sub,
		internal = false,
		meta
	}: {
		href: string;
		icon?: Component<Record<string, unknown>>;
		/** real brand mark (an image URL) instead of a lucide glyph */
		mark?: string;
		title: string;
		sub: string;
		internal?: boolean;
		meta?: string;
	} = $props();

	const Icon = $derived(icon);
</script>

<a
	{href}
	target={internal ? undefined : '_blank'}
	rel={internal ? undefined : 'noopener noreferrer'}
	data-lens="r"
	class="bn-pressable is-row bl-card group flex h-full items-start gap-3 rounded-[10px] border p-4"
>
	<span class="bl-mark bn-accent grid h-9 w-9 shrink-0 place-items-center rounded-[6px] border">
		{#if mark}
			<img src={mark} alt="" class="h-5 w-5 object-contain" />
		{:else if Icon}
			<Icon size={16} strokeWidth={1.8} />
		{/if}
	</span>
	<span class="min-w-0 flex-1">
		<span class="bn-text flex items-center gap-1.5 text-[13.5px] font-semibold">
			{title}
			{#if internal}<ArrowUpRight size={14} class="bl-go bn-dim" />{:else}<ExternalLink size={14} class="bl-go bn-dim" />{/if}
		</span>
		<span class="bn-dim mt-0.5 block text-[12px] leading-relaxed [text-wrap:pretty]">{sub}</span>
		<span class="bn-muted mt-1.5 block truncate font-mono text-[10px]">{meta ?? href.replace('https://', '')}</span>
	</span>
</a>

<style>
	.bl-card {
		border-color: var(--bn-border);
		background: var(--bn-bg);
	}
	.bl-card:hover :global(.bl-go) {
		color: var(--bn-accent);
	}
	.bl-mark {
		border-color: var(--bn-border);
		background: var(--bn-surface-2);
	}
</style>
