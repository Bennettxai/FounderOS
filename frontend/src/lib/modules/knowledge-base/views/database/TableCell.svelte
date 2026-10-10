<script lang="ts">
	/**
	 * Table Cell - Database cell renderer and editor
	 * Renders different UI based on column type
	 */
	import { Check, ExternalLink, Calendar, User, File, X } from 'lucide-svelte';
	import type { CellValue, ColumnSchema, SelectOption, TextDelta } from '../../entities/block';

	interface Props {
		column: ColumnSchema;
		value: CellValue | null;
		isEditing?: boolean;
		onUpdate?: (value: CellValue) => void;
		onStartEdit?: () => void;
		onEndEdit?: () => void;
	}

	let {
		column,
		value,
		isEditing = false,
		onUpdate,
		onStartEdit,
		onEndEdit
	}: Props = $props();

	// Get display value based on type
	function getDisplayValue(): string {
		if (!value) return '';

		switch (value.type) {
			case 'title':
				return value.value.map((d: TextDelta) => d.insert).join('');
			case 'text':
			case 'url':
			case 'email':
			case 'phone':
				return value.value;
			case 'number':
				return value.value?.toString() ?? '';
			case 'select':
				return getSelectLabel(value.value);
			case 'multi-select':
				return value.value.map(getSelectLabel).join(', ');
			case 'date':
				return value.value ? formatDate(value.value.start) : '';
			case 'checkbox':
				return '';
			case 'created-time':
			case 'updated-time':
				return formatDate(value.value);
			case 'created-by':
			case 'updated-by':
				return value.value;
			default:
				return '';
		}
	}

	function getSelectLabel(optionId: string | null): string {
		if (!optionId) return '';
		const data = column.data as { options?: SelectOption[] } | undefined;
		const option = data?.options?.find((o) => o.id === optionId);
		return option?.value ?? '';
	}

	function getSelectOption(optionId: string | null): SelectOption | null {
		if (!optionId) return null;
		const data = column.data as { options?: SelectOption[] } | undefined;
		return data?.options?.find((o) => o.id === optionId) ?? null;
	}

	function getSelectOptions(): SelectOption[] {
		const data = column.data as { options?: SelectOption[] } | undefined;
		return data?.options ?? [];
	}

	function formatDate(dateStr: string): string {
		try {
			return new Date(dateStr).toLocaleDateString();
		} catch {
			return dateStr;
		}
	}

	// Input handling
	let inputValue = $state('');

	$effect(() => {
		if (isEditing) {
			inputValue = getDisplayValue();
		}
	});

	function handleInputChange(e: Event) {
		const target = e.target as HTMLInputElement;
		inputValue = target.value;
	}

	function handleInputBlur() {
		commitValue();
		onEndEdit?.();
	}

	function handleInputKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter') {
			commitValue();
			onEndEdit?.();
		} else if (e.key === 'Escape') {
			onEndEdit?.();
		}
	}

	function commitValue() {
		if (!onUpdate) return;

		switch (column.type) {
			case 'title':
				onUpdate({ type: 'title', value: [{ insert: inputValue }] });
				break;
			case 'text':
				onUpdate({ type: 'text', value: inputValue });
				break;
			case 'number':
				const num = parseFloat(inputValue);
				onUpdate({ type: 'number', value: isNaN(num) ? null : num });
				break;
			case 'url':
				onUpdate({ type: 'url', value: inputValue });
				break;
			case 'email':
				onUpdate({ type: 'email', value: inputValue });
				break;
			case 'phone':
				onUpdate({ type: 'phone', value: inputValue });
				break;
		}
	}

	function handleCheckboxToggle() {
		if (!onUpdate) return;
		const current = value?.type === 'checkbox' ? value.value : false;
		onUpdate({ type: 'checkbox', value: !current });
	}

	function handleSelectChange(optionId: string) {
		if (!onUpdate) return;
		onUpdate({ type: 'select', value: optionId });
		onEndEdit?.();
	}

	function handleMultiSelectToggle(optionId: string) {
		if (!onUpdate) return;
		const current = value?.type === 'multi-select' ? value.value : [];
		const newValue = current.includes(optionId)
			? current.filter((id: string) => id !== optionId)
			: [...current, optionId];
		onUpdate({ type: 'multi-select', value: newValue });
	}

	// Derived state
	const displayValue = $derived(getDisplayValue());
	const isCheckbox = $derived(column.type === 'checkbox');
	const isSelect = $derived(column.type === 'select');
	const isMultiSelect = $derived(column.type === 'multi-select');
	const isUrl = $derived(column.type === 'url');
	const isReadOnly = $derived(['created-time', 'created-by', 'updated-time', 'updated-by', 'formula', 'rollup'].includes(column.type));
</script>

<div
	class="bos-table-cell"
	class:bos-table-cell--editing={isEditing}
	class:bos-table-cell--readonly={isReadOnly}
	onclick={() => !isReadOnly && !isEditing && onStartEdit?.()}
	onkeydown={(e) => e.key === 'Enter' && !isReadOnly && !isEditing && onStartEdit?.()}
	role="gridcell"
	tabindex={0}
