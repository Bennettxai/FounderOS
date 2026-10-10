<script lang="ts">
	/**
	 * Sidebar Header - BusinessOS Style
	 * Modern document-centric workspace selector and quick-search patterns
	 */
	import { Search, Plus, Settings, Trash2 } from 'lucide-svelte';
	import { Tooltip } from '$lib/ui';

	interface Props {
		onNewDocument?: () => void;
		onOpenSearch?: () => void;
		onOpenSettings?: () => void;
		onOpenTrash?: () => void;
	}

	let { onNewDocument, onOpenSearch, onOpenSettings, onOpenTrash }: Props = $props();
</script>

<div class="bos-sidebar-header">
	<div class="bos-sidebar-header__workspace">
		<span class="bos-sidebar-header__logo">P</span>
		<span class="bos-sidebar-header__name">Pages</span>
	</div>

	<div class="bos-sidebar-header__actions">
		<!-- Quick Search Button -->
		<button
			class="bos-sidebar-header__search"
			onclick={onOpenSearch}
			aria-label="Search"
		>
			<Search />
			<span>Search</span>
			<kbd>⌘K</kbd>
		</button>

		<!-- Action buttons -->
		<div class="bos-sidebar-header__buttons">
			<Tooltip content="Trash" side="bottom">
				<button
					class="bos-sidebar-header__action"
					onclick={onOpenTrash}
					aria-label="Trash"
				>
					<Trash2 />
				</button>
			</Tooltip>
			<Tooltip content="Settings" side="bottom">
				<button
					class="bos-sidebar-header__action"
					onclick={onOpenSettings}
					aria-label="Settings"
				>
					<Settings />
				</button>
			</Tooltip>
			<Tooltip content="New page" side="bottom">
				<button
					class="bos-sidebar-header__action bos-sidebar-header__action--primary"
					onclick={onNewDocument}
					aria-label="New page"
				>
					<Plus />
				</button>
			</Tooltip>
		</div>
	</div>
</div>

<style>
	.bos-sidebar-header {
		display: flex;
		flex-direction: column;
		padding: 12px;
		gap: 12px;
	}

	.bos-sidebar-header__workspace {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 4px;
	}

	.bos-sidebar-header__logo {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 24px;
		height: 24px;
		border-radius: 4px;
		background: linear-gradient(135deg, #1e96eb 0%, #1976d2 100%);
		color: #ffffff;
		font-size: 10px;
		font-weight: 700;
		letter-spacing: 0.5px;
	}

	.bos-sidebar-header__name {
		font-size: 14px;
		font-weight: 600;
		color: var(--dt);
	}

	.bos-sidebar-header__actions {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	/* Search button */
	.bos-sidebar-header__search {
		display: flex;
		align-items: center;
		gap: 8px;
		width: 100%;
		height: 32px;
		padding: 0 12px;
		border-radius: 8px;
		background-color: var(--dbg);
		border: 1px solid var(--dbd);
		color: var(--dt2);
		font-size: 14px;
		cursor: pointer;
		transition: border-color 0.15s;
	}

	.bos-sidebar-header__search:hover {
		border-color: #1e96eb;
	}

	.bos-sidebar-header__search :global(svg) {
		width: 14px;
		height: 14px;
		flex-shrink: 0;
	}

	.bos-sidebar-header__search span {
		flex: 1;
		text-align: left;
	}

	.bos-sidebar-header__search kbd {
		display: inline-flex;
		align-items: center;
		padding: 2px 6px;
		border-radius: 4px;
		background-color: var(--dbg2);
		font-family: monospace;
		font-size: 10px;
		color: var(--dt3);
	}

	/* Action buttons */
	.bos-sidebar-header__buttons {
		display: flex;
		gap: 4px;
	}

	.bos-sidebar-header__action {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 28px;
		height: 28px;
		border-radius: 4px;
		background: transparent;
		border: none;
		color: var(--dt3);
		cursor: pointer;
		transition: background-color 0.15s;
	}

	.bos-sidebar-header__action:hover {
		background: var(--dbg3);
	}

	.bos-sidebar-header__action :global(svg) {
		width: 16px;
		height: 16px;
	}

	/* Primary action (New page) - subtle emphasis */
	.bos-sidebar-header__action--primary {
		color: #1e96eb;
	}

	.bos-sidebar-header__action--primary:hover {
		background: #1e96eb;
		color: #ffffff;
	}
</style>
