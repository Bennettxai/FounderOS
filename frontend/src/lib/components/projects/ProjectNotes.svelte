<script lang="ts">
	import type { Project } from '$lib/api';
	import { api } from '$lib/api';
	import { formatDate, formatTime } from '$lib/utils/project';

	interface Props {
		project: Project;
		onProjectUpdate: () => Promise<void>;
	}

	let { project, onProjectUpdate }: Props = $props();

	let newNote = $state('');
	let isAddingNote = $state(false);

	async function handleAddNote() {
		if (!newNote.trim()) return;
		isAddingNote = true;
		try {
			await api.addProjectNote(project.id, newNote);
			await onProjectUpdate();
			newNote = '';
		} catch (err) {
			console.error('Error adding note:', err);
		} finally {
			isAddingNote = false;
		}
	}
</script>

<div class="prm-notes">
	<h2 class="prm-notes__title">Notes</h2>

	<!-- Add Note -->
	<div class="mb-6">
		<textarea
			bind:value={newNote}
			placeholder="Add a note..."
			class="input input-square resize-none w-full"
			rows="3"
		></textarea>
		<div class="flex justify-end mt-2">
			<button
				onclick={handleAddNote}
				disabled={!newNote.trim() || isAddingNote}
				class="btn-pill btn-pill-primary btn-pill-sm"
			>
				{isAddingNote ? 'Adding...' : 'Add Note'}
			</button>
		</div>
	</div>

	<!-- Notes List -->
	{#if project.notes.length === 0}
		<div class="text-center py-8 prm-notes__empty">
			<svg class="w-12 h-12 mx-auto mb-3 opacity-50" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
			</svg>
			<p>No notes yet. Add your first note above.</p>
		</div>
	{:else}
		<div class="space-y-4">
			{#each project.notes as note}
				<div class="prm-notes__card">
					<p class="prm-notes__content">{note.content}</p>
					<p class="prm-notes__meta">
						{formatDate(note.created_at)} at {formatTime(note.created_at)}
					</p>
				</div>
			{/each}
		</div>
	{/if}
</div>

<style>
	.prm-notes {
		background: var(--dbg, #fff);
		border: 1px solid var(--dbd, #e0e0e0);
		border-radius: 0.75rem;
		padding: 1.5rem;
	}
	.prm-notes__title {
		font-size: 1.125rem;
		font-weight: 500;
		color: var(--dt, #111);
		margin-bottom: 1rem;
	}
	.prm-notes__empty {
		color: var(--dt3, #888);
	}
	.prm-notes__card {
		padding: 1rem;
		background: var(--dbg2, #f5f5f5);
		border-radius: 0.75rem;
	}
	.prm-notes__content {
		color: var(--dt, #111);
		white-space: pre-wrap;
	}
	.prm-notes__meta {
		font-size: 0.75rem;
		color: var(--dt3, #888);
		margin-top: 0.75rem;
	}
</style>
