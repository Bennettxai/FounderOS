<!-- Trakyo content & tracking links (FounderOS v1 components/TrakyoLinkStudio.tsx).
     Opening it reads GET /pages/trakyo/workspace. The two forms are prod's,
     label for label; their creates post to /pages/trakyo/content and
     /pages/trakyo/links, outbound Trakyo writes the bridge guard refuses while
     FOUNDEROS_WRITES=0. A refusal reads as held, says nothing left the bridge,
     and pauses further writes exactly as prod pauses on an uncertain result. -->
<script lang="ts">
	import { FounderosApiError, founderosFetch, isGuardRefusal } from '$lib/founderos/api';
	import type { TrakyoContent } from './types';

	type Workspace = {
		state: string;
		scopes: string[];
		messages: string[];
		domains?: { id: string; hostname: string }[];
		links: { domain_id: string; key: string; short_link: string; url: string; clicks: number }[];
		hasMore?: boolean;
	};
	type Created = { id: string; name: string; source: string };

	let { content = [], onCreated = () => {} }: { content?: TrakyoContent[]; onCreated?: () => void } = $props();

	const uid = $props.id();
	let workspace = $state<Workspace | null>(null);
	let loading = $state(false);
	let busy = $state(false);
	let error = $state('');
	let uncertain = $state(false);
	let held = $state(false);
	let source = $state('Instagram');
	let name = $state('');
	let description = $state('');
	let selected = $state('');
	let created = $state<Created | null>(null);
	let url = $state('');
	let slug = $state('');
	let domain = $state('');
	let result = $state('');
	let copyStatus = $state('');

	const control = 'fn-input w-full rounded-lg border px-3 py-2 text-sm disabled:opacity-50';
	const pressableButton = 'bn-pressable bn-pill rounded-full border px-4 py-2 text-xs disabled:opacity-40';
	const canContent = $derived(workspace?.scopes.includes('content:write') ?? false);
	const canLinks = $derived(workspace?.scopes.includes('links:write') ?? false);
	const choices = $derived([
		...new Map(
			[...content.filter((c) => c.id?.startsWith('ci_')).map((c) => ({ id: c.id!, name: c.name, source: c.source })), ...(created ? [created] : [])].map((c) => [c.id, c])
		).values()
	]);

	async function load() {
		loading = true;
		error = '';
		try {
			workspace = await founderosFetch<Workspace>('/pages/trakyo/workspace');
		} catch (e) {
			error =
				e instanceof FounderosApiError && e.status === 404
					? 'Trakyo link studio is not served on the bridge yet (/pages/trakyo/workspace).'
					: 'Could not read Trakyo link permissions. Try again.';
		} finally {
			loading = false;
		}
	}

	async function post<T>(path: string, json: unknown): Promise<T> {
		try {
			return await founderosFetch<T>(path, { method: 'POST', json });
		} catch (e) {
			if (isGuardRefusal(e)) {
				held = true;
				throw new Error('Held: writes are off (FOUNDEROS_WRITES=0), so nothing was sent to Trakyo.');
			}
			const body = (e instanceof FounderosApiError ? e.body : null) as { error?: string; uncertain?: boolean } | null;
			if (!(e instanceof FounderosApiError) || body?.uncertain) {
				uncertain = true;
				throw new Error('Result unconfirmed. Check Trakyo before retrying.');
			}
			throw new Error(body?.error ?? 'Unable to save in Trakyo.');
		}
	}

	async function saveContent(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		error = '';
		try {
			const item = await post<Created>('/pages/trakyo/content', { source, name, ...(description ? { description } : {}) });
			created = item;
			selected = item.id;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Unable to save content.';
		} finally {
			busy = false;
		}
	}

	async function saveLink(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		error = '';
		result = '';
		try {
			const link = await post<{ short_link: string }>('/pages/trakyo/links', {
				url,
				content_item_id: selected,
				...(slug.trim() ? { slug: slug.trim() } : {}),
				...(domain ? { domain } : {}),
				append_tracking_param: true
			});
			result = link.short_link;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Unable to create a link.';
		} finally {
			busy = false;
		}
	}

	async function copy() {
		try {
			await navigator.clipboard.writeText(result);
			copyStatus = 'Copied';
		} catch {
			copyStatus = 'Select the link and copy it manually.';
		}
	}
</script>

