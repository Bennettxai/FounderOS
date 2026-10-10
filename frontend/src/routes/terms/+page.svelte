<!--
	Terms of Service — standalone placeholder page
-->
<script lang="ts">
	// No imports needed for this static page
</script>

<svelte:head>
	<title>Terms of Service - BusinessOS</title>
</svelte:head>

<div class="terms-page">
	<div class="terms-container">
		<header class="terms-header">
			<a href="/register" class="back-link">← Back to Register</a>
			<h1 class="terms-title">Terms of Service</h1>
			<p class="terms-effective">Effective date: January 1, 2025</p>
		</header>

		<section class="terms-section">
			<h2>Usage</h2>
			<p>These terms of service govern your use of BusinessOS and all related services provided by MIOSA LLC. By accessing or using BusinessOS, you agree to be bound by these terms. You may not use the service for any unlawful purpose or in violation of any applicable regulations.</p>
		</section>

		<section class="terms-section">
			<h2>Privacy</h2>
			<p>Your privacy is important to us. We collect and process data necessary to operate the service. We do not sell your personal information to third parties. Your data is stored securely and accessed only as required to provide the service.</p>
		</section>

		<section class="terms-section">
			<h2>Data</h2>
			<p>You retain ownership of all data you create within BusinessOS. We act as a data processor on your behalf. You are responsible for ensuring your data complies with applicable laws. We maintain backups and take reasonable steps to protect your data from loss.</p>
		</section>

		<section class="terms-section">
			<h2>Billing</h2>
			<p>Paid plans are billed on a monthly or annual basis. You may cancel at any time; your access continues until the end of the current billing period. Refunds are handled on a case-by-case basis at our discretion.</p>
		</section>

		<section class="terms-section">
			<h2>Termination</h2>
			<p>Either party may terminate this agreement at any time. We reserve the right to suspend or terminate accounts that violate these terms. Upon termination, your data will be available for export for 30 days before permanent deletion.</p>
		</section>

		<footer class="terms-footer">
			<p>Questions? Contact us at <a href="mailto:support@businessos.ai">support@businessos.ai</a></p>
		</footer>
	</div>
</div>

<style>
	.terms-page {
		min-height: 100vh;
		background: var(--dbg, #f9fafb);
		display: flex;
		justify-content: center;
		padding: 3rem 1.5rem;
	}

	.terms-container {
		width: 100%;
		max-width: 720px;
	}

	.terms-header {
		margin-bottom: 2.5rem;
	}

	.back-link {
		display: inline-block;
		font-size: 0.875rem;
		color: var(--dt3, #6b7280);
		text-decoration: none;
		margin-bottom: 1.5rem;
		transition: color 0.15s ease;
	}

	.back-link:hover {
		color: var(--dt, #111827);
	}

	.terms-title {
		font-size: 2rem;
		font-weight: 700;
		color: var(--dt, #111827);
		margin: 0 0 0.5rem;
		letter-spacing: -0.02em;
	}

	.terms-effective {
		font-size: 0.875rem;
		color: var(--dt3, #6b7280);
		margin: 0;
	}

	.terms-section {
		margin-bottom: 2rem;
		padding-bottom: 2rem;
		border-bottom: 1px solid var(--dbd, #e5e7eb);
	}

	.terms-section:last-of-type {
		border-bottom: none;
	}

	.terms-section h2 {
		font-size: 1.125rem;
		font-weight: 600;
		color: var(--dt, #111827);
		margin: 0 0 0.75rem;
	}

	.terms-section p {
		font-size: 0.9375rem;
		line-height: 1.7;
		color: var(--dt2, #374151);
		margin: 0;
	}

	.terms-footer {
		margin-top: 2.5rem;
		padding-top: 1.5rem;
		border-top: 1px solid var(--dbd, #e5e7eb);
	}

	.terms-footer p {
		font-size: 0.875rem;
		color: var(--dt3, #6b7280);
		margin: 0;
	}

	.terms-footer a {
		color: var(--dt, #111827);
		font-weight: 500;
	}
</style>
