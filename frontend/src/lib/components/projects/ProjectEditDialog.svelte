<script lang="ts">
	import { Dialog } from 'bits-ui';
	import type { Project, ClientListResponse } from '$lib/api';
	import { api } from '$lib/api';

	interface Props {
		open: boolean;
		project: Project;
		clients: ClientListResponse[];
		onClose: () => void;
		onProjectUpdate: () => Promise<void>;
	}

	let { open = $bindable(), project, clients, onClose, onProjectUpdate }: Props = $props();

	let editForm = $state({
		name: project.name,
		description: project.description || '',
		status: project.status as 'active' | 'paused' | 'completed' | 'archived',
		priority: project.priority as 'critical' | 'high' | 'medium' | 'low',
		client_name: project.client_name || '',
		project_type: project.project_type
	});

	let isSaving = $state(false);

	// Sync editForm when project prop changes
	$effect(() => {
		editForm = {
			name: project.name,
			description: project.description || '',
			status: project.status,
			priority: project.priority,
			client_name: project.client_name || '',
			project_type: project.project_type
		};
	});

	async function handleSave(e: Event) {
		e.preventDefault();
		isSaving = true;
		try {
			await api.updateProject(project.id, editForm);
			await onProjectUpdate();
			onClose();
		} catch (err) {
			console.error('Error saving project:', err);
		} finally {
			isSaving = false;
		}
	}
</script>

<Dialog.Root bind:open>
	<Dialog.Portal>
		<Dialog.Overlay class="fixed inset-0 bg-black/40 z-50" />
		<Dialog.Content class="prm-dialog fixed top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 rounded-2xl shadow-xl p-6 w-full max-w-md z-50">
			<Dialog.Title class="prm-dialog__title">Edit Project</Dialog.Title>

			<form onsubmit={handleSave} class="space-y-4">
				<div>
					<label for="edit-name" class="prm-dialog__label">Name</label>
					<input id="edit-name" type="text" bind:value={editForm.name} class="input input-square" required />
				</div>

				<div>
					<label for="edit-client" class="prm-dialog__label">Client</label>
					<select id="edit-client" bind:value={editForm.client_name} class="input input-square">
						<option value="">No client</option>
						{#each clients as client}
							<option value={client.name}>{client.name}</option>
						{/each}
					</select>
				</div>

				<div>
					<label for="edit-description" class="prm-dialog__label">Description</label>
					<textarea
						id="edit-description"
						bind:value={editForm.description}
						class="input input-square resize-none"
						rows="3"
					></textarea>
				</div>

				<div class="grid grid-cols-2 gap-3">
					<div>
						<label for="edit-status" class="prm-dialog__label">Status</label>
						<select id="edit-status" bind:value={editForm.status} class="input input-square">
							<option value="active">Active</option>
							<option value="paused">Paused</option>
							<option value="completed">Completed</option>
							<option value="archived">Archived</option>
						</select>
					</div>
					<div>
						<label for="edit-priority" class="prm-dialog__label">Priority</label>
						<select id="edit-priority" bind:value={editForm.priority} class="input input-square">
							<option value="low">Low</option>
							<option value="medium">Medium</option>
							<option value="high">High</option>
							<option value="critical">Critical</option>
						</select>
					</div>
				</div>

				<div>
					<label for="edit-type" class="prm-dialog__label">Type</label>
					<select id="edit-type" bind:value={editForm.project_type} class="input input-square">
						<option value="internal">Internal</option>
						<option value="client_work">Client Work</option>
						<option value="learning">Learning</option>
					</select>
				</div>

				<div class="flex gap-3 pt-2">
					<button type="button" onclick={onClose} class="btn-pill btn-pill-ghost flex-1">Cancel</button>
					<button type="submit" disabled={isSaving} class="btn-pill btn-pill-primary flex-1">
						{isSaving ? 'Saving...' : 'Save Changes'}
					</button>
				</div>
			</form>
		</Dialog.Content>
	</Dialog.Portal>
</Dialog.Root>

<style>
	.prm-dialog {
		background: var(--dbg, #fff);
	}
	.prm-dialog__title {
		font-size: 1.125rem;
		font-weight: 600;
		color: var(--dt, #111);
		margin-bottom: 1rem;
	}
	.prm-dialog__label {
		display: block;
		font-size: 0.875rem;
		font-weight: 500;
		color: var(--dt, #111);
		margin-bottom: 0.25rem;
	}
</style>