>
	{#if isEditing && !isReadOnly}
		{#if isSelect}
			<!-- Select dropdown -->
			<div class="bos-table-cell__select-list">
				{#each getSelectOptions() as option}
					<button
						class="bos-table-cell__select-option"
						class:bos-table-cell__select-option--selected={value?.type === 'select' && value.value === option.id}
						style:--option-color={option.color}
						onclick={() => handleSelectChange(option.id)}
					>
						<span class="bos-table-cell__select-badge" style:background={option.color}>
							{option.value}
						</span>
					</button>
				{/each}
			</div>
		{:else if isMultiSelect}
			<!-- Multi-select list -->
			<div class="bos-table-cell__select-list">
				{#each getSelectOptions() as option}
					{@const isSelected = value?.type === 'multi-select' && value.value.includes(option.id)}
					<button
						class="bos-table-cell__select-option"
						class:bos-table-cell__select-option--selected={isSelected}
						onclick={() => handleMultiSelectToggle(option.id)}
					>
						<span class="bos-table-cell__checkbox" class:bos-table-cell__checkbox--checked={isSelected}>
							{#if isSelected}<Check />{/if}
						</span>
						<span class="bos-table-cell__select-badge" style:background={option.color}>
							{option.value}
						</span>
					</button>
				{/each}
			</div>
		{:else}
			<!-- Text input -->
			<input
				type={column.type === 'number' ? 'number' : 'text'}
				class="bos-table-cell__input"
				value={inputValue}
				oninput={handleInputChange}
				onblur={handleInputBlur}
				onkeydown={handleInputKeydown}
				autofocus
			/>
		{/if}
	{:else}
		<!-- Display mode -->
		{#if isCheckbox}
			<button
				class="bos-table-cell__checkbox"
				class:bos-table-cell__checkbox--checked={value?.type === 'checkbox' && value.value}
				onclick={handleCheckboxToggle}
			>
				{#if value?.type === 'checkbox' && value.value}
					<Check />
				{/if}
			</button>
		{:else if isSelect && value?.type === 'select' && value.value}
			{@const option = getSelectOption(value.value)}
			{#if option}
				<span class="bos-table-cell__select-badge" style:background={option.color}>
					{option.value}
				</span>
			{/if}
		{:else if isMultiSelect && value?.type === 'multi-select' && value.value.length > 0}
			<div class="bos-table-cell__tags">
				{#each value.value as optionId}
					{@const option = getSelectOption(optionId)}
					{#if option}
						<span class="bos-table-cell__select-badge" style:background={option.color}>
							{option.value}
						</span>
					{/if}
				{/each}
			</div>
		{:else if isUrl && displayValue}
			<a href={displayValue} target="_blank" rel="noopener noreferrer" class="bos-table-cell__link">
				{displayValue}
				<ExternalLink />
			</a>
		{:else}
			<span class="bos-table-cell__text">{displayValue}</span>
		{/if}
	{/if}
</div>

<style>
	.bos-table-cell {
		display: flex;
		align-items: center;
		min-height: 32px;
		padding: 4px 8px;
		cursor: pointer;
		overflow: hidden;
	}

	.bos-table-cell--readonly {
		cursor: default;
		color: var(--dt2);
	}

	.bos-table-cell--editing {
		padding: 0;
	}

	.bos-table-cell__input {
		width: 100%;
		height: 32px;
		padding: 4px 8px;
		border: none;
		background: transparent;
		font-size: 14px;
		color: var(--dt);
		outline: none;
	}

	.bos-table-cell__input:focus {
		background: var(--dbg);
		box-shadow: inset 0 0 0 2px #1e96eb;
	}

	.bos-table-cell__text {
		font-size: 14px;
		color: var(--dt);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.bos-table-cell__checkbox {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 16px;
		height: 16px;
		border: 1.5px solid var(--dbd);
		border-radius: 4px;
		background: transparent;
		cursor: pointer;
		transition: all 0.15s;
	}

	.bos-table-cell__checkbox--checked {
		background: #1e96eb;
		border-color: #1e96eb;
		color: white;
	}

	.bos-table-cell__checkbox :global(svg) {
		width: 12px;
		height: 12px;
	}

	.bos-table-cell__select-badge {
		display: inline-flex;
		align-items: center;
		padding: 2px 8px;
		border-radius: 4px;
		font-size: 12px;
		font-weight: 500;
		color: white;
		white-space: nowrap;
	}

	.bos-table-cell__tags {
		display: flex;
		flex-wrap: wrap;
		gap: 4px;
	}

	.bos-table-cell__link {
		display: inline-flex;
		align-items: center;
		gap: 4px;
		font-size: 14px;
		color: #1e96eb;
		text-decoration: none;
	}

	.bos-table-cell__link:hover {
		text-decoration: underline;
	}

	.bos-table-cell__link :global(svg) {
		width: 12px;
		height: 12px;
	}

	.bos-table-cell__select-list {
		display: flex;
		flex-direction: column;
		gap: 2px;
		padding: 4px;
		background: var(--dbg);
		border: 1px solid var(--dbd);
		border-radius: 8px;
		box-shadow: 0 4px 12px rgba(0, 0, 0, 0.12);
		max-height: 200px;
		overflow-y: auto;
	}

	.bos-table-cell__select-option {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 6px 8px;
		border: none;
		background: transparent;
		border-radius: 4px;
		cursor: pointer;
		text-align: left;
	}

	.bos-table-cell__select-option:hover {
		background: var(--dbg3);
	}

	.bos-table-cell__select-option--selected {
		background: var(--dbg3);
	}
</style>
