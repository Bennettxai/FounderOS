<!--
	PillSelect.svelte
	Card/Pill-based selection component for onboarding
	Options displayed as tappable pills instead of dropdown

	Usage:
	<PillSelect
		label="Your Role"
		bind:value={role}
		options={[
			{ value: 'founder', label: 'Founder' },
			{ value: 'consultant', label: 'Consultant' }
		]}
	/>
-->
<script lang="ts">
	interface Option {
		value: string;
		label: string;
	}

	interface Props {
		label?: string;
		value?: string;
		options: Option[];
		error?: string;
		required?: boolean;
		columns?: 2 | 3 | 4; // Grid columns
		class?: string;
	}

	let {
		label,
		value = $bindable(''),
		options,
		error,
		required = false,
		columns = 3,
		class: className = ''
	}: Props = $props();

	function selectOption(optionValue: string) {
		value = optionValue;
	}

	function handleKeydown(e: KeyboardEvent, optionValue: string) {
		if (e.key === 'Enter' || e.key === ' ') {
			e.preventDefault();
			selectOption(optionValue);
		}
	}
</script>

<div class="pill-select-container {className}">
	{#if label}
		<label class="pill-select-label">
			{label}
			{#if required}
				<span class="text-red-500">*</span>
			{/if}
		</label>
	{/if}

	<div
		class="pill-select-grid"
		style="--columns: {columns}"
		role="radiogroup"
		aria-label={label}
	>
		{#each options as option}
			<button
				type="button"
				role="radio"
				aria-checked={value === option.value}
				class="pill-option"
				class:selected={value === option.value}
				onclick={() => selectOption(option.value)}
				onkeydown={(e) => handleKeydown(e, option.value)}
			>
				{option.label}
			</button>
		{/each}
	</div>

	{#if error}
		<p class="pill-select-error">{error}</p>
	{/if}
</div>

<style>
	.pill-select-container {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
		width: 100%;
	}

	.pill-select-label {
		font-size: 0.875rem;
		font-weight: 500;
		color: var(--bos-text-primary-color, #374151);
		text-align: left;
	}

	.pill-select-grid {
		display: grid;
		grid-template-columns: repeat(var(--columns), 1fr);
		gap: 0.5rem;
	}

	.pill-option {
		padding: 0.75rem 1rem;
		font-size: 0.875rem;
		font-weight: 500;
		color: var(--bos-text-secondary-color, #4B5563);
		background: var(--bos-v2-layer-background-secondary, rgba(255, 255, 255, 0.6));
		backdrop-filter: blur(8px);
		-webkit-backdrop-filter: blur(8px);
		border: 1.5px solid var(--bos-v2-layer-insideBorder-border, rgba(0, 0, 0, 0.08));
		border-radius: 0.75rem;
		cursor: pointer;
		transition: all 0.2s ease;
		font-family: inherit;
		text-align: center;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.pill-option:hover {
		background: var(--bos-v2-layer-background-primary, rgba(255, 255, 255, 0.9));
		border-color: var(--bos-border-color, #CCCCCC);
		transform: translateY(-1px);
	}

	.pill-option:focus {
		outline: none;
		box-shadow: 0 0 0 3px rgba(26, 26, 26, 0.1);
	}

	.pill-option.selected {
		background: var(--bos-brand-color, #1e96eb);
		color: var(--bos-btn-cta-text, white);
		border-color: var(--bos-brand-color, #1e96eb);
		box-shadow: var(--bos-btn-cta-glow, 0 2px 8px rgba(0, 0, 0, 0.15));
	}

	.pill-option.selected:hover {
		background: var(--bos-btn-cta-hover, #1a82d4);
		border-color: var(--bos-btn-cta-hover, #1a82d4);
		transform: translateY(-1px);
	}

	.pill-select-error {
		font-size: 0.75rem;
		color: var(--bos-error-color, #DC2626);
		margin: 0;
		text-align: left;
	}

	/* Responsive: 2 columns on mobile */
	@media (max-width: 480px) {
		.pill-select-grid {
			grid-template-columns: repeat(2, 1fr);
		}
	}
</style>
