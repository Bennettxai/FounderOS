<script lang="ts">
	/**
	 * FilterModal - Add/Edit filter dialog
	 * Allows selecting column, operator, and value with type-aware inputs
	 */
	import { X } from 'lucide-svelte';
	import type { Filter, Column, ColumnType, FilterOperator } from '$lib/api/tables/types';

	interface Props {
		open: boolean;
		columns: Column[];
		editFilter?: Filter | null;
		onClose: () => void;
		onSave: (filter: Omit<Filter, 'id'> & { id?: string }) => void;
	}

	let { open, columns, editFilter = null, onClose, onSave }: Props = $props();

	// Form state
	let columnId = $state('');
	let operator = $state<FilterOperator>('eq');
	let value = $state<unknown>('');
	let logicalOp = $state<'and' | 'or'>('and');
	let error = $state('');

	// Get selected column
	const selectedColumn = $derived(columns.find((c) => c.id === columnId));

	// Operators available per column type
	const operatorsByType: Record<string, FilterOperator[]> = {
		text: ['eq', 'neq', 'contains', 'not_contains', 'starts_with', 'ends_with', 'is_empty', 'is_not_empty'],
		long_text: ['eq', 'neq', 'contains', 'not_contains', 'is_empty', 'is_not_empty'],
		number: ['eq', 'neq', 'gt', 'gte', 'lt', 'lte', 'is_empty', 'is_not_empty'],
		currency: ['eq', 'neq', 'gt', 'gte', 'lt', 'lte', 'is_empty', 'is_not_empty'],
		percent: ['eq', 'neq', 'gt', 'gte', 'lt', 'lte', 'is_empty', 'is_not_empty'],
		single_select: ['eq', 'neq', 'is_empty', 'is_not_empty'],
		multi_select: ['contains', 'not_contains', 'is_empty', 'is_not_empty'],
		date: ['eq', 'neq', 'is_before', 'is_after', 'is_on_or_before', 'is_on_or_after', 'is_within', 'is_empty', 'is_not_empty'],
		datetime: ['eq', 'neq', 'is_before', 'is_after', 'is_on_or_before', 'is_on_or_after', 'is_within', 'is_empty', 'is_not_empty'],
		checkbox: ['eq', 'neq'],
		url: ['eq', 'neq', 'contains', 'is_empty', 'is_not_empty'],
		email: ['eq', 'neq', 'contains', 'is_empty', 'is_not_empty'],
		phone: ['eq', 'neq', 'contains', 'is_empty', 'is_not_empty'],
		user: ['eq', 'neq', 'is_empty', 'is_not_empty'],
		rating: ['eq', 'neq', 'gt', 'gte', 'lt', 'lte', 'is_empty', 'is_not_empty']
	};

	// Get operators for selected column type
	const availableOperators = $derived.by(() => {
		if (!selectedColumn) return [];
		return operatorsByType[selectedColumn.type] || operatorsByType.text;
	});

	// Operator labels
	const operatorLabels: Record<FilterOperator, string> = {
		eq: 'is',
		neq: 'is not',
		gt: 'greater than',
		gte: 'greater than or equal',
		lt: 'less than',
		lte: 'less than or equal',
		contains: 'contains',
		not_contains: 'does not contain',
		starts_with: 'starts with',
		ends_with: 'ends with',
		is_empty: 'is empty',
		is_not_empty: 'is not empty',
		is_null: 'is null',
		is_not_null: 'is not null',
		in: 'is any of',
		not_in: 'is none of',
		is_within: 'is within',
		is_before: 'is before',
		is_after: 'is after',
		is_on_or_before: 'is on or before',
		is_on_or_after: 'is on or after'
	};

	// Check if operator needs value input
	const needsValue = $derived(!['is_empty', 'is_not_empty', 'is_null', 'is_not_null'].includes(operator));

	// Reset form when modal opens
	$effect(() => {
		if (open) {
			if (editFilter) {
				columnId = editFilter.column_id;
				operator = editFilter.operator;
				value = editFilter.value ?? '';
				logicalOp = editFilter.logical_op;
			} else {
				resetForm();
			}
		}
	});

	function resetForm() {
		columnId = columns[0]?.id ?? '';
		operator = 'eq';
		value = '';
		logicalOp = 'and';
		error = '';
	}

	function handleColumnChange(newColumnId: string) {
		columnId = newColumnId;
		// Reset operator if not available for new column type
		const col = columns.find((c) => c.id === newColumnId);
		if (col) {
			const ops = operatorsByType[col.type] || operatorsByType.text;
			if (!ops.includes(operator)) {
				operator = ops[0];
			}
		}
		value = '';
	}

	function handleSubmit(e: Event) {
		e.preventDefault();

		if (!columnId) {
			error = 'Select a column';
			return;
		}

		if (needsValue && (value === '' || value === null || value === undefined)) {
			error = 'Enter a value';
			return;
		}

		error = '';

		onSave({
			id: editFilter?.id,
			column_id: columnId,
			operator,
			value: needsValue ? value : null,
			logical_op: logicalOp
		});

		onClose();
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') {
			onClose();
		}
	}

	// Get input type for value based on column type
	function getInputType(colType?: ColumnType): string {
		if (!colType) return 'text';
		switch (colType) {
			case 'number':
			case 'currency':
			case 'percent':
			case 'rating':
				return 'number';
			case 'date':
				return 'date';
			case 'datetime':
				return 'datetime-local';
			case 'checkbox':
				return 'checkbox';
			default:
				return 'text';
		}
	}
</script>

