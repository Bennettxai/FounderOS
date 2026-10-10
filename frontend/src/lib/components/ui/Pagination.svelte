<script lang="ts">
	import { ChevronLeft, ChevronRight, ChevronsLeft, ChevronsRight } from 'lucide-svelte';

	interface Props {
		page: number;
		pageSize: number;
		total: number;
		pageSizeOptions?: number[];
		onPageChange?: (newPage: number) => void;
		onPageSizeChange?: (newPageSize: number) => void;
	}

	let {
		page,
		pageSize,
		total,
		pageSizeOptions = [10, 20, 50, 100],
		onPageChange,
		onPageSizeChange
	}: Props = $props();

	const totalPages = $derived(Math.max(1, Math.ceil(total / pageSize)));
	const startItem = $derived(total === 0 ? 0 : (page - 1) * pageSize + 1);
	const endItem = $derived(Math.min(page * pageSize, total));

	const isFirstPage = $derived(page === 1);
	const isLastPage = $derived(page >= totalPages);

	function goToPage(newPage: number) {
		if (newPage >= 1 && newPage <= totalPages && onPageChange) {
			onPageChange(newPage);
		}
	}

	function goToFirstPage() {
		goToPage(1);
	}

	function goToPreviousPage() {
		goToPage(page - 1);
	}

	function goToNextPage() {
		goToPage(page + 1);
	}

	function goToLastPage() {
		goToPage(totalPages);
	}

	function handlePageSizeChange(event: Event) {
		const select = event.currentTarget as HTMLSelectElement;
		const newSize = Number(select.value);
		if (!isNaN(newSize) && newSize > 0 && onPageSizeChange) {
			onPageSizeChange(newSize);
		}
	}

	// Determine which page numbers to show
	const pageNumbers = $derived.by(() => {
		const pages: (number | string)[] = [];
		const maxPages = 7;
		const siblingCount = 1;

		if (totalPages <= maxPages) {
			// Show all pages if 7 or fewer
			for (let i = 1; i <= totalPages; i++) {
				pages.push(i);
			}
		} else {
			// Always show first page
			pages.push(1);

			// Calculate left and right range around current page
			const leftStart = Math.max(2, page - siblingCount);
			const rightEnd = Math.min(totalPages - 1, page + siblingCount);

			// Add left ellipsis if needed
			if (leftStart > 2) {
				pages.push('...');
			}

			// Add pages around current page
			for (let i = leftStart; i <= rightEnd; i++) {
				pages.push(i);
			}

			// Add right ellipsis if needed
			if (rightEnd < totalPages - 1) {
				pages.push('...');
			}

			// Always show last page
			pages.push(totalPages);
		}

		return pages;
	});
</script>

