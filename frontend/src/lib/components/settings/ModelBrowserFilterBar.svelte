<script lang="ts">
  import type { ModelCapability } from '$lib/stores/aiSettings';
  import { availableModels, capabilityInfo } from '$lib/stores/aiSettings';

  interface Props {
    modelSearchQuery: string;
    selectedCapabilityFilters: ModelCapability[];
    selectedProviderFilter: 'all' | 'local' | 'cloud';
    modelSortBy: 'recommended' | 'name' | 'size' | 'downloads';
    showOnlyInstalled: boolean;
    onSearchChange: (q: string) => void;
    onCapabilityFiltersChange: (filters: ModelCapability[]) => void;
    onProviderFilterChange: (f: 'all' | 'local' | 'cloud') => void;
    onSortByChange: (s: 'recommended' | 'name' | 'size' | 'downloads') => void;
    onShowOnlyInstalledChange: (v: boolean) => void;
  }

  let {
    modelSearchQuery,
    selectedCapabilityFilters,
    selectedProviderFilter,
    modelSortBy,
    showOnlyInstalled,
    onSearchChange,
    onCapabilityFiltersChange,
    onProviderFilterChange,
    onSortByChange,
    onShowOnlyInstalledChange,
  }: Props = $props();

  let showSourceDropdown = $state(false);
  let showFiltersDropdown = $state(false);

  function handleClickOutside(e: MouseEvent) {
    const target = e.target as HTMLElement;
    if (!target.closest('.filter-dropdown-wrapper')) {
      showSourceDropdown = false;
      showFiltersDropdown = false;
    }
  }

  function toggleCapability(cap: ModelCapability) {
    if (selectedCapabilityFilters.includes(cap)) {
      onCapabilityFiltersChange(selectedCapabilityFilters.filter((c) => c !== cap));
    } else {
      onCapabilityFiltersChange([...selectedCapabilityFilters, cap]);
    }
  }
</script>

<svelte:document onclick={handleClickOutside} />

