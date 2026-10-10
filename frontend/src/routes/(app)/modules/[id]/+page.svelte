<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { getModule, listRecords, createRecord, updateRecord, deleteRecord } from '$lib/api/modules';
	import type { CustomModule } from '$lib/types/modules';
	import type { RecordField, RecordsConfig, ModuleRecord } from '$lib/api/modules';
	import OperatingModuleView from '$lib/components/modules/OperatingModuleView.svelte';
	import { currentWorkspace } from '$lib/stores/workspaces';
	import { ChevronLeft, Plus, Loader2, X, Trash2, Check, Pencil, Inbox, Layers3 } from 'lucide-svelte';

	const moduleId = $derived($page.params.id ?? '');

	let mod = $state<CustomModule | null>(null);
	let records = $state<ModuleRecord[]>([]);
	let loadingModule = $state(true);
	let loadingRecords = $state(true);
	let error = $state<string | null>(null);
	let missingModule = $state(false);

	// Derived fields from config
	const fields = $derived.by<RecordField[]>(() => {
		if (!mod) return [];
		const cs = (mod.config ?? mod.config_schema) as Record<string, unknown> | null;
		if (cs?.kind === 'records' && Array.isArray(cs.fields)) {
			return cs.fields as RecordField[];
		}
		return [];
	});
	const isOperatingModule = $derived.by(() => {
		const config = (mod?.config ?? mod?.config_schema) as Record<string, unknown> | null;
		return config?.kind === 'operating_module';
	});

	// Add-row state
	let showAdd = $state(false);
	let addData = $state<Record<string, unknown>>({});
	let adding = $state(false);

	// Edit state: maps recordId -> field key -> draft value
	let editingId = $state<string | null>(null);
	let editData = $state<Record<string, unknown>>({});
	let saving = $state(false);
	let deletingId = $state<string | null>(null);

	// Reload on workspace change
	let wsId = $state<string | null | undefined>(null);
	let loadedModuleId = $state<string | null>(null);
	let loadRequest = 0;
	$effect(() => {
		const id = $currentWorkspace?.id ?? null;
		const nextModuleId = moduleId;
		if (id && (id !== wsId || nextModuleId !== loadedModuleId)) {
			wsId = id;
			loadedModuleId = nextModuleId;
			loadAll(id, nextModuleId);
		}
	});

	async function loadAll(workspaceId: string, requestedModuleId: string) {
		const requestId = ++loadRequest;
		loadingModule = true; loadingRecords = true; error = null; missingModule = false;
		try {
			const nextModule = await getModule(requestedModuleId);
			if (requestId !== loadRequest || workspaceId !== $currentWorkspace?.id) return;
			mod = nextModule;
			loadingModule = false;

			const config = (nextModule.config ?? nextModule.config_schema) as Record<string, unknown> | null;
			if (config?.kind === 'operating_module') {
				records = [];
				loadingRecords = false;
				return;
			}

			const nextRecords = await listRecords(requestedModuleId);
			if (requestId === loadRequest && workspaceId === $currentWorkspace?.id) {
				records = nextRecords;
			}
		} catch (e) {
			if (requestId === loadRequest) {
				const message = e instanceof Error ? e.message : 'Failed to load module';
				missingModule = message.includes('HTTP 404');
				error = missingModule ? null : message;
			}
		} finally {
			if (requestId === loadRequest) {
				loadingModule = false;
				loadingRecords = false;
			}
		}
	}

	// ── helpers ──────────────────────────────────────────────────────────────

	function displayValue(field: RecordField, raw: unknown): string {
		if (raw === null || raw === undefined || raw === '') return '';
		if (field.type === 'checkbox') return raw ? 'Yes' : 'No';
		if (field.type === 'date' && typeof raw === 'string') {
			try {
				const d = new Date(raw);
				if (!Number.isNaN(d.getTime())) {
					return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' });
				}
			} catch { /* fall through to raw string */ }
			return String(raw);
		}
		if (field.type === 'number') {
			const n = Number(raw);
			if (Number.isFinite(n)) return n.toLocaleString();
		}
		return String(raw);
	}

	function isEmptyValue(raw: unknown): boolean {
		return raw === null || raw === undefined || raw === '';
	}

	function inputType(field: RecordField): string {
		if (field.type === 'number') return 'number';
		if (field.type === 'date') return 'date';
		if (field.type === 'checkbox') return 'checkbox';
		return 'text';
	}

	function castValue(field: RecordField, raw: string | boolean): unknown {
		if (field.type === 'number') return raw === '' ? null : Number(raw);
		if (field.type === 'checkbox') return Boolean(raw);
		return raw;
	}

	// ── add row ───────────────────────────────────────────────────────────────

	function openAdd() {
		addData = {};
		showAdd = true;
	}

	async function submitAdd(e: Event) {
		e.preventDefault();
		if (!mod) return;
		adding = true; error = null;
		try {
			const created = await createRecord(moduleId, addData);
			records = [...records, created];
			showAdd = false;
			addData = {};
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to create record';
		} finally {
			adding = false;
		}
	}

	// ── edit row ──────────────────────────────────────────────────────────────

	function startEdit(r: ModuleRecord) {
		editingId = r.id;
		editData = { ...r.data };
	}

	function cancelEdit() {
		editingId = null;
		editData = {};
	}

	async function saveEdit(r: ModuleRecord) {
		saving = true; error = null;
		try {
			const updated = await updateRecord(moduleId, r.id, { data: editData });
			records = records.map((x) => x.id === r.id ? updated : x);
			editingId = null;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to save';
		} finally {
			saving = false;
		}
	}

	// ── delete row ────────────────────────────────────────────────────────────

	async function removeRecord(r: ModuleRecord) {
		if (!confirm(`Delete this record?`)) return;
		deletingId = r.id; error = null;
		try {
			await deleteRecord(moduleId, r.id);
			records = records.filter((x) => x.id !== r.id);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to delete';
		} finally {
			deletingId = null;
		}
	}
