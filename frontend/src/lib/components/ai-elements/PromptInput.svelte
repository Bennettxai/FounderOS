<script lang="ts">
	import { fly } from 'svelte/transition';
	import type { Snippet } from 'svelte';

	interface Props {
		value?: string;
		placeholder?: string;
		disabled?: boolean;
		loading?: boolean;
		leftSlot?: Snippet;
		rightSlot?: Snippet;
		onSubmit?: (value: string) => void;
		onStop?: () => void;
		class?: string;
	}

	let {
		value = $bindable(''),
		placeholder = 'Send a message...',
		disabled = false,
		loading = false,
		leftSlot,
		rightSlot,
		onSubmit,
		onStop,
		class: className = ''
	}: Props = $props();

	let textareaRef: HTMLTextAreaElement | undefined = $state(undefined);

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter' && !e.shiftKey) {
			e.preventDefault();
			handleSubmit();
		}
	}

	function handleSubmit() {
		if (!value.trim() || disabled || loading) return;
		onSubmit?.(value);
		value = '';
		resetHeight();
	}

	function handleInput() {
		if (textareaRef) {
			textareaRef.style.height = 'auto';
			textareaRef.style.height = Math.min(textareaRef.scrollHeight, 200) + 'px';
		}
	}

	function resetHeight() {
		if (textareaRef) {
			textareaRef.style.height = 'auto';
		}
	}

	export function focus() {
		textareaRef?.focus();
	}
</script>

<div class="ai-prompt-input {className}">
	<div class="ai-prompt-input__container">
		{#if leftSlot}
			<div class="ai-prompt-input__left">
				{@render leftSlot()}
			</div>
		{/if}

		<div class="ai-prompt-input__field">
			<textarea
				bind:this={textareaRef}
				bind:value
				{placeholder}
				disabled={disabled || loading}
				rows={1}
				class="ai-prompt-input__textarea"
				onkeydown={handleKeydown}
				oninput={handleInput}
			></textarea>
		</div>

		<div class="ai-prompt-input__right">
			{#if rightSlot}
				{@render rightSlot()}
			{/if}

			{#if loading}
				<button
					type="button"
					onclick={onStop}
					class="btn-pill btn-pill-ghost ai-prompt-input__button ai-prompt-input__button--stop"
					aria-label="Stop generating"
				>
					<svg class="ai-prompt-input__icon" fill="currentColor" viewBox="0 0 24 24">
						<rect x="6" y="6" width="12" height="12" rx="2" />
					</svg>
				</button>
			{:else}
				<button
					type="button"
					onclick={handleSubmit}
					disabled={!value.trim() || disabled}
					class="btn-pill btn-pill-ghost ai-prompt-input__button ai-prompt-input__button--send"
					aria-label="Send message"
				>
					<svg class="ai-prompt-input__icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 10l7-7m0 0l7 7m-7-7v18" />
					</svg>
				</button>
			{/if}
		</div>
	</div>
</div>

<style>
	.ai-prompt-input {
		width: 100%;
	}

	.ai-prompt-input__container {
		display: flex;
		align-items: flex-end;
		gap: 0.5rem;
		padding: 0.75rem;
		background-color: var(--card);
		border: 1px solid var(--border);
		border-radius: 1.5rem;
		box-shadow: 0 1px 3px 0 rgb(0 0 0 / 0.1);
		transition: border-color 0.2s ease, box-shadow 0.2s ease;
	}

	.ai-prompt-input__container:focus-within {
		border-color: var(--ring);
		box-shadow: 0 0 0 3px rgb(0 0 0 / 0.05);
	}

	.ai-prompt-input__left {
		display: flex;
		align-items: center;
		gap: 0.25rem;
		flex-shrink: 0;
	}

	.ai-prompt-input__field {
		flex: 1;
		min-width: 0;
	}

	.ai-prompt-input__textarea {
		width: 100%;
		min-height: 2.75rem;
		max-height: 200px;
		padding: 0.5rem 0.75rem;
		font-size: 0.9375rem;
		line-height: 1.5;
		color: var(--foreground);
		background-color: transparent;
		border: none;
		resize: none;
		outline: none;
	}

	.ai-prompt-input__textarea::placeholder {
		color: var(--muted-foreground);
	}

	.ai-prompt-input__textarea:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.ai-prompt-input__right {
		display: flex;
		align-items: center;
		gap: 0.25rem;
		flex-shrink: 0;
	}

	.ai-prompt-input__button {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 2.5rem;
		height: 2.5rem;
		border-radius: 0.75rem;
		border: none;
		cursor: pointer;
		transition: all 0.2s ease;
	}

	.ai-prompt-input__button--send {
		background-color: #3b82f6;
		color: white;
	}

	.ai-prompt-input__button--send:hover:not(:disabled) {
		background-color: #2563eb;
	}

	.ai-prompt-input__button--send:disabled {
		opacity: 0.4;
		cursor: not-allowed;
	}

	.ai-prompt-input__button--stop {
		background-color: #ef4444;
		color: white;
	}

	.ai-prompt-input__button--stop:hover {
		background-color: #dc2626;
	}

	.ai-prompt-input__icon {
		width: 1.25rem;
		height: 1.25rem;
	}

	/* Dark mode */
	:global(.dark) .ai-prompt-input__container {
		background-color: var(--card);
		border-color: var(--border);
	}

	:global(.dark) .ai-prompt-input__container:focus-within {
		box-shadow: 0 0 0 3px rgb(255 255 255 / 0.1);
	}
</style>
