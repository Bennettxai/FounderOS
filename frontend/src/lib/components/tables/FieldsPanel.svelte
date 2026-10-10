<script lang="ts">
	/**
	 * FieldsPanel - Show/hide columns panel
	 * NocoDB-style fields manager with visibility toggles and reorder
	 */
	import {
		X,
		Eye,
		EyeOff,
		GripVertical,
		Search,
		Type,
		Hash,
		Calendar,
		CheckSquare,
		CircleDot,
		Link,
		Mail,
		Paperclip,
		User,
		DollarSign,
		Percent,
		Star,
		Timer,
		Phone,
		Calculator,
		Sigma,
		Link2,
		QrCode,
		Barcode,
		MousePointer,
		Braces,
		Clock,
		AlignLeft
	} from 'lucide-svelte';
	import type { Column, ColumnType } from '$lib/api/tables/types';
	import type { ComponentType, SvelteComponent } from 'svelte';

	type IconComponent = ComponentType<SvelteComponent>;

	interface Props {
		open: boolean;
		columns: Column[];
		hiddenColumns: string[];
		onClose: () => void;
		onToggleColumn: (columnId: string) => void;
		onShowAll: () => void;
		onHideAll: () => void;
		onReorderColumns?: (columnIds: string[]) => void;
	}

	let {
		open,
		columns,
		hiddenColumns,
		onClose,
		onToggleColumn,
		onShowAll,
		onHideAll,
		onReorderColumns
	}: Props = $props();

	let searchQuery = $state('');

	// Get icon for column type
	function getColumnIcon(type: ColumnType): IconComponent {
		const icons: Record<ColumnType, IconComponent> = {
			text: Type as unknown as IconComponent,
			long_text: AlignLeft as unknown as IconComponent,
			number: Hash as unknown as IconComponent,
			single_select: CircleDot as unknown as IconComponent,
			multi_select: CheckSquare as unknown as IconComponent,
			date: Calendar as unknown as IconComponent,
			datetime: Clock as unknown as IconComponent,
			checkbox: CheckSquare as unknown as IconComponent,
			url: Link as unknown as IconComponent,
			email: Mail as unknown as IconComponent,
			attachment: Paperclip as unknown as IconComponent,
			user: User as unknown as IconComponent,
			currency: DollarSign as unknown as IconComponent,
			percent: Percent as unknown as IconComponent,
			rating: Star as unknown as IconComponent,
			duration: Timer as unknown as IconComponent,
			phone: Phone as unknown as IconComponent,
			lookup: Calculator as unknown as IconComponent,
			rollup: Calculator as unknown as IconComponent,
			formula: Sigma as unknown as IconComponent,
			link_to_record: Link2 as unknown as IconComponent,
			qr_code: QrCode as unknown as IconComponent,
			barcode: Barcode as unknown as IconComponent,
			button: MousePointer as unknown as IconComponent,
			json: Braces as unknown as IconComponent
		};
		return icons[type] || (Type as unknown as IconComponent);
	}

	// Filter columns by search
	const filteredColumns = $derived(
		columns.filter((c) => c.name.toLowerCase().includes(searchQuery.toLowerCase()))
	);

	// Count visible/hidden
	const visibleCount = $derived(columns.length - hiddenColumns.length);
	const hiddenCount = $derived(hiddenColumns.length);

	function isHidden(columnId: string): boolean {
		return hiddenColumns.includes(columnId);
	}

	// Handle escape key
	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') {
			onClose();
		}
	}
</script>

<svelte:window on:keydown={handleKeydown} />

