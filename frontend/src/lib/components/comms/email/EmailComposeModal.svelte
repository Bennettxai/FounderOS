<script lang="ts">
	import { Send, Paperclip, X } from 'lucide-svelte';
	import PillButton from '$lib/components/ui/PillButton.svelte';
	import EmailRecipientField, {
		type RecipientSuggestion,
	} from './EmailRecipientField.svelte';
	import EmailRichTextBody from './EmailRichTextBody.svelte';
	import EmailAttachmentList from './EmailAttachmentList.svelte';
	import { isValidEmail, providerLabel } from './commsEmailUtils';
	import type { EmailAccount, EmailAttachment } from '$lib/api/comms';

	export interface ComposeAttachment extends EmailAttachment {
		// Local file kept until upload + send. Lives on the modal-level state so
		// the orchestrator can post multipart on send.
		local?: File;
	}

	export interface ComposeDraft {
		from_account_id?: string;
		to: string[];
		cc: string[];
		bcc: string[];
		subject: string;
		body: string;
		attachments: ComposeAttachment[];
	}

	interface Props {
		open: boolean;
		draft: ComposeDraft;
		accounts: EmailAccount[];
		recipientSuggestions: RecipientSuggestion[];
		isSending: boolean;
		errorMessage?: string | null;
		autoSaveStatus?: 'idle' | 'saving' | 'saved' | 'error';
		autoSaveAt?: string | null;
		onClose: () => void;
		onChange: (draft: ComposeDraft) => void;
		onSend: () => void;
		onDiscard: () => void;
		onPickAttachments: (files: FileList) => void;
		onRemoveAttachment: (attachment: ComposeAttachment) => void;
		onRecipientQueryChange?: (
			field: 'to' | 'cc' | 'bcc',
			query: string,
		) => void;
	}

	let {
		open,
		draft,
		accounts,
		recipientSuggestions,
		isSending,
		errorMessage = null,
		autoSaveStatus = 'idle',
		autoSaveAt = null,
		onClose,
		onChange,
		onSend,
		onDiscard,
		onPickAttachments,
		onRemoveAttachment,
		onRecipientQueryChange,
	}: Props = $props();

	let showCc = $state(false);
	let showBcc = $state(false);
	let dragOver = $state(false);
	let fileInputRef = $state<HTMLInputElement | null>(null);

	$effect(() => {
		if (draft.cc.length) showCc = true;
		if (draft.bcc.length) showBcc = true;
	});

	const fromAccount = $derived.by(() => {
		if (!accounts.length) return null;
		if (draft.from_account_id) {
			return (
				accounts.find((a) => a.account_id === draft.from_account_id) ??
				accounts[0]
			);
		}
		return accounts[0];
	});

	const recipientError = $derived(
		draft.to.length === 0 ? null : draft.to.some((r) => !isValidEmail(r))
			? 'Some recipients are not valid email addresses.'
			: null,
	);

	const canSend = $derived(
		!isSending && draft.to.length > 0 && draft.to.every(isValidEmail),
	);

	const autoSaveLabel = $derived.by(() => {
		switch (autoSaveStatus) {
			case 'saving':
				return 'Saving…';
			case 'saved':
				return autoSaveAt ? `Auto-saved ${formatRelativeShort(autoSaveAt)}` : 'Auto-saved';
			case 'error':
				return "Couldn't save — Retry";
			default:
				return '';
		}
	});

	function formatRelativeShort(iso: string): string {
		const ms = Date.now() - new Date(iso).getTime();
		const minutes = Math.floor(ms / 60_000);
		if (minutes < 1) return 'just now';
		if (minutes < 60) return `${minutes}m ago`;
		const hours = Math.floor(minutes / 60);
		return `${hours}h ago`;
	}

	function update<K extends keyof ComposeDraft>(key: K, value: ComposeDraft[K]) {
		onChange({ ...draft, [key]: value });
	}

	function handleFromChange(event: Event) {
		const target = event.target as HTMLSelectElement;
		update('from_account_id', target.value);
	}

	function handleSendClick() {
		if (draft.subject.trim() === '') {
			if (!confirm('Send without a subject?')) return;
		} else if (draft.body.trim() === '') {
			if (!confirm('Send without a body?')) return;
		}
		onSend();
	}

	function handleDiscardClick() {
		if (!confirm("Discard draft? This can't be undone.")) return;
		onDiscard();
	}

	function handleKeydown(event: KeyboardEvent) {
		if (event.key === 'Escape') {
			event.preventDefault();
			onClose();
		} else if ((event.metaKey || event.ctrlKey) && event.key === 'Enter') {
			event.preventDefault();
			if (canSend) handleSendClick();
		}
	}

	function handleDrop(event: DragEvent) {
		event.preventDefault();
		dragOver = false;
		if (event.dataTransfer?.files?.length) {
			onPickAttachments(event.dataTransfer.files);
		}
	}

	function handleFilePick(event: Event) {
		const target = event.target as HTMLInputElement;
		if (target.files?.length) onPickAttachments(target.files);
		target.value = '';
	}
