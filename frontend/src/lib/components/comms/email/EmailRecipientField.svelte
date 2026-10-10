<script lang="ts">
	import { X } from 'lucide-svelte';
	import { isValidEmail } from './commsEmailUtils';

	export interface RecipientSuggestion {
		email: string;
		name?: string;
	}

	interface Props {
		label: string;
		recipients: string[];
		placeholder?: string;
		suggestions: RecipientSuggestion[];
		errorMessage?: string | null;
		onChange: (recipients: string[]) => void;
		onQueryChange?: (query: string) => void;
		// Optional slot for trailing controls in the field row (e.g. "Add Cc · Bcc").
		trailing?: import('svelte').Snippet;
	}

	let {
		label,
		recipients,
		placeholder = '',
		suggestions,
		errorMessage = null,
		onChange,
		onQueryChange,
		trailing,
	}: Props = $props();

	let inputValue = $state('');
	let isFocused = $state(false);
	let highlightIndex = $state(0);

	const showSuggestions = $derived(
		isFocused && inputValue.trim().length > 0 && suggestions.length > 0,
	);

	function commitChip(value: string): boolean {
		const trimmed = value.trim().replace(/[,;]+$/, '').trim();
		if (!trimmed) return false;
		if (recipients.includes(trimmed)) {
			inputValue = '';
			return true;
		}
		onChange([...recipients, trimmed]);
		inputValue = '';
		highlightIndex = 0;
		onQueryChange?.('');
		return true;
	}

	function removeChip(target: string) {
		onChange(recipients.filter((r) => r !== target));
	}

	function handleKeydown(event: KeyboardEvent) {
		if (event.key === 'Enter') {
			event.preventDefault();
			if (showSuggestions && suggestions[highlightIndex]) {
				commitChip(suggestions[highlightIndex].email);
			} else if (inputValue.trim()) {
				commitChip(inputValue);
			}
		} else if (event.key === ',' || event.key === ';' || event.key === 'Tab') {
			if (inputValue.trim()) {
				event.preventDefault();
				commitChip(inputValue);
			}
		} else if (event.key === 'Backspace' && !inputValue && recipients.length) {
			onChange(recipients.slice(0, -1));
		} else if (event.key === 'ArrowDown' && showSuggestions) {
			event.preventDefault();
			highlightIndex = (highlightIndex + 1) % suggestions.length;
		} else if (event.key === 'ArrowUp' && showSuggestions) {
			event.preventDefault();
			highlightIndex =
				(highlightIndex - 1 + suggestions.length) % suggestions.length;
		} else if (event.key === 'Escape') {
			isFocused = false;
		}
	}

	function handleInput(event: Event) {
		const target = event.target as HTMLInputElement;
		inputValue = target.value;
		highlightIndex = 0;
		onQueryChange?.(inputValue);
	}

	function handleBlur() {
		// Allow click on a suggestion to land before tearing down the popover.
		setTimeout(() => {
			isFocused = false;
			if (inputValue.trim()) commitChip(inputValue);
		}, 120);
	}

	function pickSuggestion(suggestion: RecipientSuggestion) {
		commitChip(suggestion.email);
	}
</script>

