<script lang="ts">
	import type { Block } from '../../entities/types';
	import { ChevronRight, ChevronDown, ImageIcon, Link2, ExternalLink, Lightbulb, AlertCircle, Info, AlertTriangle, CheckCircle } from 'lucide-svelte';
	import TableBlock from './TableBlock.svelte';

	interface Props {
		block: Block;
		index: number;
		readOnly: boolean;
		// Svelte action for contenteditable sync
		contenteditableAction: (node: HTMLElement) => { update(): void; destroy(): void };
		onContentInput: (e: Event) => void;
		onKeydown: (e: KeyboardEvent) => void;
		onMouseUp: () => void;
		onCheckboxChange: (e: Event) => void;
		onToggleExpand: () => void;
		onDividerClick: () => void;
		onSelectDividerStyle: (style: string) => void;
		onCloseDividerPicker: () => void;
		onBlockChange: (block: Block) => void;
		showDividerPicker: boolean;
		getPlaceholder: (type: string) => string;
	}

	let {
		block,
		index,
		readOnly,
		contenteditableAction,
		onContentInput,
		onKeydown,
		onMouseUp,
		onCheckboxChange,
		onToggleExpand,
		onDividerClick,
		onSelectDividerStyle,
		onCloseDividerPicker,
		onBlockChange,
		showDividerPicker,
		getPlaceholder
	}: Props = $props();

	const dividerStyles = [
		{ value: 'solid', label: 'Solid' },
		{ value: 'dashed', label: 'Dashed' },
		{ value: 'dotted', label: 'Dotted' },
		{ value: 'thick', label: 'Thick' },
		{ value: 'double', label: 'Double' },
		{ value: 'gradient', label: 'Gradient' }
	] as const;

	// Map callout icon names to Lucide components
	const calloutIcons: Record<string, typeof Lightbulb> = {
		Lightbulb,
		AlertCircle,
		Info,
		AlertTriangle,
		CheckCircle
	};

	function getCalloutIcon(iconName: string | unknown): typeof Lightbulb {
		if (typeof iconName === 'string' && iconName in calloutIcons) {
			return calloutIcons[iconName];
		}
		return Lightbulb;
	}
</script>

