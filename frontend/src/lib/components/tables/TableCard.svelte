<script lang="ts">
	/**
	 * TableCard - Rich table card with preview and quick actions
	 * Features: Stats preview, column badges, last modified, quick actions
	 */
	import {
		Table2,
		MoreHorizontal,
		Star,
		StarOff,
		Edit3,
		Copy,
		Trash2,
		ExternalLink,
		Grid3X3,
		LayoutGrid,
		Calendar,
		Columns3,
		Database,
		Upload,
		Link
	} from 'lucide-svelte';
	import type { TableListItem } from '$lib/api/tables/types';
	import type { ComponentType, SvelteComponent } from 'svelte';

	type IconComponent = ComponentType<SvelteComponent>;

	interface Props {
		table: TableListItem;
		isFavorite?: boolean;
		onOpen: (id: string) => void;
		onToggleFavorite: (id: string) => void;
		onRename: (id: string) => void;
		onDuplicate: (id: string) => void;
		onDelete: (id: string) => void;
	}

	let {
		table,
		isFavorite = false,
		onOpen,
		onToggleFavorite,
		onRename,
		onDuplicate,
		onDelete
	}: Props = $props();

	let showMenu = $state(false);

	// Get icon and color based on source
	function getSourceIcon(): { icon: IconComponent; color: string; bg: string; label: string } {
		switch (table.source) {
			case 'import':
				return {
					icon: Upload as unknown as IconComponent,
					color: 'text-orange-600',
					bg: 'bg-orange-100',
					label: 'Imported'
				};
			case 'integration':
				return {
					icon: Link as unknown as IconComponent,
					color: 'text-green-600',
					bg: 'bg-green-100',
					label: 'Connected'
				};
			default:
				return {
					icon: Database as unknown as IconComponent,
					color: 'text-blue-600',
					bg: 'bg-blue-100',
					label: 'Custom'
				};
		}
	}

	// Format relative time
	function getRelativeTime(dateString: string): string {
		const date = new Date(dateString);
		const now = new Date();
		const diffMs = now.getTime() - date.getTime();
		const diffMins = Math.floor(diffMs / 60000);
		const diffHours = Math.floor(diffMins / 60);
		const diffDays = Math.floor(diffHours / 24);

		if (diffMins < 1) return 'Just now';
		if (diffMins < 60) return `${diffMins}m ago`;
		if (diffHours < 24) return `${diffHours}h ago`;
		if (diffDays < 7) return `${diffDays}d ago`;
		return date.toLocaleDateString();
	}

	// Get view type icon
	function getViewIcon(viewType: string): IconComponent {
		switch (viewType) {
			case 'kanban':
				return Columns3 as unknown as IconComponent;
			case 'gallery':
				return LayoutGrid as unknown as IconComponent;
			case 'calendar':
				return Calendar as unknown as IconComponent;
			default:
				return Grid3X3 as unknown as IconComponent;
		}
	}

	const sourceInfo = $derived(getSourceIcon());

	function handleMenuClick(e: MouseEvent) {
		e.stopPropagation();
		showMenu = !showMenu;
	}

	function handleAction(action: () => void) {
		action();
		showMenu = false;
	}

	function handleClickOutside() {
		if (showMenu) showMenu = false;
	}
</script>

<svelte:window onclick={handleClickOutside} />

<div
	class="group relative flex flex-col rounded-xl transition-all" style="border: 1px solid var(--dbd); background: var(--dbg); box-shadow: var(--shadow-sm);"
	onmouseenter={(e) => { (e.currentTarget as HTMLElement).style.borderColor = 'var(--dbd)'; (e.currentTarget as HTMLElement).style.boxShadow = 'var(--shadow-md)'; }}
	onmouseleave={(e) => { (e.currentTarget as HTMLElement).style.borderColor = 'var(--dbd)'; (e.currentTarget as HTMLElement).style.boxShadow = 'var(--shadow-sm)'; }}