<div class="cm-email-recipient" class:cm-email-recipient--error={!!errorMessage}>
	<label class="cm-email-recipient__label">{label}</label>
	<div class="cm-email-recipient__field">
		{#each recipients as recipient (recipient)}
			{@const valid = isValidEmail(recipient)}
			<span
				class="cm-email-recipient__chip"
				class:cm-email-recipient__chip--invalid={!valid}
			>
				<span class="cm-email-recipient__chip-text">{recipient}</span>
				<button
					type="button"
					class="cm-email-recipient__chip-remove"
					onclick={() => removeChip(recipient)}
					aria-label="Remove {recipient}"
				>
					<X size={12} />
				</button>
			</span>
		{/each}
		<input
			type="text"
			class="cm-email-recipient__input"
			value={inputValue}
			placeholder={recipients.length ? '' : placeholder}
			autocomplete="off"
			oninput={handleInput}
			onkeydown={handleKeydown}
			onfocus={() => (isFocused = true)}
			onblur={handleBlur}
			aria-label={label}
		/>
		{#if trailing}
			<span class="cm-email-recipient__trailing">{@render trailing()}</span>
		{/if}
		{#if showSuggestions}
			<ul class="cm-email-recipient__suggestions" role="listbox">
				{#each suggestions as suggestion, i (suggestion.email)}
					<li>
						<button
							type="button"
							class="cm-email-recipient__suggestion"
							class:cm-email-recipient__suggestion--active={i === highlightIndex}
							onmousedown={(e) => {
								e.preventDefault();
								pickSuggestion(suggestion);
							}}
							role="option"
							aria-selected={i === highlightIndex}
						>
							{#if suggestion.name}
								<span class="cm-email-recipient__suggestion-name">{suggestion.name}</span>
							{/if}
							<span class="cm-email-recipient__suggestion-email">{suggestion.email}</span>
						</button>
					</li>
				{/each}
			</ul>
		{/if}
	</div>
</div>
{#if errorMessage}
	<p class="cm-email-recipient__error" role="alert">{errorMessage}</p>
{/if}

<style>
	.cm-email-recipient {
		display: flex;
		align-items: flex-start;
		gap: var(--space-3);
		padding: var(--space-2) var(--space-4);
		border-bottom: 1px solid var(--dbd);
	}

	.cm-email-recipient__label {
		flex-shrink: 0;
		width: 64px;
		padding-top: 6px;
		font-size: var(--text-xs);
		font-weight: var(--font-medium);
		color: var(--dt3);
	}

	.cm-email-recipient__field {
		position: relative;
		flex: 1;
		min-height: 28px;
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: var(--space-2);
		padding: var(--space-1) 0;
	}

	.cm-email-recipient__chip {
		display: inline-flex;
		align-items: center;
		gap: var(--space-1);
		padding: 2px var(--space-2);
		background: var(--dbg2);
		border: 1px solid var(--dbd);
		border-radius: var(--radius-full);
		font-size: var(--text-xs);
		color: var(--dt2);
	}

	.cm-email-recipient__chip--invalid {
		color: var(--bos-status-error-text);
		border-color: var(--bos-status-error);
	}

	.cm-email-recipient__chip-text {
		max-width: 220px;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.cm-email-recipient__chip-remove {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		background: none;
		border: none;
		padding: 0;
		color: var(--dt3);
		cursor: pointer;
	}

	.cm-email-recipient__chip-remove:hover {
		color: var(--dt);
	}

	.cm-email-recipient__input {
		flex: 1;
		min-width: 120px;
		background: none;
		border: none;
		outline: none;
		font-size: var(--text-sm);
		color: var(--dt);
		font-family: inherit;
		padding: var(--space-1) 0;
	}

	.cm-email-recipient__input::placeholder {
		color: var(--dt4);
	}

	.cm-email-recipient__trailing {
		margin-left: auto;
	}

	.cm-email-recipient__suggestions {
		list-style: none;
		margin: 0;
		padding: var(--space-1) 0;
		position: absolute;
		top: calc(100% + var(--space-1));
		left: 0;
		right: 0;
		background: var(--dbg);
		border: 1px solid var(--dbd);
		border-radius: var(--radius-md);
		box-shadow: var(--bos-popover-shadow);
		z-index: var(--bos-z-popover);
		max-height: 240px;
		overflow-y: auto;
	}

	.cm-email-recipient__suggestion {
		display: flex;
		flex-direction: column;
		gap: 1px;
		width: 100%;
		padding: var(--space-2) var(--space-3);
		background: none;
		border: none;
		text-align: left;
		font-family: inherit;
		cursor: pointer;
	}

	.cm-email-recipient__suggestion:hover,
	.cm-email-recipient__suggestion--active {
		background: var(--dbg2);
	}

	.cm-email-recipient__suggestion-name {
		font-size: var(--text-sm);
		color: var(--dt);
	}

	.cm-email-recipient__suggestion-email {
		font-size: var(--text-xs);
		color: var(--dt3);
	}

	.cm-email-recipient__error {
		margin: 0 var(--space-4) var(--space-2) calc(var(--space-4) + 64px + var(--space-3));
		font-size: var(--text-xs);
		color: var(--bos-status-error-text);
	}
</style>
