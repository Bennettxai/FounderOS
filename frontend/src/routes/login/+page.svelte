<script lang="ts">
	import { goto } from '$app/navigation';
	import { returnPath } from '$lib/auth/returnTo';
	import { page } from '$app/stores';
	import { fly, fade } from 'svelte/transition';
	import { signInWithEmail, initiateGoogleOAuth, refreshSession, useSession } from '$lib/auth-client';
	import { AuthLayout, FormInput, PasswordInput } from '$lib/components/auth';
	import { onMount } from 'svelte';
	import { initCSRF } from '$lib/api/base';

	// If a session already exists (e.g. we just returned from the Google OAuth
	// deep link, which installs the session then reloads onto this page), go
	// straight into the app instead of sitting on the login screen.
	const session = useSession();
	let redirecting = $state(false);
	$effect(() => {
		if (!redirecting && $session?.data?.user) {
			redirecting = true;
			goto(redirectTo || '/os');
		}
	});

	// Honor ?redirect= (workspace invites) and ?next= (the session gate) —
	// only same-site paths.
	const redirectTo = $derived(returnPath($page.url.searchParams));

	let email = $state('');
	let password = $state('');
	let error = $state('');
	let loading = $state(false);
	let rememberMe = $state(false);
	let rememberedUser = $state<{ email: string; name?: string } | null>(null);

	onMount(async () => {
		await initCSRF();

		try {
			const stored = localStorage.getItem('rememberedUser');
			if (stored) {
				rememberedUser = JSON.parse(stored);
				if (rememberedUser?.email) {
					email = rememberedUser.email;
					rememberMe = true;
				}
			}
		} catch {
			// localStorage read failure — non-critical, proceed without remembered user
		}
	});

	function clearRememberedUser() {
		localStorage.removeItem('rememberedUser');
		rememberedUser = null;
		email = '';
		rememberMe = false;
	}

	async function handleSubmit(e: Event) {
		e.preventDefault();
		error = '';
		loading = true;

		try {
			const result = await signInWithEmail(email, password);

			if (result.error) {
				error = result.error.message || 'Invalid email or password';
				loading = false;
				return;
			}

			await refreshSession();

			if (rememberMe) {
				localStorage.setItem('rememberedUser', JSON.stringify({ email }));
			} else {
				localStorage.removeItem('rememberedUser');
			}

			const isInIframe = window.self !== window.top;

			if (isInIframe) {
				loading = false;
				if (window.parent) {
					window.parent.postMessage({ type: 'login-success' }, window.location.origin);
				}
			} else {
				loading = false;
				// Founder OS opens on its operator console; BusinessOS's own
				// onboarding wizard and desktop stay reachable from the sidebar.
				goto(redirectTo || '/os');
			}
		} catch (err) {
			error = (err as Error).message || 'Authentication failed';
			loading = false;
		}
	}

	function handleGoogleSignIn() {
		initiateGoogleOAuth();
	}
</script>

