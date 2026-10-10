<!-- Compact capture in the header's right slot (FounderOS v1 components/BrainDump.tsx,
     compact mode): type, talk (browser speech recognition, no key) or drop / upload
     text documents. Save is one Optimal Engine capture per note through
     POST /pages/brain/dump; nothing is written anywhere else, so a failed capture
     says nothing was saved. -->
<script lang="ts">
	import { Mic, MicOff, Upload } from '$lib/founderos/icons';
	import { onMount } from 'svelte';
	import { founderosFetch } from '$lib/founderos/api';
	import type { DumpResult } from './types';

	let { folder = 'inbox', onSaved }: { folder?: string; onSaved?: () => void } = $props();

	type Recognizer = {
		continuous: boolean;
		interimResults: boolean;
		lang: string;
		start(): void;
		stop(): void;
		onresult: ((e: { results: ArrayLike<{ 0: { transcript: string } }> }) => void) | null;
		onend: (() => void) | null;
		onerror: ((e: { error: string }) => void) | null;
	};

	function recognizer(): Recognizer | null {
		if (typeof window === 'undefined') return null;
		const w = window as unknown as { SpeechRecognition?: new () => Recognizer; webkitSpeechRecognition?: new () => Recognizer };
		const Ctor = w.SpeechRecognition ?? w.webkitSpeechRecognition;
		return Ctor ? new Ctor() : null;
	}

	/** documents we can honestly ingest: anything that reads as text */
	const TEXTY = /\.(md|markdown|txt|csv|json|ya?ml|html?|log)$/i;
	const MAX_DOC_BYTES = 1_000_000;

	let text = $state('');
	let listening = $state(false);
	let supported = $state(false);
	let dragOver = $state(false);
	let status = $state<{ kind: 'idle' | 'saving' | 'saved' | 'error'; detail?: string; embedded?: boolean; slug?: string }>({ kind: 'idle' });
	let rec: Recognizer | null = null;
	let base = '';
	let fileInput: HTMLInputElement | undefined = $state();

	onMount(() => {
		supported = recognizer() !== null;
	});

	function stopListening() {
		rec?.stop();
		rec = null;
		listening = false;
	}

	function startListening() {
		const r = recognizer();
		if (!r) return;
		base = text ? `${text.trim()} ` : '';
		r.continuous = true;
		r.interimResults = true;
		r.lang = 'en-US';
		r.onresult = (e) => {
			let t = '';
			for (let i = 0; i < e.results.length; i++) t += e.results[i][0].transcript;
			text = base + t.trim();
		};
		r.onend = () => (listening = false);
		r.onerror = (e) => {
			listening = false;
			status = { kind: 'error', detail: `mic: ${e.error}` };
		};
		rec = r;
		status = { kind: 'idle' };
		listening = true;
		r.start();
	}

	const message = (e: unknown, fallback: string) => (e instanceof Error ? e.message : fallback);

	async function post(body: { text: string; title?: string }): Promise<DumpResult> {
		return founderosFetch<DumpResult>('/pages/brain/dump', { method: 'POST', json: { ...body, folder, tags: [] } });
	}

	async function save() {
		if (!text.trim() || status.kind === 'saving') return;
		stopListening();
		status = { kind: 'saving' };
		try {
			const r = await post({ text });
			status = { kind: 'saved', detail: r.relPath, embedded: r.embedded, slug: r.slug };
			text = '';
			onSaved?.();
		} catch (e) {
			status = { kind: 'error', detail: `nothing saved: ${message(e, 'save failed')}` };
		}
	}

	/** dropped documents flow through the same capture, one note per file, filename as title */
	async function ingestFiles(files: File[]) {
		const texty = files.filter((f) => TEXTY.test(f.name) || f.type.startsWith('text/'));
		const skipped = files.length - texty.length;
		if (texty.length === 0) {
			status = { kind: 'error', detail: 'only text documents for now (.md .txt .csv .json …)' };
			return;
		}
		status = { kind: 'saving' };
		let saved = 0;
		try {
			for (const f of texty) {
				if (f.size > MAX_DOC_BYTES) throw new Error(`${f.name} is over 1MB`);
				const content = await f.text();
				if (!content.trim()) continue;
				await post({ text: content, title: f.name.replace(/\.[^.]+$/, '') });
				saved += 1;
			}
			status = { kind: 'saved', embedded: true, detail: `${saved} doc${saved === 1 ? '' : 's'}${skipped ? ` · ${skipped} skipped (not text)` : ''}` };
			onSaved?.();
		} catch (e) {
			status = { kind: 'error', detail: `${saved} saved, then: ${message(e, 'ingest failed')}` };
		}
	}

	const footer = $derived(
		status.kind === 'saving'
			? 'saving…'
			: status.kind === 'saved'
				? `✓ ${status.embedded ? 'embedded' : 'saved'} · ${status.slug ?? status.detail}`
				: status.kind === 'error'
					? `✗ ${status.detail}`
					: 'text · voice · drag or upload'
	);
