<!-- The ask bar, docked at the bottom of the map. It knows what is selected:
     the scope chip takes the selection's kind colour and the suggested
     questions change with it. Port of FounderOS v1
     components/blueprint/AskBar.tsx. It looks and behaves like production:
     a live input, suggested questions, `/` to focus, and an ask first reads
     "Looking at ..." on a busy bar. Production posts to an analyst route it
     never shipped, so every ask there ends in an error line; here the ask
     ends in one short honest line without a request. -->
<script lang="ts">
	import { Send } from '$lib/founderos/icons';
	import type { HAny, HKind } from './hierarchy';

	let { scope, onclearscope }: { scope: HAny | null; onclearscope: () => void } = $props();

	const SUGGEST: Partial<Record<HKind | '_general' | 'container' | 'group' | 'default', string[]>> = {
		_general: ['What is not wired yet?', 'Which agents actually run, and on what?', 'Where does the OS depend on one thing?'],
		agent: ['What does {n} do, and is it wired?', 'Which connectors and models does {n} depend on?', 'Why is {n} marked the way it is?'],
		daemon: ['What breaks if {n} stops?', 'What does {n} read and write?', 'Who approves what {n} sends?'],
		skill: ['Which agents load {n}?', 'What would improve {n}?'],
		connector: ['Is {n} configured here?', 'Which agents use {n}?', 'What would it take to make {n} live?'],
		model: ['What routes to {n}?', 'Is {n} the only path, or is there a fallback?'],
		store: ['Who reads and writes {n}?', 'What page shows {n}?'],
		department: ['Which agents in {n} actually run?', 'What is missing in {n}?'],
		group: ['Summarise {n} in three lines.', 'What in {n} is not configured?'],
		container: ['Summarise what runs on {n}.', 'What is the biggest risk on {n}?'],
		service: ['What depends on {n}?', 'What happens if {n} goes down?'],
		docker: ['What does {n} run, and on which machine?', 'What breaks if {n} stops?'],
		tool: ['What do I use {n} for?', 'Which machine is {n} on, and what does it touch?'],
		machine: ['Summarise what runs on {n}.', 'What is the biggest risk on {n}?'],
		operator: ['What do I touch directly, and what runs without me?', 'Where am I the single point of failure?'],
		command: ['What runs on {n}?', 'What happens if {n} is missing?'],
		app: ['Summarise the OS in three lines.', 'What in the OS is designed but not wired?'],
		router: ['What does {n} decide?', 'What lands on each side of {n}?'],
		surface: ['What does {n} read?', 'Which agents feed {n}?'],
		person: ['What does {n} own?', 'Which agents work beside {n}?'],
		cloud: ['What depends on {n}?', 'What happens if {n} is unavailable?'],
		default: ['What is {n} and what is it connected to?', 'What is wrong with {n}?']
	};

	let inputEl: HTMLInputElement | undefined = $state();
	let value = $state('');
	let busy = $state(false);
	let answer = $state<{ state: 'thinking' | 'error'; text: string } | null>(null);

	/** No analyst exists behind this bar (in FounderOS v1 either), so say so briefly. */
	const NOT_WIRED = 'Not wired yet: no analyst answers this bar.';
	/** About as long as production's failed round trip to its missing route. */
	const THINK_MS = 350;

	async function ask(question?: string) {
		const q = (question ?? value).trim();
		if (q.length < 3 || busy) return;
		busy = true;
		answer = { state: 'thinking', text: scope ? `Looking at ${scope.name}, its members and its connections…` : 'Looking at the whole map…' };
		await new Promise((r) => setTimeout(r, THINK_MS));
		answer = { state: 'error', text: NOT_WIRED };
		busy = false;
		value = '';
	}

	export function focus() {
		inputEl?.focus();
	}
	export function isFocused() {
		return typeof document !== 'undefined' && document.activeElement === inputEl;
	}

	const kindKey = $derived<keyof typeof SUGGEST>(
		scope
			? scope.type === 'container'
				? scope.kind === 'machine'
					? 'machine'
					: 'container'
				: scope.type === 'group'
					? scope.kind === 'department'
						? 'department'
						: 'group'
					: scope.kind
			: '_general'
	);
	const chips = $derived((SUGGEST[kindKey] ?? SUGGEST.default ?? []).map((q) => q.replace('{n}', scope ? scope.name : '')));
	const style = $derived(scope ? `--sc: var(--bh-k-${scope.kind}, var(--accent))` : undefined);
</script>

<div class="bh-ask {scope ? 'has-scope' : 'has-chips'} {busy ? 'is-busy' : ''}" {style}>
	<div class="bh-ask-orbit" aria-hidden="true"></div>
	<div class="bh-ask-glass"></div>
	<div class="bh-ask-inner">
		<div class="bh-ask-row">
			<button type="button" class="bn-pressable bh-ask-scope" title={scope ? 'Clear selection (esc)' : 'Ask about the whole system'} onclick={() => (scope ? onclearscope() : inputEl?.focus())}>
				<i class="sw"></i>
				<span>{scope ? `Ask about ${scope.name}` : 'Ask Founder OS'}</span>
			</button>
			<input
				bind:this={inputEl}
				bind:value
				type="text"
				placeholder={scope ? `ask about ${scope.name}…` : 'ask anything about the system…'}
				autocomplete="off"
				spellcheck={false}
				onkeydown={(e) => {
					if (e.key === 'Enter') ask();
					if (e.key === 'Escape') inputEl?.blur();
				}}
			/>
			<button type="button" class="bn-pressable bh-ask-send" aria-label="Ask" onclick={() => ask()}><Send /></button>
		</div>
		<div class="bh-ask-chips">
			{#each chips as q (q)}
				<button type="button" class="bn-pressable bh-ask-chip" onclick={() => ask(q)}>{q}</button>
			{/each}
		</div>
		{#if answer}
			<div class="bh-ask-answer is-on {answer.state === 'thinking' ? 'is-thinking' : 'is-error'}">
				<div>{answer.text}</div>
			</div>
		{/if}
	</div>
</div>
