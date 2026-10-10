<script lang="ts">
	import { windowStore, focusedWindow } from '$lib/stores/windowStore';
	import { desktopSettings } from '$lib/stores/desktopStore';
	import { currentWorkspace } from '$lib/stores/workspaces';
	import { deleteWorkspaceDesktopSpace, saveWorkspaceDesktopSpace } from '$lib/api/desktop-spaces';
	import { desktop3dLayoutStore, isEditMode, activeLayout } from '$lib/stores/desktop3dLayoutStore';
	import { signOut } from '$lib/auth-client';
	import { goto } from '$app/navigation';
	import { browser } from '$app/environment';
	import { get } from 'svelte/store';
	import { isElectron, isMacOS } from '$lib/utils/platform';
	import LayoutManager from '$lib/components/desktop3d/LayoutManager.svelte';
	import MenuBarDesktopMenu from '$lib/components/desktop/menubar/MenuBarDesktopMenu.svelte';
	import MenuBarMenus from '$lib/components/desktop/menubar/MenuBarMenus.svelte';
	import MenuBarClock from '$lib/components/desktop/menubar/MenuBarClock.svelte';
	import MenuBarUserMenu from '$lib/components/desktop/menubar/MenuBarUserMenu.svelte';
	import SaveLayoutModal from '$lib/components/desktop/menubar/SaveLayoutModal.svelte';

	// Electron / macOS detection
	const inElectron = $derived(browser && isElectron());
	const onMac = $derived(browser && isMacOS());
	const needsTrafficLightSpace = $derived(inElectron && onMac);

	// Menu open/close state
	let activeMenu: string | null = $state(null);

	// 3D Desktop layout management
	let showLayoutManager = $state(false);
	let showSaveLayoutModal = $state(false);
	let desktopModalMode = $state<'new' | 'duplicate' | 'rename' | 'delete' | null>(null);
	let desktopNameInput = $state('');
	let desktopDeleteConfirmation = $state('');
	let desktopKindInput = $state<'personal' | 'team' | 'workspace'>('workspace');
	let desktopModalError = $state('');

	function toggleMenu(menu: string) {
		activeMenu = activeMenu === menu ? null : menu;
	}

	function closeMenus() {
		activeMenu = null;
	}

	function handleMenuAction(action: string) {
		closeMenus();

		if (action.startsWith('desktop:switch:')) {
			windowStore.switchDesktopSpace(action.replace('desktop:switch:', ''));
			return;
		}

		switch (action) {
			case 'new-window':
				if ($focusedWindow) windowStore.openWindow($focusedWindow.module);
				break;
			case 'close-window':
				if ($focusedWindow) windowStore.closeWindow($focusedWindow.id);
				break;
			case 'close-all':
				$windowStore.windows.forEach(w => windowStore.closeWindow(w.id));
				break;
			case 'minimize':
				if ($focusedWindow) windowStore.minimizeWindow($focusedWindow.id);
				break;
			case 'maximize':
				if ($focusedWindow) windowStore.toggleMaximize($focusedWindow.id);
				break;
			case 'desktop-settings':
				windowStore.openWindow('desktop-settings');
				break;
			case 'profile':
				windowStore.openWindow('profile', { title: 'Profile' });
				break;
			case 'settings':
				windowStore.openWindow('settings', { title: 'Settings' });
				break;
			case 'desktop:open-collab':
				void openCollaborativeWorkspaceDesktop();
				break;
			case 'desktop:open-infinity':
				void openInfinityDesktop();
				break;
			case 'new-desktop': {
				desktopNameInput = `Desktop ${$windowStore.desktopSpaces.length + 1}`;
				desktopDeleteConfirmation = '';
				desktopKindInput = 'personal';
				desktopModalError = '';
				desktopModalMode = 'new';
				break;
			}
			case 'duplicate-desktop': {
				const current = $windowStore.desktopSpaces.find((space) => space.id === $windowStore.activeDesktopId);
				desktopNameInput = `${current?.name || 'Desktop'} Copy`;
				desktopDeleteConfirmation = '';
				desktopKindInput = current?.kind || 'workspace';
				desktopModalError = '';
				desktopModalMode = 'duplicate';
				break;
			}
			case 'rename-desktop': {
				const current = $windowStore.desktopSpaces.find((space) => space.id === $windowStore.activeDesktopId);
				if (!current) break;
				desktopNameInput = current.name;
				desktopDeleteConfirmation = '';
				desktopKindInput = current.kind || 'personal';
				desktopModalError = '';
				desktopModalMode = 'rename';
				break;
			}
			case 'delete-desktop': {
				const current = $windowStore.desktopSpaces.find((space) => space.id === $windowStore.activeDesktopId);
				if (!current || $windowStore.desktopSpaces.length <= 1) break;
				desktopNameInput = current.name;
				desktopDeleteConfirmation = '';
				desktopKindInput = current.kind || 'personal';
				desktopModalError = '';
				desktopModalMode = 'delete';
				break;
			}
			case 'exit-desktop':
				goto('/dashboard');
				break;
			case 'logout':
				signOut();
				break;
			case 'open-terminal':
				windowStore.openWindow('terminal');
				break;
			case 'open-docs':
				goto('/docs');
				break;
			case 'edit-layout':
				desktop3dLayoutStore.enterEditMode();
				break;
			case 'save-layout':
				showSaveLayoutModal = true;
				break;
			case 'cancel-edit':
				desktop3dLayoutStore.exitEditMode();
				break;
			case 'manage-layouts':
				showLayoutManager = true;
				break;
			case 'reset-layout':
				if (confirm('Reset to default layout? This will discard unsaved changes.')) {
					desktop3dLayoutStore.loadLayout('default');
					desktop3dLayoutStore.exitEditMode();
				}
				break;
		}
	}

	function closeDesktopModal() {
		desktopModalMode = null;
		desktopNameInput = '';
		desktopDeleteConfirmation = '';
		desktopModalError = '';
	}

	function getDesktopTypeLabel(space: { id: string; name: string; kind: 'personal' | 'team' | 'workspace' }) {
		if (space.name.toLowerCase() === 'infinity desktop') return 'Infinity Canvas';
		if (space.kind === 'workspace' || space.kind === 'team') return 'Workspace Desktop';
		return 'Personal';
	}

	function createStableDesktopId() {
		if (typeof crypto !== 'undefined' && 'randomUUID' in crypto) {
			return crypto.randomUUID();
		}
		return `desktop-${Date.now()}-${Math.random().toString(16).slice(2)}`;
	}

	function isUuid(value: string) {
		return /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(value);
	}

	async function persistActiveDesktopSpace() {
		const workspaceId = $currentWorkspace?.id;
		if (!workspaceId) return;
		const state = get(windowStore);
		const space = state.desktopSpaces.find((item) => item.id === state.activeDesktopId);
		if (!space) return;
		await saveWorkspaceDesktopSpace(workspaceId, space);
	}

	async function switchToWorkspaceBackedDesktop(desktopId: string) {
		let nextDesktopId = desktopId;
		if (!isUuid(nextDesktopId)) {
			nextDesktopId = createStableDesktopId();
			windowStore.rekeyDesktopSpace(desktopId, nextDesktopId);
		}
		windowStore.switchDesktopSpace(nextDesktopId);
		if (nextDesktopId !== desktopId) {
			try {
				await persistActiveDesktopSpace();
			} catch (error) {
				console.error('[desktop-spaces] failed to save upgraded desktop space id', error);
			}
		}
		desktopSettings.set3DDesktop(false);
	}

	async function openCollaborativeWorkspaceDesktop() {
		const existing = get(windowStore).desktopSpaces.find(
			(space) =>
				(space.kind === 'workspace' || space.kind === 'team') &&
				space.name.toLowerCase() !== 'infinity desktop'
		);
		if (existing) {
			await switchToWorkspaceBackedDesktop(existing.id);
			return;
		}

		windowStore.createDesktopSpace('Workspace Desktop', {
			id: createStableDesktopId(),
			kind: 'workspace'
		});
		try {
			await persistActiveDesktopSpace();
		} catch (error) {
			console.error('[desktop-spaces] failed to save collaborative workspace desktop', error);
		}
		desktopSettings.set3DDesktop(false);
	}

	async function openInfinityDesktop() {
		const existing = get(windowStore).desktopSpaces.find((space) => space.name.toLowerCase() === 'infinity desktop');
		if (existing) {
			await switchToWorkspaceBackedDesktop(existing.id);
			return;
		}

		windowStore.createDesktopSpace('Infinity Desktop', {
			id: createStableDesktopId(),
			kind: 'workspace'
		});
		try {
			await persistActiveDesktopSpace();
		} catch (error) {
			console.error('[desktop-spaces] failed to save infinity desktop', error);
		}
		desktopSettings.set3DDesktop(false);
	}

	async function renameDesktopSpace(desktopId: string, name: string) {
		windowStore.renameDesktopSpace(desktopId, name);
		if (desktopId === get(windowStore).activeDesktopId) {
			try {
				await persistActiveDesktopSpace();
			} catch (error) {
				console.error('[desktop-spaces] failed to save renamed desktop space', error);
			}
		}
	}

	async function deleteDesktopSpace(desktopId: string) {
		const workspaceId = $currentWorkspace?.id;
		windowStore.deleteDesktopSpace(desktopId);
		if (!workspaceId || !isUuid(desktopId)) return;
		try {
			await deleteWorkspaceDesktopSpace(workspaceId, desktopId);
		} catch (error) {
			console.error('[desktop-spaces] failed to delete desktop space', error);
		}
	}

	async function submitDesktopModal(event: SubmitEvent) {
		event.preventDefault();
		const current = get(windowStore).desktopSpaces.find((space) => space.id === get(windowStore).activeDesktopId);
		if (!current || !desktopModalMode) return;

		if (desktopModalMode === 'delete') {
			if (get(windowStore).desktopSpaces.length <= 1) {
				desktopModalError = 'Keep at least one desktop.';
				return;
			}
			if (desktopDeleteConfirmation.trim() !== current.name) {
				desktopModalError = `Type "${current.name}" to delete this desktop.`;
				return;
			}
			await deleteDesktopSpace(current.id);
			closeDesktopModal();
			return;
		}

		const name = desktopNameInput.trim();
		if (!name || !desktopModalMode) return;
		desktopModalError = '';
		if (desktopModalMode === 'new') {
			const id = createStableDesktopId();
			windowStore.createDesktopSpace(name, { id, kind: desktopKindInput });
		} else if (desktopModalMode === 'duplicate') {
			const id = createStableDesktopId();
			windowStore.duplicateDesktopSpace(name, { id, kind: desktopKindInput });
		} else if (desktopModalMode === 'rename') {
			await renameDesktopSpace(current.id, name);
			closeDesktopModal();
			return;
		}
		try {
			await persistActiveDesktopSpace();
		} catch (error) {
			console.error('[desktop-spaces] failed to save desktop space', error);
			desktopModalError = 'Created locally, but failed to save to the workspace.';
			return;
		}
		closeDesktopModal();
	}

	function handleWindowSelect(windowId: string) {
		closeMenus();
		const win = $windowStore.windows.find(w => w.id === windowId);
		if (win?.minimized) {
			windowStore.restoreWindow(windowId);
		} else {
			windowStore.focusWindow(windowId);
		}
	}

	// Click outside — close all menus
	function handleClickOutside(event: MouseEvent) {
		const target = event.target as HTMLElement;
		if (
			!target.closest('.menu-bar-item') &&
			!target.closest('.menu-dropdown') &&
			!target.closest('.menu-bar-logo') &&
			!target.closest('.menu-bar-avatar') &&
			!target.closest('.menu-bar-clock')
		) {
			closeMenus();
		}
	}

	$effect(() => {
		if (activeMenu) {
			document.addEventListener('click', handleClickOutside);
			return () => document.removeEventListener('click', handleClickOutside);
		}
	});

	// Derived menu definitions
	const menus = $derived([
		{
			id: 'file',
			label: 'File',
			items: [
				{ label: 'New Window', shortcut: 'Cmd+N', action: 'new-window', disabled: !$focusedWindow },
				{ type: 'separator' },
				{ label: 'Close Window', shortcut: 'Cmd+W', action: 'close-window', disabled: !$focusedWindow },
				{ label: 'Close All Windows', action: 'close-all', disabled: $windowStore.windows.length === 0 },
				{ type: 'separator' },
				{ label: 'Exit Desktop View', action: 'exit-desktop' },
			]
		},
		{
			id: 'edit',
			label: 'Edit',
			items: [
				{ label: 'Undo', shortcut: 'Cmd+Z', action: 'undo', disabled: true },
				{ label: 'Redo', shortcut: 'Cmd+Shift+Z', action: 'redo', disabled: true },
				{ type: 'separator' },
				{ label: 'Cut', shortcut: 'Cmd+X', action: 'cut', disabled: true },
				{ label: 'Copy', shortcut: 'Cmd+C', action: 'copy', disabled: true },
				{ label: 'Paste', shortcut: 'Cmd+V', action: 'paste', disabled: true },
				{ label: 'Select All', shortcut: 'Cmd+A', action: 'select-all', disabled: true },
			]
		},
		{
			id: 'view',
			label: 'View',
			items: $desktopSettings.enable3DDesktop
				? [
						{ label: `Current: ${$activeLayout?.name || 'Default'}`, action: '', disabled: true },
						{ type: 'separator' },
						{ label: $isEditMode ? 'Cancel Edit Mode' : 'Edit Layout', action: $isEditMode ? 'cancel-edit' : 'edit-layout' },
						{ label: 'Save Layout As...', action: 'save-layout', disabled: !$isEditMode },
						{ label: 'Manage Layouts', action: 'manage-layouts' },
						{ type: 'separator' },
						{ label: 'Reset Layout', action: 'reset-layout' },
				  ]
				: [
						{ label: 'Zoom In', shortcut: 'Cmd++', action: 'zoom-in', disabled: true },
						{ label: 'Zoom Out', shortcut: 'Cmd+-', action: 'zoom-out', disabled: true },
						{ label: 'Actual Size', shortcut: 'Cmd+0', action: 'zoom-reset', disabled: true },
						{ type: 'separator' },
						{ label: 'Arrange Windows', action: 'arrange', disabled: true },
						{ label: 'Tile Windows', action: 'tile', disabled: true },
				  ]
		},
		{
			id: 'spaces',
			label: 'Desktops',
			items: [
				{ label: `Current: ${$windowStore.desktopSpaces.find((space) => space.id === $windowStore.activeDesktopId)?.name || 'Personal'}`, action: '', disabled: true },
				{ type: 'separator' },
				{ label: 'New Desktop...', action: 'new-desktop' },
				{ label: 'Duplicate Current Desktop...', action: 'duplicate-desktop' },
				{ label: 'Rename Current Desktop...', action: 'rename-desktop' },
				{ label: 'Delete Current Desktop', action: 'delete-desktop', disabled: $windowStore.desktopSpaces.length <= 1 },
				{ type: 'separator' },
				...$windowStore.desktopSpaces.map((space) => ({
					label: space.name,
					shortcut: getDesktopTypeLabel(space),
					action: `desktop:switch:${space.id}`,
					checked: space.id === $windowStore.activeDesktopId
				})),
			]
		},
		{
			id: 'window',
			label: 'Window',
			items: [
				{ label: 'Minimize', shortcut: 'Cmd+M', action: 'minimize', disabled: !$focusedWindow },
				{ label: $focusedWindow?.maximized ? 'Restore' : 'Maximize', action: 'maximize', disabled: !$focusedWindow },
				{ type: 'separator' },
				...$windowStore.windows.map(w => ({
					label: w.title + (w.minimized ? ' (minimized)' : ''),
					action: `window:${w.id}`,
					checked: w.id === $focusedWindow?.id
				})),
				...($windowStore.windows.length > 0 ? [{ type: 'separator' }] : []),
				{ label: 'Bring All to Front', action: 'bring-all-front', disabled: true },
			]
		},
		{
			id: 'help',
			label: 'Help',
			items: [
				{ label: 'Keyboard Shortcuts', action: 'shortcuts', disabled: true },
				{ label: 'Documentation', action: 'open-docs' },
				{ type: 'separator' },
				{ label: 'About Business OS', action: 'about', disabled: true },
			]
		},
	]);
