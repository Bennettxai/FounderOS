<!--
	Welcome Page — First-Run Experience
	Shown on first launch before the user has completed setup.
	Standalone full-screen page — NOT inside (app) layout.
-->
<script lang="ts">
	import { browser } from '$app/environment';
	import { goto } from '$app/navigation';

	function handleLocalMode() {
		if (!browser) return;
		localStorage.setItem('businessos_setup_complete', 'local');
		goto('/onboarding');
	}

	function handleCloudMode() {
		if (!browser) return;
		// Mark as cloud intent so layout doesn't redirect back here
		localStorage.setItem('businessos_setup_complete', 'cloud');
		goto('/settings?tab=cloud');
	}
</script>

<svelte:head>
	<title>Welcome to BusinessOS</title>
</svelte:head>

<div class="welcome-root">
	<div class="welcome-inner">
		<!-- Logo mark -->
		<div class="welcome-logo" aria-hidden="true">
			<svg width="48" height="48" viewBox="0 0 48 48" fill="none" xmlns="http://www.w3.org/2000/svg">
				<rect width="48" height="48" rx="12" fill="currentColor" class="logo-bg" />
				<path
					d="M14 24C14 18.477 18.477 14 24 14C29.523 14 34 18.477 34 24C34 29.523 29.523 34 24 34C18.477 34 14 29.523 14 24Z"
					stroke="white"
					stroke-width="2.5"
					fill="none"
				/>
				<path
					d="M20 24H28M24 20V28"
					stroke="white"
					stroke-width="2.5"
					stroke-linecap="round"
				/>
			</svg>
		</div>

		<!-- Heading -->
		<div class="welcome-heading">
			<h1 class="welcome-title">Welcome to BusinessOS</h1>
			<p class="welcome-subtitle">Choose how you want to get started.</p>
		</div>

		<!-- Option cards -->
		<div class="option-grid" role="group" aria-label="Setup options">
			<!-- Local Mode -->
			<button class="option-card" onclick={handleLocalMode} aria-label="Start in Local Mode">
				<div class="option-icon option-icon--local">
					<svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round">
						<rect x="2" y="3" width="20" height="14" rx="2" />
						<path d="M8 21h8M12 17v4" />
					</svg>
				</div>
				<div class="option-content">
					<h2 class="option-title">Start Local</h2>
					<p class="option-desc">Everything runs on your machine. No account needed.</p>
				</div>
				<div class="option-cta">
					<span class="option-btn">Get Started</span>
					<svg class="option-arrow" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
						<path d="M5 12h14M12 5l7 7-7 7" />
					</svg>
				</div>
			</button>

			<!-- Cloud Mode -->
			<button class="option-card" onclick={handleCloudMode} aria-label="Connect to Cloud">
				<div class="option-icon option-icon--cloud">
					<svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round">
						<path d="M18 10h-1.26A8 8 0 1 0 9 20h9a5 5 0 0 0 0-10z" />
					</svg>
				</div>
				<div class="option-content">
					<h2 class="option-title">Connect to Cloud</h2>
					<p class="option-desc">Sync across devices. Get a cloud computer. Invite your team.</p>
				</div>
				<div class="option-cta">
					<span class="option-btn option-btn--secondary">Connect</span>
					<svg class="option-arrow" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
						<path d="M5 12h14M12 5l7 7-7 7" />
					</svg>
				</div>
			</button>
		</div>

		<!-- Footer note -->
		<p class="welcome-footer-note">
			You can always connect to cloud later from Settings.
		</p>
	</div>
</div>

