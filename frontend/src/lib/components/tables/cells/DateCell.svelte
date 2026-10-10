<script lang="ts">
	/**
	 * DateCell - Date/DateTime input cell
	 */
	import type { ColumnType, ColumnOptions } from '$lib/api/tables/types';
	import { Calendar } from 'lucide-svelte';

	interface Props {
		value: unknown;
		options?: ColumnOptions;
		editing: boolean;
		type: ColumnType;
		onChange: (value: unknown) => void;
		onBlur: () => void;
	}

	let { value, options, editing, type, onChange, onBlur }: Props = $props();

	let inputRef = $state<HTMLInputElement | null>(null);

	const isDateTime = $derived(type === 'datetime' || options?.include_time);
	const dateFormat = $derived(options?.date_format ?? 'MMM d, yyyy');
	const timeFormat = $derived(options?.time_format ?? '12h');

	// Parse value to Date
	const dateValue = $derived.by(() => {
		if (!value) return null;
		const date = value instanceof Date ? value : new Date(String(value));
		return isNaN(date.getTime()) ? null : date;
	});

	// Format for input
	const inputValue = $derived.by(() => {
		if (!dateValue) return '';
		if (isDateTime) {
			return dateValue.toISOString().slice(0, 16); // yyyy-MM-ddTHH:mm
		}
		return dateValue.toISOString().slice(0, 10); // yyyy-MM-dd
	});

	// Format for display
	const displayValue = $derived.by(() => {
		if (!dateValue) return '';

		const dateOptions: Intl.DateTimeFormatOptions = {
			month: 'short',
			day: 'numeric',
			year: 'numeric'
		};

		if (isDateTime) {
			dateOptions.hour = 'numeric';
			dateOptions.minute = '2-digit';
			dateOptions.hour12 = timeFormat === '12h';
		}

		return dateValue.toLocaleDateString('en-US', dateOptions);
	});

	$effect(() => {
		if (editing && inputRef) {
			inputRef.focus();
			inputRef.showPicker?.();
		}
	});

	function handleChange(e: Event) {
		const target = e.target as HTMLInputElement;
		if (target.value) {
			onChange(new Date(target.value).toISOString());
		} else {
			onChange(null);
		}
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter') {
			onBlur();
		}
	}
</script>

{#if editing}
	<input
		bind:this={inputRef}
		type={isDateTime ? 'datetime-local' : 'date'}
		value={inputValue}
		oninput={handleChange}
		onblur={onBlur}
		onkeydown={handleKeydown}
		class="h-full w-full bg-transparent text-sm outline-none"
	/>
{:else}
	<div class="flex items-center gap-1 text-sm">
		{#if displayValue}
			<Calendar class="h-3 w-3 text-gray-400" />
			<span class="text-gray-900">{displayValue}</span>
		{:else}
			<span class="text-gray-300">-</span>
		{/if}
	</div>
{/if}
