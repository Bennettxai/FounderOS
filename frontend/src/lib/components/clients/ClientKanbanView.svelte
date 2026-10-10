<script lang="ts">
	import type { ClientListResponse, ClientStatus } from '$lib/api';
	import { statusColors, statusLabels } from '$lib/stores/clients';

	interface Props {
		clients: ClientListResponse[];
		onClientClick: (id: string) => void;
		onStatusChange: (id: string, status: ClientStatus) => void;
	}

	let { clients, onClientClick, onStatusChange }: Props = $props();

	const columns: { id: ClientStatus; label: string; color: string }[] = [
		{ id: 'lead', label: 'Leads', color: '#8b5cf6' },
		{ id: 'prospect', label: 'Prospects', color: '#3b82f6' },
		{ id: 'active', label: 'Active', color: '#22c55e' },
		{ id: 'inactive', label: 'Inactive', color: '#6b7280' },
		{ id: 'churned', label: 'Churned', color: '#ef4444' }
	];

	function getClientsByStatus(status: ClientStatus): ClientListResponse[] {
		return clients.filter((c) => c.status === status);
	}

	function formatCurrency(value: number | null): string {
		if (value === null) return '-';
		return new Intl.NumberFormat('en-US', {
			style: 'currency',
			currency: 'USD',
			minimumFractionDigits: 0,
			maximumFractionDigits: 0
		}).format(value);
	}

	function getInitials(name: string): string {
		return name
			.split(' ')
			.map((n) => n[0])
			.join('')
			.toUpperCase()
			.slice(0, 2);
	}

	function getTotalValue(clients: ClientListResponse[]): number {
		return clients.reduce((sum, c) => sum + (c.lifetime_value || 0), 0);
	}

	// Drag and drop state
	let draggedClient = $state<ClientListResponse | null>(null);
	let dragOverColumn = $state<ClientStatus | null>(null);
	// Tracks whether a drag gesture actually started so clicks are not swallowed.
	let isDragging = false;

	function handleDragStart(e: DragEvent, client: ClientListResponse) {
		isDragging = true;
		draggedClient = client;
		if (e.dataTransfer) {
			e.dataTransfer.effectAllowed = 'move';
			e.dataTransfer.setData('text/plain', client.id);
		}
	}

	function handleDragEnd() {
		// Use a microtask delay so the onclick that fires after dragend sees
		// isDragging=true and skips navigation, then we reset for next time.
		setTimeout(() => {
			isDragging = false;
		}, 0);
		draggedClient = null;
		dragOverColumn = null;
	}

	function handleDragOver(e: DragEvent, status: ClientStatus) {
		e.preventDefault();
		dragOverColumn = status;
	}

	function handleDragLeave() {
		dragOverColumn = null;
	}

	function handleDrop(e: DragEvent, status: ClientStatus) {
		e.preventDefault();
		if (draggedClient && draggedClient.status !== status) {
			onStatusChange(draggedClient.id, status);
		}
		draggedClient = null;
		dragOverColumn = null;
	}
</script>

