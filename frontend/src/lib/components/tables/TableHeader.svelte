<script lang="ts">
	/**
	 * TableHeader - Table name, views tabs, and actions
	 */
	import {
		ChevronDown,
		Plus,
		Star,
		Settings,
		Table2,
		Columns3,
		LayoutGrid,
		Calendar,
		FileInput
	} from 'lucide-svelte';
	import type { Table, TableView, ViewType } from '$lib/api/tables/types';
	import { VIEW_TYPES } from '$lib/api/tables/types';

	interface Props {
		table: Table;
		currentView: TableView | null;
		onViewChange: (viewId: string) => void;
		onCreateView: (type: ViewType) => void;
		onFavoriteToggle: () => void;
		onSettingsClick?: () => void;
	}

	let { table, currentView, onViewChange, onCreateView, onFavoriteToggle, onSettingsClick }: Props =
		$props();

	let showViewMenu = $state(false);

	function getViewIcon(type: ViewType) {
		switch (type) {
			case 'grid':
				return Table2;
			case 'kanban':
				return Columns3;
			case 'gallery':
				return LayoutGrid;
			case 'calendar':
				return Calendar;
			case 'form':
				return FileInput;
			default:
				return Table2;
		}
	}

	function handleClickOutside() {
		showViewMenu = false;
	}
</script>

<svelte:window onclick={handleClickOutside} />

<div class="dt2-header flex items-center justify-between px-6 py-3">
	<!-- Left: Table name and views -->
	<div class="flex items-center gap-4">
		<!-- Table Icon & Name -->
		<div class="flex items-center gap-2">
			{#if table.icon}
				<span class="text-2xl">{table.icon}</span>
			{:else}
				<div class="flex h-8 w-8 items-center justify-center rounded-lg bg-blue-100">
					<Table2 class="h-4 w-4 text-blue-600" />
				</div>
			{/if}
			<h1 class="text-xl font-semibold text-gray-900">{table.name}</h1>
			<button
				type="button"
				class="btn-pill btn-pill-ghost btn-pill-icon {table.is_favorite ? 'text-yellow-400' : ''}"
				onclick={onFavoriteToggle}
			>
				<Star class="h-5 w-5 {table.is_favorite ? 'fill-current' : ''}" />
			</button>
		</div>

		<!-- View Tabs -->
		<div class="flex items-center gap-1 pl-4" style="border-left: 1px solid var(--dbd);">
			{#each table.views as view}
				<button
					type="button"
					class="btn-pill btn-pill-sm flex items-center gap-1.5 {currentView?.id ===
					view.id
						? 'btn-pill-soft'
						: 'btn-pill-ghost'}"
					onclick={() => onViewChange(view.id)}
				>
					<svelte:component this={getViewIcon(view.type)} class="h-4 w-4" />
					{view.name}
				</button>
			{/each}

			<!-- Add View -->
			<div class="relative">
				<button
					type="button"
					class="btn-pill btn-pill-ghost btn-pill-sm flex items-center gap-1"
					onclick={(e) => {
						e.stopPropagation();
						showViewMenu = !showViewMenu;
					}}
				>
					<Plus class="h-4 w-4" />
				</button>

				{#if showViewMenu}
					<div
						class="dt2-dropdown absolute left-0 top-full mt-1 w-48 rounded-lg py-1"
					>
						<div class="px-3 py-2 text-xs font-medium uppercase" style="color: var(--dt3);">Add View</div>
						{#each VIEW_TYPES as viewType}
							<button
								type="button"
								class="dt2-dropdown__item flex w-full items-center gap-2 px-3 py-2 text-sm transition-colors"
								onclick={() => {
									onCreateView(viewType.type);
									showViewMenu = false;
								}}
							>
								<svelte:component this={getViewIcon(viewType.type)} class="h-4 w-4" style="color: var(--dt3);" />
								{viewType.label}
							</button>
						{/each}
					</div>
				{/if}
			</div>
		</div>
	</div>

	<!-- Right: Actions -->
	<div class="flex items-center gap-2">
		{#if onSettingsClick}
			<button
				type="button"
				class="btn-pill btn-pill-ghost btn-pill-icon"
				onclick={onSettingsClick}
			>
				<Settings class="h-5 w-5" />
			</button>
		{/if}
	</div>
</div>

<style>
	.dt2-header {
		background: var(--dbg);
		border-bottom: 1px solid var(--dbd);
		color: var(--dt);
	}

	.dt2-dropdown {
		z-index: var(--bos-z-index-popover, 1001);
		background: var(--dbg);
		border: 1px solid var(--dbd);
		box-shadow: var(--shadow-lg);
	}

	.dt2-dropdown__item {
		color: var(--dt);
		border-radius: 6px;
	}

	.dt2-dropdown__item:hover {
		background: var(--dbg3);
	}
</style>
