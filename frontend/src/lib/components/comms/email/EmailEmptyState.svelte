<script lang="ts">
	import { Mail, AlertCircle, Search, Loader2, Inbox } from 'lucide-svelte';
	import PillButton from '$lib/components/ui/PillButton.svelte';

	export type EmptyVariant =
		| 'empty-folder'
		| 'no-match'
		| 'no-selection'
		| 'unsupported'
		| 'error'
		| 'loading';

	type IconKey = 'mail' | 'alert' | 'search' | 'loader' | 'inbox';

	interface Props {
		variant?: EmptyVariant;
		size?: 'sm' | 'md' | 'lg';
		title: string;
		description?: string;
		actionLabel?: string;
		onAction?: () => void;
		icon?: IconKey;
	}

	let {
		variant = 'empty-folder',
		size = 'md',
		title,
		description,
		actionLabel,
		onAction,
		icon,
	}: Props = $props();

	const iconKey: IconKey = $derived.by(() => {
		if (icon) return icon;
		if (variant === 'error') return 'alert';
		if (variant === 'no-match') return 'search';
		if (variant === 'loading') return 'loader';
		if (variant === 'no-selection') return 'mail';
		return 'inbox';
	});

	const iconSize = $derived(size === 'lg' ? 40 : size === 'sm' ? 22 : 28);

	const Icon = $derived.by(() => {
		switch (iconKey) {
			case 'alert':
				return AlertCircle;
			case 'search':
				return Search;
			case 'loader':
				return Loader2;
			case 'mail':
				return Mail;
			default:
				return Inbox;
		}
	});
</script>

<div
	class="cm-email-empty cm-email-empty--{size}"
	class:cm-email-empty--error={variant === 'error'}
	role={variant === 'error' ? 'alert' : 'status'}
>
	<span class="cm-email-empty__icon" aria-hidden="true">
		<Icon size={iconSize} strokeWidth={1.5} class={variant === 'loading' ? 'cm-spin' : ''} />
	</span>
	<h3 class="cm-email-empty__title">{title}</h3>
	{#if description}
		<p class="cm-email-empty__body">{description}</p>
	{/if}
	{#if actionLabel && onAction}
		<PillButton variant="soft" size="sm" onclick={onAction}>
			{actionLabel}
		</PillButton>
	{/if}
</div>

<style>
	.cm-email-empty {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: var(--space-3);
		padding: var(--space-12) var(--space-4);
		text-align: center;
		min-height: 0;
	}

	.cm-email-empty--sm {
		padding: var(--space-6) var(--space-4);
	}

	.cm-email-empty--lg {
		padding: var(--space-16) var(--space-4);
	}

	.cm-email-empty__icon {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		color: var(--dt3);
		width: 72px;
		height: 72px;
		border-radius: var(--radius-full);
		background: var(--dbg2);
		border: 1px solid var(--dbd2);
		margin-bottom: var(--space-2);
	}

	.cm-email-empty--sm .cm-email-empty__icon {
		width: 48px;
		height: 48px;
	}

	.cm-email-empty--lg .cm-email-empty__icon {
		width: 88px;
		height: 88px;
	}

	.cm-email-empty--error .cm-email-empty__icon {
		color: var(--bos-status-error-text);
		background: var(--bos-status-error-bg);
		border-color: var(--bos-status-error);
	}

	.cm-email-empty__title {
		margin: 0;
		font-size: var(--text-sm);
		font-weight: var(--font-semibold);
		color: var(--dt2);
	}

	.cm-email-empty--error .cm-email-empty__title {
		color: var(--dt);
	}

	.cm-email-empty__body {
		margin: 0;
		max-width: 280px;
		font-size: var(--text-xs);
		color: var(--dt3);
		line-height: 1.5;
	}
</style>
