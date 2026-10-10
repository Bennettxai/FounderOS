<script lang="ts">
	import { Loader2, ShieldOff, ShieldCheck } from 'lucide-svelte';
	import { adminControl } from '$lib/api/admin-control';
	import type { AdminUser } from '$lib/api/admin';

	interface Props {
		user: AdminUser;
		onDone: () => void;
	}

	let { user, onDone }: Props = $props();

	let busy = $state(false);
	let error = $state<string | null>(null);
	let notSupported = $state(false);

	// Derive suspended status from platform_role or a dedicated field if present.
	// The backend doesn't surface a `suspended` field on AdminUser yet, so we
	// treat it as unknown until a suspend/unsuspend call confirms it.
	let suspended = $state<boolean | null>(null);

	async function toggleSuspend() {
		busy = true;
		error = null;
		try {
			if (suspended) {
				await adminControl.unsuspendUser(user.id);
				suspended = false;
			} else {
				await adminControl.suspendUser(user.id);
				suspended = true;
			}
			onDone();
		} catch (e) {
			const msg = e instanceof Error ? e.message : 'Action failed';
			// 501 = backend does not yet implement this endpoint
			if (msg.includes('501') || msg.toLowerCase().includes('not implemented')) {
				notSupported = true;
			} else {
				error = msg;
			}
		} finally {
			busy = false;
		}
	}
</script>

{#if !notSupported}
	<div class="user-actions">
		{#if error}
			<div class="banner banner--error">{error}</div>
		{/if}
		<button
			class="action-btn"
			class:action-btn--danger={suspended !== true}
			class:action-btn--safe={suspended === true}
			onclick={toggleSuspend}
			disabled={busy}
			aria-label={suspended ? 'Unsuspend user' : 'Suspend user'}
		>
			{#if busy}
				<Loader2 size={13} class="spin" />
			{:else if suspended}
				<ShieldCheck size={13} />
			{:else}
				<ShieldOff size={13} />
			{/if}
			{suspended ? 'Unsuspend' : 'Suspend'}
		</button>
	</div>
{/if}

<style>
	.user-actions {
		display: flex;
		flex-direction: column;
		gap: 6px;
	}
	.banner--error {
		background: color-mix(in srgb, #ef4444 12%, transparent);
		color: #ef4444;
		padding: 8px 12px;
		border-radius: 9px;
		font-size: 0.8rem;
	}
	.action-btn {
		display: inline-flex;
		align-items: center;
		gap: 5px;
		padding: 5px 11px;
		border-radius: 8px;
		font-size: 0.8rem;
		font-weight: 520;
		cursor: pointer;
		background: transparent;
		border: 1px solid var(--dbd);
		color: var(--dt3);
		transition: color 0.1s, border-color 0.1s;
	}
	.action-btn:disabled {
		opacity: 0.45;
		pointer-events: none;
	}
	.action-btn--danger {
		color: color-mix(in srgb, #ef4444 80%, var(--dt3));
		border-color: color-mix(in srgb, #ef4444 30%, var(--dbd));
	}
	.action-btn--danger:hover {
		color: #ef4444;
		border-color: color-mix(in srgb, #ef4444 60%, transparent);
	}
	.action-btn--safe {
		color: color-mix(in srgb, #22c55e 80%, var(--dt3));
		border-color: color-mix(in srgb, #22c55e 30%, var(--dbd));
	}
	.action-btn--safe:hover {
		color: #22c55e;
		border-color: color-mix(in srgb, #22c55e 60%, transparent);
	}
	:global(.spin) {
		animation: spin 0.9s linear infinite;
	}
	@keyframes spin {
		to {
			transform: rotate(360deg);
		}
	}
</style>
