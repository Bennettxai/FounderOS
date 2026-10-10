<!-- Publish composer + queue (FounderOS v1 components/PostComposer.tsx). Upload
     and post go through the guarded /pages/social/upload and /pages/social/posts;
     with writes off or Zernio unconfigured the backend refuses them and the
     composer says so (a refused post is recorded as failed, as v1 records it). -->
<script lang="ts">
	import { Clock, Link2, Send, Upload, X } from '$lib/founderos/icons';
	import Badge from '$lib/founderos/kit/Badge.svelte';
	import { founderosFetch, FounderosApiError } from '$lib/founderos/api';
	import { platformLabel } from './lib';
	import type { SocialPost } from './types';

	let { initialPosts, onPosted }: { initialPosts: SocialPost[]; onPosted?: (p: SocialPost) => void } = $props();

	const PLATFORMS = ['instagram', 'tiktok', 'twitter', 'youtube', 'linkedin'];

	// svelte-ignore state_referenced_locally
	let posts = $state<SocialPost[]>(initialPosts);
	let caption = $state('');
	let selected = $state<Set<string>>(new Set(['instagram', 'tiktok']));
	let mediaUrl = $state('');
	let media = $state<{ url: string; name: string }[]>([]);
	let uploading = $state(0);
	let dragOver = $state(false);
	let scheduledFor = $state('');
	let busy = $state(false);
	let done = $state<string | null>(null);
	let error = $state<string | null>(null);
	let fileInput: HTMLInputElement | undefined = $state();

	const queued = $derived(posts.filter((p) => p.status === 'queued'));

	function toggle(id: string) {
		const next = new Set(selected);
		if (next.has(id)) next.delete(id);
		else next.add(id);
		selected = next;
	}

	function reason(err: unknown): string {
		if (err instanceof FounderosApiError) {
			const b = err.body as { error?: unknown } | undefined;
			if (typeof b?.error === 'string') return b.error;
		}
		return err instanceof Error ? err.message : String(err);
	}

	async function uploadFiles(files: FileList | File[]) {
		error = null;
		for (const file of Array.from(files)) {
			uploading += 1;
			try {
				const form = new FormData();
				form.append('file', file);
				const body = await founderosFetch<{ url: string }>('/pages/social/upload', { method: 'POST', body: form });
				media = [...media, { url: body.url, name: file.name }];
			} catch (err) {
				error = `${file.name}: ${reason(err)}`;
			} finally {
				uploading -= 1;
			}
		}
	}

	async function submit() {
		error = null;
		if (!caption.trim()) return (error = 'Add a caption first.');
		if (selected.size === 0) return (error = 'Pick at least one platform.');
		if (uploading > 0) return (error = 'Media still uploading — one sec.');
		busy = true;
		try {
			const body = await founderosFetch<{ post: SocialPost }>('/pages/social/posts', {
				method: 'POST',
				json: {
					caption: caption.trim(),
					platforms: [...selected],
					mediaUrl: media[0]?.url ?? (mediaUrl.trim() || null),
					mediaUrls: [...media.map((m) => m.url), ...(mediaUrl.trim() ? [mediaUrl.trim()] : [])],
					scheduledFor: scheduledFor ? new Date(scheduledFor).toISOString() : null
				}
			});
			posts = [body.post, ...posts];
			onPosted?.(body.post);
			done = body.post.status === 'queued' ? `✓ scheduled · ${selected.size} platform${selected.size === 1 ? '' : 's'}` : '✓ posted';
			setTimeout(() => (done = null), 1800);
			caption = '';
			mediaUrl = '';
			media = [];
			scheduledFor = '';
		} catch (err) {
			// A refused/failed post comes back 502 with the recorded row.
			const b = err instanceof FounderosApiError ? (err.body as { post?: SocialPost } | undefined) : undefined;
			if (b?.post) posts = [b.post, ...posts];
			error = `Not posted: ${reason(err)}`;
		} finally {
			busy = false;
		}
	}

	function fmtWhen(iso: string): string {
		const d = new Date(iso);
		return Number.isNaN(d.getTime()) ? iso : d.toLocaleString('en-US', { month: 'short', day: 'numeric', hour: 'numeric', minute: '2-digit' });
	}
</script>

