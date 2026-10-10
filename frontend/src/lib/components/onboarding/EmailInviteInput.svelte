<!--
  EmailInviteInput.svelte
  Add/remove team member email invites
-->
<script lang="ts">
	interface Props {
		emails?: string[];
		placeholder?: string;
		maxEmails?: number;
		onEmailsChange?: (emails: string[]) => void;
		class?: string;
	}

	let {
		emails = $bindable([]),
		placeholder = 'Enter email address',
		maxEmails = 10,
		onEmailsChange,
		class: className = ''
	}: Props = $props();

	let inputValue = $state('');
	let errorMessage = $state('');

	function isValidEmail(email: string): boolean {
		const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
		return emailRegex.test(email);
	}

	function addEmail() {
		const trimmed = inputValue.trim().toLowerCase();
		errorMessage = '';

		if (!trimmed) return;

		if (!isValidEmail(trimmed)) {
			errorMessage = 'Please enter a valid email address';
			return;
		}

		if (emails.includes(trimmed)) {
			errorMessage = 'This email has already been added';
			return;
		}

		if (emails.length >= maxEmails) {
			errorMessage = `Maximum of ${maxEmails} emails allowed`;
			return;
		}

		emails = [...emails, trimmed];
		inputValue = '';
		onEmailsChange?.(emails);
	}

	function removeEmail(email: string) {
		emails = emails.filter((e) => e !== email);
		onEmailsChange?.(emails);
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter') {
			e.preventDefault();
			addEmail();
		}
	}
</script>

<div class="email-invite {className}">
	<div class="input-row">
		<input
			type="email"
			class="email-input"
			bind:value={inputValue}
			{placeholder}
			onkeydown={handleKeydown}
		/>
		<button
			type="button"
			class="btn-pill btn-pill-ghost add-btn"
			onclick={addEmail}
			disabled={!inputValue.trim()}
		>
			Add
		</button>
	</div>

	{#if errorMessage}
		<p class="error">{errorMessage}</p>
	{/if}

	{#if emails.length > 0}
		<div class="email-list">
			{#each emails as email (email)}
				<div class="email-tag">
					<span class="email-text">{email}</span>
					<button
						type="button"
						class="btn-pill btn-pill-ghost remove-btn"
						onclick={() => removeEmail(email)}
						aria-label="Remove {email}"
					>
						<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
							<path d="M18 6 6 18" />
							<path d="m6 6 12 12" />
						</svg>
					</button>
				</div>
			{/each}
		</div>
	{/if}
</div>

<style>
	.email-invite {
		display: flex;
		flex-direction: column;
		gap: 12px;
	}

	.input-row {
		display: flex;
		gap: 8px;
	}

	.email-input {
		flex: 1;
		height: 40px;
		padding: 0 16px;
		font-size: 14px;
		font-family: inherit;
		color: var(--foreground, #1f2937);
		background-color: var(--background, #ffffff);
		border: 1px solid var(--border, #e5e7eb);
		border-radius: 8px;
		outline: none;
		transition: border-color 0.2s ease;
	}

	.email-input::placeholder {
		color: var(--muted-foreground, #9ca3af);
	}

	.email-input:focus {
		border-color: var(--primary, #000000);
	}

	.add-btn {
		padding: 0 20px;
		height: 40px;
		font-size: 14px;
		font-weight: 500;
		border: none;
		border-radius: 8px;
		background-color: var(--primary, #000000);
		color: var(--primary-foreground, #ffffff);
		cursor: pointer;
		transition: opacity 0.2s ease;
	}

	.add-btn:hover:not(:disabled) {
		opacity: 0.9;
	}

	.add-btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.error {
		font-size: 13px;
		color: var(--error, #ef4444);
		margin: 0;
	}

	.email-list {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
	}

	.email-tag {
		display: flex;
		align-items: center;
		gap: 6px;
		padding: 6px 10px;
		background-color: var(--secondary, #f9fafb);
		border: 1px solid var(--border, #e5e7eb);
		border-radius: 20px;
	}

	.email-text {
		font-size: 13px;
		color: var(--foreground, #1f2937);
	}

	.remove-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 18px;
		height: 18px;
		padding: 0;
		border: none;
		border-radius: 50%;
		background-color: transparent;
		color: var(--muted-foreground, #6b7280);
		cursor: pointer;
		transition: all 0.2s ease;
	}

	.remove-btn:hover {
		background-color: var(--error, #ef4444);
		color: white;
	}

	/* Dark mode */
	:global(.dark) .email-input {
		background-color: var(--background, #0a0a0a);
		color: var(--foreground, #f9fafb);
		border-color: var(--border, #2a2a2a);
	}

	:global(.dark) .email-input:focus {
		border-color: var(--primary, #ffffff);
	}

	:global(.dark) .add-btn {
		background-color: var(--primary, #ffffff);
		color: var(--primary-foreground, #000000);
	}

	:global(.dark) .email-tag {
		background-color: var(--secondary, #1a1a1a);
		border-color: var(--border, #2a2a2a);
	}

	:global(.dark) .email-text {
		color: var(--foreground, #f9fafb);
	}
</style>