</script>

<div
	class="menu-bar"
	class:electron={inElectron}
	class:traffic-light-space={needsTrafficLightSpace}
	role="menubar"
>
	<!-- Left: logo, app name, menus -->
	<div class="menu-bar-left">
		<MenuBarDesktopMenu
			isOpen={activeMenu === 'desktop'}
			onToggle={() => toggleMenu('desktop')}
			onAction={handleMenuAction}
		/>

		<span class="menu-bar-app-name">
			{$focusedWindow?.title || 'Business OS'}
		</span>

		<MenuBarMenus
			{menus}
			{activeMenu}
			onToggle={toggleMenu}
			onAction={handleMenuAction}
			onWindowSelect={handleWindowSelect}
		/>
	</div>

	{#if inElectron}
		<div class="menu-bar-drag-region" aria-hidden="true"></div>
	{/if}

	<!-- Right: clock, user avatar -->
	<div class="menu-bar-right">
		<MenuBarClock
			isOpen={activeMenu === 'calendar'}
			onToggle={() => toggleMenu('calendar')}
		/>

		<MenuBarUserMenu
			isOpen={activeMenu === 'user'}
			onToggle={() => toggleMenu('user')}
			onClose={closeMenus}
			onAction={handleMenuAction}
		/>
	</div>
</div>

<!-- Save Layout Modal (3D Desktop) -->
{#if showSaveLayoutModal}
	<SaveLayoutModal
		onClose={() => { showSaveLayoutModal = false; }}
		onSaved={() => { showSaveLayoutModal = false; closeMenus(); }}
	/>
{/if}

<!-- Layout Manager Modal (3D Desktop) -->
<LayoutManager show={showLayoutManager} onClose={() => showLayoutManager = false} />

{#if desktopModalMode}
	<div class="desktop-modal-overlay">
		<form class="desktop-modal" onsubmit={submitDesktopModal}>
			<div>
				<h2>
					{#if desktopModalMode === 'new'}
						New desktop
					{:else if desktopModalMode === 'duplicate'}
						Duplicate desktop
					{:else if desktopModalMode === 'rename'}
						Rename desktop
					{:else}
						Delete desktop
					{/if}
				</h2>
				<p>
					{#if desktopModalMode === 'new'}
						Start with an empty desktop, then add modules and apps from settings.
					{:else if desktopModalMode === 'duplicate'}
						Copy the current desktop layout, windows, modules, apps, and wallpaper.
					{:else if desktopModalMode === 'rename'}
						Rename the current desktop without changing its layout or settings.
					{:else}
						This permanently removes the current desktop locally and from the workspace when it has been synced.
					{/if}
				</p>
			</div>
			{#if desktopModalMode === 'delete'}
				<div class="desktop-delete-summary">
					<strong>{desktopNameInput}</strong>
					<span>Next desktop: {$windowStore.desktopSpaces.find((space) => space.id !== $windowStore.activeDesktopId)?.name || 'Personal'}</span>
				</div>
				<label>
					<span>Type desktop name to delete</span>
					<input bind:value={desktopDeleteConfirmation} autocomplete="off" />
				</label>
			{:else}
				<label>
					<span>Name</span>
					<input bind:value={desktopNameInput} autocomplete="off" />
				</label>
				{#if desktopModalMode !== 'rename'}
					<div class="desktop-kind-options" role="radiogroup" aria-label="Desktop type">
						<label class:active={desktopKindInput === 'personal'}>
							<input type="radio" bind:group={desktopKindInput} value="personal" />
							<span>Personal</span>
						</label>
						<label class:active={desktopKindInput === 'team'}>
							<input type="radio" bind:group={desktopKindInput} value="team" />
							<span>Team</span>
						</label>
						<label class:active={desktopKindInput === 'workspace'}>
							<input type="radio" bind:group={desktopKindInput} value="workspace" />
							<span>Workspace</span>
						</label>
					</div>
				{/if}
			{/if}
			{#if desktopModalError}
				<p class="desktop-modal-error">{desktopModalError}</p>
			{/if}
			<div class="desktop-modal-actions">
				<button type="button" class="secondary" onclick={closeDesktopModal}>Cancel</button>
				<button type="submit" class:danger={desktopModalMode === 'delete'}>
					{#if desktopModalMode === 'new'}
						Create desktop
					{:else if desktopModalMode === 'duplicate'}
						Duplicate
					{:else if desktopModalMode === 'rename'}
						Rename
					{:else}
						Delete desktop
					{/if}
				</button>
			</div>
		</form>
	</div>
{/if}

<style>
	.menu-bar {
		position: fixed;
		top: 0;
		left: 0;
		right: 0;
		height: 26px;
		background: rgba(255, 255, 255, 0.85);
		backdrop-filter: blur(20px);
		-webkit-backdrop-filter: blur(20px);
		border-bottom: 1px solid rgba(0, 0, 0, 0.1);
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 0 8px;
		z-index: 10000;
		font-size: 13px;
		font-weight: 500;
		color: #333;
		user-select: none;
	}

	.menu-bar.traffic-light-space {
		height: 28px;
		padding-left: 112px;
		padding-top: 0;
		padding-bottom: 0;
		align-items: center;
		box-sizing: border-box;
	}

	.menu-bar-left {
		display: flex;
		align-items: center;
		gap: 0;
		position: relative;
		z-index: 1;
		-webkit-app-region: no-drag;
	}

	.menu-bar-drag-region {
		align-self: stretch;
		flex: 1 1 auto;
		min-width: 24px;
		-webkit-app-region: drag;
	}

	.menu-bar-right {
		display: flex;
		align-items: center;
		gap: 12px;
		position: relative;
		z-index: 1;
		-webkit-app-region: no-drag;
	}

	.menu-bar-app-name {
		font-weight: 600;
		padding: 0 12px 0 4px;
		color: #111;
		max-width: 200px;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	/* Shared dropdown styles used by all child components */
	:global(.menu-dropdown) {
		position: absolute;
		top: 100%;
		left: 0;
		margin-top: 2px;
		min-width: 220px;
		background: rgba(255, 255, 255, 0.98);
		backdrop-filter: blur(20px);
		-webkit-backdrop-filter: blur(20px);
		border: 1px solid rgba(0, 0, 0, 0.15);
		border-radius: 6px;
		box-shadow: 0 10px 40px rgba(0, 0, 0, 0.15);
		padding: 4px 0;
		z-index: 10001;
	}

	:global(.menu-dropdown.user-menu) {
		right: 0;
		left: auto;
	}

	:global(.menu-item) {
		display: flex;
		align-items: center;
		width: 100%;
		padding: 6px 12px;
		background: none;
		border: none;
		cursor: pointer;
		font-size: 13px;
		color: #333;
		text-align: left;
		gap: 8px;
		border-radius: 4px;
		margin: 0 4px;
		width: calc(100% - 8px);
	}

	:global(.menu-item:hover:not(.disabled)) {
		background: #0066FF;
		color: white;
	}

	:global(.menu-item:hover:not(.disabled) .menu-item-shortcut) {
		color: rgba(255, 255, 255, 0.7);
	}

	:global(.menu-item.disabled) {
		color: #999;
		cursor: default;
	}

	:global(.menu-item-check) {
		width: 16px;
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
	}

	:global(.menu-item-label) {
		flex: 1;
	}

	:global(.menu-item-shortcut) {
		color: #999;
		font-size: 12px;
		margin-left: auto;
	}

	:global(.menu-item-shortcut.beta-badge) {
		background: linear-gradient(135deg, #6366f1, #8b5cf6);
		color: white;
		padding: 2px 6px;
		border-radius: 4px;
		font-size: 10px;
		font-weight: 600;
		text-transform: uppercase;
	}

	.desktop-modal-overlay {
		position: fixed;
		inset: 0;
		z-index: 20000;
		display: flex;
		align-items: center;
		justify-content: center;
		background: rgba(15, 23, 42, 0.28);
		backdrop-filter: blur(6px);
		-webkit-backdrop-filter: blur(6px);
		-webkit-app-region: no-drag;
	}

	.desktop-modal {
		width: min(420px, calc(100vw - 32px));
		display: flex;
		flex-direction: column;
		gap: 18px;
		padding: 22px;
		border: 1px solid rgba(17, 24, 39, 0.12);
		border-radius: 12px;
		background: rgba(255, 255, 255, 0.98);
		box-shadow: 0 24px 70px rgba(15, 23, 42, 0.24);
		color: #111827;
	}

	.desktop-modal h2 {
		margin: 0 0 6px;
		font-size: 18px;
		font-weight: 780;
	}

	.desktop-modal p {
		margin: 0;
		color: #6b7280;
		font-size: 12px;
		line-height: 1.45;
	}

	.desktop-modal label {
		display: flex;
		flex-direction: column;
		gap: 7px;
		font-size: 11px;
		font-weight: 750;
		color: #4b5563;
		text-transform: uppercase;
		letter-spacing: 0.06em;
	}

	.desktop-modal input {
		height: 40px;
		padding: 0 12px;
		border: 1px solid #d1d5db;
		border-radius: 8px;
		background: #fff;
		color: #111827;
		font-size: 14px;
		font-weight: 600;
		outline: none;
	}

	.desktop-modal input:focus {
		border-color: #111827;
		box-shadow: 0 0 0 3px rgba(17, 24, 39, 0.08);
	}

	.desktop-kind-options {
		display: grid;
		grid-template-columns: repeat(3, minmax(0, 1fr));
		gap: 8px;
	}

	.desktop-kind-options label {
		height: 38px;
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 0 10px;
		border: 1px solid #e5e7eb;
		border-radius: 8px;
		background: #fff;
		color: #4b5563;
		font-size: 12px;
		font-weight: 750;
		text-transform: none;
		letter-spacing: 0;
		cursor: pointer;
	}

	.desktop-kind-options label.active {
		border-color: #111827;
		background: #111827;
		color: #fff;
	}

	.desktop-kind-options input {
		position: absolute;
		opacity: 0;
		pointer-events: none;
	}

	.desktop-modal-error {
		padding: 9px 10px;
		border-radius: 8px;
		background: #fef2f2;
		color: #b91c1c !important;
		font-weight: 700;
	}

	.desktop-delete-summary {
		display: flex;
		flex-direction: column;
		gap: 4px;
		padding: 12px;
		border: 1px solid #fecaca;
		border-radius: 8px;
		background: #fff7f7;
	}

	.desktop-delete-summary strong {
		color: #991b1b;
		font-size: 14px;
	}

	.desktop-delete-summary span {
		color: #6b7280;
		font-size: 12px;
	}

	.desktop-modal-actions {
		display: flex;
		justify-content: flex-end;
		gap: 8px;
	}

	.desktop-modal-actions button {
		height: 36px;
		padding: 0 14px;
		border: 0;
		border-radius: 8px;
		background: #111827;
		color: #fff;
		font-size: 12px;
		font-weight: 750;
		cursor: pointer;
	}

	.desktop-modal-actions button.secondary {
		border: 1px solid #e5e7eb;
		background: #fff;
		color: #374151;
	}

	.desktop-modal-actions button.danger {
		background: #b91c1c;
	}

	:global(.menu-item.experimental-3d:hover .menu-item-shortcut.beta-badge) {
		background: linear-gradient(135deg, #818cf8, #a78bfa);
	}

	:global(.menu-separator) {
		height: 1px;
		background: rgba(0, 0, 0, 0.1);
		margin: 4px 8px;
	}

	/* ===== DARK MODE ===== */
	:global(.dark) .menu-bar {
		background: rgba(28, 28, 30, 0.85);
		border-bottom-color: rgba(255, 255, 255, 0.12);
		color: #f5f5f7;
	}

	:global(.dark) .menu-bar-app-name {
		color: #f5f5f7;
	}

	:global(.dark) :global(.menu-dropdown) {
		background: rgba(44, 44, 46, 0.98);
		border-color: rgba(255, 255, 255, 0.12);
		box-shadow: 0 10px 40px rgba(0, 0, 0, 0.4);
	}

	:global(.dark) :global(.menu-item) {
		color: #f5f5f7;
	}

	:global(.dark) :global(.menu-item:hover:not(.disabled)) {
		background: #0A84FF;
	}

	:global(.dark) :global(.menu-item.disabled) {
		color: #6e6e73;
	}

	:global(.dark) :global(.menu-item-shortcut) {
		color: #6e6e73;
	}

	:global(.dark) :global(.menu-separator) {
		background: rgba(255, 255, 255, 0.1);
	}
</style>