<div class="grid gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(280px,0.7fr)]" data-part="composer">
	<div class="bn-card p-4">
		<textarea bind:value={caption} rows="4" placeholder="Write a caption — posts for REAL via Zernio…" class="pc-input w-full resize-none px-3 py-2.5 text-[13px] leading-relaxed"></textarea>
		<div class="mt-3 flex flex-wrap gap-1.5">
			{#each PLATFORMS as p (p)}
				<button type="button" class="soc-chip px-2.5 py-1 font-mono text-[10.5px] uppercase tracking-[0.08em]" data-on={selected.has(p)} onclick={() => toggle(p)}>{platformLabel(p)}</button>
			{/each}
		</div>
		<div
			role="button"
			tabindex="0"
			class="soc-chip mt-3 flex cursor-pointer flex-col items-center justify-center gap-1 border-dashed px-3 py-4"
			data-on={dragOver}
			ondragover={(e) => {
				e.preventDefault();
				dragOver = true;
			}}
			ondragleave={() => (dragOver = false)}
			ondrop={(e) => {
				e.preventDefault();
				dragOver = false;
				if (e.dataTransfer?.files.length) void uploadFiles(e.dataTransfer.files);
			}}
			onclick={() => fileInput?.click()}
			onkeydown={(e) => (e.key === 'Enter' || e.key === ' ') && fileInput?.click()}
		>
			<Upload size={16} />
			<span class="bn-dim font-mono text-[10.5px]">{uploading > 0 ? `uploading ${uploading} file${uploading === 1 ? '' : 's'}…` : 'drop images / video here — or click to browse'}</span>
			<input
				bind:this={fileInput}
				type="file"
				multiple
				accept="image/*,video/*"
				class="hidden"
				onchange={(e) => {
					const t = e.currentTarget as HTMLInputElement;
					if (t.files?.length) void uploadFiles(t.files);
					t.value = '';
				}}
			/>
		</div>
		{#if media.length > 0}
			<div class="mt-2 flex flex-wrap gap-1.5">
				{#each media as m (m.url)}
					<span class="soc-chip bn-muted flex items-center gap-1.5 px-2 py-1 font-mono text-[10px]">
						<Link2 size={12} /><span class="max-w-[180px] truncate">{m.name}</span>
						<button type="button" aria-label={`remove ${m.name}`} onclick={() => (media = media.filter((x) => x.url !== m.url))}><X size={12} /></button>
					</span>
				{/each}
			</div>
		{/if}
		<div class="mt-3 flex flex-wrap items-center gap-2">
			<label class="pc-input flex flex-1 items-center gap-2 px-2.5 py-1.5">
				<Link2 size={14} class="bn-dim shrink-0" />
				<input bind:value={mediaUrl} placeholder="or paste a media URL (optional)" class="bn-text w-full bg-transparent font-mono text-[11px] outline-none" />
			</label>
			<label class="pc-input flex items-center gap-2 px-2.5 py-1.5">
				<Clock size={14} class="bn-dim shrink-0" />
				<input type="datetime-local" bind:value={scheduledFor} class="bn-text bg-transparent font-mono text-[11px] outline-none [color-scheme:dark]" />
			</label>
		</div>
		<div class="mt-3 flex items-center justify-between gap-3">
			<span class="bn-dim font-mono text-[10px]">Publishes for real via Zernio — scheduled posts go out at their time.</span>
			<button type="button" class="soc-primary flex items-center gap-2 whitespace-nowrap px-3.5 py-[7px] text-[12.5px] font-semibold" disabled={busy} onclick={submit}>
				{#if busy}<span class="font-mono text-[11px]">posting…</span>{:else if done}<span class="font-mono text-[11px]">{done}</span>{:else}<Send size={13} /> Post{/if}
			</button>
		</div>
		{#if error}<p class="soc-err mt-2 font-mono text-[11px]" data-part="composer-error">{error}</p>{/if}
	</div>

	<div class="bn-card p-1">
		<div class="flex items-center justify-between px-3 py-2.5">
			<span class="bn-dim font-mono text-[10px] uppercase tracking-[0.2em]">Queue</span>
			<span class="bn-muted font-mono text-[10px]">{queued.length} pending</span>
		</div>
		<div class="flex max-h-[280px] flex-col gap-1 overflow-y-auto px-1 pb-1">
			{#if queued.length === 0}
				<p class="bn-dim px-3 py-6 text-center font-mono text-[10.5px]">nothing queued yet</p>
			{/if}
			{#each queued as post (post.id)}
				<div class="pc-panel px-3 py-2.5">
					<p class="bn-text line-clamp-2 text-[12px] leading-snug">{post.caption}</p>
					<div class="mt-2 flex flex-wrap items-center gap-1.5">
						{#each post.platforms as p (p)}<span class="bn-dim font-mono text-[9px] uppercase tracking-wider">{platformLabel(p)}</span>{/each}
						<span class="ml-auto"><Badge tone={post.scheduledFor ? 'warn' : 'accent'}>{post.scheduledFor ? fmtWhen(post.scheduledFor) : 'queued'}</Badge></span>
					</div>
				</div>
			{/each}
		</div>
	</div>
</div>
