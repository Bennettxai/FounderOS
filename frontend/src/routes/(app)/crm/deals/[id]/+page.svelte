<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import {
		crm,
		dealStatusLabels,
		dealPriorityLabels,
		activityTypeLabels,
		formatCurrency,
		formatProbability
	} from '$lib/stores/crm';
	import type { Deal, CRMActivity, CreateActivityData, ActivityType } from '$lib/api/crm';
	import {
		ArrowLeft,
		Plus,
		Phone,
		Mail,
		Users,
		FileText,
		Clock,
		CheckCircle2,
		Circle,
		CalendarDays,
		TrendingUp,
		Building2,
		X
	} from 'lucide-svelte';

	// Route param
	const dealId = $derived($page.params.id);
	const embedSuffix = $derived(
		$page.url.searchParams.get('embed') === 'true' ? '?embed=true' : ''
	);

	// Store slices
	let deal = $derived($crm.currentDeal);
	let activities = $derived($crm.activities);
	let loading = $derived($crm.loading);
	let error = $derived($crm.error);

	// Modal state
	let showStatusModal = $state(false);
	let showActivityModal = $state(false);
	let activitySubmitting = $state(false);
	let activityError = $state<string | null>(null);
	let selectedStatus = $state<'open' | 'won' | 'lost'>('open');

	// Activity form
	let activityType = $state<ActivityType>('call');
	let activitySubject = $state('');
	let activityDescription = $state('');

	// Load guard — prevents loop re-triggers
	let loadedDealId: string | null = null;
	$effect(() => {
		const id = dealId;
		if (id && id !== loadedDealId) {
			loadedDealId = id;
			crm.loadDeal(id);
		}
	});

	// Sync selected status when deal loads
	$effect(() => {
		if (deal?.status) {
			selectedStatus = deal.status;
		}
	});

	onMount(() => {
		return () => {
			crm.clearCurrentDeal();
		};
	});

	// Handlers
	function handleBack() {
		goto(`/crm${embedSuffix}`);
	}

	async function handleUpdateStatus() {
		if (!deal) return;
		try {
			await crm.updateDealStatus(deal.id, selectedStatus);
			showStatusModal = false;
		} catch (err) {
			console.error('Failed to update status:', err);
		}
	}

	async function handleLogActivity() {
		if (!deal || !activitySubject.trim()) return;
		activitySubmitting = true;
		activityError = null;
		const data: CreateActivityData = {
			activity_type: activityType,
			subject: activitySubject.trim(),
			description: activityDescription.trim() || undefined,
			deal_id: deal.id,
			activity_date: new Date().toISOString()
		};
		try {
			await crm.createActivity(data);
			showActivityModal = false;
			activitySubject = '';
			activityDescription = '';
			activityType = 'call';
		} catch (err) {
			activityError = err instanceof Error ? err.message : 'Failed to log activity.';
		} finally {
			activitySubmitting = false;
		}
	}

	async function handleCompleteActivity(activityId: string) {
		try {
			await crm.completeActivity(activityId);
		} catch (err) {
			console.error('Failed to complete activity:', err);
		}
	}

	// Helpers
	function formatDate(dateStr: string | undefined): string {
		if (!dateStr) return '—';
		return new Date(dateStr).toLocaleDateString('en-US', {
			month: 'short',
			day: 'numeric',
			year: 'numeric'
		});
	}

	function formatDateShort(dateStr: string | undefined): string {
		if (!dateStr) return '—';
		const d = new Date(dateStr);
		const now = new Date();
		const diff = Math.floor((now.getTime() - d.getTime()) / 86400000);
		if (diff === 0) return 'Today';
		if (diff === 1) return 'Yesterday';
		if (diff === -1) return 'Tomorrow';
		if (diff > 0 && diff < 7) return `${diff}d ago`;
		if (diff < 0 && diff > -7) return `In ${Math.abs(diff)}d`;
		return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
	}

	function getStatusVar(status: string | undefined): string {
		if (status === 'won') return 'var(--bos-status-success)';
		if (status === 'lost') return 'var(--bos-status-error)';
		return 'var(--bos-status-info)';
	}

	function getPriorityVar(priority: string | undefined): string {
		if (priority === 'urgent' || priority === 'critical') return 'var(--bos-priority-critical)';
		if (priority === 'high') return 'var(--bos-priority-high)';
		if (priority === 'medium') return 'var(--bos-priority-medium)';
		return 'var(--bos-priority-low)';
	}

	function getActivityTypeVar(type: ActivityType): string {
		if (type === 'call') return 'var(--bos-status-info)';
		if (type === 'email') return 'var(--bos-status-accent)';
		if (type === 'meeting') return 'var(--bos-status-success)';
		if (type === 'demo') return 'var(--bos-status-warning)';
		return 'var(--bos-status-neutral)';
	}
