// Export all types
export * from './types';

// Export individual functions
export * from './thinking';

// Import all functions for grouped export
import * as thinkingApi from './thinking';

// Export grouped API object
export const api = {
  // Thinking Traces
  getConversationTraces: thinkingApi.getConversationTraces,
  getMessageTrace: thinkingApi.getMessageTrace,
  deleteConversationTraces: thinkingApi.deleteConversationTraces,

  // Reasoning Templates
  getReasoningTemplates: thinkingApi.getReasoningTemplates,
  getReasoningTemplate: thinkingApi.getReasoningTemplate,
  createReasoningTemplate: thinkingApi.createReasoningTemplate,
  updateReasoningTemplate: thinkingApi.updateReasoningTemplate,
  deleteReasoningTemplate: thinkingApi.deleteReasoningTemplate,
  setDefaultTemplate: thinkingApi.setDefaultTemplate,

  // Thinking Settings
  getThinkingSettings: thinkingApi.getThinkingSettings,
  updateThinkingSettings: thinkingApi.updateThinkingSettings,
};

export default api;
