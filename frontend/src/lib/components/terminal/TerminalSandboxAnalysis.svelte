<script lang="ts">
	import { tick } from 'svelte';
	import { getActiveWorkspaceHeaders, getApiBaseUrl, getCSRFToken, initCSRF } from '$lib/api/base';

	interface Props {
		sessionId: string;
		visible: boolean;
		onMerge: () => void;
		onStay: () => void;
	}

	let { sessionId, visible, onMerge, onStay }: Props = $props();

	let analysisState = $state<'loading' | 'analyzing' | 'ready'>('loading');
	let files = $state<string[]>([]);
	let aiAnalysis = $state('');
	let fileCount = $state(0);

	$effect(() => {
		if (visible && sessionId) {
			loadChanges();
		}
	});

	async function loadChanges() {
		analysisState = 'loading';
		aiAnalysis = '';
		files = [];
		fileCount = 0;
		try {
			const res = await fetch(`${getApiBaseUrl()}/terminal/sessions/${sessionId}/changes`, {
				credentials: 'include',
				headers: getActiveWorkspaceHeaders()
			});
			if (!res.ok) throw new Error('Failed to load changes');
			const data = await res.json();
			files = data.files ?? [];
			fileCount = data.file_count ?? 0;

			if (fileCount === 0) {
				aiAnalysis = 'No changes detected in sandbox. Safe to merge.';
				analysisState = 'ready';
				return;
			}

			// Request AI analysis
			analysisState = 'analyzing';
			await analyzeChanges(data.summary ?? '');
		} catch (err) {
			aiAnalysis = `Error loading changes: ${(err as Error).message}`;
			analysisState = 'ready';
		}
	}

	async function analyzeChanges(summary: string) {
		try {
			let csrfToken = getCSRFToken();
			if (!csrfToken) {
				await initCSRF();
				csrfToken = getCSRFToken();
			}

			const headers: Record<string, string> = {
				'Content-Type': 'application/json',
				...getActiveWorkspaceHeaders()
			};
			if (csrfToken) headers['X-CSRF-Token'] = csrfToken;

			const res = await fetch(`${getApiBaseUrl()}/chat/message`, {
				method: 'POST',
				headers,
				credentials: 'include',
				body: JSON.stringify({
					message: `Analyze these code changes for safety, security vulnerabilities, breaking changes, and production readiness. Be concise.\n\nChanges:\n${summary}\n\nFiles: ${files.join(', ')}`,
					model: 'claude-sonnet-4-20250514',
					structured_output: true,
					max_tokens: 1024,
				}),
			});

			if (!res.ok) throw new Error('AI analysis failed');

			const reader = res.body?.getReader();
			if (!reader) throw new Error('No response body');

			const decoder = new TextDecoder();
			let buffer = '';

			while (true) {
				const { done, value } = await reader.read();
				if (done) break;

				buffer += decoder.decode(value, { stream: true });
				const lines = buffer.split('\n');
				buffer = lines.pop() ?? '';

				for (const line of lines) {
					if (!line.startsWith('data: ')) continue;
					const data = line.slice(6);
					if (data === '[DONE]') continue;
					try {
						const evt = JSON.parse(data);
						if (evt.type === 'token' && evt.content) {
							aiAnalysis += evt.content;
							await tick();
						}
					} catch { /* skip non-JSON */ }
				}
			}
		} catch (err) {
			aiAnalysis = `Analysis unavailable: ${(err as Error).message}`;
		}
		analysisState = 'ready';
	}
</script>

