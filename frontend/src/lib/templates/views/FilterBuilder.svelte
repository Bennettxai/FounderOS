<script lang="ts">
	/**
	 * FilterBuilder - Visual filter builder UI for app templates
	 */

	import type { Field } from '../types/field';
	import type { FilterCondition, FilterGroup, FilterOperator } from '../types/view';
	import { TemplateButton, TemplateSelect, TemplateInput } from '../primitives';
	import FilterBuilder from './FilterBuilder.svelte';

	interface Props {
		fields: Field[];
		filter?: FilterGroup;
		onchange?: (filter: FilterGroup) => void;
		onclose?: () => void;
	}

	let {
		fields,
		filter = $bindable({ id: 'root', operator: 'and', conditions: [] }),
		onchange,
		onclose
	}: Props = $props();

	const operatorLabels: Record<FilterOperator, string> = {
		equals: 'equals',
		not_equals: 'does not equal',
		contains: 'contains',
		not_contains: 'does not contain',
		starts_with: 'starts with',
		ends_with: 'ends with',
		is_empty: 'is empty',
		is_not_empty: 'is not empty',
		greater_than: 'greater than',
		less_than: 'less than',
		greater_or_equal: 'greater or equal',
		less_or_equal: 'less or equal',
		between: 'between',
		in: 'is any of',
		not_in: 'is none of'
	};

	function getOperatorsForField(field: Field): FilterOperator[] {
		const textOps: FilterOperator[] = ['equals', 'not_equals', 'contains', 'not_contains', 'starts_with', 'ends_with', 'is_empty', 'is_not_empty'];
		const numberOps: FilterOperator[] = ['equals', 'not_equals', 'greater_than', 'less_than', 'greater_or_equal', 'less_or_equal', 'between', 'is_empty', 'is_not_empty'];
		const selectOps: FilterOperator[] = ['equals', 'not_equals', 'in', 'not_in', 'is_empty', 'is_not_empty'];

		switch (field.type) {
			case 'number':
			case 'currency':
			case 'rating':
			case 'progress':
				return numberOps;
			case 'date':
			case 'datetime':
				return numberOps;
			case 'select':
			case 'status':
			case 'multiselect':
				return selectOps;
			case 'checkbox':
				return ['equals'];
			default:
				return textOps;
		}
	}

	function generateId(): string {
		return Math.random().toString(36).substring(2, 9);
	}

	function addCondition() {
		const newCondition: FilterCondition = {
			id: generateId(),
			fieldId: fields[0]?.id || '',
			operator: 'equals',
			value: ''
		};
		filter.conditions = [...filter.conditions, newCondition];
		onchange?.(filter);
	}

	function addGroup() {
		const newGroup: FilterGroup = {
			id: generateId(),
			operator: 'and',
			conditions: []
		};
		filter.conditions = [...filter.conditions, newGroup];
		onchange?.(filter);
	}

	function removeCondition(id: string) {
		filter.conditions = filter.conditions.filter(c => c.id !== id);
		onchange?.(filter);
	}

	function updateCondition(id: string, updates: Partial<FilterCondition>) {
		filter.conditions = filter.conditions.map(c => {
			if (c.id === id && 'fieldId' in c) {
				return { ...c, ...updates } as FilterCondition;
			}
			return c;
		});
		onchange?.(filter);
	}

	function isCondition(item: FilterCondition | FilterGroup): item is FilterCondition {
		return 'fieldId' in item;
	}

	function clearAll() {
		filter = { id: filter.id, operator: filter.operator, conditions: [] };
		onchange?.(filter);
	}

	function applyFilter() {
		onchange?.(filter);
		onclose?.();
	}

	function getFieldLabel(field: Field): string {
		return field.label ?? field.name ?? field.id;
	}

	const fieldOptions = $derived(fields.map(f => ({ value: f.id, label: getFieldLabel(f) })));
</script>

