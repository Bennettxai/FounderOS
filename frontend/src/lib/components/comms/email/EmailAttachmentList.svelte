<script lang="ts">
	import { FileText, Image as ImageIcon, Download, X } from 'lucide-svelte';
	import type { EmailAttachment } from '$lib/api/comms';

	interface Props {
		attachments: EmailAttachment[];
		variant?: 'preview' | 'compose';
		onDownload?: (attachment: EmailAttachment) => void;
		onRemove?: (attachment: EmailAttachment) => void;
	}

	let {
		attachments,
		variant = 'preview',
		onDownload,
		onRemove,
	}: Props = $props();

	function iconFor(mime: string) {
		if (mime.startsWith('image/')) return ImageIcon;
		return FileText;
	}

	function formatSize(bytes: number): string {
		if (bytes < 1024) return `${bytes} B`;
		if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
		return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
	}
</script>

{#if attachments.length > 0}
	<ul class="cm-email-attachments" aria-label="Attachments">
		{#each attachments as attachment (attachment.id)}
			{@const Icon = iconFor(attachment.mime_type)}
			<li class="cm-email-attachments__chip">
				<button
					type="button"
					class="cm-email-attachments__main"
					onclick={() => onDownload?.(attachment)}
					disabled={!onDownload}
					aria-label="{variant === 'compose' ? 'Attached' : 'Download'} {attachment.filename}"
				>
					<Icon size={14} aria-hidden="true" />
					<span class="cm-email-attachments__filename">{attachment.filename}</span>
					<span class="cm-email-attachments__size">· {formatSize(attachment.size)}</span>
					{#if onDownload && variant === 'preview'}
						<Download size={12} class="cm-email-attachments__download" aria-hidden="true" />
					{/if}
				</button>
				{#if onRemove}
					<button
						type="button"
						class="cm-email-attachments__remove"
						onclick={() => onRemove(attachment)}
						aria-label="Remove attachment {attachment.filename}"
					>
						<X size={12} />
					</button>
				{/if}
			</li>
		{/each}
	</ul>
{/if}

<style>
	.cm-email-attachments {
		list-style: none;
		margin: var(--space-3) 0 0;
		padding: 0;
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2);
	}

	.cm-email-attachments__chip {
		display: inline-flex;
		align-items: center;
		background: var(--dbg2);
		border: 1px solid var(--dbd);
		border-radius: var(--radius-md);
		transition: background var(--bos-transition-fast);
	}

	.cm-email-attachments__chip:hover {
		background: var(--dbg3);
	}

	.cm-email-attachments__main {
		display: inline-flex;
		align-items: center;
		gap: var(--space-2);
		padding: var(--space-2) var(--space-3);
		background: none;
		border: none;
		font-family: inherit;
		font-size: var(--text-xs);
		color: var(--dt2);
		cursor: pointer;
	}

	.cm-email-attachments__main:disabled {
		cursor: default;
	}

	.cm-email-attachments__filename {
		max-width: 180px;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.cm-email-attachments__size {
		color: var(--dt3);
	}

	:global(.cm-email-attachments__download) {
		color: var(--dt2);
		opacity: 0;
		transition: opacity var(--bos-transition-fast);
	}

	.cm-email-attachments__chip:hover :global(.cm-email-attachments__download) {
		opacity: 1;
	}

	.cm-email-attachments__remove {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		padding: var(--space-2);
		background: none;
		border: none;
		border-left: 1px solid var(--dbd);
		color: var(--dt3);
		cursor: pointer;
	}

	.cm-email-attachments__remove:hover {
		color: var(--dt);
	}

	@media (prefers-reduced-motion: reduce) {
		.cm-email-attachments__chip,
		:global(.cm-email-attachments__download) {
			transition: none;
		}
	}
</style>
