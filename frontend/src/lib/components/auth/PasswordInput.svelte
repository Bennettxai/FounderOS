<script lang="ts">
	import type { HTMLInputAttributes } from 'svelte/elements';

	interface Props {
		id: string;
		label: string;
		value: string;
		placeholder?: string;
		error?: string;
		autocomplete?: HTMLInputAttributes['autocomplete'];
		required?: boolean;
		showStrength?: boolean;
		maxlength?: number;
	}

	let {
		id,
		label,
		value = $bindable(),
		placeholder = '',
		error = '',
		autocomplete = 'current-password',
		required = false,
		showStrength = false,
		maxlength = undefined
	}: Props = $props();

	let showPassword = $state(false);

	function getStrength(password: string): { score: number; label: string; color: string } {
		let score = 0;
		if (password.length >= 8) score++;
		if (password.length >= 12) score++;
		if (/[a-z]/.test(password)) score++;
		if (/[A-Z]/.test(password)) score++;
		if (/[0-9]/.test(password)) score++;
		if (/[^a-zA-Z0-9]/.test(password)) score++;

		if (score <= 2) return { score, label: 'Weak', color: 'bg-red-500' };
		if (score <= 4) return { score, label: 'Medium', color: 'bg-yellow-500' };
		if (score <= 5) return { score, label: 'Strong', color: 'bg-green-500' };
		return { score, label: 'Very strong', color: 'bg-green-600' };
	}

	const strength = $derived(showStrength && value ? getStrength(value) : null);
</script>

<div class="space-y-1.5">
	<label for={id} class="block text-sm font-medium text-gray-700">
		{label}
	</label>
	<div class="relative">
		<input
			{id}
			name={id}
			type={showPassword ? 'text' : 'password'}
			bind:value
			{placeholder}
			{autocomplete}
			{required}
			maxlength={maxlength}
			class="input input-square w-full pr-10 {error ? 'border-red-500 focus:ring-red-500' : ''}"
		/>
		<button
			type="button"
			onclick={() => showPassword = !showPassword}
			class="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600 transition-colors"
			tabindex={-1}
		>
			{#if showPassword}
				<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21" />
				</svg>
			{:else}
				<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
				</svg>
			{/if}
		</button>
	</div>

	{#if showStrength && value && strength}
		<div class="space-y-1">
			<div class="h-1.5 bg-gray-200 rounded-full overflow-hidden">
				<div
					class="h-full transition-all duration-300 {strength.color}"
					style="width: {(strength.score / 6) * 100}%"
				></div>
			</div>
			<p class="text-xs {strength.score <= 2 ? 'text-red-600' : strength.score <= 4 ? 'text-yellow-600' : 'text-green-600'}">
				{strength.label}
			</p>
		</div>
	{/if}

	{#if error}
		<p class="text-sm text-red-600 flex items-center gap-1">
			<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
			</svg>
			{error}
		</p>
	{/if}
</div>
