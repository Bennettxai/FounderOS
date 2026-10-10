<script lang="ts">
	import { tick } from 'svelte';
	import ChannelMessage from './ChannelMessage.svelte';
	import ChannelEmptyState from './ChannelEmptyState.svelte';
	import { buildStreamItems, isOwnMessage } from './commsChannelsUtils';
	import type { CommsChannel, CommsMessage } from '$lib/api/comms';

	interface Props {
		channel: CommsChannel | null;
		messages: CommsMessage[];
		isLoading: boolean;
		error: string | null;
		// Self user id is used to right-align own messages. Optional — when absent,
		// no message gets the "own" treatment.
		selfUserId?: string;
		onOpenThread?: (message: CommsMessage) => void;
		onReplyInThread?: (message: CommsMessage) => void;
		onToggleReaction?: (message: CommsMessage, emoji: string) => void;
		onPickEmoji?: (message: CommsMessage) => void;
		onRetry?: () => void;
		// Fires when the scroll position crosses the top boundary, so the
		// channel toolbar can fade in its elevation shadow.
		onScrolledChange?: (scrolled: boolean) => void;
	}

	let {
		channel,
		messages,
		isLoading,
		error,
		selfUserId,
		onOpenThread,
		onReplyInThread,
		onToggleReaction,
		onPickEmoji,
		onRetry,
		onScrolledChange,
	}: Props = $props();

	const streamItems = $derived(buildStreamItems(messages));

	let scrollEl = $state<HTMLDivElement | null>(null);
	let lastMessageCount = $state(0);
	let isScrolled = $state(false);

	function handleScroll() {
		if (!scrollEl) return;
		const scrolled = scrollEl.scrollTop > 4;
		if (scrolled !== isScrolled) {
			isScrolled = scrolled;
			onScrolledChange?.(scrolled);
		}
	}

	// Pin scroll to the bottom on initial mount and when new messages arrive
	// at the tail. If the user has scrolled up, don't yank them down.
	$effect(() => {
		const count = messages.length;
		if (!scrollEl) return;
		const grew = count > lastMessageCount;
		lastMessageCount = count;
		if (!grew) return;

		const el = scrollEl;
		const distanceFromBottom =
			el.scrollHeight - el.scrollTop - el.clientHeight;
		const nearBottom = distanceFromBottom < 120;

		if (nearBottom || count === messages.length) {
			tick().then(() => {
				el.scrollTop = el.scrollHeight;
			});
		}
	});

	// Per-channel scroll reset: when the channel changes, jump to the bottom
	// (most channels render the latest activity).
	$effect(() => {
		const id = channel?.id;
		if (!id || !scrollEl) return;
		const el = scrollEl;
		tick().then(() => {
			el.scrollTop = el.scrollHeight;
		});
	});

	function emptyTitle(c: CommsChannel | null): string {
		if (!c) return 'Select a channel to view messages';
		if (c.is_dm) return 'No messages yet';
		return `No messages in #${c.name}`;
	}

	function emptyDescription(c: CommsChannel | null): string {
		if (!c) return 'Pick a channel or DM from the sidebar to get started.';
		if (c.is_dm) return 'Send a message to start the conversation.';
		return c.is_private
			? 'Start the conversation when you’re ready.'
			: 'Be the first to say hello.';
	}
</script>

