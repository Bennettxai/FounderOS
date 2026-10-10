<script lang="ts">
  import { goto } from '$app/navigation';
  import {
    CreditCard,
    Zap,
    Users,
    FolderOpen,
    HardDrive,
    CheckCircle2,
    ArrowUpRight,
    Crown,
    Rocket,
    Building2,
  } from 'lucide-svelte';
  import type { Workspace } from '$lib/api/workspaces';

  interface Props {
    workspace: Workspace;
  }

  let { workspace }: Props = $props();

  type PlanTier = 'free' | 'starter' | 'professional' | 'enterprise';

  const plans: Array<{
    id: PlanTier;
    name: string;
    price: string;
    description: string;
    features: string[];
    highlight?: boolean;
  }> = [
    {
      id: 'free',
      name: 'Free',
      price: '$0/mo',
      description: 'For personal projects and exploration',
      features: [
        'Up to 3 members',
        '2 projects',
        '1 GB storage',
        'Basic AI features',
        'Community support',
      ],
    },
    {
      id: 'starter',
      name: 'Starter',
      price: '$29/mo',
      description: 'For small teams getting started',
      features: [
        'Up to 10 members',
        '25 projects',
        '20 GB storage',
        'All AI models',
        'Email support',
      ],
      highlight: true,
    },
    {
      id: 'professional',
      name: 'Professional',
      price: '$79/mo',
      description: 'For growing teams with advanced needs',
      features: [
        'Up to 50 members',
        'Unlimited projects',
        '100 GB storage',
        'Priority AI access',
        'Priority support',
        'Custom roles',
        'Advanced analytics',
      ],
    },
    {
      id: 'enterprise',
      name: 'Enterprise',
      price: 'Custom',
      description: 'For large organizations needing full control',
      features: [
        'Unlimited members',
        'Unlimited projects',
        'Unlimited storage',
        'Dedicated AI compute',
        'SLA + dedicated support',
        'SSO / SAML',
        'Custom integrations',
        'Audit logs',
      ],
    },
  ];

  const planTierIndex: Record<PlanTier, number> = {
    free: 0,
    starter: 1,
    professional: 2,
    enterprise: 3,
  };

  let currentPlan = $derived(workspace.plan_type as PlanTier);
  let currentIndex = $derived(planTierIndex[currentPlan] ?? 0);

  function isCurrentPlan(planId: PlanTier): boolean {
    return planId === currentPlan;
  }

  function isUpgrade(planId: PlanTier): boolean {
    return planTierIndex[planId] > currentIndex;
  }

  function handleUpgrade(planId: PlanTier) {
    goto(`/register?plan=${planId}&workspace=${workspace.id}`);
  }

  function formatStorage(gb: number): string {
    return gb === -1 ? 'Unlimited' : `${gb} GB`;
  }

  function formatLimit(val: number): string {
    return val === -1 ? 'Unlimited' : String(val);
  }

  const planIconMap: Record<PlanTier, typeof CreditCard> = {
    free: CreditCard,
    starter: Zap,
    professional: Rocket,
    enterprise: Crown,
  };
</script>

