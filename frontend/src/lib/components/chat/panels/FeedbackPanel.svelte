<script lang="ts">
	import { learning } from '$lib/stores/learning';
	import { onMount } from 'svelte';
	import { fade, slide } from 'svelte/transition';

	interface Props {
		agentType?: string;
	}

	let { agentType }: Props = $props();

	onMount(() => {
		learning.loadLearnings(agentType);
		learning.detectPatterns();
	});

	function getFeedbackIcon(type: string) {
		switch (type) {
			case 'thumbs_up':
				return `<path stroke-linecap="round" stroke-linejoin="round" d="M14 10h4.764a2 2 0 011.789 2.894l-3.5 7A2 2 0 0115.263 21h-4.017c-.163 0-.326-.02-.485-.06L7 20m7-10V5a2 2 0 00-2-2h-.095c-.5 0-.905.405-.905.905 0 .714-.211 1.412-.608 2.006L7 11v9m7-10h-2M7 20H5a2 2 0 01-2-2v-6a2 2 0 012-2h2.5" />`;
			case 'thumbs_down':
				return `<path stroke-linecap="round" stroke-linejoin="round" d="M10 14H5.236a2 2 0 01-1.789-2.894l3.5-7A2 2 0 018.736 3h4.018a2 2 0 01.485.06l3.76.94m-7 10v5a2 2 0 002 2h.096c.5 0 .905-.405.905-.904 0-.715.211-1.413.608-2.008L17 13V4m-7 10h2m5-10h2a2 2 0 012 2v6a2 2 0 01-2 2h-2.5" />`;
			case 'correction':
				return `<path stroke-linecap="round" stroke-linejoin="round" d="m16.862 4.487 1.687-1.688a1.875 1.875 0 1 1 2.652 2.652L10.582 16.07a4.5 4.5 0 0 1-1.897 1.13L6 18l.8-2.685a4.5 4.5 0 0 1 1.13-1.897l8.932-8.931Zm0 0L19.5 7.125M18 14v4.75A2.25 2.25 0 0 1 15.75 21H5.25A2.25 2.25 0 0 1 3 18.75V8.25A2.25 2.25 0 0 1 5.25 6H10" />`;
			default:
				return `<path stroke-linecap="round" stroke-linejoin="round" d="M7.5 8.25h9m-9 3h9m-9 3h9m-6.75-12.75h10.5a2.25 2.25 0 0 1 2.25 2.25v13.5a2.25 2.25 0 0 1-2.25 2.25H6.75A2.25 2.25 0 0 1 4.5 19.5V5.25a2.25 2.25 0 0 1 2.25-2.25Z" />`;
		}
	}
</script>

