<script lang="ts">
  import type { CustomAgent } from '$lib/api/ai/types';
  import { onMount } from 'svelte';

  interface Props {
    agents: CustomAgent[];
    selectedId?: string | null;
    onSelect: (agent: CustomAgent | null) => void;
    placeholder?: string;
    includeDefault?: boolean;
    onCreateNew?: () => void;
    onManage?: () => void;
  }

  let {
    agents = [],
    selectedId = null,
    onSelect,
    placeholder = 'Select an agent',
    includeDefault = true,
    onCreateNew,
    onManage
  }: Props = $props();

  // State
  let isOpen = $state(false);
  let searchQuery = $state('');
  let dropdownRef = $state<HTMLDivElement | null>(null);
  let buttonRef = $state<HTMLButtonElement | null>(null);
  let searchInputRef = $state<HTMLInputElement | null>(null);
  let selectedIndex = $state(-1);

  // Derived: Selected agent
  let selectedAgent = $derived(
    selectedId ? agents.find(a => a.id === selectedId) : null
  );

  // Derived: Group agents by category
  let groupedAgents = $derived(() => {
    const query = searchQuery.toLowerCase().trim();

    // Filter agents by search
    const filtered = agents.filter(agent => {
      if (!query) return true;
      return (
        agent.display_name.toLowerCase().includes(query) ||
        agent.name.toLowerCase().includes(query) ||
        agent.description?.toLowerCase().includes(query) ||
        agent.category?.toLowerCase().includes(query)
      );
    });

    // Group by category
    const groups: Record<string, CustomAgent[]> = {};

    filtered.forEach(agent => {
      const category = agent.category || 'uncategorized';
      if (!groups[category]) {
        groups[category] = [];
      }
      groups[category].push(agent);
    });

    return groups;
  });

  // Derived: Flat list for keyboard navigation
  let flatAgentList = $derived(() => {
    const list: (CustomAgent | null)[] = [];
    if (includeDefault) {
      list.push(null); // Default agent option
    }

    const groups = groupedAgents();
    Object.keys(groups).sort().forEach(category => {
      list.push(...groups[category]);
    });

    return list;
  });

  // Handle open/close
  function toggleDropdown() {
    isOpen = !isOpen;
    if (isOpen) {
      searchQuery = '';
      selectedIndex = -1;
      setTimeout(() => searchInputRef?.focus(), 50);
    }
  }

  function closeDropdown() {
    isOpen = false;
    searchQuery = '';
    selectedIndex = -1;
  }

  // Handle selection
  function handleSelect(agent: CustomAgent | null) {
    onSelect(agent);
    closeDropdown();
  }

  // Keyboard navigation
  function handleKeyDown(e: KeyboardEvent) {
    if (!isOpen) {
      if (e.key === 'Enter' || e.key === ' ') {
        e.preventDefault();
        toggleDropdown();
      }
      return;
    }

    const list = flatAgentList();

    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault();
        selectedIndex = selectedIndex < list.length - 1 ? selectedIndex + 1 : 0;
        scrollToIndex(selectedIndex);
        break;

      case 'ArrowUp':
        e.preventDefault();
        selectedIndex = selectedIndex > 0 ? selectedIndex - 1 : list.length - 1;
        scrollToIndex(selectedIndex);
        break;

      case 'Enter':
        e.preventDefault();
        if (selectedIndex >= 0 && selectedIndex < list.length) {
          handleSelect(list[selectedIndex]);
        }
        break;

      case 'Escape':
        e.preventDefault();
        closeDropdown();
        buttonRef?.focus();
        break;
    }
  }

  function scrollToIndex(index: number) {
    const items = dropdownRef?.querySelectorAll('[data-agent-item]');
    if (items && items[index]) {
      items[index].scrollIntoView({ block: 'nearest', behavior: 'smooth' });
    }
  }

  // Close on outside click
  function handleClickOutside(e: MouseEvent) {
    if (
      isOpen &&
      dropdownRef &&
      buttonRef &&
      !dropdownRef.contains(e.target as Node) &&
      !buttonRef.contains(e.target as Node)
    ) {
      closeDropdown();
    }
  }

  onMount(() => {
    document.addEventListener('click', handleClickOutside);
    return () => document.removeEventListener('click', handleClickOutside);
  });

  // Get model badge text
  function getModelBadge(agent: CustomAgent | null): string {
    if (!agent || !agent.model_preference) return '';

    // Extract short model name (e.g., "gpt-4" -> "GPT-4", "claude-3-opus" -> "Opus")
    const model = agent.model_preference;
    if (model.includes('gpt-4')) return 'GPT-4';
    if (model.includes('gpt-3.5')) return 'GPT-3.5';
    if (model.includes('claude-3-opus')) return 'Opus';
    if (model.includes('claude-3-sonnet')) return 'Sonnet';
    if (model.includes('claude-3-haiku')) return 'Haiku';
    if (model.includes('gemini')) return 'Gemini';
    return model.split('-')[0].toUpperCase();
  }

  // Format category name
  function formatCategory(category: string): string {
    return category
      .split('_')
      .map(word => word.charAt(0).toUpperCase() + word.slice(1))
      .join(' ');
  }
