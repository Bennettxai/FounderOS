<script lang="ts">
	/**
	 * TableListView - Table display as a list/table
	 */
	import {
		Table2,
		Star,
		MoreHorizontal,
		Trash2,
		Copy,
		Pencil,
		ExternalLink,
		Database,
		Upload,
		Rows3
	} from 'lucide-svelte';
	import type { TableListItem, TableSource } from '$lib/api/tables/types';

	interface Props {
		tables: TableListItem[];
		onTableClick: (id: string) => void;
		onFavoriteToggle?: (id: string) => void;
		onDelete?: (id: string) => void;
		onDuplicate?: (id: string) => void;
	}

	let { tables, onTableClick, onFavoriteToggle, onDelete, onDuplicate }: Props = $props();

	let openMenuId = $state<string | null>(null);

	function formatDate(dateStr: string): string {
		const date = new Date(dateStr);
		return date.toLocaleDateString('en-US', {
			month: 'short',
			day: 'numeric',
			year: 'numeric'
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

	function getSourceLabel(source: TableSource, integration?: string): string {
		switch (source) {
			case 'import':
				return 'Imported';
			case 'integration':
				return integration || 'Integration';
			default:
				return 'Custom';
		}
	}

	function handleMenuToggle(e: MouseEvent, tableId: string) {
		e.stopPropagation();
		openMenuId = openMenuId === tableId ? null : tableId;
	}

	function handleMenuAction(e: MouseEvent, action: () => void) {
		e.stopPropagation();
		action();
		openMenuId = null;
	}

	function handleClickOutside() {
		openMenuId = null;
	}
</script>

<svelte:window onclick={handleClickOutside} />

<div class="overflow-x-auto">
	<table class="w-full">
		<thead class="sticky top-0 bg-gray-50">
			<tr>
				<th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">
					Name
				</th>
				<th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">
					Source
				</th>
				<th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">
					Records
				</th>
				<th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">
					Columns
				</th>
				<th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">
					Last Updated
				</th>
				<th class="w-10 px-4 py-3"></th>
			</tr>
		</thead>
		<tbody class="divide-y divide-gray-100 bg-white">
			{#each tables as table (table.id)}
				<tr
					class="cursor-pointer transition-colors hover:bg-gray-50"
					onclick={() => onTableClick(table.id)}
				>
					<!-- Name -->
					<td class="px-4 py-3">
						<div class="flex items-center gap-3">
							{#if table.icon}
								<span class="text-xl">{table.icon}</span>
							{:else}
								<div class="flex h-8 w-8 items-center justify-center rounded-lg bg-blue-100">
									<Table2 class="h-4 w-4 text-blue-600" />
								</div>
							{/if}
							<div>
								<div class="flex items-center gap-2">
									<span class="font-medium text-gray-900">{table.name}</span>
									{#if table.is_favorite}
										<Star class="h-4 w-4 fill-yellow-400 text-yellow-400" />
									{/if}
								</div>
								{#if table.description}
									<p class="text-sm text-gray-500 line-clamp-1">{table.description}</p>
								{/if}
							</div>
						</div>
					</td>

					<!-- Source -->
					<td class="px-4 py-3">
						<div class="flex items-center gap-1.5">
							<svelte:component this={getSourceIcon(table.source)} class="h-4 w-4 text-gray-400" />
							<span class="text-sm text-gray-600">
								{getSourceLabel(table.source, table.source_integration)}
							</span>
						</div>
					</td>

					<!-- Records -->
					<td class="px-4 py-3">
						<div class="flex items-center gap-1.5">
							<Rows3 class="h-4 w-4 text-gray-400" />
							<span class="text-sm text-gray-600">
								{table.row_count.toLocaleString()}
							</span>
						</div>
					</td>

					<!-- Columns -->
					<td class="px-4 py-3">
						<span class="text-sm text-gray-600">{table.column_count}</span>
					</td>

					<!-- Last Updated -->
					<td class="px-4 py-3">
						<span class="text-sm text-gray-500">{formatDate(table.updated_at)}</span>
					</td>

					<!-- Actions -->
					<td class="px-4 py-3">
						<div class="relative">
							<button
								type="button"
								class="btn-pill btn-pill-ghost btn-pill-icon"
								onclick={(e) => handleMenuToggle(e, table.id)}
							>
								<MoreHorizontal class="h-5 w-5" />
							</button>

							{#if openMenuId === table.id}
								<div
									class="absolute right-0 top-full z-10 mt-1 w-48 rounded-lg border border-gray-200 bg-white py-1 shadow-lg"
								>
									<button
										type="button"
										class="flex w-full items-center gap-2 px-3 py-2 rounded-lg text-sm hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
										onclick={(e) => handleMenuAction(e, () => onTableClick(table.id))}
									>
										<ExternalLink class="h-4 w-4" />
										Open
									</button>

									{#if onFavoriteToggle}
										<button
											type="button"
											class="flex w-full items-center gap-2 px-3 py-2 rounded-lg text-sm hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
											onclick={(e) => handleMenuAction(e, () => onFavoriteToggle(table.id))}
										>
											<Star class="h-4 w-4" />
											{table.is_favorite ? 'Remove from favorites' : 'Add to favorites'}
										</button>
									{/if}

									{#if onDuplicate}
										<button
											type="button"
											class="flex w-full items-center gap-2 px-3 py-2 rounded-lg text-sm hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
											onclick={(e) => handleMenuAction(e, () => onDuplicate(table.id))}
										>
											<Copy class="h-4 w-4" />
											Duplicate
										</button>
									{/if}

									<div class="my-1 border-t border-gray-100"></div>

									{#if onDelete}
										<button
											type="button"
											class="flex w-full items-center gap-2 px-3 py-2 rounded-lg text-sm hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
											onclick={(e) => handleMenuAction(e, () => onDelete(table.id))}
										>
											<Trash2 class="h-4 w-4" />
											Delete
										</button>
									{/if}
								</div>
							{/if}
						</div>
					</td>
				</tr>
			{:else}
				<tr>
					<td colspan="6" class="px-4 py-12 text-center">
						<div class="flex flex-col items-center">
							<Table2 class="mb-3 h-12 w-12 text-gray-300" />
							<p class="text-gray-500">No tables found</p>
						</div>
					</td>
				</tr>
			{/each}
		</tbody>
	</table>
</div>