</script>

{#if open}
	<div
		class="bos-modal-overlay cm-email-compose-overlay"
		onclick={(e) => {
			if (e.target === e.currentTarget) onClose();
		}}
		onkeydown={handleKeydown}
		role="presentation"
	>
		<div
			class="bos-modal cm-email-compose"
			role="dialog"
			aria-modal="true"
			aria-label="New message"
			ondragover={(e) => {
				e.preventDefault();
				dragOver = true;
			}}
			ondragleave={() => (dragOver = false)}
			ondrop={handleDrop}
		>
			<header class="bos-modal-header cm-email-compose__header">
				<h2 class="bos-modal-title">New message</h2>
				<button
					type="button"
					class="bos-modal-close"
					onclick={onClose}
					aria-label="Close compose"
				>
					<X size={16} />
				</button>
			</header>

			<div class="bos-modal-body cm-email-compose__body">
				{#if errorMessage}
					<div class="cm-email-compose__error" role="alert">{errorMessage}</div>
				{/if}

				<div class="cm-email-compose__field-row">
					<label class="cm-email-compose__field-label" for="cm-compose-from">From</label>
					{#if accounts.length > 1}
						<select
							id="cm-compose-from"
							class="cm-email-compose__from"
							value={fromAccount?.account_id ?? ''}
							onchange={handleFromChange}
						>
							{#each accounts as account (account.account_id || account.email)}
								<option value={account.account_id}>
									{providerLabel(account.provider)} · {account.email}
								</option>
							{/each}
						</select>
					{:else if fromAccount}
						<span class="cm-email-compose__from cm-email-compose__from--static">
							{providerLabel(fromAccount.provider)} · {fromAccount.email}
						</span>
					{:else}
						<span class="cm-email-compose__from cm-email-compose__from--static">
							No account connected
						</span>
					{/if}
				</div>

				<EmailRecipientField
					label="To"
					recipients={draft.to}
					placeholder="Recipients"
					suggestions={recipientSuggestions}
					errorMessage={recipientError}
					onChange={(next) => update('to', next)}
					onQueryChange={(q) => onRecipientQueryChange?.('to', q)}
				>
					{#snippet trailing()}
						{#if !showCc || !showBcc}
							<button
								type="button"
								class="cm-email-compose__cc-toggle"
								onclick={() => {
									showCc = true;
									showBcc = true;
								}}
							>
								Add Cc · Bcc
							</button>
						{/if}
					{/snippet}
				</EmailRecipientField>

				{#if showCc}
					<EmailRecipientField
						label="Cc"
						recipients={draft.cc}
						suggestions={recipientSuggestions}
						onChange={(next) => update('cc', next)}
						onQueryChange={(q) => onRecipientQueryChange?.('cc', q)}
					/>
				{/if}

				{#if showBcc}
					<EmailRecipientField
						label="Bcc"
						recipients={draft.bcc}
						suggestions={recipientSuggestions}
						onChange={(next) => update('bcc', next)}
						onQueryChange={(q) => onRecipientQueryChange?.('bcc', q)}
					/>
				{/if}

				<div class="cm-email-compose__field-row">
					<label class="cm-email-compose__field-label" for="cm-compose-subject">Subject</label>
					<input
						id="cm-compose-subject"
						type="text"
						class="cm-email-compose__subject"
						value={draft.subject}
						placeholder="Email subject"
						oninput={(e) => update('subject', (e.target as HTMLInputElement).value)}
					/>
				</div>

				<div class="cm-email-compose__body-wrap" class:cm-email-compose__body-wrap--drag={dragOver}>
					<EmailRichTextBody
						value={draft.body}
						onChange={(next) => update('body', next)}
					/>
					{#if dragOver}
						<div class="cm-email-compose__drop-overlay">Drop to attach</div>
					{/if}
				</div>

				<EmailAttachmentList
					attachments={draft.attachments}
					variant="compose"
					onRemove={onRemoveAttachment}
				/>
			</div>

			<footer class="bos-modal-footer cm-email-compose__footer">
				<div class="cm-email-compose__footer-left">
					<PillButton
						variant="cta"
						size="sm"
						disabled={!canSend}
						loading={isSending}
						onclick={handleSendClick}
					>
						<Send size={14} />
						<span style="margin-left: var(--space-2);">Send</span>
					</PillButton>
				</div>
				<div class="cm-email-compose__footer-center">
					<button
						type="button"
						class="btn-compact btn-compact-ghost btn-compact-icon"
						onclick={() => fileInputRef?.click()}
						aria-label="Attach files"
					>
						<Paperclip size={16} />
					</button>
					<input
						type="file"
						multiple
						bind:this={fileInputRef}
						class="cm-email-compose__file-input"
						onchange={handleFilePick}
					/>
				</div>
				<div class="cm-email-compose__footer-right">
					<button
						type="button"
						class="btn-compact btn-compact-ghost"
						onclick={handleDiscardClick}
					>
						Discard
					</button>
				</div>
			</footer>
			<p class="cm-email-compose__status" aria-live="polite">
				<span>⌘+Enter to send</span>
				{#if autoSaveLabel}
					<span class="cm-email-compose__status-sep">·</span>
					<span>{autoSaveLabel}</span>
				{/if}
			</p>
		</div>
	</div>
{/if}

<style>
	.cm-email-compose-overlay {
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.cm-email-compose {
		display: flex;
		flex-direction: column;
		width: min(720px, 92vw);
		max-height: 88vh;
	}

	.cm-email-compose__header {
		display: flex;
		align-items: center;
		justify-content: space-between;
	}

	.cm-email-compose__body {
		flex: 1;
		display: flex;
		flex-direction: column;
		padding: 0;
		min-height: 0;
		overflow-y: auto;
	}

	.cm-email-compose__error {
		margin: var(--space-3) var(--space-4);
		padding: var(--space-2) var(--space-3);
		background: var(--bos-status-error-bg);
		color: var(--bos-status-error-text);
		border-radius: var(--radius-md);
		font-size: var(--text-sm);
	}

	.cm-email-compose__field-row {
		display: flex;
		align-items: center;
		gap: var(--space-3);
		padding: var(--space-2) var(--space-4);
		border-bottom: 1px solid var(--dbd);
		transition: border-color 150ms ease, background 150ms ease;
	}

	.cm-email-compose__field-row:focus-within {
		border-bottom-color: var(--bos-accent-blue);
		background: rgba(var(--bos-accent-blue-rgb), 0.02);
	}

	.cm-email-compose__field-label {
		flex-shrink: 0;
		width: 64px;
		font-size: var(--text-xs);
		font-weight: var(--font-medium);
		color: var(--dt3);
	}

	.cm-email-compose__from,
	.cm-email-compose__subject {
		flex: 1;
		background: none;
		border: none;
		outline: none;
		font-family: inherit;
		font-size: var(--text-sm);
		color: var(--dt);
		padding: var(--space-1) 0;
	}

	.cm-email-compose__subject {
		font-weight: var(--font-semibold);
	}

	.cm-email-compose__subject::placeholder {
		color: var(--dt4);
		font-weight: var(--font-normal);
	}

	.cm-email-compose__from--static {
		padding: var(--space-1) 0;
	}

	.cm-email-compose__cc-toggle {
		background: none;
		border: none;
		font-family: inherit;
		font-size: var(--text-xs);
		color: var(--bos-accent-blue);
		cursor: pointer;
		padding: 0;
	}

	.cm-email-compose__cc-toggle:hover {
		text-decoration: underline;
	}

	.cm-email-compose__body-wrap {
		position: relative;
		flex: 1;
		min-height: 0;
	}

	.cm-email-compose__body-wrap--drag {
		outline: 2px dashed var(--bos-accent-blue);
		outline-offset: -2px;
		background: var(--bos-nav-active-bg);
	}

	.cm-email-compose__drop-overlay {
		position: absolute;
		inset: 0;
		display: flex;
		align-items: center;
		justify-content: center;
		background: var(--bos-nav-active-bg);
		color: var(--bos-accent-blue);
		font-size: var(--text-base);
		font-weight: var(--font-semibold);
		pointer-events: none;
	}

	.cm-email-compose__footer {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		box-shadow: 0 -1px 2px rgba(0, 0, 0, 0.03);
	}

	.cm-email-compose__footer-left {
		display: flex;
		gap: var(--space-2);
	}

	.cm-email-compose__footer-center {
		display: flex;
		gap: var(--space-1);
	}

	.cm-email-compose__footer-right {
		margin-left: auto;
		display: flex;
		gap: var(--space-2);
	}

	.cm-email-compose__file-input {
		display: none;
	}

	.cm-email-compose__status {
		margin: var(--space-2) var(--space-4) 0;
		display: flex;
		gap: var(--space-2);
		font-size: var(--text-xs);
		color: var(--dt3);
	}

	.cm-email-compose__status-sep {
		color: var(--dt4);
	}
</style>
