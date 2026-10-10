<script lang="ts">
	/**
	 * TemplateTextarea - Multi-line text input component
	 */

	interface Props {
		value?: string;
		placeholder?: string;
		rows?: number;
		minRows?: number;
		maxRows?: number;
		disabled?: boolean;
		readonly?: boolean;
		error?: string;
		resize?: 'none' | 'vertical' | 'horizontal' | 'both';
		name?: string;
		id?: string;
		required?: boolean;
		maxlength?: number;
		showCount?: boolean;
		oninput?: (e: Event) => void;
		onchange?: (e: Event) => void;
		onblur?: (e: FocusEvent) => void;
		onfocus?: (e: FocusEvent) => void;
	}

	let {
		value = $bindable(''),
		placeholder = '',
		rows = 3,
		minRows,
		maxRows,
		disabled = false,
		readonly = false,
		error,
		resize = 'vertical',
		name,
		id,
		required = false,
		maxlength,
		showCount = false,
		oninput,
		onchange,
		onblur,
		onfocus
	}: Props = $props();

	const charCount = $derived(value?.length ?? 0);
</script>

<div class="tpl-textarea-wrapper" class:tpl-textarea-error={error} class:tpl-textarea-disabled={disabled}>
	<textarea
		{name}
		{id}
		{placeholder}
		{disabled}
		{required}
		{maxlength}
		readonly={readonly}
		{rows}
		bind:value
		class="tpl-textarea"
		style="resize: {resize}; {minRows ? `min-height: ${minRows * 1.5}em;` : ''} {maxRows ? `max-height: ${maxRows * 1.5}em;` : ''}"
		{oninput}
		{onchange}
		{onblur}
		{onfocus}
	></textarea>
	{#if showCount || maxlength}
		<div class="tpl-textarea-footer">
			<span class="tpl-textarea-count">
				{charCount}{#if maxlength}/{maxlength}{/if}
			</span>
		</div>
	{/if}
</div>
{#if error}
	<p class="tpl-textarea-error-text">{error}</p>
{/if}

<style>
	.tpl-textarea-wrapper {
		display: flex;
		flex-direction: column;
		background: var(--tpl-bg-primary);
		border: 1px solid var(--tpl-border-default);
		border-radius: var(--tpl-radius-md);
		transition: border-color var(--tpl-transition-fast),
		            box-shadow var(--tpl-transition-fast);
		overflow: hidden;
	}

	.tpl-textarea-wrapper:hover:not(.tpl-textarea-disabled) {
		border-color: var(--tpl-border-hover);
	}

	.tpl-textarea-wrapper:focus-within:not(.tpl-textarea-disabled) {
		border-color: var(--tpl-border-focus);
		box-shadow: var(--tpl-shadow-focus);
	}

	.tpl-textarea-wrapper.tpl-textarea-error {
		border-color: var(--tpl-status-error);
	}

	.tpl-textarea-wrapper.tpl-textarea-error:focus-within {
		box-shadow: var(--tpl-shadow-focus-error);
	}

	.tpl-textarea-wrapper.tpl-textarea-disabled {
		background: var(--tpl-bg-secondary);
		opacity: 0.6;
	}

	.tpl-textarea {
		width: 100%;
		min-height: 80px;
		padding: var(--tpl-space-3);
		background: transparent;
		border: none;
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-sm);
		color: var(--tpl-text-primary);
		line-height: var(--tpl-leading-normal);
		outline: none;
	}

	.tpl-textarea::placeholder {
		color: var(--tpl-text-placeholder);
	}

	.tpl-textarea:disabled {
		cursor: not-allowed;
		color: var(--tpl-text-muted);
	}

	.tpl-textarea-footer {
		display: flex;
		justify-content: flex-end;
		padding: var(--tpl-space-1-5) var(--tpl-space-3);
		border-top: 1px solid var(--tpl-border-subtle);
		background: var(--tpl-bg-secondary);
	}

	.tpl-textarea-count {
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-xs);
		color: var(--tpl-text-muted);
		font-variant-numeric: tabular-nums;
	}

	.tpl-textarea-error-text {
		margin: var(--tpl-space-1) 0 0;
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-xs);
		color: var(--tpl-status-error);
		line-height: var(--tpl-leading-snug);
	}
</style>
