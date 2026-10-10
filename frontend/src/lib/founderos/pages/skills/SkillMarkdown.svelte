<!-- The reader's SKILL.md rendering (FounderOS v1 SkillsGrid Markdown). Text
     nodes only: a SKILL.md can never inject markup. -->
<script lang="ts">
	import { parseMarkdown, type Inline } from './skills';

	let { src }: { src: string } = $props();
	const blocks = $derived(parseMarkdown(src));
</script>

{#snippet parts(list: Inline[])}
	{#each list as p, i (i)}
		{#if p.kind === 'bold'}<strong class="bn-text font-semibold">{p.text}</strong>{:else if p.kind === 'code'}<code class="bn-accent rounded px-1 font-mono text-[11px]" style="background: var(--bn-surface-2)">{p.text}</code>{:else}<span>{p.text}</span>{/if}
	{/each}
{/snippet}

<div>
	{#each blocks as b, i (i)}
		{#if b.kind === 'h1'}
			<h1 class="bn-text mb-1 mt-4 text-[16px] font-bold">{@render parts(b.parts)}</h1>
		{:else if b.kind === 'h2'}
			<h2 class="bn-dim mb-1 mt-3 font-mono text-[11px] font-bold uppercase tracking-widest">{@render parts(b.parts)}</h2>
		{:else if b.kind === 'h3'}
			<h3 class="bn-text mt-2 text-[12.5px] font-semibold">{@render parts(b.parts)}</h3>
		{:else if b.kind === 'hr'}
			<hr class="my-2.5 bn-border" />
		{:else if b.kind === 'li'}
			<div class="bn-muted flex gap-2 text-[12.5px] leading-relaxed"><span class="bn-accent">·</span><span>{@render parts(b.parts)}</span></div>
		{:else if b.kind === 'gap'}
			<div class="h-2"></div>
		{:else if b.kind === 'pre'}
			<pre class="bn-muted my-2 overflow-x-auto rounded-md border p-3 font-mono text-[11px] leading-relaxed bn-border" style="background: var(--bn-bg)">{b.text}</pre>
		{:else if b.kind === 'p'}
			<p class="bn-muted text-[12.5px] leading-relaxed">{@render parts(b.parts)}</p>
		{/if}
	{/each}
</div>
