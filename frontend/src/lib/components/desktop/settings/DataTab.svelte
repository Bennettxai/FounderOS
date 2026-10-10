<script lang="ts">
	import { windowStore, type DesktopConfig } from '$lib/stores/windowStore';

	let importError = $state('');
	let importSuccess = $state(false);
	let configFileInput: HTMLInputElement | undefined = $state(undefined);

	function exportConfig() {
		const config = windowStore.exportConfig();
		const blob = new Blob([JSON.stringify(config, null, 2)], { type: 'application/json' });
		const url = URL.createObjectURL(blob);
		const a = document.createElement('a');
		a.href = url;
		a.download = `businessos-desktop-config-${new Date().toISOString().split('T')[0]}.json`;
		document.body.appendChild(a);
		a.click();
		document.body.removeChild(a);
		URL.revokeObjectURL(url);
	}

	function exportSchema() {
		const schema = windowStore.getConfigSchema();
		const blob = new Blob([JSON.stringify(schema, null, 2)], { type: 'application/json' });
		const url = URL.createObjectURL(blob);
		const a = document.createElement('a');
		a.href = url;
		a.download = 'businessos-desktop-config-schema.json';
		document.body.appendChild(a);
		a.click();
		document.body.removeChild(a);
		URL.revokeObjectURL(url);
	}

	function triggerImport() {
		configFileInput?.click();
	}

	function handleConfigImport(event: Event) {
		const target = event.target as HTMLInputElement;
		const file = target.files?.[0];
		if (!file) return;

		importError = '';
		importSuccess = false;

		const reader = new FileReader();
		reader.onload = (e) => {
			try {
				const content = e.target?.result as string;
				const config = JSON.parse(content) as DesktopConfig;
				const result = windowStore.importConfig(config);

				if (result.success) {
					importSuccess = true;
					setTimeout(() => (importSuccess = false), 3000);
				} else {
					importError = result.error || 'Import failed';
					setTimeout(() => (importError = ''), 5000);
				}
			} catch (err) {
				importError = 'Invalid JSON file';
				setTimeout(() => (importError = ''), 5000);
			}
		};
		reader.readAsText(file);

		target.value = '';
	}
</script>

<!-- Data Export/Import -->
<input
	type="file"
	accept=".json"
	bind:this={configFileInput}
	onchange={handleConfigImport}
	class="hidden-file-input"
/>

