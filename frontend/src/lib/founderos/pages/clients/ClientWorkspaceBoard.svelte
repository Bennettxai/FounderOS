<!-- The client board (FounderOS v1 components/ClientWorkspaceBoard.tsx): the
     roster, the client context, the brief form and the request list as
     SlabCards on the stagger after the page's hero (i 6..9). Saving never
     launches; launching needs the operator token and a confirm, and the
     bridge refuses it while FOUNDEROS_WRITES=0. Nothing publishes automatically. -->
<script lang="ts">
	import { founderosFetch, FounderosApiError } from '$lib/founderos/api';
	import { SlabCard, chipClass } from '$lib/founderos/kit';
	import { statusStyle, type ClientProject, type ClientWork, type StatusStep, type WorkStatus } from './types';

	let {
		clients,
		work,
		statusOrder,
		launchEnabled = false,
		onchange
	}: {
		clients: ClientProject[];
		work: ClientWork[];
		statusOrder: StatusStep[];
		launchEnabled?: boolean;
		onchange?: () => void | Promise<void>;
	} = $props();

	type Filter = 'all' | WorkStatus;

	let clientId = $state('');
	let brief = $state('');
	let token = $state('');
	let busy = $state(false);
	let message = $state('');
	let filter = $state<Filter>('all');

	const selected = $derived(clients.find((c) => c.id === clientId) ?? clients[0]);
	const mine = $derived(selected ? work.filter((w) => w.clientId === selected.id) : []);
	const shown = $derived(filter === 'all' ? mine : mine.filter((w) => w.status === filter));
	const count = (s: WorkStatus) => mine.filter((w) => w.status === s).length;

	async function submit(body: object) {
		busy = true;
		message = '';
		try {
			const data = await founderosFetch<{ work?: ClientWork }>('/pages/clients/work', {
				method: 'POST',
				json: body,
				headers: token ? { Authorization: `Bearer ${token}` } : undefined
			});
			message =
				data.work?.status === 'launched'
					? 'Agent launched in Superset. Review the workspace for results; nothing is published automatically.'
					: 'Saved. Launching an agent is a separate action; nothing is published automatically.';
			brief = '';
			await onchange?.();
		} catch (e) {
			message =
				e instanceof FounderosApiError
					? (((e.body as { error?: string } | undefined)?.error ?? e.message) as string)
					: 'Connection interrupted. Refresh and check the request list before retrying.';
			if (e instanceof FounderosApiError) await onchange?.();
		} finally {
			busy = false;
		}
	}

	function launch(id: string) {
		if (window.confirm('Start a Codex agent in a new client workspace? This can use your agent credits. It will draft only, not publish.')) {
			void submit({ action: 'launch', id });
		}
	}

	// the slab filter chips (slab.tsx chipClass)
	const chip = chipClass;
	const field = 'bn-field w-full rounded-[12px] border p-3 text-sm';
</script>

