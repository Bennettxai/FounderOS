<script lang="ts">
	/**
	 * Avatar Component - Foundation Design System
	 * User avatar with initials fallback using Foundation tokens.
	 *
	 * Usage:
	 *   <Avatar name="Alex Morgan" src="/avatar.jpg" />
	 *   <Avatar name="Pedro" size="sm" />
	 *   <Avatar name="Javaris" size="lg" gradient />
	 */

	type AvatarSize = 'xs' | 'sm' | 'md' | 'lg' | 'xl';

	interface Props {
		name: string;
		src?: string | null;
		size?: AvatarSize;
		gradient?: boolean;
		class?: string;
	}

	let { name, src = null, size = 'md', gradient = false, class: className = '' }: Props =
		$props();

	function getInitials(fullName: string): string {
		return fullName
			.split(' ')
			.map((n) => n[0])
			.join('')
			.toUpperCase()
			.slice(0, 2);
	}

	const initials = $derived(getInitials(name));
	let imgError = $state(false);
</script>

{#if src && !imgError}
	<img
		{src}
		alt={name}
		class="bos-avatar bos-avatar--{size} {className}"
		onerror={() => (imgError = true)}
	/>
{:else}
	<div
		class="bos-avatar bos-avatar--{size} {gradient ? 'bos-avatar--gradient' : 'bos-avatar--initials'} {className}"
	>
		<span class="bos-avatar__text bos-avatar__text--{size}">{initials}</span>
	</div>
{/if}

<style>
	.bos-avatar {
		flex-shrink: 0;
		border-radius: 9999px;
		display: flex;
		align-items: center;
		justify-content: center;
		object-fit: cover;
	}

	.bos-avatar--xs { width: 24px; height: 24px; }
	.bos-avatar--sm { width: 32px; height: 32px; }
	.bos-avatar--md { width: 40px; height: 40px; }
	.bos-avatar--lg { width: 48px; height: 48px; }
	.bos-avatar--xl { width: 64px; height: 64px; }

	.bos-avatar--initials {
		background: var(--dbg2, #f5f5f5);
		color: var(--dt2, #555);
	}

	.bos-avatar--gradient {
		background: linear-gradient(135deg, #3b82f6, #a855f7);
		color: #fff;
	}

	.bos-avatar__text {
		font-weight: 600;
		line-height: 1;
	}

	.bos-avatar__text--xs { font-size: 10px; }
	.bos-avatar__text--sm { font-size: 12px; }
	.bos-avatar__text--md { font-size: 14px; }
	.bos-avatar__text--lg { font-size: 14px; }
	.bos-avatar__text--xl { font-size: 20px; }

	:global(.dark) .bos-avatar--initials {
		background: var(--dbg3, #2e2e2e);
		color: var(--dt2, #aaa);
	}
</style>
