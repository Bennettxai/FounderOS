<script lang="ts">
	import DOMPurify, { type Config as DOMPurifyConfig } from 'dompurify';
	import { ChevronDown, ChevronRight } from 'lucide-svelte';
	import EmailAttachmentList from './EmailAttachmentList.svelte';
	import {
		displayName,
		formatPreviewDate,
		formatRowDate,
		initials,
		summarizeRecipients,
	} from './commsEmailUtils';
	import type { EmailAttachment, UnifiedEmail } from '$lib/api/comms';

	interface Props {
		message: UnifiedEmail;
		expanded: boolean;
		onToggle: (message: UnifiedEmail) => void;
		onDownloadAttachment?: (
			message: UnifiedEmail,
			attachment: EmailAttachment,
		) => void;
	}

	let { message, expanded, onToggle, onDownloadAttachment }: Props = $props();

	const senderName = $derived(displayName(message));
	const senderInitials = $derived(initials(senderName));
	const recipientLine = $derived(
		`to ${summarizeRecipients(message.to_emails, message.cc_emails)}`,
	);

	// DOMPurify allowlist preserved from the original implementation at
	// email/+page.svelte:399 — do not widen.
	const SANITIZE_CONFIG: DOMPurifyConfig = {
		ALLOWED_TAGS: [
			'p', 'br', 'b', 'i', 'u', 'strong', 'em', 'a', 'ul', 'ol', 'li',
			'h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'blockquote', 'pre', 'code',
			'span', 'div', 'table', 'thead', 'tbody', 'tr', 'td', 'th', 'img',
		],
		ALLOWED_ATTR: ['href', 'src', 'alt', 'class', 'style', 'target', 'rel'],
		ALLOW_DATA_ATTR: false,
	};

	const safeHtml = $derived(
		message.body_html ? DOMPurify.sanitize(message.body_html, SANITIZE_CONFIG) : '',
	);

	function handleDownload(attachment: EmailAttachment) {
		onDownloadAttachment?.(message, attachment);
	}
</script>

{#if expanded}
	<article class="cm-email-message cm-email-message--expanded">
		<header class="cm-email-message__head">
			<button
				type="button"
				class="cm-email-message__toggle"
				onclick={() => onToggle(message)}
				aria-label="Collapse message from {senderName}"
				aria-expanded="true"
			>
				<ChevronDown size={14} />
			</button>
			<span class="cm-email-message__avatar" aria-hidden="true">{senderInitials}</span>
			<div class="cm-email-message__head-text">
				<div class="cm-email-message__sender-row">
					<span class="cm-email-message__sender-name">{senderName}</span>
					{#if message.from_name && message.from_email !== senderName}
						<span class="cm-email-message__sender-email">&lt;{message.from_email}&gt;</span>
					{/if}
					<time class="cm-email-message__date" datetime={message.date}>
						{formatPreviewDate(message.date)}
					</time>
				</div>
				<div class="cm-email-message__recipients">{recipientLine}</div>
			</div>
		</header>

		<div class="cm-email-message__body">
			{#if message.body_html}
				<div class="cm-email-message__html">{@html safeHtml}</div>
			{:else if message.body_text}
				<pre class="cm-email-message__text">{message.body_text}</pre>
			{:else}
				<p class="cm-email-message__empty">No content</p>
			{/if}

			{#if message.attachments && message.attachments.length > 0}
				<EmailAttachmentList
					attachments={message.attachments}
					onDownload={onDownloadAttachment ? handleDownload : undefined}
				/>
			{/if}
		</div>
	</article>
{:else}
	<button
		type="button"
		class="cm-email-message cm-email-message--collapsed"
		onclick={() => onToggle(message)}
		aria-label="Expand message from {senderName}"
		aria-expanded="false"
	>
		<ChevronRight size={14} />
		<span class="cm-email-message__sender-name">{senderName}</span>
		<span class="cm-email-message__snippet">· {message.snippet}</span>
		<time class="cm-email-message__date" datetime={message.date}>
			{formatRowDate(message.date)}
		</time>
	</button>
{/if}

<style>
	.cm-email-message {
		display: block;
		width: 100%;
		text-align: left;
		font-family: inherit;
	}

	.cm-email-message--expanded {
		padding: var(--space-5) var(--space-6);
		border-bottom: 1px solid var(--dbd);
	}

	.cm-email-message--collapsed {
		display: flex;
		align-items: baseline;
		gap: var(--space-2);
		padding: var(--space-3) var(--space-6);
		background: var(--dbg);
		border: none;
		border-bottom: 1px solid var(--dbd);
		font-size: var(--text-sm);
		cursor: pointer;
		transition: background var(--bos-transition-fast);
	}

	.cm-email-message--collapsed:hover {
		background: var(--dbg2);
	}

	.cm-email-message__head {
		display: flex;
		gap: var(--space-3);
		align-items: flex-start;
	}

	.cm-email-message__toggle {
		background: none;
		border: none;
		padding: var(--space-1);
		color: var(--dt3);
		cursor: pointer;
		flex-shrink: 0;
	}

	.cm-email-message__toggle:hover {
		color: var(--dt);
	}

	.cm-email-message__avatar {
		width: 36px;
		height: 36px;
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

	.cm-email-message__head-text {
		flex: 1;
		min-width: 0;
	}

	.cm-email-message__sender-row {
		display: flex;
		flex-wrap: wrap;
		align-items: baseline;
		gap: var(--space-2);
	}

	.cm-email-message__sender-name {
		font-size: var(--text-sm);
		font-weight: var(--font-semibold);
		color: var(--dt);
	}

	.cm-email-message__sender-email {
		font-size: var(--text-xs);
		color: var(--dt3);
	}

	.cm-email-message__date {
		margin-left: auto;
		font-size: var(--text-xs);
		color: var(--dt3);
	}

	.cm-email-message__recipients {
		margin-top: 2px;
		font-size: var(--text-xs);
		color: var(--dt3);
	}

	.cm-email-message__body {
		margin-top: var(--space-4);
		max-width: 680px;
	}

	.cm-email-message__html,
	.cm-email-message__text,
	.cm-email-message__empty {
		font-size: var(--text-base);
		color: var(--dt2);
		line-height: 1.65;
	}

	.cm-email-message__text {
		white-space: pre-wrap;
		font-family: inherit;
		margin: 0;
	}

	.cm-email-message__empty {
		color: var(--dt4);
		font-style: italic;
		margin: 0;
	}

	.cm-email-message__html :global(a) {
		color: var(--bos-accent-blue);
		text-decoration: underline;
	}

	.cm-email-message__html :global(img) {
		max-width: 100%;
		height: auto;
	}

	.cm-email-message__html :global(blockquote) {
		background: var(--dbg2);
		border-left: 2px solid var(--dbd);
		padding: var(--space-2) var(--space-4);
		margin: var(--space-3) 0;
		color: var(--dt3);
	}

	.cm-email-message__html :global(table) {
		max-width: 100%;
		border-collapse: collapse;
	}

	.cm-email-message__snippet {
		flex: 1;
		color: var(--dt3);
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	@media (prefers-reduced-motion: reduce) {
		.cm-email-message--collapsed {
			transition: none;
		}
	}
</style>
