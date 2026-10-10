<script lang="ts">
	import { Dialog } from 'bits-ui';
	import { X } from 'lucide-svelte';
	import type { Task, TeamMemberListResponse, CreateTaskData } from '$lib/api';
	import { api } from '$lib/api';

	interface Props {
		open: boolean;
		projectId: string;
		tasks: Task[];
		teamMembers: TeamMemberListResponse[];
		onClose: () => void;
		onTaskCreated: () => Promise<void>;
	}

	let { open = $bindable(), projectId, tasks, teamMembers, onClose, onTaskCreated }: Props = $props();

	let newTask = $state<CreateTaskData>({
		title: '',
		description: '',
		priority: 'medium',
		due_date: '',
		estimated_hours: undefined,
		start_date: undefined,
		parent_task_id: undefined,
		assignee_id: undefined
	});

	let createError = $state('');

	function resetForm() {
		newTask = {
			title: '',
			description: '',
			priority: 'medium',
			due_date: '',
			estimated_hours: undefined,
			start_date: undefined,
			parent_task_id: undefined,
			assignee_id: undefined
		};
		createError = '';
	}

	async function handleCreateTask(e: Event) {
		e.preventDefault();
		if (!newTask.title.trim()) return;
		createError = '';
		try {
			await api.createTask({ ...newTask, project_id: projectId });
			await onTaskCreated();
			resetForm();
			onClose();
		} catch (err) {
			createError = (err as Error).message || 'Failed to create task';
			console.error('Error creating task:', err);
		}
	}

	function getPriorityDot(priority: string): string {
		switch (priority) {
			case 'critical': return '#ef4444';
			case 'high': return '#f97316';
			case 'medium': return '#eab308';
			default: return '#9ca3af';
		}
	}
</script>

