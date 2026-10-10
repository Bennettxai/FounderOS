<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { crm, lifecycleStageLabels, formatCurrency } from '$lib/stores/crm';
	import type { Company, CreateCompanyData } from '$lib/api/crm';
	import { Plus, Search, Building2, ArrowLeft, Globe, Mail, Phone, X } from 'lucide-svelte';

	// ── Reactive store slices ────────────────────────────────────────────────
	let companies = $derived($crm.companies);
	let loading = $derived($crm.loading);
	let error = $derived($crm.error);

	// ── Local UI state ───────────────────────────────────────────────────────
	let searchQuery = $state('');
	let showAddModal = $state(false);
	let saving = $state(false);
	let saveError = $state<string | null>(null);

	// ── Form state ───────────────────────────────────────────────────────────
	let formName = $state('');
	let formIndustry = $state('');
	let formLifecycle = $state('lead');
	let formWebsite = $state('');
	let formEmail = $state('');
	let formPhone = $state('');

	// ── Derived: filtered companies ──────────────────────────────────────────
	// Use $derived.by for multi-statement logic — avoids reactive feedback loops
	const filteredCompanies = $derived.by(() => {
		const q = searchQuery.trim().toLowerCase();
		if (!q) return companies;
		return companies.filter(
			(c) =>
				c.name.toLowerCase().includes(q) ||
				(c.industry ?? '').toLowerCase().includes(q) ||
				(c.city ?? '').toLowerCase().includes(q)
		);
	});

	// ── Load data ────────────────────────────────────────────────────────────
	onMount(() => {
		crm.loadCompanies();
	});

	// ── Helpers ──────────────────────────────────────────────────────────────
	function getInitial(name: string): string {
		return name.trim().charAt(0).toUpperCase();
	}

	function getLifecyclePillClass(stage: string | undefined): string {
		switch (stage) {
			case 'lead':        return 'cr-co-pill cr-co-pill--info';
			case 'opportunity': return 'cr-co-pill cr-co-pill--warning';
			case 'customer':    return 'cr-co-pill cr-co-pill--success';
			case 'partner':     return 'cr-co-pill cr-co-pill--accent';
			case 'churned':     return 'cr-co-pill cr-co-pill--error';
			default:            return 'cr-co-pill cr-co-pill--neutral';
		}
	}

	function getLifecycleLabel(stage: string | undefined): string {
		if (!stage) return 'Unknown';
		return lifecycleStageLabels[stage] ?? stage;
	}

	function handleCardClick(id: string) {
		goto(`/crm/companies/${id}`);
	}

	function openAddModal() {
		formName = '';
		formIndustry = '';
		formLifecycle = 'lead';
		formWebsite = '';
		formEmail = '';
		formPhone = '';
		saveError = null;
		showAddModal = true;
	}

	function closeAddModal() {
		showAddModal = false;
		saveError = null;
	}

	function closeModal() {
		closeAddModal();
	}

	async function handleCreate() {
		if (!formName.trim()) return;
		saving = true;
		saveError = null;
		try {
			const data: CreateCompanyData = {
				name: formName.trim(),
				industry: formIndustry.trim() || undefined,
				lifecycle_stage: formLifecycle || undefined,
				website: formWebsite.trim() || undefined,
				email: formEmail.trim() || undefined,
				phone: formPhone.trim() || undefined
			};
			await crm.createCompany(data);
			closeAddModal();
		} catch (err) {
			saveError = err instanceof Error ? err.message : 'Failed to create company.';
		} finally {
			saving = false;
		}
	}

	const industries = [
		'Technology', 'Finance', 'Healthcare', 'Retail', 'Manufacturing',
		'Education', 'Real Estate', 'Media', 'Legal', 'Consulting', 'Other'
	];

	const lifecycleOptions = [
		{ value: 'lead',        label: 'Lead' },
		{ value: 'opportunity', label: 'Opportunity' },
		{ value: 'customer',    label: 'Customer' },
		{ value: 'partner',     label: 'Partner' },
		{ value: 'churned',     label: 'Churned' }
	];
