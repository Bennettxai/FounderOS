import { request } from '../base';
import type {
  ThinkingTrace,
  ReasoningTemplate,
  ThinkingSettings,
  CreateTemplateData,
  UpdateTemplateData,
  UpdateSettingsData
} from './types';

// ============================================================================
// THINKING TRACES
// ============================================================================

/**
 * Get all thinking traces for a conversation
 * @param conversationId - Conversation ID
 */
export async function getConversationTraces(conversationId: string) {
  return request<ThinkingTrace[]>(`/thinking/traces/${conversationId}`);
}

/**
 * Get thinking trace for a specific message
 * @param messageId - Message ID
 */
export async function getMessageTrace(messageId: string) {
  return request<ThinkingTrace[]>(`/thinking/trace/${messageId}`);
}

/**
 * Delete all thinking traces for a conversation
 * @param conversationId - Conversation ID
 */
export async function deleteConversationTraces(conversationId: string) {
  return request<{ message: string }>(`/thinking/traces/${conversationId}`, {
    method: 'DELETE'
  });
}

// ============================================================================
// REASONING TEMPLATES
// ============================================================================

/**
 * Get all reasoning templates for the current user
 */
export async function getReasoningTemplates() {
  return request<ReasoningTemplate[]>('/reasoning/templates');
}

/**
 * Get a specific reasoning template by ID
 * @param id - Template ID
 */
export async function getReasoningTemplate(id: string) {
  return request<ReasoningTemplate>(`/reasoning/templates/${id}`);
}

/**
 * Create a new reasoning template
 * @param data - Template configuration data
 */
export async function createReasoningTemplate(data: CreateTemplateData) {
  return request<ReasoningTemplate>('/reasoning/templates', {
    method: 'POST',
    body: data
  });
}

/**
 * Update an existing reasoning template
 * @param id - Template ID
 * @param data - Partial template data to update
 */
export async function updateReasoningTemplate(id: string, data: UpdateTemplateData) {
  return request<ReasoningTemplate>(`/reasoning/templates/${id}`, {
    method: 'PUT',
    body: data
  });
}

/**
 * Delete a reasoning template
 * @param id - Template ID
 */
export async function deleteReasoningTemplate(id: string) {
  return request<{ message: string }>(`/reasoning/templates/${id}`, {
    method: 'DELETE'
  });
}

/**
 * Set a template as the default for the user
 * @param id - Template ID
 */
export async function setDefaultTemplate(id: string) {
  return request<{ message: string }>(`/reasoning/templates/${id}/default`, {
    method: 'POST'
  });
}

// ============================================================================
// THINKING SETTINGS
// ============================================================================

/**
 * Get the user's thinking settings
 */
export async function getThinkingSettings() {
  return request<ThinkingSettings>('/thinking/settings');
}

/**
 * Update the user's thinking settings
 * @param data - Settings data to update
 */
export async function updateThinkingSettings(data: UpdateSettingsData) {
  return request<ThinkingSettings>('/thinking/settings', {
    method: 'PUT',
    body: data
  });
}
