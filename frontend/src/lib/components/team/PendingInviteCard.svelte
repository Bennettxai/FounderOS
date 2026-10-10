<script lang="ts">
	import { Clock, Mail, Copy, Check, X } from 'lucide-svelte';
	import type { WorkspaceInvite } from '$lib/api/workspaces';

	interface Props {
		invite: WorkspaceInvite;
		onCopy?: (invite: WorkspaceInvite) => void;
		onRevoke?: (id: string) => void;
	}

	let { invite, onCopy, onRevoke }: Props = $props();

	let copied = $state(false);

	function formatExpiry(date: string) {
		const d = new Date(date);
		const now = new Date();
		const days = Math.ceil((d.getTime() - now.getTime()) / (1000 * 60 * 60 * 24));
		if (days <= 0) return 'Expired';
		if (days === 1) return '1 day';
		return `${days} days`;
	}

	async function handleCopy() {
		onCopy?.(invite);
		copied = true;
		setTimeout(() => copied = false, 2000);
	}
</script>

<div class="td-invite-card">
	<!-- Pending Badge -->
	<div class="td-invite-card__badge-wrap">
		<span class="td-invite-card__badge">
			<Clock class="w-3 h-3" />
			Pending
		</span>
	</div>

	<!-- Avatar placeholder -->
	<div class="td-invite-card__center">
		<div class="td-invite-card__avatar">
			<Mail class="w-7 h-7" />
		</div>

		<h3 class="td-invite-card__email">{invite.email}</h3>
		<p class="td-invite-card__sub">Invitation sent</p>
	</div>

	<!-- Divider -->
	<div class="td-invite-card__divider"></div>

	<!-- Role Badge -->
	<div class="td-invite-card__role-wrap">
		<span class="td-invite-card__role td-invite-card__role--{invite.role}">
			{invite.role}
		</span>
	</div>

	<!-- Info -->
	<div class="td-invite-card__info">
		<div class="td-invite-card__info-row">
			<span class="td-invite-card__info-label">
				<Clock class="w-4 h-4" />
				Expires in
			</span>
			<span class="td-invite-card__info-val">{formatExpiry(invite.expires_at)}</span>
		</div>
	</div>

	<!-- Actions -->
	<div class="td-invite-card__actions">
		<button
			onclick={handleCopy}
			class="btn-pill btn-pill-soft btn-pill-sm td-invite-card__copy-btn"
		>
			{#if copied}
				<Check class="w-4 h-4" style="color: var(--accent-green);" />
				Copied
			{:else}
				<Copy class="w-4 h-4" />
				Copy Link
			{/if}
		</button>
		<button
			onclick={() => onRevoke?.(invite.id)}
			class="td-invite-card__revoke-btn"
			title="Revoke invitation"
		>
			<X class="w-4 h-4" />
		</button>
	</div>
</div>

<style>
	.td-invite-card {
		background: var(--dbg, #fff);
		border: 1px dashed var(--dbd, #e0e0e0);
		border-radius: 0.75rem;
		padding: 1.25rem;
		position: relative;
		transition: border-color 0.15s;
	}
	.td-invite-card:hover {
		border-color: var(--dbd2, #ccc);
	}
	.td-invite-card__badge-wrap {
		position: absolute;
		top: 0.75rem;
		right: 0.75rem;
	}
	.td-invite-card__badge {
		display: inline-flex;
		align-items: center;
		gap: 0.25rem;
		padding: 0.125rem 0.5rem;
		font-size: 0.6875rem;
		font-weight: 600;
		border-radius: 9999px;
		background: color-mix(in srgb, var(--accent-orange) 12%, transparent);
		color: var(--accent-orange);
	}
	.td-invite-card__center {
		display: flex;
		flex-direction: column;
		align-items: center;
		text-align: center;
		margin-bottom: 1rem;
	}
	.td-invite-card__avatar {
		width: 4rem;
		height: 4rem;
		border-radius: 9999px;
		background: var(--dbg2, #f5f5f5);
		display: flex;
		align-items: center;
		justify-content: center;
		margin-bottom: 0.75rem;
		color: var(--dt3, #888);
		outline: 2px solid var(--dbg2, #f5f5f5);
		outline-offset: 0;
	}
	.td-invite-card__email {
		font-size: 0.875rem;
		font-weight: 700;
		color: var(--dt, #111);
		max-width: 100%;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.td-invite-card__sub {
		font-size: 0.8125rem;
		color: var(--dt3, #888);
	}
	.td-invite-card__divider {
		border-top: 1px solid var(--dbd, #e0e0e0);
		margin: 1rem 0;
	}
	.td-invite-card__role-wrap {
		display: flex;
		justify-content: center;
		margin-bottom: 1rem;
	}
	.td-invite-card__role {
		display: inline-flex;
		align-items: center;
		padding: 0.25rem 0.625rem;
		font-size: 0.6875rem;
		font-weight: 600;
		border-radius: 9999px;
		text-transform: capitalize;
		background: var(--dbg2, #f5f5f5);
		color: var(--dt2, #555);
	}
	.td-invite-card__role--admin {
		background: color-mix(in srgb, var(--accent-purple) 12%, transparent);
		color: var(--accent-purple);
	}
	.td-invite-card__role--manager {
		background: color-mix(in srgb, var(--accent-blue) 12%, transparent);
		color: var(--accent-blue);
	}
	.td-invite-card__info {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
		font-size: 0.8125rem;
	}
	.td-invite-card__info-row {
		display: flex;
		align-items: center;
		justify-content: space-between;
		color: var(--dt3, #888);
	}
	.td-invite-card__info-label {
		display: flex;
		align-items: center;
		gap: 0.375rem;
	}
	.td-invite-card__info-val {
		font-weight: 600;
		color: var(--dt, #111);
	}
	.td-invite-card__actions {
		display: flex;
		gap: 0.5rem;
		margin-top: 1rem;
	}
	.td-invite-card__copy-btn {
		flex: 1;
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 0.375rem;
	}
	.td-invite-card__revoke-btn {
		padding: 0.5rem 0.75rem;
		font-size: 0.875rem;
		font-weight: 500;
		color: var(--color-error);
		background: color-mix(in srgb, var(--color-error) 8%, transparent);
		border: none;
		border-radius: 0.5rem;
		cursor: pointer;
		transition: background 0.15s;
	}
	.td-invite-card__revoke-btn:hover {
		background: color-mix(in srgb, var(--color-error) 15%, transparent);
	}
</style>
