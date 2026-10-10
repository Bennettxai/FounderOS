<script lang="ts">
  import { Search, X } from 'lucide-svelte';
  import { categoryLabels } from '$lib/stores/agents';

  interface Props {
    searchQuery: string;
    selectedCategory: string | null;
    selectedStatus: 'active' | 'inactive' | null;
    sortBy: 'name' | 'created' | 'usage';
    categories: readonly string[];
    hasActiveFilters: boolean;
    onSearchInput: (e: Event) => void;
    onClearSearch: () => void;
    onSortChange: (e: Event) => void;
    onCategoryChange: (cat: string | null) => void;
    onStatusChange: (status: 'active' | 'inactive' | null) => void;
    onClearFilters: () => void;
  }

  let {
    searchQuery,
    selectedCategory,
    selectedStatus,
    sortBy,
    categories,
    hasActiveFilters,
    onSearchInput,
    onClearSearch,
    onSortChange,
    onCategoryChange,
    onStatusChange,
    onClearFilters,
  }: Props = $props();

  const statusOptions = [
    { value: null as null, label: 'All' },
    { value: 'active' as const, label: 'Active' },
    { value: 'inactive' as const, label: 'Inactive' },
  ];
</script>

<div class="afb-filters">
  <div class="afb-filters__row">
    <div class="afb-search">
      <Search class="afb-search__icon" size={14} strokeWidth={2} aria-hidden="true" />
      <input
        type="text"
        value={searchQuery}
        oninput={onSearchInput}
        placeholder="Search agents..."
        class="afb-search__input"
        aria-label="Search agents"
      />
      {#if searchQuery}
        <button class="afb-search__clear" onclick={onClearSearch} aria-label="Clear search" tabindex="-1">
          <X size={12} strokeWidth={2} aria-hidden="true" />
        </button>
      {/if}
    </div>
    <select value={sortBy} onchange={onSortChange} class="afb-sort" aria-label="Sort agents">
      <option value="name">Name A–Z</option>
      <option value="created">Newest first</option>
      <option value="usage">Most used</option>
    </select>
  </div>

  <div class="afb-filters__pills-row">
    <div class="afb-pill-group" role="group" aria-label="Filter by category">
      <button
        onclick={() => onCategoryChange(null)}
        class="afb-pill"
        class:afb-pill--active={selectedCategory === null}
      >All</button>
      {#each categories as cat}
        <button
          onclick={() => onCategoryChange(selectedCategory === cat ? null : cat)}
          class="afb-pill"
          class:afb-pill--active={selectedCategory === cat}
        >{categoryLabels[cat] || cat}</button>
      {/each}
    </div>

    <span class="afb-pill-divider" aria-hidden="true"></span>

    <div class="afb-pill-group" role="group" aria-label="Filter by status">
      {#each statusOptions as opt}
        <button
          onclick={() => onStatusChange(opt.value)}
          class="afb-pill afb-pill--status"
          class:afb-pill--active={selectedStatus === opt.value}
        >{opt.label}</button>
      {/each}
    </div>

    {#if hasActiveFilters}
      <button onclick={onClearFilters} class="afb-pill afb-pill--clear" aria-label="Clear all filters">
        <X size={10} strokeWidth={2} aria-hidden="true" />
        Clear
      </button>
    {/if}
  </div>
</div>

<style>
  .afb-filters {
    background: var(--dbg, #fff);
    border: 1px solid var(--dbd, #e0e0e0);
    border-radius: 0.75rem;
    padding: 0.75rem 1rem;
    margin-bottom: 0.625rem;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }
  .afb-filters__row {
    display: flex;
    gap: 0.5rem;
    align-items: center;
  }

  .afb-search {
    flex: 1;
    position: relative;
  }
  :global(.afb-search__icon) {
    position: absolute;
    left: 0.75rem;
    top: 50%;
    transform: translateY(-50%);
    color: var(--dt4, #bbb);
    pointer-events: none;
  }
  .afb-search__input {
    width: 100%;
    padding: 0.5625rem 2rem 0.5625rem 2.125rem;
    font-size: 0.875rem;
    border: 1px solid var(--dbd, #e0e0e0);
    border-radius: 0.5rem;
    background: var(--dbg, #fff);
    color: var(--dt, #111);
    outline: none;
    transition: border-color 0.15s, box-shadow 0.15s;
    box-sizing: border-box;
  }
  .afb-search__input::placeholder { color: var(--dt4, #bbb); }
  .afb-search__input:focus {
    border-color: var(--dt2, #555);
    box-shadow: 0 0 0 3px rgba(0,0,0,0.06);
  }
  .afb-search__clear {
    position: absolute;
    right: 0.6rem;
    top: 50%;
    transform: translateY(-50%);
    width: 20px;
    height: 20px;
    border-radius: 4px;
    border: none;
    background: transparent;
    color: var(--dt3, #888);
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: background 0.1s, color 0.1s;
  }
  .afb-search__clear:hover {
    background: var(--dbg2, #f5f5f5);
    color: var(--dt, #111);
  }

  .afb-sort {
    padding: 0.5rem 2rem 0.5rem 0.75rem;
    font-size: 0.8125rem;
    font-family: inherit;
    border: 1px solid var(--dbd, #e0e0e0);
    border-radius: 0.5rem;
    background-color: var(--dbg, #fff);
    background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 24 24' fill='none' stroke='%23888888' stroke-width='2.5' stroke-linecap='round' stroke-linejoin='round'%3E%3Cpolyline points='6 9 12 15 18 9'/%3E%3C/svg%3E");
    background-repeat: no-repeat;
    background-position: right 0.625rem center;
    color: var(--dt, #111);
    outline: none;
    cursor: pointer;
    white-space: nowrap;
    appearance: none;
    -webkit-appearance: none;
    transition: border-color 0.15s;
  }
  .afb-sort:focus { border-color: var(--dt2, #555); }

  .afb-filters__pills-row {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.3rem;
  }
  .afb-pill-group {
    display: flex;
    flex-wrap: wrap;
    gap: 0.3rem;
  }
  .afb-pill-divider {
    width: 1px;
    height: 16px;
    background: var(--dbd, #e0e0e0);
    flex-shrink: 0;
    margin: 0 0.1rem;
    align-self: center;
  }

  .afb-pill {
    padding: 0.25rem 0.6875rem;
    font-size: 0.75rem;
    font-weight: 500;
    border-radius: 9999px;
    border: 1px solid var(--dbd, #e0e0e0);
    cursor: pointer;
    transition: all 0.12s;
    background: var(--dbg, #fff);
    color: var(--dt2, #555);
    line-height: 1.5;
    white-space: nowrap;
  }
  .afb-pill:hover {
    background: var(--dbg2, #f5f5f5);
    border-color: var(--dt3, #999);
    color: var(--dt, #111);
  }
  .afb-pill--active {
    background: var(--dt, #111);
    color: var(--dbg, #fff);
    font-weight: 600;
    border-color: var(--dt, #111);
  }
  .afb-pill--active:hover {
    background: var(--dt2, #333);
    border-color: var(--dt2, #333);
  }
  .afb-pill--status {
    color: var(--dt3, #888);
  }
  .afb-pill--status.afb-pill--active {
    background: var(--dt, #111);
    color: var(--dbg, #fff);
  }
  .afb-pill--clear {
    display: inline-flex;
    align-items: center;
    gap: 0.25rem;
    color: var(--dt3, #888);
    border-color: transparent;
    background: transparent;
    margin-left: 0.1rem;
  }
  .afb-pill--clear:hover {
    background: rgba(239, 68, 68, 0.07);
    border-color: rgba(239, 68, 68, 0.25);
    color: var(--bos-status-error, #ef4444);
  }
</style>
