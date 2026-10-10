<script lang="ts">
	/**
	 * URLCell - Display and edit URLs with clickable link
	 */
	import { ExternalLink } from 'lucide-svelte';

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

	// Validate URL and add protocol if missing
	function formatUrl(url: string): string {
		if (!url) return '';
		if (url.match(/^https?:\/\//i)) return url;
		return `https://${url}`;
	}

	// Get display text (domain only for cleaner look)
	function getDisplayText(url: string): string {
		if (!url) return '';
		try {
			const formatted = formatUrl(url);
			const urlObj = new URL(formatted);
			return urlObj.hostname.replace(/^www\./, '');
		} catch {
			return url;
		}
	}

	const displayValue = $derived(String(value ?? ''));
	const isValidUrl = $derived(displayValue.length > 0);
</script>

{#if editing}
	<input
		bind:this={inputRef}
		type="url"
		bind:value={localValue}
		onkeydown={handleKeydown}
		onblur={handleBlur}
		placeholder="https://example.com"
		class="h-full w-full bg-transparent text-sm focus:outline-none"
	/>
{:else if displayValue}
	<a
		href={formatUrl(displayValue)}
		target="_blank"
		rel="noopener noreferrer"
		onclick={(e) => e.stopPropagation()}
		class="inline-flex items-center gap-1 text-sm text-blue-600 hover:text-blue-700 hover:underline"
	>
		<span class="truncate">{getDisplayText(displayValue)}</span>
		<ExternalLink class="h-3 w-3 flex-shrink-0" />
	</a>
{:else}
	<span class="text-sm text-gray-400">-</span>
{/if}
