<script lang="ts">
	import { tick } from 'svelte';
	import { X, Loader2 } from 'lucide-svelte';
	import ChannelMessage from './ChannelMessage.svelte';
	import ChannelCompose, {
		type ComposeAttachment,
	} from './ChannelCompose.svelte';
	import ChannelEmptyState from './ChannelEmptyState.svelte';
	import {
		isOwnMessage,
		senderDisplayName,
	} from './commsChannelsUtils';
	import { bindShortcuts } from '$lib/components/comms/commsKeyboard';
	import type { CommsChannel, CommsMessage } from '$lib/api/comms';

	interface Props {
		channel: CommsChannel | null;
		// The thread root + replies, sorted ascending. The root is messages[0]
		// (or the message whose external_id matches the thread_ts).
		messages: CommsMessage[];
		isLoading: boolean;
		error: string | null;
		selfUserId?: string;
		// Compose state for the reply input (orchestrator owns the value).
		replyValue: string;
		replyAttachments?: ComposeAttachment[];
		replyIsSending: boolean;
		onClose: () => void;
		onChangeReply: (value: string) => void;
		onSendReply: () => void;
		onToggleReaction?: (message: CommsMessage, emoji: string) => void;
		onPickEmoji?: (message: CommsMessage) => void;
		onRetry?: () => void;
	}

	let {
		channel,
		messages,
		isLoading,
		error,
		selfUserId,
		replyValue,
		replyAttachments = [],
		replyIsSending,
		onClose,
		onChangeReply,
		onSendReply,
		onToggleReaction,
		onPickEmoji,
		onRetry,
	}: Props = $props();

	const root = $derived(messages.find((m) => m.is_thread_root) ?? messages[0]);
	const replies = $derived(messages.filter((m) => m !== root));
	const rootContext = $derived(
		root
			? `Reply to ${senderDisplayName(root)}: ${root.content.slice(0, 60)}${root.content.length > 60 ? '…' : ''}`
			: 'Thread',
	);

	let scrollEl = $state<HTMLDivElement | null>(null);
	let drawerEl = $state<HTMLElement | null>(null);
	let lastReplyCount = $state(0);

	$effect(() => {
		if (!scrollEl) return;
		const grew = replies.length > lastReplyCount;
		lastReplyCount = replies.length;
		if (!grew) return;
		const el = scrollEl;
		tick().then(() => {
			el.scrollTop = el.scrollHeight;
		});
	});

	// Keyboard nav within the drawer. Mirrors Slack/Gmail conventions and reuses
	// Leah's shared shortcut helper at $lib/components/comms/commsKeyboard so
	// email and channels stay aligned.
	function findFocusableReplies(): HTMLElement[] {
		if (!drawerEl) return [];
		return Array.from(
			drawerEl.querySelectorAll<HTMLElement>('[data-cm-drawer-msg]'),
		);
	}

	function moveFocus(delta: number) {
		const items = findFocusableReplies();
		if (!items.length) return;
		const active = document.activeElement as HTMLElement | null;
		const currentIdx = active ? items.indexOf(active) : -1;
		const target =
			currentIdx === -1 ? (delta > 0 ? 0 : items.length - 1) : currentIdx + delta;
		const next = ((target % items.length) + items.length) % items.length;
		items[next].focus();
	}

	function focusReplyInput() {
		if (!drawerEl) return;
		drawerEl
			.querySelector<HTMLTextAreaElement>('.cm-channels-compose__textarea')
			?.focus();
	}

	$effect(() =>
		bindShortcuts([
			{
				key: 'Escape',
				allowInInput: true,
				handler: () => onClose(),
				description: 'Close thread',
			},
			{
				key: 'Mod+Enter',
				allowInInput: true,
				handler: () => onSendReply(),
				description: 'Send reply',
			},
			{ key: 'j', handler: () => moveFocus(1), description: 'Next reply' },
			{ key: 'k', handler: () => moveFocus(-1), description: 'Previous reply' },
			{ key: 'r', handler: () => focusReplyInput(), description: 'Reply' },
		]),
	);
</script>

<aside
	bind:this={drawerEl}
	class="cm-channels-drawer"
	aria-label="Thread"
	aria-labelledby="cm-channels-drawer-title"
