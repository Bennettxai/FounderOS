<script lang="ts">
	/**
	 * EmailCell - Display and edit emails with clickable mailto link
	 */
	import { Mail } from 'lucide-svelte';

	interface Props {
		value: unknown;
		editing: boolean;
		onChange: (value: unknown) => void;
		onBlur: () => void;
	}

	let { value, editing, onChange, onBlur }: Props = $props();

	let inputRef = $state<HTMLInputElement | null>(null);
	let localValue = $state(String(value ?? ''));

	// Focus input when editing starts
	$effect(() => {
		if (editing && inputRef) {
			inputRef.focus();
			inputRef.select();
		}
	});

	// Reset local value when value prop changes
	$effect(() => {
		localValue = String(value ?? '');
	});

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter' && !e.shiftKey) {
			e.preventDefault();
			onChange(localValue);
			onBlur();
		} else if (e.key === 'Escape') {
			localValue = String(value ?? '');
			onBlur();
		}
	}

	function handleBlur() {
		onChange(localValue);
		onBlur();
	}

	// Simple email validation
	function isValidEmail(email: string): boolean {
		return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email);
	}

	const displayValue = $derived(String(value ?? ''));
	const isValid = $derived(isValidEmail(displayValue));
</script>

{#if editing}
	<input
		bind:this={inputRef}
		type="email"
		bind:value={localValue}
		onkeydown={handleKeydown}
		onblur={handleBlur}
		placeholder="email@example.com"
		class="h-full w-full bg-transparent text-sm focus:outline-none"
	/>
{:else if displayValue}
	{#if isValid}
		<a
			href="mailto:{displayValue}"
			onclick={(e) => e.stopPropagation()}
			class="inline-flex items-center gap-1.5 text-sm text-blue-600 hover:text-blue-700 hover:underline"
		>
			<Mail class="h-3.5 w-3.5 flex-shrink-0" />
			<span class="truncate">{displayValue}</span>
		</a>
	{:else}
		<span class="text-sm text-gray-700">{displayValue}</span>
	{/if}
{:else}
	<span class="text-sm text-gray-400">-</span>
{/if}
