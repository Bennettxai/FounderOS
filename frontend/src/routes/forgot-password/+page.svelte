<script lang="ts">
	import { fly, scale } from 'svelte/transition';
	import { onMount } from 'svelte';
	import { AuthLayout, FormInput } from '$lib/components/auth';
	import { requestPasswordReset } from '$lib/auth-client';
	import { initCSRF } from '$lib/api/base';

	onMount(() => { initCSRF(); });

	let email = $state('');
	let error = $state('');
	let loading = $state(false);
	let success = $state(false);
	let sentEmail = $state('');

	async function handleSubmit(e: Event) {
		e.preventDefault();
		error = '';
		loading = true;

		const result = await requestPasswordReset(email);
		loading = false;

		if (!result.ok) {
			error = result.error || 'Could not send reset email';
			return;
		}

		sentEmail = email;
		success = true;
	}

	async function handleResend() {
		loading = true;
		await requestPasswordReset(sentEmail || email);
		loading = false;
	}
</script>

<AuthLayout>
	{#if success}
		<!-- Success State -->
		<div class="text-center" in:fly={{ y: 20, duration: 400 }}>
			<div class="mb-6" in:scale={{ duration: 400, start: 0.5 }}>
				<div class="w-16 h-16 bg-green-50 rounded-full flex items-center justify-center mx-auto">
					<svg class="w-8 h-8 text-green-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
					</svg>
				</div>
			</div>

			<h1 class="text-2xl font-bold text-gray-900 mb-2 font-mono tracking-tight">Check your email</h1>
			<p class="text-gray-500 text-sm font-mono mb-8">
				We sent a reset link to<br />
				<span class="text-gray-900">{sentEmail}</span>
			</p>

			<a
				href="mailto:"
				class="w-full h-12 bg-black text-white rounded-lg text-sm font-medium hover:bg-gray-800 transition-all flex items-center justify-center gap-2 font-mono mb-4"
			>
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
				</svg>
				Open email app
			</a>

			<p class="text-sm text-gray-500 font-mono">
				Didn't receive it?
				<button
					type="button"
					onclick={handleResend}
					disabled={loading}
					class="btn-pill btn-pill-ghost hover:underline ml-1"
				>
					{loading ? 'Sending...' : 'Resend'}
				</button>
			</p>

			<a href="/login" class="inline-flex items-center gap-2 text-xs text-gray-400 hover:text-gray-900 mt-8 transition-colors font-mono group">
				<svg class="w-4 h-4 transition-transform group-hover:-translate-x-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 19l-7-7m0 0l7-7m-7 7h18" />
				</svg>
				Back to sign in
			</a>
		</div>
	{:else}
		<!-- Request Form -->
		<div in:fly={{ y: 20, duration: 400 }}>
			<!-- Header -->
			<div class="mb-8">
				<h1 class="text-2xl font-bold text-gray-900 mb-2 font-mono tracking-tight">Reset password</h1>
				<p class="text-gray-500 text-sm font-mono">Enter your email to receive a reset link</p>
			</div>

			<!-- Form -->
			<form onsubmit={handleSubmit} class="space-y-5">
				{#if error}
					<div class="bg-red-50 border border-red-100 rounded-lg px-4 py-3 flex items-center gap-3" in:fly={{ y: -10, duration: 200 }}>
						<svg class="w-4 h-4 text-red-500 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
						</svg>
						<p class="text-sm text-red-600 font-mono">{error}</p>
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

				<button
					type="submit"
					class="btn-pill btn-pill-primary btn-pill-sm w-full h-12 flex items-center justify-center gap-2 font-mono"
					disabled={loading}
				>
					{#if loading}
						<svg class="animate-spin h-4 w-4" viewBox="0 0 24 24">
							<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none" />
							<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
						</svg>
						Sending...
					{:else}
						Send reset link
						<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
						</svg>
					{/if}
				</button>
			</form>

			<!-- Sign In Link -->
			<p class="mt-8 text-center text-sm text-gray-500 font-mono">
				Remember your password?
				<a href="/login" class="text-gray-900 hover:underline ml-1">
					Sign in
				</a>
			</p>
		</div>
	{/if}
</AuthLayout>

<style>
	/* Ensure tap targets meet 44px minimum on mobile */
	@media (max-width: 480px) {
		:global(.btn-pill.btn-pill-ghost) {
			min-height: 2.75rem;
			padding-top: 0.5rem;
			padding-bottom: 0.5rem;
		}
	}
</style>
