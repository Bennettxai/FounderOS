<!--
	RoundedSelect.svelte
	iOS-style rounded select dropdown with glassmorphism
	Matches RoundedInput styling

	Usage:
	<RoundedSelect
		label="Role"
		bind:value={role}
		options={[
			{ value: 'founder', label: 'Founder' },
			{ value: 'consultant', label: 'Consultant' }
		]}
		placeholder="Select your role"
	/>
-->
<script lang="ts">
	import { ChevronDown } from 'lucide-svelte';

	interface Option {
		value: string;
		label: string;
	}

	interface Props {
		label?: string;
		value?: string;
		options: Option[];
		placeholder?: string;
		disabled?: boolean;
		required?: boolean;
		error?: string;
		helperText?: string;
		id?: string;
		name?: string;
		class?: string;
	}

	let {
		label,
		value = $bindable(''),
		options,
		placeholder = 'Select an option',
		disabled = false,
		required = false,
		error,
		helperText,
		id,
		name,
		class: className = ''
	}: Props = $props();

	const selectId = id || `select-${Math.random().toString(36).substr(2, 9)}`;
	const classes = `input-rounded select-rounded ${className}`.trim();
</script>

<div class="flex flex-col gap-1.5">
	{#if label}
		<label for={selectId} class="text-sm font-medium text-gray-700 dark:text-gray-300">
			{label}
			{#if required}
				<span class="text-red-500">*</span>
			{/if}
		</label>
	{/if}

	<div class="select-wrapper">
		<select
			id={selectId}
			{name}
			bind:value
			{disabled}
			{required}
			class={classes}
			class:border-red-500={error}
			class:text-gray-400={!value}
			aria-invalid={error ? 'true' : 'false'}
			aria-describedby={error ? `${selectId}-error` : helperText ? `${selectId}-helper` : undefined}
		>
			<option value="" disabled>{placeholder}</option>
			{#each options as option}
				<option value={option.value}>{option.label}</option>
			{/each}
		</select>
		<div class="select-icon">
			<ChevronDown size={20} />
		</div>
	</div>

	{#if error}
		<p id="{selectId}-error" class="text-sm text-red-600 dark:text-red-400 animate-slide-down">
			{error}
		</p>
	{:else if helperText}
		<p id="{selectId}-helper" class="text-sm text-gray-500 dark:text-gray-400">
			{helperText}
		</p>
	{/if}
</div>

<style>
	.select-wrapper {
		position: relative;
		width: 100%;
	}

	/* Select-specific overrides for the global input-rounded class */
	.select-rounded {
		padding-right: 2.5rem;
		cursor: pointer;
		appearance: none;
		-webkit-appearance: none;
		-moz-appearance: none;
	}

	.select-rounded.text-gray-400 {
		color: #9CA3AF;
	}

	.select-rounded.border-red-500 {
		border-color: #DC2626;
	}

	.select-icon {
		position: absolute;
		right: 0.875rem;
		top: 50%;
		transform: translateY(-50%);
		pointer-events: none;
		color: #6B7280;
	}

	:global(.dark) .select-icon {
		color: #9CA3AF;
	}
</style>