<div class="cr-kanban-container">
	<div class="cr-kanban">
		{#each columns as column}
			{@const columnClients = getClientsByStatus(column.id)}
			<div
				class="cr-kanban-col {dragOverColumn === column.id ? 'cr-kanban-col--dragover' : ''}"
				ondragover={(e) => handleDragOver(e, column.id)}
				ondragleave={handleDragLeave}
				ondrop={(e) => handleDrop(e, column.id)}
				role="region"
				aria-label="{column.label} column"
			>
				<!-- Column Header -->
				<div class="cr-kanban-col__header">
					<span class="cr-kanban-col__dot" style="background: {column.color}"></span>
					<span class="cr-kanban-col__stage">{column.label}</span>
					<span class="cr-kanban-col__count">{columnClients.length}</span>
				</div>
				<div class="cr-kanban-col__total">{formatCurrency(getTotalValue(columnClients))}</div>

				<!-- Cards -->
				<div class="cr-kanban-col__cards">
					{#each columnClients as client (client.id)}
						<div
							class="cr-kanban-card {draggedClient?.id === client.id ? 'cr-kanban-card--dragging' : ''}"
							style="--stage-color: {column.color}"
							draggable="true"
							ondragstart={(e) => handleDragStart(e, client)}
							ondragend={handleDragEnd}
							onclick={() => { if (!isDragging) onClientClick(client.id); }}
							role="button"
							tabindex="0"
							onkeydown={(e) => e.key === 'Enter' && onClientClick(client.id)}
							aria-label="View {client.name}"
						>
							<div class="cr-kanban-card__bar"></div>
							<div class="cr-kanban-card__body">
								<div class="cr-kanban-card__name">{client.name}</div>
								<div class="cr-kanban-card__type">
									<span class="cr-type-pill cr-type-pill--{client.type}">
										{client.type === 'company' ? 'Company' : 'Individual'}
									</span>
								</div>
								{#if client.lifetime_value}
									<div class="cr-kanban-card__meta">
										<span class="cr-kanban-card__value">{formatCurrency(client.lifetime_value)}</span>
										<span class="cr-kanban-card__deals">{client.deals_count} deals</span>
									</div>
								{/if}
								{#if client.tags && client.tags.length > 0}
									<div class="cr-kanban-card__tags">
										{#each client.tags.slice(0, 2) as tag}
											<span class="cr-kanban-card__tag">{tag}</span>
										{/each}
									</div>
								{/if}
							</div>
						</div>
					{/each}

					{#if columnClients.length === 0}
						<div class="cr-kanban-col__empty">No clients</div>
					{/if}
				</div>
			</div>
		{/each}
	</div>
</div>

<style>
	/* ─── Kanban (Foundation CRM pattern) ──────────────────────── */
	.cr-kanban-container {
		flex: 1;
		overflow: auto;
		padding: 20px 24px;
	}
	.cr-kanban {
		display: flex;
		gap: 12px;
		min-width: max-content;
		height: 100%;
	}
	.cr-kanban-col {
		flex-shrink: 0;
		width: 260px;
		display: flex;
		flex-direction: column;
		gap: 6px;
		padding: 12px;
		border-radius: 14px;
		background: var(--dbg2, #f5f5f5);
		border: 1px solid var(--dbd, #e0e0e0);
		transition: box-shadow 0.12s;
	}
	.cr-kanban-col--dragover {
		box-shadow: 0 0 0 2px var(--dt3, #888);
	}
	.cr-kanban-col__header {
		display: flex;
		align-items: center;
		gap: 6px;
		padding: 0 2px;
	}
	.cr-kanban-col__dot {
		width: 8px;
		height: 8px;
		border-radius: 50%;
		flex-shrink: 0;
	}
	.cr-kanban-col__stage {
		flex: 1;
		font-size: 11px;
		font-weight: 700;
		color: var(--dt3, #888);
		letter-spacing: 0.01em;
		text-transform: uppercase;
	}
	.cr-kanban-col__count {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		min-width: 20px;
		height: 20px;
		padding: 0 6px;
		border-radius: 9999px;
		background: var(--dbg);
		border: 1px solid var(--dbd, #e0e0e0);
		font-size: 10px;
		font-weight: 700;
		color: var(--dt3, #888);
	}
	.cr-kanban-col__total {
		font-size: 13px;
		font-weight: 700;
		color: var(--dt, #111);
		padding: 0 2px 6px;
		letter-spacing: -0.01em;
	}
	.cr-kanban-col__cards {
		display: flex;
		flex-direction: column;
		gap: 8px;
		flex: 1;
		overflow-y: auto;
	}
	.cr-kanban-col__empty {
		text-align: center;
		padding: 24px 0;
		font-size: 12px;
		color: var(--dt4, #bbb);
	}

	/* Cards */
	.cr-kanban-card {
		position: relative;
		display: flex;
		flex-direction: column;
		padding: 10px 10px 10px 8px;
		border-radius: 10px;
		border: 1px solid var(--dbd, #e0e0e0);
		background: var(--dbg);
		overflow: hidden;
		cursor: grab;
		transition: border-color 0.12s, box-shadow 0.12s;
	}
	.cr-kanban-card:hover {
		border-color: var(--dbd2, #f0f0f0);
		box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
	}
	.cr-kanban-card--dragging {
		opacity: 0.5;
	}
	.cr-kanban-card__bar {
		position: absolute;
		top: 0;
		left: 0;
		width: 3px;
		height: 100%;
		background: var(--stage-color, var(--dbd2, #f0f0f0));
		border-radius: 3px 0 0 3px;
	}
	.cr-kanban-card__body {
		padding-left: 4px;
	}
	.cr-kanban-card__name {
		font-size: 13px;
		font-weight: 600;
		color: var(--dt, #111);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	.cr-kanban-card__type {
		margin-top: 4px;
	}
	.cr-kanban-card__meta {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 6px;
		margin-top: 8px;
	}
	.cr-kanban-card__value {
		font-size: 12px;
		font-weight: 700;
		color: var(--dt, #111);
	}
	.cr-kanban-card__deals {
		font-size: 10px;
		color: var(--dt3, #888);
		font-weight: 500;
	}
	.cr-kanban-card__tags {
		display: flex;
		gap: 4px;
		margin-top: 6px;
		flex-wrap: wrap;
	}
	.cr-kanban-card__tag {
		padding: 1px 6px;
		font-size: 10px;
		border-radius: 9999px;
		background: var(--dbg2, #f5f5f5);
		color: var(--dt3, #888);
		border: 1px solid var(--dbd, #e0e0e0);
	}

	/* Type pills */
	.cr-type-pill {
		display: inline-flex;
		align-items: center;
		height: 18px;
		padding: 0 7px;
		border-radius: 9999px;
		font-size: 10px;
		font-weight: 600;
	}
	.cr-type-pill--company { background: color-mix(in srgb, var(--bos-category-productivity) 10%, transparent); color: #6366f1; }
	.cr-type-pill--individual { background: color-mix(in srgb, var(--bos-category-ai) 10%, transparent); color: #8b5cf6; }
	:global(.dark) .cr-type-pill--company { background: color-mix(in srgb, var(--bos-category-productivity) 15%, transparent); color: var(--bos-category-productivity); }
	:global(.dark) .cr-type-pill--individual { background: color-mix(in srgb, var(--bos-category-ai) 15%, transparent); color: var(--bos-category-ai); }
</style>