>
	<!-- Card Header -->
	<div class="flex items-start gap-3 p-4">
		<!-- Clickable area for opening table -->
		<button
			type="button"
			class="btn-pill btn-pill-ghost flex flex-1 items-start gap-3 text-left"
			onclick={() => onOpen(table.id)}
		>
			<!-- Icon -->
			<div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg {sourceInfo.bg}">
				<svelte:component this={sourceInfo.icon} class="h-5 w-5 {sourceInfo.color}" />
			</div>

			<!-- Title and Description -->
			<div class="min-w-0 flex-1">
				<div class="flex items-center gap-2">
				<h3 class="truncate font-medium group-hover:text-blue-600" style="color: var(--dt);">
						{table.name}
					</h3>
					{#if isFavorite}
						<Star class="h-4 w-4 shrink-0 fill-amber-400 text-amber-400" />
					{/if}
				</div>
				{#if table.description}
					<p class="mt-0.5 line-clamp-1 text-sm" style="color: var(--dt2);">{table.description}</p>
				{/if}
			</div>
		</button>

		<!-- Menu Button (separate from clickable area) -->
		<div class="relative shrink-0">
			<button
				type="button"
				class="btn-pill btn-pill-ghost btn-pill-icon opacity-0 group-hover:opacity-100"
				onclick={handleMenuClick}
			>
				<MoreHorizontal class="h-5 w-5" />
			</button>

			<!-- Dropdown Menu -->
			{#if showMenu}
				<div
					class="absolute right-0 top-full z-20 mt-1 w-48 rounded-lg py-1"
					style="border: 1px solid var(--dbd); background: var(--dbg); box-shadow: var(--shadow-lg);"
				>
					<button
						type="button"
						class="dt2-dropdown-item"
						onclick={() => handleAction(() => onOpen(table.id))}
					>
						<ExternalLink class="h-4 w-4" />
						Open table
					</button>
					<button
						type="button"
						class="dt2-dropdown-item"
						onclick={() => handleAction(() => onToggleFavorite(table.id))}
					>
						{#if isFavorite}
							<StarOff class="h-4 w-4" />
							Remove from favorites
						{:else}
							<Star class="h-4 w-4" />
							Add to favorites
						{/if}
					</button>
					<div style="margin: 0.25rem 0; border-top: 1px solid var(--dbd);"></div>
					<button
						type="button"
						class="dt2-dropdown-item"
						onclick={() => handleAction(() => onRename(table.id))}
					>
						<Edit3 class="h-4 w-4" />
						Rename
					</button>
					<button
						type="button"
						class="dt2-dropdown-item"
						onclick={() => handleAction(() => onDuplicate(table.id))}
					>
						<Copy class="h-4 w-4" />
						Duplicate
					</button>
					<div style="margin: 0.25rem 0; border-top: 1px solid var(--dbd);"></div>
					<button
						type="button"
						class="dt2-dropdown-item dt2-dropdown-item--danger"
						onclick={() => handleAction(() => onDelete(table.id))}
					>
						<Trash2 class="h-4 w-4" />
						Delete
					</button>
				</div>
			{/if}
		</div>
	</div>

	<!-- Card Stats -->
	<div class="flex items-center gap-4 px-4 py-3" style="border-top: 1px solid var(--dbd2);">
		<!-- Row Count -->
		<div class="flex items-center gap-1.5 text-xs" style="color: var(--dt3);">
			<Table2 class="h-3.5 w-3.5" />
			<span>{table.row_count.toLocaleString()} rows</span>
		</div>

		<!-- Column Count -->
		<div class="flex items-center gap-1.5 text-xs" style="color: var(--dt3);">
			<Grid3X3 class="h-3.5 w-3.5" />
			<span>{table.columns?.length || 0} columns</span>
		</div>

		<!-- Source Badge -->
		<span
			class="ml-auto rounded-full px-2 py-0.5 text-xs font-medium {sourceInfo.bg} {sourceInfo.color}"
		>
			{sourceInfo.label}
		</span>
	</div>

	<!-- Column Preview (if available) -->
	{#if table.columns && table.columns.length > 0}
		<div class="px-4 py-3" style="border-top: 1px solid var(--dbd2);">
			<div class="flex flex-wrap gap-1">
				{#each table.columns.slice(0, 5) as col}
					<span class="rounded px-2 py-0.5 text-xs" style="background: var(--dbg2); color: var(--dt2);">
						{col.name}
					</span>
				{/each}
				{#if table.columns.length > 5}
					<span class="rounded px-2 py-0.5 text-xs" style="background: var(--dbg2); color: var(--dt4);">
						+{table.columns.length - 5}
					</span>
				{/if}
			</div>
		</div>
	{/if}

	<!-- Card Footer -->
	<div class="flex items-center justify-between px-4 py-2" style="border-top: 1px solid var(--dbd2);">
		<span class="text-xs" style="color: var(--dt4);">
			Updated {getRelativeTime(table.updated_at)}
		</span>

		<!-- Quick Views (if available) -->
		{#if table.views && table.views.length > 0}
			<div class="flex items-center gap-1">
				{#each table.views.slice(0, 3) as view}
					<button
						type="button"
						class="btn-pill btn-pill-ghost btn-pill-icon"
						title="{view.name} ({view.type})"
						onclick={(e) => {
							e.stopPropagation();
							// TODO: Open specific view
							onOpen(table.id);
						}}
					>
						<svelte:component this={getViewIcon(view.type)} class="h-3.5 w-3.5" />
					</button>
				{/each}
				{#if table.views.length > 3}
<span class="text-xs" style="color: var(--dt4);">+{table.views.length - 3}</span>
			{/if}
			</div>
		{/if}
	</div>
</div>

<style>
	.dt2-dropdown-item {
		display: flex;
		width: 100%;
		align-items: center;
		gap: 0.5rem;
		text-align: left;
		padding: 0.5rem 0.75rem;
		border-radius: var(--radius-sm);
		font-size: var(--text-sm);
		color: var(--dt2);
		background: none;
		border: none;
		cursor: pointer;
		transition: background 150ms ease;
	}
	.dt2-dropdown-item:hover {
		background: var(--dbg2);
		color: var(--dt);
	}
	.dt2-dropdown-item--danger {
		color: var(--color-error, #ef4444);
	}
	.dt2-dropdown-item--danger:hover {
		background: color-mix(in srgb, var(--color-error, #ef4444) 10%, var(--dbg));
		color: var(--color-error, #ef4444);
	}
</style>
