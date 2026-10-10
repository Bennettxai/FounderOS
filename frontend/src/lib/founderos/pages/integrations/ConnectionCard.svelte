<!-- One tile of the Connections board (FounderOS v1 components/ConnectionCard
     + ConnectFlow): brand logo beside the name and tagline, then the live
     footer. Connected state always comes from the live connector, never the
     stored key alone; a live tile carries its status in the hairline.
     Connect opens a paste-a-key form (one field per env key) that saves
     through /pages/connections/connect into ~/.founderos/.env, never anywhere
     outward. Providers with an OAuth flow show v1's OAuth strip above the
     status row; guidance-only tools show Setup with their live hint. -->
<script lang="ts">
	import { founderosFetch } from '$lib/founderos/api';
	import { Dot, Pressable } from '$lib/founderos/kit';
	import BrandLogo from './BrandLogo.svelte';
	import type { CatalogEntry, OAuthReadiness } from './types';

	let {
		entry,
		guidance,
		oauth = null,
		onchange
	}: {
		entry: CatalogEntry;
		/** Live connector detail, shown for guidance-only tools. */
		guidance?: string;
		/** Present only for providers with an authorization-code flow. */
		oauth?: OAuthReadiness | null;
		/** Called after a save or disconnect so the board re-reads live state. */
		onchange?: () => void;
	} = $props();

	let open = $state(false);
	let values = $state<Record<string, string>>({});
	let phase = $state<'idle' | 'busy' | 'done' | 'failed'>('idle');
	let error = $state<string | null>(null);

	const message = (e: unknown) => (e instanceof Error ? e.message : 'save failed');

	async function save() {
		error = null;
		if (entry.keys.some((k) => !(values[k] ?? '').trim())) {
			error = 'every field is required';
			return;
		}
		phase = 'busy';
		try {
			await founderosFetch('/pages/connections/connect', { method: 'POST', json: { slug: entry.slug, values: { ...values } } });
			phase = 'done';
			open = false;
			values = {};
			onchange?.();
		} catch (e) {
			phase = 'failed';
			error = message(e);
		}
	}

	async function disconnect() {
		phase = 'busy';
		error = null;
		try {
			await founderosFetch('/pages/connections/connect', { method: 'DELETE', json: { slug: entry.slug } });
			onchange?.();
		} catch (e) {
			error = message(e);
		} finally {
			phase = 'idle';
		}
	}

	const saveLabel = $derived(phase === 'busy' ? 'Saving…' : phase === 'failed' ? 'failed · retry' : 'Save & connect');
</script>

<div
	data-lens="r"
	class="bn-pressable is-row bn-tile group flex min-h-[112px] flex-col justify-between rounded-[var(--bn-r-tile)] p-4"
	data-slug={entry.slug}
	data-connected={entry.connected ? '' : undefined}
