<script lang="ts">
	/**
	 * SavedViewsManager - Manage saved/custom views
	 */

	import type { SavedView, ViewConfig } from '../types/view';
	import { TemplateButton, TemplateInput, TemplateModal, TemplateDropdown } from '../primitives';

	interface Props {
		views: SavedView[];
		currentViewId?: string;
		onselectview?: (view: SavedView) => void;
		onsaveview?: (name: string, config: ViewConfig) => void;
		ondeleteview?: (viewId: string) => void;
		onrenameview?: (viewId: string, name: string) => void;
		currentConfig?: ViewConfig;
	}

	let {
		views,
		currentViewId,
		onselectview,
		onsaveview,
		ondeleteview,
		onrenameview,
		currentConfig
	}: Props = $props();

	let saveModalOpen = $state(false);
	let newViewName = $state('');
	let editingViewId = $state<string | null>(null);
	let editingName = $state('');

	function handleSave(e?: Event) {
		e?.preventDefault();
		if (newViewName.trim() && currentConfig) {
			onsaveview?.(newViewName.trim(), currentConfig);
			newViewName = '';
			saveModalOpen = false;
		}
	}

	function startEditing(view: SavedView) {
		editingViewId = view.id;
		editingName = view.name;
	}

	function saveEdit(e?: Event) {
		e?.preventDefault();
		if (editingViewId && editingName.trim()) {
			onrenameview?.(editingViewId, editingName.trim());
			editingViewId = null;
			editingName = '';
		}
	}

	function cancelEdit() {
		editingViewId = null;
		editingName = '';
	}
</script>

