<script lang="ts">
	/**
	 * TextCell - Text input cell
	 */
	import type { ColumnType, ColumnOptions } from '$lib/api/tables/types';
	import { ExternalLink, Mail, Phone } from 'lucide-svelte';

	interface Props {
		value: unknown;
		options?: ColumnOptions;
		editing: boolean;
		type: ColumnType;
		onChange: (value: unknown) => void;
		onBlur: () => void;
	}

	let { value, options, editing, type, onChange, onBlur }: Props = $props();

	let inputRef = $state<HTMLInputElement | HTMLTextAreaElement | null>(null);

	const stringValue = $derived(value != null ? String(value) : '');
	const isLongText = $derived(type === 'long_text');
	const isUrl = $derived(type === 'url');
	const isEmail = $derived(type === 'email');
	const isPhone = $derived(type === 'phone');

	$effect(() => {
		if (editing && inputRef) {
			inputRef.focus();
			inputRef.select();
		}
	});

	function handleChange(e: Event) {
		const target = e.target as HTMLInputElement | HTMLTextAreaElement;
		onChange(target.value);
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter' && !isLongText) {
			onBlur();
		}
	}
</script>

{#if editing}
	{#if isLongText}
		<textarea
			bind:this={inputRef}
			value={stringValue}
			oninput={handleChange}
			onblur={onBlur}
			class="h-full min-h-[60px] w-full resize-none bg-transparent text-sm outline-none"
			rows="2"
		></textarea>
	{:else}
		<input
			bind:this={inputRef}
			type={isEmail ? 'email' : isUrl ? 'url' : isPhone ? 'tel' : 'text'}
			value={stringValue}
			oninput={handleChange}
			onblur={onBlur}
			onkeydown={handleKeydown}
			class="h-full w-full bg-transparent text-sm outline-none"
		/>
	{/if}
{:else}
	<div class="flex items-center gap-1 text-sm">
		{#if stringValue}
			{#if isUrl}
				<a
					href={stringValue.startsWith('http') ? stringValue : `https://${stringValue}`}
					target="_blank"
					rel="noopener noreferrer"
					class="flex items-center gap-1 text-blue-600 hover:underline"
					onclick={(e) => e.stopPropagation()}
				>
					<span class="truncate">{stringValue}</span>
					<ExternalLink class="h-3 w-3 flex-shrink-0" />
				</a>
			{:else if isEmail}
				<a
					href="mailto:{stringValue}"
					class="flex items-center gap-1 text-blue-600 hover:underline"
					onclick={(e) => e.stopPropagation()}
				>
					<Mail class="h-3 w-3 flex-shrink-0 text-gray-400" />
					<span class="truncate">{stringValue}</span>
				</a>
			{:else if isPhone}
				<a
					href="tel:{stringValue}"
					class="flex items-center gap-1 text-blue-600 hover:underline"
					onclick={(e) => e.stopPropagation()}
				>
					<Phone class="h-3 w-3 flex-shrink-0 text-gray-400" />
					<span class="truncate">{stringValue}</span>
				</a>
			{:else}
				<span class="truncate text-gray-900">{stringValue}</span>
			{/if}
		{:else}
			<span class="text-gray-300">-</span>
		{/if}
	</div>
{/if}
