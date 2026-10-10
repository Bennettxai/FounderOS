<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { useSession } from '$lib/auth-client';
	import { acceptOrgInvite, validateOrgInvite } from '$lib/api/organizations';
	import { initCSRF } from '$lib/api/base';
	import { Check, X, Loader2, Mail, Landmark, LogIn } from 'lucide-svelte';
	import { fade, fly } from 'svelte/transition';

	const session = useSession();

	type Status = 'idle' | 'accepting' | 'success' | 'error';
	let status = $state<Status>('idle');
	let errorMessage = $state('');
	let orgName = $state('');
	let inviteRole = $state('');

	const token = $derived($page.params.token);
	const isLoggedIn = $derived(!$session.isPending && $session.data?.user);
	const loginUrl = $derived(`/login?redirect=/invite/org/${token}`);

	onMount(async () => {
		await initCSRF();
		if (token) {
			try {
				const v = await validateOrgInvite(token);
				if (v.valid) { orgName = v.organization_name ?? ''; inviteRole = v.role ?? ''; }
				else { status = 'error'; errorMessage = v.error || 'This invitation is invalid or has expired'; }
			} catch { /* leave idle; accept will surface errors */ }
		}
	});

	async function handleAccept() {
		if (!isLoggedIn) { goto(loginUrl); return; }
		if (!token) { status = 'error'; errorMessage = 'Invalid invitation link'; return; }
		status = 'accepting';
		errorMessage = '';
		try {
			await acceptOrgInvite(token);
			status = 'success';
			setTimeout(() => goto('/settings?tab=organization'), 1800);
		} catch (err) {
			status = 'error';
			errorMessage = err instanceof Error ? err.message : 'Failed to accept invitation';
		}
	}
</script>

<svelte:head><title>Join Organization | BusinessOS</title></svelte:head>

<div class="invite-shell min-h-screen flex items-center justify-center bg-gray-50 dark:bg-[var(--bos-background-primary-color)] p-4">
	<div class="invite-card bg-white dark:bg-[var(--bos-background-secondary-color)] rounded-2xl shadow-xl w-full max-w-md overflow-hidden" in:fly={{ y: 20, duration: 400 }}>
		<div class="invite-header bg-gradient-to-br from-violet-500 to-indigo-600 px-8 py-10 text-center text-white">
			<div class="w-16 h-16 bg-white/20 rounded-2xl flex items-center justify-center mx-auto mb-4">
				<Landmark class="w-8 h-8" />
			</div>
			<h1 class="text-2xl font-bold mb-1">Join {orgName || 'the organization'}</h1>
			<p class="text-indigo-100 text-sm">You've been invited to an organization{#if inviteRole} as {inviteRole}{/if}</p>
		</div>

		<div class="invite-content px-8 py-8">
			{#if status === 'idle'}
				<div in:fade={{ duration: 200 }}>
					{#if $session.isPending}
						<div class="flex items-center justify-center py-8"><Loader2 class="w-6 h-6 animate-spin text-gray-400" /></div>
					{:else if !isLoggedIn}
						<div class="text-center">
							<div class="w-12 h-12 bg-amber-100 dark:bg-amber-900/30 rounded-full flex items-center justify-center mx-auto mb-4">
								<LogIn class="w-6 h-6 text-amber-600 dark:text-amber-400" />
							</div>
							<h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-2">Sign in to Continue</h2>
							<p class="text-gray-500 dark:text-gray-400 text-sm mb-6">Sign in or create an account to accept this invitation.</p>
							<button onclick={() => goto(loginUrl)} class="w-full py-3 px-4 bg-indigo-600 hover:bg-indigo-700 text-white font-medium rounded-xl transition-colors">Sign In to Accept</button>
						</div>
					{:else}
						<div class="text-center">
							<div class="flex items-center justify-center gap-3 mb-6 p-4 bg-gray-50 dark:bg-gray-800/50 rounded-xl">
								<Mail class="w-5 h-5 text-gray-400" />
								<span class="text-sm text-gray-600 dark:text-gray-300">Signed in as <strong>{$session.data?.user?.email}</strong></span>
							</div>
							<p class="text-gray-500 dark:text-gray-400 text-sm mb-6">Accept to join {orgName || 'the organization'}.</p>
							<button onclick={handleAccept} class="btn-pill btn-pill-primary w-full flex items-center justify-center gap-2">
								<Check class="w-5 h-5" /> Accept Invitation
							</button>
						</div>
					{/if}
				</div>
			{:else if status === 'accepting'}
				<div class="text-center py-8" in:fade={{ duration: 200 }}>
					<Loader2 class="w-10 h-10 animate-spin text-indigo-500 mx-auto mb-4" />
					<p class="text-gray-600 dark:text-gray-300">Joining…</p>
				</div>
			{:else if status === 'success'}
				<div class="text-center py-4" in:fade={{ duration: 200 }}>
					<div class="w-16 h-16 bg-green-100 dark:bg-green-900/30 rounded-full flex items-center justify-center mx-auto mb-4">
						<Check class="w-8 h-8 text-green-600 dark:text-green-400" />
					</div>
					<h2 class="text-xl font-semibold text-gray-900 dark:text-white mb-2">You're in!</h2>
					<p class="text-gray-500 dark:text-gray-400 text-sm">Taking you to your organization…</p>
				</div>
			{:else if status === 'error'}
				<div class="text-center py-4" in:fade={{ duration: 200 }}>
					<div class="w-16 h-16 bg-red-100 dark:bg-red-900/30 rounded-full flex items-center justify-center mx-auto mb-4">
						<X class="w-8 h-8 text-red-600 dark:text-red-400" />
					</div>
					<h2 class="text-xl font-semibold text-gray-900 dark:text-white mb-2">Unable to Accept</h2>
					<p class="text-gray-500 dark:text-gray-400 text-sm mb-6">{errorMessage || 'This invitation may have expired or already been used.'}</p>
					<a href="/login" class="w-full inline-block py-3 px-4 text-indigo-600 dark:text-indigo-400 font-medium text-center hover:underline">Go to Login</a>
				</div>
			{/if}
		</div>
	</div>
</div>

<style>
	@media (max-width: 480px) {
		.invite-shell {
			padding: 0.75rem;
			align-items: flex-start;
			padding-top: 2rem;
		}

		.invite-card {
			border-radius: 1rem;
		}

		.invite-header {
			padding-left: 1.25rem;
			padding-right: 1.25rem;
			padding-top: 2rem;
			padding-bottom: 2rem;
		}

		.invite-content {
			padding-left: 1.25rem;
			padding-right: 1.25rem;
		}
	}

	@media (max-width: 320px) {
		.invite-shell {
			padding: 0;
			align-items: flex-start;
		}

		.invite-card {
			border-radius: 0;
			box-shadow: none;
		}

		.invite-header {
			padding-left: 1rem;
			padding-right: 1rem;
		}

		.invite-content {
			padding-left: 1rem;
			padding-right: 1rem;
		}
	}
</style>
