<script lang="ts">
	/**
	 * CardView - Grid card layout for app templates
	 */

	import type { Field } from '../types/field';
	import type { CardViewConfig } from '../types/view';
	import { TemplateCard, TemplateAvatar, TemplateBadge, TemplateSkeleton } from '../primitives';

	interface Props {
		config: CardViewConfig;
		fields: Field[];
		data: Record<string, unknown>[];
		loading?: boolean;
		selectedIds?: Set<string>;
		onselect?: (id: string, selected: boolean) => void;
		onrowclick?: (record: Record<string, unknown>) => void;
	}

	let {
		config,
		fields,
		data,
		loading = false,
		selectedIds = new Set(),
		onselect,
		onrowclick
	}: Props = $props();

	const columnSizes = {
		small: 'repeat(auto-fill, minmax(200px, 1fr))',
		medium: 'repeat(auto-fill, minmax(280px, 1fr))',
		large: 'repeat(auto-fill, minmax(360px, 1fr))'
	};

	function getFieldValue(record: Record<string, unknown>, fieldId: string): unknown {
		return record[fieldId];
	}

	function formatValue(value: unknown, field: Field | undefined): string {
		if (value === null || value === undefined) return '';
		if (!field) return String(value);

		switch (field.type) {
			case 'currency':
				const currencySymbol = ((field as Field & { config?: { symbol?: string } }).config?.symbol) || '$';
				return `${currencySymbol}${Number(value).toLocaleString()}`;
			case 'date':
				return new Date(value as string).toLocaleDateString();
			case 'number':
				return Number(value).toLocaleString();
			default:
				return String(value);
		}
	}

	function getField(fieldId: string): Field | undefined {
		return fields.find(f => f.id === fieldId);
	}

	function getFieldOptions(field: Field | undefined) {
		if (!field || (field.type !== 'status' && field.type !== 'select' && field.type !== 'multiselect')) {
			return [];
		}
		return ((field as Field & { config?: { options?: { value: string; label: string; color?: string }[] } }).config?.options) ?? field.options;
	}

	const gridStyle = $derived(`grid-template-columns: ${config.columns ? `repeat(${config.columns}, 1fr)` : columnSizes[config.cardSize || 'medium']}`);
</script>

