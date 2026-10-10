<script lang="ts">
	/**
	 * TableCardView - Table display as cards
	 */
	import { Table2, Star, MoreHorizontal, Database, Upload, Rows3, Columns3 } from 'lucide-svelte';
	import type { TableListItem, TableSource } from '$lib/api/tables/types';

	interface Props {
		tables: TableListItem[];
		onTableClick: (id: string) => void;
		onFavoriteToggle?: (id: string) => void;
	}

	let { tables, onTableClick, onFavoriteToggle }: Props = $props();

	function formatDate(dateStr: string): string {
		const date = new Date(dateStr);
		return date.toLocaleDateString('en-US', {
			month: 'short',
			day: 'numeric'
		});
	}

	function getSourceIcon(source: TableSource) {
		switch (source) {
			case 'import':
				return Upload;
			case 'integration':
				return Database;
			default:
				return Table2;
		}
	}

	function getSourceColor(source: TableSource): string {
		switch (source) {
			case 'import':
				return 'bg-purple-100 text-purple-600';
			case 'integration':
				return 'bg-green-100 text-green-600';
			default:
				return 'bg-blue-100 text-blue-600';
		}
	}
</script>

<div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
	{#each tables as table (table.id)}
		<button
			type="button"
			class="btn-pill btn-pill-secondary group relative flex flex-col p-4 text-left"
			onclick={() => onTableClick(table.id)}
		>
			<!-- Header -->
			<div class="mb-3 flex items-start justify-between">
				<div class="flex items-center gap-3">
					{#if table.icon}
						<span class="text-2xl">{table.icon}</span>
					{:else}
						<div class="flex h-10 w-10 items-center justify-center rounded-lg {getSourceColor(table.source)}">
							<svelte:component this={getSourceIcon(table.source)} class="h-5 w-5" />
						</div>
					{/if}
				</div>

				{#if onFavoriteToggle}
					<button
						type="button"
						class="rounded-lg p-1 text-gray-300 transition-colors hover:text-yellow-400 {table.is_favorite ? 'text-yellow-400' : ''}"
						onclick={(e) => {
							e.stopPropagation();
							onFavoriteToggle(table.id);
						}}
					>
						<Star class="h-5 w-5 {table.is_favorite ? 'fill-current' : ''}" />
					</button>
				{/if}
			</div>

			<!-- Title & Description -->
			<h3 class="mb-1 font-semibold text-gray-900 group-hover:text-blue-600">
				{table.name}
			</h3>
			{#if table.description}
				<p class="mb-3 text-sm text-gray-500 line-clamp-2">
					{table.description}
				</p>
			{:else}
				<div class="mb-3"></div>
			{/if}

			<!-- Stats -->
			<div class="mt-auto flex items-center gap-4 border-t border-gray-100 pt-3 text-sm text-gray-500">
				<div class="flex items-center gap-1">
					<Rows3 class="h-4 w-4" />
					<span>{table.row_count.toLocaleString()} rows</span>
				</div>
				<div class="flex items-center gap-1">
					<Columns3 class="h-4 w-4" />
					<span>{table.column_count} cols</span>
				</div>
			</div>

			<!-- Updated date -->
			<div class="mt-2 text-xs text-gray-400">
				Updated {formatDate(table.updated_at)}
			</div>
		</button>
	{:else}
		<div class="col-span-full flex flex-col items-center py-12">
			<Table2 class="mb-3 h-12 w-12 text-gray-300" />
			<p class="text-gray-500">No tables found</p>
		</div>
	{/each}
</div>