</script>

<svelte:head>
	<title>{mod?.name ?? 'Module'} - BusinessOS</title>
</svelte:head>

{#if missingModule}
	<div class="page">
		<div class="empty empty--unbuilt">
			<span class="empty-icon empty-icon--large"><Layers3 size={28} strokeWidth={1.45} /></span>
			<p class="empty-eyebrow">Unshaped module</p>
			<h1 class="empty-heading">This part of the business has not been defined yet.</h1>
			<p class="empty-body empty-body--wide">
				A module should begin with a real responsibility, workflow, or decision boundary. Define the work first, then give it the fields and records it needs.
			</p>
			<div class="empty-actions">
				<button class="btn btn--primary" onclick={() => goto('/modules')}>
					<Plus size={15} />Shape this module
				</button>
				<button class="btn btn--ghost" onclick={() => history.back()}>
					<ChevronLeft size={15} />Go back
				</button>
			</div>
		</div>
	</div>
{:else if mod && isOperatingModule}
	{#key mod.id}
		<OperatingModuleView {mod} />
	{/key}
{:else}
<div class="page">
	<!-- Header -->
	<header class="topbar">
		<button class="back-btn" onclick={() => goto('/modules')} aria-label="Back to modules">
			<ChevronLeft size={18} />
		</button>
		<div class="title-wrap">
			{#if mod?.icon}<span class="mod-icon" aria-hidden="true">{mod.icon}</span>{/if}
			<h1>{mod?.name ?? '...'}</h1>
			{#if !loadingRecords}<span class="count">{records.length}</span>{/if}
		</div>
		<button class="btn btn--primary" onclick={openAdd} disabled={loadingModule || fields.length === 0}>
			<Plus size={15} strokeWidth={2.4} />Add row
		</button>
	</header>

	{#if error}
		<div class="banner banner--error">{error}</div>
	{/if}

	{#if loadingModule}
		<div class="center"><Loader2 class="spin" size={20} />Loading...</div>
	{:else if fields.length === 0}
		<div class="empty">
			<p class="empty-title">No fields defined</p>
			<p class="empty-body">This module has no record schema. Edit the module to add fields.</p>
		</div>
	{:else}
		<!-- Table wrapper - scrolls horizontally on small screens -->
		<div class="table-wrap">
			{#if loadingRecords}
				<!-- skeleton rows matching the module's columns -->
				<table class="rtable" aria-hidden="true">
					<thead>
						<tr>
							{#each fields as f}
								<th class:th-num={f.type === 'number'}>{f.label}</th>
							{/each}
							<th class="th-actions"></th>
						</tr>
					</thead>
					<tbody>
						{#each Array(4) as _, i (i)}
							<tr>
								{#each fields as f (f.key)}
									<td><span class="skel" style="width:{f.type === 'checkbox' ? 36 : f.type === 'number' ? 52 : 60 + ((i * 37 + f.key.length * 13) % 80)}px"></span></td>
								{/each}
								<td class="td-actions"></td>
							</tr>
						{/each}
					</tbody>
				</table>
			{:else if records.length === 0}
				<div class="empty" style="padding:60px 24px">
					<span class="empty-icon"><Inbox size={22} strokeWidth={1.6} /></span>
					<p class="empty-title">No records yet</p>
					<p class="empty-body">This module is empty. Click "Add row" to create the first record.</p>
					<button class="btn btn--primary" onclick={openAdd}><Plus size={15} />Add row</button>
				</div>
			{:else}
				<table class="rtable">
					<thead>
						<tr>
							{#each fields as f}
								<th class:th-num={f.type === 'number'}>{f.label}</th>
							{/each}
							<th class="th-actions"></th>
						</tr>
					</thead>
					<tbody>
						{#each records as r (r.id)}
							<tr class:is-editing={editingId === r.id}>
								{#each fields as f}
									<td class:td-num={f.type === 'number'}>
										{#if editingId === r.id}
											<!-- inline edit cell -->
											{#if f.type === 'checkbox'}
												<input
													type="checkbox"
													class="cell-check"
													checked={Boolean(editData[f.key])}
													onchange={(e) => (editData = { ...editData, [f.key]: (e.target as HTMLInputElement).checked })}
													aria-label={f.label}
												/>
											{:else if f.type === 'longtext'}
												<textarea
													class="cell-input"
													rows="2"
													value={String(editData[f.key] ?? '')}
													oninput={(e) => (editData = { ...editData, [f.key]: (e.target as HTMLTextAreaElement).value })}
													aria-label={f.label}
												></textarea>
											{:else if f.type === 'select' && f.options?.length}
												<select
													class="cell-input"
													value={String(editData[f.key] ?? '')}
													onchange={(e) => (editData = { ...editData, [f.key]: (e.target as HTMLSelectElement).value })}
													aria-label={f.label}
												>
													<option value=""></option>
													{#each f.options as opt}
														<option value={opt}>{opt}</option>
													{/each}
												</select>
											{:else}
												<input
													type={inputType(f)}
													class="cell-input"
													value={String(editData[f.key] ?? '')}
													oninput={(e) => (editData = { ...editData, [f.key]: castValue(f, (e.target as HTMLInputElement).value) })}
													aria-label={f.label}
												/>
											{/if}
										{:else}
											<!-- read cell, rendered per field type -->
											{#if f.type === 'checkbox'}
												<span class="cell-bool" class:cell-bool--yes={Boolean(r.data[f.key])}>
													{r.data[f.key] ? 'Yes' : 'No'}
												</span>
											{:else if isEmptyValue(r.data[f.key])}
												<span class="cell-empty" aria-label="Empty">-</span>
											{:else if f.type === 'select'}
												<span class="cell-tag">{displayValue(f, r.data[f.key])}</span>
											{:else if f.type === 'number'}
												<span class="cell-text cell-num">{displayValue(f, r.data[f.key])}</span>
											{:else if f.type === 'date'}
												<span class="cell-text cell-date">{displayValue(f, r.data[f.key])}</span>
											{:else if f.type === 'longtext'}
												<span class="cell-long">{displayValue(f, r.data[f.key])}</span>
											{:else}
												<span class="cell-text">{displayValue(f, r.data[f.key])}</span>
											{/if}
										{/if}
									</td>
								{/each}

								<!-- row actions -->
								<td class="td-actions">
									{#if editingId === r.id}
										<button
											class="icon-btn icon-btn--ok"
											onclick={() => saveEdit(r)}
											disabled={saving}
											aria-label="Save"
										>
											{#if saving}<Loader2 class="spin" size={13} />{:else}<Check size={14} />{/if}
										</button>
										<button class="icon-btn" onclick={cancelEdit} aria-label="Cancel edit">
											<X size={14} />
										</button>
									{:else}
										<button
											class="icon-btn"
											onclick={() => startEdit(r)}
											disabled={deletingId === r.id}
											aria-label="Edit row"
										>
											<Pencil size={13} />
										</button>
										<button
											class="icon-btn icon-btn--del"
											onclick={() => removeRecord(r)}
											disabled={deletingId === r.id}
											aria-label="Delete row"
										>
											{#if deletingId === r.id}<Loader2 class="spin" size={13} />{:else}<Trash2 size={13} />{/if}
										</button>
									{/if}
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			{/if}
		</div>
	{/if}
</div>
{/if}

<!-- Add row modal -->
{#if showAdd && mod}
	<div
		class="overlay"
		role="button"
		tabindex="0"
		onclick={() => (showAdd = false)}
		onkeydown={(e) => e.key === 'Escape' && (showAdd = false)}
	>
		<div
			class="modal"
			role="dialog"
			aria-modal="true"
			aria-label="Add record"
			tabindex="-1"
			onclick={(e) => e.stopPropagation()}
			onkeydown={() => {}}
		>
			<div class="modal-head">
				<h2>Add record</h2>
				<button class="icon-btn" onclick={() => (showAdd = false)} aria-label="Close">
					<X size={18} />
				</button>
			</div>

			<form onsubmit={submitAdd}>
				{#each fields as f}
					<label class="mfield">
						<span>{f.label}</span>
						{#if f.type === 'longtext'}
							<textarea
								rows="3"
								value={String(addData[f.key] ?? '')}
								oninput={(e) => (addData = { ...addData, [f.key]: (e.target as HTMLTextAreaElement).value })}
								placeholder={f.label}
							></textarea>
						{:else if f.type === 'checkbox'}
							<input
								type="checkbox"
								checked={Boolean(addData[f.key])}
								onchange={(e) => (addData = { ...addData, [f.key]: (e.target as HTMLInputElement).checked })}
							/>
						{:else if f.type === 'select' && f.options?.length}
							<select
								value={String(addData[f.key] ?? '')}
								onchange={(e) => (addData = { ...addData, [f.key]: (e.target as HTMLSelectElement).value })}
							>
								<option value=""></option>
								{#each f.options as opt}
									<option value={opt}>{opt}</option>
								{/each}
							</select>
						{:else}
							<input
								type={inputType(f)}
								value={String(addData[f.key] ?? '')}
								oninput={(e) => (addData = { ...addData, [f.key]: castValue(f, (e.target as HTMLInputElement).value) })}
								placeholder={f.label}
							/>
						{/if}
					</label>
				{/each}

				<div class="modal-actions">
					<button type="button" class="btn btn--ghost" onclick={() => (showAdd = false)}>Cancel</button>
					<button type="submit" class="btn btn--primary" disabled={adding}>
						{#if adding}<Loader2 class="spin" size={14} />{/if}
						Add record
					</button>
				</div>
			</form>
		</div>
	</div>
{/if}

<style>
	.page { height: 100%; display: flex; flex-direction: column; background: var(--dbg); color: var(--dt); overflow: hidden; }

	/* topbar */
	.topbar { display: flex; align-items: center; gap: 12px; padding: 14px 20px; border-bottom: 1px solid var(--dbd); flex-shrink: 0; }
	.back-btn { display: inline-flex; align-items: center; justify-content: center; width: 34px; height: 34px; border-radius: 8px; border: 1px solid var(--dbd); background: transparent; color: var(--dt2); cursor: pointer; flex-shrink: 0; }
	.back-btn:hover { background: color-mix(in srgb, var(--dt) 6%, transparent); }
	.title-wrap { display: flex; align-items: center; gap: 9px; flex: 1; min-width: 0; }
	.mod-icon { font-size: 1.25rem; line-height: 1; flex-shrink: 0; }
	.title-wrap h1 { font-size: 1.1rem; font-weight: 680; letter-spacing: -0.02em; margin: 0; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
	.count { font-size: 0.74rem; color: var(--dt3); background: color-mix(in srgb, var(--dt) 8%, transparent); padding: 2px 9px; border-radius: 999px; flex-shrink: 0; }

	/* table */
	.table-wrap { flex: 1; overflow: auto; }
	.rtable { width: 100%; border-collapse: collapse; font-size: 0.85rem; }
	.rtable th { padding: 10px 14px; text-align: left; font-size: 0.7rem; font-weight: 620; text-transform: uppercase; letter-spacing: 0.05em; color: var(--dt3); border-bottom: 1px solid var(--dbd); white-space: nowrap; position: sticky; top: 0; background: var(--dbg); z-index: 1; }
	.rtable td { padding: 10px 14px; border-bottom: 1px solid var(--dbd2); vertical-align: middle; }
	.rtable tr:hover td { background: color-mix(in srgb, var(--dt) 2%, transparent); }
	.rtable tr.is-editing td { background: color-mix(in srgb, var(--dt) 3%, transparent); }
	.th-actions { width: 80px; }
	.td-actions { white-space: nowrap; }

	/* cells */
	.cell-text { color: var(--dt); display: block; max-width: 280px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
	.cell-bool { font-size: 0.75rem; font-weight: 600; padding: 2px 8px; border-radius: 999px; background: color-mix(in srgb, var(--dt) 6%, transparent); color: var(--dt3); }
	.cell-bool--yes { background: color-mix(in srgb, #22c55e 14%, transparent); color: #22c55e; }
	.cell-empty { color: var(--dt3); opacity: 0.6; user-select: none; }
	.cell-tag { display: inline-block; font-size: 0.76rem; font-weight: 560; padding: 2px 9px; border-radius: 999px; background: color-mix(in srgb, var(--dt) 7%, transparent); border: 1px solid var(--dbd); color: var(--dt2); max-width: 200px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; vertical-align: middle; }
	.cell-num { font-variant-numeric: tabular-nums; }
	.cell-date { color: var(--dt2); }
	.cell-long { color: var(--dt); display: -webkit-box; -webkit-line-clamp: 2; line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; max-width: 360px; white-space: normal; line-height: 1.4; }
	.th-num, .td-num { text-align: right; }

	/* loading skeleton */
	.skel { display: inline-block; height: 12px; border-radius: 6px; background: color-mix(in srgb, var(--dt) 7%, transparent); animation: skel-pulse 1.4s ease-in-out infinite; }
	@keyframes skel-pulse { 0%, 100% { opacity: 1; } 50% { opacity: 0.45; } }
	.cell-input { padding: 6px 9px; border-radius: 7px; border: 1px solid var(--dbd); background: var(--dbg); color: var(--dt); font-size: 0.83rem; outline: none; font-family: inherit; width: 100%; min-width: 80px; min-height: 34px; }
	.cell-input:focus { border-color: color-mix(in srgb, var(--dt) 40%, transparent); }
	.cell-check { width: 18px; height: 18px; cursor: pointer; accent-color: var(--dt); }

	/* shared */
	.center { display: flex; align-items: center; justify-content: center; gap: 8px; color: var(--dt3); font-size: 0.88rem; }
	.empty { display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 10px; text-align: center; color: var(--dt3); }
	.empty-icon { display: inline-flex; align-items: center; justify-content: center; width: 44px; height: 44px; border-radius: 12px; border: 1px solid var(--dbd); background: color-mix(in srgb, var(--dt) 4%, transparent); color: var(--dt3); }
	.empty-title { font-size: 0.95rem; font-weight: 600; color: var(--dt2); margin: 4px 0 0; }
	.empty-body { font-size: 0.84rem; max-width: 340px; margin: 0; }
	.empty--unbuilt { max-width: 680px; margin: 0 auto; }
	.empty-icon--large { width: 56px; height: 56px; display: inline-flex; align-items: center; justify-content: center; border: 1px solid var(--dbd); border-radius: 16px; color: var(--dt2); background: color-mix(in srgb, var(--dt) 4%, transparent); }
	.empty-eyebrow { margin: 8px 0 -2px; color: var(--dt3); font-size: 0.7rem; font-weight: 700; letter-spacing: 0.14em; text-transform: uppercase; }
	.empty-heading { max-width: 620px; margin: 0; color: var(--dt); font-size: clamp(1.35rem, 3vw, 2rem); line-height: 1.12; letter-spacing: -0.035em; }
	.empty-body--wide { max-width: 560px; line-height: 1.6; }
	.empty-actions { display: flex; align-items: center; justify-content: center; gap: 10px; margin-top: 10px; flex-wrap: wrap; }
	.banner { padding: 10px 14px; border-radius: 10px; font-size: 0.82rem; }
	.banner--error { background: color-mix(in srgb, #ef4444 12%, transparent); color: #ef4444; margin: 12px 20px 0; }

	/* buttons */
	.btn { display: inline-flex; align-items: center; gap: 7px; padding: 8px 15px; border-radius: 9px; font-size: 0.83rem; font-weight: 580; cursor: pointer; border: 1px solid transparent; white-space: nowrap; font-family: inherit; }
	.btn--primary { background: var(--dt); color: var(--dbg); }
	.btn--ghost { background: transparent; border-color: var(--dbd); color: var(--dt2); }
	.btn:disabled { opacity: 0.5; cursor: not-allowed; }
	.icon-btn { display: inline-flex; align-items: center; justify-content: center; width: 30px; height: 30px; border-radius: 7px; border: none; background: transparent; color: var(--dt3); cursor: pointer; }
	.icon-btn:hover { background: color-mix(in srgb, var(--dt) 8%, transparent); color: var(--dt); }
	.icon-btn--ok:hover { background: color-mix(in srgb, #22c55e 14%, transparent); color: #22c55e; }
	.icon-btn--del:hover { background: color-mix(in srgb, #ef4444 12%, transparent); color: #ef4444; }
	.icon-btn:disabled { opacity: 0.4; cursor: not-allowed; }

	/* modal */
	.overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.55); display: flex; align-items: center; justify-content: center; z-index: 100; padding: 20px; }
	.modal { width: 100%; max-width: 460px; background: var(--dbg); border: 1px solid var(--dbd); border-radius: 16px; padding: 22px; box-shadow: 0 24px 60px rgba(0,0,0,0.5); max-height: 90vh; overflow-y: auto; }
	.modal-head { display: flex; align-items: center; justify-content: space-between; margin-bottom: 18px; }
	.modal-head h2 { font-size: 1.05rem; font-weight: 640; margin: 0; color: var(--dt); }
	.modal-actions { display: flex; justify-content: flex-end; gap: 10px; margin-top: 18px; }
	.mfield { display: flex; flex-direction: column; gap: 5px; margin-bottom: 13px; }
	.mfield span { font-size: 0.78rem; font-weight: 560; color: var(--dt2); }
	.mfield input:not([type='checkbox']), .mfield textarea, .mfield select { padding: 8px 11px; border-radius: 8px; border: 1px solid var(--dbd); background: color-mix(in srgb, var(--dt) 3%, transparent); color: var(--dt); font-size: 0.85rem; outline: none; font-family: inherit; min-height: 40px; width: 100%; }
	.mfield input[type='checkbox'] { width: 18px; height: 18px; cursor: pointer; accent-color: var(--dt); }
	.mfield input:focus, .mfield textarea:focus, .mfield select:focus { border-color: color-mix(in srgb, var(--dt) 40%, transparent); }

	:global(.spin) { animation: spin 0.9s linear infinite; }
	@keyframes spin { to { transform: rotate(360deg); } }

	/* 768px */
	@media (max-width: 768px) {
		.topbar { padding: 12px 16px; }
		.rtable th, .rtable td { padding: 9px 12px; }
	}

	/* 480px */
	@media (max-width: 480px) {
		.topbar { padding: 10px 12px; }
		.title-wrap h1 { font-size: 1rem; }
		.btn.btn--primary { min-height: 44px; }
		/* table scrolls, no stacking - keep columns intact but rely on table-wrap scroll */
		.rtable { min-width: 480px; }
		.overlay { align-items: flex-end; padding: 0; }
		.modal { max-width: 100%; border-radius: 20px 20px 0 0; padding: 20px 16px 28px; }
	}
</style>
