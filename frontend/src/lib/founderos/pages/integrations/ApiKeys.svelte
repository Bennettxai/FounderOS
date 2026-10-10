<!-- API keys (FounderOS v1 ApiKeys). Slot state comes from GET /pages/connections:
     set / not set, plus the last-4 mask when the server sends one.
     A set slot reads as a blank mask; "reveal" swaps in that tail (or just
     "set" when there is none) and never more. "set"/"rotate" saves through
     /pages/connections/connect into ~/.founderos/.env. "test" runs the
     slot's real connector check via /pages/admin/keys/test and reports the
     round trip; a red result stays red. -->
<script lang="ts">
	import { KeyRound } from '$lib/founderos/icons';
	import { founderosFetch } from '$lib/founderos/api';
	import { Pressable } from '$lib/founderos/kit';
	import type { KeySlot, KeyTestResult } from './types';

	let { keys, onchange }: { keys: KeySlot[]; onchange?: () => void } = $props();

	let editing = $state<string | null>(null);
	let value = $state('');
	let savedVar = $state<string | null>(null);
	let error = $state<string | null>(null);
	let tests = $state<Record<string, KeyTestResult | 'busy'>>({});
	let revealed = $state<Record<string, boolean>>({});
	/** Production's AsyncButton: the verdict pops for 1.4s, then the button reads "test" again. */
	let shown = $state<Record<string, boolean>>({});

	const groups = $derived.by(() => {
		const map = new Map<string, KeySlot[]>();
		for (const k of keys) {
			if (!map.has(k.group)) map.set(k.group, []);
			map.get(k.group)!.push(k);
		}
		return [...map.entries()];
	});

	const message = (e: unknown, fallback: string) => (e instanceof Error ? e.message : fallback);

	async function save(envVar: string) {
		if (!value.trim()) return;
		error = null;
		try {
			await founderosFetch('/pages/connections/connect', { method: 'POST', json: { slot: envVar, value: value.trim() } });
			value = '';
			editing = null;
			savedVar = envVar;
			setTimeout(() => (savedVar = null), 2500);
			onchange?.();
		} catch (e) {
			error = message(e, 'save failed');
		}
	}

	async function test(envVar: string) {
		error = null;
		tests = { ...tests, [envVar]: 'busy' };
		const started = Date.now();
		let result: KeyTestResult;
		try {
			const b = await founderosFetch<{ ok: boolean; ms?: number; detail?: string; state?: string }>('/pages/admin/keys/test', {
				method: 'POST',
				json: { envVar }
			});
			result = { ok: Boolean(b?.ok), ms: typeof b?.ms === 'number' ? b.ms : Date.now() - started, detail: b?.detail || b?.state };
		} catch (e) {
			result = { ok: false, ms: Date.now() - started, detail: message(e, 'test failed') };
		}
		tests = { ...tests, [envVar]: result };
		shown = { ...shown, [envVar]: true };
		setTimeout(() => (shown = { ...shown, [envVar]: false }), 1400);
		if (!result.ok) error = `${envVar}: ${result.detail ?? 'test failed'}`;
	}

	function testLabel(envVar: string): string {
		const t = tests[envVar];
		if (t === 'busy') return 'testing';
		if (!t || !shown[envVar]) return 'test';
		return t.ok ? `✓ ${t.ms}ms` : '✗ failed';
	}
</script>

