<script lang="ts">
	/**
	 * UserCell - Display user with avatar and name
	 */

	interface User {
		id: string;
		name: string;
		email?: string;
		avatar?: string;
	}

	interface Props {
		value: User | User[] | null | undefined;
		showEmail?: boolean;
		size?: 'sm' | 'md' | 'lg';
		max?: number;
	}

	let {
		value,
		showEmail = false,
		size = 'md',
		max = 3
	}: Props = $props();

	const users = $derived(() => {
		if (!value) return [];
		return Array.isArray(value) ? value : [value];
	});

	const displayUsers = $derived(() => {
		const u = users();
		return u.slice(0, max);
	});

	const remaining = $derived(() => {
		const u = users();
		return Math.max(0, u.length - max);
	});

	const sizes: Record<string, { avatar: string; text: string }> = {
		sm: { avatar: '24px', text: 'var(--tpl-text-xs)' },
		md: { avatar: '32px', text: 'var(--tpl-text-sm)' },
		lg: { avatar: '40px', text: 'var(--tpl-text-base)' }
	};

	function getInitials(name: string): string {
		return name
			.split(' ')
			.map((n) => n[0])
			.slice(0, 2)
			.join('')
			.toUpperCase();
	}

	function getAvatarColor(name: string): string {
		const colors = [
			'#f87171', '#fb923c', '#fbbf24', '#a3e635',
			'#34d399', '#22d3d8', '#60a5fa', '#a78bfa',
			'#f472b6', '#94a3b8'
		];
		const hash = name.split('').reduce((acc, char) => acc + char.charCodeAt(0), 0);
		return colors[hash % colors.length];
	}
</script>

<div class="tpl-user-cell" style="--avatar-size: {sizes[size].avatar}; --text-size: {sizes[size].text}">
	{#if displayUsers().length === 0}
		<span class="tpl-user-empty">—</span>
	{:else if displayUsers().length === 1}
		{@const user = displayUsers()[0]}
		<div class="tpl-user-single">
			{#if user.avatar}
				<img src={user.avatar} alt={user.name} class="tpl-user-avatar" />
			{:else}
				<span class="tpl-user-avatar tpl-user-initials" style="background-color: {getAvatarColor(user.name)}">
					{getInitials(user.name)}
				</span>
			{/if}
			<div class="tpl-user-info">
				<span class="tpl-user-name">{user.name}</span>
				{#if showEmail && user.email}
					<span class="tpl-user-email">{user.email}</span>
				{/if}
			</div>
		</div>
	{:else}
		<div class="tpl-user-stack">
			{#each displayUsers() as user, i}
				{#if user.avatar}
					<img
						src={user.avatar}
						alt={user.name}
						class="tpl-user-avatar tpl-user-stacked"
						style="z-index: {displayUsers().length - i}"
						title={user.name}
					/>
				{:else}
					<span
						class="tpl-user-avatar tpl-user-initials tpl-user-stacked"
						style="background-color: {getAvatarColor(user.name)}; z-index: {displayUsers().length - i}"
						title={user.name}
					>
						{getInitials(user.name)}
					</span>
				{/if}
			{/each}
			{#if remaining() > 0}
				<span class="tpl-user-avatar tpl-user-more">
					+{remaining()}
				</span>
			{/if}
		</div>
	{/if}
</div>

<style>
	.tpl-user-cell {
		padding: var(--tpl-space-2) var(--tpl-space-3);
	}

	.tpl-user-empty {
		color: var(--tpl-text-muted);
		font-size: var(--text-size);
	}

	.tpl-user-single {
		display: flex;
		align-items: center;
		gap: var(--tpl-space-2);
	}

	.tpl-user-avatar {
		width: var(--avatar-size);
		height: var(--avatar-size);
		border-radius: 50%;
		object-fit: cover;
		flex-shrink: 0;
	}

	.tpl-user-initials {
		display: flex;
		align-items: center;
		justify-content: center;
		color: white;
		font-size: calc(var(--avatar-size) * 0.4);
		font-weight: var(--tpl-font-medium);
	}

	.tpl-user-info {
		display: flex;
		flex-direction: column;
		min-width: 0;
	}

	.tpl-user-name {
		font-family: var(--tpl-font-sans);
		font-size: var(--text-size);
		font-weight: var(--tpl-font-medium);
		color: var(--tpl-text-primary);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.tpl-user-email {
		font-size: var(--tpl-text-xs);
		color: var(--tpl-text-muted);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.tpl-user-stack {
		display: flex;
		align-items: center;
	}

	.tpl-user-stacked {
		margin-left: -8px;
		border: 2px solid var(--tpl-bg-primary);
	}

	.tpl-user-stacked:first-child {
		margin-left: 0;
	}

	.tpl-user-more {
		display: flex;
		align-items: center;
		justify-content: center;
		margin-left: -8px;
		background: var(--tpl-bg-tertiary);
		color: var(--tpl-text-secondary);
		font-size: calc(var(--avatar-size) * 0.35);
		font-weight: var(--tpl-font-medium);
		border: 2px solid var(--tpl-bg-primary);
	}
</style>
