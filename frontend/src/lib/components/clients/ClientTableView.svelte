<script lang="ts">
	import type { ClientListResponse, ClientStatus } from '$lib/api';


	interface Props {
		clients: ClientListResponse[];
		onClientClick: (id: string) => void;
		onStatusChange: (id: string, status: ClientStatus) => void;
	}

	let { clients, onClientClick, onStatusChange }: Props = $props();

	function formatCurrency(value: number | null): string {
		if (value === null) return '-';
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

	function getInitials(name: string): string {
		return name
			.split(' ')
			.map((n) => n[0])
			.join('')
			.toUpperCase()
			.slice(0, 2);
	}
</script>

<div class="cr-table-container">
	<div class="cr-table-wrap">
		<table class="cr-table">
			<thead>
				<tr>
					<th class="cr-table__th"><span class="cr-table__th-label">Client</span></th>
					<th class="cr-table__th"><span class="cr-table__th-label">Status</span></th>
					<th class="cr-table__th"><span class="cr-table__th-label">Contact</span></th>
					<th class="cr-table__th"><span class="cr-table__th-label">Last Contact</span></th>
					<th class="cr-table__th"><span class="cr-table__th-label">Value</span></th>
					<th class="cr-table__th"><span class="cr-table__th-label">Deals</span></th>
				</tr>
			</thead>
			<tbody>
				{#each clients as client (client.id)}
					<tr
						class="cr-table__row"
						onclick={() => onClientClick(client.id)}
					>
						<td class="cr-table__td">
							<div class="cr-table__name-cell">
							<div class="cr-logo cr-logo--sm">
								<span class="cr-logo__initials">{getInitials(client.name)}</span>
							</div>
							<div>
								<div class="cr-table__name-row">
									<span class="cr-table__name">{client.name}</span>
									<span class="cr-type-pill cr-type-pill--{client.type}">
										{client.type === 'company' ? 'Co' : 'Ind'}
									</span>
								</div>
								{#if client.tags && client.tags.length > 0}
									<div class="cr-table__tags">
										{#each client.tags.slice(0, 2) as tag}
											<span class="cr-table__tag">{tag}</span>
										{/each}
										{#if client.tags.length > 2}
											<span class="cr-table__tag-more">+{client.tags.length - 2}</span>
										{/if}
									</div>
								{/if}
							</div>
							</div>
						</td>
						<td class="cr-table__td">
							<select
								value={client.status}
								onclick={(e) => e.stopPropagation()}
								onchange={(e) => {
									e.stopPropagation();
									onStatusChange(client.id, (e.target as HTMLSelectElement).value as ClientStatus);
								}}
								class="cr-table__status-select cr-status-select--{client.status}"
								aria-label="Change status for {client.name}"
							>
								<option value="lead">Lead</option>
								<option value="prospect">Prospect</option>
								<option value="active">Active</option>
								<option value="inactive">Inactive</option>
								<option value="churned">Churned</option>
							</select>
						</td>
						<td class="cr-table__td">
							<div class="cr-table__contact">{client.email || '-'}</div>
							<div class="cr-table__contact-sub">{client.phone || ''}</div>
						</td>
						<td class="cr-table__td cr-table__td--muted">
							{formatDate(client.last_contacted_at)}
						</td>
						<td class="cr-table__td">
							<div class="cr-table__value">{formatCurrency(client.lifetime_value)}</div>
							{#if client.active_deals_value > 0}
								<div class="cr-table__pipeline-value">{formatCurrency(client.active_deals_value)} in pipeline</div>
							{/if}
						</td>
						<td class="cr-table__td cr-table__td--muted">
							<span>{client.deals_count} deals</span>
							<span class="cr-table__separator">·</span>
							<span>{client.contacts_count} contacts</span>
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
</div>

<style>
	/* ─── Table (Foundation CRM pattern) ───────────────────────── */
	.cr-table-container {
		flex: 1;
		overflow: auto;
	}
	.cr-table-wrap {
		border-radius: 0;
		border-bottom: 1px solid var(--dbd);
	}
	.cr-table {
		width: 100%;
		border-collapse: collapse;
		font-size: 13px;
	}
	.cr-table__th {
		padding: 0;
		text-align: left;
		border-bottom: 1px solid var(--dbd);
		background: var(--dbg2, #f5f5f5);
		position: sticky;
		top: 0;
		z-index: 1;
	}
	.cr-table__th-label {
		display: block;
		padding: 9px 16px;
		font-size: 11px;
		font-weight: 700;
		text-transform: uppercase;
		letter-spacing: 0.04em;
		color: var(--dt3, #888);
	}
	.cr-table__row {
		cursor: pointer;
		transition: background 0.1s;
	}
	.cr-table__row:hover {
		background: var(--dbg2, #f5f5f5);
	}
	.cr-table__row:not(:last-child) .cr-table__td {
		border-bottom: 1px solid var(--dbd);
	}
	.cr-table__td {
		padding: 10px 16px;
		color: var(--dt, #111);
		vertical-align: middle;
		white-space: nowrap;
	}
	.cr-table__name-cell {
		display: flex;
		align-items: center;
		gap: 10px;
	}
	.cr-table__td--muted {
		color: var(--dt3, #888);
		font-size: 12px;
	}
	.cr-table__name-row {
		display: flex;
		align-items: center;
		gap: 6px;
	}
	.cr-table__name {
		font-size: 13px;
		font-weight: 600;
		color: var(--dt, #111);
	}
	.cr-table__tags {
		display: flex;
		gap: 3px;
		margin-top: 3px;
	}
	.cr-table__tag {
		padding: 1px 6px;
		font-size: 10px;
		border-radius: 9999px;
		background: var(--dbg2, #f5f5f5);
		color: var(--dt3, #888);
		border: 1px solid var(--dbd);
	}
	.cr-table__tag-more {
		font-size: 10px;
		color: var(--dt4, #bbb);
	}
	.cr-table__contact {
		font-size: 13px;
		color: var(--dt, #111);
	}
	.cr-table__contact-sub {
		font-size: 11px;
		color: var(--dt3, #888);
		margin-top: 1px;
	}
	.cr-table__value {
		font-size: 13px;
		font-weight: 600;
		color: var(--dt, #111);
	}
	.cr-table__pipeline-value {
		font-size: 11px;
		color: var(--color-success);
		margin-top: 1px;
	}
	.cr-table__separator {
		margin: 0 4px;
		color: var(--dbd);
	}

	/* Status Select (styled native) */
	.cr-table__status-select {
		padding: 3px 8px;
		font-size: 11px;
		font-weight: 600;
		border-radius: 9999px;
		border: 1px solid transparent !important;
		cursor: pointer;
		outline: none !important;
		appearance: none;
		-webkit-appearance: none;
		box-shadow: none !important;
	}
	.cr-table__status-select:focus {
		border-color: transparent !important;
		outline: none !important;
		box-shadow: none !important;
	}
	.cr-status-select--active { background: color-mix(in srgb, var(--color-success) 12%, transparent); color: var(--color-success); }
	.cr-status-select--lead { background: color-mix(in srgb, var(--dt3) 12%, transparent); color: var(--dt3); }
	.cr-status-select--prospect { background: color-mix(in srgb, var(--color-warning) 12%, transparent); color: var(--color-warning); }
	.cr-status-select--inactive { background: color-mix(in srgb, var(--dt4) 15%, transparent); color: var(--dt3); }
	.cr-status-select--churned { background: color-mix(in srgb, var(--color-error) 12%, transparent); color: var(--color-error); }

	/* Logo small variant */
	.cr-logo {
		width: 40px;
		height: 40px;
		border-radius: 10px;
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
		background: color-mix(in srgb, var(--dt) 8%, transparent);
	}
	.cr-logo--sm { width: 34px; height: 34px; border-radius: 8px; }
	.cr-logo__initials {
		font-size: 12px;
		font-weight: 800;
		letter-spacing: -0.01em;
		line-height: 1;
		color: var(--dt2);
	}

	/* Type pills */
	.cr-type-pill {
		display: inline-flex;
		align-items: center;
		height: 16px;
		padding: 0 5px;
		border-radius: 9999px;
		font-size: 9px;
		font-weight: 600;
		letter-spacing: 0.02em;
		flex-shrink: 0;
	}
	.cr-type-pill--company { background: color-mix(in srgb, var(--dt) 8%, transparent); color: var(--dt2); }
	.cr-type-pill--individual { background: color-mix(in srgb, var(--dt) 6%, transparent); color: var(--dt3); }
</style>