>
	<header class="cm-channels-drawer__head">
		<div class="cm-channels-drawer__head-text">
			<h3 id="cm-channels-drawer-title" class="cm-channels-drawer__title">
				Thread
			</h3>
			<p class="cm-channels-drawer__context">{rootContext}</p>
		</div>
		<button
			type="button"
			class="btn-compact btn-compact-ghost btn-compact-icon"
			aria-label="Close thread"
			onclick={onClose}
		>
			<X size={16} />
		</button>
	</header>

	<div class="cm-channels-drawer__body" bind:this={scrollEl}>
		{#if error}
			<ChannelEmptyState
				variant="error"
				size="md"
				title="Couldn't load this thread"
				description={error}
				actionLabel={onRetry ? 'Try again' : undefined}
				onAction={onRetry}
				secondaryActionLabel="Close"
				onSecondaryAction={onClose}
			/>
		{:else if isLoading && messages.length === 0}
			<div class="cm-channels-drawer__skeleton" aria-hidden="true">
				<Loader2 size={20} class="cm-spin" />
				<span>Loading thread…</span>
			</div>
		{:else if !root}
			<ChannelEmptyState
				variant="no-messages"
				size="sm"
				title="Thread unavailable"
				description="The original message couldn't be found."
			/>
		{:else}
			<div
				class="cm-channels-drawer__root cm-channels-drawer__msg"
				data-cm-drawer-msg
				tabindex="-1"
			>
				<ChannelMessage
					message={root}
					showHeader
					isOwn={isOwnMessage(root, selfUserId)}
					{onToggleReaction}
					{onPickEmoji}
				/>
			</div>

			<div class="cm-channels-drawer__divider" role="separator">
				<span class="cm-channels-drawer__divider-label">
					{replies.length}
					{replies.length === 1 ? 'reply' : 'replies'}
				</span>
			</div>

			{#each replies as reply (reply.id)}
				<div
					class="cm-channels-drawer__msg"
					data-cm-drawer-msg
					tabindex="-1"
				>
					<ChannelMessage
						message={reply}
						showHeader
						isOwn={isOwnMessage(reply, selfUserId)}
						{onToggleReaction}
						{onPickEmoji}
					/>
				</div>
			{/each}
		{/if}
	</div>

	<div class="cm-channels-drawer__compose">
		<ChannelCompose
			{channel}
			value={replyValue}
			attachments={replyAttachments}
			isSending={replyIsSending}
			placeholder="Reply to thread…"
			onChange={onChangeReply}
			onSend={onSendReply}
		/>
	</div>
</aside>

<style>
	.cm-channels-drawer {
		display: flex;
		flex-direction: column;
		width: 380px;
		flex-shrink: 0;
		background: var(--dbg);
		border-left: 1px solid var(--dbd);
		min-height: 0;
		transition: width var(--bos-transition-slow);
	}

	.cm-channels-drawer__head {
		display: flex;
		align-items: flex-start;
		gap: var(--space-3);
		padding: var(--space-3) var(--space-4);
		border-bottom: 1px solid var(--dbd);
		flex-shrink: 0;
	}

	.cm-channels-drawer__head-text {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: 2px;
	}

	.cm-channels-drawer__title {
		font-size: var(--text-base);
		font-weight: var(--font-semibold);
		color: var(--dt);
		margin: 0;
	}

	.cm-channels-drawer__context {
		font-size: var(--text-xs);
		color: var(--dt3);
		margin: 0;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.cm-channels-drawer__body {
		flex: 1;
		min-height: 0;
		overflow-y: auto;
		display: flex;
		flex-direction: column;
		padding: var(--space-3) 0;
	}

	.cm-channels-drawer__root {
		padding-bottom: var(--space-2);
	}

	.cm-channels-drawer__msg {
		outline: none;
	}

	.cm-channels-drawer__msg:focus-visible {
		box-shadow: inset 2px 0 0 var(--bos-accent-blue);
		background: var(--bos-nav-active-bg);
	}

	.cm-channels-drawer__divider {
		display: flex;
		align-items: center;
		gap: var(--space-3);
		padding: var(--space-3) var(--space-5);
		font-size: var(--text-xs);
		font-weight: var(--font-semibold);
		color: var(--dt3);
	}

	.cm-channels-drawer__divider::before,
	.cm-channels-drawer__divider::after {
		content: '';
		flex: 1;
		height: 1px;
		background: var(--dbd);
	}

	.cm-channels-drawer__divider-label {
		flex-shrink: 0;
	}

	.cm-channels-drawer__skeleton {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: var(--space-2);
		padding: var(--space-8) var(--space-4);
		font-size: var(--text-sm);
		color: var(--dt3);
	}

	.cm-channels-drawer__compose {
		flex-shrink: 0;
		border-top: 1px solid var(--dbd);
	}

	@media (max-width: 1279px) {
		.cm-channels-drawer {
			position: absolute;
			inset: 0 0 0 auto;
			width: 100%;
			max-width: 480px;
			box-shadow: var(--bos-shadow-3);
			z-index: var(--bos-z-dropdown);
		}
	}

	@media (prefers-reduced-motion: reduce) {
		.cm-channels-drawer {
			transition: none;
		}
	}
</style>
