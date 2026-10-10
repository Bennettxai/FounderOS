import { writable, derived } from 'svelte/store';
import {
	getThinkingSettings,
	updateThinkingSettings,
	getReasoningTemplates,
	getReasoningTemplate,
	createReasoningTemplate,
	updateReasoningTemplate,
	deleteReasoningTemplate,
	setDefaultTemplate,
	getMessageTrace,
	deleteConversationTraces
} from '$lib/api/thinking';
import type {
	ThinkingSettings,
	ReasoningTemplate,
	ThinkingTrace,
	ThinkingStep,
	UpdateSettingsData,
	CreateTemplateData,
	UpdateTemplateData
} from '$lib/api/thinking/types';

interface ThinkingState {
	// Settings
	settings: ThinkingSettings | null;
	settingsLoading: boolean;

	// Templates
	templates: ReasoningTemplate[];
	currentTemplate: ReasoningTemplate | null;
	templatesLoading: boolean;

	// Traces
	traces: ThinkingTrace[];
	currentTrace: ThinkingTrace | null;
	tracesLoading: boolean;

	// Streaming state
	isThinking: boolean;
	streamingStep: ThinkingStep | null;

	// Cache
	tracesCache: Map<string, ThinkingTrace>; // conversationId -> trace
	templateCache: Map<string, ReasoningTemplate>; // templateId -> template

	// Error handling
	error: string | null;
}

