<!--
  ChatInput.svelte
  Large pill-shaped chat input with send button
-->
<script lang="ts">
	import { SendIcon } from './icons';

	interface Props {
		value?: string;
		placeholder?: string;
		disabled?: boolean;
		showMic?: boolean;
		onSend?: (message: string) => void;
		class?: string;
	}

	let {
		value = $bindable(''),
		placeholder = 'Type your message...',
		disabled = false,
		showMic = false,
		onSend,
		class: className = ''
	}: Props = $props();

	function handleSubmit(e: Event) {
		e.preventDefault();
		if (value.trim() && !disabled) {
			onSend?.(value.trim());
			value = '';
		}
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter' && !e.shiftKey) {
			e.preventDefault();
			handleSubmit(e);
		}
	}
</script>

<form class="chat-input-wrapper {className}" onsubmit={handleSubmit}>
	<input
		type="text"
		class="chat-input"
		bind:value
		{placeholder}
		{disabled}
		onkeydown={handleKeydown}
	/>

	<div class="actions">
		{#if showMic}
			<button
				type="button"
				class="btn-pill btn-pill-ghost action-btn mic-btn"
				{disabled}
				aria-label="Voice input"
			>
				<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
					<path d="M12 2a3 3 0 0 0-3 3v7a3 3 0 0 0 6 0V5a3 3 0 0 0-3-3Z" />
					<path d="M19 10v2a7 7 0 0 1-14 0v-2" />
					<line x1="12" x2="12" y1="19" y2="22" />
				</svg>
			</button>
		{/if}

		<button
			type="submit"
			class="btn-pill btn-pill-ghost action-btn send-btn"
			disabled={disabled || !value.trim()}
			aria-label="Send message"
		>
			<SendIcon size={20} />
		</button>
	</div>
</form>

<style>
	.chat-input-wrapper {
		position: relative;
		width: 100%;
		display: flex;
		align-items: center;
	}

	.chat-input {
		width: 100%;
		height: 56px;
		padding: 0 100px 0 24px;
		font-size: 16px;
		font-family: inherit;
		color: var(--foreground, #1f2937);
		background-color: var(--background, #ffffff);
		border: 2px solid var(--border, #e5e7eb);
		border-radius: 9999px;
		outline: none;
		transition: border-color 0.2s ease;
	}

	.chat-input::placeholder {
		color: var(--muted-foreground, #9ca3af);
	}

	.chat-input:focus {
		border-color: var(--primary, #000000);
	}

	.chat-input:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.actions {
		position: absolute;
		right: 8px;
		display: flex;
		align-items: center;
		gap: 4px;
	}

	.action-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 40px;
		height: 40px;
		border-radius: 50%;
		border: none;
		background-color: transparent;
		color: var(--muted-foreground, #6b7280);
		cursor: pointer;
		transition: all 0.2s ease;
	}

	.action-btn:hover:not(:disabled) {
		background-color: var(--accent, #f3f4f6);
		color: var(--foreground, #1f2937);
	}

	.action-btn:disabled {
		opacity: 0.3;
		cursor: not-allowed;
	}

	.send-btn:not(:disabled) {
		background-color: var(--primary, #000000);
		color: var(--primary-foreground, #ffffff);
	}

	.send-btn:hover:not(:disabled) {
		opacity: 0.9;
	}

	/* Dark mode */
	:global(.dark) .chat-input {
		background-color: var(--background, #0a0a0a);
		color: var(--foreground, #f9fafb);
		border-color: var(--border, #2a2a2a);
	}

	:global(.dark) .chat-input:focus {
		border-color: var(--primary, #ffffff);
	}

	:global(.dark) .action-btn:hover:not(:disabled) {
		background-color: var(--accent, #2a2a2a);
		color: var(--foreground, #f9fafb);
	}

	:global(.dark) .send-btn:not(:disabled) {
		background-color: var(--primary, #ffffff);
		color: var(--primary-foreground, #000000);
	}
</style>
