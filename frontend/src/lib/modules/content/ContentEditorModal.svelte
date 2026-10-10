<script lang="ts">
	import type { ContentItem } from '$lib/api/content';
	import { Loader2, X } from 'lucide-svelte';
	import { CHANNELS, CONTENT_THEMES, STAGE_META, STAGES, TYPES, type ContentForm } from './model';

	type Props = {
		form: ContentForm;
		editing: ContentItem | null;
		profiles: string[];
		workstreams: string[];
		memberOptions: string[];
		themeOptions?: string[];
		stages?: typeof STAGES;
		saving: boolean;
		onClose: () => void;
		onSave: (event: SubmitEvent) => void;
	};

	let { form = $bindable(), editing, profiles, workstreams, memberOptions, themeOptions = CONTENT_THEMES, stages = STAGES, saving, onClose, onSave }: Props = $props();
</script>

<div class="overlay" role="presentation" onclick={onClose} onkeydown={(event) => event.key === 'Escape' && onClose()}>
	<div class="modal" role="dialog" aria-modal="true" aria-labelledby="content-editor-title" tabindex="-1" onclick={(event) => event.stopPropagation()} onkeydown={() => {}}>
		<header>
			<div><h2 id="content-editor-title">{editing ? 'Edit content' : 'New content'}</h2><p>Manage production, review, publishing, and assignment details.</p></div>
			<button class="close" type="button" onclick={onClose} aria-label="Close"><X size={18} /></button>
		</header>
		<form onsubmit={onSave}>
			<label class="field full"><span>Title</span><input bind:value={form.title} placeholder="Content title" required /></label>
			<div class="grid">
				<label class="field"><span>Workstream</span><input bind:value={form.category} list="content-workstreams" placeholder="Organic Content" /><datalist id="content-workstreams">{#each workstreams as value}<option value={value}></option>{/each}</datalist></label>
				<label class="field"><span>Stage</span><select bind:value={form.status}>{#each stages as stage}<option value={stage}>{STAGE_META[stage].label}</option>{/each}</select></label>
				<label class="field"><span>Type</span><select bind:value={form.content_type}>{#each TYPES as type}<option value={type}>{type}</option>{/each}</select></label>
				<label class="field"><span>Profile / brand</span><input bind:value={form.client} list="content-profiles" placeholder="Optional profile or brand" /><datalist id="content-profiles">{#each profiles as value}<option value={value}></option>{/each}</datalist></label>
				<label class="field"><span>Theme</span><select bind:value={form.theme}><option value="">Select theme</option>{#each themeOptions as theme}<option value={theme}>{theme}</option>{/each}{#if form.theme && !themeOptions.includes(form.theme)}<option value={form.theme}>{form.theme}</option>{/if}</select></label>
				<label class="field"><span>Campaign</span><input bind:value={form.campaign} placeholder="Launch, nurture, proof..." /></label>
				<label class="field"><span>Owner</span><select bind:value={form.owner}><option value="">Unassigned</option>{#each memberOptions as member}<option value={member}>{member}</option>{/each}{#if form.owner && !memberOptions.includes(form.owner)}<option value={form.owner}>{form.owner}</option>{/if}</select></label>
				<label class="field"><span>Editor</span><select bind:value={form.editor}><option value="">Unassigned</option>{#each memberOptions as member}<option value={member}>{member}</option>{/each}{#if form.editor && !memberOptions.includes(form.editor)}<option value={form.editor}>{form.editor}</option>{/if}</select></label>
				<label class="field"><span>Film date</span><input type="date" bind:value={form.due_date} /></label>
				<label class="field"><span>Post date</span><input type="date" bind:value={form.publish_date} /></label>
				<label class="field"><span>Platform / channel</span><select bind:value={form.channel}><option value="">Select platform</option>{#each CHANNELS as channel}<option value={channel}>{channel}</option>{/each}{#if form.channel && !CHANNELS.includes(form.channel)}<option value={form.channel}>{form.channel}</option>{/if}</select></label>
				<label class="field"><span>CTA</span><input bind:value={form.cta} placeholder="Book a call, reply, subscribe..." /></label>
			</div>
			<label class="field full"><span>Hook</span><textarea bind:value={form.hook} rows="2" placeholder="Opening line"></textarea></label>
			<label class="field full"><span>Script / copy / direction</span><textarea class="long" bind:value={form.body} rows="10" placeholder="Script, shot list, edit notes, or outline"></textarea></label>
			<label class="field full"><span>Caption</span><textarea bind:value={form.caption} rows="6" placeholder="Final post caption and platform copy"></textarea></label>
			<div class="grid">
				<label class="field"><span>Asset link</span><input bind:value={form.asset_link} placeholder="Drive, Dropbox, Frame.io..." /></label>
				<label class="field"><span>Review link</span><input bind:value={form.review_link} placeholder="Review or approval link" /></label>
				<label class="field"><span>Live link</span><input bind:value={form.link} placeholder="Published URL" /></label>
			</div>
			<section class="analytics">
				<div class="section-head"><h3>Analytics</h3><p>Track the numbers after this post goes live.</p></div>
				<div class="grid analytics-grid">
					<label class="field"><span>Views</span><input type="number" min="0" bind:value={form.views} /></label>
					<label class="field"><span>Reach</span><input type="number" min="0" bind:value={form.reach} /></label>
					<label class="field"><span>Likes</span><input type="number" min="0" bind:value={form.likes} /></label>
					<label class="field"><span>Comments</span><input type="number" min="0" bind:value={form.comments} /></label>
					<label class="field"><span>Saves</span><input type="number" min="0" bind:value={form.saves} /></label>
					<label class="field"><span>Shares</span><input type="number" min="0" bind:value={form.shares} /></label>
					<label class="field"><span>Reposts</span><input type="number" min="0" bind:value={form.reposts} /></label>
					<label class="field"><span>Follows</span><input type="number" min="0" bind:value={form.follows} /></label>
					<label class="field"><span>Profile activity</span><input type="number" min="0" bind:value={form.profile_activity} /></label>
					<label class="field"><span>Accounts engaged</span><input type="number" min="0" bind:value={form.accounts_engaged} /></label>
					<label class="field"><span>Avg watch seconds</span><input type="number" min="0" step="0.1" bind:value={form.avg_watch_time_seconds} /></label>
					<label class="field"><span>Retention %</span><input type="number" min="0" step="0.1" bind:value={form.retention_rate} /></label>
				</div>
				<label class="field full"><span>Analytics notes</span><textarea bind:value={form.analytics_notes} rows="3" placeholder="Why this won or lost, hook notes, audience signal..."></textarea></label>
			</section>
			<label class="field full"><span>Revision notes / feedback</span><textarea bind:value={form.revision_notes} rows="3"></textarea></label>
			<label class="field full"><span>Internal notes</span><textarea bind:value={form.notes} rows="3"></textarea></label>
			<footer><button type="button" onclick={onClose}>Cancel</button><button class="primary" type="submit" disabled={saving || !form.title.trim()}>{#if saving}<span class="spinner"><Loader2 size={15} /></span>{/if}{editing ? 'Save changes' : 'Add content'}</button></footer>
		</form>
	</div>
</div>

<style>
	.overlay { position: fixed; inset: 0; z-index: 20000; display: grid; place-items: center; padding: 20px; background: rgb(0 0 0 / .48); }
	.modal { width: min(840px, 100%); max-height: calc(100vh - 40px); overflow: auto; border: 1px solid var(--dbd); border-radius: 8px; background: var(--dbg); box-shadow: 0 24px 80px rgb(0 0 0 / .25); }
	header { position: sticky; top: 0; z-index: 2; display: flex; align-items: flex-start; justify-content: space-between; gap: 16px; padding: 16px 18px; border-bottom: 1px solid var(--dbd); background: var(--dbg); }
	h2 { margin: 0; color: var(--dt); font-size: 1.05rem; letter-spacing: 0; }
	header p { margin: 4px 0 0; color: var(--dt3); font-size: .76rem; }
	.close { display: grid; width: 32px; height: 32px; place-items: center; padding: 0; }
	form { display: grid; gap: 12px; padding: 18px; }
	.grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 11px; }
	.field { display: grid; gap: 6px; min-width: 0; }
	.field span { color: var(--dt2); font-size: .72rem; font-weight: 700; }
	input, select, textarea { width: 100%; min-width: 0; border: 1px solid var(--dbd); border-radius: 6px; outline: none; background: color-mix(in srgb, var(--dt) 2%, var(--dbg)); color: var(--dt); font: inherit; font-size: .8rem; }
	input, select { height: 36px; padding: 0 9px; } textarea { padding: 9px; resize: vertical; line-height: 1.45; }
	input:focus, select:focus, textarea:focus { border-color: #0f766e; box-shadow: 0 0 0 2px color-mix(in srgb, #0f766e 14%, transparent); }
	textarea.long { min-height: 190px; }
	.analytics { display: grid; gap: 11px; padding: 12px; border: 1px solid var(--dbd); border-radius: 7px; background: color-mix(in srgb, var(--dt) 1.5%, var(--dbg)); }
	.section-head { display: flex; align-items: baseline; justify-content: space-between; gap: 12px; }
	.section-head h3 { margin: 0; color: var(--dt); font-size: .86rem; letter-spacing: 0; }
	.section-head p { margin: 0; color: var(--dt3); font-size: .7rem; }
	.analytics-grid { grid-template-columns: repeat(4, minmax(0, 1fr)); }
	footer { display: flex; justify-content: flex-end; gap: 8px; padding-top: 4px; }
	button { display: inline-flex; align-items: center; justify-content: center; gap: 7px; min-height: 34px; padding: 7px 11px; border: 1px solid var(--dbd); border-radius: 7px; background: transparent; color: var(--dt2); font: inherit; font-size: .78rem; font-weight: 680; cursor: pointer; }
	button.primary { border-color: var(--dt); background: var(--dt); color: var(--dbg); } button:disabled { opacity: .55; cursor: not-allowed; }
	.spinner { display: inline-flex; animation: spin .8s linear infinite; } @keyframes spin { to { transform: rotate(360deg); } }
	@media (max-width: 900px) { .analytics-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
	@media (max-width: 650px) { .overlay { padding: 0; } .modal { max-height: 100vh; border-radius: 0; } .grid, .analytics-grid { grid-template-columns: 1fr; } .section-head { display: grid; } }
</style>
