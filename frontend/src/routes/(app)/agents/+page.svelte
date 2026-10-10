<script lang="ts">
	import { onMount } from 'svelte';
	import { currentWorkspace } from '$lib/stores/workspaces';
	import {
		listAgents,
		createAgent,
		updateAgent,
		deleteAgent,
		runAgent,
		listRuns,
		AGENT_MODELS,
		DEFAULT_AGENT_MODEL,
		modelLabel,
		type WorkspaceAgent,
		type WorkspaceAgentRun,
		type AgentInput
	} from '$lib/api/workspaceAgents';
	import {
		Bot,
		Plus,
		Play,
		Loader2,
		Pencil,
		Trash2,
		Cpu,
		Sparkles,
		X,
		Send,
		CircleCheck,
		CircleX,
		History
	} from 'lucide-svelte';

	let agents = $state<WorkspaceAgent[]>([]);
	let aiAvailable = $state(false);
	let loading = $state(true);
	let error = $state<string | null>(null);

	// Reload when the workspace changes.
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
			const r = await listAgents();
			agents = r.agents ?? [];
			aiAvailable = r.ai_available ?? false;
			// Keep the selected agent valid across reloads.
			if (selectedId && !agents.some((a) => a.id === selectedId)) selectedId = null;
			if (!selectedId && agents.length) selectAgent(agents[0].id);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load agents';
		} finally {
			loading = false;
		}
	}

	// ---- Create / edit modal ----
	let modalOpen = $state(false);
	let editingId = $state<string | null>(null);
	let saving = $state(false);
	let form = $state<AgentInput>(blankForm());

	function blankForm(): AgentInput {
		return {
			name: '',
			role: 'Researcher',
			description: '',
			model: DEFAULT_AGENT_MODEL,
			system_prompt: '',
			status: 'active'
		};
	}

	const ROLE_PRESETS = ['Researcher', 'Writer', 'Strategist', 'Triage'];

	function openCreate() {
		editingId = null;
		form = blankForm();
		modalOpen = true;
	}
	function openEdit(a: WorkspaceAgent) {
		editingId = a.id;
		form = {
			name: a.name,
			role: a.role,
			description: a.description,
			model: a.model,
			system_prompt: a.system_prompt,
			status: a.status
		};
		modalOpen = true;
	}
	function closeModal() {
		modalOpen = false;
	}

	async function save() {
		if (!form.name.trim()) {
			error = 'Name is required';
			return;
		}
		saving = true;
		error = null;
		try {
			if (editingId) {
				const updated = await updateAgent(editingId, form);
				agents = agents.map((a) => (a.id === updated.id ? updated : a));
			} else {
				const created = await createAgent(form);
				agents = [created, ...agents];
				selectAgent(created.id);
			}
			modalOpen = false;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to save agent';
		} finally {
			saving = false;
		}
	}

	async function remove(a: WorkspaceAgent) {
		if (!confirm(`Delete agent "${a.name}"? This also removes its run history.`)) return;
		try {
			await deleteAgent(a.id);
			agents = agents.filter((x) => x.id !== a.id);
			if (selectedId === a.id) {
				selectedId = null;
				runs = [];
				if (agents.length) selectAgent(agents[0].id);
			}
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to delete agent';
		}
	}

	// ---- Run panel ----
	let selectedId = $state<string | null>(null);
	let input = $state('');
	let output = $state('');
	let outModel = $state('');
	let running = $state(false);
	let runs = $state<WorkspaceAgentRun[]>([]);

	const selected = $derived(agents.find((a) => a.id === selectedId) ?? null);

	async function selectAgent(id: string) {
		selectedId = id;
		output = '';
		outModel = '';
		await loadRuns(id);
	}

	async function loadRuns(id: string) {
		try {
			const r = await listRuns(id);
			runs = r.runs ?? [];
		} catch {
			runs = [];
		}
	}

	async function run() {
		if (!selected || !input.trim()) return;
		running = true;
		error = null;
		output = '';
		try {
			const r = await runAgent(selected.id, input.trim());
			output = r.output;
			outModel = r.model;
			await loadRuns(selected.id);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Agent run failed';
		} finally {
			running = false;
		}
	}

	function roleClass(role: string): string {
		const r = (role || '').toLowerCase();
		if (r.includes('research')) return 'r-research';
		if (r.includes('writ')) return 'r-writer';
		if (r.includes('strateg')) return 'r-strategist';
		if (r.includes('triage')) return 'r-triage';
		return 'r-default';
	}

	function timeAgo(iso: string | null): string {
		if (!iso) return '';
		const secs = Math.max(0, (Date.now() - new Date(iso).getTime()) / 1000);
		if (secs < 60) return 'just now';
		if (secs < 3600) return `${Math.floor(secs / 60)}m ago`;
		if (secs < 86400) return `${Math.floor(secs / 3600)}h ago`;
		return `${Math.floor(secs / 86400)}d ago`;
	}
