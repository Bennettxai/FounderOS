<script lang="ts">
	/**
	 * AddTableModal - Create new table dialog
	 */
	import { X, Table2, FileSpreadsheet, Database, Upload } from 'lucide-svelte';
	import type { CreateTableData, TableSource } from '$lib/api/tables/types';

	interface Props {
		open: boolean;
		onClose: () => void;
		onCreate: (data: CreateTableData) => void;
	}

	let { open, onClose, onCreate }: Props = $props();

	let name = $state('');
	let description = $state('');
	let source = $state<TableSource>('custom');
	let loading = $state(false);
	let error = $state('');

	const sourceOptions: { value: TableSource; label: string; icon: typeof Table2; description: string }[] = [
		{
			value: 'custom',
			label: 'Blank Table',
			icon: Table2,
			description: 'Start with an empty table'
		},
		{
			value: 'import',
			label: 'Import CSV/Excel',
			icon: Upload,
			description: 'Import data from a file'
		},
		{
			value: 'integration',
			label: 'From Integration',
			icon: Database,
			description: 'Sync from connected app'
		}
	];

	function resetForm() {
		name = '';
		description = '';
		source = 'custom';
		error = '';
	}

	function handleClose() {
		resetForm();
		onClose();
	}

	async function handleSubmit(e: Event) {
		e.preventDefault();

		if (!name.trim()) {
			error = 'Table name is required';
			return;
		}

		loading = true;
		error = '';

		try {
			await onCreate({
				name: name.trim(),
				description: description.trim() || undefined,
				source,
				// Default columns for a new table
				columns: [
					{
						name: 'Name',
						type: 'text',
						is_primary: true
					}
				]
			});
			handleClose();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to create table';
		} finally {
			loading = false;
		}
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') {
			handleClose();
		}
	}
</script>

{#if open}
	<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
	<div
		class="bos-modal-overlay"
		role="dialog"
		aria-modal="true"
		aria-labelledby="modal-title"
		onkeydown={handleKeydown}
	>
		<!-- Backdrop -->
		<button
			type="button"
			class="absolute inset-0 cursor-default"
			onclick={handleClose}
			aria-label="Close modal"
		></button>

		<!-- Modal -->
		<form onsubmit={handleSubmit} class="bos-modal bos-modal--md">
			<!-- Header -->
			<div class="bos-modal-header">
				<h2 id="modal-title" class="bos-modal-title">Create New Table</h2>
				<button
					type="button"
					onclick={handleClose}
					class="bos-modal-close"
					aria-label="Close modal"
				>
					<X class="h-4 w-4" />
				</button>
			</div>

			<!-- Body -->
			<div class="bos-modal-body">
				{#if error}
					<div class="bos-error-banner mb-4">
						{error}
					</div>
				{/if}

				<!-- Source Selection -->
				<div class="mb-6">
					<label class="bos-label mb-2">Start from</label>
					<div class="grid grid-cols-3 gap-3">
						{#each sourceOptions as option}
							<button
								type="button"
								class="flex flex-col items-center p-4 text-center rounded-lg border-2 transition-colors"
								style={source === option.value
									? 'border-color: var(--bos-primary-color); background: color-mix(in srgb, var(--bos-primary-color) 10%, var(--bos-modal-bg));'
									: 'border-color: var(--bos-border-color); background: transparent;'}
								onclick={() => (source = option.value)}
							>
								<option.icon
									class="mb-2 h-6 w-6"
									style={source === option.value
										? 'color: var(--bos-primary-color);'
										: 'color: var(--bos-text-secondary-color);'}
								/>
								<span
									class="text-sm font-medium"
									style={source === option.value
										? 'color: var(--bos-primary-color);'
										: 'color: var(--bos-text-primary-color);'}
								>
									{option.label}
								</span>
							</button>
						{/each}
					</div>
				</div>

				<!-- Name -->
				<div class="mb-4">
					<label for="table-name" class="bos-label bos-label--required mb-1.5">
						Table Name
					</label>
					<input
						id="table-name"
						type="text"
						bind:value={name}
						placeholder="e.g., Customer Database"
						class="bos-input w-full"
						required
					/>
				</div>

				<!-- Description -->
				<div class="mb-0">
					<label for="table-description" class="bos-label mb-1.5">
						Description
					</label>
					<textarea
						id="table-description"
						bind:value={description}
						placeholder="What is this table for?"
						rows="2"
						class="bos-textarea w-full"
					></textarea>
				</div>
			</div>

			<!-- Footer -->
			<div class="bos-modal-footer">
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
						Creating...
					{:else}
						Create Table
					{/if}
				</button>
			</div>
		</form>
	</div>
{/if}
