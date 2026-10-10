<script lang="ts">
	import type { ContentItem } from '$lib/api/content';
	import { Loader2, X } from 'lucide-svelte';

	type Props = { item: ContentItem; liveLink: string; saving: boolean; onClose: () => void; onConfirm: () => void };
	let { item, liveLink = $bindable(), saving, onClose, onConfirm }: Props = $props();
</script>

<div class="overlay" role="presentation" onclick={onClose} onkeydown={(event) => event.key === 'Escape' && onClose()}>
	<div class="modal" role="dialog" aria-modal="true" aria-labelledby="post-title" tabindex="-1" onclick={(event) => event.stopPropagation()} onkeydown={() => {}}>
		<header><div><h2 id="post-title">Mark content as posted</h2><p>Record the published URL to complete this pipeline step.</p></div><button class="close" onclick={onClose} aria-label="Close"><X size={18} /></button></header>
		<section class="preview"><span>{item.content_type}</span><h3>{item.title}</h3>{#if item.caption}<p>{item.caption}</p>{/if}</section>
		<div class="fields"><label><span>Live link</span><input bind:value={liveLink} placeholder="https://..." /></label><label><span>Platform / channel</span><input value={item.channel || 'Not selected'} disabled /></label></div>
		<footer><button onclick={onClose}>Cancel</button><button class="primary" onclick={onConfirm} disabled={saving || !liveLink.trim()}>{#if saving}<span class="spinner"><Loader2 size={15} /></span>{/if}Mark posted</button></footer>
	</div>
</div>

<style>
	.overlay { position: fixed; inset: 0; z-index: 100; display: grid; place-items: center; padding: 20px; background: rgb(0 0 0 / .48); }
	.modal { width: min(620px, 100%); border: 1px solid var(--dbd); border-radius: 8px; background: var(--dbg); box-shadow: 0 24px 80px rgb(0 0 0 / .25); }
	header { display: flex; justify-content: space-between; gap: 16px; padding: 16px 18px; border-bottom: 1px solid var(--dbd); }
	h2, h3 { margin: 0; color: var(--dt); letter-spacing: 0; } h2 { font-size: 1.05rem; } h3 { margin-top: 5px; font-size: .95rem; }
	header p, .preview p { margin: 4px 0 0; color: var(--dt3); font-size: .76rem; line-height: 1.45; }
	.close { display: grid; width: 32px; height: 32px; place-items: center; padding: 0; }
	.preview { margin: 16px 18px 0; padding: 13px; border: 1px solid var(--dbd); border-radius: 7px; background: color-mix(in srgb, var(--dt) 2%, var(--dbg)); }
	.preview > span { color: #0f766e; font-size: .68rem; font-weight: 750; text-transform: capitalize; }
	.fields { display: grid; grid-template-columns: 1fr 1fr; gap: 11px; padding: 14px 18px; }
	label { display: grid; gap: 6px; } label span { color: var(--dt2); font-size: .72rem; font-weight: 700; }
	input { width: 100%; min-width: 0; height: 36px; padding: 0 9px; border: 1px solid var(--dbd); border-radius: 6px; background: color-mix(in srgb, var(--dt) 2%, var(--dbg)); color: var(--dt); font: inherit; font-size: .8rem; }
	footer { display: flex; justify-content: flex-end; gap: 8px; padding: 0 18px 18px; }
	button { display: inline-flex; align-items: center; justify-content: center; gap: 7px; min-height: 34px; padding: 7px 11px; border: 1px solid var(--dbd); border-radius: 7px; background: transparent; color: var(--dt2); font: inherit; font-size: .78rem; font-weight: 680; cursor: pointer; }
	button.primary { border-color: var(--dt); background: var(--dt); color: var(--dbg); } button:disabled { opacity: .55; cursor: not-allowed; }
	.spinner { display: inline-flex; animation: spin .8s linear infinite; } @keyframes spin { to { transform: rotate(360deg); } }
	@media (max-width: 600px) { .fields { grid-template-columns: 1fr; } }
</style>
