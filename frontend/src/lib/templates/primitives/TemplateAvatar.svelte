<script lang="ts">
	/**
	 * TemplateAvatar - Avatar component with image or initials fallback
	 */

	type AvatarSize = 'xs' | 'sm' | 'md' | 'lg' | 'xl';
	type AvatarShape = 'circle' | 'square';

	interface Props {
		src?: string | null;
		alt?: string;
		name?: string;
		size?: AvatarSize;
		shape?: AvatarShape;
		status?: 'online' | 'offline' | 'busy' | 'away' | null;
	}

	let {
		src,
		alt = '',
		name = '',
		size = 'md',
		shape = 'circle',
		status = null
	}: Props = $props();

	let imageError = $state(false);

	const initials = $derived(() => {
		if (!name) return '?';
		const parts = name.trim().split(/\s+/);
		if (parts.length === 1) {
			return parts[0].charAt(0).toUpperCase();
		}
		return (parts[0].charAt(0) + parts[parts.length - 1].charAt(0)).toUpperCase();
	});

	const bgColor = $derived(() => {
		if (!name) return 'var(--tpl-bg-tertiary)';
		// Generate consistent color from name
		let hash = 0;
		for (let i = 0; i < name.length; i++) {
			hash = name.charCodeAt(i) + ((hash << 5) - hash);
		}
		const colors = [
			'#ef4444', '#f97316', '#f59e0b', '#84cc16', '#22c55e',
			'#14b8a6', '#06b6d4', '#3b82f6', '#6366f1', '#8b5cf6',
			'#a855f7', '#d946ef', '#ec4899', '#f43f5e'
		];
		return colors[Math.abs(hash) % colors.length];
	});

	function handleImageError() {
		imageError = true;
	}
</script>

<div class="tpl-avatar tpl-avatar-{size} tpl-avatar-{shape}">
	{#if src && !imageError}
		<img
			{src}
			alt={alt || name}
			class="tpl-avatar-image"
			onerror={handleImageError}
		/>
	{:else}
		<span
			class="tpl-avatar-initials"
			style="background-color: {bgColor()}"
		>
			{initials()}
		</span>
	{/if}
	{#if status}
		<span class="tpl-avatar-status tpl-avatar-status-{status}"></span>
	{/if}
</div>

<style>
	.tpl-avatar {
		position: relative;
		display: inline-flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
		overflow: hidden;
		background: var(--tpl-bg-tertiary);
	}

	/* ─────────────────────────────────────────────────────────────────────────
	   SIZES
	   ───────────────────────────────────────────────────────────────────────── */
	.tpl-avatar-xs {
		width: 24px;
		height: 24px;
		font-size: 10px;
	}

	.tpl-avatar-sm {
		width: 32px;
		height: 32px;
		font-size: 12px;
	}

	.tpl-avatar-md {
		width: 40px;
		height: 40px;
		font-size: 14px;
	}

	.tpl-avatar-lg {
		width: 48px;
		height: 48px;
		font-size: 18px;
	}

	.tpl-avatar-xl {
		width: 64px;
		height: 64px;
		font-size: 24px;
	}

	/* ─────────────────────────────────────────────────────────────────────────
	   SHAPES
	   ───────────────────────────────────────────────────────────────────────── */
	.tpl-avatar-circle {
		border-radius: 50%;
	}

	.tpl-avatar-square {
		border-radius: var(--tpl-radius-md);
	}

	/* ─────────────────────────────────────────────────────────────────────────
	   IMAGE & INITIALS
	   ───────────────────────────────────────────────────────────────────────── */
	.tpl-avatar-image {
		width: 100%;
		height: 100%;
		object-fit: cover;
	}

	.tpl-avatar-initials {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 100%;
		height: 100%;
		font-family: var(--tpl-font-sans);
		font-weight: var(--tpl-font-medium);
		color: white;
		text-transform: uppercase;
	}

	/* ─────────────────────────────────────────────────────────────────────────
	   STATUS INDICATOR
	   ───────────────────────────────────────────────────────────────────────── */
	.tpl-avatar-status {
		position: absolute;
		bottom: 0;
		right: 0;
		border: 2px solid var(--tpl-bg-primary);
		border-radius: 50%;
	}

	.tpl-avatar-xs .tpl-avatar-status {
		width: 8px;
		height: 8px;
		border-width: 1.5px;
	}

	.tpl-avatar-sm .tpl-avatar-status {
		width: 10px;
		height: 10px;
	}

	.tpl-avatar-md .tpl-avatar-status {
		width: 12px;
		height: 12px;
	}

	.tpl-avatar-lg .tpl-avatar-status {
		width: 14px;
		height: 14px;
	}

	.tpl-avatar-xl .tpl-avatar-status {
		width: 16px;
		height: 16px;
	}

	.tpl-avatar-status-online {
		background: var(--tpl-status-success);
	}

	.tpl-avatar-status-offline {
		background: var(--tpl-text-muted);
	}

	.tpl-avatar-status-busy {
		background: var(--tpl-status-error);
	}

	.tpl-avatar-status-away {
		background: var(--tpl-status-warning);
	}
</style>
