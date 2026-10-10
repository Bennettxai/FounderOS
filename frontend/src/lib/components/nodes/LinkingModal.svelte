<script lang="ts">
	import { fly } from 'svelte/transition';
	import { nodes } from '$lib/stores/nodes';
	import { getProjects } from '$lib/api/projects';
	import { getContexts } from '$lib/api/contexts';
	import { getConversations } from '$lib/api/conversations';
	import type { Project } from '$lib/api/projects/types';
	import type { ContextListItem } from '$lib/api/contexts/types';
	import type { Conversation } from '$lib/api/conversations/types';
	import type { LinkedProject, LinkedContext, LinkedConversation } from '$lib/api/nodes/types';

	interface Props {
		nodeId: string;
		nodeName: string;
		onClose: () => void;
	}

	let { nodeId, nodeName, onClose }: Props = $props();

	// Tab state
	type TabType = 'projects' | 'contexts' | 'conversations';
	let activeTab: TabType = $state('projects');
	let searchQuery = $state('');

	// Data state
	let allProjects: Project[] = $state([]);
	let allContexts: ContextListItem[] = $state([]);
	let allConversations: Conversation[] = $state([]);

	// Loading state
	let loadingAvailable = $state(true);
	let linking = $state(false);

	// Derived: linked items from store
	let linkedProjects = $derived($nodes.currentNodeLinks?.projects ?? []);
	let linkedContexts = $derived($nodes.currentNodeLinks?.contexts ?? []);
	let linkedConversations = $derived($nodes.currentNodeLinks?.conversations ?? []);
	let linksLoading = $derived($nodes.linksLoading);

	// Derived: IDs of already linked items
	let linkedProjectIds = $derived(new Set(linkedProjects.map(p => p.id)));
	let linkedContextIds = $derived(new Set(linkedContexts.map(c => c.id)));
	let linkedConversationIds = $derived(new Set(linkedConversations.map(c => c.id)));

	// Derived: filtered available items (not already linked)
	let availableProjects = $derived(
		allProjects
			.filter(p => !linkedProjectIds.has(p.id))
			.filter(p => searchQuery === '' || p.name.toLowerCase().includes(searchQuery.toLowerCase()))
	);

	let availableContexts = $derived(
		allContexts
			.filter(c => !linkedContextIds.has(c.id))
			.filter(c => searchQuery === '' || c.name.toLowerCase().includes(searchQuery.toLowerCase()))
	);

	let availableConversations = $derived(
		allConversations
			.filter(c => !linkedConversationIds.has(c.id))
			.filter(c => {
				const title = c.title || 'Untitled';
				return searchQuery === '' || title.toLowerCase().includes(searchQuery.toLowerCase());
			})
	);

	// Load available items
	async function loadAvailableItems() {
		loadingAvailable = true;
		try {
			const [projects, contexts, conversationsData] = await Promise.all([
				getProjects(),
				getContexts(),
				getConversations()
			]);
			allProjects = projects;
			allContexts = contexts;
			allConversations = conversationsData.conversations;
		} catch (error) {
			console.error('Failed to load available items:', error);
		} finally {
			loadingAvailable = false;
		}
	}

	// Load linked items on mount
	$effect(() => {
		loadAvailableItems();
		nodes.loadLinks(nodeId);
	});

	// Link handlers
	async function handleLinkProject(projectId: string) {
		linking = true;
		try {
			await nodes.linkProject(nodeId, projectId);
		} catch (error) {
			console.error('Failed to link project:', error);
		} finally {
			linking = false;
		}
	}

	async function handleUnlinkProject(projectId: string) {
		linking = true;
		try {
			await nodes.unlinkProject(nodeId, projectId);
		} catch (error) {
			console.error('Failed to unlink project:', error);
		} finally {
			linking = false;
		}
	}

	async function handleLinkContext(contextId: string) {
		linking = true;
		try {
			await nodes.linkContext(nodeId, contextId);
		} catch (error) {
			console.error('Failed to link context:', error);
		} finally {
			linking = false;
		}
	}

	async function handleUnlinkContext(contextId: string) {
		linking = true;
		try {
			await nodes.unlinkContext(nodeId, contextId);
		} catch (error) {
			console.error('Failed to unlink context:', error);
		} finally {
			linking = false;
		}
	}

	async function handleLinkConversation(conversationId: string) {
		linking = true;
		try {
			await nodes.linkConversation(nodeId, conversationId);
		} catch (error) {
			console.error('Failed to link conversation:', error);
		} finally {
			linking = false;
		}
	}

	async function handleUnlinkConversation(conversationId: string) {
		linking = true;
		try {
			await nodes.unlinkConversation(nodeId, conversationId);
		} catch (error) {
			console.error('Failed to unlink conversation:', error);
		} finally {
			linking = false;
		}
	}

	// Helper for status colors
	function getStatusColor(status: string) {
		switch (status) {
			case 'active': return 'ng-status-tag--active';
			case 'completed': return 'ng-status-tag--completed';
			case 'paused': return 'ng-status-tag--paused';
			case 'archived': return 'ng-status-tag--archived';
			default: return 'ng-status-tag--archived';
		}
	}

	function getContextTypeIcon(type: string) {
		switch (type) {
			case 'person': return 'M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z';
			case 'business': return 'M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4';
			case 'project': return 'M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z';
			case 'document': return 'M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z';
			default: return 'M4 6h16M4 12h16M4 18h16';
		}
	}

	function formatDate(dateString: string) {
		return new Date(dateString).toLocaleDateString('en-US', {
			month: 'short',
			day: 'numeric',
			year: 'numeric'
		});
	}