<div class="block-content" data-block-type={block.type}>
	{#if block.type === 'paragraph'}
		<p
			class="block block--paragraph"
			contenteditable={!readOnly}
			oninput={onContentInput}
			onkeydown={onKeydown}
			onmouseup={onMouseUp}
			data-placeholder={getPlaceholder('paragraph')}
			use:contenteditableAction
		></p>
	{:else if block.type === 'heading_1'}
		<h1
			class="block block--h1"
			contenteditable={!readOnly}
			oninput={onContentInput}
			onkeydown={onKeydown}
			onmouseup={onMouseUp}
			data-placeholder="Heading 1"
			use:contenteditableAction
		></h1>
	{:else if block.type === 'heading_2'}
		<h2
			class="block block--h2"
			contenteditable={!readOnly}
			oninput={onContentInput}
			onkeydown={onKeydown}
			onmouseup={onMouseUp}
			data-placeholder="Heading 2"
			use:contenteditableAction
		></h2>
	{:else if block.type === 'heading_3'}
		<h3
			class="block block--h3"
			contenteditable={!readOnly}
			oninput={onContentInput}
			onkeydown={onKeydown}
			onmouseup={onMouseUp}
			data-placeholder="Heading 3"
			use:contenteditableAction
		></h3>
	{:else if block.type === 'bulleted_list'}
		<div class="block block--list">
			<span class="block__bullet">•</span>
			<span
				class="block__list-content"
				contenteditable={!readOnly}
				oninput={onContentInput}
				onkeydown={onKeydown}
				onmouseup={onMouseUp}
				data-placeholder="List item"
				use:contenteditableAction
			></span>
		</div>
	{:else if block.type === 'numbered_list'}
		<div class="block block--list">
			<span class="block__number">{index + 1}.</span>
			<span
				class="block__list-content"
				contenteditable={!readOnly}
				oninput={onContentInput}
				onkeydown={onKeydown}
				onmouseup={onMouseUp}
				data-placeholder="List item"
				use:contenteditableAction
			></span>
		</div>
	{:else if block.type === 'to_do'}
		<div class="block block--todo">
			<input
				type="checkbox"
				class="block__checkbox"
				checked={block.properties.checked ?? false}
				onchange={onCheckboxChange}
				disabled={readOnly}
			/>
			<span
				class="block__todo-content"
				class:block__todo-content--checked={block.properties.checked}
				contenteditable={!readOnly}
				oninput={onContentInput}
				onkeydown={onKeydown}
				onmouseup={onMouseUp}
				data-placeholder="To-do"
				use:contenteditableAction
			></span>
		</div>
	{:else if block.type === 'quote'}
		<blockquote
			class="block block--quote"
			contenteditable={!readOnly}
			oninput={onContentInput}
			onkeydown={onKeydown}
			onmouseup={onMouseUp}
			data-placeholder="Quote"
			use:contenteditableAction
		></blockquote>
	{:else if block.type === 'divider'}
		{@const dividerStyle = block.properties?.divider_style || 'solid'}
		<div class="divider-wrapper">
			<!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
			<div
				class="block block--divider block--divider--{dividerStyle}"
				onclick={() => !readOnly && onDividerClick()}
				role={readOnly ? undefined : 'button'}
				tabindex={readOnly ? undefined : 0}
			>
				{#if dividerStyle === 'gradient'}
					<div class="divider-gradient"></div>
				{:else}
					<hr />
				{/if}
			</div>
			{#if showDividerPicker && !readOnly}
				<!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
				<div class="divider-picker" onclick={(e) => e.stopPropagation()}>
					<div class="divider-picker__label">Divider Style</div>
					<div class="divider-picker__options">
						{#each dividerStyles as style}
							<button
								class="btn-pill btn-pill-ghost divider-picker__option"
								class:divider-picker__option--active={dividerStyle === style.value}
								onclick={() => onSelectDividerStyle(style.value)}
							>
								<span class="divider-picker__preview divider-picker__preview--{style.value}"></span>
								<span>{style.label}</span>
							</button>
						{/each}
					</div>
				</div>
			{/if}
		</div>
	{:else if block.type === 'code'}
		<pre class="block block--code"><code
			contenteditable={!readOnly}
			oninput={onContentInput}
			onkeydown={onKeydown}
			onmouseup={onMouseUp}
			data-placeholder="Code"
			use:contenteditableAction
		></code></pre>
	{:else if block.type === 'callout'}
		{@const CalloutIcon = getCalloutIcon(block.properties.icon)}
		<div class="block block--callout">
			<span class="block__callout-icon">
				<CalloutIcon class="h-5 w-5" />
			</span>
			<span
				class="block__callout-content"
				contenteditable={!readOnly}
				oninput={onContentInput}
				onkeydown={onKeydown}
				onmouseup={onMouseUp}
				data-placeholder="Callout"
				use:contenteditableAction
			></span>
		</div>
	{:else if block.type === 'toggle'}
		<div class="block block--toggle">
			<button
				class="block__toggle-btn"
				onclick={onToggleExpand}
				disabled={readOnly}
			>
				{#if block.properties.expanded}
					<ChevronDown class="h-4 w-4" />
				{:else}
					<ChevronRight class="h-4 w-4" />
				{/if}
			</button>
			<span
				class="block__toggle-content"
				contenteditable={!readOnly}
				oninput={onContentInput}
				onkeydown={onKeydown}
				onmouseup={onMouseUp}
				data-placeholder="Toggle"
				use:contenteditableAction
			></span>
		</div>
	{:else if block.type === 'image'}
		<div class="block block--image">
			{#if block.properties.url}
				<img
					src={block.properties.url as string}
					alt={typeof block.properties.caption === 'string' ? block.properties.caption : 'Image'}
					class="block__image-img"
				/>
				{#if block.properties.caption}
					<p class="block__image-caption">{typeof block.properties.caption === 'string' ? block.properties.caption : ''}</p>
				{/if}
			{:else}
				<div class="block__image-placeholder">
					<ImageIcon class="h-8 w-8" />
					<span>Click to add an image</span>
					<input
						type="file"
						accept="image/*"
						class="block__image-input"
						onchange={(e) => {
							const file = (e.target as HTMLInputElement).files?.[0];
							if (file) {
								const url = URL.createObjectURL(file);
								onBlockChange({
									...block,
									properties: { ...block.properties, url }
								});
							}
						}}
					/>
				</div>
			{/if}
		</div>
	{:else if block.type === 'bookmark'}
		<div class="block block--bookmark">
			{#if block.properties.url}
				<a
					href={block.properties.url as string}
					target="_blank"
					rel="noopener noreferrer"
					class="block__bookmark-link"
				>
					<div class="block__bookmark-content">
						<div class="block__bookmark-title">
							{block.properties.title || block.properties.url}
						</div>
						{#if block.properties.description}
							<div class="block__bookmark-description">
								{block.properties.description}
							</div>
						{/if}
						<div class="block__bookmark-url">
							<Link2 class="h-3 w-3" />
							{block.properties.url}
						</div>
					</div>
					<ExternalLink class="h-4 w-4 block__bookmark-icon" />
				</a>
			{:else}
				<div class="block__bookmark-empty">
					<Link2 class="h-5 w-5" />
					<input
						type="url"
						placeholder="Paste a link..."
						class="block__bookmark-input"
						onkeydown={(e) => {
							if (e.key === 'Enter') {
								const url = (e.target as HTMLInputElement).value;
								if (url) {
									onBlockChange({
										...block,
										properties: { ...block.properties, url, title: url }
									});
								}
							}
						}}
					/>
				</div>
			{/if}
		</div>
	{:else if block.type === 'table'}
		<TableBlock
			{block}
			{readOnly}
			onBlockChange={(updated) => onBlockChange(updated)}
		/>
	{:else}
		<p
			class="block block--paragraph"
			contenteditable={!readOnly}
			oninput={onContentInput}
			onkeydown={onKeydown}
			onmouseup={onMouseUp}
			use:contenteditableAction
		></p>
	{/if}
</div>

<style>
	.block-content {
		flex: 1;
		min-width: 0;
	}

	.block {
		outline: none;
		word-break: break-word;
	}

	.block:empty::before {
		content: attr(data-placeholder);
		color: var(--dt4);
	}

	.block--paragraph {
		font-size: 1rem;
		line-height: 1.6;
		margin: 0;
		padding: 0.125rem 0;
	}

	.block--h1 {
		font-size: 1.875rem;
		font-weight: 700;
		line-height: 1.3;
		margin: 1rem 0 0.5rem;
	}

	.block--h2 {
		font-size: 1.5rem;
		font-weight: 600;
		line-height: 1.35;
		margin: 0.875rem 0 0.375rem;
	}

	.block--h3 {
		font-size: 1.25rem;
		font-weight: 600;
		line-height: 1.4;
		margin: 0.75rem 0 0.25rem;
	}

	.block--list {
		display: flex;
		align-items: flex-start;
		gap: 0.5rem;
		padding: 0.125rem 0;
	}

	.block__bullet,
	.block__number {
		color: var(--dt3);
		font-size: 1rem;
		line-height: 1.6;
		min-width: 1.25rem;
	}

	.block__list-content {
		flex: 1;
		outline: none;
	}

	.block--todo {
		display: flex;
		align-items: flex-start;
		gap: 0.5rem;
		padding: 0.125rem 0;
	}

	.block__checkbox {
		width: 16px;
		height: 16px;
		margin-top: 0.25rem;
		cursor: pointer;
	}

	.block__todo-content {
		flex: 1;
		outline: none;
	}

	.block__todo-content--checked {
		text-decoration: line-through;
		color: var(--dt3);
	}

	.block--quote {
		border-left: 3px solid var(--dbd);
		padding-left: 1rem;
		margin: 0.25rem 0;
		font-style: italic;
		color: var(--dt3);
	}

	/* Divider wrapper */
	.divider-wrapper {
		position: relative;
		margin: 1rem 0;
	}

	.block--divider {
		cursor: pointer;
		padding: 0.5rem 0;
		transition: opacity 0.15s;
	}

	.block--divider:hover {
		opacity: 0.7;
	}

	.block--divider hr {
		border: none;
		margin: 0;
	}

	.block--divider--solid hr {
		height: 1px;
		background-color: var(--dbd);
	}

	.block--divider--dashed hr {
		height: 1px;
		background-image: repeating-linear-gradient(
			90deg,
			var(--dbd) 0px,
			var(--dbd) 8px,
			transparent 8px,
			transparent 16px
		);
	}

	.block--divider--dotted hr {
		height: 2px;
		background-image: repeating-linear-gradient(
			90deg,
			var(--dbd) 0px,
			var(--dbd) 3px,
			transparent 3px,
			transparent 8px
		);
	}

	.block--divider--thick hr {
		height: 3px;
		background-color: var(--dbd);
	}

	.block--divider--double hr {
		height: 5px;
		background-image: linear-gradient(
			to bottom,
			var(--dbd) 0px,
			var(--dbd) 1px,
			transparent 1px,
			transparent 4px,
			var(--dbd) 4px,
			var(--dbd) 5px
		);
	}

	.divider-gradient {
		height: 2px;
		background: linear-gradient(90deg,
			transparent 0%,
			rgba(30, 150, 235, 0.5) 20%,
			#1e96eb 50%,
			rgba(30, 150, 235, 0.5) 80%,
			transparent 100%
		);
		border-radius: 1px;
	}

	.divider-picker {
		position: absolute;
		top: 100%;
		left: 50%;
		transform: translateX(-50%);
		z-index: 50;
		margin-top: 0.5rem;
		padding: 0.5rem;
		background: var(--dbg);
		border: 1px solid var(--dbd);
		border-radius: 0.5rem;
		box-shadow: 0 4px 16px rgba(0, 0, 0, 0.12);
		min-width: 160px;
	}

	.divider-picker__label {
		font-size: 0.75rem;
		font-weight: 500;
		color: var(--dt3);
		margin-bottom: 0.5rem;
		padding: 0 0.25rem;
	}

	.divider-picker__options {
		display: flex;
		flex-direction: column;
		gap: 0.125rem;
	}

	.divider-picker__option {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		padding: 0.5rem 0.5rem;
		background: transparent;
		border: none;
		border-radius: 0.375rem;
		font-size: 0.875rem;
		color: var(--dt);
		cursor: pointer;
		transition: background-color 0.15s;
	}

	.divider-picker__option:hover {
		background: var(--dbg2);
	}

	.divider-picker__option--active {
		background: rgba(30, 150, 235, 0.1);
		color: #1e96eb;
	}

	.divider-picker__preview {
		width: 40px;
		height: 2px;
		flex-shrink: 0;
		border-radius: 1px;
	}

	.divider-picker__preview--solid {
		background-color: currentColor;
		opacity: 0.3;
	}

	.divider-picker__preview--dashed {
		background-image: repeating-linear-gradient(
			90deg,
			currentColor 0px,
			currentColor 4px,
			transparent 4px,
			transparent 8px
		);
		opacity: 0.3;
	}

	.divider-picker__preview--dotted {
		background-image: repeating-linear-gradient(
			90deg,
			currentColor 0px,
			currentColor 2px,
			transparent 2px,
			transparent 6px
		);
		opacity: 0.3;
	}

	.divider-picker__preview--thick {
		height: 4px;
		background-color: currentColor;
		opacity: 0.3;
	}

	.divider-picker__preview--double {
		height: 6px;
		background-image: linear-gradient(
			to bottom,
			currentColor 0px,
			currentColor 2px,
			transparent 2px,
			transparent 4px,
			currentColor 4px,
			currentColor 6px
		);
		opacity: 0.3;
	}

	.divider-picker__preview--gradient {
		background: linear-gradient(90deg,
			transparent 0%,
			#1e96eb 50%,
			transparent 100%
		);
	}

	.block--code {
		background-color: var(--dbg2);
		border-radius: 0.375rem;
		padding: 1rem;
		margin: 0.25rem 0;
		font-family: ui-monospace, monospace;
		font-size: 0.875rem;
		overflow-x: auto;
	}

	.block--code code {
		display: block;
		outline: none;
	}

	.block--callout {
		display: flex;
		align-items: flex-start;
		gap: 0.75rem;
		background-color: var(--dbg2);
		border-radius: 0.375rem;
		padding: 1rem;
		margin: 0.25rem 0;
	}

	.block__callout-icon {
		display: flex;
		align-items: center;
		justify-content: center;
		color: var(--dt);
		flex-shrink: 0;
	}

	.block__callout-content {
		flex: 1;
		outline: none;
	}

	.block--toggle {
		display: flex;
		align-items: flex-start;
		gap: 0.25rem;
		padding: 0.125rem 0;
	}

	.block__toggle-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 24px;
		height: 24px;
		padding: 0;
		background: transparent;
		border: none;
		border-radius: 0.25rem;
		color: var(--dt3);
		cursor: pointer;
		transition: background-color 0.1s;
		flex-shrink: 0;
	}

	.block__toggle-btn:hover {
		background-color: var(--dbg2);
	}

	.block__toggle-content {
		flex: 1;
		outline: none;
		line-height: 1.6;
	}

	.block--image {
		margin: 0.5rem 0;
	}

	.block__image-img {
		max-width: 100%;
		border-radius: 0.375rem;
	}

	.block__image-caption {
		font-size: 0.875rem;
		color: var(--dt3);
		text-align: center;
		margin-top: 0.5rem;
	}

	.block__image-placeholder {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 0.5rem;
		padding: 2rem;
		background-color: var(--dbg2);
		border: 2px dashed var(--dbd);
		border-radius: 0.5rem;
		color: var(--dt3);
		cursor: pointer;
		position: relative;
		transition: border-color 0.15s, background-color 0.15s;
	}

	.block__image-placeholder:hover {
		border-color: var(--dt3);
		background-color: var(--dbg2);
	}

	.block__image-input {
		position: absolute;
		inset: 0;
		opacity: 0;
		cursor: pointer;
	}

	.block--bookmark {
		margin: 0.5rem 0;
	}

	.block__bookmark-link {
		display: flex;
		align-items: center;
		gap: 1rem;
		padding: 1rem;
		background-color: var(--dbg2);
		border: 1px solid var(--dbd);
		border-radius: 0.5rem;
		text-decoration: none;
		color: inherit;
		transition: background-color 0.15s;
	}

	.block__bookmark-link:hover {
		background-color: var(--dbg2);
	}

	.block__bookmark-content {
		flex: 1;
		min-width: 0;
	}

	.block__bookmark-title {
		font-weight: 500;
		color: var(--dt);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.block__bookmark-description {
		font-size: 0.875rem;
		color: var(--dt3);
		margin-top: 0.25rem;
		display: -webkit-box;
		-webkit-line-clamp: 2;
		-webkit-box-orient: vertical;
		overflow: hidden;
	}

	.block__bookmark-url {
		display: flex;
		align-items: center;
		gap: 0.375rem;
		font-size: 0.75rem;
		color: var(--dt3);
		margin-top: 0.5rem;
	}

	.block__bookmark-icon {
		color: var(--dt3);
		flex-shrink: 0;
	}

	.block__bookmark-empty {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		padding: 0.75rem 1rem;
		background-color: var(--dbg2);
		border: 1px solid var(--dbd);
		border-radius: 0.5rem;
		color: var(--dt3);
	}

	.block__bookmark-input {
		flex: 1;
		background: transparent;
		border: none;
		outline: none;
		font-size: 0.875rem;
		color: var(--dt);
	}

	.block__bookmark-input::placeholder {
		color: var(--dt3);
	}

	/* Inline formatting styles */
	.block :global(.inline-code),
	.block :global(code:not([class])) {
		background-color: var(--dbg2);
		padding: 0.125rem 0.375rem;
		border-radius: 0.25rem;
		font-family: ui-monospace, monospace;
		font-size: 0.875em;
		color: var(--dt);
	}

	.block :global(a) {
		color: #1e96eb;
		text-decoration: underline;
		text-underline-offset: 2px;
	}

	.block :global(a:hover) {
		text-decoration-thickness: 2px;
	}

	.block :global(b),
	.block :global(strong) {
		font-weight: 600;
	}

	.block :global(i),
	.block :global(em) {
		font-style: italic;
	}

	.block :global(u) {
		text-decoration: underline;
	}

	.block :global(s),
	.block :global(strike),
	.block :global(del) {
		text-decoration: line-through;
	}
</style>
