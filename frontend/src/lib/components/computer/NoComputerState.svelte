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
		provisionError: string | null;
		onGetStarted: () => void;
		onDismissError: () => void;
	}

	let { plans, provisionError, onGetStarted, onDismissError }: Props = $props();

	function formatPrice(cents: number): string {
		return `$${(cents / 100).toFixed(0)}`;
	}
</script>

<div class="cp-empty" in:fade={{ duration: 250 }}>
	<div class="cp-empty-card" in:fly={{ y: 16, duration: 300 }}>
		{#if provisionError}
			<div class="cp-error-banner" role="alert">
				<span>{provisionError}</span>
				<button onclick={onDismissError}>Dismiss</button>
			</div>
		{/if}
		<div class="cp-empty-icon" aria-hidden="true">
			<svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.25" stroke-linecap="round" stroke-linejoin="round">
				<rect x="2" y="3" width="20" height="14" rx="2"/>
				<path d="M8 21h8M12 17v4"/>
			</svg>
		</div>
		<h1 class="cp-empty-title">Create Your Computer</h1>
		<p class="cp-empty-subtitle">
			Access BusinessOS from anywhere. Invite your team. Run AI agents in the cloud.
		</p>

		<div class="cp-empty-plans">
			{#each plans.slice(0, 3) as plan}
				<div class="cp-empty-plan">
					<div class="cp-empty-plan-name">{plan.name}</div>
					<div class="cp-empty-plan-price">{formatPrice(plan.price_cents)}<span>/mo</span></div>
					<ul class="cp-empty-plan-specs">
						<li>{plan.ram_gb} GB RAM</li>
						<li>{plan.cpu_cores} vCPUs</li>
						<li>{plan.storage_gb} GB SSD</li>
					</ul>
					{#if plan.is_popular}
						<div class="cp-empty-plan-badge">Popular</div>
					{/if}
				</div>
			{/each}
		</div>

		<button class="cp-btn cp-btn--primary cp-btn--lg" onclick={onGetStarted}>
			Get Started
		</button>
	</div>
</div>

<style>
	.cp-empty {
		display: flex;
		align-items: center;
		justify-content: center;
		min-height: calc(100vh - 160px);
	}

	.cp-empty-card {
		background: var(--dbg, #ffffff);
		border: 1px solid var(--dbd, rgba(0,0,0,0.07));
		border-radius: 16px;
		padding: 48px 40px;
		display: flex;
		flex-direction: column;
		align-items: center;
		text-align: center;
		gap: 16px;
		max-width: 640px;
		width: 100%;
	}
	:global(.dark) .cp-empty-card {
		background: var(--dbg, rgba(255,255,255,0.03));
		border-color: var(--dbd, rgba(255,255,255,0.07));
	}

	.cp-empty-icon {
		width: 72px;
		height: 72px;
		border-radius: 18px;
		background: linear-gradient(135deg, rgba(59, 130, 246, 0.12), rgba(139, 92, 246, 0.12));
		color: #3b82f6;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.cp-error-banner {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 12px;
		width: 100%;
		padding: 10px 14px;
		border-radius: 8px;
		background: rgba(239, 68, 68, 0.1);
		border: 1px solid rgba(239, 68, 68, 0.25);
		color: #dc2626;
		font-size: 13px;
		margin-bottom: 12px;
	}

	.cp-error-banner button {
		background: none;
		border: none;
		color: #dc2626;
		font-weight: 600;
		cursor: pointer;
		font-size: 12px;
		padding: 2px 8px;
		border-radius: 4px;
	}

	.cp-error-banner button:hover {
		background: rgba(239, 68, 68, 0.15);
	}

	.cp-empty-title {
		font-size: 26px;
		font-weight: 700;
		color: var(--dt, #111827);
		margin: 0;
		letter-spacing: -0.03em;
	}
	:global(.dark) .cp-empty-title { color: var(--dt, #f3f4f6); }

	.cp-empty-subtitle {
		font-size: 14px;
		color: var(--dt3, #6b7280);
		margin: 0;
		max-width: 400px;
		line-height: 1.6;
	}
	:global(.dark) .cp-empty-subtitle { color: var(--dt3, rgba(255,255,255,0.4)); }

	.cp-empty-plans {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: 10px;
		width: 100%;
		margin: 8px 0;
	}

	@media (max-width: 600px) {
		.cp-empty-plans { grid-template-columns: 1fr; }
	}

	.cp-empty-plan {
		background: var(--dbg2, rgba(0,0,0,0.03));
		border: 1px solid var(--dbd, rgba(0,0,0,0.07));
		border-radius: 10px;
		padding: 14px 12px;
		display: flex;
		flex-direction: column;
		gap: 4px;
		position: relative;
	}
	:global(.dark) .cp-empty-plan {
		background: rgba(255,255,255,0.04);
		border-color: rgba(255,255,255,0.08);
	}

	.cp-empty-plan-name {
		font-size: 12px;
		font-weight: 600;
		color: var(--dt2, #374151);
		text-transform: uppercase;
		letter-spacing: 0.05em;
	}
	:global(.dark) .cp-empty-plan-name { color: var(--dt2, rgba(255,255,255,0.6)); }

	.cp-empty-plan-price {
		font-size: 20px;
		font-weight: 700;
		color: var(--dt, #111827);
		letter-spacing: -0.02em;
		margin-bottom: 6px;
	}
	:global(.dark) .cp-empty-plan-price { color: var(--dt, #f3f4f6); }

	.cp-empty-plan-price span {
		font-size: 12px;
		font-weight: 400;
		color: var(--dt3, #9ca3af);
	}

	.cp-empty-plan-specs {
		list-style: none;
		padding: 0;
		margin: 0;
		display: flex;
		flex-direction: column;
		gap: 2px;
	}

	.cp-empty-plan-specs li {
		font-size: 11px;
		color: var(--dt3, #6b7280);
	}
	:global(.dark) .cp-empty-plan-specs li { color: var(--dt3, rgba(255,255,255,0.4)); }

	.cp-empty-plan-badge {
		position: absolute;
		top: -8px;
		left: 50%;
		transform: translateX(-50%);
		font-size: 10px;
		font-weight: 700;
		padding: 2px 8px;
		border-radius: 20px;
		background: linear-gradient(90deg, #3b82f6, #8b5cf6);
		color: #ffffff;
		white-space: nowrap;
	}

	.cp-btn {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		font-size: 13px;
		font-weight: 500;
		padding: 8px 14px;
		border-radius: 8px;
		cursor: pointer;
		border: none;
		transition: background 140ms ease-out, color 140ms ease-out, opacity 140ms ease-out;
		text-decoration: none;
		line-height: 1;
		white-space: nowrap;
	}

	.cp-btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.cp-btn--primary {
		background: #3b82f6;
		color: #ffffff;
	}
	.cp-btn--primary:hover:not(:disabled) { background: #2563eb; }

	.cp-btn--lg { padding: 12px 24px; font-size: 14px; }
</style>
