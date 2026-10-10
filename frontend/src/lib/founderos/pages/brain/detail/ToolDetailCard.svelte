<!-- Tool wiki card (FounderOS v1 KnowledgeDetail ToolDetailCard): what its page
     says, how it is wired, and who uses it. The port has no per-tool page
     (G-Brain is retired), so the page line says so honestly. -->
<script lang="ts">
	import { FileText, Link2, User } from '$lib/founderos/icons';
	import PanelHeader from './PanelHeader.svelte';
	import SectionLabel from './SectionLabel.svelte';
	import WikiLink from './WikiLink.svelte';
	import type { ToolWiki, WikiRef } from '../kg';

	let { wiki, onBack, onClose }: { wiki: ToolWiki; onBack?: () => void; onClose?: () => void } = $props();
	const fields = $derived(Object.entries(wiki.fields));
</script>

{#snippet refs(label: string, list: WikiRef[])}
	{#if list.length > 0}
		<SectionLabel icon={Link2}>{label} ({list.length})</SectionLabel>
		<div class="mb-4 flex flex-wrap gap-1.5">
			{#each list as r, i (`${r.target}-${i}`)}
				<span class={r.slug ? undefined : 'line-through opacity-50'}><WikiLink label={r.slug ? r.title : r.target} /></span>
			{/each}
		</div>
	{/if}
{/snippet}

<div class="flex h-full flex-col" data-card="tool">
	<PanelHeader title={wiki.name} sub={wiki.kind} {onBack} {onClose} />
	<div class="min-h-0 flex-1 overflow-y-auto overscroll-contain px-3 py-2.5">
		{#if wiki.summary}<p class="bn-muted mb-3 text-[11px] leading-relaxed">{wiki.summary}</p>{/if}
		{#if fields.length > 0}
			<div class="mb-3 flex flex-col gap-0.5">
				{#each fields as [k, v] (k)}
					<div class="flex gap-2 font-mono text-[10.5px]"><span class="bn-dim shrink-0">{k}</span><span class="bn-muted">{v}</span></div>
				{/each}
			</div>
		{/if}

		<SectionLabel icon={FileText}>page{wiki.hasPage ? ` · ${wiki.path}` : ''}</SectionLabel>
		<div class="mb-4 flex items-center gap-2">
			{#if wiki.hasPage}
				<WikiLink label={`${wiki.slug}.md`} mcp={wiki.mcp} />
			{:else}
				<span class="bn-dim font-mono text-[10.5px]">no page in the brain yet</span>
			{/if}
		</div>

		{@render refs('links to', wiki.links)}
		{@render refs('linked from', wiki.backlinks)}

		<SectionLabel icon={User}>used by ({wiki.usedBy.length})</SectionLabel>
		{#if wiki.usedBy.length === 0}
			<p class="bn-dim font-mono text-[10.5px]">no agents in view</p>
		{:else}
			<div class="flex flex-col gap-1">
				{#each wiki.usedBy as n (n)}<span class="bn-muted font-mono text-[11px]">{n}</span>{/each}
			</div>
		{/if}
	</div>
</div>
