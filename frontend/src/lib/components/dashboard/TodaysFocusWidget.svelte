<script lang="ts">
	import { fly, scale } from 'svelte/transition';
	import { flip } from 'svelte/animate';

	interface FocusItem {
		id: string;
		title?: string;
		text?: string;
		description?: string;
		completed: boolean;
	}

	interface Props {
		items?: FocusItem[];
		isLoading?: boolean;
		onToggle?: (id: string) => void;
		onAdd?: (title: string, description?: string) => void;
		onRemove?: (id: string) => void;
		onReorder?: (items: FocusItem[]) => void;
		onEdit?: () => void;
	}

	let { items = [], isLoading = false, onToggle, onAdd, onRemove, onReorder, onEdit }: Props = $props();

	let isAdding = $state(false);
	let newTitle = $state('');
	let newDescription = $state('');

	let completedCount = $derived(items.filter((i) => i.completed).length);
	let total = $derived(items.length);

	// SVG progress ring: 72px diameter, r=28, circumference ≈ 175.9
	const RING_R = 28;
	const RING_C = 2 * Math.PI * RING_R;
	let ringOffset = $derived(
		total === 0 ? RING_C : RING_C - (completedCount / total) * RING_C
	);

	function handleToggle(id: string) {
		onToggle?.(id);
	}

	function handleAdd() {
		if (newTitle.trim()) {
			onAdd?.(newTitle.trim(), newDescription.trim() || undefined);
			newTitle = '';
			newDescription = '';
			isAdding = false;
		}
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter' && !e.shiftKey) {
			e.preventDefault();
			handleAdd();
		}
		if (e.key === 'Escape') {
			isAdding = false;
			newTitle = '';
			newDescription = '';
		}
	}
</script>

