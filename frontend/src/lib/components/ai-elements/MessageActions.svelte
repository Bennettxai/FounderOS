<script lang="ts">
	import { learning } from '$lib/stores/learning';
	import type { FeedbackType } from '$lib/api/learning';

	interface Props {
		messageId?: string;
		conversationId?: string;
		agentType?: string;
		originalContent?: string;
		onCopy?: () => void;
		onRegenerate?: () => void;
		onFeedback?: (type: 'positive' | 'negative') => void;
		copied?: boolean;
		class?: string;
	}

	let {
		messageId,
		conversationId,
		agentType,
		originalContent,
		onCopy,
		onRegenerate,
		onFeedback,
		copied = false,
		class: className = ''
	}: Props = $props();

	let feedbackGiven = $state<'positive' | 'negative' | null>(null);
	let feedbackLoading = $state(false);

	async function handleFeedback(type: 'positive' | 'negative') {
		// Call parent callback if provided
		onFeedback?.(type);

		// If we have a messageId, record feedback to the backend
		if (messageId && !feedbackLoading) {
			feedbackLoading = true;
			try {
				const feedbackType: FeedbackType = type === 'positive' ? 'thumbs_up' : 'thumbs_down';
				await learning.recordFeedback({
					target_type: 'message',
					target_id: messageId,
					feedback_type: feedbackType,
					conversation_id: conversationId,
					agent_type: agentType,
					original_content: originalContent
				});
				feedbackGiven = type;
			} catch (error) {
				console.error('Failed to record feedback:', error);
			} finally {
				feedbackLoading = false;
			}
		} else if (!messageId) {
			// No messageId, just update visual state
			feedbackGiven = type;
		}
	}
</script>

<div class="ai-message-actions {className}">
	{#if onCopy}
		<button
			type="button"
			onclick={onCopy}
			class="btn-pill btn-pill-ghost ai-message-action"
			aria-label="Copy message"
		>
			{#if copied}
				<svg class="ai-message-action__icon ai-message-action__icon--success" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
				</svg>
				<span class="ai-message-action__label">Copied</span>
			{:else}
				<svg class="ai-message-action__icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
				</svg>
				<span class="ai-message-action__label">Copy</span>
			{/if}
		</button>
	{/if}

	{#if onRegenerate}
		<button
			type="button"
			onclick={onRegenerate}
			class="btn-pill btn-pill-ghost ai-message-action"
			aria-label="Regenerate response"
		>
			<svg class="ai-message-action__icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
			</svg>
			<span class="ai-message-action__label">Regenerate</span>
		</button>
	{/if}

	{#if onFeedback || messageId}
		<div class="ai-message-action__divider"></div>
		<button
			type="button"
			onclick={() => handleFeedback('positive')}
			class="ai-message-action ai-message-action--feedback"
			class:ai-message-action--active={feedbackGiven === 'positive'}
			disabled={feedbackLoading}
			aria-label="Good response"
		>
			<svg class="ai-message-action__icon" fill={feedbackGiven === 'positive' ? 'currentColor' : 'none'} stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14 10h4.764a2 2 0 011.789 2.894l-3.5 7A2 2 0 0115.263 21h-4.017c-.163 0-.326-.02-.485-.06L7 20m7-10V5a2 2 0 00-2-2h-.095c-.5 0-.905.405-.905.905 0 .714-.211 1.412-.608 2.006L7 11v9m7-10h-2M7 20H5a2 2 0 01-2-2v-6a2 2 0 012-2h2.5" />
			</svg>
		</button>
		<button
			type="button"
			onclick={() => handleFeedback('negative')}
			class="ai-message-action ai-message-action--feedback"
			class:ai-message-action--active={feedbackGiven === 'negative'}
			disabled={feedbackLoading}
			aria-label="Bad response"
		>
			<svg class="ai-message-action__icon" fill={feedbackGiven === 'negative' ? 'currentColor' : 'none'} stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 14H5.236a2 2 0 01-1.789-2.894l3.5-7A2 2 0 018.736 3h4.018a2 2 0 01.485.06l3.76.94m-7 10v5a2 2 0 002 2h.096c.5 0 .905-.405.905-.904 0-.715.211-1.413.608-2.008L17 13V4m-7 10h2m5-10h2a2 2 0 012 2v6a2 2 0 01-2 2h-2.5" />
			</svg>
		</button>
	{/if}
</div>

<style>
	.ai-message-actions {
		display: flex;
		align-items: center;
		gap: 0.25rem;
	}

	.ai-message-action {
		display: flex;
		align-items: center;
		gap: 0.375rem;
		padding: 0.375rem 0.625rem;
		font-size: 0.75rem;
		color: var(--muted-foreground);
		background-color: transparent;
		border: none;
		border-radius: 0.375rem;
		cursor: pointer;
		transition: all 0.2s ease;
	}

	.ai-message-action:hover {
		color: var(--foreground);
		background-color: var(--accent);
	}

	.ai-message-action--feedback {
		padding: 0.375rem;
	}

	.ai-message-action--feedback:hover {
		color: var(--foreground);
	}

	.ai-message-action--feedback.ai-message-action--active {
		color: var(--primary);
	}

	.ai-message-action--feedback:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.ai-message-action__icon {
		width: 0.875rem;
		height: 0.875rem;
	}

	.ai-message-action__icon--success {
		color: #4ade80;
	}

	.ai-message-action__label {
		font-weight: 400;
	}

	.ai-message-action__divider {
		width: 1px;
		height: 1rem;
		margin: 0 0.375rem;
		background-color: var(--border);
	}
</style>
