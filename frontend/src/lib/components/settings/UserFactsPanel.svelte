<script lang="ts">
	import { api } from '$lib/api';
	import type { UserFact } from '$lib/api/memory/types';

	let facts = $state<UserFact[]>([]);
	let loading = $state(false);
	let error = $state<string | null>(null);
	let filterType = $state<'all' | 'preference' | 'fact' | 'style'>('all');
	let filterStatus = $state<'all' | 'pending' | 'confirmed' | 'rejected'>('all');
	let editingFact = $state<UserFact | null>(null);
	let editValue = $state('');

	const factTypes = [
		{ value: 'all', label: 'All Types' },
		{ value: 'preference', label: 'Preferences' },
		{ value: 'fact', label: 'Facts' },
		{ value: 'style', label: 'Styles' }
	] as const;

	const statusOptions = [
		{ value: 'all', label: 'All Status' },
		{ value: 'pending', label: 'Pending' },
		{ value: 'confirmed', label: 'Confirmed' },
		{ value: 'rejected', label: 'Rejected' }
	] as const;

	async function loadFacts() {
		loading = true;
		error = null;
		try {
			const allFacts = await api.getUserFacts({
				activeOnly: filterStatus !== 'rejected',
				type: filterType !== 'all' ? filterType : undefined
			});
			facts = allFacts;
		} catch (err) {
			console.error('Failed to load user facts:', err);
			error = err instanceof Error ? err.message : 'Failed to load user facts';
			facts = [];
		} finally {
			loading = false;
		}
	}

	async function confirmFact(fact: UserFact) {
		try {
			const key = fact.key ?? fact.fact_key;
			await api.confirmUserFact(key);
			await loadFacts();
		} catch (err) {
			console.error('Failed to confirm fact:', err);
			error = err instanceof Error ? err.message : 'Failed to confirm fact';
		}
	}

	async function rejectFact(fact: UserFact) {
		try {
			const key = fact.key ?? fact.fact_key;
			await api.rejectUserFact(key);
			await loadFacts();
		} catch (err) {
			console.error('Failed to reject fact:', err);
			error = err instanceof Error ? err.message : 'Failed to reject fact';
		}
	}

	async function deleteFact(fact: UserFact) {
		const key = fact.key ?? fact.fact_key;
		if (!confirm(`Delete fact "${key}"?`)) return;

		try {
			await api.deleteUserFact(key);
			await loadFacts();
		} catch (err) {
			console.error('Failed to delete fact:', err);
			error = err instanceof Error ? err.message : 'Failed to delete fact';
		}
	}

	function startEdit(fact: UserFact) {
		editingFact = fact;
		editValue = fact.value ?? fact.fact_value;
	}

	function cancelEdit() {
		editingFact = null;
		editValue = '';
	}

	async function saveEdit() {
		if (!editingFact) return;

		try {
			const key = editingFact.key ?? editingFact.fact_key;
			await api.updateUserFact(key, { fact_value: editValue });
			await loadFacts();
			cancelEdit();
		} catch (err) {
			console.error('Failed to update fact:', err);
			error = err instanceof Error ? err.message : 'Failed to update fact';
		}
	}

	function getStatusBadgeClass(status: string): string {
		switch (status) {
			case 'confirmed': return 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400';
			case 'rejected': return 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400';
			default: return 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400';
		}
	}

	function getTypeBadgeClass(type: string): string {
		switch (type) {
			case 'preference': return 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400';
			case 'style': return 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400';
			default: return 'bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-400';
		}
	}

	const filteredFacts = $derived.by(() => {
		let filtered = [...facts];

		if (filterStatus !== 'all') {
			filtered = filtered.filter(f => {
				if (filterStatus === 'pending') return !f.is_confirmed && !f.is_rejected;
				if (filterStatus === 'confirmed') return f.is_confirmed;
				if (filterStatus === 'rejected') return f.is_rejected;
				return true;
			});
		}

		return filtered;
	});

	$effect(() => {
		loadFacts();
	});
</script>

