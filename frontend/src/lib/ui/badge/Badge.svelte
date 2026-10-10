<script lang="ts">
	/**
	 * Badge Component - Foundation Design System
	 * Status & priority badges using Foundation tokens.
	 *
	 * Usage:
	 *   <Badge variant="success">Active</Badge>
	 *   <Badge variant="warning" size="sm">Pending</Badge>
	 *   <Badge variant="danger" dot>Critical</Badge>
	 */
	import { type Snippet } from 'svelte';

	type BadgeVariant =
		| 'default'
		| 'primary'
		| 'success'
		| 'warning'
		| 'danger'
		| 'info'
		| 'purple'
		| 'outline';
	type BadgeSize = 'sm' | 'md';

	interface Props {
		variant?: BadgeVariant;
		size?: BadgeSize;
		dot?: boolean;
		class?: string;
		children?: Snippet;
	}

	let { variant = 'default', size = 'sm', dot = false, class: className = '', children }: Props =
		$props();
</script>

<span
	class="bos-badge bos-badge--{variant} bos-badge--{size} {className}"
>
	{#if dot}
		<span class="bos-badge__dot bos-badge__dot--{variant}"></span>
	{/if}
	{#if children}
		{@render children()}
	{/if}
</span>

<style>
	.bos-badge {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		border-radius: 9999px;
		font-weight: 500;
		white-space: nowrap;
	}

	.bos-badge--sm { padding: 2px 8px; font-size: 12px; }
	.bos-badge--md { padding: 4px 10px; font-size: 14px; }

	/* Variant colors */
	.bos-badge--default { background: var(--dbg2, #f5f5f5); color: var(--dt2, #555); }
	.bos-badge--primary { background: var(--dt, #111); color: #fff; }
	.bos-badge--success { background: rgba(5, 150, 105, 0.1); color: #059669; }
	.bos-badge--warning { background: rgba(217, 119, 6, 0.1); color: #d97706; }
	.bos-badge--danger { background: rgba(220, 38, 38, 0.1); color: #dc2626; }
	.bos-badge--info { background: rgba(59, 130, 246, 0.1); color: #3b82f6; }
	.bos-badge--purple { background: rgba(168, 85, 247, 0.1); color: #a855f7; }
	.bos-badge--outline {
		background: transparent;
		color: var(--dt2, #555);
		border: 1px solid var(--dbd, #e0e0e0);
	}

	/* Dot indicator */
	.bos-badge__dot {
		width: 6px;
		height: 6px;
		border-radius: 50%;
		flex-shrink: 0;
	}

	.bos-badge__dot--default { background: var(--dt4, #bbb); }
	.bos-badge__dot--primary { background: #fff; }
	.bos-badge__dot--success { background: #059669; }
	.bos-badge__dot--warning { background: #d97706; }
	.bos-badge__dot--danger { background: #dc2626; }
	.bos-badge__dot--info { background: #3b82f6; }
	.bos-badge__dot--purple { background: #a855f7; }
	.bos-badge__dot--outline { background: var(--dt4, #bbb); }

	/* Dark mode — Foundation tokens handle text/bg, status colors stay vibrant */
	:global(.dark) .bos-badge--default {
		background: rgba(255, 255, 255, 0.08);
		color: var(--dt2, #aaa);
	}
	:global(.dark) .bos-badge--primary {
		background: rgba(255, 255, 255, 0.9);
		color: #111;
	}
	:global(.dark) .bos-badge--outline {
		border-color: var(--dbd, #333);
		color: var(--dt2, #aaa);
	}
</style>