</script>

<svelte:head><title>Agents - BusinessOS</title></svelte:head>

<div class="page">
	<header class="page-header">
		<div class="page-icon"><Bot size={22} strokeWidth={1.8} /></div>
		<div class="head-text">
			<h1 class="page-title">Agents</h1>
			<p class="page-desc">Your workspace's AI agents — define a role and prompt, then run them.</p>
		</div>
		<button class="btn btn-primary" onclick={openCreate}><Plus size={15} /> New agent</button>
	</header>

	{#if error}<div class="error-bar">{error}</div>{/if}

	{#if loading}
		<div class="loading"><Loader2 size={22} class="spin" /> Loading your agents…</div>
	{:else if agents.length === 0}
		<div class="empty-state">
			<Bot size={40} strokeWidth={1.4} class="empty-icon" />
			<p class="empty-title">No agents yet</p>
			<p class="empty-body">
				Agents are your workspace's AI workers. Give one a role and a system prompt, pick a Claude
				model, and run it against any input.
			</p>
			<button class="btn btn-primary" onclick={openCreate}><Plus size={15} /> Create your first agent</button>
		</div>
	{:else}
		<div class="layout">
			<!-- Roster -->
			<section class="roster">
				<h2 class="sec-title"><Sparkles size={15} /> Roster <span class="ct">{agents.length}</span></h2>
				<div class="cards">
					{#each agents as a (a.id)}
						<button
							class="card agent {selectedId === a.id ? 'is-selected' : ''}"
							onclick={() => selectAgent(a.id)}
						>
							<div class="a-top">
								<div class="a-avatar"><Bot size={16} strokeWidth={1.8} /></div>
								<div class="a-actions">
									<span
										class="icon-btn"
										role="button"
										tabindex="0"
										title="Edit"
										onclick={(e) => {
											e.stopPropagation();
											openEdit(a);
										}}
										onkeydown={(e) => {
											if (e.key === 'Enter') {
												e.stopPropagation();
												openEdit(a);
											}
										}}
									>
										<Pencil size={13} />
									</span>
									<span
										class="icon-btn danger"
										role="button"
										tabindex="0"
										title="Delete"
										onclick={(e) => {
											e.stopPropagation();
											remove(a);
										}}
										onkeydown={(e) => {
											if (e.key === 'Enter') {
												e.stopPropagation();
												remove(a);
											}
										}}
									>
										<Trash2 size={13} />
									</span>
								</div>
							</div>
							<div class="a-name">{a.name}</div>
							<div class="a-meta">
								<span class="role-badge {roleClass(a.role)}">{a.role || 'Agent'}</span>
								<span class="model-badge"><Cpu size={11} /> {modelLabel(a.model)}</span>
							</div>
							{#if a.description}<p class="a-desc">{a.description}</p>{/if}
						</button>
					{/each}
				</div>
			</section>

			<!-- Run panel -->
			<section class="runner">
				<h2 class="sec-title"><Play size={15} /> Run</h2>
				{#if selected}
					<div class="run-card">
						<div class="run-head">
							<div class="a-avatar sm"><Bot size={15} strokeWidth={1.8} /></div>
							<div class="run-head-text">
								<div class="run-name">{selected.name}</div>
								<div class="a-meta">
									<span class="role-badge {roleClass(selected.role)}">{selected.role || 'Agent'}</span>
									<span class="model-badge"><Cpu size={11} /> {modelLabel(selected.model)}</span>
								</div>
							</div>
						</div>

						<textarea
							class="run-input"
							rows="4"
							placeholder="Give {selected.name} something to work on…"
							bind:value={input}
						></textarea>

						<div class="run-bar">
							{#if !aiAvailable}
								<span class="ai-note">AI is not configured for this workspace.</span>
							{/if}
							<button
								class="btn btn-primary"
								onclick={run}
								disabled={running || !input.trim() || !aiAvailable}
							>
								{#if running}<Loader2 size={15} class="spin" /> Running…{:else}<Send size={15} /> Run agent{/if}
							</button>
						</div>

						{#if output}
							<div class="output">
								<div class="output-head">
									<span class="output-label"><Sparkles size={13} /> Output</span>
									{#if outModel}<span class="ai-meta">{modelLabel(outModel)}</span>{/if}
								</div>
								<div class="output-body">{output}</div>
							</div>
						{/if}

						<!-- Run history -->
						<div class="history">
							<div class="history-head"><History size={13} /> Recent runs</div>
							{#if runs.length}
								{#each runs as r (r.id)}
									<div class="hrun">
										<div class="hrun-top">
											{#if r.status === 'done'}
												<CircleCheck size={13} class="ok" />
											{:else}
												<CircleX size={13} class="bad" />
											{/if}
											<span class="hrun-input">{r.input}</span>
											<span class="hrun-time">{timeAgo(r.created_at)}</span>
										</div>
										<p class="hrun-output">{r.output}</p>
									</div>
								{/each}
							{:else}
								<div class="mini-empty">No runs yet. Send this agent an input above.</div>
							{/if}
						</div>
					</div>
				{:else}
					<div class="mini-empty">Select an agent from the roster to run it.</div>
				{/if}
			</section>
		</div>
	{/if}
</div>

<!-- Create / edit modal -->
{#if modalOpen}
	<div
		class="modal-overlay"
		role="button"
		tabindex="0"
		onclick={closeModal}
		onkeydown={(e) => {
			if (e.key === 'Escape') closeModal();
		}}
	>
		<div
			class="modal"
			role="dialog"
			aria-modal="true"
			tabindex="-1"
			onclick={(e) => e.stopPropagation()}
			onkeydown={() => {}}
		>
			<div class="modal-head">
				<h3 class="modal-title">{editingId ? 'Edit agent' : 'New agent'}</h3>
				<button class="icon-btn" onclick={closeModal} title="Close"><X size={16} /></button>
			</div>

			<div class="modal-body">
				<label class="field">
					<span class="field-label">Name</span>
					<input class="field-input" type="text" placeholder="e.g. Market Researcher" bind:value={form.name} />
				</label>

				<div class="field-row">
					<label class="field">
						<span class="field-label">Role</span>
						<input class="field-input" type="text" list="role-presets" placeholder="Researcher" bind:value={form.role} />
						<datalist id="role-presets">
							{#each ROLE_PRESETS as r}<option value={r}></option>{/each}
						</datalist>
					</label>
					<label class="field">
						<span class="field-label">Model</span>
						<select class="field-input" bind:value={form.model}>
							{#each AGENT_MODELS as m}
								<option value={m.id}>{m.label} — {m.hint}</option>
							{/each}
						</select>
					</label>
				</div>

				<label class="field">
					<span class="field-label">Description</span>
					<input class="field-input" type="text" placeholder="What this agent is for" bind:value={form.description} />
				</label>

				<label class="field">
					<span class="field-label">System prompt</span>
					<textarea
						class="field-input"
						rows="6"
						placeholder="You are a meticulous market researcher. Given a topic, return…"
						bind:value={form.system_prompt}
					></textarea>
				</label>
			</div>

			<div class="modal-foot">
				<button class="btn btn-ghost" onclick={closeModal}>Cancel</button>
				<button class="btn btn-primary" onclick={save} disabled={saving || !form.name.trim()}>
					{#if saving}<Loader2 size={15} class="spin" /> Saving…{:else}{editingId ? 'Save changes' : 'Create agent'}{/if}
				</button>
			</div>
		</div>
	</div>
{/if}

<style>
	.page { display: flex; flex-direction: column; gap: 24px; padding: 28px 32px; height: 100%; overflow-y: auto; }
	.page-header { display: flex; align-items: flex-start; gap: 14px; }
	.page-icon { width: 42px; height: 42px; border-radius: 10px; border: 1px solid var(--dbd); background: color-mix(in srgb, var(--dt) 5%, transparent); color: var(--dt2); display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
	.head-text { flex: 1; min-width: 0; }
	.page-title { font-size: 1.25rem; font-weight: 650; color: var(--dt); letter-spacing: -0.02em; margin: 0; }
	.page-desc { font-size: 0.875rem; color: var(--dt3); margin: 2px 0 0; }

	.btn { display: inline-flex; align-items: center; gap: 6px; padding: 8px 14px; border-radius: 8px; font-size: 0.84rem; font-weight: 550; cursor: pointer; border: 1px solid transparent; white-space: nowrap; }
	.btn-primary { background: var(--bos-v2-button-primary); color: var(--bos-v2-button-pureWhiteText); }
	.btn-primary:hover { opacity: 0.9; }
	.btn-ghost { background: transparent; border: 1px solid var(--dbd); color: var(--dt2); }
	.btn-ghost:hover { background: color-mix(in srgb, var(--dt) 4%, transparent); }
	.btn:disabled { opacity: 0.5; cursor: default; }

	.error-bar { padding: 10px 14px; border-radius: 8px; background: color-mix(in srgb, #ef4444 12%, transparent); color: #ef4444; font-size: 0.83rem; }
	.loading { display: flex; align-items: center; gap: 8px; padding: 48px; justify-content: center; color: var(--dt3); font-size: 0.9rem; }

	.layout { display: grid; grid-template-columns: minmax(0, 1.1fr) minmax(0, 1fr); gap: 24px; align-items: start; }
	section { display: flex; flex-direction: column; gap: 12px; min-width: 0; }
	.sec-title { display: flex; align-items: center; gap: 7px; font-size: 0.82rem; font-weight: 650; text-transform: uppercase; letter-spacing: 0.05em; color: var(--dt2); margin: 0; }
	.ct { font-size: 0.7rem; background: color-mix(in srgb, var(--dt) 8%, transparent); color: var(--dt2); border-radius: 20px; padding: 1px 8px; }

	.cards { display: grid; grid-template-columns: repeat(auto-fill, minmax(220px, 1fr)); gap: 12px; }
	.card.agent { text-align: left; border: 1px solid var(--dbd); border-radius: 12px; padding: 14px 16px; display: flex; flex-direction: column; gap: 8px; background: color-mix(in srgb, var(--dt) 2%, transparent); cursor: pointer; }
	.card.agent:hover { border-color: color-mix(in srgb, var(--dt) 25%, var(--dbd)); }
	.card.agent.is-selected { border-color: var(--dt2); box-shadow: 0 0 0 1px var(--dt2) inset; }
	.a-top { display: flex; align-items: center; justify-content: space-between; }
	.a-avatar { width: 30px; height: 30px; border-radius: 8px; background: color-mix(in srgb, var(--dt) 8%, transparent); color: var(--dt2); display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
	.a-avatar.sm { width: 28px; height: 28px; }
	.a-actions { display: flex; gap: 4px; opacity: 0.7; }
	.icon-btn { display: inline-flex; align-items: center; justify-content: center; width: 24px; height: 24px; border-radius: 6px; border: 1px solid transparent; background: transparent; color: var(--dt3); cursor: pointer; }
	.icon-btn:hover { background: color-mix(in srgb, var(--dt) 8%, transparent); color: var(--dt2); }
	.icon-btn.danger:hover { color: #ef4444; }
	.a-name { font-size: 0.92rem; font-weight: 600; color: var(--dt); }
	.a-meta { display: flex; flex-wrap: wrap; align-items: center; gap: 6px; }
	.role-badge { font-size: 0.68rem; font-weight: 600; text-transform: uppercase; letter-spacing: 0.03em; padding: 2px 7px; border-radius: 5px; background: color-mix(in srgb, var(--dt) 8%, transparent); color: var(--dt2); }
	.role-badge.r-research { background: color-mix(in srgb, #6366f1 16%, transparent); color: #6366f1; }
	.role-badge.r-writer { background: color-mix(in srgb, #10b981 16%, transparent); color: #059669; }
	.role-badge.r-strategist { background: color-mix(in srgb, #f59e0b 18%, transparent); color: #d97706; }
	.role-badge.r-triage { background: color-mix(in srgb, #ef4444 15%, transparent); color: #ef4444; }
	.model-badge { display: inline-flex; align-items: center; gap: 4px; font-size: 0.68rem; color: var(--dt3); border: 1px solid var(--dbd); border-radius: 5px; padding: 2px 6px; }
	.a-desc { font-size: 0.8rem; color: var(--dt3); margin: 0; line-height: 1.4; }

	.run-card { border: 1px solid var(--dbd); border-radius: 12px; padding: 16px; display: flex; flex-direction: column; gap: 14px; background: color-mix(in srgb, var(--dt) 2%, transparent); }
	.run-head { display: flex; align-items: center; gap: 10px; }
	.run-head-text { display: flex; flex-direction: column; gap: 4px; min-width: 0; }
	.run-name { font-size: 0.95rem; font-weight: 650; color: var(--dt); }
	.run-input, .field-input { width: 100%; box-sizing: border-box; border: 1px solid var(--dbd); border-radius: 8px; padding: 10px 12px; font-size: 0.86rem; font-family: inherit; color: var(--dt); background: var(--dbg); resize: vertical; }
	.run-input:focus, .field-input:focus { outline: none; border-color: var(--dt2); }
	.run-bar { display: flex; align-items: center; justify-content: flex-end; gap: 12px; }
	.ai-note { font-size: 0.78rem; color: #d97706; margin-right: auto; }

	.output { border: 1px solid var(--dbd); border-radius: 10px; overflow: hidden; background: color-mix(in srgb, #6366f1 5%, transparent); }
	.output-head { display: flex; align-items: center; justify-content: space-between; padding: 8px 12px; border-bottom: 1px solid var(--dbd); }
	.output-label { display: inline-flex; align-items: center; gap: 6px; font-size: 0.72rem; font-weight: 650; text-transform: uppercase; letter-spacing: 0.04em; color: var(--dt2); }
	.ai-meta { font-size: 0.72rem; color: var(--dt3); }
	.output-body { padding: 12px 14px; font-size: 0.86rem; line-height: 1.55; color: var(--dt2); white-space: pre-wrap; word-break: break-word; max-height: 360px; overflow-y: auto; }

	.history { display: flex; flex-direction: column; gap: 8px; }
	.history-head { display: inline-flex; align-items: center; gap: 6px; font-size: 0.72rem; font-weight: 650; text-transform: uppercase; letter-spacing: 0.04em; color: var(--dt3); }
	.hrun { border: 1px solid var(--dbd); border-radius: 8px; padding: 9px 11px; display: flex; flex-direction: column; gap: 4px; }
	.hrun-top { display: flex; align-items: center; gap: 6px; }
	.hrun-top :global(.ok) { color: #16a34a; }
	.hrun-top :global(.bad) { color: #ef4444; }
	.hrun-input { flex: 1; font-size: 0.8rem; font-weight: 550; color: var(--dt); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
	.hrun-time { font-size: 0.7rem; color: var(--dt3); flex-shrink: 0; }
	.hrun-output { font-size: 0.78rem; color: var(--dt3); margin: 0; line-height: 1.45; display: -webkit-box; -webkit-line-clamp: 3; line-clamp: 3; -webkit-box-orient: vertical; overflow: hidden; }

	.mini-empty { padding: 16px; border: 1px dashed var(--dbd); border-radius: 10px; color: var(--dt3); font-size: 0.85rem; }
	.empty-state { flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 10px; padding: 56px 24px; text-align: center; }
	.empty-state :global(.empty-icon) { color: var(--dt4, var(--dt3)); opacity: 0.5; }
	.empty-title { font-size: 0.95rem; font-weight: 600; color: var(--dt2); margin: 4px 0 0; }
	.empty-body { font-size: 0.84rem; color: var(--dt3); max-width: 420px; margin: 0 0 8px; line-height: 1.5; }

	/* Modal */
	.modal-overlay { position: fixed; inset: 0; background: color-mix(in srgb, #000 45%, transparent); display: flex; align-items: center; justify-content: center; padding: 24px; z-index: 100; }
	.modal { width: 100%; max-width: 520px; max-height: 90vh; overflow-y: auto; background: var(--dbg); border: 1px solid var(--dbd); border-radius: 14px; display: flex; flex-direction: column; }
	.modal-head { display: flex; align-items: center; justify-content: space-between; padding: 16px 18px; border-bottom: 1px solid var(--dbd); }
	.modal-title { font-size: 1rem; font-weight: 650; color: var(--dt); margin: 0; }
	.modal-body { padding: 18px; display: flex; flex-direction: column; gap: 14px; }
	.field { display: flex; flex-direction: column; gap: 6px; }
	.field-row { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; }
	.field-label { font-size: 0.76rem; font-weight: 600; color: var(--dt2); }
	.modal-foot { display: flex; align-items: center; justify-content: flex-end; gap: 10px; padding: 14px 18px; border-top: 1px solid var(--dbd); }

	:global(.spin) { animation: spin 0.8s linear infinite; }
	@keyframes spin { to { transform: rotate(360deg); } }
	@media (max-width: 900px) { .layout { grid-template-columns: 1fr; } }
	@media (max-width: 768px) { .page { padding: 16px 18px; } .cards { grid-template-columns: 1fr; } .field-row { grid-template-columns: 1fr; } }
</style>
