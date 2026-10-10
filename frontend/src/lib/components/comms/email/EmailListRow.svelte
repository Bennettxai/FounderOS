<script lang="ts">
	import { Paperclip, Star } from 'lucide-svelte';
	import {
		displayName,
		formatRowDate,
		initials,
		normalizeSubject,
	} from './commsEmailUtils';
	import type { EmailThread } from '$lib/api/comms';

	interface Props {
		thread: EmailThread;
		isSelected: boolean;
		showProviderBadge: boolean;
		onSelect: (thread: EmailThread) => void;
		onToggleStar?: (thread: EmailThread) => void;
	}

	let {
		thread,
		isSelected,
		showProviderBadge,
		onSelect,
		onToggleStar,
	}: Props = $props();

	const latest = $derived(thread.messages[0]);
	const senderLabel = $derived(displayName(latest));
	const avatarInitials = $derived(initials(senderLabel));
	const subjectInfo = $derived(normalizeSubject(thread.subject));
	const time = $derived(formatRowDate(thread.latest_date));
	const hasReplies = $derived(thread.message_count > 1);

	function handleStar(event: MouseEvent) {
		event.stopPropagation();
		onToggleStar?.(thread);
	}

	function handleKeydown(event: KeyboardEvent) {
		if (event.key === 'Enter' || event.key === ' ') {
			event.preventDefault();
			onSelect(thread);
		}
	}
</script>

<div
	class="cm-email-row"
	class:cm-email-row--unread={thread.unread}
	class:cm-email-row--selected={isSelected}
	role="button"
	tabindex="0"
	onclick={() => onSelect(thread)}
	onkeydown={handleKeydown}
	aria-label="{subjectInfo.subject} from {senderLabel}, {thread.message_count} message{thread.message_count === 1 ? '' : 's'}"
>
	<span class="cm-email-row__indicators" aria-hidden="true">
		{#if showProviderBadge}
			<span
				class="cm-email-row__provider-dot cm-email-row__provider-dot--{thread.provider}"
			></span>
		{/if}
		<span class="cm-email-row__unread-dot" class:cm-email-row__unread-dot--read={!thread.unread}></span>
	</span>

	<span class="cm-email-row__avatar" aria-hidden="true">{avatarInitials}</span>

	<span class="cm-email-row__body">
		<span class="cm-email-row__line cm-email-row__line--top">
			<span class="cm-email-row__sender">{senderLabel}</span>
			{#if hasReplies}
				<span class="cm-email-row__count">· {thread.message_count}</span>
			{/if}
			<span class="cm-email-row__time">{time}</span>
		</span>
		<span class="cm-email-row__line cm-email-row__line--bottom">
			<span class="cm-email-row__subject">
				{#if subjectInfo.prefix}<span class="cm-email-row__prefix">{subjectInfo.prefix}</span>{/if}
				{subjectInfo.subject}
			</span>
			<span class="cm-email-row__icons" aria-hidden="true">
				{#if thread.has_attachments}
					<Paperclip size={12} />
				{/if}
			</span>
		</span>
		<span class="cm-email-row__snippet">{thread.snippet}</span>
	</span>

	<span class="cm-email-row__star">
		<button
			type="button"
			class="cm-email-row__star-button"
			class:cm-email-row__star-button--active={thread.starred}
			onclick={handleStar}
			aria-label={thread.starred ? 'Unstar thread' : 'Star thread'}
		>
			<Star size={14} fill={thread.starred ? 'currentColor' : 'none'} />
		</button>
	</span>
</div>

<style>
	.cm-email-row {
		display: grid;
		grid-template-columns: 16px 32px 1fr auto;
		align-items: flex-start;
		gap: var(--space-3);
		width: 100%;
		padding: var(--space-3) var(--space-4);
		background: var(--dbg);
		border-bottom: 1px solid var(--dbd);
		font-family: inherit;
		text-align: left;
		cursor: pointer;
		transition: background var(--bos-transition-fast);
	}

	.cm-email-row:hover {
		background: var(--dbg2);
	}

	.cm-email-row:focus-visible {
		outline: 2px solid var(--bos-accent-blue);
		outline-offset: -2px;
	}

	.cm-email-row--selected {
		background: var(--dbg3);
		box-shadow: inset 3px 0 0 var(--bos-accent-blue);
	}

	.cm-email-row__indicators {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: var(--space-1);
		margin-top: var(--space-2);
		flex-shrink: 0;
	}

	.cm-email-row__provider-dot {
		width: 8px;
		height: 8px;
		border-radius: var(--radius-full);
	}

	.cm-email-row__provider-dot--gmail {
		background: var(--bos-status-error);
	}

	.cm-email-row__provider-dot--outlook {
		background: var(--bos-status-info);
	}

	.cm-email-row__unread-dot {
		width: 7px;
		height: 7px;
		border-radius: var(--radius-full);
		background: var(--bos-accent-blue);
	}

	.cm-email-row__unread-dot--read {
		background: transparent;
		border: 1px solid var(--dt4);
	}

	.cm-email-row__avatar {
		width: 32px;
		height: 32px;
		border-radius: var(--radius-full);
		background: var(--dt3);
		color: var(--dbg);
		font-size: var(--text-xs);
		font-weight: var(--font-bold);
		display: inline-flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
	}

	.cm-email-row__body {
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: 2px;
	}

	.cm-email-row__line {
		display: flex;
		align-items: baseline;
		gap: var(--space-2);
		min-width: 0;
	}

	.cm-email-row__sender {
		font-size: var(--text-sm);
		color: var(--dt2);
		font-weight: var(--font-normal);
		flex-shrink: 1;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.cm-email-row--unread .cm-email-row__sender {
		color: var(--dt);
		font-weight: var(--font-semibold);
	}

	.cm-email-row__count {
		font-size: var(--text-xs);
		color: var(--dt3);
		flex-shrink: 0;
	}

	.cm-email-row__time {
		margin-left: auto;
		font-size: var(--text-xs);
		color: var(--dt3);
		flex-shrink: 0;
	}

	.cm-email-row__subject {
		flex: 1;
		font-size: var(--text-sm);
		color: var(--dt2);
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.cm-email-row--unread .cm-email-row__subject {
		color: var(--dt);
		font-weight: var(--font-semibold);
	}

	.cm-email-row__prefix {
		color: var(--dt3);
		margin-right: var(--space-1);
	}

	.cm-email-row__icons {
		display: inline-flex;
		gap: var(--space-1);
		color: var(--dt3);
		flex-shrink: 0;
	}

	.cm-email-row__snippet {
		font-size: var(--text-xs);
		color: var(--dt3);
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.cm-email-row__star {
		display: inline-flex;
		align-items: flex-start;
		margin-top: 6px;
	}

	.cm-email-row__star-button {
		background: none;
		border: none;
		padding: 2px;
		cursor: pointer;
		color: var(--dt4);
		opacity: 0;
		transition: color var(--bos-transition-fast), opacity 150ms ease;
	}

	.cm-email-row:hover .cm-email-row__star-button,
	.cm-email-row__star-button:focus-visible,
	.cm-email-row__star-button--active {
		opacity: 1;
	}

	.cm-email-row__star-button:hover {
		color: var(--dt2);
	}

	.cm-email-row__star-button--active {
		color: var(--bos-status-warning);
	}

	@media (prefers-reduced-motion: reduce) {
		.cm-email-row,
		.cm-email-row__star-button {
			transition: none;
		}
	}
</style>
