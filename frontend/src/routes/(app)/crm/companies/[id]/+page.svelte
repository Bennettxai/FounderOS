<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import {
		crm,
		lifecycleStageLabels,
		dealStatusLabels,
		dealPriorityLabels,
		activityTypeLabels,
		formatCurrency,
	} from '$lib/stores/crm';
	import * as crmApi from '$lib/api/crm';
	import type { Company, Deal, CRMActivity, ActivityType } from '$lib/api/crm';
	import {
		ArrowLeft,
		Building2,
		Globe,
		Mail,
		Phone,
		MapPin,
		TrendingUp,
		Users,
		Briefcase,
		Clock,
		CheckCircle2,
		Circle,
		CalendarDays,
		ExternalLink,
		Linkedin,
		Twitter,
	} from 'lucide-svelte';

	// ── Route param ─────────────────────────────────────────────────────────────
	const companyId = $derived($page.params.id);

	// ── Local state ──────────────────────────────────────────────────────────────
	let company = $state<Company | null>(null);
	let deals = $state<Deal[]>([]);
	let activities = $state<CRMActivity[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);

	// Track what we've loaded to prevent loop re-triggers
	let loadedId: string | null = null;

	// ── Active tab ───────────────────────────────────────────────────────────────
	type Tab = 'overview' | 'deals' | 'activities';
	let activeTab = $state<Tab>('overview');

	// ── Load data ─────────────────────────────────────────────────────────────────
	$effect(() => {
		const id = companyId;
		if (!id || id === loadedId) return;
		loadedId = id;
		loadAll(id);
	});

	async function loadAll(id: string) {
		loading = true;
		error = null;

		// 1. Company record — try API, fall back to store cache
		try {
			const result = await Promise.race([
				crmApi.getCompany(id),
				new Promise<never>((_, reject) =>
					setTimeout(() => reject(new Error('Company API timeout')), 4000),
				),
			]);
			company = result;
		} catch {
			// Try the already-loaded companies list in the store
			const cached = $crm.companies.find((c) => c.id === id) ?? null;
			company = cached;
			if (!cached) {
				error = 'Company not found.';
				loading = false;
				return;
			}
		}

		// 2. Deals for this company — best-effort from the store's deal list
		// (no dedicated endpoint, filter by company_id)
		const storeDeals = $crm.deals.filter(
			(d) => d.company_id === id || d.company_name === company?.name,
		);
		deals = storeDeals;

		// Attempt API load of all deals if store is empty
		if (storeDeals.length === 0) {
			try {
				const resp = await Promise.race([
					crmApi.getDeals(),
					new Promise<never>((_, reject) =>
						setTimeout(() => reject(new Error('Deals timeout')), 3000),
					),
				]);
				deals = resp.deals.filter(
					(d) => d.company_id === id || d.company_name === company?.name,
				);
			} catch {
				// leave deals empty — not fatal
			}
		}

		// 3. Activities — best-effort, only from persisted CRM data
		try {
			const resp = await Promise.race([
				crmApi.getActivities(),
				new Promise<never>((_, reject) =>
					setTimeout(() => reject(new Error('Activities timeout')), 3000),
				),
			]);
			const filtered = resp.activities.filter((a) => a.company_id === id);
			activities = filtered;
		} catch {
			activities = [];
		}

		loading = false;
	}

	onMount(() => {
		// Ensure companies are in the store for the cache fallback
		if ($crm.companies.length === 0) {
			crm.loadCompanies();
		}
		return () => {
			crm.clearCurrentCompany();
		};
	});

	// ── Helpers ──────────────────────────────────────────────────────────────────
	function getInitial(name: string): string {
		return name.trim().charAt(0).toUpperCase();
	}

	function getLifecyclePillClass(stage: string | undefined): string {
		switch (stage) {
			case 'lead':        return 'cr-cd-pill cr-cd-pill--info';
			case 'opportunity': return 'cr-cd-pill cr-cd-pill--warning';
			case 'customer':    return 'cr-cd-pill cr-cd-pill--success';
			case 'partner':     return 'cr-cd-pill cr-cd-pill--accent';
			case 'churned':     return 'cr-cd-pill cr-cd-pill--error';
			default:            return 'cr-cd-pill cr-cd-pill--neutral';
		}
	}

	function getDealStatusPillClass(status: string | undefined): string {
		switch (status) {
			case 'won':  return 'cr-cd-pill cr-cd-pill--success';
			case 'lost': return 'cr-cd-pill cr-cd-pill--error';
			default:     return 'cr-cd-pill cr-cd-pill--info';
		}
	}

	function getPriorityPillClass(priority: string | undefined): string {
		switch (priority) {
			case 'urgent': return 'cr-cd-pill cr-cd-pill--error';
			case 'high':   return 'cr-cd-pill cr-cd-pill--warning';
			case 'medium': return 'cr-cd-pill cr-cd-pill--info';
			default:       return 'cr-cd-pill cr-cd-pill--neutral';
		}
	}

	function getActivityIcon(type: ActivityType) {
		switch (type) {
			case 'call':    return Phone;
			case 'email':   return Mail;
			case 'meeting': return Users;
			case 'demo':    return TrendingUp;
			case 'task':    return CheckCircle2;
			default:        return Clock;
		}
	}

	function formatDate(dateStr: string | undefined): string {
		if (!dateStr) return '-';
		return new Date(dateStr).toLocaleDateString('en-US', {
			month: 'short',
			day: 'numeric',
			year: 'numeric',
		});
	}

	function formatActivityDate(dateStr: string): string {
		const d = new Date(dateStr);
		const now = new Date();
		const diff = (now.getTime() - d.getTime()) / 1000;
		if (diff < 60)   return 'just now';
		if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
		if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
		const days = Math.floor(diff / 86400);
		if (days < 7)    return `${days}d ago`;
		if (days < 0)    return `in ${Math.abs(days)}d`;
		return formatDate(dateStr);
	}

	function formatActivityDateFuture(dateStr: string): string {
		const d = new Date(dateStr);
		const now = new Date();
		const diff = (d.getTime() - now.getTime()) / 1000;
		if (diff > 0) {
			const days = Math.floor(diff / 86400);
			if (days === 0) return 'today';
			if (days === 1) return 'tomorrow';
			return `in ${days}d`;
		}
		return formatActivityDate(dateStr);
	}

	function buildAddress(c: Company): string {
		const parts = [
			c.address_line1,
			c.address_line2,
			[c.city, c.state].filter(Boolean).join(', '),
			c.postal_code,
			c.country,
		].filter(Boolean);
		return parts.join(', ');
	}

	function stripProtocol(url: string): string {
		return url.replace(/^https?:\/\//, '').replace(/\/$/, '');
	}

	// Counts for tab badges
	const dealsCount     = $derived(deals.length);
	const activitiesCount = $derived(activities.length);
</script>

<svelte:head>
	<title>{company ? `${company.name} — Companies` : 'Company'} - BusinessOS</title>
</svelte:head>

<!-- ═══════════════════════════════════════════════════════════════════════════
     PAGE
     ═══════════════════════════════════════════════════════════════════════════ -->
<div class="cr-cd-page">

	<!-- ── Header ─────────────────────────────────────────────────────────────── -->
	<header class="cr-cd-header">
		<div class="cr-cd-header-left">
			<a
				href="/crm/companies"
				class="cr-cd-back-btn"
				aria-label="Back to Companies"
			>
				<ArrowLeft size={15} aria-hidden="true" />
				<span>Companies</span>
			</a>
		</div>
	</header>

	<!-- ── Loading state ──────────────────────────────────────────────────────── -->
	{#if loading}
		<div class="cr-cd-loading" aria-label="Loading company">
			<div class="cr-cd-spinner" aria-hidden="true"></div>
			<p class="cr-cd-loading-text">Loading company...</p>
		</div>

	<!-- ── Error state ────────────────────────────────────────────────────────── -->
	{:else if error || !company}
		<div class="cr-cd-error-wrap" role="alert">
			<Building2 size={36} class="cr-cd-error-icon" aria-hidden="true" />
			<p class="cr-cd-error-title">{error ?? 'Company not found'}</p>
			<button
				class="btn-pill btn-pill-secondary btn-pill-sm"
				onclick={() => goto('/crm/companies')}
			>
				Back to Companies
			</button>
		</div>

	<!-- ── Main content ───────────────────────────────────────────────────────── -->
	{:else}
		<!-- Hero card -->
		<div class="cr-cd-hero">
			<div class="cr-cd-hero-avatar" aria-hidden="true">
				{#if company.logo_url}
					<img src={company.logo_url} alt="{company.name} logo" class="cr-cd-hero-logo" />
				{:else}
					{getInitial(company.name)}
				{/if}
			</div>
			<div class="cr-cd-hero-info">
				<h1 class="cr-cd-hero-name">{company.name}</h1>
				{#if company.legal_name && company.legal_name !== company.name}
					<p class="cr-cd-hero-legal">{company.legal_name}</p>
				{/if}
				<div class="cr-cd-hero-meta">
					{#if company.industry}
						<span class="cr-cd-hero-industry">{company.industry}</span>
					{/if}
					{#if company.lifecycle_stage}
						<span class={getLifecyclePillClass(company.lifecycle_stage)}>
							{lifecycleStageLabels[company.lifecycle_stage] ?? company.lifecycle_stage}
						</span>
					{/if}
					{#if company.company_size}
						<span class="cr-cd-hero-size">
							<Users size={12} aria-hidden="true" />
							{company.company_size}
						</span>
					{/if}
				</div>
			</div>

			<!-- Quick contact links -->
			<div class="cr-cd-hero-actions">
				{#if company.website}
					<a
						href={company.website}
						target="_blank"
						rel="noopener noreferrer"
						class="cr-cd-contact-btn"
						aria-label="Visit website"
					>
						<Globe size={14} aria-hidden="true" />
						<span>Website</span>
						<ExternalLink size={11} class="cr-cd-contact-btn-ext" aria-hidden="true" />
					</a>
				{/if}
				{#if company.email}
					<a
						href="mailto:{company.email}"
						class="cr-cd-contact-btn"
						aria-label="Email {company.email}"
					>
						<Mail size={14} aria-hidden="true" />
						<span class="cr-cd-contact-btn-label">{company.email}</span>
					</a>
				{/if}
				{#if company.phone}
					<a
						href="tel:{company.phone}"
						class="cr-cd-contact-btn"
						aria-label="Call {company.phone}"
					>
						<Phone size={14} aria-hidden="true" />
						<span class="cr-cd-contact-btn-label">{company.phone}</span>
					</a>
				{/if}
			</div>
		</div>

		<!-- Tab bar -->
		<nav class="cr-cd-tabs" aria-label="Company sections">
			<button
				class="cr-cd-tab {activeTab === 'overview' ? 'cr-cd-tab--active' : ''}"
				onclick={() => (activeTab = 'overview')}
				aria-selected={activeTab === 'overview'}
				role="tab"
			>
				Overview
			</button>
			<button
				class="cr-cd-tab {activeTab === 'deals' ? 'cr-cd-tab--active' : ''}"
				onclick={() => (activeTab = 'deals')}
				aria-selected={activeTab === 'deals'}
				role="tab"
			>
				Deals
				{#if dealsCount > 0}
					<span class="cr-cd-tab-badge">{dealsCount}</span>
				{/if}
			</button>
			<button
				class="cr-cd-tab {activeTab === 'activities' ? 'cr-cd-tab--active' : ''}"
				onclick={() => (activeTab = 'activities')}
				aria-selected={activeTab === 'activities'}
				role="tab"
			>
				Activities
				{#if activitiesCount > 0}
					<span class="cr-cd-tab-badge">{activitiesCount}</span>
				{/if}
			</button>
		</nav>

		<!-- Tab panels -->
		<div class="cr-cd-body">

			<!-- ── OVERVIEW TAB ─────────────────────────────────────────────────── -->
			{#if activeTab === 'overview'}
				<div class="cr-cd-overview">

					<!-- Details card -->
					<section class="cr-cd-card" aria-labelledby="details-heading">
						<h2 class="cr-cd-card-title" id="details-heading">Company Details</h2>

						<dl class="cr-cd-detail-list">
							{#if buildAddress(company)}
								<div class="cr-cd-detail-row">
									<dt class="cr-cd-detail-label">
										<MapPin size={13} aria-hidden="true" />
										Address
									</dt>
									<dd class="cr-cd-detail-value">{buildAddress(company)}</dd>
								</div>
							{/if}

							{#if company.website}
								<div class="cr-cd-detail-row">
									<dt class="cr-cd-detail-label">
										<Globe size={13} aria-hidden="true" />
										Website
									</dt>
									<dd class="cr-cd-detail-value">
										<a
											href={company.website}
											target="_blank"
											rel="noopener noreferrer"
											class="cr-cd-link"
										>
											{stripProtocol(company.website)}
										</a>
									</dd>
								</div>
							{/if}

							{#if company.email}
								<div class="cr-cd-detail-row">
									<dt class="cr-cd-detail-label">
										<Mail size={13} aria-hidden="true" />
										Email
									</dt>
									<dd class="cr-cd-detail-value">
										<a href="mailto:{company.email}" class="cr-cd-link">{company.email}</a>
									</dd>
								</div>
							{/if}

							{#if company.phone}
								<div class="cr-cd-detail-row">
									<dt class="cr-cd-detail-label">
										<Phone size={13} aria-hidden="true" />
										Phone
									</dt>
									<dd class="cr-cd-detail-value">
										<a href="tel:{company.phone}" class="cr-cd-link">{company.phone}</a>
									</dd>
								</div>
							{/if}

							{#if company.annual_revenue}
								<div class="cr-cd-detail-row">
									<dt class="cr-cd-detail-label">
										<TrendingUp size={13} aria-hidden="true" />
										Annual Revenue
									</dt>
									<dd class="cr-cd-detail-value cr-cd-detail-value--number">
										{formatCurrency(company.annual_revenue, company.currency ?? 'USD')}
									</dd>
								</div>
							{/if}

							{#if company.lead_source}
								<div class="cr-cd-detail-row">
									<dt class="cr-cd-detail-label">
										<Briefcase size={13} aria-hidden="true" />
										Lead Source
									</dt>
									<dd class="cr-cd-detail-value">{company.lead_source}</dd>
								</div>
							{/if}

							{#if company.linkedin_url}
								<div class="cr-cd-detail-row">
									<dt class="cr-cd-detail-label">
										<Linkedin size={13} aria-hidden="true" />
										LinkedIn
									</dt>
									<dd class="cr-cd-detail-value">
										<a
											href={company.linkedin_url}
											target="_blank"
											rel="noopener noreferrer"
											class="cr-cd-link"
										>
											{stripProtocol(company.linkedin_url)}
										</a>
									</dd>
								</div>
							{/if}

							{#if company.twitter_handle}
								<div class="cr-cd-detail-row">
									<dt class="cr-cd-detail-label">
										<Twitter size={13} aria-hidden="true" />
										Twitter
									</dt>
									<dd class="cr-cd-detail-value">
										<a
											href="https://twitter.com/{company.twitter_handle.replace(/^@/, '')}"
											target="_blank"
											rel="noopener noreferrer"
											class="cr-cd-link"
										>
											@{company.twitter_handle.replace(/^@/, '')}
										</a>
									</dd>
								</div>
							{/if}

							<div class="cr-cd-detail-row">
								<dt class="cr-cd-detail-label">
									<CalendarDays size={13} aria-hidden="true" />
									Added
								</dt>
								<dd class="cr-cd-detail-value">{formatDate(company.created_at)}</dd>
							</div>
						</dl>
					</section>

					<!-- Scores card (only when available) -->
					{#if company.health_score !== undefined || company.engagement_score !== undefined}
						<section class="cr-cd-card" aria-labelledby="scores-heading">
							<h2 class="cr-cd-card-title" id="scores-heading">Account Health</h2>
							<div class="cr-cd-scores-grid">
								{#if company.health_score !== undefined}
									<div class="cr-cd-score-item">
										<span class="cr-cd-score-label">Health Score</span>
										<div class="cr-cd-score-bar-wrap" aria-label="Health score {company.health_score}%">
											<div
												class="cr-cd-score-bar"
												style="width: {Math.min(company.health_score, 100)}%"
											></div>
										</div>
										<span class="cr-cd-score-value">{company.health_score}</span>
									</div>
								{/if}
								{#if company.engagement_score !== undefined}
									<div class="cr-cd-score-item">
										<span class="cr-cd-score-label">Engagement Score</span>
										<div class="cr-cd-score-bar-wrap" aria-label="Engagement score {company.engagement_score}%">
											<div
												class="cr-cd-score-bar cr-cd-score-bar--engagement"
												style="width: {Math.min(company.engagement_score, 100)}%"
											></div>
										</div>
										<span class="cr-cd-score-value">{company.engagement_score}</span>
									</div>
								{/if}
							</div>
						</section>
					{/if}

					<!-- Recent deals snapshot -->
					{#if deals.length > 0}
						<section class="cr-cd-card" aria-labelledby="recent-deals-heading">
							<div class="cr-cd-card-header">
								<h2 class="cr-cd-card-title" id="recent-deals-heading">Recent Deals</h2>
								<button
									class="btn-pill btn-pill-ghost btn-pill-sm"
									onclick={() => (activeTab = 'deals')}
								>
									View all
								</button>
							</div>
							<ul class="cr-cd-deal-list" role="list">
								{#each deals.slice(0, 3) as deal (deal.id)}
									<li class="cr-cd-deal-row">
										<a
											href="/crm/deals/{deal.id}"
											class="cr-cd-deal-link"
											aria-label="Open deal {deal.name}"
										>
											<span class="cr-cd-deal-name">{deal.name}</span>
											<span class="cr-cd-deal-meta">
												{#if deal.amount}
													<span class="cr-cd-deal-amount">
														{formatCurrency(deal.amount, deal.currency ?? 'USD')}
													</span>
												{/if}
												<span class={getDealStatusPillClass(deal.status)}>
													{dealStatusLabels[deal.status ?? 'open'] ?? 'Open'}
												</span>
											</span>
										</a>
									</li>
								{/each}
							</ul>
						</section>
					{/if}
				</div>

			<!-- ── DEALS TAB ────────────────────────────────────────────────────── -->
			{:else if activeTab === 'deals'}
				<div class="cr-cd-tab-panel">
					{#if deals.length === 0}
						<div class="cr-cd-empty">
							<div class="cr-cd-empty-icon" aria-hidden="true">
								<Briefcase size={32} />
							</div>
							<p class="cr-cd-empty-title">No deals yet</p>
							<p class="cr-cd-empty-sub">Deals linked to this company will appear here.</p>
							<a href="/crm" class="btn-pill btn-pill-secondary btn-pill-sm cr-cd-empty-action">
								Go to Pipeline
							</a>
						</div>
					{:else}
						<ul class="cr-cd-deal-cards" role="list">
							{#each deals as deal (deal.id)}
								<li>
									<a
										href="/crm/deals/{deal.id}"
										class="cr-cd-deal-card"
										aria-label="Open deal {deal.name}"
									>
										<div class="cr-cd-deal-card-top">
											<span class="cr-cd-deal-card-name">{deal.name}</span>
											{#if deal.priority}
												<span class={getPriorityPillClass(deal.priority)}>
													{dealPriorityLabels[deal.priority] ?? deal.priority}
												</span>
											{/if}
										</div>
										<div class="cr-cd-deal-card-mid">
											{#if deal.amount}
												<span class="cr-cd-deal-card-amount">
													{formatCurrency(deal.amount, deal.currency ?? 'USD')}
												</span>
											{/if}
											{#if deal.stage_name}
												<span class="cr-cd-detail-value cr-cd-deal-card-stage">
													{deal.stage_name}
												</span>
											{/if}
											<span class={getDealStatusPillClass(deal.status)}>
												{dealStatusLabels[deal.status ?? 'open'] ?? 'Open'}
											</span>
										</div>
										{#if deal.expected_close_date}
											<div class="cr-cd-deal-card-foot">
												<CalendarDays size={12} aria-hidden="true" />
												<span>Close {formatDate(deal.expected_close_date)}</span>
											</div>
										{/if}
									</a>
								</li>
							{/each}
						</ul>
					{/if}
				</div>

			<!-- ── ACTIVITIES TAB ──────────────────────────────────────────────── -->
			{:else if activeTab === 'activities'}
				<div class="cr-cd-tab-panel">
					{#if activities.length === 0}
						<div class="cr-cd-empty">
							<div class="cr-cd-empty-icon" aria-hidden="true">
								<Clock size={32} />
							</div>
							<p class="cr-cd-empty-title">No activities recorded</p>
							<p class="cr-cd-empty-sub">Log calls, emails, and meetings to track engagement.</p>
						</div>
					{:else}
						<ol class="cr-cd-timeline" aria-label="Activity timeline">
							{#each activities as activity (activity.id)}
								{@const Icon = getActivityIcon(activity.activity_type)}
								{@const isFuture = new Date(activity.activity_date) > new Date()}
								<li class="cr-cd-timeline-item {isFuture ? 'cr-cd-timeline-item--future' : ''}">
									<div class="cr-cd-timeline-icon-wrap" aria-hidden="true">
										<div class="cr-cd-timeline-icon">
											<Icon size={13} />
										</div>
										<div class="cr-cd-timeline-line"></div>
									</div>
									<div class="cr-cd-timeline-content">
										<div class="cr-cd-timeline-top">
											<span class="cr-cd-timeline-subject">{activity.subject}</span>
											<div class="cr-cd-timeline-badges">
												<span class="cr-cd-activity-type-pill">
													{activityTypeLabels[activity.activity_type] ?? activity.activity_type}
												</span>
												{#if activity.is_completed}
													<span class="cr-cd-activity-done" title="Completed">
														<CheckCircle2 size={13} aria-hidden="true" />
													</span>
												{:else}
													<span class="cr-cd-activity-pending" title="Upcoming">
														<Circle size={13} aria-hidden="true" />
													</span>
												{/if}
											</div>
										</div>
										{#if activity.description}
											<p class="cr-cd-timeline-desc">{activity.description}</p>
										{/if}
										{#if activity.outcome}
											<p class="cr-cd-timeline-outcome">{activity.outcome}</p>
										{/if}
										<div class="cr-cd-timeline-foot">
											<time
												datetime={activity.activity_date}
												class="cr-cd-timeline-time"
												title={formatDate(activity.activity_date)}
											>
												{isFuture
													? formatActivityDateFuture(activity.activity_date)
													: formatActivityDate(activity.activity_date)}
											</time>
											{#if activity.duration_minutes}
												<span class="cr-cd-timeline-duration">
													{activity.duration_minutes}m
												</span>
											{/if}
										</div>
									</div>
								</li>
							{/each}
						</ol>
					{/if}
				</div>
			{/if}
		</div>
	{/if}
</div>

<style>
	/* ── Page layout ─────────────────────────────────────────────────────────── */
	.cr-cd-page {
		display: flex;
		flex-direction: column;
		min-height: 100%;
		background: var(--dbg);
		font-family: var(--bos-font-family);
	}

	/* ── Header / back button ────────────────────────────────────────────────── */
	.cr-cd-header {
		display: flex;
		align-items: center;
		padding: 14px 24px;
		border-bottom: 1px solid var(--dbd2);
		background: var(--dbg);
		flex-shrink: 0;
	}

	.cr-cd-back-btn {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		font-size: 13px;
		font-weight: 500;
		color: var(--dt3);
		text-decoration: none;
		padding: 5px 10px 5px 6px;
		border-radius: 7px;
		transition: background 120ms ease, color 120ms ease;
	}

	.cr-cd-back-btn:hover {
		background: var(--dbg3);
		color: var(--dt);
	}

	.cr-cd-back-btn:focus-visible {
		outline: 2px solid var(--bos-status-info);
		outline-offset: 2px;
	}

	/* ── Loading ─────────────────────────────────────────────────────────────── */
	.cr-cd-loading {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 12px;
		flex: 1;
		padding: 64px 24px;
	}

	.cr-cd-spinner {
		width: 26px;
		height: 26px;
		border: 2px solid var(--dbd);
		border-top-color: var(--dt2);
		border-radius: 50%;
		animation: cr-cd-spin 600ms linear infinite;
	}

	.cr-cd-loading-text {
		font-size: 13px;
		color: var(--dt3);
		margin: 0;
	}

	/* ── Error state ─────────────────────────────────────────────────────────── */
	.cr-cd-error-wrap {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 12px;
		flex: 1;
		padding: 72px 24px;
		text-align: center;
	}

	:global(.cr-cd-error-icon) {
		color: var(--dt4);
	}

	.cr-cd-error-title {
		font-size: 15px;
		font-weight: 600;
		color: var(--dt);
		margin: 0;
	}

	/* ── Hero card ───────────────────────────────────────────────────────────── */
	.cr-cd-hero {
		display: flex;
		align-items: flex-start;
		gap: 16px;
		padding: 20px 24px;
		border-bottom: 1px solid var(--dbd2);
		background: var(--dbg);
		flex-wrap: wrap;
	}

	.cr-cd-hero-avatar {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 52px;
		height: 52px;
		border-radius: 12px;
		background: var(--dbg3);
		color: var(--dt2);
		font-size: 22px;
		font-weight: 700;
		flex-shrink: 0;
		letter-spacing: -0.01em;
		overflow: hidden;
	}

	.cr-cd-hero-logo {
		width: 100%;
		height: 100%;
		object-fit: contain;
	}

	.cr-cd-hero-info {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: 6px;
	}

	.cr-cd-hero-name {
		font-size: 20px;
		font-weight: 700;
		color: var(--dt);
		margin: 0;
		line-height: 1.25;
	}

	.cr-cd-hero-legal {
		font-size: 12px;
		color: var(--dt4);
		margin: 0;
	}

	.cr-cd-hero-meta {
		display: flex;
		align-items: center;
		gap: 8px;
		flex-wrap: wrap;
	}

	.cr-cd-hero-industry {
		font-size: 12px;
		color: var(--dt3);
	}

	.cr-cd-hero-size {
		display: inline-flex;
		align-items: center;
		gap: 4px;
		font-size: 12px;
		color: var(--dt3);
	}

	/* ── Quick contact links ─────────────────────────────────────────────────── */
	.cr-cd-hero-actions {
		display: flex;
		align-items: center;
		gap: 8px;
		flex-wrap: wrap;
		flex-shrink: 0;
	}

	.cr-cd-contact-btn {
		display: inline-flex;
		align-items: center;
		gap: 5px;
		padding: 6px 12px;
		background: var(--dbg2);
		border: 1px solid var(--dbd2);
		border-radius: 8px;
		font-size: 12px;
		font-weight: 500;
		color: var(--dt2);
		text-decoration: none;
		transition: background 120ms ease, border-color 120ms ease, color 120ms ease;
		white-space: nowrap;
		max-width: 200px;
	}

	.cr-cd-contact-btn:hover {
		background: var(--dbg3);
		border-color: var(--dbd);
		color: var(--dt);
	}

	.cr-cd-contact-btn:focus-visible {
		outline: 2px solid var(--bos-status-info);
		outline-offset: 2px;
	}

	.cr-cd-contact-btn-label {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	:global(.cr-cd-contact-btn-ext) {
		flex-shrink: 0;
		color: var(--dt4);
	}

	/* ── Tab bar ─────────────────────────────────────────────────────────────── */
	.cr-cd-tabs {
		display: flex;
		align-items: center;
		gap: 2px;
		padding: 0 20px;
		border-bottom: 1px solid var(--dbd2);
		background: var(--dbg);
		flex-shrink: 0;
	}

	.cr-cd-tab {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		padding: 12px 14px;
		font-size: 13px;
		font-weight: 500;
		color: var(--dt3);
		background: transparent;
		border: none;
		border-bottom: 2px solid transparent;
		cursor: pointer;
		transition: color 120ms ease, border-color 120ms ease;
		font-family: var(--bos-font-family);
		white-space: nowrap;
		margin-bottom: -1px;
	}

	.cr-cd-tab:hover {
		color: var(--dt);
	}

	.cr-cd-tab:focus-visible {
		outline: 2px solid var(--bos-status-info);
		outline-offset: -2px;
		border-radius: 4px 4px 0 0;
	}

	.cr-cd-tab--active {
		color: var(--dt);
		border-bottom-color: var(--dt);
	}

	.cr-cd-tab-badge {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		min-width: 18px;
		height: 18px;
		padding: 0 5px;
		border-radius: 999px;
		background: var(--dbg3);
		color: var(--dt3);
		font-size: 10px;
		font-weight: 600;
		font-variant-numeric: tabular-nums;
	}

	/* ── Body / tab panels ───────────────────────────────────────────────────── */
	.cr-cd-body {
		flex: 1;
		overflow-y: auto;
		padding: 20px 24px;
	}

	.cr-cd-overview {
		display: flex;
		flex-direction: column;
		gap: 16px;
		max-width: 680px;
	}

	.cr-cd-tab-panel {
		max-width: 680px;
	}

	/* ── Card ────────────────────────────────────────────────────────────────── */
	.cr-cd-card {
		background: var(--dbg);
		border: 1px solid var(--dbd2);
		border-radius: 10px;
		padding: 16px 18px;
	}

	.cr-cd-card-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-bottom: 14px;
	}

	.cr-cd-card-title {
		font-size: 13px;
		font-weight: 600;
		color: var(--dt);
		margin: 0 0 14px;
		letter-spacing: 0.01em;
	}

	.cr-cd-card-header .cr-cd-card-title {
		margin: 0;
	}

	/* ── Detail list ─────────────────────────────────────────────────────────── */
	.cr-cd-detail-list {
		display: flex;
		flex-direction: column;
		gap: 0;
		margin: 0;
		padding: 0;
	}

	.cr-cd-detail-row {
		display: flex;
		align-items: baseline;
		gap: 12px;
		padding: 9px 0;
		border-bottom: 1px solid var(--dbd2);
	}

	.cr-cd-detail-row:last-child {
		border-bottom: none;
		padding-bottom: 0;
	}

	.cr-cd-detail-row:first-child {
		padding-top: 0;
	}

	.cr-cd-detail-label {
		display: inline-flex;
		align-items: center;
		gap: 5px;
		font-size: 12px;
		color: var(--dt3);
		min-width: 130px;
		flex-shrink: 0;
	}

	.cr-cd-detail-value {
		font-size: 13px;
		color: var(--dt);
		min-width: 0;
		word-break: break-word;
	}

	.cr-cd-detail-value--number {
		font-family: var(--bos-font-number-family);
		font-variant-numeric: tabular-nums;
	}

	.cr-cd-link {
		color: var(--bos-status-info-text);
		text-decoration: none;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		display: block;
	}

	.cr-cd-link:hover {
		text-decoration: underline;
	}

	/* ── Score bars ──────────────────────────────────────────────────────────── */
	.cr-cd-scores-grid {
		display: flex;
		flex-direction: column;
		gap: 14px;
	}

	.cr-cd-score-item {
		display: flex;
		align-items: center;
		gap: 10px;
	}

	.cr-cd-score-label {
		font-size: 12px;
		color: var(--dt3);
		min-width: 130px;
		flex-shrink: 0;
	}

	.cr-cd-score-bar-wrap {
		flex: 1;
		height: 6px;
		background: var(--dbg3);
		border-radius: 999px;
		overflow: hidden;
	}

	.cr-cd-score-bar {
		height: 100%;
		background: var(--bos-status-success);
		border-radius: 999px;
		transition: width 300ms ease;
	}

	.cr-cd-score-bar--engagement {
		background: var(--bos-status-info);
	}

	.cr-cd-score-value {
		font-size: 12px;
		font-weight: 600;
		color: var(--dt2);
		font-family: var(--bos-font-number-family);
		min-width: 28px;
		text-align: right;
	}

	/* ── Deal list (compact, in Overview card) ───────────────────────────────── */
	.cr-cd-deal-list {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 0;
	}

	.cr-cd-deal-row {
		border-bottom: 1px solid var(--dbd2);
	}

	.cr-cd-deal-row:last-child {
		border-bottom: none;
	}

	.cr-cd-deal-link {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 12px;
		padding: 10px 0;
		text-decoration: none;
		color: var(--dt);
		transition: color 120ms ease;
	}

	.cr-cd-deal-link:hover .cr-cd-deal-name {
		color: var(--bos-status-info-text);
	}

	.cr-cd-deal-name {
		font-size: 13px;
		font-weight: 500;
		color: var(--dt);
		min-width: 0;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		transition: color 120ms ease;
	}

	.cr-cd-deal-meta {
		display: flex;
		align-items: center;
		gap: 8px;
		flex-shrink: 0;
	}

	.cr-cd-deal-amount {
		font-size: 12px;
		color: var(--dt2);
		font-family: var(--bos-font-number-family);
		font-variant-numeric: tabular-nums;
	}

	/* ── Deal cards (Deals tab) ──────────────────────────────────────────────── */
	.cr-cd-deal-cards {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 10px;
	}

	.cr-cd-deal-card {
		display: flex;
		flex-direction: column;
		gap: 8px;
		padding: 14px 16px;
		background: var(--dbg);
		border: 1px solid var(--dbd2);
		border-radius: 10px;
		text-decoration: none;
		transition: box-shadow 150ms ease, border-color 150ms ease;
	}

	.cr-cd-deal-card:hover {
		box-shadow: var(--bos-shadow-1);
		border-color: var(--dbd);
	}

	.cr-cd-deal-card:focus-visible {
		outline: 2px solid var(--bos-status-info);
		outline-offset: 2px;
	}

	.cr-cd-deal-card-top {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: 10px;
	}

	.cr-cd-deal-card-name {
		font-size: 13px;
		font-weight: 600;
		color: var(--dt);
		min-width: 0;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.cr-cd-deal-card-mid {
		display: flex;
		align-items: center;
		gap: 8px;
		flex-wrap: wrap;
	}

	.cr-cd-deal-card-amount {
		font-size: 13px;
		font-weight: 600;
		color: var(--dt);
		font-family: var(--bos-font-number-family);
		font-variant-numeric: tabular-nums;
	}

	.cr-cd-deal-card-stage {
		font-size: 12px;
		color: var(--dt3);
	}

	.cr-cd-deal-card-foot {
		display: flex;
		align-items: center;
		gap: 5px;
		font-size: 12px;
		color: var(--dt3);
	}

	/* ── Empty state ─────────────────────────────────────────────────────────── */
	.cr-cd-empty {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 10px;
		padding: 56px 24px;
		text-align: center;
	}

	.cr-cd-empty-icon {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 56px;
		height: 56px;
		border-radius: 14px;
		background: var(--dbg3);
		color: var(--dt3);
		margin-bottom: 4px;
	}

	.cr-cd-empty-title {
		font-size: 14px;
		font-weight: 600;
		color: var(--dt);
		margin: 0;
	}

	.cr-cd-empty-sub {
		font-size: 12px;
		color: var(--dt3);
		margin: 0;
	}

	.cr-cd-empty-action {
		margin-top: 6px;
	}

	/* ── Lifecycle / status pills ────────────────────────────────────────────── */
	.cr-cd-pill {
		display: inline-flex;
		align-items: center;
		padding: 2px 8px;
		border-radius: 999px;
		font-size: 11px;
		font-weight: 500;
		line-height: 1.6;
		white-space: nowrap;
	}

	.cr-cd-pill--info {
		background: var(--bos-status-info-bg);
		color: var(--bos-status-info-text);
	}

	.cr-cd-pill--warning {
		background: var(--bos-status-warning-bg);
		color: var(--bos-status-warning-text);
	}

	.cr-cd-pill--success {
		background: var(--bos-status-success-bg);
		color: var(--bos-status-success-text);
	}

	.cr-cd-pill--accent {
		background: var(--bos-status-accent-bg);
		color: var(--bos-status-accent-text);
	}

	.cr-cd-pill--error {
		background: var(--bos-status-error-bg);
		color: var(--bos-status-error-text);
	}

	.cr-cd-pill--neutral {
		background: var(--bos-status-neutral-bg);
		color: var(--bos-status-neutral-text);
	}

	/* ── Activity type pill ──────────────────────────────────────────────────── */
	.cr-cd-activity-type-pill {
		display: inline-flex;
		align-items: center;
		padding: 1px 7px;
		border-radius: 999px;
		font-size: 10px;
		font-weight: 500;
		background: var(--dbg3);
		color: var(--dt3);
		white-space: nowrap;
	}

	/* ── Timeline ────────────────────────────────────────────────────────────── */
	.cr-cd-timeline {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
	}

	.cr-cd-timeline-item {
		display: flex;
		gap: 14px;
		position: relative;
	}

	.cr-cd-timeline-item--future .cr-cd-timeline-content {
		opacity: 0.75;
	}

	.cr-cd-timeline-icon-wrap {
		display: flex;
		flex-direction: column;
		align-items: center;
		flex-shrink: 0;
		padding-top: 2px;
	}

	.cr-cd-timeline-icon {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 28px;
		height: 28px;
		border-radius: 50%;
		background: var(--dbg3);
		color: var(--dt3);
		flex-shrink: 0;
	}

	.cr-cd-timeline-item--future .cr-cd-timeline-icon {
		border: 1px dashed var(--dbd);
		background: var(--dbg2);
	}

	.cr-cd-timeline-line {
		width: 1px;
		flex: 1;
		background: var(--dbd2);
		margin: 4px 0;
		min-height: 16px;
	}

	.cr-cd-timeline-item:last-child .cr-cd-timeline-line {
		display: none;
	}

	.cr-cd-timeline-content {
		flex: 1;
		min-width: 0;
		padding-bottom: 20px;
	}

	.cr-cd-timeline-item:last-child .cr-cd-timeline-content {
		padding-bottom: 0;
	}

	.cr-cd-timeline-top {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: 8px;
		margin-bottom: 4px;
	}

	.cr-cd-timeline-subject {
		font-size: 13px;
		font-weight: 500;
		color: var(--dt);
		min-width: 0;
	}

	.cr-cd-timeline-badges {
		display: flex;
		align-items: center;
		gap: 6px;
		flex-shrink: 0;
	}

	.cr-cd-activity-done {
		color: var(--bos-status-success);
	}

	.cr-cd-activity-pending {
		color: var(--dt4);
	}

	.cr-cd-timeline-desc {
		font-size: 12px;
		color: var(--dt3);
		margin: 0 0 4px;
		line-height: 1.5;
	}

	.cr-cd-timeline-outcome {
		font-size: 12px;
		color: var(--bos-status-success-text);
		margin: 0 0 4px;
		font-style: italic;
	}

	.cr-cd-timeline-foot {
		display: flex;
		align-items: center;
		gap: 8px;
		margin-top: 6px;
	}

	.cr-cd-timeline-time {
		font-size: 11px;
		color: var(--dt4);
	}

	.cr-cd-timeline-duration {
		font-size: 11px;
		color: var(--dt4);
		background: var(--dbg3);
		padding: 1px 6px;
		border-radius: 999px;
	}

	/* ── Animation ───────────────────────────────────────────────────────────── */
	@keyframes cr-cd-spin {
		to { transform: rotate(360deg); }
	}

	/* ── Responsive ──────────────────────────────────────────────────────────── */
	@media (max-width: 640px) {
		.cr-cd-hero {
			flex-direction: column;
		}

		.cr-cd-hero-actions {
			width: 100%;
		}

		.cr-cd-contact-btn {
			flex: 1;
			justify-content: center;
		}

		.cr-cd-detail-label {
			min-width: 100px;
		}

		.cr-cd-body {
			padding: 16px;
		}
	}
</style>
