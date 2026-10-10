<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import {
		listBoards,
		createBoard,
		deleteBoard,
		pinBoard,
		type Board,
		type BoardView,
		type BoardLayoutEntry
	} from '$lib/api/boards';
	import { getClients, type ClientListResponse } from '$lib/api/clients';
	import { currentWorkspace } from '$lib/stores/workspaces';
	import { SquareKanban, Plus, Loader2, X, Trash2, Star } from 'lucide-svelte';

	const VIEW_OPTIONS: { value: BoardView; label: string; hint: string }[] = [
		{ value: 'projects', label: 'Projects', hint: 'Project cards' },
		{ value: 'tasks', label: 'Tasks', hint: 'Task list by status' },
		{ value: 'team', label: 'Team', hint: 'People on the work' },
		{ value: 'deals', label: 'Deals', hint: 'Pipeline for the context' },
		{ value: 'clients', label: 'Clients', hint: 'Client records' }
	];

	let boards = $state<Board[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);

	// Builder (New board) state
	let showCreate = $state(false);
	let creating = $state(false);
	let formName = $state('');
	let formClientId = $state('');
	let selectedViews = $state<BoardView[]>([]); // order = checked order
	let formError = $state<string | null>(null);

	// Client context picker
	let clients = $state<ClientListResponse[]>([]);
	let clientsLoaded = $state(false);
	let clientsLoading = $state(false);
	let clientsError = $state<string | null>(null);

	// Per-row busy flags
	let pinBusy = $state<string | null>(null);
	let deleteBusy = $state<string | null>(null);

	// Reload when workspace changes
	let wsId = $state<string | null | undefined>(null);
	$effect(() => {
		const id = $currentWorkspace?.id ?? null;
		if (id !== wsId) { wsId = id; load(); }
	});

	onMount(load);

	async function load() {
		loading = true; error = null;
		try {
			boards = await listBoards();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load boards';
		} finally {
			loading = false;
		}
	}

	async function openCreate() {
		showCreate = true;
		if (!clientsLoaded && !clientsLoading) {
			clientsLoading = true; clientsError = null;
			try {
				clients = await getClients();
				clientsLoaded = true;
			} catch (e) {
				clientsError = e instanceof Error ? e.message : 'Failed to load clients';
			} finally {
				clientsLoading = false;
			}
		}
	}

	function closeCreate() {
		showCreate = false;
		resetForm();
	}

	function resetForm() {
		formName = '';
		formClientId = '';
		selectedViews = [];
		formError = null;
	}

	function toggleView(view: BoardView) {
		if (selectedViews.includes(view)) {
			selectedViews = selectedViews.filter((v) => v !== view);
		} else {
			selectedViews = [...selectedViews, view];
		}
	}

	function viewOrder(view: BoardView): number {
		return selectedViews.indexOf(view) + 1;
	}

	async function handleCreate(e: Event) {
		e.preventDefault();
		formError = null;

		const name = formName.trim();
		if (!name) { formError = 'Name is required.'; return; }
		if (selectedViews.length === 0) { formError = 'Pick at least one view.'; return; }

		const clientId = formClientId || undefined;
		const layout: BoardLayoutEntry[] = selectedViews.map((view) => ({
			view,
			...(clientId ? { filters: { client_id: clientId } } : {})
		}));

		creating = true;
		try {
			const created = await createBoard({
				name,
				kind: clientId ? 'client' : 'board',
				...(clientId ? { subject_type: 'client', subject_id: clientId } : {}),
				layout,
				is_pinned: false
			});
			showCreate = false;
			resetForm();
			goto(`/boards/${created.id}`);
		} catch (e) {
			formError = e instanceof Error ? e.message : 'Failed to create board';
		} finally {
			creating = false;
		}
	}

	async function togglePin(board: Board) {
		if (pinBusy) return;
		pinBusy = board.id;
		try {
			const updated = await pinBoard(board.id, !board.is_pinned);
			boards = boards.map((b) =>
				b.id === board.id ? { ...b, is_pinned: updated?.is_pinned ?? !board.is_pinned } : b
			);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to update pin';
		} finally {
			pinBusy = null;
		}
	}

	async function handleDelete(board: Board) {
		if (deleteBusy) return;
		if (!confirm(`Delete board "${board.name}"? This removes the board layout only; your module data is untouched.`)) return;
		deleteBusy = board.id;
		try {
			await deleteBoard(board.id);
			boards = boards.filter((b) => b.id !== board.id);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to delete board';
		} finally {
			deleteBusy = null;
		}
	}

	function viewSummary(board: Board): string {
		if (!Array.isArray(board.layout) || board.layout.length === 0) return 'No views';
		return board.layout
			.map((entry) => VIEW_OPTIONS.find((o) => o.value === entry.view)?.label ?? entry.view)
			.join(' · ');
	}

	function kindLabel(board: Board): string {
		if (board.subject_type === 'client' && board.subject_id) return 'Client board';
		return board.kind === 'board' ? 'Board' : board.kind;
	}
