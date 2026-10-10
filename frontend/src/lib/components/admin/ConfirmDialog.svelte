<script lang="ts">
	import { Loader2 } from 'lucide-svelte';

	interface Props {
		title: string;
		message: string;
		confirmLabel?: string;
		danger?: boolean;
		loading?: boolean;
		onConfirm: () => void;
		onCancel: () => void;
	}

	let {
		title,
		message,
		confirmLabel = 'Confirm',
		danger = false,
		loading = false,
		onConfirm,
		onCancel,
	}: Props = $props();
</script>

<!-- backdrop -->
<div
	class="overlay"
	role="button"
	tabindex="0"
	onclick={onCancel}
	onkeydown={(e) => e.key === 'Escape' && onCancel()}
></div>

<div class="dialog" role="dialog" aria-modal="true" aria-labelledby="dialog-title">
	<h2 id="dialog-title">{title}</h2>
	<p>{message}</p>
	<div class="actions">
		<button class="btn btn-ghost" onclick={onCancel} disabled={loading}>Cancel</button>
		<button
			class="btn"
			class:btn-danger={danger}
			class:btn-primary={!danger}
			onclick={onConfirm}
			disabled={loading}
		>
			{#if loading}<Loader2 size={14} class="spin" />{/if}
			{confirmLabel}
		</button>
	</div>
</div>

<style>
	.overlay {
		position: fixed;
		inset: 0;
		background: rgba(0, 0, 0, 0.6);
		z-index: 50;
	}
	.dialog {
		position: fixed;
		top: 50%;
		left: 50%;
		translate: -50% -50%;
		z-index: 51;
		background: var(--dbg);
		border: 1px solid var(--dbd);
		border-radius: 14px;
		padding: 24px 24px 20px;
		width: 360px;
		max-width: 92vw;
		box-shadow: 0 24px 60px rgba(0, 0, 0, 0.5);
	}
	h2 {
		margin: 0 0 10px;
		font-size: 1rem;
		font-weight: 640;
		color: var(--dt);
	}
	p {
		margin: 0 0 20px;
		font-size: 0.85rem;
		color: var(--dt3);
		line-height: 1.5;
	}
	.actions {
		display: flex;
		justify-content: flex-end;
		gap: 8px;
	}
	.btn {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		padding: 7px 16px;
		min-height: 40px;
		border-radius: 9px;
		font-size: 0.84rem;
		font-weight: 560;
		cursor: pointer;
		border: 1px solid transparent;
		transition: opacity 0.12s;
	}
	.btn:disabled {
		opacity: 0.5;
		pointer-events: none;
	}
	.btn-ghost {
		background: transparent;
		border-color: var(--dbd);
		color: var(--dt3);
	}
	.btn-ghost:hover {
		color: var(--dt);
		border-color: var(--dt3);
	}
	.btn-primary {
		background: #6366f1;
		color: #fff;
	}
	.btn-primary:hover {
		background: #4f46e5;
	}
	.btn-danger {
		background: color-mix(in srgb, #ef4444 18%, transparent);
		border-color: color-mix(in srgb, #ef4444 40%, transparent);
		color: #ef4444;
	}
	.btn-danger:hover {
		background: color-mix(in srgb, #ef4444 28%, transparent);
	}
	:global(.spin) {
		animation: spin 0.9s linear infinite;
	}
	@keyframes spin {
		to {
			transform: rotate(360deg);
		}
	}
	@media (max-width: 480px) {
		.dialog { padding: 20px 18px 16px; }
		.actions { flex-direction: column-reverse; }
		.btn { justify-content: center; }
	}
</style>
