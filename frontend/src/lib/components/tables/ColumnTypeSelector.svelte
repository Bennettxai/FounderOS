<script lang="ts">
	/**
	 * ColumnTypeSelector - Grid of column types to choose from
	 */
	import {
		Type,
		AlignLeft,
		Hash,
		CircleDot,
		CheckSquare,
		Calendar,
		Clock,
		Link,
		Mail,
		Paperclip,
		User,
		DollarSign,
		Percent,
		Star,
		Timer,
		Phone,
		Search,
		Calculator,
		Sigma,
		Link2,
		QrCode,
		Barcode,
		MousePointer,
		Braces
	} from 'lucide-svelte';
	import type { ColumnType } from '$lib/api/tables/types';
	import { COLUMN_TYPES } from '$lib/api/tables/types';

	interface Props {
		selectedType: ColumnType | null;
		onSelect: (type: ColumnType) => void;
		showAdvanced?: boolean;
	}

	let { selectedType, onSelect, showAdvanced = false }: Props = $props();

	const iconMap: Record<string, typeof Type> = {
		Type,
		AlignLeft,
		Hash,
		CircleDot,
		CheckSquare,
		Calendar,
		Clock,
		Link,
		Mail,
		Paperclip,
		User,
		DollarSign,
		Percent,
		Star,
		Timer,
		Phone,
		Search,
		Calculator,
		Sigma,
		Link2,
		QrCode,
		Barcode,
		MousePointer,
		Braces
	};

	function getIcon(iconName: string) {
		return iconMap[iconName] || Type;
	}

	const basicTypes = COLUMN_TYPES.filter((t) => t.category === 'basic');
	const advancedTypes = COLUMN_TYPES.filter((t) => t.category === 'advanced');
	const computedTypes = COLUMN_TYPES.filter((t) => t.category === 'computed');
	const specialTypes = COLUMN_TYPES.filter((t) => t.category === 'special');
</script>

<div class="space-y-4">
	<!-- Basic Types -->
	<div>
		<h4 class="mb-2 text-xs font-medium uppercase text-gray-400">Basic</h4>
		<div class="grid grid-cols-3 gap-2">
			{#each basicTypes as colType}
				<button
					type="button"
					class="btn-pill btn-pill-sm flex items-center gap-2 border {selectedType ===
					colType.type
						? 'btn-pill-soft border-blue-500'
						: 'btn-pill-secondary border-gray-200'}"
					onclick={() => onSelect(colType.type)}
				>
					<svelte:component
						this={getIcon(colType.icon)}
						class="h-4 w-4 {selectedType === colType.type ? 'text-blue-600' : 'text-gray-400'}"
					/>
					{colType.label}
				</button>
			{/each}
		</div>
	</div>

	{#if showAdvanced}
		<!-- Advanced Types -->
		<div>
			<h4 class="mb-2 text-xs font-medium uppercase text-gray-400">Advanced</h4>
			<div class="grid grid-cols-3 gap-2">
				{#each advancedTypes as colType}
					<button
						type="button"
						class="btn-pill btn-pill-sm flex items-center gap-2 border {selectedType ===
						colType.type
							? 'btn-pill-soft border-blue-500'
							: 'btn-pill-secondary border-gray-200'}"
						onclick={() => onSelect(colType.type)}
					>
						<svelte:component
							this={getIcon(colType.icon)}
							class="h-4 w-4 {selectedType === colType.type ? 'text-blue-600' : 'text-gray-400'}"
						/>
						{colType.label}
					</button>
				{/each}
			</div>
		</div>

		<!-- Computed Types -->
		<div>
			<h4 class="mb-2 text-xs font-medium uppercase text-gray-400">Computed</h4>
			<div class="grid grid-cols-3 gap-2">
				{#each computedTypes as colType}
					<button
						type="button"
						class="btn-pill btn-pill-sm flex items-center gap-2 border {selectedType ===
						colType.type
							? 'btn-pill-soft border-blue-500'
							: 'btn-pill-secondary border-gray-200'}"
						onclick={() => onSelect(colType.type)}
					>
						<svelte:component
							this={getIcon(colType.icon)}
							class="h-4 w-4 {selectedType === colType.type ? 'text-blue-600' : 'text-gray-400'}"
						/>
						{colType.label}
					</button>
				{/each}
			</div>
		</div>

		<!-- Special Types -->
		<div>
			<h4 class="mb-2 text-xs font-medium uppercase text-gray-400">Special</h4>
			<div class="grid grid-cols-3 gap-2">
				{#each specialTypes as colType}
					<button
						type="button"
						class="btn-pill btn-pill-sm flex items-center gap-2 border {selectedType ===
						colType.type
							? 'btn-pill-soft border-blue-500'
							: 'btn-pill-secondary border-gray-200'}"
						onclick={() => onSelect(colType.type)}
					>
						<svelte:component
							this={getIcon(colType.icon)}
							class="h-4 w-4 {selectedType === colType.type ? 'text-blue-600' : 'text-gray-400'}"
						/>
						{colType.label}
					</button>
				{/each}
			</div>
		</div>
	{/if}
</div>
