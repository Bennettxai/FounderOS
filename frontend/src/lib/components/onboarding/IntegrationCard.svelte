<!--
  IntegrationCard.svelte
  OAuth integration card with connect/connected states
-->
<script lang="ts">
	import { CheckIcon } from './icons';

	type ConnectionStatus = 'disconnected' | 'connecting' | 'connected' | 'error';

	interface Props {
		name: string;
		icon: string;
		description?: string;
		status?: ConnectionStatus;
		onConnect?: () => void;
		onDisconnect?: () => void;
		class?: string;
	}

	let {
		name,
		icon,
		description = '',
		status = $bindable('disconnected'),
		onConnect,
		onDisconnect,
		class: className = ''
	}: Props = $props();

	const isConnected = $derived(status === 'connected');
	const isConnecting = $derived(status === 'connecting');

	function handleClick() {
		if (isConnected) {
			onDisconnect?.();
		} else if (!isConnecting) {
			onConnect?.();
		}
	}
</script>

<div
	class="integration-card {className}"
	class:is-connected={isConnected}
	class:is-connecting={isConnecting}
	class:is-error={status === 'error'}
>
	<div class="card-content">
		<div class="icon-wrapper">
			{@html icon}
		</div>

		<div class="info">
			<h4 class="name">{name}</h4>
			{#if description}
				<p class="description">{description}</p>
			{/if}
		</div>
	</div>

	<button
		type="button"
		class="btn-pill btn-pill-ghost action-btn"
		onclick={handleClick}
		disabled={isConnecting}
	>
		{#if isConnected}
			<CheckIcon size={16} />
			<span>Connected</span>
		{:else if isConnecting}
			<span class="spinner"></span>
			<span>Connecting...</span>
		{:else if status === 'error'}
			<span>Retry</span>
		{:else}
			<span>Connect</span>
		{/if}
	</button>
</div>

<style>
	.integration-card {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 16px;
		border: 1px solid var(--border, #e5e7eb);
		border-radius: 12px;
		background-color: var(--background, #ffffff);
		transition: all 0.2s ease;
		box-shadow: 0 1px 2px rgba(0, 0, 0, 0.04);
	}

	.integration-card:hover {
		border-color: var(--primary, #6366f1);
		box-shadow: 0 4px 12px rgba(99, 102, 241, 0.15);
	}

	.integration-card.is-connected {
		border-color: var(--success, #10b981);
		background-color: rgba(16, 185, 129, 0.05);
		box-shadow: 0 2px 8px rgba(16, 185, 129, 0.12);
	}

	.integration-card.is-connecting {
		border-color: var(--primary, #6366f1);
		animation: pulse-border 1.5s ease-in-out infinite;
	}

	@keyframes pulse-border {
		0%, 100% { 
			border-color: var(--primary, #6366f1);
			box-shadow: 0 0 0 0 rgba(99, 102, 241, 0.2);
		}
		50% { 
			border-color: var(--primary, #6366f1);
			box-shadow: 0 0 0 4px rgba(99, 102, 241, 0.1);
		}
	}

	.integration-card.is-error {
		border-color: var(--error, #ef4444);
	}

	.card-content {
		display: flex;
		align-items: center;
		gap: 12px;
	}

	.icon-wrapper {
		width: 40px;
		height: 40px;
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
	}

	.icon-wrapper :global(svg),
	.icon-wrapper :global(img) {
		width: 32px;
		height: 32px;
		object-fit: contain;
	}

	.info {
		display: flex;
		flex-direction: column;
		gap: 2px;
	}

	.name {
		font-size: 14px;
		font-weight: 500;
		color: var(--foreground, #1f2937);
		margin: 0;
	}

	.description {
		font-size: 12px;
		color: var(--muted-foreground, #6b7280);
		margin: 0;
	}

	.action-btn {
		display: flex;
		align-items: center;
		gap: 6px;
		padding: 8px 16px;
		font-size: 13px;
		font-weight: 500;
		border-radius: 8px;
		border: none;
		cursor: pointer;
		transition: all 0.2s ease;
	}

	.integration-card:not(.is-connected) .action-btn {
		background-color: var(--primary, #000000);
		color: var(--primary-foreground, #ffffff);
	}

	.integration-card:not(.is-connected) .action-btn:hover:not(:disabled) {
		opacity: 0.9;
	}

	.integration-card.is-connected .action-btn {
		background-color: var(--success, #10b981);
		color: white;
	}

	.integration-card.is-error .action-btn {
		background-color: var(--error, #ef4444);
		color: white;
	}

	.action-btn:disabled {
		opacity: 0.7;
		cursor: not-allowed;
	}

	.spinner {
		width: 14px;
		height: 14px;
		border: 2px solid transparent;
		border-top-color: currentColor;
		border-radius: 50%;
		animation: spin 0.8s linear infinite;
	}

	@keyframes spin {
		to {
			transform: rotate(360deg);
		}
	}

	/* Dark mode */
	:global(.dark) .integration-card {
		background-color: var(--background, #0a0a0a);
		border-color: var(--border, #2a2a2a);
	}

	:global(.dark) .integration-card:hover {
		border-color: var(--primary, #ffffff);
	}

	:global(.dark) .integration-card.is-connected {
		background-color: rgba(16, 185, 129, 0.1);
	}

	:global(.dark) .name {
		color: var(--foreground, #f9fafb);
	}

	:global(.dark) .integration-card:not(.is-connected) .action-btn {
		background-color: var(--primary, #ffffff);
		color: var(--primary-foreground, #000000);
	}
</style>
