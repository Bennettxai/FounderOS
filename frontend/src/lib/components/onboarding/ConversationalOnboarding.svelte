<!--
  ConversationalOnboarding.svelte
  Orchestrator: manages session state, API calls, and phase transitions.
  UI is delegated to OnboardingConversation and OnboardingIntegrations.

  Supports two modes:
  1. API Mode (default): Uses backend API for session management
  2. Local Mode: Self-contained with mock responses (for testing/fallback)
-->
<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { fly } from 'svelte/transition';
	import PurpleOrb from './PurpleOrb.svelte';
	import OnboardingConversation from './OnboardingConversation.svelte';
	import OnboardingIntegrations from './OnboardingIntegrations.svelte';
	import { onboardingApi } from '$lib/api/onboarding';
	import {
		questions,
		integrationDefs,
		computeRecommendedIntegrations,
		generateId,
		getApiBase,
		type OnboardingPhase,
		type QuestionType,
		type ExtractedData,
		type HistoryEntry
	} from './onboardingTypes.ts';

	interface Props {
		sessionId?: string;
		onComplete?: (data: ExtractedData) => void;
		useApi?: boolean;
		class?: string;
	}

	let {
		sessionId: initialSessionId,
		onComplete,
		useApi = true,
		class: className = ''
	}: Props = $props();

	// ─── Session state ────────────────────────────────────────────────────────

	let sessionId = $state<string | null>(initialSessionId || null);
	let apiError = $state<string | null>(null);
	let sessionExpired = $state(false);
	let lastErrorAction = $state<(() => void) | null>(null);

	// ─── History (go back) ────────────────────────────────────────────────────

	let questionHistory = $state<HistoryEntry[]>([]);

	// ─── Core state ───────────────────────────────────────────────────────────

	let phase = $state<OnboardingPhase>(useApi ? 'loading' : 'intro');
	let currentQuestion = $state<QuestionType>('company_name');
	let isAgentTyping = $state(false);
	let extractedData = $state<ExtractedData>({});
	let currentAgentMessage = $state('');

	let canGoBack = $derived(
		questionHistory.length > 0 && phase === 'conversation' && !isAgentTyping
	);

	// ─── Integration state ────────────────────────────────────────────────────

	let selectedIntegrations = $state<string[]>([]);
	let integrationStatuses = $state<Record<string, 'disconnected' | 'connecting' | 'connected' | 'error'>>({});
	let recommendedIntegrations = $state<string[]>([]);

	let allRecommendedConnected = $derived(
		recommendedIntegrations.length > 0 &&
		recommendedIntegrations.every(id => integrationStatuses[id] === 'connected')
	);

	// ─── Input state ──────────────────────────────────────────────────────────

	let inputValue = $state('');
	let isRecording = $state(false);
	let inputRef = $state<HTMLInputElement | null>(null);

	// ─── Resume state ─────────────────────────────────────────────────────────

	let isResuming = $state(false);
	let resumeMessage = $state('');

	// ─── OAuth error state ────────────────────────────────────────────────────

	let oauthError = $state<string | null>(null);
	let failedIntegrationId = $state<string | null>(null);

	// ─── Derived ──────────────────────────────────────────────────────────────

	let currentQuestionConfig = $derived(questions[currentQuestion] || questions.company_name);

	let currentStep = $derived(
		phase === 'loading' ? 0 :
		phase === 'intro' ? 1 :
		phase === 'conversation' ? 2 :
		3
	);

	const introLines = [
		"Hi! I'm here to help set up your workspace.",
		"What's your company called?"
	];

	// ─── Auto-focus input ─────────────────────────────────────────────────────

	$effect(() => {
		if (phase === 'conversation' && currentQuestionConfig.inputType === 'chat' && inputRef && !isAgentTyping) {
			setTimeout(() => inputRef?.focus(), 100);
		}
	});

	// ─── Session initialization ───────────────────────────────────────────────

	onMount(async () => {
		if (!useApi) return;

		try {
			const status = await onboardingApi.checkStatus();

			if (!status.needs_onboarding) {
				goto('/window');
				return;
			}

			const resumeResult = await onboardingApi.getResumeableSession();

			if (resumeResult.has_session && resumeResult.session) {
				sessionId = resumeResult.session.id;
				currentQuestion = (resumeResult.session.current_step as QuestionType) || 'company_name';

				if (resumeResult.session.extracted_data) {
					const apiData = resumeResult.session.extracted_data;
					extractedData = {
						workspaceName: apiData.workspace_name,
						businessType: apiData.business_type,
						teamSize: apiData.team_size,
						role: apiData.role,
						challenge: apiData.challenge,
						integrations: apiData.integrations
					};
				}

				if (currentQuestion !== 'company_name') {
					isResuming = true;
					const parts = ['Welcome back!'];
					if (extractedData.workspaceName) {
						parts.push(`Setting up ${extractedData.workspaceName}`);
						if (extractedData.businessType) {
							parts[1] += ` - ${extractedData.businessType}`;
						}
						parts[1] += '.';
					}
					resumeMessage = parts.join(' ');
				}

				currentAgentMessage = questions[currentQuestion]?.message || '';
				phase = currentQuestion === 'integrations' || currentQuestion === 'complete'
					? 'integrations'
					: 'conversation';
			} else {
				const { session } = await onboardingApi.createSession();
				sessionId = session.id;
				phase = 'intro';
			}
		} catch (error) {
			console.error('Failed to initialize onboarding:', error);
			const errorMessage = error instanceof Error ? error.message : 'Failed to start onboarding';

			if (
				errorMessage.toLowerCase().includes('expired') ||
				errorMessage.toLowerCase().includes('invalid session') ||
				errorMessage.toLowerCase().includes('session not found')
			) {
				sessionExpired = true;
				apiError = 'Your session has expired. Starting fresh...';
				setTimeout(async () => {
					try {
						const { session } = await onboardingApi.createSession();
						sessionId = session.id;
						sessionExpired = false;
						apiError = null;
						phase = 'intro';
					} catch {
						apiError = 'Unable to start a new session. Please refresh the page.';
					}
				}, 2000);
			} else {
				apiError = errorMessage;
			}
			phase = 'intro';
		}
	});

	// ─── History ──────────────────────────────────────────────────────────────

	function saveToHistory() {
		questionHistory = [...questionHistory, {
			question: currentQuestion,
			data: { ...extractedData },
			agentMessage: currentAgentMessage
		}];
	}

	function goBack() {
		if (questionHistory.length === 0 || phase !== 'conversation') return;
		const lastEntry = questionHistory[questionHistory.length - 1];
		questionHistory = questionHistory.slice(0, -1);
		currentQuestion = lastEntry.question;
		extractedData = lastEntry.data;
		currentAgentMessage = lastEntry.agentMessage;
	}

	// ─── Phase transitions ────────────────────────────────────────────────────

	function handleIntroComplete() {
		currentAgentMessage = questions.company_name.message;
		setTimeout(() => { phase = 'conversation'; }, 300);
	}

	async function skipToIntegrations() {
		phase = 'integrations';

		if (useApi && sessionId) {
			try {
				const recs = await onboardingApi.getRecommendations(sessionId);
				if (recs?.length) {
					recommendedIntegrations = recs;
					return;
				}
			} catch {
				// fall through to local computation
			}
		}
		recommendedIntegrations = computeRecommendedIntegrations(extractedData);
	}

	// ─── Answer handlers ──────────────────────────────────────────────────────

	async function handleChipSelect(chipId: string) {
		if (useApi && sessionId) {
			isAgentTyping = true;
			saveToHistory();
			lastErrorAction = () => handleChipSelect(chipId);

			try {
				const response = await onboardingApi.sendMessage(sessionId, chipId);
				lastErrorAction = null;

				const apiData = response.extracted_data;
				extractedData = {
					workspaceName: apiData.workspace_name,
					businessType: apiData.business_type,
					teamSize: apiData.team_size,
					role: apiData.role,
					challenge: apiData.challenge,
					integrations: apiData.integrations
				};

				const aiMessage = response.message?.content || '';
				const nextStep = response.next_step as QuestionType;

				if (response.recommended_integrations?.length) {
					recommendedIntegrations = response.recommended_integrations;
				}

				if (nextStep === 'integrations' || nextStep === 'complete') {
					currentAgentMessage = aiMessage || "Perfect! Let me recommend some tools for you.";
					setTimeout(() => { phase = 'integrations'; }, 2000);
				} else {
					currentQuestion = nextStep;
					currentAgentMessage = aiMessage || questions[nextStep]?.message || '';
				}
			} catch (error) {
				console.error('Failed to process chip selection:', error);
				apiError = error instanceof Error ? error.message : 'Failed to process selection';
			} finally {
				isAgentTyping = false;
			}
		} else {
			// Local mode
			if (currentQuestion === 'business_type') {
				extractedData = { ...extractedData, businessType: chipId };
				if (chipId === 'freelance') {
					extractedData = { ...extractedData, teamSize: 'solo' };
					advanceToQuestion('role');
				} else {
					advanceToQuestion('team_size');
				}
			} else if (currentQuestion === 'team_size') {
				extractedData = { ...extractedData, teamSize: chipId };
				advanceToQuestion('role');
			}
		}
	}

	async function handleChatAnswer(answer: string) {
		if (useApi && sessionId) {
			isAgentTyping = true;
			saveToHistory();
			lastErrorAction = () => handleChatAnswer(answer);

			try {
				const response = await onboardingApi.sendMessage(sessionId, answer);
				lastErrorAction = null;

				const apiData = response.extracted_data;
				extractedData = {
					workspaceName: apiData.workspace_name,
					businessType: apiData.business_type,
					teamSize: apiData.team_size,
					role: apiData.role,
					challenge: apiData.challenge,
					integrations: apiData.integrations
				};

				const aiMessage = response.message?.content || '';
				const nextStep = response.next_step as QuestionType;

				if (response.recommended_integrations?.length) {
					recommendedIntegrations = response.recommended_integrations;
				}

				if (nextStep === 'integrations' || nextStep === 'complete') {
					currentAgentMessage = aiMessage || "Perfect! Based on what you've shared, let me recommend some tools.";
					setTimeout(() => { phase = 'integrations'; }, 2500);
				} else {
					currentQuestion = nextStep;
					currentAgentMessage = aiMessage || questions[nextStep]?.message || '';
				}
			} catch (error) {
				console.error('Failed to process message:', error);
				apiError = error instanceof Error ? error.message : 'Failed to process message';
			} finally {
				isAgentTyping = false;
			}
		} else {
			// Local mode
			if (currentQuestion === 'company_name') {
				extractedData = { ...extractedData, workspaceName: answer };
				advanceToQuestion('business_type');
			} else if (currentQuestion === 'role') {
				extractedData = { ...extractedData, role: answer };
				advanceToQuestion('challenge');
			} else if (currentQuestion === 'challenge') {
				extractedData = { ...extractedData, challenge: answer };
				advanceToQuestion('complete');
			}
		}
	}

	function advanceToQuestion(nextQuestion: QuestionType) {
		isAgentTyping = true;
		setTimeout(() => {
			currentQuestion = nextQuestion;
			currentAgentMessage = questions[nextQuestion]?.message || '';
			isAgentTyping = false;

			if (nextQuestion === 'complete' || nextQuestion === 'integrations') {
				recommendedIntegrations = computeRecommendedIntegrations(extractedData);
				setTimeout(() => { phase = 'integrations'; }, 1500);
			}
		}, 800);
	}

	// ─── Input handlers ───────────────────────────────────────────────────────

	function handleFormSubmit(e: Event) {
		e.preventDefault();
		if (inputValue.trim() && !isAgentTyping) {
			handleChatAnswer(inputValue.trim());
			inputValue = '';
		}
	}

	function toggleVoiceInput() {
		isRecording = !isRecording;
	}

	// ─── Error handlers ───────────────────────────────────────────────────────

	function dismissError() {
		apiError = null;
		oauthError = null;
		lastErrorAction = null;
	}

	function retryLastAction() {
		if (lastErrorAction) {
			apiError = null;
			oauthError = null;
			lastErrorAction();
			lastErrorAction = null;
		} else if (failedIntegrationId) {
			oauthError = null;
			handleIntegrationConnect(failedIntegrationId);
			failedIntegrationId = null;
		}
	}

	// ─── Integration handlers ─────────────────────────────────────────────────

	async function handleIntegrationConnect(integrationId: string) {
		integrationStatuses[integrationId] = 'connecting';

		try {
			const apiBase = getApiBase();
			const providerMap: Record<string, string> = {
				google: 'google', microsoft: 'microsoft', slack: 'slack',
				notion: 'notion', linear: 'linear', hubspot: 'hubspot',
				airtable: 'airtable', clickup: 'clickup', fathom: 'fathom'
			};

			const provider = providerMap[integrationId];
			if (!provider) throw new Error(`Unknown integration: ${integrationId}`);

			localStorage.setItem('onboarding_oauth_provider', integrationId);
			localStorage.setItem('onboarding_session_id', sessionId || '');

			const width = 600, height = 700;
			const left = window.screenX + (window.outerWidth - width) / 2;
			const top = window.screenY + (window.outerHeight - height) / 2;
			const authUrl = `${apiBase}/integrations/${provider}/auth`;

			const popup = window.open(
				authUrl,
				'oauth_popup',
				`width=${width},height=${height},left=${left},top=${top},scrollbars=yes,resizable=yes`
			);

			const pollInterval = setInterval(() => {
				if (!popup || popup.closed) {
					clearInterval(pollInterval);
					checkIntegrationStatus(integrationId);
				}
			}, 500);

			const handleMessage = (event: MessageEvent) => {
				if (event.data?.type === 'oauth_callback' && event.data?.provider === integrationId) {
					clearInterval(pollInterval);
					window.removeEventListener('message', handleMessage);

					if (event.data.success) {
						integrationStatuses[integrationId] = 'connected';
						oauthError = null;
						failedIntegrationId = null;
						if (!selectedIntegrations.includes(integrationId)) {
							selectedIntegrations = [...selectedIntegrations, integrationId];
						}
					} else {
						integrationStatuses[integrationId] = 'error';
						failedIntegrationId = integrationId;
						oauthError = event.data.error ||
							`Failed to connect ${integrationDefs.find(i => i.id === integrationId)?.name || integrationId}`;
					}
				}
			};
			window.addEventListener('message', handleMessage);

			// Timeout after 5 minutes
			setTimeout(() => {
				clearInterval(pollInterval);
				window.removeEventListener('message', handleMessage);
				if (integrationStatuses[integrationId] === 'connecting') {
					integrationStatuses[integrationId] = 'disconnected';
					oauthError = 'Connection timed out. Please try again.';
					failedIntegrationId = integrationId;
				}
			}, 5 * 60 * 1000);

		} catch (error) {
			console.error('Failed to connect integration:', error);
			integrationStatuses[integrationId] = 'error';
			failedIntegrationId = integrationId;
			oauthError = error instanceof Error ? error.message : 'Failed to start OAuth connection';
		}
	}

	async function checkIntegrationStatus(integrationId: string) {
		try {
			const apiBase = getApiBase();
			const response = await fetch(`${apiBase}/integrations/${integrationId}/status`, {
				credentials: 'include'
			});
			if (response.ok) {
				const data = await response.json();
				if (data.connected) {
					integrationStatuses[integrationId] = 'connected';
					if (!selectedIntegrations.includes(integrationId)) {
						selectedIntegrations = [...selectedIntegrations, integrationId];
					}
				} else {
					integrationStatuses[integrationId] = 'disconnected';
				}
			}
		} catch (error) {
			console.error('Failed to check integration status:', error);
			if (integrationStatuses[integrationId] === 'connecting') {
				integrationStatuses[integrationId] = 'disconnected';
			}
		}
	}

	function handleIntegrationDisconnect(integrationId: string) {
		integrationStatuses[integrationId] = 'disconnected';
		selectedIntegrations = selectedIntegrations.filter(id => id !== integrationId);
	}

	function connectAllRecommended() {
		for (const id of recommendedIntegrations) {
			if (integrationStatuses[id] !== 'connected' && integrationStatuses[id] !== 'connecting') {
				handleIntegrationConnect(id);
			}
		}
	}

	async function completeIntegrations() {
		extractedData = { ...extractedData, integrations: selectedIntegrations };

		if (useApi && sessionId) {
			try {
				const result = await onboardingApi.completeOnboarding(sessionId, selectedIntegrations);
				localStorage.setItem('onboarding_completed', 'true');
				localStorage.setItem('workspace_id', result.workspace_id);
				goto(result.redirect_url || '/window');
			} catch (error) {
				console.error('Failed to complete onboarding:', error);
				apiError = error instanceof Error ? error.message : 'Failed to complete';
				localStorage.setItem('onboarding_completed', 'true');
				localStorage.setItem('onboarding_data', JSON.stringify(extractedData));

				if (onComplete) {
					onComplete(extractedData);
				} else {
					goto('/window');
				}
			}
		} else {
			if (onComplete) {
				onComplete(extractedData);
			} else {
				goto('/window');
			}
		}
	}