</script>

<!-- ═══════════════════════════════════════════════════════════════════════
     PAGE
     ═══════════════════════════════════════════════════════════════════════ -->
<div class="cr-co-page">

	<!-- Header -->
	<header class="cr-co-header">
		<div class="cr-co-header-left">
			<h1 class="cr-co-title">Companies</h1>
			<p class="cr-co-subtitle">Manage your company accounts and relationships</p>
		</div>
		<div class="cr-co-header-actions">
			<a href="/crm" class="btn-pill btn-pill-secondary btn-pill-sm cr-co-back-link">
				<ArrowLeft size={14} />
				Back to Pipeline
			</a>
			<button class="btn-cta cr-co-add-btn" onclick={openAddModal}>
				<Plus size={15} />
				Add Company
			</button>
		</div>
	</header>

	<!-- Search bar -->
	<div class="cr-co-search-bar">
		<div class="cr-co-search-wrap">
			<Search size={15} class="cr-co-search-icon" />
			<input
				class="cr-co-search-input"
				type="search"
				placeholder="Search companies by name, industry, or city..."
				bind:value={searchQuery}
				aria-label="Search companies"
			/>
		</div>
	</div>

	<!-- Body -->
	<div class="cr-co-body">

		<!-- Error state -->
		{#if error}
			<div class="cr-co-error-banner" role="alert">
				<span>{error}</span>
				<button
					class="btn-pill btn-pill-ghost btn-pill-sm"
					onclick={() => crm.loadCompanies()}
				>
					Retry
				</button>
			</div>
		{/if}

		<!-- Loading state -->
		{#if loading && companies.length === 0}
			<div class="cr-co-loading" aria-label="Loading companies">
				<div class="cr-co-spinner" aria-hidden="true"></div>
				<p class="cr-co-loading-text">Loading companies...</p>
			</div>

		<!-- Empty state -->
		{:else if !loading && filteredCompanies.length === 0}
			<div class="cr-co-empty">
				<div class="cr-co-empty-icon" aria-hidden="true">
					<Building2 size={40} />
				</div>
				{#if searchQuery.trim()}
					<p class="cr-co-empty-title">No companies match your search</p>
					<p class="cr-co-empty-sub">Try a different name, industry, or city.</p>
				{:else}
					<p class="cr-co-empty-title">No companies yet</p>
					<p class="cr-co-empty-sub">Add your first company to get started.</p>
					<button class="btn-cta cr-co-empty-cta" onclick={openAddModal}>
						<Plus size={15} />
						Add Company
					</button>
				{/if}
			</div>

		<!-- Companies grid -->
		{:else}
			<div class="cr-co-count-row">
				<span class="cr-co-count">
					{filteredCompanies.length}
					{filteredCompanies.length === 1 ? 'company' : 'companies'}
				</span>
			</div>
			<div class="cr-co-grid">
				{#each filteredCompanies as company (company.id)}
					<button
						class="cr-co-card"
						onclick={() => handleCardClick(company.id)}
						aria-label="View {company.name}"
					>
						<!-- Card top row -->
						<div class="cr-co-card-top">
							<div class="cr-co-avatar" aria-hidden="true">
								{getInitial(company.name)}
							</div>
							<div class="cr-co-card-identity">
								<span class="cr-co-card-name">{company.name}</span>
								{#if company.industry}
									<span class="cr-co-card-industry">{company.industry}</span>
								{/if}
							</div>
						</div>

						<!-- Card middle row -->
						<div class="cr-co-card-mid">
							<span class={getLifecyclePillClass(company.lifecycle_stage)}>
								{getLifecycleLabel(company.lifecycle_stage)}
							</span>
							{#if company.annual_revenue}
								<span class="cr-co-revenue">
									{formatCurrency(company.annual_revenue, company.currency ?? 'USD')}
								</span>
							{/if}
						</div>

						<!-- Card footer row -->
						<div class="cr-co-card-foot">
							{#if company.website}
								<span class="cr-co-card-meta">
									<Globe size={12} aria-hidden="true" />
									<span class="cr-co-card-meta-text">
										{company.website.replace(/^https?:\/\//, '')}
									</span>
								</span>
							{:else if company.email}
								<span class="cr-co-card-meta">
									<Mail size={12} aria-hidden="true" />
									<span class="cr-co-card-meta-text">{company.email}</span>
								</span>
							{:else if company.phone}
								<span class="cr-co-card-meta">
									<Phone size={12} aria-hidden="true" />
									<span class="cr-co-card-meta-text">{company.phone}</span>
								</span>
							{:else}
								<span class="cr-co-card-meta cr-co-card-meta--empty">No contact info</span>
							{/if}
						</div>
					</button>
				{/each}
			</div>
		{/if}
	</div>
</div>

<!-- ═══════════════════════════════════════════════════════════════════════
     ADD COMPANY MODAL
     ═══════════════════════════════════════════════════════════════════════ -->
{#if showAddModal}
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div class="cr-modal-backdrop" onclick={closeModal} onkeydown={(e) => e.key === 'Escape' && closeModal()}>
		<!-- svelte-ignore a11y_no_static_element_interactions -->
		<div class="cr-modal-box" onclick={(e) => e.stopPropagation()} role="dialog" aria-modal="true" aria-label="Add company">

			<div class="cr-modal-box__header">
				<h2 class="cr-modal-box__title">Add Company</h2>
				<button
					class="cr-modal-box__close"
					onclick={closeModal}
					aria-label="Close modal"
					disabled={saving}
				>
					<X size={16} />
				</button>
			</div>

			<div class="cr-modal-box__body">
				{#if saveError}
					<div class="cr-co-save-error" role="alert">{saveError}</div>
				{/if}

				<!-- Company Name -->
				<div class="cr-co-field">
					<label class="bos-label bos-label--required" for="co-name">Company Name</label>
					<input
						id="co-name"
						class="bos-input"
						type="text"
						placeholder="Acme Corp"
						bind:value={formName}
						disabled={saving}
						autocomplete="organization"
					/>
				</div>

				<!-- Industry + Lifecycle -->
				<div class="cr-co-field-row">
					<div class="cr-co-field">
						<label class="bos-label" for="co-industry">Industry</label>
						<select id="co-industry" class="bos-input" bind:value={formIndustry} disabled={saving}>
							<option value="">Select industry</option>
							{#each industries as ind}
								<option value={ind}>{ind}</option>
							{/each}
						</select>
					</div>
					<div class="cr-co-field">
						<label class="bos-label" for="co-lifecycle">Lifecycle Stage</label>
						<select id="co-lifecycle" class="bos-input" bind:value={formLifecycle} disabled={saving}>
							{#each lifecycleOptions as opt}
								<option value={opt.value}>{opt.label}</option>
							{/each}
						</select>
					</div>
				</div>

				<!-- Website -->
				<div class="cr-co-field">
					<label class="bos-label" for="co-website">Website</label>
					<input
						id="co-website"
						class="bos-input"
						type="url"
						placeholder="https://example.com"
						bind:value={formWebsite}
						disabled={saving}
						autocomplete="url"
					/>
				</div>

				<!-- Email + Phone -->
				<div class="cr-co-field-row">
					<div class="cr-co-field">
						<label class="bos-label" for="co-email">Email</label>
						<input
							id="co-email"
							class="bos-input"
							type="email"
							placeholder="hello@example.com"
							bind:value={formEmail}
							disabled={saving}
							autocomplete="email"
						/>
					</div>
					<div class="cr-co-field">
						<label class="bos-label" for="co-phone">Phone</label>
						<input
							id="co-phone"
							class="bos-input"
							type="tel"
							placeholder="+1 (555) 000-0000"
							bind:value={formPhone}
							disabled={saving}
							autocomplete="tel"
						/>
					</div>
				</div>
			</div>

			<div class="cr-modal-box__footer">
				<button
					class="btn-pill btn-pill-ghost"
					onclick={closeModal}
					disabled={saving}
				>
					Cancel
				</button>
				<button
					class="btn-cta cr-co-create-btn"
					onclick={handleCreate}
					disabled={saving || !formName.trim()}
					aria-busy={saving}
				>
					{#if saving}
						<span class="cr-co-btn-spinner" aria-hidden="true"></span>
						Creating...
					{:else}
						<Plus size={15} />
						Create Company
					{/if}
				</button>
			</div>

		</div>
	</div>
{/if}

<style>
	/* ── Page layout ───────────────────────────────────────────────────────── */
	.cr-co-page {
		display: flex;
		flex-direction: column;
		min-height: 100%;
		background: var(--dbg);
		font-family: var(--bos-font-family);
	}

	/* ── Header ────────────────────────────────────────────────────────────── */
	.cr-co-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 1rem;
		padding: 20px 24px 18px;
		background: var(--dbg);
		border-bottom: 1px solid var(--dbd);
		flex-shrink: 0;
	}

	.cr-co-header-left {
		display: flex;
		flex-direction: column;
		gap: 2px;
	}

	.cr-co-title {
		font-size: 18px;
		font-weight: 600;
		color: var(--dt);
		margin: 0;
		line-height: 1.3;
	}

	.cr-co-subtitle {
		font-size: 13px;
		color: var(--dt3);
		margin: 0;
	}

	.cr-co-header-actions {
		display: flex;
		align-items: center;
		gap: 10px;
		flex-shrink: 0;
	}

	.cr-co-back-link {
		display: flex;
		align-items: center;
		gap: 5px;
		text-decoration: none;
	}

	.cr-co-add-btn {
		display: flex;
		align-items: center;
		gap: 6px;
		padding: 8px 16px;
		font-size: 13px;
		font-weight: 500;
	}

	/* ── Search bar ─────────────────────────────────────────────────────────── */
	.cr-co-search-bar {
		padding: 14px 24px;
		border-bottom: 1px solid var(--dbd2);
		background: var(--dbg);
		flex-shrink: 0;
	}

	.cr-co-search-wrap {
		position: relative;
		max-width: 28rem;
	}

	:global(.cr-co-search-icon) {
		position: absolute;
		left: 10px;
		top: 50%;
		transform: translateY(-50%);
		color: var(--dt3);
		pointer-events: none;
	}

	.cr-co-search-input {
		width: 100%;
		padding: 8px 12px 8px 34px;
		background: var(--dbg2);
		border: 1px solid var(--dbd);
		border-radius: 8px;
		font-size: 13px;
		color: var(--dt);
		font-family: var(--bos-font-family);
		outline: none;
		transition: border-color 150ms ease, box-shadow 150ms ease;
	}

	.cr-co-search-input::placeholder {
		color: var(--dt4);
	}

	.cr-co-search-input:focus {
		border-color: var(--bos-status-info);
		box-shadow: 0 0 0 3px var(--bos-status-info-bg);
	}

	/* ── Body ───────────────────────────────────────────────────────────────── */
	.cr-co-body {
		flex: 1;
		padding: 20px 24px;
		display: flex;
		flex-direction: column;
		gap: 16px;
	}

	/* ── Error banner ───────────────────────────────────────────────────────── */
	.cr-co-error-banner {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 12px;
		padding: 12px 16px;
		background: var(--bos-status-error-bg);
		border: 1px solid var(--bos-status-error);
		border-radius: 8px;
		font-size: 13px;
		color: var(--bos-status-error-text);
	}

	/* ── Loading ────────────────────────────────────────────────────────────── */
	.cr-co-loading {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 12px;
		padding: 64px 24px;
	}

	.cr-co-spinner {
		width: 28px;
		height: 28px;
		border: 2px solid var(--dbd);
		border-top-color: var(--dt2);
		border-radius: 50%;
		animation: cr-co-spin 600ms linear infinite;
	}

	.cr-co-loading-text {
		font-size: 13px;
		color: var(--dt3);
		margin: 0;
	}

	/* ── Empty state ────────────────────────────────────────────────────────── */
	.cr-co-empty {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 10px;
		padding: 72px 24px;
		text-align: center;
	}

	.cr-co-empty-icon {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 64px;
		height: 64px;
		border-radius: 16px;
		background: var(--dbg3);
		color: var(--dt3);
		margin-bottom: 4px;
	}

	.cr-co-empty-title {
		font-size: 15px;
		font-weight: 600;
		color: var(--dt);
		margin: 0;
	}

	.cr-co-empty-sub {
		font-size: 13px;
		color: var(--dt3);
		margin: 0;
	}

	.cr-co-empty-cta {
		display: flex;
		align-items: center;
		gap: 6px;
		margin-top: 8px;
		padding: 9px 18px;
		font-size: 13px;
		font-weight: 500;
	}

	/* ── Count row ──────────────────────────────────────────────────────────── */
	.cr-co-count-row {
		display: flex;
		align-items: center;
	}

	.cr-co-count {
		font-size: 12px;
		color: var(--dt3);
		font-variant-numeric: tabular-nums;
		font-family: var(--bos-font-number-family);
	}

	/* ── Grid ───────────────────────────────────────────────────────────────── */
	.cr-co-grid {
		display: grid;
		grid-template-columns: 1fr;
		gap: 12px;
	}

	@media (min-width: 640px) {
		.cr-co-grid {
			grid-template-columns: repeat(2, 1fr);
		}
	}

	@media (min-width: 1024px) {
		.cr-co-grid {
			grid-template-columns: repeat(3, 1fr);
		}
	}

	/* ── Card ───────────────────────────────────────────────────────────────── */
	.cr-co-card {
		display: flex;
		flex-direction: column;
		gap: 12px;
		padding: 16px;
		background: var(--dbg);
		border: 1px solid var(--dbd2);
		border-radius: 10px;
		cursor: pointer;
		text-align: left;
		font-family: var(--bos-font-family);
		transition: box-shadow 150ms ease, border-color 150ms ease;
		width: 100%;
	}

	.cr-co-card:hover {
		box-shadow: var(--bos-shadow-1);
		border-color: var(--dbd);
	}

	.cr-co-card:focus-visible {
		outline: 2px solid var(--bos-status-info);
		outline-offset: 2px;
	}

	/* ── Card top row ───────────────────────────────────────────────────────── */
	.cr-co-card-top {
		display: flex;
		align-items: center;
		gap: 12px;
	}

	.cr-co-avatar {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 38px;
		height: 38px;
		border-radius: 10px;
		background: var(--dbg3);
		color: var(--dt3);
		font-size: 15px;
		font-weight: 600;
		flex-shrink: 0;
		letter-spacing: -0.01em;
	}

	.cr-co-card-identity {
		display: flex;
		flex-direction: column;
		gap: 2px;
		min-width: 0;
	}

	.cr-co-card-name {
		font-size: 14px;
		font-weight: 600;
		color: var(--dt);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
		display: block;
	}

	.cr-co-card-industry {
		font-size: 12px;
		color: var(--dt3);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
		display: block;
	}

	/* ── Card middle row ────────────────────────────────────────────────────── */
	.cr-co-card-mid {
		display: flex;
		align-items: center;
		gap: 8px;
		flex-wrap: wrap;
	}

	.cr-co-revenue {
		font-size: 12px;
		color: var(--dt2);
		font-family: var(--bos-font-number-family);
		font-variant-numeric: tabular-nums;
	}

	/* ── Card footer row ────────────────────────────────────────────────────── */
	.cr-co-card-foot {
		display: flex;
		align-items: center;
	}

	.cr-co-card-meta {
		display: flex;
		align-items: center;
		gap: 5px;
		color: var(--dt3);
		font-size: 12px;
		min-width: 0;
	}

	.cr-co-card-meta-text {
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.cr-co-card-meta--empty {
		font-style: italic;
		color: var(--dt4);
	}

	/* ── Lifecycle pills ────────────────────────────────────────────────────── */
	.cr-co-pill {
		display: inline-flex;
		align-items: center;
		padding: 2px 8px;
		border-radius: 999px;
		font-size: 11px;
		font-weight: 500;
		line-height: 1.6;
		white-space: nowrap;
	}

	.cr-co-pill--info {
		background: var(--bos-status-info-bg);
		color: var(--bos-status-info-text);
	}

	.cr-co-pill--warning {
		background: var(--bos-status-warning-bg);
		color: var(--bos-status-warning-text);
	}

	.cr-co-pill--success {
		background: var(--bos-status-success-bg);
		color: var(--bos-status-success-text);
	}

	.cr-co-pill--accent {
		background: var(--bos-status-accent-bg);
		color: var(--bos-status-accent-text);
	}

	.cr-co-pill--error {
		background: var(--bos-status-error-bg);
		color: var(--bos-status-error-text);
	}

	.cr-co-pill--neutral {
		background: var(--bos-status-neutral-bg);
		color: var(--bos-status-neutral-text);
	}

	/* ── Modal form ─────────────────────────────────────────────────────────── */
	.cr-co-save-error {
		padding: 10px 14px;
		margin-bottom: 16px;
		background: var(--bos-status-error-bg);
		border: 1px solid var(--bos-status-error);
		border-radius: 8px;
		font-size: 13px;
		color: var(--bos-status-error-text);
	}

	.cr-co-field {
		display: flex;
		flex-direction: column;
		gap: 6px;
		margin-bottom: 16px;
	}

	.cr-co-field:last-child {
		margin-bottom: 0;
	}

	.cr-co-field-row {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 12px;
		margin-bottom: 16px;
	}

	.cr-co-field-row .cr-co-field {
		margin-bottom: 0;
	}

	.cr-co-create-btn {
		display: flex;
		align-items: center;
		gap: 6px;
		padding: 8px 18px;
		font-size: 13px;
		font-weight: 500;
	}

	.cr-co-btn-spinner {
		width: 13px;
		height: 13px;
		border: 2px solid rgba(255, 255, 255, 0.3);
		border-top-color: #fff;
		border-radius: 50%;
		animation: cr-co-spin 600ms linear infinite;
		flex-shrink: 0;
	}

	/* ── Animations ─────────────────────────────────────────────────────────── */
	@keyframes cr-co-spin {
		to { transform: rotate(360deg); }
	}

	/* ── Modal backdrop & box ───────────────────────────────────────────────── */
	.cr-modal-backdrop {
		position: fixed;
		inset: 0;
		z-index: 1000;
		background: rgba(0, 0, 0, 0.5);
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.cr-modal-box {
		width: 100%;
		max-width: 480px;
		background: var(--dbg);
		border: 1px solid var(--dbd);
		border-radius: 12px;
		box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.1);
		display: flex;
		flex-direction: column;
		max-height: 90vh;
		overflow: hidden;
	}

	.cr-modal-box__header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 12px;
		padding: 18px 20px 16px;
		border-bottom: 1px solid var(--dbd2);
		flex-shrink: 0;
	}

	.cr-modal-box__title {
		font-size: 15px;
		font-weight: 600;
		color: var(--dt);
		margin: 0;
		line-height: 1.3;
	}

	.cr-modal-box__close {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 28px;
		height: 28px;
		border: none;
		background: transparent;
		color: var(--dt3);
		border-radius: 6px;
		cursor: pointer;
		padding: 0;
		flex-shrink: 0;
		transition: background 120ms ease, color 120ms ease;
	}

	.cr-modal-box__close:hover:not(:disabled) {
		background: var(--dbg3);
		color: var(--dt);
	}

	.cr-modal-box__close:disabled {
		opacity: 0.4;
		cursor: not-allowed;
	}

	.cr-modal-box__body {
		flex: 1;
		overflow-y: auto;
		padding: 20px;
	}

	.cr-modal-box__footer {
		display: flex;
		align-items: center;
		justify-content: flex-end;
		gap: 10px;
		padding: 14px 20px;
		border-top: 1px solid var(--dbd2);
		flex-shrink: 0;
	}
</style>