<div class="tpl-filter-builder">
	<div class="tpl-filter-header">
		<h3 class="tpl-filter-title">Filter</h3>
		<div class="tpl-filter-header-actions">
			<TemplateButton variant="ghost" size="sm" onclick={clearAll}>Clear all</TemplateButton>
		</div>
	</div>

	<div class="tpl-filter-group">
		<div class="tpl-filter-group-header">
			<span class="tpl-filter-group-label">Show records where</span>
			<select
				class="tpl-filter-operator-select"
				bind:value={filter.operator}
				onchange={() => onchange?.(filter)}
			>
				<option value="and">all</option>
				<option value="or">any</option>
			</select>
			<span class="tpl-filter-group-label">of the following are true:</span>
		</div>

		<div class="tpl-filter-conditions">
			{#each filter.conditions as condition, index}
				{#if isCondition(condition)}
					{@const field = fields.find(f => f.id === condition.fieldId)}
					{@const operators = field ? getOperatorsForField(field) : []}

					<div class="tpl-filter-condition">
						{#if index > 0}
							<span class="tpl-filter-connector">{filter.operator === 'and' ? 'AND' : 'OR'}</span>
						{/if}
						<div class="tpl-filter-condition-row">
							<TemplateSelect
								options={fieldOptions}
								value={condition.fieldId}
								size="sm"
								onchange={(e) => updateCondition(condition.id, { fieldId: (e.target as HTMLSelectElement).value, value: '' })}
							/>
							<TemplateSelect
								options={operators.map(op => ({ value: op, label: operatorLabels[op] }))}
								value={condition.operator}
								size="sm"
								onchange={(e) => updateCondition(condition.id, { operator: (e.target as HTMLSelectElement).value as FilterOperator })}
							/>
								{#if condition.operator !== 'is_empty' && condition.operator !== 'is_not_empty'}
								{#if field?.type === 'select' || field?.type === 'status'}
									<TemplateSelect
										options={(field.config?.options ?? field.options).map((o) => ({ value: o.value, label: o.label || o.value }))}
										value={String(condition.value || '')}
										size="sm"
										onchange={(e) => updateCondition(condition.id, { value: (e.target as HTMLSelectElement).value })}
									/>
								{:else if field?.type === 'checkbox'}
									<TemplateSelect
										options={[{ value: 'true', label: 'Checked' }, { value: 'false', label: 'Unchecked' }]}
										value={String(condition.value || 'true')}
										size="sm"
										onchange={(e) => updateCondition(condition.id, { value: (e.target as HTMLSelectElement).value === 'true' })}
									/>
								{:else if field?.type === 'number' || field?.type === 'currency' || field?.type === 'rating'}
									<TemplateInput
										type="number"
										value={String(condition.value || '')}
										size="sm"
										placeholder="Value"
										onchange={(e) => updateCondition(condition.id, { value: Number((e.target as HTMLInputElement).value) })}
									/>
									{#if condition.operator === 'between'}
										<span class="tpl-filter-between">and</span>
										<TemplateInput
											type="number"
											value={String(condition.value2 || '')}
											size="sm"
											placeholder="Value"
											onchange={(e) => updateCondition(condition.id, { value2: Number((e.target as HTMLInputElement).value) })}
										/>
									{/if}
								{:else if field?.type === 'date' || field?.type === 'datetime'}
									<TemplateInput
										type="text"
										value={String(condition.value || '')}
										size="sm"
										onchange={(e) => updateCondition(condition.id, { value: (e.target as HTMLInputElement).value })}
									/>
									{#if condition.operator === 'between'}
										<span class="tpl-filter-between">and</span>
										<TemplateInput
											type="text"
											value={String(condition.value2 || '')}
											size="sm"
											onchange={(e) => updateCondition(condition.id, { value2: (e.target as HTMLInputElement).value })}
										/>
									{/if}
								{:else}
									<TemplateInput
										type="text"
										value={String(condition.value || '')}
										size="sm"
										placeholder="Value"
										onchange={(e) => updateCondition(condition.id, { value: (e.target as HTMLInputElement).value })}
									/>
								{/if}
							{/if}
							<button class="tpl-filter-remove" aria-label="Remove condition" onclick={() => removeCondition(condition.id)}>
								<svg viewBox="0 0 20 20" fill="currentColor">
									<path fill-rule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clip-rule="evenodd" />
								</svg>
							</button>
						</div>
					</div>
				{:else}
					<!-- Nested group (recursive) -->
					<div class="tpl-filter-nested-group">
						<FilterBuilder
							{fields}
							filter={condition}
							onchange={() => onchange?.(filter)}
						/>
					</div>
				{/if}
			{/each}
		</div>

		<div class="tpl-filter-add-actions">
			<TemplateButton variant="ghost" size="sm" onclick={addCondition}>
				<svg viewBox="0 0 20 20" fill="currentColor" width="16" height="16">
					<path fill-rule="evenodd" d="M10 3a1 1 0 011 1v5h5a1 1 0 110 2h-5v5a1 1 0 11-2 0v-5H4a1 1 0 110-2h5V4a1 1 0 011-1z" clip-rule="evenodd" />
				</svg>
				Add condition
			</TemplateButton>
			<TemplateButton variant="ghost" size="sm" onclick={addGroup}>
				<svg viewBox="0 0 20 20" fill="currentColor" width="16" height="16">
					<path fill-rule="evenodd" d="M3 4a1 1 0 011-1h4a1 1 0 010 2H6.414l2.293 2.293a1 1 0 01-1.414 1.414L5 6.414V8a1 1 0 01-2 0V4zm9 1a1 1 0 110-2h4a1 1 0 011 1v4a1 1 0 11-2 0V6.414l-2.293 2.293a1 1 0 11-1.414-1.414L13.586 5H12zm-9 7a1 1 0 112 0v1.586l2.293-2.293a1 1 0 011.414 1.414L6.414 15H8a1 1 0 110 2H4a1 1 0 01-1-1v-4zm13 1a1 1 0 10-2 0v1.586l-2.293-2.293a1 1 0 00-1.414 1.414L13.586 15H12a1 1 0 100 2h4a1 1 0 001-1v-4z" clip-rule="evenodd" />
				</svg>
				Add group
			</TemplateButton>
		</div>
	</div>

	<div class="tpl-filter-footer">
		{#if onclose}
			<TemplateButton variant="outline" size="sm" onclick={onclose}>Cancel</TemplateButton>
		{/if}
		<TemplateButton variant="primary" size="sm" onclick={applyFilter}>Apply filter</TemplateButton>
	</div>
</div>

<style>
	.tpl-filter-builder {
		padding: var(--tpl-space-4);
		background: var(--tpl-bg-primary);
		border: 1px solid var(--tpl-border-default);
		border-radius: var(--tpl-radius-lg);
		box-shadow: var(--tpl-shadow-lg);
		min-width: 500px;
	}

	.tpl-filter-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-bottom: var(--tpl-space-4);
		padding-bottom: var(--tpl-space-3);
		border-bottom: 1px solid var(--tpl-border-subtle);
	}

	.tpl-filter-title {
		margin: 0;
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-base);
		font-weight: var(--tpl-font-semibold);
		color: var(--tpl-text-primary);
	}

	.tpl-filter-group-header {
		display: flex;
		align-items: center;
		gap: var(--tpl-space-2);
		margin-bottom: var(--tpl-space-3);
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-sm);
		color: var(--tpl-text-secondary);
	}

	.tpl-filter-operator-select {
		padding: var(--tpl-space-1) var(--tpl-space-2);
		background: var(--tpl-bg-secondary);
		border: 1px solid var(--tpl-border-default);
		border-radius: var(--tpl-radius-md);
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-sm);
		font-weight: var(--tpl-font-medium);
		color: var(--tpl-text-primary);
		cursor: pointer;
	}

	.tpl-filter-conditions {
		display: flex;
		flex-direction: column;
		gap: var(--tpl-space-2);
		margin-bottom: var(--tpl-space-3);
	}

	.tpl-filter-condition {
		display: flex;
		flex-direction: column;
		gap: var(--tpl-space-1);
	}

	.tpl-filter-connector {
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-xs);
		font-weight: var(--tpl-font-medium);
		color: var(--tpl-text-muted);
		padding-left: var(--tpl-space-2);
	}

	.tpl-filter-condition-row {
		display: flex;
		align-items: center;
		gap: var(--tpl-space-2);
		flex-wrap: wrap;
	}

	.tpl-filter-condition-row > :global(*) {
		flex: 1;
		min-width: 100px;
	}

	.tpl-filter-between {
		flex: 0 !important;
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-sm);
		color: var(--tpl-text-muted);
	}

	.tpl-filter-remove {
		flex: 0 !important;
		width: 28px;
		height: 28px;
		padding: 0;
		display: flex;
		align-items: center;
		justify-content: center;
		background: transparent;
		border: none;
		border-radius: var(--tpl-radius-md);
		color: var(--tpl-text-muted);
		cursor: pointer;
		transition: all var(--tpl-transition-fast);
	}

	.tpl-filter-remove:hover {
		background: var(--tpl-status-error-bg);
		color: var(--tpl-status-error);
	}

	.tpl-filter-remove svg {
		width: 16px;
		height: 16px;
	}

	.tpl-filter-nested-group {
		margin-left: var(--tpl-space-4);
		padding-left: var(--tpl-space-3);
		border-left: 2px solid var(--tpl-border-default);
	}

	.tpl-filter-add-actions {
		display: flex;
		gap: var(--tpl-space-2);
	}

	.tpl-filter-footer {
		display: flex;
		justify-content: flex-end;
		gap: var(--tpl-space-2);
		margin-top: var(--tpl-space-4);
		padding-top: var(--tpl-space-3);
		border-top: 1px solid var(--tpl-border-subtle);
	}
</style>
