<script lang="ts">
	interface MenuItem {
		type?: string;
		label?: string;
		shortcut?: string;
		action?: string;
		disabled?: boolean;
		checked?: boolean;
	}

	interface Menu {
		id: string;
		label: string;
		items: MenuItem[];
	}

	interface Props {
		menus: Menu[];
		activeMenu: string | null;
		onToggle: (menuId: string) => void;
		onAction: (action: string) => void;
		onWindowSelect: (windowId: string) => void;
	}

	let { menus, activeMenu, onToggle, onAction, onWindowSelect }: Props = $props();
</script>

{#each menus as menu}
	<div class="menu-bar-item-wrapper">
		<button
			class="menu-bar-item"
			class:active={activeMenu === menu.id}
			onclick={() => onToggle(menu.id)}
			onmouseenter={() => {
				if (activeMenu && activeMenu !== 'desktop' && activeMenu !== 'user') {
					onToggle(menu.id);
				}
			}}
			aria-haspopup="menu"
			aria-expanded={activeMenu === menu.id}
		>
			{menu.label}
		</button>

		{#if activeMenu === menu.id}
			<div class="menu-dropdown" role="menu">
				{#each menu.items as item}
					{#if item.type === 'separator'}
						<div class="menu-separator" role="separator"></div>
					{:else}
						<button
							class="menu-item"
							class:disabled={item.disabled}
							class:checked={item.checked}
							disabled={item.disabled}
							role="menuitem"
							aria-checked={item.checked}
							onclick={() => {
								if (item.action?.startsWith('window:')) {
									onWindowSelect(item.action.replace('window:', ''));
								} else if (item.action) {
									onAction(item.action);
								}
							}}
						>
							<span class="menu-item-check">
								{#if item.checked}
									<svg class="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" aria-hidden="true">
										<polyline points="20 6 9 17 4 12"></polyline>
									</svg>
								{/if}
							</span>
							<span class="menu-item-label">{item.label}</span>
							{#if item.shortcut}
								<span class="menu-item-shortcut">{item.shortcut}</span>
							{/if}
						</button>
					{/if}
				{/each}
			</div>
		{/if}
	</div>
{/each}

<style>
	.menu-bar-item-wrapper {
		position: relative;
	}

	.menu-bar-item {
		padding: 4px 10px;
		border-radius: 4px;
		background: none;
		border: none;
		cursor: pointer;
		font-size: 13px;
		font-weight: 500;
		color: #333;
	}

	.menu-bar-item:hover,
	.menu-bar-item.active {
		background: rgba(0, 0, 0, 0.08);
	}

	:global(.dark) .menu-bar-item {
		color: #f5f5f7;
	}

	:global(.dark) .menu-bar-item:hover,
	:global(.dark) .menu-bar-item.active {
		background: rgba(255, 255, 255, 0.1);
	}
</style>