</script>

<!-- ── Loading ───────────────────────────────────────────────────── -->
{#if loading && !deal}
	<div class="cr-deal-loading">
		<div class="cr-deal-spinner"></div>
		<p class="cr-deal-loading-text">Loading deal...</p>
	</div>

<!-- ── Error ─────────────────────────────────────────────────────── -->
{:else if error && !deal}
	<div class="cr-deal-error-wrap">
		<div class="cr-deal-error-card">
			<p class="cr-deal-error-msg">{error}</p>
			<button class="btn-pill btn-pill-secondary" onclick={handleBack}>
				<ArrowLeft size={14} />
				Back to CRM
			</button>
		</div>
	</div>

<!-- ── Main content ──────────────────────────────────────────────── -->
{:else if deal}
	<!-- Header -->
	<header class="cr-deal-header">
		<div class="cr-deal-header-left">
			<button class="btn-pill btn-pill-ghost btn-pill-sm" onclick={handleBack} aria-label="Back to CRM">
				<ArrowLeft size={14} />
				Back
			</button>
			<div class="cr-deal-header-title-group">
				<h1 class="cr-deal-title">{deal.name}</h1>
				<p class="cr-deal-subtitle">
					{#if deal.company_name}{deal.company_name}{/if}
					{#if deal.company_name && deal.pipeline_name} · {/if}
					{#if deal.pipeline_name}{deal.pipeline_name}{/if}
					{#if deal.stage_name && (deal.company_name || deal.pipeline_name)} / {/if}
					{#if deal.stage_name}{deal.stage_name}{/if}
				</p>
			</div>
		</div>
		<div class="cr-deal-header-right">
			<span class="cr-deal-status-pill" data-status={deal.status ?? 'open'}>
				{dealStatusLabels[deal.status ?? 'open'] ?? 'Open'}
			</span>
			<button
				class="btn-pill btn-pill-secondary btn-pill-sm"
				onclick={() => { showStatusModal = true; }}
				aria-label="Update deal status"
			>
				Update Status
			</button>
			<button
				class="btn-cta"
				onclick={() => { showActivityModal = true; }}
				aria-label="Log activity"
			>
				<Plus size={14} />
				Log Activity
			</button>
		</div>
	</header>

	<!-- Two-column layout -->
	<div class="cr-deal-body">
		<!-- Left column -->
		<div class="cr-deal-left">
			<!-- Deal Details Card -->
			<section class="cr-deal-card" aria-label="Deal details">
				<h2 class="cr-deal-card-heading">Deal Details</h2>
				<div class="cr-deal-metrics-grid">
					<!-- Amount -->
					<div class="cr-deal-metric">
						<span class="cr-deal-metric-label">Amount</span>
						<span class="cr-deal-metric-value cr-deal-metric-value--number">
							{formatCurrency(deal.amount, deal.currency ?? 'USD')}
						</span>
					</div>
					<!-- Probability -->
					<div class="cr-deal-metric">
						<span class="cr-deal-metric-label">Probability</span>
						<span class="cr-deal-metric-value cr-deal-metric-value--number">
							{formatProbability(deal.probability)}
						</span>
					</div>
					<!-- Stage -->
					<div class="cr-deal-metric">
						<span class="cr-deal-metric-label">Stage</span>
						<span class="cr-deal-metric-value">
							{deal.stage_name ?? '—'}
						</span>
					</div>
					<!-- Priority -->
					<div class="cr-deal-metric">
						<span class="cr-deal-metric-label">Priority</span>
						<span class="cr-deal-priority-badge">
							<span
								class="cr-deal-priority-dot"
								style="background: {getPriorityVar(deal.priority)};"
							></span>
							{dealPriorityLabels[deal.priority ?? 'medium'] ?? 'Medium'}
						</span>
					</div>
					<!-- Expected Close -->
					<div class="cr-deal-metric">
						<span class="cr-deal-metric-label">Expected Close</span>
						<span class="cr-deal-metric-value">
							{formatDate(deal.expected_close_date)}
						</span>
					</div>
					<!-- Lead Source -->
					<div class="cr-deal-metric">
						<span class="cr-deal-metric-label">Lead Source</span>
						<span class="cr-deal-metric-value">
							{deal.lead_source ?? '—'}
						</span>
					</div>
				</div>

				{#if deal.description}
					<div class="cr-deal-description-block">
						<p class="cr-deal-description-text">{deal.description}</p>
					</div>
				{/if}
			</section>

			<!-- Activity Timeline -->
			<section class="cr-deal-card" aria-label="Activity timeline">
				<div class="cr-deal-section-header">
					<h2 class="cr-deal-card-heading">Activity Timeline</h2>
					<button
						class="btn-pill btn-pill-ghost btn-pill-sm"
						onclick={() => { showActivityModal = true; }}
						aria-label="Add activity"
					>
						<Plus size={12} />
						Add Activity
					</button>
				</div>

				{#if activities.length === 0}
					<div class="cr-deal-empty">
						<Clock size={32} />
						<p class="cr-deal-empty-text">No activities yet</p>
						<button
							class="btn-pill btn-pill-secondary btn-pill-sm"
							onclick={() => { showActivityModal = true; }}
						>
							Log first activity
						</button>
					</div>
				{:else}
					<ol class="cr-deal-timeline">
						{#each activities as activity (activity.id)}
							<li class="cr-deal-timeline-item">
								<!-- Icon -->
								<div
									class="cr-deal-timeline-icon"
									style="background: color-mix(in srgb, {getActivityTypeVar(activity.activity_type)} 12%, transparent); color: {getActivityTypeVar(activity.activity_type)};"
									aria-hidden="true"
								>
									{#if activity.activity_type === 'call'}
										<Phone size={14} />
									{:else if activity.activity_type === 'email'}
										<Mail size={14} />
									{:else if activity.activity_type === 'meeting'}
										<Users size={14} />
									{:else}
										<FileText size={14} />
									{/if}
								</div>

								<!-- Content -->
								<div class="cr-deal-timeline-content">
									<div class="cr-deal-timeline-row">
										<span class="cr-deal-timeline-subject">{activity.subject}</span>
										<time class="cr-deal-timeline-date" datetime={activity.activity_date}>
											{formatDateShort(activity.activity_date)}
										</time>
									</div>
									{#if activity.description}
										<p class="cr-deal-timeline-desc">{activity.description}</p>
									{/if}
									<div class="cr-deal-timeline-footer">
										<span
											class="cr-deal-type-pill"
											style="background: color-mix(in srgb, {getActivityTypeVar(activity.activity_type)} 10%, transparent); color: {getActivityTypeVar(activity.activity_type)};"
										>
											{activityTypeLabels[activity.activity_type]}
										</span>
										{#if !activity.is_completed}
											<button
												class="btn-pill btn-pill-ghost btn-pill-sm"
												onclick={() => handleCompleteActivity(activity.id)}
												aria-label="Mark activity as complete"
											>
												<Circle size={12} />
												Mark complete
											</button>
										{:else}
											<span class="cr-deal-completed-badge">
												<CheckCircle2 size={12} />
												Completed
											</span>
										{/if}
									</div>
								</div>
							</li>
						{/each}
					</ol>
				{/if}
			</section>
		</div>

		<!-- Right sidebar -->
		<aside class="cr-deal-sidebar">
			<!-- Deal Score -->
			{#if deal.deal_score !== undefined}
				<div class="cr-deal-card cr-deal-card--compact">
					<div class="cr-deal-sidebar-row">
						<span class="cr-deal-sidebar-label">
							<TrendingUp size={13} />
							Deal Score
						</span>
						<span class="cr-deal-score-value">{deal.deal_score}</span>
					</div>
				</div>
			{/if}

			<!-- Important Dates -->
			<div class="cr-deal-card cr-deal-card--compact">
				<h3 class="cr-deal-sidebar-heading">
					<CalendarDays size={13} />
					Important Dates
				</h3>
				<dl class="cr-deal-date-list">
					<div class="cr-deal-date-row">
						<dt class="cr-deal-date-label">Created</dt>
						<dd class="cr-deal-date-value">{formatDate(deal.created_at)}</dd>
					</div>
					<div class="cr-deal-date-row">
						<dt class="cr-deal-date-label">Updated</dt>
						<dd class="cr-deal-date-value">{formatDate(deal.updated_at)}</dd>
					</div>
					{#if deal.actual_close_date}
						<div class="cr-deal-date-row">
							<dt class="cr-deal-date-label">Closed</dt>
							<dd class="cr-deal-date-value">{formatDate(deal.actual_close_date)}</dd>
						</div>
					{/if}
					{#if deal.expected_close_date}
						<div class="cr-deal-date-row">
							<dt class="cr-deal-date-label">Expected</dt>
							<dd class="cr-deal-date-value">{formatDate(deal.expected_close_date)}</dd>
						</div>
					{/if}
				</dl>
			</div>

			<!-- Company -->
			{#if deal.company_id}
				<div class="cr-deal-card cr-deal-card--compact">
					<h3 class="cr-deal-sidebar-heading">
						<Building2 size={13} />
						Company
					</h3>
					<a
						href="/crm/companies/{deal.company_id}{embedSuffix}"
						class="cr-deal-company-link"
					>
						{deal.company_name ?? deal.company_id}
					</a>
				</div>
			{/if}
		</aside>
	</div>
{/if}

<!-- ── Update Status Modal ───────────────────────────────────────── -->
{#if showStatusModal}
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div
		class="crd-modal-backdrop"
		onclick={() => { showStatusModal = false; }}
		onkeydown={(e) => e.key === 'Escape' && (showStatusModal = false)}
	>
		<!-- svelte-ignore a11y_no_static_element_interactions -->
		<div
			class="crd-modal-box"
			onclick={(e) => e.stopPropagation()}
			role="dialog"
			aria-modal="true"
			aria-label="Update deal status"
		>
			<div class="crd-modal-header">
				<h2 class="crd-modal-title">Update Status</h2>
				<button
					class="btn-pill btn-pill-icon btn-pill-ghost btn-pill-sm"
					onclick={() => { showStatusModal = false; }}
					aria-label="Close modal"
				>
					<X size={14} />
				</button>
			</div>
			<div class="crd-modal-body">
				<div class="cr-deal-status-options">
					{#each (['open', 'won', 'lost'] as const) as statusOption}
						<button
							class="cr-deal-status-option"
							class:cr-deal-status-option--active={selectedStatus === statusOption}
							data-status={statusOption}
							onclick={() => { selectedStatus = statusOption; }}
							aria-pressed={selectedStatus === statusOption}
						>
							<span class="cr-deal-status-option-label">{dealStatusLabels[statusOption]}</span>
							<span class="cr-deal-status-option-desc">
								{#if statusOption === 'open'}In progress — deal is actively being worked{:else if statusOption === 'won'}Deal successfully closed and won{:else}Deal closed without a win{/if}
							</span>
						</button>
					{/each}
				</div>
			</div>
			<div class="crd-modal-footer">
				<button
					class="btn-pill btn-pill-secondary"
					onclick={() => { showStatusModal = false; }}
				>
					Cancel
				</button>
				<button class="btn-cta" onclick={handleUpdateStatus}>
					Save Status
				</button>
			</div>
		</div>
	</div>
{/if}

<!-- ── Log Activity Modal ────────────────────────────────────────── -->
{#if showActivityModal}
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div
		class="crd-modal-backdrop"
		onclick={() => { showActivityModal = false; activityError = null; }}
		onkeydown={(e) => e.key === 'Escape' && (showActivityModal = false)}
	>
		<!-- svelte-ignore a11y_no_static_element_interactions -->
		<div
			class="crd-modal-box"
			onclick={(e) => e.stopPropagation()}
			role="dialog"
			aria-modal="true"
			aria-label="Log activity"
		>
			<div class="crd-modal-header">
				<h2 class="crd-modal-title">Log Activity</h2>
				<button
					class="btn-pill btn-pill-icon btn-pill-ghost btn-pill-sm"
					onclick={() => { showActivityModal = false; activityError = null; }}
					aria-label="Close modal"
				>
					<X size={14} />
				</button>
			</div>
			<div class="crd-modal-body">
				{#if activityError}
					<p class="cr-deal-form-error">{activityError}</p>
				{/if}
				<div class="cr-deal-form-field">
					<label class="cr-deal-form-label" for="activity-type">Activity Type</label>
					<select
						id="activity-type"
						class="cr-deal-form-input"
						bind:value={activityType}
					>
						{#each Object.entries(activityTypeLabels) as [value, label]}
							<option value={value}>{label}</option>
						{/each}
					</select>
				</div>
				<div class="cr-deal-form-field">
					<label class="cr-deal-form-label cr-deal-form-label--req" for="activity-subject">Subject</label>
					<input
						id="activity-subject"
						type="text"
						class="cr-deal-form-input"
						placeholder="e.g. Discovery call with CEO"
						bind:value={activitySubject}
						required
					/>
				</div>
				<div class="cr-deal-form-field">
					<label class="cr-deal-form-label" for="activity-description">Description</label>
					<textarea
						id="activity-description"
						class="cr-deal-form-input cr-deal-textarea"
						placeholder="Optional notes about this activity..."
						bind:value={activityDescription}
						rows="3"
					></textarea>
				</div>
			</div>
			<div class="crd-modal-footer">
				<button
					class="btn-pill btn-pill-secondary"
					onclick={() => { showActivityModal = false; activityError = null; }}
				>
					Cancel
				</button>
				<button
					class="btn-cta"
					onclick={handleLogActivity}
					disabled={activitySubmitting || !activitySubject.trim()}
				>
					{#if activitySubmitting}Logging...{:else}Log Activity{/if}
				</button>
			</div>
		</div>
	</div>
{/if}

<style>
	/* ── Loading ───────────────────────────────────────────────── */
	.cr-deal-loading {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 12px;
		height: 50vh;
	}

	.cr-deal-spinner {
		width: 28px;
		height: 28px;
		border: 2px solid var(--dbd);
		border-top-color: var(--dt);
		border-radius: 50%;
		animation: cr-deal-spin 0.7s linear infinite;
	}

	@keyframes cr-deal-spin {
		to { transform: rotate(360deg); }
	}

	.cr-deal-loading-text {
		font-size: 13px;
		color: var(--dt3);
		font-family: var(--bos-font-family);
	}

	/* ── Error ─────────────────────────────────────────────────── */
	.cr-deal-error-wrap {
		display: flex;
		align-items: center;
		justify-content: center;
		height: 50vh;
	}

	.cr-deal-error-card {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 12px;
		padding: 32px;
		background: var(--dbg2);
		border: 1px solid var(--dbd);
		border-radius: 10px;
		max-width: 360px;
		text-align: center;
	}

	.cr-deal-error-msg {
		font-size: 13px;
		color: var(--bos-status-error);
		font-family: var(--bos-font-family);
	}

	/* ── Header ────────────────────────────────────────────────── */
	.cr-deal-header {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: 16px;
		padding: 16px 24px;
		background: var(--dbg);
		border-bottom: 1px solid var(--dbd);
		flex-wrap: wrap;
	}

	.cr-deal-header-left {
		display: flex;
		align-items: flex-start;
		gap: 12px;
	}

	.cr-deal-header-title-group {
		display: flex;
		flex-direction: column;
		gap: 3px;
	}

	.cr-deal-title {
		font-size: 18px;
		font-weight: 600;
		color: var(--dt);
		font-family: var(--bos-font-family);
		margin: 0;
		line-height: 1.3;
	}

	.cr-deal-subtitle {
		font-size: 12px;
		color: var(--dt3);
		font-family: var(--bos-font-family);
		margin: 0;
	}

	.cr-deal-header-right {
		display: flex;
		align-items: center;
		gap: 8px;
		flex-shrink: 0;
	}

	/* Status pill in header */
	.cr-deal-status-pill {
		display: inline-flex;
		align-items: center;
		padding: 3px 10px;
		border-radius: 999px;
		font-size: 11px;
		font-weight: 500;
		font-family: var(--bos-font-family);
		letter-spacing: 0.02em;
		text-transform: uppercase;
	}

	.cr-deal-status-pill[data-status='open'] {
		background: color-mix(in srgb, var(--bos-status-info) 12%, transparent);
		color: var(--bos-status-info);
	}

	.cr-deal-status-pill[data-status='won'] {
		background: color-mix(in srgb, var(--bos-status-success) 12%, transparent);
		color: var(--bos-status-success);
	}

	.cr-deal-status-pill[data-status='lost'] {
		background: color-mix(in srgb, var(--bos-status-error) 12%, transparent);
		color: var(--bos-status-error);
	}

	/* ── Body layout ───────────────────────────────────────────── */
	.cr-deal-body {
		display: grid;
		grid-template-columns: 2fr 1fr;
		gap: 20px;
		padding: 20px 24px;
		align-items: start;
		font-family: var(--bos-font-family);
	}

	@media (max-width: 900px) {
		.cr-deal-body {
			grid-template-columns: 1fr;
		}
	}

	.cr-deal-left {
		display: flex;
		flex-direction: column;
		gap: 16px;
	}

	.cr-deal-sidebar {
		display: flex;
		flex-direction: column;
		gap: 12px;
	}

	/* ── Cards ─────────────────────────────────────────────────── */
	.cr-deal-card {
		background: var(--dbg);
		border: 1px solid var(--dbd2);
		border-radius: 10px;
		padding: 20px;
		box-shadow: var(--bos-shadow-1);
	}

	.cr-deal-card--compact {
		padding: 14px 16px;
	}

	.cr-deal-card-heading {
		font-size: 13px;
		font-weight: 600;
		color: var(--dt);
		margin: 0 0 16px 0;
		text-transform: uppercase;
		letter-spacing: 0.04em;
	}

	/* ── Deal Metrics Grid ─────────────────────────────────────── */
	.cr-deal-metrics-grid {
		display: grid;
		grid-template-columns: repeat(2, 1fr);
		gap: 16px 24px;
	}

	@media (min-width: 640px) {
		.cr-deal-metrics-grid {
			grid-template-columns: repeat(3, 1fr);
		}
	}

	.cr-deal-metric {
		display: flex;
		flex-direction: column;
		gap: 4px;
	}

	.cr-deal-metric-label {
		font-size: 11px;
		color: var(--dt3);
		font-weight: 500;
		letter-spacing: 0.03em;
		text-transform: uppercase;
	}

	.cr-deal-metric-value {
		font-size: 15px;
		font-weight: 500;
		color: var(--dt);
	}

	.cr-deal-metric-value--number {
		font-family: var(--bos-font-number-family);
		font-size: 22px;
		font-weight: 600;
		letter-spacing: -0.02em;
	}

	.cr-deal-priority-badge {
		display: flex;
		align-items: center;
		gap: 6px;
		font-size: 13px;
		font-weight: 500;
		color: var(--dt);
	}

	.cr-deal-priority-dot {
		width: 8px;
		height: 8px;
		border-radius: 50%;
		flex-shrink: 0;
	}

	/* Description block */
	.cr-deal-description-block {
		margin-top: 16px;
		padding-top: 16px;
		border-top: 1px solid var(--dbd2);
	}

	.cr-deal-description-text {
		font-size: 13px;
		color: var(--dt2);
		line-height: 1.6;
		margin: 0;
		white-space: pre-wrap;
	}

	/* ── Section header ────────────────────────────────────────── */
	.cr-deal-section-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-bottom: 16px;
	}

	.cr-deal-section-header .cr-deal-card-heading {
		margin-bottom: 0;
	}

	/* ── Empty state ───────────────────────────────────────────── */
	.cr-deal-empty {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 10px;
		padding: 32px 0;
		color: var(--dt4);
		text-align: center;
	}

	.cr-deal-empty-text {
		font-size: 13px;
		color: var(--dt3);
		margin: 0;
	}

	/* ── Timeline ──────────────────────────────────────────────── */
	.cr-deal-timeline {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 0;
	}

	.cr-deal-timeline-item {
		display: flex;
		gap: 12px;
		padding: 12px 0;
		border-bottom: 1px solid var(--dbd2);
	}

	.cr-deal-timeline-item:last-child {
		border-bottom: none;
		padding-bottom: 0;
	}

	.cr-deal-timeline-icon {
		width: 32px;
		height: 32px;
		border-radius: 8px;
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
		margin-top: 2px;
	}

	.cr-deal-timeline-content {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: 4px;
	}

	.cr-deal-timeline-row {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: 8px;
	}

	.cr-deal-timeline-subject {
		font-size: 13px;
		font-weight: 500;
		color: var(--dt);
		line-height: 1.4;
	}

	.cr-deal-timeline-date {
		font-size: 11px;
		color: var(--dt3);
		white-space: nowrap;
		flex-shrink: 0;
	}

	.cr-deal-timeline-desc {
		font-size: 12px;
		color: var(--dt2);
		line-height: 1.5;
		margin: 0;
	}

	.cr-deal-timeline-footer {
		display: flex;
		align-items: center;
		gap: 8px;
		margin-top: 2px;
	}

	.cr-deal-type-pill {
		display: inline-flex;
		align-items: center;
		padding: 2px 8px;
		border-radius: 999px;
		font-size: 11px;
		font-weight: 500;
	}

	.cr-deal-completed-badge {
		display: inline-flex;
		align-items: center;
		gap: 4px;
		font-size: 11px;
		color: var(--bos-status-success);
		font-weight: 500;
	}

	/* ── Sidebar components ────────────────────────────────────── */
	.cr-deal-sidebar-heading {
		display: flex;
		align-items: center;
		gap: 6px;
		font-size: 11px;
		font-weight: 600;
		color: var(--dt3);
		text-transform: uppercase;
		letter-spacing: 0.04em;
		margin: 0 0 10px 0;
	}

	.cr-deal-sidebar-row {
		display: flex;
		align-items: center;
		justify-content: space-between;
	}

	.cr-deal-sidebar-label {
		display: flex;
		align-items: center;
		gap: 6px;
		font-size: 11px;
		font-weight: 600;
		color: var(--dt3);
		text-transform: uppercase;
		letter-spacing: 0.04em;
	}

	.cr-deal-score-value {
		font-size: 24px;
		font-weight: 700;
		color: var(--dt);
		font-family: var(--bos-font-number-family);
		letter-spacing: -0.02em;
	}

	.cr-deal-date-list {
		display: flex;
		flex-direction: column;
		gap: 6px;
		margin: 0;
	}

	.cr-deal-date-row {
		display: flex;
		justify-content: space-between;
		align-items: baseline;
		gap: 8px;
	}

	.cr-deal-date-label {
		font-size: 12px;
		color: var(--dt3);
	}

	.cr-deal-date-value {
		font-size: 12px;
		color: var(--dt);
		font-weight: 500;
		text-align: right;
	}

	.cr-deal-company-link {
		font-size: 13px;
		color: var(--bos-status-info);
		font-weight: 500;
		text-decoration: none;
		display: block;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.cr-deal-company-link:hover {
		text-decoration: underline;
	}

	/* ── Modal content positioning ─────────────────────────────── */
	.cr-deal-modal-content {
		position: fixed;
		top: 50%;
		left: 50%;
		transform: translate(-50%, -50%);
		z-index: 1000;
		width: calc(100vw - 2rem);
	}

	/* ── Status options ────────────────────────────────────────── */
	.cr-deal-status-options {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.cr-deal-status-option {
		display: flex;
		flex-direction: column;
		gap: 3px;
		padding: 12px 14px;
		border-radius: 8px;
		border: 1px solid var(--dbd);
		background: var(--dbg2);
		cursor: pointer;
		text-align: left;
		transition: border-color 150ms ease;
		font-family: var(--bos-font-family);
	}

	.cr-deal-status-option:hover {
		background: var(--dbg3);
	}

	.cr-deal-status-option--active[data-status='open'] {
		border-color: var(--bos-status-info);
		background: color-mix(in srgb, var(--bos-status-info) 8%, var(--dbg2));
	}

	.cr-deal-status-option--active[data-status='won'] {
		border-color: var(--bos-status-success);
		background: color-mix(in srgb, var(--bos-status-success) 8%, var(--dbg2));
	}

	.cr-deal-status-option--active[data-status='lost'] {
		border-color: var(--bos-status-error);
		background: color-mix(in srgb, var(--bos-status-error) 8%, var(--dbg2));
	}

	.cr-deal-status-option-label {
		font-size: 13px;
		font-weight: 600;
		color: var(--dt);
	}

	.cr-deal-status-option-desc {
		font-size: 12px;
		color: var(--dt3);
	}

	/* ── Form ──────────────────────────────────────────────────── */
	.cr-deal-form-field {
		display: flex;
		flex-direction: column;
		gap: 6px;
		margin-bottom: 14px;
	}

	.cr-deal-form-field:last-child {
		margin-bottom: 0;
	}

	.cr-deal-textarea {
		resize: vertical;
		min-height: 72px;
	}

	.cr-deal-form-error {
		font-size: 12px;
		color: var(--bos-status-error);
		margin: 0 0 12px;
		padding: 8px 12px;
		background: color-mix(in srgb, var(--bos-status-error) 8%, transparent);
		border-radius: 6px;
		border: 1px solid color-mix(in srgb, var(--bos-status-error) 20%, transparent);
	}

	/* ── Plain-HTML modals (no bits-ui) ────────────────────────── */
	.crd-modal-backdrop {
		position: fixed;
		inset: 0;
		z-index: 1000;
		background: rgba(0, 0, 0, 0.5);
		display: flex;
		align-items: center;
		justify-content: center;
	}
	.crd-modal-box {
		width: 100%;
		max-width: 460px;
		background: var(--dbg);
		border: 1px solid var(--dbd);
		border-radius: 12px;
		box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.15);
		display: flex;
		flex-direction: column;
		max-height: 90vh;
		overflow: hidden;
	}
	.crd-modal-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 1rem 1.25rem;
		border-bottom: 1px solid var(--dbd);
		flex-shrink: 0;
	}
	.crd-modal-title {
		margin: 0;
		font-size: 0.9375rem;
		font-weight: 600;
		color: var(--dt);
	}
	.crd-modal-body {
		padding: 1.25rem;
		overflow-y: auto;
		flex: 1;
		display: flex;
		flex-direction: column;
		gap: 0.875rem;
	}
	.crd-modal-footer {
		display: flex;
		align-items: center;
		justify-content: flex-end;
		gap: 0.5rem;
		padding: 0.875rem 1.25rem;
		border-top: 1px solid var(--dbd);
		flex-shrink: 0;
	}
	.cr-deal-form-label {
		display: block;
		font-size: 0.8125rem;
		font-weight: 500;
		color: var(--dt2);
		margin-bottom: 0.375rem;
	}
	.cr-deal-form-label--req::after {
		content: ' *';
		color: var(--bos-status-error);
	}
	.cr-deal-form-input {
		width: 100%;
		height: 34px;
		padding: 0 0.75rem;
		border: 1px solid var(--dbd);
		border-radius: 7px;
		background: var(--dbg2);
		color: var(--dt);
		font-size: 0.8125rem;
		font-family: var(--bos-font-family);
		outline: none;
		box-sizing: border-box;
	}
	.cr-deal-form-input:focus {
		border-color: var(--bos-nav-active);
		box-shadow: 0 0 0 2px color-mix(in srgb, var(--bos-nav-active) 18%, transparent);
	}
	select.cr-deal-form-input {
		cursor: pointer;
	}
	textarea.cr-deal-form-input {
		height: auto;
		padding: 0.5rem 0.75rem;
		resize: vertical;
		min-height: 72px;
	}
</style>
