<script lang="ts">
	import type { ConfigField } from '$lib/api/templates/types';

	interface Props {
		schema: Record<string, ConfigField>;
		values: Record<string, string>;
		onchange: (key: string, value: string) => void;
		disabled?: boolean;
	}

	let { schema, values, onchange, disabled = false }: Props = $props();

	function getValue(key: string, field: ConfigField): string {
		return key in values ? values[key] : field.default;
	}

	function handleInput(key: string, event: Event) {
		const target = event.currentTarget as HTMLInputElement | HTMLSelectElement;
		onchange(key, target.value);
	}

	function handleToggle(key: string, field: ConfigField) {
		if (disabled) return;
		const current = getValue(key, field);
		onchange(key, current === 'true' ? 'false' : 'true');
	}
</script>

<div class="sfr-form">
	{#each Object.entries(schema) as [key, field]}
		<div class="sfr-row">
			<div class="sfr-label">
				<span class="sfr-label-text">
					{field.label}
					{#if field.required}
						<span class="sfr-required" aria-hidden="true">*</span>
					{/if}
				</span>
				{#if field.description}
					<span class="sfr-label-desc">{field.description}</span>
				{/if}
			</div>

			<div class="sfr-control">
				{#if field.type === 'boolean'}
					{@const isOn = getValue(key, field) === 'true'}
					<button
						type="button"
						role="switch"
						aria-checked={isOn}
						aria-label={field.label}
						class="sfr-toggle {isOn ? 'sfr-toggle--on' : ''}"
						{disabled}
						onclick={() => handleToggle(key, field)}
					>
						<span class="sfr-toggle-knob"></span>
					</button>

				{:else if field.type === 'select'}
					<select
						class="sfr-select"
						value={getValue(key, field)}
						{disabled}
						aria-label={field.label}
						onchange={(e) => handleInput(key, e)}
					>
						{#each field.options ?? [] as option}
							<option value={option}>{option}</option>
						{/each}
					</select>

				{:else if field.type === 'number'}
					<input
						type="number"
						class="sfr-input"
						value={getValue(key, field)}
						{disabled}
						aria-label={field.label}
						oninput={(e) => handleInput(key, e)}
					/>

				{:else}
					<!-- string (default) -->
					<input
						type="text"
						class="sfr-input"
						value={getValue(key, field)}
						{disabled}
						aria-label={field.label}
						oninput={(e) => handleInput(key, e)}
					/>
				{/if}
			</div>
		</div>
	{/each}
</div>

<style>
	/* ═══════════════════════════════════════════════════════════════
	   SCHEMA FORM RENDERER — sfr- prefix, Foundation tokens
	   Matches md-settings-card / md-setting-row conventions
	═══════════════════════════════════════════════════════════════ */

	.sfr-form {
		display: flex;
		flex-direction: column;
		gap: 0;
	}

	/* Row — mirrors .md-setting-row layout */
	.sfr-row {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 24px;
		padding: 14px 0;
		border-bottom: 1px solid var(--dbd2);
	}

	.sfr-row:last-child {
		border-bottom: none;
		padding-bottom: 0;
	}

	.sfr-row:first-child {
		padding-top: 0;
	}

	/* Label column */
	.sfr-label {
		display: flex;
		flex-direction: column;
		gap: 3px;
		min-width: 0;
		flex: 1;
	}

	.sfr-label-text {
		font-size: 13px;
		font-weight: 600;
		color: var(--dt);
		line-height: 1.4;
	}

	.sfr-required {
		color: var(--bos-status-error, #ef4444);
		margin-left: 2px;
	}

	.sfr-label-desc {
		font-size: 12px;
		color: var(--dt3);
		line-height: 1.5;
	}

	/* Control column */
	.sfr-control {
		flex-shrink: 0;
	}

	/* Text / number inputs */
	.sfr-input {
		height: 32px;
		min-width: 200px;
		padding: 0 10px;
		border-radius: 8px;
		border: 1px solid var(--dbd);
		background: var(--dbg2);
		color: var(--dt);
		font-size: 13px;
		outline: none;
		transition: border-color 0.15s;
	}

	.sfr-input:focus {
		border-color: var(--dt3);
	}

	.sfr-input:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	/* Select */
	.sfr-select {
		height: 32px;
		min-width: 200px;
		padding: 0 28px 0 10px;
		border-radius: 8px;
		border: 1px solid var(--dbd);
		background: var(--dbg2);
		color: var(--dt);
		font-size: 13px;
		outline: none;
		appearance: none;
		-webkit-appearance: none;
		background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 24 24' fill='none' stroke='%23888' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'%3E%3Cpolyline points='6 9 12 15 18 9'%3E%3C/polyline%3E%3C/svg%3E");
		background-repeat: no-repeat;
		background-position: right 9px center;
		cursor: pointer;
		transition: border-color 0.15s;
	}

	.sfr-select:focus {
		border-color: var(--dt3);
	}

	.sfr-select:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	/* Toggle switch — track */
	.sfr-toggle {
		position: relative;
		display: inline-flex;
		align-items: center;
		width: 40px;
		height: 22px;
		border-radius: 999px;
		border: none;
		background: var(--dbd);
		cursor: pointer;
		padding: 0;
		flex-shrink: 0;
		transition: background 0.2s;
	}

	.sfr-toggle--on {
		background: var(--bos-status-success, #22c55e);
	}

	.sfr-toggle:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	/* Toggle knob — sliding circle */
	.sfr-toggle-knob {
		position: absolute;
		top: 2px;
		left: 2px;
		width: 18px;
		height: 18px;
		border-radius: 50%;
		background: var(--dbg, #ffffff);
		transition: transform 0.2s;
		pointer-events: none;
	}

	.sfr-toggle--on .sfr-toggle-knob {
		transform: translateX(18px);
	}

	/* Dark mode overrides */
	:global(.dark) .sfr-input,
	:global(.dark) .sfr-select {
		background: var(--dbg2);
		border-color: var(--dbd);
		color: var(--dt);
	}

	:global(.dark) .sfr-toggle {
		background: var(--dbd);
	}

	:global(.dark) .sfr-toggle--on {
		background: var(--bos-status-success, #22c55e);
	}

	:global(.dark) .sfr-toggle-knob {
		background: var(--dbg);
	}
</style>