{#if visible}
	<div class="sandbox-overlay" role="dialog" aria-label="Sandbox Analysis">
		<div class="sandbox-modal">
			<div class="modal-header">
				<span class="modal-title">Sandbox Analysis</span>
				<button class="modal-close" onclick={onStay} aria-label="Close">&times;</button>
			</div>

			<div class="modal-body">
				{#if analysisState === 'loading'}
					<div class="loading">Loading sandbox changes...</div>
				{:else}
					{#if files.length > 0}
						<div class="files-section">
							<span class="files-header">{fileCount} file{fileCount !== 1 ? 's' : ''} changed</span>
							<div class="file-list">
								{#each files as file}
									<div class="file-item">{file}</div>
								{/each}
							</div>
						</div>
					{/if}

					<div class="analysis-section">
						<span class="analysis-header">AI Analysis</span>
						<div class="analysis-content">
							{#if analysisState === 'analyzing' && !aiAnalysis}
								<span class="analyzing-dot">Analyzing</span>
							{/if}
							{#if aiAnalysis}
								<pre class="analysis-text">{aiAnalysis}</pre>
							{/if}
						</div>
					</div>
				{/if}
			</div>

			<div class="modal-footer">
				<button class="action-btn stay" onclick={onStay}>
					Stay in Sandbox
				</button>
				<button class="action-btn merge" onclick={onMerge} disabled={analysisState !== 'ready'}>
					Merge to Production
				</button>
			</div>
		</div>
	</div>
{/if}

<style>
	.sandbox-overlay {
		position: fixed;
		inset: 0;
		background: rgba(0, 0, 0, 0.7);
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 2000;
	}

	.sandbox-modal {
		background: #111;
		border: 1px solid #333;
		border-radius: 8px;
		width: 90%;
		max-width: 520px;
		max-height: 80vh;
		display: flex;
		flex-direction: column;
		overflow: hidden;
	}

	.modal-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 10px 14px;
		border-bottom: 1px solid #222;
	}

	.modal-title {
		font-family: 'SF Mono', monospace;
		font-size: 13px;
		font-weight: 600;
		color: #eab308;
	}

	.modal-close {
		background: none;
		border: none;
		color: #555;
		font-size: 18px;
		cursor: pointer;
		padding: 0 4px;
	}

	.modal-close:hover { color: #ccc; }

	.modal-body {
		flex: 1;
		overflow-y: auto;
		padding: 12px 14px;
		display: flex;
		flex-direction: column;
		gap: 12px;
	}

	.loading {
		color: #888;
		font-family: 'SF Mono', monospace;
		font-size: 12px;
		text-align: center;
		padding: 20px;
	}

	.files-section, .analysis-section {
		display: flex;
		flex-direction: column;
		gap: 6px;
	}

	.files-header, .analysis-header {
		font-family: 'SF Mono', monospace;
		font-size: 10px;
		font-weight: 600;
		color: #666;
		text-transform: uppercase;
		letter-spacing: 0.5px;
	}

	.file-list {
		display: flex;
		flex-direction: column;
		gap: 2px;
		max-height: 120px;
		overflow-y: auto;
	}

	.file-item {
		font-family: 'SF Mono', monospace;
		font-size: 11px;
		color: #aaa;
		padding: 2px 6px;
		background: #1a1a1a;
		border-radius: 3px;
	}

	.analysis-content {
		background: #0a0a0a;
		border: 1px solid #222;
		border-radius: 4px;
		padding: 10px;
		min-height: 80px;
		max-height: 240px;
		overflow-y: auto;
	}

	.analyzing-dot {
		color: #eab308;
		font-family: 'SF Mono', monospace;
		font-size: 11px;
		animation: pulse-text 1.5s ease-in-out infinite;
	}

	@keyframes pulse-text {
		0%, 100% { opacity: 0.4; }
		50% { opacity: 1; }
	}

	.analysis-text {
		font-family: 'SF Mono', monospace;
		font-size: 11px;
		color: #ccc;
		margin: 0;
		white-space: pre-wrap;
		word-break: break-word;
		line-height: 1.5;
	}

	.modal-footer {
		display: flex;
		justify-content: flex-end;
		gap: 8px;
		padding: 10px 14px;
		border-top: 1px solid #222;
	}

	.action-btn {
		padding: 6px 14px;
		border-radius: 4px;
		border: 1px solid #333;
		font-family: 'SF Mono', monospace;
		font-size: 11px;
		font-weight: 600;
		cursor: pointer;
	}

	.action-btn.stay {
		background: transparent;
		color: #eab308;
		border-color: #eab30855;
	}

	.action-btn.stay:hover { background: #eab30815; }

	.action-btn.merge {
		background: #22c55e;
		color: #000;
		border-color: #22c55e;
	}

	.action-btn.merge:hover { background: #16a34a; }
	.action-btn.merge:disabled { opacity: 0.4; cursor: default; }
</style>
