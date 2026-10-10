<script lang="ts">
	/**
	 * AddColumnModal - Create/Edit column dialog
	 * Allows selecting column type, configuring options, and setting properties
	 */
	import { X, Plus, Trash2 } from 'lucide-svelte';
	import type { ColumnType, CreateColumnData, ColumnOptions, SelectChoice } from '$lib/api/tables/types';
	import ColumnTypeSelector from './ColumnTypeSelector.svelte';

	interface Props {
		open: boolean;
		onClose: () => void;
		onCreate: (data: CreateColumnData) => void;
		editColumn?: {
			id: string;
			name: string;
			type: ColumnType;
			is_required: boolean;
			is_unique: boolean;
			options?: ColumnOptions;
		} | null;
	}

	let { open, onClose, onCreate, editColumn = null }: Props = $props();

	// Form state
	let name = $state('');
	let type = $state<ColumnType>('text');
	let isRequired = $state(false);
	let isUnique = $state(false);
	let options = $state<ColumnOptions>({});
	let loading = $state(false);
	let error = $state('');
	let showAdvancedTypes = $state(false);

	// Select options state
	let selectChoices = $state<SelectChoice[]>([]);
	let newChoiceLabel = $state('');

	// Color palette for select options
	const colorPalette = [
		'var(--bos-status-error)', 'var(--bos-status-warning)', 'var(--bos-priority-medium)',
		'var(--bos-status-success)', 'var(--bos-status-info)', 'var(--bos-nav-active)',
		'var(--bos-category-ai)', 'var(--bos-category-automation)', 'var(--bos-category-communication)',
		'var(--bos-status-neutral)'
	];

	// Reset form when modal opens/closes or editColumn changes
	$effect(() => {
		if (open) {
			if (editColumn) {
				name = editColumn.name;
				type = editColumn.type;
				isRequired = editColumn.is_required;
				isUnique = editColumn.is_unique;
				options = editColumn.options || {};
				selectChoices = editColumn.options?.choices || [];
			} else {
				resetForm();
			}
		}
	});

	function resetForm() {
		name = '';
		type = 'text';
		isRequired = false;
		isUnique = false;
		options = {};
		selectChoices = [];
		newChoiceLabel = '';
		error = '';
		showAdvancedTypes = false;
	}

	function handleClose() {
		resetForm();
		onClose();
	}

	function handleTypeChange(newType: ColumnType) {
		type = newType;
		// Reset type-specific options when type changes
		options = {};
		selectChoices = [];
	}

	function addSelectChoice() {
		if (!newChoiceLabel.trim()) return;

		const newChoice: SelectChoice = {
			id: crypto.randomUUID(),
			label: newChoiceLabel.trim(),
			color: colorPalette[selectChoices.length % colorPalette.length],
			order: selectChoices.length
		};

		selectChoices = [...selectChoices, newChoice];
		newChoiceLabel = '';
	}

	function removeSelectChoice(id: string) {
		selectChoices = selectChoices.filter((c) => c.id !== id);
	}

	function updateChoiceColor(id: string, color: string) {
		selectChoices = selectChoices.map((c) => (c.id === id ? { ...c, color } : c));
	}

	async function handleSubmit(e: Event) {
		e.preventDefault();

		if (!name.trim()) {
			error = 'Column name is required';
			return;
		}

		if ((type === 'single_select' || type === 'multi_select') && selectChoices.length === 0) {
			error = 'Add at least one option for select fields';
			return;
		}

		loading = true;
		error = '';

		try {
			const columnData: CreateColumnData = {
				name: name.trim(),
				type,
				is_required: isRequired,
				is_unique: isUnique,
				options: buildOptions()
			};

			await onCreate(columnData);
			handleClose();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to create column';
		} finally {
			loading = false;
		}
	}

	function buildOptions(): ColumnOptions {
		const opts: ColumnOptions = { ...options };

		// Add select choices
		if (type === 'single_select' || type === 'multi_select') {
			opts.choices = selectChoices;
		}

		return Object.keys(opts).length > 0 ? opts : undefined as unknown as ColumnOptions;
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') {
			handleClose();
		}
	}

	// Check if type needs options UI
	const needsSelectOptions = $derived(type === 'single_select' || type === 'multi_select');
	const needsNumberOptions = $derived(type === 'number' || type === 'currency' || type === 'percent');
	const needsRatingOptions = $derived(type === 'rating');
</script>