<div class="user-facts-panel">
	<div class="panel-header">
		<div>
			<h3 class="panel-title">User Facts</h3>
			<p class="panel-subtitle">Manage learned preferences, facts, and styles</p>
		</div>
		<button onclick={loadFacts} class="btn-pill btn-pill-ghost refresh-btn" disabled={loading}>
			<svg class="w-4 h-4" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0 3.181 3.183a8.25 8.25 0 0 0 13.803-3.7M4.031 9.865a8.25 8.25 0 0 1 13.803-3.7l3.181 3.182m0-4.991v4.99" />
			</svg>
			Refresh
		</button>
	</div>

	<div class="filters-section">
		<div class="filter-group">
			<label class="filter-label">Type</label>
			<select bind:value={filterType} class="filter-select" onchange={loadFacts}>
				{#each factTypes as type}
					<option value={type.value}>{type.label}</option>
				{/each}
			</select>
		</div>

		<div class="filter-group">
			<label class="filter-label">Status</label>
			<select bind:value={filterStatus} class="filter-select" onchange={loadFacts}>
				{#each statusOptions as status}
					<option value={status.value}>{status.label}</option>
				{/each}
			</select>
		</div>
	</div>

	{#if error}
		<div class="error-message">
			<svg class="w-5 h-5" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m9-.75a9 9 0 1 1-18 0 9 9 0 0 1 18 0Zm-9 3.75h.008v.008H12v-.008Z" />
			</svg>
			{error}
		</div>
	{/if}

	<div class="facts-list">
		{#if loading}
			<div class="loading-state">
				<div class="spinner"></div>
				<p>Loading facts...</p>
			</div>
		{:else if filteredFacts.length === 0}
			<div class="empty-state">
				<svg class="empty-icon" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" d="M9.568 3H5.25A2.25 2.25 0 0 0 3 5.25v4.318c0 .597.237 1.17.659 1.591l9.581 9.581c.699.699 1.78.872 2.607.33a18.095 18.095 0 0 0 5.223-5.223c.542-.827.369-1.908-.33-2.607L11.16 3.66A2.25 2.25 0 0 0 9.568 3Z" />
				</svg>
				<p class="empty-text">No facts found</p>
				<p class="empty-hint">Facts will be learned from your conversations</p>
			</div>
		{:else}
			{#each filteredFacts as fact (fact.key ?? fact.fact_key ?? fact.id)}
				<div class="fact-card">
					<div class="fact-header">
						<div class="fact-key">{fact.key ?? fact.fact_key}</div>
						<div class="badges">
							<span class="badge {getTypeBadgeClass(fact.type ?? fact.fact_type)}">{fact.type ?? fact.fact_type}</span>
							<span class="badge {getStatusBadgeClass(fact.is_confirmed ? 'confirmed' : fact.is_rejected ? 'rejected' : 'pending')}">
								{fact.is_confirmed ? 'Confirmed' : fact.is_rejected ? 'Rejected' : 'Pending'}
							</span>
						</div>
					</div>

					{#if (editingFact?.key ?? editingFact?.fact_key) === (fact.key ?? fact.fact_key)}
						<div class="fact-edit">
							<input
								type="text"
								bind:value={editValue}
								class="edit-input"
								placeholder="Fact value"
							/>
							<div class="edit-actions">
								<button onclick={saveEdit} class="btn-pill btn-pill-ghost save-btn">Save</button>
								<button onclick={cancelEdit} class="btn-pill btn-pill-ghost cancel-btn">Cancel</button>
							</div>
						</div>
					{:else}
						<div class="fact-value">{fact.value}</div>
					{/if}

					{#if fact.description}
						<div class="fact-description">{fact.description}</div>
					{/if}

					{#if fact.source}
						<div class="fact-meta">
							<span class="meta-label">Source:</span>
							<span class="meta-value">{fact.source}</span>
						</div>
					{/if}

					{#if fact.confidence_score !== undefined && fact.confidence_score !== null}
						<div class="fact-meta">
							<span class="meta-label">Confidence:</span>
							<span class="meta-value">{Math.round(fact.confidence_score * 100)}%</span>
						</div>
					{/if}

					<div class="fact-actions">
						{#if !fact.is_confirmed && !fact.is_rejected}
							<button onclick={() => confirmFact(fact)} class="action-btn confirm-btn">
								<svg class="w-4 h-4" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" d="m4.5 12.75 6 6 9-13.5" />
								</svg>
								Confirm
							</button>
							<button onclick={() => rejectFact(fact)} class="action-btn reject-btn">
								<svg class="w-4 h-4" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" d="M6 18 18 6M6 6l12 12" />
								</svg>
								Reject
							</button>
						{/if}
						<button onclick={() => startEdit(fact)} class="action-btn edit-btn">
							<svg class="w-4 h-4" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" d="m16.862 4.487 1.687-1.688a1.875 1.875 0 1 1 2.652 2.652L10.582 16.07a4.5 4.5 0 0 1-1.897 1.13L6 18l.8-2.685a4.5 4.5 0 0 1 1.13-1.897l8.932-8.931Zm0 0L19.5 7.125M18 14v4.75A2.25 2.25 0 0 1 15.75 21H5.25A2.25 2.25 0 0 1 3 18.75V8.25A2.25 2.25 0 0 1 5.25 6H10" />
							</svg>
							Edit
						</button>
						<button onclick={() => deleteFact(fact)} class="action-btn delete-btn">
							<svg class="w-4 h-4" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" d="m14.74 9-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 0 1-2.244 2.077H8.084a2.25 2.25 0 0 1-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 0 0-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 0 1 3.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 0 0-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 0 0-7.5 0" />
							</svg>
							Delete
						</button>
					</div>
				</div>
			{/each}
		{/if}
	</div>

	<div class="panel-footer">
		<span class="fact-count">{filteredFacts.length} fact{filteredFacts.length !== 1 ? 's' : ''}</span>
	</div>
</div>

<style>
	.user-facts-panel {
		display: flex;
		flex-direction: column;
		height: 100%;
		background: var(--color-bg);
	}

	.panel-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 20px 24px;
		border-bottom: 1px solid var(--color-border);
	}

	.panel-title {
		font-size: 18px;
		font-weight: 600;
		color: var(--color-text);
		margin: 0 0 4px 0;
	}

	.panel-subtitle {
		font-size: 13px;
		color: var(--color-text-muted);
		margin: 0;
	}

	.refresh-btn {
		display: flex;
		align-items: center;
		gap: 6px;
		padding: 8px 14px;
		font-size: 13px;
		font-weight: 500;
		color: var(--color-text);
		background: var(--color-bg-secondary);
		border: 1px solid var(--color-border);
		border-radius: 6px;
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.refresh-btn:hover:not(:disabled) {
		background: var(--color-bg-tertiary);
	}

	.refresh-btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.filters-section {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 12px;
		padding: 16px 24px;
		border-bottom: 1px solid var(--color-border);
	}

	.filter-group {
		display: flex;
		flex-direction: column;
		gap: 6px;
	}

	.filter-label {
		font-size: 12px;
		font-weight: 500;
		color: var(--color-text-muted);
		text-transform: uppercase;
		letter-spacing: 0.5px;
	}

	.filter-select {
		padding: 8px 12px;
		font-size: 13px;
		color: var(--color-text);
		background: var(--color-bg-secondary);
		border: 1px solid var(--color-border);
		border-radius: 6px;
		cursor: pointer;
		outline: none;
		transition: all 0.15s ease;
	}

	.filter-select:focus {
		border-color: #3b82f6;
	}

	.error-message {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 12px 24px;
		background: #fee;
		color: #c00;
		font-size: 13px;
		border-bottom: 1px solid #fcc;
	}

	.facts-list {
		flex: 1;
		overflow-y: auto;
		padding: 16px 24px;
	}

	.fact-card {
		padding: 16px;
		background: var(--color-bg-secondary);
		border: 1px solid var(--color-border);
		border-radius: 8px;
		margin-bottom: 12px;
		transition: all 0.15s ease;
	}

	.fact-card:hover {
		border-color: #3b82f6;
		box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
	}

	.fact-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-bottom: 8px;
	}

	.fact-key {
		font-size: 14px;
		font-weight: 600;
		color: var(--color-text);
	}

	.badges {
		display: flex;
		gap: 6px;
	}

	.badge {
		padding: 2px 8px;
		font-size: 11px;
		font-weight: 500;
		border-radius: 4px;
		text-transform: uppercase;
		letter-spacing: 0.3px;
	}

	.fact-value {
		font-size: 14px;
		color: var(--color-text);
		margin-bottom: 8px;
		line-height: 1.5;
	}

	.fact-description {
		font-size: 13px;
		color: var(--color-text-muted);
		margin-bottom: 8px;
		line-height: 1.5;
	}

	.fact-meta {
		display: flex;
		align-items: center;
		gap: 6px;
		font-size: 12px;
		color: var(--color-text-muted);
		margin-bottom: 4px;
	}

	.meta-label {
		font-weight: 500;
	}

	.meta-value {
		color: var(--color-text);
	}

	.fact-edit {
		margin-bottom: 12px;
	}

	.edit-input {
		width: 100%;
		padding: 8px 12px;
		font-size: 14px;
		color: var(--color-text);
		background: var(--color-bg);
		border: 1px solid var(--color-border);
		border-radius: 6px;
		margin-bottom: 8px;
		outline: none;
	}

	.edit-input:focus {
		border-color: #3b82f6;
	}

	.edit-actions {
		display: flex;
		gap: 8px;
	}

	.save-btn,
	.cancel-btn {
		padding: 6px 14px;
		font-size: 13px;
		font-weight: 500;
		border: none;
		border-radius: 6px;
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.save-btn {
		color: white;
		background: #3b82f6;
	}

	.save-btn:hover {
		background: #2563eb;
	}

	.cancel-btn {
		color: var(--color-text);
		background: var(--color-bg-tertiary);
	}

	.cancel-btn:hover {
		background: var(--color-border);
	}

	.fact-actions {
		display: flex;
		gap: 8px;
		margin-top: 12px;
		padding-top: 12px;
		border-top: 1px solid var(--color-border);
	}

	.action-btn {
		display: flex;
		align-items: center;
		gap: 4px;
		padding: 6px 12px;
		font-size: 12px;
		font-weight: 500;
		border: 1px solid var(--color-border);
		border-radius: 6px;
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.confirm-btn {
		color: #22c55e;
		background: #22c55e15;
		border-color: #22c55e;
	}

	.confirm-btn:hover {
		background: #22c55e25;
	}

	.reject-btn {
		color: #ef4444;
		background: #ef444415;
		border-color: #ef4444;
	}

	.reject-btn:hover {
		background: #ef444425;
	}

	.edit-btn {
		color: #3b82f6;
		background: #3b82f615;
		border-color: #3b82f6;
	}

	.edit-btn:hover {
		background: #3b82f625;
	}

	.delete-btn {
		color: #ef4444;
		background: transparent;
		border-color: var(--color-border);
	}

	.delete-btn:hover {
		background: #ef444415;
		border-color: #ef4444;
	}

	.loading-state,
	.empty-state {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: 60px 24px;
		text-align: center;
	}

	.spinner {
		width: 32px;
		height: 32px;
		border: 3px solid var(--color-border);
		border-top-color: #3b82f6;
		border-radius: 50%;
		animation: spin 0.8s linear infinite;
		margin-bottom: 16px;
	}

	@keyframes spin {
		to { transform: rotate(360deg); }
	}

	.empty-icon {
		width: 48px;
		height: 48px;
		color: var(--color-text-muted);
		margin-bottom: 16px;
	}

	.empty-text {
		font-size: 14px;
		font-weight: 500;
		color: var(--color-text);
		margin: 0 0 4px 0;
	}

	.empty-hint {
		font-size: 13px;
		color: var(--color-text-muted);
		margin: 0;
	}

	.panel-footer {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 12px 24px;
		border-top: 1px solid var(--color-border);
	}

	.fact-count {
		font-size: 12px;
		color: var(--color-text-muted);
	}
</style>