</script>

<svelte:head><title>Boards - BusinessOS</title></svelte:head>

<div class="page">
	<header class="topbar">
		<div class="title-wrap">
			<SquareKanban size={20} strokeWidth={1.8} />
			<h1>Boards</h1>
			{#if !loading}<span class="count">{boards.length}</span>{/if}
		</div>
		<button class="btn btn--primary" onclick={openCreate}>
			<Plus size={15} strokeWidth={2.4} />New board
		</button>
	</header>

	{#if error}
		<div class="banner banner--error">{error}</div>
	{/if}

	{#if loading}
		<div class="center"><Loader2 class="spin" size={20} />Loading boards...</div>
	{:else if boards.length === 0}
		<div class="empty">
			<SquareKanban size={36} strokeWidth={1.3} />
			<p class="empty-title">No boards yet</p>
			<p class="empty-body">
				A board composes views of your modules (projects, tasks, team, deals, clients)
				on one surface. Optionally filter every view to a single client, and pin the
				board to the sidebar so it works like a module.
			</p>
			<button class="btn btn--primary" onclick={openCreate}>
				<Plus size={15} />Create your first board
			</button>
		</div>
	{:else}
		<div class="grid">
			{#each boards as b (b.id)}
				<div class="bcard">
					<button
						class="bcard__main"
						onclick={() => goto(`/boards/${b.id}`)}
						aria-label="Open {b.name}"
					>
						<span class="bcard__name">{b.name}</span>
						<span class="bcard__meta">
							<span class="bcard__badge">{kindLabel(b)}</span>
							<span class="bcard__views">{viewSummary(b)}</span>
						</span>
					</button>
					<div class="bcard__actions">
						<button
							class="icon-btn"
							class:icon-btn--pinned={b.is_pinned}
							onclick={() => togglePin(b)}
							disabled={pinBusy === b.id}
							aria-label={b.is_pinned ? 'Unpin from sidebar' : 'Pin to sidebar'}
							title={b.is_pinned ? 'Unpin from sidebar' : 'Pin to sidebar'}
						>
							{#if pinBusy === b.id}
								<Loader2 class="spin" size={14} />
							{:else}
								<Star size={14} fill={b.is_pinned ? 'currentColor' : 'none'} />
							{/if}
						</button>
						<button
							class="icon-btn icon-btn--del"
							onclick={() => handleDelete(b)}
							disabled={deleteBusy === b.id}
							aria-label="Delete {b.name}"
							title="Delete board"
						>
							{#if deleteBusy === b.id}
								<Loader2 class="spin" size={14} />
							{:else}
								<Trash2 size={14} />
							{/if}
						</button>
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>

{#if showCreate}
	<div
		class="overlay"
		role="button"
		tabindex="0"
		onclick={closeCreate}
		onkeydown={(e) => e.key === 'Escape' && closeCreate()}
	>
		<div
			class="modal"
			role="dialog"
			aria-modal="true"
			aria-label="Create board"
			tabindex="-1"
			onclick={(e) => e.stopPropagation()}
			onkeydown={() => {}}
		>
			<div class="modal-head">
				<h2>New board</h2>
				<button class="icon-btn" onclick={closeCreate} aria-label="Close">
					<X size={18} />
				</button>
			</div>

			<form onsubmit={handleCreate}>
				{#if formError}
					<div class="banner banner--error" style="margin-bottom:14px">{formError}</div>
				{/if}

				<label class="field">
					<span>Name</span>
					<input bind:value={formName} placeholder="e.g. Operations HQ" required />
				</label>

				<label class="field">
					<span>Context (optional)</span>
					{#if clientsLoading}
						<div class="field-note"><Loader2 class="spin" size={13} />Loading clients...</div>
					{:else if clientsError}
						<div class="field-note field-note--warn">Could not load clients ({clientsError}). You can still create a board without a context.</div>
					{:else}
						<select bind:value={formClientId} disabled={clients.length === 0}>
							<option value="">No context - show everything</option>
							{#each clients as c (c.id)}
								<option value={c.id}>{c.name}</option>
							{/each}
						</select>
						{#if clientsLoaded && clients.length === 0}
							<div class="field-note">No clients yet. Add one in Relationships to build a client-scoped board.</div>
						{/if}
					{/if}
					<span class="field-help">Picking a client filters every view on the board to that client.</span>
				</label>

				<div class="views-section">
					<div class="views-header">
						<span>Views</span>
						<span class="views-hint">Order on the board = the order you check them</span>
					</div>
					<div class="views-list">
						{#each VIEW_OPTIONS as opt (opt.value)}
							{@const idx = viewOrder(opt.value)}
							<label class="view-row" class:view-row--on={idx > 0}>
								<input
									type="checkbox"
									checked={idx > 0}
									onchange={() => toggleView(opt.value)}
								/>
								<span class="view-row__label">{opt.label}</span>
								<span class="view-row__hint">{opt.hint}</span>
								{#if idx > 0}
									<span class="view-row__order">{idx}</span>
								{/if}
							</label>
						{/each}
					</div>
				</div>

				<div class="modal-actions">
					<button type="button" class="btn btn--ghost" onclick={closeCreate}>Cancel</button>
					<button
						type="submit"
						class="btn btn--primary"
						disabled={creating || !formName.trim() || selectedViews.length === 0}
					>
						{#if creating}<Loader2 class="spin" size={14} />{/if}
						Create board
					</button>
				</div>
			</form>
		</div>
	</div>
{/if}

<style>
	.page { height: 100%; display: flex; flex-direction: column; background: var(--dbg); color: var(--dt); overflow: hidden; }

	/* topbar */
	.topbar { display: flex; align-items: center; justify-content: space-between; gap: 16px; padding: 18px 24px; border-bottom: 1px solid var(--dbd); flex-shrink: 0; }
	.title-wrap { display: flex; align-items: center; gap: 10px; color: var(--dt2); }
	.title-wrap h1 { font-size: 1.15rem; font-weight: 680; letter-spacing: -0.02em; margin: 0; color: var(--dt); }
	.count { font-size: 0.74rem; color: var(--dt3); background: color-mix(in srgb, var(--dt) 8%, transparent); padding: 2px 9px; border-radius: 999px; }

	/* grid */
	.grid { flex: 1; overflow-y: auto; display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 12px; padding: 20px 24px; align-content: start; }

	/* board card */
	.bcard { display: flex; align-items: stretch; gap: 4px; border-radius: 12px; border: 1px solid var(--dbd2); background: var(--dbg); transition: border-color 0.15s, box-shadow 0.15s; }
	.bcard:hover { border-color: var(--dbd); box-shadow: 0 2px 12px rgba(0,0,0,0.07); }
	.bcard__main { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 6px; padding: 14px 4px 14px 16px; background: transparent; border: none; cursor: pointer; text-align: left; color: inherit; font-family: inherit; }
	.bcard__name { font-size: 0.9rem; font-weight: 620; color: var(--dt); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
	.bcard__meta { display: flex; align-items: center; gap: 8px; min-width: 0; }
	.bcard__badge { font-size: 0.68rem; font-weight: 600; padding: 2px 7px; border-radius: 999px; background: color-mix(in srgb, var(--dt) 8%, transparent); color: var(--dt3); flex-shrink: 0; white-space: nowrap; }
	.bcard__views { font-size: 0.74rem; color: var(--dt3); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
	.bcard__actions { display: flex; flex-direction: column; justify-content: center; gap: 2px; padding: 8px 10px 8px 0; flex-shrink: 0; }

	/* shared states */
	.center { flex: 1; display: flex; align-items: center; justify-content: center; gap: 8px; color: var(--dt3); font-size: 0.88rem; }
	.empty { flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 10px; padding: 64px 24px; text-align: center; color: var(--dt3); }
	.empty-title { font-size: 0.95rem; font-weight: 600; color: var(--dt2); margin: 4px 0 0; }
	.empty-body { font-size: 0.84rem; max-width: 400px; margin: 0; line-height: 1.55; }
	.banner { padding: 10px 14px; border-radius: 10px; font-size: 0.82rem; }
	.banner--error { background: color-mix(in srgb, #ef4444 12%, transparent); color: #ef4444; margin: 12px 24px 0; }

	/* buttons */
	.btn { display: inline-flex; align-items: center; gap: 7px; padding: 8px 15px; border-radius: 9px; font-size: 0.83rem; font-weight: 580; cursor: pointer; border: 1px solid transparent; white-space: nowrap; font-family: inherit; }
	.btn--primary { background: var(--dt); color: var(--dbg); }
	.btn--ghost { background: transparent; border-color: var(--dbd); color: var(--dt2); }
	.btn:disabled { opacity: 0.5; cursor: not-allowed; }
	.icon-btn { display: inline-flex; align-items: center; justify-content: center; width: 30px; height: 30px; border-radius: 7px; border: none; background: transparent; color: var(--dt3); cursor: pointer; flex-shrink: 0; }
	.icon-btn:hover { background: color-mix(in srgb, var(--dt) 8%, transparent); color: var(--dt); }
	.icon-btn--pinned { color: var(--dt); }
	.icon-btn--del:hover { background: color-mix(in srgb, #ef4444 12%, transparent); color: #ef4444; }
	.icon-btn:disabled { opacity: 0.4; cursor: not-allowed; }

	/* modal */
	.overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.55); display: flex; align-items: center; justify-content: center; z-index: 100; padding: 20px; }
	.modal { width: 100%; max-width: 520px; background: var(--dbg); border: 1px solid var(--dbd); border-radius: 16px; padding: 22px; box-shadow: 0 24px 60px rgba(0,0,0,0.5); max-height: 90vh; overflow-y: auto; }
	.modal-head { display: flex; align-items: center; justify-content: space-between; margin-bottom: 18px; }
	.modal-head h2 { font-size: 1.05rem; font-weight: 640; margin: 0; color: var(--dt); }
	.modal-actions { display: flex; justify-content: flex-end; gap: 10px; margin-top: 18px; }

	/* form */
	.field { display: flex; flex-direction: column; gap: 5px; margin-bottom: 14px; }
	.field > span:first-child { font-size: 0.78rem; font-weight: 560; color: var(--dt2); }
	.field input, .field select { padding: 8px 11px; border-radius: 8px; border: 1px solid var(--dbd); background: color-mix(in srgb, var(--dt) 3%, transparent); color: var(--dt); font-size: 0.85rem; outline: none; font-family: inherit; min-height: 40px; }
	.field input:focus, .field select:focus { border-color: color-mix(in srgb, var(--dt) 40%, transparent); }
	.field select:disabled { opacity: 0.55; cursor: not-allowed; }
	.field-help { font-size: 0.72rem; color: var(--dt3); }
	.field-note { display: flex; align-items: center; gap: 7px; font-size: 0.78rem; color: var(--dt3); padding: 8px 2px; }
	.field-note--warn { color: #f59e0b; }

	/* views section */
	.views-section { margin-bottom: 4px; }
	.views-header { display: flex; align-items: baseline; justify-content: space-between; gap: 10px; margin-bottom: 10px; }
	.views-header > span:first-child { font-size: 0.78rem; font-weight: 560; color: var(--dt2); }
	.views-hint { font-size: 0.7rem; color: var(--dt3); }
	.views-list { display: flex; flex-direction: column; gap: 6px; }
	.view-row { display: flex; align-items: center; gap: 10px; padding: 9px 12px; border-radius: 9px; border: 1px solid var(--dbd2); cursor: pointer; transition: border-color 0.15s, background 0.15s; }
	.view-row:hover { border-color: var(--dbd); }
	.view-row--on { border-color: var(--dbd); background: color-mix(in srgb, var(--dt) 4%, transparent); }
	.view-row input[type='checkbox'] { accent-color: var(--dt); width: 15px; height: 15px; margin: 0; flex-shrink: 0; cursor: pointer; }
	.view-row__label { font-size: 0.84rem; font-weight: 580; color: var(--dt); flex-shrink: 0; }
	.view-row__hint { flex: 1; font-size: 0.74rem; color: var(--dt3); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
	.view-row__order { font-size: 0.7rem; font-weight: 640; color: var(--dt2); background: color-mix(in srgb, var(--dt) 10%, transparent); width: 20px; height: 20px; border-radius: 999px; display: inline-flex; align-items: center; justify-content: center; flex-shrink: 0; }

	/* global */
	:global(.spin) { animation: spin 0.9s linear infinite; }
	@keyframes spin { to { transform: rotate(360deg); } }

	/* 768px */
	@media (max-width: 768px) {
		.topbar { padding: 14px 16px; }
		.grid { padding: 14px 16px; grid-template-columns: repeat(auto-fill, minmax(230px, 1fr)); gap: 10px; }
		.banner--error { margin: 10px 16px 0; }
	}

	/* 480px */
	@media (max-width: 480px) {
		.topbar { padding: 12px 14px; }
		.grid { grid-template-columns: 1fr; padding: 12px 14px; }
		.btn.btn--primary { min-height: 44px; }
		.overlay { align-items: flex-end; padding: 0; }
		.modal { max-width: 100%; border-radius: 20px 20px 0 0; padding: 20px 16px 28px; }
	}
</style>
