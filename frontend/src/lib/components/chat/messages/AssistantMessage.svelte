<script lang="ts">
	import { fly, fade } from 'svelte/transition';
	import { renderMarkdown } from '$lib/utils/markdownRenderer';

	import BlockRenderer from './BlockRenderer.svelte';
	import type { Block } from '$lib/api/conversations/types';

	interface Props {
		content: string;
		blocks?: Block[];
		timestamp?: string;
		isStreaming?: boolean;
		model?: string;
		onCopy?: () => void;
		onRegenerate?: () => void;
		onFeedback?: (type: 'good' | 'bad') => void;
	}

	let { content, blocks, timestamp, isStreaming = false, model, onCopy, onRegenerate, onFeedback }: Props = $props();

	let copied = $state(false);

	function formatTime(dateStr: string) {
		return new Date(dateStr).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
	}

	function handleCopy() {
		navigator.clipboard.writeText(content);
		copied = true;
		setTimeout(() => copied = false, 2000);
		onCopy?.();
	}

</script>

<div class="flex justify-start group" in:fly={{ y: 20, duration: 300 }}>
	<div class="max-w-[85%]">
		<div class="flex items-center gap-2 mb-2">
			<div class="asst-avatar flex items-center justify-center">
				<span class="asst-avatar-label">OSA</span>
			</div>
			<span class="text-sm font-medium asst-label">Assistant</span>
			{#if model}
				<span class="text-xs asst-model-badge px-2 py-0.5 rounded">{model}</span>
			{/if}
		</div>

		<div class="asst-bubble asst-bubble-radius px-5 py-4">
			<div class="text-[15px] leading-relaxed asst-body prose prose-sm max-w-none">
				{#if blocks && blocks.length > 0}
					<BlockRenderer {blocks} {isStreaming} />
				{:else}
					{@html renderMarkdown(content)}
					{#if isStreaming}
						<span class="inline-block w-1.5 h-5 asst-cursor animate-pulse ml-0.5 rounded-sm align-middle"></span>
					{/if}
				{/if}
			</div>
		</div>

		{#if !isStreaming}
			<div class="flex items-center gap-3 mt-2 px-1 msg-timestamp">
				{#if timestamp}
					<span class="text-xs msg-meta-text">{formatTime(timestamp)}</span>
				{/if}

				<div class="flex items-center gap-1">
					<button
						onclick={handleCopy}
						class="flex items-center gap-1 px-2 py-1 text-xs asst-action-btn rounded transition-colors"
					>
						{#if copied}
							<svg class="w-3.5 h-3.5 text-green-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
							</svg>
							<span class="text-green-600">Copied</span>
						{:else}
							<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
							</svg>
							<span>Copy</span>
						{/if}
					</button>

					{#if onRegenerate}
						<button
							onclick={onRegenerate}
							class="flex items-center gap-1 px-2 py-1 text-xs asst-action-btn rounded transition-colors"
						>
							<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
							</svg>
							<span>Regenerate</span>
						</button>
					{/if}

					{#if onFeedback}
						<div class="flex items-center gap-0.5 ml-2 asst-feedback-divider pl-2">
							<button
								onclick={() => onFeedback?.('good')}
								class="p-1.5 msg-meta-text hover:text-green-600 hover:bg-green-50 rounded transition-colors"
								title="Good response"
							>
								<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14 10h4.764a2 2 0 011.789 2.894l-3.5 7A2 2 0 0115.263 21h-4.017c-.163 0-.326-.02-.485-.06L7 20m7-10V5a2 2 0 00-2-2h-.095c-.5 0-.905.405-.905.905 0 .714-.211 1.412-.608 2.006L7 11v9m7-10h-2M7 20H5a2 2 0 01-2-2v-6a2 2 0 012-2h2.5" />
								</svg>
							</button>
							<button
								onclick={() => onFeedback?.('bad')}
								class="p-1.5 msg-meta-text hover:text-red-600 hover:bg-red-50 rounded transition-colors"
								title="Bad response"
							>
								<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 14H5.236a2 2 0 01-1.789-2.894l3.5-7A2 2 0 018.736 3h4.018a2 2 0 01.485.06l3.76.94m-7 10v5a2 2 0 002 2h.096c.5 0 .905-.405.905-.904 0-.715.211-1.413.608-2.008L17 13V4m-7 10h2m5-10h2a2 2 0 012 2v6a2 2 0 01-2 2h-2.5" />
								</svg>
							</button>
						</div>
					{/if}
				</div>
			</div>
		{/if}
	</div>
</div>

<style>
	.asst-avatar {
		width: 28px;
		height: 28px;
		border-radius: 50%;
		background: linear-gradient(135deg, #111 0%, #333 100%);
		flex-shrink: 0;
	}

	.asst-avatar-label {
		font-size: 8px;
		font-weight: 700;
		color: #fff;
		letter-spacing: 0.02em;
	}

	.asst-label {
		color: var(--dt);
	}

	.asst-model-badge {
		color: var(--dt3);
		background: var(--dbg3);
	}

	.asst-bubble {
		background: var(--dbg2);
		border: 1px solid var(--dbd);
	}

	.asst-bubble-radius {
		border-radius: 1rem 1rem 1rem 0.25rem;
	}

	.asst-body {
		color: var(--dt);
	}

	.asst-cursor {
		background: var(--dt3);
	}

	.msg-meta-text {
		color: var(--dt3);
	}

	.msg-timestamp {
		opacity: 0;
		transition: opacity 0.15s;
	}

	.group:hover .msg-timestamp {
		opacity: 1;
	}

	.asst-action-btn {
		color: var(--dt2);
	}

	.asst-action-btn:hover {
		color: var(--dt);
		background: var(--dbg3);
	}

	.asst-feedback-divider {
		border-left: 1px solid var(--dbd);
	}
</style>
