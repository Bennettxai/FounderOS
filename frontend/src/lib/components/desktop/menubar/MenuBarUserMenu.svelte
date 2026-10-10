<script lang="ts">
	import { useSession } from '$lib/auth-client';
	import { themeStore } from '$lib/stores/themeStore';

	interface Props {
		isOpen: boolean;
		onToggle: () => void;
		onClose: () => void;
		onAction: (action: string) => void;
	}

	let { isOpen, onToggle, onClose, onAction }: Props = $props();

	const session = useSession();
	const themeMode = $derived($themeStore.theme);

	function setThemeMode(mode: 'light' | 'dark' | 'system') {
		themeStore.setTheme(mode);
	}
</script>

<!-- User avatar -->
<button
	class="menu-bar-avatar"
	onclick={onToggle}
	aria-label="User menu"
	aria-haspopup="menu"
	aria-expanded={isOpen}
>
	<span class="avatar-initials">
		{$session.data?.user?.name?.charAt(0).toUpperCase() || 'U'}
	</span>
</button>

{#if isOpen}
	<div class="menu-dropdown user-menu" role="menu">
		<div class="menu-user-info">
			<span class="menu-user-name">{$session.data?.user?.name || 'User'}</span>
			<span class="menu-user-email">{$session.data?.user?.email || ''}</span>
		</div>

		<div class="menu-separator" role="separator"></div>

		<div class="menu-section-label" aria-hidden="true">Appearance</div>
		<div class="theme-toggle-group" role="group" aria-label="Color theme">
			<button
				class="theme-toggle-btn"
				class:active={themeMode === 'light'}
				onclick={() => setThemeMode('light')}
				aria-pressed={themeMode === 'light'}
			>
				<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
					<circle cx="12" cy="12" r="5"/>
					<path d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42"/>
				</svg>
				<span>Light</span>
			</button>
			<button
				class="theme-toggle-btn"
				class:active={themeMode === 'dark'}
				onclick={() => setThemeMode('dark')}
				aria-pressed={themeMode === 'dark'}
			>
				<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
					<path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/>
				</svg>
				<span>Dark</span>
			</button>
			<button
				class="theme-toggle-btn"
				class:active={themeMode === 'system'}
				onclick={() => setThemeMode('system')}
				aria-pressed={themeMode === 'system'}
			>
				<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
					<rect x="2" y="3" width="20" height="14" rx="2"/>
					<path d="M8 21h8M12 17v4"/>
				</svg>
				<span>Auto</span>
			</button>
		</div>

		<div class="menu-separator" role="separator"></div>

		<button class="menu-item" role="menuitem" onclick={() => { onClose(); onAction('profile'); }}>
			<span class="menu-item-check"></span>
			<span class="menu-item-label">Profile</span>
		</button>
		<button class="menu-item" role="menuitem" onclick={() => { onClose(); onAction('settings'); }}>
			<span class="menu-item-check"></span>
			<span class="menu-item-label">Settings</span>
		</button>

		<div class="menu-separator" role="separator"></div>

		<button class="menu-item" role="menuitem" onclick={() => onAction('logout')}>
			<span class="menu-item-check"></span>
			<span class="menu-item-label">Log Out</span>
		</button>
	</div>
{/if}

<style>
	.menu-bar-avatar {
		width: 20px;
		height: 20px;
		border-radius: 50%;
		background: #333;
		border: none;
		cursor: pointer;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.avatar-initials {
		color: white;
		font-size: 10px;
		font-weight: 600;
	}

	.menu-user-info {
		padding: 8px 12px;
		display: flex;
		flex-direction: column;
		gap: 2px;
	}

	.menu-user-name {
		font-weight: 600;
		color: #111;
	}

	.menu-user-email {
		font-size: 11px;
		color: #666;
	}

	.menu-section-label {
		padding: 4px 12px 2px;
		font-size: 10px;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.5px;
		color: #999;
	}

	.theme-toggle-group {
		display: flex;
		gap: 4px;
		padding: 6px 8px;
	}

	.theme-toggle-btn {
		flex: 1;
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 4px;
		padding: 8px 4px;
		border: 1px solid rgba(0, 0, 0, 0.1);
		background: transparent;
		border-radius: 8px;
		cursor: pointer;
		font-size: 10px;
		color: #666;
		transition: all 0.15s;
	}

	.theme-toggle-btn svg {
		width: 16px;
		height: 16px;
	}

	.theme-toggle-btn:hover {
		background: rgba(0, 0, 0, 0.05);
		border-color: rgba(0, 0, 0, 0.15);
	}

	.theme-toggle-btn.active {
		background: #0066FF;
		border-color: #0066FF;
		color: white;
	}

	/* Dark mode */
	:global(.dark) .menu-bar-avatar {
		background: #48484a;
	}

	:global(.dark) .menu-user-name {
		color: #f5f5f7;
	}

	:global(.dark) .menu-user-email {
		color: #a1a1a6;
	}

	:global(.dark) .menu-section-label {
		color: #6e6e73;
	}

	:global(.dark) .theme-toggle-btn {
		border-color: rgba(255, 255, 255, 0.12);
		color: #a1a1a6;
	}

	:global(.dark) .theme-toggle-btn:hover {
		background: rgba(255, 255, 255, 0.08);
		border-color: rgba(255, 255, 255, 0.2);
	}

	:global(.dark) .theme-toggle-btn.active {
		background: #0A84FF;
		border-color: #0A84FF;
		color: white;
	}
</style>
