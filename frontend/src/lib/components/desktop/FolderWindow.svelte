<script lang="ts">
	import { windowStore, type DesktopIcon as DesktopIconType } from '$lib/stores/windowStore';
	import DesktopIcon from './DesktopIcon.svelte';

	interface Props {
		folderId: string;
	}

	let { folderId }: Props = $props();

	// Get folder data
	let folder = $derived($windowStore.folders.find(f => f.id === folderId));
	let folderIcons = $derived($windowStore.desktopIcons.filter(
		icon => icon.folderId === folderId && icon.type !== 'folder'
	));

	// Local state
	let isRenaming = $state(false);
	let newName = $state('');
	let selectedIconId = $state<string | null>(null);

	function startRename() {
		if (folder) {
			newName = folder.name;
			isRenaming = true;
		}
	}

	function saveRename() {
		if (folder && newName.trim()) {
			windowStore.renameFolder(folderId, newName.trim());
		}
		isRenaming = false;
	}

	function handleIconDoubleClick(icon: DesktopIconType) {
		windowStore.openWindow(icon.module);
	}

	function removeFromFolder(iconId: string) {
		windowStore.removeIconFromFolder(iconId);
	}

	function handleDragOver(e: DragEvent) {
		e.preventDefault();
		if (e.dataTransfer) {
			e.dataTransfer.dropEffect = 'move';
		}
	}

	function handleDrop(e: DragEvent) {
		e.preventDefault();
		const iconId = e.dataTransfer?.getData('text/plain');
		if (iconId && folder) {
			windowStore.moveIconToFolder(iconId, folderId);
		}
	}

	// Color options for folder
	const folderColors = [
		{ color: '#3B82F6', name: 'Blue' },
		{ color: '#10B981', name: 'Green' },
		{ color: '#F59E0B', name: 'Yellow' },
		{ color: '#EF4444', name: 'Red' },
		{ color: '#8B5CF6', name: 'Purple' },
		{ color: '#EC4899', name: 'Pink' },
		{ color: '#6B7280', name: 'Gray' },
		{ color: '#F97316', name: 'Orange' },
	];

	function setColor(color: string) {
		windowStore.setFolderColor(folderId, color);
	}
</script>

<div class="folder-window" ondragover={handleDragOver} ondrop={handleDrop}>
	<!-- Folder header -->
	<div class="folder-header" style="background: {folder?.color || '#3B82F6'}20; border-bottom-color: {folder?.color || '#3B82F6'}40">
		<div class="folder-title">
			{#if isRenaming}
				<input
					type="text"
					bind:value={newName}
					onblur={saveRename}
					onkeydown={(e) => e.key === 'Enter' && saveRename()}
					class="rename-input"
					autofocus
				/>
			{:else}
				<button class="title-btn" ondblclick={startRename}>
					<svg class="folder-icon" viewBox="0 0 24 24" fill="{folder?.color || '#3B82F6'}">
						<path d="M3 7V17C3 18.1046 3.89543 19 5 19H19C20.1046 19 21 18.1046 21 17V9C21 7.89543 20.1046 7 19 7H12L10 5H5C3.89543 5 3 5.89543 3 7Z"/>
					</svg>
					<span>{folder?.name || 'Folder'}</span>
				</button>
			{/if}
		</div>
		<div class="folder-actions">
			<!-- Color picker -->
			<div class="color-picker">
				{#each folderColors as { color, name }}
					<button
						class="color-btn"
						class:active={folder?.color === color}
						style="background: {color}"
						onclick={() => setColor(color)}
						title={name}
					></button>
				{/each}
			</div>
			<span class="item-count">{folderIcons.length} item{folderIcons.length !== 1 ? 's' : ''}</span>
		</div>
	</div>

	<!-- Folder contents -->
	<div class="folder-contents">
		{#if folderIcons.length === 0}
			<div class="empty-folder">
				<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
					<path d="M3 7V17C3 18.1046 3.89543 19 5 19H19C20.1046 19 21 18.1046 21 17V9C21 7.89543 20.1046 7 19 7H12L10 5H5C3.89543 5 3 5.89543 3 7Z"/>
				</svg>
				<p>Drag items here to add them to this folder</p>
			</div>
		{:else}
			<div class="icons-grid">
				{#each folderIcons as icon (icon.id)}
					<div class="folder-item">
						<DesktopIcon
							id={icon.id}
							module={icon.module}
							label={icon.label}
							selected={selectedIconId === icon.id}
							posX={0}
							posY={0}
							darkBackground={false}
							onSelect={(id, additive) => selectedIconId = id}
							onOpen={(module) => windowStore.openWindow(module)}
							onDragStart={(id) => {}}
							onDragMove={(id, x, y) => {}}
							onDragEnd={(id, x, y) => {}}
						/>
						<button
							class="remove-btn"
							onclick={() => removeFromFolder(icon.id)}
							title="Remove from folder"
						>
							<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
								<line x1="18" y1="6" x2="6" y2="18"/>
								<line x1="6" y1="6" x2="18" y2="18"/>
							</svg>
						</button>
					</div>
				{/each}
			</div>
		{/if}
	</div>
</div>

<style>
	.folder-window {
		display: flex;
		flex-direction: column;
		height: 100%;
		background: white;
	}

	.folder-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 12px 16px;
		border-bottom: 1px solid;
	}

	.folder-title {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.title-btn {
		display: flex;
		align-items: center;
		gap: 8px;
		background: none;
		border: none;
		cursor: pointer;
		font-size: 16px;
		font-weight: 600;
		color: #111;
		padding: 4px 8px;
		border-radius: 6px;
	}

	.title-btn:hover {
		background: rgba(0, 0, 0, 0.05);
	}

	.folder-icon {
		width: 24px;
		height: 24px;
	}

	.rename-input {
		font-size: 16px;
		font-weight: 600;
		border: 1px solid #3B82F6;
		border-radius: 4px;
		padding: 4px 8px;
		outline: none;
	}

	.folder-actions {
		display: flex;
		align-items: center;
		gap: 12px;
	}

	.color-picker {
		display: flex;
		gap: 4px;
	}

	.color-btn {
		width: 16px;
		height: 16px;
		padding: 0;
		border-radius: 50%;
		border: 2px solid transparent;
		cursor: pointer;
		transition: all 0.15s;
	}

	.color-btn:hover {
		transform: scale(1.2);
	}

	.color-btn.active {
		border-color: #111;
	}

	.item-count {
		font-size: 12px;
		color: #666;
	}

	.folder-contents {
		flex: 1;
		overflow-y: auto;
		padding: 16px;
	}

	.empty-folder {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		height: 100%;
		color: #999;
		gap: 12px;
	}

	.empty-folder svg {
		width: 64px;
		height: 64px;
		opacity: 0.5;
	}

	.empty-folder p {
		font-size: 14px;
	}

	.icons-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(80px, 1fr));
		gap: 16px;
	}

	.folder-item {
		position: relative;
	}

	.remove-btn {
		position: absolute;
		top: -4px;
		right: -4px;
		width: 20px;
		height: 20px;
		border-radius: 50%;
		background: #ef4444;
		border: none;
		cursor: pointer;
		display: flex;
		align-items: center;
		justify-content: center;
		opacity: 0;
		transition: opacity 0.15s;
	}

	.folder-item:hover .remove-btn {
		opacity: 1;
	}

	.remove-btn svg {
		width: 12px;
		height: 12px;
		color: white;
	}

	.remove-btn:hover {
		background: #dc2626;
	}
</style>
