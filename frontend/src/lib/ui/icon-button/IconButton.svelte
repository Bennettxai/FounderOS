<script lang="ts">
	/**
	 * IconButton Component - Small icon-only buttons for toolbars & actions
	 * Extracted from BusinessOS2 toolbar patterns (KBSidebar, SandboxWindow, etc.)
	 *
	 * This component replaces raw Tailwind toolbar icon buttons and prevents
	 * the btn-pill contamination issue (btn-pill is for pill-shaped action
	 * buttons, NOT for small toolbar icons).
	 *
	 * Usage:
	 *   <IconButton onclick={handleClick}><X class="h-4 w-4" /></IconButton>
	 *   <IconButton size="lg" variant="ghost"><Settings class="h-5 w-5" /></IconButton>
	 *   <IconButton size="xs" label="Close"><X class="h-3 w-3" /></IconButton>
	 */
	import { type Snippet } from 'svelte';
	import type { HTMLButtonAttributes } from 'svelte/elements';

	type IconButtonSize = 'xs' | 'sm' | 'md' | 'lg';
	type IconButtonVariant = 'ghost' | 'subtle' | 'outline';

	interface Props extends Omit<HTMLButtonAttributes, 'class'> {
		size?: IconButtonSize;
		variant?: IconButtonVariant;
		label?: string;
		active?: boolean;
		class?: string;
		children?: Snippet;
	}

	let {
		size = 'md',
		variant = 'ghost',
		label,
		active = false,
		class: className = '',
		children,
		...restProps
	}: Props = $props();

	const sizeClasses: Record<IconButtonSize, string> = {
		xs: 'w-6 h-6 rounded',
		sm: 'w-7 h-7 rounded-md',
		md: 'w-8 h-8 rounded-lg',
		lg: 'w-9 h-9 rounded-lg'
	};
</script>

<button
	type="button"
	class="icon-btn {sizeClasses[size]} {className}"
	class:active
	data-variant={variant}
	aria-label={label}
	{...restProps}
>
	{#if children}
		{@render children()}
	{/if}
</button>

<style>
	.icon-btn {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
		border: none;
		cursor: pointer;
		transition: all 0.15s ease;
		color: #9ca3af;
	}

	/* Ghost - most common, transparent bg */
	.icon-btn[data-variant='ghost'] {
		background: transparent;
	}

	.icon-btn[data-variant='ghost']:hover {
		background: #f3f4f6;
		color: #374151;
	}

	/* Subtle - light bg always visible */
	.icon-btn[data-variant='subtle'] {
		background: #f3f4f6;
		color: #6b7280;
	}

	.icon-btn[data-variant='subtle']:hover {
		background: #e5e7eb;
		color: #374151;
	}

	/* Outline - with border */
	.icon-btn[data-variant='outline'] {
		background: transparent;
		border: 1px solid #e5e7eb;
		color: #6b7280;
	}

	.icon-btn[data-variant='outline']:hover {
		background: #f9fafb;
		border-color: #d1d5db;
		color: #374151;
	}

	/* Active state */
	.icon-btn.active {
		background: #f3f4f6;
		color: #111827;
	}

	/* Dark mode */
	:global(.dark) .icon-btn {
		color: #9ca3af;
	}

	:global(.dark) .icon-btn[data-variant='ghost']:hover {
		background: #374151;
		color: #e5e7eb;
	}

	:global(.dark) .icon-btn[data-variant='subtle'] {
		background: #374151;
		color: #9ca3af;
	}

	:global(.dark) .icon-btn[data-variant='subtle']:hover {
		background: #4b5563;
		color: #e5e7eb;
	}

	:global(.dark) .icon-btn[data-variant='outline'] {
		border-color: #4b5563;
		color: #9ca3af;
	}

	:global(.dark) .icon-btn[data-variant='outline']:hover {
		background: #374151;
		border-color: #6b7280;
		color: #e5e7eb;
	}

	:global(.dark) .icon-btn.active {
		background: #374151;
		color: #f3f4f6;
	}
</style>
