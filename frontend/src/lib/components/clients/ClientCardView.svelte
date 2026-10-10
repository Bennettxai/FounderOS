<script lang="ts">
	import type { ClientListResponse } from '$lib/api';
	import { statusColors, statusLabels } from '$lib/stores/clients';

	interface Props {
		clients: ClientListResponse[];
		onClientClick: (id: string) => void;
	}

	let { clients, onClientClick }: Props = $props();

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

	function getTimeAgo(dateStr: string | null): string {
		if (!dateStr) return 'Never';
		const date = new Date(dateStr);
		const now = new Date();
		const diffMs = now.getTime() - date.getTime();
		const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));

		if (diffDays === 0) return 'Today';
		if (diffDays === 1) return 'Yesterday';
		if (diffDays < 7) return `${diffDays} days ago`;
		if (diffDays < 30) return `${Math.floor(diffDays / 7)} weeks ago`;
		if (diffDays < 365) return `${Math.floor(diffDays / 30)} months ago`;
		return `${Math.floor(diffDays / 365)} years ago`;
	}
</script>

<div class="cr-cards-container">
	<div class="cr-card-grid">
		{#each clients as client (client.id)}
			<button
				onclick={() => onClientClick(client.id)}
				class="cr-client-card"
				aria-label="View {client.name}"
			>
				<!-- Header -->
				<div class="cr-client-card__header">
					<div class="cr-logo" style="background: {client.type === 'company' ? 'rgba(99,102,241,0.1)' : 'rgba(139,92,246,0.1)'}; border-color: {client.type === 'company' ? 'rgba(99,102,241,0.15)' : 'rgba(139,92,246,0.15)'}">
						<span class="cr-logo__initials" style="color: {client.type === 'company' ? '#6366f1' : '#8b5cf6'}">{getInitials(client.name)}</span>
					</div>
					<div class="cr-client-card__header-info">
						<span class="cr-client-card__company">{client.name}</span>
						<div class="cr-client-card__badges">
							<span class="cr-type-pill cr-type-pill--{client.type}">
								{client.type === 'company' ? 'Company' : 'Individual'}
							</span>
							<span class="cr-status-pill cr-status-pill--{client.status}">
								{statusLabels[client.status]}
							</span>
						</div>
					</div>
				</div>

				<!-- Contact Info -->
				<div class="cr-client-card__contact">
					{#if client.email}
						<div class="cr-client-card__contact-email">
							<svg width="12" height="12" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
							</svg>
							<span>{client.email}</span>
						</div>
					{/if}
					{#if client.phone}
						<div class="cr-client-card__contact-email" style="margin-top: 4px;">
							<svg width="12" height="12" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 5a2 2 0 012-2h3.28a1 1 0 01.948.684l1.498 4.493a1 1 0 01-.502 1.21l-2.257 1.13a11.042 11.042 0 005.516 5.516l1.13-2.257a1 1 0 011.21-.502l4.493 1.498a1 1 0 01.684.949V19a2 2 0 01-2 2h-1C9.716 21 3 14.284 3 6V5z" />
							</svg>
							<span>{client.phone}</span>
						</div>
					{/if}
				</div>

				<!-- Stats -->
				<div class="cr-client-card__meta">
					<div>
						<span class="cr-deal-badge">
							<svg width="10" height="10" viewBox="0 0 16 16" fill="currentColor" aria-hidden="true"><path d="M8 1.5a.5.5 0 01.5.5v5.5H14a.5.5 0 010 1H8.5V14a.5.5 0 01-1 0V8.5H2a.5.5 0 010-1h5.5V2a.5.5 0 01.5-.5z"/></svg>
							{formatCurrency(client.lifetime_value)}
						</span>
						<span class="cr-client-card__stat">{client.deals_count} deals</span>
					</div>
					<span class="cr-client-card__last-contact">
						<svg width="10" height="10" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
						{getTimeAgo(client.last_contacted_at)}
					</span>
				</div>

				<!-- Tags -->
				{#if client.tags && client.tags.length > 0}
					<div class="cr-client-card__tags">
						{#each client.tags.slice(0, 2) as tag}
							<span class="cr-client-card__tag">{tag}</span>
						{/each}
						{#if client.tags.length > 2}
							<span class="cr-client-card__tag-more">+{client.tags.length - 2}</span>
						{/if}
					</div>
				{/if}
			</button>
		{/each}
	</div>
</div>

<style>
	/* ─── Card Grid (Foundation CRM pattern) ───────────────────── */
	.cr-cards-container {
		flex: 1;
		overflow: auto;
		padding: 20px 24px;
	}
	.cr-card-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
		gap: 12px;
	}
	.cr-client-card {
		display: flex;
		flex-direction: column;
		gap: 10px;
		padding: 14px;
		border-radius: 14px;
		border: 1px solid var(--dbd, #e0e0e0);
		background: var(--dbg);
		text-align: left;
		cursor: pointer;
		transition: border-color 0.13s, box-shadow 0.13s;
	}
	.cr-client-card:hover {
		border-color: var(--dbd2, #f0f0f0);
		box-shadow: 0 3px 14px rgba(0, 0, 0, 0.06);
	}
	.cr-client-card__header {
		display: flex;
		align-items: flex-start;
		gap: 10px;
	}
	.cr-client-card__header-info {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: 5px;
	}
	.cr-client-card__company {
		font-size: 14px;
		font-weight: 700;
		color: var(--dt, #111);
		letter-spacing: -0.01em;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	.cr-client-card__badges {
		display: flex;
		align-items: center;
		gap: 5px;
	}
	.cr-client-card__contact {
		border-top: 1px solid var(--dbd, #e0e0e0);
		padding-top: 10px;
	}
	.cr-client-card__contact-email {
		display: flex;
		align-items: center;
		gap: 5px;
		font-size: 11px;
		color: var(--dt3, #888);
	}
	.cr-client-card__contact-email svg { flex-shrink: 0; color: var(--dt4, #bbb); }
	.cr-client-card__meta {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 6px;
		border-top: 1px solid var(--dbd, #e0e0e0);
		padding-top: 10px;
	}
	.cr-client-card__stat {
		font-size: 11px;
		color: var(--dt3, #888);
		margin-left: 8px;
		font-weight: 500;
	}
	.cr-client-card__last-contact {
		display: flex;
		align-items: center;
		gap: 4px;
		font-size: 10px;
		color: var(--dt3, #888);
		font-weight: 500;
	}
	.cr-client-card__tags {
		display: flex;
		gap: 4px;
		flex-wrap: wrap;
	}
	.cr-client-card__tag {
		padding: 2px 7px;
		font-size: 10px;
		font-weight: 500;
		border-radius: 9999px;
		background: var(--dbg2, #f5f5f5);
		color: var(--dt3, #888);
		border: 1px solid var(--dbd, #e0e0e0);
	}
	.cr-client-card__tag-more {
		font-size: 10px;
		color: var(--dt4, #bbb);
		padding: 2px 4px;
	}

	/* ─── Logo (Foundation CRM) ────────────────────────────────── */
	.cr-logo {
		width: 42px;
		height: 42px;
		border-radius: 10px;
		border: 1.5px solid transparent;
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
	}
	.cr-logo__initials {
		font-size: 14px;
		font-weight: 800;
		letter-spacing: -0.01em;
		line-height: 1;
	}

	/* ─── Status & Type Pills (Foundation CRM) ─────────────────── */
	.cr-status-pill {
		display: inline-flex;
		align-items: center;
		height: 18px;
		padding: 0 7px;
		border-radius: 9999px;
		font-size: 10px;
		font-weight: 600;
		letter-spacing: 0.01em;
	}
	.cr-status-pill--active { background: var(--bos-status-success-bg); color: var(--bos-status-success-text); }
	.cr-status-pill--lead { background: var(--bos-status-neutral-bg); color: var(--bos-status-neutral-text); }
	.cr-status-pill--prospect { background: rgba(59, 130, 246, 0.12); color: #2563eb; }
	.cr-status-pill--inactive { background: rgba(156, 163, 175, 0.12); color: #6b7280; }
	.cr-status-pill--churned { background: rgba(239, 68, 68, 0.12); color: #ef4444; }
	:global(.dark) .cr-status-pill--active { background: var(--bos-status-success-bg); color: var(--bos-status-success); }
	:global(.dark) .cr-status-pill--lead { background: var(--bos-status-neutral-bg); color: var(--bos-status-neutral); }
	:global(.dark) .cr-status-pill--prospect { background: rgba(59, 130, 246, 0.15); color: #60a5fa; }
	:global(.dark) .cr-status-pill--inactive { background: var(--bos-status-neutral-bg); color: var(--bos-status-neutral); }
	:global(.dark) .cr-status-pill--churned { background: rgba(239, 68, 68, 0.15); color: #f87171; }
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

	/* ─── Deal Badge (Foundation CRM) ──────────────────────────── */
	.cr-deal-badge {
		display: inline-flex;
		align-items: center;
		gap: 3px;
		padding: 2px 8px;
		border-radius: 9999px;
		background: var(--bos-status-success-bg);
		border: 1px solid rgba(34, 197, 94, 0.15);
		font-size: 11px;
		font-weight: 700;
		color: var(--bos-status-success-text);
	}
	:global(.dark) .cr-deal-badge {
		background: var(--bos-status-success-bg);
		border-color: rgba(34, 197, 94, 0.2);
		color: var(--bos-status-success);
	}
</style>