{#if total > 0}
	<div class="bos-pagination">
		<!-- Left: results info + page size selector -->
		<div class="bos-pagination__info">
			<span>
				Showing
				<span class="bos-pagination__info-strong">{startItem}</span>
				–
				<span class="bos-pagination__info-strong">{endItem}</span>
				of
				<span class="bos-pagination__info-strong">{total}</span>
				results
			</span>

			{#if onPageSizeChange}
				<label class="bos-pagination__size-label">
					<span class="sr-only">Rows per page</span>
					<span aria-hidden="true">Per page:</span>
					<select
						value={pageSize}
						onchange={handlePageSizeChange}
						class="bos-pagination__select"
						aria-label="Rows per page"
					>
						{#each pageSizeOptions as option (option)}
							<option value={option} selected={option === pageSize}>{option}</option>
						{/each}
					</select>
				</label>
			{/if}
		</div>

		<!-- Right: page navigation -->
		<nav aria-label="Pagination" class="bos-pagination__nav">
			<!-- First page -->
			<button
				type="button"
				onclick={goToFirstPage}
				disabled={isFirstPage}
				class="bos-pagination__btn"
				aria-label="First page"
			>
				<ChevronsLeft style="width: 16px; height: 16px;" aria-hidden="true" />
			</button>

			<!-- Previous page -->
			<button
				type="button"
				onclick={goToPreviousPage}
				disabled={isFirstPage}
				class="bos-pagination__btn"
				aria-label="Previous page"
			>
				<ChevronLeft style="width: 16px; height: 16px;" aria-hidden="true" />
			</button>

			<!-- Page numbers -->
			<ol class="bos-pagination__pages" aria-label="Page numbers">
				{#each pageNumbers as pageNum, idx (idx)}
					<li>
						{#if pageNum === '...'}
							<span class="bos-pagination__ellipsis" aria-hidden="true">•••</span>
						{:else}
							<button
								type="button"
								onclick={() => goToPage(pageNum as number)}
								class="bos-pagination__btn bos-pagination__page-btn"
								class:bos-pagination__page-btn--active={page === pageNum}
								aria-label="Go to page {pageNum}"
								aria-current={page === pageNum ? 'page' : undefined}
							>
								{pageNum}
							</button>
						{/if}
					</li>
				{/each}
			</ol>

			<!-- Next page -->
			<button
				type="button"
				onclick={goToNextPage}
				disabled={isLastPage}
				class="bos-pagination__btn"
				aria-label="Next page"
			>
				<ChevronRight style="width: 16px; height: 16px;" aria-hidden="true" />
			</button>

			<!-- Last page -->
			<button
				type="button"
				onclick={goToLastPage}
				disabled={isLastPage}
				class="bos-pagination__btn"
				aria-label="Last page"
			>
				<ChevronsRight style="width: 16px; height: 16px;" aria-hidden="true" />
			</button>
		</nav>
	</div>
{/if}

<style>
	.bos-pagination {
		display: flex;
		flex-direction: column;
		gap: 12px;
		align-items: center;
		justify-content: space-between;
		padding: 16px 8px;
		margin-top: 24px;
		padding-top: 16px;
		border-top: 1px solid var(--dbd2);
	}

	@media (min-width: 640px) {
		.bos-pagination {
			flex-direction: row;
			gap: 0;
			padding-left: 0;
			padding-right: 0;
		}
	}

	.bos-pagination__info {
		display: flex;
		align-items: center;
		gap: 16px;
		font-size: 13px;
		color: var(--dt3);
	}

	.bos-pagination__info-strong {
		font-weight: 500;
		color: var(--dt2);
	}

	.bos-pagination__size-label {
		display: flex;
		align-items: center;
		gap: 6px;
	}

	.bos-pagination__select {
		height: 32px;
		border-radius: 6px;
		border: 1px solid var(--dbd);
		background-color: var(--dbg);
		color: var(--dt2);
		font-size: 13px;
		padding: 0 24px 0 6px;
		cursor: pointer;
		appearance: none;
		outline: none;
	}

	.bos-pagination__select:focus {
		outline: 2px solid var(--color-primary, #6366f1);
		border-color: transparent;
	}

	.bos-pagination__nav {
		display: flex;
		align-items: center;
		gap: 4px;
	}

	.bos-pagination__btn {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		height: 32px;
		width: 32px;
		border-radius: 6px;
		border: 1px solid var(--dbd);
		background-color: var(--dbg);
		color: var(--dt2);
		cursor: pointer;
		transition: background-color 0.15s, color 0.15s;
	}

	.bos-pagination__btn:hover:not(:disabled) {
		background-color: var(--dbg2);
		color: var(--dt);
	}

	.bos-pagination__btn:disabled {
		cursor: not-allowed;
		opacity: 0.4;
	}

	.bos-pagination__pages {
		display: flex;
		align-items: center;
		gap: 2px;
		list-style: none;
		margin: 0;
		padding: 0;
	}

	.bos-pagination__page-btn {
		font-size: 13px;
		font-weight: 500;
	}

	.bos-pagination__page-btn--active {
		border-color: var(--color-primary, #6366f1);
		background-color: var(--color-primary, #6366f1);
		color: #fff;
	}

	.bos-pagination__page-btn--active:hover:not(:disabled) {
		background-color: var(--color-primary, #6366f1);
		color: #fff;
	}

	.bos-pagination__ellipsis {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		height: 32px;
		width: 32px;
		font-size: 13px;
		color: var(--dt4);
	}

	.sr-only {
		position: absolute;
		width: 1px;
		height: 1px;
		padding: 0;
		margin: -1px;
		overflow: hidden;
		clip: rect(0, 0, 0, 0);
		white-space: nowrap;
		border: 0;
	}
</style>
