<script lang="ts">
	/**
	 * Switch Component - iOS-style toggle switch
	 * Extracted from BusinessOS2 toggle patterns
	 *
	 * Usage:
	 *   <Switch bind:checked={isEnabled} />
	 *   <Switch checked={darkMode} onchange={(v) => setDarkMode(v)} size="sm" />
	 *   <Switch checked disabled />
	 */

	type SwitchSize = 'sm' | 'md' | 'lg';

	interface Props {
		checked?: boolean;
		disabled?: boolean;
		size?: SwitchSize;
		label?: string;
		class?: string;
		onchange?: (checked: boolean) => void;
	}

	let {
		checked = $bindable(false),
		disabled = false,
		size = 'md',
		label,
		class: className = '',
		onchange
	}: Props = $props();

	function toggle() {
		if (disabled) return;
		checked = !checked;
		onchange?.(checked);
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === ' ' || e.key === 'Enter') {
			e.preventDefault();
			toggle();
		}
	}
</script>

<button
	type="button"
	role="switch"
	aria-checked={checked}
	aria-label={label}
	class="switch {className}"
	class:checked
	data-size={size}
	{disabled}
	onclick={toggle}
	onkeydown={handleKeydown}
>
	<span class="switch-thumb"></span>
</button>

<style>
	.switch {
		position: relative;
		display: inline-flex;
		flex-shrink: 0;
		cursor: pointer;
		border: none;
		border-radius: 9999px;
		background: #d1d5db;
		transition: background-color 0.2s ease;
		padding: 0;
	}

	.switch:focus-visible {
		outline: 2px solid #3b82f6;
		outline-offset: 2px;
	}

	.switch.checked {
		background: #111827;
	}

	.switch:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	/* Thumb */
	.switch-thumb {
		display: block;
		border-radius: 9999px;
		background: white;
		box-shadow: 0 1px 3px 0 rgba(0, 0, 0, 0.1);
		transition: transform 0.2s ease;
	}

	/* Size: sm */
	.switch[data-size='sm'] {
		width: 32px;
		height: 18px;
		padding: 2px;
	}

	.switch[data-size='sm'] .switch-thumb {
		width: 14px;
		height: 14px;
	}

	.switch[data-size='sm'].checked .switch-thumb {
		transform: translateX(14px);
	}

	/* Size: md (default) */
	.switch[data-size='md'] {
		width: 40px;
		height: 22px;
		padding: 2px;
	}

	.switch[data-size='md'] .switch-thumb {
		width: 18px;
		height: 18px;
	}

	.switch[data-size='md'].checked .switch-thumb {
		transform: translateX(18px);
	}

	/* Size: lg */
	.switch[data-size='lg'] {
		width: 48px;
		height: 26px;
		padding: 3px;
	}

	.switch[data-size='lg'] .switch-thumb {
		width: 20px;
		height: 20px;
	}

	.switch[data-size='lg'].checked .switch-thumb {
		transform: translateX(22px);
	}

	/* Dark mode */
	:global(.dark) .switch {
		background: #4b5563;
	}

	:global(.dark) .switch.checked {
		background: #3b82f6;
	}

	:global(.dark) .switch-thumb {
		background: #f3f4f6;
	}
</style>
