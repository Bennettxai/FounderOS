<!-- The workflow builder modal (FounderOS v1 components/WorkflowBuilder.tsx):
     create / edit / delete, index-based branches, and a "Draft" assistant
     that calls /pages/workflows/draft ONLY when the button is pressed. A
     draft only fills the form; nothing saves until Save. -->
<script lang="ts">
	import { onMount, untrack } from 'svelte';
	import { founderosFetch } from '$lib/founderos/api';
	import type { AutomationState, DraftResult, OwnerKind, SimpleAgent, Workflow } from './types';
	import { builderPayload, emptyStep, stepsFromDraft, stepsFromWorkflow, TOOL_IDS, toolName, validateBuilder, type BuilderStep } from './workflows';

	let { agents, workflow, onclose, onsaved }: { agents: SimpleAgent[]; workflow: Workflow | null; onclose: () => void; onsaved?: () => void } = $props();

	// The form is seeded once from the workflow being edited (a snapshot, on purpose).
	const init = untrack(() => ({ workflow, agents }));
	const editing = init.workflow !== null;
	let name = $state(init.workflow?.name ?? '');
	let subtitle = $state(init.workflow?.subtitle ?? '');
	let steps = $state<BuilderStep[]>(init.workflow ? stepsFromWorkflow(init.workflow, init.agents) : [emptyStep()]);
	let busy = $state(false);
	let error = $state<string | null>(null);
	let deleteArmed = $state(false);
	let deleting = $state(false);
	let chatPrompt = $state('');
	let chatBusy = $state(false);
	let chatError = $state<string | null>(null);
	let chatUnavailable = $state(false);

	onMount(() => {
		const onKey = (e: KeyboardEvent) => e.key === 'Escape' && onclose();
		window.addEventListener('keydown', onKey);
		return () => window.removeEventListener('keydown', onKey);
	});

	function update(key: string, patch: Partial<BuilderStep>) {
		steps = steps.map((s) => (s.key === key ? { ...s, ...patch } : s));
	}
	function removeStep(key: string) {
		steps = steps.filter((s) => s.key !== key).map((s) => (s.branchFromKey === key ? { ...s, branchFromKey: '', branchCondition: '' } : s));
	}
	function toggleTool(key: string, tool: string) {
		steps = steps.map((s) => (s.key === key ? { ...s, tools: s.tools.includes(tool) ? s.tools.filter((t) => t !== tool) : [...s.tools, tool] } : s));
	}

	async function submit(e: SubmitEvent) {
		e.preventDefault();
		const v = validateBuilder(name, steps);
		if (v) {
			error = v;
			return;
		}
		error = null;
		busy = true;
		try {
			await founderosFetch(editing ? `/pages/workflows/${workflow!.id}` : '/pages/workflows', {
				method: editing ? 'PATCH' : 'POST',
				json: builderPayload(name, subtitle, steps, agents)
			});
			onsaved?.();
			onclose();
		} catch (err) {
			error = err instanceof Error ? err.message : String(err);
		} finally {
			busy = false;
		}
	}

	async function confirmDelete() {
		if (!workflow) return;
		if (!deleteArmed) {
			deleteArmed = true;
			return;
		}
		deleting = true;
		error = null;
		try {
			await founderosFetch(`/pages/workflows/${workflow.id}`, { method: 'DELETE' });
			onsaved?.();
			onclose();
		} catch (err) {
			error = err instanceof Error ? err.message : String(err);
			deleteArmed = false;
		} finally {
			deleting = false;
		}
	}

	async function draft() {
		const prompt = chatPrompt.trim();
		if (!prompt) return;
		chatBusy = true;
		chatError = null;
		chatUnavailable = false;
		try {
			const body = await founderosFetch<DraftResult>('/pages/workflows/draft', { method: 'POST', json: { prompt } });
			if (!body.ok || !body.draft) {
				chatUnavailable = Boolean(body.unavailable);
				chatError = body.error ?? 'drafting failed';
				return;
			}
			name = body.draft.name;
			subtitle = body.draft.subtitle;
			steps = stepsFromDraft(body.draft, agents);
		} catch (err) {
			chatError = err instanceof Error ? err.message : String(err);
		} finally {
			chatBusy = false;
		}
	}

	const field = 'bn-text border px-2.5 py-2 text-[12.5px] outline-none';
	const fieldStyle = 'border-color: var(--bn-border); background: var(--bn-bg)';
	const lab = 'bn-dim font-mono text-[10px] font-bold uppercase tracking-[0.26em]';
</script>