<section class="mt-10">
	<div class="mb-1 flex items-center gap-2">
		<KeyRound class="bn-muted h-4 w-4" />
		<h2 class="bn-muted text-sm font-bold uppercase tracking-widest">API keys</h2>
	</div>
	<p class="bn-dim mb-4 text-xs">
		Stored in <code>~/.founderos/.env</code>, applied live. Values shown masked — the OS never echoes a secret back.
	</p>
	{#if error}<p class="bn-muted mb-3 font-mono text-[11px]">✗ {error}</p>{/if}

	{#if keys.length === 0}
		<p class="bn-dim font-mono text-[11px]">no key slots</p>
	{:else}
		<div class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
			{#each groups as [group, slots], gi (group + gi)}
				<div data-group={group} data-lens="r" class="bn-pressable is-row bn-keygroup rounded-[var(--bn-r-panel)] p-4">
					<h3 class="bn-text mb-2 text-xs font-bold">{group}</h3>
					<ul class="space-y-1.5">
						{#each slots as slot (slot.envVar)}
							{@const t = tests[slot.envVar]}
							<li class="text-[11px]" data-env={slot.envVar}>
								<div class="flex items-center gap-2">
									<span class="bn-slot-dot h-1.5 w-1.5 shrink-0 rounded-full" data-present={slot.present ? '' : undefined}></span>
									<span class="bn-muted w-40 truncate font-mono" title={slot.label}>{slot.envVar}</span>
									<span class="bn-dim font-mono"
										>{slot.present ? (revealed[slot.envVar] ? slot.masked || 'set' : '••••••••') : 'not set'}</span
									>
									{#if savedVar === slot.envVar}<span class="bn-text">✓</span>{/if}
									<span class="ml-auto flex items-center gap-2">
										{#if slot.present}
											<button
												type="button"
												data-lens="c"
												class="bn-pressable bn-dim text-[10px]"
												onclick={() => (revealed = { ...revealed, [slot.envVar]: !revealed[slot.envVar] })}
												>{revealed[slot.envVar] ? 'hide' : 'reveal'}</button
											>
										{/if}
										{#if slot.present && slot.connectorId}
											<Pressable
												tone="secondary"
												class="bn-test"
												data-result={t && t !== 'busy' && shown[slot.envVar] ? (t.ok ? 'ok' : 'err') : undefined}
												aria-busy={t === 'busy'}
												disabled={t === 'busy'}
												onclick={() => void test(slot.envVar)}>{testLabel(slot.envVar)}</Pressable
											>
										{/if}
										<button
											type="button"
											data-lens="c"
											class="bn-pressable bn-dim text-[10px]"
											onclick={() => {
												editing = editing === slot.envVar ? null : slot.envVar;
												value = '';
												error = null;
											}}>{editing === slot.envVar ? 'cancel' : slot.present ? 'rotate' : 'set'}</button
										>
									</span>
								</div>
								{#if editing === slot.envVar}
									<div class="mt-1.5 flex gap-1.5 pl-3.5">
										<input
											type="password"
											autocomplete="off"
											bind:value
											onkeydown={(e) => e.key === 'Enter' && void save(slot.envVar)}
											placeholder={slot.hint ?? 'paste value'}
											class="bn-keyfield flex-1 rounded-[var(--bn-r-ctl)] px-2 py-1 font-mono text-[11px]"
										/>
										<button type="button" data-lens="c" class="bn-pressable is-primary bn-keysave rounded-[var(--bn-r-ctl)] px-2.5 py-1 text-[10px] font-bold" onclick={() => void save(slot.envVar)}
											>Save</button
										>
									</div>
								{/if}
							</li>
						{/each}
					</ul>
				</div>
			{/each}
		</div>
	{/if}
</section>

<style>
	.bn-keygroup {
		border: 1px solid var(--bn-border);
		background: var(--bn-surface);
	}
	.bn-slot-dot {
		border: 1px solid var(--bn-text-3);
	}
	.bn-slot-dot[data-present] {
		background: var(--bn-text);
		border-color: var(--bn-text);
	}
	:global(.bn-pressable.bn-test[data-result='ok']) {
		color: var(--bn-ok);
	}
	:global(.bn-pressable.bn-test[data-result='err']) {
		color: var(--bn-err);
	}
	.bn-keyfield {
		border: 1px solid var(--bn-border);
		background: var(--bn-bg);
		color: var(--bn-text);
	}
	.bn-keyfield:focus {
		outline: none;
		border-color: var(--bn-border-strong);
	}
	.bn-keysave {
		background: var(--bn-text);
		color: var(--bn-bg);
	}
</style>
