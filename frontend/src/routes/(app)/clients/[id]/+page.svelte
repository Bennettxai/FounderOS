<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { clients, statusColors, statusLabels, dealStageColors, dealStageLabels, interactionTypeLabels, interactionTypeIcons } from '$lib/stores/clients';
	import type {
		ClientDetailResponse,
		ClientStatus,
		ContactResponse,
		InteractionResponse,
		DealResponse,
		CreateContactData,
		CreateInteractionData,
		CreateDealData,
		InteractionType,
		DealStage
	} from '$lib/api';

	// Check if we're in embed mode to propagate to links
	const embedSuffix = $derived($page.url.searchParams.get('embed') === 'true' ? '?embed=true' : '');

	type Tab = 'overview' | 'contacts' | 'interactions' | 'deals';

	// State
	let activeTab = $state<Tab>('overview');
	let loadTimedOut = $state(false);

	// Derive from store auto-subscription
	let client = $derived($clients.currentClient);
	let loading = $derived($clients.loading);
	let error = $derived($clients.error);

	// Modal states
	let showAddContactModal = $state(false);
	let showAddInteractionModal = $state(false);
	let showAddDealModal = $state(false);
	let showEditModal = $state(false);

	// Form states
	let contactForm = $state<CreateContactData>({ name: '' });
	let interactionForm = $state<CreateInteractionData>({ type: 'call', subject: '' });
	let dealForm = $state<CreateDealData>({ name: '' });

	// Load client on mount / when ID changes
	onMount(() => {
		const id = $page.params.id;
		if (id) {
			// Timeout fallback: if the load hasn't resolved in 10 s, stop the
			// spinner so the UI never freezes on loading=true indefinitely.
			const timeoutId = setTimeout(() => {
				if ($clients.loading) {
					clients.clearCurrent();
					loadTimedOut = true;
				}
			}, 10000);

			clients.loadClient(id).catch(() => {
				clearTimeout(timeoutId);
			}).then(() => {
				clearTimeout(timeoutId);
			});
		}
		return () => {
			clients.clearCurrent();
		};
	});

	function formatCurrency(value: number | null): string {
		if (value === null || value === undefined) return '-';
		return new Intl.NumberFormat('en-US', {
			style: 'currency',
			currency: 'USD',
			minimumFractionDigits: 0,
			maximumFractionDigits: 0
		}).format(value);
	}

	function formatDate(dateStr: string | null): string {
		if (!dateStr) return '-';
		return new Date(dateStr).toLocaleDateString('en-US', {
			month: 'short',
			day: 'numeric',
			year: 'numeric'
		});
	}

	function formatDateTime(dateStr: string): string {
		return new Date(dateStr).toLocaleDateString('en-US', {
			month: 'short',
			day: 'numeric',
			year: 'numeric',
			hour: 'numeric',
			minute: '2-digit'
		});
	}

	function getInitials(name: string): string {
		return name
			.split(' ')
			.map((n) => n[0])
			.join('')
			.toUpperCase()
			.slice(0, 2);
	}

	function getStatusStyle(status: ClientStatus): string {
		const map: Record<ClientStatus, string> = {
			lead: 'background: var(--bos-status-info-bg); color: var(--bos-status-info-text)',
			prospect: 'background: var(--bos-status-warning-bg); color: var(--bos-status-warning-text)',
			active: 'background: var(--bos-status-success-bg); color: var(--bos-status-success-text)',
			inactive: 'background: var(--bos-status-neutral-bg); color: var(--bos-status-neutral-text)',
			churned: 'background: var(--bos-status-error-bg); color: var(--bos-status-error-text)',
		};
		return map[status] || '';
	}

	async function handleStatusChange(status: ClientStatus) {
		if (!client) return;
		try {
			await clients.updateClientStatus(client.id, status);
		} catch (err) {
			console.error('Failed to update status:', err);
		}
	}

	async function handleAddContact() {
		if (!client || !contactForm.name.trim()) return;
		try {
			await clients.createContact(client.id, contactForm);
			showAddContactModal = false;
			contactForm = { name: '' };
		} catch (err) {
			console.error('Failed to add contact:', err);
		}
	}

	async function handleDeleteContact(contactId: string) {
		if (!client) return;
		if (!confirm('Are you sure you want to delete this contact?')) return;
		try {
			await clients.deleteContact(client.id, contactId);
		} catch (err) {
			console.error('Failed to delete contact:', err);
		}
	}

	async function handleAddInteraction() {
		if (!client || !interactionForm.subject.trim()) return;
		try {
			await clients.createInteraction(client.id, interactionForm);
			showAddInteractionModal = false;
			interactionForm = { type: 'call', subject: '' };
		} catch (err) {
			console.error('Failed to add interaction:', err);
		}
	}

	async function handleAddDeal() {
		if (!client || !dealForm.name.trim()) return;
		try {
			await clients.createDeal(client.id, dealForm);
			showAddDealModal = false;
			dealForm = { name: '' };
		} catch (err) {
			console.error('Failed to add deal:', err);
		}
	}

	async function handleDeleteClient() {
		if (!client) return;
		if (!confirm('Are you sure you want to delete this client? This will also delete all contacts, interactions, and deals.')) return;
		try {
			await clients.deleteClient(client.id);
			goto('/clients' + embedSuffix);
		} catch (err) {
			console.error('Failed to delete client:', err);
		}
	}

	function getTotalDealsValue(): number {
		if (!client?.deals) return 0;
		return client.deals.reduce((sum, d) => sum + (d.value || 0), 0);
	}

	function getActiveDealsValue(): number {
		if (!client?.deals) return 0;
		return client.deals
			.filter((d) => d.stage !== 'closed_won' && d.stage !== 'closed_lost')
			.reduce((sum, d) => sum + (d.value || 0), 0);
	}