{#if open}
	<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
		role="dialog"
		aria-modal="true"
		aria-labelledby="filter-modal-title"
		onkeydown={handleKeydown}
	>
		<!-- Backdrop -->
		<button
			type="button"
			class="btn-pill btn-pill-ghost absolute inset-0 cursor-default"
			onclick={onClose}
			aria-label="Close modal"
		></button>

		<!-- Modal -->
		<div class="relative w-full max-w-md rounded-xl shadow-2xl" style="background: var(--dbg);">
			<!-- Header -->
			<div class="flex items-center justify-between px-6 py-4" style="border-bottom: 1px solid var(--dbd);">
				<h2 id="filter-modal-title" class="text-lg font-semibold" style="color: var(--dt);">
					{editFilter ? 'Edit Filter' : 'Add Filter'}
				</h2>
				<button
					type="button"
					onclick={onClose}
					class="btn-pill btn-pill-ghost btn-pill-icon"
				>
					<X class="h-5 w-5" />
				</button>
			</div>

			<!-- Form -->
			<form onsubmit={handleSubmit} class="p-6">
				{#if error}
					<div class="mb-4 rounded-lg p-3 text-sm" style="background: var(--bos-background-error-color); color: var(--bos-status-error-text);">
						{error}
					</div>
				{/if}

				<!-- Column Selection -->
				<div class="mb-4">
					<label for="filter-column" class="mb-1.5 block text-sm font-medium" style="color: var(--dt2);">
						Column
					</label>
					<select
						id="filter-column"
						value={columnId}
						onchange={(e) => handleColumnChange(e.currentTarget.value)}
						class="w-full rounded-lg px-3 py-2 text-sm focus:outline-none"
						style="border: 1px solid var(--dbd); background: var(--dbg2); color: var(--dt);"
					>
						<option value="" disabled>Select a column</option>
						{#each columns as col}
							<option value={col.id}>{col.name}</option>
						{/each}
					</select>
				</div>

				<!-- Operator Selection -->
				<div class="mb-4">
					<label for="filter-operator" class="mb-1.5 block text-sm font-medium" style="color: var(--dt2);">
						Condition
					</label>
					<select
						id="filter-operator"
						bind:value={operator}
						class="w-full rounded-lg px-3 py-2 text-sm focus:outline-none"
						style="border: 1px solid var(--dbd); background: var(--dbg2); color: var(--dt);"
					>
						{#each availableOperators as op}
							<option value={op}>{operatorLabels[op]}</option>
						{/each}
					</select>
				</div>

				<!-- Value Input -->
				{#if needsValue}
					<div class="mb-4">
						<label for="filter-value" class="mb-1.5 block text-sm font-medium" style="color: var(--dt2);">
							Value
						</label>

						{#if selectedColumn?.type === 'single_select' && selectedColumn.options?.choices}
							<!-- Select dropdown for single_select columns -->
							<select
								id="filter-value"
								bind:value
								class="w-full rounded-lg px-3 py-2 text-sm focus:outline-none"
								style="border: 1px solid var(--dbd); background: var(--dbg2); color: var(--dt);"
							>
								<option value="">Select an option</option>
								{#each selectedColumn.options.choices as choice}
									<option value={choice.id}>{choice.label}</option>
								{/each}
							</select>
						{:else if selectedColumn?.type === 'checkbox'}
							<!-- Checkbox toggle -->
							<div class="flex items-center gap-3">
								<label class="flex items-center gap-2">
									<input
										type="radio"
										name="checkbox-value"
										checked={value === true}
										onchange={() => (value = true)}
										class="h-4 w-4"
										style="border-color: var(--dbd); accent-color: var(--bos-brand-color);"
									/>
									<span class="text-sm" style="color: var(--dt2);">Checked</span>
								</label>
								<label class="flex items-center gap-2">
									<input
										type="radio"
										name="checkbox-value"
										checked={value === false}
										onchange={() => (value = false)}
										class="h-4 w-4"
										style="border-color: var(--dbd); accent-color: var(--bos-brand-color);"
									/>
									<span class="text-sm" style="color: var(--dt2);">Unchecked</span>
								</label>
							</div>
						{:else}
							<!-- Standard input -->
							<input
								id="filter-value"
								type={getInputType(selectedColumn?.type)}
								bind:value
								placeholder="Enter value..."
								class="w-full rounded-lg px-3 py-2 text-sm focus:outline-none"
								style="border: 1px solid var(--dbd); background: var(--dbg2); color: var(--dt);"
							/>
						{/if}
					</div>
				{/if}

				<!-- Logical Operator (for additional filters) -->
				{#if editFilter}
					<div class="mb-6">
						<label class="mb-1.5 block text-sm font-medium" style="color: var(--dt2);">
							Combine with
						</label>
						<div class="flex gap-4">
							<label class="flex items-center gap-2">
								<input
									type="radio"
									name="logical-op"
									value="and"
									bind:group={logicalOp}
									class="h-4 w-4"
									style="accent-color: var(--bos-brand-color);"
								/>
								<span class="text-sm" style="color: var(--dt2);">AND</span>
							</label>
							<label class="flex items-center gap-2">
								<input
									type="radio"
									name="logical-op"
									value="or"
									bind:group={logicalOp}
									class="h-4 w-4"
									style="accent-color: var(--bos-brand-color);"
								/>
								<span class="text-sm" style="color: var(--dt2);">OR</span>
							</label>
						</div>
					</div>
				{/if}

				<!-- Actions -->
				<div class="flex justify-end gap-3">
					<button
						type="button"
						onclick={onClose}
						class="btn-pill btn-pill-ghost btn-pill-sm"
					>
						Cancel
					</button>
					<button
						type="submit"
						class="btn-cta"
					>
						{editFilter ? 'Update Filter' : 'Add Filter'}
					</button>
				</div>
			</form>
		</div>
	</div>
{/if}
