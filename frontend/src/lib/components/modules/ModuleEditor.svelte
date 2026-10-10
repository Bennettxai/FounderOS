<script lang="ts">
	import type { ModuleCategory } from '$lib/types/modules';
	import { categoryLabels } from '$lib/types/modules';

	interface Props {
		name: string;
		description: string;
		category: ModuleCategory;
		onNameChange: (value: string) => void;
		onDescriptionChange: (value: string) => void;
		onCategoryChange: (value: ModuleCategory) => void;
	}

	let { name, description, category, onNameChange, onDescriptionChange, onCategoryChange }: Props = $props();

	const categories: ModuleCategory[] = [
		'productivity',
		'communication',
		'finance',
		'analytics',
		'automation',
		'integration',
		'utilities',
		'custom'
	];
</script>

<div class="am-editor">
	<!-- Name -->
	<div class="am-editor__field">
		<label class="am-editor__label">
			Module Name <span class="am-editor__req">*</span>
		</label>
		<input
			type="text"
			value={name}
			oninput={(e) => onNameChange(e.currentTarget.value)}
			placeholder="e.g., Email Automation"
			class="am-editor__input"
			required
			aria-label="Module name"
		/>
	</div>

	<!-- Description -->
	<div class="am-editor__field">
		<label class="am-editor__label">
			Description <span class="am-editor__req">*</span>
		</label>
		<textarea
			value={description}
			oninput={(e) => onDescriptionChange(e.currentTarget.value)}
			placeholder="Describe what your module does..."
			rows="4"
			class="am-editor__input am-editor__textarea"
			required
			aria-label="Module description"
		></textarea>
	</div>

	<!-- Category -->
	<div class="am-editor__field">
		<label class="am-editor__label">
			Category <span class="am-editor__req">*</span>
		</label>
		<select
			value={category}
			onchange={(e) => onCategoryChange(e.currentTarget.value as ModuleCategory)}
			class="am-editor__input am-editor__select"
			aria-label="Module category"
		>
			{#each categories as cat}
				<option value={cat}>{categoryLabels[cat]}</option>
			{/each}
		</select>
	</div>
</div>

<style>
	.am-editor {
		display: flex;
		flex-direction: column;
		gap: 16px;
	}
	.am-editor__field {
		display: flex;
		flex-direction: column;
	}
	.am-editor__label {
		font-size: 13px;
		font-weight: 500;
		color: var(--dt2, #555);
		margin-bottom: 8px;
	}
	.am-editor__req {
		color: var(--color-error, #ef4444);
	}
	.am-editor__input {
		width: 100%;
		padding: 10px 14px;
		border: 1px solid var(--dbd, #e0e0e0);
		border-radius: 10px;
		background: var(--dbg2, #f5f5f5);
		color: var(--dt, #111);
		font-size: 14px;
		outline: none;
		transition: border-color .15s;
	}
	.am-editor__input:focus {
		border-color: var(--accent-blue, #3b82f6);
	}
	.am-editor__input::placeholder {
		color: var(--dt4, #bbb);
	}
	.am-editor__textarea {
		resize: vertical;
	}
	.am-editor__select {
		appearance: none;
		background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 24 24' fill='none' stroke='%23888' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'%3E%3Cpolyline points='6 9 12 15 18 9'/%3E%3C/svg%3E");
		background-repeat: no-repeat;
		background-position: right 14px center;
		padding-right: 36px;
		cursor: pointer;
	}
</style>