>
	<div class="flex items-start gap-3">
		<BrandLogo slug={entry.slug} name={entry.name} />
		<div class="min-w-0 flex-1 pt-0.5">
			<div class="bn-text truncate text-[13.5px] font-semibold leading-tight">{entry.name}</div>
			<div class="bn-dim mt-1 truncate text-[11px] leading-tight">{entry.tagline}</div>
		</div>
	</div>

	{#if open}
		<div class="mt-3">
			{#each entry.keys as k (k)}
				<input
					type="password"
					autocomplete="off"
					placeholder={k}
					value={values[k] ?? ''}
					oninput={(e) => (values = { ...values, [k]: (e.currentTarget as HTMLInputElement).value })}
					class="bn-field mb-1.5 w-full rounded-[var(--bn-r-ctl)] px-2 py-1.5 font-mono text-[10.5px]"
				/>
			{/each}
			{#if error}<div class="bn-err-text mb-1.5 font-mono text-[9.5px]">{error}</div>{/if}
			<div class="flex items-center justify-end gap-2">
				<button
					type="button"
					data-lens="c"
					class="bn-pressable bn-dim rounded-full px-2.5 py-1 font-mono text-[10px] uppercase tracking-[0.1em]"
					onclick={() => {
						open = false;
						error = null;
						phase = 'idle';
					}}>Cancel</button
				>
				<Pressable tone="primary" disabled={phase === 'busy'} aria-busy={phase === 'busy'} onclick={() => void save()}
					>{saveLabel}</Pressable
				>
			</div>
		</div>
	{:else}
		<div class="mt-3">
			{#if oauth}
				<!-- the preferred path when the provider supports it; the key form stays the fallback -->
				<div class="bn-oauth bn-dim mb-2 pb-2 font-mono text-[9.5px] leading-relaxed">
					{#if oauth.redirectKind === 'https-public'}
						OAuth needs a public https redirect — put a domain or tunnel in front of the OS first, then register
						<span class="bn-muted">/api/oauth/callback</span> at
						<a href={oauth.consoleUrl} target="_blank" rel="noreferrer" class="bn-linky bn-muted">{oauth.name} console</a>.
					{:else if oauth.appConfigured}
						OAuth app saved{oauth.connected ? (oauth.expired ? ' · token expired' : ' · token in hand') : ''}. The authorize flow is not wired
						yet, so connect with a key.
					{:else}
						Register an app at
						<a href={oauth.consoleUrl} target="_blank" rel="noreferrer" class="bn-linky bn-muted">{oauth.name} console</a>
						with redirect <span class="bn-muted">/api/oauth/callback</span>, then save
						<span class="bn-muted">{oauth.clientIdEnv}</span> and <span class="bn-muted">{oauth.clientSecretEnv}</span> below.{oauth.redirectKind ===
						'loopback'
							? ' This provider allows http only on localhost, so open the OS on the box itself to finish it.'
							: ''}
					{/if}
				</div>
			{/if}
			<div class="flex items-center justify-between gap-2">
				{#if entry.connected}
					<span class="bn-ok-text flex items-center gap-1.5 whitespace-nowrap font-mono text-[10px] uppercase tracking-[0.12em]"
						><Dot state="connected" /><span>Connected</span></span
					>
				{:else if entry.keySaved}
					<span class="bn-warn-text flex items-center gap-1.5 whitespace-nowrap font-mono text-[10px] uppercase tracking-[0.12em]"
						><Dot state="warn" /><span>Key saved</span></span
					>
				{:else}
					<span class="bn-dim whitespace-nowrap font-mono text-[10px] uppercase tracking-[0.12em]">Not connected</span>
				{/if}

				{#if entry.keySaved}
					<button
						type="button"
						data-lens="c"
						class="bn-pressable bn-dim rounded-full px-2.5 py-1 font-mono text-[10px] uppercase tracking-[0.1em] disabled:opacity-40"
						disabled={phase === 'busy'}
						onclick={() => void disconnect()}>Disconnect</button
					>
				{:else if entry.connected}
					<!-- v1's text-os-dim/60 never resolves its alpha, so Managed renders in the text colour there -->
					<span
						class="bn-text cursor-default rounded-full px-2.5 py-1 font-mono text-[10px] uppercase tracking-[0.1em]"
						title="Credentials managed outside Founder OS (canonical machine files)">Managed</span
					>
				{:else if entry.keys.length > 0}
					<button
						type="button"
						data-lens="c"
						class="bn-pressable bn-connect rounded-full px-3 py-1 font-mono text-[10px] uppercase tracking-[0.1em]"
						onclick={() => {
							open = true;
							phase = 'idle';
						}}>+ Connect</button
					>
				{:else}
					<span
						class="bn-setup bn-dim cursor-help px-3 py-1 font-mono text-[10px] uppercase tracking-[0.1em]"
						title={guidance ?? 'Connects through local setup, not a pasted key'}>Setup</span
					>
				{/if}
			</div>
			{#if error}<div class="bn-err-text mt-1.5 font-mono text-[9.5px]">{error}</div>{/if}
		</div>
	{/if}
</div>

<style>
	.bn-tile {
		border: 1px solid var(--bn-border);
		background: var(--bn-surface);
	}
	/* a live tile carries its status in the hairline: colour means status only */
	.bn-tile[data-connected]:not(:hover) {
		border-color: color-mix(in oklab, var(--bn-ok) 38%, var(--bn-border));
	}
	.bn-field {
		border: 1px solid var(--bn-border);
		background: var(--bn-surface-2);
		color: var(--bn-text);
	}
	.bn-field:focus {
		outline: none;
		border-color: var(--bn-border-strong);
	}
	.bn-oauth {
		border-bottom: 1px solid var(--bn-border);
	}
	.bn-connect {
		border: 1px solid var(--bn-border-strong);
		color: var(--bn-text);
		border-radius: 999px;
	}
	.bn-connect:hover {
		background: var(--bn-text);
		color: var(--bn-bg);
	}
	.bn-setup {
		border: 1px solid var(--bn-border);
		border-radius: 999px;
	}
	.bn-ok-text {
		color: var(--bn-ok);
	}
	.bn-warn-text {
		color: var(--bn-warn);
	}
	.bn-err-text {
		color: var(--bn-err);
	}
</style>