<div class="dw-focus-card">
	<!-- Header -->
	<div class="dw-focus-header">
		<div class="dw-focus-header-left">
			<div class="dw-focus-icon-wrap" aria-hidden="true">
				<svg width="16" height="16" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
						d="M13 10V3L4 14h7v7l9-11h-7z" />
				</svg>
			</div>
			<h2 class="dw-focus-title">Today's Focus</h2>
			{#if total > 0}
				<span class="dw-focus-badge">{completedCount}/{total}</span>
			{/if}
		</div>

		<div class="dw-focus-header-right">
			{#if total > 0}
				<button
					onclick={() => onEdit?.()}
					class="dw-focus-edit-btn"
					aria-label="Edit focus items"
				>
					Edit
				</button>
			{/if}
			{#if total < 5 && !isAdding}
				<button
					onclick={() => (isAdding = true)}
					class="dw-focus-add-btn"
					aria-label="Add focus item"
				>
					<svg width="14" height="14" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5"
							d="M12 4v16m8-8H4" />
					</svg>
					Add
				</button>
			{/if}
		</div>
	</div>

	<!-- Skeleton loading state -->
	{#if isLoading}
		<div class="dw-focus-skeleton" aria-hidden="true">
			{#each [1, 2, 3] as _}
				<div class="dw-focus-sk-row">
					<div class="dw-focus-sk dw-focus-sk--circle"></div>
					<div class="dw-focus-sk dw-focus-sk--circle"></div>
					<div class="dw-focus-sk dw-focus-sk--line" style="width: {55 + Math.random() * 30}%"></div>
				</div>
			{/each}
		</div>

	<!-- Empty state -->
	{:else if total === 0 && !isAdding}
		<div class="dw-focus-empty" in:fly={{ y: 8, duration: 250 }}>
			<svg class="dw-focus-empty-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5"
					d="M13 10V3L4 14h7v7l9-11h-7z" />
			</svg>
			<p class="dw-focus-empty-text">No focus items for today</p>
			<button
				onclick={() => (isAdding = true)}
				class="dw-focus-empty-btn"
				aria-label="Add your first focus item"
			>
				Add focus item
			</button>
		</div>

	<!-- Items list + ring -->
	{:else}
		<div class="dw-focus-wrap">
			<!-- Task list -->
			<div class="dw-focus-tasks">
				{#each items as item, index (item.id)}
					<div
						class="dw-focus-task {item.completed ? 'dw-focus-task--done' : ''}"
						animate:flip={{ duration: 200 }}
						in:fly={{ y: 10, duration: 300, delay: index * 50 }}
					>
						<!-- Number badge -->
						<span class="dw-focus-num" aria-hidden="true">{index + 1}</span>

						<!-- Checkbox -->
						<label class="dw-focus-checkbox" aria-label="Toggle: {item.title || item.text}">
							<input
								type="checkbox"
								checked={item.completed}
								onchange={() => handleToggle(item.id)}
								class="sr-only"
							/>
							<span class="dw-focus-check-mark" aria-hidden="true">
								{#if item.completed}
									<svg
										width="11"
										height="11"
										fill="none"
										stroke="currentColor"
										viewBox="0 0 24 24"
										in:scale={{ duration: 180, start: 0.4 }}
									>
										<path
											stroke-linecap="round"
											stroke-linejoin="round"
											stroke-width="3.5"
											d="M5 13l4 4L19 7"
										/>
									</svg>
								{/if}
							</span>
						</label>

						<!-- Text -->
						<div class="dw-focus-text-wrap">
							<span class="dw-focus-text {item.completed ? 'dw-focus-text--done' : ''}">
								{item.title || item.text}
							</span>
							{#if item.description}
								<span class="dw-focus-time">{item.description}</span>
							{/if}
						</div>
					</div>
				{/each}
			</div>

			<!-- Progress ring -->
			{#if total > 0}
				<div class="dw-focus-ring-wrap" aria-label="{completedCount} of {total} completed">
					<svg width="72" height="72" viewBox="0 0 72 72" fill="none">
						<!-- Track ring -->
						<circle
							cx="36"
							cy="36"
							r={RING_R}
							stroke="var(--dbd)"
							stroke-width="5"
							fill="none"
						/>
						<!-- Progress ring -->
						<circle
							cx="36"
							cy="36"
							r={RING_R}
							stroke="var(--dt)"
							stroke-width="5"
							fill="none"
							stroke-linecap="round"
							stroke-dasharray={RING_C}
							stroke-dashoffset={ringOffset}
							transform="rotate(-90 36 36)"
							style="transition: stroke-dashoffset 0.4s ease"
						/>
					</svg>
					<div class="dw-focus-ring-label">
						<span class="dw-focus-ring-num">{completedCount}</span>
						<span class="dw-focus-ring-denom">/{total}</span>
					</div>
				</div>
			{/if}
		</div>

		<!-- Add form -->
		{#if isAdding}
			<div class="dw-focus-add-form" in:fly={{ y: 8, duration: 200 }}>
				<input
					type="text"
					bind:value={newTitle}
					onkeydown={handleKeydown}
					placeholder="What's your focus?"
					class="dw-focus-input"
					autofocus
					aria-label="Focus item title"
				/>
				<input
					type="text"
					bind:value={newDescription}
					onkeydown={handleKeydown}
					placeholder="Add context (optional)"
					class="dw-focus-input dw-focus-input--sub"
					aria-label="Focus item description"
				/>
				<div class="dw-focus-form-actions">
					<button
						onclick={handleAdd}
						disabled={!newTitle.trim()}
						class="dw-focus-form-btn dw-focus-form-btn--primary"
					>
						Add
					</button>
					<button
						onclick={() => {
							isAdding = false;
							newTitle = '';
							newDescription = '';
						}}
						class="dw-focus-form-btn dw-focus-form-btn--ghost"
					>
						Cancel
					</button>
				</div>
			</div>
		{/if}
	{/if}
</div>

<style>
	/* ── Today's Focus Widget (dw-focus-*) — Foundation Tokens ── */

	.dw-focus-card {
		background: var(--dbg);
		border-radius: var(--radius-md);
		border: 1px solid var(--dbd);
		padding: var(--space-5);
		box-shadow: var(--shadow-sm);
		transition: box-shadow 0.25s ease;
	}

	.dw-focus-card:hover {
		box-shadow: var(--shadow-md);
	}

	/* ── Header ── */
	.dw-focus-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-bottom: var(--space-4);
		gap: var(--space-3);
	}

	.dw-focus-header-left {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		min-width: 0;
	}

	.dw-focus-header-right {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		flex-shrink: 0;
	}

	.dw-focus-icon-wrap {
		width: 2rem;
		height: 2rem;
		border-radius: var(--radius-sm);
		background: var(--dt);
		color: var(--dbg);
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
	}

	.dw-focus-title {
		font-size: var(--text-base);
		font-weight: var(--font-semibold);
		color: var(--dt);
		white-space: nowrap;
	}

	.dw-focus-badge {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		padding: 0 var(--space-2);
		height: 20px;
		border-radius: var(--radius-full);
		background: var(--dbg3);
		color: var(--dt3);
		font-size: 0.7rem;
		font-weight: var(--font-semibold);
		letter-spacing: 0.01em;
	}

	.dw-focus-edit-btn {
		height: 26px;
		padding: 0 var(--space-3);
		border-radius: var(--radius-full);
		border: 1px solid var(--dbd);
		background: transparent;
		color: var(--dt3);
		font-size: 0.75rem;
		font-weight: var(--font-medium);
		cursor: pointer;
		transition: border-color 0.15s, color 0.15s, background 0.15s;
	}

	.dw-focus-edit-btn:hover {
		border-color: var(--dbd2);
		background: var(--dbg2);
		color: var(--dt2);
	}

	.dw-focus-add-btn {
		display: inline-flex;
		align-items: center;
		gap: var(--space-1);
		height: 26px;
		padding: 0 var(--space-3);
		border-radius: var(--radius-full);
		border: 1px solid var(--dbd);
		background: var(--dt);
		color: var(--dbg);
		font-size: 0.75rem;
		font-weight: var(--font-medium);
		cursor: pointer;
		transition: opacity 0.15s;
	}

	.dw-focus-add-btn:hover {
		opacity: 0.85;
	}

	/* ── Empty state ── */
	.dw-focus-empty {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: var(--space-3);
		padding: 2rem var(--space-4);
		text-align: center;
	}

	.dw-focus-empty-icon {
		width: 1.5rem;
		height: 1.5rem;
		color: var(--dt3);
		flex-shrink: 0;
	}

	.dw-focus-empty-text {
		font-size: 0.85rem;
		color: var(--dt2);
	}

	.dw-focus-empty-btn {
		display: inline-flex;
		align-items: center;
		gap: var(--space-2);
		height: 28px;
		padding: 0 var(--space-4);
		border-radius: var(--radius-full);
		border: 1px solid var(--dbd);
		background: transparent;
		color: var(--dt2);
		font-size: 0.8rem;
		font-weight: var(--font-medium);
		cursor: pointer;
		transition: border-color 0.15s, background 0.15s, color 0.15s;
	}

	.dw-focus-empty-btn:hover {
		border-color: var(--dt4);
		background: var(--dbg2);
		color: var(--dt);
	}

	/* ── Body: list + ring ── */
	.dw-focus-wrap {
		display: flex;
		align-items: flex-start;
		gap: var(--space-6);
	}

	.dw-focus-tasks {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: var(--space-2);
	}

	/* ── Individual task row ── */
	.dw-focus-task {
		display: flex;
		flex-direction: row;
		align-items: center;
		gap: 0.6rem;
		padding: 0.6rem 0.75rem;
		background: var(--dbg2);
		border: 1px solid var(--dbd);
		border-radius: var(--radius-sm);
		transition: border-color 0.15s, background 0.15s;
	}

	.dw-focus-task:hover {
		border-color: var(--dt4);
	}

	.dw-focus-task--done {
		background: var(--dbg3);
		border-color: var(--dbd2);
	}

	/* ── Number badge ── */
	.dw-focus-num {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 22px;
		height: 22px;
		border-radius: var(--radius-full);
		background: var(--dbg3);
		color: var(--dt2);
		font-size: 0.72rem;
		font-weight: var(--font-bold);
		flex-shrink: 0;
	}

	/* ── Checkbox ── */
	.dw-focus-checkbox {
		display: inline-flex;
		flex-shrink: 0;
		cursor: pointer;
	}

	.dw-focus-check-mark {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 18px;
		height: 18px;
		border-radius: 5px;
		border: 1.5px solid var(--dbd);
		background: var(--dbg);
		transition: background 0.15s, border-color 0.15s;
		color: var(--dbg);
	}

	.dw-focus-task--done .dw-focus-check-mark {
		background: var(--dt2);
		border-color: var(--dt2);
	}

	.dw-focus-checkbox:hover .dw-focus-check-mark {
		border-color: var(--dt3);
	}

	/* ── Text content ── */
	.dw-focus-text-wrap {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: 2px;
	}

	.dw-focus-text {
		font-size: 0.84rem;
		font-weight: var(--font-medium);
		color: var(--dt);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
		transition: color 0.15s;
	}

	.dw-focus-text--done {
		text-decoration: line-through;
		opacity: 0.5;
	}

	.dw-focus-time {
		font-size: 0.72rem;
		color: var(--dt4);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	/* ── Progress ring ── */
	.dw-focus-ring-wrap {
		flex-shrink: 0;
		position: relative;
		width: 72px;
		height: 72px;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.dw-focus-ring-wrap svg {
		position: absolute;
		inset: 0;
	}

	.dw-focus-ring-label {
		position: relative;
		display: flex;
		align-items: baseline;
		gap: 1px;
		z-index: 1;
	}

	.dw-focus-ring-num {
		font-size: 1.1rem;
		font-weight: var(--font-bold);
		color: var(--dt);
		line-height: 1;
	}

	.dw-focus-ring-denom {
		font-size: 0.65rem;
		font-weight: var(--font-medium);
		color: var(--dt4);
	}

	/* ── Add form ── */
	.dw-focus-add-form {
		margin-top: var(--space-3);
		padding: var(--space-3) var(--space-4);
		border-radius: var(--radius-sm);
		border: 1px solid var(--dbd);
		background: var(--dbg2);
		display: flex;
		flex-direction: column;
		gap: var(--space-2);
	}

	.dw-focus-input {
		width: 100%;
		font-size: var(--text-sm);
		background: transparent;
		border: none;
		outline: none;
		color: var(--dt);
		font-family: inherit;
	}

	.dw-focus-input::placeholder {
		color: var(--dt4);
	}

	.dw-focus-input--sub {
		font-size: 0.78rem;
		color: var(--dt3);
		border-top: 1px solid var(--dbd2);
		padding-top: var(--space-2);
	}

	.dw-focus-form-actions {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		margin-top: var(--space-1);
	}

	.dw-focus-form-btn {
		height: 26px;
		padding: 0 var(--space-3);
		border-radius: var(--radius-full);
		font-size: 0.75rem;
		font-weight: var(--font-medium);
		cursor: pointer;
		transition: opacity 0.15s, background 0.15s, border-color 0.15s;
		font-family: inherit;
	}

	.dw-focus-form-btn--primary {
		background: var(--dt);
		color: var(--dbg);
		border: 1px solid var(--dt);
	}

	.dw-focus-form-btn--primary:disabled {
		opacity: 0.35;
		cursor: default;
	}

	.dw-focus-form-btn--primary:not(:disabled):hover {
		opacity: 0.85;
	}

	.dw-focus-form-btn--ghost {
		background: transparent;
		color: var(--dt3);
		border: 1px solid var(--dbd);
	}

	.dw-focus-form-btn--ghost:hover {
		background: var(--dbg3);
		border-color: var(--dt4);
		color: var(--dt2);
	}

	/* ── Skeleton ── */
	.dw-focus-skeleton {
		display: flex;
		flex-direction: column;
		gap: var(--space-2);
		padding: var(--space-1) 0;
	}

	.dw-focus-sk-row {
		display: flex;
		align-items: center;
		gap: 0.6rem;
		padding: 0.6rem 0.75rem;
		background: var(--dbg2);
		border: 1px solid var(--dbd);
		border-radius: var(--radius-sm);
	}

	@keyframes dw-focus-pulse {
		50% { opacity: 0.5; }
	}

	.dw-focus-sk {
		background: var(--dbg3, color-mix(in srgb, var(--dt) 8%, transparent));
		animation: dw-focus-pulse 1.5s ease-in-out infinite;
		flex-shrink: 0;
	}

	.dw-focus-sk--circle {
		width: 22px;
		height: 22px;
		border-radius: var(--radius-full);
	}

	.dw-focus-sk--line {
		height: 12px;
		border-radius: 4px;
		flex: 1;
		max-width: 100%;
	}

	/* ── sr-only utility ── */
	.sr-only {
		position: absolute;
		width: 1px;
		height: 1px;
		padding: 0;
		margin: -1px;
		overflow: hidden;
		clip: rect(0, 0, 0, 0);
		white-space: nowrap;
		border-width: 0;
	}
</style>