</script>

{#if loadTimedOut}
	<div class="flex-1 flex items-center justify-center" style="background: var(--dbg)">
		<div class="text-center flex flex-col items-center gap-4">
			<svg class="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24" style="color: var(--dt3)">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
			</svg>
			<div>
				<p class="text-sm font-medium" style="color: var(--dt)">Loading is taking longer than expected</p>
				<p class="text-xs mt-1" style="color: var(--dt3)">The server may be slow or unavailable.</p>
			</div>
			<div class="flex items-center gap-4">
				<button
					onclick={() => {
						loadTimedOut = false;
						const id = $page.params.id;
						if (id) clients.loadClient(id).catch(() => {});
					}}
					class="px-4 py-1.5 text-sm font-medium rounded-lg transition-colors hover:opacity-80"
					style="background: var(--dt); color: var(--dbg)"
				>
					Retry
				</button>
				<a href={'/clients' + embedSuffix} class="text-sm underline" style="color: var(--dt2)">
					Back to clients
				</a>
			</div>
		</div>
	</div>
{:else if loading && !client}
	<div class="flex-1 flex items-center justify-center" style="background: var(--dbg)">
		<div class="flex flex-col items-center gap-3" style="color: var(--dt3)">
			<svg class="w-8 h-8 animate-spin" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path
					stroke-linecap="round"
					stroke-linejoin="round"
					stroke-width="2"
					d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
				/>
			</svg>
			<p class="text-sm">Loading client...</p>
		</div>
	</div>
{:else if error}
	<div class="flex-1 flex items-center justify-center" style="background: var(--dbg)">
		<div class="text-center">
			<p style="color: var(--bos-status-error-text)">{error}</p>
			<button onclick={() => goto('/clients' + embedSuffix)} class="mt-4 text-sm underline" style="color: var(--dt2)">
				Back to clients
			</button>
		</div>
	</div>
{:else if client}
	<div class="flex flex-col h-full" style="background: var(--dbg2)">
		<!-- Header -->
		<div style="background: var(--dbg); border-bottom: 1px solid var(--dbd)">
			<div class="px-6 py-4">
				<div class="flex items-start justify-between">
					<div class="flex items-start gap-3">
						<button onclick={() => goto('/clients' + embedSuffix)} class="mt-1 p-1 rounded-lg transition-colors hover:opacity-80" style="color: var(--dt2)">
							<svg class="w-5 h-5" style="color: var(--dt3)" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
							</svg>
						</button>
						<div class="w-14 h-14 rounded-xl flex items-center justify-center text-xl font-semibold" style="background: var(--dbg2); color: var(--dt2); border: 1px solid var(--dbd)">
							{getInitials(client.name)}
						</div>
						<div>
							<h1 class="text-2xl font-semibold" style="color: var(--dt); letter-spacing: -0.02em">{client.name}</h1>
							<div class="flex items-center gap-3 mt-1">
								<span class="inline-flex items-center px-2 py-0.5 text-xs rounded-md" style="background: {client.type === 'company' ? 'var(--bos-status-info-bg)' : 'var(--bos-status-neutral-bg)'}; color: {client.type === 'company' ? 'var(--bos-status-info-text)' : 'var(--bos-status-neutral-text)'}">
									{client.type === 'company' ? 'Company' : 'Individual'}
								</span>
								<select
									value={client.status}
									onchange={(e) => handleStatusChange((e.target as HTMLSelectElement).value as ClientStatus)}
									class="px-2 py-0.5 text-xs rounded-md border-0 cursor-pointer focus:ring-2 focus:ring-gray-900" style={getStatusStyle(client.status)}
								>
									<option value="lead">Lead</option>
									<option value="prospect">Prospect</option>
									<option value="active">Active</option>
									<option value="inactive">Inactive</option>
									<option value="churned">Churned</option>
								</select>
								{#if client.source}
									<span class="text-xs" style="color: var(--dt3)">via {client.source}</span>
								{/if}
							</div>
						</div>
					</div>
					<div class="flex items-center gap-2">
						<button
							onclick={() => (showAddInteractionModal = true)}
							class="px-3 py-1.5 text-sm font-medium rounded-lg transition-colors hover:opacity-80" style="color: var(--dt2)"
						>
							Log Interaction
						</button>
						<button
							onclick={() => (showAddDealModal = true)}
							class="btn-cta px-3 py-1.5 text-sm font-medium rounded-lg transition-colors"
						>
							Add Deal
						</button>
						<button
							onclick={handleDeleteClient}
							class="btn-pill btn-pill-ghost btn-pill-icon"
							title="Delete client"
						>
							<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
							</svg>
						</button>
					</div>
				</div>
			</div>

			<!-- Tabs -->
			<div class="px-6 flex gap-6" style="border-top: 1px solid var(--dbd2)">
				{#each [
					{ id: 'overview', label: 'Overview' },
					{ id: 'contacts', label: `Contacts (${client.contacts?.length ?? 0})` },
					{ id: 'interactions', label: `Interactions (${client.interactions?.length ?? 0})` },
					{ id: 'deals', label: `Deals (${client.deals?.length ?? 0})` }
				] as tab}
					<button
						onclick={() => (activeTab = tab.id as Tab)}
						class="py-3 text-sm font-medium border-b-2 transition-colors {activeTab === tab.id
							? ''
							: 'border-transparent'}" style={activeTab === tab.id ? 'border-color: var(--bos-nav-active); color: var(--bos-nav-active); background: var(--bos-nav-active-bg); border-radius: 6px 6px 0 0' : 'color: var(--dt3)'}
					>
						{tab.label}
					</button>
				{/each}
			</div>
		</div>

		<!-- Content -->
		<div class="flex-1 overflow-auto">
			{#if activeTab === 'overview'}
				<div class="p-6">
					<div class="grid grid-cols-3 gap-6">
						<!-- Main Info -->
						<div class="col-span-2 space-y-6">
							<!-- Contact Info Card -->
							<div class="rounded-xl p-6" style="background: var(--dbg); border: 1px solid var(--dbd)">
								<h3 class="text-sm font-medium mb-4" style="color: var(--dt)">Contact Information</h3>
								<div class="grid grid-cols-2 gap-4">
									<div>
										<div class="text-xs" style="color: var(--dt3)">Email</div>
										<div class="text-sm" style="color: var(--dt)">{client.email || '-'}</div>
									</div>
									<div>
										<div class="text-xs" style="color: var(--dt3)">Phone</div>
										<div class="text-sm" style="color: var(--dt)">{client.phone || '-'}</div>
									</div>
									{#if client.website}
										<div>
											<div class="text-xs" style="color: var(--dt3)">Website</div>
											<a href={client.website} target="_blank" class="text-sm hover:underline" style="color: var(--bos-nav-active)">{client.website}</a>
										</div>
									{/if}
									{#if client.type === 'company'}
										<div>
											<div class="text-xs" style="color: var(--dt3)">Industry</div>
											<div class="text-sm" style="color: var(--dt)">{client.industry || '-'}</div>
										</div>
										<div>
											<div class="text-xs" style="color: var(--dt3)">Company Size</div>
											<div class="text-sm" style="color: var(--dt)">{client.company_size || '-'}</div>
										</div>
									{/if}
								</div>

								{#if client.address || client.city}
									<div class="mt-4 pt-4" style="border-top: 1px solid var(--dbd2)">
										<div class="text-xs" style="color: var(--dt3)">Address</div>
										<div class="text-sm" style="color: var(--dt)">
											{client.address || ''}{client.address && client.city ? ', ' : ''}{client.city || ''}{client.city && client.state ? ', ' : ''}{client.state || ''} {client.zip_code || ''}
											{#if client.country}<br />{client.country}{/if}
										</div>
									</div>
								{/if}
							</div>

							<!-- Notes -->
							{#if client.notes}
								<div class="rounded-xl p-6" style="background: var(--dbg); border: 1px solid var(--dbd)">
									<h3 class="text-sm font-medium mb-2" style="color: var(--dt)">Notes</h3>
									<p class="text-sm whitespace-pre-wrap" style="color: var(--dt2)">{client.notes}</p>
								</div>
							{/if}

							<!-- Recent Interactions -->
							<div class="rounded-xl p-6" style="background: var(--dbg); border: 1px solid var(--dbd)">
								<div class="flex items-center justify-between mb-4">
									<h3 class="text-sm font-medium" style="color: var(--dt)">Recent Activity</h3>
									<button onclick={() => (activeTab = 'interactions')} class="text-xs hover:underline" style="color: var(--dt3)">
										View all
									</button>
								</div>
								{#if !client.interactions || client.interactions.length === 0}
									<p class="text-sm" style="color: var(--dt3)">No interactions yet</p>
								{:else}
									<div class="space-y-3">
										{#each client.interactions.slice(0, 5) as interaction}
											<div class="flex items-start gap-3">
												<span class="text-lg">{interactionTypeIcons[interaction.type]}</span>
												<div class="flex-1 min-w-0">
													<div class="text-sm font-medium" style="color: var(--dt)">{interaction.subject}</div>
													<div class="text-xs" style="color: var(--dt3)">{formatDateTime(interaction.occurred_at)}</div>
												</div>
											</div>
										{/each}
									</div>
								{/if}
							</div>
						</div>

						<!-- Sidebar -->
						<div class="space-y-6">
							<!-- Stats Card -->
							<div class="rounded-xl p-6" style="background: var(--dbg); border: 1px solid var(--dbd)">
								<h3 class="text-sm font-medium mb-4" style="color: var(--dt)">Summary</h3>
								<div class="space-y-4">
									<div>
										<div class="text-xs" style="color: var(--dt3)">Lifetime Value</div>
										<div style="font-size: 1.75rem; font-weight: 700; letter-spacing: -0.03em; color: var(--dt)">{formatCurrency(client.lifetime_value)}</div>
									</div>
									<div>
										<div class="text-xs" style="color: var(--dt3)">Pipeline Value</div>
										<div class="text-lg font-medium" style="color: var(--bos-status-success-text)">{formatCurrency(getActiveDealsValue())}</div>
									</div>
									<div class="pt-3 grid grid-cols-2 gap-4" style="border-top: 1px solid var(--dbd2)">
										<div>
										<div class="text-2xl font-semibold" style="color: var(--dt)">{client.deals?.length ?? 0}</div>
										<div class="text-xs" style="color: var(--dt3)">Total Deals</div>
									</div>
									<div>
										<div class="text-2xl font-semibold" style="color: var(--dt)">{client.contacts?.length ?? 0}</div>
											<div class="text-xs" style="color: var(--dt3)">Contacts</div>
										</div>
									</div>
								</div>
							</div>

							<!-- Tags -->
							{#if client.tags && client.tags.length > 0}
								<div class="rounded-xl p-6" style="background: var(--dbg); border: 1px solid var(--dbd)">
									<h3 class="text-sm font-medium mb-3" style="color: var(--dt)">Tags</h3>
									<div class="flex flex-wrap gap-2">
										{#each client.tags as tag}
											<span class="px-2 py-1 text-xs rounded-md" style="background: var(--dbg2); color: var(--dt2)">{tag}</span>
										{/each}
									</div>
								</div>
							{/if}

							<!-- Dates -->
							<div class="rounded-xl p-6" style="background: var(--dbg); border: 1px solid var(--dbd)">
								<h3 class="text-sm font-medium mb-3" style="color: var(--dt)">Dates</h3>
								<div class="space-y-2 text-sm">
									<div class="flex justify-between">
										<span style="color: var(--dt3)">Created</span>
										<span style="color: var(--dt)">{formatDate(client.created_at)}</span>
									</div>
									<div class="flex justify-between">
										<span style="color: var(--dt3)">Last Updated</span>
										<span style="color: var(--dt)">{formatDate(client.updated_at)}</span>
									</div>
									<div class="flex justify-between">
										<span style="color: var(--dt3)">Last Contact</span>
										<span style="color: var(--dt)">{formatDate(client.last_contacted_at)}</span>
									</div>
								</div>
							</div>
						</div>
					</div>
				</div>
			{:else if activeTab === 'contacts'}
				<div class="p-6">
					<div class="rounded-xl" style="background: var(--dbg); border: 1px solid var(--dbd)">
						<div class="px-6 py-4 flex items-center justify-between" style="border-bottom: 1px solid var(--dbd)">
							<h3 class="text-sm font-medium" style="color: var(--dt)">Contacts</h3>
							<button
								onclick={() => (showAddContactModal = true)}
								class="flex items-center gap-1 px-3 py-1.5 text-sm font-medium rounded-lg transition-colors hover:opacity-80" style="color: var(--dt2)"
							>
								<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
								</svg>
								Add Contact
							</button>
						</div>
						{#if !client.contacts || client.contacts.length === 0}
							<div class="px-6 py-12 text-center" style="color: var(--dt3)">
								<p>No contacts yet</p>
							</div>
						{:else}
							<div class="divide-y" style="--tw-divide-color: var(--dbd2)">
								{#each client.contacts as contact}
									<div class="px-6 py-4 flex items-center justify-between hover:opacity-90">
										<div class="flex items-center gap-4">
											<div class="w-10 h-10 rounded-full flex items-center justify-center text-sm font-medium" style="background: var(--dbg2); color: var(--dt2)">
												{getInitials(contact.name)}
											</div>
											<div>
												<div class="flex items-center gap-2">
													<span class="text-sm font-medium" style="color: var(--dt)">{contact.name}</span>
													{#if contact.is_primary}
														<span class="px-1.5 py-0.5 text-xs rounded" style="background: var(--bos-status-info-bg); color: var(--bos-status-info-text)">Primary</span>
													{/if}
												</div>
												<div class="text-xs" style="color: var(--dt3)">{contact.role || 'No role specified'}</div>
											</div>
										</div>
										<div class="flex items-center gap-4">
											<div class="text-right text-sm">
												<div style="color: var(--dt)">{contact.email || '-'}</div>
												<div style="color: var(--dt3)">{contact.phone || '-'}</div>
											</div>
											<button
												onclick={() => handleDeleteContact(contact.id)}
												class="p-1 rounded transition-colors" style="color: var(--dt4)"
											>
												<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
													<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
												</svg>
											</button>
										</div>
									</div>
								{/each}
							</div>
						{/if}
					</div>
				</div>
			{:else if activeTab === 'interactions'}
				<div class="p-6">
					<div class="rounded-xl" style="background: var(--dbg); border: 1px solid var(--dbd)">
						<div class="px-6 py-4 flex items-center justify-between" style="border-bottom: 1px solid var(--dbd)">
							<h3 class="text-sm font-medium" style="color: var(--dt)">Interaction Timeline</h3>
							<button
								onclick={() => (showAddInteractionModal = true)}
								class="flex items-center gap-1 px-3 py-1.5 text-sm font-medium rounded-lg transition-colors hover:opacity-80" style="color: var(--dt2)"
							>
								<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
								</svg>
								Log Interaction
							</button>
						</div>
						{#if !client.interactions || client.interactions.length === 0}
							<div class="px-6 py-12 text-center" style="color: var(--dt3)">
								<p>No interactions yet</p>
							</div>
						{:else}
							<div class="divide-y" style="--tw-divide-color: var(--dbd2)">
								{#each client.interactions as interaction}
									<div class="px-6 py-4">
										<div class="flex items-start gap-4">
											<div class="w-10 h-10 rounded-lg flex items-center justify-center text-lg" style="background: var(--dbg2)">
												{interactionTypeIcons[interaction.type]}
											</div>
											<div class="flex-1">
												<div class="flex items-center gap-2">
													<span class="text-sm font-medium" style="color: var(--dt)">{interaction.subject}</span>
													<span class="px-1.5 py-0.5 text-xs rounded" style="background: var(--dbg2); color: var(--dt2)">{interactionTypeLabels[interaction.type]}</span>
												</div>
												{#if interaction.description}
													<p class="text-sm mt-1" style="color: var(--dt2)">{interaction.description}</p>
												{/if}
												{#if interaction.outcome}
													<div class="mt-2 text-xs" style="color: var(--dt3)">
														<span class="font-medium">Outcome:</span> {interaction.outcome}
													</div>
												{/if}
												<div class="text-xs mt-2" style="color: var(--dt4)">{formatDateTime(interaction.occurred_at)}</div>
											</div>
										</div>
									</div>
								{/each}
							</div>
						{/if}
					</div>
				</div>
			{:else if activeTab === 'deals'}
				<div class="p-6">
					<div class="rounded-xl" style="background: var(--dbg); border: 1px solid var(--dbd)">
						<div class="px-6 py-4 flex items-center justify-between" style="border-bottom: 1px solid var(--dbd)">
							<h3 class="text-sm font-medium" style="color: var(--dt)">Deals</h3>
							<button
								onclick={() => (showAddDealModal = true)}
								class="flex items-center gap-1 px-3 py-1.5 text-sm font-medium rounded-lg transition-colors hover:opacity-80" style="color: var(--dt2)"
							>
								<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
								</svg>
								Add Deal
							</button>
						</div>
						{#if !client.deals || client.deals.length === 0}
							<div class="px-6 py-12 text-center" style="color: var(--dt3)">
								<p>No deals yet</p>
							</div>
						{:else}
							<div class="divide-y" style="--tw-divide-color: var(--dbd2)">
								{#each client.deals as deal}
									<div class="px-6 py-4">
										<div class="flex items-center justify-between">
											<div>
												<div class="flex items-center gap-2">
													<span class="text-sm font-medium" style="color: var(--dt)">{deal.name}</span>
													<span class="px-1.5 py-0.5 text-xs rounded {dealStageColors[deal.stage]}">{dealStageLabels[deal.stage]}</span>
												</div>
												<div class="text-xs mt-1" style="color: var(--dt3)">
													{deal.probability}% probability
													{#if deal.expected_close_date}
														 &middot; Expected close: {formatDate(deal.expected_close_date)}
													{/if}
												</div>
											</div>
											<div class="text-right">
												<div class="text-lg font-semibold" style="color: var(--dt)">{formatCurrency(deal.value)}</div>
												<div class="text-xs" style="color: var(--dt3)">Created {formatDate(deal.created_at)}</div>
											</div>
										</div>
										{#if deal.notes}
											<p class="text-sm mt-2" style="color: var(--dt2)">{deal.notes}</p>
										{/if}
									</div>
								{/each}
							</div>
						{/if}
					</div>
				</div>
			{/if}
		</div>
	</div>

	<!-- Add Contact Modal -->
	{#if showAddContactModal}
		<div class="fixed inset-0 z-50 flex items-center justify-center">
			<div class="absolute inset-0 bg-black/50" onclick={() => (showAddContactModal = false)} role="presentation"></div>
			<div class="relative rounded-xl shadow-xl w-full max-w-md mx-4 p-6" style="background: var(--dbg); border: 1px solid var(--dbd)">
				<h3 class="text-lg font-semibold mb-4" style="color: var(--dt)">Add Contact</h3>
				<form onsubmit={(e) => { e.preventDefault(); handleAddContact(); }}>
					<div class="space-y-4">
						<div>
							<label class="block text-sm font-medium mb-1" style="color: var(--dt2)">Name *</label>
							<input type="text" bind:value={contactForm.name} required class="w-full px-3 py-2 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500" style="border: 1px solid var(--dbd); background: var(--dbg2); color: var(--dt)" />
						</div>
						<div class="grid grid-cols-2 gap-4">
							<div>
								<label class="block text-sm font-medium mb-1" style="color: var(--dt2)">Email</label>
								<input type="email" bind:value={contactForm.email} class="w-full px-3 py-2 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500" style="border: 1px solid var(--dbd); background: var(--dbg2); color: var(--dt)" />
							</div>
							<div>
								<label class="block text-sm font-medium mb-1" style="color: var(--dt2)">Phone</label>
								<input type="tel" bind:value={contactForm.phone} class="w-full px-3 py-2 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500" style="border: 1px solid var(--dbd); background: var(--dbg2); color: var(--dt)" />
							</div>
						</div>
						<div>
							<label class="block text-sm font-medium mb-1" style="color: var(--dt2)">Role</label>
							<input type="text" bind:value={contactForm.role} placeholder="CEO, CTO, etc." class="w-full px-3 py-2 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500" style="border: 1px solid var(--dbd); background: var(--dbg2); color: var(--dt)" />
						</div>
						<div class="flex items-center gap-2">
							<input type="checkbox" id="isPrimary" bind:checked={contactForm.is_primary} class="rounded" />
							<label for="isPrimary" class="text-sm" style="color: var(--dt2)">Primary contact</label>
						</div>
					</div>
					<div class="flex justify-end gap-3 mt-6">
						<button type="button" onclick={() => (showAddContactModal = false)} class="px-4 py-2 text-sm font-medium rounded-lg hover:opacity-80 transition-colors" style="color: var(--dt2)">Cancel</button>
						<button type="submit" class="btn-pill btn-pill-primary btn-pill-sm">Add Contact</button>
					</div>
				</form>
			</div>
		</div>
	{/if}

	<!-- Add Interaction Modal -->
	{#if showAddInteractionModal}
		<div class="fixed inset-0 z-50 flex items-center justify-center">
			<div class="absolute inset-0 bg-black/50" onclick={() => (showAddInteractionModal = false)} role="presentation"></div>
			<div class="relative rounded-xl shadow-xl w-full max-w-md mx-4 p-6" style="background: var(--dbg); border: 1px solid var(--dbd)">
				<h3 class="text-lg font-semibold mb-4" style="color: var(--dt)">Log Interaction</h3>
				<form onsubmit={(e) => { e.preventDefault(); handleAddInteraction(); }}>
					<div class="space-y-4">
						<div>
							<label class="block text-sm font-medium mb-1" style="color: var(--dt2)">Type</label>
							<select bind:value={interactionForm.type} class="w-full px-3 py-2 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500" style="border: 1px solid var(--dbd); background: var(--dbg2); color: var(--dt)">
								<option value="call">Call</option>
								<option value="email">Email</option>
								<option value="meeting">Meeting</option>
								<option value="note">Note</option>
							</select>
						</div>
						<div>
							<label class="block text-sm font-medium mb-1" style="color: var(--dt2)">Subject *</label>
							<input type="text" bind:value={interactionForm.subject} required class="w-full px-3 py-2 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500" style="border: 1px solid var(--dbd); background: var(--dbg2); color: var(--dt)" />
						</div>
						<div>
							<label class="block text-sm font-medium mb-1" style="color: var(--dt2)">Description</label>
							<textarea bind:value={interactionForm.description} rows="3" class="w-full px-3 py-2 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500" style="border: 1px solid var(--dbd); background: var(--dbg2); color: var(--dt)"></textarea>
						</div>
						<div>
							<label class="block text-sm font-medium mb-1" style="color: var(--dt2)">Outcome</label>
							<input type="text" bind:value={interactionForm.outcome} placeholder="e.g., Scheduled follow-up call" class="w-full px-3 py-2 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500" style="border: 1px solid var(--dbd); background: var(--dbg2); color: var(--dt)" />
						</div>
					</div>
					<div class="flex justify-end gap-3 mt-6">
						<button type="button" onclick={() => (showAddInteractionModal = false)} class="px-4 py-2 text-sm font-medium rounded-lg hover:opacity-80 transition-colors" style="color: var(--dt2)">Cancel</button>
						<button type="submit" class="btn-pill btn-pill-primary btn-pill-sm">Log Interaction</button>
					</div>
				</form>
			</div>
		</div>
	{/if}

	<!-- Add Deal Modal -->
	{#if showAddDealModal}
		<div class="fixed inset-0 z-50 flex items-center justify-center">
			<div class="absolute inset-0 bg-black/50" onclick={() => (showAddDealModal = false)} role="presentation"></div>
			<div class="relative rounded-xl shadow-xl w-full max-w-md mx-4 p-6" style="background: var(--dbg); border: 1px solid var(--dbd)">
				<h3 class="text-lg font-semibold mb-4" style="color: var(--dt)">Add Deal</h3>
				<form onsubmit={(e) => { e.preventDefault(); handleAddDeal(); }}>
					<div class="space-y-4">
						<div>
							<label class="block text-sm font-medium mb-1" style="color: var(--dt2)">Deal Name *</label>
							<input type="text" bind:value={dealForm.name} required class="w-full px-3 py-2 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500" style="border: 1px solid var(--dbd); background: var(--dbg2); color: var(--dt)" />
						</div>
						<div class="grid grid-cols-2 gap-4">
							<div>
								<label class="block text-sm font-medium mb-1" style="color: var(--dt2)">Value</label>
								<input type="number" bind:value={dealForm.value} min="0" step="0.01" class="w-full px-3 py-2 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500" style="border: 1px solid var(--dbd); background: var(--dbg2); color: var(--dt)" />
							</div>
							<div>
								<label class="block text-sm font-medium mb-1" style="color: var(--dt2)">Probability %</label>
								<input type="number" bind:value={dealForm.probability} min="0" max="100" class="w-full px-3 py-2 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500" style="border: 1px solid var(--dbd); background: var(--dbg2); color: var(--dt)" />
							</div>
						</div>
						<div class="grid grid-cols-2 gap-4">
							<div>
								<label class="block text-sm font-medium mb-1" style="color: var(--dt2)">Stage</label>
								<select bind:value={dealForm.stage} class="w-full px-3 py-2 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500" style="border: 1px solid var(--dbd); background: var(--dbg2); color: var(--dt)">
									<option value="qualification">Qualification</option>
									<option value="proposal">Proposal</option>
									<option value="negotiation">Negotiation</option>
									<option value="closed_won">Closed Won</option>
									<option value="closed_lost">Closed Lost</option>
								</select>
							</div>
							<div>
								<label class="block text-sm font-medium mb-1" style="color: var(--dt2)">Expected Close</label>
								<input type="date" bind:value={dealForm.expected_close_date} class="w-full px-3 py-2 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500" style="border: 1px solid var(--dbd); background: var(--dbg2); color: var(--dt)" />
							</div>
						</div>
						<div>
							<label class="block text-sm font-medium mb-1" style="color: var(--dt2)">Notes</label>
							<textarea bind:value={dealForm.notes} rows="2" class="w-full px-3 py-2 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500" style="border: 1px solid var(--dbd); background: var(--dbg2); color: var(--dt)"></textarea>
						</div>
					</div>
					<div class="flex justify-end gap-3 mt-6">
						<button type="button" onclick={() => (showAddDealModal = false)} class="px-4 py-2 text-sm font-medium rounded-lg hover:opacity-80 transition-colors" style="color: var(--dt2)">Cancel</button>
						<button type="submit" class="btn-pill btn-pill-primary btn-pill-sm">Add Deal</button>
					</div>
				</form>
			</div>
		</div>
	{/if}
{/if}
