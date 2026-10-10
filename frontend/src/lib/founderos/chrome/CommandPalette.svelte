<!-- ⌘K: one box for jump / run / ask (FounderOS v1 CommandPalette). Groups: Go to,
     Run (every agent, POST /agents/:id/run), Ask (Conductor prompts to the board
     thread); results land on the OS-wide toast stack. Tab cycles scope, ↑↓ move, ↵ fires,
     Esc closes; no match + ↵ asks the Conductor the typed text. Digits 1–9 jump
     views while it is closed. -->
<script lang="ts">
	import { onMount, tick } from 'svelte';
	import '../kit/kit.css';
	import Kbd from '../kit/Kbd.svelte';
	import ToggleChip from '../kit/ToggleChip.svelte';
	import { founderosFetch } from '../api';
	import {
		PALETTE_SCOPES,
		buildPaletteCommands,
		digitTarget,
		filterPalette,
		type PaletteAgent,
		type PaletteCommand,
		type PaletteKind
	} from '../palette';
	import { PALETTE_EVENT, isTyping } from './chrome';
	import { REFUSED_NOTICE, failureMessage } from './conductor';
	import { toast } from './toast';

	let { navigate }: { navigate: (href: string) => void } = $props();

	const SCOPE_LABEL = { all: 'All', go: 'Go to', run: 'Run', ask: 'Ask' } as const;
	const GROUPS: [PaletteKind, string][] = [
		['go', 'Go to'],
		['run', 'Run'],
		['ask', 'Ask']
	];

	let open = $state(false);
	let q = $state('');
	let scope = $state(0);
	let sel = $state(0);
	let input: HTMLInputElement | undefined = $state();

	let agents = $state<PaletteAgent[]>([]);
	let agentsError = $state<string | null>(null);
	let agentsLoaded = false;

	const commands = $derived(buildPaletteCommands(agents));
	const rows = $derived(filterPalette(commands, q, PALETTE_SCOPES[scope]));
	const groups = $derived(GROUPS.map(([k, label]) => ({ k, label, items: rows.filter((r) => r.kind === k) })).filter((g) => g.items.length));

	async function loadAgents() {
		if (agentsLoaded) return;
		agentsLoaded = true;
		try {
			const body = await founderosFetch<{ agents?: { id: string; name: string; departmentId?: string; description?: string }[] | null }>('/agents');
			agents = (body.agents ?? []).map((a) => ({ id: a.id, name: a.name, role: a.departmentId?.replace(/^dept-/, '') || a.description || 'agent' }));
			agentsError = null;
		} catch (err) {
			agentsLoaded = false; // try again on the next open
			agentsError = `Agents unavailable: ${failureMessage(err)}`;
		}
	}

	async function show() {
		q = '';
		sel = 0;
		scope = 0;
		open = true;
		void loadAgents();
		await tick();
		input?.focus();
	}

	// A FOUNDEROS_WRITES refusal is a held state, not a failure.
	function failed(id: number, err: unknown, prefix: string) {
		const why = failureMessage(err);
		if (why === REFUSED_NOTICE) toast.update(id, 'warn', REFUSED_NOTICE);
		else toast.update(id, 'err', `${prefix}${why}`);
	}

	async function ask(message: string, label: string) {
		const id = toast.busy(`Asking the Conductor · "${label}"`);
		try {
			await founderosFetch('/pages/conductor/chat', { method: 'POST', json: { message } });
			toast.update(id, 'ok', `Sent to the Conductor · "${label}"`);
		} catch (err) {
			failed(id, err, 'Conductor unreachable · ');
		}
	}

	// One run per agent at a time: a second Enter while it runs is refused.
	const running = new Set<string>();
	async function run(c: PaletteCommand) {
		const agent = c.agentId!;
		if (running.has(agent)) {
			toast.warn(`${c.title} is already running`);
			return;
		}
		running.add(agent);
		const id = toast.busy(`${c.title}…`);
		try {
			const r = await founderosFetch<{ ok?: boolean; summary?: string }>(`/agents/${encodeURIComponent(agent)}/run`, { method: 'POST' });
			if (r?.ok === false) toast.update(id, 'err', `${c.title} failed${r.summary ? `: ${r.summary}` : ''}`);
			else toast.update(id, 'ok', `${c.title} · done${r?.summary ? ` · ${r.summary}` : ''}`);
		} catch (err) {
			failed(id, err, `${c.title} failed: `);
		} finally {
			running.delete(agent);
		}
	}

	async function fire(c?: PaletteCommand) {
		open = false;
		const text = q.trim();
		if (!c) {
			if (text) await ask(text, text.length > 40 ? `${text.slice(0, 40)}…` : text);
			return;
		}
		if (c.kind === 'run' && c.agentId) return run(c);
		if (c.kind === 'ask' && c.prompt) return ask(c.prompt, c.title);
		if (c.href) navigate(c.href);
	}

	onMount(() => {
		const onKeydown = (e: KeyboardEvent) => {
			if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
				e.preventDefault();
				if (open) open = false;
				else show();
			} else if (e.key === 'Escape') {
				open = false;
			} else if (!open && !e.metaKey && !e.ctrlKey && !e.altKey && !isTyping()) {
				const href = digitTarget(e.key);
				if (href) navigate(href);
			}
		};
		const onOpen = () => show();
		window.addEventListener('keydown', onKeydown);
		window.addEventListener(PALETTE_EVENT, onOpen);
		return () => {
			window.removeEventListener('keydown', onKeydown);
			window.removeEventListener(PALETTE_EVENT, onOpen);
		};
	});

	function setScope(i: number) {
		scope = i;
		sel = 0;
		input?.focus();
	}

	function onKey(e: KeyboardEvent) {
		if (e.key === 'ArrowDown') {
			e.preventDefault();
			sel = Math.min(rows.length - 1, sel + 1);
		} else if (e.key === 'ArrowUp') {
			e.preventDefault();
			sel = Math.max(0, sel - 1);
		} else if (e.key === 'Tab') {
			e.preventDefault();
			scope = (scope + 1) % PALETTE_SCOPES.length;
			sel = 0;
		} else if (e.key === 'Enter') {
			e.preventDefault();
			void fire(rows[sel]);
		}
	}
