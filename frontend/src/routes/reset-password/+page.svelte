<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { fly, scale } from 'svelte/transition';
	import { onMount } from 'svelte';
	import { AuthLayout, PasswordInput } from '$lib/components/auth';
	import { resetPasswordWithToken } from '$lib/auth-client';
	import { initCSRF } from '$lib/api/base';

	let password = $state('');
	let confirmPassword = $state('');
	let error = $state('');
	let loading = $state(false);
	let success = $state(false);
	let tokenValid = $state(true);
	let token = $state('');

	onMount(async () => {
		// Get token from URL
		token = $page.url.searchParams.get('token') || '';
		if (!token) {
			tokenValid = false;
		}
		// Ensure a CSRF token exists for the reset POST.
		await initCSRF();
	});

	async function handleSubmit(e: Event) {
		e.preventDefault();
		error = '';

		if (password.length < 8) {
			error = 'Password must be at least 8 characters';
			return;
		}

		if (password !== confirmPassword) {
			error = 'Passwords do not match';
			return;
		}

		loading = true;

		const result = await resetPasswordWithToken(token, password);
		loading = false;

		if (!result.ok) {
			error = result.error || 'Could not reset password';
			return;
		}

		success = true;
		// Redirect to login after 2 seconds
		setTimeout(() => goto('/login'), 2000);
	}
</script>

<AuthLayout>
	{#if !tokenValid}
		<!-- Invalid/Expired Token -->
		<div class="text-center" in:fly={{ y: 20, duration: 400 }}>
			<div class="mb-6" in:scale={{ duration: 400, start: 0.5 }}>
				<div class="w-16 h-16 bg-red-100 rounded-full flex items-center justify-center mx-auto">
					<svg class="w-8 h-8 text-red-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
					</svg>
				</div>
			</div>

			<h1 class="text-2xl font-bold text-gray-900 mb-2">Link expired</h1>
			<p class="text-gray-600 mb-8">
				This password reset link has expired or is invalid.
			</p>

			<a
				href="/forgot-password"
				class="btn btn-primary w-full h-12 text-base flex items-center justify-center"
			>
				Request new link
			</a>

			<a href="/login" class="inline-flex items-center gap-2 text-sm text-gray-600 hover:text-gray-900 mt-8 transition-colors">
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 19l-7-7m0 0l7-7m-7 7h18" />
				</svg>
				Back to sign in
			</a>
		</div>
	{:else if success}
		<!-- Success State -->
		<div class="text-center" in:fly={{ y: 20, duration: 400 }}>
			<div class="mb-6" in:scale={{ duration: 400, start: 0.5 }}>
				<div class="w-16 h-16 bg-green-100 rounded-full flex items-center justify-center mx-auto">
					<svg class="w-8 h-8 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
					</svg>
				</div>
			</div>

			<h1 class="text-2xl font-bold text-gray-900 mb-2">Password reset!</h1>
			<p class="text-gray-600 mb-8">
				Your password has been successfully reset.<br />
				Redirecting to sign in...
			</p>

			<a
				href="/login"
				class="btn btn-primary w-full h-12 text-base flex items-center justify-center"
			>
				Sign in now
			</a>
		</div>
	{:else}
		<!-- Reset Form -->
		<div in:fly={{ y: 20, duration: 400 }}>
			<!-- Header -->
			<div class="mb-8">
				<h1 class="text-2xl font-bold text-gray-900 mb-2">Set new password</h1>
				<p class="text-gray-600">Enter your new password below.</p>
			</div>

			<!-- Form -->
			<form onsubmit={handleSubmit} class="space-y-5">
				{#if error}
					<div class="bg-red-50 border border-red-200 rounded-xl px-4 py-3 flex items-center gap-3" in:fly={{ y: -10, duration: 200 }}>
						<svg class="w-5 h-5 text-red-500 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
						</svg>
						<p class="text-sm text-red-700">{error}</p>
					</div>
				{/if}

				<PasswordInput
					id="password"
					label="New password"
					bind:value={password}
					autocomplete="new-password"
					required
					showStrength
				/>

				<PasswordInput
					id="confirmPassword"
					label="Confirm password"
					bind:value={confirmPassword}
					autocomplete="new-password"
					required
					error={confirmPassword && password !== confirmPassword ? 'Passwords do not match' : ''}
				/>

				<button
					type="submit"
					class="btn-pill btn-pill-ghost btn btn-primary w-full h-12 flex items-center justify-center gap-2"
					disabled={loading}
				>
					{#if loading}
						<svg class="animate-spin h-5 w-5" viewBox="0 0 24 24">
							<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none" />
							<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
						</svg>
						Resetting...
					{:else}
						Reset password
					{/if}
				</button>
			</form>
		</div>
	{/if}
</AuthLayout>

<style>
	@media (max-width: 480px) {
		/* "Request new link" and "Sign in now" full-width buttons */
		:global(.btn.btn-primary) {
			min-height: 3rem;
		}

		/* Back to sign in link tap target */
		:global(a[href="/login"]) {
			min-height: 2.75rem;
			display: inline-flex;
			align-items: center;
		}
	}
</style>
