<!--
  CompletionScreen.svelte
  Final onboarding summary with checkmarks
-->
<script lang="ts">
	import { Button } from '$lib/ui';
	import { CheckIcon } from './icons';
	import PurpleOrb from './PurpleOrb.svelte';

	interface CompletionItem {
		label: string;
		completed: boolean;
	}

	interface Props {
		title?: string;
		subtitle?: string;
		items?: CompletionItem[];
		primaryAction?: string;
		secondaryAction?: string;
		onPrimaryClick?: () => void;
		onSecondaryClick?: () => void;
		class?: string;
	}

	let {
		title = 'You\'re all set!',
		subtitle = 'Your workspace is ready. Here\'s what we\'ve configured for you:',
		items = [],
		primaryAction = 'Go to Dashboard',
		secondaryAction,
		onPrimaryClick,
		onSecondaryClick,
		class: className = ''
	}: Props = $props();
</script>

<div class="completion-screen {className}">
	<div class="orb-container">
		<PurpleOrb size="lg" isPulsing={false} />
	</div>

	<div class="content">
		<h1 class="title">{title}</h1>
		<p class="subtitle">{subtitle}</p>

		{#if items.length > 0}
			<ul class="checklist">
				{#each items as item, i (i)}
					<li class="checklist-item" class:is-completed={item.completed}>
						<div class="check-circle">
							{#if item.completed}
								<CheckIcon size={16} />
							{/if}
						</div>
						<span class="item-label">{item.label}</span>
					</li>
				{/each}
			</ul>
		{/if}

		<div class="actions">
			<Button variant="primary" size="large" onclick={onPrimaryClick}>
				{primaryAction}
			</Button>
			{#if secondaryAction}
				<Button variant="secondary" onclick={onSecondaryClick}>
					{secondaryAction}
				</Button>
			{/if}
		</div>
	</div>
</div>

<style>
	.completion-screen {
		display: flex;
		flex-direction: column;
		align-items: center;
		text-align: center;
		padding: 48px 24px;
		max-width: 500px;
		margin: 0 auto;
		animation: fade-in 0.5s ease-out;
	}

	.orb-container {
		margin-bottom: 32px;
	}

	.content {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 16px;
	}

	.title {
		font-size: 28px;
		font-weight: 600;
		color: var(--foreground, #1f2937);
		margin: 0;
	}

	.subtitle {
		font-size: 16px;
		color: var(--muted-foreground, #6b7280);
		margin: 0;
		line-height: 1.5;
	}

	.checklist {
		list-style: none;
		margin: 24px 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 12px;
		width: 100%;
		text-align: left;
	}

	.checklist-item {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 12px 16px;
		background-color: var(--secondary, #f9fafb);
		border-radius: 8px;
		transition: all 0.2s ease;
	}

	.checklist-item.is-completed {
		background-color: rgba(16, 185, 129, 0.1);
	}

	.check-circle {
		width: 24px;
		height: 24px;
		border-radius: 50%;
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
		border: 2px solid var(--border, #e5e7eb);
		color: white;
	}

	.is-completed .check-circle {
		background-color: var(--success, #10b981);
		border-color: var(--success, #10b981);
	}

	.item-label {
		font-size: 14px;
		color: var(--foreground, #1f2937);
	}

	.actions {
		display: flex;
		flex-direction: column;
		gap: 12px;
		width: 100%;
		margin-top: 16px;
	}

	@keyframes fade-in {
		from {
			opacity: 0;
			transform: translateY(20px);
		}
		to {
			opacity: 1;
			transform: translateY(0);
		}
	}

	/* Dark mode */
	:global(.dark) .title {
		color: var(--foreground, #f9fafb);
	}

	:global(.dark) .checklist-item {
		background-color: var(--secondary, #1a1a1a);
	}

	:global(.dark) .checklist-item.is-completed {
		background-color: rgba(16, 185, 129, 0.15);
	}

	:global(.dark) .check-circle {
		border-color: var(--border, #2a2a2a);
	}

	:global(.dark) .item-label {
		color: var(--foreground, #f9fafb);
	}
</style>
