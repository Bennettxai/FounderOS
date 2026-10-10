<!--
  FallbackForm.svelte
  Traditional form fallback when agent confidence is low
-->
<script lang="ts">
	import { Button } from '$lib/ui';

	interface FormField {
		id: string;
		label: string;
		type: 'text' | 'email' | 'textarea' | 'select';
		placeholder?: string;
		required?: boolean;
		options?: { value: string; label: string }[];
	}

	interface Props {
		title?: string;
		subtitle?: string;
		fields: FormField[];
		values?: Record<string, string>;
		submitLabel?: string;
		onSubmit?: (values: Record<string, string>) => void;
		class?: string;
	}

	let {
		title = 'Let\'s get your info',
		subtitle = 'We need a few more details to complete your setup.',
		fields,
		values = $bindable({}),
		submitLabel = 'Continue',
		onSubmit,
		class: className = ''
	}: Props = $props();

	// Initialize values
	$effect(() => {
		for (const field of fields) {
			if (!(field.id in values)) {
				values[field.id] = '';
			}
		}
	});

	function handleSubmit(e: Event) {
		e.preventDefault();
		onSubmit?.(values);
	}

	const isValid = $derived(
		fields.filter((f) => f.required).every((f) => values[f.id]?.trim())
	);
</script>

<form class="fallback-form {className}" onsubmit={handleSubmit}>
	{#if title}
		<h2 class="form-title">{title}</h2>
	{/if}
	{#if subtitle}
		<p class="form-subtitle">{subtitle}</p>
	{/if}

	<div class="form-fields">
		{#each fields as field (field.id)}
			<div class="form-group">
				<label for={field.id} class="label">
					{field.label}
					{#if field.required}
						<span class="required">*</span>
					{/if}
				</label>

				{#if field.type === 'textarea'}
					<textarea
						id={field.id}
						class="input textarea"
						bind:value={values[field.id]}
						placeholder={field.placeholder}
						rows="4"
					></textarea>
				{:else if field.type === 'select'}
					<select
						id={field.id}
						class="input select"
						bind:value={values[field.id]}
					>
						<option value="" disabled>
							{field.placeholder || 'Select an option'}
						</option>
						{#if field.options}
							{#each field.options as option (option.value)}
								<option value={option.value}>{option.label}</option>
							{/each}
						{/if}
					</select>
				{:else}
					<input
						id={field.id}
						type={field.type}
						class="input"
						bind:value={values[field.id]}
						placeholder={field.placeholder}
					/>
				{/if}
			</div>
		{/each}
	</div>

	<div class="form-actions">
		<Button type="submit" variant="primary" disabled={!isValid}>
			{submitLabel}
		</Button>
	</div>
</form>

<style>
	.fallback-form {
		display: flex;
		flex-direction: column;
		gap: 24px;
		max-width: 480px;
		margin: 0 auto;
	}

	.form-title {
		font-size: 24px;
		font-weight: 600;
		color: var(--foreground, #1f2937);
		margin: 0;
	}

	.form-subtitle {
		font-size: 15px;
		color: var(--muted-foreground, #6b7280);
		margin: 0;
		line-height: 1.5;
	}

	.form-fields {
		display: flex;
		flex-direction: column;
		gap: 16px;
	}

	.form-group {
		display: flex;
		flex-direction: column;
		gap: 6px;
	}

	.label {
		font-size: 14px;
		font-weight: 500;
		color: var(--foreground, #1f2937);
	}

	.required {
		color: var(--error, #ef4444);
	}

	.input {
		width: 100%;
		height: 44px;
		padding: 0 16px;
		font-size: 15px;
		font-family: inherit;
		color: var(--foreground, #1f2937);
		background-color: var(--background, #ffffff);
		border: 1px solid var(--border, #e5e7eb);
		border-radius: 8px;
		outline: none;
		transition: border-color 0.2s ease;
	}

	.input::placeholder {
		color: var(--muted-foreground, #9ca3af);
	}

	.input:focus {
		border-color: var(--primary, #000000);
	}

	.textarea {
		height: auto;
		padding: 12px 16px;
		resize: vertical;
		min-height: 100px;
	}

	.select {
		cursor: pointer;
		appearance: none;
		background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='20' height='20' viewBox='0 0 24 24' fill='none' stroke='%236b7280' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'%3E%3Cpath d='m6 9 6 6 6-6'/%3E%3C/svg%3E");
		background-repeat: no-repeat;
		background-position: right 12px center;
		padding-right: 40px;
	}

	.form-actions {
		padding-top: 8px;
	}

	/* Dark mode */
	:global(.dark) .form-title {
		color: var(--foreground, #f9fafb);
	}

	:global(.dark) .label {
		color: var(--foreground, #f9fafb);
	}

	:global(.dark) .input {
		background-color: var(--background, #0a0a0a);
		color: var(--foreground, #f9fafb);
		border-color: var(--border, #2a2a2a);
	}

	:global(.dark) .input:focus {
		border-color: var(--primary, #ffffff);
	}

	:global(.dark) .select {
		background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='20' height='20' viewBox='0 0 24 24' fill='none' stroke='%239ca3af' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'%3E%3Cpath d='m6 9 6 6 6-6'/%3E%3C/svg%3E");
	}
</style>