{#if open}
	<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
		role="dialog"
		aria-modal="true"
		aria-labelledby="modal-title"
		onkeydown={handleKeydown}
	>
		<!-- Backdrop -->
		<button
			type="button"
			class="btn-pill btn-pill-ghost absolute inset-0 cursor-default"
			onclick={handleClose}
			aria-label="Close modal"
		></button>

		<!-- Modal -->
		<div class="relative w-full max-w-2xl rounded-xl shadow-2xl" style="background: var(--dbg)">
			<!-- Header -->
			<div class="flex items-center justify-between border-b px-6 py-4" style="border-color: var(--dbd)">
				<h2 id="modal-title" class="text-lg font-semibold" style="color: var(--dt)">
					{editColumn ? 'Edit Column' : 'Add Column'}
				</h2>
				<button
					type="button"
					onclick={handleClose}
					class="btn-pill btn-pill-ghost btn-pill-icon"
				>
					<X class="h-5 w-5" />
				</button>
			</div>

			<!-- Form -->
			<form onsubmit={handleSubmit} class="max-h-[calc(100vh-200px)] overflow-y-auto p-6">
				{#if error}
					<div class="mb-4 rounded-lg p-3 text-sm" style="background: var(--bos-status-error-bg); color: var(--bos-status-error-text)">
						{error}
					</div>
				{/if}

				<!-- Column Name -->
				<div class="mb-6">
					<label for="column-name" class="mb-1.5 block text-sm font-medium" style="color: var(--dt2)">
						Column Name <span style="color: var(--bos-status-error)">*</span>
					</label>
					<input
						id="column-name"
						type="text"
						bind:value={name}
						placeholder="e.g., Status, Email, Due Date"
						class="w-full rounded-lg border px-3 py-2 text-sm focus:outline-none"
						style="border-color: var(--dbd); background: var(--dbg2); color: var(--dt)"
						required
					/>
				</div>

				<!-- Column Type -->
				<div class="mb-6">
					<label class="mb-2 block text-sm font-medium" style="color: var(--dt2)">Column Type</label>
					<ColumnTypeSelector
						selectedType={type}
						onSelect={handleTypeChange}
						showAdvanced={showAdvancedTypes}
					/>
					{#if !showAdvancedTypes}
						<button
							type="button"
							class="btn-pill btn-pill-ghost btn-pill-sm mt-3"
							onclick={() => (showAdvancedTypes = true)}
						>
							Show advanced types...
						</button>
					{/if}
				</div>

				<!-- Type-Specific Options -->
				{#if needsSelectOptions}
					<div class="mb-6 rounded-lg border p-4" style="border-color: var(--dbd); background: var(--dbg2)">
						<h4 class="mb-3 text-sm font-medium" style="color: var(--dt2)">Options</h4>

						<!-- Existing choices -->
						{#if selectChoices.length > 0}
							<div class="mb-3 space-y-2">
								{#each selectChoices as choice (choice.id)}
									<div class="flex items-center gap-2">
										<input
											type="color"
											value={choice.color}
											onchange={(e) => updateChoiceColor(choice.id, e.currentTarget.value)}
											class="h-8 w-8 cursor-pointer rounded border-0"
										/>
										<span
											class="flex-1 rounded-full px-3 py-1 text-sm"
											style="background-color: {choice.color}; color: var(--bos-surface-on-color)"
										>
											{choice.label}
										</span>
										<button
											type="button"
											onclick={() => removeSelectChoice(choice.id)}
											class="btn-pill btn-pill-ghost btn-pill-icon"
										>
											<Trash2 class="h-4 w-4" />
										</button>
									</div>
								{/each}
							</div>
						{/if}

						<!-- Add new choice -->
						<div class="flex gap-2">
							<input
								type="text"
								bind:value={newChoiceLabel}
								placeholder="Add an option..."
								class="flex-1 rounded-lg border px-3 py-2 text-sm focus:outline-none"
								style="border-color: var(--dbd); background: var(--dbg); color: var(--dt)"
								onkeydown={(e) => e.key === 'Enter' && (e.preventDefault(), addSelectChoice())}
							/>
							<button
								type="button"
								onclick={addSelectChoice}
								disabled={!newChoiceLabel.trim()}
								class="btn-pill btn-pill-soft btn-pill-sm flex items-center gap-1"
							>
								<Plus class="h-4 w-4" />
								Add
							</button>
						</div>
					</div>
				{/if}

				{#if needsNumberOptions}
					<div class="mb-6 rounded-lg border p-4" style="border-color: var(--dbd); background: var(--dbg2)">
						<h4 class="mb-3 text-sm font-medium" style="color: var(--dt2)">Number Options</h4>
						<div class="grid grid-cols-2 gap-4">
							<div>
								<label for="precision" class="mb-1 block text-xs" style="color: var(--dt3)">Decimal Places</label>
								<select
									id="precision"
									bind:value={options.precision}
									class="w-full rounded-lg border px-3 py-2 text-sm focus:outline-none"
									style="border-color: var(--dbd); background: var(--dbg); color: var(--dt)"
								>
									<option value={0}>0 (Integer)</option>
									<option value={1}>1</option>
									<option value={2}>2</option>
									<option value={3}>3</option>
									<option value={4}>4</option>
								</select>
							</div>
							{#if type === 'currency'}
								<div>
									<label for="currency" class="mb-1 block text-xs" style="color: var(--dt3)">Currency</label>
									<select
										id="currency"
										bind:value={options.currency_code}
										class="w-full rounded-lg border px-3 py-2 text-sm focus:outline-none"
										style="border-color: var(--dbd); background: var(--dbg); color: var(--dt)"
									>
										<option value="USD">USD ($)</option>
										<option value="EUR">EUR</option>
										<option value="GBP">GBP</option>
										<option value="JPY">JPY</option>
										<option value="CAD">CAD</option>
										<option value="AUD">AUD</option>
									</select>
								</div>
							{/if}
						</div>
					</div>
				{/if}

				{#if needsRatingOptions}
					<div class="mb-6 rounded-lg border p-4" style="border-color: var(--dbd); background: var(--dbg2)">
						<h4 class="mb-3 text-sm font-medium" style="color: var(--dt2)">Rating Options</h4>
						<div class="grid grid-cols-2 gap-4">
							<div>
								<label for="max-rating" class="mb-1 block text-xs" style="color: var(--dt3)">Max Rating</label>
								<select
									id="max-rating"
									bind:value={options.rating_max}
									class="w-full rounded-lg border px-3 py-2 text-sm focus:outline-none"
									style="border-color: var(--dbd); background: var(--dbg); color: var(--dt)"
								>
									<option value={5}>5 Stars</option>
									<option value={10}>10 Stars</option>
								</select>
							</div>
							<div>
								<label for="rating-icon" class="mb-1 block text-xs" style="color: var(--dt3)">Icon</label>
								<select
									id="rating-icon"
									bind:value={options.rating_icon}
									class="w-full rounded-lg border px-3 py-2 text-sm focus:outline-none"
									style="border-color: var(--dbd); background: var(--dbg); color: var(--dt)"
								>
									<option value="star">Star</option>
									<option value="heart">Heart</option>
									<option value="thumb">Thumbs Up</option>
								</select>
							</div>
						</div>
					</div>
				{/if}

				<!-- Column Properties -->
				<div class="mb-6">
					<h4 class="mb-3 text-sm font-medium" style="color: var(--dt2)">Properties</h4>
					<div class="space-y-3">
						<label class="flex items-center gap-3">
							<input
								type="checkbox"
								bind:checked={isRequired}
								class="h-4 w-4 rounded"
								style="accent-color: var(--bos-nav-active); border-color: var(--dbd)"
							/>
							<div>
								<span class="text-sm font-medium" style="color: var(--dt2)">Required</span>
								<p class="text-xs" style="color: var(--dt3)">This field must have a value</p>
							</div>
						</label>
						<label class="flex items-center gap-3">
							<input
								type="checkbox"
								bind:checked={isUnique}
								class="h-4 w-4 rounded"
								style="accent-color: var(--bos-nav-active); border-color: var(--dbd)"
							/>
							<div>
								<span class="text-sm font-medium" style="color: var(--dt2)">Unique</span>
								<p class="text-xs" style="color: var(--dt3)">No duplicate values allowed</p>
							</div>
						</label>
					</div>
				</div>

				<!-- Actions -->
				<div class="flex justify-end gap-3 border-t pt-4" style="border-color: var(--dbd)">
					<button
						type="button"
						onclick={handleClose}
						class="btn-pill btn-pill-ghost btn-pill-sm"
					>
						Cancel
					</button>
					<button
						type="submit"
						disabled={loading || !name.trim()}
						class="btn-cta"
					>
						{#if loading}
							{editColumn ? 'Saving...' : 'Creating...'}
						{:else}
							{editColumn ? 'Save Changes' : 'Add Column'}
						{/if}
					</button>
				</div>
			</form>
		</div>
	</div>
{/if}