<!-- svelte-ignore a11y_click_events_have_key_events -->
<div class="fixed inset-0 z-[100] flex items-center justify-center p-6" style="background: color-mix(in oklab, var(--bn-bg) 70%, transparent); backdrop-filter: blur(12px)" role="presentation" onclick={onclose}>
	<!-- svelte-ignore a11y_click_events_have_key_events -->
	<div class="bn-surface relative flex w-full max-w-[820px] flex-col overflow-hidden border" style="max-height: 90vh; border-color: var(--bn-border-strong)" role="dialog" aria-modal="true" aria-label={editing ? `Edit ${workflow!.name}` : 'New workflow'} tabindex="-1" onclick={(e) => e.stopPropagation()}>
		<button type="button" aria-label="Close" class="bn-dim absolute right-4 top-4 z-10 border px-2 py-0.5 font-mono text-[12px]" style="border-color: var(--bn-border)" onclick={onclose}>✕</button>
		<form class="flex flex-1 flex-col overflow-hidden" onsubmit={submit}>
			<div class="flex-1 overflow-y-auto px-7 py-7">
				<h2 class="bn-text pr-10 text-[18px] font-medium leading-tight">{editing ? 'Edit workflow' : 'New workflow'}</h2>

				<section class="mt-5">
					<span class={lab}>Draft it for you</span>
					<div class="mt-2.5 flex items-center gap-2">
						<input bind:value={chatPrompt} placeholder="Describe the workflow and I'll draft it…" class="{field} w-full flex-1" style={fieldStyle} onkeydown={(e) => { if (e.key === 'Enter') { e.preventDefault(); draft(); } }} />
						<button type="button" class="bn-text shrink-0 border px-3 py-2 font-mono text-[11px] disabled:opacity-40" style="border-color: var(--bn-border-strong)" disabled={chatBusy || !chatPrompt.trim()} onclick={draft}>✦ {chatBusy ? 'Drafting…' : 'Draft'}</button>
					</div>
					{#if chatError}<p data-part="draft-error" class="bn-dim mt-2 text-[12px]">{chatUnavailable ? 'Drafting assistant unavailable: build manually.' : chatError}</p>{/if}
					<p class="bn-dim mt-2 text-[11px]">Fills the form below for you to review: nothing saves until you hit Save.</p>
				</section>

				<section class="mt-6 grid gap-3 sm:grid-cols-2">
					<label class="flex flex-col gap-1.5"><span class={lab}>Name</span><input bind:value={name} placeholder="e.g. Renewal outreach" class={field} style={fieldStyle} /></label>
					<label class="flex flex-col gap-1.5"><span class={lab}>Trigger / description</span><input bind:value={subtitle} placeholder="What kicks this off, in one line" class={field} style={fieldStyle} /></label>
				</section>

				<section class="mt-6">
					<div class="flex items-center justify-between">
						<span class={lab}>Steps</span>
						<button type="button" class="bn-muted border px-2.5 py-1 font-mono text-[10.5px]" style="border-color: var(--bn-border)" onclick={() => (steps = [...steps, emptyStep()])}>+ Add step</button>
					</div>
					<div class="mt-3 flex flex-col gap-3">
						{#each steps as s, i (s.key)}
							<div data-builder-step={i} class="flex flex-col gap-3 border px-4 py-4" style="border-color: var(--bn-border)">
								<div class="flex items-center justify-between gap-2">
									<span class="bn-dim font-mono text-[10.5px]">step {i + 1}</span>
									{#if steps.length > 1}<button type="button" aria-label="Remove step {i + 1}" class="bn-dim font-mono text-[11px]" onclick={() => removeStep(s.key)}>✕</button>{/if}
								</div>
								<div class="grid gap-2.5 sm:grid-cols-2">
									<label class="flex flex-col gap-1"><span class="bn-dim text-[10.5px]">Title</span><input value={s.title} oninput={(e) => update(s.key, { title: e.currentTarget.value })} placeholder="What happens in this step" class={field} style={fieldStyle} /></label>
									<div class="flex flex-col gap-1">
										<span class="bn-dim text-[10.5px]">Owner</span>
										<div class="flex items-center gap-1.5">
											<select aria-label="Owner kind" value={s.ownerKind} onchange={(e) => update(s.key, { ownerKind: e.currentTarget.value as OwnerKind })} class="{field} shrink-0" style={fieldStyle}>
												<option value="agent">Agent</option><option value="human">Human</option>
											</select>
											{#if s.ownerKind === 'agent'}
												<select aria-label="Owner agent" value={s.ownerAgentId} onchange={(e) => update(s.key, { ownerAgentId: e.currentTarget.value })} class="{field} min-w-0 flex-1" style={fieldStyle}>
													<option value="">Pick an agent…</option>
													{#each agents as a (a.id)}<option value={a.id}>{a.name}</option>{/each}
												</select>
											{:else}
												<input value={s.ownerHumanName} oninput={(e) => update(s.key, { ownerHumanName: e.currentTarget.value })} placeholder="Person's name" class="{field} min-w-0 flex-1" style={fieldStyle} />
											{/if}
										</div>
									</div>
								</div>
								<label class="flex flex-col gap-1"><span class="bn-dim text-[10.5px]">Detail</span><textarea rows="2" value={s.detail} oninput={(e) => update(s.key, { detail: e.currentTarget.value })} placeholder="One or two sentences: what actually happens here" class="{field} resize-none" style={fieldStyle}></textarea></label>
								<div class="grid gap-2.5 sm:grid-cols-[100px_1fr]">
									<label class="flex flex-col gap-1"><span class="bn-dim text-[10.5px]">Hours / wk</span><input type="number" min="0" step="0.5" value={s.hoursPerWeek} oninput={(e) => update(s.key, { hoursPerWeek: e.currentTarget.value })} class={field} style={fieldStyle} /></label>
									<div class="flex flex-col gap-1">
										<span class="bn-dim text-[10.5px]">Tools</span>
										<div class="flex flex-wrap gap-1.5">
											{#each TOOL_IDS as t (t)}
												{@const on = s.tools.includes(t)}
												<button type="button" aria-pressed={on} class="border px-2 py-[3px] text-[10.5px] {on ? 'bn-accent' : 'bn-dim'}" style="border-color: {on ? 'var(--bn-accent)' : 'var(--bn-border)'}" onclick={() => toggleTool(s.key, t)}>{toolName(t)}</button>
											{/each}
										</div>
									</div>
								</div>
								<div class="grid gap-2.5 sm:grid-cols-[auto_1fr_1fr]">
									<label class="bn-dim flex items-center gap-2 text-[11.5px]"><input type="checkbox" checked={s.automationOn} onchange={(e) => update(s.key, { automationOn: e.currentTarget.checked })} /> Automated</label>
									{#if s.automationOn}
										<input value={s.automationTitle} oninput={(e) => update(s.key, { automationTitle: e.currentTarget.value })} placeholder="What the automation does" class={field} style={fieldStyle} />
										<select value={s.automationState} onchange={(e) => update(s.key, { automationState: e.currentTarget.value as AutomationState })} class={field} style={fieldStyle}><option value="live">Live</option><option value="suggested">Suggested</option></select>
									{/if}
								</div>
								{#if i > 0}
									<div class="grid gap-2.5 sm:grid-cols-2">
										<label class="flex flex-col gap-1">
											<span class="bn-dim text-[10.5px]">Branches from (optional)</span>
											<select value={s.branchFromKey} onchange={(e) => update(s.key, { branchFromKey: e.currentTarget.value })} class={field} style={fieldStyle}>
												<option value="">No: continues the sequence</option>
												{#each steps.slice(0, i) as prior, j (prior.key)}<option value={prior.key}>step {j + 1}{prior.title ? `: ${prior.title}` : ''}</option>{/each}
											</select>
										</label>
										{#if s.branchFromKey}
											<label class="flex flex-col gap-1"><span class="bn-dim text-[10.5px]">Condition</span><input value={s.branchCondition} oninput={(e) => update(s.key, { branchCondition: e.currentTarget.value })} placeholder={'e.g. "approved", "went quiet"'} class={field} style={fieldStyle} /></label>
										{/if}
									</div>
								{/if}
							</div>
						{/each}
					</div>
				</section>

				{#if error}<p data-part="builder-error" class="bn-muted mt-4 flex items-center gap-2 text-[12.5px]"><span class="bn-dot err"></span>{error}</p>{/if}
			</div>

			<div class="flex shrink-0 items-center justify-between gap-3 border-t px-6 py-4" style="border-color: var(--bn-border)">
				<div>
					{#if editing}
						<button type="button" class="border px-3 py-2 text-[12.5px]" style="color: var(--bn-err); border-color: color-mix(in oklab, var(--bn-err) 35%, transparent)" disabled={deleting} onclick={confirmDelete}>
							✕ {deleting ? 'Deleting…' : deleteArmed ? 'Really delete? Click again' : 'Delete workflow'}
						</button>
					{/if}
				</div>
				<div class="flex items-center gap-2">
					<button type="button" class="bn-muted border px-3 py-2 font-mono text-[11px]" style="border-color: var(--bn-border)" onclick={onclose}>Cancel</button>
					<button type="submit" class="border px-3 py-2 font-mono text-[11px] disabled:opacity-40" style="border-color: var(--bn-accent); background: var(--bn-accent); color: var(--bn-accent-ink)" disabled={busy}>{busy ? 'Saving…' : 'Save'}</button>
				</div>
			</div>
		</form>
	</div>
</div>
