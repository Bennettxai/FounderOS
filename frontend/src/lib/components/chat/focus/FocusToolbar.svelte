<script lang="ts">
	interface ContextItem {
		id: string;
		name: string;
		icon?: string;
	}

	interface Props {
		canSubmit: boolean;
		selectedProjectId?: string | null;
		inputValue: string;
		attachedFilesCount: number;
		availableContexts?: ContextItem[];
		selectedContextIds?: string[];
		onAttach: () => void;
		onContextToggle?: (contextId: string) => void;
		onSubmit: () => void;
	}

	let {
		canSubmit,
		selectedProjectId = null,
		inputValue,
		attachedFilesCount,
		availableContexts = [],
		selectedContextIds = [],
		onAttach,
		onContextToggle,
		onSubmit
	}: Props = $props();

	let showContextDropdown = $state(false);
</script>

<div class="input-row">
	<div class="input-row-left">
		<!-- Attach button -->
		<button
			class="attach-btn"
			onclick={onAttach}
			title="Attach files"
			aria-label="Attach files"
		>
			<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" width="20" height="20">
				<path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
			</svg>
		</button>

		<!-- Context selector -->
		{#if availableContexts.length > 0}
			<div class="context-selector">
				<button
					class="context-btn"
					onclick={() => showContextDropdown = !showContextDropdown}
					title="Select context"
					aria-label="Select context"
				>
					<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" width="18" height="18">
						<path stroke-linecap="round" stroke-linejoin="round" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
					</svg>
					{#if selectedContextIds.length > 0}
						<span class="context-count">{selectedContextIds.length}</span>
					{/if}
				</button>

				{#if showContextDropdown}
					<div class="context-dropdown">
						{#if selectedContextIds.length > 0}
							<button
								class="context-clear"
								onclick={() => {
									selectedContextIds.forEach(id => onContextToggle?.(id));
									showContextDropdown = false;
								}}
							>
								<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" width="14" height="14">
									<path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
								</svg>
								Clear ({selectedContextIds.length})
							</button>
						{/if}
						{#each availableContexts as ctx (ctx.id)}
							{@const isSelected = selectedContextIds.includes(ctx.id)}
							<button
								class="context-item"
								class:selected={isSelected}
								onclick={() => onContextToggle?.(ctx.id)}
							>
								{#if isSelected}
									<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" width="14" height="14">
										<path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
									</svg>
								{:else}
									<span class="context-icon">{ctx.icon || '📄'}</span>
								{/if}
								<span class="context-name">{ctx.name}</span>
							</button>
						{/each}
					</div>
				{/if}
			</div>
		{/if}
	</div>

	<!-- Submit button -->
	<button
		class="submit-btn"
		onclick={onSubmit}
		disabled={!canSubmit}
		title={!selectedProjectId ? 'Select a project first' : ''}
	>
		<span>{!selectedProjectId && (inputValue.trim() || attachedFilesCount > 0) ? 'Select project' : "Let's go"}</span>
		<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" width="16" height="16">
			<path stroke-linecap="round" stroke-linejoin="round" d="M13.5 4.5 21 12m0 0-7.5 7.5M21 12H3" />
		</svg>
	</button>
</div>

<style>
	/* Bottom row */
	.input-row {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 8px;
		margin-top: 8px;
	}

	.input-row-left {
		display: flex;
		align-items: center;
		gap: 4px;
	}

	/* Attach button */
	.attach-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 36px;
		height: 36px;
		border: none;
		background: transparent;
		color: var(--dt3);
		cursor: pointer;
		border-radius: 8px;
		transition: all 0.15s ease;
		flex-shrink: 0;
	}

	.attach-btn:hover {
		background: var(--dbg2);
		color: var(--dt);
	}

	/* Context selector */
	.context-selector {
		position: relative;
	}

	.context-btn {
		display: flex;
		align-items: center;
		gap: 4px;
		padding: 8px;
		border: none;
		background: transparent;
		color: var(--dt3);
		cursor: pointer;
		border-radius: 8px;
		transition: all 0.15s ease;
	}

	.context-btn:hover {
		background: var(--dbg2);
		color: var(--dt);
	}

	.context-count {
		font-size: 11px;
		font-weight: 600;
		background: var(--dt);
		color: var(--dbg);
		padding: 1px 5px;
		border-radius: 10px;
		min-width: 16px;
		text-align: center;
	}

	.context-dropdown {
		position: absolute;
		bottom: 100%;
		left: 0;
		margin-bottom: 8px;
		background: var(--dbg);
		border: 1px solid var(--dbd);
		border-radius: 12px;
		box-shadow: 0 4px 16px rgba(0, 0, 0, 0.15);
		min-width: 200px;
		max-height: 240px;
		overflow-y: auto;
		z-index: 50;
	}

	.context-clear {
		display: flex;
		align-items: center;
		gap: 8px;
		width: 100%;
		padding: 10px 14px;
		border: none;
		background: transparent;
		color: var(--dt3);
		font-size: 13px;
		cursor: pointer;
		border-bottom: 1px solid var(--dbd);
		text-align: left;
	}

	.context-clear:hover {
		background: var(--dbg2);
	}

	.context-item {
		display: flex;
		align-items: center;
		gap: 8px;
		width: 100%;
		padding: 10px 14px;
		border: none;
		background: transparent;
		color: var(--dt);
		font-size: 13px;
		cursor: pointer;
		text-align: left;
	}

	.context-item:hover {
		background: var(--dbg2);
	}

	.context-item.selected {
		color: var(--dt);
		font-weight: 600;
	}

	.context-icon {
		font-size: 14px;
	}

	.context-name {
		flex: 1;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	/* Submit button — monochromatic */
	.submit-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 8px;
		align-self: flex-end;
		padding: 10px 20px;
		background: var(--dt);
		color: var(--dbg);
		border: none;
		border-radius: 24px;
		font-size: 14px;
		font-weight: 500;
		cursor: pointer;
		transition: opacity 0.15s ease, transform 0.15s ease;
	}

	.submit-btn:hover:not(:disabled) {
		opacity: 0.85;
		transform: translateY(-1px);
	}

	.submit-btn:disabled {
		opacity: 0.35;
		cursor: not-allowed;
	}
</style>