<div class="cm-channels-messages" bind:this={scrollEl} onscroll={handleScroll}>
	{#if !channel}
		<ChannelEmptyState
			variant="no-channel-selected"
			size="lg"
			title={emptyTitle(null)}
			description={emptyDescription(null)}
		/>
	{:else if error}
		<ChannelEmptyState
			variant="error"
			size="md"
			title="Couldn't load this channel"
			description={error}
			actionLabel={onRetry ? 'Try again' : undefined}
			onAction={onRetry}
		/>
	{:else if isLoading && messages.length === 0}
		<div class="cm-channels-messages__skeleton" aria-hidden="true">
			{#each Array(5) as _, i (i)}
				<div class="cm-channels-messages__skeleton-group">
					<div class="cm-channels-messages__skeleton-avatar"></div>
					<div class="cm-channels-messages__skeleton-body">
						<div class="cm-channels-messages__skeleton-line cm-channels-messages__skeleton-line--name"></div>
						<div class="cm-channels-messages__skeleton-line cm-channels-messages__skeleton-line--long"></div>
						<div class="cm-channels-messages__skeleton-line cm-channels-messages__skeleton-line--mid"></div>
					</div>
				</div>
			{/each}
		</div>
	{:else if messages.length === 0}
		<ChannelEmptyState
			variant="no-messages"
			size="lg"
			title={emptyTitle(channel)}
			description={emptyDescription(channel)}
		/>
	{:else}
		<div class="cm-channels-messages__stream">
			{#each streamItems as item (item.kind === 'divider' ? item.id : item.group.groupId)}
				{#if item.kind === 'divider'}
					<div class="cm-channels-messages__divider" role="separator">
						<span class="cm-channels-messages__divider-label">{item.label}</span>
					</div>
				{:else}
					<div class="cm-channels-messages__group">
						{#each item.group.messages as message, idx (message.id)}
							<ChannelMessage
								{message}
								showHeader={idx === 0}
								isOwn={isOwnMessage(message, selfUserId)}
								{onOpenThread}
								{onReplyInThread}
								{onToggleReaction}
								{onPickEmoji}
							/>
						{/each}
					</div>
				{/if}
			{/each}
		</div>
	{/if}
</div>

<style>
	.cm-channels-messages {
		flex: 1;
		min-height: 0;
		overflow-y: auto;
		display: flex;
		flex-direction: column;
		background: var(--dbg);
	}

	.cm-channels-messages__stream {
		display: flex;
		flex-direction: column;
		padding: var(--space-3) 0 var(--space-4);
	}

	.cm-channels-messages__group {
		display: flex;
		flex-direction: column;
	}

	.cm-channels-messages__divider {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: var(--space-3);
		padding: var(--space-4) var(--space-5) var(--space-2);
		font-size: var(--text-xs);
		font-weight: var(--font-semibold);
		color: var(--dt3);
		text-transform: none;
		position: sticky;
		top: 0;
		z-index: 1;
		pointer-events: none;
	}

	.cm-channels-messages__divider::before,
	.cm-channels-messages__divider::after {
		content: '';
		flex: 1;
		height: 1px;
		background: linear-gradient(to right, transparent, var(--dbd2), transparent);
	}

	.cm-channels-messages__divider-label {
		flex-shrink: 0;
		padding: 3px 12px;
		background: var(--dbg2);
		border: 1px solid var(--dbd2);
		border-radius: var(--radius-full);
		color: var(--dt2);
		font-size: 0.7rem;
		letter-spacing: 0.02em;
	}

	.cm-channels-messages__skeleton {
		display: flex;
		flex-direction: column;
		gap: var(--space-4);
		padding: var(--space-5);
	}

	.cm-channels-messages__skeleton-group {
		display: flex;
		gap: var(--space-3);
	}

	.cm-channels-messages__skeleton-avatar {
		width: 36px;
		height: 36px;
		border-radius: var(--radius-md);
		background: var(--dbg2);
		flex-shrink: 0;
		animation: cm-channels-skeleton-pulse 1.6s ease-in-out infinite;
	}

	.cm-channels-messages__skeleton-body {
		flex: 1;
		display: flex;
		flex-direction: column;
		gap: var(--space-2);
	}

	.cm-channels-messages__skeleton-line {
		height: 10px;
		border-radius: var(--radius-sm);
		background: var(--dbg2);
		animation: cm-channels-skeleton-pulse 1.6s ease-in-out infinite;
	}

	.cm-channels-messages__skeleton-line--name {
		width: 30%;
		height: 12px;
	}

	.cm-channels-messages__skeleton-line--long {
		width: 90%;
	}

	.cm-channels-messages__skeleton-line--mid {
		width: 60%;
	}

	@keyframes cm-channels-skeleton-pulse {
		0%, 100% {
			opacity: 0.45;
		}
		50% {
			opacity: 0.85;
		}
	}

	@media (prefers-reduced-motion: reduce) {
		.cm-channels-messages__skeleton-avatar,
		.cm-channels-messages__skeleton-line {
			animation: none;
		}
	}
</style>