</script>

<div class="onboarding-screen {className}">
	<!-- Progress indicator -->
	<div class="progress-dots">
		<span class="dot" class:active={currentStep >= 1} class:current={currentStep === 1}></span>
		<span class="dot" class:active={currentStep >= 2} class:current={currentStep === 2}></span>
		<span class="dot" class:active={currentStep >= 3} class:current={currentStep === 3}></span>
	</div>

	<!-- Error Banner -->
	{#if apiError || oauthError}
		<div class="error-banner" transition:fly={{ y: -20, duration: 300 }}>
			<div class="error-content">
				<svg class="error-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<circle cx="12" cy="12" r="10"/>
					<line x1="12" y1="8" x2="12" y2="12"/>
					<line x1="12" y1="16" x2="12.01" y2="16"/>
				</svg>
				<span class="error-message">{apiError || oauthError}</span>
			</div>
			<div class="error-actions">
				{#if lastErrorAction || failedIntegrationId}
					<button class="btn-pill btn-pill-ghost error-btn retry" onclick={retryLastAction}>
						<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<polyline points="23 4 23 10 17 10"/>
							<path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"/>
						</svg>
						Retry
					</button>
				{/if}
				<button class="btn-pill btn-pill-ghost error-btn dismiss" onclick={dismissError} aria-label="Dismiss error">
					<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<line x1="18" y1="6" x2="6" y2="18"/>
						<line x1="6" y1="6" x2="18" y2="18"/>
					</svg>
				</button>
			</div>
		</div>
	{/if}

	{#if phase === 'loading'}
		<div class="centered-layout">
			<div class="orb-section">
				<PurpleOrb size="lg" isThinking={true} />
			</div>
			<div class="text-section">
				<p class="agent-text">Setting things up...</p>
			</div>
		</div>
	{:else if phase === 'intro' || phase === 'conversation'}
		<OnboardingConversation
			{phase}
			{currentQuestion}
			{currentQuestionConfig}
			{currentAgentMessage}
			{isAgentTyping}
			{isResuming}
			{resumeMessage}
			{canGoBack}
			bind:inputValue
			{isRecording}
			{introLines}
			onIntroComplete={handleIntroComplete}
			onChipSelect={handleChipSelect}
			onFormSubmit={handleFormSubmit}
			onGoBack={goBack}
			onSkip={skipToIntegrations}
			onContinueResume={() => { isResuming = false; }}
			onToggleVoice={toggleVoiceInput}
			onInputRef={(el) => { inputRef = el; }}
		/>
	{:else if phase === 'integrations'}
		<OnboardingIntegrations
			integrations={integrationDefs}
			{recommendedIntegrations}
			{integrationStatuses}
			{selectedIntegrations}
			{allRecommendedConnected}
			onConnect={handleIntegrationConnect}
			onDisconnect={handleIntegrationDisconnect}
			onConnectAll={connectAllRecommended}
			onComplete={completeIntegrations}
		/>
	{/if}
</div>

<style>
	.onboarding-screen {
		min-height: 100vh;
		background-color: var(--background, #ffffff);
		color: var(--foreground, #1f2937);
		position: relative;
	}

	/* Progress dots */
	.progress-dots {
		position: fixed;
		top: 24px;
		left: 50%;
		transform: translateX(-50%);
		display: flex;
		gap: 8px;
		z-index: 10;
	}

	.progress-dots .dot {
		width: 8px;
		height: 8px;
		border-radius: 50%;
		background-color: var(--border, #e5e7eb);
		transition: all 0.3s ease;
	}

	.progress-dots .dot.active {
		background-color: var(--primary, #6366f1);
	}

	.progress-dots .dot.current {
		transform: scale(1.25);
	}

	/* Error Banner */
	.error-banner {
		position: fixed;
		top: 56px;
		left: 50%;
		transform: translateX(-50%);
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 16px;
		padding: 12px 16px;
		background: var(--destructive-bg, #fef2f2);
		border: 1px solid var(--destructive-border, #fecaca);
		border-radius: 10px;
		box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
		max-width: 90%;
		min-width: 320px;
		z-index: 100;
	}

	.error-content {
		display: flex;
		align-items: center;
		gap: 10px;
		flex: 1;
	}

	.error-icon {
		width: 20px;
		height: 20px;
		color: var(--destructive, #ef4444);
		flex-shrink: 0;
	}

	.error-message {
		font-size: 14px;
		color: var(--destructive-text, #991b1b);
		line-height: 1.4;
	}

	.error-actions {
		display: flex;
		align-items: center;
		gap: 8px;
		flex-shrink: 0;
	}

	.error-btn {
		display: flex;
		align-items: center;
		gap: 4px;
		padding: 6px 12px;
		font-size: 13px;
		font-weight: 500;
		border: none;
		border-radius: 6px;
		cursor: pointer;
		transition: background 0.2s, transform 0.1s;
	}

	.error-btn svg {
		width: 14px;
		height: 14px;
	}

	.error-btn.retry {
		color: var(--destructive, #ef4444);
		background: var(--destructive-btn-bg, #fee2e2);
	}

	.error-btn.retry:hover {
		background: var(--destructive-btn-hover, #fecaca);
	}

	.error-btn.dismiss {
		padding: 6px;
		color: var(--muted-foreground, #6b7280);
		background: transparent;
	}

	.error-btn.dismiss:hover {
		background: var(--muted, #f3f4f6);
	}

	/* Loading layout (reused from conversation layout) */
	.centered-layout {
		min-height: 100vh;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: 48px 24px;
		gap: 32px;
	}

	.orb-section {
		display: flex;
		justify-content: center;
	}

	.text-section {
		text-align: center;
		max-width: 400px;
	}

	.agent-text {
		font-size: 18px;
		line-height: 1.6;
		color: var(--foreground, #1f2937);
		margin: 0;
	}
</style>