</script>

<div
	data-part="brain-dump"
	role="region"
	aria-label="Brain dump"
	data-lens="r"
	class="bn-dump w-full rounded-[10px] border p-2"
	class:is-over={dragOver}
	ondragover={(e) => {
		e.preventDefault();
		dragOver = true;
	}}
	ondragleave={() => (dragOver = false)}
	ondrop={(e) => {
		e.preventDefault();
		dragOver = false;
		void ingestFiles([...(e.dataTransfer?.files ?? [])]);
	}}
>
	<div class="relative">
		<textarea
			bind:value={text}
			onkeydown={(e) => {
				if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) void save();
			}}
			rows="2"
			aria-label="Dump into the brain"
			placeholder={dragOver ? 'drop it — one note per document' : 'dump into the brain… or drop documents'}
			class="bn-text w-full resize-none border-0 bg-transparent px-1.5 py-1 pr-9 font-mono text-[11.5px] leading-relaxed outline-none"
			style="background: transparent; box-shadow: none"
		></textarea>
		{#if supported}
			<button
				type="button"
				onclick={listening ? stopListening : startListening}
				title={listening ? 'Stop dictation' : 'Start dictation'}
				aria-label={listening ? 'Stop dictation' : 'Start dictation'}
				data-lens="c"
				class="bn-pressable bn-mic absolute right-0 top-0 flex h-6 w-6 items-center justify-center rounded-[6px] border"
				class:is-on={listening}
				>{#if listening}<MicOff size={12} style="width: 12px; height: 12px" />{:else}<Mic size={12} style="width: 12px; height: 12px" />{/if}</button
			>
		{/if}
	</div>
	<input
		bind:this={fileInput}
		type="file"
		multiple
		accept=".md,.markdown,.txt,.csv,.json,.yaml,.yml,.html,.htm,.log,text/*"
		class="hidden"
		data-testid="dump-file"
		onchange={(e) => {
			const input = e.currentTarget as HTMLInputElement;
			void ingestFiles([...(input.files ?? [])]);
			input.value = '';
		}}
	/>
	<div class="bn-dump-foot mt-1.5 flex items-center gap-1.5 border-t px-1.5 pt-1.5">
		<span data-part="dump-status" class="min-w-0 flex-1 truncate font-mono text-[9.5px]" class:bn-dim={status.kind !== 'error'} class:bn-err-text={status.kind === 'error'}>{footer}</span>
		<button type="button" onclick={() => fileInput?.click()} title="Choose documents to upload" data-lens="c" class="bn-pressable bn-ctl bn-muted flex shrink-0 items-center gap-1 rounded-[6px] border px-2 py-0.5 font-mono text-[10px]"
			><Upload size={12} style="width: 12px; height: 12px" />Upload</button
		>
		<button
			type="button"
			onclick={save}
			disabled={!text.trim() || status.kind === 'saving'}
			data-lens="c"
			class="bn-pressable bn-save inline-flex shrink-0 items-center rounded-[6px] border px-2.5 font-mono text-[10px] font-bold"
			style="height: 22px">{status.kind === 'saving' ? 'saving' : 'Save'}</button
		>
	</div>
</div>

<style>
	.bn-dump {
		border-color: var(--bn-border);
		background: var(--bn-surface);
		transition: border-color var(--bn-dur-lens, 0.2s) var(--bn-ease-lens, ease);
	}
	.bn-dump.is-over {
		border-color: var(--bn-accent);
	}
	.bn-dump textarea::placeholder {
		color: var(--bn-text-3);
	}
	.bn-dump-foot,
	.bn-ctl,
	.bn-mic {
		border-color: var(--bn-border);
	}
	.bn-mic {
		background: var(--bn-surface);
		color: var(--bn-text-2);
	}
	.bn-mic.is-on {
		animation: bn-mic-pulse 1.2s ease-in-out infinite;
		border-color: var(--bn-err);
		background: var(--bn-err);
		color: var(--bn-bg);
	}
	@keyframes bn-mic-pulse {
		50% {
			opacity: 0.55;
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.bn-mic.is-on {
			animation: none;
		}
	}
	.bn-save {
		border-color: var(--bn-accent);
		background: var(--bn-accent);
		color: var(--bn-accent-ink);
	}
	.bn-save:disabled {
		opacity: 0.35;
		cursor: not-allowed;
	}
	.bn-err-text {
		color: var(--bn-err);
	}
</style>
