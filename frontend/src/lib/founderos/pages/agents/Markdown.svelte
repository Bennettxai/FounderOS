<!-- An agent deliverable rendered as the document it is (FounderOS v1
     components/Markdown.tsx). Built from parsed blocks, never innerHTML, so an
     agent-authored file cannot inject markup. -->
<script lang="ts">
	import { parseInline, parseMarkdown } from './markdown-blocks';

	let { text }: { text: string } = $props();
	const blocks = $derived(parseMarkdown(text));
	const HEADING_SIZE = ['text-[15px]', 'text-[14px]', 'text-[13px]', 'text-[12px]', 'text-[12px]', 'text-[12px]'];
</script>

{#snippet marks(src: string)}
	{#each parseInline(src) as n, i (i)}
		{#if n.type === 'bold'}<strong class="bn-text font-semibold">{n.text}</strong>{:else if n.type === 'italic'}<em class="italic">{n.text}</em>{:else if n.type === 'code'}<code
				class="bn-md-code border px-1 py-px font-mono text-[11px]">{n.text}</code
			>{:else if n.type === 'link'}<a href={n.href} target="_blank" rel="noreferrer noopener" class="bn-md-link underline underline-offset-2">{n.text}</a
			>{:else}{n.text}{/if}
	{/each}
{/snippet}

{#if blocks.length === 0}
	<p class="bn-dim font-mono text-[10.5px]">This file is empty.</p>
{:else}
	<div class="bn-muted space-y-3 text-[12.5px] leading-relaxed">
		{#each blocks as b, i (i)}
			{#if b.type === 'heading'}
				<h3 class="{HEADING_SIZE[b.level - 1] ?? 'text-[12px]'} bn-text pt-1 font-semibold">{@render marks(b.text)}</h3>
			{:else if b.type === 'paragraph'}
				<p>{@render marks(b.text)}</p>
			{:else if b.type === 'list'}
				{#if b.ordered}
					<ol class="list-decimal space-y-1 pl-5">
						{#each b.items as it, j (j)}<li>{@render marks(it)}</li>{/each}
					</ol>
				{:else}
					<ul class="list-disc space-y-1 pl-5">
						{#each b.items as it, j (j)}<li>{@render marks(it)}</li>{/each}
					</ul>
				{/if}
			{:else if b.type === 'quote'}
				<blockquote class="bn-dim bn-md-quote border-l-2 pl-3 italic">{@render marks(b.text)}</blockquote>
			{:else if b.type === 'code'}
				<pre class="bn-md-pre bn-muted overflow-x-auto border p-2.5 font-mono text-[11px]">{b.text}</pre>
			{:else if b.type === 'table'}
				<!-- Wide tables scroll inside their own box rather than pushing the panel sideways. -->
				<div class="bn-md-box overflow-x-auto border">
					<table class="w-full border-collapse text-[11.5px]">
						<thead>
							<tr class="bn-md-head border-b">
								{#each b.header as h, j (j)}<th class="bn-text px-2 py-1.5 text-left font-semibold">{@render marks(h)}</th>{/each}
							</tr>
						</thead>
						<tbody>
							{#each b.rows as r, j (j)}
								<tr class="bn-md-row border-b last:border-b-0">
									{#each r as c, k (k)}<td class="px-2 py-1.5 align-top">{@render marks(c)}</td>{/each}
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{:else if b.type === 'rule'}
				<hr class="bn-md-row" />
			{/if}
		{/each}
	</div>
{/if}

<style>
	.bn-md-code {
		border-color: var(--bn-border);
		border-radius: var(--bn-r-chip, 5px);
		background: var(--bn-bg);
		color: var(--bn-text);
	}
	.bn-md-link {
		color: var(--bn-text);
		text-decoration-color: var(--bn-text-3);
	}
	.bn-md-link:hover {
		text-decoration-color: var(--bn-text);
	}
	.bn-md-quote {
		border-color: var(--bn-border-strong);
	}
	.bn-md-pre,
	.bn-md-box {
		border-color: var(--bn-border);
		border-radius: var(--bn-r-card, 8px);
	}
	.bn-md-pre {
		background: var(--bn-bg);
	}
	.bn-md-head {
		border-color: var(--bn-border-strong);
	}
	.bn-md-row {
		border-color: var(--bn-border);
	}
</style>
