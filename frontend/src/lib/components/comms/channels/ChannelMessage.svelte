<script lang="ts">
	import { MessageSquare, Smile, Reply, ChevronRight } from 'lucide-svelte';
	import {
		formatMessageTime,
		formatFullTimestamp,
		initials,
		senderDisplayName,
	} from './commsChannelsUtils';
	import type { CommsMessage } from '$lib/api/comms';

	interface Props {
		message: CommsMessage;
		// Head messages render avatar + sender name + time. Tail messages render
		// only the body, indented to align with the body of the head above.
		showHeader: boolean;
		// Own messages (sent by the current user) right-align with a soft tint.
		isOwn?: boolean;
		// Reply summary surfaces under thread roots when reply_count > 0.
		onOpenThread?: (message: CommsMessage) => void;
		// Hover quick-reply (icon ↩).
		onReplyInThread?: (message: CommsMessage) => void;
		// Toggling a reaction. The handler decides add vs remove based on
		// reacted_by_me on the chip.
		onToggleReaction?: (message: CommsMessage, emoji: string) => void;
		// Click the "+" chip → emoji picker.
		onPickEmoji?: (message: CommsMessage) => void;
	}

	let {
		message,
		showHeader,
		isOwn = false,
		onOpenThread,
		onReplyInThread,
		onToggleReaction,
		onPickEmoji,
	}: Props = $props();

	const senderLabel = $derived(senderDisplayName(message));
	const senderInitial = $derived(initials(senderLabel));
	const fullTimestamp = $derived(formatFullTimestamp(message.sent_at));
	const compactTime = $derived(formatMessageTime(message.sent_at));

	const showReplySummary = $derived(
		message.reply_count > 0 && message.is_thread_root,
	);
	const showReactions = $derived(message.reactions.length > 0);
	const isDeleted = $derived(message.is_deleted);
</script>

<article
	class="cm-channels-msg"
	class:cm-channels-msg--head={showHeader}
	class:cm-channels-msg--tail={!showHeader}
	class:cm-channels-msg--own={isOwn}