<div class="browser-controls">
  <div class="compact-search">
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.35-4.35"/></svg>
    <input type="text" value={modelSearchQuery} oninput={(e) => onSearchChange((e.target as HTMLInputElement).value)} placeholder="Search..." />
    {#if modelSearchQuery}
      <button class="btn-pill btn-pill-ghost clear-search" onclick={() => onSearchChange('')} aria-label="Clear search">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 6L6 18M6 6l12 12"/></svg>
      </button>
    {:else}
      <span class="search-shortcut">⌘K</span>
    {/if}
  </div>

  <div class="filter-dropdown-wrapper">
    <button class="filter-dropdown-btn" class:active={selectedProviderFilter !== 'all'} onclick={() => { showSourceDropdown = !showSourceDropdown; showFiltersDropdown = false; }}>
      {#if selectedProviderFilter === 'local'}
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14"><rect x="2" y="2" width="20" height="8" rx="2"/><rect x="2" y="14" width="20" height="8" rx="2"/><circle cx="6" cy="6" r="1" fill="currentColor"/><circle cx="6" cy="18" r="1" fill="currentColor"/></svg>
        Local
      {:else if selectedProviderFilter === 'cloud'}
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14"><path d="M18 10h-1.26A8 8 0 109 20h9a5 5 0 000-10z"/></svg>
        Cloud
      {:else}
        Source
      {/if}
      <span class="dropdown-count">{selectedProviderFilter === 'all' ? availableModels.length : availableModels.filter(m => m.provider === selectedProviderFilter).length}</span>
      <svg class="dropdown-chevron" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="m6 9 6 6 6-6"/></svg>
    </button>
    {#if showSourceDropdown}
      <div class="filter-dropdown-menu" role="menu">
        <button class="dropdown-item" class:selected={selectedProviderFilter === 'all'} onclick={() => { onProviderFilterChange('all'); showSourceDropdown = false; }}>
          All <span class="item-count">{availableModels.length}</span>
        </button>
        <button class="dropdown-item" class:selected={selectedProviderFilter === 'local'} onclick={() => { onProviderFilterChange('local'); showSourceDropdown = false; }}>
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14"><rect x="2" y="2" width="20" height="8" rx="2"/><rect x="2" y="14" width="20" height="8" rx="2"/><circle cx="6" cy="6" r="1" fill="currentColor"/><circle cx="6" cy="18" r="1" fill="currentColor"/></svg>
          Local <span class="item-count">{availableModels.filter(m => m.provider === 'local').length}</span>
        </button>
        <button class="dropdown-item" class:selected={selectedProviderFilter === 'cloud'} onclick={() => { onProviderFilterChange('cloud'); showSourceDropdown = false; }}>
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14"><path d="M18 10h-1.26A8 8 0 109 20h9a5 5 0 000-10z"/></svg>
          Cloud <span class="item-count">{availableModels.filter(m => m.provider === 'cloud').length}</span>
        </button>
      </div>
    {/if}
  </div>

  <div class="filter-dropdown-wrapper">
    <button class="filter-dropdown-btn" class:active={selectedCapabilityFilters.length > 0} onclick={() => { showFiltersDropdown = !showFiltersDropdown; showSourceDropdown = false; }}>
      Filters
      {#if selectedCapabilityFilters.length > 0}
        <span class="dropdown-count">{selectedCapabilityFilters.length}</span>
      {/if}
      <svg class="dropdown-chevron" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="m6 9 6 6 6-6"/></svg>
    </button>
    {#if showFiltersDropdown}
      <div class="filter-dropdown-menu capabilities-menu" role="menu">
        {#each Object.entries(capabilityInfo) as [cap, info]}
          <label class="dropdown-checkbox-item">
            <input type="checkbox" checked={selectedCapabilityFilters.includes(cap as ModelCapability)} onchange={() => toggleCapability(cap as ModelCapability)} />
            <svg class="cap-icon-svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d={info.iconPath}/></svg>
            {info.label}
          </label>
        {/each}
        {#if selectedCapabilityFilters.length > 0}
          <button class="dropdown-clear" onclick={() => onCapabilityFiltersChange([])}>Clear all</button>
        {/if}
      </div>
    {/if}
  </div>

  {#if selectedCapabilityFilters.length > 0}
    <div class="active-filter-chips">
      {#each selectedCapabilityFilters as cap}
        <button class="filter-chip" onclick={() => onCapabilityFiltersChange(selectedCapabilityFilters.filter(c => c !== cap))}>
          {capabilityInfo[cap].label}
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 6L6 18M6 6l12 12"/></svg>
        </button>
      {/each}
    </div>
  {/if}

  <label class="compact-toggle">
    <input type="checkbox" checked={showOnlyInstalled} onchange={(e) => onShowOnlyInstalledChange((e.target as HTMLInputElement).checked)} />
    <span class="toggle-slider"></span>
    <span class="toggle-label">Installed</span>
  </label>

  <select value={modelSortBy} onchange={(e) => onSortByChange((e.target as HTMLSelectElement).value as 'recommended' | 'name' | 'size' | 'downloads')} class="compact-sort-select">
    <option value="recommended">Recommended</option>
    <option value="name">Name</option>
    <option value="size">Size</option>
    <option value="downloads">Downloads</option>
  </select>
</div>

<style>
  .browser-controls {
    display: flex; flex-direction: row; align-items: center; gap: 10px;
    padding: 10px 14px; background: var(--color-bg); border: 1px solid var(--color-border);
    border-radius: 10px; position: sticky; top: 0; z-index: 50;
    box-shadow: 0 2px 12px rgba(0,0,0,0.06); min-height: 52px;
  }
  :global(.dark) .browser-controls { background: var(--dbg); box-shadow: 0 2px 12px rgba(0,0,0,0.3); border-color: var(--dbd); }
  .compact-search { display: flex; align-items: center; gap: 8px; padding: 6px 12px; background: var(--color-bg-secondary); border: 1px solid var(--color-border); border-radius: 8px; min-width: 160px; max-width: 220px; transition: all 0.2s ease; }
  .compact-search:focus-within { border-color: var(--color-primary); box-shadow: 0 0 0 2px rgba(59,130,246,0.12); min-width: 200px; }
  :global(.dark) .compact-search { background: rgba(255,255,255,0.04); }
  .compact-search svg { width: 15px; height: 15px; color: var(--color-text-muted); flex-shrink: 0; }
  .compact-search input { flex: 1; min-width: 0; background: none; border: none; font-size: 13px; color: var(--color-text); outline: none; }
  .compact-search input::placeholder { color: var(--color-text-muted); }
  .clear-search { display: flex; align-items: center; justify-content: center; width: 18px; height: 18px; background: var(--color-bg-tertiary); border: none; border-radius: 4px; cursor: pointer; color: var(--color-text-muted); padding: 0; }
  .clear-search:hover { background: var(--color-bg-secondary); color: var(--color-text); }
  .clear-search svg { width: 12px; height: 12px; }
  .search-shortcut { font-size: 10px; font-weight: 500; color: var(--color-text-muted); background: var(--color-bg-tertiary); padding: 2px 6px; border-radius: 4px; border: 1px solid var(--color-border); opacity: 0.6; }
  .filter-dropdown-wrapper { position: relative; }
  .filter-dropdown-btn { display: flex; align-items: center; gap: 6px; padding: 6px 10px; background: var(--color-bg-secondary); border: 1px solid var(--color-border); border-radius: 8px; font-size: 13px; font-weight: 500; color: var(--color-text-secondary); cursor: pointer; transition: all 0.15s ease; white-space: nowrap; }
  .filter-dropdown-btn:hover { border-color: var(--color-border-hover); color: var(--color-text); }
  .filter-dropdown-btn.active { background: var(--color-bg-tertiary); border-color: var(--color-border-hover); color: var(--color-text); }
  :global(.dark) .filter-dropdown-btn { background: rgba(255,255,255,0.04); }
  :global(.dark) .filter-dropdown-btn.active { background: rgba(255,255,255,0.08); }
  .filter-dropdown-btn svg:not(.dropdown-chevron) { width: 14px; height: 14px; }
  .dropdown-chevron { width: 12px; height: 12px; opacity: 0.5; margin-left: 2px; }
  .dropdown-count { font-size: 11px; font-weight: 600; padding: 1px 5px; background: rgba(0,0,0,0.06); border-radius: 8px; color: var(--color-text-muted); min-width: 18px; text-align: center; }
  :global(.dark) .dropdown-count { background: rgba(255,255,255,0.1); }
  .filter-dropdown-menu { position: absolute; top: calc(100% + 6px); left: 0; min-width: 160px; background: var(--color-bg); border: 1px solid var(--color-border); border-radius: 10px; box-shadow: 0 8px 24px rgba(0,0,0,0.12); z-index: 60; padding: 6px; animation: dropdownFadeIn 0.15s ease; }
  :global(.dark) .filter-dropdown-menu { background: var(--dbg); border-color: var(--dbd); box-shadow: 0 8px 24px rgba(0,0,0,0.4); }
  @keyframes dropdownFadeIn { from { opacity: 0; transform: translateY(-4px); } to { opacity: 1; transform: translateY(0); } }
  .dropdown-item { display: flex; align-items: center; gap: 8px; width: 100%; padding: 8px 10px; background: transparent; border: none; border-radius: 6px; font-size: 13px; color: var(--color-text); cursor: pointer; text-align: left; transition: background 0.1s ease; }
  .dropdown-item:hover { background: var(--color-bg-secondary); }
  .dropdown-item.selected { background: var(--color-bg-tertiary); font-weight: 500; }
  .dropdown-item svg { width: 14px; height: 14px; opacity: 0.7; }
  .item-count { margin-left: auto; font-size: 11px; color: var(--color-text-muted); font-weight: 500; }
  .capabilities-menu { min-width: 180px; }
  .dropdown-checkbox-item { display: flex; align-items: center; gap: 8px; padding: 8px 10px; border-radius: 6px; font-size: 13px; color: var(--color-text); cursor: pointer; transition: background 0.1s ease; }
  .dropdown-checkbox-item:hover { background: var(--color-bg-secondary); }
  .dropdown-checkbox-item input[type="checkbox"] { width: 14px; height: 14px; accent-color: var(--bos-success-color, #34c759); cursor: pointer; }
  .dropdown-clear { display: block; width: 100%; padding: 8px 10px; margin-top: 4px; background: transparent; border: none; border-top: 1px solid var(--color-border); font-size: 12px; color: var(--color-text-muted); cursor: pointer; text-align: center; }
  .dropdown-clear:hover { color: var(--color-text); }
  .cap-icon-svg { width: 14px; height: 14px; flex-shrink: 0; }
  .active-filter-chips { display: flex; align-items: center; gap: 6px; flex-wrap: wrap; }
  .filter-chip { display: flex; align-items: center; gap: 4px; padding: 4px 8px; background: var(--color-bg-tertiary); border: 1px solid var(--color-border); border-radius: 6px; font-size: 11px; font-weight: 500; color: var(--color-text); cursor: pointer; transition: all 0.15s ease; }
  .filter-chip:hover { background: var(--color-bg-secondary); border-color: var(--color-border-hover); }
  .filter-chip svg { width: 10px; height: 10px; }
  .compact-toggle { display: flex; align-items: center; gap: 8px; cursor: pointer; font-size: 12px; color: var(--color-text-secondary); white-space: nowrap; }
  .compact-toggle input { position: absolute; opacity: 0; width: 0; height: 0; }
  .toggle-slider { position: relative; width: 36px; height: 20px; background: rgba(0,0,0,0.15); border-radius: 20px; transition: background 0.2s ease; flex-shrink: 0; }
  .toggle-slider::after { content: ''; position: absolute; top: 2px; left: 2px; width: 16px; height: 16px; background: var(--color-bg, #fff); border-radius: 50%; box-shadow: 0 1px 3px rgba(0,0,0,0.2); transition: transform 0.2s ease; }
  .compact-toggle input:checked + .toggle-slider { background: var(--bos-success-color, #34c759); }
  .compact-toggle input:checked + .toggle-slider::after { transform: translateX(16px); }
  :global(.dark) .toggle-slider { background: rgba(255,255,255,0.15); }
  :global(.dark) .toggle-slider::after { background: var(--dbg, #e5e5e5); }
  .toggle-label { font-weight: 500; }
  .compact-sort-select { padding: 6px 10px; padding-right: 28px; background: var(--color-bg-secondary); border: 1px solid var(--color-border); border-radius: 8px; font-size: 12px; font-weight: 500; color: var(--color-text); cursor: pointer; appearance: none; background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 24 24' fill='none' stroke='%239ca3af' stroke-width='2'%3E%3Cpath d='m6 9 6 6 6-6'/%3E%3C/svg%3E"); background-repeat: no-repeat; background-position: right 8px center; }
  .compact-sort-select:hover { border-color: var(--color-border-hover); }
  :global(.dark) .compact-sort-select { background-color: rgba(255,255,255,0.04); }
</style>
