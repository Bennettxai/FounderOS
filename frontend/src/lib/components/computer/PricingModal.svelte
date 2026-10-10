<script lang="ts">
	import { fade, fly } from 'svelte/transition';

	interface Plan {
		id: string;
		name: string;
		price_cents: number;
		interval: 'month' | 'year';
		ram_gb: number;
		cpu_cores: number;
		storage_gb: number;
		credits: number;
		seats: string;
		features: string[];
		is_popular?: boolean;
		is_current?: boolean;
	}

	interface Props {
		plans: Plan[];
		provisioning: boolean;
		hasSubscription: boolean;
		onClose: () => void;
		onSelectPlan: (planId: string) => void;
	}

	let { plans, provisioning, hasSubscription, onClose, onSelectPlan }: Props = $props();

	function formatPrice(cents: number): string {
		return `$${(cents / 100).toFixed(0)}`;
	}
</script>

<div
	class="cp-modal-backdrop"
	onclick={onClose}
	role="dialog"
	aria-modal="true"
	aria-label="Choose a plan"
	in:fade={{ duration: 180 }}
	out:fade={{ duration: 150 }}
>
	<div
		class="cp-modal"
		onclick={(e) => e.stopPropagation()}
		in:fly={{ y: 24, duration: 220 }}
	>
		<div class="cp-modal-header">
			<div>
				<h2 class="cp-modal-title">Choose Your Plan</h2>
				<p class="cp-modal-subtitle">Scale your cloud computer as your team grows</p>
			</div>
			<button
				class="cp-modal-close"
				onclick={onClose}
				aria-label="Close pricing modal"
			>
				<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
					<path d="M18 6L6 18M6 6l12 12"/>
				</svg>
			</button>
		</div>

		<div class="cp-plans-grid">
			{#each plans as plan}
				<div class="cp-plan-card {plan.is_current ? 'cp-plan-card--current' : ''} {plan.is_popular ? 'cp-plan-card--popular' : ''}">
					{#if plan.is_popular}
						<div class="cp-plan-badge cp-plan-badge--popular">Popular</div>
					{/if}
					{#if plan.is_current}
						<div class="cp-plan-badge cp-plan-badge--current">Current Plan</div>
					{/if}

					<div class="cp-plan-header">
						<h3 class="cp-plan-name">{plan.name}</h3>
						<div class="cp-plan-price">
							<span class="cp-plan-price-amount">{formatPrice(plan.price_cents)}</span>
							<span class="cp-plan-price-period">/{plan.interval}</span>
						</div>
					</div>

					<ul class="cp-plan-specs">
						<li class="cp-plan-spec">
							<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
								<rect x="2" y="3" width="20" height="14" rx="2"/>
								<path d="M8 21h8M12 17v4"/>
							</svg>
							{plan.ram_gb} GB RAM
						</li>
						<li class="cp-plan-spec">
							<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
								<rect x="4" y="4" width="16" height="16" rx="2"/>
								<rect x="9" y="9" width="6" height="6"/>
								<path d="M15 2v2M9 2v2M2 9h2M2 15h2M22 9h-2M22 15h-2M15 22v-2M9 22v-2"/>
							</svg>
							{plan.cpu_cores} vCPUs
						</li>
						<li class="cp-plan-spec">
							<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
								<ellipse cx="12" cy="5" rx="9" ry="3"/>
								<path d="M21 12c0 1.66-4 3-9 3s-9-1.34-9-3"/>
								<path d="M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5"/>
							</svg>
							{plan.storage_gb} GB SSD
						</li>
						<li class="cp-plan-spec">
							<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
								<circle cx="12" cy="12" r="10"/>
								<path d="M12 6v6l4 2"/>
							</svg>
							{plan.credits >= 1000 ? `${plan.credits / 1000}K` : plan.credits} credits/mo
						</li>
						<li class="cp-plan-spec">
							<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
								<path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/>
								<circle cx="9" cy="7" r="4"/>
								<path d="M23 21v-2a4 4 0 0 0-3-3.87M16 3.13a4 4 0 0 1 0 7.75"/>
							</svg>
							{plan.seats} seats
						</li>
						{#each plan.features as feature}
							<li class="cp-plan-spec">
								<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
									<polyline points="20 6 9 17 4 12"/>
								</svg>
								{feature}
							</li>
						{/each}
					</ul>

					{#if plan.is_current}
						<button class="cp-plan-btn cp-plan-btn--current" disabled aria-disabled="true">
							Current Plan
						</button>
					{:else}
						<button
							class="cp-plan-btn cp-plan-btn--upgrade"
							onclick={() => onSelectPlan(plan.id)}
							disabled={provisioning}
						>
							{provisioning ? 'Creating...' : hasSubscription ? 'Upgrade' : 'Get Started'}
						</button>
					{/if}
				</div>
			{/each}
		</div>
	</div>
</div>

<style>
	.cp-modal-backdrop {
		position: fixed;
		inset: 0;
		z-index: 100;
		background: rgba(0, 0, 0, 0.55);
		backdrop-filter: blur(6px);
		-webkit-backdrop-filter: blur(6px);
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 24px;
	}

	.cp-modal {
		background: var(--dbg, #ffffff);
		border: 1px solid var(--dbd, rgba(0,0,0,0.1));
		border-radius: 16px;
		width: 100%;
		max-width: 860px;
		max-height: calc(100vh - 48px);
		overflow-y: auto;
		box-shadow: 0 24px 80px rgba(0, 0, 0, 0.18);
	}
	:global(.dark) .cp-modal {
		background: #1a1a1f;
		border-color: rgba(255,255,255,0.1);
		box-shadow: 0 24px 80px rgba(0, 0, 0, 0.5);
	}

	.cp-modal-header {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: 16px;
		padding: 28px 28px 20px;
		border-bottom: 1px solid var(--dbd, rgba(0,0,0,0.07));
	}
	:global(.dark) .cp-modal-header { border-color: rgba(255,255,255,0.07); }

	.cp-modal-title {
		font-size: 20px;
		font-weight: 700;
		color: var(--dt, #111827);
		margin: 0 0 4px;
		letter-spacing: -0.02em;
	}
	:global(.dark) .cp-modal-title { color: var(--dt, #f3f4f6); }

	.cp-modal-subtitle {
		font-size: 13px;
		color: var(--dt3, #9ca3af);
		margin: 0;
	}

	.cp-modal-close {
		width: 32px;
		height: 32px;
		border-radius: 8px;
		background: transparent;
		border: 1px solid var(--dbd, rgba(0,0,0,0.08));
		color: var(--dt3, #9ca3af);
		cursor: pointer;
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
		transition: background 120ms, color 120ms;
	}
	.cp-modal-close:hover {
		background: var(--dbg2, rgba(0,0,0,0.05));
		color: var(--dt, #111827);
	}
	:global(.dark) .cp-modal-close {
		border-color: rgba(255,255,255,0.08);
		color: rgba(255,255,255,0.4);
	}
	:global(.dark) .cp-modal-close:hover {
		background: rgba(255,255,255,0.06);
		color: #f3f4f6;
	}

	.cp-plans-grid {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: 16px;
		padding: 24px 28px 28px;
	}

	@media (max-width: 700px) {
		.cp-plans-grid { grid-template-columns: 1fr; }
	}

	.cp-plan-card {
		background: var(--dbg2, rgba(0,0,0,0.03));
		border: 1px solid var(--dbd, rgba(0,0,0,0.08));
		border-radius: 12px;
		padding: 24px 20px 20px;
		display: flex;
		flex-direction: column;
		gap: 16px;
		position: relative;
		transition: border-color 140ms ease-out, box-shadow 140ms ease-out;
	}
	:global(.dark) .cp-plan-card {
		background: rgba(255,255,255,0.04);
		border-color: rgba(255,255,255,0.08);
	}

	.cp-plan-card--popular {
		border-color: #3b82f6;
		box-shadow: 0 0 0 1px rgba(59, 130, 246, 0.2);
	}
	:global(.dark) .cp-plan-card--popular {
		border-color: #3b82f6;
		box-shadow: 0 0 0 1px rgba(59, 130, 246, 0.3);
	}

	.cp-plan-card--current {
		border-color: #22c55e;
		box-shadow: 0 0 0 1px rgba(34, 197, 94, 0.2);
	}

	.cp-plan-badge {
		position: absolute;
		top: -11px;
		left: 50%;
		transform: translateX(-50%);
		font-size: 10px;
		font-weight: 700;
		padding: 3px 10px;
		border-radius: 20px;
		white-space: nowrap;
		letter-spacing: 0.04em;
		text-transform: uppercase;
	}

	.cp-plan-badge--popular {
		background: linear-gradient(90deg, #3b82f6, #8b5cf6);
		color: #ffffff;
	}

	.cp-plan-badge--current {
		background: rgba(34, 197, 94, 0.15);
		color: #16a34a;
		border: 1px solid rgba(34, 197, 94, 0.3);
	}
	:global(.dark) .cp-plan-badge--current {
		background: rgba(34, 197, 94, 0.2);
		color: #4ade80;
	}

	.cp-plan-header {
		display: flex;
		flex-direction: column;
		gap: 4px;
	}

	.cp-plan-name {
		font-size: 14px;
		font-weight: 700;
		color: var(--dt, #111827);
		margin: 0;
		text-transform: uppercase;
		letter-spacing: 0.05em;
	}
	:global(.dark) .cp-plan-name { color: var(--dt, #f3f4f6); }

	.cp-plan-price {
		display: flex;
		align-items: baseline;
		gap: 2px;
	}

	.cp-plan-price-amount {
		font-size: 28px;
		font-weight: 800;
		color: var(--dt, #111827);
		letter-spacing: -0.03em;
	}
	:global(.dark) .cp-plan-price-amount { color: var(--dt, #f3f4f6); }

	.cp-plan-price-period {
		font-size: 13px;
		color: var(--dt3, #9ca3af);
		font-weight: 400;
	}

	.cp-plan-specs {
		list-style: none;
		padding: 0;
		margin: 0;
		display: flex;
		flex-direction: column;
		gap: 8px;
		flex: 1;
	}

	.cp-plan-spec {
		display: flex;
		align-items: center;
		gap: 8px;
		font-size: 12.5px;
		color: var(--dt2, #4b5563);
	}
	:global(.dark) .cp-plan-spec { color: var(--dt2, rgba(255,255,255,0.6)); }

	.cp-plan-spec svg {
		flex-shrink: 0;
		opacity: 0.6;
	}

	.cp-plan-btn {
		width: 100%;
		padding: 10px;
		border-radius: 8px;
		font-size: 13px;
		font-weight: 600;
		cursor: pointer;
		border: none;
		transition: background 140ms, opacity 140ms;
	}

	.cp-plan-btn--upgrade {
		background: linear-gradient(135deg, #3b82f6, #8b5cf6);
		color: #ffffff;
	}
	.cp-plan-btn--upgrade:hover { opacity: 0.9; }

	.cp-plan-btn--current {
		background: rgba(34, 197, 94, 0.1);
		color: #16a34a;
		cursor: default;
		border: 1px solid rgba(34, 197, 94, 0.2);
	}
	:global(.dark) .cp-plan-btn--current {
		background: rgba(34, 197, 94, 0.12);
		color: #4ade80;
	}
</style>
