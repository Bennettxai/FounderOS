<script lang="ts">
	import {
		Reply,
		ReplyAll,
		Forward,
		Star,
		Archive,
		Trash2,
		MoreHorizontal,
	} from 'lucide-svelte';
	import PillButton from '$lib/components/ui/PillButton.svelte';
	import EmailMessageCard from './EmailMessageCard.svelte';
	import EmailEmptyState from './EmailEmptyState.svelte';
	import { providerLabel } from './commsEmailUtils';
	import type { EmailAttachment, EmailThread, UnifiedEmail } from '$lib/api/comms';

	interface Props {
		thread: EmailThread | null;
		isLoading: boolean;
		errorMessage?: string | null;
		folderLabel: string;
		onReply: (thread: EmailThread) => void;
		onReplyAll: (thread: EmailThread) => void;
		onForward: (thread: EmailThread) => void;
		onArchive: (thread: EmailThread) => void;
		onDelete: (thread: EmailThread) => void;
		onToggleStar: (thread: EmailThread) => void;
		onMore?: (thread: EmailThread) => void;
		onCompose?: () => void;
		onRetry?: () => void;
		onDownloadAttachment?: (
			message: UnifiedEmail,
			attachment: EmailAttachment,
		) => void;
	}

	let {
		thread,
		isLoading,
		errorMessage = null,
		folderLabel,
		onReply,
		onReplyAll,
		onForward,
		onArchive,
		onDelete,
		onToggleStar,
		onMore,
		onCompose,
		onRetry,
		onDownloadAttachment,
	}: Props = $props();

	let expandedIds = $state<Set<string>>(new Set());

	$effect(() => {
		if (thread) {
			const next = new Set<string>();
			if (thread.messages.length > 0) next.add(thread.messages[0].id);
			expandedIds = next;
		} else {
			expandedIds = new Set();
		}
	});

	function toggleMessage(message: UnifiedEmail) {
		const next = new Set(expandedIds);
		if (next.has(message.id)) next.delete(message.id);
		else next.add(message.id);
		expandedIds = next;
	}
</script>

