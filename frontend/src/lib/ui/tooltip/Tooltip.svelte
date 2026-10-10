<script lang="ts">
	/**
	 * Tooltip Component - BusinessOS Style
	 * Modern document-centric tooltip patterns
	 */
	import { Tooltip as TooltipPrimitive } from 'bits-ui';
	import { type Snippet } from 'svelte';

	type TooltipSide = 'top' | 'right' | 'bottom' | 'left';

	interface Props {
		content?: string | Snippet;
		shortcut?: string | string[];
		side?: TooltipSide;
		align?: 'start' | 'center' | 'end';
		delayDuration?: number;
		class?: string;
		children: Snippet;
	}

	let {
		content,
		shortcut,
		side = 'top',
		align = 'center',
		delayDuration = 500,
		class: className = '',
		children
	}: Props = $props();

	const formatShortcut = (shortcut: string | string[]): string[] => {
		const shortcuts = Array.isArray(shortcut) ? shortcut : [shortcut];
		return shortcuts.map((s) => {
			return s
				.replace('$mod', navigator?.platform?.includes('Mac') ? '⌘' : 'Ctrl')
				.replace('$alt', navigator?.platform?.includes('Mac') ? '⌥' : 'Alt')
				.replace('$shift', navigator?.platform?.includes('Mac') ? '⇧' : 'Shift');
		});
	};

	const formattedShortcut = $derived(shortcut ? formatShortcut(shortcut) : []);
</script>

{#if content}
	<TooltipPrimitive.Provider>
		<TooltipPrimitive.Root {delayDuration}>
			<TooltipPrimitive.Trigger>
				{#snippet child({ props })}
					<span {...props}>
						{@render children()}
					</span>
				{/snippet}
			</TooltipPrimitive.Trigger>
			<TooltipPrimitive.Portal>
				<TooltipPrimitive.Content
					{side}
					{align}
					sideOffset={6}
					class="bos-tooltip {className}"
				>
					{#if shortcut}
						<div class="bos-tooltip__with-shortcut">
							<span class="bos-tooltip__text">
								{#if typeof content === 'string'}
									{content}
								{:else if content}
									{@render content()}
								{/if}
							</span>
							<div class="bos-tooltip__shortcut-group">
								{#each formattedShortcut as key}
									<kbd class="bos-tooltip__shortcut">{key}</kbd>
								{/each}
							</div>
						</div>
					{:else if typeof content === 'string'}
						{content}
					{:else}
						{@render content()}
					{/if}
				</TooltipPrimitive.Content>
			</TooltipPrimitive.Portal>
		</TooltipPrimitive.Root>
	</TooltipPrimitive.Provider>
{:else}
	{@render children()}
{/if}

<style>
	:global(.bos-tooltip) {
		z-index: 1001;
		max-width: 280px;
		padding: 4px 12px;
		font-size: 12px;
		font-family: var(--font-text, -apple-system, BlinkMacSystemFont, sans-serif);
		line-height: 1.4;
		color: #ffffff;
		background-color: rgba(30, 30, 30, 0.92);
		backdrop-filter: blur(12px);
		-webkit-backdrop-filter: blur(12px);
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 8px;
		box-shadow: 0 4px 16px rgba(0, 0, 0, 0.2);
		user-select: none;
	}

	/* Animation */
	:global(.bos-tooltip[data-state='delayed-open']) {
		animation: tooltip-in 0.15s ease-out;
	}

	:global(.bos-tooltip[data-state='closed']) {
		animation: tooltip-out 0.1s ease-in;
	}

	@keyframes tooltip-in {
		from {
			opacity: 0;
			transform: scale(0.96);
		}
		to {
			opacity: 1;
			transform: scale(1);
		}
	}

	@keyframes tooltip-out {
		from {
			opacity: 1;
			transform: scale(1);
		}
		to {
			opacity: 0;
			transform: scale(0.96);
		}
	}

	:global(.bos-tooltip__with-shortcut) {
		display: flex;
		align-items: center;
		gap: 12px;
	}

	:global(.bos-tooltip__text) {
		flex: 1;
	}

	:global(.bos-tooltip__shortcut-group) {
		display: flex;
		gap: 4px;
	}

	:global(.bos-tooltip__shortcut) {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		min-width: 20px;
		height: 20px;
		padding: 0 6px;
		font-size: 10px;
		font-family: var(--font-code, monospace);
		font-weight: 500;
		color: rgba(255, 255, 255, 0.5);
		background-color: rgba(255, 255, 255, 0.1);
		border-radius: 4px;
	}

	/* Dark mode — inverted tooltip */
	:global(.dark .bos-tooltip) {
		background-color: rgba(245, 245, 247, 0.95);
		color: #1a1a1a;
		border-color: rgba(0, 0, 0, 0.06);
		box-shadow: 0 4px 16px rgba(0, 0, 0, 0.3);
	}

	:global(.dark .bos-tooltip__shortcut) {
		color: rgba(0, 0, 0, 0.4);
		background-color: rgba(0, 0, 0, 0.08);
	}
</style>