<div class="feedback-panel">
	<div class="section">
		<h3 class="section-title">AI Learnings</h3>
		<p class="section-desc">What the AI has learned from your feedback and interactions.</p>

		{#if $learning.loading && $learning.learnings.length === 0}
			<div class="loading-state">
				<div class="spinner"></div>
				<p>Loading learnings...</p>
			</div>
		{:else if $learning.learnings.length === 0}
			<div class="empty-state">
				<p>No learnings recorded yet.</p>
				<p class="hint">Give feedback to help the AI improve.</p>
			</div>
		{:else}
			<div class="learning-list">
				{#each $learning.learnings as item (item.id)}
					<div class="learning-card" transition:slide>
						<div class="learning-type">{item.learning_type}</div>
						<p class="learning-content">{item.learning_content}</p>
						{#if item.learning_summary}
							<p class="learning-summary">{item.learning_summary}</p>
						{/if}
						<div class="learning-meta">
							<span class="confidence">Confidence: {Math.round(item.confidence_score * 100)}%</span>
							<span class="applied">Applied {item.times_applied} times</span>
						</div>
					</div>
				{/each}
			</div>
		{/if}
	</div>

	<div class="section">
		<h3 class="section-title">Recent Feedback</h3>
		<div class="feedback-list">
			{#each $learning.feedbackHistory.slice(0, 10) as feedback (feedback.id)}
				<div class="feedback-item" transition:fade>
					<div class="feedback-icon" class:positive={feedback.feedback_type === 'thumbs_up'} class:negative={feedback.feedback_type === 'thumbs_down'}>
						<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" width="14" height="14">
							{@html getFeedbackIcon(feedback.feedback_type)}
						</svg>
					</div>
					<div class="feedback-info">
						<div class="feedback-target">{feedback.target_type}: {feedback.target_id.slice(0, 8)}...</div>
						{#if feedback.feedback_value}
							<div class="feedback-comment">"{feedback.feedback_value}"</div>
						{/if}
						<div class="feedback-time">{new Date(feedback.created_at).toLocaleString()}</div>
					</div>
					{#if feedback.was_processed}
						<div class="processed-badge" title="Processed by AI Learning System">
							<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" width="12" height="12">
								<path fill-rule="evenodd" d="M10 18a8 8 0 1 0 0-16 8 8 0 0 0 0 16Zm3.857-9.809a.75.75 0 0 0-1.214-.882l-3.483 4.79-1.88-1.88a.75.75 0 1 0-1.06 1.061l2.5 2.5a.75.75 0 0 0 1.137-.089l4-5.5Z" clip-rule="evenodd" />
							</svg>
						</div>
					{/if}
				</div>
			{:else}
				<p class="empty-list-text">No recent feedback history.</p>
			{/each}
		</div>
	</div>
</div>

<style>
	.feedback-panel {
		display: flex;
		flex-direction: column;
		gap: 24px;
		padding: 16px;
	}

	.section-title {
		font-size: 14px;
		font-weight: 600;
		color: var(--color-text);
		margin-bottom: 4px;
	}

	.section-desc {
		font-size: 12px;
		color: var(--color-text-muted);
		margin-bottom: 12px;
	}

	.loading-state, .empty-state {
		display: flex;
		flex-direction: column;
		align-items: center;
		padding: 24px;
		text-align: center;
		color: var(--color-text-muted);
		font-size: 13px;
		background: var(--color-bg-secondary);
		border-radius: 8px;
	}

	.spinner {
		width: 20px;
		height: 20px;
		border: 2px solid var(--color-border);
		border-top-color: var(--color-text);
		border-radius: 50%;
		animation: spin 0.8s linear infinite;
		margin-bottom: 8px;
	}

	@keyframes spin {
		to { transform: rotate(360deg); }
	}

	.learning-list {
		display: flex;
		flex-direction: column;
		gap: 12px;
	}

	.learning-card {
		padding: 12px;
		background: var(--color-bg-secondary);
		border-radius: 8px;
		border-left: 3px solid var(--color-primary);
	}

	.learning-type {
		font-size: 10px;
		font-weight: 700;
		text-transform: uppercase;
		color: var(--color-primary);
		margin-bottom: 4px;
	}

	.learning-content {
		font-size: 13px;
		color: var(--color-text);
		margin-bottom: 4px;
		line-height: 1.4;
	}

	.learning-summary {
		font-size: 12px;
		color: var(--color-text-muted);
		font-style: italic;
		margin-bottom: 8px;
	}

	.learning-meta {
		display: flex;
		justify-content: space-between;
		font-size: 11px;
		color: var(--color-text-muted);
		opacity: 0.8;
	}

	.feedback-list {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.feedback-item {
		display: flex;
		align-items: flex-start;
		gap: 10px;
		padding: 10px;
		background: var(--color-bg-secondary);
		border-radius: 8px;
		position: relative;
	}

	.feedback-icon {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 24px;
		height: 24px;
		border-radius: 50%;
		background: var(--color-bg-tertiary);
		flex-shrink: 0;
	}

	.feedback-icon.positive { color: #22c55e; background: rgba(34, 197, 94, 0.1); }
	.feedback-icon.negative { color: #ef4444; background: rgba(239, 68, 68, 0.1); }

	.feedback-info {
		flex: 1;
		min-width: 0;
	}

	.feedback-target {
		font-size: 11px;
		font-weight: 600;
		color: var(--color-text-muted);
		text-transform: uppercase;
	}

	.feedback-comment {
		font-size: 12px;
		color: var(--color-text);
		margin: 2px 0;
	}

	.feedback-time {
		font-size: 10px;
		color: var(--color-text-muted);
	}

	.processed-badge {
		position: absolute;
		top: 10px;
		right: 10px;
		color: #22c55e;
	}

	.empty-list-text {
		font-size: 12px;
		color: var(--color-text-muted);
		text-align: center;
		padding: 12px;
	}
</style>
