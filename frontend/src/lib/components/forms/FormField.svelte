<script lang="ts">
	import type { FormFieldConfig } from '$lib/types/forms';
	import type { HTMLInputAttributes } from 'svelte/elements';

	interface Props extends Partial<FormFieldConfig> {
		value: string | number | boolean | string[];
		error?: string;
		onchange?: () => void;
	}

	let {
		name = '',
		label = '',
		type = 'text',
		required = false,
		placeholder = '',
		help = '',
		disabled = false,
		value = $bindable(),
		error = '',
		options = [],
		minLength,
		maxLength,
		min,
		max,
		rows = 3,
		autoComplete = '',
		onchange
	}: Props = $props();

	let inputAutoComplete = $derived(autoComplete as HTMLInputAttributes['autocomplete']);
</script>

<div class="form-control w-full">
	<!-- Label -->
	<label for={name} class="label">
		<span class="label-text font-medium">
			{label}
			{#if required}
				<span class="text-error">*</span>
			{/if}
		</span>
	</label>

	<!-- Text Input -->
	{#if type === 'text' || type === 'email' || type === 'password'}
		<input
			{name}
			type={type}
			{required}
			{disabled}
			{placeholder}
			minlength={minLength}
			maxlength={maxLength}
			autocomplete={inputAutoComplete}
			{onchange}
			bind:value
			class="input input-bordered w-full {error ? 'input-error' : ''}"
		/>
	{/if}

	<!-- Number Input -->
	{#if type === 'number'}
		<input
			{name}
			type="number"
			{required}
			{disabled}
			{placeholder}
			{min}
			{max}
			{onchange}
			bind:value
			class="input input-bordered w-full {error ? 'input-error' : ''}"
		/>
	{/if}

	<!-- Textarea -->
	{#if type === 'textarea'}
		<textarea
			{name}
			{required}
			{disabled}
			{placeholder}
			{rows}
			minlength={minLength}
			maxlength={maxLength}
			{onchange}
			bind:value
			class="textarea textarea-bordered w-full {error ? 'textarea-error' : ''}"
		></textarea>
	{/if}

	<!-- Select -->
	{#if type === 'select'}
		<select
			{name}
			{required}
			{disabled}
			{onchange}
			bind:value
			class="select select-bordered w-full {error ? 'select-error' : ''}"
		>
			<option disabled selected value="">
				Choose one
			</option>
			{#each options as option (option.value)}
				<option value={option.value}>
					{option.label}
				</option>
			{/each}
		</select>
	{/if}

	<!-- Checkbox -->
	{#if type === 'checkbox'}
		<label class="label cursor-pointer justify-start gap-3">
			<input
				{name}
				type="checkbox"
				{required}
				disabled={!!disabled}
				checked={value === true}
				onchange={(event) => {
					value = event.currentTarget.checked;
					onchange?.();
				}}
				class="checkbox checkbox-primary"
			/>
			<span class="label-text">{label}</span>
		</label>
	{/if}

	<!-- Radio Buttons -->
	{#if type === 'radio'}
		<div class="space-y-2">
			{#each options as option (option.value)}
				<label class="label cursor-pointer justify-start gap-3">
					<input
						{name}
						type="radio"
						{required}
						{disabled}
						value={option.value}
						{onchange}
						bind:group={value}
						class="radio radio-primary"
					/>
					<span class="label-text">{option.label}</span>
				</label>
			{/each}
		</div>
	{/if}

	<!-- Error Message -->
	{#if error}
		<div class="label">
			<span class="label-text-alt text-error flex items-center gap-1">
				<svg class="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
					<path
						fill-rule="evenodd"
						d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z"
						clip-rule="evenodd"
					/>
				</svg>
				{error}
			</span>
		</div>
	{/if}

	<!-- Help Text -->
	{#if help}
		<div class="label">
			<span class="label-text-alt text-gray-500">{help}</span>
		</div>
	{/if}
</div>

<style>
	.form-control {
		@apply mb-4;
	}
</style>
