<script lang="ts">
	import { PillButton } from '$lib/components/osa';
	import { onboardingStore } from '$lib/stores/onboardingStore';
	import { goto } from '$app/navigation';

	interface Props {
		show: boolean;
	}

	let { show = $bindable() }: Props = $props();

	const currentStep = $derived($onboardingStore.currentStep);

	function handleContinue() {
		show = false;
		// Navigate to the current step
		const routes = ['', 'quick-info', 'username', 'connect', 'building', 'ready'];
		if (currentStep < routes.length) {
			goto(`/onboarding/${routes[currentStep]}`);
		}
	}

	function handleStartFresh() {
		onboardingStore.reset();
		show = false;
		goto('/onboarding');
	}
</script>

{#if show}
	<div class="modal-overlay" onclick={() => (show = false)}>
		<div class="modal-content" onclick={(e) => e.stopPropagation()}>
			<div class="modal-header">
				<span class="wave">👋</span>
				<h2 class="modal-title">Welcome back!</h2>
			</div>

			<p class="modal-message">
				You were setting up your workspace.<br />
				Pick up where you left off?
			</p>

			<div class="modal-actions">
				<PillButton variant="primary" size="md" onclick={handleContinue}>
					Continue Setup
				</PillButton>
				<button class="btn-pill btn-pill-ghost start-fresh-btn" onclick={handleStartFresh}>
					Start Fresh
				</button>
			</div>
		</div>
	</div>
{/if}

<style>
	.modal-overlay {
		position: fixed;
		top: 0;
		left: 0;
		right: 0;
		bottom: 0;
		background: rgba(0, 0, 0, 0.5);
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 1000;
		animation: fadeIn 0.2s ease-out;
	}

	.modal-content {
		background: white;
		border-radius: 16px;
		padding: 2rem;
		max-width: 400px;
		width: 90%;
		box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
		animation: slideUp 0.3s ease-out;
		display: flex;
		flex-direction: column;
		gap: 1.5rem;
		text-align: center;
	}

	.modal-header {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 0.75rem;
	}

	.wave {
		font-size: 3rem;
		animation: wave 2s ease-in-out infinite;
	}

	@keyframes wave {
		0%, 100% { transform: rotate(0deg); }
		10%, 30% { transform: rotate(14deg); }
		20%, 40% { transform: rotate(-8deg); }
		50% { transform: rotate(0deg); }
	}

	.modal-title {
		font-size: 1.5rem;
		font-weight: 700;
		color: #1A1A1A;
		margin: 0;
	}

	.modal-message {
		font-size: 1rem;
		color: #666666;
		line-height: 1.6;
		margin: 0;
	}

	.modal-actions {
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
		margin-top: 0.5rem;
	}

	.start-fresh-btn {
		background: transparent;
		border: none;
		color: #666666;
		font-size: 0.875rem;
		font-weight: 500;
		cursor: pointer;
		padding: 0.5rem;
		font-family: inherit;
		transition: color 0.2s ease;
	}

	.start-fresh-btn:hover {
		color: #1A1A1A;
	}

	@keyframes fadeIn {
		from { opacity: 0; }
		to { opacity: 1; }
	}

	@keyframes slideUp {
		from {
			opacity: 0;
			transform: translateY(20px);
		}
		to {
			opacity: 1;
			transform: translateY(0);
		}
	}
</style>