{#if importSuccess}
	<div class="status-banner success">
		<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
			<path d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"/>
		</svg>
		Configuration imported successfully!
	</div>
{/if}

{#if importError}
	<div class="status-banner error">
		<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
			<circle cx="12" cy="12" r="10"/>
			<path d="M15 9l-6 6M9 9l6 6"/>
		</svg>
		{importError}
	</div>
{/if}

<div class="section">
	<label class="section-title">Export Configuration</label>
	<p class="section-desc">
		Download your desktop layout, dock items, and folder structure as a JSON file. Use this to backup or transfer your setup.
	</p>
	<div class="data-actions">
		<button class="data-btn primary" onclick={exportConfig}>
			<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4"/>
				<path d="M7 10l5 5 5-5"/>
				<path d="M12 15V3"/>
			</svg>
			Export Desktop Config
		</button>
		<button class="data-btn secondary" onclick={exportSchema}>
			<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/>
				<path d="M14 2v6h6M16 13H8M16 17H8M10 9H8"/>
			</svg>
			Download Schema
		</button>
	</div>
</div>

<div class="section">
	<label class="section-title">Import Configuration</label>
	<p class="section-desc">
		Load a previously exported configuration file. This will replace your current desktop layout.
	</p>
	<div
		class="import-area"
		onclick={triggerImport}
		onkeydown={(e) => e.key === 'Enter' && triggerImport()}
		tabindex="0"
		role="button"
		aria-label="Click to import configuration file"
	>
		<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
			<path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4"/>
			<path d="M17 8l-5-5-5 5"/>
			<path d="M12 3v12"/>
		</svg>
		<span class="import-text">Click to import configuration file</span>
		<span class="import-hint">Accepts .json files only</span>
	</div>
</div>

<div class="section">
	<label class="section-title">Configuration Schema</label>
	<p class="section-desc">
		The desktop configuration follows a JSON schema for validation. You can use the schema for programmatic config generation.
	</p>
	<div class="schema-preview">
		<pre><code>{JSON.stringify({
	version: "1.0.0",
	desktopIcons: [{ id: "...", module: "...", label: "...", x: 0, y: 0 }],
	dockPinnedItems: ["finder", "dashboard", "..."],
	folders: [{ id: "...", name: "...", color: "#...", iconIds: [] }]
}, null, 2)}</code></pre>
	</div>
</div>

<div class="section">
	<div class="shortcut-note">
		<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
			<circle cx="12" cy="12" r="10"/>
			<path d="M12 16v-4M12 8h.01"/>
		</svg>
		<span>
			Configuration files store icon positions, dock items, and folders. Window states and open apps are not included.
		</span>
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

	.section-desc {
		font-size: 12px;
		color: #666;
		margin: -8px 0 12px 0;
		line-height: 1.5;
	}

	.hidden-file-input {
		display: none;
	}

	.status-banner {
		display: flex;
		align-items: center;
		gap: 10px;
		padding: 12px 16px;
		border-radius: 8px;
		font-size: 13px;
		font-weight: 500;
		margin-bottom: 16px;
	}

	.status-banner svg {
		width: 18px;
		height: 18px;
		flex-shrink: 0;
	}

	.status-banner.success {
		background: #d4edda;
		color: #155724;
		border: 1px solid #c3e6cb;
	}

	.status-banner.error {
		background: #f8d7da;
		color: #721c24;
		border: 1px solid #f5c6cb;
	}

	.data-actions {
		display: flex;
		gap: 12px;
	}

	.data-btn {
		flex: 1;
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 8px;
		padding: 14px 20px;
		border-radius: 8px;
		font-size: 13px;
		font-weight: 500;
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.data-btn svg {
		width: 18px;
		height: 18px;
	}

	.data-btn.primary {
		background: #111;
		color: white;
		border: none;
	}

	.data-btn.primary:hover {
		background: #333;
	}

	.data-btn.secondary {
		background: white;
		color: #333;
		border: 1px solid #ddd;
	}

	.data-btn.secondary:hover {
		background: #f5f5f5;
		border-color: #ccc;
	}

	.import-area {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 8px;
		padding: 32px 20px;
		background: #f8f9fa;
		border: 2px dashed #ddd;
		border-radius: 12px;
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.import-area:hover {
		background: #f0f0f0;
		border-color: #bbb;
	}

	.import-area:focus {
		outline: none;
		border-color: #333;
		box-shadow: 0 0 0 3px rgba(0, 0, 0, 0.1);
	}

	.import-area svg {
		width: 32px;
		height: 32px;
		color: #888;
	}

	.import-text {
		font-size: 14px;
		font-weight: 500;
		color: #333;
	}

	.import-hint {
		font-size: 11px;
		color: #999;
	}

	.schema-preview {
		background: #1e1e1e;
		border-radius: 8px;
		padding: 16px;
		overflow-x: auto;
	}

	.schema-preview pre {
		margin: 0;
	}

	.schema-preview code {
		font-family: 'SF Mono', Monaco, 'Fira Code', monospace;
		font-size: 11px;
		line-height: 1.6;
		color: #9cdcfe;
	}

	.shortcut-note {
		display: flex;
		align-items: flex-start;
		gap: 10px;
		padding: 12px 14px;
		background: #f8f9fa;
		border: 1px solid #e9ecef;
		border-radius: 8px;
		font-size: 12px;
		color: #666;
		line-height: 1.5;
	}

	.shortcut-note svg {
		width: 16px;
		height: 16px;
		flex-shrink: 0;
		color: #6c757d;
		margin-top: 1px;
	}
</style>
