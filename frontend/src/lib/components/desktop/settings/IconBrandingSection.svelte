<script lang="ts">
	import { desktopSettings } from '$lib/stores/desktopStore';

	interface Props {
		companyNameInput: string;
		onSave: (name: string) => void;
	}

	let { companyNameInput = $bindable(), onSave }: Props = $props();
</script>

<!-- Company Branding -->
<div class="section">
	<label class="section-title">Company Branding</label>
	<div class="branding-row">
		<div class="branding-preview">
			<span class="preview-name">{companyNameInput || 'BUSINESS'}</span>
			<span class="preview-os">OS</span>
		</div>
		<div class="branding-input-row">
			<input
				type="text"
				placeholder="Company Name"
				class="company-name-input"
				bind:value={companyNameInput}
				onkeydown={(e) => e.key === 'Enter' && onSave(companyNameInput)}
				maxlength="16"
			/>
			<button
				class="save-name-btn"
				onclick={() => onSave(companyNameInput)}
			>
				Save
			</button>
		</div>
		<p class="branding-hint">This name appears on the loading screen</p>
	</div>
</div>

<style>
	.section {
		margin-bottom: 24px;
	}

	.section-title {
		font-size: 13px;
		font-weight: 600;
		color: #333;
		display: block;
		margin-bottom: 12px;
	}

	.branding-row {
		background: white;
		border-radius: 8px;
		padding: 16px;
		display: flex;
		flex-direction: column;
		gap: 12px;
	}

	.branding-preview {
		display: flex;
		align-items: baseline;
		justify-content: center;
		gap: 2px;
		padding: 16px;
		background: #fafafa;
		border-radius: 8px;
		font-family: 'SF Mono', 'Monaco', 'Fira Code', monospace;
	}

	.preview-name {
		font-size: 24px;
		font-weight: 800;
		color: #111;
		letter-spacing: 3px;
		text-transform: uppercase;
	}

	.preview-os {
		font-size: 20px;
		font-weight: 400;
		color: #111;
		opacity: 0.4;
	}

	.branding-input-row {
		display: flex;
		gap: 8px;
	}

	.company-name-input {
		flex: 1;
		padding: 10px 12px;
		border: 1px solid #ddd;
		border-radius: 6px;
		font-size: 13px;
		font-weight: 500;
		text-transform: uppercase;
		letter-spacing: 1px;
		outline: none;
		transition: border-color 0.15s ease;
	}

	.company-name-input:focus {
		border-color: #333;
	}

	.company-name-input::placeholder {
		color: #999;
		text-transform: none;
		letter-spacing: 0;
	}

	.save-name-btn {
		padding: 10px 18px;
		background: #333;
		color: white;
		border: none;
		border-radius: 6px;
		font-size: 13px;
		font-weight: 500;
		cursor: pointer;
		transition: background 0.15s ease;
	}

	.save-name-btn:hover {
		background: #555;
	}

	.branding-hint {
		font-size: 11px;
		color: #999;
		margin: 0;
		text-align: center;
	}
</style>
