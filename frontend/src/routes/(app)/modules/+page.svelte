<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { getModules, createModule } from '$lib/api/modules';
	import type { CustomModule } from '$lib/types/modules';
	import type { RecordField, RecordFieldType, RecordsConfig } from '$lib/api/modules';
	import { currentWorkspace } from '$lib/stores/workspaces';
	import { Layers, Plus, Loader2, X, Trash2 } from 'lucide-svelte';

	const FIELD_TYPES: { value: RecordFieldType; label: string }[] = [
		{ value: 'text', label: 'Text' },
		{ value: 'longtext', label: 'Long text' },
		{ value: 'number', label: 'Number' },
		{ value: 'date', label: 'Date' },
		{ value: 'select', label: 'Select' },
		{ value: 'checkbox', label: 'Checkbox' }
	];

	let modules = $state<CustomModule[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let showCreate = $state(false);
	let creating = $state(false);

	// Create form state
	let formName = $state('');
	let formIcon = $state('');
	let formFields = $state<RecordField[]>([{ key: '', label: '', type: 'text' }]);
	let formError = $state<string | null>(null);

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
			const res = await getModules();
			modules = res.modules;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load modules';
		} finally {
			loading = false;
		}
	}

	function addField() {
		formFields = [...formFields, { key: '', label: '', type: 'text' }];
	}

	function removeField(i: number) {
		formFields = formFields.filter((_, idx) => idx !== i);
	}

	function updateField(i: number, patch: Partial<RecordField>) {
		formFields = formFields.map((f, idx) => idx === i ? { ...f, ...patch } : f);
	}

	// Auto-derive key from label (snake_case)
	function labelToKey(label: string): string {
		return label.toLowerCase().replace(/\s+/g, '_').replace(/[^a-z0-9_]/g, '');
	}

	function resetForm() {
		formName = '';
		formIcon = '';
		formFields = [{ key: '', label: '', type: 'text' }];
		formError = null;
	}

	async function handleCreate(e: Event) {
		e.preventDefault();
		formError = null;

		const name = formName.trim();
		if (!name) { formError = 'Name is required.'; return; }

		const fields: RecordField[] = formFields
			.filter((f) => f.label.trim())
			.map((f) => ({
				key: f.key.trim() || labelToKey(f.label),
				label: f.label.trim(),
				type: f.type,
				...(f.type === 'select' && f.options ? { options: f.options } : {})
			}));

		if (fields.length === 0) { formError = 'Add at least one field.'; return; }

		// Validate unique keys
		const keys = fields.map((f) => f.key);
		if (new Set(keys).size !== keys.length) { formError = 'Field keys must be unique.'; return; }

		const config: RecordsConfig = { kind: 'records', fields };

		creating = true;
		try {
			const created = await createModule({
				name,
				description: '',
				category: 'custom',
				icon: formIcon.trim() || undefined,
				manifest: {
					name,
					version: '1.0.0',
					description: '',
					author: '',
					category: 'custom',
					actions: []
				},
				config: config as unknown as Record<string, unknown>
			});
			modules = [created, ...modules];
			showCreate = false;
			resetForm();
		} catch (e) {
			formError = e instanceof Error ? e.message : 'Failed to create module';
		} finally {
			creating = false;
		}
	}

	function scopeBadge(m: CustomModule): string {
		const scope = m.share_scope ?? m.visibility;
		if (scope === 'organization' || scope === 'public') return 'Org';
		return 'Workspace';
	}

	function moduleIcon(m: CustomModule): string {
		if (m.icon && m.icon.trim()) return m.icon;
		return '📦';
	}
</script>

<svelte:head><title>Modules - BusinessOS</title></svelte:head>

