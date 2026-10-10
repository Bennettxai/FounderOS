<!-- Memory note detail (FounderOS v1 KnowledgeDetail MemoryNoteCard): one page
     of the Optimal Engine constellation. The engines report no word or chunk
     counts, so the meta line shows what they do report. -->
<script lang="ts">
	import { FileText } from '$lib/founderos/icons';
	import PanelHeader from './PanelHeader.svelte';
	import SectionLabel from './SectionLabel.svelte';
	import WikiLink from './WikiLink.svelte';
	import type { MemoryNode } from '../memory-core';

	let { note, color, onBack, onClose }: { note: MemoryNode; color: string; onBack?: () => void; onClose?: () => void } = $props();
	const meta = $derived(
		note.wordCount != null
			? `${note.wordCount} words · ${note.chunks ?? 0} chunk${note.chunks === 1 ? '' : 's'}`
			: `${note.genre ? `${note.genre} · ` : ''}${note.links} link${note.links === 1 ? '' : 's'}`
	);
</script>

<div class="flex h-full flex-col" data-card="memory-note">
	<PanelHeader title={note.label} sub="memory · optimal engine" {color} {onBack} {onClose} />
	<div class="min-h-0 flex-1 overflow-y-auto overscroll-contain px-3 py-2.5">
		<div class="mb-3 flex items-center gap-2">
			<span class="rounded-[5px] border px-1.5 py-0.5 font-mono text-[9px] uppercase tracking-wide" style="border-color: {color}; color: {color}">{note.folder}</span>
			<span class="bn-dim font-mono text-[9.5px]">{meta}</span>
		</div>
		{#if note.excerpt}
			<p class="bn-muted mb-3 text-[11px] leading-relaxed">{note.excerpt}</p>
		{:else}
			<p class="bn-dim mb-3 font-mono text-[10.5px]">no excerpt</p>
		{/if}
		<SectionLabel icon={FileText}>source</SectionLabel>
		<WikiLink label={`${note.id}.md`} />
	</div>
</div>
