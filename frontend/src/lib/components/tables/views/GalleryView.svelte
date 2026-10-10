<script lang="ts">
	/**
	 * GalleryView - Card gallery view with image covers
	 * Features: Card grid, cover images from attachment columns, card previews
	 */
	import { Plus, Image, MoreHorizontal } from 'lucide-svelte';
	import type { Column, Row } from '$lib/api/tables/types';

	interface Props {
		columns: Column[];
		rows: Row[];
		coverColumnId: string | null;
		onCardClick: (rowId: string) => void;
		onAddCard: () => void;
	}

	let { columns, rows, coverColumnId, onCardClick, onAddCard }: Props = $props();

	// Get primary display column (first text column or first column)
	const primaryColumn = $derived(
		columns.find((c) => c.is_primary) || columns.find((c) => c.type === 'text') || columns[0]
	);

	// Get secondary column for subtitle
	const secondaryColumn = $derived(
		columns.find((c) => c.id !== primaryColumn?.id && c.type === 'text' && !c.is_hidden)
	);

	// Get preview fields (first few columns excluding cover and primary)
	const previewColumns = $derived(
		columns
			.filter((c) => c.id !== coverColumnId && c.id !== primaryColumn?.id && !c.is_hidden)
			.slice(0, 4)
	);

	// Get display value for a row
	function getDisplayValue(row: Row, column: Column | undefined): string {
		if (!column) return '';
		const value = row.data[column.id];
		if (value === null || value === undefined) return '';
		if (typeof value === 'boolean') return value ? 'Yes' : 'No';
		return String(value);
	}

	// Get cover image URL
	function getCoverImage(row: Row): string | null {
		if (!coverColumnId) return null;
		const value = row.data[coverColumnId];
		if (!value) return null;

		// Handle attachment column value (could be array or single URL)
		if (Array.isArray(value) && value.length > 0) {
			const first = value[0];
			if (typeof first === 'string') return first;
			if (first && typeof first === 'object' && 'url' in first) return first.url as string;
		}
		if (typeof value === 'string') return value;
		return null;
	}

	// Format column value for display
	function formatValue(value: unknown, column: Column): string {
		if (value === null || value === undefined) return '-';

		switch (column.type) {
			case 'checkbox':
				return value ? 'Yes' : 'No';
			case 'date':
			case 'datetime':
				if (value instanceof Date) return value.toLocaleDateString();
				if (typeof value === 'string') return new Date(value).toLocaleDateString();
				return String(value);
			case 'currency':
				const code = column.options?.currency_code || 'USD';
				return new Intl.NumberFormat('en-US', { style: 'currency', currency: code }).format(
					Number(value)
				);
			case 'percent':
				return `${Number(value)}%`;
			case 'single_select':
				// Find choice label
				const choice = column.options?.choices?.find((c) => c.id === value);
				return choice?.label || String(value);
			case 'multi_select':
				if (Array.isArray(value)) {
					return value
						.map((v) => {
							const c = column.options?.choices?.find((ch) => ch.id === v);
							return c?.label || v;
						})
						.join(', ');
				}
				return String(value);
			default:
				return String(value);
		}
	}

	// Get select choice color
	function getChoiceColor(value: unknown, column: Column): string | null {
		if (column.type !== 'single_select') return null;
		const choice = column.options?.choices?.find((c) => c.id === value);
		return choice?.color || null;
	}
</script>

<div class="h-full overflow-auto p-6">
	<div class="grid grid-cols-1 gap-6 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5">
		{#each rows as row (row.id)}
			{@const coverUrl = getCoverImage(row)}
			<div
				class="dt2-gallery-card group relative flex flex-col overflow-hidden rounded-xl text-left transition-all hover:shadow-md"
			>
				<!-- Cover Image -->
				<button
					type="button"
					class="btn-pill btn-pill-soft relative aspect-video w-full overflow-hidden"
					onclick={() => onCardClick(row.id)}
				>
					{#if coverUrl}
						<img src={coverUrl} alt="" class="h-full w-full object-cover" />
					{:else}
						<div class="flex h-full w-full items-center justify-center">
							<Image class="h-10 w-10" style="color: var(--dt3);" />
						</div>
					{/if}
				</button>

				<!-- More Options Button (outside clickable area) -->
				<div class="absolute right-2 top-2 opacity-0 transition-opacity group-hover:opacity-100">
					<button
						type="button"
						class="btn-pill btn-pill-ghost btn-pill-icon bg-white/90 shadow-lg hover:bg-white"
						onclick={() => {
							// TODO: Open card menu
						}}
					>
						<MoreHorizontal class="h-4 w-4" />
					</button>
				</div>

				<!-- Card Content (clickable) -->
				<button
					type="button"
					class="btn-pill btn-pill-ghost flex flex-1 flex-col p-4 text-left"
					onclick={() => onCardClick(row.id)}
				>
					<!-- Primary Value (Title) -->
					<h3 class="mb-1 truncate font-medium" style="color: var(--dt);">
						{getDisplayValue(row, primaryColumn) || 'Untitled'}
					</h3>

					<!-- Secondary Value (Subtitle) -->
					{#if secondaryColumn}
						{@const secondaryValue = getDisplayValue(row, secondaryColumn)}
						{#if secondaryValue}
							<p class="mb-3 truncate text-sm" style="color: var(--dt2);">
								{secondaryValue}
							</p>
						{/if}
					{/if}

					<!-- Preview Fields -->
					{#if previewColumns.length > 0}
						<div class="mt-auto space-y-2 pt-3" style="border-top: 1px solid var(--dbd2);">
							{#each previewColumns as col}
								{@const value = row.data[col.id]}
								{#if value !== null && value !== undefined}
									<div class="flex items-center gap-2 text-xs">
										<span class="shrink-0" style="color: var(--dt3);">{col.name}:</span>
										{#if col.type === 'single_select' && getChoiceColor(value, col)}
											<span
												class="truncate rounded-full px-2 py-0.5 text-white"
												style="background-color: {getChoiceColor(value, col)}"
											>
												{formatValue(value, col)}
											</span>
										{:else}
											<span class="truncate" style="color: var(--dt2);">
												{formatValue(value, col)}
											</span>
										{/if}
									</div>
								{/if}
							{/each}
						</div>
					{/if}
				</button>
			</div>
		{/each}

		<!-- Add Card Button -->
		<button
			type="button"
			class="btn-pill btn-pill-soft flex aspect-[4/3] flex-col items-center justify-center border-2 border-dashed"
			onclick={onAddCard}
		>
			<Plus class="mb-2 h-8 w-8" />
			<span class="text-sm font-medium">Add Card</span>
		</button>
	</div>

	<!-- Empty State -->
	{#if rows.length === 0}
		<div class="flex flex-col items-center justify-center py-16">
			<div class="mb-4 rounded-full p-4" style="background: var(--dbg2);">
				<Image class="h-10 w-10" style="color: var(--dt3);" />
			</div>
			<h3 class="mb-1 text-lg font-medium" style="color: var(--dt);">No items yet</h3>
			<p class="mb-4 text-sm" style="color: var(--dt2);">Create your first item to get started</p>
			<button
				type="button"
				class="btn-pill btn-pill-primary btn-pill-sm flex items-center gap-2"
				onclick={onAddCard}
			>
				<Plus class="h-4 w-4" />
				Add First Item
			</button>
		</div>
	{/if}
</div>

<style>
	.dt2-gallery-card {
		background: var(--dbg);
		border: 1px solid var(--dbd);
		box-shadow: var(--shadow-sm);
	}

	.dt2-gallery-card:hover {
		border-color: var(--dt3);
	}
</style>
