<script lang="ts">
	import {
		MessageSquare,
		Hash,
		Lock,
		AlertCircle,
		Loader2,
		Search,
		Plug,
	} from 'lucide-svelte';
	import PillButton from '$lib/components/ui/PillButton.svelte';

	export type EmptyVariant =
		| 'not-connected'
		| 'no-channels'
		| 'no-channel-selected'
		| 'no-messages'
		| 'no-match'
		| 'loading'
		| 'error';

	type IconKey =
		| 'message'
		| 'hash'
		| 'lock'
		| 'alert'
		| 'loader'
		| 'search'
		| 'plug';

	interface Props {
		variant?: EmptyVariant;
		size?: 'sm' | 'md' | 'lg';
		title: string;
		description?: string;
		actionLabel?: string;
		onAction?: () => void;
		secondaryActionLabel?: string;
		onSecondaryAction?: () => void;
		icon?: IconKey;
	}

	let {
		variant = 'no-messages',
		size = 'md',
		title,
		description,
		actionLabel,
		onAction,
		secondaryActionLabel,
		onSecondaryAction,
		icon,
	}: Props = $props();

	const iconKey: IconKey = $derived.by(() => {
		if (icon) return icon;
		if (variant === 'error') return 'alert';
		if (variant === 'loading') return 'loader';
		if (variant === 'not-connected') return 'plug';
		if (variant === 'no-match') return 'search';
		if (variant === 'no-channel-selected') return 'message';
		if (variant === 'no-channels') return 'hash';
		return 'message';
	});

	const iconSize = $derived(size === 'lg' ? 40 : size === 'sm' ? 22 : 28);

	const Icon = $derived.by(() => {
		switch (iconKey) {
			case 'alert':
				return AlertCircle;
			case 'loader':
				return Loader2;
			case 'plug':
				return Plug;
			case 'search':
				return Search;
			case 'hash':
				return Hash;
			case 'lock':
				return Lock;
			case 'message':
			default:
				return MessageSquare;
		}
	});
</script>

<div
	class="cm-channels-empty cm-channels-empty--{size}"
	class:cm-channels-empty--error={variant === 'error'}
	role={variant === 'error' ? 'alert' : 'status'}
>
	<span class="cm-channels-empty__icon" aria-hidden="true">
		<Icon
			size={iconSize}
			strokeWidth={1.5}
			class={variant === 'loading' ? 'cm-spin' : ''}
		/>
	</span>
	<h3 class="cm-channels-empty__title">{title}</h3>
	{#if description}
		<p class="cm-channels-empty__body">{description}</p>
	{/if}
	{#if actionLabel && onAction}
		<div class="cm-channels-empty__actions">
			<PillButton
				variant={variant === 'not-connected' ? 'cta' : 'soft'}
				size="sm"
				onclick={onAction}
			>
				{actionLabel}
			</PillButton>
			{#if secondaryActionLabel && onSecondaryAction}
				<PillButton variant="ghost" size="sm" onclick={onSecondaryAction}>
					{secondaryActionLabel}
				</PillButton>
			{/if}
		</div>
	{/if}
</div>

<style>
	.cm-channels-empty {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: var(--space-3);
		padding: var(--space-12) var(--space-4);
		text-align: center;
		min-height: 0;
	}

	.cm-channels-empty--sm {
		padding: var(--space-6) var(--space-4);
	}

	.cm-channels-empty--lg {
		padding: var(--space-16) var(--space-4);
	}

	.cm-channels-empty__icon {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		color: var(--dt3);
	}

	.cm-channels-empty--error .cm-channels-empty__icon {
		color: var(--bos-status-error-text);
	}

	.cm-channels-empty__title {
		margin: 0;
		font-size: var(--text-sm);
		font-weight: var(--font-semibold);
		color: var(--dt2);
	}

	.cm-channels-empty--error .cm-channels-empty__title {
		color: var(--dt);
	}

	.cm-channels-empty__body {
		margin: 0;
		max-width: 320px;
		font-size: var(--text-xs);
		color: var(--dt3);
		line-height: 1.5;
	}

	.cm-channels-empty__actions {
		display: flex;
		gap: var(--space-2);
	}
</style>
