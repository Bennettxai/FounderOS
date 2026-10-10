<!--
  MessageBubble.svelte
  Chat bubble for agent (left) and user (right) messages
-->
<script lang="ts">
	import { type Snippet } from 'svelte';
	import TypingIndicator from './TypingIndicator.svelte';

	type Sender = 'agent' | 'user';

	interface Props {
		sender: Sender;
		isTyping?: boolean;
		showAvatar?: boolean;
		children?: Snippet;
		class?: string;
	}

	let {
		sender,
		isTyping = false,
		showAvatar = true,
		children,
		class: className = ''
	}: Props = $props();

	const isAgent = $derived(sender === 'agent');
</script>

<div
	class="message-bubble {className}"
	class:is-agent={isAgent}
	class:is-user={!isAgent}
>
	{#if showAvatar && isAgent}
		<div class="avatar">
			<div class="avatar-inner">
				<span>AI</span>
			</div>
		</div>
	{/if}

	<div class="bubble">
		{#if isTyping}
			<TypingIndicator />
		{:else if children}
			{@render children()}
		{/if}
	</div>

	{#if showAvatar && !isAgent}
		<div class="avatar user-avatar">
			<div class="avatar-inner">
				<span>U</span>
			</div>
		</div>
	{/if}
</div>

<style>
	.message-bubble {
		display: flex;
		align-items: flex-start;
		gap: 12px;
		max-width: 85%;
		animation: slide-up 0.3s ease-out;
	}

	.message-bubble.is-agent {
		align-self: flex-start;
	}

	.message-bubble.is-user {
		align-self: flex-end;
		flex-direction: row-reverse;
	}

	.avatar {
		flex-shrink: 0;
		width: 36px;
		height: 36px;
	}

	.avatar-inner {
		width: 100%;
		height: 100%;
		border-radius: 50%;
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 12px;
		font-weight: 600;
	}

	.is-agent .avatar-inner {
		background: linear-gradient(135deg, #6366f1, #4f46e5);
		color: white;
	}

	.user-avatar .avatar-inner {
		background-color: var(--secondary, #f9fafb);
		color: var(--foreground, #1f2937);
		border: 1px solid var(--border, #e5e7eb);
	}

	.bubble {
		padding: 12px 16px;
		border-radius: 16px;
		line-height: 1.5;
		font-size: 15px;
	}

	.is-agent .bubble {
		background-color: var(--secondary, #f9fafb);
		color: var(--foreground, #1f2937);
		border-bottom-left-radius: 4px;
	}

	.is-user .bubble {
		background-color: var(--primary, #000000);
		color: var(--primary-foreground, #ffffff);
		border-bottom-right-radius: 4px;
	}

	@keyframes slide-up {
		from {
			opacity: 0;
			transform: translateY(10px);
		}
		to {
			opacity: 1;
			transform: translateY(0);
		}
	}

	/* Dark mode */
	:global(.dark) .is-agent .bubble {
		background-color: var(--secondary, #1a1a1a);
		color: var(--foreground, #f9fafb);
	}

	:global(.dark) .is-user .bubble {
		background-color: var(--primary, #ffffff);
		color: var(--primary-foreground, #000000);
	}

	:global(.dark) .user-avatar .avatar-inner {
		background-color: var(--secondary, #1a1a1a);
		color: var(--foreground, #f9fafb);
		border-color: var(--border, #2a2a2a);
	}
</style>
