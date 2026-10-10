<script lang="ts">
	/**
	 * TemplateSwitch - Toggle switch component
	 */

	type SwitchSize = 'sm' | 'md' | 'lg';

	interface Props {
		checked?: boolean;
		size?: SwitchSize;
		disabled?: boolean;
		label?: string;
		description?: string;
		name?: string;
		id?: string;
		onchange?: (checked: boolean) => void;
	}

	let {
		checked = $bindable(false),
		size = 'md',
		disabled = false,
		label,
		description,
		name,
		id,
		onchange
	}: Props = $props();

	function handleClick() {
		if (!disabled) {
			checked = !checked;
			onchange?.(checked);
		}
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === ' ' || e.key === 'Enter') {
			e.preventDefault();
			handleClick();
		}
	}
</script>

<label class="tpl-switch-wrapper" class:tpl-switch-disabled={disabled}>
	<button
		type="button"
		role="switch"
		aria-checked={checked}
		{disabled}
		{name}
		{id}
		class="tpl-switch tpl-switch- {size}"
		class:tpl-switch-checked={checked}
		onclick={handleClick}
		onkeydown={handleKeydown}
	>
		<span class="tpl-switch-thumb"></span>
	</button>
	{#if label || description}
		<div class="tpl-switch-content">
			{#if label}
				<span class="tpl-switch-label">{label}</span>
			{/if}
			{#if description}
				<span class="tpl-switch-description">{description}</span>
			{/if}
		</div>
	{/if}
</label>

<style>
	.tpl-switch-wrapper {
		display: inline-flex;
		align-items: flex-start;
		gap: var(--tpl-space-3);
		cursor: pointer;
	}

	.tpl-switch-disabled {
		cursor: not-allowed;
		opacity: 0.5;
	}

	.tpl-switch {
		position: relative;
		display: inline-flex;
		align-items: center;
		flex-shrink: 0;
		padding: 2px;
		background: var(--tpl-bg-tertiary);
		border: none;
		border-radius: var(--tpl-radius-full);
		cursor: pointer;
		transition: background-color var(--tpl-transition-fast);
	}

	.tpl-switch:focus-visible {
		outline: none;
		box-shadow: var(--tpl-shadow-focus);
	}

	.tpl-switch:disabled {
		cursor: not-allowed;
	}

	.tpl-switch-checked {
		background: var(--tpl-accent-primary);
	}

	/* ─────────────────────────────────────────────────────────────────────────
	   SIZES
	   ───────────────────────────────────────────────────────────────────────── */
	.tpl-switch-sm {
		width: 32px;
		height: 18px;
	}

	.tpl-switch-sm .tpl-switch-thumb {
		width: 14px;
		height: 14px;
	}

	.tpl-switch-sm.tpl-switch-checked .tpl-switch-thumb {
		transform: translateX(14px);
	}

	.tpl-switch-md {
		width: 40px;
		height: 22px;
	}

	.tpl-switch-md .tpl-switch-thumb {
		width: 18px;
		height: 18px;
	}

	.tpl-switch-md.tpl-switch-checked .tpl-switch-thumb {
		transform: translateX(18px);
	}

	.tpl-switch-lg {
		width: 48px;
		height: 26px;
	}

	.tpl-switch-lg .tpl-switch-thumb {
		width: 22px;
		height: 22px;
	}

	.tpl-switch-lg.tpl-switch-checked .tpl-switch-thumb {
		transform: translateX(22px);
	}

	/* ─────────────────────────────────────────────────────────────────────────
	   THUMB
	   ───────────────────────────────────────────────────────────────────────── */
	.tpl-switch-thumb {
		background: white;
		border-radius: 50%;
		box-shadow: var(--tpl-shadow-sm);
		transition: transform var(--tpl-transition-fast);
	}

	/* ─────────────────────────────────────────────────────────────────────────
	   LABEL & DESCRIPTION
	   ───────────────────────────────────────────────────────────────────────── */
	.tpl-switch-content {
		display: flex;
		flex-direction: column;
		gap: var(--tpl-space-0-5);
	}

	.tpl-switch-label {
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-sm);
		font-weight: var(--tpl-font-medium);
		color: var(--tpl-text-primary);
		line-height: var(--tpl-leading-snug);
	}

	.tpl-switch-description {
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-xs);
		color: var(--tpl-text-muted);
		line-height: var(--tpl-leading-normal);
	}
</style>
