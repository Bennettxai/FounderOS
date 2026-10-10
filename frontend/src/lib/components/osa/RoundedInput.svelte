<!--
	RoundedInput.svelte
	iOS-style rounded input with glassmorphism

	Usage:
	<RoundedInput
		label="Email"
		type="email"
		placeholder="you@company.com"
		bind:value={email}
	/>
-->
<script lang="ts">
	import type { HTMLInputAttributes } from 'svelte/elements';

	interface Props {
		label?: string;
		type?: string;
		value?: string;
		placeholder?: string;
		disabled?: boolean;
		required?: boolean;
		autocomplete?: HTMLInputAttributes['autocomplete'];
		error?: string;
		helperText?: string;
		id?: string;
		name?: string;
		class?: string;
	}

	let {
		label,
		type = 'text',
		value = $bindable(''),
		placeholder,
		disabled = false,
		required = false,
		autocomplete,
		error,
		helperText,
		id,
		name,
		class: className = ''
	}: Props = $props();

	let inputId = $derived(id || `input-${Math.random().toString(36).substr(2, 9)}`);
</script>

<div class="input-container">
	{#if label}
		<label for={inputId} class="input-label">
			{label}
			{#if required}
				<span class="required-star">*</span>
			{/if}
		</label>
	{/if}

	<input
		id={inputId}
		{name}
		{type}
		bind:value
		{placeholder}
		{disabled}
		{required}
		{autocomplete}
		class="glass-input {className}"
		class:input-error={error}
		aria-invalid={error ? 'true' : 'false'}
		aria-describedby={error ? `${inputId}-error` : helperText ? `${inputId}-helper` : undefined}
	/>

	{#if error}
		<p id="{inputId}-error" class="error-text">
			{error}
		</p>
	{:else if helperText}
		<p id="{inputId}-helper" class="helper-text">
			{helperText}
		</p>
	{/if}
</div>

<style>
	.input-container {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
		width: 100%;
	}

	.input-label {
		font-size: 0.875rem;
		font-weight: 500;
		color: #374151;
		text-align: left;
	}

	.required-star {
		color: #DC2626;
		margin-left: 0.125rem;
	}

	.glass-input {
		width: 100%;
		padding: 0.875rem 1rem;
		font-size: 1rem;
		color: #1A1A1A;
		background: rgba(255, 255, 255, 0.75);
		backdrop-filter: blur(12px);
		-webkit-backdrop-filter: blur(12px);
		border: 1.5px solid rgba(0, 0, 0, 0.1);
		border-radius: 0.75rem;
		font-family: inherit;
		transition: all 0.2s ease;
		box-shadow:
			0 2px 8px rgba(0, 0, 0, 0.04),
			inset 0 1px 0 rgba(255, 255, 255, 0.6);
	}

	.glass-input::placeholder {
		color: #9CA3AF;
	}

	.glass-input:focus {
		outline: none;
		background: rgba(255, 255, 255, 0.9);
		border-color: rgba(26, 26, 26, 0.25);
		box-shadow:
			0 4px 12px rgba(0, 0, 0, 0.08),
			0 0 0 3px rgba(26, 26, 26, 0.06);
	}

	.glass-input:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.glass-input.input-error {
		border-color: #DC2626;
	}

	.error-text {
		font-size: 0.75rem;
		color: #DC2626;
		margin: 0;
		text-align: left;
	}

	.helper-text {
		font-size: 0.75rem;
		color: #6B7280;
		margin: 0;
		text-align: left;
	}
</style>
