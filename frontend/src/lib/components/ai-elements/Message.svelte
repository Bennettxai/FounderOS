<script lang="ts">
	import { fly } from 'svelte/transition';
	import type { Snippet } from 'svelte';

	interface Props {
		role: 'user' | 'assistant';
		children: Snippet;
		avatar?: Snippet;
		actions?: Snippet;
		isStreaming?: boolean;
		class?: string;
	}

	let {
		role,
		children,
		avatar,
		actions,
		isStreaming = false,
		class: className = ''
	}: Props = $props();
</script>

<div
	class="ai-message ai-message--{role} {className}"
	class:ai-message--streaming={isStreaming}
	in:fly={{ y: 10, duration: 200 }}
>
	{#if role === 'user'}
		<!-- User message: right-aligned bubble -->
		<div class="ai-message__content ai-message__content--user">
			<div class="ai-message__bubble ai-message__bubble--user">
				{@render children()}
			</div>
		</div>
	{:else}
		<!-- Assistant message: left-aligned with avatar -->
		<div class="ai-message__content ai-message__content--assistant">
			{#if avatar}
				<div class="ai-message__avatar">
					{@render avatar()}
				</div>
			{:else}
				<div class="ai-message__avatar ai-message__avatar--default">
					<svg class="ai-message__avatar-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z" />
					</svg>
				</div>
			{/if}
			<div class="ai-message__body">
				<div class="ai-message__text">
					{@render children()}
					{#if isStreaming}
						<span class="ai-message__cursor"></span>
					{/if}
				</div>
				{#if actions && !isStreaming}
					<div class="ai-message__actions">
						{@render actions()}
					</div>
				{/if}
			</div>
		</div>
	{/if}
</div>

<style>
	.ai-message {
		width: 100%;
	}

	.ai-message__content {
		display: flex;
		gap: 0.75rem;
	}

	.ai-message__content--user {
		justify-content: flex-end;
	}

	.ai-message__content--assistant {
		justify-content: flex-start;
	}

	/* User bubble */
	.ai-message__bubble--user {
		max-width: 80%;
		padding: 0.875rem 1.125rem;
		border-radius: 1.5rem;
		background-color: var(--primary);
		color: var(--primary-foreground);
		font-size: 0.9375rem;
		line-height: 1.6;
		white-space: pre-wrap;
		word-wrap: break-word;
	}

	/* Assistant avatar */
	.ai-message__avatar {
		flex-shrink: 0;
		width: 2rem;
		height: 2rem;
		border-radius: 50%;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.ai-message__avatar--default {
		background: linear-gradient(135deg, #3b82f6 0%, #2563eb 100%);
	}

	.ai-message__avatar-icon {
		width: 1rem;
		height: 1rem;
		color: white;
	}

	/* Assistant body */
	.ai-message__body {
		flex: 1;
		min-width: 0;
		padding-top: 0.25rem;
	}

	.ai-message__text {
		font-size: 0.9375rem;
		line-height: 1.7;
		color: var(--foreground);
		white-space: pre-wrap;
		word-wrap: break-word;
	}

	/* Streaming cursor */
	.ai-message__cursor {
		display: inline-block;
		width: 0.5rem;
		height: 1.25rem;
		margin-left: 0.25rem;
		background-color: #3b82f6;
		border-radius: 0.125rem;
		vertical-align: text-bottom;
		animation: cursor-blink 1s ease-in-out infinite;
	}

	@keyframes cursor-blink {
		0%, 50% { opacity: 1; }
		51%, 100% { opacity: 0; }
	}

	/* Actions */
	.ai-message__actions {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		margin-top: 0.75rem;
	}

	/* Dark mode handled via CSS variables */
	:global(.dark) .ai-message__bubble--user {
		background-color: var(--primary);
		color: var(--primary-foreground);
	}
</style>
