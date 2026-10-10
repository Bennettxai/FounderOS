<script lang="ts">
	import { fly, fade } from 'svelte/transition';

	interface Props {
		currentStep: number;
		totalSteps: number;
		showBack?: boolean;
		showSkip?: boolean;
		continueText?: string;
		continueDisabled?: boolean;
		onBack?: () => void;
		onContinue?: () => void;
		onSkip?: () => void;
		children: any;
	}

	let {
		currentStep,
		totalSteps,
		showBack = true,
		showSkip = true,
		continueText = 'Continue',
		continueDisabled = false,
		onBack,
		onContinue,
		onSkip,
		children
	}: Props = $props();

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter' && !continueDisabled && onContinue) {
			onContinue();
		} else if (e.key === 'Escape' && showBack && onBack) {
			onBack();
		}
	}
</script>

<svelte:window on:keydown={handleKeydown} />

<div class="min-h-screen bg-white flex flex-col">
	<!-- Header -->
	<header class="px-6 py-4 flex items-center justify-between border-b border-gray-100">
		<div class="flex items-center gap-2">
			<div class="w-8 h-8 bg-gray-900 rounded-lg flex items-center justify-center">
				<span class="text-white text-sm font-bold">B</span>
			</div>
			<span class="text-gray-900 font-semibold hidden sm:block">Business OS</span>
		</div>

		<div class="flex items-center gap-4">
			<span class="text-sm text-gray-500">Step {currentStep} of {totalSteps}</span>
			{#if showSkip && onSkip}
				<button
					type="button"
					onclick={onSkip}
					class="btn-pill-sm"
				>
					Skip
				</button>
			{/if}
		</div>
	</header>

	<!-- Progress Bar -->
	<div class="px-6 py-4">
		<div class="max-w-md mx-auto">
			<div class="flex items-center gap-2">
				{#each Array(totalSteps) as _, i}
					<div class="flex-1 flex items-center">
						<div
							class="w-3 h-3 rounded-full transition-all duration-300 {i + 1 < currentStep
								? 'bg-gray-900'
								: i + 1 === currentStep
									? 'bg-gray-900 ring-4 ring-gray-200'
									: 'bg-gray-200'}"
						></div>
						{#if i < totalSteps - 1}
							<div
								class="flex-1 h-0.5 mx-1 transition-all duration-300 {i + 1 < currentStep
									? 'bg-gray-900'
									: 'bg-gray-200'}"
							></div>
						{/if}
					</div>
				{/each}
			</div>
		</div>
	</div>

	<!-- Content -->
	<main class="flex-1 flex flex-col items-center justify-center px-6 py-8">
		<div class="w-full max-w-lg">
			{@render children()}
		</div>
	</main>

	<!-- Navigation -->
	<footer class="px-6 py-6 border-t border-gray-100">
		<div class="max-w-lg mx-auto flex items-center justify-between">
			{#if showBack && currentStep > 1 && onBack}
				<button
					type="button"
					onclick={onBack}
					class="btn-pill flex items-center gap-2"
					in:fade={{ duration: 200 }}
				>
					<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 19l-7-7m0 0l7-7m-7 7h18" />
					</svg>
					Back
				</button>
			{:else}
				<div></div>
			{/if}

			{#if onContinue}
				<button
					type="button"
					onclick={onContinue}
					disabled={continueDisabled}
					class="btn-pill btn-pill-primary flex items-center gap-2"
				>
					{continueText}
					<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14 5l7 7m0 0l-7 7m7-7H3" />
					</svg>
				</button>
			{/if}
		</div>
	</footer>
</div>