function createThinkingStore() {
	const { subscribe, update } = writable<ThinkingState>({
		settings: null,
		settingsLoading: false,

		templates: [],
		currentTemplate: null,
		templatesLoading: false,

		traces: [],
		currentTrace: null,
		tracesLoading: false,

		isThinking: false,
		streamingStep: null,

		tracesCache: new Map(),
		templateCache: new Map(),

		error: null
	});

	return {
		subscribe,

		// ============ Settings Methods ============

		async loadSettings() {
			update((s) => ({ ...s, settingsLoading: true, error: null }));
			try {
				const settings = await getThinkingSettings();
				update((s) => ({ ...s, settings, settingsLoading: false }));
				return settings;
			} catch (error) {
				console.error('Failed to load thinking settings:', error);
				update((s) => ({
					...s,
					settingsLoading: false,
					error: error instanceof Error ? error.message : 'Failed to load settings'
				}));
				return null;
			}
		},

		async updateSettings(settings: UpdateSettingsData) {
			try {
				const updated = await updateThinkingSettings(settings);
				update((s) => ({ ...s, settings: updated }));
				return updated;
			} catch (error) {
				console.error('Failed to update thinking settings:', error);
				update((s) => ({
					...s,
					error: error instanceof Error ? error.message : 'Failed to update settings'
				}));
				throw error;
			}
		},

		async setShowThinking(enabled: boolean) {
			try {
				// Get current settings via subscribe
				let settings: ThinkingSettings | null = null;
				const unsubscribe = subscribe((state) => {
					settings = state.settings;
				});
				unsubscribe();

				if (!settings) {
					throw new Error('Settings not loaded');
				}

				// Type assertion after null check
				const currentSettings = settings as ThinkingSettings;

				const updated = await updateThinkingSettings({
					enabled: currentSettings.enabled,
					show_in_ui: enabled,
					save_traces: currentSettings.save_traces,
					max_tokens: currentSettings.max_tokens,
					default_template_id: currentSettings.default_template_id || null
				});
				update((s) => ({ ...s, settings: updated }));
				return updated;
			} catch (error) {
				console.error('Failed to update show_thinking setting:', error);
				throw error;
			}
		},

		// ============ Template Methods ============

		async loadTemplates() {
			update((s) => ({ ...s, templatesLoading: true, error: null }));
			try {
				const templates = await getReasoningTemplates();

				// Update template cache
				const templateCache = new Map<string, ReasoningTemplate>();
				templates.forEach((t: ReasoningTemplate) => templateCache.set(t.id, t));

				update((s) => ({
					...s,
					templates,
					templateCache,
					templatesLoading: false
				}));
			} catch (error) {
				console.error('Failed to load reasoning templates:', error);
				update((s) => ({
					...s,
					templatesLoading: false,
					error: error instanceof Error ? error.message : 'Failed to load templates'
				}));
			}
		},

		async loadTemplate(id: string) {
			// Check cache first
			let state: ThinkingState;
			update((s) => {
				state = s;
				return s;
			});

			const cached = state!.templateCache.get(id);
			if (cached) {
				update((s) => ({ ...s, currentTemplate: cached }));
				return cached;
			}

			// Load from API
			try {
				const template = await getReasoningTemplate(id);
				update((s) => ({
					...s,
					currentTemplate: template,
					templateCache: new Map(s.templateCache).set(id, template)
				}));
				return template;
			} catch (error) {
				console.error('Failed to load template:', error);
				update((s) => ({
					...s,
					error: error instanceof Error ? error.message : 'Failed to load template'
				}));
				return null;
			}
		},

		async createTemplate(data: CreateTemplateData) {
			try {
				const template = await createReasoningTemplate(data);
				update((s) => ({
					...s,
					templates: [template, ...s.templates],
					templateCache: new Map(s.templateCache).set(template.id, template)
				}));
				return template;
			} catch (error) {
				console.error('Failed to create template:', error);
				throw error;
			}
		},

		async updateTemplate(id: string, data: UpdateTemplateData) {
			try {
				const template = await updateReasoningTemplate(id, data);
				update((s) => ({
					...s,
					templates: s.templates.map((t) => (t.id === id ? template : t)),
					currentTemplate: s.currentTemplate?.id === id ? template : s.currentTemplate,
					templateCache: new Map(s.templateCache).set(id, template)
				}));
				return template;
			} catch (error) {
				console.error('Failed to update template:', error);
				throw error;
			}
		},

		async deleteTemplate(id: string) {
			try {
				await deleteReasoningTemplate(id);
				update((s) => {
					const newCache = new Map(s.templateCache);
					newCache.delete(id);
					return {
						...s,
						templates: s.templates.filter((t) => t.id !== id),
						currentTemplate: s.currentTemplate?.id === id ? null : s.currentTemplate,
						templateCache: newCache
					};
				});
			} catch (error) {
				console.error('Failed to delete template:', error);
				throw error;
			}
		},

		async setDefaultTemplate(templateId: string) {
			try {
				await setDefaultTemplate(templateId);
				// Reload settings to reflect the change
				await this.loadSettings();
			} catch (error) {
				console.error('Failed to set default template:', error);
				throw error;
			}
		},

		setCurrentTemplate(template: ReasoningTemplate | null) {
			update((s) => ({ ...s, currentTemplate: template }));
		},

		// ============ Trace Methods ============

		async loadTraces(filters?: { conversationId?: string; limit?: number; offset?: number }) {
			update((s) => ({ ...s, tracesLoading: true, error: null }));
			try {
				// Note: Backend doesn't have a list endpoint yet, so we'll just clear traces
				// When backend implements GET /api/thinking/traces, update this
				update((s) => ({
					...s,
					traces: [],
					tracesLoading: false
				}));
			} catch (error) {
				console.error('Failed to load traces:', error);
				update((s) => ({
					...s,
					tracesLoading: false,
					error: error instanceof Error ? error.message : 'Failed to load traces'
				}));
			}
		},

		async getTraceForMessage(conversationId: string, messageId: string) {
			// Check cache first
			let state: ThinkingState;
			update((s) => {
				state = s;
				return s;
			});

			const cacheKey = `${conversationId}:${messageId}`;
			const cached = state!.tracesCache.get(cacheKey);
			if (cached) {
				update((s) => ({ ...s, currentTrace: cached }));
				return cached;
			}

			// Load from API
			try {
				const traces = await getMessageTrace(messageId);
				const trace = traces.length > 0 ? traces[0] : null;

				if (trace) {
					update((s) => ({
						...s,
						currentTrace: trace,
						tracesCache: new Map(s.tracesCache).set(cacheKey, trace)
					}));
				}
				return trace;
			} catch (error) {
				console.error('Failed to get trace:', error);
				update((s) => ({
					...s,
					error: error instanceof Error ? error.message : 'Failed to get trace'
				}));
				return null;
			}
		},

		async deleteTraces(conversationId: string) {
			try {
				await deleteConversationTraces(conversationId);

				// Clear from cache
				update((s) => {
					const newCache = new Map(s.tracesCache);
					// Remove all traces for this conversation
					Array.from(newCache.keys())
						.filter((key) => key.startsWith(`${conversationId}:`))
						.forEach((key) => newCache.delete(key));

					return {
						...s,
						traces: s.traces.filter((t) => t.conversation_id !== conversationId),
						currentTrace:
							s.currentTrace?.conversation_id === conversationId ? null : s.currentTrace,
						tracesCache: newCache
					};
				});
			} catch (error) {
				console.error('Failed to delete traces:', error);
				throw error;
			}
		},

		setCurrentTrace(trace: ThinkingTrace | null) {
			update((s) => ({ ...s, currentTrace: trace }));
		},

		// ============ Streaming Methods ============

		startThinking() {
			update((s) => ({
				...s,
				isThinking: true,
				streamingStep: null
			}));
		},

		updateThinkingStep(step: ThinkingStep) {
			update((s) => ({
				...s,
				streamingStep: step
			}));
		},

		completeThinking(trace?: ThinkingTrace) {
			update((s) => {
				const newCache = trace
					? new Map(s.tracesCache).set(
							`${trace.conversation_id}:${trace.message_id}`,
							trace
					  )
					: s.tracesCache;

				return {
					...s,
					isThinking: false,
					streamingStep: null,
					currentTrace: trace || s.currentTrace,
					traces: trace ? [trace, ...s.traces] : s.traces,
					tracesCache: newCache
				};
			});
		},

		// ============ Cache Methods ============

		clearCache() {
			update((s) => ({
				...s,
				tracesCache: new Map(),
				templateCache: new Map()
			}));
		},

		clearConversationCache(conversationId: string) {
			update((s) => {
				const newCache = new Map(s.tracesCache);
				Array.from(newCache.keys())
					.filter((key) => key.startsWith(`${conversationId}:`))
					.forEach((key) => newCache.delete(key));

				return {
					...s,
					tracesCache: newCache
				};
			});
		},

		// ============ Utility Methods ============

		clearError() {
			update((s) => ({ ...s, error: null }));
		},

		reset() {
			update((s) => ({
				...s,
				currentTemplate: null,
				currentTrace: null,
				isThinking: false,
				streamingStep: null,
				error: null
			}));
		}
	};
}

export const thinking = createThinkingStore();

// ============ Derived Stores ============

export const thinkingEnabled = derived(thinking, ($thinking) => {
	return $thinking.settings?.enabled ?? false;
});

export const showThinkingByDefault = derived(thinking, ($thinking) => {
	return $thinking.settings?.show_in_ui ?? false;
});

export const hasActiveThinking = derived(thinking, ($thinking) => {
	return $thinking.isThinking || $thinking.streamingStep !== null;
});

export const defaultTemplate = derived(thinking, ($thinking) => {
	if (!$thinking.settings?.default_template_id) return null;
	return (
		$thinking.templates.find((t) => t.id === $thinking.settings!.default_template_id) ?? null
	);
});

export const activeTemplates = derived(thinking, ($thinking) => {
	return $thinking.templates;
});
