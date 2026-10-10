<script lang="ts">
	/**
	 * Database Component
	 * Main database block renderer with multiple view types
	 */
	import { onMount, onDestroy } from 'svelte';
	import DatabaseViewTabs from './DatabaseViewTabs.svelte';
	import DatabaseTable from './DatabaseTable.svelte';
	import { createDatabaseStore, type DatabaseStore } from '../../stores/database-store';
	import type { BlockStore } from '../../stores/yjs-block-store';
	import type { DatabaseViewType, DatabaseView, Block } from '../../entities/block';

	interface Props {
		blockStore: BlockStore;
		databaseId: string;
		showTitle?: boolean;
		class?: string;
	}

	let {
		blockStore,
		databaseId,
		showTitle = true,
		class: className = ''
	}: Props = $props();

	// Create database store
	let dbStore: DatabaseStore | null = $state(null);

	onMount(() => {
		dbStore = createDatabaseStore(blockStore, databaseId);
	});

	onDestroy(() => {
		dbStore?.destroy();
	});

	// Derived state - use subscribe pattern to avoid 'never' type issue
	let storeValue = $state<{ database: unknown; activeViewId: string | null } | null>(null);

	$effect(() => {
		if (dbStore) {
			const unsubscribe = dbStore.subscribe(value => {
				storeValue = value;
			});
			return unsubscribe;
		} else {
			storeValue = null;
		}
	});

	let database = $derived(storeValue?.database as Block<'bos:database'> | null);
	let views = $derived(database?.props.views ?? []);
	let activeViewId = $derived(storeValue?.activeViewId ?? null);
	let activeView = $derived(views.find((v: DatabaseView) => v.id === activeViewId));
	let title = $derived(database?.props.title?.delta?.map((d: { insert: string }) => d.insert).join('') ?? 'Untitled Database');

	// View handlers
	function handleSelectView(viewId: string) {
		dbStore?.setActiveView(viewId);
	}

	function handleAddView(type: DatabaseViewType) {
		const name = `${type.charAt(0).toUpperCase()}${type.slice(1)} view`;
		dbStore?.addView({
			name,
			type,
			columns: database?.props.columns.map((c: { id: string }) => c.id) ?? []
		});
	}

	function handleRenameView(viewId: string, name: string) {
		dbStore?.updateView(viewId, { name });
	}

	function handleDuplicateView(viewId: string) {
		const view = views.find((v: DatabaseView) => v.id === viewId);
		if (!view) return;

		dbStore?.addView({
			name: `${view.name} (copy)`,
			type: view.type,
			columns: [...view.columns],
			filter: view.filter,
			sorts: view.sorts,
			groupBy: view.groupBy
		});
	}

	function handleDeleteView(viewId: string) {
		dbStore?.deleteView(viewId);
	}
</script>

{#if dbStore}
	<div class="bos-database {className}">
		<!-- Title -->
		{#if showTitle}
			<div class="bos-database__header">
				<h2 class="bos-database__title">{title}</h2>
			</div>
		{/if}

		<!-- View tabs -->
		<DatabaseViewTabs
			{views}
			{activeViewId}
			onSelectView={handleSelectView}
			onAddView={handleAddView}
			onRenameView={handleRenameView}
			onDuplicateView={handleDuplicateView}
			onDeleteView={handleDeleteView}
		/>

		<!-- View content -->
		<div class="bos-database__content">
			{#if activeView}
				{#if activeView.type === 'table'}
					<DatabaseTable store={dbStore} />
				{:else if activeView.type === 'kanban'}
					<div class="bos-database__placeholder">
						Kanban view coming soon
					</div>
				{:else if activeView.type === 'calendar'}
					<div class="bos-database__placeholder">
						Calendar view coming soon
					</div>
				{:else if activeView.type === 'gallery'}
					<div class="bos-database__placeholder">
						Gallery view coming soon
					</div>
				{:else if activeView.type === 'list'}
					<div class="bos-database__placeholder">
						List view coming soon
					</div>
				{/if}
			{:else}
				<div class="bos-database__empty">
					No views configured
				</div>
			{/if}
		</div>
	</div>
{:else}
	<div class="bos-database__loading">
		<div class="bos-database__spinner"></div>
		<span>Loading database...</span>
	</div>
{/if}

<style>
	.bos-database {
		display: flex;
		flex-direction: column;
		height: 100%;
		background: var(--dbg);
		border: 1px solid var(--dbd);
		border-radius: 8px;
		overflow: hidden;
	}

	.bos-database__header {
		padding: 16px 16px 0;
	}

	.bos-database__title {
		font-size: 20px;
		font-weight: 600;
		color: var(--dt);
		margin: 0;
	}

	.bos-database__content {
		flex: 1;
		min-height: 0;
		overflow: hidden;
	}

	.bos-database__placeholder,
	.bos-database__empty {
		display: flex;
		align-items: center;
		justify-content: center;
		height: 200px;
		font-size: 14px;
		color: var(--dt2);
	}

	.bos-database__loading {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 12px;
		height: 200px;
		color: var(--dt2);
	}

	.bos-database__spinner {
		width: 24px;
		height: 24px;
		border: 2px solid var(--dbd);
		border-top-color: #1e96eb;
		border-radius: 50%;
		animation: spin 0.8s linear infinite;
	}

	@keyframes spin {
		to {
			transform: rotate(360deg);
		}
	}
</style>