{#if selected}
	<div data-part="board" class="mt-6 grid gap-6 lg:grid-cols-[300px_minmax(0,1fr)]">
		<SlabCard i={6} title="Clients" sub="{clients.length} confirmed" class="self-start">
			<div class="space-y-3 px-6 pb-6 pt-4">
				{#each clients as c (c.id)}
					<button
						type="button"
						aria-pressed={c.id === selected.id}
						onclick={() => {
							clientId = c.id;
							filter = 'all';
						}}
						data-lens="c"
						class="bn-pressable bn-client w-full rounded-[12px] border px-4 py-3.5 text-left"
						data-on={c.id === selected.id}
					>
						<span class="block text-[13.5px] font-medium">{c.name}</span>
						<span class="bn-dim mt-1 block font-mono text-[11px]">{c.service} · Client confirmed by the operator</span>
					</button>
				{/each}
				<p class="bn-dim text-[12px] leading-relaxed">
					Only confirmed client projects appear here. Paying cohort members and unrelated Superset projects are not automatically treated as retainers.
				</p>
				<div class="bn-border rounded-[12px] border p-4 text-xs leading-relaxed">
					<div class="flex items-center justify-between gap-2">
						<h2 class="font-semibold">Slack bridge</h2>
						<span class="rounded-full px-2.5 py-0.5 font-mono text-[10.5px] uppercase tracking-[0.12em]" style={statusStyle(statusOrder, 'launching')}>not activated</span>
					</div>
					<p class="bn-dim mt-2">Not activated. An operator channel, authorized sender, and execution host must be configured before Slack can launch work.</p>
				</div>
			</div>
		</SlabCard>

		<div class="min-w-0 space-y-6">
			<SlabCard i={7} title={selected.name} sub={selected.service}>
				<div class="px-6 pb-6 pt-3">
					<p class="bn-muted text-[13.5px] leading-relaxed">{selected.context}</p>
					<dl class="bn-border mt-4 grid gap-4 border-t pt-4 text-xs sm:grid-cols-2">
						<div><dt class="bn-dim font-mono text-[11px]">Superset project</dt><dd class="mt-1 text-[13px]">{selected.projectName}</dd></div>
						<div><dt class="bn-dim font-mono text-[11px]">Retainer</dt><dd class="mt-1 text-[13px]">{selected.retainer}</dd></div>
					</dl>
					<p class="bn-dim mt-4 font-mono text-[11px]">Context sources: {selected.sources.join(' · ')}</p>
				</div>
			</SlabCard>

			<SlabCard i={8} title="New request" sub="saved as a draft">
				<form
					id="new-request"
					class="px-6 pb-6 pt-3"
					onsubmit={(e) => {
						e.preventDefault();
						void submit({ action: 'save', clientId: selected.id, brief });
					}}
				>
					<label for="client-brief" class="block text-sm font-semibold">What should we make?</label>
					<p class="bn-dim mb-3 mt-1 text-xs">
						Try: Draft a five-slide game-day carousel using the restaurant’s verified brand facts. Flag anything that needs Silvio’s confirmation.
					</p>
					<textarea
						id="client-brief"
						class={field}
						rows={5}
						minlength={5}
						maxlength={6000}
						required
						bind:value={brief}
						placeholder="Audience, deliverable, objective, offer, and deadline..."
					></textarea>
					<button type="submit" disabled={busy} data-lens="c" class="bn-pressable bn-accent-pill mt-3 rounded-full border px-4 py-2 text-[13px] disabled:opacity-50">Save draft request</button>
					<p class="bn-dim mt-3 text-xs">Saving does not start an agent, spend credits, send messages, or publish content.</p>
				</form>
			</SlabCard>

			<SlabCard i={9} title="Work requests" sub="{shown.length} of {mine.length}">
				{#snippet action()}
					<button type="button" data-lens="c" class={chip(filter === 'all')} onclick={() => (filter = 'all')}>All {mine.length}</button>
					{#each statusOrder as s (s.key)}
						<button type="button" data-lens="c" class={chip(filter === s.key)} onclick={() => (filter = s.key)}>{s.meter} {count(s.key)}</button>
					{/each}
				{/snippet}
				<div class="px-6 pt-3">
					<label class="bn-dim block text-xs"
						>Operator token for agent launches (kept in memory only)<input type="password" autocomplete="off" class="{field} mt-2" bind:value={token} /></label
					>
					<p class="bn-dim mt-2 text-xs">
						Launches use the OS server’s local Superset installation. The registered client project must exist there. An offline or different host cannot execute it.
					</p>
					{#if !launchEnabled}
						<p data-part="launch-off" class="mt-2 font-mono text-[11px]" style="color: var(--bn-warn)">
							Launching is off on the bridge (FOUNDEROS_WRITES=0): a launch is refused and the request stays saved.
						</p>
					{/if}
				</div>
				<div class="bn-border mt-4 border-t">
					{#if mine.length === 0}
						<p class="bn-dim px-6 py-8 text-center text-[12.5px]">No requests yet.</p>
					{:else if shown.length === 0}
						<p class="bn-dim px-6 py-8 text-center text-[12.5px]">Nothing matches that filter.</p>
					{/if}
					{#each shown as w (w.id)}
						<article data-part="request" class="bn-border border-b px-6 py-4 last:border-0">
							<div class="flex flex-wrap items-center justify-between gap-2">
								<span class="rounded-full px-2.5 py-0.5 font-mono text-[10.5px] uppercase tracking-[0.12em]" style={statusStyle(statusOrder, w.status)}
									>{w.status.replace('_', ' ')}</span
								>
								<time class="bn-dim font-mono text-[11px] tabular-nums">{w.createdAt.slice(0, 16).replace('T', ' ')} UTC</time>
							</div>
							<p class="mt-2 whitespace-pre-wrap text-[13.5px]">{w.brief}</p>
							{#if w.detail}<p class="bn-dim mt-2 text-xs">{w.detail}</p>{/if}
							{#if w.workspaceId}<p class="mt-2 break-all font-mono text-xs">Superset workspace: {w.workspaceId}</p>{/if}
							{#if w.status === 'launching'}<p class="bn-dim mt-2 text-xs">Launch pending or interrupted. Check Superset before creating another request.</p>{/if}
							{#if w.status === 'saved'}
								<button
									type="button"
									disabled={busy || !token}
									data-lens="c"
									class="bn-pressable bn-chip-off mt-3 rounded-full border px-4 py-2 text-[12.5px] disabled:opacity-50"
									onclick={() => launch(w.id)}>Start draft in Superset</button
								>
							{/if}
						</article>
					{/each}
				</div>
			</SlabCard>
			{#if message}<p role="status" class="bn-border rounded-[12px] border px-4 py-3 text-sm">{message}</p>{/if}
		</div>
	</div>
{/if}

<style>
	.bn-client {
		border-color: var(--bn-border);
	}
	.bn-client[data-on='true'] {
		border-color: var(--bn-accent-line);
		background: var(--bn-accent-soft);
	}
	.bn-field {
		border-color: var(--bn-border);
		background: var(--bn-bg);
		color: var(--bn-text);
	}
	.bn-accent-pill {
		border-color: var(--bn-accent-line);
		background: var(--bn-accent-soft);
		color: var(--bn-accent);
	}
	.bn-chip-off {
		border-color: var(--bn-border);
		color: var(--bn-text-2);
	}
	.bn-chip-off:hover {
		color: var(--bn-text);
	}
</style>
