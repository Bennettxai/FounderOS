<script lang="ts">
	/**
	 * SidebarItem Component - Navigation item for sidebars
	 * Extracted from BusinessOS2 KBSidebar, TablesSidebar patterns
	 *
	 * This component replaces the common pattern of:
	 *   class="w-full flex items-center gap-3 px-2.5 py-1.5 text-sm
	 *          text-gray-600 hover:bg-gray-100 rounded-md transition-colors"
	 *
	 * Usage:
	 *   <SidebarItem selected={isActive} onclick={handleClick}>
	 *     {#snippet prefix()}<Home class="h-4 w-4" />{/snippet}
	 *     Dashboard
	 *   </SidebarItem>
	 *
	 *   <SidebarItem variant="section">SECTION HEADER</SidebarItem>
	 */
	import { type Snippet } from 'svelte';

	type SidebarItemVariant = 'default' | 'section' | 'compact';

	interface Props {
		variant?: SidebarItemVariant;
		selected?: boolean;
		disabled?: boolean;
		depth?: number;
		class?: string;
		onclick?: (e: MouseEvent) => void;
		prefix?: Snippet;
		suffix?: Snippet;
		children?: Snippet;
	}

	let {
		variant = 'default',
		selected = false,
		disabled = false,
		depth = 0,
		class: className = '',
		onclick,
		prefix,
		suffix,
		children
	}: Props = $props();
</script>

{#if variant === 'section'}
	<div
		class="sidebar-section {className}"
		style={depth > 0 ? `padding-left: ${depth * 12 + 8}px` : undefined}
	>
		{#if children}
			{@render children()}
		{/if}
	</div>
{:else}
	<button
		type="button"
		class="sidebar-item {className}"
		class:selected
		class:compact={variant === 'compact'}
		{disabled}
		style={depth > 0 ? `padding-left: ${depth * 12 + 10}px` : undefined}
		{onclick}
	>
		{#if prefix}
			<span class="sidebar-item-icon">
				{@render prefix()}
			</span>
		{/if}

		<span class="sidebar-item-label">
			{#if children}
				{@render children()}
			{/if}
		</span>

		{#if suffix}
			<span class="sidebar-item-suffix">
				{@render suffix()}
			</span>
		{/if}
	</button>
{/if}

<style>
	.sidebar-item {
		display: flex;
		align-items: center;
		gap: 0.625rem;
		width: 100%;
		padding: 0.375rem 0.625rem;
		font-size: 0.875rem;
		line-height: 1.25rem;
		color: #4b5563;
		background: transparent;
		border: none;
		border-radius: 0.375rem;
		cursor: pointer;
		transition: all 0.15s ease;
		text-align: left;
	}

	.sidebar-item:hover {
		background: #f3f4f6;
		color: #1f2937;
	}

	.sidebar-item.selected {
		background: #f3f4f6;
		color: #111827;
		font-weight: 500;
	}

	.sidebar-item.compact {
		padding: 0.25rem 0.5rem;
		font-size: 0.8125rem;
		gap: 0.5rem;
	}

	.sidebar-item:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.sidebar-item-icon {
		flex-shrink: 0;
		display: flex;
		align-items: center;
		color: #9ca3af;
	}

	.sidebar-item.selected .sidebar-item-icon {
		color: #6b7280;
	}

	.sidebar-item-label {
		flex: 1;
		min-width: 0;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.sidebar-item-suffix {
		flex-shrink: 0;
		display: flex;
		align-items: center;
		color: #9ca3af;
		font-size: 0.75rem;
	}

	/* Section header */
	.sidebar-section {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		padding: 0.375rem 0.5rem;
		font-size: 0.6875rem;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		color: #9ca3af;
	}

	/* Dark mode */
	:global(.dark) .sidebar-item {
		color: #9ca3af;
	}

	:global(.dark) .sidebar-item:hover {
		background: #374151;
		color: #e5e7eb;
	}

	:global(.dark) .sidebar-item.selected {
		background: #374151;
		color: #f3f4f6;
	}

	:global(.dark) .sidebar-item-icon {
		color: #6b7280;
	}

	:global(.dark) .sidebar-item.selected .sidebar-item-icon {
		color: #9ca3af;
	}

	:global(.dark) .sidebar-section {
		color: #6b7280;
	}
</style>
