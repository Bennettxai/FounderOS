<script lang="ts">
	import { onMount } from 'svelte';
	import { currentWorkspace } from '$lib/stores/workspaces';
	import {
		getRhythm,
		createRhythmEntry,
		updateRhythmEntry,
		deleteRhythmEntry,
		type RhythmEntry,
		type RhythmPeriod,
		type RhythmKind
	} from '$lib/api/rhythm';
	import { Repeat, Plus, Loader2, X, Trash2, Pencil } from 'lucide-svelte';

	const PERIODS: { value: RhythmPeriod; label: string }[] = [
		{ value: 'daily', label: 'Daily' },
		{ value: 'weekly', label: 'Weekly' },
		{ value: 'monthly', label: 'Monthly' }
	];

	const KINDS: { value: RhythmKind; label: string; heading: string; color: string }[] = [
		{ value: 'focus', label: 'Focus', heading: 'Focuses', color: '#818cf8' },
		{ value: 'blocker', label: 'Blocker', heading: 'Blockers', color: '#f87171' },
		{ value: 'priority', label: 'Priority', heading: 'Priorities', color: '#facc15' },
		{ value: 'note', label: 'Note', heading: 'Notes', color: '#94a3b8' }
	];

	let entries = $state<RhythmEntry[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let period = $state<RhythmPeriod>('daily');

	let showEdit = $state(false);
	let saving = $state(false);
	let busyId = $state<string | null>(null);
	let editing = $state<RhythmEntry | null>(null);
	let form = $state<{ kind: RhythmKind; content: string; owner: string }>({ kind: 'focus', content: '', owner: '' });

	// Focus the first field when the modal mounts.
	function autofocus(node: HTMLElement) {
		node.focus();
	}

	let wsId = $state<string | null | undefined>(undefined);
	$effect(() => {
		const id = $currentWorkspace?.id ?? null;
		if (id !== wsId) {
			wsId = id;
			load();
		}
	});

	onMount(load);

	async function load() {
		loading = true;
		error = null;
		try {
			const res = await getRhythm(period);
			entries = res.entries;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load rhythm';
		} finally {
			loading = false;
		}
	}

	function selectPeriod(p: RhythmPeriod) {
		if (p === period) return;
		period = p;
		load();
	}

	// Group entries by kind for display, in the canonical KINDS order.
	const grouped = $derived.by(() => {
		return KINDS.map((k) => ({
			kind: k,
			items: entries.filter((e) => e.kind === k.value)
		})).filter((g) => g.items.length > 0);
	});

	function openNew() {
		editing = null;
		form = { kind: 'focus', content: '', owner: '' };
		showEdit = true;
	}
	function openEdit(e: RhythmEntry) {
		editing = e;
		form = { kind: e.kind, content: e.content, owner: e.owner ?? '' };
		showEdit = true;
	}

	async function save(ev: Event) {
		ev.preventDefault();
		if (!form.content.trim()) return;
		saving = true;
		error = null;
		try {
			const body = {
				period,
				kind: form.kind,
				content: form.content.trim(),
				owner: form.owner.trim()
			};
			if (editing) await updateRhythmEntry(editing.id, body);
			else await createRhythmEntry(body);
			showEdit = false;
			await load();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to save entry';
		} finally {
			saving = false;
		}
	}

	async function remove(e: RhythmEntry) {
		if (!confirm('Delete this entry?')) return;
		busyId = e.id;
		error = null;
		try {
			await deleteRhythmEntry(e.id);
			entries = entries.filter((x) => x.id !== e.id);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to delete';
		} finally {
			busyId = null;
		}
	}
</script>

<svelte:head><title>Rhythm - BusinessOS</title></svelte:head>

<svelte:window
	onkeydown={(e) => {
		if (e.key === 'Escape' && showEdit) showEdit = false;
	}}
/>

<div class="rh-root">
	<header class="topbar">
		<div class="title-wrap"><h1>Rhythm</h1><span class="count">{entries.length}</span></div>
		<div class="tools">
			<div class="seg" role="tablist">
				{#each PERIODS as p}
					<button
						class="seg-btn"
						class:active={period === p.value}
						role="tab"
						aria-selected={period === p.value}
						onclick={() => selectPeriod(p.value)}>{p.label}</button
					>
				{/each}
			</div>
			<button class="btn btn--primary" onclick={openNew}
				><Plus size={16} strokeWidth={2.4} />New entry</button
			>
		</div>
	</header>

	{#if error}<div class="banner banner--error">{error}</div>{/if}

	{#if loading}
		<div class="loading"><Loader2 class="spin" size={20} /> Loading rhythm…</div>
	{:else if entries.length === 0}
		<div class="empty">
			<Repeat size={26} strokeWidth={1.4} />
			<p>
				No {period} rhythm yet. Capture this cadence's focuses, blockers, priorities, and notes so
				people and AI agents stay in sync.
			</p>
			<button class="btn btn--primary" onclick={openNew}
				><Plus size={16} />Add your first entry</button
			>
		</div>
	{:else}
		<div class="list">
			{#each grouped as g (g.kind.value)}
				<div class="cat">
					<div class="cat-head">
						<span class="dot" style="background:{g.kind.color}"></span>
						{g.kind.heading}
						<span class="cat-count">{g.items.length}</span>
					</div>
					{#each g.items as e (e.id)}
						<div class="entry" style="--accent:{g.kind.color}">
							<div class="entry-main">
								<div class="entry-content">{e.content || '-'}</div>
								{#if e.owner}<div class="entry-owner">{e.owner}</div>{/if}
							</div>
							<div class="entry-actions">
								<button class="icon-btn" title="Edit" onclick={() => openEdit(e)}
									><Pencil size={14} /></button
								>
								<button
									class="icon-btn icon-btn--danger"
									title="Delete"
									disabled={busyId === e.id}
									onclick={() => remove(e)}
									>{#if busyId === e.id}<Loader2 class="spin" size={14} />{:else}<Trash2
											size={14}
										/>{/if}</button
								>
							</div>
						</div>
					{/each}
				</div>
			{/each}
		</div>
	{/if}
</div>

{#if showEdit}
	<div
		class="overlay"
		role="button"
		tabindex="0"
		onclick={() => (showEdit = false)}
		onkeydown={(e) => e.key === 'Escape' && (showEdit = false)}
	>
		<div
			class="modal"
			role="dialog"
			tabindex="-1"
			onclick={(e) => e.stopPropagation()}
			onkeydown={() => {}}
		>
			<div class="modal-head">
				<h2>{editing ? 'Edit entry' : 'New entry'}</h2>
				<button class="card-x" onclick={() => (showEdit = false)} aria-label="Close"
					><X size={18} /></button
				>
			</div>
			<form onsubmit={save}>
				<label class="field">
					<span>Kind</span>
					<select bind:value={form.kind}>
						{#each KINDS as k}
							<option value={k.value}>{k.label}</option>
						{/each}
					</select>
				</label>
				<label class="field">
					<span>Content</span>
					<textarea
						bind:value={form.content}
						rows="4"
						placeholder="What this {period} entry is…"
						required
						use:autofocus
					></textarea>
				</label>
				<label class="field">
					<span>Owner</span>
					<input bind:value={form.owner} placeholder="e.g. owner or team (optional)" />
				</label>
				<div class="modal-actions">
					<button type="button" class="btn btn--ghost" onclick={() => (showEdit = false)}
						>Cancel</button
					>
					<button
						type="submit"
						class="btn btn--primary"
						disabled={saving || !form.content.trim()}
						>{#if saving}<Loader2 class="spin" size={15} />{/if}{editing
							? 'Save'
							: 'Add entry'}</button
					>
				</div>
			</form>
		</div>
	</div>
{/if}

<style>
	.rh-root {
		height: 100%;
		display: flex;
		flex-direction: column;
		background: var(--dbg);
		color: var(--dt);
		overflow: hidden;
	}
	.topbar {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 16px;
		padding: 16px 24px;
		border-bottom: 1px solid var(--dbd);
		flex-shrink: 0;
	}
	.title-wrap {
		display: flex;
		align-items: center;
		gap: 10px;
	}
	.title-wrap h1 {
		font-size: 1.15rem;
		font-weight: 680;
		letter-spacing: -0.02em;
		margin: 0;
	}
	.count {
		font-size: 0.74rem;
		color: var(--dt3);
		background: color-mix(in srgb, var(--dt) 8%, transparent);
		padding: 2px 9px;
		border-radius: 999px;
	}
	.tools {
		display: flex;
		align-items: center;
		gap: 10px;
	}
	.seg {
		display: inline-flex;
		align-items: center;
		gap: 2px;
		padding: 3px;
		border: 1px solid var(--dbd);
		border-radius: 10px;
	}
	.seg-btn {
		padding: 6px 13px;
		border-radius: 7px;
		font-size: 0.81rem;
		font-weight: 560;
		cursor: pointer;
		border: none;
		background: transparent;
		color: var(--dt3);
		font-family: inherit;
	}
	.seg-btn:hover {
		color: var(--dt);
	}
	.seg-btn.active {
		background: color-mix(in srgb, var(--dt) 10%, transparent);
		color: var(--dt);
	}
	.btn {
		display: inline-flex;
		align-items: center;
		gap: 7px;
		padding: 8px 15px;
		border-radius: 9px;
		font-size: 0.83rem;
		font-weight: 580;
		cursor: pointer;
		border: 1px solid transparent;
		white-space: nowrap;
	}
	.btn--primary {
		background: var(--dt);
		color: var(--dbg);
	}
	.btn--ghost {
		background: transparent;
		border-color: var(--dbd);
		color: var(--dt2);
	}
	.btn:disabled {
		opacity: 0.55;
		cursor: not-allowed;
	}
	.loading,
	.empty {
		flex: 1;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 14px;
		color: var(--dt3);
		text-align: center;
		padding: 0 24px;
	}
	.empty p {
		max-width: 420px;
		line-height: 1.5;
	}
	.banner {
		margin: 14px 24px 0;
		padding: 11px 14px;
		border-radius: 10px;
		font-size: 0.83rem;
	}
	.banner--error {
		background: color-mix(in srgb, #ef4444 12%, transparent);
		color: #ef4444;
	}
	.list {
		flex: 1;
		overflow-y: auto;
		padding: 18px 24px;
		max-width: 820px;
		width: 100%;
		margin: 0 auto;
	}
	.cat {
		margin-bottom: 24px;
	}
	.cat-head {
		display: flex;
		align-items: center;
		gap: 8px;
		font-size: 0.74rem;
		font-weight: 640;
		color: var(--dt3);
		text-transform: uppercase;
		letter-spacing: 0.06em;
		margin-bottom: 10px;
		padding-bottom: 7px;
		border-bottom: 1px solid var(--dbd);
	}
	.dot {
		width: 7px;
		height: 7px;
		border-radius: 999px;
		flex-shrink: 0;
	}
	.cat-count {
		margin-left: auto;
		font-size: 0.72rem;
		font-weight: 600;
		color: var(--dt3);
		letter-spacing: 0;
	}
	.entry {
		display: flex;
		align-items: flex-start;
		gap: 14px;
		padding: 12px 4px 12px 12px;
		border-left: 2px solid transparent;
		transition: border-color 0.12s;
	}
	.entry:hover {
		border-left-color: var(--accent);
	}
	.entry:not(:last-child) {
		border-bottom: 1px solid color-mix(in srgb, var(--dt) 5%, transparent);
	}
	.entry-main {
		flex: 1;
		min-width: 0;
	}
	.entry-content {
		font-size: 0.88rem;
		color: var(--dt);
		line-height: 1.55;
		white-space: pre-wrap;
	}
	.entry-owner {
		font-size: 0.72rem;
		font-weight: 600;
		color: var(--dt3);
		margin-top: 5px;
		text-transform: uppercase;
		letter-spacing: 0.04em;
	}
	.entry-actions {
		display: flex;
		gap: 4px;
		flex-shrink: 0;
		opacity: 0;
		transition: opacity 0.12s;
	}
	.entry:hover .entry-actions {
		opacity: 1;
	}
	.icon-btn {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 28px;
		height: 28px;
		border-radius: 7px;
		border: none;
		background: transparent;
		color: var(--dt3);
		cursor: pointer;
	}
	.icon-btn:hover {
		background: color-mix(in srgb, var(--dt) 8%, transparent);
		color: var(--dt);
	}
	.icon-btn:disabled {
		opacity: 0.55;
		cursor: not-allowed;
	}
	.icon-btn--danger:hover {
		background: color-mix(in srgb, #ef4444 12%, transparent);
		color: #ef4444;
	}
	.overlay {
		position: fixed;
		inset: 0;
		background: rgba(0, 0, 0, 0.55);
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 100;
		padding: 20px;
	}
	.modal {
		width: 100%;
		max-width: 480px;
		background: var(--dbg);
		border: 1px solid var(--dbd);
		border-radius: 16px;
		padding: 22px;
		box-shadow: 0 24px 60px rgba(0, 0, 0, 0.5);
	}
	.modal-head {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-bottom: 18px;
	}
	.modal-head h2 {
		font-size: 1.05rem;
		font-weight: 640;
		margin: 0;
	}
	.card-x {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 28px;
		height: 28px;
		border-radius: 7px;
		border: none;
		background: transparent;
		color: var(--dt3);
		cursor: pointer;
	}
	.card-x:hover {
		background: color-mix(in srgb, var(--dt) 8%, transparent);
	}
	.field {
		display: flex;
		flex-direction: column;
		gap: 6px;
		margin-bottom: 14px;
	}
	.field span {
		font-size: 0.8rem;
		font-weight: 560;
		color: var(--dt2);
	}
	.field input,
	.field textarea,
	.field select {
		padding: 9px 12px;
		border-radius: 9px;
		border: 1px solid var(--dbd);
		background: var(--dbg);
		color: var(--dt);
		font-size: 0.86rem;
		outline: none;
		font-family: inherit;
		resize: vertical;
	}
	.field input:focus,
	.field textarea:focus,
	.field select:focus {
		border-color: color-mix(in srgb, var(--dt) 40%, transparent);
	}
	.modal-actions {
		display: flex;
		justify-content: flex-end;
		gap: 10px;
		margin-top: 6px;
	}
	:global(.spin) {
		animation: spin 0.9s linear infinite;
	}
	@keyframes spin {
		to {
			transform: rotate(360deg);
		}
	}
</style>