>
	{#if showHeader}
		{#if message.sender_avatar}
			<img
				class="cm-channels-msg__avatar"
				src={message.sender_avatar}
				alt=""
			/>
		{:else}
			<span class="cm-channels-msg__avatar cm-channels-msg__avatar--initial" aria-hidden="true">
				{senderInitial}
			</span>
		{/if}
	{:else}
		<span class="cm-channels-msg__avatar-spacer" aria-hidden="true"></span>
	{/if}

	<div class="cm-channels-msg__body-wrap">
		{#if showHeader}
			<header class="cm-channels-msg__head">
				<span class="cm-channels-msg__sender">{senderLabel}</span>
				<time
					class="cm-channels-msg__time"
					datetime={message.sent_at}
					title={fullTimestamp}
				>
					{compactTime}
				</time>
				{#if message.is_edited}
					<span class="cm-channels-msg__edited">(edited)</span>
				{/if}
			</header>
		{/if}

		{#if isDeleted}
			<p class="cm-channels-msg__deleted">This message was deleted.</p>
		{:else}
			<p class="cm-channels-msg__text">{message.content}</p>
		{/if}

		{#if showReactions}
			<div class="cm-channels-msg__reactions" role="group" aria-label="Reactions">
				{#each message.reactions as reaction (reaction.emoji)}
					<button
						type="button"
						class="cm-channels-msg__reaction"
						class:cm-channels-msg__reaction--mine={reaction.reacted_by_me}
						onclick={() => onToggleReaction?.(message, reaction.emoji)}
						aria-label="{reaction.reacted_by_me ? 'Remove' : 'Add'} {reaction.emoji} reaction, {reaction.count} total"
					>
						<span class="cm-channels-msg__reaction-emoji" aria-hidden="true">
							{reaction.emoji}
						</span>
						<span class="cm-channels-msg__reaction-count">{reaction.count}</span>
					</button>
				{/each}
				{#if onPickEmoji}
					<button
						type="button"
						class="cm-channels-msg__reaction cm-channels-msg__reaction--add"
						onclick={() => onPickEmoji(message)}
						aria-label="Add reaction"
					>
						<Smile size={12} />
					</button>
				{/if}
			</div>
		{/if}

		{#if showReplySummary && onOpenThread}
			<button
				type="button"
				class="cm-channels-msg__reply-summary"
				onclick={() => onOpenThread(message)}
				aria-label="View {message.reply_count} {message.reply_count === 1 ? 'reply' : 'replies'} in thread"
			>
				<MessageSquare size={12} />
				<span>
					{message.reply_count}
					{message.reply_count === 1 ? 'reply' : 'replies'}
				</span>
				<ChevronRight size={12} />
			</button>
		{/if}
	</div>

	<div class="cm-channels-msg__hover-actions" aria-hidden="true">
		{#if onPickEmoji}
			<button
				type="button"
				class="cm-channels-msg__hover-btn"
				onclick={() => onPickEmoji(message)}
				aria-label="Add reaction"
				tabindex="-1"
			>
				<Smile size={14} />
			</button>
		{/if}
		{#if onReplyInThread}
			<button
				type="button"
				class="cm-channels-msg__hover-btn"
				onclick={() => onReplyInThread(message)}
				aria-label="Reply in thread"
				tabindex="-1"
			>
				<Reply size={14} />
			</button>
		{/if}
	</div>
</article>

<style>
	.cm-channels-msg {
		position: relative;
		display: flex;
		gap: var(--space-3);
		padding: var(--space-2) var(--space-5);
		transition: background var(--bos-transition-fast);
	}

	.cm-channels-msg--head {
		padding-top: var(--space-3);
	}

	.cm-channels-msg--tail {
		padding-top: 0;
		padding-bottom: 0;
	}

	.cm-channels-msg:hover {
		background: var(--dbg2);
	}

	.cm-channels-msg:hover .cm-channels-msg__time {
		color: var(--dt2);
	}

	.cm-channels-msg--own {
		flex-direction: row-reverse;
	}

	.cm-channels-msg--own .cm-channels-msg__body-wrap {
		max-width: min(680px, 70%);
		margin-left: auto;
		background: var(--bos-nav-active-bg);
		border-radius: var(--radius-md);
		padding: var(--space-2) var(--space-3);
	}

	.cm-channels-msg__avatar {
		width: 36px;
		height: 36px;
		border-radius: var(--radius-md);
		flex-shrink: 0;
		object-fit: cover;
	}

	.cm-channels-msg__avatar--initial {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		background: var(--bos-avatar-default);
		color: var(--bos-avatar-default-text);
		font-size: var(--text-xs);
		font-weight: var(--font-bold);
	}

	.cm-channels-msg__avatar-spacer {
		width: 36px;
		flex-shrink: 0;
	}

	.cm-channels-msg__body-wrap {
		flex: 1;
		min-width: 0;
		max-width: 680px;
	}

	.cm-channels-msg__head {
		display: flex;
		align-items: baseline;
		gap: var(--space-2);
	}

	.cm-channels-msg__sender {
		font-size: var(--text-sm);
		font-weight: var(--font-semibold);
		color: var(--dt);
	}

	.cm-channels-msg__time {
		font-size: var(--text-xs);
		color: var(--dt3);
	}

	.cm-channels-msg__edited {
		font-size: var(--text-xs);
		color: var(--dt4);
		font-weight: var(--font-normal);
	}

	.cm-channels-msg__text {
		font-size: var(--text-sm);
		color: var(--dt);
		line-height: 1.55;
		margin: 0;
		white-space: pre-wrap;
		word-wrap: break-word;
	}

	.cm-channels-msg__deleted {
		font-size: var(--text-sm);
		color: var(--dt4);
		font-style: italic;
		margin: 0;
	}

	.cm-channels-msg__reactions {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-1);
		margin-top: var(--space-2);
	}

	.cm-channels-msg__reaction {
		display: inline-flex;
		align-items: center;
		gap: var(--space-1);
		padding: 2px var(--space-2);
		background: var(--dbg2);
		color: var(--dt3);
		border: none;
		border-radius: var(--radius-full);
		font-family: inherit;
		font-size: var(--text-xs);
		cursor: pointer;
		transition: background var(--bos-transition-fast), color var(--bos-transition-fast);
	}

	.cm-channels-msg__reaction:hover {
		background: var(--dbg3);
	}

	.cm-channels-msg__reaction--mine {
		background: var(--bos-nav-active-bg);
		color: var(--bos-nav-active);
	}

	.cm-channels-msg__reaction--add {
		color: var(--dt3);
	}

	.cm-channels-msg__reaction-count {
		font-weight: var(--font-medium);
	}

	.cm-channels-msg__reply-summary {
		display: inline-flex;
		align-items: center;
		gap: var(--space-2);
		margin-top: var(--space-1);
		padding: var(--space-1) var(--space-3);
		background: var(--dbg2);
		color: var(--dt3);
		border: none;
		border-radius: var(--radius-sm);
		font-family: inherit;
		font-size: var(--text-xs);
		cursor: pointer;
		transition: background var(--bos-transition-fast), color var(--bos-transition-fast);
	}

	.cm-channels-msg__reply-summary:hover {
		background: var(--dbg3);
		color: var(--dt2);
	}

	.cm-channels-msg__hover-actions {
		position: absolute;
		top: -14px;
		right: var(--space-3);
		display: flex;
		gap: 0;
		padding: 2px var(--space-1);
		background: var(--dbg);
		border: 1px solid var(--dbd);
		border-radius: var(--radius-md);
		box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06), 0 4px 16px rgba(0, 0, 0, 0.04);
		opacity: 0;
		pointer-events: none;
		transform: translateY(2px);
		transition: opacity 150ms ease, transform 150ms ease;
		z-index: 2;
	}

	.cm-channels-msg:hover .cm-channels-msg__hover-actions {
		opacity: 1;
		pointer-events: auto;
		transform: translateY(0);
	}

	.cm-channels-msg--own .cm-channels-msg__hover-actions {
		right: auto;
		left: var(--space-3);
	}

	.cm-channels-msg__hover-btn {
		width: 24px;
		height: 24px;
		display: inline-flex;
		align-items: center;
		justify-content: center;
		background: none;
		border: none;
		color: var(--dt3);
		cursor: pointer;
		border-radius: var(--radius-sm);
		transition: background var(--bos-transition-fast), color var(--bos-transition-fast);
	}

	.cm-channels-msg__hover-btn:hover {
		background: var(--dbg3);
		color: var(--dt);
	}

	@media (prefers-reduced-motion: reduce) {
		.cm-channels-msg,
		.cm-channels-msg__reaction,
		.cm-channels-msg__reply-summary,
		.cm-channels-msg__hover-actions,
		.cm-channels-msg__hover-btn {
			transition: none;
		}
	}
</style>