<div class="tpl-card-view" style={gridStyle}>
	{#if loading}
		{#each Array(6) as _}
			<div class="tpl-card-skeleton">
				{#if config.imageField}
					<TemplateSkeleton variant="rectangular" height="160px" />
				{/if}
				<div class="tpl-card-skeleton-content">
					<TemplateSkeleton variant="text" width="60%" />
					<TemplateSkeleton variant="text" width="80%" />
					<TemplateSkeleton variant="text" lines={2} />
				</div>
			</div>
		{/each}
	{:else if data.length === 0}
		<div class="tpl-card-view-empty">
			<div class="tpl-card-view-empty-icon">
				<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
					<rect x="3" y="3" width="7" height="7" rx="1" />
					<rect x="14" y="3" width="7" height="7" rx="1" />
					<rect x="3" y="14" width="7" height="7" rx="1" />
					<rect x="14" y="14" width="7" height="7" rx="1" />
				</svg>
			</div>
			<p class="tpl-card-view-empty-text">No items to display</p>
		</div>
	{:else}
		{#each data as record}
			{@const id = String(record.id || record._id || '')}
			{@const isSelected = selectedIds.has(id)}
			{@const title = getFieldValue(record, config.titleField)}
			{@const subtitle = config.subtitleField ? getFieldValue(record, config.subtitleField) : null}
			{@const image = config.imageField ? getFieldValue(record, config.imageField) : null}
			{@const badge = config.badgeField ? getFieldValue(record, config.badgeField) : null}
			{@const description = config.descriptionField ? getFieldValue(record, config.descriptionField) : null}

			<TemplateCard
				variant="default"
				padding="none"
				clickable
				selected={isSelected}
				onclick={() => onrowclick?.(record)}
			>
				{#if image}
					<div class="tpl-card-image">
						<img src={String(image)} alt={String(title)} />
					</div>
				{/if}
				<div class="tpl-card-body">
					<div class="tpl-card-header-row">
						<h3 class="tpl-card-title">{title}</h3>
						{#if badge}
							{@const badgeField = getField(config.badgeField!)}
							{#if badgeField?.type === 'status'}
								{@const option = getFieldOptions(badgeField).find((o) => o.value === badge)}
								<TemplateBadge color={option?.color || 'gray'}>{badge}</TemplateBadge>
							{:else}
								<TemplateBadge>{badge}</TemplateBadge>
							{/if}
						{/if}
					</div>
					{#if subtitle}
						<p class="tpl-card-subtitle">{formatValue(subtitle, getField(config.subtitleField!))}</p>
					{/if}
					{#if config.showDescription && description}
						<p class="tpl-card-description">{description}</p>
					{/if}
				</div>
				{#if onselect}
					<div class="tpl-card-checkbox">
						<input
							type="checkbox"
							checked={isSelected}
							onchange={(e) => onselect?.(id, e.currentTarget.checked)}
							onclick={(e) => e.stopPropagation()}
						/>
					</div>
				{/if}
			</TemplateCard>
		{/each}
	{/if}
</div>

<style>
	.tpl-card-view {
		display: grid;
		gap: var(--tpl-space-4);
		padding: var(--tpl-space-4);
	}

	.tpl-card-view-empty {
		grid-column: 1 / -1;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: var(--tpl-space-16);
		text-align: center;
	}

	.tpl-card-view-empty-icon {
		width: 64px;
		height: 64px;
		color: var(--tpl-text-muted);
		margin-bottom: var(--tpl-space-4);
	}

	.tpl-card-view-empty-icon svg {
		width: 100%;
		height: 100%;
	}

	.tpl-card-view-empty-text {
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-sm);
		color: var(--tpl-text-muted);
		margin: 0;
	}

	.tpl-card-skeleton {
		background: var(--tpl-card-bg);
		border: 1px solid var(--tpl-card-border);
		border-radius: var(--tpl-card-radius);
		overflow: hidden;
	}

	.tpl-card-skeleton-content {
		padding: var(--tpl-space-4);
		display: flex;
		flex-direction: column;
		gap: var(--tpl-space-2);
	}

	.tpl-card-image {
		width: 100%;
		height: 160px;
		overflow: hidden;
		background: var(--tpl-bg-secondary);
	}

	.tpl-card-image img {
		width: 100%;
		height: 100%;
		object-fit: cover;
		transition: transform var(--tpl-transition-normal);
	}

	:global(.tpl-card:hover) .tpl-card-image img {
		transform: scale(1.05);
	}

	.tpl-card-body {
		padding: var(--tpl-space-4);
	}

	.tpl-card-header-row {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: var(--tpl-space-2);
		margin-bottom: var(--tpl-space-1);
	}

	.tpl-card-title {
		margin: 0;
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-base);
		font-weight: var(--tpl-font-semibold);
		color: var(--tpl-text-primary);
		line-height: var(--tpl-leading-snug);
	}

	.tpl-card-subtitle {
		margin: 0;
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-sm);
		color: var(--tpl-text-secondary);
	}

	.tpl-card-description {
		margin: var(--tpl-space-2) 0 0;
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-sm);
		color: var(--tpl-text-tertiary);
		line-height: var(--tpl-leading-normal);
		display: -webkit-box;
		line-clamp: 2;
		-webkit-line-clamp: 2;
		-webkit-box-orient: vertical;
		overflow: hidden;
	}

	.tpl-card-checkbox {
		position: absolute;
		top: var(--tpl-space-3);
		right: var(--tpl-space-3);
		z-index: 1;
	}

	.tpl-card-checkbox input {
		width: 18px;
		height: 18px;
		cursor: pointer;
		accent-color: var(--tpl-accent-primary);
	}

	:global(.tpl-card) {
		position: relative;
	}
</style>