<Dialog.Root bind:open>
	<Dialog.Portal>
		<Dialog.Overlay class="bos-modal-overlay" style="position:fixed;padding:0;" />
		<Dialog.Content class="pm-add-task-modal" aria-describedby={undefined}>
			<form onsubmit={handleCreateTask} class="pm-atm__form">
				<!-- Header -->
				<div class="pm-atm__header">
					<Dialog.Title class="pm-atm__title">Add Task</Dialog.Title>
					<Dialog.Close class="pm-atm__close" onclick={() => { resetForm(); onClose(); }} aria-label="Close">
						<X size={16} />
					</Dialog.Close>
				</div>

				<!-- Body -->
				<div class="pm-atm__body">
					<!-- Title -->
					<div class="pm-atm__field">
						<label for="atm-title" class="pm-atm__label">Title</label>
						<input
							id="atm-title"
							type="text"
							bind:value={newTask.title}
							class="pm-atm__input"
							placeholder="What needs to be done?"
							required
						/>
					</div>

					<!-- Description -->
					<div class="pm-atm__field">
						<label for="atm-desc" class="pm-atm__label">Description <span class="pm-atm__optional">(optional)</span></label>
						<textarea
							id="atm-desc"
							bind:value={newTask.description}
							class="pm-atm__textarea"
							rows="2"
							placeholder="Add more details..."
						></textarea>
					</div>

					<!-- Priority + Due Date -->
					<div class="pm-atm__row">
						<div class="pm-atm__field">
							<label class="pm-atm__label">Priority</label>
							<div class="pm-atm__priority-group">
								{#each [
									{ value: 'low' as const, label: 'Low' },
									{ value: 'medium' as const, label: 'Med' },
									{ value: 'high' as const, label: 'High' },
									{ value: 'critical' as const, label: 'Crit' }
								] as opt}
									<button
										type="button"
										onclick={() => newTask.priority = opt.value}
										class="pm-atm__priority-btn {newTask.priority === opt.value ? 'pm-atm__priority-btn--active' : ''}"
									>
										<span class="pm-atm__priority-dot" style="background: {getPriorityDot(opt.value)}"></span>
										{opt.label}
									</button>
								{/each}
							</div>
						</div>
						<div class="pm-atm__field">
							<label for="atm-due" class="pm-atm__label">Due Date</label>
							<input id="atm-due" type="date" bind:value={newTask.due_date} class="pm-atm__input" />
						</div>
					</div>

					<!-- Estimated Hours + Start Date -->
					<div class="pm-atm__row">
						<div class="pm-atm__field">
							<label for="atm-hours" class="pm-atm__label">Hours <span class="pm-atm__optional">(est.)</span></label>
							<input
								id="atm-hours"
								type="number"
								min="0"
								step="0.5"
								bind:value={newTask.estimated_hours}
								class="pm-atm__input"
								placeholder="0"
							/>
						</div>
						<div class="pm-atm__field">
							<label for="atm-start" class="pm-atm__label">Start Date</label>
							<input id="atm-start" type="date" bind:value={newTask.start_date} class="pm-atm__input" />
						</div>
					</div>

					<!-- Parent Task + Assignee -->
					<div class="pm-atm__row">
						<div class="pm-atm__field">
							<label for="atm-parent" class="pm-atm__label">Parent <span class="pm-atm__optional">(opt.)</span></label>
							<select id="atm-parent" bind:value={newTask.parent_task_id} class="pm-atm__select">
								<option value="">None</option>
								{#each tasks.filter((t) => t.status !== 'done') as task}
									<option value={task.id}>{task.title}</option>
								{/each}
							</select>
						</div>
						<div class="pm-atm__field">
							<label for="atm-assignee" class="pm-atm__label">Assignee <span class="pm-atm__optional">(opt.)</span></label>
							<select id="atm-assignee" bind:value={newTask.assignee_id} class="pm-atm__select">
								<option value="">Unassigned</option>
								{#each teamMembers as member}
									<option value={member.id}>{member.name}</option>
								{/each}
							</select>
						</div>
					</div>

					{#if createError}
						<div class="pm-atm__error">
							{createError}
						</div>
					{/if}
				</div>

				<!-- Footer -->
				<div class="pm-atm__footer">
					<button type="button" onclick={() => { resetForm(); onClose(); }} class="pm-atm__btn pm-atm__btn--ghost">
						Cancel
					</button>
					<button type="submit" disabled={!newTask.title.trim()} class="pm-atm__btn pm-atm__btn--primary">
						Add Task
					</button>
				</div>
			</form>
		</Dialog.Content>
	</Dialog.Portal>
</Dialog.Root>

<style>
	:global(.pm-add-task-modal) {
		position: fixed;
		left: 50%;
		top: 50%;
		transform: translate(-50%, -52%);
		width: calc(100% - 2rem);
		max-width: 440px;
		max-height: calc(85vh - 60px);
		z-index: 1000;
		background: var(--dbg, #fff);
		border: 1px solid var(--dbd, #e0e0e0);
		border-radius: 14px;
		box-shadow: 0 20px 40px -8px rgba(0, 0, 0, 0.15);
		overflow: hidden;
		display: flex;
		flex-direction: column;
	}
	:global(.dark .pm-add-task-modal) {
		background: var(--dbg, #141414);
		border-color: var(--dbd, #1e1e1e);
		box-shadow: 0 20px 40px -8px rgba(0, 0, 0, 0.5);
	}
	:global(.pm-atm__form) { display: flex; flex-direction: column; height: 100%; overflow: hidden; }
	:global(.pm-atm__header) { display: flex; align-items: center; justify-content: space-between; padding: 18px 20px 0; }
	:global(.pm-atm__title) { font-size: 16px; font-weight: 700; color: var(--dt, #111); letter-spacing: -0.02em; margin: 0; }
	:global(.pm-atm__close) { display: flex; align-items: center; justify-content: center; width: 26px; height: 26px; border-radius: 6px; border: none; background: transparent; color: var(--dt3, #888); cursor: pointer; transition: all 0.15s; }
	:global(.pm-atm__close:hover) { background: var(--dbg3, #eee); color: var(--dt, #111); }
	:global(.pm-atm__body) { flex: 1; overflow-y: auto; padding: 16px 20px 20px; display: flex; flex-direction: column; gap: 14px; }
	:global(.pm-atm__field) { display: flex; flex-direction: column; gap: 4px; }
	:global(.pm-atm__row) { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; }
	:global(.pm-atm__label) { font-size: 11px; font-weight: 600; color: var(--dt2, #555); text-transform: uppercase; letter-spacing: 0.05em; }
	:global(.pm-atm__optional) { font-weight: 400; color: var(--dt4, #bbb); text-transform: none; letter-spacing: 0; }

	:global(.pm-atm__input),
	:global(.pm-atm__textarea),
	:global(.pm-atm__select) {
		width: 100%;
		padding: 7px 10px;
		font-size: 13px;
		color: var(--dt, #111);
		background: var(--dbg2, #f5f5f5);
		border: 1px solid var(--dbd, #e0e0e0);
		border-radius: 8px;
		outline: none;
		font-family: inherit;
		transition: border-color 0.15s;
	}
	:global(.pm-atm__input:focus),
	:global(.pm-atm__textarea:focus),
	:global(.pm-atm__select:focus) {
		border-color: var(--dt, #111);
		background: var(--dbg, #fff);
	}
	:global(.dark .pm-atm__input:focus),
	:global(.dark .pm-atm__textarea:focus),
	:global(.dark .pm-atm__select:focus) {
		background: var(--dbg, #141414);
	}
	:global(.pm-atm__input::placeholder),
	:global(.pm-atm__textarea::placeholder) { color: var(--dt4, #bbb); }
	:global(.pm-atm__textarea) { resize: vertical; min-height: 48px; line-height: 1.5; }
	:global(.pm-atm__select) { cursor: pointer; appearance: auto; }

	/* Priority pills */
	:global(.pm-atm__priority-group) { display: flex; gap: 4px; }
	:global(.pm-atm__priority-btn) {
		flex: 1; display: flex; align-items: center; justify-content: center; gap: 4px;
		padding: 6px 4px; font-size: 11px; font-weight: 500; color: var(--dt3, #888);
		background: transparent; border: 1px solid var(--dbd, #e0e0e0); border-radius: 6px;
		cursor: pointer; transition: all 0.15s;
	}
	:global(.pm-atm__priority-btn:hover) { border-color: var(--dt3, #888); color: var(--dt, #111); }
	:global(.pm-atm__priority-btn--active) { border-color: var(--dt, #111); color: var(--dt, #111); font-weight: 600; background: var(--dbg, #fff); }
	:global(.dark .pm-atm__priority-btn--active) { background: var(--dbg3, #1e1e1e); }
	:global(.pm-atm__priority-dot) { width: 6px; height: 6px; border-radius: 50%; flex-shrink: 0; }

	/* Error */
	:global(.pm-atm__error) { padding: 8px 10px; font-size: 12px; color: var(--bos-status-error, #ef4444); background: var(--bos-status-error-bg); border-radius: 6px; }

	/* Footer */
	:global(.pm-atm__footer) { display: flex; align-items: center; justify-content: flex-end; gap: 8px; padding: 12px 20px; border-top: 1px solid var(--dbd, #e0e0e0); }
	:global(.pm-atm__btn) { padding: 7px 16px; font-size: 12px; font-weight: 600; border-radius: 6px; border: none; cursor: pointer; transition: all 0.15s; }
	:global(.pm-atm__btn--ghost) { background: transparent; color: var(--dt2, #555); }
	:global(.pm-atm__btn--ghost:hover) { background: var(--dbg2, #f5f5f5); color: var(--dt, #111); }
	:global(.pm-atm__btn--primary) { background: var(--bos-btn-cta-bg, #111); color: var(--bos-btn-cta-text, #fff); box-shadow: var(--bos-btn-cta-glow); border: 1px solid var(--bos-btn-cta-border); }
	:global(.pm-atm__btn--primary:hover:not(:disabled)) { box-shadow: var(--bos-btn-cta-glow-hover); transform: translateY(-0.5px); }
	:global(.pm-atm__btn--primary:disabled) { opacity: 0.3; cursor: not-allowed; box-shadow: none; }
</style>