<style>
	/* ---- Layout ---- */
	.welcome-root {
		min-height: 100vh;
		width: 100%;
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 2rem;
		background-color: var(--dbg, #f5f5f5);
		background-image: url('/logos/integrations/MIOSABRANDBackround.png');
		background-size: cover;
		background-position: center;
		background-repeat: no-repeat;
	}

	.welcome-inner {
		width: 100%;
		max-width: 680px;
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 2.5rem;
		text-align: center;
		animation: wl-fade-up 0.6s ease-out both;
	}

	/* ---- Logo ---- */
	.welcome-logo {
		animation: wl-fade-up 0.6s ease-out 0.05s both;
	}

	.logo-bg {
		color: #1a1a1a;
	}

	:global(.dark) .logo-bg {
		color: #ffffff;
	}

	/* ---- Heading ---- */
	.welcome-heading {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
		animation: wl-fade-up 0.6s ease-out 0.1s both;
	}

	.welcome-title {
		font-size: 2.25rem;
		font-weight: 700;
		letter-spacing: -0.03em;
		line-height: 1.15;
		margin: 0;
		color: var(--dt, #1a1a1a);
	}

	.welcome-subtitle {
		font-size: 1.0625rem;
		color: var(--dt3, #666666);
		margin: 0;
	}

	/* ---- Option grid ---- */
	.option-grid {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 1rem;
		width: 100%;
		animation: wl-fade-up 0.6s ease-out 0.18s both;
	}

	.option-card {
		display: flex;
		flex-direction: column;
		align-items: flex-start;
		gap: 1.25rem;
		padding: 1.75rem;
		background: rgba(255, 255, 255, 0.85);
		border: 1px solid var(--dbd, rgba(0, 0, 0, 0.08));
		border-radius: 16px;
		text-align: left;
		cursor: pointer;
		transition:
			transform 0.18s ease,
			box-shadow 0.18s ease,
			background 0.18s ease,
			border-color 0.18s ease;
		backdrop-filter: blur(12px);
		-webkit-backdrop-filter: blur(12px);
	}

	.option-card:hover {
		transform: translateY(-2px);
		box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1);
		border-color: var(--dbd2, rgba(0, 0, 0, 0.15));
		background: rgba(255, 255, 255, 0.95);
	}

	.option-card:active {
		transform: translateY(0);
		box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
	}

	:global(.dark) .option-card {
		background: rgba(30, 30, 30, 0.85);
		border-color: rgba(255, 255, 255, 0.08);
	}

	:global(.dark) .option-card:hover {
		background: rgba(40, 40, 40, 0.95);
		border-color: rgba(255, 255, 255, 0.15);
		box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
	}

	/* ---- Option icon ---- */
	.option-icon {
		width: 52px;
		height: 52px;
		border-radius: 12px;
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
	}

	.option-icon--local {
		background: rgba(0, 0, 0, 0.06);
		color: #1a1a1a;
	}

	.option-icon--cloud {
		background: rgba(59, 130, 246, 0.1);
		color: #3b82f6;
	}

	:global(.dark) .option-icon--local {
		background: rgba(255, 255, 255, 0.08);
		color: #e5e5e5;
	}

	/* ---- Option content ---- */
	.option-content {
		flex: 1;
		display: flex;
		flex-direction: column;
		gap: 0.375rem;
	}

	.option-title {
		font-size: 1.0625rem;
		font-weight: 600;
		color: var(--dt, #1a1a1a);
		margin: 0;
		letter-spacing: -0.01em;
	}

	.option-desc {
		font-size: 0.875rem;
		color: var(--dt3, #666666);
		margin: 0;
		line-height: 1.5;
	}

	/* ---- Option CTA ---- */
	.option-cta {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		margin-top: auto;
	}

	.option-btn {
		font-size: 0.875rem;
		font-weight: 600;
		color: #1a1a1a;
		padding: 0.5rem 1rem;
		border-radius: 8px;
		background: rgba(0, 0, 0, 0.06);
		transition: background 0.15s ease;
	}

	.option-card:hover .option-btn {
		background: rgba(0, 0, 0, 0.1);
	}

	.option-btn--secondary {
		color: #3b82f6;
		background: rgba(59, 130, 246, 0.08);
	}

	.option-card:hover .option-btn--secondary {
		background: rgba(59, 130, 246, 0.15);
	}

	:global(.dark) .option-btn {
		color: #e5e5e5;
		background: rgba(255, 255, 255, 0.08);
	}

	:global(.dark) .option-card:hover .option-btn {
		background: rgba(255, 255, 255, 0.14);
	}

	.option-arrow {
		color: var(--dt4, #999999);
		flex-shrink: 0;
	}

	/* ---- Footer note ---- */
	.welcome-footer-note {
		font-size: 0.8125rem;
		color: var(--dt4, #999999);
		margin: 0;
		animation: wl-fade-up 0.6s ease-out 0.25s both;
	}

	/* ---- Animation ---- */
	@keyframes wl-fade-up {
		from {
			opacity: 0;
			transform: translateY(16px);
		}
		to {
			opacity: 1;
			transform: translateY(0);
		}
	}

	/* ---- Responsive ---- */
	@media (max-width: 600px) {
		.option-grid {
			grid-template-columns: 1fr;
		}

		.welcome-title {
			font-size: 1.75rem;
		}
	}
</style>
