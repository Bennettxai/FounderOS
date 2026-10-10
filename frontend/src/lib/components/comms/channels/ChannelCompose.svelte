<script lang="ts">
	import { tick } from 'svelte';
	import { AtSign, Smile, Paperclip, Send, X, Loader2 } from 'lucide-svelte';
	import PillButton from '$lib/components/ui/PillButton.svelte';
	import type { CommsChannel } from '$lib/api/comms';

	export interface ComposeAttachment {
		id: string;
		name: string;
		size_bytes: number;
		mime_type: string;
		// File reference is kept on the orchestrator side; this component renders chips only.
	}

	interface Props {
		channel: CommsChannel | null;
		// Pre-filled draft text — used when the orchestrator restores from localStorage
		// or pre-populates a thread reply context.
		value: string;
		attachments?: ComposeAttachment[];
		isSending: boolean;
		// Placeholder hint — defaults to "Message #channel" but the thread drawer
		// passes "Reply to thread…".
		placeholder?: string;
		onChange: (value: string) => void;
		onSend: () => void;
		onAttach?: () => void;
		onMention?: () => void;
		onEmojiPicker?: () => void;
		onRemoveAttachment?: (attachment: ComposeAttachment) => void;
	}

	let {
		channel,
		value,
		attachments = [],
		isSending,
		placeholder,
		onChange,
		onSend,
		onAttach,
		onMention,
		onEmojiPicker,
		onRemoveAttachment,
	}: Props = $props();

	const computedPlaceholder = $derived(
		placeholder ??
			(channel
				? channel.is_dm
					? `Message ${channel.name}`
					: `Message #${channel.name}`
				: 'Message…'),
	);

	const trimmedValue = $derived(value.trim());
	const canSend = $derived(
		!isSending && (!!trimmedValue || attachments.length > 0) && !!channel,
	);

	let textareaEl = $state<HTMLTextAreaElement | null>(null);

	function handleInput(event: Event) {
		const target = event.target as HTMLTextAreaElement;
		onChange(target.value);
		autoresize(target);
	}

	function autoresize(el: HTMLTextAreaElement) {
		el.style.height = 'auto';
		const next = Math.min(el.scrollHeight, 160);
		el.style.height = `${next}px`;
	}

	function handleKeydown(event: KeyboardEvent) {
		if (event.key !== 'Enter') return;
		// Shift+Enter = newline, plain Enter = send, ⌘/Ctrl+Enter = send.
		if (event.shiftKey) return;
		event.preventDefault();
		if (canSend) onSend();
	}

	// Refocus the textarea when the channel changes (and there's a channel).
	$effect(() => {
		const id = channel?.id;
		if (!id || !textareaEl) return;
		const el = textareaEl;
		tick().then(() => {
			el.focus();
			autoresize(el);
		});
	});

	function formatSize(bytes: number): string {
		if (bytes < 1024) return `${bytes} B`;
		if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
		return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
	}
</script>

<div class="cm-channels-compose">
	{#if attachments.length > 0}
		<ul class="cm-channels-compose__chips" role="list">
			{#each attachments as attachment (attachment.id)}
				<li class="cm-channels-compose__chip">
					<Paperclip size={12} />
					<span class="cm-channels-compose__chip-name">{attachment.name}</span>
					<span class="cm-channels-compose__chip-size">
						{formatSize(attachment.size_bytes)}
					</span>
					{#if onRemoveAttachment}
						<button
							type="button"
							class="cm-channels-compose__chip-remove"
							onclick={() => onRemoveAttachment(attachment)}
							aria-label="Remove attachment {attachment.name}"
						>
							<X size={12} />
						</button>
					{/if}
				</li>
			{/each}
		</ul>
	{/if}

	<textarea
		bind:this={textareaEl}
		class="bos-textarea cm-channels-compose__textarea"
		placeholder={computedPlaceholder}
		aria-label={computedPlaceholder}
		rows="1"
		value={value}
		disabled={!channel || isSending}
		oninput={handleInput}
		onkeydown={handleKeydown}
	></textarea>

	<div class="cm-channels-compose__actions">
		{#if onMention}
			<button
				type="button"
				class="btn-compact btn-compact-ghost btn-compact-icon"
				aria-label="Mention someone"
				disabled={!channel}
				onclick={onMention}
			>
				<AtSign size={14} />
			</button>
		{/if}
		{#if onEmojiPicker}
			<button
				type="button"
				class="btn-compact btn-compact-ghost btn-compact-icon"
				aria-label="Insert emoji"
				disabled={!channel}
				onclick={onEmojiPicker}
			>
				<Smile size={14} />
			</button>
		{/if}
		{#if onAttach}
			<button
				type="button"
				class="btn-compact btn-compact-ghost btn-compact-icon"
				aria-label="Attach file"
				disabled={!channel}
				onclick={onAttach}
			>
				<Paperclip size={14} />
			</button>
		{/if}

		<span class="cm-channels-compose__hint" aria-hidden="true">
			Enter to send · Shift+Enter newline
		</span>

		<PillButton
			variant="cta"
			size="sm"
			disabled={!canSend}
			onclick={onSend}
			aria-label="Send message"
		>
			{#if isSending}
				<Loader2 size={14} class="cm-spin" />
			{:else}
				<Send size={14} />
			{/if}
		</PillButton>
	</div>
</div>

<style>
	.cm-channels-compose {
		flex-shrink: 0;
		padding: var(--space-3) var(--space-4);
		background: var(--dbg2);
		border-top: 1px solid var(--dbd);
		display: flex;
		flex-direction: column;
		gap: var(--space-2);
	}

	.cm-channels-compose__textarea {
		width: 100%;
		min-height: 40px;
		max-height: 160px;
		resize: none;
		font-size: var(--text-sm);
		line-height: 1.5;
		transition: border-color 150ms ease, box-shadow 150ms ease;
	}

	.cm-channels-compose__textarea:focus,
	.cm-channels-compose__textarea:focus-visible {
		outline: none;
		border-color: var(--bos-accent-blue);
		box-shadow: 0 0 0 3px rgba(var(--bos-accent-blue-rgb), 0.18);
	}

	.cm-channels-compose__actions {
		display: flex;
		align-items: center;
		gap: var(--space-1);
	}

	.cm-channels-compose__hint {
		flex: 1;
		font-size: var(--text-xs);
		color: var(--dt3);
		text-align: right;
		margin-right: var(--space-2);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.cm-channels-compose__chips {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2);
	}

	.cm-channels-compose__chip {
		display: inline-flex;
		align-items: center;
		gap: var(--space-2);
		padding: var(--space-1) var(--space-3);
		background: var(--dbg);
		color: var(--dt2);
		border: 1px solid var(--dbd);
		border-radius: var(--radius-md);
		font-size: var(--text-xs);
	}

	.cm-channels-compose__chip-name {
		max-width: 220px;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.cm-channels-compose__chip-size {
		color: var(--dt3);
	}

	.cm-channels-compose__chip-remove {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 18px;
		height: 18px;
		background: none;
		border: none;
		color: var(--dt3);
		cursor: pointer;
		border-radius: var(--radius-full);
		transition: background var(--bos-transition-fast), color var(--bos-transition-fast);
	}

	.cm-channels-compose__chip-remove:hover {
		background: var(--dbg3);
		color: var(--dt);
	}

	@media (prefers-reduced-motion: reduce) {
		.cm-channels-compose__chip-remove {
			transition: none;
		}
	}
</style>