</script>

<div class="ng-modal-backdrop">
	<!-- Backdrop -->
	<button
		class="ng-modal-backdrop__dismiss"
		onclick={onClose}
	></button>

	<!-- Modal -->
	<div
		class="ng-modal ng-modal--lg"
		transition:fly={{ y: 20, duration: 200 }}
	>
		<!-- Header -->
		<div class="ng-modal__header">
			<div class="ng-modal__header-top">
				<div>
					<h2 class="ng-modal__title">Link Items</h2>
					<p class="ng-modal__subtitle">Connect projects, context profiles, and conversations to <span class="ng-modal__highlight">{nodeName}</span></p>
				</div>
				<button
					onclick={onClose}
					class="btn-pill btn-pill-ghost btn-pill-icon"
				>
					<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
					</svg>
				</button>
			</div>

			<!-- Tabs -->
			<div class="ng-tabs">
				{#each [
					{ id: 'projects', label: 'Projects', count: linkedProjects.length },
					{ id: 'contexts', label: 'Context Profiles', count: linkedContexts.length },
					{ id: 'conversations', label: 'Conversations', count: linkedConversations.length }
				] as tab}
					<button
						onclick={() => { activeTab = tab.id as TabType; searchQuery = ''; }}
						class="ng-tab {activeTab === tab.id ? 'ng-tab--active' : ''}"
					>
						{tab.label}
						{#if tab.count > 0}
							<span class="ng-tab__badge">{tab.count}</span>
						{/if}
					</button>
				{/each}
			</div>
		</div>

		<!-- Search -->
		<div class="ng-modal__search-bar">
			<div class="ng-search-wrap">
				<svg class="ng-search-wrap__icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
				</svg>
				<input
					type="text"
					bind:value={searchQuery}
					placeholder="Search {activeTab}..."
					class="ng-search-input"
				/>
			</div>
		</div>

		<!-- Content -->
		<div class="ng-modal__body ng-modal__body--scroll">
			{#if loadingAvailable || linksLoading}
				<div class="ng-spinner-wrap">
					<div class="ng-spinner"></div>
				</div>
			{:else}
				<!-- Linked Items Section -->
				{#if activeTab === 'projects' && linkedProjects.length > 0}
					<div class="ng-linked-section">
						<h3 class="ng-linked-section__title">Linked Projects ({linkedProjects.length})</h3>
						<div class="ng-linked-list">
							{#each linkedProjects as project}
								<div class="ng-linked-item">
									<div class="ng-linked-item__left">
										<div class="ng-link-icon ng-link-icon--project">
											<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
												<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
											</svg>
										</div>
										<div>
											<p class="ng-linked-item__name">{project.name}</p>
											<p class="ng-linked-item__meta">Linked {formatDate(project.linked_at)}</p>
										</div>
									</div>
									<button
										onclick={() => handleUnlinkProject(project.id)}
										disabled={linking}
										class="btn-pill btn-pill-danger btn-pill-icon disabled:opacity-50"
										title="Unlink"
									>
										<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
										</svg>
									</button>
								</div>
							{/each}
						</div>
					</div>
				{/if}

				{#if activeTab === 'contexts' && linkedContexts.length > 0}
					<div class="ng-linked-section">
						<h3 class="ng-linked-section__title">Linked Context Profiles ({linkedContexts.length})</h3>
						<div class="ng-linked-list">
							{#each linkedContexts as context}
								<div class="ng-linked-item">
									<div class="ng-linked-item__left">
										<div class="ng-link-icon ng-link-icon--context">
											<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
												<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d={getContextTypeIcon(context.type)} />
											</svg>
										</div>
										<div>
											<p class="ng-linked-item__name">{context.name}</p>
											<p class="ng-linked-item__meta">Linked {formatDate(context.linked_at)}</p>
										</div>
									</div>
									<button
										onclick={() => handleUnlinkContext(context.id)}
										disabled={linking}
										class="btn-pill btn-pill-danger btn-pill-icon disabled:opacity-50"
										title="Unlink"
									>
										<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
										</svg>
									</button>
								</div>
							{/each}
						</div>
					</div>
				{/if}

				{#if activeTab === 'conversations' && linkedConversations.length > 0}
					<div class="ng-linked-section">
						<h3 class="ng-linked-section__title">Linked Conversations ({linkedConversations.length})</h3>
						<div class="ng-linked-list">
							{#each linkedConversations as conversation}
								<div class="ng-linked-item">
									<div class="ng-linked-item__left">
										<div class="ng-link-icon ng-link-icon--conversation">
											<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
												<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
											</svg>
										</div>
										<div>
											<p class="ng-linked-item__name">{conversation.title || 'Untitled Conversation'}</p>
											<p class="ng-linked-item__meta">Linked {formatDate(conversation.linked_at)}</p>
										</div>
									</div>
									<button
										onclick={() => handleUnlinkConversation(conversation.id)}
										disabled={linking}
										class="btn-pill btn-pill-danger btn-pill-icon disabled:opacity-50"
										title="Unlink"
									>
										<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
										</svg>
									</button>
								</div>
							{/each}
						</div>
					</div>
				{/if}

				<!-- Available Items Section -->
				{#if activeTab === 'projects'}
					<div>
						<h3 class="ng-linked-section__title">Available Projects ({availableProjects.length})</h3>
						{#if availableProjects.length === 0}
							<p class="ng-empty-text ng-empty-text--center">
								{searchQuery ? 'No matching projects found' : 'All projects are already linked'}
							</p>
						{:else}
							<div class="ng-linked-list">
								{#each availableProjects as project}
									<button
										onclick={() => handleLinkProject(project.id)}
										disabled={linking}
										class="ng-available-item"
									>
										<div class="ng-linked-item__left">
											<div class="ng-link-icon ng-link-icon--project">
												<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
													<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
												</svg>
											</div>
											<div class="ng-available-item__info">
												<p class="ng-linked-item__name">{project.name}</p>
												<div class="ng-available-item__meta-row">
													<span class="ng-status-tag {getStatusColor(project.status)}">{project.status}</span>
													{#if project.client_name}
														<span class="ng-linked-item__meta">{project.client_name}</span>
													{/if}
												</div>
											</div>
										</div>
										<svg class="w-5 h-5 ng-add-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
										</svg>
									</button>
								{/each}
							</div>
						{/if}
					</div>
				{/if}

				{#if activeTab === 'contexts'}
					<div>
						<h3 class="ng-linked-section__title">Available Context Profiles ({availableContexts.length})</h3>
						{#if availableContexts.length === 0}
							<p class="ng-empty-text ng-empty-text--center">
								{searchQuery ? 'No matching context profiles found' : 'All context profiles are already linked'}
							</p>
						{:else}
							<div class="ng-linked-list">
								{#each availableContexts as context}
									<button
										onclick={() => handleLinkContext(context.id)}
										disabled={linking}
										class="ng-available-item"
									>
										<div class="ng-linked-item__left">
											<div class="ng-link-icon ng-link-icon--context">
												<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
													<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d={getContextTypeIcon(context.type)} />
												</svg>
											</div>
											<div class="ng-available-item__info">
												<p class="ng-linked-item__name">{context.name}</p>
												<p class="ng-linked-item__meta">{context.type}</p>
											</div>
										</div>
										<svg class="w-5 h-5 ng-add-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
										</svg>
									</button>
								{/each}
							</div>
						{/if}
					</div>
				{/if}

				{#if activeTab === 'conversations'}
					<div>
						<h3 class="ng-linked-section__title">Available Conversations ({availableConversations.length})</h3>
						{#if availableConversations.length === 0}
							<p class="ng-empty-text ng-empty-text--center">
								{searchQuery ? 'No matching conversations found' : 'All conversations are already linked'}
							</p>
						{:else}
							<div class="ng-linked-list">
								{#each availableConversations as conversation}
									<button
										onclick={() => handleLinkConversation(conversation.id)}
										disabled={linking}
										class="ng-available-item"
									>
										<div class="ng-linked-item__left">
											<div class="ng-link-icon ng-link-icon--conversation">
												<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
													<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
												</svg>
											</div>
											<div class="ng-available-item__info">
												<p class="ng-linked-item__name">{conversation.title || 'Untitled Conversation'}</p>
												<p class="ng-linked-item__meta">{formatDate(conversation.updated_at)}</p>
											</div>
										</div>
										<svg class="w-5 h-5 ng-add-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
										</svg>
									</button>
								{/each}
							</div>
						{/if}
					</div>
				{/if}
			{/if}
		</div>

		<!-- Footer -->
		<div class="ng-modal__footer">
			<p class="ng-modal__footer-text">
				{linkedProjects.length + linkedContexts.length + linkedConversations.length} items linked
			</p>
			<button
				onclick={onClose}
				class="btn-pill btn-pill-primary btn-pill-sm"
			>
				Done
			</button>
		</div>
	</div>
</div>

<style>
	.ng-modal-backdrop {
		position: fixed;
		inset: 0;
		z-index: 50;
		display: flex;
		align-items: center;
		justify-content: center;
	}
	.ng-modal-backdrop__dismiss {
		position: absolute;
		inset: 0;
		background: rgba(0,0,0,0.5);
		cursor: default;
		border: none;
	}
	.ng-modal {
		position: relative;
		background: var(--dbg);
		border-radius: 1rem;
		box-shadow: 0 20px 60px rgba(0,0,0,0.3);
		width: 100%;
		max-width: 42rem;
		margin: 0 1rem;
		max-height: 85vh;
		display: flex;
		flex-direction: column;
		overflow: hidden;
	}
	.ng-modal__header {
		padding: 1.5rem;
		border-bottom: 1px solid var(--dbd);
		flex-shrink: 0;
	}
	.ng-modal__header-top {
		display: flex;
		align-items: center;
		justify-content: space-between;
	}
	.ng-modal__title {
		font-size: 1.25rem;
		font-weight: 600;
		color: var(--dt);
	}
	.ng-modal__subtitle {
		font-size: 0.875rem;
		color: var(--dt3);
		margin-top: 0.25rem;
	}
	.ng-modal__highlight { font-weight: 500; }
	.ng-tabs {
		display: flex;
		gap: 0.25rem;
		margin-top: 1rem;
		border-bottom: 1px solid var(--dbd);
		margin-bottom: -1px;
	}
	.ng-tab {
		padding: 0.5rem 0.75rem;
		font-size: 0.875rem;
		color: var(--dt3);
		border: none;
		background: none;
		cursor: pointer;
		border-bottom: 2px solid transparent;
	}
	.ng-tab--active {
		color: #3b82f6;
		border-bottom-color: #3b82f6;
	}
	.ng-tab__badge {
		margin-left: 0.375rem;
		padding: 0.125rem 0.375rem;
		font-size: 0.75rem;
		border-radius: 9999px;
		background: rgba(59, 130, 246, 0.1);
		color: #2563eb;
	}
	.ng-modal__search-bar {
		padding: 0.75rem 1.5rem;
		border-bottom: 1px solid var(--dbd2);
		flex-shrink: 0;
	}
	.ng-search-wrap { position: relative; }
	.ng-search-wrap__icon {
		position: absolute;
		left: 0.75rem;
		top: 50%;
		transform: translateY(-50%);
		width: 1rem;
		height: 1rem;
		color: var(--dt4);
	}
	.ng-search-input {
		width: 100%;
		padding: 0.5rem 1rem 0.5rem 2.5rem;
		font-size: 0.875rem;
		border: 1px solid var(--dbd);
		border-radius: 0.5rem;
		background: var(--dbg);
		color: var(--dt);
	}
	.ng-search-input:focus {
		outline: none;
		box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.4);
	}
	.ng-modal__body { flex: 1; overflow-y: auto; padding: 1.5rem; }
	.ng-spinner-wrap {
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 3rem 0;
	}
	.ng-spinner {
		width: 2rem;
		height: 2rem;
		border: 2px solid #3b82f6;
		border-top-color: transparent;
		border-radius: 50%;
		animation: spin 0.6s linear infinite;
	}
	@keyframes spin { to { transform: rotate(360deg); } }
	.ng-linked-section { margin-bottom: 1.5rem; }
	.ng-linked-section__title {
		font-size: 0.875rem;
		font-weight: 500;
		color: var(--dt2);
		margin-bottom: 0.75rem;
	}
	.ng-linked-list {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}
	.ng-linked-item {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 0.75rem;
		background: rgba(59, 130, 246, 0.06);
		border-radius: 0.5rem;
		border: 1px solid rgba(59, 130, 246, 0.15);
	}
	.ng-linked-item__left {
		display: flex;
		align-items: center;
		gap: 0.75rem;
	}
	.ng-link-icon {
		width: 2rem;
		height: 2rem;
		border-radius: 0.5rem;
		display: flex;
		align-items: center;
		justify-content: center;
	}
	.ng-link-icon--project { background: rgba(34, 197, 94, 0.1); color: #16a34a; }
	.ng-link-icon--context { background: rgba(168, 85, 247, 0.1); color: #9333ea; }
	.ng-link-icon--conversation { background: rgba(59, 130, 246, 0.1); color: #2563eb; }
	.ng-linked-item__name {
		font-weight: 500;
		color: var(--dt);
	}
	.ng-linked-item__meta {
		font-size: 0.75rem;
		color: var(--dt3);
	}
	.ng-available-item {
		width: 100%;
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 0.75rem;
		background: var(--dbg2);
		border: 1px solid var(--dbd);
		border-radius: 0.5rem;
		cursor: pointer;
		transition: background 0.15s;
	}
	.ng-available-item:hover { background: var(--dbg3); }
	.ng-available-item:disabled { opacity: 0.5; cursor: default; }
	.ng-available-item__info { text-align: left; }
	.ng-available-item__meta-row {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		margin-top: 0.125rem;
	}
	.ng-status-tag {
		padding: 0.125rem 0.375rem;
		font-size: 0.75rem;
		border-radius: 0.25rem;
	}
	.ng-status-tag--active { background: rgba(34, 197, 94, 0.1); color: #16a34a; }
	.ng-status-tag--completed { background: rgba(59, 130, 246, 0.1); color: #2563eb; }
	.ng-status-tag--paused { background: rgba(245, 158, 11, 0.1); color: #d97706; }
	.ng-status-tag--archived { background: var(--dbg2); color: var(--dt2); }
	.ng-add-icon { color: var(--dt4); }
	.ng-empty-text { color: var(--dt3); font-size: 0.875rem; }
	.ng-empty-text--center { text-align: center; padding: 2rem 0; }
	.ng-modal__footer {
		padding: 1rem 1.5rem;
		border-top: 1px solid var(--dbd);
		flex-shrink: 0;
		background: var(--dbg2);
		display: flex;
		align-items: center;
		justify-content: space-between;
	}
	.ng-modal__footer-text {
		font-size: 0.875rem;
		color: var(--dt3);
	}
</style>