<div class="wpb-container">
  <!-- Current Plan Banner -->
  <div class="wpb-current-plan">
    <div class="wpb-current-left">
      <div class="wpb-plan-icon">
        {#if currentPlan === 'free'}
          <CreditCard class="w-5 h-5" />
        {:else if currentPlan === 'starter'}
          <Zap class="w-5 h-5" />
        {:else if currentPlan === 'professional'}
          <Rocket class="w-5 h-5" />
        {:else}
          <Crown class="w-5 h-5" />
        {/if}
      </div>
      <div>
        <div class="wpb-current-name">
          {plans.find((p) => p.id === currentPlan)?.name ?? currentPlan} Plan
        </div>
        <div class="wpb-current-sub">
          {plans.find((p) => p.id === currentPlan)?.description ?? ''}
        </div>
      </div>
    </div>
    <div class="wpb-current-limits">
      <div class="wpb-limit-item">
        <Users class="w-4 h-4" />
        <span>{formatLimit(workspace.max_members)} members</span>
      </div>
      <div class="wpb-limit-item">
        <FolderOpen class="w-4 h-4" />
        <span>{formatLimit(workspace.max_projects)} projects</span>
      </div>
      <div class="wpb-limit-item">
        <HardDrive class="w-4 h-4" />
        <span>{formatStorage(workspace.max_storage_gb)} storage</span>
      </div>
    </div>
  </div>

  <!-- Section divider -->
  <div class="wpb-section-label">Available Plans</div>

  <!-- Plan Cards -->
  <div class="wpb-plans-grid">
    {#each plans as plan}
      <div
        class="wpb-plan-card"
        class:wpb-plan-card--current={isCurrentPlan(plan.id)}
        class:wpb-plan-card--highlight={plan.highlight && !isCurrentPlan(plan.id)}
      >
        {#if isCurrentPlan(plan.id)}
          <div class="wpb-current-badge">Current Plan</div>
        {:else if plan.highlight}
          <div class="wpb-popular-badge">Most Popular</div>
        {/if}

        <div class="wpb-card-header">
          <div class="wpb-card-icon">
            {#if plan.id === 'free'}
              <CreditCard class="w-5 h-5" />
            {:else if plan.id === 'starter'}
              <Zap class="w-5 h-5" />
            {:else if plan.id === 'professional'}
              <Rocket class="w-5 h-5" />
            {:else}
              <Building2 class="w-5 h-5" />
            {/if}
          </div>
          <div>
            <div class="wpb-card-name">{plan.name}</div>
            <div class="wpb-card-price">{plan.price}</div>
          </div>
        </div>

        <p class="wpb-card-desc">{plan.description}</p>

        <ul class="wpb-features">
          {#each plan.features as feature}
            <li class="wpb-feature-item">
              <CheckCircle2 class="w-4 h-4 wpb-check" />
              <span>{feature}</span>
            </li>
          {/each}
        </ul>

        <div class="wpb-card-action">
          {#if isCurrentPlan(plan.id)}
            <button class="wpb-btn wpb-btn--current" disabled>
              Current Plan
            </button>
          {:else if isUpgrade(plan.id)}
            <button
              class="wpb-btn wpb-btn--upgrade"
              onclick={() => handleUpgrade(plan.id)}
            >
              <ArrowUpRight class="w-4 h-4" />
              Upgrade to {plan.name}
            </button>
          {:else}
            <button
              class="wpb-btn wpb-btn--downgrade"
              onclick={() => handleUpgrade(plan.id)}
            >
              Switch to {plan.name}
            </button>
          {/if}
        </div>
      </div>
    {/each}
  </div>

  <!-- Enterprise CTA -->
  <div class="wpb-enterprise-cta">
    <div class="wpb-enterprise-left">
      <Crown class="w-6 h-6 wpb-crown-icon" />
      <div>
        <div class="wpb-enterprise-title">Need a custom plan?</div>
        <div class="wpb-enterprise-sub">
          Enterprise plans include SSO, audit logs, dedicated support, and custom integrations.
        </div>
      </div>
    </div>
    <button
      class="wpb-btn wpb-btn--enterprise"
      onclick={() => goto('/register?plan=enterprise')}
    >
      Contact Sales
      <ArrowUpRight class="w-4 h-4" />
    </button>
  </div>
</div>

<style>
  .wpb-container {
    padding: 2rem;
    display: flex;
    flex-direction: column;
    gap: 1.5rem;
  }

  /* Current Plan Banner */
  .wpb-current-plan {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1.5rem;
    padding: 1.25rem 1.5rem;
    background: var(--dbg2);
    border: 1px solid var(--dbd);
    border-radius: 0.75rem;
    flex-wrap: wrap;
  }

  .wpb-current-left {
    display: flex;
    align-items: center;
    gap: 1rem;
  }

  .wpb-plan-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 2.5rem;
    height: 2.5rem;
    background: var(--dbg3);
    border: 1px solid var(--dbd);
    border-radius: 0.5rem;
    color: var(--dt2);
    flex-shrink: 0;
  }

  .wpb-current-name {
    font-size: 1rem;
    font-weight: 600;
    color: var(--dt);
  }

  .wpb-current-sub {
    font-size: 0.8125rem;
    color: var(--dt3);
    margin-top: 0.125rem;
  }

  .wpb-current-limits {
    display: flex;
    gap: 1.25rem;
    flex-wrap: wrap;
  }

  .wpb-limit-item {
    display: flex;
    align-items: center;
    gap: 0.375rem;
    font-size: 0.8125rem;
    color: var(--dt2);
  }

  .wpb-limit-item svg {
    color: var(--dt3);
    flex-shrink: 0;
  }

  /* Section label */
  .wpb-section-label {
    font-size: 0.75rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: var(--dt3);
  }

  /* Plan Cards Grid */
  .wpb-plans-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
    gap: 1rem;
  }

  .wpb-plan-card {
    position: relative;
    display: flex;
    flex-direction: column;
    gap: 1rem;
    padding: 1.25rem;
    background: var(--dbg);
    border: 1px solid var(--dbd);
    border-radius: 0.75rem;
    transition: border-color 0.15s;
  }

  .wpb-plan-card:hover {
    border-color: var(--dbd2);
  }

  .wpb-plan-card--current {
    border-color: var(--dt);
    background: var(--dbg2);
  }

  .wpb-plan-card--highlight {
    border-color: var(--dbd2);
  }

  /* Badges */
  .wpb-current-badge,
  .wpb-popular-badge {
    position: absolute;
    top: -10px;
    left: 50%;
    transform: translateX(-50%);
    white-space: nowrap;
    font-size: 0.6875rem;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    padding: 0.25rem 0.625rem;
    border-radius: 9999px;
  }

  .wpb-current-badge {
    background: var(--dt);
    color: var(--dbg);
  }

  .wpb-popular-badge {
    background: var(--dbg3);
    color: var(--dt2);
    border: 1px solid var(--dbd);
  }

  /* Card Header */
  .wpb-card-header {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    margin-top: 0.5rem;
  }

  .wpb-card-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 2rem;
    height: 2rem;
    background: var(--dbg3);
    border-radius: 0.375rem;
    color: var(--dt2);
    flex-shrink: 0;
  }

  .wpb-card-name {
    font-size: 0.9375rem;
    font-weight: 600;
    color: var(--dt);
    line-height: 1.2;
  }

  .wpb-card-price {
    font-size: 0.8125rem;
    color: var(--dt3);
    margin-top: 0.125rem;
  }

  .wpb-card-desc {
    font-size: 0.8125rem;
    color: var(--dt3);
    margin: 0;
    line-height: 1.5;
  }

  /* Features List */
  .wpb-features {
    list-style: none;
    padding: 0;
    margin: 0;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    flex: 1;
  }

  .wpb-feature-item {
    display: flex;
    align-items: flex-start;
    gap: 0.5rem;
    font-size: 0.8125rem;
    color: var(--dt2);
    line-height: 1.4;
  }

  .wpb-check {
    flex-shrink: 0;
    margin-top: 1px;
    color: var(--dt3);
  }

  .wpb-plan-card--current .wpb-check {
    color: var(--dt);
  }

  /* Action Buttons */
  .wpb-card-action {
    margin-top: auto;
  }

  .wpb-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 0.375rem;
    width: 100%;
    padding: 0.5rem 1rem;
    font-size: 0.8125rem;
    font-weight: 500;
    border-radius: 0.375rem;
    cursor: pointer;
    transition: all 0.15s;
    border: 1px solid transparent;
  }

  .wpb-btn--current {
    background: var(--dbg3);
    color: var(--dt3);
    border-color: var(--dbd);
    cursor: default;
  }

  .wpb-btn--upgrade {
    background: var(--dt);
    color: var(--dbg);
    border-color: var(--dt);
  }

  .wpb-btn--upgrade:hover {
    opacity: 0.88;
  }

  .wpb-btn--downgrade {
    background: transparent;
    color: var(--dt2);
    border-color: var(--dbd);
  }

  .wpb-btn--downgrade:hover {
    background: var(--dbg2);
    color: var(--dt);
  }

  /* Enterprise CTA */
  .wpb-enterprise-cta {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1.5rem;
    padding: 1.25rem 1.5rem;
    background: var(--dbg2);
    border: 1px solid var(--dbd);
    border-radius: 0.75rem;
    flex-wrap: wrap;
  }

  .wpb-enterprise-left {
    display: flex;
    align-items: center;
    gap: 1rem;
  }

  .wpb-crown-icon {
    color: var(--dt3);
    flex-shrink: 0;
  }

  .wpb-enterprise-title {
    font-size: 0.9375rem;
    font-weight: 600;
    color: var(--dt);
  }

  .wpb-enterprise-sub {
    font-size: 0.8125rem;
    color: var(--dt3);
    margin-top: 0.25rem;
    max-width: 480px;
    line-height: 1.5;
  }

  .wpb-btn--enterprise {
    background: transparent;
    color: var(--dt);
    border-color: var(--dbd);
    white-space: nowrap;
    padding: 0.625rem 1.25rem;
  }

  .wpb-btn--enterprise:hover {
    background: var(--dbg3);
    border-color: var(--dbd2);
  }
</style>