<AuthLayout>
	<div in:fly={{ y: 20, duration: 400 }}>
		<!-- Logo -->
		<div class="auth-logo-wrap">
			<div class="auth-logo-mark" aria-hidden="true">
				<svg width="36" height="36" viewBox="0 0 48 48" fill="none" xmlns="http://www.w3.org/2000/svg">
					<rect width="48" height="48" rx="11" fill="currentColor" class="auth-logo-bg" />
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
			<span class="auth-logo-text">
				<span class="auth-logo-word">FOUNDER</span><span class="auth-logo-suffix">OS</span>
			</span>
		</div>

		<!-- Header -->
		<div class="auth-header">
			{#if rememberedUser}
				<h1 class="auth-title">Welcome back</h1>
				<div class="auth-remembered-user">
					<div class="auth-avatar">
						{rememberedUser.email.charAt(0).toUpperCase()}
					</div>
					<div class="auth-remembered-info">
						<p class="auth-remembered-email">{rememberedUser.email}</p>
						<button
							type="button"
							onclick={clearRememberedUser}
							class="auth-remembered-switch"
						>
							Use a different account
						</button>
					</div>
				</div>
			{:else}
				<h1 class="auth-title">Sign in</h1>
				<p class="auth-subtitle">Welcome back to Founder OS</p>
			{/if}
		</div>

		<!-- Google OAuth -->
		<button
			type="button"
			onclick={handleGoogleSignIn}
			class="auth-google-btn"
			aria-label="Sign in with Google"
		>
			<svg class="auth-google-icon" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
				<path fill="#4285F4" d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"/>
				<path fill="#34A853" d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"/>
				<path fill="#FBBC05" d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"/>
				<path fill="#EA4335" d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"/>
			</svg>
			Sign in with Google
		</button>

		<!-- Divider -->
		<div class="auth-divider">
			<div class="auth-divider-line"></div>
			<span class="auth-divider-text">or</span>
			<div class="auth-divider-line"></div>
		</div>

		<!-- Form -->
		<form onsubmit={handleSubmit} class="auth-form">
			{#if error}
				<div class="auth-error" in:fly={{ y: -10, duration: 200 }} role="alert">
					<svg class="auth-error-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
					</svg>
					<p>{error}</p>
				</div>
			{/if}

			<FormInput
				id="email"
				label="Email"
				type="email"
				bind:value={email}
				placeholder="you@company.com"
				autocomplete="email"
				required
			/>

			<div class="auth-password-wrap">
				<PasswordInput
					id="password"
					label="Password"
					bind:value={password}
					autocomplete="current-password"
					required
				/>
				<div class="auth-password-meta">
					<label class="auth-remember-label">
						<input
							type="checkbox"
							bind:checked={rememberMe}
							class="auth-checkbox"
						/>
						<span>Remember me</span>
					</label>
					<a href="/forgot-password" class="auth-forgot-link">
						Forgot password?
					</a>
				</div>
			</div>

			<button
				type="submit"
				class="btn-pill btn-pill-primary btn-pill-sm auth-submit-btn"
				disabled={loading}
			>
				{#if loading}
					<svg class="auth-spinner" viewBox="0 0 24 24" aria-hidden="true">
						<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none" />
						<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
					</svg>
					Authenticating...
				{:else}
					Sign in
					<svg class="auth-arrow" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14 5l7 7m0 0l-7 7m7-7H3" />
					</svg>
				{/if}
			</button>
		</form>

		<!-- Sign Up Link -->
		<p class="auth-footer-link">
			Don't have an account?
			<a href={redirectTo ? `/register?redirect=${encodeURIComponent(redirectTo)}` : '/register'} class="auth-footer-action">Sign up</a>
		</p>
	</div>
</AuthLayout>

<style>
	/* ---- Logo ---- */
	.auth-logo-wrap {
		display: flex;
		align-items: center;
		gap: 0.625rem;
		margin-bottom: 2rem;
	}

	.auth-logo-mark {
		color: #1a1a1a;
		flex-shrink: 0;
	}

	:global(.dark) .auth-logo-bg {
		color: #ffffff;
	}

	.auth-logo-bg {
		color: #1a1a1a;
	}

	.auth-logo-text {
		font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
		font-size: 1.125rem;
		font-weight: 800;
		letter-spacing: 0.12em;
		line-height: 1;
	}

	.auth-logo-word {
		color: #1a1a1a;
	}

	:global(.dark) .auth-logo-word {
		color: #f5f5f5;
	}

	.auth-logo-suffix {
		color: rgba(0, 0, 0, 0.3);
		font-weight: 300;
	}

	:global(.dark) .auth-logo-suffix {
		color: rgba(255, 255, 255, 0.3);
	}

	/* ---- Header ---- */
	.auth-header {
		margin-bottom: 1.75rem;
	}

	.auth-title {
		font-size: 1.5rem;
		font-weight: 700;
		letter-spacing: -0.025em;
		margin: 0 0 0.25rem;
		color: var(--dt, #1a1a1a);
		font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
	}

	.auth-subtitle {
		font-size: 0.875rem;
		color: var(--dt3, #666666);
		margin: 0;
		font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
	}

	/* ---- Remembered user ---- */
	.auth-remembered-user {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		margin-top: 0.75rem;
		padding: 0.75rem;
		background: var(--dbg2, #f5f5f5);
		border: 1px solid var(--dbd, rgba(0, 0, 0, 0.08));
		border-radius: 10px;
	}

	.auth-avatar {
		width: 2.5rem;
		height: 2.5rem;
		background: #1a1a1a;
		color: white;
		border-radius: 50%;
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 0.875rem;
		font-weight: 700;
		font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
		flex-shrink: 0;
	}

	:global(.dark) .auth-avatar {
		background: #ffffff;
		color: #1a1a1a;
	}

	.auth-remembered-info {
		flex: 1;
		min-width: 0;
	}

	.auth-remembered-email {
		font-size: 0.875rem;
		font-weight: 500;
		color: var(--dt, #1a1a1a);
		margin: 0 0 0.125rem;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
	}

	.auth-remembered-switch {
		font-size: 0.75rem;
		color: var(--dt3, #666666);
		background: none;
		border: none;
		padding: 0;
		cursor: pointer;
		font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
		transition: color 0.15s ease;
	}

	.auth-remembered-switch:hover {
		color: var(--dt, #1a1a1a);
		text-decoration: underline;
	}

	/* ---- Google button ---- */
	.auth-google-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 0.75rem;
		width: 100%;
		height: 2.75rem;
		background: #ffffff;
		border: 1px solid rgba(0, 0, 0, 0.12);
		border-radius: 8px;
		font-size: 0.9375rem;
		font-weight: 500;
		color: #3c4043;
		cursor: pointer;
		transition: background 0.15s ease, border-color 0.15s ease, box-shadow 0.15s ease;
		font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
		box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
	}

	.auth-google-btn:hover {
		background: #f8f8f8;
		border-color: rgba(0, 0, 0, 0.2);
		box-shadow: 0 2px 6px rgba(0, 0, 0, 0.08);
	}

	.auth-google-btn:active {
		background: #f0f0f0;
		box-shadow: none;
	}

	:global(.dark) .auth-google-btn {
		background: rgba(255, 255, 255, 0.06);
		border-color: rgba(255, 255, 255, 0.12);
		color: #e8eaed;
	}

	:global(.dark) .auth-google-btn:hover {
		background: rgba(255, 255, 255, 0.1);
		border-color: rgba(255, 255, 255, 0.2);
	}

	.auth-google-icon {
		width: 1.25rem;
		height: 1.25rem;
		flex-shrink: 0;
	}

	/* ---- Divider ---- */
	.auth-divider {
		display: flex;
		align-items: center;
		gap: 1rem;
		margin: 1.25rem 0;
	}

	.auth-divider-line {
		flex: 1;
		height: 1px;
		background: var(--dbd, rgba(0, 0, 0, 0.08));
	}

	.auth-divider-text {
		font-size: 0.6875rem;
		color: var(--dt4, #999999);
		text-transform: uppercase;
		letter-spacing: 0.08em;
		font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
	}

	/* ---- Form ---- */
	.auth-form {
		display: flex;
		flex-direction: column;
		gap: 1.25rem;
	}

	.auth-error {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		padding: 0.75rem 1rem;
		background: #fff5f5;
		border: 1px solid #fed7d7;
		border-radius: 8px;
		font-size: 0.875rem;
		color: #c53030;
		font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
	}

	:global(.dark) .auth-error {
		background: rgba(197, 48, 48, 0.12);
		border-color: rgba(197, 48, 48, 0.3);
		color: #fc8181;
	}

	.auth-error-icon {
		width: 1rem;
		height: 1rem;
		flex-shrink: 0;
		color: #e53e3e;
	}

	.auth-password-wrap {
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
	}

	.auth-password-meta {
		display: flex;
		align-items: center;
		justify-content: space-between;
	}

	.auth-remember-label {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		cursor: pointer;
		font-size: 0.75rem;
		color: var(--dt3, #666666);
		font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
	}

	.auth-checkbox {
		width: 1rem;
		height: 1rem;
		border-radius: 4px;
		border: 1px solid var(--dbd, rgba(0, 0, 0, 0.2));
		cursor: pointer;
		accent-color: #1a1a1a;
	}

	.auth-forgot-link {
		font-size: 0.75rem;
		color: var(--dt4, #999999);
		text-decoration: none;
		font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
		transition: color 0.15s ease;
	}

	.auth-forgot-link:hover {
		color: var(--dt, #1a1a1a);
	}

	/* ---- Submit button ---- */
	.auth-submit-btn {
		width: 100%;
		height: 2.75rem;
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 0.5rem;
		font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
	}

	.auth-spinner {
		width: 1rem;
		height: 1rem;
		animation: auth-spin 0.8s linear infinite;
	}

	@keyframes auth-spin {
		from { transform: rotate(0deg); }
		to { transform: rotate(360deg); }
	}

	.auth-arrow {
		width: 1rem;
		height: 1rem;
	}

	/* ---- Footer link ---- */
	.auth-footer-link {
		margin-top: 1.75rem;
		text-align: center;
		font-size: 0.875rem;
		color: var(--dt3, #666666);
		font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
	}

	.auth-footer-action {
		margin-left: 0.25rem;
		color: var(--dt, #1a1a1a);
		font-weight: 600;
		text-decoration: none;
		transition: opacity 0.15s ease;
	}

	.auth-footer-action:hover {
		text-decoration: underline;
	}

	/* ---- Mobile ---- */
	@media (max-width: 480px) {
		.auth-google-btn {
			height: 3rem;
			font-size: 1rem;
		}

		.auth-submit-btn {
			height: 3rem;
			font-size: 1rem;
		}

		.auth-title {
			font-size: 1.375rem;
		}

		.auth-password-meta {
			flex-wrap: wrap;
			gap: 0.5rem;
		}

		.auth-checkbox {
			width: 1.25rem;
			height: 1.25rem;
		}

		.auth-remembered-switch {
			min-height: 44px;
			display: flex;
			align-items: center;
		}

		.auth-forgot-link {
			min-height: 44px;
			display: flex;
			align-items: center;
		}
	}

	@media (max-width: 320px) {
		.auth-logo-wrap {
			margin-bottom: 1.25rem;
		}

		.auth-header {
			margin-bottom: 1.25rem;
		}
	}
</style>