</script>

{#if open}
	<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
	<div class="bn-scrim bn-enter fixed inset-0 z-50" onclick={() => (open = false)}>
		<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
		<div
			role="dialog"
			aria-modal="true"
			aria-label="Command palette"
			tabindex="-1"
			onclick={(e) => e.stopPropagation()}
			class="bn-palette bn-palette-in absolute left-1/2 top-14 w-[560px] max-w-[calc(100vw-32px)] -translate-x-1/2 overflow-hidden rounded-[12px] border"
		>
			<div class="bn-rule flex items-center gap-2.5 border-b px-3.5 py-3">
				<span class="bn-dim">›</span>
				<input
					bind:this={input}
					value={q}
					oninput={(e) => {
						q = e.currentTarget.value;
						sel = 0;
					}}
					onkeydown={onKey}
					role="combobox"
					aria-expanded="true"
					aria-controls="bn-palette-list"
					aria-autocomplete="list"
					placeholder="Jump, run, ask… (type an agent name, a route, or a question)"
					class="bn-text min-w-0 flex-1 bg-transparent font-mono text-[12.5px] outline-none"
				/>
				<Kbd size="sm">esc</Kbd>
			</div>
			<div class="bn-hair flex gap-1.5 border-b px-3.5 py-2" role="tablist" aria-label="Scope">
				{#each PALETTE_SCOPES as s, i (s)}
					<ToggleChip on={scope === i} role="tab" aria-selected={scope === i} onclick={() => setScope(i)}>{SCOPE_LABEL[s]}</ToggleChip>
				{/each}
				<span class="bn-dim ml-auto self-center font-mono text-[9px]">{rows.length} results</span>
			</div>
			<div id="bn-palette-list" role="listbox" class="max-h-[330px] overflow-auto p-1.5">
				{#each groups as g (g.k)}
					<div data-part="palette-group" class="bn-dim px-2.5 pb-1 pt-2 font-mono text-[8.5px] uppercase tracking-[.2em]">{g.label}</div>
					{#each g.items as r (r.id)}
						{@const i = rows.indexOf(r)}
						<div
							role="option"
							tabindex="-1"
							aria-selected={i === sel}
							onmouseenter={() => (sel = i)}
							onclick={() => void fire(r)}
							onkeydown={() => {}}
							data-lens="r"
							class="bn-pressable is-row bn-enter bn-row grid cursor-pointer grid-cols-[22px_minmax(0,1fr)_auto] items-center gap-2.5 rounded-[6px] border px-2.5 py-[7px]"
						>
							<span class="bn-glyph grid h-[22px] w-[22px] place-items-center rounded-[5px] border font-mono text-[10px]" data-kind={r.kind}>{r.glyph}</span>
							<div class="min-w-0">
								<div class="bn-text truncate text-[11.5px] font-semibold">{r.title}</div>
								<div class="bn-dim truncate font-mono text-[9.5px]">{r.sub}</div>
							</div>
							<span class="bn-dim flex items-center gap-1 font-mono text-[9.5px]">{r.kind === 'go' ? 'jump' : r.kind}{#if i === sel}<Kbd size="sm">↵</Kbd>{/if}</span>
						</div>
					{/each}
				{/each}
				{#if agentsError && (PALETTE_SCOPES[scope] === 'run' || PALETTE_SCOPES[scope] === 'all')}
					<div class="bn-warn px-2.5 py-2 font-mono text-[9.5px]">{agentsError}</div>
				{/if}
				{#if rows.length === 0}
					<div class="bn-dim p-7 text-center font-mono text-[10.5px] leading-relaxed">
						<span class="bn-text">No match for «{q}».</span><br />
						<span class="text-[9.5px]">press ↵ to ask the Conductor instead</span>
					</div>
				{/if}
			</div>
			<div class="bn-rule bn-dim flex items-center gap-3.5 border-t px-3.5 py-2 font-mono text-[9.5px]">
				<span><Kbd size="sm">↑</Kbd> <Kbd size="sm">↓</Kbd> move</span>
				<span><Kbd size="sm">↵</Kbd> run</span>
				<span><Kbd size="sm">tab</Kbd> scope</span>
				<span data-part="listening" class="ml-auto inline-flex items-center gap-1.5">
					<span class="bn-blink bn-listening h-1.5 w-1.5" aria-hidden="true"></span>
					Conductor listening
				</span>
			</div>
		</div>
	</div>
{/if}

<style>
	.bn-scrim {
		background: color-mix(in oklab, var(--bn-bg) 55%, transparent);
	}
	.bn-palette {
		border-color: var(--bn-border-strong);
		background: var(--bn-bg);
		box-shadow: var(--bn-shadow-pop);
	}
	.bn-listening {
		background: var(--bn-ok);
	}
	.bn-rule {
		border-color: var(--bn-border);
	}
	.bn-hair {
		border-color: var(--bn-hairline);
	}
	.bn-row {
		border-color: transparent;
	}
	.bn-row:where([aria-selected='true']) {
		border-color: var(--bn-border-strong);
		background: var(--bn-surface-2);
	}
	.bn-glyph {
		border-color: var(--bn-border);
		color: var(--bn-text-2);
	}
	.bn-glyph[data-kind='run'] {
		color: var(--bn-ok);
	}
	.bn-glyph[data-kind='ask'] {
		color: var(--bn-warn);
	}
	.bn-warn {
		color: var(--bn-warn);
	}
	input::placeholder {
		color: var(--bn-text-3);
	}
</style>
