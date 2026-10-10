<script lang="ts">
	import { onMount, onDestroy, tick } from 'svelte';
	import type { TerminalProvider } from '$lib/stores/terminal/terminalTypes';
	import { PROVIDER_CONFIGS } from '$lib/stores/terminal/terminalTypes';
	import { chatModelStore } from '$lib/stores/chat/chatModelStore.svelte';
	import { MODE_COLORS, type OsaMode } from '$lib/stores/osa';
	import { getThemeColors } from './themes';
	import { getActiveWorkspaceHeaders, getApiBaseUrl, getCSRFToken, initCSRF } from '$lib/api/base';
	import { fetchSlashCommands, FALLBACK_COMMANDS } from '$lib/api/commands';

	// ─── Types ──────────────────────────────────────────────────────────────────

	interface ChatMessage {
		id: string;
		role: 'user' | 'assistant';
		content: string;
		timestamp: Date;
		isError?: boolean;
		thinkingContent?: string;
		thinkingCollapsed?: boolean;
		toolCalls?: ToolCallInfo[];
		artifacts?: ArtifactInfo[];
		usageData?: UsageData;
		signalMode?: string;
		signalConfidence?: number;
	}

	interface ToolCallInfo {
		name: string;
		id: string;
		status: 'running' | 'completed' | 'error';
		result?: string;
	}

	interface ArtifactInfo {
		title: string;
		type: string;
		content: string;
	}

	interface UsageData {
		input_tokens?: number;
		output_tokens?: number;
		thinking_tokens?: number;
		total_tokens?: number;
		duration_ms?: number;
		tps?: number;
		provider?: string;
		model?: string;
		estimated_cost?: number;
	}

	interface SlashCommand {
		id: string;
		label: string;
		description: string;
	}

	interface Props {
		paneId: string;
		provider: TerminalProvider;
		themeId?: string;
		visible?: boolean;
		focusMode?: string;
		onFocus?: (paneId: string) => void;
	}

	let { paneId, provider, themeId = 'dark', visible = true, focusMode, onFocus }: Props = $props();

	// ─── State ──────────────────────────────────────────────────────────────────

	let messages = $state<ChatMessage[]>([]);
	let inputValue = $state('');
	let isStreaming = $state(false);
	let streamingContent = $state('');
	let streamingThinking = $state('');
	let isThinking = $state(false);
	let streamingToolCalls = $state<ToolCallInfo[]>([]);
	let streamingArtifacts = $state<ArtifactInfo[]>([]);
	let messagesContainer: HTMLDivElement;
	let inputEl: HTMLTextAreaElement;
	let abortController: AbortController | null = null;
	let conversationId: string | null = null;

	// Slash command state
	let showSlashMenu = $state(false);
	let slashFilter = $state('');
	let slashSelectedIdx = $state(0);
	let activeCommand = $state<string | null>(null);

	// OSA mode state
	let osaActiveMode = $state<OsaMode>('ASSIST');

	// ─── Slash commands (loaded from shared API service) ────────────────────────

	/**
	 * Convert the canonical API SlashCommand shape to the local terminal shape.
	 * The terminal only needs id (prefixed with "/"), label, and description.
	 */
	function toTerminalCommand(c: { name: string; description: string }): SlashCommand {
		return { id: `/${c.name}`, label: `/${c.name}`, description: c.description };
	}

	/** Start with the fallback set so the dropdown works before the API responds. */
	let slashCommandList = $state<SlashCommand[]>(FALLBACK_COMMANDS.map(toTerminalCommand));

	const OSA_MODES: { mode: OsaMode; label: string; color: string }[] = [
		{ mode: 'BUILD', label: 'BUILD', color: MODE_COLORS.BUILD },
		{ mode: 'ASSIST', label: 'ASSIST', color: MODE_COLORS.ASSIST },
		{ mode: 'ANALYZE', label: 'ANALYZE', color: MODE_COLORS.ANALYZE },
		{ mode: 'EXECUTE', label: 'EXECUTE', color: MODE_COLORS.EXECUTE },
		{ mode: 'MAINTAIN', label: 'MAINTAIN', color: MODE_COLORS.MAINTAIN }
	];

	const OSA_MODE_TO_FOCUS: Record<OsaMode, string> = {
		BUILD: 'build',
		ASSIST: 'general',
		ANALYZE: 'analyze',
		EXECUTE: 'general',
		MAINTAIN: 'general'
	};

	// ─── Derived ────────────────────────────────────────────────────────────────

	const providerConfig = $derived(PROVIDER_CONFIGS.find(p => p.id === provider));
	const themeColors = $derived(getThemeColors(themeId));

	const currentModel = $derived.by(() => {
		if (provider === 'shell') return '';
		if (provider === 'osa') return 'claude-sonnet-4-20250514';
		// Use chatModelStore selected model if available
		if (chatModelStore.selectedModel) return chatModelStore.selectedModel;
		// Fallback defaults
		const defaults: Record<string, string> = {
			claude: 'claude-sonnet-4-20250514',
			codex: 'gpt-4o',
			ollama: 'llama3.2:latest'
		};
		return defaults[provider] ?? 'claude-sonnet-4-20250514';
	});

	const currentModelDisplay = $derived.by(() => {
		if (provider === 'osa') return 'OSA Agent';
		if (chatModelStore.currentModelName) return chatModelStore.currentModelName;
		return currentModel.split(':')[0];
	});

	const filteredSlashCommands = $derived(
		slashCommandList.filter(cmd =>
			cmd.id.toLowerCase().includes(slashFilter.toLowerCase())
		)
	);

	// ─── Send Message ───────────────────────────────────────────────────────────

	async function sendMessage() {
		const content = inputValue.trim();
		if (!content || isStreaming) return;

		// If a slash command was used, extract it
		let command: string | undefined;
		let actualMessage = content;
		if (activeCommand) {
			command = activeCommand.replace('/', '');
			activeCommand = null;
		} else if (content.startsWith('/')) {
			const spaceIdx = content.indexOf(' ');
			if (spaceIdx > 0) {
				command = content.substring(1, spaceIdx);
				actualMessage = content.substring(spaceIdx + 1).trim();
			}
		}

		const userMsg: ChatMessage = {
			id: crypto.randomUUID(),
			role: 'user',
			content,
			timestamp: new Date()
		};
		messages = [...messages, userMsg];
		inputValue = '';
		isStreaming = true;
		streamingContent = '';
		streamingThinking = '';
		isThinking = false;
		streamingToolCalls = [];
		streamingArtifacts = [];

		await tick();
		scrollToBottom();

		let reader: ReadableStreamDefaultReader<Uint8Array> | null = null;

		try {
			abortController = new AbortController();

			const effectiveFocusMode = provider === 'osa'
				? OSA_MODE_TO_FOCUS[osaActiveMode]
				: (focusMode ?? 'general');

			const body: Record<string, unknown> = {
				message: actualMessage,
				model: currentModel,
				conversation_id: conversationId,
				temperature: chatModelStore.aiTemperature,
				max_tokens: chatModelStore.aiMaxTokens,
				top_p: chatModelStore.aiTopP,
				structured_output: true,
				use_cot: chatModelStore.useCOT,
				focus_mode: effectiveFocusMode
			};

			if (command) body.command = command;

			// Ensure CSRF token is available (double-submit cookie pattern)
			let csrfToken = getCSRFToken();
			if (!csrfToken) {
				await initCSRF();
				csrfToken = getCSRFToken();
			}

			const fetchHeaders: Record<string, string> = {
				'Content-Type': 'application/json',
				...getActiveWorkspaceHeaders()
			};
			if (csrfToken) {
				fetchHeaders['X-CSRF-Token'] = csrfToken;
			}

			const response = await fetch(`${getApiBaseUrl()}/chat/message`, {
				method: 'POST',
				headers: fetchHeaders,
				credentials: 'include',
				body: JSON.stringify(body),
				signal: abortController.signal
			});

			const convId = response.headers.get('X-Conversation-Id');
			if (convId) conversationId = convId;

			if (!response.ok) {
				const errText = await response.text().catch(() => '');
				throw new Error(`HTTP ${response.status}${errText ? ': ' + errText.slice(0, 200) : ''}`);
			}

			reader = response.body?.getReader() ?? null;
			if (!reader) throw new Error('No response body');

			const decoder = new TextDecoder();
			let accumulated = '';
			let thinkingAccumulated = '';
			let usageData: UsageData | undefined;
			let signalMode: string | undefined;
			let signalConfidence: number | undefined;
			let buffer = '';

			while (true) {
				const { done, value } = await reader.read();
				if (done) break;

				buffer += decoder.decode(value, { stream: true });
				const lines = buffer.split('\n');
				buffer = lines.pop() ?? '';

				for (const rawLine of lines) {
					const line = rawLine.replace(/\r$/, '');
					if (!line.startsWith('data: ')) continue;
					const data = line.slice(6).trim();
					if (!data || data === '[DONE]') continue;

					try {
						const evt = JSON.parse(data);
						switch (evt.type) {
							case 'token':
								if (evt.content) {
									accumulated += evt.content;
									streamingContent = accumulated;
								}
								break;

							case 'thinking_start':
								isThinking = true;
								thinkingAccumulated = '';
								streamingThinking = '';
								break;

							case 'thinking_chunk':
								if (evt.content) {
									thinkingAccumulated += evt.content;
									streamingThinking = thinkingAccumulated;
								}
								break;

							case 'thinking_end':
								isThinking = false;
								break;

							case 'tool_call': {
								const tc: ToolCallInfo = {
									name: evt.toolName ?? evt.tool_name ?? 'unknown',
									id: evt.toolCallId ?? evt.tool_call_id ?? crypto.randomUUID(),
									status: 'running'
								};
								streamingToolCalls = [...streamingToolCalls, tc];
								break;
							}

							case 'tool_result': {
								const tcId = evt.toolCallId ?? evt.tool_call_id;
								streamingToolCalls = streamingToolCalls.map(tc =>
									tc.id === tcId
										? { ...tc, status: (evt.status ?? 'completed') as 'completed' | 'error', result: evt.result }
										: tc
								);
								break;
							}

							case 'signal_classified':
								signalMode = evt.mode ?? evt.data?.mode;
								signalConfidence = evt.confidence ?? evt.data?.confidence;
								// Auto-update OSA mode if provider is OSA
								if (provider === 'osa' && signalMode) {
									const modeUpper = signalMode.toUpperCase() as OsaMode;
									if (['BUILD', 'ASSIST', 'ANALYZE', 'EXECUTE', 'MAINTAIN'].includes(modeUpper)) {
										osaActiveMode = modeUpper;
									}
								}
								break;

							case 'artifact_start':
								streamingArtifacts = [...streamingArtifacts, {
									title: evt.title ?? 'Artifact',
									type: evt.artifact_type ?? evt.type ?? 'text',
									content: ''
								}];
								break;

							case 'artifact_complete':
								if (streamingArtifacts.length > 0) {
									const last = streamingArtifacts[streamingArtifacts.length - 1];
									streamingArtifacts = streamingArtifacts.map((a, i) =>
										i === streamingArtifacts.length - 1
											? { ...a, content: evt.content ?? a.content }
											: a
									);
								}
								break;

							case 'error':
								throw new Error(evt.content ?? evt.message ?? 'Stream error');

							case 'done': {
								// Parse usage from content
								const usageMatch = accumulated.match(/<!--USAGE:(.*?)-->/);
								if (usageMatch) {
									try {
										usageData = JSON.parse(usageMatch[1]) as UsageData;
										accumulated = accumulated.replace(/<!--USAGE:.*?-->/, '').trim();
									} catch { /* ignore parse error */ }
								}
								break;
							}
						}
					} catch (parseErr) {
						// Non-JSON SSE data — treat as raw token
						if (!data.startsWith('{')) {
							accumulated += data;
							streamingContent = accumulated;
						}
					}

					await tick();
					scrollToBottom();
				}
			}

			const assistantMsg: ChatMessage = {
				id: crypto.randomUUID(),
				role: 'assistant',
				content: accumulated || streamingContent,
				timestamp: new Date(),
				thinkingContent: thinkingAccumulated || undefined,
				thinkingCollapsed: true,
				toolCalls: streamingToolCalls.length > 0 ? [...streamingToolCalls] : undefined,
				artifacts: streamingArtifacts.length > 0 ? [...streamingArtifacts] : undefined,
				usageData,
				signalMode,
				signalConfidence
			};
			messages = [...messages, assistantMsg];

		} catch (err) {
			if ((err as Error).name !== 'AbortError') {
				const errorMsg: ChatMessage = {
					id: crypto.randomUUID(),
					role: 'assistant',
					content: `Error: ${(err as Error).message}`,
					timestamp: new Date(),
					isError: true
				};
				messages = [...messages, errorMsg];
			}
		} finally {
			reader?.cancel().catch(() => {});
			isStreaming = false;
			streamingContent = '';
			streamingThinking = '';
			isThinking = false;
			streamingToolCalls = [];
			streamingArtifacts = [];
			abortController = null;
		}
	}

	function cancelStream() {
		abortController?.abort();
	}

	// ─── Slash Command Handling ─────────────────────────────────────────────────

	function handleInput() {
		if (inputValue === '/') {
			showSlashMenu = true;
			slashFilter = '/';
			slashSelectedIdx = 0;
		} else if (inputValue.startsWith('/') && !inputValue.includes(' ')) {
			showSlashMenu = true;
			slashFilter = inputValue;
			slashSelectedIdx = 0;
		} else {
			showSlashMenu = false;
		}
	}

	function selectSlashCommand(cmd: SlashCommand) {
		activeCommand = cmd.id;
		inputValue = '';
		showSlashMenu = false;
		inputEl?.focus();
	}

	function clearActiveCommand() {
		activeCommand = null;
	}

	// ─── Keyboard ───────────────────────────────────────────────────────────────

	function handleKeydown(e: KeyboardEvent) {
		if (showSlashMenu) {
			if (e.key === 'ArrowDown') {
				e.preventDefault();
				slashSelectedIdx = Math.min(slashSelectedIdx + 1, filteredSlashCommands.length - 1);
			} else if (e.key === 'ArrowUp') {
				e.preventDefault();
				slashSelectedIdx = Math.max(slashSelectedIdx - 1, 0);
			} else if (e.key === 'Enter') {
				e.preventDefault();
				if (filteredSlashCommands[slashSelectedIdx]) {
					selectSlashCommand(filteredSlashCommands[slashSelectedIdx]);
				}
			} else if (e.key === 'Escape') {
				e.preventDefault();
				showSlashMenu = false;
			} else if (e.key === 'Tab') {
				e.preventDefault();
				if (filteredSlashCommands[slashSelectedIdx]) {
					selectSlashCommand(filteredSlashCommands[slashSelectedIdx]);
				}
			}
			return;
		}

		if (e.key === 'Enter' && !e.shiftKey) {
			e.preventDefault();
			sendMessage();
		}
	}

	// ─── Markdown Rendering ─────────────────────────────────────────────────────

	function renderMarkdown(text: string): string {
		let html = escapeHtml(text);

		// Fenced code blocks
		html = html.replace(/```(\w*)\n([\s\S]*?)```/g, (_match, lang, code) => {
			return `<div class="md-code-block"><div class="md-code-lang">${lang || 'code'}</div><pre><code>${code.trim()}</code></pre></div>`;
		});

		// Inline code
		html = html.replace(/`([^`]+)`/g, '<code class="md-inline-code">$1</code>');

		// Bold
		html = html.replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>');

		// Italic
		html = html.replace(/\*(.+?)\*/g, '<em>$1</em>');

		// Links
		html = html.replace(/\[([^\]]+)\]\(([^)]+)\)/g, '<a href="$2" target="_blank" rel="noopener" class="md-link">$1</a>');

		// Unordered lists
		html = html.replace(/^- (.+)$/gm, '<li class="md-li">$1</li>');
		html = html.replace(/(<li class="md-li">.*<\/li>\n?)+/g, (match) => `<ul class="md-ul">${match}</ul>`);

		// Ordered lists
		html = html.replace(/^\d+\. (.+)$/gm, '<li class="md-oli">$1</li>');
		html = html.replace(/(<li class="md-oli">.*<\/li>\n?)+/g, (match) => `<ol class="md-ol">${match}</ol>`);

		// Line breaks (preserve paragraph structure)
		html = html.replace(/\n\n/g, '<br/><br/>');
		html = html.replace(/\n/g, '<br/>');

		return html;
	}

	function escapeHtml(text: string): string {
		return text
			.replace(/&/g, '&amp;')
			.replace(/</g, '&lt;')
			.replace(/>/g, '&gt;');
	}

	// ─── Scroll ─────────────────────────────────────────────────────────────────

	function scrollToBottom() {
		if (messagesContainer) {
			messagesContainer.scrollTop = messagesContainer.scrollHeight;
		}
	}

	function clearChat() {
		messages = [];
		conversationId = null;
	}

	function toggleThinking(msgId: string) {
		messages = messages.map(m =>
			m.id === msgId ? { ...m, thinkingCollapsed: !m.thinkingCollapsed } : m
		);
	}

	export function focus() {
		inputEl?.focus();
	}

	onMount(() => {
		inputEl?.focus();
		// Load slash commands from the shared API service.
		// fetchSlashCommands caches the result so this is a no-op on re-mounts.
		fetchSlashCommands().then((commands) => {
			if (commands.length > 0) {
				slashCommandList = commands.map(toTerminalCommand);
			}
		});
	});

	onDestroy(() => {
		abortController?.abort();
		abortController = null;
	});
</script>

<div
	class="ai-chat"
	class:hidden={!visible}
	style="--bg: {themeColors.background}; --fg: {themeColors.foreground}; --accent: {providerConfig?.color ?? '#00ff00'};"
	onclick={() => onFocus?.(paneId)}
	role="region"
	aria-label="AI Chat - {providerConfig?.label ?? provider}"
>
	<!-- Header -->
	{#if messages.length > 0}
		<div class="chat-header">
			<span class="model-badge">{currentModelDisplay}</span>
			<button class="clear-btn" onclick={clearChat} aria-label="Clear chat">Clear</button>
		</div>
	{/if}

	<!-- Messages -->
	<div class="messages" bind:this={messagesContainer}>
		{#if messages.length === 0 && !isStreaming}
			<div class="empty-state">
				{#if provider === 'osa'}
					<div class="osa-terminal-ready">
						<span class="ready-mark">OSA</span>
						<div>
							<strong>Agent session ready</strong>
							<span>{currentModelDisplay}</span>
						</div>
					</div>
					<div class="osa-modes">
						{#each OSA_MODES as m (m.mode)}
							<button
								class="osa-mode-pill"
								class:active={osaActiveMode === m.mode}
								style="--mode-color: {m.color}"
								onclick={() => osaActiveMode = m.mode}
							>
								{m.label}
							</button>
						{/each}
					</div>
					<p class="empty-hint">Choose a mode, then enter a task below.</p>
				{:else if provider === 'claude'}
					<pre class="ascii-banner claude-banner">
<span class="ab-claude">  ██████╗██╗      █████╗ ██╗   ██╗██████╗ ███████╗</span>
<span class="ab-claude"> ██╔════╝██║     ██╔══██╗██║   ██║██╔══██╗██╔════╝</span>
<span class="ab-claude"> ██║     ██║     ███████║██║   ██║██║  ██║█████╗  </span>
<span class="ab-claude"> ██║     ██║     ██╔══██║██║   ██║██║  ██║██╔══╝  </span>
<span class="ab-claude"> ╚██████╗███████╗██║  ██║╚██████╔╝██████╔╝███████╗</span>
<span class="ab-claude">  ╚═════╝╚══════╝╚═╝  ╚═╝ ╚═════╝ ╚═════╝ ╚══════╝</span></pre>
					<span class="model-name">{currentModelDisplay}</span>
					<div class="suggested-prompts">
						<button class="prompt-chip" onclick={() => { inputValue = 'Explain this codebase'; sendMessage(); }}>Explain this codebase</button>
						<button class="prompt-chip" onclick={() => { inputValue = 'Help me debug'; sendMessage(); }}>Help me debug</button>
						<button class="prompt-chip" onclick={() => { inputValue = 'Write a test'; sendMessage(); }}>Write a test</button>
					</div>
				{:else if provider === 'codex'}
					<pre class="ascii-banner codex-banner">
<span class="ab-codex">  ██████╗ ██████╗ ██████╗ ███████╗██╗  ██╗</span>
<span class="ab-codex"> ██╔════╝██╔═══██╗██╔══██╗██╔════╝╚██╗██╔╝</span>
<span class="ab-codex"> ██║     ██║   ██║██║  ██║█████╗   ╚███╔╝ </span>
<span class="ab-codex"> ██║     ██║   ██║██║  ██║██╔══╝   ██╔██╗ </span>
<span class="ab-codex"> ╚██████╗╚██████╔╝██████╔╝███████╗██╔╝ ██╗</span>
<span class="ab-codex">  ╚═════╝ ╚═════╝ ╚═════╝ ╚══════╝╚═╝  ╚═╝</span></pre>
					<span class="model-name">{currentModelDisplay}</span>
					<p class="empty-hint">Code generation &amp; analysis</p>
				{:else if provider === 'ollama'}
					<pre class="ascii-banner ollama-banner">
<span class="ab-ollama">  ██████╗ ██╗     ██╗      █████╗ ███╗   ███╗ █████╗ </span>
<span class="ab-ollama"> ██╔═══██╗██║     ██║     ██╔══██╗████╗ ████║██╔══██╗</span>
<span class="ab-ollama"> ██║   ██║██║     ██║     ███████║██╔████╔██║███████║</span>
<span class="ab-ollama"> ██║   ██║██║     ██║     ██╔══██║██║╚██╔╝██║██╔══██║</span>
<span class="ab-ollama"> ╚██████╔╝███████╗███████╗██║  ██║██║ ╚═╝ ██║██║  ██║</span>
<span class="ab-ollama">  ╚═════╝ ╚══════╝╚══════╝╚═╝  ╚═╝╚═╝     ╚═╝╚═╝  ╚═╝</span></pre>
					<span class="model-name">{currentModelDisplay}</span>
					<p class="empty-hint">Running locally &mdash; private AI</p>
				{:else}
					<span class="provider-badge" style="color: {providerConfig?.color}">
						{providerConfig?.label ?? provider}
					</span>
					<p class="empty-hint">Type a message to start</p>
				{/if}
			</div>
		{/if}

		{#each messages as msg (msg.id)}
			<div class="message" class:user={msg.role === 'user'} class:assistant={msg.role === 'assistant'} class:error={msg.isError}>
				<div class="message-role">
					{msg.role === 'user' ? '>' : providerConfig?.label ?? ''}
				</div>
				<div class="message-body">
					<!-- Thinking block -->
					{#if msg.thinkingContent}
						<div class="thinking-block">
							<button class="thinking-toggle" onclick={() => toggleThinking(msg.id)}>
								{msg.thinkingCollapsed ? '>' : 'v'} Thinking
							</button>
							{#if !msg.thinkingCollapsed}
								<pre class="thinking-content">{msg.thinkingContent}</pre>
							{/if}
						</div>
					{/if}

					<!-- Tool calls -->
					{#if msg.toolCalls}
						{#each msg.toolCalls as tc (tc.id)}
							<div class="tool-call" class:completed={tc.status === 'completed'} class:error={tc.status === 'error'}>
								<span class="tool-icon">{tc.status === 'running' ? '...' : tc.status === 'completed' ? '+' : '!'}</span>
								<span class="tool-name">{tc.name}</span>
								{#if tc.result}
									<span class="tool-result">{tc.result.slice(0, 100)}</span>
								{/if}
							</div>
						{/each}
					{/if}

					<!-- Main content (markdown rendered for assistant) -->
					{#if msg.role === 'assistant' && !msg.isError}
						<div class="message-content md-rendered">{@html renderMarkdown(msg.content)}</div>
					{:else}
						<div class="message-content">{msg.content}</div>
					{/if}

					<!-- Artifacts -->
					{#if msg.artifacts}
						{#each msg.artifacts as artifact}
							<div class="artifact-card">
								<div class="artifact-header">
									<span class="artifact-type">{artifact.type}</span>
									<span class="artifact-title">{artifact.title}</span>
								</div>
								{#if artifact.content}
									<pre class="artifact-content">{artifact.content.slice(0, 500)}{artifact.content.length > 500 ? '...' : ''}</pre>
								{/if}
							</div>
						{/each}
					{/if}

					<!-- Signal & Usage metadata -->
					{#if msg.signalMode || msg.usageData}
						<div class="message-meta">
							{#if msg.signalMode}
								<span class="signal-badge">{msg.signalMode}{msg.signalConfidence ? ` ${Math.round(msg.signalConfidence * 100)}%` : ''}</span>
							{/if}
							{#if msg.usageData}
								<span class="usage-badge">
									{msg.usageData.total_tokens ?? 0} tok
									{#if msg.usageData.duration_ms}
										&middot; {(msg.usageData.duration_ms / 1000).toFixed(1)}s
									{/if}
									{#if msg.usageData.tps}
										&middot; {msg.usageData.tps.toFixed(0)} t/s
									{/if}
								</span>
							{/if}
						</div>
					{/if}
				</div>
			</div>
		{/each}

		<!-- Streaming state -->
		{#if isStreaming}
			<div class="message assistant streaming">
				<div class="message-role">{providerConfig?.label ?? ''}</div>
				<div class="message-body">
					{#if isThinking && streamingThinking}
						<div class="thinking-block active">
							<span class="thinking-label">Thinking...</span>
							<pre class="thinking-content">{streamingThinking}</pre>
						</div>
					{/if}

					{#if streamingToolCalls.length > 0}
						{#each streamingToolCalls as tc (tc.id)}
							<div class="tool-call" class:completed={tc.status === 'completed'} class:error={tc.status === 'error'}>
								<span class="tool-icon">{tc.status === 'running' ? '...' : tc.status === 'completed' ? '+' : '!'}</span>
								<span class="tool-name">{tc.name}</span>
							</div>
						{/each}
					{/if}

					{#if streamingContent}
						<div class="message-content md-rendered">{@html renderMarkdown(streamingContent)}</div>
					{:else if !isThinking}
						<span class="cursor-blink">|</span>
					{/if}
				</div>
			</div>
		{/if}
	</div>

	<!-- Input Area -->
	<div class="input-area">
		<!-- Active command tag -->
		{#if activeCommand}
			<div class="command-tag">
				<span>{activeCommand}</span>
				<button class="command-tag-x" onclick={clearActiveCommand}>&times;</button>
			</div>
		{/if}

		<!-- Slash command dropdown -->
		{#if showSlashMenu && filteredSlashCommands.length > 0}
			<div class="slash-menu">
				{#each filteredSlashCommands as cmd, idx (cmd.id)}
					<button
						class="slash-item"
						class:selected={idx === slashSelectedIdx}
						onclick={() => selectSlashCommand(cmd)}
						onmouseenter={() => slashSelectedIdx = idx}
					>
						<span class="slash-cmd">{cmd.label}</span>
						<span class="slash-desc">{cmd.description}</span>
					</button>
				{/each}
			</div>
		{/if}

		<div class="input-row">
			{#if isStreaming}
				<button class="cancel-btn" onclick={cancelStream}>Stop</button>
			{/if}
			<textarea
				bind:this={inputEl}
				bind:value={inputValue}
				oninput={handleInput}
				onkeydown={handleKeydown}
				placeholder={activeCommand ? `${activeCommand} > message...` : `${providerConfig?.label ?? provider} > type a message...`}
				rows="1"
				disabled={isStreaming}
			></textarea>
			<button
				class="send-btn"
				onclick={sendMessage}
				disabled={!inputValue.trim() || isStreaming}
				aria-label="Send message"
			>
				&crarr;
			</button>
		</div>
	</div>
</div>

<style>
	.ai-chat {
		display: flex;
		flex-direction: column;
		width: 100%;
		height: 100%;
		background: var(--bg, #1a1a1a);
		color: var(--fg, #00ff00);
		font-family: 'SF Mono', 'Monaco', 'Fira Code', monospace;
		font-size: 13px;
	}

	.ai-chat.hidden { display: none; }

	/* ─── Header ────────────────────────────────────────────────────────── */
	.chat-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 4px 8px;
		border-bottom: 1px solid #222;
	}

	.model-badge {
		font-size: 10px;
		color: #666;
		font-weight: 500;
	}

	.clear-btn {
		background: transparent;
		border: 1px solid #333;
		color: #666;
		font-family: inherit;
		font-size: 10px;
		padding: 2px 8px;
		border-radius: 3px;
		cursor: pointer;
	}

	.clear-btn:hover { color: #ccc; border-color: #555; }

	/* ─── Messages ──────────────────────────────────────────────────────── */
	.messages {
		flex: 1;
		overflow-y: auto;
		padding: 12px;
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.messages::-webkit-scrollbar { width: 6px; }
	.messages::-webkit-scrollbar-track { background: transparent; }
	.messages::-webkit-scrollbar-thumb { background: #333; border-radius: 3px; }

	/* ─── Empty State ───────────────────────────────────────────────────── */
	.empty-state {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		flex: 1;
		gap: 12px;
		opacity: 1;
	}

	.osa-terminal-ready {
		display: flex;
		align-items: center;
		gap: 12px;
		min-width: 280px;
		padding: 14px 16px;
		border: 1px solid #2d3036;
		border-radius: 8px;
		background: #151619;
		box-shadow: 0 12px 32px rgba(0, 0, 0, 0.18);
	}

	.osa-terminal-ready .ready-mark {
		display: grid;
		place-items: center;
		width: 38px;
		height: 38px;
		border-radius: 7px;
		background: #8b5cf6;
		color: #fff;
		font-size: 11px;
		font-weight: 800;
		letter-spacing: 0.08em;
	}

	.osa-terminal-ready div {
		display: flex;
		flex-direction: column;
		gap: 3px;
	}

	.osa-terminal-ready strong {
		color: #f3f4f6;
		font-size: 13px;
		font-weight: 600;
	}

	.osa-terminal-ready div span {
		color: #737985;
		font-size: 10px;
	}

	.provider-badge { font-size: 18px; font-weight: 700; }
	.model-name { font-size: 11px; color: #555; }
	.empty-hint { margin: 0; font-size: 12px; color: #555; }

	/* ─── OSA Mode Pills ────────────────────────────────────────────────── */
	.osa-modes {
		display: flex;
		gap: 4px;
		flex-wrap: wrap;
		justify-content: center;
	}

	.osa-mode-pill {
		padding: 3px 10px;
		border-radius: 10px;
		border: 1px solid #333;
		background: transparent;
		color: var(--mode-color, #888);
		font-family: inherit;
		font-size: 9px;
		font-weight: 600;
		cursor: pointer;
		text-transform: uppercase;
		letter-spacing: 0.5px;
		transition: all 0.15s ease;
	}

	.osa-mode-pill:hover {
		border-color: var(--mode-color);
		background: color-mix(in srgb, var(--mode-color) 15%, transparent);
	}

	.osa-mode-pill.active {
		background: var(--mode-color);
		color: #fff;
		border-color: var(--mode-color);
	}

	/* ─── Suggested Prompts ─────────────────────────────────────────────── */
	.suggested-prompts {
		display: flex;
		gap: 6px;
		flex-wrap: wrap;
		justify-content: center;
	}

	.prompt-chip {
		padding: 4px 10px;
		border-radius: 8px;
		border: 1px solid #333;
		background: transparent;
		color: #888;
		font-family: inherit;
		font-size: 10px;
		cursor: pointer;
		transition: all 0.12s ease;
	}

	.prompt-chip:hover {
		border-color: var(--accent);
		color: var(--accent);
	}

	/* ─── Message ───────────────────────────────────────────────────────── */
	.message {
		display: flex;
		gap: 8px;
		line-height: 1.5;
	}

	.message-role {
		color: var(--accent, #00ff00);
		font-weight: 600;
		white-space: nowrap;
		min-width: 60px;
	}

	.message.user .message-role { color: #888; }
	.message.error .message-content { color: #ff5555; }

	.message-body {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: 4px;
	}

	.message-content {
		white-space: pre-wrap;
		word-break: break-word;
	}

	.streaming .message-content { opacity: 0.9; }

	/* ─── Markdown Rendering ────────────────────────────────────────────── */
	:global(.md-rendered .md-code-block) {
		background: #111;
		border: 1px solid #2a2a2a;
		border-radius: 4px;
		margin: 6px 0;
		overflow-x: auto;
	}

	:global(.md-rendered .md-code-lang) {
		font-size: 9px;
		color: #555;
		padding: 2px 8px;
		border-bottom: 1px solid #222;
		text-transform: uppercase;
	}

	:global(.md-rendered .md-code-block pre) {
		margin: 0;
		padding: 8px;
		font-size: 12px;
	}

	:global(.md-rendered .md-code-block code) {
		color: #e0e0e0;
	}

	:global(.md-rendered .md-inline-code) {
		background: #222;
		padding: 1px 4px;
		border-radius: 3px;
		font-size: 12px;
		color: #e8a0bf;
	}

	:global(.md-rendered strong) { color: #fff; }

	:global(.md-rendered .md-link) {
		color: var(--accent, #00ff00);
		text-decoration: underline;
		text-underline-offset: 2px;
	}

	:global(.md-rendered .md-ul),
	:global(.md-rendered .md-ol) {
		padding-left: 16px;
		margin: 4px 0;
	}

	:global(.md-rendered .md-li),
	:global(.md-rendered .md-oli) {
		margin: 2px 0;
		color: #ccc;
	}

	/* ─── Thinking Block ────────────────────────────────────────────────── */
	.thinking-block {
		border: 1px solid #2a2a2a;
		border-radius: 4px;
		background: #111;
		padding: 4px 8px;
		margin-bottom: 4px;
	}

	.thinking-block.active {
		border-color: #8b5cf6;
		animation: pulse-border 2s infinite;
	}

	@keyframes pulse-border {
		0%, 100% { border-color: #2a2a2a; }
		50% { border-color: #8b5cf6; }
	}

	.thinking-toggle {
		background: none;
		border: none;
		color: #8b5cf6;
		font-family: inherit;
		font-size: 10px;
		cursor: pointer;
		padding: 0;
		font-weight: 600;
	}

	.thinking-label {
		color: #8b5cf6;
		font-size: 10px;
		font-weight: 600;
	}

	.thinking-content {
		font-size: 11px;
		color: #777;
		margin: 4px 0 0;
		white-space: pre-wrap;
		word-break: break-word;
		max-height: 200px;
		overflow-y: auto;
	}

	/* ─── Tool Calls ────────────────────────────────────────────────────── */
	.tool-call {
		display: flex;
		align-items: center;
		gap: 6px;
		padding: 3px 8px;
		background: #111;
		border: 1px solid #2a2a2a;
		border-radius: 4px;
		font-size: 11px;
	}

	.tool-call.completed { border-color: #22c55e33; }
	.tool-call.error { border-color: #ff555533; }

	.tool-icon { font-weight: 700; color: #888; font-size: 10px; }
	.tool-call.completed .tool-icon { color: #22c55e; }
	.tool-call.error .tool-icon { color: #ff5555; }

	.tool-name { color: #aaa; font-weight: 500; }
	.tool-result { color: #666; font-size: 10px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; max-width: 200px; }

	/* ─── Artifacts ─────────────────────────────────────────────────────── */
	.artifact-card {
		border: 1px solid #333;
		border-radius: 4px;
		margin-top: 4px;
		overflow: hidden;
	}

	.artifact-header {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 4px 8px;
		background: #1a1a1a;
		border-bottom: 1px solid #2a2a2a;
	}

	.artifact-type {
		font-size: 9px;
		color: var(--accent);
		font-weight: 700;
		text-transform: uppercase;
	}

	.artifact-title { font-size: 11px; color: #ccc; }

	.artifact-content {
		padding: 8px;
		font-size: 11px;
		color: #999;
		margin: 0;
		white-space: pre-wrap;
		max-height: 200px;
		overflow-y: auto;
	}

	/* ─── Message Metadata ──────────────────────────────────────────────── */
	.message-meta {
		display: flex;
		gap: 8px;
		margin-top: 4px;
	}

	.signal-badge {
		font-size: 9px;
		color: #8b5cf6;
		background: #8b5cf620;
		padding: 1px 6px;
		border-radius: 4px;
		text-transform: uppercase;
		font-weight: 600;
	}

	.usage-badge {
		font-size: 9px;
		color: #555;
	}

	.cursor-blink {
		animation: blink 1s step-end infinite;
		color: var(--accent);
	}

	@keyframes blink { 50% { opacity: 0; } }

	/* ─── Input Area ────────────────────────────────────────────────────── */
	.input-area {
		position: relative;
		padding: 0;
		border-top: 1px solid #2a2a2a;
		background: rgba(0, 0, 0, 0.2);
	}

	.command-tag {
		display: inline-flex;
		align-items: center;
		gap: 4px;
		margin: 6px 12px 0;
		padding: 2px 8px;
		background: var(--accent, #8b5cf6);
		color: #000;
		border-radius: 4px;
		font-size: 10px;
		font-weight: 700;
	}

	.command-tag-x {
		background: none;
		border: none;
		color: #000;
		cursor: pointer;
		font-size: 12px;
		padding: 0;
		line-height: 1;
		opacity: 0.6;
	}

	.command-tag-x:hover { opacity: 1; }

	.input-row {
		display: flex;
		align-items: flex-end;
		gap: 4px;
		padding: 8px 12px;
	}

	.input-row textarea {
		flex: 1;
		background: transparent;
		border: 1px solid #333;
		border-radius: 4px;
		color: inherit;
		font-family: inherit;
		font-size: inherit;
		padding: 6px 8px;
		resize: none;
		outline: none;
		min-height: 28px;
		max-height: 120px;
	}

	.input-row textarea:focus { border-color: var(--accent, #00ff00); }
	.input-row textarea::placeholder { color: #555; }

	.send-btn {
		background: var(--accent, #00ff00);
		color: #000;
		border: none;
		border-radius: 4px;
		padding: 6px 10px;
		cursor: pointer;
		font-family: inherit;
		font-size: 14px;
		font-weight: 600;
		line-height: 1;
	}

	.send-btn:disabled { opacity: 0.3; cursor: default; }

	.cancel-btn {
		background: #ff5555;
		color: #fff;
		border: none;
		border-radius: 4px;
		padding: 4px 8px;
		cursor: pointer;
		font-family: inherit;
		font-size: 11px;
	}

	/* ─── Slash Command Menu ────────────────────────────────────────────── */
	.slash-menu {
		position: absolute;
		bottom: 100%;
		left: 12px;
		right: 12px;
		background: #111;
		border: 1px solid #333;
		border-radius: 6px;
		padding: 4px;
		max-height: 240px;
		overflow-y: auto;
		z-index: 50;
		box-shadow: 0 -4px 16px rgba(0,0,0,0.4);
	}

	.slash-item {
		display: flex;
		align-items: center;
		gap: 8px;
		width: 100%;
		padding: 6px 8px;
		border: none;
		background: transparent;
		color: #ccc;
		font-family: inherit;
		font-size: 12px;
		cursor: pointer;
		border-radius: 4px;
		text-align: left;
	}

	.slash-item.selected,
	.slash-item:hover {
		background: #222;
	}

	.slash-cmd {
		color: var(--accent, #00ff00);
		font-weight: 600;
		min-width: 80px;
	}

	.slash-desc {
		color: #666;
		font-size: 11px;
	}

	/* ─── ASCII Art Banners ────────────────────────────────────────────── */
	.ascii-banner {
		font-family: 'SF Mono', 'Monaco', 'Fira Code', monospace;
		font-size: 8px;
		line-height: 1.1;
		margin: 0;
		text-align: center;
		white-space: pre;
		user-select: none;
	}

	.osa-banner .ab-l1 { color: #00ffff; }
	.osa-banner .ab-l2 { color: #00ff00; }
	.osa-banner .ab-l3 { color: #3b82f6; }
	.osa-banner .ab-l4 { color: #d946ef; }
	.osa-banner .ab-l5 { color: #eab308; }

	.ab-claude { color: #d97706; }
	.ab-codex { color: #10b981; }
	.ab-ollama { color: #3b82f6; }
</style>