<section class="cm-email-thread" aria-label="Email preview">
	{#if errorMessage}
		<EmailEmptyState
			variant="error"
			title="Couldn't load this email"
			description={errorMessage}
			actionLabel={onRetry ? 'Try again' : undefined}
			onAction={onRetry}
		/>
	{:else if isLoading}
		<div class="cm-email-thread__skeleton" aria-busy="true" aria-label="Loading email">
			<header class="cm-email-thread__skeleton-header">
				<span class="cm-email-thread__skel-bar cm-email-thread__skel-bar--subject"></span>
				<span class="cm-email-thread__skel-bar cm-email-thread__skel-bar--meta"></span>
			</header>
			{#each [0, 1, 2] as i (i)}
				<div class="cm-email-thread__skeleton-message">
					<span class="cm-email-thread__skel-avatar"></span>
					<div class="cm-email-thread__skeleton-message-body">
						<span class="cm-email-thread__skel-bar cm-email-thread__skel-bar--sender"></span>
						<span class="cm-email-thread__skel-bar cm-email-thread__skel-bar--line"></span>
						<span class="cm-email-thread__skel-bar cm-email-thread__skel-bar--line cm-email-thread__skel-bar--line-mid"></span>
						<span class="cm-email-thread__skel-bar cm-email-thread__skel-bar--line cm-email-thread__skel-bar--line-short"></span>
					</div>
				</div>
			{/each}
		</div>
	{:else if !thread}
		<EmailEmptyState
			variant="no-selection"
			size="lg"
			title="Select an email to read"
			description="Or compose a new message."
			actionLabel={onCompose ? 'Compose' : undefined}
			onAction={onCompose}
		/>
	{:else}
		<header class="cm-email-thread__header">
			<h2 class="cm-email-thread__subject">{thread.subject}</h2>
			<p class="cm-email-thread__meta">
				{folderLabel} · {providerLabel(thread.provider)} · {thread.message_count} message{thread.message_count === 1 ? '' : 's'}
			</p>
		</header>

		<div class="cm-email-thread__messages">
			{#each thread.messages as message (message.id)}
				<EmailMessageCard
					{message}
					expanded={expandedIds.has(message.id)}
					onToggle={toggleMessage}
					{onDownloadAttachment}
				/>
			{/each}
		</div>

		<footer class="cm-email-thread__actions">
			<div class="cm-email-thread__actions-primary">
				<PillButton variant="soft" size="sm" onclick={() => onReply(thread!)}>
					<Reply size={14} />
					<span style="margin-left: var(--space-2);">Reply</span>
				</PillButton>
				<PillButton variant="soft" size="sm" onclick={() => onReplyAll(thread!)}>
					<ReplyAll size={14} />
					<span style="margin-left: var(--space-2);">Reply all</span>
				</PillButton>
				<PillButton variant="soft" size="sm" onclick={() => onForward(thread!)}>
					<Forward size={14} />
					<span style="margin-left: var(--space-2);">Forward</span>
				</PillButton>
			</div>
			<div class="cm-email-thread__actions-secondary">
				<button
					type="button"
					class="btn-compact btn-compact-ghost btn-compact-icon"
					class:cm-email-thread__star--active={thread.starred}
					onclick={() => onToggleStar(thread!)}
					aria-label={thread.starred ? 'Unstar thread' : 'Star thread'}
				>
					<Star size={16} fill={thread.starred ? 'currentColor' : 'none'} />
				</button>
				<button
					type="button"
					class="btn-compact btn-compact-ghost btn-compact-icon"
					onclick={() => onArchive(thread!)}
					aria-label="Archive thread"
				>
					<Archive size={16} />
				</button>
				<button
					type="button"
					class="btn-compact btn-compact-ghost btn-compact-icon"
					onclick={() => onDelete(thread!)}
					aria-label="Delete thread"
				>
					<Trash2 size={16} />
				</button>
				{#if onMore}
					<button
						type="button"
						class="btn-compact btn-compact-ghost btn-compact-icon"
						onclick={() => onMore?.(thread!)}
						aria-label="More options"
					>
						<MoreHorizontal size={16} />
					</button>
				{/if}
			</div>
		</footer>
	{/if}
</section>

<style>
	.cm-email-thread {
		flex: 1;
		display: flex;
		flex-direction: column;
		min-height: 0;
		background: var(--dbg);
	}

	.cm-email-thread__header {
		padding: var(--space-5) var(--space-6) var(--space-3);
		border-bottom: 1px solid var(--dbd);
	}

	.cm-email-thread__subject {
		margin: 0;
		font-size: var(--text-lg);
		font-weight: var(--font-semibold);
		color: var(--dt);
	}

	.cm-email-thread__meta {
		margin: var(--space-1) 0 0;
		font-size: var(--text-xs);
		color: var(--dt3);
	}

	.cm-email-thread__messages {
		flex: 1;
		min-height: 0;
		overflow-y: auto;
	}

	.cm-email-thread__actions {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		padding: var(--space-3) var(--space-6);
		background: var(--dbg2);
		border-top: 1px solid var(--dbd);
		flex-shrink: 0;
	}

	.cm-email-thread__actions-primary {
		display: flex;
		gap: var(--space-2);
	}

	.cm-email-thread__actions-secondary {
		display: flex;
		gap: var(--space-1);
		margin-left: auto;
	}

	.cm-email-thread__star--active {
		color: var(--bos-status-warning);
	}

	/* ── Skeleton (loading) ── */
	.cm-email-thread__skeleton {
		flex: 1;
		min-height: 0;
		overflow: hidden;
	}

	.cm-email-thread__skeleton-header {
		display: flex;
		flex-direction: column;
		gap: var(--space-2);
		padding: var(--space-5) var(--space-6) var(--space-3);
		border-bottom: 1px solid var(--dbd);
	}

	.cm-email-thread__skeleton-message {
		display: flex;
		gap: var(--space-3);
		padding: var(--space-5) var(--space-6);
		border-bottom: 1px solid var(--dbd);
	}

	.cm-email-thread__skeleton-message-body {
		flex: 1;
		display: flex;
		flex-direction: column;
		gap: var(--space-2);
	}

	.cm-email-thread__skel-avatar {
		width: 36px;
		height: 36px;
		border-radius: var(--radius-full);
		flex-shrink: 0;
		background: linear-gradient(
			90deg,
			var(--dbg2) 0%,
			var(--dbg3) 50%,
			var(--dbg2) 100%
		);
		background-size: 200% 100%;
		animation: cm-email-shimmer 1.6s ease-in-out infinite;
	}

	.cm-email-thread__skel-bar {
		display: block;
		height: 12px;
		border-radius: var(--radius-xs);
		background: linear-gradient(
			90deg,
			var(--dbg2) 0%,
			var(--dbg3) 50%,
			var(--dbg2) 100%
		);
		background-size: 200% 100%;
		animation: cm-email-shimmer 1.6s ease-in-out infinite;
	}

	.cm-email-thread__skel-bar--subject {
		height: 18px;
		width: 60%;
	}

	.cm-email-thread__skel-bar--meta {
		height: 10px;
		width: 35%;
	}

	.cm-email-thread__skel-bar--sender {
		height: 14px;
		width: 30%;
	}

	.cm-email-thread__skel-bar--line {
		width: 100%;
	}

	.cm-email-thread__skel-bar--line-mid { width: 92%; }
	.cm-email-thread__skel-bar--line-short { width: 65%; }

	@keyframes cm-email-shimmer {
		0% { background-position: 200% 0; }
		100% { background-position: -200% 0; }
	}

	@media (prefers-reduced-motion: reduce) {
		.cm-email-thread__skel-avatar,
		.cm-email-thread__skel-bar {
			animation: none;
		}
	}
</style>
