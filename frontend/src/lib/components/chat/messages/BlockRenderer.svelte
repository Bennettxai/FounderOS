<script lang="ts">
	import type { Block } from '$lib/api/conversations/types';

	interface Props {
		blocks: Block[];
		isStreaming?: boolean;
	}

	let { blocks, isStreaming = false }: Props = $props();

	function getCalloutIcon(type: string) {
		switch (type.toLowerCase()) {
			case 'note': return '📝';
			case 'tip': return '💡';
			case 'important': return '⚠️';
			case 'warning': return '🚫';
			case 'caution': return '🛑';
			default: return 'ℹ️';
		}
	}

	function getCalloutColor(type: string) {
		switch (type.toLowerCase()) {
			case 'note': return 'bg-blue-50 border-blue-200 text-blue-800';
			case 'tip': return 'bg-green-50 border-green-200 text-green-800';
			case 'important': return 'bg-yellow-50 border-yellow-200 text-yellow-800';
			case 'warning': return 'bg-orange-50 border-orange-200 text-orange-800';
			case 'caution': return 'bg-red-50 border-red-200 text-red-800';
			default: return 'bg-gray-50 border-gray-200 text-gray-800';
		}
	}
</script>

<div class="space-y-4">
	{#each blocks as block (block.id)}
		{#if block.type === 'paragraph'}
			<p class="text-[15px] leading-relaxed text-gray-800">
				{block.content}
			</p>
		{:else if block.type === 'heading'}
			{#if block.level === 1}
				<h1 class="text-2xl font-bold text-gray-900 mt-6 mb-4">{block.content}</h1>
			{:else if block.level === 2}
				<h2 class="text-xl font-semibold text-gray-900 mt-5 mb-3">{block.content}</h2>
			{:else}
				<h3 class="text-lg font-medium text-gray-900 mt-4 mb-2">{block.content}</h3>
			{/if}
		{:else if block.type === 'list'}
			<ul class="space-y-1.5 ml-4">
				{#each block.children || [] as item}
					<li class="flex items-start gap-2">
						<span class="text-gray-400 mt-1.5">•</span>
						<span class="text-gray-700">{item.content}</span>
					</li>
				{/each}
			</ul>
		{:else if block.type === 'code'}
			<div class="rounded-xl overflow-hidden border border-gray-200 bg-gray-900 my-4">
				{#if block.language}
					<div class="px-4 py-2 border-b border-gray-800 bg-gray-800/50 flex items-center justify-between">
						<span class="text-xs font-mono text-gray-400">{block.language}</span>
					</div>
				{/if}
				<pre class="p-4 overflow-x-auto text-sm font-mono text-gray-300"><code>{block.content}</code></pre>
			</div>
		{:else if block.type === 'blockquote'}
			<blockquote class="pl-4 border-l-4 border-gray-200 italic text-gray-600 my-4">
				{block.content}
			</blockquote>
		{:else if block.type === 'callout'}
			<div class="p-4 rounded-xl border-l-4 {getCalloutColor(String(block.metadata?.callout_type || 'note'))} my-4">
				<div class="flex items-center gap-2 mb-1 font-medium">
					<span>{getCalloutIcon(String(block.metadata?.callout_type || 'note'))}</span>
					<span class="uppercase text-xs tracking-wider">{block.metadata?.callout_type || 'Note'}</span>
				</div>
				<div class="text-sm opacity-90">{block.content}</div>
			</div>
		{:else if block.type === 'table'}
			<div class="overflow-x-auto my-4 rounded-xl border border-gray-200 text-sm">
				<table class="w-full text-left border-collapse">
					<thead>
						<tr class="bg-gray-50 border-b border-gray-200">
							{#each (block.children?.[0]?.children || []) as cell}
								<th class="px-4 py-2 font-semibold text-gray-700">{cell.content}</th>
							{/each}
						</tr>
					</thead>
					<tbody class="divide-y divide-gray-100">
						{#each (block.children?.slice(1) || []) as row}
							<tr>
								{#each (row.children || []) as cell}
									<td class="px-4 py-2 text-gray-600">{cell.content}</td>
								{/each}
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{:else if block.type === 'thinking'}
			<details class="bg-purple-50/50 border border-purple-100 rounded-xl p-3 my-4">
				<summary class="text-xs font-medium text-purple-700 cursor-pointer hover:text-purple-900 transition-colors">
					Chain of Thought
				</summary>
				<div class="mt-2 text-xs text-purple-600 leading-relaxed font-mono whitespace-pre-wrap">
					{block.content}
				</div>
			</details>
		{/if}
	{/each}
	{#if isStreaming}
		<span class="inline-block w-1.5 h-5 bg-gray-400 animate-pulse ml-0.5 rounded-sm align-middle"></span>
	{/if}
</div>