<div class="tpl-saved-views">
	<div class="tpl-saved-views-header">
		<span class="tpl-saved-views-label">Views</span>
		<TemplateButton variant="ghost" size="sm" onclick={() => saveModalOpen = true}>
			<svg viewBox="0 0 20 20" fill="currentColor" width="14" height="14">
				<path fill-rule="evenodd" d="M10 3a1 1 0 011 1v5h5a1 1 0 110 2h-5v5a1 1 0 11-2 0v-5H4a1 1 0 110-2h5V4a1 1 0 011-1z" clip-rule="evenodd" />
			</svg>
			Save view
		</TemplateButton>
	</div>

	<div class="tpl-saved-views-list">
		{#each views as view}
			<div
				class="tpl-saved-view-item"
				class:tpl-saved-view-item-active={view.id === currentViewId}
			>
				{#if editingViewId === view.id}
					<form class="tpl-saved-view-edit" onsubmit={saveEdit}>
						<TemplateInput
							value={editingName}
							size="sm"
							oninput={(e) => editingName = (e.target as HTMLInputElement).value}
						/>
						<TemplateButton variant="ghost" size="sm" onclick={saveEdit}>
							<svg viewBox="0 0 20 20" fill="currentColor" width="14" height="14">
								<path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd" />
							</svg>
						</TemplateButton>
						<TemplateButton variant="ghost" size="sm" onclick={cancelEdit}>
							<svg viewBox="0 0 20 20" fill="currentColor" width="14" height="14">
								<path fill-rule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clip-rule="evenodd" />
							</svg>
						</TemplateButton>
					</form>
				{:else}
					<button
						class="tpl-saved-view-btn"
						onclick={() => onselectview?.(view)}
					>
						<svg viewBox="0 0 20 20" fill="currentColor" class="tpl-saved-view-icon">
							{#if view.config.type === 'table'}
								<path fill-rule="evenodd" d="M5 4a2 2 0 00-2 2v8a2 2 0 002 2h10a2 2 0 002-2V6a2 2 0 00-2-2H5zm0 2h10v2H5V6zm0 4h4v4H5v-4zm6 0h4v4h-4v-4z" clip-rule="evenodd" />
							{:else if view.config.type === 'card'}
								<path d="M5 3a2 2 0 00-2 2v2a2 2 0 002 2h2a2 2 0 002-2V5a2 2 0 00-2-2H5zM5 11a2 2 0 00-2 2v2a2 2 0 002 2h2a2 2 0 002-2v-2a2 2 0 00-2-2H5zM11 5a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V5zM11 13a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z" />
							{:else if view.config.type === 'kanban'}
								<path d="M2 4a1 1 0 011-1h2a1 1 0 011 1v12a1 1 0 01-1 1H3a1 1 0 01-1-1V4zM8 4a1 1 0 011-1h2a1 1 0 011 1v12a1 1 0 01-1 1H9a1 1 0 01-1-1V4zM15 3a1 1 0 00-1 1v12a1 1 0 001 1h2a1 1 0 001-1V4a1 1 0 00-1-1h-2z" />
							{:else}
								<path fill-rule="evenodd" d="M3 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm0 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm0 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm0 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1z" clip-rule="evenodd" />
							{/if}
						</svg>
						<span class="tpl-saved-view-name">{view.name}</span>
						{#if view.isPersonal}
							<span class="tpl-saved-view-badge">Personal</span>
						{/if}
					</button>
					<TemplateDropdown
						items={[
							{ id: 'rename', label: 'Rename' },
							{ id: 'duplicate', label: 'Duplicate' },
							{ id: 'delete', label: 'Delete', danger: true }
						]}
						align="end"
						onselect={(item) => {
							if (item.id === 'rename') {
								startEditing(view);
							} else if (item.id === 'delete') {
								ondeleteview?.(view.id);
							}
						}}
					>
						{#snippet trigger()}
							<button class="tpl-saved-view-menu">
								<svg viewBox="0 0 20 20" fill="currentColor">
									<path d="M6 10a2 2 0 11-4 0 2 2 0 014 0zM12 10a2 2 0 11-4 0 2 2 0 014 0zM16 12a2 2 0 100-4 2 2 0 000 4z" />
								</svg>
							</button>
						{/snippet}
					</TemplateDropdown>
				{/if}
			</div>
		{/each}
	</div>
</div>

<!-- Save View Modal -->
<TemplateModal bind:open={saveModalOpen}>
	<div class="tpl-save-view-modal">
		<h2 class="tpl-save-view-title">Save View</h2>
		<p class="tpl-save-view-desc">Save the current filters, sort, and column settings as a new view.</p>
		<form onsubmit={handleSave}>
			<TemplateInput
				value={newViewName}
				placeholder="View name"
				oninput={(e) => newViewName = (e.target as HTMLInputElement).value}
			/>
			<div class="tpl-save-view-actions">
				<TemplateButton variant="outline" onclick={() => saveModalOpen = false}>Cancel</TemplateButton>
				<TemplateButton variant="primary" disabled={!newViewName.trim()} onclick={handleSave}>
					Save View
				</TemplateButton>
			</div>
		</form>
	</div>
</TemplateModal>

<style>
	.tpl-saved-views {
		min-width: 200px;
	}

	.tpl-saved-views-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: var(--tpl-space-2) var(--tpl-space-3);
		border-bottom: 1px solid var(--tpl-border-subtle);
	}

	.tpl-saved-views-label {
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-xs);
		font-weight: var(--tpl-font-semibold);
		color: var(--tpl-text-muted);
		text-transform: uppercase;
		letter-spacing: var(--tpl-tracking-wide);
	}

	.tpl-saved-views-list {
		padding: var(--tpl-space-1);
	}

	.tpl-saved-view-item {
		display: flex;
		align-items: center;
		border-radius: var(--tpl-radius-md);
		transition: background var(--tpl-transition-fast);
	}

	.tpl-saved-view-item:hover {
		background: var(--tpl-bg-hover);
	}

	.tpl-saved-view-item-active {
		background: var(--tpl-bg-selected);
	}

	.tpl-saved-view-btn {
		flex: 1;
		display: flex;
		align-items: center;
		gap: var(--tpl-space-2);
		padding: var(--tpl-space-2) var(--tpl-space-3);
		background: transparent;
		border: none;
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-sm);
		color: var(--tpl-text-primary);
		text-align: left;
		cursor: pointer;
	}

	.tpl-saved-view-icon {
		width: 16px;
		height: 16px;
		color: var(--tpl-text-muted);
	}

	.tpl-saved-view-name {
		flex: 1;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.tpl-saved-view-badge {
		font-size: var(--tpl-text-2xs);
		padding: 1px var(--tpl-space-1-5);
		background: var(--tpl-bg-tertiary);
		border-radius: var(--tpl-radius-sm);
		color: var(--tpl-text-muted);
	}

	.tpl-saved-view-menu {
		width: 28px;
		height: 28px;
		padding: 0;
		display: flex;
		align-items: center;
		justify-content: center;
		background: transparent;
		border: none;
		border-radius: var(--tpl-radius-md);
		color: var(--tpl-text-muted);
		cursor: pointer;
		opacity: 0;
		transition: all var(--tpl-transition-fast);
	}

	.tpl-saved-view-item:hover .tpl-saved-view-menu {
		opacity: 1;
	}

	.tpl-saved-view-menu:hover {
		background: var(--tpl-bg-hover);
		color: var(--tpl-text-primary);
	}

	.tpl-saved-view-menu svg {
		width: 16px;
		height: 16px;
	}

	.tpl-saved-view-edit {
		flex: 1;
		display: flex;
		align-items: center;
		gap: var(--tpl-space-1);
		padding: var(--tpl-space-1);
	}

	.tpl-saved-view-edit :global(input) {
		flex: 1;
	}

	/* Save Modal */
	.tpl-save-view-modal {
		width: 400px;
		padding: var(--tpl-space-6);
	}

	.tpl-save-view-title {
		margin: 0 0 var(--tpl-space-1);
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-lg);
		font-weight: var(--tpl-font-semibold);
		color: var(--tpl-text-primary);
	}

	.tpl-save-view-desc {
		margin: 0 0 var(--tpl-space-4);
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-sm);
		color: var(--tpl-text-muted);
	}

	.tpl-save-view-actions {
		display: flex;
		justify-content: flex-end;
		gap: var(--tpl-space-2);
		margin-top: var(--tpl-space-4);
	}
</style>