<div class="page">
	<header class="topbar">
		<div class="title-wrap">
			<Layers size={20} strokeWidth={1.8} />
			<h1>Modules</h1>
			{#if !loading}<span class="count">{modules.length}</span>{/if}
		</div>
		<button class="btn btn--primary" onclick={() => (showCreate = true)}>
			<Plus size={15} strokeWidth={2.4} />New module
		</button>
	</header>

	{#if error}
		<div class="banner banner--error">{error}</div>
	{/if}

	{#if loading}
		<div class="center"><Loader2 class="spin" size={20} />Loading modules...</div>
	{:else if modules.length === 0}
		<div class="empty">
			<Layers size={36} strokeWidth={1.3} />
			<p class="empty-title">No modules yet</p>
			<p class="empty-body">Create a module to define a custom records table for your workspace.</p>
			<button class="btn btn--primary" onclick={() => (showCreate = true)}>
				<Plus size={15} />Create your first module
			</button>
		</div>
	{:else}
		<div class="grid">
			{#each modules as m (m.id)}
				<button
					class="mcard"
					onclick={() => goto(`/modules/${m.id}`)}
					aria-label="Open {m.name}"
				>
					<div class="mcard__icon" aria-hidden="true">{moduleIcon(m)}</div>
					<div class="mcard__body">
						<span class="mcard__name">{m.name}</span>
						{#if m.description}
							<span class="mcard__desc">{m.description}</span>
						{/if}
					</div>
					<span class="mcard__badge">{scopeBadge(m)}</span>
				</button>
			{/each}
		</div>
	{/if}
</div>

{#if showCreate}
	<div
		class="overlay"
		role="button"
		tabindex="0"
		onclick={() => { showCreate = false; resetForm(); }}
		onkeydown={(e) => e.key === 'Escape' && (showCreate = false, resetForm())}
	>
		<div
			class="modal"
			role="dialog"
			aria-modal="true"
			aria-label="Create module"
			tabindex="-1"
			onclick={(e) => e.stopPropagation()}
			onkeydown={() => {}}
		>
			<div class="modal-head">
				<h2>New module</h2>
				<button class="icon-btn" onclick={() => { showCreate = false; resetForm(); }} aria-label="Close">
					<X size={18} />
				</button>
			</div>

			<form onsubmit={handleCreate}>
				{#if formError}
					<div class="banner banner--error" style="margin-bottom:14px">{formError}</div>
				{/if}

				<div class="field-row">
					<label class="field" style="flex:1">
						<span>Name</span>
						<input bind:value={formName} placeholder="e.g. Contacts" required />
					</label>
					<label class="field" style="width:100px">
						<span>Icon (emoji)</span>
						<input bind:value={formIcon} placeholder="📋" maxlength="4" />
					</label>
				</div>

				<div class="fields-section">
					<div class="fields-header">
						<span>Fields</span>
						<button type="button" class="btn btn--ghost btn--sm" onclick={addField}>
							<Plus size={13} />Add field
						</button>
					</div>

					<div class="fields-list">
						{#each formFields as field, i (i)}
							<div class="field-row field-def">
								<label class="field" style="flex:1.2">
									<span class:sr-only={i > 0}>Label</span>
									<input
										value={field.label}
										placeholder="Label"
										oninput={(e) => updateField(i, { label: (e.target as HTMLInputElement).value })}
									/>
								</label>
								<label class="field" style="flex:1">
									<span class:sr-only={i > 0}>Key</span>
									<input
										value={field.key || labelToKey(field.label)}
										placeholder={labelToKey(field.label) || 'key'}
										oninput={(e) => updateField(i, { key: (e.target as HTMLInputElement).value })}
									/>
								</label>
								<label class="field" style="flex:0.9">
									<span class:sr-only={i > 0}>Type</span>
									<select
										value={field.type}
										onchange={(e) => updateField(i, { type: (e.target as HTMLSelectElement).value as RecordFieldType })}
									>
										{#each FIELD_TYPES as ft}
											<option value={ft.value}>{ft.label}</option>
										{/each}
									</select>
								</label>
								<button
									type="button"
									class="icon-btn icon-btn--del"
									onclick={() => removeField(i)}
									aria-label="Remove field"
									disabled={formFields.length === 1}
								>
									<Trash2 size={13} />
								</button>
							</div>
						{/each}
					</div>
				</div>

				<div class="modal-actions">
					<button type="button" class="btn btn--ghost" onclick={() => { showCreate = false; resetForm(); }}>Cancel</button>
					<button type="submit" class="btn btn--primary" disabled={creating || !formName.trim()}>
						{#if creating}<Loader2 class="spin" size={14} />{/if}
						Create module
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
	.grid { flex: 1; overflow-y: auto; display: grid; grid-template-columns: repeat(auto-fill, minmax(240px, 1fr)); gap: 12px; padding: 20px 24px; align-content: start; }

	/* module card */
	.mcard { display: flex; align-items: center; gap: 12px; padding: 14px 16px; border-radius: 12px; border: 1px solid var(--dbd2); background: var(--dbg); cursor: pointer; text-align: left; transition: border-color 0.15s, box-shadow 0.15s; }
	.mcard:hover { border-color: var(--dbd); box-shadow: 0 2px 12px rgba(0,0,0,0.07); }
	.mcard__icon { font-size: 1.5rem; flex-shrink: 0; line-height: 1; width: 36px; text-align: center; }
	.mcard__body { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 2px; }
	.mcard__name { font-size: 0.9rem; font-weight: 620; color: var(--dt); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
	.mcard__desc { font-size: 0.76rem; color: var(--dt3); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
	.mcard__badge { font-size: 0.68rem; font-weight: 600; padding: 2px 7px; border-radius: 999px; background: color-mix(in srgb, var(--dt) 8%, transparent); color: var(--dt3); flex-shrink: 0; white-space: nowrap; }

	/* shared states */
	.center { flex: 1; display: flex; align-items: center; justify-content: center; gap: 8px; color: var(--dt3); font-size: 0.88rem; }
	.empty { flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 10px; padding: 64px 24px; text-align: center; color: var(--dt3); }
	.empty-title { font-size: 0.95rem; font-weight: 600; color: var(--dt2); margin: 4px 0 0; }
	.empty-body { font-size: 0.84rem; max-width: 340px; margin: 0; }
	.banner { padding: 10px 14px; border-radius: 10px; font-size: 0.82rem; }
	.banner--error { background: color-mix(in srgb, #ef4444 12%, transparent); color: #ef4444; margin: 12px 24px 0; }

	/* buttons */
	.btn { display: inline-flex; align-items: center; gap: 7px; padding: 8px 15px; border-radius: 9px; font-size: 0.83rem; font-weight: 580; cursor: pointer; border: 1px solid transparent; white-space: nowrap; font-family: inherit; }
	.btn--primary { background: var(--dt); color: var(--dbg); }
	.btn--ghost { background: transparent; border-color: var(--dbd); color: var(--dt2); }
	.btn--sm { padding: 5px 10px; font-size: 0.78rem; }
	.btn:disabled { opacity: 0.5; cursor: not-allowed; }
	.icon-btn { display: inline-flex; align-items: center; justify-content: center; width: 32px; height: 32px; border-radius: 7px; border: none; background: transparent; color: var(--dt3); cursor: pointer; flex-shrink: 0; }
	.icon-btn:hover { background: color-mix(in srgb, var(--dt) 8%, transparent); color: var(--dt); }
	.icon-btn--del:hover { background: color-mix(in srgb, #ef4444 12%, transparent); color: #ef4444; }
	.icon-btn:disabled { opacity: 0.3; cursor: not-allowed; }

	/* modal */
	.overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.55); display: flex; align-items: center; justify-content: center; z-index: 100; padding: 20px; }
	.modal { width: 100%; max-width: 560px; background: var(--dbg); border: 1px solid var(--dbd); border-radius: 16px; padding: 22px; box-shadow: 0 24px 60px rgba(0,0,0,0.5); max-height: 90vh; overflow-y: auto; }
	.modal-head { display: flex; align-items: center; justify-content: space-between; margin-bottom: 18px; }
	.modal-head h2 { font-size: 1.05rem; font-weight: 640; margin: 0; color: var(--dt); }
	.modal-actions { display: flex; justify-content: flex-end; gap: 10px; margin-top: 18px; }

	/* form */
	.field { display: flex; flex-direction: column; gap: 5px; margin-bottom: 14px; }
	.field span { font-size: 0.78rem; font-weight: 560; color: var(--dt2); }
	.field input, .field select { padding: 8px 11px; border-radius: 8px; border: 1px solid var(--dbd); background: color-mix(in srgb, var(--dt) 3%, transparent); color: var(--dt); font-size: 0.85rem; outline: none; font-family: inherit; min-height: 40px; }
	.field input:focus, .field select:focus { border-color: color-mix(in srgb, var(--dt) 40%, transparent); }
	.field-row { display: flex; gap: 10px; align-items: flex-end; }

	/* fields section */
	.fields-section { margin-bottom: 4px; }
	.fields-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 10px; font-size: 0.78rem; font-weight: 560; color: var(--dt2); }
	.fields-list { display: flex; flex-direction: column; gap: 6px; }
	.field-def { align-items: flex-end; margin-bottom: 0; gap: 8px; }
	.field-def .field { margin-bottom: 0; }

	/* accessibility */
	.sr-only { position: absolute; width: 1px; height: 1px; padding: 0; overflow: hidden; clip: rect(0,0,0,0); white-space: nowrap; border: 0; }

	/* global */
	:global(.spin) { animation: spin 0.9s linear infinite; }
	@keyframes spin { to { transform: rotate(360deg); } }

	/* 768px */
	@media (max-width: 768px) {
		.topbar { padding: 14px 16px; }
		.grid { padding: 14px 16px; grid-template-columns: repeat(auto-fill, minmax(200px, 1fr)); gap: 10px; }
		.banner--error { margin: 10px 16px 0; }
	}

	/* 480px */
	@media (max-width: 480px) {
		.topbar { padding: 12px 14px; }
		.grid { grid-template-columns: 1fr; padding: 12px 14px; }
		.btn.btn--primary { min-height: 44px; }
		.overlay { align-items: flex-end; padding: 0; }
		.modal { max-width: 100%; border-radius: 20px 20px 0 0; padding: 20px 16px 28px; }
		.field-row { flex-direction: column; gap: 0; }
		.field-def { flex-wrap: wrap; }
	}
</style>
