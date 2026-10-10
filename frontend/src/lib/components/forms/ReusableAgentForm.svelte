<script lang="ts">
	import FormField from './FormField.svelte';
	import FormSection from './FormSection.svelte';
	import type { FormFieldConfig, AgentFormData } from '$lib/types/forms';

	interface Props {
		data: AgentFormData;
		fields: FormFieldConfig[];
		title?: string;
		description?: string;
		loading?: boolean;
		error?: string;
		successMessage?: string;
		submitLabel?: string;
		cancelLabel?: string;
		onSubmit: (data: AgentFormData) => Promise<void>;
		onCancel?: () => void;
	}

	let {
		data = $bindable(),
		fields,
		title = 'Agent Configuration',
		description = undefined,
		loading = false,
		error = '',
		successMessage = '',
		submitLabel = 'Save Agent',
		cancelLabel = 'Cancel',
		onSubmit,
		onCancel
	}: Props = $props();

	let isSubmitting = $state(false);
	let localError = $state('');
	let localSuccess = $state('');
	let fieldErrors = $state<Record<string, string>>({});
	let fieldValues = $derived(data as Record<string, string | number | boolean | string[]>);

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();

		// Reset previous errors
		localError = '';
		localSuccess = '';
		fieldErrors = {};

		// Run field validation if provided
		const errors: Record<string, string> = {};
		for (const field of fields) {
			if (field.validation) {
				const fieldValue = data[field.name];
				const error = field.validation(fieldValue);
				if (error) {
					errors[field.name] = error;
				}
			} else if (field.required && !data[field.name]) {
				errors[field.name] = `${field.label} is required`;
			}
		}

		if (Object.keys(errors).length > 0) {
			fieldErrors = errors;
			localError = 'Please fix the errors below';
			return;
		}

		isSubmitting = true;

		try {
			await onSubmit(data);
			localSuccess = 'Changes saved successfully!';
			setTimeout(() => {
				localSuccess = '';
			}, 3000);
		} catch (err) {
			localError = err instanceof Error ? err.message : 'An error occurred while saving';
			setTimeout(() => {
				localError = '';
			}, 4000);
		} finally {
			isSubmitting = false;
		}
	}

	function resetErrors(fieldName: string) {
		if (fieldErrors[fieldName]) {
			fieldErrors[fieldName] = '';
		}
	}
</script>

<form onsubmit={handleSubmit} class="space-y-6">
	<!-- Header -->
	{#if title}
		<div class="mb-6">
			<h2 class="text-2xl font-bold text-base-content">{title}</h2>
			{#if description}
				<p class="text-base-content/60 mt-2">{description}</p>
			{/if}
		</div>
	{/if}

	<!-- Error Alert -->
	{#if localError || error}
		<div class="alert alert-error gap-2">
			<svg
				xmlns="http://www.w3.org/2000/svg"
				class="h-6 w-6 shrink-0 stroke-current"
				fill="none"
				viewBox="0 0 24 24"
			>
				<path
					stroke-linecap="round"
					stroke-linejoin="round"
					stroke-width="2"
					d="M12 9v2m0 4v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
				/>
			</svg>
			<span>{localError || error}</span>
		</div>
	{/if}

	<!-- Success Alert -->
	{#if localSuccess || successMessage}
		<div class="alert alert-success gap-2">
			<svg
				xmlns="http://www.w3.org/2000/svg"
				class="h-6 w-6 shrink-0 stroke-current"
				fill="none"
				viewBox="0 0 24 24"
			>
				<path
					stroke-linecap="round"
					stroke-linejoin="round"
					stroke-width="2"
					d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"
				/>
			</svg>
			<span>{localSuccess || successMessage}</span>
		</div>
	{/if}

	<!-- Form Fields -->
	<FormSection title="Basic Information" description="Agent identity and core settings">
		{#each fields as field (field.name)}
			{#if !['system_prompt', 'category', 'temperature', 'max_tokens', 'thinking_enabled', 'streaming_enabled'].includes(field.name)}
				<FormField
					{...field}
					error={fieldErrors[field.name] ?? ''}
					onchange={() => resetErrors(field.name)}
					bind:value={fieldValues[field.name]}
				/>
			{/if}
		{/each}
	</FormSection>

	<!-- Advanced Settings Section -->
	<FormSection title="Advanced Settings" description="Model preferences and behavior">
		{#each fields as field (field.name)}
			{#if ['category', 'temperature', 'max_tokens'].includes(field.name)}
				<FormField
					{...field}
					error={fieldErrors[field.name] ?? ''}
					onchange={() => resetErrors(field.name)}
					bind:value={fieldValues[field.name]}
				/>
			{/if}
		{/each}
	</FormSection>

	<!-- System Prompt Section -->
	{#each fields as field (field.name)}
		{#if field.name === 'system_prompt'}
			<FormSection title="System Prompt" description="Define how the agent behaves and responds">
				<FormField
					{...field}
					error={fieldErrors[field.name] ?? ''}
					onchange={() => resetErrors(field.name)}
					bind:value={fieldValues[field.name]}
				/>
			</FormSection>
		{/if}
	{/each}

	<!-- Form Actions -->
	<div class="flex justify-end gap-3 pt-4 border-t border-base-300">
		{#if onCancel}
			<button
				type="button"
				onclick={onCancel}
				disabled={isSubmitting || loading}
				class="btn-pill btn-pill-ghost btn btn-outline"
			>
				{cancelLabel}
			</button>
		{/if}

		<button
			type="submit"
			disabled={isSubmitting || loading}
			class="btn-pill btn-pill-ghost btn btn-primary"
		>
			{#if isSubmitting || loading}
				<span class="loading loading-spinner loading-sm"></span>
				Saving...
			{:else}
				{submitLabel}
			{/if}
		</button>
	</div>
</form>

<style>
	form {
		@apply max-w-2xl;
	}
</style>
