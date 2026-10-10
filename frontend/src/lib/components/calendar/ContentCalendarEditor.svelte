<script lang="ts">
	import { Loader2, Trash2, X } from 'lucide-svelte';
	import type { ContentItemInput, ContentStatus, ContentType } from '$lib/api/content';
	import { CONTENT_THEMES } from '$lib/modules/content/model';

	interface Props {
		form: ContentItemInput;
		contentProfiles: string[];
		workstreams: string[];
		stages: ContentStatus[];
		stageLabels: Record<string, string>;
		types: ContentType[];
		channels: string[];
		saving: boolean;
		deleting: boolean;
		onClose: () => void;
		onSave: (event: Event) => void | Promise<void>;
		onDelete: () => void | Promise<void>;
	}

	let {
		form = $bindable(),
		contentProfiles,
		workstreams,
		stages,
		stageLabels,
		types,
		channels,
		saving,
		deleting,
		onClose,
		onSave,
		onDelete
	}: Props = $props();
</script>

<div class="overlay" role="button" tabindex="0" onclick={onClose} onkeydown={(e) => e.key === 'Escape' && onClose()}>
	<div class="modal modal--content" role="dialog" tabindex="-1" onclick={(e) => e.stopPropagation()} onkeydown={() => {}}>
		<div class="modal-head">
			<div>
				<h2>Edit content</h2>
				<p>Update the ContentOS card. The calendar block follows this card.</p>
			</div>
			<button class="card-x" onclick={onClose} aria-label="Close"><X size={18} /></button>
		</div>
		<form onsubmit={onSave}>
			<label class="field field--full"><span>Title</span><input bind:value={form.title} required /></label>
			<div class="field-grid">
				<label class="field"><span>Workstream</span>
					<select bind:value={form.category}>
						{#each workstreams as stream}<option value={stream}>{stream}</option>{/each}
						{#if form.category && !workstreams.includes(form.category)}<option value={form.category}>{form.category}</option>{/if}
					</select>
				</label>
				<label class="field"><span>Stage</span>
					<select bind:value={form.status}>
						{#each stages as stage}<option value={stage}>{stageLabels[stage] || stage}</option>{/each}
					</select>
				</label>
				<label class="field"><span>Type</span>
					<select bind:value={form.content_type}>
						{#each types as type}<option value={type}>{type}</option>{/each}
					</select>
				</label>
				<label class="field"><span>Profile / brand</span>
					<select bind:value={form.client}>
						{#each contentProfiles as profile}<option value={profile}>{profile}</option>{/each}
						{#if form.client && !contentProfiles.includes(form.client)}<option value={form.client}>{form.client}</option>{/if}
					</select>
				</label>
				<label class="field"><span>Theme</span>
					<select bind:value={form.theme}>
						<option value="">Select theme</option>
						{#each CONTENT_THEMES as theme}<option value={theme}>{theme}</option>{/each}
						{#if form.theme && !CONTENT_THEMES.includes(form.theme)}<option value={form.theme}>{form.theme}</option>{/if}
					</select>
				</label>
				<label class="field"><span>Campaign</span><input bind:value={form.campaign} /></label>
				<label class="field"><span>Owner</span><input bind:value={form.owner} /></label>
				<label class="field"><span>Film date</span><input type="date" bind:value={form.due_date} /></label>
				<label class="field"><span>Post date</span><input type="date" bind:value={form.publish_date} /></label>
				<label class="field"><span>Platform / channel</span>
					<select bind:value={form.channel}>
						<option value="">Select platform</option>
						{#each channels as channel}<option value={channel}>{channel}</option>{/each}
						{#if form.channel && !channels.includes(form.channel)}<option value={form.channel}>{form.channel}</option>{/if}
					</select>
				</label>
				<label class="field"><span>CTA</span><input bind:value={form.cta} /></label>
			</div>
			<label class="field field--full"><span>Hook</span><textarea bind:value={form.hook} rows="2"></textarea></label>
			<label class="field field--full"><span>Script / copy / direction</span><textarea class="textarea-long" bind:value={form.body} rows="8"></textarea></label>
			<label class="field field--full"><span>Caption</span><textarea bind:value={form.caption} rows="5"></textarea></label>
			<div class="field-grid">
				<label class="field"><span>Asset link</span><input bind:value={form.asset_link} /></label>
				<label class="field"><span>Review link</span><input bind:value={form.review_link} /></label>
				<label class="field"><span>Live link</span><input bind:value={form.link} /></label>
			</div>
			<section class="analytics">
				<div class="section-head">
					<h3>Analytics</h3>
					<p>Post-performance data for reporting and future scraping agents.</p>
				</div>
				<div class="field-grid analytics-grid">
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
				<label class="field field--full"><span>Analytics notes</span><textarea bind:value={form.analytics_notes} rows="3" placeholder="Why this won or lost, hook notes, audience signal..."></textarea></label>
			</section>
			<label class="field field--full"><span>Internal notes</span><textarea bind:value={form.notes} rows="3"></textarea></label>
			<div class="modal-actions modal-actions--split">
				<button type="button" class="btn btn--danger" onclick={onDelete} disabled={deleting || saving}>
					{#if deleting}<Loader2 class="spin" size={15} />{:else}<Trash2 size={15} />{/if}Delete card
				</button>
				<div>
					<button type="button" class="btn btn--ghost" onclick={onClose}>Cancel</button>
					<button type="submit" class="btn btn--primary" disabled={saving || !form.title.trim()}>
						{#if saving}<Loader2 class="spin" size={15} />{/if}Save
					</button>
				</div>
			</div>
		</form>
	</div>
</div>

<style>
	.overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.55); display: flex; align-items: center; justify-content: center; z-index: 100; padding: 20px; }
	.modal { width: 100%; background: var(--dbg); border: 1px solid var(--dbd); border-radius: 16px; padding: 22px; box-shadow: 0 24px 60px rgba(0,0,0,0.5); }
	.modal--content { max-width: 820px; max-height: min(86vh, 860px); overflow-y: auto; }
	.modal-head { display: flex; align-items: center; justify-content: space-between; margin-bottom: 18px; gap: 12px; }
	.modal-head h2 { font-size: 1.05rem; font-weight: 640; margin: 0; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
	.modal-head p { margin: 4px 0 0; color: var(--dt3); font-size: 0.8rem; line-height: 1.35; }
	.card-x { display: inline-flex; align-items: center; justify-content: center; width: 28px; height: 28px; border-radius: 7px; border: none; background: transparent; color: var(--dt3); cursor: pointer; flex-shrink: 0; }
	.card-x:hover { background: color-mix(in srgb, var(--dt) 8%, transparent); }
	.field { display: flex; flex-direction: column; gap: 6px; margin-bottom: 14px; }
	.field span { font-size: 0.8rem; font-weight: 560; color: var(--dt2); }
	.field input, .field textarea, .field select { padding: 9px 12px; border-radius: 9px; border: 1px solid var(--dbd); background: var(--dbg); color: var(--dt); font-size: 0.86rem; outline: none; font-family: inherit; }
	.field textarea { resize: vertical; min-height: 70px; line-height: 1.45; }
	.field .textarea-long { min-height: 180px; }
	.field input:focus, .field textarea:focus, .field select:focus { border-color: color-mix(in srgb, var(--dt) 40%, transparent); }
	.field--full { grid-column: 1 / -1; }
	.field-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 0 12px; }
	.analytics { display: grid; gap: 12px; margin: 2px 0 14px; padding: 13px; border: 1px solid var(--dbd); border-radius: 10px; background: color-mix(in srgb, var(--dt) 2%, transparent); }
	.section-head { display: flex; align-items: baseline; justify-content: space-between; gap: 12px; }
	.section-head h3 { margin: 0; color: var(--dt); font-size: 0.9rem; letter-spacing: 0; }
	.section-head p { margin: 0; color: var(--dt3); font-size: 0.72rem; line-height: 1.35; }
	.analytics-grid { grid-template-columns: repeat(4, minmax(0, 1fr)); }
	.modal-actions { display: flex; justify-content: flex-end; gap: 10px; margin-top: 6px; }
	.modal-actions--split { justify-content: space-between; align-items: center; }
	.modal-actions--split > div { display: flex; justify-content: flex-end; gap: 10px; }
	.btn { display: inline-flex; align-items: center; gap: 7px; padding: 8px 15px; border-radius: 9px; font-size: 0.83rem; font-weight: 580; cursor: pointer; border: 1px solid transparent; white-space: nowrap; }
	.btn--primary { background: var(--dt); color: var(--dbg); }
	.btn--ghost { background: transparent; border-color: var(--dbd); color: var(--dt2); }
	.btn--danger { background: color-mix(in srgb, #ef4444 14%, transparent); color: #ef4444; }
	.btn:disabled { opacity: 0.55; cursor: not-allowed; }
	:global(.spin) { animation: spin 0.9s linear infinite; }
	@keyframes spin { to { transform: rotate(360deg); } }
	@media (max-width: 900px) { .analytics-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
	@media (max-width: 760px) { .field-grid, .analytics-grid { grid-template-columns: 1fr; } .section-head { display: grid; } }
</style>