</script>

<div class="agent-selector">
  <!-- Trigger Button -->
  <button
    bind:this={buttonRef}
    type="button"
    class="btn-pill btn-pill-ghost selector-button"
    onclick={toggleDropdown}
    onkeydown={handleKeyDown}
    aria-haspopup="listbox"
    aria-expanded={isOpen}
  >
    <div class="button-content">
      {#if selectedAgent}
        <!-- Selected Agent Display -->
        <div class="selected-agent">
          {#if selectedAgent.avatar}
            <img src={selectedAgent.avatar} alt={selectedAgent.display_name} class="agent-avatar" />
          {:else}
            <div class="agent-avatar-placeholder">
              {selectedAgent.display_name.charAt(0).toUpperCase()}
            </div>
          {/if}
          <div class="agent-info">
            <span class="agent-name">{selectedAgent.display_name}</span>
            {#if selectedAgent.model_preference}
              <span class="model-badge">{getModelBadge(selectedAgent)}</span>
            {/if}
          </div>
        </div>
      {:else}
        <!-- Placeholder -->
        <span class="placeholder-text">{placeholder}</span>
      {/if}
    </div>

    <!-- Chevron Icon -->
    <svg
      class="chevron-icon"
      class:rotated={isOpen}
      xmlns="http://www.w3.org/2000/svg"
      fill="none"
      viewBox="0 0 24 24"
      stroke-width="2"
      stroke="currentColor"
      width="16"
      height="16"
    >
      <path stroke-linecap="round" stroke-linejoin="round" d="m19.5 8.25-7.5 7.5-7.5-7.5" />
    </svg>
  </button>

  <!-- Dropdown Menu -->
  {#if isOpen}
    <div bind:this={dropdownRef} class="dropdown-menu" role="listbox">
      <!-- Search Input -->
      <div class="search-container">
        <svg
          class="search-icon"
          xmlns="http://www.w3.org/2000/svg"
          fill="none"
          viewBox="0 0 24 24"
          stroke-width="2"
          stroke="currentColor"
          width="16"
          height="16"
        >
          <path stroke-linecap="round" stroke-linejoin="round" d="m21 21-5.197-5.197m0 0A7.5 7.5 0 1 0 5.196 5.196a7.5 7.5 0 0 0 10.607 10.607Z" />
        </svg>
        <input
          bind:this={searchInputRef}
          bind:value={searchQuery}
          type="text"
          id="agent-search"
          name="agentSearch"
          autocomplete="off"
          placeholder="Search agents..."
          class="search-input"
          onkeydown={handleKeyDown}
        />
      </div>

      <!-- Agent List -->
      <div class="agent-list">
        {#if includeDefault}
          <button
            type="button"
            class="btn-pill btn-pill-ghost agent-item"
            class:selected={!selectedId}
            class:highlighted={selectedIndex === 0}
            onclick={() => handleSelect(null)}
            data-agent-item
            role="option"
            aria-selected={!selectedId}
          >
            <div class="agent-avatar-placeholder default">
              <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" width="16" height="16">
                <path stroke-linecap="round" stroke-linejoin="round" d="M9.813 15.904 9 18.75l-.813-2.846a4.5 4.5 0 0 0-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 0 0 3.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 0 0 3.09 3.09L15.75 12l-2.846.813a4.5 4.5 0 0 0-3.09 3.09ZM18.259 8.715 18 9.75l-.259-1.035a3.375 3.375 0 0 0-2.455-2.456L14.25 6l1.036-.259a3.375 3.375 0 0 0 2.455-2.456L18 2.25l.259 1.035a3.375 3.375 0 0 0 2.456 2.456L21.75 6l-1.035.259a3.375 3.375 0 0 0-2.456 2.456ZM16.894 20.567 16.5 21.75l-.394-1.183a2.25 2.25 0 0 0-1.423-1.423L13.5 18.75l1.183-.394a2.25 2.25 0 0 0 1.423-1.423l.394-1.183.394 1.183a2.25 2.25 0 0 0 1.423 1.423l1.183.394-1.183.394a2.25 2.25 0 0 0-1.423 1.423Z" />
              </svg>
            </div>
            <div class="item-content">
              <span class="item-name">Default Agent</span>
              <span class="item-description">Use system default</span>
            </div>
            {#if !selectedId}
              <svg class="checkmark" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" width="16" height="16">
                <path stroke-linecap="round" stroke-linejoin="round" d="m4.5 12.75 6 6 9-13.5" />
              </svg>
            {/if}
          </button>
        {/if}

        <!-- Grouped Agents -->
        {#if Object.keys(groupedAgents()).length > 0}
          {@const groups = groupedAgents()}
          {@const sortedCategories = Object.keys(groups).sort()}

        {#if sortedCategories.length === 0}
          <div class="empty-state">
            {#if searchQuery}
              <p>No agents found matching "{searchQuery}"</p>
            {:else}
              <p>No custom agents available</p>
            {/if}
          </div>
        {:else}
          {#each sortedCategories as category}
            {#if groups[category].length > 0}
              <div class="category-group">
                <div class="category-header">{formatCategory(category)}</div>
                {#each groups[category] as agent, idx}
                  {@const flatIndex = flatAgentList().indexOf(agent)}
                  <button
                    type="button"
                    class="btn-pill btn-pill-ghost agent-item"
                    class:selected={selectedId === agent.id}
                    class:highlighted={selectedIndex === flatIndex}
                    onclick={() => handleSelect(agent)}
                    data-agent-item
                    role="option"
                    aria-selected={selectedId === agent.id}
                  >
                    {#if agent.avatar}
                      <img src={agent.avatar} alt={agent.display_name} class="agent-avatar" />
                    {:else}
                      <div class="agent-avatar-placeholder">
                        {agent.display_name.charAt(0).toUpperCase()}
                      </div>
                    {/if}
                    <div class="item-content">
                      <div class="item-name-row">
                        <span class="item-name">{agent.display_name}</span>
                        {#if agent.model_preference}
                          <span class="model-badge small">{getModelBadge(agent)}</span>
                        {/if}
                      </div>
                      {#if agent.description}
                        <span class="item-description">{agent.description}</span>
                      {/if}
                    </div>
                    {#if selectedId === agent.id}
                      <svg class="checkmark" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" width="16" height="16">
                        <path stroke-linecap="round" stroke-linejoin="round" d="m4.5 12.75 6 6 9-13.5" />
                      </svg>
                    {/if}
                  </button>
                {/each}
              </div>
            {/if}
          {/each}
        {/if}
        {/if}
      </div>

      <!-- Actions Footer -->
      {#if onCreateNew || onManage}
        <div class="actions-footer">
          {#if onCreateNew}
            <button type="button" class="btn-pill btn-pill-ghost action-button" onclick={() => { closeDropdown(); onCreateNew?.(); }}>
              <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" width="16" height="16">
                <path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
              </svg>
              Create New Agent
            </button>
          {/if}
          {#if onManage}
            <button type="button" class="btn-pill btn-pill-ghost action-link" onclick={() => { closeDropdown(); onManage?.(); }}>
              Manage Agents
            </button>
          {/if}
        </div>
      {/if}
    </div>
  {/if}
</div>

<style>
  .agent-selector {
    position: relative;
    width: 100%;
  }

  /* Trigger button */
  .selector-button {
    display: flex;
    align-items: center;
    justify-content: space-between;
    width: 100%;
    padding: 7px 12px;
    background: var(--dbg2);
    border: 1px solid var(--dbd);
    border-radius: 8px;
    font-size: 13px;
    cursor: pointer;
    transition: all 0.15s ease;
  }

  .selector-button:hover {
    border-color: var(--dt3);
    background: var(--dbg3, var(--dbg2));
  }

  .selector-button:focus {
    outline: none;
    border-color: var(--dt2);
  }

  .button-content {
    flex: 1;
    display: flex;
    align-items: center;
    gap: 8px;
    min-width: 0;
  }

  .selected-agent {
    display: flex;
    align-items: center;
    gap: 8px;
    min-width: 0;
  }

  .agent-avatar {
    width: 24px;
    height: 24px;
    border-radius: 6px;
    object-fit: cover;
    flex-shrink: 0;
  }

  .agent-avatar-placeholder {
    width: 24px;
    height: 24px;
    border-radius: 6px;
    background: var(--dt);
    color: var(--dbg);
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 11px;
    font-weight: 700;
    flex-shrink: 0;
  }

  .agent-avatar-placeholder.default {
    background: var(--dt2);
  }

  .agent-info {
    display: flex;
    align-items: center;
    gap: 6px;
    min-width: 0;
  }

  .agent-name {
    font-weight: 500;
    color: var(--dt);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    font-size: 13px;
  }

  .model-badge {
    padding: 1px 5px;
    background: var(--dbg3, var(--dbd));
    color: var(--dt3);
    font-size: 9px;
    font-weight: 600;
    border-radius: 4px;
    text-transform: uppercase;
    flex-shrink: 0;
    border: 1px solid var(--dbd);
  }

  .model-badge.small {
    font-size: 8px;
    padding: 1px 4px;
  }

  .placeholder-text {
    color: var(--dt3);
    font-size: 13px;
  }

  .chevron-icon {
    color: var(--dt3);
    transition: transform 0.2s ease;
    flex-shrink: 0;
  }

  .chevron-icon.rotated {
    transform: rotate(180deg);
  }

  /* Dropdown */
  .dropdown-menu {
    position: absolute;
    top: calc(100% + 4px);
    left: 0;
    right: 0;
    background: var(--dbg);
    border: 1px solid var(--dbd);
    border-radius: 12px;
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.12);
    z-index: 50;
    display: flex;
    flex-direction: column;
    max-height: 400px;
  }

  .search-container {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 10px 12px;
    border-bottom: 1px solid var(--dbd);
  }

  .search-icon {
    color: var(--dt3);
    flex-shrink: 0;
  }

  .search-input {
    flex: 1;
    border: none;
    background: transparent;
    font-size: 13px;
    color: var(--dt);
    outline: none;
  }

  .search-input::placeholder {
    color: var(--dt3);
  }

  .agent-list {
    overflow-y: auto;
    max-height: 300px;
    padding: 4px;
  }

  .category-group {
    margin-bottom: 4px;
  }

  .category-header {
    padding: 6px 12px 3px;
    font-size: 10px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: var(--dt3);
  }

  .agent-item {
    display: flex;
    align-items: center;
    gap: 10px;
    width: 100%;
    padding: 8px 10px;
    background: transparent;
    border: none;
    border-radius: 8px;
    cursor: pointer;
    transition: background 0.1s ease;
    text-align: left;
  }

  .agent-item:hover {
    background: var(--dbg2);
  }

  .agent-item.highlighted {
    background: var(--dbg2);
    outline: 1px solid var(--dbd);
    outline-offset: -1px;
  }

  .agent-item.selected {
    background: var(--dbg2);
  }

  .item-content {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 1px;
  }

  .item-name-row {
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .item-name {
    font-size: 13px;
    font-weight: 500;
    color: var(--dt);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .item-description {
    font-size: 11px;
    color: var(--dt3);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .checkmark {
    color: var(--dt2);
    flex-shrink: 0;
  }

  .empty-state {
    padding: 28px 16px;
    text-align: center;
    color: var(--dt3);
    font-size: 13px;
  }

  .actions-footer {
    display: flex;
    flex-direction: column;
    gap: 4px;
    padding: 6px;
    border-top: 1px solid var(--dbd);
  }

  /* Monochromatic action button */
  .action-button {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 6px;
    width: 100%;
    padding: 7px 12px;
    background: var(--dt);
    color: var(--dbg);
    border: none;
    border-radius: 6px;
    font-size: 12px;
    font-weight: 500;
    cursor: pointer;
    transition: opacity 0.15s ease;
  }

  .action-button:hover {
    opacity: 0.85;
  }

  .action-link {
    padding: 7px 12px;
    background: transparent;
    color: var(--dt3);
    border: none;
    font-size: 12px;
    font-weight: 500;
    cursor: pointer;
    text-align: center;
    transition: color 0.15s ease;
  }

  .action-link:hover {
    color: var(--dt);
    text-decoration: underline;
  }
</style>