<details class="mt-7 border-t pt-5" style="border-color: var(--bn-border)" ontoggle={(e) => { if (e.currentTarget.open && !workspace && !loading) void load(); }}>
	<summary class="bn-text cursor-pointer text-sm font-semibold">Create content & tracking links <span class="bn-dim ml-2 text-xs font-normal">Trakyo</span></summary>
	<p class="bn-dim mt-3 text-xs leading-relaxed">Save a content record, then create the link to use in your bio, caption or ManyChat reply. This registers attribution in Trakyo; it does not publish a social post.</p>
	{#if loading}<p role="status" class="mt-3 text-xs">Checking Trakyo permissions...</p>{/if}
	{#if error}<p role="alert" class="mt-3 text-xs" style="color: var(--bn-warn)">{error}</p>{/if}
	{#if !workspace && !loading}<button data-lens="c" class="{pressableButton} mt-3" onclick={load}>Retry connection</button>{/if}
	{#if workspace?.state === 'not_configured'}<p class="mt-3 text-xs">Connect Trakyo in Integrations to create tracking links.</p>{/if}
	{#each workspace?.messages ?? [] as m, k (k)}<p class="mt-2 text-xs" style="color: var(--bn-warn)">{m}</p>{/each}
	{#if workspace && workspace.state !== 'not_configured'}
		<div class="mt-5 grid gap-8 lg:grid-cols-2">
			<form onsubmit={saveContent} class="space-y-3">
				<h3 class="bn-text text-sm font-medium">1. Register the content</h3>
				<label class="bn-dim block text-xs" for="{uid}-source">Channel / source<input id="{uid}-source" required maxlength={200} class="{control} mt-1" bind:value={source} disabled={busy || !!created} /></label>
				<label class="bn-dim block text-xs" for="{uid}-name">Content name<input id="{uid}-name" required maxlength={500} placeholder="IG Reel: three ways to automate follow-up" class="{control} mt-1" bind:value={name} disabled={busy || !!created} /></label>
				<label class="bn-dim block text-xs" for="{uid}-description">Caption / notes<textarea id="{uid}-description" maxlength={5000} rows={3} class="{control} mt-1" bind:value={description} disabled={busy || !!created}></textarea></label>
				{#if created}
					<p role="status" class="text-xs">Saved: {created.name}. Continue with its tracking link.</p>
				{:else}
					<button data-lens="c" type="submit" disabled={busy || uncertain || held || !canContent} class={pressableButton}>{busy ? 'Saving...' : 'Create content in Trakyo'}</button>
				{/if}
				{#if !canContent}<p class="bn-dim text-xs">Creating content needs the content:write scope. You can still select existing content.</p>{/if}
			</form>
			<form onsubmit={saveLink} class="space-y-3">
				<h3 class="bn-text text-sm font-medium">2. Create its tracking link</h3>
				<label class="bn-dim block text-xs" for="{uid}-content">Content item<select id="{uid}-content" required class="{control} mt-1" bind:value={selected} disabled={busy} onchange={() => (result = '')}><option value="">Choose existing or newly created content</option>{#each choices as c (c.id)}<option value={c.id}>{c.name} / {c.source}</option>{/each}</select></label>
				<label class="bn-dim block text-xs" for="{uid}-url">Destination URL<input id="{uid}-url" type="url" required placeholder="https://your-landing-page.com" class="{control} mt-1" bind:value={url} disabled={busy} oninput={() => (result = '')} /></label>
				<div class="grid grid-cols-2 gap-3">
					<label class="bn-dim block text-xs" for="{uid}-domain">Domain<select id="{uid}-domain" class="{control} mt-1" bind:value={domain} disabled={busy} onchange={() => (result = '')}><option value="">trakyo.link (default)</option>{#each workspace.domains ?? [] as d (d.id)}<option value={d.id}>{d.hostname}</option>{/each}</select></label>
					<label class="bn-dim block text-xs" for="{uid}-slug">Custom slug (optional)<input id="{uid}-slug" pattern="[A-Za-z0-9_-]+" maxlength={200} placeholder="ig-follow-up" class="{control} mt-1" bind:value={slug} disabled={busy} oninput={() => (result = '')} /></label>
				</div>
				<p class="bn-dim text-[11px]">Tracking parameters stay enabled to connect the click to later visits. The destination page also needs the Trakyo tag.</p>
				<button data-lens="c" type="submit" disabled={busy || uncertain || held || !!result || !selected || !canLinks} class={pressableButton}>{busy ? 'Saving...' : 'Create tracking link'}</button>
				{#if !canLinks}<p class="bn-dim text-xs">Enable links:write on the connected Trakyo key to create links.</p>{/if}
			</form>
		</div>
		{#if result}
			<div role="status" class="mt-5 flex flex-wrap items-center gap-3 border-y py-4" style="border-color: var(--bn-border)">
				<code class="break-all text-sm">{result}</code>
				<button data-lens="c" class={pressableButton} onclick={copy}>Copy link</button>
				<span class="text-xs">{copyStatus}</span>
				<button data-lens="c" class={pressableButton} onclick={onCreated}>Refresh analytics</button>
			</div>
		{/if}
		{#if uncertain}<p class="mt-3 text-xs" style="color: var(--bn-warn)">Writes are paused to avoid duplicates. Check the content and links in Trakyo, then refresh this page.</p>{/if}
		{#if workspace.links.length > 0}
			<div class="mt-6 overflow-x-auto">
				<h3 class="mb-2 text-xs font-medium">Recent tracking links {#if workspace.hasMore}<span class="bn-dim">/ first 100 returned</span>{/if}</h3>
				<table class="w-full text-left text-xs">
					<thead><tr class="bn-dim border-b" style="border-color: var(--bn-border)"><th class="py-2 font-normal">Link</th><th class="py-2 font-normal">Destination</th><th class="py-2 text-right font-normal">Lifetime clicks</th></tr></thead>
					<tbody>
						{#each workspace.links.slice(0, 8) as link (`${link.domain_id}/${link.key}`)}
							<tr class="border-b" style="border-color: var(--bn-border)"><td class="whitespace-nowrap py-2 pr-4"><code>{link.short_link}</code></td><td class="max-w-[280px] truncate py-2 pr-4" title={link.url}>{link.url}</td><td class="py-2 text-right tabular-nums">{link.clicks.toLocaleString()}</td></tr>
						{/each}
					</tbody>
				</table>
				<p class="bn-dim mt-2 text-[10px]">Showing {Math.min(workspace.links.length, 8)} of {workspace.links.length} loaded links. Link clicks are lifetime totals.</p>
			</div>
		{/if}
	{/if}
</details>