{#if open}
	<!-- Backdrop -->
	<div
		class="fixed inset-0 z-40 bg-black/50"
		onclick={onClose}
		role="button"
		tabindex="-1"
		aria-label="Close panel"
	></div>

	<!-- Panel (positioned from right) -->
	<div
		class="dt2-panel fixed right-0 top-0 z-50 flex h-full w-80 flex-col"
		role="dialog"
		aria-modal="true"
		aria-labelledby="fields-panel-title"
	>
		<!-- Header -->
		<div class="flex items-center justify-between px-4 py-4" style="border-bottom: 1px solid var(--dbd);">
			<h2 id="fields-panel-title" class="text-lg font-semibold" style="color: var(--dt);">Fields</h2>
			<button
				type="button"
				class="btn-pill btn-pill-ghost btn-pill-icon"
				onclick={onClose}
			>
				<X class="h-5 w-5" />
			</button>
		</div>

		<!-- Search -->
		<div class="px-4 py-3" style="border-bottom: 1px solid var(--dbd2);">
			<div class="relative">
				<Search class="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2" style="color: var(--dt3);" />
				<input
					type="text"
					placeholder="Search fields..."
					bind:value={searchQuery}
					class="dt2-search w-full rounded-lg py-2 pl-9 pr-3 text-sm focus:outline-none"
				/>
			</div>
		</div>

		<!-- Quick Actions -->
		<div class="flex items-center justify-between px-4 py-2" style="border-bottom: 1px solid var(--dbd2);">
			<div class="flex items-center gap-4 text-xs" style="color: var(--dt3);">
				<span class="flex items-center gap-1">
					<Eye class="h-3.5 w-3.5" />
					{visibleCount} visible
				</span>
				<span class="flex items-center gap-1">
					<EyeOff class="h-3.5 w-3.5" />
					{hiddenCount} hidden
				</span>
			</div>
			<div class="flex items-center gap-2">
				<button
					type="button"
					class="btn-pill btn-pill-ghost btn-pill-xs"
					onclick={onShowAll}
				>
					Show all
				</button>
				<span class="text-gray-300">|</span>
				<button
					type="button"
					class="btn-pill btn-pill-ghost btn-pill-xs"
					onclick={onHideAll}
				>
					Hide all
				</button>
			</div>
		</div>

		<!-- Column List -->
		<div class="flex-1 overflow-y-auto">
			<div class="divide-y divide-gray-100">
				{#each filteredColumns as column (column.id)}
					{@const hidden = isHidden(column.id)}
					{@const ColumnIcon = getColumnIcon(column.type)}

					<div
						class="group flex items-center gap-3 px-4 py-3 hover:bg-gray-50 {hidden
							? 'opacity-50'
							: ''}"
					>
						<!-- Drag Handle (for future reorder) -->
						{#if onReorderColumns}
							<div class="cursor-grab text-gray-300 group-hover:text-gray-400">
								<GripVertical class="h-4 w-4" />
							</div>
						{/if}

						<!-- Column Icon -->
						<svelte:component this={ColumnIcon} class="h-4 w-4 text-gray-400" />

						<!-- Column Name -->
						<span class="flex-1 truncate text-sm text-gray-700">
							{column.name}
							{#if column.is_primary}
								<span class="ml-1 text-xs text-blue-600">(Primary)</span>
							{/if}
						</span>

						<!-- Visibility Toggle -->
						<button
							type="button"
							class="btn-pill btn-pill-icon {hidden ? 'btn-pill-ghost' : 'btn-pill-soft'}"
							onclick={() => onToggleColumn(column.id)}
							title={hidden ? 'Show field' : 'Hide field'}
							disabled={column.is_primary}
						>
							{#if hidden}
								<EyeOff class="h-4 w-4" />
							{:else}
								<Eye class="h-4 w-4" />
							{/if}
						</button>
					</div>
				{/each}
			</div>

			{#if filteredColumns.length === 0}
				<div class="py-8 text-center text-sm text-gray-500">
					No fields match "{searchQuery}"
				</div>
			{/if}
		</div>

		<!-- Footer -->
		<div class="px-4 py-3" style="border-top: 1px solid var(--dbd); background: var(--dbg2);">
			<p class="text-xs" style="color: var(--dt3);">
				Tip: Primary field cannot be hidden. Drag to reorder fields in the view.
			</p>
		</div>
	</div>
{/if}

<style>
	.dt2-panel {
		background: var(--dbg);
		box-shadow: var(--shadow-2xl, 0 25px 50px -12px rgba(0, 0, 0, 0.25));
		color: var(--dt);
	}

	.dt2-search {
		background: var(--dbg2);
		border: 1px solid var(--dbd);
		color: var(--dt);
	}

	.dt2-search:focus {
		background: var(--dbg);
		border-color: var(--dt2-accent, #3b82f6);
		box-shadow: 0 0 0 1px var(--dt2-accent, #3b82f6);
	}

	/* Slide-in animation */
	@keyframes slideInRight {
		from {
			transform: translateX(100%);
		}
		to {
			transform: translateX(0);
		}
	}

	div[role='dialog'] {
		animation: slideInRight 0.2s ease-out;
	}
</style>
