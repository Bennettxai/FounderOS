<script lang="ts">
	/**
	 * CellRenderer - Dynamic cell component based on column type
	 */
	import type { ColumnType, ColumnOptions } from '$lib/api/tables/types';
	import TextCell from './TextCell.svelte';
	import NumberCell from './NumberCell.svelte';
	import CheckboxCell from './CheckboxCell.svelte';
	import SelectCell from './SelectCell.svelte';
	import DateCell from './DateCell.svelte';
	import URLCell from './URLCell.svelte';
	import EmailCell from './EmailCell.svelte';
	import RatingCell from './RatingCell.svelte';

	interface Props {
		type: ColumnType;
		value: unknown;
		options?: ColumnOptions;
		editing: boolean;
		expanded?: boolean; // For row expand modal - larger display
		onChange: (value: unknown) => void;
		onBlur: () => void;
	}

	let { type, value, options, editing, expanded = false, onChange, onBlur }: Props = $props();

	// Map column types to cell components
	const cellComponentMap: Record<string, typeof TextCell> = {
		text: TextCell,
		long_text: TextCell,
		url: URLCell,
		email: EmailCell,
		phone: TextCell,
		number: NumberCell,
		currency: NumberCell,
		percent: NumberCell,
		rating: RatingCell,
		duration: NumberCell,
		checkbox: CheckboxCell,
		single_select: SelectCell,
		multi_select: SelectCell,
		date: DateCell,
		datetime: DateCell
	};

	const CellComponent = $derived(cellComponentMap[type] || TextCell);
</script>

<svelte:component
	this={CellComponent}
	{value}
	{options}
	{editing}
	{type}
	{onChange}
	{onBlur}
/>
