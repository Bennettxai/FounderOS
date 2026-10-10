<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { useSession } from '$lib/auth-client';
	import { acceptWorkspaceInvite } from '$lib/api/workspaces';
	import { switchWorkspace } from '$lib/stores/workspaces';
	import { Check, X, Loader2, Mail, Building2, LogIn } from 'lucide-svelte';
	import { fade, fly } from 'svelte/transition';

	const session = useSession();
	
	type Status = 'idle' | 'accepting' | 'success' | 'error';
	
	let status = $state<Status>('idle');
	let errorMessage = $state('');
	let workspaceName = $state('');

	const token = $derived($page.params.token);
	const isLoggedIn = $derived(!$session.isPending && $session.data?.user);
	const loginUrl = $derived(`/login?redirect=/invite/${token}`);

	async function handleAccept() {
		if (!isLoggedIn) {
			goto(loginUrl);
			return;
		}

		status = 'accepting';
		errorMessage = '';

		if (!token) {
			status = 'error';
			errorMessage = 'Invalid invitation link';
			return;
		}

		try {
			const res = await acceptWorkspaceInvite(token);
			// Make the workspace they just joined the ACTIVE one, so they land in it
			// (not their auto-created personal workspace). Best-effort - never block
			// the success state on the switch.
			if (res?.workspace_id) {
				try {
					await switchWorkspace(res.workspace_id);
				} catch {
					/* switch is best-effort; the membership is already created */
				}
			}
			status = 'success';

			// Redirect to dashboard after short delay
			setTimeout(() => {
				goto('/dashboard');
			}, 2000);
		} catch (err) {
			status = 'error';
			errorMessage = err instanceof Error ? err.message : 'Failed to accept invitation';
		}
	}
</script>

<svelte:head>
	<title>Accept Invitation | BusinessOS</title>
</svelte:head>

<div class="invite-shell min-h-screen flex items-center justify-center bg-gray-50 dark:bg-[var(--bos-background-primary-color)] p-4">
	<div
		class="invite-card bg-white dark:bg-[var(--bos-background-secondary-color)] rounded-2xl shadow-xl w-full max-w-md overflow-hidden"
		in:fly={{ y: 20, duration: 400 }}
	>
		<!-- Header -->
		<div class="invite-header bg-gradient-to-br from-blue-500 to-blue-600 px-8 py-10 text-center text-white">
			<div class="w-16 h-16 bg-white/20 rounded-2xl flex items-center justify-center mx-auto mb-4">
				<Building2 class="w-8 h-8" />
			</div>
			<h1 class="text-2xl font-bold mb-1">You're Invited!</h1>
			<p class="text-blue-100 text-sm">You've been invited to join a workspace</p>
		</div>

		<!-- Content -->
		<div class="invite-content px-8 py-8">
			{#if status === 'idle'}
				<div in:fade={{ duration: 200 }}>
					{#if $session.isPending}
						<div class="flex items-center justify-center py-8">
							<Loader2 class="w-6 h-6 animate-spin text-gray-400" />
						</div>
					{:else if !isLoggedIn}
						<!-- Not logged in state -->
						<div class="text-center">
							<div class="w-12 h-12 bg-amber-100 dark:bg-amber-900/30 rounded-full flex items-center justify-center mx-auto mb-4">
								<LogIn class="w-6 h-6 text-amber-600 dark:text-amber-400" />
							</div>
							<h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-2">
								Sign in to Continue
							</h2>
							<p class="text-gray-500 dark:text-gray-400 text-sm mb-6">
								Please sign in or create an account to accept this invitation.
							</p>
							<button
								onclick={() => goto(loginUrl)}
								class="w-full py-3 px-4 bg-blue-600 hover:bg-blue-700 text-white font-medium rounded-xl transition-colors"
							>
								Sign In to Accept
							</button>
							<p class="mt-4 text-xs text-gray-400 dark:text-gray-500">
								Don't have an account? You can create one after clicking above.
							</p>
						</div>
					{:else}
						<!-- Logged in state -->
						<div class="text-center">
							<div class="flex items-center justify-center gap-3 mb-6 p-4 bg-gray-50 dark:bg-gray-800/50 rounded-xl">
								<Mail class="w-5 h-5 text-gray-400" />
								<span class="text-sm text-gray-600 dark:text-gray-300">
									Signed in as <strong>{$session.data?.user?.email}</strong>
								</span>
							</div>
							<p class="text-gray-500 dark:text-gray-400 text-sm mb-6">
								Click below to accept the invitation and join the workspace.
							</p>
							<button
								onclick={handleAccept}
								class="btn-pill btn-pill-primary w-full flex items-center justify-center gap-2"
							>
								<Check class="w-5 h-5" />
								Accept Invitation
							</button>
						</div>
					{/if}
				</div>

			{:else if status === 'accepting'}
				<div class="text-center py-8" in:fade={{ duration: 200 }}>
					<Loader2 class="w-10 h-10 animate-spin text-blue-500 mx-auto mb-4" />
					<p class="text-gray-600 dark:text-gray-300">Accepting invitation...</p>
				</div>

			{:else if status === 'success'}
				<div class="text-center py-4" in:fade={{ duration: 200 }}>
					<div class="w-16 h-16 bg-green-100 dark:bg-green-900/30 rounded-full flex items-center justify-center mx-auto mb-4">
						<Check class="w-8 h-8 text-green-600 dark:text-green-400" />
					</div>
					<h2 class="text-xl font-semibold text-gray-900 dark:text-white mb-2">
						Welcome to the Team!
					</h2>
					<p class="text-gray-500 dark:text-gray-400 text-sm mb-4">
						You've successfully joined the workspace.
					</p>
					<p class="text-gray-400 dark:text-gray-500 text-xs">
						Redirecting to dashboard...
					</p>
				</div>

			{:else if status === 'error'}
				<div class="text-center py-4" in:fade={{ duration: 200 }}>
					<div class="w-16 h-16 bg-red-100 dark:bg-red-900/30 rounded-full flex items-center justify-center mx-auto mb-4">
						<X class="w-8 h-8 text-red-600 dark:text-red-400" />
					</div>
					<h2 class="text-xl font-semibold text-gray-900 dark:text-white mb-2">
						Unable to Accept
					</h2>
					<p class="text-gray-500 dark:text-gray-400 text-sm mb-6">
						{errorMessage || 'This invitation may have expired or already been used.'}
					</p>
					<div class="flex flex-col gap-3">
						<button
							onclick={() => status = 'idle'}
							class="w-full py-3 px-4 bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 text-gray-900 dark:text-white font-medium rounded-xl transition-colors"
						>
							Try Again
						</button>
						<a
							href="/login"
							class="w-full py-3 px-4 text-blue-600 dark:text-blue-400 font-medium text-center hover:underline"
						>
							Go to Login
						</a>
					</div>
				</div>
			{/if}
		</div>

		<!-- Footer -->
		<div class="invite-footer px-8 py-4 bg-gray-50 dark:bg-[#252527] border-t border-gray-100 dark:border-gray-700">
			<p class="text-xs text-center text-gray-400 dark:text-gray-500">
				By accepting, you agree to the workspace's terms and conditions.
			</p>
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

		.invite-footer {
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
