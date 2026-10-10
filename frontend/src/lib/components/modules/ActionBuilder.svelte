<script lang="ts">
	import { Plus, X, GripVertical } from 'lucide-svelte';
	import type { ModuleAction, ModuleActionType } from '$lib/types/modules';

	interface Props {
		actions: ModuleAction[];
		onActionsChange: (actions: ModuleAction[]) => void;
	}

	let { actions, onActionsChange }: Props = $props();

	function addAction() {
		const newAction: ModuleAction = {
			name: '',
			description: '',
			type: 'function',
			parameters: {},
			returns: {}
		};
		onActionsChange([...actions, newAction]);
	}

	function removeAction(index: number) {
		onActionsChange(actions.filter((_, i) => i !== index));
	}

	function updateAction(index: number, field: keyof ModuleAction, value: unknown) {
		const updated = [...actions];
		updated[index] = { ...updated[index], [field]: value };
		onActionsChange(updated);
	}

	const actionTypes: ModuleActionType[] = ['function', 'api', 'workflow'];
</script>

<div class="am-ab">
	<!-- Actions List -->
	{#each actions as action, index}
		<div class="am-ab__card">
			<!-- Header with drag handle and delete -->
			<div class="am-ab__card-header">
				<button
					type="button"
					class="am-ab__drag"
					title="Drag to reorder"
					aria-label="Drag to reorder"
				>
					<GripVertical class="w-5 h-5" />
				</button>
				<span class="am-ab__card-num">Action {index + 1}</span>
				<button
					type="button"
					onclick={() => removeAction(index)}
					class="am-ab__remove"
					title="Remove action"
					aria-label="Remove action"
				>
					<X class="w-5 h-5" />
				</button>
			</div>

			<!-- Action Fields -->
			<div class="am-ab__grid">
				<!-- Name -->
				<div class="am-ab__field">
					<label class="am-ab__label">
						Name <span class="am-ab__req">*</span>
					</label>
					<input
						type="text"
						value={action.name}
						oninput={(e) => updateAction(index, 'name', e.currentTarget.value)}
						placeholder="e.g., sendEmail"
						class="am-ab__input"
						required
						aria-label="Action name"
					/>
				</div>

				<!-- Type -->
				<div class="am-ab__field">
					<label class="am-ab__label">
						Type <span class="am-ab__req">*</span>
					</label>
					<select
						value={action.type}
						onchange={(e) => updateAction(index, 'type', e.currentTarget.value)}
						class="am-ab__input am-ab__select"
						aria-label="Action type"
					>
						{#each actionTypes as type}
							<option value={type}>{type.charAt(0).toUpperCase() + type.slice(1)}</option>
						{/each}
					</select>
				</div>

				<!-- Description -->
				<div class="am-ab__field am-ab__field--full">
					<label class="am-ab__label">
						Description <span class="am-ab__req">*</span>
					</label>
					<textarea
						value={action.description}
						oninput={(e) => updateAction(index, 'description', e.currentTarget.value)}
						placeholder="Describe what this action does..."
						rows="2"
						class="am-ab__input am-ab__textarea"
						required
						aria-label="Action description"
					></textarea>
				</div>

				<!-- Parameters (JSON) -->
				<div class="am-ab__field">
					<label class="am-ab__label">
						Parameters (JSON)
					</label>
					<textarea
						value={JSON.stringify(action.parameters, null, 2)}
						oninput={(e) => {
							try {
								const parsed = JSON.parse(e.currentTarget.value);
								updateAction(index, 'parameters', parsed);
							} catch {
								// Invalid JSON, ignore
							}
						}}
						placeholder={'{"param1": "type", "param2": "type"}'}
						rows="3"
						class="am-ab__input am-ab__textarea am-ab__mono"
						aria-label="Parameters JSON"
					></textarea>
				</div>

				<!-- Returns (JSON) -->
				<div class="am-ab__field">
					<label class="am-ab__label">
						Returns (JSON)
					</label>
					<textarea
						value={JSON.stringify(action.returns, null, 2)}
						oninput={(e) => {
							try {
								const parsed = JSON.parse(e.currentTarget.value);
								updateAction(index, 'returns', parsed);
							} catch {
								// Invalid JSON, ignore
							}
						}}
						placeholder={'{"result": "type"}'}
						rows="3"
						class="am-ab__input am-ab__textarea am-ab__mono"
						aria-label="Returns JSON"
					></textarea>
				</div>
			</div>
		</div>
	{/each}

	<!-- Add Action Button -->
	<button
		type="button"
		onclick={addAction}
		class="am-ab__add"
		aria-label="Add action"
	>
		<Plus class="w-5 h-5" />
		<span>Add Action</span>
	</button>
</div>

<style>
	.am-ab {
		display: flex;
		flex-direction: column;
		gap: 16px;
	}
	.am-ab__card {
		background: var(--dbg2, #f5f5f5);
		border: 1px solid var(--dbd, #e0e0e0);
		border-radius: 12px;
		padding: 16px;
	}
	.am-ab__card-header {
		display: flex;
		align-items: center;
		gap: 8px;
		margin-bottom: 12px;
	}
	.am-ab__drag {
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 4px;
		border: none;
		background: none;
		color: var(--dt3, #888);
		cursor: move;
		border-radius: 6px;
	}
	.am-ab__drag:hover {
		background: var(--dbg3, #eee);
	}
	.am-ab__card-num {
		font-size: 13px;
		font-weight: 500;
		color: var(--dt2, #555);
	}
	.am-ab__remove {
		margin-left: auto;
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 4px;
		border: none;
		background: none;
		color: var(--color-error, #ef4444);
		cursor: pointer;
		border-radius: 6px;
		transition: background .15s;
	}
	.am-ab__remove:hover {
		background: rgba(239, 68, 68, 0.08);
	}
	.am-ab__grid {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 12px;
	}
	.am-ab__field {
		display: flex;
		flex-direction: column;
	}
	.am-ab__field--full {
		grid-column: 1 / -1;
	}
	.am-ab__label {
		font-size: 13px;
		font-weight: 500;
		color: var(--dt2, #555);
		margin-bottom: 4px;
	}
	.am-ab__req {
		color: var(--color-error, #ef4444);
	}
	.am-ab__input {
		width: 100%;
		padding: 8px 12px;
		border: 1px solid var(--dbd, #e0e0e0);
		border-radius: 8px;
		background: var(--dbg, #fff);
		color: var(--dt, #111);
		font-size: 13px;
		outline: none;
		transition: border-color .15s;
	}
	.am-ab__input:focus {
		border-color: var(--accent-blue, #3b82f6);
	}
	.am-ab__input::placeholder {
		color: var(--dt4, #bbb);
	}
	.am-ab__textarea {
		resize: vertical;
	}
	.am-ab__mono {
		font-family: monospace;
	}
	.am-ab__select {
		appearance: none;
		background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 24 24' fill='none' stroke='%23888' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'%3E%3Cpolyline points='6 9 12 15 18 9'/%3E%3C/svg%3E");
		background-repeat: no-repeat;
		background-position: right 12px center;
		padding-right: 32px;
		cursor: pointer;
	}
	.am-ab__add {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 8px;
		width: 100%;
		padding: 14px;
		border: 2px dashed var(--dbd, #e0e0e0);
		border-radius: 12px;
		background: transparent;
		color: var(--dt2, #555);
		font-size: 14px;
		font-weight: 500;
		cursor: pointer;
		transition: all .15s;
	}
	.am-ab__add:hover {
		border-color: var(--dt3, #888);
		color: var(--dt, #111);
		background: var(--dbg2, #f5f5f5);
	}
</style>
