<script lang="ts">
	/**
	 * MenuItem Component - BusinessOS Style
	 * Modern document-centric menu item patterns (30px height, hover overlay)
	 */
	import { DropdownMenu as MenuPrimitive } from 'bits-ui';
	import { type Snippet } from 'svelte';

	interface Props {
		disabled?: boolean;
		destructive?: boolean;
		onSelect?: () => void;
		class?: string;
		prefix?: Snippet;
		suffix?: Snippet;
		shortcut?: string;
		children: Snippet;
	}

	let {
		disabled = false,
		destructive = false,
		onSelect,
		class: className = '',
		prefix,
		suffix,
		shortcut,
		children
	}: Props = $props();
</script>

<MenuPrimitive.Item
	{disabled}
	onSelect={onSelect}
	class="bos-menu-item {className}"
	data-destructive={destructive || undefined}
	data-disabled={disabled || undefined}
>
	{#if prefix}
		<span class="bos-menu-item__icon">
			{@render prefix()}
		</span>
	{/if}

	<span class="bos-menu-item__content">
		{@render children()}
	</span>

	{#if shortcut}
		<span class="bos-menu-item__shortcut">
			{shortcut}
		</span>
	{/if}

	{#if suffix}
		<span class="bos-menu-item__icon">
			{@render suffix()}
		</span>
	{/if}
</MenuPrimitive.Item>

<style>
	:global(.bos-menu-item) {
		display: flex;
		align-items: center;
		gap: 8px;
		width: 100%;
		min-height: 30px;
		padding: 4px 12px;
		border-radius: 4px;
		font-size: 14px;
		color: var(--dt);
		background: transparent;
		cursor: pointer;
		user-select: none;
		outline: none;
		transition: background-color 0.15s;
	}

	:global(.bos-menu-item:hover),
	:global(.bos-menu-item:focus) {
		background: var(--dbg2);
	}

	:global(.bos-menu-item[data-disabled]) {
		opacity: 0.5;
		pointer-events: none;
	}

	:global(.bos-menu-item[data-destructive]) {
		color: var(--color-error, #eb4335);
	}

	:global(.bos-menu-item[data-destructive]:hover) {
		background: rgba(235, 67, 53, 0.1);
	}

	:global(.bos-menu-item__icon) {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 16px;
		height: 16px;
		flex-shrink: 0;
		color: var(--dt3);
	}

	:global(.bos-menu-item__icon svg) {
		width: 16px;
		height: 16px;
	}

	:global(.bos-menu-item[data-destructive] .bos-menu-item__icon) {
		color: var(--color-error, #eb4335);
	}

	:global(.bos-menu-item__content) {
		flex: 1;
		text-overflow: ellipsis;
		white-space: nowrap;
		overflow: hidden;
	}

	:global(.bos-menu-item__shortcut) {
		margin-left: auto;
		font-size: 12px;
		color: var(--dt3);
	}

	/* Dark mode */
	:global(.dark .bos-menu-item:hover),
	:global(.dark .bos-menu-item:focus) {
		background: var(--dbg3);
	}
</style>
